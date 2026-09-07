package tema

import (
	"fmt"
	"sort"
	"strings"
)

// Espaciados y formas, en la escala que usan macOS y Windows para sus controles.
// Van aquí y no en una hoja de estilos suelta por el mismo motivo que los
// colores: una sola fuente de verdad, y del lado que tiene los tests.
var Medidas = map[string]string{
	"espacio-1":     "4px",
	"espacio-2":     "8px",
	"espacio-3":     "12px",
	"espacio-4":     "16px",
	"espacio-5":     "24px",
	"espacio-6":     "32px",
	"radio-chico":   "6px",
	"radio":         "8px",
	"radio-grande":  "12px",
	"alto-control":  "32px",
	"texto-chico":   "12px",
	"texto":         "13px",
	"texto-grande":  "15px",
	"titulo":        "20px",
	"fuente":        `-apple-system, BlinkMacSystemFont, "Segoe UI Variable Text", "Segoe UI", system-ui, sans-serif`,
	"fuente-mono":   `ui-monospace, SFMono-Regular, "SF Mono", "Cascadia Mono", Menlo, Consolas, monospace`,
}

// campos del tema que se publican como variables de color, en el orden en que se
// escriben. La lista es explícita para que añadir un campo al tema sea una
// decisión y no un efecto colateral.
var campos = []struct {
	nombre string
	de     func(Tema) RGB
}{
	{"lienzo", func(t Tema) RGB { return t.Lienzo }},
	{"suave", func(t Tema) RGB { return t.Suave }},
	{"tarjeta", func(t Tema) RGB { return t.Tarjeta }},
	{"elevada", func(t Tema) RGB { return t.Elevada }},
	{"filete", func(t Tema) RGB { return t.Filete }},
	{"filete-fuerte", func(t Tema) RGB { return t.FileteFuerte }},
	{"tinta", func(t Tema) RGB { return t.Tinta }},
	{"cuerpo", func(t Tema) RGB { return t.Cuerpo }},
	{"apagado", func(t Tema) RGB { return t.Apagado }},
	{"relleno", func(t Tema) RGB { return t.Relleno }},
	{"relleno-vivo", func(t Tema) RGB { return t.RellenoVivo }},
	{"sobre-acento", func(t Tema) RGB { return t.SobreAcento }},
	{"acento", func(t Tema) RGB { return t.Acento }},
	{"exito", func(t Tema) RGB { return t.Exito }},
	{"aviso", func(t Tema) RGB { return t.Aviso }},
	{"error", func(t Tema) RGB { return t.Error }},
}

// GenerarCSS escribe los tokens que consume la interfaz.
//
// El tema claro va en :root y el oscuro se aplica de dos maneras: por la
// preferencia del sistema y por un atributo, para que se pueda forzar uno u otro
// sin depender de los ajustes del escritorio. Ningún color se define solo dentro
// de la consulta de medios: si lo estuviera, un navegador sin soporte se
// quedaría sin él.
func GenerarCSS() string {
	var b strings.Builder

	b.WriteString("/* Generado por internal/tema. No se edita a mano:\n")
	b.WriteString("   cámbialo en Go y ejecuta «make tokens», que además mide el\n")
	b.WriteString("   contraste de cada pareja y falla si una no cumple AA. */\n\n")

	b.WriteString(":root {\n")
	for _, c := range campos {
		fmt.Fprintf(&b, "  --%s: %s;\n", c.nombre, c.de(TemaClaro).Hex())
	}
	b.WriteString("\n")

	claves := make([]string, 0, len(Medidas))
	for k := range Medidas {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	for _, k := range claves {
		fmt.Fprintf(&b, "  --%s: %s;\n", k, Medidas[k])
	}
	b.WriteString("}\n\n")

	oscuro := func(selector string) string {
		var s strings.Builder
		fmt.Fprintf(&s, "%s {\n", selector)
		for _, c := range campos {
			fmt.Fprintf(&s, "  --%s: %s;\n", c.nombre, c.de(TemaOscuro).Hex())
		}
		s.WriteString("}\n")
		return s.String()
	}

	b.WriteString("@media (prefers-color-scheme: dark) {\n")
	for _, linea := range strings.Split(oscuro(`:root:not([data-tema="claro"])`), "\n") {
		if linea == "" {
			continue
		}
		b.WriteString("  " + linea + "\n")
	}
	b.WriteString("}\n\n")
	b.WriteString(oscuro(`:root[data-tema="oscuro"]`))

	return b.String()
}
