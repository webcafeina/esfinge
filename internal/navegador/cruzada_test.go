package navegador

import (
	"bytes"
	"encoding/hex"
	"strings"
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

// **La firma de punta a punta: se firma en un lado y se verifica en el otro.**
//
// Es lo único que dice que la conversión de P1363 a DER está bien sin un sitio de
// verdad. Los bytes de una firma ECDSA **cambian en cada llamada** —lleva azar
// dentro—, así que compararlos no vale para nada: lo que hay que comprobar es que
// la otra parte la acepte. Y las dos direcciones, porque cada lado firma con una
// biblioteca distinta: Go saca DER de una vez con `SignASN1`, y WebCrypto solo da
// `r ‖ s` crudos y hay que convertirlos a mano.
func TestCruzadaFirmaDePuntaAPunta(t *testing.T) {
	// Una llave hecha aquí y otra hecha allí: las dos tienen que valer en los dos
	// sitios, o guardar una llave con cuenta y usarla sin ella no funcionaría.
	privadaGo, publicaGo, err := CrearLlave()
	if err != nil {
		t.Fatal(err)
	}
	var suya struct {
		Privada string `json:"privada"`
		Publica string `json:"publica"`
	}
	cruzada.Pedir(t, map[string]any{"orden": "crearLlave"}, &suya)
	privadaExt, err := hex.DecodeString(suya.Privada)
	if err != nil {
		t.Fatal(err)
	}
	publicaExt, err := hex.DecodeString(suya.Publica)
	if err != nil {
		t.Fatal(err)
	}

	// **Los datos que se firman de verdad**, no un «hola»: el autenticador y el
	// hash del cliente, que es lo que va a firmar una llave de acceso.
	cliente := DatosDelCliente("webauthn.get", []byte("un reto de prueba de 32 bytes.."), "https://github.com")
	datos := LoQueSeFirma(DatosDelAutenticador("github.com", BanderasAlFirmar), cliente)

	// 1 · Firma la extensión con las dos llaves, y verifica Go.
	type caso struct {
		Privada string `json:"privada"`
		Datos   string `json:"datos"`
	}
	var firmas []string
	cruzada.Pedir(t, map[string]any{"orden": "firmarLlave", "casos": []caso{
		{hex.EncodeToString(privadaExt), hex.EncodeToString(datos)},
		{hex.EncodeToString(privadaGo), hex.EncodeToString(datos)},
	}}, &firmas)

	for i, pub := range [][]byte{publicaExt, publicaGo} {
		f, err := hex.DecodeString(firmas[i])
		if err != nil {
			t.Fatal(err)
		}
		if !VerificarFirma(pub, datos, f) {
			t.Errorf("firma %d de la extensión: Go no la acepta (%d bytes, empieza por %#x)", i, len(f), f[0])
		}
		// Y que no acepte cualquier cosa, que si no lo de arriba no dice nada.
		if VerificarFirma(pub, append(datos, 'x'), f) {
			t.Errorf("firma %d: Go acepta la firma sobre otros datos", i)
		}
	}

	// **Y la dirección contraria no está, a propósito.** Sería «firma Go y verifica
	// la extensión», y no se puede sin escribir código que producción no usa:
	// WebCrypto **solo verifica en P1363**, los mismos `r ‖ s` crudos con los que
	// firma, así que para darle la firma de Go habría que escribir un descodificador
	// de DER que no hace falta en ningún sitio. Y tampoco haría falta probarlo: la
	// extensión **nunca verifica** —verifica el sitio— y el DER de Go lo escribe la
	// biblioteca estándar, que no es código nuestro.
	//
	// Lo que sí hay que probar es lo de arriba, y es justo lo contrario de lo que
	// parece: **el DER lo escribimos nosotros solo en la extensión**, a mano, sobre
	// lo que da WebCrypto.

	// 3 · Y la pública que saca Go de una privada de la extensión es la misma que
	// sacó la extensión: si no, al crear una llave se le mandaría al sitio una
	// pública que no corresponde y la cuenta quedaría inaccesible.
	deLaPrivada, err := PublicaDe(privadaExt)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(deLaPrivada) != suya.Publica {
		t.Error("Go saca de la privada de la extensión otra pública distinta")
	}
}

// **El camino entero de firmar una llave de acceso, verificado por Go** (ADR 0048).
//
// La extensión guarda una llave en una bóveda de verdad, atiende el verbo como lo
// pediría el navegador y devuelve la aserción; aquí se comprueba que **la firma la
// acepta la pública**, que el `clientDataJSON` dice lo que tiene que decir y que el
// `authenticatorData` lleva el hash del sitio y el contador a cero.
//
// Es lo único que puede decirlo: WebCrypto **no sabe verificar DER**, así que desde
// la extensión «ha firmado» solo significa que no ha lanzado.
func TestCruzadaAfirmarLlave(t *testing.T) {
	var r struct {
		Publica    string `json:"publica"`
		OK         bool   `json:"ok"`
		Afirmacion *struct {
			IDCredencial         string `json:"idCredencial"`
			IDUsuario            string `json:"idUsuario"`
			DatosDelCliente      string `json:"datosDelCliente"`
			DatosDelAutenticador string `json:"datosDelAutenticador"`
			Firma                string `json:"firma"`
		} `json:"afirmacion"`
	}
	cruzada.Pedir(t, map[string]any{
		"orden":   "afirmarLlave",
		"maestra": "una maestra larga para la prueba cruzada de las llaves",
		"origen":  "https://github.com/login",
		"rpId":    "github.com",
		"reto":    B64URL.EncodeToString([]byte("un reto de treinta y dos bytes..")),
	}, &r)

	if !r.OK || r.Afirmacion == nil {
		t.Fatalf("la extensión no ha firmado: %+v", r)
	}
	a := r.Afirmacion
	if a.IDCredencial != "Y3JlZC0x" || a.IDUsuario != "dXN1LTE" {
		t.Errorf("identificadores: %+v", a)
	}

	de := func(s string) []byte {
		b, err := B64URL.DecodeString(s)
		if err != nil {
			t.Fatalf("no es base64url: %q", s)
		}
		return b
	}
	cliente, autenticador, firma := de(a.DatosDelCliente), de(a.DatosDelAutenticador), de(a.Firma)

	// **Lo que se firma lo escribe este lado también**, así que se compara: el
	// cliente tiene que ser exactamente el que Go escribiría para esos datos.
	quiero := DatosDelCliente("webauthn.get", []byte("un reto de treinta y dos bytes.."), "https://github.com")
	if !bytes.Equal(cliente, quiero) {
		t.Errorf("clientDataJSON:\n  extensión: %s\n  Go:        %s", cliente, quiero)
	}
	if !bytes.Equal(autenticador, DatosDelAutenticador("github.com", BanderasAlFirmar)) {
		t.Errorf("authenticatorData: %x", autenticador)
	}

	publica, err := hex.DecodeString(r.Publica)
	if err != nil {
		t.Fatal(err)
	}
	if !VerificarFirma(publica, LoQueSeFirma(autenticador, cliente), firma) {
		t.Error("la firma de la extensión no la acepta su propia pública")
	}
	// Y que no valga para otra cosa, que si no lo de arriba no dice nada.
	if VerificarFirma(publica, LoQueSeFirma(autenticador, append(cliente, 'x')), firma) {
		t.Error("la firma vale para unos datos que no son los suyos")
	}
}

// **El origen se escribe igual en los dos lados** (ADR 0048).
//
// Va dentro del `clientDataJSON` y el sitio lo compara: cuatro caracteres de
// diferencia y la firma se rechaza sin decir por qué. En la extensión lo escribe
// `new URL(x).origin` y en Go `OrigenDe`, y lo que los podría separar es
// precisamente lo que nadie escribe a mano: el puerto por defecto.
func TestCruzadaOrigen(t *testing.T) {
	casos := []string{
		"https://github.com/login",
		// **El que separa las dos formas.** Pegando esquema y anfitrión daría
		// `https://github.com:443`, y el navegador dice `https://github.com`.
		"https://github.com:443/login",
		"https://github.com:8443/x?y#z",
		"http://ejemplo.com:80/",
		"http://ejemplo.com:8080/",
		"https://LOGIN.EJEMPLO.COM/",
		"https://usuario:clave@github.com/",
		"https://[::1]:8443/",
		"https://github.com.",
		"",
		"javascript:alert(1)",
		"about:blank",
	}
	var suyos []string
	cruzada.Pedir(t, map[string]any{"orden": "origen", "casos": casos}, &suyos)
	for i, c := range casos {
		if quiero := OrigenDe(c); suyos[i] != quiero {
			t.Errorf("%q: Go dice %q y la extensión %q", c, quiero, suyos[i])
		}
	}
}

// **Los bytes que el sitio se guarda para siempre** (ADR 0048, P3).
//
// Al crear una llave, lo que sale hacia el sitio no es una firma que caduca: es el
// identificador de la credencial y **su clave pública**, que el sitio guarda y con
// la que verificará todas las firmas futuras. Si Go y la extensión los escriben
// distinto, una llave creada con cuenta no sirve sin ella y al revés — y eso no se
// ve hasta que alguien no puede entrar.
//
// Los casos están elegidos por lo que puede salir distinto, no por variedad:
//
//   - **Una `x` o una `y` que empiezan por cero.** En el COSE las coordenadas son de
//     largo fijo y **el cero se conserva**; en el DER de una firma se **quita**. Es la
//     misma pareja de reglas contrarias que ya costó un rodeo en la P2, así que aquí
//     va explícita y en las dos coordenadas.
//   - **Una coordenada más corta de 32 bytes**, que es lo que devuelve una biblioteca
//     que recorta: hay que rellenar por delante, no por detrás.
//   - **Identificadores de credencial de largos raros**, incluido uno de más de 255
//     bytes: su largo va en dos bytes, y con el orden cambiado el sitio lee basura.
//   - **Un `rpId` con acentos**, porque el hash es sobre los bytes UTF-8.
func TestCruzadaAtestacion(t *testing.T) {
	treinta := "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e"
	casos := []struct {
		RPID         string `json:"rpId"`
		Banderas     byte   `json:"banderas"`
		IDCredencial string `json:"idCredencial"`
		X            string `json:"x"`
		Y            string `json:"y"`
	}{
		// Lo normal: 32 bytes cada coordenada y un identificador de 16.
		{"github.com", BanderasAlCrear, "00112233445566778899aabbccddeeff",
			"1111111111111111111111111111111111111111111111111111111111111111",
			"2222222222222222222222222222222222222222222222222222222222222222"},
		// **La `x` empieza por cero**, que es el caso que el DER trata al revés.
		{"ejemplo.com", BanderasAlCrear, "aabb",
			"0011111111111111111111111111111111111111111111111111111111111111",
			"2222222222222222222222222222222222222222222222222222222222222222"},
		// Y la `y`, que es un sitio distinto del mapa.
		{"ejemplo.com", BanderasAlCrear, "aabb",
			"1111111111111111111111111111111111111111111111111111111111111111",
			"0000222222222222222222222222222222222222222222222222222222222222"},
		// Coordenadas **cortas**: hay que rellenar por delante hasta 32.
		{"ejemplo.com", BanderasAlCrear, "cc", treinta, treinta},
		// Un identificador **de más de 255 bytes**, para que su largo use los dos bytes.
		{"ejemplo.com", BanderasAlCrear, strings.Repeat("ab", 200),
			"1111111111111111111111111111111111111111111111111111111111111111",
			"2222222222222222222222222222222222222222222222222222222222222222"},
		// Y uno vacío, que es el extremo del otro lado.
		{"ejemplo.com", BanderasAlCrear, "",
			"1111111111111111111111111111111111111111111111111111111111111111",
			"2222222222222222222222222222222222222222222222222222222222222222"},
		// **Un `rpId` que no es ASCII**: el hash es sobre sus bytes UTF-8.
		{"bücher.de", BanderasAlCrear, "dd",
			"1111111111111111111111111111111111111111111111111111111111111111",
			"2222222222222222222222222222222222222222222222222222222222222222"},
	}
	type respuesta struct {
		COSE   string `json:"cose"`
		Datos  string `json:"datos"`
		Objeto string `json:"objeto"`
	}
	var suyas []respuesta
	cruzada.Pedir(t, map[string]any{"orden": "atestacion", "casos": casos}, &suyas)
	for i, c := range casos {
		id, err := hex.DecodeString(c.IDCredencial)
		if err != nil {
			t.Fatal(err)
		}
		x, err := hex.DecodeString(c.X)
		if err != nil {
			t.Fatal(err)
		}
		y, err := hex.DecodeString(c.Y)
		if err != nil {
			t.Fatal(err)
		}
		cose := ClaveCOSE(x, y)
		datos := DatosDelAutenticadorAlCrear(c.RPID, c.Banderas, id, cose)
		quiero := respuesta{
			COSE:   hex.EncodeToString(cose),
			Datos:  hex.EncodeToString(datos),
			Objeto: hex.EncodeToString(ObjetoDeAtestacion(datos)),
		}
		if suyas[i] != quiero {
			t.Errorf("caso %d (%s):\n  Go:        %+v\n  extensión: %+v", i, c.RPID, quiero, suyas[i])
		}
	}
}

// **El orden canónico de un mapa CBOR, ejercitado a propósito** (ADR 0048).
//
// Esta prueba existe porque una mutación pasó en verde: quitando la comparación por
// **largo** de `cborDeMapa`, `TestCruzadaAtestacion` seguía pasando. Y con razón —
// ninguno de los dos mapas que Esfinge escribe de verdad distingue las dos reglas:
// en el COSE todas las claves miden un byte, y en el objeto de atestación ordenar
// por largo y ordenar por bytes dan el mismo resultado.
//
// O sea: la regla de CTAP2 —primero el largo, luego los bytes— **no estaba
// comprobada en ninguna parte**, en ninguno de los dos lados. Se comprueba aquí con
// claves donde sí discrepa, para que el día que haya un mapa con claves de largos
// distintos ya esté bien. Y se comprueba cruzada, porque el espejo de TypeScript
// ordena con su propio comparador y **comparar cadenas allí daría el orden de
// UTF-16**, que es la trampa de la forma canónica de la bóveda otra vez.
func TestCruzadaOrdenDelMapaCBOR(t *testing.T) {
	casos := []struct {
		Pares [][2]string `json:"pares"`
	}{
		// **El caso que la atestación no cubre**: una clave larga que en bytes iría
		// antes que una corta. Ordenando solo por bytes saldría `0x40…` delante de
		// `0x61`; con el largo primero, detrás.
		{[][2]string{{"4041424344", "01"}, {"61", "02"}}},
		// Las claves del COSE, en el orden en que se leen: tienen que salir al revés.
		{[][2]string{{"22", "01"}, {"21", "02"}, {"20", "03"}, {"03", "04"}, {"01", "05"}}},
		// Dos del mismo largo, que se desempatan por bytes.
		{[][2]string{{"ff", "01"}, {"00", "02"}, {"80", "03"}}},
		// Uno vacío y uno de una sola clave: los extremos.
		{nil},
		{[][2]string{{"01", "f6"}}},
	}
	var suyas []string
	cruzada.Pedir(t, map[string]any{"orden": "cbor", "casos": casos}, &suyas)
	for i, c := range casos {
		pares := make([][2][]byte, 0, len(c.Pares))
		for _, p := range c.Pares {
			k, err := hex.DecodeString(p[0])
			if err != nil {
				t.Fatal(err)
			}
			v, err := hex.DecodeString(p[1])
			if err != nil {
				t.Fatal(err)
			}
			pares = append(pares, [2][]byte{k, v})
		}
		quiero := hex.EncodeToString(cborDeMapa(pares))
		if suyas[i] != quiero {
			t.Errorf("caso %d: Go dice %s y la extensión %s", i, quiero, suyas[i])
		}
	}
}

// **El COSE de la misma clave, igual en los dos lados** (ADR 0048, P3).
//
// Es lo que dice que una llave creada con cuenta sirve sin ella y al revés: lo que el
// sitio se guarda es el COSE de la pública, y si los dos lados sacaran coordenadas
// distintas de la misma clave —o las rellenaran distinto— la llave **solo valdría
// donde se creó**, y quien la usara en el otro equipo no podría entrar.
//
// Las llaves las genera la extensión, con su WebCrypto, y Go saca el COSE de la misma
// pública por su camino: `ParsePKIXPublicKey` y `FillBytes`, que no se parece en nada
// a exportar un JWK. Diez, no una: **una `x` empieza por cero una vez de cada 256**, y
// con una sola llave esa rama no se tocaría casi nunca.
func TestCruzadaCOSEDeUnaLlaveNueva(t *testing.T) {
	const cuantas = 10
	var suyas []struct {
		Privada string `json:"privada"`
		Publica string `json:"publica"`
		COSE    string `json:"cose"`
	}
	cruzada.Pedir(t, map[string]any{"orden": "coseDeUnaLlaveNueva", "cuantas": cuantas}, &suyas)
	if len(suyas) != cuantas {
		t.Fatalf("han venido %d llaves y se pidieron %d", len(suyas), cuantas)
	}
	for i, c := range suyas {
		publica, err := hex.DecodeString(c.Publica)
		if err != nil {
			t.Fatal(err)
		}
		cose, err := PublicaEnCOSE(publica)
		if err != nil {
			t.Fatalf("llave %d: Go no entiende la pública que hizo la extensión: %v", i, err)
		}
		if hex.EncodeToString(cose) != c.COSE {
			t.Errorf("llave %d: el COSE no coincide\n  Go:        %s\n  extensión: %s",
				i, hex.EncodeToString(cose), c.COSE)
		}
	}
}

// **Una clave cuyas dos coordenadas empiezan por cero, fijada a mano** (ADR 0048, P3).
//
// Existe porque la cruzada de arriba —diez llaves al azar— **no caza el relleno**: una
// coordenada de P-256 empieza por cero una vez de cada 256, así que con veinte
// coordenadas la rama se toca el 7 % de las veces. Comprobado mutándolo: quitando el
// `FillBytes` de Go, aquella prueba seguía en verde.
//
// Y dejarlo al azar sería peor que no probarlo: una prueba que falla una de cada
// trece veces es un intermitente, que es exactamente lo que este proyecto acaba de
// aprender a no dar por bueno. Así que **vector fijo**, igual que con el recorte de
// ceros del DER, y por el mismo motivo.
//
// La clave se buscó generando hasta encontrar una con `len(x) < 32` **y** `len(y) < 32`,
// para que cubra las dos posiciones del mapa COSE a la vez. La privada no hace falta:
// lo que se compara es cómo se escribe la pública.
func TestCruzadaCOSEConCoordenadasCortas(t *testing.T) {
	casos := []struct {
		SPKI string `json:"spki"`
	}{
		// x de 31 bytes y y de 31 bytes: las dos piden un cero por delante.
		{"3059301306072a8648ce3d020106082a8648ce3d0301070342000400a4c0403261d347d6fa3eace236e6cc4c6ff9fa241" +
			"fe39acd68691f30a925bf002213f187c80776a89f51a0ac8df343d9a697a886e154e961af2976e9dd9290"},
	}
	var suyas []string
	cruzada.Pedir(t, map[string]any{"orden": "coseDeUnaPublica", "casos": casos}, &suyas)
	for i, c := range casos {
		spki, err := hex.DecodeString(c.SPKI)
		if err != nil {
			t.Fatal(err)
		}
		cose, err := PublicaEnCOSE(spki)
		if err != nil {
			t.Fatal(err)
		}
		// **Y se comprueba que de verdad llevan el cero**, no solo que coincidan: si
		// los dos lados se equivocaran igual, coincidirían y estarían mal los dos.
		// El COSE tiene `-2` (`0x21`) seguido de `0x5820` —32 bytes— y luego la `x`.
		marca := []byte{0x21, 0x58, 0x20, 0x00}
		if !bytes.Contains(cose, marca) {
			t.Errorf("la x no lleva el cero de delante: %s", hex.EncodeToString(cose))
		}
		if hex.EncodeToString(cose) != suyas[i] {
			t.Errorf("caso %d: Go dice %s y la extensión %s", i, hex.EncodeToString(cose), suyas[i])
		}
	}
}
