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

/* ------------------------------------------------ el código de un solo uso */

/**
 * Dónde va el código de segundo factor: un campo, o una fila de casillas de un
 * carácter.
 */
export type DestinoDeCodigo =
  | { tipo: "uno"; campo: HTMLInputElement }
  | { tipo: "casillas"; campos: HTMLInputElement[] };

/** Los tipos de campo donde cabe un código. */
const TIPOS_DE_CODIGO = new Set(["text", "tel", "number", ""]);

/**
 * Los nombres que dicen «esto es un segundo factor».
 *
 * **«code» a secas no está, y es la ausencia importante**: es el código postal,
 * el promocional, el de la tarjeta regalo y el de verificación del CVC. Todos
 * esos tienen seis caracteres a menudo, y escribir ahí un código de un solo uso
 * es regalarlo a un formulario que no lo pidió.
 */
const NOMBRE_DE_CODIGO =
  /(^|[^a-z])(otp|totp|mfa|2fa)([^a-z]|$)|two.?factor|one.?time|authenticat|verification.?code/i;

/**
 * buscarCodigo encuentra dónde escribir un código de un solo uso, **o nada**.
 *
 * Tres formas, en orden de cuánto hay que creerse a la página:
 *
 *   1. **Lo declara el sitio**: `autocomplete="one-time-code"`, que es el estándar.
 *      Si hay más de uno declarado, no se sabe cuál y no se toca.
 *   2. **Seis u ocho casillas de un carácter**, juntas. Es la forma más común de
 *      los formularios de segundo factor que no declaran nada, y la que un campo
 *      suelto no detecta. **Solo seis u ocho**: cuatro es un PIN, y un PIN no es
 *      esto.
 *   3. **Por el nombre**, y solo si en la página no hay ninguna contraseña visible
 *      y el campo tiene cara de numérico. Es la regla más débil, así que es la
 *      que más condiciones lleva.
 *
 * Y lo mismo que en todo este fichero: **ante la duda, nada**. Un código escrito
 * en el campo equivocado es un segundo factor regalado.
 */
export function buscarCodigo(raiz: Document = document): DestinoDeCodigo | null {
  const candidatos = [...raiz.querySelectorAll<HTMLInputElement>("input")]
    .filter(sePuedeEscribir)
    .filter((c) => TIPOS_DE_CODIGO.has(c.type.toLowerCase()));

  // 1. **Las casillas, antes que nada.** Hasta la 2.19.0 iban segundas, y el
  // formulario de Cloudflare no se detectaba por eso: sus seis casillas declaran
  // todas `one-time-code`, así que la regla del campo declarado veía seis y se
  // callaba por no saber cuál. Una fila de seis u ocho campos que se parecen es
  // una forma inconfundible; un campo declarado suelto, no tanto.
  //
  // Y cuenta como casilla más de lo que contaba: `maxlength="1"`, o un `pattern`
  // de una sola cifra —que es lo que llevan las de Cloudflare, donde **ninguna**
  // tiene `maxlength="1"` porque la primera acepta el código entero para el
  // autorrelleno del sistema—, o declarar `one-time-code` dentro de un grupo.
  const posibles = candidatos.filter(
    (c) => !esDeTarjeta(c) && (esCasilla(c) || tokens(c).includes("one-time-code")),
  );
  const grupos = agruparCasillas(posibles).filter((g) => g.length === 6 || g.length === 8);
  if (grupos.length === 1) return { tipo: "casillas", campos: grupos[0] };
  if (grupos.length > 1) return null;

  // 2. Lo que el sitio declara, si es un solo campo entero. Si hay más de uno, o
  // el único que hay es una casilla suelta, no se sabe dónde va y no se toca.
  const declarados = candidatos.filter((c) => tokens(c).includes("one-time-code"));
  if (declarados.length === 1 && !esCasilla(declarados[0])) {
    return { tipo: "uno", campo: declarados[0] };
  }
  if (declarados.length > 0) return null;

  // 3. Por el nombre, con todas las cautelas.
  const hayContrasena = [...raiz.querySelectorAll<HTMLInputElement>('input[type="password"]')].some(
    sePuedeEscribir,
  );
  if (hayContrasena) return null;
  const porNombre = candidatos.filter(
    (c) =>
      !esDeTarjeta(c) &&
      NOMBRE_DE_CODIGO.test(
        [c.name, c.id, c.getAttribute("aria-label") ?? "", c.placeholder].join(" "),
      ) &&
      pareceNumerico(c),
  );
  return porNombre.length === 1 ? { tipo: "uno", campo: porNombre[0] } : null;
}

