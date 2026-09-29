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
