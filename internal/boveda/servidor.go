package boveda

// La ranura que abre la bóveda **en un servidor**, donde no hay nadie delante
// para teclear la contraseña maestra (ADR 0057).
//
// La máquina está en `ranura_con_secreto.go`, compartida con la del sistema: de
// las dos, lo único que cambia es **quién guarda el secreto**. Aquí lo guarda un
// fichero del disco, y de eso salen las cuatro cosas que hay que leer antes de
// tocar nada:
//
//   - **Lo que protege es menos que un cerrojo, y hay que decirlo con esas
//     palabras.** Touch ID protege de quien se sienta delante de tu ordenador
//     desbloqueado; esto **no protege de nadie que entre en el servidor**, porque
//     la llave está ahí al lado. Lo que gana a cambio está escrito en la ADR y en
//     `docs/seguridad.md`: lo que hoy son ficheros `.env` en claro queda cifrado
//     en reposo, se cambia desde el Mac, deja rastro y **se revoca rotando la
//     clave** en vez de entrando a borrar un fichero.
//   - **El fichero de la llave va FUERA de la carpeta de las bóvedas.** Es la
//     misma regla de la ADR 0044: con la llave al lado, quien copie la carpeta
//     abre la bóveda, y eso no es un cerrojo peor, es quitar la puerta. Quien
//     decide dónde va es quien guarda el secreto (`internal/llavero`), no esto.
//   - **No se activa sola en ningún sitio.** Se pide a mano, una vez, con la
//     bóveda abierta —o sea sabiendo la maestra—. En un Linux de escritorio sigue
//     sin haber nada: lo que ahí sería un cerrojo sin cerradura, en un servidor es
//     justo lo que hace falta, porque allí no hay a quién proteger de quien se
//     sienta delante — el servidor *es* el que trabaja.
//   - **No se sube nunca.** Lo garantiza `ranurasLocales` en `sincronizar.go`, con
//     su prueba, igual que la del sistema: es de un equipo y de uno solo.
//
// Y una que no es de esta ranura y manda sobre dónde se pone: va **solo en la
// bóveda personal del servidor**, nunca en una de proyecto. Un proyecto se abre
// con la clave de bóveda de la personal (ADR 0050) y una compartida con su
// identidad (ADR 0052), así que esta ranura es **la única llave en disco** y lo
// demás cuelga de ella en cascada. Puesta en un proyecto no serviría de mucho
// —no se sube, así que sería de este equipo y de nadie más— y habría tantas
// llaves en el disco como proyectos.

import (
	"errors"
)

// RanuraDelServidor es el tipo de sobre que guarda la clave de bóveda envuelta con
// la llave que este servidor tiene en un fichero. **Está en `ranurasLocales`**: no
// se sube.
const RanuraDelServidor = "llave-del-servidor"

// ErrSinRanuraDelServidor: esta bóveda no se abre con la llave de este servidor.
// Lo dice igual cuando no tiene la ranura y cuando la llave ya no la abre, y el
// porqué está en `errRanuraQueNoAbre`.
var ErrSinRanuraDelServidor = errors.New("Esta bóveda no se abre con la llave de este servidor")

// SecretoDelServidor fabrica la llave que se le da a guardar al servidor. Son 32
// bytes al azar: no se deriva de nada y no hay que recordarla.
func SecretoDelServidor() ([]byte, error) { return secretoDeRanura() }

// PonerRanuraDelServidor envuelve la clave de esta bóveda con esa llave, y
// reemplaza la ranura si ya había una —que es también cómo se rota la llave del
// servidor—. Exige la bóveda abierta: es lo que impide activarlo sin saber la
// maestra.
func (b *Boveda) PonerRanuraDelServidor(secreto []byte) error {
	return b.ponerRanuraConSecreto(RanuraDelServidor, secreto)
}

// QuitarRanuraDelServidor deja la bóveda sin ese desbloqueo. La llave la borra
// quien la guardó: aquí solo se va el sobre, y sin él esa llave ya no abre nada.
func (b *Boveda) QuitarRanuraDelServidor() error { return b.quitarRanura(RanuraDelServidor) }

// TieneRanuraDelServidor dice si esta bóveda se puede abrir con la llave del
// servidor.
func (b *Boveda) TieneRanuraDelServidor() bool { return b.tieneRanura(RanuraDelServidor) }

// RanuraDelServidorEn dice si el fichero de esa ruta trae la ranura, **sin
// abrirlo**: es lo que hay que mirar para saber si se puede abrir sin preguntar
// nada, antes de preguntar nada.
func RanuraDelServidorEn(ruta string) bool { return ranuraEn(ruta, RanuraDelServidor) }

// AbrirConElServidor abre la bóveda del disco con la llave de este servidor,
// **probando solo esa ranura**.
func AbrirConElServidor(ruta string, secreto []byte) (*Boveda, error) {
	b, err := abrirConSecretoDeRanura(ruta, RanuraDelServidor, secreto)
	if errors.Is(err, errRanuraQueNoAbre) {
		return nil, ErrSinRanuraDelServidor
	}
	return b, err
}
