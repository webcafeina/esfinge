package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// apiFalsa hace de GitHub. Cuenta las peticiones, que es la mitad de lo que hay
// que vigilar aquí.
func apiFalsa(t *testing.T, etiqueta string, peticiones *atomic.Int32) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if peticiones != nil {
			peticiones.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": etiqueta,
			"html_url": "https://github.com/webcafeina/esfinge/releases/tag/" + etiqueta,
			"assets": []map[string]any{
				{"name": "Esfinge-2.1.0.dmg", "browser_download_url": "https://x/dmg", "size": 8},
				{"name": "Esfinge-2.1.0-windows-instalador.exe", "browser_download_url": "https://x/exe"},
				{"name": "esfinge_2.1.0_amd64.deb", "browser_download_url": "https://x/deb"},
				{"name": "esfinge-2.1.0-linux-amd64.tar.gz", "browser_download_url": "https://x/tgz"},
			},
		})
	}))
	t.Cleanup(s.Close)
	return s
}

// paraActualizar monta la aplicación con una versión que sí se puede comparar y
// apuntando a una API de mentira.
func paraActualizar(t *testing.T, etiqueta string, peticiones *atomic.Int32) (*App, *sistemaFalso) {
	t.Helper()
	casa := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", casa)
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)

	s := &sistemaFalso{}
	a := Nueva("2.0.3", s)
	ApuntarAAPI(a, apiFalsa(t, etiqueta, peticiones).URL)
	return a, s
}

