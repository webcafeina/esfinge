import { test } from "@playwright/test";

/**
 * No comprueba nada: recorre la interfaz sacando capturas para poder mirarla
 * antes de compilar. Se ejecuta a propósito, con
 * «pnpm exec playwright test capturas --project=claro».
 *
 * Vive aquí y no en un script suelto para aprovechar que la configuración ya
 * levanta el Go de verdad y Vite: así las capturas salen de la aplicación
 * funcionando, no de una maqueta.
 */
test.describe("Capturas", () => {
  test.skip(!process.env.CAPTURAS, "Solo cuando se piden con CAPTURAS=1");

  test("recorrido", async ({ page }, info) => {
    const tema = info.project.name;
    const donde = process.env.CAPTURAS_EN ?? "capturas";
    const foto = (n: string) => page.screenshot({ path: `${donde}/${n}-${tema}.png` });

    await page.setViewportSize({ width: 840, height: 640 });
    await page.goto("/");

    // El rótulo de «Modo desarrollo» es cierto aquí y mentira en la portada del
    // repositorio, que es donde acaban estas capturas.
    if (process.env.CAPTURAS_SIN_DEV) {
      await page.addStyleTag({ content: ".pie span:last-child{visibility:hidden}" });
    }

    await page.getByLabel("Qué quieres cifrar").fill("postgres://usuario:secreto@host/basededatos");
    await page.locator("#clave").fill("caballo grapa batería correcto");
    await page.waitForTimeout(500);
    await foto("1-cifrar");

    await page.getByRole("button", { name: "Cifrar", exact: true }).click();
    await page.waitForSelector(".resultado", { timeout: 20_000 });
    await page.waitForTimeout(600);
    await foto("2-resultado");

    await page.getByRole("tab", { name: "Ficheros" }).click();
    await page.locator(".soltar").click();
    await page.waitForTimeout(700);
    await foto("3-ficheros");

    await page.getByRole("tab", { name: "Generar" }).click();
    await page.waitForSelector(".resultado", { timeout: 20_000 });
    await page.waitForTimeout(400);
    await foto("4-generar");

    await page.getByRole("tab", { name: "Historial" }).click();
    await page.waitForTimeout(500);
    await foto("5-historial");
  });
});
