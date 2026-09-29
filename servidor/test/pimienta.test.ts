import { env } from "cloudflare:workers";
import { afterEach, describe, expect, it } from "vitest";

import { algunaPimientaDa, conLaPimienta, igualesHex, partir, sellar, versionActual } from "../src/pimienta";
import { ARGON2, azarB64, darDeAlta, nuevaIP, pedir, ultimoCodigo } from "./ayuda";

/**
 * Rotar la pimienta **en las pruebas** es cambiarle el entorno al Worker y al objeto
 * de la cuenta a la vez. Se hace mutando `env`, que es el mismo objeto que ve el
 * Durable Object: **comprobado ejecutándolo**, no deducido de la documentación del
 * motor de pruebas. Si algún día deja de ser cierto, esta prueba se pone roja por sí
 * sola y no en silencio — la de «entra y migra» fallaría.
 */
const original = { v: env.PIMIENTA_VERSION, a: env.PIMIENTA_ANTERIOR, p: env.PIMIENTA };

/** El entorno es de solo lectura para el tipo, pero el objeto es el mismo. */
const poner = (clave: string, valor: string | undefined) => {
	(env as unknown as Record<string, unknown>)[clave] = valor;
};

function rotar(nueva: string) {
	const antes = env.PIMIENTA;
	poner("PIMIENTA_ANTERIOR", antes);
	poner("PIMIENTA", nueva);
	poner("PIMIENTA_VERSION", String(versionActual(env) + 1));
}

afterEach(() => {
	poner("PIMIENTA_VERSION", original.v);
	poner("PIMIENTA_ANTERIOR", original.a);
	poner("PIMIENTA", original.p);
});

describe("cómo se guarda la versión", () => {
	it("va dentro del verificador, y lo de antes se lee como la 1", () => {
		expect(partir("abc123")).toEqual({ version: 1, resumen: "abc123" });
		expect(partir("2:abc123")).toEqual({ version: 2, resumen: "abc123" });
		expect(sellar(3, "abc")).toBe("3:abc");
	});

	/**
	 * Un prefijo que no es un número **no se cuela como versión 1**. Si se colara, un
	 * dato corrupto se compararía contra la pimienta de ahora y el fallo aparecería
	 * como «contraseña incorrecta», que es el mensaje equivocado.
	 */
	it("un prefijo que no es un número no se lee como la 1", () => {
		expect(partir("hola:abc").version).toBe(0);
		expect(partir("-1:abc").version).toBe(0);
	});
});

describe("qué pimienta se usa", () => {
	it("sin rotar nada, todo es la versión 1", () => {
		expect(versionActual(env)).toBe(1);
		expect(conLaPimienta(env, 1, "x")).resolves.not.toBeNull();
	});

	it("tras rotar, valen la de ahora y la de antes, y ninguna más", async () => {
		rotar("la-pimienta-nueva-0123456789abcdef-01234");
		expect(versionActual(env)).toBe(2);
		expect(await conLaPimienta(env, 2, "x")).not.toBeNull();
		expect(await conLaPimienta(env, 1, "x")).not.toBeNull();
		expect(await conLaPimienta(env, 0, "x")).toBeNull();
		expect(await conLaPimienta(env, 3, "x")).toBeNull();
	});

	/**
	 * **Solo dos generaciones, a propósito.** Si contestara a más, bastaría con
	 * olvidarse de mirar el contador para dejar cuentas colgadas sin que nada lo dijera.
	 */
	it("rotar dos veces deja fuera a la primera", async () => {
		rotar("la-segunda-0123456789abcdef0123-01234567");
		rotar("la-tercera-0123456789abcdef0123-01234567");
		expect(versionActual(env)).toBe(3);
		expect(await conLaPimienta(env, 1, "x")).toBeNull();
	});

	it("con las dos pimientas da lo de antes y lo de ahora", async () => {
		const conLaVieja = (await conLaPimienta(env, 1, "codigo|x|123456"))!;
		rotar("otra-mas-0123456789abcdef012345-01234567");
		expect(await algunaPimientaDa(env, "codigo|x|123456", conLaVieja)).toBe(true);
		const conLaNueva = (await conLaPimienta(env, 2, "codigo|x|123456"))!;
		expect(await algunaPimientaDa(env, "codigo|x|123456", conLaNueva)).toBe(true);
		expect(await algunaPimientaDa(env, "codigo|x|999999", conLaNueva)).toBe(false);
	});

	it("compara en tiempo constante y no por prefijo", () => {
		expect(igualesHex("abcd", "abcd")).toBe(true);
		expect(igualesHex("abcd", "abce")).toBe(false);
		expect(igualesHex("abcd", "abc")).toBe(false);
	});
});

