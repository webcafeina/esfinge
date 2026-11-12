package actualizacion

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Modo dice cómo se puede instalar la actualización en esta máquina, que es lo
// que decide qué se le ofrece a quien mira.
type Modo string

const (
	// ModoSolo: Esfinge se sustituye y se reinicia. No hay nada que hacer.
	ModoSolo Modo = "sola"
	// ModoInstalador: hace falta el instalador del sistema, porque escribir donde
	// vive la aplicación pide permisos que este proceso no tiene. Es el caso del
	// .deb, que instala como root, y el de una copia en una carpeta ajena.
	ModoInstalador Modo = "instalador"
	// ModoAMano: lo descargado **no lo instala nadie**, lo pone en su sitio una persona.
	//
	// Es el caso de la AppImage, y hace falta decirlo aparte porque las palabras del
	// instalador son falsas ahí: lo que se baja es la aplicación entera, no un
	// instalador, así que «abrir el instalador» abriría una segunda Esfinge —o se lo
	// daría a un gestor de archivos— y la que se está usando seguiría siendo la vieja.
	//
	// Lo honrado es enseñar dónde ha quedado y decir que reemplace con ella la que usa.
	ModoAMano Modo = "a mano"
)

// ComoSeInstala mira si esta copia puede reemplazarse sola.
//
// Lo que decide no es el sistema, es **si se puede escribir donde vive la
// aplicación**. Una copia en /Applications de un usuario administrador sí; una
// instalada por el gestor de paquetes en /usr/bin, no.
func ComoSeInstala() Modo {
	switch runtime.GOOS {
	case "darwin":
		paquete, err := paqueteDeLaApp()
		if err != nil {
			return ModoInstalador
		}
		if !sePuedeEscribirEn(filepath.Dir(paquete)) {
			return ModoInstalador
		}
		return ModoSolo
	case "windows":
		// El instalador NSIS sabe hacerlo solo y en silencio; lo que hace falta
		// es que no esté la aplicación en medio, y de eso se encarga el guion.
		return ModoSolo
	default:
		// **Una AppImage no la instala nadie.** Y ojo con el porqué, que está escrito al
		// revés en más de un sitio: lo que lo impide **no** es que el montaje sea de solo
		// lectura —el fichero de la AppImage es un fichero normal del usuario, y
		// renombrar encima de él funciona incluso mientras corre—, es que aquí se ha
		// elegido no reemplazarse sola. Queda apuntado en deuda.md con lo que costaría.
		if enUnaAppImage() {
			return ModoAMano
		}
		return ModoInstalador
	}
}

// enUnaAppImage dice si esta copia se está ejecutando desde una AppImage, que se anuncia
// ella misma por el entorno. Es la misma pregunta que decide qué paquete ofrecer.
func enUnaAppImage() bool { return os.Getenv("APPIMAGE") != "" }

// Instalar pone la versión descargada en su sitio.
//
// En los sistemas donde Esfinge puede reemplazarse sola, esto **no vuelve**: deja
// preparado un guion que espera a que este proceso muera, hace el cambiazo y
// vuelve a abrir la aplicación. Quien llama tiene que cerrar la ventana justo
// después.
//
// Donde no puede, se le entrega el fichero al sistema y que se ocupe él.
func Instalar(ruta string) error {
	if _, err := os.Stat(ruta); err != nil {
		return fmt.Errorf("No está el fichero descargado: %w", err)
	}

	switch ComoSeInstala() {
	case ModoSolo:
		switch runtime.GOOS {
		case "darwin":
			return reemplazarseEnMac(ruta)
		case "windows":
			return reemplazarseEnWindows(ruta)
		}
	case ModoAMano:
		// **La carpeta, no el fichero.** Abrir el fichero sería arrancar una segunda
		// Esfinge; lo que hace falta es ver dónde ha quedado para ponerlo en el sitio
		// del que se usa.
		return abrir(filepath.Dir(ruta))
	}
	return abrir(ruta)
}

// abrir es `abrirConElSistema`, en una variable **para que una prueba pueda decir qué se
// abre**.
//
// Hace falta porque lo que se abre no es siempre el fichero descargado: desde una AppImage
// es su carpeta, y la diferencia entre las dos cosas es «mira dónde ha quedado» o «arranca
// una segunda Esfinge y sigue usando la vieja». Sin costura, eso solo lo vería una persona
// con una AppImage delante.
var abrir = abrirConElSistema

// abrirConElSistema es lo de siempre: se lo damos al escritorio y que lo abra
// quien sepa. En Linux lo recoge el instalador de paquetes.
func abrirConElSistema(ruta string) error {
	var orden *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		orden = exec.Command("open", ruta)
	case "windows":
		// «start» no es un programa, es una orden del intérprete; de ahí el rodeo.
		orden = exec.Command("cmd", "/c", "start", "", ruta)
	case "linux":
		orden = exec.Command("xdg-open", ruta)
	default:
		return fmt.Errorf("No sé abrir un instalador en este sistema")
	}

	if err := orden.Start(); err != nil {
		return fmt.Errorf("No se ha podido abrir el instalador: %w", err)
	}
	go func() { _ = orden.Wait() }()
	return nil
}

// paqueteDeLaApp devuelve el .app dentro del que corre este proceso.
//
// El ejecutable vive en Esfinge.app/Contents/MacOS/Esfinge, así que el paquete
// son tres carpetas hacia arriba. Si la ruta no tiene esa forma es que no se está
// ejecutando desde un paquete —compilado suelto, o corriendo desde el código— y
// entonces no hay nada que reemplazar.
func paqueteDeLaApp() (string, error) {
	yo, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resuelto, err := filepath.EvalSymlinks(yo); err == nil {
		yo = resuelto
	}

	paquete := filepath.Dir(filepath.Dir(filepath.Dir(yo)))
	if !strings.HasSuffix(paquete, ".app") {
		return "", fmt.Errorf("Esfinge no se está ejecutando desde un paquete .app")
	}
	return paquete, nil
}

// sePuedeEscribirEn comprueba a lo bruto que se puede crear algo en la carpeta.
//
// Mirar los permisos con Stat no vale: en macOS hay listas de control de acceso,
// y en cualquier sistema el usuario puede no ser quien parece. Lo único que
// responde de verdad a «¿puedo escribir aquí?» es escribir.
func sePuedeEscribirEn(carpeta string) bool {
	f, err := os.CreateTemp(carpeta, ".esfinge-prueba-")
	if err != nil {
		return false
	}
	nombre := f.Name()
	f.Close()
	os.Remove(nombre)
	return true
}
