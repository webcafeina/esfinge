// Los correos que manda el servidor: códigos y avisos.
//
// **El código va en el cuerpo, nunca en el asunto**: el asunto se ve en la pantalla
// bloqueada del móvil de cualquiera que esté al lado.
//
// En producción los manda Resend; en el Worker de pruebas se quedan en una tabla
// que lee la ruta `/_pruebas/buzon`, y así las pruebas de punta a punta pueden
// entrar sin un buzón de verdad. **Esa ruta no existe fuera de pruebas**, y hay una
// prueba que lo vigila.

import type { Env } from "./protocolo";

export interface Carta {
	para: string;
	asunto: string;
	texto: string;
	/**
	 * La versión con formato, **solo donde hace falta pulsar algo**.
	 *
	 * Todos los demás correos van en texto pelado a propósito: llevan un código que
	 * se copia, y un correo con formato solo añade sitio donde esconder cosas. La
	 * invitación es la excepción, porque lo que se pide ahí es **ir a un sitio**, y
	 * un enlace desnudo en medio de un párrafo se lee como el correo que nadie
	 * pulsa. Cuando va, **el texto sigue yendo y dice lo mismo**: hay quien lee el
	 * correo en texto, y la dirección tiene que estar también ahí.
	 */
	html?: string;
	/** Para que un reintento no mande el mismo correo dos veces. */
	idempotencia?: string;
}

/**
 * Qué ha pasado al mandar. **«Cupo» no es «fallo»**, y por eso son tres y no dos
 * (ADR 0041): con el plan gratuito de Resend —cien correos al día— agotarlo es lo
 * más probable que pase, y decir «prueba otra vez en un momento» sería mentir: no
 * es en un momento, es mañana.
 */
export type Entregado = "ok" | "fallo" | "cupo";

export interface Cartero {
	mandar(c: Carta): Promise<Entregado>;
}

export function carteroPara(env: Env): Cartero {
	if (env.ENTORNO === "pruebas") return new CarteroDePruebas(env.BD);
	return new CarteroResend(env.RESEND_API_KEY ?? "", env.REMITENTE);
}

class CarteroResend implements Cartero {
	constructor(
		private clave: string,
		private remitente: string,
	) {}

	async mandar(c: Carta): Promise<Entregado> {
		if (!this.clave) return "fallo";
		const cabeceras: Record<string, string> = {
			Authorization: `Bearer ${this.clave}`,
			"Content-Type": "application/json",
		};
		if (c.idempotencia) cabeceras["Idempotency-Key"] = c.idempotencia;
		try {
			const r = await fetch("https://api.resend.com/emails", {
				method: "POST",
				headers: cabeceras,
				body: JSON.stringify({
					from: this.remitente,
					to: [c.para],
					subject: c.asunto,
					text: c.texto,
					...(c.html ? { html: c.html } : {}),
				}),
			});
			if (r.ok) return "ok";
			// 429 es «demasiados»; Resend lo usa tanto para el cupo del día como para su
			// límite por segundo. Desde fuera no se distinguen, y para quien lo lee el
			// consejo es el mismo: esto no se arregla volviendo a pulsar.
			return r.status === 429 ? "cupo" : "fallo";
		} catch {
			return "fallo";
		}
	}
}

class CarteroDePruebas implements Cartero {
	constructor(private bd: D1Database) {}

	async mandar(c: Carta): Promise<Entregado> {
		await this.bd
			.prepare("INSERT INTO buzon_pruebas (correo, asunto, cuerpo, momento) VALUES (?, ?, ?, ?)")
			.bind(c.para, c.asunto, c.texto, Date.now())
			.run();
		return "ok";
	}
}


