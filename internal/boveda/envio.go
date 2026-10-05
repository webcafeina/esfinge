package boveda

// Mandar una copia de una entrada a otra cuenta (ADR 0035 y 0043).
//
// **Una copia sin permisos**: al recibirla es de quien la recibe, y si cambia hay
// que volver a mandarla. Aquí solo está el sobre —cifrarlo, firmarlo y abrirlo—;
// quién lo lleva y dónde espera es cosa del servidor.
//
// El sobre va cifrado con **HPKE** hacia la llave de quien lo recibe, que no lo
// abre nadie más —tampoco el servidor—, y **firmado** con la de quien lo manda,
// para que el que recibe sepa de quién es y pueda comparar su huella.
//
// Dos cosas que conviene mirar antes de tocar esto:
//
//   - **La forma canónica manda.** Lo que se firma es el JSON canónico del sobre
//     sin la firma, y lo que se cifra es el JSON canónico de la entrada. Si Go y
//     la extensión no escriben los mismos bytes, la firma no cuadra y el sobre no
//     se abre. Lo vigilan las pruebas cruzadas.
//   - **La entrada viaja sin identificador.** Quien la recibe le pone el suyo: es
//     una copia, no la misma entrada en dos bóvedas, y compartir el identificador
//     haría que la fusión de la otra persona creyera que son la misma.

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hpke"
	"encoding/json"
	"errors"
	"fmt"
)

// Las versiones del sobre. **Lo primero que se mira al abrir**, y lo que separa un
// sobre de otro: `version` va dentro de lo autenticado y de lo firmado, así que un
// sobre de una clase no se puede reinterpretar como el de la otra ni cambiándole el
// número.
const (
	// VersionDeEnvio lleva **una copia de una entrada** (ADR 0043).
	VersionDeEnvio = 1
	// VersionDeAcceso lleva **el sitio de una bóveda compartida** (ADR 0052). Y
	// **no lleva su clave**: la clave está dentro del propio fichero de la bóveda,
	// sellada hacia quien recibe. Lo que viaja aquí es dónde está y quién eres tú
	// en ella.
	VersionDeAcceso = 2
)

// VersionMaximaDeEnvio es hasta dónde entiende esta versión de Esfinge.
const VersionMaximaDeEnvio = VersionDeAcceso

const infoDeEnvio = "esfinge/envio/v1"

var (
	// ErrSobreDeOtro: el sobre no viene cifrado hacia esta identidad.
	ErrSobreDeOtro = errors.New("Este envío no es para esta bóveda")
	// ErrFirmaDelEnvio: el sobre se abre pero la firma no cuadra.
	ErrFirmaDelEnvio = errors.New("El envío no lo ha firmado quien dice")
	// ErrEnvioNuevo: viene de una versión que no se entiende.
	ErrEnvioNuevo = errors.New("Este envío lo hizo una versión de Esfinge más nueva")
)

// Envio es el sobre que viaja. **Todo lo que no es el cuerpo va en claro**, a
// propósito: quien recibe tiene que poder ver de quién es antes de abrirlo.
type Envio struct {
	Esfinge string `json:"esfinge"`
	Version int    `json:"version"`
	Suite   string `json:"suite"`
	// De son las llaves públicas de quien manda, para comprobar la firma y
	// enseñar su huella.
	De EnvioDe `json:"de"`
	// Para es la llave de cifrado de quien recibe: sirve para saber, sin probar a
	// abrirlo, si este sobre es para esta bóveda.
	Para []byte `json:"para"`
	Enc  []byte `json:"enc"`
	// Cuerpo es la entrada cifrada: el JSON canónico de la entrada, sin su
	// identificador.
	Cuerpo []byte `json:"cuerpo"`
	Firma  []byte `json:"firma"`
}

// EnvioDe son las llaves públicas de quien manda.
type EnvioDe struct {
	Cifrado []byte `json:"cifrado"`
	Firma   []byte `json:"firma"`
}

// HuellaDe es la de quien manda, para comparar por teléfono.
func (s Envio) HuellaDe() string {
	return HuellaDeIdentidad(s.Suite, s.De.Cifrado, s.De.Firma)
}

