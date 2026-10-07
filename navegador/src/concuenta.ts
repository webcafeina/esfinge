/**
 * La extensión con cuenta (ADR 0040): la bóveda vive dentro del navegador y se
 * sincroniza con el servidor de cuentas **sin la aplicación**.
 *
 * Vive en el trabajador de fondo, que es lo único que dura algo en MV3, y aun así
 * se muere solo cada pocos minutos. Así que **nada de lo que importa vive solo en
 * memoria**:
 *
 * - En `storage.local`, que sobrevive a todo: la bóveda **cifrada** —el mismo
 *   documento que la aplicación tiene en el disco—, la base de la fusión, qué
 *   versión del servidor se vio, y los datos de la cuenta: el correo, la sesión
 *   **sellada con la clave de la bóveda** y el testigo de confianza del equipo.
 *   Sin la contraseña maestra, nada de eso habla con el servidor ni se lee.
 * - En `storage.session`, que vive **en memoria** y se va al cerrar el navegador:
 *   la clave de la bóveda abierta y cuándo se tocó por última vez. Es lo que deja
 *   la bóveda abierta aunque el trabajador se duerma, y lo que hace que cerrar el
 *   navegador la cierre. Los guiones de las páginas no pueden leerlo.
 *
 * Y dos reglas de la aplicación que aquí se cumplen igual:
 *
 * - **Solo cuenta como actividad lo que hace una persona**: abrir el panel, rellenar
 *   desde él, copiar, desbloquear. El relleno automático, el refresco del icono y
 *   la sincronización no, o la bóveda no se cerraría nunca sola.
 * - **Un 401 cierra la bóveda** (2.24.5): si la cuenta ya no reconoce este
 *   navegador —se olvidó desde otro equipo, se cambió la contraseña, caducó—, se
 *   cierra y se olvida la sesión.
 */

import { api } from "./api";
import { Boveda, ErrorBoveda } from "./nucleo/boveda";
import { Cliente, ErrorDeRed, RAIZ_POR_DEFECTO, revocado, sesionCaducada, sinAcceso, sinBoveda, type Sesion } from "./nucleo/cliente";
import { derivarAcceso, normalizarCorreo } from "./nucleo/cuenta";
import { base64url, desdeBase64 } from "./nucleo/esf1";
import { abrirEnvio, mandarEntrada, type Envio } from "./nucleo/envio";
import { huellaDeIdentidad, identidadDeSemilla } from "./nucleo/identidad";
import { fundir } from "./nucleo/fundir";
import { atender } from "./nucleo/fuente";
import { pasada, pendiente, type Memoria, type Recuerdo } from "./nucleo/sincro";
import type { Peticion, Respuesta } from "./protocolo";

// ------------------------------------------------------------------ lo guardado

const L = {
  datos: "cuenta",
  boveda: "cuenta-boveda",
  base: "cuenta-base",
  recuerdo: "cuenta-recuerdo",
  /** La que había aquí cuando una fusión no se pudo hacer. **No se borra sola.** */
  apartada: "cuenta-boveda-apartada",
  /** La huella ya publicada en el servidor, para no repetir el envío cada minuto. */
  llaves: "cuenta-llaves-publicadas",
} as const;
const S = {
  llave: "cuenta-llave",
  actividad: "cuenta-actividad",
  sincro: "cuenta-sincro",
  entrando: "cuenta-entrando",
  /**
   * La bóveda de proyecto abierta, o vacío si es la personal (ADR 0050). **Una a la
   * vez**, igual que en la aplicación: esto es una cadena y no un conjunto.
   */
  activa: "cuenta-activa",
  /**
   * **De quién** es la bóveda abierta (ADR 0052): vacío es mía —la personal o un
   * proyecto mío—, y con algo dentro es la cuenta de otra persona.
   *
   * Va aparte de `activa` y no pegado a ella porque la referencia es la de **esa**
   * cuenta: dos personas pueden tener proyectos con la misma referencia sin saberlo, y
   * lo que distingue una bóveda no es la referencia sino la pareja.
   */
  dueno: "cuenta-dueno",
  /**
   * La clave de la bóveda **personal**, mientras hay un proyecto abierto: sin ella,
   * cambiar de proyecto pediría la contraseña maestra cada vez.
   *
   * En `storage.session` como la otra, así que **se va al cerrar el navegador** y
   * los guiones de las páginas no la leen. Y se borra al bloquear, o el plazo de
   * los quince minutos dejaría de significar lo que dice en las demás bóvedas.
   */
  llavePrincipal: "cuenta-llave-principal",
} as const;

/**
 * Qué se le añade a la clave de `storage.local` para hablar de una bóveda y no de
 * otra: nada para la personal, la referencia para un proyecto mío (ADR 0050) y
 * **`de:<dueño>:<ref>` para una de otra persona** (ADR 0052).
 *
 * Sin sufijo para la personal **a propósito**: es lo que ya está escrito en los
 * navegadores que hay, así que no hace falta migrar nada.
 *
 * Y lo ajeno va con su marca delante, como en la aplicación va en su propia carpeta:
 * una referencia a secas podría ser un proyecto mío o el de otro, y confundirlos sería
 * subir una bóveda a la dirección equivocada.
 */
function sufijoDeBoveda(ref: string, dueno: string): string {
  if (dueno !== "") return `:de:${dueno}:${ref}`;
  return ref === "" ? "" : `:${ref}`;
}

function claveDeBoveda(ref: string, dueno = ""): string {
  return L.boveda + sufijoDeBoveda(ref, dueno);
}

/** Y lo mismo para lo que la sincronización recuerda de cada una. */
function claveDeBase(ref: string, dueno = ""): string {
  return L.base + sufijoDeBoveda(ref, dueno);
}
function claveDeRecuerdo(ref: string, dueno = ""): string {
  return L.recuerdo + sufijoDeBoveda(ref, dueno);
}

/** Los datos de la cuenta en este navegador. Nada de esto sirve sin la maestra, salvo el testigo. */
type Datos = {
  correo: string;
  servidor: string;
  /** El nombre con que este navegador sale en la lista de equipos: «Chrome en Mac». */
  equipo: string;
  /** La sesión, **sellada con la clave de la bóveda**. Vacía si hay que volver a entrar. */
  sesion: string;
  /** El testigo de «este equipo es de confianza», en claro, como en la aplicación (ADR 0037). */
  confianza: string;
};

export type EstadoSincro = {
  estado:
    | "apagada"
    | "sincronizando"
    | "al-dia"
    | "sin-conexion"
    | "hay-que-entrar"
    | "muchos-borrados"
    /**
     * Te han quitado el acceso a esta bóveda ajena, o solo puedes verla (ADR 0052).
     * **No es «hay-que-entrar»**: la sesión está bien y la bóveda propia no se toca.
     */
    | "sin-acceso"
    | "error";
  ultima?: string;
  mensaje?: string;
};

