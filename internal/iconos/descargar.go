// Package iconos trae el icono de un sitio web para enseñarlo en la bóveda.
//
// **Es la segunda cosa que Esfinge hace fuera de esta máquina**, y la primera
// que va a un sitio que no elegimos nosotros. Eso cambia el problema entero:
// la comprobación de versiones habla siempre con `api.github.com`, un destino
// fijo y conocido; aquí el destino sale de un CSV que alguien importó, así que
// **lo elige el mundo exterior**. Todo lo que hay en este fichero existe por esa
// diferencia:
//
//   - Se rechazan las direcciones privadas **en el momento de conectar**, no
//     mirando el nombre: cualquier dominio público puede resolver a 10.0.0.5.
//   - Se limitan los saltos de redirección y se prohíbe bajar a `http://`, o el
//     filtro anterior se esquiva con un `Location:`.
//   - Lo que llega se lee con tope, se mira **antes** de decodificar y se vuelve
//     a codificar aquí. Lo que se guarda no es el fichero del sitio: son píxeles
//     re-emitidos por nuestro codificador.
//
// Lo que este paquete **no** promete, y conviene decirlo donde se lee el código
// y no solo en la ADR: pedir el icono a cada sitio no oculta qué sitios hay en
// la bóveda. El nombre viaja en claro en la consulta de DNS y en el saludo TLS,
// así que quien mire la red los ve. Lo que se gana yendo directo es **no meter
// un tercero de confianza** en un programa que guarda contraseñas, que es una
// razón distinta y suficiente.
package iconos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // para que DecodeConfig los reconozca
	_ "image/jpeg" //
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// loQueSeLee es el tope de lo que se baja de un sitio. Un icono que no cabe
	// en esto no es un icono.
	loQueSeLee = 128 << 10

	// ladoMaximo es el tope de la imagen **declarada en la cabecera**, que se mira
	// antes de decodificar. Un PNG de treinta kilobytes puede declarar
	// 30.000×30.000 y reventar la memoria al abrirlo: son 3,6 GB de píxeles. Se
	// mira la cabecera primero justamente por eso.
	ladoMaximo = 1024

	// ladoGuardado es a lo que se reduce todo antes de guardarlo. Con esto un
	// icono pesa uno o dos kilobytes en vez de los treinta que sirven algunos
	// sitios, y eso es lo que decide si la bóveda engorda o no.
	ladoGuardado = 32

	// saltos es cuántas redirecciones se siguen. Tres son de sobra para el
	// «www» y el «https»; más es que alguien está jugando.
	saltos = 3
)

// ErrNoHay dice que ese sitio no tiene icono que valga. No es un fallo: la
// mayoría de las veces es la respuesta correcta y hay que recordarla para no
// volver a preguntar.
var ErrNoHay = errors.New("Ese sitio no da ningún icono que se pueda usar")

// dondeMirar son las rutas conocidas, en el orden en que compensa probarlas.
//
// **Sin analizar el HTML de la página**, que sería lo completo: eso obligaría a
// bajarse la portada de cada sitio de la bóveda —mucho más tráfico y mucha más
// superficie— y a meter un analizador de HTML en el binario que guarda las
// contraseñas. Con estas tres se cubre la mayoría, y el cuadro de color tapa el
// hueco cuando no hay ninguna.
//
// **`/favicon.ico` no está**, y no es un olvido: la biblioteca estándar de Go no
// sabe decodificar ICO, así que aceptarlo obligaría a analizar a mano un formato
// de los noventa sobre datos de un tercero. No compensa.
var dondeMirar = []string{
	"/apple-touch-icon.png",
	"/apple-touch-icon-precomposed.png",
	"/favicon.png",
}

