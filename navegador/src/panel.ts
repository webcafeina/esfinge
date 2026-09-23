/**
 * El panel: lo que se ve al pulsar el botón de Esfinge.
 *
 * Enseña las cuentas que hay para el sitio de la pestaña, rellena la que se elija
 * y copia lo que se le pida.
 *
 * **Y nunca ve un secreto**, que es lo que hay que conservar al tocarlo. Al copiar
 * copia Esfinge, y por el canal solo vuelve cuánto tardará en borrarse del
 * portapapeles. Al rellenar, lo que se manda es un identificador: quien pide la
 * contraseña es el guion de la página, que es el que tiene el campo donde
 * escribirla. Este panel se cierra solo al perder el foco, que es la peor clase de
 * sitio donde dejar un secreto aunque fuera un instante.
 *
 * La dirección de la pestaña la da el navegador (`tabs.query`), no la página.
 *
 * # Cómo está dibujado
 *
 * Con los tokens de la ventana (ADR 0029; `panel.css` dice cuáles). **Todo el texto
 * se escribe con `textContent`**, y los dibujos —la marca y los iconos, nuestros— se
 * insertan con `dibujar`, **sin `innerHTML` en ningún sitio** (ADR 0033).
 */
import marcaSVG from "../../build/marca.svg?raw";
import { dominioDe, inicialDe, tinteDe } from "../../frontend/src/monograma";
import { api } from "./api";
import { aceptado, aceptar } from "./consentimiento";
import { dibujar } from "./dibujo";
import type { EstadoDeCuenta, PeticionDeCuenta, RespuestaDeCuenta } from "./concuenta";
import type { Cuenta, Motivo, Peticion, Respuesta } from "./protocolo";

const donde = document.getElementById("donde") as HTMLElement;
const favicon = document.getElementById("favicon") as HTMLImageElement;
const inicialSitio = document.getElementById("inicial-sitio") as HTMLElement;
const abierta = document.getElementById("abierta") as HTMLElement;
const cargando = document.getElementById("cargando") as HTMLElement;
const lista = document.getElementById("lista") as HTMLElement;
const estado = document.getElementById("estado") as HTMLElement;
const aviso = document.getElementById("aviso") as HTMLElement;
const resultado = document.getElementById("resultado") as HTMLElement;
const pie = document.getElementById("pie") as HTMLElement;
const seccionCuenta = document.getElementById("cuenta") as HTMLElement;
const gestos = document.getElementById("gestos") as HTMLElement;

/**
 * Los iconos, a trazo y en `currentColor` como los de la barra lateral de la
 * ventana (ADR 0019): cada sitio los pinta con su token.
 */
const ICONOS = {
  llave:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="7.5" cy="15.5" r="4.5"/><path d="m10.7 12.3 9.8-9.8M16 7l3 3M14 9l2 2"/></svg>',
  reloj:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>',
  bien: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg>',
  mal: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M12 7.5v5.5M12 16.5v.01"/></svg>',
  candado:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/></svg>',
  lupa: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></svg>',
  // Un sobre, para mandar una copia (ADR 0043). Con la solapa marcada: sin ella,
  // a quince píxeles es un rectángulo más.
  sobre:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3.5 7 8.5 6 8.5-6"/></svg>',
};

/**
 * Lo que se espera a que el trabajador conteste antes de darlo por perdido.
 *
 * **Existe porque un panel que espera para siempre no dice nada**, y eso ya pasó.
 */
const PLAZO = 5000;

/**
 * Lo que se espera, tras oír el primer «aquí no hay nada», por si otra trama de la
 * misma página sí tiene el formulario. Corto: es lo que se paga cuando de verdad no
 * hay dónde rellenar.
 */
const GRACIA = 600;

/** Lo que dura «✓ Hecho» en la fila. */
const HECHO = 2000;

/**
 * pedir habla con el trabajador de fondo, **y nunca lanza**. Si lanzara, el panel
 * se quedaría con la cabecera puesta y nada debajo, sin decir por qué.
 */
async function pedir(p: Omit<Peticion, "version">): Promise<Respuesta> {
  try {
    const r = await conPlazo(porElPuerto(p));
    if (!r || typeof r.ok !== "boolean") {
      return {
        ok: false,
        error: "La extensión no ha recibido respuesta de su propio trabajador de fondo.",
      };
    }
    return r;
  } catch (e) {
    return { ok: false, error: `No se ha podido hablar con Esfinge: ${e}` };
  }
}

/**
 * porElPuerto manda la petición por un puerto y espera la respuesta.
 *
 * **Un puerto y no `sendMessage`**: prometer una respuesta para más tarde no se dice
 * igual en los dos navegadores, y con la forma de Chrome, Firefox contesta «Promised
 * response from onMessage listener went out of scope». Un puerto no promete nada.
 */
function porElPuerto(p: Omit<Peticion, "version"> | PeticionDeCuenta): Promise<Respuesta> {
  return new Promise((resolver, rechazar) => {
    const puerto = api.runtime.connect({ name: "panel" });
    let contestado = false;
    puerto.onMessage.addListener((r) => {
      contestado = true;
      puerto.disconnect();
      resolver(r as Respuesta);
    });
    puerto.onDisconnect.addListener(() => {
      if (!contestado) rechazar(new Error("el trabajador de fondo se ha ido sin contestar"));
    });
    puerto.postMessage(p);
  });
}

