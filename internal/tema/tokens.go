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
	"espacio-1": "4px",
	"espacio-2": "8px",
	"espacio-3": "12px",
	"espacio-4": "16px",
	"espacio-5": "24px",
	"espacio-6": "32px",
	// macOS pasó a esquinas mucho más redondeadas, y una ventana con radios de
	// hace diez años canta enseguida.
	"radio-chico":  "6px",
	"radio":        "9px",
	"radio-grande": "14px",
	"alto-control": "28px",
	// La estructura de una aplicación con barra lateral. Los tres primeros están
	// medidos sobre capturas de macOS 26, usando los semáforos como regla: miden
	// 12 pt clavados, así que sirven para sacar la escala de cualquier captura.
	//
	//	barra lateral ....... 224,6 pt
	//	paso entre filas .... 38,5 pt
	//	semáforos ........... a 21 pt del borde de arriba
	"barra-lateral": "225px",
	"alto-fila":     "38px",
	// El hueco que hay que dejar libre arriba de la barra lateral para los
	// semáforos. Sin barra de título son ellos los que mandan.
	"hueco-semaforos": "52px",
	"texto-chico":     "12px",
	"texto":           "13px",
	"texto-grande":    "15px",
	"titulo":          "20px",
	"fuente":          `-apple-system, BlinkMacSystemFont, "Segoe UI Variable Text", "Segoe UI", system-ui, sans-serif`,
	"fuente-mono":     `ui-monospace, SFMono-Regular, "SF Mono", "Cascadia Mono", Menlo, Consolas, monospace`,
}

// Aquí vivía «alfaDelVidrio», el tinte que la barra lateral pintaba encima del
// material del sistema. **Ya no hay tinte**, y la historia merece quedarse.
//
// Se probó a 0,82 y no se notaba; se bajó a 0,55 y tampoco. Las dos veces se
// eligió el número mirando una simulación, y las dos veces se estaba resolviendo
// el problema equivocado: el material no se veía porque Wails nunca le pone
// material a su NSVisualEffectView, no porque lo tapase el tinte.
//
// Y con eso resuelto, el tinte sobra por definición: en una barra lateral de
// macOS **el material es el fondo**, no lleva otra capa encima. Pintar cualquier
// cosa por delante es volver a la casilla de salida. Ver vidrio_darwin.go.

// conAlfa escribe un color con transparencia, para el anillo de foco.
func conAlfa(c RGB, alfa float64) string {
	return fmt.Sprintf("rgb(%d %d %d / %.2f)", c.R, c.G, c.B, alfa)
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
	{"barra", func(t Tema) RGB { return t.Barra }},
	{"barra-encima", func(t Tema) RGB { return t.BarraEncima }},
	{"campo", func(t Tema) RGB { return t.Campo }},
	{"boton", func(t Tema) RGB { return t.Boton }},
	{"boton-encima", func(t Tema) RGB { return t.BotonEncima }},
	{"tinta-icono", func(t Tema) RGB { return t.TintaDeIcono }},
}

// Y los ocho tintes de los cuadros de la bóveda, que no caben en la tabla de
// arriba porque son una lista y no un campo. Van aparte y no se mezclan: la
// tabla existe para que añadir **un** color sea una decisión visible, y meterle
// un bucle dentro la volvería difícil de leer de un vistazo.
func tintesDe(t Tema) string {
	var b strings.Builder
	for i, tinte := range t.TintesDeIcono {
		fmt.Fprintf(&b, "  --tinte-icono-%d: %s;\n", i+1, tinte.Hex())
	}
	return b.String()
}

// juego son las dos paletas de un sistema y el selector del que cuelgan.
//
// El de la base va **vacío**: es `:root` a secas, y es lo que ven el panel de la
// extensión y los escritorios que no tienen hoja propia. Los demás refinan con
// un atributo, que pesa más que `:root` solo, así que mandan en su sistema sin
// que haya que ordenar nada a mano.
type juego struct {
	selector      string
	claro, oscuro Tema
}

func juegos() []juego {
	return []juego{
		{"", TemaClaro, TemaOscuro},
		{`[data-sistema="windows"]`, WindowsClaro, WindowsOscuro},
		{`[data-sistema="darwin"]`, MacosClaro, MacosOscuro},
	}
}

