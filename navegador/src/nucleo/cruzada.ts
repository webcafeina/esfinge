/**
 * La puerta por la que las pruebas de Go ejecutan el núcleo de la extensión
 * (ADR 0040). **No lo importa ningún punto de entrada de la extensión**: solo
 * `herramientas/cruzada.mjs`, que lo empaqueta al vuelo para Node.
 *
 * Cada orden recibe lo que manda Go en JSON y devuelve lo que Go compara. Lo que
 * se compara son **formas canónicas**, no objetos: la pregunta es si los dos lados
 * escribirían los mismos bytes.
 */

import { canonico, type ValorJSON } from "./canon";
import { Boveda, canonContenido, contenidoDesde, relojParaPruebas, type Sobre } from "./boveda";
import { codigoEn, leerSemilla } from "./codigos";
import { identidadDeSemilla } from "./identidad";
import { abrirEnvio, mandarEntrada, type Envio } from "./envio";
import { derivarAcceso, normalizarCorreo } from "./cuenta";
import { dominioDeOrigen, dominioDeSitio } from "./dominios";
import { crearLlave, firmarConLlave, origenDe, publicaEnCOSE, rpIdPermitido } from "./llaves";
import { cborDeMapa, claveCOSE, objetoDeAtestacion } from "./cbor";
import { atender } from "./fuente";
import {
  aBase64Url,
  datosDelAutenticador,
  datosDelAutenticadorAlCrear,
  datosDelCliente,
  loQueSeFirma,
} from "./afirmacion";
import { canonEntrada, entradaDesde, sinSecretos } from "./entrada";
import { fundir, fundirPiezas } from "./fundir";

const hex = (b: Uint8Array) => Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
const deHex = (s: string) => Uint8Array.from(s.match(/../g) ?? [], (h) => parseInt(h, 16));

type Caso = {
  l: unknown;
  r: unknown;
  b: unknown | null;
  sobresL: Sobre[] | null;
  sobresR: Sobre[] | null;
  sobresB: Sobre[] | null;
  ahora: string;
};

async function resumen(b: Boveda) {
  const entradas = [];
  for (const e of b.buscar("")) entradas.push(canonEntrada(b.ver(e.id)!));
  for (const e of b.papelera()) entradas.push(canonEntrada(b.ver(e.id)!));
  return { id: b.id, serie: b.serie, entradas, excluidos: b.excluidos(), posesion: hex(await b.posesion()) };
}

