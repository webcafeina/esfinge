/**
 * Dibuja el panel de la extensión en cada uno de sus estados, en claro y en
 * oscuro, y deja las capturas en `capturas/`.
 *
 * # Por qué existe
 *
 * El panel salió en la 2.17.0 y **nadie lo había mirado nunca en una captura**.
 * En este proyecto, la mitad de los fallos de interfaz se han encontrado así
 * —«Hacienda» con una «A», los avisos partidos en columnas, el selector de orden
 * cortado— y ninguno lo encontró una prueba en verde.
 *
 * Aquí no hay navegador con la extensión cargada, así que se abre el `panel.html`
 * **compilado** —el mismo que se publica— con una `chrome` de mentira que contesta
 * lo que cada estado necesita. No se prueba el canal: se mira el dibujo.
 *
 * Uso: `pnpm run build && node herramientas/capturas.mjs`
 */
import { chromium } from "@playwright/test";
import { mkdirSync, readFileSync } from "node:fs";
import { extname, join } from "node:path";

const raiz = new URL("..", import.meta.url).pathname;
const compilado = join(raiz, "dist", "chrome");
const salida = join(raiz, "capturas");

/**
 * **Por un origen de mentira y no con `file://`**, porque el panel compilado pide
 * `/panel.css` y `/panel.js` con la ruta desde la raíz. Dentro de una extensión
 * esa raíz es la de la extensión y funciona; desde `file://` es la raíz del disco
 * y el navegador lo bloquea. Así el panel se carga exactamente como se publica.
 */
const ORIGEN = "http://panel.esfinge.test";
const panel = `${ORIGEN}/panel.html`;
const TIPOS = { ".html": "text/html", ".css": "text/css", ".js": "text/javascript" };
mkdirSync(salida, { recursive: true });

/** Los estados que hay que ver, con lo que contesta el trabajador en cada uno. */
const estados = {
  una: { ok: true, cuentas: [{ id: "1", titulo: "Brevo", usuario: "info@webcafeina.com", tieneCodigo: false }] },
  varias: {
    ok: true,
    cuentas: [
      { id: "1", titulo: "Cloudflare", usuario: "info@webcafeina.com", tieneCodigo: true },
      { id: "2", titulo: "Cloudflare · clientes de la agencia", usuario: "administracion.clientes.largos@webcafeina.com", tieneCodigo: false },
      { id: "3", titulo: "", usuario: "", tieneCodigo: true },
    ],
  },
  ninguna: { ok: true, cuentas: [] },
  cerrada: { ok: false, motivo: "cerrada", error: "La bóveda está cerrada" },
  "sin-emparejar": { ok: false, motivo: "sin-emparejar", error: "Permite este navegador" },
  "sin-esfinge": {
    ok: false,
    motivo: "sin-esfinge",
    error:
      "No se puede hablar con Esfinge. Comprueba que está instalada y que el canal con el " +
      "navegador está encendido en sus Ajustes. El navegador dice: Specified native messaging host not found.",
  },
  origen: { ok: false, motivo: "origen", error: "Esfinge solo rellena en https, y eso es «http»" },
  // Nunca contesta: es lo que se ve mientras se pregunta.
  preguntando: null,
};
// Y el resultado de un gesto: la lista de varias, con «Rellenar» pulsado.
estados.rellenado = estados.varias;

/**
 * La `chrome` de mentira. Se inyecta antes de que cargue el panel, y contesta por
 * puerto igual que el trabajador de verdad.
 */
function falsa(respuesta) {
  const puerto = (alMandar) => {
    const oyentes = [];
    const alIrse = [];
    return {
      name: "panel",
      onMessage: { addListener: (f) => oyentes.push(f) },
      onDisconnect: { addListener: (f) => alIrse.push(f) },
      disconnect() {},
      postMessage(m) {
        const r = alMandar(m);
        if (r !== null) setTimeout(() => oyentes.forEach((f) => f(r)), 30);
      },
    };
  };
  globalThis.chrome = {
    runtime: {
      getManifest: () => ({ version: "2.19.0" }),
      connect: () => puerto((m) => (m.que === "cuentas" ? respuesta : { ok: true, copiado: { portapapeles: 30 } })),
    },
    tabs: {
      query: async () => [{ id: 7, url: "https://login.brevo.com/entrar?x=1" }],
      connect: () => puerto(() => ({ ok: true })),
    },
    storage: { local: { get: async () => ({}), set: async () => {} } },
  };
}

const navegador = await chromium.launch();
for (const tema of ["light", "dark"]) {
  const contexto = await navegador.newContext({
    colorScheme: tema,
    deviceScaleFactor: 2,
    viewport: { width: 360, height: 420 },
  });
  await contexto.route(`${ORIGEN}/**`, (ruta) => {
    const fichero = new URL(ruta.request().url()).pathname;
    try {
      ruta.fulfill({
        body: readFileSync(join(compilado, fichero)),
        contentType: TIPOS[extname(fichero)] ?? "application/octet-stream",
      });
    } catch {
      ruta.fulfill({ status: 404 });
    }
  });
  for (const [nombre, respuesta] of Object.entries(estados)) {
    const pagina = await contexto.newPage();
    const errores = [];
    pagina.on("pageerror", (e) => errores.push(String(e)));
    pagina.on("console", (m) => m.type() === "error" && errores.push(m.text()));
    await pagina.addInitScript(falsa, respuesta);
    await pagina.goto(panel);
    await pagina.waitForTimeout(250);
    if (nombre === "rellenado") {
      await pagina.locator(".rellenar").first().click();
      await pagina.waitForTimeout(200);
    }
    // El panel de una extensión mide lo que mide su contenido: se recorta al
    // cuerpo, que es lo que el navegador enseña.
    const cuerpo = await pagina.locator("body").boundingBox();
    await pagina.screenshot({
      path: join(salida, `${nombre}-${tema === "light" ? "claro" : "oscuro"}.png`),
      clip: { x: 0, y: 0, width: cuerpo.width, height: Math.ceil(cuerpo.height) },
    });
    if (errores.length) console.error(`${nombre} (${tema}):`, errores.join(" | "));
    await pagina.close();
  }
  await contexto.close();
}
await navegador.close();
console.log(`Capturas en ${salida}`);
