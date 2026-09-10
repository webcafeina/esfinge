import { defineConfig } from "vite";
import { copyFileSync, mkdirSync } from "node:fs";
import { resolve } from "node:path";

/**
 * El panel. **El trabajador de fondo se compila aparte** (`vite.fondo.config.ts`)
 * y no como una entrada más de aquí, y no es manía: compartiendo un módulo entre
 * las dos entradas, el empaquetador saca un trozo común y mete un `import` en el
 * fichero del trabajador. Eso obliga a declararlo como módulo en el manifiesto,
 * que es justo la clase de detalle que funciona en un navegador y no en el otro.
 * Compilado aparte y en una sola pieza, el trabajador no importa nada.
 *
 * Dos compilaciones más, una por navegador, porque **los manifiestos no son el
 * mismo** y la diferencia no es cosmética: Chrome quiere un `service_worker` y
 * Firefox una lista de `scripts`, y el identificador de la extensión lo elige uno
 * en Firefox y lo asigna la tienda en Chrome.
 *
 * Se elige con NAVEGADOR=chrome|firefox. Por defecto, Chrome.
 */
const navegador = process.env.NAVEGADOR === "firefox" ? "firefox" : "chrome";
const salida = resolve(__dirname, "dist", navegador);

export default defineConfig({
  root: resolve(__dirname, "src"),
  build: {
    outDir: salida,
    emptyOutDir: true,
    // Sin trocear ni renombrar: un manifiesto nombra ficheros concretos, y una
    // extensión no gana nada con nombres con huella.
    rollupOptions: {
      input: { panel: resolve(__dirname, "src/panel.html") },
      output: {
        entryFileNames: "[name].js",
        chunkFileNames: "[name].js",
        assetFileNames: "[name].[ext]",
      },
    },
  },
  plugins: [
    {
      name: "manifiesto",
      closeBundle() {
        mkdirSync(salida, { recursive: true });
        copyFileSync(
          resolve(__dirname, `manifiesto.${navegador}.json`),
          resolve(salida, "manifest.json"),
        );
      },
    },
  ],
});
