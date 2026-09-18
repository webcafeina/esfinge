import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

/**
 * La cuenta, de la ventana de un equipo a la de otro (ADR 0035).
 *
 * Dos ventanas a la vez —5174 y 5175—, cada una con su Go y su carpeta de
 * configuración recién hecha, y **el servidor de cuentas de verdad** levantado en
 * local (8792): es la tubería entera, de la bienvenida del primer equipo a la
 * lista de la bóveda del segundo. Las capturas de las pantallas nuevas salen con
 * CAPTURAS=1, en los dos temas, para poder mirarlas.
 *
 * Solo en el proyecto «claro»: los dos equipos se estrenan una vez por tanda, y el
 * tema oscuro de estas pantallas se mira en las capturas.
 */

const SERVIDOR = "http://127.0.0.1:8792";
const A = "http://127.0.0.1:5174";
const B = "http://127.0.0.1:5175";
const MAESTRA = "una maestra larga para probar la cuenta en la ventana";

/**
 * Apunta los errores de la página y **las respuestas fallidas con su dirección**:
 * el «Failed to load resource» de la consola no dice cuál ha sido, y aquí hay una
 * que falla a propósito —la contraseña débil—. Se aceptan solo las esperadas.
 */
function vigilarConsola(page: Page): string[] {
  const errores: string[] = [];
  page.on("console", (m) => {
    if (m.type() === "error" && !m.text().startsWith("Failed to load resource")) errores.push(m.text());
  });
  page.on("pageerror", (e) => errores.push(String(e)));
  page.on("response", (r) => {
    if (r.status() >= 400) errores.push(`${r.status()} ${new URL(r.url()).pathname}`);
  });
  return errores;
}

async function codigo(request: APIRequestContext, correo: string): Promise<string> {
  const r = await request.get(`${SERVIDOR}/_pruebas/buzon?correo=${encodeURIComponent(correo)}`);
  const { mensajes } = (await r.json()) as { mensajes: { cuerpo: string }[] };
  const m = /^\s+(\d{6})$/m.exec(mensajes[0]?.cuerpo ?? "");
  if (!m) throw new Error(`No ha llegado ningún código a ${correo}`);
  return m[1];
}

/** Captura en los dos temas, solo si se piden. */
async function retratar(page: Page, nombre: string) {
  if (!process.env.CAPTURAS) return;
  const donde = process.env.CAPTURAS_EN ?? "capturas";
  for (const tema of ["light", "dark"] as const) {
    await page.emulateMedia({ colorScheme: tema });
    // Los botones cambian de color con una transición: sin esperar, la captura
    // salía a mitad y los botones de oscuro parecían blancos con letra blanca.
    await page.waitForTimeout(300);
    await page.screenshot({ path: `${donde}/${nombre}-${tema === "light" ? "claro" : "oscuro"}.png` });
  }
  await page.emulateMedia({ colorScheme: "light" });
}

const accion = (page: Page, nombre: string) =>
  page.getByRole("button", { name: nombre, exact: true }).and(page.locator("button:visible"));

