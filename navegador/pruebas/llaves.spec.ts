import { expect, test } from "@playwright/test";
import { aDER } from "../src/nucleo/llaves";

/**
 * La conversión de la firma a DER, **con vectores fijos** (ADR 0048).
 *
 * WebCrypto firma en P1363 —`r ‖ s` crudos— y WebAuthn exige ASN.1 DER. La
 * conversión es de treinta líneas y es donde se equivoca todo el mundo, por dos
 * reglas que el formato no perdona:
 *
 *  - un `INTEGER` va **sin ceros por delante**;
 *  - y si el primer bit está a uno, hay que **añadir un cero**, o se lee negativo.
 *
 * **Los vectores son fijos y no una firma de verdad**, y esa es toda la gracia: una
 * firma al azar casi nunca empieza por cero —una vez de cada doscientas cincuenta y
 * seis—, así que la prueba de punta a punta **no ejercita ese camino**. Se comprobó
 * mutándola: quitar el recorte de ceros la dejaba en verde.
 */

const rellenar = (primeros: number[]) => {
  const b = new Uint8Array(32);
  b.set(primeros, 0);
  b[31] = 0x11;
  return b;
};

const juntar = (r: Uint8Array, s: Uint8Array) => {
  const b = new Uint8Array(64);
  b.set(r, 0);
  b.set(s, 32);
  return b;
};

const enHex = (b: Uint8Array) => Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");

test("DER: lo normal, sin recortar ni rellenar", () => {
  const r = rellenar([0x7f]);
  const s = rellenar([0x01]);
  const d = aDER(juntar(r, s));
  // 0x30 len 0x02 32 <r> 0x02 32 <s>
  expect(d[0]).toBe(0x30);
  expect(d[1]).toBe(68);
  expect(d.length).toBe(70);
  expect(enHex(d.slice(2))).toBe("0220" + enHex(r) + "0220" + enHex(s));
});

test("DER: con el primer bit a uno se añade un cero, o se lee negativo", () => {
  const r = rellenar([0x80]);
  const s = rellenar([0xff]);
  const d = aDER(juntar(r, s));
  expect(d[1]).toBe(70);
  expect(d.length).toBe(72);
  expect(enHex(d.slice(2))).toBe("022100" + enHex(r) + "022100" + enHex(s));
});

test("DER: los ceros de delante se recortan", () => {
  const r = rellenar([0x00, 0x2a]);
  const s = rellenar([0x00, 0x00, 0x03]);
  const d = aDER(juntar(r, s));
  expect(enHex(d.slice(2))).toBe("021f" + enHex(r.slice(1)) + "021e" + enHex(s.slice(2)));
});

test("DER: un cero que tapa un bit a uno se recorta y se vuelve a poner", () => {
  // Éste es el caso que se lleva por delante a quien hace solo una de las dos
  // cosas: quitar el cero deja un 0x80 delante, y hay que devolverlo.
  const r = rellenar([0x00, 0x80]);
  const d = aDER(juntar(r, rellenar([0x01])));
  // Recortado queda en 31 bytes, y el cero del relleno lo devuelve a 32: no a 33.
  expect(enHex(d.slice(2, 2 + 2 + 32))).toBe("022000" + enHex(r.slice(1)));
});

test("DER: un valor entero a cero sigue siendo un entero", () => {
  const cero = new Uint8Array(32);
  const d = aDER(juntar(cero, rellenar([0x01])));
  // No se puede recortar hasta la nada: queda un solo byte, el cero.
  expect(enHex(d.slice(2, 5))).toBe("020100");
});

test("DER: con una curva más grande se para en vez de escribir algo malo", () => {
  // El cuerpo de una firma de P-256 mide 70 bytes como mucho, así que la longitud
  // cabe en un byte. Con una curva mayor —P-521 son 66 bytes por número— no cabe,
  // y escribirla igual daría un DER **con la longitud truncada**: algo con forma de
  // firma que ningún sitio acepta y que no dice por qué. Se para y se dice.
  // Relleno de 0xff y no de ceros: unos ceros se recortarían a un solo byte y el
  // cuerpo cabría de sobra, que es como este caso no probaba nada la primera vez.
  expect(() => aDER(new Uint8Array(132).fill(0xff))).toThrow(/forma corta/);
});
