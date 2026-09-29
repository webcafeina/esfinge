import { createExecutionContext } from "cloudflare:test";
import { env } from "cloudflare:workers";
import { describe, expect, it } from "vitest";
import trabajador from "../src/indice";
import { modoDeCorreo } from "../src/correo";
import { peticion } from "./ayuda";

/** El Worker con otra configuración: la de producción, u otro modo de registro. */
function con(cambios: Partial<typeof env>, metodo: string, ruta: string, cuerpo?: unknown) {
	return trabajador.fetch(peticion(metodo, ruta, { cuerpo, ip: "10.99.0.1" }), { ...env, ...cambios }, createExecutionContext());
}

describe("lo que solo es de pruebas no está en producción", () => {
	it("el buzón de pruebas da 404 fuera del Worker de pruebas", async () => {
		expect((await con({ ENTORNO: "produccion", CARTERO_SECRETO: "el-secreto-del-cartero" }, "GET", "/_pruebas/buzon?correo=a@ejemplo.com")).status).toBe(404);
		expect((await con({ ENTORNO: "pruebas" }, "GET", "/_pruebas/buzon?correo=a@ejemplo.com")).status).toBe(200);
	});

	it("sin sus secretos, el servidor no contesta nada", async () => {
		for (const falta of [
			{ PIMIENTA: undefined },
			{ PIMIENTA: "corta" },
			{ SECRETO_PRELOGIN: undefined },
			{ SECRETO_PRELOGIN: env.PIMIENTA },
			{ ENTORNO: "produccion", CARTERO_SECRETO: undefined },
		]) {
			const r = await con(falta as Partial<typeof env>, "GET", "/v1/salud");
			expect(r.status, JSON.stringify(falta)).toBe(503);
		}
		expect((await con({ ENTORNO: "produccion", CARTERO_SECRETO: "el-secreto-del-cartero" }, "GET", "/v1/salud")).status).toBe(200);
	});

	/**
	 * Un espacio pegado al copiar dejaba el interruptor sin efecto, y el síntoma era
	 * **que parecía funcionar**: `202` y el correo al buzón de pruebas. Se vio mirando
	 * el registro del servidor de la otra punta, no aquí.
	 */
	it("el interruptor del correo aguanta los espacios de un copiar y pegar", () => {
		for (const v of ["enviar", " enviar", "enviar\n", "  enviar  "]) {
			expect(modoDeCorreo({ ...env, CORREO: v } as Partial<typeof env> as never), JSON.stringify(v)).toBe("enviar");
		}
		for (const v of [undefined, "", "buzon", "enviarr"]) {
			expect(modoDeCorreo({ ...env, CORREO: v } as Partial<typeof env> as never), JSON.stringify(v)).toBe("buzon");
		}
	});

	/** Y el Worker de pruebas lo dice, para no tener que deducirlo de otro registro. */
	it("la salud del Worker de pruebas dice por dónde manda el correo", async () => {
		const r = await con({ ENTORNO: "pruebas" }, "GET", "/v1/salud");
		expect(((await r.json()) as { correo: string }).correo).toBe("buzon");
		const q = await con({ ENTORNO: "produccion", CARTERO_SECRETO: "el-secreto-del-cartero" }, "GET", "/v1/salud");
		expect((await q.json()) as object).not.toHaveProperty("correo");
	});

	it("un fallo por dentro no cuenta nada en producción", async () => {
		const r = await con({ ENTORNO: "produccion", CARTERO_SECRETO: "el-secreto-del-cartero", JURISDICCION: "us" }, "POST", "/v1/prelogin", { correo: "a@ejemplo.com" });
		expect(r.status).toBe(500);
		expect(Object.keys((await r.json()) as object)).toEqual(["error"]);
	});

	it("los dos Workers de verdad guardan las cuentas en la UE", () => {
		const sinComentarios = env.CONFIGURACION.replace(/^\s*\/\/.*$/gm, "");
		const conf = JSON.parse(sinComentarios);
		expect(conf.vars.JURISDICCION).toBe("eu");
		expect(conf.env.pruebas.vars.JURISDICCION).toBe("eu");
		expect(conf.observability.enabled).toBe(false);
	});

	// **Con el registro abierto, el tope del día tiene que caber en lo que da el
	// correo** (ADR 0041, y ahora la 0045). Cada alta gasta dos correos —el código y
	// la constancia—, así que subir este número sin mirar el correo deja a la gente a
	// medias: cuenta creada y sin poder entrar desde otro equipo, o alta que no llega
	// a su código.
	//
	// Lo que guarda esta prueba **no es el número, es el invariante**: por eso sigue
	// aquí después de cambiar de proveedor, solo con otro presupuesto.
	it("si el registro está abierto, las altas del día caben en lo que da el correo", () => {
		// El relay de Google Workspace, por usuario y 24 h.
		const CORREOS_AL_DIA = 10_000;
		const conf = JSON.parse(env.CONFIGURACION.replace(/^\s*\/\/.*$/gm, ""));
		if (conf.vars.REGISTRO !== "abierto") return;
		const altas = Number(conf.vars.TOPE_ALTAS_DIA);
		expect(Number.isInteger(altas)).toBe(true);
		// La mitad del presupuesto para las altas; la otra mitad, para entradas desde
		// equipos nuevos, recuperaciones, cambios de contraseña e invitaciones.
		expect(altas * 2).toBeLessThanOrEqual(CORREOS_AL_DIA / 2);
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
		expect((await (await con(lista, "GET", "/v1/salud")).json()) as object).toEqual({ registro: "lista", protocolo: 1, correo: "buzon" });
	});
});
