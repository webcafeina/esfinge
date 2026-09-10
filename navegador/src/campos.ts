/**
 * Encontrar el formulario de entrar en una página que no es nuestra.
 *
 * # Por qué esto es lo difícil de la entrega 2
 *
 * El canal está probado de punta a punta y decide bien: dominio registrable
 * contra dominio registrable, una entrada de otro sitio no sale. Nada de eso
 * sirve si aquí se elige el campo equivocado. **Equivocarse de campo es escribir
 * una contraseña donde la va a leer alguien**: un buscador, un campo de comentario,
 * el formulario de registro de otra cuenta.
 *
 * Así que las reglas de aquí están escritas en negativo. Ante la duda, **no se
 * rellena**: no rellenar es una molestia, rellenar mal es el fallo que hace daño.
 *
 * # Lo que no se hace, y son decisiones
 *
 *   - **No se busca dentro de un `shadowRoot` cerrado**, porque no se puede. Los
 *     abiertos tampoco se recorren todavía: es trabajo de verdad y hasta ver
 *     cuántos sitios lo usan no vale la pena.
 *   - **No se toca un formulario con dos contraseñas visibles.** Eso es registrarse
 *     o cambiar la contraseña, no entrar, y ahí escribir la vieja es lo contrario
 *     de lo que hace falta.
 *   - **No se inventa el usuario cuando no hay ninguno delante.** Muchas páginas
 *     ponen el buscador arriba del todo; cogerlo «porque es el único que queda»
 *     es cómo se escribe un correo electrónico en la caja de búsqueda de un sitio.
 */

/** Un par de campos que juntos sirven para entrar. */
export type Formulario = {
  /** Dónde va el usuario, si es que hay dónde. */
  usuario: HTMLInputElement | null;
  /** Dónde va la contraseña. Nulo en la primera pantalla de las que van en dos. */
  secreto: HTMLInputElement | null;
};

/**
 * Los tipos de campo donde cabe un usuario.
 *
 * `search` no está, y es la ausencia importante: es el campo con el que se
 * confunde el usuario en media web, y lo que se escribe en un buscador acaba en
 * el servidor y en el historial.
 */
const TIPOS_DE_USUARIO = new Set(["text", "email", "tel", ""]);

/** Lo que dice `autocomplete` cuando el campo **no** es para entrar. */
const AUTOCOMPLETADO_AJENO = [
  "new-password",
  "one-time-code",
  "cc-number",
  "cc-csc",
  "cc-name",
  "cc-exp",
];

/**
 * buscarFormularios devuelve los sitios donde se podría entrar en esta página.
 *
 * Devuelve una lista y no uno solo porque las páginas tienen más de uno más a
 * menudo de lo que parece —el de la cabecera y el del cuerpo, el mismo dos veces
 * en dos pestañas—, y quien llama decide qué hacer con eso.
 */
export function buscarFormularios(raiz: Document = document): Formulario[] {
  const contrasenas = [...raiz.querySelectorAll<HTMLInputElement>('input[type="password"]')]
    .filter(sePuedeEscribir)
    .filter((c) => !esAjeno(c));

  const salida: Formulario[] = [];
  const vistos = new Set<HTMLInputElement>();

  for (const secreto of contrasenas) {
    // **Dos contraseñas visibles en el mismo formulario es registrarse.** La
    // segunda es «repite la contraseña», o la primera es la vieja y la segunda la
    // nueva. En los dos casos, rellenar con la guardada está mal.
    if (cuantasContrasenas(secreto, contrasenas) > 1) continue;
    if (vistos.has(secreto)) continue;
    vistos.add(secreto);
    salida.push({ usuario: usuarioPara(secreto, raiz), secreto });
  }

  if (salida.length > 0) return salida;

  // **Y las que van en dos pantallas**, que son las de Google, Microsoft y unas
  // cuantas más: primero el usuario, y la contraseña en la página siguiente. Aquí
  // solo se acepta con la declaración explícita del sitio (`autocomplete`), sin
  // adivinar por el nombre del campo: sin contraseña delante que confirme de qué
  // va el formulario, adivinar es cómo se acaba escribiendo en un buscador.
  const solos = [...raiz.querySelectorAll<HTMLInputElement>("input")]
    .filter(sePuedeEscribir)
    .filter((c) => TIPOS_DE_USUARIO.has(c.type.toLowerCase()))
    .filter((c) => tokens(c).includes("username"));
  return solos.map((usuario) => ({ usuario, secreto: null }));
}

/** cuantasContrasenas dice cuántas hay en el mismo formulario que ésta. */
function cuantasContrasenas(secreto: HTMLInputElement, todas: HTMLInputElement[]): number {
  // Sin `<form>` alrededor —que es media web moderna— el formulario es la página.
  const suyo = secreto.form;
  if (!suyo) return todas.filter((c) => !c.form).length;
  return todas.filter((c) => c.form === suyo).length;
}

