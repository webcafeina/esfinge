/**
 * El aviso de qué datos toca la extensión, y si se ha aceptado.
 *
 * # Por qué existe
 *
 * **Lo exige la tienda de Chrome** (ADR 0033): un aviso visible y una aceptación
 * activa **dentro de la propia extensión**, aunque los datos no salgan del ordenador.
 * Firefox, además, enseña el suyo al instalar, por `data_collection_permissions`.
 *
 * Se pide **en el panel, la primera vez que se abre**, que es lo que eligió el
 * cliente. Y hasta aceptarlo la extensión no hace nada, que es lo que da sentido al
 * aviso: el trabajador de fondo no habla con Esfinge y el guion de la página no mira
 * ningún formulario.
 *
 * Se guarda en `storage.local` con la versión del aviso. **Si lo que dice el aviso
 * cambia, sube `VERSION_DEL_AVISO` y se vuelve a pedir**, que es lo que pide Chrome
 * cuando cambian las prácticas de datos después de instalar.
 */
import { api } from "./api";

export const CLAVE_DEL_CONSENTIMIENTO = "consentimiento";
/**
 * **2 desde la E2** (ADR 0040): con cuenta, la extensión se conecta al servidor de
 * cuentas y guarda la bóveda cifrada en el navegador, y el aviso lo dice.
 *
 * **3 desde la B4** (ADR 0043): desde el panel se puede mandar una copia a otra
 * persona, así que por el canal sale **la dirección de quien la recibe**, y a quien
 * no tenga cuenta el servidor le manda una invitación con la del que la manda. Es
 * un dato nuevo que sale del navegador y va a un tercero: sube y se vuelve a
 * preguntar, aunque eso cueste que todo el mundo vea el aviso otra vez.
 *
 * **4 desde la P2 de las llaves de acceso** (ADR 0048), y es la subida menos
 * discutible de las cuatro: **cambia dónde corre el código**. Hasta ahora todo lo de
 * Esfinge iba en el mundo aislado —la página no lo veía ni lo podía tocar— y ahora
 * hay una pieza **dentro** de cada página `https`, sustituyendo el método con el que
 * un sitio identifica a la gente. Que lo que sale sea una firma y nunca la clave no
 * quita que eso sea una práctica de datos distinta, y darlo por sabido sería
 * decidirlo por quien lo instaló antes.
 *
 * **5 desde la P3** (ADR 0048): la extensión ya no solo usa llaves, **las crea y las
 * guarda**. Sube aunque la 4 no haya llegado a nadie todavía —la versión que la lleva
 * está en revisión en las tiendas— y aunque no haya ninguna categoría de datos nueva:
 * lo que hay es una **frase del aviso que dejaría de ser cierta**, porque decía que lo
 * que sale hacia el sitio es la firma, y ahora también sale una clave pública recién
 * hecha. La regla de la ADR 0033 no dice «si cambian los datos», dice **si cambia lo
 * que dice el aviso**, y es mejor preguntar una vez de más que dejar aceptado un texto
 * que ya no describe lo que pasa.
 *
 * **6 desde la C6 de las bóvedas compartidas** (ADR 0052), y lo que la sube no es que
 * haya otra bóveda en el navegador: es **a dónde van los cambios**. Hasta aquí, todo lo
 * que la extensión guardaba iba a la cuenta de quien la usa y a ninguna otra; desde
 * ahora, lo que se guarde dentro de una bóveda que alguien te ha compartido **sube a la
 * cuenta de esa persona**. Un destino nuevo para datos que pone quien usa la extensión
 * es exactamente el caso que esta cuenta existe para no dar por sabido.
 */
export const VERSION_DEL_AVISO = 6;

/** vale dice si lo guardado es la aceptación del aviso de ahora. */
export function vale(guardado: unknown): boolean {
  return (
    typeof guardado === "object" &&
    guardado !== null &&
    (guardado as { version?: unknown }).version === VERSION_DEL_AVISO
  );
}

/** aceptado dice si se ha aceptado el aviso de ahora. **Nunca lanza**: ante la duda, no. */
export async function aceptado(): Promise<boolean> {
  try {
    const g = await api.storage.local.get(CLAVE_DEL_CONSENTIMIENTO);
    return vale(g[CLAVE_DEL_CONSENTIMIENTO]);
  } catch {
    return false;
  }
}

export async function aceptar(): Promise<void> {
  await api.storage.local.set({
    [CLAVE_DEL_CONSENTIMIENTO]: { version: VERSION_DEL_AVISO, cuando: new Date().toISOString() },
  });
}

/**
 * alAceptar avisa cuando se acepta el aviso en otro sitio —el panel—, para que una
 * página ya abierta o el icono empiecen sin recargar. Devuelve cómo dejar de oír.
 */
export function alAceptar(alSer: () => void): () => void {
  const oyente = (cambios: Record<string, { newValue?: unknown }>, zona: string) => {
    if (zona === "local" && vale(cambios[CLAVE_DEL_CONSENTIMIENTO]?.newValue)) alSer();
  };
  try {
    api.storage.onChanged.addListener(oyente);
  } catch {
    return () => {};
  }
  return () => {
    try {
      api.storage.onChanged.removeListener(oyente);
    } catch {
      /* ya no hay a quién quitar */
    }
  };
}
