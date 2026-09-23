// La puerta del servidor de cuentas: recibe, valida, frena y pasa cada cosa a la
// cuenta que toca (ADR 0035). La descripción de cada ruta está en LÉEME.md.
//
// Dos reglas que vienen de lo que ya costó en Esfinge:
//
// - **Nada se escribe en los registros**: ni cuerpos, ni cabeceras, ni correos, ni
//   IP. Las peticiones llevan el testigo de sesión, y lo que no se escribe no se
//   puede filtrar.
// - **Ningún contador en una variable del Worker.** Los aislados nacen y mueren
//   solos: un contador ahí se pone a cero cuando quiere, que es la misma trampa que
//   el freno del canal con el navegador. Los frenos van en el enlace `ratelimit`,
//   en D1 o en el Durable Object de la cuenta.

import { cartas, carteroPara, type Carta, type Entregado } from "./correo";
import { Cuenta, SESION_CADUCADA, cuentaDeReto, cuentaDeSesion } from "./cuenta";
import { aBase64url, aHex, azar, codigoDeSeisCifras, deBase64url, hmac, iguales } from "./cripto";
import {
	ARGON2_POR_DEFECTO,
	type Env,
	HORA,
	MINUTO,
	type Resultado,
	TAMANO_MAXIMO,
	argon2Valido,
	nombreDeEquipo,
	normalizarCorreo,
	SUITE_POR_DEFECTO,
} from "./protocolo";

export { Cuenta };

const LIMITE_JSON = 64 * 1024;
const CODIGOS_DE_ALTA_POR_HORA = 3;
const INTENTOS_DE_ALTA = 5;

/**
 * Los topes por IP y día, con el valor de producción por defecto. Solo el entorno
 * `local` —el de las pruebas de Go, donde todo llega desde 127.0.0.1— los sube.
 */
function tope(valor: string | undefined, porDefecto: number): number {
	const n = Number(valor);
	return Number.isInteger(n) && n > 0 ? n : porDefecto;
}

class Fallo extends Error {
	constructor(
		public estado: number,
		mensaje: string,
		public extra: Record<string, unknown> = {},
	) {
		super(mensaje);
	}
}

export default {
	async fetch(peticion: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
		try {
			configuracionCompleta(env);
			return await atender(peticion, env, ctx);
		} catch (e) {
			if (e instanceof Fallo) return json(e.estado, { error: e.message, ...e.extra });
			// En el Worker de pruebas se dice qué ha fallado por dentro: un 500 mudo no se
			// depura. En producción no, porque ahí no lo lee quien lo tiene que arreglar.
			const detalle = env.ENTORNO === "pruebas" ? { detalle: String(e) } : {};
			return json(500, { error: "Algo ha fallado en el servidor. Prueba otra vez en un momento.", ...detalle });
		}
	},

	/**
	 * La limpieza de D1, cada hora (revisión de los textos, 2026-09-23).
	 *
	 * **Una limpieza que solo corre dentro de una petición no cumple ningún plazo.**
	 * Las tres tablas con datos de alguien —el correo de un alta a medias, el correo
	 * de cada envío y la huella de IP de los contadores— se barrían de paso, en
	 * `empezarAlta` y en `contar`: con el registro por invitación eso puede tardar
	 * semanas en volver a pasar, así que «se borra a los diez minutos» era falso.
	 * Con el reloj del Worker, el plazo escrito es el plazo de verdad.
	 */
	async scheduled(_evento: ScheduledController, env: Env, ctx: ExecutionContext): Promise<void> {
		ctx.waitUntil(limpiar(env));
	},
} satisfies ExportedHandler<Env>;

/** Lo que caduca, fuera. Devuelve cuántas filas se ha llevado, para las pruebas. */
export async function limpiar(env: Env): Promise<number> {
	const ahora = Date.now();
	const dosDias = new Date(ahora - 2 * 24 * HORA).toISOString().slice(0, 10);
	const hechos = await env.BD.batch([
		env.BD.prepare("DELETE FROM altas WHERE caduca <= ?").bind(ahora),
		env.BD.prepare("DELETE FROM envios_alta WHERE momento <= ?").bind(ahora - HORA),
		env.BD.prepare("DELETE FROM contadores WHERE dia < ?").bind(dosDias),
	]);
	return hechos.reduce((n, r) => n + (r.meta?.changes ?? 0), 0);
}

