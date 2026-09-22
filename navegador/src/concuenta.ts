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
import { Cliente, ErrorDeRed, RAIZ_POR_DEFECTO, sesionCaducada, sinBoveda, type Sesion } from "./nucleo/cliente";
import { derivarAcceso, normalizarCorreo } from "./nucleo/cuenta";
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
} as const;
const S = {
  llave: "cuenta-llave",
  actividad: "cuenta-actividad",
  sincro: "cuenta-sincro",
  entrando: "cuenta-entrando",
} as const;

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
  estado: "apagada" | "sincronizando" | "al-dia" | "sin-conexion" | "hay-que-entrar" | "error";
  ultima?: string;
  mensaje?: string;
};

export type EstadoDeCuenta = {
  modo: "local" | "cuenta";
  correo?: string;
  abierta: boolean;
  /** Hay una entrada a medias esperando el código del correo. */
  codigoPendiente: boolean;
  sincro: EstadoSincro;
  /** Minutos sin tocar nada antes de cerrarse sola. */
  bloqueo: number;
};

/** Lo que el panel —y solo el panel— le puede pedir a la cuenta. */
export type PeticionDeCuenta =
  | { cuenta: "estado" }
  | { cuenta: "entrar"; correo: string; maestra: string; servidor?: string }
  | { cuenta: "codigo"; codigo: string }
  /** Deja la entrada a medias: se vuelve a empezar. */
  | { cuenta: "cancelar" }
  | { cuenta: "desbloquear"; maestra: string }
  | { cuenta: "bloquear" }
  | { cuenta: "salir" }
  | { cuenta: "sincronizar" };

export type RespuestaDeCuenta = {
  ok: boolean;
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
  const texto = await local<string>(L.boveda);
  if (!llave || !texto) return null;
  try {
    abierta = await Boveda.abrirConClave(texto, llave);
    conectar(abierta);
    return abierta;
  } catch {
    await api.storage.session.remove(S.llave);
    return null;
  }
}

/** Cada guardado va a `storage.local` y pide subir, con una espera para juntar los seguidos. */
function conectar(b: Boveda) {
  b.alGuardar = (texto) => {
    api.storage.local.set({ [L.boveda]: texto }).catch(() => {});
    pedirSincro(ESPERA_TRAS_GUARDAR);
  };
}

async function ponerLaAbierta(b: Boveda) {
  abierta = b;
  conectar(b);
  await api.storage.local.set({ [L.boveda]: b.documento() });
  await api.storage.session.set({ [S.llave]: b._llaveParaLaSesion(), [S.actividad]: Date.now() });
}

export async function bloquear(): Promise<void> {
  abierta?.cerrar();
  abierta = null;
  await api.storage.session.remove([S.llave, S.actividad]);
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
  return {
    modo: d ? "cuenta" : "local",
    correo: d?.correo,
    abierta: b !== null,
    codigoPendiente: (await entrando()) !== null,
    sincro,
    bloqueo: MINUTOS_DE_BLOQUEO,
  };
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
      // Otra bóveda, o una que no se deja fundir: manda la de la cuenta. Lo de aquí
      // era una copia de esa misma cuenta, no algo que solo existiera aquí.
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
  await api.storage.local.remove([L.datos, L.boveda, L.base, L.recuerdo]);
  await api.storage.session.remove(S.sincro);
}

// ------------------------------------------------------------------ sincronizar

const memoria: Memoria = {
  async cargar() {
    const recuerdo = await local<Recuerdo>(L.recuerdo);
    const base = (await local<string>(L.base)) ?? null;
    if (!recuerdo) return { recuerdo: { version: 0, serie: -1 }, base: null };
    // Sin base no se sabe qué hay subido: se fuerza la pasada, como en Go.
    return { recuerdo: base ? recuerdo : { version: recuerdo.version, serie: -1 }, base };
  },
  async guardar(r, base) {
    await api.storage.local.set({ [L.recuerdo]: r, [L.base]: base });
  },
};

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
export function sincronizar(): Promise<void> {
  if (enMarcha) {
    otraVez = true;
    return enMarcha;
  }
  enMarcha = (async () => {
    try {
      do {
        otraVez = false;
        await unaPasada();
      } while (otraVez);
    } finally {
      enMarcha = null;
    }
  })();
  return enMarcha;
}

async function unaPasada(): Promise<void> {
  const d = await datos();
  const b = await laBoveda();
  if (!d || !b) return;
  if (!d.sesion) {
    await ponerSincro({ estado: "hay-que-entrar", mensaje: "Vuelve a entrar en la cuenta para sincronizar" });
    return;
  }
  const token = await b.abrirSecreto(d.sesion);
  try {
    const r = await pasada(b, new Cliente(d.servidor), token, memoria);
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
    if (e instanceof ErrorDeRed) {
      await ponerSincro({ estado: "sin-conexion", mensaje: "Sin conexión con el servidor de cuentas: se sube al volver" });
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
  const ultima = await sesion<number>(S.actividad);
  if ((await sesion<string>(S.llave)) && ultima !== undefined && ahora - ultima >= MINUTOS_DE_BLOQUEO * 60_000) {
    // Antes de cerrar, lo pendiente se sube: cerrada ya no se puede.
    const b = await laBoveda();
    if (b && (await pendiente(b, memoria))) await sincronizar().catch(() => {});
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
        r = await entrar(p.correo, p.maestra, p.servidor || (await datos())?.servidor || RAIZ_POR_DEFECTO);
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
        await sincronizar();
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
  return atender(p, { existe, boveda: await laBoveda() });
}
