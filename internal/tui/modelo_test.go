package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/webcafeina/esfinge/internal/ui"
)

func nuevo(t *testing.T) modelo {
	t.Helper()
	m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "prueba").(modelo)
	sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
	return sig.(modelo)
}

func tecla(t *testing.T, m modelo, k string) (modelo, tea.Cmd) {
	t.Helper()
	var msg tea.KeyMsg
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	sig, cmd := m.Update(msg)
	return sig.(modelo), cmd
}

// resultadoDe ejecuta el comando que devuelve el formulario y saca el mensaje
// con el resultado. Hace falta desenvolver el lote porque, además del trabajo
// criptográfico, ahora va el latido del indicador de progreso.
func resultadoDe(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("no ha lanzado ningún comando")
	}

	msg := cmd()
	lote, esLote := msg.(tea.BatchMsg)
	if !esLote {
		return msg
	}
	for _, c := range lote {
		if sub, ok := c().(listoMsg); ok {
			return sub
		}
	}
	t.Fatal("el lote de comandos no traía ningún resultado")
	return nil
}

func escribir(t *testing.T, m modelo, texto string) modelo {
	t.Helper()
	for _, r := range texto {
		sig, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = sig.(modelo)
	}
	return m
}

func TestMenuSePinta(t *testing.T) {
	v := nuevo(t).View()
	// La marca va en la barra de arriba con el tracking de ClickHouse, así que
	// en pantalla es «W E B C A F E Í N A».
	for _, quiero := range []string{ui.Mayusculas(ui.Wordmark), "Cifrar", "Descifrar", "Generar", "Salir"} {
		if !strings.Contains(v, quiero) {
			t.Errorf("el menú no contiene %q", quiero)
		}
	}
}

func TestMenuNavega(t *testing.T) {
	m := nuevo(t)
	if m.cursor != 0 {
		t.Fatalf("el cursor empieza en %d", m.cursor)
	}

	m, _ = tecla(t, m, "down")
	m, _ = tecla(t, m, "down")
	if m.cursor != 2 {
		t.Errorf("tras dos «abajo» el cursor está en %d", m.cursor)
	}

	m, _ = tecla(t, m, "up")
	if m.cursor != 1 {
		t.Errorf("tras subir, el cursor está en %d", m.cursor)
	}

	// No se sale por los extremos.
	for i := 0; i < 10; i++ {
		m, _ = tecla(t, m, "up")
	}
	if m.cursor != 0 {
		t.Errorf("el cursor se ha ido por arriba hasta %d", m.cursor)
	}
	for i := 0; i < 20; i++ {
		m, _ = tecla(t, m, "down")
	}
	if m.cursor != len(menu)-1 {
		t.Errorf("el cursor se ha ido por abajo hasta %d", m.cursor)
	}
}

// TestCicloCompleto es la prueba que importa: cifrar desde los menús y descifrar
// lo que salga, con el mismo camino que recorre una persona.
func TestCicloCompleto(t *testing.T) {
	const secreto = "postgres://u:pa55@host/db"
	const clave = "una clave de prueba"

	m := nuevo(t)
	m, _ = tecla(t, m, "enter") // Cifrar
	if m.pantalla != pantFormulario {
		t.Fatal("no ha entrado en el formulario")
	}

	m = escribir(t, m, secreto)
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, clave)
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, clave)
	m, cmd := tecla(t, m, "enter")

	if !m.trabajando {
		t.Error("no ha marcado que está trabajando")
	}

	sig, _ := m.Update(resultadoDe(t, cmd))
	m = sig.(modelo)

	if m.err != nil {
		t.Fatalf("al cifrar: %v", m.err)
	}
	if m.pantalla != pantResultado {
		t.Fatal("no ha llegado a la pantalla de resultado")
	}
	if !strings.HasPrefix(m.resultado, "ESF1.") {
		t.Fatalf("el resultado no es un contenedor: %q", m.resultado)
	}
	cifrado := m.resultado

	if v := m.View(); !strings.Contains(v, "Guardar en un fichero") {
		t.Error("la pantalla de resultado no ofrece el botón de guardar")
	}

	// Y ahora la vuelta. Volver desde un resultado sin guardar pregunta antes,
	// así que hay que pasar por la confirmación.
	m, _ = tecla(t, m, "enter") // pide confirmación
	if m.pantalla != pantConfirmar {
		t.Fatal("no ha preguntado antes de perder el resultado")
	}
	m, _ = tecla(t, m, "enter") // salir sin guardar
	if m.pantalla != pantMenu {
		t.Fatal("la confirmación no llevó al menú")
	}
	m, _ = tecla(t, m, "down")
	m, _ = tecla(t, m, "enter") // Descifrar

	m = escribir(t, m, cifrado)
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, clave)
	m, cmd = tecla(t, m, "enter")

	sig, _ = m.Update(resultadoDe(t, cmd))
	m = sig.(modelo)

	if m.err != nil {
		t.Fatalf("al descifrar: %v", m.err)
	}
	if m.resultado != secreto {
		t.Errorf("la vuelta dio %q y esperaba %q", m.resultado, secreto)
	}
}

