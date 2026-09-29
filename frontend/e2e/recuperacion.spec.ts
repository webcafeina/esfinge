/**
 * La clave de recuperación, **en su propia ventana**.
 *
 * Estas dos pruebas vivían en `esfinge.spec.ts` y **se pusieron rojas tres veces en
 * quince días** —el 2026-09-10, el 2026-09-14 y el 2026-09-25—, siempre en verde al
 * repetirlas solas. La causa era compartir bóveda con todas las demás del fichero: la
 * primera exige **la consola limpia**, y cualquier ruido de fondo de otra prueba
 * —el goteo de iconos de unas entradas que creó la de al lado, una llamada al puente
 * cortada por una navegación— la tumbaba. Repetir la tanda entera las ponía en verde,
 * que es la firma de una dependencia del orden.
 *
 * Aquí tienen **su propio Go y su propia carpeta de configuración** (el 34473 y el
 * 5176 de `playwright.config.ts`), así que en esa bóveda no hay nada que ellas no
 * hayan puesto. La deuda decía «cada prueba con su bóveda, o su propia carpeta de
 * configuración»; esto es lo segundo, que es lo que el montaje ya sabía hacer.
 */
import { expect, test, type Page } from "@playwright/test";

/** La suya, no la de `baseURL`: estas pruebas no comparten servidor con nadie. */
const VENTANA = "http://127.0.0.1:5176";

const MAESTRA = "una maestra de prueba";

const accion = (page: Page, nombre: string) =>
  page.locator(".contenido").getByRole("button", { name: nombre, exact: true });
const seccion = (page: Page, nombre: string) =>
  page.locator(".lateral").getByRole("button", { name: nombre, exact: true });

function vigilarConsola(page: Page): string[] {
  const errores: string[] = [];
  page.on("console", (m) => m.type() === "error" && errores.push(m.text()));
  page.on("pageerror", (e) => errores.push(String(e)));
  return errores;
}

/** Como en `esfinge.spec.ts`: sin elegir modo, la bienvenida tapa la ventana. */
test.beforeEach(async ({ request }) => {
  const r = await request.post(`${VENTANA}/api/ElegirModoLocal`, { data: [] });
  expect(r.ok(), await r.text()).toBe(true);
});

/**
 * Deja la bóveda abierta, venga de donde venga: la primera pasada la crea y las
 * demás la abren. Aquí solo la tocan estas dos pruebas.
 */
async function conLaBovedaAbierta(page: Page) {
  await seccion(page, "Bóveda").click();

  const crear = accion(page, "Crear la bóveda");
  const abrir = accion(page, "Abrir la bóveda");
  await expect(crear.or(abrir).or(page.locator("#boveda-buscar"))).toBeVisible({ timeout: 20_000 });

  if (await crear.isVisible()) {
    await page.locator("#boveda-maestra").fill(MAESTRA);
    await page.locator("#boveda-maestra-2").fill(MAESTRA);
    await crear.click();
    const clave = page.locator(".clave-recuperacion");
    await expect(clave).toBeVisible({ timeout: 20_000 });
    await page.getByText("La he apuntado en un sitio seguro").click();
    await accion(page, "Continuar").click();
  } else if (await abrir.isVisible()) {
    await page.locator("#boveda-llave").fill(MAESTRA);
    await abrir.click();
  }
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });
}

test("la clave de recuperación abre la bóveda", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto(VENTANA);
  await conLaBovedaAbierta(page);

  // Se genera una nueva en vez de usar la del principio, porque la del principio
  // solo se ve si esta pasada fue la que creó la bóveda. Y de paso se comprueba
  // lo que hace rotar: que la de antes deja de valer y sale otra.
  await page.getByRole("button", { name: "Contraseña maestra y clave de recuperación" }).click();
  await page.getByRole("button", { name: "Generar otra…" }).click();

  const clave = page.locator(".clave-recuperacion");
  await expect(clave).toBeVisible({ timeout: 20_000 });
  const recuperacion = (await clave.innerText()).trim();

  await page.getByText("La he apuntado en un sitio seguro").click();
  await accion(page, "Continuar").click();

  await accion(page, "Cerrar la bóveda").click();
  await expect(accion(page, "Abrir la bóveda")).toBeVisible();

  // **Y con la bóveda cerrada, quien haya olvidado la maestra tiene dónde mirar.**
  // Sin cuenta no hay asistente que recupere nada, así que lo único que puede hacer
  // la ventana es decir que la clave de recuperación se escribe en ese mismo campo
  // —que no se adivina— y qué pasa si tampoco se tiene (revisión de los textos).
  await accion(page, "¿Has olvidado la contraseña maestra?").click();
  const ayuda = page.locator(".contenido .nota", { hasText: "clave de recuperación" }).first();
  await expect(ayuda).toBeVisible();
  await expect(ayuda).toContainText("no hay forma de abrir esta bóveda");

  await page.locator("#boveda-llave").fill(recuperacion);
  await accion(page, "Abrir la bóveda").click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("una clave de recuperación con una errata se distingue de una que no abre", async ({ page }) => {
  // Sin vigilar la consola: aquí se piden dos aperturas que tienen que fallar, y
  // el navegador anota cada respuesta 400 como error suyo.
  await page.goto(VENTANA);
  await conLaBovedaAbierta(page);
  await accion(page, "Cerrar la bóveda").click();

  // La suma de control es la diferencia entre «te has equivocado al copiarla» y
  // «has perdido la bóveda», y se ve antes de gastar medio segundo derivando.
  await page.locator("#boveda-llave").fill("ESF-ABCD-EFGH-JKMN-PQRS-TVWX-YZ01-2345-6789");
  await accion(page, "Abrir la bóveda").click();
  await expect(page.locator(".error:visible")).toContainText("revísala");

  await page.locator("#boveda-llave").fill("esta no es la contraseña");
  await accion(page, "Abrir la bóveda").click();
  await expect(page.locator(".error:visible")).toContainText("no abre esta bóveda");
});
