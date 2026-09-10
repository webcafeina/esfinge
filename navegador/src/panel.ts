/**
 * El panel: lo que se ve al pulsar el botón de Esfinge.
 *
 * Enseña las cuentas que hay para el sitio de la pestaña y copia lo que se le
 * pida. **No escribe nada en la página** —eso es la entrega siguiente— y **nunca
 * ve un secreto**: copia Esfinge, y por el canal solo vuelve cuánto tardará en
 * borrarse del portapapeles.
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
 * pedir habla con el trabajador de fondo, **y nunca lanza**.
 *
 * Si lanzara, `arrancar` se cortaría a media función y el panel se quedaría con
 * la cabecera puesta y nada debajo, sin decir por qué. Eso es exactamente lo que
 * pasó la primera vez que se probó en Firefox, y es el peor fallo posible aquí:
 * el sitio de un fallo que no se ve es la cabeza de quien lo mira.
 */
async function pedir(p: Omit<Peticion, "version">): Promise<Respuesta> {
  try {
    const r = (await conPlazo(api.runtime.sendMessage(p))) as Respuesta | undefined;
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
      return "Esfinge no está abierta, o su canal con el navegador está apagado en Ajustes.";
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

function fila(cuenta: Cuenta, origen: string): HTMLElement {
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
    lista.append(fila(c, origen));
  }
  lista.hidden = false;
  aviso.hidden = true;
}

// **Con red debajo, y no por costumbre.** Cualquier cosa que se escape aquí deja
// el panel en blanco, que no le dice nada a quien lo mira ni a quien lo va a
// arreglar. Un panel que enseña el error es un panel que se puede depurar por
// teléfono.
arrancar().catch((e) => decir(`Algo ha fallado dentro de la extensión: ${e}`));
