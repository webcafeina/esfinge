import { expect, test } from "@playwright/test";
import { cuentaParaRellenarSola, mismoUsuario } from "../src/identidad";

/**
 * Con qué cuenta se rellena sola una página, sabiendo quién entra.
 *
 * Sale de lo que vio el cliente en Google con la 2.21.0: con `info@` guardada, entró
 * como `alvaro@` y la página de la contraseña se rellenó con la de `info@`. Y de lo
 * que pidió después: con varias cuentas, la que coincida con el correo.
 */

const INFO = { id: "1", titulo: "Google", usuario: "info@webcafeina.com" };
const ALVARO = { id: "2", titulo: "Google", usuario: "alvaro@webcafeina.com" };

test("identidad: con otro usuario tecleado, no se rellena con la única cuenta", () => {
  expect(cuentaParaRellenarSola([INFO], "alvaro@webcafeina.com")).toBeNull();
});

test("identidad: con el mismo usuario, o sin saber quién entra, se rellena como siempre", () => {
  expect(cuentaParaRellenarSola([INFO], "info@webcafeina.com")).toEqual(INFO);
  expect(cuentaParaRellenarSola([INFO], " INFO@webcafeina.com ")).toEqual(INFO);
  expect(cuentaParaRellenarSola([INFO], "")).toEqual(INFO);
  expect(cuentaParaRellenarSola([INFO], "   ")).toEqual(INFO);
});

test("identidad: con varias cuentas, la que coincide con quien entra", () => {
  expect(cuentaParaRellenarSola([INFO, ALVARO], "alvaro@webcafeina.com")).toEqual(ALVARO);
  expect(cuentaParaRellenarSola([INFO, ALVARO], "Info@Webcafeina.com")).toEqual(INFO);
});

test("identidad: con varias y sin saber quién entra, o sin ninguna que coincida, nada", () => {
  expect(cuentaParaRellenarSola([INFO, ALVARO], "")).toBeNull();
  expect(cuentaParaRellenarSola([INFO, ALVARO], "otra@webcafeina.com")).toBeNull();
  expect(cuentaParaRellenarSola([], "")).toBeNull();
});

test("identidad: dos cuentas con el mismo usuario no se deciden solas", () => {
  expect(cuentaParaRellenarSola([ALVARO, { ...ALVARO, id: "3" }], "alvaro@webcafeina.com")).toBeNull();
});

test("identidad: una cuenta sin usuario no vale para quien ha tecleado uno", () => {
  expect(cuentaParaRellenarSola([{ ...INFO, usuario: "" }], "alvaro@webcafeina.com")).toBeNull();
});

test("identidad: los usuarios se comparan como en Go", () => {
  expect(mismoUsuario("Alvaro@Webcafeina.com", " alvaro@webcafeina.com")).toBe(true);
  expect(mismoUsuario("alvaro@webcafeina.com", "info@webcafeina.com")).toBe(false);
});
