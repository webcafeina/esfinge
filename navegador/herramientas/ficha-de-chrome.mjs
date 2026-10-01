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
//
// **Y hay un aplazamiento, porque el freno se trababa.** Chrome no deja editar la ficha
// mientras revisa una versión, así que entre cambiar el texto y poder pegarlo pasan
// días — y en ese hueco esto dejaba sin publicar **todo**, incluidos Firefox y las
// descargas de GitHub, que no tienen nada que ver con esa ficha. Apareció el mismo día
// de escribirlo.
//
// El aplazamiento **caduca**, y ahí está la diferencia con quitar el freno: esperar es
// algo acotado y con motivo escrito, olvidarse sigue parando. Pasados los días, vuelve
// a fallar solo.

import { createHash } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const RAIZ = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const FICHA = resolve(RAIZ, "docs/tiendas/ficha.md");
const ESTADO = resolve(RAIZ, "docs/tiendas/pegado-en-chrome.txt");

/**
 * La marca de apertura, que **puede llevar el tope del campo**:
 * `<!-- consola de Chrome: empieza (máximo 1000) -->`.
 *
 * Los campos de la consola tienen topes de caracteres y desde aquí **no se ven**, así
 * que el tope lo trae quien los mira. El de la justificación de los permisos de host son
 * mil, y se supo el 2026-10-01 **al ir a pegar** un texto de 1197: escribirlo y que no
 * entre es la clase de fallo que se descubre en el peor sitio. Declarado aquí, para ese
 * campo pasarse **para** la comprobación.
 *
 * Un bloque sin tope no se mide: no sabemos cuál es, y adivinarlo sería peor que no
 * comprobar nada.
 */
const EMPIEZA = /<!-- consola de Chrome: empieza(?: \(máximo (\d+)\))? -->/g;
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
  const largos = [];
  let desde = 0;
  EMPIEZA.lastIndex = 0;
  for (let m = EMPIEZA.exec(texto); m; m = EMPIEZA.exec(texto)) {
    const a = m.index;
    const b = texto.indexOf(ACABA, a);
    if (b === -1) throw new Error(`una marca de «empieza» sin su «acaba» (carácter ${a})`);
    const trozo = texto.slice(a + m[0].length, b).trim();
    trozos.push(trozo);
    if (m[1]) largos.push({ tope: Number(m[1]), tiene: trozo.length, trozo });
    desde = b + ACABA.length;
    EMPIEZA.lastIndex = desde;
  }
  if (texto.indexOf(ACABA, desde) !== -1) throw new Error("un «acaba» sin su «empieza»");
  if (trozos.length === 0) throw new Error("ficha.md no tiene ningún bloque marcado para la consola de Chrome");
  for (const { tope, tiene, trozo } of largos) {
    if (tiene > tope) {
      throw new Error(
        `un bloque se pasa del tope del campo: ${tiene} caracteres y caben ${tope}\n` +
          `  empieza por «${trozo.slice(0, 60)}…»\n` +
          "  El tope lo pone la consola de Chrome y está escrito en la marca de ese bloque.",
      );
    }
  }
  return trozos;
}

const trozos = loQueSePega(readFileSync(FICHA, "utf8"));
const huella = createHash("sha256").update(trozos.join("\n\n")).digest("hex").slice(0, 16);

/** Cuánto vale un aplazamiento. Una revisión de Chrome tarda días, no semanas. */
const DIAS_DE_APLAZAMIENTO = 7;

const hoy = () => new Date().toISOString().slice(0, 10);

/** Los días entre dos fechas `AAAA-MM-DD`, sin horas que compliquen nada. */
function diasDesde(fecha) {
  const d = Date.parse(fecha + "T00:00:00Z");
  if (Number.isNaN(d)) return Infinity;
  return Math.floor((Date.parse(hoy() + "T00:00:00Z") - d) / 86400000);
}

