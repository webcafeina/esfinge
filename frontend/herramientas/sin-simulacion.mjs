// La simulación de plataforma **no puede acabar en el binario del cliente**.
//
// `plataforma.ts` deja pedir otro sistema o otro escritorio por la URL
// (`?sistema=windows`, `?escritorio=kde`), que es lo que permite probar y
// retratar los cuatro aspectos desde un servidor sin pantalla. En la ventana de
// alguien eso no tiene nada que hacer: sería una forma de que una página —o una
// orden pegada— le cambiara el aspecto a la aplicación.
//
// Para eso está `import.meta.env.DEV`, que Vite sustituye por `false` al
// construir y deja el código muerto fuera del paquete. **Pero una promesa no
// basta**: lo que hay que respetar tiene que parar algo, y es la lección que en
// esta casa ya costó seis versiones. Así que esto lee lo construido de verdad y
// falla si encuentra la marca.
//
// Es el equivalente en la interfaz de la etiqueta `dev` de Go: allí el
// compilador deja fuera `dev.go`, y aquí el empaquetador deja fuera la
// simulación. Corre en `make frontend`, así que pasa por `make app` y por la
// publicación.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";

// La misma cadena que `plataforma.ts` escribe en `data-simulado`. Si allí se
// cambia y aquí no, esto deja de vigilar en silencio — por eso el mensaje de
// abajo dice dónde mirar.
const MARCA = "esfinge:plataforma-simulada";

const raiz = new URL("../dist", import.meta.url).pathname;

function ficheros(dir) {
  return readdirSync(dir).flatMap((n) => {
    const ruta = join(dir, n);
    return statSync(ruta).isDirectory() ? ficheros(ruta) : [ruta];
  });
}

let hay = [];
try {
  hay = ficheros(raiz).filter((f) => /\.(js|css|html)$/.test(f));
} catch {
  console.error(`✕ No encuentro ${raiz}: esto va después de «pnpm run build».`);
  process.exit(1);
}

if (hay.length === 0) {
  console.error(`✕ No hay nada construido en ${raiz}, así que esto no comprueba nada.`);
  process.exit(1);
}

const culpables = hay.filter((f) => readFileSync(f, "utf8").includes(MARCA));
if (culpables.length > 0) {
  console.error(
    `✕ La simulación de plataforma ha entrado en lo construido:\n` +
      culpables.map((f) => `    ${f}`).join("\n") +
      `\n  Tiene que quedar dentro de «if (import.meta.env.DEV)» en frontend/src/plataforma.ts,` +
      `\n  que es lo que hace que Vite la deje fuera. En la ventana de alguien, pedir otro` +
      `\n  sistema por la URL sería una forma de cambiarle el aspecto a la aplicación.`,
  );
  process.exit(1);
}

console.log(`✓ ${hay.length} ficheros construidos y ninguno lleva la simulación`);
