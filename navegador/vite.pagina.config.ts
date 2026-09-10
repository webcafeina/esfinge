import { defineConfig } from "vite";
import { resolve } from "node:path";

/**
 * El guion que se pone en las páginas, **en una sola pieza y sin módulos**.
 *
 * Por lo mismo que el trabajador de fondo, y aquí con menos remedio todavía: un
 * guion de contenido declarado en el manifiesto **no puede ser un módulo**. Si el
 * empaquetador saca un trozo común con el panel y mete un `import`, el navegador
 * lo carga y falla en la primera línea, en la página de otro y sin que nadie lo
 * vea. Compilado aparte no hay nada que importar.
 *
 * `emptyOutDir` va apagado a propósito: el panel y el trabajador ya están ahí.
 */
const navegador = process.env.NAVEGADOR === "firefox" ? "firefox" : "chrome";

export default defineConfig({
  build: {
    outDir: resolve(__dirname, "dist", navegador),
    emptyOutDir: false,
    lib: {
      entry: resolve(__dirname, "src/pagina.ts"),
      formats: ["iife"],
      name: "EsfingePagina",
      fileName: () => "pagina.js",
    },
  },
});
