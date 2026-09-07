package actualizacion

import (
	"os"
	"path/filepath"
	"strings"
)

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
