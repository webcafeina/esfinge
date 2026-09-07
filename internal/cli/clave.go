package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/ui"
)

// OrigenClave reúne las tres formas de dar la clave, en orden de precedencia.
//
// Ninguna de ellas es un argumento de la línea de comandos, y es deliberado: lo
// que se escribe en la línea queda en el historial del shell y lo puede leer
// cualquiera con un `ps` mientras el proceso corre.
type OrigenClave struct {
	Variable string // --clave-env NOMBRE
	Fichero  string // --clave-fichero ruta
}

// Leer obtiene la clave. Cuando toca preguntar y se está cifrando, la pide dos
// veces: una errata al cifrar no da error, da un contenedor que no se abre nunca.
func (o OrigenClave) Leer(e ui.Estilos, confirmar bool) ([]byte, error) {
	switch {
	case o.Variable != "":
		v, ok := os.LookupEnv(o.Variable)
		if !ok {
			return nil, fmt.Errorf("La variable de entorno %s no existe", o.Variable)
		}
		if v == "" {
			return nil, fmt.Errorf("La variable de entorno %s está vacía", o.Variable)
		}
		return []byte(v), nil

	case o.Fichero != "":
		b, err := os.ReadFile(o.Fichero)
		if err != nil {
			return nil, fmt.Errorf("No puedo leer la clave de %s: %w", o.Fichero, err)
		}
		// Un fichero de clave escrito a mano acaba casi siempre con un salto de
		// línea que nadie quiso poner ahí.
		b = bytes.TrimRight(b, "\r\n")
		if len(b) == 0 {
			return nil, fmt.Errorf("El fichero de clave %s está vacío", o.Fichero)
		}
		return b, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New(
			"No hay clave y no puedo preguntarla: usa --clave-env VARIABLE o --clave-fichero RUTA")
	}
	return preguntar(e, confirmar)
}

func preguntar(e ui.Estilos, confirmar bool) ([]byte, error) {
	clave, err := leerOculta(e, "Clave")
	if err != nil {
		return nil, err
	}
	if len(clave) == 0 {
		return nil, errors.New("La clave no puede estar vacía")
	}

	if confirmar {
		f := cripto.Evaluar(string(clave))
		fmt.Fprintln(os.Stderr, "  "+e.Medidor(f.Nivel, 4, 16)+" "+e.Apagado.Render(f.Etiqueta))
		if f.Sugerencia != "" {
			fmt.Fprintln(os.Stderr, "  "+e.Ojo(f.Sugerencia))
		}

		otra, err := leerOculta(e, "Repite la clave")
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(clave, otra) {
			return nil, errors.New("Las dos claves no coinciden")
		}
	}
	return clave, nil
}

func leerOculta(e ui.Estilos, etiqueta string) ([]byte, error) {
	// El prompt va a stderr para no contaminar la salida cuando stdout es una
	// tubería.
	fmt.Fprint(os.Stderr, e.Acento.Render(ui.GlifoBarra)+" "+e.Cuerpo.Render(etiqueta)+e.Apagado.Render(": "))

	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("No he podido leer la clave: %w", err)
	}
	return []byte(strings.TrimRight(string(b), "\r\n")), nil
}