async function atender(p: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
	const url = new URL(p.url);
	const ruta = url.pathname.replace(/\/+$/, "") || "/";
	const metodo = p.method;
	const r = (m: string, camino: string) => metodo === m && ruta === camino;

	if (ruta === "/_pruebas/buzon") {
		// Solo existe en el Worker de pruebas. En cualquier otro, igual que una ruta que no hay.
		if (env.ENTORNO !== "pruebas" || metodo !== "GET") throw new Fallo(404, "No existe.");
		return buzonDePruebas(env, url.searchParams.get("correo"));
	}

	if (r("GET", "/v1/salud")) return json(200, { registro: modoDeRegistro(env), protocolo: 1 });
	if (r("POST", "/v1/prelogin")) return prelogin(p, env);
	if (r("POST", "/v1/registro/inicio")) return empezarAlta(p, env);
	if (r("POST", "/v1/registro/fin")) return terminarAlta(p, env, ctx);
	if (r("POST", "/v1/sesion")) return entrar(p, env);
	if (r("POST", "/v1/sesion/codigo")) return confirmarEntrada(p, env, ctx);
	if (r("DELETE", "/v1/sesion")) return cerrarSesion(p, env);
	if (r("GET", "/v1/boveda")) return leerBoveda(p, env);
	if (r("PUT", "/v1/boveda")) return escribirBoveda(p, env);
	if (r("GET", "/v1/boveda/versiones")) return listarVersiones(p, env);
	if (metodo === "GET" && /^\/v1\/boveda\/versiones\/\d+$/.test(ruta)) {
		return leerVersion(p, env, Number(ruta.split("/").pop()));
	}
	if (r("PUT", "/v1/cuenta/clave")) return cambiarClave(p, env, ctx);
	if (r("POST", "/v1/recuperacion/inicio")) return empezarRecuperacion(p, env);
	if (r("POST", "/v1/recuperacion/codigo")) return comprobarRecuperacion(p, env);
	if (r("POST", "/v1/recuperacion/fin")) return terminarRecuperacion(p, env);
	if (r("GET", "/v1/dispositivos")) return listarEquipos(p, env);
	if (metodo === "DELETE" && ruta.startsWith("/v1/dispositivos/")) {
		return olvidarEquipo(p, env, decodeURIComponent(ruta.slice("/v1/dispositivos/".length)));
	}
	if (r("POST", "/v1/cuenta/borrado")) return pedirBorrado(p, env);
	if (r("DELETE", "/v1/cuenta")) return borrarCuenta(p, env, ctx);
	if (r("GET", "/v1/cuenta/exportacion")) return exportar(p, env);
	if (r("PUT", "/v1/llaves")) return publicarLlaves(p, env);
	if (r("POST", "/v1/llaves/de")) return llavesDe(p, env);
	if (r("POST", "/v1/envios")) return mandarEnvio(p, env);
	if (r("GET", "/v1/buzon")) return verBuzon(p, env);
	if (metodo === "DELETE" && ruta.startsWith("/v1/buzon/")) {
		return tirarDelBuzon(p, env, decodeURIComponent(ruta.slice("/v1/buzon/".length)));
	}

	throw new Fallo(404, "No existe.");
}

// ================================================================ alta

async function prelogin(p: Request, env: Env): Promise<Response> {
	await frenar(env.FRENO_ENTRAR, p, env);
	const { correo } = await leerJSON(p);
	const c = correoValido(correo);
	const cuenta = await cuentaDeCorreo(env, c);
	// Con cuenta o sin ella, el mismo camino: un correo que no existe recibe una sal
	// inventada pero fija, y así preguntar por correos no dice cuáles tienen cuenta.
	const datos = await objeto(env, cuenta ?? SIN_CUENTA).prelogin();
	if (datos) return json(200, datos);
	const sal = (await hmac(env.SECRETO_PRELOGIN, `sal|${c}`)).slice(0, 16);
	return json(200, { sal: aBase64url(sal), argon2: ARGON2_POR_DEFECTO });
}

