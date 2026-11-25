package adjuntos

import (
	"encoding/xml"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Las pruebas de `empaquetado/linux/esfinge.metainfo.xml`, la ficha que lee un centro de software.
//
// # Por qué hace falta vigilarla
//
// Esa ficha repite, en un fichero que nadie compila, cosas que ya están escritas en otros tres
// sitios: el identificador del puente con el navegador, el nombre del lanzador y el del icono. Es
// **la misma regla en dos listas** —lo que en esta casa ya ha costado siete fallos—, con el
// agravante de que aquí nada se pone rojo: un `<launchable>` que no case con el `.desktop`
// instalado no es un error de XML, es una ficha que **el centro de software no asocia con la
// aplicación**, y eso sólo se ve abriendo el App Center de una máquina que no tenemos.
//
// Lo que estas pruebas **no** hacen es validar AppStream: eso es `appstreamcli validate --pedantic`
// y se corre a mano aquí. No entra en `make comprobar` a propósito, porque `appstreamcli` está en
// este VPS y **puede no estar en las máquinas que publican** —la tercera vez que eso cuesta algo,
// después de `zip` y de NSIS—, y una comprobación que se salta cuando falta la herramienta es un
// adorno.
const rutaDeLaFicha = "../../empaquetado/linux/esfinge.metainfo.xml"

type tradDeLaFicha struct {
	Idioma string `xml:"lang,attr"`
	Texto  string `xml:",chardata"`
}

type fichaDeAppStream struct {
	ID              string          `xml:"id"`
	LicenciaDeDatos string          `xml:"metadata_license"`
	LicenciaDelObra string          `xml:"project_license"`
	Nombres         []tradDeLaFicha `xml:"name"`
	Resumenes       []tradDeLaFicha `xml:"summary"`
	Parrafos        []tradDeLaFicha `xml:"description>p"`
	Lanzable        struct {
		Tipo  string `xml:"type,attr"`
		Valor string `xml:",chardata"`
	} `xml:"launchable"`
	Icono struct {
		Tipo  string `xml:"type,attr"`
		Valor string `xml:",chardata"`
	} `xml:"icon"`
	Clasificacion *struct {
		Tipo string `xml:"type,attr"`
	} `xml:"content_rating"`
}

func laFicha(t *testing.T) fichaDeAppStream {
	t.Helper()
	crudo, err := os.ReadFile(rutaDeLaFicha)
	if err != nil {
		t.Fatalf("no he podido leer la ficha de AppStream: %v", err)
	}
	var f fichaDeAppStream
	if err := xml.Unmarshal(crudo, &f); err != nil {
		t.Fatalf("la ficha de AppStream no es XML válido: %v", err)
	}
	return f
}

// Lo mínimo que un centro de software necesita para enseñar algo.
func TestLaFichaDeAppStreamTieneLoQueLaTiendaLee(t *testing.T) {
	f := laFicha(t)

	if f.ID == "" || f.LicenciaDeDatos == "" || f.LicenciaDelObra == "" {
		t.Errorf("faltan campos obligatorios: id=%q metadata_license=%q project_license=%q",
			f.ID, f.LicenciaDeDatos, f.LicenciaDelObra)
	}
	if f.Lanzable.Tipo != "desktop-id" || f.Lanzable.Valor == "" {
		t.Errorf("sin <launchable type=\"desktop-id\"> la ficha no se asocia con la aplicación instalada: tipo=%q valor=%q",
			f.Lanzable.Tipo, f.Lanzable.Valor)
	}
	if f.Clasificacion == nil {
		t.Error("sin <content_rating> la ficha sale con un aviso de contenido en GNOME Software")
	}
}

// **El identificador es el mismo que el del puente con el navegador**, y eso no es coincidencia:
// los dos nombran a Esfinge en DNS invertido. Escritos por separado se desalinean, y entonces hay
// dos nombres para la misma aplicación sin que nada avise.
func TestElIdentificadorDeAppStreamEsElDelPuente(t *testing.T) {
	fuente, err := os.ReadFile("../../internal/app/manifiestos.go")
	if err != nil {
		t.Fatalf("no he podido leer manifiestos.go: %v", err)
	}
	m := regexp.MustCompile(`nombreDelHost\s*=\s*"([^"]+)"`).FindSubmatch(fuente)
	if m == nil {
		t.Fatal("no encuentro «nombreDelHost» en manifiestos.go: o ha cambiado de nombre o esta prueba no comprueba nada")
	}
	delPuente := string(m[1])

	if f := laFicha(t); f.ID != delPuente {
		t.Errorf("el identificador de la ficha es %q y el del puente %q: Esfinge tendría dos nombres", f.ID, delPuente)
	}
}

// Y apunta al lanzador y al icono **que el paquete instala de verdad**, no a los que uno recuerda.
func TestLaFichaDeAppStreamApuntaALoQueSeInstala(t *testing.T) {
	nfpm, err := os.ReadFile("../../empaquetado/linux/nfpm.yaml")
	if err != nil {
		t.Fatalf("no he podido leer nfpm.yaml: %v", err)
	}
	f := laFicha(t)

	// El lanzador, por el nombre con el que acaba en /usr/share/applications.
	m := regexp.MustCompile(`dst:\s*/usr/share/applications/(\S+\.desktop)`).FindSubmatch(nfpm)
	if m == nil {
		t.Fatal("no encuentro el destino del .desktop en nfpm.yaml: o el fichero ha cambiado de forma o esta prueba no comprueba nada")
	}
	if instalado := string(m[1]); f.Lanzable.Valor != instalado {
		t.Errorf("la ficha apunta al lanzador %q y el paquete instala %q: el centro de software no la asociaría con la aplicación",
			f.Lanzable.Valor, instalado)
	}

	// El icono, por el nombre con el que acaba en hicolor/*/apps.
	m = regexp.MustCompile(`dst:\s*/usr/share/icons/hicolor/\d+x\d+/apps/(\S+)\.png`).FindSubmatch(nfpm)
	if m == nil {
		t.Fatal("no encuentro el destino de los iconos de la aplicación en nfpm.yaml")
	}
	if instalado := string(m[1]); f.Icono.Valor != instalado {
		t.Errorf("la ficha pide el icono %q y el paquete instala %q: saldría sin icono", f.Icono.Valor, instalado)
	}

	// Y lo mismo que pide el propio lanzador, que es la tercera copia de ese nombre.
	lanzador, err := os.ReadFile("../../empaquetado/linux/esfinge.desktop")
	if err != nil {
		t.Fatalf("no he podido leer el lanzador: %v", err)
	}
	if m := regexp.MustCompile(`(?m)^Icon=(.+)$`).FindSubmatch(lanzador); m == nil {
		t.Fatal("el lanzador no declara «Icon=»")
	} else if delLanzador := strings.TrimSpace(string(m[1])); f.Icono.Valor != delLanzador {
		t.Errorf("la ficha pide el icono %q y el lanzador %q", f.Icono.Valor, delLanzador)
	}
}

// **La llevan los tres empaquetados**, que es lo que la hace servir para algo: el `.deb` y el
// `.rpm` salen los dos de `nfpm.yaml`, y la AppImage se arma con su propio guion — o sea el sitio
// donde en esta casa se olvida siempre el tercero.
//
// # Y esta prueba nació mala, que es lo que hay que aprenderse
//
// La primera versión preguntaba si `esfinge.metainfo.xml` **aparecía en el fichero**, y comentando
// la orden de copia **seguía en verde**: el nombre sigue ahí, en el comentario que la explica y en
// la lista de «qué falta antes de armar». Es literalmente la trampa que este proyecto tiene
// escrita desde el tap de Homebrew —«preguntaba si el marcador aparecía en el fichero, y aparece
// en el comentario que lo explica»—. Lo dijo mutarla. Ahora se afirma **la orden**, sobre el
// fichero sin comentarios.
func TestLaFichaDeAppStreamViajaEnLosTresEmpaquetados(t *testing.T) {
	f := laFicha(t)

	// AppStream exige que el fichero instalado **se llame como el identificador**; si no, el
	// catálogo no lo recoge.
	seLlama := f.ID + ".metainfo.xml"

	nfpm := sinComentarios(t, "../../empaquetado/linux/nfpm.yaml", "#")
	puesto := regexp.MustCompile(`src:\s*\S*esfinge\.metainfo\.xml\s+dst:\s*/usr/share/metainfo/(\S+)`).FindStringSubmatch(nfpm)
	if puesto == nil {
		t.Fatal("nfpm.yaml no empaqueta la ficha en /usr/share/metainfo: el .deb y el .rpm saldrían sin ella")
	}
	if puesto[1] != seLlama {
		t.Errorf("el paquete instala la ficha como %q y AppStream la busca como %q: el catálogo no la recogería", puesto[1], seLlama)
	}

	// Y la AppImage, que es la tercera y la que se olvida.
	appimage := sinComentarios(t, "../../empaquetado/linux/armar-appimage.sh", "#")
	copia := regexp.MustCompile(`cp\s[^\n]*esfinge\.metainfo\.xml[^\n]*usr/share/metainfo/(\S+)"`).FindStringSubmatch(appimage)
	if copia == nil {
		t.Fatal("el guion de la AppImage no copia la ficha a usr/share/metainfo: es el tercer empaquetado y es el que se olvida")
	}
	if copia[1] != seLlama {
		t.Errorf("la AppImage instala la ficha como %q y AppStream la busca como %q", copia[1], seLlama)
	}
}

// Lee un fichero quitándole los comentarios y juntando las líneas que siguen con «\».
//
// Las dos cosas hacen falta: **sin quitar los comentarios** se da por buena una orden comentada
// —el fallo con el que nació la prueba de arriba— y **sin juntar las continuaciones** una orden
// partida en dos líneas no se puede afirmar de una pieza.
func sinComentarios(t *testing.T, ruta, marca string) string {
	t.Helper()
	crudo, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no he podido leer %s: %v", ruta, err)
	}
	var vivas []string
	for _, linea := range strings.Split(string(crudo), "\n") {
		if strings.HasPrefix(strings.TrimSpace(linea), marca) {
			continue
		}
		vivas = append(vivas, linea)
	}
	return strings.ReplaceAll(strings.Join(vivas, "\n"), "\\\n", " ")
}

