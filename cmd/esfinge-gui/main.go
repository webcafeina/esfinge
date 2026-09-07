//go:build !dev

// Esfinge, con ventana.
//
// La misma herramienta que la línea de comandos, el mismo núcleo y el mismo
// formato de contenedor; lo único distinto es que aquí hay ratón, diálogos del
// sistema y sitio donde soltar ficheros.
package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"github.com/webcafeina/esfinge/internal/app"
	"github.com/webcafeina/esfinge/internal/interfaz"
)

// version la pone el enlazador al compilar.
var version = "dev"

func main() {
	escritorio := app.NuevoEscritorio()
	aplicacion := app.Nueva(version, escritorio)

	// Doble clic en un .esf: en Windows y en Linux el sistema pasa la ruta como
	// argumento, y con eso la aplicación abre directamente en descifrar.
	//
	// En macOS no llega así, sino por un evento de Apple que Wails v2 no expone.
	// Ahí la asociación queda declarada —el Finder enseña el icono y ofrece abrir
	// con Esfinge— pero el fichero hay que arrastrarlo o elegirlo. Está dicho en
	// el README para no prometer lo que no hace.
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		aplicacion.AlAbrirCon(os.Args[1])
	}

	err := wails.Run(&options.App{
		Title: "Esfinge",

		// La ventana arranca con sitio para lo más alto que hay —cifrar con sus
		// tres campos y el resultado— sin obligar a desplazarse nada más abrir.
		Width:     900,
		Height:    720,
		MinWidth:  560,
		MinHeight: 480,

		AssetServer: &assetserver.Options{Assets: interfaz.Recursos},

		// EnableFileDrop entrega las rutas absolutas de lo que se suelte, que es
		// lo único que sirve: el arrastrar y soltar del navegador da un objeto sin
		// ruta, y por ahí no se llega al fichero desde Go.
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},

		OnStartup: func(ctx context.Context) {
			escritorio.Arrancar(ctx)
			aplicacion.Arrancar(ctx)
		},

		Bind: []any{aplicacion},

		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			About: &mac.AboutInfo{
				Title:   "Esfinge",
				Message: "Cifra y descifra secretos con una clave.\nDe Webcafeína.",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
