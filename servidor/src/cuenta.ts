// Una cuenta: un Durable Object con su propia base SQLite, en la UE.
//
// **Todo lo de una cuenta vive aquí y pasa por aquí de uno en uno.** Eso es lo que
// hace que «¿sigue siendo la versión 17? pues escribo la 18» no pueda partirse en
// dos por una segunda subida que llegue en medio, y que cambiar la contraseña
// —bóveda nueva, verificador nuevo y sesiones revocadas— sea una sola cosa.
//
// Y es también donde viven **los frenos de la cuenta**, que duran lo que dura la
// cuenta. La lección del canal con el navegador (CLAUDE.md): un contador que vive
// en algo más corto que lo que quiere frenar no frena nada.
//
// La bóveda se guarda **aquí mismo, en trozos**, y no en R2: una bóveda real pesa
// cientos de kilobytes, y en la misma base la escritura es atómica con todo lo
// demás. Con R2 al lado, esperar a la subida abre un hueco entre comprobar la
// versión y escribirla por el que cabe otra petición.

import { DurableObject } from "cloudflare:workers";
import { aBase64url, aHex, azar, codigoDeSeisCifras, deBase64url, sha256 } from "./cripto";
import { algunaPimientaDa, conLaPimienta, igualesHex, partir, sellar, versionActual } from "./pimienta";
import {
	type Argon2,
	DIA,
	type Env,
	HORA,
	MINUTO,
	type Resultado,
	TAMANO_MAXIMO,
	bien,
	mal,
	nombreDeEquipo,
} from "./protocolo";

const TROZO = 1024 * 1024; // una fila de SQLite en un Durable Object admite hasta 2 MB

/**
 * La clave de un ajuste que es **por bóveda** (ADR 0050).
 *
 * La bóveda personal usa la clave de siempre, sin prefijo —`version`, `idBoveda`,
 * `recuperacion`—, y los proyectos llevan la suya detrás: `version:a1b2…`. Así una
 * cuenta que ya existe no necesita migrar ningún ajuste, y **una versión anterior
 * del servidor seguiría leyendo la bóveda personal** si hubiera que volver atrás:
 * las claves que no entiende las ignora.
 */
function claveDe(que: string, ref: string): string {
	return ref === "" ? que : `${que}:${ref}`;
}

/** Una referencia de bóveda de proyecto: hex de 8 bytes. Vacío es la personal. */
export function refValida(ref: string): boolean {
	return /^[0-9a-f]{16}$/.test(ref);
}

/**
 * Cuántas bóvedas de proyecto admite una cuenta.
 *
 * Es una decisión y no un límite técnico: mejor un número escrito con su mensaje
 * que descubrir el de SQLite el día que alguien pase de él. Cabe subirlo.
 */
const BOVEDAS_POR_CUENTA = 50;

/**
 * Y **de un proyecto se guardan menos versiones que de la personal**: el
 * almacenamiento del Durable Object crece con el número de bóvedas, y lo que las
 * versiones compran —deshacer una fusión mala— pesa menos en una bóveda que se
 * abre de vez en cuando.
 */
const VERSIONES_DE_PROYECTO = 3;

const SESION_DESLIZANTE = 30 * DIA;
const SESION_MAXIMA = 90 * DIA;
const SESION_RESTRINGIDA = 15 * MINUTO;
const CONFIANZA = 90 * DIA;
const RETO = 10 * MINUTO;
const INTENTOS_POR_RETO = 5;
// **Un cupo por propósito, y no uno para todo** (revisión del 2026-09-23). Con un
// solo cupo, cinco peticiones de recuperación por hora —que no piden
// autenticación, basta saber el correo— dejaban a esa persona sin poder entrar
// desde un equipo nuevo **y** sin poder recuperar la cuenta, y además le llegaban
// cinco correos. Ahora cada cosa gasta del suyo, y hay un tope al día para que
// nadie use la cuenta ajena como bombardeo de correo.
const RETOS_POR_HORA = 5;
const RETOS_POR_DIA = 20;
const FALLOS_PARA_FRENAR = 10;
const VENTANA_DE_FALLOS = 15 * MINUTO;
const SUBIDAS_POR_HORA = 60;
const VERSIONES_RECIENTES = 10;
const DIAS_CON_VERSION = 30;
const EVENTOS_GUARDADOS = 200;
/** Un sobre lleva una entrada cifrada; 64 KiB son de sobra y frenan el abuso. */
const TAMANO_DE_ENVIO = 64 * 1024;
const ENVIOS_EN_BUZON = 50;

/** Las llaves públicas de una cuenta, en base64url. */
export type Llaves = { suite: string; cifrado: string; firma: string };

export function llavesValidas(l: unknown): l is Llaves {
	if (!l || typeof l !== "object") return false;
	const { suite, cifrado, firma } = l as Record<string, unknown>;
	return (
		typeof suite === "string" &&
		suite.length > 0 &&
		suite.length <= 80 &&
		deBase64url(cifrado, 32) !== null &&
		deBase64url(firma, 32) !== null
	);
}

export const SESION_CADUCADA = "La sesión ha caducado; vuelve a entrar.";
const NO_VALE = "Correo o contraseña no válidos.";
const CODIGO_MALO = "El código no es correcto o ha caducado. Pide otro.";

type Proposito = "entrar" | "recuperar" | "recuperar-fin" | "borrar";

// Un `type` y no una `interface`: las filas de SQL piden un tipo que admita
// cualquier clave, y eso solo lo cumple un alias.
type FilaDeEquipo = { id: string; nombre: string; creado: number; visto: number };
export type Equipo = FilaDeEquipo & { actual?: boolean };

export interface DatosDeAlta {
	cuenta: string;
	correo: string;
	sal: string;
	argon2: Argon2;
	claveDeAcceso: string;
	posesion: string;
	llaves?: unknown;
	dispositivo: string;
	confiar: boolean;
}

interface Sesion {
	dispositivo: string;
	restringida: boolean;
}

export class Cuenta extends DurableObject<Env> {
	private sql: SqlStorage;

	constructor(ctx: DurableObjectState, env: Env) {
		super(ctx, env);
		this.sql = ctx.storage.sql;
		ctx.blockConcurrencyWhile(async () => this.migrar());
	}

