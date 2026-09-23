import { describe, expect, it } from "vitest";
import { azarB64, buzon, darDeAlta, nuevaIP, pedir, ultimoCodigo } from "./ayuda";

describe("entrar", () => {
	it("con una contraseña mala no entra, y no se distingue de un correo sin cuenta", async () => {
		await darDeAlta("gil@ejemplo.com");
		const ip = nuevaIP();
		const mala = await pedir("POST", "/v1/sesion", { ip, cuerpo: { correo: "gil@ejemplo.com", claveDeAcceso: azarB64(32) } });
		const nadie = await pedir("POST", "/v1/sesion", { ip, cuerpo: { correo: "nadie@ejemplo.com", claveDeAcceso: azarB64(32) } });
		expect(mala.status).toBe(401);
		expect(nadie.status).toBe(401);
		expect(await mala.text()).toBe(await nadie.text());
	});

	// **El freno no puede delatar que la cuenta existe** (revisión del 2026-09-23).
	// Hasta entonces, tras diez fallos una cuenta real contestaba 429 y un correo
	// desconocido contestaba siempre 401: once intentos con una contraseña inventada
	// decían si ese correo está registrado.
	it("tras muchos fallos sigue sin distinguirse de un correo sin cuenta", async () => {
		const a = await darDeAlta("frenada@ejemplo.com");
		const ip = nuevaIP();
		for (let i = 0; i < 12; i++) {
			await pedir("POST", "/v1/sesion", { ip, cuerpo: { correo: a.correo, claveDeAcceso: azarB64(32) } });
		}
		const mala = await pedir("POST", "/v1/sesion", { ip, cuerpo: { correo: a.correo, claveDeAcceso: azarB64(32) } });
		const nadie = await pedir("POST", "/v1/sesion", { ip, cuerpo: { correo: "tampoco@ejemplo.com", claveDeAcceso: azarB64(32) } });
		expect(mala.status).toBe(401);
		expect(nadie.status).toBe(401);
		expect(await mala.text()).toBe(await nadie.text());

		// Y a quien acierta la contraseña sí se le dice que está frenada: ése ya sabe
		// que la cuenta existe, y necesita entender por qué no entra.
		const buena = await pedir("POST", "/v1/sesion", {
			ip,
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, dispositivo: "Uno nuevo" },
		});
		expect(buena.status).toBe(429);
	});

	it("desde un equipo nuevo pide el código **después** de comprobar la contraseña", async () => {
		const a = await darDeAlta("hugo@ejemplo.com");
		const antes = (await buzon(a.correo)).length;
		// Con la contraseña mala no sale ningún correo: nadie puede bombardear el buzón de otro.
		await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: azarB64(32) } });
		expect((await buzon(a.correo)).length).toBe(antes);

		const r = await pedir("POST", "/v1/sesion", {
			ip: a.ip,
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, dispositivo: "Mac de la oficina" },
		});
		expect(r.status).toBe(202);
		const { reto } = (await r.json()) as { reto: string };
		const [carta] = await buzon(a.correo);
		expect(carta.cuerpo).toContain("Mac de la oficina");

		const codigo = await ultimoCodigo(a.correo);
		const hecho = await pedir("POST", "/v1/sesion/codigo", { ip: a.ip, cuerpo: { reto, codigo, confiar: true } });
		expect(hecho.status).toBe(200);
		const d = (await hecho.json()) as { sesion: string; confianza: string };
		expect(d.sesion).toMatch(/^s1\./);
		expect(d.confianza).toMatch(/^c1\./);

		// Y con el testigo de confianza, la vez siguiente ya no hay código.
		const otra = await pedir("POST", "/v1/sesion", {
			ip: a.ip,
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, confianza: d.confianza },
		});
		expect(otra.status).toBe(200);
	});

	it("un código sirve una sola vez, y a los cinco fallos el reto muere", async () => {
		const a = await darDeAlta("ines@ejemplo.com");
		const pedirReto = async () => {
			const r = await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso } });
			return ((await r.json()) as { reto: string }).reto;
		};
		const reto = await pedirReto();
		const codigo = await ultimoCodigo(a.correo);
		expect((await pedir("POST", "/v1/sesion/codigo", { ip: a.ip, cuerpo: { reto, codigo } })).status).toBe(200);
		expect((await pedir("POST", "/v1/sesion/codigo", { ip: a.ip, cuerpo: { reto, codigo } })).status).toBe(401);

		const otro = await pedirReto();
		const bueno = await ultimoCodigo(a.correo);
		const malo = bueno === "123456" ? "654321" : "123456";
		for (let i = 0; i < 5; i++) {
			expect((await pedir("POST", "/v1/sesion/codigo", { ip: a.ip, cuerpo: { reto: otro, codigo: malo } })).status).toBe(401);
		}
		expect((await pedir("POST", "/v1/sesion/codigo", { ip: a.ip, cuerpo: { reto: otro, codigo: bueno } })).status).toBe(401);
	});

	it("cerrar la sesión la deja sin valor", async () => {
		const a = await darDeAlta("jon@ejemplo.com");
		expect((await pedir("DELETE", "/v1/sesion", { token: a.sesion })).status).toBe(200);
		expect((await pedir("GET", "/v1/dispositivos", { token: a.sesion })).status).toBe(401);
	});

	it("un testigo inventado no abre nada", async () => {
		const a = await darDeAlta("kai@ejemplo.com");
		const falso = `s1.${a.cuenta}.${azarB64(32)}`;
		expect((await pedir("GET", "/v1/boveda", { token: falso })).status).toBe(401);
		expect((await pedir("GET", "/v1/boveda", { token: "basura" })).status).toBe(401);
		expect((await pedir("GET", "/v1/boveda")).status).toBe(401);
	});
});
