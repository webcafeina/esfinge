package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/webcafeina/esfinge/internal/ui"
)

func medida(m modelo) (int, string) {
	v := m.View()
	return len(strings.Split(v, "\n")), v
}

// TestLaVistaNuncaSePasaDelAlto es el test del fallo que hacía que el ratón no
// funcionara en la pantalla de cifrar.
//
// Cuando la vista tiene más líneas que la ventana, el terminal la desplaza hacia
// arriba para que quepa. Las coordenadas que manda el ratón siguen siendo las de
// la ventana, así que el mapa de zonas queda corrido y los clics dejan de caer
// donde tocan. Cifrar, con sus tres campos, se salía por abajo en un terminal de
// 24 filas; descifrar, con dos, cabía. De ahí que uno respondiera al ratón y el
// otro no.
func TestLaVistaNuncaSePasaDelAlto(t *testing.T) {
	altos := []int{16, 20, 24, 30, 40}
	anchos := []int{60, 90}

	for _, alto := range altos {
		for _, ancho := range anchos {
			for i, entrada := range menu {
				if entrada.accion == accSalir {
					continue
				}
				nombre := fmt.Sprintf("%dx%d/%s", ancho, alto, entrada.titulo)
				t.Run(nombre, func(t *testing.T) {
					m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "1.0.0").(modelo)
					sig, _ := m.Update(tea.WindowSizeMsg{Width: ancho, Height: alto})
					mm := sig.(modelo)

					// El menú.
					if n, _ := medida(mm); n > alto {
						t.Fatalf("el menú ocupa %d líneas en una ventana de %d", n, alto)
					}

					sig, _ = mm.elegir(entrada.accion)
					mm = sig.(modelo)
					if n, _ := medida(mm); n > alto {
						t.Fatalf("«%s» ocupa %d líneas en una ventana de %d", entrada.titulo, n, alto)
					}

					if mm.pantalla != pantFormulario {
						return
					}

					// Con los campos llenos, que es cuando aparece el medidor.
					mm = escribir(t, mm, "un secreto cualquiera")
					mm, _ = tecla(t, mm, "enter")
					mm = escribir(t, mm, "hunter2")
					if n, _ := medida(mm); n > alto {
						t.Fatalf("«%s» relleno ocupa %d líneas en una ventana de %d",
							entrada.titulo, n, alto)
					}

					// Y en modo fichero.
					sig, _ = mm.cambiarModo(modoFichero)
					if n, _ := medida(sig.(modelo)); n > alto {
						t.Fatalf("«%s» en modo fichero ocupa %d líneas en una ventana de %d",
							entrada.titulo, n, alto)
					}
					_ = i
				})
			}
		}
	}
}

// Y con la vista recortada, las zonas que quedan tienen que seguir cayendo
// dentro de la pantalla: una zona en una fila que no se dibuja es un clic que no
// llega nunca.
func TestLasZonasCaenDentroDeLaPantalla(t *testing.T) {
	for _, alto := range []int{16, 20, 24, 40} {
		m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "1.0.0").(modelo)
		sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: alto})
		mm := sig.(modelo)

		sig, _ = mm.elegir(accCifrar)
		mm = sig.(modelo)
		mm = escribir(t, mm, "secreto")
		mm, _ = tecla(t, mm, "enter")
		mm = escribir(t, mm, "clave")

		n, _ := medida(mm)
		for _, z := range mm.zonas.zonas {
			if z.fila >= n {
				t.Errorf("con alto %d, la zona %q está en la fila %d y la vista tiene %d líneas",
					alto, z.id, z.fila, n)
			}
		}
	}
}

// TestSePuedeCifrarConElRatonEnUnaVentanaPequena recorre con el ratón el camino
// entero en el tamaño de terminal que rompía: veinticuatro filas.
func TestSePuedeCifrarConElRatonEnUnaVentanaPequena(t *testing.T) {
	m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "1.0.0").(modelo)
	sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	mm := sig.(modelo)

	mm, _ = clic(t, mm, "menu:0") // Cifrar
	if mm.pantalla != pantFormulario {
		t.Fatal("el clic en Cifrar no abrió el formulario")
	}

	// Los campos que se ven se enfocan con un clic. En una ventana de 24 filas el
	// tercero cae fuera, y ahí lo que toca es que el contenido se desplace hasta
	// él —con el tabulador o con la rueda— y entonces sí se pueda pulsar.
	for i := 0; i < 3; i++ {
		if i == 2 {
			for mm.foco != 2 {
				mm, _ = tecla(t, mm, "tab")
			}
			_ = mm.View() // el ancla lo trae a la vista al dibujar
		}
		mm, _ = clic(t, mm, fmt.Sprintf("campo:%d", i))
		if mm.foco != i {
			t.Fatalf("el clic en el campo %d dejó el foco en %d", i, mm.foco)
		}
	}

	// Y el conmutador de modo, y los botones.
	mm, _ = clic(t, mm, zonaModoFich)
	if mm.modo != modoFichero {
		t.Error("el clic en Fichero no cambió de modo")
	}
	mm, _ = clic(t, mm, zonaModoTexto)
	if mm.modo != modoTexto {
		t.Error("el clic en Texto no cambió de modo")
	}

	mm, _ = clic(t, mm, zonaVolver)
	if mm.pantalla != pantMenu {
		t.Error("el clic en Volver no llevó al menú")
	}
}

