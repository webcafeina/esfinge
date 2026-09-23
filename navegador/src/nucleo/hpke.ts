/**
 * HPKE en modo base, lo justo para los envíos (RFC 9180, ADR 0043).
 *
 * **Solo un conjunto**: `DHKEM(X25519, HKDF-SHA256)` con `ChaCha20-Poly1305`, que
 * es el que usa `crypto/hpke` de Go al otro lado. No hay biblioteca nueva: X25519
 * y HKDF salen de WebCrypto y el cifrado de `@noble/ciphers`, que ya estaba.
 *
 * Se implementa a mano porque hace falta **poco y muy concreto** —cifrar una vez
 * hacia una llave y descifrar una vez— y porque así no entra otra dependencia en
 * la revisión de Mozilla. Lo que lo sostiene no es la lectura de este fichero: son
 * las pruebas cruzadas, donde Go cierra sobres que esto abre y al revés.
 *
 * Las etiquetas y los números del RFC no se tocan: si algo aquí se «arregla» sin
 * cambiarlo en Go, el sobre deja de abrirse.
 */

import { chacha20poly1305 } from "@noble/ciphers/chacha.js";

const utf8 = new TextEncoder();

/** Los identificadores del conjunto, tal como los numera el RFC 9180. */
const KEM_ID = 0x0020; // DHKEM(X25519, HKDF-SHA256)
const KDF_ID = 0x0001; // HKDF-SHA256
const AEAD_ID = 0x0003; // ChaCha20-Poly1305
const N_SECRET = 32;
const N_K = 32;
const N_N = 12;

function juntar(...partes: Uint8Array[]): Uint8Array {
  const total = partes.reduce((n, p) => n + p.length, 0);
  const out = new Uint8Array(total);
  let i = 0;
  for (const p of partes) {
    out.set(p, i);
    i += p.length;
  }
  return out;
}

/** Un entero de dos bytes, como `I2OSP(x, 2)` del RFC. */
function dosBytes(n: number): Uint8Array {
  return new Uint8Array([(n >> 8) & 0xff, n & 0xff]);
}

const SUITE_KEM = juntar(utf8.encode("KEM"), dosBytes(KEM_ID));
const SUITE_HPKE = juntar(utf8.encode("HPKE"), dosBytes(KEM_ID), dosBytes(KDF_ID), dosBytes(AEAD_ID));

/**
 * HKDF por partes.
 *
 * WebCrypto solo ofrece extraer y expandir de una vez, y el RFC los usa por
 * separado, así que se montan sobre HMAC-SHA256, que sí está.
 */
