/**
 * La forma canónica de un valor JSON, **byte a byte la misma que la de Go**
 * (`canon` en `internal/boveda/sincronizar.go`): claves en orden a todos los
 * niveles, sin espacios y sin escapar `<`, `>` ni `&`.
 *
 * La fusión la usa para comparar entradas y, en el último desempate, hace una
 * huella SHA-256 de esto. Si un byte saliera distinto aquí y en Go, cada lado
 * elegiría otra ganadora y los equipos se pasarían la bóveda sin fin. Por eso no
 * vale `JSON.stringify`, que se diferencia de Go en dos cosas:
 *
 * - Go escapa **U+2028 y U+2029** (` `, ` `) aunque no escape el HTML, y
 *   JavaScript no. Comprobado con Go 1.27 el 2026-09-22.
 * - Go ordena las claves de un mapa **por sus bytes UTF-8**; `sort()` ordena por
 *   unidades UTF-16, que no es lo mismo fuera del plano básico.
 *
 * Lo demás coincide: `\b \f \n \r \t` con su letra, el resto de controles como
 * `\u00xx` en minúscula, y `/` y DEL sin escapar.
 *
 * Una diferencia que **no** se arregla aquí, dicha para que nadie la busque: Go
 * conserva el texto de un número (`1.50`) y JavaScript no (`1.5`). Los números de
 * la bóveda son enteros —la revisión—, así que no pasa; un campo desconocido con
 * decimales podría dar una huella distinta.
 */

export type ValorJSON = null | boolean | number | string | ValorJSON[] | { [k: string]: ValorJSON };

/** Compara dos claves como Go: por sus bytes UTF-8, que es lo mismo que por punto de código. */
export function compararComoGo(a: string, b: string): number {
  const ia = a[Symbol.iterator]();
  const ib = b[Symbol.iterator]();
  for (;;) {
    const ra = ia.next();
    const rb = ib.next();
    if (ra.done || rb.done) return ra.done && rb.done ? 0 : ra.done ? -1 : 1;
    const ca = ra.value.codePointAt(0)!;
    const cb = rb.value.codePointAt(0)!;
    if (ca !== cb) return ca < cb ? -1 : 1;
  }
}

export function cadenaComoGo(s: string): string {
  let out = '"';
  for (const ch of s) {
    const c = ch.codePointAt(0)!;
    switch (ch) {
      case '"':
        out += '\\"';
        continue;
      case "\\":
        out += "\\\\";
        continue;
      case "\b":
        out += "\\b";
        continue;
      case "\f":
        out += "\\f";
        continue;
      case "\n":
        out += "\\n";
        continue;
      case "\r":
        out += "\\r";
        continue;
      case "\t":
        out += "\\t";
        continue;
    }
    if (c < 0x20 || c === 0x2028 || c === 0x2029) {
      out += "\\u" + c.toString(16).padStart(4, "0");
    } else if (c >= 0xd800 && c <= 0xdfff) {
      // Un sustituto suelto no es UTF-8 válido; Go lo habría leído como U+FFFD.
      out += "�";
    } else {
      out += ch;
    }
  }
  return out + '"';
}

/** La forma canónica de cualquier valor JSON. */
export function canonico(v: ValorJSON | undefined): string {
  if (v === null || v === undefined) return "null";
  if (typeof v === "string") return cadenaComoGo(v);
  if (typeof v === "number") return Number.isFinite(v) ? JSON.stringify(v) : "null";
  if (typeof v === "boolean") return v ? "true" : "false";
  if (Array.isArray(v)) return "[" + v.map((x) => canonico(x)).join(",") + "]";
  const claves = Object.keys(v).sort(compararComoGo);
  return "{" + claves.map((k) => cadenaComoGo(k) + ":" + canonico(v[k])).join(",") + "}";
}

/** SHA-256 en hexadecimal de un texto, como `huellaDe` en Go. */
export async function huella(texto: string): Promise<string> {
  const h = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(texto));
  return Array.from(new Uint8Array(h), (b) => b.toString(16).padStart(2, "0")).join("");
}
