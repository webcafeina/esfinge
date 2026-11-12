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

# **La tabla sale del catálogo** (`internal/adjuntos`), no de un heredoc.
#
# Estaba escrita a mano aquí, con `amd64` cableado, y había **otra igual** en el cuerpo de
# la publicación (`publicar.yml`): la misma lista en dos ficheros distintos, que es como se
# desalinea. Y una tabla desalineada no da error: da un enlace que contesta 404, y lo
# descubre quien va a descargar.
#
# En Go por la lección de `zip`: es lo único que seguro está en las cinco máquinas que
# publican.
nueva=$(cd "$raiz" && go run ./herramientas/catalogo tabla-readme "$version")

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