async function hmac(clave: Uint8Array, datos: Uint8Array): Promise<Uint8Array> {
  const k = await crypto.subtle.importKey("raw", new Uint8Array(clave), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  return new Uint8Array(await crypto.subtle.sign("HMAC", k, new Uint8Array(datos)));
}

/**
 * Extraer, con el detalle del RFC 5869 que WebCrypto obliga a mirar: **sin sal, la
 * sal son 32 bytes de ceros**, no una clave vacía. `crypto.subtle` rechaza una
 * clave HMAC de longitud cero con «Zero-length key is not supported», así que
 * escribirlo mal no da un resultado distinto: da un error que no dice de qué va.
 */
const extraer = (sal: Uint8Array, ikm: Uint8Array) => hmac(sal.length > 0 ? sal : new Uint8Array(32), ikm);

async function expandir(prk: Uint8Array, info: Uint8Array, largo: number): Promise<Uint8Array> {
  const out = new Uint8Array(largo);
  let t: Uint8Array = new Uint8Array(0);
  let puesto = 0;
  for (let i = 1; puesto < largo; i++) {
    t = await hmac(prk, juntar(t, info, new Uint8Array([i])));
    const cabe = Math.min(t.length, largo - puesto);
    out.set(t.subarray(0, cabe), puesto);
    puesto += cabe;
  }
  return out;
}

const etiquetado = (suite: Uint8Array, etiqueta: string, x: Uint8Array) =>
  juntar(utf8.encode("HPKE-v1"), suite, utf8.encode(etiqueta), x);

const extraerConEtiqueta = (suite: Uint8Array, sal: Uint8Array, etiqueta: string, ikm: Uint8Array) =>
  extraer(sal, etiquetado(suite, etiqueta, ikm));

const expandirConEtiqueta = (suite: Uint8Array, prk: Uint8Array, etiqueta: string, info: Uint8Array, largo: number) =>
  expandir(prk, juntar(dosBytes(largo), etiquetado(suite, etiqueta, info)), largo);

const vacio = new Uint8Array(0);

/** El secreto compartido del KEM, a partir del Diffie-Hellman y el contexto. */
async function secretoDelKem(dh: Uint8Array, contexto: Uint8Array): Promise<Uint8Array> {
  const prk = await extraerConEtiqueta(SUITE_KEM, vacio, "eae_prk", dh);
  return expandirConEtiqueta(SUITE_KEM, prk, "shared_secret", contexto, N_SECRET);
}

/** La clave y el nonce de la sesión (modo base: sin PSK). */
async function claveYNonce(secreto: Uint8Array, info: Uint8Array): Promise<{ clave: Uint8Array; nonce: Uint8Array }> {
  const pskIdHash = await extraerConEtiqueta(SUITE_HPKE, vacio, "psk_id_hash", vacio);
  const infoHash = await extraerConEtiqueta(SUITE_HPKE, vacio, "info_hash", info);
  const contexto = juntar(new Uint8Array([0x00]), pskIdHash, infoHash); // 0x00 = modo base
  const secretoBase = await extraerConEtiqueta(SUITE_HPKE, secreto, "secret", vacio);
  return {
    clave: await expandirConEtiqueta(SUITE_HPKE, secretoBase, "key", contexto, N_K),
    nonce: await expandirConEtiqueta(SUITE_HPKE, secretoBase, "base_nonce", contexto, N_N),
  };
}

async function dhConPrivadaCruda(privada: Uint8Array, publica: Uint8Array): Promise<Uint8Array> {
  const pkcs8 = new Uint8Array([
    0x30, 0x2e, 0x02, 0x01, 0x00, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x6e, 0x04, 0x22, 0x04, 0x20,
    ...privada,
  ]);
  const mia = await crypto.subtle.importKey("pkcs8", pkcs8, { name: "X25519" }, false, ["deriveBits"]);
  const suya = await crypto.subtle.importKey("raw", new Uint8Array(publica), { name: "X25519" }, false, []);
  return new Uint8Array(await crypto.subtle.deriveBits({ name: "X25519", public: suya }, mia, 256));
}

/**
 * Prepara un envío hacia `publica` y devuelve **el encapsulado y una función para
 * cerrar**, en dos pasos y no en uno.
 *
 * Esto no es un capricho de estilo: el `enc` viaja dentro de la cabecera del
 * sobre, y la cabecera va como datos autenticados del cifrado. Hace falta saber
 * el `enc` **antes** de cifrar. Go lo resuelve igual —`NewSender` devuelve el
 * `enc` y luego se llama a `Seal`—, y hacerlo de otra forma obliga a cifrar dos
 * veces o a dejar el `enc` fuera de lo autenticado.
 */
export async function nuevoEmisor(
  publica: Uint8Array,
  info: Uint8Array,
): Promise<{ enc: Uint8Array; sellar: (aad: Uint8Array, claro: Uint8Array) => Uint8Array }> {
  const efimera = (await crypto.subtle.generateKey({ name: "X25519" }, true, ["deriveBits"])) as CryptoKeyPair;
  const enc = new Uint8Array(await crypto.subtle.exportKey("raw", efimera.publicKey));
  const suya = await crypto.subtle.importKey("raw", new Uint8Array(publica), { name: "X25519" }, false, []);
  const dh = new Uint8Array(await crypto.subtle.deriveBits({ name: "X25519", public: suya }, efimera.privateKey, 256));
  const secreto = await secretoDelKem(dh, juntar(enc, publica));
  const { clave, nonce } = await claveYNonce(secreto, info);
  return { enc, sellar: (aad, claro) => chacha20poly1305(clave, nonce, aad).encrypt(claro) };
}

/** Lo contrario: abre con la privada de 32 bytes derivada de la semilla. */
export async function abrirHpke(
  privada: Uint8Array,
  publicaPropia: Uint8Array,
  enc: Uint8Array,
  info: Uint8Array,
  aad: Uint8Array,
  cifrado: Uint8Array,
): Promise<Uint8Array> {
  const dh = await dhConPrivadaCruda(privada, enc);
  const secreto = await secretoDelKem(dh, juntar(enc, publicaPropia));
  const { clave, nonce } = await claveYNonce(secreto, info);
  return chacha20poly1305(clave, nonce, aad).decrypt(cifrado);
}
