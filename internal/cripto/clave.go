// Package cripto implementa el contenedor ESF1: derivación de clave con Argon2id
// y cifrado autenticado con XChaCha20-Poly1305.
package cripto

import (
	"crypto/rand"
	"errors"
	"math"
	"unicode"

	"golang.org/x/crypto/argon2"
)

// Parametros son los ajustes de Argon2id con los que se derivó una clave. Viajan
// dentro de la cabecera del contenedor, así que subir el coste en una versión
// futura no rompe los mensajes ya cifrados: cada uno lleva los suyos.
type Parametros struct {
	Memoria     uint32 // en KiB
	Pasadas     uint32
	Paralelismo uint8
}

// PerfilInteractivo es el coste por defecto: 64 MiB, 3 pasadas, 4 hilos. Ronda
// el medio segundo en una máquina de hoy, que castiga la fuerza bruta sin que la
// interfaz parezca colgada.
var PerfilInteractivo = Parametros{Memoria: 64 * 1024, Pasadas: 3, Paralelismo: 4}

const (
	tamSal   = 16
	tamClave = 32
)

// límites de cordura al abrir un contenedor ajeno: sin ellos, una cabecera
// manipulada pidiendo 64 GiB de memoria tumba el proceso antes de que nadie
// pueda comprobar la etiqueta de autenticación.
const (
	memoriaMaxima = 1024 * 1024 // 1 GiB en KiB
	pasadasMaximo = 16
)

func (p Parametros) validar() error {
	switch {
	case p.Memoria < 8*1024 || p.Memoria > memoriaMaxima:
		return ErrFormato
	case p.Pasadas < 1 || p.Pasadas > pasadasMaximo:
		return ErrFormato
	case p.Paralelismo < 1:
		return ErrFormato
	}
	return nil
}

// derivar convierte la clave humana en 32 bytes de clave simétrica.
func derivar(clave, sal []byte, p Parametros) []byte {
	return argon2.IDKey(clave, sal, p.Pasadas, p.Memoria, p.Paralelismo, tamClave)
}

// Borrar sobrescribe un búfer de material sensible. No es una garantía —el
// recolector de basura de Go puede haber copiado el búfer antes— pero acorta la
// ventana en la que la clave derivada vive en memoria.
func Borrar(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func azar(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, errors.New("No hay entropía disponible en el sistema: " + err.Error())
	}
	return b, nil
}

// Fuerza es la valoración de una clave tecleada por una persona.
type Fuerza struct {
	Bits       float64 // entropía estimada, en bits
	Nivel      int     // 0 muy débil · 1 débil · 2 aceptable · 3 buena · 4 excelente
	Etiqueta   string
	Sugerencia string
}

// Evaluar estima la entropía de una clave por el tamaño del alfabeto que usa.
// Es una estimación optimista —no sabe nada de palabras de diccionario ni de
// patrones de teclado— y por eso la interfaz la presenta como orientación y no
// como permiso.
func Evaluar(clave string) Fuerza {
	if clave == "" {
		return Fuerza{Nivel: 0, Etiqueta: "Vacía", Sugerencia: "Hace falta una clave"}
	}

	var minus, mayus, digitos, simbolos, noASCII bool
	for _, r := range clave {
		switch {
		case r > unicode.MaxASCII:
			noASCII = true
		case unicode.IsLower(r):
			minus = true
		case unicode.IsUpper(r):
			mayus = true
		case unicode.IsDigit(r):
			digitos = true
		default:
			simbolos = true
		}
	}

	alfabeto := 0
	for _, c := range []struct {
		usado bool
		tam   int
	}{{minus, 26}, {mayus, 26}, {digitos, 10}, {simbolos, 33}, {noASCII, 100}} {
		if c.usado {
			alfabeto += c.tam
		}
	}

	// log2(alfabeto) * longitud, contando runas y no bytes: una clave con acentos
	// no es más fuerte por ocupar más bytes en UTF-8.
	longitud := len([]rune(clave))
	bits := math.Log2(float64(alfabeto)) * float64(longitud)

	f := Fuerza{Bits: bits}
	switch {
	case bits < 40:
		f.Nivel, f.Etiqueta = 0, "Muy débil"
		f.Sugerencia = "Se rompe en minutos: alarga la clave"
	case bits < 60:
		f.Nivel, f.Etiqueta = 1, "Débil"
		f.Sugerencia = "Vale para un secreto de usar y tirar, no para uno que dure"
	case bits < 80:
		f.Nivel, f.Etiqueta = 2, "Aceptable"
		f.Sugerencia = "Razonable; cuatro palabras sin relación serían mejor"
	case bits < 100:
		f.Nivel, f.Etiqueta = 3, "Buena"
	default:
		f.Nivel, f.Etiqueta = 4, "Excelente"
	}
	return f
}
