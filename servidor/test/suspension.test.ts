import { env } from "cloudflare:test";
import { describe, expect, it } from "vitest";
import { boveda, darDeAlta, nuevaIP, pedir } from "./ayuda";

/**
 * Cerrar una cuenta, que es lo que prometen las condiciones de uso (ADR 0042).
 *
 * La marca vive en D1 y se pone a mano con `wrangler d1 execute`, así que aquí se
 * escribe igual: un `UPDATE`. **Lo que más importa de estas pruebas es lo que
 * sigue funcionando**: suspendida, la cuenta no entra ni sube, pero baja y exporta,
 * porque las condiciones prometen un plazo para llevarse los datos.
 */
describe("una cuenta suspendida", () => {
	const suspender = (cuenta: string) =>
		env.BD.prepare("UPDATE cuentas SET suspendida = 1 WHERE cuenta = ?").bind(cuenta).run();

	it("no deja entrar, y lo dice con a quién escribir", async () => {
		const a = await darDeAlta("luis@ejemplo.com", true);
		await suspender(a.cuenta);

		const r = await pedir("POST", "/v1/sesion", {
			ip: nuevaIP(),
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, confianza: a.confianza },
		});
		expect(r.status).toBe(403);
		expect(((await r.json()) as { error: string }).error).toContain("info@webcafeina.com");
	});

	it("no deja subir, pero sí bajar y exportar", async () => {
		const a = await darDeAlta("marta@ejemplo.com");
		const subida = await pedir("PUT", "/v1/boveda", {
			token: a.sesion,
			crudo: boveda("0123456789abcdef0123456789abcdef"),
			cabeceras: { "If-Match": '"0"', "Content-Type": "application/json" },
		});
		expect(subida.status).toBe(200);

		await suspender(a.cuenta);

		const otra = await pedir("PUT", "/v1/boveda", {
			token: a.sesion,
			crudo: boveda("0123456789abcdef0123456789abcdef", "x"),
			cabeceras: { "If-Match": '"1"', "Content-Type": "application/json" },
		});
		expect(otra.status).toBe(403);

		// Y lo que tiene que seguir yendo, porque es la promesa del plazo:
		expect((await pedir("GET", "/v1/boveda", { token: a.sesion })).status).toBe(200);
		expect((await pedir("GET", "/v1/cuenta/exportacion", { token: a.sesion })).status).toBe(200);
	});

	it("sin suspender, todo sigue igual", async () => {
		const a = await darDeAlta("nuria@ejemplo.com", true);
		const r = await pedir("POST", "/v1/sesion", {
			ip: nuevaIP(),
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, confianza: a.confianza },
		});
		expect(r.status).toBe(200);
	});
});
