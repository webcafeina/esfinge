package escritura

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func escribirTexto(s string) func(io.Writer) error {
	return func(w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	}
}

func TestEscribeYDejaLosPermisosBien(t *testing.T) {
	destino := filepath.Join(t.TempDir(), "secreto.txt")
	if err := Atomica(destino, Opciones{}, escribirTexto("hola")); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hola" {
		t.Errorf("contenido: %q", b)
	}

	if runtime.GOOS == "windows" {
		t.Skip("los permisos de Unix no significan lo mismo aquí")
	}
	info, err := os.Stat(destino)
	if err != nil {
		t.Fatal(err)
	}
	// 0600 por defecto y no 0644: lo que escribe Esfinge es un secreto o está a
	// un paso de serlo.
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("permisos: %o, se esperaba 600", got)
	}
}

// Lo que da nombre al paquete: si la escritura falla, lo que había sigue entero.
func TestSiFallaAMediasNoTocaLoQueHabia(t *testing.T) {
	dir := t.TempDir()
	destino := filepath.Join(dir, "boveda")
	if err := Atomica(destino, Opciones{}, escribirTexto("lo bueno")); err != nil {
		t.Fatal(err)
	}

	fallo := errors.New("se cortó a la mitad")
	err := Atomica(destino, Opciones{}, func(w io.Writer) error {
		io.WriteString(w, "basura a medio")
		return fallo
	})
	if !errors.Is(err, fallo) {
		t.Fatalf("quiero el error de la escritura, tengo %v", err)
	}

	b, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "lo bueno" {
		t.Errorf("el fichero de antes se ha estropeado: %q", b)
	}

	// Y no queda ningún temporal tirado.
	entradas, _ := os.ReadDir(dir)
	for _, e := range entradas {
		if strings.HasPrefix(e.Name(), Prefijo) {
			t.Errorf("ha quedado un temporal suelto: %s", e.Name())
		}
	}
}

// El seguro contra un fallo de nuestro propio serializador, que la atomicidad no
// cubre: lo que está mal se escribe mal de forma perfectamente atómica.
func TestGuardaLaGeneracionAnterior(t *testing.T) {
	dir := t.TempDir()
	destino := filepath.Join(dir, "boveda")

	// La primera vez no hay nada que copiar, y eso no puede ser un error.
	if err := Atomica(destino, Opciones{Anterior: ".anterior"}, escribirTexto("uno")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(destino + ".anterior"); !os.IsNotExist(err) {
		t.Error("ha creado una copia de algo que no existía")
	}

	if err := Atomica(destino, Opciones{Anterior: ".anterior"}, escribirTexto("dos")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(destino + ".anterior")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "uno" {
		t.Errorf("la copia anterior dice %q, se esperaba «uno»", b)
	}

	if runtime.GOOS != "windows" {
		info, _ := os.Stat(destino + ".anterior")
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("la copia anterior tiene permisos %o: un secreto de ayer sigue siendo un secreto", got)
		}
	}
}

func TestCreaLaCarpetaSoloSiSeLePide(t *testing.T) {
	// Sin pedirlo, falla y lo dice: en la línea de comandos, inventarse tres
	// directorios en silencio sería peor que el error.
	suelto := filepath.Join(t.TempDir(), "sin", "crear", "fichero")
	if err := Atomica(suelto, Opciones{}, escribirTexto("hola")); err == nil {
		t.Error("ha escrito en una carpeta que no existía sin que nadie se lo pidiera")
	}

	destino := filepath.Join(t.TempDir(), "sin", "crear", "boveda")
	if err := Atomica(destino, Opciones{CrearCarpeta: true}, escribirTexto("hola")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(destino); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(filepath.Dir(destino))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("la carpeta tiene permisos %o, se esperaba 700", got)
	}
}

// Un temporal reciente puede ser de otro Esfinge escribiendo ahora mismo:
// borrarlo sería provocar el fallo que este paquete evita.
func TestSoloLimpiaLosHuerfanosViejosYPropios(t *testing.T) {
	dir := t.TempDir()
	viejo := filepath.Join(dir, Prefijo+"deAyer")
	nuevo := filepath.Join(dir, Prefijo+"deAhora")
	ajeno := filepath.Join(dir, "documento.txt")
	for _, f := range []string{viejo, nuevo, ajeno} {
		if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	hace := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(viejo, hace, hace); err != nil {
		t.Fatal(err)
	}

	LimpiarHuerfanos(dir, 24*time.Hour)

	if _, err := os.Stat(viejo); !os.IsNotExist(err) {
		t.Error("el huérfano viejo sigue ahí")
	}
	if _, err := os.Stat(nuevo); err != nil {
		t.Error("ha borrado un temporal reciente, que puede ser de otro proceso vivo")
	}
	if _, err := os.Stat(ajeno); err != nil {
		t.Error("ha borrado un fichero que no es suyo")
	}
}
