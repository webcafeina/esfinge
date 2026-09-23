/**
 * La contraseña que se acaba de enviar, esperando a que alguien decida si se guarda.
 *
 * # Por qué existe, y por qué así
 *
 * Al pulsar «Entrar» la página cambia y el guion que leyó el formulario muere con
 * ella. La tarjeta de «¿Guardar en Esfinge?» sale en la página siguiente, así que lo
 * enviado tiene que esperar en algún sitio mientras tanto. Ese sitio es **la memoria
 * del trabajador de fondo**, y en ningún otro: nunca en `storage`, nunca en la
 * página siguiente.
 *
 * Y con límites, que son lo que decide esta función pura (probada entera en
 * `pruebas/pendientes.spec.ts`):
 *
 *   - **Caduca a los dos minutos.** Una oferta que llega tarde no se enseña.
 *   - **Solo sirve para el mismo sitio.** Si la pestaña se va a otro, se olvida.
 */
import { dominioRegistrable } from "./nucleo/dominios";

import type { Forma } from "./protocolo";

export const VIDA_DEL_PENDIENTE = 2 * 60 * 1000;

/**
 * Lo que dura el usuario tecleado en la página de solo usuario (ver
 * `identidad.ts`). Más que el pendiente, porque entre las dos páginas puede haber un
 * captcha o una espera; y no es una contraseña.
 */
export const VIDA_DEL_USUARIO = 5 * 60 * 1000;

/** Algo que se recuerda de una pestaña: de dónde vino y cuándo. */
export type Recuerdo = {
  /** La dirección en la que pasó, tal como la dio el navegador. */
  origen: string;
  cuando: number;
};

export type Pendiente = Recuerdo & {
  usuario: string;
  secreto: string;
  forma: Forma;
};

/** El usuario que una persona tecleó en la página de solo usuario. */
export type UsuarioEscrito = Recuerdo & { usuario: string };

function anfitrion(url: string): string {
  try {
    const u = new URL(url);
    return u.protocol === "https:" ? u.hostname.toLowerCase() : "";
  } catch {
    return "";
  }
}

/**
 * mismoSitio dice si dos anfitriones son del mismo sitio, **sin la lista de sufijos
 * públicos**, que aquí no está: iguales, o con las dos últimas etiquetas iguales
 * —`accounts.google.com` y `myaccount.google.com`—, salvo que la penúltima sea corta,
 * como la de `co.uk` o `com.es`, que ahí harían falta tres.
 *
 * **Que se equivoque no entrega nada a nadie**: esto solo decide si se enseña la
 * tarjeta. Lo que se guarda va siempre para el sitio del que salió, y eso lo
 * comprueba Esfinge con la lista de verdad.
 */
export function mismoSitio(a: string, b: string): boolean {
  if (!a || !b) return false;
  if (a === b) return true;
  // **Con la lista de sufijos públicos**, que desde la 2.25.0 ya va dentro de la
  // extensión (revisión del 2026-09-23). Contando etiquetas a ojo,
  // `evil.github.io` y `victima.github.io` eran «el mismo sitio»: no se entrega
  // ningún secreto por ahí —lo que se guarda va al origen del envío—, pero el
  // usuario tecleado en uno salía propuesto en la tarjeta del otro.
  const da = dominioRegistrable(a);
  const db = dominioRegistrable(b);
  return da !== null && da === db;
}

export function vigente(p: Recuerdo, ahora: number, vida = VIDA_DEL_PENDIENTE): boolean {
  return ahora - p.cuando < vida;
}

/** sirvePara dice si lo recordado vale en esa dirección, ahora. */
export function sirvePara(p: Recuerdo, url: string, ahora: number, vida = VIDA_DEL_PENDIENTE): boolean {
  return vigente(p, ahora, vida) && mismoSitio(anfitrion(p.origen), anfitrion(url));
}
