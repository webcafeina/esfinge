#!/bin/sh
# Refresca las bases de datos del escritorio para que el lanzador aparezca en el
# menú y los .esf salgan con su icono sin tener que reiniciar la sesión.
#
# Y deja escrito **con qué gestor se instaló esto**, que es lo único que Esfinge no
# puede averiguar sola: el binario de un .deb y el de un .rpm viven los dos en
# /usr/bin, así que mirar la ruta no distingue. Sin este dato, a una Fedora se le
# ofrecería un .deb al actualizarse. Lo lee gestorQueLoInstalo(), en
# internal/actualizacion/abrir.go.
#
# Va escrito a pelo y **cada empaquetador tiene su propio guion** (nfpm los elige con
# `overrides`), porque este fichero se copia dentro del paquete y se ejecuta en el
# equipo de quien instala: una variable de entorno de aquí no llegaría allí, y
# adivinarlo en el equipo del cliente —mirando si hay `rpm` o `dpkg`— es exactamente la
# suposición que este fichero viene a quitar.
set -e
mkdir -p /usr/share/esfinge
printf 'rpm\n' > /usr/share/esfinge/gestor
chmod 644 /usr/share/esfinge/gestor
if command -v update-mime-database >/dev/null 2>&1; then
    update-mime-database /usr/share/mime || true
fi
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor || true
fi
