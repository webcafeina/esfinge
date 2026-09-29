package navegador

import (
	"encoding/hex"
	"testing"

	"github.com/webcafeina/esfinge/internal/cruzada"
)

// Qué cuentas son de qué sitio, **igual en Go y en la extensión** (ADR 0040). Con
// cuenta, la extensión lo decide sola, y decidirlo distinto es entregar una
// contraseña al sitio equivocado o no rellenar donde se rellenaba. Los casos son
// los de la tabla de `dominios_test.go` y los que rompen un analizador de URL.
func TestCruzadaDominios(t *testing.T) {
	casos := []string{
		"https://www.banco.es/particulares", "https://banco.es.malo.com/", "https://foo.github.io/x",
		"https://bar.github.io", "https://mail.google.com", "https://accounts.google.com/signin",
		"http://banco.es", "https://127.0.0.1/", "https://[::1]:8443/", "https://localhost/",
		"https://github.io", "https://com", "https://a.b.com.es", "https://s3.amazonaws.com/cubo",
		"https://bücher.de", "https://xn--bcher-kva.de", "https://BANCO.ES.", "banco.es", "www.banco.es/entrar",
		"  banco.es  ", "https://usuario:clave@banco.es", "ftp://banco.es", "javascript:alert(1)", "",
		"https://banco.es:8443/x?y#z", "sub.dominio.co.uk", "https://blogspot.com", "https://yo.blogspot.com",
		"https://agenciatributaria.gob.es", "https://sede.agenciatributaria.gob.es/Sede/",
	}
	type respuesta struct {
		Origen string `json:"origen"`
		Sitio  string `json:"sitio"`
	}
	var suyas []respuesta
	cruzada.Pedir(t, map[string]any{"orden": "dominios", "casos": casos}, &suyas)
	for i, c := range casos {
		origen, err := DominioDeOrigen(c)
		if err != nil {
			origen = "error"
		}
		quiero := respuesta{origen, DominioDeSitio(c)}
		if suyas[i] != quiero {
			t.Errorf("%q: Go dice %+v y la extensión %+v", c, quiero, suyas[i])
		}
	}
}

// **Con qué `rpId` se puede firmar una llave de acceso, igual en los dos lados**
// (ADR 0048).
//
// Es la tabla que más importa de todas las cruzadas, porque esto no decide si se
// rellena: decide **para quién se firma**. Y es donde se caza que la lista de
// sufijos públicos de `golang.org/x/net` y la de `tldts` hayan dejado de
// coincidir, que es un riesgo ya escrito en `dominios.ts` y que aquí deja de ser
// una molestia para ser un agujero.
func TestCruzadaRPID(t *testing.T) {
	casos := []struct {
		RPID   string `json:"rpId"`
		Origen string `json:"origen"`
	}{
		// Lo normal: sin `rpId`, manda el anfitrión **entero**.
		{"", "https://github.com/login"},
		{"", "https://login.ejemplo.com/"},
		{"github.com", "https://github.com/login"},
		// Un sufijo del anfitrión, que es lo que hace un sitio con subdominios.
		{"ejemplo.com", "https://login.ejemplo.com/"},
		{"ejemplo.com", "https://a.b.ejemplo.com/"},
		{"b.ejemplo.com", "https://a.b.ejemplo.com/"},
		// **Lo que no puede pasar**, y es para lo que está la función.
		{"otro.com", "https://ejemplo.com/"},
		{"ejemplo.com", "https://ejemplo.com.malo.com/"},
		{"ejemplo.com", "https://malaejemplo.com/"},
		{"ejemplo.com", "https://xn--maloejemplo.com/"},
		{"malo.com", "https://ejemplo.com/"},
		{"com", "https://ejemplo.com/"},
		// Un sufijo público **privado**: si esto pasara, cualquier página alojada en
		// `github.io` firmaría por todas las demás.
		{"github.io", "https://foo.github.io/"},
		{"blogspot.com", "https://yo.blogspot.com/"},
		{"co.uk", "https://algo.co.uk/"},
		{"gob.es", "https://sede.agenciatributaria.gob.es/"},
		{"agenciatributaria.gob.es", "https://sede.agenciatributaria.gob.es/"},
		// Sin contexto seguro no se firma, y una IP no tiene dominio.
		{"", "http://ejemplo.com/"},
		{"", "https://127.0.0.1/"},
		{"", "https://[::1]:8443/"},
		{"", "https://localhost/"},
		// Formas raras: el punto final, las mayúsculas, el puerto, lo que no es ASCII.
		{"EJEMPLO.COM", "https://login.ejemplo.com/"},
		{"ejemplo.com.", "https://login.ejemplo.com/"},
		{"", "https://login.ejemplo.com:8443/x?y#z"},
		{"", "https://bücher.de/"},
		{"bücher.de", "https://bücher.de/"},
		{"", ""},
		{"", "javascript:alert(1)"},
	}
	var suyas []string
	cruzada.Pedir(t, map[string]any{"orden": "rpid", "casos": casos}, &suyas)
	for i, c := range casos {
		quiero := RPIDPermitido(c.RPID, c.Origen)
		if suyas[i] != quiero {
			t.Errorf("rpId %q en %q: Go dice %q y la extensión %q", c.RPID, c.Origen, quiero, suyas[i])
		}
	}
}

