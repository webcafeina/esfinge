/**
 * Lo que se le ha dado a un agente de IA, en el espejo (ADR 0054).
 *
 * **La extensión no apunta nada aquí**: los agentes hablan con la ventana, no con el
 * navegador. Lo único que tiene que hacer con esta sección es **fundirla igual que Go**,
 * por la misma razón que `titular.ts` y `compartida.ts`: vive en `extra`, y una sección
 * que este lado no conoce se funde **como un bloque** —gana la del servidor entera—, así
 * que lo apuntado en la ventana mientras la extensión sincronizaba se perdería, y los
 * dos lados podrían quedarse con listas distintas y pasarse la bóveda sin fin.
 *
 * **Y no se purga aquí**, igual que en Go: el purgado va al abrir la bóveda, para que los
 * dos lados tiren lo mismo cuando les toca. Purgando al fundir, el lado que acaba de
 * abrir le quitaría al otro apuntes que el otro todavía no ha visto.
 */

import { compararComoGo, type ValorJSON } from "./canon";

/** Lo que caben. Espejo de `TopeDelRegistro`. */
export const TOPE_DEL_REGISTRO = 2000;

/** Una cosa que se le dio a un agente, o que se le negó. **Nunca lleva un secreto.** */
export type Apunte = {
  /** Del apunte, no de la entrada: es lo que deja fundir sin pisar. */
  id: string;
  cuando: string;
  quien: string;
  que: string;
  sobre?: string;
  titulo?: string;
  resultado: string;
  como?: string;
};

function esApunte(x: unknown): x is Apunte {
  if (typeof x !== "object" || x === null) return false;
  const a = x as Record<string, unknown>;
  return (
    typeof a.id === "string" &&
    typeof a.cuando === "string" &&
    typeof a.quien === "string" &&
    typeof a.que === "string" &&
    typeof a.resultado === "string"
  );
}

/** Los de una sección `extra`, si los lleva y se entienden. */
export function registroDe(extra: Record<string, ValorJSON> | undefined): Apunte[] {
  const lista = extra?.registro as unknown;
  if (!Array.isArray(lista)) return [];
  return lista.filter(esApunte);
}

/** Los deja en la sección, o la quita si no queda ninguno. Como el `omitempty` de Go. */
export function ponerRegistro(
  extra: Record<string, ValorJSON> | undefined,
  lista: Apunte[],
): Record<string, ValorJSON> | undefined {
  const out = { ...(extra ?? {}) };
  if (lista.length > 0) out.registro = lista as unknown as ValorJSON;
  else delete out.registro;
  return Object.keys(out).length > 0 ? out : undefined;
}

/**
 * Une dos registros. **Es la sección más fácil de fundir de la bóveda**, porque un
 * apunte es un hecho que pasó: no se edita, no se borra y no hay conflicto posible. Lo
 * único que hay que hacer es no perder ninguno y no duplicarlos.
 */
export function fundirRegistro(l: Apunte[], r: Apunte[]): Apunte[] {
  const vistos = new Set<string>();
  const out: Apunte[] = [];
  for (const lista of [l, r]) {
    for (const a of lista) {
      if (a.id === "" || vistos.has(a.id)) continue;
      vistos.add(a.id);
      out.push(a);
    }
  }
  // Por fecha y, a igualdad, por identificador: sin lo segundo, dos apuntes del mismo
  // segundo saldrían en cualquier orden y los dos lados darían bytes distintos.
  out.sort((x, y) => {
    const porFecha = compararComoGo(x.cuando, y.cuando);
    return porFecha !== 0 ? porFecha : compararComoGo(x.id, y.id);
  });
  return out.length > TOPE_DEL_REGISTRO ? out.slice(out.length - TOPE_DEL_REGISTRO) : out;
}