export type EstadoDeCuenta = {
  modo: "local" | "cuenta";
  correo?: string;
  abierta: boolean;
  /** Hay una entrada a medias esperando el código del correo. */
  codigoPendiente: boolean;
  /** Hay una bóveda de este navegador apartada porque no se pudo fundir. */
  apartada: boolean;
  sincro: EstadoSincro;
  /** Minutos sin tocar nada antes de cerrarse sola. */
  bloqueo: number;
  /**
   * En qué bóveda se está trabajando (ADR 0050): la referencia, vacío la personal,
   * y su nombre para poder enseñarlo. Y la lista, para poder cambiar.
   *
   * Va dentro del estado y no en una petición aparte **porque el panel lo necesita
   * cada vez que se abre**, y pedirlo con la bóveda cerrada sería un error seguro.
   */
  activa: string;
  nombreActiva: string;
  proyectos: { ref: string; nombre: string; enEsteNavegador: boolean }[];
  /**
   * De quién es la bóveda abierta (ADR 0052): vacío, mía. Y las bóvedas ajenas a las
   * que tengo acceso, para poder cambiar a ellas.
   *
   * El permiso viaja **informativo**: el que manda es el del servidor, que contesta
   * 403. Sirve para dos cosas que no se pueden hacer después del 403 — decir «solo
   * puedes ver» antes de que alguien lo intente, y no ofrecer guardar en una página.
   */
  duenoActiva: string;
  compartidas: { dueno: string; ref: string; nombre: string; permiso: string; enEsteNavegador: boolean }[];
  /** La abierta es ajena y solo se puede mirar. Lo mira el panel y lo mira la fuente. */
  soloPuedoVer: boolean;
};

/** Lo que el panel —y solo el panel— le puede pedir a la cuenta. */
export type PeticionDeCuenta =
  | { cuenta: "estado" }
  | { cuenta: "entrar"; correo: string; maestra: string }
  | { cuenta: "codigo"; codigo: string }
  /** Deja la entrada a medias: se vuelve a empezar. */
  | { cuenta: "cancelar" }
  | { cuenta: "desbloquear"; maestra: string }
  | { cuenta: "bloquear" }
  | { cuenta: "salir" }
  /** Con `igual`, acepta una fusión que se lleve media bóveda: la salida de la parada. */
  | { cuenta: "sincronizar"; igual?: boolean }
  // Compartir (ADR 0043). **Como todo lo de la cuenta, solo por el puerto del
  // panel**: una página no puede pedir nada de esto.
  | { cuenta: "miHuella" }
  | { cuenta: "huellaDe"; correo: string }
  | { cuenta: "compartir"; id: string; correo: string }
  | { cuenta: "buzon" }
  | { cuenta: "aceptar"; envio: string }
  | { cuenta: "tirar"; envio: string }
  // Las bóvedas de proyecto (ADR 0050). **Una abierta a la vez**: abrir una cierra
  // la anterior, así que todo lo demás —rellenar, los códigos, guardar— sigue
  // hablando de «la bóveda» y contesta de la que esté activa.
  | { cuenta: "abrirProyecto"; ref: string }
  // Y las bóvedas de otras personas (ADR 0052). **Solo abrirlas**: dar acceso, quitarlo
  // y aceptarlo pasa entero en la ventana, y el panel ni lo nombra.
  | { cuenta: "abrirCompartida"; dueno: string; ref: string }
  | { cuenta: "volverALaPersonal" };

/** Lo que el panel enseña de un envío: de quién viene y qué es, sin secretos. */
export type EnvioParaElPanel = {
  id: string;
  huella: string;
  titulo: string;
  usuario: string;
  error?: string;
};

export type RespuestaDeCuenta = {
  ok: boolean;
  /** La huella pedida, si la petición era una de las dos que la traen. */
  huella?: string;
  buzon?: EnvioParaElPanel[];
  error?: string;
  estado?: EstadoDeCuenta;
  /** Ha llegado un código al correo: se sigue con `codigo`. */
  necesitaCodigo?: boolean;
};

/** Lo que tarda en cerrarse sola sin tocar nada, lo mismo que la aplicación por defecto. */
export const MINUTOS_DE_BLOQUEO = 15;
/** Lo que se espera tras guardar antes de subir, para juntar los guardados seguidos. */
const ESPERA_TRAS_GUARDAR = 3000;
export const ALARMA_DE_LA_CUENTA = "cuenta-reloj";

async function local<T>(clave: string): Promise<T | undefined> {
  return (await api.storage.local.get(clave))[clave] as T | undefined;
}
async function sesion<T>(clave: string): Promise<T | undefined> {
  return (await api.storage.session.get(clave))[clave] as T | undefined;
}

async function datos(): Promise<Datos | undefined> {
  return local<Datos>(L.datos);
}

// ------------------------------------------------------------------ la bóveda abierta

/**
 * La bóveda abierta **en este trabajador**. Se rehace desde `storage.session` al
 * despertar: la clave está allí y el documento en `storage.local`.
 */
let abierta: Boveda | null = null;
/**
 * Una entrada a medias: el reto del código y la maestra, esperando el código del
 * correo.
 *
 * **En `storage.session` y no en una variable**, y lo contó el cliente la primera
 * vez que lo probó: para leer el código hay que ir al correo, el panel se cierra y
 * el trabajador de fondo se duerme a los pocos segundos. En una variable, al volver
 * no había nada y había que empezar otra vez, con otro correo. `storage.session`
 * vive en memoria, no la leen las páginas y se va al cerrar el navegador; y esto se
 * borra al confirmar, al cancelar y **a los diez minutos**, que es lo que dura el
 * código.
 */
type Entrando = { correo: string; maestra: string; reto: string; servidor: string; equipo: string; hasta: number };
const VIDA_DEL_CODIGO_MS = 10 * 60_000;

async function entrando(): Promise<Entrando | null> {
  const e = await sesion<Entrando>(S.entrando);
  if (!e) return null;
  if (Date.now() > e.hasta) {
    await api.storage.session.remove(S.entrando);
    return null;
  }
  return e;
}

async function olvidarEntrada(): Promise<void> {
  await api.storage.session.remove(S.entrando);
}

async function laBoveda(): Promise<Boveda | null> {
  if (abierta?.abierta) return abierta;
  const llave = await sesion<string>(S.llave);
  const ref = await refActiva();
  const dueno = await duenoActivo();
  const texto = await local<string>(claveDeBoveda(ref, dueno));
  if (!llave || !texto) return null;
  try {
    abierta = await Boveda.abrirConClave(texto, llave);
    conectar(abierta, ref, dueno);
    return abierta;
  } catch {
    await api.storage.session.remove(S.llave);
    return null;
  }
}

