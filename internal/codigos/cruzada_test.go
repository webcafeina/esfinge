package codigos

import (
	"encoding/base32"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/cruzada"
)

// El código de la extensión, **contra el de éste** (ADR 0040): en su Mac se
// comprobó que el de Go da lo mismo que Dashlane, así que el de la extensión
// tiene que dar lo mismo que el de Go. Semillas al azar, los tres algoritmos, de
// seis a diez cifras —con el desbordamiento de diez incluido— y semillas mal
// copiadas, que tienen que fallar en los dos lados.
func TestCruzadaCodigos(t *testing.T) {
	cruzada.Activa(t)
	azar := rand.New(rand.NewPCG(20260922, 1))
	type caso struct {
		Semilla string `json:"semilla"`
		Unix    int64  `json:"unix"`
	}
	var casos []caso
	for i := 0; i < 200; i++ {
		clave := make([]byte, 10+azar.IntN(30))
		for j := range clave {
			clave[j] = byte(azar.IntN(256))
		}
		b32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(clave)
		semilla := b32
		switch i % 4 {
		case 1:
			semilla = fmt.Sprintf("otpauth://totp/Sitio:yo%%40x.com?secret=%s&digits=%d&algorithm=%s&period=%d",
				b32, 6+azar.IntN(5), []string{"SHA1", "sha256", "SHA512"}[azar.IntN(3)], []int{30, 60, 15}[azar.IntN(3)])
		case 2:
			semilla = b32[:len(b32)-1] // le falta un carácter: a veces pasa, a veces no, igual en los dos
		case 3:
			semilla = "  " + string([]rune(b32)[:4]) + "-" + b32[4:] + "\n"
		}
		casos = append(casos, caso{semilla, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix() + azar.Int64N(1e9)})
	}
	casos = append(casos, caso{"otpauth://hotp/x?secret=" + semillaDelRFC, 0}, caso{"0189", 0})

	var suyos []string
	cruzada.Pedir(t, map[string]any{"orden": "codigos", "casos": casos}, &suyos)
	for i, c := range casos {
		quiero := ""
		s, err := Leer(c.Semilla)
		if err == nil {
			quiero, err = s.En(time.Unix(c.Unix, 0))
		}
		if err != nil {
			quiero = "error: " + err.Error()
		}
		if suyos[i] != quiero {
			t.Errorf("%q en %d: Go dice %q y la extensión %q", c.Semilla, c.Unix, quiero, suyos[i])
		}
	}
}
