package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// Un fichero que llega antes de que la ventana esté escuchando no se puede
// mandar por evento: no hay nadie al otro lado. Se guarda y se entrega cuando
// pregunta. Éste es el caso de abrir Esfinge haciendo doble clic en un .esf.
func TestElFicheroQueLlegaAntesDeLaVentanaSeGuarda(t *testing.T) {
	a, s := nuevaDePrueba(t)

	a.AlAbrirCon("/tmp/secreto.esf")

	if len(s.avisosDe(EventoFicheroAbierto)) != 0 {
		t.Error("ha mandado un evento cuando todavía no había nadie escuchando")
	}

	ap := a.AperturaDeArranque()
	if len(ap.Rutas) != 1 || ap.Rutas[0] != "/tmp/secreto.esf" {
		t.Fatalf("quiero [/tmp/secreto.esf], tengo %v", ap.Rutas)
	}

	// Una sola vez: si se devolviera siempre, cambiar de pestaña repondría el
	// fichero una y otra vez.
	if segunda := a.AperturaDeArranque(); len(segunda.Rutas) != 0 {
		t.Errorf("los ha entregado dos veces: %v", segunda.Rutas)
	}
}

// Con la ventana ya abierta nadie va a volver a preguntar, así que hay que
// avisar. Éste es el caso de un doble clic mientras Esfinge corre, y es el que
// no hacía nada: el evento se emitía pero la interfaz no lo escuchaba.
func TestConLaVentanaAbiertaElFicheroLlegaPorEvento(t *testing.T) {
	a, s := nuevaDePrueba(t)

	a.AperturaDeArranque() // la ventana dice que ya está

	a.AlAbrirCon("/tmp/otro.esf")

	avisos := s.avisosDe(EventoFicheroAbierto)
	if len(avisos) != 1 || len(avisos[0].Rutas) != 1 || avisos[0].Rutas[0] != "/tmp/otro.esf" {
		t.Fatalf("quiero un aviso con /tmp/otro.esf, tengo %+v", avisos)
	}
	// Y no se queda además guardado, que lo abriría dos veces.
	if ap := a.AperturaDeArranque(); len(ap.Rutas) != 0 {
		t.Errorf("además lo ha guardado: %v", ap.Rutas)
	}
}

// macOS manda un evento por fichero, así que abrir varios de golpe son varias
// llamadas antes de que la ventana exista.
func TestVariosFicherosDeGolpe(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	a.AlAbrirCon("/tmp/uno.esf")
	a.AlAbrirCon("/tmp/dos.esf")

	if ap := a.AperturaDeArranque(); len(ap.Rutas) != 2 {
		t.Errorf("quiero dos ficheros, tengo %v", ap.Rutas)
	}
}

func TestUnaRutaVaciaNoCuenta(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	a.AlAbrirCon("")
	if ap := a.AperturaDeArranque(); len(ap.Rutas) != 0 {
		t.Errorf("ha guardado una ruta vacía: %v", ap.Rutas)
	}
}

// Un .esf puede llevar un fichero cifrado o el contenedor de una línea que sale
// de cifrar un texto. Abrir el segundo en la pantalla de ficheros es un lío: lo
// que se quiere ver ahí es el secreto, no otro fichero al lado.
func TestUnEsfDeTextoAbreLaPantallaDeTexto(t *testing.T) {
	dir := t.TempDir()

	linea, err := cripto.SellarTexto([]byte("un secreto"), []byte("clave"),
		cripto.PerfilInteractivo)
	if err != nil {
		t.Fatal(err)
	}

	deTexto := filepath.Join(dir, "secreto.esf")
	if err := os.WriteFile(deTexto, []byte(linea+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ap := AperturaDe([]string{deTexto})
	if ap.Modo != "texto" {
		t.Fatalf("modo: quiero texto, tengo %q", ap.Modo)
	}
	if ap.Texto != linea {
		t.Errorf("texto:\nquiero %q\ntengo  %q", linea, ap.Texto)
	}
	if len(ap.Rutas) != 0 {
		t.Errorf("además lo ha puesto como fichero: %v", ap.Rutas)
	}
}

func TestUnEsfDeFicheroAbreLaPantallaDeFicheros(t *testing.T) {
	dir := t.TempDir()

	origen := filepath.Join(dir, "credenciales.env")
	if err := os.WriteFile(origen, []byte("API_KEY=abc123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cifrado, err := CifrarFichero(origen, []byte("clave"))
	if err != nil {
		t.Fatal(err)
	}

	ap := AperturaDe([]string{cifrado})
	if ap.Modo != "ficheros" {
		t.Fatalf("modo: quiero ficheros, tengo %q", ap.Modo)
	}
	if len(ap.Rutas) != 1 || ap.Rutas[0] != cifrado {
		t.Errorf("rutas: quiero [%s], tengo %v", cifrado, ap.Rutas)
	}
}

// Con varios no se puede adivinar cuál iría en la pantalla de texto, así que van
// todos como ficheros.
func TestVariosSiempreVanComoFicheros(t *testing.T) {
	dir := t.TempDir()

	linea, _ := cripto.SellarTexto([]byte("x"), []byte("clave"), cripto.PerfilInteractivo)
	var rutas []string
	for _, n := range []string{"uno.esf", "dos.esf"} {
		ruta := filepath.Join(dir, n)
		if err := os.WriteFile(ruta, []byte(linea), 0o600); err != nil {
			t.Fatal(err)
		}
		rutas = append(rutas, ruta)
	}

	if ap := AperturaDe(rutas); ap.Modo != "ficheros" {
		t.Errorf("modo: quiero ficheros, tengo %q", ap.Modo)
	}
}

// Un fichero que no existe no puede colarse como texto ni reventar.
func TestUnFicheroQueNoEstaVaComoFichero(t *testing.T) {
	ap := AperturaDe([]string{"/no/existe/nada.esf"})
	if ap.Modo != "ficheros" || len(ap.Rutas) != 1 {
		t.Errorf("quiero que vaya como fichero, tengo %+v", ap)
	}
}