/** Qué bóveda está abierta: vacío, la personal. */
export async function refActiva(): Promise<string> {
  return (await sesion<string>(S.activa)) ?? "";
}

/** Y de quién es (ADR 0052): vacío, mía. */
export async function duenoActivo(): Promise<string> {
  return (await sesion<string>(S.dueno)) ?? "";
}

/**
 * Hace algo con la **bóveda personal**, esté abierta o detrás de un proyecto.
 *
 * Es el espejo de `conLaPersonal` de Go, y hace falta por lo mismo: la lista de
 * proyectos, la sesión de la cuenta y la identidad para compartir viven en la
 * personal, y con un proyecto delante la personal no es la que está abierta. Con la
 * clave que se guardó al conmutar se abre su documento el instante que dure la
 * operación. **Secuencial, nunca dos abiertas de cara a quien mira.**
 */
async function conLaPersonal<T>(hacer: (b: Boveda) => Promise<T>): Promise<T> {
  // **Las dos cosas, no solo la referencia**: una bóveda ajena tiene referencia, así
  // que preguntar solo por ella bastaría hoy —las referencias no son vacías— y dejaría
  // de bastar el día que algo pudiera abrirse sin ella.
  if ((await refActiva()) === "" && (await duenoActivo()) === "") {
    const b = await laBoveda();
    if (!b) throw new ErrorBoveda("cerrada");
    return hacer(b);
  }
  const llave = await sesion<string>(S.llavePrincipal);
  const texto = await local<string>(claveDeBoveda(""));
  if (!llave || !texto) throw new ErrorBoveda("cerrada");
  const p = await Boveda.abrirConClave(texto, llave);
  try {
    return await hacer(p);
  } finally {
    p.cerrar();
  }
}

/** Cada guardado va a `storage.local` y pide subir, con una espera para juntar los seguidos. */
function conectar(b: Boveda, ref: string, dueno = "") {
  b.alGuardar = (texto) => {
    api.storage.local.set({ [claveDeBoveda(ref, dueno)]: texto }).catch(() => {});
    pedirSincro(ESPERA_TRAS_GUARDAR);
  };
}

async function ponerLaAbierta(b: Boveda, ref = "", dueno = "") {
  abierta = b;
  conectar(b, ref, dueno);
  await api.storage.local.set({ [claveDeBoveda(ref, dueno)]: b.documento() });
  await api.storage.session.set({
    [S.llave]: b._llaveParaLaSesion(),
    [S.actividad]: Date.now(),
    [S.activa]: ref,
    [S.dueno]: dueno,
  });
}

export async function bloquear(): Promise<void> {
  abierta?.cerrar();
  abierta = null;
  // **Y la clave de la bóveda personal, si había un proyecto abierto.** Sin esta
  // línea, bloquear cerraría el proyecto y dejaría en memoria con qué abrir todos
  // los demás: el plazo dejaría de significar lo que dice justo en las bóvedas que
  // no se están mirando.
  await api.storage.session.remove([S.llave, S.actividad, S.activa, S.dueno, S.llavePrincipal]);
}

/** Lo que hace una persona. **Nada que se repita solo llama aquí.** */
export async function actividad(): Promise<void> {
  if (await sesion<string>(S.llave)) await api.storage.session.set({ [S.actividad]: Date.now() });
}

// ------------------------------------------------------------------ el estado

async function ponerSincro(e: EstadoSincro) {
  const antes = await sesion<EstadoSincro>(S.sincro);
  await api.storage.session.set({ [S.sincro]: { ...e, ultima: e.ultima ?? antes?.ultima } });
}

export async function conCuenta(): Promise<boolean> {
  return (await datos()) !== undefined;
}

export async function estado(): Promise<EstadoDeCuenta> {
  const d = await datos();
  const b = await laBoveda();
  let sincro = (await sesion<EstadoSincro>(S.sincro)) ?? { estado: "apagada" };
  if (d && !d.sesion) sincro = { estado: "hay-que-entrar", mensaje: sincro.mensaje ?? "Vuelve a entrar en la cuenta para sincronizar" };
  else if (!b) sincro = { ...sincro, estado: sincro.estado === "hay-que-entrar" ? sincro.estado : "apagada" };
  const activa = await refActiva();
  const dueno = await duenoActivo();
  // Las dos listas solo se pueden leer con la bóveda abierta —viven en el cuerpo
  // cifrado de la personal—, así que cerrada se contestan vacías y el panel no enseña
  // nada: es lo mismo que pasa con las cuentas.
  let proyectos: { ref: string; nombre: string; enEsteNavegador: boolean }[] = [];
  let compartidas: EstadoDeCuenta["compartidas"] = [];
  if (b) {
    try {
      const leido = await conLaPersonal(async (p) => {
        const mios = p.proyectos().filter((x) => !x.archivado);
        const ajenas = p.compartidas();
        // **Una sola lectura de `storage` para las dos listas**: pedirla por bóveda
        // haría una llamada por fila cada vez que se abre el panel.
        const claves = [...mios.map((x) => claveDeBoveda(x.ref)), ...ajenas.map((x) => claveDeBoveda(x.ref, x.dueno))];
        const hay = await api.storage.local.get(claves);
        return {
          proyectos: mios.map((x) => ({
            ref: x.ref,
            nombre: x.nombre,
            enEsteNavegador: hay[claveDeBoveda(x.ref)] !== undefined,
          })),
          compartidas: ajenas.map((x) => ({
            dueno: x.dueno,
            ref: x.ref,
            nombre: x.nombre,
            permiso: x.permiso,
            enEsteNavegador: hay[claveDeBoveda(x.ref, x.dueno)] !== undefined,
          })),
        };
      });
      proyectos = leido.proyectos;
      compartidas = leido.compartidas;
    } catch {
      /* la bóveda se ha cerrado entre medias: el panel lo verá por `abierta` */
    }
  }
  const laAjenaAbierta = dueno === "" ? undefined : compartidas.find((x) => x.dueno === dueno && x.ref === activa);
  return {
    modo: d ? "cuenta" : "local",
    correo: d?.correo,
    abierta: b !== null,
    codigoPendiente: (await entrando()) !== null,
    apartada: (await local<string>(L.apartada)) !== undefined,
    sincro,
    bloqueo: MINUTOS_DE_BLOQUEO,
    activa,
    nombreActiva: (dueno === "" ? proyectos.find((x) => x.ref === activa)?.nombre : laAjenaAbierta?.nombre) ?? "",
    proyectos,
    duenoActiva: dueno,
    compartidas,
    // **Sin saber el permiso no se da por bueno escribir.** Con la bóveda ajena abierta
    // y la lista todavía sin leer, `laAjenaAbierta` es `undefined`: ahí lo prudente es
    // «solo puedo ver», porque equivocarse hacia ahí cuesta una tarjeta que no sale y
    // equivocarse al otro lado cuesta escribir algo que no va a subir nunca.
    soloPuedoVer: dueno !== "" && laAjenaAbierta?.permiso !== "editar",
  };
}

