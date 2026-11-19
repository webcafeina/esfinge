package adjuntos

import (
	"os"
	"path/filepath"
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

// TestCadaAdjuntoPublicadoTieneQuienLoFabrique: para cada ficha marcada como publicada,
// **su clave aparece en el flujo**.
//
// # Por qué la clave y no la plataforma
//
// La primera versión de esto comparaba las **plataformas** que compila `wails build` con
// las que el catálogo implica, y tenía dos agujeros que encontró mutarla, no leerla:
//
//   - marcar el `.rpm` como publicado **pasaba en verde**, porque un `.rpm` de amd64 vive
//     en una plataforma que ya se compilaba. Plataforma cubierta no es adjunto fabricado.
//   - y quitar la pata de Windows ARM de la matriz **también pasaba**, porque las
//     arquitecturas se recogían del fichero entero y la matriz de Linux declara las mismas.
//
// Desde la fase 2 el flujo **le pregunta el nombre al catálogo por su clave**
// (`catalogo nombre deb-${{ matrix.arq }}`), así que la clave es lo que de verdad dice que
// alguien lo fabrica. Y las arquitecturas se leen **por trabajo**.
func TestCadaAdjuntoPublicadoTieneQuienLoFabrique(t *testing.T) {
	yaml := leer(t, "../../.github/workflows/publicar.yml")
	normal := strings.ReplaceAll(yaml, "${{ matrix.arq }}", "ARQ")

	// Lo que nombra otro sitio, con su razón. La lista es corta a propósito: cada
	// excepción es un adjunto cuyo nombre no sale del catálogo, y eso es deuda.
	fuera := map[string]string{
		"app-darwin": "lo nombra empaquetado/macos/armar-dmg.sh, que es quien arma la imagen",
	}

	for _, f := range Catalogo() {
		if !f.Publicado {
			continue
		}
		switch f.Clase {
		case ClaseApp, ClasePaquete, ClaseMCPB:
		default:
			// La línea de comandos sale del Makefile, y la extensión y las sumas se
			// nombran en su propio paso: no pasan por clave.
			continue
		}
		if _, hay := fuera[f.Clave]; hay {
			continue
		}
		conArq := f.Clave
		if f.Arq != "" {
			conArq = strings.ReplaceAll(f.Clave, f.Arq, "ARQ")
		}
		if strings.Contains(normal, f.Clave) || strings.Contains(normal, conArq) {
			continue
		}
		t.Errorf("«%s» está marcado como publicado y el flujo no lo nombra: nadie lo fabrica, "+
			"así que la publicación saldría sin él y la puerta la pararía", f.Clave)
	}
}

// TestLasArquitecturasDeCadaTrabajoCubrenSuPlataforma comprueba, **trabajo por trabajo**,
// que la matriz declara las arquitecturas que ese trabajo compila.
//
// Leer las `arq:` del fichero entero no vale: la matriz de Linux declara las mismas que la
// de Windows, así que quitarle una pata a una quedaba tapado por la otra. Lo dijo mutarla.
func TestLasArquitecturasDeCadaTrabajoCubrenSuPlataforma(t *testing.T) {
	yaml := leer(t, "../../.github/workflows/publicar.yml")

	// Los trabajos empiezan con dos espacios y acaban donde empieza el siguiente.
	cabecera := regexp.MustCompile(`(?m)^  ([a-z-]+):$`)
	sitios := cabecera.FindAllStringSubmatchIndex(yaml, -1)
	vistos := 0
	for i, s := range sitios {
		nombre := yaml[s[2]:s[3]]
		fin := len(yaml)
		if i+1 < len(sitios) {
			fin = sitios[i+1][0]
		}
		trozo := yaml[s[0]:fin]
		plats := regexp.MustCompile(`wails build -platform (\S+)`).FindAllStringSubmatch(trozo, -1)
		if len(plats) == 0 {
			continue
		}
		for _, pl := range plats {
			if !strings.Contains(pl[1], "matrix.arq") && !strings.Contains(pl[1], "${{") {
				continue // una plataforma fija, como darwin/universal
			}
			arqs := regexp.MustCompile(`(?m)^\s*-?\s*arq:\s*(\S+)`).FindAllStringSubmatch(trozo, -1)
			if len(arqs) < 2 {
				t.Errorf("el trabajo «%s» compila una plataforma de la matriz y declara %d "+
					"arquitecturas: le falta alguna pata", nombre, len(arqs))
			}
			vistos++
		}
	}
	if vistos == 0 {
		t.Fatal("ningún trabajo compila por matriz: la prueba no está mirando lo que cree")
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

// TestLosDosPostinstallSoloSeDiferencianEnElGestor.
//
// Cada empaquetador lleva su propio guion de postinstalación porque ese fichero se copia
// dentro del paquete y se ejecuta **en el equipo de quien instala**: una variable de la
// máquina que lo arma no llegaría allí, y adivinar el gestor en el equipo del cliente
// —mirando si hay `rpm` o `dpkg`— es la suposición que todo esto vino a quitar.
//
// El precio es **dos ficheros casi iguales**, y en esta casa eso ya ha costado cinco
// fallos: el día que alguien añada un paso al de Debian, el de Fedora se queda sin él y
// nadie se entera. No se pueden juntar —el fichero compartido no estaría instalado—, así
// que toca lo que manda aquí cuando dos listas no se pueden juntar: **compararlas**, y
// exigir que la única diferencia sea la línea del gestor.
func TestLosDosPostinstallSoloSeDiferencianEnElGestor(t *testing.T) {
	deb := strings.Split(leer(t, "../../empaquetado/linux/postinstall.sh"), "\n")
	rpm := strings.Split(leer(t, "../../empaquetado/linux/postinstall-rpm.sh"), "\n")
	if len(deb) != len(rpm) {
		t.Fatalf("los dos guiones no tienen el mismo número de líneas (%d y %d): alguien añadió "+
			"un paso a uno y no al otro", len(deb), len(rpm))
	}
	distintas := 0
	for i := range deb {
		if deb[i] == rpm[i] {
			continue
		}
		distintas++
		// La única diferencia que vale es cuál gestor se escribe.
		if !strings.Contains(deb[i], "/usr/share/esfinge/gestor") ||
			!strings.Contains(rpm[i], "/usr/share/esfinge/gestor") {
			t.Errorf("línea %d: los dos guiones se diferencian en algo que no es el gestor.\n"+
				"  deb: %s\n  rpm: %s", i+1, deb[i], rpm[i])
		}
	}
	if distintas != 1 {
		t.Errorf("se diferencian en %d líneas y tiene que ser exactamente una: la del gestor", distintas)
	}
	// Y que cada uno escriba el suyo, que es lo único que los distingue.
	if !strings.Contains(strings.Join(deb, "\n"), "printf 'deb\\n'") {
		t.Error("el postinstall del .deb no deja escrito «deb»")
	}
	if !strings.Contains(strings.Join(rpm, "\n"), "printf 'rpm\\n'") {
		t.Error("el postinstall del .rpm no deja escrito «rpm»")
	}
}

// TestLosCanalesCuelganDeLaPublicacionYNoDelEvento vigila la trampa que el diseño de la fase 5 cazó
// antes de escribir una línea, y que es de las que se desandan solas.
//
// **`on: release: published` no se dispara cuando la publicación la crea `GITHUB_TOKEN`.** Es regla
// de GitHub, sin excepciones. Apolo cuelga sus canales de ese evento y le funciona porque su
// publicación sale **como borrador** y la publica una persona; la de Esfinge la crea
// `softprops/action-gh-release` ya publicada. Copiar sus flujos tal cual —que es lo natural, están
// al lado y funcionan allí— daría flujos que **no se ejecutan nunca, en silencio**, con el tap y el
// AUR sirviendo una versión de hace meses y ningún correo rojo que lo diga.
//
// Así que aquí se afirman dos cosas: que **ningún** flujo cuelga de ese evento, y que los dos que
// publican hacia fuera —la web con sus repositorios, y los canales— **los llama `publicar.yml`**.
// La segunda mitad es la del método del puente que no llama nadie: un flujo con `workflow_call` y
// sin quien lo llame parece terminado y no se ejecuta jamás.
func TestLosCanalesCuelganDeLaPublicacionYNoDelEvento(t *testing.T) {
	flujos, err := filepath.Glob("../../.github/workflows/*.yml")
	if err != nil || len(flujos) == 0 {
		t.Fatalf("no encuentro los flujos: %v", err)
	}

	for _, f := range flujos {
		crudo, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		texto := string(crudo)
		// Se busca la línea del disparador, no la palabra: los comentarios de estos ficheros
		// explican precisamente por qué **no** se usa, y señalar eso haría que el aviso se dejara
		// de leer —que es como se pierde un vigilante—.
		for _, l := range strings.Split(texto, "\n") {
			limpia := strings.TrimSpace(l)
			if strings.HasPrefix(limpia, "#") {
				continue
			}
			if limpia == "release:" {
				t.Errorf("%s cuelga de «on: release», que no se dispara cuando la publicación la crea GITHUB_TOKEN: eso deja el canal sin actualizarse y sin decirlo", filepath.Base(f))
			}
		}
	}

	publicar, err := os.ReadFile("../../.github/workflows/publicar.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, quien := range []string{"./.github/workflows/web.yml", "./.github/workflows/canales.yml"} {
		if !strings.Contains(string(publicar), "uses: "+quien) {
			t.Errorf("publicar.yml no llama a %s: un flujo con «workflow_call» y sin quien lo llame no se ejecuta nunca", quien)
		}
	}
	// Y que lo llame **después** de publicar, no a la vez: los paquetes los baja de la
	// publicación, así que antes no existen.
	if !strings.Contains(string(publicar), "needs: publicar") {
		t.Error("publicar.yml llama a los flujos de fuera sin esperar a que la publicación exista")
	}
}

// TestTodoTrabajoQueLeeDelRepositorioHaceCheckout, que es de las que nacen de una publicación
// caída.
//
// Al trabajo de humo se le añadió el cheque de la firma del `.rpm`, que monta en el contenedor la
// clave pública del repositorio — y **ese trabajo no hacía `checkout`**: solo bajaba artefactos, y
// nunca le había hecho falta. **Docker no avisa de eso**: cuando la ruta del anfitrión no existe,
// crea un directorio vacío y lo monta igual, así que dentro había una carpeta donde se esperaba una
// clave y lo que salió fue «import read failed(1)», que no se parece a la causa. Tiró la 2.66.0.
//
// Así que aquí se afirma lo que se puede afirmar leyendo los flujos: **si un trabajo nombra un
// fichero del repositorio, tiene que traerse el repositorio**.
func TestTodoTrabajoQueLeeDelRepositorioHaceCheckout(t *testing.T) {
	// Las carpetas del repositorio que un paso puede querer leer. No se buscan rutas en general
	// —«paquetes/» o «_site/» las crea el propio flujo—, solo lo que existe en git.
	carpetas := regexp.MustCompile(`(?:\./)?(empaquetado|herramientas|internal|build|frontend|navegador|web|cmd|docs)/[A-Za-z0-9._/-]+`)

	flujos, err := filepath.Glob("../../.github/workflows/*.yml")
	if err != nil || len(flujos) == 0 {
		t.Fatalf("no encuentro los flujos: %v", err)
	}
	mirados := 0
	for _, f := range flujos {
		crudo, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		// Se parte por trabajo con el propio texto: los flujos llevan comentarios largos entre
		// medias y lo que importa es qué nombra cada bloque.
		for _, bloque := range enTrabajos(string(crudo)) {
			if bloque.pasos == "" {
				continue
			}
			nombradas := map[string]bool{}
			for _, l := range strings.Split(bloque.pasos, "\n") {
				limpia := strings.TrimSpace(l)
				if strings.HasPrefix(limpia, "#") {
					continue
				}
				for _, m := range carpetas.FindAllString(limpia, -1) {
					nombradas[m] = true
				}
			}
			if len(nombradas) == 0 {
				continue
			}
			mirados++
			if !strings.Contains(bloque.pasos, "actions/checkout") && !strings.Contains(bloque.pasos, "uses: ./") {
				var cuales []string
				for n := range nombradas {
					cuales = append(cuales, n)
				}
				sort.Strings(cuales)
				t.Errorf("%s · el trabajo «%s» nombra %v y no hace checkout: Docker montaría un directorio vacío ahí sin decir nada", filepath.Base(f), bloque.nombre, cuales)
			}
		}
	}
	if mirados == 0 {
		t.Fatal("no he encontrado ningún trabajo que lea del repositorio: esta prueba no está comprobando nada")
	}
}

type trabajoDelFlujo struct {
	nombre string
	pasos  string
}

// enTrabajos parte un flujo en sus trabajos por la sangría, que es lo único que los separa sin
// interpretar el YAML entero —y hace falta el texto crudo, porque los comentarios cuentan—.
func enTrabajos(texto string) []trabajoDelFlujo {
	var out []trabajoDelFlujo
	dentro := false
	actual := trabajoDelFlujo{}
	for _, l := range strings.Split(texto, "\n") {
		if strings.HasPrefix(l, "jobs:") {
			dentro = true
			continue
		}
		if !dentro {
			continue
		}
		// Un trabajo empieza con exactamente dos espacios y acaba en «:».
		if m := regexp.MustCompile(`^  ([a-zA-Z0-9_-]+):\s*$`).FindStringSubmatch(l); m != nil {
			if actual.nombre != "" {
				out = append(out, actual)
			}
			actual = trabajoDelFlujo{nombre: m[1]}
			continue
		}
		if actual.nombre != "" {
			actual.pasos += l + "\n"
		}
	}
	if actual.nombre != "" {
		out = append(out, actual)
	}
	return out
}

// TestCadaPasoQueUnTrabajoNombraExiste, que nace de la publicación 2.66.2.
//
// **En GitHub, una expresión que nombra algo que no existe no da error: vale cadena vacía.** La
// salida del trabajo que publica se escribió como `${{ steps.v.outputs.n }}`, copiando la expresión
// de los otros trabajos, y ahí **no había ningún paso con `id: v`**. Nada se puso rojo: los canales
// recibieron una versión vacía y se cayeron al pedirle a la publicación la etiqueta «v», que es un
// error que no habla de lo que pasa.
//
// Es la familia de «comparar las dos puntas no prueba que estén unidas», con la vuelta de que aquí
// una punta puede no existir y el lenguaje se calla. Así que esto afirma lo único que lo cierra: si
// un trabajo nombra `steps.X`, **ese paso tiene que estar en ese trabajo**.
func TestCadaPasoQueUnTrabajoNombraExiste(t *testing.T) {
	nombrado := regexp.MustCompile(`steps\.([A-Za-z0-9_-]+)\.`)

	flujos, err := filepath.Glob("../../.github/workflows/*.yml")
	if err != nil || len(flujos) == 0 {
		t.Fatalf("no encuentro los flujos: %v", err)
	}
	mirados := 0
	for _, f := range flujos {
		crudo, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, bloque := range enTrabajos(string(crudo)) {
			// Los `id:` que ese trabajo declara.
			hay := map[string]bool{}
			for _, m := range regexp.MustCompile(`(?m)^\s*-?\s*id:\s*([A-Za-z0-9_-]+)\s*$`).FindAllStringSubmatch(bloque.pasos, -1) {
				hay[m[1]] = true
			}
			for _, l := range strings.Split(bloque.pasos, "\n") {
				if strings.HasPrefix(strings.TrimSpace(l), "#") {
					continue
				}
				for _, m := range nombrado.FindAllStringSubmatch(l, -1) {
					mirados++
					if !hay[m[1]] {
						t.Errorf("%s · el trabajo «%s» nombra «steps.%s» y no tiene ningún paso con ese id: en GitHub eso no falla, vale cadena vacía",
							filepath.Base(f), bloque.nombre, m[1])
					}
				}
			}
		}
	}
	if mirados == 0 {
		t.Fatal("no he encontrado ninguna referencia a «steps.»: esta prueba no está comprobando nada")
	}
}

// Los canales se pueden tocar **de uno en uno**, y la lista de los que se pueden elegir tiene que
// ser exactamente la de los que miran esa elección.
//
// # Qué pasa si se desalinean
//
// Las dos mitades fallan calladas y en direcciones contrarias. Un canal **que no mira el
// selector** se publica cuando se ha pedido otro: reempujar la ficha de Chocolatey devolvería el
// tap de Homebrew a esa versión vieja, que es justo lo que el selector viene a impedir. Y un canal
// **que no está en la lista** no se puede reempujar nunca: el día que su tienda devuelva algo, hay
// que bajar el paquete y empujarlo a mano desde otra máquina con la clave delante.
//
// Es la sexta vez en este proyecto que una regla vive en dos listas, así que van comparadas: la
// lista es la del `options:` y los canales son los trabajos que cuelgan de `recetas`, no una
// tercera lista escrita aquí que habría que acordarse de ampliar.
func TestLosCanalesQueSePuedenElegirSonLosQueMiranLaEleccion(t *testing.T) {
	yaml := leer(t, "../../.github/workflows/canales.yml")

	m := regexp.MustCompile(`options:\s*\[([^\]]+)\]`).FindStringSubmatch(yaml)
	if m == nil {
		t.Fatal("no encuentro el «options:» del canal en canales.yml, o cambió cómo se declara")
	}
	sePuedenElegir := map[string]bool{}
	for _, c := range strings.Split(m[1], ",") {
		c = strings.TrimSpace(c)
		if c != "" && c != "todos" {
			sePuedenElegir[c] = true
		}
	}
	if len(sePuedenElegir) == 0 {
		t.Fatal("la lista de canales que se pueden elegir está vacía: esta prueba no comprueba nada")
	}

	// Lo que cuelga de las recetas **y no publica**, con su razón. La lista es corta a propósito:
	// cada excepción es un trabajo que podría estar publicando sin que esto lo mirara.
	noPublican := map[string]string{
		"cargan": "es la puerta que carga las dos recetas en un Mac antes de empujarlas; " +
			"no escribe en ningún sitio",
	}

	// Los trabajos y su cuerpo, para preguntarle a cada uno por lo suyo.
	cabecera := regexp.MustCompile(`(?m)^  ([a-z-]+):$`)
	// Colgar de las recetas se escribe de dos formas —`needs: recetas` y `needs: [recetas, …]`—, y
	// la primera versión de esto sólo veía la primera: el día que al tap se le añadió una puerta
	// delante, dejó de contar como canal y la prueba lo dijo por el otro lado.
	deLasRecetas := regexp.MustCompile(`needs:\s*(recetas\b|\[[^\]]*\brecetas\b)`)
	sitios := cabecera.FindAllStringSubmatchIndex(yaml, -1)
	canales := map[string]bool{}
	hay := map[string]bool{}
	for i, s := range sitios {
		nombre := yaml[s[2]:s[3]]
		fin := len(yaml)
		if i+1 < len(sitios) {
			fin = sitios[i+1][0]
		}
		cuerpo := yaml[s[0]:fin]
		hay[nombre] = true

		if !deLasRecetas.MatchString(cuerpo) {
			continue
		}
		if _, exento := noPublican[nombre]; exento {
			continue
		}
		canales[nombre] = true
		if !strings.Contains(cuerpo, "inputs.canal == '"+nombre+"'") {
			t.Errorf("el trabajo «%s» publica en un canal y no mira cuál se ha pedido: "+
				"reempujar otro canal lo publicaría también, con la versión que se esté "+
				"reempujando", nombre)
		}
		if !sePuedenElegir[nombre] {
			t.Errorf("el trabajo «%s» publica en un canal y no está en el «options:», "+
				"así que no se puede reempujar sin bajar el paquete y hacerlo a mano", nombre)
		}
	}

	for c := range sePuedenElegir {
		if !canales[c] {
			t.Errorf("se puede elegir el canal «%s» y no hay ningún trabajo que lo publique: "+
				"elegirlo no haría nada y parecería que sí", c)
		}
	}
	for n := range noPublican {
		if !hay[n] {
			t.Errorf("la lista de exentos nombra «%s», que ya no es un trabajo: un nombre muerto "+
				"ahí es donde mañana se esconde un canal de verdad", n)
		}
	}
}

// **El tap no se empuja sin que las dos recetas carguen en un Mac.**
//
// La puerta existe porque el 2026-11-19 dos fallos del cask llegaron hasta el Mac del cliente —un
// `depends_on` que Homebrew ha desactivado y un `#{appdir}` que ese DSL no tiene—, y los dos
// pasaron por aquí en verde: nadie evaluaba el cask nunca. Una puerta que corre **al lado** del
// empujón en vez de delante no protege nada, y quitarle la dependencia es una línea que no se nota
// — se comprobó, y sin esta prueba se quedaba en verde.
func TestElTapNoSeEmpujaSinQueLasRecetasCarguen(t *testing.T) {
	yaml := leer(t, "../../.github/workflows/canales.yml")

	cabecera := regexp.MustCompile(`(?m)^  ([a-z-]+):$`)
	sitios := cabecera.FindAllStringSubmatchIndex(yaml, -1)
	cuerpos := map[string]string{}
	for i, s := range sitios {
		fin := len(yaml)
		if i+1 < len(sitios) {
			fin = sitios[i+1][0]
		}
		cuerpos[yaml[s[2]:s[3]]] = yaml[s[0]:fin]
	}

	puerta, hay := cuerpos["cargan"]
	if !hay {
		t.Fatal("ya no hay trabajo «cargan» en canales.yml: si la puerta se ha quitado o se llama " +
			"de otra forma, esta prueba y su razón hay que mirarlas a mano, no borrarlas")
	}
	// Que de verdad cargue las recetas, y no sea un trabajo vacío con ese nombre.
	if !strings.Contains(puerta, "brew info") {
		t.Error("el trabajo «cargan» no llama a «brew info»: entonces no está cargando nada")
	}
	if !strings.Contains(puerta, "macos") {
		t.Error("el trabajo «cargan» no corre en un macOS, y «brew info --cask» no se puede " +
			"contestar en Linux")
	}

	tap := cuerpos["homebrew"]
	if !regexp.MustCompile(`needs:\s*\[[^\]]*\bcargan\b`).MatchString(tap) {
		t.Error("el trabajo que empuja el tap no depende de «cargan», así que las recetas se " +
			"publican mientras se comprueban: un cask que no carga llegaría al tap igual, y ahí " +
			"tumba también la fórmula")
	}
}