async function empezarAlta(p: Request, env: Env): Promise<Response> {
	const { correo } = await leerJSON(p);
	const c = correoValido(correo);
	await puedeDarseDeAlta(env, c);
	await frenar(env.FRENO_ALTAS, p, env);
	const ip = await claveDeIP(p, env);
	await contar(env, `codigos-alta:${ip}`, tope(env.TOPE_CODIGOS_IP_DIA, 20), "Se han pedido demasiados códigos desde aquí hoy.");

	const ahora = Date.now();
	const enviados = await env.BD.prepare("SELECT COUNT(*) AS n FROM envios_alta WHERE correo = ? AND momento > ?")
		.bind(c, ahora - HORA)
		.first<{ n: number }>();
	if ((enviados?.n ?? 0) >= CODIGOS_DE_ALTA_POR_HORA) {
		throw new Fallo(429, "Se han pedido demasiados códigos para este correo. Espera un rato.");
	}
	await env.BD.batch([
		env.BD.prepare("DELETE FROM envios_alta WHERE momento <= ?").bind(ahora - HORA),
		env.BD.prepare("INSERT INTO envios_alta (correo, momento) VALUES (?, ?)").bind(c, ahora),
		// **Y el correo de quien pidió un código y no terminó, no se queda ahí para
		// siempre** (revisión de los textos, 2026-09-23). La fila del alta solo se
		// borraba al completarla, así que quien probara y se arrepintiera dejaba su
		// dirección en la base sin cuenta ni nada que la sostuviera. El código ya no
		// vale pasados diez minutos: la fila tampoco.
		env.BD.prepare("DELETE FROM altas WHERE caduca <= ?").bind(ahora),
	]);

	// Si ya tiene cuenta no se dice aquí —la respuesta es la misma—, sino en su buzón.
	if (await cuentaDeCorreo(env, c)) {
		await mandarOFallar(env, cartas.yaTienesCuenta(c));
		return json(202, {});
	}
	const codigo = codigoDeSeisCifras();
	await env.BD.prepare(
		"INSERT OR REPLACE INTO altas (correo, codigo, caduca, intentos) VALUES (?, ?, ?, 0)",
	)
		.bind(c, await huellaDeAlta(env, c, codigo), ahora + 10 * MINUTO)
		.run();
	await mandarOFallar(env, cartas.codigoDeAlta(c, codigo));
	return json(202, {});
}

async function terminarAlta(p: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
	const d = await leerJSON(p);
	const c = correoValido(d.correo);
	// **Esta ruta también pasa por el freno por IP** (revisión del 2026-09-23): es
	// donde se prueba el código de seis cifras, y sin freno la única cuenta eran los
	// cinco intentos de la fila —que vuelven a cero pidiendo otro código—. Va con el
	// freno de entrar, no con el de altas, porque lo que hace es **comprobar un
	// secreto**, como `/v1/sesion` y `/v1/sesion/codigo`; el de altas ya lo gastó
	// `/v1/registro/inicio` al mandar el código, y cobrarlo dos veces por el mismo
	// registro dejaría al tercer intento de alta del día sin poder terminar.
	await frenar(env.FRENO_ENTRAR, p, env);
	await puedeDarseDeAlta(env, c);
	if (!deBase64url(d.sal, 16)) throw new Fallo(400, "Falta la sal o no es válida.");
	if (!argon2Valido(d.argon2)) throw new Fallo(400, "Los parámetros de Argon2id no son válidos.");
	if (!deBase64url(d.claveDeAcceso, 32)) throw new Fallo(400, "Falta la clave de acceso o no es válida.");
	if (!deBase64url(d.posesion, 32)) throw new Fallo(400, "Falta la prueba de posesión o no es válida.");

	const ahora = Date.now();
	// **El intento se apunta en la misma sentencia que lo lee** (revisión del
	// 2026-09-23). Antes eran tres viajes a D1 —leer, comparar, sumar el intento—, y
	// con peticiones a la vez toda la que leyera antes de la quinta escritura se
	// evaluaba igual: los cinco intentos de un código de seis cifras dejaban de ser
	// cinco. Con `RETURNING`, quien no cabe no lee el código.
	const alta = await env.BD.prepare(
		"UPDATE altas SET intentos = intentos + 1 WHERE correo = ? AND intentos < ? AND caduca >= ? RETURNING codigo",
	)
		.bind(c, INTENTOS_DE_ALTA, ahora)
		.first<{ codigo: string }>();
	const limpio = typeof d.codigo === "string" ? d.codigo.replace(/\s/g, "") : "";
	const calculado = await huellaDeAlta(env, c, limpio);
	const vale =
		!!alta &&
		/^\d{6}$/.test(limpio) &&
		iguales(new TextEncoder().encode(calculado), new TextEncoder().encode(alta.codigo));
	if (!vale) {
		throw new Fallo(401, "El código no es correcto o ha caducado. Pide otro.");
	}

	const ip = await claveDeIP(p, env);
	await contar(env, `altas:${ip}`, tope(env.TOPE_ALTAS_IP_DIA, 3), "Se han creado demasiadas cuentas desde aquí hoy.");
	await contar(env, "altas:todas", tope(env.TOPE_ALTAS_DIA, 200), "Hoy no se pueden crear más cuentas. Prueba mañana.");

	const cuenta = aHex(azar(16));
	const previa = await cuentaDeCorreo(env, c);
	if (previa) {
		// Una fila huérfana —la de una cuenta cuyo borrado no llegó a D1— se reutiliza;
		// una cuenta viva, no.
		if (await objeto(env, previa).prelogin()) throw new Fallo(409, "Ya hay una cuenta con este correo.");
		await env.BD.prepare("UPDATE cuentas SET cuenta = ?, creada = ? WHERE correo = ?").bind(cuenta, ahora, c).run();
	} else {
		await env.BD.prepare("INSERT INTO cuentas (correo, cuenta, creada) VALUES (?, ?, ?)").bind(c, cuenta, ahora).run();
	}

	const creada = await objeto(env, cuenta).crear({
		cuenta,
		correo: c,
		sal: d.sal as string,
		argon2: d.argon2,
		claveDeAcceso: d.claveDeAcceso as string,
		posesion: d.posesion as string,
		llaves: d.llaves,
		dispositivo: nombreDeEquipo(d.dispositivo),
		confiar: d.confiar === true,
	});
	if (!creada.ok) {
		await env.BD.prepare("DELETE FROM cuentas WHERE cuenta = ?").bind(cuenta).run();
		throw new Fallo(creada.estado, creada.error);
	}
	await env.BD.prepare("DELETE FROM altas WHERE correo = ?").bind(c).run();
	// La constancia del alta va **después** y sin bloquear la respuesta: que el correo
	// falle no puede dejar a medias una cuenta que ya existe.
	ctx.waitUntil(mandar(env, cartas.cuentaCreada(c)));
	return json(201, { cuenta, ...creada.datos });
}

