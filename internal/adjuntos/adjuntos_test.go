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

// TestLaTablaDelLeemeNoInventaNingunNombre comprueba cada enlace del README contra el
// catálogo: su nombre, y que la dirección sea la de esa versión.
//
// # Por qué no compara la tabla entera
//
// La primera versión de esto exigía que el bloque del README fuera **exactamente** lo que
// genera el catálogo para la versión que el propio README nombra. Suena más fuerte y es
// **falso por diseño**: el README describe la última versión **publicada** y el catálogo
// dice lo que se fabrica **ahora**, así que entre marcar un adjunto como publicado y
// publicarlo de verdad los dos difieren a propósito — y con la prueba fuerte eso bloquea
// la publicación que arreglaría la diferencia.
//
// Lo que sí es cierto siempre, y es lo que se afirma aquí: **ningún enlace de esa tabla
// puede nombrar algo que el catálogo no conozca**, ni apuntar a otra versión, ni a un
// adjunto que no sale en tablas. Eso caza lo que de verdad puede pasar: que alguien edite
// la tabla a mano, o que un nombre cambie en el catálogo y el README se quede con el viejo.
//
// Lo que esto **no** caza: que un adjunto nuevo tarde en aparecer en la tabla. De eso se
// encarga el propio flujo, que la reescribe en cada publicación con el único generador que
// hay. Está anotado en deuda.md: con una versión «desde» en cada ficha se podría volver a
// la comparación completa.
func TestLaTablaDelLeemeNoInventaNingunNombre(t *testing.T) {
	crudo, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	leeme := string(crudo)

	m := regexp.MustCompile(`Versión \*\*([0-9]+\.[0-9]+\.[0-9]+)\*\*`).FindStringSubmatch(leeme)
	if m == nil {
		t.Fatal("el README no dice de qué versión es su tabla")
	}
	version := m[1]

	entre := regexp.MustCompile(`(?s)<!-- descargas:inicio.*?<!-- descargas:fin -->`)
	bloque := entre.FindString(leeme)
	if bloque == "" {
		t.Fatal("el README no tiene las marcas de la tabla")
	}

	// Lo que el catálogo sabe nombrar para esa versión, y con qué rótulo.
	buenos := map[string]Ficha{}
	for _, f := range Catalogo() {
		buenos[f.Nombre(version)] = f
	}

	enlaces := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).FindAllStringSubmatch(bloque, -1)
	if len(enlaces) == 0 {
		t.Fatal("la tabla del README no tiene ni un enlace")
	}
	vistos := 0
	for _, e := range enlaces {
		nombre, url := e[1], e[2]
		f, esAdjunto := buenos[nombre]
		if !esAdjunto {
			// Los enlaces que no son ficheros —la página de publicaciones— se saltan.
			if strings.Contains(url, "/releases/latest") || strings.HasSuffix(url, "/releases") {
				continue
			}
			t.Errorf("la tabla enlaza «%s», que no es ningún adjunto del catálogo para la %s",
				nombre, version)
			continue
		}
		vistos++
		quiere := base(version) + "/" + nombre
		if url != quiere {
			t.Errorf("«%s» apunta a %s y le toca %s", nombre, url, quiere)
		}
		if f.Rotulo == "" {
			t.Errorf("la tabla enlaza «%s», que es un adjunto que no sale en tablas", nombre)
		}
	}
	if vistos == 0 {
		t.Error("ningún enlace de la tabla era un adjunto: la prueba no está mirando lo que cree")
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

// TestLosTresBinariosSeIgnoran compara la lista de `.gitignore` con la del catálogo.
//
// # Por qué existe
//
// `.gitignore` nombraba **dos** de los tres binarios que se compilan en la raíz:
// `esfinge-mcp` faltaba. Mientras nada los dejara escritos ahí eso no se notaba, y el día
// que un contenedor de pruebas con el repositorio montado con escritura los dejó, un
// `git add -A` **se llevó `esfinge-mcp` al repositorio** — y la compilación de la
// publicación murió con «build output "esfinge-mcp" already exists and is not an object
// file», tirando la 2.65.0 con todo lo demás en verde.
//
// Es la familia de siempre en esta casa: **una regla escrita en dos listas se desalinea
// sola**, y ya ha costado cinco fallos. Lo que la cierra cuando las dos listas no se pueden
// juntar —un `.gitignore` no puede preguntarle nada a Go— es **una prueba que las compara
// entre sí**, que es lo mismo que se hizo con los botones de la tarjeta y con los dos
// `postinstall`.
//
// Y pide **las dos formas**: sin `.exe`, lo que compila `make publicar` para Windows en la
// raíz tampoco se ignoraría.
func TestLosTresBinariosSeIgnoran(t *testing.T) {
	crudo, err := os.ReadFile("../../.gitignore")
	if err != nil {
		t.Fatal(err)
	}
	var reglas []string
	for _, l := range strings.Split(string(crudo), "\n") {
		reglas = append(reglas, strings.TrimSpace(l))
	}
	tiene := func(r string) bool {
		for _, l := range reglas {
			if l == r {
				return true
			}
		}
		return false
	}

	if len(losBinarios) == 0 {
		t.Fatal("el catálogo no dice qué binarios hay, así que esto no comprueba nada")
	}
	for _, b := range losBinarios {
		for _, quiero := range []string{"/" + b, "/" + b + ".exe"} {
			if !tiene(quiero) {
				t.Errorf("«%s» no está en .gitignore: el día que algo lo deje escrito en la raíz, un «git add -A» se lo lleva al repositorio y la compilación de la publicación se cae", quiero)
			}
		}
	}
}
