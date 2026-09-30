package navegador

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"net"
	"net/url"
	"strings"
)

// RPIDPermitido dice con qué `rpId` se puede firmar en ese origen, o vacío.
//
// **Es la pieza de seguridad de las llaves de acceso** (ADR 0048): la única vía
// por la que esta función de todo el asunto puede *entregar* algo a un atacante.
// Firmar con el `rpId` equivocado no se ve; es una identificación válida en otro
// sitio.
//
// **Y `Encaja` no sirve para esto**, por mucho que se le parezca. Aquélla compara
// dominio registrable contra dominio registrable, y por eso `accounts.google.com`
// y `mail.google.com` son «el mismo sitio»: es lo que se quiere para ofrecer una
// contraseña. WebAuthn pide otra cosa y más estrecha —el `rpId` tiene que ser el
// anfitrión o un sufijo suyo separado por punto— y además **el hash se calcula
// sobre la cadena exacta**, así que dar por buenos dos nombres distintos no es ser
// tolerante: es firmar algo que el sitio rechazará, o peor, firmar para quien no
// es.
//
// Devuelve **la cadena exacta que hay que hashear**, no un dominio registrable.
func RPIDPermitido(rpID, origen string) string {
	u, err := url.Parse(strings.TrimSpace(origen))
	if err != nil || !strings.EqualFold(u.Scheme, "https") {
		return ""
	}
	anfitrion := strings.ToLower(strings.Trim(u.Hostname(), "[]"))
	if anfitrion == "" || net.ParseIP(anfitrion) != nil {
		return ""
	}
	for _, r := range anfitrion {
		if r > 127 {
			return ""
		}
	}
	if _, err := dominioRegistrable(anfitrion); err != nil {
		return ""
	}

	pedido := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(rpID), "."))
	// Sin `rpId`, manda **el anfitrión entero** y no su dominio registrable: en
	// `login.ejemplo.com` el `rpId` por defecto es `login.ejemplo.com`.
	if pedido == "" {
		return anfitrion
	}
	if pedido == anfitrion {
		return pedido
	}
	if !strings.HasSuffix(anfitrion, "."+pedido) {
		return ""
	}
	// **Que tenga algo por debajo que registrar.** Es lo que separa «el dominio de
	// arriba» de «un sufijo público»: `ejemplo.com` sí, `com` y `github.io` no; si
	// no, cualquier página alojada en `github.io` firmaría por todas las demás.
	if _, err := dominioRegistrable(pedido); err != nil {
		return ""
	}
	return pedido
}

// OrigenDe escribe el origen **como lo escribe el navegador**, que es lo que el
// sitio compara.
//
// No vale pegar esquema y anfitrión: con el puerto por defecto escrito a mano
// —`https://github.com:443/`— eso da `https://github.com:443`, y el navegador y el
// sitio dicen `https://github.com`. Una diferencia de cuatro caracteres dentro del
// `clientDataJSON` y el sitio rechaza la firma sin decir por qué. Lo mismo hace
// `new URL(x).origin` en la extensión, y hay una prueba cruzada de tabla.
func OrigenDe(origen string) string {
	u, err := url.Parse(strings.TrimSpace(origen))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	esquema := strings.ToLower(u.Scheme)
	// **Con los corchetes puestos si es IPv6.** `Hostname()` los quita, y sin ellos
	// `https://::1:8443` no es un origen: el navegador escribe `https://[::1]:8443`.
	// Aquí no se va a firmar nunca —una IP no tiene `rpId`— pero una función que
	// contesta mal contesta mal, y lo cazó la prueba cruzada.
	anfitrion := strings.ToLower(u.Hostname())
	if strings.Contains(anfitrion, ":") {
		anfitrion = "[" + anfitrion + "]"
	}
	puerto := u.Port()
	if (esquema == "https" && puerto == "443") || (esquema == "http" && puerto == "80") {
		puerto = ""
	}
	if puerto != "" {
		return esquema + "://" + anfitrion + ":" + puerto
	}
	return esquema + "://" + anfitrion
}

// LargoDelIDDeCredencial son los bytes de azar que identifican una llave.
//
// Treinta y dos, y el número no lo fija WebAuthn: lo elige el autenticador. Se
// escoge así porque **el sitio lo guarda y lo usa como nombre**, y dos llaves con el
// mismo identificador en el mismo sitio serían la misma para él. Con 32 bytes de
// azar eso no ocurre.
const LargoDelIDDeCredencial = 32

