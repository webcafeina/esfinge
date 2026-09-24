// Package llavero guarda un secreto **fuera de Esfinge**, donde el sistema pide
// la huella para devolverlo: Touch ID en macOS, Windows Hello en Windows. En
// Linux no hay ninguno y este paquete lo dice.
//
// Es la fase C del plan de cuentas (`docs/desbloqueo-del-sistema.md`), y lo
// primero que hay que tener claro al leerlo:
//
// **Esto es un cerrojo y no una llave.** Sin firmar la aplicación —decisión del
// cliente, ADR 0012— el sistema no puede atar ese secreto a Esfinge: en macOS el
// camino fuerte exige una entitlement que solo lleva una compilación firmada, y
// en Windows la credencial de Hello de una aplicación sin empaquetar **está atada
// a la cuenta de usuario y no a la aplicación**. Así que protege de quien se
// siente delante de tu ordenador desbloqueado, no de un programa que corra como
// tú. Está dicho así en `docs/seguridad.md` y en la pantalla donde se activa, y
// **no se debe escribir de otra forma**.
//
// Lo que este paquete **no** hace: no sabe qué es una bóveda ni qué guarda. Le
// dan un identificador y unos bytes.
package llavero

import "errors"

var (
	// ErrNoHay: este equipo no tiene con qué. Es lo normal en Linux.
	ErrNoHay = errors.New("Este equipo no tiene desbloqueo del sistema")
	// ErrNoEsta: no hay nada guardado con ese identificador. Pasa si se quitó
	// desde fuera de Esfinge —en Acceso a Llaveros, por ejemplo—.
	ErrNoEsta = errors.New("El sistema ya no guarda esa llave")
	// ErrNoQuiso: la persona canceló, o el sistema no la reconoció. **No es un
	// fallo**: se vuelve a la contraseña maestra sin decir nada raro.
	ErrNoQuiso = errors.New("No se ha podido comprobar quién eres")
)

// Llavero es lo que cada sistema sabe hacer con un secreto pequeño.
type Llavero interface {
	// Nombre es cómo lo llama su sistema, para poder escribirlo en la ventana:
	// «Touch ID», «Windows Hello». Vacío si no hay ninguno.
	Nombre() string
	// Hay dice si en este equipo se puede usar **ahora**: no basta con que el
	// sistema lo soporte, tiene que estar configurado.
	Hay() bool
	// Guardar deja el secreto. Reemplaza lo que hubiera con ese identificador.
	Guardar(id string, secreto []byte) error
	// Leer lo devuelve, **pidiendo identificarse**. `motivo` es la frase que el
	// sistema enseña en su diálogo, así que se escribe para quien la lee.
	Leer(id, motivo string) ([]byte, error)
	// Borrar lo quita. Que no estuviera no es un error.
	Borrar(id string) error
}

// Del sistema en el que corre esto. Nunca devuelve nil.
func Del() Llavero { return delSistema() }

// Ninguno es el llavero de un sistema que no tiene: dice que no a todo y no
// falla al construirse. Es el de Linux, y el de un Mac sin Touch ID.
type Ninguno struct{}

func (Ninguno) Nombre() string                      { return "" }
func (Ninguno) Hay() bool                           { return false }
func (Ninguno) Guardar(string, []byte) error        { return ErrNoHay }
func (Ninguno) Leer(string, string) ([]byte, error) { return nil, ErrNoHay }
func (Ninguno) Borrar(string) error                 { return nil }
