#!/usr/bin/env node
/**
 * Sube una versión de la extensión a la Chrome Web Store y la manda a publicar (ADR 0033).
 *
 * **Con la API v2 y una cuenta de servicio**, sin dependencias. La v1 deja de funcionar el
 * 15-10-2026, y con un cliente OAuth «de pruebas» el token de refresco caduca a los siete
 * días, que en una publicación automática es un fallo que llega sin avisar. Una cuenta de
 * servicio firma su propio token cada vez.
 *
 * **La ficha tiene que existir**: la API no crea fichas, y la primera subida se hace a mano
 * (docs/tiendas/pasos.md).
 *
 * Variables: `CWS_CUENTA_DE_SERVICIO` (la clave JSON entera), `CWS_EDITOR` (el ID de editor)
 * y `CWS_EXTENSION` (el identificador de la extensión en la tienda).
 *
 * Uso: node herramientas/tienda-chrome.mjs esfinge-extension-2.22.0-chrome-tienda.zip
 *
 * Las direcciones y los campos salen de la documentación de la API v2
 * (developer.chrome.com/docs/webstore/using-api y service-accounts). **Sin probar contra
 * la tienda de verdad**: la primera publicación con los secretos puestos es la prueba.
 */
import { createSign } from "node:crypto";
import { readFileSync } from "node:fs";

const zip = process.argv[2];
if (!zip) falla("Falta el zip de la extensión");
const cuenta = JSON.parse(obligatoria("CWS_CUENTA_DE_SERVICIO"));
const editor = obligatoria("CWS_EDITOR");
const extension = obligatoria("CWS_EXTENSION");

const API = "https://chromewebstore.googleapis.com";
const elemento = `publishers/${editor}/items/${extension}`;

function obligatoria(nombre) {
  const valor = process.env[nombre];
  if (!valor) falla(`Falta la variable ${nombre}`);
  return valor;
}

function falla(frase) {
  console.error(`::error::${frase}`);
  process.exit(1);
}

/** token firma un JWT con la clave de la cuenta de servicio y lo cambia por un token de acceso. */
async function token() {
  const ahora = Math.floor(Date.now() / 1000);
  const destino = cuenta.token_uri ?? "https://oauth2.googleapis.com/token";
  const trozo = (o) => Buffer.from(JSON.stringify(o)).toString("base64url");
  const sinFirma = `${trozo({ alg: "RS256", typ: "JWT" })}.${trozo({
    iss: cuenta.client_email,
    scope: "https://www.googleapis.com/auth/chromewebstore",
    aud: destino,
    iat: ahora,
    exp: ahora + 3600,
  })}`;
  const firma = createSign("RSA-SHA256").update(sinFirma).sign(cuenta.private_key, "base64url");
  const r = await fetch(destino, {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "urn:ietf:params:oauth:grant-type:jwt-bearer",
      assertion: `${sinFirma}.${firma}`,
    }),
  });
  const datos = await r.json();
  if (!r.ok || !datos.access_token) falla(`Google no ha dado un token: ${JSON.stringify(datos)}`);
  return datos.access_token;
}

/** llamar hace una petición a la API y **enseña la respuesta tal cual**: es lo que dice por qué falla. */
async function llamar(acceso, metodo, ruta, cuerpo, tipo = "application/json") {
  const r = await fetch(`${API}/${ruta}`, {
    method: metodo,
    headers: { authorization: `Bearer ${acceso}`, "content-type": tipo },
    body: cuerpo,
  });
  const texto = await r.text();
  console.log(`${metodo} ${ruta} → ${r.status}\n${texto}`);
  if (!r.ok) falla(`La tienda ha contestado ${r.status} a ${ruta}`);
  try {
    return JSON.parse(texto);
  } catch {
    return {};
  }
}

const acceso = await token();

const subida = await llamar(acceso, "POST", `upload/v2/${elemento}:upload`, readFileSync(zip), "application/zip");
let estado = subida.uploadState;
// Si la tienda sigue procesando el zip, se pregunta hasta que termine.
for (let i = 0; estado === "IN_PROGRESS" && i < 30; i++) {
  await new Promise((r) => setTimeout(r, 10_000));
  const s = await llamar(acceso, "GET", `v2/${elemento}:fetchStatus`);
  estado = s.lastAsyncUploadState ?? s.uploadState ?? estado;
}
if (estado && estado !== "SUCCEEDED") falla(`La subida ha quedado en ${estado}`);

await llamar(acceso, "POST", `v2/${elemento}:publish`, JSON.stringify({ publishType: "DEFAULT_PUBLISH" }));
console.log("Enviada a revisión: se publica sola cuando la tienda la apruebe.");
