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
	"os/exec"
	"runtime"

	"github.com/webcafeina/esfinge/internal/agente"
	"github.com/webcafeina/esfinge/internal/mcpb"
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

	// **Y si hay un servidor dentro de la Esfinge instalada, el turno es suyo.**
	//
	// El `.mcpb` lleva una copia de esto dentro, así que actualizar Esfinge no lo
	// actualizaba: el agente seguía viendo las herramientas de la versión con la que
	// se instaló el paquete, y como los clientes MCP se quedan con esa lista al
	// conectar, no aparecían ni reiniciando la conversación. Lo encontró el cliente
	// pidiendo crear una cuenta con su sitio.
	//
	// Manda el de la aplicación y no el más nuevo de los dos: es con **esa** Esfinge
	// con la que se va a hablar, así que su servidor es el que entiende su canal.
	if otro := aQuienCederle(); otro != "" {
		if relevar(otro, salida) {
			return
		}
	}

	if err := agente.Traducir(os.Stdin, salida, version, agente.RutaDelCanal()); err != nil {
		os.Exit(1)
	}
}

// aQuienCederle mira el disco. **Que falle no puede impedir arrancar**, así que
// cualquier duda devuelve vacío y sigue este binario.
func aQuienCederle() string {
	yo, err := os.Executable()
	if err != nil {
		return ""
	}
	casa, err := os.UserHomeDir()
	if err != nil {
		casa = ""
	}
	return mcpb.AQuienCederle(runtime.GOOS, casa, yo, mcpb.SePuedeEjecutar)
}

// relevar lanza al otro con esta misma entrada y salida, y dice si se ha hecho cargo.
//
// Se hace con un proceso hijo y no con `exec` del sistema porque **tiene que valer en
// los tres**, y en Windows no hay `exec`: un solo camino es un camino que se prueba.
// Lo que el cliente ve por su tubería es idéntico — el hijo hereda la entrada y la
// salida de verdad, no una copia.
//
// **Si no arranca, se devuelve `false` y sigue éste.** Lo que no puede pasar es que el
// relevo deje al agente sin servidor.
func relevar(otro string, salida *os.File) bool {
	c := exec.Command(otro)
	c.Stdin = os.Stdin
	c.Stdout = salida
	c.Stderr = os.Stderr
	if err := c.Start(); err != nil {
		return false
	}
	// A partir de aquí manda él: lo que haga con el protocolo es cosa suya, y su
	// código de salida es el nuestro.
	if err := c.Wait(); err != nil {
		if salir, vale := err.(*exec.ExitError); vale {
			os.Exit(salir.ExitCode())
		}
		os.Exit(1)
	}
	return true
}
