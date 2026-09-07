package tui

import (
	"fmt"
	"io"
)

// AltoComodo son las filas que necesita la pantalla más alta —el formulario de
// cifrar, con sus tres campos enmarcados— para verse entera sin desplazar nada,
// más un poco de margen.
const AltoComodo = 30

// PedirAltura le pide al terminal que agrande su ventana hasta que quepa todo.
//
// La secuencia es CSI 8 ; filas ; columnas t, de las operaciones de ventana de
// xterm, y con un cero en las columnas se deja el ancho como está. La entienden
// iTerm2, Terminal.app, gnome-terminal, konsole, kitty, alacritty y Windows
// Terminal.
//
// Hay terminales que la ignoran, y algunos la traen desactivada a propósito
// —dejar que un programa mueva la ventana es una puerta que no todo el mundo
// quiere abierta—. Por eso esto es una petición y no un requisito: si nadie
// contesta, la interfaz sigue funcionando y lo que sobra se desplaza, que es
// para lo que está el desplazamiento.
func PedirAltura(w io.Writer, filas int) {
	if filas < 1 {
		return
	}
	fmt.Fprintf(w, "\x1b[8;%d;0t", filas)
}
