#!/usr/bin/env bash
#
# Arma la imagen de disco de macOS: la que se monta y enseña «arrastra Esfinge a
# Aplicaciones».
#
# Vive aquí y no dentro del flujo de GitHub para que la receta sea una sola: la
# usan igual «make dmg» en un Mac y el trabajo de publicación.
#
# Necesita create-dmg (brew install create-dmg) y la aplicación ya compilada en
# build/bin/Esfinge.app.
set -euo pipefail

version="${1:-dev}"
raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$raiz"

app="build/bin/Esfinge.app"
salida="Esfinge-${version}.dmg"

if [ ! -d "$app" ]; then
  echo "No está $app. Compila antes con «make app» o «wails build»." >&2
  exit 1
fi

if ! command -v create-dmg >/dev/null; then
  echo "Falta create-dmg: brew install create-dmg" >&2
  exit 1
fi

# El fondo, en las dos resoluciones dentro del mismo TIFF. Es la única forma de
# que macOS use el doble en pantallas Retina: si se le da un PNG suelto lo
# escala, y el texto queda borroso justo en la primera pantalla que se ve.
fondo="build/darwin/fondo-dmg.png"
if command -v tiffutil >/dev/null && [ -f "build/darwin/fondo-dmg@2x.png" ]; then
  tiffutil -cathidpicheck "build/darwin/fondo-dmg.png" "build/darwin/fondo-dmg@2x.png" \
    -out "build/darwin/fondo-dmg.tiff"
  fondo="build/darwin/fondo-dmg.tiff"
fi

# Lo que se ve al montarla: la aplicación, el enlace a Aplicaciones que pone
# create-dmg, y el LÉEME.
rm -rf dmg && mkdir -p dmg
cp -R "$app" dmg/
cp "build/darwin/LÉEME.txt" dmg/

rm -f "$salida"

# create-dmg coloca los iconos hablando con el Finder por AppleScript. En una
# máquina sin sesión gráfica —un runner de integración continua— eso puede
# fallar, y cuando falla devuelve un código distinto de cero aunque la imagen
# esté hecha. Por eso lo que se comprueba es que el fichero exista.
create-dmg \
  --volname "Esfinge" \
  --volicon "$app/Contents/Resources/iconfile.icns" \
  --background "$fondo" \
  --window-pos 200 120 \
  --window-size 660 420 \
  --icon-size 96 \
  --text-size 12 \
  --icon "Esfinge.app" 165 225 \
  --icon "LÉEME.txt" 330 340 \
  --app-drop-link 495 225 \
  --hide-extension "Esfinge.app" \
  --no-internet-enable \
  "$salida" \
  "dmg/" || true

rm -rf dmg

if [ ! -f "$salida" ]; then
  echo "::error::create-dmg no ha producido la imagen" >&2
  exit 1
fi

ls -lh "$salida"
