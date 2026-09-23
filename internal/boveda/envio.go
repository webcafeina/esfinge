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

// Version del formato del sobre. Lo primero que se mira al abrir.
const VersionDeEnvio = 1

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
	claro := []byte(canon(copia))

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
		Version: VersionDeEnvio,
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
	b.mu.Lock()
	if b.llave == nil {
		b.mu.Unlock()
		return Entrada{}, Identidad{}, ErrCerrada
	}
	guardada := b.cont.Identidad
	b.mu.Unlock()
	if guardada == nil {
		return Entrada{}, Identidad{}, errSinIdentidad
	}
	if s.Version > VersionDeEnvio {
		return Entrada{}, Identidad{}, ErrEnvioNuevo
	}
	mia, err := publicaDe(guardada)
	if err != nil {
		return Entrada{}, Identidad{}, err
	}
	if s.Suite != mia.Suite || !igualBytes(s.Para, mia.Cifrado) {
		return Entrada{}, Identidad{}, ErrSobreDeOtro
	}

	// **La firma primero.** Abrir algo que no se sabe de quién es, y decidir
	// después, deja un hueco para que un sobre ajeno gaste el descifrado.
	if len(s.De.Firma) != ed25519.PublicKeySize || !ed25519.Verify(s.De.Firma, loQueSeFirma(s), s.Firma) {
		return Entrada{}, Identidad{}, ErrFirmaDelEnvio
	}

	semilla, err := b64.DecodeString(guardada.Semilla)
	if err != nil {
		return Entrada{}, Identidad{}, err
	}
	priv, _, err := derivarIdentidad(semilla)
	if err != nil {
		return Entrada{}, Identidad{}, err
	}
	mio, err := hpke.NewDHKEMPrivateKey(priv)
	if err != nil {
		return Entrada{}, Identidad{}, err
	}
	receptor, err := hpke.NewRecipient(s.Enc, mio, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), []byte(infoDeEnvio))
	if err != nil {
		return Entrada{}, Identidad{}, ErrSobreDeOtro
	}
	claro, err := receptor.Open(loQueVaAutenticado(s), s.Cuerpo)
	if err != nil {
		return Entrada{}, Identidad{}, ErrSobreDeOtro
	}

	var e Entrada
	if err := json.Unmarshal(claro, &e); err != nil {
		return Entrada{}, Identidad{}, err
	}
	de := Identidad{
		Suite:   s.Suite,
		Cifrado: s.De.Cifrado,
		Firma:   s.De.Firma,
		Huella:  s.HuellaDe(),
	}
	return e, de, nil
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
