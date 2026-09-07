package actualizacion

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompararVersiones(t *testing.T) {
	casos := []struct {
		nombre     string
		candidata  string
		actual     string
		masNueva   bool
		sabeLeerla bool
	}{
		{"una menor más alta avisa", "2.1.0", "2.0.3", true, true},
		{"un parche más alto avisa", "2.0.4", "2.0.3", true, true},
		{"una mayor más alta avisa", "3.0.0", "2.9.9", true, true},
		{"la misma no avisa", "2.0.3", "2.0.3", false, true},
		{"una anterior no avisa", "2.0.2", "2.0.3", false, true},
		{"la «v» delante da igual", "v2.1.0", "v2.0.3", true, true},
		{"diez es mayor que nueve, no menor", "2.10.0", "2.9.0", true, true},

		// Lo importante de estos tres: una compilación de trabajo no puede
		// acabar ofreciéndole una descarga a quien la está compilando.
		{"«dev» no se sabe leer", "2.1.0", "dev", false, false},
		{"lo que sale de git describe tampoco", "2.1.0", "2.0.3-3-gabc1234", false, false},
		{"ni una versión de dos números", "2.1", "2.0.3", false, false},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			mas, err := EsMasNueva(c.candidata, c.actual)
			if (err == nil) != c.sabeLeerla {
				t.Fatalf("saber leerla: quiero %v, tengo %v (%v)", c.sabeLeerla, err == nil, err)
			}
			if mas != c.masNueva {
				t.Errorf("%s frente a %s: quiero %v, tengo %v",
					c.candidata, c.actual, c.masNueva, mas)
			}
		})
	}
}

func TestBuscarElResumenEnLasSumas(t *testing.T) {
	sumas := "aaaa  Esfinge-2.1.0.dmg\n" +
		"bbbb  esfinge_2.1.0_amd64.deb\n" +
		"cccc *esfinge-2.1.0-linux-amd64\n"

	casos := map[string]string{
		"Esfinge-2.1.0.dmg":         "aaaa",
		"esfinge_2.1.0_amd64.deb":   "bbbb",
		"esfinge-2.1.0-linux-amd64": "cccc", // con el asterisco de coreutils delante
		"no-existe.zip":             "",
	}
	for fichero, quiero := range casos {
		if tengo := ResumenEn(sumas, fichero); tengo != quiero {
			t.Errorf("%s: quiero %q, tengo %q", fichero, quiero, tengo)
		}
	}
}

func TestElegirElFicheroDeCadaSistema(t *testing.T) {
	adjuntos := []Adjunto{
		{Nombre: "Esfinge-2.1.0.dmg"},
		{Nombre: "Esfinge-2.1.0-windows-instalador.exe"},
		{Nombre: "esfinge_2.1.0_amd64.deb"},
		{Nombre: "esfinge-2.1.0-linux-amd64.tar.gz"},
		{Nombre: "esfinge-2.1.0-windows-amd64.exe"}, // la línea de comandos, que no es
		{Nombre: "SHA256SUMS"},
	}

	// ParaEsteSistema mira runtime.GOOS, así que aquí solo se puede comprobar el
	// sistema en el que corren los tests. Lo que importa: que elige uno de los
	// buenos y nunca el binario suelto de la línea de comandos.
	a, vale := ParaEsteSistema(adjuntos)
	if !vale {
		t.Fatal("no ha elegido nada para este sistema")
	}
	buenos := map[string]bool{
		"Esfinge-2.1.0.dmg":                    true,
		"Esfinge-2.1.0-windows-instalador.exe": true,
		"esfinge_2.1.0_amd64.deb":              true,
		"esfinge-2.1.0-linux-amd64.tar.gz":     true,
	}
	if !buenos[a.Nombre] {
		t.Errorf("ha elegido %q, que no es un instalador", a.Nombre)
	}

	if _, vale := ParaEsteSistema(nil); vale {
		t.Error("sin adjuntos no puede elegir nada, y dice que sí")
	}
}

// servidorFalso hace de API de GitHub para no tocar la red en los tests.
func servidorFalso(t *testing.T, pub any) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("la petición va sin User-Agent, y GitHub las rechaza")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pub)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestMirarEncuentraLaVersionNueva(t *testing.T) {
	s := servidorFalso(t, map[string]any{
		"tag_name": "v2.1.0",
		"html_url": "https://github.com/webcafeina/esfinge/releases/tag/v2.1.0",
		"assets": []map[string]any{
			{"name": "Esfinge-2.1.0.dmg", "browser_download_url": "https://x/dmg", "size": 8},
			{"name": "Esfinge-2.1.0-windows-instalador.exe", "browser_download_url": "https://x/exe", "size": 7},
			{"name": "esfinge_2.1.0_amd64.deb", "browser_download_url": "https://x/deb", "size": 6},
			{"name": "esfinge-2.1.0-linux-amd64.tar.gz", "browser_download_url": "https://x/tgz", "size": 5},
		},
	})

	c := Nuevo("2.0.3")
	c.API = s.URL

	n, err := c.Mirar()
	if err != nil {
		t.Fatalf("mirar: %v", err)
	}
	if !n.Hay {
		t.Fatal("hay una 2.1.0 publicada y dice que no hay novedad")
	}
	if n.Version != "2.1.0" {
		t.Errorf("versión: quiero 2.1.0, tengo %q", n.Version)
	}
	if n.Fichero == "" || n.URL == "" {
		t.Errorf("no ha elegido fichero para este sistema: %+v", n)
	}
}

func TestMirarConLaMismaVersionNoAvisa(t *testing.T) {
	s := servidorFalso(t, map[string]any{"tag_name": "v2.0.3"})
	c := Nuevo("2.0.3")
	c.API = s.URL

	n, err := c.Mirar()
	if err != nil {
		t.Fatalf("mirar: %v", err)
	}
	if n.Hay {
		t.Error("la versión publicada es la instalada, y avisa igual")
	}
}

// Una compilación de trabajo no puede acabar ofreciendo descargas: quien tiene
// «dev» instalado está delante del código fuente.
func TestUnaCompilacionDeTrabajoNoRecibeAvisos(t *testing.T) {
	s := servidorFalso(t, map[string]any{"tag_name": "v9.9.9"})
	for _, version := range []string{"dev", "2.0.3-3-gabc1234", ""} {
		c := Nuevo(version)
		c.API = s.URL

		n, err := c.Mirar()
		if err != nil {
			t.Fatalf("%q: %v", version, err)
		}
		if n.Hay {
			t.Errorf("%q: no debería recibir avisos", version)
		}
	}
}

func TestUnaPublicacionEnBorradorNoCuenta(t *testing.T) {
	s := servidorFalso(t, map[string]any{"tag_name": "v2.1.0", "prerelease": true})
	c := Nuevo("2.0.3")
	c.API = s.URL

	if _, err := c.Mirar(); err == nil {
		t.Error("una versión previa no es para el cliente, y la ha dado por buena")
	}
}
