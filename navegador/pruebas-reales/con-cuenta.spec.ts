import { chromium, expect, test, type BrowserContext, type Page } from "@playwright/test";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { Boveda } from "../src/nucleo/boveda";
import { Cliente } from "../src/nucleo/cliente";
import { derivarAcceso } from "../src/nucleo/cuenta";
import { azarDe, base64url, PERFIL_INTERACTIVO } from "../src/nucleo/esf1";
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

let contexto: BrowserContext;
let id: string;
/** La sesión del «otro equipo», que es quien crea la cuenta. */
let otro: { token: string };

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

async function titulosEnElServidor(): Promise<string[]> {
  const bajada = (await new Cliente(SERVIDOR).bajar(otro.token, 0))!;
  return (await Boveda.abrir(bajada.datos, MAESTRA)).buscar("").map((e) => e.titulo).sort();
}

// ------------------------------------------------------------------ el navegador

test.beforeAll(async () => {
  otro = await crearCuenta();
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
function alTrabajador(p: Page, mensaje: unknown): Promise<{ ok: boolean; error?: string; estado?: { abierta: boolean; sincro: { estado: string; mensaje?: string } } }> {
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
