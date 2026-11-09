package boveda

// La cadena de claves de contenido (ADR 0059).
//
// Hasta aquí, **todas las ranuras envolvían la misma cosa**: la clave de bóveda, que es
// también la que cifra el cuerpo. Eso hacía imposible lo que la ADR 0052 prometió —rotar
// al quitarle el acceso a alguien— y, de paso, hacía que dar acceso a una persona le
// entregara la clave que protege la contraseña maestra.
//
// Ahora el cuerpo se cifra con una **clave de contenido**, y la clave de bóveda pasa a
// envolver la cadena de claves de contenido en vez del cuerpo. Rotar es entonces una
// clave nueva, recifrar el cuerpo y volver a sellar la cadena **solo para los que
// quedan**: sin tocar la maestra, sin tocar la de recuperación y sin que nadie teclee
// nada.
//
// **Y por qué es una cadena y no una clave.** Porque la fusión lo exige, y esa es la
// razón de fondo y no la comodidad: un equipo que estuvo desconectado tiene su fichero
// local cifrado con la época anterior, y si esa clave no estuviera, al fundir no podría
// descifrar su propio lado y perdería lo que no hubiera subido. Lo mismo con la base. Y
// como la cadena va envuelta con **la clave de bóveda, que no cambia nunca**, cualquier
// equipo del dueño la tiene entera siempre, sea cual sea la época en la que se quedó.
//
// **El identificador va al azar y no es un contador**, y es la decisión menos obvia de
// todo esto: dos equipos del mismo dueño que roten a la vez desde la época 2 crearían
// los dos «la 3» con claves distintas, y al unir las cadenas una taparía a la otra —o
// peor, se quedaría la etiqueta de una con la clave de la otra—. Con un identificador
// único la cadena es un **conjunto** y la colisión no existe. El número `n` solo ordena y
// sirve para poder decir «época 2» en una pantalla; puede repetirse y no pasa nada.

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// marcaDeLaCadena es lo que distingue una cadena de una clave de bóveda a secas.
//
// **Es el claro quien lo dice, no el JSON de fuera.** Una ranura de acceso vieja lleva
// los 43 caracteres de la clave de bóveda y una nueva lleva esta cadena; decidir cuál es
// mirando el campo `codificacion` sería decidir cómo interpretar un secreto por un dato
// que cualquiera que pueda escribir el fichero puede cambiar. Ese campo se escribe porque
// documenta, no porque nadie se fíe de él.
const marcaDeLaCadena = "claves-de-contenido"

// VersionDeLaCadena es la del formato de la propia cadena, por dentro. Es aparte del
// formato de la bóveda: una cadena de una versión que no se entiende no se toca.
const VersionDeLaCadena = 1

// Los motivos por los que nace una clave. Son etiquetas, no frases: lo que se lee en
// pantalla se escribe en la pantalla.
const (
	PorMigracion  = "migracion"
	PorRevocacion = "revocacion"
	PorLaMano     = "mano"
)

// RanuraDeContenido es el sobre que lleva la cadena, envuelta con la clave de bóveda.
//
// **Se sube**, al contrario que la del sistema y la del servidor: sin ella, el segundo
// equipo del dueño abriría la bóveda —tiene la clave de bóveda— y no sabría con qué
// descifrar el cuerpo. Es el mismo argumento que `boveda-principal`.
const RanuraDeContenido = "contenido"

// ErrCadenaNueva es una cadena de una versión que este código no entiende.
var ErrCadenaNueva = errors.New("Las claves de esta bóveda las escribió una versión de Esfinge más nueva; actualiza antes de tocarla")

// ClaveDeContenido es una clave con la que se ha cifrado el cuerpo alguna vez.
type ClaveDeContenido struct {
	// ID son 8 bytes en hexadecimal, al azar. Ver la cabecera: no es un contador.
	ID string `json:"id"`
	// N ordena y se enseña. No decide nada.
	N int `json:"n"`
	// Clave son 32 bytes en base64url, 43 caracteres, **manejados como texto** por lo
	// mismo que la clave de bóveda: `--clave-fichero` recorta los saltos de línea del
	// final, y una clave binaria que acabara en 0x0A se truncaría en silencio.
	Clave  string `json:"clave"`
	Creada string `json:"creada"`
	Porque string `json:"porque,omitempty"`
}

// cadenaDeClaves es lo que va dentro del sobre `contenido` y dentro de cada ranura de
// acceso: todas las claves con las que se ha cifrado esta bóveda, y cuál es la de ahora.
type cadenaDeClaves struct {
	Esfinge string             `json:"esfinge"`
	Version int                `json:"version"`
	Actual  string             `json:"actual"`
	Claves  []ClaveDeContenido `json:"claves"`
}

// nuevaCadena arranca una cadena con su primera clave, que es lo que pasa al migrar una
// bóveda de proyecto del formato 1.
func nuevaCadena(porque string) (cadenaDeClaves, error) {
	c := cadenaDeClaves{Esfinge: marcaDeLaCadena, Version: VersionDeLaCadena}
	if err := c.poner(porque); err != nil {
		return cadenaDeClaves{}, err
	}
	return c, nil
}

