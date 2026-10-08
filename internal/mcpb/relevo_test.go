package mcpb

import (
	"strings"
	"testing"
)

// **No cederse el turno a uno mismo**, que es la cadena infinita de procesos.
//
// Pasa de verdad y no es un caso raro: es **el de Claude Code**, donde la orden ya
// apunta al servidor de dentro de la aplicación. Sin esta comprobación, ese binario se
// encontraría a sí mismo en la lista de sitios y se lanzaría una y otra vez.
func TestNoSeCedeElTurnoASiMismo(t *testing.T) {
	for _, c := range []struct{ sistema, casa, yo string }{
		{"darwin", "/Users/alvaro", "/Applications/Esfinge.app/Contents/MacOS/esfinge-mcp"},
		{"windows", `C:\Users\alvaro`, `C:\Program Files\Esfinge\esfinge-mcp.exe`},
		{"linux", "/home/alvaro", "/usr/bin/esfinge-mcp"},
	} {
		// Todo existe: el único motivo para no ceder tiene que ser que sea él.
		otro := AQuienCederle(c.sistema, c.casa, c.yo, func(string) bool { return true })
		if otro != "" {
			t.Errorf("%s: siendo %q se cedería a %q, que es una cadena sin fin", c.sistema, c.yo, otro)
		}
	}
}

// Y cuando el que corre es el del paquete, el turno es del instalado.
func TestElDelPaqueteCedeAlInstalado(t *testing.T) {
	// Donde Claude Desktop instala las extensiones: una carpeta suya, no la de Esfinge.
	yo := "/Users/alvaro/Library/Application Support/Claude/extensions/esfinge/server/esfinge-mcp"
	quiero := "/Applications/Esfinge.app/Contents/MacOS/esfinge-mcp"
	otro := AQuienCederle("darwin", "/Users/alvaro", yo, func(r string) bool { return r == quiero })
	if otro != quiero {
		t.Errorf("tenía que ceder a %q y dice %q", quiero, otro)
	}
}

// **Sin Esfinge instalada no se cede nada**, y eso no es un fallo: es el respaldo. Lo
// peor que puede pasar con todo esto es quedarse como se estaba.
func TestSinEsfingeInstaladaSigueElDelPaquete(t *testing.T) {
	otro := AQuienCederle("darwin", "/Users/alvaro", "/tmp/esfinge-mcp", func(string) bool { return false })
	if otro != "" {
		t.Errorf("no hay nada instalado y dice de ceder a %q", otro)
	}
}

// Los tres sistemas miran donde deja las cosas su instalador, **y el de Windows lleva
// `.exe`**: una lista sin eso no encontraría nada y nadie se enteraría, porque no
// encontrar nada es justo el caso que no rompe.
func TestDondeSeBuscaEnCadaSistema(t *testing.T) {
	for _, c := range []struct{ sistema, casa, hace string }{
		{"darwin", "/Users/alvaro", "/Applications/Esfinge.app/Contents/MacOS/esfinge-mcp"},
		{"windows", `C:\Users\alvaro`, "esfinge-mcp.exe"},
		{"linux", "/home/alvaro", "/usr/bin/esfinge-mcp"},
	} {
		sitios := DondeBuscarElServidor(c.sistema, c.casa)
		if len(sitios) < 2 {
			t.Errorf("%s: solo se mira en %v", c.sistema, sitios)
		}
		// **Con `strings` y no con `filepath`**, por lo mismo que la función que se
		// está probando: `filepath.Base` usa el separador de esta máquina y no
		// partiría una ruta de Windows. Una prueba que mire la tabla de otro sistema
		// no puede usar las herramientas de éste.
		var hay bool
		for _, s := range sitios {
			if s == c.hace || strings.HasSuffix(s, c.hace) {
				hay = true
			}
		}
		if !hay {
			t.Errorf("%s: %q no está entre %v", c.sistema, c.hace, sitios)
		}
		// Y la del usuario, que es donde se instala sin permisos de administrador.
		var personal bool
		for _, s := range sitios {
			if len(s) > len(c.casa) && s[:len(c.casa)] == c.casa {
				personal = true
			}
		}
		if !personal {
			t.Errorf("%s: no se mira en la carpeta del usuario: %v", c.sistema, sitios)
		}
	}
}
