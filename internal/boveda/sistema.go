package boveda

// La ranura que abre la bóveda con lo que guarda el sistema: Touch ID en macOS,
// Windows Hello en Windows (fase C, `docs/desbloqueo-del-sistema.md`).
//
// La máquina está en `ranura_con_secreto.go`, compartida con la ranura del
// servidor (ADR 0057): de las dos, lo único que cambia es **quién guarda el
// secreto**. Ahí están el porqué del perfil barato y el de abrir mirando solo una
// ranura.
//
// Aquí lo que queda es lo propio de esta cara, y empieza por lo que hay que leer
// antes de tocar nada:
//
//   - **No se sube nunca.** Lo garantiza `ranurasLocales` en `sincronizar.go`,
//     con su prueba: es de un equipo, y subirla sería darle al servidor una
//     segunda puerta a la bóveda de todos los demás.
//   - Y lo que esta ranura **no** es: sin firmar la aplicación, el secreto lo
//     guarda el sistema sin poder atarlo a Esfinge, así que esto **es un cerrojo y
//     no una llave**. Protege de quien se siente delante de tu ordenador
//     desbloqueado, no de un programa que corra como tú. Está dicho en
//     `docs/seguridad.md` y en la pantalla donde se activa.

import (
	"errors"
)

// RanuraDelSistema es el tipo de sobre que guarda la clave de bóveda envuelta con
// el secreto del llavero del sistema. **Está en `ranurasLocales`**: no se sube.
const RanuraDelSistema = "llavero-del-sistema"

// ErrSinRanuraDelSistema: esta bóveda no tiene desbloqueo del sistema puesto.
var ErrSinRanuraDelSistema = errors.New("Esta bóveda no se abre con el sistema en este equipo")

// SecretoDelSistema fabrica el secreto que se le da a guardar al sistema. Son 32
// bytes al azar: no se deriva de nada y no hay que recordarlo.
func SecretoDelSistema() ([]byte, error) { return secretoDeRanura() }

// PonerRanuraDelSistema envuelve la clave de esta bóveda con ese secreto, y
// reemplaza la ranura si ya había una. Exige la bóveda abierta: es lo que impide
// ponerla desde fuera.
func (b *Boveda) PonerRanuraDelSistema(secreto []byte) error {
	return b.ponerRanuraConSecreto(RanuraDelSistema, secreto)
}

// QuitarRanuraDelSistema deja la bóveda sin ese desbloqueo. Lo que haya guardado
// el sistema lo borra quien lo guardó: aquí solo se va el sobre, y sin él ese
// secreto ya no abre nada.
func (b *Boveda) QuitarRanuraDelSistema() error { return b.quitarRanura(RanuraDelSistema) }

// TieneRanuraDelSistema dice si esta bóveda se puede abrir con el sistema.
func (b *Boveda) TieneRanuraDelSistema() bool { return b.tieneRanura(RanuraDelSistema) }

// RanuraDelSistemaEn dice si el fichero de esa ruta trae la ranura, **sin
// abrirlo**: es lo que mira la ventana para saber si ofrecer la huella antes de
// que nadie escriba nada.
func RanuraDelSistemaEn(ruta string) bool { return ranuraEn(ruta, RanuraDelSistema) }

// AbrirConElSistema abre la bóveda del disco con el secreto que guardaba el
// sistema, **probando solo esa ranura**.
func AbrirConElSistema(ruta string, secreto []byte) (*Boveda, error) {
	b, err := abrirConSecretoDeRanura(ruta, RanuraDelSistema, secreto)
	if errors.Is(err, errRanuraQueNoAbre) {
		return nil, ErrSinRanuraDelSistema
	}
	return b, err
}

// IDEn dice de qué bóveda es el fichero, **sin abrirlo**: el identificador va en
// claro, como la serie, porque la sincronización tiene que poder mirarlo sin la
// contraseña maestra.
//
// Lo usa la pantalla de desbloquear para saber si a **esta** bóveda ya se le
// ofreció el desbloqueo del sistema, que con la bóveda cerrada no se puede
// preguntar de otra forma.
func IDEn(ruta string) string {
	doc, err := documentoEn(ruta)
	if err != nil {
		return ""
	}
	return doc.ID
}
