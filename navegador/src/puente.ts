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

/**
 * Lo que el mundo aislado le **empuja** al principal, sin que nadie pregunte.
 *
 * Es una sola cosa —«en este dominio hay algo que ofrecer»— y es lo que permite que
 * el shim **ceda en la misma vuelta del bucle de eventos** en el 99 % de las
 * páginas. Sin esto habría que preguntar, preguntar es esperar al trabajador, y
 * esperar al trabajador puede agotar la activación de usuario que `create()`
 * exige: o sea, romper el inicio de sesión de quien no usa Esfinge.
 *
 * **No dice cuáles ni cuántas**: un número o una lista serían contar por el puente
 * lo que hay en la bóveda, y por ahí no pasa nada secreto.
 */
export type Aviso = {
  /** Si en este dominio hay alguna llave que ofrecer, para `get`. */
  hay: boolean;
  /**
   * Y si Esfinge **puede crear** una, para `create` (P3).
   *
   * Es otra bandera y no la misma: al crear, Esfinge se ofrece siempre —lo decidió el
   * cliente— así que no depende de que haya llaves. De lo que sí depende es del
   * interruptor de Ajustes, y **por eso hace falta decirlo aquí**: con él apagado, sin
   * esta bandera saldría el banner, la persona aceptaría y entonces se cedería al
   * navegador. O sea, prometer y no cumplir.
   *
   * Mientras no llegue vale `false`, como `hay`: ante la duda, ceder.
   */
  sePuedeCrear: boolean;
};

/**
 * Lo que el mundo principal pide, y lo único que pide.
 *
 * **Nada de esto es secreto y por eso puede viajar**: el reto y los
 * identificadores de credencial los acaba de mandar el sitio, y el `rpId` lo dice
 * él. Lo que **no** viaja en ningún sentido es la clave privada — de vuelta sube la
 * firma ya hecha— ni el `rpId` decide nada: se comprueba contra el origen que pone
 * el navegador, al otro lado.
 */
export type PeticionDelMundo = {
  /** Para emparejar la respuesta con su pregunta: puede haber dos a la vez. */
  n: number;
  /**
   * Qué se pide: usar una llave o crear una. **Ausente es usar**, para que una
   * extensión nueva entienda a una página vieja — aunque las dos vayan siempre
   * juntas, la que se queda a medias en una recarga no.
   */
  crear?: true;
  rpId?: string;
  permitidas: string[];
  reto: string;
  /**
   * Y lo que hace falta **solo para crear** (P3), todo dicho por el sitio: el nombre
   * de la cuenta, el `user.id` —bytes opacos—, cómo se llama el sitio, las llaves que
   * él dice tener ya y los algoritmos que acepta.
   */
  usuario?: string;
  idUsuario?: string;
  titulo?: string;
  excluidas?: string[];
  algoritmos?: number[];
};

/**
 * Lo que vuelve. **Sin afirmación ni atestación significa ceder**, y es el caso
 * normal: no hay llave, o el banner se cerró con «Ahora no».
 */
export type RespuestaAlMundo = {
  n: number;
  afirmacion?: {
    idCredencial: string;
    idUsuario?: string;
    datosDelCliente: string;
    datosDelAutenticador: string;
    firma: string;
  };
  atestacion?: {
    idCredencial: string;
    datosDelCliente: string;
    objeto: string;
    datosDelAutenticador: string;
    publica: string;
  };
};

/** Lo que identifica nuestro saludo entre todo el `postMessage` de una página. */
export const MARCA = "esfinge:llaves:1";

/** Cuánto se espera al acuse antes de darse por no instalado. */
const PLAZO_DEL_SALUDO = 2000;

/**
 * Cuántas veces se vuelve a tender el puente si el otro lado llega tarde.
 *
 * Tres, no una: el saludo se repite **solo cuando el mundo aislado se anuncia**, así
 * que en el caso normal se tiende una vez y las otras dos no se gastan nunca.
 */
const SALUDOS = 3;

