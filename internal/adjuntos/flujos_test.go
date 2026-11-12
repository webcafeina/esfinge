package adjuntos

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// plataformasDeLaApp son los objetivos de `wails build` que el catálogo implica.
//
// macOS va en un solo paquete universal, así que sus dos arquitecturas son **un** objetivo.
// Lo demás, uno por sistema y arquitectura.
func plataformasDeLaApp(soloPublicados bool) []string {
	vistos := map[string]bool{}
	for _, f := range Catalogo() {
		if f.Clase != ClaseApp && f.Clase != ClasePaquete {
			continue
		}
		if soloPublicados && !f.Publicado {
			continue
		}
		if f.Sistema == "darwin" {
			vistos["darwin/universal"] = true
			continue
		}
		vistos[f.Sistema+"/"+f.Arq] = true
	}
	var out []string
	for p := range vistos {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func leer(t *testing.T, camino string) string {
	t.Helper()
	b, err := os.ReadFile(camino)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestCompilarPruebaTodoLoQueElCatalogoNombra.
//
// «Compilar» es el flujo que va **por delante**: se dispara a mano, no publica nada y
// existe para que una plataforma nueva se pruebe antes de que una etiqueta dependa de
// ella. Así que tiene que cubrir **todo** lo que el catálogo nombra, publicado o no.
//
// Y esto es una lista en dos sitios —dos YAML— que no se puede juntar, así que toca lo
// que manda esta casa: **compararlas**. Es la sexta vez que se escribe esa frase aquí.
func TestCompilarPruebaTodoLoQueElCatalogoNombra(t *testing.T) {
	yaml := leer(t, "../../.github/workflows/compilar.yml")
	hay := map[string]bool{}
	for _, m := range regexp.MustCompile(`plataforma:\s*(\S+)`).FindAllStringSubmatch(yaml, -1) {
		hay[m[1]] = true
	}
	if len(hay) == 0 {
		t.Fatal("no se encuentra ninguna «plataforma:» en compilar.yml, o cambió cómo se declara")
	}

	// **Y cada pata tiene que tener su máquina.** La primera versión de esto solo
	// contaba las plataformas, así que borrar el `os:` de la pata de ARM pasaba en
	// verde: la plataforma seguía declarada y no había dónde compilarla. Lo cazó
	// mutarlo, no leerlo.
	for i, trozo := range strings.Split(yaml, "- nombre:")[1:] {
		if !strings.Contains(trozo, "plataforma:") {
			continue
		}
		if !regexp.MustCompile(`(?m)^\s+os:\s*\S+`).MatchString(trozo) {
			t.Errorf("la pata %d de la matriz declara su plataforma y no dice en qué máquina "+
				"se compila", i+1)
		}
	}
	for _, p := range plataformasDeLaApp(false) {
		if !hay[p] {
			t.Errorf("el catálogo nombra adjuntos de %s y «Compilar» no lo compila: "+
				"esa plataforma no la prueba nadie hasta el día de la etiqueta", p)
		}
	}
}

// TestPublicarHaceExactamenteLoQueEstaPublicado, en los dos sentidos.
//
// Si publica una plataforma que el catálogo no da por publicada, la puerta de la
// publicación rechazará el adjunto que salga de ahí —sobra— y la etiqueta se cae con todo
// compilado. Y si **no** compila una que sí está marcada, se cae por lo otro: falta.
//
// Las dos mitades importan, y la segunda es la que hace que flipear `Publicado` sin
// escribir el trabajo que lo fabrica se vea **aquí** y no en GitHub veinte minutos después.
func TestPublicarHaceExactamenteLoQueEstaPublicado(t *testing.T) {
	yaml := leer(t, "../../.github/workflows/publicar.yml")
	hay := map[string]bool{}
	for _, m := range regexp.MustCompile(`wails build -platform (\S+)`).FindAllStringSubmatch(yaml, -1) {
		hay[m[1]] = true
	}
	if len(hay) == 0 {
		t.Fatal("no se encuentra ningún «wails build -platform» en publicar.yml")
	}
	quiere := map[string]bool{}
	for _, p := range plataformasDeLaApp(true) {
		quiere[p] = true
		if !hay[p] {
			t.Errorf("el catálogo da por publicados adjuntos de %s y «Publicar» no compila esa "+
				"plataforma: la publicación saldría sin ellos y la puerta la pararía", p)
		}
	}
	for p := range hay {
		if !quiere[p] {
			t.Errorf("«Publicar» compila %s y el catálogo no tiene ningún adjunto publicado de ahí: "+
				"lo que salga sobrará y la puerta parará la publicación", p)
		}
	}
}

// TestLasDosMatricesFijanSuImagen: ningún trabajo cuelga de una imagen móvil.
//
// A Apolo le retiraron `macos-14` **con apagones programados de por medio**, y una
// etiqueta empujada en esas horas se habría quedado sin el Mac. `ubuntu-latest` y
// `windows-latest` se dejan a propósito —son estables y las dos están aquí desde el
// principio—, pero `macos-latest` no: es la que ya se movió.
func TestLasDosMatricesFijanSuImagen(t *testing.T) {
	// **Se mira el valor, no el texto.** La primera versión de esto buscaba
	// «macos-latest» en todo el fichero y saltaba con los dos comentarios que explican
	// por qué no se usa: un vigilante que señala lo que está bien enseña a no leerlo.
	usa := regexp.MustCompile(`(?m)^\s*(?:runs-on|os):\s*macos-latest\s*$`)
	for _, f := range []string{"compilar.yml", "publicar.yml"} {
		yaml := leer(t, "../../.github/workflows/"+f)
		if usa.MatchString(yaml) {
			t.Errorf("%s corre en «macos-latest»: fija la versión, que esa imagen ya se retiró una vez", f)
		}
	}
}