export async function ejecutar(p: { orden: string } & Record<string, unknown>): Promise<unknown> {
  switch (p.orden) {
    case "canon":
      return (p.entradas as unknown[]).map((e) => canonEntrada(entradaDesde(e)));

    // **Lo que se vacía antes de salir hacia el panel, comparado con Go.**
    //
    // Hace falta aparte de `canon` porque `canon` no puede cazar esto: la forma
    // canónica ordena las claves, así que un campo que se caiga de `CAMPOS` vuelve
    // por `extra` y **los bytes salen idénticos**. Lo que cambia es que
    // `sinSecretos` ya no lo encuentra donde lo borra, y entonces el secreto
    // **viaja al panel** sin que nada se ponga rojo. Comprobado quitando dos campos.
    case "sinSecretos":
      return (p.entradas as unknown[]).map((e) => canonEntrada(sinSecretos(entradaDesde(e))));

    case "fundir": {
      const out = [];
      for (const c of p.casos as Caso[]) {
        const r = await fundirPiezas(
          contenidoDesde(c.l),
          contenidoDesde(c.r),
          c.b === null ? null : contenidoDesde(c.b),
          c.sobresL ?? [],
          c.sobresR ?? [],
          c.sobresB ?? [],
          new Date(c.ahora),
        );
        out.push({
          contenido: canonContenido(r.contenido),
          sobres: canonico(r.sobres as unknown as ValorJSON),
          traidas: r.traidas,
          conflictos: r.conflictos,
        });
      }
      return out;
    }

    case "abrir":
      return resumen(await Boveda.abrir(p.texto as string, p.llave as string));

    case "modificar": {
      const b = await Boveda.abrir(p.texto as string, p.llave as string);
      for (const e of (p.poner as unknown[]) ?? []) await b.poner(entradaDesde(e));
      for (const id of (p.borrar as string[]) ?? []) await b.borrar(id);
      for (const d of (p.excluir as string[]) ?? []) await b.excluir(d);
      return { texto: b.documento(), ...(await resumen(b)) };
    }

    case "crear": {
      const { boveda, recuperacion } = await Boveda.crear(p.maestra as string);
      for (const e of (p.poner as unknown[]) ?? []) await boveda.poner(entradaDesde(e));
      if (!((p.poner as unknown[]) ?? []).length) await boveda.guardar();
      return { texto: boveda.documento(), recuperacion, ...(await resumen(boveda)) };
    }

    case "subida": {
      const b = await Boveda.abrir(p.texto as string, p.llave as string);
      return { texto: (await b.prepararSubida(p.version as number)).texto };
    }

    case "fundirBoveda": {
      relojParaPruebas(() => new Date(p.ahora as string));
      try {
        const b = await Boveda.abrir(p.local as string, p.llave as string);
        const f = await fundir(b, p.remoto as string, p.version as number, (p.base as string) || null);
        return { texto: b.documento(), fusion: f, ...(await resumen(b)) };
      } finally {
        relojParaPruebas(null);
      }
    }

    case "codigos": {
      const out = [];
      for (const c of p.casos as { semilla: string; unix: number }[]) {
        try {
          out.push(await codigoEn(leerSemilla(c.semilla), new Date(c.unix * 1000)));
        } catch (e) {
          out.push("error: " + (e as Error).message);
        }
      }
      return out;
    }

    // La identidad para compartir (ADR 0043): las mismas llaves y la misma huella
    // que Go para la misma semilla, o lo que se mande no lo podrá abrir nadie.
    case "identidad": {
      const out = [];
      for (const semilla of p.semillas as string[]) {
        const i = await identidadDeSemilla(deHex(semilla));
        out.push({ cifrado: hex(i.cifrado), firma: hex(i.firma), huella: i.huella, suite: i.suite });
      }
      return out;
    }

    // Los sobres de los envíos (ADR 0043): Go cierra y esto abre, y al revés. Si
    // los bytes que se firman no son los mismos, nada de esto cuadra.
    case "envioAbrir": {
      const { entrada, de } = await abrirEnvio(deHex(p.semilla as string), p.sobre as Envio);
      return { entrada: canonEntrada(entrada), huella: de.huella };
    }

    case "envioSellar": {
      const para = await identidadDeSemilla(deHex(p.paraSemilla as string));
      return await mandarEntrada(deHex(p.semilla as string), entradaDesde(p.entrada), para);
    }

    // La identidad **tal como la guarda la bóveda**: Go la crea, la extensión abre
    // esa bóveda y tiene que sacar la misma huella. Cazaría, por ejemplo, que uno
    // escribiera la semilla en base64 con relleno y el otro sin él.
    case "identidadDeBoveda": {
      const b = await Boveda.abrir(p.texto as string, p.llave as string);
      return { huella: (await identidadDeSemilla(await b.semillaDeIdentidad())).huella };
    }

    case "acceso":
      return hex(
        await derivarAcceso(p.maestra as string, deHex(p.sal as string), p.parametros as { memoria: number; pasadas: number; paralelismo: number }),
      );

    case "dominios":
      return (p.casos as string[]).map((c) => {
        let origen: string;
        try {
          origen = dominioDeOrigen(c);
        } catch {
          origen = "error";
        }
        return { origen, sitio: dominioDeSitio(c) };
      });

    // **Con qué `rpId` se puede firmar** (ADR 0048). Es la pieza que decide si una
    // llave de acceso firma para el sitio bueno, y la que caza que la lista de
    // sufijos de `tldts` y la de `golang.org/x/net` hayan dejado de coincidir.
    case "rpid":
      return (p.casos as { rpId: string; origen: string }[]).map((c) => rpIdPermitido(c.rpId, c.origen) ?? "");

    // **Los bytes que se firman**, que el sitio verifica byte a byte (ADR 0048).
    // Si Go y la extensión los escriben distinto, una llave creada con cuenta no
    // sirve sin ella, y al revés.
    case "firmado": {
      const out = [];
      for (const c of p.casos as { tipo: string; reto: string; origen: string; rpId: string; banderas: number }[]) {
        const cliente = datosDelCliente(c.tipo as "webauthn.get", deHex(c.reto), c.origen);
        const autenticador = await datosDelAutenticador(c.rpId, c.banderas);
        out.push({
          cliente: hex(cliente),
          autenticador: hex(autenticador),
          firmado: hex(await loQueSeFirma(autenticador, cliente)),
        });
      }
      return out;
    }

    // **El orden canónico de un mapa CBOR, ejercitado a propósito.**
    //
    // Ninguno de los mapas que Esfinge escribe de verdad lo distingue: en el COSE
    // todas las claves miden un byte, y en el objeto de atestación ordenar por largo
    // y ordenar por bytes dan el mismo resultado. Lo descubrió una mutación —quitar
    // la comparación por largo y la cruzada seguía verde—, así que la regla se prueba
    // aquí con claves donde **sí** discrepa. Es la regla de CTAP2, y el día que haya
    // un mapa con claves de largos distintos tiene que estar bien ya.
    case "cbor": {
      const out = [];
      for (const c of p.casos as { pares: [string, string][] }[]) {
        // `?? []` porque un slice nulo de Go llega como `null`, y el caso vacío es uno de los casos.
        out.push(hex(cborDeMapa((c.pares ?? []).map(([k, v]) => [deHex(k), deHex(v)]))));
      }
      return out;
    }

    // **Los bytes de crear una llave** (ADR 0048, P3): el COSE de la pública, el
    // `authenticatorData` con la credencial dentro y el objeto de atestación. Son
    // los que **el sitio se guarda para siempre**, así que una divergencia entre
    // los dos lados es una llave que solo sirve donde se creó.
    case "atestacion": {
      const out = [];
      for (const c of p.casos as { rpId: string; banderas: number; idCredencial: string; x: string; y: string }[]) {
        const cose = claveCOSE(deHex(c.x), deHex(c.y));
        const datos = await datosDelAutenticadorAlCrear(c.rpId, c.banderas, deHex(c.idCredencial), cose);
        out.push({ cose: hex(cose), datos: hex(datos), objeto: hex(objetoDeAtestacion(datos)) });
      }
      return out;
    }

    // **El COSE de una pública dada**, para los vectores fijos de la P3. Las llaves
    // al azar casi nunca tienen una coordenada que empiece por cero —una vez de cada
    // 256—, así que ese caso se fija a mano y no se deja al azar.
    case "coseDeUnaPublica": {
      const out = [];
      for (const c of p.casos as { spki: string }[]) {
        out.push(hex(await publicaEnCOSE(deHex(c.spki))));
      }
      return out;
    }

    // **El COSE de una pública recién creada aquí** (P3), para que Go compruebe que
    // puede verificar con ella. Es la mitad que dice que una llave creada con cuenta
    // sirve sin ella: lo que el sitio guarda es esto, y si los dos lados sacaran
    // coordenadas distintas de la misma clave, la llave solo valdría donde se creó.
    case "coseDeUnaLlaveNueva": {
      const out = [];
      for (let i = 0; i < (p.cuantas as number); i++) {
        const par = await crearLlave();
        out.push({
          privada: hex(par.privada),
          publica: hex(par.publica),
          cose: hex(await publicaEnCOSE(par.publica)),
        });
      }
      return out;
    }

    // **Firmar aquí y verificar allí, y al revés** (ADR 0048). Es lo único que
    // dice que la conversión de P1363 a DER está bien sin un sitio de verdad: los
    // bytes de una firma ECDSA cambian en cada llamada, así que compararlos no
    // vale para nada y lo que hay que comparar es que la otra parte la acepte.
    case "firmarLlave": {
      const out = [];
      for (const c of p.casos as { privada: string; datos: string }[]) {
        out.push(hex(await firmarConLlave(deHex(c.privada), deHex(c.datos))));
      }
      return out;
    }

    case "verificarFirma": {
      const out = [];
      for (const c of p.casos as { publica: string; datos: string; firma: string }[]) {
        try {
          const k = await crypto.subtle.importKey(
            "spki",
            deHex(c.publica),
            { name: "ECDSA", namedCurve: "P-256" },
            false,
            ["verify"],
          );
          out.push(
            await crypto.subtle.verify({ name: "ECDSA", hash: "SHA-256" }, k, deHex(c.firma), deHex(c.datos)),
          );
        } catch {
          out.push(false);
        }
      }
      return out;
    }

    case "crearLlave": {
      const { privada, publica } = await crearLlave();
      return { privada: hex(privada), publica: hex(publica) };
    }

    // **El verbo entero de firmar una llave de acceso** (ADR 0048): se guarda una
    // llave en una bóveda de verdad, se pide firmar como lo pediría el navegador, y
    // **Go verifica la firma**. Es lo único que dice que todo el camino está bien
    // —el `clientDataJSON`, el `authenticatorData`, el DER— sin un sitio de verdad,
    // porque WebCrypto no sabe verificar DER y aquí no hay quien lo haga.
    case "afirmarLlave": {
      const par = await crearLlave();
      const { boveda } = await Boveda.crear(p.maestra as string);
      const puesta = await boveda.poner({
        id: "",
        tipo: "llave",
        titulo: "GitHub",
        rpId: p.rpId as string,
        idCredencial: "Y3JlZC0x",
        idUsuario: "dXN1LTE",
        nombreVisible: "yo@ejemplo.com",
        algoritmo: -7,
        clavePrivada: aBase64Url(par.privada),
        creada: "",
        cambiada: "",
      });
      const r = await atender(
        {
          version: 1,
          que: "firmar-llave",
          origen: p.origen as string,
          rpId: p.rpId as string,
          id: puesta.id,
          reto: p.reto as string,
        },
        { existe: true, boveda },
      );
      return { publica: hex(par.publica), ok: r.ok, afirmacion: r.afirmacion ?? null };
    }

    // **El origen tal como lo escribe el navegador**, que va dentro del
    // `clientDataJSON` y que el sitio compara. Cuatro caracteres de diferencia y la
    // firma se rechaza sin decir por qué.
    case "origen":
      return (p.casos as string[]).map((c) => origenDe(c));

    case "correos":
      return (p.correos as string[]).map((c) => {
        try {
          return normalizarCorreo(c);
        } catch (e) {
          return "error: " + (e as Error).message;
        }
      });
  }
  throw new Error(`Orden desconocida: ${p.orden}`);
}
