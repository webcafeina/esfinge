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
import { esfinge, type Marco, type Plataforma } from "./puente";

/** Sin marco propio: lo normal, y lo que vale mientras Go no haya contestado. */
const DEL_SISTEMA: Marco = { propia: false, alto: 0, anchoBoton: 0 };

/**
 * Lo mismo que `app.MarcoDe` en Go, **y solo para la simulación por URL**.
 *
 * La regla de verdad vive en Go, que es quien tiene que saberla antes de crear
 * la ventana. Aquí hace falta una copia porque pedir el aspecto de Windows desde
 * una máquina Linux tiene que traer también **su marco**: sin esto, la barra de
 * título no saldría en ninguna captura y es justo lo que hay que mirar.
 *
 * Que las dos no se separen lo vigila `TestElMarcoEsElMismoEnLosDosLados`, que
 * lee este fichero. Es la salida de siempre cuando una regla no se puede juntar
 * porque está escrita en dos idiomas: una prueba que compara las dos listas.
 */
const MARCOS: Record<string, Marco> = {
  darwin: DEL_SISTEMA,
  windows: { propia: true, alto: 48, anchoBoton: 46 },
  linux: DEL_SISTEMA,
};

/** Lo que se sabe de la plataforma antes de preguntar, que es la base neutra. */
const SIN_RESPUESTA: Plataforma = {
  sistema: "",
  escritorio: "",
  vidrio: false,
  marco: DEL_SISTEMA,
};

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
 * GNOME, y con él todo el Linux que no es KDE.
 *
 * **Xfce, Cinnamon y COSMIC entran aquí a propósito** (`escritorio=otro`): son
 * escritorios de GTK y lo que se parece a una aplicación suya es libadwaita, no
 * Kirigami. Lo decidió el cliente al abrir la fase de nativizar.
 */
export const esGnome = () => esLinux() && !esKde();

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
  // **`base` pide la base neutra**, que no es ningún sistema y sí es un aspecto
  // de verdad: es lo que se ve si Go no contesta, y es el que importa el panel
  // de la extensión. Sin esto no se puede mirar desde aquí —`Plataforma()`
  // contesta `linux` en esta máquina—, así que las pruebas que decían medir «la
  // base» llevaban midiendo GNOME, y sólo se vio el día que GNOME dejó de
  // parecerse a ella.
  if (sistema === "base") {
    p.sistema = "";
    p.escritorio = "";
    p.marco = DEL_SISTEMA;
  }
  if (sistema === "darwin" || sistema === "windows" || sistema === "linux") {
    p.sistema = sistema;
    // Y con el sistema viaja su marco: pedir Windows sin su barra de título
    // sería retratar media decisión.
    p.marco = MARCOS[sistema];
  }
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

  // **El marco, con sus medidas puestas como propiedades de CSS.** Las decide Go
  // (ver `app.MarcoDe`) y entran aquí para que la hoja de cada sistema no tenga
  // que repetirlas: escritas en los dos sitios, el día que Windows cambie el alto
  // de su barra habría que acordarse de dos.
  if (p.marco.propia) {
    raiz.dataset.barra = "propia";
    raiz.style.setProperty("--alto-barra-titulo", `${p.marco.alto}px`);
    raiz.style.setProperty("--ancho-boton-ventana", `${p.marco.anchoBoton}px`);
  } else {
    delete raiz.dataset.barra;
    raiz.style.removeProperty("--alto-barra-titulo");
    raiz.style.removeProperty("--ancho-boton-ventana");
  }
}
