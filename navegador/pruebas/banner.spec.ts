import { expect, test, type Page } from "@playwright/test";
import { build } from "vite";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const aqui = dirname(fileURLToPath(import.meta.url));

/**
 * El banner de las llaves de acceso, en un Chromium de verdad (ADR 0048).
 *
 * Copia estructural de `tarjeta.spec.ts`, y por lo mismo: es un elemento nuestro
 * que se pulsa dentro de la web de otro, así que lo que más se prueba son sus
 * límites. Con tres cosas de más que la tarjeta no tiene: **atrapa el foco**, `Esc`
 * cede, y **no lleva ningún campo de contraseña**.
 */

let modulo = "";

test.beforeAll(async () => {
  const salida = (await build({
    configFile: false,
    logLevel: "silent",
    build: {
      write: false,
      lib: { entry: resolve(aqui, "../src/banner.ts"), formats: ["iife"], name: "B", fileName: () => "b.js" },
    },
  })) as unknown as { output: { code: string }[] }[];
  modulo = salida[0].output[0].code;
});

type Ventana = { decisiones: unknown[]; banner: { raiz: ShadowRoot; poner: (e: unknown) => void } };

const UNA = { tipo: "elegir", rpId: "github.com", llaves: [{ id: "k1", nombre: "yo@ejemplo.com" }] };
const VARIAS = {
  tipo: "elegir",
  rpId: "github.com",
  llaves: [
    { id: "k1", nombre: "yo@ejemplo.com" },
    { id: "k2", nombre: "otra@ejemplo.com" },
  ],
};

async function abrir(page: Page, estado: unknown) {
  await page.setContent(`<!doctype html><meta charset="utf-8"><body style="margin:0;min-height:700px"><button id="dela-pagina">de la página</button></body>`);
  await page.addScriptTag({ content: modulo });
  await page.evaluate((e) => {
    const w = window as unknown as Ventana;
    w.decisiones = [];
    // @ts-expect-error el módulo se inyecta como global
    w.banner = B.mostrarBanner(e, (d: unknown) => w.decisiones.push(d));
  }, estado);
}

async function centro(page: Page, selector: string) {
  return page.evaluate((s) => {
    const el = (window as unknown as Ventana).banner.raiz.querySelector(s)!;
    const r = el.getBoundingClientRect();
    return { x: r.x + r.width / 2, y: r.y + r.height / 2 };
  }, selector);
}

async function pulsar(page: Page, selector: string, espera = 500) {
  await page.waitForTimeout(espera);
  const { x, y } = await centro(page, selector);
  await page.mouse.click(x, y);
}

const decisiones = (page: Page) => page.evaluate(() => (window as unknown as Ventana).decisiones);

test("banner: sombra cerrada, y un clic fabricado por la página no decide nada", async ({ page }) => {
  await abrir(page, UNA);
  expect(await page.locator("esfinge-llave").evaluate((h) => h.shadowRoot)).toBeNull();
  // **Se deja pasar el rebote antes de fabricar el clic.** Pulsando enseguida, lo
  // que lo para es el plazo y no el `isTrusted`: la prueba salía en verde aunque se
  // quitara la comprobación de que hubo una persona. Comprobado mutándola.
  await page.waitForTimeout(500);
  await page.evaluate(() =>
    ((window as unknown as Ventana).banner.raiz.querySelector("button.principal") as HTMLButtonElement).click(),
  );
  expect(await decisiones(page)).toEqual([]);
});

/**
 * **El rebote de «Aceptar» es mayor que el de los demás**, y esta prueba fija la
 * diferencia: la tarjeta arriesga una contraseña guardada de más; esto, una
 * identificación. Un clic a los 300 ms vale para «Ahora no» y no para «Aceptar».
 */
test("banner: un clic demasiado pronto no acepta, y sí cancela", async ({ page }) => {
  await abrir(page, UNA);
  await pulsar(page, "button.principal", 300);
  expect(await decisiones(page), "ha aceptado antes de dar tiempo a leerlo").toEqual([]);

  await abrir(page, UNA);
  await pulsar(page, "button:not(.principal):not(.discreto)", 300);
  expect(await decisiones(page)).toEqual([{ accion: "ahora-no" }]);
});

test("banner: con una llave, aceptar manda su identificador", async ({ page }) => {
  await abrir(page, UNA);
  await pulsar(page, "button.principal");
  expect(await decisiones(page)).toEqual([{ accion: "aceptar", id: "k1" }]);
  // Y al decidir se va: no se queda un velo encima de la página de nadie.
  expect(await page.locator("esfinge-llave").count()).toBe(0);
});

