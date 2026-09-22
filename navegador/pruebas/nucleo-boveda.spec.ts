import { expect, test } from "@playwright/test";
import { Boveda, ErrorBoveda, relojParaPruebas } from "../src/nucleo/boveda";
import { codigoConContador, codigoEn, leerSemilla } from "../src/nucleo/codigos";
import { normalizar } from "../src/nucleo/recuperacion";

/**
 * El núcleo de la extensión por su cuenta (ADR 0040). Lo que tiene que coincidir
 * con Go lo vigilan las pruebas cruzadas de Go (`cruzada_test.go` en cada paquete); aquí va
 * lo que se puede decir sin Go: los vectores de los RFC y los caminos de error.
 */

const MAESTRA = "una maestra larga para las pruebas del núcleo";

test("códigos: los vectores de RFC 4226", async () => {
  const s = leerSemilla("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ");
  const esperados = ["755224", "287082", "359152", "969429", "338314", "254676", "287922", "162583", "399871", "520489"];
  for (let i = 0; i < esperados.length; i++) {
    const c = new Uint8Array(8);
    new DataView(c.buffer).setBigUint64(0, BigInt(i), false);
    expect(await codigoConContador(s, c)).toBe(esperados[i]);
  }
});

test("códigos: los vectores de RFC 6238, cada algoritmo con su semilla", async () => {
  const semillas = {
    SHA1: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ",
    SHA256: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZA",
    SHA512: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNA",
  };
  const tabla: [number, string, string, string][] = [
    [59, "94287082", "46119246", "90693936"],
    [1111111109, "07081804", "68084774", "25091201"],
    [1234567890, "89005924", "91819424", "93441116"],
    [20000000000, "65353130", "77737706", "47863826"],
  ];
  for (const [t, a, b, c] of tabla) {
    const porAlgoritmo: ["SHA1" | "SHA256" | "SHA512", string][] = [["SHA1", a], ["SHA256", b], ["SHA512", c]];
    for (const [alg, quiero] of porAlgoritmo) {
      const s = leerSemilla(`otpauth://totp/x?secret=${semillas[alg]}&digits=8&algorithm=${alg}`);
      expect(await codigoEn(s, new Date(t * 1000)), `${alg} en ${t}`).toBe(quiero);
    }
  }
});

test("recuperación: se acepta como la teclea la gente, y una errata no pasa", async () => {
  const { boveda, recuperacion } = await Boveda.crear(MAESTRA);
  const texto = await boveda.guardar();
  expect(recuperacion).toMatch(/^ESF(-[0-9A-HJKMNP-TV-Z]{4})+$/);
  const tecleada = recuperacion.toLowerCase().replace(/-/g, " ").replace(/0/g, "o");
  expect(await normalizar(tecleada)).toBe(await normalizar(recuperacion));
  expect((await Boveda.abrir(texto, tecleada)).id).toBe(boveda.id);

  const ultimo = recuperacion.at(-1)!;
  const errata = recuperacion.slice(0, -1) + (ultimo === "Z" ? "Y" : "Z");
  const err = await Boveda.abrir(texto, errata).catch((e: unknown) => e);
  expect((err as ErrorBoveda).codigo).toBe("checksum");
  expect(((await Boveda.abrir(texto, "otra cosa").catch((e: unknown) => e)) as ErrorBoveda).codigo).toBe("sin-ranura");
});

test("bóveda: quitar una ranura o volver a un cuerpo viejo se detecta", async () => {
  const { boveda } = await Boveda.crear(MAESTRA);
  const viejo = JSON.parse(await boveda.poner({ id: "", tipo: "credencial", titulo: "A", creada: "", cambiada: "" }).then(() => boveda.documento()));
  const nuevo = JSON.parse(await boveda.poner({ id: "", tipo: "credencial", titulo: "B", creada: "", cambiada: "" }).then(() => boveda.documento()));

  const sinRecuperacion = { ...nuevo, sobres: nuevo.sobres.filter((s: { tipo: string }) => s.tipo !== "recuperacion") };
  const cuerpoViejo = { ...nuevo, cuerpo: viejo.cuerpo };
  for (const malo of [sinRecuperacion, cuerpoViejo]) {
    const err = await Boveda.abrir(JSON.stringify(malo), MAESTRA).catch((e: unknown) => e);
    expect((err as ErrorBoveda).codigo).toBe("manipulada");
  }
});

test("bóveda: la revisión la pone ella, y la papelera se vacía sola a los treinta días", async () => {
  const { boveda } = await Boveda.crear(MAESTRA);
  const e = await boveda.poner({ id: "", tipo: "credencial", titulo: "Banco", creada: "", cambiada: "", revision: 99 });
  expect(e.revision).toBe(1);
  expect((await boveda.poner({ ...e, titulo: "Banco 2" })).revision).toBe(2);
  await boveda.borrar(e.id);
  expect(boveda.papelera()).toHaveLength(1);

  relojParaPruebas(() => new Date(Date.now() + 31 * 24 * 3600 * 1000));
  try {
    const abierta = await Boveda.abrir(boveda.documento(), MAESTRA, { purgar: true });
    expect(abierta.papelera()).toHaveLength(0);
    expect(abierta.cuantas()).toBe(0);
  } finally {
    relojParaPruebas(null);
  }
  // Sin purgar —lo que baja del servidor—, no se toca nada.
  expect((await Boveda.abrir(boveda.documento(), MAESTRA)).papelera()).toHaveLength(1);
});

test("bóveda: la lista sale sin secretos, y cerrada no deja hacer nada", async () => {
  const { boveda } = await Boveda.crear(MAESTRA);
  await boveda.poner({ id: "", tipo: "credencial", titulo: "Banco", usuario: "yo", secreto: "s3cr3t0", totp: "GEZDGNBV", notas: "n", creada: "", cambiada: "" });
  const [e] = boveda.buscar("banco");
  expect(e.secreto).toBeUndefined();
  expect(e.totp).toBeUndefined();
  expect(e.notas).toBeUndefined();
  expect(boveda.ver(e.id)!.secreto).toBe("s3cr3t0");
  expect(boveda.buscar("s3cr3t0")).toHaveLength(0); // nunca se busca por el secreto
  boveda.cerrar();
  expect(() => boveda.buscar("")).toThrow("La bóveda está cerrada");
});
