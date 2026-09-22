/**
 * El contenedor ESF1 en TypeScript: Argon2id + XChaCha20-Poly1305 (ADR 0040).
 *
 * Es **la segunda implementación** del formato de `internal/cripto`, y solo la
 * parte que usa la bóveda: el modo único y su forma de texto. El flujo por
 * segmentos es para ficheros, y eso la extensión no lo hace.
 *
 * El formato no se describe aquí sino en Go, y está congelado con vectores fijos
 * (ADR 0022). Las pruebas de este fichero **leen esos mismos vectores de su sitio**
 * —`internal/cripto/testdata/`— y exigen dos cosas: que se abren, y que sellar con
 * la misma sal y el mismo nonce da **los mismos bytes**. Si aquí algo se pone
 * rojo, el que está mal es este fichero.
 *
 *     "ESF1" | versión(1) | modo(1) | memoria(4 BE) | pasadas(4 BE) | paralelismo(1) | sal(16) | nonce(24)
 *
 * La cabecera entera va como datos asociados, y **la clave son sus bytes UTF-8
 * tal cual, sin normalizar** (lo fija `unico-utf8.esf`).
 */

import { xchacha20poly1305 } from "@noble/ciphers/chacha.js";
import { argon2id } from "hash-wasm";

export type Parametros = {
  /** En KiB. */
  memoria: number;
  pasadas: number;
  paralelismo: number;
};

/** El coste de las contraseñas de persona: 64 MiB, 3 pasadas, 4 hilos. */
export const PERFIL_INTERACTIVO: Parametros = { memoria: 64 * 1024, pasadas: 3, paralelismo: 4 };

/**
 * El coste mínimo, para sellar con la clave de bóveda, que ya son 256 bits al
 * azar: estirarla no añade nada y costaría en cada guardado. No es un descuido;
 * ver `PerfilLlave` en `internal/cripto/clave.go`.
 */
export const PERFIL_LLAVE: Parametros = { memoria: 8 * 1024, pasadas: 1, paralelismo: 1 };

const MAGIA = "ESF1";
const VERSION = 1;
const MODO_UNICO = 0;
const MODO_FLUJO = 1;
const TAM_SAL = 16;
const TAM_NONCE = 24;
const TAM_CABECERA = 4 + 1 + 1 + 4 + 4 + 1 + TAM_SAL + TAM_NONCE; // 55
export const PREFIJO = MAGIA + ".";

// Los mismos límites de cordura que Go al abrir algo ajeno: sin ellos, una
// cabecera manipulada pidiendo gigas de memoria tumba el navegador antes de que
// se pueda comprobar la etiqueta.
const MEMORIA_MAXIMA = 1024 * 1024;
const PASADAS_MAXIMO = 16;

/**
 * Los errores, con las mismas frases que Go: con cuenta, el mismo fallo puede
 * salir por la ventana o por el panel, y tiene que decir lo mismo.
 */
export class ErrorESF1 extends Error {
  constructor(
    readonly tipo: "clave" | "formato" | "version",
    mensaje: string,
  ) {
    super(mensaje);
  }
}
export const errClaveIncorrecta = () =>
  new ErrorESF1("clave", "La clave no es correcta, o el contenido llegó alterado o cortado");
export const errFormato = () => new ErrorESF1("formato", "Esto no parece un contenedor de Esfinge");
export const errVersion = () => new ErrorESF1("version", "Contenedor de una versión de Esfinge más nueva que esta");

/**
 * `azar` se puede sustituir, y es la misma costura que en Go (`azar` en
 * `clave.go`): sin ella no se puede reproducir un vector byte a byte. Solo la
 * tocan las pruebas, y siempre la devuelven a su sitio.
 */
let azar = (n: number): Uint8Array => crypto.getRandomValues(new Uint8Array(n));

export function azarParaPruebas(f: ((n: number) => Uint8Array) | null): void {
  azar = f ?? ((n: number) => crypto.getRandomValues(new Uint8Array(n)));
}

/** Bytes al azar del sistema. */
export function azarDe(n: number): Uint8Array {
  return azar(n);
}

const utf8 = new TextEncoder();

export function bytesDe(clave: string | Uint8Array): Uint8Array {
  return typeof clave === "string" ? utf8.encode(clave) : clave;
}

function validar(p: Parametros): void {
  if (
    p.memoria < 8 * 1024 ||
    p.memoria > MEMORIA_MAXIMA ||
    p.pasadas < 1 ||
    p.pasadas > PASADAS_MAXIMO ||
    p.paralelismo < 1 ||
    p.paralelismo > 255
  ) {
    throw errFormato();
  }
}

/** La clave de 32 bytes que sale de una clave humana, igual que `derivar` en Go. */
export async function derivar(clave: Uint8Array, sal: Uint8Array, p: Parametros): Promise<Uint8Array> {
  return argon2id({
    password: clave,
    salt: sal,
    parallelism: p.paralelismo,
    iterations: p.pasadas,
    memorySize: p.memoria,
    hashLength: 32,
    outputType: "binary",
  });
}

