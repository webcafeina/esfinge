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
    const clave = (p: typeof page) => p.locator("input[type=password]:visible");

    const foto = (n: string) => page.screenshot({ path: `${donde}/${n}-${tema}.png` });

    await page.setViewportSize({ width: 980, height: 680 });
    await page.goto("/");

    // Con el servidor de desarrollo recién arrancado, lo primero es la bienvenida
    // de la cuenta. Se retrata y se elige «En este ordenador», que es lo que
    // recorre el resto.
    const bienvenida = page.getByRole("button", { name: "Usar en este ordenador" });
    // Se espera a que haya algo: preguntar en el acto si se ve, con la página aún
    // pintándose, decía que no y el recorrido se quedaba en la bienvenida.
    await bienvenida.or(page.getByLabel("Qué quieres cifrar")).first().waitFor();
    if (await bienvenida.isVisible().catch(() => false)) {
      await foto("00-bienvenida");
      await bienvenida.click();
    }

    // El rótulo de «Modo desarrollo» es cierto aquí y mentira en la portada del
    // repositorio, que es donde acaban estas capturas.
    if (process.env.CAPTURAS_SIN_DEV) {
      await page.addStyleTag({ content: ".herramientas .aparte{visibility:hidden}" });
    }

    await page.getByLabel("Qué quieres cifrar").fill("postgres://usuario:secreto@host/basededatos");
    await clave(page).fill("caballo grapa batería correcto");
    await page.waitForTimeout(500);
    await foto("1-cifrar");

    await page.locator(".contenido").getByRole("button", { name: "Cifrar", exact: true }).click();
    await page.waitForSelector(".resultado", { timeout: 20_000 });
    await page.waitForTimeout(600);
    await foto("2-resultado");

    await page.getByRole("tab", { name: "Ficheros" }).click();
    await page.locator(".soltar").click();
    await page.waitForTimeout(700);
    await foto("3-ficheros");

    await page.locator(".lateral").getByRole("button", { name: "Generar", exact: true }).click();
    await page.waitForSelector(".resultado", { timeout: 20_000 });
    await page.waitForTimeout(400);
    await foto("4-generar");

    await page.locator(".lateral").getByRole("button", { name: "Historial", exact: true }).click();
    await page.waitForTimeout(500);
    await foto("5-historial");
  });

  /**
   * La pantalla de desbloquear con la huella (fase C, ADR 0044).
   *
   * Va aparte del recorrido porque necesita una bóveda, y sobre todo porque **hay
   * que mirarla**: el latido y el halo no los comprueba ninguna aserción, y esta
   * pantalla se rehízo justo por eso —el cliente vio un botón donde tenía que
   * haber una huella—.
   */
  test("desbloquear con la huella", async ({ page }, info) => {
    const tema = info.project.name;
    const donde = process.env.CAPTURAS_EN ?? "capturas";
    const foto = (n: string) => page.screenshot({ path: `${donde}/${n}-${tema}.png` });
    const maestra = "caballo grapa batería correcto";
    const dentro = page.locator(".contenido");
    const boton = (n: string) => dentro.getByRole("button", { name: n, exact: true });

    await page.setViewportSize({ width: 980, height: 680 });
    await page.goto("/");

    const bienvenida = page.getByRole("button", { name: "Usar en este ordenador" });
    await bienvenida.or(page.getByLabel("Qué quieres cifrar")).first().waitFor();
    if (await bienvenida.isVisible().catch(() => false)) await bienvenida.click();

    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    // **Se espera a que haya algo antes de preguntar cuál hay.** `isVisible` no
    // espera: con la pantalla aún pintándose contesta que no a las dos, no se crea
    // ni se abre nada, y lo que falla es la línea de después. Es la misma trampa
    // que ya está escrita arriba, en la bienvenida del recorrido.
    await boton("Crear la bóveda")
      .or(boton("Abrir la bóveda"))
      .or(page.locator("#boveda-buscar"))
      .first()
      .waitFor({ timeout: 20_000 });
    if (await boton("Crear la bóveda").isVisible().catch(() => false)) {
      await page.locator("#boveda-maestra").fill(maestra);
      await page.locator("#boveda-maestra-2").fill(maestra);
      await boton("Crear la bóveda").click();
      await page.getByText("La he apuntado en un sitio seguro").click();
      await boton("Continuar").click();
    } else if (await boton("Abrir la bóveda").isVisible().catch(() => false)) {
      await page.locator("#boveda-llave").fill(maestra);
      await boton("Abrir la bóveda").click();
    }
    await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });

    // La bóveda lo ofrece sola la primera vez, y eso es lo primero que se ve.
    await page.locator("#sugerencia-activar").waitFor({ state: "visible", timeout: 20_000 });
    await page.waitForTimeout(400);
    await foto("6a-sugerencia");
    await page.locator("#sugerencia-activar").screenshot({ path: `${donde}/6b-sugerencia-cerca-${tema}.png`, scale: "css" });
    await page.locator(".sugerencia").screenshot({ path: `${donde}/6c-sugerencia-tarjeta-${tema}.png`, scale: "css" });

    await page.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
    const interruptor = page.locator("#desbloqueo-del-sistema");
    await interruptor.waitFor({ state: "visible", timeout: 20_000 });
    if (!(await interruptor.isChecked())) {
      await interruptor.click();
      await page.waitForTimeout(1200);
    }
    await foto("6-ajustes-desbloqueo");

    // **En reposo**: el llavero dice que no, así que la pantalla se queda con la
    // huella quieta. Si dijera que sí abriría sola y no habría nada que retratar.
    await page.request.get("/api/_llavero?dice=no");
    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    await boton("Cerrar la bóveda").click();
    await page.locator("#boveda-con-el-sistema").waitFor({ state: "visible", timeout: 20_000 });
    await page.waitForTimeout(500);
    await foto("7-huella-reposo");
    // Y de cerca, que es donde se juzga el trazo: a tamaño de pantalla completa
    // esto son setenta píxeles y no se ve si se lee como una huella o como un
    // arcoíris. La primera versión era lo segundo.
    await page.locator("#boveda-con-el-sistema").screenshot({ path: `${donde}/7b-huella-cerca-${tema}.png`, scale: "css" });

    // **Esperando**: con el diálogo del sistema delante. Aquí no hay diálogo, así
    // que se congela la clase a mano para poder ver el latido a medio camino.
    await page.locator("#boveda-con-el-sistema").evaluate((b) => b.classList.add("esperando"));
    await page.waitForTimeout(900);
    await foto("8-huella-esperando");

    // Se deja como estaba.
    await page.request.get("/api/_llavero?dice=si");
    await page.locator("#boveda-llave").fill(maestra);
    await boton("Abrir la bóveda").click();
    await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });
    await page.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
    await page.locator("#desbloqueo-del-sistema").click();
    await page.waitForTimeout(1200);
  });
});
