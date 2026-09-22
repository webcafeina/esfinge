/**
 * La bóveda en TypeScript (ADR 0040): la segunda implementación de
 * `internal/boveda`, con `docs/formato-boveda.md` como contrato.
 *
 * **No tiene fichero.** En la extensión, lo que Go escribe en el disco va a
 * `storage.local`, y eso lo decide quien la use: `guardar()` devuelve el documento
 * y avisa a `alGuardar`. Por lo demás hace lo mismo que Go y con las mismas
 * frases: abrir con la maestra o con la clave de recuperación, comprobar el sello,
 * guardar sellando otra vez, la papelera con su plazo, las lápidas, los sitios
 * excluidos, preparar la subida y la prueba de posesión.
 *
 * Lo que queda fuera por ahora, porque desde el navegador no se hace: crear una
 * bóveda para el disco de un equipo, cambiar la contraseña y rotar la clave de
 * recuperación. Eso sigue siendo cosa de la aplicación.
 */

import { canonico, compararComoGo, huella, type ValorJSON } from "./canon";
import {
  abrirTexto,
  azarDe,
  base64url,
  desdeBase64,
  PERFIL_INTERACTIVO,
  PERFIL_LLAVE,
  sellarTexto,
} from "./esf1";
import { entradaAJSON, entradaDesde, rfc3339, sinSecretos, coincide, copiar, type Entrada } from "./entrada";
import { ERR_CHECKSUM, normalizar, nuevaRecuperacion, pareceRecuperacion } from "./recuperacion";

export const FORMATO = 1;
export const RANURA_MAESTRA = "maestra";
export const RANURA_RECUPERACION = "recuperacion";
/** Las que abren la bóveda solo en un equipo y no se suben nunca. */
export const RANURAS_LOCALES = new Set(["llavero-del-sistema", "pin"]);

export const PLAZO_PAPELERA_MS = 30 * 24 * 3600 * 1000;
export const PLAZO_LAPIDAS_MS = 180 * 24 * 3600 * 1000;

const MARCA = "bóveda";
const AVISO =
  "Bóveda de Esfinge. Las contraseñas van cifradas; esto de fuera solo dice cómo abrirlas. No la edites a mano.";

export type CodigoDeError =
  | "no-es-boveda"
  | "formato-nuevo"
  | "manipulada"
  | "sin-ranura"
  | "checksum"
  | "cerrada"
  | "otra-boveda"
  | "retroceso"
  | "muchos-borrados"
  | "papelera";

/** Las mismas frases que Go: el mismo fallo tiene que decir lo mismo en la ventana y en el panel. */
const MENSAJES: Record<CodigoDeError, string> = {
  "no-es-boveda": "Esto no parece una bóveda de Esfinge",
  "formato-nuevo": "Esta bóveda la escribió una versión de Esfinge más nueva; actualiza antes de tocarla",
  manipulada: "La bóveda no cuadra por dentro: alguien ha editado el fichero a mano o está a medias",
  "sin-ranura": "Esa llave no abre esta bóveda",
  checksum: ERR_CHECKSUM,
  cerrada: "La bóveda está cerrada",
  "otra-boveda": "Esa bóveda no es la de esta cuenta",
  retroceso: "El servidor ha devuelto una versión de la bóveda que no cuadra",
  "muchos-borrados": "Juntar los cambios de otro equipo borraría más de la mitad de la bóveda",
  papelera: "Esa entrada no está en la papelera",
};

export class ErrorBoveda extends Error {
  constructor(
    readonly codigo: CodigoDeError,
    mensaje?: string,
  ) {
    super(mensaje ?? MENSAJES[codigo]);
  }
}

export type Sobre = { tipo: string; creado: string; contenedor: string; codificacion?: string };

export type Documento = {
  esfinge: string;
  aviso: string;
  formato: number;
  id: string;
  serie: number;
  cambiada: string;
  sobres: Sobre[];
  sello: string;
  cuerpo: string;
};

export type Sello = { id: string; serie: number; huellas: Record<string, string>; sincro?: number; cuerpo: string };

