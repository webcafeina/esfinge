/**
 * La fusión a tres bandas, **la misma que `Fundir` en Go** (ADR 0038 y 0040).
 *
 * Es la pieza donde un desacuerdo entre las dos implementaciones sale más caro:
 * si Go y la extensión funden distinto, cada una ve la bóveda de la otra como un
 * cambio, la sube, y se la pasan sin fin entre equipos. Por eso no se prueba solo
 * aquí: una prueba de Go genera escenarios al azar, los funde con las dos y exige
 * el mismo resultado byte a byte (`internal/boveda/cruzada_test.go`).
 *
 * Las reglas están en `docs/formato-boveda.md`, «Fundir». Aquí se sigue el código
 * de Go línea a línea, incluidos los nombres, para que comparar los dos sea leer.
 */

import { canonico, compararComoGo, huella, type ValorJSON } from "./canon";
import { fundirIdentidad, type IdentidadGuardada } from "./identidad";
import { fundirPendientes, pendientesDe, ponerPendientes } from "./pendiente";
import {
  ahora,
  Boveda,
  canonContenido,
  desempaquetar,
  ErrorBoveda,
  leerDocumento,
  RANURAS_LOCALES,
  type Contenido,
  type Sobre,
} from "./boveda";
import { canonEntrada, entradaAJSON, entradaDesde, MAXIMO_HISTORIAL, rfc3339, type Antigua, type Entrada } from "./entrada";

export type Fusion = {
  /** Entradas que han llegado o cambiado desde el otro lado. */
  traidas: number;
  /** Entradas vivas aquí que se van porque se borraron allí. */
  borradas: number;
  /** Entradas tocadas en los dos lados a la vez. */
  conflictos: number;
  /** Si la bóveda de aquí ha cambiado y se ha guardado. */
  cambio: boolean;
  /** Si lo que queda aquí es distinto de lo del servidor: hay que subirlo. */
  subir: boolean;
  /** La serie del documento justo después de fundir. */
  serie: number;
};

export type Resultado = { contenido: Contenido; sobres: Sobre[]; traidas: number; conflictos: number };

/**
 * La parte pura: con los tres contenidos y las tres listas de sobres, lo que
 * queda. Es lo que compara la prueba cruzada con Go.
 */
export async function fundirPiezas(
  l: Contenido,
  r: Contenido,
  b: Contenido | null,
  sobresL: Sobre[],
  sobresR: Sobre[],
  sobresB: Sobre[],
  cuando: Date,
): Promise<Resultado> {
  const f = { traidas: 0, conflictos: 0 };
  const contenido = await fundirContenido(l, r, b, cuando, f);
  const sobres = fundirSobres(sobresL, sobresR, sobresB, b !== null);
  return { contenido, sobres, ...f };
}

/**
 * Junta en `boveda` lo que ha llegado del servidor: `remoto` en su `version` y
 * `base`, la última versión del servidor que vio este lado, si se tiene. Si
 * cambia algo aquí, se guarda.
 */
export async function fundir(
  boveda: Boveda,
  remoto: string,
  version: number,
  base: string | null,
  opciones: { aunqueBorreMucho?: boolean } = {},
): Promise<Fusion> {
  // **Entera dentro de la cola de la bóveda**: si no, lo que se guardara mientras
  // se funde —la tarjeta de una página— se perdería al poner lo fundido.
  return boveda._exclusivo(() => fundirSinCola(boveda, remoto, version, base, opciones));
}

