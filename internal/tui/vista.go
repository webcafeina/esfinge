package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/ui"
)

const sangria = "  "

// View compone la pantalla en tres bandas —cabecera fija, contenido y pie fijo—
// y, de paso, apunta qué trozos responden al ratón.
//
// Que la cabecera y el pie no se muevan no es un capricho de maquetación: los
// botones y los atajos están siempre en el mismo sitio, y el contenido de en
// medio se desplaza cuando no cabe en vez de desbordar la ventana. Desbordar era
// justo lo que rompía el ratón, porque el terminal empujaba las líneas hacia
// arriba y el mapa de zonas dejaba de coincidir con lo que se veía.
func (m modelo) View() string {
	arriba, medio, abajo := nuevoLienzo(), nuevoLienzo(), nuevoLienzo()

	arriba.blanco()
	arriba.escribe(sangria + m.e.Cabecera(m.anchoUtil(), m.version))
	arriba.escribe(sangria + m.e.Regla(m.anchoUtil()))
	arriba.blanco()

	switch m.pantalla {
	case pantMenu:
		m.verMenu(medio)
	case pantFormulario:
		m.verFormulario(arriba, medio, abajo)
	case pantResultado:
		m.verResultado(arriba, medio, abajo)
	case pantConfirmar:
		m.verConfirmacion(arriba, medio, abajo)
	case pantAyuda:
		m.verAyuda(arriba, medio, abajo)
	}

	abajo.blanco()
	abajo.escribe(sangria + m.e.Regla(m.anchoUtil()))
	m.verPie(abajo)

	// Se compone una vez para saber si algo se ha quedado fuera y, si es así,
	// otra con el aviso puesto. Saber cuánto sobra exige haber compuesto ya, y
	// arrastrar el dato del dibujado anterior daría un aviso que va siempre un
	// fotograma por detrás —y ninguno la primera vez, que es cuando más falta
	// hace—.
	salida, _ := componer(m.zonas, m.alto, m.desplazado, m.desplazadoAMano, arriba, medio, abajo)
	if aviso := m.hayMas(); aviso != "" {
		conAviso := nuevoLienzo()
		conAviso.escribe(sangria + aviso)
		conAviso.lineas = append(conAviso.lineas, abajo.lineas...)
		for _, z := range abajo.zonas {
			conAviso.marcaEn(z.fila+1, z.id, z.col1, z.col2)
		}
		salida, _ = componer(m.zonas, m.alto, m.desplazado, m.desplazadoAMano, arriba, medio, conAviso)
	}

	// La petición de copiado al terminal se cuela aquí, delante de todo: es una
	// secuencia de escape, así que no ocupa sitio ni desplaza nada de lo que hay
	// dibujado. Se vacía al emitirla para no repetirla en cada redibujado.
	if m.zonas.osc != "" {
		salida = secuenciaOSC52(m.zonas.osc) + salida
		m.zonas.osc = ""
	}
	return salida
}

func (m modelo) anchoUtil() int {
	a := m.ancho - 4
	if a < 32 {
		a = 32
	}
	if a > 76 {
		a = 76
	}
	return a
}

