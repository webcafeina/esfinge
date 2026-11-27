package tema

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLosTokensEstanAlDia compara el fichero que consume la interfaz con lo que
// dice Go ahora mismo.
//
// Sin esto, cambiar un color en Go y olvidarse de regenerar deja la interfaz
// pintando los colores de ayer, y encima los tests de contraste seguirían en
// verde porque miden el tema, no el fichero. Aquí es donde se nota.
func TestLosTokensEstanAlDia(t *testing.T) {
	if len(Ficheros) == 0 {
		t.Fatal("no hay ningún fichero de tokens declarado, así que esto no comprueba nada")
	}
	for _, f := range Ficheros {
		ruta := filepath.Join("..", "..", filepath.FromSlash(f.Ruta))
		enDisco, err := os.ReadFile(ruta)
		if err != nil {
			t.Skipf("todavía no hay quien consuma %s: %v", f.Ruta, err)
		}
		if string(enDisco) != f.CSS() {
			t.Errorf("%s no coincide con lo que genera Go. Ejecuta «make tokens».", f.Ruta)
		}
	}
}

// Cada campo del tema tiene que publicarse: añadir un color y no publicarlo lo
// deja inaccesible para la interfaz sin que nadie se entere.
func TestSePublicanTodosLosColores(t *testing.T) {
	css := GenerarCSS()
	for _, c := range campos {
		if !strings.Contains(css, "--"+c.nombre+":") {
			t.Errorf("el color %q no sale en los tokens", c.nombre)
		}
	}

	// Y los dos temas están, cada uno donde le toca.
	for _, quiero := range []string{
		":root {",
		"@media (prefers-color-scheme: dark)",
		`:root[data-tema="oscuro"]`,
		`:root:not([data-tema="claro"])`,
	} {
		if !strings.Contains(css, quiero) {
			t.Errorf("falta %q en los tokens", quiero)
		}
	}
}

// Ningún color puede definirse solo dentro de la consulta de medios: ahí, quien
// no la soporte se queda sin color, y el tema claro tiene que valer de base.
func TestElTemaClaroEsLaBase(t *testing.T) {
	css := GenerarCSS()
	base := css[strings.Index(css, ":root {"):strings.Index(css, "@media")]

	for _, c := range campos {
		if !strings.Contains(base, "--"+c.nombre+":") {
			t.Errorf("el color %q no está definido en la base", c.nombre)
		}
	}
}

// TestNingunValorOscuroSueltoEnLasHojas lee las hojas escritas a mano y exige
// que un selector de tema oscuro viva dentro de su media query.
//
// El reparto correcto lo escribe `GenerarCSS` y son tres reglas: el valor claro
// en `:root…`, el oscuro dentro de `@media (prefers-color-scheme: dark)` con
// `:not([data-tema="claro"])`, y el oscuro otra vez en `[data-tema="oscuro"]`
// para cuando se fuerza. Las hojas de cada sistema escriben alguna variable a
// mano —las que son medida y no color de la paleta— y ahí es fácil copiar el
// selector del segundo bloque **y olvidarse de la media query**.
//
// Eso fue exactamente la 2.70.0: `--borde-bajo` acabó valiendo blanco al 9 % en
// tema claro, o sea invisible sobre fondo blanco, y los campos y los botones de
// Windows se veían abiertos por abajo. **Lo encontró una captura del cliente**,
// no una prueba: un borde invisible no recorta, no desborda y no toca nada, que
// son las tres cosas que se midieron buscándolo.
func TestNingunValorOscuroSueltoEnLasHojas(t *testing.T) {
	hojas, err := filepath.Glob(filepath.Join("..", "..", "frontend", "src", "sistema-*.css"))
	if err != nil || len(hojas) == 0 {
		t.Fatalf("no se encontraron las hojas de sistema: %v", err)
	}

	for _, hoja := range hojas {
		fuente, err := os.ReadFile(hoja)
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", hoja, err)
		}

		// Se recorre contando llaves, que es lo único que dice si una regla está
		// dentro de una media query o al lado de ella. Un `strings.Contains` no
		// distingue las dos, que es justo el fallo que esto vigila.
		profundidadMedia := -1
		nivel := 0
		for n, linea := range strings.Split(string(fuente), "\n") {
			limpia := strings.TrimSpace(linea)
			if strings.HasPrefix(limpia, "@media") && strings.Contains(limpia, "prefers-color-scheme: dark") {
				profundidadMedia = nivel
			}
			if strings.Contains(limpia, `:not([data-tema="claro"])`) && profundidadMedia < 0 {
				t.Errorf("%s:%d · un selector de tema oscuro fuera de su media query:\n  %s\n"+
					"Eso aplica el valor oscuro **también en claro**. El reparto es el de\n"+
					"`GenerarCSS`: claro en «:root…», oscuro dentro de «@media (prefers-color-scheme:\n"+
					"dark)» y oscuro otra vez en «[data-tema=\"oscuro\"]».",
					filepath.Base(hoja), n+1, limpia)
			}
			nivel += strings.Count(linea, "{") - strings.Count(linea, "}")
			if profundidadMedia >= 0 && nivel <= profundidadMedia {
				profundidadMedia = -1
			}
		}
	}
}

