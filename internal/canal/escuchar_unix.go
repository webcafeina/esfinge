//go:build !windows

// Package canal abre el socket local por el que Esfinge atiende a quien se lo ha
// pedido: la extensión del navegador (ADR 0027) y los agentes de IA (ADR 0054).
//
// **Está aquí y no duplicado en cada uno** porque es exactamente el mismo código y
// porque lo que resuelve —el tope de `sun_path`, distinguir un socket vivo de uno
// huérfano, los permisos— son trampas que ya costaron encontrarse una vez. Dos copias
// significan que el día que se arregle algo, se arregle en una.
//
// Lo que **no** vive aquí es el protocolo: cada canal tiene el suyo, su lista de
// verbos y sus frenos, y ésa es la decisión de la ADR 0054.
package canal

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// largoMaximoDeRuta es el tope de la ruta de un socket de dominio unix.
//
// No es un número que se elija: es el tamaño de `sun_path` en la estructura del
// sistema, 104 bytes en macOS y en los BSD, 108 en Linux. Se usa el más corto de
// los dos porque el que falla es macOS, donde la carpeta de configuración ya es
// `~/Library/Application Support`. Se comprueba **antes** de escuchar, para que
// el error diga esto en vez de un «invalid argument» del sistema.
const largoMaximoDeRuta = 104

// Escuchar abre el socket local. `deQue` nombra el canal, y sale en el error:
// «La ruta del canal con el navegador no cabe…» dice dónde mirar; «la ruta del canal»
// a secas, no.
//
// **No es TCP, y ésa es media respuesta a por qué esto es aceptable** en un
// programa cuyo propio código dice que «un servidor HTTP en el binario del
// cliente, por local que sea, es una puerta que nadie ha pedido»
// (`internal/app/dev.go`). Un socket de dominio unix no tiene puerto: no se
// alcanza desde otra máquina, ni desde otra sesión, ni por una página web que
// pruebe direcciones locales. Lo alcanza quien pueda abrir ese fichero.
func Escuchar(ruta, deQue string) (net.Listener, error) {
	if len(ruta) > largoMaximoDeRuta {
		return nil, fmt.Errorf(
			"La ruta del canal con %s no cabe en este sistema (%d caracteres, el tope son %d)",
			deQue, len(ruta), largoMaximoDeRuta)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return nil, err
	}

	// **Un socket que ya está puede ser dos cosas muy distintas**, y confundirlas
	// es grave en las dos direcciones: si es de un Esfinge vivo y lo borramos, le
	// quitamos el canal a quien lo tenía; si es el resto de un cierre sucio y no lo
	// borramos, esto no arranca nunca más. Se distinguen preguntando: al de un
	// Esfinge vivo se le puede conectar.
	if _, err := os.Stat(ruta); err == nil {
		if c, err := net.Dial("unix", ruta); err == nil {
			c.Close()
			return nil, fmt.Errorf("Ya hay otro Esfinge escuchando en %s", ruta)
		}
		os.Remove(ruta)
	}

	oyente, err := net.Listen("unix", ruta)
	if err != nil {
		return nil, err
	}
	// Solo el dueño. En Linux y en macOS los permisos del fichero de socket sí se
	// respetan al conectar, así que esto es una frontera de verdad —la que hay
	// entre usuarios de la máquina, no entre programas del mismo usuario—.
	if err := os.Chmod(ruta, 0o600); err != nil {
		oyente.Close()
		return nil, err
	}
	return oyente, nil
}

// Limpiar borra el fichero del socket al parar.
func Limpiar(ruta string) { os.Remove(ruta) }
