// Mandar correo por SMTP desde el Worker (ADR 0045).
//
// # Por qué esto existe y está escrito a mano
//
// Hasta la 2.28.x los correos los mandaba Resend con un `POST`. Se cambió al relay
// de Google Workspace —que la casa ya usa en otro proyecto— por dos razones: el
// plan gratuito de Resend son **cien correos al día** y eso era el límite de
// crecimiento del servicio (ADR 0041), y **quita un tercero** de un camino por el
// que viajan los códigos de las cuentas.
//
// Y corrige una premisa: la ADR 0041 dio por hecho que «un Worker no puede» mandar
// correo. Puede. **Solo el puerto 25 está bloqueado**; el 465 y el 587 funcionan
// con `connect()` de `cloudflare:sockets`.
//
// Está escrito aquí en vez de traer una biblioteca porque lo único que este
// servidor necesita del transporte, además de que llegue, es **distinguir «cupo»
// de «fallo»** —la mitad de la ADR 0041—, y para eso hay que leer el código de la
// respuesta. Además así se puede partir en trozos que sí se prueban: `componer`
// es una función pura y el diálogo es una máquina de estados sobre un socket que
// se inyecta.
//
// # Lo que hay que saber antes de tocarlo
//
//   - **Puerto 465 con TLS implícito, no 587 con STARTTLS.** Con STARTTLS existe
//     una fase en claro donde un atacante activo puede quitar el anuncio y quedarse
//     con el `AUTH PLAIN` —usuario y contraseña de Workspace— **y el código de seis
//     cifras**. En un gestor de contraseñas eso no es un detalle. El host y el
//     puerto son variables por si algún día hay que caer al 587, pero **ese camino
//     no está implementado**: código sin probar que solo corre el día malo es peor
//     que no tenerlo.
//   - **Se autentica, no se autoriza por IP.** Cronos usa el mismo relay sin
//     credenciales porque Google tiene apuntada la IP del VPS. Aquí eso **no puede
//     ser**: la IP de salida de un Worker es la del centro de Cloudflare que atienda
//     la petición y cambia en cada invocación. Que nadie lo «simplifique».
//   - **El EHLO se presenta con un dominio real.** Con el nombre de la máquina,
//     Google contesta `421 4.7.0`. Lo aprendió Cronos en producción.
//   - **Plazo para todo el diálogo.** Cinco de los diez correos del servidor son
//     bloqueantes —los que llevan código—, así que un relay que no conteste dejaría
//     la petición colgada.

import type { Carta, Cartero, Entregado, Envio } from "./correo";

/** Lo que este código necesita de un socket. `connect()` lo cumple, y un doble también. */
export interface Conexion {
	readable: ReadableStream<Uint8Array>;
	writable: WritableStream<Uint8Array>;
	close(): Promise<void>;
}

export type Conectar = (host: string, puerto: number) => Conexion | Promise<Conexion>;

export interface Ajustes {
	host: string;
	puerto: number;
	/** Con qué nos presentamos. **Un dominio de verdad**, o Google dice 421. */
	ehlo: string;
	usuario: string;
	clave: string;
	/** Tal cual va en la cabecera: `Esfinge <esfinge@webcafeina.com>`. */
	remitente: string;
	/** Milisegundos para **todo** el diálogo. */
	plazo?: number;
}

const PLAZO_POR_DEFECTO = 10_000;

// ===================================================================== armar

const utf8 = new TextEncoder();

/**
 * base64 de unos bytes. `btoa` trabaja sobre cadenas de un byte por carácter, así
 * que hay que pasar por ahí: darle directamente una cadena con acentos lanza.
 */
export function base64(bytes: Uint8Array): string {
	let s = "";
	for (const b of bytes) s += String.fromCharCode(b);
	return btoa(s);
}

/** En líneas de 76, que es lo que pide MIME para `base64`. */
function enLineas(s: string, ancho = 76): string {
	const trozos: string[] = [];
	for (let i = 0; i < s.length; i += ancho) trozos.push(s.slice(i, i + ancho));
	return trozos.join("\r\n");
}

