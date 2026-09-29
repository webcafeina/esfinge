package navegador

import "testing"

// **Qué contesta, no solo que los dos lados contesten igual.**
//
// La prueba cruzada compara Go con la extensión, y dos lados pueden coincidir en
// algo que está mal. Ésta dice cuál es la respuesta buena, caso a caso, sacada de
// la regla de WebAuthn: el `rpId` tiene que ser **el anfitrión o un sufijo suyo
// separado por punto, y no un sufijo público**.
func TestRPIDPermitido(t *testing.T) {
	casos := []struct {
		rpID, origen, quiero, porque string
	}{
		{"", "https://github.com/login", "github.com",
			"sin rpId manda el anfitrión"},
		{"", "https://login.ejemplo.com/", "login.ejemplo.com",
			"y manda el anfitrión ENTERO, no su dominio registrable"},
		{"github.com", "https://github.com/login", "github.com",
			"el mismo anfitrión"},
		{"ejemplo.com", "https://login.ejemplo.com/", "ejemplo.com",
			"un sitio con subdominios puede subir al suyo"},
		{"b.ejemplo.com", "https://a.b.ejemplo.com/", "b.ejemplo.com",
			"y a cualquier escalón intermedio"},

		{"otro.com", "https://ejemplo.com/", "",
			"un sitio no firma por otro"},
		{"ejemplo.com", "https://ejemplo.com.malo.com/", "",
			"y el sufijo tiene que ir al final, no en medio"},
		// **El caso que de verdad ataca**, y que faltaba: sin exigir el punto que
		// separa, `malaejemplo.com` termina en `ejemplo.com` y firmaría por él. Lo
		// encontró una mutación —quitar el punto de la comprobación dejaba la tabla
		// en verde—, no la lectura.
		{"ejemplo.com", "https://malaejemplo.com/", "",
			"terminar en el nombre de otro no es ser subdominio suyo"},
		{"ejemplo.com", "https://xn--maloejemplo.com/", "",
			"lo mismo con cualquier cosa pegada delante"},
		{"com", "https://ejemplo.com/", "",
			"nadie firma por un sufijo público"},
		{"github.io", "https://foo.github.io/", "",
			"ni por uno privado: si no, una página de github.io firma por todas"},
		{"co.uk", "https://algo.co.uk/", "", "lo mismo con dos etiquetas"},
		{"gob.es", "https://sede.agenciatributaria.gob.es/", "", "y con el de aquí"},
		{"agenciatributaria.gob.es", "https://sede.agenciatributaria.gob.es/",
			"agenciatributaria.gob.es", "pero el dominio de verdad sí"},

		{"", "http://ejemplo.com/", "", "sin https no se firma"},
		{"", "https://127.0.0.1/", "", "una IP no tiene dominio"},
		{"", "https://localhost/", "", "ni localhost"},
		{"", "https://bücher.de/", "", "ni lo que no es ASCII, como en el resto"},

		{"EJEMPLO.COM", "https://login.ejemplo.com/", "ejemplo.com",
			"las mayúsculas no cuentan, y lo que sale es la forma que se hashea"},
		{"ejemplo.com.", "https://login.ejemplo.com/", "ejemplo.com",
			"ni el punto final"},
		{"", "https://login.ejemplo.com:8443/x?y#z", "login.ejemplo.com",
			"el puerto y lo demás no son del anfitrión"},
		{"", "", "", "y una dirección vacía no es un origen"},
	}
	for _, c := range casos {
		if tengo := RPIDPermitido(c.rpID, c.origen); tengo != c.quiero {
			t.Errorf("rpId %q en %q: tengo %q y quiero %q — %s", c.rpID, c.origen, tengo, c.quiero, c.porque)
		}
	}
}
