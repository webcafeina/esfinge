import { describe, expect, it } from "vitest";
import { cartas } from "../src/correo";
import {
	asunto,
	base64,
	CarteroSmtp,
	componer,
	doblarElPunto,
	formaDeLasCredenciales,
	huellaDeLaClave,
	Lector,
	queHaPasado,
	soloLaDireccion,
	type Conexion,
} from "../src/smtp";

/**
 * El transporte de correo (ADR 0045).
 *
 * **Lo que aquí no se puede probar, y hay que decirlo:** el transporte de verdad
 * —TCP, TLS, el apretón con Google y la autenticación— no corre en ningún sitio
 * automático. Eso solo lo dice una sesión a mano contra el Worker de pruebas.
 *
 * Lo que sí se puede, y es la razón de que el cliente esté escrito a mano en vez de
 * traído de fuera, son las dos mitades por separado: **armar el mensaje es una
 * función pura** y **el diálogo es una máquina de estados** sobre un socket que se
 * inyecta. Una biblioteca solo se prueba enchufándola a Gmail.
 */

const REMITENTE = "Esfinge <esfinge@webcafeina.com>";
const CUANDO = new Date("2026-09-28T10:00:00Z");
const ID = "11111111-2222-3333-4444-555555555555";

const decodificar = (b64: string) => new TextDecoder().decode(Uint8Array.from(atob(b64), (c) => c.charCodeAt(0)));

