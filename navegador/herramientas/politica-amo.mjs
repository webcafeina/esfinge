#!/usr/bin/env node
// Pone en la ficha de addons.mozilla.org la política de privacidad que hay en la
// web, sin que nadie la pegue a mano.
//
// **Por qué existe.** La política vive en `web/privacidad.html` y su copia en texto
// está en `docs/tiendas/privacidad-amo.txt` (ADR 0033: las tres tienen que decir lo
// mismo). Hasta ahora, cuando cambiaba, había que entrar en AMO y pegarla; y un
// texto que hay que acordarse de pegar es un texto que un día no se pega.
//
// AMO **no** deja editarla con el resto de los metadatos: tiene su propio extremo,
// `PATCH /api/v5/addons/addon/<id>/eula_policy/`, con `privacy_policy` como campo
// traducible. Eso es todo lo que hace este guion.
//
// La ficha de Chrome no necesita nada de esto: allí la política es **una dirección**
// y no un texto, y esa dirección no cambia.
//
// Sin las claves de AMO no hace nada y no falla: igual que el resto de la
// publicación, que se salta las tiendas cuando no hay secretos.

import { createHmac, randomUUID } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const AQUI = dirname(fileURLToPath(import.meta.url));
const RAIZ = resolve(AQUI, "../..");
const IDIOMA = "es-ES"; // el `default_locale` de docs/tiendas/amo-metadata.json
const EXTENSION = "esfinge@webcafeina.com"; // el id de `browser_specific_settings`

const clave = process.env.WEB_EXT_API_KEY;
const secreto = process.env.WEB_EXT_API_SECRET;
if (!clave || !secreto) {
  console.log("::notice::Sin las claves de addons.mozilla.org: la política de la ficha no se toca.");
  process.exit(0);
}

/** El texto tal cual se pega: el fichero sin sus líneas de comentario. */
function politica() {
  const crudo = readFileSync(resolve(RAIZ, "docs/tiendas/privacidad-amo.txt"), "utf8");
  const texto = crudo
    .split("\n")
    .filter((l) => !l.startsWith("#"))
    .join("\n")
    .trim();
  if (texto.length < 500) throw new Error(`La política parece vacía o cortada (${texto.length} caracteres)`);
  return texto;
}

/** El testigo que pide AMO: un JWT de cinco minutos firmado con el secreto. */
function testigo() {
  const b64 = (o) => Buffer.from(JSON.stringify(o)).toString("base64url");
  const ahora = Math.floor(Date.now() / 1000);
  const cuerpo = b64({ iss: clave, jti: randomUUID(), iat: ahora, exp: ahora + 300 });
  const cabecera = b64({ alg: "HS256", typ: "JWT" });
  const firma = createHmac("sha256", secreto).update(`${cabecera}.${cuerpo}`).digest("base64url");
  return `${cabecera}.${cuerpo}.${firma}`;
}

const texto = politica();
const r = await fetch(`https://addons.mozilla.org/api/v5/addons/addon/${EXTENSION}/eula_policy/`, {
  method: "PATCH",
  headers: { Authorization: `JWT ${testigo()}`, "Content-Type": "application/json" },
  body: JSON.stringify({ privacy_policy: { [IDIOMA]: texto } }),
});

if (!r.ok) {
  console.error(`AMO ha contestado ${r.status}: ${(await r.text()).slice(0, 400)}`);
  process.exit(1);
}
console.log(`Política de privacidad puesta en la ficha de Firefox (${texto.length} caracteres).`);
