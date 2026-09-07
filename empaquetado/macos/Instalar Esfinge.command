#!/bin/bash
#
# Instalador de Esfinge para macOS — webcafeína
#
# Copia el binario que le toca a este Mac en /usr/local/bin y le quita el
# bloqueo que macOS pone a todo lo que se descarga de internet.
#
# Funciona de dos maneras, y las dos hacen lo mismo:
#   · doble clic sobre este fichero
#   · bash "Instalar Esfinge.command"   desde el Terminal
#
# Aquí no se usa color de marca a propósito. El Terminal de macOS es blanco de
# fábrica y muchos lo ponen negro; el amarillo de Esfinge sobre blanco es
# ilegible y la tinta oscura desaparece sobre negro. Negrita y los dos colores
# que se leen en cualquier fondo, y en paz.

set -euo pipefail

negrita=$'\033[1m'; normal=$'\033[0m'
verde=$'\033[32m';  rojo=$'\033[31m'
ok="${verde}✓${normal}"; mal="${rojo}✕${normal}"

# Con doble clic, el directorio de trabajo es el home y no la carpeta del
# instalador. Sin esto no encontraría los binarios que tiene al lado.
cd "$(dirname "$0")"

destino="/usr/local/bin"
nombre="esfinge"

echo
echo "${negrita}▍ webcafeína · instalar Esfinge${normal}"
echo

# ---------------------------------------------------------------- arquitectura
case "$(uname -m)" in
  arm64)  origen="esfinge-apple-silicon"; cual="Apple Silicon" ;;
  x86_64) origen="esfinge-intel";         cual="Intel" ;;
  *)      echo "  $mal Este Mac dice ser «$(uname -m)» y no sé qué darle."
          echo "     Escribe a Webcafeína con esa palabra y te mando el binario."
          echo; read -r -n 1 -p "  Pulsa una tecla para cerrar." || true; echo; exit 1 ;;
esac

if [ ! -f "$origen" ]; then
  echo "  $mal Falta el fichero «$origen»."
  echo "     Descomprime el ZIP entero antes de ejecutar el instalador:"
  echo "     el instalador y los binarios tienen que estar en la misma carpeta."
  echo; read -r -n 1 -p "  Pulsa una tecla para cerrar." || true; echo; exit 1
fi

echo "  $ok Este Mac es $cual"

# ------------------------------------------------------------------ cuarentena
# macOS marca todo lo que viene de internet y se niega a ejecutarlo. Se le quita
# a la carpeta entera, que es lo que hace falta para que el binario arranque.
xattr -dr com.apple.quarantine . 2>/dev/null || true
echo "  $ok Quitado el bloqueo de descarga"

chmod +x "$origen"

# --------------------------------------------------------------------- destino
# /usr/local/bin ya está en el PATH de macOS de fábrica, así que instalando ahí
# el comando funciona desde cualquier carpeta sin tocar ninguna configuración.
if [ -d "$destino" ] && [ -w "$destino" ]; then
  cp "$origen" "$destino/$nombre"
else
  echo
  echo "  Para dejarlo en $destino hace falta tu contraseña de administrador."
  echo "  Es la misma con la que enciendes el Mac. Al escribirla no se ve nada:"
  echo "  es normal, teclea y pulsa intro."
  echo
  sudo mkdir -p "$destino"
  sudo cp "$origen" "$destino/$nombre"
  sudo chmod 755 "$destino/$nombre"
fi

echo "  $ok Instalado en $destino/$nombre"

# ------------------------------------------------------------- comprobación
# No basta con copiar el fichero: hay que verlo arrancar. Si la firma o los
# permisos estuvieran mal, es aquí donde se nota y no tres días después.
if version="$("$destino/$nombre" --version 2>/dev/null)"; then
  echo "  $ok Funciona · $version"
else
  echo "  $mal Se ha copiado pero no arranca."
  echo "     Manda esto a Webcafeína: $(uname -m) · $(sw_vers -productVersion 2>/dev/null || echo '?')"
  echo; read -r -n 1 -p "  Pulsa una tecla para cerrar." || true; echo; exit 1
fi

echo
echo "${negrita}  Ya está.${normal}"
echo
echo "  Abre el Terminal y escribe:"
echo
echo "      ${negrita}esfinge${normal}"
echo
echo "  y saldrán los menús. Sin nada más, es todo lo que hay que saber."
echo
echo "  Un aviso que conviene leer una vez: lo que cifres solo se puede abrir"
echo "  con la clave que hayas usado. No hay «he olvidado mi contraseña». Si se"
echo "  pierde la clave, se pierde el contenido, y no hay nadie que pueda"
echo "  recuperarlo."
echo
read -r -n 1 -p "  Pulsa una tecla para cerrar esta ventana." || true
echo