test("banner: con varias, se elige y se acepta la elegida", async ({ page }) => {
  await abrir(page, VARIAS);
  await page.evaluate(() => {
    const s = (window as unknown as Ventana).banner.raiz.querySelector("select") as HTMLSelectElement;
    s.value = "k2";
  });
  await pulsar(page, "button.principal");
  expect(await decisiones(page)).toEqual([{ accion: "aceptar", id: "k2" }]);
});

test("banner: «Usar otra llave» cede al navegador", async ({ page }) => {
  await abrir(page, UNA);
  await pulsar(page, "button.discreto");
  expect(await decisiones(page)).toEqual([{ accion: "otra-llave" }]);
});

test("banner: Esc cede, como cualquier diálogo", async ({ page }) => {
  await abrir(page, UNA);
  await page.waitForTimeout(500);
  await page.keyboard.press("Escape");
  expect(await decisiones(page)).toEqual([{ accion: "ahora-no" }]);
  expect(await page.locator("esfinge-llave").count()).toBe(0);
});

/**
 * **El foco no se sale.** Sin esto, tabular saca el foco al documento de debajo y
 * un sitio puede llevárselo a un campo suyo mientras el banner sigue a la vista.
 */
test("banner: el foco se queda dentro", async ({ page }) => {
  await abrir(page, VARIAS);
  await page.waitForTimeout(300);
  await page.evaluate(() => {
    (window as unknown as Ventana).banner.raiz.querySelector("select")!.dispatchEvent(new Event("focus"));
    ((window as unknown as Ventana).banner.raiz.querySelector("select") as HTMLElement).focus();
  });
  // **Se mira después de cada tabulador, no al final.** Al final no vale: sin
  // atrapar el foco, seis pulsaciones dan la vuelta entera y vuelven a caer dentro,
  // así que la prueba salía en verde con el atrapador quitado. Lo que hay que
  // comprobar es que **no se sale ni una vez**.
  for (let i = 0; i < 6; i++) {
    await page.keyboard.press("Tab");
    const donde = await page.evaluate(() => ({
      fuera: document.activeElement?.id ?? "",
      dentro: (window as unknown as Ventana).banner.raiz.activeElement?.tagName ?? "nada",
    }));
    expect(donde.fuera, `tras ${i + 1} tabuladores, el foco se ha ido a la página`).not.toBe("dela-pagina");
    expect(["BUTTON", "SELECT"], `tras ${i + 1} tabuladores`).toContain(donde.dentro);
  }
});

/**
 * **Y lo que no lleva, que es la mitad del diseño.** Ni con la bóveda cerrada hay
 * un campo donde teclear la maestra: es lo único que protege de que la propia
 * página dibuje una copia de esto, y por eso el pie lo dice donde se lee.
 */
test("banner: nunca hay dónde escribir una contraseña, y el pie lo dice", async ({ page }) => {
  for (const estado of [UNA, VARIAS, { tipo: "cerrada", rpId: "github.com" }]) {
    await abrir(page, estado);
    const r = await page.evaluate(() => {
      const raiz = (window as unknown as Ventana).banner.raiz;
      return {
        campos: raiz.querySelectorAll("input, textarea, [contenteditable]").length,
        pie: raiz.querySelector(".pie")?.textContent ?? "",
      };
    });
    expect(r.campos, `con el estado ${JSON.stringify(estado)}`).toBe(0);
    expect(r.pie).toContain("no el navegador");
    expect(r.pie).toContain("solo se teclea en el panel");
  }
});

test("banner: con la bóveda cerrada dice que se abra, y no ofrece aceptar", async ({ page }) => {
  await abrir(page, { tipo: "cerrada", rpId: "github.com" });
  const r = await page.evaluate(() => {
    const raiz = (window as unknown as Ventana).banner.raiz;
    return {
      principal: raiz.querySelectorAll("button.principal").length,
      texto: raiz.querySelector(".aviso")?.textContent ?? "",
      sitio: raiz.querySelector(".sitio")?.textContent ?? "",
    };
  });
  expect(r.principal).toBe(0);
  expect(r.texto).toContain("Abre Esfinge");
  expect(r.sitio).toBe("github.com");
});

/** Lo que llega de fuera se pone con `textContent`, como en toda la extensión. */
test("banner: el sitio se escribe, no se interpreta", async ({ page }) => {
  await abrir(page, { tipo: "elegir", rpId: "<img src=x onerror=alert(1)>", llaves: [{ id: "k", nombre: "<b>yo</b>" }] });
  const r = await page.evaluate(() => {
    const raiz = (window as unknown as Ventana).banner.raiz;
    return {
      sitio: raiz.querySelector(".sitio")?.textContent ?? "",
      etiquetas: raiz.querySelectorAll("img, b").length,
    };
  });
  expect(r.sitio).toBe("<img src=x onerror=alert(1)>");
  expect(r.etiquetas).toBe(0);
});
