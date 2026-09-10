// Package codigos calcula los códigos de un solo uso de la bóveda.
//
// Es TOTP (RFC 6238) sobre HOTP (RFC 4226): un HMAC de la clave con el número
// de intervalos de treinta segundos transcurridos desde 1970, truncado a seis
// cifras. Cuarenta líneas de biblioteca estándar y ninguna dependencia nueva,
// que en un programa que guarda contraseñas es la mitad del argumento para
// hacerlo aquí dentro en vez de traerse un paquete.
//
// **La bóveda guarda la semilla, no el código.** El código se calcula al
// enseñarlo y caduca solo; guardarlo no tendría sentido y guardar la semilla es
// lo que permite calcularlo sin volver a preguntarle nada a nadie. Por eso este
// paquete no toca disco, no sale a la red y no sabe qué es una bóveda: recibe
// texto y una hora, y devuelve seis cifras.
//
// Lo que **no** hace, y conviene saberlo antes de buscar el fallo en otro sitio:
// no corrige el reloj. Si la máquina va desviada más de treinta segundos, el
// código sale mal y no hay forma de que este paquete se entere —el servidor es
// quien tiene la otra mitad—. Es la misma limitación que tiene cualquier
// autenticador.
package codigos

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Semilla es lo que hace falta para calcular los códigos de una cuenta.
//
// Sale de leer lo que guardó quien exportó de otro gestor, que **no siempre es
// la misma cosa**: unos guardan la cadena en base32 pelada y otros la URI
// `otpauth://` entera, con sus parámetros. Las dos entran por `Leer`.
type Semilla struct {
	Clave     []byte
	Digitos   int
	Periodo   time.Duration
	Algoritmo string // "SHA1", "SHA256" o "SHA512"

	// Cuenta es la etiqueta que traía la URI, si traía alguna. No se usa para
	// calcular nada: sirve para poder decir de quién es el código.
	Cuenta string
}

// Los valores por defecto de RFC 6238, que son los que usa casi todo el mundo.
const (
	digitosPorDefecto = 6
	periodoPorDefecto = 30 * time.Second
)

// Leer entiende una semilla, venga como venga.
//
// Acepta la cadena en base32 tal cual la enseña un servicio —con sus espacios y
// sus minúsculas, que es como se copia de una pantalla— y también la URI
// `otpauth://totp/…` completa, que es lo que exportan varios gestores.
func Leer(texto string) (Semilla, error) {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return Semilla{}, errors.New("Aquí no hay ninguna semilla")
	}
	if strings.HasPrefix(strings.ToLower(texto), "otpauth://") {
		return deURI(texto)
	}
	clave, err := deBase32(texto)
	if err != nil {
		return Semilla{}, err
	}
	return Semilla{
		Clave: clave, Digitos: digitosPorDefecto,
		Periodo: periodoPorDefecto, Algoritmo: "SHA1",
	}, nil
}

// deURI lee una `otpauth://totp/Etiqueta?secret=…&digits=…&period=…`.
//
// **Solo `totp`.** El otro tipo que define la especificación es `hotp`, que va
// por un contador que hay que guardar y subir a cada uso: eso no es una semilla
// que se lee, es un estado que se escribe, y prometerlo a medias sería peor que
// no aceptarlo.
func deURI(texto string) (Semilla, error) {
	u, err := url.Parse(texto)
	if err != nil {
		return Semilla{}, errors.New("Esa dirección de código de un solo uso no se entiende")
	}
	if !strings.EqualFold(u.Host, "totp") {
		return Semilla{}, fmt.Errorf("Esfinge solo sabe de códigos «totp», y ése es «%s»", u.Host)
	}
	q := u.Query()

	clave, err := deBase32(q.Get("secret"))
	if err != nil {
		return Semilla{}, err
	}
	s := Semilla{
		Clave: clave, Digitos: digitosPorDefecto,
		Periodo: periodoPorDefecto, Algoritmo: "SHA1",
		Cuenta: strings.TrimPrefix(u.Path, "/"),
	}

	if v := q.Get("digits"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Semilla{}, fmt.Errorf("«%s» no es un número de cifras", v)
		}
		s.Digitos = n
	}
	if v := q.Get("period"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Semilla{}, fmt.Errorf("«%s» no es un número de segundos", v)
		}
		s.Periodo = time.Duration(n) * time.Second
	}
	if v := q.Get("algorithm"); v != "" {
		s.Algoritmo = strings.ToUpper(v)
	}

	if err := s.valida(); err != nil {
		return Semilla{}, err
	}
	return s, nil
}

