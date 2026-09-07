package cripto

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
func Generar(a Alfabeto, bytes int) (string, error) {
	if bytes < 8 {
		return "", errors.New("Menos de 8 bytes no es una contraseña, es una invitación")
	}
	if bytes > 256 {
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