async function fundirSinCola(
  boveda: Boveda,
  remoto: string,
  version: number,
  base: string | null,
  opciones: { aunqueBorreMucho?: boolean },
): Promise<Fusion> {
  if (boveda.soloLectura) throw new ErrorBoveda("formato-nuevo");
  const { doc, cont, llave } = boveda._estado;
  const docR = leerDocumento(remoto);
  if (docR.id !== doc.id) throw new ErrorBoveda("otra-boveda");
  const { sel: selR, cont: contR } = await desempaquetar(docR, llave);
  if ((selR.sincro ?? 0) !== version) {
    throw new ErrorBoveda("retroceso", `El servidor ha devuelto una versión de la bóveda que no cuadra (dice la ${version} y lleva dentro la ${selR.sincro ?? 0})`);
  }

  // La base es una ayuda, no una condición: si no se puede leer, se funde sin ella.
  let contB: Contenido | null = null;
  let sobresB: Sobre[] = [];
  if (base) {
    try {
      const docB = leerDocumento(base);
      if (docB.id === doc.id) {
        contB = (await desempaquetar(docB, llave)).cont;
        sobresB = docB.sobres;
      }
    } catch {
      contB = null;
    }
  }

  const r = await fundirPiezas(cont, contR, contB, doc.sobres, docR.sobres, sobresB, ahora());
  const f: Fusion = {
    traidas: r.traidas,
    conflictos: r.conflictos,
    borradas: 0,
    cambio: canonContenido(r.contenido) !== canonContenido(cont) || !mismasRanuras(r.sobres, doc.sobres),
    subir:
      canonContenido(r.contenido) !== canonContenido(contR) ||
      !mismasRanuras(sinLocales(r.sobres), sinLocales(docR.sobres)),
    serie: doc.serie,
  };

  // **Perdida es la que desaparece, no la que va a la papelera** (revisión del
  // 2026-09-23): ver el mismo trozo en `internal/boveda/sincronizar.go`.
  const antes = vivas(cont.entradas);
  const quedan = new Set(r.contenido.entradas.map((e) => e.id));
  for (const id of antes) if (!quedan.has(id)) f.borradas++;
  if (!opciones.aunqueBorreMucho && antes.size >= 4 && f.borradas * 2 > antes.size) {
    throw Object.assign(new ErrorBoveda("muchos-borrados"), { fusion: f });
  }
  if (!f.cambio) return f;
  boveda._ponerFundido(r.contenido, r.sobres);
  await boveda._guardarSinCola();
  f.serie = boveda.serie;
  return f;
}

// ------------------------------------------------------------------ piezas

function vivas(es: Entrada[]): Set<string> {
  return new Set(es.filter((e) => !e.papelera).map((e) => e.id));
}

function porTipo(ss: Sobre[]): Map<string, Sobre> {
  const m = new Map<string, Sobre>();
  for (const s of ss) m.set(s.tipo, s);
  return m;
}

const canonSobre = (s: Sobre | undefined) => (s ? canonico(s as unknown as ValorJSON) : "");

/** Sin mirar el orden: dos equipos que listen las ranuras distinto no han cambiado nada. */
export function mismasRanuras(a: Sobre[], b: Sobre[]): boolean {
  const ma = porTipo(a);
  const mb = porTipo(b);
  if (ma.size !== mb.size) return false;
  for (const [t, s] of ma) if (canonSobre(s) !== canonSobre(mb.get(t))) return false;
  return true;
}

function sinLocales(ss: Sobre[]): Sobre[] {
  return ss.filter((s) => !RANURAS_LOCALES.has(s.tipo));
}

const mismoSobre = (a: Sobre, b: Sobre) =>
  a.tipo === b.tipo && a.creado === b.creado && a.contenedor === b.contenedor && (a.codificacion ?? "") === (b.codificacion ?? "");

/** De cada tipo de ranura, cuál queda. Ver `fundirSobres` en Go. */
export function fundirSobres(l: Sobre[], r: Sobre[], b: Sobre[], hayBase: boolean): Sobre[] {
  const mL = porTipo(l);
  const mR = porTipo(r);
  const mB = porTipo(b);
  const orden = l.map((s) => s.tipo);
  for (const s of r) if (!mL.has(s.tipo) && !RANURAS_LOCALES.has(s.tipo)) orden.push(s.tipo);
  const out: Sobre[] = [];
  for (const tipo of orden) {
    const sl = mL.get(tipo)!;
    const sr = mR.get(tipo);
    const sb = mB.get(tipo);
    if (RANURAS_LOCALES.has(tipo) || !sr) out.push(sl);
    else if (!mL.has(tipo)) out.push(sr);
    else if (mismoSobre(sl, sr)) out.push(sl);
    else if (hayBase && sb && mismoSobre(sl, sb)) out.push(sr);
    else if (hayBase && sb && mismoSobre(sr, sb)) out.push(sl);
    else if (sr.creado > sl.creado || (sr.creado === sl.creado && compararComoGo(sr.contenedor, sl.contenedor) > 0)) out.push(sr);
    else out.push(sl);
  }
  return out;
}

