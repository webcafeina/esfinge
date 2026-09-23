/**
 * El cliente del servidor de cuentas en la extensión (ADR 0040), con lo que el
 * navegador necesita de `internal/cuenta/cliente.go`: entrar, confirmar con el
 * código, bajar y subir la bóveda y cerrar la sesión. Crear la cuenta, recuperarla
 * o borrarla sigue siendo cosa de la aplicación.
 *
 * Es **la conexión a internet de la extensión**, y solo va a un sitio: el servidor
 * de cuentas. Está dicho en el aviso del panel y en la política de privacidad.
 */

import { base64url, desdeBase64 } from "./esf1";
import { validarCoste, type ParametrosDeCuenta } from "./cuenta";

/** Lo pone la compilación de pruebas (`vite.fondo.config.ts`); en la de siempre va vacío. */
declare const __RAIZ_CUENTAS__: string;

export const RAIZ_POR_DEFECTO =
  (typeof __RAIZ_CUENTAS__ === "string" && __RAIZ_CUENTAS__) || "https://esfinge-cuentas.webcafeina.com";
const TAMANO_MAXIMO = 16 * 1024 * 1024;
const PLAZO_MS = 60_000;

export class ErrorDelServidor extends Error {
  constructor(
    readonly estado: number,
    mensaje: string,
    readonly version = 0,
  ) {
    super(mensaje);
  }
}

export class ErrorDeRed extends Error {}

export type Sesion = { cuenta?: string; sesion: string; dispositivo: string; confianza?: string };

export const sesionCaducada = (e: unknown) => e instanceof ErrorDelServidor && e.estado === 401;
export const conflicto = (e: unknown) => e instanceof ErrorDelServidor && e.estado === 412;
export const sinBoveda = (e: unknown) => e instanceof ErrorDelServidor && e.estado === 404;

/** La versión del `ETag`, fuerte o débil: Cloudflare lo debilita al comprimir. */
export function leerEtiqueta(v: string | null): number | null {
  const m = /^(?:W\/)?"?(\d{1,15})"?$/.exec((v ?? "").trim());
  return m ? Number(m[1]) : null;
}

/** Las llaves públicas de una cuenta, tal como viajan: base64url. */
export type LlavesEnLaRed = { suite: string; cifrado: string; firma: string };

/** Un sobre esperando en el buzón. */
export type EnvioEnBuzon = { id: string; momento: number; sobre: unknown };

export class Cliente {
  constructor(readonly raiz = RAIZ_POR_DEFECTO) {}

  private async pedir(metodo: string, ruta: string, opciones: { token?: string; cuerpo?: string; cabeceras?: Record<string, string> } = {}): Promise<Response> {
    const cab: Record<string, string> = { ...(opciones.cabeceras ?? {}) };
    if (opciones.cuerpo !== undefined) cab["Content-Type"] = "application/json";
    if (opciones.token) cab.Authorization = `Bearer ${opciones.token}`;
    const control = new AbortController();
    const plazo = setTimeout(() => control.abort(), PLAZO_MS);
    try {
      return await fetch(this.raiz.replace(/\/+$/, "") + ruta, {
        method: metodo,
        headers: cab,
        body: opciones.cuerpo,
        signal: control.signal,
        credentials: "omit",
        cache: "no-store",
      });
    } catch (e) {
      throw new ErrorDeRed(`No se ha podido hablar con el servidor de cuentas: ${(e as Error).message}`);
    } finally {
      clearTimeout(plazo);
    }
  }

  private async error(r: Response): Promise<ErrorDelServidor> {
    let cuerpo: { error?: string; version?: number } = {};
    try {
      cuerpo = (await r.json()) as typeof cuerpo;
    } catch {
      /* sin cuerpo que leer */
    }
    return new ErrorDelServidor(r.status, cuerpo.error || `El servidor de cuentas ha contestado ${r.status}`, cuerpo.version ?? 0);
  }

  private async json<T>(metodo: string, ruta: string, cuerpo?: unknown, token?: string): Promise<{ estado: number; datos: T }> {
    const r = await this.pedir(metodo, ruta, { token, cuerpo: cuerpo === undefined ? undefined : JSON.stringify(cuerpo) });
    if (r.status >= 300) throw await this.error(r);
    const texto = await r.text();
    try {
      return { estado: r.status, datos: (texto ? JSON.parse(texto) : {}) as T };
    } catch {
      throw new Error("El servidor de cuentas ha contestado algo que no se entiende");
    }
  }

