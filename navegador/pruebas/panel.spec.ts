import { expect, test, type Page } from "@playwright/test";
import { build } from "vite";
import { mkdtempSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, extname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { tinteDe } from "../../frontend/src/monograma";

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
  /** El icono que el navegador dice tener de la pestaña. */
  favicon?: string;
  /** Si ya se ha aceptado el aviso de datos (ADR 0033). Por defecto, sí. */
  aceptado?: boolean;
};

/** Lo que la `chrome` de mentira deja a la vista de las pruebas. */
type Rastro = {
  /** Lo que el panel ha mandado al trabajador de fondo. */
  __mensajes: { que: string }[];
  /** Lo que el panel ha guardado en `storage.local`. */
  __guardado: Record<string, { version: number }>;
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
    const rastro = globalThis as unknown as Rastro;
    rastro.__mensajes = [];
    rastro.__guardado =
      g.aceptado === false ? {} : { consentimiento: { version: 2 } };
    (globalThis as Record<string, unknown>).chrome = {
      storage: {
        local: {
          get: async () => ({ ...rastro.__guardado }),
          set: async (o: Record<string, { version: number }>) => {
            Object.assign(rastro.__guardado, o);
          },
        },
        onChanged: { addListener: () => {}, removeListener: () => {} },
      },
      runtime: {
        getManifest: () => ({ version: "9.9.9" }),
        connect: () =>
          puerto((m, oyentes) => {
            rastro.__mensajes.push(m as { que: string });
            const r = (m as { que: string }).que === "cuentas" ? g.cuentas : { ok: true, copiado: { portapapeles: 30 } };
            setTimeout(() => oyentes.forEach((f) => f(r)), 10);
          }),
      },
      tabs: {
        query: async () => [{ id: 7, url: "https://login.ejemplo.es/entrar", favIconUrl: g.favicon }],
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
  // Primero que estén: medir antes de que llegue la lista era una carrera que la
  // pregunta por la cuenta (E2), una más al abrir, hizo visible.
  await expect(page.locator(".rellenar")).toHaveCount(3);
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

/* ------------------------------------------------ la segunda pasada visual */

/**
 * El cuadro de cada cuenta sale de **la misma función que la ventana**, con el sitio
 * de la pestaña: el mismo sitio, el mismo color, en los dos sitios.
 */
test("el cuadro de la inicial tiene el tinte de la ventana para ese sitio", async ({ page }) => {
  await abrir(page, { cuentas: TRES });
  const cuadros = page.locator("#lista li .monograma");
  await expect(cuadros).toHaveCount(3);
  const esperado = String(tinteDe("login.ejemplo.es"));
  for (const t of await cuadros.evaluateAll((cs) => cs.map((c) => (c as HTMLElement).dataset.tinte))) {
    expect(t).toBe(esperado);
  }
  await expect(cuadros.nth(0)).toHaveText("C");
});

/**
 * **El icono de la web nunca se pide a internet**, que es la regla de la ADR 0024.
 * En Firefox —aquí, porque el manifiesto de mentira no lleva el permiso `favicon`—
 * solo vale uno incrustado; uno con `https:` se descarta y sale la inicial.
 */
test("el icono de la web nunca sale de internet", async ({ page }) => {
  const peticiones: string[] = [];
  page.on("request", (r) => peticiones.push(r.url()));
  await abrir(page, { cuentas: TRES, favicon: "https://login.ejemplo.es/favicon.ico" });
  await expect(page.locator("#lista li")).toHaveCount(3);
  await expect(page.locator("#favicon")).toBeHidden();
  await expect(page.locator("#inicial-sitio")).toBeVisible();
  expect(peticiones.filter((u) => u.includes("ejemplo.es"))).toEqual([]);
});

test("con un icono incrustado, se enseña", async ({ page }) => {
  const punto =
    "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==";
  await abrir(page, { cuentas: TRES, favicon: punto });
  await expect(page.locator("#favicon")).toBeVisible();
  await expect(page.locator("#inicial-sitio")).toBeHidden();
});

test("«Abierta» cuando la bóveda contesta, y no cuando está cerrada", async ({ page }) => {
  await abrir(page, { cuentas: TRES });
  await expect(page.locator("#abierta")).toBeVisible();

  const otra = await page.context().newPage();
  await abrir(otra, { cuentas: { ok: false, motivo: "cerrada" } });
  await expect(otra.locator("#estado h1")).toHaveText("La bóveda está cerrada");
  await expect(otra.locator("#abierta")).toBeHidden();
});

test("al rellenar, la fila dice «✓ Hecho» y luego vuelve", async ({ page }) => {
  await abrir(page, { cuentas: TRES, tramas: [{ ms: 50, r: { ok: true } }] });
  const boton = page.locator(".rellenar").first();
  await boton.click();
  await expect(boton).toHaveText("✓ Hecho");
  await expect(boton).toHaveText("Rellenar", { timeout: 4000 });
});

/**
 * **Sin tocar el ratón, y sin marco hasta usar el teclado.** En la 2.20.0 la primera
 * fila salía marcada al abrir el panel, sin que nadie tocara nada, y se leía como
 * seleccionada. Ahora al abrir no hay marco, Intro rellena la primera, y las flechas
 * marcan y mueven.
 */
const sombraDe = (li: import("@playwright/test").Locator) =>
  li.evaluate((e) => getComputedStyle(e).boxShadow);

test("con el teclado: sin marco al abrir, e Intro rellena la primera", async ({ page }) => {
  await abrir(page, { cuentas: TRES, tramas: [{ ms: 20, r: { ok: true } }] });
  const filas = page.locator("#lista li");
  await expect(filas).toHaveCount(3);
  await expect(filas.nth(0)).not.toBeFocused();
  expect(await sombraDe(filas.nth(0))).toBe("none");
  await page.keyboard.press("Enter");
  await expect(page.locator("#resultado")).toHaveText("Rellenado.");
});

test("con el teclado: las flechas marcan y mueven, e Intro rellena la marcada", async ({ page }) => {
  await abrir(page, { cuentas: TRES, tramas: [{ ms: 20, r: { ok: true } }] });
  const filas = page.locator("#lista li");
  await expect(filas).toHaveCount(3);
  await page.keyboard.press("ArrowDown");
  await expect(filas.nth(0)).toBeFocused();
  expect(await sombraDe(filas.nth(0))).not.toBe("none");
  await page.keyboard.press("ArrowDown");
  await expect(filas.nth(1)).toBeFocused();
  await page.keyboard.press("ArrowUp");
  await expect(filas.nth(0)).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.locator("#resultado")).toHaveText("Rellenado.");
});

test("la esfinge tenue en los avisos y la firma al pie", async ({ page }) => {
  await abrir(page, { cuentas: { ok: true, cuentas: [] } });
  await expect(page.locator("#estado h1")).toHaveText("No hay cuentas de este sitio");
  await expect(page.locator("#estado .tenue svg")).toHaveCount(1);
  await expect(page.locator("footer .firma")).toHaveText("▍webcafeína");
});

/**
 * **La esfinge tenue, entera dentro del aviso.** En la 2.20.0 salía desplazada fuera
 * de su caja y la caja la recortaba por abajo, así que se veía cortada y parecía mal
 * dibujada. Se mide la caja del dibujo contra la del aviso.
 */
test("la esfinge tenue cabe entera dentro del aviso", async ({ page }) => {
  await abrir(page, { cuentas: { ok: false, motivo: "cerrada" } });
  await expect(page.locator("#estado h1")).toHaveText("La bóveda está cerrada");
  const [dibujo, aviso] = await Promise.all([
    page.locator("#estado .tenue").boundingBox(),
    page.locator("#estado").boundingBox(),
  ]);
  expect(dibujo && aviso).toBeTruthy();
  expect(dibujo!.x).toBeGreaterThanOrEqual(aviso!.x);
  expect(dibujo!.y).toBeGreaterThanOrEqual(aviso!.y);
  expect(dibujo!.x + dibujo!.width).toBeLessThanOrEqual(aviso!.x + aviso!.width + 0.5);
  expect(dibujo!.y + dibujo!.height).toBeLessThanOrEqual(aviso!.y + aviso!.height + 0.5);
});

/**
 * **«Rellenar» con la letra del panel, fijada.** En Chrome de macOS salía más pequeño
 * que en Firefox, porque el botón traía la letra de su aspecto nativo. Esta prueba no
 * puede reproducir macOS, pero sí vigilar que el botón no depende de heredarla: sin
 * aspecto nativo y con el mismo tamaño que el título de la cuenta.
 */
test("el botón «Rellenar» lleva la letra del panel y no la del sistema", async ({ page }) => {
  await abrir(page, { cuentas: TRES });
  const boton = page.locator(".rellenar").first();
  const nombre = page.locator("#lista li .nombre").first();
  const [estiloBoton, tamanoNombre] = await Promise.all([
    boton.evaluate((b) => {
      const e = getComputedStyle(b);
      return { tamano: e.fontSize, aspecto: e.appearance };
    }),
    nombre.evaluate((n) => getComputedStyle(n).fontSize),
  ]);
  expect(estiloBoton.aspecto).toBe("none");
  expect(estiloBoton.tamano).toBe(tamanoNombre);
});

/**
 * **Y el texto del aviso no pisa la esfinge tenue.** Al dejarla entera, el texto
 * largo —el de «No se encuentra Esfinge», con su detalle— le pasaba por encima. Se
 * mide con el aviso más largo que hay.
 */
test("el texto del aviso no pasa por encima de la esfinge tenue", async ({ page }) => {
  await abrir(page, {
    cuentas: {
      ok: false,
      motivo: "sin-esfinge",
      error:
        "No se puede hablar con Esfinge. Comprueba que está instalada y que el canal con el " +
        "navegador está encendido en sus Ajustes. El navegador dice: Specified native messaging host not found.",
    },
  });
  await expect(page.locator("#estado .detalle")).toBeVisible();
  const dibujo = await page.locator("#estado .tenue").boundingBox();
  for (const sitio of ["#estado h1", "#estado .texto", "#estado .detalle"]) {
    const derecha = await page.locator(sitio).evaluate((e) => {
      // El borde derecho de lo escrito, no de la caja: un párrafo ocupa todo el ancho
      // aunque sus líneas no lleguen.
      const rango = document.createRange();
      rango.selectNodeContents(e);
      return Math.max(...[...rango.getClientRects()].map((r) => r.right));
    });
    expect(derecha, `${sitio} pisa la esfinge`).toBeLessThanOrEqual(dibujo!.x);
  }
});

/* -------------------------------------------- el aviso de datos (ADR 0033) */

const rastro = (page: Page): Promise<Rastro> =>
  page.evaluate(() => {
    // **Los dos campos y no `globalThis`**: la ventana entera no se puede pasar de la
    // página a la prueba, y llega vacía.
    const g = globalThis as unknown as Rastro;
    return { __mensajes: g.__mensajes, __guardado: g.__guardado };
  });

test("aviso de datos: sin aceptarlo, se enseña y no se pregunta nada a Esfinge", async ({ page }) => {
  const errores = await abrir(page, { cuentas: TRES, aceptado: false });
  await expect(page.locator("#aviso")).toBeVisible();
  await expect(page.locator("#aviso h1")).toHaveText("Esfinge y tus datos");
  await expect(page.locator("#lista")).toBeHidden();
  await expect(page.locator("#cargando")).toBeHidden();
  // **La prueba que importa**: ni una petición al trabajador de fondo, que es lo que
  // emparejaría el navegador y haría salir el aviso de permiso en Esfinge.
  await page.waitForTimeout(300);
  expect((await rastro(page)).__mensajes).toEqual([]);
  expect(errores).toEqual([]);
});

test("aviso de datos: aceptarlo lo guarda con su versión y trae las cuentas", async ({ page }) => {
  await abrir(page, { cuentas: TRES, aceptado: false });
  await page.click("#aceptar");
  await expect(page.locator("#lista li")).toHaveCount(3);
  await expect(page.locator("#aviso")).toBeHidden();
  const r = await rastro(page);
  expect(r.__guardado.consentimiento.version).toBe(2);
  // Primero el estado de la cuenta —¿hay cuenta y está cerrada?— y luego las
  // cuentas del sitio. Nada antes de aceptar, que es lo que mira la prueba de arriba.
  expect(r.__mensajes.map((m) => (m as { que?: string; cuenta?: string }).que ?? `cuenta:${(m as { cuenta?: string }).cuenta}`)).toEqual([
    "cuenta:estado",
    "cuentas",
  ]);
});

test("aviso de datos: Intro acepta, sin marco de foco al abrir", async ({ page }) => {
  await abrir(page, { cuentas: TRES, aceptado: false });
  await expect(page.locator("#aviso")).toBeVisible();
  expect(await page.evaluate(() => document.activeElement === document.body)).toBe(true);
  await page.keyboard.press("Enter");
  await expect(page.locator("#lista li")).toHaveCount(3);
});

test("aviso de datos: los enlaces van a la web del proyecto, en otra pestaña", async ({ page }) => {
  await abrir(page, { cuentas: TRES, aceptado: false });
  const enlaces = await page.locator("#aviso a").evaluateAll((as) =>
    as.map((a) => [(a as HTMLAnchorElement).href, (a as HTMLAnchorElement).target]),
  );
  expect(enlaces).toEqual([
    ["https://webcafeina.github.io/esfinge/privacidad.html", "_blank"],
    ["https://webcafeina.github.io/esfinge/soporte.html", "_blank"],
  ]);
});

test("aviso de datos: ya aceptado, no se enseña", async ({ page }) => {
  await abrir(page, { cuentas: TRES });
  await expect(page.locator("#lista li")).toHaveCount(3);
  await expect(page.locator("#aviso")).toBeHidden();
});
