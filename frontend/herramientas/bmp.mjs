// Dibuja el lockup de Esfinge y lo deja en los BMP que pide el instalador.
//
// **Con Chromium, como el resto de los iconos** (`rasterizar.mjs`): en esta
// máquina no hay ningún conversor de SVG y Playwright ya trae un navegador. La
// diferencia es el final: NSIS no lee PNG, así que de aquí sale un BMP.
//
// # Por qué da dos vueltas
//
// El dibujo lleva **texto con una tipografía que no está instalada**, así que la
// letra entra como `@font-face` del documento. Y un SVG cargado en un `<img>`
// **no ve las tipografías del documento**: está aislado, y lo único que usaría es
// una que viniera dentro del propio SVG. Por eso no se puede ir directo al
// lienzo. El camino es: se dibuja en el documento, se retrata el elemento, y ese
// retrato —que ya es una imagen, con la letra dentro— se devuelve al lienzo para
// sacarle los píxeles.
//
// # El BMP
//
// De 24 bits sin comprimir, que es lo que lee cualquier versión de NSIS: su
// documentación solo dice «bmp_file», ni profundidad ni compresión, así que se
// elige lo más llano que existe. Guarda las filas **de abajo arriba** y los
// canales en BGR, que son las dos cosas que se olvidan y dan una imagen
// invertida o azulada.
import { chromium } from "@playwright/test";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";

const raiz = resolve(import.meta.dirname, "..", "..");
const LETRA = "Space Grotesk";

// **`replaceAll` y no `replace`**, y el marcador armado en dos trozos para que no
// aparezca entero ni aquí ni en un comentario de los dibujos. La primera versión
// usaba `replace`, que sustituye solo la primera vez, y el comentario de
// `lado.html` nombraba el marcador para explicarlo: se sustituyó el del
// comentario y el de verdad quedó escrito en la banda. Es la misma trampa que ya
// costó una publicación en `herramientas/rellenar`.
const MARCADOR = "{{" + "LOGO" + "}}";

// Los tamaños **son los que recomienda MUI2** y no un múltiplo: no hay forma
// soportada de dar una versión de más resolución para pantallas al 200 %.
const trabajos = [
  ["build/windows/instalador/lado.html", "build/windows/instalador/lado.bmp", 164, 314],
  ["build/windows/instalador/cabecera.html", "build/windows/instalador/cabecera.bmp", 150, 57],
];

function aBMP(pixeles, ancho, alto) {
  // Cada fila se rellena hasta un múltiplo de cuatro bytes. Sin esto, cualquier
  // ancho que no sea múltiplo de cuatro sale torcido en diagonal.
  const sobra = (4 - ((ancho * 3) % 4)) % 4;
  const datos = Buffer.alloc((ancho * 3 + sobra) * alto);
  let p = 0;
  for (let y = alto - 1; y >= 0; y--) {
    for (let x = 0; x < ancho; x++) {
      const i = (y * ancho + x) * 4;
      datos[p++] = pixeles[i + 2];
      datos[p++] = pixeles[i + 1];
      datos[p++] = pixeles[i];
    }
    p += sobra;
  }
  const cabecera = Buffer.alloc(54);
  cabecera.write("BM", 0);
  cabecera.writeUInt32LE(54 + datos.length, 2);
  cabecera.writeUInt32LE(54, 10);
  cabecera.writeUInt32LE(40, 14);
  cabecera.writeInt32LE(ancho, 18);
  cabecera.writeInt32LE(alto, 22);
  cabecera.writeUInt16LE(1, 26);
  cabecera.writeUInt16LE(24, 28);
  cabecera.writeUInt32LE(datos.length, 34);
  return Buffer.concat([cabecera, datos]);
}

const tipografia = readFileSync(
  resolve(raiz, "build/tipografias/space-grotesk/SpaceGrotesk.ttf"),
).toString("base64");
// **Solo del `<svg>`, no de todo el fichero.** Quitando `width`/`height` a lo
// bruto se los lleva también al `<rect>` del fondo, y entonces el logo sale sin
// su placa: se ve la esfinge flotando. Se vio mirando el dibujo, no leyendo.
const logo = readFileSync(resolve(raiz, "build/icono.svg"), "utf8")
  .replace(/<\?xml[^>]*\?>/, "")
  .replace(/<svg\b[^>]*>/, (tag) => tag.replace(/ (width|height)="1024"/g, ""));

const navegador = await chromium.launch();
const pagina = await navegador.newPage({ deviceScaleFactor: 1 });

for (const [desde, hasta, ancho, alto] of trabajos) {
  const cuerpo = readFileSync(resolve(raiz, desde), "utf8");
  if (!cuerpo.includes(MARCADOR)) {
    console.error(`!! ${desde} no tiene dónde poner el logo. No escribo nada.`);
    process.exit(1);
  }
  await pagina.setContent(`<!doctype html>
    <style>
      @font-face {
        font-family: "${LETRA}";
        src: url(data:font/ttf;base64,${tipografia}) format("truetype");
        font-weight: 300 700;
      }
      * { box-sizing: border-box; }
      html, body { margin: 0; padding: 0; background: #fff; }
      #lienzo { width: ${ancho}px; height: ${alto}px; overflow: hidden; position: relative; }
      #lienzo svg.logo { display: block; }
      .nombre { font-family: "${LETRA}"; font-weight: 700; }
    </style>
    <div id="lienzo">${cuerpo.replaceAll(MARCADOR, logo)}</div>`);

  // Y que no haya quedado ningún marcador escrito en el dibujo, que es como se
  // vio la primera vez: salió la banda con el texto del marcador dentro.
  const sobra = await pagina.evaluate((m) => document.body.textContent.includes(m), MARCADOR);
  if (sobra) {
    console.error(`!! en ${desde} ha quedado un marcador sin sustituir. No escribo nada.`);
    process.exit(1);
  }

  await pagina.evaluate(() => document.fonts.ready);
  // **Y se comprueba que la letra está**, que es lo que separa un dibujo bueno
  // de uno con otra tipografía y nadie enterándose: un `@font-face` que no carga
  // no da error, da la letra de respaldo.
  const hayLetra = await pagina.evaluate((n) => document.fonts.check(`700 20px "${n}"`), LETRA);
  if (!hayLetra) {
    console.error(`!! «${LETRA}» no ha cargado: el dibujo saldría con otra letra. No escribo nada.`);
    process.exit(1);
  }

  const png = await pagina.locator("#lienzo").screenshot();
  const pixeles = await pagina.evaluate(
    async ({ b64, ancho, alto }) => {
      const img = new Image();
      await new Promise((ok, mal) => {
        img.onload = ok;
        img.onerror = mal;
        img.src = `data:image/png;base64,${b64}`;
      });
      const l = document.createElement("canvas");
      l.width = ancho;
      l.height = alto;
      const ctx = l.getContext("2d");
      // **Opaco a propósito**: un BMP de 24 bits no guarda transparencia, y sin
      // esto lo que no pinte el dibujo saldría negro en vez de blanco.
      ctx.fillStyle = "#ffffff";
      ctx.fillRect(0, 0, ancho, alto);
      ctx.drawImage(img, 0, 0, ancho, alto);
      return Array.from(ctx.getImageData(0, 0, ancho, alto).data);
    },
    { b64: png.toString("base64"), ancho, alto },
  );

  const destino = resolve(raiz, hasta);
  mkdirSync(dirname(destino), { recursive: true });
  writeFileSync(destino, aBMP(pixeles, ancho, alto));
  console.log(`· ${hasta} (${ancho}×${alto})`);
}
await navegador.close();
