import { expect, test, type Page } from "@playwright/test";

const SECRETO = "postgres://usuario:secreto@host/db";
const CLAVE = "una clave larga de prueba";

/**
 * La sección, que desde la estructura de macOS es una fila de la barra lateral y
 * ya no una pestaña.
 *
 * Se acota a la barra lateral a propósito: «Cifrar» es a la vez el nombre de una
 * sección y el del botón que cifra, y sin acotar el selector encuentra los dos.
 */
function seccion(page: Page, nombre: string) {
  return page.locator(".lateral").getByRole("button", { name: nombre, exact: true });
}

/**
 * El botón de acción de la pantalla, acotado al contenido por la misma razón:
 * «Cifrar» nombra la sección y el botón que cifra.
 */
function accion(page: Page, nombre: string) {
  return page.locator(".contenido").getByRole("button", { name: nombre, exact: true }).and(
    page.locator("button:visible"),
  );
}

/**
 * El campo de la clave de la pantalla que se está viendo.
 *
 * Cifrar y descifrar están montadas a la vez —para que cambiar de sección no
 * borre lo escrito— así que cada una tiene su propio campo y hay que quedarse
 * con el visible. Buscarlo por identificador fijo encontraría los dos.
 */
function clave(page: Page) {
  return page.locator("input[type=password]:visible");
}

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
  await clave(page).fill(CLAVE);
  await accion(page, "Cifrar").click();

  const resultado = page.locator(".resultado:visible");
  await expect(resultado).toBeVisible({ timeout: 20_000 });
  const cifrado = (await resultado.innerText()).trim();
  expect(cifrado.startsWith("ESF1.")).toBe(true);

  // Cifrar avisa de que sin la clave no hay vuelta atrás. Es lo único que hay
  // que entender de esta herramienta, y tiene que estar delante cuando toca
  // decidir si se guarda o se cierra. Uno solo: el del formulario se retira al
  // aparecer el resultado, para no decir dos veces lo mismo en la misma pantalla.
  await expect(page.locator(".aviso:visible")).toHaveCount(1);
  await expect(page.locator(".aviso:visible")).toContainText("no lo abre nadie");

  await seccion(page, "Descifrar").click();
  await page.getByLabel("El texto cifrado").fill(cifrado);
  await clave(page).fill(CLAVE);
  await accion(page, "Descifrar").click();

  await expect(page.locator(".resultado:visible")).toHaveText(SECRETO, { timeout: 20_000 });
  expect(errores, errores.join(' | ')).toEqual([]);
});

test("con la clave equivocada lo dice, y no revienta", async ({ page }) => {
  await page.goto("/");

  await seccion(page, "Descifrar").click();
  await page.getByLabel("El texto cifrado").fill("ESF1.esto-no-es-un-contenedor");
  await clave(page).fill("cualquiera");
  await accion(page, "Descifrar").click();

  await expect(page.locator(".error:visible")).toBeVisible({ timeout: 20_000 });
  await expect(page.locator(".error:visible")).toContainText("Esfinge");
});

test("el botón de cifrar no se puede pulsar sin lo que hace falta", async ({ page }) => {
  await page.goto("/");
  const boton = accion(page, "Cifrar");

  await expect(boton).toBeDisabled();

  await page.getByLabel("Qué quieres cifrar").fill("algo");
  await expect(boton).toBeDisabled(); // todavía falta la clave

  await clave(page).fill("una clave");
  await expect(boton).toBeEnabled();
});

test("el medidor valora la clave mientras se teclea", async ({ page }) => {
  await page.goto("/");

  await clave(page).fill("1234");
  await expect(page.locator(".medidor:visible")).toHaveAttribute("data-nivel", "0", { timeout: 10_000 });
  await expect(page.getByText("Muy débil")).toBeVisible();

  await clave(page).fill("caballo grapa batería correcto");
  await expect(page.locator(".medidor:visible")).not.toHaveAttribute("data-nivel", "0");
});

test("genera contraseñas y avisa de las que rompen una URL", async ({ page }) => {
  await page.goto("/");
  await seccion(page, "Generar").click();

  const resultado = page.locator(".resultado:visible");
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
  await expect(page.locator(".aviso:visible")).toContainText("URL");
});

test("cifra una tanda de ficheros", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("tab", { name: "Ficheros" }).click();
  await page.locator(".soltar:visible").click(); // el diálogo del sistema

  await expect(page.locator(".lista-ficheros li:visible").first()).toBeVisible({ timeout: 10_000 });
  const cuantos = await page.locator(".lista-ficheros li:visible").count();
  expect(cuantos).toBeGreaterThan(0);

  await clave(page).fill(CLAVE);
  await accion(page, "Cifrar").click();

  await expect(page.locator(".exito:visible")).toBeVisible({ timeout: 30_000 });
  await expect(page.locator(".exito:visible")).toContainText("listo");
});

