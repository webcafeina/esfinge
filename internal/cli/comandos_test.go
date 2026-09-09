package cli

// Estos tests aparecen tarde y con motivo: `conSalida` acaba de dejar de
// escribir a mano para delegar en internal/escritura, y mover las tripas de algo
// que no tenía ni una prueba es exactamente como se rompen las cosas en
// silencio. `docs/deuda.md` llevaba la falta de pruebas de la línea de comandos
// como abierta; esto cubre la parte que se ha tocado.

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/salida"
	"github.com/webcafeina/esfinge/internal/tema"
)

func estilosDePrueba() salida.Estilos { return salida.NuevosEstilos(tema.TemaOscuro) }

func ponerTexto(s string) func(io.Writer) error {
	return func(w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	}
}

func TestConSalidaEscribeConPermisosDeSecreto(t *testing.T) {
	destino := filepath.Join(t.TempDir(), "salida.esf")
	o := &opciones{silencio: true}

	if err := conSalida(estilosDePrueba(), o, destino, ponerTexto("ESF1.loquesea")); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ESF1.loquesea" {
		t.Errorf("contenido: %q", b)
	}
	if runtime.GOOS == "windows" {
		return
	}
	info, _ := os.Stat(destino)
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("permisos: %o, se esperaba 600", got)
	}
}

// Pisar un fichero cifrado sin querer es perderlo, así que hace falta decirlo.
func TestConSalidaNoPisaSinForzar(t *testing.T) {
	destino := filepath.Join(t.TempDir(), "ya-existe.esf")
	if err := os.WriteFile(destino, []byte("lo de antes"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := conSalida(estilosDePrueba(), &opciones{silencio: true}, destino, ponerTexto("lo nuevo"))
	if err == nil {
		t.Fatal("ha pisado el fichero sin --forzar")
	}
	if !strings.Contains(err.Error(), "--forzar") {
		t.Errorf("el error no dice cómo seguir: %v", err)
	}
	if b, _ := os.ReadFile(destino); string(b) != "lo de antes" {
		t.Errorf("y encima lo ha tocado: %q", b)
	}

	if err := conSalida(estilosDePrueba(), &opciones{silencio: true, forzar: true}, destino,
		ponerTexto("lo nuevo")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(destino); string(b) != "lo nuevo" {
		t.Errorf("con --forzar no ha escrito: %q", b)
	}
}

// Si la escritura falla a medias, lo que había sigue entero. Es toda la razón de
// ser del temporal y el renombrado.
func TestConSalidaNoDejaElFicheroAMedias(t *testing.T) {
	dir := t.TempDir()
	destino := filepath.Join(dir, "credenciales.esf")
	if err := os.WriteFile(destino, []byte("lo bueno"), 0o600); err != nil {
		t.Fatal(err)
	}

	fallo := errors.New("se cortó")
	err := conSalida(estilosDePrueba(), &opciones{silencio: true, forzar: true}, destino,
		func(w io.Writer) error {
			io.WriteString(w, "basura")
			return fallo
		})
	if !errors.Is(err, fallo) {
		t.Fatalf("quiero el error de la escritura, tengo %v", err)
	}
	if b, _ := os.ReadFile(destino); string(b) != "lo bueno" {
		t.Errorf("el fichero de antes se ha estropeado: %q", b)
	}

	entradas, _ := os.ReadDir(dir)
	for _, e := range entradas {
		if strings.HasPrefix(e.Name(), ".esfinge-") {
			t.Errorf("ha quedado un temporal suelto: %s", e.Name())
		}
	}
}

// /dev/null y compañía: escribir un temporal al lado y renombrar encima no tiene
// sentido —ni permiso— sobre algo que no es un fichero.
func TestConSalidaAceptaLoQueNoEsUnFichero(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("aquí no hay /dev/null")
	}
	err := conSalida(estilosDePrueba(), &opciones{silencio: true}, "/dev/null", ponerTexto("da igual"))
	if err != nil {
		t.Errorf("no ha podido escribir en /dev/null: %v", err)
	}
}