func TestClavesQueNoCoinciden(t *testing.T) {
	m := nuevo(t)
	m, _ = tecla(t, m, "enter")

	m = escribir(t, m, "secreto")
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, "una")
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, "otra")
	m, cmd := tecla(t, m, "enter")

	if cmd != nil {
		t.Error("ha lanzado el cifrado con dos claves distintas")
	}
	if m.err == nil {
		t.Fatal("no ha avisado de que las claves no coinciden")
	}
	if !strings.Contains(m.View(), "no coinciden") {
		t.Error("el aviso no llega a la pantalla")
	}
}

func TestCamposVacios(t *testing.T) {
	m := nuevo(t)
	m, _ = tecla(t, m, "enter")

	// Directo a intentar cifrar sin escribir nada.
	m, _ = tecla(t, m, "enter")
	m, _ = tecla(t, m, "enter")
	m, cmd := tecla(t, m, "enter")

	if cmd != nil {
		t.Error("ha lanzado el cifrado con los campos vacíos")
	}
	if m.err == nil {
		t.Error("no ha avisado de que falta el contenido")
	}
}

func TestClaveIncorrectaAlDescifrar(t *testing.T) {
	m := nuevo(t)
	m, _ = tecla(t, m, "down")
	m, _ = tecla(t, m, "enter") // Descifrar

	m = escribir(t, m, "ESF1.esto-no-es-un-contenedor")
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, "cualquiera")
	m, cmd := tecla(t, m, "enter")

	sig, _ := m.Update(resultadoDe(t, cmd))
	m = sig.(modelo)

	if m.err == nil {
		t.Fatal("ha aceptado un contenedor inventado")
	}
	v := m.View()
	if !strings.Contains(v, "No ha podido ser") {
		t.Error("la pantalla de error no se pinta")
	}
	if !strings.Contains(v, ui.GlifoError) {
		t.Error("el error no lleva glifo, y sin color es lo único que lo distingue")
	}
	if !strings.Contains(v, "ESF1") {
		t.Error("el error no da la pista de qué se esperaba")
	}
}

func TestGenerarDesdeElMenu(t *testing.T) {
	m := nuevo(t)
	m, _ = tecla(t, m, "down")
	m, _ = tecla(t, m, "down")
	m, _ = tecla(t, m, "enter") // Generar

	if m.err != nil {
		t.Fatalf("al generar: %v", m.err)
	}
	if len(m.resultado) != 48 {
		t.Errorf("24 bytes en hexadecimal son 48 caracteres, tengo %d: %q", len(m.resultado), m.resultado)
	}
	primera := m.resultado

	m, _ = tecla(t, m, "r") // otra
	if m.resultado == primera {
		t.Error("«otra» ha devuelto la misma contraseña")
	}
}

func TestEscVuelveAlMenu(t *testing.T) {
	m := nuevo(t)
	m, _ = tecla(t, m, "enter")
	if m.pantalla != pantFormulario {
		t.Fatal("no ha entrado en el formulario")
	}
	m, _ = tecla(t, m, "esc")
	if m.pantalla != pantMenu {
		t.Error("esc no ha vuelto al menú")
	}
}

// La clave nunca se pinta en claro, ni en el campo ni en ninguna otra parte de
// la pantalla.
func TestLaClaveNoSeVe(t *testing.T) {
	const clave = "estaclavenodebeversejamas"

	m := nuevo(t)
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, "secreto")
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, clave)

	if strings.Contains(m.View(), clave) {
		t.Error("la clave se está pintando en claro")
	}
	if !strings.Contains(m.View(), "•") {
		t.Error("el campo de la clave no está enmascarado")
	}
}

// El medidor de fuerza aparece al teclear la clave, y no antes.
func TestMedidorDeFuerza(t *testing.T) {
	m := nuevo(t)
	m, _ = tecla(t, m, "enter")
	m = escribir(t, m, "secreto")
	m, _ = tecla(t, m, "enter")

	if strings.Contains(m.View(), "Muy débil") {
		t.Error("el medidor sale con el campo vacío")
	}

	m = escribir(t, m, "1234")
	if !strings.Contains(m.View(), "Muy débil") {
		t.Error("el medidor no valora una clave mala")
	}
}

// Con el tema claro tiene que pintarse todo igual: es el fallo que ya mordió una
// vez, un título en tinta oscura que desaparecía sobre fondo negro.
func TestSePintaEnLosDosTemas(t *testing.T) {
	for _, tema := range []ui.Tema{ui.TemaOscuro, ui.TemaClaro} {
		t.Run(tema.Nombre, func(t *testing.T) {
			m := Nuevo(ui.NuevosEstilos(tema), "prueba").(modelo)
			sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
			v := sig.(modelo).View()
			if !strings.Contains(v, ui.Mayusculas(ui.Wordmark)) {
				t.Error("falta la marca")
			}
			if !strings.Contains(v, "Cifrar") {
				t.Error("falta el menú")
			}
		})
	}
}

// Un terminal estrecho no debe romper la vista.
func TestTerminalEstrecho(t *testing.T) {
	for _, ancho := range []int{20, 40, 80, 200} {
		m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "prueba").(modelo)
		sig, _ := m.Update(tea.WindowSizeMsg{Width: ancho, Height: 24})
		if v := sig.(modelo).View(); v == "" {
			t.Errorf("con %d columnas la vista sale vacía", ancho)
		}
	}
}