// ================================================================ compartir

async function publicarLlaves(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	const d = await leerJSON(p);
	return json(200, abrir(await objeto(env, cuenta).ponerLlaves(token, d.llaves)));
}

/**
 * Las llaves de un correo, **sin decir si ese correo tiene cuenta**.
 *
 * Es el mismo problema que la pre-entrada y se resuelve igual: si no hay cuenta
 * —o todavía no ha publicado llaves— se devuelven unas **inventadas pero fijas**,
 * derivadas del correo con `SECRETO_PRELOGIN`. Así, preguntar por direcciones no
 * dice cuáles están en Esfinge, que es justo lo que el resto del servidor cuida.
 *
 * Lo que cuesta: quien mande a una dirección sin cuenta **cifra hacia una llave
 * que no abre nadie**. Hoy ese envío se pierde en silencio, y por eso la B3 —las
 * invitaciones— va justo detrás.
 */
async function llavesDe(p: Request, env: Env): Promise<Response> {
	await frenar(env.FRENO_ENTRAR, p, env);
	sesionDe(p); // solo quien tiene cuenta puede preguntar
	const d = await leerJSON(p);
	const c = correoValido(d.correo);
	const cuenta = await cuentaDeCorreo(env, c);
	const suyas = cuenta ? await objeto(env, cuenta).llaves() : null;
	if (suyas) return json(200, { llaves: suyas });

	const cifrado = (await hmac(env.SECRETO_PRELOGIN, `llave-cifrado|${c}`)).slice(0, 32);
	const firma = (await hmac(env.SECRETO_PRELOGIN, `llave-firma|${c}`)).slice(0, 32);
	return json(200, {
		llaves: { suite: SUITE_POR_DEFECTO, cifrado: aBase64url(cifrado), firma: aBase64url(firma) },
	});
}

/**
 * Deja un sobre en el buzón de quien lo recibe.
 *
 * **Contesta lo mismo exista o no** esa cuenta, por lo mismo de arriba. Si no
 * existe, hoy el sobre se tira: el emisor ya lo cifró hacia unas llaves que no
 * abren nada, así que guardarlo no serviría para nada.
 */
