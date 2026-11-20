#!/usr/bin/env bash
# Deja `makensis` donde wails lo busca, en una máquina Windows de GitHub.
#
# **Está aquí y no dentro de un flujo porque hacen falta dos** —`publicar.yml` para
# armar el instalador que se cuelga de la publicación, y `compilar.yml` para que una
# errata del guion NSIS no se descubra con la etiqueta ya empujada—, y la misma regla
# escrita en dos sitios se desalinea sola: aquí ya ha costado siete veces. Y la forma
# en que se desalineó esta es la peor de todas, porque **no se pone roja**: hasta el
# 2026-11-20, `compilar.yml` llamaba a `wails build -nsis` sin haber instalado NSIS,
# así que wails avisaba de que no encuentra `makensis`, **salía con código cero** y el
# paso de verificación quedaba en verde sin haber armado ningún instalador. Un aviso
# que no para nada deja de leerse; éste nunca se llegó a leer.
#
# Lo que hace, por orden:
#
#   1 · Mira si la máquina lo trae. **`windows-11-arm` sí** —en `/c/Program Files
#       (x86)/NSIS`— y `windows-latest` no, medido en el registro de tres
#       publicaciones seguidas. Chocolatey lo instala **fuera del PATH** y wails no
#       lo busca en ningún otro sitio.
#   2 · Si no, lo baja **del proyecto NSIS y con su suma comprobada**, no de un
#       intermediario: Chocolatey devolvió 503 durante media hora y tiró dos
#       publicaciones en una tarde con todo lo demás en verde. Una descarga sin
#       comprobar sería peor que la dependencia que sustituye.
#   3 · Escribe la ruta en `GITHUB_PATH` **como la entiende Windows**. Con la de Git
#       bash —«/c/Program Files (x86)/NSIS»— el fichero está y el paso pasa, pero
#       wails sigue sin encontrar `makensis`: eso es lo que ocurrió en la 2.0.2.
set -euo pipefail

buscar() {
  for d in "/c/Program Files (x86)/NSIS" "/c/Program Files/NSIS"; do
    [ -x "$d/makensis.exe" ] && { echo "$d"; return 0; }
  done
  return 1
}

if nsis=$(buscar); then
  echo "NSIS ya viene con la máquina: $nsis"
else
  echo "NSIS no viene con la máquina; se baja del proyecto NSIS"
  version=3.10
  suma=fcdce3229717a2a148e7cda0ab5bdb667f39d8fb33ede1da8dabc336bd5ad110
  curl -sSL --retry 3 --retry-delay 5 -o nsis.zip \
    "https://downloads.sourceforge.net/project/nsis/NSIS%203/$version/nsis-$version.zip"
  echo "$suma  nsis.zip" | sha256sum -c - || {
    echo "::error::El zip de NSIS no es el que se esperaba"
    exit 1
  }
  unzip -q nsis.zip -d /c/nsis
  nsis="/c/nsis/nsis-$version"
  # NSISDIR le dice al compilador dónde están sus cabeceras y sus plantillas. Con el
  # zip no hay instalador que lo apunte en el registro, así que se dice a mano.
  echo "NSISDIR=$(cygpath -w "$nsis")" >> "$GITHUB_ENV"
  [ -x "$nsis/makensis.exe" ] || {
    echo "::error::El zip de NSIS no trae makensis donde se esperaba"
    exit 1
  }
fi
cygpath -w "$nsis" >> "$GITHUB_PATH"
