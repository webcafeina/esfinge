// Package salida pinta lo que la línea de comandos escribe en el terminal.
//
// Es lo que queda de la capa visual de la 1.x después de que los menús de
// terminal dejaran su sitio a la aplicación de escritorio: los estilos que la
// CLI usa de verdad y nada más. La paleta y el contraste viven en el paquete
// tema, que no sabe nada de terminales y sirve igual a las dos caras.
//
// Aquí sigue habiendo lipgloss, pero ya no bubbletea, que era quien consultaba
// el color de fondo del terminal en su init() y costaba cinco segundos de
// arranque en terminales que no contestan.
package salida

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/webcafeina/esfinge/internal/tema"
)

// Marca de la casa, para el pie de los comandos.
const (
	Nombre      = "esfinge"
	Wordmark    = "webcafeína"
	Descripcion = "Cifra y descifra secretos con una clave"
	GlifoBarra  = "▍"
)

// Glifos de estado. Se imprimen siempre, con color y sin él: cuando el terminal
// no tiene color —NO_COLOR, una tubería, una consola antigua— el glifo es lo
// único que distingue un acierto de un fallo.
const (
	GlifoExito = "✓"
	GlifoAviso = "!"
	GlifoError = "✕"
	GlifoInfo  = "·"
)

// Estilos es la paleta convertida en estilos de lipgloss, resueltos una vez al
// arrancar.
type Estilos struct {
	Tema tema.Tema

	Titulo   lipgloss.Style
	Etiqueta lipgloss.Style
	Cuerpo   lipgloss.Style
	Apagado  lipgloss.Style
	Acento   lipgloss.Style
	Codigo   lipgloss.Style // el contenedor cifrado, que es lo que se copia
	Panel    lipgloss.Style

	Exito lipgloss.Style
	Aviso lipgloss.Style
	Error lipgloss.Style

	Filete lipgloss.Style
}

func color(c tema.RGB) lipgloss.Color { return lipgloss.Color(c.Hex()) }

// NuevosEstilos construye los estilos de un tema.
func NuevosEstilos(t tema.Tema) Estilos {
	base := lipgloss.NewStyle()

	return Estilos{
		Tema:     t,
		Titulo:   base.Foreground(color(t.Tinta)).Bold(true),
		Etiqueta: base.Foreground(color(t.Apagado)),
		Cuerpo:   base.Foreground(color(t.Cuerpo)),
		Apagado:  base.Foreground(color(t.Apagado)),
		Acento:   base.Foreground(color(t.Acento)),
		Codigo:   base.Foreground(color(t.Acento)),
		Panel: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color(t.Filete)).
			Padding(1, 2),
		Exito:  base.Foreground(color(t.Exito)),
		Aviso:  base.Foreground(color(t.Aviso)),
		Error:  base.Foreground(color(t.Error)),
		Filete: base.Foreground(color(t.Filete)),
	}
}

// TemaPorNombre resuelve la preferencia de tema: «claro», «oscuro» o «auto».
// Un nombre vacío consulta la variable de entorno ESFINGE_TEMA antes de decidir.
//
// Averiguar el fondo de un terminal exige preguntárselo con una secuencia de
// escape y esperar su respuesta; termenv espera cinco segundos antes de
// rendirse. Por eso la consulta solo se hace cuando de verdad hace falta: con el
// tema puesto a mano se le dice a lipgloss cuál es, y con la salida redirigida
// no se pregunta nada porque no se va a pintar un solo color.
func TemaPorNombre(nombre string) (tema.Tema, error) {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	if nombre == "" || nombre == "auto" {
		if v := strings.ToLower(strings.TrimSpace(os.Getenv("ESFINGE_TEMA"))); v != "" {
			nombre = v
		}
	}

	switch nombre {
	case "claro":
		lipgloss.SetHasDarkBackground(false)
		return tema.TemaClaro, nil
	case "oscuro":
		lipgloss.SetHasDarkBackground(true)
		return tema.TemaOscuro, nil
	case "", "auto":
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			lipgloss.SetHasDarkBackground(true)
			return tema.TemaOscuro, nil
		}
		if lipgloss.HasDarkBackground() {
			return tema.TemaOscuro, nil
		}
		return tema.TemaClaro, nil
	default:
		return tema.TemaOscuro, &errorTema{nombre}
	}
}

type errorTema struct{ nombre string }

func (e *errorTema) Error() string {
	return "El tema «" + e.nombre + "» no existe: usa claro, oscuro o auto"
}

// Mayusculas imita el caption-uppercase del sistema de origen —mayúsculas con
// tracking— separando las letras con un espacio. En un terminal el tracking no
// existe, y esta es la única forma de que una etiqueta se lea como etiqueta.
func Mayusculas(s string) string {
	letras := []rune(strings.ToUpper(s))
	var b strings.Builder
	b.Grow(len(letras) * 2)
	for i, r := range letras {
		if i > 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Etiq pinta una etiqueta de sección.
func (e Estilos) Etiq(s string) string { return e.Etiqueta.Render(Mayusculas(s)) }

// Con glifo delante, para que el estado se distinga también sin color.
func (e Estilos) Ok(s string) string   { return e.Exito.Render(GlifoExito + " " + s) }
func (e Estilos) Ojo(s string) string  { return e.Aviso.Render(GlifoAviso + " " + s) }
func (e Estilos) Mal(s string) string  { return e.Error.Render(GlifoError + " " + s) }
func (e Estilos) Info(s string) string { return e.Apagado.Render(GlifoInfo + " " + s) }

// Medidor dibuja la fuerza de una clave en cinco bloques. Encendido y apagado se
// distinguen por el carácter y no solo por el color: sin color, una fila de
// bloques idénticos no dice nada.
func (e Estilos) Medidor(nivel, de int) string {
	const segmentos = 5

	estilo := e.Exito
	switch {
	case nivel <= 1:
		estilo = e.Error
	case nivel == 2:
		estilo = e.Aviso
	}

	// Nivel 0 enciende un segmento igualmente: una fila entera apagada se lee
	// como «aquí no hay medidor» y no como «esta clave es malísima».
	encendidos := nivel + 1
	if encendidos > segmentos {
		encendidos = segmentos
	}

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

// Firma es la línea de marca: barra de acento, wordmark y descripción.
func Firma(e Estilos) string {
	return e.Acento.Render(GlifoBarra) + " " +
		e.Titulo.Render(Wordmark) + " " +
		e.Apagado.Render(GlifoInfo+" "+Descripcion)
}
