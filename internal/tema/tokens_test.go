package tema

import (
	"os"
	"path/filepath"
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
