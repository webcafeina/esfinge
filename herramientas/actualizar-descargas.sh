#!/usr/bin/env bash
#
# Reescribe la tabla de descargas del README con los enlaces directos de una
# versión.
#
# Los ficheros de la publicación llevan la versión en el nombre, que es lo que se
# quiere al descargarlos —en la carpeta de Descargas se distinguen— pero deja los
# enlaces caducados en cuanto sale la siguiente. Así que no se escriben a mano:
# los pone esto, y lo llama el flujo de publicación después de crear la
# publicación.
#
#   herramientas/actualizar-descargas.sh 2.0.3
#
# La tabla vive entre dos marcas en el README. Lo de fuera no se toca.
set -euo pipefail

version="${1:?Falta la versión, por ejemplo 2.0.3}"
raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readme="$raiz/README.md"
base="https://github.com/webcafeina/esfinge/releases/download/v${version}"

nueva=$(cat <<TABLA
<!-- descargas:inicio · las escribe herramientas/actualizar-descargas.sh -->

| Sistema | Descarga | Notas |
|---|---|---|
| **macOS** | [Esfinge-${version}.dmg](${base}/Esfinge-${version}.dmg) | Universal: Apple Silicon e Intel |
| **Windows** | [Esfinge-${version}-windows-instalador.exe](${base}/Esfinge-${version}-windows-instalador.exe) | Asistente de instalación |
| **Linux · Debian y Ubuntu** | [esfinge_${version}_amd64.deb](${base}/esfinge_${version}_amd64.deb) | Aplicación y línea de comandos |
| **Linux · cualquiera** | [esfinge-${version}-linux-amd64.tar.gz](${base}/esfinge-${version}-linux-amd64.tar.gz) | Los binarios sueltos |
| **Línea de comandos** | [todos los sistemas](https://github.com/webcafeina/esfinge/releases/latest) | Seis objetivos, con sus SHA256 |

Versión **${version}**. Las anteriores, en [publicaciones](https://github.com/webcafeina/esfinge/releases).

<!-- descargas:fin -->
TABLA
)

python3 - "$readme" "$nueva" <<'PY'
import re, sys
camino, nueva = sys.argv[1], sys.argv[2]
with open(camino, encoding="utf8") as f:
    texto = f.read()
patron = re.compile(r"<!-- descargas:inicio.*?<!-- descargas:fin -->", re.S)
if not patron.search(texto):
    sys.exit("El README no tiene las marcas «descargas:inicio» y «descargas:fin»")
nuevo = patron.sub(lambda _: nueva, texto)
if nuevo == texto:
    print("La tabla ya estaba al día")
else:
    with open(camino, "w", encoding="utf8") as f:
        f.write(nuevo)
    print("Tabla de descargas actualizada")
PY
