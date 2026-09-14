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
 *     ya está probado y está fuera de la página por construcción. **Salvo que se
 *     sepa quién entra** —el usuario escrito, el escondido que declara el sitio o el
 *     de la página anterior— y coincida con una sola: entonces ésa, que lo pidió el
 *     cliente con la 2.21.1 (`identidad.ts`).
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
import {
  buscarCodigo,
  buscarFormularios,
  contrasenasVisibles,
  usuarioDeclarado,
  camposDe,
  escribir,
  escribirCodigo,
  type DestinoDeCodigo,
  type Formulario,
} from "./campos";
import { vigilarEnvios, vigilarIdentificador } from "./envios";
import { cuentaParaRellenarSola } from "./identidad";
import { avisar, ponerFilete } from "./marcas";
import {
  VERSION_DEL_PROTOCOLO,
  type Forma,
  type Oferta,
  type Peticion,
  type Respuesta,
} from "./protocolo";
import { mostrarTarjeta, type EstadoDeLaTarjeta, type Resultado } from "./tarjeta";

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
async function rellenar(
  id: string,
  formulario: Formulario,
  // **`insistir` es lo que separa a una persona del observador**, y su ausencia
  // era un fallo silencioso de los buenos: `yaRellenados` existe para que la
  // página no entre en un tira y afloja con quien está tecleando, y aplicado al
  // botón «Rellenar» del panel hacía que **borrar los campos y pulsarlo no
  // escribiera nada, contestando «Rellenado.»**. Un clic es alguien pidiéndolo.
  insistir = false,
): Promise<string> {
  const r = await pedir({ que: "rellenar", id });
  if (!r.ok || !r.relleno) {
    return r.error ?? "Esfinge no ha podido dar la contraseña.";
  }

  const { usuario, secreto } = formulario;
  const escritos: HTMLInputElement[] = [];
  if (usuario && (insistir || !yaRellenados.has(usuario)) && r.relleno.usuario) {
    escribir(usuario, r.relleno.usuario);
    yaRellenados.add(usuario);
    escritos.push(usuario);
  }
  if (secreto && (insistir || !yaRellenados.has(secreto))) {
    escribir(secreto, r.relleno.secreto);
    yaRellenados.add(secreto);
    escritos.push(secreto);
  }
  dejarConstancia(escritos, insistir ? "" : "Rellenado por Esfinge");
  return "";
}

/**
 * dejarConstancia marca lo que Esfinge acaba de escribir (ADR 0031).
 *
 * **El filete siempre; el aviso, solo en el relleno automático**: desde el panel la
 * persona acaba de pulsar «Rellenar» y ya sabe por qué ha aparecido. Y avisa al
 * trabajador de fondo para que el icono de la barra enseñe el ✓, **sin decir
 * qué**: basta con abrir el puerto, la pestaña la pone el navegador.
 */
function dejarConstancia(campos: HTMLInputElement[], aviso: string) {
  if (campos.length === 0) return;
  campos.forEach(ponerFilete);
  if (aviso) avisar(campos[campos.length - 1], aviso);
  try {
    api.runtime.connect({ name: "relleno-hecho" }).disconnect();
  } catch {
    /* sin trabajador no hay icono que poner al día, y no es un fallo del relleno */
  }
}

/**
 * Por debajo de esto, un código se da por caducado y se espera al siguiente.
 *
 * **Escribir uno al que le quedan dos segundos es escribir uno que no va a
 * servir**: entre que aparece en el campo y alguien pulsa «Verificar» se van esos
 * dos segundos, y lo que se ve es un «código incorrecto» con el código de Esfinge
 * puesto, que es la peor forma de fallar porque parece que Esfinge calcula mal.
 */
const VIDA_MINIMA_DEL_CODIGO = 3;

const esperar = (ms: number) => new Promise((r) => setTimeout(r, ms));

/**
 * rellenarCodigo escribe el código de un solo uso de una cuenta.
 *
 * Igual que `rellenar`: el código vive **dentro de esta función y en ningún otro
 * sitio**. Y con `insistir` para lo que pide una persona desde el panel.
 */
