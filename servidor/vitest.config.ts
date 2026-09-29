import { readFileSync } from "node:fs";
import { cloudflareTest, readD1Migrations } from "@cloudflare/vitest-plugin";
import { defineConfig } from "vitest/config";

// Las pruebas corren dentro del motor de Workers de verdad (workerd), con la
// configuración del Worker de pruebas y secretos de mentira.
export default defineConfig(async () => {
	const migraciones = await readD1Migrations("./migraciones");
	return {
		plugins: [
			cloudflareTest({
				wrangler: { configPath: "./wrangler.jsonc", environment: "pruebas" },
				miniflare: {
					bindings: {
						PIMIENTA: "pimienta-de-las-pruebas-0123456789abcdef",
						// **Vacías, pero declaradas.** El `env` de las pruebas deja cambiar una
						// propiedad que ya existe y **no deja añadir una nueva**, así que sin
						// esto las pruebas de la rotación no podrían inventarse `PIMIENTA_2`.
						// Vacío es «no está», que es lo que ve el Worker de verdad.
						PIMIENTA_2: "",
						PIMIENTA_3: "",
						SECRETO_PRELOGIN: "secreto-de-las-pruebas-0123456789abcdef",
						MIGRACIONES: migraciones,
						// El motor local no implementa jurisdicciones (ver objeto() en indice.ts).
						JURISDICCION: "",
						// Para que una prueba compruebe que los Workers de verdad sí la llevan.
						CONFIGURACION: readFileSync("./wrangler.jsonc", "utf8"),
					},
				},
			}),
		],
		test: { setupFiles: ["./test/preparar.ts"] },
	};
});