async function mandarEnvio(p: Request, env: Env): Promise<Response> {
	const { cuenta: mia } = sesionDe(p);
	await frenar(env.FRENO_ENTRAR, p, env);
	const d = await leerJSON(p);
	const c = correoValido(d.para);
	if (typeof d.sobre !== "object" || d.sobre === null) throw new Fallo(400, "Falta el envío.");

	const ip = await claveDeIP(p, env);
	await contar(env, `envios:${ip}`, tope(env.TOPE_ENVIOS_IP_DIA, 50), "Se han mandado demasiadas cosas desde aquí hoy.");
	await contar(env, `envios-cuenta:${mia}`, tope(env.TOPE_ENVIOS_DIA, 50), "Has mandado demasiadas cosas hoy. Prueba mañana.");

	const destino = await cuentaDeCorreo(env, c);
	if (destino) {
		const hecho = await objeto(env, destino).recibir(JSON.stringify(d.sobre));
		// Un buzón lleno o un sobre demasiado grande sí se dicen: no delatan nada
		// —cualquiera puede llenar el suyo— y callarlos dejaría al que manda
		// creyendo que ha llegado.
		if (!hecho.ok && hecho.estado !== 404) throw new Fallo(hecho.estado, hecho.error);
	}
	return json(202, {});
}

async function verBuzon(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	return json(200, abrir(await objeto(env, cuenta).buzon(token)));
}

async function tirarDelBuzon(p: Request, env: Env, id: string): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	return json(200, abrir(await objeto(env, cuenta).tirarDelBuzon(token, id)));
}

// ================================================================ entrar

async function entrar(p: Request, env: Env): Promise<Response> {
	await frenar(env.FRENO_ENTRAR, p, env);
	const d = await leerJSON(p);
	const c = correoValido(d.correo);
	const cuenta = await cuentaDeCorreo(env, c);
	// Suspendida, no se entra. Se dice claro y con a quién escribir: un 401 genérico
	// dejaría a esa persona probando su contraseña sin entender qué pasa.
	if (cuenta) await noSuspendida(env, cuenta);
	const hecho = abrir(
		await objeto(env, cuenta ?? SIN_CUENTA).entrar({
			claveDeAcceso: d.claveDeAcceso,
			confianza: d.confianza,
			dispositivo: d.dispositivo,
		}),
	);
	if ("sesion" in hecho) return json(200, hecho);
	await mandarOFallar(env, cartas.codigoDeEntrada(hecho.correo, hecho.codigo, nombreDeEquipo(d.dispositivo)));
	return json(202, { reto: hecho.reto });
}

async function confirmarEntrada(p: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
	await frenar(env.FRENO_ENTRAR, p, env);
	const d = await leerJSON(p);
	const cuenta = cuentaDeReto(d.reto);
	if (!cuenta) throw new Fallo(401, "El código no es correcto o ha caducado. Pide otro.");
	await noSuspendida(env, cuenta);
	const hecho = abrir(
		await objeto(env, cuenta).confirmar({ reto: d.reto as string, codigo: d.codigo, confiar: d.confiar === true }),
	);
	ctx.waitUntil(mandar(env, cartas.equipoNuevo(hecho.correo, hecho.nombre)));
	return json(200, { sesion: hecho.sesion, dispositivo: hecho.dispositivo, confianza: hecho.confianza });
}

async function cerrarSesion(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	abrir(await objeto(env, cuenta).cerrarSesion(token));
	return json(200, {});
}

// ================================================================ la bóveda

async function leerBoveda(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	const hecho = abrir(await objeto(env, cuenta).leer(token, etiqueta(p.headers.get("If-None-Match"))));
	const cabeceras = { ETag: `"${hecho.version}"`, "Cache-Control": "no-store" };
	if (!hecho.datos) return new Response(null, { status: 304, headers: cabeceras });
	return new Response(hecho.datos, {
		status: 200,
		headers: { ...cabeceras, "Content-Type": "application/json; charset=utf-8" },
	});
}

async function escribirBoveda(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	// **Suspendida se puede leer y exportar, pero no subir.** Las condiciones
	// prometen un plazo para llevarse los datos, y eso exige que bajar siga yendo.
	await noSuspendida(env, cuenta);
	const siCoincide = etiqueta(p.headers.get("If-Match"));
	if (siCoincide === null) throw new Fallo(428, "Falta decir sobre qué versión se escribe (If-Match).");
	const datos = await leerBytes(p, TAMANO_MAXIMO);
	const hecho = abrir(await objeto(env, cuenta).escribir(token, siCoincide, datos));
	return json(200, hecho, { ETag: `"${hecho.version}"` });
}

async function listarVersiones(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	return json(200, abrir(await objeto(env, cuenta).versiones(token)));
}

async function leerVersion(p: Request, env: Env, version: number): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	const datos = abrir(await objeto(env, cuenta).unaVersion(token, version));
	return new Response(datos, {
		headers: { "Content-Type": "application/json; charset=utf-8", "Cache-Control": "no-store" },
	});
}