// ================================================================ la maqueta
//
// **Todos los correos van con formato desde el 2026-09-24**, decidido con el
// cliente: hasta entonces solo lo llevaba la invitación y el resto era texto pelado.
// Lo que un correo obliga y una página no, y que aquí está resuelto de una vez para
// todos:
//
//   - **Estilos en línea y nada más.** No hay hoja de estilos, ni variables, ni
//     clases que sobrevivan al cliente de correo. Los colores van escritos, y **las
//     parejas son las que mide `internal/tema/extension_test.go`** (ADR 0021).
//   - **Los botones son celdas con `bgcolor`.** Un `<a>` con fondo no lo pinta
//     Gmail —lo vio el cliente en su buzón— y Outlook de escritorio, que compone
//     con Word, ignora su relleno. El atributo de HTML no lo tira nadie.
//   - **Nada de forma abreviada en CSS**: `background-color:`, nunca `background:`.
//   - **Todo lo que se interpola va escapado.** El nombre de un equipo es texto
//     libre que manda el cliente, y en un correo con formato eso es una inyección.
//   - **El texto pelado sigue yendo, y dice lo mismo.** Hay quien lee el correo en
//     texto, y además es lo que leen las pruebas.
//   - **El código se copia, así que es texto**, grande y separado, nunca una imagen.
//     Y **nunca hay un enlace que entre por ti**: esto es un gestor de contraseñas y
//     un correo que dice «pulsa aquí para entrar» es una clase de phishing gratis.

const LETRA = "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif";
const MONO = "ui-monospace,SFMono-Regular,Menlo,Consolas,'Liberation Mono',monospace";

function escapar(s: string): string {
	return s.replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c] as string);
}

/** Un párrafo. El texto ya viene escapado por quien lo arma. */
const p = (html: string) => `<p style="margin:0 0 16px;font-size:15px;line-height:1.5;color:#3c3c43;">${html}</p>`;

/** Lo pequeño del final de un bloque: plazos, «si no has sido tú», la letra chica. */
const nota = (html: string) => `<p style="margin:0 0 16px;font-size:13px;line-height:1.5;color:#3c3c43;">${html}</p>`;

/**
 * El código, para copiarlo con los ojos o con el ratón.
 *
 * **Es texto**: se selecciona, se copia y se lee con un lector de pantalla. En una
 * imagen no se podría hacer ninguna de las tres cosas.
 */
const codigo = (c: string) =>
	`<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="margin:0 0 20px;"><tr>` +
	`<td align="center" bgcolor="#f2f2f7" style="background-color:#f2f2f7;border-radius:10px;padding:18px 12px;` +
	`font-family:${MONO};font-size:30px;font-weight:600;letter-spacing:.18em;color:#1c1c1e;">${escapar(c)}</td>` +
	`</tr></table>`;

/** Lo que hay que mirar dos veces: un recuadro con su filete a la izquierda. */
const aviso = (html: string) =>
	`<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="margin:0 0 16px;"><tr>` +
	`<td bgcolor="#f2f2f7" style="background-color:#f2f2f7;border-left:3px solid #2b2b31;border-radius:0 8px 8px 0;` +
	`padding:14px 16px;font-family:${LETRA};font-size:14px;line-height:1.5;color:#3c3c43;">${html}</td>` +
	`</tr></table>`;

/**
 * El botón. **El color vive en la celda**, no en el enlace: ver arriba.
 *
 * Solo lo llevan los correos donde de verdad hay algo que abrir; los del código no,
 * porque lo que hay que hacer con un código es escribirlo en Esfinge.
 */
const boton = (texto: string, url: string) =>
	`<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 20px;"><tr>` +
	`<td align="center" bgcolor="#f2c14e" style="background-color:#f2c14e;border-radius:8px;">` +
	`<a href="${url}" style="display:inline-block;padding:13px 24px;font-family:${LETRA};font-size:15px;` +
	`font-weight:600;color:#2b2b31;text-decoration:none;">${escapar(texto)}</a></td></tr></table>`;

/** Un enlace dentro de un párrafo, con el color de la tinta: el oro como línea no se ve. */
const enlace = (texto: string, url: string) =>
	`<a href="${url}" style="color:#1c1c1e;text-decoration:underline;">${escapar(texto)}</a>`;

/**
 * La hoja entera: la banda de marca, el titular, lo que sea, y el pie.
 *
 * El icono de la cabecera **lo eligió el cliente con el coste delante** el
 * 2026-09-24: para enseñarlo, el programa de correo se lo pide a GitHub, y quien
 * sirve esa imagen se entera de que el correo se ha abierto —con hora e IP salvo en
 * Gmail, que la pide por ti—. Está dicho en la política de privacidad. Por eso el
 * nombre va **escrito al lado** y el `alt` va vacío: con las imágenes bloqueadas, que
 * es como llegan de entrada a casi todo el mundo, se sigue leyendo una cabecera y no
 * queda un hueco.
 */
