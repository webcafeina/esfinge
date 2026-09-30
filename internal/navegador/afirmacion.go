package navegador

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
)

// Lo que se firma cuando un sitio pide una llave de acceso, **igual que
// `navegador/src/nucleo/afirmacion.ts`** (ADR 0048).
//
// Está en los dos sitios por la misma razón que la bóveda: con cuenta, la
// extensión firma sola; sin ella, se lo pide a Go por el canal. Los bytes tienen
// que ser los mismos, y **el sitio los verifica byte a byte**.

// Las banderas de `authenticatorData`, en su sitio del byte.
const (
	BanderaUP = 0x01 // hubo presencia: alguien pulsó
	BanderaUV = 0x04 // hubo verificación: la bóveda estaba abierta
	BanderaBE = 0x08 // la llave se puede respaldar
	BanderaBS = 0x10 // y está respaldada
	BanderaAT = 0x40 // lleva los datos de la credencial (solo al crear)
)

// BanderasAlFirmar son las de una llave de Esfinge.
//
// `UV` va puesto **aunque el sitio pida `discouraged`**, que es lo que pide
// GitHub: la bóveda abierta más el clic en el banner *es* verificación de
// usuario, hay sitios que la exigen, y un sitio que no la pide la acepta igual.
// Al revés no funciona.
//
// `BE` y `BS` porque una llave de Esfinge **está respaldada y sincronizada**: eso
// es exactamente lo que dicen esos dos bits, y decir otra cosa sería mentirle al
// sitio sobre algo que puede cambiar cómo trata la cuenta.
const BanderasAlFirmar = BanderaUP | BanderaUV | BanderaBE | BanderaBS

// B64URL es base64url sin relleno, que es como WebAuthn escribe todo lo binario.
var B64URL = base64.RawURLEncoding

// DatosDelCliente escribe el `clientDataJSON`, **con las claves en el orden de la
// especificación**.
//
// Se escribe a mano y no con un `struct` por la misma razón por la que la forma
// canónica de la bóveda se escribe a mano: hay que poder decir exactamente qué
// bytes salen. El orden no es alfabético, es el del ejemplo de la
// especificación, y los sitios que comparan cadenas —que los hay— esperan ése.
//
// Se devuelven **los bytes**, no una cadena: el sitio compara lo que le llega con
// lo que él pidió, y volver a serializarlo al otro lado daría otra cosa.
func DatosDelCliente(tipo string, reto []byte, origen string) []byte {
	esc := func(s string) string {
		b, _ := json.Marshal(s)
		return string(b)
	}
	return []byte(`{"type":` + esc(tipo) + `,"challenge":` + esc(B64URL.EncodeToString(reto)) +
		`,"origin":` + esc(origen) + `,"crossOrigin":false}`)
}

// DatosDelAutenticador: `SHA-256(rpId)` ‖ banderas ‖ contador.
//
// **El contador va siempre a cero** (ADR 0048): una llave sincronizada entre
// equipos no puede llevarlo coherente, así que se dice que no se lleva la cuenta,
// que es lo que hacen todos los gestores y lo que ningún sitio rechaza.
func DatosDelAutenticador(rpID string, banderas byte) []byte {
	h := sha256.Sum256([]byte(rpID))
	out := make([]byte, 37)
	copy(out, h[:])
	out[32] = banderas
	// Los cuatro del contador se quedan a cero, que es lo que ya son.
	return out
}

// BanderasAlCrear son las de arriba **más los datos de la credencial**.
//
// `AT` dice que detrás del contador vienen el identificador de la llave y su clave
// pública, que es lo único que el sitio se lleva de aquí para siempre: con eso
// verificará todas las firmas futuras.
const BanderasAlCrear = BanderasAlFirmar | BanderaAT

// AAGUID es **todo ceros, a propósito** (ADR 0048).
//
// Identifica el modelo de autenticador, y los gestores suelen poner el suyo para
// que el sitio enseñe su nombre. Aquí va a cero porque **con `fmt: "none"` es lo
// que dice la especificación** —sin atestación no hay nada que identificar— y
// porque inventarse un identificador de modelo es afirmar algo que nadie ha
// certificado. El coste es que el sitio dirá «una llave de acceso» y no «Esfinge».
var AAGUID = make([]byte, 16)

// DatosDelAutenticadorAlCrear: lo de firmar **más** el AAGUID, el identificador de
// la credencial y su clave pública en COSE.
//
// El largo del identificador va en **dos bytes y en orden de red**, que es lo que
// más se equivoca la gente al escribir esto a mano: con el orden cambiado, el sitio
// lee un largo enorme, se sale del buffer y rechaza la llave sin decir por qué.
func DatosDelAutenticadorAlCrear(rpID string, banderas byte, idCredencial, claveCOSE []byte) []byte {
	out := DatosDelAutenticador(rpID, banderas)
	out = append(out, AAGUID...)
	out = append(out, byte(len(idCredencial)>>8), byte(len(idCredencial)))
	out = append(out, idCredencial...)
	return append(out, claveCOSE...)
}

// LoQueSeFirma: el autenticador y el hash de los datos del cliente, pegados.
func LoQueSeFirma(autenticador, cliente []byte) []byte {
	h := sha256.Sum256(cliente)
	return append(append([]byte{}, autenticador...), h[:]...)
}