// TestLosSistemasDeLasHojasExisten compara **los nombres que usan las hojas de
// estilo con los que Go puede emitir**, que es la única forma de cazar una hoja
// entera que no se aplica nunca.
//
// # Por qué hace falta
//
// El nombre del sistema es `runtime.GOOS`, así que macOS es **`darwin`** y no
// «macos». La hoja de macOS se escribió entera —59 reglas— colgando de
// `[data-sistema="macos"]`, un valor que no existe: `plataforma.ts` sólo acepta
// `darwin`, `windows` y `linux`, y a lo demás le pone el sistema de verdad. El
// resultado es la peor forma de fallar que hay: **todo compila, todo pasa, y las
// capturas que se miran para decidir el aspecto son las de otro sistema** — en la
// primera tanda salió GNOME, con su título centrado y su hueco de 12, y pasó por
// macOS hasta que alguien midió.
//
// Ni el navegador ni el compilador dicen nada de un selector de atributo que no
// casa con nada: es CSS perfectamente válido. Lo único que lo caza es comparar
// las dos listas, que es la regla de la casa para lo que vive en dos sitios.
//
// `linux` entra aunque hoy no tenga hoja propia: lo usa el bloque por sistema de
// `estilos.css` y lo tendrá en su fase.
func TestLosSistemasDeLasHojasExisten(t *testing.T) {
	// Los que Go puede devolver en `Plataforma`, que son los `GOOS` de los tres
	// objetivos con ventana. Si algún día hay un cuarto, se añade aquí y la hoja
	// que lo use deja de estar sola.
	validos := map[string]bool{"darwin": true, "windows": true, "linux": true}

	hojas, err := filepath.Glob(filepath.Join("..", "..", "frontend", "src", "*.css"))
	if err != nil || len(hojas) == 0 {
		t.Fatalf("no se encontraron las hojas: %v", err)
	}

	re := regexp.MustCompile(`data-sistema="([^"]*)"`)
	vistos := map[string]bool{}
	for _, hoja := range hojas {
		fuente, err := os.ReadFile(hoja)
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", hoja, err)
		}
		for _, m := range re.FindAllStringSubmatch(string(fuente), -1) {
			if !validos[m[1]] {
				t.Errorf("%s usa data-sistema=%q, que Go no emite nunca: la hoja no se aplica a nadie. "+
					"Los que valen son darwin, windows y linux (son los GOOS, así que macOS es «darwin»)",
					filepath.Base(hoja), m[1])
			}
			vistos[m[1]] = true
		}
	}

	// Y la otra dirección: un juego de tokens para un sistema sin una sola regla
	// que lo use es una paleta que no pinta nada.
	for _, j := range juegos() {
		if j.selector == "" {
			continue
		}
		m := re.FindStringSubmatch(j.selector)
		if m == nil {
			t.Fatalf("el selector %q de juegos() no nombra ningún sistema", j.selector)
		}
		if !validos[m[1]] {
			t.Errorf("juegos() emite tokens para %q, que Go no devuelve nunca: esos tokens no los "+
				"lee nadie. Los que valen son darwin, windows y linux", m[1])
		}
		if !vistos[m[1]] {
			t.Errorf("juegos() emite tokens para %q y ninguna hoja de estilo usa ese sistema: "+
				"o falta la hoja, o el nombre no coincide", m[1])
		}
	}
}

