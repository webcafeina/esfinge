import { expect, test, type Page } from "@playwright/test";
import { build } from "vite";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const aqui = dirname(fileURLToPath(import.meta.url));

/**
 * Lo que se lee cuando una persona envía o teclea, con el teclado de un Chromium de
 * verdad: un `dispatchEvent` desde la página no cuenta, y eso solo se puede probar
 * con eventos que el navegador marque como suyos.
 */

let modulo = "";

test.beforeAll(async () => {
  const salida = (await build({
    configFile: false,
    logLevel: "silent",
    build: {
      write: false,
      lib: {
        entry: resolve(aqui, "../src/envios.ts"),
        formats: ["iife"],
        name: "Envios",
        fileName: () => "envios.js",
      },
    },
  })) as any;
  modulo = salida[0].output[0].code;
});

type Ventana = { tecleados: string[] };

async function montar(page: Page, html: string) {
  await page.setContent(`<!doctype html><meta charset="utf-8">${html}`);
  await page.addScriptTag({ content: modulo });
  await page.evaluate(() => {
    const w = window as unknown as Ventana;
    w.tecleados = [];
    // @ts-expect-error el módulo se inyecta como global en la página
    Envios.vigilarIdentificador((u: string) => w.tecleados.push(u));
  });
}

const tecleados = (page: Page) => page.evaluate(() => (window as unknown as Ventana).tecleados);

const GOOGLE = `<form onsubmit="return false"><input id="correo" type="email" autocomplete="username webauthn">
  <button type="button">Siguiente</button></form>`;

test("usuario tecleado: se recuerda lo que escribe una persona en la página de solo usuario", async ({ page }) => {
  await montar(page, GOOGLE);
  await page.click("#correo");
  await page.keyboard.type("alvaro@webcafeina.com");
  await expect.poll(() => tecleados(page)).toContain("alvaro@webcafeina.com");
});

test("usuario de la página: lo que pone Esfinge o el navegador también cuenta", async ({ page }) => {
  await montar(page, GOOGLE);
  await page.evaluate(() => {
    const campo = document.getElementById("correo") as HTMLInputElement;
    campo.value = "info@webcafeina.com";
    campo.dispatchEvent(new Event("input", { bubbles: true }));
    campo.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await expect.poll(() => tecleados(page)).toContain("info@webcafeina.com");
});

test("usuario de la página: el que pone el sitio sin avisar se lee al pulsar «Siguiente»", async ({ page }) => {
  await montar(page, GOOGLE);
  await page.evaluate(() => {
    (document.getElementById("correo") as HTMLInputElement).value = "alvaro@webcafeina.com";
  });
  expect(await tecleados(page)).toEqual([]);
  await page.click("button");
  expect(await tecleados(page)).toEqual(["alvaro@webcafeina.com"]);
});

test("usuario de la página: un clic fabricado por la página no lo lee", async ({ page }) => {
  await montar(page, GOOGLE);
  await page.evaluate(() => {
    (document.getElementById("correo") as HTMLInputElement).value = "alvaro@webcafeina.com";
    document.querySelector("button")!.click();
  });
  expect(await tecleados(page)).toEqual([]);
});

test("usuario tecleado: cambiar lo que puso Esfinge por otro correo se recuerda al pulsar Intro", async ({ page }) => {
  await montar(page, GOOGLE);
  await page.evaluate(() => {
    (document.getElementById("correo") as HTMLInputElement).value = "info@webcafeina.com";
  });
  await page.click("#correo");
  await page.keyboard.press("ControlOrMeta+A");
  await page.keyboard.type("alvaro@webcafeina.com");
  await page.keyboard.press("Enter");
  expect((await tecleados(page)).at(-1)).toBe("alvaro@webcafeina.com");
});

test("usuario tecleado: con una contraseña en la página no se recuerda nada", async ({ page }) => {
  await montar(page, `<form><input id="correo" autocomplete="username"><input type="password"></form>`);
  await page.click("#correo");
  await page.keyboard.type("alvaro@webcafeina.com");
  await page.waitForTimeout(500);
  expect(await tecleados(page)).toEqual([]);
});
