//go:build windows

package navegador

import (
	"errors"
	"net"
)

// En Windows este canal **todavía no existe**, y es una carencia declarada, no un
// olvido.
//
// Go sabe hablar sockets de dominio unix en Windows desde la 10.0.17063, así que
// la tentación es usar el mismo fichero para los tres sistemas. Se ha descartado
// por dos cosas que allí no se pueden hacer:
//
//   - **No hay forma de saber quién hay al otro lado.** En Linux está
//     `SO_PEERCRED` y en macOS el token de auditoría; con AF_UNIX en Windows no
//     hay equivalente, y encima el comportamiento de las listas de control de
//     acceso sobre el fichero del socket no es algo con lo que apostar la bóveda
//     de alguien. Una tubería con nombre sí permite las dos cosas —descriptor de
//     seguridad de solo el dueño y `GetNamedPipeClientProcessId`— y además tiene
//     `FILE_FLAG_FIRST_PIPE_INSTANCE`, que es lo que impide que otro se adelante
//     y conteste en nuestro lugar.
//   - Y una que no es de seguridad pero se ve: la línea de comandos es un binario
//     de consola, así que **Chrome parpadearía una ventana negra** cada vez que
//     lanza el host. Es exactamente por esto por lo que KeePassXC publica un
//     `keepassxc-proxy` pequeño y aparte en vez de su binario principal.
//
// Las dos cosas se resuelven juntas o no se resuelven, así que van juntas y
// después. Está en `docs/deuda.md`.
func escuchar(string) (net.Listener, error) {
	return nil, errors.New("El canal con el navegador todavía no funciona en Windows")
}

func limpiar(string) {}
