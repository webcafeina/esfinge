import { defineConfig } from "@playwright/test";

/**
 * La extensión **cargada de verdad** en un Chromium, contra el servidor de cuentas
 * de verdad levantado aquí (ADR 0040). Era la deuda alta de la fase 2: lo que
 * envuelve a la detección de campos —el trabajador de fondo, los puertos, el
 * almacenamiento— no lo había probado nadie con la extensión puesta.
 *
 * Antes de empezar se compila la extensión en modo pruebas (`preparar.ts`), que
 * apunta al servidor local y sale en `dist/pruebas`, lejos de la que se publica.
 */
export const PUERTO = 8793;

export default defineConfig({
  testDir: ".",
  workers: 1,
  timeout: 120_000,
  reporter: process.env.CI ? "list" : "line",
  globalSetup: "./preparar.ts",
  webServer: {
    command: `bash ../../herramientas/servidor-para-e2e.sh ${PUERTO}`,
    url: `http://127.0.0.1:${PUERTO}/v1/salud`,
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