test("el historial enseña lo hecho y se puede vaciar", async ({ page }) => {
  await page.goto("/");

  await page.getByLabel("Qué quieres cifrar").fill("algo que dejará rastro");
  await clave(page).fill(CLAVE);
  await accion(page, "Cifrar").click();
  await expect(page.locator(".resultado:visible")).toBeVisible({ timeout: 20_000 });

  await seccion(page, "Historial").click();
  await expect(page.locator(".historial li:visible").first()).toBeVisible();

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
  await seccion(page, "Ajustes").click();
  await page.getByRole("button", { name: "Buscar ahora" }).click();

  const banda = page.locator(".novedad");
  await expect(banda).toBeVisible({ timeout: 20_000 });
  await expect(banda).toContainText("9.9.9");

  // El aviso no puede confundirse con los avisos del propio trabajo, que llevan
  // la clase «aviso» y hablan de lo que se está cifrando.
  await expect(page.locator(".aviso:visible")).toHaveCount(0);

  await banda.getByRole("button", { name: "Ahora no" }).click();
  await expect(banda).toHaveCount(0);

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("el interruptor de Ajustes se queda como se deja", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // El fichero de preferencias del servidor de desarrollo es de verdad y dura
  // entre tandas, así que la prueba **se prepara su propio punto de partida** en
  // vez de darlo por hecho: una tanda interrumpida a mitad lo deja apagado y a
  // partir de ahí fallaría siempre.
  await page.evaluate(() =>
    fetch("/api/GuardarPreferencias", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify([{ buscarActualizaciones: true }]),
    }),
  );
  await page.reload();

  await seccion(page, "Ajustes").click();
  const interruptor = page.getByRole("checkbox", { name: /versión nueva/ });
  await expect(interruptor).toBeChecked();

  // Apagarlo es lo que corta la única salida a la red del programa, así que
  // tiene que sobrevivir a cerrar y abrir la ventana.
  await interruptor.uncheck();
  await page.reload();
  await seccion(page, "Ajustes").click();
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

/**
 * pegarDesdeElMenu es lo mismo para «Pegar», que va por su propia puerta porque
 * el texto lo lee Go: el navegador no puede mirar el portapapeles del sistema.
 */
async function pegarDesdeElMenu(page: Page, texto: string) {
  await page.evaluate(
    (t) =>
      fetch("/api/OrdenarPegar", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify([t]),
      }),
    texto,
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
      return seccion(page, "Generar").getAttribute("aria-current");
    }, { timeout: 15_000 })
    .toBe("page");

  await expect
    .poll(async () => {
      await ordenar(page, "ir:cifrar");
      return seccion(page, "Cifrar").getAttribute("aria-current");
    }, { timeout: 15_000 })
    .toBe("page");

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
      return seccion(page, "Descifrar").getAttribute("aria-current");
    }, { timeout: 15_000 })
    .toBe("page");

  await expect(page.locator(".lista-ficheros li:visible")).toContainText("credenciales.env.esf");

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("un .esf que lleva un texto abre descifrar en modo texto, con la línea puesta", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Primero se fabrica uno de verdad: se cifra un texto y se guarda, que es
  // exactamente como aparece un .esf de esta clase en el disco de alguien.
  await page.getByLabel("Qué quieres cifrar").fill(SECRETO);
  await clave(page).fill(CLAVE);
  await accion(page, "Cifrar").click();

  const resultado = page.locator(".resultado:visible");
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
  await clave(page).fill(CLAVE);
  await accion(page, "Descifrar").click();
  await expect(page.locator(".resultado:visible")).toHaveText(SECRETO, { timeout: 20_000 });

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("sin el vidrio del sistema la ventana se pinta como siempre", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // El servidor de desarrollo no da vidrio, igual que Linux o un Windows sin
  // Mica. Ahí el fondo lo tiene que seguir pintando el CSS de siempre: un body
  // transparente sin nada detrás no enseña el escritorio, enseña un agujero.
  await expect(page.locator(".lateral")).toBeVisible();
  await expect(page.locator("html")).not.toHaveAttribute("data-vidrio", "si");

  const fondo = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
  expect(fondo).not.toBe("rgba(0, 0, 0, 0)");
  expect(fondo).not.toBe("transparent");

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("con vidrio, nada se pinta por delante del material del sistema", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await expect(page.locator(".lateral")).toBeVisible();

  // El servidor de desarrollo nunca da vidrio, así que se pone el atributo a
  // mano: lo que se prueba es el CSS que cuelga de él, no quién lo pone.
  await page.evaluate(() => document.documentElement.setAttribute("data-vidrio", "si"));

  // **La barra lateral no lleva fondo propio.** En macOS el material *es* el
  // fondo de la barra, y cualquier capa por delante lo apaga: ése fue el fallo
  // que se arrastró tres versiones, con un tinte que se bajó dos veces sin que
  // se notara nunca. Si alguien vuelve a poner un color aquí, que falle esto.
  const barra = await page.evaluate(
    () => getComputedStyle(document.querySelector(".lateral")!).backgroundColor,
  );
  expect(barra).toBe("rgba(0, 0, 0, 0)");

  // La columna de trabajo, al revés: opaca siempre. Ahí se lee y se teclea, y un
  // fondo que cambia con lo que haya detrás de la ventana no sirve.
  const zona = await page.evaluate(
    () => getComputedStyle(document.querySelector(".zona")!).backgroundColor,
  );
  expect(zona).not.toBe("rgba(0, 0, 0, 0)");
  expect(zona).not.toBe("transparent");

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("la fila activa de la barra lateral no cambia al pasar el ratón", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // La fila activa ya dice lo que tiene que decir con su color de selección;
  // pintarle otro encima al pasar por encima es decir dos cosas a la vez.
  //
  // Esto tiene prueba porque la causa era un empate de especificidad con la
  // regla general de «button:hover», y esos empates vuelven solos en cuanto
  // alguien añade una regla más abajo en el fichero.
  const activa = seccion(page, "Cifrar");
  const antes = await activa.evaluate((e) => getComputedStyle(e).backgroundColor);

  await activa.hover();
  await page.waitForTimeout(250);
  const despues = await activa.evaluate((e) => getComputedStyle(e).backgroundColor);
  expect(despues).toBe(antes);

  // Y en una que no está activa sí tiene que notarse, o el resaltado no existe.
  const otra = seccion(page, "Generar");
  const otraAntes = await otra.evaluate((e) => getComputedStyle(e).backgroundColor);
  await otra.hover();
  await page.waitForTimeout(250);
  const otraDespues = await otra.evaluate((e) => getComputedStyle(e).backgroundColor);
  expect(otraDespues).not.toBe(otraAntes);

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("la contraseña generada se puede usar como clave, avisando de que no está guardada", async ({
  page,
}) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  await seccion(page, "Generar").click();

  // Se espera a que la contraseña deje de cambiar antes de leerla. En desarrollo
  // React monta los efectos dos veces —StrictMode—, así que el generador puede
  // producir una y sustituirla un instante después; leer sin más pilla a veces
  // la primera y compara contra la segunda. En la aplicación empaquetada esto no
  // pasa, pero la prueba corre en desarrollo.
  let generada = "";
  await expect
    .poll(async () => {
      const ahora = (await page.locator(".resultado:visible").innerText()).trim();
      const estable = ahora !== "" && ahora === generada;
      generada = ahora;
      return estable;
    }, { timeout: 15_000 })
    .toBe(true);
  expect(generada.length).toBeGreaterThan(16);

  await accion(page, "Usar como clave").click();

  await expect(seccion(page, "Cifrar")).toHaveAttribute("aria-current", "page");
  await expect(clave(page)).toHaveValue(generada);

  // Una clave recién generada no está en ningún sitio, y eso se dice con todas
  // las letras. Pero **sigue habiendo un solo aviso**: sustituye al de siempre
  // en vez de sumarse, que dos avisos diciendo lo mismo se leen menos que uno.
  await expect(page.locator(".aviso:visible")).toHaveCount(1);
  await expect(page.locator(".aviso:visible")).toContainText("no está guardada");

  // Y al teclear encima ya es otra clave, así que vuelve el aviso de siempre.
  await clave(page).fill("una clave que me sé");
  await expect(page.locator(".aviso:visible")).toHaveCount(1);
  await expect(page.locator(".aviso:visible")).toContainText("Si pierdes la clave");

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("se puede generar una clave desde la propia pantalla de cifrar", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Se escribe primero lo que se va a cifrar: sacar una clave **no puede
  // llevarse por delante el texto**. Es lo que pasaría si esto se resolviera
  // rehaciendo la pantalla en vez de aplicando la clave sobre la que hay.
  await page.getByLabel("Qué quieres cifrar").fill(SECRETO);

  await page.locator(".contenido div:not([hidden])").getByRole("button", { name: "Generar una" }).click();

  await expect(page.getByLabel("Qué quieres cifrar")).toHaveValue(SECRETO);

  await expect(clave(page)).not.toHaveValue("", { timeout: 20_000 });
  await expect(page.locator(".aviso:visible")).toContainText("no está guardada");

  // En descifrar la clave no se elige, se recuerda: ahí el botón no pinta nada.
  await seccion(page, "Descifrar").click();
  await expect(
    page.locator(".contenido div:not([hidden])").getByRole("button", { name: "Generar una" }),
  ).toHaveCount(0);

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("cambiar de sección ya no borra lo escrito", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Era la deuda: se tecleaba el secreto, se iba uno a Generar a por una clave
  // —el camino que la propia aplicación propone con «Usar como clave»— y al
  // volver el campo estaba vacío, porque cada sección se desmontaba al salir.
  await page.getByLabel("Qué quieres cifrar").fill(SECRETO);
  await clave(page).fill(CLAVE);

  await seccion(page, "Generar").click();
  await seccion(page, "Historial").click();
  await seccion(page, "Cifrar").click();

  await expect(page.getByLabel("Qué quieres cifrar")).toHaveValue(SECRETO);
  await expect(clave(page)).toHaveValue(CLAVE);

  // Y lo de cada pantalla se queda en la suya: el texto cifrado de descifrar no
  // aparece en cifrar ni al revés.
  await seccion(page, "Descifrar").click();
  await expect(page.getByLabel("El texto cifrado")).toHaveValue("");

  expect(errores, errores.join(' | ')).toEqual([]);
});

test("el historial se refresca al volver a entrar, no solo la primera vez", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Desde que las secciones se quedan montadas, montarse pasa una sola vez. Sin
  // recargar al entrar, el historial enseñaría lo que había la primera vez que se
  // miró y no lo que se acaba de cifrar.
  //
  // La prueba **no da por buena la lista que encuentre**: el historial del
  // servidor de desarrollo es de verdad y lo comparten las demás pruebas, y
  // contar antes de que cargue da cero y engaña. Así que primero se cifra —para
  // asegurar que hay algo—, luego se vacía, y solo entonces las cuentas son
  // exactas.
  const cifrarUnaVez = async () => {
    await seccion(page, "Cifrar").click();
    await page.getByLabel("Qué quieres cifrar").fill(SECRETO);
    await clave(page).fill(CLAVE);
    await accion(page, "Cifrar").click();
    await expect(page.locator(".resultado:visible")).toBeVisible({ timeout: 20_000 });
  };

  await cifrarUnaVez();

  await seccion(page, "Historial").click();
  await expect(page.locator(".historial li:visible").first()).toBeVisible({ timeout: 10_000 });
  await accion(page, "Vaciar historial").click();
  await expect(page.locator(".historial li:visible")).toHaveCount(0, { timeout: 10_000 });

  await cifrarUnaVez();

  await seccion(page, "Historial").click();
  await expect(page.locator(".historial li:visible")).toHaveCount(1, { timeout: 10_000 });

  expect(errores, errores.join(' | ')).toEqual([]);
});

// ------------------------------------------------------------ la marca (0021)

test("la marca está en la barra lateral y no estorba a la navegación", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Arriba, el producto.
  const marca = page.locator(".lateral .marca");
  await expect(marca).toContainText("Esfinge");
  await expect(marca.locator("svg")).toBeVisible();

  // Abajo, la casa y la versión. Que la versión esté aquí y no solo en Ajustes
  // es lo que evita tener que ir a buscarla cuando algo va raro.
  const firma = page.locator(".lateral .firma");
  await expect(firma).toContainText("Webcafeína");
  await expect(firma).toContainText(/\d+\.\d+\.\d+/);

  // **Y siguen siendo seis botones** —cinco hasta que llegó la bóveda—. Todo el
  // fichero de pruebas localiza las secciones con «.lateral +
  // getByRole("button")»: si el lockup o la firma fueran interactivos, entrarían
  // en ese localizador y romperían de golpe media suite. Por eso son texto, y
  // por eso esto se cuenta.
  await expect(page.locator(".lateral").getByRole("button")).toHaveCount(6);

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("la firma de la casa va al fondo, debajo de Ajustes", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // Lo sostiene el «margin-top: auto» del grupo de abajo. Comparar las
  // posiciones es la única forma de comprobar que sigue haciendo su trabajo:
  // el orden en el DOM no dice dónde acaba pintado.
  const ajustes = await seccion(page, "Ajustes").boundingBox();
  const firma = await page.locator(".lateral .firma").boundingBox();
  expect(firma!.y).toBeGreaterThan(ajustes!.y);

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("sobre el oro de la fila activa escribe la piedra, no el blanco", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // El instinto de cualquiera que toque esto es poner texto blanco encima, que
  // es lo que hacía cuando el acento era el azul del sistema. Sobre el oro, el
  // blanco da 1,68:1. El test de Go mide los tokens; esto mide lo que se pinta.
  const activa = page.locator('.lateral nav button[aria-current="page"]');
  const estilo = await activa.evaluate((el) => {
    const c = getComputedStyle(el);
    return { fondo: c.backgroundColor, texto: c.color };
  });

  const claro = (c: string) => {
    const [r, g, b] = c.match(/\d+/g)!.map(Number);
    return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
  };
  // El fondo tiene que ser claro —el oro lo es— y el texto oscuro encima.
  expect(claro(estilo.fondo)).toBeGreaterThan(0.5);
  expect(claro(estilo.texto)).toBeLessThan(0.35);

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("el foco de un campo se sigue viendo con el acento dorado", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // **Esta prueba existe por un fallo que casi se cuela.** «make contraste» mide
  // parejas de tokens, no sitios: con el filete de foco pintado del oro habría
  // pasado en verde y el foco habría sido invisible en tema claro, porque el oro
  // sobre blanco da 1,68:1. El filete lo pinta el acento y esto lo vigila.
  const campo = clave(page);
  const borde = () => campo.evaluate((el) => getComputedStyle(el).borderColor);
  const antes = await borde();

  // Con «click» y no con «focus»: la regla es «:focus-visible» y el foco puesto
  // por código no la dispara.
  await campo.click();

  // Y con «poll», porque el borde va con transición: leerlo justo después del
  // clic lo pilla en el fotograma cero y parece que no ha cambiado nada.
  await expect.poll(borde, { timeout: 2000 }).not.toBe(antes);

  // El halo es lo que pone el oro en el foco: el filete se ve y la marca
  // asoma. Si alguien lo cambia por un gris, esto lo dice.
  const halo = await campo.evaluate((el) => getComputedStyle(el).boxShadow);
  expect(halo).toContain("242, 193, 78");

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("el historial vacío enseña la esfinge, y desaparece al haber algo", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // **Se cifra primero a propósito.** Entrar en Historial y preguntar en el acto
  // si «Vaciar» está activo es la trampa que este fichero ya ha pisado dos
  // veces: la lista se carga al entrar, así que en ese instante todavía está
  // vacía y el botón sale desactivado aunque haya entradas. Cifrando antes y
  // esperando a que aparezca la primera fila, el estado deja de ser una
  // suposición.
  await page.getByLabel("Qué quieres cifrar").fill(SECRETO);
  await clave(page).fill(CLAVE);
  await accion(page, "Cifrar").click();
  await expect(page.locator(".resultado:visible")).toContainText("ESF1", { timeout: 20_000 });

  await seccion(page, "Historial").click();
  await expect(page.locator(".historial li:visible").first()).toBeVisible();
  // Con una entrada dentro, la marca no está.
  await expect(page.locator(".contenido .vacio:visible")).toHaveCount(0);

  await accion(page, "Vaciar historial").click();

  // Y al quedarse vacío aparece. Con «:visible», que las secciones se esconden
  // con «hidden» y el bloque existiría igual en las que no se están viendo.
  await expect(page.locator(".contenido .vacio:visible")).toBeVisible();
  await expect(page.locator(".contenido .vacio:visible svg")).toBeVisible();
  await expect(page.getByText("Todavía no has hecho nada.")).toBeVisible();

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("Ajustes dice qué es esto, de qué versión y de quién", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await seccion(page, "Ajustes").click();

  // Acotada a «.contenido»: «Esfinge» está también en el lockup de la barra
  // lateral, que es la trampa de siempre de este fichero.
  const ficha = page.locator(".contenido .ficha:visible");
  await expect(ficha).toContainText("Esfinge");
  // La versión sale de la firma, que en la ficha y en la barra lateral es la
  // misma pieza: «Webcafeína ▍ 2.11.0».
  await expect(ficha.locator(".firma")).toContainText(/\d+\.\d+\.\d+/);
  await expect(ficha).toContainText("Webcafeína");
  await expect(ficha.locator("svg")).toBeVisible();

  expect(errores, errores.join(" | ")).toEqual([]);
});

// ------------------------------------------------------------- la bóveda (0023)

const MAESTRA = "una maestra de prueba";

/**
 * Deja la bóveda abierta, venga de donde venga.
 *
 * **El estado sobrevive a las pruebas, y eso no se puede fingir.** El servidor
 * de desarrollo guarda la bóveda en una carpeta de configuración aislada —de ahí
 * el `-config` de `cmd/dev`, que evita que estas pruebas escriban una bóveda con
 * una contraseña pública en la carpeta de verdad de quien desarrolla— pero esa
 * carpeta la comparten los dos temas y todas las pruebas del fichero. Así que la
 * primera que llega la crea y las demás la abren, y hay que saber estar en los
 * dos casos.
 */
async function conLaBovedaAbierta(page: Page) {
  await seccion(page, "Bóveda").click();

  const crear = accion(page, "Crear la bóveda");
  const abrir = accion(page, "Abrir la bóveda");
  await expect(crear.or(abrir).or(page.locator("#boveda-buscar"))).toBeVisible({
    timeout: 20_000,
  });

  if (await crear.isVisible()) {
    await page.locator("#boveda-maestra").fill(MAESTRA);
    await page.locator("#boveda-maestra-2").fill(MAESTRA);
    await crear.click();

    // La ceremonia: la clave se enseña una vez y hay que decir que se ha
    // apuntado para poder seguir.
    const clave = page.locator(".clave-recuperacion");
    await expect(clave).toBeVisible({ timeout: 20_000 });
    await expect(clave).toHaveText(/^ESF(-[0-9A-HJKMNP-TV-Z]{4})+$/);

    const seguir = accion(page, "Continuar");
    await expect(seguir).toBeDisabled();
    await page.getByText("La he apuntado en un sitio seguro").click();
    await seguir.click();
  } else if (await abrir.isVisible()) {
    await page.locator("#boveda-llave").fill(MAESTRA);
    await abrir.click();
  }

  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });
}

test("guarda una entrada y no enseña la contraseña hasta que se pide", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  // Título distinto en cada pasada: la bóveda sobrevive a la prueba, y dos
  // entradas iguales dejarían el localizador ambiguo.
  const titulo = `Banco ${Date.now()}`;
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(titulo);
  await page.locator("#boveda-usuario").fill("yo@ejemplo.com");
  await page.locator("#boveda-secreto").fill("s3cr3t0");
  await accion(page, "Guardar").click();

  // **La lista viaja sin contraseñas.** Que el secreto no esté en el HTML de la
  // lista no es un detalle de presentación: es la regla del puente.
  const fila = page.locator(".lista-boveda").getByRole("button", { name: titulo });
  await expect(fila).toBeVisible({ timeout: 20_000 });
  expect(await page.locator(".lista-boveda").innerText()).not.toContain("s3cr3t0");

  await fila.click();
  const dato = page.locator(".dato.secreto").first();
  await expect(dato).toHaveText("••••••••••••");
  // Acotado a lo visible: el campo de la clave de cifrar tiene ahora su propio
  // «Ver», y ese panel sigue montado aunque esté escondido.
  await accion(page, "Ver").first().click();
  await expect(dato).toHaveText("s3cr3t0");

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("copiar un secreto dice cuándo va a borrarse solo", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const titulo = `Correo ${Date.now()}`;
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(titulo);
  await page.locator("#boveda-secreto").fill("otra clave");
  await accion(page, "Guardar").click();

  await page.locator(".lista-boveda").getByRole("button", { name: titulo }).click();
  await accion(page, "Copiar").first().click();

  // El plazo lo dice Go por un evento, no un temporizador de la pantalla: el de
  // un webview se pausa y muere al recargar, y un borrado que a veces no ocurre
  // no es un borrado.
  await expect(page.locator(".exito:visible").first()).toContainText(
    /se borra del portapapeles en \d+ s/,
    // El plazo llega por el flujo de eventos, que puede tardar más que la
    // respuesta de la llamada: son dos caminos distintos.
    { timeout: 15_000 },
  );

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("la clave de recuperación abre la bóveda", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
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

  await page.locator("#boveda-llave").fill(recuperacion);
  await accion(page, "Abrir la bóveda").click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("una clave de recuperación con una errata se distingue de una que no abre", async ({ page }) => {
  // Sin vigilar la consola: aquí se piden dos aperturas que tienen que fallar, y
  // el navegador anota cada respuesta 400 como error suyo.
  await page.goto("/");
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

/** Hace algo y espera a que el guardado de preferencias haya ido y vuelto. */
async function guardandoPreferencias(page: Page, hacer: () => Promise<unknown>) {
  const ida = page.waitForResponse((r) => r.url().endsWith("/api/GuardarPreferencias"));
  await hacer();
  await ida;
}

test("Ajustes manda sobre los dos relojes de la bóveda", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await seccion(page, "Ajustes").click();

  // **Esperando a que cada guardado llegue de vuelta**, y no por cortesía: la
  // llamada es asíncrona y recargar la aborta a media petición, con lo que la
  // prueba acaba comprobando lo que había antes. En una tanda pasaba en un tema y
  // no en el otro, que es la forma que tiene una prueba de decir que hay una
  // carrera.
  await guardandoPreferencias(page, () => page.locator("#bloqueo").selectOption("5"));
  await guardandoPreferencias(page, () => page.locator("#portapapeles").selectOption("10"));

  // Que se guarde de verdad, no solo en la pantalla: se recarga y se mira.
  //
  // **Y se espera a que las preferencias lleguen antes de mirar**, que es de
  // donde venía una prueba que fallaba unas veces sí y otras no: mientras la
  // llamada está en vuelo la lista enseña su valor por defecto —quince minutos—,
  // y la aserción competía con esa llamada. Con la máquina cargada por las
  // derivaciones de la bóveda, unas veces ganaba una y otras la otra.
  await page.reload();
  const leidas = page.waitForResponse((r) => r.url().endsWith("/api/VerPreferencias"));
  await seccion(page, "Ajustes").click();
  await leidas;

  await expect(page.locator("#bloqueo")).toHaveValue("5");
  await expect(page.locator("#portapapeles")).toHaveValue("10");

  // Y la bóveda lo cuenta donde toca, que es donde se decide si dejarla abierta.
  await conLaBovedaAbierta(page);
  await expect(page.locator(".contenido")).toContainText(/5 minutos/);

  // Se deja como estaba, que las pruebas de después comparten servidor.
  await seccion(page, "Ajustes").click();
  await guardandoPreferencias(page, () => page.locator("#bloqueo").selectOption("15"));
  await guardandoPreferencias(page, () => page.locator("#portapapeles").selectOption("30"));

  expect(errores, errores.join(" | ")).toEqual([]);
});

// ---------------------------------------------------- el campo de la clave

test("pegar una clave con el menú activa el botón de cifrar", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // **Esto estuvo roto desde que existen los menús propios y no lo vio nadie.**
  // La prueba del menú ejercitaba «seleccionar todo», que es lo fácil de mirar, y
  // no pegar. En la ventana no hay pegar del sistema —los menús se construyen a
  // mano— así que ⌘V pasa por aquí: si el estado de React no se entera, la clave
  // se ve en pantalla y el botón sigue apagado, que es lo que contó el cliente.
  await page.getByLabel("Qué quieres cifrar").fill("un secreto cualquiera");

  const boton = accion(page, "Cifrar");
  await expect(boton).toBeDisabled();

  // El reintento **no puede comparar por igualdad**: cada intento pega otra vez
  // donde está el cursor, así que dos que lleguen dejan el texto duplicado y la
  // comparación no se cumpliría nunca. Lo que importa es que el texto entre, no
  // cuántas veces.
  await clave(page).focus();
  await expect
    .poll(async () => {
      await pegarDesdeElMenu(page, "una clave pegada de fuera");
      return clave(page).inputValue();
    }, { timeout: 15_000 })
    .toContain("una clave pegada de fuera");

  // Lo que importa no es lo que se ve, es que la aplicación lo sepa.
  await expect(boton).toBeEnabled();

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("la clave se puede destapar para leerla y volver a tapar", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  const campo = page.locator("#clave-cifrar");
  await accion(page, "Generar una").click();
  await expect(campo).toHaveAttribute("type", "password");
  await expect(campo).not.toHaveValue("");

  // Sin esto, la contraseña que acaba de fabricarse no se puede sacar de aquí: de
  // un campo de contraseña el navegador se niega a copiar, y ésta no está
  // apuntada en ningún otro sitio.
  //
  // El interruptor es un ojo dentro del campo, así que solo se puede localizar
  // por su nombre accesible: si alguien deja el icono sin nombre, esto se pone
  // rojo, que es exactamente lo que tiene que pasar.
  await accion(page, "Ver la clave").click();
  await expect(campo).toHaveAttribute("type", "text");

  await accion(page, "Ocultar la clave").click();
  await expect(campo).toHaveAttribute("type", "password");

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("copiar la clave con el menú sale por Go, que es el único camino que funciona", async ({
  page,
}) => {
  const errores = vigilarConsola(page);

  // Se mira la petición y no el portapapeles porque el del servidor de desarrollo
  // no se puede leer desde aquí. Lo que hay que comprobar es justo esto: que la
  // orden **sale**. Antes no salía por ninguna parte —`execCommand("copy")` sobre
  // un campo de contraseña devuelve que sí y no copia nada— y el síntoma era que
  // pulsar ⌘C no hacía absolutamente nada.
  const copiadas: string[] = [];
  page.on("request", (r) => {
    if (r.url().endsWith("/api/Copiar")) copiadas.push(r.postData() ?? "");
  });

  await page.goto("/");
  await accion(page, "Generar una").click();
  await expect(page.locator("#clave-cifrar")).not.toHaveValue("");
  const generada = await page.locator("#clave-cifrar").inputValue();

  await clave(page).focus();
  await ordenar(page, "editar:seleccionar-todo");
  await expect
    .poll(async () => {
      await ordenar(page, "editar:copiar");
      return copiadas.length;
    }, { timeout: 15_000 })
    .toBeGreaterThan(0);

  expect(copiadas[0]).toContain(generada);
  expect(errores, errores.join(" | ")).toEqual([]);
});

test("la lista se separa por clases, y «Todo» las junta", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  // Una credencial y una tarjeta, con títulos distintos en cada pasada: la
  // bóveda sobrevive a la prueba y dos entradas iguales dejarían el localizador
  // ambiguo.
  const sello = Date.now();
  const credencial = `Banco ${sello}`;
  const tarjeta = `Visa ${sello}`;

  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(credencial);
  await page.locator("#boveda-secreto").fill("s3cr3t0");
  await accion(page, "Guardar").click();

  await accion(page, "Nueva").click();
  await page.getByRole("tab", { name: "Tarjeta", exact: true }).click();
  await page.locator("#boveda-titulo").fill(tarjeta);
  await page.locator("#boveda-numero").fill("4111111111111111");
  await accion(page, "Guardar").click();

  const lista = page.locator(".lista-boveda");
  await expect(lista.getByRole("button", { name: credencial })).toBeVisible({ timeout: 20_000 });
  await expect(lista.getByRole("button", { name: tarjeta })).toBeVisible();

  // Cada pestaña enseña lo suyo **y esconde lo demás**, que es la mitad que se
  // olvida al comprobar un filtro.
  await page.getByRole("tab", { name: "Tarjetas", exact: true }).click();
  await expect(lista.getByRole("button", { name: tarjeta })).toBeVisible();
  await expect(lista.getByRole("button", { name: credencial })).toHaveCount(0);

  await page.getByRole("tab", { name: "Credenciales", exact: true }).click();
  await expect(lista.getByRole("button", { name: credencial })).toBeVisible();
  await expect(lista.getByRole("button", { name: tarjeta })).toHaveCount(0);

  // Y «Nueva» crea de la clase que se esté mirando, que es lo que se espera
  // estando en Tarjetas.
  await page.getByRole("tab", { name: "Identidades", exact: true }).click();
  await accion(page, "Nueva").click();
  await expect(page.getByRole("tab", { name: "Identidad", exact: true })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await accion(page, "← Dejarlo").click();

  await page.getByRole("tab", { name: "Todo", exact: true }).click();
  await expect(lista.getByRole("button", { name: credencial })).toBeVisible();
  await expect(lista.getByRole("button", { name: tarjeta })).toBeVisible();

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("cada entrada lleva su cuadro, y la clase solo aparece en «Todo»", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const sello = Date.now();
  const banco = `Zzz Banco ${sello}`;
  const nota = `Zzz Nota ${sello}`;

  // Una credencial con sitio y una nota sin él: el cuadro tiene que salir en las
  // dos, porque una fila sin cuadro rompe la columna y se nota más que el cuadro.
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(banco);
  await page.locator("#boveda-sitios").fill("https://www.santander.es/particulares");
  await accion(page, "Guardar").click();

  await accion(page, "Nueva").click();
  await page.getByRole("tab", { name: "Nota", exact: true }).click();
  await page.locator("#boveda-titulo").fill(nota);
  await page.locator("#boveda-notas").fill("algo que guardar");
  await accion(page, "Guardar").click();

  const lista = page.locator(".lista-boveda");
  const filaBanco = lista.locator("li", { has: page.getByText(banco, { exact: true }) });
  const filaNota = lista.locator("li", { has: page.getByText(nota, { exact: true }) });

  // **La letra sale del nombre, no del dominio.** Sacándola del dominio,
  // «Hacienda» salía con la «A» de agenciatributaria.gob.es y parecía un fallo.
  await expect(filaBanco.locator(".monograma")).toHaveText("Z");
  await expect(filaNota.locator(".monograma")).toHaveText("Z");

  // Y el color sale del sitio, así que la credencial y la nota —que no tiene
  // sitio— no tienen por qué coincidir; lo que sí tiene que pasar es que el
  // cuadro lleve uno de los ocho tintes y no se quede sin ninguno.
  await expect(filaBanco.locator(".monograma")).toHaveAttribute("data-tinte", /^[1-8]$/);

  // La clase, solo en «Todo».
  await expect(filaBanco.locator(".clase")).toBeVisible();
  await page.getByRole("tab", { name: "Credenciales", exact: true }).click();
  await expect(lista.locator("li", { has: page.getByText(banco, { exact: true }) })).toBeVisible();
  await expect(
    lista.locator("li", { has: page.getByText(banco, { exact: true }) }).locator(".clase"),
  ).toHaveCount(0);

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("la lista sale ordenada por nombre, y el orden se puede cambiar", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  // Tres títulos que se ordenan al revés de como se meten, con un acento por
  // medio: comparar cadenas a pelo pone «Ángel» detrás de «Zulo», y en castellano
  // va delante.
  const sello = Date.now();
  for (const t of [`Yyy ${sello}`, `Áaa ${sello}`, `Mmm ${sello}`]) {
    await accion(page, "Nueva").click();
    await page.locator("#boveda-titulo").fill(t);
    await page.locator("#boveda-secreto").fill("s3cr3t0");
    await accion(page, "Guardar").click();
  }

  const nombres = page.locator(".lista-boveda .nombre");
  await expect(nombres.filter({ hasText: String(sello) })).toHaveCount(3);

  const enPantalla = async () =>
    (await nombres.allInnerTexts()).filter((n) => n.includes(String(sello)));

  expect(await enPantalla()).toEqual([`Áaa ${sello}`, `Mmm ${sello}`, `Yyy ${sello}`]);

  // Por lo último cambiado. **Las tres se han creado en el mismo segundo**, y la
  // fecha se guarda con esa precisión, así que empatan: lo que las separa es el
  // desempate por nombre. Se toca una y tiene que subir sola.
  await page.locator("#boveda-orden").selectOption("cambiada");
  expect(await enPantalla()).toEqual([`Áaa ${sello}`, `Mmm ${sello}`, `Yyy ${sello}`]);

  // **Un segundo de espera, y hace falta**: la fecha se guarda con precisión de
  // segundo, así que tocar una entrada dentro del mismo segundo en que se creó no
  // la mueve. Para una persona eso da igual —«lo último que toqué» se mide en
  // días—; para una prueba que hace tres cosas en 200 ms, no.
  await page.waitForTimeout(1100);
  await page.locator(".lista-boveda").getByRole("button", { name: `Yyy ${sello}` }).click();
  await accion(page, "Editar").click();
  await page.locator("#boveda-notas").fill("tocada la última");
  await accion(page, "Guardar").click();
  // Guardar vuelve a la lista y **la vuelve a pedir**: hay que esperar a que esté
  // otra vez, o se lee la pantalla a medio dibujar.
  await expect(nombres.filter({ hasText: String(sello) })).toHaveCount(3);
  expect(await enPantalla()).toEqual([`Yyy ${sello}`, `Áaa ${sello}`, `Mmm ${sello}`]);

  await page.locator("#boveda-orden").selectOption("nombre");
  expect(await enPantalla()).toEqual([`Áaa ${sello}`, `Mmm ${sello}`, `Yyy ${sello}`]);

  expect(errores, errores.join(" | ")).toEqual([]);
});

// **Va la última del fichero a propósito**: deja la bóveda borrada, y quien venga
// detrás —el otro tema— la crea otra vez con `conLaBovedaAbierta`. Ponerla antes
// obligaría a todas las demás a saber si les toca crear o abrir, que es
// exactamente lo que ese ayudante existe para que no haya que pensar.
test("borrar la bóveda pide la contraseña maestra y no perdona", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  await page.getByRole("button", { name: "Contraseña maestra y clave de recuperación" }).click();

  const borrar = accion(page, "Borrar la bóveda…");
  await expect(borrar).toBeDisabled(); // sin contraseña no se puede ni empezar

  // **Un aviso no puede ser un contenedor flexible**, y esto lo vigila. Con
  // «display: flex» cada trozo del párrafo se convierte en un elemento por su
  // cuenta: una palabra en negrita en medio de una frase se sale a una columna
  // aparte y la frase se lee en vertical, partida en pedazos. Lo era desde el
  // principio y no se notó mientras todos los avisos fueron texto pelado; se vio
  // mirando una captura, no en una prueba en verde.
  const comoSePinta = await page
    .locator(".peligro .aviso")
    .evaluate((el) => getComputedStyle(el).display);
  expect(comoSePinta).not.toBe("flex");

  // Con la contraseña equivocada no se borra nada, y hay que decirlo antes de
  // que alguien se quede sin bóveda creyendo que se la ha llevado un fallo.
  await page.locator("#boveda-borrar").fill("ésta no es");
  await borrar.click();
  await accion(page, "Sí, borrarla para siempre").click();
  await expect(page.locator(".error:visible")).toContainText("no es la contraseña");
  await expect(page.locator("#boveda-buscar")).toBeVisible();

  // Y con la buena hacen falta dos pulsaciones: la primera solo cambia el rótulo.
  await page.locator("#boveda-borrar").fill(MAESTRA);
  await accion(page, "Borrar la bóveda…").click();
  await expect(page.locator("#boveda-buscar")).toBeVisible();
  await accion(page, "Sí, borrarla para siempre").click();

  // Y se vuelve al principio de todo, que es lo que significa haberla borrado.
  await expect(accion(page, "Crear la bóveda")).toBeVisible({ timeout: 20_000 });

  expect(errores.filter((e) => !e.includes("400")), errores.join(" | ")).toEqual([]);
});
