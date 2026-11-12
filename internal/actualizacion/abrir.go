package actualizacion

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/webcafeina/esfinge/internal/adjuntos"
)

// carpetaDeEsfinge es donde los paquetes dejan lo que el programa necesita saber de su
// propia instalación. Variable para que las pruebas la aparten.
var carpetaDeEsfinge = "/usr/share/esfinge"

// gestorQueLoInstalo dice **con qué se instaló esta copia**, que no es lo mismo que si
// vino de un paquete.
//
// # Por qué la pregunta cambió
//
// Antes esto era `instaladoConPaquete()` y miraba si el ejecutable vive en `/usr/bin`.
// Con un solo paquete en el mundo —el `.deb`— las dos preguntas valían lo mismo. En
// cuanto hay `.deb`, `.rpm` y Arch, deja de valer: una Fedora instalada con `rpm` tiene
// su binario en `/usr/bin` igual, así que la pregunta vieja contesta «sí, vengo de un
// paquete» y a esa máquina se le ofrecía **un `.deb`**, que `dnf` no sabe instalar. Es la
// familia «una función cuyo nombre contesta *casi* lo que quieres», que el 2026-11-12 ya
// costó dos fallos.
//
// Lo dice el propio paquete al instalarse, escribiendo su nombre en un fichero: es el
// único que lo sabe de verdad. Si el fichero no está, se cae al razonamiento de antes
// —`/usr/bin` quiere decir `.deb`, el único paquete que existía cuando se escribió— para
// que una copia instalada con el `.deb` de hoy siga acertando mañana.
func gestorQueLoInstalo() string {
	// Una AppImage se anuncia ella misma por el entorno, y es el caso más claro: el
	// fichero montado es de solo lectura, así que lo que haya dentro no se puede creer.
	if os.Getenv("APPIMAGE") != "" {
		return adjuntos.GestorAppImage
	}
	if b, err := os.ReadFile(filepath.Join(carpetaDeEsfinge, "gestor")); err == nil {
		switch g := strings.TrimSpace(string(b)); g {
		case adjuntos.GestorDeb, adjuntos.GestorRPM, adjuntos.GestorPacman, adjuntos.GestorAppImage:
			return g
		}
		// Un valor que no se reconoce **no se cree**: vale más caer al razonamiento de
		// abajo que ofrecerle a alguien el paquete de otro gestor.
	}

	yo, err := os.Executable()
	if err != nil {
		return adjuntos.SinGestor
	}
	if resuelto, err := filepath.EvalSymlinks(yo); err == nil {
		yo = resuelto
	}
	if strings.HasPrefix(yo, "/usr/bin/") || strings.HasPrefix(yo, "/usr/local/bin/") {
		return adjuntos.GestorDeb
	}
	// Quien se bajó el tar.gz y lo puso en su carpeta: a ése no se le ofrece un paquete,
	// que le pediría contraseña para instalar algo donde no tiene nada.
	return adjuntos.SinGestor
}