/**
 * **El saludo va en los dos sentidos, y esto no es una precaución: es la única forma
 * de que el puente se tienda.**
 *
 * Los dos guiones entran en `document_start` y **el orden entre ellos no está
 * garantizado** —el plan lo dejó escrito como lo que había que comprobar, y la
 * respuesta es que no—. Peor aún: el lado aislado no puede escuchar hasta haber
 * leído el consentimiento, y eso es un `await` a `storage`, así que **llega tarde
 * casi siempre**. Con el saludo en un solo sentido, el mensaje del mundo principal
 * se dispara contra un `window` donde todavía no hay nadie y se pierde: el shim no
 * se instala, la página funciona como si Esfinge no estuviera, y **no hay error en
 * ningún sitio**. Dio la cara como una prueba que pasaba tres veces y a la cuarta
 * no.
 *
 * Así que cada lado anuncia y cada lado escucha:
 *
 *  - el mundo principal manda `de: "mundo"` **con un puerto**, que es lo único que
 *    de verdad autentica el canal;
 *  - el aislado manda `de: "aislado"` **sin nada**, en cuanto puede escuchar, y eso
 *    hace que el principal vuelva a tender si su primer saludo se perdió.
 *
 * Es idempotente por los dos lados: el que atiende **se quita el oyente al
 * enganchar**, así que un segundo canal no sustituye al primero, y el que saluda
 * **deja de tender en cuanto tiene acuse**, así que después del saludo no se
 * transfiere ningún puerto más. Eso último importa: un puerto transferido por
 * `window.postMessage` lo puede recoger cualquier oyente de la página, y por eso
 * solo se manda mientras no existe todavía el primer `<script>` del sitio.
 */
type Saludo = { esfinge: typeof MARCA; de: "mundo" };
type Aqui = { esfinge: typeof MARCA; de: "aislado" };

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
    let quedan = SALUDOS;

    const acabar = (p: MessagePort | null) => {
      if (contestado) return;
      contestado = true;
      ventana.removeEventListener("message", oirAlAislado);
      listo(p);
    };

    /** Un canal nuevo cada vez: un puerto ya transferido no se puede volver a mandar. */
    const tender = (): boolean => {
      if (!mandarOriginal || quedan <= 0) return false;
      quedan--;
      let canal: MessageChannel;
      try {
        canal = new MessageChannel();
      } catch {
        return false;
      }
      canal.port1.onmessage = () => acabar(canal.port1);
      try {
        mandarOriginal.call(ventana, { esfinge: MARCA, de: "mundo" } satisfies Saludo, "/", [canal.port2]);
      } catch {
        return false;
      }
      return true;
    };

    /**
     * El anuncio del otro lado. **Solo mientras no haya acuse**: en cuanto lo hay,
     * este oyente se quita y de aquí no sale ningún puerto más, así que un anuncio
     * forjado por la página no consigue que se le transfiera uno.
     */
    const oirAlAislado = (e: MessageEvent) => {
      if (contestado || e.source !== ventana) return;
      const d = e.data as Aqui | null;
      if (!d || d.esfinge !== MARCA || d.de !== "aislado") return;
      tender();
    };
    ventana.addEventListener("message", oirAlAislado);

    if (!tender()) {
      acabar(null);
      return;
    }
    setTimeout(() => acabar(null), plazo);
  });
}

/**
 * Atiende el saludo desde el mundo aislado, se queda con el puerto **y se anuncia**.
 *
 * Anunciarse es la mitad que falta: este lado no puede escuchar hasta haber leído el
 * consentimiento, así que el saludo del mundo principal ya se ha perdido cuando
 * llegamos. El anuncio es lo que hace que vuelva a tenderlo.
 */
export function atenderElPuente(alLlegar: (puerto: MessagePort) => void, ventana: Window = window): void {
  const oir = (e: MessageEvent) => {
    // `source`, la marca y `de` son para no confundirse con el ruido de la página
    // —y con nuestro propio anuncio—; lo que de verdad decide es que traiga **un**
    // puerto, porque el puerto es lo que luego no se puede falsificar.
    if (e.source !== ventana) return;
    const d = e.data as Saludo | null;
    if (!d || d.esfinge !== MARCA || d.de !== "mundo") return;
    if (e.ports.length !== 1) return;
    ventana.removeEventListener("message", oir);
    const puerto = e.ports[0];
    puerto.start();
    // El acuse: hasta que llegue, el otro lado no instala nada.
    puerto.postMessage({ esfinge: MARCA });
    alLlegar(puerto);
  };
  ventana.addEventListener("message", oir);

  // **Después de escuchar, nunca antes**: si el otro lado contestara al anuncio
  // mientras todavía no hay oyente, se perdería el saludo bueno.
  try {
    ventana.postMessage({ esfinge: MARCA, de: "aislado" } satisfies Aqui, "/");
  } catch {
    // Que no se pueda anunciar no rompe nada: queda el saludo del otro lado.
  }
}
