import { runInDurableObject } from "cloudflare:test";
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
const original = { version: env.PIMIENTA_VERSION ?? "", uno: env.PIMIENTA ?? "" };
/** Las que estas pruebas inventan, para poder quitarlas todas al terminar. */
const inventadas: string[] = [];

/**
 * El entorno es de solo lectura para el tipo, pero el objeto es el mismo.
 *
 * **Se borra poniendo `""`, no `undefined`**: un binding puesto a `undefined` no
 * desaparece, así que la versión se iba acumulando entre pruebas y las últimas pedían
 * una `PIMIENTA_4` que no existe. Se vio midiendo qué veía cada lado, no leyendo.
 */
const poner = (clave: string, valor: string) => {
	(env as unknown as Record<string, unknown>)[clave] = valor;
};

/**
 * **Rotar es añadir**, nunca mover: un secreto de Cloudflare se escribe y no se puede
 * volver a leer, así que «copia el valor de una variable a otra» no es un paso que
 * nadie pueda dar. La versión 1 se queda en `PIMIENTA` para siempre.
 */
function rotar(nueva: string) {
	const n = versionActual(env) + 1;
	inventadas.push(`PIMIENTA_${n}`);
	poner(`PIMIENTA_${n}`, nueva);
	poner("PIMIENTA_VERSION", String(n));
}

afterEach(() => {
	poner("PIMIENTA_VERSION", original.version);
	// **Y la 1 también**, que una de estas pruebas la borra para simular el final de una
	// rotación. Sin esta línea, la siguiente prueba se encuentra el servidor sin
	// configurar y falla en el alta, antes de llegar a lo suyo.
	poner("PIMIENTA", original.uno);
	for (const k of inventadas.splice(0)) poner(k, "");
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

		// Se tira la vieja, como se haría al terminar la rotación de verdad: se **borra
		// la variable**, que sí se puede hacer sin conocer su valor.
		poner("PIMIENTA", "");
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
	 *
	 * **Las dos mitades son la historia de un error.** Entrar migra el acceso y deja la
	 * cuenta en 1, porque la menor de las dos sigue siendo la del verificador de
	 * posesión: con una marca por cuenta esto diría 2 y sería mentira. Y la segunda
	 * mitad es lo que hizo falta añadir: la primera versión daba por hecho que la
	 * posesión migraba «en cada pasada de la sincronización», y era falso —solo se
	 * comprueba al cambiar la maestra y al recuperar—, así que **el contador no habría
	 * llegado a cero nunca**. Se vio al ir a rotar de verdad.
	 */
	it("D1 apunta la menor, y llega a la nueva cuando van las dos", async () => {
		const a = await darDeAlta("contable@ejemplo.com");
		const dela = async () =>
			(await env.BD.prepare("SELECT pimienta FROM cuentas WHERE correo = ?").bind(a.correo).first<{ pimienta: number }>())?.pimienta;

		expect(await dela()).toBe(1);
		rotar("la-que-se-cuenta-0123456789abcd-01234567");

		await pedir("POST", "/v1/sesion", { ip: nuevaIP(), cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso } });
		expect(await dela(), "entrar solo migra el acceso").toBe(1);

		const r = await pedir("PUT", "/v1/posesion", { token: a.sesion, cuerpo: { posesion: a.posesion } });
		expect(r.status).toBe(200);
		expect(await dela(), "y la cortesía de la sincronización migra la posesión").toBe(2);
	});

	/**
	 * **Contesta lo mismo cuadre o no**, y esto lo vigila: si una posesión mala se
	 * distinguiera de una buena, esto sería un sitio donde probarlas con una sesión
	 * robada.
	 */
	it("una posesión equivocada contesta igual, y no migra nada", async () => {
		const a = await darDeAlta("posesion-mala@ejemplo.com");
		rotar("da-igual-la-posesion-0123456789-01234567");

		const r = await pedir("PUT", "/v1/posesion", { token: a.sesion, cuerpo: { posesion: azarB64(32) } });
		expect(r.status).toBe(200);

		const ns = env.CUENTAS;
		const guardado = await runInDurableObject(ns.get(ns.idFromName(a.cuenta)), (_i, ctx) =>
			ctx.storage.sql.exec<{ valor: string }>("SELECT valor FROM ajustes WHERE clave = 'posesion'").one().valor,
		);
		expect(guardado.startsWith("1:"), "una posesión mala no puede migrar nada").toBe(true);
	});

	/**
	 * Y sin sesión no se toca nada — **probado contra el objeto directamente**.
	 *
	 * Desde fuera no se puede: el Worker rechaza cualquier token que no pase
	 * `cuentaDeSesion`, así que la comprobación de dentro del objeto **es inobservable**
	 * por HTTP y una prueba que fuera por ahí pasaría igual sin ella. Se descubrió
	 * mutando: quitada la comprobación, la respuesta seguía siendo `401`, pero la daba
	 * otro. Es defensa en profundidad y se prueba donde vive.
	 */
	it("el objeto no refresca nada con una sesión que no vale", async () => {
		const a = await darDeAlta("sin-sesion@ejemplo.com");
		rotar("hace-falta-sesion-0123456789abc-01234567");

		const ns = env.CUENTAS;
		const stub = ns.get(ns.idFromName(a.cuenta));
		const r = await runInDurableObject(stub, (c: { refrescarPosesion: (t: string, p: unknown) => Promise<{ ok: boolean }> }) =>
			c.refrescarPosesion(`${a.cuenta}.${azarB64(32)}`, a.posesion),
		);
		expect(r.ok, "una sesión que no vale no puede refrescar").toBe(false);

		const guardado = await runInDurableObject(stub, (_i, ctx) =>
			ctx.storage.sql.exec<{ valor: string }>("SELECT valor FROM ajustes WHERE clave = 'posesion'").one().valor,
		);
		expect(guardado.startsWith("1:"), "sin sesión no se puede migrar nada").toBe(true);
	});
});
