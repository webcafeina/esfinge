import type { D1Migration } from "@cloudflare/vitest-plugin";
import type { Env as EnvDelServidor } from "../src/protocolo";

declare global {
	namespace Cloudflare {
		interface GlobalProps {
			mainModule: typeof import("../src/indice");
		}
		interface Env extends EnvDelServidor {
			MIGRACIONES: D1Migration[];
			CONFIGURACION: string;
		}
	}
}
