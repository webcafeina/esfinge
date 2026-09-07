// Escribe los tokens de color y medidas que consume la interfaz.
//
// La fuente de verdad es internal/tema, que es donde viven los tests de
// contraste. Este programa solo vuelca lo que aquel decide.
package main

import (
	"fmt"
	"os"

	"github.com/webcafeina/esfinge/internal/tema"
)

func main() {
	destino := "frontend/src/tokens.css"
	if len(os.Args) > 1 {
		destino = os.Args[1]
	}
	if err := os.WriteFile(destino, []byte(tema.GenerarCSS()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "No he podido escribir", destino+":", err)
		os.Exit(1)
	}
	fmt.Println("Escrito", destino)
}