/**
 * El asunto, codificado como manda RFC 2047 **si hace falta**.
 *
 * Tres cosas que se hacen mal con facilidad y por eso están escritas:
 *
 *   - **Si el asunto es ASCII puro no se codifica.** Así sigue siendo legible en
 *     los registros de cualquiera que mire por el camino, y no se gasta nada.
 *   - **Cada palabra codificada cabe en 75 caracteres**, contando `=?UTF-8?B?` y
 *     `?=`. Quedan 63 para la base64, o sea **45 bytes** de texto — y 45 es
 *     múltiplo de tres, que deja la base64 sin relleno a medias.
 *   - **Se parte por caracteres, no por bytes.** Cortar un UTF-8 multibyte por la
 *     mitad deja un rombo negro en el asunto de un correo que dice «tu código».
 */
export function asunto(s: string): string {
	// eslint-disable-next-line no-control-regex
	if (!/[^\x20-\x7E]/.test(s)) return s;

	const trozos: string[] = [];
	let actual = "";
	let bytes = 0;
	for (const c of s) {
		const n = utf8.encode(c).length;
		if (bytes + n > 45) {
			trozos.push(actual);
			actual = "";
			bytes = 0;
		}
		actual += c;
		bytes += n;
	}
	if (actual) trozos.push(actual);
	// Las palabras codificadas se unen con un salto y un espacio: así el que lo lee
	// las junta sin meter un espacio de más.
	return trozos.map((t) => `=?UTF-8?B?${base64(utf8.encode(t))}?=`).join("\r\n ");
}

/** `Esfinge <esfinge@webcafeina.com>` → `esfinge@webcafeina.com`. */
export function soloLaDireccion(remitente: string): string {
	const m = /<([^>]+)>/.exec(remitente);
	return (m ? m[1] : remitente).trim();
}

/** Como pide RFC 5322: `toUTCString` da «GMT» y ahí va el desplazamiento. */
function fecha(d: Date): string {
	return d.toUTCString().replace(/GMT$/, "+0000");
}

/**
 * Dobla el punto que empieza una línea, que en SMTP marca el final del mensaje.
 *
 * **Hoy no puede pasar**, porque el cuerpo va en base64 y el punto no está en su
 * alfabeto. Se hace igual, y se prueba **aquí y no a través de `componer`**: por ahí
 * la prueba salía verde con el escape quitado —que es como no tener prueba—. El día
 * que alguien vuelva a `quoted-printable` o mande el cuerpo tal cual, la red está
 * puesta y se nota.
 */
export function doblarElPunto(mensaje: string): string {
	return mensaje.replace(/\r\n\./g, "\r\n..");
}

/**
 * La **forma** de las credenciales, nunca su valor.
 *
 * Existe porque el `535` de Google dice «usuario o contraseña no aceptados» y no
 * distingue cuál de los dos, ni si lo que hay guardado en el Worker es lo que se
 * quiso guardar. Y el error más probable de todos es un espacio: Google enseña las
 * contraseñas de aplicación **en grupos de cuatro** y el portapapeles se los lleva.
 *
 * El usuario sale entero —es una dirección de correo, no un secreto, y es la mitad
 * del problema—. De la clave salen **cuántos caracteres tiene y si lleva espacios**,
 * que es lo que hace falta para arreglarlo sin verla nunca. Hay una prueba que mete
 * una clave de verdad y comprueba que no aparece.
 *
 * Lo enseña **solo el Worker de pruebas** (`indice.ts`), como todo lo demás que
 * cuenta por qué ha fallado algo.
 */
export function formaDeLasCredenciales(usuario: string, clave: string): string {
	const espacios = /\s/.test(clave) ? ", y lleva espacios" : "";
	return ` · usuario «${usuario}», clave de ${clave.length} caracteres${espacios}`;
}

/**
 * El mensaje entero, listo para el `DATA`. **Es una función pura**: se le pasan el
 * momento y el identificador para poder compararla byte a byte en una prueba.
 *
 * Las dos partes van en **base64**, y eso resuelve de un golpe cuatro cosas: los
 * acentos, el límite de 998 octetos por línea —el HTML de la maqueta sale con
 * líneas larguísimas de estilos en línea, y un servidor que las corte rompe el
 * correo—, que ninguna línea pueda empezar por un punto, y que no se cuele un
 * salto de línea suelto de los que trae el HTML.
 */
