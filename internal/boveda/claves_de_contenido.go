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

// tieneCadenaPropia dice si la cadena de esta bóveda es de verdad —vive en su sobre
// `contenido`— y no la degenerada del formato 1, cuya única clave es la de bóveda.
//
// Es la pregunta que decide **qué se sella dentro de una ranura de acceso**, y está
// escrita una sola vez a propósito: la misma pregunta repartida por el código es el
// patrón que este proyecto ya ha pagado tres veces.
func (b *Boveda) tieneCadenaPropia() bool { return b.ranura(RanuraDeContenido) >= 0 }

// loQueAbreUnaRanuraDeAcceso es el secreto que se sella hacia la identidad de quien
// recibe el acceso: **la cadena entera** si esta bóveda la tiene, y la clave de bóveda
// si todavía es del formato 1 (ADR 0059).
//
// La cadena entera, y no solo la época de ahora, porque al fundir hace falta poder
// descifrar el propio fichero local, que puede haberse quedado en una época anterior.
// Lo dice la cabecera de este fichero y es la razón de que esto sea una cadena.
//
// Las dos formas conviven: migrar no toca las ranuras que ya estaban, y una que lleva la
// clave de bóveda desenvuelve la cadena igual. Quien decide cuál es, al abrir, es **el
// claro** (ver `AbrirCompartidaBytes`), nunca el `codificacion` del JSON de fuera.
func (b *Boveda) loQueAbreUnaRanuraDeAcceso() ([]byte, error) {
	if b.tieneCadenaPropia() {
		return b.claves.bytes()
	}
	// En el formato 1 la cadena es la degenerada, así que su única clave ya es la de
	// bóveda: no hace falta mirar `b.llave` ni distinguir al dueño del titular.
	k, hay := b.claves.clave(b.doc.Epoca)
	if !hay {
		// La época declarada sin su clave es el estado que no debería existir; se
		// contesta como `guardar`, con la misma frase, en vez de sellar un secreto
		// vacío que nadie podría usar.
		return nil, ErrCerrada
	}
	return append([]byte(nil), k...), nil
}

// ponerEpoca deja la cadena y la época declarada **a la vez**, y se niega si esa cadena
// no lleva esa clave.
//
// **Es el riesgo número uno de la ADR 0059.** Una época declarada sin la clave con la que
// se va a cifrar el cuerpo no da un error al escribir: da un fichero que no vuelve a
// abrirse nunca, y con un mensaje —«alguien ha editado el fichero a mano»— que manda a
// mirar donde no hay nada. Por eso las dos cosas no se pueden poner por separado, y por
// eso esto es una función y no dos asignaciones en cada sitio que las necesite.
func (b *Boveda) ponerEpoca(claves cadenaDeClaves, epoca string) error {
	if _, hay := claves.clave(epoca); !hay {
		return fmt.Errorf("%w (la época %q no tiene su clave en la cadena)", ErrManipulada, epoca)
	}
	b.claves, b.doc.Epoca = claves, epoca
	return nil
}

// clavesQueAbren devuelve una cadena con la que se puede descifrar ese documento, que
// puede no ser la de aquí: al fundir, el otro lado pudo volver a cifrar la bóveda.
//
// El dueño la saca **del propio documento**, porque el sobre `contenido` va envuelto con
// la clave de bóveda y ésa no cambia nunca. Un titular no puede: la suya se la sellaron
// hacia su identidad, y para recoger la nueva hace falta su bóveda personal, que al
// fundir no está delante. A él se le dice que cierre y vuelva a abrir, que es por donde
// pasa su identidad.
func (b *Boveda) clavesQueAbren(doc documento) (cadenaDeClaves, error) {
	if _, hay := b.claves.clave(doc.Epoca); hay {
		return b.claves, nil
	}
	if b.sinClaveDeBoveda() {
		return cadenaDeClaves{}, ErrEpocaQueNoSeTiene
	}
	claves, err := laCadenaDe(doc, b.llave)
	if err != nil {
		return cadenaDeClaves{}, err
	}
	if _, hay := claves.clave(doc.Epoca); !hay {
		return cadenaDeClaves{}, ErrEpocaQueNoSeTiene
	}
	return claves, nil
}

// ladoDeLasClaves dice de qué lado de una fusión salen los sobres que llevan claves.
type ladoDeLasClaves int

const (
	// clavesDeLosDos es que los dos lados están en la misma época y no hay nada que
	// forzar: cada sobre se decide por su cuenta, como siempre. Es el caso normal y el
	// único que existe en el formato 1.
	clavesDeLosDos ladoDeLasClaves = iota
	clavesDeAqui
	clavesDeAlla
)

// llevaClaves dice si ese tipo de ranura lleva dentro con qué descifrar el cuerpo.
//
// Son dos: el sobre `contenido`, que es la cadena, y las de acceso, que llevan la cadena
// sellada hacia cada titular. Las demás —la maestra, la de recuperación, la del sistema,
// `boveda-principal`— envuelven la **clave de bóveda**, que no cambia nunca, así que una
// rotación no las toca y la fusión tampoco tiene que tratarlas distinto.
func llevaClaves(tipo string) bool {
	return tipo == RanuraDeContenido || IDDeAcceso(tipo) != ""
}

// ladoDeLaFusion es el lado de una fusión entera: «los dos» mientras las épocas sean la
// misma, que es el caso normal y el único que existe en el formato 1.
//
// **Una sola función, y la llaman la fusión de verdad y la prueba cruzada.** Escrita dos
// veces se desalinea sola, y con la cruzada preguntándole lo mismo a los dos núcleos eso
// significaría comparar dos respuestas a dos preguntas distintas.
func ladoDeLaFusion(epocaL, epocaR string, l, r []sobre) ladoDeLasClaves {
	if epocaL == epocaR {
		return clavesDeLosDos
	}
	return ladoQueGanaLaEpoca(l, r)
}

// ladoQueGanaLaEpoca dice con qué clave queda cifrado el cuerpo cuando los dos lados
// están en épocas distintas, o sea cuando uno de los dos ha rotado.
//
// Lo decide **el sobre `contenido`**, que es quien lleva la cadena, y con las dos reglas
// del final de `fundirSobres`: el más reciente, y a igualdad de fecha el contenedor
// mayor. **Sin base, a propósito**: una rotación siempre escribe un `contenido` nuevo, así
// que «el más reciente» y «el que cambió respecto de la base» dicen aquí lo mismo, y
// pedir la base obligaría a descifrarla antes de saber con qué clave se descifra.
//
// Y es **el único sitio** donde se decide: los sobres con claves no vuelven a pasar por
// las reglas de las ranuras. La misma regla escrita dos veces se desalinea sola, y aquí
// desalinearse significa un cuerpo cifrado con una clave que la ranura de alguien no
// lleva dentro.
func ladoQueGanaLaEpoca(l, r []sobre) ladoDeLasClaves {
	il, ir := indiceEnSobres(l, RanuraDeContenido), indiceEnSobres(r, RanuraDeContenido)
	switch {
	case il < 0 && ir < 0:
		return clavesDeLosDos
	case ir < 0:
		return clavesDeAqui
	case il < 0:
		return clavesDeAlla
	}
	sl, sr := l[il], r[ir]
	if sr.Creado > sl.Creado || (sr.Creado == sl.Creado && sr.Contenedor > sl.Contenedor) {
		return clavesDeAlla
	}
	return clavesDeAqui
}
