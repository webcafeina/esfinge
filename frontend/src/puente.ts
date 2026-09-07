// El puente entre la interfaz y Go.
//
// En la aplicación empaquetada, Wails cuelga los métodos de Go en window.go y la
// llamada no sale del proceso. Durante el desarrollo eso no existe —haría falta
// un entorno gráfico y webkit, que en la máquina donde se escribe esto no hay—,
// así que las mismas llamadas se hacen por HTTP contra el servidor que levanta
// «make dev».
//
// La interfaz no distingue entre los dos casos: llama a estas funciones y ya.
// Eso es lo que permite probar la interfaz entera de verdad, con un navegador,
// sin compilar la aplicación.

export type Resultado = {
  texto: string;
  aviso: string;
};

export type ResultadoFichero = {
  origen: string;
  destino: string;
  error: string;
};

export type Fuerza = {
  nivel: number;
  bits: number;
  etiqueta: string;
  sugerencia: string;
};

export type Alfabeto = {
  nombre: string;
  etiqueta: string;
  seguroURL: boolean;
  aviso: string;
};

export type Entrada = {
  accion: "cifrar" | "descifrar";
  nombre: string;
  destino: string;
  cuando: string;
};

export type Progreso = {
  hechos: number;
  total: number;
  actual: string;
};

export type Medida = {
  bytes: number;
  caracteres: number;
  bits: number;
};

/** Lo que se sabe de una versión más nueva que la instalada. */
export type Novedad = {
  hay: boolean;
  version: string;
  pagina: string;
  /** El fichero que le toca a este sistema. Vacío si no hay ninguno. */
  fichero: string;
  bytes: number;
  /** «sola» si se reemplaza y reinicia; «instalador» si hace falta el del sistema. */
  comoSeInstala: "sola" | "instalador" | "";
};

/** Cómo va la descarga de la actualización. */
export type Avance = {
  bytes: number;
  total: number;
  hecho: boolean;
};

/**
 * Lo que hay que enseñar cuando el sistema manda ficheros.
 *
 * Un .esf puede llevar un fichero cifrado o el contenedor de una línea que sale
 * de cifrar un texto. Lo decide Go mirando dentro, no la extensión.
 */
export type Apertura = {
  modo: "texto" | "ficheros" | "";
  texto: string;
  rutas: string[] | null;
};

/** Lo que pide el menú del sistema. Los valores los fija Go, en ordenes.go. */
export type Orden = {
  que: string;
  /** Solo lo lleva «editar:pegar»: el portapapeles lo lee Go. */
  texto: string;
};

export type Preferencias = {
  buscarActualizaciones: boolean;
  ultimaComprobacion: string;
  versionVista: string;
};

type MetodosGo = Record<string, (...args: unknown[]) => Promise<unknown>>;

declare global {
  interface Window {
    go?: { app?: { App?: MetodosGo } };
    runtime?: {
      EventsOn: (evento: string, cb: (...datos: unknown[]) => void) => void;
    };
  }
}

/** Dice si estamos dentro de la aplicación de verdad o en el navegador. */
export function enWails(): boolean {
  return typeof window.go?.app?.App?.CifrarTexto === "function";
}

/**
 * llamar ejecuta un método de Go por el camino que haya disponible.
 *
 * Los errores de Go llegan aquí como promesa rechazada en Wails y como respuesta
 * con estado 400 por HTTP; los dos acaban siendo un Error con el mismo mensaje,
 * que es lo que la interfaz enseña tal cual. Los mensajes ya vienen escritos
 * para leerse: no hay que adornarlos.
 */
