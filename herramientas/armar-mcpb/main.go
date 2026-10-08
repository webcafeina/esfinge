// Arma el paquete instalable de Claude Desktop (.mcpb) para una plataforma.
//
//	go run ./herramientas/armar-mcpb <versión> <binario> <salida.mcpb>
//
// Lo de verdad está en `internal/mcpb`, que es el mismo código que usa la ventana para
// el botón de Ajustes: **el paquete que se publica y el que se guarda desde Esfinge
// tienen que ser el mismo**, o el día que uno se arregle el otro se queda roto.
//
// **Y está en Go y no en `sh` por una razón concreta: `zip` no existe en la máquina
// Windows de GitHub.** El guion anterior murió ahí con `zip: command not found` y tiró
// una publicación con todo lo demás en verde. Lo que sí hay en los tres trabajos es Go,
// porque los tres compilan Esfinge. Las alternativas eran peor: `Compress-Archive` de
// PowerShell es **otro camino solo para un sistema** —o sea el que nadie prueba— y
// además **no guarda el permiso de ejecución**, que aquí hace falta en los otros dos.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/webcafeina/esfinge/internal/mcpb"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "uso: armar-mcpb <versión> <binario> <salida.mcpb>")
		os.Exit(2)
	}
	version, binario, salida := os.Args[1], os.Args[2], os.Args[3]

	if err := os.MkdirAll(filepath.Dir(salida), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
	f, err := os.Create(salida)
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
	if err := mcpb.Armar(version, binario, f); err != nil {
		f.Close()
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
	fmt.Println("  " + filepath.Base(salida))
}