// poner añade una clave nueva y la deja como la actual.
//
// **No quita ninguna.** Ver la cabecera: sin las anteriores, un equipo que vuelve de
// estar desconectado no puede abrir su propio fichero.
func (c *cadenaDeClaves) poner(porque string) error {
	bruta, err := cripto.Azar(32)
	if err != nil {
		return err
	}
	defer cripto.Borrar(bruta)
	id, err := cripto.Azar(8)
	if err != nil {
		return err
	}
	k := ClaveDeContenido{
		ID:     hex.EncodeToString(id),
		N:      c.siguienteN(),
		Clave:  base64.RawURLEncoding.EncodeToString(bruta),
		Creada: ahora().UTC().Format(time.RFC3339),
		Porque: porque,
	}
	c.Esfinge, c.Version = marcaDeLaCadena, VersionDeLaCadena
	c.Claves = append(c.Claves, k)
	c.Actual = k.ID
	c.ordenar()
	return nil
}

func (c cadenaDeClaves) siguienteN() int {
	n := 0
	for _, k := range c.Claves {
		if k.N > n {
			n = k.N
		}
	}
	return n + 1
}

// ordenar deja la cadena en **una sola forma posible**, que es lo que permite que los dos
// núcleos escriban los mismos bytes. Sin esto, el mismo contenido daría contenedores
// distintos en Go y en el espejo, y las dos bóvedas se pasarían la una a la otra sin fin.
func (c *cadenaDeClaves) ordenar() {
	sort.SliceStable(c.Claves, func(i, j int) bool {
		if c.Claves[i].N != c.Claves[j].N {
			return c.Claves[i].N < c.Claves[j].N
		}
		return c.Claves[i].ID < c.Claves[j].ID
	})
}

// clave devuelve la de esa época, como bytes ASCII, igual que `b.llave`.
func (c cadenaDeClaves) clave(id string) ([]byte, bool) {
	for _, k := range c.Claves {
		if k.ID == id {
			return []byte(k.Clave), true
		}
	}
	return nil, false
}

// deLaMasNuevaALaMasVieja son las claves en el orden en que hay que probarlas cuando la
// época declarada en el JSON de fuera no abre: esa declaración es una pista y no una
// autoridad, porque va en claro y cualquiera que pueda escribir el fichero la cambia.
func (c cadenaDeClaves) deLaMasNuevaALaMasVieja() []ClaveDeContenido {
	out := append([]ClaveDeContenido(nil), c.Claves...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].ID > out[j].ID
	})
	return out
}

func (c cadenaDeClaves) vacia() bool { return len(c.Claves) == 0 }

// bytes es la forma en que la cadena se guarda: ordenada y nada más. El orden de los
// campos lo fija la estructura, que en Go es el de declaración.
func (c cadenaDeClaves) bytes() ([]byte, error) {
	c.ordenar()
	return json.Marshal(c)
}

// esCadena dice si un claro recién abierto es una cadena de claves y no una clave de
// bóveda a secas. Mira **el claro**, por lo que dice `marcaDeLaCadena`.
func esCadena(claro []byte) bool {
	if len(claro) == 0 || claro[0] != '{' {
		return false
	}
	return strings.Contains(string(claro), `"`+marcaDeLaCadena+`"`)
}

// leerCadena lee lo que salga de un sobre `contenido` o de una ranura de acceso nueva.
func leerCadena(claro []byte) (cadenaDeClaves, error) {
	var c cadenaDeClaves
	if err := json.Unmarshal(claro, &c); err != nil {
		return cadenaDeClaves{}, ErrManipulada
	}
	if c.Esfinge != marcaDeLaCadena {
		return cadenaDeClaves{}, ErrManipulada
	}
	if c.Version > VersionDeLaCadena {
		return cadenaDeClaves{}, ErrCadenaNueva
	}
	if c.vacia() {
		return cadenaDeClaves{}, ErrManipulada
	}
	if _, hay := c.clave(c.Actual); !hay {
		return cadenaDeClaves{}, fmt.Errorf("%w (la cadena dice que la época de ahora es %q y no la lleva)", ErrManipulada, c.Actual)
	}
	return c, nil
}

// cadenaDeUnaClave es la cadena de una bóveda del formato 1: una sola clave, que es la de
// bóveda, con la época vacía en los dos lados —el documento tampoco la declara—.
//
// **Es lo que evita que haya dos caminos.** Sin esto, cada sitio que descifra el cuerpo
// tendría que preguntar «¿esta bóveda tiene claves de contenido?», y esa pregunta
// repetida en diez sitios es exactamente el patrón que este proyecto ya ha pagado tres
// veces. Con la cadena degenerada, el formato de siempre es un caso de lo nuevo.
func cadenaDeUnaClave(llave []byte) cadenaDeClaves {
	return cadenaDeClaves{
		Esfinge: marcaDeLaCadena,
		Version: VersionDeLaCadena,
		Claves:  []ClaveDeContenido{{Clave: string(llave), N: 1}},
	}
}

// laCadenaDe saca de un documento la cadena de claves, con la clave de bóveda.
func laCadenaDe(doc documento, llave []byte) (cadenaDeClaves, error) {
	i := indiceDeRanura(doc, RanuraDeContenido)
	if i < 0 {
		return cadenaDeUnaClave(llave), nil
	}
	claro, err := cripto.AbrirTexto(doc.Sobres[i].Contenedor, llave)
	if err != nil {
		return cadenaDeClaves{}, ErrManipulada
	}
	defer cripto.Borrar(claro)
	return leerCadena(claro)
}
