import { chromium, expect, test, type BrowserContext, type Page } from "@playwright/test";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { Boveda } from "../src/nucleo/boveda";
import { Cliente } from "../src/nucleo/cliente";
import { derivarAcceso } from "../src/nucleo/cuenta";
import { azarDe, base64url, desdeBase64, PERFIL_INTERACTIVO } from "../src/nucleo/esf1";
import { abrirEnvio, mandarEntrada, type Envio } from "../src/nucleo/envio";
import { identidadDeSemilla } from "../src/nucleo/identidad";
import type { Entrada } from "../src/nucleo/entrada";

/**
 * La extensión con cuenta, **cargada de verdad** (ADR 0040), sin la aplicación: entra
 * con su código, rellena, recibe lo de otro equipo, guarda y lo sube, se bloquea y
 * se desbloquea, y se cierra cuando la cuenta deja de reconocerla.
 *
 * «El otro equipo» es esta misma prueba hablando con el servidor con el núcleo de la
 * extensión, que es el mismo formato que la aplicación (lo vigilan las pruebas
 * cruzadas de Go). Las páginas `https` las sirve Playwright: el guion de la
 * extensión se pone igual que en una web de verdad.
 */

const EXTENSION = fileURLToPath(new URL("../dist/pruebas", import.meta.url));
const SERVIDOR = "http://127.0.0.1:8793";
const MAESTRA = "una maestra larga para la extensión con cuenta";
const CORREO = `extension-${Date.now()}@ejemplo.com`;
const CORREO_QUE_MANDA = `manda-${Date.now()}@ejemplo.com`;
const MAESTRA_QUE_MANDA = "la maestra larga de quien manda la copia";

let contexto: BrowserContext;
let id: string;
/** La sesión del «otro equipo», que es quien crea la cuenta. */
let otro: { token: string };
/** La otra cuenta, la de quien manda y recibe copias. */
let manda: Awaited<ReturnType<typeof quienManda>>;

// ------------------------------------------------------------------ el otro equipo

async function codigoDelBuzon(correo: string): Promise<string> {
  const r = await fetch(`${SERVIDOR}/_pruebas/buzon?correo=${encodeURIComponent(correo)}`);
  const { mensajes } = (await r.json()) as { mensajes: { cuerpo: string }[] };
  const m = /^\s+(\d{6})$/m.exec(mensajes[0]?.cuerpo ?? "");
  if (!m) throw new Error(`No ha llegado ningún código a ${correo}`);
  return m[1];
}

const credencial = (titulo: string, usuario: string, secreto: string, sitio: string): Entrada => ({
  id: "",
  tipo: "credencial",
  titulo,
  usuario,
  secreto,
  sitios: [sitio],
  creada: "",
  cambiada: "",
});

/** Da de alta la cuenta como lo hace la aplicación, con una bóveda y una cuenta dentro. */
async function crearCuenta(): Promise<{ token: string }> {
  await fetch(`${SERVIDOR}/v1/registro/inicio`, { method: "POST", body: JSON.stringify({ correo: CORREO }) });
  const codigo = await codigoDelBuzon(CORREO);
  const sal = azarDe(16);
  const argon2 = { memoria: PERFIL_INTERACTIVO.memoria, pasadas: PERFIL_INTERACTIVO.pasadas, paralelismo: PERFIL_INTERACTIVO.paralelismo };
  const clave = await derivarAcceso(MAESTRA, sal, argon2);
  const { boveda } = await Boveda.crear(MAESTRA);
  await boveda.poner(credencial("Sitio", "yo@sitio.prueba", "clave-del-sitio", "https://sitio.prueba"));
  const r = await fetch(`${SERVIDOR}/v1/registro/fin`, {
    method: "POST",
    body: JSON.stringify({
      correo: CORREO,
      codigo,
      sal: base64url(sal),
      argon2,
      claveDeAcceso: base64url(clave),
      posesion: base64url(await boveda.posesion()),
      dispositivo: "El otro equipo",
      confiar: true,
    }),
  });
  if (!r.ok) throw new Error(`Alta: ${r.status} ${await r.text()}`);
  const { sesion } = (await r.json()) as { sesion: string };
  await new Cliente(SERVIDOR).subir(sesion, 0, (await boveda.prepararSubida(1)).texto);
  return { token: sesion };
}

/** Cambia la bóveda de la cuenta desde «el otro equipo». */
async function desdeElOtroEquipo(cambiar: (b: Boveda) => Promise<void>) {
  const c = new Cliente(SERVIDOR);
  const bajada = (await c.bajar(otro.token, 0))!;
  const b = await Boveda.abrir(bajada.datos, MAESTRA);
  await cambiar(b);
  await c.subir(otro.token, bajada.version, (await b.prepararSubida(bajada.version + 1)).texto);
}

/**
 * Otra cuenta, la de quien manda la copia (ADR 0043).
 *
 * Se da de alta como cualquiera y **publica sus llaves**, que es lo que hace que
 * la huella del buzón sea comparable: el correo de quien manda no viaja dentro
 * del sobre, así que lo único que se puede cotejar con la otra persona es esto.
 */
