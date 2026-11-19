// En qué sistema y en qué escritorio corre la ventana, resuelto **antes del
// primer pintado**.
//
// # Por qué antes de pintar
//
// De estos atributos cuelga todo el CSS de cada escritorio. Hasta la 2.69.0 los
// ponía `App.tsx` en un efecto, o sea **después** del primer pintado: eso es un
// reflujo visible —se ve un instante la base y luego el sistema— y, peor, es la
// carrera que obligó a escribir que «al forzar `data-sistema` desde una prueba
// hay que esperar a que la aplicación fije el suyo, o su promesa lo sobrescribe
// y se acaba juzgando otro creyendo que es éste». Resolviendo antes de montar no
// hay instante sin atributo, así que las pruebas **dejan de forzar: piden**.
//
// # Y por qué el valor se queda aquí
//
// Porque hay sitios que necesitan saberlo mientras dibujan —qué juego de iconos
// toca, qué dice una nota de Ajustes— y preguntarlo otra vez sería otra promesa
// y otro estado nulo que gestionar. Se resuelve una vez, se guarda, y quien lo
// necesite lo lee de corrido.
import { esfinge, type Plataforma } from "./puente";

/** Lo que se sabe de la plataforma antes de preguntar, que es la base neutra. */
const SIN_RESPUESTA: Plataforma = { sistema: "", escritorio: "", vidrio: false };

/**
 * El valor que busca `herramientas/sin-simulacion.mjs` en lo construido.
 *
 * La simulación por URL **no puede acabar en el binario del cliente**, y para
 * eso está `import.meta.env.DEV`; pero una promesa no basta, así que hay una
 * puerta que lee lo que sale de `vite build` y falla si encuentra esta cadena.
 * Es la misma idea que la etiqueta `dev` de Go.
 */
const MARCA_DE_SIMULACION = "esfinge:plataforma-simulada";

let actual: Plataforma = SIN_RESPUESTA;

/** La plataforma de esta ventana. Vale desde antes del primer pintado. */
export function laPlataforma(): Plataforma {
  return actual;
}

export const esMac = () => actual.sistema === "darwin";
export const esWindows = () => actual.sistema === "windows";
export const esLinux = () => actual.sistema === "linux";

/** KDE es un caso de Linux, no un sistema aparte: por eso se preguntan los dos. */
export const esKde = () => esLinux() && actual.escritorio === "kde";

/**
 * Pregunta a Go dónde está y lo deja escrito en la raíz del documento.
 *
 * Si no contesta —no debería, pero el puente puede fallar— se pinta con la base
 * neutra, que es lo que ven el panel de la extensión y los escritorios que no
 * son GNOME ni KDE. Mejor eso que no pintar.
 */
export async function resolverPlataforma(): Promise<void> {
  let p = SIN_RESPUESTA;
  try {
    p = await esfinge.plataforma();
  } catch {
    // Sin respuesta, la base.
  }
  if (import.meta.env.DEV) p = conLaURL(p);
  actual = p;
  ponerEnLaRaiz(p);
}

/**
 * En el navegador de desarrollo, el aspecto de otro sistema se puede pedir en la
 * URL: `?sistema=windows`, `?escritorio=kde`, `?vidrio=si`.
 *
 * **Parte de lo que ha contestado Go y solo lo desvía**, así que sin parámetros
 * se sigue viendo lo de esta máquina —Linux, y el escritorio que diga el
 * entorno—, igual que antes. Eso es lo que sustituye a tener las cuatro
 * máquinas: es lo que permite probar y retratar los cuatro aspectos desde un
 * servidor sin pantalla.
 *
 * En la ventana no existe: ver `MARCA_DE_SIMULACION`.
 */
function conLaURL(real: Plataforma): Plataforma {
  const q = new URLSearchParams(window.location.search);
  const sistema = q.get("sistema");
  const escritorio = q.get("escritorio");
  const vidrio = q.get("vidrio");
  if (!sistema && !escritorio && !vidrio) return real;

  const p = { ...real };
  // Los nombres son los de Go —«darwin», no «macos»— para que lo que se pide en
  // la URL sea exactamente lo que la ventana recibe de verdad. Un alias aquí
  // sería un segundo vocabulario que mantener.
  if (sistema === "darwin" || sistema === "windows" || sistema === "linux") p.sistema = sistema;
  if (escritorio === "gnome" || escritorio === "kde" || escritorio === "otro") p.escritorio = escritorio;
  if (vidrio) p.vidrio = vidrio === "si";
  document.documentElement.dataset.simulado = MARCA_DE_SIMULACION;
  return p;
}

function ponerEnLaRaiz(p: Plataforma): void {
  const raiz = document.documentElement;
  if (p.sistema) raiz.dataset.sistema = p.sistema;
  else delete raiz.dataset.sistema;

  // El escritorio solo existe en Linux, y donde no existe no se escribe: un
  // atributo vacío invita a colgar CSS de algo que no significa nada.
  if (p.escritorio) raiz.dataset.escritorio = p.escritorio;
  else delete raiz.dataset.escritorio;

  if (p.vidrio) raiz.dataset.vidrio = "si";
  else delete raiz.dataset.vidrio;
}
