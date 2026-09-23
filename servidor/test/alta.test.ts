import { env } from "cloudflare:test";
import { describe, expect, it } from "vitest";
import { ARGON2, azarB64, buzon, darDeAlta, nuevaIP, pedir, ultimoCodigo } from "./ayuda";

describe("la salud", () => {
	it("dice cómo está el registro", async () => {
		const r = await pedir("GET", "/v1/salud");
		expect(r.status).toBe(200);
		expect(await r.json()).toEqual({ registro: "abierto", protocolo: 1 });
	});
});

describe("el alta", () => {
	it("crea la cuenta con el código del correo, y el código no va en el asunto", async () => {
		const a = await darDeAlta("ana@ejemplo.com");
		expect(a.cuenta).toMatch(/^[0-9a-f]{32}$/);
		expect(a.sesion).toMatch(new RegExp(`^s1\\.${a.cuenta}\\.`));
		const [carta] = await buzon("ana@ejemplo.com");
		expect(carta.asunto).not.toMatch(/\d{6}/);
		expect(carta.cuerpo).toMatch(/^\s+\d{6}$/m);
	});

	it("normaliza el correo: mayúsculas y espacios son la misma cuenta", async () => {
		await darDeAlta("bea@ejemplo.com");
		const ip = nuevaIP();
		const r = await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo: "  BEA@Ejemplo.com " }, ip });
		expect(r.status).toBe(202);
		const [carta] = await buzon("bea@ejemplo.com");
		expect(carta.asunto).toBe("Ya tienes una cuenta de Esfinge");
	});

	it("contesta lo mismo haya cuenta o no, y lo cuenta en el buzón", async () => {
		await darDeAlta("carla@ejemplo.com");
		const conCuenta = await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo: "carla@ejemplo.com" }, ip: nuevaIP() });
		const sinCuenta = await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo: "nadie@ejemplo.com" }, ip: nuevaIP() });
		expect(conCuenta.status).toBe(sinCuenta.status);
		expect(await conCuenta.text()).toBe(await sinCuenta.text());
	});

	it("un código malo no vale, y a los cinco intentos tampoco el bueno", async () => {
		const correo = "dani@ejemplo.com";
		const ip = nuevaIP();
		await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo }, ip });
		const bueno = await ultimoCodigo(correo);
		const malo = bueno === "000000" ? "111111" : "000000";
		const cuerpo = (codigo: string) => ({
			correo, codigo, sal: azarB64(16), argon2: ARGON2, claveDeAcceso: azarB64(32), posesion: azarB64(32),
		});
		for (let i = 0; i < 5; i++) {
			const r = await pedir("POST", "/v1/registro/fin", { cuerpo: cuerpo(malo), ip });
			expect(r.status).toBe(401);
		}
		const r = await pedir("POST", "/v1/registro/fin", { cuerpo: cuerpo(bueno), ip });
		expect(r.status).toBe(401);
	});

	// **Los cinco intentos tienen que ser cinco también en una ráfaga.** Leer,
	// comparar y sumar el intento eran tres viajes a D1: veinte peticiones a la vez
	// leían todas `intentos = 0` y todas comparaban su código, así que un código de
	// seis cifras se probaba veinte veces con cinco de cuenta. Se mira el contador,
	// que es lo único que distingue un caso del otro: con la sentencia atómica se
	// queda clavado en cinco.
	it("veinte intentos a la vez gastan cinco, no veinte", async () => {
		const correo = "gonzalo@ejemplo.com";
		await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo }, ip: nuevaIP() });
		const bueno = await ultimoCodigo(correo);
		const cuerpo = (codigo: string) => ({
			correo, codigo, sal: azarB64(16), argon2: ARGON2, claveDeAcceso: azarB64(32), posesion: azarB64(32),
		});
		// Cada una con su IP: aquí se mide el contador de la fila, no el freno.
		await Promise.all(
			Array.from({ length: 20 }, (_, i) =>
				pedir("POST", "/v1/registro/fin", { cuerpo: cuerpo(String(100000 + i)), ip: nuevaIP() }),
			),
		);
		const fila = await env.BD.prepare("SELECT intentos FROM altas WHERE correo = ?").bind(correo).first<{ intentos: number }>();
		expect(fila?.intentos).toBe(5);
		// Y el bueno ya no entra: los cinco están gastados.
		expect((await pedir("POST", "/v1/registro/fin", { cuerpo: cuerpo(bueno), ip: nuevaIP() })).status).toBe(401);
	});

	// El otro lado del mismo asunto: los cinco de la fila vuelven a cero al pedir
	// otro código, así que la ráfaga desde una IP la tiene que parar el freno.
	it("probar códigos a lo bruto desde una IP se frena", async () => {
		const ip = nuevaIP();
		const estados: number[] = [];
		for (let i = 0; i < 25; i++) {
			const r = await pedir("POST", "/v1/registro/fin", {
				ip,
				cuerpo: { correo: "hugo@ejemplo.com", codigo: String(100000 + i), sal: azarB64(16), argon2: ARGON2, claveDeAcceso: azarB64(32), posesion: azarB64(32) },
			});
			estados.push(r.status);
		}
		expect(estados).toContain(429);
		expect(estados.indexOf(429)).toBeLessThanOrEqual(20);
	});

	it("no acepta un coste de Argon2id por debajo del de Esfinge", async () => {
		const correo = "eva@ejemplo.com";
		const ip = nuevaIP();
		await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo }, ip });
		const codigo = await ultimoCodigo(correo);
		const r = await pedir("POST", "/v1/registro/fin", {
			ip,
			cuerpo: {
				correo, codigo, sal: azarB64(16), claveDeAcceso: azarB64(32), posesion: azarB64(32),
				argon2: { memoria: 8192, pasadas: 1, paralelismo: 1 },
			},
		});
		expect(r.status).toBe(400);
	});

	it("rechaza un correo que no lo es", async () => {
		const r = await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo: "no es un correo" }, ip: nuevaIP() });
		expect(r.status).toBe(400);
		expect(((await r.json()) as { error: string }).error).toMatch(/^[A-ZÁÉÍÓÚÑ]/);
	});
});

describe("la pre-entrada", () => {
	it("no distingue un correo con cuenta de uno sin ella", async () => {
		const a = await darDeAlta("fer@ejemplo.com");
		const con = (await (await pedir("POST", "/v1/prelogin", { cuerpo: { correo: a.correo } })).json()) as Record<string, unknown>;
		const sin = (await (await pedir("POST", "/v1/prelogin", { cuerpo: { correo: "fantasma@ejemplo.com" } })).json()) as Record<string, unknown>;
		expect(con.sal).toBe(a.sal);
		expect(Object.keys(sin).sort()).toEqual(Object.keys(con).sort());
		expect(sin.argon2).toEqual(con.argon2);
		// Y la sal inventada es siempre la misma: preguntar dos veces no la delata.
		const otra = (await (await pedir("POST", "/v1/prelogin", { cuerpo: { correo: "fantasma@ejemplo.com" } })).json()) as Record<string, unknown>;
		expect(otra.sal).toBe(sin.sal);
	});
});
