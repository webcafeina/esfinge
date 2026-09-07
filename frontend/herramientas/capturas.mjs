// Saca capturas de la interfaz recorriendo lo que haría una persona.
//
// Sirve para mirar el resultado antes de compilar nada: en esta máquina no hay
// entorno gráfico, así que esto es lo más cerca que se puede estar de ver la
// ventana. Usa el mismo servidor de desarrollo que las pruebas.
import { chromium } from "@playwright/test";
import { mkdirSync } from "node:fs";

const salida = process.argv[2] ?? "capturas";
const modo = process.argv[3] === "oscuro" ? "dark" : "light";
const sufijo = modo === "dark" ? "oscuro" : "claro";
mkdirSync(salida, { recursive: true });

const navegador = await chromium.launch(
  process.env.PLAYWRIGHT_CHROMIUM ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM } : {},
);
const contexto = await navegador.newContext({
  viewport: { width: 840, height: 640 },
  colorScheme: modo,
  deviceScaleFactor: 2,
});
const pagina = await contexto.newPage();
const foto = (n) => pagina.screenshot({ path: `${salida}/${n}-${sufijo}.png` });

await pagina.goto("http://127.0.0.1:5173/", { waitUntil: "networkidle" });

await pagina.getByLabel("Qué quieres cifrar").fill("postgres://usuario:secreto@host/basededatos");
await pagina.locator("#clave").fill("hunter2");
await pagina.waitForTimeout(500);
await foto("1-cifrar");

await pagina.locator("#clave").fill("caballo grapa batería correcto");
await pagina.waitForTimeout(400);
await pagina.getByRole("button", { name: "Cifrar", exact: true }).click();
await pagina.waitForSelector(".resultado", { timeout: 20000 });
await pagina.waitForTimeout(400);
await foto("2-resultado");

await pagina.getByRole("tab", { name: "Ficheros" }).click();
await pagina.locator(".soltar").click();
await pagina.waitForTimeout(600);
await foto("3-ficheros");

await pagina.getByRole("tab", { name: "Generar" }).click();
await pagina.waitForSelector(".resultado", { timeout: 20000 });
await pagina.waitForTimeout(400);
await foto("4-generar");

await pagina.getByRole("tab", { name: "Historial" }).click();
await pagina.waitForTimeout(500);
await foto("5-historial");

console.log(`Capturas en ${salida} · tema ${sufijo}`);
await navegador.close();