	private migrar() {
		this.sql.exec(`
			CREATE TABLE IF NOT EXISTS ajustes (clave TEXT PRIMARY KEY, valor TEXT NOT NULL);
			CREATE TABLE IF NOT EXISTS versiones (ref TEXT NOT NULL DEFAULT '', version INTEGER NOT NULL, fecha INTEGER NOT NULL, tamano INTEGER NOT NULL, huella TEXT NOT NULL, PRIMARY KEY (ref, version));
			CREATE TABLE IF NOT EXISTS trozos (ref TEXT NOT NULL DEFAULT '', version INTEGER NOT NULL, n INTEGER NOT NULL, datos BLOB NOT NULL, PRIMARY KEY (ref, version, n));
			CREATE TABLE IF NOT EXISTS sesiones (huella TEXT PRIMARY KEY, dispositivo TEXT NOT NULL, creada INTEGER NOT NULL, vista INTEGER NOT NULL, restringida INTEGER NOT NULL DEFAULT 0);
			CREATE TABLE IF NOT EXISTS dispositivos (id TEXT PRIMARY KEY, nombre TEXT NOT NULL, creado INTEGER NOT NULL, visto INTEGER NOT NULL, confianza TEXT, confianza_caduca INTEGER);
			CREATE TABLE IF NOT EXISTS retos (id TEXT PRIMARY KEY, proposito TEXT NOT NULL, codigo TEXT NOT NULL, caduca INTEGER NOT NULL, intentos INTEGER NOT NULL DEFAULT 0, datos TEXT);
			CREATE TABLE IF NOT EXISTS retos_creados (momento INTEGER NOT NULL, proposito TEXT NOT NULL DEFAULT 'entrar');
			CREATE TABLE IF NOT EXISTS fallos (momento INTEGER NOT NULL);
			CREATE TABLE IF NOT EXISTS subidas (momento INTEGER NOT NULL);
			CREATE TABLE IF NOT EXISTS eventos (momento INTEGER NOT NULL, tipo TEXT NOT NULL, detalle TEXT NOT NULL DEFAULT '');
			CREATE TABLE IF NOT EXISTS buzon (id TEXT PRIMARY KEY, sobre TEXT NOT NULL, momento INTEGER NOT NULL);
		`);

		// **Y lo que le falte a una cuenta que ya existía.** `CREATE TABLE IF NOT
		// EXISTS` no añade columnas: las cuentas creadas antes tienen la tabla vieja, y
		// escribir en una columna que no está revienta la petición. Pasó al separar el
		// cupo de códigos por propósito (revisión del 2026-09-23). Se mira la tabla
		// antes de tocarla, que es barato y no depende de atrapar el error bueno.
		const columnas = this.sql
			.exec<{ name: string }>("SELECT name FROM pragma_table_info('retos_creados')")
			.toArray()
			.map((c) => c.name);
		if (!columnas.includes("proposito")) {
			this.sql.exec("ALTER TABLE retos_creados ADD COLUMN proposito TEXT NOT NULL DEFAULT 'entrar'");
		}

		this.aVariasBovedas();
	}

	/**
	 * Esquema 2: las bóvedas de proyecto (ADR 0050).
	 *
	 * `versiones` y `trozos` pasan a llevar **la referencia de la bóveda en su clave
	 * primaria**, o dos bóvedas de la misma cuenta chocarían en el número de versión.
	 * Y **SQLite no deja añadir una columna a una clave primaria**, así que no vale
	 * `ALTER TABLE ADD COLUMN` como con `retos_creados`: hay que copiar las tablas.
	 *
	 * Tres cosas que esto no puede dejar de ser, porque corre **una vez y sobre la
	 * bóveda de alguien**:
	 *
	 *   - **Idempotente**: la marca `esquema` se escribe al final, así que una
	 *     interrupción a medias deja el trabajo por hacer y no a medio hacer.
	 *   - **Dentro de una transacción**, y dentro del `blockConcurrencyWhile` del
	 *     constructor: nadie lee la cuenta mientras tanto.
	 *   - **Se comprueba la copia antes de tirar lo viejo**, contando las filas. Un
	 *     `DROP` detrás de un `INSERT … SELECT` que copió de menos no tiene vuelta.
	 *
	 * La bóveda personal se queda con `ref = ''`, que es lo que hace que todo lo de
	 * antes siga funcionando sin tocar una línea del cliente.
	 */
	private aVariasBovedas() {
		if (this.leerAjuste("esquema") === "2") return;
		// Una cuenta recién creada ya nace con el esquema nuevo: las tablas de arriba
		// lo traen. Lo que hay que mirar es si las que existen son las viejas.
		const columnas = this.sql
			.exec<{ name: string }>("SELECT name FROM pragma_table_info('versiones')")
			.toArray()
			.map((c) => c.name);
		if (columnas.includes("ref")) {
			this.ponerAjuste("esquema", "2");
			return;
		}

		this.ctx.storage.transactionSync(() => {
			const antesV = this.sql.exec<{ n: number }>("SELECT COUNT(*) AS n FROM versiones").one().n;
			const antesT = this.sql.exec<{ n: number }>("SELECT COUNT(*) AS n FROM trozos").one().n;

			this.sql.exec(`
				CREATE TABLE versiones_2 (ref TEXT NOT NULL DEFAULT '', version INTEGER NOT NULL, fecha INTEGER NOT NULL, tamano INTEGER NOT NULL, huella TEXT NOT NULL, PRIMARY KEY (ref, version));
				CREATE TABLE trozos_2 (ref TEXT NOT NULL DEFAULT '', version INTEGER NOT NULL, n INTEGER NOT NULL, datos BLOB NOT NULL, PRIMARY KEY (ref, version, n));
				INSERT INTO versiones_2 (ref, version, fecha, tamano, huella) SELECT '', version, fecha, tamano, huella FROM versiones;
				INSERT INTO trozos_2 (ref, version, n, datos) SELECT '', version, n, datos FROM trozos;
			`);

			const despuesV = this.sql.exec<{ n: number }>("SELECT COUNT(*) AS n FROM versiones_2").one().n;
			const despuesT = this.sql.exec<{ n: number }>("SELECT COUNT(*) AS n FROM trozos_2").one().n;
			if (despuesV !== antesV || despuesT !== antesT) {
				// La transacción se deshace entera y la cuenta se queda como estaba. Es
				// preferible a una cuenta migrada a medias, que no se nota hasta que
				// alguien baja su bóveda y le falta un trozo.
				throw new Error(`Migración incompleta: ${antesV}/${antesT} → ${despuesV}/${despuesT}`);
			}

			this.sql.exec(`
				DROP TABLE versiones;
				DROP TABLE trozos;
				ALTER TABLE versiones_2 RENAME TO versiones;
				ALTER TABLE trozos_2 RENAME TO trozos;
			`);
			this.ponerAjuste("esquema", "2");
		});
	}

	// ---------------------------------------------------------------- ajustes

	private leerAjuste(clave: string): string | null {
		const fila = this.sql.exec<{ valor: string }>("SELECT valor FROM ajustes WHERE clave = ?", clave).toArray()[0];
		return fila ? fila.valor : null;
	}

	private ponerAjuste(clave: string, valor: string) {
		this.sql.exec("INSERT OR REPLACE INTO ajustes (clave, valor) VALUES (?, ?)", clave, valor);
	}

	private existe(): boolean {
		return this.leerAjuste("correo") !== null;
	}

	/**
	 * La versión de una bóveda. **Vacío es la personal**, y entonces la clave es
	 * `version` a secas: la de siempre, sin prefijo, porque es lo que ya está escrito
	 * en todas las cuentas que existen y lo que leería una versión anterior del
	 * servidor si hubiera que volver atrás.
	 */
	private version(ref = ""): number {
		return Number(this.leerAjuste(claveDe("version", ref)) ?? "0");
	}

	private apuntar(tipo: string, detalle = "") {
		this.sql.exec("INSERT INTO eventos (momento, tipo, detalle) VALUES (?, ?, ?)", Date.now(), tipo, detalle);
		this.sql.exec(
			"DELETE FROM eventos WHERE rowid NOT IN (SELECT rowid FROM eventos ORDER BY momento DESC, rowid DESC LIMIT ?)",
			EVENTOS_GUARDADOS,
		);
	}

	/** El verificador de una versión de la pimienta, o nulo si esa versión ya no está. */
	private async verificador(proposito: string, clave: Uint8Array, version: number): Promise<string | null> {
		return conLaPimienta(this.env, version, `${proposito}|${aBase64url(clave)}`);
	}