// Descargador trae iconos. Se construye una vez y se reutiliza.
type Descargador struct {
	// Cliente se puede sustituir en las pruebas. Si va vacío se construye el de
	// abajo, que es el único que debería usarse de verdad.
	Cliente *http.Client

	// PermitirPrivadas afloja el filtro de direcciones. **Solo para las pruebas**,
	// que levantan sus servidores en 127.0.0.1 y sin esto se rechazarían a sí
	// mismas. No se expone por el puente ni se lee de ninguna variable de entorno:
	// tiene que ser código de prueba quien lo ponga, a mano.
	PermitirPrivadas bool
}

// Nuevo construye un descargador con el cliente que toca.
func Nuevo() *Descargador {
	d := &Descargador{}
	d.Cliente = d.cliente()
	return d
}

func (d *Descargador) cliente() *http.Client {
	if d.Cliente != nil {
		return d.Cliente
	}

	dial := &net.Dialer{Timeout: 5 * time.Second}
	return &http.Client{
		// Corto a propósito: esto es un adorno, no una función. Si un sitio tarda
		// diez segundos en dar su favicon, no lo tiene.
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, red, direccion string) (net.Conn, error) {
				// **El filtro va aquí y no antes**, y es la diferencia entre filtrar y
				// aparentar que se filtra: comprobar el nombre no sirve de nada porque
				// cualquier dominio público puede resolver a una dirección de la red de
				// casa. Lo que hay que mirar es la dirección con la que se va a hablar.
				anfitrion, _, err := net.SplitHostPort(direccion)
				if err != nil {
					return nil, err
				}
				if !d.PermitirPrivadas && esPrivada(net.ParseIP(anfitrion)) {
					return nil, fmt.Errorf("%s es una dirección de una red privada", anfitrion)
				}
				return dial.DialContext(ctx, red, direccion)
			},
			TLSHandshakeTimeout: 5 * time.Second,
			DisableKeepAlives:   true,
		},
		CheckRedirect: func(pet *http.Request, anteriores []*http.Request) error {
			if len(anteriores) >= saltos {
				return errors.New("demasiados saltos")
			}
			// **Y nunca hacia atrás.** Un `https://sitio.com/favicon.png` que redirige
			// a `http://192.168.1.1/` se saltaría el filtro entero si se permitiera
			// bajar a texto claro: ahí ya no hay ni TLS que mirar.
			if pet.URL.Scheme != "https" {
				return fmt.Errorf("redirección a %s, que no es https", pet.URL.Scheme)
			}
			return nil
		},
	}
}

