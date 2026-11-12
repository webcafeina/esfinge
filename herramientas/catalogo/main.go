// Programa catalogo: lo que el catálogo de adjuntos le contesta a quien no es Go.
//
// Existe porque el catálogo vive en `internal/adjuntos` —por la lección de `zip`: Go es
// lo único que seguro está en las cinco máquinas de la publicación— y quien pregunta son
// un guion de bash, un YAML y alguna receta de empaquetado.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/webcafeina/esfinge/internal/adjuntos"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, uso)
		os.Exit(2)
	}
	if err := correr(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "✕", err)
		os.Exit(1)
	}
}

const uso = `Uso:
  catalogo lista <versión>              los nombres que tienen que estar publicados
  catalogo nombre <clave> <versión>     el nombre de uno
  catalogo para <sistema> <arq> <gestor>  las claves que le valen a ese equipo
  catalogo json <versión>               todo, para una receta de empaquetado
  catalogo tabla-readme <versión>       la tabla de descargas del README
  catalogo cuerpo-release <versión>     el cuerpo de la publicación`

func correr(verbo string, args []string) error {
	switch verbo {
	case "lista":
		if len(args) != 1 {
			return fmt.Errorf("falta la versión")
		}
		for _, n := range adjuntos.Publicados(args[0]) {
			fmt.Println(n)
		}
	case "nombre":
		if len(args) != 2 {
			return fmt.Errorf("hacen falta la clave y la versión")
		}
		n, err := adjuntos.Nombre(args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Println(n)
	case "para":
		if len(args) != 3 {
			return fmt.Errorf("hacen falta el sistema, la arquitectura y el gestor")
		}
		for _, c := range adjuntos.ParaSistema(args[0], args[1], args[2]) {
			fmt.Println(c)
		}
	case "json":
		if len(args) != 1 {
			return fmt.Errorf("falta la versión")
		}
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		return e.Encode(adjuntos.De(args[0]))
	case "tabla-readme":
		if len(args) != 1 {
			return fmt.Errorf("falta la versión")
		}
		fmt.Print(adjuntos.TablaDelLeeme(args[0]))
	case "cuerpo-release":
		if len(args) != 1 {
			return fmt.Errorf("falta la versión")
		}
		fmt.Print(adjuntos.CuerpoDeLaPublicacion(args[0]))
	default:
		return fmt.Errorf("no sé qué es «%s».\n%s", verbo, strings.TrimPrefix(uso, "Uso:\n"))
	}
	return nil
}
