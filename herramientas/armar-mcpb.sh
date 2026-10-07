#!/usr/bin/env bash
# Arma el paquete instalable de Claude Desktop (.mcpb) para una plataforma.
#
#   herramientas/armar-mcpb.sh <versión> <binario> <salida.mcpb>
#
# Un `.mcpb` es **un zip**: no hace falta la herramienta de Anthropic para armarlo, y no
# depender de ella es lo de siempre en esta casa —una dependencia menos en el camino de
# publicar, que es donde más caro sale que algo falle—. Lo que sí hay que respetar es la
# forma, y está en `empaquetado/mcpb/LÉEME.md`.
set -euo pipefail

version="${1:?falta la versión}"
binario="${2:?falta el binario}"
salida="${3:?falta dónde dejarlo}"

aqui="$(cd "$(dirname "$0")/.." && pwd)"
taller="$(mktemp -d)"
trap 'rm -rf "$taller"' EXIT

mkdir -p "$taller/server"
# **El nombre de dentro no cambia entre plataformas**: lo dice el manifiesto, y en
# Windows el `.exe` lo añade Claude Desktop por su cuenta.
nombre="esfinge-mcp"
case "$binario" in *.exe) nombre="esfinge-mcp.exe";; esac
cp "$binario" "$taller/server/$nombre"
chmod +x "$taller/server/$nombre"

# La versión del manifiesto es la de la publicación, no una escrita a mano que se
# quedaría vieja sin que nadie lo notara.
sed "s/\"version\": \"0.0.0\"/\"version\": \"$version\"/" \
  "$aqui/empaquetado/mcpb/manifest.json" > "$taller/manifest.json"
grep -q "\"version\": \"$version\"" "$taller/manifest.json" || {
  echo "::error::no se ha podido poner la versión en el manifiesto" >&2
  exit 1
}

# El icono, que es el de la aplicación.
cp "$aqui/build/appicon.png" "$taller/icon.png"

rm -f "$salida"
( cd "$taller" && zip -qr "$salida" manifest.json icon.png server )
echo "  $(basename "$salida")"
