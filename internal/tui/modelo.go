// Package tui es la cara de menús: la que ve quien abre Esfinge sin saber que
// existe una línea de comandos.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/ui"
)

type pantalla int

const (
	pantMenu pantalla = iota
	pantFormulario
	pantResultado
	pantConfirmar
	pantAyuda
)

type accion int

const (
	accCifrar accion = iota
	accDescifrar
	accGenerar
	accAyuda
	accSalir
)

// modo dice si el formulario trabaja con un texto tecleado o con un fichero.
type modo int

const (
	modoTexto modo = iota
	modoFichero
)

type entradaMenu struct {
	accion accion
	glifo  string
	titulo string
	pie    string
}

var menu = []entradaMenu{
	{accCifrar, "→", "Cifrar", "Convierte un secreto o un fichero en algo que solo se abre con la clave"},
	{accDescifrar, "←", "Descifrar", "Recupera un secreto o un fichero a partir de la clave"},
	{accGenerar, "✳", "Generar contraseña", "Una contraseña al azar, en hexadecimal por defecto"},
	{accAyuda, "?", "Ayuda", "Qué hace cada cosa y cómo usarlo desde la terminal"},
	{accSalir, "×", "Salir", "Cierra Esfinge"},
}

// Los campos del formulario.
const (
	campoSecreto = 0
	campoClave   = 1
	campoRepetir = 2
)

// Identificadores de las zonas pulsables. Los del menú y los campos llevan el
// número detrás.
const (
	zonaMenu      = "menu:"
	zonaCampo     = "campo:"
	zonaOK        = "boton:aceptar"
	zonaVolver    = "boton:volver"
	zonaGuarda    = "boton:guardar"
	zonaCopiar    = "boton:copiar"
	zonaOtra      = "boton:otra"
	zonaModoTexto = "modo:texto"
	zonaModoFich  = "modo:fichero"
	zonaSalirYa   = "boton:salir-sin-guardar"
	zonaGuardaYSal = "boton:guardar-y-salir"
	zonaCancelar  = "boton:cancelar"
	zonaSalirDelTodo = "boton:salir-del-todo"
)

type modelo struct {
	e       ui.Estilos
	version string

	pantalla pantalla
	accion   accion
	modo     modo
	cursor   int
	ancho    int
	alto     int

	campos []textinput.Model
	foco   int

	resultado string
	titulo    string
	nota      string
	aviso     string
	exito     string // confirmación destacada, para lo que ha salido bien
	err       error

	// aSalvo dice si lo que hay en pantalla ya está en algún sitio del que se
	// pueda recuperar. Es lo que decide si al salir hay que preguntar.
	aSalvo bool
	// trasConfirmar es lo que se hará si se confirma la salida.
	trasConfirmar func(modelo) (tea.Model, tea.Cmd)

	trabajando bool
	giro       int

	zonas  *registro
	encima string

	// desplazado es la primera línea visible del contenido cuando no cabe entero,
	// y desplazadoAMano dice si lo ha movido la persona con la rueda. Mientras no
	// lo haya hecho, la vista se coloca sola sobre el campo que tiene el foco.
	desplazado      int
	desplazadoAMano bool
}

// Nuevo construye el modelo inicial.
func Nuevo(e ui.Estilos, version string) tea.Model {
	return modelo{e: e, version: version, ancho: 80, alto: 24, zonas: &registro{}}
}

func (m modelo) Init() tea.Cmd { return textinput.Blink }

// listoMsg llega cuando termina el trabajo criptográfico, que corre fuera del
// hilo de la interfaz para que la derivación Argon2id no congele la pantalla.
type listoMsg struct {
	texto  string
	titulo string
	nota   string
	aviso  string
	aSalvo bool
	// copiarSolo pide que el resultado vaya al portapapeles sin que nadie lo
	// pida: al cifrar, lo siguiente que se hace con el texto es pegarlo en algún
	// sitio, siempre.
	copiarSolo bool
	err        error
}

// giroMsg mueve el indicador mientras se deriva la clave.
type giroMsg time.Time

func girar() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return giroMsg(t) })
}

