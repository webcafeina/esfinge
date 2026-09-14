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
 * Ahora el trabajador de fondo recuerda **el usuario que había** en la página del
 * usuario al seguir —lo tecleara una persona, lo pusiera el navegador, el sitio o
 * Esfinge— (ADR 0032), y esta función pura decide con eso. Probada entera en
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
 * **Si se sabe qué usuario va a entrar** —el del campo de al lado, el escondido que
 * declara el sitio o el de la página anterior—, la cuenta con ese usuario, **solo si
 * hay exactamente una**. Con varias cuentas del sitio es lo que eligió el cliente con
 * la 2.21.1 («la que coincida con el correo»), y con una sola evita rellenar la
 * contraseña de otra cuenta, que era el fallo de Google.
 *
 * **Si no se sabe**, como siempre (ADR 0028): con una sola cuenta, ésa; con varias,
 * ninguna, y se elige en el panel.
 */
export function cuentaParaRellenarSola(cuentas: Cuenta[], escrito: string): Cuenta | null {
  if (escrito.trim()) {
    const suyas = cuentas.filter((c) => mismoUsuario(c.usuario, escrito));
    return suyas.length === 1 ? suyas[0] : null;
  }
  return cuentas.length === 1 ? cuentas[0] : null;
}
