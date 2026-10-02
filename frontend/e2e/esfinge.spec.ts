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
/**
 * **Estas pruebas trabajan en local**, y lo eligen antes de empezar: el servidor
 * de desarrollo arranca sin nada, y sin elegir, la bienvenida de la cuenta taparía
 * la ventana. La bienvenida tiene sus propias pruebas, en cuentas.spec.ts.
 */
test.beforeEach(async ({ request }) => {
  const r = await request.post("/api/ElegirModoLocal", { data: [] });
  expect(r.ok(), await r.text()).toBe(true);
});

/**
 * Deja el equipo como si nunca se hubiera ofrecido el desbloqueo a esta bóveda.
 *
 * Va **por el mismo puente que usa la ventana** —`POST /api/<Método>` con los
 * argumentos en una lista—, no tocando el fichero: así la prueba no depende de
 * dónde vive ni de cómo se serializa.
 *
 * Desde la 2.28.3 lo que se guarda no es un sí/no sino **de qué bóveda se trata**,
 * así que para volver a ofrecerlo basta con dejarlo en vacío, que es lo que
 * significa «a ninguna».
 */
async function volverAOfrecerElDesbloqueo(page: Page) {
  const antes = await (await page.request.post("/api/VerPreferencias", { data: [] })).json();
  await page.request.post("/api/GuardarPreferencias", {
    data: [{ ...antes, desbloqueoSugeridoPara: "" }],
  });
}

function vigilarConsola(page: Page): string[] {
  const errores: string[] = [];
  page.on("console", (m) => m.type() === "error" && errores.push(m.text()));
  page.on("pageerror", (e) => errores.push(String(e)));
  // **Y qué petición falló, que el navegador no lo dice.** «Failed to load
  // resource: 400» es todo lo que sale por consola, y con eso no se puede ir a
  // ninguna parte: ni qué se pidió ni qué contestó Go. Apuntarlo aquí convierte
  // una prueba que falla sin explicación en una que dice dónde mirar.
  page.on("response", (r) => {
    const camino = new URL(r.url()).pathname;
    // **Menos los que son una respuesta y no un fallo.** Pedir el desbloqueo con el
    // sistema es una pregunta que puede decir que no —aquí el llavero de mentira no
    // tiene nada guardado—, y la pantalla lo trata como lo que es: se vuelve a la
    // contraseña maestra. Contarlo como error haría que la prueba fallara por que
    // algo funciona.
    if (r.status() >= 400 && !camino.endsWith("/AbrirBovedaConElSistema")) {
      errores.push(`HTTP ${r.status()} en ${r.request().method()} ${camino}`);
    }
  });
  return errores;
}

/**
 * Quita de la lista los errores que **la prueba ha provocado a propósito**, como
 * pedir algo con una contraseña que no es.
 *
 * El vigilante se queda estricto para todo lo demás, que es su trabajo: lo que no
 * se descuenta aquí tiene que no ocurrir. Y se exige que el error **esté**, porque
 * una prueba que descuenta algo que no ha pasado deja de comprobar lo que dice.
 */
function olvidarEsperado(errores: string[], patron: RegExp) {
  const antes = errores.length;
  for (let i = errores.length - 1; i >= 0; i--) {
    if (patron.test(errores[i])) errores.splice(i, 1);
  }
  expect(antes, `no ha ocurrido el error que la prueba esperaba (${patron})`).toBeGreaterThan(errores.length);
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
  //
  // Acotado a la banda: desde la 2.14.0, Ajustes tiene su propio aviso —el de que
  // los iconos salen a la red— y es legítimo que esté ahí. Lo que esta prueba
  // vigila es que **la banda de versión** no sea uno de ésos.
  await expect(banda.locator(".aviso")).toHaveCount(0);

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
 * **El freno de las llaves de acceso está en Ajustes y viene encendido** (ADR 0048).
 *
 * Es la única pieza de toda la fase que se puede usar cuando algo se rompe, así que
 * lo que hay que comprobar no es que la casilla exista: es que **viene puesta de
 * fábrica** —que es como se decidió publicarla— y que **apagarla sobrevive a cerrar
 * la ventana**, porque un freno que se suelta al reiniciar no sirve para lo que está.
 */
test("las llaves de acceso se pueden apagar en Ajustes, y se quedan apagadas", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  // **Se prepara su punto de partida, y aquí no es una precaución: es la regla del
  // cero mordiendo.** La prueba de arriba guarda las preferencias **a medias** —solo
  // `buscarActualizaciones`, a propósito, para ejercitar esa trampa— y eso llega con
  // todos los demás campos a `false`, así que **apaga las llaves de acceso**. Es el
  // lado seguro de equivocarse y está decidido así, pero significa que este valor de
  // fábrica no se puede dar por hecho.
  //
  // **Y por qué pasaba aquí y caía en GitHub**, que es la parte que no se adivina:
  // la prueba de arriba, después del guardado a medias, hace `check()` y React manda
  // las preferencias **enteras con lo que tenga en memoria**. Si las había leído antes
  // del guardado a medias, las restaura sin querer; si las lee después, las manda ya
  // apagadas. O sea, **depende de qué lectura gana la carrera** —la misma de
  // `cambiosHechos`—, y eso lo decide lo rápida que sea la máquina. Reproducido aquí
  // forzando el guardado a medias: Go contesta `false` y el síntoma es idéntico al de
  // GitHub, un `<input checked>` en el HTML con el estado real en `unchecked`, porque
  // React deja el atributo del primer render y cambia solo la propiedad.
  //
  // Se leen y se vuelven a guardar **enteras**, que es lo que hace la ventana: un
  // objeto a medias aquí volvería a apagar media pantalla.
  await page.evaluate(async () => {
    const leer = await fetch("/api/VerPreferencias", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "[]",
    });
    const prefs = (await leer.json()) as Record<string, unknown>;
    await fetch("/api/GuardarPreferencias", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify([{ ...prefs, llavesDeAccesoEnElNavegador: true }]),
    });
  });
  await page.reload();
  await seccion(page, "Ajustes").click();

  const llaves = page.getByRole("checkbox", { name: /llaves de acceso/ });
  await expect(llaves).toBeChecked();

  await llaves.uncheck();
  await page.reload();
  await seccion(page, "Ajustes").click();
  await expect(page.getByRole("checkbox", { name: /llaves de acceso/ })).not.toBeChecked();

  // Como el de arriba: el fichero de preferencias lo comparten las demás pruebas.
  await page.getByRole("checkbox", { name: /llaves de acceso/ }).check();
  await expect(page.getByRole("checkbox", { name: /llaves de acceso/ })).toBeChecked();

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

  // **Y siguen siendo siete botones** —cinco hasta la bóveda, seis hasta las
  // bóvedas de proyecto (ADR 0050)—. Todo el fichero de pruebas localiza las
  // secciones con «.lateral + getByRole("button")»: si el lockup o la firma fueran
  // interactivos, entrarían en ese localizador y romperían de golpe media suite.
  // Por eso son texto, y por eso esto se cuenta.
  await expect(page.locator(".lateral").getByRole("button")).toHaveCount(7);

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

