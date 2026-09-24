package boveda

// La ranura que abre la bóveda con lo que guarda el sistema: Touch ID en macOS,
// Windows Hello en Windows (fase C, `docs/desbloqueo-del-sistema.md`).
//
// La bóveda ya sabía hacer esto desde la [ADR 0023]: **los tipos de ranura son
// una lista abierta** y cada sobre es un contenedor `ESF1` corriente con la clave
// de bóveda dentro. Lo único que añade este fichero es envolverla con **un
// secreto binario** en vez de con algo que alguien teclea, y poder abrir
// probando **solo** esa ranura.
//
// Tres cosas que no son detalles:
//
//   - **`PerfilLlave`, no `PerfilInteractivo`.** El coste alto de Argon2id existe
//     para que una contraseña humana aguante un diccionario. Aquí el secreto son
//     32 bytes al azar que guarda el sistema: no hay diccionario que valga, y el
//     coste solo se pagaría al abrir. Es el mismo razonamiento que el del cuerpo.
//   - **Abrir mira solo su ranura.** Si se probara como se prueba una contraseña
//     tecleada —contra todos los sobres— cada desbloqueo con la huella pagaría
//     antes una derivación interactiva entera contra la ranura maestra, que es
//     justo el segundo que esto viene a quitar.
//   - **No se sube nunca.** Ya lo garantiza `ranurasLocales` en `sincronizar.go`,
//     con su prueba: es de un equipo, y subirla sería darle al servidor una
//     segunda puerta a la bóveda de todos los demás.
//
// Y lo que esta ranura **no** es, dicho aquí para que se lea al lado del código:
// sin firmar la aplicación, el secreto lo guarda el sistema sin poder atarlo a
// Esfinge, así que esto **es un cerrojo y no una llave**. Protege de quien se
// siente delante de tu ordenador desbloqueado, no de un programa que corra como
// tú. Está dicho en `docs/seguridad.md` y en la pantalla donde se activa.

import (
	"errors"
	"os"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// RanuraDelSistema es el tipo de sobre que guarda la clave de bóveda envuelta con
// el secreto del llavero del sistema. **Está en `ranurasLocales`**: no se sube.
const RanuraDelSistema = "llavero-del-sistema"

// ErrSinRanuraDelSistema: esta bóveda no tiene desbloqueo del sistema puesto.
var ErrSinRanuraDelSistema = errors.New("Esta bóveda no se abre con el sistema en este equipo")

// SecretoDelSistema fabrica el secreto que se le da a guardar al sistema. Son 32
// bytes al azar: no se deriva de nada y no hay que recordarlo.
func SecretoDelSistema() ([]byte, error) { return cripto.Azar(32) }

// PonerRanuraDelSistema envuelve la clave de esta bóveda con ese secreto, y
// reemplaza la ranura si ya había una. Exige la bóveda abierta, como rotar la
// clave de recuperación: es lo que impide ponerla desde fuera.
func (b *Boveda) PonerRanuraDelSistema(secreto []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	if len(secreto) < 16 {
		return errors.New("El secreto del sistema es demasiado corto")
	}
	s, err := envolverConSecreto(RanuraDelSistema, secreto, b.llave, ahora().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	if i := b.ranura(RanuraDelSistema); i >= 0 {
		b.doc.Sobres[i] = s
	} else {
		b.doc.Sobres = append(b.doc.Sobres, s)
	}
	return b.guardar()
}

// QuitarRanuraDelSistema deja la bóveda sin ese desbloqueo. Lo que haya guardado
// el sistema lo borra quien lo guardó: aquí solo se va el sobre, y sin él ese
// secreto ya no abre nada.
func (b *Boveda) QuitarRanuraDelSistema() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	i := b.ranura(RanuraDelSistema)
	if i < 0 {
		return nil
	}
	b.doc.Sobres = append(b.doc.Sobres[:i], b.doc.Sobres[i+1:]...)
	return b.guardar()
}

// TieneRanuraDelSistema dice si esta bóveda se puede abrir con el sistema.
func (b *Boveda) TieneRanuraDelSistema() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ranura(RanuraDelSistema) >= 0
}

// RanuraDelSistemaEn dice si el fichero de esa ruta trae la ranura, **sin
// abrirlo**: es lo que mira la ventana para saber si ofrecer la huella antes de
// que nadie escriba nada. Lo de fuera del documento va en claro a propósito.
func RanuraDelSistemaEn(ruta string) bool {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return false
	}
	doc, err := leerDocumento(datos)
	if err != nil {
		return false
	}
	for _, s := range doc.Sobres {
		if s.Tipo == RanuraDelSistema {
			return true
		}
	}
	return false
}

// AbrirConElSistema abre la bóveda del disco con el secreto que guardaba el
// sistema, **probando solo esa ranura**.
func AbrirConElSistema(ruta string, secreto []byte) (*Boveda, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	doc, err := leerDocumento(datos)
	if err != nil {
		return nil, err
	}
	i := -1
	for j, s := range doc.Sobres {
		if s.Tipo == RanuraDelSistema {
			i = j
			break
		}
	}
	if i < 0 {
		return nil, ErrSinRanuraDelSistema
	}
	llave, err := cripto.AbrirTexto(doc.Sobres[i].Contenedor, secreto)
	if err != nil {
		// El secreto que guardaba el sistema ya no abre esta bóveda. Pasa si la
		// bóveda se restauró de una copia anterior o si se quitó el desbloqueo en
		// otro equipo y la copia llegó aquí: no es un fallo del sistema.
		return nil, ErrSinRanuraDelSistema
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
