package actualizacion

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Lo que decide si Esfinge puede reemplazarse sola no es el sistema, es si se
// puede escribir donde vive. Y eso solo lo responde de verdad escribir.
func TestSePuedeEscribirDondeSePuede(t *testing.T) {
	if !sePuedeEscribirEn(t.TempDir()) {
		t.Error("en una carpeta temporal propia sí se puede escribir")
	}
	if sePuedeEscribirEn(filepath.Join(t.TempDir(), "no-existe")) {
		t.Error("dice que se puede escribir en una carpeta que no existe")
	}

	// Una carpeta sin permiso de escritura. Como root no vale, que puede con
	// todo, y en Windows los permisos POSIX no significan lo mismo.
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("los permisos POSIX no deciden aquí")
	}
	cerrada := filepath.Join(t.TempDir(), "cerrada")
	if err := os.Mkdir(cerrada, 0o500); err != nil {
		t.Fatal(err)
	}
	if sePuedeEscribirEn(cerrada) {
		t.Error("dice que se puede escribir en una carpeta de solo lectura")
	}
}

// Fuera de un paquete .app no hay nada que reemplazar, y hay que decirlo en vez
// de intentar el cambiazo sobre una ruta cualquiera.
func TestSinPaqueteNoHayNadaQueReemplazar(t *testing.T) {
	if runtime.GOOS != "darwin" {
		// paqueteDeLaApp mira dónde está este ejecutable; fuera de macOS el
		// binario de los tests nunca vive en un .app, que es justo el caso.
		if _, err := paqueteDeLaApp(); err == nil {
			t.Error("ha creído estar dentro de un paquete .app")
		}
	}
}

// En Linux se instala con el gestor de paquetes: pedir escribir en /usr/bin
// desde la aplicación sería pedir la contraseña de root para algo que el sistema
// ya sabe hacer.
func TestEnLinuxSeUsaElInstaladorDelSistema(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	if ComoSeInstala() != ModoInstalador {
		t.Error("en Linux la actualización pasa por el instalador del sistema")
	}
}