export type Contenido = {
  entradas: Entrada[];
  sitiosExcluidos?: string[];
  lapidas?: Record<string, string>;
  /** Las secciones que esta versión no conoce. */
  extra?: Record<string, ValorJSON>;
};

const CLAVES_DEL_CONTENIDO = new Set(["entradas", "sitiosExcluidos", "lapidas"]);

// ------------------------------------------------------------------ el reloj

/** `ahora` se puede parar en las pruebas, como en Go. */
let reloj = () => new Date();
export function relojParaPruebas(f: (() => Date) | null): void {
  reloj = f ?? (() => new Date());
}
export const ahora = () => reloj();

// ------------------------------------------------------------------ piezas sueltas

function azarHex(): string {
  return Array.from(azarDe(16), (b) => b.toString(16).padStart(2, "0")).join("");
}

const utf8 = new TextEncoder();
const deUtf8 = new TextDecoder();

/** Lee el JSON de fuera y comprueba que es una bóveda que se entiende. */
export function leerDocumento(texto: string): Documento {
  let o: Record<string, unknown>;
  try {
    o = JSON.parse(texto) as Record<string, unknown>;
  } catch {
    throw new ErrorBoveda("no-es-boveda");
  }
  if (!o || typeof o !== "object" || o.esfinge !== MARCA) throw new ErrorBoveda("no-es-boveda");
  const formato = typeof o.formato === "number" ? o.formato : 0;
  if (formato > FORMATO) throw new ErrorBoveda("formato-nuevo");
  const t = (v: unknown) => (typeof v === "string" ? v : "");
  const sobres = Array.isArray(o.sobres)
    ? (o.sobres as Record<string, unknown>[]).map((s) => {
        const x: Sobre = { tipo: t(s?.tipo), creado: t(s?.creado), contenedor: t(s?.contenedor) };
        if (t(s?.codificacion)) x.codificacion = t(s.codificacion);
        return x;
      })
    : [];
  return {
    esfinge: MARCA,
    aviso: t(o.aviso),
    formato,
    id: t(o.id),
    serie: typeof o.serie === "number" ? o.serie : 0,
    cambiada: t(o.cambiada),
    sobres,
    sello: t(o.sello),
    cuerpo: t(o.cuerpo),
  };
}

/** El identificador de una bóveda sin abrirla. */
export function idDe(texto: string): string {
  return leerDocumento(texto).id;
}

export function contenidoDesde(crudo: unknown): Contenido {
  if (!crudo || typeof crudo !== "object" || Array.isArray(crudo)) throw new ErrorBoveda("manipulada");
  const o = crudo as Record<string, unknown>;
  const c: Contenido = { entradas: [] };
  if (o.entradas !== null && o.entradas !== undefined) {
    if (!Array.isArray(o.entradas)) throw new ErrorBoveda("manipulada");
    c.entradas = o.entradas.map(entradaDesde);
  }
  if (Array.isArray(o.sitiosExcluidos) && o.sitiosExcluidos.length > 0) {
    c.sitiosExcluidos = o.sitiosExcluidos.map((s) => (typeof s === "string" ? s : ""));
  }
  if (o.lapidas && typeof o.lapidas === "object" && !Array.isArray(o.lapidas)) {
    const l: Record<string, string> = {};
    for (const [k, v] of Object.entries(o.lapidas as Record<string, unknown>)) l[k] = typeof v === "string" ? v : "";
    if (Object.keys(l).length > 0) c.lapidas = l;
  }
  const extra: Record<string, ValorJSON> = {};
  for (const [k, v] of Object.entries(o)) if (!CLAVES_DEL_CONTENIDO.has(k)) extra[k] = v as ValorJSON;
  if (Object.keys(extra).length > 0) c.extra = extra;
  return c;
}

/** El contenido como JSON, como `json.Marshal(contenido)` en Go: vacíos fuera, lo desconocido dentro. */
export function contenidoAJSON(c: Contenido): Record<string, ValorJSON> {
  const out: Record<string, ValorJSON> = { entradas: c.entradas.map(entradaAJSON) };
  if (c.sitiosExcluidos && c.sitiosExcluidos.length > 0) out.sitiosExcluidos = c.sitiosExcluidos;
  if (c.lapidas && Object.keys(c.lapidas).length > 0) out.lapidas = c.lapidas;
  for (const [k, v] of Object.entries(c.extra ?? {})) if (!CLAVES_DEL_CONTENIDO.has(k)) out[k] = v;
  return out;
}