/**
 * esCasilla: un campo hecho para una sola cifra. O lo dice `maxlength`, o lo dice
 * el `pattern` —`\\d`, `\\d{1}`, `[0-9]`—, que es lo que usan los componentes que no
 * pueden poner `maxlength="1"` porque la primera casilla tiene que aceptar el
 * código entero.
 */
function esCasilla(campo: HTMLInputElement): boolean {
  return campo.maxLength === 1 || /^(\\d|\[0-9\])(\{1\})?$/.test(campo.pattern);
}

/** esDeTarjeta: lo que el sitio declara de una tarjeta no es un segundo factor. */
function esDeTarjeta(campo: HTMLInputElement): boolean {
  return tokens(campo).some((t) => t.startsWith("cc-"));
}

/** pareceNumerico: las señales de que un campo espera seis u ocho cifras. */
function pareceNumerico(campo: HTMLInputElement): boolean {
  return (
    campo.inputMode === "numeric" ||
    campo.type === "tel" ||
    campo.type === "number" ||
    campo.maxLength === 6 ||
    campo.maxLength === 8 ||
    /\\d|\[0-9\]/.test(campo.pattern)
  );
}

/**
 * agruparCasillas junta las casillas que van juntas.
 *
 * **Muchas van cada una en su caja** —un `div` por casilla, para dibujarle el
 * borde—, así que agrupar por el padre directo daría seis grupos de una. Cuando la
 * caja solo contiene esa casilla, se sube un nivel.
 */
function agruparCasillas(casillas: HTMLInputElement[]): HTMLInputElement[][] {
  const porPadre = new Map<Element, HTMLInputElement[]>();
  for (const c of casillas) {
    let padre = c.parentElement;
    if (padre && padre.querySelectorAll("input").length === 1 && padre.parentElement) {
      padre = padre.parentElement;
    }
    if (!padre) continue;
    const grupo = porPadre.get(padre) ?? [];
    grupo.push(c);
    porPadre.set(padre, grupo);
  }
  return [...porPadre.values()];
}

/** camposDe devuelve los campos de un destino, para marcarlos como rellenados. */
export function camposDe(destino: DestinoDeCodigo): HTMLInputElement[] {
  return destino.tipo === "uno" ? [destino.campo] : destino.campos;
}

/** Lo que se deja al componente de la página para que se entere antes de mirar. */
const unRespiro = () => new Promise<void>((r) => setTimeout(r, 30));

/**
 * escribirCodigo pone el código en su sitio **y dice si ha quedado puesto**.
 *
 * Las casillas se escriben una a una, como si se tecleara, y al final **se lee lo
 * que hay**: «he escrito» no es «ha quedado puesto», y la diferencia es un botón de
 * verificar que no se activa. Si no ha quedado, se devuelve falso y quien llama lo
 * dice, en vez de contar que ha ido bien.
 *
 * **Y una cosa que se creyó y no era**, para que no vuelva a escribirse: leyendo el
 * código del componente de Cloudflare (Base UI, `otp-field`) parecía que cada
 * casilla recalculaba el código desde el valor de la última vez que se dibujó, y
 * que escribir las seis seguidas las haría pisarse. Se escribió un rodeo para eso
 * —el código entero en la primera casilla— y **la prueba contra el componente de
 * verdad dijo que no hacía falta**: React atiende cada `input` al momento, y cada
 * casilla ya ve lo que dejó la anterior. El rodeo se quitó. Lo comprueba
 * `pruebas/otp-de-verdad.spec.ts`.
 */
export async function escribirCodigo(destino: DestinoDeCodigo, codigo: string): Promise<boolean> {
  if (destino.tipo === "uno") {
    escribir(destino.campo, codigo);
    return true;
  }
  const { campos } = destino;
  // Seis casillas y un código de ocho cifras no es sitio donde escribir a medias.
  if (campos.length !== codigo.length) return false;

  campos.forEach((campo, i) => escribir(campo, codigo[i]));
  await unRespiro();
  return campos.map((c) => c.value).join("") === codigo;
}
