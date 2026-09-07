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
		return ModoInstalador
	}
}

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

	if ComoSeInstala() == ModoSolo {
		switch runtime.GOOS {
		case "darwin":
			return reemplazarseEnMac(ruta)
		case "windows":
			return reemplazarseEnWindows(ruta)
		}
	}
	return abrirConElSistema(ruta)
}

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