/** Abre el sello y el cuerpo con la clave de bóveda y comprueba que cuadran con lo de fuera. */
export async function desempaquetar(doc: Documento, llave: string): Promise<{ sel: Sello; cont: Contenido }> {
  let sel: Sello;
  let cont: Contenido;
  try {
    const s = JSON.parse(deUtf8.decode(await abrirTexto(doc.sello, llave))) as Record<string, unknown>;
    sel = {
      id: typeof s.id === "string" ? s.id : "",
      serie: typeof s.serie === "number" ? s.serie : 0,
      huellas: (s.huellas && typeof s.huellas === "object" ? s.huellas : {}) as Record<string, string>,
      cuerpo: typeof s.cuerpo === "string" ? s.cuerpo : "",
    };
    if (typeof s.sincro === "number" && s.sincro !== 0) sel.sincro = s.sincro;
    cont = contenidoDesde(JSON.parse(deUtf8.decode(await abrirTexto(doc.cuerpo, llave))));
  } catch {
    throw new ErrorBoveda("manipulada");
  }
  await coherente(doc, sel);
  return { sel, cont };
}

/**
 * Compara lo de dentro con lo de fuera. El JSON de fuera no va autenticado, así
 * que quien pueda escribirlo puede quitar una ranura o volver a un cuerpo viejo:
 * esto no lo impide, lo **detecta**.
 */
async function coherente(doc: Documento, sel: Sello): Promise<void> {
  if (sel.id !== doc.id || sel.serie !== doc.serie) throw new ErrorBoveda("manipulada");
  if (sel.cuerpo !== (await huella(doc.cuerpo))) throw new ErrorBoveda("manipulada");
  const tiene = new Set<string>();
  for (const s of doc.sobres) {
    tiene.add(s.tipo);
    const esperada = sel.huellas[s.tipo];
    if (esperada === undefined) continue; // de una versión más nueva: no impide abrir
    if ((await huella(s.contenedor)) !== esperada) throw new ErrorBoveda("manipulada");
  }
  for (const tipo of Object.keys(sel.huellas)) if (!tiene.has(tipo)) throw new ErrorBoveda("manipulada");
}

/** Sella la clave de bóveda con una llave: un sobre, con el coste de siempre. */
async function envolver(tipo: string, clave: string, llave: string, cuando: string): Promise<Sobre> {
  let k = clave;
  if (tipo === RANURA_RECUPERACION) {
    const n = await normalizar(clave);
    if (!n) throw new ErrorBoveda("checksum");
    k = n;
  }
  const s: Sobre = { tipo, creado: cuando, contenedor: await sellarTexto(utf8.encode(llave), k, PERFIL_INTERACTIVO) };
  if (tipo === RANURA_RECUPERACION) s.codificacion = "crockford32-v1";
  return s;
}

/** La prueba de que se tiene la clave de bóveda: HKDF de sus 32 bytes, como `PosesionDeLlave` en Go. */
export async function posesionDeLlave(llave: string): Promise<Uint8Array> {
  const bruta = desdeBase64(llave);
  const k = await crypto.subtle.importKey("raw", new Uint8Array(bruta), "HKDF", false, ["deriveBits"]);
  bruta.fill(0);
  const bits = await crypto.subtle.deriveBits(
    { name: "HKDF", hash: "SHA-256", salt: new Uint8Array(0), info: utf8.encode("esfinge/cuenta/posesion/v1") },
    k,
    256,
  );
  return new Uint8Array(bits);
}

// ------------------------------------------------------------------ la bóveda

export class Boveda {
  private constructor(
    private doc: Documento,
    private sel: Sello,
    private cont: Contenido,
    private llave: string | null,
    readonly soloLectura: boolean,
  ) {}

