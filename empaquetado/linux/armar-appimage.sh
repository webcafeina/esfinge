#!/usr/bin/env bash
#
# Arma la AppImage de Esfinge: la aplicación con ventana, autosuficiente, para cualquier
# distribución.
#
# Se usa igual aquí y en la publicación —como `empaquetado/macos/armar-dmg.sh`—, porque un
# camino que solo existe en CI es el que nadie prueba:
#
#     empaquetado/linux/armar-appimage.sh 2.65.0 x86_64
#
# # Qué es «autosuficiente», y por qué cuesta
#
# El `.deb` y el `.rpm` piden WebKitGTK al sistema. La AppImage **lo lleva dentro**, que es
# lo que la hace servir en una distribución que no lo empaquete: son unos 70 MiB más y es
# lo que decidió el cliente, mirando que la de Apolo son 98 MiB frente a 26 del `.deb`.
#
# Y las bibliotecas no son la parte difícil: `linuxdeploy` copia lo que el binario pide y
# el plugin de GTK añade los esquemas, los typelibs, los módulos y los cargadores de
# imágenes. **Lo que ninguno de los dos puede saber es que WebKitGTK arranca procesos
# auxiliares** —`WebKitWebProcess`, `WebKitNetworkProcess`, `WebKitGPUProcess`— que no son
# dependencias dinámicas de nadie: no aparecen en un `ldd`, así que no se copian solos. Sin
# ellos la ventana abre y se queda en blanco. Van a mano, y dónde viven está **medido** en un
# contenedor de Ubuntu 24.04 y no supuesto: en Debian y Ubuntu, en /usr/lib/<triplete>/webkit2gtk-4.1/
# y no en /usr/libexec como en otras distribuciones.
#
# **Y copiarlos dentro no basta, que es lo que costó la 2.67.0.** La ruta de esos programas va
# compilada a fuego dentro de libwebkit, así que WebKit arranca los **del sistema de quien la
# ejecuta** aunque los nuestros estén al lado. Aquí se exportaba `WEBKIT_EXEC_PATH` para
# redirigirlo y **esa variable no existe**: sólo la mira WebKit en compilaciones de desarrollo, y
# en la que empaquetan las distribuciones ni siquiera está la cadena dentro de la biblioteca. Lo
# que lo resuelve es `webkit-de-dentro.c`, que se interpone en la llamada que los lanza; su
# cabecera cuenta los dos fallos que daba, que no se parecen y son el mismo.
#
# # Las herramientas: del origen, con su suma, y fijadas
#
# Son **cuatro** piezas, no tres, y conviene saber por qué. `appimagetool` y `linuxdeploy`
# publican una etiqueta rodante «continuous» —cuyo contenido cambia cuando les parece, o
# sea una dependencia que puede tirar una publicación cualquier martes— **pero las dos
# tienen además etiquetas con fecha**, inmutables. El plugin de GTK no publica nada: se coge
# por su URL cruda **en un commit concreto**, inmutable por construcción.
#
# **Y la cuarta es el `runtime`, que al principio se me escapó.** `appimagetool` se lo baja
# él solo al armar, de `type2-runtime/releases/continuous` y **sin comprobar nada** — se vio
# en el registro de una publicación, no leyendo. Eso no es un detalle: el runtime es **lo
# primero que se ejecuta** en el equipo de quien abre la AppImage, así que va fijado como los
# otros tres y se le pasa con `--runtime-file`. De paso, armar deja de tocar la red.
#
# Ese sí cuelga de una etiqueta rodante, porque `type2-runtime` **no publica otra cosa**. El
# precio es que el día que la reconstruyan esto se pondrá rojo; es a propósito. Un cambio
# silencioso en el cargador de la aplicación es peor que una compilación en rojo que alguien
# tiene que mirar.
#
# Las sumas están escritas abajo y se comprueban siempre. Si alguna no cuadra, esto para:
# una descarga sin comprobar es peor que la dependencia que sustituye.
set -euo pipefail

