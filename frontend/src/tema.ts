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
  marcarSiElMaterialNoCuadra();
}

/**
 * **Dice si el material del sistema se ha quedado del otro tema.**
 *
 * Bajo vidrio, la barra lateral **no lleva fondo propio**: el material del sistema
 * *es* el fondo (ADR 0017). Y ese material lo elige macOS por la apariencia de la
 * ventana, que sigue a la del equipo y **no sabe nada de este atributo**: Wails sólo
 * deja elegirla al arrancar (`options.Mac.Appearance`), así que al forzar un tema
 * distinto del que tiene el Mac, todo se pinta oscuro y **la barra lateral se queda
 * clara**. Lo vio el cliente con la 2.76.0.
 *
 * Cuando eso pasa se marca aquí, y la hoja de macOS **le devuelve su fondo** a la
 * barra: deja de ser vidrio, que es lo honesto —el material de debajo es el que no
 * toca— y se ve de una pieza. Cuando el tema elegido coincide con el del equipo, el
 * vidrio se queda como siempre.
 *
 * `matchMedia` contesta por **el equipo**, no por lo que digamos nosotros, que es
 * justo lo que hace falta para poder compararlos.
 */
function marcarSiElMaterialNoCuadra(): void {
  const raiz = document.documentElement;
  const elegido = raiz.dataset.tema;
  if (!elegido) {
    delete raiz.dataset.material;
    return;
  }
  const delEquipo = window.matchMedia?.("(prefers-color-scheme: dark)").matches ? "oscuro" : "claro";
  if (elegido === delEquipo) delete raiz.dataset.material;
  else raiz.dataset.material = "ajeno";
}

/**
 * Lo pregunta y lo aplica. Sin respuesta, el del sistema.
 *
 * **Y con un plazo, porque esto bloquea el primer pintado.** La plataforma tiene que
 * llegar antes de pintar —de `data-sistema` cuelga todo el CSS— y el tema se pide a
 * la vez para no encadenar dos viajes; el precio es que ahora son **dos** respuestas
 * las que hacen falta para que haya ventana. Si una se queda colgada, lo que ve
 * alguien no es un tema equivocado: es **una ventana que no aparece**, que en un
 * gestor de contraseñas es el peor final posible.
 *
 * Con el plazo, lo peor que puede pasar es un fogonazo del tema del sistema en quien
 * haya elegido otro, y la respuesta que llegue tarde se aplica igual: `aplicarTema`
 * no sabe ni le importa si alguien estaba esperándola.
 */
export async function resolverTema(): Promise<void> {
  const plazo = new Promise<void>((listo) => setTimeout(listo, 1200));
  const preguntar = esfinge
    .verPreferencias()
    .then((p) => aplicarTema(p.tema))
    .catch(() => aplicarTema(undefined));
  await Promise.race([preguntar, plazo]);
}
