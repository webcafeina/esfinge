import { describe, expect, it } from "vitest";
import { boveda, buzon, darDeAlta, pedir, ultimoCodigo } from "./ayuda";

const ID = "0123456789abcdef0123456789abcdef";

describe("los equipos", () => {
	it("se listan, se marca el propio y olvidar uno le quita la sesión", async () => {
		const a = await darDeAlta("abel@ejemplo.com");
		const r = await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, dispositivo: "Chrome en el MacBook" } });
		const { reto } = (await r.json()) as { reto: string };
		const hecho = await pedir("POST", "/v1/sesion/codigo", { ip: a.ip, cuerpo: { reto, codigo: await ultimoCodigo(a.correo) } });
		const otra = (await hecho.json()) as { sesion: string; dispositivo: string };
		// Entrar desde un equipo nuevo avisa en el buzón.
		expect((await buzon(a.correo))[0]?.asunto).toBe("Un equipo nuevo ha entrado en tu cuenta de Esfinge");

		const lista = (await (await pedir("GET", "/v1/dispositivos", { token: a.sesion })).json()) as { id: string; nombre: string; actual: boolean }[];
		expect(lista.map((e) => e.nombre).sort()).toEqual(["Chrome en el MacBook", "Portátil"]);
		expect(lista.find((e) => e.actual)?.id).toBe(a.dispositivo);

		expect((await pedir("DELETE", `/v1/dispositivos/${otra.dispositivo}`, { token: a.sesion })).status).toBe(200);
		expect((await pedir("GET", "/v1/boveda", { token: otra.sesion })).status).toBe(401);
	});
});

describe("borrar la cuenta", () => {
	it("pide contraseña y código, y no deja nada: ni bóveda, ni correo, ni sesiones", async () => {
		const a = await darDeAlta("bruno@ejemplo.com");
		await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID), cabeceras: { "If-Match": '"0"' } });
		const pide = await pedir("POST", "/v1/cuenta/borrado", { token: a.sesion });
		expect(pide.status).toBe(202);
		const { reto } = (await pide.json()) as { reto: string };
		const codigo = await ultimoCodigo(a.correo);

		// Sin la contraseña, no.
		const sin = await pedir("DELETE", "/v1/cuenta", { token: a.sesion, cuerpo: { reto, codigo, claveDeAcceso: "AAAA" } });
		expect(sin.status).toBe(401);
		const r = await pedir("DELETE", "/v1/cuenta", { token: a.sesion, cuerpo: { reto, codigo, claveDeAcceso: a.claveDeAcceso } });
		expect(r.status).toBe(200);

		expect((await pedir("GET", "/v1/boveda", { token: a.sesion })).status).toBe(401);
		const pre = (await (await pedir("POST", "/v1/prelogin", { cuerpo: { correo: a.correo } })).json()) as { sal: string };
		expect(pre.sal).not.toBe(a.sal);
		expect((await pedir("POST", "/v1/sesion", { ip: a.ip, cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso } })).status).toBe(401);
		expect((await buzon(a.correo))[0]?.asunto).toBe("Tu cuenta de Esfinge se ha borrado");

		// Y el correo queda libre para una cuenta nueva.
		const otra = await darDeAlta(a.correo);
		expect(otra.cuenta).not.toBe(a.cuenta);
	});
});

describe("exportar", () => {
	it("entrega lo que hay de la cuenta, con la bóveda tal cual: cifrada", async () => {
		const a = await darDeAlta("clara@ejemplo.com");
		const doc = boveda(ID);
		await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: doc, cabeceras: { "If-Match": '"0"' } });
		const r = await pedir("GET", "/v1/cuenta/exportacion", { token: a.sesion });
		expect(r.status).toBe(200);
		expect(r.headers.get("Content-Disposition")).toContain("attachment");
		const d = (await r.json()) as Record<string, any>;
		expect(d.cuenta.correo).toBe(a.correo);
		expect(d.boveda).toBe(doc);
		expect(d.equipos).toHaveLength(1);
		expect(d.eventos.map((e: { tipo: string }) => e.tipo)).toContain("alta");
		// Nada de lo que permitiría suplantar: ni verificadores, ni sal, ni testigos.
		const texto = JSON.stringify(d);
		expect(texto).not.toContain(a.sal);
		expect(texto).not.toContain(a.sesion.split(".")[2]);
	});
});
