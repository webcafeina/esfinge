import type { Orden } from "./puente";

/**
 * obedecer hace lo que pide el menú del sistema dentro de la ventana.
 *
 * El menú lo dibuja el sistema y vive en Go, pero casi todo lo que se pide desde
 * él pasa aquí dentro: cambiar de pestaña, copiar lo que hay seleccionado, pegar
 * en el campo que tiene el foco. Go no sabe cuál es ese campo; el navegador sí.
 *
 * Devuelve true si la orden era de edición y se ha atendido, para que quien
 * llama sepa si le toca a él ocuparse del resto.
 */
const COMANDOS: Record<string, string> = {
  "editar:deshacer": "undo",
  "editar:rehacer": "redo",
  "editar:cortar": "cut",
  "editar:copiar": "copy",
};

export function obedecerEdicion(o: Orden): boolean {
  const foco = document.activeElement;
  const campo =
    foco instanceof HTMLInputElement || foco instanceof HTMLTextAreaElement ? foco : null;

  // execCommand está marcado como obsoleto, pero para los comandos de edición
  // sigue siendo lo único que funciona igual en WKWebView y en WebView2, que son
  // los dos motores que lleva Esfinge. La alternativa moderna —la API de
  // portapapeles— pide permisos que en una ventana de escritorio no hay a quién
  // pedirle.
  const comando = COMANDOS[o.que];
  if (comando) {
    document.execCommand(comando);
    return true;
  }

  switch (o.que) {
    case "editar:seleccionar-todo":
      if (campo) campo.select();
      else document.execCommand("selectAll");
      return true;

    case "editar:pegar":
      pegarEn(campo, o.texto);
      return true;

    default:
      return false;
  }
}

/**
 * pegarEn mete el texto donde está el cursor, respetando lo que hubiera
 * seleccionado, y deja el cursor detrás de lo pegado.
 *
 * Se hace a mano porque el texto viene de Go: el navegador no puede leer el
 * portapapeles del sistema, así que tampoco puede pegarlo él.
 */
function pegarEn(campo: HTMLInputElement | HTMLTextAreaElement | null, texto: string) {
  if (!campo || !texto) return;

  const desde = campo.selectionStart ?? campo.value.length;
  const hasta = campo.selectionEnd ?? desde;
  campo.value = campo.value.slice(0, desde) + texto + campo.value.slice(hasta);

  const cursor = desde + texto.length;
  campo.setSelectionRange(cursor, cursor);

  // React no se entera de un cambio hecho sobre el valor del elemento: hay que
  // decírselo con el evento que él escucha.
  campo.dispatchEvent(new Event("input", { bubbles: true }));
}
