package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Estilos es la paleta ya convertida en estilos de lipgloss, resueltos una vez
// al arrancar.
type Estilos struct {
	Tema Tema

	Titulo   lipgloss.Style
	Subtitulo lipgloss.Style
	Etiqueta lipgloss.Style // caption-uppercase de ClickHouse: mayúsculas con tracking
	Cuerpo   lipgloss.Style
	Apagado  lipgloss.Style
	Acento   lipgloss.Style
	Codigo   lipgloss.Style // el secreto cifrado, que es lo que se copia

	Panel      lipgloss.Style
	PanelFoco  lipgloss.Style
	Campo      lipgloss.Style
	CampoFoco  lipgloss.Style
	Relleno    lipgloss.Style // fondo amarillo con tinta encima
	Seleccion  lipgloss.Style

	Exito lipgloss.Style
	Aviso lipgloss.Style
	Error lipgloss.Style

	Filete lipgloss.Style
}

func color(c RGB) lipgloss.Color { return lipgloss.Color(c.Hex()) }

// TemaPorNombre resuelve la preferencia de tema: «claro», «oscuro» o «auto».
// Un nombre vacío consulta la variable de entorno ESFINGE_TEMA antes de decidir.
//
// Averiguar el fondo de un terminal exige preguntárselo con una secuencia de
// escape y esperar su respuesta; termenv espera cinco segundos —OSCTimeout, que
// es una constante y no se puede tocar— antes de rendirse. Aquí la consulta se
// hace solo cuando de verdad hace falta: con el tema puesto a mano se le dice a
// lipgloss cuál es, y con la salida redirigida no se pregunta nada porque no se
// va a pintar un solo color.
//
// Ojo, que esto no es la historia entera: bubbletea llama a HasDarkBackground()
// en su propio init(), o sea antes de que este código llegue a ejecutarse, y esa
// primera consulta no hay forma de evitarla mientras la TUI esté en el mismo
// binario. En un terminal que no conteste al OSC 11, el arranque cuesta cinco
// segundos hasta en «esfinge generar». La salida documentada es CI=1 o TERM=dumb,
// que hacen que termenv ni pregunte. Bubbletea marca ese init como provisional
// («This workaround will be removed in v2»), así que conviene volver aquí al
// actualizar.
func TemaPorNombre(nombre string) (Tema, error) {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	if nombre == "" || nombre == "auto" {
		if v := strings.ToLower(strings.TrimSpace(os.Getenv("ESFINGE_TEMA"))); v != "" {
			nombre = v
		}
	}

	switch nombre {
	case "claro":
		lipgloss.SetHasDarkBackground(false)
		return TemaClaro, nil
	case "oscuro":
		lipgloss.SetHasDarkBackground(true)
		return TemaOscuro, nil
	case "", "auto":
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			lipgloss.SetHasDarkBackground(true)
			return TemaOscuro, nil
		}
		if lipgloss.HasDarkBackground() {
			return TemaOscuro, nil
		}
		return TemaClaro, nil
	default:
		return TemaOscuro, &erroresTema{nombre}
	}
}

type erroresTema struct{ nombre string }

func (e *erroresTema) Error() string {
	return "El tema «" + e.nombre + "» no existe: usa claro, oscuro o auto"
}

// NuevosEstilos construye los estilos de un tema.
func NuevosEstilos(t Tema) Estilos {
	base := lipgloss.NewStyle()

	return Estilos{
		Tema: t,

		Titulo:    base.Foreground(color(t.Tinta)).Bold(true),
		Subtitulo: base.Foreground(color(t.Cuerpo)),
		Etiqueta:  base.Foreground(color(t.Apagado)),
		Cuerpo:    base.Foreground(color(t.Cuerpo)),
		Apagado:   base.Foreground(color(t.Apagado)),
		Acento:    base.Foreground(color(t.Acento)),
		Codigo:    base.Foreground(color(t.Acento)),

		// Padding 1×2: el «comfortable» de ClickHouse —24px de relleno de tarjeta,
		// 16px entre elementos— traducido a celdas de terminal.
		Panel: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color(t.Filete)).
			Padding(1, 2),

		// Los campos van más apretados que los paneles: con el relleno vertical de
		// una tarjeta, tres campos y sus etiquetas no caben en las veinticuatro
		// líneas de un terminal corriente.
		Campo: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color(t.Filete)).
			Padding(0, 1),
		CampoFoco: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color(t.Acento)).
			Padding(0, 1),

		PanelFoco: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color(t.Acento)).
			Padding(1, 2),

		Relleno:   base.Background(color(t.Relleno)).Foreground(color(t.SobreAcento)).Bold(true).Padding(0, 1),
		Seleccion: base.Foreground(color(t.Acento)).Bold(true),

		Exito: base.Foreground(color(t.Exito)),
		Aviso: base.Foreground(color(t.Aviso)),
		Error: base.Foreground(color(t.Error)),

		Filete: base.Foreground(color(t.Filete)),
	}
}

// Mayusculas imita el caption-uppercase de ClickHouse —12px, tracking 1,5px—
// separando las letras con un espacio. En un terminal el tracking no existe, y
// esta es la única forma de que una etiqueta se lea como etiqueta y no como texto.
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

// Etiq pinta una etiqueta de sección al estilo de ClickHouse.
func (e Estilos) Etiq(s string) string { return e.Etiqueta.Render(Mayusculas(s)) }

// Con glifo delante, para que el estado se distinga también sin color.
func (e Estilos) Ok(s string) string     { return e.Exito.Render(GlifoExito + " " + s) }
func (e Estilos) Ojo(s string) string    { return e.Aviso.Render(GlifoAviso + " " + s) }
func (e Estilos) Mal(s string) string    { return e.Error.Render(GlifoError + " " + s) }
func (e Estilos) Info(s string) string   { return e.Apagado.Render(GlifoInfo + " " + s) }

// Regla es el filete de 1px de ClickHouse, que es su única separación.
func (e Estilos) Regla(ancho int) string {
	if ancho < 1 {
		ancho = 1
	}
	return e.Filete.Render(strings.Repeat("─", ancho))
}

// Medidor dibuja una barra de progreso discreta, para la fuerza de la clave.
func (e Estilos) Medidor(nivel, de, ancho int) string {
	lleno := ancho * nivel / de
	if lleno > ancho {
		lleno = ancho
	}
	// El nivel más bajo no deja la barra vacía: una barra sin nada dentro se lee
	// como «aquí no hay medidor», y no como «esta clave es malísima».
	if lleno < 1 {
		lleno = 1
	}
	var estilo lipgloss.Style
	switch {
	case nivel <= 1:
		estilo = e.Error
	case nivel == 2:
		estilo = e.Aviso
	default:
		estilo = e.Exito
	}
	return estilo.Render(strings.Repeat("█", lleno)) +
		e.Filete.Render(strings.Repeat("░", ancho-lleno))
}

// Sobre el color: no hace falta comprobarlo a mano en ninguna parte. Lipgloss
// degrada solo —truecolor, 256, 16, ninguno—, respeta NO_COLOR y no emite un
// escape cuando la salida no es un terminal. Lo que sí hace falta es que los
// estados lleven glifo además de color, y de eso se encargan Ok, Ojo y Mal.
