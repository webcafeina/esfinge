import { describe, expect, it } from "vitest";
import { boveda, darDeAlta, pedir } from "./ayuda";

const ID = "0123456789abcdef0123456789abcdef";

describe("la bóveda", () => {
	it("se sube sobre la versión 0, se baja igual y dice su versión", async () => {
		const a = await darDeAlta("lia@ejemplo.com");
		expect((await pedir("GET", "/v1/boveda", { token: a.sesion })).status).toBe(404);

		const doc = boveda(ID);
		const sube = await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: doc, cabeceras: { "If-Match": '"0"' } });
		expect(sube.status).toBe(200);
		expect(await sube.json()).toEqual({ version: 1 });

		const baja = await pedir("GET", "/v1/boveda", { token: a.sesion });
		expect(baja.status).toBe(200);
		expect(baja.headers.get("ETag")).toBe('"1"');
		expect(await baja.text()).toBe(doc);

		const igual = await pedir("GET", "/v1/boveda", { token: a.sesion, cabeceras: { "If-None-Match": '"1"' } });
		expect(igual.status).toBe(304);
	});

	it("sobre una versión vieja no escribe, y dice cuál es la de ahora", async () => {
		const a = await darDeAlta("mar@ejemplo.com");
		await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID), cabeceras: { "If-Match": '"0"' } });
		const vieja = await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID, "x"), cabeceras: { "If-Match": '"0"' } });
		expect(vieja.status).toBe(412);
		expect(((await vieja.json()) as { version: number }).version).toBe(1);
		// Y sin decir sobre qué versión se escribe, tampoco.
		expect((await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID) })).status).toBe(428);
	});

	it("de dos subidas a la vez sobre la misma versión, gana una y la otra lo sabe", async () => {
		const a = await darDeAlta("nora@ejemplo.com");
		await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID), cabeceras: { "If-Match": '"0"' } });
		const resultados = await Promise.all(
			Array.from({ length: 6 }, (_, i) =>
				pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID, String(i)), cabeceras: { "If-Match": '"1"' } }),
			),
		);
		const estados = resultados.map((r) => r.status).sort();
		expect(estados.filter((e) => e === 200)).toHaveLength(1);
		expect(estados.filter((e) => e === 412)).toHaveLength(5);
	});

	it("no guarda lo que no es una bóveda, ni la bóveda de otro", async () => {
		const a = await darDeAlta("olga@ejemplo.com");
		expect((await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: "hola", cabeceras: { "If-Match": '"0"' } })).status).toBe(400);
		expect(
			(await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: JSON.stringify({ esfinge: "otra cosa" }), cabeceras: { "If-Match": '"0"' } })).status,
		).toBe(400);
		await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID), cabeceras: { "If-Match": '"0"' } });
		const otra = await pedir("PUT", "/v1/boveda", {
			token: a.sesion, crudo: boveda("ffffffffffffffffffffffffffffffff"), cabeceras: { "If-Match": '"1"' },
		});
		expect(otra.status).toBe(409);
	});

	it("una bóveda grande se parte en trozos y vuelve entera", async () => {
		const a = await darDeAlta("pau@ejemplo.com");
		const doc = boveda(ID, "y".repeat(3 * 1024 * 1024 + 17)); // más de tres trozos
		expect((await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: doc, cabeceras: { "If-Match": '"0"' } })).status).toBe(200);
		expect(await (await pedir("GET", "/v1/boveda", { token: a.sesion })).text()).toBe(doc);
	});

	it("una demasiado grande no entra", async () => {
		const a = await darDeAlta("quim@ejemplo.com");
		const doc = boveda(ID, "z".repeat(8 * 1024 * 1024));
		expect((await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: doc, cabeceras: { "If-Match": '"0"' } })).status).toBe(413);
	});

	it("guarda las diez últimas versiones y se puede bajar cualquiera de ellas", async () => {
		const a = await darDeAlta("rut@ejemplo.com");
		for (let v = 0; v < 15; v++) {
			const r = await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID, `v${v + 1}`), cabeceras: { "If-Match": `"${v}"` } });
			expect(r.status).toBe(200);
		}
		const lista = (await (await pedir("GET", "/v1/boveda/versiones", { token: a.sesion })).json()) as { version: number }[];
		expect(lista.map((v) => v.version)).toEqual([15, 14, 13, 12, 11, 10, 9, 8, 7, 6]);
		const la7 = await pedir("GET", "/v1/boveda/versiones/7", { token: a.sesion });
		expect(await la7.text()).toBe(boveda(ID, "v7"));
		expect((await pedir("GET", "/v1/boveda/versiones/2", { token: a.sesion })).status).toBe(404);
	});
});