function conPlazo<T>(promesa: Promise<T>, plazo = PLAZO): Promise<T> {
  return Promise.race([
    promesa,
    new Promise<T>((_, rechazar) =>
      setTimeout(() => rechazar(new Error("el trabajador de fondo no ha contestado")), plazo),
    ),
  ]);
}

/**
 * Entrar o desbloquear hace dos Argon2id de 64 MiB y habla con el servidor: cerca
 * de un par de segundos en un equipo normal, y más con una red lenta. El plazo de
 * siempre lo daría por perdido.
 */
const PLAZO_DE_LA_CUENTA = 60_000;

/** pedirCuenta es `pedir` para lo de la cuenta. Tampoco lanza nunca. */
async function pedirCuenta(p: PeticionDeCuenta): Promise<RespuestaDeCuenta> {
  try {
    const r = (await conPlazo(porElPuerto(p), PLAZO_DE_LA_CUENTA)) as unknown as RespuestaDeCuenta;
    if (!r || typeof r.ok !== "boolean") {
      return { ok: false, error: "La extensión no ha recibido respuesta de su propio trabajador de fondo." };
    }
    return r;
  } catch (e) {
    return { ok: false, error: `No se ha podido hablar con el trabajador de fondo: ${e}` };
  }
}

/** Lo que se enseña cuando no hay lista: un título, qué hacer, y el detalle. */
type QueHacer = {
  titulo: string;
  texto: string;
  /** Lo que dijo el navegador o Esfinge tal cual. Para quien lo arregla. */
  detalle?: string;
  glifo: keyof typeof ICONOS;
  error?: boolean;
};

/** queHacer traduce un motivo a algo que se pueda leer, **por motivo y no por texto**. */
function queHacer(motivo: Motivo | undefined, error: string | undefined): QueHacer {
  switch (motivo) {
    case "sin-esfinge": {
      // Lo que dice el navegador va detrás de «El navegador dice:», y se enseña
      // aparte: es lo único que distingue un manifiesto que falta de un binario que
      // no se puede ejecutar, pero no es una instrucción.
      const [, navegador] = (error ?? "").split(" El navegador dice: ");
      return {
        titulo: "No se encuentra Esfinge",
        texto:
          "Comprueba que Esfinge está abierta y que el canal con el navegador está " +
          "encendido en sus Ajustes.",
        detalle: navegador ?? (error && !error.startsWith("No se puede hablar") ? error : undefined),
        glifo: "mal",
        error: true,
      };
    }
    case "sin-emparejar":
      return {
        titulo: "Falta dar permiso a este navegador",
        texto: "Permítelo en la ventana de Esfinge, en Ajustes, y vuelve a abrir este panel.",
        glifo: "candado",
      };
    case "cerrada":
      return {
        titulo: "La bóveda está cerrada",
        texto: "Ábrela en Esfinge y vuelve a abrir este panel.",
        glifo: "candado",
      };
    case "sin-boveda":
      return {
        titulo: "Todavía no hay bóveda",
        texto: "Créala en Esfinge, y las cuentas que guardes aparecerán aquí.",
        glifo: "candado",
      };
    case "origen":
      return {
        titulo: "Aquí no se rellena",
        texto: error ?? "Esfinge solo rellena en sitios con https.",
        glifo: "mal",
      };
    case "sin-consentimiento":
      return {
        titulo: "Falta aceptar el aviso de datos",
        texto: "Vuelve a abrir este panel para verlo.",
        glifo: "candado",
      };
    case "demasiado":
      return {
        titulo: "Demasiadas preguntas seguidas",
        texto: "Espera un momento y vuelve a abrir este panel.",
        glifo: "mal",
        error: true,
      };
    default:
      return {
        titulo: "Algo no ha ido bien",
        texto: error ?? "Esfinge no ha contestado lo que se esperaba.",
        glifo: "mal",
        error: true,
      };
  }
}

/** enseñar pone el estado en lugar de la lista. */
function ensenar(q: QueHacer) {
  cargando.hidden = true;
  lista.hidden = true;
  estado.hidden = false;
  estado.classList.toggle("error", Boolean(q.error));
  const tenue = estado.querySelector(".tenue") as HTMLElement;
  if (!tenue.firstChild) dibujar(tenue, marcaSVG);
  dibujar(estado.querySelector(".glifo") as HTMLElement, ICONOS[q.glifo]);
  (estado.querySelector("h1") as HTMLElement).textContent = q.titulo;
  (estado.querySelector(".texto") as HTMLElement).textContent = q.texto;
  const detalle = estado.querySelector(".detalle") as HTMLElement;
  detalle.textContent = q.detalle ?? "";
  detalle.hidden = !q.detalle;
}

/**
 * pedirConsentimiento enseña el aviso de datos y espera a que se acepte (ADR 0033).
 *
 * **Antes de preguntar nada a Esfinge**, y el orden no es un detalle: la primera
 * pregunta del panel empareja el navegador si no lo está, y eso hace salir un aviso
 * de permiso en la ventana de Esfinge. Sin aceptar, ni eso.
 *
 * **Sin foco al abrir**, por lo mismo que las filas (2.20.0): un botón con su marco
 * nada más abrir se lee como algo ya elegido. Intro acepta igualmente, y el tabulador
 * llega a los enlaces y al botón.
 */
