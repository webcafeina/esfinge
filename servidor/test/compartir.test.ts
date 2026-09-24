import { env } from "cloudflare:test";
import { describe, expect, it } from "vitest";
import { limpiar } from "../src/indice";
import { azarB64, buzon, darDeAlta, nuevaIP, pedir } from "./ayuda";

/**
 * Compartir copias: publicar las llaves, preguntar por las de otro y el buzón
 * (ADR 0043, entrega B2).
 *
 * **El servidor no abre ni un sobre**: aquí se prueba que los lleva y los cuenta,
 * y sobre todo que **preguntar por un correo no dice si tiene cuenta**, que es lo
 * que el resto del servidor cuida y lo que compartir podría estropear.
 */
const LLAVES = { suite: "DHKEM(X25519)/HKDF-SHA256/ChaCha20-Poly1305", cifrado: azarB64(32), firma: azarB64(32) };
const sobre = (relleno = "") => ({
	esfinge: "envío", version: 1, suite: LLAVES.suite,
	de: { cifrado: azarB64(32), firma: azarB64(32) },
	para: LLAVES.cifrado, enc: azarB64(32), cuerpo: azarB64(48) + relleno, firma: azarB64(64),
});

describe("compartir", () => {
	it("publica las llaves y otro las encuentra", async () => {
		const ana = await darDeAlta("ana-comparte@ejemplo.com");
		const luis = await darDeAlta("luis-comparte@ejemplo.com");

		expect((await pedir("PUT", "/v1/llaves", { token: ana.sesion, cuerpo: { llaves: LLAVES } })).status).toBe(200);
		const r = await pedir("POST", "/v1/llaves/de", { token: luis.sesion, ip: nuevaIP(), cuerpo: { correo: ana.correo } });
		expect(r.status).toBe(200);
		expect(((await r.json()) as { llaves: unknown }).llaves).toEqual(LLAVES);
	});

	// **Lo que más importa de este fichero.** Si preguntar delatara, compartir
	// desharía la protección contra la enumeración de correos.
	it("preguntar por un correo sin cuenta contesta igual, y siempre lo mismo", async () => {
		const quien = await darDeAlta("quien-pregunta@ejemplo.com");
		const con = await pedir("POST", "/v1/llaves/de", { token: quien.sesion, ip: nuevaIP(), cuerpo: { correo: quien.correo } });
		const sin = await pedir("POST", "/v1/llaves/de", { token: quien.sesion, ip: nuevaIP(), cuerpo: { correo: "fantasma@ejemplo.com" } });
		expect(sin.status).toBe(con.status);

		const unas = (await sin.json()) as { llaves: { suite: string; cifrado: string; firma: string } };
		expect(unas.llaves.suite).toBe(LLAVES.suite);
		expect(unas.llaves.cifrado).toMatch(/^[A-Za-z0-9_-]{43}$/);

		// Y dos veces lo mismo: si cambiaran, preguntar dos veces delataría.
		const otra = await pedir("POST", "/v1/llaves/de", { token: quien.sesion, ip: nuevaIP(), cuerpo: { correo: "fantasma@ejemplo.com" } });
		expect(await otra.json()).toEqual(unas);
	});

	it("un envío llega al buzón de quien lo recibe, y de nadie más", async () => {
		const ana = await darDeAlta("ana-buzon@ejemplo.com");
		const luis = await darDeAlta("luis-buzon@ejemplo.com");
		await pedir("PUT", "/v1/llaves", { token: ana.sesion, cuerpo: { llaves: LLAVES } });

		const mandado = await pedir("POST", "/v1/envios", {
			token: luis.sesion, ip: nuevaIP(), cuerpo: { para: ana.correo, sobre: sobre() },
		});
		expect(mandado.status).toBe(202);

		const suyo = (await (await pedir("GET", "/v1/buzon", { token: ana.sesion })).json()) as {
			envios: { id: string; sobre: { cuerpo: string } }[];
		};
		expect(suyo.envios).toHaveLength(1);
		expect(suyo.envios[0].sobre.cuerpo).toBeTruthy();

		// El que lo mandó no lo tiene en el suyo.
		const delOtro = (await (await pedir("GET", "/v1/buzon", { token: luis.sesion })).json()) as { envios: unknown[] };
		expect(delOtro.envios).toHaveLength(0);

		// Y se puede tirar —que es también lo que se hace al aceptarlo—.
		expect((await pedir("DELETE", `/v1/buzon/${suyo.envios[0].id}`, { token: ana.sesion })).status).toBe(200);
		const vacio = (await (await pedir("GET", "/v1/buzon", { token: ana.sesion })).json()) as { envios: unknown[] };
		expect(vacio.envios).toHaveLength(0);
	});

	it("mandar a un correo sin cuenta contesta lo mismo que mandar a uno con ella", async () => {
		const luis = await darDeAlta("luis-manda@ejemplo.com");
		const ana = await darDeAlta("ana-recibe@ejemplo.com");
		const conCuenta = await pedir("POST", "/v1/envios", { token: luis.sesion, ip: nuevaIP(), cuerpo: { para: ana.correo, sobre: sobre() } });
		const sinCuenta = await pedir("POST", "/v1/envios", { token: luis.sesion, ip: nuevaIP(), cuerpo: { para: "nadie-de-nada@ejemplo.com", sobre: sobre() } });
		expect(sinCuenta.status).toBe(conCuenta.status);
		expect(await sinCuenta.text()).toBe(await conCuenta.text());
	});

	it("a quien no tiene cuenta le llega una invitación, con quién le invita y dónde crearla", async () => {
		const luis = await darDeAlta("luis-invita@ejemplo.com");
		const r = await pedir("POST", "/v1/envios", {
			token: luis.sesion, ip: nuevaIP(), cuerpo: { para: "sin-cuenta@ejemplo.com", sobre: sobre() },
		});
		expect(r.status).toBe(202);

		const [carta] = await buzon("sin-cuenta@ejemplo.com");
		expect(carta.asunto).toBe("luis-invita@ejemplo.com te quiere mandar una contraseña");
		expect(carta.cuerpo).toContain("luis-invita@ejemplo.com");
		expect(carta.cuerpo).toContain("https://webcafeina.github.io/esfinge/#descargar");
		// **Del sobre no sale nada**: espera en la bóveda de quien lo manda, y el
		// servidor ni lo guarda. Si algo suyo apareciera aquí, la contraseña estaría
		// viajando por correo, que es justo lo que este camino evita.
		expect(carta.cuerpo).not.toContain("esfinge/envío");
	});

	it("invitar dos veces a la misma dirección manda un solo correo", async () => {
		const luis = await darDeAlta("luis-dos-veces@ejemplo.com");
		for (let i = 0; i < 3; i++) {
			await pedir("POST", "/v1/envios", {
				token: luis.sesion, ip: nuevaIP(), cuerpo: { para: "una-sola@ejemplo.com", sobre: sobre() },
			});
		}
		expect(await buzon("una-sola@ejemplo.com")).toHaveLength(1);
	});

	// **Lo que no puede pasar aunque el freno salte.** Si pasarse del tope diera otra
	// respuesta, mandar envíos sería otra forma de averiguar quién tiene cuenta.
	it("pasarse del tope de invitaciones no cambia la respuesta: deja de mandar y calla", async () => {
		const luis = await darDeAlta("luis-tope-invita@ejemplo.com");
		const ana = await darDeAlta("ana-con-cuenta@ejemplo.com");
		const respuestas: string[] = [];
		for (let i = 0; i < 8; i++) {
			const r = await pedir("POST", "/v1/envios", {
				token: luis.sesion, ip: nuevaIP(), cuerpo: { para: `gente-${i}@ejemplo.com`, sobre: sobre() },
			});
			respuestas.push(`${r.status} ${await r.text()}`);
		}
		const conCuenta = await pedir("POST", "/v1/envios", {
			token: luis.sesion, ip: nuevaIP(), cuerpo: { para: ana.correo, sobre: sobre() },
		});
		expect(new Set(respuestas).size).toBe(1);
		expect(`${conCuenta.status} ${await conCuenta.text()}`).toBe(respuestas[0]);

		// Cinco invitaciones al día: las tres últimas no han mandado nada.
		const mandadas = await Promise.all([...Array(8).keys()].map((i) => buzon(`gente-${i}@ejemplo.com`)));
		expect(mandadas.filter((m) => m.length > 0)).toHaveLength(5);
	});

	it("la limpieza se lleva las invitaciones caducadas", async () => {
		await env.BD.prepare("INSERT INTO invitaciones (correo, de_cuenta, caduca) VALUES (?, ?, ?)")
			.bind("vieja@ejemplo.com", "c".repeat(32), Date.now() - 1000)
			.run();
		await limpiar(env);
		const queda = await env.BD.prepare("SELECT 1 FROM invitaciones WHERE correo = ?").bind("vieja@ejemplo.com").first();
		expect(queda).toBeNull();
	});

	it("sin sesión no se puede preguntar ni mandar ni mirar el buzón", async () => {
		for (const [metodo, ruta, cuerpo] of [
			["POST", "/v1/llaves/de", { correo: "ana@ejemplo.com" }],
			["POST", "/v1/envios", { para: "ana@ejemplo.com", sobre: sobre() }],
			["GET", "/v1/buzon", undefined],
			["PUT", "/v1/llaves", { llaves: LLAVES }],
		] as const) {
			const r = await pedir(metodo, ruta, { ip: nuevaIP(), cuerpo });
			expect(`${ruta}=${r.status}`).toBe(`${ruta}=401`);
		}
	});

	it("un sobre demasiado grande no entra, y el buzón tiene tope", async () => {
		const ana = await darDeAlta("ana-tope@ejemplo.com");
		const luis = await darDeAlta("luis-tope@ejemplo.com");
		const enorme = await pedir("POST", "/v1/envios", {
			token: luis.sesion, ip: nuevaIP(), cuerpo: { para: ana.correo, sobre: sobre("x".repeat(70 * 1024)) },
		});
		expect(enorme.status).toBe(413);
	});

	it("unas llaves que no lo son se rechazan", async () => {
		const ana = await darDeAlta("ana-llaves-malas@ejemplo.com");
		for (const llaves of [{ suite: "x", cifrado: "corto", firma: azarB64(32) }, { suite: "", cifrado: azarB64(32), firma: azarB64(32) }, null]) {
			const r = await pedir("PUT", "/v1/llaves", { token: ana.sesion, cuerpo: { llaves } });
			expect(r.status).toBe(400);
		}
	});
});
