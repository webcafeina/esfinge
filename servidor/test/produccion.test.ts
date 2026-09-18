import { createExecutionContext } from "cloudflare:test";
import { env } from "cloudflare:workers";
import { describe, expect, it } from "vitest";
import trabajador from "../src/indice";
import { peticion } from "./ayuda";

/** El Worker con otra configuración: la de producción, u otro modo de registro. */
function con(cambios: Partial<typeof env>, metodo: string, ruta: string, cuerpo?: unknown) {
	return trabajador.fetch(peticion(metodo, ruta, { cuerpo, ip: "10.99.0.1" }), { ...env, ...cambios }, createExecutionContext());
}

describe("lo que solo es de pruebas no está en producción", () => {
	it("el buzón de pruebas da 404 fuera del Worker de pruebas", async () => {
		expect((await con({ ENTORNO: "produccion", RESEND_API_KEY: "re_x" }, "GET", "/_pruebas/buzon?correo=a@ejemplo.com")).status).toBe(404);
		expect((await con({ ENTORNO: "pruebas" }, "GET", "/_pruebas/buzon?correo=a@ejemplo.com")).status).toBe(200);
	});

	it("sin sus secretos, el servidor no contesta nada", async () => {
		for (const falta of [
			{ PIMIENTA: undefined },
			{ PIMIENTA: "corta" },
			{ SECRETO_PRELOGIN: undefined },
			{ SECRETO_PRELOGIN: env.PIMIENTA },
			{ ENTORNO: "produccion", RESEND_API_KEY: undefined },
		]) {
			const r = await con(falta as Partial<typeof env>, "GET", "/v1/salud");
			expect(r.status, JSON.stringify(falta)).toBe(503);
		}
		expect((await con({ ENTORNO: "produccion", RESEND_API_KEY: "re_x" }, "GET", "/v1/salud")).status).toBe(200);
	});

	it("un fallo por dentro no cuenta nada en producción", async () => {
		const r = await con({ ENTORNO: "produccion", RESEND_API_KEY: "re_x", JURISDICCION: "us" }, "POST", "/v1/prelogin", { correo: "a@ejemplo.com" });
		expect(r.status).toBe(500);
		expect(Object.keys((await r.json()) as object)).toEqual(["error"]);
	});

	it("los dos Workers de verdad guardan las cuentas en la UE", () => {
		const sinComentarios = env.CONFIGURACION.replace(/^\s*\/\/.*$/gm, "");
		const conf = JSON.parse(sinComentarios);
		expect(conf.vars.JURISDICCION).toBe("eu");
		expect(conf.env.pruebas.vars.JURISDICCION).toBe("eu");
		// Y producción empieza por invitación: abrir el registro es después de la auditoría.
		expect(conf.vars.REGISTRO).toBe("lista");
		expect(conf.observability.enabled).toBe(false);
	});
});

describe("el registro por invitación", () => {
	it("deja entrar solo a lo que está en la lista, por correo o por dominio", async () => {
		await env.BD.prepare("INSERT INTO admision (patron) VALUES (?), (?)").bind("@webcafeina.com", "cliente@ejemplo.com").run();
		const lista = { REGISTRO: "lista" };
		expect((await con(lista, "POST", "/v1/registro/inicio", { correo: "info@webcafeina.com" })).status).toBe(202);
		expect((await con(lista, "POST", "/v1/registro/inicio", { correo: "cliente@ejemplo.com" })).status).toBe(202);
		const fuera = await con(lista, "POST", "/v1/registro/inicio", { correo: "otro@ejemplo.com" });
		expect(fuera.status).toBe(403);
		expect((await con({ REGISTRO: "cerrado" }, "POST", "/v1/registro/inicio", { correo: "info@webcafeina.com" })).status).toBe(403);
		expect((await (await con(lista, "GET", "/v1/salud")).json()) as object).toEqual({ registro: "lista", protocolo: 1 });
	});
});
