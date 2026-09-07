package cripto

import "testing"

// Los dos formatos empiezan por «ESF1», así que la magia sola no vale para
// distinguirlos. De acertar aquí depende que abrir un .esf salga en la pantalla
// que toca.
func TestDistinguirLosDosFormatos(t *testing.T) {
	texto, err := SellarTexto([]byte("un secreto"), []byte("clave"), pruebas)
	if err != nil {
		t.Fatal(err)
	}
	binario, err := Sellar([]byte("un secreto"), []byte("clave"), pruebas)
	if err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre string
		datos  []byte
		quiero Forma
	}{
		{"el de texto", []byte(texto), FormaTexto},
		{"el de texto con espacios delante", []byte("\n  " + texto), FormaTexto},
		{"el binario", binario, FormaBinaria},
		{"un fichero cualquiera", []byte("DATABASE_URL=postgres://u:p@h/db"), FormaDesconocida},
		{"vacío", nil, FormaDesconocida},
		{"solo la magia", []byte("ESF1"), FormaDesconocida},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if tengo := FormaDe(c.datos); tengo != c.quiero {
				t.Errorf("quiero %v, tengo %v", c.quiero, tengo)
			}
		})
	}
}