// valida comprueba lo que venía en la URI. Se hace aquí y no al calcular para
// que el fallo salga al leer, que es cuando hay una entrada delante a la que
// echarle la culpa.
func (s Semilla) valida() error {
	// El tope de arriba no es capricho: la truncación dinámica de RFC 4226 saca
	// treinta y un bits, así que a partir de diez cifras las de la izquierda
	// serían siempre las mismas y el código valdría menos de lo que aparenta.
	if s.Digitos < 6 || s.Digitos > 10 {
		return fmt.Errorf("Un código de un solo uso tiene entre 6 y 10 cifras, no %d", s.Digitos)
	}
	if s.Periodo <= 0 {
		return errors.New("El código tiene que durar algo más que nada")
	}
	if _, err := s.picadora(); err != nil {
		return err
	}
	return nil
}

func (s Semilla) picadora() (func() hash.Hash, error) {
	switch strings.ToUpper(s.Algoritmo) {
	case "", "SHA1":
		return sha1.New, nil
	case "SHA256":
		return sha256.New, nil
	case "SHA512":
		return sha512.New, nil
	}
	return nil, fmt.Errorf("Esfinge no sabe calcular códigos con «%s»", s.Algoritmo)
}

// deBase32 descifra el alfabeto en el que viajan estas semillas.
//
// Tolerante a propósito con **la forma**, nunca con el contenido: los servicios
// enseñan la semilla partida en grupos de cuatro para poder copiarla a mano, y
// unos la rellenan con «=» y otros no. Espacios, guiones y minúsculas se quitan;
// una letra que no sea del alfabeto es un error y no se ignora, porque una
// semilla a la que le falta un carácter da códigos que no valen y el fallo
// aparecería lejos de aquí, en la pantalla del servicio.
func deBase32(texto string) ([]byte, error) {
	limpio := strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, texto)
	limpio = strings.ToUpper(strings.TrimRight(limpio, "="))
	if limpio == "" {
		return nil, errors.New("Aquí no hay ninguna semilla")
	}
	// El largo se comprueba a mano porque **el descifrador de Go no lo hace**:
	// sin relleno, un grupo final de un solo carácter no le parece un error y
	// devuelve los bytes anteriores como si nada. Una semilla a la que se le ha
	// caído una letra al copiarla entraría entera y daría códigos que no valen.
	// Los restos posibles de un grupo de ocho son 0, 2, 4, 5 y 7; los demás no
	// existen.
	switch len(limpio) % 8 {
	case 0, 2, 4, 5, 7:
	default:
		return nil, errors.New("A esa semilla le falta o le sobra algún carácter")
	}

	clave, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(limpio)
	if err != nil {
		return nil, errors.New("Esa semilla no está en base32; cópiala otra vez del servicio")
	}
	if len(clave) == 0 {
		return nil, errors.New("Esa semilla se queda en nada al descifrarla")
	}
	return clave, nil
}

// En devuelve el código que vale en ese instante.
func (s Semilla) En(t time.Time) (string, error) {
	if err := s.valida(); err != nil {
		return "", err
	}
	nueva, _ := s.picadora() // valida ya lo ha comprobado

	var contador [8]byte
	binary.BigEndian.PutUint64(contador[:], uint64(t.Unix()/int64(s.Periodo/time.Second)))

	m := hmac.New(nueva, s.Clave)
	m.Write(contador[:])
	suma := m.Sum(nil)

	// Truncación dinámica de RFC 4226: los cuatro bits de abajo del último byte
	// dicen por dónde cortar, y el bit de arriba se tira para que el número no
	// salga negativo al leerlo con signo.
	desde := suma[len(suma)-1] & 0x0f
	valor := binary.BigEndian.Uint32(suma[desde:desde+4]) & 0x7fffffff

	diez := uint32(1)
	for i := 0; i < s.Digitos; i++ {
		diez *= 10
	}
	return fmt.Sprintf("%0*d", s.Digitos, valor%diez), nil
}

// Quedan dice cuánto le sobra de vida al código que vale ahora.
//
// Se cuenta contra el reloj y no contra cuándo se pidió: dos códigos leídos con
// un segundo de diferencia caducan **a la vez**, porque los intervalos son fijos
// desde 1970 y no empiezan cuando a uno le apetece mirar.
func (s Semilla) Quedan(t time.Time) time.Duration {
	if s.Periodo <= 0 {
		return 0
	}
	return s.Periodo - time.Duration(t.UnixNano()%int64(s.Periodo))
}
