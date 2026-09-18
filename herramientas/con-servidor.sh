#!/usr/bin/env bash
#
# Levanta el servidor de cuentas en local, corre una orden con
# ESFINGE_SERVIDOR_PRUEBAS apuntando a él, y lo apaga pase lo que pase.
#
# # Por qué existe
#
# Las pruebas de Go del cliente y de la sincronización tienen que hablar con **el
# mismo código que está desplegado**, no con un servidor de mentira escrito en Go:
# un ayudante que repite producción prueba el ayudante (la lección de la tubería).
# Así que cada vez se levanta el Worker de verdad con `wrangler dev`, con una base
# nueva en una carpeta temporal y el entorno `local` —frenos holgados, porque
# aquí todo llega desde 127.0.0.1—.
#
# Uso:
#     herramientas/con-servidor.sh go test ./internal/cuenta ./internal/sincro
set -euo pipefail

raiz=$(cd "$(dirname "$0")/.." && pwd)
puerto=${PUERTO_SERVIDOR:-8791}
estado=$(mktemp -d)

cd "$raiz/servidor"
[ -d node_modules ] || pnpm install --frozen-lockfile > /dev/null
pnpm exec wrangler d1 migrations apply BD --local --env local --persist-to "$estado" > "$estado/migraciones.log" 2>&1

# En su propio grupo de procesos: `wrangler dev` lanza workerd por debajo, y
# matar solo al padre deja el motor vivo y el puerto ocupado.
setsid pnpm exec wrangler dev --env local --port "$puerto" --persist-to "$estado" \
	--var PIMIENTA:pimienta-local-de-las-pruebas-de-go-0123 \
	--var SECRETO_PRELOGIN:secreto-local-de-las-pruebas-de-go-0123 \
	> "$estado/servidor.log" 2>&1 &
grupo=$!
apagar() {
	kill -- -"$grupo" 2> /dev/null || true
	wait "$grupo" 2> /dev/null || true
	rm -rf "$estado"
}
trap apagar EXIT

for _ in $(seq 1 60); do
	curl -sf "http://127.0.0.1:$puerto/v1/salud" > /dev/null 2>&1 && break
	sleep 1
done
if ! curl -sf "http://127.0.0.1:$puerto/v1/salud" > /dev/null 2>&1; then
	echo "El servidor de cuentas no ha arrancado en el puerto $puerto:" >&2
	cat "$estado/servidor.log" >&2
	exit 1
fi

cd "$raiz"
ESFINGE_SERVIDOR_PRUEBAS="http://127.0.0.1:$puerto" "$@"