async function rellenarCodigo(
  id: string,
  destino: DestinoDeCodigo,
  insistir = false,
): Promise<string> {
  const campos = camposDe(destino);
  if (!insistir && campos.some((c) => yaRellenados.has(c))) return "";

  let r = await pedir({ que: "rellenar-codigo", id });
  if (r.ok && r.codigo && r.codigo.quedan < VIDA_MINIMA_DEL_CODIGO) {
    await esperar((r.codigo.quedan + 1) * 1000);
    r = await pedir({ que: "rellenar-codigo", id });
  }
  if (!r.ok || !r.codigo) {
    return r.error ?? "Esfinge no ha podido dar el código.";
  }
  // **Esperando a que diga si ha quedado puesto**, que no es lo mismo que haberlo
  // escrito: ver `escribirCodigo`, y el formulario de Cloudflare.
  if (!(await escribirCodigo(destino, r.codigo.codigo))) {
    return "El código no ha quedado puesto en este formulario. Cópialo desde el panel y pégalo.";
  }
  campos.forEach((c) => yaRellenados.add(c));
  dejarConstancia(campos, insistir ? "" : "Código rellenado por Esfinge");
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

/**
 * quienEntra es el usuario que se tecleó en la página de solo usuario de este sitio,
 * o cadena vacía. Lo guarda el trabajador de fondo, porque esa página ya no está.
 */
async function quienEntra(): Promise<string> {
  const r = await hablarConElFondo<{ usuario?: string }>({ que: "quien-entra" });
  return r?.usuario ?? "";
}

/** Y un cerrojo, para no solapar dos vueltas mientras se espera la respuesta. */
let preguntando = false;

/** cuentasDeAqui pregunta a Esfinge qué hay guardado de este sitio. */
async function cuentasDeAqui() {
  const r = await pedir({ que: "cuentas" });
  return r.ok ? (r.cuentas ?? []) : [];
}

/**
 * mirar es lo que se hace cada vez que la página cambia: buscar formulario y, si se
 * sabe con qué cuenta, rellenarlo.
 *
 * **Con qué cuenta lo decide `cuentaParaRellenarSola`**: la que coincida con quien
 * entra, si se sabe; si no, la única que haya. Con varias y sin saber quién entra no
 * hay forma de acertar sin preguntar, y preguntar aquí sería dibujar en la página: se
 * rellena desde el panel, que es donde se elige.
 */
async function mirar() {
  if (preguntando || preguntadas >= TOPE_DE_PREGUNTAS) return;

  const formularios = buscarFormularios().filter(
    (f) =>
      (f.secreto && !yaRellenados.has(f.secreto)) || (f.usuario && !yaRellenados.has(f.usuario)),
  );
  // **Y el código de un solo uso**, que suele llegar en la pantalla siguiente a la
  // de la contraseña, o en la misma cuando el sitio lo pide todo a la vez.
  const codigo = buscarCodigo();
  const codigoPendiente = codigo && !camposDe(codigo).some((c) => yaRellenados.has(c));
  if (formularios.length === 0 && !codigoPendiente) return;

  preguntando = true;
  try {
    preguntadas++;
    const cuentas = await cuentasDeAqui();
    if (cuentas.length === 0) return;
    // **Quién entra** (`identidad.ts`): el usuario escondido que declara el sitio o,
    // si no, el que había en la página de solo usuario, que en la de la contraseña
    // ya no se ve. Con eso se elige la cuenta, también entre varias.
    const antes = usuarioDeclarado() || (await quienEntra());
    for (const f of formularios) {
      // Y si el formulario tiene su propio usuario escrito, manda ése.
      const cuenta = cuentaParaRellenarSola(cuentas, f.usuario?.value.trim() || antes);
      if (!cuenta) continue;
      // En la página de solo usuario, lo que ya había también es quién entra.
      if (!f.secreto && f.usuario?.value.trim()) {
        hablarConElFondo({ que: "usuario-escrito", usuario: f.usuario.value.trim() });
      }
      await rellenar(cuenta.id, f);
    }
    // Solo si Esfinge **dice** que esa cuenta tiene código: una Esfinge anterior a
    // la 2.19.0 no lo dice, y ahí no se pide uno a ciegas.
    const deCodigo = cuentaParaRellenarSola(cuentas, antes);
    if (codigo && codigoPendiente && deCodigo?.tieneCodigo === true) {
      await rellenarCodigo(deCodigo.id, codigo);
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
 *
 * # Y una trampa que costó una versión: esto le llega a **todas las tramas**
 *
 * `tabs.connect` sin `frameId` abre el puerto a todas las tramas de la pestaña
 * —lo dice su propia documentación: «instead of all frames in the tab»—, así que
 * lo contesta cada guion que haya en la página. Con este oyente registrado
 * incondicionalmente, el `iframe` del captcha de un sitio contestaba «aquí no hay
 * ningún formulario» **antes** que la trama de verdad, y el panel se quedaba con
 * la primera respuesta que le llegaba. En Brevo y en Cloudflare el relleno
 * automático funcionaba y el botón del panel no, que es un síntoma raro de
 * explicar y con una causa muy concreta.
 *
 * Dos cosas lo arreglan, y las dos son correctas por su cuenta:
 *
 *   - **El oyente vive dentro del guardián**, no fuera. Una trama de otro origen
 *     no rellena y por tanto tampoco tiene nada que contestar. Antes el guardián
 *     solo protegía el relleno automático, que era la mitad del trabajo.
 *   - **El panel espera un momento a que conteste alguien que sí tenga formulario**
 *     en vez de creerse al primero. Está en `panel.ts`.
 */
function atenderAlPanel() {
  api.runtime.onConnect.addListener((puerto) => {
    if (puerto.name !== "rellenar") return;
    puerto.onMessage.addListener((m) => {
      const { id = "", tieneCodigo } = m as { id?: string; tieneCodigo?: boolean };
      const contestar = (error: string) => {
        try {
          puerto.postMessage({ ok: !error, error });
        } catch {
          /* el panel se ha ido */
        }
      };
      const formularios = buscarFormularios();
      // El código solo si la cuenta lo tiene; `undefined` es un panel anterior, y
      // ahí se intenta, que si no hay código Esfinge lo dirá.
      const codigo = tieneCodigo === false ? null : buscarCodigo();
      if (formularios.length === 0 && !codigo) {
        contestar("Aquí no hay ningún formulario de entrar ni de código que Esfinge sepa rellenar.");
        return;
      }
      // **Insistiendo**: lo ha pedido una persona, así que se escribe aunque ya se
      // hubiera rellenado antes y se hayan borrado los campos a mano. Primero el
      // formulario de entrar y luego el código; si hay formulario y el código
      // falla, lo que se cuenta es que el formulario se ha rellenado.
      (async () => {
        if (formularios.length > 0) {
          const fallo = await rellenar(id, formularios[0], true);
          if (fallo) return fallo;
          if (codigo) await rellenarCodigo(id, codigo, true);
          return "";
        }
        return codigo ? rellenarCodigo(id, codigo, true) : "";
      })()
        .then(contestar)
        // Con red debajo, como todo lo que arranca solo en este proyecto: sin esto,
        // una excepción aquí es una promesa rechazada que nadie recoge y el panel se
        // queda esperando su plazo sin saber por qué.
        .catch((e) => contestar(`La extensión ha fallado por dentro: ${e}`));
    });
  });
}

/* ------------------------------------------------ guardar desde la página */

/**
 * hablarConElFondo manda un mensaje por el puerto de la tarjeta y espera la
 * respuesta, **o nulo si no llega**: aquí nada puede dejar la página esperando.
 */
function hablarConElFondo<T>(m: unknown): Promise<T | null> {
  return new Promise((resolver) => {
    let hecho = false;
    const terminar = (r: T | null) => {
      if (hecho) return;
      hecho = true;
      clearTimeout(plazo);
      resolver(r);
    };
    const plazo = setTimeout(() => terminar(null), PLAZO);
    try {
      const puerto = api.runtime.connect({ name: "tarjeta" });
      puerto.onMessage.addListener((r) => {
        puerto.disconnect();
        terminar(r as T);
      });
      puerto.onDisconnect.addListener(() => terminar(null));
      puerto.postMessage(m);
    } catch {
      terminar(null);
    }
  });
}

type LoQueHayPendiente = {
  nada?: boolean;
  cerrada?: boolean;
  sitio?: string;
  usuario?: string;
  forma?: Forma;
  oferta?: Oferta;
};

/** La tarjeta que hay a la vista, para no poner dos. */
let tarjetaAbierta: ReturnType<typeof mostrarTarjeta> | null = null;

/**
 * Lo que se espera después de un envío antes de mirar si hay que ofrecer algo en la
 * misma página, para los sitios que entran sin cambiar de página.
 */
const ESPERAS_TRAS_ENVIAR = [3000, 8000, 15000];

/**
 * pareceFallido dice si, después de enviar, **el formulario sigue ahí**, que es la
 * señal de que no ha ido bien: la contraseña era mala, o el sitio no aceptó la
 * nueva. **Guardar una contraseña equivocada es peor que no ofrecer.**
 *
 * Al entrar, cualquier campo de contraseña visible cuenta. Al registrarse o cambiar
 * no: tras un cambio bien hecho, muchos sitios dejan el formulario puesto y vacío,
 * así que ahí solo cuenta si los campos siguen rellenos.
 */
function pareceFallido(forma: Forma | undefined): boolean {
  const visibles = contrasenasVisibles();
  if (forma === "entrar") return visibles.length > 0;
  return visibles.some((c) => c.value !== "");
}

/**
 * mirarPendiente pregunta si quedó algo por ofrecer y, si lo hay, enseña la tarjeta.
 *
 * `alCargar` dice si es la página nueva. **Solo al cargar se descarta** lo que parece
 * fallido: a los tres segundos de enviar puede ser todavía la página de antes, con
 * su formulario, esperando a que el sitio conteste; descartar ahí perdería una
 * oferta buena.
 */
async function mirarPendiente(alCargar: boolean) {
  const r = await hablarConElFondo<LoQueHayPendiente>({ que: "mirar" });
  if (!r) return;
  if (r.nada) {
    // Si había una tarjeta a la vista —«Ya la he abierto» tarde, con la oferta
    // caducada—, no puede quedarse ofreciendo algo que ya no existe.
    tarjetaAbierta?.cerrar();
    tarjetaAbierta = null;
    return;
  }
  if (pareceFallido(r.forma)) {
    if (alCargar) hablarConElFondo({ que: "descartar" });
    return;
  }
  const estado: EstadoDeLaTarjeta =
    r.cerrada || !r.oferta
      ? { tipo: "cerrada", sitio: r.sitio ?? "", usuario: r.usuario ?? "" }
      : { tipo: "oferta", oferta: r.oferta, usuario: r.usuario ?? "" };
  if (tarjetaAbierta) {
    tarjetaAbierta.poner(estado);
    return;
  }
  tarjetaAbierta = mostrarTarjeta(
    estado,
    async (d) => {
      const res = await hablarConElFondo<Resultado>({ que: "decidir", ...d });
      if (d.accion === "ahora-no" || res?.ok) tarjetaAbierta = null;
      return res ?? { ok: false, error: "La extensión no ha contestado. Inténtalo otra vez." };
    },
    () => {
      mirarPendiente(false).catch(() => {});
    },
  );
}

/**
 * vigilarLoQueSeEnvia manda al trabajador de fondo lo que se envía y mira si hay que
 * ofrecer algo: al cargar, por lo que quedó de la página anterior, y a los tres
 * segundos de enviar, por los sitios que entran sin cambiar de página.
 *
 * **La tarjeta solo en la trama de arriba**: con marcos del mismo origen habría una
 * por marco. Los envíos, en cambio, se leen en todas, porque el formulario puede
 * estar en un marco.
 */
function vigilarLoQueSeEnvia() {
  vigilarIdentificador((usuario) => {
    hablarConElFondo({ que: "usuario-escrito", usuario });
  });
  vigilarEnvios((envio) => {
    hablarConElFondo({ que: "envio", ...envio });
    if (window.top === window.self) {
      // **Varias veces**: un sitio que cambia la contraseña sin cambiar de página
      // —Brevo— puede tardar en contestar y en vaciar el formulario. Y sin repintar
      // una tarjeta que ya esté a la vista, que borraría el título que se esté
      // escribiendo.
      for (const espera of ESPERAS_TRAS_ENVIAR) {
        setTimeout(() => {
          if (!tarjetaAbierta) mirarPendiente(false).catch(() => {});
        }, espera);
      }
    }
  });
  if (window.top === window.self) mirarPendiente(true).catch(() => {});
}

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
  // **El guardián va antes que todo, no antes de una parte.** Una trama de otro
  // origen no rellena sola y tampoco contesta al panel: si contestara, sería una
  // voz más en una conversación donde el panel se cree la primera que oye.
  if (enMarcoAjeno()) return;

  atenderAlPanel();
  vigilarLoQueSeEnvia();
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