describe("armar el mensaje", () => {
	const carta = { para: "ana@ejemplo.com", asunto: "Tu código de Esfinge", texto: "Hola\ncódigo: 123456", html: "<p>Hola</p>" };

	it("lleva las cabeceras que hacen falta y el cuerpo en dos partes", () => {
		const m = componer(carta, REMITENTE, CUANDO, ID);
		expect(m).toContain("From: Esfinge <esfinge@webcafeina.com>");
		expect(m).toContain("To: ana@ejemplo.com");
		expect(m).toContain("Date: Mon, 28 Sep 2026 10:00:00 +0000");
		expect(m).toContain(`Message-ID: <${ID}@webcafeina.com>`);
		expect(m).toContain("MIME-Version: 1.0");
		// Que ningún «estoy de vacaciones» conteste a un correo con un código dentro.
		expect(m).toContain("Auto-Submitted: auto-generated");
		expect(m).toContain('Content-Type: multipart/alternative; boundary="=_esfinge_' + ID + '"');
		expect(m).toContain("Content-Type: text/plain; charset=UTF-8");
		expect(m).toContain("Content-Type: text/html; charset=UTF-8");
		expect(m.endsWith(`--=_esfinge_${ID}--\r\n`)).toBe(true);
	});

	it("el texto y el html sobreviven a la ida y vuelta de base64", () => {
		const m = componer(carta, REMITENTE, CUANDO, ID);
		const partes = m.split(`--=_esfinge_${ID}`).slice(1, 3);
		const cuerpos = partes.map((p) => decodificar(p.split("\r\n\r\n")[1].replace(/\r\n/g, "").replace(/--$/, "")));
		expect(cuerpos[0]).toBe(carta.texto);
		expect(cuerpos[1]).toBe(carta.html);
	});

	it("sin html va una sola parte", () => {
		const m = componer({ para: "a@b.c", asunto: "Hola", texto: "Solo texto" }, REMITENTE, CUANDO, ID);
		expect(m.match(/Content-Type: text\//g)).toHaveLength(1);
		expect(m).not.toContain("text/html");
	});

	/**
	 * **Ningún salto suelto y ninguna línea larguísima.** El HTML de la maqueta sale
	 * con líneas de estilos en línea que se pasan de los 998 octetos que permite
	 * SMTP, y un servidor que las corte rompe el correo. Va en base64 justamente por
	 * esto, y esta prueba lo mira **con una carta de verdad**, no con una inventada.
	 */
	it("con el HTML de verdad, ninguna línea se pasa de 998 ni hay saltos sueltos", () => {
		const m = componer(cartas.invitacion("ana@ejemplo.com", "luis@webcafeina.com"), REMITENTE, CUANDO, ID);
		expect(m.replace(/\r\n/g, "")).not.toContain("\n");
		for (const linea of m.split("\r\n")) {
			expect(new TextEncoder().encode(linea).length).toBeLessThanOrEqual(998);
		}
	});

	/**
	 * **Se prueba la función, no `componer`.** Con base64 ninguna línea puede empezar
	 * por un punto, así que a través de `componer` esta prueba salía verde **con el
	 * escape quitado** — o sea, no probaba nada. Se vio mutando el código, que es
	 * para lo que sirve.
	 */
	it("dobla el punto que empieza una línea", () => {
		expect(doblarElPunto("uno\r\n.dos\r\ntres")).toBe("uno\r\n..dos\r\ntres");
		expect(doblarElPunto("uno\r\ndos")).toBe("uno\r\ndos");
	});
});

describe("la forma de las credenciales", () => {
	/**
	 * Lo único que de verdad importa de esta función: **la contraseña no sale**. Lo
	 * demás son comodidades para quien la está configurando; esto es la razón de que
	 * exista una función y no un `${clave}` escrito a mano en el sitio del error.
	 */
	it("no enseña la contraseña, ni entera ni en trozos", async () => {
		const clave = "abcdefghijklmnop";
		const forma = await formaDeLasCredenciales("esfinge@webcafeina.com", clave);
		expect(forma).not.toContain(clave);
		for (let i = 0; i + 4 <= clave.length; i++) expect(forma).not.toContain(clave.slice(i, i + 4));
	});

	it("dice cuántos caracteres tiene y quién es el usuario", async () => {
		expect(await formaDeLasCredenciales("esfinge@webcafeina.com", "abcdefghijklmnop")).toBe(
			" · usuario «esfinge@webcafeina.com», clave de 16 caracteres, huella f39dac6c",
		);
	});

	/**
	 * **La huella es la misma que saca `shasum` en un Mac**, que es de donde viene la
	 * otra mitad de la comparación. Si esto se calculara de otra forma —con sal, o
	 * sobre otra codificación— la comparación no diría nada y nadie se daría cuenta.
	 */
	it("saca los primeros cuatro bytes del SHA-256, como shasum", async () => {
		// printf %s hola | shasum -a 256  →  b221d9db…
		expect(await huellaDeLaClave("hola")).toBe("b221d9db");
	});

	it("distingue dos contraseñas de aplicación, que miden las dos dieciséis", async () => {
		expect(await huellaDeLaClave("abcdefghijklmnop")).not.toBe(await huellaDeLaClave("abcdefghijklmnoq"));
	});

	/**
	 * **El caso para el que se escribió.** Google enseña las contraseñas de
	 * aplicación en grupos de cuatro y el portapapeles se lleva los espacios; lo que
	 * sale de ahí es un `535` idéntico al de una contraseña equivocada.
	 */
	it("avisa de los espacios, que es el error que se comete", async () => {
		expect(await formaDeLasCredenciales("x@y.com", "abcd efgh ijkl mnop")).toContain("y lleva espacios");
		expect(await formaDeLasCredenciales("x@y.com", "abcdefghijklmnop\n")).toContain("y lleva espacios");
	});
});

describe("el asunto", () => {
	it("si es ASCII puro no se toca, para que se pueda leer", () => {
		expect(asunto("Your Esfinge code")).toBe("Your Esfinge code");
	});

	it("con acentos se codifica y vuelve a decir lo mismo", () => {
		const s = "Tu código para entrar en Esfinge";
		const codificado = asunto(s);
		expect(codificado).toMatch(/^=\?UTF-8\?B\?/);
		expect(juntar(codificado)).toBe(s);
	});

	/**
	 * **Cada palabra codificada cabe en 75 caracteres, y se parte por caracteres.**
	 * Partir por bytes cortaría un UTF-8 por la mitad y dejaría un rombo negro en el
	 * asunto de un correo que dice «tu código».
	 */
	it("uno largo se parte sin romper ningún carácter", () => {
		const s = "Código para recuperar tu cuenta de Esfinge en el ordenador de la oficina de Málaga";
		const codificado = asunto(s);
		expect(codificado).toContain("\r\n ");
		for (const palabra of codificado.split("\r\n ")) {
			expect(palabra.length).toBeLessThanOrEqual(75);
		}
		expect(juntar(codificado)).toBe(s);
	});

	function juntar(codificado: string) {
		return codificado
			.split("\r\n ")
			.map((p) => decodificar(/=\?UTF-8\?B\?(.*)\?=/.exec(p)![1]))
			.join("");
	}
});

describe("leer lo que contesta el servidor", () => {
	it("junta una respuesta de varias líneas y deja lo que venga detrás", async () => {
		const lector = new Lector(unFlujo("250-mx.google.com\r\n250-SIZE\r\n250 AUTH LOGIN PLAIN\r\n235 ok\r\n"));
		const primera = await lector.respuesta();
		expect(primera.codigo).toBe(250);
		expect(primera.texto).toContain("AUTH LOGIN PLAIN");
		expect((await lector.respuesta()).codigo).toBe(235);
	});

	it("una respuesta partida en dos trozos se espera entera", async () => {
		const lector = new Lector(unFlujo("250-mx.goo", "gle.com\r\n250 ok\r\n"));
		expect((await lector.respuesta()).codigo).toBe(250);
	});

	it("si el servidor cierra a media conversación, se sabe", async () => {
		const lector = new Lector(unFlujo("250-a medias\r\n"));
		await expect(lector.respuesta()).rejects.toThrow(/cerrado/);
	});
});

describe("qué significa lo que contesta", () => {
	/**
	 * Las dos reglas por separado. Juntas —un texto que casa con las dos— tapaban que
	 * una de ellas sobrara o faltara: se vio mutando.
	 */
	it("el límite del día es «cupo», que es mañana", () => {
		// Solo por el estado ampliado, sin ninguna palabra que delate.
		expect(queHaPasado({ codigo: 550, texto: "550 5.4.5 Rechazado" })).toBe("cupo");
		expect(queHaPasado({ codigo: 452, texto: "452 4.4.5 Rechazado" })).toBe("cupo");
		// Y solo por el texto, con un estado ampliado que no dice nada.
		expect(queHaPasado({ codigo: 550, texto: "550 5.7.0 Daily SMTP relay limit exceeded" })).toBe("cupo");
	});

	/**
	 * **El `421 4.7.0` no es cupo**: es de minutos, y ahí «prueba otra vez en un
	 * momento» sí es verdad. Es el criterio de la ADR 0041 aplicado al revés.
	 */
	it("el estrangulamiento de un rato y la autenticación son «fallo»", () => {
		expect(queHaPasado({ codigo: 421, texto: "421 4.7.0 Try again later" })).toBe("fallo");
		expect(queHaPasado({ codigo: 535, texto: "535 5.7.8 Username and Password not accepted" })).toBe("fallo");
	});
});

describe("el diálogo entero", () => {
	const carta = { para: "ana@ejemplo.com", asunto: "Tu código", texto: "123456" };
	const ajustes = {
		host: "smtp-relay.gmail.com",
		puerto: 465,
		ehlo: "esfinge-cuentas.webcafeina.com",
		usuario: "quien@webcafeina.com",
		clave: "la-de-aplicacion",
		remitente: REMITENTE,
	};

	it("el camino feliz manda el correo y se despide", async () => {
		const s = servidorFalso(["220 mx listo", "250-mx\r\n250 AUTH PLAIN LOGIN", "235 ok", "250 ok", "250 ok", "354 dale", "250 aceptado"]);
		expect((await new CarteroSmtp(ajustes, s.conectar).mandar(carta)).entregado).toBe("ok");

		expect(s.ordenes[0]).toBe("EHLO esfinge-cuentas.webcafeina.com");
		expect(s.ordenes[1]).toBe(`AUTH PLAIN ${base64(new TextEncoder().encode("\0quien@webcafeina.com\0la-de-aplicacion"))}`);
		expect(s.ordenes[2]).toBe("MAIL FROM:<esfinge@webcafeina.com>");
		expect(s.ordenes[3]).toBe("RCPT TO:<ana@ejemplo.com>");
		expect(s.ordenes[4]).toBe("DATA");
		expect(s.ordenes[5]).toBe("QUIT");
		// Las cabeceras van **dentro** del DATA, no antes. Y el asunto lleva acento,
		// así que va codificado: pedir «Subject: Tu código» aquí sería pedir que la
		// otra prueba estuviera mal.
		expect(s.cuerpo).toContain("From: Esfinge <esfinge@webcafeina.com>");
		expect(s.cuerpo).toContain("Subject: =?UTF-8?B?");
		expect(s.cuerpo).toContain("Content-Type: multipart/alternative");
		expect(s.cerrado).toBe(true);
	});

	it("si solo anuncia LOGIN, se autentica con LOGIN", async () => {
		const s = servidorFalso(["220 mx", "250-mx\r\n250 AUTH LOGIN", "334 usuario", "334 clave", "235 ok", "250 ok", "250 ok", "354 dale", "250 aceptado"]);
		expect((await new CarteroSmtp(ajustes, s.conectar).mandar(carta)).entregado).toBe("ok");
		expect(s.ordenes[1]).toBe("AUTH LOGIN");
		expect(decodificar(s.ordenes[2])).toBe("quien@webcafeina.com");
		expect(decodificar(s.ordenes[3])).toBe("la-de-aplicacion");
	});

	/**
	 * Para poder preguntarle a un servidor que dice que no si dice que no a las dos
	 * formas. Un `535` con las dos no puede ser de cómo se manda la credencial, y eso
	 * es lo que separa arreglar el cliente de cambiar por dónde sale la conexión.
	 */
	it("forzando LOGIN se usa LOGIN, aunque anuncie PLAIN", async () => {
		const s = servidorFalso(["220 mx", "250-mx\r\n250 AUTH PLAIN LOGIN", "334 usuario", "334 clave", "235 ok", "250 ok", "250 ok", "354 dale", "250 aceptado"]);
		expect((await new CarteroSmtp({ ...ajustes, mecanismo: "login" }, s.conectar).mandar(carta)).entregado).toBe("ok");
		expect(s.ordenes[1]).toBe("AUTH LOGIN");
	});

	it("forzando PLAIN se usa PLAIN, aunque solo anuncie LOGIN", async () => {
		const s = servidorFalso(["220 mx", "250-mx\r\n250 AUTH LOGIN", "235 ok", "250 ok", "250 ok", "354 dale", "250 aceptado"]);
		expect((await new CarteroSmtp({ ...ajustes, mecanismo: "plain" }, s.conectar).mandar(carta)).entregado).toBe("ok");
		expect(s.ordenes[1]).toMatch(/^AUTH PLAIN /);
	});

	it("el límite del día llega como «cupo» hasta arriba", async () => {
		const s = servidorFalso(["220 mx", "250 AUTH PLAIN", "235 ok", "250 ok", "250 ok", "354 dale", "550 5.4.5 Daily SMTP relay limit exceeded"]);
		expect((await new CarteroSmtp(ajustes, s.conectar).mandar(carta)).entregado).toBe("cupo");
		expect(s.cerrado).toBe(true);
	});

	/**
	 * **Y dice en qué paso se cayó.** Sin esto, lo único que queda de un envío que
	 * falla es la palabra «fallo», que es lo mismo que nada — y eso costó la primera
	 * vez que se probó contra Google de verdad.
	 */
	it("una contraseña que no vale es «fallo», dice por qué, y se cierra igual", async () => {
		const s = servidorFalso(["220 mx", "250 AUTH PLAIN", "535 5.7.8 Username and Password not accepted"]);
		const envio = await new CarteroSmtp(ajustes, s.conectar).mandar(carta);
		expect(envio.entregado).toBe("fallo");
		expect(envio.porque).toContain("AUTH PLAIN");
		expect(envio.porque).toContain("535");
		// Y lo que no puede llevar nunca: la contraseña.
		expect(envio.porque).not.toContain(ajustes.clave);
		expect(s.cerrado).toBe(true);
	});

	it("si el servidor se calla, el plazo lo corta", async () => {
		const s = servidorFalso(["220 mx"]); // saluda y no contesta nunca más
		const antes = Date.now();
		expect((await new CarteroSmtp({ ...ajustes, plazo: 150 }, s.conectar).mandar(carta)).entregado).toBe("fallo");
		expect(Date.now() - antes).toBeLessThan(2_000);
		expect(s.cerrado).toBe(true);
	});

	it("si no se puede ni conectar, es «fallo» y no revienta", async () => {
		const cartero = new CarteroSmtp(ajustes, () => {
			throw new Error("ECONNREFUSED");
		});
		expect((await cartero.mandar(carta)).entregado).toBe("fallo");
	});
});

it("saca la dirección de dentro del remitente", () => {
	expect(soloLaDireccion("Esfinge <esfinge@webcafeina.com>")).toBe("esfinge@webcafeina.com");
	expect(soloLaDireccion("esfinge@webcafeina.com")).toBe("esfinge@webcafeina.com");
});

// ------------------------------------------------------------------ ayudas

function unFlujo(...trozos: string[]): ReadableStream<Uint8Array> {
	return new ReadableStream({
		start(c) {
			for (const t of trozos) c.enqueue(new TextEncoder().encode(t));
			c.close();
		},
	});
}

/**
 * Un servidor de SMTP de mentira, con guion.
 *
 * Contesta una línea del guion por cada orden, y **sabe que dentro de `DATA` no hay
 * órdenes** hasta el punto solo: sin eso, el cuerpo del mensaje se leería como un
 * puñado de órdenes y el guion se desharía.
 */
function servidorFalso(guion: string[]) {
	const hacia = new TransformStream<Uint8Array, Uint8Array>();
	const escritor = hacia.writable.getWriter();
	const estado = { ordenes: [] as string[], cuerpo: "", cerrado: false };
	let pendiente = "";
	let enDatos = false;
	let i = 0;

	const contestar = () => {
		const linea = guion[i++];
		if (linea === undefined) return Promise.resolve(); // se calla a propósito
		return escritor.write(new TextEncoder().encode(linea + "\r\n"));
	};
	void contestar(); // el saludo, antes de que el cliente diga nada

	const writable = new WritableStream<Uint8Array>({
		write(trozo) {
			pendiente += new TextDecoder().decode(trozo);
			for (;;) {
				const corte = pendiente.indexOf("\r\n");
				if (corte < 0) break;
				const linea = pendiente.slice(0, corte);
				pendiente = pendiente.slice(corte + 2);
				if (enDatos) {
					if (linea === ".") {
						enDatos = false;
						void contestar();
					} else {
						estado.cuerpo += linea + "\r\n";
					}
					continue;
				}
				estado.ordenes.push(linea);
				if (linea === "DATA") enDatos = true;
				// **Sin esperar a que salga.** Esperando aquí se abrazan los dos: el
				// cliente no termina su escritura hasta que este manejador acabe, y este
				// manejador no acaba hasta que el cliente lea lo que le acabamos de
				// poner. Encolar y seguir es lo que hace un servidor de verdad.
				void contestar();
			}
		},
	});

	const conectar = () =>
		({
			readable: hacia.readable,
			writable,
			close: async () => {
				estado.cerrado = true;
				await escritor.close().catch(() => {});
			},
		}) as Conexion;

	return {
		conectar,
		get ordenes() {
			return estado.ordenes;
		},
		get cuerpo() {
			return estado.cuerpo;
		},
		get cerrado() {
			return estado.cerrado;
		},
	};
}