// verPie dibuja los recordatorios de abajo y los deja pulsables. Se ven como
// botones, así que tienen que comportarse como botones: quien lee «C Copiar»
// abajo y hace clic espera que copie, no que no pase nada.
func (m modelo) verPie(l *lienzo) {
	type atajo struct{ tecla, texto, id string }

	var atajos []atajo
	switch m.pantalla {
	case pantMenu:
		atajos = []atajo{
			{"↑↓", "Mover", ""},
			{"Intro", "Elegir", ""},
			{"Q", "Salir", zonaSalirDelTodo},
		}
	case pantFormulario:
		accion := "Cifrar"
		if m.accion == accDescifrar {
			accion = "Descifrar"
		}
		atajos = []atajo{
			{"Tab", "Cambiar de campo", ""},
			{"Intro", accion, zonaOK},
			{"Esc", "Volver", zonaVolver},
		}
	case pantResultado:
		atajos = []atajo{
			{"C", "Copiar", zonaCopiar},
			{"G", "Guardar", zonaGuarda},
			{"Intro", "Volver", zonaVolver},
		}
		if m.accion == accGenerar {
			atajos = append(atajos, atajo{"R", "Otra", zonaOtra})
		}
	case pantAyuda:
		atajos = []atajo{{"Intro", "Volver", zonaVolver}}
	case pantConfirmar:
		atajos = []atajo{
			{"G", "Guardar y salir", zonaGuardaYSal},
			{"Q", "Salir igualmente", zonaSalirYa},
			{"Esc", "Cancelar", zonaCancelar},
		}
	}

	trozos := make([]string, len(atajos))
	for i, a := range atajos {
		estilo := m.e.Apagado
		if a.id != "" && m.encima == a.id {
			estilo = m.e.Acento
		}
		trozos[i] = m.e.Tecla(a.tecla) + estilo.Render(" "+a.texto)
	}

	linea, limites := ui.Fila(len(sangria), trozos)
	l.escribe(linea)
	for i, a := range atajos {
		if a.id != "" {
			l.marca(a.id, limites[i][0], limites[i][1])
		}
	}
}

// aviso de que hay más contenido del que se ve. Sin esto, un formulario
// desplazado parece un formulario al que le faltan campos.
func (m modelo) hayMas() string {
	switch {
	case m.zonas.arriba > 0 && m.zonas.abajo > 0:
		return m.e.Apagado.Render(fmt.Sprintf("▲ %d ·  ▼ %d · rueda del ratón", m.zonas.arriba, m.zonas.abajo))
	case m.zonas.abajo > 0:
		return m.e.Apagado.Render(fmt.Sprintf("▼ %d líneas más · rueda del ratón", m.zonas.abajo))
	case m.zonas.arriba > 0:
		return m.e.Apagado.Render(fmt.Sprintf("▲ %d líneas más arriba", m.zonas.arriba))
	}
	return ""
}

func (m modelo) verMenu(l *lienzo) {
	// El rótulo cabe en cualquier terminal razonable, y es la cara de la
	// herramienta: solo desaparece en ventanas de menos de veinte filas, donde no
	// hay sitio ni para el menú.
	if m.alto >= 20 {
		l.escribe(sangrar(ui.Banner(m.e)))
		l.blanco()
	}
	l.escribe(sangria + m.e.Etiq("Qué quieres hacer"))
	l.blanco()

	for i, e := range menu {
		elegido := i == m.cursor
		bajoElPuntero := m.encima == fmt.Sprintf("%s%d", zonaMenu, i)

		marcador := "  "
		titulo := m.e.Cuerpo.Render(e.titulo)
		glifo := m.e.Apagado.Render(e.glifo)

		switch {
		case elegido:
			// El foco lo marca el acento, que es lo que hace ClickHouse. Un gris
			// más fuerte no se distingue: su filete #3a3a3a sobre el lienzo da
			// 1,74:1 y no llega ni a insinuarse.
			marcador = m.e.Acento.Render(ui.GlifoBarra + " ")
			titulo = m.e.Seleccion.Render(e.titulo)
			glifo = m.e.Acento.Render(e.glifo)
		case bajoElPuntero:
			marcador = m.e.Filete.Render(ui.GlifoBarra + " ")
			titulo = m.e.Titulo.Render(e.titulo)
		}

		l.escribe(sangria + marcador + glifo + "  " + titulo)
		l.marcaFila(fmt.Sprintf("%s%d", zonaMenu, i))

		if elegido && e.pie != "" {
			l.escribe(sangria + "      " + m.e.Apagado.Render(e.pie))
		}
	}
}

