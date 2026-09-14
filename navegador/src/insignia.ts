/**
 * Qué enseña el icono de la barra, decidido sin tocar el navegador.
 *
 * # Por qué es una función pura
 *
 * El icono refleja el estado de la extensión en cada pestaña —cuántas cuentas hay,
 * si ha rellenado, si la bóveda está cerrada, si algo falla— y lo pinta el
 * trabajador de fondo. **Lo que lo envuelve no se puede probar aquí** —hace falta
 * la extensión cargada, que sigue siendo deuda—, así que todo lo que decide va en
 * esta función y se prueba entera (`pruebas/insignia.spec.ts`). Al trabajador solo
 * le queda preguntar y pintar.
 *
 * # Y el orden de las comprobaciones es la decisión
 *
 * **Los problemas ganan a lo demás.** Una página rellenada hace un rato con la
 * bóveda ya cerrada enseña el candado, no el ✓: lo que importa ahora es que no
 * se puede rellenar.
 */
import type { Motivo, Respuesta } from "./protocolo";

export type Variante = "activo" | "apagado" | "cerrado";

export type QueMostrar = {
  icono: Variante;
  /** El texto de la insignia, o vacío para no enseñarla. */
  insignia: string;
  fondoInsignia: string;
  /** La frase que sale al pasar el ratón. */
  titulo: string;
};

/**
 * Los colores de las insignias. **No son tokens** —la insignia la dibuja el
 * navegador, no nuestro CSS— así que están medidos aparte, en
 * `internal/tema/extension_test.go`, que lee este fichero para comprobar que
 * mide los mismos.
 */
export const PIEDRA = "#2b2b31";
export const EXITO = "#1d6f31";
export const AVISO = "#8f5300";
export const TEXTO_DE_INSIGNIA = "#ffffff";

export function hostDe(url?: string): string {
  try {
    return new URL(url ?? "").hostname;
  } catch {
    return "";
  }
}

/** sePuedeRellenar: la misma regla que Esfinge aplica, solo https. */
export function sePuedeRellenar(url?: string): boolean {
  try {
    return new URL(url ?? "").protocol === "https:";
  } catch {
    return false;
  }
}

const sinInsignia = (icono: Variante, titulo: string): QueMostrar => ({
  icono,
  insignia: "",
  fondoInsignia: PIEDRA,
  titulo,
});

const conAviso = (titulo: string): QueMostrar => ({
  icono: "apagado",
  insignia: "!",
  fondoInsignia: AVISO,
  titulo,
});

function porMotivo(motivo: Motivo | undefined): QueMostrar {
  switch (motivo) {
    case "cerrada":
      return sinInsignia("cerrado", "Esfinge · La bóveda está cerrada");
    case "sin-boveda":
      return conAviso("Esfinge · Todavía no hay bóveda");
    case "sin-emparejar":
      return conAviso("Esfinge · Falta permitir este navegador en los Ajustes de Esfinge");
    case "sin-esfinge":
      return conAviso(
        "Esfinge · No se encuentra Esfinge: comprueba que está abierta y con el canal encendido",
      );
    case "demasiado":
      return conAviso("Esfinge · Demasiadas preguntas seguidas; espera un momento");
    case "origen":
      return sinInsignia("apagado", "Esfinge no rellena en esta página");
    default:
      return conAviso("Esfinge · Algo no ha ido bien; abre el panel para ver qué");
  }
}

/**
 * queMostrar decide el icono de una pestaña.
 *
 * Recibe lo que haya: la respuesta a «estado», la de «cuentas», o las dos; y si
 * Esfinge ha rellenado esa página.
 */
export function queMostrar(p: {
  url?: string;
  estado?: Respuesta;
  cuentas?: Respuesta;
  rellenado?: boolean;
  /**
   * Si se ha aceptado el aviso de datos (ADR 0033). **Falso gana a todo**, también a
   * las páginas donde no se rellena: hasta aceptarlo la extensión no hace nada en
   * ninguna, y lo único útil que puede decir el icono es dónde se empieza.
   */
  aceptado?: boolean;
}): QueMostrar {
  if (p.aceptado === false) return conAviso("Esfinge · Abre el panel para empezar");
  if (!sePuedeRellenar(p.url)) return sinInsignia("apagado", "Esfinge no rellena en esta página");
  const host = hostDe(p.url);

  // Primero los problemas.
  if (p.cuentas && !p.cuentas.ok) return porMotivo(p.cuentas.motivo);
  if (p.estado && !p.estado.ok) return porMotivo(p.estado.motivo);
  if (p.estado?.ok && !p.estado.estado?.existe) return porMotivo("sin-boveda");
  if (p.estado?.ok && !p.estado.estado?.abierta) return porMotivo("cerrada");

  if (p.rellenado) {
    return {
      icono: "activo",
      insignia: "✓",
      fondoInsignia: EXITO,
      titulo: "Esfinge ha rellenado esta página",
    };
  }

  if (p.cuentas) {
    const n = p.cuentas.cuentas?.length ?? 0;
    if (n === 0) return sinInsignia("activo", `Esfinge · Nada guardado de ${host}`);
    return {
      icono: "activo",
      insignia: n > 9 ? "9+" : String(n),
      fondoInsignia: PIEDRA,
      titulo: `Esfinge · ${n} ${n === 1 ? "cuenta" : "cuentas"} de ${host}`,
    };
  }

  if (p.estado) return sinInsignia("activo", `Esfinge · ${host}`);
  return conAviso("Esfinge · No se ha podido preguntar a Esfinge");
}
