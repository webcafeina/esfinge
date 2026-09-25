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
  /** Las últimas carpetas de cada diálogo. Las lleva Go; la interfaz no las toca. */
  carpetaAbrir: string;
  carpetaGuardar: string;
  /**
   * Los dos relojes de la bóveda. **«Nunca» es NUNCA, no cero.**
   *
   * El cero significa «no lo he dicho, deja lo que había», porque es lo que
   * llega cuando alguien guarda un objeto a medias. Lo explica entero
   * internal/app/preferencias.go.
   */
  minutosParaBloquear: number;
  segundosDePortapapeles: number;
  /**
   * Trae el icono de cada sitio de la bóveda, pidiéndoselo al propio sitio.
   * **Es la segunda salida a la red del programa.**
   */
  descargarIconos: boolean;
  /** Si ya se dijo lo que eso hace. El aviso se da una vez. */
  iconosAvisados: boolean;
  /**
   * Abre el canal por el que la extensión del navegador consulta la bóveda.
   * **Viene apagado**: esto no sale a la red, abre una puerta a esta máquina.
   */
  puenteDelNavegador: boolean;
  /**
   * Si ya se ofreció desbloquear con el sistema. Se ofrece **una vez**, porque el
   * interruptor vive en Ajustes y ahí no entra quien no sabe que existe.
   */
  desbloqueoSugerido: boolean;
};

/** Lo que se manda para apagar uno de los dos relojes de la bóveda. */
export const NUNCA = -1;

/** Las cuatro clases de cosa que caben en la bóveda. Los nombres los fija Go. */
export type TipoEntrada = "credencial" | "nota" | "tarjeta" | "identidad";

/** Una contraseña que se sustituyó, con la fecha en que dejó de valer. */
export type Antigua = {
  secreto: string;
  hasta: string;
};

/**
 * Una entrada de la bóveda.
 *
 * **Llega de dos formas y hay que saber cuál se tiene.** La lista viaja sin
 * contraseñas —`buscarEnBoveda`— y la entrada entera solo cuando se pide una
 * concreta —`verDeBoveda`—. Es la regla del puente: los secretos salen de uno en
 * uno. Por eso todo lo sensible es opcional aquí.
 */
export type EntradaBoveda = {
  id: string;
  tipo: TipoEntrada;
  titulo: string;
  notas?: string;
  etiquetas?: string[];
  carpeta?: string;
  creada: string;
  cambiada: string;
  papelera?: boolean;
  borradaEn?: string;

  usuario?: string;
  secreto?: string;
  sitios?: string[];
  totp?: string;
  historial?: Antigua[];

  titular?: string;
  numero?: string;
  caduca?: string;
  verificacion?: string;

  nombreCompleto?: string;
  documento?: string;
  numeroDocumento?: string;
};

/**
 * El código de un solo uso de una entrada, ya calculado.
 *
 * **Lo que cruza el puente son las seis cifras, no la semilla**: el código
 * caduca en treinta segundos y la semilla es el segundo factor entero.
 */
export type CodigoDeUnSoloUso = {
  codigo: string;
  /** Segundos que le quedan de vida. */
  quedan: number;
  /** Cuánto dura entero, para poder dibujar qué fracción queda. */
  periodo: number;
};

/** Un navegador al que se le ha dado permiso para hablar con la bóveda. */
export type NavegadorPermitido = {
  /** Cómo se llamó a sí mismo. **No se cree**: sirve para poder enseñarlo. */
  quien: string;
  /** Cuándo se le dio el permiso. Es lo que lo identifica al retirarlo: el
   *  testigo no cruza el puente. */
  desde: string;
};