// **La ficha es bilingüe, y el texto sin idioma es el inglés.** Es la lección del moderador de
// Chocolatey, que devolvió una ficha escrita sólo en español: lo que ve quien no tiene el sistema
// en español es el texto sin `xml:lang`, y lo que necesita saber antes de instalar es **que el
// programa está en español**. Escrita sólo en español, esto se pone rojo.
func TestLaFichaDeAppStreamHablaLosDosIdiomas(t *testing.T) {
	f := laFicha(t)

	porIdioma := func(ts []tradDeLaFicha) (sinIdioma, enEspanol int) {
		for _, x := range ts {
			if strings.TrimSpace(x.Texto) == "" {
				continue
			}
			if x.Idioma == "" {
				sinIdioma++
			} else if x.Idioma == "es" {
				enEspanol++
			}
		}
		return
	}

	for _, c := range []struct {
		que  string
		cual []tradDeLaFicha
	}{
		{"resumen", f.Resumenes},
		{"descripción", f.Parrafos},
	} {
		sinIdioma, enEspanol := porIdioma(c.cual)
		if sinIdioma == 0 {
			t.Errorf("el %s no tiene versión sin «xml:lang»: es la que ve quien no tiene el sistema en español", c.que)
		}
		if enEspanol == 0 {
			t.Errorf("el %s no tiene versión en español, que es el idioma del programa", c.que)
		}
	}

	// Y el resumen por omisión dice en qué idioma está el programa, que es lo que pedía el
	// moderador de Chocolatey: «you can keep the Spanish after it», pero diciéndolo.
	var pordefecto string
	for _, r := range f.Resumenes {
		if r.Idioma == "" {
			pordefecto = strings.ToLower(r.Texto)
		}
	}
	if !strings.Contains(pordefecto, "spanish") {
		t.Errorf("el resumen en inglés no dice que el programa está en español, y es lo primero que necesita saber quien lo lee: %q", pordefecto)
	}
}

