/**
 * Las claves de la cuenta en la extensión, igual que `internal/cuenta/claves.go`
 * (ADR 0037): la clave de acceso se deriva aparte de la ranura, con la sal de la
 * cuenta, y el coste que diga el servidor **nunca por debajo del de siempre**.
 */

import { derivar, PERFIL_INTERACTIVO, type Parametros } from "./esf1";

/** Lo que devuelve el servidor en la pre-entrada, con los nombres de Go. */
export type ParametrosDeCuenta = { memoria: number; pasadas: number; paralelismo: number };

export const ERR_COSTE_BAJO = "El servidor pide proteger la contraseña con menos coste del normal; no se sigue";

export function validarCoste(p: ParametrosDeCuenta): void {
  const d = PERFIL_INTERACTIVO;
  if (p.memoria < d.memoria || p.pasadas < d.pasadas || p.paralelismo < 1) throw new Error(ERR_COSTE_BAJO);
  // Los mismos topes que al abrir un contenedor —256 MiB—: un servidor que
  // pidiera gigas tumbaría el trabajador de fondo antes de que nadie pudiera
  // decir nada.
  if (p.memoria > 256 * 1024 || p.pasadas > 16 || p.paralelismo > 16) {
    throw new Error("El servidor pide un coste de derivación imposible");
  }
}

/** HKDF-SHA256 sin sal, como `hkdf.Key(sha256.New, raiz, nil, info, 32)` en Go. */
async function hkdf(raiz: Uint8Array, info: string): Promise<Uint8Array> {
  const k = await crypto.subtle.importKey("raw", new Uint8Array(raiz), "HKDF", false, ["deriveBits"]);
  const bits = await crypto.subtle.deriveBits(
    { name: "HKDF", hash: "SHA-256", salt: new Uint8Array(0), info: new TextEncoder().encode(info) },
    k,
    256,
  );
  return new Uint8Array(bits);
}

/** La clave de acceso: HKDF(Argon2id(maestra, sal de la cuenta), «esfinge/cuenta/acceso/v1»). */
export async function derivarAcceso(maestra: string, sal: Uint8Array, p: ParametrosDeCuenta): Promise<Uint8Array> {
  validarCoste(p);
  if (sal.length !== 16) throw new Error("La sal de la cuenta no mide lo que debe");
  if (!maestra.trim()) throw new Error("La contraseña maestra no puede estar vacía");
  const par: Parametros = { memoria: p.memoria, pasadas: p.pasadas, paralelismo: p.paralelismo };
  const raiz = await derivar(new TextEncoder().encode(maestra), sal, par);
  try {
    return await hkdf(raiz, "esfinge/cuenta/acceso/v1");
  } finally {
    raiz.fill(0);
  }
}

const PARECE_CORREO = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/**
 * Como la guarda el servidor: sin espacios alrededor, en NFC y en minúsculas. Nada
 * de quitar puntos ni lo que va tras un «+». Tiene que coincidir con
 * `normalizarCorreo` del servidor, que también es TypeScript.
 */
export function normalizarCorreo(c: string): string {
  const n = c.trim().normalize("NFC").toLowerCase();
  if (n.length < 3 || n.length > 254 || !PARECE_CORREO.test(n)) throw new Error("Ese correo no parece válido");
  return n;
}