// esperarNovedad da tiempo a la gorrutina del arranque sin dormir un plazo fijo.
func esperarNovedad(t *testing.T, s *sistemaFalso) []Novedad {
	t.Helper()
	for i := 0; i < 200; i++ {
		if n := s.verNovedades(); len(n) > 0 {
			return n
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

func TestAlArrancarAvisaDeLaVersionNueva(t *testing.T) {
	a, s := paraActualizar(t, "v2.1.0", nil)
	a.Arrancar(context.Background())

	novedades := esperarNovedad(t, s)
	if len(novedades) == 0 {
		t.Fatal("hay una 2.1.0 publicada y no ha avisado")
	}
	if novedades[0].Version != "2.1.0" {
		t.Errorf("versión: quiero 2.1.0, tengo %q", novedades[0].Version)
	}
	if !a.NovedadPendiente().Hay {
		t.Error("la ventana pregunta al montarse y le dice que no hay nada")
	}
}

func TestSinVersionNuevaNoDiceNada(t *testing.T) {
	a, s := paraActualizar(t, "v2.0.3", nil)
	a.Arrancar(context.Background())

	// Se le da el mismo margen que en el caso bueno; aquí lo que se espera es que
	// pase el rato sin que llegue nada.
	if novedades := esperarNovedad(t, s); len(novedades) != 0 {
		t.Errorf("no hay versión nueva y ha avisado igual: %+v", novedades)
	}
}

// El interruptor de Ajustes tiene que cortar la salida a la red de verdad, no
// solo esconder el aviso.
func TestApagadaNiSiquieraPregunta(t *testing.T) {
	var peticiones atomic.Int32
	a, s := paraActualizar(t, "v2.1.0", &peticiones)

	if err := a.GuardarPreferencias(Preferencias{BuscarActualizaciones: false}); err != nil {
		t.Fatalf("guardar preferencias: %v", err)
	}
	a.Arrancar(context.Background())
	esperarNovedad(t, s)

	if n := peticiones.Load(); n != 0 {
		t.Errorf("está apagada y ha hecho %d peticiones", n)
	}
}

// «Buscar ahora» sale a preguntar aunque la fecha diga que no toca: para eso es
// un botón.
func TestBuscarAhoraPreguntaAunqueNoToque(t *testing.T) {
	var peticiones atomic.Int32
	a, _ := paraActualizar(t, "v2.1.0", &peticiones)

	if _, err := a.ComprobarActualizacion(); err != nil {
		t.Fatalf("comprobar: %v", err)
	}
	n, err := a.ComprobarActualizacion()
	if err != nil {
		t.Fatalf("comprobar otra vez: %v", err)
	}
	if !n.Hay {
		t.Error("hay una 2.1.0 y dice que no")
	}
	if peticiones.Load() != 2 {
		t.Errorf("dos pulsaciones, dos peticiones: tengo %d", peticiones.Load())
	}
}

// Descargar antes de haber encontrado nada no puede acabar bajándose algo a
// ciegas.
func TestSinNovedadNoHayNadaQueDescargarNiInstalar(t *testing.T) {
	a, _ := paraActualizar(t, "v2.0.3", nil)

	if _, err := a.DescargarActualizacion(); err == nil {
		t.Error("no hay novedad y ha intentado descargar")
	}
	if err := a.InstalarActualizacion(); err == nil {
		t.Error("no hay nada descargado y ha intentado instalar")
	}
}

// Las órdenes del menú del sistema llegan a la ventana por el mismo camino que
// el progreso: si esto se rompe, el menú queda de adorno.
func TestLasOrdenesDelMenuLleganALaVentana(t *testing.T) {
	a, s := paraActualizar(t, "v2.0.3", nil)

	a.Ordenar(OrdenIrAAjustes)
	a.OrdenarPegar("una contraseña")

	ordenes := s.verOrdenes()
	if len(ordenes) != 2 {
		t.Fatalf("quiero 2 órdenes, tengo %d", len(ordenes))
	}
	if ordenes[0].Que != OrdenIrAAjustes {
		t.Errorf("la primera: quiero %q, tengo %q", OrdenIrAAjustes, ordenes[0].Que)
	}
	if ordenes[1].Que != OrdenPegar || ordenes[1].Texto != "una contraseña" {
		t.Errorf("la segunda: quiero pegar «una contraseña», tengo %+v", ordenes[1])
	}
}

// El reloj es la diferencia entre «comprueba una vez al día», que es lo que
// promete la portada, y «comprueba al abrir la ventana», que es lo que hacía
// hasta la 2.10.3. Quien deja Esfinge abierta —que es lo normal— no se enteraba
// nunca de una versión nueva. Lo dijo el cliente: no le salía la banda.
func TestElRelojAvisaSinReiniciarLaVentana(t *testing.T) {
	a, s := paraActualizar(t, "v2.1.0", nil)

	// Nada de Arrancar: aquí lo que se prueba es que el aviso llega **sin** que
	// nadie vuelva a abrir la ventana. Si esto pasara solo por el arranque, la
	// prueba no diría nada.
	ctx, parar := context.WithCancel(context.Background())
	defer parar()
	a.vigilar(ctx, 5*time.Millisecond)

	novedades := esperarNovedad(t, s)
	if len(novedades) == 0 {
		t.Fatal("el reloj no ha avisado de la 2.1.0 con la ventana ya abierta")
	}
	if novedades[0].Version != "2.1.0" {
		t.Errorf("versión: quiero 2.1.0, tengo %q", novedades[0].Version)
	}
}

// Asomarse a menudo no es preguntar a menudo: quien decide sigue siendo la
// puerta de las 24 horas. Sin esto, un reloj de una hora serían veinticuatro
// peticiones al día en vez de una.
func TestElRelojNoDisparaMasPeticionesQueLaPuertaDeLasVeinticuatroHoras(t *testing.T) {
	var peticiones atomic.Int32
	a, s := paraActualizar(t, "v2.1.0", &peticiones)

	ctx, parar := context.WithCancel(context.Background())
	defer parar()
	a.vigilar(ctx, time.Millisecond)

	if len(esperarNovedad(t, s)) == 0 {
		t.Fatal("el reloj no ha llegado a comprobar ni una vez")
	}
	// Con un reloj de 1 ms, aquí caben cientos de vueltas. Si la puerta no
	// funcionara, se notaría de sobra.
	time.Sleep(150 * time.Millisecond)

	if n := peticiones.Load(); n != 1 {
		t.Errorf("quiero exactamente 1 petición, tengo %d: la puerta de las 24 h no está frenando al reloj", n)
	}
}

// Y el reloj se va con la ventana: si no, cada ventana abierta dejaría una
// gorrutina viva pidiendo a GitHub para siempre.
func TestElRelojSeParaConLaVentana(t *testing.T) {
	var peticiones atomic.Int32
	a, s := paraActualizar(t, "v2.1.0", &peticiones)

	ctx, parar := context.WithCancel(context.Background())
	a.vigilar(ctx, time.Millisecond)
	if len(esperarNovedad(t, s)) == 0 {
		t.Fatal("el reloj no ha llegado a comprobar")
	}

	parar()
	// Un respiro para que termine lo que ya estuviera en la red cuando se cerró:
	// lo que se prueba es que no empiece nada nuevo, no que se corte a media
	// petición.
	time.Sleep(50 * time.Millisecond)

	// Se le borra la fecha, así que la puerta dejaría pasar: lo único que puede
	// frenarlo ya es que el reloj esté parado de verdad.
	a.ajustes.mu.Lock()
	a.ajustes.p.UltimaComprobacion = ""
	a.ajustes.mu.Unlock()

	antes := peticiones.Load()
	time.Sleep(100 * time.Millisecond)
	if despues := peticiones.Load(); despues != antes {
		t.Errorf("la ventana se cerró y el reloj siguió pidiendo: %d → %d", antes, despues)
	}
}
