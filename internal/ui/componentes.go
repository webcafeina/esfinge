package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// EstadoBoton dice cómo se pinta un botón: normal, bajo el puntero o pulsado.
type EstadoBoton int

const (
	BotonNormal EstadoBoton = iota
	BotonEncima             // el puntero está sobre él
	BotonActivo             // es el que se activa con la tecla de siempre
)

// Boton dibuja un botón pulsable.
//
// El primario lleva el amarillo de ClickHouse como fondo con la tinta encima,
// que es exactamente para lo que ese color sirve; el secundario se queda en un
// filete. Al pasar el puntero por encima se eleva la superficie en vez de
// cambiar el color: subir un escalón en la escala de superficies es el gesto del
// sistema, y además no obliga a inventar un color más que habría que medir.
func (e Estilos) Boton(texto string, primario bool, estado EstadoBoton) string {
	// Los corchetes no son decoración: cuando no hay color, el fondo desaparece y
	// sin ellos un botón queda indistinguible de una frase suelta con espacios
	// alrededor. Con color, además, siguen leyéndose como lo que son.
	etiqueta := "[ " + texto + " ]"
	base := lipgloss.NewStyle().Padding(0, 1)

	switch {
	case primario && estado == BotonEncima:
		return base.Background(color(e.Tema.RellenoVivo)).
			Foreground(color(e.Tema.SobreAcento)).Bold(true).Render(etiqueta)
	case primario:
		return base.Background(color(e.Tema.Relleno)).
			Foreground(color(e.Tema.SobreAcento)).Bold(true).Render(etiqueta)
	case estado == BotonEncima:
		return base.Background(color(e.Tema.Elevada)).
			Foreground(color(e.Tema.Tinta)).Render(etiqueta)
	case estado == BotonActivo:
		return base.Background(color(e.Tema.Tarjeta)).
			Foreground(color(e.Tema.Acento)).Render(etiqueta)
	default:
		return base.Background(color(e.Tema.Tarjeta)).
			Foreground(color(e.Tema.Cuerpo)).Render(etiqueta)
	}
}

// Tecla dibuja el nombre de una tecla como se ve en un teclado: un recuadro
// sobre la superficie elevada. Sirve para que las teclas de una ayuda no se
// confundan con el texto que las explica.
func (e Estilos) Tecla(t string) string {
	return lipgloss.NewStyle().
		Background(color(e.Tema.Elevada)).
		Foreground(color(e.Tema.Cuerpo)).
		Padding(0, 1).
		Render(t)
}

// Chip es una etiqueta con fondo, para el estado de una pantalla.
func (e Estilos) Chip(texto string) string {
	return lipgloss.NewStyle().
		Background(color(e.Tema.Relleno)).
		Foreground(color(e.Tema.SobreAcento)).
		Bold(true).
		Padding(0, 1).
		Render(texto)
}

// Migas sitúa dónde se está: «Menú › Cifrar». El último tramo va en acento
// porque es el sitio donde se está ahora.
func (e Estilos) Migas(tramos ...string) string {
	if len(tramos) == 0 {
		return ""
	}
	partes := make([]string, 0, len(tramos))
	for i, t := range tramos {
		if i == len(tramos)-1 {
			partes = append(partes, e.Acento.Render(t))
			continue
		}
		partes = append(partes, e.Apagado.Render(t))
	}
	return strings.Join(partes, e.Filete.Render(" › "))
}

// Cabecera es la barra de arriba: la marca a la izquierda, la versión a la
// derecha y el ancho justo entre las dos.
func (e Estilos) Cabecera(ancho int, version string) string {
	izq := e.Acento.Render(GlifoBarra) + " " + e.Titulo.Render(Mayusculas(Wordmark))
	der := e.Apagado.Render(Mayusculas(Nombre + " " + version))

	hueco := ancho - lipgloss.Width(izq) - lipgloss.Width(der)
	if hueco < 1 {
		return izq
	}
	return izq + strings.Repeat(" ", hueco) + der
}

// MedidorSegmentos es el medidor de fuerza en cinco bloques separados en vez de
// una barra continua: se cuenta de un vistazo cuántos hay encendidos, que es lo
// que se quiere saber, y no cuánto mide una barra contra un fondo.
func (e Estilos) MedidorSegmentos(nivel int) string {
	const segmentos = 5

	var estilo lipgloss.Style
	switch {
	case nivel <= 1:
		estilo = e.Error
	case nivel == 2:
		estilo = e.Aviso
	default:
		estilo = e.Exito
	}

	// Nivel 0 enciende un segmento igualmente: una fila entera apagada se lee
	// como «aquí no hay medidor» y no como «esta clave es malísima».
	encendidos := nivel + 1
	if encendidos > segmentos {
		encendidos = segmentos
	}

	// Encendido y apagado se distinguen por el carácter y no solo por el color:
	// con NO_COLOR, o en un terminal sin color, una fila de bloques idénticos no
	// dice absolutamente nada.
	partes := make([]string, segmentos)
	for i := range partes {
		if i < encendidos {
			partes[i] = estilo.Render("███")
			continue
		}
		partes[i] = e.Filete.Render("░░░")
	}
	return strings.Join(partes, " ")
}

// Fila coloca varios trozos separados por dos espacios y devuelve, además del
// texto, la columna donde empieza y acaba cada uno. Eso es lo que permite saber
// después sobre cuál de ellos ha caído un clic.
func Fila(sangria int, trozos []string) (string, [][2]int) {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", sangria))

	limites := make([][2]int, len(trozos))
	col := sangria
	for i, t := range trozos {
		if i > 0 {
			b.WriteString("  ")
			col += 2
		}
		ancho := lipgloss.Width(t)
		limites[i] = [2]int{col, col + ancho - 1}
		col += ancho
		b.WriteString(t)
	}
	return b.String(), limites
}
