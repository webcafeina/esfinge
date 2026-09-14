import { expect, test } from "@playwright/test";
import { queMostrar } from "../src/insignia";

/**
 * Qué enseña el icono de la barra en cada situación.
 *
 * Es una función pura a propósito: lo que la envuelve —el trabajador de fondo
 * pintando con `action.*`— necesita la extensión cargada, que sigue siendo deuda.
 * Todo lo que decide está aquí y se prueba entero, sin navegador.
 */

const URL = "https://login.brevo.com/entrar";
const abierta = { ok: true, estado: { existe: true, abierta: true } };
const cuentas = (n: number) => ({
  ok: true,
  cuentas: Array.from({ length: n }, (_, i) => ({ id: String(i), titulo: "Brevo", usuario: "yo" })),
});

test("insignia: donde no se rellena, apagado y sin insignia", () => {
  for (const url of ["http://router.local/", "chrome://newtab/", "about:blank", undefined]) {
    expect(queMostrar({ url, estado: abierta })).toMatchObject({ icono: "apagado", insignia: "" });
  }
});

test("insignia: el número de cuentas, con singular y con 9+", () => {
  expect(queMostrar({ url: URL, estado: abierta, cuentas: cuentas(0) })).toMatchObject({
    icono: "activo",
    insignia: "",
    titulo: "Esfinge · Nada guardado de login.brevo.com",
  });
  expect(queMostrar({ url: URL, estado: abierta, cuentas: cuentas(1) })).toMatchObject({
    insignia: "1",
    titulo: "Esfinge · 1 cuenta de login.brevo.com",
  });
  expect(queMostrar({ url: URL, estado: abierta, cuentas: cuentas(2) }).titulo).toBe(
    "Esfinge · 2 cuentas de login.brevo.com",
  );
  expect(queMostrar({ url: URL, estado: abierta, cuentas: cuentas(12) }).insignia).toBe("9+");
});

test("insignia: rellenado enseña el ✓", () => {
  expect(queMostrar({ url: URL, estado: abierta, cuentas: cuentas(2), rellenado: true })).toMatchObject({
    icono: "activo",
    insignia: "✓",
  });
});

test("insignia: la bóveda cerrada, con candado y sin insignia", () => {
  const cerrada = { ok: true, estado: { existe: true, abierta: false } };
  expect(queMostrar({ url: URL, estado: cerrada })).toMatchObject({ icono: "cerrado", insignia: "" });
  expect(queMostrar({ url: URL, cuentas: { ok: false, motivo: "cerrada" } })).toMatchObject({
    icono: "cerrado",
  });
});

/**
 * **Los problemas ganan al ✓**: una página rellenada hace un rato, con la bóveda ya
 * cerrada, enseña el candado. Lo que importa ahora es que no se puede rellenar.
 */
test("insignia: un problema gana a lo rellenado", () => {
  const cerrada = { ok: true, estado: { existe: true, abierta: false } };
  expect(queMostrar({ url: URL, estado: cerrada, rellenado: true }).icono).toBe("cerrado");
  expect(
    queMostrar({ url: URL, estado: { ok: false, motivo: "sin-esfinge" }, rellenado: true }).insignia,
  ).toBe("!");
});

test("insignia: los problemas, apagado con ! y la frase de cada uno", () => {
  const casos: [unknown, RegExp][] = [
    [{ estado: { ok: true, estado: { existe: false, abierta: false } } }, /Todavía no hay bóveda/],
    [{ estado: abierta, cuentas: { ok: false, motivo: "sin-emparejar" } }, /Falta permitir/],
    [{ estado: { ok: false, motivo: "sin-esfinge" } }, /No se encuentra Esfinge/],
    [{ cuentas: { ok: false, motivo: "demasiado" } }, /Demasiadas preguntas/],
    [{}, /No se ha podido preguntar/],
  ];
  for (const [entrada, frase] of casos) {
    const q = queMostrar({ url: URL, ...(entrada as object) });
    expect(q.icono).toBe("apagado");
    expect(q.insignia).toBe("!");
    expect(q.titulo).toMatch(frase);
  }
});

test("insignia: todas las frases empiezan en mayúscula", () => {
  const frases = [
    queMostrar({ url: "http://x/" }),
    queMostrar({ url: URL, estado: abierta, cuentas: cuentas(0) }),
    queMostrar({ url: URL, cuentas: { ok: false, motivo: "cerrada" } }),
    queMostrar({ url: URL, estado: abierta, rellenado: true }),
  ].map((q) => q.titulo);
  for (const f of frases) expect(f[0]).toBe(f[0].toUpperCase());
});

test("insignia: sin aceptar el aviso de datos, «!» y abrir el panel, antes que nada", () => {
  for (const q of [
    { url: URL, aceptado: false },
    { url: "http://router.local/", aceptado: false },
    { url: URL, aceptado: false, rellenado: true, estado: abierta, cuentas: cuentas(2) },
  ]) {
    expect(queMostrar(q)).toMatchObject({
      icono: "apagado",
      insignia: "!",
      titulo: "Esfinge · Abre el panel para empezar",
    });
  }
});
