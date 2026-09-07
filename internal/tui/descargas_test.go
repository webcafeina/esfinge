package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// descargasDePrueba monta una carpeta personal de mentira con su Descargas
// dentro, y devuelve esa ruta. Así los tests no escriben en la carpeta de
// verdad de quien los ejecuta.
func descargasDePrueba(t *testing.T) string {
	t.Helper()
	casa := t.TempDir()
	descargas := filepath.Join(casa, "Downloads")
	if err := os.MkdirAll(descargas, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa) // Windows
	t.Setenv("XDG_DOWNLOAD_DIR", descargas)
	return descargas
}

func TestCarpetaDeDescargas(t *testing.T) {
	descargas := descargasDePrueba(t)
	if got := CarpetaDeDescargas(); got != descargas {
		t.Errorf("CarpetaDeDescargas() = %q, quiero %q", got, descargas)
	}
}

// Sin carpeta de descargas por ninguna parte, cae en la carpeta personal en vez
// de fallar: mejor dejar el fichero en un sitio raro que no dejarlo.
func TestCarpetaDeDescargasSinDescargas(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)
	t.Setenv("XDG_DOWNLOAD_DIR", "")

	if got := CarpetaDeDescargas(); got != casa {
		t.Errorf("CarpetaDeDescargas() = %q, quiero la carpeta personal %q", got, casa)
	}
}

// Una carpeta en español también vale: en algunos Linux es la que existe.
func TestCarpetaDeDescargasEnEspanol(t *testing.T) {
	casa := t.TempDir()
	if err := os.MkdirAll(filepath.Join(casa, "Descargas"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)
	t.Setenv("XDG_DOWNLOAD_DIR", "")

	if got := CarpetaDeDescargas(); got != filepath.Join(casa, "Descargas") {
		t.Errorf("CarpetaDeDescargas() = %q", got)
	}
}
