package app

import "testing"

// Sin vidrio, la interfaz tiene que dejar la ventana como siempre: un fondo
// transparente sin nada detrás no enseña el escritorio, enseña un agujero. Por
// eso el valor por defecto importa.
func TestSinVidrioLaVentanaSePintaComoSiempre(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	if a.Vidrio() {
		t.Error("de fábrica no hay vidrio, y dice que sí")
	}

	MarcarVidrio(a, true)
	if !a.Vidrio() {
		t.Error("se ha marcado el vidrio y sigue diciendo que no")
	}
}
