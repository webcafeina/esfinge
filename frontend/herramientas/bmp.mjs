// Convierte los SVG del instalador de Windows en los BMP que pide NSIS.
//
// **Con Chromium, como el resto de los iconos** (`rasterizar.mjs`): en esta
// máquina no hay ningún conversor de SVG instalado y Playwright ya trae un
// navegador para las pruebas. La diferencia es el final: NSIS no lee PNG, así
// que los píxeles se sacan del lienzo y el BMP se escribe aquí.
//
// **BMP de 24 bits sin comprimir**, que es lo que lee cualquier versión de NSIS.
// La documentación de MUI2 solo dice «bmp_file» —ni profundidad ni compresión—,
// así que se elige lo más llano que existe en vez de adivinar qué más admite.
// Y el formato guarda las filas **de abajo arriba** y los canales en BGR, que
// son las dos cosas que se olvidan y dan una imagen invertida o azulada.
import { chromium } from "@playwright/test";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";

const raiz = resolve(import.meta.dirname, "..", "..");

// Los tamaños **son los que la documentación de MUI2 recomienda**, y no un
// múltiplo: no hay forma soportada de dar una versión de más resolución para
// pantallas al 200 %. Lo que se estira es ésta, así que los dibujos van sin una
// sola letra dentro y con formas grandes. Los textos los pinta Windows.
const trabajos = [
  ["build/windows/instalador/lado.svg", "build/windows/instalador/lado.bmp", 164, 314],
  ["build/windows/instalador/cabecera.svg", "build/windows/instalador/cabecera.bmp", 150, 57],
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

const navegador = await chromium.launch();
const pagina = await navegador.newPage();
for (const [desde, hasta, ancho, alto] of trabajos) {
  const svg = readFileSync(resolve(raiz, desde), "utf8");
  const pixeles = await pagina.evaluate(
    async ({ svg, ancho, alto }) => {
      const url = URL.createObjectURL(new Blob([svg], { type: "image/svg+xml" }));
      const img = new Image();
      await new Promise((ok, mal) => {
        img.onload = ok;
        img.onerror = mal;
        img.src = url;
      });
      const lienzo = document.createElement("canvas");
      lienzo.width = ancho;
      lienzo.height = alto;
      const ctx = lienzo.getContext("2d");
      // **Opaco a propósito**: un BMP de 24 bits no guarda transparencia, y sin
      // esto lo que no pinte el dibujo saldría negro en vez de blanco.
      ctx.fillStyle = "#ffffff";
      ctx.fillRect(0, 0, ancho, alto);
      ctx.drawImage(img, 0, 0, ancho, alto);
      return Array.from(ctx.getImageData(0, 0, ancho, alto).data);
    },
    { svg, ancho, alto },
  );
  const destino = resolve(raiz, hasta);
  mkdirSync(dirname(destino), { recursive: true });
  writeFileSync(destino, aBMP(pixeles, ancho, alto));
  console.log(`· ${hasta} (${ancho}×${alto})`);
}
await navegador.close();
