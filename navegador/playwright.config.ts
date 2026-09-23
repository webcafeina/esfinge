import { defineConfig } from "@playwright/test";

/**
 * Las pruebas de la extensión, **en un navegador de verdad**.
 *
 * No hay ningún servidor que levantar: lo que se prueba es la detección de campos
 * contra páginas escritas a mano, y para eso hace falta un DOM auténtico y no uno
 * de mentira. Un DOM simulado devolvería lo que le hubiéramos enseñado a devolver
 * —`getComputedStyle`, `getBoundingClientRect`, `compareDocumentPosition`— y lo
 * que probaría es el simulador. Este proyecto ya se dio ese golpe con los iconos.
 */
export default defineConfig({
  testDir: "./pruebas",
  fullyParallel: true,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "line",
  // Como en `frontend/`: en la máquina de GitHub, un fallo deja traza y captura.
  use: process.env.CI ? { trace: "retain-on-failure", screenshot: "only-on-failure" } : {},
});
