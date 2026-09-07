#!/bin/bash
#
# Instalador de Esfinge para Linux — webcafeína
#
#   ./instalar.sh          instala para tu usuario, sin pedir contraseña
#   sudo ./instalar.sh     instala para todo el mundo, en /usr/local/bin
#
# Sin argumentos elige solo: si puede escribir en /usr/local/bin lo deja ahí, y
# si no, en ~/.local/bin, que es el sitio estándar para los programas de un
# usuario y no necesita permisos de administrador.

set -euo pipefail

negrita=$'\033[1m'; normal=$'\033[0m'
verde=$'\033[32m';  rojo=$'\033[31m'; amarillo=$'\033[33m'
ok="${verde}✓${normal}"; mal="${rojo}✕${normal}"; ojo="${amarillo}!${normal}"

cd "$(dirname "$0")"
nombre="esfinge"

echo
echo "${negrita}▍ webcafeína · instalar Esfinge${normal}"
echo

# ---------------------------------------------------------------- arquitectura
case "$(uname -m)" in
  x86_64|amd64)  origen="esfinge-linux-amd64"; cual="x86-64" ;;
  aarch64|arm64) origen="esfinge-linux-arm64"; cual="ARM64" ;;
  *) echo "  $mal Esta máquina dice ser «$(uname -m)» y no tengo binario para ella."
     echo "     Se compila con: make esfinge"
     exit 1 ;;
esac

# Vale tanto el binario con nombre de plataforma como uno ya renombrado, que es
# lo que sale de «make esfinge».
if [ ! -f "$origen" ] && [ -f "$nombre" ]; then
  origen="$nombre"
fi
if [ ! -f "$origen" ]; then
  echo "  $mal No encuentro el binario «$origen» en $(pwd)."
  exit 1
fi

echo "  $ok Esta máquina es $cual"
chmod +x "$origen"

# --------------------------------------------------------------------- destino
if [ -w /usr/local/bin ] 2>/dev/null; then
  destino="/usr/local/bin"
else
  destino="$HOME/.local/bin"
  mkdir -p "$destino"
fi

install -m 755 "$origen" "$destino/$nombre"
echo "  $ok Instalado en $destino/$nombre"

# ------------------------------------------------------------------ comprobación
if version="$("$destino/$nombre" --version 2>/dev/null)"; then
  echo "  $ok Funciona · $version"
else
  echo "  $mal Se ha copiado pero no arranca."
  exit 1
fi

# ------------------------------------------------------------------------- PATH
# ~/.local/bin no está en el PATH de todas las distribuciones, y un programa que
# se ha instalado pero no se encuentra al escribir su nombre parece no estar.
if ! command -v "$nombre" >/dev/null 2>&1; then
  echo
  echo "  $ojo $destino no está en tu PATH."
  echo "     Añádelo con esta línea y abre un terminal nuevo:"
  echo
  case "$(basename "${SHELL:-bash}")" in
    zsh)  perfil="~/.zshrc" ;;
    fish) perfil="~/.config/fish/config.fish" ;;
    *)    perfil="~/.bashrc" ;;
  esac
  echo "         echo 'export PATH=\"$destino:\$PATH\"' >> $perfil"
fi

# ------------------------------------------------------------- portapapeles
# El botón de copiar usa la herramienta de portapapeles del sistema, que en
# Linux no viene de serie. Sin ella el botón falla, y más vale decirlo ahora que
# cuando haga falta.
if [ -n "${WAYLAND_DISPLAY:-}${DISPLAY:-}" ] &&
   ! command -v xclip >/dev/null 2>&1 &&
   ! command -v xsel  >/dev/null 2>&1 &&
   ! command -v wl-copy >/dev/null 2>&1; then
  echo
  echo "  $ojo Para que funcione el botón «Copiar» hace falta xclip, xsel o wl-copy:"
  echo "         sudo apt install xclip        # Debian y Ubuntu"
  echo "         sudo dnf install xclip        # Fedora"
  echo "     Sin eso, el botón «Guardar en un fichero» sigue funcionando igual."
fi

echo
echo "${negrita}  Ya está.${normal} Escribe ${negrita}$nombre${normal} y saldrán los menús."
echo
