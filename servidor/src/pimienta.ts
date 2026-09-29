// La pimienta del servidor, y cómo se cambia sin dejar a nadie fuera.
//
// # Qué es
//
// El servidor **nunca ve la contraseña maestra**. El cliente la pasa por Argon2 con
// la sal que el servidor guarda, y lo que viaja es la **clave de acceso**. El
// servidor tiene que poder decir «sí, ésa es» sin poder fabricarla, así que no la
// guarda: guarda `HMAC(PIMIENTA, "acceso|<clave>")`.
//
// `PIMIENTA` vive **solo en el entorno del Worker**, nunca en la base. Ésa es la
// diferencia con la sal, que está al lado del dato: quien se lleve la base entera se
// lleva los verificadores, y **sin la pimienta no puede probar nada contra ellos**.
//
// # Por qué esto existe
//
// Hasta el 2026-09-29, cambiar `PIMIENTA` **dejaba fuera a todas las cuentas**: los
// verificadores guardados dejaban de cuadrar y nadie podía volver a entrar. Una
// pimienta que no se puede rotar es **un secreto al que no se puede responder**: si
// se filtra, la respuesta correcta es cambiarla ya, y la respuesta real era «no
// puedo». La condición era escribir esto **antes de abrir el registro**, y el
// registro se abrió el 2026-09-23 sin ello ([`../../docs/deuda.md`](../../docs/deuda.md)).
//
// # Cómo funciona
//
// **La versión va dentro de cada verificador** —`2:abc123…`—, no en un campo aparte
// de la cuenta. Y no es un capricho: una cuenta tiene **dos** verificadores, el de
// acceso y el de posesión, y **no migran a la vez**. Al entrar se tiene la clave de
// acceso pero no la de posesión; al sincronizar, al revés. Con una sola marca por
// cuenta, la primera migración mentiría sobre la otra.
//
// Lo que no lleva prefijo se lee como la versión 1, que es como está escrito todo lo
// de antes de esto: **no hay que reescribir nada al desplegar**.
//
// **Solo conviven dos generaciones**, la actual y la anterior. Quien se quede dos por
// detrás no se puede verificar, y eso **no es «contraseña incorrecta»**: es un estado
// roto que hay que decir con otras palabras, porque decirle a alguien que su
// contraseña está mal cuando no lo está es el peor mensaje posible.
//
// # Cómo se rota, y cuándo se puede terminar
//
//  1. `PIMIENTA_<n>` ← uno nuevo (`openssl rand -hex 32`), con `n` el número siguiente.
//  2. `PIMIENTA_VERSION` ← ese mismo `n`.
//
// **Y ya está: no se toca ninguna de las que había.** La versión 1 vive en `PIMIENTA`
// para siempre, y las siguientes en `PIMIENTA_2`, `PIMIENTA_3`… Rotar es **añadir**.
//
// La primera versión de esto decía «mueve el valor de `PIMIENTA` a `PIMIENTA_ANTERIOR`»,
// y era **un paso que nadie puede ejecutar**: un secreto de Cloudflare se escribe y no
// se vuelve a leer. Se descubrió al ir a rotar, no al escribirlo.
//
// A partir de ahí **cada cuenta migra sola en cuanto se usa**: al entrar migra su
// verificador de acceso y en cualquier pasada de la sincronización —cada cinco
// minutos con la aplicación abierta— el de posesión.
//
// Cuándo se puede borrar la anterior **se mira, no se supone**: la tabla
// `cuentas` de D1 guarda la versión menor de cada cuenta, así que
//
//     SELECT pimienta, COUNT(*) FROM cuentas GROUP BY pimienta;
//
// dice cuántas quedan atrás. **Mientras ese número no sea cero, no se rota otra vez**
// ni se borra la anterior: quien se quede dos generaciones por detrás queda fuera.
//
// Borrar la anterior sí se puede sin conocer su valor: es quitar la variable.
import { aHex, hmac } from "./cripto";
import type { Env } from "./protocolo";

/** Lo que había antes de que esto existiera, y lo que se lee sin prefijo. */
export const VERSION_INICIAL = 1;

export function versionActual(env: Env): number {
	const v = Number((env.PIMIENTA_VERSION ?? "").trim());
	return Number.isInteger(v) && v >= VERSION_INICIAL ? v : VERSION_INICIAL;
}

