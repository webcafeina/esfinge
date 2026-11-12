# Dónde está Go, para los guiones.
#
# Se incluye con `.` o `source`, no se ejecuta: deja puesta la variable `GO`.
#
# # Por qué existe
#
# **Go no está en el PATH del sistema de esta máquina**: vive en ~/.local/go porque se
# instaló sin sudo, y el Makefile ya lo tiene en cuenta (`GO ?= $(HOME)/.local/go/bin/go`).
# Los guiones no lo tenían, y el del VPS murió con «go: command not found» justo después
# de una publicación en verde.
#
# Y la razón de que esto sea un fichero y no dos copias de cuatro líneas es la de siempre
# en esta casa: **una regla escrita en N sitios se desalinea sola**. Ya ha costado cinco
# fallos —`vaciarLoSensible`, `enConfiguracionDePruebas`, `PasaPorLaValvula`…—, así que la
# segunda vez que hacía falta buscar Go se escribe una vez y los demás preguntan.
#
# Se respeta `GO` si viene puesta: así se puede probar otra instalación sin tocar nada.
if [ -z "${GO:-}" ]; then
  if command -v go >/dev/null 2>&1; then
    GO=go
  elif [ -x "$HOME/.local/go/bin/go" ]; then
    GO="$HOME/.local/go/bin/go"
  else
    echo "✕ No encuentro Go. Pon GO=/ruta/a/go, o instálalo." >&2
    exit 1
  fi
fi