async function llamar<T>(metodo: string, ...args: unknown[]): Promise<T> {
  const go = window.go?.app?.App;
  if (go && typeof go[metodo] === "function") {
    return (await go[metodo](...args)) as T;
  }

  const respuesta = await fetch(`/api/${metodo}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(args),
  });

  const cuerpo = await respuesta.json();
  if (!respuesta.ok) {
    throw new Error(cuerpo?.error ?? "Algo ha fallado y no sé decir qué");
  }
  return cuerpo as T;
}

export const esfinge = {
  cifrarTexto: (texto: string, clave: string) =>
    llamar<Resultado>("CifrarTexto", texto, clave),

  descifrarTexto: (texto: string, clave: string) =>
    llamar<Resultado>("DescifrarTexto", texto, clave),

  cifrarFicheros: (rutas: string[], clave: string) =>
    llamar<ResultadoFichero[]>("CifrarFicheros", rutas, clave),

  descifrarFicheros: (rutas: string[], clave: string) =>
    llamar<ResultadoFichero[]>("DescifrarFicheros", rutas, clave),

  generarContrasena: (bytes: number, alfabeto: string) =>
    llamar<string>("GenerarContrasena", bytes, alfabeto),

  evaluarClave: (clave: string) => llamar<Fuerza>("EvaluarClave", clave),

  alfabetos: () => llamar<Alfabeto[]>("Alfabetos"),

  elegirFicheros: (varios: boolean) => llamar<string[]>("ElegirFicheros", varios),

  elegirCifrados: () => llamar<string[]>("ElegirCifrados"),

  guardarTexto: (nombre: string, contenido: string) =>
    llamar<string>("GuardarTexto", nombre, contenido),

  verHistorial: () => llamar<Entrada[]>("VerHistorial"),

  vaciarHistorial: () => llamar<void>("VaciarHistorial"),

  dondeVive: () => llamar<string>("DondeVive"),

  version: () => llamar<string>("Version"),

  aperturaDeArranque: () => llamar<Apertura>("AperturaDeArranque"),

  medirPorCaracteres: (caracteres: number, alfabeto: string) =>
    llamar<Medida>("MedirPorCaracteres", caracteres, alfabeto),

  novedadPendiente: () => llamar<Novedad>("NovedadPendiente"),

  comprobarActualizacion: () => llamar<Novedad>("ComprobarActualizacion"),

  descargarActualizacion: () => llamar<string>("DescargarActualizacion"),

  instalarActualizacion: () => llamar<void>("InstalarActualizacion"),

  verPreferencias: () => llamar<Preferencias>("VerPreferencias"),

  /** Lo dispara el menú del sistema; aquí está para poder probarlo sin menú. */
  ordenar: (que: string) => llamar<void>("Ordenar", que),

  guardarPreferencias: (p: Preferencias) => llamar<void>("GuardarPreferencias", p),
};

/** Los límites de longitud los pone Go, no la interfaz. */
export const CARACTERES_MINIMO = 16;
export const CARACTERES_MAXIMO = 96;

/**
 * alHaberNovedad escucha el aviso de que hay una versión nueva.
 *
 * Llega tarde a propósito: la comprobación sale a la red en su propia gorrutina
 * para no retrasar la ventana, así que este evento puede caer segundos después
 * de abrir. Quien se monte más tarde puede preguntar por «novedadPendiente».
 */
export function alHaberNovedad(cb: (n: Novedad) => void): () => void {
  return escuchar("novedad", cb);
}

/**
 * alAbrirFichero escucha los ficheros que manda el sistema con la ventana ya
 * abierta: doble clic en un .esf mientras Esfinge corre.
 *
 * Los que llegan **antes** de que la interfaz esté escuchando no vienen por
 * aquí: ésos los guarda Go y se recogen con «ficherosDeArranque». Por eso hay
 * que suscribirse antes de preguntar, y no al revés.
 */
export function alAbrirFichero(cb: (a: Apertura) => void): () => void {
  return escuchar("fichero-abierto", cb);
}

/** alOrdenar escucha lo que se pide desde el menú del sistema. */
export function alOrdenar(cb: (o: Orden) => void): () => void {
  return escuchar("orden", cb);
}

/** alDescargar escucha el avance de la descarga de la actualización. */
export function alDescargar(cb: (a: Avance) => void): () => void {
  return escuchar("descarga", cb);
}

/**
 * escuchar es el camino de dos vías de siempre: los eventos de Wails cuando hay
 * ventana, y el flujo del servidor de desarrollo cuando hay navegador.
 */
function escuchar<T>(evento: string, cb: (datos: T) => void): () => void {
  if (window.runtime?.EventsOn) {
    window.runtime.EventsOn(evento, (datos) => cb(datos as T));
    return () => {};
  }

  const fuente = new EventSource("/api/eventos");
  fuente.addEventListener(evento, (e) => {
    cb(JSON.parse((e as MessageEvent).data) as T);
  });
  return () => fuente.close();
}

/**
 * alEmpezarProgreso escucha cómo va una tanda de ficheros.
 *
 * Wails lo entrega por su sistema de eventos; el servidor de desarrollo, por un
 * flujo de eventos del servidor. Devuelve la función que deja de escuchar.
 */
export function alProgresar(cb: (p: Progreso) => void): () => void {
  return escuchar("progreso", cb);
}

/**
 * alSoltarFicheros escucha lo que se arrastre encima de la ventana.
 *
 * Wails entrega las rutas absolutas de lo que se ha soltado, que es lo que hace
 * falta para poder cifrarlo: el arrastrar y soltar del navegador da un objeto
 * File sin ruta, y por ahí no se llega al fichero desde Go.
 */
export function alSoltarFicheros(cb: (rutas: string[]) => void): () => void {
  const runtime = window.runtime as
    | { OnFileDrop?: (cb: (x: number, y: number, rutas: string[]) => void, usarCSS: boolean) => void }
    | undefined;

  if (runtime?.OnFileDrop) {
    // El segundo argumento es «solo sobre zonas marcadas». Va en false: se
    // acepta en toda la ventana. Con true, Wails exige que el elemento declare
    // la propiedad CSS --wails-drop-target, y soltar fuera de ella no hace nada
    // ni avisa de por qué.
    runtime.OnFileDrop((_x, _y, rutas) => cb(rutas), false);
    return () => {};
  }

  // En el navegador no hay rutas absolutas, así que solo se usa para poder
  // ejercitar la interfaz: se entregan los nombres y la propia interfaz avisa
  // de que ahí no se puede trabajar.
  const evitar = (e: DragEvent) => e.preventDefault();
  const soltar = (e: DragEvent) => {
    e.preventDefault();
    const nombres = Array.from(e.dataTransfer?.files ?? []).map((f) => f.name);
    if (nombres.length) cb(nombres);
  };
  window.addEventListener("dragover", evitar);
  window.addEventListener("drop", soltar);
  return () => {
    window.removeEventListener("dragover", evitar);
    window.removeEventListener("drop", soltar);
  };
}
