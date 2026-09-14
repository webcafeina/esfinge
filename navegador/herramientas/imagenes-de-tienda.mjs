/**
 * Las imágenes de las fichas de la extensión en las tiendas (ADR 0033): cinco capturas de
 * 1280×800 y el mosaico de 440×280, en `docs/tiendas/imagenes/`.
 *
 * **Se componen con las capturas de verdad** que deja `pnpm run capturas` —el panel
 * compilado, la marca en el campo, la tarjeta— sobre la piedra de la marca. Nada de
 * maquetas dibujadas a mano: lo que se enseña en la tienda es lo que se instala.
 *
 * Los colores del texto son **blanco y `#d8d8de` sobre la piedra `#2b2b31`**, las dos
 * parejas que ya mide `internal/tema/extension_test.go` para la tarjeta de guardar.
 *
 * Uso: `pnpm run capturas && node herramientas/imagenes-de-tienda.mjs`
 */
import { chromium } from "@playwright/test";
import { existsSync, mkdirSync, readFileSync } from "node:fs";
import { join } from "node:path";

const raiz = new URL("..", import.meta.url).pathname;
const capturas = join(raiz, "capturas");
const salida = join(raiz, "..", "docs", "tiendas", "imagenes");
mkdirSync(salida, { recursive: true });

const PIEDRA = "#2b2b31";
const BLANCO = "#ffffff";
const SUAVE = "#d8d8de";

const silueta = readFileSync(join(raiz, "..", "build", "icono-barra.svg"), "utf8");

function captura(nombre) {
  const fichero = join(capturas, nombre);
  if (!existsSync(fichero)) throw new Error(`Falta ${nombre}: ejecuta «pnpm run capturas» antes.`);
  return `data:image/png;base64,${readFileSync(fichero).toString("base64")}`;
}

/** Las cinco pantallas: el nombre del fichero, el título, la frase y la captura. */
const pantallas = [
  [
    "1-rellena",
    "Rellena al entrar, sin pulsar nada",
    "Con una cuenta guardada del sitio, escribe el usuario y la contraseña, y te dice que ha sido Esfinge.",
    "campo-web-clara.png",
  ],
  [
    "2-panel",
    "Tus cuentas del sitio, a un clic",
    "Rellena la que elijas, o copia su contraseña y su código de un solo uso.",
    "varias-claro.png",
  ],
  [
    "3-guardar",
    "Guarda lo que escribes",
    "Al entrar, registrarte o cambiar la contraseña, te ofrece guardarla en tu bóveda.",
    "tarjeta-guardar-web-oscura.png",
  ],
  [
    "4-tus-datos",
    "Nada sale de tu ordenador",
    "La extensión habla solo con Esfinge, en tu ordenador, y te lo explica antes de empezar.",
    "aviso-claro.png",
  ],
  [
    "5-cerrada",
    "Sabe cuándo la bóveda está cerrada",
    "Y te dice qué hacer, sin rellenar nada hasta que la abras en Esfinge.",
    "cerrada-oscuro.png",
  ],
];

const ESTILO = `
  * { box-sizing: border-box; }
  body { margin: 0; background: ${PIEDRA}; color: ${BLANCO};
         font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif; }
  .marca svg { display: block; width: 100%; height: 100%; }
`;

const navegador = await chromium.launch();

for (const [nombre, titulo, frase, imagen] of pantallas) {
  const pagina = await navegador.newPage({ viewport: { width: 1280, height: 800 }, deviceScaleFactor: 1 });
  await pagina.setContent(`<!doctype html><meta charset="utf-8"><style>${ESTILO}
    .lienzo { width: 1280px; height: 800px; display: grid; grid-template-columns: 470px 1fr;
              align-items: center; gap: 48px; padding: 0 72px; }
    .marca { width: 56px; height: 56px; margin-bottom: 28px; }
    h1 { margin: 0 0 18px; font-size: 44px; line-height: 1.12; font-weight: 700; }
    p { margin: 0; font-size: 22px; line-height: 1.45; color: ${SUAVE}; }
    .captura { display: grid; place-items: center; }
    .captura img { max-width: 640px; max-height: 640px; border-radius: 14px;
                   box-shadow: 0 24px 60px rgb(0 0 0 / 0.45); }
  </style>
  <div class="lienzo">
    <div>
      <div class="marca">${silueta}</div>
      <h1></h1>
      <p></p>
    </div>
    <div class="captura"><img src="${captura(imagen)}" alt=""></div>
  </div>`);
  // Los textos con textContent, como en la extensión.
  await pagina.evaluate(
    ([t, f]) => {
      document.querySelector("h1").textContent = t;
      document.querySelector("p").textContent = f;
    },
    [titulo, frase],
  );
  await pagina.waitForTimeout(150);
  await pagina.screenshot({ path: join(salida, `captura-${nombre}.png`) });
  await pagina.close();
}

// El mosaico promocional pequeño.
const mosaico = await navegador.newPage({ viewport: { width: 440, height: 280 }, deviceScaleFactor: 1 });
await mosaico.setContent(`<!doctype html><meta charset="utf-8"><style>${ESTILO}
  .lienzo { width: 440px; height: 280px; display: flex; flex-direction: column; justify-content: center;
            padding: 0 40px; }
  .fila { display: flex; align-items: center; gap: 16px; margin-bottom: 14px; }
  .marca { width: 60px; height: 60px; }
  strong { font-size: 40px; line-height: 1; font-weight: 700; }
  p { margin: 0; font-size: 19px; line-height: 1.4; color: ${SUAVE}; }
</style>
<div class="lienzo">
  <div class="fila"><div class="marca">${silueta}</div><strong>Esfinge</strong></div>
  <p>Tus contraseñas, rellenadas en el navegador y guardadas en tu ordenador.</p>
</div>`);
await mosaico.waitForTimeout(150);
await mosaico.screenshot({ path: join(salida, "mosaico-440x280.png") });

await navegador.close();
console.log(`Imágenes de las fichas en ${salida}`);