export function componer(c: Carta, remitente: string, cuando = new Date(), id = crypto.randomUUID()): string {
	const dominio = soloLaDireccion(remitente).split("@")[1] ?? "localhost";
	const frontera = `=_esfinge_${id}`;

	const cabeceras = [
		`From: ${remitente}`,
		`To: ${c.para}`,
		`Subject: ${asunto(c.asunto)}`,
		`Date: ${fecha(cuando)}`,
		`Message-ID: <${id}@${dominio}>`,
		"MIME-Version: 1.0",
		// Que ningún «estoy de vacaciones» conteste a un correo con un código dentro.
		"Auto-Submitted: auto-generated",
		`Content-Type: multipart/alternative; boundary="${frontera}"`,
	];

	const parte = (tipo: string, cuerpo: string) =>
		[
			`--${frontera}`,
			`Content-Type: ${tipo}; charset=UTF-8`,
			"Content-Transfer-Encoding: base64",
			"",
			enLineas(base64(utf8.encode(cuerpo))),
			"",
		].join("\r\n");

	const partes = [parte("text/plain", c.texto)];
	if (c.html) partes.push(parte("text/html", c.html));

	const mensaje = [cabeceras.join("\r\n"), "", partes.join(""), `--${frontera}--`, ""].join("\r\n");

	return doblarElPunto(mensaje);
}

// ================================================================== el habla

export interface Respuesta {
	codigo: number;
	texto: string;
}

/**
 * Lee respuestas de SMTP, que pueden venir en varias líneas.
 *
 * Una respuesta termina en la línea cuyo código lleva **un espacio** detrás; las
 * de antes llevan un guion. Y no se puede dar por hecho que cada lectura del
 * socket traiga una respuesta entera ni una sola: hay que acumular.
 */
export class Lector {
	private pendiente = "";
	private lector: ReadableStreamDefaultReader<Uint8Array>;
	private decodificador = new TextDecoder();

	constructor(r: ReadableStream<Uint8Array>) {
		this.lector = r.getReader();
	}

	async respuesta(): Promise<Respuesta> {
		for (;;) {
			const hecha = this.entera();
			if (hecha) return hecha;
			const { value, done } = await this.lector.read();
			if (done) throw new Error("El servidor de correo ha cerrado la conexión");
			this.pendiente += this.decodificador.decode(value, { stream: true });
		}
	}

	private entera(): Respuesta | null {
		const lineas = this.pendiente.split("\r\n");
		// La última puede estar a medias, así que no se mira.
		for (let i = 0; i < lineas.length - 1; i++) {
			if (!/^\d{3} /.test(lineas[i])) continue;
			const usadas = lineas.slice(0, i + 1);
			this.pendiente = lineas.slice(i + 1).join("\r\n");
			return { codigo: Number(lineas[i].slice(0, 3)), texto: usadas.join("\n") };
		}
		return null;
	}
}

/**
 * Qué significa una respuesta que no es un «sí».
 *
 * **«Cupo» es mañana y «fallo» es dentro de un rato**, y la diferencia se la
 * enseñamos a quien lo lee: decir «prueba otra vez en un momento» cuando la verdad
 * es «mañana» fue justo lo que la ADR 0041 vino a arreglar.
 *
 * Google documenta `5.4.5` como «Daily SMTP relay limit exceeded». El `421 4.7.0`
 * **no** es cupo: es estrangulamiento de minutos, y ahí «prueba otra vez en un
 * momento» sí es verdad.
 */
export function queHaPasado(r: Respuesta): Entregado {
	if (/\b[45]\.4\.5\b/.test(r.texto)) return "cupo";
	if (/daily .*limit|sending quota exceeded|relay limit/i.test(r.texto)) return "cupo";
	return "fallo";
}

class Fallo extends Error {
	constructor(
		public entregado: Entregado,
		public porque: string,
	) {
		super(porque);
	}
}

