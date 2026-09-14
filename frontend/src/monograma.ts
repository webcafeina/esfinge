/**
 * El monograma de un sitio: su inicial y cuál de los ocho tintes le toca.
 *
 * **Vive aquí, sin React, porque lo usan dos sitios**: la lista de la bóveda de la
 * ventana (`componentes.tsx`) y el panel de la extensión del navegador. Tienen que
 * dar el mismo color para el mismo sitio, y la forma de garantizarlo es que sea la
 * misma función y no dos copias. Los tintes son tokens de `internal/tema`, con el
 * contraste de la inicial sobre cada uno medido en `contraste_test.go`.
 */

/**
 * dominioDe saca el anfitrión de lo que haya escrito en el campo del sitio, que
 * es texto libre: llegan `https://www.banco.es/login?x=1`, `banco.es`, con
 * espacios y con mayúsculas, según quién lo escribiera o qué gestor lo exportara.
 *
 * Se queda con el anfitrión completo y **no reduce a dominio de segundo nivel**:
 * eso exigiría la lista de sufijos públicos —una dependencia más en un programa
 * que guarda contraseñas— para que `mail.google.com` y `drive.google.com`
 * compartieran color. No compensa: que dos subdominios salgan distintos es
 * inocuo, y con el nombre al lado nadie se pierde.
 */
export function dominioDe(sitio?: string): string {
  if (!sitio) return "";
  const limpio = sitio.trim().toLowerCase();
  if (!limpio) return "";
  try {
    const url = new URL(limpio.includes("://") ? limpio : `https://${limpio}`);
    return url.hostname.replace(/^www\./, "");
  } catch {
    return limpio.replace(/^www\./, "").split("/")[0];
  }
}

/**
 * inicialDe se queda con la primera **letra o cifra**, en mayúscula.
 *
 * Por runas y no por bytes: un título que empiece por «Á» o por «Ñ» tiene que
 * salir entero, y `cadena[0]` de un carácter de dos unidades devuelve medio
 * carácter. Y saltándose lo que no es letra ni cifra, que en un título escrito a
 * mano hay comillas, guiones y corchetes de sobra.
 */
export function inicialDe(texto: string): string {
  for (const c of texto) {
    if (/\p{L}|\p{N}/u.test(c)) return c.toUpperCase();
  }
  return "•";
}

/**
 * tinteDe elige uno de los ocho cuadros, del 1 al 8.
 *
 * Con FNV-1a y **no con el hash que traiga el motor**: el color de un sitio tiene
 * que ser el mismo hoy, mañana y en la otra máquina. Un hash que cambie entre
 * versiones haría que la lista entera cambiara de colores sola, y eso se lee como
 * un fallo aunque no lo sea.
 */
export function tinteDe(clave: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < clave.length; i++) {
    h ^= clave.charCodeAt(i);
    h = Math.imul(h, 0x01000193) >>> 0;
  }
  return (h % 8) + 1;
}
