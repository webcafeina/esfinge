import { expect, test } from "@playwright/test";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import {
  abrir,
  abrirTexto,
  azarParaPruebas,
  ErrorESF1,
  PERFIL_LLAVE,
  sellar,
  sellarTexto,
  type Parametros,
} from "../src/nucleo/esf1";

/**
 * El ESF1 de la extensión **contra los vectores fijos de Go** (ADR 0022 y 0040).
 *
 * Se leen de `internal/cripto/testdata/`, en su sitio: copiarlos aquí sería tener
 * dos juegos que pueden separarse. Y se comprueban los dos caminos, igual que en
 * Go: que se abren, y que sellar con la misma sal y el mismo nonce da los mismos
 * bytes. El segundo es el que atrapa un cambio coherente en los dos sentidos.
 */

const CARPETA = fileURLToPath(new URL("../../internal/cripto/testdata/", import.meta.url));

type Vector = {
  fichero: string;
  clave: string;
  modo: string;
  parametros: { Memoria: number; Pasadas: number; Paralelismo: number };
  sal: string;
  nonce: string;
  claro?: string;
  claroBytes?: number;
  sha256: string;
};

const manifiesto = JSON.parse(readFileSync(join(CARPETA, "vectores.json"), "utf8")) as {
  vectores: Vector[];
  rotos: { fichero: string; clave: string; error: string }[];
};
// El flujo por segmentos es para ficheros: la extensión no lo implementa.
const deLaBoveda = manifiesto.vectores.filter((v) => v.modo === "unico" || v.modo === "texto");

const hex = (s: string) => Uint8Array.from(s.match(/../g) ?? [], (h) => parseInt(h, 16));
const parametrosDe = (v: Vector): Parametros => ({
  memoria: v.parametros.Memoria,
  pasadas: v.parametros.Pasadas,
  paralelismo: v.parametros.Paralelismo,
});
const claroDe = (v: Vector) => new TextEncoder().encode(v.claro ?? "");

test("esf1: hay vectores de la bóveda que probar, y no han cambiado en disco", () => {
  expect(deLaBoveda.map((v) => v.fichero)).toEqual(
    expect.arrayContaining(["unico-vacio.esf", "unico-corto.esf", "unico-utf8.esf", "unico-parametros.esf", "unico-perfil.esf", "texto.esf1"]),
  );
  for (const v of deLaBoveda) {
    const h = createHash("sha256").update(readFileSync(join(CARPETA, v.fichero))).digest("hex");
    expect(h, v.fichero).toBe(v.sha256);
  }
});

for (const v of deLaBoveda) {
  test(`esf1: ${v.fichero} se abre`, async () => {
    test.setTimeout(60_000);
    const bruto = readFileSync(join(CARPETA, v.fichero));
    const claro =
      v.modo === "texto" ? await abrirTexto(bruto.toString("utf8"), v.clave) : await abrir(new Uint8Array(bruto), v.clave);
    expect(Buffer.from(claro).equals(Buffer.from(claroDe(v)))).toBe(true);
  });

  test(`esf1: ${v.fichero} se sella igual, byte a byte`, async () => {
    test.setTimeout(60_000);
    const sal = hex(v.sal);
    const nonce = hex(v.nonce);
    let n = 0;
    azarParaPruebas(() => (++n === 1 ? sal.slice() : nonce.slice()));
    try {
      const hecho =
        v.modo === "texto"
          ? new TextEncoder().encode(await sellarTexto(claroDe(v), v.clave, parametrosDe(v)))
          : await sellar(claroDe(v), v.clave, parametrosDe(v));
      expect(Buffer.from(hecho).equals(readFileSync(join(CARPETA, v.fichero)))).toBe(true);
    } finally {
      azarParaPruebas(null);
    }
  });
}

test("esf1: los rotos fallan por lo que son, no solo fallan", async () => {
  const tipos: Record<string, string> = { ErrFormato: "formato", ErrClaveIncorrecta: "clave", ErrVersion: "version" };
  for (const r of manifiesto.rotos) {
    const bruto = new Uint8Array(readFileSync(join(CARPETA, r.fichero)));
    const err = await abrir(bruto, r.clave).catch((e: unknown) => e);
    // El truncado de Go es un flujo cortado: aquí, como no es modo único entero,
    // lo que importa es que no se abra y que no se tome por una clave mala.
    expect(err, r.fichero).toBeInstanceOf(ErrorESF1);
    if (r.fichero !== "roto-truncado.esf") expect((err as ErrorESF1).tipo, r.fichero).toBe(tipos[r.error]);
  }
});

test("esf1: ida y vuelta, y con otra clave no abre", async () => {
  const texto = await sellarTexto(new TextEncoder().encode("hola, ñandú"), "una clave", PERFIL_LLAVE);
  expect(texto.startsWith("ESF1.")).toBe(true);
  expect(new TextDecoder().decode(await abrirTexto(`  ${texto}\n`, "una clave"))).toBe("hola, ñandú");
  const err = await abrirTexto(texto, "otra clave").catch((e: unknown) => e);
  expect((err as ErrorESF1).tipo).toBe("clave");
});

test("esf1: una cabecera que pide memoria de más no se intenta derivar", async () => {
  const bueno = await sellar(new Uint8Array([1, 2, 3]), "k", PERFIL_LLAVE);
  const malo = bueno.slice();
  new DataView(malo.buffer).setUint32(6, 64 * 1024 * 1024, false); // 64 GiB
  const err = await abrir(malo, "k").catch((e: unknown) => e);
  expect((err as ErrorESF1).tipo).toBe("formato");
});
