/**
 * El guion del **mundo principal**: donde vive `navigator.credentials` (ADR 0048).
 *
 * Es la pieza más arriesgada de toda la extensión, y conviene decir por qué antes
 * de leer una línea. Todo lo demás de Esfinge corre en el mundo aislado: la página
 * no lo ve, no lo puede tocar, y un fallo nuestro rompe el relleno. Esto corre
 * **dentro de la página**, sustituyendo un método que el sitio usa para
 * identificar a la gente, así que un fallo aquí **rompe el inicio de sesión del
 * sitio** — también para quien no use Esfinge en esa cuenta.
 *
 * De ahí que todo esté escrito en negativo:
 *
 *  - **En un marco ajeno no se instala nada.** Ahí el propio navegador ya exige
 *    `permissions-policy`, y es donde más difícil es razonar.
 *  - **Sin puente no se instala nada.** Si el mundo aislado no contesta —el aviso
 *    de datos sin aceptar, un fallo, lo que sea—, la página funciona como si
 *    Esfinge no estuviera.
 *  - **Si otro gestor ya ha parcheado, no se instala nada.** Dos peleándose por el
 *    mismo prototipo rompen el inicio de sesión de los dos.
 *  - **Ante cualquier duda, se cede**, y ceder es llamar a la original con los
 *    mismos argumentos y el mismo `this`. Un `catch` que envuelve todo cede
 *    también: de aquí no sale nunca una excepción hacia la página.
 *
 * **Aquí no hay `chrome` ni `browser`.** En el mundo principal las API de la
 * extensión no existen, así que todo lo que haga falta preguntar va por el puente.
 * `herramientas/permisos.mjs` lo comprueba, porque el fallo sería mudo: `undefined`
 * en la primera línea, dentro de la página de otro, y nadie se entera.
 */

import { abrirPuente, type Aviso } from "./puente";

/** Lo que una llamada trae y hay que mirar **sin esperar a nadie**. */
type Opciones = CredentialRequestOptions & CredentialCreationOptions;

function enMarcoAjeno(): boolean {
  if (window.top === window.self) return false;
  try {
    return window.top!.location.origin !== window.location.origin;
  } catch {
    return true;
  }
}

/**
 * Si esta llamada es de las que Esfinge puede atender, **decidido sin preguntar a
 * nadie**.
 *
 * Es la puerta que hace que el 99 % de las páginas del mundo no paguen ni un
 * milisegundo por tener Esfinge instalada, y sobre todo la que evita el peor
 * riesgo de la fase: `create()` exige **activación de usuario**, que dura unos
 * segundos, y esperar a un trabajador dormido puede agotarla. Lo que se cede aquí
 * se cede **en la misma vuelta del bucle de eventos**.
 */
function podemosAtender(o: Opciones | undefined, hay: boolean): boolean {
  // Lo primero y lo más barato: si en este dominio no hay nada que ofrecer, esto
  // se acaba aquí sin mirar nada más.
  if (!hay) return false;
  if (!o || !o.publicKey) return false;
  // `conditional` es la interfaz de autorrelleno del **propio navegador**. No se
  // puede dibujar desde aquí, y su promesa vive minutos esperando un gesto que
  // ocurre fuera de nuestro alcance. Se cede siempre.
  if ("mediation" in o && o.mediation === "conditional") return false;
  // **Se cuentan las claves de dentro, no que el campo exista**: GitHub manda
  // `extensions` presente y vacío, así que mirando el campo cederíamos siempre y
  // Esfinge no funcionaría nunca. Lo dijo el diagnóstico del 2026-09-29.
  const ext = o.publicKey.extensions;
  if (ext && Object.keys(ext).length > 0) return false;
  return true;
}

/**
 * Deja puestas las nuestras **para siempre**: ni `delete`, ni reasignar, ni volver
 * a definirlas.
 *
 * Y si al ponerlas salta —porque otro gestor llegó antes y las dejó no
 * configurables— **se deshace y no se instala nada**.
 */
function instalar(cc: CredentialsContainer, get: PropertyDescriptor["value"], create: PropertyDescriptor["value"]): boolean {
  try {
    Object.defineProperty(cc, "get", { value: get, writable: false, configurable: false, enumerable: false });
    Object.defineProperty(cc, "create", { value: create, writable: false, configurable: false, enumerable: false });
    return true;
  } catch {
    return false;
  }
}

async function arrancar() {
  if (enMarcoAjeno()) return;
  if (typeof CredentialsContainer === "undefined") return;

  const cc = CredentialsContainer.prototype;
  const original = { get: cc.get, create: cc.create };
  if (typeof original.get !== "function" || typeof original.create !== "function") return;
  // Si ya no dicen `[native code]`, alguien ha parcheado antes: otro gestor, o la
  // propia página. En los dos casos, apartarse.
  const nativa = (f: unknown) => typeof f === "function" && /\[native code\]/.test(Function.prototype.toString.call(f));
  if (!nativa(original.get) || !nativa(original.create)) return;

  const puerto = await abrirPuente();
  if (!puerto) return;

  // **Lo que el otro lado empuja**, y nunca lo que este lado pregunta. Llega al
  // conectar y cada vez que cambie —al abrirse o cerrarse la bóveda—, y mientras no
  // llegue vale `false`: ante la duda, ceder.
  let hayLlaves = false;
  puerto.onmessage = (e: MessageEvent) => {
    const a = e.data as Aviso | null;
    if (a && typeof a.hay === "boolean") hayLlaves = a.hay;
  };

  const nuestro = (cual: "get" | "create") =>
    async function (this: CredentialsContainer, ...argumentos: unknown[]) {
      // **El `catch` de todo, y cede.** De aquí no puede salir una excepción hacia
      // la página: sería romper el inicio de sesión de un sitio por un fallo
      // nuestro, que es el riesgo número uno de esta fase.
      try {
        const o = argumentos[0] as Opciones | undefined;
        if (!podemosAtender(o, hayLlaves)) {
          return Reflect.apply(original[cual], this, argumentos);
        }
        // **Todavía no hay nada que ofrecer**: la búsqueda de llaves llega en el
        // paso siguiente. Hasta entonces se cede siempre, que es exactamente lo que
        // tiene que hacer esta pieza cuando no tiene nada que aportar.
        return Reflect.apply(original[cual], this, argumentos);
      } catch {
        return Reflect.apply(original[cual], this, argumentos);
      }
    };

  // **Los mismos argumentos y la misma identidad de objeto** al ceder: nada de
  // clonar. Clonando se pierde `options.signal` —el `AbortSignal` deja de abortar
  // la llamada de verdad— y se rompe cualquier `getter` que el sitio haya puesto.
  if (!instalar(cc, nuestro("get"), nuestro("create"))) return;
}

arrancar().catch(() => {});