// **El resumen del paquete es la misma frase que el de la ficha, y va sin acentos.**
//
// # Por qué sin acentos, siendo nuestro texto el correcto
//
// La ventana que abre un doble clic sobre el `.deb` **convierte esta línea a ASCII y pone «?»
// donde hay un acento**: el cliente leyó «contrase?as» en la primera línea de un gestor de
// contraseñas (2026-11-25). El fallo es de quien lo enseña —la política pide los ficheros de
// control en UTF-8 (5.1) y eso es lo que llevamos, y le pasa igual a `gh` y a otros 22 paquetes de
// este servidor—, así que esto no lo arregla: hace que **lo primero que se lee no dependa de él**.
//
// Y por eso el resumen es **la misma frase** que el `<summary>` por omisión de la ficha: son dos
// sitios diciendo lo mismo, o sea la séptima vez que una regla en dos listas se puede desalinear
// aquí. Se comparan.
func TestElResumenDelPaqueteEsElDeLaFicha(t *testing.T) {
	nfpm, err := os.ReadFile("../../empaquetado/linux/nfpm.yaml")
	if err != nil {
		t.Fatalf("no he podido leer nfpm.yaml: %v", err)
	}
	m := regexp.MustCompile(`(?m)^description: \|\n {2}(.+)$`).FindSubmatch(nfpm)
	if m == nil {
		t.Fatal("no encuentro el bloque «description» de nfpm.yaml: o ha cambiado de forma o esta prueba no comprueba nada")
	}
	resumen := strings.TrimSpace(string(m[1]))

	// Pura ASCII: es lo que sobrevive a la conversión del instalador.
	for _, r := range resumen {
		if r > 127 {
			t.Errorf("el resumen del paquete lleva %q, y la ventana del instalador lo convierte en «?»: %q", r, resumen)
			break
		}
	}

	// La misma frase que la ficha.
	var deLaFicha string
	for _, r := range laFicha(t).Resumenes {
		if r.Idioma == "" {
			deLaFicha = strings.TrimSpace(r.Texto)
		}
	}
	if resumen != deLaFicha {
		t.Errorf("el paquete dice %q y la ficha de AppStream %q: son dos sitios diciendo lo mismo y se han desalineado", resumen, deLaFicha)
	}

	// Y lo que pide la política de Debian de esa línea (3.4.1): breve y sin el nombre del paquete.
	if len(resumen) >= 80 {
		t.Errorf("el resumen mide %d caracteres y la política pide menos de 80, que es lo que se ve solo: %q", len(resumen), resumen)
	}
	if strings.Contains(strings.ToLower(resumen), "esfinge") {
		t.Errorf("el resumen repite el nombre del paquete, que la política dice que no vaya ahí: %q", resumen)
	}
}