// TestLasHojasNoInventanSelectores compara **los atributos que usan las hojas de
// sistema con los que el marcado escribe de verdad**.
//
// # Por qué hace falta
//
// Las hojas se escriben mirando la de Apolo, que es la misma especificación ya
// traducida — y **su marcado no es el nuestro**. Copiar un selector de allí no da
// ningún error: `[aria-checked="true"]` es CSS perfectamente válido que **no casa
// con nada**, así que la regla no se aplica, nada se pone rojo y lo que se ve es
// un control que no reacciona.
//
// Van tres en dos días, todas en la hoja de macOS y todas invisibles para el
// compilador:
//
//   - `[role="switch"]`, que aquí no existe: el interruptor es una casilla a
//     propósito, para no renombrar el control a los lectores de pantalla. El
//     estilo de la píldora no se aplicaba a nadie.
//   - `[aria-checked="true"]` en el segmentado, que usa `role="tab"` con
//     **`aria-selected`**: la opción elegida no se quedaba marcada en ninguna
//     pantalla, y lo vio el cliente.
//   - `[data-sistema="macos"]`, que vigila `TestLosSistemasDeLasHojasExisten`.
//
// Se comprueba el **par**: el nombre del atributo y su valor tienen que aparecer
// los dos en `frontend/src/`. Con el nombre solo no bastaría —`role` aparece
// siempre— y con el valor solo tampoco, que «true» está por todas partes.
func TestLasHojasNoInventanSelectores(t *testing.T) {
	hojas, err := filepath.Glob(filepath.Join("..", "..", "frontend", "src", "sistema-*.css"))
	if err != nil || len(hojas) == 0 {
		t.Fatalf("no se encontraron las hojas de sistema: %v", err)
	}

	// El marcado y lo que lo acompaña: los `.tsx` lo escriben y `plataforma.ts`
	// pone los atributos de sistema antes del primer pintado.
	var marcado strings.Builder
	fuentes, err := filepath.Glob(filepath.Join("..", "..", "frontend", "src", "*.ts*"))
	if err != nil || len(fuentes) == 0 {
		t.Fatalf("no se encontró el marcado: %v", err)
	}
	for _, f := range fuentes {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", f, err)
		}
		marcado.Write(b)
	}
	todo := marcado.String()

	// **`data-tema` se exime, y conviene decir por qué**: no lo escribe el marcado
	// sino `GenerarCSS`, que reparte cada color en tres bloques y usa el tercero
	// —`[data-tema="oscuro"]`— para cuando el tema se **fuerza**. Esfinge no ofrece
	// ese interruptor todavía, así que hoy ese bloque no lo activa nadie: es una
	// convención generada, no un selector inventado. Si algún día hay un selector de
	// tema en Ajustes, esta excepción deja de hacer falta y nadie tiene que acordarse
	// de nada.
	generados := map[string]bool{"data-tema": true}

	re := regexp.MustCompile(`\[((?:aria-|data-|role)[a-z-]*)="([^"]+)"\]`)
	for _, hoja := range hojas {
		fuente, err := os.ReadFile(hoja)
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", hoja, err)
		}
		for _, m := range re.FindAllStringSubmatch(string(fuente), -1) {
			atributo, valor := m[1], m[2]
			if generados[atributo] {
				continue
			}
			// **Y en camello también cuenta.** `plataforma.ts` los pone con
			// `dataset`, que escribe `data-barra` como `dataset.barra` y
			// `data-mi-cosa` como `dataset.miCosa`: sin esto, todo lo que no venga
			// de un `.tsx` saldría como inventado.
			if !strings.Contains(todo, atributo) && !strings.Contains(todo, enCamello(atributo)) {
				t.Errorf("%s usa [%s=%q] y **el marcado no escribe %s en ninguna parte**: "+
					"esa regla no se aplica a nadie", filepath.Base(hoja), atributo, valor, atributo)
				continue
			}
			if !strings.Contains(todo, valor) {
				t.Errorf("%s usa [%s=%q] y **el marcado no escribe nunca %q**: "+
					"esa regla no se aplica a nadie", filepath.Base(hoja), atributo, valor, valor)
			}
		}
	}
}

