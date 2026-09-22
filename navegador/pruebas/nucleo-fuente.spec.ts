import { expect, test } from "@playwright/test";
import { Boveda } from "../src/nucleo/boveda";
import { atender } from "../src/nucleo/fuente";
import type { Entrada } from "../src/nucleo/entrada";
import type { Peticion } from "../src/protocolo";

/**
 * Los verbos contestados con la bóveda del navegador (ADR 0040), con las reglas de
 * `fuenteDelNavegador` en Go. Lo que importa de verdad está en tres: que **nunca**
 * salga una contraseña hacia otro sitio, qué se ofrece guardar o actualizar, y el
 * freno de rellenos.
 */

const MAESTRA = "una maestra larga para las pruebas de los verbos";
const cred = (titulo: string, usuario: string, secreto: string, sitio: string, extra: Partial<Entrada> = {}): Entrada => ({
  id: "",
  tipo: "credencial",
  titulo,
  usuario,
  secreto,
  sitios: [sitio],
  creada: "",
  cambiada: "",
  ...extra,
});
const p = (x: Partial<Peticion>): Peticion => ({ version: 1, que: "cuentas", ...x }) as Peticion;

async function bovedaDePrueba() {
  const { boveda } = await Boveda.crear(MAESTRA);
  const banco = await boveda.poner(cred("Banco", "yo", "clave-banco", "https://www.banco.es/entrar", { totp: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" }));
  await boveda.poner(cred("Correo", "yo@correo.es", "clave-correo", "correo.es"));
  await boveda.poner({ id: "", tipo: "nota", titulo: "Nota del banco", notas: "x", sitios: ["banco.es"], creada: "", cambiada: "" });
  return { b: boveda, banco };
}

test("verbos: cuentas del sitio, solo credenciales, y con el código dicho", async () => {
  const { b } = await bovedaDePrueba();
  const r = await atender(p({ que: "cuentas", origen: "https://banco.es/x" }), { existe: true, boveda: b });
  expect(r.cuentas?.map((c) => [c.titulo, c.tieneCodigo])).toEqual([["Banco", true]]);
});

test("verbos: una contraseña no sale nunca hacia otro sitio", async () => {
  const { b, banco } = await bovedaDePrueba();
  for (const origen of ["https://banco.es.malo.com/", "https://correo.es/", "http://banco.es/"]) {
    const r = await atender(p({ que: "rellenar", id: banco.id, origen }), { existe: true, boveda: b });
    expect(r.ok, origen).toBe(false);
    expect(r.relleno).toBeUndefined();
  }
  const bien = await atender(p({ que: "rellenar", id: banco.id, origen: "https://banco.es/" }), { existe: true, boveda: b });
  expect(bien.relleno).toEqual({ usuario: "yo", secreto: "clave-banco" });
});

test("verbos: cerrada lo dice, y el origen se mira antes que la bóveda", async () => {
  expect((await atender(p({ que: "cuentas", origen: "https://banco.es" }), { existe: true, boveda: null })).motivo).toBe("cerrada");
  expect((await atender(p({ que: "cuentas", origen: "https://banco.es" }), { existe: false, boveda: null })).motivo).toBe("sin-boveda");
  expect((await atender(p({ que: "cuentas", origen: "http://banco.es" }), { existe: true, boveda: null })).motivo).toBe("origen");
});

test("verbos: qué se ofrece tras un envío", async () => {
  const { b, banco } = await bovedaDePrueba();
  const ofrece = async (usuario: string, secreto: string) =>
    (await atender(p({ que: "ofrecer", origen: "https://banco.es/entrar", usuario, secreto }), { existe: true, boveda: b })).oferta;
  expect((await ofrece("yo", "clave-banco"))?.accion).toBe("nada"); // ya está
  expect(await ofrece("YO", "otra")).toMatchObject({ accion: "actualizar", cuentas: [{ id: banco.id }] });
  expect(await ofrece("otra persona", "otra")).toMatchObject({ accion: "guardar", titulo: "Banco", sitio: "banco.es" });
  expect(await ofrece("", "otra")).toMatchObject({ accion: "actualizar", cuentas: [{ id: banco.id }] });
  await b.excluir("banco.es");
  expect((await ofrece("otra persona", "otra"))?.accion).toBe("nada"); // «Nunca en este sitio»
});

test("verbos: guardar lo pone para el sitio del envío, y actualizar deja la anterior en el historial", async () => {
  const { b, banco } = await bovedaDePrueba();
  const g = await atender(p({ que: "guardar-cuenta", origen: "https://login.nuevo.es/entrar", usuario: "yo", secreto: "s", titulo: "" }), {
    existe: true,
    boveda: b,
  });
  expect(g.guardada?.titulo).toBe("Nuevo");
  const nueva = b.buscar("Nuevo")[0];
  expect(nueva.sitios).toEqual(["https://login.nuevo.es"]);

  await atender(p({ que: "actualizar-cuenta", origen: "https://banco.es/", id: banco.id, secreto: "clave-nueva" }), { existe: true, boveda: b });
  const v = b.ver(banco.id)!;
  expect(v.secreto).toBe("clave-nueva");
  expect(v.historial?.[0].secreto).toBe("clave-banco");
});

test("verbos: el freno de rellenos, doce por minuto", async () => {
  const { b, banco } = await bovedaDePrueba();
  const t = Date.now() + 10 * 60_000; // una ventana nueva, sin lo de las otras pruebas
  let bien = 0;
  for (let i = 0; i < 15; i++) {
    const r = await atender(p({ que: "rellenar", id: banco.id, origen: "https://banco.es/" }), { existe: true, boveda: b }, t);
    if (r.ok) bien++;
    else expect(r.motivo).toBe("demasiado");
  }
  expect(bien).toBe(12);
});