/** Cómo está el canal por el que la extensión del navegador consulta la bóveda. */
export type EstadoDelNavegador = {
  encendido: boolean;
  /** Si de verdad hay un socket abierto: el ajuste puede estar puesto y el canal
   *  no haber podido arrancar, y eso hay que poder verlo. */
  escuchando: boolean;
  donde: string;
  error?: string;
  /** El navegador que está esperando permiso, si hay alguno. */
  pide?: string;
  /**
   * Los navegadores a los que se ha dejado el manifiesto que declara el puente.
   *
   * **Se enseña**, y es lo que convierte «no funciona» en «ya veo por qué»: sin
   * esto, un navegador al que no se avisó se ve exactamente igual que uno avisado.
   */
  avisados: string[];
  permitidos: NavegadorPermitido[];
};

/** Lo que hace falta saber para decidir qué pantalla de la bóveda se enseña. */
/** Cómo va la sincronización con la cuenta (ADR 0038). */
export type EstadoSincro = {
  estado:
    | "apagada"
    | "sincronizando"
    | "al-dia"
    | "sin-conexion"
    | "sin-red"
    | "hay-que-entrar"
    | "muchos-borrados"
    | "error";
  /** Cuándo salió bien la última pasada, en RFC3339. */
  ultima?: string;
  /** Lo que hay que enseñar si algo no va bien, tal cual. */
  mensaje?: string;
};

/** Si este equipo trabaja en local o con cuenta (ADR 0035). */
export type EstadoCuenta = {
  /** Vacío: todavía no se ha elegido, y toca la bienvenida. */
  modo: "" | "local" | "cuenta";
  correo?: string;
  equipo?: string;
  servidor: string;
  sincro: EstadoSincro;
  /** Una entrada a medias espera el código del correo (al abrir con la contraseña nueva). */
  codigoPendiente?: boolean;
};

/** Un equipo con sesión en la cuenta. */
/** Lo que se enseña de una identidad: su huella, y nada más. */
export type IdentidadParaCompartir = { huella: string; suite: string };

/**
 * Lo que la ventana necesita del desbloqueo del sistema (fase C).
 *
 * `nombre` es cómo lo llama cada sistema —«Touch ID», «Windows Hello»— y viene
 * vacío donde no hay ninguno: entonces **no se ofrece nada**, en vez de enseñar
 * un botón que no puede funcionar.
 */
export type EstadoDesbloqueo = {
  hay: boolean;
  nombre: string;
  puesto: boolean;
  /**
   * Esta versión todavía no tiene el permiso del llavero, así que la primera
   * huella traerá un diálogo del sistema pidiendo la contraseña del equipo. Pasa
   * **una vez por actualización**: sin firmar, cada versión es un binario nuevo.
   */
  trasActualizar: boolean;
};

/**
 * Una copia mandada a quien todavía no tenía cuenta, esperando (ADR 0043, B3).
 * **No lleva el secreto**: dice a quién, cuándo y con qué llaves se intentó.
 */
export type EnvioEsperando = {
  id: string;
  entrada: string;
  correo: string;
  huella: string;
  creado: string;
};

/** Un envío esperando en el buzón, ya abierto y comprobado. */
export type EnvioRecibido = {
  id: string;
  huella: string;
  titulo: string;
  usuario: string;
  tipo: string;
  momento: number;
  /** Por qué no se puede abrir, si es el caso. Se enseña en vez de esconderlo. */
  error?: string;
};

export type EquipoDeCuenta = {
  id: string;
  nombre: string;
  /** Cuándo se usó por última vez, en RFC3339. */
  visto: string;
  actual: boolean;
};

/** En qué punto se ha quedado entrar en una cuenta. */
export type ResultadoEntrada = {
  necesitaCodigo: boolean;
  hayOtraBoveda: boolean;
  listo: boolean;
  /** Dónde ha quedado la bóveda que había en este equipo, si se apartó. */
  apartada?: string;
};

export type EstadoBoveda = {
  existe: boolean;
  abierta: boolean;
  ruta: string;
  cuantas: number;
  soloLectura: boolean;
  minutosParaBloquear: number;
  /** Cuántas entradas hay en la papelera, para saber si enseñar el botón. */
  enLaPapelera: number;
};

