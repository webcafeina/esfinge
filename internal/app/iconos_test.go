package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/iconos"
)

// **Con la red apagada no se abre ni un socket**, y da igual lo que digan los
// ajustes: ESFINGE_SIN_RED es un freno, no una preferencia más.
//
// La prueba mira lo único que se puede mirar sin red: que el goteo ni siquiera
// arranca. Si arrancara, lo siguiente sería una petición.
func TestConLaRedApagadaNoSeBuscanIconos(t *testing.T) {
	t.Setenv("ESFINGE_SIN_RED", "1")
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Titulo: "Banco", Sitios: []string{"https://banco.es"},
	}); err != nil {
		t.Fatal(err)
	}

	// Con la red apagada no hay nada que hacer, así que no se toca la caché.
	a.buscarIconosSiProcede(t.Context())
	if len(a.bov.Iconos()) != 0 {
		t.Error("ha guardado algo con la red apagada")
	}
}

// Y con el ajuste apagado, tampoco.
func TestConElAjusteApagadoNoSeBuscanIconos(t *testing.T) {
	a, _, _ := conReloj(t)
	if err := a.GuardarPreferencias(Preferencias{DescargarIconos: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	a.buscarIconosSiProcede(t.Context())
	if len(a.bov.Iconos()) != 0 {
		t.Error("ha guardado algo con el ajuste apagado")
	}
}

// De la bóveda salen los anfitriones a los que preguntar, y **solo los que se
// pueden visitar**: una intranet o una dirección de la red de casa no se tocan.
func TestLoQueFaltaPorMirarDejaFueraLoQueNoSeVisita(t *testing.T) {
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	for _, sitio := range []string{
		"https://banco.es/login",
		"https://www.banco.es/otra",      // el mismo anfitrión no, el mismo no: www. cuenta aparte
		"https://intranet.empresa.local", // no se toca
		"http://192.168.1.1",             // tampoco
		"",                               // ni esto
	} {
		if err := a.GuardarEnBoveda(boveda.Entrada{
			Titulo: "x " + sitio, Sitios: []string{sitio},
		}); err != nil {
			t.Fatal(err)
		}
	}

	faltan := a.loQueFaltaPorMirar()
	for _, a := range faltan {
		if a == "intranet.empresa.local" || a == "192.168.1.1" {
			t.Errorf("va a preguntarle a %q", a)
		}
	}
	if len(faltan) != 2 {
		t.Errorf("esperaba banco.es y www.banco.es, y tengo %v", faltan)
	}
}

// **El camino entero, de la bóveda al almacén, y nunca se había probado.**
//
// Aquí estaban los dos fallos que vio el cliente y no vieron las pruebas: que el
// filtro rechazaba todos los sitios del mundo, y que lo bajado no se guardaba
// hasta terminar la tanda —minutos— así que cerrar antes lo tiraba todo. Las
// piezas estaban probadas una a una; **la tubería, no**.
func TestElCaminoEnteroDeUnIcono(t *testing.T) {
	// Un sitio de mentira que sirve un icono en la primera ruta conocida.
	pedidas := make(chan string, 10)
	// **Con TLS**, porque el descargador solo habla https y no se rebaja: probarlo
	// contra un servidor en claro sería probar otra cosa.
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case pedidas <- r.URL.Path:
		default:
		}
		if r.URL.Path != "/apple-touch-icon.png" {
			http.NotFound(w, r)
			return
		}
		w.Write(unPNGdePrueba(t))
	}))
	defer s.Close()
	anfitrion := strings.TrimPrefix(s.URL, "https://")

	a, sis, _ := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Titulo: "El sitio", Sitios: []string{s.URL},
	}); err != nil {
		t.Fatal(err)
	}

	// El descargador de prueba habla por HTTP con 127.0.0.1, que en producción está
	// prohibido y aquí hay que dejar pasar a mano.
	d := &iconos.Descargador{PermitirPrivadas: true, Cliente: s.Client()}
	ApuntarIconosA(a, d, ritmo{entre: time.Millisecond, alPrincipio: time.Millisecond, cuantos: 5})

	a.gotearIconosDesde(t.Context(), []string{anfitrion})

	// **Se guarda uno a uno**, así que en cuanto termina el primero ya está dentro.
	guardados := a.bov.Iconos()
	i, hay := guardados[anfitrion]
	if !hay {
		t.Fatalf("no se ha guardado nada; se pidió %v", vaciar(pedidas))
	}
	if !strings.HasPrefix(i.URI, "data:image/png;base64,") {
		t.Errorf("lo guardado no es un icono: %.40s", i.URI)
	}
	if i.Mirado == "" {
		t.Error("no se ha apuntado cuándo se miró")
	}

	// Y llega a la ventana, que es lo que hace que se vea sin recargar.
	if !sis.hanAvisadoDe(EventoIconos) {
		t.Error("no se ha avisado a la ventana")
	}

	// Y el puente lo devuelve listo para pintar.
	porPuente, err := a.IconosDeBoveda()
	if err != nil {
		t.Fatal(err)
	}
	if porPuente[anfitrion] != i.URI {
		t.Errorf("por el puente sale otra cosa: %.40s", porPuente[anfitrion])
	}
}

// Un sitio que no da icono se apunta igual, para no volver a preguntarle mañana.
func TestUnSitioSinIconoSeApuntaIgual(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(http.NotFound))
	defer s.Close()
	anfitrion := strings.TrimPrefix(s.URL, "https://")

	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	ApuntarIconosA(a,
		&iconos.Descargador{PermitirPrivadas: true, Cliente: s.Client()},
		ritmo{entre: time.Millisecond, alPrincipio: time.Millisecond, cuantos: 5})

	a.gotearIconosDesde(t.Context(), []string{anfitrion})

	i, hay := a.bov.Iconos()[anfitrion]
	if !hay || i.URI != "" || i.Mirado == "" {
		t.Errorf("un sitio sin icono tiene que quedar apuntado como mirado: %+v", i)
	}
}

func unPNGdePrueba(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 30, B: 40, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func vaciar(c chan string) []string {
	var out []string
	for {
		select {
		case v := <-c:
			out = append(out, v)
		default:
			return out
		}
	}
}