function hoja(titulo: string, ...bloques: string[]): string {
	return (
		`<!doctype html><html lang="es"><body style="margin:0;padding:24px;background-color:#f2f2f7;font-family:${LETRA};color:#3c3c43;">` +
		`<div style="max-width:520px;margin:0 auto;background-color:#ffffff;border:1px solid #d8d8de;border-radius:12px;overflow:hidden;">` +
		`<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%"><tr>` +
		`<td bgcolor="#2b2b31" style="background-color:#2b2b31;padding:14px 28px;">` +
		`<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>` +
		`<td style="padding-right:10px;line-height:0;"><img src="${ICONO}" alt="" width="28" height="28" style="display:block;width:28px;height:28px;border:0;border-radius:6px;"></td>` +
		`<td style="font-family:${LETRA};font-size:17px;font-weight:700;color:#ffffff;letter-spacing:.01em;">Esfinge</td>` +
		`</tr></table></td></tr></table>` +
		`<div style="padding:28px;">` +
		`<p style="margin:0 0 16px;font-size:18px;font-weight:600;color:#1c1c1e;line-height:1.35;">${titulo}</p>` +
		bloques.join("") +
		`<hr style="border:0;border-top:1px solid #d8d8de;margin:24px 0 16px;">` +
		`<p style="margin:0;font-size:12px;line-height:1.5;color:#3c3c43;">Esfinge · Webcafeína<br>` +
		`Nunca te pediremos tu contraseña maestra ni tu clave de recuperación.</p>` +
		`</div></div></body></html>`
	);
}

const PIE = "\n\n—\nEsfinge · Webcafeína\nNunca te pediremos tu contraseña maestra ni tu clave de recuperación.";

/** Dónde se descarga Esfinge y se crea la cuenta. No hay alta desde la web. */
export const DONDE_CREARLA = "https://webcafeina.github.io/esfinge/#descargar";
const CONDICIONES = "https://webcafeina.github.io/esfinge/condiciones.html";
const PRIVACIDAD = "https://webcafeina.github.io/esfinge/privacidad.html";

/** El icono de la cabecera, servido desde la web del proyecto. Ver `hoja`. */
const ICONO = "https://webcafeina.github.io/esfinge/imagenes/icono.png";

/** Lo que dura una invitación antes de que haya que volver a mandarla. */
export const DIAS_DE_INVITACION = 30;

