#!/usr/bin/env bash
# El código fuente de la extensión, tal como lo pide Mozilla para revisarla (ADR 0033).
#
# **Por qué hace falta**: lo que se sube a addons.mozilla.org lo empaqueta Vite en
# tres ficheros, y Mozilla exige el código de verdad con instrucciones para compilarlo
# y comprueba que sale **lo mismo byte a byte**. Si no sale, rechaza la versión.
#
# **Qué lleva**: lo que sigue git en `navegador/` —nunca `node_modules`, `dist` ni las
# capturas—, más los cuatro ficheros de fuera que la extensión importa, en sus mismas
# rutas para que los `import` sigan valiendo, y `COMPILAR.md` en la raíz con la
# versión escrita. Si la extensión empieza a importar otro fichero de fuera, hay que
# añadirlo aquí: lo detecta `comprobar-fuente-de-la-extension.sh`, porque sin él la
# compilación falla.
#
# Uso: herramientas/fuente-de-la-extension.sh 2.23.0 dist/esfinge-extension-2.23.0-fuente.zip
set -euo pipefail

version="${1:?Falta la versión, por ejemplo 2.23.0}"
salida="${2:?Falta el zip de salida}"

raiz="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$(dirname "$salida")"
salida="$(cd "$(dirname "$salida")" && pwd)/$(basename "$salida")"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cd "$raiz"
# Lo que sigue git y lo nuevo que no ignora: así también vale antes de confirmar.
git ls-files -z --cached --others --exclude-standard -- \
  navegador \
  build/marca.svg \
  build/icono-barra.svg \
  frontend/src/monograma.ts \
  frontend/src/tokens.css |
  xargs -0 cp --parents -t "$tmp"

sed "s/{{VERSION}}/$version/g" navegador/COMPILAR.md >"$tmp/COMPILAR.md"

rm -f "$salida"
(cd "$tmp" && zip -qrX "$salida" .)
echo "Código fuente de la extensión: $salida"