async function quienManda(correo = CORREO_QUE_MANDA, maestra = MAESTRA_QUE_MANDA) {
  await fetch(`${SERVIDOR}/v1/registro/inicio`, { method: "POST", body: JSON.stringify({ correo }) });
  const codigo = await codigoDelBuzon(correo);
  const sal = azarDe(16);
  const argon2 = { memoria: PERFIL_INTERACTIVO.memoria, pasadas: PERFIL_INTERACTIVO.pasadas, paralelismo: PERFIL_INTERACTIVO.paralelismo };
  const clave = await derivarAcceso(maestra, sal, argon2);
  const { boveda } = await Boveda.crear(maestra);
  const r = await fetch(`${SERVIDOR}/v1/registro/fin`, {
    method: "POST",
    body: JSON.stringify({
      correo,
      codigo,
      sal: base64url(sal),
      argon2,
      claveDeAcceso: base64url(clave),
      posesion: base64url(await boveda.posesion()),
      dispositivo: `Equipo de ${correo}`,
      confiar: true,
    }),
  });
  if (!r.ok) throw new Error(`Alta de ${correo}: ${r.status} ${await r.text()}`);
  const { sesion } = (await r.json()) as { sesion: string };
  const cliente = new Cliente(SERVIDOR);
  const semilla = await boveda.semillaDeIdentidad();
  const yo = await identidadDeSemilla(semilla);
  await cliente.publicarLlaves(sesion, { suite: yo.suite, cifrado: base64url(yo.cifrado), firma: base64url(yo.firma) });
  return {
    huella: yo.huella,
    /** Lo que espera en su buzón, ya abierto: es lo que prueba que le llegó de verdad. */
    async buzon() {
      const envios = await cliente.buzon(sesion);
      return Promise.all(envios.map(async (x) => (await abrirEnvio(semilla, x.sobre as Envio)).entrada));
    },
    async mandar(entrada: Entrada, a: string) {
      const l = await cliente.llavesDe(sesion, a);
      const sobre = await mandarEntrada(semilla, entrada, {
        suite: l.suite,
        cifrado: desdeBase64(l.cifrado),
        firma: desdeBase64(l.firma),
        huella: "",
      });
      await cliente.mandar(sesion, a, sobre);
    },
  };
}

async function titulosEnElServidor(): Promise<string[]> {
  const bajada = (await new Cliente(SERVIDOR).bajar(otro.token, 0))!;
  return (await Boveda.abrir(bajada.datos, MAESTRA)).buscar("").map((e) => e.titulo).sort();
}

// ------------------------------------------------------------------ el navegador

test.beforeAll(async () => {
  otro = await crearCuenta();
  manda = await quienManda();
  contexto = await chromium.launchPersistentContext(mkdtempSync(join(tmpdir(), "esfinge-perfil-")), {
    channel: "chromium",
    headless: true,
    args: [`--disable-extensions-except=${EXTENSION}`, `--load-extension=${EXTENSION}`],
  });
  // Un formulario de entrar en cualquier sitio `.prueba`, servido por Playwright.
  await contexto.route("https://*.prueba/**", (ruta) =>
    ruta.fulfill({
      contentType: "text/html",
      body:
        '<!doctype html><title>Entrar</title><form action="/hecho" method="post">' +
        '<input name="usuario" autocomplete="username">' +
        '<input name="clave" type="password" autocomplete="current-password">' +
        "<button>Entrar</button></form>",
    }),
  );
  const trabajador = contexto.serviceWorkers()[0] ?? (await contexto.waitForEvent("serviceworker"));
  id = new URL(trabajador.url()).host;
});

test.afterAll(async () => {
  await contexto?.close();
});

async function panel(): Promise<Page> {
  const p = await contexto.newPage();
  await p.setViewportSize({ width: 360, height: 560 });
  await p.goto(`chrome-extension://${id}/panel.html`);
  return p;
}

/** Captura del panel en los dos temas, solo con CAPTURAS=carpeta, para mirarla. */
async function retratar(p: Page, nombre: string) {
  const donde = process.env.CAPTURAS;
  if (!donde) return;
  for (const tema of ["light", "dark"] as const) {
    await p.emulateMedia({ colorScheme: tema });
    await p.waitForTimeout(250);
    await p.screenshot({ path: `${donde}/panel-${nombre}-${tema === "light" ? "claro" : "oscuro"}.png`, fullPage: true });
  }
  await p.emulateMedia({ colorScheme: "light" });
}

/** Lo que mandaría el panel por su puerto, con su respuesta. */
function alTrabajador(
  p: Page,
  mensaje: unknown,
): Promise<{
  ok: boolean;
  error?: string;
  motivo?: string;
  estado?: { abierta: boolean; sincro: { estado: string; mensaje?: string } };
  atestacion?: { idCredencial: string; datosDelCliente: string; objeto: string; datosDelAutenticador: string; publica: string };
  afirmacion?: { idCredencial: string; datosDelCliente: string; datosDelAutenticador: string; firma: string };
  llaves?: { id: string; nombre: string }[];
}> {
  return p.evaluate(
    (m) =>
      new Promise((resolver) => {
        const puerto = chrome.runtime.connect({ name: "panel" });
        puerto.onMessage.addListener((r) => {
          puerto.disconnect();
          resolver(r);
        });
        puerto.postMessage(m);
      }),
    mensaje,
  ) as never;
}