	/**
	 * **Tres respuestas y no dos.** `imposible` es una cuenta que se quedó dos
	 * generaciones de pimienta por detrás: no es que la contraseña esté mal, es que no
	 * se puede comprobar, y decirle a alguien que su contraseña falla cuando no falla
	 * es el peor mensaje que se le puede dar (ver `pimienta.ts`).
	 */
	private async coincide(proposito: "acceso" | "posesion", clave: string): Promise<"si" | "no" | "imposible"> {
		const b = deBase64url(clave, 32);
		const guardado = this.leerAjuste(proposito);
		const { version, resumen } = partir(guardado ?? "");
		// Se calcula el HMAC aunque falte algo, para que un caso no tarde menos que el otro.
		const calculado = await this.verificador(proposito, b ?? new Uint8Array(32), version);
		if (!b || !guardado) return "no";
		if (calculado === null) return "imposible";
		if (!igualesHex(calculado, resumen)) return "no";
		// Cuadra: si venía de una pimienta vieja, se reescribe con la de ahora. Aquí y
		// no en otro sitio, porque **éste es el único momento en que se tiene la clave**.
		await this.repimentar(proposito, b, version);
		return "si";
	}

	/**
	 * Reescribe un verificador con la pimienta de ahora, y apunta en D1 por dónde va
	 * esta cuenta.
	 *
	 * El orden importa: **primero el objeto y después D1**. Si D1 falla, lo que queda
	 * apuntado es una versión más vieja de la que hay, y ése es el lado seguro del
	 * error — se espera de más antes de borrar la pimienta anterior, en vez de borrarla
	 * creyendo que no queda nadie.
	 */
	private async repimentar(proposito: "acceso" | "posesion", clave: Uint8Array, version: number) {
		const actual = versionActual(this.env);
		if (version === actual) return;
		const nuevo = await this.verificador(proposito, clave, actual);
		if (nuevo === null) return;
		this.ponerAjuste(proposito, sellar(actual, nuevo));
		await this.apuntarLaPimientaEnD1();
	}

	/** La menor de las dos versiones de esta cuenta, a la tabla que se puede contar. */
	private async apuntarLaPimientaEnD1() {
		const cuenta = this.leerAjuste("cuenta");
		if (!cuenta) return;
		const menor = Math.min(
			...(["acceso", "posesion"] as const).map((k) => partir(this.leerAjuste(k) ?? "").version),
		);
		try {
			await this.env.BD.prepare("UPDATE cuentas SET pimienta = ? WHERE cuenta = ?").bind(menor, cuenta).run();
		} catch (e) {
			// **No se traga en silencio, pero tampoco tumba la entrada**: esto es
			// contabilidad para saber cuándo se puede borrar la pimienta vieja, no parte
			// de abrir la cuenta. Quedarse sin apuntarlo retrasa esa decisión; hacer
			// fallar la entrada por ello sería mucho peor.
			console.error(JSON.stringify({ nivel: "error", msg: "no se pudo apuntar la pimienta en D1", porque: `${e}` }));
		}
	}

	// ---------------------------------------------------------------- alta

	/** Lo que la pre-entrada necesita: la sal y el coste. Null si aquí no hay cuenta. */
	async prelogin(): Promise<{ sal: string; argon2: Argon2 } | null> {
		if (!this.existe()) return null;
		return { sal: this.leerAjuste("sal")!, argon2: JSON.parse(this.leerAjuste("argon2")!) };
	}

	async crear(d: DatosDeAlta): Promise<Resultado<{ sesion: string; dispositivo: string; confianza?: string }>> {
		if (this.existe()) return mal(409, "Ya hay una cuenta con este correo.");
		const ahoraMismo = versionActual(this.env);
		const acceso = sellar(ahoraMismo, (await this.verificador("acceso", deBase64url(d.claveDeAcceso, 32)!, ahoraMismo))!);
		const posesion = sellar(ahoraMismo, (await this.verificador("posesion", deBase64url(d.posesion, 32)!, ahoraMismo))!);
		const emitido = await this.emitir(d.dispositivo, d.confiar, d.cuenta);

		this.ctx.storage.transactionSync(() => {
			this.ponerAjuste("cuenta", d.cuenta);
			this.ponerAjuste("correo", d.correo);
			this.ponerAjuste("creada", String(Date.now()));
			this.ponerAjuste("sal", d.sal);
			this.ponerAjuste("argon2", JSON.stringify(d.argon2));
			this.ponerAjuste("acceso", acceso);
			this.ponerAjuste("posesion", posesion);
			this.ponerAjuste("pimienta", "1");
			this.ponerAjuste("version", "0");
			if (d.llaves !== undefined) this.ponerAjuste("llaves", JSON.stringify(d.llaves));
			this.guardarEmitido(emitido, false);
			this.apuntar("alta", emitido.nombre);
		});
		return bien({ sesion: emitido.sesion, dispositivo: emitido.id, confianza: emitido.confianza });
	}

	// ---------------------------------------------------------------- entrar

	async entrar(p: {
		claveDeAcceso: unknown;
		confianza?: unknown;
		dispositivo: unknown;
	}): Promise<
		Resultado<
			| { sesion: string; dispositivo: string }
			| { reto: string; codigo: string; correo: string }
		>
	> {
		if (!this.existe()) {
			await this.verificador("acceso", new Uint8Array(32), versionActual(this.env)); // el mismo trabajo que con cuenta
			return mal(401, NO_VALE);
		}
		const ahora = Date.now();
		const equipoDeConfianza = await this.equipoDeConfianza(p.confianza);
		// **La contraseña se comprueba antes que el freno**, y el orden importa
		// (revisión del 2026-09-23). Al revés, una cuenta que existe contestaba 429
		// tras diez fallos y un correo desconocido contestaba siempre 401: once
		// intentos con una contraseña inventada decían si ese correo tiene cuenta,
		// justo lo que `docs/seguridad.md` promete que no se puede saber. Quien
		// acierta la contraseña ya sabe que la cuenta existe, así que a ése sí se le
		// puede decir que está frenada; a los demás se les contesta lo de siempre.
		const cuadraAcceso = typeof p.claveDeAcceso === "string" ? await this.coincide("acceso", p.claveDeAcceso) : "no";
		if (cuadraAcceso === "imposible") return mal(409, "Esta cuenta no se puede abrir con esta versión del servidor. Escribe a info@webcafeina.com.");
		if (cuadraAcceso !== "si") {
			this.sql.exec("INSERT INTO fallos (momento) VALUES (?)", ahora);
			this.sql.exec("DELETE FROM fallos WHERE momento <= ?", ahora - VENTANA_DE_FALLOS);
			this.apuntar("entrada-fallida");
			return mal(401, NO_VALE);
		}
		const fallos = this.sql
			.exec<{ n: number }>("SELECT COUNT(*) AS n FROM fallos WHERE momento > ?", ahora - VENTANA_DE_FALLOS)
			.one().n;
		// Frenar sin bloquear: tras muchos fallos, solo entra quien ya era de confianza.
		// Bloquear la cuenta entera sería dejar que cualquiera la cierre a su dueño.
		if (fallos >= FALLOS_PARA_FRENAR && !equipoDeConfianza) {
			return mal(429, "Demasiados intentos fallidos. Espera unos minutos o entra desde un equipo de confianza.");
		}

		if (equipoDeConfianza) {
			const sesion = await this.nuevaSesion();
			this.ctx.storage.transactionSync(() => {
				this.sql.exec(
					"INSERT INTO sesiones (huella, dispositivo, creada, vista, restringida) VALUES (?, ?, ?, ?, 0)",
					sesion.huella, equipoDeConfianza, ahora, ahora,
				);
				this.sql.exec(
					"UPDATE dispositivos SET visto = ?, confianza_caduca = ? WHERE id = ?",
					ahora, ahora + CONFIANZA, equipoDeConfianza,
				);
				this.apuntar("entrada", equipoDeConfianza);
			});
			return bien({ sesion: sesion.token, dispositivo: equipoDeConfianza });
		}

		const reto = await this.nuevoReto("entrar", { nombre: nombreDeEquipo(p.dispositivo) });
		if (!reto.ok) return reto;
		return bien({ ...reto.datos, correo: this.leerAjuste("correo")! });
	}

