/**
 * La identidad de la bóveda para compartir copias, en TypeScript (ADR 0043).
 *
 * **El espejo de `internal/boveda/identidad.go`**, y como todo lo de `nucleo/`,
 * tiene que dar exactamente lo mismo: las mismas llaves y la misma huella para la
 * misma semilla. Lo vigilan las pruebas cruzadas.
 *
 * Se usa WebCrypto y **ninguna biblioteca nueva**: X25519 y Ed25519 están en
 * `crypto.subtle` de los navegadores a los que va la extensión —comprobado en el
 * Chromium de las pruebas, y Firefox se pide 140 o más—. La única maña que hace
 * falta es que WebCrypto no deriva la llave pública de una privada: se importa la
 * privada y se exporta como JWK, que trae la pública en `x`.
 */

const utf8 = new TextEncoder();

/** El mismo conjunto que Go: viaja como campo para poder cambiarlo algún día. */
export const SUITE = "DHKEM(X25519)/HKDF-SHA256/ChaCha20-Poly1305";

const INFO_CIFRADO = "esfinge/identidad/cifrado/v1";
const INFO_FIRMA = "esfinge/identidad/firma/v1";

/** El alfabeto de la clave de recuperación: 32 símbolos sin I, L, O ni U. */
const ALFABETO = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";

export type Identidad = {
  suite: string;
  /** X25519, 32 bytes. */
  cifrado: Uint8Array;
  /** Ed25519, 32 bytes. */
  firma: Uint8Array;
  huella: string;
};

/** Lo que se guarda en el cuerpo de la bóveda, en la sección `identidad`. */
export type IdentidadGuardada = { semilla: string; creada: string; suite?: string };

async function hkdf(semilla: Uint8Array, info: string): Promise<Uint8Array> {
  const k = await crypto.subtle.importKey("raw", new Uint8Array(semilla), "HKDF", false, ["deriveBits"]);
  const bits = await crypto.subtle.deriveBits(
    { name: "HKDF", hash: "SHA-256", salt: new Uint8Array(0), info: utf8.encode(info) },
    k,
    256,
  );
  return new Uint8Array(bits);
}

/**
 * La llave pública de una privada de 32 bytes.
 *
 * WebCrypto no sabe «dame la pública de esta privada», así que se envuelve la
 * privada en el PKCS#8 mínimo —que para estas dos curvas es una cabecera fija más
 * los 32 bytes— y se exporta como JWK, que trae la pública en `x`.
 */
async function publicaDe(privada: Uint8Array, alg: "X25519" | "Ed25519"): Promise<Uint8Array> {
  const oid = alg === "X25519" ? 0x6e : 0x70;
  const pkcs8 = new Uint8Array([
    0x30, 0x2e, 0x02, 0x01, 0x00, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, oid, 0x04, 0x22, 0x04, 0x20,
    ...privada,
  ]);
  const usos: KeyUsage[] = alg === "X25519" ? ["deriveBits"] : ["sign"];
  const k = await crypto.subtle.importKey("pkcs8", pkcs8, { name: alg }, true, usos);
  const jwk = (await crypto.subtle.exportKey("jwk", k)) as { x?: string };
  if (!jwk.x) throw new Error(`WebCrypto no ha dado la llave pública de ${alg}`);
  return desdeBase64url(jwk.x);
}

function desdeBase64url(s: string): Uint8Array {
  const b = atob(s.replace(/-/g, "+").replace(/_/g, "/"));
  return Uint8Array.from(b, (c) => c.charCodeAt(0));
}

/** Cinco bits por símbolo, como la clave de recuperación y como `aPalabras` en Go. */
function aPalabras(b: Uint8Array): string {
  let salida = "";
  let acumulado = 0;
  let bits = 0;
  for (const x of b) {
    acumulado = ((acumulado << 8) | x) >>> 0;
    bits += 8;
    while (bits >= 5) {
      bits -= 5;
      salida += ALFABETO[(acumulado >>> bits) & 31];
    }
  }
  return salida;
}

/**
 * La huella que se compara por teléfono: siete grupos de cuatro símbolos que
 * cubren la suite y las dos llaves. Igual que `HuellaDeIdentidad` en Go.
 */
export async function huellaDeIdentidad(suite: string, cifrado: Uint8Array, firma: Uint8Array): Promise<string> {
  const partes = [utf8.encode(suite), new Uint8Array([0]), cifrado, new Uint8Array([0]), firma];
  const total = partes.reduce((n, p) => n + p.length, 0);
  const todo = new Uint8Array(total);
  let i = 0;
  for (const p of partes) {
    todo.set(p, i);
    i += p.length;
  }
  const h = new Uint8Array(await crypto.subtle.digest("SHA-256", todo));
  const palabras = aPalabras(h).slice(0, 28);
  return (palabras.match(/.{1,4}/g) ?? []).join("-");
}

/** La identidad pública que sale de una semilla de 32 bytes. */
export async function identidadDeSemilla(semilla: Uint8Array, suite = SUITE): Promise<Identidad> {
  if (semilla.length !== 32) throw new Error("La semilla de la identidad tiene que ser de 32 bytes");
  const cifrado = await publicaDe(await hkdf(semilla, INFO_CIFRADO), "X25519");
  const firma = await publicaDe(await hkdf(semilla, INFO_FIRMA), "Ed25519");
  return { suite, cifrado, firma, huella: await huellaDeIdentidad(suite, cifrado, firma) };
}

/**
 * De las dos, la misma que elegiría Go: la más antigua, y si empatan la de semilla
 * menor. Solo hay dos si dos equipos crearon la suya antes de verse.
 */
export function fundirIdentidad(
  l: IdentidadGuardada | undefined,
  r: IdentidadGuardada | undefined,
): IdentidadGuardada | undefined {
  if (!l) return r;
  if (!r) return l;
  if (l.creada !== r.creada) return l.creada < r.creada ? l : r;
  return l.semilla <= r.semilla ? l : r;
}
