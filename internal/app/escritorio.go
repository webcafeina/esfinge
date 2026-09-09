//go:build !dev

package app

import (
	"context"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Escritorio conecta la aplicación con el sistema de verdad: los diálogos de
// abrir y guardar son los de macOS y Windows, no unos dibujados por nosotros.
// Es la diferencia entre una ventana que se siente del sistema y una que no.
type Escritorio struct {
	ctx context.Context
}

// NuevoEscritorio se construye antes de que Wails tenga contexto; el contexto
// llega después, en Arrancar.
func NuevoEscritorio() *Escritorio { return &Escritorio{} }

func (e *Escritorio) Arrancar(ctx context.Context) { e.ctx = ctx }

// filtrosDe traduce lo que se busca a lo que entiende el diálogo de cada
// sistema, **que no es lo mismo en los tres y ahí estaba el fallo**.
//
// En macOS los filtros no son una lista desplegable: son la única lista de
// extensiones que el panel deja seleccionar, y todo lo demás sale en gris. Y
// Wails los prepara quitándoles el `*.` por delante, así que el `*.*` de «todos
// los ficheros» —que en Windows es lo idiomático y ahí funciona— llegaba a macOS
// convertido en una extensión llamada literalmente `*`, que no tiene ningún
// fichero. Resultado: el panel se abría **sin dejar elegir nada**.
//
// Nadie lo había visto porque en macOS los ficheros se arrastran a la ventana, y
// ese camino no pasa por aquí. Importar de otro gestor no tiene arrastrar y
// soltar, así que fue lo primero que se dio de bruces con ello.
//
// Por eso en macOS **no se manda ningún filtro**: sin filtros Wails llama a
// `setAllowsOtherFileTypes:true` y el panel acepta lo que sea, que es lo que hace
// falta. Se pierde el resaltado de los `.esf`, que era una comodidad; no se
// pierde poder trabajar, que no lo es.
func filtrosDe(filtro Filtro) []runtime.FileFilter {
	return filtrosPara(goruntime.GOOS, filtro)
}

// filtrosPara es lo mismo con el sistema como argumento, que es lo único que
// permite comprobar los tres desde una sola máquina. Sin esta costura, la regla
// de macOS —la que estaba mal— no se puede probar en ninguna parte.
func filtrosPara(sistema string, filtro Filtro) []runtime.FileFilter {
	if sistema == "darwin" {
		return nil
	}

	// El patrón de «cualquier fichero» tampoco es el mismo: en Windows es `*.*` y
	// en GTK hace falta `*`, porque `*.*` deja fuera los que no tienen extensión.
	todos := runtime.FileFilter{DisplayName: "Todos los ficheros", Pattern: "*"}
	if sistema == "windows" {
		todos.Pattern = "*.*"
	}

	// Y en los dos el filtro es una lista desplegable, así que ofrecer lo probable
	// primero no le quita a nadie la posibilidad de elegir otra cosa.
	switch filtro {
	case FiltroCifrados:
		return []runtime.FileFilter{
			{DisplayName: "Cifrados de Esfinge (*.esf)", Pattern: "*.esf"},
			todos,
		}
	case FiltroTablas:
		return []runtime.FileFilter{
			{DisplayName: "Exportaciones (*.csv)", Pattern: "*.csv"},
			todos,
		}
	default:
		return []runtime.FileFilter{todos}
	}
}

func (e *Escritorio) ElegirFicheros(titulo, desde string, varios bool, filtro Filtro) ([]string, error) {
	opciones := runtime.OpenDialogOptions{
		Title:            titulo,
		DefaultDirectory: desde,
		Filters:          filtrosDe(filtro),
	}

	if !varios {
		uno, err := runtime.OpenFileDialog(e.ctx, opciones)
		if err != nil || uno == "" {
			return nil, err
		}
		return []string{uno}, nil
	}
	return runtime.OpenMultipleFilesDialog(e.ctx, opciones)
}

func (e *Escritorio) ElegirDondeGuardar(titulo, nombreSugerido, desde string) (string, error) {
	return runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:            titulo,
		DefaultFilename:  nombreSugerido,
		DefaultDirectory: desde,
	})
}

// Cerrar pide a Wails que termine, que es lo que deja libre el paquete para que
// el guion de actualización pueda reemplazarlo.
func (e *Escritorio) Cerrar() {
	if e.ctx == nil {
		return
	}
	runtime.Quit(e.ctx)
}

// Avisar manda un evento a la ventana.
func (e *Escritorio) Avisar(evento string, datos any) {
	if e.ctx == nil {
		return
	}
	runtime.EventsEmit(e.ctx, evento, datos)
}

// PonerEnPortapapeles y LeerPortapapeles usan el runtime de Wails, que es quien
// habla con el portapapeles del sistema. La interfaz no puede: el navegador sabe
// copiar, pero no borrar pasado un rato de forma fiable.
func (e *Escritorio) PonerEnPortapapeles(texto string) error {
	if e.ctx == nil {
		return nil
	}
	return runtime.ClipboardSetText(e.ctx, texto)
}

func (e *Escritorio) LeerPortapapeles() (string, error) {
	if e.ctx == nil {
		return "", nil
	}
	return runtime.ClipboardGetText(e.ctx)
}