/**
 * usuarioPara busca el campo de usuario que acompaña a una contraseña.
 *
 * **Se mira hacia atrás y no hacia delante**, porque en un formulario de entrar el
 * usuario va antes que la contraseña, siempre. Hacia delante están el buscador de
 * la cabecera siguiente, el cupón de descuento y la casilla de la nueva cuenta.
 */
function usuarioPara(secreto: HTMLInputElement, raiz: Document): HTMLInputElement | null {
  const ambito = secreto.form ?? raiz;
  const candidatos = [...ambito.querySelectorAll<HTMLInputElement>("input")].filter(
    (c) => sePuedeEscribir(c) && TIPOS_DE_USUARIO.has(c.type.toLowerCase()) && !esAjeno(c),
  );

  // El último que está **antes** de la contraseña en el orden del documento.
  let elegido: HTMLInputElement | null = null;
  for (const c of candidatos) {
    if (c.compareDocumentPosition(secreto) & Node.DOCUMENT_POSITION_FOLLOWING) {
      elegido = c;
    }
  }
  if (elegido) return elegido;

  // Y si no hay ninguno delante, solo vale cuando el sitio lo dice él mismo. Es la
  // misma cautela de arriba: sin nada delante, cualquier otra regla acaba cogiendo
  // el buscador.
  return candidatos.find((c) => tokens(c).includes("username")) ?? null;
}

/**
 * tokens son las palabras de `autocomplete`, que es lo que el sitio declara.
 *
 * **Se lee el atributo y no la propiedad**: la propiedad está normalizada y
 * devuelve cadena vacía para lo que el navegador no reconoce, así que un sitio con
 * `autocomplete="one-time-code"` en un navegador que no lo entienda se leería como
 * si no hubiera dicho nada. Lo que hace falta aquí es lo que el sitio escribió.
 */
function tokens(campo: HTMLInputElement): string[] {
  return (campo.getAttribute("autocomplete") ?? "")
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean);
}

/** esAjeno dice si el propio sitio ha declarado que ese campo no es para entrar. */
function esAjeno(campo: HTMLInputElement): boolean {
  const suyos = tokens(campo);
  return AUTOCOMPLETADO_AJENO.some((a) => suyos.includes(a));
}

/**
 * sePuedeEscribir junta las tres razones por las que un campo no se toca.
 *
 * La de la visibilidad no es cosmética y es la que evita el fallo silencioso más
 * probable del autorrelleno: **muchos sitios llevan un formulario de entrar
 * escondido en todas sus páginas** —en un menú desplegado, en una ventana que se
 * abre a veces—. Rellenarlo pondría la contraseña en el DOM de cada página del
 * sitio sin que nadie hubiera pedido entrar.
 */
function sePuedeEscribir(campo: HTMLInputElement): boolean {
  if (campo.disabled || campo.readOnly) return false;
  if (campo.type.toLowerCase() === "hidden") return false;
  return esVisible(campo);
}

/**
 * esVisible mira la caja y el estilo calculado.
 *
 * **No se usa `offsetParent`**, que es lo primero que se escribe y está mal: vale
 * nulo para cualquier elemento con `position: fixed`, y un formulario de entrar
 * dentro de un cuadro flotante es exactamente eso.
 */
function esVisible(campo: HTMLInputElement): boolean {
  const caja = campo.getBoundingClientRect();
  // Ocho píxeles: un campo de verdad es más grande, y los de mentira que ponen
  // algunos sitios para engañar a los rellenadores miden uno o cero.
  if (caja.width < 8 || caja.height < 8) return false;
  const estilo = getComputedStyle(campo);
  if (estilo.visibility === "hidden" || estilo.display === "none") return false;
  return Number(estilo.opacity || "1") >= 0.1;
}

/**
 * escribir pone un valor en un campo **como si lo hubiera tecleado una persona**.
 *
 * # Esto es la lección de `CLAUDE.md`, otra vez y en casa ajena
 *
 * Este proyecto ya perdió versiones con esto: React sustituye la propiedad `value`
 * **del elemento concreto** por un accesor que mantiene su registro interno, así
 * que al asignar directamente ese registro se pone al día antes de tiempo y el
 * evento `input` que se dispara después **no le parece un cambio**. En pantalla el
 * valor está puesto y para la aplicación el campo sigue vacío: el botón de entrar
 * no se activa.
 *
 * Allí eran nuestros formularios. Aquí son los de todo el mundo, y la mitad están
 * hechos con React, así que lo que allí costó encontrar aquí es la norma. Se
 * escribe con el accesor **del prototipo**, que no toca el registro del elemento,
 * y luego se avisa.
 */
export function escribir(campo: HTMLInputElement, valor: string) {
  const accesor = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
  if (accesor) {
    accesor.call(campo, valor);
  } else {
    campo.value = valor;
  }
  // Y los eventos que espera un formulario cualquiera. El orden importa: hay
  // sitios que solo miran `change`, y sitios que validan al perder el foco.
  campo.dispatchEvent(new Event("input", { bubbles: true }));
  campo.dispatchEvent(new Event("change", { bubbles: true }));
}
