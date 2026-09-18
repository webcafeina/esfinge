#!/usr/bin/env bash
#
# El servidor de cuentas para las pruebas de la interfaz (frontend/e2e/cuentas.spec.ts):
# el Worker de verdad con `wrangler dev`, el entorno `local` y una base nueva cada
# vez. Lo arranca Playwright y lo apaga al terminar, así que aquí se hace `exec`:
# el proceso que Playwright mata tiene que ser el servidor, no este guion.
#
# Uso: herramientas/servidor-para-e2e.sh <puerto>
set -euo pipefail

raiz=$(cd "$(dirname "$0")/.." && pwd)
puerto=${1:-8792}
estado=$(mktemp -d)

cd "$raiz/servidor"
[ -d node_modules ] || pnpm install --frozen-lockfile > /dev/null
pnpm exec wrangler d1 migrations apply BD --local --env local --persist-to "$estado" > /dev/null 2>&1
exec pnpm exec wrangler dev --env local --port "$puerto" --persist-to "$estado" \
	--var PIMIENTA:pimienta-local-de-las-pruebas-e2e-0123456789 \
	--var SECRETO_PRELOGIN:secreto-local-de-las-pruebas-e2e-0123456789
