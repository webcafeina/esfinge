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
import { build } from "vite";
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
// Y el aviso de datos de la primera vez (ADR 0033), que tapa todo lo demás.
estados.aviso = estados.una;

/**
 * La `chrome` de mentira. Se inyecta antes de que cargue el panel, y contesta por
 * puerto igual que el trabajador de verdad.
 */
function falsa([respuesta, aceptado]) {
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
      query: async () => [{ id: 7, url: "https://login.brevo.com/entrar?x=1", favIconUrl: "data:image/svg+xml;utf8,<svg xmlns=%27http://www.w3.org/2000/svg%27 viewBox=%270 0 16 16%27><rect width=%2716%27 height=%2716%27 rx=%274%27 fill=%27%230b996e%27/></svg>" }],
      connect: () => puerto(() => ({ ok: true })),
    },
    storage: {
      local: {
        get: async () => (aceptado ? { consentimiento: { version: 1, cuando: "2026-09-14" } } : {}),
        set: async () => {},
      },
      onChanged: { addListener() {}, removeListener() {} },
    },
  };
}

const navegador = await chromium.launch();
for (const tema of ["light", "dark"]) {
  const contexto = await navegador.newContext({
    colorScheme: tema,
    deviceScaleFactor: 2,
    // Más ancho que el panel: el cuerpo mide 360 más su relleno, y con una ventana
    // de 360 la captura cortaba el borde derecho —«Abiert…»— y parecía un fallo.
    viewport: { width: 420, height: 460 },
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
    await pagina.addInitScript(falsa, [respuesta, nombre !== "aviso"]);
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
/* ----------------------------------------- el icono de la barra, compuesto */

/**
 * **La barra de verdad no se puede capturar aquí**, así que esto es una
 * composición: el icono tal cual, a 16 px con pantalla Retina, sobre el color de la
 * barra de cada navegador, con una insignia dibujada a imitación de la de Chrome.
 * Sirve para ver si la silueta se lee y si la insignia tapa algo; **no** para dar
 * por buena la de un navegador concreto.
 */
const iconoEnDatos = (v) =>
  `data:image/png;base64,${readFileSync(join(raiz, "iconos", `barra-${v}-32.png`)).toString("base64")}`;
const muestras = [
  ["activo", "", ""],
  ["activo", "2", "#2b2b31"],
  ["activo", "✓", "#1d6f31"],
  ["cerrado", "", ""],
  ["apagado", "!", "#8f5300"],
];
for (const [nombreBarra, fondoBarra] of [
  ["chrome-claro", "#ffffff"],
  ["chrome-oscuro", "#3c3c3c"],
  ["firefox-claro", "#f9f9fb"],
  ["firefox-oscuro", "#2b2a33"],
]) {
  const pagina = await navegador.newPage({ viewport: { width: 330, height: 40 }, deviceScaleFactor: 2 });
  await pagina.setContent(`<!doctype html><style>
    body{margin:0;background:${fondoBarra};display:flex;gap:30px;padding:10px 18px;font:700 9px system-ui}
    .i{position:relative;width:16px;height:16px}
    .i img{width:16px;height:16px;display:block}
    .b{position:absolute;right:-7px;bottom:-5px;min-width:9px;height:11px;padding:0 2px;border-radius:3px;
       color:#fff;display:grid;place-items:center;line-height:1}
  </style>${muestras
    .map(
      ([v, t, c]) =>
        `<div class="i"><img src="${iconoEnDatos(v)}">${t ? `<span class="b" style="background:${c}">${t}</span>` : ""}</div>`,
    )
    .join("")}`);
  await pagina.screenshot({ path: join(salida, `barra-${nombreBarra}.png`) });
  await pagina.close();
}

/* ----------------------------------------- el campo rellenado, con su marca */

const marcas = (
  await build({
    configFile: false,
    logLevel: "silent",
    build: {
      write: false,
      lib: {
        entry: join(raiz, "src", "marcas.ts"),
        formats: ["iife"],
        name: "Marcas",
        fileName: () => "marcas.js",
      },
    },
  })
)[0].output[0].code;

for (const [nombreWeb, fondo, tinta, campo] of [
  ["web-clara", "#ffffff", "#1c1c1e", "#ffffff"],
  ["web-oscura", "#1b1b1f", "#f2f2f5", "#26262b"],
]) {
  const pagina = await navegador.newPage({ viewport: { width: 380, height: 250 }, deviceScaleFactor: 2 });
  await pagina.setContent(`<!doctype html><meta charset="utf-8"><style>
    body{margin:0;padding:24px 28px;background:${fondo};color:${tinta};font:14px system-ui}
    label{display:block;margin:10px 0 4px;font-size:12px;opacity:.75}
    input{width:300px;box-sizing:border-box;padding:9px 11px;border:1px solid #8888;border-radius:6px;
          background:${campo};color:${tinta};font:inherit}
  </style><form><label>Correo</label><input id="u" value="info@webcafeina.com">
  <label>Contraseña</label><input id="p" type="password" value="secretisimo"></form>`);
  await pagina.addScriptTag({ content: marcas });
  await pagina.evaluate(() => {
    const u = document.getElementById("u");
    const p = document.getElementById("p");
    Marcas.ponerFilete(u);
    Marcas.ponerFilete(p);
    Marcas.avisar(p, "Rellenado por Esfinge");
  });
  await pagina.waitForTimeout(250);
  await pagina.screenshot({ path: join(salida, `campo-${nombreWeb}.png`) });
  await pagina.close();
}

/* ------------------------------------------ la tarjeta de guardar */

const tarjeta = (
  await build({
    configFile: false,
    logLevel: "silent",
    build: {
      write: false,
      lib: {
        entry: join(raiz, "src", "tarjeta.ts"),
        formats: ["iife"],
        name: "Tarjeta",
        fileName: () => "tarjeta.js",
      },
    },
  })
)[0].output[0].code;

const estadosDeTarjeta = {
  guardar: {
    tipo: "oferta",
    oferta: { accion: "guardar", sitio: "login.brevo.com", titulo: "Brevo" },
    usuario: "info@webcafeina.com",
  },
  actualizar: {
    tipo: "oferta",
    oferta: {
      accion: "actualizar",
      sitio: "dash.cloudflare.com",
      cuentas: [
        { id: "1", titulo: "Cloudflare", usuario: "info@webcafeina.com" },
        { id: "2", titulo: "Cloudflare clientes", usuario: "clientes@webcafeina.com" },
      ],
    },
    usuario: "",
  },
  cerrada: { tipo: "cerrada", sitio: "login.brevo.com", usuario: "info@webcafeina.com" },
};
for (const [nombreWeb, fondo, tinta] of [
  ["web-clara", "#ffffff", "#1c1c1e"],
  ["web-oscura", "#1b1b1f", "#f2f2f5"],
]) {
  for (const [nombreEstado, estado] of Object.entries(estadosDeTarjeta)) {
    const pagina = await navegador.newPage({ viewport: { width: 520, height: 300 }, deviceScaleFactor: 2 });
    await pagina.setContent(`<!doctype html><meta charset="utf-8"><style>
      body{margin:0;padding:24px;background:${fondo};color:${tinta};font:14px system-ui}
    </style><h2>Panel de la web</h2><p>Contenido de la página que queda debajo.</p>`);
    await pagina.addScriptTag({ content: tarjeta });
    await pagina.evaluate((e) => Tarjeta.mostrarTarjeta(e, async () => ({ ok: true }), () => {}), estado);
    await pagina.waitForTimeout(300);
    await pagina.screenshot({ path: join(salida, `tarjeta-${nombreEstado}-${nombreWeb}.png`) });
    await pagina.close();
  }

  // Y cómo queda tras pulsar: con un clic de ratón de verdad, que es lo único a lo
  // que hace caso. Una respuesta buena y una con error.
  for (const [nombreFinal, respuesta] of [
    ["hecho", { ok: true }],
    ["error", { ok: false, error: "Demasiados cambios seguidos en la bóveda. Espera un momento." }],
  ]) {
    const pagina = await navegador.newPage({ viewport: { width: 520, height: 360 }, deviceScaleFactor: 2 });
    await pagina.setContent(`<!doctype html><meta charset="utf-8"><style>
      body{margin:0;padding:24px;background:${fondo};color:${tinta};font:14px system-ui}
    </style><h2>Panel de la web</h2><p>Contenido de la página que queda debajo.</p>`);
    await pagina.addScriptTag({ content: tarjeta });
    const donde = await pagina.evaluate(
      ([e, r]) => {
        const t = Tarjeta.mostrarTarjeta(e, async () => r, () => {});
        const b = t.raiz.querySelector("button.principal").getBoundingClientRect();
        return { x: b.x + b.width / 2, y: b.y + b.height / 2 };
      },
      [estadosDeTarjeta.guardar, respuesta],
    );
    await pagina.mouse.click(donde.x, donde.y);
    await pagina.waitForTimeout(300);
    await pagina.screenshot({ path: join(salida, `tarjeta-${nombreFinal}-${nombreWeb}.png`) });
    await pagina.close();
  }
}

await navegador.close();
console.log(`Capturas en ${salida}`);
