#!/usr/bin/env bash
# Arma la web del proyecto —portada, privacidad y soporte— en una carpeta lista para
# GitHub Pages (ADR 0033). La usa .github/workflows/web.yml, y sirve igual en local
# para mirarla antes de publicar.
#
# **Los colores no se copian a mano**: se trae `frontend/src/tokens.css`, que genera
# internal/tema y cuyo contraste ya está medido. Y **se comprueba que no haya enlaces
# rotos a ficheros de la propia web**: una política de privacidad que no carga es la
# que enlazan las dos tiendas.
#
# Uso: herramientas/armar-web.sh _site
set -euo pipefail

salida="${1:?Falta la carpeta de salida}"
raiz="$(cd "$(dirname "$0")/.." && pwd)"
cd "$raiz"

rm -rf "$salida"
mkdir -p "$salida/imagenes"
cp web/*.html web/*.css "$salida/"
cp frontend/src/tokens.css "$salida/tokens.css"
# Sin las capturas de la portada del README, que son de la 2.0.3 y no enseñan la
# bóveda: a quien llega desde una tienda es peor enseñarle una ventana que ya no es
# así que no enseñarle ninguna (docs/deuda.md).
cp docs/imagenes/icono.png "$salida/imagenes/"
# Sin Jekyll: la web ya está hecha y no hay nada que procesar.
touch "$salida/.nojekyll"

# Los enlaces y recursos locales —ni http, ni mailto, ni anclas de la misma página—
# tienen que existir.
fallos=0
for pagina in "$salida"/*.html; do
  while IFS= read -r destino; do
    fichero="${destino%%#*}"
    [ -z "$fichero" ] && continue
    [ "$fichero" = "./" ] && fichero="index.html"
    if [ ! -e "$salida/$fichero" ]; then
      echo "Enlace roto en $(basename "$pagina"): $destino" >&2
      fallos=1
    fi
  done < <(grep -oE '(href|src|srcset)="[^"]+"' "$pagina" | sed -E 's/^[a-z]+="//; s/"$//' | grep -vE '^(https?:|mailto:|#)')
done
[ "$fallos" -eq 0 ] || exit 1

echo "Web armada en $salida"