func (m modelo) verFormulario(cabeza, l, pie *lienzo) {
	etiquetas := []string{"Secreto", "Clave", "Repite la clave"}
	titulo := "Cifrar"
	if m.accion == accDescifrar {
		etiquetas = []string{"Texto cifrado", "Clave"}
		titulo = "Descifrar"
	}
	if m.modo == modoFichero {
		etiquetas[0] = "Fichero"
	}

	// Las migas y el conmutador van en la banda fija de arriba. Elegir entre
	// texto y fichero es lo primero que se decide y lo que hay que poder cambiar
	// en cualquier momento: si viajara con el contenido, se iría de la pantalla
	// en cuanto el formulario se desplazase.
	cabeza.escribe(sangria + m.e.Migas("Menú", titulo))
	cabeza.blanco()
	m.botonesEn(cabeza, []boton{
		{zonaModoTexto, "Texto", m.modo == modoTexto},
		{zonaModoFich, "Fichero", m.modo == modoFichero},
	})
	cabeza.blanco()

	for i, c := range m.campos {
		enfocado := i == m.foco
		etiqueta := m.e.Apagado.Render(etiquetas[i])
		estilo := m.e.Campo
		if enfocado {
			etiqueta = m.e.Acento.Render(etiquetas[i])
			estilo = m.e.CampoFoco
			l.ancla = l.fila() // el campo con el foco no puede quedarse fuera
		}

		l.escribe(sangria + etiqueta)
		l.escribe(sangrar(estilo.Width(m.anchoUtil() - 4).Render(c.View())))

		// Las tres líneas del recuadro responden al ratón, para que un clic en
		// cualquier punto de él enfoque el campo.
		id := fmt.Sprintf("%s%d", zonaCampo, i)
		for f := 3; f >= 1; f-- {
			l.marcaEn(l.fila()-f, id, 0, 9999)
		}

		// El medidor solo bajo el campo de la clave y solo al cifrar: al
		// descifrar la clave es la que es, y valorarla no ayuda a nadie.
		if i == campoClave && m.accion == accCifrar && c.Value() != "" {
			f := cripto.Evaluar(c.Value())
			linea := sangria + m.e.MedidorSegmentos(f.Nivel) + " " + m.e.Apagado.Render(f.Etiqueta)
			if f.Sugerencia != "" {
				linea += m.e.Apagado.Render(" · " + f.Sugerencia)
			}
			l.escribe(linea)
		}
		l.blanco()
	}

	// Los mensajes de estado van en la banda fija, no en el contenido: un error
	// escrito abajo del formulario se queda fuera de la vista en cuanto hay que
	// desplazar, y entonces el botón parece no hacer nada. Es exactamente lo que
	// pasaba con «Falta el fichero».
	switch {
	case m.trabajando:
		pie.escribe(sangria + m.e.Acento.Render(indicador(m.giro)+" Derivando la clave…"))
	case m.err != nil:
		pie.escribe(sangria + m.e.Mal(m.err.Error()))
	case m.accion == accCifrar:
		pie.escribe(sangria + m.e.Ojo("Sin la clave no hay forma de recuperar esto"))
	}

	if !m.trabajando {
		accion := "Cifrar"
		if m.accion == accDescifrar {
			accion = "Descifrar"
		}
		m.botonesEn(pie, []boton{
			{zonaOK, accion, true},
			{zonaVolver, "Volver", false},
		})
	}
}

func (m modelo) verResultado(cabeza, l, pie *lienzo) {
	if m.err != nil {
		cabeza.escribe(sangria + m.e.Migas("Menú", "No ha podido ser"))
		cabeza.blanco()
		l.escribe(sangria + m.e.Mal(m.err.Error()))
		if pista := pistaDe(m.err); pista != "" {
			l.escribe(sangria + "  " + m.e.Apagado.Render(pista))
		}
		m.botonesEn(pie, []boton{{zonaVolver, "Volver", true}})
		return
	}

	cabeza.escribe(sangria + m.e.Migas("Menú", m.titulo))
	cabeza.blanco()
	l.escribe(sangrar(m.e.Panel.Width(m.anchoUtil() - 4).
		Render(m.e.Codigo.Render(ajustar(m.resultado, m.anchoUtil()-10)))))
	l.blanco()

	// Lo que ha salido bien va destacado y con su glifo: que el texto esté ya en
	// el portapapeles es la mitad del trabajo hecho, y enterarse de ello no puede
	// depender de leer una línea gris.
	if m.exito != "" {
		l.escribe(sangria + m.e.Ok(m.exito))
	}
	if m.nota != "" {
		l.escribe(sangria + m.e.Info(m.nota))
	}

	// El aviso de que esto no se recupera va aquí, y no solo antes de cifrar:
	// este es el momento en que la persona está a punto de cerrar y decidir si
	// guarda, que es cuando de verdad importa.
	if m.aviso != "" {
		l.escribe(sangria + m.e.Ojo(m.aviso))
	}

	botones := []boton{
		{zonaCopiar, "Copiar", m.exito == ""},
		{zonaGuarda, "Guardar en un fichero", false},
	}
	if m.accion == accGenerar {
		botones = append(botones, boton{zonaOtra, "Generar otra", false})
	}
	botones = append(botones, boton{zonaVolver, "Volver", false})
	m.botonesEn(pie, botones)
}

