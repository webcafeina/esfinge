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

const PIE = "\n\n—\nEsfinge · Webcafeína\nNunca te pediremos tu contraseña maestra ni tu clave de recuperación.";

/** Dónde se descarga Esfinge y se crea la cuenta. No hay alta desde la web. */
export const DONDE_CREARLA = "https://webcafeina.github.io/esfinge/#descargar";

/**
 * El icono de la cabecera, servido desde la web del proyecto.
 *
 * **Lo eligió el cliente el 2026-09-24, con el coste delante**: pedir esta imagen
 * le cuenta a quien la sirve que el correo se ha abierto —a Google si es Gmail,
 * que hace de intermediario; directamente a GitHub, con IP y hora, en Apple Mail o
 * en Outlook—. Por eso el nombre va **al lado en texto** y el `alt` va vacío: con
 * las imágenes bloqueadas no se pierde nada, que es lo único que esto sí puede
 * garantizar. Está dicho en la política de privacidad.
 */
const ICONO = "https://webcafeina.github.io/esfinge/imagenes/icono.png";

/** Lo que dura una invitación antes de que haya que volver a mandarla. */
export const DIAS_DE_INVITACION = 30;

function escapar(s: string): string {
	return s.replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c] as string);
}

/**
 * El único correo con formato, y solo por el botón (decisión del cliente,
 * 2026-09-24). Tres cosas que un correo obliga y una página no:
 *
 * - **Estilos en línea y nada más**: no hay hoja de estilos, ni variables, ni
 *   clases que sobrevivan. Los colores son los de siempre escritos a mano, y **las
 *   parejas son las que ya mide `internal/tema`** (ADR 0021).
 * - **El botón es una tabla con `bgcolor`, no un enlace con fondo.** Se hizo con un
 *   `<a>` y `background:` abreviado, y **Gmail no lo pintó**: lo vio el cliente en
 *   su buzón el 2026-09-24, aquí no lo decía ninguna prueba. Gmail se come la forma
 *   abreviada, y Outlook de escritorio —que compone con Word— ignora el relleno de
 *   un enlace, así que el color y el tamaño tienen que vivir en una celda. El
 *   `bgcolor` va **además** del `background-color`: es un atributo de HTML y no hay
 *   cliente que lo tire.
 * - **El enlace va también debajo, en texto**, porque un botón que no pinta deja
 *   un correo sin salida; y la versión en texto pelado lleva la misma dirección.
 * - **La marca va en la banda de arriba: el icono y el nombre escrito al lado.** El
 *   icono es una imagen de fuera y eso tiene su precio, dicho en `ICONO`; el nombre
 *   en texto es lo que hace que un cliente con las imágenes bloqueadas siga
 *   enseñando una cabecera y no un hueco.
 */
function cuerpoDeInvitacion(de: string, enlace: string): string {
	const quien = escapar(de);
	const letra = "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif";
	return `<!doctype html><html lang="es"><body style="margin:0;padding:24px;background-color:#f2f2f7;font-family:${letra};color:#3c3c43;">
<div style="max-width:520px;margin:0 auto;background-color:#ffffff;border:1px solid #d8d8de;border-radius:12px;overflow:hidden;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%"><tr>
<td bgcolor="#2b2b31" style="background-color:#2b2b31;padding:14px 28px;"><table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>
<td style="padding-right:10px;line-height:0;"><img src="${ICONO}" alt="" width="28" height="28" style="display:block;width:28px;height:28px;border:0;border-radius:6px;"></td>
<td style="font-family:${letra};font-size:17px;font-weight:700;color:#ffffff;letter-spacing:.01em;">Esfinge</td>
</tr></table></td>
</tr></table>
<div style="padding:28px;">
<p style="margin:0 0 16px;font-size:18px;font-weight:600;color:#1c1c1e;line-height:1.35;">${quien} te quiere mandar una contraseña</p>
<p style="margin:0 0 16px;font-size:15px;line-height:1.5;">${quien}, que usa Esfinge, quiere mandarte una contraseña de forma segura.</p>
<p style="margin:0 0 24px;font-size:15px;line-height:1.5;">Esfinge es un gestor de contraseñas que las cifra en tu propio ordenador: ni Webcafeína ni nadie más puede leerlas. Para recibirla necesitas tu cuenta.</p>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 24px;"><tr>
<td align="center" bgcolor="#f2c14e" style="background-color:#f2c14e;border-radius:8px;"><a href="${enlace}" style="display:inline-block;padding:13px 24px;font-family:${letra};font-size:15px;font-weight:600;color:#2b2b31;text-decoration:none;">Crear mi cuenta de Esfinge</a></td>
</tr></table>
<p style="margin:0 0 24px;font-size:13px;line-height:1.5;color:#3c3c43;">Si el botón no funciona, copia esta dirección en tu navegador:<br><span style="word-break:break-all;">${escapar(enlace)}</span></p>
<p style="margin:0 0 16px;font-size:15px;line-height:1.5;">En cuanto la tengas, la copia te llegará a tu buzón de Esfinge y podrás guardarla o descartarla. La invitación dura ${DIAS_DE_INVITACION} días.</p>
<p style="margin:0;font-size:13px;line-height:1.5;color:#3c3c43;">Si no esperabas esto, ignora este correo: sin cuenta no te llega nada.</p>
<hr style="border:0;border-top:1px solid #d8d8de;margin:24px 0 16px;">
<p style="margin:0;font-size:12px;line-height:1.5;color:#3c3c43;">Esfinge · Webcafeína<br>Nunca te pediremos tu contraseña maestra ni tu clave de recuperación.</p>
</div></div></body></html>`;
}

