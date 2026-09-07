//go:build !dev

package app

import (
	"context"

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

// filtros de los diálogos. Se ofrecen, no se imponen: quien quiera cifrar un
// fichero sin extensión conocida tiene que poder.
var (
	filtroTodos    = runtime.FileFilter{DisplayName: "Todos los ficheros", Pattern: "*.*"}
	filtroCifrados = runtime.FileFilter{DisplayName: "Cifrados de Esfinge (*.esf)", Pattern: "*.esf"}
)

func (e *Escritorio) ElegirFicheros(titulo, desde string, varios bool) ([]string, error) {
	opciones := runtime.OpenDialogOptions{
		Title:            titulo,
		DefaultDirectory: desde,
		Filters:          []runtime.FileFilter{filtroTodos},
	}
	// Al descifrar se ofrece primero el filtro de contenedores, que es lo que se
	// va a buscar el noventa y nueve por ciento de las veces.
	if titulo == "Elige qué descifrar" {
		opciones.Filters = []runtime.FileFilter{filtroCifrados, filtroTodos}
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
