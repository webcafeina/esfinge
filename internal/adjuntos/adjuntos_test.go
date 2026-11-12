package adjuntos

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestNingunSufijoEsSufijoDeOtro es **la prueba estructural del fallo que esto viene a
// cerrar**.
//
// El que elige la descarga busca por sufijo —para que una copia ya instalada encuentre su
// fichero aunque el nombre cambie de versión—, y eso solo es seguro si dos adjuntos que se
// le pueden ofrecer al mismo equipo no se confunden. El fallo que había era exactamente
// esto: la lista llevaba `".deb"` a secas detrás de `"_arm64.deb"`, y `".deb"` es sufijo de
// `esfinge_2.61.3_amd64.deb`, así que una máquina ARM se bajaba el paquete de amd64.
//
// Se mira solo entre los que `ParaSistema` puede devolver —la aplicación y los paquetes—
// porque **es ahí donde se busca por sufijo**. Los dieciocho de la línea de comandos
// comparten sufijo a propósito (`esfinge-{v}-linux-amd64` y
// `esfinge-puente-{v}-linux-amd64` acaban igual): a ésos se los distingue por delante y
// nadie los elige por el final.
func TestNingunSufijoEsSufijoDeOtro(t *testing.T) {
	var puedenOfrecerse []Ficha
	for _, f := range Catalogo() {
		if f.Clase == ClaseApp || f.Clase == ClasePaquete {
			puedenOfrecerse = append(puedenOfrecerse, f)
		}
	}
	if len(puedenOfrecerse) < 10 {
		t.Fatalf("solo hay %d adjuntos ofrecibles: la prueba no está mirando lo que cree", len(puedenOfrecerse))
	}
	for _, a := range puedenOfrecerse {
		for _, b := range puedenOfrecerse {
			if a.Clave == b.Clave {
				continue
			}
			if strings.HasSuffix(a.Sufijo(), b.Sufijo()) {
				t.Errorf("el sufijo de «%s» (%s) acaba en el de «%s» (%s): a un equipo se le "+
					"puede dar uno creyendo que le da el otro",
					a.Clave, a.Sufijo(), b.Clave, b.Sufijo())
			}
		}
	}
}

// TestCadaFichaEstaEnteraYEsUnica: una clave repetida o una plantilla sin versión dentro
// son errores que no dan la cara hasta que alguien pide ese nombre.
func TestCadaFichaEstaEnteraYEsUnica(t *testing.T) {
	claves := map[string]bool{}
	nombres := map[string]string{}
	for _, f := range Catalogo() {
		if f.Clave == "" || f.Plantilla == "" {
			t.Errorf("una ficha sin clave o sin plantilla: %+v", f)
			continue
		}
		if claves[f.Clave] {
			t.Errorf("la clave «%s» está dos veces", f.Clave)
		}
		claves[f.Clave] = true

		n := f.Nombre("9.9.9")
		if otra, ya := nombres[n]; ya {
			t.Errorf("«%s» y «%s» se llaman igual: %s", f.Clave, otra, n)
		}
		nombres[n] = f.Clave

		// Todo lleva la versión dentro menos las sumas, que se llaman igual siempre
		// porque el que las lee las busca por nombre exacto.
		if f.Clase != ClaseSumas && !strings.Contains(f.Plantilla, "{v}") {
			t.Errorf("«%s» no lleva la versión en el nombre: %s", f.Clave, f.Plantilla)
		}
		// Un paquete sin gestor es el tar.gz, y es el único que vale.
		if f.Clase == ClasePaquete && f.Gestor == SinGestor && !strings.Contains(f.Plantilla, "tar.gz") {
			t.Errorf("«%s» es un paquete y no dice con qué se instala", f.Clave)
		}
	}
}

// TestLaTablaDelLeemeSaleDelCatalogo compara el README con lo que genera el catálogo.
//
// Las dos tablas de descargas —la del README y la del cuerpo de la publicación— estaban
// escritas a mano, en dos ficheros distintos, con `amd64` cableado en las dos. Una tabla
// desalineada **no da error**: da un enlace que contesta 404, y lo descubre quien va a
// descargar.
func TestLaTablaDelLeemeSaleDelCatalogo(t *testing.T) {
	crudo, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	leeme := string(crudo)

	// La versión se saca del propio README: la escribe el flujo al publicar, así que es
	// la única que puede estar ahí.
	m := regexp.MustCompile(`Versión \*\*([0-9]+\.[0-9]+\.[0-9]+)\*\*`).FindStringSubmatch(leeme)
	if m == nil {
		t.Fatal("el README no dice de qué versión es su tabla")
	}
	version := m[1]

	entre := regexp.MustCompile(`(?s)<!-- descargas:inicio.*?<!-- descargas:fin -->`)
	hay := entre.FindString(leeme)
	if hay == "" {
		t.Fatal("el README no tiene las marcas de la tabla")
	}
	quiere := strings.TrimSpace(TablaDelLeeme(version))
	if strings.TrimSpace(hay) != quiere {
		t.Errorf("la tabla del README no es la que sale del catálogo para la %s.\n"+
			"Si el cambio es a propósito, se cambia el catálogo y se corre "+
			"«herramientas/actualizar-descargas.sh %s».\n\n--- el README dice ---\n%s\n"+
			"\n--- el catálogo dice ---\n%s", version, version, hay, quiere)
	}
}

