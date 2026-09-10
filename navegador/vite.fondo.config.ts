import { defineConfig } from "vite";
import { resolve } from "node:path";

/**
 * El trabajador de fondo, **en una sola pieza y sin módulos**.
 *
 * Va en su propia compilación para que no comparta nada con el panel: en cuanto
 * comparten un módulo, el empaquetador saca un trozo común y el trabajador
 * empieza con un `import`. Un fichero con `import` hay que declararlo como módulo
 * en el manifiesto, y el soporte de eso **no es el mismo en los dos
 * navegadores**. Sin imports no hay nada que declarar y no hay nada que falle en
 * uno de los dos.
 *
 * `emptyOutDir` va apagado a propósito: el panel ya está en esa carpeta.
 */
const navegador = process.env.NAVEGADOR === "firefox" ? "firefox" : "chrome";

export default defineConfig({
  build: {
    outDir: resolve(__dirname, "dist", navegador),
    emptyOutDir: false,
    lib: {
      entry: resolve(__dirname, "src/fondo.ts"),
      formats: ["iife"],
      name: "EsfingeFondo",
      fileName: () => "fondo.js",
    },
  },
});
