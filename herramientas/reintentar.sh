#!/usr/bin/env bash
#
# Repite una orden que falla por la red, con espera creciente.
#
# # Por qué existe
#
# Este proyecto lleva ya **cuatro publicaciones caídas por un tercero que tuvo un
# mal minuto**, con todo lo nuestro en verde: un repositorio de apt con el índice
# caducado, el servidor de Chocolatey devolviendo 503 dos veces seguidas, y la
# 2.18.0 por esto:
#
#     go install …/wails@v2.15.0: verifying module:
#       Get "https://sum.golang.org/lookup/…": net/http: TLS handshake timeout
#
# De ahí salió una regla que ya está en `CLAUDE.md` —«si hay que bajar algo, del
# origen y con su suma»— y esto es la otra mitad: **del origen, con su suma, y
# reintentando**. Un apretón de manos TLS que expira no es un fallo de la
# compilación, es un segundo malo; tratarlo como un fallo es tirar una publicación
# entera y hacer que el correo de «ha fallado» deje de significar algo.
#
# Lo que **no** hace, y es a propósito: no baja la guardia. Nada de `GOFLAGS`
# saltándose la base de datos de sumas ni de `--insecure`, que es la forma fácil de
# que esto no vuelva a fallar y también la forma de que un día entre otra cosa. Se
# reintenta exactamente lo mismo.
#
# Uso:
#     herramientas/reintentar.sh go install github.com/…@v2.15.0
set -u

veces=${REINTENTOS:-4}
espera=5

for intento in $(seq 1 "$veces"); do
	if "$@"; then
		exit 0
	fi
	if [ "$intento" -eq "$veces" ]; then
		echo "Se ha intentado $veces veces y sigue fallando: $*" >&2
		exit 1
	fi
	echo "Intento $intento de $veces fallido; se repite en ${espera}s: $*" >&2
	sleep "$espera"
	espera=$((espera * 2))
done