  /** La sal y el coste para derivar la clave de acceso. El coste nunca por debajo del de siempre. */
  async prelogin(correo: string): Promise<{ sal: Uint8Array; argon2: ParametrosDeCuenta }> {
    const { datos } = await this.json<{ sal: string; argon2: ParametrosDeCuenta }>("POST", "/v1/prelogin", { correo });
    let sal: Uint8Array;
    try {
      sal = desdeBase64(datos.sal);
    } catch {
      throw new Error("El servidor ha devuelto una sal que no vale");
    }
    if (sal.length !== 16) throw new Error("El servidor ha devuelto una sal que no vale");
    validarCoste(datos.argon2);
    return { sal, argon2: datos.argon2 };
  }

  /** Entra: con la sesión si el equipo es de confianza, o con un reto si hace falta el código. */
  async entrar(correo: string, clave: Uint8Array, dispositivo: string, confianza: string): Promise<{ sesion?: Sesion; reto?: string }> {
    const cuerpo: Record<string, unknown> = { correo, claveDeAcceso: base64url(clave), dispositivo };
    if (confianza) cuerpo.confianza = confianza;
    const { estado, datos } = await this.json<Sesion & { reto?: string }>("POST", "/v1/sesion", cuerpo);
    if (estado === 202) return { reto: datos.reto ?? "" };
    return { sesion: datos };
  }

  async confirmar(reto: string, codigo: string, confiar: boolean): Promise<Sesion> {
    return (await this.json<Sesion>("POST", "/v1/sesion/codigo", { reto, codigo, confiar })).datos;
  }

  async cerrarSesion(token: string): Promise<void> {
    await this.json("DELETE", "/v1/sesion", undefined, token);
  }

  // -------------------------------------------------------------- compartir

  /** Publica las llaves públicas de esta cuenta, para que otros puedan mandarle. */
  async publicarLlaves(token: string, llaves: LlavesEnLaRed): Promise<void> {
    await this.json("PUT", "/v1/llaves", { llaves }, token);
  }

  /**
   * Las llaves de un correo.
   *
   * **Siempre contesta**, tenga cuenta o no: el servidor devuelve unas inventadas
   * pero fijas para las direcciones que no la tienen, y así preguntar no dice
   * quién está en Esfinge (ADR 0043).
   */
  async llavesDe(token: string, correo: string): Promise<LlavesEnLaRed> {
    return (await this.json<{ llaves: LlavesEnLaRed }>("POST", "/v1/llaves/de", { correo }, token)).datos.llaves;
  }

  /** Deja un sobre para ese correo. Contesta lo mismo exista o no esa cuenta. */
  async mandar(token: string, para: string, sobre: unknown): Promise<void> {
    await this.json("POST", "/v1/envios", { para, sobre }, token);
  }

  /** Lo que espera en el buzón, lo más nuevo primero. */
  async buzon(token: string): Promise<EnvioEnBuzon[]> {
    return (await this.json<{ envios: EnvioEnBuzon[] }>("GET", "/v1/buzon", undefined, token)).datos.envios ?? [];
  }

  /** Quita uno: lo mismo al aceptarlo que al tirarlo. */
  async tirarDelBuzon(token: string, id: string): Promise<void> {
    await this.json("DELETE", `/v1/buzon/${encodeURIComponent(id)}`, undefined, token);
  }

  /** La bóveda del servidor; `null` si no ha cambiado desde `siNoCoincide`. Lanza con 404 si no hay. */
  async bajar(token: string, siNoCoincide: number): Promise<{ datos: string; version: number } | null> {
    const cabeceras: Record<string, string> = {};
    if (siNoCoincide > 0) cabeceras["If-None-Match"] = `"${siNoCoincide}"`;
    const r = await this.pedir("GET", "/v1/boveda", { token, cabeceras });
    if (r.status === 304) return null;
    if (r.status !== 200) throw await this.error(r);
    const version = leerEtiqueta(r.headers.get("ETag"));
    if (version === null) throw new Error("El servidor no ha dicho qué versión de la bóveda es");
    const datos = await r.text();
    if (datos.length > TAMANO_MAXIMO) throw new Error("Lo que ha llegado del servidor es demasiado grande");
    return { datos, version };
  }

  /** Sube sobre `siCoincide` y devuelve la versión nueva. Un 412 es que otro subió en medio. */
  async subir(token: string, siCoincide: number, datos: string): Promise<number> {
    const r = await this.pedir("PUT", "/v1/boveda", { token, cuerpo: datos, cabeceras: { "If-Match": `"${siCoincide}"` } });
    if (r.status !== 200) throw await this.error(r);
    const j = (await r.json()) as { version?: number };
    return j.version ?? 0;
  }
}