/**
 * La pimienta de una versión, o nulo si esa versión ya no está.
 *
 * Solo contesta a la actual y a la anterior **a propósito**: si contestara a más,
 * bastaría con olvidarse de mirar el contador para dejar cuentas colgadas sin que
 * nada lo dijera.
 */
export function pimientaDe(env: Env, version: number): string | null {
	const actual = versionActual(env);
	if (version !== actual && version !== actual - 1) return null;
	// **La 1 es `PIMIENTA` y no se toca nunca**; de la 2 en adelante, `PIMIENTA_<n>`.
	// Rotar es **añadir** una variable, no mover el valor de una a otra — y eso no es
	// estética: **un secreto de Cloudflare no se puede volver a leer**, así que «copia
	// el valor de aquí a allá» es un paso que nadie puede ejecutar salvo que tuviera
	// una copia guardada aparte. Se descubrió al ir a rotar de verdad.
	if (version === VERSION_INICIAL) {
		// Sin recortar, a propósito: si `PIMIENTA` llevara un espacio, los verificadores
		// de antes se calcularon **con él**, y quitarlo ahora los rompería todos.
		return env.PIMIENTA ? env.PIMIENTA : null;
	}
	// Las nuevas sí se recortan: todavía no hay nada calculado con ellas, así que un
	// espacio pegado al copiar se puede quitar sin romper nada.
	const otra = ((env as unknown as Record<string, string | undefined>)[`PIMIENTA_${version}`] ?? "").trim();
	return otra === "" ? null : otra;
}

/** `2:abc…` → la versión y el resumen. Sin prefijo, la versión 1. */
export function partir(guardado: string): { version: number; resumen: string } {
	const dosPuntos = guardado.indexOf(":");
	if (dosPuntos === -1) return { version: VERSION_INICIAL, resumen: guardado };
	const version = Number(guardado.slice(0, dosPuntos));
	if (!Number.isInteger(version) || version < VERSION_INICIAL) {
		// Un prefijo que no es un número no es nuestro. Se trata como lo que es —algo
		// que no vamos a poder verificar— en vez de colarlo como versión 1.
		return { version: 0, resumen: guardado };
	}
	return { version, resumen: guardado.slice(dosPuntos + 1) };
}

/** Cómo se guarda. El resumen es hexadecimal, así que los dos puntos no ambiguan. */
export function sellar(version: number, resumen: string): string {
	return `${version}:${resumen}`;
}

/**
 * El HMAC de algo con la pimienta de una versión, o nulo si esa versión no está.
 *
 * **Tarda lo mismo cuando no está**: hace el cálculo con la actual y tira el
 * resultado. Sin eso, una cuenta colgada se distinguiría de una contraseña mala por
 * lo que tarda en decir que no.
 */
export async function conLaPimienta(env: Env, version: number, datos: string): Promise<string | null> {
	const p = pimientaDe(env, version);
	// Cuando la versión ya no está se calcula igual, con la de ahora, y se tira: así una
	// cuenta colgada no se distingue de una contraseña mala por lo que tarda en fallar.
	const resultado = aHex(await hmac(p ?? pimientaDe(env, versionActual(env)) ?? "", datos));
	return p === null ? null : resultado;
}

/**
 * Para lo que dura diez minutos: se prueba con la actual y con la anterior, y no se
 * migra nada.
 *
 * Los códigos de seis cifras caducan antes de que a nadie le dé tiempo a migrarlos,
 * así que no llevan versión escrita: se emiten con la actual y, durante el solape de
 * una rotación, se aceptan también los que se emitieron con la anterior.
 */
export async function algunaPimientaDa(env: Env, datos: string, esperado: string): Promise<boolean> {
	const actual = versionActual(env);
	let vale = false;
	// Sin cortocircuito: se prueban las dos siempre, para que acertar con la vieja no
	// tarde distinto que acertar con la nueva.
	for (const v of [actual, actual - 1]) {
		const calculado = await conLaPimienta(env, v, datos);
		if (calculado !== null && igualesHex(calculado, esperado)) vale = true;
	}
	return vale;
}

/** Comparación en tiempo constante de dos cadenas hexadecimales. */
export function igualesHex(a: string, b: string): boolean {
	if (a.length !== b.length) return false;
	let distintos = 0;
	for (let i = 0; i < a.length; i++) distintos |= a.charCodeAt(i) ^ b.charCodeAt(i);
	return distintos === 0;
}
