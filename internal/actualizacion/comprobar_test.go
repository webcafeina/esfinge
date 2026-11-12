package actualizacion

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/adjuntos"
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
	// **Los seis objetivos y los cuatro gestores, desde una sola máquina.**
	//
	// La prueba de antes decía en su propio comentario que «solo se puede comprobar el
	// sistema en el que corren los tests», porque la función leía `runtime` por dentro.
	// Eso es lo que hizo que nadie viera el fallo que esperaba al primer `.rpm`: a una
	// Fedora se le iba a ofrecer un `.deb`. Con el equipo por parámetro, aquí se
	// comprueban los veinticuatro casos.
	const v = "2.1.0"
	todos := func() []Adjunto {
		var out []Adjunto
		for _, n := range adjuntos.De(v) {
			out = append(out, Adjunto{Nombre: n})
		}
		return out
	}
	publicadosHoy := func() []Adjunto {
		var out []Adjunto
		for _, n := range adjuntos.Publicados(v) {
			out = append(out, Adjunto{Nombre: n})
		}
		return out
	}

	// Con todo publicado —el mundo de después de la última fase—, cada equipo se lleva
	// el suyo y nada más.
	casos := []struct {
		sistema, arq, gestor, quiere string
	}{
		{"darwin", "arm64", "", "Esfinge-2.1.0.dmg"},
		{"darwin", "amd64", "", "Esfinge-2.1.0.dmg"},

		{"windows", "amd64", "", "Esfinge-2.1.0-windows-amd64-instalador.exe"},
		{"windows", "arm64", "", "Esfinge-2.1.0-windows-arm64-instalador.exe"},

		{"linux", "amd64", adjuntos.GestorDeb, "esfinge_2.1.0_amd64.deb"},
		{"linux", "arm64", adjuntos.GestorDeb, "esfinge_2.1.0_arm64.deb"},
		{"linux", "amd64", adjuntos.GestorRPM, "esfinge-2.1.0-1.x86_64.rpm"},
		{"linux", "arm64", adjuntos.GestorRPM, "esfinge-2.1.0-1.aarch64.rpm"},
		{"linux", "amd64", adjuntos.GestorAppImage, "Esfinge-2.1.0-x86_64.AppImage"},
		{"linux", "arm64", adjuntos.GestorAppImage, "Esfinge-2.1.0-aarch64.AppImage"},
		{"linux", "amd64", adjuntos.SinGestor, "esfinge-2.1.0-linux-amd64.tar.gz"},
		{"linux", "arm64", adjuntos.SinGestor, "esfinge-2.1.0-linux-arm64.tar.gz"},
	}
	for _, c := range casos {
		a, vale := elDeEsteEquipo(todos(), c.sistema, c.arq, c.gestor)
		if !vale {
			t.Errorf("%s/%s con gestor %q: no ha elegido nada", c.sistema, c.arq, c.gestor)
			continue
		}
		if a.Nombre != c.quiere {
			t.Errorf("%s/%s con gestor %q: ha elegido %q y le toca %q",
				c.sistema, c.arq, c.gestor, a.Nombre, c.quiere)
		}
	}

	// **En Arch no se descarga nada**: lo actualiza `pacman`, y ofrecerle un fichero
	// que su gestor no conoce es peor que no ofrecerle ninguno.
	for _, arq := range []string{"amd64", "arm64"} {
		if a, vale := elDeEsteEquipo(todos(), "linux", arq, adjuntos.GestorPacman); vale {
			t.Errorf("a una máquina de pacman se le ofrece %q", a.Nombre)
		}
	}

	// **El mundo de antes de la fase 2**, escrito a mano y no leído del catálogo.
	//
	// Aquí se afirma la propiedad que hace que el renombrado del instalador de Windows
	// cueste **una vez y no dos**: una copia instalada cuando solo existía el nombre viejo
	// lo encuentra igual, y una máquina ARM —para la que entonces no había nada— no se
	// lleva el binario de la otra arquitectura, sino nada, que es el camino que manda a la
	// página.
	//
	// **Y va escrito y no leído a propósito.** La primera versión preguntaba por
	// `adjuntos.Publicados`, o sea «lo que se fabrica hoy», y el día que la fase 2 marcó
	// los adjuntos de ARM como publicados estas tres afirmaciones se cayeron —con razón:
	// el mundo había cambiado—. Una prueba que dice «hoy» envejece sola; lo que no
	// envejece es «en un mundo donde solo existía esto». Es la lección de `soloLectura`.
	antesDeLaFase2 := []Adjunto{
		{Nombre: "Esfinge-2.1.0.dmg"},
		{Nombre: "Esfinge-2.1.0-windows-instalador.exe"},
		{Nombre: "esfinge_2.1.0_amd64.deb"},
		{Nombre: "esfinge-2.1.0-linux-amd64.tar.gz"},
		{Nombre: "SHA256SUMS"},
	}
	if a, vale := elDeEsteEquipo(antesDeLaFase2, "windows", "amd64", ""); !vale {
		t.Error("con solo el nombre viejo, un Windows amd64 se queda sin instalador")
	} else if a.Nombre != "Esfinge-2.1.0-windows-instalador.exe" {
		t.Errorf("con solo el nombre viejo ha elegido %q", a.Nombre)
	}
	if a, vale := elDeEsteEquipo(antesDeLaFase2, "windows", "arm64", ""); vale {
		t.Errorf("sin instalador de ARM se le ofrece %q, que es el de la otra arquitectura", a.Nombre)
	}
	if a, vale := elDeEsteEquipo(antesDeLaFase2, "linux", "arm64", adjuntos.GestorDeb); vale {
		t.Errorf("sin .deb de ARM se le ofrece %q, que dpkg rechazaría", a.Nombre)
	}

	// **Y el mundo de ahora**, el que de verdad se fabrica: cada arquitectura se lleva el
	// suyo, y el de Windows amd64 es ya **el nombre nuevo** y no el viejo.
	if a, vale := elDeEsteEquipo(publicadosHoy(), "windows", "amd64", ""); !vale {
		t.Error("un Windows amd64 se queda sin instalador")
	} else if a.Nombre != "Esfinge-2.1.0-windows-amd64-instalador.exe" {
		t.Errorf("un Windows amd64 se lleva %q y le toca el nombre nuevo", a.Nombre)
	}
	if a, vale := elDeEsteEquipo(publicadosHoy(), "windows", "arm64", ""); !vale {
		t.Error("un Windows ARM se queda sin instalador, y ya se publica el suyo")
	} else if a.Nombre != "Esfinge-2.1.0-windows-arm64-instalador.exe" {
		t.Errorf("un Windows ARM se lleva %q", a.Nombre)
	}
	if a, vale := elDeEsteEquipo(publicadosHoy(), "linux", "arm64", adjuntos.GestorDeb); !vale {
		t.Error("un Linux ARM con .deb se queda sin paquete, y ya se publica el suyo")
	} else if a.Nombre != "esfinge_2.1.0_arm64.deb" {
		t.Errorf("un Linux ARM se lleva %q", a.Nombre)
	}

	// Y nunca, en ningún caso, un binario suelto de la línea de comandos ni las sumas.
	for _, c := range casos {
		a, _ := elDeEsteEquipo(todos(), c.sistema, c.arq, c.gestor)
		if a.Nombre == "SHA256SUMS" || strings.HasPrefix(a.Nombre, "esfinge-mcp-") ||
			strings.HasPrefix(a.Nombre, "esfinge-puente-") {
			t.Errorf("%s/%s: ha elegido %q, que no es un instalador", c.sistema, c.arq, a.Nombre)
		}
	}

	if _, vale := elDeEsteEquipo(nil, "darwin", "arm64", ""); vale {
		t.Error("sin adjuntos no puede elegir nada, y dice que sí")
	}
	if _, vale := elDeEsteEquipo(todos(), "plan9", "amd64", ""); vale {
		t.Error("para un sistema que no existe ha elegido algo")
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
