/**
 * El código de un solo uso (TOTP), igual que `internal/codigos` en Go: RFC 6238
 * sobre RFC 4226, con SHA-1, SHA-256 o SHA-512 de WebCrypto.
 *
 * Se prueba con los vectores de los dos RFC y **contra el de Go con la misma
 * semilla**, que es lo que de verdad importa: en su Mac se comprobó que el de Go
 * da lo mismo que Dashlane (2.15.0), y éste tiene que dar lo mismo que el de Go.
 */

export type Semilla = {
  clave: Uint8Array;
  digitos: number;
  /** En segundos. */
  periodo: number;
  algoritmo: "SHA1" | "SHA256" | "SHA512";
  cuenta: string;
};

const BASE32 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

/**
 * Base32 sin relleno, como `StdEncoding.WithPadding(NoPadding)` de Go. Cualquier
 * cadena de letras es base32 válida, así que lo único que se puede cazar es un
 * carácter de más o de menos: los restos posibles de un grupo de ocho son 0, 2,
 * 4, 5 y 7 (ver CLAUDE.md).
 */
function deBase32(texto: string): Uint8Array {
  const limpio = texto.replace(/[ \-\t\n\r]/g, "").replace(/=+$/, "").toUpperCase();
  if (!limpio) throw new Error("Aquí no hay ninguna semilla");
  if (![0, 2, 4, 5, 7].includes(limpio.length % 8)) throw new Error("A esa semilla le falta o le sobra algún carácter");
  const out: number[] = [];
  let acumulado = 0;
  let bits = 0;
  for (const c of limpio) {
    const i = BASE32.indexOf(c);
    if (i < 0) throw new Error("Esa semilla no está en base32; cópiala otra vez del servicio");
    acumulado = ((acumulado << 5) | i) & 0xffff;
    bits += 5;
    if (bits >= 8) {
      bits -= 8;
      out.push((acumulado >> bits) & 0xff);
    }
  }
  if (out.length === 0) throw new Error("Esa semilla se queda en nada al descifrarla");
  return Uint8Array.from(out);
}

/** Lee la semilla tal como se guarda: base32 suelto o una dirección `otpauth://totp/…`. */
export function leerSemilla(texto: string): Semilla {
  const t = texto.trim();
  if (!t) throw new Error("Aquí no hay ninguna semilla");
  if (!t.toLowerCase().startsWith("otpauth://")) {
    return { clave: deBase32(t), digitos: 6, periodo: 30, algoritmo: "SHA1", cuenta: "" };
  }
  let u: URL;
  try {
    u = new URL(t);
  } catch {
    throw new Error("Esa dirección de código de un solo uso no se entiende");
  }
  if (u.host.toLowerCase() !== "totp") throw new Error(`Esfinge solo sabe de códigos «totp», y ése es «${u.host}»`);
  const q = u.searchParams;
  const s: Semilla = {
    clave: deBase32(q.get("secret") ?? ""),
    digitos: 6,
    periodo: 30,
    algoritmo: "SHA1",
    cuenta: decodeURIComponent(u.pathname.replace(/^\//, "")),
  };
  const d = q.get("digits");
  if (d) {
    if (!/^[-+]?\d+$/.test(d)) throw new Error(`«${d}» no es un número de cifras`);
    s.digitos = Number(d);
  }
  const p = q.get("period");
  if (p) {
    if (!/^[-+]?\d+$/.test(p)) throw new Error(`«${p}» no es un número de segundos`);
    s.periodo = Number(p);
  }
  const a = q.get("algorithm");
  if (a) {
    const A = a.toUpperCase();
    if (A !== "SHA1" && A !== "SHA256" && A !== "SHA512") throw new Error(`Esfinge no sabe calcular códigos con «${a}»`);
    s.algoritmo = A;
  }
  if (s.digitos < 6 || s.digitos > 10) throw new Error(`Un código de un solo uso tiene entre 6 y 10 cifras, no ${s.digitos}`);
  if (s.periodo <= 0) throw new Error("El código tiene que durar algo más que nada");
  return s;
}

const HASH = { SHA1: "SHA-1", SHA256: "SHA-256", SHA512: "SHA-512" } as const;

/** El código en el instante `t`. */
export async function codigoEn(s: Semilla, t: Date): Promise<string> {
  const contador = BigInt(Math.floor(Math.floor(t.getTime() / 1000) / s.periodo));
  const mensaje = new Uint8Array(8);
  new DataView(mensaje.buffer).setBigUint64(0, contador, false);
  return codigoConContador(s, mensaje);
}

/** HOTP: el código de un contador de 8 bytes. Aparte para probar con los vectores de RFC 4226. */
export async function codigoConContador(s: Semilla, contador: Uint8Array): Promise<string> {
  const k = await crypto.subtle.importKey("raw", new Uint8Array(s.clave), { name: "HMAC", hash: HASH[s.algoritmo] }, false, ["sign"]);
  const suma = new Uint8Array(await crypto.subtle.sign("HMAC", k, new Uint8Array(contador)));
  const desde = suma[suma.length - 1] & 0x0f;
  const valor = new DataView(suma.buffer).getUint32(desde, false) & 0x7fffffff;
  // **Como Go, desbordamiento incluido**: allí el divisor es un `uint32`, y con
  // diez cifras 10¹⁰ no cabe y se queda en 10¹⁰ mod 2³². Raro, pero es lo que
  // enseña la ventana, y el panel tiene que enseñar lo mismo.
  const diez = Number(10n ** BigInt(s.digitos) % 2n ** 32n);
  return String(valor % diez).padStart(s.digitos, "0");
}

/** Los segundos que le quedan al código de ahora. */
export function quedan(s: Semilla, t: Date): number {
  const ms = s.periodo * 1000;
  return (ms - (t.getTime() % ms)) / 1000;
}
