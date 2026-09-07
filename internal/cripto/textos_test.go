package cripto

import (
	"testing"
	"unicode"
)

// empiezaEnMayuscula acepta también lo que no empieza por letra —una ruta, un
// prefijo como ESF1— porque ahí no hay nada que capitalizar.
func empiezaEnMayuscula(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return true
		}
		return unicode.IsUpper(r)
	}
	return true
}

// TestLosTextosEmpiezanEnMayuscula vigila una decisión de estilo que se toma
// una vez y luego se olvida: todo lo que lee una persona empieza con mayúscula,
// aunque sea una sola palabra.
//
// Va contra la costumbre de Go, donde los errores se escriben en minúscula para
// poder encadenarlos con %w. Aquí manda lo que se ve en pantalla, porque estos
// errores se le enseñan tal cual a quien usa el programa.
func TestLosTextosEmpiezanEnMayuscula(t *testing.T) {
	errores := map[string]error{
		"ErrClaveIncorrecta": ErrClaveIncorrecta,
		"ErrFormato":         ErrFormato,
		"ErrTruncado":        ErrTruncado,
		"ErrDanado":          ErrDanado,
		"ErrVersion":         ErrVersion,
	}
	for nombre, err := range errores {
		if !empiezaEnMayuscula(err.Error()) {
			t.Errorf("%s empieza en minúscula: %q", nombre, err)
		}
	}

	// Las valoraciones de la fuerza de una clave se pintan en el medidor.
	claves := []string{"", "1234", "hunter2", "Tr0ub4dor&3", "caballo grapa batería correcto"}
	for _, c := range claves {
		f := Evaluar(c)
		if !empiezaEnMayuscula(f.Etiqueta) {
			t.Errorf("la etiqueta de %q empieza en minúscula: %q", c, f.Etiqueta)
		}
		if f.Sugerencia != "" && !empiezaEnMayuscula(f.Sugerencia) {
			t.Errorf("la sugerencia de %q empieza en minúscula: %q", c, f.Sugerencia)
		}
	}

	// Y el aviso de los alfabetos que no valen dentro de una URL.
	for nombre, a := range Alfabetos {
		if a.Aviso != "" && !empiezaEnMayuscula(a.Aviso) {
			t.Errorf("el aviso de %q empieza en minúscula: %q", nombre, a.Aviso)
		}
	}

	// Los errores que devuelve el generador.
	for _, caso := range []int{4, 1000} {
		if _, err := Generar(AlfHex, caso); err != nil && !empiezaEnMayuscula(err.Error()) {
			t.Errorf("Generar(%d) devuelve un error en minúscula: %q", caso, err)
		}
	}
}
