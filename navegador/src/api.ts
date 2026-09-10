/**
 * El mismo API con dos nombres, y no es un detalle: **rompió la extensión
 * entera en Firefox el primer día**.
 *
 * Los dos navegadores traen las dos formas, pero no las mismas:
 *
 *   - En Chrome, `chrome.*` **devuelve promesas** desde MV3.
 *   - En Firefox, `chrome.*` existe solo por compatibilidad y es **de
 *     retrollamada**: llamarlo sin una devuelve `undefined`. Las promesas están
 *     en `browser.*`.
 *
 * Escrito con `chrome.*` y un `await` delante, en Chrome funciona y en Firefox
 * `await` recibe `undefined`, lo siguiente revienta con un error de tipo y —si
 * nadie lo recoge— **el panel se queda en blanco sin decir nada**. Que es
 * exactamente lo que pasó: se veía la cabecera con el sitio y debajo, nada.
 *
 * Con `browser` primero, los dos dan promesas y el código es uno.
 */
type ApiDeExtension = typeof chrome;

const global = globalThis as unknown as {
  browser?: ApiDeExtension;
  chrome?: ApiDeExtension;
};

export const api: ApiDeExtension = global.browser ?? (global.chrome as ApiDeExtension);
