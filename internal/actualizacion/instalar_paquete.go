package actualizacion

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/webcafeina/esfinge/internal/adjuntos"
)

// Instalar el paquete de Linux desde la propia ventana.
//
// # Por qué esto existe, y por qué la ADR 0016 se quedó corta
//
// La 0016 dijo que en Linux Esfinge no se reemplaza a sí misma «porque el paquete instala
// como root». Lo primero es cierto y **lo segundo no es una razón suficiente**: para eso
// está `pkexec`, que le pide la autorización al escritorio con su propio diálogo y ejecuta
// la orden como root. Apolo lo hace así desde octubre y el cliente lo usó en la misma
// máquina virtual donde ha probado esto, así que no es una hipótesis.
//
// Lo que había mientras tanto —bajar el paquete y abrirlo con `xdg-open`— funciona, pero
// deja al Centro de aplicaciones terminar el trabajo: tres pasos más, y la frase «Instalar
// y reiniciar» que la banda enseña en macOS y en Windows allí no era verdad.
//
// # Lo que no cambia
//
// **Una AppImage sigue sin reemplazarse sola** —eso es la otra mitad de la 0016 y está en
// `deuda.md` con lo que costaría—, y **en Arch no se toca nada**: ahí no publicamos un
// paquete suelto, así que la banda dice que lo actualiza su gestor y se calla.
//
// # La orden
//
// Se arma por tabla y **recibiendo qué programas hay**, no preguntándoselo al sistema donde
// corre: así los cuatro casos se comprueban desde este VPS, que es la misma razón por la
// que las rutas de Windows se calculan para un sistema dado y no para `runtime.GOOS`.
//
// Se prefiere el gestor de paquetes al instalador crudo —`apt-get` antes que `dpkg`, `dnf`
// antes que `rpm`— porque una versión nueva puede traer una dependencia nueva: `dpkg -i`
// la dejaría a medias con un «dependency problems», y `apt-get install` la resuelve. El
// crudo se queda como último recurso, que es mejor que no ofrecer nada.
func ordenParaInstalar(gestor, ruta string, hay func(string) bool) []string {
	// Sin `pkexec` no hay forma de pedir permiso sin una terminal, y pedirlo en una
	// terminal que nadie está mirando es no pedirlo.
	if !hay("pkexec") {
		return nil
	}
	switch gestor {
	case adjuntos.GestorDeb:
		switch {
		case hay("apt-get"):
			// `--allow-downgrades` no es para bajar de versión por gusto: es lo que
			// permite reinstalar la misma o volver atrás si alguna vez hiciera falta,
			// y sin ello apt se niega y el error no dice gran cosa.
			return []string{"pkexec", "apt-get", "install", "-y", "--allow-downgrades", ruta}
		case hay("dpkg"):
			return []string{"pkexec", "dpkg", "-i", ruta}
		}
	case adjuntos.GestorRPM:
		switch {
		case hay("dnf"):
			return []string{"pkexec", "dnf", "install", "-y", ruta}
		case hay("zypper"):
			// openSUSE, que es el otro sistema al que va el .rpm (ADR 0061).
			return []string{"pkexec", "zypper", "--non-interactive", "install", "--allow-unsigned-rpm", ruta}
		case hay("rpm"):
			return []string{"pkexec", "rpm", "-U", ruta}
		}
	}
	// AppImage, pacman y quien se bajó el tar.gz: aquí no se instala nada.
	return nil
}

// hayPrograma dice si algo se puede ejecutar. Variable para que las pruebas no dependan de
// lo que haya instalado en la máquina que las corre.
var hayPrograma = func(programa string) bool {
	_, err := exec.LookPath(programa)
	return err == nil
}

// sePuedeInstalarSola dice si en esta máquina la banda puede terminar el trabajo.
func sePuedeInstalarSola() bool {
	return ordenParaInstalar(gestorQueLoInstalo(), "paquete", hayPrograma) != nil
}

// ejecutar lanza la orden y espera. Variable por lo mismo de arriba: una prueba no puede
// instalar paquetes de verdad.
var ejecutar = func(orden []string) error {
	cmd := exec.Command(orden[0], orden[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	return cmd.Run()
}

// instalarPaqueteDeLinux instala lo descargado y deja preparado el reinicio.
//
// Se hace **con la ventana todavía abierta**, a propósito: el diálogo de `pkexec` sale
// encima de Esfinge y se entiende de quién es. Hacerlo después de cerrar dejaría un
// diálogo de contraseña huérfano, que es exactamente lo que un gestor de contraseñas
// enseña a no creerse.
func instalarPaqueteDeLinux(ruta string) error {
	orden := ordenParaInstalar(gestorQueLoInstalo(), ruta, hayPrograma)
	if orden == nil {
		return fmt.Errorf("Esta copia no se puede actualizar sola")
	}

	if err := ejecutar(orden); err != nil {
		var salida *exec.ExitError
		if errors.As(err, &salida) {
			// 126 es lo que devuelve `pkexec` cuando no se autoriza —se cancela el
			// diálogo o la contraseña no vale—, y 127 cuando no ha podido ni
			// intentarlo. Un «exit status 126» no le dice nada a nadie.
			switch salida.ExitCode() {
			case 126:
				return fmt.Errorf("No se ha dado la autorización para instalar la actualización")
			case 127:
				return fmt.Errorf("No se ha podido pedir la autorización para instalar la actualización")
			}
		}
		return fmt.Errorf("No se ha podido instalar la actualización: %w", err)
	}

	return reabrir()
}

// reabrir es el relevo: se deja en una variable para que una prueba pueda decir si se ha
// pedido **sin** que se lance de verdad un guion que espera a que muera el proceso de la
// prueba y vuelve a ejecutarlo.
var reabrir = reabrirseEnLinux

// reabrirseEnLinux deja un guion que espera a que esta ventana muera y vuelve a abrir
// Esfinge. El binario ya es el nuevo: lo acaba de poner el gestor de paquetes en la misma
// ruta, así que no hay nada que mover.
func reabrirseEnLinux() error {
	yo, err := os.Executable()
	if err != nil {
		// Instalar sí ha funcionado, y eso es lo que importa: no se puede volver a
		// abrir sola, y la persona la abrirá cuando quiera.
		return nil
	}

	guion, err := escribirGuion("reabrir-*.sh", fmt.Sprintf(`#!/bin/sh
# Vuelve a abrir Esfinge cuando la ventana vieja se haya cerrado. Lo escribe Esfinge al
# actualizarse y se borra solo al terminar.
pid=%[2]d
esfinge=%[1]q

# Esperar a que el proceso muera de verdad, con tope: si algo lo dejara colgado, mejor
# abrir una ventana de más que no abrir ninguna.
i=0
while kill -0 "$pid" 2>/dev/null && [ "$i" -lt 100 ]; do
  sleep 0.2
  i=$((i + 1))
done

rm -f "$0"
exec "$esfinge"
`, yo, os.Getpid()))
	if err != nil {
		return nil
	}
	if err := lanzarSuelto(guion); err != nil {
		return nil
	}
	return nil
}
