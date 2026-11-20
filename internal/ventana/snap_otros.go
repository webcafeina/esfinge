//go:build !windows

// Package ventana es lo que hay que hacer con la ventana y no cabe en la
// interfaz. Fuera de Windows no hay nada que hacer: macOS y GNOME dibujan sus
// botones, y lo que hace Snap allí lo hace el escritorio.
package ventana

// Montar no hace nada aquí. Existe para que `main.go` la llame sin preguntar
// por el sistema: una etiqueta de compilación que excluye un sistema es una
// promesa, y aquí no se ve si no se cumple.
func Montar(clase string, alto, ancho int, alAvisar func(string)) {}
