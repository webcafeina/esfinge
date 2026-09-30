import { defineConfig } from "vite";
import { resolve } from "node:path";

/**
 * El guion del **mundo principal**, en una sola pieza y sin módulos.
 *
 * Por lo mismo que `vite.pagina.config.ts`, y aquí con menos remedio todavía: éste
 * corre **dentro de la página**, donde un `import` que reventara se lo llevaría por
 * delante todo sin que nadie lo vea. Compilado aparte no hay nada que importar.
 */
const navegador = process.env.NAVEGADOR === "firefox" ? "firefox" : "chrome";

export default defineConfig({
  build: {
    outDir: process.env.ESFINGE_CUENTAS_PRUEBAS ? resolve(__dirname, "dist", "pruebas") : resolve(__dirname, "dist", navegador),
    emptyOutDir: false,
    lib: {
      entry: resolve(__dirname, "src/mundo.ts"),
      formats: ["iife"],
      name: "EsfingeMundo",
      fileName: () => "mundo.js",
    },
  },
});