// ------------------------------------------------------------------ los proyectos

/**
 * Abre una bóveda de proyecto: cierra la que haya y deja ésa (ADR 0050).
 *
 * Si no está en este navegador se baja del servidor, que es lo que hace que un
 * navegador nuevo pueda usarlas sin pasar por la aplicación.
 */
export async function abrirProyecto(ref: string): Promise<void> {
  if (ref === "") return volverALaPersonal();
  // La clave de la personal: la de la abierta si es ella, o la guardada al conmutar.
  const llavePrincipal =
    (await refActiva()) === ""
      ? (await laBoveda())?._llaveParaLaSesion()
      : await sesion<string>(S.llavePrincipal);
  if (!llavePrincipal) throw new ErrorBoveda("cerrada");

  let texto = await local<string>(claveDeBoveda(ref));
  if (texto === undefined) {
    // Dormida en este navegador: se baja. **Y se guarda solo si abre**, que es lo
    // que impide dejar puesto un documento que luego no sirve.
    const d = await datos();
    if (!d?.sesion) throw new Error("Vuelve a entrar en la cuenta para traerte esa bóveda");
    const token = await conLaPersonal((p) => p.abrirSecreto(d.sesion));
    const bajada = await new Cliente(d.servidor).bajar(token, 0, ref);
    if (!bajada) throw new Error("Esa bóveda no está en la cuenta");
    texto = bajada.datos;
  }

  const p = await Boveda.abrirProyecto(texto, llavePrincipal);
  await api.storage.local.set({ [claveDeBoveda(ref)]: texto });
  // **Se guarda la clave de la personal antes de conmutar**, porque desde aquí ya no
  // se puede volver a pedir: la que queda abierta es la del proyecto.
  await api.storage.session.set({ [S.llavePrincipal]: llavePrincipal });
  abierta?.cerrar();
  await ponerLaAbierta(p, ref);
  avisarDeCambios();
  pedirSincro(0);
}

/**
 * Abre una bóveda **de otra persona** a la que me han dado acceso (ADR 0052).
 *
 * Es `abrirProyecto` con tres diferencias, y las tres importan:
 *
 *   - **La ranura es otra**: no está envuelta con la clave de mi personal sino sellada
 *     hacia mi identidad, así que la abre `abrirCompartida` y no `abrirProyecto`.
 *   - **El fichero vive en otra cuenta**, así que se baja por la ruta de las
 *     compartidas y el dueño se guarda al conmutar.
 *   - **Puede fallar con un 403** aunque esté en mi lista: el acceso lo quita el dueño
 *     cuando quiere y mi lista no se entera hasta la siguiente pasada. Entonces no se
 *     deja nada a medias: ni documento guardado, ni bóveda conmutada.
 */
export async function abrirCompartida(dueno: string, ref: string): Promise<void> {
  if (dueno === "") return abrirProyecto(ref);
  const llavePrincipal =
    (await refActiva()) === "" && (await duenoActivo()) === ""
      ? (await laBoveda())?._llaveParaLaSesion()
      : await sesion<string>(S.llavePrincipal);
  if (!llavePrincipal) throw new ErrorBoveda("cerrada");

  // Quién soy yo en esa bóveda sale de **mi** lista, que es donde lo dejó la ventana al
  // aceptar el acceso. Sin esa fila no hay titular, y sin titular no hay ranura que
  // probar: el fichero podría estar aquí y seguiría sin poderse abrir.
  const mia = await conLaPersonal(async (p) => p.laCompartida(dueno, ref));
  if (!mia) throw new Error("Esa bóveda compartida no está en tu lista: acéptala en la aplicación de Esfinge");

  let texto = await local<string>(claveDeBoveda(ref, dueno));
  if (texto === undefined) {
    const d = await datos();
    if (!d?.sesion) throw new Error("Vuelve a entrar en la cuenta para traerte esa bóveda");
    const token = await conLaPersonal((p) => p.abrirSecreto(d.sesion));
    const bajada = await new Cliente(d.servidor).bajarCompartida(token, dueno, ref, 0);
    if (!bajada) throw new Error("Esa bóveda compartida ya no está en el servidor");
    texto = bajada.datos;
  }

  // **Se abre antes de guardar nada**, igual que un proyecto: así un acceso retirado no
  // deja puesto un documento que ya no sirve.
  const p = await conLaPersonal((mi) => Boveda.abrirCompartida(texto!, mia.titular, mi));
  await api.storage.local.set({ [claveDeBoveda(ref, dueno)]: texto });
  await api.storage.session.set({ [S.llavePrincipal]: llavePrincipal });
  abierta?.cerrar();
  await ponerLaAbierta(p, ref, dueno);
  avisarDeCambios();
  pedirSincro(0);
}

/**
 * Vuelve a la bóveda personal. **Sin pedir la contraseña**, al contrario que en la
 * aplicación: aquí la clave de la personal está en `storage.session` y lo que la
 * borra es bloquear o cerrar el navegador, que es lo mismo que la protege.
 */
export async function volverALaPersonal(): Promise<void> {
  if ((await refActiva()) === "" && (await duenoActivo()) === "") return;
  const llave = await sesion<string>(S.llavePrincipal);
  const texto = await local<string>(claveDeBoveda(""));
  if (!llave || !texto) throw new ErrorBoveda("cerrada");
  const p = await Boveda.abrirConClave(texto, llave);
  abierta?.cerrar();
  await ponerLaAbierta(p, "");
  await api.storage.session.remove(S.llavePrincipal);
  avisarDeCambios();
  pedirSincro(0);
}

// ------------------------------------------------------------------ entrar

/** El nombre de este navegador en la lista de equipos. Es un rótulo, no una prueba de nada. */
export function nombreDelEquipo(agente = navigator.userAgent): string {
  const nav = agente.includes("Edg/") ? "Edge" : agente.includes("Firefox/") ? "Firefox" : agente.includes("Chrome/") ? "Chrome" : "Un navegador";
  const so = /Mac OS X|Macintosh/.test(agente) ? "Mac" : /Windows/.test(agente) ? "Windows" : /Linux/.test(agente) ? "Linux" : "";
  return so ? `${nav} en ${so}` : nav;
}

