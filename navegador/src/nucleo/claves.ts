/**
 * La cadena de claves de contenido (ADR 0059), **espejo de
 * `internal/boveda/claves_de_contenido.go`**.
 *
 * El porqué está entero en el fichero de Go y no se repite aquí: lo que importa de este
 * lado es que las dos implementaciones elijan **la misma clave** para el mismo documento,
 * porque de eso depende que la extensión pueda abrir lo que escribió la ventana.
 *
 * **Lo que aquí no está, y es a propósito**: crear una clave, poner una nueva en la
 * cadena y escribirla en un sobre. La extensión no rota —rotar es del dueño y se hace en
 * la ventana (publicación B de la ADR 0059)— y un espejo con piezas que nadie llama no es
 * más fiel: es más código que se puede desviar sin que nada lo note. Por lo mismo no hay
 * forma canónica de una cadena: la escribe quien la crea, y aquí solo se lee.
 */

import type { Sobre } from "./boveda";
import { compararComoGo } from "./canon";
import { abrirTexto } from "./esf1";

const deUtf8 = new TextDecoder();

/** Lo que distingue una cadena de una clave de bóveda a secas. Ver Go: lo dice **el claro**. */
const MARCA_DE_LA_CADENA = "claves-de-contenido";

/** La versión del formato de la propia cadena, aparte del formato de la bóveda. */
export const VERSION_DE_LA_CADENA = 1;

/** El sobre que lleva la cadena, envuelta con la clave de bóveda. **Se sube.** */
export const RANURA_DE_CONTENIDO = "contenido";

/**
 * La misma frase que `ErrCadenaNueva` de Go.
 *
 * Va aquí como texto y no como `ErrorBoveda` porque este módulo es una **hoja**: ninguno
 * de los de al lado importa `boveda.ts`, que es lo que mantiene el núcleo sin ciclos
 * —`compartida.ts` y `recuperacion.ts` hacen lo mismo con los suyos—. Quien la traduce
 * al error que lee la pantalla es `boveda.ts`.
 */
export const ERR_CADENA_NUEVA =
  "Las claves de esta bóveda las escribió una versión de Esfinge más nueva; actualiza antes de tocarla";

/** Y lo que lanza cuando la cadena no se entiende, que allí es `ErrManipulada`. */
export const ERR_CADENA_MALA = "La cadena de claves de esta bóveda no cuadra";

export type ClaveDeContenido = {
  /** Ocho bytes en hexadecimal, al azar. **No es un contador**: ver Go. */
  id: string;
  /** Ordena y se enseña. No decide nada. */
  n: number;
  /** 32 bytes en base64url, 43 caracteres, manejados como texto igual que la de bóveda. */
  clave: string;
  creada: string;
  porque?: string;
};

export type CadenaDeClaves = {
  esfinge: string;
  version: number;
  actual: string;
  claves: ClaveDeContenido[];
};