func (m modelo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ancho, m.alto = msg.Width, msg.Height
		return m, nil

	case giroMsg:
		if !m.trabajando {
			return m, nil
		}
		m.giro++
		return m, girar()

	case listoMsg:
		m.trabajando = false
		m.resultado, m.titulo, m.nota = msg.texto, msg.titulo, msg.nota
		m.aviso, m.aSalvo, m.err = msg.aviso, msg.aSalvo, msg.err
		m.exito = ""
		m.pantalla = pantResultado
		m.desplazado, m.desplazadoAMano = 0, false

		if msg.copiarSolo && msg.err == nil {
			sig, _ := m.copiar()
			return sig, nil
		}
		return m, nil

	case tea.MouseMsg:
		return m.actualizarRaton(msg)

	case tea.KeyMsg:
		if m.trabajando {
			return m, nil // mientras deriva, solo se puede esperar
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.pantalla {
		case pantMenu:
			return m.actualizarMenu(msg)
		case pantFormulario:
			return m.actualizarFormulario(msg)
		case pantResultado:
			return m.actualizarResultado(msg)
		case pantConfirmar:
			return m.actualizarConfirmacion(msg)
		case pantAyuda:
			return m.actualizarAyuda(msg)
		}
	}

	if m.pantalla == pantFormulario {
		return m.propagarACampos(msg)
	}
	return m, nil
}

// actualizarRaton resuelve el puntero: mover resalta lo que hay debajo y soltar
// el botón lo activa.
func (m modelo) actualizarRaton(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.trabajando {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelDown:
		m.desplazado++
		m.desplazadoAMano = true
		return m, nil
	case tea.MouseButtonWheelUp:
		if m.desplazado > 0 {
			m.desplazado--
		}
		m.desplazadoAMano = m.desplazado > 0
		return m, nil
	}

	id := m.zonas.en(msg.X, msg.Y)

	// El resaltado sigue al puntero aunque no se pulse nada: es lo que hace que
	// se note que la cosa responde al ratón.
	if msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionPress {
		m.encima = id
	}

	if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	if id == "" {
		return m, nil
	}
	return m.activar(id)
}

// activar ejecuta lo que representa una zona. Es el mismo camino que recorren
// las teclas, para que el ratón y el teclado no puedan acabar haciendo cosas
// distintas.
func (m modelo) activar(id string) (tea.Model, tea.Cmd) {
	switch {
	case strings.HasPrefix(id, zonaMenu):
		var i int
		if _, err := fmt.Sscanf(id, zonaMenu+"%d", &i); err == nil && i >= 0 && i < len(menu) {
			m.cursor = i
			return m.elegir(menu[i].accion)
		}

	case strings.HasPrefix(id, zonaCampo):
		var i int
		if _, err := fmt.Sscanf(id, zonaCampo+"%d", &i); err == nil && i >= 0 && i < len(m.campos) {
			m.foco = i
			return m.enfocar()
		}

	case id == zonaModoTexto:
		return m.cambiarModo(modoTexto)
	case id == zonaModoFich:
		return m.cambiarModo(modoFichero)

	case id == zonaOK:
		return m.ejecutar()

	case id == zonaVolver:
		return m.salirDelResultado(func(mm modelo) (tea.Model, tea.Cmd) { return mm.alMenu() })

	case id == zonaGuarda:
		return m.guardar()

	case id == zonaCopiar:
		return m.copiar()

	case id == zonaOtra:
		return m.elegir(accGenerar)

	case id == zonaSalirYa:
		if m.trasConfirmar != nil {
			return m.trasConfirmar(m)
		}
		return m.alMenu()

	case id == zonaGuardaYSal:
		sig, _ := m.guardar()
		mm := sig.(modelo)
		if mm.err != nil {
			mm.pantalla = pantResultado
			return mm, nil
		}
		if mm.trasConfirmar != nil {
			return mm.trasConfirmar(mm)
		}
		return mm.alMenu()

	case id == zonaCancelar:
		m.pantalla = pantResultado
		return m, nil

	case id == zonaSalirDelTodo:
		return m, tea.Quit
	}
	return m, nil
}

func (m modelo) actualizarMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(menu)-1 {
			m.cursor++
		}
	case "enter", " ":
		return m.elegir(menu[m.cursor].accion)
	}
	return m, nil
}