// Los recordatorios del pie se ven como botones, así que tienen que responder al
// ratón como botones. Se comprueba pulsando la zona del pie por su posición, que
// es distinta de la del botón grande aunque compartan lo que hacen.
func TestElPieResponde(t *testing.T) {
	m := nuevo(t)
	m, _ = clic(t, m, "menu:3") // Ayuda, que no pregunta al salir

	// En la pantalla de ayuda el pie solo trae «Intro Volver». Se busca la zona
	// que esté en la última línea, que es la del pie y no la del botón.
	_ = m.View()
	var filaPie, colPie int = -1, 0
	for _, z := range m.zonas.zonas {
		if z.id == zonaVolver && z.fila > filaPie {
			filaPie, colPie = z.fila, (z.col1+z.col2)/2
		}
	}
	if filaPie < 0 {
		t.Fatal("el pie no registra ninguna zona pulsable")
	}

	sig, _ := m.Update(tea.MouseMsg{
		X: colPie, Y: filaPie,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	if sig.(modelo).pantalla != pantMenu {
		t.Errorf("el clic en el pie no ha llevado al menú: pantalla %v", sig.(modelo).pantalla)
	}
}

// El campo que tiene el foco no puede quedarse nunca fuera de la vista: escribir
// en algo que no se ve es peor que cualquier recorte.
func TestElCampoConFocoSiempreSeVe(t *testing.T) {
	for _, alto := range []int{18, 20, 24, 30} {
		m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "1.0.0").(modelo)
		sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: alto})
		mm := sig.(modelo)
		sig, _ = mm.elegir(accCifrar)
		mm = sig.(modelo)

		for campo := 0; campo < 3; campo++ {
			mm.foco = campo
			_ = mm.View()

			visto := false
			for _, z := range mm.zonas.zonas {
				if z.id == fmt.Sprintf("campo:%d", campo) {
					visto = true
					break
				}
			}
			if !visto {
				t.Errorf("con alto %d, el campo %d tiene el foco y no se ve", alto, campo)
			}
		}
	}
}

// La rueda del ratón desplaza el contenido, y no se pasa por los extremos.
func TestLaRuedaDesplaza(t *testing.T) {
	m := Nuevo(ui.NuevosEstilos(ui.TemaOscuro), "1.0.0").(modelo)
	sig, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 22})
	mm := sig.(modelo)
	sig, _ = mm.elegir(accCifrar)
	mm = sig.(modelo)
	_ = mm.View()

	if mm.zonas.abajo == 0 {
		t.Skip("en este tamaño cabe todo, no hay nada que desplazar")
	}

	rueda := func(m modelo, arriba bool) modelo {
		b := tea.MouseButtonWheelDown
		if arriba {
			b = tea.MouseButtonWheelUp
		}
		sig, _ := m.Update(tea.MouseMsg{Button: b, Action: tea.MouseActionPress})
		return sig.(modelo)
	}

	antes := mm.desplazado
	mm = rueda(mm, false)
	if mm.desplazado <= antes {
		t.Error("la rueda hacia abajo no ha desplazado")
	}

	for i := 0; i < 20; i++ {
		mm = rueda(mm, true)
	}
	if mm.desplazado != 0 {
		t.Errorf("la rueda hacia arriba se ha pasado del principio: %d", mm.desplazado)
	}

	// Y por abajo no se puede desplazar más allá del contenido.
	for i := 0; i < 50; i++ {
		mm = rueda(mm, false)
	}
	_ = mm.View()
	if mm.zonas.abajo != 0 {
		t.Errorf("tras desplazar hasta el fondo aún quedan %d líneas ocultas", mm.zonas.abajo)
	}
}
