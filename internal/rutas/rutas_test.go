package rutas

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLasRutasCuelganDeLaCarpetaDeEsfinge(t *testing.T) {
	dir := t.TempDir()
	ApartarLaCarpeta(dir, t.Setenv)

	c := DeEsfinge()
	if c == "" {
		t.Fatal("no sabe cuál es la carpeta de Esfinge")
	}
	if !strings.HasPrefix(c, dir) {
		t.Fatalf("la carpeta es %q y la prueba la había apartado a %q", c, dir)
	}
	// Las tres clases de bóveda, cada una en su sitio y ninguna encima de otra.
	personal, proyecto, compartida := BovedaPrincipal(), Proyecto("abc123"), Compartida("dueno1", "abc123")
	if personal == proyecto || proyecto == compartida || personal == compartida {
		t.Fatalf("dos bóvedas en el mismo fichero: %q %q %q", personal, proyecto, compartida)
	}
	for _, r := range []string{personal, proyecto, compartida} {
		if !strings.HasPrefix(r, c) {
			t.Errorf("%q no está dentro de la carpeta de Esfinge", r)
		}
	}
	// **La compartida lleva de quién es**: dos personas pueden compartir bóvedas con
	// la misma referencia sin saber nada la una de la otra, y sin el dueño en el
	// nombre la segunda pisaría a la primera.
	if Compartida("dueno2", "abc123") == compartida {
		t.Error("dos personas con la misma referencia caen en el mismo fichero")
	}
	// Y la relativa del índice es la misma ruta sin la carpeta de delante.
	if filepath.Join(c, RelativaDeProyecto("abc123")) != proyecto {
		t.Errorf("la ruta relativa del índice (%q) no lleva al mismo sitio que %q",
			RelativaDeProyecto("abc123"), proyecto)
	}
}

// **Apartar la carpeta de configuración se hace en un solo sitio, y esto lo vigila
// en todo el repositorio.**
//
// Cada prueba que escribe en la carpeta del usuario tiene que apartarla, y son cinco
// variables porque `os.UserConfigDir` mira una distinta en cada sistema. Escribirlas
// a mano **sale bien en esta máquina y mal en Windows**, donde nadie mira: lo que se
// ve allí es una prueba quejándose de encontrarse una bóveda que no ha creado.
// Pasó, y **tiró una publicación**.
//
// Antes vivía en `internal/app` y miraba solo sus ficheros. Desde la ADR 0057 hay dos
// paquetes que apartan la carpeta, así que esto recorre **todos los ficheros de
// prueba del repositorio**: un paquete nuevo queda vigilado sin que nadie se acuerde
// de nada.
func TestLaCarpetaDeConfiguracionSeApartaEnUnSoloSitio(t *testing.T) {
	// Los que pueden nombrarla: el que la pone, y el equipo de cuentas de la ventana,
	// que elige su propia carpeta para poder tener dos equipos a la vez y lo dice en
	// su comentario.
	salvo := map[string]bool{
		filepath.Join("internal", "rutas", "rutas.go"):      true,
		filepath.Join("internal", "rutas", "rutas_test.go"): true,
		filepath.Join("internal", "app", "cuenta_test.go"):  true,
	}
	raiz := filepath.Join("..", "..")
	var sueltos []string
	err := filepath.WalkDir(raiz, func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "referencias", "empaquetado":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(raiz, ruta)
		if err != nil || salvo[rel] {
			return nil
		}
		crudo, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		if strings.Contains(string(crudo), `Setenv("XDG_CONFIG_HOME"`) {
			sueltos = append(sueltos, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range sueltos {
		t.Errorf("%s aparta la carpeta por su cuenta; usa rutas.ApartarLaCarpeta, "+
			"que es donde están los cinco nombres y por qué hacen falta", f)
	}
}
