/**
 * Lo que se firma cuando un sitio pide una llave de acceso, **igual que
 * `internal/navegador/afirmacion.go`** (ADR 0048).
 *
 * Tres bloques de bytes, y los tres tienen que salir exactos porque **el sitio los
 * verifica byte a byte**:
 *
 *  - `datosDelCliente`: el JSON que describe la petición. Se devuelven **los bytes
 *    que se han hasheado**, no el objeto: el sitio compara lo que le llega con lo
 *    que él pidió, y volver a serializarlo al otro lado daría otra cosa.
 *  - `datosDelAutenticador`: el hash del `rpId`, las banderas y el contador.
 *  - La firma, que es ECDSA P-256 sobre `autenticador ‖ SHA-256(cliente)`.
 *
 * **El origen y el `rpId` los pone quien sabe cuál es**, nunca la página: llegan
 * del navegador (`sender.tab.url`) y pasan por `rpIdPermitido`.
 */

/** Las banderas de `authenticatorData`, en su sitio del byte. */
export const BANDERA_UP = 0x01; // hubo presencia: alguien pulsó
export const BANDERA_UV = 0x04; // hubo verificación: la bóveda estaba abierta
export const BANDERA_BE = 0x08; // la llave se puede respaldar
export const BANDERA_BS = 0x10; // y está respaldada
export const BANDERA_AT = 0x40; // lleva los datos de la credencial (solo al crear)

/**
 * Las banderas de una llave de Esfinge al firmar.
 *
 * `UV` va puesto **aunque el sitio pida `discouraged`**, que es lo que pide GitHub:
 * la bóveda abierta más el clic en el banner *es* verificación de usuario, hay
 * sitios que la exigen, y un sitio que no la pide la acepta igual. Al revés no
 * funciona.
 *
 * `BE` y `BS` porque una llave de Esfinge **está respaldada y sincronizada**: eso
 * es exactamente lo que esos dos bits dicen, y decir otra cosa sería mentirle al
 * sitio sobre algo que puede afectar a cómo trata la cuenta.
 */
export const BANDERAS_AL_FIRMAR = BANDERA_UP | BANDERA_UV | BANDERA_BE | BANDERA_BS;

const utf8 = new TextEncoder();

/** base64url sin relleno, que es como WebAuthn escribe todo lo binario. */
export function aBase64Url(b: Uint8Array): string {
  let s = "";
  for (const x of b) s += String.fromCharCode(x);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function deBase64Url(s: string): Uint8Array {
  const t = s.replace(/-/g, "+").replace(/_/g, "/");
  const crudo = atob(t + "=".repeat((4 - (t.length % 4)) % 4));
  return Uint8Array.from(crudo, (c) => c.charCodeAt(0));
}

/**
 * El `clientDataJSON`, **con las claves en el orden de la especificación**.
 *
 * Se escribe a mano y no con `JSON.stringify` de un objeto por la misma razón por
 * la que la forma canónica de la bóveda se escribe a mano: hay que poder decir
 * exactamente qué bytes salen. Aquí el orden no es alfabético, es el del ejemplo
 * de la especificación, y los sitios que comparan cadenas —los hay— esperan ése.
 */
export function datosDelCliente(
  tipo: "webauthn.get" | "webauthn.create",
  reto: Uint8Array,
  origen: string,
): Uint8Array<ArrayBuffer> {
  const esc = (s: string) => JSON.stringify(s);
  return utf8.encode(
    `{"type":${esc(tipo)},"challenge":${esc(aBase64Url(reto))},"origin":${esc(origen)},"crossOrigin":false}`,
  );
}

/**
 * El `authenticatorData`: `SHA-256(rpId)` ‖ banderas ‖ contador.
 *
 * **El contador va siempre a cero** (ADR 0048): una llave sincronizada entre
 * equipos no puede llevarlo coherente, así que se dice que no se lleva la cuenta,
 * que es lo que hacen todos los gestores y lo que ningún sitio rechaza.
 */
export async function datosDelAutenticador(rpId: string, banderas: number): Promise<Uint8Array<ArrayBuffer>> {
  const hash = new Uint8Array(await crypto.subtle.digest("SHA-256", utf8.encode(rpId)));
  const out = new Uint8Array(37);
  out.set(hash, 0);
  out[32] = banderas & 0xff;
  // Los cuatro del contador se quedan a cero, que es lo que ya son.
  return out;
}

/** Lo que se firma: el autenticador y el hash de los datos del cliente, pegados. */
export async function loQueSeFirma(
  autenticador: Uint8Array,
  cliente: Uint8Array<ArrayBuffer>,
): Promise<Uint8Array<ArrayBuffer>> {
  const hash = new Uint8Array(await crypto.subtle.digest("SHA-256", cliente));
  const out = new Uint8Array(autenticador.length + hash.length);
  out.set(autenticador, 0);
  out.set(hash, autenticador.length);
  return out;
}
