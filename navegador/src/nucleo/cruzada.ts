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
import { derivarAcceso, normalizarCorreo } from "./cuenta";
import { dominioDeOrigen, dominioDeSitio } from "./dominios";
import { canonEntrada, entradaDesde } from "./entrada";
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