async function cambiarClave(p: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	const d = await leerJSON(p, TAMANO_MAXIMO * 2);
	if (!deBase64url(d.sal, 16)) throw new Fallo(400, "Falta la sal o no es válida.");
	if (!argon2Valido(d.argon2)) throw new Fallo(400, "Los parámetros de Argon2id no son válidos.");
	if (!deBase64url(d.claveDeAcceso, 32)) throw new Fallo(400, "Falta la clave de acceso o no es válida.");
	if (!Number.isInteger(d.version)) throw new Fallo(400, "Falta la versión sobre la que se escribe.");
	if (typeof d.documento !== "string") throw new Fallo(400, "Falta la bóveda con la contraseña nueva.");
	const documento = new TextEncoder().encode(d.documento);
	const hecho = abrir(
		await objeto(env, cuenta).cambiarClave(token, {
			posesion: d.posesion,
			sal: d.sal as string,
			argon2: d.argon2,
			claveDeAcceso: d.claveDeAcceso as string,
			siCoincide: d.version as number,
			documento: documento.buffer as ArrayBuffer, // encode() da un búfer propio y justo
		}),
	);
	ctx.waitUntil(mandar(env, cartas.claveCambiada(hecho.correo)));
	return json(200, { version: hecho.version, sesion: hecho.sesion }, { ETag: `"${hecho.version}"` });
}

// ================================================================ recuperar

async function empezarRecuperacion(p: Request, env: Env): Promise<Response> {
	await frenar(env.FRENO_ALTAS, p, env);
	const { correo } = await leerJSON(p);
	const c = correoValido(correo);
	const cuenta = await cuentaDeCorreo(env, c);
	// Siempre la misma respuesta: si hay cuenta, el código llega al buzón; si no, nada.
	if (cuenta) {
		const r = await objeto(env, cuenta).retoRecuperacion();
		if (r?.ok) await mandarOFallar(env, cartas.codigoDeRecuperacion(r.datos.correo, r.datos.codigo));
	}
	return json(202, {});
}

async function comprobarRecuperacion(p: Request, env: Env): Promise<Response> {
	await frenar(env.FRENO_ENTRAR, p, env);
	const d = await leerJSON(p);
	const c = correoValido(d.correo);
	const cuenta = await cuentaDeCorreo(env, c);
	return json(200, abrir(await objeto(env, cuenta ?? SIN_CUENTA).comprobarRecuperacion(d.codigo)));
}

async function terminarRecuperacion(p: Request, env: Env): Promise<Response> {
	await frenar(env.FRENO_ENTRAR, p, env);
	const d = await leerJSON(p);
	const cuenta = cuentaDeReto(d.reto);
	if (!cuenta) throw new Fallo(401, "La recuperación ha caducado. Empieza otra vez.");
	return json(200, abrir(await objeto(env, cuenta).terminarRecuperacion(d.reto as string, d.posesion, d.dispositivo)));
}

// ================================================================ equipos, borrar y exportar

async function listarEquipos(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	return json(200, abrir(await objeto(env, cuenta).dispositivos(token)));
}

async function olvidarEquipo(p: Request, env: Env, id: string): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	abrir(await objeto(env, cuenta).olvidarDispositivo(token, id));
	return json(200, {});
}

async function pedirBorrado(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	const hecho = abrir(await objeto(env, cuenta).retoBorrado(token));
	await mandarOFallar(env, cartas.codigoDeBorrado(hecho.correo, hecho.codigo));
	return json(202, { reto: hecho.reto });
}

async function borrarCuenta(p: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	const d = await leerJSON(p);
	if (typeof d.reto !== "string") throw new Fallo(400, "Falta el código de borrado.");
	const hecho = abrir(
		await objeto(env, cuenta).borrar(token, { claveDeAcceso: d.claveDeAcceso, reto: d.reto, codigo: d.codigo }),
	);
	await env.BD.prepare("DELETE FROM cuentas WHERE cuenta = ?").bind(cuenta).run();
	ctx.waitUntil(mandar(env, cartas.cuentaBorrada(hecho.correo)));
	return json(200, {});
}

async function exportar(p: Request, env: Env): Promise<Response> {
	const { token, cuenta } = sesionDe(p);
	const datos = abrir(await objeto(env, cuenta).exportar(token));
	return json(200, datos, { "Content-Disposition": 'attachment; filename="esfinge-cuenta.json"' });
}