/**
 * Como `alTrabajador`, pero **esperando si el freno está lleno**.
 *
 * No es un apaño: es un hecho del diseño. El freno de sesenta preguntas por minuto
 * vive en el trabajador de fondo y su ventana es de un minuto, así que una tanda larga
 * lo agota y las últimas pruebas se lo encuentran cerrado. Reintentar es lo que haría
 * el panel, y esperar aquí es más honesto que mover la prueba a donde el freno esté
 * fresco: eso la haría depender del orden de las demás.
 */
async function conPaciencia(p: Page, mensaje: unknown): Promise<Awaited<ReturnType<typeof alTrabajador>>> {
  let ultima!: Awaited<ReturnType<typeof alTrabajador>>;
  await expect
    .poll(
      async () => {
        ultima = await alTrabajador(p, mensaje);
        return ultima.motivo === "demasiado" ? "esperando" : "listo";
      },
      { timeout: 75_000, intervals: [1000, 5000, 10_000] },
    )
    .toBe("listo");
  return ultima;
}

/** Abre un sitio `.prueba` y devuelve lo que la extensión haya escrito en la contraseña. */
async function contrasenaRellenada(sitio: string, esperar = true): Promise<string> {
  const p = await contexto.newPage();
  await p.goto(`https://${sitio}/entrar`);
  const campo = p.locator('input[type="password"]');
  if (esperar) await expect(campo).not.toHaveValue("", { timeout: 15_000 });
  else await p.waitForTimeout(2500);
  const valor = await campo.inputValue();
  await p.close();
  return valor;
}

/**
 * **Por qué no salió el banner**, para que un fallo aquí no sea «contó 0» y nada más.
 *
 * Tres tramos, y los tres se pueden ver desde fuera: si el `shim` está instalado —si no,
 * es el puente—, qué dominios hay apuntados, y **qué contestó el trabajador a cada
 * pregunta por las llaves**, que es lo que faltaba las dos veces que esto se cayó. Lo
 * último sale de la bitácora que la compilación de pruebas escribe en `storage.session`.
 */
async function porQueNoSalio(pagina: Page): Promise<string> {
  const nativa = await pagina
    .evaluate(() => /\[native code\]/.test(Function.prototype.toString.call(navigator.credentials.get)))
    .catch(() => null);
  const w = contexto.serviceWorkers()[0];
  const dominios = await w
    .evaluate(async () => (await chrome.storage.session.get("dominios-con-llave"))["dominios-con-llave"])
    .catch(() => "no se pudo leer");
  const bitacora = await w
    .evaluate(async () => (await chrome.storage.session.get("pruebas-llaves"))["pruebas-llaves"])
    .catch(() => "no se pudo leer");
  return (
    `el banner no salió · shim instalado: ${nativa === null ? "no se pudo mirar" : !nativa}` +
    ` · dominios apuntados: ${JSON.stringify(dominios)}` +
    `\n  lo que contestó el trabajador: ${JSON.stringify(bitacora, null, 1)}`
  );
}

/**
 * Pide una llave de acceso como lo haría el sitio. **Una llamada es una pregunta** al
 * trabajador cuando la bandera está apagada, así que se cuentan.
 */
async function pedirLlave(pagina: Page, rpId = "sitio.prueba") {
  await pagina.evaluate((quien) => {
    navigator.credentials
      .get({
        publicKey: {
          challenge: new Uint8Array(32).fill(7),
          rpId: quien,
          userVerification: "discouraged",
          allowCredentials: [],
        },
      })
      .catch(() => {});
  }, rpId);
}

/**
 * **Espera a que la bandera pueda estar puesta, en vez de llamar en bucle.**
 *
 * Y esto no es una comodidad: **llamar en bucle agotaba el freno de sesenta preguntas por
 * minuto y la prueba se envenenaba sola**. Cada `credentials.get` con la bandera apagada
 * es una pregunta al trabajador; veinte llamadas en treinta segundos, más el refresco del
 * icono y lo que pidan las pruebas vecinas del mismo minuto, y la respuesta pasaba a ser
 * `motivo: "demasiado"` — que es concluyente, así que la bandera se queda apagada y el
 * banner **no sale nunca**. Dos tandas completas se cayeron por esto y el síntoma, «el
 * banner no salió», no se parecía en nada a la causa.
 *
 * Lo que se espera es que el trabajador haya contestado algo que enciende la bandera para
 * ese origen: una llave usable o un `quizas` con la bóveda cerrada. Sale de la bitácora
 * que escribe la compilación de pruebas, y **no cuesta ninguna pregunta**.
 */
/**
 * **Y se limpia antes de cada escenario**, o la espera se cumple con una línea de la
 * prueba anterior: la bitácora guarda las últimas veinte de toda la tanda y el origen se
 * repite entre pruebas. Sin esto, `esperarLaBandera` pasaría sin que hubiera pasado nada.
 */
async function limpiarLaBitacora() {
  await contexto.serviceWorkers()[0].evaluate(() => chrome.storage.session.remove("pruebas-llaves"));
}

