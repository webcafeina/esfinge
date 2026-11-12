#!/usr/bin/env bash
#
# Pone este servidor en la versión publicada que se le diga.
#
#   herramientas/actualizar-vps.sh 2.61.3
#
# # Por qué existe
#
# **Publicar no acaba en la etiqueta** (lo pidió el cliente el 2026-11-10): los Mac se
# actualizan solos —se lo dice la banda de aviso— y un servidor Linux no, porque ahí
# Esfinge no se reemplaza a sí misma (el .deb instala como root, ADR 0016) y la línea de
# comandos vive en ~/.local/bin puesta a mano. Y dejarla atrás no es cosmético: este
# servidor es titular de una bóveda de proyecto, así que una versión vieja puede acabar sin
# saber leer lo que el Mac acaba de escribir — el día que se rote una bóveda, el formato
# cambia — y el despliegue se queda sin sus secretos. Pasó el 2026-11-12.
#
# La receta estaba escrita en el CLAUDE.md, con `-linux-amd64` cableado y cinco órdenes
# que había que copiar en orden. Aquí está entera, **con la arquitectura sacada del
# catálogo** y sin ningún paso que se pueda saltar: la suma se comprueba siempre, el
# binario anterior se guarda al lado, y al final se prueba que hace su trabajo de verdad.
set -euo pipefail

version="${1:?Falta la versión, por ejemplo 2.61.3}"
raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
destino="${ESFINGE_BIN:-$HOME/.local/bin}"
quien="${ESFINGE_BOVEDA_DE_PRUEBA:-Tempero}"

arq="$(uname -m)"
case "$arq" in
  x86_64|amd64)  arq=amd64 ;;
  aarch64|arm64) arq=arm64 ;;
  *) echo "✕ Esta máquina dice ser «$arq» y no hay binario para ella." >&2; exit 1 ;;
esac

# El nombre no se compone aquí: lo dice el catálogo, que es el único sitio donde viven los
# nombres de los adjuntos (internal/adjuntos).
fichero="$(cd "$raiz" && go run ./herramientas/catalogo nombre "cli-linux-$arq" "$version")"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cd "$tmp"

echo "▍ webcafeína · Esfinge $version en este servidor ($arq)"
gh release download "v$version" -R webcafeina/esfinge -p "$fichero" -p SHA256SUMS

# **La suma, siempre.** Es lo único que distingue «he bajado el binario» de «he bajado
# algo».
sha256sum -c --ignore-missing SHA256SUMS

mkdir -p "$destino"
if [ -f "$destino/esfinge" ]; then
  cp "$destino/esfinge" "$destino/esfinge-ANTERIOR"
  echo "· El anterior queda en $destino/esfinge-ANTERIOR ($("$destino/esfinge-ANTERIOR" --version 2>/dev/null || echo 'sin versión'))"
fi
install -m 755 "$fichero" "$destino/esfinge"

echo "· Ahora: $("$destino/esfinge" --version)"

# **Y que haga su trabajo, que es lo que de verdad dice que va.** `--version` solo dice
# que el fichero arranca; esto dice que abre la bóveda compartida, se pone al día con la
# cuenta y lee su contenido. Sin este paso, una versión que no sabe leer el formato nuevo
# pasaría por buena.
#
# Ojo con la primera línea de la salida: un «no he podido poner al día… esto puede estar
# viejo» sale **con el listado completo debajo**, así que es fácil leer el listado y no el
# aviso. Por eso aquí se mira que no esté.
salida="$("$destino/esfinge" boveda listar --compartida "$quien" 2>&1)"
echo "$salida"
if grep -q "no he podido poner al día" <<<"$salida"; then
  echo "✕ Ha servido una copia vieja: la sincronización ha fallado. No te quedes con el listado." >&2
  exit 1
fi
if ! grep -q .  <<<"$salida"; then
  echo "✕ La bóveda «$quien» no ha devuelto nada." >&2
  exit 1
fi
echo "✓ $version instalada, con su suma comprobada, y leyendo «$quien»."
