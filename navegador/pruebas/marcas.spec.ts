import { expect, test } from "@playwright/test";
import { build } from "vite";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const aqui = dirname(fileURLToPath(import.meta.url));

/**
 * Lo que Esfinge deja en un campo que rellena: el filete y el aviso (ADR 0031).
 *
 * Es lo que la ADR 0028 decía que no se hacía —dibujar en la página de otro—, así
 * que lo que se prueba son **sus límites**: el filete no lo quita Esfinge al
 * escribir y sí una persona, y devuelve el campo como estaba; el aviso no se
 * puede pulsar, va cerrado y se va solo.
 */

let modulo = "";

test.beforeAll(async () => {
  const salida = (await build({
    configFile: false,
    logLevel: "silent",
    build: {
      write: false,
      lib: {
        entry: resolve(aqui, "../src/marcas.ts"),
        formats: ["iife"],
        name: "Marcas",
        fileName: () => "marcas.js",
      },
    },
  })) as any;
  modulo = salida[0].output[0].code;
});

const leerSombra = () =>
  (window as unknown as { leer: () => { sombra: string; prioridad: string } }).leer();

test("el filete aguanta lo que escribe Esfinge y se va con lo que escribe una persona", async ({ page }) => {
  await page.setContent(
    `<!doctype html><meta charset="utf-8"><input id="p" type="password" style="box-shadow: red 1px 1px">`,
  );
  await page.addScriptTag({ content: modulo });
  await page.evaluate(() => {
    const c = document.getElementById("p") as HTMLInputElement;
    (window as unknown as { leer: () => unknown }).leer = () => ({
      sombra: c.style.getPropertyValue("box-shadow"),
      prioridad: c.style.getPropertyPriority("box-shadow"),
    });
  });

  await page.evaluate(() => {
    const c = document.getElementById("p") as HTMLInputElement;
    // @ts-expect-error el módulo se inyecta como global en la página
    Marcas.ponerFilete(c);
    // Lo que dispara Esfinge al escribir: no es de confianza, y no lo quita.
    c.dispatchEvent(new Event("input", { bubbles: true }));
  });
  const puesto = await page.evaluate(leerSombra);
  expect(puesto.prioridad).toBe("important");
  expect(puesto.sombra).toContain("rgb(242, 193, 78)");
  // **Sin anillo de piedra**: se leía como un borde negro, y se quitó en la 2.20.1.
  expect(puesto.sombra).not.toContain("43, 43, 49");

  // Una tecla de verdad, y el campo vuelve a como estaba, estilo de la web incluido.
  await page.locator("#p").focus();
  await page.keyboard.type("a");
  const despues = await page.evaluate(leerSombra);
  expect(despues.sombra).toContain("red");
  expect(despues.sombra).not.toContain("242, 193, 78");
  expect(despues.prioridad).toBe("");
});

test("el aviso va cerrado, no se puede pulsar, es uno solo y se va a los tres segundos", async ({ page }) => {
  await page.clock.install();
  await page.setContent(`<!doctype html><meta charset="utf-8"><form><input id="p" type="password"></form>`);
  await page.addScriptTag({ content: modulo });

  await page.evaluate(() => {
    // @ts-expect-error el módulo se inyecta como global en la página
    Marcas.avisar(document.getElementById("p"), "Rellenado por Esfinge");
  });
  const aviso = page.locator("esfinge-aviso");
  await expect(aviso).toHaveCount(1);
  expect(
    await aviso.evaluate((h) => ({ sombra: h.shadowRoot, eventos: getComputedStyle(h).pointerEvents })),
  ).toEqual({ sombra: null, eventos: "none" });

  // El del código sustituye al de la contraseña: uno a la vez.
  await page.evaluate(() => {
    // @ts-expect-error el módulo se inyecta como global en la página
    Marcas.avisar(document.getElementById("p"), "Código rellenado por Esfinge");
  });
  await expect(aviso).toHaveCount(1);

  await page.clock.runFor(3100);
  await expect(aviso).toHaveCount(0);
});
