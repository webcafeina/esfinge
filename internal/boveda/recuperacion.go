package boveda

import (
	"crypto/sha256"
	"errors"
	"strings"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// La clave de recuperación: lo que abre la bóveda cuando se olvida la maestra.
//
// **Está pensada para escribirse en un papel y teclearse un año después**, y de
// ahí sale todo lo que sigue.
//
//   - **Base32 de Crockford**, que quita las cuatro letras que se confunden al
//     leer a mano: I, L, O y U. Y al teclearla se acepta lo que la gente escribe
//     de verdad: minúsculas, con guiones o sin ellos, con «O» donde va un cero y
//     con «I» o «L» donde va un uno.
//   - **Una suma de control**, que es la diferencia entre «te has equivocado al
//     copiarla» y «clave incorrecta». Sin ella, una errata se parecería
//     exactamente a haber perdido la bóveda, y eso es un mal rato que no hace
//     falta pasar. Además se comprueba **antes** de gastar medio segundo en
//     derivar.
//   - **Grupos de cuatro**, porque copiar veintiocho caracteres seguidos a mano
//     es cómo se cometen las erratas.
//
// 128 bits de `crypto/rand`. No hacen falta más: lo que protege esto no es la
// longitud de la clave sino que nadie la vea.

// prefijo delante de la clave para que se reconozca de un vistazo dentro de un
// gestor de contraseñas ajeno, un correo o una nota.
const prefijo = "ESF"

// alfabeto de Crockford: 32 símbolos sin I, L, O ni U.
const alfabeto = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

const (
	bytesDeAzar = 16 // 128 bits
	grupo       = 4
)

// ErrChecksum dice que lo tecleado no es una clave de recuperación de Esfinge, o
// que se ha copiado mal. Es un error distinto de «no abre» a propósito.
var ErrChecksum = errors.New("Esa no parece una clave de recuperación de Esfinge: revísala, puede que falte o sobre un carácter")

// NuevaRecuperacion genera una clave nueva y la devuelve ya presentada, lista
// para enseñarla y apuntarla.
func NuevaRecuperacion() (string, error) {
	b, err := cripto.Azar(bytesDeAzar)
	if err != nil {
		return "", err
	}
	return presentar(conSuma(b)), nil
}

// conSuma añade el byte de control: el primero del SHA-256 de los datos.
func conSuma(b []byte) []byte {
	h := sha256.Sum256(b)
	return append(append([]byte(nil), b...), h[0])
}

// presentar convierte los bytes en la cadena que ve una persona.
func presentar(b []byte) string {
	var sb strings.Builder
	sb.WriteString(prefijo)

	texto := codificar(b)
	for i := 0; i < len(texto); i += grupo {
		fin := i + grupo
		if fin > len(texto) {
			fin = len(texto)
		}
		sb.WriteByte('-')
		sb.WriteString(texto[i:fin])
	}
	return sb.String()
}

// PareceRecuperacion dice si lo tecleado pretendía ser una clave de recuperación.
//
// Es lo que permite contestar «revísala» en vez de «no abre» cuando hay una
// errata. No decide nada por sí solo: primero se intenta abrir con lo que sea que
// hayan escrito —una contraseña maestra que empiece por ESF es rara pero
// legítima— y esto solo se mira cuando ya no ha abierto nada.
func PareceRecuperacion(tecleado string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(tecleado)), prefijo)
}

// Normalizar convierte lo que alguien teclea en la forma canónica, y comprueba
// la suma.
//
// **Lo que devuelve es lo que se usa como clave de la ranura**, y son los bytes
// ASCII de esta cadena, no los 16 bytes decodificados. Dos razones: que la clave
// de la ranura sea literalmente lo que la persona apuntó en el papel, y que
// `--clave-fichero` de la línea de comandos recorta los saltos de línea del
// final —con lo que una clave binaria que acabara en 0x0A se truncaría en
// silencio, y eso pasa una de cada cien veces—.
func Normalizar(tecleado string) (string, error) {
	limpio := strings.ToUpper(strings.TrimSpace(tecleado))
	limpio = strings.TrimPrefix(limpio, prefijo)

	var sb strings.Builder
	for _, r := range limpio {
		switch r {
		case '-', ' ', '\t', '\n', '\r', '_':
			continue
		// Las ambigüedades de siempre al copiar a mano de un papel.
		case 'O':
			sb.WriteByte('0')
		case 'I', 'L':
			sb.WriteByte('1')
		default:
			if !strings.ContainsRune(alfabeto, r) {
				return "", ErrChecksum
			}
			sb.WriteRune(r)
		}
	}

	texto := sb.String()
	b, err := decodificar(texto)
	if err != nil || len(b) != bytesDeAzar+1 {
		return "", ErrChecksum
	}
	h := sha256.Sum256(b[:bytesDeAzar])
	if h[0] != b[bytesDeAzar] {
		return "", ErrChecksum
	}
	return prefijo + texto, nil
}

// codificar y decodificar son base32 de Crockford a mano.
//
// No se usa `encoding/base32`: su alfabeto lleva I, L y O, que es justo lo que
// aquí no puede haber, y su relleno con «=» sobra en algo que se copia a mano.
func codificar(b []byte) string {
	var sb strings.Builder
	var acumulado, bits uint

	for _, x := range b {
		acumulado = acumulado<<8 | uint(x)
		bits += 8
		for bits >= 5 {
			bits -= 5
			sb.WriteByte(alfabeto[(acumulado>>bits)&31])
		}
	}
	if bits > 0 {
		sb.WriteByte(alfabeto[(acumulado<<(5-bits))&31])
	}
	return sb.String()
}

func decodificar(s string) ([]byte, error) {
	var out []byte
	var acumulado, bits uint

	for _, r := range s {
		i := strings.IndexRune(alfabeto, r)
		if i < 0 {
			return nil, ErrChecksum
		}
		acumulado = acumulado<<5 | uint(i)
		bits += 5
		if bits >= 8 {
			bits -= 8
			out = append(out, byte((acumulado>>bits)&0xFF))
		}
	}
	// **Los bits de relleno tienen que ser cero**, y esto lo encontró una prueba.
	//
	// Diecisiete bytes son 136 bits, que en grupos de cinco son 28 caracteres con
	// cuatro bits de sobra. Sin esta comprobación, esos cuatro bits no los mira
	// nadie: cambiar el último carácter de la clave no cambiaba nada al
	// decodificar, así que una errata justo ahí pasaba por buena y la suma de
	// control no llegaba a enterarse.
	if bits > 0 && acumulado&((1<<bits)-1) != 0 {
		return nil, ErrChecksum
	}
	return out, nil
}