// Son dos cosas distintas y conviene no confundirlas, que ya costó una vuelta:
//
//   - **Lo autenticado** es la cabecera **sin el cuerpo**, y va como datos
//     asociados del cifrado. Tiene que ser lo mismo al cerrar y al abrir, y al
//     cerrar el cuerpo todavía no existe.
//   - **Lo firmado** es eso y además el cuerpo ya cifrado, que es lo que ata la
//     firma a este envío y no a otro.
//
// Los dos se construyen a mano y no con el struct: así no puede colarse un campo
// nuevo sin decidir si va autenticado, firmado o ninguna de las dos cosas.
func loQueVaAutenticado(s Envio) []byte {
	b, _ := json.Marshal(map[string]any{
		"de":      map[string]any{"cifrado": s.De.Cifrado, "firma": s.De.Firma},
		"enc":     s.Enc,
		"esfinge": s.Esfinge,
		"para":    s.Para,
		"suite":   s.Suite,
		"version": s.Version,
	})
	return b
}

func loQueSeFirma(s Envio) []byte {
	b, _ := json.Marshal(map[string]any{
		"cabecera": json.RawMessage(loQueVaAutenticado(s)),
		"cuerpo":   s.Cuerpo,
	})
	return b
}

// MandarEntrada prepara el sobre de `e` para `para`, firmado con esta bóveda.
//
// **Se manda una copia sin el identificador ni el historial**: lo que se comparte
// es la contraseña de ahora, no de dónde viene ni por dónde ha pasado.
func (b *Boveda) MandarEntrada(e Entrada, para Identidad) (Envio, error) {
	b.mu.Lock()
	if b.llave == nil {
		b.mu.Unlock()
		return Envio{}, ErrCerrada
	}
	guardada := b.cont.Identidad
	b.mu.Unlock()
	if guardada == nil {
		// Se crea al vuelo con Identidad(), que guarda; aquí solo se avisa.
		return Envio{}, errSinIdentidad
	}
	mia, err := publicaDe(guardada)
	if err != nil {
		return Envio{}, err
	}
	if para.Suite != mia.Suite {
		return Envio{}, fmt.Errorf("esa identidad usa otro cifrado (%s)", para.Suite)
	}

	copia := e
	copia.ID = ""
	copia.Historial = nil
	copia.Papelera, copia.BorradaEn = false, ""
	copia.Revision = 0
	return b.sellarSobre([]byte(canon(copia)), VersionDeEnvio, para)
}

// sellarSobre es lo común a todos los sobres: cifrar hacia quien recibe y firmar con
// esta bóveda. Lo que cambia entre uno y otro es **la carga y la versión**, y la
// versión va dentro de lo autenticado y de lo firmado, así que es lo que los separa.
//
// Se sacó de `MandarEntrada` al añadir el sobre de acceso (ADR 0052), **sin tocar
// `loQueVaAutenticado` ni `loQueSeFirma`**: ahí está la distinción que ya costó una
// vuelta y no hacía falta moverla.
func (b *Boveda) sellarSobre(claro []byte, version int, para Identidad) (Envio, error) {
	b.mu.Lock()
	if b.llave == nil {
		b.mu.Unlock()
		return Envio{}, ErrCerrada
	}
	guardada := b.cont.Identidad
	b.mu.Unlock()
	if guardada == nil {
		return Envio{}, errSinIdentidad
	}
	mia, err := publicaDe(guardada)
	if err != nil {
		return Envio{}, err
	}
	if para.Suite != mia.Suite {
		return Envio{}, fmt.Errorf("esa identidad usa otro cifrado (%s)", para.Suite)
	}

	pub, err := ecdh.X25519().NewPublicKey(para.Cifrado)
	if err != nil {
		return Envio{}, err
	}
	destino, err := hpke.NewDHKEMPublicKey(pub)
	if err != nil {
		return Envio{}, err
	}
	enc, emisor, err := hpke.NewSender(destino, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), []byte(infoDeEnvio))
	if err != nil {
		return Envio{}, err
	}

	s := Envio{
		Esfinge: "envío",
		Version: version,
		Suite:   mia.Suite,
		De:      EnvioDe{Cifrado: mia.Cifrado, Firma: mia.Firma},
		Para:    para.Cifrado,
		Enc:     enc,
	}
	// Lo de fuera va como datos autenticados: cambiar a quién dice que va, o de
	// quién dice que viene, rompe la apertura además de la firma.
	s.Cuerpo, err = emisor.Seal(loQueVaAutenticado(s), claro)
	if err != nil {
		return Envio{}, err
	}

	semilla, err := b64.DecodeString(guardada.Semilla)
	if err != nil {
		return Envio{}, err
	}
	_, firmante, err := derivarIdentidad(semilla)
	if err != nil {
		return Envio{}, err
	}
	s.Firma = ed25519.Sign(firmante, loQueSeFirma(s))
	return s, nil
}

