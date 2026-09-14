import { expect, test, type Page } from "@playwright/test";
import { build } from "vite";
import { mkdtempSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, extname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const aqui = dirname(fileURLToPath(import.meta.url));

/**
 * El panel, compilado de verdad y movido con una `chrome` de mentira.
 *
 * # Por qué existe
 *
 * **Los dos fallos de la 2.18.0 estaban fuera de lo probado.** La detección de
 * campos tenía doce casos en un Chromium de verdad; lo que la envuelve no lo
 * probaba nadie, y ahí salieron el oyente que contestaba desde el `iframe` del
 * captcha y el relleno a mano que no escribía. Los dos los vio el cliente a la
 * primera. Esto prueba **el panel**, que es la mitad de ese envoltorio.
 *
 * No carga la extensión en el navegador —eso sigue siendo deuda, y es la otra
 * mitad—, pero sí el `panel.html` que se publica, compilado con la configuración
 * de verdad, con la marca, los tokens y el CSS dentro. Lo falso es solo el
 * trabajador de fondo y la página, que aquí contestan lo que cada caso necesita.
 */

let compilado = "";

test.beforeAll(async () => {
  compilado = mkdtempSync(join(tmpdir(), "esfinge-panel-"));
  await build({
    configFile: resolve(aqui, "../vite.config.ts"),
    logLevel: "silent",
    build: { outDir: compilado, emptyOutDir: true },
  });
});

/** Lo que contesta la `chrome` de mentira en cada caso. */
type Guion = {
  /** Lo que contesta el trabajador a «cuentas». */
  cuentas: unknown;
  /**
   * Lo que contestan las tramas de la página al pedir «rellenar», con el retraso
   * de cada una. **Varias a propósito**: `tabs.connect` llega a todas las tramas.
   */
  tramas?: { ms: number; r: unknown }[];
};

const ORIGEN = "http://panel.esfinge.test";
const TIPOS: Record<string, string> = {
  ".html": "text/html",
  ".css": "text/css",
  ".js": "text/javascript",
};

async function abrir(page: Page, guion: Guion) {
  const errores: string[] = [];
  page.on("pageerror", (e) => errores.push(String(e)));
  page.on("console", (m) => m.type() === "error" && errores.push(m.text()));

  await page.route(`${ORIGEN}/**`, (ruta) => {
    const fichero = new URL(ruta.request().url()).pathname;
    try {
      ruta.fulfill({
        body: readFileSync(join(compilado, fichero)),
        contentType: TIPOS[extname(fichero)] ?? "application/octet-stream",
      });
    } catch {
      ruta.fulfill({ status: 404 });
    }
  });

  await page.addInitScript((g: Guion) => {
    type Oyente = (m: unknown) => void;
    const puerto = (alMandar: (m: unknown, oyentes: Oyente[]) => void) => {
      const oyentes: Oyente[] = [];
      return {
        onMessage: { addListener: (f: Oyente) => oyentes.push(f) },
        onDisconnect: { addListener: () => {} },
        disconnect() {},
        postMessage(m: unknown) {
          alMandar(m, oyentes);
        },
      };
    };
    (globalThis as Record<string, unknown>).chrome = {
      runtime: {
        getManifest: () => ({ version: "9.9.9" }),
        connect: () =>
          puerto((m, oyentes) => {
            const r = (m as { que: string }).que === "cuentas" ? g.cuentas : { ok: true, copiado: { portapapeles: 30 } };
            setTimeout(() => oyentes.forEach((f) => f(r)), 10);
          }),
      },
      tabs: {
        query: async () => [{ id: 7, url: "https://login.ejemplo.es/entrar" }],
        connect: () =>
          puerto((_, oyentes) => {
            for (const t of g.tramas ?? []) setTimeout(() => oyentes.forEach((f) => f(t.r)), t.ms);
          }),
      },
    };
  }, guion);

  await page.goto(`${ORIGEN}/panel.html`);
  return errores;
}

const TRES = {
  ok: true,
  cuentas: [
    { id: "1", titulo: "Correo", usuario: "yo@ejemplo.es", tieneCodigo: true },
    { id: "2", titulo: "Correo de la agencia", usuario: "agencia@ejemplo.es", tieneCodigo: false },
    { id: "3", titulo: "", usuario: "", tieneCodigo: true },
  ],
};

test("una fila por cuenta, y el código solo en las que lo tienen", async ({ page }) => {
  const errores = await abrir(page, { cuentas: TRES });

  const filas = page.locator("#lista li");
  await expect(filas).toHaveCount(3);
  await expect(page.locator("#cargando")).toBeHidden();

  await expect(filas.nth(0).getByRole("button", { name: "Copiar el código de un solo uso" })).toHaveCount(1);
  await expect(filas.nth(1).getByRole("button", { name: "Copiar el código de un solo uso" })).toHaveCount(0);

  // Lo que falta se dice, no se deja en blanco: con varias cuentas del mismo
  // sitio, la segunda línea es lo único que las distingue.
  await expect(filas.nth(2)).toContainText("Sin título");
  await expect(filas.nth(2)).toContainText("Sin usuario");

  expect(errores).toEqual([]);
});

/**
 * **El «Rellenar» de todas las filas cae en la misma columna.** En la primera
 * captura del panel nuevo no caía: la fila sin código tenía las acciones más
 * estrechas y su botón se iba a la derecha. Una prueba lo mide porque este tipo
 * de cosa vuelve sola en cuanto alguien toca el CSS.
 */
test("los botones de rellenar están alineados aunque falte el de código", async ({ page }) => {
  await abrir(page, { cuentas: TRES });
  const xs = await page.locator(".rellenar").evaluateAll((bs) => bs.map((b) => Math.round(b.getBoundingClientRect().left)));
  expect(xs).toHaveLength(3);
  expect(new Set(xs).size).toBe(1);
});

/**
 * Lo que viene de la bóveda se escribe como texto. El título de una entrada lo
 * escribió alguien —o salió de un CSV importado—, y el panel vive dentro de la
 * extensión, con sus permisos.
 */
test("un título con HTML dentro se enseña como texto", async ({ page }) => {
  const errores = await abrir(page, {
    cuentas: {
      ok: true,
      cuentas: [{ id: "1", titulo: '<img src=x onerror="document.title=1">', usuario: "<b>yo</b>" }],
    },
  });
  await expect(page.locator("#lista li")).toHaveCount(1);
  await expect(page.locator("#lista img, #lista b")).toHaveCount(0);
  await expect(page.locator("#lista li .nombre")).toHaveText('<img src=x onerror="document.title=1">');
  expect(await page.title()).toBe("Esfinge");
  expect(errores).toEqual([]);
});

/**
 * **El fallo de la 2.18.0, reproducido.** `tabs.connect` sin `frameId` llega a
 * todas las tramas de la pestaña: el `iframe` del captcha contestaba «aquí no hay
 * formulario» antes que la trama buena, y el panel se quedaba con eso en una
 * página que acababa de rellenarse sola.
 */
test("si una trama dice que no y otra que sí, gana el sí", async ({ page }) => {
  await abrir(page, {
    cuentas: TRES,
    tramas: [
      { ms: 10, r: { ok: false, error: "Aquí no hay ningún formulario de entrar que Esfinge sepa rellenar." } },
      { ms: 150, r: { ok: true } },
    ],
  });
  await page.locator(".rellenar").first().click();
  await expect(page.locator("#resultado")).toHaveText("Rellenado.");
  await expect(page.locator("#resultado")).toHaveClass(/bien/);
});

test("si todas dicen que no, se dice lo que dijeron", async ({ page }) => {
  await abrir(page, {
    cuentas: TRES,
    tramas: [{ ms: 10, r: { ok: false, error: "Aquí no hay ningún formulario de entrar que Esfinge sepa rellenar." } }],
  });
  await page.locator(".rellenar").first().click();
  await expect(page.locator("#resultado")).toHaveText(/Aquí no hay ningún formulario/);
  await expect(page.locator("#resultado")).toHaveClass(/mal/);
});

/**
 * Sin Esfinge, **lo que hay que hacer y lo que dijo el navegador van aparte**. Lo
 * segundo es lo único que distingue un manifiesto que falta de un binario que no
 * se puede ejecutar, así que no se esconde; pero no se mezcla con la instrucción.
 */
test("sin Esfinge, la instrucción y el detalle del navegador van separados", async ({ page }) => {
  const errores = await abrir(page, {
    cuentas: {
      ok: false,
      motivo: "sin-esfinge",
      error:
        "No se puede hablar con Esfinge. Comprueba que está instalada y que el canal con el " +
        "navegador está encendido en sus Ajustes. El navegador dice: Specified native messaging host not found.",
    },
  });
  await expect(page.locator("#estado h1")).toHaveText("No se encuentra Esfinge");
  await expect(page.locator("#estado .texto")).not.toContainText("El navegador dice");
  await expect(page.locator("#estado .detalle")).toHaveText("Specified native messaging host not found.");
  await expect(page.locator("#lista")).toBeHidden();
  expect(errores).toEqual([]);
});