export class CarteroSmtp implements Cartero {
	constructor(
		private a: Ajustes,
		private conectar: Conectar,
	) {}

	async mandar(c: Carta): Promise<Envio> {
		let cx: Conexion | null = null;
		let reloj: ReturnType<typeof setTimeout> | undefined;
		try {
			cx = await this.conectar(this.a.host, this.a.puerto);
			const conexion = cx;
			const plazo = new Promise<never>((_, romper) => {
				reloj = setTimeout(() => romper(new Fallo("fallo", "se acabó el plazo")), this.a.plazo ?? PLAZO_POR_DEFECTO);
			});
			return await Promise.race([this.conversar(conexion, c), plazo]);
		} catch (e) {
			if (e instanceof Fallo) return { entregado: e.entregado, porque: e.porque };
			return { entregado: "fallo", porque: `${e}` };
		} finally {
			if (reloj !== undefined) clearTimeout(reloj);
			// **Se cierra siempre**, también por el camino del error: un socket que se
			// queda abierto en un aislado que se muere es una conexión que Google ve
			// colgada.
			await cx?.close().catch(() => {});
		}
	}

	private async conversar(cx: Conexion, c: Carta): Promise<Envio> {
		const lector = new Lector(cx.readable);
		const escritor = cx.writable.getWriter();
		const decir = async (linea: string) => {
			await escritor.write(utf8.encode(linea + "\r\n"));
		};
		// **Cada espera dice en qué paso estaba.** Sin eso, lo único que queda de un
		// envío que falla es «fallo», que es lo mismo que nada.
		const esperar = async (paso: string, bueno: (n: number) => boolean) => {
			const r = await lector.respuesta();
			if (!bueno(r.codigo)) throw new Fallo(queHaPasado(r), `${paso}: ${r.texto.split("\n")[0]}`);
			return r;
		};

		try {
			await esperar("saludo", (n) => n === 220);

			await decir(`EHLO ${this.a.ehlo}`);
			const saludo = await esperar("EHLO", (n) => n === 250);

			await this.autenticar(saludo.texto, decir, esperar);

			await decir(`MAIL FROM:<${soloLaDireccion(this.a.remitente)}>`);
			await esperar("MAIL FROM", (n) => n === 250);

			await decir(`RCPT TO:<${c.para}>`);
			await esperar("RCPT TO", (n) => n === 250 || n === 251);

			await decir("DATA");
			await esperar("DATA", (n) => n === 354);

			await escritor.write(utf8.encode(componer(c, this.a.remitente)));
			await decir(".");
			await esperar("cuerpo", (n) => n === 250);

			// El adiós es cortesía, no condición: si falla, el correo ya está dentro.
			await decir("QUIT").catch(() => {});
			return { entregado: "ok" };
		} finally {
			await escritor.close().catch(() => {});
		}
	}

	/**
	 * `PLAIN` si el servidor lo anuncia, y si no `LOGIN`.
	 *
	 * Se mira lo que anuncia en vez de dar uno por hecho, que es lo que separa un
	 * cliente de un guion: si Google cambia lo que ofrece, esto sigue hablando.
	 */
	private async autenticar(
		saludo: string,
		decir: (l: string) => Promise<void>,
		esperar: (paso: string, b: (n: number) => boolean) => Promise<Respuesta>,
	) {
		const anunciado = /^250[ -]AUTH (.*)$/im.exec(saludo)?.[1]?.toUpperCase() ?? "";
		const { usuario, clave } = this.a;

		if (!anunciado.includes("LOGIN") || anunciado.includes("PLAIN")) {
			await decir(`AUTH PLAIN ${base64(utf8.encode(`\0${usuario}\0${clave}`))}`);
			await esperar("AUTH PLAIN", (n) => n === 235);
			return;
		}
		await decir("AUTH LOGIN");
		await esperar("AUTH LOGIN", (n) => n === 334);
		await decir(base64(utf8.encode(usuario)));
		await esperar("AUTH usuario", (n) => n === 334);
		await decir(base64(utf8.encode(clave)));
		await esperar("AUTH clave", (n) => n === 235);
	}
}
