/**
 * El sobre de un envío, **el espejo de `internal/boveda/envio.go`** (ADR 0043).
 *
 * Lo único delicado aquí no es la criptografía —eso está en `hpke.ts`— sino
 * **escribir los mismos bytes que Go**: lo autenticado y lo firmado son JSON hecho
 * con `encoding/json` sobre un mapa, así que hay que imitar tres cosas suyas:
 * las claves en orden alfabético, los `[]byte` en base64 **estándar y con
 * relleno**, y nada de espacios. Si eso se desvía, la firma no cuadra y el sobre
 * no se abre; lo vigilan las pruebas cruzadas.
 */

import {
  comprobarFirma,
  firmarConSemilla,
  huellaDeIdentidad,
  identidadDeSemilla,
  privadasDeSemilla,
  SUITE,
  type Identidad,
} from "./identidad";
import { abrirHpke, nuevoEmisor } from "./hpke";
import { canonEntrada, entradaDesde, type Entrada } from "./entrada";

const utf8 = new TextEncoder();
const deUtf8 = new TextDecoder();

export const VERSION_DE_ENVIO = 1;
const INFO = "esfinge/envio/v1";

export type Envio = {
  esfinge: string;
  version: number;
  suite: string;
  de: { cifrado: string; firma: string };
  para: string;
  enc: string;
  cuerpo: string;
  firma: string;
};

/** Base64 estándar **con relleno**, que es lo que escribe Go para un `[]byte`. */
export function aBase64(b: Uint8Array): string {
  let s = "";
  for (const x of b) s += String.fromCharCode(x);
  return btoa(s);
}

export function deBase64(s: string): Uint8Array {
  const b = atob(s);
  return Uint8Array.from(b, (c) => c.charCodeAt(0));
}

/**
 * La cabecera tal como la escribe `json.Marshal` de un mapa en Go: claves en
 * orden alfabético y sin espacios.
 */
function loQueVaAutenticado(s: Envio): Uint8Array {
  const cabecera =
    `{"de":{"cifrado":${JSON.stringify(s.de.cifrado)},"firma":${JSON.stringify(s.de.firma)}},` +
    `"enc":${JSON.stringify(s.enc)},"esfinge":${JSON.stringify(s.esfinge)},` +
    `"para":${JSON.stringify(s.para)},"suite":${JSON.stringify(s.suite)},"version":${s.version}}`;
  return utf8.encode(cabecera);
}

function loQueSeFirma(s: Envio): Uint8Array {
  const cabecera = deUtf8.decode(loQueVaAutenticado(s));
  return utf8.encode(`{"cabecera":${cabecera},"cuerpo":${JSON.stringify(s.cuerpo)}}`);
}

/**
 * Prepara el sobre de `entrada` para `para`, firmado con la identidad de
 * `semilla`. **Copia sin identificador ni historial**, como en Go.
 */
export async function mandarEntrada(semilla: Uint8Array, entrada: Entrada, para: Identidad): Promise<Envio> {
  const mia = await identidadDeSemilla(semilla);
  if (para.suite !== mia.suite) throw new Error(`Esa identidad usa otro cifrado (${para.suite})`);

  const copia: Entrada = { ...entrada, id: "", revision: 0 };
  delete copia.historial;
  delete copia.papelera;
  delete copia.borradaEn;
  const claro = utf8.encode(canonEntrada(copia));

  const s: Envio = {
    esfinge: "envío",
    version: VERSION_DE_ENVIO,
    suite: mia.suite,
    de: { cifrado: aBase64(mia.cifrado), firma: aBase64(mia.firma) },
    para: aBase64(para.cifrado),
    enc: "",
    cuerpo: "",
    firma: "",
  };

  // El encapsulado va dentro de lo autenticado, así que se sabe primero y se
  // cifra después, igual que en Go.
  const emisor = await nuevoEmisor(para.cifrado, utf8.encode(INFO));
  s.enc = aBase64(emisor.enc);
  s.cuerpo = aBase64(emisor.sellar(loQueVaAutenticado(s), claro));
  s.firma = aBase64(await firmarConSemilla(semilla, loQueSeFirma(s)));
  return s;
}

/** Abre un sobre dirigido a la identidad de `semilla`. */
export async function abrirEnvio(
  semilla: Uint8Array,
  s: Envio,
): Promise<{ entrada: Entrada; de: Identidad }> {
  const mia = await identidadDeSemilla(semilla);
  if (s.version > VERSION_DE_ENVIO) throw new Error("Este envío lo hizo una versión de Esfinge más nueva");
  if (s.suite !== mia.suite || s.para !== aBase64(mia.cifrado)) throw new Error("Este envío no es para esta bóveda");

  const firmante = deBase64(s.de.firma);
  if (firmante.length !== 32 || !(await comprobarFirma(firmante, loQueSeFirma(s), deBase64(s.firma)))) {
    throw new Error("El envío no lo ha firmado quien dice");
  }

  const { cifrado: privada } = await privadasDeSemilla(semilla);
  const claro = await abrirHpke(
    privada,
    mia.cifrado,
    deBase64(s.enc),
    utf8.encode(INFO),
    loQueVaAutenticado(s),
    deBase64(s.cuerpo),
  );
  const entrada = entradaDesde(JSON.parse(deUtf8.decode(claro)));
  const cifrado = deBase64(s.de.cifrado);
  return {
    entrada,
    de: {
      suite: s.suite,
      cifrado,
      firma: firmante,
      huella: await huellaDeIdentidad(s.suite, cifrado, firmante),
    },
  };
}

export { SUITE };