func (m modelo) elegir(a accion) (tea.Model, tea.Cmd) {
	m.accion = a
	m.err = nil
	m.encima = ""
	m.aviso = ""

	switch a {
	case accSalir:
		return m, tea.Quit

	case accAyuda:
		m.pantalla = pantAyuda
		m.titulo = "Ayuda"
		m.resultado = ""
		m.nota = ""
		m.aSalvo = true // la ayuda no se pierde: está siempre ahí
		return m, nil

	case accGenerar:
		p, err := cripto.Generar(cripto.AlfHex, 24)
		m.pantalla = pantResultado
		m.titulo = "Contraseña · hexadecimal · 192 bits"
		m.resultado, m.err = p, err
		m.nota = "Segura dentro de una URL"
		m.aviso = "Si cierras sin guardarla ni copiarla, esta contraseña se pierde"
		m.aSalvo = false
		return m, nil
	}

	m.modo = modoTexto
	m.construirCampos(false)
	m.pantalla = pantFormulario
	m.foco = 0
	m.desplazado, m.desplazadoAMano = 0, false
	m.campos[0].Focus()
	return m, textinput.Blink
}

// construirCampos arma el formulario según la acción y el modo.
//
// La clave solo se conserva al alternar entre texto y fichero, que es la misma
// operación cambiando de sitio de dónde sale el contenido. Al empezar otra cosa
// se descarta: arrastrar la clave de un cifrado al descifrado siguiente es un
// error esperando a pasar, y además la deja más rato en memoria de lo necesario.
func (m *modelo) construirCampos(conservarClave bool) {
	clave, repetir := "", ""
	if conservarClave {
		if len(m.campos) > campoClave {
			clave = m.campos[campoClave].Value()
		}
		if len(m.campos) > campoRepetir {
			repetir = m.campos[campoRepetir].Value()
		}
	}

	primero := "Una contraseña, un token, lo que sea"
	if m.accion == accDescifrar {
		primero = "Pega aquí el ESF1.… que te han pasado"
	}
	if m.modo == modoFichero {
		primero = "Arrastra el fichero hasta aquí, o escribe su ruta"
	}

	campos := []textinput.Model{
		m.campo(primero, false),
		m.campo("La que tendrá que usar quien lo abra", true),
	}
	if m.accion == accCifrar {
		campos = append(campos, m.campo("", true))
	}

	campos[campoClave].SetValue(clave)
	if len(campos) > campoRepetir {
		campos[campoRepetir].SetValue(repetir)
	}
	m.campos = campos
}

func (m modelo) cambiarModo(nuevo modo) (tea.Model, tea.Cmd) {
	if m.modo == nuevo {
		return m, nil
	}
	m.modo = nuevo
	m.err = nil
	m.construirCampos(true)
	m.foco = 0
	return m.enfocar()
}

func (m modelo) alMenu() (tea.Model, tea.Cmd) {
	m.pantalla = pantMenu
	m.desplazado, m.desplazadoAMano = 0, false
	m.resultado, m.err, m.nota, m.aviso = "", nil, "", ""
	m.encima = ""
	m.aSalvo = false
	m.trasConfirmar = nil
	// Los campos se sueltan al volver al menú para que la clave no siga viva en
	// memoria más tiempo del que dura la operación.
	m.campos = nil
	return m, nil
}

// salirDelResultado se interpone cuando hay algo en pantalla que no está en
// ningún otro sitio. Cerrar y perder un secreto recién cifrado, sin más aviso
// que el propio cierre, es la clase de cosa que solo se descubre cuando ya no
// tiene arreglo.
func (m modelo) salirDelResultado(luego func(modelo) (tea.Model, tea.Cmd)) (tea.Model, tea.Cmd) {
	if m.pantalla != pantResultado || m.aSalvo || m.resultado == "" || m.err != nil {
		return luego(m)
	}
	m.trasConfirmar = luego
	m.pantalla = pantConfirmar
	m.encima = ""
	return m, nil
}

func (m modelo) campo(etiqueta string, oculto bool) textinput.Model {
	t := textinput.New()
	t.Placeholder = etiqueta
	t.Prompt = ""
	t.CharLimit = 0
	t.Width = 52
	t.PromptStyle = m.e.Acento
	t.TextStyle = m.e.Cuerpo
	t.PlaceholderStyle = m.e.Apagado
	t.Cursor.Style = m.e.Acento
	if oculto {
		t.EchoMode = textinput.EchoPassword
		t.EchoCharacter = '•'
	}
	return t
}

