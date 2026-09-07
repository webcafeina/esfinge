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
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

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
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			aplicacion.AlAbrirCon(arg)
		}
	}

	// El contexto no existe hasta que Wails arranca, y el menú se construye
	// antes: se le pasa una función que lo consulta cuando hace falta, que es
	// siempre después de que la ventana esté abierta.
	var ctx context.Context

	// El vidrio del sistema: macOS lo da con NSVisualEffectView y Windows 11 con
	// Mica. Linux no lo ofrece en Wails, y ahí la ventana se queda opaca.
	//
	// Va acompañado de un fondo transparente, porque si el webview deja pasar la
	// luz el color lo tiene que poner el CSS. Eso es lo que la interfaz consulta
	// con Vidrio(): sin él dejaría la ventana vacía justo donde no hay efecto.
	vidrio := runtime.GOOS == "darwin" || runtime.GOOS == "windows"
	app.MarcarVidrio(aplicacion, vidrio)

	err := wails.Run(&options.App{
		Title: "Esfinge",

		// La barra de menús, en español. Ver menu.go: los roles de Wails traen
		// los rótulos en inglés escritos a fuego, así que se construye entera.
		Menu: menuEnEspanol(func() context.Context { return ctx }, aplicacion),

		// Con barra lateral la ventana ya no puede encogerse tanto: 225 px se los
		// lleva la navegación, y al contenido hay que dejarle sitio para el panel
		// de 560 sin que se estruje.
		Width:     980,
		Height:    720,
		MinWidth:  760,
		MinHeight: 520,

		AssetServer: &assetserver.Options{Assets: interfaz.Recursos},

		// Transparente para que se vea el vidrio de detrás. Donde no hay vidrio
		// esto no se llega a poner: ver más abajo.
		BackgroundColour: fondoDeLaVentana(vidrio),

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

			// El remate del vidrio en macOS, que Wails deja a medias: la ventana
			// se queda opaca y el material no tiene nada que mezclar. Va aquí
			// porque hasta ahora no existía la ventana. Ver vidrio_darwin.go.
			if vidrio {
				ponerElVidrio()
			}
		},

		Bind: []any{aplicacion},

		Mac: &mac.Options{
			// Doble clic en un .esf. macOS no pasa el fichero como argumento
			// —lo entrega por un evento de Apple— y esto es lo que lo recoge.
			OnFileOpen: func(ruta string) {
				aplicacion.AlAbrirCon(ruta)
				escritorio.Avisar(app.EventoFicheroAbierto, ruta)
			},
			// Sin barra de título propia: el contenido llega hasta arriba y los
			// semáforos quedan encima de la barra lateral. Es lo que hacen Finder,
			// Correo y los Ajustes del sistema.
			//
			// **Hidden y no HiddenInset.** El «Inset» activa `UseToolbar`, y
			// entonces macOS dibuja **su propia banda de barra de herramientas**
			// justo donde va nuestro título: se ve un fondo que no cuadra con el
			// resto de la ventana. Aquí la barra de herramientas la dibujamos
			// nosotros, así que no queremos la del sistema.
			//
			// A cambio, **hay que declarar por dónde se arrastra la ventana** o se
			// queda clavada en la pantalla. Eso va en el CSS, con
			// «--wails-draggable» en la barra lateral y la de herramientas.
			TitleBar: mac.TitleBarHidden(),

			// El vidrio de macOS. WebviewIsTransparent deja pasar la luz a través
			// de la página, y WindowIsTranslucent pone detrás la vista de efecto
			// del sistema, que es la que desenfoca el escritorio de verdad.
			WebviewIsTransparent: vidrio,
			WindowIsTranslucent:  vidrio,
			About: &mac.AboutInfo{
				Title:   "Esfinge",
				Message: "Cifra y descifra secretos con una clave.\nDe Webcafeína.",
			},
		},
		Windows: &windows.Options{
			// Mica es el equivalente en Windows 11: toma el fondo de escritorio,
			// muy desenfocado y quieto. En Windows 10 no existe y la ventana se
			// queda opaca, sin error y sin aviso, que es lo correcto.
			WebviewIsTransparent: vidrio,
			WindowIsTranslucent:  vidrio,
			BackdropType:         windows.Mica,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// fondoDeLaVentana es transparente cuando hay vidrio y nada cuando no lo hay.
//
// Devolver nil deja que Wails ponga el fondo de siempre, que es justo lo que
// tiene que pasar en Linux: un fondo transparente sin vidrio detrás no enseña el
// escritorio, enseña un agujero.
func fondoDeLaVentana(vidrio bool) *options.RGBA {
	if !vidrio {
		return nil
	}
	return options.NewRGBA(0, 0, 0, 0)
}
