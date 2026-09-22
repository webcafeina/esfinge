package navegador

import (
	"testing"

	"github.com/webcafeina/esfinge/internal/cruzada"
)

// Qué cuentas son de qué sitio, **igual en Go y en la extensión** (ADR 0040). Con
// cuenta, la extensión lo decide sola, y decidirlo distinto es entregar una
// contraseña al sitio equivocado o no rellenar donde se rellenaba. Los casos son
// los de la tabla de `dominios_test.go` y los que rompen un analizador de URL.
func TestCruzadaDominios(t *testing.T) {
	casos := []string{
		"https://www.banco.es/particulares", "https://banco.es.malo.com/", "https://foo.github.io/x",
		"https://bar.github.io", "https://mail.google.com", "https://accounts.google.com/signin",
		"http://banco.es", "https://127.0.0.1/", "https://[::1]:8443/", "https://localhost/",
		"https://github.io", "https://com", "https://a.b.com.es", "https://s3.amazonaws.com/cubo",
		"https://bücher.de", "https://xn--bcher-kva.de", "https://BANCO.ES.", "banco.es", "www.banco.es/entrar",
		"  banco.es  ", "https://usuario:clave@banco.es", "ftp://banco.es", "javascript:alert(1)", "",
		"https://banco.es:8443/x?y#z", "sub.dominio.co.uk", "https://blogspot.com", "https://yo.blogspot.com",
		"https://agenciatributaria.gob.es", "https://sede.agenciatributaria.gob.es/Sede/",
	}
	type respuesta struct {
		Origen string `json:"origen"`
		Sitio  string `json:"sitio"`
	}
	var suyas []respuesta
	cruzada.Pedir(t, map[string]any{"orden": "dominios", "casos": casos}, &suyas)
	for i, c := range casos {
		origen, err := DominioDeOrigen(c)
		if err != nil {
			origen = "error"
		}
		quiero := respuesta{origen, DominioDeSitio(c)}
		if suyas[i] != quiero {
			t.Errorf("%q: Go dice %+v y la extensión %+v", c, quiero, suyas[i])
		}
	}
}