function pedirConsentimiento(): Promise<void> {
  cargando.hidden = true;
  aviso.hidden = false;
  const boton = document.getElementById("aceptar") as HTMLButtonElement;
  return new Promise((resolver) => {
    const conIntro = (e: KeyboardEvent) => {
      const activa = document.activeElement;
      if (e.key === "Enter" && (!activa || activa === document.body)) {
        e.preventDefault();
        boton.click();
      }
    };
    document.addEventListener("keydown", conIntro);
    boton.addEventListener(
      "click",
      async () => {
        boton.disabled = true;
        document.removeEventListener("keydown", conIntro);
        await aceptar();
        aviso.hidden = true;
        cargando.hidden = false;
        resolver();
      },
      { once: true },
    );
  });
}

/* ------------------------------------------------ la cuenta (ADR 0040) */

type Paso = "entrar" | "desbloquear" | "codigo";

/**
 * pasoDeCuenta enseña el formulario de un paso y espera a que la bóveda quede
 * abierta. Entrar puede pedir el código del correo, y entonces sigue con ese paso
 * **en el mismo formulario**, sin volver a teclear nada.
 */
function pasoDeCuenta(primero: Paso, ec?: EstadoDeCuenta): Promise<void> {
  cargando.hidden = true;
  lista.hidden = true;
  estado.hidden = true;
  seccionCuenta.hidden = false;
  const titulo = document.getElementById("cuenta-titulo") as HTMLElement;
  const texto = document.getElementById("cuenta-texto") as HTMLElement;
  const form = document.getElementById("cuenta-form") as HTMLFormElement;
  const correo = document.getElementById("cuenta-correo") as HTMLInputElement;
  const maestra = document.getElementById("cuenta-maestra") as HTMLInputElement;
  const codigo = document.getElementById("cuenta-codigo") as HTMLInputElement;
  const error = document.getElementById("cuenta-error") as HTMLElement;
  const enviar = document.getElementById("cuenta-enviar") as HTMLButtonElement;
  const volver = document.getElementById("cuenta-volver") as HTMLButtonElement;
  if (ec?.correo) correo.value = ec.correo;

  let paso = primero;
  const poner = (p: Paso) => {
    paso = p;
    error.hidden = true;
    (document.getElementById("paso-correo") as HTMLElement).hidden = p !== "entrar";
    (document.getElementById("paso-maestra") as HTMLElement).hidden = p === "codigo";
    (document.getElementById("paso-codigo") as HTMLElement).hidden = p !== "codigo";
    // «Volver» al entrar sin cuenta, y en el código siempre: puede que el correo no
    // llegue o que se quiera empezar con otro.
    volver.hidden = p === "desbloquear" || (p === "entrar" && ec?.modo === "cuenta");
    if (p === "entrar") {
      titulo.textContent = "Entrar con tu cuenta de Esfinge";
      texto.textContent =
        "La extensión guardará tu bóveda cifrada en este navegador y la tendrá al día con tu " +
        "cuenta, sin necesidad de la aplicación.";
      enviar.textContent = "Entrar";
      (correo.value ? maestra : correo).focus();
    } else if (p === "desbloquear") {
      titulo.textContent = "Tu bóveda está cerrada";
      texto.textContent =
        ec?.sincro.estado === "hay-que-entrar" && ec.sincro.mensaje
          ? ec.sincro.mensaje
          : `Escribe tu contraseña maestra para abrirla en este navegador${ec?.correo ? ` (${ec.correo})` : ""}.`;
      enviar.textContent = "Abrir la bóveda";
      maestra.focus();
    } else {
      titulo.textContent = "Revisa tu correo";
      texto.textContent =
        "Te hemos mandado un código para confirmar este navegador. Puedes cerrar este panel para ir a " +
        "buscarlo: al abrirlo otra vez seguirás aquí. Caduca en diez minutos.";
      enviar.textContent = "Confirmar";
      codigo.value = "";
      codigo.focus();
    }
  };
  poner(primero);

  volver.addEventListener(
    "click",
    async () => {
      if (paso === "codigo") await pedirCuenta({ cuenta: "cancelar" });
      location.reload();
    },
    { once: true },
  );
  return new Promise((resolver) => {
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      if (enviar.disabled) return;
      const antes = enviar.textContent;
      enviar.disabled = true;
      enviar.textContent = paso === "codigo" ? "Comprobando…" : paso === "entrar" ? "Entrando…" : "Abriendo…";
      const r =
        paso === "entrar"
          ? await pedirCuenta({ cuenta: "entrar", correo: correo.value, maestra: maestra.value })
          : paso === "desbloquear"
            ? await pedirCuenta({ cuenta: "desbloquear", maestra: maestra.value })
            : await pedirCuenta({ cuenta: "codigo", codigo: codigo.value });
      enviar.disabled = false;
      enviar.textContent = antes;
      if (!r.ok) {
        error.textContent = r.error ?? "No ha ido bien.";
        error.hidden = false;
        return;
      }
      if (r.necesitaCodigo) {
        poner("codigo");
        return;
      }
      maestra.value = "";
      codigo.value = "";
      seccionCuenta.hidden = true;
      if (r.estado) pintarGestos(r.estado);
      resolver();
    });
  });
}

/**
 * Si esta bóveda es la de una cuenta. Lo que cuelga de esto es compartir: sin
 * cuenta no hay a quién mandar una copia ni de quién recibirla.
 */
let conCuenta = false;

/**
 * pintarGestos pone lo de abajo: con cuenta, de quién es y «Bloquear» y «Salir»; sin
 * cuenta, la puerta para entrar en una.
 */
