/**
 * Las llaves de acceso: qué se puede firmar y para quién, **igual que
 * `internal/navegador/llaves.go`** (ADR 0048).
 *
 * Aquí vive `rpIdPermitido`, que es **la pieza de seguridad de toda la fase**: es
 * la única vía por la que esto puede *entregar* algo a un atacante. Firmar con el
 * `rpId` equivocado no es un error que se vea; es una identificación válida en
 * otro sitio.
 *
 * **Y no vale `encaja()` de `dominios.ts`**, por mucho que se le parezca. Aquélla
 * compara dominio registrable contra dominio registrable, y por eso
 * `accounts.google.com` y `mail.google.com` son «el mismo sitio»: es lo que se
 * quiere para ofrecer una contraseña. WebAuthn pide otra cosa y más estrecha —el
 * `rpId` tiene que ser **el anfitrión o un sufijo suyo separado por punto**— y
 * además **el hash se calcula sobre la cadena exacta**, así que dar por buenos dos
 * nombres distintos no es ser tolerante: es firmar algo que el sitio va a
 * rechazar, o peor, firmar para quien no es.
 */

import { dominioRegistrable, hostDe } from "./dominios";
import type { Entrada } from "./entrada";

/**
 * El `rpId` con el que se puede firmar en ese origen, o `null`.
 *
 * Devuelve **la cadena exacta que hay que hashear**, no un dominio registrable:
 * el `rpIdHash` de `authenticatorData` se calcula sobre ella y el sitio lo
 * compara byte a byte.
 *
 * Las reglas, que son las de la especificación:
 *
 *  - Solo `https`. Lo demás no es contexto seguro y no se firma.
 *  - Sin `rpId`, manda **el anfitrión entero**, no su dominio registrable: en
 *    `login.ejemplo.com` el `rpId` por defecto es `login.ejemplo.com`.
 *  - Con `rpId`, tiene que ser igual al anfitrión o un sufijo suyo separado por
 *    punto — y **no puede ser un sufijo público**: `login.github.io` no puede
 *    firmar por `github.io`, porque entonces cualquier página alojada ahí firmaría
 *    por todas las demás.
 */
export function rpIdPermitido(rpId: string | undefined, origen: string): string | null {
  let u: URL;
  try {
    u = new URL(origen.trim());
  } catch {
    return null;
  }
  if (u.protocol.toLowerCase() !== "https:") return null;

  const anfitrion = hostDe(origen);
  // `hostDe` ya deja fuera lo que no es ASCII; falta la IP, que tiene anfitrión y
  // no tiene dominio, y sobre la que no se firma.
  if (!anfitrion || dominioRegistrable(anfitrion) === null) return null;

  const pedido = (rpId ?? "").trim().replace(/\.$/, "").toLowerCase();
  if (pedido === "") return anfitrion;
  if (pedido === anfitrion) return pedido;
  if (!anfitrion.endsWith("." + pedido)) return null;
  // **Que tenga algo por debajo que registrar.** Es lo que separa «el dominio de
  // arriba» de «un sufijo público»: `ejemplo.com` sí, `com` y `github.io` no.
  if (dominioRegistrable(pedido) === null) return null;
  return pedido;
}

/**
 * El origen **como lo escribe el navegador**, que es lo que el sitio compara.
 *
 * Es `new URL(x).origin` y no pegar esquema y anfitrión: con el puerto por defecto
 * escrito a mano —`https://github.com:443/`— pegarlo daría `https://github.com:443`
 * y el sitio dice `https://github.com`. Cuatro caracteres de diferencia dentro del
 * `clientDataJSON` y la firma se rechaza sin decir por qué. Go tiene su pareja,
 * `OrigenDe`, y una prueba cruzada de tabla.
 */
export function origenDe(origen: string): string {
  try {
    const u = new URL(origen.trim());
    return u.origin === "null" ? "" : u.origin;
  } catch {
    return "";
  }
}