async function esperarLaBandera(origen: string) {
  try {
    await esperarLaBanderaSinMas(origen);
  } catch (e) {
    const bitacora = await contexto
      .serviceWorkers()[0]
      .evaluate(async () => (await chrome.storage.session.get("pruebas-llaves"))["pruebas-llaves"])
      .catch(() => "no se pudo leer");
    throw new Error(`la bandera no se encendió\n  bitácora: ${JSON.stringify(bitacora, null, 1)}\n${String(e)}`);
  }
}

async function esperarLaBanderaSinMas(origen: string) {
  await expect
    .poll(
      () =>
        contexto.serviceWorkers()[0].evaluate(async (donde) => {
          const b = ((await chrome.storage.session.get("pruebas-llaves"))["pruebas-llaves"] ?? []) as {
            origen: string;
            rpId: string | null;
            quizas: boolean | null;
            llaves: number | null;
          }[];
          return b.some((l) => l.origen === donde && l.rpId === null && (l.quizas === true || (l.llaves ?? 0) > 0));
        }, origen),
      { timeout: 30_000, intervals: [300, 500, 1000], message: "el trabajador no ha dicho que aquí haya llave" },
    )
    .toBe(true);
}

test.describe.serial("la extensión con cuenta, sin la aplicación", () => {
  test("entra con su código y rellena sola en un sitio guardado", async () => {
    const p = await panel();
    await p.click("#aceptar");
    // Sin la aplicación en esta máquina, el panel lo dice y ofrece la cuenta.
    await retratar(p, "sin-aplicacion");
    await p.click("#usar-cuenta");
    await p.fill("#cuenta-correo", CORREO);
    await p.fill("#cuenta-maestra", MAESTRA);
    await retratar(p, "entrar");
    await p.click("#cuenta-enviar");
    await expect(p.locator("#cuenta-codigo")).toBeVisible({ timeout: 30_000 });
    await retratar(p, "codigo");
    // **Como lo hace una persona**: el panel se cierra al ir al correo a por el
    // código. Al abrirlo otra vez tiene que seguir en el código, sin volver a
    // empezar ni mandar otro. Lo contó el cliente la primera vez que lo probó.
    await p.close();
    const p2 = await panel();
    await expect(p2.locator("#cuenta-codigo")).toBeVisible({ timeout: 15_000 });
    await p2.fill("#cuenta-codigo", await codigoDelBuzon(CORREO));
    await p2.click("#cuenta-enviar");
    await expect(p2.locator("#gestos-correo")).toHaveText(CORREO, { timeout: 30_000 });
    await expect(p2.locator("#bloquear")).toBeVisible();
    await retratar(p2, "abierta");
    await p2.close();

    expect(await contrasenaRellenada("sitio.prueba")).toBe("clave-del-sitio");
    // Y en un sitio sin nada guardado, no escribe nada.
    expect(await contrasenaRellenada("nada.prueba", false)).toBe("");
  });

  test("lo que se guarda en otro equipo llega al navegador", async () => {
    await desdeElOtroEquipo((b) => b.poner(credencial("Otro", "yo@otro.prueba", "clave-del-otro", "https://otro.prueba")).then(() => {}));
    // Con el botón del panel, sin esperar a la pasada de cada minuto (2.25.2).
    const p = await panel();
    await p.click("#sincronizar");
    // Gira mientras sincroniza, y para al acabar (2.25.2).
    await expect(p.locator("#sincronizar")).toHaveClass(/girando/);
    await expect(p.locator("#resultado")).toContainText("Sincronizada con tu cuenta.", { timeout: 20_000 });
    await expect(p.locator("#sincronizar")).not.toHaveClass(/girando/);
    await retratar(p, "sincronizada");
    await p.close();
    expect(await contrasenaRellenada("otro.prueba")).toBe("clave-del-otro");
  });

  test("lo que se guarda en el navegador sube a la cuenta", async () => {
    const p = await panel();
    const r = await alTrabajador(p, {
      version: 1,
      que: "guardar-cuenta",
      origen: "https://nuevo.prueba/entrar",
      usuario: "yo@nuevo.prueba",
      secreto: "clave-nueva",
      titulo: "Nuevo",
    });
    expect(r.ok, r.error).toBe(true);
    await expect.poll(titulosEnElServidor, { timeout: 20_000 }).toEqual(["Nuevo", "Otro", "Sitio"]);
    await p.close();
  });

  test("lo que te mandan espera en el buzón hasta que lo guardas", async () => {
    await manda.mandar(credencial("Regalo", "yo@regalo.prueba", "clave-regalada", "https://regalo.prueba"), CORREO);

    const p = await panel();
    await expect(p.locator("#buzon")).toBeVisible({ timeout: 20_000 });
    await expect(p.locator("#buzon-lista .titulo")).toHaveText("Regalo");
    // **La huella es la de quien manda**, no la de la cuenta: es lo único que se
    // puede comparar por teléfono con la otra persona.
    await expect(p.locator("#buzon-lista .huella")).toHaveText(`De ${manda.huella}`);
    await retratar(p, "buzon");
    // Y hasta que alguien pulsa «Guardar», la contraseña no entra en la bóveda:
    // en ese sitio todavía no se rellena nada.
    expect(await contrasenaRellenada("regalo.prueba", false)).toBe("");

    await p.click("#buzon-lista button.primario");
    await expect(p.locator("#resultado")).toHaveText("Copia guardada en tu bóveda.", { timeout: 20_000 });
    await expect(p.locator("#buzon")).toBeHidden();
    await p.close();

    expect(await contrasenaRellenada("regalo.prueba")).toBe("clave-regalada");
    // Y lo aceptado sube a la cuenta, como cualquier otro guardado del navegador.
    await expect.poll(titulosEnElServidor, { timeout: 20_000 }).toEqual(["Nuevo", "Otro", "Regalo", "Sitio"]);
  });

  test("y desde el panel se manda una copia, con la huella delante", async () => {
    // **El panel mira la pestaña activa**, y aquí es una pestaña más. Así que se
    // abre el sitio, se pone delante y se recarga el panel: entonces pregunta por
    // las cuentas de `sitio.prueba`, como el panel de verdad, que no es pestaña.
    const sitio = await contexto.newPage();
    await sitio.goto("https://sitio.prueba/entrar");
    const p = await panel();
    await sitio.bringToFront();
    await p.reload();
    await expect(p.locator("#lista li")).toHaveCount(1, { timeout: 20_000 });
    // La fila entera, que es donde el sobre tiene que caber sin apretar a los demás.
    await retratar(p, "lista");
    await p.click('#lista li button[aria-label="Mandar una copia"]');
    await expect(p.locator("#compartir")).toBeVisible();

    await p.fill("#compartir-correo", CORREO_QUE_MANDA);
    await p.click("#compartir-enviar");
    // **La huella antes que el envío**, y es la de quien la va a recibir.
    await expect(p.locator("#compartir-huella")).toHaveText(manda.huella, { timeout: 20_000 });
    await retratar(p, "compartir");
    await expect(p.locator("#compartir-enviar")).toHaveText("Mandar la copia");

    await p.click("#compartir-enviar");
    await expect(p.locator("#resultado")).toContainText(`mandada a ${CORREO_QUE_MANDA}`, { timeout: 20_000 });
    await expect(p.locator("#compartir")).toBeHidden();
    await p.close();
    await sitio.close();

    // Y al otro lado se abre con su identidad, con la contraseña dentro.
    const suyo = await manda.buzon();
    expect(suyo.map((e) => e.titulo)).toEqual(["Sitio"]);
    expect(suyo[0].secreto).toBe("clave-del-sitio");
  });

  test("a quien no tiene cuenta, la copia le espera y sale cuando la crea", async () => {
    const correoInvitado = `invitado-${Date.now()}@ejemplo.com`;

    // Se manda desde el panel, igual que antes, pero a una dirección detrás de la
    // cual todavía no hay nadie.
    const sitio = await contexto.newPage();
    await sitio.goto("https://sitio.prueba/entrar");
    const p = await panel();
    await sitio.bringToFront();
    await p.reload();
    await expect(p.locator("#lista li")).toHaveCount(1, { timeout: 20_000 });
    await p.click('#lista li button[aria-label="Mandar una copia"]');
    await p.fill("#compartir-correo", correoInvitado);
    await p.click("#compartir-enviar");
    await expect(p.locator("#compartir-huella")).not.toBeEmpty({ timeout: 20_000 });
    await p.click("#compartir-enviar");
    // **Lo que se dice vale tenga cuenta o no**: desde aquí no se puede saber cuál
    // de las dos cosas ha pasado, y por eso la frase habla de la invitación.
    await expect(p.locator("#resultado")).toContainText("invitación", { timeout: 20_000 });
    await p.close();
    await sitio.close();

    // Y al buzón de esa dirección le ha llegado la invitación, con quién invita.
    const invitacion = await fetch(`${SERVIDOR}/_pruebas/buzon?correo=${encodeURIComponent(correoInvitado)}`);
    const { mensajes } = (await invitacion.json()) as { mensajes: { asunto: string; cuerpo: string }[] };
    expect(mensajes[0]?.asunto).toContain(CORREO);
    expect(mensajes[0]?.cuerpo).toContain("webcafeina.github.io/esfinge");

    // Ahora esa persona crea su cuenta y publica sus llaves.
    const invitado = await quienManda(correoInvitado, "la maestra larga de quien recibe la copia");
    expect(await invitado.buzon()).toHaveLength(0);

    // Y en la siguiente sincronización de la extensión, la copia sale sola.
    const p2 = await panel();
    await p2.click("#sincronizar");
    await expect(p2.locator("#resultado")).toContainText("Sincronizada con tu cuenta.", { timeout: 20_000 });
    await p2.close();

    await expect.poll(async () => (await invitado.buzon()).map((e) => e.titulo), { timeout: 20_000 }).toEqual(["Sitio"]);
    const [recibida] = await invitado.buzon();
    expect(recibida.secreto).toBe("clave-del-sitio");
  });

  test("bloqueada no rellena, y se desbloquea con la maestra", async () => {
    const p = await panel();
    await p.click("#bloquear");
    await expect(p.locator("#cuenta-titulo")).toHaveText("Tu bóveda está cerrada");
    expect(await contrasenaRellenada("sitio.prueba", false)).toBe("");

    await p.fill("#cuenta-maestra", "otra que no es");
    await p.click("#cuenta-enviar");
    await expect(p.locator("#cuenta-error")).toBeVisible({ timeout: 20_000 });
    await retratar(p, "cerrada-con-error");
    await p.fill("#cuenta-maestra", MAESTRA);
    await p.click("#cuenta-enviar");
    await expect(p.locator("#bloquear")).toBeVisible({ timeout: 20_000 });
    await p.close();
    expect(await contrasenaRellenada("sitio.prueba")).toBe("clave-del-sitio");
  });

  /**
   * **El banner de la llave de acceso sale con la bóveda cerrada** (ADR 0048), que es
   * lo que decidió el cliente y lo que no se puede probar en ninguna otra parte.
   *
   * Hacen falta las tres piezas juntas y de verdad: la llave dentro de la bóveda, la
   * lista de dominios apuntada en `storage.session` mientras estaba abierta, y el
   * shim del mundo principal preguntando en vez de ceder. Con la bóveda cerrada no hay
   * a quién preguntar, así que **sin la lista el shim cedería** y saldría el diálogo
   * del navegador: la decisión —«sale y ofrece abrirla»— quedaría escrita y no hecha.
   *
   * Lo que se mira es que aparezca el anfitrión del banner. **No se pulsa**: va en una
   * sombra cerrada, y desde aquí no se llega a sus botones. Lo que se decide pulsando
   * lo prueban `banner.spec.ts` y `mundo.spec.ts`; lo que esta prueba dice, y solo
   * ella, es que las tres piezas están conectadas.
   *
   * Y deja la bóveda **abierta**, como se la encontró: esto es una serie.
   */
  test("con la bóveda cerrada, el banner de la llave sale igual", async () => {
    const { crearLlave } = await import("../src/nucleo/llaves");
    const { aBase64Url } = await import("../src/nucleo/afirmacion");
    const par = await crearLlave();
    await desdeElOtroEquipo(async (b) => {
      await b.poner({
        id: "",
        tipo: "llave",
        titulo: "Sitio de prueba",
        rpId: "sitio.prueba",
        idCredencial: "Y3JlZC1kZWwtc2l0aW8",
        idUsuario: "dXN1YXJpbw",
        nombreVisible: "yo@sitio.prueba",
        algoritmo: -7,
        clavePrivada: aBase64Url(par.privada),
        creada: "",
        cambiada: "",
      } as Entrada);
    });
    const p = await panel();
    await p.click("#sincronizar");
    await expect(p.locator("#resultado")).toContainText("Sincronizada con tu cuenta.", { timeout: 20_000 });
    await p.close();

    // **Una visita con la bóveda abierta**, que es lo que apunta la lista. Sin esta
    // vuelta no hay nada apuntado, y eso es justo lo que la ADR dice en voz alta:
    // recién abierto el navegador, y hasta abrir la bóveda una vez, el banner no sale.
    const abierta = await contexto.newPage();
    await abierta.goto("https://sitio.prueba/entrar");
    await expect
      .poll(
        () =>
          contexto.serviceWorkers()[0].evaluate(
            async () => ((await chrome.storage.session.get("dominios-con-llave"))["dominios-con-llave"] as string[]) ?? [],
          ),
        { timeout: 15_000 },
      )
      .toContain("sitio.prueba");
    await abierta.close();

    const cerrar = await panel();
    await cerrar.click("#bloquear");
    await expect(cerrar.locator("#cuenta-titulo")).toHaveText("Tu bóveda está cerrada");
    await cerrar.close();

    await limpiarLaBitacora();
    const sitio = await contexto.newPage();
    await sitio.goto("https://sitio.prueba/entrar");
    // **Primero se espera a que la bandera esté, y después se llama una vez.** La
    // bandera «aquí hay algo» la **empuja** el mundo aislado después de cargar la página
    // —tiene que preguntárselo al trabajador, que puede estar dormido—, y mientras no
    // llega vale `false` y el shim cede sin esperar a nadie. Es a propósito: esperar
    // dentro de `get` es lo que agota la activación de usuario y rompe el inicio de
    // sesión de quien no usa Esfinge. Una persona tarda segundos en pulsar «Entrar»;
    // una prueba, cero.
    //
    // Antes esto se resolvía **llamando en bucle hasta que saliera**, y eso agotaba el
    // freno de sesenta preguntas por minuto: la respuesta pasaba a «demasiado» y el
    // banner no salía nunca. Ver `esperarLaBandera`.
    await esperarLaBandera("https://sitio.prueba/entrar");
    try {
      await pedirLlave(sitio);
      await expect(sitio.locator("esfinge-llave")).toHaveCount(1, { timeout: 10_000 });
    } catch (e) {
      throw new Error(`${await porQueNoSalio(sitio)}\n${String(e)}`);
    }
    await sitio.close();

    const abrir = await panel();
    await abrir.fill("#cuenta-maestra", MAESTRA);
    await abrir.click("#cuenta-enviar");
    await expect(abrir.locator("#bloquear")).toBeVisible({ timeout: 20_000 });
    await abrir.close();
  });

  /**
   * **Y abriendo la bóveda con la pestaña ya abierta, sin recargar** (ADR 0048).
   *
   * El caso que faltaba: **navegador recién abierto, bóveda sin abrir ni una vez y vas
   * derecho a entrar**. Ahí la lista de dominios está vacía, así que el `shim` cede y el
   * banner no sale; y como la bandera se calcula al cargar la página, abrir la bóveda
   * después no cambiaba nada — había que recargar, y eso nadie lo adivina.
   *
   * La prueba empieza **comprobando el caso malo**, que es lo que la hace valer: si el
   * banner saliera ya ahí, lo de después no estaría comprobando nada.
   */
  test("abriendo la bóveda después, el banner sale sin recargar la página", async () => {
    const trabajador = contexto.serviceWorkers()[0];
    // La bóveda cerrada y la lista vacía: el navegador recién abierto, sin haber pasado
    // por ninguna página con la bóveda abierta.
    const cerrar = await panel();
    await cerrar.click("#bloquear");
    await expect(cerrar.locator("#cuenta-titulo")).toHaveText("Tu bóveda está cerrada");
    await cerrar.close();
    await trabajador.evaluate(() => chrome.storage.session.remove("dominios-con-llave"));
    await limpiarLaBitacora();

    const sitio = await contexto.newPage();
    await sitio.goto("https://sitio.prueba/entrar");
    // **El caso malo, y se espera a que el shim esté instalado antes de darlo por bueno**:
    // sin esa espera, «no sale el banner» podría ser solo que el puente no ha llegado.
    await expect
      .poll(
        () =>
          sitio.evaluate(
            () => !/\[native code\]/.test(Function.prototype.toString.call(CredentialsContainer.prototype.get)),
          ),
        { timeout: 20_000, message: "el shim no se ha instalado" },
      )
      .toBe(true);
    await pedirLlave(sitio);
    await expect(sitio.locator("esfinge-llave")).toHaveCount(0);

    // Se abre la bóveda, **sin tocar la pestaña**.
    const abrir = await panel();
    await abrir.fill("#cuenta-maestra", MAESTRA);
    await abrir.click("#cuenta-enviar");
    await expect(abrir.locator("#bloquear")).toBeVisible({ timeout: 20_000 });
    await abrir.close();

    // Y se vuelve a la pestaña, que es lo que hace cualquiera después de abrir la bóveda
    // en el panel — y lo que el refresco del icono necesita, porque trabaja sobre **la
    // pestaña activa**. Con el panel recién cerrado, la activa no era ésta.
    await sitio.bringToFront();
    // Y suena el reloj del icono, que es quien mantiene la lista y avisa a la pestaña. En
    // uso normal llega solo, dentro del minuto; aquí se hace sonar para no esperarlo.
    await trabajador.evaluate(() =>
      chrome.alarms.create("refrescar-el-icono", { when: Date.now() + 50, periodInMinutes: 1 }),
    );

    // Y se espera a que el trabajador lo diga —sin gastar preguntas— antes de volver a
    // pedir la llave: lo que se comprueba es que **la misma pestaña** acaba ofreciéndola.
    await esperarLaBandera("https://sitio.prueba/entrar");
    try {
      await pedirLlave(sitio);
      await expect(sitio.locator("esfinge-llave")).toHaveCount(1, { timeout: 10_000 });
    } catch (e) {
      throw new Error(`${await porQueNoSalio(sitio)}\n${String(e)}`);
    }
    await sitio.close();
  });

  /**
   * **Crear una llave de acceso con la extensión de verdad, y que sirva** (ADR 0048, P3).
   *
   * Es la prueba que el plan pedía antes de publicar la P3, y lo que comprueba es el
   * ciclo entero con las piezas de verdad: el trabajador de fondo, la bóveda del
   * navegador y el servidor de cuentas. No hay dobles.
   *
   * Se pide por el puerto del panel y no pulsando el banner, por la razón de siempre:
   * **el banner va en una sombra cerrada** y desde aquí no se llega a sus botones. Que
   * el banner sale lo comprueba la segunda mitad; que lo que hay detrás funciona, la
   * primera.
   *
   * Y lo último es lo que de verdad importa: **se firma con la llave recién creada**. Si
   * al crear se guardara algo incompleto, esto sería una cuenta con una credencial
   * registrada en el sitio y sin forma de entrar — el peor fallo posible de esta clase.
   */
  test("crea una llave de acceso, la guarda y luego firma con ella", async () => {
    const p = await panel();
    const reto = "MDEyMzQ1Njc4OWFiY2RlZg";
    const creada = await conPaciencia(p, {
      version: 1,
      que: "crear-llave",
      origen: "https://llaves.prueba/registro",
      rpId: "llaves.prueba",
      reto,
      usuario: "yo@llaves.prueba",
      idUsuario: "dXN1YXJpby1kZWwtc2l0aW8",
      titulo: "Llaves de prueba",
      algoritmos: [-7],
    });
    expect(creada.ok, creada.error).toBe(true);
    expect(creada.atestacion?.idCredencial).toBeTruthy();
    expect(creada.atestacion?.publica).toBeTruthy();

    // **El clientDataJSON dice que se está creando y con el origen que puso el
    // trabajador**, no el que mandó nadie.
    const cliente = Buffer.from(creada.atestacion!.datosDelCliente, "base64url").toString();
    expect(cliente).toContain('"type":"webauthn.create"');
    expect(cliente).toContain('"origin":"https://llaves.prueba"');

    // Está en la bóveda y el sitio la ve como suya.
    const suyas = await conPaciencia(p, {
      version: 1,
      que: "llaves",
      origen: "https://llaves.prueba/entrar",
      rpId: "llaves.prueba",
    });
    expect(suyas.ok, suyas.error).toBe(true);
    expect(suyas.llaves).toHaveLength(1);
    expect(suyas.llaves?.[0].nombre).toBe("yo@llaves.prueba");

    // **Y se firma con ella.** Es la mitad que dice que lo guardado sirve.
    const firmada = await conPaciencia(p, {
      version: 1,
      que: "firmar-llave",
      origen: "https://llaves.prueba/entrar",
      rpId: "llaves.prueba",
      id: suyas.llaves![0].id,
      reto: "b3Ryby1yZXRvLWN1YWxxdWllcmE",
    });
    expect(firmada.ok, firmada.error).toBe(true);
    expect(firmada.afirmacion?.firma).toBeTruthy();
    expect(firmada.afirmacion?.idCredencial).toBe(creada.atestacion?.idCredencial);

    // Y sube a la cuenta como cualquier otra entrada.
    await p.click("#sincronizar");
    await expect(p.locator("#resultado")).toContainText("Sincronizada con tu cuenta.", { timeout: 20_000 });
    await p.close();
    await expect.poll(titulosEnElServidor, { timeout: 20_000 }).toContain("Llaves de prueba");

    // **La segunda mitad: que en una página de verdad salga el banner de crear.**
    // Sin esperar la promesa, que con el banner delante no se resuelve hasta que
    // alguien decide. Se insiste porque la bandera llega después de cargar la página.
    const sitio = await contexto.newPage();
    await sitio.goto("https://otro.llaves.prueba/registro");
    await expect
      .poll(
        async () => {
          await sitio.evaluate(() => {
            navigator.credentials
              .create({
                publicKey: {
                  challenge: new Uint8Array(32).fill(5),
                  rp: { id: "otro.llaves.prueba", name: "Otro" },
                  user: { id: new Uint8Array(16).fill(6), name: "otro@llaves.prueba", displayName: "Otro" },
                  pubKeyCredParams: [{ type: "public-key", alg: -7 }],
                },
              })
              .catch(() => {});
          });
          return sitio.locator("esfinge-llave").count();
        },
        { timeout: 30_000, intervals: [500, 1000, 2000] },
      )
      .toBe(1);
    await sitio.close();
  });

  test("sin tocarla quince minutos, se cierra sola", async () => {
    // El reloj de la bóveda es la alarma de cada minuto. Se hace sonar ya, con la
    // última actividad de hace dieciséis minutos, desde el propio trabajador.
    const trabajador = contexto.serviceWorkers()[0];
    await trabajador.evaluate(async () => {
      await chrome.storage.session.set({ "cuenta-actividad": Date.now() - 16 * 60_000 });
      await chrome.alarms.create("refrescar-el-icono", { when: Date.now() + 50, periodInMinutes: 1 });
    });
    // Antes de abrir el panel, que se haya cerrado: abrirlo **es** actividad, y
    // abierto antes de que suene la alarma la bóveda ya no se cerraría.
    await expect
      .poll(() => trabajador.evaluate(async () => (await chrome.storage.session.get("cuenta-llave"))["cuenta-llave"] ?? null), {
        timeout: 15_000,
      })
      .toBeNull();
    const p = await panel();
    await expect(p.locator("#cuenta-titulo")).toHaveText("Tu bóveda está cerrada", { timeout: 15_000 });
    await p.fill("#cuenta-maestra", MAESTRA);
    await p.click("#cuenta-enviar");
    await expect(p.locator("#bloquear")).toBeVisible({ timeout: 20_000 });
    await p.close();
  });

  test("olvidado desde otro equipo, se cierra y lo dice", async () => {
    const c = await fetch(`${SERVIDOR}/v1/dispositivos`, { headers: { Authorization: `Bearer ${otro.token}` } });
    const equipos = (await c.json()) as { id: string; actual: boolean }[];
    const delNavegador = equipos.find((e) => !e.actual)!;
    await fetch(`${SERVIDOR}/v1/dispositivos/${delNavegador.id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${otro.token}` },
    });

    const p = await panel();
    const r = await alTrabajador(p, { cuenta: "sincronizar" });
    expect(r.estado?.abierta).toBe(false);
    expect(r.estado?.sincro.mensaje).toContain("ya no reconoce este navegador");
    await p.reload();
    await expect(p.locator("#cuenta-texto")).toContainText("ya no reconoce este navegador");
    await p.fill("#cuenta-maestra", MAESTRA);
    await p.click("#cuenta-enviar");
    // Se abre para trabajar aquí, y ofrece volver a entrar para sincronizar.
    await expect(p.locator("#volver-a-entrar")).toBeVisible({ timeout: 20_000 });
    await retratar(p, "sin-sesion");
    await p.close();
  });
});