function pintarGestos(ec: EstadoDeCuenta) {
  gestos.hidden = false;
  conCuenta = ec.modo === "cuenta";
  (document.getElementById("gestos-correo") as HTMLElement).textContent = conCuenta ? (ec.correo ?? "") : "";
  (document.getElementById("bloquear") as HTMLElement).hidden = !(conCuenta && ec.abierta);
  (document.getElementById("sincronizar") as HTMLElement).hidden = !(conCuenta && ec.abierta && ec.sincro.estado !== "hay-que-entrar");
  // Abierta pero sin sesión —la cuenta dejó de reconocer este navegador—: se trabaja
  // aquí, sin sincronizar, hasta volver a entrar.
  (document.getElementById("volver-a-entrar") as HTMLElement).hidden = !(conCuenta && ec.abierta && ec.sincro.estado === "hay-que-entrar");
  (document.getElementById("juntar-igual") as HTMLElement).hidden = !(conCuenta && ec.sincro.estado === "muchos-borrados");
  (document.getElementById("salir") as HTMLElement).hidden = !conCuenta;
  (document.getElementById("usar-cuenta") as HTMLElement).hidden = conCuenta;
  // Y si una fusión dejó aparte la bóveda de este navegador, se dice hasta que se
  // salga de la cuenta: lo guardado aquí y no subido está ahí dentro.
  (document.getElementById("apartada") as HTMLElement).hidden = !(conCuenta && ec.apartada);
  // El buzón se pide aparte, y solo con la bóveda abierta: sin ella no hay con
  // qué abrir los sobres.
  if (conCuenta && ec.abierta) void pintarBuzon();
  else (document.getElementById("buzon") as HTMLElement).hidden = true;
}

/**
 * Lo que te han mandado (ADR 0043).
 *
 * **Sale solo si hay algo.** Y cada envío enseña de quién viene por su huella,
 * porque es lo único que se puede comparar con la otra persona: el correo de
 * quien manda no viaja en el sobre.
 */
async function pintarBuzon() {
  const caja = document.getElementById("buzon") as HTMLElement;
  const lista = document.getElementById("buzon-lista") as HTMLElement;
  const r = await pedirCuenta({ cuenta: "buzon" });
  const envios = r.ok ? (r.buzon ?? []) : [];
  caja.hidden = envios.length === 0;
  lista.replaceChildren();
  for (const e of envios) {
    const fila = document.createElement("li");
    const que = document.createElement("span");
    que.className = "que";
    const titulo = document.createElement("span");
    titulo.className = "titulo";
    titulo.textContent = e.error ? "No se puede abrir" : e.titulo || "Sin título";
    const de = document.createElement("span");
    de.className = "huella";
    de.textContent = e.error ? e.error : `De ${e.huella}`;
    que.append(titulo, de);

    const guardar = document.createElement("button");
    guardar.type = "button";
    guardar.className = "primario";
    guardar.textContent = "Guardar";
    guardar.hidden = Boolean(e.error);
    guardar.addEventListener("click", async () => {
      guardar.disabled = true;
      const hecho = await pedirCuenta({ cuenta: "aceptar", envio: e.id });
      contar(hecho.ok ? "Copia guardada en tu bóveda." : (hecho.error ?? "No se ha podido guardar."), hecho.ok);
      await pintarBuzon();
      await pintarCuentas();
    });

    const tirar = document.createElement("button");
    tirar.type = "button";
    tirar.className = "discreto";
    tirar.textContent = "Descartar";
    tirar.addEventListener("click", async () => {
      tirar.disabled = true;
      const hecho = await pedirCuenta({ cuenta: "tirar", envio: e.id });
      contar(hecho.ok ? "Envío descartado." : (hecho.error ?? "No se ha podido descartar."), hecho.ok);
      await pintarBuzon();
    });

    fila.append(que, guardar, tirar);
    lista.append(fila);
  }
}

/**
 * Mandar una copia de una cuenta a otra persona (ADR 0043).
 *
 * **Dos pulsaciones a propósito**: primero se ve la huella de quien la va a
 * recibir y solo después se manda. El servidor contesta lo mismo tenga cuenta esa
 * dirección o no —así es como no dice quién está en Esfinge—, así que la huella
 * puede ser la de nadie: lo único que lo distingue es compararla con la otra
 * persona, y por eso se enseña antes y no después.
 */