version="${1:-}"
arq="${2:-}"
if [ -z "$version" ] || [ -z "$arq" ]; then
  echo "Uso: $0 <versión> <x86_64|aarch64> [fichero de salida]" >&2
  exit 2
fi
case "$arq" in
  x86_64)  arqgo=amd64 ;;
  aarch64) arqgo=arm64 ;;
  *) echo "✕ La arquitectura de una AppImage es «x86_64» o «aarch64», y me has dado «$arq»." >&2; exit 2 ;;
esac

raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$raiz"
. "herramientas/go.sh"

# El nombre lo dice el catálogo, que es el único sitio donde viven los nombres de los
# adjuntos. Se puede dar otro por el tercer argumento, para probar sin pisar nada.
salida="${3:-$("$GO" run ./herramientas/catalogo nombre "appimage-$arqgo" "$version")}"

# **Lo que hace falta se mira antes de bajar treinta megas de herramientas**, que es la
# lección de `ExportarLlaves`: preguntar antes de preguntar. Y se nombra lo que falta, no
# «falta algo».
falta=()
for f in build/bin/Esfinge esfinge esfinge-puente esfinge-mcp \
         empaquetado/linux/esfinge.desktop empaquetado/linux/esfinge-mime.xml \
         build/linux/hicolor/256x256.png; do
  [ -e "$f" ] || falta+=("$f")