function cabecera(p: Parametros, sal: Uint8Array, nonce: Uint8Array): Uint8Array {
  const b = new Uint8Array(TAM_CABECERA);
  const v = new DataView(b.buffer);
  b.set(utf8.encode(MAGIA), 0);
  b[4] = VERSION;
  b[5] = MODO_UNICO;
  v.setUint32(6, p.memoria, false);
  v.setUint32(10, p.pasadas, false);
  b[14] = p.paralelismo;
  b.set(sal, 15);
  b.set(nonce, 15 + TAM_SAL);
  return b;
}

/** Cifra y devuelve el contenedor binario completo. */
export async function sellar(datos: Uint8Array, clave: string | Uint8Array, p: Parametros): Promise<Uint8Array> {
  validar(p);
  // El orden importa, y es el de Go: primero la sal y después el nonce.
  const sal = azar(TAM_SAL);
  const nonce = azar(TAM_NONCE);
  const cab = cabecera(p, sal, nonce);
  const k = await derivar(bytesDe(clave), sal, p);
  try {
    const cifrado = xchacha20poly1305(k, nonce, cab).encrypt(datos);
    const out = new Uint8Array(cab.length + cifrado.length);
    out.set(cab, 0);
    out.set(cifrado, cab.length);
    return out;
  } finally {
    k.fill(0);
  }
}

/** Descifra un contenedor binario de modo único. */
export async function abrir(contenedor: Uint8Array, clave: string | Uint8Array): Promise<Uint8Array> {
  if (contenedor.length < TAM_CABECERA) throw errFormato();
  if (new TextDecoder().decode(contenedor.subarray(0, 4)) !== MAGIA) throw errFormato();
  if (contenedor[4] !== VERSION) throw errVersion();
  const modo = contenedor[5];
  if (modo === MODO_FLUJO) {
    throw new ErrorESF1("formato", "Este contenedor es un flujo por segmentos: usa el modo fichero");
  }
  if (modo !== MODO_UNICO) throw errFormato();
  const v = new DataView(contenedor.buffer, contenedor.byteOffset, contenedor.byteLength);
  const p: Parametros = { memoria: v.getUint32(6, false), pasadas: v.getUint32(10, false), paralelismo: contenedor[14] };
  validar(p);
  const sal = contenedor.slice(15, 15 + TAM_SAL);
  const nonce = contenedor.slice(15 + TAM_SAL, TAM_CABECERA);
  const cab = contenedor.slice(0, TAM_CABECERA);
  const k = await derivar(bytesDe(clave), sal, p);
  try {
    return xchacha20poly1305(k, nonce, cab).decrypt(contenedor.slice(TAM_CABECERA));
  } catch {
    throw errClaveIncorrecta();
  } finally {
    k.fill(0);
  }
}

/** Cifra y devuelve la forma de una línea, `ESF1.<base64url>`. */
export async function sellarTexto(datos: Uint8Array, clave: string | Uint8Array, p: Parametros): Promise<string> {
  return PREFIJO + base64url(await sellar(datos, clave, p));
}

/**
 * Descifra la forma de una línea. Como en Go, tolera espacios y saltos, la falta
 * del prefijo y el base64 estándar: el texto viene de copiar y pegar.
 */
export async function abrirTexto(texto: string, clave: string | Uint8Array): Promise<Uint8Array> {
  let limpio = texto.replace(/[ \n\r\t]/g, "");
  if (limpio.startsWith(PREFIJO)) limpio = limpio.slice(PREFIJO.length);
  let bruto: Uint8Array;
  try {
    bruto = desdeBase64(limpio);
  } catch {
    throw errFormato();
  }
  return abrir(bruto, clave);
}

// ------------------------------------------------------------------ base64

/** base64url sin relleno, que es `base64.RawURLEncoding` de Go. */
export function base64url(b: Uint8Array): string {
  let s = "";
  for (const x of b) s += String.fromCharCode(x);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/**
 * Lee base64url sin relleno o, si no lo es, base64 estándar con relleno, como
 * `AbrirTexto` en Go. Estricto con el largo: un resto de un carácter no es base64.
 */
export function desdeBase64(s: string): Uint8Array {
  let t: string;
  if (/^[A-Za-z0-9_-]*$/.test(s)) {
    if (s.length % 4 === 1) throw new Error("base64");
    t = s.replace(/-/g, "+").replace(/_/g, "/");
    t += "=".repeat((4 - (t.length % 4)) % 4);
  } else if (/^[A-Za-z0-9+/]*={0,2}$/.test(s) && s.length % 4 === 0) {
    t = s;
  } else {
    throw new Error("base64");
  }
  const bin = atob(t);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}
