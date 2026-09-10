package navegador

import "testing"

// **La prueba que decide si esto se puede publicar.** Emparejar mal una entrada
// con una pestaña es entregar una contraseña al sitio equivocado, que es la
// única forma en que un gestor de contraseñas hace daño de verdad.
func TestQueEntradaCorrespondeAQuePestana(t *testing.T) {
	casos := []struct {
		nombre  string
		sitio   string
		pestana string
		encaja  bool
	}{
		// Lo que tiene que funcionar, porque es lo que hay guardado de verdad.
		{"el caso normal", "https://banco.es", "https://banco.es", true},
		{"con www delante", "https://banco.es", "https://www.banco.es", true},
		{"con www guardado", "https://www.banco.es", "https://banco.es", true},
		{"con ruta guardada", "https://banco.es/particulares/entrar", "https://banco.es", true},
		{"sin esquema guardado", "banco.es", "https://www.banco.es/entrar", true},
		{"con espacios, como sale de un CSV", "  https://banco.es  ", "https://banco.es", true},
		{"en mayúsculas", "HTTPS://BANCO.ES", "https://banco.es", true},
		{"con puerto en la pestaña", "banco.es", "https://banco.es:8443/x", true},
		{"con punto final", "banco.es.", "https://banco.es", true},
		// **La decisión**: dos subdominios del mismo dominio registrable son el
		// mismo sitio. Es lo que hace que una cuenta guardada de Gmail sirva para
		// entrar por la página de cuentas de Google.
		{"otro subdominio del mismo dominio", "https://mail.google.com", "https://accounts.google.com", true},

		// Y lo que **no** puede pasar nunca.
		{
			"un dominio que acaba pareciéndose",
			"https://banco.es", "https://banco.es.malo.com", false,
		},
		{
			"al revés: el guardado dentro del malo",
			"https://banco.es.malo.com", "https://banco.es", false,
		},
		{
			"un prefijo que se le parece",
			"https://banco.es", "https://mibanco.es", false,
		},
		{
			// La razón entera por la que se usa la lista de sufijos públicos:
			// cortando por el último punto, éstos serían el mismo sitio.
			"dos páginas de GitHub distintas",
			"https://foo.github.io", "https://bar.github.io", false,
		},
		{"otro dominio y ya", "https://banco.es", "https://otracosa.com", false},
		{"nada guardado", "", "https://banco.es", false},
		{"basura guardada", "no es una dirección", "https://banco.es", false},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dominio, err := DominioDeOrigen(c.pestana)
			if err != nil {
				t.Fatalf("la pestaña «%s» no da dominio: %v", c.pestana, err)
			}
			if tengo := Encaja(c.sitio, dominio); tengo != c.encaja {
				t.Errorf("«%s» con la pestaña en «%s»: encaja=%v, debería ser %v",
					c.sitio, c.pestana, tengo, c.encaja)
			}
		})
	}
}

// De una pestaña de la que no se puede sacar un dominio **no se enseña nada**.
// El fallo tiene que ser hacia el lado de no ofrecer, nunca hacia el de ofrecer
// todo.
func TestLasPestanasEnLasQueNoSeRellena(t *testing.T) {
	malas := []struct {
		pestana string
		porque  string
	}{
		{"http://banco.es", "texto claro: la contraseña se regala a quien mire"},
		{"https://192.168.1.1", "una IP no tiene dominio registrable"},
		{"https://127.0.0.1:8080/admin", "lo mismo, y encima local"},
		{"https://[::1]/", "lo mismo en IPv6"},
		{"https://github.io", "es un sufijo público: no hay nada que registrar"},
		{"https://com", "lo mismo, más obvio"},
		{"https://localhost", "sin punto no hay dominio"},
		{"file:///home/alguien/entrar.html", "ni siquiera es la red"},
		{"about:blank", "tampoco"},
		{"javascript:alert(1)", "menos todavía"},
		{"", "vacío"},
		{"https://bánco.es", "sin convertir a punycode no se compara con nada"},
	}

	for _, m := range malas {
		if d, err := DominioDeOrigen(m.pestana); err == nil {
			t.Errorf("«%s» ha dado el dominio «%s» y no debería dar ninguno (%s)",
				m.pestana, d, m.porque)
		}
	}
}

// Y el corolario: sin dominio de pestaña, nada encaja. Es la garantía de que un
// error al leer el origen no se convierte en «enséñalo todo».
func TestSinDominioDePestanaNoEncajaNada(t *testing.T) {
	for _, sitio := range []string{"https://banco.es", "banco.es", "", "cualquier cosa"} {
		if Encaja(sitio, "") {
			t.Errorf("«%s» encaja con una pestaña sin dominio", sitio)
		}
	}
}

// Casos que conviene fijar del extractor, porque son los que se olvidan al
// tocarlo.
func TestElDominioRegistrableDeUnSitioGuardado(t *testing.T) {
	casos := map[string]string{
		"https://www.banco.es/particulares": "banco.es",
		"banco.es":                          "banco.es",
		"http://banco.es":                   "banco.es", // el esquema del guardado no decide nada
		"https://uno.dos.tres.banco.es":     "banco.es",
		"https://foo.github.io":             "foo.github.io", // sufijo público de dos tramos
		"https://tienda.amazon.co.uk":       "amazon.co.uk",  // y de tres
		"https://github.io":                 "",
		"https://192.168.1.1":               "",
		"":                                  "",
		"   ":                               "",
	}
	for sitio, quiero := range casos {
		if tengo := DominioDeSitio(sitio); tengo != quiero {
			t.Errorf("«%s» da «%s» y debería dar «%s»", sitio, tengo, quiero)
		}
	}
}
