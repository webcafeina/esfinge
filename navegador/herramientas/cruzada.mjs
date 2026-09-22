// Ejecuta el núcleo de la extensión para las pruebas cruzadas de Go (ADR 0040).
//
// Lee una petición JSON por la entrada estándar, la pasa a `ejecutar` de
// `src/nucleo/cruzada.ts` y escribe la respuesta JSON por la salida. El núcleo se
// empaqueta al vuelo con Vite en modo servidor —esbuild no se puede importar
// suelto aquí—, todo en una pieza, así que corre igual en esta máquina que en la
// de GitHub. Un error sale por la salida de errores con código distinto de cero,
// para que Go lo enseñe tal cual.

import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { build } from "vite";

const raiz = fileURLToPath(new URL("..", import.meta.url));
const salida = mkdtempSync(join(tmpdir(), "esfinge-cruzada-"));

try {
  await build({
    configFile: false,
    logLevel: "silent",
    root: raiz,
    ssr: { noExternal: true },
    build: {
      ssr: join(raiz, "src/nucleo/cruzada.ts"),
      outDir: salida,
      emptyOutDir: true,
      target: "node22",
      minify: false,
      rollupOptions: { output: { format: "es", entryFileNames: "cruzada.mjs" } },
    },
  });
  const { ejecutar } = await import(pathToFileURL(join(salida, "cruzada.mjs")).href);
  // Como flujo y no con readFileSync(0): con una tubería que va llegando, leerla
  // de golpe da EAGAIN en cuanto la petición pasa de unos kilobytes.
  const trozos = [];
  for await (const t of process.stdin) trozos.push(t);
  const peticion = JSON.parse(Buffer.concat(trozos).toString("utf8"));
  const respuesta = await ejecutar(peticion);
  process.stdout.write(JSON.stringify(respuesta));
} catch (e) {
  process.stderr.write(String(e?.stack ?? e));
  process.exitCode = 1;
} finally {
  rmSync(salida, { recursive: true, force: true });
}
