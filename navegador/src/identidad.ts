/**
 * Quién está entrando, para no rellenar con la cuenta de otro.
 *
 * # Por qué existe
 *
 * Google, Microsoft y unos cuantos más piden el usuario en una página y la
 * contraseña en la siguiente. En la segunda **no queda ningún campo con el usuario
 * a la vista**, así que hasta la 2.21.0 la regla «una sola cuenta del sitio, se
 * rellena sola» escribía su contraseña aunque se acabara de teclear otro correo en
 * la página anterior. Lo vio el cliente en Google: con `info@` guardada, entró como
 * `alvaro@` y Esfinge le puso la contraseña de `info@` —y luego le ofreció
 * actualizarla con la de `alvaro@`—.
 *
 * Ahora el trabajador de fondo recuerda **lo que una persona teclea** en la página
 * del usuario (ADR 0032), y esta función pura decide con eso. Probada entera en
 * `pruebas/identidad.spec.ts`.
 */
import type { Cuenta } from "./protocolo";

/** mismoUsuario compara como Go: sin espacios alrededor y sin distinguir mayúsculas. */
export function mismoUsuario(a: string, b: string): boolean {
  return a.trim().toLowerCase() === b.trim().toLowerCase();
}

/**
 * cuentaParaRellenarSola dice con qué cuenta se rellena sin que nadie pulse nada, o
 * nulo si con ninguna.
 *
 * Con **una sola** cuenta del sitio, como siempre (ADR 0028). Y ahora **solo si lo
 * escrito no la contradice**: si se sabe qué usuario ha escrito una persona —en el
 * campo de al lado o en la página anterior— y no es el de esa cuenta, no se rellena.
 * Ante la duda, no: rellenar con la contraseña de otra cuenta es enseñarla en un
 * formulario que no es el suyo.
 */
export function cuentaParaRellenarSola(cuentas: Cuenta[], escrito: string): Cuenta | null {
  if (cuentas.length !== 1) return null;
  if (escrito.trim() && !mismoUsuario(cuentas[0].usuario, escrito)) return null;
  return cuentas[0];
}