describe("una cuenta que venía de la pimienta vieja", () => {
	/**
	 * **La prueba que justifica todo esto.** Antes de hoy, rotar la pimienta dejaba
	 * fuera a todas las cuentas: ésta se pone roja en el acto si eso vuelve a pasar.
	 */
	it("entra con su contraseña de siempre después de rotar", async () => {
		const a = await darDeAlta("vieja@ejemplo.com");
		rotar("la-pimienta-de-despues-01234567-01234567");

		const r = await pedir("POST", "/v1/sesion", {
			ip: nuevaIP(),
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso },
		});
		expect(r.status).toBe(202);
	});

	/** Y al entrar se reescribe sola: la segunda vez ya no hace falta la vieja. */
	it("se migra al entrar, y luego aguanta sin la pimienta vieja", async () => {
		const a = await darDeAlta("migra@ejemplo.com");
		rotar("la-que-migra-0123456789abcdef01-01234567");

		expect((await pedir("POST", "/v1/sesion", { ip: nuevaIP(), cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso } })).status).toBe(202);

		// Se tira la vieja, como se haría al terminar la rotación de verdad.
		poner("PIMIENTA_ANTERIOR", "");
		expect((await pedir("POST", "/v1/sesion", { ip: nuevaIP(), cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso } })).status).toBe(202);
	});

	/**
	 * Dos generaciones por detrás **no es «contraseña incorrecta»**. Decirle eso a
	 * alguien cuya contraseña está bien es el peor mensaje que se le puede dar: se pone
	 * a probar contraseñas y a dudar de su gestor de contraseñas.
	 */
	it("dos generaciones por detrás lo dice con otras palabras, no con un 401", async () => {
		const a = await darDeAlta("colgada@ejemplo.com");
		rotar("la-segunda-de-verdad-0123456789-01234567");
		rotar("la-tercera-de-verdad-0123456789-01234567");

		const r = await pedir("POST", "/v1/sesion", {
			ip: nuevaIP(),
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso },
		});
		expect(r.status).toBe(409);
		const { error } = (await r.json()) as { error: string };
		expect(error).not.toMatch(/contraseñ|incorrect|no vale/i);
		expect(error).toContain("info@webcafeina.com");
	});

	/** Un código que se mandó justo antes de rotar tiene que valer sus diez minutos. */
	it("un código de antes de rotar sigue valiendo", async () => {
		const correo = "codigo@ejemplo.com";
		expect((await pedir("POST", "/v1/registro/inicio", { ip: nuevaIP(), cuerpo: { correo } })).status).toBe(202);
		const codigo = await ultimoCodigo(correo);

		rotar("rotada-en-medio-0123456789abcd-012345678");

		const r = await pedir("POST", "/v1/registro/fin", {
			ip: nuevaIP(),
			cuerpo: { correo, codigo, sal: azarB64(16), argon2: ARGON2, claveDeAcceso: azarB64(32), posesion: azarB64(32) },
		});
		expect(r.status).toBe(201);
	});

	/**
	 * Y lo que decide cuándo se puede borrar la pimienta vieja: **que se pueda contar**.
	 * Sin esto, la decisión sería una fecha a ciegas.
	 */
	it("D1 apunta por dónde va cada cuenta, para poder contarlas", async () => {
		const a = await darDeAlta("contable@ejemplo.com");
		const dela = async () =>
			(await env.BD.prepare("SELECT pimienta FROM cuentas WHERE correo = ?").bind(a.correo).first<{ pimienta: number }>())?.pimienta;

		expect(await dela()).toBe(1);
		rotar("la-que-se-cuenta-0123456789abcd-01234567");
		await pedir("POST", "/v1/sesion", { ip: nuevaIP(), cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso } });

		// Solo migró el de acceso; el de posesión sigue en la 1, así que la menor es 1.
		// **Ése es el punto**: con una sola marca por cuenta, esto diría 2 y sería mentira.
		expect(await dela()).toBe(1);
	});
});
