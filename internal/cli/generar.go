package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/cripto"
)

func comandoGenerar(o *opciones) *cobra.Command {
	var (
		bytes    int
		alfabeto string
		cuantas  int
	)

	cmd := &cobra.Command{
		Use:   "generar",
		Short: "Genera contraseñas al azar",
		Long: "Genera contraseñas con entropía de crypto/rand.\n\n" +
			"Por defecto en hexadecimal, que es el único alfabeto que se puede meter\n" +
			"dentro de una cadena de conexión sin pensárselo. Los otros dos avisan.",
		Example: "  esfinge generar\n" +
			"  esfinge generar --bytes 32 -n 5\n" +
			"  esfinge generar --alfabeto alnum",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := estilos(o.tema)
			if err != nil {
				return err
			}

			a, ok := cripto.Alfabetos[alfabeto]
			if !ok {
				return fmt.Errorf("El alfabeto «%s» no existe: %s", alfabeto, alfabetosDisponibles())
			}
			if cuantas < 1 || cuantas > 100 {
				return fmt.Errorf("El número de contraseñas tiene que estar entre 1 y 100")
			}

			claves := make([]string, cuantas)
			for i := range claves {
				p, err := cripto.Generar(a, bytes)
				if err != nil {
					return err
				}
				claves[i] = p
			}

			// A una tubería van las contraseñas y nada más. El aviso no se pierde
			// por eso: sale por stderr, que es donde la persona lo sigue viendo
			// aunque la salida se esté redirigiendo a un fichero. Perder justo ese
			// aviso sería perder el único que importa.
			if !aTerminal(os.Stdout) {
				for _, p := range claves {
					fmt.Println(p)
				}
				// Sin condición de terminal, a diferencia de la marca: si alguien
				// redirige stderr a un fichero de registro, el aviso tiene que
				// acabar en el registro, no evaporarse.
				if a.Aviso != "" && !o.silencio {
					fmt.Fprintln(os.Stderr, e.Ojo(a.Aviso))
				}
				marcaEnStderr(e, o)
				return nil
			}

			cuerpo := e.Etiq(a.Nombre+" · "+fmt.Sprintf("%d bits", bytes*8)) + "\n\n" +
				e.Codigo.Render(strings.Join(claves, "\n"))
			fmt.Println()
			fmt.Println(e.Panel.Render(cuerpo))

			if a.Aviso != "" {
				fmt.Println("  " + e.Ojo(a.Aviso))
			} else {
				fmt.Println("  " + e.Info("Segura dentro de una URL"))
			}
			marca(e, o, os.Stdout)
			return nil
		},
	}

	cmd.Flags().IntVar(&bytes, "bytes", 24, "bits de entropía, en bytes (24 = 192 bits)")
	cmd.Flags().StringVar(&alfabeto, "alfabeto", cripto.AlfHex.Nombre, "hex, alnum o simbolos")
	cmd.Flags().IntVarP(&cuantas, "cuantas", "n", 1, "cuántas generar")
	cmd.Flags().BoolVarP(&o.silencio, "silencio", "q", false, "sin marca ni mensajes de cortesía")
	return cmd
}

func alfabetosDisponibles() string {
	nombres := make([]string, 0, len(cripto.Alfabetos))
	for n := range cripto.Alfabetos {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	return strings.Join(nombres, ", ")
}
