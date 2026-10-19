package cripto

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
)

// Alfabeto es el juego de caracteres con el que se genera una contraseña.
type Alfabeto struct {
	Nombre    string
	Runas     string // vacío en hexadecimal, que se genera aparte
	SeguroURL bool
	Aviso     string
}

// Los tres alfabetos disponibles. El hexadecimal es el que va primero por un
// motivo concreto: una contraseña con «/» parte la cadena de conexión —
// postgres://usuario:pa/ss@host deja de ser una URL— y el error que sale por el
// otro lado no menciona la contraseña por ninguna parte, así que se pierden
// horas buscando donde no es.
var (
	AlfHex = Alfabeto{
		Nombre:    "hex",
		SeguroURL: true,
	}
	AlfAlnum = Alfabeto{
		Nombre:    "alnum",
		Runas:     "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		SeguroURL: true,
	}
	AlfSimbolos = Alfabeto{
		Nombre: "simbolos",
		Runas:  "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!#$%&()*+,-.:;<=>?@[]^_{|}~",
		Aviso:  "No metas esta contraseña dentro de una URL: / + @ : # la parten en dos",
	}
)

// Alfabetos por nombre, para la línea de comandos.
var Alfabetos = map[string]Alfabeto{
	AlfHex.Nombre:      AlfHex,
	AlfAlnum.Nombre:    AlfAlnum,
	AlfSimbolos.Nombre: AlfSimbolos,
}

// Caracteres devuelve cuántos caracteres salen de pedir esos bytes de entropía,
// que en hexadecimal son dos por byte y en los demás alfabetos dependen de su
// tamaño.
func Caracteres(a Alfabeto, bytes int) int {
	if a.Runas == "" {
		return bytes * 2
	}
	bits := float64(bytes) * 8
	return int(math.Ceil(bits / math.Log2(float64(len([]rune(a.Runas))))))
}

// BytesParaCaracteres es el camino de vuelta: cuántos bytes de entropía hay que
// pedir para que salga esa cantidad de caracteres.
//
// Existe porque una contraseña se mide en caracteres cuando se va a pegar en un
// formulario que limita la longitud, y en bits cuando lo que importa es lo cara
// que sea de adivinar. Los dos números son el mismo dato mirado de dos maneras,
// y la interfaz deja mover cualquiera de los dos.
func BytesParaCaracteres(a Alfabeto, caracteres int) int {
	if a.Runas == "" {
		return (caracteres + 1) / 2
	}
	bits := float64(caracteres) * math.Log2(float64(len([]rune(a.Runas))))
	bytes := int(bits / 8)
	if bytes < 1 {
		bytes = 1
	}
	return bytes
}

// Generar devuelve una contraseña con la entropía de bytes indicados.
//
// En hexadecimal salen 2 caracteres por byte. En los demás alfabetos, tantos
// caracteres como hagan falta para alcanzar los mismos bits de entropía, de modo
// que --bytes signifique siempre lo mismo se elija el alfabeto que se elija.
// topeDeBytes es lo máximo que se acepta de entropía: más no aporta nada y no cabe en
// ningún sitio.
const topeDeBytes = 256

func Generar(a Alfabeto, bytes int) (string, error) {
	if bytes < 8 {
		return "", errors.New("Menos de 8 bytes no es una contraseña, es una invitación")
	}
	if bytes > topeDeBytes {
		return "", errors.New("Más de 256 bytes no aporta nada y no cabe en ningún sitio")
	}

	if a.Runas == "" {
		b := make([]byte, bytes)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		return hex.EncodeToString(b), nil
	}

	runas := []rune(a.Runas)
	bits := float64(bytes) * 8
	n := int(math.Ceil(bits / math.Log2(float64(len(runas)))))

	tope := big.NewInt(int64(len(runas)))
	out := make([]rune, n)
	for i := range out {
		// rand.Int reparte de forma uniforme sobre el alfabeto. Hacer el módulo de
		// un byte al azar sería más corto y estaría mal: sesgaría el resultado
		// hacia los primeros caracteres del alfabeto siempre que su tamaño no sea
		// una potencia de dos, que es el caso de los tres de aquí.
		v, err := rand.Int(rand.Reader, tope)
		if err != nil {
			return "", err
		}
		out[i] = runas[v.Int64()]
	}
	return string(out), nil
}