	async confirmar(p: {
		reto: string;
		codigo: unknown;
		confiar: boolean;
	}): Promise<Resultado<{ sesion: string; dispositivo: string; confianza?: string; correo: string; nombre: string }>> {
		const r = await this.gastarReto(p.reto, "entrar", p.codigo);
		if (!r.ok) return r;
		const nombre = nombreDeEquipo((r.datos as { nombre?: string }).nombre);
		const emitido = await this.emitir(nombre, p.confiar);
		this.ctx.storage.transactionSync(() => {
			this.guardarEmitido(emitido, false);
			this.apuntar("equipo-nuevo", nombre);
		});
		return bien({
			sesion: emitido.sesion,
			dispositivo: emitido.id,
			confianza: emitido.confianza,
			correo: this.leerAjuste("correo")!,
			nombre,
		});
	}

	async cerrarSesion(token: string): Promise<Resultado<null>> {
		const h = await this.huellaDeSesion(token);
		if (h) this.sql.exec("DELETE FROM sesiones WHERE huella = ?", h);
		return bien(null);
	}

	// ---------------------------------------------------------------- la bóveda

	async leer(token: string, siNoCoincide: number | null, ref = ""): Promise<Resultado<{ version: number; datos: ArrayBuffer | null }>> {
		const s = await this.sesion(token, true);
		if (!s.ok) return s;
		const version = this.version(ref);
		if (version === 0) {
			return mal(404, ref === "" ? "Esta cuenta todavía no tiene bóveda." : "Esa bóveda no está en esta cuenta.");
		}
		if (siNoCoincide === version) return bien({ version, datos: null });
		return bien({ version, datos: this.bytesDe(version, ref) });
	}

	async escribir(token: string, siCoincide: number, datos: ArrayBuffer, ref = ""): Promise<Resultado<{ version: number }> & { version?: number }> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const comprobado = this.comprobarDocumento(datos);
		if (!comprobado.ok) return comprobado;
		const huella = aHex(await sha256(new Uint8Array(datos)));
		const ahora = Date.now();

