package app

import "testing"

// Un fichero que llega antes de que la ventana esté escuchando no se puede
// mandar por evento: no hay nadie al otro lado. Se guarda y se entrega cuando
// pregunta. Éste es el caso de abrir Esfinge haciendo doble clic en un .esf.
func TestElFicheroQueLlegaAntesDeLaVentanaSeGuarda(t *testing.T) {
	a, s := nuevaDePrueba(t)

	a.AlAbrirCon("/tmp/secreto.esf")

	if len(s.avisosDe(EventoFicheroAbierto)) != 0 {
		t.Error("ha mandado un evento cuando todavía no había nadie escuchando")
	}

	rutas := a.FicherosDeArranque()
	if len(rutas) != 1 || rutas[0] != "/tmp/secreto.esf" {
		t.Fatalf("quiero [/tmp/secreto.esf], tengo %v", rutas)
	}

	// Una sola vez: si se devolvieran siempre, cambiar de pestaña repondría el
	// fichero una y otra vez.
	if segunda := a.FicherosDeArranque(); len(segunda) != 0 {
		t.Errorf("los ha entregado dos veces: %v", segunda)
	}
}

// Con la ventana ya abierta nadie va a volver a preguntar, así que hay que
// avisar. Éste es el caso de un doble clic mientras Esfinge corre, y es el que
// no hacía nada: el evento se emitía pero la interfaz no lo escuchaba.
func TestConLaVentanaAbiertaElFicheroLlegaPorEvento(t *testing.T) {
	a, s := nuevaDePrueba(t)

	a.FicherosDeArranque() // la ventana dice que ya está

	a.AlAbrirCon("/tmp/otro.esf")

	avisos := s.avisosDe(EventoFicheroAbierto)
	if len(avisos) != 1 || avisos[0] != "/tmp/otro.esf" {
		t.Fatalf("quiero un aviso con /tmp/otro.esf, tengo %v", avisos)
	}
	// Y no se queda además guardado, que lo abriría dos veces.
	if rutas := a.FicherosDeArranque(); len(rutas) != 0 {
		t.Errorf("además lo ha guardado: %v", rutas)
	}
}

// macOS manda un evento por fichero, así que abrir varios de golpe son varias
// llamadas antes de que la ventana exista.
func TestVariosFicherosDeGolpe(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	a.AlAbrirCon("/tmp/uno.esf")
	a.AlAbrirCon("/tmp/dos.esf")

	if rutas := a.FicherosDeArranque(); len(rutas) != 2 {
		t.Errorf("quiero dos ficheros, tengo %v", rutas)
	}
}

func TestUnaRutaVaciaNoCuenta(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	a.AlAbrirCon("")
	if rutas := a.FicherosDeArranque(); len(rutas) != 0 {
		t.Errorf("ha guardado una ruta vacía: %v", rutas)
	}
}
