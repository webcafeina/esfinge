import { execSync } from "node:child_process";
import { fileURLToPath } from "node:url";

/** Compila la extensión en modo pruebas: habla con el servidor local y sale en `dist/pruebas`. */
export default function preparar() {
  execSync("pnpm run build", {
    cwd: fileURLToPath(new URL("..", import.meta.url)),
    env: { ...process.env, ESFINGE_CUENTAS_PRUEBAS: "http://127.0.0.1:8793" },
    stdio: "ignore",
  });
}
