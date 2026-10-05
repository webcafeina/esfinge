/**
 * Las bóvedas de otras personas a las que tengo acceso, en el espejo (ADR 0052).
 *
 * **La extensión no gestiona accesos**: no los da, no los quita y no los acepta. Eso
 * pasa entero en la ventana. Lo único que tiene que hacer con esta sección es lo
 * mismo que con los proyectos — **fundirla igual que Go** — y por la misma razón, que
 * no es evidente: la sección vive en `extra` porque el `Contenido` de aquí no la
 * conoce, y **una sección desconocida se funde como un bloque**, o sea que gana la
 * del servidor entera. Con eso, un acceso aceptado en la ventana mientras la
 * extensión sincronizaba se perdería; peor, los dos lados podrían quedarse con listas
 * distintas y pasarse la bóveda sin fin, que es el fallo que la ADR 0038 vino a
 * cerrar.
 *
 * Lo que esta sección **no** lleva es la clave de ninguna bóveda compartida: lo que
 * abre una es la ranura sellada que va dentro de su propio fichero. Ver
 * `compartida.go`.
 */

import { compararComoGo, type ValorJSON } from "./canon";

/** Lo que mi bóveda guarda de cada bóveda ajena a la que tengo acceso. */
export type Compartida = {
  /** La cuenta de quien la comparte y la referencia **en esa cuenta**. */
  dueno: string;
  ref: string;
  nombre: string;
  /** Quién soy yo en esa bóveda: el identificador de mi ranura. */
  titular: string;
  /** "ver" o "editar". **Informativo**: el que manda es el del servidor. */
  permiso: string;
  huella?: string;
  desde: string;
  usado?: string;
};

function esCompartida(x: unknown): x is Compartida {
  if (typeof x !== "object" || x === null) return false;
  const c = x as Record<string, unknown>;
  return (
    typeof c.dueno === "string" &&
    typeof c.ref === "string" &&
    typeof c.nombre === "string" &&
    typeof c.titular === "string" &&
    typeof c.permiso === "string" &&
    typeof c.desde === "string"
  );
}

/** Las de una sección `extra`, si las lleva y se entienden. */
export function compartidasDe(extra: Record<string, ValorJSON> | undefined): Compartida[] {
  const lista = extra?.compartidas as unknown;
  if (!Array.isArray(lista)) return [];
  return lista.filter(esCompartida);
}

/** Las deja en la sección, o la quita si no queda ninguna. Como el `omitempty` de Go. */
export function ponerCompartidas(
  extra: Record<string, ValorJSON> | undefined,
  lista: Compartida[],
): Record<string, ValorJSON> | undefined {
  const out = { ...(extra ?? {}) };
  if (lista.length > 0) out.compartidas = lista as unknown as ValorJSON;
  else delete out.compartidas;
  return Object.keys(out).length > 0 ? out : undefined;
}

const clave = (c: Compartida) => `${c.dueno}/${c.ref}`;

/**
 * Conjunto por dueño+ref a tres bandas, igual que los proyectos: dejar de ver una
 * aquí no la devuelve el otro equipo, y aceptar una allí llega aquí.
 *
 * Tiene que dar lo mismo que `fundirCompartidas` de Go byte a byte, y lo vigila
 * `TestCruzadaFusionAlAzar`.
 */
export function fundirCompartidas(
  l: Compartida[],
  r: Compartida[],
  b: Compartida[],
  hayBase: boolean,
): Compartida[] {
  const en = (lista: Compartida[]) => new Map(lista.map((c) => [clave(c), c]));
  const ml = en(l);
  const mr = en(r);
  const mb = en(b);
  const out: Compartida[] = [];
  for (const k of new Map([...ml, ...mr]).keys()) {
    const cl = ml.get(k);
    const cr = mr.get(k);
    const cb = mb.get(k);
    const enL = cl !== undefined;
    const enR = cr !== undefined;
    const enB = cb !== undefined;
    if (!((enL && enR) || !hayBase || (enL && !enB) || (enR && !enB))) continue;
    if (!enL) out.push(cr!);
    else if (!enR) out.push(cl);
    else out.push(unaCompartida(cl, cr, cb, enB));
  }
  return out.sort((x, y) => compararComoGo(clave(x), clave(y)));
}

function unaCompartida(
  l: Compartida,
  r: Compartida,
  b: Compartida | undefined,
  hayBase: boolean,
): Compartida {
  const out: Compartida = { ...r };
  if ((l.usado ?? "") > (out.usado ?? "")) out.usado = l.usado;

  if (!hayBase) {
    if (compararComoGo(l.nombre, r.nombre) > 0) out.nombre = l.nombre;
  } else if (l.nombre !== b!.nombre && r.nombre === b!.nombre) {
    out.nombre = l.nombre;
  } else if (l.nombre !== b!.nombre && r.nombre !== b!.nombre && compararComoGo(l.nombre, r.nombre) > 0) {
    out.nombre = l.nombre;
  }

  // **El permiso no lo deciden mis equipos: lo decide el servidor.** Aquí se queda el
  // del lado que lo cambió, y si cambió en los dos, el más estrecho: equivocarse
  // hacia «ver» cuesta un 403 que se explica, y hacia «editar» cuesta pedirle a
  // alguien que teclee algo que va a acabar rechazado.
  if (!hayBase) {
    if (l.permiso === "ver" || r.permiso === "ver") out.permiso = "ver";
  } else if (l.permiso !== b!.permiso && r.permiso === b!.permiso) {
    out.permiso = l.permiso;
  } else if (l.permiso !== b!.permiso && r.permiso !== b!.permiso) {
    out.permiso = "ver";
  }
  return out;
}