// verConfirmacion se interpone entre un resultado sin guardar y la salida.
func (m modelo) verConfirmacion(cabeza, l, pie *lienzo) {
	cabeza.escribe(sangria + m.e.Migas("Menú", m.titulo, "¿Salir sin guardar?"))
	cabeza.blanco()
	l.escribe(sangria + m.e.Ojo("Esto todavía no está en ningún sitio"))
	l.blanco()
	l.escribe(sangria + m.e.Cuerpo.Render("Si sales ahora, lo que hay en pantalla se pierde y no"))
	l.escribe(sangria + m.e.Cuerpo.Render("hay forma de volver a generarlo igual."))

	m.botonesEn(pie, []boton{
		{zonaGuardaYSal, "Guardar y salir", true},
		{zonaSalirYa, "Salir sin guardar", false},
		{zonaCancelar, "Cancelar", false},
	})
}

type boton struct {
	id       string
	texto    string
	primario bool
}

// botonesEn dibuja una fila de botones y apunta dónde cae cada uno.
func (m modelo) botonesEn(l *lienzo, bs []boton) {
	trozos := make([]string, len(bs))
	for i, b := range bs {
		estado := ui.BotonNormal
		if m.encima == b.id {
			estado = ui.BotonEncima
		}
		trozos[i] = m.e.Boton(b.texto, b.primario, estado)
	}

	linea, limites := ui.Fila(len(sangria), trozos)
	l.escribe(linea)
	for i, b := range bs {
		l.marca(b.id, limites[i][0], limites[i][1])
	}
}

func sangrar(s string) string {
	lineas := strings.Split(s, "\n")
	for i, l := range lineas {
		lineas[i] = sangria + l
	}
	return strings.Join(lineas, "\n")
}

// indicador es el punto que gira mientras Argon2id hace su trabajo. Sin él, el
// medio segundo largo de la derivación parece un cuelgue.
func indicador(paso int) string {
	fotogramas := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return fotogramas[paso%len(fotogramas)]
}

// ajustar parte una línea larga —un contenedor ESF1 lo es siempre— para que
// quepa en el panel sin desbordar el terminal.
func ajustar(s string, ancho int) string {
	if ancho < 8 {
		ancho = 8
	}
	var b strings.Builder
	for _, linea := range strings.Split(s, "\n") {
		for lipgloss.Width(linea) > ancho {
			b.WriteString(linea[:ancho] + "\n")
			linea = linea[ancho:]
		}
		b.WriteString(linea + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// pistaDe traduce el error a lo que la persona puede hacer al respecto.
func pistaDe(err error) string {
	switch {
	case errors.Is(err, cripto.ErrClaveIncorrecta):
		return "Comprueba la clave, y que el texto esté entero"
	case errors.Is(err, cripto.ErrFormato):
		return "El texto tiene que empezar por ESF1."
	case errors.Is(err, cripto.ErrTruncado), errors.Is(err, cripto.ErrDanado):
		return "No es la clave: pide que te lo manden otra vez"
	case errors.Is(err, cripto.ErrVersion):
		return "Hace falta una versión más nueva de Esfinge"
	}
	return ""
}
