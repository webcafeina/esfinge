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
import type { Forma } from "./protocolo";

export const VIDA_DEL_PENDIENTE = 2 * 60 * 1000;

export type Pendiente = {
  /** La dirección desde la que se envió, tal como la dio el navegador. */
  origen: string;
  usuario: string;
  secreto: string;
  forma: Forma;
  cuando: number;
};

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
  const pa = a.split(".");
  const pb = b.split(".");
  const cuantas = pa.length >= 2 && pa[pa.length - 2].length <= 3 ? 3 : 2;
  if (pa.length < cuantas || pb.length < cuantas) return false;
  return pa.slice(-cuantas).join(".") === pb.slice(-cuantas).join(".");
}

export function vigente(p: Pendiente, ahora: number): boolean {
  return ahora - p.cuando < VIDA_DEL_PENDIENTE;
}

/** sirvePara dice si un pendiente se puede ofrecer en esa dirección, ahora. */
export function sirvePara(p: Pendiente, url: string, ahora: number): boolean {
  return vigente(p, ahora) && mismoSitio(anfitrion(p.origen), anfitrion(url));
}
