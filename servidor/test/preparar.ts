import { applyD1Migrations } from "cloudflare:test";
import { env } from "cloudflare:workers";

// Cada fichero de pruebas empieza con su propia base, así que las migraciones se
// aplican antes de cada uno. Es idempotente: lo ya aplicado se salta.
await applyD1Migrations(env.BD, env.MIGRACIONES);
