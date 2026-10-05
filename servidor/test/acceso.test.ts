import { describe, expect, it } from "vitest";
import { boveda, darDeAlta, pedir } from "./ayuda";

// Dar acceso a una bóveda de proyecto (ADR 0052), por el lado del servidor.
//
// Lo que aquí se comprueba no es el cifrado —eso pasa entero en los clientes y el
// servidor no puede leer nada— sino **las tres cosas que solo puede hacer cumplir el
// servidor**: que quien no tiene acceso no entre, que quien solo puede ver no
// escriba, y que nadie se pueda hacer pasar por otra cuenta.

const REF = "a1b2c3d4e5f60718";
const ID = "0123456789abcdef0123456789abcdef";
const TITULAR = "1111222233334444";

/** Una cuenta con una bóveda de proyecto dentro, lista para compartir. */
async function conUnProyecto(correo: string) {
	const a = await darDeAlta(correo);
	const sube = await pedir("PUT", `/v1/bovedas/${REF}`, {
		token: a.sesion,
		crudo: boveda(ID),
		cabeceras: { "If-Match": '"0"' },
	});
	expect(sube.status).toBe(200);
	return a;
}

describe("dar acceso a una bóveda de proyecto", () => {
	it("quien lo recibe la baja y la sube, y quien no lo tiene no la ve", async () => {
		const dueno = await conUnProyecto("duena@ejemplo.com");
		const ana = await darDeAlta("ana@ejemplo.com");
		const otro = await darDeAlta("otro@ejemplo.com");

		// **Sin acceso, 403 y no 404**: que la bóveda existe ya lo sabe quien pregunta
		// por ella con su referencia, y distinguirlo no delata nada que no se sepa.
		const antes = await pedir("GET", `/v1/compartidas/${dueno.cuenta}/${REF}`, { token: ana.sesion });
		expect(antes.status).toBe(403);

		const da = await pedir("POST", `/v1/bovedas/${REF}/miembros`, {
			token: dueno.sesion,
			cuerpo: { para: "ana@ejemplo.com", permiso: "editar", titular: TITULAR },
		});
		expect(da.status).toBe(202);

		const baja = await pedir("GET", `/v1/compartidas/${dueno.cuenta}/${REF}`, { token: ana.sesion });
		expect(baja.status).toBe(200);
		expect(baja.headers.get("ETag")).toBe('"1"');
		expect(await baja.text()).toBe(boveda(ID));

		// Y escribe, que es lo que la hace viva: lo que sube lo ve el dueño.
		const sube = await pedir("PUT", `/v1/compartidas/${dueno.cuenta}/${REF}`, {
			token: ana.sesion,
			crudo: boveda(ID, "lo de Ana"),
			cabeceras: { "If-Match": '"1"' },
		});
		expect(sube.status).toBe(200);
		const delDueno = await pedir("GET", `/v1/bovedas/${REF}`, { token: dueno.sesion });
		expect(await delDueno.text()).toBe(boveda(ID, "lo de Ana"));

		// Un tercero sigue fuera: el acceso es de quien se le dio, no de quien sepa la
		// dirección de la bóveda.
		const tercero = await pedir("GET", `/v1/compartidas/${dueno.cuenta}/${REF}`, { token: otro.sesion });
		expect(tercero.status).toBe(403);
	});

	it("quien solo puede ver, baja pero no sube", async () => {
		const dueno = await conUnProyecto("duena2@ejemplo.com");
		const ana = await darDeAlta("ana2@ejemplo.com");
		await pedir("POST", `/v1/bovedas/${REF}/miembros`, {
			token: dueno.sesion,
			cuerpo: { para: "ana2@ejemplo.com", permiso: "ver", titular: TITULAR },
		});

		expect((await pedir("GET", `/v1/compartidas/${dueno.cuenta}/${REF}`, { token: ana.sesion })).status).toBe(200);

		// **Esto es lo único que impide escribir**: Ana tiene la clave de la bóveda, así
		// que puede fabricar un documento válido. Lo que no puede es dejarlo aquí.
		const sube = await pedir("PUT", `/v1/compartidas/${dueno.cuenta}/${REF}`, {
			token: ana.sesion,
			crudo: boveda(ID, "esto no debería entrar"),
			cabeceras: { "If-Match": '"1"' },
		});
		expect(sube.status).toBe(403);
		// Y no ha entrado: lo del dueño sigue como estaba.
		const delDueno = await pedir("GET", `/v1/bovedas/${REF}`, { token: dueno.sesion });
		expect(await delDueno.text()).toBe(boveda(ID));
	});

	it("un testigo con la cuenta de otra persona y el secreto inventado no entra", async () => {
		const dueno = await conUnProyecto("duena3@ejemplo.com");
		const ana = await darDeAlta("ana3@ejemplo.com");
		await pedir("POST", `/v1/bovedas/${REF}/miembros`, {
			token: dueno.sesion,
			cuerpo: { para: "ana3@ejemplo.com", permiso: "editar", titular: TITULAR },
		});

		// **El testigo dice de qué cuenta es, y eso no lo comprueba quien lo lee.** Lo
		// comprueba el objeto de esa cuenta, buscando la huella del secreto. Mientras
		// cada petición acaba en el objeto que dice el testigo, basta; en una petición
		// sobre la bóveda de otro, no: si el Worker sacara el «quién» del testigo sin
		// preguntar, esto entraría como Ana.
		const inventado = `s1.${ana.cuenta}.${"a".repeat(43)}`;
		const r = await pedir("GET", `/v1/compartidas/${dueno.cuenta}/${REF}`, { token: inventado });
		expect(r.status).toBe(401);
	});

	it("quitar el acceso cierra la puerta en la siguiente petición", async () => {
		const dueno = await conUnProyecto("duena4@ejemplo.com");
		const ana = await darDeAlta("ana4@ejemplo.com");
		await pedir("POST", `/v1/bovedas/${REF}/miembros`, {
			token: dueno.sesion,
			cuerpo: { para: "ana4@ejemplo.com", permiso: "editar", titular: TITULAR },
		});
		expect((await pedir("GET", `/v1/compartidas/${dueno.cuenta}/${REF}`, { token: ana.sesion })).status).toBe(200);

		const quita = await pedir("DELETE", `/v1/bovedas/${REF}/miembros/${TITULAR}`, { token: dueno.sesion });
		expect(quita.status).toBe(200);

		expect((await pedir("GET", `/v1/compartidas/${dueno.cuenta}/${REF}`, { token: ana.sesion })).status).toBe(403);
	});

	it("la lista de miembros la da el servidor por titular, nunca por cuenta", async () => {
		const dueno = await conUnProyecto("duena5@ejemplo.com");
		await darDeAlta("ana5@ejemplo.com");
		await pedir("POST", `/v1/bovedas/${REF}/miembros`, {
			token: dueno.sesion,
			cuerpo: { para: "ana5@ejemplo.com", permiso: "ver", titular: TITULAR },
		});

		const r = await pedir("GET", `/v1/bovedas/${REF}/miembros`, { token: dueno.sesion });
		expect(r.status).toBe(200);
		const d = (await r.json()) as { miembros: { titular: string; permiso: string }[] };
		expect(d.miembros).toHaveLength(1);
		expect(d.miembros[0].titular).toBe(TITULAR);
		expect(d.miembros[0].permiso).toBe("ver");
		// **Y no sale ninguna cuenta.** A quién corresponde cada titular lo sabe la
		// bóveda, que es donde están los correos; el servidor no reparte eso.
		expect(JSON.stringify(d)).not.toContain("cuenta");
	});

	it("dar acceso a quien no tiene cuenta contesta lo mismo que a quien la tiene", async () => {
		const dueno = await conUnProyecto("duena6@ejemplo.com");
		// **Media protección contra la enumeración de correos**, la misma regla que al
		// mandar un sobre: si contestara distinto, esto diría quién usa Esfinge.
		const r = await pedir("POST", `/v1/bovedas/${REF}/miembros`, {
			token: dueno.sesion,
			cuerpo: { para: "nadie@ejemplo.com", permiso: "ver", titular: TITULAR },
		});
		expect(r.status).toBe(202);
	});

	it("borrar la bóveda se lleva a quien tenía acceso", async () => {
		const dueno = await conUnProyecto("duena7@ejemplo.com");
		await darDeAlta("ana7@ejemplo.com");
		await pedir("POST", `/v1/bovedas/${REF}/miembros`, {
			token: dueno.sesion,
			cuerpo: { para: "ana7@ejemplo.com", permiso: "editar", titular: TITULAR },
		});
		expect((await pedir("DELETE", `/v1/bovedas/${REF}`, { token: dueno.sesion })).status).toBe(200);

		const r = await pedir("GET", `/v1/bovedas/${REF}/miembros`, { token: dueno.sesion });
		expect(((await r.json()) as { miembros: unknown[] }).miembros).toHaveLength(0);
	});
});
