// Escribe los tokens de color y medidas que consumen la ventana y el panel de la
// extensión.
//
// La fuente de verdad es internal/tema, que es donde viven los tests de
// contraste. Este programa solo vuelca lo que aquel decide.
//
// **Son dos ficheros y no uno**: la ventana se lleva la base y un bloque por
// sistema, y el panel de la extensión solo la base, porque allí `data-sistema`
// no existe ni puede existir. Ver tema.Ficheros.
package main

import (
	"fmt"
	"os"

	"github.com/webcafeina/esfinge/internal/tema"
)

func main() {
	for _, f := range tema.Ficheros {
		if err := os.WriteFile(f.Ruta, []byte(f.CSS()), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "No he podido escribir", f.Ruta+":", err)
			os.Exit(1)
		}
		fmt.Println("Escrito", f.Ruta)
	}
}
