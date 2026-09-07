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
// Cada entrada: fichero SVG de origen, destino, y ancho en píxeles. El alto sale
// de la proporción del propio SVG.
const trabajos = [
  ["build/icono.svg", "build/appicon.png", 1024],
  ["build/icono.svg", "frontend/public/icono-256.png", 256],
  ["build/icono.svg", "frontend/public/favicon.png", 64],
  ["build/icono.svg", "docs/imagenes/icono.png", 256],

  // Los tamaños que espera el tema de iconos de Linux. Con uno solo, cada
  // entorno se lo reescala como puede y en el menú de aplicaciones se nota;
  // dárselos hechos es lo que hace cualquier paquete del sistema.
  ...[16, 24, 32, 48, 64, 128, 256, 512].map((n) => [
    "build/icono.svg",
    `build/linux/hicolor/${n}x${n}.png`,
    n,
  ]),
  // El fondo del DMG va al doble de la ventana, para pantallas Retina: macOS lo
  // reduce a la mitad y así no se ve borroso.
  ["build/darwin/fondo-dmg.svg", "build/darwin/fondo-dmg.png", 660],
  ["build/darwin/fondo-dmg.svg", "build/darwin/fondo-dmg@2x.png", 1320],

  // El icono del volumen montado, en los tamaños que pide «iconutil» para armar
  // el .icns. Los nombres son los suyos y no se pueden cambiar: si uno falta o
  // se llama distinto, iconutil se niega a construir el fichero.
  ...[16, 32, 128, 256, 512].flatMap((n) => [
    [`build/darwin/disco.svg`, `build/darwin/disco.iconset/icon_${n}x${n}.png`, n],
    [`build/darwin/disco.svg`, `build/darwin/disco.iconset/icon_${n}x${n}@2x.png`, n * 2],
  ]),
];

const navegador = await chromium.launch(
  process.env.PLAYWRIGHT_CHROMIUM ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM } : {},
);

for (const [origen, destino, ancho] of trabajos) {
  const svg = readFileSync(resolve(raiz, origen), "utf8");

  // La proporción se lee del viewBox, para no tener que repetirla aquí.
  const [, , anchoSvg, altoSvg] = (svg.match(/viewBox="([^"]+)"/)?.[1] ?? "0 0 1 1")
    .split(/\s+/)
    .map(Number);
  const alto = Math.round((ancho * altoSvg) / anchoSvg);

  const pagina = await navegador.newPage({
    viewport: { width: ancho, height: alto },
    deviceScaleFactor: 1,
  });
  await pagina.setContent(
    `<style>html,body{margin:0;padding:0}svg{display:block;width:${ancho}px;height:${alto}px}</style>${svg}`,
  );
  const png = await pagina.locator("svg").screenshot({ omitBackground: true });
  const ruta = resolve(raiz, destino);
  mkdirSync(dirname(ruta), { recursive: true });
  writeFileSync(ruta, png);
  console.log(`${destino} · ${ancho}×${alto} · ${(png.length / 1024).toFixed(1)} kB`);
  await pagina.close();
}

await navegador.close();
