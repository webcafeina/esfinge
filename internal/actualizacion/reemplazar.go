package actualizacion

import (
	"fmt"
	"os"
	"os/exec"
)

// El cambiazo no lo puede hacer la propia aplicación: mientras corre, su paquete
// está en uso y en Windows sus ficheros están bloqueados. Así que se deja
// preparado un guion, se lanza suelto —sin quedar colgando de este proceso— y
// acto seguido la ventana se cierra. El guion espera a que el proceso muera,
// cambia lo viejo por lo nuevo y vuelve a abrir Esfinge.
//
// **Lo que no puede pasar nunca es quedarse sin aplicación.** Por eso lo viejo no
// se borra hasta que lo nuevo está en su sitio, y si algo falla se devuelve lo
// viejo a donde estaba.

// reemplazarseEnMac saca el .app de la imagen de disco y lo pone en el lugar del
// que se está ejecutando.
func reemplazarseEnMac(dmg string) error {
	paquete, err := paqueteDeLaApp()
	if err != nil {
		return err
	}

	guion, err := escribirGuion("actualizar-*.sh", fmt.Sprintf(`#!/bin/sh
# Cambia Esfinge por la versión recién descargada. Lo escribe Esfinge al
# actualizarse y se borra solo al terminar.
set -e

dmg=%[1]q
paquete=%[2]q
pid=%[3]d
monte=$(mktemp -d)
nueva=$(mktemp -d)
viejo="$paquete.reemplazado"

limpiar() {
  hdiutil detach "$monte" -quiet 2>/dev/null || true
  rm -rf "$monte" "$nueva"
}
trap limpiar EXIT

# Esperar a que la ventana se cierre de verdad. Sin esto el cambiazo pillaría al
# proceso vivo y macOS dejaría el paquete a medias.
i=0
while kill -0 "$pid" 2>/dev/null && [ $i -lt 100 ]; do
  sleep 0.1
  i=$((i + 1))
done

hdiutil attach "$dmg" -nobrowse -readonly -quiet -mountpoint "$monte"

# ditto y no cp: conserva los atributos extendidos y la firma del paquete, que
# con cp se pierden y macOS lo rechaza.
ditto "$monte/Esfinge.app" "$nueva/Esfinge.app"
hdiutil detach "$monte" -quiet

# Comprobar que lo que se ha sacado es una aplicación de verdad antes de tocar la
# que funciona.
[ -x "$nueva/Esfinge.app/Contents/MacOS/Esfinge" ] || exit 1

# La cuarentena: si la imagen la trajera marcada, el paquete instalado heredaría
# el aviso de Gatekeeper. Se quita aquí, que es donde se sabe que el fichero lo
# ha descargado Esfinge y no un navegador.
xattr -dr com.apple.quarantine "$nueva/Esfinge.app" 2>/dev/null || true

# El cambiazo, en el orden que no deja a nadie sin aplicación.
rm -rf "$viejo"
mv "$paquete" "$viejo"
if ! ditto "$nueva/Esfinge.app" "$paquete"; then
  mv "$viejo" "$paquete"
  exit 1
fi
rm -rf "$viejo"

open "$paquete"
rm -f "$0"
`, dmg, paquete, os.Getpid()))
	if err != nil {
		return err
	}
	return lanzarSuelto("/bin/sh", guion)
}

// reemplazarseEnWindows deja que el instalador NSIS haga lo suyo en silencio y
// vuelve a abrir la aplicación.
//
// El instalador ya sabe reemplazar la versión anterior y registrar los .esf; lo
// único que no puede es hacerlo con Esfinge abierta. Pedirá permiso de
// administrador, que es del sistema y no hay forma de saltárselo: se instala para
// toda la máquina.
func reemplazarseEnWindows(instalador string) error {
	yo, err := os.Executable()
	if err != nil {
		return err
	}

	guion, err := escribirGuion("actualizar-*.cmd", fmt.Sprintf(`@echo off
rem Cambia Esfinge por la versión recién descargada. Lo escribe Esfinge al
rem actualizarse y se borra solo al terminar.

rem Esperar a que la ventana se cierre: mientras corre, sus ficheros están
rem bloqueados y el instalador no podría reemplazarlos.
for /l %%%%i in (1,1,50) do (
  tasklist /fi "PID eq %[3]d" 2>nul | find "%[3]d" >nul || goto :instalar
  timeout /t 1 /nobreak >nul
)

:instalar
"%[1]s" /S
start "" "%[2]s"
del "%%~f0"
`, instalador, yo, os.Getpid()))
	if err != nil {
		return err
	}
	return lanzarSuelto("cmd", "/c", guion)
}

// escribirGuion deja el guion en un fichero temporal, ejecutable y solo para su
// dueño: lleva rutas de esta máquina y va a correr con los permisos de quien usa
// Esfinge.
func escribirGuion(patron, contenido string) (string, error) {
	f, err := os.CreateTemp("", patron)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(contenido); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(f.Name(), 0o700); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// lanzarSuelto arranca el guion de forma que sobreviva a la muerte de Esfinge,
// que es justo lo que va a pasar un segundo después.
func lanzarSuelto(nombre string, args ...string) error {
	orden := exec.Command(nombre, args...)
	// Desde la carpeta temporal: si se quedara dentro del paquete que va a
	// reemplazar, lo estaría sujetando mientras lo cambian.
	orden.Dir = os.TempDir()
	ponerEnSuPropioGrupo(orden)
	if err := orden.Start(); err != nil {
		return fmt.Errorf("No se ha podido preparar la actualización: %w", err)
	}
	// No se espera: el guion tiene que seguir vivo cuando este proceso ya no lo
	// esté. Liberar el proceso hijo evita dejar un zombi mientras tanto.
	return orden.Process.Release()
}