  /** Se llama tras cada guardado con el documento nuevo. Quien use la bóveda decide dónde vive. */
  alGuardar: ((documento: string) => void) | null = null;
  private cuerpoSucio = false;

  /** Una bóveda nueva y su clave de recuperación, que no se vuelve a ver. */
  static async crear(maestra: string): Promise<{ boveda: Boveda; recuperacion: string }> {
    if (!maestra.trim()) throw new Error("La contraseña maestra no puede estar vacía");
    const llave = base64url(azarDe(32));
    const recuperacion = await nuevaRecuperacion();
    const cuando = rfc3339(ahora());
    const doc: Documento = {
      esfinge: MARCA,
      aviso: AVISO,
      formato: FORMATO,
      id: azarHex(),
      serie: 0,
      cambiada: cuando,
      sobres: [
        await envolver(RANURA_MAESTRA, maestra, llave, cuando),
        await envolver(RANURA_RECUPERACION, recuperacion, llave, cuando),
      ],
      sello: "",
      cuerpo: "",
    };
    const b = new Boveda(doc, { id: doc.id, serie: 0, huellas: {}, cuerpo: "" }, { entradas: [] }, llave, false);
    b.cuerpoSucio = true;
    return { boveda: b, recuperacion };
  }

  /**
   * Abre con la maestra o con la clave de recuperación. Con `purgar`, lo que lleva
   * más de treinta días en la papelera y las lápidas caducadas se van, y si se va
   * algo se guarda: es lo que hace Go al abrir el fichero de un equipo. Lo que baja
   * del servidor se abre sin purgar.
   */
  static async abrir(texto: string, tecleado: string, opciones: { purgar?: boolean } = {}): Promise<Boveda> {
    const doc = leerDocumento(texto);
    const norm = await normalizar(tecleado);
    const candidatas = norm ? [norm, tecleado] : [tecleado];
    let llave: string | null = null;
    fuera: for (const s of doc.sobres) {
      for (const c of candidatas) {
        try {
          llave = deUtf8.decode(await abrirTexto(s.contenedor, c));
          break fuera;
        } catch {
          // Siguiente.
        }
      }
    }
    if (llave === null) {
      // «Te has equivocado al copiarla» no es «no abre», y se dice al final: una
      // maestra que empiece por ESF es rara pero legítima.
      if (!norm && pareceRecuperacion(tecleado)) throw new ErrorBoveda("checksum");
      throw new ErrorBoveda("sin-ranura");
    }
    return Boveda.conLlave(doc, llave, opciones);
  }

  /** Abre con la clave de bóveda de otra abierta: la misma bóveda, con otra contraseña en sus sobres. */
  static async abrirConLaLlaveDe(texto: string, otra: Boveda): Promise<Boveda> {
    const doc = leerDocumento(texto);
    if (doc.id !== otra.doc.id) throw new ErrorBoveda("otra-boveda");
    return Boveda.conLlave(doc, otra.clave(), {});
  }

  /**
   * Abre con la clave de la bóveda, sin contraseña: la que la extensión guarda en
   * `storage.session` mientras está abierta, para que el trabajador de fondo, que
   * se duerme cada pocos minutos, la recupere al despertar.
   */
  static async abrirConClave(texto: string, llave: string): Promise<Boveda> {
    return Boveda.conLlave(leerDocumento(texto), llave, {});
  }

  /**
   * @internal La clave de la bóveda, para `storage.session` y para nada más: es lo
   * que abre la bóveda sin contraseña, y vive en memoria hasta cerrar el navegador.
   */
  _llaveParaLaSesion(): string {
    return this.clave();
  }

  private static async conLlave(doc: Documento, llave: string, opciones: { purgar?: boolean }): Promise<Boveda> {
    const { sel, cont } = await desempaquetar(doc, llave);
    const b = new Boveda(doc, sel, cont, llave, doc.formato < FORMATO);
    if (opciones.purgar) {
      const t = ahora().getTime();
      if (b.purgarPapelera(new Date(t - PLAZO_PAPELERA_MS)) + b.purgarLapidas(new Date(t - PLAZO_LAPIDAS_MS)) > 0) {
        b.cuerpoSucio = true;
        await b.guardar();
      }
    }
    return b;
  }

