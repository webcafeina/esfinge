// Lo que viaja entre Esfinge y el servidor, y las reglas que lo validan.
//
// **Todas las frases que puede ver una persona empiezan en mayúscula**, como en el
// resto de Esfinge: el cliente las enseña tal cual.

import type { Cuenta } from "./cuenta";

export interface Env {
	BD: D1Database;
	CUENTAS: DurableObjectNamespace<Cuenta>;
	FRENO_ENTRAR: RateLimit;
	FRENO_ALTAS: RateLimit;
	ENTORNO: string;
	JURISDICCION: string;
	REGISTRO: string;
	REMITENTE: string;
	PIMIENTA: string;
	SECRETO_PRELOGIN: string;
	RESEND_API_KEY?: string;
	TOPE_ALTAS_DIA?: string;
	TOPE_ALTAS_IP_DIA?: string;
	TOPE_CODIGOS_IP_DIA?: string;
	TOPE_ENVIOS_DIA?: string;
	TOPE_ENVIOS_IP_DIA?: string;
	TOPE_INVITACIONES_DIA?: string;
}

/** Lo que devuelve cada operación de una cuenta: o datos, o un error con su estado. */
export type Resultado<T> = { ok: true; datos: T } | { ok: false; estado: number; error: string };

export function bien<T>(datos: T): Resultado<T> {
	return { ok: true, datos };
}

export function mal<T = never>(estado: number, error: string): Resultado<T> {
	return { ok: false, estado, error };
}

export interface Argon2 {
	memoria: number; // KiB
	pasadas: number;
	paralelismo: number;
}

/**
 * Los parámetros por defecto, que son `cripto.PerfilInteractivo` de Go.
 *
 * Son también los que se dan a un correo sin cuenta en la pre-entrada: si fueran
 * otros, distinguirían las cuentas de verdad de las inventadas.
 */
export const ARGON2_POR_DEFECTO: Argon2 = { memoria: 64 * 1024, pasadas: 3, paralelismo: 4 };

/**
 * Los mismos límites que comprueba el cliente, por los dos lados.
 *
 * **El de abajo es el que importa**: un servidor que devolviera un coste
 * bajísimo haría que la clave de acceso se pudiera atacar sin conexión. El
 * cliente rechaza cualquier cosa por debajo, y el servidor no la acepta al
 * registrar para que nadie se encuentre una cuenta que su propia Esfinge no abre.
 */
export function argon2Valido(a: unknown): a is Argon2 {
	if (typeof a !== "object" || a === null) return false;
	const { memoria, pasadas, paralelismo } = a as Record<string, unknown>;
	return (
		Number.isInteger(memoria) &&
		Number.isInteger(pasadas) &&
		Number.isInteger(paralelismo) &&
		(memoria as number) >= 64 * 1024 &&
		// Y el de arriba, 256 MiB, es el que el cliente acepta al abrir un sobre
		// (revisión del 2026-09-23): guardar un coste mayor sería dejar registrada
		// una cuenta que ninguna Esfinge puede abrir.
		(memoria as number) <= 256 * 1024 &&
		(pasadas as number) >= 3 &&
		(pasadas as number) <= 16 &&
		(paralelismo as number) >= 1 &&
		(paralelismo as number) <= 16
	);
}

/**
 * El correo, en la forma que se guarda y se compara.
 *
 * Minúsculas y sin espacios, y nada más: quitar puntos o lo que va tras un «+»
 * es una regla de Gmail, no del correo, y aplicarla uniría cuentas de personas
 * distintas. **El cliente hace exactamente lo mismo** antes de mandarlo.
 */
export function normalizarCorreo(c: unknown): string | null {
	if (typeof c !== "string") return null;
	const n = c.trim().normalize("NFC").toLowerCase();
	if (n.length < 3 || n.length > 254) return null;
	if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(n)) return null;
	return n;
}

export function nombreDeEquipo(n: unknown): string {
	if (typeof n !== "string") return "Equipo sin nombre";
	const limpio = n.replace(/[\u0000-\u001f\u007f]/g, "").trim().slice(0, 80);
	return limpio || "Equipo sin nombre";
}

/**
 * El conjunto de HPKE con el que se cifra hacia una identidad (ADR 0043).
 *
 * El servidor **no cifra ni descifra nada**; solo lo necesita para inventarse unas
 * llaves creíbles cuando le preguntan por un correo que no tiene cuenta, y así no
 * delatar quién la tiene.
 */
export const SUITE_POR_DEFECTO = "DHKEM(X25519)/HKDF-SHA256/ChaCha20-Poly1305";

/** Tope de una bóveda subida. Una de veinte mil entradas ronda los 4 MB. */
export const TAMANO_MAXIMO = 8 * 1024 * 1024;

export const MINUTO = 60 * 1000;
export const HORA = 60 * MINUTO;
export const DIA = 24 * HORA;
