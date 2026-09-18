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
	/** Para que un reintento no mande el mismo correo dos veces. */
	idempotencia?: string;
}

export interface Cartero {
	mandar(c: Carta): Promise<boolean>;
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

	async mandar(c: Carta): Promise<boolean> {
		if (!this.clave) return false;
		const cabeceras: Record<string, string> = {
			Authorization: `Bearer ${this.clave}`,
			"Content-Type": "application/json",
		};
		if (c.idempotencia) cabeceras["Idempotency-Key"] = c.idempotencia;
		try {
			const r = await fetch("https://api.resend.com/emails", {
				method: "POST",
				headers: cabeceras,
				body: JSON.stringify({ from: this.remitente, to: [c.para], subject: c.asunto, text: c.texto }),
			});
			return r.ok;
		} catch {
			return false;
		}
	}
}

class CarteroDePruebas implements Cartero {
	constructor(private bd: D1Database) {}

	async mandar(c: Carta): Promise<boolean> {
		await this.bd
			.prepare("INSERT INTO buzon_pruebas (correo, asunto, cuerpo, momento) VALUES (?, ?, ?, ?)")
			.bind(c.para, c.asunto, c.texto, Date.now())
			.run();
		return true;
	}
}

const PIE = "\n\n—\nEsfinge · Webcafeína\nNunca te pediremos tu contraseña maestra ni tu clave de recuperación.";

export const cartas = {
	codigoDeAlta: (para: string, codigo: string): Carta => ({
		para,
		asunto: "Tu código para crear la cuenta de Esfinge",
		texto: `Tu código para crear la cuenta de Esfinge es:\n\n    ${codigo}\n\nCaduca en diez minutos. Si no lo has pedido tú, ignora este correo: sin el código no se crea nada.${PIE}`,
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
	cuentaBorrada: (para: string): Carta => ({
		para,
		asunto: "Tu cuenta de Esfinge se ha borrado",
		texto: `Tu cuenta de Esfinge y todo lo que había en el servidor se han borrado. Lo que tengas en tus equipos se queda en ellos.${PIE}`,
	}),
};
