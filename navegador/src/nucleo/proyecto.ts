/**
 * Las bóvedas de proyecto, en el espejo (ADR 0050).
 *
 * **La extensión no gestiona proyectos**: no los crea, no los abre y no los
 * enseña. Trabaja siempre con la bóveda activa, que elige quien la usa, así que
 * esto no es la mitad de una funcionalidad que falte — es lo único que la
 * extensión tiene que hacer con la sección: **fundirla igual que Go**.
 *
 * Y hace falta, aunque parezca que no. La sección vive en `extra` porque el
 * `Contenido` de aquí no la conoce, y una sección desconocida se funde como un
 * bloque: **gana la del servidor entera**. Con eso, un proyecto creado en la
 * ventana mientras la extensión sincronizaba se perdería; peor, los dos lados
 * podrían quedarse con listas distintas y pasarse la bóveda sin fin, que es
 * exactamente el fallo que la [ADR 0038] vino a cerrar. Es el mismo caso que la
 * identidad y las copias pendientes, y se resuelve igual.
 *
 * Lo que esta sección **no** lleva es la clave de ningún proyecto: lo que abre un
 * proyecto es la ranura que va dentro de su propio fichero. Ver `proyecto.go`.
 */

import { compararComoGo, type ValorJSON } from "./canon";

/** Lo que la bóveda personal guarda de cada proyecto suyo. */
export type Proyecto = {
  /** Hex de 8 bytes, y el nombre de su fichero. */
  ref: string;
  nombre: string;
  creado: string;
  /** La última vez que se abrió. Por aquí se ordena la lista. */
  usado?: string;
  archivado?: boolean;
};

function esProyecto(x: unknown): x is Proyecto {
  if (typeof x !== "object" || x === null) return false;
  const p = x as Record<string, unknown>;
  return typeof p.ref === "string" && typeof p.nombre === "string" && typeof p.creado === "string";
}

/** Los proyectos de una sección `extra`, si los lleva y se entienden. */
export function proyectosDe(extra: Record<string, ValorJSON> | undefined): Proyecto[] {
  const lista = extra?.proyectos as unknown;
  if (!Array.isArray(lista)) return [];
  return lista.filter(esProyecto);
}

/** Los deja en la sección, o la quita si no queda ninguno. Como el `omitempty` de Go. */
export function ponerProyectos(
  extra: Record<string, ValorJSON> | undefined,
  lista: Proyecto[],
): Record<string, ValorJSON> | undefined {
  const out = { ...(extra ?? {}) };
  if (lista.length > 0) out.proyectos = lista as unknown as ValorJSON;
  else delete out.proyectos;
  return Object.keys(out).length > 0 ? out : undefined;
}

/**
 * Junta las dos listas **como un conjunto por ref**, a tres bandas contra la base,
 * igual que los pendientes: lo que estaba en la base y falta en un lado, lo quitó
 * ese lado y no vuelve.
 *
 * Y lo que los pendientes no necesitan: **un proyecto se edita en los dos equipos**,
 * así que no vale quedarse con el del servidor a ciegas. `usado` es «la última vez
 * que se abrió», así que las dos son verdad y **gana la mayor**; el nombre lo decide
 * la base, y si los dos lo cambiaron, el mayor por bytes, que es arbitrario pero
 * **igual en los dos equipos**. Tiene que dar lo mismo que `fundirProyectos` de Go
 * byte a byte: lo vigila `TestCruzadaFusionAlAzar`.
 */
export function fundirProyectos(l: Proyecto[], r: Proyecto[], b: Proyecto[], hayBase: boolean): Proyecto[] {
  const en = (lista: Proyecto[]) => new Map(lista.map((p) => [p.ref, p]));
  const ml = en(l);
  const mr = en(r);
  const mb = en(b);
  const out: Proyecto[] = [];
  for (const ref of new Map([...ml, ...mr]).keys()) {
    const pl = ml.get(ref);
    const pr = mr.get(ref);
    const pb = mb.get(ref);
    const enL = pl !== undefined;
    const enR = pr !== undefined;
    const enB = pb !== undefined;
    if (!((enL && enR) || !hayBase || (enL && !enB) || (enR && !enB))) continue;
    if (!enL) out.push(pr!);
    else if (!enR) out.push(pl);
    else out.push(unProyecto(pl, pr, pb, enB));
  }
  return out.sort((x, y) => compararComoGo(x.ref, y.ref));
}

function unProyecto(l: Proyecto, r: Proyecto, b: Proyecto | undefined, hayBase: boolean): Proyecto {
  const out: Proyecto = { ...r };
  if ((l.usado ?? "") > (out.usado ?? "")) out.usado = l.usado;

  if (!hayBase) {
    if (compararComoGo(l.nombre, r.nombre) > 0) out.nombre = l.nombre;
  } else if (l.nombre !== b!.nombre && r.nombre === b!.nombre) {
    out.nombre = l.nombre;
  } else if (l.nombre !== b!.nombre && r.nombre !== b!.nombre && compararComoGo(l.nombre, r.nombre) > 0) {
    out.nombre = l.nombre;
  }

  // Y archivar: si los dos equipos lo movieron a sitios distintos gana
  // **desarchivado**, que es el estado que lo enseña en vez de esconderlo.
  const al = l.archivado ?? false;
  const ar = r.archivado ?? false;
  const ab = b?.archivado ?? false;
  let archivado = ar;
  if (!hayBase) archivado = al && ar;
  else if (al !== ab && ar === ab) archivado = al;
  else if (al !== ab && ar !== ab) archivado = false;
  if (archivado) out.archivado = true;
  else delete out.archivado;
  return out;
}
