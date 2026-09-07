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
	ruta := filepath.Join("..", "..", "frontend", "src", "tokens.css")

	enDisco, err := os.ReadFile(ruta)
	if err != nil {
		t.Skipf("todavía no hay interfaz que consuma los tokens: %v", err)
	}
	if string(enDisco) != GenerarCSS() {
		t.Errorf("%s no coincide con lo que genera Go. Ejecuta «make tokens».", ruta)
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