// IDDeCredencial saca un identificador nuevo.
func IDDeCredencial() ([]byte, error) {
	b := make([]byte, LargoDelIDDeCredencial)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// PublicaEnCOSE convierte una pública en SPKI al COSE que guarda el sitio.
//
// Va aquí y no en `cbor.go` porque es lo que junta las dos mitades: sacar las
// coordenadas de la clave es cosa de `crypto`, y escribirlas es cosa del formato. El
// llamante no tiene que tocar coordenadas, que es donde está la trampa de los ceros
// por delante.
func PublicaEnCOSE(publicaSPKI []byte) ([]byte, error) {
	k, err := x509.ParsePKIXPublicKey(publicaSPKI)
	if err != nil {
		return nil, errors.New("Esa clave pública no se entiende")
	}
	ec, vale := k.(*ecdsa.PublicKey)
	if !vale || ec.Curve != elliptic.P256() {
		return nil, errors.New("Esfinge solo sabe de llaves P-256")
	}
	// **Aquí solo se extrae; rellenar a 32 es de `ClaveCOSE`.** Estaba en los dos
	// sitios, y eso hacía que ninguna de las dos líneas estuviera probada: quitando
	// una, la otra tapaba el fallo y todas las mutaciones pasaban en verde. El relleno
	// es una invariante del **formato**, así que vive donde se escribe el formato y se
	// prueba allí, con coordenadas cortas metidas a mano.
	return ClaveCOSE(ec.X.Bytes(), ec.Y.Bytes()), nil
}

// CrearLlave hace una llave de acceso nueva: la privada en PKCS#8, la pública en
// SPKI, que es lo mismo que guarda y entiende la extensión.
func CrearLlave() (privada, publica []byte, err error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	privada, err = x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		return nil, nil, err
	}
	publica, err = x509.MarshalPKIXPublicKey(&k.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	return privada, publica, nil
}

// FirmarConLlave firma con una llave guardada y devuelve la firma **en DER**, que
// es lo que WebAuthn exige.
//
// Aquí sale en DER de una vez (`SignASN1`); la extensión firma en P1363 —`r ‖ s`
// crudos— porque es lo único que da WebCrypto, y tiene que convertirla. Esa
// conversión es donde se equivoca todo el mundo, y por eso hay una prueba cruzada
// que **firma allí y verifica aquí**.
func FirmarConLlave(privadaPKCS8, datos []byte) ([]byte, error) {
	k, err := x509.ParsePKCS8PrivateKey(privadaPKCS8)
	if err != nil {
		return nil, err
	}
	priv, vale := k.(*ecdsa.PrivateKey)
	if !vale {
		return nil, errors.New("Esa llave no es de la curva que Esfinge firma")
	}
	h := sha256.Sum256(datos)
	return ecdsa.SignASN1(rand.Reader, priv, h[:])
}

// VerificarFirma comprueba una firma DER contra una pública en SPKI. Existe para
// las pruebas: **Esfinge nunca verifica, verifica el sitio**. Pero sin poder
// verificar aquí, «la firma está bien» no se puede comprobar sin un sitio de
// verdad, y eso es lo que no se hace en este proyecto.
func VerificarFirma(publicaSPKI, datos, firma []byte) bool {
	p, err := x509.ParsePKIXPublicKey(publicaSPKI)
	if err != nil {
		return false
	}
	pub, vale := p.(*ecdsa.PublicKey)
	if !vale {
		return false
	}
	h := sha256.Sum256(datos)
	return ecdsa.VerifyASN1(pub, h[:], firma)
}

// PublicaDe saca la SPKI de una privada en PKCS#8.
func PublicaDe(privadaPKCS8 []byte) ([]byte, error) {
	k, err := x509.ParsePKCS8PrivateKey(privadaPKCS8)
	if err != nil {
		return nil, err
	}
	priv, vale := k.(*ecdsa.PrivateKey)
	if !vale {
		return nil, errors.New("Esa llave no es de la curva que Esfinge firma")
	}
	return x509.MarshalPKIXPublicKey(&priv.PublicKey)
}
