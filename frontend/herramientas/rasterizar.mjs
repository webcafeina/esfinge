// Convierte el icono SVG a los PNG que piden los sistemas.
//
// Se hace con Chromium porque es lo que hay: en esta máquina no está instalado
// ningún conversor de SVG, y Playwright ya viene con un navegador para las
// pruebas de la interfaz. Wails genera desde appicon.png los tamaños de cada
// plataforma, así que con el de 1024 basta; los demás son para el favicon.
import { chromium } from "@playwright/test";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";

const raiz = resolve(import.meta.dirname, "..", "..");
const svg = readFileSync(resolve(raiz, "build/icono.svg"), "utf8");

const salidas = [
  ["build/appicon.png", 1024],
  ["frontend/public/icono-256.png", 256],
  ["frontend/public/favicon.png", 64],
];

const navegador = await chromium.launch(
  process.env.PLAYWRIGHT_CHROMIUM ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM } : {},
);

for (const [destino, tamano] of salidas) {
  const pagina = await navegador.newPage({
    viewport: { width: tamano, height: tamano },
    deviceScaleFactor: 1,
  });
  await pagina.setContent(
    `<style>html,body{margin:0;padding:0}svg{display:block;width:${tamano}px;height:${tamano}px}</style>${svg}`,
  );
  const png = await pagina.locator("svg").screenshot({ omitBackground: true });
  const ruta = resolve(raiz, destino);
  mkdirSync(dirname(ruta), { recursive: true });
  writeFileSync(ruta, png);
  console.log(`${destino} · ${tamano}×${tamano} · ${(png.length / 1024).toFixed(1)} kB`);
  await pagina.close();
}

await navegador.close();
