// Lo que repiten las pruebas: pedir al Worker, leer el buzón de pruebas y dar de
// alta una cuenta. **Hacen lo que hará Esfinge**, con claves al azar en vez de
// derivadas: el servidor no sabe ni debe saber cómo se derivan.

import { exports } from "cloudflare:workers";

let ultimaIP = 0;

/** Cada prueba con su IP, para que el freno por IP de una no le caiga a otra. */
export function nuevaIP(): string {
	ultimaIP++;
	return `10.0.${Math.floor(ultimaIP / 250)}.${ultimaIP % 250}`;
}

export interface Opciones {
	cuerpo?: unknown;
	crudo?: BodyInit;
	token?: string;
	cabeceras?: Record<string, string>;
	ip?: string;
}

export function peticion(metodo: string, ruta: string, o: Opciones = {}): Request {
	const cabeceras: Record<string, string> = { "CF-Connecting-IP": o.ip ?? "10.255.255.1", ...o.cabeceras };
	if (o.token) cabeceras.Authorization = `Bearer ${o.token}`;
	let cuerpo: BodyInit | undefined = o.crudo;
	if (o.cuerpo !== undefined) {
		cuerpo = JSON.stringify(o.cuerpo);
		cabeceras["Content-Type"] = "application/json";
	}
	return new Request(`https://esfinge-cuentas.test${ruta}`, { method: metodo, headers: cabeceras, body: cuerpo });
}

export function pedir(metodo: string, ruta: string, o: Opciones = {}): Promise<Response> {
	return exports.default.fetch(peticion(metodo, ruta, o));
}

export function azarB64(n: number): string {
	const b = new Uint8Array(n);
	crypto.getRandomValues(b);
	let s = "";
	for (const x of b) s += String.fromCharCode(x);
	return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export async function buzon(correo: string): Promise<{ asunto: string; cuerpo: string }[]> {
	const r = await pedir("GET", `/_pruebas/buzon?correo=${encodeURIComponent(correo)}`);
	return ((await r.json()) as { mensajes: { asunto: string; cuerpo: string }[] }).mensajes;
}

/**
 * El código de seis cifras del **último correo que lleve uno**.
 *
 * No vale mirar solo el último que ha llegado: desde que el alta manda también su
 * confirmación (LSSI 28), el más reciente puede no llevar código.
 */
export async function ultimoCodigo(correo: string): Promise<string> {
	for (const carta of await buzon(correo)) {
		const m = /^\s+(\d{6})$/m.exec(carta.cuerpo);
		if (m) return m[1];
	}
	throw new Error(`No hay código en el buzón de ${correo}`);
}

export const ARGON2 = { memoria: 65536, pasadas: 3, paralelismo: 4 };

export interface Alta {
	correo: string;
	cuenta: string;
	sesion: string;
	dispositivo: string;
	confianza?: string;
	claveDeAcceso: string;
	posesion: string;
	sal: string;
	ip: string;
}

export async function darDeAlta(correo: string, confiar = false): Promise<Alta> {
	const ip = nuevaIP();
	const inicio = await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo }, ip });
	if (inicio.status !== 202) throw new Error(`Alta: ${inicio.status} ${await inicio.text()}`);
	const codigo = await ultimoCodigo(correo);
	const claveDeAcceso = azarB64(32);
	const posesion = azarB64(32);
	const sal = azarB64(16);
	const fin = await pedir("POST", "/v1/registro/fin", {
		ip,
		cuerpo: { correo, codigo, sal, argon2: ARGON2, claveDeAcceso, posesion, dispositivo: "Portátil", confiar },
	});
	if (fin.status !== 201) throw new Error(`Alta: ${fin.status} ${await fin.text()}`);
	const d = (await fin.json()) as { cuenta: string; sesion: string; dispositivo: string; confianza?: string };
	return { correo, ...d, claveDeAcceso, posesion, sal, ip };
}

/** Una bóveda de mentira con la forma de la de verdad: el servidor solo mira la forma. */
export function boveda(id: string, relleno = "", recuperacion = "ESF1.recuperacion-de-mentira"): string {
	return JSON.stringify({
		esfinge: "bóveda",
		aviso: "No la edites a mano.",
		formato: 1,
		id,
		serie: 1,
		cambiada: "2026-09-18T10:00:00Z",
		sobres: [
			{ tipo: "maestra", creado: "2026-09-18T10:00:00Z", contenedor: "ESF1.maestra-de-mentira" },
			{ tipo: "recuperacion", creado: "2026-09-18T10:00:00Z", contenedor: recuperacion, codificacion: "palabras" },
		],
		sello: "ESF1.sello",
		cuerpo: `ESF1.cuerpo${relleno}`,
	});
}
