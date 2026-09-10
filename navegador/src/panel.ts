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
 */
import { api } from "./api";
import type { Cuenta, Motivo, Peticion, Respuesta } from "./protocolo";

const donde = document.getElementById("donde") as HTMLElement;
const lista = document.getElementById("lista") as HTMLElement;
const aviso = document.getElementById("aviso") as HTMLElement;
const pie = document.getElementById("pie") as HTMLElement;

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

/** Lo que se enseña cuando algo no se puede hacer, por motivo y no por texto. */
function queHacer(motivo: Motivo | undefined, error: string | undefined): string {
  switch (motivo) {
    case "sin-esfinge":
      // **El texto de dentro, no una frase fija.** Aquí es donde el trabajador
      // mete lo que dice el navegador al negarse a lanzar el puente, y es lo
      // único que distingue «no encuentro el manifiesto» de «no puedo ejecutar
      // eso» de «se ha muerto». Tenerlo y no enseñarlo —que es lo que hacía esta
      // línea— es el mismo error de todo el día con otra cara.
      return error ?? "Esfinge no está abierta, o su canal está apagado en sus Ajustes.";
    case "sin-emparejar":
      return "Permite este navegador en la ventana de Esfinge, en Ajustes, y vuelve a abrir esto.";
    case "cerrada":
      return "La bóveda está cerrada. Ábrela en Esfinge.";
    case "sin-boveda":
      return "Todavía no hay ninguna bóveda. Créala en Esfinge.";
    case "origen":
      // Aquí el texto de Esfinge dice **por qué** —no es https, es una IP, es un
      // dominio que no se puede reducir— y eso es más útil que una frase fija.
      return error ?? "Aquí no se puede rellenar.";
    case "demasiado":
      return "Demasiadas preguntas seguidas. Espera un momento.";
    default:
      return error ?? "Algo no ha ido bien.";
  }
}

function decir(texto: string) {
  aviso.textContent = texto;
  aviso.hidden = false;
}

/** copiar pide a Esfinge que copie, y cuenta lo que ha pasado. */
async function copiar(que: "copiar-secreto" | "copiar-codigo", cuenta: Cuenta, origen: string) {
  const r = await pedir({ que, id: cuenta.id, origen });
  if (!r.ok || !r.copiado) {
    decir(queHacer(r.motivo, r.error));
    return;
  }
  const trozos = [que === "copiar-codigo" ? "Código copiado." : "Contraseña copiada."];
  if (r.copiado.quedan) {
    trozos.push(`Vale ${r.copiado.quedan} s más.`);
  }
  if (r.copiado.portapapeles) {
    trozos.push(`Se borra del portapapeles en ${r.copiado.portapapeles} s.`);
  }
  decir(trozos.join(" "));
}

/**
 * rellenar le dice al guion de la página que escriba esta cuenta en el formulario.
 *
 * **Por aquí no pasa ninguna contraseña**, y es a propósito: lo que se manda es un
 * identificador, y quien pide el secreto es el guion de la página, que es el que
 * tiene el campo donde escribirlo. El panel se cierra solo al perder el foco, que
 * es la peor clase de sitio donde dejar un secreto aunque fuera un instante.
 *
 * Y por un puerto, como todo lo demás de esta extensión, por la misma razón de
 * siempre: prometer una respuesta para más tarde no se dice igual en los dos
 * navegadores.
 */
function rellenar(pestana: number, cuenta: Cuenta): Promise<string> {
  return new Promise((resolver) => {
    let hecho = false;
    let gracia: ReturnType<typeof setTimeout> | undefined;
    // Lo que dijo la primera trama que contestó que no. Se guarda para poder
    // enseñarlo si al final no acierta nadie, y para distinguir «alguien ha
    // contestado» de «aquí no hay ningún guion».
    let primerFallo = "";

    const terminar = (aviso: string) => {
      if (hecho) return;
      hecho = true;
      clearTimeout(plazo);
      clearTimeout(gracia);
      resolver(aviso);
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
          terminar("Rellenado.");
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

      puerto.postMessage({ id: cuenta.id });
    } catch (e) {
      terminar(`No se ha podido rellenar: ${e}`);
    }
  });
}

function fila(cuenta: Cuenta, origen: string, pestana: number | undefined): HTMLElement {
  const li = document.createElement("li");

  const nombre = document.createElement("span");
  nombre.className = "nombre";
  nombre.textContent = cuenta.titulo || "Sin título";
  li.append(nombre);

  if (cuenta.usuario) {
    const usuario = document.createElement("span");
    usuario.className = "usuario";
    usuario.textContent = cuenta.usuario;
    li.append(usuario);
  }

  const acciones = document.createElement("span");
  acciones.className = "acciones";

  // **Rellenar va primero y destacado**, porque desde la entrega 2 es lo que se
  // quiere hacer aquí el noventa por ciento de las veces. Copiar se queda para lo
  // que no se puede rellenar: una aplicación que dibuja su propio campo, un
  // diálogo del sistema, un sitio en http.
  if (pestana !== undefined) {
    const boton = document.createElement("button");
    boton.textContent = "Rellenar";
    boton.className = "principal";
    boton.addEventListener("click", async () => {
      boton.disabled = true;
      decir(await rellenar(pestana, cuenta));
      boton.disabled = false;
    });
    acciones.append(boton);
  }

  for (const [texto, que] of [
    ["Contraseña", "copiar-secreto"],
    ["Código", "copiar-codigo"],
  ] as const) {
    const boton = document.createElement("button");
    boton.textContent = texto;
    boton.addEventListener("click", () => copiar(que, cuenta, origen));
    acciones.append(boton);
  }
  li.append(acciones);
  return li;
}

async function arrancar() {
  // **Lo primero que se ve es que está preguntando.** Un panel en blanco no
  // distingue «está pensando» de «se ha roto», y quien lo mira no tiene forma de
  // saber cuál de las dos es.
  decir("Preguntando a Esfinge…");
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
    decir(queHacer(r.motivo, r.error));
    return;
  }
  const cuentas = r.cuentas ?? [];
  if (cuentas.length === 0) {
    decir("No tienes ninguna cuenta guardada de este sitio.");
    return;
  }
  for (const c of cuentas) {
    lista.append(fila(c, origen, pestana?.id));
  }
  lista.hidden = false;
  aviso.hidden = true;
}

// **Con red debajo, y no por costumbre.** Cualquier cosa que se escape aquí deja
// el panel en blanco, que no le dice nada a quien lo mira ni a quien lo va a
// arreglar. Un panel que enseña el error es un panel que se puede depurar por
// teléfono.
arrancar().catch((e) => decir(`Algo ha fallado dentro de la extensión: ${e}`));
