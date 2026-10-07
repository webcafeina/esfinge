// Arma el paquete instalable de Claude Desktop (.mcpb) para una plataforma.
//
//	go run ./herramientas/armar-mcpb <versión> <binario> <salida.mcpb>
//
// Un `.mcpb` es **un zip**: no hace falta la herramienta de Anthropic para armarlo, y
// no depender de ella es lo de siempre en esta casa —una dependencia menos en el camino
// de publicar, que es donde más caro sale que algo falle—. Lo que sí hay que respetar
// es la forma, y está en `empaquetado/mcpb/LÉEME.md`.
//
// **Y está en Go y no en `sh` por una razón concreta: `zip` no existe en la máquina
// Windows de GitHub.** El guion anterior murió ahí con `zip: command not found` y tiró
// una publicación con todo lo demás en verde. Lo que sí hay en los tres trabajos es Go,
// porque los tres compilan Esfinge. Las alternativas eran peor: `Compress-Archive` de
// PowerShell es **otro camino solo para un sistema** —o sea el que nadie prueba— y
// además **no guarda el permiso de ejecución**, que aquí hace falta en los otros dos.
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "uso: armar-mcpb <versión> <binario> <salida.mcpb>")
		os.Exit(2)
	}
	if err := armar(os.Args[1], os.Args[2], os.Args[3]); err != nil {
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
	fmt.Println("  " + filepath.Base(os.Args[3]))
}

func armar(version, binario, salida string) error {
	raiz, err := raizDelRepo()
	if err != nil {
		return err
	}

	// La versión del manifiesto es la de la publicación, no una escrita a mano que se
	// quedaría vieja sin que nadie lo notara.
	manifiesto, err := os.ReadFile(filepath.Join(raiz, "empaquetado", "mcpb", "manifest.json"))
	if err != nil {
		return err
	}
	puesta := bytes.Replace(manifiesto,
		[]byte(`"version": "0.0.0"`),
		[]byte(`"version": "`+version+`"`), 1)
	if bytes.Equal(puesta, manifiesto) {
		return fmt.Errorf("no se ha podido poner la versión %q en el manifiesto", version)
	}

	icono, err := os.ReadFile(filepath.Join(raiz, "build", "appicon.png"))
	if err != nil {
		return err
	}
	servidor, err := os.ReadFile(binario)
	if err != nil {
		return err
	}

	// **El nombre de dentro no cambia entre plataformas**: lo dice el manifiesto, y en
	// Windows el `.exe` lo añade Claude Desktop por su cuenta.
	dentro := "server/esfinge-mcp"
	if strings.HasSuffix(binario, ".exe") {
		dentro += ".exe"
	}

	if err := os.MkdirAll(filepath.Dir(salida), 0o755); err != nil {
		return err
	}
	f, err := os.Create(salida)
	if err != nil {
		return err
	}
	defer f.Close()

	z := zip.NewWriter(f)
	poner := func(nombre string, datos []byte, modo os.FileMode) error {
		cab := &zip.FileHeader{Name: nombre, Method: zip.Deflate}
		cab.SetMode(modo)
		w, err := z.CreateHeader(cab)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, bytes.NewReader(datos))
		return err
	}
	if err := poner("manifest.json", puesta, 0o644); err != nil {
		return err
	}
	if err := poner("icon.png", icono, 0o644); err != nil {
		return err
	}
	// **Ejecutable, y eso hay que decirlo aquí.** Lo que se instala en macOS y en Linux
	// sale de este zip, y un servidor sin el bit puesto no arranca.
	if err := poner(dentro, servidor, 0o755); err != nil {
		return err
	}
	if err := z.Close(); err != nil {
		return err
	}
	return f.Close()
}

// raizDelRepo sube desde este fichero hasta el `go.mod`, para que dé igual desde dónde
// se llame.
func raizDelRepo() (string, error) {
	aqui, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(aqui, "go.mod")); err == nil {
			return aqui, nil
		}
		padre := filepath.Dir(aqui)
		if padre == aqui {
			return "", fmt.Errorf("no encuentro la raíz del repositorio desde %s", aqui)
		}
		aqui = padre
	}
}
