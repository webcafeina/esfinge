import { describe, expect, it } from "vitest";
import { ARGON2, azarB64, boveda, buzon, darDeAlta, nuevaIP, pedir, ultimoCodigo } from "./ayuda";

const ID = "0123456789abcdef0123456789abcdef";

async function conBoveda(correo: string) {
	const a = await darDeAlta(correo, true);
	await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID, "", "ESF1.sobre-de-recuperacion"), cabeceras: { "If-Match": '"0"' } });
	return a;
}

/** Entra desde otro equipo, código incluido, y devuelve su sesión. */
async function otroEquipo(correo: string, claveDeAcceso: string, ip: string): Promise<string> {
	const r = await pedir("POST", "/v1/sesion", { ip, cuerpo: { correo, claveDeAcceso, dispositivo: "Otro" } });
	const { reto } = (await r.json()) as { reto: string };
	const hecho = await pedir("POST", "/v1/sesion/codigo", { ip, cuerpo: { reto, codigo: await ultimoCodigo(correo) } });
	return ((await hecho.json()) as { sesion: string }).sesion;
}

describe("cambiar la contraseña", () => {
	// **Y se lleva por delante los testigos de confianza de los demás equipos**
	// (revisión del 2026-09-23). Sin esto, quien hubiera pasado una vez el segundo
	// factor —o hubiera copiado `cuenta.json`, que lleva el testigo en claro— entraba
	// sin código en cuanto consiguiera la contraseña nueva, y cambiarla porque
	// alguien la sabe es justo ese caso.
	it("deja sin confianza a los demás equipos, y al que cambia no", async () => {
		const a = await conBoveda("elena@ejemplo.com");
		// Otro equipo entra con su código y se queda con su testigo de confianza.
		const r = await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, dispositivo: "El perdido" } });
		const { reto } = (await r.json()) as { reto: string };
		const hecho = await pedir("POST", "/v1/sesion/codigo", { ip: a.ip, cuerpo: { reto, codigo: await ultimoCodigo(a.correo), confiar: true } });
		const perdido = (await hecho.json()) as { confianza: string };
		expect(perdido.confianza).toMatch(/^c1\./);

		const nueva = azarB64(32);
		const cambio = await pedir("PUT", "/v1/cuenta/clave", {
			token: a.sesion,
			cuerpo: { posesion: a.posesion, sal: azarB64(16), argon2: ARGON2, claveDeAcceso: nueva, version: 1, documento: boveda(ID, "sin confianza") },
		});
		expect(cambio.status).toBe(200);

		// El equipo perdido, con la contraseña nueva y su testigo, tiene que pasar por
		// el código otra vez; el que cambió, no.
		const conTestigo = await pedir("POST", "/v1/sesion", {
			ip: a.ip,
			cuerpo: { correo: a.correo, claveDeAcceso: nueva, confianza: perdido.confianza, dispositivo: "El perdido" },
		});
		expect(conTestigo.status).toBe(202);
		const elQueCambio = await pedir("POST", "/v1/sesion", {
			ip: a.ip,
			cuerpo: { correo: a.correo, claveDeAcceso: nueva, confianza: a.confianza },
		});
		expect(elQueCambio.status).toBe(200);
	});

	it("es todo o nada: bóveda, verificador y sal nuevos, y las demás sesiones fuera", async () => {
		const a = await conBoveda("sara@ejemplo.com");
		const otra = await otroEquipo(a.correo, a.claveDeAcceso, a.ip);
		const nueva = azarB64(32);
		const sal = azarB64(16);
		const r = await pedir("PUT", "/v1/cuenta/clave", {
			token: a.sesion,
			cuerpo: { posesion: a.posesion, sal, argon2: ARGON2, claveDeAcceso: nueva, version: 1, documento: boveda(ID, "nueva") },
		});
		expect(r.status).toBe(200);
		expect(((await r.json()) as { version: number }).version).toBe(2);

		// La sesión de quien cambia sigue; la del otro equipo, no.
		expect((await pedir("GET", "/v1/boveda", { token: a.sesion })).status).toBe(200);
		expect((await pedir("GET", "/v1/boveda", { token: otra })).status).toBe(401);
		// La contraseña vieja ya no entra y la nueva sí; y la sal es la nueva.
		const vieja = await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, confianza: a.confianza } });
		expect(vieja.status).toBe(401);
		const buena = await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: nueva, confianza: a.confianza } });
		expect(buena.status).toBe(200);
		const pre = (await (await pedir("POST", "/v1/prelogin", { cuerpo: { correo: a.correo } })).json()) as { sal: string };
		expect(pre.sal).toBe(sal);
		const [carta] = await buzon(a.correo);
		expect(carta.asunto).toBe("Has cambiado la contraseña de Esfinge");
	});

	it("sin la prueba de posesión no cambia nada", async () => {
		const a = await conBoveda("tere@ejemplo.com");
		const r = await pedir("PUT", "/v1/cuenta/clave", {
			token: a.sesion,
			cuerpo: { posesion: azarB64(32), sal: azarB64(16), argon2: ARGON2, claveDeAcceso: azarB64(32), version: 1, documento: boveda(ID, "x") },
		});
		expect(r.status).toBe(403);
		expect((await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, confianza: a.confianza } })).status).toBe(200);
	});

	it("sobre una versión vieja no cambia nada, tampoco la contraseña", async () => {
		const a = await conBoveda("uma@ejemplo.com");
		const nueva = azarB64(32);
		const r = await pedir("PUT", "/v1/cuenta/clave", {
			token: a.sesion,
			cuerpo: { posesion: a.posesion, sal: azarB64(16), argon2: ARGON2, claveDeAcceso: nueva, version: 0, documento: boveda(ID, "x") },
		});
		expect(r.status).toBe(412);
		expect((await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: nueva, confianza: a.confianza } })).status).toBe(401);
	});
});