/**
 * La firma en formato DER, que es lo que WebAuthn exige.
 *
 * WebCrypto firma en **P1363** —`r ‖ s` crudos, 64 bytes en P-256— y el sitio
 * espera **ASN.1 DER**. La conversión es de treinta líneas y es donde se equivoca
 * todo el mundo, por dos motivos que el formato no perdona: un `INTEGER` de DER va
 * **sin ceros por delante**, y si el primer bit está a uno hay que **añadir un cero**
 * para que no se lea como negativo. Una firma con un cero de más o de menos no se
 * verifica, y el sitio no dice por qué.
 */
function enteroDER(b: Uint8Array): number[] {
  let i = 0;
  while (i < b.length - 1 && b[i] === 0) i++;
  const v = Array.from(b.slice(i));
  if ((v[0] & 0x80) !== 0) v.unshift(0);
  return [0x02, v.length, ...v];
}

export function aDER(p1363: Uint8Array): Uint8Array {
  const n = p1363.length / 2;
  const cuerpo = [...enteroDER(p1363.slice(0, n)), ...enteroDER(p1363.slice(n))];
  // En P-256 el cuerpo mide 70 bytes como mucho, así que la longitud cabe en un
  // byte y no hace falta la forma larga de DER. Con otra curva habría que mirarlo.
  if (cuerpo.length > 127) throw new Error("La firma no cabe en la forma corta de DER");
  return new Uint8Array([0x30, cuerpo.length, ...cuerpo]);
}

/**
 * Una llave de acceso nueva: la privada en **PKCS#8**, la pública en SPKI.
 *
 * **La privada se guarda entera y no solo el escalar**, y eso corrige lo que decía
 * la ADR 0048 al escribirla: se dio por hecho que la parte pública «se recalcula»,
 * y **WebCrypto no puede** — importar una privada P-256 exige `x` e `y`, y no hay
 * forma de multiplicar un escalar por el generador desde ahí. Se comprobó
 * intentándolo. PKCS#8 lleva las dos partes dentro, lo entienden los dos lados sin
 * escribir nada, y son 138 bytes.
 */
export async function crearLlave(): Promise<{ privada: Uint8Array; publica: Uint8Array }> {
  const par = await crypto.subtle.generateKey({ name: "ECDSA", namedCurve: "P-256" }, true, ["sign", "verify"]);
  return {
    privada: new Uint8Array(await crypto.subtle.exportKey("pkcs8", par.privateKey)),
    publica: new Uint8Array(await crypto.subtle.exportKey("spki", par.publicKey)),
  };
}

/** Firma con una llave guardada, y devuelve la firma **en DER**. */
export async function firmarConLlave(privadaPKCS8: Uint8Array, datos: Uint8Array): Promise<Uint8Array> {
  const k = await crypto.subtle.importKey(
    "pkcs8",
    new Uint8Array(privadaPKCS8),
    { name: "ECDSA", namedCurve: "P-256" },
    false,
    ["sign"],
  );
  const cruda = new Uint8Array(await crypto.subtle.sign({ name: "ECDSA", hash: "SHA-256" }, k, new Uint8Array(datos)));
  return aDER(cruda);
}


/**
 * Las llaves de acceso de un sitio, **ya filtradas por lo que el sitio acepta**.
 *
 * `permitidas` son los identificadores de credencial que el sitio ha mandado en
 * `allowCredentials`. Cuando viene con algo —y GitHub lo manda—, **no es una
 * credencial descubrible**: el sitio dice exactamente qué llave quiere, y ofrecer
 * otra sería ofrecer algo que va a rechazar. Vacío significa «la que tengas».
 */
export function llavesDe(entradas: Entrada[], rpId: string, permitidas?: string[]): Entrada[] {
  const quiere = new Set(permitidas ?? []);
  return entradas.filter(
    (x) =>
      x.tipo === "llave" &&
      !x.papelera &&
      x.rpId === rpId &&
      Boolean(x.clavePrivada) &&
      (quiere.size === 0 || (x.idCredencial !== undefined && quiere.has(x.idCredencial))),
  );
}
