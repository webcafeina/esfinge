package tui

import (
	"encoding/base64"
	"errors"

	"github.com/atotto/clipboard"
)

// Copiar deja un texto en el portapapeles por dos caminos a la vez, porque
// ninguno de los dos cubre todos los casos.
//
//   - El portapapeles del sistema (pbcopy en macOS, la API de Windows, xclip o
//     xsel en Linux). Es el que funciona cuando Esfinge corre en la máquina de
//     quien está mirando la pantalla, que es lo normal. En Linux sin xclip ni
//     xsel instalados, falla.
//   - OSC 52, una secuencia de escape que le pide el copiado al propio terminal.
//     Es la única que funciona a través de SSH —el portapapeles del sistema
//     copiaría en la máquina remota, que no es donde está la persona— pero hay
//     terminales que no la implementan, y el Terminal.app de macOS es uno.
//
// Entre los dos cubren lo que hay. Se emiten siempre los dos: cuál acierta
// depende de dónde se esté ejecutando, y no cuesta nada intentarlo.
func Copiar(texto string) error {
	if texto == "" {
		return errors.New("No hay nada que copiar")
	}
	return clipboard.WriteAll(texto)
}

// secuenciaOSC52 construye la petición de copiado dirigida al terminal.
//
// Se devuelve como texto para incrustarla en el dibujado: es una secuencia de
// escape, así que el terminal se la come sin pintar nada, y ese es el único
// hueco por el que se puede colar mientras Bubble Tea tiene tomada la pantalla.
func secuenciaOSC52(texto string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(texto)) + "\x07"
}