// El código de un solo uso, que es lo que ata a un gestor de contraseñas cuando
// todo lo demás ya se puede llevar uno.
//
// La semilla es la de los vectores del RFC, así que el código que sale de aquí
// lo puede comprobar cualquiera con cualquier autenticador.
test("el código de un solo uso sale calculado y contando atrás", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const semilla = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ";
  const titulo = `Segundo factor ${Date.now()}`;
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(titulo);
  await page.locator("#boveda-secreto").fill("s3cr3t0");
  await page.locator("#boveda-totp").fill(semilla);
  await accion(page, "Guardar").click();

  await page.locator(".lista-boveda").getByRole("button", { name: titulo }).click();

  const codigo = page.locator(".codigo-unico");
  await expect(codigo).toBeVisible({ timeout: 20_000 });

  // **Seis cifras seguidas en el texto**, aunque en pantalla se vean en dos
  // grupos: el hueco lo pone el CSS, y quien las seleccione con el ratón se
  // lleva lo que el servicio espera, no «123 456».
  expect(await codigo.evaluate((n) => n.textContent ?? "")).toMatch(/^\d{6}$/);

  // Y la semilla **no está en la pantalla**. Es el segundo factor entero: se
  // enseña el código, que caduca en treinta segundos, no lo que lo genera.
  expect(await page.locator(".contenido").innerText()).not.toContain("GEZDGNBV");

  // La cuenta atrás corre de verdad. Se mira que cambie, no que baje: entre las
  // dos lecturas puede haber saltado de intervalo y volver a treinta.
  const cuanto = page.getByText(/^Vale \d+ s más$/);
  const antes = await cuanto.innerText();
  await expect(cuanto).not.toHaveText(antes, { timeout: 5_000 });

  expect(errores, errores.join(" | ")).toEqual([]);
});

// La papelera, que es lo que hace que borrar deje de ser irreversible.
//
// Se comprueba **con la contraseña**, y ahí está la gracia: una papelera que
// devuelve el título y no el secreto no sirve para nada, y es justo lo que hacía
// esto antes de la 2.16.0.
test("lo borrado va a la papelera y vuelve entero", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const titulo = `Se borra ${Date.now()}`;
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(titulo);
  await page.locator("#boveda-secreto").fill("vuelve entera");
  await accion(page, "Guardar").click();

  await page.locator(".lista-boveda").getByRole("button", { name: titulo }).click();
  await accion(page, "Borrar").click();
  // La segunda pulsación dice adónde va: es el único momento en que alguien que
  // duda se entera de que esto se puede deshacer.
  await accion(page, "Sí, a la papelera").click();

  await expect(page.locator(".exito:visible").first()).toContainText(
    "está en la papelera",
  );
  await expect(page.locator(".lista-boveda").getByRole("button", { name: titulo })).toHaveCount(0);

  // El botón de la papelera aparece solo cuando hay algo dentro, y la cuenta la
  // comparten todas las pruebas de este fichero: por eso no se afirma cuál es.
  const papelera = page.locator(".contenido").getByRole("button", { name: /^Papelera \(\d+\)$/ });
  await papelera.click();

  const fila = page.locator(".lista-papelera li").filter({ hasText: titulo });
  await expect(fila).toHaveCount(1);
  await expect(fila).toContainText("Borrada hoy");
  await fila.getByRole("button", { name: "Restaurar" }).click();
  await expect(fila).toHaveCount(0);

  await accion(page, "← Volver").click();
  await page.locator(".lista-boveda").getByRole("button", { name: titulo }).click();
  await accion(page, "Ver").first().click();
  await expect(page.locator(".dato.secreto").first()).toHaveText("vuelve entera");

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("vaciar la papelera se lo lleva, y pide una segunda pulsación", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const titulo = `Se va del todo ${Date.now()}`;
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(titulo);
  await page.locator("#boveda-secreto").fill("s3cr3t0");
  await accion(page, "Guardar").click();

  await page.locator(".lista-boveda").getByRole("button", { name: titulo }).click();
  await accion(page, "Borrar").click();
  await accion(page, "Sí, a la papelera").click();

  await page.locator(".contenido").getByRole("button", { name: /^Papelera \(\d+\)$/ }).click();
  await expect(page.locator(".lista-papelera li").filter({ hasText: titulo })).toHaveCount(1);

  // Dos pulsaciones, como todo lo que no tiene vuelta atrás en esta pantalla.
  await accion(page, "Vaciar la papelera").click();
  await page.locator(".contenido").getByRole("button", { name: /^Sí, vaciar las \d+$/ }).click();

  await expect(page.locator(".lista-papelera")).toHaveCount(0);
  await expect(page.getByText("La papelera está vacía.")).toBeVisible();

  // Y de vuelta en la lista ya no hay botón de papelera, porque no hay papelera.
  await accion(page, "← Volver").click();
  await expect(
    page.locator(".contenido").getByRole("button", { name: /^Papelera \(\d+\)$/ }),
  ).toHaveCount(0);

  expect(errores, errores.join(" | ")).toEqual([]);
});

// El canal con el navegador (fase 2).
//
// **Viene apagado**, al revés que las otras dos cosas que Esfinge hace fuera de
// sí misma, y la diferencia importa: aquéllas salen a la red y ésta abre una
// puerta a este ordenador. Encenderlo y apagarlo tiene que valer desde ya.
//
// Lo que esta prueba **no** cubre, y hay que decirlo: el aviso de «un navegador
// quiere consultar tu bóveda». Para provocarlo hace falta que algo se conecte al
// socket, y desde un navegador no se puede abrir un socket de dominio unix. Ese
// camino está probado en Go (`TestElEmparejamientoSePideYSeConcedeUnaVez`) y se
// verá de verdad cuando exista la extensión.
test("el canal con el navegador viene apagado y se enciende en Ajustes", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await seccion(page, "Ajustes").click();

  const casilla = page.getByLabel("Dejar que la extensión del navegador consulte la bóveda");
  await expect(casilla).toBeVisible({ timeout: 20_000 });
  // Sin cuenta, el canal es la única forma de que la extensión pregunte: la nota
  // que dice que sobra no puede salir aquí.
  await expect(page.getByText("Con cuenta no hace falta.")).toHaveCount(0);
  // Punto de partida propio: el fichero de preferencias dura entre tandas.
  if (await casilla.isChecked()) {
    await casilla.uncheck();
  }
  await expect(page.getByText(/^Escucha en /)).toHaveCount(0);

  await casilla.check();
  // **Que diga dónde escucha no es un adorno**: en un programa que guarda
  // contraseñas, una puerta abierta se dice dónde está.
  await expect(page.getByText(/^Escucha en /)).toBeVisible({ timeout: 20_000 });

  await casilla.uncheck();
  await expect(page.getByText(/^Escucha en /)).toHaveCount(0);

  expect(errores, errores.join(" | ")).toEqual([]);
});

/**
 * La lista de sitios donde no se ofrece guardar, en Ajustes.
 *
 * **El estado se siembra por `/api/_excluir`**, un extremo del servidor de desarrollo
 * que no existe fuera de la etiqueta `dev`. Y no es un atajo: «Nunca en este sitio» se
 * decide en la tarjeta de la página y llega por el canal de la extensión (ADR 0032),
 * así que **desde la ventana solo se puede quitar, nunca poner**. La alternativa era
 * exportar un método en `App` para comodidad de esta prueba, que es exactamente lo que
 * la lista blanca del puente existe para impedir.
 *
 * Estaba en `docs/deuda.md` desde que se escribió la lista: «Go está probado; la lista
 * de la ventana no la ejercita `make e2e`».
 */
