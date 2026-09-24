import { describe, expect, it } from "vitest";
import { cartas } from "../src/correo";

/**
 * Los correos, desde que **todos van con formato** (2026-09-24, decidido con el
 * cliente).
 *
 * Aquí no se mira cómo se ven —eso solo lo dice un cliente de correo, y ya costó
 * un botón que Gmail no pintaba—; se miran las tres cosas que sí se pueden
 * comprobar y que, si se rompen, no se nota al mandarlos:
 *
 *   - que **lo que se interpola va escapado**, que es lo único de esta tanda que es
 *     un agujero y no un defecto: el nombre de un equipo es texto libre del cliente;
 *   - que **el texto pelado sigue yendo y lleva el código**, porque hay quien lee el
 *     correo así y porque es lo que leen las demás pruebas;
 *   - que **ningún correo con código lleva un enlace que entre por ti**, que en un
 *     gestor de contraseñas es entrenar a la gente para el phishing.
 */
const CODIGO = "123456";

const conCodigo = {
	codigoDeAlta: cartas.codigoDeAlta("ana@ejemplo.com", CODIGO),
	codigoDeEntrada: cartas.codigoDeEntrada("ana@ejemplo.com", CODIGO, "Portátil"),
	codigoDeRecuperacion: cartas.codigoDeRecuperacion("ana@ejemplo.com", CODIGO),
	codigoDeBorrado: cartas.codigoDeBorrado("ana@ejemplo.com", CODIGO),
};

const todos = {
	...conCodigo,
	cuentaCreada: cartas.cuentaCreada("ana@ejemplo.com"),
	yaTienesCuenta: cartas.yaTienesCuenta("ana@ejemplo.com"),
	equipoNuevo: cartas.equipoNuevo("ana@ejemplo.com", "Portátil"),
	claveCambiada: cartas.claveCambiada("ana@ejemplo.com"),
	cuentaBorrada: cartas.cuentaBorrada("ana@ejemplo.com"),
	invitacion: cartas.invitacion("ana@ejemplo.com", "luis@ejemplo.com"),
};

describe("los correos", () => {
	it("van todos con formato, y con su texto pelado", () => {
		for (const [nombre, c] of Object.entries(todos)) {
			expect(`${nombre}: ${c.html ? "con formato" : "pelado"}`).toBe(`${nombre}: con formato`);
			expect(c.texto.length).toBeGreaterThan(40);
			// Toda frase empieza en mayúscula (`internal/cripto/textos_test.go`), **salvo
			// la que empieza por una dirección de correo**: poner en mayúscula la de otra
			// persona sería escribirla mal. Es la única excepción, y se nombra.
			if (nombre !== "invitacion") expect(`${nombre}: ${c.asunto}`).toMatch(/: [A-ZÁÉÍÓÚÑ]/);
			expect(c.html).toContain("<!doctype html>");
		}
	});

	// **El agujero de convertir un correo en HTML.** El nombre del equipo lo manda
	// el cliente y son ochenta caracteres libres: en texto pelado daba igual.
	it("escapa lo que se interpola: un nombre de equipo no puede meter etiquetas", () => {
		const malo = '<img src=x onerror="robar()"> & "comillas"';
		for (const c of [cartas.codigoDeEntrada("ana@ejemplo.com", CODIGO, malo), cartas.equipoNuevo("ana@ejemplo.com", malo)]) {
			// Lo que importa no es que la palabra «onerror» no aparezca —como texto es
			// inofensiva—, sino que **no pueda ser un atributo**: para eso hacen falta el
			// `<` y las comillas, y los dos salen escapados.
			expect(c.html).not.toContain("<img src=x");
			expect(c.html).not.toContain('onerror="');
			expect(c.html).toContain("&lt;img src=x");
			expect(c.html).toContain("onerror=&quot;");
			expect(c.html).toContain("&amp;");
		}
		// Y lo mismo con el correo de quien invita, que también viene de fuera.
		const inv = cartas.invitacion("ana@ejemplo.com", '<b>luis</b>@ejemplo.com');
		expect(inv.html).not.toContain("<b>luis</b>");
		expect(inv.asunto).toContain("<b>luis</b>"); // el asunto es texto: ahí no se escapa
	});

	it("el código va entero en las dos versiones, y se puede copiar", () => {
		for (const [nombre, c] of Object.entries(conCodigo)) {
			// En el texto, con su sangría: es lo que leen `ultimoCodigo` y quien lo lea en texto.
			expect(`${nombre}: ${/^\s+\d{6}$/m.test(c.texto)}`).toBe(`${nombre}: true`);
			// Y en el formato, **como texto y no como imagen**: se selecciona y se lee.
			expect(c.html).toContain(CODIGO);
			expect(c.html).not.toContain("<img src=\"data:");
		}
	});

	it("ningún correo con código lleva un enlace que entre por ti", () => {
		for (const [nombre, c] of Object.entries(conCodigo)) {
			expect(`${nombre}: ${/<a\s/.test(c.html ?? "")}`).toBe(`${nombre}: false`);
		}
	});

	it("el código nunca va en el asunto, que se ve con la pantalla bloqueada", () => {
		for (const [nombre, c] of Object.entries(todos)) {
			expect(`${nombre}: ${/\d{6}/.test(c.asunto)}`).toBe(`${nombre}: false`);
		}
	});
});