// enCamello traduce `data-mi-cosa` a `dataset.miCosa`, que es como lo escribe
// quien pone el atributo desde TypeScript.
func enCamello(atributo string) string {
	resto, hay := strings.CutPrefix(atributo, "data-")
	if !hay {
		return atributo
	}
	trozos := strings.Split(resto, "-")
	for i := 1; i < len(trozos); i++ {
		if trozos[i] != "" {
			trozos[i] = strings.ToUpper(trozos[i][:1]) + trozos[i][1:]
		}
	}
	return "dataset." + strings.Join(trozos, "")
}

// Ninguna hoja lee una variable que no escriba nadie.
//
// # Qué pasó
//
// La hoja de GNOME pintaba lo recorrido del deslizador con `var(--lleno, 0%)`, y
// **`--lleno` no lo escribe nadie en este proyecto**: así se llama en Apolo, de donde se copió el
// CSS. El tope del degradado era siempre el 0 % de la reserva, así que **la pista salía entera
// gris** — el deslizador perdía el oro, que es el único color propio en una pantalla de trabajo
// (ADR 0021). Lo vio el cliente en su Ubuntu con la 2.79.1.
//
// # Por qué no lo cazó nada
//
// Un `var()` con reserva **no falla**: pinta la reserva y sigue. Las pruebas de interfaz miden
// medidas y colores calculados, y aquí el color calculado del degradado era válido. Y las dos
// hojas que sí funcionan —Windows y macOS— leen `--recorrido`, así que compararlas entre ellas
// tampoco habría dicho nada: la que se desalineó fue la tercera.
//
// Es la regla de la casa de «una regla en dos listas», con el agravante de que **la segunda lista
// vive en otro proyecto**: al escribir la hoja de KDE se va a copiar de la misma fuente.
func TestLasHojasNoLeenVariablesQueNadieEscribe(t *testing.T) {
	raiz := filepath.Join("..", "..", "frontend", "src")

	hojas, err := filepath.Glob(filepath.Join(raiz, "sistema-*.css"))
	if err != nil || len(hojas) == 0 {
		t.Fatalf("no se encontraron las hojas de sistema: %v", err)
	}

	// Dónde se puede escribir una variable: en cualquier CSS de la interfaz —incluidos los
	// tokens que genera Go y la propia hoja— y en el marcado, que las pone como estilo en
	// línea (`"--recorrido": …`) o con `setProperty`.
	escritas := map[string]bool{}
	patrones := []string{filepath.Join(raiz, "*.css"), filepath.Join(raiz, "*.ts"), filepath.Join(raiz, "*.tsx")}
	declara := regexp.MustCompile(`--([a-z0-9-]+)\s*"?\s*[:,]`)
	for _, patron := range patrones {
		ficheros, err := filepath.Glob(patron)
		if err != nil {
			t.Fatalf("no se pudo listar %s: %v", patron, err)
		}
		for _, f := range ficheros {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("no se pudo leer %s: %v", f, err)
			}
			// Lo que está **dentro** de un `var(...)` es una lectura, no una escritura.
			sinLecturas := regexp.MustCompile(`var\(\s*--[a-z0-9-]+`).ReplaceAllString(string(b), "var(")
			for _, m := range declara.FindAllStringSubmatch(sinLecturas, -1) {
				escritas[m[1]] = true
			}
		}
	}
	if len(escritas) < 20 {
		t.Fatalf("sólo he encontrado %d variables escritas: o la interfaz ha cambiado de forma o esta prueba no comprueba nada", len(escritas))
	}

	lee := regexp.MustCompile(`var\(\s*--([a-z0-9-]+)`)
	fallos := 0
	for _, hoja := range hojas {
		b, err := os.ReadFile(hoja)
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", hoja, err)
		}
		vistas := map[string]bool{}
		for _, m := range lee.FindAllStringSubmatch(string(b), -1) {
			nombre := m[1]
			if escritas[nombre] || vistas[nombre] {
				continue
			}
			vistas[nombre] = true
			fallos++
			t.Errorf("%s lee «--%s» y no la escribe nadie: con reserva no falla, pinta la reserva y el control se queda sin su valor",
				filepath.Base(hoja), nombre)
		}
	}
	if fallos == 0 {
		t.Log("todas las variables que leen las hojas las escribe alguien")
	}
}

