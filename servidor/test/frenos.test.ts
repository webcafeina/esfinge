import { describe, expect, it } from "vitest";
import { azarB64, darDeAlta, nuevaIP, pedir } from "./ayuda";

describe("los frenos", () => {
	it("por IP: pasado el tope por minuto, espera", async () => {
		const ip = nuevaIP();
		const estados: number[] = [];
		// Una petición nueva cada vez, que es como llegan de verdad: un freno que solo
		// aguanta dentro de una conexión no frena nada (la lección del canal).
		for (let i = 0; i < 25; i++) {
			estados.push((await pedir("POST", "/v1/prelogin", { ip, cuerpo: { correo: `x${i}@ejemplo.com` } })).status);
		}
		expect(estados.slice(0, 20).every((e) => e === 200)).toBe(true);
		expect(estados.slice(20)).toContain(429);
		// Y otra IP sigue pasando.
		expect((await pedir("POST", "/v1/prelogin", { ip: nuevaIP(), cuerpo: { correo: "y@ejemplo.com" } })).status).toBe(200);
	});

	it("por cuenta: tras diez fallos frena aun con la contraseña buena, desde IP distintas", async () => {
		const a = await darDeAlta("xavi@ejemplo.com", true);
		for (let i = 0; i < 10; i++) {
			const r = await pedir("POST", "/v1/sesion", { ip: nuevaIP(), cuerpo: { correo: a.correo, claveDeAcceso: azarB64(32) } });
			expect(r.status).toBe(401);
		}
		const buena = await pedir("POST", "/v1/sesion", { ip: nuevaIP(), cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso } });
		expect(buena.status).toBe(429);
		// Pero no es un cierre: el equipo de confianza entra. Bloquear la cuenta entera
		// sería regalar a cualquiera la forma de dejar fuera a su dueño.
		const deConfianza = await pedir("POST", "/v1/sesion", {
			ip: nuevaIP(),
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, confianza: a.confianza },
		});
		expect(deConfianza.status).toBe(200);
	});

	it("por cuenta: como mucho cinco códigos por hora", async () => {
		const a = await darDeAlta("yago@ejemplo.com");
		const estados: number[] = [];
		for (let i = 0; i < 6; i++) {
			estados.push(
				(await pedir("POST", "/v1/sesion", { ip: nuevaIP(), cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso } })).status,
			);
		}
		expect(estados).toEqual([202, 202, 202, 202, 202, 429]);
	});

	it("por correo: como mucho tres códigos de alta por hora", async () => {
		const estados: number[] = [];
		for (let i = 0; i < 4; i++) {
			estados.push((await pedir("POST", "/v1/registro/inicio", { ip: nuevaIP(), cuerpo: { correo: "zoe@ejemplo.com" } })).status);
		}
		expect(estados).toEqual([202, 202, 202, 429]);
	});

	it("por IP y día: como mucho tres altas", async () => {
		const ip = nuevaIP();
		const estados: number[] = [];
		for (let i = 0; i < 4; i++) {
			const correo = `alta${i}@ejemplo.com`;
			await pedir("POST", "/v1/registro/inicio", { ip, cuerpo: { correo } });
			const { ultimoCodigo, ARGON2 } = await import("./ayuda");
			const r = await pedir("POST", "/v1/registro/fin", {
				ip,
				cuerpo: { correo, codigo: await ultimoCodigo(correo), sal: azarB64(16), argon2: ARGON2, claveDeAcceso: azarB64(32), posesion: azarB64(32) },
			});
			estados.push(r.status);
		}
		expect(estados).toEqual([201, 201, 201, 429]);
	});
});
