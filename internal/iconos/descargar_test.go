package iconos

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// unPNG dibuja un cuadrado de un color, del tamaño que se pida.
func unPNG(t *testing.T, lado int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, lado, lado))
	for y := 0; y < lado; y++ {
		for x := 0; x < lado; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 30, B: 40, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// conServidor levanta un sitio de mentira y devuelve un descargador que sí puede
// hablar con él: el filtro de direcciones privadas rechazaría 127.0.0.1, que es
// justo lo que tiene que hacer en producción.
func conServidor(t *testing.T, h http.HandlerFunc) (*Descargador, string) {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)

	d := &Descargador{PermitirPrivadas: true, Cliente: s.Client()}
	return d, strings.TrimPrefix(s.URL, "http://")
}

// deEsteSitio pide el icono por HTTP, que es lo que sirve httptest.
func (d *Descargador) deEsteSitio(ctx context.Context, base string) (string, error) {
	for _, ruta := range dondeMirar {
		datos, err := d.bajar(ctx, "http://"+base+ruta)
		if err != nil {
			continue
		}
		png, err := aPNGPequeño(datos)
		if err != nil {
			continue
		}
		return "data:image/png;base64," + enBase64(png), nil
	}
	return "", ErrNoHay
}

func TestTraeElIconoYLoDejaPequeño(t *testing.T) {
	grande := unPNG(t, 180) // lo que sirve un apple-touch-icon de verdad
	d, base := conServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apple-touch-icon.png" {
			http.NotFound(w, r)
			return
		}
		w.Write(grande)
	})

	uri, err := d.deEsteSitio(t.Context(), base)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Fatalf("no parece una URI de datos: %.40s", uri)
	}

	crudo, err := base64Std.DecodeString(strings.TrimPrefix(uri, "data:image/png;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(crudo))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != ladoGuardado || cfg.Height != ladoGuardado {
		t.Errorf("guardado a %d×%d, y tiene que ser %d", cfg.Width, cfg.Height, ladoGuardado)
	}

	// **Y tiene que pesar poco**, que es la razón de reducirlo: sin esto, sesenta y
	// cinco iconos son megabytes.
	if len(crudo) > 4<<10 {
		t.Errorf("pesa %d bytes; a 32×32 no debería llegar a cuatro kilobytes", len(crudo))
	}
}

// **El caso más común de un sitio sin icono no es un 404: es un 200 con la
// página de error dentro.** Por eso lo que decide es decodificar, no la cabecera.
func TestUnaPaginaDeErrorNoEsUnIcono(t *testing.T) {
	d, base := conServidor(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png") // mintiendo, además
		w.Write([]byte("<!doctype html><html><body>No existe</body></html>"))
	})

	if _, err := d.deEsteSitio(t.Context(), base); err == nil {
		t.Error("ha dado por bueno un trozo de HTML")
	}
}

// Una imagen que dice medir treinta mil píxeles de lado no se abre: son gigas de
// memoria. Se mira la cabecera antes de decodificar justo para esto.
func TestNoSeAbreUnaImagenQueDiceSerEnorme(t *testing.T) {
	// Una cabecera PNG con dimensiones absurdas, sin los datos.
	var b bytes.Buffer
	b.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	b.Write([]byte{0, 0, 0, 13, 'I', 'H', 'D', 'R'})
	b.Write([]byte{0x00, 0x00, 0x75, 0x30}) // 30.000 de ancho
	b.Write([]byte{0x00, 0x00, 0x75, 0x30}) // 30.000 de alto
	b.Write([]byte{8, 6, 0, 0, 0})
	b.Write([]byte{0, 0, 0, 0}) // CRC de mentira

	if _, err := aPNGPequeño(b.Bytes()); err == nil {
		t.Error("ha aceptado una imagen de 30.000 píxeles de lado")
	}
}

// Lo que llega de la red se lee con tope. Un sitio que sirve diez megabytes por
// su favicon no puede llevarse diez megabytes de memoria.
func TestLoQueLlegaSeLeeConTope(t *testing.T) {
	d, base := conServidor(t, func(w http.ResponseWriter, _ *http.Request) {
		relleno := bytes.Repeat([]byte{0}, 1<<20)
		for i := 0; i < 10; i++ {
			w.Write(relleno)
		}
	})

	datos, err := d.bajar(t.Context(), "http://"+base+"/apple-touch-icon.png")
	if err != nil {
		t.Fatal(err)
	}
	if len(datos) > loQueSeLee {
		t.Errorf("se ha leído %d bytes con un tope de %d", len(datos), loQueSeLee)
	}
}

