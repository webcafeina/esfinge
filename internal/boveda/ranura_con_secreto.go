package boveda

// Una ranura envuelta con **un secreto binario** en vez de con algo que alguien
// teclea, y la forma de abrir probando **solo** esa ranura.
//
// La bóveda ya sabía hacer esto desde la [ADR 0023]: **los tipos de ranura son una
// lista abierta** y cada sobre es un contenedor `ESF1` corriente con la clave de
// bóveda dentro. Lo único que añade esto es con qué se envuelve.
//
// Hay dos caras, y entre ellas **lo único que cambia es quién guarda el secreto**:
//
//   - `sistema.go`: el llavero del sistema, donde hay alguien que pone el dedo
//     —Touch ID, Windows Hello— (ADR 0044).
//   - `servidor.go`: un fichero del disco, en un servidor donde no hay nadie
//     (ADR 0057).
//
// Está junto y no copiado dos veces por lo de siempre: la misma regla en dos
// sitios se desalinea sola, y aquí desalinearse es que una de las dos ranuras
// deje de abrir o —peor— que viaje. Cada cara pone su tipo de sobre y su error;
// todo lo demás es esto.
//
// Dos cosas que valen para las dos y no son detalles:
//
//   - **`PerfilLlave`, no `PerfilInteractivo`.** El coste alto de Argon2id existe
//     para que una contraseña humana aguante un diccionario. Aquí el secreto son
//     32 bytes al azar: no hay diccionario que valga, y el coste solo se pagaría
//     al abrir. Es el mismo razonamiento que el del cuerpo.
//   - **Abrir mira solo su ranura.** Si se probara como se prueba una contraseña
//     tecleada —contra todos los sobres— cada apertura pagaría antes una
//     derivación interactiva entera contra la ranura maestra, que es justo el
//     segundo que esto viene a quitar.

import (
	"errors"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// errRanuraQueNoAbre es interno, y cada cara lo traduce al error que se lee.
//
// **Que falte el sobre y que el secreto no lo abra se cuentan igual**, a
// propósito: de los dos casos se sale por el mismo sitio —tecleando la clave— y
// distinguirlos le diría a quien está probando secretos cuál de las dos cosas
// tiene delante.
var errRanuraQueNoAbre = errors.New("esa ranura no abre esta bóveda")

// secretoDeRanura fabrica el secreto que se le da a guardar a quien lo vaya a
// guardar. Son 32 bytes al azar: no se deriva de nada y no hay que recordarlo.
func secretoDeRanura() ([]byte, error) { return cripto.Azar(32) }

// ponerRanuraConSecreto envuelve la clave de esta bóveda con ese secreto, y
// reemplaza la ranura si ya había una —que es también cómo se rota—. Exige la
// bóveda abierta, como rotar la clave de recuperación: es lo que impide ponerla
// desde fuera.
func (b *Boveda) ponerRanuraConSecreto(tipo string, secreto []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	if len(secreto) < 16 {
		return errors.New("El secreto de esa ranura es demasiado corto")
	}
	s, err := envolverConSecreto(tipo, secreto, b.llave, ahora().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	if i := b.ranura(tipo); i >= 0 {
		b.doc.Sobres[i] = s
	} else {
		b.doc.Sobres = append(b.doc.Sobres, s)
	}
	return b.guardar()
}

// quitarRanura deja la bóveda sin ese desbloqueo. El secreto lo borra quien lo
// guardó: aquí solo se va el sobre, y sin él ese secreto ya no abre nada.
func (b *Boveda) quitarRanura(tipo string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	i := b.ranura(tipo)
	if i < 0 {
		return nil
	}
	b.doc.Sobres = append(b.doc.Sobres[:i], b.doc.Sobres[i+1:]...)
	return b.guardar()
}

func (b *Boveda) tieneRanura(tipo string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ranura(tipo) >= 0
}

// ranuraEn dice si el fichero de esa ruta trae la ranura, **sin abrirlo**: es lo
// que hay que poder preguntar antes de que nadie escriba nada. Lo de fuera del
// documento va en claro a propósito.
func ranuraEn(ruta, tipo string) bool {
	doc, err := documentoEn(ruta)
	if err != nil {
		return false
	}
	for _, s := range doc.Sobres {
		if s.Tipo == tipo {
			return true
		}
	}
	return false
}

// abrirConSecretoDeRanura abre la bóveda del disco con ese secreto, **probando
// solo esa ranura**.
func abrirConSecretoDeRanura(ruta, tipo string, secreto []byte) (*Boveda, error) {
	doc, err := documentoEn(ruta)
	if err != nil {
		return nil, err
	}
	i := -1
	for j, s := range doc.Sobres {
		if s.Tipo == tipo {
			i = j
			break
		}
	}
	if i < 0 {
		return nil, errRanuraQueNoAbre
	}
	llave, err := cripto.AbrirTexto(doc.Sobres[i].Contenedor, secreto)
	if err != nil {
		// El secreto ya no abre esta bóveda. Pasa si se restauró de una copia
		// anterior o si se quitó el desbloqueo en otro equipo y la copia llegó
		// aquí: no es un fallo de quien guardaba el secreto.
		return nil, errRanuraQueNoAbre
	}
	return conLlave(ruta, doc, llave, true)
}

func envolverConSecreto(tipo string, secreto, llave []byte, cuando string) (sobre, error) {
	texto, err := cripto.SellarTexto(llave, secreto, cripto.PerfilLlave)
	if err != nil {
		return sobre{}, err
	}
	return sobre{Tipo: tipo, Creado: cuando, Contenedor: texto}, nil
}
