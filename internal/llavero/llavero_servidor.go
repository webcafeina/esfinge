package llavero

// El llavero de un servidor: **un fichero del disco** (ADR 0057).
//
// Es la tercera implementación de `Llavero`, y la única que no esconde el secreto
// detrás de nadie. Por eso lo primero es decir qué protege, porque no es lo mismo
// que las otras dos:
//
// **Esto no es un cerrojo.** Touch ID y Windows Hello protegen de quien se sienta
// delante de tu ordenador desbloqueado. Aquí no hay nadie a quien pedirle nada, y
// la llave está en el disco: **quien entre en el servidor abre la bóveda**. Lo que
// se gana a cambio está en la ADR y en `docs/seguridad.md` —lo que hoy son
// ficheros `.env` en claro queda cifrado en reposo, se cambia desde el Mac, deja
// rastro, y se revoca rotando la clave en vez de entrando a borrar un fichero— y
// **no se debe escribir de otra forma**.
//
// Tres cosas que no son detalles:
//
//   - **El fichero va fuera de la carpeta de las bóvedas**, y esa decisión vive
//     aquí porque aquí se calcula la ruta. Es la regla de la ADR 0044: con la
//     llave al lado, quien copie la carpeta abre la bóveda, y eso no es un cerrojo
//     peor, es quitar la puerta. Lo vigila
//     `TestLaCarpetaDeLaBovedaNoLlevaLaLlaveDelServidor`.
//   - **No lo devuelve `Del()`.** En Linux sigue sin haber llavero, con su porqué
//     escrito en `llavero_otros.go`, y esto no lo cambia: hay que pedirlo. Que un
//     fichero en el disco apareciera solo como «desbloqueo del sistema» en el
//     portátil de cualquiera sería justo el cerrojo sin cerradura que allí se
//     descartó.
//   - **No tiene etiqueta de compilación.** Vale en los tres sistemas, lo que de
//     paso permite probarlo aquí: es lo contrario de `llavero_darwin.go`, que no
//     se puede ni compilar en esta máquina.

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/webcafeina/esfinge/internal/escritura"
)

// VariableDeLaLlave dice dónde está el fichero, para poder ponerlo en otro sitio
// —otro disco, un volumen cifrado que se monta a mano— sin tocar el código.
const VariableDeLaLlave = "ESFINGE_LLAVE_DEL_SERVIDOR"

// EnFichero guarda los secretos en un JSON de `{identificador: hex}`, 0600.
//
// Es un mapa y no un secreto pelado porque la interfaz va por identificador; hoy
// solo hay uno —el de la bóveda personal de este equipo, que es la única llave en
// disco— y guardarlo por su nombre es lo que hace que una llave de otra bóveda no
// se lea como la de ésta.
type EnFichero struct{ Ruta string }

// DelServidor es el llavero de fichero en la ruta de siempre.
func DelServidor() EnFichero { return EnFichero{Ruta: RutaDeLaLlaveDelServidor()} }

// RutaDeLaLlaveDelServidor es `~/.esfinge-servidor`, o lo que diga
// ESFINGE_LLAVE_DEL_SERVIDOR.
//
// **No es la carpeta de la bóveda**, y eso es lo único que esta ruta tiene que
// cumplir: la bóveda vive en la carpeta de configuración, así que una copia de
// seguridad de ahí no se lleva la llave dentro.
//
// Se recorta lo que venga del entorno: un espacio de más en una variable no da un
// error, da la apariencia de funcionar apuntando a un sitio que no es.
func RutaDeLaLlaveDelServidor() string {
	if v := strings.TrimSpace(os.Getenv(VariableDeLaLlave)); v != "" {
		return v
	}
	casa, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(casa, ".esfinge-servidor")
}

// Nombre va **dentro de una frase**, no al principio: aquí no hay ventana que
// ponga un rótulo, y lo que se lee es «ábrela con la llave de este servidor».
func (EnFichero) Nombre() string { return "la llave de este servidor" }

// Hay dice si esta máquina tiene llave **ahora**: que el fichero esté y se
// entienda. Un fichero roto cuenta como que no hay, porque de ahí no sale ningún
// secreto y lo que toca es volver a activarlo.
func (f EnFichero) Hay() bool {
	if f.Ruta == "" {
		return false
	}
	_, err := f.leer()
	return err == nil
}

// Guardar deja el secreto con ese identificador, y reemplaza el que hubiera —que
// es cómo se rota—. El fichero se escribe de una pieza y a 0600.
func (f EnFichero) Guardar(id string, secreto []byte) error {
	if f.Ruta == "" {
		return ErrNoHay
	}
	llaves, err := f.leer()
	if err != nil {
		// Lo que no se entiende se reemplaza: aquí no hay nada que conservar que no
		// sea una llave, y una llave que no se puede leer no abre nada.
		llaves = map[string]string{}
	}
	llaves[id] = hex.EncodeToString(secreto)
	return f.escribir(llaves)
}

// Leer devuelve el secreto. **El motivo no se usa**: no hay diálogo que enseñarle
// a nadie, y ésa es justo la diferencia con Touch ID.
func (f EnFichero) Leer(id, _ string) ([]byte, error) {
	if f.Ruta == "" {
		return nil, ErrNoHay
	}
	llaves, err := f.leer()
	if err != nil {
		return nil, err
	}
	crudo, hay := llaves[id]
	if !hay {
		return nil, ErrNoEsta
	}
	secreto, err := hex.DecodeString(crudo)
	if err != nil || len(secreto) == 0 {
		return nil, ErrNoEsta
	}
	return secreto, nil
}

// Borrar quita ese identificador, y el fichero entero cuando se queda sin
// ninguno: un fichero vacío con ese nombre haría creer que ahí hay una llave.
func (f EnFichero) Borrar(id string) error {
	if f.Ruta == "" {
		return nil
	}
	llaves, err := f.leer()
	if err != nil {
		return nil
	}
	if _, hay := llaves[id]; !hay {
		return nil
	}
	delete(llaves, id)
	if len(llaves) == 0 {
		if err := os.Remove(f.Ruta); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return f.escribir(llaves)
}

func (f EnFichero) leer() (map[string]string, error) {
	datos, err := os.ReadFile(f.Ruta)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoHay
		}
		return nil, err
	}
	var llaves map[string]string
	if err := json.Unmarshal(datos, &llaves); err != nil || llaves == nil {
		return nil, ErrNoHay
	}
	return llaves, nil
}

func (f EnFichero) escribir(llaves map[string]string) error {
	return escritura.Atomica(f.Ruta, escritura.Opciones{CrearCarpeta: true}, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(llaves)
	})
}
