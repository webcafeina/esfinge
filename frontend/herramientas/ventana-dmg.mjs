// Dibuja cómo quedará la ventana del DMG, sin necesidad de un Mac.
//
// El fondo se diseña a ciegas: los iconos no los pone el fondo, los coloca
// create-dmg encima, y hasta montar la imagen en un Mac no se sabe si algo se
// pisa. Pasó en la 2.0.2, donde el nombre del LÉEME cayó sobre el aviso de
// Gatekeeper.
//
// Esto lo enseña antes. Las posiciones **se leen del propio armar-dmg.sh**, no
// se repiten aquí: si se mueve un icono allí, esta simulación se mueve con él.
//
//   node frontend/herramientas/ventana-dmg.mjs   (o «make ventana-dmg»)
import { chromium } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const raiz = resolve(import.meta.dirname, "..", "..");
const guion = readFileSync(resolve(raiz, "empaquetado/macos/armar-dmg.sh"), "utf8");

const [, ancho, alto] = guion.match(/--window-size (\d+) (\d+)/).map(Number);
const [, tamañoIcono] = guion.match(/--icon-size (\d+)/).map(Number);
const [, tamañoTexto] = guion.match(/--text-size (\d+)/).map(Number);

// «--icon "Nombre" x y» y «--app-drop-link x y», que es la carpeta Aplicaciones.
const iconos = [
  ...guion.matchAll(/--icon "([^"]+)" (\d+) (\d+)/g),
].map(([, nombre, x, y]) => [Number(x), Number(y), nombre.replace(/\.app$/, "")]);
const enlace = guion.match(/--app-drop-link (\d+) (\d+)/);
if (enlace) iconos.push([Number(enlace[1]), Number(enlace[2]), "Applications"]);

const fondo = readFileSync(resolve(raiz, "build/darwin/fondo-dmg.png")).toString("base64");
const mitad = tamañoIcono / 2;

const html = `<style>
  body { margin: 0; width: ${ancho}px; height: ${alto}px; position: relative;
         background: url(data:image/png;base64,${fondo}) no-repeat;
         background-size: ${ancho}px ${alto}px; }
  .icono { position: absolute; width: ${tamañoIcono}px; text-align: center; }
  .hueco { width: ${tamañoIcono}px; height: ${tamañoIcono}px; border-radius: 18px;
           box-sizing: border-box; background: rgba(255,255,255,.22);
           border: 1px dashed rgba(255,255,255,.55); }
  .nombre { font: ${tamañoTexto}px -apple-system, sans-serif; color: #fff;
            margin-top: 3px; line-height: ${tamañoTexto + 3}px;
            text-shadow: 0 0 3px #000, 0 0 3px #000; }
</style>` + iconos.map(([x, y, nombre]) =>
  `<div class="icono" style="left:${x - mitad}px;top:${y - mitad}px">
     <div class="hueco"></div><div class="nombre">${nombre}</div>
   </div>`).join("");

const navegador = await chromium.launch(
  process.env.PLAYWRIGHT_CHROMIUM ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM } : {},
);
const pagina = await navegador.newPage({
  viewport: { width: ancho, height: alto },
  deviceScaleFactor: 2,
});
await pagina.setContent(html);
const destino = resolve(raiz, "build/darwin/ventana-dmg.png");
await pagina.screenshot({ path: destino });
await navegador.close();

console.log(`build/darwin/ventana-dmg.png · ${ancho}×${alto} · ${iconos.length} iconos`);
for (const [x, y, nombre] of iconos) console.log(`  ${nombre} en (${x}, ${y})`);
