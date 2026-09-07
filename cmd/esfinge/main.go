// Esfinge cifra y descifra secretos con una clave.
//
// Este es el ejecutable de línea de comandos, para tuberías y scripts. La
// interfaz con ventana es otra aplicación, en cmd/esfinge-gui, y las dos
// comparten el mismo núcleo y el mismo formato de contenedor.
package main

import (
	"os"

	"github.com/webcafeina/esfinge/internal/cli"
)

// version la pone el enlazador al compilar: -ldflags "-X main.version=..."
var version = "dev"

func main() {
	os.Exit(cli.Ejecutar(version))
}
