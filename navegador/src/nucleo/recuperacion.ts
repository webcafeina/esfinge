/**
 * La clave de recuperación, `ESF-XXXX-…`, igual que `internal/boveda/recuperacion.go`:
 * 16 bytes al azar más un byte de control (el primero de su SHA-256), en base32 de
 * Crockford y en grupos de cuatro.
 *
 * Lo que abre la ranura **es el texto normalizado**, «ESF» y los 28 caracteres
 * seguidos, no los bytes decodificados. Ver `Normalizar` en Go.
 */

import { azarDe } from "./esf1";

const PREFIJO = "ESF";
const ALFABETO = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";
const BYTES_DE_AZAR = 16;

export const ERR_CHECKSUM =
  "Esa no parece una clave de recuperación de Esfinge: revísala, puede que falte o sobre un carácter";

async function sha256(b: Uint8Array): Promise<Uint8Array> {
  return new Uint8Array(await crypto.subtle.digest("SHA-256", new Uint8Array(b)));
}

function codificar(b: Uint8Array): string {
  let out = "";
  let acumulado = 0;
  let bits = 0;
  for (const x of b) {
    acumulado = ((acumulado << 8) | x) & 0xffff;
    bits += 8;
    while (bits >= 5) {
      bits -= 5;
      out += ALFABETO[(acumulado >> bits) & 31];
    }
  }
  if (bits > 0) out += ALFABETO[(acumulado << (5 - bits)) & 31];
  return out;
}

function decodificar(s: string): Uint8Array | null {
  const out: number[] = [];
  let acumulado = 0;
  let bits = 0;
  for (const r of s) {
    const i = ALFABETO.indexOf(r);
    if (i < 0) return null;
    acumulado = ((acumulado << 5) | i) & 0xffff;
    bits += 5;
    if (bits >= 8) {
      bits -= 8;
      out.push((acumulado >> bits) & 0xff);
    }
  }
  // Los bits de relleno tienen que ser cero: si no, una errata en el último
  // carácter pasaría por buena (lo encontró una prueba en Go).
  if (bits > 0 && (acumulado & ((1 << bits) - 1)) !== 0) return null;
  return Uint8Array.from(out);
}

/** Una clave nueva, ya presentada para enseñarla. */
export async function nuevaRecuperacion(): Promise<string> {
  const b = azarDe(BYTES_DE_AZAR);
  const h = await sha256(b);
  const con = new Uint8Array(BYTES_DE_AZAR + 1);
  con.set(b);
  con[BYTES_DE_AZAR] = h[0];
  const texto = codificar(con);
  let out = PREFIJO;
  for (let i = 0; i < texto.length; i += 4) out += "-" + texto.slice(i, i + 4);
  return out;
}

export function pareceRecuperacion(tecleado: string): boolean {
  return tecleado.trim().toUpperCase().startsWith(PREFIJO);
}

/**
 * La forma canónica de lo tecleado, con la suma comprobada, o `null` si no es una
 * clave de recuperación. Acepta minúsculas, sin guiones y con O, I y L por 0 y 1.
 */
export async function normalizar(tecleado: string): Promise<string | null> {
  let limpio = tecleado.trim().toUpperCase();
  if (limpio.startsWith(PREFIJO)) limpio = limpio.slice(PREFIJO.length);
  let texto = "";
  for (const r of limpio) {
    if ("- \t\n\r_".includes(r)) continue;
    if (r === "O") texto += "0";
    else if (r === "I" || r === "L") texto += "1";
    else if (ALFABETO.includes(r)) texto += r;
    else return null;
  }
  const b = decodificar(texto);
  if (!b || b.length !== BYTES_DE_AZAR + 1) return null;
  const h = await sha256(b.subarray(0, BYTES_DE_AZAR));
  if (h[0] !== b[BYTES_DE_AZAR]) return null;
  return PREFIJO + texto;
}
