#!/usr/bin/env node
// Saca de `web/privacidad.html` la copia en texto que pide addons.mozilla.org,
// `docs/tiendas/privacidad-amo.txt`.
//
// **Por qué se genera y no se escribe.** La ADR 0033 dice que la web, el aviso del
// panel y lo declarado en las dos tiendas tienen que decir lo mismo, y hasta ahora
// esa copia se mantenía a mano. Es la trampa de siempre en este proyecto —lo mismo
// escrito en dos sitios y solo uno completo—: el día que alguien cambie la política
// y no vuelva a pegar el texto, la ficha de Firefox dirá otra cosa que la web, y
// nadie se enterará porque las dos siguen existiendo.
//
// Con `--comprobar` no escribe nada y falla si lo guardado no es lo que saldría.
// Eso es lo que corre en `make comprobar`: **lo que hay que respetar tiene que
// parar algo**.

import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const RAIZ = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const FUENTE = resolve(RAIZ, "web/privacidad.html");
const DESTINO = resolve(RAIZ, "docs/tiendas/privacidad-amo.txt");

const CABECERA = `# Esta es la política de privacidad de web/privacidad.html en texto plano, para pegarla donde
# addons.mozilla.org pide el texto («Esta extensión tiene una política de privacidad»). Se genera
# desde la web con \`node navegador/herramientas/politica-a-texto.mjs\`, y \`make comprobar\` falla si
# no está al día. (Estas cuatro líneas no se pegan.)`;

/** Un trozo de HTML a texto: sin etiquetas, sin entidades y sin espacios de sobra. */
function texto(html) {
  return html
    .replace(/<[^>]+>/g, "")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/\s+/g, " ")
    .trim();
}

function convertir(html) {
  const dentro = /<article class="lectura documento">([\s\S]*?)<\/article>/.exec(html);
  if (!dentro) throw new Error("no se encuentra el artículo de la política en web/privacidad.html");
  const bloques = [];
  // Los bloques de primer nivel, en el orden en que están. Una lista se parte en
  // sus puntos; lo demás va tal cual.
  const trozos = dentro[1].matchAll(/<(h1|h2|h3|p|ul)\b[^>]*>([\s\S]*?)<\/\1>/g);
  for (const [, etiqueta, cuerpo] of trozos) {
    if (etiqueta === "ul") {
      for (const [, punto] of cuerpo.matchAll(/<li\b[^>]*>([\s\S]*?)<\/li>/g)) {
        const t = texto(punto);
        if (t) bloques.push(`- ${t}`);
      }
    } else {
      const t = texto(cuerpo);
      if (t) bloques.push(t);
    }
  }
  if (bloques.length < 20) throw new Error(`la política parece cortada (${bloques.length} bloques)`);
  return `${CABECERA}\n\n${bloques.join("\n\n")}\n`;
}

const salida = convertir(readFileSync(FUENTE, "utf8"));

if (process.argv.includes("--comprobar")) {
  let guardado = "";
  try {
    guardado = readFileSync(DESTINO, "utf8");
  } catch {
    /* no está: se dirá abajo */
  }
  if (guardado !== salida) {
    console.error(
      "docs/tiendas/privacidad-amo.txt no dice lo mismo que web/privacidad.html.\n" +
        "Vuelve a sacarlo con: node navegador/herramientas/politica-a-texto.mjs",
    );
    process.exit(1);
  }
  console.log("La política de la ficha de Firefox dice lo mismo que la web.");
} else {
  writeFileSync(DESTINO, salida);
  console.log(`Escrito ${DESTINO} (${salida.length} caracteres).`);
}
