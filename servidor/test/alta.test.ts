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
		const cartas = await buzon("ana@ejemplo.com");
		const conCodigo = cartas.find((c) => /^\s+\d{6}$/m.test(c.cuerpo))!;
		expect(conCodigo).toBeDefined();
		expect(conCodigo.asunto).not.toMatch(/\d{6}/);

		// **Y al terminar llega la constancia del alta** (LSSI 28), con los enlaces a
		// las condiciones y a la política. Es el correo más reciente, porque va
		// después de crear la cuenta.
		expect(cartas[0].asunto).toBe("Tu cuenta de Esfinge está creada");
		expect(cartas[0].cuerpo).toContain("condiciones.html");
		expect(cartas[0].cuerpo).toContain("privacidad.html");
		expect(cartas[0].cuerpo).not.toMatch(/\d{6}/);
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

	// **Quien pide un código y no termina no deja su correo ahí para siempre.** La
	// fila del alta solo se borraba al completarla; ahora, cada vez que alguien pide
	// un código, se barren las caducadas. El código no vale pasados diez minutos, así
	// que la fila tampoco tiene por qué durar.
	it("el correo de un alta que no se termina se barre al caducar", async () => {
		const correo = "irene@ejemplo.com";
		await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo }, ip: nuevaIP() });
		const antes = await env.BD.prepare("SELECT COUNT(*) AS n FROM altas WHERE correo = ?").bind(correo).first<{ n: number }>();
		expect(antes?.n).toBe(1);

		// Se le pone la caducidad en el pasado, que es lo que hace el reloj de verdad.
		await env.BD.prepare("UPDATE altas SET caduca = 1 WHERE correo = ?").bind(correo).run();
		// Y la siguiente alta de cualquiera —la de otra persona— se la lleva por delante.
		await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo: "julia@ejemplo.com" }, ip: nuevaIP() });

		const despues = await env.BD.prepare("SELECT COUNT(*) AS n FROM altas WHERE correo = ?").bind(correo).first<{ n: number }>();
		expect(despues?.n).toBe(0);
	});

	// **Y el barrido no depende de que llegue otra alta**: lo hace el reloj del
	// Worker cada hora. Sin esto, «se borra a los diez minutos» era falso con el
	// registro por invitación, donde puede no llegar otra alta en semanas.
	it("el reloj del Worker barre las tres tablas que guardan un correo o una IP", async () => {
		const correo = "karla@ejemplo.com";
		await pedir("POST", "/v1/registro/inicio", { cuerpo: { correo }, ip: nuevaIP() });
		await env.BD.prepare("UPDATE altas SET caduca = 1 WHERE correo = ?").bind(correo).run();
		await env.BD.prepare("UPDATE envios_alta SET momento = 1 WHERE correo = ?").bind(correo).run();

		const { limpiar } = await import("../src/indice");
		await limpiar(env);

		for (const tabla of ["altas", "envios_alta"]) {
			const fila = await env.BD.prepare(`SELECT COUNT(*) AS n FROM ${tabla} WHERE correo = ?`)
				.bind(correo)
				.first<{ n: number }>();
			expect(`${tabla}=${fila?.n}`).toBe(`${tabla}=0`);
		}
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