// **Los bytes que se firman, iguales en los dos lados** (ADR 0048).
//
// El sitio los verifica byte a byte, así que esto no admite «parecido»: si Go y
// la extensión escriben el `clientDataJSON` con otro orden de claves, o el
// `authenticatorData` con otro contador, una llave creada con cuenta deja de
// servir sin ella. Y no es teórico: el `clientDataJSON` se escribe a mano en los
// dos precisamente para poder decir qué bytes salen.
func TestCruzadaLoQueSeFirma(t *testing.T) {
	casos := []struct {
		Tipo     string `json:"tipo"`
		Reto     string `json:"reto"`
		Origen   string `json:"origen"`
		RPID     string `json:"rpId"`
		Banderas byte   `json:"banderas"`
	}{
		// Lo que pide GitHub de verdad: reto de 32 bytes y `rpId` igual al anfitrión.
		{"webauthn.get", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
			"https://github.com", "github.com", BanderasAlFirmar},
		// Un subdominio que firma por el dominio de arriba.
		{"webauthn.get", "ffeeddccbbaa99887766554433221100",
			"https://login.ejemplo.com", "ejemplo.com", BanderasAlFirmar},
		// Al crear, que lleva otra bandera y otro tipo.
		{"webauthn.create", "0102030405060708",
			"https://ejemplo.com", "ejemplo.com", BanderasAlFirmar | BanderaAT},
		// **Un origen con puerto y con caracteres que hay que escapar**: el
		// `clientDataJSON` es JSON escrito a mano, así que esto es justo lo que puede
		// salir distinto en los dos lados.
		{"webauthn.get", "00", "https://ejemplo.com:8443", "ejemplo.com", BanderasAlFirmar},
		{"webauthn.get", "00", "https://ejemplo.com/\"raro\"\\", "ejemplo.com", BanderasAlFirmar},
		// Un reto vacío y unas banderas a cero: los extremos.
		{"webauthn.get", "", "https://ejemplo.com", "ejemplo.com", 0},
	}
	type respuesta struct {
		Cliente      string `json:"cliente"`
		Autenticador string `json:"autenticador"`
		Firmado      string `json:"firmado"`
	}
	var suyas []respuesta
	cruzada.Pedir(t, map[string]any{"orden": "firmado", "casos": casos}, &suyas)
	for i, c := range casos {
		reto, err := hex.DecodeString(c.Reto)
		if err != nil {
			t.Fatal(err)
		}
		cliente := DatosDelCliente(c.Tipo, reto, c.Origen)
		autenticador := DatosDelAutenticador(c.RPID, c.Banderas)
		quiero := respuesta{
			Cliente:      hex.EncodeToString(cliente),
			Autenticador: hex.EncodeToString(autenticador),
			Firmado:      hex.EncodeToString(LoQueSeFirma(autenticador, cliente)),
		}
		if suyas[i] != quiero {
			t.Errorf("caso %d (%s en %s):\n  Go:        %+v\n  extensión: %+v", i, c.Tipo, c.Origen, quiero, suyas[i])
		}
	}
}