// AbrirEnvio saca la entrada de un sobre dirigido a esta bóveda. Devuelve también
// la identidad de quien lo manda, **ya comprobada**: la firma cuadra.
func (b *Boveda) AbrirEnvio(s Envio) (Entrada, Identidad, error) {
	claro, de, err := b.abrirSobre(s, VersionDeEnvio)
	if err != nil {
		return Entrada{}, Identidad{}, err
	}
	var e Entrada
	if err := json.Unmarshal(claro, &e); err != nil {
		return Entrada{}, Identidad{}, err
	}
	return e, de, nil
}

// abrirSobre es lo común a abrir cualquier sobre: comprobar que es para esta bóveda,
// **comprobar la firma antes de descifrar** y devolver la carga en claro.
//
// `espera` es la versión que el que llama sabe leer. Un sobre de otra clase no se
// abre aquí aunque sea para esta bóveda: lo que saldría sería una carga con cara de
// lo que no es.
func (b *Boveda) abrirSobre(s Envio, espera int) ([]byte, Identidad, error) {
	b.mu.Lock()
	if b.llave == nil {
		b.mu.Unlock()
		return nil, Identidad{}, ErrCerrada
	}
	guardada := b.cont.Identidad
	b.mu.Unlock()
	if guardada == nil {
		return nil, Identidad{}, errSinIdentidad
	}
	if s.Version > VersionMaximaDeEnvio {
		return nil, Identidad{}, ErrEnvioNuevo
	}
	// **Un sobre de una clase no se abre como el de la otra.** Si se dejara, lo que
	// saldría de un acceso leído como entrada sería una entrada vacía con cara de
	// normal.
	if s.Version != espera {
		return nil, Identidad{}, ErrSobreDeOtro
	}
	mia, err := publicaDe(guardada)
	if err != nil {
		return nil, Identidad{}, err
	}
	if s.Suite != mia.Suite || !igualBytes(s.Para, mia.Cifrado) {
		return nil, Identidad{}, ErrSobreDeOtro
	}

	// **La firma primero.** Abrir algo que no se sabe de quién es, y decidir
	// después, deja un hueco para que un sobre ajeno gaste el descifrado.
	if len(s.De.Firma) != ed25519.PublicKeySize || !ed25519.Verify(s.De.Firma, loQueSeFirma(s), s.Firma) {
		return nil, Identidad{}, ErrFirmaDelEnvio
	}

	semilla, err := b64.DecodeString(guardada.Semilla)
	if err != nil {
		return nil, Identidad{}, err
	}
	priv, _, err := derivarIdentidad(semilla)
	if err != nil {
		return nil, Identidad{}, err
	}
	mio, err := hpke.NewDHKEMPrivateKey(priv)
	if err != nil {
		return nil, Identidad{}, err
	}
	receptor, err := hpke.NewRecipient(s.Enc, mio, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), []byte(infoDeEnvio))
	if err != nil {
		return nil, Identidad{}, ErrSobreDeOtro
	}
	claro, err := receptor.Open(loQueVaAutenticado(s), s.Cuerpo)
	if err != nil {
		return nil, Identidad{}, ErrSobreDeOtro
	}

	de := Identidad{
		Suite:   s.Suite,
		Cifrado: s.De.Cifrado,
		Firma:   s.De.Firma,
		Huella:  s.HuellaDe(),
	}
	return claro, de, nil
}

func igualBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
