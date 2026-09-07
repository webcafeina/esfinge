package actualizacion

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Instalar entrega el fichero descargado al sistema para que haga lo suyo.
//
// Esfinge no se sustituye a sí misma. Hacerlo bien en macOS exige firmar con
// Apple —99 $ al año, descartado— y en los otros dos choca con los permisos del
// sitio donde está instalada. Así que se llega hasta la puerta y llama: el
// sistema pone el resto.
//
//   - macOS: monta la imagen y sale la ventana de arrastrar a Aplicaciones.
//   - Windows: arranca el asistente, que pedirá cerrar Esfinge.
//   - Linux: lo recoge el instalador de paquetes del escritorio.
func Instalar(ruta string) error {
	if _, err := os.Stat(ruta); err != nil {
		return fmt.Errorf("No está el fichero descargado: %w", err)
	}

	var orden *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		orden = exec.Command("open", ruta)
	case "windows":
		// «start» no es un programa, es una orden del intérprete; de ahí el rodeo.
		// La cadena vacía es el título de la ventana, que start espera cuando el
		// argumento siguiente va entre comillas.
		orden = exec.Command("cmd", "/c", "start", "", ruta)
	case "linux":
		orden = exec.Command("xdg-open", ruta)
	default:
		return fmt.Errorf("No sé abrir un instalador en este sistema")
	}

	if err := orden.Start(); err != nil {
		return fmt.Errorf("No se ha podido abrir el instalador: %w", err)
	}
	// No se espera a que termine: el instalador vive más que esta llamada, y en
	// Windows más que la propia Esfinge.
	go func() { _ = orden.Wait() }()
	return nil
}

// instaladoConPaquete dice si esta copia viene del .deb.
//
// Se mira dónde vive el ejecutable: el paquete lo pone en /usr/bin, y quien se
// bajó el tar.gz lo tendrá en su carpeta. A ése no se le ofrece un .deb, que le
// pediría contraseña para instalar algo donde no tiene nada.
func instaladoConPaquete() bool {
	yo, err := os.Executable()
	if err != nil {
		return false
	}
	if resuelto, err := filepath.EvalSymlinks(yo); err == nil {
		yo = resuelto
	}
	return strings.HasPrefix(yo, "/usr/bin/") || strings.HasPrefix(yo, "/usr/local/bin/")
}
