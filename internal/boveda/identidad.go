package boveda

// La identidad de una bóveda para compartir copias (ADR 0043).
//
// **Una semilla de 32 bytes dentro del cuerpo cifrado, y todo lo demás se deriva
// de ella.** No se guarda ninguna llave: derivarlas cuesta microsegundos y una
// llave guardada es una llave que se puede desincronizar de su semilla.
//
// De la semilla salen dos cosas, con etiquetas distintas para que no puedan
// confundirse nunca:
//
//   - **X25519**, para que otros cifren hacia ti (HPKE, RFC 9180).
//   - **Ed25519**, para firmar lo que mandas.
//
// **Se crea una sola vez y no cambia**: cambiarla es cambiar de identidad, y lo
// que te mandaron antes se queda sin abrir.

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// La semilla se guarda en base64url sin relleno, como la clave de bóveda.
var b64 = base64.RawURLEncoding

// Suite es el conjunto de HPKE con el que se cifra hacia esta identidad.
//
// Viaja como campo —y no como una constante del código— para que el día que haya
// ML-KEM razonable en el navegador se pueda añadir sin romper lo ya mandado
// (ADR 0043). Hoy solo hay una, y es la que el navegador puede hacer con una
// biblioteca pequeña.
const Suite = "DHKEM(X25519)/HKDF-SHA256/ChaCha20-Poly1305"

const (
	infoCifrado = "esfinge/identidad/cifrado/v1"
	infoFirma   = "esfinge/identidad/firma/v1"
)

// identidad es lo que se guarda: la semilla y cuándo nació.
//
// `Creada` no es adorno: es lo que desempata si dos equipos crearan una identidad
// cada uno antes de verse (ADR 0043).
type identidad struct {
	Semilla string `json:"semilla"` // 32 bytes en base64url
	Creada  string `json:"creada"`
	Suite   string `json:"suite"`
}

// Identidad es lo que se puede enseñar de una identidad: sus llaves públicas y su
// huella. **La semilla no sale de aquí.**
type Identidad struct {
	Suite   string `json:"suite"`
	Cifrado []byte `json:"cifrado"` // X25519, 32 bytes
	Firma   []byte `json:"firma"`   // Ed25519, 32 bytes
	Huella  string `json:"huella"`
}

var errSinIdentidad = errors.New("Esta bóveda todavía no tiene identidad")

// derivar saca de la semilla las dos llaves privadas.
func derivarIdentidad(semilla []byte) (*ecdh.PrivateKey, ed25519.PrivateKey, error) {
	deCifrado, err := hkdf.Key(sha256.New, semilla, nil, infoCifrado, 32)
	if err != nil {
		return nil, nil, err
	}
	deFirma, err := hkdf.Key(sha256.New, semilla, nil, infoFirma, 32)
	if err != nil {
		return nil, nil, err
	}
	cifrado, err := ecdh.X25519().NewPrivateKey(deCifrado)
	if err != nil {
		return nil, nil, err
	}
	return cifrado, ed25519.NewKeyFromSeed(deFirma), nil
}

// HuellaDeIdentidad es lo que se compara por teléfono: siete grupos de cuatro
// símbolos del alfabeto de la clave de recuperación, que está pensado para leerse
// en voz alta sin confundir la I con el 1 ni la O con el 0.
//
// Cubre **la suite y las dos llaves**: cambiar cualquiera de las tres cambia la
// huella, que es justo lo que hace falta para que comparar sirva de algo.
func HuellaDeIdentidad(suite string, cifrado, firma []byte) string {
	h := sha256.New()
	h.Write([]byte(suite))
	h.Write([]byte{0})
	h.Write(cifrado)
	h.Write([]byte{0})
	h.Write(firma)
	return agrupar(aPalabras(h.Sum(nil))[:28], 4)
}

// aPalabras es la misma conversión que la clave de recuperación: cinco bits por
// símbolo, sin I, L, O ni U.
func aPalabras(b []byte) string {
	var sb []byte
	var acumulado, bits uint32
	for _, x := range b {
		acumulado = acumulado<<8 | uint32(x)
		bits += 8
		for bits >= 5 {
			bits -= 5
			sb = append(sb, alfabeto[(acumulado>>bits)&31])
		}
	}
	return string(sb)
}

func agrupar(s string, cada int) string {
	var sb []byte
	for i, r := range s {
		if i > 0 && i%cada == 0 {
			sb = append(sb, '-')
		}
		sb = append(sb, byte(r))
	}
	return string(sb)
}

// Identidad devuelve la identidad pública de esta bóveda, creándola si todavía no
// la tiene. **Guarda si la crea**, porque una identidad que no llega al disco es
// una identidad distinta en el próximo arranque.
func (b *Boveda) Identidad() (Identidad, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return Identidad{}, ErrCerrada
	}
	if b.cont.Identidad == nil {
		semilla, err := cripto.Azar(32)
		if err != nil {
			return Identidad{}, err
		}
		b.cont.Identidad = &identidad{
			Semilla: b64.EncodeToString(semilla),
			Creada:  ahora().UTC().Format(time.RFC3339),
			Suite:   Suite,
		}
		b.cuerpoSucio = true
		if err := b.guardar(); err != nil {
			b.cont.Identidad = nil
			return Identidad{}, err
		}
	}
	return publicaDe(b.cont.Identidad)
}

func publicaDe(i *identidad) (Identidad, error) {
	if i == nil {
		return Identidad{}, errSinIdentidad
	}
	semilla, err := b64.DecodeString(i.Semilla)
	if err != nil || len(semilla) != 32 {
		return Identidad{}, errors.New("La identidad de la bóveda no se entiende")
	}
	cifrado, firma, err := derivarIdentidad(semilla)
	if err != nil {
		return Identidad{}, err
	}
	pubCifrado := cifrado.PublicKey().Bytes()
	pubFirma := []byte(firma.Public().(ed25519.PublicKey))
	suite := i.Suite
	if suite == "" {
		suite = Suite
	}
	return Identidad{
		Suite:   suite,
		Cifrado: pubCifrado,
		Firma:   pubFirma,
		Huella:  HuellaDeIdentidad(suite, pubCifrado, pubFirma),
	}, nil
}

// fundirIdentidad elige una de las dos, y **siempre la misma la mire quien la
// mire**: la más antigua, y si empatan la de semilla menor. Solo puede haber dos
// si dos equipos crearon la suya antes de verse.
func fundirIdentidad(l, r *identidad) *identidad {
	switch {
	case l == nil:
		return r
	case r == nil:
		return l
	case l.Creada != r.Creada:
		if l.Creada < r.Creada {
			return l
		}
		return r
	case l.Semilla <= r.Semilla:
		return l
	default:
		return r
	}
}

// comoJSON es lo que se escribe en el cuerpo. Se usa desde las pruebas cruzadas.
func (i *identidad) comoJSON() json.RawMessage {
	b, _ := json.Marshal(i)
	return b
}