// Cada regla de una hoja de sistema nombra su sistema.
//
// # Qué pasó
//
// Catorce reglas de la hoja de Windows colgaban de `[data-barra="propia"]` **a secas**, sin decir
// que eran de Windows. Era cierto mientras Windows fuera el único con barra de título propia, o sea
// mientras la bandera y el sistema significaran lo mismo. Al ir a dársela a GNOME —que es lo que
// pide una ventana de libadwaita— esas catorce se le habrían aplicado: la banda de Windows, sus
// botones cuadrados y el rojo de cerrar, dentro de una ventana de GNOME.
//
// No llegó a pasar porque se vio **antes** de escribir la hoja, y esta prueba es lo que hace que no
// dependa de que alguien se acuerde. Las hojas se cargan todas a la vez —son un solo paquete—, así
// que una regla que no nombra su sistema es una regla ofrecida a los otros tres.
//
// Qué se salta: los comentarios, las at-rules (`@media`, `@container`, `@keyframes`) y los pasos de
// una animación, que no son selectores.
func TestCadaReglaDeUnaHojaNombraSuSistema(t *testing.T) {
	// Cómo se llama cada hoja y qué valor de `data-sistema` le toca. macOS es **darwin**, que es lo
	// que devuelve `runtime.GOOS` y lo que ya costó 73 capturas en verde de otro sistema.
	sistemas := map[string]string{"linux": "linux", "macos": "darwin", "windows": "windows", "kde": "kde"}

	hojas, err := filepath.Glob(filepath.Join("..", "..", "frontend", "src", "sistema-*.css"))
	if err != nil || len(hojas) == 0 {
		t.Fatalf("no se encontraron las hojas de sistema: %v", err)
	}

	comentarios := regexp.MustCompile(`(?s)/\*.*?\*/`)
	selectores := regexp.MustCompile(`([^{}]+)\{`)
	pasos := regexp.MustCompile(`^(from|to|[\d.]+%)(\s*,\s*(from|to|[\d.]+%))*$`)
	espacios := regexp.MustCompile(`\s+`)

	for _, hoja := range hojas {
		nombre := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(hoja), "sistema-"), ".css")
		quiere, hay := sistemas[nombre]
		if !hay {
			t.Errorf("hay una hoja de un sistema que esta prueba no conoce (%s): añádelo a la tabla o no se vigila", filepath.Base(hoja))
			continue
		}

		b, err := os.ReadFile(hoja)
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", hoja, err)
		}
		texto := comentarios.ReplaceAllString(string(b), "")

		vistos := 0
		for _, m := range selectores.FindAllStringSubmatch(texto, -1) {
			sel := strings.TrimSpace(espacios.ReplaceAllString(m[1], " "))
			if sel == "" || strings.HasPrefix(sel, "@") || pasos.MatchString(sel) {
				continue
			}
			vistos++
			if !strings.Contains(sel, `data-sistema="`+quiere+`"`) {
				t.Errorf("%s: la regla «%s» no dice que es de %s, así que se la lleva también cualquier otro sistema",
					filepath.Base(hoja), sel, quiere)
			}
		}
		if vistos < 20 {
			t.Errorf("sólo he leído %d selectores de %s: o la hoja ha cambiado de forma o esta prueba no comprueba nada", vistos, filepath.Base(hoja))
		}
	}
}
