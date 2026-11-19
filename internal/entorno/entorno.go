// Package entorno contesta en qué escritorio corre Esfinge, que es lo que en
// Linux no dice el sistema operativo.
//
// # Por qué hace falta
//
// `runtime.GOOS` dice «linux» y con eso no se puede dibujar una ventana: GNOME y
// KDE organizan la suya de forma distinta —densidades, radios, dónde van los
// botones, quién pinta la barra de arriba— y una aplicación que los trata igual
// se nota en los dos. Hasta la 2.69.0 Esfinge enseñaba en Plasma el aspecto de
// GNOME, cabecera centrada de libadwaita incluida.
//
// # Por qué recibe la lista en vez de leerla
//
// Para poder comprobar los cuatro escritorios desde un VPS que no tiene ninguno.
// Es la forma que ya usa `rutas.ApartarLaCarpeta`: la decisión se escribe como
// una función del dato, y quien tiene el dato —el arranque— se lo pasa. Así una
// prueba no toca el entorno del proceso ni se queda a merced de en qué máquina
// corre.
package entorno

import (
	"os"
	"runtime"
	"strings"
)

// Los tres escritorios que Esfinge distingue.
//
// `Otro` no es un hueco: es una decisión. Xfce, Cinnamon, COSMIC y los demás se
// ven como GNOME, que es lo más parecido a un escritorio de GTK, en vez de
// inventar un cuarto aspecto para cada uno. Lo mismo hace Apolo.
const (
	GNOME = "gnome"
	KDE   = "kde"
	Otro  = "otro"
)

// Cual dice qué escritorio es, a partir de lo que trae `XDG_CURRENT_DESKTOP`.
//
// **Es una lista, no un nombre**, y por eso no vale compararla entera: en Ubuntu
// vale `ubuntu:GNOME`, en Budgie `Budgie:GNOME` y en una Fedora KDE, `KDE`. Se
// parte por `:` y se compara trozo a trozo sin distinguir mayúsculas.
//
// KDE se pregunta primero a propósito: una sesión que se declare de los dos
// —hay escritorios que listan GNOME por compatibilidad— es más KDE que GNOME,
// porque lo que la dibuja es KWin.
func Cual(lista string) string {
	tiene := func(nombre string) bool {
		for _, parte := range strings.Split(lista, ":") {
			if strings.EqualFold(strings.TrimSpace(parte), nombre) {
				return true
			}
		}
		return false
	}
	switch {
	case tiene("kde"):
		return KDE
	case tiene("gnome"):
		return GNOME
	default:
		return Otro
	}
}

// DeAqui es el escritorio de esta máquina, y **vacío fuera de Linux**.
//
// En macOS y en Windows no hay nada que preguntar: el sistema ya dice cómo se
// organiza una ventana suya, y devolver «otro» ahí sería un dato que invita a
// mirarlo cuando no significa nada.
func DeAqui() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	return Cual(os.Getenv("XDG_CURRENT_DESKTOP"))
}