function pantallaCompartir(cuenta: Cuenta) {
  const seccion = document.getElementById("compartir") as HTMLElement;
  const buzon = document.getElementById("buzon") as HTMLElement;
  const que = document.getElementById("compartir-que") as HTMLElement;
  const form = document.getElementById("compartir-form") as HTMLFormElement;
  const correo = document.getElementById("compartir-correo") as HTMLInputElement;
  const quien = document.getElementById("compartir-quien") as HTMLElement;
  const huella = document.getElementById("compartir-huella") as HTMLElement;
  const avisoHuella = document.getElementById("compartir-aviso") as HTMLElement;
  const error = document.getElementById("compartir-error") as HTMLElement;
  const enviar = document.getElementById("compartir-enviar") as HTMLButtonElement;
  const volver = document.getElementById("compartir-volver") as HTMLButtonElement;

  // Lo de la lista se esconde y se devuelve como estaba: el panel es una sola
  // página, y volver no puede dejar el buzón puesto si no lo estaba.
  const antes = { lista: lista.hidden, gestos: gestos.hidden, buzon: buzon.hidden, resultado: resultado.hidden };
  lista.hidden = true;
  gestos.hidden = true;
  buzon.hidden = true;
  resultado.hidden = true;
  seccion.hidden = false;

  que.textContent = `${cuenta.titulo || "Sin título"}${cuenta.usuario ? ` · ${cuenta.usuario}` : ""}`;
  correo.value = "";
  quien.hidden = true;
  huella.hidden = true;
  avisoHuella.hidden = true;
  error.hidden = true;
  enviar.textContent = "Ver su huella";
  correo.focus();

  // **Los oyentes se cortan al salir**: los elementos son los mismos cada vez que
  // se abre esta pantalla, y sin cortarlos se apilarían.
  const corte = new AbortController();
  /** El correo cuya huella se está enseñando ahora mismo. */
  let vista = "";

  const salir = () => {
    corte.abort();
    seccion.hidden = true;
    lista.hidden = antes.lista;
    gestos.hidden = antes.gestos;
    buzon.hidden = antes.buzon;
    resultado.hidden = antes.resultado;
  };

  // Al cambiar el correo se olvida la huella: mandar con la de la dirección
  // anterior sería mandar a ciegas, que es justo lo que esto evita.
  correo.addEventListener(
    "input",
    () => {
      vista = "";
      quien.hidden = true;
      huella.hidden = true;
      avisoHuella.hidden = true;
      enviar.textContent = "Ver su huella";
    },
    { signal: corte.signal },
  );
  volver.addEventListener("click", salir, { signal: corte.signal });

  form.addEventListener(
    "submit",
    async (e) => {
      e.preventDefault();
      if (enviar.disabled) return;
      const destino = correo.value.trim();
      if (!destino) return;
      error.hidden = true;
      enviar.disabled = true;
      const mirando = vista !== destino;
      enviar.textContent = mirando ? "Mirando…" : "Mandando…";
      const r = mirando
        ? await pedirCuenta({ cuenta: "huellaDe", correo: destino })
        : await pedirCuenta({ cuenta: "compartir", id: cuenta.id, correo: destino });
      enviar.disabled = false;
      if (!r.ok) {
        enviar.textContent = mirando ? "Ver su huella" : "Mandar la copia";
        error.textContent = r.error ?? (mirando ? "No se ha podido mirar esa dirección." : "No se ha podido mandar.");
        error.hidden = false;
        return;
      }
      if (mirando) {
        vista = destino;
        huella.textContent = r.huella ?? "";
        quien.hidden = false;
        huella.hidden = false;
        avisoHuella.hidden = false;
        enviar.textContent = "Mandar la copia";
        return;
      }
      salir();
      contar(`Copia de «${cuenta.titulo || "Sin título"}» mandada a ${destino}.`, true);
    },
    { signal: corte.signal },
  );
}

function atenderGestos() {
  // **Sincronizar a mano** (2.25.2): sin esperar a la pasada de cada minuto. Lo que
  // llegue se ve en la lista en el acto.
  const sincronizar = document.getElementById("sincronizar") as HTMLButtonElement;
  sincronizar.addEventListener("click", async () => {
    // Gira mientras dura, y al menos medio segundo para que se vea.
    const desde = Date.now();
    sincronizar.disabled = true;
    sincronizar.classList.add("girando");
    sincronizar.setAttribute("aria-busy", "true");
    const r = await pedirCuenta({ cuenta: "sincronizar" });
    await new Promise((listo) => setTimeout(listo, Math.max(0, 500 - (Date.now() - desde))));
    sincronizar.classList.remove("girando");
    sincronizar.removeAttribute("aria-busy");
    sincronizar.disabled = false;
    const s = r.estado?.sincro;
    if (!r.ok || !s || s.estado !== "al-dia") {
      contar(s?.mensaje ?? r.error ?? "No se ha podido sincronizar.", false);
      if (r.estado) pintarGestos(r.estado);
      return;
    }
    contar("Sincronizada con tu cuenta.", true);
    await pintarCuentas();
  });
  // **Juntarlo igual**: la salida de «la fusión borraría media bóveda». Vale para
  // una sola pasada, y lo pide una persona que está viendo lo que va a pasar.
  const juntar = document.getElementById("juntar-igual") as HTMLButtonElement;
  juntar.addEventListener("click", async () => {
    juntar.disabled = true;
    const r = await pedirCuenta({ cuenta: "sincronizar", igual: true });
    juntar.disabled = false;
    const s = r.estado?.sincro;
    if (r.estado) pintarGestos(r.estado);
    if (!r.ok || s?.estado !== "al-dia") {
      contar(s?.mensaje ?? r.error ?? "No se ha podido sincronizar.", false);
      return;
    }
    contar("Juntado con tu cuenta.", true);
    await pintarCuentas();
  });
  document.getElementById("bloquear")!.addEventListener("click", async () => {
    await pedirCuenta({ cuenta: "bloquear" });
    location.reload();
  });
  document.getElementById("volver-a-entrar")!.addEventListener("click", async () => {
    gestos.hidden = true;
    const ec = await pedirCuenta({ cuenta: "estado" });
    await pasoDeCuenta("entrar", ec.estado);
    location.reload();
  });
  document.getElementById("usar-cuenta")!.addEventListener("click", async () => {
    gestos.hidden = true;
    abierta.hidden = true;
    await pasoDeCuenta("entrar");
    location.reload();
  });
  // **Salir pide dos pulsaciones**: borra de este navegador la copia de la bóveda
  // —la de la cuenta sigue en el servidor y en los demás equipos—, y un clic
  // despistado en un panel pequeño no puede costar volver a entrar con el código.
  const salir = document.getElementById("salir") as HTMLButtonElement;
  let seguro = false;
  salir.addEventListener("click", async () => {
    if (!seguro) {
      seguro = true;
      salir.textContent = "Pulsa otra vez para salir: se borra la copia de este navegador";
      return;
    }
    salir.disabled = true;
    await pedirCuenta({ cuenta: "salir" });
    location.reload();
  });
}

