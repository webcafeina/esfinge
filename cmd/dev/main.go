//go:build dev

// Levanta la interfaz contra el Go de verdad, sin necesitar entorno gráfico.
//
// Es lo que permite probar la aplicación entera en una máquina sin escritorio:
// el navegador hace de ventana y este servidor hace de puente. Nunca entra en lo
// que se distribuye.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/webcafeina/esfinge/internal/app"
)

func main() {
	direccion := flag.String("direccion", "127.0.0.1:34443", "Dónde escuchar")
	carpeta := flag.String("carpeta", "", "Carpeta que hace de diálogo de ficheros")
	version := flag.String("version", "dev", "Versión que enseña la interfaz")
	api := flag.String("api", "", "API de publicaciones de mentira, para probar el aviso de versión nueva")
	flag.Parse()

	if *carpeta == "" {
		tmp, err := os.MkdirTemp("", "esfinge-dev-")
		if err != nil {
			log.Fatal(err)
		}
		*carpeta = tmp
		// Un par de ficheros para poder probar el camino de ficheros sin tener que
		// prepararlos a mano cada vez.
		os.WriteFile(filepath.Join(tmp, "credenciales.env"),
			[]byte("DATABASE_URL=postgres://u:p@h/db\nAPI_KEY=abc123\n"), 0o644)
		os.WriteFile(filepath.Join(tmp, "notas.txt"), []byte("un secreto cualquiera\n"), 0o644)
	}

	sistema := app.NuevoSistemaDeDesarrollo(*carpeta)
	aplicacion := app.Nueva(*version, sistema)

	// Con -api se apunta la comprobación de versiones a un servidor de mentira.
	// Sin él no se toca: nadie quiere que levantar la interfaz para desarrollar
	// salga a internet.
	if *api != "" {
		app.ApuntarAAPI(aplicacion, *api)
	} else {
		app.ApuntarAAPI(aplicacion, "http://127.0.0.1:1") // a ninguna parte
	}

	log.Fatal(app.Servir(aplicacion, sistema, *direccion))
}
