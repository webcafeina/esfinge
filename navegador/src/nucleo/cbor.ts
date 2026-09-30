/**
 * El CBOR que hace falta para crear una llave de acceso, **y solo ése** (ADR 0048).
 *
 * Espejo de `internal/navegador/cbor.go`, y lo es por la razón de siempre: con
 * cuenta la extensión crea la llave sola y sin ella se lo pide a Go, y **el sitio
 * guarda estos bytes para siempre**. Si los dos lados no escriben lo mismo, la misma
 * llave sería distinta según dónde se creó. Lo vigila una prueba cruzada.
 *
 * **Aquí solo hay codificador.** No se descodifica CBOR en ninguna parte, y no
 * conviene: un descodificador es superficie de ataque sobre bytes de fuera, y nada
 * de lo que hacemos lo necesita.
 */

const ENTERO = 0 << 5;
const NEGATIVO = 1 << 5;
const BYTES = 2 << 5;
const TEXTO = 3 << 5;
const MAPA = 5 << 5;

/**
 * El tipo mayor y el valor, **con el tamaño más corto que quepa**.
 *
 * No es una optimización: es la forma canónica que exige CTAP2, y un sitio que
 * compruebe la atestación puede volver a codificar lo que reciba y comparar.
 */
function cabecera(mayor: number, valor: number): Uint8Array<ArrayBuffer> {
  if (valor < 24) return new Uint8Array([mayor | valor]);
  if (valor <= 0xff) return new Uint8Array([mayor | 24, valor]);
  if (valor <= 0xffff) return new Uint8Array([mayor | 25, valor >>> 8, valor & 0xff]);
  if (valor <= 0xffffffff) {
    return new Uint8Array([mayor | 26, (valor >>> 24) & 0xff, (valor >>> 16) & 0xff, (valor >>> 8) & 0xff, valor & 0xff]);
  }
  // Más de 2^32 no ocurre aquí —lo más largo es una clave pública—, y fingir que se
  // soporta sería escribir bytes que nadie ha probado.
  throw new Error("Ese valor no cabe en el CBOR que Esfinge escribe");
}

function pegar(...trozos: Uint8Array[]): Uint8Array<ArrayBuffer> {
  const total = trozos.reduce((n, t) => n + t.length, 0);
  const out = new Uint8Array(total);
  let i = 0;
  for (const t of trozos) {
    out.set(t, i);
    i += t.length;
  }
  return out;
}

export function cborDeEntero(n: number): Uint8Array<ArrayBuffer> {
  return n < 0 ? cabecera(NEGATIVO, -1 - n) : cabecera(ENTERO, n);
}

export function cborDeBytes(b: Uint8Array): Uint8Array<ArrayBuffer> {
  return pegar(cabecera(BYTES, b.length), b);
}

export function cborDeTexto(s: string): Uint8Array<ArrayBuffer> {
  const b = new TextEncoder().encode(s);
  return pegar(cabecera(TEXTO, b.length), b);
}

/**
 * Un mapa **en el orden canónico de CTAP2**: las claves ya codificadas, ordenadas
 * primero por largo y luego por bytes.
 *
 * El orden importa y no es el que uno escribiría: con las claves del COSE —1, 3,
 * -1, -2, -3— sale `1, 3, -1, -2, -3`, porque los negativos se codifican como
 * `0x20`, `0x21` y `0x22`, que son mayores que `0x01` y `0x03`.
 *
 * **Se comparan bytes, no cadenas.** Ordenar por el texto de un `TextDecoder` daría
 * el orden de UTF-16 y no el de los bytes, que es la misma trampa que la forma
 * canónica de la bóveda.
 */
export function cborDeMapa(pares: [Uint8Array, Uint8Array][]): Uint8Array<ArrayBuffer> {
  const orden = [...pares].sort(([a], [b]) => {
    if (a.length !== b.length) return a.length - b.length;
    for (let i = 0; i < a.length; i++) {
      if (a[i] !== b[i]) return a[i] - b[i];
    }
    return 0;
  });
  return pegar(cabecera(MAPA, orden.length), ...orden.flat());
}

/** Una coordenada en sus 32 bytes exactos. */
function deTreintaYDos(b: Uint8Array): Uint8Array<ArrayBuffer> {
  if (b.length >= 32) return b.slice(b.length - 32) as Uint8Array<ArrayBuffer>;
  const out = new Uint8Array(32);
  out.set(b, 32 - b.length);
  return out;
}

/**
 * Una pública P-256 en el formato que WebAuthn guarda.
 *
 * `1: 2` clave de curva elíptica, `3: -7` ECDSA con SHA-256, `-1: 1` la curva
 * P-256, y `-2`/`-3` las coordenadas **de 32 bytes y con los ceros de delante
 * puestos**.
 *
 * Eso último es lo contrario de lo que pide el DER de una firma, donde los ceros de
 * delante se **quitan**. Aquí son de largo fijo: una `x` que empiece por cero tiene
 * que llevarlo, o el sitio guardará otra clave y la llave no volverá a servir. Pasa
 * una vez de cada 256, así que no se ve probando a mano.
 */
export function claveCOSE(x: Uint8Array, y: Uint8Array): Uint8Array<ArrayBuffer> {
  return cborDeMapa([
    [cborDeEntero(1), cborDeEntero(2)],
    [cborDeEntero(3), cborDeEntero(-7)],
    [cborDeEntero(-1), cborDeEntero(1)],
    [cborDeEntero(-2), cborDeBytes(deTreintaYDos(x))],
    [cborDeEntero(-3), cborDeBytes(deTreintaYDos(y))],
  ]);
}

/**
 * Los datos del autenticador envueltos **sin atestación**.
 *
 * `fmt: "none"` y `attStmt` vacío: Esfinge no es un autenticador certificado y no va
 * a decir que lo es. Un sitio que exija atestación de verdad rechazará la llave, y
 * hace bien.
 */
export function objetoDeAtestacion(datosDelAutenticador: Uint8Array): Uint8Array<ArrayBuffer> {
  return cborDeMapa([
    [cborDeTexto("fmt"), cborDeTexto("none")],
    [cborDeTexto("attStmt"), cborDeMapa([])],
    [cborDeTexto("authData"), cborDeBytes(datosDelAutenticador)],
  ]);
}