/** El reloj de la cuenta atrás del código, para poder pararlo si llega otro resultado. */
let cuentaAtras: ReturnType<typeof setInterval> | undefined;

/**
 * contar pone el resultado del último gesto, bien o mal.
 *
 * Con `quedan`, **una cuenta atrás**: un anillo que se vacía con los segundos de vida
 * del código y el número al lado, que es lo que decide si da tiempo a pegarlo.
 */
function contar(texto: string, bien: boolean, quedan?: number) {
  clearInterval(cuentaAtras);
  resultado.hidden = false;
  resultado.className = `resultado ${bien ? "bien" : "mal"}`;
  resultado.replaceChildren();

  if (quedan !== undefined && bien) {
    const vida = Math.max(0, Math.min(30, Math.round(quedan)));
    dibujar(
      resultado,
      `<svg class="anillo" viewBox="0 0 20 20" aria-hidden="true">` +
        `<circle class="pista" cx="10" cy="10" r="8" pathLength="30"/>` +
        `<circle class="resto" cx="10" cy="10" r="8" pathLength="30" ` +
        `style="stroke-dashoffset:${30 - vida};animation-duration:${vida}s"/></svg>`,
    );
    const frase = document.createElement("span");
    let restantes = vida;
    const escribirFrase = () => {
      frase.textContent = `${texto} Caduca en ${restantes} s.`;
    };
    escribirFrase();
    resultado.append(frase);
    cuentaAtras = setInterval(() => {
      restantes -= 1;
      if (restantes <= 0) {
        clearInterval(cuentaAtras);
        frase.textContent = "El código ha caducado. Cópialo otra vez si te hace falta.";
        return;
      }
      escribirFrase();
    }, 1000);
    return;
  }

  dibujar(resultado, ICONOS[bien ? "bien" : "mal"]);
  const frase = document.createElement("span");
  frase.textContent = texto;
  resultado.append(frase);
}

/** copiar pide a Esfinge que copie, y cuenta lo que ha pasado. */
async function copiar(que: "copiar-secreto" | "copiar-codigo", cuenta: Cuenta, origen: string) {
  const r = await pedir({ que, id: cuenta.id, origen });
  if (!r.ok || !r.copiado) {
    const q = queHacer(r.motivo, r.error);
    contar(`${q.titulo}. ${q.texto}`, false);
    return;
  }
  // **Con cuenta copia el panel** (ADR 0040): la bóveda está en el trabajador de
  // fondo, que no puede tocar el portapapeles. Se copia aquí, dentro del margen que
  // deja el clic, y se dice que no se borra solo: la aplicación sí lo borra.
  if (r.paraCopiar !== undefined) {
    try {
      await navigator.clipboard.writeText(r.paraCopiar);
    } catch {
      contar("No se ha podido copiar. Vuelve a pulsar el botón.", false);
      return;
    }
  }
  const borrado = r.copiado.portapapeles
    ? ` Se borra del portapapeles en ${r.copiado.portapapeles} s.`
    : r.paraCopiar !== undefined
      ? " El portapapeles no se vacía solo."
      : "";
  if (que === "copiar-codigo") {
    contar(`Código copiado.${borrado}`, true, r.copiado.quedan);
  } else {
    contar(`Contraseña copiada.${borrado}`, true);
  }
}

/**
 * rellenar le dice al guion de la página que escriba esta cuenta en el formulario.
 *
 * **Por aquí no pasa ninguna contraseña**: lo que se manda es un identificador, y
 * quien pide el secreto es el guion de la página. Devuelve el aviso y si ha ido bien.
 */
function rellenar(pestana: number, cuenta: Cuenta): Promise<[string, boolean]> {
  return new Promise((resolver) => {
    let hecho = false;
    let gracia: ReturnType<typeof setTimeout> | undefined;
    // Lo que dijo la primera trama que contestó que no.
    let primerFallo = "";

    const terminar = (aviso: string, bien = false) => {
      if (hecho) return;
      hecho = true;
      clearTimeout(plazo);
      clearTimeout(gracia);
      resolver([aviso, bien]);
    };

    const plazo = setTimeout(
      () =>
        terminar(
          "El guion de Esfinge no ha contestado en esta página. Recárgala e inténtalo otra vez.",
        ),
      PLAZO,
    );

    try {
      const puerto = api.tabs.connect(pestana, { name: "rellenar" });

      puerto.onMessage.addListener((r) => {
        const { ok, error } = r as { ok: boolean; error?: string };
        // **Un «no» no cierra la conversación; un «sí», sí.** Este puerto llega a
        // todas las tramas de la pestaña —`tabs.connect` sin `frameId`— y contesta
        // cada guion que haya. Creerse al primero costó la 2.18.1.
        if (ok) {
          puerto.disconnect();
          terminar("Rellenado.", true);
          return;
        }
        if (!primerFallo) {
          primerFallo = error ?? "No se ha podido rellenar.";
          gracia = setTimeout(() => {
            puerto.disconnect();
            terminar(primerFallo);
          }, GRACIA);
        }
      });

      // Sin ningún guion en la pestaña; solo significa eso si no ha contestado nadie.
      puerto.onDisconnect.addListener(() => {
        terminar(
          primerFallo ||
            "Esfinge no está puesta en esta página. Si acabas de instalar o actualizar " +
              "la extensión, recarga la pestaña.",
        );
      });

      puerto.postMessage({ id: cuenta.id, tieneCodigo: cuenta.tieneCodigo });
    } catch (e) {
      terminar(`No se ha podido rellenar: ${e}`);
    }
  });
}