async function buzonDePruebas(env: Env, correo: string | null): Promise<Response> {
	const c = normalizarCorreo(correo);
	if (!c) throw new Fallo(400, "Falta el correo.");
	const { results } = await env.BD.prepare(
		"SELECT asunto, cuerpo, momento FROM buzon_pruebas WHERE correo = ? ORDER BY momento DESC, rowid DESC",
	)
		.bind(c)
		.all();
	return json(200, { mensajes: results });
}

// ================================================================ piezas

/**
 * Sin secretos, el servidor no contesta nada.
 *
 * Un secreto que falta no da error en JavaScript: da `undefined`, y
 * `encode(undefined)` es la palabra «undefined». Los verificadores quedarían
 * firmados con una clave que cualquiera conoce, y todo funcionaría. Es el fallo
 * mudo de siempre —el permiso `storage` que costó cinco versiones— con peores
 * consecuencias, así que se comprueba en cada petición y antes que nada.
 */
function configuracionCompleta(env: Env) {
	const corto = (s: unknown) => typeof s !== "string" || s.length < 32;
	const falta =
		corto(env.PIMIENTA) ||
		corto(env.SECRETO_PRELOGIN) ||
		env.PIMIENTA === env.SECRETO_PRELOGIN ||
		(env.ENTORNO !== "pruebas" && !env.RESEND_API_KEY);
	if (falta) throw new Fallo(503, "El servidor no está configurado todavía.");
}

/**
 * El objeto de la cuenta, **en la UE**.
 *
 * La jurisdicción sale de `JURISDICCION` y no va escrita aquí porque el motor local
 * —el de las pruebas y el de `wrangler dev`— no la implementa y revienta al
 * pedirla. Los dos Workers de verdad la llevan a `eu` en `wrangler.jsonc`, y hay una
 * prueba que lee ese fichero y se pone roja si alguien la quita.
 */
function objeto(env: Env, cuenta: string) {
	if (env.JURISDICCION !== "eu" && env.JURISDICCION !== "") throw new Error("JURISDICCION tiene que ser «eu»");
	const ns = env.JURISDICCION === "eu" ? env.CUENTAS.jurisdiction("eu") : env.CUENTAS;
	return ns.get(ns.idFromName(cuenta));
}

/**
 * Adonde van las preguntas por un correo que no tiene cuenta: un objeto vacío que
 * contesta «no» haciendo el mismo trabajo, para que el tiempo de respuesta no
 * distinga un caso del otro.
 */
const SIN_CUENTA = "0".repeat(32);

function modoDeRegistro(env: Env): "cerrado" | "lista" | "abierto" {
	return env.REGISTRO === "abierto" || env.REGISTRO === "lista" ? env.REGISTRO : "cerrado";
}

async function puedeDarseDeAlta(env: Env, correo: string) {
	const modo = modoDeRegistro(env);
	if (modo === "cerrado") throw new Fallo(403, "El registro está cerrado.");
	if (modo === "abierto") return;
	const dominio = correo.slice(correo.lastIndexOf("@"));
	const admitido = await env.BD.prepare("SELECT 1 FROM admision WHERE patron = ? OR patron = ?")
		.bind(correo, dominio)
		.first();
	if (!admitido) throw new Fallo(403, "El registro todavía no está abierto para este correo.");
}

async function cuentaDeCorreo(env: Env, correo: string): Promise<string | null> {
	const fila = await env.BD.prepare("SELECT cuenta FROM cuentas WHERE correo = ?").bind(correo).first<{ cuenta: string }>();
	return fila?.cuenta ?? null;
}

/**
 * Lo que las condiciones de uso llaman cerrar una cuenta, aquí (ADR 0042).
 *
 * La marca vive en D1 y **se pone a mano con `wrangler d1 execute`**, como la lista
 * de admisión: así no hace falta ninguna ruta de administración en el servidor, que
 * sería una puerta nueva a un sitio donde no queremos puertas.
 *
 * **Suspendida no es borrada**: no se puede entrar ni subir, pero **sí leer y
 * exportar**, porque las condiciones prometen un plazo para llevarse los datos y
 * esa promesa tiene que poder cumplirse.
 */
const SUSPENDIDA = "Esta cuenta está suspendida. Escríbenos a info@webcafeina.com.";

async function estaSuspendida(env: Env, cuenta: string): Promise<boolean> {
	const fila = await env.BD.prepare("SELECT suspendida FROM cuentas WHERE cuenta = ?")
		.bind(cuenta)
		.first<{ suspendida: number }>();
	return (fila?.suspendida ?? 0) !== 0;
}