/** Lo que se cuenta después de traer un CSV de otro gestor. */
export type ResumenImportacion = {
  metidas: number;
  /** Ya estaban **exactamente igual**, así que no se han vuelto a meter. */
  repetidas: number;
  /** Ya estaban **con otra contraseña**: ésas sí entran, marcadas. */
  conflictos: number;
  /** Las filas que traía el fichero, sin la cabecera. Es lo que contesta «¿están todas?». */
  filas: number;
  /** Las filas que no llevaban nada que guardar. */
  vacias: number;
  deDonde: string;
  /** El CSV del que se importó, para poder ofrecer borrarlo. */
  fichero: string;
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

  /** Si el sistema ha puesto una ventana translúcida detrás. */
  vidrio: () => llamar<boolean>("Vidrio"),

  /** «darwin», «windows» o «linux»: lo que devuelve Go. */
  plataforma: () => llamar<string>("Plataforma"),

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

  // --------------------------------------------------------------- la bóveda

  /**
   * copiar pone algo en el portapapeles **desde Go**, que es lo que arma el
   * borrado pasado el plazo.
   *
   * Copiar sabe hacerlo el navegador; borrar pasado un rato, no: el temporizador
   * de un webview muere al recargar y el sistema lo puede pausar, y entonces un
   * secreto se queda ahí para siempre creyendo que se limpió.
   */
  copiar: (texto: string) => llamar<void>("Copiar", texto),

  /** Dice que alguien está usando la aplicación, para aplazar el bloqueo. */
  actividad: () => llamar<void>("Actividad"),

  // --------------------------------------------------------------- la cuenta

  estadoDeCuenta: () => llamar<EstadoCuenta>("EstadoDeCuenta"),

  /**
   * Repite la pasada aceptando una fusión que se lleve más de la mitad de las
   * entradas: la salida de «Parada: los cambios de otro equipo borrarían media
   * bóveda». Vale para una sola pasada.
   */
  sincronizarAunqueBorre: () => llamar<void>("SincronizarAunqueBorre"),

  /** Lo que se elige en la bienvenida para trabajar sin cuenta. */
  elegirModoLocal: () => llamar<void>("ElegirModoLocal"),

  sincronizarAhora: () => llamar<void>("SincronizarAhora"),

  /**
   * Deja la cuenta en este equipo: la bóveda se queda aquí tal como está y deja de
   * sincronizarse. La cuenta sigue en el servidor para los demás equipos.
   */
  salirDeCuenta: (maestra: string) => llamar<void>("SalirDeCuenta", maestra),

  /** Manda un código al correo para recuperar la cuenta. */
  empezarRecuperacion: (correo: string) => llamar<void>("EmpezarRecuperacion", correo),

  /**
   * Recupera la cuenta con el código, la clave de recuperación y una contraseña nueva,
   * y deja su bóveda abierta en este equipo, como al entrar.
   */
  terminarRecuperacion: (correo: string, codigo: string, clave: string, nueva: string) =>
    llamar<ResultadoEntrada>("TerminarRecuperacion", correo, codigo, clave, nueva),

  dispositivosDeCuenta: () =>
    llamar<EquipoDeCuenta[] | null>("DispositivosDeCuenta").then((l) => l ?? []),

  olvidarDispositivo: (id: string) => llamar<void>("OlvidarDispositivo", id),

  pedirCodigoParaBorrarCuenta: () => llamar<void>("PedirCodigoParaBorrarCuenta"),

  /** Borra la cuenta del servidor. La bóveda de este equipo se queda, en local. */
  borrarCuenta: (maestra: string, codigo: string) => llamar<void>("BorrarCuenta", maestra, codigo),

  /** Guarda en un fichero lo que el servidor tiene de la cuenta; devuelve dónde. */
  exportarDatosDeCuenta: () => llamar<string>("ExportarDatosDeCuenta"),

  // ---------------------------------------------------------------- compartir
  //
  // **Lo que cruza de una identidad es su huella**, nunca la semilla ni las
  // llaves (ADR 0043). Y del buzón, de quién viene y qué es: la contraseña solo
  // entra en la bóveda al aceptarla, como el resto de los secretos.

  /** La huella de esta bóveda. La crea si aún no la tiene y la publica. */
  miIdentidad: () => llamar<IdentidadParaCompartir>("MiIdentidad"),

  /**
   * La huella de quien tiene ese correo, para comparar antes de mandar.
   *
   * **Siempre contesta**, tenga cuenta o no esa dirección: el servidor no dice
   * quién está en Esfinge. Por eso lo que se enseña al lado es «compárala con
   * quien la tenga delante» y no «esta persona existe».
   */
  huellaDe: (correo: string) => llamar<IdentidadParaCompartir>("HuellaDe", correo),

  /** Manda una copia de esa entrada a ese correo. */
  mandarCopia: (id: string, correo: string) => llamar<void>("MandarCopia", id, correo),

  /** Las copias de esa entrada que esperan a que quien las recibe tenga cuenta. */
  enviosPendientes: (id: string) => llamar<EnvioEsperando[] | null>("EnviosPendientes", id).then((l) => l ?? []),

  // Desbloquear con el sistema: Touch ID o Windows Hello (fase C). **El secreto
  // no cruza por aquí**: vive en el llavero del sistema y lo lee Go.
  estadoDelDesbloqueo: () => llamar<EstadoDesbloqueo>("EstadoDelDesbloqueo"),
  activarDesbloqueo: () => llamar<void>("ActivarDesbloqueo"),
  quitarDesbloqueo: () => llamar<void>("QuitarDesbloqueo"),
  abrirBovedaConElSistema: () => llamar<void>("AbrirBovedaConElSistema"),

  /** Lo que ha llegado, abierto y con la firma comprobada, pero sin secretos. */
  buzon: () => llamar<EnvioRecibido[] | null>("Buzon").then((l) => l ?? []),

  /** Mete la copia en la bóveda, con identificador nuevo, y la quita del buzón. */
  aceptarDelBuzon: (id: string) => llamar<void>("AceptarDelBuzon", id),

  /** La tira sin abrirla. */
  tirarDelBuzon: (id: string) => llamar<void>("TirarDelBuzon", id),

  /** Manda el código al correo para crear la cuenta. */
  empezarRegistro: (correo: string) => llamar<void>("EmpezarRegistro", correo),

  /**
   * Crea la cuenta. Si en este equipo no había bóveda, se crea una y **devuelve su
   * clave de recuperación, que es la única vez que se ve**; si la había, vacío.
   */
  terminarRegistro: (correo: string, codigo: string, maestra: string, nueva = "") =>
    llamar<string>("TerminarRegistro", correo, codigo, maestra, nueva),

  entrarEnCuenta: (correo: string, maestra: string) =>
    llamar<ResultadoEntrada>("EntrarEnCuenta", correo, maestra),

  confirmarEntrada: (codigo: string) => llamar<ResultadoEntrada>("ConfirmarEntrada", codigo),

  /** Juntar la bóveda que había aquí con la de la cuenta, o apartarla sin más. */
  resolverOtraBoveda: (juntar: boolean, maestraLocal: string) =>
    llamar<ResultadoEntrada>("ResolverOtraBoveda", juntar, maestraLocal),

  // --------------------------------------------------------------- la bóveda en sí

  estadoBoveda: () => llamar<EstadoBoveda>("EstadoBoveda"),

  /** Devuelve la clave de recuperación, **y es la única vez que se ve**. */
  crearBoveda: (maestra: string) => llamar<string>("CrearBoveda", maestra),

  /** Vale la contraseña maestra o la clave de recuperación; no hay que decir cuál. */
  abrirBoveda: (llave: string) => llamar<void>("AbrirBoveda", llave),

  cerrarBoveda: () => llamar<void>("CerrarBoveda"),

  /** La lista, **sin contraseñas**. */
  buscarEnBoveda: (q: string) =>
    llamar<EntradaBoveda[] | null>("BuscarEnBoveda", q).then((l) => l ?? []),

  /** Una entrada entera, con su secreto. De una en una a propósito. */
  verDeBoveda: (id: string) => llamar<EntradaBoveda>("VerDeBoveda", id),

  /**
   * El código de un solo uso que vale **ahora**, calculado en Go.
   *
   * Se pide de nuevo cada vez que caduca, y por eso **no cuenta como
   * actividad** al otro lado: si contara, una entrada abierta encima de la mesa
   * mantendría la bóveda abierta para siempre.
   */
  codigoDeBoveda: (id: string) => llamar<CodigoDeUnSoloUso>("CodigoDeBoveda", id),

  guardarEnBoveda: (e: EntradaBoveda) => llamar<void>("GuardarEnBoveda", e),

  /** Manda la entrada a la papelera, **entera**: de ahí se saca durante 30 días. */
  borrarDeBoveda: (id: string) => llamar<void>("BorrarDeBoveda", id),

  /** Lo borrado que todavía se puede recuperar, sin secretos y con lo último arriba. */
  /** Cuántas entradas sobran por ser iguales en todo a otra. */
  repetidasEnBoveda: () => llamar<number>("RepetidasEnBoveda"),

  /** Manda a la papelera las repetidas; devuelve cuántas. */
  quitarRepetidasDeBoveda: () => llamar<number>("QuitarRepetidasDeBoveda"),

  papeleraDeBoveda: () =>
    llamar<EntradaBoveda[] | null>("PapeleraDeBoveda").then((l) => l ?? []),

  restaurarDeBoveda: (id: string) => llamar<void>("RestaurarDeBoveda", id),

  /** Quita una entrada de la papelera. Esta vez no hay vuelta atrás. */
  borrarDelTodoDeBoveda: (id: string) => llamar<void>("BorrarDelTodoDeBoveda", id),

  /** Vacía la papelera entera y dice cuántas entradas se ha llevado. */
  vaciarPapeleraDeBoveda: () => llamar<number>("VaciarPapeleraDeBoveda"),

  cambiarMaestraDeBoveda: (vieja: string, nueva: string) =>
    llamar<void>("CambiarMaestraDeBoveda", vieja, nueva),

  /** Genera otra clave de recuperación y deja la anterior inservible. */
  rotarRecuperacionDeBoveda: () => llamar<string>("RotarRecuperacionDeBoveda"),

  /**
   * Borra la bóveda del disco. **No hay vuelta atrás y la clave de recuperación
   * no sirve de nada**: lo que se borra es el fichero que ella abriría.
   */
  borrarBoveda: (maestra: string) => llamar<void>("BorrarBoveda", maestra),

  /**
   * Los iconos de los sitios, por anfitrión.
   *
   * **Van por su propio método y no dentro de la lista de entradas**: la lista se
   * vuelve a pedir en cada tecla del buscador, y meterlos ahí sería mandarlos
   * todos por el puente en cada pulsación.
   */
  iconosDeBoveda: () =>
    llamar<Record<string, string> | null>("IconosDeBoveda").then((m) => m ?? {}),

  importarEnBoveda: (deDonde: string) =>
    llamar<ResumenImportacion>("ImportarEnBoveda", deDonde),

  estadoDelNavegador: () =>
    // **Con la lista puesta a la fuerza.** Una porción vacía de Go se serializa
    // como `null`, no como `[]`, y un `null.length` en el render tira **el panel
    // entero de Ajustes**, no solo esa línea: React desmonta el árbol y la
    // pantalla se queda en blanco. Es el mismo remiendo que ya llevan
    // `buscarEnBoveda` y `papeleraDeBoveda`, y por lo mismo.
    llamar<EstadoDelNavegador>("EstadoDelNavegador").then((e) => ({
      ...e,
      permitidos: e.permitidos ?? [],
      avisados: e.avisados ?? [],
    })),

  /** El «sí» de la persona: el navegador que esté esperando recibe su permiso. */
  permitirNavegador: () => llamar<void>("PermitirNavegador"),

  /** Retira un permiso dado, por la fecha en que se dio. */
  olvidarNavegador: (desde: string) => llamar<void>("OlvidarNavegador", desde),

  /** Escribe las entradas **en claro**, por el diálogo del sistema. */
  exportarBoveda: () => llamar<string>("ExportarBoveda"),

  borrarElCSVImportado: (ruta: string) => llamar<void>("BorrarElCSVImportado", ruta),

  /**
   * Los sitios en los que la extensión no ofrece guardar (ADR 0032). **Nunca nula**:
   * un `null` de Go aquí ya tumbó Ajustes una vez.
   */
  sitiosExcluidos: () => llamar<string[]>("SitiosExcluidos").then((l) => l ?? []),

  /** Vuelve a dejar que la extensión ofrezca guardar en ese sitio. */
  quitarSitioExcluido: (dominio: string) => llamar<void>("QuitarSitioExcluido", dominio),
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
 * alBloquearseLaBoveda avisa de que la bóveda se ha cerrado sola.
 *
 * Lo decide un reloj de Go y no uno de aquí, y por la misma razón por la que el
 * portapapeles se borra desde allí: los temporizadores de un webview se pausan y
 * se pierden al recargar, y un bloqueo que a veces no ocurre no es un bloqueo.
 */
/** Avisa de que un navegador quiere conectarse a la bóveda. */
/** Avisa de cómo va la sincronización con la cuenta. */
export function alCambiarLaSincro(cb: (e: EstadoSincro) => void): () => void {
  return escuchar("sincro", cb);
}

export function alPedirloUnNavegador(cb: (quien: string) => void) {
  return escuchar("navegador-pide", cb);
}

/**
 * Avisa cada vez que la bóveda se abre o se cierra, por el camino que sea: a mano,
 * sola por inactividad, al entrar en la cuenta o al borrarla. Lo usa el candado de
 * la barra lateral.
 */
export function alCambiarElEstadoDeLaBoveda(cb: (abierta: boolean) => void): () => void {
  return escuchar("boveda-estado", cb);
}

export function alBloquearseLaBoveda(cb: () => void): () => void {
  return escuchar("boveda-bloqueada", cb);
}

/** alHaberIconos avisa de que la tanda de fondo ha traído alguno nuevo. */
export function alHaberIconos(cb: () => void): () => void {
  return escuchar("iconos", cb);
}

/**
 * alCambiarLaBoveda avisa de que **alguien de fuera ha escrito en la bóveda**: el
 * navegador ha guardado o actualizado una cuenta, o ha apuntado un sitio en el que
 * no ofrecer. Sin esto, la lista se quedaba con lo de antes.
 */
export function alCambiarLaBoveda(cb: () => void): () => void {
  return escuchar("boveda-cambiada", cb);
}

/**
 * alCambiarElPortapapeles trae los segundos que le quedan a lo copiado, o cero
 * cuando ya se ha borrado.
 */
export function alCambiarElPortapapeles(cb: (segundos: number) => void): () => void {
  return escuchar("portapapeles", cb);
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

  const oyente = (e: Event) => cb(JSON.parse((e as MessageEvent).data) as T);
  const fuente = laFuente();
  fuente.addEventListener(evento, oyente);
  return () => fuente.removeEventListener(evento, oyente);
}

/**
 * laFuente es **una sola conexión de eventos para todos los oyentes**.
 *
 * Un EventSource por suscripción parece inocente y no lo es. El navegador solo
 * abre seis conexiones a la vez contra el mismo origen, y un flujo de eventos no
 * termina nunca: a partir del sexto oyente **toda llamada al puente se queda
 * esperando para siempre**, sin error, sin petición en la red y sin nada que
 * mirar. Con cinco oyentes la aplicación funcionaba; la bóveda trajo el sexto y
 * el síntoma fue un botón de copiar que no hacía nada.
 *
 * Esto solo pasa en el navegador. En la ventana los eventos los reparte Wails
 * por dentro y no hay ninguna conexión de por medio.
 */
let fuenteUnica: EventSource | null = null;

function laFuente(): EventSource {
  if (!fuenteUnica) fuenteUnica = new EventSource("/api/eventos");
  return fuenteUnica;
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
