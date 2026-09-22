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

import type { Boveda } from "./boveda";
import { conflicto, sinBoveda, type Cliente } from "./cliente";
import { fundir, type Fusion } from "./fundir";

/** Lo que se recuerda de la última vez: qué versión del servidor y qué serie de aquí la tenían. */
export type Recuerdo = { version: number; serie: number };

/** Dónde se guarda el recuerdo y la base (la versión común). En la extensión, `storage.local`. */
export interface Memoria {
  cargar(): Promise<{ recuerdo: Recuerdo; base: string | null }>;
  guardar(r: Recuerdo, base: string): Promise<void>;
}

export type Resultado = { version: number; bajo: boolean; subio: boolean; fusion?: Fusion };

export const ERR_RETROCESO =
  "El servidor tiene una bóveda más vieja que la que ya se vio aquí; no se sincroniza hasta mirarlo";
const INTENTOS = 5;

/** Si hay algo aquí sin subir. */
export async function pendiente(b: Boveda, m: Memoria): Promise<boolean> {
  const { recuerdo } = await m.cargar();
  return recuerdo.serie !== b.serie;
}

export async function pasada(b: Boveda, cliente: Cliente, token: string, m: Memoria): Promise<Resultado> {
  const r: Resultado = { version: 0, bajo: false, subio: false };
  let { recuerdo, base } = await m.cargar();
  for (let intento = 1; intento <= INTENTOS; intento++) {
    const siNoCoincide = base ? recuerdo.version : 0;
    let bajada: { datos: string; version: number } | null = null;
    let sobre: number;
    let vacia = false;
    try {
      bajada = await cliente.bajar(token, siNoCoincide);
    } catch (e) {
      if (!sinBoveda(e)) throw e;
      vacia = true;
    }
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
      const f = await fundir(b, bajada.datos, bajada.version, base);
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
      nueva = await cliente.subir(token, sobre, subida);
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
