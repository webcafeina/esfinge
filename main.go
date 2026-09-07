//go:build !dev

// Esfinge, con ventana.
//
// La misma herramienta que la línea de comandos, el mismo núcleo y el mismo
// formato de contenedor; lo único distinto es que aquí hay ratón, diálogos del
// sistema y sitio donde soltar ficheros.
//
// Está en la raíz del proyecto y no en cmd/, que sería lo idiomático en Go,
// porque Wails genera los enlaces con la interfaz buscando el paquete main
// justo donde está wails.json. Con el punto de entrada en cmd/esfinge-gui, la
// compilación falla con «no Go files». La línea de comandos sí vive en cmd/.
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
	// argumento, y con eso la aplicación abre directamente en descifrar. En macOS
	// no llega así sino por un evento de Apple, que se recoge más abajo en
	// Mac.OnFileOpen.
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		aplicacion.AlAbrirCon(os.Args[1])
	}

	// El contexto no existe hasta que Wails arranca, y el menú se construye
	// antes: se le pasa una función que lo consulta cuando hace falta, que es
	// siempre después de que la ventana esté abierta.
	var ctx context.Context

	err := wails.Run(&options.App{
		Title: "Esfinge",

		// La barra de menús, en español. Ver menu.go: los roles de Wails traen
		// los rótulos en inglés escritos a fuego, así que se construye entera.
		Menu: menuEnEspanol(func() context.Context { return ctx }, aplicacion),

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

		OnStartup: func(c context.Context) {
			ctx = c
			escritorio.Arrancar(c)
			aplicacion.Arrancar(c)
		},

		Bind: []any{aplicacion},

		Mac: &mac.Options{
			// Doble clic en un .esf. macOS no pasa el fichero como argumento
			// —lo entrega por un evento de Apple— y esto es lo que lo recoge.
			OnFileOpen: func(ruta string) {
				aplicacion.AlAbrirCon(ruta)
				escritorio.Avisar(app.EventoFicheroAbierto, ruta)
			},
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
