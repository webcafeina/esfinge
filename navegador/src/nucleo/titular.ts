/**
 * Quién tiene acceso a esta bóveda, en el espejo (ADR 0052).
 *
 * **La extensión no gestiona accesos**: no los da ni los quita. Lo único que tiene
 * que hacer con esta sección es **fundirla igual que Go**, por la misma razón que las
 * otras dos: vive en `extra`, y una sección que este lado no conoce se funde como un
 * bloque —gana la del servidor entera—, así que quien diera un acceso en la ventana
 * mientras la extensión sincronizaba lo perdería, y los dos lados podrían quedarse
 * con listas distintas y pasarse la bóveda sin fin.
 */

import { compararComoGo, type ValorJSON } from "./canon";

/** Una persona con acceso a esta bóveda. */
export type Titular = {
  /** El identificador de su ranura, elegido por quien dio el acceso. */
  id: string;
  correo: string;
  permiso: string;
  huella?: string;
  desde: string;
};

function esTitular(x: unknown): x is Titular {
  if (typeof x !== "object" || x === null) return false;
  const t = x as Record<string, unknown>;
  return (
    typeof t.id === "string" &&
    typeof t.correo === "string" &&
    typeof t.permiso === "string" &&
    typeof t.desde === "string"
  );
}

export function titularesDe(extra: Record<string, ValorJSON> | undefined): Titular[] {
  const lista = extra?.titulares as unknown;
  if (!Array.isArray(lista)) return [];
  return lista.filter(esTitular);
}

export function ponerTitulares(
  extra: Record<string, ValorJSON> | undefined,
  lista: Titular[],
): Record<string, ValorJSON> | undefined {
  const out = { ...(extra ?? {}) };
  if (lista.length > 0) out.titulares = lista as unknown as ValorJSON;
  else delete out.titulares;
  return Object.keys(out).length > 0 ? out : undefined;
}

/**
 * Conjunto por identificador a tres bandas. El permiso, si cambió en los dos lados,
 * se queda **en el más estrecho**: equivocarse hacia «ver» cuesta un 403 que se
 * explica, y hacia «editar» cuesta pedirle a alguien que teclee algo que va a acabar
 * rechazado.
 */
export function fundirTitulares(l: Titular[], r: Titular[], b: Titular[], hayBase: boolean): Titular[] {
  const en = (lista: Titular[]) => new Map(lista.map((t) => [t.id, t]));
  const ml = en(l);
  const mr = en(r);
  const mb = en(b);
  const out: Titular[] = [];
  for (const id of new Map([...ml, ...mr]).keys()) {
    const tl = ml.get(id);
    const tr = mr.get(id);
    const tb = mb.get(id);
    const enL = tl !== undefined;
    const enR = tr !== undefined;
    const enB = tb !== undefined;
    if (!((enL && enR) || !hayBase || (enL && !enB) || (enR && !enB))) continue;
    if (!enL) {
      out.push(tr!);
      continue;
    }
    if (!enR) {
      out.push(tl);
      continue;
    }
    const t: Titular = { ...tr };
    if (!enB) {
      if (tl.permiso === "ver" || tr.permiso === "ver") t.permiso = "ver";
    } else if (tl.permiso !== tb!.permiso && tr.permiso === tb!.permiso) {
      t.permiso = tl.permiso;
    } else if (tl.permiso !== tb!.permiso && tr.permiso !== tb!.permiso) {
      t.permiso = "ver";
    }
    out.push(t);
  }
  return out.sort((x, y) => compararComoGo(x.id, y.id));
}
