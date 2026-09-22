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

/**
 * **Solo para las pruebas con la extensión cargada** (ADR 0040): el servidor de
 * cuentas al que habla, en vez del de producción, y en una carpeta aparte para que
 * nunca se confunda con la que se publica. Sin la variable, nada cambia.
 */
const pruebas = process.env.ESFINGE_CUENTAS_PRUEBAS ?? "";

export default defineConfig({
  define: { __RAIZ_CUENTAS__: JSON.stringify(pruebas) },
  build: {
    outDir: pruebas ? resolve(__dirname, "dist", "pruebas") : resolve(__dirname, "dist", navegador),
    emptyOutDir: false,
    lib: {
      entry: resolve(__dirname, "src/fondo.ts"),
      formats: ["iife"],
      name: "EsfingeFondo",
      fileName: () => "fondo.js",
    },
  },
});
