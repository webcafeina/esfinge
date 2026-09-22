import { expect, test, type Page } from "@playwright/test";
import { build } from "vite";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const aqui = dirname(fileURLToPath(import.meta.url));

/**
 * La tarjeta «¿Guardar en Esfinge?», en un Chromium de verdad.
 *
 * Es **el primer elemento de Esfinge que se puede pulsar dentro de la web de otro**
 * (ADR 0032), así que lo que más se prueba son sus límites: la sombra va cerrada y
 * **un clic fabricado no hace nada**; solo decide una persona.
 *
 * Los clics de verdad se dan con el ratón de Playwright sobre la posición de cada
 * botón. Para saber dónde están, la función devuelve la raíz de la sombra; en la
 * extensión esa raíz vive en el mundo aislado del guion, donde la página no llega.
 */

let modulo = "";

test.beforeAll(async () => {
  const salida = (await build({
    configFile: false,
    logLevel: "silent",
    build: {
      write: false,
      lib: {
        entry: resolve(aqui, "../src/tarjeta.ts"),
        formats: ["iife"],
        name: "Tarjeta",
        fileName: () => "tarjeta.js",
      },
    },
  })) as any;
  modulo = salida[0].output[0].code;
});

type Ventana = {
  decisiones: unknown[];
  reintentos: number;
  tarjeta: { raiz: ShadowRoot; poner: (e: unknown) => void };
};

async function abrir(page: Page, estado: unknown, resultado = { ok: true }) {
  await page.setContent(`<!doctype html><meta charset="utf-8"><body style="margin:0;min-height:700px"></body>`);
  await page.addScriptTag({ content: modulo });
  await page.evaluate(
    ([e, r]) => {
      const w = window as unknown as Ventana;
      w.decisiones = [];
      w.reintentos = 0;
      // @ts-expect-error el módulo se inyecta como global en la página
      w.tarjeta = Tarjeta.mostrarTarjeta(
        e,
        async (d: unknown) => {
          w.decisiones.push(d);
          return r;
        },
        () => {
          w.reintentos++;
        },
      );
    },
    [estado, resultado],
  );
}

/** El centro de un elemento de dentro de la tarjeta, para pulsarlo con el ratón. */
async function centro(page: Page, selector: string) {
  return page.evaluate((s) => {
    const el = (window as unknown as Ventana).tarjeta.raiz.querySelector(s)!;
    const r = el.getBoundingClientRect();
    return { x: r.x + r.width / 2, y: r.y + r.height / 2 };
  }, selector);
}

async function pulsar(page: Page, selector: string) {
  const { x, y } = await centro(page, selector);
  await page.mouse.click(x, y);
}

const decisiones = (page: Page) => page.evaluate(() => (window as unknown as Ventana).decisiones);

const GUARDAR = {
  tipo: "oferta",
  oferta: { accion: "guardar", sitio: "login.brevo.com", titulo: "Brevo" },
  usuario: "info@webcafeina.com",
};

test("tarjeta: va cerrada y un clic fabricado por la página no hace nada", async ({ page }) => {
  await abrir(page, GUARDAR);
  expect(await page.locator("esfinge-tarjeta").evaluate((h) => h.shadowRoot)).toBeNull();
  await page.evaluate(() =>
    ((window as unknown as Ventana).tarjeta.raiz.querySelector("button.principal") as HTMLButtonElement).click(),
  );
  expect(await decisiones(page)).toEqual([]);
});

test("tarjeta: guardar con el título sugerido cambiado", async ({ page }) => {
  await abrir(page, GUARDAR);
  expect(await page.evaluate(() => ((window as unknown as Ventana).tarjeta.raiz.querySelector("input") as HTMLInputElement).value)).toBe("Brevo");
  await pulsar(page, "input");
  await page.keyboard.press("ControlOrMeta+A");
  await page.keyboard.type("Brevo trabajo");
  await pulsar(page, "button.principal");
  await expect.poll(() => decisiones(page)).toEqual([{ accion: "guardar", titulo: "Brevo trabajo" }]);
});

test("tarjeta: tras guardar dice que se ha guardado y se va sola", async ({ page }) => {
  await page.clock.install();
  await abrir(page, GUARDAR);
  await pulsar(page, "button.principal");
  await expect
    .poll(() => page.evaluate(() => (window as unknown as Ventana).tarjeta.raiz.textContent))
    .toContain("Guardado en Esfinge");
  await page.clock.runFor(2100);
  await expect(page.locator("esfinge-tarjeta")).toHaveCount(0);
});

test("tarjeta: con varias cuentas se elige cuál actualizar", async ({ page }) => {
  await abrir(page, {
    tipo: "oferta",
    oferta: {
      accion: "actualizar",
      sitio: "dash.cloudflare.com",
      cuentas: [
        { id: "1", titulo: "Cloudflare", usuario: "info@webcafeina.com" },
        { id: "2", titulo: "Cloudflare clientes", usuario: "clientes@webcafeina.com" },
      ],
    },
    usuario: "",
  });
  await page.evaluate(() => {
    ((window as unknown as Ventana).tarjeta.raiz.querySelector("select") as HTMLSelectElement).value = "2";
  });
  await pulsar(page, "button.principal");
  await expect.poll(() => decisiones(page)).toEqual([{ accion: "actualizar", id: "2" }]);
});

test("tarjeta: con la bóveda cerrada pide abrirla, y «Ya la he abierto» vuelve a mirar", async ({ page }) => {
  await abrir(page, { tipo: "cerrada", sitio: "login.brevo.com", usuario: "info@webcafeina.com" });
  expect(await page.evaluate(() => (window as unknown as Ventana).tarjeta.raiz.textContent)).toContain(
    "Abre la bóveda de Esfinge",
  );
  await pulsar(page, "button.principal");
  expect(await page.evaluate(() => (window as unknown as Ventana).reintentos)).toBe(1);
});

test("tarjeta: «Nunca en este sitio» y «Ahora no»", async ({ page }) => {
  await abrir(page, GUARDAR);
  await pulsar(page, "button.discreto");
  await expect.poll(() => decisiones(page)).toEqual([{ accion: "nunca" }]);

  await abrir(page, GUARDAR);
  const botones = await page.evaluate(() =>
    [...(window as unknown as Ventana).tarjeta.raiz.querySelectorAll("button")].map((b) => b.textContent),
  );
  expect(botones).toEqual(["Ahora no", "Guardar", "Nunca en este sitio"]);
  const ahoraNo = await page.evaluate(() => {
    const b = [...(window as unknown as Ventana).tarjeta.raiz.querySelectorAll("button")][0];
    const r = b.getBoundingClientRect();
    return { x: r.x + r.width / 2, y: r.y + r.height / 2 };
  });
  await page.mouse.click(ahoraNo.x, ahoraNo.y);
  await expect.poll(() => decisiones(page)).toEqual([{ accion: "ahora-no" }]);
  await expect(page.locator("esfinge-tarjeta")).toHaveCount(0);
});

test("tarjeta: si guardar falla, lo dice y deja volver a intentarlo", async ({ page }) => {
  await abrir(page, GUARDAR, { ok: false, error: "Demasiados cambios seguidos en la bóveda" } as never);
  await pulsar(page, "button.principal");
  await expect
    .poll(() => page.evaluate(() => (window as unknown as Ventana).tarjeta.raiz.textContent))
    .toContain("Demasiados cambios seguidos");
  expect(
    await page.evaluate(() =>
      ((window as unknown as Ventana).tarjeta.raiz.querySelector("button.principal") as HTMLButtonElement).disabled,
    ),
  ).toBe(false);
});