export const cartas = {
	codigoDeAlta: (para: string, codigo: string): Carta => ({
		para,
		asunto: "Tu código para crear la cuenta de Esfinge",
		texto: `Tu código para crear la cuenta de Esfinge es:\n\n    ${codigo}\n\nCaduca en diez minutos. Si no lo has pedido tú, ignora este correo: sin el código no se crea nada.${PIE}`,
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
		texto: `Tu cuenta de Esfinge está creada con este correo.\n\nTres cosas que conviene no olvidar:\n\n  · Tu bóveda se cifra en tu ordenador antes de salir. No podemos leerla, ni recuperarla si pierdes tus claves.\n  · Guarda tu clave de recuperación fuera del ordenador. Es lo único que abre la bóveda si olvidas la contraseña maestra.\n  · Tu buzón de correo importa tanto como tus claves: por aquí van los códigos.\n\nLas condiciones de uso y la política de privacidad, en https://webcafeina.github.io/esfinge/condiciones.html y https://webcafeina.github.io/esfinge/privacidad.html${PIE}`,
	}),
	yaTienesCuenta: (para: string): Carta => ({
		para,
		asunto: "Ya tienes una cuenta de Esfinge",
		texto: `Alguien ha intentado crear una cuenta de Esfinge con este correo, y ya tienes una.\n\nSi has sido tú, entra con tu contraseña maestra. Si la has olvidado, usa tu clave de recuperación desde «¿La has olvidado?». Si no has sido tú, no tienes que hacer nada.${PIE}`,
	}),
	codigoDeEntrada: (para: string, codigo: string, equipo: string): Carta => ({
		para,
		asunto: "Tu código para entrar en Esfinge",
		texto: `Tu código para entrar en Esfinge desde «${equipo}» es:\n\n    ${codigo}\n\nCaduca en diez minutos. Si no estás entrando tú, alguien conoce tu contraseña maestra: cámbiala cuanto antes.${PIE}`,
	}),
	equipoNuevo: (para: string, equipo: string): Carta => ({
		para,
		asunto: "Un equipo nuevo ha entrado en tu cuenta de Esfinge",
		texto: `«${equipo}» acaba de entrar en tu cuenta de Esfinge.\n\nSi no has sido tú, cambia tu contraseña maestra y olvida ese equipo en Ajustes → Cuenta.${PIE}`,
	}),
	codigoDeRecuperacion: (para: string, codigo: string): Carta => ({
		para,
		asunto: "Tu código para recuperar la cuenta de Esfinge",
		texto: `Tu código para recuperar la cuenta de Esfinge es:\n\n    ${codigo}\n\nCon él y con tu clave de recuperación podrás poner una contraseña maestra nueva. Caduca en diez minutos. Si no lo has pedido tú, ignora este correo: sin la clave de recuperación no sirve de nada.${PIE}`,
	}),
	claveCambiada: (para: string): Carta => ({
		para,
		asunto: "Has cambiado la contraseña de Esfinge",
		texto: `La contraseña maestra de tu cuenta de Esfinge acaba de cambiar, y los demás equipos te la pedirán al volver.\n\nSi no has sido tú, recupera la cuenta con tu clave de recuperación cuanto antes.${PIE}`,
	}),
	codigoDeBorrado: (para: string, codigo: string): Carta => ({
		para,
		asunto: "Tu código para borrar la cuenta de Esfinge",
		texto: `Tu código para borrar la cuenta de Esfinge es:\n\n    ${codigo}\n\nBorrar la cuenta no tiene vuelta atrás: se va la bóveda del servidor y todo lo demás. Lo que tengas en tus equipos se queda en ellos. Si no lo has pedido tú, cambia tu contraseña maestra.${PIE}`,
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
		html: cuerpoDeInvitacion(de, DONDE_CREARLA),
	}),
	cuentaBorrada: (para: string): Carta => ({
		para,
		asunto: "Tu cuenta de Esfinge se ha borrado",
		texto: `Tu cuenta de Esfinge y todo lo que había en el servidor se han borrado. Lo que tengas en tus equipos se queda en ellos.${PIE}`,
	}),
};
