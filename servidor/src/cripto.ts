// Lo poco de criptografía que hace el servidor: HMAC, SHA-256, azar y comparar sin
// filtrar tiempos. **Nunca descifra nada**: todo lo que guarda llega cifrado y
// sale igual (ADR 0035).

const codificador = new TextEncoder();

export function azar(n: number): Uint8Array {
	const b = new Uint8Array(n);
	crypto.getRandomValues(b);
	return b;
}

export function aBase64url(b: Uint8Array): string {
	let s = "";
	for (const x of b) s += String.fromCharCode(x);
	return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/** Devuelve null si no es base64url válido o no mide lo que se pide. */
export function deBase64url(s: unknown, largo?: number): Uint8Array | null {
	if (typeof s !== "string" || !/^[A-Za-z0-9_-]*$/.test(s)) return null;
	let b64 = s.replace(/-/g, "+").replace(/_/g, "/");
	while (b64.length % 4) b64 += "=";
	let crudo: string;
	try {
		crudo = atob(b64);
	} catch {
		return null;
	}
	const b = Uint8Array.from(crudo, (c) => c.charCodeAt(0));
	if (largo !== undefined && b.length !== largo) return null;
	return b;
}

export function aHex(b: Uint8Array): string {
	return Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
}

function bytes(d: string | Uint8Array): Uint8Array {
	return typeof d === "string" ? codificador.encode(d) : d;
}

export async function hmac(clave: string, datos: string | Uint8Array): Promise<Uint8Array> {
	const k = await crypto.subtle.importKey(
		"raw",
		codificador.encode(clave),
		{ name: "HMAC", hash: "SHA-256" },
		false,
		["sign"],
	);
	return new Uint8Array(await crypto.subtle.sign("HMAC", k, bytes(datos)));
}

export async function sha256(datos: string | Uint8Array): Promise<Uint8Array> {
	return new Uint8Array(await crypto.subtle.digest("SHA-256", bytes(datos)));
}

/** Compara en tiempo constante. Dos largos distintos son distintos, sin más. */
export function iguales(a: Uint8Array, b: Uint8Array): boolean {
	if (a.length !== b.length) return false;
	return crypto.subtle.timingSafeEqual(a, b);
}

/**
 * Un código de seis cifras repartido por igual.
 *
 * Con `% 1_000_000` sobre 32 bits los primeros códigos salen un poco más a menudo
 * que los últimos; se descartan los valores del tramo final que no llega entero.
 */
export function codigoDeSeisCifras(): string {
	const tope = Math.floor(0x1_0000_0000 / 1_000_000) * 1_000_000;
	for (;;) {
		const n = new DataView(azar(4).buffer).getUint32(0);
		if (n < tope) return String(n % 1_000_000).padStart(6, "0");
	}
}