async function fundirContenido(
  l: Contenido,
  r: Contenido,
  b: Contenido | null,
  cuando: Date,
  f: { traidas: number; conflictos: number },
): Promise<Contenido> {
  const lapidas = unirLapidas(l.lapidas, r.lapidas);
  const out: Contenido = { entradas: await fundirEntradas(l, r, b, lapidas, cuando, f) };
  if (Object.keys(lapidas).length > 0) out.lapidas = lapidas;
  const excluidos = fundirConjunto(l.sitiosExcluidos ?? [], r.sitiosExcluidos ?? [], b?.sitiosExcluidos ?? [], b !== null);
  if (excluidos.length > 0) out.sitiosExcluidos = excluidos;
  const extra = fundirSecciones(l.extra ?? {}, r.extra ?? {}, b?.extra ?? {});
  if (extra) out.extra = extra;
  // **La identidad no se funde como una sección cualquiera: se elige una**
  // (ADR 0043). Go hace lo mismo, y tiene que ser la misma: como sección
  // desconocida ganaría la del servidor, y cada lado podría quedarse con una.
  const identidad = fundirIdentidad(identidadDe(l), identidadDe(r));
  if (identidad) out.extra = { ...(out.extra ?? {}), identidad: identidad as unknown as ValorJSON };
  else if (out.extra) delete out.extra.identidad;
  // **Y las copias que esperan, como un conjunto** (ADR 0043, B3): como sección
  // desconocida, la lista entera del servidor ganaría y se perdería la nota de un
  // envío hecho aquí, que es una copia que ya no sale nunca.
  out.extra = ponerPendientes(
    out.extra,
    fundirPendientes(
      pendientesDe(l.extra),
      pendientesDe(r.extra),
      pendientesDe(b?.extra),
      b !== null,
    ),
  );
  return out;
}

function unirLapidas(a?: Record<string, string>, b?: Record<string, string>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const m of [a ?? {}, b ?? {}]) {
    for (const [id, cuando] of Object.entries(m)) if (cuando > (out[id] ?? "")) out[id] = cuando;
  }
  return out;
}

async function fundirEntradas(
  l: Contenido,
  r: Contenido,
  b: Contenido | null,
  lapidas: Record<string, string>,
  cuando: Date,
  f: { traidas: number; conflictos: number },
): Promise<Entrada[]> {
  const indice = (es: Entrada[]) => new Map(es.map((e) => [e.id, e] as const));
  const mL = indice(l.entradas);
  const mR = indice(r.entradas);
  const mB = b ? indice(b.entradas) : new Map<string, Entrada>();
  const lapidasB = b?.lapidas ?? {};
  const lapidasL = l.lapidas ?? {};
  const lapidasR = r.lapidas ?? {};

  // El orden es el del servidor, con lo nuevo de aquí al final.
  const orden: string[] = [];
  const visto = new Set<string>();
  for (const lista of [r.entradas, l.entradas]) {
    for (const e of lista) {
      if (!visto.has(e.id)) {
        visto.add(e.id);
        orden.push(e.id);
      }
    }
  }

  const sigue = (p: Entrada, eb: Entrada | undefined, lapidaDelOtro: string, lapidaPropia: string): boolean => {
    if (eb) return canonEntrada(p) !== canonEntrada(eb); // la edición gana al borrado
    if (!lapidaDelOtro) return true; // nueva de este lado
    if (b !== null && (lapidasB[p.id] ?? "") !== "" && lapidaPropia === "") return true;
    return p.cambiada >= lapidaDelOtro;
  };

  const out: Entrada[] = [];
  for (const id of orden) {
    const el = mL.get(id);
    const er = mR.get(id);
    const eb = mB.get(id);
    if (el && er) {
      delete lapidas[id];
      if (canonEntrada(el) === canonEntrada(er)) {
        out.push(er);
      } else if (eb && canonEntrada(el) === canonEntrada(eb)) {
        out.push(er);
        f.traidas++;
      } else if (eb && canonEntrada(er) === canonEntrada(eb)) {
        out.push(el);
      } else {
        out.push(await fundirCampos(el, er, eb ?? null, cuando));
        f.conflictos++;
      }
    } else if (el) {
      if (sigue(el, eb, lapidasR[id] ?? "", lapidasL[id] ?? "")) {
        delete lapidas[id];
        out.push(el);
        continue;
      }
      if (!lapidas[id]) lapidas[id] = rfc3339(cuando);
    } else if (er) {
      if (sigue(er, eb, lapidasL[id] ?? "", lapidasR[id] ?? "")) {
        delete lapidas[id];
        out.push(er);
        f.traidas++;
        continue;
      }
      if (!lapidas[id]) lapidas[id] = rfc3339(cuando);
    }
  }
  return out;
}