test("la lista de sitios excluidos se ve y se puede quitar", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const sitio = `no-ofrecer-${Date.now()}.ejemplo.com`;
  const puesto = await page.request.post(`/api/_excluir?dominio=${sitio}`);
  expect(puesto.ok(), await puesto.text()).toBe(true);

  await seccion(page, "Ajustes").click();
  const fila = page.locator(".lista-papelera li", { hasText: sitio });
  await expect(fila).toBeVisible({ timeout: 20_000 });

  await fila.getByRole("button", { name: "Quitar" }).click();
  await expect(fila).toHaveCount(0);

  // Y no vuelve al recargar: se quitó de la bóveda, no de la pantalla.
  await page.reload();
  await conLaBovedaAbierta(page);
  await seccion(page, "Ajustes").click();
  await expect(page.locator(".lista-papelera li", { hasText: sitio })).toHaveCount(0);

  expect(errores, errores.join(" | ")).toEqual([]);
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

  // **Se afirma con `expect` sobre el localizador y no leyendo los textos a
  // mano**, y no es una preferencia de estilo: leerlos devuelve lo que hubiera
  // en pantalla en ese instante, y guardar una entrada vuelve a pedir la lista,
  // así que la lectura puede llegar antes que la lista nueva. Contar tres
  // tampoco salva —ya eran tres antes de guardar—. Esta forma reintenta hasta
  // que el orden es el que se espera, que es lo que hay que comprobar.
  const nombres = page.locator(".lista-boveda .nombre").filter({ hasText: String(sello) });
  await expect(nombres).toHaveText([`Áaa ${sello}`, `Mmm ${sello}`, `Yyy ${sello}`]);

  // Por lo último cambiado: lo que se acaba de tocar sube arriba.
  //
  // **No se afirma en qué orden quedan las otras dos**, y es a propósito: la fecha
  // se guarda con precisión de segundo, así que según lo rápida que vaya la
  // máquina las tres caen en el mismo segundo —y manda el desempate por nombre— o
  // en segundos distintos —y manda la fecha—. Afirmar una de las dos cosas es
  // escribir una prueba que falla una de cada cinco veces por el reloj.
  await page.locator("#boveda-orden").selectOption("cambiada");

  // **Un segundo de espera, y hace falta**: tocar una entrada dentro del mismo
  // segundo en que se creó no la mueve. Para una persona eso da igual —«lo último
  // que toqué» se mide en días—; para una prueba que hace tres cosas en 200 ms, no.
  await page.waitForTimeout(1100);
  await page.locator(".lista-boveda").getByRole("button", { name: `Yyy ${sello}` }).click();
  await accion(page, "Editar").click();
  await page.locator("#boveda-notas").fill("tocada la última");
  await accion(page, "Guardar").click();
  // Guardar vuelve a la lista y **la vuelve a pedir**: hay que esperar a que la
  // de después esté puesta, o se lee la de antes.
  await expect(nombres.first()).toHaveText(`Yyy ${sello}`);

  await page.locator("#boveda-orden").selectOption("nombre");
  await expect(nombres).toHaveText([`Áaa ${sello}`, `Mmm ${sello}`, `Yyy ${sello}`]);

  expect(errores, errores.join(" | ")).toEqual([]);
});

test("las clases de la bóveda van sin rótulo, y los gestores con él", async ({
  page,
}) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  // **Y no es lo mismo, aunque los dos controles lleven glifo.** Una llave o una
  // tarjeta se adivinan; cinco marcas ajenas sin su nombre al lado no las
  // reconoce nadie. Por eso solo las clases se encogen.
  await page.setViewportSize({ width: 700, height: 620 });

  const clase = page.getByRole("tab", { name: "Tarjetas", exact: true });
  const gestor = page.getByRole("tab", { name: "Bitwarden", exact: true });

  // El rótulo sigue estando para quien lee la pantalla en voz alta —y para estos
  // localizadores— aunque no se vea.
  await expect(clase).toBeVisible();
  await expect(gestor).toBeVisible();

  const anchoDe = (l: typeof clase) => l.evaluate((el) => el.getBoundingClientRect().width);
  const claseEstrecha = await anchoDe(clase);
  expect(claseEstrecha).toBeLessThan(await anchoDe(gestor));

  // **Y al ensanchar no vuelven, que es lo contrario de lo que esta prueba decía
  // hasta la ADR 0047.** La columna de contenido está topada en 560 px y no crece
  // con la ventana, así que lo que quepa no lo decide el ancho de la ventana. Con
  // cinco clases los rótulos medían 527 y entraban por doce píxeles; la sexta pide
  // 160 más. La regla que los devolvía miraba la ventana y acertaba de milagro.
  await page.setViewportSize({ width: 1400, height: 620 });
  await expect.poll(() => anchoDe(clase)).toBe(claseEstrecha);

  expect(errores, errores.join(" | ")).toEqual([]);
});

// **La dirección se guarda en nueve campos y se lee compuesta, en el orden del
// sobre.**
//
// La composición está escrita dos veces —`Entrada.Direccion()` en Go y `direccionDe`
// en la ventana— porque es una frase para leer y no un formato que tenga que dar los
// mismos bytes. Esto es lo que ata la de la ventana a lo que se espera: el orden de
// los campos no es el del sobre, y por orden de campo saldría «Calle Mayor 1,
// España, Madrid, 28001», que es una lista y no una dirección.
test("un dato personal guarda la dirección por trozos y la enseña compuesta", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const sello = Date.now();
  await accion(page, "Nueva").click();
  await page.getByRole("tab", { name: "Dato personal", exact: true }).click();
  await page.locator("#boveda-titulo").fill(`Casa ${sello}`);
  for (const [id, valor] of [
    ["destinatario", "Álvaro Cabezas"],
    ["calle", "Calle Mayor 1"],
    ["edificio", "Portal B"],
    ["piso", "3"],
    ["puerta", "B"],
    ["codigo-postal", "28001"],
    ["ciudad", "Madrid"],
    // **La misma que la ciudad a propósito**: así no se repite.
    ["provincia", "Madrid"],
    ["pais", "España"],
  ]) {
    await page.locator(`#boveda-${id}`).fill(valor);
  }
  await accion(page, "Guardar").click();

  await page.getByRole("button", { name: `Casa ${sello}`, exact: false }).first().click();
  const direccion = page.locator(".grupo > div", { hasText: "Dirección" }).first();
  await direccion.getByRole("button", { name: "Ver", exact: true }).click();
  // La provincia no sale: se llama igual que la ciudad.
  await expect(direccion.locator(".dato")).toHaveText(
    ["Álvaro Cabezas", "Calle Mayor 1, Portal B", "3, B", "28001 Madrid", "España"].join("\n"),
  );

  expect(errores, errores.join(" | ")).toEqual([]);
});