/** botonIcono hace uno de los botones de copiar. El nombre va en los dos sitios. */
function botonIcono(icono: keyof typeof ICONOS, nombre: string, alPulsar: () => Promise<void>) {
  const boton = document.createElement("button");
  boton.className = "icono";
  dibujar(boton, ICONOS[icono]);
  boton.title = nombre;
  boton.setAttribute("aria-label", nombre);
  boton.addEventListener("click", async () => {
    boton.disabled = true;
    await alPulsar();
    boton.disabled = false;
  });
  return boton;
}

/** monograma hace el cuadro con la inicial, con la misma función que la ventana. */
function monograma(clave: string, texto: string): HTMLElement {
  const cuadro = document.createElement("span");
  cuadro.className = "monograma";
  cuadro.setAttribute("aria-hidden", "true");
  cuadro.dataset.tinte = String(tinteDe(clave));
  cuadro.textContent = inicialDe(texto);
  return cuadro;
}

function fila(cuenta: Cuenta, origen: string, pestana: number | undefined): HTMLElement {
  const li = document.createElement("li");
  li.tabIndex = 0;
  li.setAttribute(
    "aria-label",
    `${cuenta.titulo || "Sin título"}, ${cuenta.usuario || "sin usuario"}. Intro para rellenar.`,
  );

  // **El color sale del sitio de la pestaña**, que es el que tienen todas las
  // cuentas de este panel: al panel no le llega el sitio guardado de cada entrada.
  // Coincide con la ventana salvo si una entrada se guardó con otro subdominio.
  const host = dominioDe(origen);
  li.append(monograma(host || cuenta.titulo, cuenta.titulo || cuenta.usuario || host));

  const texto = document.createElement("div");
  texto.className = "texto-cuenta";

  const nombre = document.createElement("span");
  nombre.className = "nombre";
  nombre.textContent = cuenta.titulo || "Sin título";
  if (cuenta.titulo) nombre.title = cuenta.titulo;
  texto.append(nombre);

  const usuario = document.createElement("span");
  usuario.className = cuenta.usuario ? "usuario" : "usuario sin-usuario";
  usuario.textContent = cuenta.usuario || "Sin usuario";
  if (cuenta.usuario) usuario.title = cuenta.usuario;
  texto.append(usuario);

  li.append(texto);

  const acciones = document.createElement("div");
  acciones.className = "acciones";

  if (pestana !== undefined) {
    const boton = document.createElement("button");
    boton.className = "rellenar";
    boton.textContent = "Rellenar";
    boton.addEventListener("click", async () => {
      boton.disabled = true;
      boton.classList.add("trabajando");
      boton.setAttribute("aria-busy", "true");
      const [aviso, bien] = await rellenar(pestana, cuenta);
      boton.classList.remove("trabajando");
      boton.removeAttribute("aria-busy");
      contar(aviso, bien);
      // **La confirmación en la propia fila**, además de la frase de abajo: con
      // varias cuentas, dice cuál se ha rellenado sin tener que leer.
      if (bien) {
        boton.textContent = "✓ Hecho";
        setTimeout(() => {
          boton.textContent = "Rellenar";
          boton.disabled = false;
        }, HECHO);
      } else {
        boton.disabled = false;
      }
    });
    acciones.append(boton);
  }

  acciones.append(
    botonIcono("llave", "Copiar la contraseña", () => copiar("copiar-secreto", cuenta, origen)),
  );
  if (cuenta.tieneCodigo !== false) {
    acciones.append(
      botonIcono("reloj", "Copiar el código de un solo uso", () =>
        copiar("copiar-codigo", cuenta, origen),
      ),
    );
  } else {
    // Un hueco del mismo ancho, para que el «Rellenar» de todas las filas caiga en
    // la misma columna.
    const hueco = document.createElement("span");
    hueco.className = "icono-hueco";
    hueco.setAttribute("aria-hidden", "true");
    acciones.append(hueco);
  }

  // Mandar una copia **solo con cuenta** (ADR 0043): sin ella no hay identidad
  // con que cifrarla ni servidor por donde mandarla.
  if (conCuenta) {
    acciones.append(
      botonIcono("sobre", "Mandar una copia", async () => {
        pantallaCompartir(cuenta);
      }),
    );
  }

  li.append(acciones);
  return li;
}

/**
 * ponerElSitio escribe la dirección de la pestaña con su icono, **sin salir a
 * internet**, como pide la ADR 0024: nunca un `<img src="https://…">`.
 *
 * En Chrome, la dirección `_favicon` de la propia extensión, que lo sirve desde la
 * caché del navegador y exige el permiso `favicon`, que solo lleva su manifiesto.
 * En Firefox, el icono de la pestaña solo si ya viene incrustado (`data:`). Si no
 * hay ninguno de los dos, el cuadro con la inicial del sitio.
 */