export const cartas = {
	codigoDeAlta: (para: string, c: string): Carta => ({
		para,
		asunto: "Tu código para crear la cuenta de Esfinge",
		texto: `Tu código para crear la cuenta de Esfinge es:\n\n    ${c}\n\nCaduca en diez minutos. Si no lo has pedido tú, ignora este correo: sin el código no se crea nada.${PIE}`,
		html: hoja(
			"Tu código para crear la cuenta",
			p("Escríbelo en Esfinge para terminar de crear tu cuenta:"),
			codigo(c),
			nota("Caduca en diez minutos. Si no lo has pedido tú, ignora este correo: sin el código no se crea nada."),
		),
	}),
	/**
	 * La confirmación del alta, que **pide el artículo 28 de la LSSI**: quien
	 * contrata a distancia tiene que recibir constancia de lo que ha contratado y
	 * dónde están las condiciones. Va después de crear la cuenta, no antes.
	 *
	 * Y cuesta un correo más de los cien al día del plan gratuito de Resend
	 * (ADR 0041): dos por alta en vez de uno.
	 */
	cuentaCreada: (para: string): Carta => ({
		para,
		asunto: "Tu cuenta de Esfinge está creada",
		texto: `Tu cuenta de Esfinge está creada con este correo.\n\nTres cosas que conviene no olvidar:\n\n  · Tu bóveda se cifra en tu ordenador antes de salir. No podemos leerla, ni recuperarla si pierdes tus claves.\n  · Guarda tu clave de recuperación fuera del ordenador. Es lo único que abre la bóveda si olvidas la contraseña maestra.\n  · Tu buzón de correo importa tanto como tus claves: por aquí van los códigos.\n\nLas condiciones de uso y la política de privacidad, en ${CONDICIONES} y ${PRIVACIDAD}${PIE}`,
		html: hoja(
			"Tu cuenta de Esfinge está creada",
			p("Ya tienes cuenta con este correo. Tres cosas que conviene no olvidar:"),
			aviso(
				"<strong>Tu bóveda se cifra en tu ordenador antes de salir.</strong> No podemos leerla, ni recuperarla si pierdes tus claves.<br><br>" +
					"<strong>Guarda tu clave de recuperación fuera del ordenador.</strong> Es lo único que abre la bóveda si olvidas la contraseña maestra.<br><br>" +
					"<strong>Tu buzón de correo importa tanto como tus claves</strong>: por aquí van los códigos.",
			),
			nota(`Las ${enlace("condiciones de uso", CONDICIONES)} y la ${enlace("política de privacidad", PRIVACIDAD)}.`),
		),
	}),
	yaTienesCuenta: (para: string): Carta => ({
		para,
		asunto: "Ya tienes una cuenta de Esfinge",
		texto: `Alguien ha intentado crear una cuenta de Esfinge con este correo, y ya tienes una.\n\nSi has sido tú, entra con tu contraseña maestra. Si la has olvidado, usa tu clave de recuperación desde «¿La has olvidado?». Si no has sido tú, no tienes que hacer nada.${PIE}`,
		html: hoja(
			"Ya tienes una cuenta de Esfinge",
			p("Alguien ha intentado crear una cuenta con este correo, y ya tienes una."),
			p("Si has sido tú, entra con tu contraseña maestra. Si la has olvidado, usa tu clave de recuperación desde «¿La has olvidado?»."),
			nota("Si no has sido tú, no tienes que hacer nada: con este correo no se puede crear otra cuenta."),
		),
	}),
	codigoDeEntrada: (para: string, c: string, equipo: string): Carta => ({
		para,
		asunto: "Tu código para entrar en Esfinge",
		texto: `Tu código para entrar en Esfinge desde «${equipo}» es:\n\n    ${c}\n\nCaduca en diez minutos. Si no estás entrando tú, alguien conoce tu contraseña maestra: cámbiala cuanto antes.${PIE}`,
		html: hoja(
			"Tu código para entrar",
			p(`Alguien está entrando en tu cuenta desde <strong>${escapar(equipo)}</strong>. Si eres tú, escribe este código:`),
			codigo(c),
			nota("Caduca en diez minutos."),
			aviso("<strong>Si no estás entrando tú, alguien conoce tu contraseña maestra.</strong> Cámbiala cuanto antes."),
		),
	}),
	equipoNuevo: (para: string, equipo: string): Carta => ({
		para,
		asunto: "Un equipo nuevo ha entrado en tu cuenta de Esfinge",
		texto: `«${equipo}» acaba de entrar en tu cuenta de Esfinge.\n\nSi no has sido tú, cambia tu contraseña maestra y olvida ese equipo en Ajustes → Cuenta.${PIE}`,
		html: hoja(
			"Un equipo nuevo ha entrado en tu cuenta",
			p(`<strong>${escapar(equipo)}</strong> acaba de entrar en tu cuenta de Esfinge.`),
			aviso("<strong>Si no has sido tú</strong>, cambia tu contraseña maestra y olvida ese equipo en Ajustes → Cuenta."),
		),
	}),
	codigoDeRecuperacion: (para: string, c: string): Carta => ({
		para,
		asunto: "Tu código para recuperar la cuenta de Esfinge",
		texto: `Tu código para recuperar la cuenta de Esfinge es:\n\n    ${c}\n\nCon él y con tu clave de recuperación podrás poner una contraseña maestra nueva. Caduca en diez minutos. Si no lo has pedido tú, ignora este correo: sin la clave de recuperación no sirve de nada.${PIE}`,
		html: hoja(
			"Tu código para recuperar la cuenta",
			p("Con este código y con tu clave de recuperación podrás poner una contraseña maestra nueva:"),
			codigo(c),
			nota("Caduca en diez minutos."),
			aviso("Si no lo has pedido tú, <strong>ignora este correo</strong>: sin tu clave de recuperación no sirve de nada."),
		),
	}),
	claveCambiada: (para: string): Carta => ({
		para,
		asunto: "Has cambiado la contraseña de Esfinge",
		texto: `La contraseña maestra de tu cuenta de Esfinge acaba de cambiar, y los demás equipos te la pedirán al volver.\n\nSi no has sido tú, recupera la cuenta con tu clave de recuperación cuanto antes.${PIE}`,
		html: hoja(
			"Has cambiado la contraseña de Esfinge",
			p("La contraseña maestra de tu cuenta acaba de cambiar. Los demás equipos te la pedirán al volver."),
			aviso("<strong>Si no has sido tú</strong>, recupera la cuenta con tu clave de recuperación cuanto antes."),
		),
	}),
	codigoDeBorrado: (para: string, c: string): Carta => ({
		para,
		asunto: "Tu código para borrar la cuenta de Esfinge",
		texto: `Tu código para borrar la cuenta de Esfinge es:\n\n    ${c}\n\nBorrar la cuenta no tiene vuelta atrás: se va la bóveda del servidor y todo lo demás. Lo que tengas en tus equipos se queda en ellos. Si no lo has pedido tú, cambia tu contraseña maestra.${PIE}`,
		html: hoja(
			"Tu código para borrar la cuenta",
			p("Escríbelo en Esfinge para borrar tu cuenta:"),
			codigo(c),
			aviso(
				"<strong>Borrar la cuenta no tiene vuelta atrás</strong>: se va la bóveda del servidor y todo lo demás. " +
					"Lo que tengas en tus equipos se queda en ellos.<br><br>Si no lo has pedido tú, <strong>cambia tu contraseña maestra</strong>.",
			),
		),
	}),
	cuentaBorrada: (para: string): Carta => ({
		para,
		asunto: "Tu cuenta de Esfinge se ha borrado",
		texto: `Tu cuenta de Esfinge y todo lo que había en el servidor se han borrado. Lo que tengas en tus equipos se queda en ellos.${PIE}`,
		html: hoja(
			"Tu cuenta de Esfinge se ha borrado",
			p("Tu cuenta y todo lo que había en el servidor se han borrado."),
			p("Lo que tengas en tus equipos se queda en ellos: Esfinge sigue funcionando ahí, sin cuenta."),
		),
	}),
	/**
	 * La invitación a quien todavía no tiene cuenta (ADR 0043, entrega B3).
	 *
	 * **Lleva el correo de quien invita**, que lo eligió el cliente: sin él es un
	 * correo anónimo pidiendo que te des de alta en algo, y eso no lo abre nadie. A
	 * cambio, el servidor le cuenta a un desconocido que esa persona usa Esfinge, y
	 * deja mandar un correo con el nombre de alguien dentro. Lo segundo lo frena el
	 * tope de invitaciones por cuenta y día; lo primero está dicho en la ADR.
	 *
	 * **Lo que no lleva es la contraseña**, ni nada con que sacarla: el sobre no ha
	 * salido todavía y espera en la bóveda de quien lo manda.
	 */
	invitacion: (para: string, de: string): Carta => ({
		para,
		asunto: `${de} te quiere mandar una contraseña`,
		texto: `${de}, que usa Esfinge, quiere mandarte una contraseña de forma segura.\n\nEsfinge es un gestor de contraseñas que las cifra en tu propio ordenador: ni Webcafeína ni nadie más puede leerlas. Para recibirla necesitas tu cuenta.\n\nCrea la tuya aquí:\n\n    ${DONDE_CREARLA}\n\nEn cuanto la tengas, la copia te llegará a tu buzón de Esfinge y podrás guardarla o descartarla. La invitación dura ${DIAS_DE_INVITACION} días.\n\nSi no esperabas esto, ignora este correo: sin cuenta no te llega nada.${PIE}`,
		html: hoja(
			`${escapar(de)} te quiere mandar una contraseña`,
			p(`<strong>${escapar(de)}</strong>, que usa Esfinge, quiere mandarte una contraseña de forma segura.`),
			p("Esfinge es un gestor de contraseñas que las cifra en tu propio ordenador: ni Webcafeína ni nadie más puede leerlas. Para recibirla necesitas tu cuenta."),
			boton("Crear mi cuenta de Esfinge", DONDE_CREARLA),
			nota(`Si el botón no funciona, copia esta dirección en tu navegador:<br><span style="word-break:break-all;">${escapar(DONDE_CREARLA)}</span>`),
			p(`En cuanto la tengas, la copia te llegará a tu buzón de Esfinge y podrás guardarla o descartarla. La invitación dura ${DIAS_DE_INVITACION} días.`),
			nota("Si no esperabas esto, ignora este correo: sin cuenta no te llega nada."),
		),
	}),
};