async function noSuspendida(env: Env, cuenta: string): Promise<void> {
	if (await estaSuspendida(env, cuenta)) throw new Fallo(403, SUSPENDIDA);
}

function correoValido(correo: unknown): string {
	const c = normalizarCorreo(correo);
	if (!c) throw new Fallo(400, "Ese correo no parece válido.");
	return c;
}

async function huellaDeAlta(env: Env, correo: string, codigo: string): Promise<string> {
	return aHex(await hmac(env.PIMIENTA, `alta|${correo}|${codigo}`));
}

/** La IP, nunca en claro: un HMAC con un secreto del servidor. */
async function claveDeIP(p: Request, env: Env): Promise<string> {
	const ip = p.headers.get("CF-Connecting-IP") ?? "sin-ip";
	return aHex(await hmac(env.SECRETO_PRELOGIN, `ip|${ip}`)).slice(0, 32);
}

async function frenar(freno: RateLimit, p: Request, env: Env) {
	const { success } = await freno.limit({ key: await claveDeIP(p, env) });
	if (!success) throw new Fallo(429, "Demasiadas peticiones seguidas. Espera un minuto.");
}

/** Un contador por día en D1. Se cuenta antes de comprobar para que dos a la vez no pasen los dos. */
async function contar(env: Env, clave: string, tope: number, mensaje: string) {
	const dia = new Date().toISOString().slice(0, 10);
	const fila = await env.BD.prepare(
		"INSERT INTO contadores (clave, dia, n) VALUES (?, ?, 1) ON CONFLICT (clave, dia) DO UPDATE SET n = n + 1 RETURNING n",
	)
		.bind(clave, dia)
		.first<{ n: number }>();
	if ((fila?.n ?? 0) > tope) throw new Fallo(429, mensaje);
	// De paso, lo de hace más de dos días ya no cuenta para nada.
	const antes = new Date(Date.now() - 2 * 24 * HORA).toISOString().slice(0, 10);
	await env.BD.prepare("DELETE FROM contadores WHERE dia < ?").bind(antes).run();
}

async function mandar(env: Env, c: Carta): Promise<Entregado> {
	return carteroPara(env).mandar(c);
}

async function mandarOFallar(env: Env, c: Carta) {
	const como = await mandar(env, c);
	if (como === "cupo") {
		throw new Fallo(503, "Hoy no se pueden mandar más correos. Vuelve a intentarlo mañana.");
	}
	if (como !== "ok") {
		throw new Fallo(502, "No se ha podido mandar el correo. Prueba otra vez en un momento.");
	}
}

function sesionDe(p: Request): { token: string; cuenta: string } {
	const cabecera = p.headers.get("Authorization") ?? "";
	const token = cabecera.startsWith("Bearer ") ? cabecera.slice(7).trim() : "";
	const cuenta = cuentaDeSesion(token);
	if (!cuenta) throw new Fallo(401, SESION_CADUCADA);
	return { token, cuenta };
}

/** `"17"` → 17. Cualquier otra cosa, null. */
function etiqueta(v: string | null): number | null {
	const m = /^(?:W\/)?"?(\d{1,15})"?$/.exec(v?.trim() ?? "");
	return m ? Number(m[1]) : null;
}

function abrir<T>(r: Resultado<T> & { version?: number }): T {
	if (r.ok) return r.datos;
	throw new Fallo(r.estado, r.error, r.version !== undefined ? { version: r.version } : {});
}

async function leerBytes(p: Request, limite: number): Promise<ArrayBuffer> {
	const declarado = Number(p.headers.get("Content-Length") ?? "0");
	if (declarado > limite) throw new Fallo(413, "Es demasiado grande.");
	const datos = await p.arrayBuffer();
	if (datos.byteLength > limite) throw new Fallo(413, "Es demasiado grande.");
	return datos;
}

async function leerJSON(p: Request, limite = LIMITE_JSON): Promise<Record<string, unknown>> {
	const datos = await leerBytes(p, limite);
	try {
		const v = JSON.parse(new TextDecoder().decode(datos));
		if (typeof v === "object" && v !== null && !Array.isArray(v)) return v as Record<string, unknown>;
	} catch {
		// cae abajo
	}
	throw new Fallo(400, "La petición no es un JSON válido.");
}

function json(estado: number, cuerpo: unknown, cabeceras: Record<string, string> = {}): Response {
	return new Response(JSON.stringify(cuerpo), {
		status: estado,
		headers: { "Content-Type": "application/json; charset=utf-8", "Cache-Control": "no-store", ...cabeceras },
	});
}
