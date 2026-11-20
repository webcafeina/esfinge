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