// **Las siete clases caben en su barra**, que es lo que ninguna aserción miraba.
//
// Se añadió la de datos personales, todo siguió en verde y en la captura el
// rótulo de la última salía cortado por el borde del panel. Comparar lo que la
// barra necesita con lo que mide es la pregunta que el CSS no contesta solo, y
// vale para cualquier clase que se añada después de ésta.
test("las clases de la bóveda caben en su barra", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  // **Con cada una activa, no solo con la primera.** La activa es la única que
  // lleva rótulo, así que el caso peor es la de nombre más largo —«Datos
  // personales», 160 px— y mirando solo la que viene puesta se comprueba el mejor.
  const clases = ["Todo", "Credenciales", "Notas", "Tarjetas", "Identidades", "Datos personales", "Llaves de acceso", "Wi-Fi"];
  for (const ancho of [700, 980, 1400]) {
    await page.setViewportSize({ width: ancho, height: 620 });
    for (const nombre of clases) {
      await page.getByRole("tab", { name: nombre, exact: true }).click();
      const m = await page
        .locator(".segmentado.compacto")
        .evaluate((el) => ({
          necesita: el.scrollWidth,
          mide: el.clientWidth,
          cabe: el.parentElement ? el.parentElement.clientWidth : 0,
        }));
      expect(m.necesita, `«${nombre}» activa con la ventana de ${ancho}: ${JSON.stringify(m)}`)
        .toBeLessThanOrEqual(m.mide);
      expect(m.mide, `«${nombre}» activa con la ventana de ${ancho}: ${JSON.stringify(m)}`)
        .toBeLessThanOrEqual(m.cabe);
    }
    const medidas = await page
      .locator(".segmentado.compacto")
      .evaluate((el) => ({
        necesita: el.scrollWidth,
        mide: el.clientWidth,
        cabe: el.parentElement ? el.parentElement.clientWidth : 0,
      }));
    expect(medidas.necesita, `con la ventana de ${ancho}: ${JSON.stringify(medidas)}`)
      .toBeLessThanOrEqual(medidas.mide);
    expect(medidas.mide, `con la ventana de ${ancho}: ${JSON.stringify(medidas)}`)
      .toBeLessThanOrEqual(medidas.cabe);
  }

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

// Juntar dos bóvedas importadas del mismo gestor dejó cada cuenta dos veces en los
// Macs del cliente (2.24.1). La bóveda lo dice y las limpia de un clic, a la
// papelera: se queda una de cada.
test("las entradas repetidas se ven y se quitan de un clic", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const titulo = `Repetida ${Date.now()}`;
  // Iguales a la vista, distintas en lo que la ventana no enseña: así estaban las
  // del cliente, y la 2.24.2 —que pedía iguales en todo— no las encontró.
  for (const carpeta of ["", "Dashlane", "Email"]) {
    await page.evaluate(
      ([t, c]) =>
        fetch("/api/GuardarEnBoveda", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify([{ tipo: "credencial", titulo: t, usuario: "yo", secreto: "igual", carpeta: c }]),
        }),
      [titulo, carpeta],
    );
  }
  await page.reload();
  await conLaBovedaAbierta(page);
  await page.locator("#boveda-buscar").fill(titulo);
  await expect(page.locator(".panel:visible").getByText(titulo)).toHaveCount(3);

  await expect(page.getByText("Hay 2 cuentas repetidas: mismo título, usuario y contraseña que otra.", { exact: false })).toBeVisible();
  await page.screenshot({ path: `test-results/repetidas-${test.info().project.name}.png` });
  await accion(page, "Quitar las repetidas").click();

  await expect(page.getByText("2 cuentas repetidas están en la papelera", { exact: false })).toBeVisible();
  await expect(page.locator(".panel:visible").getByText(titulo)).toHaveCount(1);
  await expect(accion(page, "Quitar las repetidas")).toHaveCount(0);
  expect(errores).toEqual([]);
});

// La fila de la bóveda dice si está abierta o cerrada, con un candado que avisa Go
// —no la pantalla de la bóveda—, así que también cambia cuando se cierra desde
// fuera. Y el nombre de la sección sigue siendo «Bóveda».
test("la barra lateral dice si la bóveda está abierta o cerrada", async ({ page, request }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);
  const candado = page.locator(".lateral .estado-boveda");
  await expect(candado).toHaveAttribute("data-abierta", "si");
  await expect(seccion(page, "Bóveda")).toHaveAttribute("title", "La bóveda está abierta");
  await seccion(page, "Cifrar").click();
  await page.locator(".lateral").screenshot({ path: `test-results/candado-abierta-${test.info().project.name}.png` });
  await seccion(page, "Bóveda").click();

  // Cerrada desde fuera de la ventana —como la cierra el reloj de inactividad—.
  expect((await request.post("/api/CerrarBoveda", { data: [] })).ok()).toBe(true);
  await expect(candado).toHaveAttribute("data-abierta", "no");
  await expect(seccion(page, "Bóveda")).toHaveAttribute("title", "La bóveda está cerrada");
  await page.locator(".lateral").screenshot({ path: `test-results/candado-cerrada-${test.info().project.name}.png` });
  expect(errores).toEqual([]);
});

/**
 * Desbloquear con el sistema, de punta a punta (fase C).
 *
 * Aquí no hay Touch ID ni Windows Hello, así que el servidor de desarrollo lleva
 * un **llavero de mentira** en memoria (`-sin-llavero` simula el equipo que no
 * tiene ninguno). Lo que esta prueba comprueba no es la biometría —eso solo lo
 * dice un Mac— sino **todo lo demás**: que se activa con la bóveda abierta, que
 * entonces la pantalla de desbloquear lo ofrece, que abre, que la maestra sigue
 * abriendo, y que al quitarlo desaparece.
 */