async function entrar(correoTecleado: string, maestra: string, servidor: string): Promise<RespuestaDeCuenta> {
  const correo = normalizarCorreo(correoTecleado);
  const d = await datos();
  if (d && d.correo !== correo) throw new Error("Este navegador ya está en otra cuenta: sal de ella primero");
  const cliente = new Cliente(servidor);
  const pre = await cliente.prelogin(correo);
  const clave = await derivarAcceso(maestra, pre.sal, pre.argon2);
  const equipo = d?.equipo ?? nombreDelEquipo();
  const r = await cliente.entrar(correo, clave, equipo, d?.confianza ?? "");
  if (r.reto !== undefined) {
    await api.storage.session.set({
      [S.entrando]: { correo, maestra, reto: r.reto, servidor, equipo, hasta: Date.now() + VIDA_DEL_CODIGO_MS },
    });
    return { ok: true, necesitaCodigo: true };
  }
  await terminarEntrada(correo, maestra, servidor, equipo, r.sesion!);
  return { ok: true };
}

async function confirmar(codigo: string): Promise<RespuestaDeCuenta> {
  const p = await entrando();
  if (!p) throw new Error("Empieza otra vez: no hay ninguna entrada a medias");
  const s = await new Cliente(p.servidor).confirmar(p.reto, codigo.replace(/\D/g, ""), true);
  await terminarEntrada(p.correo, p.maestra, p.servidor, p.equipo, s);
  await olvidarEntrada();
  return { ok: true };
}

/**
 * Con la sesión en la mano: bajar la bóveda de la cuenta, abrirla con la maestra
 * y quedarse con ella. Si en el navegador ya estaba la misma bóveda —se volvió a
 * entrar, o la contraseña cambió en otro equipo—, **se funde** con lo de aquí en
 * vez de pisarlo, como hace la aplicación.
 */
async function terminarEntrada(correo: string, maestra: string, servidor: string, equipo: string, s: Sesion) {
  const cliente = new Cliente(servidor);
  let bajada: { datos: string; version: number } | null;
  try {
    bajada = await cliente.bajar(s.sesion, 0);
  } catch (e) {
    if (sinBoveda(e)) throw new Error("Esta cuenta todavía no tiene bóveda: créala desde la aplicación de Esfinge");
    throw e;
  }
  if (!bajada) throw new Error("El servidor no ha devuelto la bóveda");
  let remota: Boveda;
  try {
    remota = await Boveda.abrir(bajada.datos, maestra);
  } catch {
    throw new Error("La contraseña de la cuenta no abre la bóveda que hay en ella");
  }

  let queda = remota;
  let recuerdo: Recuerdo = { version: bajada.version, serie: remota.serie };
  const aqui = await local<string>(L.boveda);
  if (aqui) {
    try {
      const vieja = await Boveda.abrirConLaLlaveDe(aqui, remota);
      const base = (await local<string>(L.base)) ?? null;
      const f = await fundir(vieja, bajada.datos, bajada.version, base);
      queda = vieja;
      recuerdo = { version: bajada.version, serie: f.serie };
      remota.cerrar();
    } catch {
      // **Lo de aquí no se pisa** (revisión del 2026-09-23). El comentario de antes
      // decía que lo de aquí era «una copia de esa misma cuenta», y desde la E2 eso
      // ya no es verdad: la extensión guarda contraseñas, y si la fusión no se puede
      // hacer —«muchos borrados», una bóveda a medias— escribir encima se llevaba
      // por delante lo guardado aquí y no subido, sin aviso y sin copia. La
      // aplicación, en el mismo caso, aparta el fichero y dice dónde queda.
      //
      // Aquí se hace lo mismo: manda la de la cuenta para poder seguir trabajando, y
      // la de este navegador se guarda aparte. El panel lo dice y no se borra sola.
      await api.storage.local.set({ [L.apartada]: aqui });
    }
  }

  const d: Datos = {
    correo,
    servidor,
    equipo,
    sesion: await queda.sellarSecreto(s.sesion),
    confianza: s.confianza ?? (await datos())?.confianza ?? "",
  };
  await api.storage.local.set({ [L.datos]: d, [L.base]: bajada.datos, [L.recuerdo]: recuerdo });
  await ponerLaAbierta(queda);
  await ponerSincro({ estado: "al-dia", ultima: new Date().toISOString() });
  pedirSincro(0);
}

/**
 * Desbloquear con la maestra. Si no abre la copia de aquí, **puede ser la nueva**,
 * cambiada en otro equipo (2.24.1): se prueba a entrar en la cuenta con ella.
 */
async function desbloquear(maestra: string): Promise<RespuestaDeCuenta> {
  const d = await datos();
  const texto = await local<string>(L.boveda);
  if (!d || !texto) throw new Error("Este navegador no tiene ninguna bóveda de cuenta");
  try {
    const b = await Boveda.abrir(texto, maestra, { purgar: true });
    await ponerLaAbierta(b);
    await ponerSincro({ estado: d.sesion ? "sincronizando" : "hay-que-entrar" });
    pedirSincro(0);
    return { ok: true };
  } catch (e) {
    if (!(e instanceof ErrorBoveda) || e.codigo !== "sin-ranura") throw e;
    try {
      return await entrar(d.correo, maestra, d.servidor);
    } catch (e2) {
      if (e2 instanceof ErrorDeRed) {
        throw new Error("Esa contraseña no abre la copia de este navegador. Si la cambiaste en otro equipo, hace falta conexión para ponerla al día");
      }
      throw e;
    }
  }
}

/** Salir de la cuenta en este navegador: cierra la sesión en el servidor y lo borra todo de aquí. */
async function salir(): Promise<void> {
  const d = await datos();
  const b = await laBoveda();
  if (d && b && d.sesion) {
    try {
      await new Cliente(d.servidor).cerrarSesion(await b.abrirSecreto(d.sesion));
    } catch {
      /* sin conexión o ya caducada: aquí se borra igual */
    }
  }
  await olvidarEntrada();
  await bloquear();
  await api.storage.local.remove([L.datos, L.boveda, L.base, L.recuerdo, L.llaves]);
  await api.storage.session.remove(S.sincro);
}

// ------------------------------------------------------------------ sincronizar

/**
 * Lo que la sincronización recuerda, **de la bóveda que esté abierta** (ADR 0050):
 * por qué versión del servidor va y con qué bytes se fundió la última vez.
 *
 * Va por bóveda y no puede ser de otra forma: con una sola memoria, conmutar
 * dejaría la versión de una bóveda apuntada como si fuera la de la otra, y la
 * siguiente pasada subiría encima de lo que no tocaba.
 */
