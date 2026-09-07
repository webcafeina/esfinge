import { defineConfig, devices } from "@playwright/test";

/**
 * Estas pruebas mueven la interfaz de verdad contra el Go de verdad: el servidor
 * de desarrollo levanta el mismo código que llevará la ventana, y solo cambia el
 * transporte. Es lo que permite comprobar el camino entero en una máquina sin
 * entorno gráfico.
 *
 * PLAYWRIGHT_CHROMIUM permite apuntar a un navegador ya descargado, que es lo
 * que hay en la máquina de desarrollo; en integración continua se deja vacío y
 * Playwright usa el suyo.
 */
const ejecutable = process.env.PLAYWRIGHT_CHROMIUM;

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  fullyParallel: false,
  workers: 1,
  reporter: process.env.CI ? "list" : [["list"]],
  use: {
    baseURL: "http://127.0.0.1:5173",
    viewport: { width: 820, height: 620 },
    ...(ejecutable ? { launchOptions: { executablePath: ejecutable } } : {}),
  },
  projects: [
    { name: "claro", use: { ...devices["Desktop Chrome"], colorScheme: "light" } },
    { name: "oscuro", use: { ...devices["Desktop Chrome"], colorScheme: "dark" } },
  ],
});