/** Deja la cadena en una sola forma posible, por `(n, id)`, igual que `ordenar` en Go. */
function ordenar(c: CadenaDeClaves): CadenaDeClaves {
  const claves = [...c.claves].sort((a, b) => (a.n !== b.n ? a.n - b.n : a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
  return { ...c, claves };
}

/** La clave de esa época, como texto, igual que `b.llave`. */
export function claveDe(c: CadenaDeClaves, id: string): string | null {
  for (const k of c.claves) if (k.id === id) return k.clave;
  return null;
}

/**
 * Las claves en el orden en que hay que probarlas cuando la época declarada en el JSON de
 * fuera no abre: esa declaración va en claro, así que es una pista y no una autoridad.
 */
export function deLaMasNuevaALaMasVieja(c: CadenaDeClaves): ClaveDeContenido[] {
  return [...c.claves].sort((a, b) => (a.n !== b.n ? b.n - a.n : a.id < b.id ? 1 : a.id > b.id ? -1 : 0));
}

export function cadenaVacia(c: CadenaDeClaves | null): boolean {
  return c === null || c.claves.length === 0;
}

/**
 * Dice si un claro recién abierto es una cadena y no una clave de bóveda a secas. Mira
 * **el claro**, por lo que dice Go: el `codificacion` del JSON de fuera lo puede cambiar
 * cualquiera que pueda escribir el fichero.
 */
export function esCadena(claro: string): boolean {
  return claro.startsWith("{") && claro.includes(`"${MARCA_DE_LA_CADENA}"`);
}

/** Lee lo que salga de un sobre `contenido` o de una ranura de acceso nueva. */
export function leerCadena(claro: string): CadenaDeClaves {
  let o: Record<string, unknown>;
  try {
    o = JSON.parse(claro) as Record<string, unknown>;
  } catch {
    throw new Error(ERR_CADENA_MALA);
  }
  if (!o || typeof o !== "object" || o.esfinge !== MARCA_DE_LA_CADENA) throw new Error(ERR_CADENA_MALA);
  const version = typeof o.version === "number" ? o.version : 0;
  if (version > VERSION_DE_LA_CADENA) throw new Error(ERR_CADENA_NUEVA);
  const claves: ClaveDeContenido[] = (Array.isArray(o.claves) ? (o.claves as Record<string, unknown>[]) : []).map((k) => {
    const x: ClaveDeContenido = {
      id: typeof k?.id === "string" ? k.id : "",
      n: typeof k?.n === "number" ? k.n : 0,
      clave: typeof k?.clave === "string" ? k.clave : "",
      creada: typeof k?.creada === "string" ? k.creada : "",
    };
    if (typeof k?.porque === "string" && k.porque) x.porque = k.porque;
    return x;
  });
  const c = ordenar({
    esfinge: MARCA_DE_LA_CADENA,
    version,
    actual: typeof o.actual === "string" ? o.actual : "",
    claves,
  });
  if (c.claves.length === 0) throw new Error(ERR_CADENA_MALA);
  if (claveDe(c, c.actual) === null) throw new Error(ERR_CADENA_MALA);
  return c;
}

/**
 * La cadena de una bóveda del formato 1: **una sola clave, que es la de bóveda**, con la
 * época vacía en los dos lados.
 *
 * Es lo que evita que haya dos caminos, igual que en Go: sin esto, cada sitio que
 * descifra el cuerpo tendría que preguntar si esta bóveda tiene claves de contenido, y
 * esa pregunta repartida es el patrón que este proyecto ya ha pagado tres veces.
 */
export function cadenaDeUnaClave(llave: string): CadenaDeClaves {
  return {
    esfinge: MARCA_DE_LA_CADENA,
    version: VERSION_DE_LA_CADENA,
    actual: "",
    claves: [{ id: "", n: 1, clave: llave, creada: "" }],
  };
}

/** Saca de los sobres de un documento la cadena, con la clave de bóveda. */
export async function laCadenaDe(sobres: Sobre[], llave: string): Promise<CadenaDeClaves> {
  const s = sobres.find((x) => x.tipo === RANURA_DE_CONTENIDO);
  if (!s) return cadenaDeUnaClave(llave);
  let claro: string;
  try {
    claro = deUtf8.decode(await abrirTexto(s.contenedor, llave));
  } catch {
    throw new Error(ERR_CADENA_MALA);
  }
  return leerCadena(claro);
}

// ------------------------------------------------- de qué lado salen las claves

/** De qué lado de una fusión salen los sobres que llevan claves. */
export type LadoDeLasClaves = "los-dos" | "aqui" | "alla";

/** Si ese tipo es la ranura de alguien con acceso. Lo mismo que `IDDeAcceso(tipo) != ""`. */
export function esDeAcceso(tipo: string): boolean {
  return tipo.startsWith("acceso:");
}

/**
 * Con qué clave queda cifrado el cuerpo cuando los dos lados están en épocas distintas,
 * o sea cuando uno de los dos ha rotado. **El mismo razonamiento que en Go**, incluido el
 * no mirar la base.
 */
export function ladoQueGanaLaEpoca(l: Sobre[], r: Sobre[]): LadoDeLasClaves {
  const sl = l.find((s) => s.tipo === RANURA_DE_CONTENIDO);
  const sr = r.find((s) => s.tipo === RANURA_DE_CONTENIDO);
  if (!sl && !sr) return "los-dos";
  if (!sr) return "aqui";
  if (!sl) return "alla";
  if (sr.creado > sl.creado || (sr.creado === sl.creado && compararComoGo(sr.contenedor, sl.contenedor) > 0)) return "alla";
  return "aqui";
}

/**
 * El lado de una fusión entera: «los dos» mientras las épocas sean la misma, que es el
 * caso normal y el único que existe en el formato 1.
 *
 * **Una sola función, y la llaman la fusión de verdad y la prueba cruzada.** Escrita dos
 * veces se desalinea sola, y aquí desalinearse significa un cuerpo cifrado con una clave
 * que la ranura de alguien no lleva dentro.
 */
export function ladoDeLaFusion(epocaL: string, epocaR: string, l: Sobre[], r: Sobre[]): LadoDeLasClaves {
  if (epocaL === epocaR) return "los-dos";
  return ladoQueGanaLaEpoca(l, r);
}
