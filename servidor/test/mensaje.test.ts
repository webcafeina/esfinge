import { describe, expect, it } from "vitest";
import { cartas, CarteroPorElVps } from "../src/correo";
import { asunto, componer, soloLaDireccion } from "../src/mensaje";

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

it("saca la dirección de dentro del remitente", () => {
	expect(soloLaDireccion("Esfinge <esfinge@webcafeina.com>")).toBe("esfinge@webcafeina.com");
	expect(soloLaDireccion("esfinge@webcafeina.com")).toBe("esfinge@webcafeina.com");
});

describe("el cartero del VPS", () => {
	const carta = { para: "quien@ejemplo.com", asunto: "Tu código", texto: "Es 123456." };
	const ajustes = { url: "https://cartero.ejemplo.com/entregar", secreto: "el-secreto", remitente: REMITENTE };

	const doble = (contestar: (p: Request) => Response) => {
		const vistas: Request[] = [];
		const pedir = (async (u: RequestInfo | URL, i?: RequestInit) => {
			const p = new Request(u as RequestInfo, i);
			vistas.push(p);
			return contestar(p);
		}) as unknown as typeof fetch;
		return { pedir, vistas };
	};

	it("manda el mensaje entero, con el secreto en su cabecera", async () => {
		const { pedir, vistas } = doble(() => new Response("{}", { status: 200 }));
		expect((await new CarteroPorElVps(ajustes, pedir).mandar(carta)).entregado).toBe("ok");
		expect(vistas[0].method).toBe("POST");
		expect(vistas[0].url).toBe(ajustes.url);
		expect(vistas[0].headers.get("x-esfinge-secreto")).toBe("el-secreto");
		const cuerpo = await vistas[0].text();
		expect(cuerpo).toContain(`From: ${REMITENTE}`);
		expect(cuerpo).toContain("To: quien@ejemplo.com");
	});

	/**
	 * **«Cupo» no es «fallo»** (ADR 0041): decir «prueba otra vez en un momento»
	 * cuando la verdad es «mañana» sería mentir, y esa distinción es la razón de que
	 * `Entregado` tenga tres valores y no dos.
	 */
	it("un 429 del cartero es cupo, no fallo", async () => {
		const { pedir } = doble(() => new Response('{"error":"cupo"}', { status: 429 }));
		expect((await new CarteroPorElVps(ajustes, pedir).mandar(carta)).entregado).toBe("cupo");
	});

	it("cualquier otro error es fallo, y dice cuál", async () => {
		const { pedir } = doble(() => new Response("no", { status: 401 }));
		const envio = await new CarteroPorElVps(ajustes, pedir).mandar(carta);
		expect(envio.entregado).toBe("fallo");
		expect(envio.porque).toContain("401");
	});

	it("si no se puede ni hablar con él, es fallo y también lo dice", async () => {
		const pedir = (() => Promise.reject(new Error("ECONNREFUSED"))) as unknown as typeof fetch;
		const envio = await new CarteroPorElVps(ajustes, pedir).mandar(carta);
		expect(envio.entregado).toBe("fallo");
		expect(envio.porque).toContain("ECONNREFUSED");
	});

	/**
	 * Cinco de los diez correos del servidor son bloqueantes, así que un cartero que
	 * no conteste dejaría la petición colgada. Sin el plazo, esta prueba no termina.
	 */
	it("se rinde si el cartero no contesta", async () => {
		const pedir = ((_u: unknown, i: RequestInit) =>
			new Promise<Response>((_, romper) => {
				i.signal?.addEventListener("abort", () => romper(new Error("se acabó el plazo")));
			})) as unknown as typeof fetch;
		const envio = await new CarteroPorElVps({ ...ajustes, plazo: 20 }, pedir).mandar(carta);
		expect(envio.entregado).toBe("fallo");
	});

	/**
	 * **Con el `fetch` de verdad, no con un doble.** Guardado como propiedad y llamado
	 * con `this.pedir(…)`, Workers contesta «Illegal invocation» porque `fetch` exige
	 * `globalThis` como `this`. Todas las demás pruebas de aquí inyectan un doble, que
	 * no tiene ese problema, así que **ninguna podía cazarlo**: lo dijo el primer envío
	 * de verdad contra el cartero. Esta va contra un sitio que no existe: lo que se
	 * comprueba no es que llegue, es **cómo falla**.
	 */
	it("llama al fetch de verdad sin que Workers se queje del `this`", async () => {
		const envio = await new CarteroPorElVps({ ...ajustes, url: "https://no-existe.invalid/entregar", plazo: 5000 }).mandar(carta);
		expect(envio.entregado).toBe("fallo");
		expect(envio.porque).not.toContain("Illegal invocation");
	});

	/**
	 * Lo que contesta el cartero es nuestro, pero lo que contesta un proxy en medio un
	 * día raro puede ser una página entera, y eso acaba en la respuesta del Worker de
	 * pruebas.
	 */
	it("acota lo que repite de la respuesta", async () => {
		const { pedir } = doble(() => new Response("x".repeat(5000), { status: 500 }));
		const envio = await new CarteroPorElVps(ajustes, pedir).mandar(carta);
		expect(envio.porque!.length).toBeLessThan(300);
	});
});
