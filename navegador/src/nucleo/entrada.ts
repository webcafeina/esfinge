/**
 * Una entrada de la bóveda, igual que `Entrada` en `internal/boveda/entrada.go`.
 *
 * Dos reglas de Go que aquí hay que cumplir a mano, porque de ellas depende que
 * la forma canónica salga igual en los dos lados:
 *
 * - **Los vacíos no se escriben** (`omitempty`), salvo `id`, `tipo`, `titulo`,
 *   `creada` y `cambiada`, que van siempre. Y en el historial, `secreto` y `hasta`
 *   también van siempre, y nada más: Go no conserva lo desconocido ahí dentro.
 * - **Lo desconocido se conserva** en `extra`, tal cual, y lo conocido manda: una
 *   copia vieja de un campo que se entiende no pisa el de verdad.
 */

import { canonico, type ValorJSON } from "./canon";

export type Tipo = "credencial" | "nota" | "tarjeta" | "identidad";

export type Antigua = { secreto: string; hasta: string };

export type Entrada = {
  id: string;
  tipo: Tipo | string;
  titulo: string;
  notas?: string;
  etiquetas?: string[];
  carpeta?: string;
  creada: string;
  cambiada: string;
  revision?: number;
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
  /** Los campos que esta versión no conoce. */
  extra?: Record<string, ValorJSON>;
};

/** El orden y la clase de cada campo conocido, que es el orden de la estructura de Go. */
const CAMPOS: [keyof Entrada, "siempre" | "texto" | "lista" | "numero" | "si" | "historial"][] = [
  ["id", "siempre"],
  ["tipo", "siempre"],
  ["titulo", "siempre"],
  ["notas", "texto"],
  ["etiquetas", "lista"],
  ["carpeta", "texto"],
  ["creada", "siempre"],
  ["cambiada", "siempre"],
  ["revision", "numero"],
  ["papelera", "si"],
  ["borradaEn", "texto"],
  ["usuario", "texto"],
  ["secreto", "texto"],
  ["sitios", "lista"],
  ["totp", "texto"],
  ["historial", "historial"],
  ["titular", "texto"],
  ["numero", "texto"],
  ["caduca", "texto"],
  ["verificacion", "texto"],
  ["nombreCompleto", "texto"],
  ["documento", "texto"],
  ["numeroDocumento", "texto"],
];
const CONOCIDOS = new Set<string>(CAMPOS.map(([k]) => k as string));

export const MAXIMO_HISTORIAL = 10;

class ErrorDeForma extends Error {}

function texto(v: unknown): string {
  if (v === null || v === undefined) return "";
  if (typeof v !== "string") throw new ErrorDeForma("Una entrada tiene un campo de texto que no es texto");
  return v;
}

/** Lee una entrada del JSON del cuerpo, como `json.Unmarshal` en Go. */
export function entradaDesde(crudo: unknown): Entrada {
  if (crudo === null || typeof crudo !== "object" || Array.isArray(crudo)) {
    throw new ErrorDeForma("Una entrada no es un objeto");
  }
  const o = crudo as Record<string, unknown>;
  const e: Entrada = { id: "", tipo: "", titulo: "", creada: "", cambiada: "" };
  for (const [k, clase] of CAMPOS) {
    const v = o[k as string];
    switch (clase) {
      case "siempre":
      case "texto": {
        const t = texto(v);
        if (clase === "siempre" || t !== "") (e as Record<string, unknown>)[k] = t;
        break;
      }
      case "lista": {
        if (v === null || v === undefined) break;
        if (!Array.isArray(v)) throw new ErrorDeForma("Una entrada tiene una lista que no es una lista");
        const l = v.map(texto);
        if (l.length > 0) (e as Record<string, unknown>)[k] = l;
        break;
      }
      case "numero": {
        if (v === null || v === undefined) break;
        if (typeof v !== "number" || !Number.isInteger(v)) throw new ErrorDeForma("La revisión no es un número entero");
        if (v !== 0) e.revision = v;
        break;
      }
      case "si": {
        if (v === null || v === undefined) break;
        if (typeof v !== "boolean") throw new ErrorDeForma("La papelera no es sí o no");
        if (v) e.papelera = true;
        break;
      }
      case "historial": {
        if (v === null || v === undefined) break;
        if (!Array.isArray(v)) throw new ErrorDeForma("El historial no es una lista");
        const h = v.map((a) => {
          const x = (a ?? {}) as Record<string, unknown>;
          return { secreto: texto(x.secreto), hasta: texto(x.hasta) };
        });
        if (h.length > 0) e.historial = h;
        break;
      }
    }
  }
  const extra: Record<string, ValorJSON> = {};
  for (const [k, v] of Object.entries(o)) {
    if (!CONOCIDOS.has(k)) extra[k] = v as ValorJSON;
  }
  if (Object.keys(extra).length > 0) e.extra = extra;
  return e;
}

