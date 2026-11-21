// **Con qué aspecto se pinta la ventana: el del sistema, claro u oscuro.**
//
// Los tokens traen desde siempre sus tres bloques —el claro, el oscuro de la consulta
// del sistema y el oscuro forzado por `data-tema`— y hasta hoy **nadie escribía ese
// atributo**, así que elegir no se podía. Lo pidió el cliente el 2026-11-21 para los
// cuatro sistemas.
//
// Va donde va la plataforma y por lo mismo: **antes del primer pintado**. Puesto
// después, quien tenga elegido el oscuro con el sistema en claro vería un fogonazo
// blanco en cada arranque, que es exactamente el reflujo que `plataforma.ts` explica.
import { esfinge } from "./puente";

/**
 * Escribe —o quita— el atributo del que cuelgan los tres bloques de color.
 *
 * «sistema» **quita el atributo** en vez de escribir uno: así manda la consulta
 * `prefers-color-scheme`, que es lo que hace que la ventana siga al sistema cuando
 * éste cambia de aspecto a mediodía sin que nadie vuelva a preguntar nada. Y
 * cualquier cosa que no se entienda cae también ahí, que es el lado seguro: sin
 * atributo la ventana se ve, con uno inventado se vería la base a medias.
 */
export function aplicarTema(t: string | undefined): void {
  const raiz = document.documentElement;
  if (t === "claro" || t === "oscuro") raiz.dataset.tema = t;
  else delete raiz.dataset.tema;
}

/** Lo pregunta y lo aplica. Sin respuesta, el del sistema. */
export async function resolverTema(): Promise<void> {
  try {
    aplicarTema((await esfinge.verPreferencias()).tema);
  } catch {
    aplicarTema(undefined);
  }
}
