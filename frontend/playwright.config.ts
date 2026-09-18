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

  // Las pruebas levantan lo que necesitan: el Go de verdad y Vite. Así «pnpm
  // e2e» funciona de una sola orden, aquí y en integración continua, sin que
  // nadie tenga que acordarse de arrancar dos servidores en el orden correcto.
  webServer: [
    {
      command: "node e2e/api-falsa.mjs",
      url: "http://127.0.0.1:34444/salud",
      reuseExistingServer: !process.env.CI,
      timeout: 30_000,
    },
    {
      // La versión tiene que poder compararse para que el aviso de actualización
      // sea comprobable: con «dev» el propio Go se calla a propósito, que es lo
      // que se quiere en una compilación de trabajo.
      command:
        `go run -tags dev ../cmd/dev` +
        ` -version ${process.env.VERSION_CAPTURAS ?? "2.0.3"}` +
        ` -api http://127.0.0.1:34444`,
      url: "http://127.0.0.1:34443/api/salud",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: { PATH: `${process.env.HOME}/.local/go/bin:${process.env.PATH}` },
    },
    {
      command: "vite --port 5173 --host 127.0.0.1",
      url: "http://127.0.0.1:5173",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },

    // **Y dos equipos con cuenta** (e2e/cuentas.spec.ts): el servidor de cuentas
    // de verdad en local, y dos ventanas, cada una con su Go y su carpeta de
    // configuración recién hecha. Es la tubería entera de la cuenta, de la ventana
    // de un equipo a la de otro, que es la prueba que faltó con los iconos.
    {
      command: "bash ../herramientas/servidor-para-e2e.sh 8792",
      url: "http://127.0.0.1:8792/v1/salud",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
    ...[
      { go: 34453, vite: 5174 },
      { go: 34463, vite: 5175 },
    ].flatMap(({ go, vite }) => [
      {
        command:
          `go run -tags dev ../cmd/dev -direccion 127.0.0.1:${go}` +
          ` -version 2.23.0 -cuentas http://127.0.0.1:8792`,
        url: `http://127.0.0.1:${go}/api/salud`,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
        env: { PATH: `${process.env.HOME}/.local/go/bin:${process.env.PATH}` },
      },
      {
        command: `vite --port ${vite} --host 127.0.0.1`,
        url: `http://127.0.0.1:${vite}`,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
        env: { ESFINGE_GO: `http://127.0.0.1:${go}` },
      },
    ]),
  ],
});
