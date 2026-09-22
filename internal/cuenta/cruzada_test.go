package cuenta

import (
	"encoding/hex"
	"testing"

	"github.com/webcafeina/esfinge/internal/cruzada"
)

// La clave de acceso de la extensión, **contra la de éste** (ADR 0040): si no
// coinciden, quien entre desde el navegador no entra, o entra en otra cuenta.
func TestCruzadaClaveDeAcceso(t *testing.T) {
	cruzada.Activa(t)
	for _, c := range []struct {
		maestra string
		sal     string
		p       Parametros
	}{
		{"una maestra larga para probar", "00112233445566778899aabbccddeeff", PorDefecto},
		{"contraseña ñandú 😀", "ffeeddccbbaa99887766554433221100", Parametros{Memoria: 128 * 1024, Pasadas: 4, Paralelismo: 2}},
	} {
		sal, _ := hex.DecodeString(c.sal)
		quiero, err := DerivarAcceso(c.maestra, sal, c.p)
		if err != nil {
			t.Fatal(err)
		}
		var suya string
		cruzada.Pedir(t, map[string]any{"orden": "acceso", "maestra": c.maestra, "sal": c.sal, "parametros": c.p}, &suya)
		if suya != hex.EncodeToString(quiero) {
			t.Errorf("%q: Go da %x y la extensión %s", c.maestra, quiero, suya)
		}
	}
}

// El correo, igual en los dos: con uno distinto, el mismo correo serían dos
// cuentas. Los casos son los que la gente escribe de verdad.
func TestCruzadaCorreo(t *testing.T) {
	correos := []string{" Hola@Ejemplo.COM ", "nacho+esfinge@webcafeina.com", "ÁLVARO@correo.es", "álvaro@correo.es", "mal", "sin@punto", "dos@@x.com"}
	var suyos []string
	cruzada.Pedir(t, map[string]any{"orden": "correos", "correos": correos}, &suyos)
	for i, c := range correos {
		quiero, err := NormalizarCorreo(c)
		if err != nil {
			quiero = "error: " + err.Error()
		}
		if suyos[i] != quiero {
			t.Errorf("%q: Go dice %q y la extensión %q", c, quiero, suyos[i])
		}
	}
}
