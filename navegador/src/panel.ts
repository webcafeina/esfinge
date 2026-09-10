/**
 * El panel: lo que se ve al pulsar el botón de Esfinge.
 *
 * Enseña las cuentas que hay para el sitio de la pestaña y copia lo que se le
 * pida. **No escribe nada en la página** —eso es la entrega siguiente— y **nunca
 * ve un secreto**: copia Esfinge, y por el canal solo vuelve cuánto tardará en
 * borrarse del portapapeles.
 *
 * La dirección de la pestaña la da el navegador (`chrome.tabs`), no la página. Es
 * la diferencia entre preguntar por un sitio y preguntar por lo que un documento
 * dice que es.
 */
import type { Cuenta, Motivo, Peticion, Respuesta } from "./protocolo";

const donde = document.getElementById("donde") as HTMLElement;
const lista = document.getElementById("lista") as HTMLElement;
const aviso = document.getElementById("aviso") as HTMLElement;

function pedir(p: Omit<Peticion, "version">): Promise<Respuesta> {
  return chrome.runtime.sendMessage(p);
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
  const [pestana] = await chrome.tabs.query({ active: true, currentWindow: true });
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
}

arrancar();
