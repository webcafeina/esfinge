/**
 * Comprueba que el manifiesto declara **todo lo que el código usa**.
 *
 * Existe por un fallo que costó cinco versiones publicadas y una tarde entera de
 * ida y vuelta con el cliente: el código llamaba a `api.storage.local` y el
 * manifiesto no declaraba el permiso `storage`, así que `api.storage` era
 * `undefined`. La excepción saltaba en la primera línea del trabajador de fondo,
 * nadie la recogía, y desde fuera se veía como si el puente con Esfinge no
 * contestara. Se persiguió el puente, el socket, los manifiestos de native
 * messaging, el orden de bytes y hasta el modelo de mensajes entre navegadores.
 * Era una palabra que faltaba en una lista.
 *
 * **Es la misma clase de lista que ya se vigila en Go** —la de lo que cruza el
 * puente con la ventana, la de lo que se le puede pedir al canal— y por la misma
 * razón: una lista que hay que acordarse de actualizar se queda atrás, y aquí
 * quedarse atrás no da un error, da silencio.
 */
import { existsSync, readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

/**
 * Las APIs que **no se pueden usar sin pedir permiso**, con el nombre del
 * permiso que hace falta. Las que no están aquí —`runtime`, `i18n`, `extension`—
 * están siempre disponibles.
 */
const PIDEN_PERMISO = {
  storage: "storage",
  alarms: "alarms",
  cookies: "cookies",
  scripting: "scripting",
  notifications: "notifications",
  bookmarks: "bookmarks",
  history: "history",
  downloads: "downloads",
  webRequest: "webRequest",
  contextMenus: "contextMenus",
  idle: "idle",
  clipboard: "clipboardWrite",
};

/** Lo que además exige `nativeMessaging`, que no se llama igual que su API. */
const APARTE = { connectNative: "nativeMessaging", sendNativeMessage: "nativeMessaging" };

const raiz = new URL("..", import.meta.url).pathname;
const fuentes = readdirSync(join(raiz, "src"))
  .filter((f) => f.endsWith(".ts"))
  .map((f) => readFileSync(join(raiz, "src", f), "utf8"))
  .join("\n");

// Se mira **el código, no los comentarios**: un ejemplo dentro de un comentario
// no usa nada, y hacerlo fallar por eso enseña a desactivar la comprobación.
const codigo = fuentes.replace(/\/\*[\s\S]*?\*\//g, "").replace(/^\s*\/\/.*$/gm, "");

const usadas = new Set();
for (const [, api] of codigo.matchAll(/\bapi\.(\w+)\./g)) usadas.add(api);
for (const [, fn] of codigo.matchAll(/\bapi\.runtime\.(\w+)/g)) {
  if (APARTE[fn]) usadas.add(fn);
}

let mal = 0;
for (const navegador of ["chrome", "firefox"]) {
  const manifiesto = JSON.parse(
    readFileSync(join(raiz, `manifiesto.${navegador}.json`), "utf8"),
  );
  const declarados = new Set(manifiesto.permissions ?? []);

  for (const api of usadas) {
    const permiso = PIDEN_PERMISO[api] ?? APARTE[api];
    if (!permiso) continue;
    if (!declarados.has(permiso)) {
      console.error(
        `manifiesto.${navegador}.json: el código usa «${api}» y falta el permiso «${permiso}».\n` +
          `  Sin él, esa API es undefined y el fallo salta en tiempo de ejecución, no al compilar.`,
      );
      mal++;
    }
  }

  // **Y la dirección de la pestaña, que es la misma trampa con otra cara.**
  // `sender.tab.url` no lanza cuando falta el permiso de anfitrión: llega
  // `undefined`, el origen se va vacío, y desde fuera se ve como que Esfinge dice
  // que ahí no rellena. Es exactamente el fallo mudo de `storage` otra vez, y por
  // eso se comprueba aquí en vez de confiar en acordarse.
  if (/sender\?\.tab\?\.url|sender\.tab\.url/.test(codigo)) {
    const anfitriones = manifiesto.host_permissions ?? [];
    if (anfitriones.length === 0 && !declarados.has("tabs")) {
      console.error(
        `manifiesto.${navegador}.json: el código lee «sender.tab.url» y no hay ` +
          `«host_permissions» ni el permiso «tabs».\n` +
          `  Sin eso llega undefined, sin error, y el origen viaja vacío.`,
      );
      mal++;
    }
  }

  // Los ficheros que el manifiesto nombra tienen que salir de algún fuente. Un
  // guion de contenido que no existe **no da error**: el navegador carga la
  // extensión igual y en las páginas no hay nada.
  for (const guion of manifiesto.content_scripts ?? []) {
    for (const js of guion.js ?? []) {
      const fuente = js.replace(/\.js$/, ".ts");
      if (!existsSync(join(raiz, "src", fuente))) {
        console.error(
          `manifiesto.${navegador}.json: declara el guion «${js}» y no hay «src/${fuente}».`,
        );
        mal++;
      }
    }
    // Y sus «matches» tienen que estar cubiertos por los permisos de anfitrión, o
    // el guion se inyecta y luego no puede hablar con nadie.
    for (const donde of guion.matches ?? []) {
      if (!(manifiesto.host_permissions ?? []).includes(donde)) {
        console.error(
          `manifiesto.${navegador}.json: el guion se pone en «${donde}» y eso no ` +
            `está en «host_permissions».`,
        );
        mal++;
      }
    }
  }
}

if (mal > 0) process.exit(1);
console.log("Permisos: el manifiesto declara todo lo que el código usa.");
