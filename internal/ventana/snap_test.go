package ventana

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// **`comctl32` no se carga nunca con `NewLazySystemDLL`.**
//
// De esa palabra dependió que Esfinge abriera o no durante cinco versiones.
// `SetWindowSubclass` y `DefSubclassProc` sólo se exportan **por su nombre** en la
// versión 6 de esa biblioteca, que es la que trae el contexto de activación gracias
// al manifiesto. `NewLazySystemDLL` carga con `LOAD_LIBRARY_SEARCH_SYSTEM32`, que
// **se salta ese contexto** y trae la 5.82 de `System32`, donde esas dos están sólo
// por número: el `.Call()` no las encuentra y **entra en pánico**, así que la
// aplicación no llega a abrirse. Sin ventana, sin error y sin nada en el Visor de
// eventos, porque un pánico de Go no es un fallo de aplicación de Windows.
//
// Aquí no hay ningún Windows donde ejecutarlo —y las puertas sólo dicen que
// enlaza—, así que lo único que se puede vigilar es **lo que el fuente dice**.
func TestComctl32NoSeCargaDeSystem32(t *testing.T) {
	fuente, err := os.ReadFile("snap_windows.go")
	if err != nil {
		t.Fatalf("no se ha podido leer el fuente: %v", err)
	}

	// Se mira la línea que la declara, no el fichero entero: el comentario que
	// explica esto nombra las dos formas, y buscar a secas daría por roto un
	// fichero correcto. Es la trampa de la guarda que pregunta por la prosa.
	linea := regexp.MustCompile(`(?m)^\s*comctl32\s*=\s*windows\.(\w+)\(`)
	m := linea.FindSubmatch(fuente)
	if m == nil {
		t.Fatal("no se encuentra la línea que carga comctl32")
	}
	if cual := string(m[1]); cual != "NewLazyDLL" {
		t.Errorf("comctl32 se carga con %s; con eso SetWindowSubclass no existe y Esfinge no abre", cual)
	}
}

// **Y montar el menú de Snap no puede impedir que la ventana se abra.**
//
// Es una comodidad —el menú al pasar por encima de «Maximizar»— hecha de llamadas
// a Win32 por nombre, y una función que no esté entra en pánico. Esa capacidad no
// la puede tener algo que, cuando falla, lo único que se pierde es un menú.
func TestMontarNoPuedeTumbarElArranque(t *testing.T) {
	fuente, err := os.ReadFile("snap_windows.go")
	if err != nil {
		t.Fatalf("no se ha podido leer el fuente: %v", err)
	}
	cuerpo := string(fuente)
	i := strings.Index(cuerpo, "func Montar(")
	if i < 0 {
		t.Fatal("no se encuentra Montar")
	}
	j := strings.Index(cuerpo[i:], "\n}\n")
	if j < 0 {
		t.Fatal("no se encuentra el final de Montar")
	}
	if !strings.Contains(cuerpo[i:i+j], "recover()") {
		t.Error("Montar no recoge el pánico: un fallo del menú de Snap dejaría a Esfinge sin abrirse")
	}
}
