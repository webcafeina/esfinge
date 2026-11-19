package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/adjuntos"
)

// **Las recetas de verdad se rellenan enteras**, con el catálogo y con un SHA256SUMS de mentira.
//
// # Qué agujero tapa esto
//
// `rellenar` ya falla si le queda un marcador, pero falla **en medio de una publicación**, y antes
// de hoy nadie pasaba las plantillas por él sin red. El 2026-11-19 eso costó una vuelta en la VM
// del cliente: el cask necesita `{{appdir}}` —un marcador **de Homebrew**, que él sustituye al
// instalar— y esta herramienta se habría puesto roja al verlo, o peor, lo habría tomado por uno
// nuestro mal escrito. Aquí se contesta en `make comprobar`: cada plantilla sale completa y los
// marcadores ajenos **siguen ahí**.
//
// Las plantillas se buscan recorriendo `empaquetado/`, no con una lista escrita aquí: una receta
// nueva queda cubierta sin que nadie se acuerde de nada.
func TestLasRecetasSeRellenanEnteras(t *testing.T) {
	const version = "9.9.9"

	// Un SHA256SUMS de mentira con **todo lo que el catálogo publica**: lo que se comprueba es que
	// la plantilla no pida nada que no exista, no las sumas, que viven en la publicación.
	sumas := map[string]string{}
	for _, nombre := range adjuntos.Publicados(version) {
		sumas[nombre] = strings.Repeat("ab", 32)
	}

	cuantas := 0
	raiz := filepath.Join("..", "..", "empaquetado")
	err := filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		crudo, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		plantilla := string(crudo)
		if !strings.Contains(plantilla, "{{") {
			return nil
		}
		cuantas++
		salida, problemas := Rellenar(plantilla, version, sumas)
		for _, p := range problemas {
			t.Errorf("%s: %s", ruta, p)
		}
		// Y que lo nuestro se haya ido de verdad: un marcador que se quedara dentro llegaría al
		// canal, y ahí lo lee el equipo de quien instala.
		for _, nuestro := range []string{"{{VERSION}}", "{{NOMBRE:", "{{SHA:"} {
			if strings.Contains(salida, nuestro) {
				t.Errorf("%s: queda «%s» sin sustituir", ruta, nuestro)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo %s: %v", raiz, err)
	}
	if cuantas == 0 {
		t.Fatal("no he encontrado ninguna plantilla: esta prueba no comprueba nada")
	}
}

// El marcador de Homebrew **llega intacto al tap**, que es lo suyo: lo sustituye él en el equipo
// de quien instala, con el directorio de aplicaciones que tenga configurado.
//
// # Lo que se mira, y por qué no vale mirar el fichero
//
// La primera versión de esto preguntaba si «{{appdir}}» aparecía **en el cask**, y pasaba en verde
// con el fallo dentro: el nombre aparece también en el comentario que lo explica, así que devolver
// la línea del `xattr` a la forma de Ruby no la ponía roja. Lo dijo mutarla, no leerla. Lo que se
// mira es **la línea que recibe el `xattr`**, que es la única que Homebrew ejecuta.
func TestElMarcadorDeHomebrewLlegaIntacto(t *testing.T) {
	const cask = "../../empaquetado/homebrew/esfinge-cask.rb"
	crudo, err := os.ReadFile(cask)
	if err != nil {
		t.Fatalf("no encuentro el cask: %v", err)
	}

	// La línea de los argumentos del xattr: la que nombra el atributo de la cuarentena.
	var laLinea string
	for _, l := range strings.Split(string(crudo), "\n") {
		if strings.Contains(l, "com.apple.quarantine") && strings.Contains(l, "args:") {
			laLinea = l
			break
		}
	}
	if laLinea == "" {
		t.Fatal("no encuentro la línea que le pasa la cuarentena al xattr; si ese paso ya no existe, " +
			"esta prueba sobra, y si ha cambiado de forma hay que mirarla a mano")
	}
	// `#{...}` es la interpolación de Ruby, y dentro de este bloque **no hay método `appdir`**: el
	// cask queda «unreadable» y eso tumba también la fórmula, porque cargar el tap evalúa las dos
	// recetas. Lo vio el cliente el 2026-11-19 estrenando su VM de macOS.
	if strings.Contains(laLinea, "#{appdir}") {
		t.Errorf("%s le pasa al xattr «#{appdir}», que es interpolación de Ruby: ahí no existe ese "+
			"método y el cask no carga. Va el marcador de Homebrew, con llaves dobles. Dice: %s",
			cask, strings.TrimSpace(laLinea))
	}
	if !strings.Contains(laLinea, "{{appdir}}") {
		t.Errorf("%s no le pasa al xattr el marcador «{{appdir}}», así que Homebrew no sabrá dónde "+
			"está la aplicación. Dice: %s", cask, strings.TrimSpace(laLinea))
	}

	// Y que sobreviva al relleno: si `rellenar` lo tomara por nuestro, llegaría al tap sustituido
	// por cualquier cosa o pararía la publicación.
	salida, problemas := Rellenar(string(crudo), "9.9.9", map[string]string{
		adjuntos.Catalogo()[0].Nombre("9.9.9"): strings.Repeat("ab", 32),
	})
	for _, p := range problemas {
		if strings.Contains(p, "appdir") {
			t.Errorf("rellenar se queja de un marcador que es de Homebrew: %s", p)
		}
	}
	if !strings.Contains(salida, `"{{appdir}}/Esfinge.app"`) {
		t.Error("el cask rellenado ya no le pasa «{{appdir}}/Esfinge.app» al xattr")
	}
}
