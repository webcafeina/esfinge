import { describe, expect, it } from "vitest";

import { cabecerasDe, elSecretoCuadra, laDireccion, Rechazado, revisar, sinDirecciones } from "../src/relevo.js";

const DE = "Esfinge <esfinge@webcafeina.com>";

/** Un mensaje con la forma que saca `componer` en el Worker. */
const mensaje = (cabeceras: string[], cuerpo = "Hola.") => [...cabeceras, "", cuerpo].join("\r\n");

const bueno = (extra: string[] = []) =>
	mensaje([`From: ${DE}`, "To: quien@ejemplo.com", "Subject: Tu codigo", "MIME-Version: 1.0", ...extra]);

describe("las cabeceras", () => {
	it("se leen sin distinguir mayúsculas", () => {
		expect(cabecerasDe(mensaje(["FROM: a@b.com", "to: c@d.com"])).get("from")).toBe("a@b.com");
	});

	/**
	 * **El asunto largo sale partido en dos líneas** —lo exige el RFC 2047 y lo hace
	 * `componer`—, y la segunda empieza por espacio. Sin desdoblar, esa segunda mitad
	 * parecería una cabecera con un nombre rarísimo y, lo que importa, **la cabecera
	 * siguiente de verdad se leería mal**.
	 */
	it("desdobla una cabecera que sigue en la línea de abajo", () => {
		const c = cabecerasDe(mensaje(["Subject: =?UTF-8?B?dW5v?=", " =?UTF-8?B?ZG9z?=", `From: ${DE}`]));
		expect(c.get("subject")).toBe("=?UTF-8?B?dW5v?= =?UTF-8?B?ZG9z?=");
		expect(c.get("from")).toBe(DE);
	});

	/**
	 * Un `From` de más no puede ganarle al que se comprueba. Es el caso que convierte
	 * la revisión en un adorno: si valiera el último, bastaría añadir uno al final.
	 */
	it("se queda con la primera de dos cabeceras iguales", () => {
		expect(cabecerasDe(mensaje([`From: ${DE}`, "From: Otro <malo@ejemplo.com>"])).get("from")).toBe(DE);
	});

	it("no lee el cuerpo como cabeceras", () => {
		expect(cabecerasDe(bueno() + "\r\nBcc: otro@ejemplo.com").has("bcc")).toBe(false);
	});
});

describe("la dirección", () => {
	it("sale de dentro de los ángulos, o del campo entero", () => {
		expect(laDireccion("Quien <a@b.com>")).toBe("a@b.com");
		expect(laDireccion("  a@b.com ")).toBe("a@b.com");
	});

	it("no admite dos", () => {
		expect(laDireccion("a@b.com, c@d.com")).toBeNull();
		expect(laDireccion("a@b.com; c@d.com")).toBeNull();
	});

	it("no admite lo que no es una dirección", () => {
		expect(laDireccion("")).toBeNull();
		expect(laDireccion("sin-arroba")).toBeNull();
		expect(laDireccion("a@b")).toBeNull();
	});
});

describe("la revisión", () => {
	it("deja pasar lo que compone el Worker y dice a quién va", () => {
		expect(revisar(bueno(), DE)).toEqual({ para: "quien@ejemplo.com" });
	});

	/**
	 * **Lo que este servicio existe para impedir.** Con el secreto en la mano, un
	 * correo firmado por nuestro dominio —con SPF y DKIM en regla— diciendo ser un
	 * banco es exactamente la pieza que le falta a quien quiera pescar a los usuarios
	 * de un gestor de contraseñas.
	 */
	it("no deja mandar con otro nombre, aunque la dirección sea la nuestra", () => {
		const suplantado = mensaje(["From: Banco <esfinge@webcafeina.com>", "To: victima@ejemplo.com"]);
		expect(() => revisar(suplantado, DE)).toThrow(Rechazado);
	});

	it("no deja mandar desde otra dirección", () => {
		expect(() => revisar(mensaje(["From: otro@webcafeina.com", "To: a@b.com"]), DE)).toThrow(Rechazado);
	});

	it("no deja una tanda, ni escrita ni escondida", () => {
		expect(() => revisar(mensaje([`From: ${DE}`, "To: a@b.com, c@d.com"]), DE)).toThrow(Rechazado);
		expect(() => revisar(mensaje([`From: ${DE}`, "To: a@b.com", "Bcc: c@d.com"]), DE)).toThrow(Rechazado);
		expect(() => revisar(mensaje([`From: ${DE}`, "To: a@b.com", "Cc: c@d.com"]), DE)).toThrow(Rechazado);
	});

	it("no deja un mensaje sin destinatario", () => {
		expect(() => revisar(mensaje([`From: ${DE}`]), DE)).toThrow(Rechazado);
	});

	it("no deja un mensaje enorme", () => {
		expect(() => revisar(bueno() + "a".repeat(512 * 1024), DE)).toThrow(Rechazado);
	});
});

describe("el secreto", () => {
	it("cuadra consigo mismo y con nada más", () => {
		expect(elSecretoCuadra("abc123", "abc123")).toBe(true);
		expect(elSecretoCuadra("abc124", "abc123")).toBe(false);
		expect(elSecretoCuadra("abc", "abc123")).toBe(false);
		expect(elSecretoCuadra("abc1234", "abc123")).toBe(false);
	});

	/**
	 * Un secreto vacío es un secreto que falta, y entonces **cualquiera cuadraría**.
	 * El servicio ya no arranca sin él; esto es el cinturón del tirante.
	 */
	it("no cuadra con el vacío", () => {
		expect(elSecretoCuadra("", "")).toBe(false);
		expect(elSecretoCuadra("lo que sea", "")).toBe(false);
	});
});

describe("lo que se registra", () => {
	/**
	 * **La política de privacidad dice que este servicio no guarda nada**, y de los
	 * mensajes es verdad. Pero un rechazo de SMTP puede traer la dirección dentro del
	 * error, y el error sí se registra: Docker lo guarda. Se vio revisando el texto
	 * contra el código el 2026-09-29, no escribiéndolo.
	 */
	it("un error de SMTP no deja la dirección en el registro", () => {
		const e = "Error: 550 5.1.1 The email account that you tried to reach does not exist. quien@ejemplo.com";
		const limpio = sinDirecciones(e);
		expect(limpio).not.toContain("quien@ejemplo.com");
		expect(limpio).not.toContain("quien");
		expect(limpio, "el dominio se queda: sirve para diagnosticar y no dice a quién").toContain("ejemplo.com");
		expect(limpio).toContain("550 5.1.1");
	});

	/**
	 * **Lo que esto NO prueba, y se dice aquí:** que `cartero.ts` la llame. Estas
	 * pruebas son de la función; quitar el `sinDirecciones` del `console.error` las deja
	 * verdes igual —comprobado mutándolo—. Cerrarlo exige hacer inyectable el transporte
	 * y montar una petición de verdad contra el manejador, y eso es rehacer el servicio.
	 * Está apuntado en `docs/deuda.md`.
	 */
	it("tacha varias y no se come lo que no es una dirección", () => {
		expect(sinDirecciones("a@b.com y c@d.org")).toBe("…@b.com y …@d.org");
		expect(sinDirecciones("sin arrobas aquí")).toBe("sin arrobas aquí");
		expect(sinDirecciones("550 5.7.1 rechazado")).toBe("550 5.7.1 rechazado");
	});
});