// **La comprobación que de verdad importa.** Un programa que guarda contraseñas
// no puede hacer peticiones a la red de casa de nadie, y el nombre no basta:
// cualquier dominio público puede resolver a una dirección privada.
func TestNoSeTocaLaRedPrivada(t *testing.T) {
	privadas := []string{
		"127.0.0.1", "::1", "10.0.0.5", "192.168.1.1", "172.16.0.1",
		"169.254.169.254", // la de metadatos de las nubes
		"100.64.0.1",      // la franja del operador
		"0.0.0.0",
	}
	for _, s := range privadas {
		if !esPrivada(mustIP(t, s)) {
			t.Errorf("%s tendría que estar prohibida", s)
		}
	}
	for _, s := range []string{"93.184.216.34", "8.8.8.8", "2606:2800:220:1::1"} {
		if esPrivada(mustIP(t, s)) {
			t.Errorf("%s es pública y se ha rechazado", s)
		}
	}

	// Y de punta a punta: sin aflojar el filtro, un servidor en 127.0.0.1 no se
	// deja visitar ni aunque se le pida directamente.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(unPNG(t, 32))
	}))
	defer s.Close()

	d := &Descargador{} // sin PermitirPrivadas
	if _, err := d.bajar(t.Context(), s.URL+"/apple-touch-icon.png"); err == nil {
		t.Error("ha llegado a una dirección privada")
	}
}

// Y por nombre, lo que ni siquiera hay que resolver: preguntar por «.local» ya
// es tocar la red de alguien.
func TestLosNombresQueNoSeVisitan(t *testing.T) {
	for _, malo := range []string{
		"intranet.empresa.local", "servidor.internal", "algo.test",
		"localhost", "maquina", "", "   ",
		"http://192.168.1.1/", "https://127.0.0.1:8080/x",
		"https://señor.es", // sin punycode no se sabe pedirlo
	} {
		if a := Anfitrion(malo); a != "" {
			t.Errorf("%q se ha dado por visitable como %q", malo, a)
		}
	}

	// Y lo que sí, normalizado.
	for entra, sale := range map[string]string{
		"https://www.banco.es/particulares?x=1": "www.banco.es",
		"banco.es":                              "banco.es",
		"  HTTPS://GitHub.com  ":                "github.com",
	} {
		if a := Anfitrion(entra); a != sale {
			t.Errorf("%q dio %q y quería %q", entra, a, sale)
		}
	}
}

// Una redirección no puede servir para esquivar el filtro: si se permitiera
// bajar a http, un «Location: http://192.168.1.1/» pasaría por encima de todo.
func TestUnaRedireccionNoPuedeBajarATextoClaro(t *testing.T) {
	d := Nuevo()
	pet, _ := http.NewRequest(http.MethodGet, "http://192.168.1.1/x", nil)
	if err := d.Cliente.CheckRedirect(pet, nil); err == nil {
		t.Error("ha aceptado una redirección a http")
	}

	// Y hay tope de saltos.
	seguro, _ := http.NewRequest(http.MethodGet, "https://otro.es/x", nil)
	muchos := make([]*http.Request, saltos)
	if err := d.Cliente.CheckRedirect(seguro, muchos); err == nil {
		t.Error("ha seguido más saltos de la cuenta")
	}
}

// El sitio no se entera de qué programa pregunta. Es el único dato
// identificativo que podría salir de la máquina, y no sale.
func TestNoSeDiceQuienPregunta(t *testing.T) {
	visto := make(chan string, 1)
	d, base := conServidor(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case visto <- r.Header.Get("User-Agent"):
		default:
		}
		http.NotFound(w, r)
	})
	d.deEsteSitio(t.Context(), base)

	ua := <-visto
	if strings.Contains(strings.ToLower(ua), "esfinge") {
		t.Errorf("el sitio se entera de que es Esfinge: %q", ua)
	}
}

// Un sitio que no tiene ninguna de las rutas conocidas devuelve «no hay», que no
// es un fallo: es la respuesta correcta y hay que poder recordarla.
func TestUnSitioSinIconoLoDiceComoTal(t *testing.T) {
	d, base := conServidor(t, http.NotFound)
	if _, err := d.deEsteSitio(t.Context(), base); !errors.Is(err, ErrNoHay) {
		t.Errorf("quiero ErrNoHay, tengo %v", err)
	}
}

func mustIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("%q no es una dirección", s)
	}
	return ip
}

// **La prueba que faltaba, y que habría ahorrado una versión entera.**
//
// El filtro de direcciones privadas estaba en `DialContext`, que recibe el
// **nombre sin resolver**: `ParseIP("github.com")` da nulo, la regla «si no se
// sabe qué es, no se va» se cumplía, y se rechazaban todos los sitios del mundo.
// La 2.14.0 salió así y ninguna entrada llegó a tener icono nunca.
//
// Lo que se comprueba aquí es que el rechazo llega **después de resolver**: se
// pide por un nombre —«localhost», que resuelve sin DNS— y el motivo del rechazo
// tiene que hablar de la dirección, no del nombre. Si el filtro volviera a
// mirarlo antes de resolver, esto se pondría rojo.
func TestElFiltroMiraLaDireccionResueltaYNoElNombre(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(unPNG(t, 32))
	}))
	defer s.Close()
	_, puerto, _ := net.SplitHostPort(strings.TrimPrefix(s.URL, "http://"))

	d := &Descargador{} // con el filtro puesto
	_, err := d.bajar(t.Context(), "http://localhost:"+puerto+"/favicon.png")
	if err == nil {
		t.Fatal("ha entrado en una dirección privada por su nombre")
	}
	if !strings.Contains(err.Error(), "127.0.0.1") && !strings.Contains(err.Error(), "::1") {
		t.Errorf("el rechazo no habla de la dirección resuelta: %v", err)
	}
	if strings.Contains(err.Error(), "\"localhost\" no es una dirección resuelta") {
		t.Error("el filtro está mirando el nombre en vez de la dirección")
	}
}