// Ficheros son los sitios donde se escriben los tokens, y **son dos**.
//
// La extensión importa estos mismos tokens en su panel, y allí `data-sistema` no
// existe ni puede existir: un navegador no es una ventana de Windows. Mandarle
// el fichero entero sería mandarle bloques que nunca puede activar, crecientes
// con cada sistema que se añada. Así que recibe solo la base, que es justo lo que
// usa. Es el mismo reparto que hace Apolo con su web.
var Ficheros = []struct {
	Ruta string
	CSS  func() string
}{
	{"frontend/src/tokens.css", GenerarCSS},
	{"navegador/src/tokens.css", GenerarCSSBase},
}

// GenerarCSS escribe los tokens que consume la ventana: la base y un bloque por
// sistema.
//
// El tema claro va en :root y el oscuro se aplica de dos maneras: por la
// preferencia del sistema y por un atributo, para que se pueda forzar uno u otro
// sin depender de los ajustes del escritorio. Ningún color se define solo dentro
// de la consulta de medios: si lo estuviera, un navegador sin soporte se
// quedaría sin él.
func GenerarCSS() string { return generar(juegos()) }

// GenerarCSSBase escribe solo la base, para el panel de la extensión. Ver
// Ficheros.
func GenerarCSSBase() string { return generar(juegos()[:1]) }

func generar(js []juego) string {
	var b strings.Builder

	b.WriteString("/* Generado por internal/tema. No se edita a mano:\n")
	b.WriteString("   cámbialo en Go y ejecuta «make tokens», que además mide el\n")
	b.WriteString("   contraste de cada pareja y falla si una no cumple AA. */\n\n")

	for i, j := range js {
		// Las medidas van **una sola vez**, con la base. Lo que cada sistema cambia
		// de medidas lo cambia su hoja (`sistema-windows.css` y compañía), que es
		// donde una altura vive al lado del relleno y del interlineado que la
		// acompañan: partir eso en dos ficheros es partir una decisión en dos.
		b.WriteString(colores(":root"+j.selector, j.claro, i == 0))
		b.WriteString("\n")

		b.WriteString("@media (prefers-color-scheme: dark) {\n")
		for _, linea := range strings.Split(colores(":root"+j.selector+`:not([data-tema="claro"])`, j.oscuro, false), "\n") {
			if linea == "" {
				continue
			}
			b.WriteString("  " + linea + "\n")
		}
		b.WriteString("}\n\n")

		b.WriteString(colores(":root"+j.selector+`[data-tema="oscuro"]`, j.oscuro, false))
		if i+1 < len(js) {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// colores escribe el bloque de un tema, con sus medidas si se le piden.
func colores(selector string, t Tema, conMedidas bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s {\n", selector)
	for _, c := range campos {
		fmt.Fprintf(&b, "  --%s: %s;\n", c.nombre, c.de(t).Hex())
	}
	b.WriteString(tintesDe(t))

	// El anillo de foco es el color de acción a media tinta: así se ve sobre
	// cualquier superficie sin tener que inventar un color por cada una. En
	// oscuro hace falta más alfa para que se llegue a ver.
	anillo, tenue := 0.35, 0.18
	if t.Oscuro {
		anillo, tenue = 0.45, 0.22
	}
	fmt.Fprintf(&b, "  --anillo: %s;\n", conAlfa(t.Relleno, anillo))
	// El acento muy diluido, para la selección de Fluent: en Windows la fila
	// activa no se rellena, se tiñe, y quien dice «estás aquí» es la barra de al
	// lado. Va translúcido a propósito, para que sirva sobre cualquier superficie.
	fmt.Fprintf(&b, "  --relleno-tenue: %s;\n", conAlfa(t.Relleno, tenue))

	if conMedidas {
		b.WriteString("\n")
		claves := make([]string, 0, len(Medidas))
		for k := range Medidas {
			claves = append(claves, k)
		}
		sort.Strings(claves)
		for _, k := range claves {
			fmt.Fprintf(&b, "  --%s: %s;\n", k, Medidas[k])
		}
	}
	b.WriteString("}\n")
	return b.String()
}
