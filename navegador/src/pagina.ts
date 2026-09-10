/**
 * El guion que Esfinge pone en las páginas. **La entrega 2 entera vive aquí.**
 *
 * # Lo que este fichero es, dicho antes que nada
 *
 * Es **código nuestro en todas las páginas que abras**, y eso es lo que se compró
 * al decidir que Esfinge rellenara sola. Hasta la entrega 1 el canal no tocaba
 * ninguna página; desde ésta, sí. La ADR 0027 lo dice con esas palabras y aquí se
 * repite porque es donde hay que acordarse.
 *
 * # Y lo que **no** es, que es la decisión de la entrega
 *
 * **No dibuja nada.** Ni un desplegable sobre el campo, ni un icono dentro, ni un
 * marco flotante. Solo lee el formulario y escribe en él. Las razones, por orden:
 *
 *   - Es lo que reduce de verdad la superficie. Un desplegable propio en la página
 *     de otro obliga a un marco de nuestro origen, a pelearse con el `z-index` y
 *     el `position` de cada sitio, y a que la página no pueda leerlo ni pulsarlo
 *     por su cuenta. Todo eso es trabajo, y **todo eso es superficie**.
 *   - Con una sola cuenta guardada —el caso corriente— no hace falta elegir nada:
 *     se rellena y ya está, que es justo lo que se pidió.
 *   - Y con varias, elegir se hace **en el panel de la extensión**, que ya existe,
 *     ya está probado y está fuera de la página por construcción.
 *
 * Si al usarlo resulta que hacen falta dos clics demasiado a menudo, el
 * desplegable dentro del campo es lo siguiente y se decide entonces. Está en
 * `docs/deuda.md` dicho así.
 *
 * # Las tres reglas que no se tocan
 *
 *   - **Nunca en un marco de otro origen.** Un `iframe` que no comparte origen con
 *     la barra de direcciones puede ser cualquiera, y la dirección que se ve no es
 *     la suya. Se comprueba abajo, y en cuanto falla este guion no hace nada más.
 *   - **El origen lo dice el navegador.** Aquí no se manda ninguna dirección: el
 *     trabajador de fondo la saca de `sender.tab.url` y pisa lo que llegue. Una
 *     página no puede decir de qué sitio es.
 *   - **La contraseña no se guarda.** Llega, se escribe en el campo y se va con la
 *     función. Ni una variable de módulo, ni `storage`, ni un registro. Esto es lo
 *     único que el canal no puede comprobar por nosotros —lo dice
 *     `docs/seguridad.md`— así que es una promesa de este fichero.
 */
import { api } from "./api";
import { buscarFormularios, escribir, type Formulario } from "./campos";
import { VERSION_DEL_PROTOCOLO, type Peticion, type Respuesta } from "./protocolo";

/**
 * enMarcoAjeno dice si esto se está ejecutando donde no debe.
 *
 * Leer `location` del marco de arriba **lanza** cuando el origen no es el mismo, y
 * ésa es justamente la comprobación: si lanza, no somos de la casa. Se da por
 * ajeno también cuando algo va mal por cualquier otro motivo, que es la dirección
 * segura del error.
 */
function enMarcoAjeno(): boolean {
  if (window.top === window.self) return false;
  try {
    return window.top!.location.origin !== window.location.origin;
  } catch {
    return true;
  }
}

/** Lo que se espera al trabajador de fondo antes de darlo por perdido. */
const PLAZO = 5000;

/**
 * pedir habla con el trabajador de fondo por un puerto.
 *
 * **Por un puerto y no con `sendMessage`**, por lo mismo que el panel: prometer
 * una respuesta para más tarde no se dice igual en los dos navegadores, y con la
 * forma de Chrome, Firefox contesta «Promised response from onMessage listener
 * went out of scope». Un puerto no promete nada.
 *
 * El nombre del puerto es «pagina», y no es decorativo: es lo que hace que el
 * trabajador ponga el origen que da el navegador en vez del que llegue.
 */
function pedir(p: Omit<Peticion, "version" | "origen">): Promise<Respuesta> {
  return new Promise((resolver) => {
    let hecho = false;
    const terminar = (r: Respuesta) => {
      if (hecho) return;
      hecho = true;
      resolver(r);
    };
    const plazo = setTimeout(
      () => terminar({ ok: false, error: "El trabajador de fondo no ha contestado" }),
      PLAZO,
    );
    try {
      const puerto = api.runtime.connect({ name: "pagina" });
      puerto.onMessage.addListener((r) => {
        clearTimeout(plazo);
        puerto.disconnect();
        terminar(r as Respuesta);
      });
      puerto.onDisconnect.addListener(() => {
        clearTimeout(plazo);
        terminar({ ok: false, error: "El trabajador de fondo se ha ido sin contestar" });
      });
      puerto.postMessage({ ...p, version: VERSION_DEL_PROTOCOLO });
    } catch (e) {
      clearTimeout(plazo);
      terminar({ ok: false, error: `${e}` });
    }
  });
}

/**
 * yaRellenados son los campos en los que ya se ha escrito en esta carga.
 *
 * **Existe para no rellenar dos veces**, y eso importa más de lo que parece: la
 * página se observa mientras cambia —los formularios llegan tarde en media web—,
 * así que sin esto un sitio que vacía su campo al arrancar entraría en un ida y
 * vuelta con Esfinge escribiendo encima de la persona que está tecleando.
 *
 * Guarda **elementos, no valores**. Aquí no hay ninguna contraseña.
 */
const yaRellenados = new WeakSet<HTMLInputElement>();

