package adjuntos

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Las recetas de los canales —el PKGBUILD del AUR, la fórmula y el cask de Homebrew, el paquete de
// Chocolatey— nombran los adjuntos **por su clave del catálogo**, y `herramientas/rellenar`
// sustituye esos marcadores al publicar.
//
// # Por qué esto se comprueba aquí y no allí
//
// `rellenar` ya falla si un marcador no se puede sustituir, y eso es lo que impide publicar un
// PKGBUILD con un `{{SHA:deb-arm64}}` dentro. Pero falla **en medio de una publicación**, con la
// etiqueta ya empujada: una clave mal escrita costaría una etiqueta huérfana, que en la fase 4 ya
// pasó tres veces. Esta prueba contesta lo mismo **antes**, en `make comprobar`, y sin red: las
// claves se pueden comprobar contra el catálogo; las sumas, no, porque viven en la publicación.
//
// Es la misma idea que la puerta de listas de la publicación, un paso antes.
func TestLasRecetasDeLosCanalesNombranAdjuntosQueExisten(t *testing.T) {
	marcador := regexp.MustCompile(`\{\{(NOMBRE|SHA):([a-z0-9-]+)\}\}`)

	recetas := []string{
		"../../empaquetado/aur/PKGBUILD",
		"../../empaquetado/homebrew/esfinge.rb",
		"../../empaquetado/homebrew/esfinge-cask.rb",
		"../../empaquetado/chocolatey/esfinge.nuspec",
		"../../empaquetado/chocolatey/tools/chocolateyinstall.ps1",
		"../../empaquetado/winget/Webcafeina.Esfinge.yaml",
		"../../empaquetado/winget/Webcafeina.Esfinge.locale.es-ES.yaml",
		"../../empaquetado/winget/Webcafeina.Esfinge.installer.yaml",
	}

	// **Y que no haya una receta que esta prueba no mire.** Añadir un fichero a `empaquetado/` y
	// olvidarse de esta lista dejaría sus marcadores sin vigilar, que es exactamente el agujero
	// que esto viene a tapar.
	vistas := map[string]bool{}
	for _, r := range recetas {
		vistas[filepath.Clean(r)] = true
	}
	for _, dir := range []string{"../../empaquetado/aur", "../../empaquetado/homebrew", "../../empaquetado/chocolatey", "../../empaquetado/chocolatey/tools", "../../empaquetado/winget"} {
		entradas, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("no encuentro %s: %v", dir, err)
		}
		for _, e := range entradas {
			if e.IsDir() {
				continue
			}
			ruta := filepath.Join(dir, e.Name())
			crudo, err := os.ReadFile(ruta)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(crudo), "{{") {
				continue
			}
			if !vistas[filepath.Clean(ruta)] {
				t.Errorf("%s lleva marcadores y no está en la lista de esta prueba", ruta)
			}
		}
	}

	cuantos := 0
	for _, r := range recetas {
		crudo, err := os.ReadFile(r)
		if err != nil {
			t.Fatalf("no encuentro la receta %s: %v", r, err)
		}
		for _, m := range marcador.FindAllStringSubmatch(string(crudo), -1) {
			cuantos++
			clave := m[2]
			f, hay := Por(clave)
			if !hay {
				t.Errorf("%s: %s nombra «%s», que no está en el catálogo", r, m[0], clave)
				continue
			}
			// **Y tiene que estar publicado**, no solo existir: un adjunto que el catálogo nombra
			// pero que nadie fabrica todavía no está en la publicación, así que `rellenar` no
			// encontraría su suma y la receta saldría a medias.
			if !f.Publicado {
				t.Errorf("%s: %s nombra «%s», que el catálogo todavía no publica", r, m[0], clave)
			}
		}
	}
	if cuantos == 0 {
		t.Fatal("no he encontrado ningún marcador: o las recetas han cambiado de forma o esta prueba no comprueba nada")
	}
}

// La ficha de Chocolatey va **en inglés primero y en español detrás**, y eso no es una preferencia
// nuestra: lo pidió su moderación el 2026-11-18 —«the description is only in Spanish… it should be
// written in English, and you can keep the Spanish after it»— y mientras no se corrigió, el paquete
// se quedó esperando sin que nada aquí lo dijera.
//
// # Por qué hace falta una prueba para esto
//
// Porque en este proyecto **todo se escribe en español**, así que lo natural al tocar esta ficha es
// volver a escribirla en español, y el aviso llega **días después** y por correo: el paquete entra,
// pasa las comprobaciones automáticas, y se para en «Waiting for Maintainer». Eso cuesta una ronda
// de moderación entera, que son días, y no lo caza ninguna otra puerta.
//
// Lo que se mira es **sólo lo que llega a la tienda** —el resumen y la descripción—, no los
// comentarios del fichero, que son nuestros y van en español a propósito. Y winget no entra aquí:
// allí la ficha va por idioma (`locale.es-ES`), así que el español es lo correcto.
func TestLaFichaDeChocolateyEmpiezaEnIngles(t *testing.T) {
	const ficha = "../../empaquetado/chocolatey/esfinge.nuspec"
	crudo, err := os.ReadFile(ficha)
	if err != nil {
		t.Fatalf("no encuentro la ficha: %v", err)
	}

	saca := func(patron string) string {
		m := regexp.MustCompile(`(?s)` + patron).FindStringSubmatch(string(crudo))
		if m == nil {
			t.Fatalf("%s: no encuentro %s; la ficha ha cambiado de forma y esta prueba no comprueba nada", ficha, patron)
		}
		return m[1]
	}
	resumen := saca(`<summary>(.*?)</summary>`)
	descripcion := saca(`<!\[CDATA\[(.*?)\]\]>`)

	// El título de la parte en español es lo que marca dónde acaba el inglés, así que si
	// desaparece, esta prueba no sabría dónde mirar: antes que dar un verde que no significa
	// nada, se dice.
	const enEspanol = "## En español"
	corte := strings.Index(descripcion, enEspanol)
	if corte < 0 {
		t.Fatalf("%s: la descripción no lleva «%s», que es lo que separa los dos idiomas", ficha, enEspanol)
	}

	// Lo que delata que un trozo está en español sin tener que saber inglés: lo que el inglés no
	// escribe nunca.
	deEspanol := "áéíóúüñÁÉÍÓÚÜÑ¿¡«»"
	mira := func(que, texto string) {
		for i, r := range texto {
			if strings.ContainsRune(deEspanol, r) {
				desde := i - 60
				if desde < 0 {
					desde = 0
				}
				t.Errorf("%s: %s lleva «%c», o sea que está en español, y la moderación de Chocolatey lo quiere en inglés (el español va detrás de «%s»). Dice: …%s…",
					ficha, que, r, enEspanol, texto[desde:i+1])
				return
			}
		}
	}
	mira("el resumen", resumen)
	mira("la descripción, antes de la parte en español", descripcion[:corte])
}
