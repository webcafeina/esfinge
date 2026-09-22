/**
 * Qué cuentas son de qué sitio, **igual que `internal/navegador/dominios.go`**.
 *
 * Es la pieza que hay que hacer bien: decidir mal aquí es entregar una contraseña
 * al sitio equivocado. Con cuenta, la extensión lo decide sola (ADR 0040), así que
 * tiene que decidirlo exactamente como Go, y lo vigila una prueba cruzada.
 *
 * Las reglas, las mismas: solo `https`; nada de IP ni de anfitriones que no sean
 * ASCII; y se compara **dominio registrable contra dominio registrable**, con la
 * lista de sufijos públicos **incluidos los privados** —si no, `foo.github.io` y
 * `bar.github.io` serían el mismo sitio—. Go usa la de `golang.org/x/net`; aquí,
 * la de `tldts`. Son dos copias de la misma lista con fechas distintas, y un sufijo
 * muy reciente podría estar en una y no en la otra: la prueba cruzada mira los
 * casos que importan.
 */

import { getDomain } from "tldts";

export const ERR_SIN_DOMINIO = "De esa dirección no se puede sacar un dominio";

function esIP(h: string): boolean {
  return /^\d{1,3}(\.\d{1,3}){3}$/.test(h) || h.includes(":");
}

/** Reduce un anfitrión a lo que se puede registrar, o `null`. */
export function dominioRegistrable(anfitrion: string): string | null {
  const h = anfitrion.trim().replace(/\.$/, "").toLowerCase();
  if (!h || esIP(h) || /[^\x00-\x7f]/.test(h)) return null;
  const d = getDomain(h, { allowPrivateDomains: true, extractHostname: false, validateHostname: false });
  return d ?? null;
}

/**
 * **Lo que no es ASCII se rechaza mirando lo escrito, antes de analizarlo.** El
 * navegador convierte solo `bücher.de` en `xn--bcher-kva.de`, y Go no: allí no se
 * rellena, así que aquí tampoco. Lo cazó la prueba cruzada.
 */
function autoridadNoAscii(url: string): boolean {
  const m = /^[a-z][a-z0-9+.-]*:\/\/([^/?#]*)/i.exec(url);
  return m !== null && /[^\x00-\x7f]/.test(m[1]);
}

function anfitrionDe(url: string): string | null {
  if (autoridadNoAscii(url)) return null;
  try {
    return new URL(url).hostname.replace(/^\[|\]$/g, "");
  } catch {
    return null;
  }
}

/** El dominio de la pestaña: solo `https`. Lanza con la frase de Go si no se puede. */
export function dominioDeOrigen(origen: string): string {
  let u: URL;
  if (autoridadNoAscii(origen.trim())) throw new Error(ERR_SIN_DOMINIO);
  try {
    u = new URL(origen.trim());
  } catch {
    throw new Error(ERR_SIN_DOMINIO);
  }
  const esquema = u.protocol.replace(/:$/, "");
  if (esquema.toLowerCase() !== "https") throw new Error(`Esfinge solo rellena en https, y eso es «${esquema}»`);
  const d = dominioRegistrable(u.hostname.replace(/^\[|\]$/g, ""));
  if (!d) throw new Error(ERR_SIN_DOMINIO);
  return d;
}

/** El dominio de lo guardado en una entrada, tolerante con la forma; vacío si no sale ninguno. */
export function dominioDeSitio(sitio: string): string {
  let s = sitio.trim();
  if (!s) return "";
  if (!s.includes("://")) s = "https://" + s;
  const h = anfitrionDe(s);
  if (h === null) return "";
  return dominioRegistrable(h) ?? "";
}

/** Si lo guardado corresponde a la pestaña. */
export function encaja(sitioGuardado: string, dominioDeLaPestana: string): boolean {
  return dominioDeLaPestana !== "" && dominioDeSitio(sitioGuardado) === dominioDeLaPestana;
}

/** El anfitrión de la dirección que da el navegador, en minúsculas. */
export function hostDe(origen: string): string {
  return (anfitrionDe(origen.trim()) ?? "").toLowerCase();
}