func (m modelo) actualizarFormulario(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pantalla = pantMenu
		m.err = nil
		return m, nil

	case "ctrl+f":
		if m.modo == modoTexto {
			return m.cambiarModo(modoFichero)
		}
		return m.cambiarModo(modoTexto)

	case "tab", "down":
		m.foco = (m.foco + 1) % len(m.campos)
		return m.enfocar()
	case "shift+tab", "up":
		m.foco = (m.foco - 1 + len(m.campos)) % len(m.campos)
		return m.enfocar()

	case "enter":
		if m.foco < len(m.campos)-1 {
			m.foco++
			return m.enfocar()
		}
		return m.ejecutar()
	}
	return m.propagarACampos(msg)
}

func (m modelo) enfocar() (tea.Model, tea.Cmd) {
	// Al cambiar de campo vuelve a mandar el ancla, que es lo que garantiza que
	// el campo recién enfocado se vea.
	m.desplazadoAMano = false
	for i := range m.campos {
		if i == m.foco {
			m.campos[i].Focus()
		} else {
			m.campos[i].Blur()
		}
	}
	return m, textinput.Blink
}

func (m modelo) propagarACampos(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.campos) == 0 {
		return m, nil
	}
	var cmd tea.Cmd
	m.campos[m.foco], cmd = m.campos[m.foco].Update(msg)
	return m, cmd
}

func (m modelo) ejecutar() (tea.Model, tea.Cmd) {
	if m.pantalla != pantFormulario || len(m.campos) == 0 {
		return m, nil
	}

	entrada := m.campos[campoSecreto].Value()
	clave := m.campos[campoClave].Value()

	if strings.TrimSpace(entrada) == "" {
		m.err = fmt.Errorf("Falta el contenido")
		if m.modo == modoFichero {
			m.err = fmt.Errorf("Falta el fichero")
		}
		return m, nil
	}
	if clave == "" {
		m.err = fmt.Errorf("Falta la clave")
		return m, nil
	}
	if m.accion == accCifrar && clave != m.campos[campoRepetir].Value() {
		m.err = fmt.Errorf("Las dos claves no coinciden")
		return m, nil
	}

	m.trabajando = true
	m.giro = 0
	m.err = nil

	accionElegida, modoElegido := m.accion, m.modo

	trabajo := func() tea.Msg {
		k := []byte(clave)
		defer cripto.Borrar(k)

		if modoElegido == modoFichero {
			return trabajarFichero(accionElegida, entrada, k)
		}
		return trabajarTexto(accionElegida, entrada, k)
	}
	return m, tea.Batch(trabajo, girar())
}

func trabajarTexto(a accion, entrada string, k []byte) tea.Msg {
	if a == accCifrar {
		texto, err := cripto.SellarTexto([]byte(entrada), k, cripto.PerfilInteractivo)
		return listoMsg{
			texto:      texto,
			titulo:     "Cifrado",
			nota:       "Cópialo entero, con el prefijo ESF1.",
			aviso:      "Sin la clave, esto no lo abre nadie: si la pierdes, se pierde el contenido",
			copiarSolo: true,
			err:        err,
		}
	}
	datos, err := cripto.AbrirTexto(entrada, k)
	return listoMsg{
		texto:  string(datos),
		titulo: "Descifrado",
		nota:   "No lo dejes en pantalla más de lo necesario",
		err:    err,
	}
}

// trabajarFichero deja el resultado en disco, así que lo que sale por pantalla es
// la ruta y no el contenido. Y va marcado como a salvo: ya está guardado.
func trabajarFichero(a accion, ruta string, k []byte) tea.Msg {
	if a == accCifrar {
		destino, err := CifrarFichero(ruta, k)
		if err != nil {
			return listoMsg{err: err, titulo: "Cifrar un fichero"}
		}
		abs, _ := filepath.Abs(destino)
		return listoMsg{
			texto:  abs,
			titulo: "Fichero cifrado",
			nota:   "El original sigue donde estaba, sin tocar",
			aviso:  "Sin la clave, este fichero no lo abre nadie",
			aSalvo: true,
		}
	}

	destino, err := DescifrarFichero(ruta, k)
	if err != nil {
		return listoMsg{err: err, titulo: "Descifrar un fichero"}
	}
	abs, _ := filepath.Abs(destino)
	return listoMsg{
		texto:  abs,
		titulo: "Fichero descifrado",
		nota:   "Guardado con permisos 600: solo lo lee tu usuario",
		aSalvo: true,
	}
}