describe("recuperar la cuenta", () => {
	it("con el código entrega solo el sobre de recuperación, y con la posesión deja poner contraseña nueva", async () => {
		const a = await conBoveda("vera@ejemplo.com");
		const ip = nuevaIP();
		expect((await pedir("POST", "/v1/recuperacion/inicio", { ip, cuerpo: { correo: a.correo } })).status).toBe(202);
		const codigo = await ultimoCodigo(a.correo);
		const r = await pedir("POST", "/v1/recuperacion/codigo", { ip, cuerpo: { correo: a.correo, codigo } });
		expect(r.status).toBe(200);
		const d = (await r.json()) as { reto: string; sobre: { contenedor: string; codificacion: string } };
		expect(d.sobre).toEqual({ contenedor: "ESF1.sobre-de-recuperacion", codificacion: "palabras" });

		// Una posesión que no es la de esta bóveda no pasa.
		expect((await pedir("POST", "/v1/recuperacion/fin", { ip, cuerpo: { reto: d.reto, posesion: azarB64(32) } })).status).toBe(403);
		const fin = await pedir("POST", "/v1/recuperacion/fin", { ip, cuerpo: { reto: d.reto, posesion: a.posesion, dispositivo: "Portátil nuevo" } });
		expect(fin.status).toBe(200);
		const { sesion } = (await fin.json()) as { sesion: string };

		// La sesión de la recuperación solo sirve para bajar la bóveda y poner la contraseña nueva.
		expect((await pedir("GET", "/v1/boveda", { token: sesion })).status).toBe(200);
		expect((await pedir("GET", "/v1/dispositivos", { token: sesion })).status).toBe(403);
		expect((await pedir("PUT", "/v1/boveda", { token: sesion, crudo: boveda(ID, "x"), cabeceras: { "If-Match": '"1"' } })).status).toBe(403);

		const nueva = azarB64(32);
		const cambio = await pedir("PUT", "/v1/cuenta/clave", {
			token: sesion,
			cuerpo: { posesion: a.posesion, sal: azarB64(16), argon2: ARGON2, claveDeAcceso: nueva, version: 1, documento: boveda(ID, "recuperada") },
		});
		expect(cambio.status).toBe(200);
		const { sesion: normal } = (await cambio.json()) as { sesion: string };
		expect(normal).toMatch(/^s1\./);
		expect((await pedir("GET", "/v1/dispositivos", { token: normal })).status).toBe(200);
		expect((await pedir("GET", "/v1/boveda", { token: sesion })).status).toBe(401);
	});

	it("contesta igual con cuenta que sin ella, y sin cuenta no manda nada", async () => {
		const ip = nuevaIP();
		const sin = await pedir("POST", "/v1/recuperacion/inicio", { ip, cuerpo: { correo: "nadie2@ejemplo.com" } });
		expect(sin.status).toBe(202);
		expect(await buzon("nadie2@ejemplo.com")).toHaveLength(0);
		expect((await pedir("POST", "/v1/recuperacion/codigo", { ip, cuerpo: { correo: "nadie2@ejemplo.com", codigo: "123456" } })).status).toBe(401);
	});

	it("pedir otro código invalida el anterior", async () => {
		const a = await conBoveda("wen@ejemplo.com");
		const ip = nuevaIP();
		await pedir("POST", "/v1/recuperacion/inicio", { ip, cuerpo: { correo: a.correo } });
		const primero = await ultimoCodigo(a.correo);
		await pedir("POST", "/v1/recuperacion/inicio", { ip, cuerpo: { correo: a.correo } });
		const segundo = await ultimoCodigo(a.correo);
		if (primero !== segundo) {
			expect((await pedir("POST", "/v1/recuperacion/codigo", { ip, cuerpo: { correo: a.correo, codigo: primero } })).status).toBe(401);
		}
		expect((await pedir("POST", "/v1/recuperacion/codigo", { ip, cuerpo: { correo: a.correo, codigo: segundo } })).status).toBe(200);
	});
});
