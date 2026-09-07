// Esfinge cifra y descifra secretos con una clave.
//
// Sin argumentos abre una interfaz de menús; con argumentos se comporta como un
// comando corriente y se deja meter en una tubería.
package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/webcafeina/esfinge/internal/cli"
	"github.com/webcafeina/esfinge/internal/tui"
	"github.com/webcafeina/esfinge/internal/ui"
)

// version la pone el enlazador al compilar: -ldflags "-X main.version=..."
var version = "dev"

func main() {
	os.Exit(cli.Ejecutar(version, abrirTUI))
}

func abrirTUI(e ui.Estilos, redimensionar bool) error {
	// La interfaz de menús necesita un terminal de verdad en las dos direcciones.
	// Sin él —una tubería, un CI, una tarea de fondo— lo que toca es la ayuda, no
	// un programa que parece colgado esperando teclas que nunca llegan.
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return errors.New("No hay terminal interactivo: prueba con «esfinge --help»")
	}

	// Si la ventana viene pequeña, se le pide que se agrande hasta que quepa la
	// pantalla más alta. Es una petición: hay terminales que no la atienden, y
	// entonces la interfaz se desplaza como siempre.
	//
	// Al terminar se devuelve la ventana a su tamaño, que es lo mínimo: nadie
	// abre un programa esperando que le deje el terminal de otro tamaño.
	if _, filas, err := term.GetSize(int(os.Stdout.Fd())); redimensionar && err == nil && filas < tui.AltoComodo {
		tui.PedirAltura(os.Stdout, tui.AltoComodo)
		defer tui.PedirAltura(os.Stdout, filas)
	}

	// WithMouseCellMotion recoge clics y movimiento del puntero, que es lo que
	// hace falta para los botones y para que se resalte lo que hay debajo del
	// cursor. No se usa WithMouseAllMotion, que además captura el arrastre y se
	// come la selección de texto del propio terminal.
	p := tea.NewProgram(tui.Nuevo(e, version),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("La interfaz ha fallado: %w", err)
	}
	return nil
}
