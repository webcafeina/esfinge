package tui

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/webcafeina/esfinge/internal/ui"
)

func TestPedirAltura(t *testing.T) {
	var b bytes.Buffer
	PedirAltura(&b, 30)
	if got := b.String(); got != "\x1b[8;30;0t" {
		t.Errorf("la petición es %q", got)
	}

	// Un número absurdo no se manda.
	b.Reset()
	PedirAltura(&b, 0)
	if b.Len() != 0 {
		t.Errorf("ha mandado algo con cero filas: %q", b.String())
	}
}

// TestAltoComodoBastaParaTodo comprueba que la altura que se le pide al terminal
// da de verdad para la pantalla más alta.
//
// Es la clase de constante que se queda obsoleta en cuanto alguien añade un
// campo o una línea de aviso, y entonces el desplazamiento reaparece sin que
// nadie se entere. Con esto, quien la deje corta se entera al pasar los tests.
func TestAltoComodoBastaParaTodo(t *testing.T) {
	descargasDePrueba(t)

	casos := []struct {
		nombre   string
		preparar func(modelo) modelo
	}{
		{"menú", func(m modelo) modelo { return m }},
		{"cifrar con el medidor a la vista", func(m modelo) modelo {
			s, _ := m.elegir(accCifrar)
			mm := escribir(t, s.(modelo), "un secreto")
			mm, _ = tecla(t, mm, "enter")
			return escribir(t, mm, "una clave")
		}},
		{"cifrar en modo fichero", func(m modelo) modelo {
			s, _ := m.elegir(accCifrar)
			s2, _ := s.(modelo).cambiarModo(modoFichero)
			return s2.(modelo)
		}},
		{"descifrar", func(m modelo) modelo { s, _ := m.elegir(accDescifrar); return s.(modelo) }},
		{"ayuda", func(m modelo) modelo { s, _ := m.elegir(accAyuda); return s.(modelo) }},
		{"contraseña generada", func(m modelo) modelo { s, _ := m.elegir(accGenerar); return s.(modelo) }},
		{"confirmación al salir", func(m modelo) modelo {
			s, _ := m.elegir(accGenerar)
			s2, _ := s.(modelo).salirDelResultado(func(mm modelo) (tea.Model, tea.Cmd) { return mm.alMenu() })
			return s2.(modelo)
		}},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "1.0.0").(modelo)
			sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: AltoComodo})
			mm := c.preparar(sig.(modelo))
			_ = mm.View()

			if mm.zonas.arriba != 0 || mm.zonas.abajo != 0 {
				t.Errorf("con %d filas quedan %d líneas ocultas arriba y %d abajo: sube AltoComodo",
					AltoComodo, mm.zonas.arriba, mm.zonas.abajo)
			}
		})
	}
}

// Y con un error en pantalla, que añade una línea al pie.
func TestAltoComodoConUnErrorDelante(t *testing.T) {
	m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "1.0.0").(modelo)
	sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: AltoComodo})
	mm := sig.(modelo)

	sig, _ = mm.elegir(accCifrar)
	mm = sig.(modelo)
	mm = escribir(t, mm, "secreto")
	mm, _ = tecla(t, mm, "enter")
	mm = escribir(t, mm, "una")
	mm, _ = tecla(t, mm, "enter")
	mm = escribir(t, mm, "otra")
	sig, _ = mm.ejecutar() // las claves no coinciden
	mm = sig.(modelo)

	if mm.err == nil {
		t.Fatal("no ha dado el error esperado")
	}
	_ = mm.View()
	if mm.zonas.abajo != 0 {
		t.Errorf("con un error en pantalla quedan %d líneas ocultas", mm.zonas.abajo)
	}
}