test("la bóveda se abre con el sistema, y la maestra sigue abriendo", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  // **La bóveda lo ofrece sola la primera vez** (2.27.2), que es lo que impide
  // que esto sea una función que solo encuentra quien ya la buscaba: el
  // interruptor vive en Ajustes y ahí no entra quien no sabe que existe.
  await seccion(page, "Bóveda").click();
  const ofrecer = page.locator("#sugerencia-activar");
  await expect(ofrecer).toBeVisible({ timeout: 20_000 });
  await expect(ofrecer).toHaveText(/Touch ID/);

  // Y «Ahora no» la quita **para siempre**, no hasta la próxima vez: una tarjeta
  // que reaparece cada vez que abres la bóveda deja de ser una sugerencia.
  await page.locator("#sugerencia-ahora-no").click();
  await expect(ofrecer).toHaveCount(0);
  await accion(page, "Cerrar la bóveda").click();
  await expect(page.locator("#boveda-activar-al-abrir")).toHaveCount(0);
  await page.locator("#boveda-llave").fill(MAESTRA);
  await accion(page, "Abrir la bóveda").click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });
  await expect(ofrecer).toHaveCount(0);

  // **Y la otra forma de ofrecerlo: la casilla de la pantalla de desbloquear**
  // (2.27.3). Activarlo exige la bóveda abierta, así que ahí no puede haber un
  // botón que lo haga; lo que hay es una casilla que lo deja activado **al
  // abrir**, con la maestra recién escrita.
  //
  // Se vuelve a poner como si no se hubiera ofrecido, por el mismo puente que usa
  // la ventana: en una tanda la bóveda es la misma, y la tarjeta de arriba ya ha
  // gastado su turno.
  await volverAOfrecerElDesbloqueo(page);
  await accion(page, "Cerrar la bóveda").click();
  const casilla = page.locator("#boveda-activar-al-abrir");
  await expect(casilla).toBeVisible({ timeout: 20_000 });
  await expect(page.locator("label", { has: casilla })).toContainText("Touch ID");
  await casilla.click();
  await expect(casilla).toBeChecked();
  await page.locator("#boveda-llave").fill(MAESTRA);
  await accion(page, "Abrir la bóveda").click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });

  // **Y dentro no se vuelve a ofrecer lo que se acaba de pedir**, se confirma.
  // Faltaba esta línea, y por eso la prueba estaba en verde mientras el cliente
  // veía la tarjeta ofreciéndoselo otra vez después de marcar la casilla.
  await expect(page.getByText("Touch ID activado")).toBeVisible({ timeout: 20_000 });
  await expect(page.locator("#sugerencia-activar")).toHaveCount(0);

  // En Ajustes aparece ya marcado —lo activó abrir, no un botón de allí—, y ahí
  // es donde se dice lo que protege y lo que no.
  await seccion(page, "Ajustes").click();
  const interruptor = page.locator("#desbloqueo-del-sistema");
  await expect(interruptor).toBeVisible({ timeout: 20_000 });
  await expect(interruptor).toBeChecked({ timeout: 20_000 });
  await expect(page.getByText("no de un programa que corra en él")).toBeVisible();

  // **El llavero diciendo que no**, que es la única forma de llegar al campo de la
  // maestra desde que la huella se pide sola: en un Mac se cancela el diálogo, y
  // aquí no hay diálogo que cancelar. Con esto la prueba ejercita además **el
  // camino de cancelar**, que hasta ahora no miraba nadie.
  //
  // Va **antes** de cerrar la bóveda a propósito: puesto después, entre el cierre
  // y la orden cabe la huella automática, y lo que se mira depende de quién llegue
  // primero. Así no hay carrera que valga.
  await page.request.get("/api/_llavero?dice=no");
  await seccion(page, "Bóveda").click();
  await accion(page, "Cerrar la bóveda").click();
  const conElSistema = page.locator("#boveda-con-el-sistema");
  await expect(conElSistema).toBeVisible({ timeout: 20_000 });
  await expect(conElSistema).toHaveText(/Touch ID/);
  // Cancelar **no se pinta en rojo**: es una decisión, no un fallo.
  await expect(page.locator(".panel .error")).toHaveCount(0);

  // **Y la maestra sigue abriendo** con la ranura del sistema puesta: es la regla
  // que no se puede romper, y por eso se comprueba sin quitarla antes.
  await page.locator("#boveda-llave").fill(MAESTRA);
  await accion(page, "Abrir la bóveda").click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });

  // **Y ahora que se pide sola al llegar** (2.27.1), sin pulsar nada.
  //
  // Se comprueba **recargando** con la bóveda cerrada, y no mirando si se vuelve a
  // abrir al cerrarla: eso último no distingue «la huella la abrió» de «no llegó a
  // cerrarse», porque las dos cosas dejan la misma pantalla. Tras recargar, la
  // bóveda está cerrada sin lugar a dudas y nadie ha tocado nada.
  await page.request.get("/api/_llavero?dice=no");
  await accion(page, "Cerrar la bóveda").click();
  await expect(conElSistema).toBeVisible({ timeout: 20_000 });
  await page.request.get("/api/_llavero?dice=si");
  await page.reload();
  await seccion(page, "Bóveda").click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });

  // Y al quitarlo, la pantalla de desbloquear deja de ofrecerlo.
  await seccion(page, "Ajustes").click();
  await page.locator("#desbloqueo-del-sistema").click();
  await expect(page.locator("#desbloqueo-del-sistema")).not.toBeChecked({ timeout: 20_000 });
  await seccion(page, "Bóveda").click();
  await accion(page, "Cerrar la bóveda").click();
  await expect(page.locator("#boveda-con-el-sistema")).toHaveCount(0);

  // Se deja como estaba, que la bóveda sobrevive entre pruebas.
  await page.locator("#boveda-llave").fill(MAESTRA);
  await accion(page, "Abrir la bóveda").click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });

  // **Y la sugerencia no vuelve**, aunque el desbloqueo se haya quitado: se
  // ofreció y se contestó. Una tarjeta que reaparece cada vez que abres la bóveda
  // deja de ser una sugerencia y pasa a ser una insistencia.
  await expect(page.locator("#sugerencia-activar")).toHaveCount(0);

  // **Los 400 esperados se descuentan, y solo aquí.** En el servidor de
  // desarrollo *cualquier* error de Go vuelve como 400, así que cancelar la huella
  // —que es un final legítimo y la razón de ser de esta prueba— deja su línea en
  // la consola. En la ventana no pasa: allí no hay HTTP. Lo que no se descuenta es
  // ningún otro error.
  const inesperados = errores.filter((e) => !/status of 400/.test(e));
  expect(inesperados, inesperados.join(" | ")).toEqual([]);
});

/**
 * **Dos llaves de acceso del mismo sitio se distinguen** (ADR 0048, P3).
 *
 * Lo encontró el cliente en la primera prueba de verdad: creó una llave en GitHub,
 * abrió la bóveda y **había cuatro**, todas de GitHub y todas con la misma cuenta —las
 * tres primeras eran de los intentos que fallaron después de guardarse—. Y en pantalla
 * eran **cuatro filas idénticas**: el título es el nombre del sitio y la segunda línea,
 * la cuenta. No había ningún dato que dijera cuál era cuál.
 *
 * Eso en una contraseña es molesto; en una llave de acceso es otra cosa, porque **no se
 * puede recrear ni corregir**: borrar la que no es cuesta perder la forma de entrar.
 *
 * Lo único que las separa es cuándo se crearon, así que la hora tiene que estar. Se
 * comprueban **los dos sitios** —la lista y la ficha—, porque tener que abrir cuatro
 * entradas para compararlas no es distinguirlas.
 */
test("dos llaves del mismo sitio y la misma cuenta se distinguen por cuándo se crearon", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const sitio = `llaves-${Date.now()}.prueba`;
  for (const cual of ["una", "otra"]) {
    await page.evaluate(
      ([s, c]) =>
        fetch("/api/GuardarEnBoveda", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify([
            {
              tipo: "llave",
              titulo: s,
              rpId: s,
              idCredencial: `cred-${c}`,
              nombreVisible: "yo@ejemplo.com",
              usuario: "yo@ejemplo.com",
              algoritmo: -7,
              clavePrivada: `privada-${c}`,
            },
          ]),
        }),
      [sitio, cual],
    );
    // Un segundo entre las dos: las fechas de la bóveda tienen resolución de un
    // segundo, así que sin esperar saldrían **con la misma hora** y la prueba diría
    // que no se distinguen cuando lo que pasa es que se crearon a la vez.
    //
    // Y esta espera es la que destapó que la primera versión llegaba **solo hasta el
    // minuto**: las dos llaves caían en el mismo y salían idénticas, que es justo lo
    // que le pasó al cliente con cuatro intentos seguidos.
    await page.waitForTimeout(1100);
  }
  await page.reload();
  await conLaBovedaAbierta(page);
  await page.locator("#boveda-buscar").fill(sitio);

  const filas = page.locator(".panel:visible .lista-boveda li");
  await expect(filas).toHaveCount(2);
  // **Lo que de verdad se comprueba**: que las dos segundas líneas no dicen lo mismo.
  // Mirar solo que aparece una hora pasaría en verde con las dos idénticas.
  const notas = await filas.locator(".nota").allTextContents();
  expect(notas[0], "las dos filas dicen lo mismo: no hay forma de saber cuál es cuál").not.toBe(notas[1]);
  expect(notas[0]).toContain("yo@ejemplo.com");

  // Y en la ficha —la que se abre al pulsar la fila, no el editor— su propia línea.
  // La primera versión miraba el campo del editor y fallaba por eso: **el dato que
  // distingue tiene que estar donde se mira primero**, no detrás de otro clic.
  await filas.first().locator("button").click();
  const ficha = page.locator(".panel:visible");
  await expect(ficha.getByText("Creada", { exact: true })).toBeVisible();

  expect(errores, errores.join(" | ")).toEqual([]);
});

