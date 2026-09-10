package cli

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/app"
)

// El proceso que lanza el navegador.
//
// # Qué es esto
//
// Una extensión no puede abrir un socket ni hablar con un programa del sistema:
// lo único que le deja el navegador es **lanzar un proceso declarado de antemano
// y hablar con él por la entrada y la salida estándar**. Eso es «native
// messaging», y esto es ese proceso.
//
// No hace nada más que traducir: lee un mensaje del navegador, se lo pasa a
// Esfinge por el socket local y devuelve la respuesta. **No abre la bóveda ni
// sabe la contraseña maestra**, y ésa es la decisión que sostiene todo lo demás:
// la bóveda descifrada vive en un solo sitio, la ventana, y aquí solo pasa gente.
//
// Eso corrige a medias una frase que lleva escrita en `internal/cli/boveda.go`
// desde que existe la bóveda: «la extensión hablará con esto, no con la ventana».
// Acierta en que habla con este binario y se equivoca en lo demás — si este
// proceso abriera la bóveda, la contraseña maestra tendría que llegar desde el
// navegador, que es exactamente lo que no puede pasar.
//
// # Tres cosas que rompen esto en silencio
//
//   - **La salida estándar es el protocolo.** Cualquier cosa que se escriba ahí
//     —un `fmt.Println` olvidado, una línea de ayuda, un aviso— parte una trama y
//     el navegador mata el proceso sin decir por qué. Lo primero que hace este
//     comando es **quedarse la salida y apuntar `os.Stdout` al error**, para que
//     un descuido futuro sea ruido en un registro y no un fallo imposible de
//     encontrar.
//   - **El aviso de versión nueva.** Ya se calla solo, porque `vigilar` no
//     arranca si la salida de error no es un terminal (`novedad.go`) y aquí es una
//     tubería del navegador. Está comprobado, no supuesto.
//   - **El proceso se relanza constantemente.** El trabajador de una extensión
//     MV3 se muere a los pocos minutos, así que esto arranca y muere decenas de
//     veces por sesión. No puede guardar estado, no puede tardar en arrancar y no
//     puede dejar nada detrás.
func comandoPuenteNavegador() *cobra.Command {
	return &cobra.Command{
		Use: "puente-navegador",
		// Oculto porque **no es para personas**: lo lanza el navegador, y una
		// persona que lo escriba en un terminal se queda mirando un proceso que
		// espera bytes binarios. Sigue estando documentado aquí y en la ADR.
		Hidden:        true,
		Short:         "Traduce entre la extensión del navegador y Esfinge",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(*cobra.Command, []string) error {
			// **Lo primero de todo**: quedarse la salida de verdad y dejar `os.Stdout`
			// apuntando al error. A partir de aquí, lo único que llega al navegador es
			// lo que escriba este fichero.
			salida := os.Stdout
			os.Stdout = os.Stderr
			return Traducir(os.Stdin, salida, app.RutaDelCanal())
		},
	}
}

// topeDelNavegador es lo más grande que Chrome acepta **hacia** la extensión: un
// megabyte. Hacia aquí admite sesenta y cuatro, pero lo que sale es lo que
// importa, y de aquí no sale nada grande: una lista de cuentas sin secretos.
//
// Conviene saberlo antes de que a alguien se le ocurra mandar iconos por este
// canal, que es justo el tamaño que no cabe.
const topeDelNavegador = 1 << 20

// Traducir hace de intérprete entre el navegador y Esfinge hasta que uno de los
// dos se va.
//
// Recibe los extremos en vez de cogerlos de `os` para poder probarse con dos
// tuberías y un socket de mentira, que es como se comprueba un protocolo sin
// tener delante un navegador.
func Traducir(entra io.Reader, sale io.Writer, socket string) error {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		// Esfinge no está abierta, o el canal está apagado en Ajustes. **Se contesta
		// en vez de morirse**: la extensión necesita poder decir «abre Esfinge» y no
		// «algo ha fallado», que son cosas distintas para quien mira.
		return responderSiempre(entra, sale, respuestaSinEsfinge())
	}
	defer conn.Close()

	desdeEsfinge := json.NewDecoder(conn)
	haciaEsfinge := json.NewEncoder(conn)

	for {
		crudo, err := leerDelNavegador(entra)
		if err != nil {
			return nil // el navegador ha cerrado; es lo normal al terminar
		}

		var respuesta json.RawMessage
		if err := haciaEsfinge.Encode(json.RawMessage(crudo)); err != nil {
			return escribirAlNavegador(sale, respuestaSinEsfinge())
		}
		if err := desdeEsfinge.Decode(&respuesta); err != nil {
			return escribirAlNavegador(sale, respuestaSinEsfinge())
		}
		if err := escribirAlNavegador(sale, respuesta); err != nil {
			return err
		}
	}
}

// leerDelNavegador lee un mensaje: cuatro bytes de longitud y el JSON detrás.
//
// **La longitud va en el orden de bytes de la máquina**, no en uno fijo: así lo
// dice la documentación de Chrome, y es de las pocas cosas de un protocolo que se
// escriben con esa forma. En las máquinas donde esto se publica —Intel y ARM— eso
// es el orden pequeño, así que se usa ése y queda dicho que es una suposición,
// no un descuido.
func leerDelNavegador(r io.Reader) ([]byte, error) {
	var largo [4]byte
	if _, err := io.ReadFull(r, largo[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint32(largo[:])
	if n == 0 || n > 64<<20 {
		return nil, fmt.Errorf("mensaje de %d bytes", n)
	}
	crudo := make([]byte, n)
	if _, err := io.ReadFull(r, crudo); err != nil {
		return nil, err
	}
	return crudo, nil
}

func escribirAlNavegador(w io.Writer, mensaje []byte) error {
	if len(mensaje) > topeDelNavegador {
		return errors.New("la respuesta no cabe en un mensaje del navegador")
	}
	var largo [4]byte
	binary.LittleEndian.PutUint32(largo[:], uint32(len(mensaje)))
	if _, err := w.Write(largo[:]); err != nil {
		return err
	}
	_, err := w.Write(mensaje)
	return err
}

// respuestaSinEsfinge es lo que se contesta cuando no hay con quién hablar. Se
// escribe a mano y no se serializa desde el paquete `navegador` para que este
// camino no dependa de nada que pueda fallar justo cuando ya ha fallado algo.
func respuestaSinEsfinge() []byte {
	return []byte(`{"ok":false,"motivo":"sin-esfinge",` +
		`"error":"Esfinge no está abierta, o el canal con el navegador está apagado en Ajustes"}`)
}

// responderSiempre contesta lo mismo a todo lo que llegue, hasta que el navegador
// se cansa. Es lo que se hace sin Esfinge al otro lado.
func responderSiempre(entra io.Reader, sale io.Writer, que []byte) error {
	for {
		if _, err := leerDelNavegador(entra); err != nil {
			return nil
		}
		if err := escribirAlNavegador(sale, que); err != nil {
			return err
		}
	}
}
