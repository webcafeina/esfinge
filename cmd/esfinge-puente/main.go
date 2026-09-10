// El puente entre la extensión del navegador y Esfinge.
//
// Un binario diminuto y aparte, y las dos cosas a propósito.
//
// **Aparte** porque en Windows el host lo lanza el navegador cada pocos minutos
// y `esfinge` es un binario de consola: cada arranque parpadearía una ventana
// negra. Éste se compila para el subsistema gráfico —que no abre consola y **sí**
// conserva la entrada y la salida estándar, porque los descriptores los pasa
// quien lo lanza— y en los otros dos sistemas da igual, así que es el mismo en
// los tres. Es lo que hace KeePassXC con su `keepassxc-proxy`, por lo mismo.
//
// **Diminuto** porque arranca y muere decenas de veces por sesión: el trabajador
// de una extensión MV3 se cae a los pocos minutos y con él se va el proceso. Aquí
// no se monta cobra, ni los estilos, ni el vigilante de versiones.
//
// Lo único que hace es traducir. **No abre la bóveda y no sabe la contraseña
// maestra**: la bóveda descifrada vive en un solo sitio, que es la ventana. Y no
// es solo que no lo haga: **no importa el paquete de la bóveda**, así que este
// binario no lleva dentro ni el cifrado ni el formato. No sabría abrir una
// aunque quisiera.
package main

import (
	"os"

	"github.com/webcafeina/esfinge/internal/navegador"
)

func main() {
	// **Lo primero de todo.** La salida estándar de este proceso *es* el
	// protocolo: cualquier cosa que se escriba ahí parte una trama y el navegador
	// mata el proceso sin decir por qué. Se guarda la de verdad y se apunta
	// `os.Stdout` al error, para que un descuido futuro acabe en un registro y no
	// en un fallo imposible de encontrar.
	salida := os.Stdout
	os.Stdout = os.Stderr

	if err := navegador.Traducir(os.Stdin, salida, navegador.RutaDelCanal()); err != nil {
		os.Exit(1)
	}
}
