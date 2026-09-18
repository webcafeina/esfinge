// Package cuenta habla con el servidor de cuentas de Esfinge (ADR 0035 y 0036).
//
// **El servidor no ve nunca la contraseña maestra ni la clave de la bóveda.** De
// la maestra se deriva aquí una clave de acceso —con su propia sal, distinta de
// la de la ranura de la bóveda— y eso es lo único que viaja; el servidor guarda
// un HMAC de ella. La bóveda sube y baja cifrada, tal cual está en el disco.
//
// No sabe nada de Wails ni de la ventana: la usa la aplicación y se prueba sola.
package cuenta

import (
	"crypto/hkdf"
	"crypto/sha256"
	"errors"
	"regexp"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// Parametros son el coste de Argon2id con el que se deriva la clave de acceso.
type Parametros struct {
	Memoria     uint32 `json:"memoria"` // KiB
	Pasadas     uint32 `json:"pasadas"`
	Paralelismo uint8  `json:"paralelismo"`
}

// PorDefecto es el coste de siempre, el mismo que protege la ranura de la
// contraseña maestra: `cripto.PerfilInteractivo`.
var PorDefecto = Parametros{
	Memoria:     cripto.PerfilInteractivo.Memoria,
	Pasadas:     cripto.PerfilInteractivo.Pasadas,
	Paralelismo: cripto.PerfilInteractivo.Paralelismo,
}

// ErrCosteBajo: el servidor pide derivar con menos coste del que usa Esfinge.
//
// **Es la trampa de dejar que el servidor diga los parámetros.** Uno que pidiera
// una pasada y 8 MiB haría que la clave de acceso —que sí le llega— se pudiera
// atacar sin conexión mucho más deprisa que la ranura de la bóveda. Por debajo de
// lo de siempre no se deriva, lo diga quien lo diga.
var ErrCosteBajo = errors.New("El servidor pide proteger la contraseña con menos coste del normal; no se sigue")

// Validar comprueba que un coste está entre lo de siempre y el tope.
func (p Parametros) Validar() error {
	if p.Memoria < PorDefecto.Memoria || p.Pasadas < PorDefecto.Pasadas || p.Paralelismo < 1 {
		return ErrCosteBajo
	}
	// Los mismos topes que al abrir un contenedor: un servidor que pidiera 64 GiB
	// tumbaría el proceso antes de que nadie pudiera decir nada.
	if p.Memoria > 1024*1024 || p.Pasadas > 16 || p.Paralelismo > 16 {
		return errors.New("El servidor pide un coste de derivación imposible")
	}
	return nil
}

// DerivarAcceso convierte la contraseña maestra en la clave de acceso a la cuenta.
//
//	raíz  = Argon2id(maestra, sal de la cuenta, coste)
//	clave = HKDF-SHA256(raíz, «esfinge/cuenta/acceso/v1»)
//
// El HKDF separa esta clave de cualquier otra que salga algún día de la misma
// raíz. Y la sal es la de la cuenta, no la de la ranura de la bóveda: aunque la
// contraseña sea la misma, las dos derivaciones no tienen nada que ver.
func DerivarAcceso(maestra string, sal []byte, p Parametros) ([]byte, error) {
	if err := p.Validar(); err != nil {
		return nil, err
	}
	if len(sal) != 16 {
		return nil, errors.New("La sal de la cuenta no mide lo que debe")
	}
	if strings.TrimSpace(maestra) == "" {
		return nil, errors.New("La contraseña maestra no puede estar vacía")
	}
	raiz := argon2.IDKey([]byte(maestra), sal, p.Pasadas, p.Memoria, p.Paralelismo, 32)
	defer cripto.Borrar(raiz)
	return hkdf.Key(sha256.New, raiz, nil, "esfinge/cuenta/acceso/v1", 32)
}

var pareceCorreo = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// NormalizarCorreo deja el correo como lo guarda el servidor: sin espacios
// alrededor, en NFC y en minúsculas. **Tiene que hacer exactamente lo mismo que
// `normalizarCorreo` en servidor/src/protocolo.ts**: si no, el mismo correo sería
// dos cuentas. Nada de quitar puntos ni lo que va tras un «+»: eso es una regla
// de Gmail, no del correo.
func NormalizarCorreo(c string) (string, error) {
	n := strings.ToLower(norm.NFC.String(strings.TrimSpace(c)))
	if len(n) < 3 || len(n) > 254 || !pareceCorreo.MatchString(n) {
		return "", errors.New("Ese correo no parece válido")
	}
	return n, nil
}
