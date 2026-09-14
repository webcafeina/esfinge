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
 * La dirección de la pestaña la da el navegador (`tabs.query`), no la página. Es
 * la diferencia entre preguntar por un sitio y preguntar por lo que un documento
 * dice que es.
 *
 * # Cómo está dibujado
 *
 * Con los tokens de la ventana (`panel.css` explica cuáles y por qué). Aquí solo
 * hay estructura: la cabecera con la marca, una fila por cuenta con la misma
 * forma todas, un estado con jerarquía cuando no hay lista, y una línea con el
 * resultado del último gesto. **Todo el texto se escribe con `textContent`**, y
 * los únicos `innerHTML` son dibujos nuestros —la marca y cuatro iconos—, nunca
 * nada que venga de la bóveda ni de la página.
 */
import marcaSVG from "../../build/marca.svg?raw";
import { api } from "./api";
import type { Cuenta, Motivo, Peticion, Respuesta } from "./protocolo";

const donde = document.getElementById("donde") as HTMLElement;
const cargando = document.getElementById("cargando") as HTMLElement;
const lista = document.getElementById("lista") as HTMLElement;
const estado = document.getElementById("estado") as HTMLElement;
const resultado = document.getElementById("resultado") as HTMLElement;
const pie = document.getElementById("pie") as HTMLElement;

/**
 * Los iconos, a trazo y en `currentColor` como los de la barra lateral de la
 * ventana (ADR 0019): cada sitio los pinta con su token.
 */
const ICONOS = {
  // Una llave: la contraseña.
  llave:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="7.5" cy="15.5" r="4.5"/><path d="m10.7 12.3 9.8-9.8M16 7l3 3M14 9l2 2"/></svg>',
  // Un reloj: el código, que caduca.
  reloj:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>',
  bien: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg>',
  mal: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M12 7.5v5.5M12 16.5v.01"/></svg>',
  // Un candado: la bóveda cerrada, o sin permiso.
  candado:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/></svg>',
  // Una lupa: no hay nada de este sitio.
  lupa: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></svg>',
};

/**
 * Lo que se espera a que el trabajador conteste antes de darlo por perdido.
 *
 * **Existe porque un panel que espera para siempre no dice nada**, y eso ya pasó:
 * el trabajador prometía contestar más tarde —`return true`— y si su respuesta no
 * llegaba nunca, el panel se quedaba con la cabecera puesta y nada debajo. Desde
 * fuera es idéntico a un fallo, y no hay forma de distinguirlos mirando. Cinco
 * segundos son una eternidad para lo que esto hace.
 */
const PLAZO = 5000;

/**
 * Lo que se espera, tras oír el primer «aquí no hay nada», por si otra trama de la
 * misma página sí tiene el formulario.
 *
 * **Corto a propósito.** Es el retraso que se paga cuando de verdad no hay dónde
 * rellenar, y ahí lo que importa es que la respuesta llegue enseguida. Los guiones
 * de una misma página arrancan casi a la vez, así que si alguno va a acertar, lo
 * dice dentro de este margen.
 */
const GRACIA = 600;

/**
 * pedir habla con el trabajador de fondo, **y nunca lanza**.
 *
 * Si lanzara, `arrancar` se cortaría a media función y el panel se quedaría con
 * la cabecera puesta y nada debajo, sin decir por qué. Eso es exactamente lo que
 * pasó la primera vez que se probó en Firefox, y es el peor fallo posible aquí:
 * el sitio de un fallo que no se ve es la cabeza de quien lo mira.
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
 * **Un puerto y no `sendMessage`**, y es la tercera forma que tienen estas
 * líneas. Con un mensaje suelto hay que prometer que la respuesta llega después,
 * y eso no se promete igual en los dos navegadores: Chrome quiere `return true`
 * y una retrollamada, Firefox quiere la promesa devuelta. Con el `return true`
 * de Chrome, Firefox contesta «Promised response from onMessage listener went
 * out of scope» y aquí no llega nada. Un puerto no promete nada: la respuesta es
 * otro mensaje, igual en los dos.
 */
