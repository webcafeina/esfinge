import { expect, test } from "@playwright/test";
import { avisoDe, empujarLaBandera, INTENTOS, seHaPodidoPreguntar } from "../src/bandera";
import type { Respuesta } from "../src/protocolo";

/**
 * La bandera que el mundo aislado empuja al principal (ADR 0048).
 *
 * Lo que se vigila aquí es **que insista cuando no se ha podido preguntar y solo
 * entonces**. El fallo que lo trajo no se veía en ninguna prueba: el banner no salía en
 * treinta segundos, con el `shim` instalado y la lista de dominios puesta, porque la
 * primera pregunta se había quedado sin contestar y la bandera se quedaba en falso para
 * el resto de la vida de la pestaña.
 */
const pedirLo = (...respuestas: Respuesta[]) => {
  let i = 0;
  const veces: number[] = [];
  return {
    veces,
    pedir: async () => {
      veces.push(i);
      return respuestas[Math.min(i++, respuestas.length - 1)];
    },
  };
};

/** Sin esperas de verdad: lo que se prueba es cuántas veces pregunta, no el reloj. */
const yaEsta = async () => {};

test("la bandera: una respuesta es una respuesta, y no se repite", async () => {
  const { pedir, veces } = pedirLo({ ok: true, llaves: [] });
  const avisos: unknown[] = [];
  await empujarLaBandera(pedir, (a) => avisos.push(a), yaEsta);
  expect(veces).toHaveLength(1);
  expect(avisos).toEqual([{ hay: false, sePuedeCrear: false }]);
});

/**
 * **El caso que costaba el banner.** `pedir` no lanza cuando el trabajador no contesta:
 * devuelve `ok: false` con un `error` y sin `motivo`. Antes eso se mandaba como un «no»
 * y nadie lo corregía nunca.
 */
test("la bandera: sin respuesta se vuelve a preguntar, y la segunda vale", async () => {
  const { pedir, veces } = pedirLo(
    { ok: false, error: "El trabajador de fondo no ha contestado" },
    { ok: true, llaves: [{ id: "1", nombre: "yo@github.com" }], puedeCrear: true },
  );
  const avisos: unknown[] = [];
  await empujarLaBandera(pedir, (a) => avisos.push(a), yaEsta);
  expect(veces).toHaveLength(2);
  expect(avisos).toEqual([{ hay: true, sePuedeCrear: true }]);
});

test("la bandera: si nunca contesta, se rinde y manda que ceda", async () => {
  const { pedir, veces } = pedirLo({ ok: false, error: "nada" });
  const avisos: unknown[] = [];
  await empujarLaBandera(pedir, (a) => avisos.push(a), yaEsta);
  expect(veces).toHaveLength(INTENTOS);
  expect(avisos).toEqual([{ hay: false, sePuedeCrear: false }]);
});

/**
 * Un `motivo` sí es una respuesta: lo pone el núcleo. **La bóveda cerrada no se vuelve a
 * preguntar**, y además es el caso en que la bandera tiene que salir en `true` —es lo que
 * permite al banner ofrecer abrirla—.
 */
test("la bandera: la bóveda cerrada es concluyente, y ofrece igual", async () => {
  const { pedir, veces } = pedirLo({ ok: false, motivo: "cerrada", quizas: true });
  const avisos: unknown[] = [];
  await empujarLaBandera(pedir, (a) => avisos.push(a), yaEsta);
  expect(veces).toHaveLength(1);
  expect(avisos).toEqual([{ hay: true, sePuedeCrear: true }]);
});

test("la bandera: si pedir lanza, también se insiste", async () => {
  let n = 0;
  const avisos: unknown[] = [];
  await empujarLaBandera(
    async () => {
      if (++n === 1) throw new Error("el puerto se ha caído");
      return { ok: false, motivo: "no-encaja" } satisfies Respuesta;
    },
    (a) => avisos.push(a),
    yaEsta,
  );
  expect(n).toBe(2);
  expect(avisos).toEqual([{ hay: false, sePuedeCrear: false }]);
});

test("la bandera: qué cuenta como haber podido preguntar", () => {
  const tabla: [Respuesta, boolean][] = [
    [{ ok: true }, true],
    [{ ok: false, motivo: "cerrada" }, true],
    [{ ok: false, motivo: "sin-consentimiento" }, true],
    [{ ok: false, motivo: "no-encaja" }, true],
    // Sin `motivo` nadie ha hablado: el plazo del puente o un error del canal.
    [{ ok: false, error: "El trabajador de fondo no ha contestado" }, false],
    [{ ok: false }, false],
  ];
  for (const [r, esperado] of tabla) {
    expect(seHaPodidoPreguntar(r), JSON.stringify(r)).toBe(esperado);
  }
});

test("la bandera: lo que se empuja sale de lo que contesta el núcleo", () => {
  expect(avisoDe({ ok: true, llaves: [], puedeCrear: true })).toEqual({ hay: false, sePuedeCrear: true });
  expect(avisoDe({ ok: true, llaves: [{ id: "1", nombre: "x" }] })).toEqual({ hay: true, sePuedeCrear: false });
  expect(avisoDe({ ok: false, motivo: "no-encaja" })).toEqual({ hay: false, sePuedeCrear: false });
});