function memoriaDe(ref: string, dueno = ""): Memoria {
  return {
    async cargar() {
      const recuerdo = await local<Recuerdo>(claveDeRecuerdo(ref, dueno));
      const base = (await local<string>(claveDeBase(ref, dueno))) ?? null;
      if (!recuerdo) return { recuerdo: { version: 0, serie: -1 }, base: null };
      // Sin base no se sabe qué hay subido: se fuerza la pasada, como en Go.
      return { recuerdo: base ? recuerdo : { version: recuerdo.version, serie: -1 }, base };
    },
    async guardar(r, base) {
      await api.storage.local.set({ [claveDeRecuerdo(ref, dueno)]: r, [claveDeBase(ref, dueno)]: base });
    },
  };
}

let temporizador: ReturnType<typeof setTimeout> | undefined;
let enMarcha: Promise<void> | null = null;
let otraVez = false;

/** Pide una pasada dentro de `ms`. La alarma de cada minuto es la red de seguridad si el trabajador se muere antes. */
export function pedirSincro(ms: number) {
  clearTimeout(temporizador);
  temporizador = setTimeout(() => {
    sincronizar().catch(() => {});
  }, ms);
}

/**
 * Una pasada, con el turno reservado: si ya hay una, se apunta otra para cuando
 * acabe **y se espera a las dos**. Volver en el acto diciendo «ya hay una» hacía
 * que el botón del panel contestara antes de que la pasada que lanzó abrirlo
 * terminara, con lo de antes (lo cazó la prueba con la extensión cargada).
 */
export function sincronizar(aunqueBorreMucho = false): Promise<void> {
  if (enMarcha) {
    otraVez = true;
    return enMarcha;
  }
  enMarcha = (async () => {
    try {
      do {
        otraVez = false;
        await unaPasada(aunqueBorreMucho);
        // **Solo la primera pasada** va con el permiso: lo dio una persona para
        // esta vez, no para siempre.
        aunqueBorreMucho = false;
      } while (otraVez);
    } finally {
      enMarcha = null;
    }
  })();
  return enMarcha;
}

// ------------------------------------------------------------------ compartir
//
// Lo mismo que hace la ventana (`internal/app/compartir.go`), con la misma regla:
// **lo que llega espera en el buzón**, y la huella se enseña siempre.

/** Con la bóveda abierta y la sesión lista, o se dice por qué no. */
async function conSesion<T>(hacer: (b: Boveda, cliente: Cliente, token: string) => Promise<T>): Promise<T> {
  const d = await datos();
  const b = await laBoveda();
  if (!d || !b) throw new Error("Abre la bóveda de la cuenta para hacer esto");
  if (!d.sesion) throw new Error("Vuelve a entrar en la cuenta para hacer esto");
  return hacer(b, new Cliente(d.servidor), await b.abrirSecreto(d.sesion));
}

/**
 * Publica las llaves de esta bóveda, **que es lo que hace falta para recibir**.
 *
 * Va con cada sincronización y no al entrar en «Compartir»: quien nunca ha
 * mandado nada tiene que poder recibir igual. Sin publicarlas, el servidor da a
 * quien manda una llave inventada —así es como no dice quién tiene cuenta— y el
 * sobre llega ilegible sin que ninguno de los dos entienda por qué.
 *
 * Se apunta la huella publicada para no repetir el envío cada minuto.
 */
async function publicarLasLlaves(b: Boveda, cliente: Cliente, token: string) {
  const i = await identidadDeSemilla(await b.semillaDeIdentidad());
  if ((await local<string>(L.llaves)) === i.huella) return i;
  await cliente.publicarLlaves(token, { suite: i.suite, cifrado: base64url(i.cifrado), firma: base64url(i.firma) });
  await api.storage.local.set({ [L.llaves]: i.huella });
  return i;
}

/** La identidad de esta bóveda, creándola si hace falta, y publicada. */
async function miIdentidad(): Promise<{ huella: string; cifrado: Uint8Array; firma: Uint8Array; suite: string }> {
  return conSesion(async (b, cliente, token) => {
    // Publicarlas es de cortesía: que el servidor no esté no puede impedir ver la
    // propia huella.
    try {
      return await publicarLasLlaves(b, cliente, token);
    } catch {
      return identidadDeSemilla(await b.semillaDeIdentidad());
    }
  });
}

async function huellaDe(correo: string): Promise<string> {
  return conSesion(async (b, cliente, token) => {
    void b;
    const l = await cliente.llavesDe(token, normalizarCorreo(correo));
    return huellaDeIdentidad(l.suite, desdeBase64(l.cifrado), desdeBase64(l.firma));
  });
}

/**
 * Manda una copia y **deja siempre la nota** (ADR 0043, B3), igual que
 * `MandarCopia` en Go: desde aquí no se puede saber si esa dirección tenía cuenta
 * —el servidor contesta lo mismo a propósito—, así que se anota en los dos casos.
 * Si la tenía, la nota caduca sin hacer nada; si no, es lo que hará salir la copia
 * cuando cree la suya.
 */
async function compartir(id: string, correo: string): Promise<void> {
  await conSesion(async (b, cliente, token) => {
    const e = b.ver(id);
    if (!e) throw new Error("Esa entrada ya no está");
    const c = normalizarCorreo(correo);
    const l = await cliente.llavesDe(token, c);
    const semilla = await b.semillaDeIdentidad();
    const para = { suite: l.suite, cifrado: desdeBase64(l.cifrado), firma: desdeBase64(l.firma), huella: "" };
    await cliente.mandar(token, c, await mandarEntrada(semilla, e, para));
    await b.anotarPendiente(id, c, await huellaDeIdentidad(l.suite, para.cifrado, para.firma));
  });
}

/**
 * Mira si alguna de las copias que esperan ya tiene a quién mandarse, y la manda.
 * Espejo de `repasarPendientes` en Go, y con los mismos frenos: **no cuenta como
 * actividad** y no dice nada por el panel.
 *
 * Va detrás de cada pasada de sincronización, que es cuando se sabe que hay red y
 * que la sesión vale.
 */
async function repasarPendientes(b: Boveda, cliente: Cliente, token: string): Promise<void> {
  for (const p of b.pendientes()) {
    const e = b.ver(p.entrada);
    if (!e) {
      await b.olvidarPendiente(p.id);
      continue;
    }
    const l = await cliente.llavesDe(token, p.correo);
    const cifrado = desdeBase64(l.cifrado);
    const firma = desdeBase64(l.firma);
    // **Que las llaves hayan cambiado es la señal**: mientras sean las inventadas
    // son siempre las mismas, así que cambiar solo puede ser que ya publica las suyas.
    if ((await huellaDeIdentidad(l.suite, cifrado, firma)) === p.huella) continue;
    const semilla = await b.semillaDeIdentidad();
    await cliente.mandar(token, p.correo, await mandarEntrada(semilla, e, { suite: l.suite, cifrado, firma, huella: "" }));
    await b.olvidarPendiente(p.id);
  }
}