/** Junta una entrada tocada en los dos lados. Ninguna contraseña se pierde: la que no gana va al historial. */
async function fundirCampos(l: Entrada, r: Entrada, b: Entrada | null, cuando: Date): Promise<Entrada> {
  const ganaL = await mayor(l, r);
  const [ganadora, perdedora] = ganaL ? [l, r] : [r, l];

  let out: Entrada;
  if (!b) {
    out = entradaDesde(JSON.parse(JSON.stringify(entradaAJSON(ganadora))));
  } else {
    const ml = entradaAJSON(l);
    const mr = entradaAJSON(r);
    const mb = entradaAJSON(b);
    const igual = (x: ValorJSON | undefined, y: ValorJSON | undefined) =>
      x === undefined || y === undefined ? x === y : canonico(x) === canonico(y);
    const fundido: Record<string, ValorJSON> = {};
    for (const k of new Set([...Object.keys(ml), ...Object.keys(mr)])) {
      const vl = ml[k];
      const vr = mr[k];
      const vb = mb[k];
      let v: ValorJSON | undefined;
      if (igual(vl, vr)) v = vl;
      else if (igual(vl, vb)) v = vr;
      else if (igual(vr, vb)) v = vl;
      else v = ganaL ? vl : vr;
      if (v !== undefined) fundido[k] = v;
    }
    try {
      out = entradaDesde(JSON.parse(JSON.stringify(fundido)));
    } catch {
      out = entradaDesde(JSON.parse(JSON.stringify(entradaAJSON(ganadora))));
    }
  }

  // **La edición gana al borrado también con la papelera** (revisión del
  // 2026-09-23): ver el mismo trozo en `internal/boveda/sincronizar.go`.
  if (b) {
    const soloL = soloALaPapelera(l, b);
    const soloR = soloALaPapelera(r, b);
    if (soloL !== soloR) {
      const conContenido = soloR ? l : r;
      if (conContenido.papelera) out.papelera = true;
      else delete out.papelera;
      if (conContenido.borradaEn) out.borradaEn = conContenido.borradaEn;
      else delete out.borradaEn;
    }
  }

  const listas: Antigua[][] = [l.historial ?? [], r.historial ?? []];
  if (perdedora.secreto && perdedora.secreto !== (out.secreto ?? "")) {
    listas.push([{ secreto: perdedora.secreto, hasta: rfc3339(cuando) }]);
  }
  const h = unirHistorial(out.secreto ?? "", listas);
  if (h.length > 0) out.historial = h;
  else delete out.historial;
  out.revision = Math.max(l.revision ?? 0, r.revision ?? 0) + 1;
  out.cambiada = l.cambiada > r.cambiada ? l.cambiada : r.cambiada;
  return out;
}

