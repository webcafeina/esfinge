import { esfinge, type Orden } from "./puente";

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

  // **Copiar de un campo de contraseña no lo hace el navegador.** WebKit y
  // Chromium se niegan a copiar de un `type="password"`, a propósito y sin decir
  // nada: `execCommand("copy")` devuelve que sí y el portapapeles se queda como
  // estaba. Eso deja «Generar una» en la pantalla de cifrar sin forma de sacar la
  // contraseña que acaba de fabricar, que es justo la que no está en ningún otro
  // sitio.
  //
  // Se copia por Go, que además es el camino que arma el borrado del portapapeles
  // pasado el plazo. Para un secreto eso es mejor que el del navegador, no peor.
  if (o.que === "editar:copiar" || o.que === "editar:cortar") {
    const secreto = loSeleccionadoDeUnaContrasena(campo);
    if (secreto !== null) {
      esfinge.copiar(secreto).catch(() => {});
      if (o.que === "editar:cortar") quitarLoSeleccionado(campo!);
      return true;
    }
  }

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
 * loSeleccionadoDeUnaContrasena devuelve lo que haya seleccionado en un campo de
 * contraseña, o null si no es ese caso —y entonces manda el camino de siempre—.
 */
function loSeleccionadoDeUnaContrasena(
  campo: HTMLInputElement | HTMLTextAreaElement | null,
): string | null {
  if (!(campo instanceof HTMLInputElement) || campo.type !== "password") return null;
  const desde = campo.selectionStart ?? 0;
  const hasta = campo.selectionEnd ?? 0;
  if (hasta <= desde) return null;
  return campo.value.slice(desde, hasta);
}

function quitarLoSeleccionado(campo: HTMLInputElement | HTMLTextAreaElement) {
  const desde = campo.selectionStart ?? 0;
  const hasta = campo.selectionEnd ?? desde;
  ponerValor(campo, campo.value.slice(0, desde) + campo.value.slice(hasta));
  campo.setSelectionRange(desde, desde);
  campo.dispatchEvent(new Event("input", { bubbles: true }));
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
  ponerValor(campo, campo.value.slice(0, desde) + texto + campo.value.slice(hasta));

  const cursor = desde + texto.length;
  campo.setSelectionRange(cursor, cursor);

  // React no se entera de un cambio hecho sobre el valor del elemento: hay que
  // decírselo con el evento que él escucha.
  campo.dispatchEvent(new Event("input", { bubbles: true }));
}

/**
 * ponerValor escribe en el campo **de forma que React lo vea**, que es harina de
 * otro costal que un `campo.value = …`.
 *
 * React lleva su propio registro del último valor de cada campo, y para
 * mantenerlo sustituye la propiedad `value` **del elemento concreto** por un
 * accesor que lo actualiza. Asignando directamente, ese registro se pone al día
 * antes de tiempo: cuando después llega el evento «input», React compara el valor
 * con lo que tiene apuntado, no encuentra diferencia y **no dispara onChange**.
 * El estado se queda con lo de antes.
 *
 * Eso es lo que hacía que pegar una clave con ⌘V no activara el botón de cifrar:
 * en pantalla estaba puesta y para React el campo seguía vacío. Nunca se había
 * notado porque la prueba del menú ejercitaba «seleccionar todo» y no pegar.
 *
 * El accesor original sigue en el prototipo, y llamarlo desde ahí escribe el
 * valor sin tocar el registro. Entonces el evento sí cuenta como un cambio.
 */
function ponerValor(campo: HTMLInputElement | HTMLTextAreaElement, valor: string) {
  const nativo = Object.getOwnPropertyDescriptor(
    Object.getPrototypeOf(campo),
    "value",
  )?.set;

  if (nativo) nativo.call(campo, valor);
  else campo.value = valor;
}
