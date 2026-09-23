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
import { aBase64url, aHex, azar, codigoDeSeisCifras, deBase64url, hmac, iguales, sha256 } from "./cripto";
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
			CREATE TABLE IF NOT EXISTS versiones (version INTEGER PRIMARY KEY, fecha INTEGER NOT NULL, tamano INTEGER NOT NULL, huella TEXT NOT NULL);
			CREATE TABLE IF NOT EXISTS trozos (version INTEGER NOT NULL, n INTEGER NOT NULL, datos BLOB NOT NULL, PRIMARY KEY (version, n));
			CREATE TABLE IF NOT EXISTS sesiones (huella TEXT PRIMARY KEY, dispositivo TEXT NOT NULL, creada INTEGER NOT NULL, vista INTEGER NOT NULL, restringida INTEGER NOT NULL DEFAULT 0);
			CREATE TABLE IF NOT EXISTS dispositivos (id TEXT PRIMARY KEY, nombre TEXT NOT NULL, creado INTEGER NOT NULL, visto INTEGER NOT NULL, confianza TEXT, confianza_caduca INTEGER);
			CREATE TABLE IF NOT EXISTS retos (id TEXT PRIMARY KEY, proposito TEXT NOT NULL, codigo TEXT NOT NULL, caduca INTEGER NOT NULL, intentos INTEGER NOT NULL DEFAULT 0, datos TEXT);
			CREATE TABLE IF NOT EXISTS retos_creados (momento INTEGER NOT NULL, proposito TEXT NOT NULL DEFAULT 'entrar');
			CREATE TABLE IF NOT EXISTS fallos (momento INTEGER NOT NULL);
			CREATE TABLE IF NOT EXISTS subidas (momento INTEGER NOT NULL);
			CREATE TABLE IF NOT EXISTS eventos (momento INTEGER NOT NULL, tipo TEXT NOT NULL, detalle TEXT NOT NULL DEFAULT '');
		`);
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

	private version(): number {
		return Number(this.leerAjuste("version") ?? "0");
	}

	private apuntar(tipo: string, detalle = "") {
		this.sql.exec("INSERT INTO eventos (momento, tipo, detalle) VALUES (?, ?, ?)", Date.now(), tipo, detalle);
		this.sql.exec(
			"DELETE FROM eventos WHERE rowid NOT IN (SELECT rowid FROM eventos ORDER BY momento DESC, rowid DESC LIMIT ?)",
			EVENTOS_GUARDADOS,
		);
	}

	private async verificador(proposito: string, clave: Uint8Array): Promise<string> {
		return aHex(await hmac(this.env.PIMIENTA, `${proposito}|${aBase64url(clave)}`));
	}

	private async coincide(proposito: "acceso" | "posesion", clave: string): Promise<boolean> {
		const b = deBase64url(clave, 32);
		const guardado = this.leerAjuste(proposito);
		// Se calcula el HMAC aunque falte algo, para que un caso no tarde menos que el otro.
		const calculado = await this.verificador(proposito, b ?? new Uint8Array(32));
		if (!b || !guardado) return false;
		return iguales(new TextEncoder().encode(calculado), new TextEncoder().encode(guardado));
	}

	// ---------------------------------------------------------------- alta

	/** Lo que la pre-entrada necesita: la sal y el coste. Null si aquí no hay cuenta. */
	async prelogin(): Promise<{ sal: string; argon2: Argon2 } | null> {
		if (!this.existe()) return null;
		return { sal: this.leerAjuste("sal")!, argon2: JSON.parse(this.leerAjuste("argon2")!) };
	}

	async crear(d: DatosDeAlta): Promise<Resultado<{ sesion: string; dispositivo: string; confianza?: string }>> {
		if (this.existe()) return mal(409, "Ya hay una cuenta con este correo.");
		const acceso = await this.verificador("acceso", deBase64url(d.claveDeAcceso, 32)!);
		const posesion = await this.verificador("posesion", deBase64url(d.posesion, 32)!);
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
			await this.verificador("acceso", new Uint8Array(32)); // el mismo trabajo que con cuenta
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
		if (typeof p.claveDeAcceso !== "string" || !(await this.coincide("acceso", p.claveDeAcceso))) {
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

	async leer(token: string, siNoCoincide: number | null): Promise<Resultado<{ version: number; datos: ArrayBuffer | null }>> {
		const s = await this.sesion(token, true);
		if (!s.ok) return s;
		const version = this.version();
		if (version === 0) return mal(404, "Esta cuenta todavía no tiene bóveda.");
		if (siNoCoincide === version) return bien({ version, datos: null });
		return bien({ version, datos: this.bytesDe(version) });
	}

	async escribir(token: string, siCoincide: number, datos: ArrayBuffer): Promise<Resultado<{ version: number }> & { version?: number }> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const comprobado = this.comprobarDocumento(datos);
		if (!comprobado.ok) return comprobado;
		const huella = aHex(await sha256(new Uint8Array(datos)));
		const ahora = Date.now();

		return this.ctx.storage.transactionSync(() => {
			const actual = this.version();
			if (siCoincide !== actual) return { ...mal(412, "La bóveda ha cambiado en el servidor."), version: actual };
			const subidas = this.sql
				.exec<{ n: number }>("SELECT COUNT(*) AS n FROM subidas WHERE momento > ?", ahora - HORA)
				.one().n;
			if (subidas >= SUBIDAS_POR_HORA) return mal(429, "Demasiadas subidas en una hora. Espera un poco.");
			const idGuardado = this.leerAjuste("idBoveda");
			if (idGuardado && idGuardado !== comprobado.datos.id) {
				return mal(409, "Esa bóveda no es la de esta cuenta.");
			}
			const nueva = this.escribirVersion(actual + 1, datos, huella, ahora);
			this.ponerAjuste("idBoveda", comprobado.datos.id);
			this.ponerAjuste("recuperacion", JSON.stringify(comprobado.datos.recuperacion));
			this.sql.exec("INSERT INTO subidas (momento) VALUES (?)", ahora);
			this.sql.exec("DELETE FROM subidas WHERE momento <= ?", ahora - HORA);
			this.tocar(s.datos.dispositivo, ahora);
			return bien({ version: nueva });
		});
	}

	async versiones(token: string): Promise<Resultado<{ version: number; fecha: number; tamano: number }[]>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		return bien(
			this.sql
				.exec<{ version: number; fecha: number; tamano: number }>(
					"SELECT version, fecha, tamano FROM versiones ORDER BY version DESC",
				)
				.toArray(),
		);
	}

	async unaVersion(token: string, version: number): Promise<Resultado<ArrayBuffer>> {
		const s = await this.sesion(token, false);
		if (!s.ok) return s;
		const hay = this.sql.exec("SELECT 1 FROM versiones WHERE version = ?", version).toArray().length > 0;
		if (!hay) return mal(404, "Esa versión ya no se guarda.");
		return bien(this.bytesDe(version));
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
		if (typeof p.posesion !== "string" || !(await this.coincide("posesion", p.posesion))) {
			return mal(403, "Esa clave no abre la bóveda de esta cuenta.");
		}
		const comprobado = this.comprobarDocumento(p.documento);
		if (!comprobado.ok) return comprobado;
		const acceso = await this.verificador("acceso", deBase64url(p.claveDeAcceso, 32)!);
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
		if (typeof posesion !== "string" || !(await this.coincide("posesion", posesion))) {
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
		if (typeof p.claveDeAcceso !== "string" || !(await this.coincide("acceso", p.claveDeAcceso))) {
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
				.exec<{ version: number; fecha: number; tamano: number }>("SELECT version, fecha, tamano FROM versiones ORDER BY version")
				.toArray()
				.map((v) => ({ ...v, fecha: new Date(v.fecha).toISOString() })),
			// La bóveda, tal como está en el servidor: cifrada. Webcafeína no tiene con qué abrirla.
			boveda: version ? new TextDecoder().decode(this.bytesDe(version)) : null,
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

	private escribirVersion(version: number, datos: ArrayBuffer, huella: string, ahora: number): number {
		for (let i = 0, n = 0; i < datos.byteLength || n === 0; i += TROZO, n++) {
			this.sql.exec(
				"INSERT INTO trozos (version, n, datos) VALUES (?, ?, ?)",
				version, n, datos.slice(i, i + TROZO),
			);
		}
		this.sql.exec(
			"INSERT INTO versiones (version, fecha, tamano, huella) VALUES (?, ?, ?, ?)",
			version, ahora, datos.byteLength, huella,
		);
		this.ponerAjuste("version", String(version));
		this.podar(ahora);
		return version;
	}

	/**
	 * Qué versiones se conservan: las diez últimas y la última de cada uno de los
	 * treinta días anteriores. Con diez a secas, una tarde editando se come todo el
	 * margen para deshacer una fusión mala; con todas, la base crece sin tope.
	 */
	private podar(ahora: number) {
		const todas = this.sql
			.exec<{ version: number; fecha: number }>("SELECT version, fecha FROM versiones ORDER BY version DESC")
			.toArray();
		const quedan = new Set<number>(todas.slice(0, VERSIONES_RECIENTES).map((v) => v.version));
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
			this.sql.exec("DELETE FROM trozos WHERE version = ?", v.version);
			this.sql.exec("DELETE FROM versiones WHERE version = ?", v.version);
		}
	}

	private bytesDe(version: number): ArrayBuffer {
		const trozos = this.sql
			.exec<{ datos: ArrayBuffer }>("SELECT datos FROM trozos WHERE version = ? ORDER BY n", version)
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
		const huella = aHex(await hmac(this.env.PIMIENTA, `codigo|${id}|${codigo}`));
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
		const calculado = aHex(await hmac(this.env.PIMIENTA, `codigo|${id}|${limpio}`));
		if (!/^\d{6}$/.test(limpio) || !iguales(new TextEncoder().encode(calculado), new TextEncoder().encode(fila.codigo))) {
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
