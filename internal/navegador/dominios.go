// Package navegador es el canal por el que una extensión del navegador consulta
// la bóveda.
//
// # Por qué existe y qué lo gobierna
//
// Es la fase 2 de sustituir a Dashlane: el autorrelleno. Y es **la primera vez
// que Esfinge escucha en algo**, así que hereda una frase que este proyecto
// tenía escrita en contra —`internal/app/dev.go`: «un servidor HTTP en el
// binario del cliente, por local que sea, es una puerta que nadie ha pedido»—.
// Ésta se ha pedido. Lo que la hace aceptable no es que se pidiera, sino cómo
// está hecha:
//
//   - **No es TCP.** Un socket de dominio unix, en la carpeta del usuario. No hay
//     puerto al que nadie pueda conectarse desde fuera ni desde otra sesión.
//   - **Viene apagada** y se enciende en Ajustes, al lado de las otras dos salidas.
//   - **El navegador no abre la bóveda.** Lo que hay al otro lado del socket es la
//     ventana, que es la única que tiene la bóveda descifrada. Si el proceso que
//     lanza el navegador la abriera por su cuenta, la contraseña maestra acabaría
//     dentro del navegador.
//
// # El emparejamiento de dominios es la pieza que hay que hacer bien
//
// Todo lo demás de este paquete es fontanería. Esto no: **decidir mal qué entrada
// corresponde a qué pestaña es entregar una contraseña al sitio equivocado**, que
// es la única forma en que un gestor de contraseñas hace daño de verdad.
package navegador

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// ErrSinDominio dice que de ahí no se puede sacar un dominio con el que decidir
// nada. **Siempre significa «no enseñes nada»**, nunca «enseña todo».
var ErrSinDominio = errors.New("De esa dirección no se puede sacar un dominio")

// DominioDeOrigen saca el dominio registrable de la dirección de una pestaña.
//
// El origen **lo dice el navegador**, no la página: llega de `sender.tab.url` y
// no de nada que el documento pueda escribir. Aquí se comprueba lo que se puede
// comprobar de todas formas, porque un canal local no puede fiarse de que quien
// habla sea quien dice.
//
// Dos negativas que parecen severas y no lo son:
//
//   - **Solo `https`.** Rellenar sobre texto claro es regalarle la contraseña a
//     cualquiera que mire la red, y además convierte un ataque de red en un robo
//     de credenciales sin más trabajo. Deja fuera la página de administración de
//     un router, y eso está dicho como carencia conocida en vez de resuelto a
//     medias.
//   - **Nada de direcciones IP.** Una IP no tiene dominio registrable, así que no
//     hay forma de decidir si «es el mismo sitio» sin inventarse una regla. Lo
//     honesto es no rellenar.
func DominioDeOrigen(origen string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(origen))
	if err != nil {
		return "", ErrSinDominio
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return "", fmt.Errorf("Esfinge solo rellena en https, y eso es «%s»", u.Scheme)
	}
	return dominioRegistrable(u.Hostname())
}

// DominioDeSitio saca el dominio registrable de lo que hay guardado en una
// entrada.
//
// **Es tolerante con la forma y nunca con el resultado.** Lo guardado viene de la
// exportación de otro gestor y llega como le dé la gana: `https://banco.es/entrar`,
// `www.banco.es`, `banco.es ` con un espacio detrás. Todo eso es el mismo sitio.
// Lo que no se hace nunca es adivinar: si de ahí no sale un dominio, devuelve
// vacío y esa entrada **no encaja con nada**.
func DominioDeSitio(sitio string) string {
	sitio = strings.TrimSpace(sitio)
	if sitio == "" {
		return ""
	}
	// Sin esquema no hay anfitrión para url.Parse: `banco.es/entrar` se lee
	// entero como ruta. Se le pone uno, que aquí no significa nada —el esquema del
	// sitio guardado no decide si se rellena; eso lo decide el de la pestaña—.
	if !strings.Contains(sitio, "://") {
		sitio = "https://" + sitio
	}
	u, err := url.Parse(sitio)
	if err != nil {
		return ""
	}
	dominio, err := dominioRegistrable(u.Hostname())
	if err != nil {
		return ""
	}
	return dominio
}

// dominioRegistrable reduce un anfitrión a lo que se puede registrar.
//
// **Con la lista de sufijos públicos, y no cortando por el último punto.** La
// diferencia no es académica: cortando a mano, `foo.github.io` y `bar.github.io`
// serían «el mismo sitio» —lo son para el que corta, no para el que aloja— y una
// contraseña acabaría ofrecida a la página de otro. La lista sabe que `github.io`
// es un sufijo público, y también `com.es`, `s3.amazonaws.com` y otros dos mil
// casos que nadie tiene en la cabeza.
//
// Ya estaba en el disco: `golang.org/x/net` viene con Wails desde siempre. Lo
// único que cambia es que pasa de dependencia indirecta a directa, y eso vale la
// pena decirlo porque en su día se decidió lo contrario en la interfaz
// (`frontend/src/componentes.tsx`), donde el dominio solo elige un color y
// equivocarse no cuesta nada.
func dominioRegistrable(anfitrion string) (string, error) {
	anfitrion = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(anfitrion), "."))
	if anfitrion == "" {
		return "", ErrSinDominio
	}
	// Una IP no tiene dominio registrable. `publicsuffix` no lo sabe y devolvería
	// algo con forma de dominio —«0.1» para 127.0.0.1—, que es la clase de
	// respuesta con la que se cometen errores.
	if net.ParseIP(anfitrion) != nil {
		return "", ErrSinDominio
	}
	// Y nada que no sea ASCII. La lista de sufijos trabaja en punycode, así que un
	// dominio con acentos sin convertir no se compara con lo que debería. Antes de
	// arrastrar un conversor de IDNA a un programa que guarda contraseñas, se
	// rechaza y se dice. Es la misma decisión que ya tomó `internal/iconos`.
	for _, r := range anfitrion {
		if r > 127 {
			return "", ErrSinDominio
		}
	}

	dominio, err := publicsuffix.EffectiveTLDPlusOne(anfitrion)
	if err != nil {
		// Pasa con `github.io` a secas, con `com`, y con cualquier anfitrión que
		// **sea** un sufijo público: no hay nada por debajo que registrar, así que
		// no hay sitio con el que comparar.
		return "", ErrSinDominio
	}
	return dominio, nil
}

// Encaja dice si una entrada guardada corresponde a la pestaña que se está
// mirando.
//
// Se compara **dominio registrable contra dominio registrable**, nunca cadena
// contra cadena. Esa es toda la regla, y de ella salen las tres respuestas que
// importan:
//
//	banco.es           ↔ https://www.banco.es/particulares   sí
//	mail.google.com    ↔ https://accounts.google.com         sí
//	banco.es           ↔ https://banco.es.malo.com           NO
//	foo.github.io      ↔ https://bar.github.io               NO
//
// La segunda es una decisión, no un efecto: dos subdominios del mismo dominio
// registrable se dan por el mismo sitio. Es lo que hace que una cuenta guardada
// como `mail.google.com` sirva para entrar por `accounts.google.com`, que es lo
// que pasa de verdad. El precio es que dos cuentas distintas de dos subdominios
// del mismo dominio se ofrecen las dos, y eso es una molestia, no un agujero.
func Encaja(sitioGuardado, dominioDeLaPestana string) bool {
	if dominioDeLaPestana == "" {
		return false
	}
	return DominioDeSitio(sitioGuardado) == dominioDeLaPestana
}