// esPrivada dice si una dirección es de una red que no hay que tocar.
//
// La lista no es de manual: es lo que un programa que guarda contraseñas no
// tiene ningún motivo para visitar. La de metadatos de las nubes (169.254.169.254)
// entra por «enlace local», que es donde vive.
func esPrivada(ip net.IP) bool {
	if ip == nil {
		return true // si no se sabe qué es, no se va
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() ||
		// La franja del operador (RFC 6598), que no cubre IsPrivate.
		enLaFranja(ip, "100.64.0.0/10")
}

func enLaFranja(ip net.IP, cidr string) bool {
	_, franja, err := net.ParseCIDR(cidr)
	return err == nil && franja.Contains(ip)
}

// tldsReservados son los sufijos que nunca salen a internet. Se rechazan por
// nombre y sin resolver nada: preguntar por ellos ya es tocar la red de alguien.
var tldsReservados = []string{
	".local", ".internal", ".home.arpa", ".test", ".invalid",
	".localhost", ".example", ".lan", ".intranet", ".corp", ".home",
}

// Anfitrion normaliza lo que haya escrito en el campo del sitio y dice si se le
// puede preguntar.
//
// Devuelve cadena vacía cuando no hay a quién preguntar, que incluye tanto lo
// que no es un anfitrión como lo que **no se debe** visitar.
func Anfitrion(sitio string) string {
	limpio := strings.ToLower(strings.TrimSpace(sitio))
	if limpio == "" {
		return ""
	}
	if !strings.Contains(limpio, "://") {
		limpio = "https://" + limpio
	}
	u, err := url.Parse(limpio)
	if err != nil {
		return ""
	}
	anfitrion := u.Hostname()

	// Sin puntos no es un nombre de internet: es una máquina de la red de al lado.
	if anfitrion == "" || !strings.Contains(anfitrion, ".") {
		return ""
	}
	for _, malo := range tldsReservados {
		if strings.HasSuffix(anfitrion, malo) {
			return ""
		}
	}
	// Una dirección escrita a pelo se juzga como dirección, no como nombre.
	if ip := net.ParseIP(anfitrion); ip != nil && esPrivada(ip) {
		return ""
	}
	// **Sin caracteres no ASCII.** Un dominio internacionalizado habría que
	// pasarlo por punycode, y eso es una dependencia más; sin él, la petición
	// saldría mal formada. Se quedan sin icono, que es inofensivo.
	for _, r := range anfitrion {
		if r > 127 {
			return ""
		}
	}
	return anfitrion
}

// De trae el icono de un anfitrión, ya reducido y re-codificado, como URI de
// datos listo para pintarse.
func (d *Descargador) De(ctx context.Context, anfitrion string) (string, error) {
	if Anfitrion(anfitrion) == "" {
		return "", ErrNoHay
	}

	var ultimo error
	for _, ruta := range dondeMirar {
		datos, err := d.bajar(ctx, "https://"+anfitrion+ruta)
		if err != nil {
			ultimo = err
			continue
		}
		png, err := aPNGPequeño(datos)
		if err != nil {
			ultimo = err
			continue
		}
		return "data:image/png;base64," + enBase64(png), nil
	}
	if ultimo == nil {
		ultimo = ErrNoHay
	}
	return "", ultimo
}

func (d *Descargador) bajar(ctx context.Context, donde string) ([]byte, error) {
	pet, err := http.NewRequestWithContext(ctx, http.MethodGet, donde, nil)
	if err != nil {
		return nil, err
	}
	// **Sin decir quién somos.** A GitHub se le manda «Esfinge/versión» porque es
	// lo que se compara y porque rechaza a quien no se identifica. A un sitio
	// cualquiera eso sería contarle que quien pregunta usa un gestor de
	// contraseñas concreto, en su versión exacta —con los fallos que esa versión
	// tenga— y que tiene cuenta allí. Es el único dato identificativo que saldría
	// de la máquina, y no sale.
	pet.Header.Set("User-Agent", "Mozilla/5.0")
	pet.Header.Set("Accept", "image/png,image/*")

	resp, err := d.cliente().Do(pet)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s contestó %d", donde, resp.StatusCode)
	}
	// Con tope, como todo lo que llega de la red (ver internal/actualizacion).
	return io.ReadAll(io.LimitReader(resp.Body, loQueSeLee))
}

// aPNGPequeño comprueba que lo que llegó es de verdad una imagen y devuelve un
// PNG de 32×32.
//
// **Decodificar es la comprobación**, no la cabecera `Content-Type`: el caso más
// común de un sitio sin icono no es un 404, es un 200 con la página de error
// dentro. Y re-codificar tiene un segundo efecto que vale por sí solo: lo que se
// guarda son píxeles nuestros, sin los metadatos ni los trozos accesorios que
// trajera el original.
func aPNGPequeño(datos []byte) ([]byte, error) {
	// Primero la cabecera, que dice el tamaño sin reservar un solo píxel.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(datos))
	if err != nil {
		return nil, ErrNoHay
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > ladoMaximo || cfg.Height > ladoMaximo {
		return nil, fmt.Errorf("%w: dice medir %d×%d", ErrNoHay, cfg.Width, cfg.Height)
	}

	img, _, err := image.Decode(bytes.NewReader(datos))
	if err != nil {
		return nil, ErrNoHay
	}

	var b bytes.Buffer
	if err := png.Encode(&b, reducir(img, ladoGuardado)); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func enBase64(b []byte) string {
	return base64Std.EncodeToString(b)
}
