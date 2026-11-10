/**
 * Una pasada de sincronización, **la misma que `Sincronizador.pasada` en Go**
 * (`internal/sincro/sincro.go`, ADR 0038): bajar lo del servidor si ha cambiado,
 * fundirlo con lo de aquí contra la última versión común, y subir si hace falta,
 * volviendo a empezar si otro equipo ha subido en medio.
 *
 * Lo que aquí no está —el reloj, esperar tras guardar, reintentar con espera
 * creciente— lo pone quien la llame, porque en el navegador lo marcan las alarmas
 * y no un bucle: el trabajador de fondo se muere solo cada pocos minutos.
 */

import { ErrorBoveda, type Boveda } from "./boveda";
import { conflicto, sinBoveda, type Cliente } from "./cliente";
import { fundir, type Fusion } from "./fundir";

/** Lo que se recuerda de la última vez: qué versión del servidor y qué serie de aquí la tenían. */
export type Recuerdo = { version: number; serie: number };

/** Dónde se guarda el recuerdo y la base (la versión común). En la extensión, `storage.local`. */
export interface Memoria {
  cargar(): Promise<{ recuerdo: Recuerdo; base: string | null }>;
  guardar(r: Recuerdo, base: string): Promise<void>;
}

export type Resultado = {
  version: number;
  bajo: boolean;
  subio: boolean;
  fusion?: Fusion;
  /**
   * Con qué permiso me deja el servidor, cuando la bóveda es de otra persona y el
   * servidor lo ha dicho (ADR 0058). Vacío en la personal y en un proyecto, donde no
   * hay permiso que mirar.
   *
   * Sale de aquí y no se aplica aquí: `pasada` no sabe de bóvedas personales ni de
   * listas de compartidas, que es donde vive lo que hay que poner al día.
   */
  permiso?: string;
};

export const ERR_RETROCESO =
  "El servidor tiene una bóveda más vieja que la que ya se vio aquí; no se sincroniza hasta mirarlo";
const INTENTOS = 5;

/** Si hay algo aquí sin subir. */
export async function pendiente(b: Boveda, m: Memoria): Promise<boolean> {
  const { recuerdo } = await m.cargar();
  return recuerdo.serie !== b.serie;
}

export async function pasada(
  b: Boveda,
  cliente: Cliente,
  token: string,
  m: Memoria,
  /** Acepta una fusión que se lleve más de la mitad de las entradas: lo pide una persona. */
  aunqueBorreMucho = false,
  /** Qué bóveda de la cuenta se sincroniza: vacío, la personal (ADR 0050). */
  ref = "",
  /**
   * De quién es esa bóveda (ADR 0052): vacío soy yo, que es todo lo de antes. Con
   * dueño, la bóveda vive en el objeto de **otra cuenta** y la dirección es otra.
   *
   * Aquí y no en un adaptador como en Go, donde `internal/sincro` no sabe que hay más
   * de una bóveda: este lado ya sabía de `ref` desde la ADR 0050, así que esconder el
   * dueño detrás de una indirección sería ordenar media casa.
   */
  dueno = "",
  /**
   * Se llama cuando lo que baja va cifrado con una clave que esta bóveda no tiene, o sea
   * cuando **el dueño ha rotado** (ADR 0059). Lo pone quien tiene la bóveda personal
   * delante, porque la ranura de quien tiene acceso solo la abre su identidad. Sin esto,
   * una rotación le dejaría la sincronización parada para siempre.
   */
  recogerLaCadena: ((datos: string) => Promise<void>) | null = null,
): Promise<Resultado> {
  // **Lo que diga el servidor del permiso se apunta de paso** (ADR 0058), aunque la
  // bajada no traiga cuerpo: ahí está el único camino por el que un permiso cambiado
  // llega al equipo de quien lo tiene.
  let permiso = "";
  const bajar = async (siNoCoincide: number) => {
    if (dueno === "") return cliente.bajar(token, siNoCoincide, ref);
    const r = await cliente.bajarCompartida(token, dueno, ref, siNoCoincide);
    if (r.permiso !== "") permiso = r.permiso;
    return r.cuerpo;
  };
  const subir = (siCoincide: number, datos: string) =>
    dueno === ""
      ? cliente.subir(token, siCoincide, datos, ref)
      : cliente.subirACompartida(token, dueno, ref, siCoincide, datos);
  const r: Resultado = { version: 0, bajo: false, subio: false };
  let { recuerdo, base } = await m.cargar();
  for (let intento = 1; intento <= INTENTOS; intento++) {
    const siNoCoincide = base ? recuerdo.version : 0;
    let bajada: { datos: string; version: number } | null = null;
    let sobre: number;
    let vacia = false;
    try {
      bajada = await bajar(siNoCoincide);
    } catch (e) {
      if (!sinBoveda(e)) throw e;
      vacia = true;
    }
    // **En un solo sitio y no en cada salida**: de aquí abajo hay tres `return r`, y
    // acordarse en los tres es como se queda uno sin él.
    r.permiso = permiso;
    if (vacia) {
      if (recuerdo.version > 0) throw new Error(`${ERR_RETROCESO} (aquí se vio la ${recuerdo.version} y allí no hay ninguna)`);
      sobre = 0; // la cuenta está vacía: lo de aquí es la primera versión
    } else if (bajada === null) {
      r.version = recuerdo.version;
      if (b.serie === recuerdo.serie) return r; // nada nuevo en ningún lado
      sobre = recuerdo.version;
    } else {
      if (bajada.version < recuerdo.version) {
        throw new Error(`${ERR_RETROCESO} (aquí se vio la ${recuerdo.version} y allí dice la ${bajada.version})`);
      }
      let f;
      try {
        f = await fundir(b, bajada.datos, bajada.version, base, { aunqueBorreMucho });
      } catch (e) {
        // **Una rotación no puede dejar a nadie parado para siempre** (ADR 0059): se
        // recoge la cadena nueva por donde se puede y se vuelve a fundir. Una vez: si
        // tampoco con ella, lo que falla es otra cosa y se dice.
        if (!(e instanceof ErrorBoveda) || e.codigo !== "epoca-que-no-se-tiene" || !recogerLaCadena) throw e;
        await recogerLaCadena(bajada.datos);
        f = await fundir(b, bajada.datos, bajada.version, base, { aunqueBorreMucho });
      }
      r.fusion = f;
      r.bajo = f.cambio;
      recuerdo = { version: bajada.version, serie: f.serie };
      base = bajada.datos;
      await m.guardar(recuerdo, base);
      r.version = bajada.version;
      if (!f.subir) return r;
      sobre = bajada.version;
    }

    // La serie sale **junto con lo que se sube**: leída aparte, un guardado en medio
    // se daría por subido sin estarlo (lo mismo que `PrepararSubida` en Go).
    const { texto: subida, serie } = await b.prepararSubida(sobre + 1);
    let nueva: number;
    try {
      nueva = await subir(sobre, subida);
    } catch (e) {
      if (conflicto(e)) continue; // otro equipo ha subido en medio: a bajar otra vez
      throw e;
    }
    if (nueva !== sobre + 1) {
      throw new Error(`El servidor ha guardado la bóveda como la versión ${nueva}, y se esperaba la ${sobre + 1}`);
    }
    recuerdo = { version: nueva, serie };
    base = subida;
    await m.guardar(recuerdo, base);
    r.version = nueva;
    r.subio = true;
    return r;
  }
  throw new Error("Otros equipos están subiendo cambios sin parar; se probará más tarde");
}