function ponerElSitio(pestana: chrome.tabs.Tab | undefined) {
  const url = pestana?.url ?? "";
  let host = "";
  try {
    host = new URL(url).hostname;
  } catch {
    /* página sin dirección que enseñar */
  }
  donde.textContent = host;
  if (!host) return;

  let src = "";
  if (api.runtime.getManifest().permissions?.includes("favicon")) {
    const direccion = new URL(api.runtime.getURL("/_favicon/"));
    direccion.searchParams.set("pageUrl", url);
    direccion.searchParams.set("size", "32");
    src = direccion.toString();
  } else if (pestana?.favIconUrl?.startsWith("data:")) {
    src = pestana.favIconUrl;
  }

  const sinIcono = () => {
    favicon.hidden = true;
    inicialSitio.hidden = false;
    inicialSitio.dataset.tinte = String(tinteDe(dominioDe(url)));
    inicialSitio.textContent = inicialDe(host.replace(/^www\./, ""));
  };
  if (!src) {
    sinIcono();
    return;
  }
  favicon.onerror = sinIcono;
  favicon.src = src;
  favicon.hidden = false;
}

/**
 * Los atajos: ↑ y ↓ entre cuentas, Intro rellena, Esc cierra.
 *
 * **El marco de la fila solo aparece cuando se usa el teclado**, no al abrir. En la
 * 2.20.0 la primera cuenta recibía el foco nada más abrirse el panel, y el
 * navegador le pintaba el marco sin que nadie hubiera tocado nada: se leía como una
 * cuenta seleccionada. Ahora nada tiene el foco al abrir, **Intro rellena la
 * primera igualmente** —que era para lo que servía aquel foco—, y el marco aparece
 * con la primera flecha o el tabulador. Con el ratón se va otra vez.
 */
function atenderAlTeclado() {
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      window.close();
      return;
    }
    if (e.key === "Tab") lista.classList.add("con-teclado");
    const filas = [...lista.querySelectorAll<HTMLElement>("li")];
    if (filas.length === 0 || lista.hidden) return;
    const activa = document.activeElement as HTMLElement | null;
    const actual = activa?.closest("li") ?? null;
    const i = actual ? filas.indexOf(actual) : -1;

    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      lista.classList.add("con-teclado");
      const j = e.key === "ArrowDown" ? Math.min(filas.length - 1, i + 1) : Math.max(0, i - 1);
      filas[j].focus();
      return;
    }
    if (e.key !== "Enter") return;
    // Intro en una fila rellena esa; sin nada con el foco, la primera.
    const fila = actual && activa === actual ? actual : !activa || activa === document.body ? filas[0] : null;
    if (fila) {
      e.preventDefault();
      fila.querySelector<HTMLButtonElement>(".rellenar")?.click();
    }
  });
  document.addEventListener("pointerdown", () => lista.classList.remove("con-teclado"));
}

async function arrancar() {
  dibujar(document.querySelector(".marca") as HTMLElement, marcaSVG);
  pie.textContent = `Extensión ${api.runtime.getManifest().version}`;
  atenderAlTeclado();

  const [pestana] = await api.tabs.query({ active: true, currentWindow: true });
  const origen = pestana?.url ?? "";
  ponerElSitio(pestana);

  if (!(await aceptado())) await pedirConsentimiento();

  // **Primero la cuenta**: con cuenta y la bóveda cerrada, se abre aquí mismo, sin
  // la aplicación. Preguntarlo cuenta como actividad: alguien ha abierto el panel.
  atenderGestos();
  const ec = await pedirCuenta({ cuenta: "estado" });
  if (ec.estado) {
    pintarGestos(ec.estado);
    // **Una entrada a medias va primero**, haya cuenta o no: es quien fue al correo a
    // por el código y vuelve con él.
    if (ec.estado.codigoPendiente) {
      gestos.hidden = true;
      await pasoDeCuenta("codigo", ec.estado);
      location.reload();
      return;
    }
    if (ec.estado.modo === "cuenta" && !ec.estado.abierta) {
      await pasoDeCuenta("desbloquear", ec.estado);
      cargando.hidden = false;
    }
  }

  sitioDelPanel = { origen, pestana: pestana?.id };
  await pintarCuentas();
}

/** El sitio de la pestaña, para volver a pintar sus cuentas tras sincronizar. */
let sitioDelPanel: { origen: string; pestana: number | undefined } = { origen: "", pestana: undefined };

/** pintarCuentas pide las cuentas del sitio y las pone en la lista, o dice por qué no hay. */
async function pintarCuentas() {
  const { origen, pestana } = sitioDelPanel;
  const r = await pedir({ que: "cuentas", origen });
  lista.replaceChildren();
  if (!r.ok) {
    ensenar(queHacer(r.motivo, r.error));
    return;
  }
  // Si contesta con cuentas —aunque sean cero—, la bóveda está abierta.
  abierta.hidden = false;

  const cuentas = r.cuentas ?? [];
  if (cuentas.length === 0) {
    ensenar({
      titulo: "No hay cuentas de este sitio",
      texto: "Cuando guardes una en la bóveda de Esfinge, aparecerá aquí.",
      glifo: "lupa",
    });
    return;
  }
  for (const c of cuentas) {
    lista.append(fila(c, origen, pestana));
  }
  estado.hidden = true;
  cargando.hidden = true;
  lista.hidden = false;
}

arrancar().catch((e) =>
  ensenar({
    titulo: "La extensión ha fallado por dentro",
    texto: "Vuelve a abrir este panel. Si se repite, avisa con el detalle de abajo.",
    detalle: String(e),
    glifo: "mal",
    error: true,
  }),
);