/**
 * rellenar escribe una cuenta en un formulario.
 *
 * La contraseña vive **dentro de esta función y en ningún otro sitio**: llega en
 * la respuesta, se escribe y se va con ella. Es la promesa del encabezado de este
 * fichero, y está escrita así para que romperla exija mover código.
 */
async function rellenar(id: string, formulario: Formulario): Promise<string> {
  const r = await pedir({ que: "rellenar", id });
  if (!r.ok || !r.relleno) {
    return r.error ?? "Esfinge no ha podido dar la contraseña.";
  }

  const { usuario, secreto } = formulario;
  if (usuario && !yaRellenados.has(usuario) && r.relleno.usuario) {
    escribir(usuario, r.relleno.usuario);
    yaRellenados.add(usuario);
  }
  if (secreto && !yaRellenados.has(secreto)) {
    escribir(secreto, r.relleno.secreto);
    yaRellenados.add(secreto);
  }
  return "";
}

/**
 * Cuántas veces se le puede preguntar a Esfinge por una carga de página.
 *
 * **Existe porque el observador puede dispararse muchas veces.** Una aplicación
 * que redibuja sin parar produce una ráfaga de mutaciones cada pocos cientos de
 * milisegundos, y con una pregunta por ráfaga una sola pestaña podría gastarse
 * ella sola el freno de sesenta por minuto del canal —y dejar sin contestar al
 * panel, que es lo que se ve—. Cinco cubren de sobra lo que tarda en montarse un
 * formulario que llega tarde.
 */
const TOPE_DE_PREGUNTAS = 5;
let preguntadas = 0;

/** Y un cerrojo, para no solapar dos vueltas mientras se espera la respuesta. */
let preguntando = false;

/** cuentasDeAqui pregunta a Esfinge qué hay guardado de este sitio. */
async function cuentasDeAqui() {
  const r = await pedir({ que: "cuentas" });
  return r.ok ? (r.cuentas ?? []) : [];
}

/**
 * mirar es lo que se hace cada vez que la página cambia: buscar formulario y, si
 * hay **una sola** cuenta, rellenarlo.
 *
 * **Una sola, y con dos ya no se hace nada.** Con dos no hay forma de acertar sin
 * preguntar, y preguntar aquí sería dibujar en la página. Con dos se rellena desde
 * el panel, que es donde se elige.
 */
async function mirar() {
  if (preguntando || preguntadas >= TOPE_DE_PREGUNTAS) return;

  const formularios = buscarFormularios().filter(
    (f) =>
      (f.secreto && !yaRellenados.has(f.secreto)) || (f.usuario && !yaRellenados.has(f.usuario)),
  );
  if (formularios.length === 0) return;

  preguntando = true;
  try {
    preguntadas++;
    const cuentas = await cuentasDeAqui();
    if (cuentas.length !== 1) return;
    for (const f of formularios) {
      await rellenar(cuentas[0].id, f);
    }
  } finally {
    preguntando = false;
  }
}

/**
 * El panel pide rellenar con una cuenta concreta, y esto lo hace.
 *
 * **La contraseña no pasa por el panel.** El panel manda un identificador; quien
 * pide el secreto es este guion, que es el que tiene el campo donde escribirlo.
 * Un sitio menos por el que pasa una contraseña, y encima el panel se cierra solo
 * al perder el foco, que es la peor clase de sitio donde dejar algo.
 */
api.runtime.onConnect.addListener((puerto) => {
  if (puerto.name !== "rellenar") return;
  puerto.onMessage.addListener((m) => {
    const id = (m as { id?: string }).id ?? "";
    const contestar = (error: string) => {
      try {
        puerto.postMessage({ ok: !error, error });
      } catch {
        /* el panel se ha ido */
      }
    };
    const formularios = buscarFormularios();
    if (formularios.length === 0) {
      contestar("Aquí no hay ningún formulario de entrar que Esfinge sepa rellenar.");
      return;
    }
    // Se pide una vez y se escribe en todos los que haya, que casi siempre es uno.
    rellenar(id, formularios[0])
      .then(contestar)
      // Con red debajo, como todo lo que arranca solo en este proyecto: sin esto,
      // una excepción aquí es una promesa rechazada que nadie recoge y el panel se
      // queda esperando su plazo sin saber por qué.
      .catch((e) => contestar(`La extensión ha fallado por dentro: ${e}`));
  });
});

/**
 * Y el arranque: mirar ahora y mirar mientras la página se monta.
 *
 * **Con un plazo de observación y no para siempre.** Un observador vivo en cada
 * pestaña abierta durante horas es trabajo constante en páginas que ya no van a
 * traer ningún formulario. Treinta segundos cubren lo que tarda en montarse
 * cualquier aplicación de las que llegan tarde; lo que aparezca después se rellena
 * desde el panel.
 */
const PLAZO_DE_OBSERVACION = 30000;

function arrancar() {
  if (enMarcoAjeno()) return;

  mirar().catch(() => {});

  let pendiente: ReturnType<typeof setTimeout> | undefined;
  const observador = new MutationObserver(() => {
    // Sin prisa: los cambios llegan a ráfagas y buscar formularios recorre el
    // documento. Una vuelta por ráfaga es de sobra.
    clearTimeout(pendiente);
    pendiente = setTimeout(() => mirar().catch(() => {}), 300);
  });
  observador.observe(document.documentElement, { childList: true, subtree: true });
  setTimeout(() => observador.disconnect(), PLAZO_DE_OBSERVACION);
}

arrancar();
