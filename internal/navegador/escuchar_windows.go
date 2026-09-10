//go:build windows

package navegador

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// En Windows el canal es **el mismo socket**, y conviene decir por qué, porque la
// respuesta evidente era otra.
//
// Go sabe hablar sockets de dominio unix en Windows desde la 10.0.17063
// (`internal/syscall/windows/version_windows.go`), así que el código es el mismo
// que en los otros dos sistemas. Lo que se sopesó y se descartó fue una **tubería
// con nombre**, que es lo que usa KeePassXC allí. A favor tenía tres cosas:
//
//  1. saber quién hay al otro lado (`GetNamedPipeClientProcessId`);
//  2. un descriptor de seguridad de solo el dueño;
//  3. `FILE_FLAG_FIRST_PIPE_INSTANCE`, que impide que otro se adelante y conteste
//     en nuestro lugar.
//
// Y se descartó por lo que costaba comprarlas:
//
//   - **La primera no se usa en ninguna parte.** Esfinge no comprueba quién se
//     conecta tampoco en macOS ni en Linux, y no por descuido: 1Password lo hace
//     verificando la firma del navegador, y aquí no se puede porque **Esfinge no
//     está firmada** (ADR 0012). Comprar una capacidad que no se va a usar no es
//     seguridad, es código.
//   - **La segunda la da la carpeta.** El socket vive en el perfil del usuario,
//     que en Windows ya está cerrado a los demás usuarios por lista de control de
//     acceso. Es la misma frontera que dan los permisos 0600 en los otros
//     sistemas: la que hay entre usuarios de la máquina, no entre programas del
//     mismo usuario. **Eso está dicho en `docs/seguridad.md`.**
//   - **La tercera se consigue igual**, preguntando: si el fichero está y contesta
//     alguien, aquí no se escucha.
//   - Y lo que costaba: doscientas líneas de llamadas al sistema de Windows
//     —tuberías, entrada y salida solapada, descriptores de seguridad— **en la
//     frontera de seguridad de un gestor de contraseñas, escritas en una máquina
//     donde no se pueden ejecutar**. Este proyecto ya sabe cómo acaba eso: el
//     Objective-C de `vidrio_darwin.go` compilaba y cerraba la aplicación al
//     arrancar, y costó una versión rota y una revertida.
//
// Si algún día se comprueba quién se conecta —porque Esfinge se firme, por
// ejemplo— esta decisión se revisa, y entonces la tubería sí se paga sola.
func escuchar(ruta string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return nil, err
	}

	// Lo mismo que en los otros sistemas: un socket que ya está puede ser de un
	// Esfinge vivo —y entonces no se toca— o el resto de un cierre sucio.
	if _, err := os.Stat(ruta); err == nil {
		if c, err := net.Dial("unix", ruta); err == nil {
			c.Close()
			return nil, fmt.Errorf("Ya hay otro Esfinge escuchando en %s", ruta)
		}
		os.Remove(ruta)
	}

	// **Sin `Chmod`**, y no es un olvido: en Windows no significa lo que significa
	// en Unix —solo toca el bit de solo lectura— y llamarlo aquí daría la
	// impresión de una frontera que no pone. La pone la carpeta.
	return net.Listen("unix", ruta)
}

func limpiar(ruta string) { os.Remove(ruta) }
