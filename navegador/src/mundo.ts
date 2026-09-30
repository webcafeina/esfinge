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

import { abrirPuente, type Aviso, type PeticionDelMundo, type RespuestaAlMundo } from "./puente";
import { aBase64Url, deBase64Url } from "./nucleo/afirmacion";

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
  let siguiente = 0;
  const esperando = new Map<number, (r: RespuestaAlMundo) => void>();
  puerto.onmessage = (e: MessageEvent) => {
    const d = e.data as (Aviso & Partial<RespuestaAlMundo>) | null;
    if (!d) return;
    if (typeof d.hay === "boolean") hayLlaves = d.hay;
    if (typeof d.n === "number") {
      const quien = esperando.get(d.n);
      esperando.delete(d.n);
      quien?.(d as RespuestaAlMundo);
    }
  };

  /**
   * Pregunta al otro lado y espera. **Sin plazo**, y eso es a propósito: a partir de
   * aquí hay un banner delante de una persona, y ponerle prisa a una persona es
   * decidir por ella. El plazo que importa ya ha pasado —el de decidir si se
   * pregunta— y lo resuelve la bandera, sin esperar a nadie.
   */
  const preguntar = (p: Omit<PeticionDelMundo, "n">) =>
    new Promise<RespuestaAlMundo>((listo) => {
      const n = ++siguiente;
      esperando.set(n, listo);
      puerto.postMessage({ n, ...p } satisfies PeticionDelMundo);
    });

  /**
   * La credencial que se le devuelve al sitio.
   *
   * **No se construye un `PublicKeyCredential`**: no se puede. Se hace un objeto con
   * su prototipo y **propiedades propias de datos**, que tapan los accesores del
   * prototipo — sin eso, leer `cred.id` da «Illegal invocation». Lleva también
   * `getClientExtensionResults` y `toJSON`, que los sitios modernos llaman.
   */
  const credencial = (a: NonNullable<RespuestaAlMundo["afirmacion"]>) => {
    const bytes = (s: string) => {
      const b = deBase64Url(s);
      // Un `ArrayBuffer` propio, no la vista: es lo que el sitio espera leer.
      return b.buffer.slice(b.byteOffset, b.byteOffset + b.byteLength) as ArrayBuffer;
    };
    const respuesta = Object.create(
      typeof AuthenticatorAssertionResponse === "undefined" ? Object.prototype : AuthenticatorAssertionResponse.prototype,
    ) as Record<string, unknown>;
    Object.defineProperties(respuesta, {
      clientDataJSON: { value: bytes(a.datosDelCliente), enumerable: true },
      authenticatorData: { value: bytes(a.datosDelAutenticador), enumerable: true },
      signature: { value: bytes(a.firma), enumerable: true },
      userHandle: { value: a.idUsuario ? bytes(a.idUsuario) : null, enumerable: true },
    });

    const cred = Object.create(
      typeof PublicKeyCredential === "undefined" ? Object.prototype : PublicKeyCredential.prototype,
    ) as Record<string, unknown>;
    Object.defineProperties(cred, {
      id: { value: a.idCredencial, enumerable: true },
      rawId: { value: bytes(a.idCredencial), enumerable: true },
      type: { value: "public-key", enumerable: true },
      authenticatorAttachment: { value: "platform", enumerable: true },
      response: { value: respuesta, enumerable: true },
      getClientExtensionResults: { value: () => ({}) },
      toJSON: {
        value: () => ({
          id: a.idCredencial,
          rawId: a.idCredencial,
          type: "public-key",
          authenticatorAttachment: "platform",
          clientExtensionResults: {},
          response: {
            clientDataJSON: a.datosDelCliente,
            authenticatorData: a.datosDelAutenticador,
            signature: a.firma,
            userHandle: a.idUsuario ?? null,
          },
        }),
      },
    });
    return cred;
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
        // **Crear llaves es la entrega siguiente** (P3). Hasta entonces se cede, que
        // es lo que tiene que hacer esta pieza cuando no tiene nada que aportar.
        if (cual === "create") return Reflect.apply(original[cual], this, argumentos);

        const pk = o!.publicKey as PublicKeyCredentialRequestOptions;
        const r = await preguntar({
          rpId: pk.rpId,
          permitidas: (pk.allowCredentials ?? []).map((c) => aBase64Url(new Uint8Array(c.id as ArrayBuffer))),
          reto: aBase64Url(new Uint8Array(pk.challenge as ArrayBuffer)),
        });
        // **Sin afirmación se cede, y ése es el caso normal**: no hay llave, el
        // banner se ha cerrado con «Ahora no» o con «Usar otra llave». Ceder es
        // llamar a la original, así que a continuación sale el diálogo del
        // navegador: exactamente lo que se habría visto sin Esfinge.
        if (!r.afirmacion) return Reflect.apply(original[cual], this, argumentos);
        return credencial(r.afirmacion);
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
