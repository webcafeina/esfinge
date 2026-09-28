// Componer el correo que manda el servidor (ADR 0045).
//
// # Por qué esto existe y está escrito a mano
//
// Hasta la 2.28.x los correos los mandaba Resend con un `POST`, con **cien al día**
// de techo —que era el límite de crecimiento del servicio (ADR 0041)— y un tercero
// en medio de un camino por el que viajan los códigos de las cuentas. Ahora salen
// por el relé de Google Workspace que la casa ya usa en Cronos, y el mensaje lo
// arma este fichero.
//
// # Lo que aquí **no** está, y por qué
//
// El diálogo SMTP. Lo intentó el Worker y **Google no le deja**: rechaza con
// `535 … BadCredentials` las mismas credenciales —comprobado con su huella— que
// acepta desde cualquier otra máquina, contra los dos servidores y con los dos
// mecanismos. No es la configuración del relé ni el secreto: es de dónde sale la
// conexión. Así que entrega `cartero/`, en el VPS, cuya IPv4 sí está autorizada, y
// lo que cruza es este mensaje ya compuesto.
//
// Se quedó lo que se podía seguir probando y se fue lo que no. Si algún día vuelve
// a hacer falta hablar SMTP desde aquí, está en el repositorio, en el commit que lo
// quitó — no hace falta guardarlo en un fichero que nadie puede ejercitar.
//
// # Lo que hay que saber antes de tocarlo
//
//   - **Componer es puro.** Se le pasan el momento y el identificador para poder
//     compararlo byte a byte en una prueba.
//   - **El asunto es el trozo delicado**: `=?UTF-8?B?…?=` con el límite de RFC 2047,
//     partiendo **por caracteres y no por octetos** —cortar un UTF-8 por la mitad
//     deja un rombo negro— y uniendo con salto y espacio.
//   - **Las dos partes van en base64**, y eso resuelve de un golpe los acentos, el
//     límite de 998 octetos por línea —el HTML de la maqueta sale con líneas
//     larguísimas de estilos en línea— y los saltos de línea sueltos del HTML.

import type { Carta } from "./correo";

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

/** El mensaje entero, tal cual se lo lleva el cartero. */
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

	return [cabeceras.join("\r\n"), "", partes.join(""), `--${frontera}--`, ""].join("\r\n");
}
