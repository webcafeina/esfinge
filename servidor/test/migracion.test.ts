import { env, runInDurableObject } from "cloudflare:test";
import { describe, expect, it } from "vitest";
import { buzon, darDeAlta, nuevaIP, pedir } from "./ayuda";

/**
 * Lo que le pasa a una cuenta **que ya existía** cuando el servidor cambia de forma.
 *
 * `CREATE TABLE IF NOT EXISTS` no añade columnas: las cuentas creadas antes tienen
 * las tablas de antes, y escribir en una columna que no está revienta la petición
 * —y con ella, entrar desde un equipo nuevo—. Aquí se fabrica a propósito una
 * cuenta con la tabla vieja y se comprueba que el objeto se pone al día solo.
 */
describe("cuentas de antes", () => {
	it("una con la tabla de códigos vieja sigue pudiendo pedir uno", async () => {
		const a = await darDeAlta("antigua@ejemplo.com");
		const ns = env.CUENTAS;
		const stub = ns.get(ns.idFromName(a.cuenta));

		// Se le deja la tabla como estaba antes de separar el cupo por propósito, y se
		// reinicia el objeto: **el arreglo corre al arrancar**, que es lo que pasa al
		// desplegar una versión nueva.
		await runInDurableObject(stub, (_instancia, estado) => {
			estado.storage.sql.exec("DROP TABLE retos_creados");
			estado.storage.sql.exec("CREATE TABLE retos_creados (momento INTEGER NOT NULL)");
		});
		await runInDurableObject(stub, (_instancia, estado) => {
			try {
				estado.abort("con la tabla vieja, como una cuenta de antes");
			} catch {
				/* abortar mata este objeto: el siguiente que lo use lo arranca otra vez */
			}
		}).catch(() => {});

		// Y una petición cualquiera la pone al día: entrar desde un equipo nuevo pide
		// su código y el correo llega.
		const antes = (await buzon(a.correo)).length;
		const r = await pedir("POST", "/v1/sesion", {
			ip: nuevaIP(),
			cuerpo: { correo: a.correo, claveDeAcceso: a.claveDeAcceso, dispositivo: "Uno nuevo" },
		});
		expect(r.status).toBe(202);
		expect((await buzon(a.correo)).length).toBe(antes + 1);

		// Y la columna nueva está, con su valor por defecto para lo que hubiera.
		// Con un enlace nuevo: el de antes se rompió al abortar, como en la vida real.
		const columnas = await runInDurableObject(ns.get(ns.idFromName(a.cuenta)), (_i, estado) =>
			estado.storage.sql
				.exec("SELECT name FROM pragma_table_info('retos_creados')")
				.toArray()
				.map((c) => (c as { name: string }).name),
		);
		expect(columnas).toContain("proposito");
	});
});
