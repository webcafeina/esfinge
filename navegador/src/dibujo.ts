/**
 * Poner uno de nuestros dibujos dentro de un elemento, **sin `innerHTML`**.
 *
 * Los dibujos son nuestros —la marca, la silueta de la barra y los iconos del panel,
 * escritos en el código—, así que `innerHTML` no metía nada ajeno. Pero es lo primero
 * que marca la revisión de Mozilla (`UNSAFE_VAR_ASSIGNMENT`), y en una extensión que
 * lee contraseñas conviene que no haya ni una línea que un revisor tenga que pararse
 * a descartar (ADR 0033). Se parsean como SVG y se insertan como nodos.
 */

const ESPACIO_SVG = "http://www.w3.org/2000/svg";

/** dibujar sustituye el contenido de `donde` por el dibujo. Si no es un SVG, lo deja vacío. */
export function dibujar(donde: Element, svg: string): void {
  // Los iconos del panel van sin `xmlns`, que en HTML no hace falta y al parsearlos
  // como SVG sí: sin él, el navegador no los dibuja.
  const conEspacio = svg.includes("xmlns=") ? svg : svg.replace("<svg", `<svg xmlns="${ESPACIO_SVG}"`);
  const leido = new DOMParser().parseFromString(conEspacio, "image/svg+xml").documentElement;
  const doc = donde.ownerDocument ?? document;
  if (leido.namespaceURI !== ESPACIO_SVG || leido.localName !== "svg") {
    donde.replaceChildren();
    return;
  }
  donde.replaceChildren(doc.importNode(leido, true));
}
