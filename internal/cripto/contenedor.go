package cripto

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
)

// Errores que el programa distingue de cara al usuario y en el código de salida.
// «La clave no es correcta» y «esto no es un contenedor de Esfinge» piden cosas
// distintas de quien los recibe: reintentar en un caso, buscar el fichero bueno
// en el otro.
var (
	// ErrClaveIncorrecta es deliberadamente ambiguo, porque la ambigüedad es la
	// verdad: una etiqueta que no cuadra puede ser una clave equivocada, un
	// contenido manipulado o un texto que llegó a medias, y sin la clave correcta
	// no hay forma de saber cuál de las tres. Nombrar las tres ahorra el rato que
	// se pierde comprobando una clave que estaba bien.
	ErrClaveIncorrecta = errors.New("La clave no es correcta, o el contenido llegó alterado o cortado")
	ErrFormato         = errors.New("Esto no parece un contenedor de Esfinge")
	ErrTruncado        = errors.New("El contenedor está incompleto: le falta el final")
	ErrDanado          = errors.New("El contenedor llegó dañado o cortado: la clave es correcta, el contenido no")
	ErrVersion         = errors.New("Contenedor de una versión de Esfinge más nueva que esta")
)

// Cabecera del contenedor, en claro y autenticada como datos asociados: alterar
// un parámetro de derivación invalida la etiqueta del primer segmento.
//
//	"ESF1" | versión(1) | modo(1) | memoria(4 BE) | pasadas(4 BE) | paralelismo(1) | sal(16) | nonce(24)
const (
	magia      = "ESF1"
	version    = 1
	tamNonce   = chacha20poly1305.NonceSizeX // 24
	tamEtiqueta = chacha20poly1305.Overhead   // 16
	tamCabecera = 4 + 1 + 1 + 4 + 4 + 1 + tamSal + tamNonce // 55
)

// Modo distingue el mensaje de un solo bloque del flujo por segmentos.
type Modo uint8

const (
	// ModoUnico sella todo el contenido con una sola etiqueta. Es lo que se usa
	// para una contraseña o un secreto corto.
	ModoUnico Modo = 0
	// ModoFlujo trocea el contenido en segmentos con una etiqueta cada uno, para
	// no cargar un fichero entero en memoria.
	ModoFlujo Modo = 1
)

// Prefijo del formato de texto. Se emite como ESF1.<base64url> en una sola línea:
// base64url no tiene «/» ni «+», así que el resultado sobrevive dentro de una URL,
// de un .env o de un JSON sin escapar nada.
const Prefijo = magia + "."

var codificacion = base64.RawURLEncoding

type cabecera struct {
	Modo  Modo
	Par   Parametros
	Sal   [tamSal]byte
	Nonce [tamNonce]byte
}

func (c cabecera) bytes() []byte {
	b := make([]byte, 0, tamCabecera)
	b = append(b, magia...)
	b = append(b, version, byte(c.Modo))
	b = binary.BigEndian.AppendUint32(b, c.Par.Memoria)
	b = binary.BigEndian.AppendUint32(b, c.Par.Pasadas)
	b = append(b, c.Par.Paralelismo)
	b = append(b, c.Sal[:]...)
	b = append(b, c.Nonce[:]...)
	return b
}

func leerCabecera(b []byte) (cabecera, error) {
	var c cabecera
	if len(b) < tamCabecera {
		return c, ErrFormato
	}
	if string(b[:4]) != magia {
		return c, ErrFormato
	}
	if b[4] != version {
		return c, ErrVersion
	}
	c.Modo = Modo(b[5])
	if c.Modo != ModoUnico && c.Modo != ModoFlujo {
		return c, ErrFormato
	}
	c.Par.Memoria = binary.BigEndian.Uint32(b[6:10])
	c.Par.Pasadas = binary.BigEndian.Uint32(b[10:14])
	c.Par.Paralelismo = b[14]
	if err := c.Par.validar(); err != nil {
		return c, err
	}
	copy(c.Sal[:], b[15:15+tamSal])
	copy(c.Nonce[:], b[15+tamSal:tamCabecera])
	return c, nil
}

func nuevaCabecera(modo Modo, p Parametros) (cabecera, error) {
	var c cabecera
	c.Modo, c.Par = modo, p

	sal, err := azar(tamSal)
	if err != nil {
		return c, err
	}
	copy(c.Sal[:], sal)

	nonce, err := azar(tamNonce)
	if err != nil {
		return c, err
	}
	copy(c.Nonce[:], nonce)

	// En modo flujo los cinco últimos bytes del nonce los ocupan el contador de
	// segmento y la marca de final, así que solo hay 19 bytes aleatorios. Con
	// XChaCha20 sobran: es justo el motivo de tener un nonce de 24.
	if modo == ModoFlujo {
		for i := 19; i < tamNonce; i++ {
			c.Nonce[i] = 0
		}
	}
	return c, nil
}

// Sellar cifra datos con la clave y devuelve el contenedor binario completo.
func Sellar(datos, clave []byte, p Parametros) ([]byte, error) {
	cab, err := nuevaCabecera(ModoUnico, p)
	if err != nil {
		return nil, err
	}

	k := derivar(clave, cab.Sal[:], p)
	defer Borrar(k)

	aead, err := chacha20poly1305.NewX(k)
	if err != nil {
		return nil, err
	}

	cb := cab.bytes()
	return aead.Seal(cb, cab.Nonce[:], datos, cb), nil
}

// Abrir descifra un contenedor binario. Devuelve ErrClaveIncorrecta cuando la
// etiqueta no cuadra, que es tanto el caso de la clave equivocada como el de la
// manipulación del contenido: criptográficamente son indistinguibles, y presentar
// dos mensajes distintos sería mentir.
func Abrir(contenedor, clave []byte) ([]byte, error) {
	cab, err := leerCabecera(contenedor)
	if err != nil {
		return nil, err
	}
	if cab.Modo != ModoUnico {
		return nil, errors.New("Este contenedor es un flujo por segmentos: usa el modo fichero")
	}

	k := derivar(clave, cab.Sal[:], cab.Par)
	defer Borrar(k)

	aead, err := chacha20poly1305.NewX(k)
	if err != nil {
		return nil, err
	}

	cb := contenedor[:tamCabecera]
	claro, err := aead.Open(nil, cab.Nonce[:], contenedor[tamCabecera:], cb)
	if err != nil {
		return nil, ErrClaveIncorrecta
	}
	return claro, nil
}

// SellarTexto cifra y devuelve la representación de una línea, ESF1.<base64url>.
func SellarTexto(datos, clave []byte, p Parametros) (string, error) {
	b, err := Sellar(datos, clave, p)
	if err != nil {
		return "", err
	}
	return Prefijo + codificacion.EncodeToString(b), nil
}

// AbrirTexto descifra la representación de una línea. Tolera espacios y saltos
// alrededor, y también el texto sin el prefijo: pegar desde un correo o un chat
// arrastra basura invisible con demasiada facilidad como para ser estricto aquí.
func AbrirTexto(texto string, clave []byte) ([]byte, error) {
	limpio := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, texto)
	limpio = strings.TrimPrefix(limpio, Prefijo)

	bruto, err := codificacion.DecodeString(limpio)
	if err != nil {
		// base64 estándar por si el texto viene de otra herramienta o de un copiado
		// que sustituyó los caracteres.
		bruto, err = base64.StdEncoding.DecodeString(limpio)
		if err != nil {
			return nil, ErrFormato
		}
	}
	return Abrir(bruto, clave)
}