func (m modelo) actualizarResultado(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		return m.salirDelResultado(func(mm modelo) (tea.Model, tea.Cmd) { return mm.alMenu() })
	case "q":
		return m.salirDelResultado(func(mm modelo) (tea.Model, tea.Cmd) { return mm, tea.Quit })
	case "g", "s":
		return m.guardar()
	case "c":
		return m.copiar()
	case "r":
		if m.accion == accGenerar {
			return m.elegir(accGenerar)
		}
	}
	return m, nil
}

func (m modelo) actualizarAyuda(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		return m.alMenu()
	case "g", "s":
		return m.guardarAyuda()
	}
	return m, nil
}

// guardarAyuda deja la ayuda en un fichero de texto, para quien prefiera leerla
// fuera o mandársela a alguien.
func (m modelo) guardarAyuda() (tea.Model, tea.Cmd) {
	ruta, err := escribirSinPisar(CarpetaDeDescargas(), "esfinge-ayuda", []byte(textoAyuda))
	if err != nil {
		m.err = err
		return m, nil
	}
	m.exito = "Guardado en " + ruta
	return m, nil
}

func (m modelo) actualizarConfirmacion(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pantalla = pantResultado
		return m, nil
	case "g", "s":
		return m.activar(zonaGuardaYSal)
	case "q", "enter":
		return m.activar(zonaSalirYa)
	}
	return m, nil
}

// copiar deja el resultado en el portapapeles.
func (m modelo) copiar() (tea.Model, tea.Cmd) {
	if m.resultado == "" || m.err != nil {
		return m, nil
	}

	// El aviso a OSC 52 se cuela en el próximo dibujado; el registro es lo único
	// que sobrevive entre Update y View.
	m.zonas.osc = m.resultado

	if err := Copiar(m.resultado); err != nil {
		m.exito = ""
		m.nota = "No he podido usar el portapapeles del sistema; usa Guardar en un fichero"
		return m, nil
	}
	m.exito = "Copiado al portapapeles · ya lo puedes pegar donde haga falta"
	m.aSalvo = true
	return m, nil
}

// guardar deja el resultado en un fichero.
//
// El nombre lleva la fecha y la hora porque el anterior era fijo y cada guardado
// se comía el de antes sin avisar. Y se dice la ruta completa: con doble clic en
// el instalador el directorio de trabajo es la carpeta personal, no la carpeta
// donde está Esfinge, y sin la ruta entera el fichero parece no haberse creado.
func (m modelo) guardar() (tea.Model, tea.Cmd) {
	if m.resultado == "" || m.err != nil {
		return m, nil
	}

	prefijo := "esfinge-cifrado"
	switch m.accion {
	case accDescifrar:
		prefijo = "esfinge-descifrado"
	case accGenerar:
		prefijo = "esfinge-contrasena"
	case accAyuda:
		prefijo = "esfinge-ayuda"
	}

	ruta, err := escribirSinPisar(CarpetaDeDescargas(), prefijo, []byte(m.resultado+"\n"))
	if err != nil {
		m.err = fmt.Errorf("No he podido guardar el fichero: %w", err)
		return m, nil
	}
	m.exito = "Guardado en " + ruta
	m.nota = ""
	m.aSalvo = true
	return m, nil
}

// escribirSinPisar crea un fichero nuevo y devuelve el nombre que le ha tocado.
//
// La marca de tiempo llega al segundo, así que dos guardados seguidos caerían en
// el mismo nombre y el segundo se comería al primero. Cuando el nombre ya está
// cogido se prueba con un número detrás, y la creación es exclusiva para que ni
// siquiera dos Esfinges a la vez puedan pisarse.
func escribirSinPisar(carpeta, prefijo string, datos []byte) (string, error) {
	sello := time.Now().Format("20060102-150405")

	for i := 0; i < 100; i++ {
		nombre := filepath.Join(carpeta, fmt.Sprintf("%s-%s.txt", prefijo, sello))
		if i > 0 {
			nombre = filepath.Join(carpeta, fmt.Sprintf("%s-%s-%d.txt", prefijo, sello, i+1))
		}

		f, err := os.OpenFile(nombre, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		defer f.Close()

		if _, err := f.Write(datos); err != nil {
			return "", err
		}
		return nombre, nil
	}
	return "", fmt.Errorf("Hay demasiados ficheros «%s-%s» en %s", prefijo, sello, carpeta)
}