const CABECERA = `# Lo que hay pegado en la consola de la Chrome Web Store, y cuándo se pegó.
#
# La ficha de Chrome no la actualiza el flujo de publicación: su API sube el paquete y no
# edita la ficha. Así que los bloques marcados de docs/tiendas/ficha.md se pegan a mano, y
# esto es lo que hace que olvidarse pare algo en vez de pasar desapercibido.
#
# Tras pegarlos: node navegador/herramientas/ficha-de-chrome.mjs --pegado
#
# Y si Chrome no deja editarla todavía —está revisando otra versión—, se aplaza con
# motivo:  node navegador/herramientas/ficha-de-chrome.mjs --aplazado "por qué"
# El aplazamiento caduca a los ${DIAS_DE_APLAZAMIENTO} días y entonces vuelve a parar.`;

const escribir = () => {
  writeFileSync(ESTADO, `${CABECERA}\n\nhuella: ${huella}\nfecha: ${hoy()}\n`);
  console.log(`Apuntado: la ficha de Chrome está al día (${huella}, ${hoy()}).`);
};

const aplazar = (porque) => {
  const guardado = leer();
  const pegada = /^huella: (\w+)$/m.exec(guardado)?.[1] ?? "";
  const fechaPegada = /^fecha: ([\d-]+)$/m.exec(guardado)?.[1] ?? "";
  writeFileSync(
    ESTADO,
    `${CABECERA}\n\nhuella: ${pegada}\nfecha: ${fechaPegada}\n` +
      `pendiente: ${huella}\naplazado: ${hoy()}\nporque: ${porque}\n`,
  );
  console.log(
    `Aplazado hasta ${DIAS_DE_APLAZAMIENTO} días: ${porque}\n` +
      "La ficha de Chrome sigue sin pegar, y esto lo volverá a decir cada vez.",
  );
};

function leer() {
  try {
    return readFileSync(ESTADO, "utf8");
  } catch {
    return "";
  }
}

const porque = (() => {
  const i = process.argv.indexOf("--aplazado");
  return i === -1 ? null : (process.argv[i + 1] ?? "").trim();
})();

if (process.argv.includes("--pegado")) {
  escribir();
} else if (porque !== null) {
  if (!porque) {
    console.error("Un aplazamiento sin motivo no vale: di por qué no se puede pegar todavía.");
    process.exit(1);
  }
  aplazar(porque);
} else {
  const guardado = leer();
  const puesta = /^huella: (\w+)$/m.exec(guardado)?.[1] ?? "";
  const pendiente = /^pendiente: (\w+)$/m.exec(guardado)?.[1] ?? "";
  const aplazado = /^aplazado: ([\d-]+)$/m.exec(guardado)?.[1] ?? "";
  const razon = /^porque: (.+)$/m.exec(guardado)?.[1] ?? "";

  // **Un aplazamiento vale mientras no caduque y siga siendo el mismo texto.** Si
  // `ficha.md` vuelve a cambiar, la huella deja de cuadrar y esto para otra vez: lo
  // aplazado era aquello, no lo que se escriba después.
  if (puesta !== huella && pendiente === huella) {
    const dias = diasDesde(aplazado);
    if (dias <= DIAS_DE_APLAZAMIENTO) {
      console.log(
        `La ficha de Chrome está sin pegar, aplazado hace ${dias} ${dias === 1 ? "día" : "días"}: ${razon}\n` +
          `  Caduca a los ${DIAS_DE_APLAZAMIENTO} y entonces esto vuelve a fallar.\n` +
          "  Cuando se pegue: node navegador/herramientas/ficha-de-chrome.mjs --pegado",
      );
      process.exit(0);
    }
    console.error(
      `El aplazamiento de la ficha de Chrome ha caducado: ${dias} días desde «${razon}».\n` +
        "Si sigue sin poder pegarse, vuelve a aplazarlo diciendo por qué.",
    );
    process.exit(1);
  }

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