  private clave(): string {
    if (this.llave === null) throw new ErrorBoveda("cerrada");
    return this.llave;
  }

  get abierta(): boolean {
    return this.llave !== null;
  }

  /** Olvida la clave y el contenido. No hay vuelta atrás: se abre otra vez. */
  cerrar(): void {
    this.llave = null;
    this.cont = { entradas: [] };
  }

  get id(): string {
    return this.doc.id;
  }

  get serie(): number {
    return this.doc.serie;
  }

  /** El documento tal como está ahora, en texto. */
  documento(): string {
    return JSON.stringify(this.doc, null, 2) + "\n";
  }

  /**
   * **Lo que cambia la bóveda pasa de uno en uno**, y hace falta escribirlo porque
   * aquí no hay cerrojos: entre dos `await` se cuela cualquiera. Dos guardados
   * cruzados acababan con **la misma serie**, y el segundo cambio no se subía nunca
   * porque la sincronización lo daba por subido; y una fusión que tarda —bajar,
   * descifrar, fundir— pisaba lo que la tarjeta de la página hubiera guardado en
   * medio. En Go lo evita el cerrojo de la bóveda; aquí, esta cola.
   */
  private cola: Promise<unknown> = Promise.resolve();

  /** @internal Ejecuta `f` cuando haya terminado todo lo anterior, y lo siguiente espera a `f`. */
  _exclusivo<T>(f: () => Promise<T>): Promise<T> {
    const turno = this.cola.then(f, f);
    this.cola = turno.catch(() => undefined);
    return turno;
  }

  /** Sella y devuelve el documento nuevo. Solo cambia el cuerpo si han cambiado las entradas. */
  guardar(): Promise<string> {
    return this._exclusivo(() => this._guardarSinCola());
  }

  /** @internal Guardar, ya dentro de la cola. */
  async _guardarSinCola(): Promise<string> {
    const llave = this.clave();
    if (this.soloLectura) throw new ErrorBoveda("formato-nuevo");
    const doc = { ...this.doc, serie: this.doc.serie + 1, cambiada: rfc3339(new Date()) };
    if (this.cuerpoSucio || !doc.cuerpo) {
      doc.cuerpo = await sellarTexto(utf8.encode(JSON.stringify(contenidoAJSON(this.cont))), llave, PERFIL_LLAVE);
    }
    const sel: Sello = { id: doc.id, serie: doc.serie, huellas: {}, cuerpo: await huella(doc.cuerpo) };
    for (const s of doc.sobres) sel.huellas[s.tipo] = await huella(s.contenedor);
    if (this.sel.sincro) sel.sincro = this.sel.sincro;
    doc.sello = await sellarTexto(utf8.encode(JSON.stringify(ordenSello(sel))), llave, PERFIL_LLAVE);
    this.doc = doc;
    this.sel = sel;
    this.cuerpoSucio = false;
    const texto = this.documento();
    this.alGuardar?.(texto);
    return texto;
  }

  /**
   * La bóveda tal como se sube como `version`: sin las ranuras de este equipo y con
   * la versión sellada dentro. **No cambia la de aquí.**
   */
  prepararSubida(version: number): Promise<{ texto: string; serie: number }> {
    return this._exclusivo(() => this.prepararSubidaSinCola(version));
  }

  private async prepararSubidaSinCola(version: number): Promise<{ texto: string; serie: number }> {
    const llave = this.clave();
    if (this.cuerpoSucio) throw new Error("La bóveda tiene cambios sin guardar");
    const sobres = this.doc.sobres.filter((s) => !RANURAS_LOCALES.has(s.tipo));
    const sel: Sello = { id: this.doc.id, serie: this.doc.serie, huellas: {}, cuerpo: await huella(this.doc.cuerpo) };
    for (const s of sobres) sel.huellas[s.tipo] = await huella(s.contenedor);
    if (version) sel.sincro = version;
    const doc: Documento = {
      ...this.doc,
      sobres,
      sello: await sellarTexto(utf8.encode(JSON.stringify(ordenSello(sel))), llave, PERFIL_LLAVE),
    };
    // La serie sale de lo mismo que se sube: leída aparte, un guardado en medio se
    // daría por subido sin estarlo (como `PrepararSubida` en Go).
    return { texto: JSON.stringify(doc, null, 2) + "\n", serie: doc.serie };
  }

