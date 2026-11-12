package cli

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

// TestLoQueLaAyudaPrometeExiste recorre el árbol de comandos y exige que **cada bandera
// que se nombra en la ayuda esté declarada** en ese comando o heredada de su padre.
//
// # Por qué existe
//
// Es la hermana de `TestLoQueElNucleoAplicaEstaEnElEsquema`, una planta más abajo: allí
// lo que no está declarado no existe para el modelo, y aquí lo que la ayuda promete y no
// existe **es un ejemplo que no se puede copiar**. Ha costado dos veces. `-q` en la raíz
// de «boveda», que fallaba con «unknown flag» justo en el programa que más se mete en un
// guion. Y `-o` en «boveda exportar» (2026-11-12), cuyo `Example` dice literalmente
// «esfinge boveda exportar -o copia.csv» y contestaba «unknown shorthand flag: 'o'».
//
// Las dos se encontraron **usándolas**, no leyendo. Y el `Example` es el peor sitio donde
// tener una mentira, porque es lo que la gente pega.
func TestLoQueLaAyudaPrometeExiste(t *testing.T) {
	var o opciones
	raiz := laRaiz(&o, "pruebas")

	// Lo que se busca son banderas escritas como se escriben: `--larga` o `-c` suelta.
	larga := regexp.MustCompile(`--[a-záéíóúñ][a-záéíóúñ0-9-]+`)
	corta := regexp.MustCompile(`(^|[\s"'(])-([a-zA-Z])($|[\s"'),.])`)

	// hay dice si esa bandera está declarada aquí, heredada o puesta como persistente por
	// este mismo comando. **Las tres cuentan**: `cmd.Flags()` no siempre trae las
	// persistentes del propio comando, y mirar solo ahí daba por ausente la `--clave-env`
	// de «boveda», que la declara él.
	hay := func(cmd *cobra.Command, nombre string) bool {
		return cmd.Flags().Lookup(nombre) != nil ||
			cmd.PersistentFlags().Lookup(nombre) != nil ||
			cmd.InheritedFlags().Lookup(nombre) != nil
	}
	hayCorta := func(cmd *cobra.Command, letra string) bool {
		return cmd.Flags().ShorthandLookup(letra) != nil ||
			cmd.PersistentFlags().ShorthandLookup(letra) != nil ||
			cmd.InheritedFlags().ShorthandLookup(letra) != nil
	}

	// deEsfinge deja de cada ejemplo **solo los trozos que invocan a Esfinge**. Sin esto,
	// `echo -n 'secreto' | esfinge cifrar` hacía reclamar una `-n` que es de `echo`: en un
	// ejemplo con tubería hay más de un programa, y las banderas de los demás no son
	// promesas nuestras.
	deEsfinge := func(ejemplo string) string {
		var vale []string
		for _, linea := range strings.Split(ejemplo, "\n") {
			for _, trozo := range strings.Split(linea, "|") {
				if strings.Contains(trozo, "esfinge") {
					vale = append(vale, trozo)
				}
			}
		}
		return strings.Join(vale, "\n")
	}

	// Y **una queja por bandera y comando**, no una por sitio donde se nombre: el mismo
	// `-o` sale en el `Long` y en el `Example`, y repetir el aviso es como se consigue que
	// deje de leerse.
	dicho := map[string]bool{}
	quejarse := func(cmd *cobra.Command, guion, nombre string) {
		clave := cmd.CommandPath() + " " + guion + nombre
		if dicho[clave] {
			return
		}
		dicho[clave] = true
		t.Errorf("«%s» promete %s%s en su ayuda y esa bandera no existe", cmd.CommandPath(), guion, nombre)
	}

	var recorrer func(cmd *cobra.Command)
	recorrer = func(cmd *cobra.Command) {
		// **El `Long` y el `Example` juntos**: el ejemplo es lo que se copia y el texto es
		// lo que se cree. Las dos cosas son promesas.
		texto := cmd.Long + "\n" + deEsfinge(cmd.Example)
		for _, m := range larga.FindAllString(texto, -1) {
			nombre := strings.TrimPrefix(m, "--")
			if !hay(cmd, nombre) {
				quejarse(cmd, "--", nombre)
			}
		}
		for _, m := range corta.FindAllStringSubmatch(texto, -1) {
			letra := m[2]
			if !hayCorta(cmd, letra) {
				quejarse(cmd, "-", letra)
			}
		}
		for _, hijo := range cmd.Commands() {
			recorrer(hijo)
		}
	}
	recorrer(raiz)
}

// TestLaAyudaTambienSeLlamaAyuda comprueba que `--ayuda` vale en todo el árbol.
//
// La bandera era solo `--help` en un programa donde **todas las demás frases están en
// español** y donde hay un subcomando `ayuda`: una asimetría que no se adivina. La
// encontraron **dos de cuatro agentes** probando el VPS, gastando turnos en `--ayuda` y en
// `boveda ayuda <verbo>`.
//
// Se recorre el árbol entero y no un par de comandos a mano: lo que se arregló es una
// normalización de la raíz que cobra propaga a los hijos, así que lo que hay que vigilar
// es precisamente que **llegue a todos** — incluidos los que se añadan mañana.
func TestLaAyudaTambienSeLlamaAyuda(t *testing.T) {
	var o opciones
	raiz := laRaiz(&o, "pruebas")

	// **En un hijo hay que mirar también las heredadas**, y esto costó una pasada en rojo:
	// `--help` lo declara la raíz como persistente, y `cmd.Flags()` de un hijo no las trae
	// hasta que cobra las funde al ejecutar. El binario funcionaba y la prueba decía que
	// «boveda ha perdido --help». Es la segunda vez el mismo día que los conjuntos de
	// banderas de cobra engañan al mirarlos antes de `Execute`.
	busca := func(cmd *cobra.Command, nombre string) *pflag.Flag {
		for _, f := range []*pflag.Flag{
			cmd.Flags().Lookup(nombre),
			cmd.PersistentFlags().Lookup(nombre),
			cmd.InheritedFlags().Lookup(nombre),
		} {
			if f != nil {
				return f
			}
		}
		return nil
	}

	var recorrer func(cmd *cobra.Command)
	recorrer = func(cmd *cobra.Command) {
		// `--ayuda` se normaliza a `help`, así que lo que se busca es que **la encuentre**.
		if f := busca(cmd, "ayuda"); f == nil {
			t.Errorf("«%s» no entiende --ayuda", cmd.CommandPath())
		} else if f.Name != "help" {
			t.Errorf("«%s»: --ayuda apunta a --%s y no a --help", cmd.CommandPath(), f.Name)
		}
		// Y la canónica sigue siendo la de siempre, que es la que sale en la ayuda.
		if busca(cmd, "help") == nil {
			t.Errorf("«%s» ha perdido --help", cmd.CommandPath())
		}
		for _, hijo := range cmd.Commands() {
			recorrer(hijo)
		}
	}
	recorrer(raiz)
}
