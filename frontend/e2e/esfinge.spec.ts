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

test("avisa de la versión nueva, y se puede quitar de en medio", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Se pide desde Ajustes, que recorre el mismo camino que la comprobación del
  // arranque en la aplicación de verdad. Aquí no se comprueba sola al abrir: en
  // desarrollo eso saldría a la red en cada recarga.
  //
  // Y no se afirma que la banda no esté antes de pedirlo: el servidor de
  // desarrollo es uno solo para todas las pruebas y se acuerda de la novedad que
  // encontró la anterior, así que al recargar puede salir sola. Eso es correcto
  // en la aplicación; aquí solo haría la prueba dependiente del orden.
  await page.getByRole("tab", { name: "Ajustes" }).click();
  await page.getByRole("button", { name: "Buscar ahora" }).click();

  const banda = page.locator(".novedad");
  await expect(banda).toBeVisible({ timeout: 20_000 });
  await expect(banda).toContainText("9.9.9");

  // El aviso no puede confundirse con los avisos del propio trabajo, que llevan
  // la clase «aviso» y hablan de lo que se está cifrando.
  await expect(page.locator(".aviso")).toHaveCount(0);

  await banda.getByRole("button", { name: "Ahora no" }).click();
  await expect(banda).toHaveCount(0);

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("el interruptor de Ajustes se queda como se deja", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  await page.getByRole("tab", { name: "Ajustes" }).click();
  const interruptor = page.getByRole("checkbox", { name: /versión nueva/ });
  await expect(interruptor).toBeChecked();

  // Apagarlo es lo que corta la única salida a la red del programa, así que
  // tiene que sobrevivir a cerrar y abrir la ventana.
  await interruptor.uncheck();
  await page.reload();
  await page.getByRole("tab", { name: "Ajustes" }).click();
  await expect(page.getByRole("checkbox", { name: /versión nueva/ })).not.toBeChecked();

  // Y se deja como estaba, que el fichero de preferencias es de verdad y lo
  // comparten las demás pruebas.
  await page.getByRole("checkbox", { name: /versión nueva/ }).check();

  expect(errores, errores.join(' | ')).toEqual([]);
});

/**
 * ordenar manda una orden como la mandaría el menú del sistema.
 *
 * Se reintenta porque el flujo de eventos del servidor de desarrollo solo llega
 * a quien ya está conectado: una orden emitida en el instante entre cargar la
 * página y engancharse se pierde. En la aplicación de verdad no puede pasar
 * —para pulsar un menú la ventana ya tiene que estar abierta—, así que esto es
 * una cautela de la prueba, no un remiendo del programa.
 */
async function ordenar(page: Page, que: string) {
  await page.evaluate(
    (q) =>
      fetch("/api/Ordenar", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify([q]),
      }),
    que,
  );
}

test("el menú del sistema cambia de pantalla y edita el campo con el foco", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // El menú no se puede pulsar desde un navegador: lo dibuja el sistema. Lo que
  // sí se puede recorrer es el camino entero desde que Go manda la orden, que es
  // exactamente lo que hace el menú al pulsarlo.
  await expect
    .poll(async () => {
      await ordenar(page, "ir:generar");
      return page.getByRole("tab", { name: "Generar" }).getAttribute("aria-selected");
    }, { timeout: 15_000 })
    .toBe("true");

  await expect
    .poll(async () => {
      await ordenar(page, "ir:cifrar");
      return page.getByRole("tab", { name: "Cifrar", exact: true }).getAttribute("aria-selected");
    }, { timeout: 15_000 })
    .toBe("true");

  // Seleccionar todo tiene que actuar sobre el campo que tiene el foco, que es
  // lo que Go no puede saber y por eso la orden se resuelve en la interfaz.
  const campo = page.getByLabel("Qué quieres cifrar");
  await campo.fill("un secreto cualquiera");
  await campo.focus();

  await expect
    .poll(async () => {
      await ordenar(page, "editar:seleccionar-todo");
      return page.evaluate(() => {
        const e = document.activeElement as HTMLTextAreaElement | null;
        return e && e.selectionEnd !== null && e.selectionStart !== null
          ? e.selectionEnd - e.selectionStart
          : 0;
      });
    }, { timeout: 15_000 })
    .toBe("un secreto cualquiera".length);

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("un .esf de fichero abre descifrar en modo ficheros", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Es lo que hace macOS al hacer doble clic en un .esf con Esfinge ya abierta:
  // no llega como argumento, llega por un evento. Antes se emitía y nadie lo
  // escuchaba, así que la ventana se quedaba como estaba.
  await expect
    .poll(async () => {
      await page.evaluate(() =>
        fetch("/api/AlAbrirCon", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(["/tmp/credenciales.env.esf"]),
        }),
      );
      return page.getByRole("tab", { name: "Descifrar" }).getAttribute("aria-selected");
    }, { timeout: 15_000 })
    .toBe("true");

  await expect(page.locator(".lista-ficheros li")).toContainText("credenciales.env.esf");

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("un .esf que lleva un texto abre descifrar en modo texto, con la línea puesta", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Primero se fabrica uno de verdad: se cifra un texto y se guarda, que es
  // exactamente como aparece un .esf de esta clase en el disco de alguien.
  await page.getByLabel("Qué quieres cifrar").fill(SECRETO);
  await page.locator("#clave").fill(CLAVE);
  await page.getByRole("button", { name: "Cifrar", exact: true }).click();

  const resultado = page.locator(".resultado");
  await expect(resultado).toBeVisible({ timeout: 20_000 });
  const cifrado = (await resultado.innerText()).trim();

  const donde = await page.evaluate(async (contenido) => {
    const r = await fetch("/api/GuardarTexto", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(["secreto.esf", contenido]),
    });
    return (await r.json()) as string;
  }, cifrado);
  expect(donde).toBeTruthy();

  // Y ahora se abre como lo abriría el Finder. Lo que tiene que salir no es la
  // pantalla de ficheros con otro fichero al lado: es el texto, para poder verlo.
  await expect
    .poll(async () => {
      await page.evaluate(async (ruta) => {
        await fetch("/api/AlAbrirCon", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify([ruta]),
        });
      }, donde);
      return page.getByRole("tab", { name: "Texto" }).getAttribute("aria-selected");
    }, { timeout: 15_000 })
    .toBe("true");

  await expect(page.getByLabel("El texto cifrado")).toHaveValue(cifrado);

  // Y se descifra desde ahí, que es lo que se quería hacer al abrirlo.
  await page.locator("#clave").fill(CLAVE);
  await page.getByRole("button", { name: "Descifrar", exact: true }).click();
  await expect(page.locator(".resultado")).toHaveText(SECRETO, { timeout: 20_000 });

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("sin el vidrio del sistema la ventana se pinta como siempre", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // El servidor de desarrollo no da vidrio, igual que Linux o un Windows sin
  // Mica. Ahí el fondo lo tiene que seguir pintando el CSS de siempre: un body
  // transparente sin nada detrás no enseña el escritorio, enseña un agujero.
  await expect(page.locator(".barra")).toBeVisible();
  await expect(page.locator("html")).not.toHaveAttribute("data-vidrio", "si");

  const fondo = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
  expect(fondo).not.toBe("rgba(0, 0, 0, 0)");
  expect(fondo).not.toBe("transparent");

  expect(errores, errores.join(' | ')).toEqual([]);
});
