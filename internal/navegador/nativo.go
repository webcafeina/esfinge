package navegador

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
)

// El lado del navegador: «native messaging».
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
// Acierta en que habla con un binario de línea de comandos y se equivoca en lo
// demás — si ese proceso abriera la bóveda, la contraseña maestra tendría que
// llegar desde el navegador, que es exactamente lo que no puede pasar.
//
// # Y por qué es un binario aparte y no un subcomando de `esfinge`
//
// Empezó siendo un subcomando oculto, que es lo que pedía la ADR 0001 —un solo
// binario— y duró dos commits. Lo tumbó Windows, con dos razones que no se
// arreglan por separado:
//
//   - **`esfinge` es un binario de consola, y Chrome lanza el host cada pocos
//     minutos.** Cada arranque parpadearía una ventana negra en la cara de quien
//     esté navegando. Un binario del subsistema gráfico no la abre, y **sigue
//     teniendo entrada y salida estándar**, porque los descriptores los pasa quien
//     lo lanza. Es exactamente por esto por lo que KeePassXC publica un
//     `keepassxc-proxy` en vez de usar su binario principal.
//   - **Y porque arranca decenas de veces por sesión.** El binario de la línea de
//     comandos monta cobra, los estilos y el vigilante de versiones antes de
//     llegar a `main`. Esto no monta nada.
//
// # Tres cosas que rompen esto en silencio
//
//   - **La salida estándar es el protocolo.** Cualquier cosa que se escriba ahí
//     —un `fmt.Println` olvidado, una línea de ayuda, un aviso— parte una trama y
//     el navegador mata el proceso sin decir por qué. Lo primero que hace este
//     comando es **quedarse la salida y apuntar `os.Stdout` al error**, para que
//     un descuido futuro sea ruido en un registro y no un fallo imposible de
//     encontrar. Eso lo hace `cmd/esfinge-puente`.
//   - **El aviso de versión nueva.** Ya se calla solo, porque `vigilar` no
//     arranca si la salida de error no es un terminal (`novedad.go`) y aquí es una
//     tubería del navegador. Está comprobado, no supuesto.
//   - **El proceso se relanza constantemente.** El trabajador de una extensión
//     MV3 se muere a los pocos minutos, así que esto arranca y muere decenas de
//     veces por sesión. No puede guardar estado, no puede tardar en arrancar y no
//     puede dejar nada detrás.

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

// respuestaSinEsfinge es lo que se contesta cuando no hay con quién hablar. Va
// escrita a mano y no serializada desde `Respuesta` a propósito: este camino es
// el de cuando ya ha fallado algo, y no puede depender de nada más que pueda
// fallar.
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
