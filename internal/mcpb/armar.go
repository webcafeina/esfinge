// Package mcpb arma el paquete instalable de Claude Desktop, un `.mcpb`.
//
// **Vive aquí y no en el guion de la publicación porque la ventana también lo arma.**
// Mandar a alguien a la página de descargas a por un fichero que su propio Esfinge
// ya lleva dentro es pedirle que elija bien el sistema y acierte con la versión;
// desde Ajustes es un botón, y sale siempre el de la versión que tiene puesta.
//
// Un `.mcpb` es **un zip** con tres cosas: el manifiesto, un icono y el binario. No
// hace falta la herramienta de Anthropic —una dependencia menos en el camino de
// publicar, que es donde más caro sale que algo falle— y lo que hay que respetar está
// en `empaquetado/mcpb/LÉEME.md`.
package mcpb

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
)

// **El manifiesto y el icono van embebidos**, porque dentro de la aplicación no hay
// repositorio del que leerlos.
//
// `go:embed` no puede salir del directorio de su paquete, así que el icono es una
// copia de `build/appicon.png`. Una copia a ciegas ya costó una versión aquí —el icono
// del documento era `appicon.png` byte a byte sin que nadie lo notara—, así que
// `TestElIconoEsElDeLaAplicacion` compara los dos ficheros y se pone rojo si se
// separan.
var (
	//go:embed manifest.json
	manifiesto []byte
	//go:embed icono.png
	icono []byte
)

// LaVersionDePrueba es lo que lleva el manifiesto del repositorio, para que se vea que
// nunca se publica tal cual: la pone quien arma.
const LaVersionDePrueba = "0.0.0"

// Armar escribe el paquete de `version` con `binario` dentro.
//
// `binario` es la ruta del servidor MCP de **esta** plataforma; si acaba en `.exe`, el
// de dentro también, que es lo que espera Claude Desktop en Windows.
func Armar(version, binario string, destino io.Writer) error {
	servidor, err := os.ReadFile(binario)
	if err != nil {
		return fmt.Errorf("no se encuentra el servidor MCP: %w", err)
	}
	return ArmarCon(version, NombreDeDentro(binario), servidor, destino)
}

// NombreDeDentro es cómo se llama el servidor dentro del paquete.
//
// **No cambia entre plataformas salvo por el `.exe`**: lo dice el manifiesto, y en
// Windows Claude Desktop se lo añade por su cuenta.
func NombreDeDentro(binario string) string {
	if strings.HasSuffix(strings.ToLower(binario), ".exe") {
		return "server/esfinge-mcp.exe"
	}
	return "server/esfinge-mcp"
}

// ArmarCon es Armar con el binario ya leído, que es lo que necesita la ventana.
func ArmarCon(version, dentro string, servidor []byte, destino io.Writer) error {
	// La versión del manifiesto es la de quien arma, no una escrita a mano que se
	// quedaría vieja sin que nadie lo notara.
	puesta := bytes.Replace(manifiesto,
		[]byte(`"version": "`+LaVersionDePrueba+`"`),
		[]byte(`"version": "`+version+`"`), 1)
	if bytes.Equal(puesta, manifiesto) {
		return fmt.Errorf("no se ha podido poner la versión %q en el manifiesto", version)
	}

	z := zip.NewWriter(destino)
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
	// **Ejecutable, y eso hay que decirlo aquí.** Lo que se instala en macOS y en
	// Linux sale de este zip, y un servidor sin el bit puesto no arranca.
	if err := poner(dentro, servidor, 0o755); err != nil {
		return err
	}
	return z.Close()
}
