package navegador

import (
	"encoding/binary"
	"sort"
)

// El CBOR que hace falta para crear una llave de acceso, **y solo ése** (ADR 0048).
//
// WebAuthn usa CBOR en dos sitios al crear: el `attestationObject` que se le
// devuelve al sitio y la clave pública en formato COSE que va dentro. No hace
// falta nada más, así que **aquí solo hay codificador**: no se descodifica CBOR en
// ningún sitio, y un descodificador es superficie de ataque sobre bytes que vienen
// de fuera.
//
// Está espejado en `navegador/src/nucleo/cbor.ts` por la razón de siempre: con
// cuenta la extensión crea la llave sola, y **el sitio guarda estos bytes**. Si los
// dos lados no escriben lo mismo, una llave creada con cuenta y otra sin ella
// serían distintas para el mismo sitio.

// Los tipos mayores de CBOR que se usan, en su sitio del primer byte.
const (
	cborEntero   = 0 << 5 // 0..2^64-1
	cborNegativo = 1 << 5 // -1..-2^64
	cborBytes    = 2 << 5
	cborTexto    = 3 << 5
	cborMapa     = 5 << 5
)

// cabecera escribe el tipo mayor y el valor, con el tamaño más corto que quepa.
//
// **El más corto no es una optimización: es la forma canónica.** CTAP2 exige
// codificación determinista, y un sitio que compruebe la atestación puede volver a
// codificar lo que reciba y comparar.
func cabecera(mayor byte, valor uint64) []byte {
	switch {
	case valor < 24:
		return []byte{mayor | byte(valor)}
	case valor <= 0xff:
		return []byte{mayor | 24, byte(valor)}
	case valor <= 0xffff:
		b := []byte{mayor | 25, 0, 0}
		binary.BigEndian.PutUint16(b[1:], uint16(valor))
		return b
	case valor <= 0xffffffff:
		b := []byte{mayor | 26, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(b[1:], uint32(valor))
		return b
	default:
		b := []byte{mayor | 27, 0, 0, 0, 0, 0, 0, 0, 0}
		binary.BigEndian.PutUint64(b[1:], valor)
		return b
	}
}

func cborDeEntero(n int64) []byte {
	if n < 0 {
		return cabecera(cborNegativo, uint64(-1-n))
	}
	return cabecera(cborEntero, uint64(n))
}

func cborDeBytes(b []byte) []byte {
	return append(cabecera(cborBytes, uint64(len(b))), b...)
}

func cborDeTexto(s string) []byte {
	return append(cabecera(cborTexto, uint64(len(s))), s...)
}

// cborDeMapa escribe un mapa **en el orden canónico de CTAP2**: las claves ya
// codificadas, ordenadas primero por largo y luego por bytes.
//
// El orden importa y no es el que uno escribiría: con las claves del COSE —1, 3,
// -1, -2, -3— sale `1, 3, -1, -2, -3`, porque los negativos se codifican como
// `0x20`, `0x21` y `0x22`, que son mayores que `0x01` y `0x03`. Escribirlas en el
// orden en que se leen daría otros bytes.
func cborDeMapa(pares [][2][]byte) []byte {
	orden := make([][2][]byte, len(pares))
	copy(orden, pares)
	sort.SliceStable(orden, func(i, j int) bool {
		a, b := orden[i][0], orden[j][0]
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return string(a) < string(b)
	})
	out := cabecera(cborMapa, uint64(len(orden)))
	for _, p := range orden {
		out = append(out, p[0]...)
		out = append(out, p[1]...)
	}
	return out
}

// ClaveCOSE escribe una pública P-256 en el formato que WebAuthn guarda.
//
// Es un mapa CBOR con cinco entradas, y las claves son números: `1` el tipo de
// clave —`2`, curva elíptica—, `3` el algoritmo —`-7`, ECDSA con SHA-256—, `-1` la
// curva —`1`, P-256— y `-2` y `-3` las dos coordenadas, **de 32 bytes cada una y
// con los ceros de delante puestos**.
//
// Eso último es lo contrario de lo que pide el DER de una firma, donde los ceros de
// delante se quitan. Aquí son de largo fijo: una `x` que empiece por cero **tiene
// que llevarlo**, o el sitio leerá otra clave y la llave no servirá para entrar
// nunca más. Pasa una vez de cada 256.
func ClaveCOSE(x, y []byte) []byte {
	return cborDeMapa([][2][]byte{
		{cborDeEntero(1), cborDeEntero(2)},
		{cborDeEntero(3), cborDeEntero(-7)},
		{cborDeEntero(-1), cborDeEntero(1)},
		{cborDeEntero(-2), cborDeBytes(deTreintaYDos(x))},
		{cborDeEntero(-3), cborDeBytes(deTreintaYDos(y))},
	})
}

// deTreintaYDos deja una coordenada en sus 32 bytes exactos.
func deTreintaYDos(b []byte) []byte {
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// ObjetoDeAtestacion envuelve los datos del autenticador **sin atestación**.
//
// `fmt: "none"` y `attStmt` vacío, que es lo que pide la ADR 0048 y lo que hacen
// todos los gestores: Esfinge no es un autenticador certificado y no va a decir que
// lo es. Un sitio que exija atestación de verdad rechazará la llave, y hace bien.
//
// **El orden de las tres claves es el de la especificación** —`fmt`, `attStmt`,
// `authData`—, que además coincide con el canónico porque los tres nombres miden
// distinto o van en orden.
func ObjetoDeAtestacion(datosDelAutenticador []byte) []byte {
	return cborDeMapa([][2][]byte{
		{cborDeTexto("fmt"), cborDeTexto("none")},
		{cborDeTexto("attStmt"), cborDeMapa(nil)},
		{cborDeTexto("authData"), cborDeBytes(datosDelAutenticador)},
	})
}
