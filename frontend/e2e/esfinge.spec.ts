import { expect, test, type Page } from "@playwright/test";

const SECRETO = "postgres://usuario:secreto@host/db";
const CLAVE = "una clave larga de prueba";

/** Ningún error de la consola pasa desapercibido. */
function vigilarConsola(page: Page): string[] {
  const errores: string[] = [];
  page.on("console", (m) => m.type() === "error" && errores.push(m.text()));
  page.on("pageerror", (e) => errores.push(String(e)));
  return errores;
}

test("cifra un texto y lo vuelve a abrir", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  await page.getByLabel("Qué quieres cifrar").fill(SECRETO);
  await page.locator("#clave").fill(CLAVE);
  await page.getByRole("button", { name: "Cifrar", exact: true }).click();

  const resultado = page.locator(".resultado");
  await expect(resultado).toBeVisible({ timeout: 20_000 });
  const cifrado = (await resultado.innerText()).trim();
  expect(cifrado.startsWith("ESF1.")).toBe(true);

  // Cifrar avisa de que sin la clave no hay vuelta atrás. Es lo único que hay
  // que entender de esta herramienta, y tiene que estar delante cuando toca
  // decidir si se guarda o se cierra. Uno solo: el del formulario se retira al
  // aparecer el resultado, para no decir dos veces lo mismo en la misma pantalla.
  await expect(page.locator(".aviso")).toHaveCount(1);
  await expect(page.locator(".aviso")).toContainText("no lo abre nadie");

  await page.getByRole("tab", { name: "Descifrar" }).click();
  await page.getByLabel("El texto cifrado").fill(cifrado);
  await page.locator("#clave").fill(CLAVE);
  await page.getByRole("button", { name: "Descifrar", exact: true }).click();

  await expect(page.locator(".resultado")).toHaveText(SECRETO, { timeout: 20_000 });
  expect(errores, errores.join(' | ')).toEqual([]);
});

test("con la clave equivocada lo dice, y no revienta", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("tab", { name: "Descifrar" }).click();
  await page.getByLabel("El texto cifrado").fill("ESF1.esto-no-es-un-contenedor");
  await page.locator("#clave").fill("cualquiera");
  await page.getByRole("button", { name: "Descifrar", exact: true }).click();

  await expect(page.locator(".error")).toBeVisible({ timeout: 20_000 });
  await expect(page.locator(".error")).toContainText("Esfinge");
});

test("el botón de cifrar no se puede pulsar sin lo que hace falta", async ({ page }) => {
  await page.goto("/");
  const boton = page.getByRole("button", { name: "Cifrar", exact: true });

  await expect(boton).toBeDisabled();

  await page.getByLabel("Qué quieres cifrar").fill("algo");
  await expect(boton).toBeDisabled(); // todavía falta la clave

  await page.locator("#clave").fill("una clave");
  await expect(boton).toBeEnabled();
});

test("el medidor valora la clave mientras se teclea", async ({ page }) => {
  await page.goto("/");

  await page.locator("#clave").fill("1234");
  await expect(page.locator(".medidor")).toHaveAttribute("data-nivel", "0", { timeout: 10_000 });
  await expect(page.getByText("Muy débil")).toBeVisible();

  await page.locator("#clave").fill("caballo grapa batería correcto");
  await expect(page.locator(".medidor")).not.toHaveAttribute("data-nivel", "0");
});

test("genera contraseñas y avisa de las que rompen una URL", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("tab", { name: "Generar" }).click();

  const resultado = page.locator(".resultado");
  await expect(resultado).toBeVisible({ timeout: 20_000 });

  const hex = (await resultado.innerText()).trim();
  expect(hex).toMatch(/^[0-9a-f]+$/);
  // La longitud se pide en caracteres, y sale la que se pide.
  expect(hex.length).toBe(32);
  await expect(page.locator(".nota", { hasText: "Segura dentro de una URL" })).toBeVisible();

  // Otra distinta.
  await page.getByRole("button", { name: "Generar otra" }).click();
  await expect(resultado).not.toHaveText(hex);

  // Y el alfabeto con símbolos avisa, que es el que rompe cadenas de conexión.
  await page.getByRole("tab", { name: "Con símbolos" }).click();
  await expect(page.locator(".aviso")).toContainText("URL");
});

test("cifra una tanda de ficheros", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("tab", { name: "Ficheros" }).click();
  await page.locator(".soltar").click(); // el diálogo del sistema

  await expect(page.locator(".lista-ficheros li").first()).toBeVisible({ timeout: 10_000 });
  const cuantos = await page.locator(".lista-ficheros li").count();
  expect(cuantos).toBeGreaterThan(0);

  await page.locator("#clave").fill(CLAVE);
  await page.getByRole("button", { name: "Cifrar", exact: true }).click();

  await expect(page.locator(".exito")).toBeVisible({ timeout: 30_000 });
  await expect(page.locator(".exito")).toContainText("listo");
});

test("el historial enseña lo hecho y se puede vaciar", async ({ page }) => {
  await page.goto("/");

  await page.getByLabel("Qué quieres cifrar").fill("algo que dejará rastro");
  await page.locator("#clave").fill(CLAVE);
  await page.getByRole("button", { name: "Cifrar", exact: true }).click();
  await expect(page.locator(".resultado")).toBeVisible({ timeout: 20_000 });

  await page.getByRole("tab", { name: "Historial" }).click();
  await expect(page.locator(".historial li").first()).toBeVisible();

  // Y dice dónde vive, para que no haya que fiarse de la palabra de nadie.
  await expect(page.getByText("Se guarda en")).toBeVisible();

  await page.getByRole("button", { name: "Vaciar historial" }).click();
  await expect(page.getByText("Todavía no has hecho nada.")).toBeVisible();
});