// Un nombre que no resuelve tampoco puede colarse.
func TestUnNombreQueNoResuelveNoConecta(t *testing.T) {
	d := &Descargador{}
	if _, err := d.bajar(t.Context(), "https://esto.no.existe.invalid/favicon.png"); err == nil {
		t.Error("ha conectado con un nombre que no existe")
	}
}

// --------------------------------------------------------------------- el ICO

// unICO monta un icono con lo que se le dé dentro.
func unICO(ancho, alto byte, dentro []byte) []byte {
	var b []byte
	b = append(b, 0, 0, 1, 0, 1, 0) // reservado, tipo 1, una entrada
	entrada := make([]byte, 16)
	entrada[0], entrada[1] = ancho, alto
	binary.LittleEndian.PutUint32(entrada[8:12], uint32(len(dentro)))
	binary.LittleEndian.PutUint32(entrada[12:16], 22) // 6 de cabecera + 16 de índice
	b = append(b, entrada...)
	return append(b, dentro...)
}

// unBMPde32 monta el mapa de bits que llevan dentro los ICO de los sitios
// grandes: cabecera de cuarenta bytes, cuatro bytes por píxel y el alto doblado.
func unBMPde32(lado int) []byte {
	b := make([]byte, 40)
	binary.LittleEndian.PutUint32(b[0:4], 40)
	binary.LittleEndian.PutUint32(b[4:8], uint32(lado))
	binary.LittleEndian.PutUint32(b[8:12], uint32(lado*2)) // doblado, con su máscara
	binary.LittleEndian.PutUint16(b[12:14], 1)
	binary.LittleEndian.PutUint16(b[14:16], 32)
	for i := 0; i < lado*lado; i++ {
		b = append(b, 10, 20, 30, 255) // azul, verde, rojo, alfa
	}
	return b
}

// Los dos contenidos posibles de un ICO. El primero es lo normal hoy; el segundo
// es lo que sirven todavía Google, Amazon y Netflix, y por eso está.
func TestUnICOSeLeeConPNGYConMapaDeBits(t *testing.T) {
	t.Run("con un PNG dentro", func(t *testing.T) {
		png, img, err := deUnICO(unICO(32, 32, unPNG(t, 32)))
		if err != nil || img != nil || png == nil {
			t.Fatalf("png=%d img=%v err=%v", len(png), img != nil, err)
		}
	})

	t.Run("con un mapa de bits de 32 dentro", func(t *testing.T) {
		_, img, err := deUnICO(unICO(32, 32, unBMPde32(32)))
		if err != nil {
			t.Fatal(err)
		}
		if img == nil || img.Bounds().Dx() != 32 {
			t.Fatalf("img=%v", img)
		}
		// Los componentes van en orden azul, verde, rojo: si se leen al revés, el
		// icono sale con los colores cambiados y nadie lo nota hasta verlo.
		r, g, b, _ := img.At(0, 0).RGBA()
		if r>>8 != 30 || g>>8 != 20 || b>>8 != 10 {
			t.Errorf("colores del revés: %d %d %d", r>>8, g>>8, b>>8)
		}
	})

	t.Run("con un mapa de bits de otra profundidad se deja", func(t *testing.T) {
		malo := unBMPde32(8)
		binary.LittleEndian.PutUint16(malo[14:16], 8) // 8 bits, con paleta
		if _, _, err := deUnICO(unICO(8, 8, malo)); err == nil {
			t.Error("ha intentado leer un mapa de bits con paleta")
		}
	})
}

// **Un ICO es un fichero de un tercero**, así que sus números no se creen: un
// desplazamiento inventado no puede tumbar el programa.
func TestUnICOInventadoNoRompeNada(t *testing.T) {
	malos := [][]byte{
		{0, 0, 1, 0, 1, 0},                    // dice tener una entrada y no la trae
		{0, 0, 1, 0, 200, 0},                  // dice tener doscientas
		append(unICO(32, 32, unPNG(t, 4)), 0), // con un byte de más
		{0, 0, 2, 0, 1, 0, 0, 0, 0, 0, 0, 0},  // tipo 2, que es un cursor
	}
	// Y uno con el desplazamiento apuntando fuera del fichero.
	fuera := unICO(32, 32, unPNG(t, 4))
	binary.LittleEndian.PutUint32(fuera[6+12:6+16], 999999)
	malos = append(malos, fuera)

	for i, m := range malos {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("el caso %d ha reventado: %v", i, r)
				}
			}()
			deUnICO(m)     // no puede entrar en pánico
			aPNGPequeño(m) // ni por este camino
		}()
	}
}
