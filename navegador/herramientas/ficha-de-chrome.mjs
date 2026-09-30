#!/usr/bin/env node
// Vigila que lo que hay que pegar a mano en la consola de Chrome esté pegado.
//
// **Por qué hace falta una herramienta para esto.** La ficha de Firefox viaja con
// cada publicación (`amo-metadata.json` en `publicar.yml`), pero **la de Chrome se
// escribe en su consola y su API no la edita**: sube el paquete y nada más. Así que
// cambiar `docs/tiendas/ficha.md` no cambia lo que la tienda enseña, y **no hay nada
// que avise**.
//
// No es una hipótesis: lo que la entrega de compartir cambió el 2026-09-24 nunca se
// pegó, y la ficha se quedó seis días declarando algo distinto de lo que la extensión
// hacía. Lo vio el cliente el 2026-09-30 buscando en su consola una línea que se daba
// por puesta. Es la misma trampa que `politica-a-texto.mjs` cerró para el texto de la
// política de Firefox, dejada abierta para Chrome.
//
// **Cómo lo hace:** los bloques que se pegan van marcados en `ficha.md`, y aquí se
// guarda su huella. Si la huella cambia y nadie ha dicho que lo ha pegado, esto falla.
// Se hashea **solo lo marcado** y no el fichero entero, a propósito: un aviso que salta
// porque alguien arregló una coma de una nota interna se deja de leer, y este proyecto
// ya sabe lo que cuesta un aviso que no significa nada.
//
// Lo que **no** puede comprobar, y hay que decirlo: que lo pegado sea de verdad esto.
// Sigue siendo palabra de quien pasa `--pegado`. Lo que consigue es que **haya que
// decirlo**, en vez de olvidarse sin que nada pase.

import { createHash } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const RAIZ = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const FICHA = resolve(RAIZ, "docs/tiendas/ficha.md");
const ESTADO = resolve(RAIZ, "docs/tiendas/pegado-en-chrome.txt");

const EMPIEZA = "<!-- consola de Chrome: empieza -->";
const ACABA = "<!-- consola de Chrome: acaba -->";

/**
 * Los trozos marcados, en orden.
 *
 * Se exige que las marcas estén emparejadas **y que haya alguna**: un `ficha.md` sin
 * marcas daría una huella estable y esto pasaría en verde sin vigilar nada, que es
 * justo el fallo mudo que la herramienta existe para evitar.
 */
function loQueSePega(texto) {
  const trozos = [];
  let desde = 0;
  for (;;) {
    const a = texto.indexOf(EMPIEZA, desde);
    if (a === -1) break;
    const b = texto.indexOf(ACABA, a);
    if (b === -1) throw new Error(`una marca de «empieza» sin su «acaba» (carácter ${a})`);
    trozos.push(texto.slice(a + EMPIEZA.length, b).trim());
    desde = b + ACABA.length;
  }
  if (texto.indexOf(ACABA, desde) !== -1) throw new Error("un «acaba» sin su «empieza»");
  if (trozos.length === 0) throw new Error("ficha.md no tiene ningún bloque marcado para la consola de Chrome");
  return trozos;
}

const trozos = loQueSePega(readFileSync(FICHA, "utf8"));
const huella = createHash("sha256").update(trozos.join("\n\n")).digest("hex").slice(0, 16);

const CABECERA = `# Lo que hay pegado en la consola de la Chrome Web Store, y cuándo se pegó.
#
# La ficha de Chrome no la actualiza el flujo de publicación: su API sube el paquete y no
# edita la ficha. Así que los bloques marcados de docs/tiendas/ficha.md se pegan a mano, y
# esto es lo que hace que olvidarse pare algo en vez de pasar desapercibido.
#
# Tras pegarlos: node navegador/herramientas/ficha-de-chrome.mjs --pegado`;

const escribir = () => {
  const hoy = new Date().toISOString().slice(0, 10);
  writeFileSync(ESTADO, `${CABECERA}\n\nhuella: ${huella}\nfecha: ${hoy}\n`);
  console.log(`Apuntado: la ficha de Chrome está al día (${huella}, ${hoy}).`);
};

if (process.argv.includes("--pegado")) {
  escribir();
} else {
  let guardado = "";
  try {
    guardado = readFileSync(ESTADO, "utf8");
  } catch {
    /* no está: se dirá abajo */
  }
  const puesta = /^huella: (\w+)$/m.exec(guardado)?.[1] ?? "";
  if (puesta !== huella) {
    console.error(
      "Lo que hay que pegar en la consola de Chrome ha cambiado y la ficha de la tienda sigue como estaba.\n" +
        `  ahora: ${huella}\n  pegado: ${puesta || "nada"}\n\n` +
        "Los bloques están marcados en docs/tiendas/ficha.md entre «consola de Chrome: empieza» y «acaba».\n" +
        "Se pegan en https://chrome.google.com/webstore/devconsole y luego:\n" +
        "  node navegador/herramientas/ficha-de-chrome.mjs --pegado",
    );
    process.exit(1);
  }
  console.log("La ficha de la Chrome Web Store está al día con ficha.md.");
}