/**
 * **La ficha de una llave dice si el sitio la reconoce** (ADR 0048), que es lo que
 * distingue una huérfana de una que sirve.
 *
 * Se comprueban los dos estados, y el de «todavía no» es el que importa: una llave
 * recién creada está sin reconocer y eso es **lo normal**, así que el texto tiene que
 * decirlo como un hecho y no como una alarma. Enseñar solo la fecha cuando la hay, y
 * nada cuando no, dejaría la pregunta sin contestar justo en el caso que preocupa.
 */
test("la ficha de una llave dice si se ha usado y si el sitio la reconoce", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");

  /**
   * **Se comprueba que el guardado ha funcionado**, y no es celo: la primera versión
   * de esta prueba tiraba el `fetch` sin mirar la respuesta, pasaba sola y **fallaba en
   * la tanda** — otra prueba baja el reloj de bloqueo de la bóveda, así que para cuando
   * llegaba la segunda escritura la bóveda podía estar cerrada y `GuardarEnBoveda`
   * fallaba en silencio. El síntoma era «no encuentro este texto en la ficha», que no
   * se parece en nada a «no se guardó la entrada».
   */
  const guardar = async (entrada: Record<string, unknown>) => {
    // **Se recarga antes**, y eso tampoco es celo: `conLaBovedaAbierta` espera el
    // buscador o el botón de abrir, y **con una ficha abierta no hay ninguno de los
    // dos** — la ficha sustituye la lista—. El segundo guardado llegaba justo así y se
    // quedaba esperando veinte segundos a un buscador que no podía estar.
    await page.reload();
    await conLaBovedaAbierta(page);
    const r = await page.evaluate(
      (e) =>
        fetch("/api/GuardarEnBoveda", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify([e]),
        }).then((x) => x.status),
      entrada,
    );
    expect(r, "no se ha podido guardar la entrada de la prueba").toBe(200);
  };
  const abrirLaFicha = async (titulo: string) => {
    await page.reload();
    await conLaBovedaAbierta(page);
    await page.locator("#boveda-buscar").fill(titulo);
    const filas = page.locator(".panel:visible .lista-boveda li");
    await expect(filas, `la entrada «${titulo}» no está en la lista`).toHaveCount(1);
    await filas.first().locator("button").click();
    // Y que la ficha abierta es la suya, no la que hubiera antes.
    await expect(page.locator(".panel:visible").getByText(titulo, { exact: false }).first()).toBeVisible();
  };

  const sitio = `reconocida-${Date.now()}.prueba`;
  const base = {
    tipo: "llave", rpId: sitio, nombreVisible: "yo@ejemplo.com", algoritmo: -7, clavePrivada: "privada",
  };
  await guardar({ ...base, titulo: sitio, idCredencial: "cred-sin" });
  await abrirLaFicha(sitio);

  const ficha = page.locator(".panel:visible");
  // Sin ninguna de las dos señales: el aviso, y ninguno de los dos datos.
  await expect(ficha.getByText("Esfinge no ha apuntado ningún uso de esta llave", { exact: false })).toBeVisible();
  await expect(ficha.getByText("Reconocida por el sitio", { exact: true })).toHaveCount(0);
  await expect(ficha.getByText("Usada", { exact: true })).toHaveCount(0);

  // **Usada pero sin reconocer**, que es el caso corriente y la razón de que haya dos
  // datos: en el «entrar con llave de acceso» de un sitio que no nombra ninguna, esto es
  // lo único que se sabe. El aviso tiene que irse, porque ya se ha entrado con ella.
  const soloUsada = `${sitio}-usada`;
  await guardar({
    ...base, titulo: soloUsada, rpId: soloUsada, idCredencial: "cred-usada",
    usada: "2026-10-01T11:05:00Z",
  });
  await abrirLaFicha(soloUsada);
  await expect(ficha.getByText("Usada", { exact: true })).toBeVisible();
  await expect(ficha.getByText("Reconocida por el sitio", { exact: true })).toHaveCount(0);
  await expect(ficha.getByText("Esfinge no ha apuntado ningún uso de esta llave", { exact: false })).toHaveCount(0);

  // Y con las dos, las dos: no se sustituyen, dicen cosas distintas.
  const conFecha = `${sitio}-ok`;
  await guardar({
    ...base, titulo: conFecha, rpId: conFecha, idCredencial: "cred-con",
    confirmada: "2026-10-01T09:30:00Z", usada: "2026-10-01T11:06:00Z",
  });
  await abrirLaFicha(conFecha);
  await expect(ficha.getByText("Reconocida por el sitio", { exact: true })).toBeVisible();
  await expect(ficha.getByText("Usada", { exact: true })).toBeVisible();
  await expect(ficha.getByText("Esfinge no ha apuntado ningún uso de esta llave", { exact: false })).toHaveCount(0);

  expect(errores, errores.join(" | ")).toEqual([]);
});

/**
 * **Una red wifi guarda sus campos y la ficha dibuja su código** (ADR 0049).
 *
 * Lo que de verdad se comprueba aquí es que el código **llega y se dibuja**: que tenga
 * tantos módulos como dice su lado y que no sea un cuadro vacío. Si lo que dibuja se
 * puede escanear no lo dice ninguna prueba de aquí —eso lo cerró un móvil—, pero que la
 * matriz cruce el puente entera y la ficha la pinte, sí.
 */