  /** La prueba de posesión para el servidor. */
  posesion(): Promise<Uint8Array> {
    return posesionDeLlave(this.clave());
  }

  /** Cifra algo pequeño con la clave de bóveda (la sesión de la cuenta). */
  async sellarSecreto(claro: string): Promise<string> {
    return sellarTexto(utf8.encode(claro), this.clave(), PERFIL_LLAVE);
  }

  async abrirSecreto(sellado: string): Promise<string> {
    return deUtf8.decode(await abrirTexto(sellado, this.clave()));
  }

  // ---------------------------------------------------------------- entradas

  /** Las que encajan, **sin secretos** y sin la papelera. */
  buscar(q: string): Entrada[] {
    this.clave();
    return this.cont.entradas.filter((e) => !e.papelera && coincide(e, q)).map(sinSecretos);
  }

  /** Una entrada entera, con sus secretos. */
  ver(id: string): Entrada | null {
    this.clave();
    const e = this.cont.entradas.find((x) => x.id === id);
    return e ? copiar(e) : null;
  }

  /** Añade o sustituye una entrada y guarda. La revisión la pone la bóveda. */
  poner(nueva: Entrada): Promise<Entrada> {
    return this._exclusivo(() => this.ponerSinCola(nueva));
  }

  private async ponerSinCola(nueva: Entrada): Promise<Entrada> {
    this.clave();
    const e = copiar(nueva);
    const cuando = rfc3339(new Date());
    if (!e.id) {
      e.id = azarHex();
      e.creada = cuando;
    }
    e.cambiada = cuando;
    if (!e.tipo) e.tipo = "credencial";
    e.revision = 1;
    const i = this.cont.entradas.findIndex((x) => x.id === e.id);
    if (i >= 0) {
      if (!e.creada) e.creada = this.cont.entradas[i].creada;
      e.revision = (this.cont.entradas[i].revision ?? 0) + 1;
      this.cont.entradas[i] = e;
    } else {
      this.cont.entradas.push(e);
    }
    this.cuerpoSucio = true;
    await this._guardarSinCola();
    return copiar(e);
  }

  /** A la papelera, entera (ADR 0026). */
  borrar(id: string): Promise<void> {
    return this._exclusivo(() => this.borrarSinCola(id));
  }

  private async borrarSinCola(id: string): Promise<void> {
    this.clave();
    const e = this.cont.entradas.find((x) => x.id === id && !x.papelera);
    if (!e) return;
    e.papelera = true;
    e.borradaEn = rfc3339(ahora());
    e.revision = (e.revision ?? 0) + 1;
    this.cuerpoSucio = true;
    await this._guardarSinCola();
  }

  restaurar(id: string): Promise<void> {
    return this._exclusivo(() => this.restaurarSinCola(id));
  }

  private async restaurarSinCola(id: string): Promise<void> {
    this.clave();
    const e = this.cont.entradas.find((x) => x.id === id && x.papelera);
    if (!e) throw new Error("Esa entrada ya no está en la papelera");
    delete e.papelera;
    delete e.borradaEn;
    e.revision = (e.revision ?? 0) + 1;
    this.cuerpoSucio = true;
    await this._guardarSinCola();
  }

  /** Solo desde la papelera, y deja su lápida. */
  borrarDelTodo(id: string): Promise<void> {
    return this._exclusivo(() => this.borrarDelTodoSinCola(id));
  }

  private async borrarDelTodoSinCola(id: string): Promise<void> {
    this.clave();
    const i = this.cont.entradas.findIndex((x) => x.id === id && x.papelera);
    if (i < 0) throw new ErrorBoveda("papelera");
    this.cont.entradas.splice(i, 1);
    this.enterrar(id, ahora());
    this.cuerpoSucio = true;
    await this._guardarSinCola();
  }

