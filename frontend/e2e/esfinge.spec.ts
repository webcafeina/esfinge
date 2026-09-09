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

  // Abajo, la casa.
  await expect(page.locator(".lateral .firma")).toContainText("webcafeína");

  // **Y siguen siendo cinco botones.** Todo el fichero de pruebas localiza las
  // secciones con «.lateral + getByRole("button")»: si el lockup o la firma
  // fueran interactivos, entrarían en ese localizador y romperían de golpe
  // media suite. Por eso son texto, y por eso esto se cuenta.
  await expect(page.locator(".lateral").getByRole("button")).toHaveCount(5);

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
  await expect(ficha).toContainText("Versión");
  await expect(ficha).toContainText("webcafeína");
  await expect(ficha.locator("svg")).toBeVisible();

  expect(errores, errores.join(" | ")).toEqual([]);
});
