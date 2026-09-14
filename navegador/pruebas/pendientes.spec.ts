import { expect, test } from "@playwright/test";
import {
  mismoSitio,
  sirvePara,
  vigente,
  VIDA_DEL_PENDIENTE,
  VIDA_DEL_USUARIO,
  type Pendiente,
} from "../src/pendientes";

/**
 * Cuándo se ofrece lo que se acaba de enviar, sin navegador.
 *
 * Es una función pura porque lo que la envuelve —el trabajador de fondo con la
 * contraseña en memoria— necesita la extensión cargada, que sigue siendo deuda.
 */

const envio = (origen: string, cuando = 0): Pendiente => ({
  origen,
  usuario: "yo@ejemplo.es",
  secreto: "clave",
  forma: "entrar",
  cuando,
});

test("pendiente: caduca a los dos minutos", () => {
  const p = envio("https://login.brevo.com/", 1000);
  expect(vigente(p, 1000 + VIDA_DEL_PENDIENTE - 1)).toBe(true);
  expect(vigente(p, 1000 + VIDA_DEL_PENDIENTE)).toBe(false);
});

test("pendiente: el mismo sitio, aunque cambie el subdominio", () => {
  expect(mismoSitio("login.brevo.com", "app.brevo.com")).toBe(true);
  expect(mismoSitio("accounts.google.com", "myaccount.google.com")).toBe(true);
  expect(mismoSitio("dash.cloudflare.com", "dash.cloudflare.com")).toBe(true);
});

test("pendiente: otro sitio no, ni con un sufijo corto compartido", () => {
  expect(mismoSitio("login.brevo.com", "login.otro.com")).toBe(false);
  expect(mismoSitio("banco.co.uk", "malo.co.uk")).toBe(false);
  expect(mismoSitio("agencia.com.es", "otra.com.es")).toBe(false);
  expect(mismoSitio("", "brevo.com")).toBe(false);
});

test("pendiente: solo sirve en https, en el mismo sitio y a tiempo", () => {
  const p = envio("https://login.brevo.com/entrar", 0);
  expect(sirvePara(p, "https://app.brevo.com/panel", 5000)).toBe(true);
  expect(sirvePara(p, "http://app.brevo.com/panel", 5000)).toBe(false);
  expect(sirvePara(p, "https://otra-web.com/", 5000)).toBe(false);
  expect(sirvePara(p, "https://app.brevo.com/panel", VIDA_DEL_PENDIENTE + 1)).toBe(false);
});

test("pendiente: el usuario tecleado dura más que la contraseña, y solo en su sitio", () => {
  const escrito = { origen: "https://accounts.google.com/v3/signin/identifier", usuario: "alvaro@webcafeina.com", cuando: 0 };
  const despues = VIDA_DEL_PENDIENTE + 1000;
  expect(VIDA_DEL_USUARIO).toBeGreaterThan(VIDA_DEL_PENDIENTE);
  expect(sirvePara(escrito, "https://accounts.google.com/v3/signin/challenge/pwd", despues, VIDA_DEL_USUARIO)).toBe(true);
  expect(sirvePara(escrito, "https://accounts.google.com/v3/signin/challenge/pwd", VIDA_DEL_USUARIO, VIDA_DEL_USUARIO)).toBe(false);
  expect(sirvePara(escrito, "https://otra-web.com/", 1000, VIDA_DEL_USUARIO)).toBe(false);
});
