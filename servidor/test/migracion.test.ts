import { env, runInDurableObject } from "cloudflare:test";
import { describe, expect, it } from "vitest";
import { boveda, buzon, darDeAlta, nuevaIP, pedir } from "./ayuda";

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

/**
 * **La migración a varias bóvedas, sobre una cuenta que ya tenía la suya dentro.**
 *
 * Es el paso que más miedo da de todo el proyecto: `versiones` y `trozos` pasan a
 * llevar la referencia de la bóveda en su clave primaria, **SQLite no deja añadir
 * una columna a una clave primaria**, y por tanto hay que copiar las tablas. Eso
 * corre una vez y sobre la bóveda de alguien.
 *
 * Lo que se comprueba aquí es lo único que importa: que **los bytes que había
 * siguen estando**, que el ETag no se mueve —un cliente al día no se baja nada— y
 * que correrlo dos veces no hace nada la segunda.
 */
describe("la migración a varias bóvedas", () => {
	it("conserva la bóveda que ya estaba, su versión y sus bytes", async () => {
		const a = await darDeAlta("conboveda@ejemplo.com");
		const ID = "aaaabbbbccccdddd";
		await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID, "uno"), cabeceras: { "If-Match": '"0"' } });
		await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID, "dos"), cabeceras: { "If-Match": '"1"' } });
		const antes = await pedir("GET", "/v1/boveda", { token: a.sesion });
		const bytesDeAntes = await antes.text();
		expect(antes.headers.get("ETag")).toBe('"2"');

		const ns = env.CUENTAS;
		const stub = ns.get(ns.idFromName(a.cuenta));

		// Se le dejan las tablas **como eran antes**: sin `ref`, con la versión como
		// clave primaria, y con los datos dentro. Es una cuenta de la 2.37.
		await runInDurableObject(stub, (_instancia, estado) => {
			const sql = estado.storage.sql;
			const versiones = sql.exec("SELECT version, fecha, tamano, huella FROM versiones").toArray();
			const trozos = sql.exec("SELECT version, n, datos FROM trozos").toArray();
			sql.exec("DROP TABLE versiones");
			sql.exec("DROP TABLE trozos");
			sql.exec(
				"CREATE TABLE versiones (version INTEGER PRIMARY KEY, fecha INTEGER NOT NULL, tamano INTEGER NOT NULL, huella TEXT NOT NULL)",
			);
			sql.exec(
				"CREATE TABLE trozos (version INTEGER NOT NULL, n INTEGER NOT NULL, datos BLOB NOT NULL, PRIMARY KEY (version, n))",
			);
			for (const v of versiones as { version: number; fecha: number; tamano: number; huella: string }[]) {
				sql.exec("INSERT INTO versiones (version, fecha, tamano, huella) VALUES (?, ?, ?, ?)", v.version, v.fecha, v.tamano, v.huella);
			}
			for (const t of trozos as { version: number; n: number; datos: ArrayBuffer }[]) {
				sql.exec("INSERT INTO trozos (version, n, datos) VALUES (?, ?, ?)", t.version, t.n, t.datos);
			}
			sql.exec("DELETE FROM ajustes WHERE clave = 'esquema'");
		});
		await runInDurableObject(stub, (_i, estado) => {
			try {
				estado.abort("con las tablas viejas, como una cuenta de antes");
			} catch {
				/* abortar mata este objeto: el siguiente que lo use lo arranca otra vez */
			}
		}).catch(() => {});

		// Y ahora una petición cualquiera la pone al día. **Los mismos bytes y el
		// mismo ETag**: para quien ya estaba sincronizado, aquí no ha pasado nada.
		const despues = await pedir("GET", "/v1/boveda", { token: a.sesion });
		expect(despues.status).toBe(200);
		expect(despues.headers.get("ETag")).toBe('"2"');
		expect(await despues.text()).toBe(bytesDeAntes);

		// Las dos versiones siguen ahí, con su historial.
		const lista = (await (await pedir("GET", "/v1/boveda/versiones", { token: a.sesion })).json()) as {
			version: number;
		}[];
		expect(lista.map((v) => v.version)).toEqual([2, 1]);

		// Y la tabla tiene la columna nueva, con la bóveda personal en `ref = ''`.
		const refs = await runInDurableObject(ns.get(ns.idFromName(a.cuenta)), (_i, estado) =>
			estado.storage.sql.exec("SELECT DISTINCT ref FROM versiones").toArray().map((f) => (f as { ref: string }).ref),
		);
		expect(refs).toEqual([""]);
	});

	it("y correrla otra vez no hace nada", async () => {
		const a = await darDeAlta("dosveces@ejemplo.com");
		const ID = "aaaabbbbccccdddd";
		await pedir("PUT", "/v1/boveda", { token: a.sesion, crudo: boveda(ID, "uno"), cabeceras: { "If-Match": '"0"' } });

		const ns = env.CUENTAS;
		// Se le quita la marca **sin tocar las tablas**, que es el caso de una
		// migración que se quedó a medias justo antes de escribirla: lo que no puede
		// pasar es que vuelva a copiar y duplique.
		await runInDurableObject(ns.get(ns.idFromName(a.cuenta)), (_i, estado) => {
			estado.storage.sql.exec("DELETE FROM ajustes WHERE clave = 'esquema'");
		});
		await runInDurableObject(ns.get(ns.idFromName(a.cuenta)), (_i, estado) => {
			try {
				estado.abort("para que arranque otra vez");
			} catch {
				/* igual que arriba */
			}
		}).catch(() => {});

		const r = await pedir("GET", "/v1/boveda", { token: a.sesion });
		expect(r.status).toBe(200);
		expect(r.headers.get("ETag")).toBe('"1"');
		const cuantas = await runInDurableObject(ns.get(ns.idFromName(a.cuenta)), (_i, estado) =>
			estado.storage.sql.exec("SELECT COUNT(*) AS n FROM versiones").one(),
		);
		expect((cuantas as { n: number }).n).toBe(1);
	});
});
