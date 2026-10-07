/**
 * Las bóvedas de otras personas a las que tengo acceso, en el espejo (ADR 0052).
 *
 * **La extensión no gestiona accesos**: no los da, no los quita y no los acepta. Eso
 * pasa entero en la ventana. Lo que sí hace, desde la C6, es **abrirlas y
 * sincronizarlas**, así que aquí hay dos cosas que no se parecen:
 *
 *   - **La sección del cuerpo** (`compartidas`), que se funde igual que los
 *     proyectos y por la misma razón, que no es evidente: la sección vive en `extra`
 *     porque el `Contenido` de aquí no la conoce, y **una sección desconocida se
 *     funde como un bloque**, o sea que gana la del servidor entera. Con eso, un
 *     acceso aceptado en la ventana mientras la extensión sincronizaba se perdería;
 *     peor, los dos lados podrían quedarse con listas distintas y pasarse la bóveda
 *     sin fin, que es el fallo que la ADR 0038 vino a cerrar.
 *   - **La ranura sellada** que abre esa bóveda, que no está en esta sección sino
 *     dentro del fichero de la bóveda ajena. La clave no viaja por ningún otro
 *     sitio: ni en la lista, ni en el sobre que da el acceso.
 *
 * **Y de la ranura solo está la mitad: abrir.** Sellar no se mirrorea, y no es un
 * olvido — sellar es lo que hace quien **da** acceso, y eso vive en la ventana
 * (`SellarHaciaIdentidad` en `compartida.go`). Escribirlo aquí sería dejar en la
 * extensión la pieza que permite repartir la clave de una bóveda, sin que nada en la
 * extensión la vaya a llamar nunca.
 *
 * Lo que sostiene que esta mitad valga no es leer este fichero: es la cruzada
 * `accesoAbrir`, donde **Go sella y esto abre**. Un byte de diferencia en la etiqueta
 * del `info` o en lo autenticado y las bóvedas compartidas no se abren en el
 * navegador, sin que nada más se ponga rojo.
 */

import { compararComoGo, type ValorJSON } from "./canon";
import { abrirHpke } from "./hpke";
import { identidadDeSemilla, privadasDeSemilla } from "./identidad";
import { desdeBase64 } from "./esf1";

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
  /**
   * Cuándo el dueño quitó el acceso (ADR 0053). Con fecha, **el fichero ya no está**:
   * lo que queda es esta fila para poder decir cuál se fue y de quién era. En el
   * navegador solo se lee —para no ofrecer una bóveda que no está—; tacharla es de la
   * ventana, que es la que se enteró por el servidor.
   */
  retirada?: string;
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
  // **Una retirada gana siempre** (ADR 0053): el acceso lo quita el dueño, no mis
  // equipos, así que un lado que no se ha enterado todavía no puede deshacerlo.
  if ((l.retirada ?? "") > (out.retirada ?? "")) out.retirada = l.retirada;

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

// ------------------------------------------------------------------ la ranura sellada

const utf8 = new TextEncoder();

/**
 * Encabeza el tipo de la ranura de quien tiene acceso; detrás va su identificador de
 * miembro, hex de 8 bytes.
 *
 * **El tipo lleva dentro a quién es, y eso no es estilo.** El sello indexa los sobres
 * por su tipo (`sel.huellas[tipo]`, `sel.sobres[tipo]`) y la fusión los mete en un
 * mapa por tipo: dos ranuras del mismo tipo no hacen que sobre una, hacen que **la
 * bóveda no abra** y que la segunda desaparezca al sincronizar sin que nadie se
 * entere.
 */
const PREFIJO_ACCESO = "acceso:";

/**
 * La lápida: el tipo se queda y el contenedor se vacía.
 *
 * **No se borra el sobre**, porque quitar una ranura no es representable en la
 * fusión: une por tipo y conserva la que está en un solo lado. Sin lápida, el dueño
 * quita la ranura del revocado, sube, y **el primer equipo con una copia de antes la
 * resucita**.
 */
export const CODIFICACION_RETIRADA = "retirada";

/**
 * Separa estos sobres de los de compartir una entrada. Son la misma primitiva y **no
 * tienen por qué poder confundirse**: uno no se abre como el otro ni aunque alguien lo
 * intente. Tiene que ser byte a byte el `infoDeAcceso` de Go.
 */
const INFO_DE_ACCESO = "esfinge/acceso/v1";

/**
 * Encabeza el contenedor, como `ESF1` encabeza los otros, y **va dentro de los datos
 * autenticados**: así un contenedor de ésos no se puede hacer pasar por otra cosa
 * cambiándole las letras de delante.
 */
const MARCA_DEL_SELLADO = "ACC1";

/** La misma frase que `ErrSinAcceso` de Go: el mismo fallo dice lo mismo en los dos sitios. */
export const ERR_SIN_ACCESO = "No tienes acceso a esa bóveda";

/**
 * El tipo de ranura de un miembro.
 *
 * De lo que Go tiene al lado, aquí **no está lo que nadie lee**: ni `IDDeAcceso` —que
 * saca el miembro de un tipo, y en la extensión nada recorre las ranuras buscando
 * quién tiene acceso: eso lo dice la sección `compartidas` de mi propia bóveda— ni la
 * codificación `sobre-x25519-v1`, que es la que **escribe** quien sella. Un espejo con
 * piezas que nadie llama no es más fiel: es más código que puede desviarse sin que
 * nada lo note.
 */
export function tipoDeAcceso(id: string): string {
  return PREFIJO_ACCESO + id;
}

/**
 * Abre un contenedor sellado hacia la identidad de `semilla` — la de **mi** bóveda
 * personal. Espejo de `(*Boveda).AbrirSellado`.
 *
 * Todo lo que no cuadra sale como «no tienes acceso», incluido un contenedor roto: desde
 * fuera, «esto no es para ti» y «esto no se entiende» son lo mismo, y distinguirlos solo
 * ayudaría a quien prueba sobres ajenos. Go devuelve el error del base64 en ese caso; es
 * la única diferencia, y no la mira ninguna cruzada porque no son bytes del formato.
 */
export async function abrirSellado(sellado: string, semilla: Uint8Array): Promise<Uint8Array> {
  const partes = sellado.split(".");
  if (partes.length !== 3 || partes[0] !== MARCA_DEL_SELLADO) throw new Error(ERR_SIN_ACCESO);
  try {
    const enc = desdeBase64(partes[1]);
    const cuerpo = desdeBase64(partes[2]);
    const { cifrado: privada } = await privadasDeSemilla(semilla);
    // La pública propia entra en el contexto del KEM; en Go la saca `NewRecipient` de
    // la privada, y aquí hay que dársela porque WebCrypto no la deriva.
    const mia = await identidadDeSemilla(semilla);
    return await abrirHpke(
      privada,
      mia.cifrado,
      enc,
      utf8.encode(INFO_DE_ACCESO),
      utf8.encode(MARCA_DEL_SELLADO),
      cuerpo,
    );
  } catch {
    throw new Error(ERR_SIN_ACCESO);
  }
}
