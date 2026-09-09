//go:build !dev

package main

import (
	"context"
	"runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/webcafeina/esfinge/internal/app"
)

// menuEnEspanol arma la barra de menús de los tres sistemas.
//
// **Por qué está escrito a mano y no se usan los roles de Wails.** Wails trae
// roles —AppMenu, EditMenu, WindowMenu— que producen los menús de siempre, pero
// con los rótulos **en inglés escritos a fuego en su Objective-C**
// (`WailsMenu.m`: «Hide Others», «Paste and Match Style», «Quit Esfinge»). No
// son los menús que localiza macOS, así que no hay forma de traducirlos desde
// fuera: o se usan en inglés o se construyen. Aquí se construyen.
//
// La consecuencia es que las acciones de edición hay que hacerlas nosotros: sin
// los roles no hay selectores nativos detrás. Se resuelve mandando una orden a
// la ventana, que es quien sabe qué campo tiene el foco y qué hay seleccionado.
// El pegar es el único que necesita las dos mitades: el portapapeles del sistema
// lo lee Go —el navegador no puede— y el texto lo coloca la interfaz.
func menuEnEspanol(ctx func() context.Context, aplicacion *app.App) *menu.Menu {
	// pedir manda una orden a la ventana y no hace nada más.
	pedir := func(que string) func(*menu.CallbackData) {
		return func(*menu.CallbackData) { aplicacion.Ordenar(que) }
	}

	barra := menu.NewMenu()

	// En macOS el primer menú es siempre el de la aplicación, lleve el nombre que
	// lleve: el sistema pone ahí «Esfinge». En Windows y Linux es un menú más, y
	// por eso se llama «Archivo», que es lo que espera quien viene de allí.
	primero := "Esfinge"
	if runtime.GOOS != "darwin" {
		primero = "Archivo"
	}
	aplicacionMenu := barra.AddSubmenu(primero)
	aplicacionMenu.AddText("Acerca de Esfinge", nil, pedir(app.OrdenIrAAjustes))
	aplicacionMenu.AddText("Buscar actualizaciones…", nil, pedir(app.OrdenBuscarVersion))
	aplicacionMenu.AddSeparator()
	if runtime.GOOS == "darwin" {
		aplicacionMenu.AddText("Ocultar Esfinge", keys.CmdOrCtrl("h"), func(*menu.CallbackData) {
			wruntime.Hide(ctx())
		})
		aplicacionMenu.AddSeparator()
	}
	aplicacionMenu.AddText("Salir", keys.CmdOrCtrl("q"), func(*menu.CallbackData) {
		wruntime.Quit(ctx())
	})

	edicion := barra.AddSubmenu("Edición")
	edicion.AddText("Deshacer", keys.CmdOrCtrl("z"), pedir(app.OrdenDeshacer))
	edicion.AddText("Rehacer", keys.Combo("z", keys.CmdOrCtrlKey, keys.ShiftKey), pedir(app.OrdenRehacer))
	edicion.AddSeparator()
	edicion.AddText("Cortar", keys.CmdOrCtrl("x"), pedir(app.OrdenCortar))
	edicion.AddText("Copiar", keys.CmdOrCtrl("c"), pedir(app.OrdenCopiar))
	// El pegar no se puede delegar entero en la ventana: leer el portapapeles
	// desde el navegador está prohibido sin un permiso que aquí no hay a quién
	// pedirle. Lo lee Go y manda el texto.
	edicion.AddText("Pegar", keys.CmdOrCtrl("v"), func(*menu.CallbackData) {
		texto, err := wruntime.ClipboardGetText(ctx())
		if err != nil {
			return
		}
		aplicacion.OrdenarPegar(texto)
	})
	edicion.AddSeparator()
	edicion.AddText("Seleccionar todo", keys.CmdOrCtrl("a"), pedir(app.OrdenSeleccionarTodo))

	ver := barra.AddSubmenu("Ver")
	ver.AddText("Cifrar", keys.CmdOrCtrl("1"), pedir(app.OrdenIrACifrar))
	ver.AddText("Descifrar", keys.CmdOrCtrl("2"), pedir(app.OrdenIrADescifrar))
	ver.AddText("Generar", keys.CmdOrCtrl("3"), pedir(app.OrdenIrAGenerar))
	ver.AddText("Bóveda", keys.CmdOrCtrl("4"), pedir(app.OrdenIrABoveda))
	ver.AddText("Historial", keys.CmdOrCtrl("5"), pedir(app.OrdenIrAHistorial))
	ver.AddText("Ajustes", keys.CmdOrCtrl("6"), pedir(app.OrdenIrAAjustes))

	ventana := barra.AddSubmenu("Ventana")
	ventana.AddText("Minimizar", keys.CmdOrCtrl("m"), func(*menu.CallbackData) {
		wruntime.WindowMinimise(ctx())
	})
	ventana.AddText("Pantalla completa", nil, func(*menu.CallbackData) {
		if wruntime.WindowIsFullscreen(ctx()) {
			wruntime.WindowUnfullscreen(ctx())
			return
		}
		wruntime.WindowFullscreen(ctx())
	})

	ayuda := barra.AddSubmenu("Ayuda")
	ayuda.AddText("Documentación de Esfinge", nil, func(*menu.CallbackData) {
		wruntime.BrowserOpenURL(ctx(), "https://github.com/webcafeina/esfinge")
	})
	ayuda.AddText("Qué protege y qué no", nil, func(*menu.CallbackData) {
		wruntime.BrowserOpenURL(ctx(),
			"https://github.com/webcafeina/esfinge/blob/main/docs/seguridad.md")
	})

	return barra
}
