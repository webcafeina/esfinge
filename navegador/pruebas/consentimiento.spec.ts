import { expect, test } from "@playwright/test";
import { vale, VERSION_DEL_AVISO } from "../src/consentimiento";

/**
 * Qué cuenta como aceptado el aviso de datos (ADR 0033).
 *
 * **Solo la versión de ahora**: si lo que dice el aviso cambia, sube la versión y
 * todo el que lo aceptó antes vuelve a verlo, que es lo que pide Chrome cuando
 * cambian las prácticas de datos después de instalar.
 */
test("consentimiento: solo vale la aceptación del aviso de ahora", () => {
  expect(vale({ version: VERSION_DEL_AVISO, cuando: "2026-09-14T10:00:00Z" })).toBe(true);
  expect(vale({ version: VERSION_DEL_AVISO - 1, cuando: "2026-01-01T10:00:00Z" })).toBe(false);
  expect(vale({ version: String(VERSION_DEL_AVISO) })).toBe(false);
});

test("consentimiento: lo que no es una aceptación, no vale", () => {
  for (const cosa of [undefined, null, "sí", true, 1, {}, []]) {
    expect(vale(cosa)).toBe(false);
  }
});