/** La entrada como objeto JSON, con los vacíos fuera y lo desconocido dentro. */
export function entradaAJSON(e: Entrada): Record<string, ValorJSON> {
  const out: Record<string, ValorJSON> = {};
  for (const [k, clase] of CAMPOS) {
    const v = e[k];
    switch (clase) {
      case "siempre":
        out[k] = (v as string | undefined) ?? "";
        break;
      case "texto":
        if (v) out[k] = v as string;
        break;
      case "lista":
        if (Array.isArray(v) && v.length > 0) out[k] = v as string[];
        break;
      case "numero":
        if (v) out[k] = v as number;
        break;
      case "si":
        if (v) out[k] = true;
        break;
      case "historial":
        if (Array.isArray(v) && v.length > 0) {
          out[k] = (v as Antigua[]).map((a) => ({ secreto: a.secreto ?? "", hasta: a.hasta ?? "" }));
        }
        break;
    }
  }
  for (const [k, v] of Object.entries(e.extra ?? {})) {
    if (!CONOCIDOS.has(k)) out[k] = v;
  }
  return out;
}

/** La forma canónica de una entrada (docs/formato-boveda.md). */
export function canonEntrada(e: Entrada): string {
  return canonico(entradaAJSON(e));
}

/** Una copia que no comparte listas ni mapas con el original. */
export function copiar(e: Entrada): Entrada {
  return entradaDesde(JSON.parse(JSON.stringify(entradaAJSON(e))));
}

/** La entrada con lo sensible fuera: lo que cruza a la ventana o al panel en una lista. */
export function sinSecretos(e: Entrada): Entrada {
  const c = copiar(e);
  delete c.secreto;
  delete c.totp;
  delete c.historial;
  delete c.verificacion;
  delete c.numero;
  delete c.numeroDocumento;
  delete c.notas;
  return c;
}

/** Busca por título, usuario, carpeta, nombre, titular, sitios y etiquetas. **Nunca por el secreto.** */
export function coincide(e: Entrada, q: string): boolean {
  const b = q.trim().toLowerCase();
  if (!b) return true;
  const campos = [e.titulo, e.usuario, e.carpeta, e.nombreCompleto, e.titular, ...(e.sitios ?? []), ...(e.etiquetas ?? [])];
  return campos.some((c) => (c ?? "").toLowerCase().includes(b));
}

/** Pone una contraseña nueva y guarda la anterior, como `CambiarSecreto` en Go. */
export function cambiarSecreto(e: Entrada, nuevo: string, ahora: Date): void {
  const cuando = rfc3339(ahora);
  if (e.secreto && e.secreto !== nuevo) {
    e.historial = [{ secreto: e.secreto, hasta: cuando }, ...(e.historial ?? [])].slice(0, MAXIMO_HISTORIAL);
  }
  e.secreto = nuevo;
  e.cambiada = cuando;
}

/** Fecha en RFC3339 a segundos y en UTC, que es como las escribe Go. */
export function rfc3339(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, "Z");
}
