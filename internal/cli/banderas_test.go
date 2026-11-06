package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// porElArbol ejecuta la línea de comandos **como lo hace el programa**: por la raíz
// y con `Execute`, no llamando a `RunE` a mano.
//
// Es la diferencia que decide si esta prueba vigila algo. `correr`, el ayudante de
// `escribir_test.go`, llama directamente a `cmd.RunE` — y así **el
// `PersistentPreRunE` no se ejecuta nunca**, que es justo la pieza que esto
// comprueba. Es la trampa del testigo puesto a mano otra vez: un ayudante que se
// salta el paso que hace producción deja ese paso sin probar.
func porElArbol(t *testing.T, args ...string) error {
	t.Helper()
	var o opciones
	raiz := laRaiz(&o, "pruebas")
	raiz.SetArgs(args)
	raiz.SetOut(&bytes.Buffer{})
	raiz.SetErr(&bytes.Buffer{})
	return raiz.Execute()
}

// **Una bandera puesta y vacía no es «no se ha dicho».**
//
// `--compartida ""` caía en el camino de la bóveda propia y contestaba «Guardada»:
// quien creyera estar escribiendo en la bóveda de un cliente escribía en la de su
// equipo, sin que nada lo dijera. Y pasa fuera de una prueba, porque en un servidor
// `--compartida "$PROYECTO"` con la variable sin definir es un despiste de un guion.
func TestUnaBanderaPuestaYVaciaEsUnErrorDeUso(t *testing.T) {
	for _, bandera := range []string{"compartida", "boveda", "clave-env", "clave-fichero"} {
		t.Run(bandera, func(t *testing.T) {
			err := porElArbol(t, "boveda", "listar", "--"+bandera, "")
			if err == nil {
				t.Fatalf("--%s puesta y vacía no dio error: Esfinge trabajaría con otra bóveda sin avisar", bandera)
			}
			if !strings.Contains(err.Error(), "--"+bandera) {
				t.Errorf("el error no dice cuál es la bandera: %v", err)
			}
			// **Y se dice que no es lo mismo que no decirla**, que es lo que convierte
			// un «no funciona» en algo que se arregla solo con leerlo.
			if !strings.Contains(err.Error(), "no es lo mismo que no decirla") {
				t.Errorf("el error no explica la diferencia: %v", err)
			}
			if codigoDe(err) != 2 {
				t.Errorf("un error de uso tiene que salir con 2, y salió con %d", codigoDe(err))
			}
		})
	}
}

// Y los dos orígenes del secreto, que son de la misma familia: `--secreto-env ""` se
// leería como que nadie dijo de dónde sale.
func TestLosOrigenesDelSecretoTampocoAdmitenElVacio(t *testing.T) {
	for _, bandera := range []string{"secreto-env", "secreto-fichero"} {
		t.Run(bandera, func(t *testing.T) {
			err := porElArbol(t, "boveda", "crear", "--titulo", "lo que sea", "--"+bandera, "")
			if err == nil || !strings.Contains(err.Error(), "--"+bandera) {
				t.Fatalf("--%s puesta y vacía no dio error de uso: %v", bandera, err)
			}
		})
	}
}

// **Y lo que de verdad cierra la familia: que nadie se salte la vigilancia.**
//
// Cobra ejecuta **un solo** `PersistentPreRunE`, el más cercano al comando que corre.
// Así que el día que un verbo de la bóveda defina el suyo —para cualquier otra cosa—
// esta comprobación deja de correr **en ese verbo y solo en ése**, en silencio. Es el
// mismo patrón que ya costó tres fallos aquí: añadir un caso obliga a acordarse de
// otro sitio, y acordarse no es defensa.
func TestNingunVerboDeLaBovedaSeSaltaLaVigilancia(t *testing.T) {
	var o opciones
	raiz := laRaiz(&o, "pruebas")
	var laBoveda *cobra.Command
	for _, c := range raiz.Commands() {
		if c.Name() == "boveda" {
			laBoveda = c
		}
	}
	if laBoveda == nil {
		t.Fatal("no está el comando «boveda»")
	}
	if laBoveda.PersistentPreRunE == nil {
		t.Fatal("«boveda» ya no vigila las banderas vacías")
	}
	var mirar func(c *cobra.Command)
	mirar = func(c *cobra.Command) {
		for _, h := range c.Commands() {
			if h.PersistentPreRunE != nil || h.PersistentPreRun != nil {
				t.Errorf("«%s» define su propio PersistentPreRun, así que se salta la vigilancia "+
					"de las banderas vacías: o llama a vigilarLasQueExigenValor, o esto deja de "+
					"protegerlo sin que nadie se entere", h.CommandPath())
			}
			mirar(h)
		}
	}
	mirar(laBoveda)
}

// Y que las que llevan **contenido** siguen admitiendo el vacío, porque vaciar un
// campo es para lo que sirven: `editar --notas ""` quita las notas. Sin esto, el
// arreglo de arriba se llevaría por delante la única forma de borrar un campo.
func TestLasBanderasDeContenidoSiAdmitenElVacio(t *testing.T) {
	var o opciones
	raiz := laRaiz(&o, "pruebas")
	for _, camino := range [][]string{{"boveda", "crear"}, {"boveda", "editar"}} {
		cmd, _, err := raiz.Find(camino)
		if err != nil {
			t.Fatalf("no encuentro «%s»: %v", strings.Join(camino, " "), err)
		}
		for _, n := range []string{"notas", "usuario", "carpeta"} {
			f := cmd.Flags().Lookup(n)
			if f == nil {
				continue
			}
			if f.Annotations[exigeValorAnotacion] != nil {
				t.Errorf("«%s --%s» no admite el vacío, y vaciar un campo es para lo que sirve",
					strings.Join(camino, " "), n)
			}
		}
	}
}
