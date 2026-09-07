package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/webcafeina/esfinge/internal/ui"
)

// zonaDe encuentra dónde ha quedado dibujado un elemento pulsable. Devuelve un
// punto que cae dentro de él, que es lo que se le pasaría al ratón.
func zonaDe(t *testing.T, m modelo, id string) (x, y int) {
	t.Helper()
	_ = m.View() // dibujar es lo que rellena el registro de zonas

	for _, z := range m.zonas.zonas {
		if z.id == id {
			x = z.col1
			if z.col2 < 9999 {
				x = (z.col1 + z.col2) / 2
			}
			return x, z.fila
		}
	}
	t.Fatalf("no hay ninguna zona con el identificador %q", id)
	return 0, 0
}

func clic(t *testing.T, m modelo, id string) (modelo, tea.Cmd) {
	t.Helper()
	x, y := zonaDe(t, m, id)
	sig, cmd := m.Update(tea.MouseMsg{
		X: x, Y: y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	return sig.(modelo), cmd
}

func mover(t *testing.T, m modelo, id string) modelo {
	t.Helper()
	x, y := zonaDe(t, m, id)
	sig, _ := m.Update(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion})
	return sig.(modelo)
}

func TestClicEnElMenu(t *testing.T) {
	m := nuevo(t)

	// La tercera entrada es «Generar contraseña».
	m, _ = clic(t, m, "menu:2")
	if m.pantalla != pantResultado {
		t.Fatalf("un clic en «Generar» dejó la pantalla en %v", m.pantalla)
	}
	if len(m.resultado) != 48 {
		t.Errorf("no ha generado una contraseña: %q", m.resultado)
	}
}

func TestClicAbreElFormulario(t *testing.T) {
	m := nuevo(t)
	m, _ = clic(t, m, "menu:0") // Cifrar
	if m.pantalla != pantFormulario {
		t.Fatal("un clic en «Cifrar» no abrió el formulario")
	}
}

func TestElPunteroResalta(t *testing.T) {
	m := nuevo(t)
	if m.encima != "" {
		t.Fatal("empieza con algo resaltado")
	}

	m = mover(t, m, "menu:3")
	if m.encima != "menu:3" {
		t.Errorf("el puntero sobre la cuarta entrada dejó encima=%q", m.encima)
	}

	// Y fuera de cualquier zona no resalta nada.
	sig, _ := m.Update(tea.MouseMsg{X: 0, Y: 0, Action: tea.MouseActionMotion})
	if sig.(modelo).encima != "" {
		t.Error("resalta algo con el puntero en una esquina vacía")
	}
}

func TestClicEnLosBotonesDelFormulario(t *testing.T) {
	m := nuevo(t)
	m, _ = clic(t, m, "menu:0")

	// Volver, con el ratón.
	m, _ = clic(t, m, zonaVolver)
	if m.pantalla != pantMenu {
		t.Error("el botón Volver no llevó al menú")
	}

	// Y el botón de cifrar con los campos vacíos avisa, no revienta.
	m, _ = clic(t, m, "menu:0")
	m, cmd := clic(t, m, zonaOK)
	if cmd != nil {
		t.Error("ha lanzado el cifrado con los campos vacíos")
	}
	if m.err == nil {
		t.Error("no ha avisado de que falta el contenido")
	}
}

func TestClicEnfocaUnCampo(t *testing.T) {
	m := nuevo(t)
	m, _ = clic(t, m, "menu:0") // Cifrar, tres campos
	if m.foco != 0 {
		t.Fatalf("el foco empieza en %d", m.foco)
	}

	m, _ = clic(t, m, "campo:2")
	if m.foco != 2 {
		t.Errorf("un clic en el tercer campo dejó el foco en %d", m.foco)
	}
	if !m.campos[2].Focused() {
		t.Error("el campo no quedó enfocado de verdad")
	}
}

// TestGuardarConElBoton es el fallo que se coló: la pantalla de resultado decía
// «G guardar» y quien pulsaba otra tecla volvía al menú sin que se escribiera
// nada. Ahora hay un botón, y esto comprueba que el fichero acaba existiendo.
func TestGuardarConElBoton(t *testing.T) {
	dir := descargasDePrueba(t)

	m := nuevo(t)
	m, _ = clic(t, m, "menu:2") // Generar
	contrasena := m.resultado

	m, _ = clic(t, m, zonaGuarda)
	if m.err != nil {
		t.Fatalf("al guardar: %v", m.err)
	}

	ficheros, err := filepath.Glob(filepath.Join(dir, "esfinge-contrasena-*.txt"))
	if err != nil || len(ficheros) != 1 {
		t.Fatalf("esperaba un fichero guardado, encontré %v (%v)", ficheros, err)
	}

	contenido, err := os.ReadFile(ficheros[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(contenido)) != contrasena {
		t.Errorf("el fichero tiene %q y esperaba %q", contenido, contrasena)
	}

	// Un secreto no puede quedar legible para el resto de la máquina.
	info, err := os.Stat(ficheros[0])
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permisos %o, esperaba 600", perm)
	}

	// Y la nota tiene que decir dónde ha ido a parar, con la ruta entera: al
	// abrir Esfinge con doble clic, el directorio de trabajo no es el que la
	// persona cree.
	if !strings.Contains(m.exito, dir) {
		t.Errorf("la confirmación no dice dónde está el fichero: %q", m.exito)
	}
	if !strings.Contains(m.View(), "Guardado en") {
		t.Error("la pantalla no confirma el guardado")
	}
}

// Guardar dos veces no puede pisar el fichero anterior.
func TestGuardarDosVecesNoPisa(t *testing.T) {
	dir := descargasDePrueba(t)

	m := nuevo(t)
	m, _ = clic(t, m, "menu:2")
	m, _ = clic(t, m, zonaGuarda)
	m, _ = clic(t, m, zonaOtra) // otra contraseña
	m, _ = clic(t, m, zonaGuarda)

	ficheros, _ := filepath.Glob(filepath.Join(dir, "esfinge-contrasena-*.txt"))
	if len(ficheros) != 2 {
		t.Errorf("esperaba dos ficheros, hay %d: %v", len(ficheros), ficheros)
	}
}

func TestGuardarConLaTecla(t *testing.T) {
	dir := descargasDePrueba(t)

	m := nuevo(t)
	m, _ = clic(t, m, "menu:2")
	m, _ = tecla(t, m, "g")

	ficheros, _ := filepath.Glob(filepath.Join(dir, "esfinge-*.txt"))
	if len(ficheros) != 1 {
		t.Errorf("la tecla G no guardó nada: %v", ficheros)
	}
}

// El botón de guardar sale en la pantalla de resultado, que es donde hay algo
// que guardar, y no en las demás.
func TestElBotonDeGuardarSoloSaleDondeToca(t *testing.T) {
	m := nuevo(t)
	if v := m.View(); strings.Contains(v, "Guardar en un fichero") {
		t.Error("el menú ofrece guardar y no hay nada que guardar")
	}

	m, _ = clic(t, m, "menu:2")
	if v := m.View(); !strings.Contains(v, "Guardar en un fichero") {
		t.Error("la pantalla de resultado no ofrece guardar")
	}
}

func TestRatonYTecladoHacenLoMismo(t *testing.T) {
	conRaton := nuevo(t)
	conRaton, _ = clic(t, conRaton, "menu:0")

	conTeclado := nuevo(t)
	conTeclado, _ = tecla(t, conTeclado, "enter")

	if conRaton.pantalla != conTeclado.pantalla || conRaton.accion != conTeclado.accion {
		t.Errorf("el ratón deja la interfaz en %v/%v y el teclado en %v/%v",
			conRaton.pantalla, conRaton.accion, conTeclado.pantalla, conTeclado.accion)
	}
}

// Las zonas se recalculan en cada dibujado: si se acumularan, un clic acabaría
// activando un botón de una pantalla anterior.
func TestLasZonasNoSeAcumulan(t *testing.T) {
	m := nuevo(t)
	_ = m.View()
	primera := len(m.zonas.zonas)

	for i := 0; i < 5; i++ {
		_ = m.View()
	}
	if len(m.zonas.zonas) != primera {
		t.Errorf("tras seis dibujados hay %d zonas y al principio había %d",
			len(m.zonas.zonas), primera)
	}
}

func TestSePintaEnLosDosTemasConRaton(t *testing.T) {
	for _, tema := range []ui.Tema{ui.TemaOscuro, ui.TemaClaro} {
		t.Run(tema.Nombre, func(t *testing.T) {
			m := Nuevo(ui.NuevosEstilos(tema), "prueba").(modelo)
			sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 34})
			mm := sig.(modelo)
			mm, _ = clic(t, mm, "menu:2")
			if !strings.Contains(mm.View(), "Guardar en un fichero") {
				t.Error("falta el botón de guardar")
			}
		})
	}
}

