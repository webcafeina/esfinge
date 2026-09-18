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
