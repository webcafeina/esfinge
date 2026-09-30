/**
 * El puente entre el mundo principal y el mundo aislado (ADR 0048).
 *
 * Existe porque las llaves de acceso obligan a poner código nuestro **dentro** de
 * la página —ahí es donde vive `navigator.credentials`— y ahí no hay `chrome` ni
 * `browser`: no se puede hablar con el trabajador de fondo. Hace falta un canal
 * entre los dos mundos, y ese canal es superficie nueva en cada página `https`,
 * porque la página también puede escribir en `window`.
 *
 * # Lo que lo cierra, y lo que no
 *
 * Se usa un **`MessagePort`**, no `window.postMessage` a secas ni un `CustomEvent`.
 * El puerto se transfiere **una vez**, en `document_start`, antes de que exista el
 * primer `<script>` de la página, y a partir de ahí **no se puede escribir en él
 * sin tenerlo**. Un identificador secreto no valdría: viaja en ese primer mensaje,
 * que cualquier oyente de la página puede leer, y a partir de ahí lo tiene. La
 * marca sirve para **distinguir nuestro tráfico del ruido** —Stripe, Intercom y
 * media web usan `postMessage` en el mismo `window`—, no para autenticar.
 *
 * **Y el puerto es endurecimiento, no la frontera.** La frontera son tres
 * invariantes, y se sostienen aunque alguien consiguiera escribir en el canal:
 *
 *  1. **Por aquí no pasa nada secreto.** Ni la clave privada, ni la maestra, ni
 *     identificadores de credencial antes de que alguien acepte. Lo que baja al
 *     mundo principal es lo que la página va a recibir de todas formas como
 *     respuesta de `navigator.credentials.get`.
 *  2. **El origen lo sigue poniendo el trabajador**, con `sender.tab.url`. El
 *     mundo principal no puede decir para qué sitio se firma.
 *  3. **Nada consecuente ocurre sin un clic `isTrusted`** en la sombra cerrada del
 *     mundo aislado. Un mensaje forjado, como mucho, saca un banner.
 */

/** Lo que identifica nuestro saludo entre todo el `postMessage` de una página. */
export const MARCA = "esfinge:llaves:1";

/** Cuánto se espera al acuse antes de darse por no instalado. */
const PLAZO_DEL_SALUDO = 2000;

type Saludo = { esfinge: typeof MARCA };

/**
 * `postMessage` capturada **al cargar el módulo**, que en el mundo principal es
 * `document_start`: antes del primer `<script>` de la página.
 *
 * Y no se coge del prototipo, que es lo que parecía obvio: **`postMessage` no está
 * en `Window.prototype`**, es una propiedad propia del objeto global —comprobado
 * preguntándoselo a un Chromium de verdad, porque razonarlo daba lo contrario— y
 * además es `writable` y `configurable`, así que **la página puede sustituirla**.
 * Capturarla aquí es lo único que garantiza que se manda por la de verdad, y es la
 * mitad del porqué de `document_start`.
 */
// El tipo se escribe a mano porque `postMessage` tiene dos formas —con destino y
// transferencias, o con un objeto de opciones— y llamándola con `call` TypeScript
// elige la que no es.
type MandarConPuerto = (this: Window, mensaje: unknown, destino: string, transferir: Transferable[]) => void;
const mandarOriginal: MandarConPuerto | undefined =
  typeof window === "undefined" ? undefined : (window.postMessage as MandarConPuerto);

/**
 * Abre el puente desde el mundo principal y devuelve el puerto, o `null`.
 *
 * **Sin acuse no hay puente, y sin puente no se instala nada**: la página funciona
 * exactamente como si Esfinge no estuviera. Es la salida de emergencia de toda la
 * fase, así que tiene que ser la primera línea y no una rama olvidada.
 */
export function abrirPuente(ventana: Window = window, plazo = PLAZO_DEL_SALUDO): Promise<MessagePort | null> {
  return new Promise((listo) => {
    let contestado = false;
    const acabar = (p: MessagePort | null) => {
      if (contestado) return;
      contestado = true;
      listo(p);
    };

    let canal: MessageChannel;
    try {
      canal = new MessageChannel();
    } catch {
      acabar(null);
      return;
    }

    canal.port1.onmessage = () => acabar(canal.port1);
    if (!mandarOriginal) {
      acabar(null);
      return;
    }
    try {
      mandarOriginal.call(ventana, { esfinge: MARCA } satisfies Saludo, "/", [canal.port2]);
    } catch {
      acabar(null);
      return;
    }
    setTimeout(() => acabar(null), plazo);
  });
}

/**
 * Atiende el saludo desde el mundo aislado y se queda con el puerto.
 *
 * Hay que llamarlo **en la primera línea del guion aislado**: el saludo llega en
 * `document_start` y un oyente puesto después no lo ve.
 */
export function atenderElPuente(alLlegar: (puerto: MessagePort) => void, ventana: Window = window): void {
  const oir = (e: MessageEvent) => {
    // `source` y la marca son para no confundirse con el ruido de la página; lo que
    // de verdad decide es que traiga **un** puerto, porque el puerto es lo que
    // luego no se puede falsificar.
    if (e.source !== ventana) return;
    const d = e.data as Saludo | null;
    if (!d || d.esfinge !== MARCA) return;
    if (e.ports.length !== 1) return;
    ventana.removeEventListener("message", oir);
    const puerto = e.ports[0];
    puerto.start();
    // El acuse: hasta que llegue, el otro lado no instala nada.
    puerto.postMessage({ esfinge: MARCA });
    alLlegar(puerto);
  };
  ventana.addEventListener("message", oir);
}
