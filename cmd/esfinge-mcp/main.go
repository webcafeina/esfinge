// El servidor MCP de Esfinge: lo que lanza un agente de IA para hablar con la bóveda.
//
// Un binario diminuto y aparte, por las mismas dos razones que `esfinge-puente` y una
// tercera.
//
// **Aparte** porque lo lanza otro programa —Claude Code al abrir una sesión, Claude
// Desktop al arrancar— y `esfinge` es un binario de consola: en Windows cada arranque
// parpadearía una ventana negra. Éste se compila para el subsistema gráfico, que no
// abre consola y **sí** conserva la entrada y la salida estándar.
//
// **Diminuto** porque aquí no hace falta nada de lo que Esfinge monta: ni cobra, ni los
// estilos, ni el vigilante de versiones.
//
// **Y sin la bóveda dentro**: igual que el puente del navegador, esto **no importa el
// paquete de la bóveda**, así que no sabría abrir una aunque quisiera. La bóveda
// descifrada vive en un solo sitio, que es la ventana.
//
// Lo único que sabe hacer por su cuenta es **saludar**: `initialize` y `tools/list` se
// contestan aquí, sin preguntarle a nadie. No es una optimización (ADR 0054): un agente
// que arranque con Esfinge cerrada **se queda con la lista de herramientas que le demos
// en ese momento**, así que contestar una lista vacía lo dejaría sin herramientas toda
// la sesión aunque la ventana se abriera dos minutos después.
package main

import (
	"os"

	"github.com/webcafeina/esfinge/internal/agente"
)

// version la pone el compilador al publicar, como en los demás binarios.
var version = "dev"

func main() {
	// **Lo primero de todo.** La salida estándar de este proceso *es* el protocolo:
	// cualquier cosa que se escriba ahí parte un mensaje y el cliente cierra el
	// servidor sin decir por qué. Se guarda la de verdad y se apunta `os.Stdout` al
	// error, para que un descuido futuro acabe en un registro y no en un fallo
	// imposible de encontrar. Es la misma línea que `esfinge-puente`, y por lo mismo.
	salida := os.Stdout
	os.Stdout = os.Stderr

	if err := agente.Traducir(os.Stdin, salida, version, agente.RutaDelCanal()); err != nil {
		os.Exit(1)
	}
}