// TestLasDosTablasDicenLosMismosFicheros, **sobre un mundo de mentira donde ARM ya está
// publicado**.
//
// Con el catálogo de hoy esta prueba no podría fallar: no hay ni un adjunto de arm64 con
// `Publicado`, así que filtrarlos no cambiaría nada y la mutación pasaba en verde. Lo
// comprobé y por eso las dos tablas reciben la lista por parámetro. Es la lección de
// `soloLectura`: el caso que importa no se puede construir con las constantes de hoy.
func TestLasDosTablasDicenLosMismosFicheros(t *testing.T) {
	const v = "3.2.1"
	mundo := []Ficha{
		{Clave: "app-darwin", Clase: ClaseApp, Arq: "universal",
			Plantilla: "Esfinge-{v}.dmg", Publicado: true, Rotulo: "macOS", Nota: "Universal"},
		{Clave: "deb-amd64", Clase: ClasePaquete, Arq: "amd64", Gestor: GestorDeb,
			Plantilla: "esfinge_{v}_amd64.deb", Publicado: true, Rotulo: "Debian", Nota: "x"},
		{Clave: "deb-arm64", Clase: ClasePaquete, Arq: "arm64", Gestor: GestorDeb,
			Plantilla: "esfinge_{v}_arm64.deb", Publicado: true, Rotulo: "Debian ARM", Nota: "x"},
		{Clave: "mcpb-linux-arm64", Clase: ClaseMCPB, Arq: "arm64",
			Plantilla: "Esfinge-{v}-linux-arm64.mcpb", Publicado: true, Rotulo: "Linux ARM"},
		// Y uno que todavía no se fabrica: no puede salir en ninguna tabla.
		{Clave: "rpm-arm64", Clase: ClasePaquete, Arq: "arm64", Gestor: GestorRPM,
			Plantilla: "esfinge-{v}-1.aarch64.rpm", Rotulo: "Fedora ARM", Nota: "x"},
	}
	readme, cuerpo := tablaDelLeeme(mundo, v), cuerpoDeLaPublicacion(mundo, v)

	for _, f := range mundo {
		n := f.Nombre(v)
		enReadme, enCuerpo := strings.Contains(readme, n), strings.Contains(cuerpo, n)
		if f.Publicado {
			if !enReadme {
				t.Errorf("«%s» está publicado y no sale en la tabla del README", n)
			}
			if !enCuerpo {
				t.Errorf("«%s» está publicado y no sale en el cuerpo de la publicación", n)
			}
			continue
		}
		// Un enlace a un fichero que no existe es peor que no tener fila.
		if enReadme {
			t.Errorf("«%s» no está publicado y el README lo enlaza", n)
		}
		if enCuerpo {
			t.Errorf("«%s» no está publicado y el cuerpo de la publicación lo nombra", n)
		}
	}
}

// TestLaApiFalsaDelE2eUsaNombresDelCatalogo compara la lista de la API de mentira con el
// catálogo.
//
// Esa API hace de GitHub en el e2e para que el aviso de versión nueva recorra el camino
// entero, y **lleva los nombres escritos a mano**. Es una lista más, y no se puede juntar
// con la de Go porque vive en un `.mjs`: entonces toca lo que manda esta casa cuando dos
// listas no se pueden juntar, que es **compararlas entre sí**. Si el catálogo cambia un
// nombre y esa API se queda con el viejo, el e2e se cae diciendo algo que no se parece a
// la causa — y eso ya ha costado tardes aquí.
func TestLaApiFalsaDelE2eUsaNombresDelCatalogo(t *testing.T) {
	crudo, err := os.ReadFile("../../frontend/e2e/api-falsa.mjs")
	if err != nil {
		t.Fatal(err)
	}
	// La API escribe los nombres con la versión interpolada: `Esfinge-${VERSION}.dmg`.
	// Se buscan tal cual, con el marcador puesto, y se comparan con las plantillas.
	// El `[{\\s]` de delante deja fuera el `tag_name`, que es la etiqueta y no un fichero.
	nombres := regexp.MustCompile("[{\\s]name: `([^`]+)`").FindAllStringSubmatch(string(crudo), -1)
	if len(nombres) == 0 {
		t.Fatal("la API falsa no declara ningún adjunto, o cambió cómo los escribe")
	}
	plantillas := map[string]bool{}
	for _, f := range Catalogo() {
		plantillas[strings.ReplaceAll(f.Plantilla, "{v}", "${VERSION}")] = true
	}
	for _, m := range nombres {
		if !plantillas[m[1]] {
			t.Errorf("la API falsa del e2e sirve «%s», que no es ningún adjunto del catálogo", m[1])
		}
	}
}
