package actualizacion

import (
	"os"
	"path/filepath"
	"runtime"
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
	yo, err := os.Executable()
	if err == nil {
		if resuelto, err := filepath.EvalSymlinks(yo); err == nil {
			yo = resuelto
		}
	}
	return gestorSegun(runtime.GOOS, os.Getenv("APPIMAGE"), carpetaDeEsfinge, yo, loQueDiga)
}

// loQueDiga lee el fichero donde el paquete dejó su nombre. Vacío si no hay.
func loQueDiga(fichero string) string {
	b, err := os.ReadFile(fichero)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// gestorSegun es la decisión, con el mundo por parámetro.
//
// Recibirlo en vez de preguntárselo al sistema es lo que permite comprobar **los cinco
// gestores desde este VPS**, igual que las rutas de los manifiestos del navegador se calculan
// para un sistema dado y no para `runtime.GOOS`. Lo que cambia por sistema es sólo dónde se
// mira:
//
//   - en Linux, en /usr/share/esfinge/gestor, que es donde lo deja el `postinstall` del
//     paquete, y además una AppImage se anuncia ella misma por el entorno;
//   - en Windows **al lado del programa**, porque allí no hay un sitio compartido y el
//     instalador escribe en su propia carpeta. Lo pone cuando el paquete se lo pide con
//     `/GESTOR=`, que es lo único que distingue un `choco install` de un doble clic (ADR 0063).
func gestorSegun(sistema, appimage, carpetaCompartida, ejecutable string, leer func(string) string) string {
	// Una AppImage se anuncia ella misma, y es el caso más claro: el fichero montado es de
	// solo lectura, así que lo que haya dentro no se puede creer.
	if appimage != "" {
		return adjuntos.GestorAppImage
	}

	donde := carpetaCompartida + "/gestor"
	if sistema == "windows" {
		if ejecutable == "" {
			return adjuntos.SinGestor
		}
		// **El separador es el del sistema que se está calculando, no el de esta máquina.**
		// `filepath.Dir` sobre `C:\…\esfinge.exe` corriendo en Linux contesta «.», porque
		// allí la barra invertida no separa nada — y entonces esto miraría un fichero que no
		// es. Es lo mismo que ya pasa con las rutas de los manifiestos del navegador.
		corte := strings.LastIndexAny(ejecutable, `\/`)
		if corte < 0 {
			return adjuntos.SinGestor
		}
		donde = ejecutable[:corte] + `\gestor`
	}

	// **Recortado aquí y no en quien lee.** El `postinstall` escribe un salto de línea al
	// final, así que sin esto el `switch` no reconocería «deb\n» y la copia caería en el
	// razonamiento de abajo. Lo cazó la tabla: el caso de Debian pasaba en verde **por el
	// camino equivocado**, que es el peor modo de pasar.
	switch g := strings.TrimSpace(leer(donde)); g {
	case adjuntos.GestorDeb, adjuntos.GestorRPM, adjuntos.GestorPacman, adjuntos.GestorAppImage,
		adjuntos.GestorChocolatey, adjuntos.GestorWinget:
		return g
	}
	// Un valor que no se reconoce **no se cree**: vale más caer al razonamiento de abajo que
	// ofrecerle a alguien el paquete de otro gestor.

	// En Windows no hay nada más que mirar: quien instaló con el `.exe` a mano no viene de
	// ningún gestor, y eso es lo correcto — a ése sí se le ofrece la actualización.
	if sistema == "windows" || ejecutable == "" {
		return adjuntos.SinGestor
	}
	if strings.HasPrefix(ejecutable, "/usr/bin/") || strings.HasPrefix(ejecutable, "/usr/local/bin/") {
		// El `.deb` era el único paquete que existía cuando se escribió esto, así que una
		// copia instalada con él **antes** de que el postinstall dejara el fichero sigue
		// acertando.
		return adjuntos.GestorDeb
	}
	// Quien se bajó el tar.gz y lo puso en su carpeta: a ése no se le ofrece un paquete,
	// que le pediría contraseña para instalar algo donde no tiene nada.
	return adjuntos.SinGestor
}
