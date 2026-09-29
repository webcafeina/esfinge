/**
 * El cartero de Esfinge: recibe un correo ya compuesto y lo entrega.
 *
 * Vive en el VPS porque el servidor de cuentas no puede entregarlo él mismo: Google
 * rechaza la autenticación SMTP cuando la conexión sale de Cloudflare (ADR 0045). La
 * IPv4 del VPS sí está autorizada en el relé de Workspace, así que aquí no hace falta
 * ni usuario ni contraseña contra Google — **el relé autentica por IP**, igual que en
 * Cronos, que es de donde sale toda la configuración de abajo.
 *
 * Lo que este servicio **no** hace, y conviene tenerlo escrito: no compone el correo
 * —eso sigue en el Worker, con sus pruebas y su maqueta—, no guarda nada y no sabe
 * nada de cuentas. Recibe un mensaje, lo revisa y lo entrega.
 */
import { createServer } from "node:http";
import { setDefaultResultOrder } from "node:dns";
import nodemailer from "nodemailer";

import { elSecretoCuadra, laDireccion, Rechazado, revisar, sinDirecciones } from "./relevo.js";

const pedir = (nombre: string): string => {
	const v = process.env[nombre]?.trim();
	// **Sin sus secretos no arranca**, que es la misma regla que el Worker: un secreto
	// que falta es `undefined`, y comparar contra la palabra «undefined» funcionaría
	// sin que nada avisara.
	if (!v) throw new Error(`Falta ${nombre}`);
	return v;
};

const SECRETO = pedir("CARTERO_SECRETO");
const REMITENTE = pedir("SMTP_FROM");
// La dirección sola, para el sobre. Si `SMTP_FROM` no la lleva, no hay nada que
// entregar y es mejor no arrancar que descubrirlo con el primer correo.
const DIRECCION = laDireccion(REMITENTE);
if (DIRECCION === null) throw new Error("SMTP_FROM no lleva una dirección");
const PUERTO = Number(process.env.PUERTO ?? "8791");

// **IPv4 primero, en todo el proceso.** El relé autoriza por la IP pública del VPS,
// que es v4; si la máquina prefiere IPv6, Google contesta «550 5.7.1 Invalid
// credentials for relay» porque esa dirección no está en su lista. Lo aprendió Cronos
// en producción y se reprodujo aquí con `openssl s_client` antes de escribir nada.
setDefaultResultOrder("ipv4first");

const transporte = nodemailer.createTransport({
	host: process.env.SMTP_HOST ?? "smtp-relay.gmail.com",
	port: Number(process.env.SMTP_PORT ?? "587"),
	secure: false,
	requireTLS: true,
	// **El relé exige presentarse con un dominio de verdad.** Con el nombre de máquina
	// del VPS, Google contesta «421 4.7.0» antes de aceptar nada. También de Cronos.
	name: process.env.SMTP_EHLO ?? "esfinge-cuentas.webcafeina.com",
});

/** Lee el cuerpo entero, con tope: sin él, quien tenga el secreto llena la memoria. */
async function cuerpoDe(peticion: NodeJS.ReadableStream): Promise<string> {
	const trozos: Buffer[] = [];
	let n = 0;
	for await (const t of peticion) {
		n += (t as Buffer).length;
		if (n > 1024 * 1024) throw new Rechazado("El cuerpo es demasiado largo.");
		trozos.push(t as Buffer);
	}
	return Buffer.concat(trozos).toString("utf8");
}

const servidor = createServer(async (peticion, respuesta) => {
	const contestar = (estado: number, cuerpo: unknown) => {
		respuesta.writeHead(estado, { "content-type": "application/json" });
		respuesta.end(JSON.stringify(cuerpo));
	};

	try {
		// Para que el contenedor se sepa vivo sin mandar un correo.
		if (peticion.method === "GET" && peticion.url === "/salud") return contestar(200, { salud: "bien" });
		if (peticion.method !== "POST" || peticion.url !== "/entregar") return contestar(404, { error: "No existe." });

		const dado = peticion.headers["x-esfinge-secreto"];
		if (typeof dado !== "string" || !elSecretoCuadra(dado, SECRETO)) return contestar(401, { error: "No." });

		const mensaje = await cuerpoDe(peticion);
		const { para } = revisar(mensaje, REMITENTE);

		// **Se entrega por el sobre, no por las cabeceras.** `envelope` es lo que va en
		// el `MAIL FROM` y el `RCPT TO`; las cabeceras son lo que se lee. Dejando que
		// nodemailer lo dedujera del mensaje, lo que se entrega dejaría de ser lo que
		// se acaba de revisar.
		await transporte.sendMail({ envelope: { from: DIRECCION, to: para }, raw: mensaje });
		contestar(200, { entregado: "ok" });
	} catch (e) {
		if (e instanceof Rechazado) return contestar(400, { error: e.message });
		// El porqué va **al registro**, no a la respuesta: quien llama es el Worker y no
		// puede hacer nada con él, y una respuesta de SMTP puede decir más de lo que
		// hace falta contar. Pero callarlo del todo es lo que costó una tarde entera.
		// **Sin la dirección**: un rechazo de SMTP puede traerla dentro, y esto se
		// registra. Ver `sinDirecciones`.
		console.error(JSON.stringify({ nivel: "error", msg: "no se pudo entregar", porque: sinDirecciones(`${e}`) }));
		// **El cupo del día no es un fallo pasajero**, y decir «prueba en un momento»
		// cuando la verdad es «mañana» es la mitad de la ADR 0041. Google lo dice con
		// un 550 5.4.5, que nodemailer deja leer en `responseCode`.
		const cupo = /5\.4\.5|Daily .*limit exceeded/i.test(`${e}`);
		contestar(cupo ? 429 : 502, { error: cupo ? "cupo" : "fallo" });
	}
});

servidor.listen(PUERTO, () => {
	console.log(JSON.stringify({ nivel: "info", msg: "cartero escuchando", puerto: PUERTO }));
});

// Sin esto, `docker stop` espera diez segundos y mata el proceso a lo bruto.
for (const senal of ["SIGTERM", "SIGINT"] as const) {
	process.on(senal, () => servidor.close(() => process.exit(0)));
}