  papelera(): Entrada[] {
    this.clave();
    return this.cont.entradas
      .filter((e) => e.papelera)
      .map(sinSecretos)
      .sort((a, b) => ((a.borradaEn ?? "") < (b.borradaEn ?? "") ? 1 : (a.borradaEn ?? "") > (b.borradaEn ?? "") ? -1 : 0));
  }

  cuantas(): number {
    return this.cont.entradas.filter((e) => !e.papelera).length;
  }

  private purgarPapelera(limite: Date): number {
    const corte = rfc3339(limite);
    const antes = this.cont.entradas.length;
    this.cont.entradas = this.cont.entradas.filter((e) => {
      // Sin fecha no se toca: tirar datos por no saber cuándo se borraron, no.
      if (e.papelera && e.borradaEn && e.borradaEn < corte) {
        this.enterrar(e.id, ahora());
        return false;
      }
      return true;
    });
    return antes - this.cont.entradas.length;
  }

  private enterrar(id: string, cuando: Date): void {
    this.cont.lapidas ??= {};
    this.cont.lapidas[id] = rfc3339(cuando);
  }

  private purgarLapidas(limite: Date): number {
    const corte = rfc3339(limite);
    let n = 0;
    for (const [id, cuando] of Object.entries(this.cont.lapidas ?? {})) {
      if (cuando < corte) {
        delete this.cont.lapidas![id];
        n++;
      }
    }
    if (this.cont.lapidas && Object.keys(this.cont.lapidas).length === 0) delete this.cont.lapidas;
    return n;
  }

  // ---------------------------------------------------------------- sitios excluidos

  excluir(dominio: string): Promise<void> {
    return this._exclusivo(() => this.excluirSinCola(dominio));
  }

  private async excluirSinCola(dominio: string): Promise<void> {
    this.clave();
    const d = dominio.trim().toLowerCase();
    if (!d) throw new Error("Hace falta un sitio que excluir");
    const l = this.cont.sitiosExcluidos ?? [];
    if (l.includes(d)) return;
    this.cont.sitiosExcluidos = [...l, d].sort(compararComoGo);
    this.cuerpoSucio = true;
    await this._guardarSinCola();
  }

  quitarExclusion(dominio: string): Promise<void> {
    return this._exclusivo(() => this.quitarExclusionSinCola(dominio));
  }

  private async quitarExclusionSinCola(dominio: string): Promise<void> {
    this.clave();
    const d = dominio.trim().toLowerCase();
    const l = this.cont.sitiosExcluidos ?? [];
    const quedan = l.filter((x) => x !== d);
    if (quedan.length === l.length) throw new Error("Ese sitio no estaba excluido");
    this.cont.sitiosExcluidos = quedan.length > 0 ? quedan : undefined;
    this.cuerpoSucio = true;
    await this._guardarSinCola();
  }

  excluido(dominio: string): boolean {
    return (this.cont.sitiosExcluidos ?? []).includes(dominio.trim().toLowerCase());
  }

  excluidos(): string[] {
    return [...(this.cont.sitiosExcluidos ?? [])];
  }

  // ---------------------------------------------------------------- para fundir

  /** @internal Lo que usa `fundir.ts`. */
  get _estado() {
    return { doc: this.doc, cont: this.cont, llave: this.clave() };
  }

  /** @internal */
  _ponerFundido(cont: Contenido, sobres: Sobre[]): void {
    this.cont = cont;
    this.doc = { ...this.doc, sobres };
    this.cuerpoSucio = true;
  }
}

/** El sello con sus claves en el orden de Go, para que el JSON se lea igual en los dos. */
function ordenSello(s: Sello): Record<string, unknown> {
  const o: Record<string, unknown> = { id: s.id, serie: s.serie, huellas: s.huellas };
  if (s.sincro) o.sincro = s.sincro;
  o.cuerpo = s.cuerpo;
  return o;
}

/** Para comparar contenidos: la forma canónica del JSON, con los vacíos fuera. */
export function canonContenido(c: Contenido): string {
  return canonico(contenidoAJSON(c));
}