test("una red wifi guarda su nombre y la ficha dibuja su código", async ({ page }) => {
  const errores: string[] = [];
  page.on("pageerror", (e) => errores.push(String(e)));
  // El modo local lo elige el beforeEach de arriba, como en todas las demás.
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const dentro = page.locator(".contenido");
  await dentro.getByRole("button", { name: "Nueva", exact: true }).click();
  await dentro.getByRole("tab", { name: "Wi-Fi", exact: true }).click();
  await page.locator("#boveda-titulo").fill("La oficina");
  await page.locator("#boveda-ssid").fill("WEBCAFEINA");
  await page.locator("#boveda-clave-wifi").fill("una-clave-de-prueba");
  await page.locator("#boveda-oculta").check();
  await dentro.getByRole("button", { name: "Guardar", exact: true }).click();

  await page.locator("#boveda-buscar").fill("WEBCAFEINA");
  await page.locator(".panel:visible .lista-boveda li").first().locator("button").click();

  const ficha = page.locator(".panel:visible");
  await expect(ficha.getByText("Nombre de la red", { exact: true })).toBeVisible();
  await expect(ficha.getByText("WEBCAFEINA", { exact: true })).toBeVisible();
  await expect(ficha.getByText("WPA / WPA2 / WPA3", { exact: true })).toBeVisible();
  await expect(ficha.getByText("Sí, no anuncia su nombre", { exact: true })).toBeVisible();

  // El código: un SVG con sus módulos dentro, y el aviso de lo que es.
  const codigo = ficha.locator(".codigo-wifi svg");
  await expect(codigo).toBeVisible();
  const cuantos = await codigo.locator("rect").count();
  // Uno es el fondo blanco; los demás son módulos oscuros. Un código de red tiene
  // cientos, así que con exigir más de cien se distingue de un cuadro vacío sin atarse
  // a una matriz concreta.
  expect(cuantos, "el código ha salido sin módulos").toBeGreaterThan(100);
  await expect(ficha.getByText("Este dibujo es la contraseña", { exact: false })).toBeVisible();

  expect(errores, errores.join(" | ")).toEqual([]);
});

/**
 * **El apartado «Proyectos»: crear una bóveda de cliente, entrar y volver** (ADR 0050).
 *
 * Lo que de verdad se comprueba aquí, y no es que los botones existan:
 *
 *   - que al entrar en un proyecto **lo que se ve dentro es lo suyo** y no lo de la
 *     bóveda personal, que es lo único que separa de verdad una bóveda de otra;
 *   - que la barra de herramientas dice **en qué bóveda se trabaja**, porque con varias
 *     «Bóveda» a secas ya no identifica nada;
 *   - y que la pantalla dice las dos cosas que no se ven mirándola: que **no hay clave de
 *     recuperación propia** y que **esto todavía no se sincroniza**.
 */
test("una bóveda de proyecto: se crea, se entra y lo de dentro es lo suyo", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  // Una entrada en la bóveda personal, para poder comprobar después que no se ve
  // desde el proyecto.
  const mia = `Mi banco ${Date.now()}`;
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(mia);
  await page.locator("#boveda-secreto").fill("la mia");
  await accion(page, "Guardar").click();
  await expect(page.locator(".lista-boveda").getByRole("button", { name: mia })).toBeVisible({
    timeout: 20_000,
  });

  await seccion(page, "Proyectos").click();
  const panel = page.locator(".panel:visible");

  // **Lo que todavía no hace, dicho donde se decide guardar algo.**
  await expect(panel.getByText("todavía no se sincronizan", { exact: false })).toBeVisible({
    timeout: 20_000,
  });

  const nombre = `Acme ${Date.now()}`;
  await accion(page, "Nueva bóveda de proyecto").click();

  // **La ausencia de la ceremonia, explicada antes de crear.** Quien ha creado una
  // bóveda antes espera la clave de recuperación a pantalla entera.
  await expect(panel.getByText("no tiene clave de recuperación propia", { exact: false })).toBeVisible();

  await page.locator("#proyecto-nombre").fill(nombre);
  await accion(page, "Crear").click();

  const fila = panel.locator(".proyectos").getByRole("button", { name: new RegExp(nombre) });
  await expect(fila).toBeVisible({ timeout: 20_000 });
  // Recién creada no se ha abierto nunca, y la lista lo dice en vez de dejar el hueco.
  await expect(fila).toContainText("Sin abrir todavía");

  // Se entra: conmuta y lleva a la bóveda.
  await fila.click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });

  // **Y la barra de herramientas dice cuál es.**
  await expect(page.locator(".herramientas")).toContainText(nombre, { timeout: 20_000 });

  // **Lo de la bóveda personal no está aquí.** Es la comprobación que importa: sin
  // ella, «se ha abierto otra bóveda» podría ser la misma con otro rótulo.
  await page.locator("#boveda-buscar").fill(mia);
  await expect(page.locator(".panel:visible").getByRole("button", { name: mia })).toHaveCount(0);

  // Se guarda algo que es de este proyecto.
  await page.locator("#boveda-buscar").fill("");
  const deAcme = `Hosting de Acme ${Date.now()}`;
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(deAcme);
  await page.locator("#boveda-secreto").fill("la de acme");
  await accion(page, "Guardar").click();
  await expect(page.locator(".lista-boveda").getByRole("button", { name: deAcme })).toBeVisible({
    timeout: 20_000,
  });

  // Se vuelve a la bóveda personal. **Pide la maestra otra vez**, a propósito: la
  // personal no se queda abierta por detrás.
  await seccion(page, "Proyectos").click();
  await expect(panel.locator(".proyectos").getByRole("button", { name: new RegExp(nombre) })).toContainText(
    "La estás usando",
  );
  await volverALaPersonal(page);

  expect(errores, errores.join(" | ")).toEqual([]);
});

/**
 * **Una prueba que conmuta de bóveda tiene que dejar abierta la de siempre.**
 *
 * El Go es uno y la bóveda sigue abierta entre pruebas y **entre temas**: una que
 * acabe dentro de un proyecto deja a las siguientes mirando otra bóveda, y lo que
 * se ve entonces son cuatrocientos en la consola y dos pruebas de la lista cayendo
 * sin relación aparente. Es la misma trampa que ya costó una publicación con el
 * `test.skip` de las capturas.
 */
async function volverALaPersonal(page: Page) {
  await seccion(page, "Proyectos").click();
  const panel = page.locator(".panel:visible");
  // Si no hay ninguna activa, ya estamos en la personal.
  if ((await panel.locator('.proyectos li[data-activo="si"]').count()) === 0) return;
  await seccion(page, "Bóveda").click();
  await accion(page, "Cerrar la bóveda").click();
  await conLaBovedaAbierta(page);
}

/**
 * Y lo que la lista tiene que decir cuando **no se puede hacer nada**: con la bóveda
 * cerrada no hay nombres que enseñar, porque viven dentro de ella.
 */
test("con la bóveda cerrada, Proyectos dice qué hacer y no enseña ningún nombre", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  // Se cierra a mano, que es lo que hace el botón de la bóveda.
  await accion(page, "Cerrar la bóveda").click();

  await seccion(page, "Proyectos").click();
  const panel = page.locator(".panel:visible");
  await expect(panel.getByText("Abre tu bóveda para ver tus proyectos", { exact: false })).toBeVisible({
    timeout: 20_000,
  });
  // Y no hay lista: no es que esté vacía, es que no se puede leer.
  await expect(panel.locator(".proyectos")).toHaveCount(0);

  expect(errores, errores.join(" | ")).toEqual([]);
});

/**
 * **Llevar una entrada a otra bóveda** (ADR 0050, E3).
 *
 * Lo que se comprueba es el camino entero y por los dos lados: que sale de donde
 * estaba, que **aparece en la otra con su contraseña**, y que aquí queda en la
 * papelera en vez de borrarse. Sin lo del medio, «se movió» podría ser «se perdió».
 */
