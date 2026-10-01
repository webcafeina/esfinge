/**
 * La bandera que el mundo aislado **empuja** al mundo principal: «en este sitio hay
 * algo que ofrecer» (ADR 0048).
 *
 * Es lo que permite al `shim` ceder **en la misma vuelta del bucle de eventos** cuando
 * aquí no hay nada, sin preguntarle a nadie: esperar es lo que agota la activación de
 * usuario y rompe el inicio de sesión de quien no usa Esfinge.
 *
 * Vive aparte de `pagina.ts` por una razón concreta: **para poder probar que insiste**.
 * Allí el `pedir` es el canal de verdad y no hay forma de hacerle fallar la primera vez.
 */
import type { Respuesta } from "./protocolo";
import type { Aviso } from "./puente";

/**
 * Lo que el trabajador de fondo le manda a una pestaña para que **vuelva a mirar**.
 *
 * Vive aquí, de este lado, para que no haya dos literales que tengan que coincidir: lo
 * manda `fondo.ts` y lo escucha `pagina.ts`.
 */
export const MIRA_OTRA_VEZ = "esfinge:mira-otra-vez";

/**
 * Cuántas veces se vuelve a preguntar cuando **no se ha podido** preguntar.
 *
 * Tres, y solo en ese caso: una respuesta que dice «aquí no hay llaves» es una
 * respuesta y no se repite. Insistir sobre eso gastaría una pregunta del freno en cada
 * carga de cada sitio del mundo, que es casi siempre el caso.
 */
export const INTENTOS = 3;

/** Lo que se espera entre intentos, en milisegundos. */
export const ESPERAS = [500, 2000];

/**
 * **Que no haya respuesta no es que la respuesta sea «no».**
 *
 * `pedir` no lanza cuando el trabajador de fondo no contesta: devuelve `ok: false` con
 * un `error` y **sin `motivo`**, porque el motivo lo pone el núcleo y aquí no ha llegado
 * a hablar nadie. Tomar eso por un «no» es lo que dejaba la bandera en falso **para
 * siempre** en esa pestaña: el `shim` cedía en todas las llamadas y el banner no salía
 * aunque hubiera llaves, sin un error en ningún sitio.
 *
 * Y pasa de verdad: el trabajador de MV3 se muere cada pocos minutos, así que la primera
 * pregunta de una página puede llegarle dormido. Se vio el 2026-10-01 en la tanda
 * completa de `make comprobar`, con el banner sin salir en treinta segundos.
 *
 * Lo que sí es concluyente es cualquier `motivo` —la bóveda cerrada, el sitio que no
 * encaja, el aviso de datos sin aceptar— y, por descontado, un `ok`.
 */
export function seHaPodidoPreguntar(r: Respuesta): boolean {
  return r.ok || r.motivo !== undefined;
}

/** Lo que se le dice al mundo principal a partir de una respuesta del núcleo. */
export function avisoDe(r: Respuesta): Aviso {
  return {
    // **Con la bóveda cerrada también hay algo que ofrecer**, y eso es `quizas`: lo
    // pone el trabajador desde la lista de dominios, porque cerrada no hay a quién
    // preguntar. Sin esta rama el shim cedería antes de preguntar y el banner que
    // ofrece abrir la bóveda no podría salir nunca.
    hay: r.ok ? (r.llaves ?? []).length > 0 : r.quizas === true,
    // **Crear es otra bandera**: Esfinge se ofrece siempre a crear, así que no depende
    // de que haya llaves aquí. Lo que sí la apaga es el interruptor de Ajustes, y con
    // la bóveda cerrada se puede igual —el banner ofrece abrirla—.
    sePuedeCrear: r.ok ? r.puedeCrear === true : r.motivo === "cerrada",
  };
}

/**
 * Pregunta y empuja la bandera, **insistiendo solo si no se ha podido preguntar**.
 *
 * Si se agotan los intentos se manda `false`, que es lo que ya vale por defecto al otro
 * lado: ante la duda, que ceda.
 */
export async function empujarLaBandera(
  pedir: () => Promise<Respuesta>,
  mandar: (a: Aviso) => void,
  esperar: (ms: number) => Promise<void> = (ms) => new Promise((x) => setTimeout(x, ms)),
): Promise<void> {
  for (let i = 0; i < INTENTOS; i++) {
    let r: Respuesta;
    try {
      r = await pedir();
    } catch {
      r = { ok: false, error: "No se ha podido preguntar" };
    }
    if (seHaPodidoPreguntar(r)) return mandar(avisoDe(r));
    if (i < INTENTOS - 1) await esperar(ESPERAS[Math.min(i, ESPERAS.length - 1)]);
  }
  mandar({ hay: false, sePuedeCrear: false });
}