function porElPuerto(p: Omit<Peticion, "version">): Promise<Respuesta> {
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

/** conPlazo convierte una espera infinita en una respuesta. */
function conPlazo<T>(promesa: Promise<T>): Promise<T> {
  return Promise.race([
    promesa,
    new Promise<T>((_, rechazar) =>
      setTimeout(() => rechazar(new Error("el trabajador de fondo no ha contestado")), PLAZO),
    ),
  ]);
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

/**
 * queHacer traduce un motivo a algo que se pueda leer, **por motivo y no por
 * texto**: los textos de Esfinge cambian, las etiquetas no.
 */
function queHacer(motivo: Motivo | undefined, error: string | undefined): QueHacer {
  switch (motivo) {
    case "sin-esfinge": {
      // El trabajador mete aquí lo que dice el navegador al negarse a lanzar el
      // puente, detrás de «El navegador dice:». **Se separa y se enseña aparte**:
      // es lo único que distingue «no encuentro el manifiesto» de «no puedo
      // ejecutar eso» de «se ha muerto», pero no es una instrucción.
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
      // Aquí el texto de Esfinge dice **por qué** —no es https, es una IP, es un
      // dominio que no se puede reducir—, y eso es más útil que una frase fija.
      return {
        titulo: "Aquí no se rellena",
        texto: error ?? "Esfinge solo rellena en sitios con https.",
        glifo: "mal",
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
  (estado.querySelector(".glifo") as HTMLElement).innerHTML = ICONOS[q.glifo];
  (estado.querySelector("h1") as HTMLElement).textContent = q.titulo;
  (estado.querySelector(".texto") as HTMLElement).textContent = q.texto;
  const detalle = estado.querySelector(".detalle") as HTMLElement;
  detalle.textContent = q.detalle ?? "";
  detalle.hidden = !q.detalle;
}

/** contar pone el resultado del último gesto, bien o mal. */
function contar(texto: string, bien: boolean) {
  resultado.hidden = false;
  resultado.className = `resultado ${bien ? "bien" : "mal"}`;
  resultado.innerHTML = ICONOS[bien ? "bien" : "mal"];
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
  const trozos = [que === "copiar-codigo" ? "Código copiado." : "Contraseña copiada."];
  if (r.copiado.quedan) {
    trozos.push(`Vale ${r.copiado.quedan} s más.`);
  }
  if (r.copiado.portapapeles) {
    trozos.push(`Se borra del portapapeles en ${r.copiado.portapapeles} s.`);
  }
  contar(trozos.join(" "), true);
}

/**
 * rellenar le dice al guion de la página que escriba esta cuenta en el formulario.
 *
 * **Por aquí no pasa ninguna contraseña**, y es a propósito: lo que se manda es un
 * identificador, y quien pide el secreto es el guion de la página, que es el que
 * tiene el campo donde escribirlo.
 *
 * Devuelve el aviso y si ha ido bien. Y por un puerto, como todo lo demás de esta
 * extensión, por la misma razón de siempre: prometer una respuesta para más tarde
 * no se dice igual en los dos navegadores.
 */
function rellenar(pestana: number, cuenta: Cuenta): Promise<[string, boolean]> {
  return new Promise((resolver) => {
    let hecho = false;
    let gracia: ReturnType<typeof setTimeout> | undefined;
    // Lo que dijo la primera trama que contestó que no. Se guarda para poder
    // enseñarlo si al final no acierta nadie, y para distinguir «alguien ha
    // contestado» de «aquí no hay ningún guion».
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
        // **todas las tramas de la pestaña** —`tabs.connect` sin `frameId` lo dice
        // así: «instead of all frames in the tab»— y contesta cada guion que haya.
        // En una página con marcos, el que no tiene formulario puede contestar
        // antes que el que lo tiene, y creerse al primero era decir «aquí no hay
        // ningún formulario» en una página que acababa de rellenarse sola. Costó
        // una versión.
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

      // **Se dispara cuando no hay ningún guion en esa pestaña**, que pasa en las
      // páginas internas del navegador, en las que se abrieron antes de instalar la
      // extensión y en cualquier cosa que no sea https. Pero solo significa eso si
      // no ha contestado nadie: si ya hubo respuesta, lo que hay que enseñar es lo
      // que dijo.
      puerto.onDisconnect.addListener(() => {
        terminar(
          primerFallo ||
            "Esfinge no está puesta en esta página. Si acabas de instalar o actualizar " +
              "la extensión, recarga la pestaña.",
        );
      });

      // Con si tiene código, para que la página no pida uno que no existe cuando
      // además del formulario de entrar hay un campo de segundo factor.
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
  boton.innerHTML = ICONOS[icono];
  boton.title = nombre;
  boton.setAttribute("aria-label", nombre);
  boton.addEventListener("click", async () => {
    boton.disabled = true;
    await alPulsar();
    boton.disabled = false;
  });
  return boton;
}

function fila(cuenta: Cuenta, origen: string, pestana: number | undefined): HTMLElement {
  const li = document.createElement("li");

  const texto = document.createElement("div");
  texto.className = "texto-cuenta";

  const nombre = document.createElement("span");
  nombre.className = "nombre";
  nombre.textContent = cuenta.titulo || "Sin título";
  if (cuenta.titulo) nombre.title = cuenta.titulo;
  texto.append(nombre);

  // **El usuario siempre tiene su línea**, aunque falte: con varias cuentas del
  // mismo sitio es lo único que las distingue, y una fila sin segunda línea se
  // lee como otra clase de cosa. Entero en el `title`, por si no cabe.
  const usuario = document.createElement("span");
  usuario.className = cuenta.usuario ? "usuario" : "usuario sin-usuario";
  usuario.textContent = cuenta.usuario || "Sin usuario";
  if (cuenta.usuario) usuario.title = cuenta.usuario;
  texto.append(usuario);

  li.append(texto);

  const acciones = document.createElement("div");
  acciones.className = "acciones";

  // **Rellenar va primero y es el único con rótulo**, porque desde la entrega 2
  // es lo que se quiere hacer aquí casi siempre. Copiar se queda para lo que no se
  // puede rellenar: una aplicación que dibuja su propio campo, un diálogo del
  // sistema, un sitio en http.
  if (pestana !== undefined) {
    const boton = document.createElement("button");
    boton.className = "rellenar";
    boton.textContent = "Rellenar";
    boton.addEventListener("click", async () => {
      boton.disabled = true;
      const [aviso, bien] = await rellenar(pestana, cuenta);
      contar(aviso, bien);
      boton.disabled = false;
    });
    acciones.append(boton);
  }

  acciones.append(
    botonIcono("llave", "Copiar la contraseña", () => copiar("copiar-secreto", cuenta, origen)),
  );
  // El código solo si la cuenta tiene. `undefined` es una Esfinge anterior a la
  // 2.19.0, que no lo dice: ahí se enseña, como antes, y si no hay contestará.
  if (cuenta.tieneCodigo !== false) {
    acciones.append(
      botonIcono("reloj", "Copiar el código de un solo uso", () =>
        copiar("copiar-codigo", cuenta, origen),
      ),
    );
  } else {
    // **Un hueco del mismo ancho**, y no por simetría: sin él, la fila sin código
    // tiene las acciones más estrechas y su «Rellenar» cae más a la derecha que el
    // de las demás. Se vio en la primera captura del panel nuevo.
    const hueco = document.createElement("span");
    hueco.className = "icono-hueco";
    hueco.setAttribute("aria-hidden", "true");
    acciones.append(hueco);
  }

  li.append(acciones);
  return li;
}

async function arrancar() {
  (document.querySelector(".marca") as HTMLElement).innerHTML = marcaSVG;
  pie.textContent = `Extensión ${api.runtime.getManifest().version}`;

  const [pestana] = await api.tabs.query({ active: true, currentWindow: true });
  const origen = pestana?.url ?? "";
  try {
    donde.textContent = new URL(origen).hostname;
  } catch {
    donde.textContent = "";
  }

  const r = await pedir({ que: "cuentas", origen });
  if (!r.ok) {
    ensenar(queHacer(r.motivo, r.error));
    return;
  }
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
    lista.append(fila(c, origen, pestana?.id));
  }
  cargando.hidden = true;
  lista.hidden = false;
}

// **Con red debajo, y no por costumbre.** Cualquier cosa que se escape aquí deja
// el panel en blanco, que no le dice nada a quien lo mira ni a quien lo va a
// arreglar. Un panel que enseña el error es un panel que se puede depurar por
// teléfono.
arrancar().catch((e) =>
  ensenar({
    titulo: "La extensión ha fallado por dentro",
    texto: "Vuelve a abrir este panel. Si se repite, avisa con el detalle de abajo.",
    detalle: String(e),
    glifo: "mal",
    error: true,
  }),
);