// MinimoDeCaracteres es el menor largo que se acepta con ese alfabeto, y **sale de los
// bits, no de un número redondo**.
//
// El suelo de esta casa son los 8 bytes de [Generar] —«menos de 8 bytes no es una
// contraseña, es una invitación»—, o sea 64 bits. Cuántos caracteres son eso depende del
// alfabeto: 16 en hexadecimal, 11 con letras y números, 10 con símbolos. Poner un mínimo
// igual para los tres sería o dejar pasar basura en hex o prohibir largos que con
// símbolos están bien.
func MinimoDeCaracteres(a Alfabeto) int {
	for n := 1; n <= 64; n++ {
		if BytesParaCaracteres(a, n) >= 8 {
			return n
		}
	}
	return 64
}

// MaximoDeCaracteres es el mayor largo que se acepta, por el mismo tope de 256 bytes que
// [Generar]: más no aporta nada y no cabe en ningún sitio.
func MaximoDeCaracteres(a Alfabeto) int { return Caracteres(a, topeDeBytes) }

// GenerarDeLargo devuelve una contraseña de **exactamente** esos caracteres.
//
// Existe porque un sitio limita lo que acepta **en caracteres** y no en bits: Apple, que
// pide una mayúscula, o cualquier formulario con un máximo. Hasta el 2026-10-19 el canal
// de los agentes solo podía pedir bytes de entropía, así que para acortar una contraseña
// había que generarla aparte y pegarla — es decir, **hacerla pasar por la conversación de
// un modelo**, que es justo lo que `generar` existe para evitar. Lo destapó el agente del
// cliente prometiéndole que podía acortarla.
//
// **Se pide de más y se recorta**, y eso no sesga nada: cada carácter sale de un sorteo
// uniforme e independiente (ver el porqué de `rand.Int` en [Generar]), así que quedarse
// con los primeros n es tan uniforme como el conjunto. Lo que **sí** haría falta mirar es
// lo contrario —rellenar a mano si faltaran—, y por eso se pide un byte de más en vez de
// ajustar al límite.
func GenerarDeLargo(a Alfabeto, caracteres int) (string, error) {
	if min := MinimoDeCaracteres(a); caracteres < min {
		return "", fmt.Errorf(
			"Con %s, menos de %d caracteres no es una contraseña: son menos de 64 bits",
			a.Nombre, min)
	}
	if max := MaximoDeCaracteres(a); caracteres > max {
		return "", fmt.Errorf(
			"Con %s, más de %d caracteres no aporta nada y no cabe en ningún sitio",
			a.Nombre, max)
	}
	// El byte de más es para no quedarse corto por el redondeo, **y no puede pasarse del
	// tope de [Generar]**: con hexadecimal, el largo máximo ya pide los 256 justos, así
	// que sumarle uno lo rechazaba. Lo cazó la prueba de los bordes, no la lectura.
	bytes := BytesParaCaracteres(a, caracteres) + 1
	if bytes > topeDeBytes {
		bytes = topeDeBytes
	}
	clave, err := Generar(a, bytes)
	if err != nil {
		return "", err
	}
	runas := []rune(clave)
	if len(runas) < caracteres {
		// No puede pasar —se pide un byte de más— y si pasara, lo que no se hace es
		// devolver una contraseña más corta de la que alguien va a dar por buena.
		return "", fmt.Errorf("Esfinge no ha sabido hacer una contraseña de %d caracteres", caracteres)
	}
	return string(runas[:caracteres]), nil
}
