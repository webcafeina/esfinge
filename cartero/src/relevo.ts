/**
 * Lo que se mira antes de entregar un mensaje, y por qué se mira tanto.
 *
 * Este servicio existe porque Google **no acepta la autenticación SMTP cuando la
 * conexión sale de Cloudflare** (ADR 0045): rechaza con `535 … BadCredentials` las
 * mismas credenciales que acepta desde cualquier otra máquina. La IPv4 del VPS sí
 * está autorizada en el relé, así que el correo lo compone el Worker y lo entrega
 * esta pieza.
 *
 * Y eso convierte a esta pieza en **algo que manda correo como webcafeína a quien se
 * le diga**. El secreto compartido es lo que la cierra, pero un secreto se filtra, y
 * lo que hay detrás no puede ser un relé abierto: por eso el mensaje se revisa aunque
 * venga firmado. Dos reglas, las dos por lo mismo:
 *
 * - **El remitente es el nuestro, exacto.** Sin esto, quien tuviera el secreto
 *   mandaría correo firmado por nuestro dominio —con SPF y DKIM en regla, que es lo
 *   que lo hace peligroso— diciendo ser cualquiera.
 * - **Un solo destinatario.** Sin esto, la misma llave sirve para una tanda.
 *
 * Nada de esto es paranoia de más: el cliente al que sirve Esfinge es un gestor de
 * contraseñas, y un correo creíble desde su propio dominio es exactamente la pieza
 * que le falta a quien quiera pescar a sus usuarios.
 */

/** Lo que hay que saber de un mensaje para entregarlo: a quién va. */
export interface Sobre {
	para: string;
}

export class Rechazado extends Error {}

/**
 * Las cabeceras de un mensaje, hasta la línea en blanco.
 *
 * Se desdoblan las continuaciones —una cabecera puede seguir en la línea siguiente si
 * empieza por espacio, y el asunto largo de `componer` sale así— porque si no, la
 * segunda mitad de un asunto partido parecería una cabecera con un nombre rarísimo.
 */
export function cabecerasDe(mensaje: string): Map<string, string> {
	const corte = mensaje.indexOf("\r\n\r\n");
	const crudas = (corte === -1 ? mensaje : mensaje.slice(0, corte)).split("\r\n");
	const lineas: string[] = [];
	for (const l of crudas) {
		const ultima = lineas.length - 1;
		if (/^[ \t]/.test(l) && ultima >= 0) lineas[ultima] = `${lineas[ultima]} ${l.trim()}`;
		else lineas.push(l);
	}
	const cabeceras = new Map<string, string>();
	for (const l of lineas) {
		const dosPuntos = l.indexOf(":");
		if (dosPuntos === -1) continue;
		const nombre = l.slice(0, dosPuntos).trim().toLowerCase();
		// La primera gana: si alguien colara un `From:` de más, el que cuenta es el
		// primero, que es el que se comprueba.
		if (!cabeceras.has(nombre)) cabeceras.set(nombre, l.slice(dosPuntos + 1).trim());
	}
	return cabeceras;
}

/**
 * La dirección de dentro de `Nombre <a@b>`, o de `a@b` a pelo.
 *
 * Devuelve nulo si hay más de una o si no parece una dirección. **Un `undefined` no
 * vale como respuesta**: quien llama tiene que poder distinguir «no hay» de «hay algo
 * que no entiendo», y las dos cosas terminan igual —rechazando—, pero por escrito.
 */
export function laDireccion(campo: string): string | null {
	if (campo.includes(",") || campo.includes(";")) return null;
	const direccion = (/<([^>]*)>/.exec(campo)?.[1] ?? campo).trim();
	if (!/^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/.test(direccion)) return null;
	return direccion;
}

/**
 * Revisa un mensaje ya compuesto y devuelve a quién va.
 *
 * `remitente` es la cabecera `From` entera y tal cual la escribe el Worker
 * (`Esfinge <esfinge@webcafeina.com>`), no solo la dirección: comparar la cabecera
 * completa es lo que impide que alguien mande como «Banco Santander
 * <esfinge@webcafeina.com>», que pasaría cualquier comprobación que solo mirase la
 * dirección y es justo el correo que nadie debería poder mandar desde aquí.
 */
export function revisar(mensaje: string, remitente: string): Sobre {
	if (mensaje.length > 512 * 1024) throw new Rechazado("El mensaje es demasiado largo.");
	const cabeceras = cabecerasDe(mensaje);

	const de = cabeceras.get("from");
	if (de !== remitente) throw new Rechazado("El remitente no es el de Esfinge.");

	const a = cabeceras.get("to");
	if (a === undefined) throw new Rechazado("El mensaje no dice a quién va.");
	const para = laDireccion(a);
	if (para === null) throw new Rechazado("El destinatario no es una dirección sola.");

	// Copias ocultas incluidas: una `Bcc` no la ve el destinatario, pero sí la
	// entregaría el servidor, así que sería la forma de mandar una tanda con un solo
	// destinatario escrito. Aquí no se entregan por la cabecera, se entrega por el
	// sobre, pero se rechazan igual: un mensaje que las lleva no lo ha compuesto el
	// Worker, y eso ya es razón suficiente.
	for (const otra of ["cc", "bcc"]) {
		if (cabeceras.has(otra)) throw new Rechazado("El mensaje lleva más destinatarios.");
	}
	return { para };
}

/**
 * Compara dos secretos **en tiempo constante**.
 *
 * Con `===`, el tiempo que tarda en decir que no depende de cuántos caracteres del
 * principio ha acertado quien prueba, y eso deja adivinarlo carácter a carácter con
 * suficientes intentos. Es barato de escribir bien y caro de descubrir mal.
 */
export function elSecretoCuadra(dado: string, bueno: string): boolean {
	if (bueno.length === 0) return false;
	// Se compara siempre la misma cantidad de bytes, pase lo que pase con el largo.
	const a = Buffer.from(dado.padEnd(bueno.length, "\0").slice(0, bueno.length));
	const b = Buffer.from(bueno);
	let distintos = dado.length === bueno.length ? 0 : 1;
	for (let i = 0; i < b.length; i++) distintos |= (a[i] ?? 0) ^ (b[i] ?? 0);
	return distintos === 0;
}
