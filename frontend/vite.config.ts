import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// En desarrollo, lo que la interfaz le pide a Go va al servidor que levanta
// «make dev». En producción no hay servidor: las mismas llamadas las atiende
// Wails dentro del propio proceso. De eso se encarga src/puente.ts.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // ESFINGE_GO deja levantar varias ventanas a la vez, cada una con su Go: las
    // pruebas de la cuenta necesitan dos equipos (e2e/cuentas.spec.ts).
    proxy: { "/api": process.env.ESFINGE_GO ?? "http://127.0.0.1:34443" },
  },
  build: { outDir: "dist", emptyOutDir: true },
});