		return this.ctx.storage.transactionSync(() => {
			const actual = this.version(ref);
			if (siCoincide !== actual) return { ...mal(412, "La bóveda ha cambiado en el servidor."), version: actual };
			// **El freno de subidas es de la cuenta y lo comparten todas sus bóvedas, a
			// propósito**: lo que protege es el almacenamiento del Durable Object, que es
			// uno. Y en la práctica no aprieta más que antes, porque **solo se sube la
			// bóveda que está abierta** y solo hay una.
			const subidas = this.sql
				.exec<{ n: number }>("SELECT COUNT(*) AS n FROM subidas WHERE momento > ?", ahora - HORA)
				.one().n;
			if (subidas >= SUBIDAS_POR_HORA) return mal(429, "Demasiadas subidas en una hora. Espera un poco.");
			// Y un tope de bóvedas, que se mira solo al estrenar una.
			if (ref !== "" && actual === 0 && this.cuantasBovedas() >= BOVEDAS_POR_CUENTA) {
				return mal(409, `Esta cuenta ya tiene ${BOVEDAS_POR_CUENTA} bóvedas de proyecto.`);
			}
			// **Cada bóveda recuerda cuál es, y rechaza otra**: es lo que impide que un
			// cliente despistado suba su bóveda personal encima de la de un proyecto.
			const idGuardado = this.leerAjuste(claveDe("idBoveda", ref));
			if (idGuardado && idGuardado !== comprobado.datos.id) {
				return mal(409, "Esa bóveda no es la de esta cuenta.");
			}
			const nueva = this.escribirVersion(actual + 1, datos, huella, ahora, ref);
			this.ponerAjuste(claveDe("idBoveda", ref), comprobado.datos.id);
			// **La recuperación solo se guarda de la personal.** Un proyecto no tiene
			// clave de recuperación propia (ADR 0050): la de la personal lo recupera, y
			// guardar aquí un sobre que no abre nada sería prometer una puerta que no hay.
			if (ref === "") this.ponerAjuste("recuperacion", JSON.stringify(comprobado.datos.recuperacion));
			this.sql.exec("INSERT INTO subidas (momento) VALUES (?)", ahora);
			this.sql.exec("DELETE FROM subidas WHERE momento <= ?", ahora - HORA);
			this.tocar(s.datos.dispositivo, ahora);
			return bien({ version: nueva });
		});
	}

	async versiones(token: string, ref = ""): Promise<Resultado<{ version: number; fecha: number; tamano: number }[]>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		return bien(
			this.sql
				.exec<{ version: number; fecha: number; tamano: number }>(
					"SELECT version, fecha, tamano FROM versiones WHERE ref = ? ORDER BY version DESC",
					ref,
				)
				.toArray(),
		);
	}

	async unaVersion(token: string, version: number, ref = ""): Promise<Resultado<ArrayBuffer>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const hay = this.sql.exec("SELECT 1 FROM versiones WHERE ref = ? AND version = ?", ref, version).toArray().length > 0;
		if (!hay) return mal(404, "Esa versión ya no se guarda.");
		return bien(this.bytesDe(version, ref));
	}

	/**
	 * Las bóvedas de proyecto que hay en esta cuenta: qué referencias y por qué
	 * versión van. **Sin nombres**, que viven cifrados dentro de la bóveda personal.
	 *
	 * Es lo que necesita un equipo nuevo para saber qué bajarse, y lo único que el
	 * servidor sabe de ellas.
	 */
	async bovedas(token: string): Promise<Resultado<{ ref: string; version: number; tamano: number; fecha: number }[]>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const filas = this.sql
			.exec<{ ref: string; version: number; tamano: number; fecha: number }>(
				`SELECT v.ref AS ref, v.version AS version, v.tamano AS tamano, v.fecha AS fecha
				 FROM versiones v
				 WHERE v.ref <> '' AND v.version = (SELECT MAX(version) FROM versiones WHERE ref = v.ref)
				 ORDER BY v.ref`,
			)
			.toArray();
		return bien(filas);
	}

	/**
	 * Borra una bóveda de proyecto del servidor, con todas sus versiones.
	 *
	 * **La bóveda personal no se borra por aquí**: para eso está borrar la cuenta,
	 * que pide la contraseña y un código por correo. Dejar que una ruta de bóveda se
	 * llevara la principal sería una puerta de atrás a eso.
	 */
	async olvidarBoveda(token: string, ref: string): Promise<Resultado<{ borradas: number }>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		if (ref === "") return mal(400, "Para borrar tu bóveda hay que borrar la cuenta.");
		return this.ctx.storage.transactionSync(() => {
			const cuantas = this.sql
				.exec<{ n: number }>("SELECT COUNT(*) AS n FROM versiones WHERE ref = ?", ref)
				.one().n;
			this.sql.exec("DELETE FROM trozos WHERE ref = ?", ref);
			this.sql.exec("DELETE FROM versiones WHERE ref = ?", ref);
			this.sql.exec("DELETE FROM ajustes WHERE clave IN (?, ?)", claveDe("version", ref), claveDe("idBoveda", ref));
			return bien({ borradas: cuantas });
		});
	}

	private cuantasBovedas(): number {
		return this.sql
			.exec<{ n: number }>("SELECT COUNT(DISTINCT ref) AS n FROM versiones WHERE ref <> ''")
			.one().n;
	}

	/**
	 * Cambia la contraseña: bóveda con la ranura nueva, verificador y sal nuevos, y
	 * las demás sesiones fuera. **Todo o nada**, y el cliente guarda en local solo
	 * cuando esto ha dicho que sí.
	 *
	 * Se demuestra con la **posesión** —derivada de la clave de bóveda, que no
	 * cambia nunca— y no con la contraseña vieja: así sirve igual para quien la ha
	 * olvidado y entra con su clave de recuperación.
	 */
	async cambiarClave(
		token: string,
		p: { posesion: unknown; sal: string; argon2: Argon2; claveDeAcceso: string; siCoincide: number; documento: ArrayBuffer },
	): Promise<Resultado<{ version: number; sesion?: string; correo: string }> & { version?: number }> {
		const s = await this.sesion(token, true);
		if (!s.ok) return s;
		const cuadraPosesion = typeof p.posesion === "string" ? await this.coincide("posesion", p.posesion) : "no";
		if (cuadraPosesion === "imposible") return mal(409, "Esta cuenta no se puede abrir con esta versión del servidor. Escribe a info@webcafeina.com.");
		if (cuadraPosesion !== "si") {
			return mal(403, "Esa clave no abre la bóveda de esta cuenta.");
		}
		const comprobado = this.comprobarDocumento(p.documento);
		if (!comprobado.ok) return comprobado;
		const acceso = sellar(versionActual(this.env), (await this.verificador("acceso", deBase64url(p.claveDeAcceso, 32)!, versionActual(this.env)))!);
		const huella = aHex(await sha256(new Uint8Array(p.documento)));
		const huellaActual = await this.huellaDeSesion(token);
		const nueva = s.datos.restringida ? await this.nuevaSesion() : null;
		const ahora = Date.now();

		return this.ctx.storage.transactionSync(() => {
			const actual = this.version();
			if (p.siCoincide !== actual) return { ...mal(412, "La bóveda ha cambiado en el servidor."), version: actual };
			const idGuardado = this.leerAjuste("idBoveda");
			if (idGuardado && idGuardado !== comprobado.datos.id) return mal(409, "Esa bóveda no es la de esta cuenta.");
			const version = this.escribirVersion(actual + 1, p.documento, huella, ahora);
			this.ponerAjuste("idBoveda", comprobado.datos.id);
			this.ponerAjuste("recuperacion", JSON.stringify(comprobado.datos.recuperacion));
			this.ponerAjuste("acceso", acceso);
			this.ponerAjuste("sal", p.sal);
			this.ponerAjuste("argon2", JSON.stringify(p.argon2));
			// Fuera las demás sesiones. La de quien cambia sigue, salvo que fuera la
			// restringida de una recuperación: ésa se cambia por una normal.
			this.sql.exec("DELETE FROM sesiones WHERE huella != ?", huellaActual ?? "");
			// **Y fuera los testigos de confianza de los demás equipos, y los retos
			// vivos** (revisión del 2026-09-23). Sin esto, quien hubiera pasado una vez
			// el segundo factor —o hubiera copiado `cuenta.json`, que lo lleva en
			// claro— entraba **sin código** en cuanto consiguiera la contraseña nueva,
			// y cambiar la contraseña porque alguien la sabe es justo ese caso. Si
			// quien cambia viene de una recuperación no hay equipo al que respetar:
			// se van todos.
			const respetar = s.datos.restringida ? "" : (s.datos.dispositivo ?? "");
			this.sql.exec(
				"UPDATE dispositivos SET confianza = NULL, confianza_caduca = 0 WHERE id != ?",
				respetar,
			);
			this.sql.exec("DELETE FROM retos");
			if (nueva) {
				this.sql.exec("DELETE FROM sesiones WHERE huella = ?", huellaActual ?? "");
				this.sql.exec(
					"INSERT INTO sesiones (huella, dispositivo, creada, vista, restringida) VALUES (?, ?, ?, ?, 0)",
					nueva.huella, s.datos.dispositivo, ahora, ahora,
				);
			}
			this.apuntar("clave-cambiada", s.datos.dispositivo);
			return bien({ version, sesion: nueva?.token, correo: this.leerAjuste("correo")! });
		});
	}

	// ---------------------------------------------------------------- recuperar

	async retoRecuperacion(): Promise<Resultado<{ reto: string; codigo: string; correo: string }> | null> {
		if (!this.existe()) return null;
		// **Pedir otro no mata el anterior** (revisión del 2026-09-23). Antes se
		// borraba aquí, así que cualquiera que supiera el correo podía invalidar el
		// código que su dueño estuviera escribiendo, una y otra vez. Ahora valen los
		// que estén vivos —diez minutos— y se comprueban todos; el cupo por propósito
		// es lo que impide que se acumulen.
		const r = await this.nuevoReto("recuperar", {});
		if (!r.ok) return r;
		return bien({ ...r.datos, correo: this.leerAjuste("correo")! });
	}

	/** Con el código bueno se entrega **solo el sobre de recuperación**, y un reto para el último paso. */
	async comprobarRecuperacion(codigo: unknown): Promise<Resultado<{ reto: string; sobre: unknown }>> {
		if (!this.existe()) return mal(401, CODIGO_MALO);
		// Se prueban **todos los que estén vivos**, del más nuevo al más viejo: quien
		// escribe un código no tiene por qué saber cuál de los que ha pedido es.
		const vivos = this.sql
			.exec<{ id: string }>("SELECT id FROM retos WHERE proposito = 'recuperar' ORDER BY caduca DESC")
			.toArray();
		if (vivos.length === 0) return mal(401, CODIGO_MALO);
		let r: Resultado<unknown> = mal(401, CODIGO_MALO);
		for (const vivo of vivos) {
			r = await this.gastarReto(`${this.leerAjuste("cuenta")}.${vivo.id}`, "recuperar", codigo);
			if (r.ok) break;
		}
		if (!r.ok) return r;
		const sobre = JSON.parse(this.leerAjuste("recuperacion") ?? "null");
		if (!sobre) return mal(404, "Esta cuenta no tiene clave de recuperación en el servidor.");
		const siguiente = await this.nuevoReto("recuperar-fin", {}, false);
		if (!siguiente.ok) return siguiente;
		return bien({ reto: siguiente.datos.reto, sobre });
	}

	async terminarRecuperacion(
		reto: string,
		posesion: unknown,
		dispositivo: unknown,
	): Promise<Resultado<{ sesion: string; dispositivo: string }>> {
		// El reto del último paso no lleva código: lo que se demuestra es la posesión.
		const fila = this.sql
			.exec<{ caduca: number; intentos: number }>(
				"SELECT caduca, intentos FROM retos WHERE id = ? AND proposito = 'recuperar-fin'",
				idDeReto(reto),
			)
			.toArray()[0];
		if (!fila || fila.caduca < Date.now() || fila.intentos >= INTENTOS_POR_RETO) {
			return mal(401, "La recuperación ha caducado. Empieza otra vez.");
		}
		const cuadraPosesion = typeof posesion === "string" ? await this.coincide("posesion", posesion) : "no";
		if (cuadraPosesion === "imposible") return mal(409, "Esta cuenta no se puede abrir con esta versión del servidor. Escribe a info@webcafeina.com.");
		if (cuadraPosesion !== "si") {
			this.sql.exec("UPDATE retos SET intentos = intentos + 1 WHERE id = ?", idDeReto(reto));
			return mal(403, "Esa clave de recuperación no abre la bóveda de esta cuenta.");
		}
		const nombre = nombreDeEquipo(dispositivo);
		const emitido = await this.emitir(nombre, false);
		this.ctx.storage.transactionSync(() => {
			this.sql.exec("DELETE FROM retos WHERE id = ?", idDeReto(reto));
			this.guardarEmitido(emitido, true);
			this.apuntar("recuperacion", nombre);
		});
		return bien({ sesion: emitido.sesion, dispositivo: emitido.id });
	}

	// ---------------------------------------------------------------- equipos

	async dispositivos(token: string): Promise<Resultado<Equipo[]>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const lista = this.sql
			.exec<FilaDeEquipo>("SELECT id, nombre, creado, visto FROM dispositivos ORDER BY visto DESC")
			.toArray()
			.map((e) => ({ ...e, actual: e.id === s.datos.dispositivo }));
		return bien(lista);
	}

	async olvidarDispositivo(token: string, id: string): Promise<Resultado<null>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		this.ctx.storage.transactionSync(() => {
			this.sql.exec("DELETE FROM sesiones WHERE dispositivo = ?", id);
			this.sql.exec("DELETE FROM dispositivos WHERE id = ?", id);
			this.apuntar("equipo-olvidado", id);
		});
		return bien(null);
	}

	// ---------------------------------------------------------------- borrar y exportar

	async retoBorrado(token: string): Promise<Resultado<{ reto: string; codigo: string; correo: string }>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const r = await this.nuevoReto("borrar", {});
		if (!r.ok) return r;
		return bien({ ...r.datos, correo: this.leerAjuste("correo")! });
	}

	/** Borra la cuenta entera: pide la contraseña **y** un código, porque no tiene vuelta atrás. */
	async borrar(token: string, p: { claveDeAcceso: unknown; reto: string; codigo: unknown }): Promise<Resultado<{ correo: string }>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const cuadraAcceso = typeof p.claveDeAcceso === "string" ? await this.coincide("acceso", p.claveDeAcceso) : "no";
		if (cuadraAcceso === "imposible") return mal(409, "Esta cuenta no se puede abrir con esta versión del servidor. Escribe a info@webcafeina.com.");
		if (cuadraAcceso !== "si") {
			return mal(401, "La contraseña no es correcta.");
		}
		const r = await this.gastarReto(p.reto, "borrar", p.codigo);
		if (!r.ok) return r;
		const correo = this.leerAjuste("correo")!;
		await this.ctx.storage.deleteAll();
		this.migrar(); // deleteAll se lleva también las tablas
		return bien({ correo });
	}

	async exportar(token: string): Promise<Resultado<unknown>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const version = this.version();
		return bien({
			formato: 1,
			exportada: new Date().toISOString(),
			cuenta: {
				id: this.leerAjuste("cuenta"),
				correo: this.leerAjuste("correo"),
				creada: new Date(Number(this.leerAjuste("creada"))).toISOString(),
			},
			equipos: this.sql
				.exec<FilaDeEquipo>("SELECT id, nombre, creado, visto FROM dispositivos ORDER BY creado")
				.toArray()
				.map((e) => ({ ...e, creado: new Date(e.creado).toISOString(), visto: new Date(e.visto).toISOString() })),
			eventos: this.sql
				.exec<{ momento: number; tipo: string; detalle: string }>("SELECT momento, tipo, detalle FROM eventos ORDER BY momento")
				.toArray()
				.map((e) => ({ ...e, momento: new Date(e.momento).toISOString() })),
			versiones: this.sql
				.exec<{ version: number; fecha: number; tamano: number }>(
					"SELECT version, fecha, tamano FROM versiones WHERE ref = '' ORDER BY version",
				)
				.toArray()
				.map((v) => ({ ...v, fecha: new Date(v.fecha).toISOString() })),
			// La bóveda, tal como está en el servidor: cifrada. Webcafeína no tiene con qué abrirla.
			boveda: version ? new TextDecoder().decode(this.bytesDe(version)) : null,
			// **Y las bóvedas de proyecto, enteras** (ADR 0050). Sin esto, la
			// exportación dejaría de ser lo que las condiciones prometen —todo lo que la
			// cuenta guarda— **el día que alguien creara su primer proyecto**, y nadie se
			// enteraría hasta necesitarla. Van cifradas, como la personal.
			proyectos: this.sql
				.exec<{ ref: string }>("SELECT DISTINCT ref FROM versiones WHERE ref <> '' ORDER BY ref")
				.toArray()
				.map(({ ref }) => {
					const v = this.version(ref);
					return {
						ref,
						versiones: this.sql
							.exec<{ version: number; fecha: number; tamano: number }>(
								"SELECT version, fecha, tamano FROM versiones WHERE ref = ? ORDER BY version",
								ref,
							)
							.toArray()
							.map((x) => ({ ...x, fecha: new Date(x.fecha).toISOString() })),
						boveda: v ? new TextDecoder().decode(this.bytesDe(v, ref)) : null,
					};
				}),
		});
	}

	// ---------------------------------------------------------------- piezas

	/** Lo mínimo para no guardar basura: que sea una bóveda de Esfinge y de quién es. */
	private comprobarDocumento(
		datos: ArrayBuffer,
	): Resultado<{ id: string; recuperacion: { contenedor: string; codificacion?: string } | null }> {
		if (datos.byteLength > TAMANO_MAXIMO) return mal(413, "La bóveda es demasiado grande para subirla.");
		let doc: unknown;
		try {
			doc = JSON.parse(new TextDecoder("utf-8", { fatal: true, ignoreBOM: false }).decode(datos));
		} catch {
			return mal(400, "Eso no es una bóveda de Esfinge.");
		}
		const d = doc as Record<string, unknown>;
		if (
			typeof d !== "object" || d === null ||
			d.esfinge !== "bóveda" ||
			typeof d.formato !== "number" ||
			typeof d.id !== "string" || d.id.length < 8 || d.id.length > 128 ||
			!Array.isArray(d.sobres)
		) {
			return mal(400, "Eso no es una bóveda de Esfinge.");
		}
		const sobre = (d.sobres as Record<string, unknown>[]).find((s) => s?.tipo === "recuperacion");
		const recuperacion =
			sobre && typeof sobre.contenedor === "string"
				? {
						contenedor: sobre.contenedor,
						...(typeof sobre.codificacion === "string" ? { codificacion: sobre.codificacion } : {}),
					}
				: null;
		return bien({ id: d.id, recuperacion });
	}

	private escribirVersion(version: number, datos: ArrayBuffer, huella: string, ahora: number, ref = ""): number {
		for (let i = 0, n = 0; i < datos.byteLength || n === 0; i += TROZO, n++) {
			this.sql.exec(
				"INSERT INTO trozos (ref, version, n, datos) VALUES (?, ?, ?, ?)",
				ref, version, n, datos.slice(i, i + TROZO),
			);
		}
		this.sql.exec(
			"INSERT INTO versiones (ref, version, fecha, tamano, huella) VALUES (?, ?, ?, ?, ?)",
			ref, version, ahora, datos.byteLength, huella,
		);
		this.ponerAjuste(claveDe("version", ref), String(version));
		this.podar(ahora, ref);
		return version;
	}

	/**
	 * Qué versiones se conservan: las diez últimas y la última de cada uno de los
	 * treinta días anteriores. Con diez a secas, una tarde editando se come todo el
	 * margen para deshacer una fusión mala; con todas, la base crece sin tope.
	 */
	private podar(ahora: number, ref = "") {
		const todas = this.sql
			.exec<{ version: number; fecha: number }>(
				"SELECT version, fecha FROM versiones WHERE ref = ? ORDER BY version DESC",
				ref,
			)
			.toArray();
		const cuantas = ref === "" ? VERSIONES_RECIENTES : VERSIONES_DE_PROYECTO;
		const quedan = new Set<number>(todas.slice(0, cuantas).map((v) => v.version));
		const dias = new Set<string>();
		for (const v of todas) {
			if (v.fecha < ahora - DIAS_CON_VERSION * DIA) continue;
			const dia = new Date(v.fecha).toISOString().slice(0, 10);
			if (!dias.has(dia)) {
				dias.add(dia);
				quedan.add(v.version);
			}
		}
		for (const v of todas) {
			if (quedan.has(v.version)) continue;
			this.sql.exec("DELETE FROM trozos WHERE ref = ? AND version = ?", ref, v.version);
			this.sql.exec("DELETE FROM versiones WHERE ref = ? AND version = ?", ref, v.version);
		}
	}

	private bytesDe(version: number, ref = ""): ArrayBuffer {
		const trozos = this.sql
			.exec<{ datos: ArrayBuffer }>(
				"SELECT datos FROM trozos WHERE ref = ? AND version = ? ORDER BY n",
				ref, version,
			)
			.toArray();
		const total = trozos.reduce((t, x) => t + x.datos.byteLength, 0);
		const salida = new Uint8Array(total);
		let i = 0;
		for (const t of trozos) {
			salida.set(new Uint8Array(t.datos), i);
			i += t.datos.byteLength;
		}
		return salida.buffer;
	}

	private async nuevaSesion(cuenta = this.leerAjuste("cuenta")): Promise<{ token: string; huella: string }> {
		const secreto = aBase64url(azar(32));
		return {
			token: `s1.${cuenta}.${secreto}`,
			huella: aHex(await sha256(secreto)),
		};
	}

	private async huellaDeSesion(token: string): Promise<string | null> {
		const partes = typeof token === "string" ? token.split(".") : [];
		if (partes.length !== 3 || partes[0] !== "s1") return null;
		return aHex(await sha256(partes[2]));
	}

	/** Comprueba una sesión y la alarga. La restringida —la de una recuperación— solo vale donde se diga. */
	// ============================================================ compartir

	/**
	 * Publica las llaves públicas de esta cuenta, para que otros puedan cifrar
	 * hacia ella (ADR 0043). Se pueden volver a publicar: una bóveda restaurada de
	 * una copia vieja conserva su identidad, pero una bóveda nueva trae otra.
	 */
	async ponerLlaves(token: string, llaves: unknown): Promise<Resultado<Record<string, never>>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		if (!llavesValidas(llaves)) return mal(400, "Esas llaves no son válidas.");
		this.ponerAjuste("llaves", JSON.stringify(llaves));
		return bien({});
	}

	/**
	 * Pone al día el verificador de posesión con la pimienta de ahora, si hacía falta.
	 *
	 * **Existe solo para que la rotación pueda terminar** (ADR 0046). El verificador de
	 * acceso migra al entrar, pero el de posesión solo se comprueba al cambiar la
	 * maestra y al recuperar —dos cosas que casi nadie hace nunca—, así que sin esto el
	 * contador de D1 **no llegaría a cero jamás** y la pimienta vieja no se podría
	 * borrar. Se descubrió al ir a rotar de verdad, no escribiéndolo.
	 *
	 * La llama el cliente de cortesía al arrancar la sincronización, que es el momento
	 * en que tiene la bóveda abierta **y** una sesión. Al entrar no puede: la posesión
	 * se deriva de la clave de la bóveda, y al entrar la bóveda todavía no está abierta.
	 *
	 * **Contesta lo mismo cuadre o no.** Decir si cuadró convertiría esto en un sitio
	 * donde probar claves de posesión con una sesión robada.
	 */
	async refrescarPosesion(token: string, posesion: unknown): Promise<Resultado<Record<string, never>>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		if (typeof posesion === "string") await this.coincide("posesion", posesion);
		return bien({});
	}

	/** Las llaves publicadas, o null si esta cuenta no tiene o no existe. */
	llaves(): Llaves | null {
		if (!this.existe()) return null;
		const crudo = this.leerAjuste("llaves");
		if (!crudo) return null;
		try {
			const l = JSON.parse(crudo);
			return llavesValidas(l) ? l : null;
		} catch {
			return null;
		}
	}

	/**
	 * Guarda un sobre en el buzón de esta cuenta.
	 *
	 * **El servidor no puede abrirlo** —va cifrado hacia la llave de esta cuenta—,
	 * así que lo único que hace aquí es contarlo y guardarlo. Los topes son lo que
	 * impide que el buzón de alguien se use como vertedero.
	 */
	async recibir(sobre: string): Promise<Resultado<{ id: string }>> {
		if (!this.existe()) return mal(404, "No hay cuenta.");
		if (sobre.length > TAMANO_DE_ENVIO) return mal(413, "Ese envío es demasiado grande.");
		const cuantos = this.sql.exec<{ n: number }>("SELECT COUNT(*) AS n FROM buzon").one().n;
		if (cuantos >= ENVIOS_EN_BUZON) return mal(429, "El buzón de esa persona está lleno.");
		const id = aHex(azar(16));
		this.sql.exec("INSERT INTO buzon (id, sobre, momento) VALUES (?, ?, ?)", id, sobre, Date.now());
		this.apuntar("envio-recibido", "");
		return bien({ id });
	}

	/** Lo que espera en el buzón, lo más nuevo primero. */
	async buzon(token: string): Promise<Resultado<{ envios: { id: string; momento: number; sobre: unknown }[] }>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const filas = this.sql
			.exec<{ id: string; sobre: string; momento: number }>("SELECT id, sobre, momento FROM buzon ORDER BY momento DESC")
			.toArray();
		return bien({ envios: filas.map((f) => ({ id: f.id, momento: f.momento, sobre: JSON.parse(f.sobre) })) });
	}

	/** Quita uno del buzón: lo mismo vale para aceptarlo que para tirarlo. */
	async tirarDelBuzon(token: string, id: string): Promise<Resultado<Record<string, never>>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		this.sql.exec("DELETE FROM buzon WHERE id = ?", id);
		return bien({});
	}

	private async sesion(token: string, admiteRestringida: boolean): Promise<Resultado<Sesion>> {
		const h = await this.huellaDeSesion(token);
		if (!h || !this.existe()) return mal(401, SESION_CADUCADA);
		const fila = this.sql
			.exec<{ dispositivo: string; creada: number; vista: number; restringida: number }>(
				"SELECT dispositivo, creada, vista, restringida FROM sesiones WHERE huella = ?",
				h,
			)
			.toArray()[0];
		const ahora = Date.now();
		if (!fila) return mal(401, SESION_CADUCADA);
		const restringida = fila.restringida === 1;
		const caducada = restringida
			? fila.creada + SESION_RESTRINGIDA < ahora
			: fila.vista + SESION_DESLIZANTE < ahora || fila.creada + SESION_MAXIMA < ahora;
		if (caducada) {
			this.sql.exec("DELETE FROM sesiones WHERE huella = ?", h);
			return mal(401, SESION_CADUCADA);
		}
		if (restringida && !admiteRestringida) return mal(403, "Termina primero de poner la contraseña nueva.");
		if (fila.vista < ahora - HORA) {
			this.sql.exec("UPDATE sesiones SET vista = ? WHERE huella = ?", ahora, h);
			this.tocar(fila.dispositivo, ahora);
		}
		return bien({ dispositivo: fila.dispositivo, restringida });
	}

	private tocar(dispositivo: string, ahora: number) {
		this.sql.exec("UPDATE dispositivos SET visto = ? WHERE id = ?", ahora, dispositivo);
	}

	private async equipoDeConfianza(confianza: unknown): Promise<string | null> {
		if (typeof confianza !== "string" || !confianza.startsWith("c1.")) return null;
		const h = aHex(await sha256(confianza.slice(3)));
		const fila = this.sql
			.exec<{ id: string; confianza_caduca: number }>(
				"SELECT id, confianza_caduca FROM dispositivos WHERE confianza = ?",
				h,
			)
			.toArray()[0];
		if (!fila || fila.confianza_caduca < Date.now()) return null;
		return fila.id;
	}

	private async emitir(nombre: string, confiar: boolean, cuenta?: string) {
		const sesion = await this.nuevaSesion(cuenta);
		const confianza = confiar ? aBase64url(azar(32)) : undefined;
		return {
			id: aBase64url(azar(12)),
			nombre,
			sesion: sesion.token,
			huella: sesion.huella,
			confianza: confianza ? `c1.${confianza}` : undefined,
			huellaConfianza: confianza ? aHex(await sha256(confianza)) : null,
		};
	}

	private guardarEmitido(e: Awaited<ReturnType<Cuenta["emitir"]>>, restringida: boolean) {
		const ahora = Date.now();
		this.sql.exec(
			"INSERT INTO dispositivos (id, nombre, creado, visto, confianza, confianza_caduca) VALUES (?, ?, ?, ?, ?, ?)",
			e.id, e.nombre, ahora, ahora, e.huellaConfianza, e.huellaConfianza ? ahora + CONFIANZA : null,
		);
		this.sql.exec(
			"INSERT INTO sesiones (huella, dispositivo, creada, vista, restringida) VALUES (?, ?, ?, ?, ?)",
			e.huella, e.id, ahora, ahora, restringida ? 1 : 0,
		);
	}

	private async nuevoReto(
		proposito: Proposito,
		datos: object,
		cuentaParaElTope = true,
	): Promise<Resultado<{ reto: string; codigo: string }>> {
		const ahora = Date.now();
		if (cuentaParaElTope) {
			const creados = this.sql
				.exec<{ n: number }>(
					"SELECT COUNT(*) AS n FROM retos_creados WHERE momento > ? AND proposito = ?",
					ahora - HORA, proposito,
				)
				.one().n;
			if (creados >= RETOS_POR_HORA) {
				return mal(429, "Se han pedido demasiados códigos. Espera un rato antes de pedir otro.");
			}
			const delDia = this.sql
				.exec<{ n: number }>("SELECT COUNT(*) AS n FROM retos_creados WHERE momento > ?", ahora - DIA)
				.one().n;
			if (delDia >= RETOS_POR_DIA) {
				return mal(429, "Se han pedido demasiados códigos hoy. Prueba mañana.");
			}
		}
		const id = aBase64url(azar(16));
		const codigo = codigoDeSeisCifras();
		const huella = (await conLaPimienta(this.env, versionActual(this.env), `codigo|${id}|${codigo}`))!;
		this.ctx.storage.transactionSync(() => {
			this.sql.exec("DELETE FROM retos WHERE caduca < ?", ahora);
			this.sql.exec("DELETE FROM retos_creados WHERE momento <= ?", ahora - DIA);
			if (cuentaParaElTope) {
				this.sql.exec("INSERT INTO retos_creados (momento, proposito) VALUES (?, ?)", ahora, proposito);
			}
			this.sql.exec(
				"INSERT INTO retos (id, proposito, codigo, caduca, datos) VALUES (?, ?, ?, ?, ?)",
				id, proposito, huella, ahora + RETO, JSON.stringify(datos),
			);
		});
		return bien({ reto: `${this.leerAjuste("cuenta")}.${id}`, codigo });
	}

	/** Comprueba un código y, si vale, gasta el reto: cada código sirve una sola vez. */
	private async gastarReto(reto: string, proposito: Proposito, codigo: unknown): Promise<Resultado<unknown>> {
		const id = idDeReto(reto);
		const fila = this.sql
			.exec<{ codigo: string; caduca: number; intentos: number; datos: string }>(
				"SELECT codigo, caduca, intentos, datos FROM retos WHERE id = ? AND proposito = ?",
				id, proposito,
			)
			.toArray()[0];
		if (!fila || fila.caduca < Date.now() || fila.intentos >= INTENTOS_POR_RETO) return mal(401, CODIGO_MALO);
		const limpio = typeof codigo === "string" ? codigo.replace(/\s/g, "") : "";
		// Con las dos pimientas: un código emitido justo antes de rotar tiene que seguir
		// valiendo sus diez minutos. No se migra nada, porque caduca antes.
		const cuadra = await algunaPimientaDa(this.env, `codigo|${id}|${limpio}`, fila.codigo);
		if (!/^\d{6}$/.test(limpio) || !cuadra) {
			this.sql.exec("UPDATE retos SET intentos = intentos + 1 WHERE id = ?", id);
			return mal(401, CODIGO_MALO);
		}
		this.sql.exec("DELETE FROM retos WHERE id = ?", id);
		return bien(JSON.parse(fila.datos ?? "{}"));
	}
}

/** Un reto es «cuenta.id»: la cuenta dice a qué objeto ir y el id qué reto es. */
export function cuentaDeReto(reto: unknown): string | null {
	if (typeof reto !== "string") return null;
	const m = /^([0-9a-f]{32})\.([A-Za-z0-9_-]{22})$/.exec(reto);
	return m ? m[1] : null;
}

function idDeReto(reto: string): string {
	return reto.split(".")[1] ?? "";
}

export function cuentaDeSesion(token: unknown): string | null {
	if (typeof token !== "string") return null;
	const m = /^s1\.([0-9a-f]{32})\.[A-Za-z0-9_-]{43}$/.exec(token);
	return m ? m[1] : null;
}