async function verBuzon(): Promise<EnvioParaElPanel[]> {
  return conSesion(async (b, cliente, token) => {
    const semilla = await b.semillaDeIdentidad();
    const out: EnvioParaElPanel[] = [];
    for (const x of await cliente.buzon(token)) {
      try {
        const { entrada, de } = await abrirEnvio(semilla, x.sobre as Envio);
        out.push({ id: x.id, huella: de.huella, titulo: entrada.titulo, usuario: entrada.usuario ?? "" });
      } catch (e) {
        out.push({ id: x.id, huella: "", titulo: "", usuario: "", error: (e as Error).message });
      }
    }
    return out;
  });
}

async function aceptarDelBuzon(envio: string): Promise<void> {
  await conSesion(async (b, cliente, token) => {
    const semilla = await b.semillaDeIdentidad();
    for (const x of await cliente.buzon(token)) {
      if (x.id !== envio) continue;
      const { entrada, de } = await abrirEnvio(semilla, x.sobre as Envio);
      entrada.notas = entrada.notas ? `${entrada.notas}\n\nRecibida de la identidad ${de.huella}` : `Recibida de la identidad ${de.huella}`;
      await b.poner(entrada);
      await cliente.tirarDelBuzon(token, envio);
      pedirSincro(0);
      return;
    }
    throw new Error("Ese envío ya no está en el buzón");
  });
}

/**
 * Olvida una bóveda compartida a la que el dueño ha quitado el acceso (ADR 0053): su
 * copia cifrada, lo que la sincronización recordaba de ella, y salir si era la abierta.
 *
 * **Es la otra mitad del borrado de la ventana.** Esta copia vive en `storage.local` del
 * navegador y es otra: borrando solo el fichero de la aplicación, la bóveda de un
 * cliente se queda entera aquí.
 *
 * **Lo que esto no hace es tachar la fila** en la lista de la bóveda personal, que es
 * cosa de la ventana: tacharla desde aquí sería una segunda implementación del mismo
 * cambio sobre el mismo cuerpo, que es justo lo que las cruzadas existen para no tener.
 * Mientras no se abra la ventana, el panel enseña lo que ya enseña —que no hay acceso— y
 * la bóveda no está. Está dicho en `deuda.md`.
 *
 * **Nunca lanza.** Lo llama el `catch` de la pasada, y un fallo aquí no puede tapar el
 * estado que esa pasada acaba de dejar puesto.
 */
async function olvidarLaRevocada(dueno: string, ref: string): Promise<void> {
  try {
    if (dueno === "" || ref === "") return;
    // Primero salir, y después borrar: al revés queda la bóveda abierta encima de algo
    // que ya no está, que es el mismo orden que en la aplicación.
    if ((await duenoActivo()) === dueno && (await refActiva()) === ref) {
      await volverALaPersonal();
    }
    await api.storage.local.remove([claveDeBoveda(ref, dueno), claveDeBase(ref, dueno), claveDeRecuerdo(ref, dueno)]);
    avisarDeCambios();
  } catch {
    /* a la siguiente pasada, que llegará con el mismo 403 */
  }
}

async function unaPasada(aunqueBorreMucho = false): Promise<void> {
  const d = await datos();
  const b = await laBoveda();
  if (!d || !b) return;
  if (!d.sesion) {
    await ponerSincro({ estado: "hay-que-entrar", mensaje: "Vuelve a entrar en la cuenta para sincronizar" });
    return;
  }
  // **La sesión está sellada con la clave de la bóveda personal**, así que no la
  // abre la que esté activa: con un proyecto delante hay que ir a la personal a por
  // ella (ADR 0050). Con la personal abierta, `conLaPersonal` devuelve la misma.
  const token = await conLaPersonal((p) => p.abrirSecreto(d.sesion));
  const cliente = new Cliente(d.servidor);
  // **Antes de la pasada**, para que la identidad recién creada suba en ella. Que
  // falle no puede parar la sincronización: se reintenta a la siguiente.
  //
  // **Y con la bóveda personal, siempre**: la identidad para compartir y la
  // posesión son de la cuenta y viven en ella. Publicando las de un proyecto, el
  // servidor le daría a quien te manda una copia unas llaves que no son las tuyas y
  // el sobre llegaría cifrado hacia una bóveda que puedes tener cerrada.
  try {
    await conLaPersonal(async (p) => {
      await publicarLasLlaves(p, cliente, token);
      // Y de paso la posesión, por si en el servidor han rotado la pimienta (ADR
      // 0046). Misma forma y misma razón: quien lo necesita es el servidor.
      await cliente.refrescarPosesion(token, await p.posesion());
    });
  } catch {
    /* a la siguiente */
  }
  // **Cuál es la bóveda de esta pasada, antes del `try`**: el `catch` tiene que saber
  // cuál se ha ido para poder borrarla (ADR 0053), y dentro del `try` no se ve.
  const ref = await refActiva();
  const dueno = await duenoActivo();
  try {
    const r = await pasada(b, cliente, token, memoriaDe(ref, dueno), aunqueBorreMucho, ref, dueno);
    // **Y de paso, las copias que esperaban** (B3). Que falle no ensucia la
    // sincronización, que sí ha ido bien: se repasa en la siguiente.
    await repasarPendientes(b, cliente, token).catch(() => {});
    await ponerSincro({ estado: "al-dia", ultima: new Date().toISOString() });
    if (r.bajo) avisarDeCambios();
  } catch (e) {
    if (sesionCaducada(e)) {
      // Como la aplicación desde la 2.24.5: se olvida la sesión y se cierra.
      await api.storage.local.set({ [L.datos]: { ...d, sesion: "" } });
      await bloquear();
      await ponerSincro({
        estado: "hay-que-entrar",
        mensaje:
          "Se ha cerrado porque tu cuenta ya no reconoce este navegador: se olvidó desde otro equipo, se cambió la contraseña o la sesión caducó.",
      });
      return;
    }
    if (sinAcceso(e)) {
      // **Te han quitado el acceso a esta bóveda, o solo puedes verla** (ADR 0052).
      //
      // No se puede tratar como el 401: aquél olvida la sesión y cierra, porque la
      // sesión se perdió. Aquí la sesión está bien y la bóveda **propia** no tiene nada
      // que ver — cerrarla porque alguien te quitó el acceso a la suya sería castigarte
      // por lo que hizo otro.
      await ponerSincro({ estado: "sin-acceso", mensaje: (e as Error).message });
      // **Y si el servidor dice que ya no soy titular, la copia de este navegador se
      // va** (ADR 0053). Es la otra mitad del borrado: haciéndolo solo en la ventana, la
      // bóveda de un cliente se queda entera en el navegador de quien ya no trabaja con
      // él, que es exactamente lo que esto venía a evitar.
      //
      // `revocado` y no `sinAcceso`: el 403 también es «solo puedes ver» —y entonces no
      // hay nada que borrar— y además puede venir de un portero puesto delante.
      if (revocado(e)) await olvidarLaRevocada(dueno, ref);
      return;
    }
    if (e instanceof ErrorDeRed) {
      await ponerSincro({ estado: "sin-conexion", mensaje: "Sin conexión con el servidor de cuentas: se sube al volver" });
      return;
    }
    // La parada por muchos borrados tiene su propio estado, porque tiene salida: el
    // panel enseña «Juntarlo igual» (revisión del 2026-09-23).
    if (e instanceof ErrorBoveda && e.codigo === "muchos-borrados") {
      await ponerSincro({ estado: "muchos-borrados", mensaje: e.message });
      return;
    }
    await ponerSincro({ estado: "error", mensaje: (e as Error).message });
  }
}

