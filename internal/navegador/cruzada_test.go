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

// **Con qué `rpId` se puede firmar una llave de acceso, igual en los dos lados**
// (ADR 0048).
//
// Es la tabla que más importa de todas las cruzadas, porque esto no decide si se
// rellena: decide **para quién se firma**. Y es donde se caza que la lista de
// sufijos públicos de `golang.org/x/net` y la de `tldts` hayan dejado de
// coincidir, que es un riesgo ya escrito en `dominios.ts` y que aquí deja de ser
// una molestia para ser un agujero.
func TestCruzadaRPID(t *testing.T) {
	casos := []struct {
		RPID   string `json:"rpId"`
		Origen string `json:"origen"`
	}{
		// Lo normal: sin `rpId`, manda el anfitrión **entero**.
		{"", "https://github.com/login"},
		{"", "https://login.ejemplo.com/"},
		{"github.com", "https://github.com/login"},
		// Un sufijo del anfitrión, que es lo que hace un sitio con subdominios.
		{"ejemplo.com", "https://login.ejemplo.com/"},
		{"ejemplo.com", "https://a.b.ejemplo.com/"},
		{"b.ejemplo.com", "https://a.b.ejemplo.com/"},
		// **Lo que no puede pasar**, y es para lo que está la función.
		{"otro.com", "https://ejemplo.com/"},
		{"ejemplo.com", "https://ejemplo.com.malo.com/"},
		{"ejemplo.com", "https://malaejemplo.com/"},
		{"ejemplo.com", "https://xn--maloejemplo.com/"},
		{"malo.com", "https://ejemplo.com/"},
		{"com", "https://ejemplo.com/"},
		// Un sufijo público **privado**: si esto pasara, cualquier página alojada en
		// `github.io` firmaría por todas las demás.
		{"github.io", "https://foo.github.io/"},
		{"blogspot.com", "https://yo.blogspot.com/"},
		{"co.uk", "https://algo.co.uk/"},
		{"gob.es", "https://sede.agenciatributaria.gob.es/"},
		{"agenciatributaria.gob.es", "https://sede.agenciatributaria.gob.es/"},
		// Sin contexto seguro no se firma, y una IP no tiene dominio.
		{"", "http://ejemplo.com/"},
		{"", "https://127.0.0.1/"},
		{"", "https://[::1]:8443/"},
		{"", "https://localhost/"},
		// Formas raras: el punto final, las mayúsculas, el puerto, lo que no es ASCII.
		{"EJEMPLO.COM", "https://login.ejemplo.com/"},
		{"ejemplo.com.", "https://login.ejemplo.com/"},
		{"", "https://login.ejemplo.com:8443/x?y#z"},
		{"", "https://bücher.de/"},
		{"bücher.de", "https://bücher.de/"},
		{"", ""},
		{"", "javascript:alert(1)"},
	}
	var suyas []string
	cruzada.Pedir(t, map[string]any{"orden": "rpid", "casos": casos}, &suyas)
	for i, c := range casos {
		quiero := RPIDPermitido(c.RPID, c.Origen)
		if suyas[i] != quiero {
			t.Errorf("rpId %q en %q: Go dice %q y la extensión %q", c.RPID, c.Origen, quiero, suyas[i])
		}
	}
}
