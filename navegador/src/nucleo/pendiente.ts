/**
 * Las copias que esperan a que quien las recibe tenga cuenta (ADR 0043, B3).
 *
 * **Espejo de `internal/boveda/pendiente.go`**, y ahí está el porqué largo: el
 * sobre no sale hasta que hay llaves de verdad a las que cifrarlo, porque la
 * alternativa era mandar la contraseña por correo. Lo que queda aquí es una nota
 * —a quién, qué entrada y con qué llaves se intentó—, nunca el secreto.
 *
 * Vive en la sección `envios` del contenido cifrado, que esta implementación
 * trata como desconocida salvo aquí y al fundir, igual que `identidad`. Cualquier
 * cambio de estas reglas **se hace en los dos lenguajes**, y lo vigilan las
 * pruebas cruzadas.
 */

import { compararComoGo } from "./canon";
import type { ValorJSON } from "./canon";
import { rfc3339 } from "./entrada";

/** Lo mismo que `PlazoPendientes` en Go: treinta días, como la invitación. */
export const DIAS_DE_PENDIENTE = 30;

export type Pendiente = {
  id: string;
  entrada: string;
  correo: string;
  huella: string;
  creado: string;
};

function esPendiente(x: unknown): x is Pendiente {
  if (!x || typeof x !== "object" || Array.isArray(x)) return false;
  const p = x as Record<string, unknown>;
  return ["id", "entrada", "correo", "huella", "creado"].every((k) => typeof p[k] === "string");
}

/** Los que lleva un contenido, tal cual están guardados. */
export function pendientesDe(extra: Record<string, ValorJSON> | undefined): Pendiente[] {
  const lista = extra?.envios as unknown;
  if (!Array.isArray(lista)) return [];
  return lista.filter(esPendiente);
}

/** Los deja en la sección, o la quita si no queda ninguno. Como el `omitempty` de Go. */
export function ponerPendientes(
  extra: Record<string, ValorJSON> | undefined,
  lista: Pendiente[],
): Record<string, ValorJSON> | undefined {
  const out = { ...(extra ?? {}) };
  if (lista.length > 0) out.envios = lista as unknown as ValorJSON;
  else delete out.envios;
  return Object.keys(out).length > 0 ? out : undefined;
}

/** Los que ya han pasado del plazo, fuera. */
export function purgarPendientes(lista: Pendiente[], cuando: Date): Pendiente[] {
  const corte = rfc3339(new Date(cuando.getTime() - DIAS_DE_PENDIENTE * 24 * 3600_000));
  return lista.filter((p) => p.creado > corte);
}

/**
 * Junta las dos listas **como un conjunto por identificador**, con la regla de los
 * sitios excluidos: lo que estaba en la base y falta en un lado, lo quitó ese lado.
 * Ordenadas por identificador, que es el único orden que los dos equipos calculan
 * igual. Espejo exacto de `fundirPendientes` en Go.
 */
export function fundirPendientes(l: Pendiente[], r: Pendiente[], b: Pendiente[], hayBase: boolean): Pendiente[] {
  const en = (lista: Pendiente[]) => new Map(lista.map((p) => [p.id, p]));
  const ml = en(l);
  const mr = en(r);
  const mb = en(b);
  const out: Pendiente[] = [];
  for (const [id, p] of new Map([...ml, ...mr])) {
    const enL = ml.has(id);
    const enR = mr.has(id);
    const enB = mb.has(id);
    if ((enL && enR) || !hayBase || (enL && !enB) || (enR && !enB)) out.push(mr.get(id) ?? p);
  }
  return out.sort((x, y) => compararComoGo(x.id, y.id));
}