test("de la bienvenida de un equipo a la bóveda del otro", async ({ browser, request }) => {
  test.skip(test.info().project.name !== "claro", "Los dos equipos se estrenan una vez por tanda");
  test.setTimeout(120_000);
  const correo = `ventana-${Date.now()}@ejemplo.com`;

  // ---------------------------------------------------------------- equipo A
  const a = await (await browser.newContext({ viewport: { width: 980, height: 680 } })).newPage();
  const erroresA = vigilarConsola(a);
  await a.goto(A);

  // Sin nada en el equipo, la bienvenida: las dos opciones con lo bueno y lo malo.
  await expect(a.getByRole("heading", { name: "Te damos la bienvenida a Esfinge" })).toBeVisible();
  await expect(a.getByRole("heading", { name: "En este ordenador" })).toBeVisible();
  await expect(a.getByRole("heading", { name: "Con cuenta" })).toBeVisible();
  await expect(a.getByText("el servidor no puede leerla")).toBeVisible();
  // Y la ventana no tiene todavía barra lateral: se elige antes de entrar.
  await expect(a.locator(".lateral")).toHaveCount(0);
  await retratar(a, "bienvenida");

  await accion(a, "Crear una cuenta").click();
  await a.locator("#cuenta-correo").fill(correo);
  await retratar(a, "cuenta-correo");
  await accion(a, "Mandarme el código").click();

  await expect(a.locator("#cuenta-codigo")).toBeVisible({ timeout: 20_000 });
  await a.locator("#cuenta-codigo").fill(await codigo(request, correo));
  // Con cuenta, una contraseña débil no vale, y lo dice Go con su frase.
  await a.locator("#cuenta-maestra").fill("corta");
  await a.locator("#cuenta-maestra-2").fill("corta");
  await accion(a, "Crear la cuenta").click();
  await expect(a.locator(".error")).toContainText("al menos «Buena»");
  // Ése es el único fallo que se espera en todo el recorrido.
  expect(erroresA).toEqual(["400 /api/TerminarRegistro"]);
  erroresA.length = 0;
  await a.locator("#cuenta-maestra").fill(MAESTRA);
  await a.locator("#cuenta-maestra-2").fill(MAESTRA);
  await retratar(a, "cuenta-codigo");
  await accion(a, "Crear la cuenta").click();

  // La clave de recuperación de la bóveda nueva, una sola vez.
  const clave = a.locator(".clave-recuperacion");
  await expect(clave).toBeVisible({ timeout: 20_000 });
  await a.getByText("La he apuntado en un sitio seguro").click();
  await accion(a, "Continuar").click();

  // Dentro: la bóveda abierta, en la sección de la bóveda, y sincronizada.
  await expect(a.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });
  await expect(a.locator(".linea-sincro")).toContainText("Sincronizada", { timeout: 20_000 });

  await accion(a, "Nueva").click();
  await a.locator("#boveda-titulo").fill("Banco de la cuenta");
  await a.locator("#boveda-secreto").fill("s3cr3t0");
  await accion(a, "Guardar").click();
  await expect(a.locator(".lista-boveda").getByRole("button", { name: "Banco de la cuenta" })).toBeVisible();
  // Se cierra enseguida: lo pendiente sube al cerrar, sin esperar a nadie.
  await accion(a, "Cerrar la bóveda").click();
  await expect(accion(a, "Abrir la bóveda")).toBeVisible();
  expect(erroresA, erroresA.join(" | ")).toEqual([]);

  // ---------------------------------------------------------------- equipo B
  const b = await (await browser.newContext({ viewport: { width: 980, height: 680 } })).newPage();
  const erroresB = vigilarConsola(b);
  await b.goto(B);
  await accion(b, "Ya tengo cuenta").click();
  await b.locator("#entrar-correo").fill(correo);
  await b.locator("#entrar-maestra").fill(MAESTRA);
  await retratar(b, "entrar");
  await accion(b, "Entrar").click();
  // Primero que salga el campo —es cuando el código ya se ha mandado— y después
  // el buzón. Leyéndolo antes, se cogía el código del alta, que era el último.
  await expect(b.locator("#entrar-codigo")).toBeVisible({ timeout: 20_000 });
  await b.locator("#entrar-codigo").fill(await codigo(request, correo));
  await retratar(b, "entrar-codigo");
  await accion(b, "Confirmar").click();

  // La bóveda de la cuenta, abierta en el otro equipo con lo que guardó el primero.
  await expect(b.locator(".lista-boveda").getByRole("button", { name: "Banco de la cuenta" })).toBeVisible({
    timeout: 20_000,
  });
  await expect(b.locator(".linea-sincro")).toContainText("Sincronizada", { timeout: 20_000 });
  await retratar(b, "boveda-sincronizada");

  // Y Ajustes cuenta en qué cuenta está este equipo.
  await b.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
  await expect(b.getByRole("heading", { name: "Cuenta y sincronización" })).toBeVisible();
  await expect(b.getByText(correo)).toBeVisible();
  await retratar(b, "ajustes-cuenta");

  // Y se puede cambiar de idea: dejar la cuenta en este equipo, con la contraseña.
  await accion(b, "Dejar la cuenta en este equipo…").click();
  await b.locator("#salir-maestra").fill(MAESTRA);
  await retratar(b, "ajustes-salir");
  await accion(b, "Dejar la cuenta en este equipo").click();
  await expect(accion(b, "Crear una cuenta")).toBeVisible();
  await expect(b.getByText("Tu bóveda está solo en este ordenador.")).toBeVisible();
  expect(erroresB, erroresB.join(" | ")).toEqual([]);
});
