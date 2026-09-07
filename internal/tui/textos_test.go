package tui

import (
	"testing"
	"unicode"
)

func empiezaEnMayuscula(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return true
		}
		return unicode.IsUpper(r)
	}
	return true
}

// Lo mismo que en el paquete cripto, pero para lo que se lee en los menús: cada
// entrada, su explicación y las pistas de los errores.
func TestLosTextosDeLaInterfazEmpiezanEnMayuscula(t *testing.T) {
	for _, e := range menu {
		if !empiezaEnMayuscula(e.titulo) {
			t.Errorf("la entrada de menú %q empieza en minúscula", e.titulo)
		}
		if !empiezaEnMayuscula(e.pie) {
			t.Errorf("la explicación de %q empieza en minúscula: %q", e.titulo, e.pie)
		}
	}

	// Los títulos y las notas de las tres pantallas de resultado.
	m := nuevo(t)
	for _, a := range []accion{accGenerar, accAyuda} {
		sig, _ := m.elegir(a)
		mm := sig.(modelo)
		if !empiezaEnMayuscula(mm.titulo) {
			t.Errorf("el título %q empieza en minúscula", mm.titulo)
		}
		if !empiezaEnMayuscula(mm.nota) {
			t.Errorf("la nota %q empieza en minúscula", mm.nota)
		}
	}

	// Los avisos del formulario cuando falta algo.
	mm := nuevo(t)
	sig, _ := mm.elegir(accCifrar)
	mm = sig.(modelo)
	sig, _ = mm.ejecutar()
	if err := sig.(modelo).err; err == nil || !empiezaEnMayuscula(err.Error()) {
		t.Errorf("el aviso de campos vacíos empieza en minúscula: %v", err)
	}
}