test("una entrada se lleva a la bóveda de un proyecto, con su contraseña", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const proyecto = `Beta ${Date.now()}`;
  await seccion(page, "Proyectos").click();
  await accion(page, "Nueva bóveda de proyecto").click();
  await page.locator("#proyecto-nombre").fill(proyecto);
  await accion(page, "Crear").click();
  await expect(
    page.locator(".panel:visible .proyectos").getByRole("button", { name: new RegExp(proyecto) }),
  ).toBeVisible({ timeout: 20_000 });

  // Una entrada en la bóveda personal.
  await seccion(page, "Bóveda").click();
  const titulo = `Servidor de Beta ${Date.now()}`;
  await accion(page, "Nueva").click();
  await page.locator("#boveda-titulo").fill(titulo);
  await page.locator("#boveda-usuario").fill("root");
  await page.locator("#boveda-secreto").fill("la-del-servidor");
  await accion(page, "Guardar").click();

  const fila = page.locator(".lista-boveda").getByRole("button", { name: titulo });
  await expect(fila).toBeVisible({ timeout: 20_000 });
  await fila.click();

  await accion(page, "Llevar a otra bóveda").click();
  const destino = page.locator(".panel:visible .proyectos li").filter({ hasText: proyecto });
  await expect(destino).toBeVisible({ timeout: 20_000 });
  await destino.getByRole("button", { name: "Mover", exact: true }).click();

  // Ya no está aquí viva, **y lo que se dice es adónde ha ido y qué queda**.
  await expect(page.locator(".panel:visible")).toContainText("papelera", { timeout: 20_000 });
  await page.locator("#boveda-buscar").fill(titulo);
  await expect(page.locator(".lista-boveda").getByRole("button", { name: titulo })).toHaveCount(0);

  // Y en el proyecto está, **con su contraseña**: eso es lo que separa moverla de
  // perderla.
  await seccion(page, "Proyectos").click();
  await page
    .locator(".panel:visible .proyectos")
    .getByRole("button", { name: new RegExp(proyecto) })
    .click();
  await expect(page.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });
  const llegada = page.locator(".lista-boveda").getByRole("button", { name: titulo });
  await expect(llegada).toBeVisible({ timeout: 20_000 });
  await llegada.click();
  await accion(page, "Ver").first().click();
  await expect(page.locator(".panel:visible")).toContainText("la-del-servidor", { timeout: 20_000 });

  await volverALaPersonal(page);
  expect(errores, errores.join(" | ")).toEqual([]);
});

/**
 * **Entregar un proyecto, archivarlo y borrarlo** (ADR 0050 E6, ADR 0051).
 *
 * Las tres van detrás de «Al acabar…» porque se usan una vez en la vida de un
 * proyecto. Lo que se comprueba de cada una es lo que la distingue: entregar
 * **enseña la clave de recuperación una vez**, archivar **se lleva el fichero de
 * este equipo** y borrar **pide la contraseña maestra**.
 */
test("al acabar un proyecto: entregarlo, archivarlo y borrarlo", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const nombre = `Gamma ${Date.now()}`;
  await seccion(page, "Proyectos").click();
  await accion(page, "Nueva bóveda de proyecto").click();
  await page.locator("#proyecto-nombre").fill(nombre);
  await accion(page, "Crear").click();

  const panel = page.locator(".panel:visible");
  const fila = panel.locator(".proyectos li").filter({ hasText: nombre });
  await expect(fila).toBeVisible({ timeout: 20_000 });

  // --- Entregar: la ceremonia de la clave de recuperación, una vez ---
  await fila.getByRole("button", { name: "Al acabar…" }).click();
  await panel.getByRole("button", { name: "Entregársela al cliente" }).click();
  // **Se dice que la contraseña va por otro camino**, que es lo que convierte
  // entregar un fichero en entregarlo bien.
  await expect(panel.getByText("Dísela por otro camino", { exact: false })).toBeVisible();
  await panel.locator(`input[type="password"]`).first().fill("la contraseña del cliente");
  await accion(page, "Elegir dónde guardarla").click();

  const clave = page.locator(".clave-recuperacion");
  await expect(clave).toBeVisible({ timeout: 20_000 });
  await expect(clave).toHaveText(/^ESF(-[0-9A-HJKMNP-TV-Z]{4})+$/);
  const seguir = accion(page, "Continuar");
  await expect(seguir).toBeDisabled();
  await page.getByText("La he apuntado en un sitio seguro").click();
  await seguir.click();

  // Y el proyecto sigue aquí: entregar es dar una copia.
  await expect(panel.locator(".proyectos li").filter({ hasText: nombre })).toBeVisible({ timeout: 20_000 });

  // --- Archivar: se lleva el fichero y se va de la lista del día a día ---
  await panel.locator(".proyectos li").filter({ hasText: nombre }).getByRole("button", { name: "Al acabar…" }).click();
  await panel.getByRole("button", { name: "Archivarla" }).click();
  await expect(panel.locator(".proyectos li").filter({ hasText: nombre })).toHaveCount(0, { timeout: 20_000 });
  await panel.getByRole("button", { name: /^Archivados/ }).click();
  const archivada = panel.locator(".proyectos li").filter({ hasText: nombre });
  await expect(archivada).toBeVisible();
  // Y dice que ya no está aquí, sin llamarlo error.
  await expect(archivada).toContainText("Dormido en este equipo");

  expect(errores, errores.join(" | ")).toEqual([]);
});

/** Y borrar **pide la contraseña maestra**: es lo único irreversible que hay aquí. */
test("borrar una bóveda de proyecto pide la contraseña maestra", async ({ page }) => {
  const errores = vigilarConsola(page);
  await page.goto("/");
  await conLaBovedaAbierta(page);

  const nombre = `Delta ${Date.now()}`;
  await seccion(page, "Proyectos").click();
  await accion(page, "Nueva bóveda de proyecto").click();
  await page.locator("#proyecto-nombre").fill(nombre);
  await accion(page, "Crear").click();

  const panel = page.locator(".panel:visible");
  const fila = panel.locator(".proyectos li").filter({ hasText: nombre });
  await expect(fila).toBeVisible({ timeout: 20_000 });
  await fila.getByRole("button", { name: "Al acabar…" }).click();
  await panel.getByRole("button", { name: "Borrarla" }).click();

  // **Se dice que no se puede deshacer antes de pedir nada.**
  await expect(panel.getByText("Esto no se puede deshacer", { exact: false })).toBeVisible();

  // Con una contraseña que no es, no se borra y se dice.
  await panel.locator(`input[type="password"]`).first().fill("esa no es la maestra");
  await accion(page, "Borrarla para siempre").click();
  await expect(panel.locator(".error")).toBeVisible({ timeout: 20_000 });
  await expect(panel.locator(".proyectos li").filter({ hasText: nombre })).toBeVisible();
  // Ese rechazo es lo que la prueba venía a provocar: el 400 que deja en la consola
  // es la respuesta correcta, no un fallo.
  olvidarEsperado(errores, /BorrarProyecto|Failed to load resource/);

  // Con la buena, se va de la lista.
  await panel.locator(`input[type="password"]`).first().fill(MAESTRA);
  await accion(page, "Borrarla para siempre").click();
  await expect(panel.locator(".proyectos li").filter({ hasText: nombre })).toHaveCount(0, { timeout: 20_000 });

  expect(errores, errores.join(" | ")).toEqual([]);
});
