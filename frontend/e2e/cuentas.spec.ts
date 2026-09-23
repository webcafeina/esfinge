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
  test.setTimeout(180_000);
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
  // Con cuenta, una contraseña débil no vale.
  await a.locator("#cuenta-maestra").fill("corta");
  await a.locator("#cuenta-maestra-2").fill("corta");
  // El botón no se deja pulsar: el medidor dice que no llega. Go lo comprueba
  // también (internal/app), por si algo llegara sin pasar por aquí.
  await expect(accion(a, "Crear la cuenta")).toBeDisabled();
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
  // **El estado lo pone Go y la línea solo lo pinta**, así que se mira primero lo
  // que dice Go: cuando esto falló en la máquina de GitHub —y aquí nunca—, lo
  // único que quedaba era «Sincronizando…», que no dice si la pasada falló, se
  // canceló o sigue en marcha (2.25.4).
  await expect
    .poll(
      async () => {
        const r = await request.post(`${A}/api/EstadoDeCuenta`, { data: [] });
        return ((await r.json()) as { sincro: unknown }).sincro;
      },
      // **Más de lo que tarda la red en rendirse** (el cliente de la cuenta espera
      // como mucho un minuto). Así, si esto vuelve a fallar, el estado que salga
      // dice de qué se trata: «al-dia» tarde es lentitud de la máquina; «sin
      // conexión» o «error» es una petición que se colgó y lo cuenta; y seguir en
      // «sincronizando» es una pasada que se canceló sin decir nada.
      { timeout: 70_000 },
    )
    .toMatchObject({ estado: "al-dia" });
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

  // **Con las dos bóvedas abiertas**, lo que guarda un equipo aparece en el otro
  // sin cerrar ni volver a abrir nada: la lista se refresca sola al llegar.
  await accion(a, "Abrir la bóveda").isVisible();
  await a.locator("#boveda-llave").fill(MAESTRA);
  await accion(a, "Abrir la bóveda").click();
  await expect(a.locator("#boveda-buscar")).toBeVisible({ timeout: 20_000 });
  await accion(a, "Nueva").click();
  // La hora de guardar, sin milisegundos: la de la sincronización va al segundo.
  const guardadoEn = new Date(Math.floor(Date.now() / 1000) * 1000 + 1000).toISOString().replace(".000Z", "Z");
  await a.locator("#boveda-titulo").fill("Llega sin reabrir");
  await accion(a, "Guardar").click();
  await expect(a.locator(".lista-boveda").getByRole("button", { name: "Llega sin reabrir" })).toBeVisible();
  // Primero que A lo haya subido: guarda y espera tres segundos para juntar los
  // guardados seguidos. Sin esperar, B preguntaba antes de que hubiera nada y la
  // siguiente pasada tocaba al minuto (lo vio la máquina de GitHub, no ésta).
  await expect
    .poll(async () => {
      const r = await request.post(`${A}/api/EstadoDeCuenta`, { data: [] });
      const e = (await r.json()) as { sincro: { estado: string; ultima?: string } };
      return e.sincro.estado === "al-dia" && (e.sincro.ultima ?? "") >= guardadoEn;
    }, { timeout: 20_000 })
    .toBe(true);
  // Y volver a la ventana de B basta: pide una pasada sola.
  await b.bringToFront();
  await b.evaluate(() => window.dispatchEvent(new Event("focus")));
  await expect(b.locator(".lista-boveda").getByRole("button", { name: "Llega sin reabrir" })).toBeVisible({
    timeout: 30_000,
  });

  // Y con el botón, sin esperar a la pasada de cada minuto ni volver a la ventana
  // (2.25.2). Primero que A lo haya subido, como arriba.
  const conElBoton = new Date(Math.floor(Date.now() / 1000) * 1000 + 1000).toISOString().replace(".000Z", "Z");
  await accion(a, "Nueva").click();
  await a.locator("#boveda-titulo").fill("Llega con el botón");
  await accion(a, "Guardar").click();
  await expect
    .poll(async () => {
      const r = await request.post(`${A}/api/EstadoDeCuenta`, { data: [] });
      const e = (await r.json()) as { sincro: { estado: string; ultima?: string } };
      return e.sincro.estado === "al-dia" && (e.sincro.ultima ?? "") >= conElBoton;
    }, { timeout: 20_000 })
    .toBe(true);
  const botonSincro = b.locator(".linea-sincro").getByRole("button", { name: "Sincronizar ahora" });
  await botonSincro.click();
  // Las flechas giran hasta que llega el resultado, y paran (2.25.2).
  await expect(botonSincro).toHaveClass(/girando/);
  await expect(b.locator(".lista-boveda").getByRole("button", { name: "Llega con el botón" })).toBeVisible({
    timeout: 8_000,
  });
  await expect(botonSincro).not.toHaveClass(/girando/, { timeout: 8_000 });
  await retratar(b, "boveda-con-sincronizar");
  // Y el candado de la barra lateral dice que está abierta.
  await expect(b.locator(".lateral .estado-boveda")).toHaveAttribute("data-abierta", "si");

  // Con cuenta y la bóveda cerrada, la pantalla ofrece entrar con una contraseña
  // cambiada en otro equipo y recuperar la cuenta.
  await accion(a, "Cerrar la bóveda").click();
  await expect(accion(a, "¿Cambiaste la contraseña en otro equipo?")).toBeVisible();
  await accion(a, "¿La has olvidado?").click();
  await expect(a.getByRole("heading", { name: "Recuperar tu cuenta" })).toBeVisible();
  await expect(a.locator("#recuperar-correo")).toHaveValue(correo);
  await retratar(a, "recuperar");
  await accion(a, "Volver").click();
  await expect(accion(a, "Abrir la bóveda")).toBeVisible();

  // B cambia la contraseña, y en A, cerrado, **basta escribir la nueva** en «Abrir
  // la bóveda»: entra en la cuenta sin código y se pone al día (2.24.1).
  const NUEVA = MAESTRA + " cambiada en B";
  const cambiada = await request.post(`${B}/api/CambiarMaestraDeBoveda`, { data: [MAESTRA, NUEVA] });
  expect(cambiada.ok(), await cambiada.text()).toBe(true);
  await a.locator("#boveda-llave").fill(NUEVA);
  await accion(a, "Abrir la bóveda").click();
  await expect(a.locator(".lista-boveda").getByRole("button", { name: "Llega sin reabrir" })).toBeVisible({
    timeout: 20_000,
  });
  await expect(a.locator(".linea-sincro")).toContainText("Sincronizada", { timeout: 20_000 });

  // Y si A ya no es de confianza —aquí porque B lo olvida; en la vida, porque
  // caducó a los 90 días—, la nueva no se toma por mala: pide el código del correo
  // en la misma pantalla, y con él abre.
  await accion(a, "Cerrar la bóveda").click();
  const equipos = (await (await request.post(`${B}/api/DispositivosDeCuenta`, { data: [] })).json()) as {
    id: string;
    actual: boolean;
  }[];
  const deA = equipos.find((e) => !e.actual);
  expect(deA).toBeTruthy();
  expect((await request.post(`${B}/api/OlvidarDispositivo`, { data: [deA!.id] })).ok()).toBe(true);
  const OTRA = NUEVA + " y otra vez";
  const otraVez = await request.post(`${B}/api/CambiarMaestraDeBoveda`, { data: [NUEVA, OTRA] });
  expect(otraVez.ok(), await otraVez.text()).toBe(true);
  await a.locator("#boveda-llave").fill(OTRA);
  await accion(a, "Abrir la bóveda").click();
  await expect(a.getByText("Es la contraseña nueva de tu cuenta.")).toBeVisible({ timeout: 20_000 });
  await retratar(a, "abrir-con-codigo");
  // Esa respuesta es un error a propósito —«falta el código»—, y es la única.
  expect(erroresA.splice(erroresA.indexOf("400 /api/AbrirBoveda"), 1)).toEqual(["400 /api/AbrirBoveda"]);
  await a.locator("#abrir-codigo").fill(await codigo(request, correo));
  await accion(a, "Abrir la bóveda").click();
  await expect(a.locator(".lista-boveda").getByRole("button", { name: "Llega sin reabrir" })).toBeVisible({
    timeout: 20_000,
  });
  expect(erroresA, erroresA.join(" | ")).toEqual([]);

  // Y si B olvida a A con la bóveda abierta, a A se le cierra sola y dice por qué
  // (2.24.5): olvidar es sobre todo para un equipo perdido.
  const deNuevo = (await (await request.post(`${B}/api/DispositivosDeCuenta`, { data: [] })).json()) as {
    id: string;
    actual: boolean;
  }[];
  const aOtraVez = deNuevo.find((e) => !e.actual);
  expect((await request.post(`${B}/api/OlvidarDispositivo`, { data: [aOtraVez!.id] })).ok()).toBe(true);
  await request.post(`${A}/api/SincronizarAhora`, { data: [] });
  await expect(accion(a, "Abrir la bóveda")).toBeVisible({ timeout: 20_000 });
  await expect(a.getByText("tu cuenta ya no reconoce este equipo", { exact: false })).toBeVisible();
  await expect(a.locator(".lateral .estado-boveda")).toHaveAttribute("data-abierta", "no");
  await retratar(a, "cerrada-por-olvido");
  expect(erroresA, erroresA.join(" | ")).toEqual([]);

  // Y Ajustes cuenta en qué cuenta está este equipo.
  await b.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
  await expect(b.getByRole("heading", { name: "Cuenta y sincronización" })).toBeVisible();
  await expect(b.getByText(correo)).toBeVisible();
  // El equipo de ahora, marcado; y exportar lo que hay de la cuenta.
  // Y el canal con el navegador, que con cuenta sobra: sigue estando —en local es
  // la única forma— y lo dice donde se ve (2.25.4).
  await expect(b.getByLabel("Dejar que la extensión del navegador consulte la bóveda")).not.toBeChecked();
  await expect(b.getByText("Con cuenta no hace falta.")).toBeVisible();
  await b.getByText("Con cuenta no hace falta.").scrollIntoViewIfNeeded();
  await retratar(b, "ajustes-canal-con-cuenta");

  await expect(b.getByRole("heading", { name: "Equipos con tu cuenta" })).toBeVisible();
  // Solo B: a A lo acaba de olvidar.
  await expect(b.locator(".lista-equipos li")).toHaveCount(1);
  await expect(b.locator(".lista-equipos")).toContainText("Este equipo");
  await retratar(b, "ajustes-cuenta");
  await b.getByRole("heading", { name: "Equipos con tu cuenta" }).scrollIntoViewIfNeeded();
  await retratar(b, "ajustes-equipos");
  await accion(b, "Exportar los datos de la cuenta…").click();
  await expect(b.getByText(/^Guardado en /)).toBeVisible();
  await b.getByRole("heading", { name: "Tus datos en el servidor" }).scrollIntoViewIfNeeded();
  await retratar(b, "ajustes-cuenta-datos");

  // Y se puede cambiar de idea: dejar la cuenta en este equipo, con la contraseña.
  await accion(b, "Dejar la cuenta en este equipo…").click();
  await b.locator("#salir-maestra").fill(OTRA);
  await retratar(b, "ajustes-salir");
  await accion(b, "Dejar la cuenta en este equipo").click();
  await expect(accion(b, "Crear una cuenta")).toBeVisible();
  await expect(b.getByText("Tu bóveda está solo en este ordenador.")).toBeVisible();
  expect(erroresB, erroresB.join(" | ")).toEqual([]);

  // ---------------------------------------------------------------- una maestra floja
  // B vuelve a estar en local. Con una contraseña que no llega a «Buena», crear una
  // cuenta nueva la pide nueva en el mismo paso, sin salir del asistente.
  const FLOJA = "contrasena1";
  const cambio = await request.post(`${B}/api/CambiarMaestraDeBoveda`, { data: [OTRA, FLOJA] });
  expect(cambio.ok(), await cambio.text()).toBe(true);
  const otroCorreo = `floja-${Date.now()}@ejemplo.com`;
  await accion(b, "Crear una cuenta").click();
  await b.locator("#cuenta-correo").fill(otroCorreo);
  await accion(b, "Mandarme el código").click();
  await expect(b.locator("#cuenta-codigo")).toBeVisible({ timeout: 20_000 });
  await b.locator("#cuenta-codigo").fill(await codigo(request, otroCorreo));
  await b.locator("#cuenta-maestra").fill(FLOJA);
  await expect(b.getByText("Tu contraseña de ahora no llega a «Buena»")).toBeVisible();
  await expect(accion(b, "Crear la cuenta")).toBeDisabled();
  await b.locator("#cuenta-maestra-nueva").fill(MAESTRA + " nueva");
  await b.locator("#cuenta-maestra-nueva-2").fill(MAESTRA + " nueva");
  await retratar(b, "cuenta-maestra-floja");
  await accion(b, "Crear la cuenta").click();
  // Al terminar, a la bóveda, sincronizada; y abre con la nueva.
  await expect(b.locator(".linea-sincro")).toContainText("Sincronizada", { timeout: 20_000 });
  // Se cierra con el botón, como lo haría una persona: cerrarla por detrás justo
  // al montarse la lista dejaba su primera búsqueda en el aire (lo vio la máquina
  // de GitHub con la 2.24.4).
  await accion(b, "Cerrar la bóveda").click();
  await expect(accion(b, "Abrir la bóveda")).toBeVisible();
  const conLaNueva = await request.post(`${B}/api/AbrirBoveda`, { data: [MAESTRA + " nueva"] });
  expect(conLaNueva.ok(), await conLaNueva.text()).toBe(true);
  expect(erroresB, erroresB.join(" | ")).toEqual([]);
});
