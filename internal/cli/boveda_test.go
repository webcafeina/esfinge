package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/boveda"
)

// **Cuando la búsqueda encaja con varias, no se adivina.** Es la regla que
// separa una herramienta de tubería de una trampa: sacar la contraseña
// equivocada por la salida estándar es peor que no sacar ninguna, porque el
// fallo aparece al otro lado y sin nada que lo explique.
//
// Vale para «ver» y para «codigo», que salen las dos de aquí.
func TestLaBusquedaDeLaLineaDeComandosNoAdivina(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	b, _, err := boveda.Crear(ruta, "una contraseña maestra larga")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []boveda.Entrada{
		{Titulo: "Banco del norte", Secreto: "una"},
		{Titulo: "Banco del sur", Secreto: "otra"},
		{Titulo: "Correo", Secreto: "tercera", TOTP: "GEZDGNBVGY3TQOJQ"},
	} {
		if err := b.Poner(e); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := unaSola(b, "banco"); err == nil {
		t.Error("con dos candidatas ha elegido una")
	} else if !strings.Contains(err.Error(), "afina la búsqueda") {
		t.Errorf("no dice qué hacer: %v", err)
	}

	if _, err := unaSola(b, "esto no está"); err == nil {
		t.Error("ha encontrado algo que no existe")
	}

	// Y con una sola, la entrada **entera**: la búsqueda devuelve la lista sin
	// secretos, así que si no se pidiera después la entrada por su identificador
	// esto sacaría una contraseña vacía por la tubería sin decir nada.
	e, err := unaSola(b, "correo")
	if err != nil {
		t.Fatal(err)
	}
	if e.Secreto != "tercera" || e.TOTP == "" {
		t.Errorf("la entrada llega sin sus secretos: %+v", e)
	}
}