/** Lo que haya que hacer cuando llega algo de otro equipo: lo pone el trabajador de fondo. */
let alCambiar: () => void = () => {};
export function alCambiarLaBoveda(f: () => void) {
  alCambiar = f;
}
function avisarDeCambios() {
  alCambiar();
}

/**
 * El reloj de cada minuto: cierra la bóveda si lleva el plazo sin tocarse, y si no,
 * sincroniza. **No cuenta como actividad**, claro.
 */
export async function tic(ahora = Date.now()): Promise<void> {
  if (!(await conCuenta())) return;
  // **La contraseña maestra a medias no espera a que alguien la lea.** Mientras se
  // va al correo a por el código, la entrada a medias la guarda con ella (ver
  // `Entrando`), y hasta la revisión del 2026-09-23 solo caducaba al leerla: si
  // nadie volvía, se quedaba los diez minutos y más. Aquí se barre.
  const aMedias = await sesion<{ hasta: number }>(S.entrando);
  if (aMedias && ahora > aMedias.hasta) await api.storage.session.remove(S.entrando);

  const ultima = await sesion<number>(S.actividad);
  if ((await sesion<string>(S.llave)) && ultima !== undefined && ahora - ultima >= MINUTOS_DE_BLOQUEO * 60_000) {
    // Antes de cerrar, lo pendiente se sube: cerrada ya no se puede.
    const b = await laBoveda();
    if (b && (await pendiente(b, memoriaDe(await refActiva(), await duenoActivo())))) await sincronizar().catch(() => {});
    await bloquear();
    return;
  }
  await sincronizar();
}

// ------------------------------------------------------------------ las dos puertas

/** Lo que pide el panel sobre la cuenta. **Solo se atiende desde el puerto del panel.** */
export async function atenderAlPanel(p: PeticionDeCuenta): Promise<RespuestaDeCuenta> {
  try {
    let r: RespuestaDeCuenta = { ok: true };
    switch (p.cuenta) {
      case "estado":
        await actividad();
        // **Abrir el panel pide una pasada**, como la aplicación al volver a su
        // ventana: quien lo abre suele venir de cambiar algo en otro equipo. No se
        // espera a que acabe; la lista la pinta el panel con lo que haya.
        if (await sesion<string>(S.llave)) pedirSincro(0);
        break;
      case "entrar":
        // **El servidor no viaja en el mensaje** (revisión del 2026-09-23): lo pone
        // la compilación, y la de pruebas apunta al servidor local. Aceptarlo desde
        // fuera era dejar que quien pudiera mandar un mensaje se llevara a otro
        // sitio la clave de acceso derivada de la maestra.
        r = await entrar(p.correo, p.maestra, (await datos())?.servidor || RAIZ_POR_DEFECTO);
        break;
      case "codigo":
        r = await confirmar(p.codigo);
        break;
      case "cancelar":
        await olvidarEntrada();
        break;
      case "desbloquear":
        r = await desbloquear(p.maestra);
        break;
      case "bloquear":
        await bloquear();
        break;
      case "salir":
        await salir();
        break;
      case "sincronizar":
        await sincronizar(p.igual === true);
        break;
      case "miHuella":
        await actividad();
        r = { ok: true, huella: (await miIdentidad()).huella };
        break;
      case "huellaDe":
        await actividad();
        r = { ok: true, huella: await huellaDe(p.correo) };
        break;
      case "compartir":
        await actividad();
        await compartir(p.id, p.correo);
        break;
      case "buzon":
        await actividad();
        r = { ok: true, buzon: await verBuzon() };
        break;
      case "aceptar":
        await actividad();
        await aceptarDelBuzon(p.envio);
        break;
      case "abrirProyecto":
        await actividad();
        await abrirProyecto(p.ref);
        break;
      case "abrirCompartida":
        await actividad();
        await abrirCompartida(p.dueno, p.ref);
        break;
      case "volverALaPersonal":
        await actividad();
        await volverALaPersonal();
        break;
      case "tirar":
        await actividad();
        await conSesion(async (b, cliente, token) => {
          void b;
          await cliente.tirarDelBuzon(token, p.envio);
        });
        break;
    }
    return { ...r, estado: await estado() };
  } catch (e) {
    return { ok: false, error: (e as Error).message, estado: await estado() };
  }
}

/**
 * Los verbos de siempre contestados con la bóveda de aquí. `dePersona` dice si lo
 * pide alguien —el panel— y cuenta como actividad; lo que llega de las páginas o
 * del refresco del icono, no.
 */
export async function atenderConCuenta(p: Peticion, dePersona: boolean): Promise<Respuesta> {
  if (dePersona) await actividad();
  const existe = (await local<string>(L.boveda)) !== undefined;
  return atender(p, { existe, boveda: await laBoveda(), soloVer: await soloPuedoVer() });
}

/**
 * Si lo que está abierto es una bóveda ajena de solo ver (ADR 0052).
 *
 * Se mira **aquí y no en la fuente** porque el permiso no está en la bóveda: está en mi
 * lista de compartidas, que vive en la personal, y la fuente no sabe de personales ni de
 * cuentas.
 *
 * Y se mira con cuidado al revés: si no se puede leer la lista —la bóveda se cerró entre
 * medias— se contesta que **sí** solo se puede ver. Equivocarse hacia ahí cuesta una
 * tarjeta que no sale; al otro lado cuesta escribir algo que no va a subir nunca.
 */
async function soloPuedoVer(): Promise<boolean> {
  const dueno = await duenoActivo();
  if (dueno === "") return false;
  const ref = await refActiva();
  try {
    return (await conLaPersonal(async (p) => p.laCompartida(dueno, ref)))?.permiso !== "editar";
  } catch {
    return true;
  }
}