/**
 * Si este lado, respecto a la versión común, **no ha hecho más que mandarla a la
 * papelera**: ni contraseña, ni título, ni nada.
 */
function soloALaPapelera(x: Entrada, b: Entrada): boolean {
  if (!x.papelera || Boolean(x.papelera) === Boolean(b.papelera)) return false;
  const sinPapelera = (e: Entrada) => {
    const c = { ...e };
    delete c.papelera;
    delete c.borradaEn;
    delete c.revision;
    // `cambiada` también: borrar la toca, así que dejarla haría que cualquier
    // borrado pareciera un cambio de contenido. Va siempre, así que se vacía en
    // vez de quitarse; lo mismo hace Go.
    c.cambiada = "";
    return canonEntrada(c);
  };
  return sinPapelera(x) === sinPapelera(b);
}

/** Si `a` gana a `b`: más revisiones, luego la fecha, luego la huella de su forma canónica. */
async function mayor(a: Entrada, b: Entrada): Promise<boolean> {
  if ((a.revision ?? 0) !== (b.revision ?? 0)) return (a.revision ?? 0) > (b.revision ?? 0);
  if (a.cambiada !== b.cambiada) return a.cambiada > b.cambiada;
  return (await huella(canonEntrada(a))) >= (await huella(canonEntrada(b)));
}

function unirHistorial(actual: string, listas: Antigua[][]): Antigua[] {
  const porSecreto = new Map<string, Antigua>();
  for (const lista of listas) {
    for (const a of lista) {
      if (!a.secreto || a.secreto === actual) continue;
      const v = porSecreto.get(a.secreto);
      if (!v || a.hasta > v.hasta) porSecreto.set(a.secreto, { secreto: a.secreto, hasta: a.hasta });
    }
  }
  return [...porSecreto.values()]
    .sort((x, y) => (x.hasta !== y.hasta ? (x.hasta > y.hasta ? -1 : 1) : compararComoGo(x.secreto, y.secreto)))
    .slice(0, MAXIMO_HISTORIAL);
}

/** Con base, algo está si está en los dos lados o si lo añadió uno; sin base, se unen. */
function fundirConjunto(l: string[], r: string[], b: string[], hayBase: boolean): string[] {
  const sl = new Set(l);
  const sr = new Set(r);
  const sb = new Set(b);
  const out: string[] = [];
  for (const x of new Set([...l, ...r])) {
    if ((sl.has(x) && sr.has(x)) || !hayBase || (sl.has(x) && !sb.has(x)) || (sr.has(x) && !sb.has(x))) out.push(x);
  }
  return out.sort(compararComoGo);
}

/** La identidad guardada de un contenido, si la lleva y se entiende. */
function identidadDe(c: Contenido): IdentidadGuardada | undefined {
  const i = c.extra?.identidad as unknown;
  if (!i || typeof i !== "object") return undefined;
  const { semilla, creada } = i as Record<string, unknown>;
  if (typeof semilla !== "string" || typeof creada !== "string") return undefined;
  return i as IdentidadGuardada;
}

/** Las secciones que no se conocen, enteras: si cambió en los dos lados, gana la del servidor. */
function fundirSecciones(
  l: Record<string, ValorJSON>,
  r: Record<string, ValorJSON>,
  b: Record<string, ValorJSON>,
): Record<string, ValorJSON> | null {
  const igual = (x: ValorJSON | undefined, y: ValorJSON | undefined) =>
    x === undefined || y === undefined ? x === y : canonico(x) === canonico(y);
  const out: Record<string, ValorJSON> = {};
  for (const k of new Set([...Object.keys(l), ...Object.keys(r)])) {
    const vl = l[k];
    const vr = r[k];
    const vb = b[k];
    let v: ValorJSON | undefined;
    if (igual(vl, vr)) v = vl;
    else if (igual(vl, vb)) v = vr;
    else if (igual(vr, vb)) v = vl;
    else v = vr;
    if (v !== undefined) out[k] = v;
  }
  return Object.keys(out).length > 0 ? out : null;
}
