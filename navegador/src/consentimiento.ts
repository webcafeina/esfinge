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
 */
export const VERSION_DEL_AVISO = 3;

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