done
if [ ${#falta[@]} -gt 0 ]; then
  echo "✕ Antes de armar la AppImage hace falta compilar. No están:" >&2
  printf '    %s\n' "${falta[@]}" >&2
  echo "  La ventana sale de «wails build» y los tres binarios de «make publicar»." >&2
  exit 1
fi

# ---------------------------------------------------------------- las herramientas

APPIMAGETOOL=1.9.1
LINUXDEPLOY=1-alpha-20251107-1
PLUGIN_GTK=7a3fbc31a9e5075073ff8790f26effbac5f84453

# Sumas tomadas el 2026-11-12 de las descargas de arriba. **No se regeneran cuando algo
# falla**: si una no cuadra es que lo que hay al otro lado no es lo que se comprobó.
suma_appimagetool_x86_64=ed4ce84f0d9caff66f50bcca6ff6f35aae54ce8135408b3fa33abfc3cb384eb0
suma_appimagetool_aarch64=f0837e7448a0c1e4e650a93bb3e85802546e60654ef287576f46c71c126a9158
suma_linuxdeploy_x86_64=c20cd71e3a4e3b80c3483cef793cda3f4e990aca14014d23c544ca3ce1270b4d
suma_linuxdeploy_aarch64=620095110d693282b8ebeb244a95b5e911cf8f65f76c88b4b47d16ae6346fcff
suma_plugin_gtk=b0f4cbc684a0103a9651f0955b635eaea0096b3a66c0f5a2c2aa337960375171
suma_runtime_x86_64=617aa77314d57c8b8e4fae9c3f0b35af7b30d5529cb69e98f64f92510ed98a46
suma_runtime_aarch64=9a72a744d221576c9347ec074243acc5c3c7568896b12714e0e82e9f0bf47900

herr="${ESFINGE_HERRAMIENTAS:-$HOME/.cache/esfinge/appimage}"
mkdir -p "$herr"

# **Que falte `curl` no es un mal minuto de red**, y `reintentar.sh` está escrito para lo
# segundo: sin esta línea tarda treinta y cinco segundos en decir cuatro veces «command not
# found» para contar algo que se sabía antes de empezar.
command -v curl >/dev/null 2>&1 || { echo "✕ Hace falta curl para traer las herramientas." >&2; exit 1; }
command -v sha256sum >/dev/null 2>&1 || { echo "✕ Hace falta sha256sum para comprobarlas." >&2; exit 1; }

# traer <destino> <url> <suma esperada>
traer() {
  local destino="$1" url="$2" esperada="$3"
  if [ -f "$destino" ] && [ "$(sha256sum "$destino" | cut -d' ' -f1)" = "$esperada" ]; then
    # **El permiso se pone también aquí, no solo al bajar.** Con el caché caliente esto no
    # descargaba nada y daba por bueno un fichero sin el bit de ejecución: «Permission
    # denied» al llamarlo, que no se parece en nada a la causa. Es el mismo fallo que el
    # zip de los `.mcpb`, donde lo ponía una herramienta por costumbre y no lo comprobaba
    # nadie.
    chmod +x "$destino"
    return 0
  fi
  echo "· bajando $(basename "$destino")"
  bash herramientas/reintentar.sh curl -fsSL -o "$destino.parcial" "$url"
  local vista
  vista="$(sha256sum "$destino.parcial" | cut -d' ' -f1)"
  if [ "$vista" != "$esperada" ]; then
    rm -f "$destino.parcial"
    echo "✕ La suma de $(basename "$destino") no es la que debería:" >&2
    echo "    esperaba $esperada" >&2
    echo "    he visto  $vista" >&2
    echo "  Eso no se arregla actualizando la suma: mira primero qué hay al otro lado." >&2
    exit 1
  fi
  mv "$destino.parcial" "$destino"
  chmod +x "$destino"
}

eval "suma_at=\$suma_appimagetool_$arq"
eval "suma_ld=\$suma_linuxdeploy_$arq"
traer "$herr/appimagetool-$arq" \
  "https://github.com/AppImage/appimagetool/releases/download/$APPIMAGETOOL/appimagetool-$arq.AppImage" "$suma_at"
traer "$herr/linuxdeploy-$arq" \
  "https://github.com/linuxdeploy/linuxdeploy/releases/download/$LINUXDEPLOY/linuxdeploy-$arq.AppImage" "$suma_ld"
traer "$herr/linuxdeploy-plugin-gtk.sh" \
  "https://raw.githubusercontent.com/linuxdeploy/linuxdeploy-plugin-gtk/$PLUGIN_GTK/linuxdeploy-plugin-gtk.sh" "$suma_plugin_gtk"
eval "suma_rt=\$suma_runtime_$arq"
traer "$herr/runtime-$arq" \
  "https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-$arq" "$suma_rt"

# **Las tres herramientas son AppImages, y en CI no hay FUSE**: sin esto se quedan en
# «Cannot mount AppImage, please check your FUSE setup».
export APPIMAGE_EXTRACT_AND_RUN=1
# linuxdeploy busca sus plugins en el PATH, por su nombre.
export PATH="$herr:$PATH"
ln -sf "$herr/linuxdeploy-plugin-gtk.sh" "$herr/linuxdeploy-plugin-gtk"

# ---------------------------------------------------------------- el AppDir

appdir="$(mktemp -d)/AppDir"
trap 'rm -rf "$(dirname "$appdir")"' EXIT
mkdir -p "$appdir/usr/bin" "$appdir/usr/share/applications" "$appdir/usr/share/mime/packages" "$appdir/apprun-hooks"

cp build/bin/Esfinge "$appdir/usr/bin/esfinge-gui"
# Los tres de siempre, por lo mismo que en el paquete: el puente del navegador, la línea de
# comandos y el servidor para los agentes.
cp esfinge esfinge-puente esfinge-mcp "$appdir/usr/bin/"
chmod +x "$appdir/usr/bin/"*

# **El lanzador se deriva del del paquete, no se escribe otro**: así el nombre, las
# categorías y el tipo de fichero tienen un solo sitio donde viven. Lo único que cambia es
# el `Exec`, que dentro de una AppImage **no puede ser absoluto** —/usr/bin/esfinge-gui no
# existe en la máquina de quien la ejecuta— y lo resuelve el AppRun desde $APPDIR.
sed 's|^Exec=.*|Exec=esfinge-gui %f|' empaquetado/linux/esfinge.desktop \
  > "$appdir/usr/share/applications/esfinge.desktop"
cp empaquetado/linux/esfinge-mime.xml "$appdir/usr/share/mime/packages/esfinge.xml"

for t in 16 24 32 48 64 128 256 512; do
  mkdir -p "$appdir/usr/share/icons/hicolor/${t}x${t}/apps"
  cp "build/linux/hicolor/${t}x${t}.png" "$appdir/usr/share/icons/hicolor/${t}x${t}/apps/esfinge.png"
done

# ---------------------------------------------------------------- WebKitGTK, a mano

# Dónde está la carpeta de los procesos auxiliares. Se busca con un glob en vez de armar el
# triplete, que cambia por arquitectura y se saca con herramientas distintas en cada
# distribución; y **si no está, esto para**: una AppImage sin ellos abre una ventana en
# blanco, que es el peor fallo posible porque parece que el programa está roto.
libexec=""
for d in /usr/lib/*/webkit2gtk-4.1 /usr/libexec/webkit2gtk-4.1 /usr/lib/webkit2gtk-4.1; do
  [ -d "$d" ] && [ -x "$d/WebKitWebProcess" ] && { libexec="$d"; break; }
done
if [ -z "$libexec" ]; then
  echo "✕ No encuentro los procesos auxiliares de WebKitGTK (WebKitWebProcess)." >&2
  echo "  Sin ellos la AppImage abre la ventana en blanco. Instala libwebkit2gtk-4.1-0." >&2
  exit 1
fi
destino_libexec="$appdir${libexec}"
mkdir -p "$destino_libexec"
# MiniBrowser no: es el ejemplo de WebKit y son 2 MiB que no hacen nada aquí.
for f in WebKitWebProcess WebKitNetworkProcess WebKitGPUProcess; do
  [ -e "$libexec/$f" ] && cp "$libexec/$f" "$destino_libexec/"
done
if [ -d "$libexec/injected-bundle" ]; then
  mkdir -p "$destino_libexec/injected-bundle"
  cp "$libexec/injected-bundle/"*.so "$destino_libexec/injected-bundle/"
fi

# **Y la pieza que hace que WebKit use ésos y no los del sistema.**
#
# Copiarlos dentro no es suficiente: la ruta va compilada dentro de libwebkit, así que sin esto
# se arrancan los de /usr del equipo de quien la ejecuta — y si no están, la ventana no llega a
# pintarse; y si están pero son de otra versión, el proceso aborta con «Received invalid
# message». El porqué entero está en la cabecera de `webkit-de-dentro.c`.
#
# Se compila aquí, en la máquina que arma, porque tiene que ser de la misma arquitectura que todo
# lo demás. Es lo mismo que decidió que el `.mcpb` lo armara un programa de Go: un compilador de C
# sí está garantizado en las dos patas de Linux, porque las dos compilan Esfinge con cgo.
cc="${CC:-cc}"
command -v "$cc" >/dev/null || { echo "✕ Hace falta un compilador de C ($cc) para armar la AppImage." >&2; exit 1; }
"$cc" -shared -fPIC -O2 -Wall -Wextra \
  -o "$appdir/usr/lib/esfinge-webkit-de-dentro.so" \
  "$(dirname "$0")/webkit-de-dentro.c" -ldl
echo "· la interposición de las rutas de WebKit, compilada"

# ---------------------------------------------------------------- los módulos de gio

# **Y los módulos de gio dentro, con su variable.**
#
# Sin esto, glib carga los **del sistema de quien la ejecuta** dentro de **nuestra glib**, y si su
# sistema es más nuevo revienta: lo vio el cliente en una Ubuntu 26.04 con la AppImage de ARM
# —empaquetada en 24.04—, con un «libgvfscommon.so: undefined symbol:
# g_variant_builder_init_static», que es un símbolo que su glib tiene y la nuestra no.
#
# Así que se copian los nuestros y se apunta GIO_MODULE_DIR ahí. Y si no hubiera ninguno, la
# variable se pone igual **a una carpeta vacía**: lo que hace falta es que glib **no** vaya a
# buscarlos a /usr. Lo que se pierde con ello es gvfs, que sirve para montar sistemas de ficheros
# remotos en los diálogos; Esfinge abre y guarda ficheros locales, así que no lo usa.
giomod=""
for d in /usr/lib/*/gio/modules; do
  [ -d "$d" ] && { giomod="$d"; break; }
done
if [ -n "$giomod" ]; then
  mkdir -p "$appdir$giomod"
  cp -a "$giomod/." "$appdir$giomod/" 2>/dev/null || true
  echo "· módulos de gio: $(ls -1 "$appdir$giomod" | wc -l) ficheros"
else
  giomod="/usr/lib/gio/modules"
  mkdir -p "$appdir$giomod"
  echo "· este equipo no tiene módulos de gio; la variable apuntará a una carpeta vacía"
fi

cat > "$appdir/apprun-hooks/esfinge-gio.sh" <<HOOK
# Los módulos de gio, los nuestros. Sin esta variable, glib carga los del sistema de quien ejecuta
# la AppImage dentro de nuestra glib, y con un sistema más nuevo que el que la empaquetó eso falla
# con un «undefined symbol» que no habla de nosotros.
export GIO_MODULE_DIR="\$APPDIR${giomod}"
HOOK

# Y el gancho que le dice a WebKit dónde han quedado. Va en `apprun-hooks/`, que es lo que
# el AppRun de linuxdeploy ejecuta antes de arrancar; el plugin de GTK pone otro al lado.
cat > "$appdir/apprun-hooks/esfinge-webkit.sh" <<HOOK
# WebKitGTK arranca procesos auxiliares que no son dependencias dinámicas de nadie. Los de dentro
# están copiados al lado de la biblioteca, pero **la ruta va compilada dentro de libwebkit**, así
# que sin lo de abajo se lanzan los del sistema de quien ejecuta la AppImage.
#
# Aquí hubo un \`WEBKIT_EXEC_PATH\` que no hacía nada: esa variable sólo la mira WebKit en
# compilaciones de desarrollo y ni siquiera está en la biblioteca que empaquetan las
# distribuciones. La que sí existe es la del paquete inyectado, y se queda.
export WEBKIT_INJECTED_BUNDLE_PATH="\$APPDIR${libexec}/injected-bundle"
export LD_PRELOAD="\$APPDIR/usr/lib/esfinge-webkit-de-dentro.so\${LD_PRELOAD:+:\$LD_PRELOAD}"
HOOK

# ---------------------------------------------------------------- empaquetar

echo "· linuxdeploy, con el plugin de GTK"
# De los auxiliares de WebKit hay que mirarles las dependencias —son programas, y lo que a
# ellos les falte no lo dice ningún `ldd` del nuestro—, pero **no moverlos**: tienen que
# quedarse donde apunta WEBKIT_EXEC_PATH.
#
# Eso es `--deploy-deps-only` y no `--executable`, que hace las dos cosas: con `--executable`
# los tres acabaron **además** copiados en usr/bin, que es basura y veinte megas. Lo dijo
# mirar dentro de la AppImage armada, no leer la ayuda.
"$herr/linuxdeploy-$arq" \
  --appdir "$appdir" \
  --executable "$appdir/usr/bin/esfinge-gui" \
  --deploy-deps-only "$destino_libexec" \
  --desktop-file "$appdir/usr/share/applications/esfinge.desktop" \
  --icon-file "build/linux/hicolor/256x256.png" --icon-filename esfinge \
  --plugin gtk

# **Y lo que a propósito NO va dentro: libEGL, libGL y lo que cuelga del driver gráfico.**
# `linuxdeploy` las excluye por su lista de siempre, y hace bien: esas tienen que ser las del
# equipo que la ejecuta o no habla con su tarjeta. Así que de las 157 bibliotecas que lleva,
# un `ldd` sobre la ventana y sobre los tres auxiliares deja `libEGL.so.1` «not found` en una
# máquina sin gráficos —este VPS— y la encuentra en cualquier escritorio. Si alguien la mete
# dentro «para que no falte», lo que consigue es que falle en los equipos con otro driver.

echo "· appimagetool"
# Con `--runtime-file`, appimagetool **no sale a la red**: usa el runtime que ya está
# comprobado aquí arriba en vez de bajarse uno sin mirar.
ARCH="$arq" "$herr/appimagetool-$arq" --runtime-file "$herr/runtime-$arq" "$appdir" "$salida"

# ---------------------------------------------------------------- y mirar dentro

# **Lo que la hace autosuficiente se comprueba abriendo el fichero**, no dando por hecho que
# los pasos de arriba salieron bien. Si falta cualquiera de estas cosas la AppImage arranca y
# **enseña una ventana en blanco**, que es el peor fallo posible porque parece que el
# programa está roto y no que le falte un fichero.
#
# Va aquí dentro y no en el flujo de publicar a propósito: lo mismo escrito en dos sitios se
# desalinea solo, y así lo comprueba también quien lo arme en un portátil.
echo "· mirando qué lleva dentro"
# **La ruta absoluta se resuelve aquí, antes de movernos.** La primera versión la resolvía
# dentro del subshell —después del `cd`—, así que con un `$salida` relativo, que es lo que
# pasa la publicación, `dirname` daba «.» y eso ya era el directorio temporal: buscaba la
# AppImage donde no estaba y el paso moría con «No such file or directory» **después** de
# haberla armado bien. Pasó en la 2.65.1. Aquí se probó con una ruta absoluta, que es el
# único caso que no tiene el fallo.
hecha="$(cd "$(dirname "$salida")" && pwd)/$(basename "$salida")"
revisar="$(mktemp -d)"
( cd "$revisar" && APPIMAGE_EXTRACT_AND_RUN=1 "$hecha" --appimage-extract >/dev/null )
dentro="$revisar/squashfs-root"
falta=()
for q in \
  apprun-hooks/esfinge-webkit.sh \
  apprun-hooks/esfinge-gio.sh \
  apprun-hooks/linuxdeploy-plugin-gtk.sh \
  usr/bin/esfinge-gui usr/bin/esfinge usr/bin/esfinge-puente usr/bin/esfinge-mcp \
  usr/lib/esfinge-webkit-de-dentro.so \
  usr/share/glib-2.0/schemas/gschemas.compiled; do
  [ -e "$dentro/$q" ] || falta+=("$q")
done
ls "$dentro"/usr/lib/*/webkit2gtk-4.1/WebKitWebProcess >/dev/null 2>&1 || falta+=("los procesos auxiliares de WebKitGTK")
ls "$dentro"/usr/lib/libwebkit2gtk-4.1.so.* >/dev/null 2>&1 || falta+=("libwebkit2gtk, que es lo que la hace autosuficiente")
# **Que esté no basta: tiene que cargarse.** El AppRun de linuxdeploy nombra los ganchos uno a uno
# en vez de recorrer la carpeta, así que un gancho que no acabe nombrado ahí se queda dentro sin
# ejecutarse nunca — y esto, sin ejecutarse, es un fichero de 16 KiB que no hace nada.
grep -q "esfinge-webkit-de-dentro.so" "$dentro/apprun-hooks/esfinge-webkit.sh" 2>/dev/null \
  || falta+=("el LD_PRELOAD de la interposición en apprun-hooks/esfinge-webkit.sh")
grep -q "esfinge-webkit.sh" "$dentro/AppRun" 2>/dev/null \
  || falta+=("que el AppRun cargue apprun-hooks/esfinge-webkit.sh")
rm -rf "$revisar"
if [ ${#falta[@]} -gt 0 ]; then
  echo "✕ La AppImage está armada pero no lleva dentro:" >&2
  printf '    %s\n' "${falta[@]}" >&2
  exit 1
fi

echo "✓ $salida · $(du -h "$salida" | cut -f1)"
