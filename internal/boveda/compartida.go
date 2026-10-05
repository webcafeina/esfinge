package boveda

// Dar acceso a una bóveda de proyecto (ADR 0052).
//
// Una bóveda compartida es **la misma bóveda de proyecto de siempre** (ADR 0050);
// lo único que cambia es que tiene más de una ranura que la abre: la del dueño
// —`boveda-principal`, envuelta con la clave de bóveda de su personal— y una por
// cada persona con acceso.
//
// Y la ranura de esas personas **no se envuelve, se sella**, que es la decisión
// entera de la ADR:
//
//	Tipo:         "acceso:<idMiembro>"
//	Codificacion: "sobre-x25519-v1"
//	Contenedor:   HPKE(la clave de esta bóveda → su identidad pública)
//
// Lo evidente sería envolverla con la clave de bóveda de esa persona, igual que el
// dueño. Funciona, y **mata la revocación**: solo ella podría volver a crear su
// ranura, así que el día que haya que rotar la clave —que es lo que pasa al quitarle
// el acceso a alguien— el dueño no podría reenvolver para los que quedan sin
// tenerlos delante. Sellando hacia la pública, que está publicada en el servidor
// desde la ADR 0043, el dueño rehace la de cualquiera él solo.
//
// **El tipo lleva dentro a quién es, y eso no es estilo.** El sello indexa los
// sobres por su tipo (`sel.Huellas[tipo]`, `sel.Sobres[tipo]`) y `fundirSobres` los
// mete en un mapa por tipo: dos ranuras del mismo tipo no hacen que sobre una, hacen
// que **la bóveda no abra** —`ErrManipulada`— y que la segunda desaparezca al
// sincronizar sin que nadie se entere.
//
// **Y quitar una ranura no es representable en la fusión**, que es lo que obliga a
// la lápida de aquí abajo: `fundirSobres` une por tipo y conserva la que está en un
// solo lado. Sin lápida, el dueño quita la ranura del revocado, sube, y **el primer
// equipo con una copia de antes la resucita**.

import (
	"crypto/ecdh"
	"crypto/hpke"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	// PrefijoAcceso encabeza el tipo de las ranuras de quien tiene acceso. El resto
	// del tipo es el identificador del miembro, hex de 8 bytes.
	PrefijoAcceso = "acceso:"

	// CodificacionSellada marca que el contenedor no es un `ESF1` envuelto con una
	// llave, sino un sobre sellado hacia una identidad.
	CodificacionSellada = "sobre-x25519-v1"

	// CodificacionRetirada es la lápida: el tipo se queda, el contenedor se vacía.
	//
	// **No se borra el sobre**, por lo que dice la cabecera: la fusión no sabe
	// expresar «esta ranura se quitó», así que borrarla a secas la resucita en
	// cuanto un equipo con una copia de antes sincronice.
	CodificacionRetirada = "retirada"

	// infoDeAcceso separa estos sobres de los de compartir una entrada. Son la misma
	// primitiva y **no tienen por qué poder confundirse**: un sobre de uno no se abre
	// como el otro ni aunque alguien lo intente.
	infoDeAcceso = "esfinge/acceso/v1"

	// marcaDelSellado encabeza el contenedor, como `ESF1` encabeza los otros. Sirve
	// para lo mismo: que un contenedor diga qué es antes de intentar abrirlo.
	marcaDelSellado = "ACC1"
)

// ErrSinAcceso es que esta bóveda no tiene ranura para quien intenta abrirla, o que
// la tenía y se la han quitado.
var ErrSinAcceso = errors.New("No tienes acceso a esa bóveda")

// TipoDeAcceso es el tipo de ranura de un miembro.
func TipoDeAcceso(id string) string { return PrefijoAcceso + id }

// IDDeAcceso saca el miembro del tipo, o vacío si ese tipo no es una ranura de
// acceso.
func IDDeAcceso(tipo string) string {
	if !strings.HasPrefix(tipo, PrefijoAcceso) {
		return ""
	}
	return strings.TrimPrefix(tipo, PrefijoAcceso)
}

// SellarHaciaIdentidad cierra un secreto para que **solo** esa identidad lo abra.
//
// No es un método de `*Boveda` a propósito: sellar necesita la pública de quien
// recibe y nada más, así que esto se puede hacer sobre la bóveda de un proyecto que
// ni siquiera tiene identidad (y no debe tenerla: ver `Identidad`).
func SellarHaciaIdentidad(secreto []byte, para Identidad) (string, error) {
	if para.Suite != Suite {
		return "", fmt.Errorf("esa identidad usa otro cifrado (%s)", para.Suite)
	}
	pub, err := ecdh.X25519().NewPublicKey(para.Cifrado)
	if err != nil {
		return "", err
	}
	destino, err := hpke.NewDHKEMPublicKey(pub)
	if err != nil {
		return "", err
	}
	enc, emisor, err := hpke.NewSender(destino, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), []byte(infoDeAcceso))
	if err != nil {
		return "", err
	}
	// La marca va **dentro de los datos autenticados**: así un contenedor de ésos no
	// se puede hacer pasar por otra cosa cambiándole las letras de delante.
	cuerpo, err := emisor.Seal([]byte(marcaDelSellado), secreto)
	if err != nil {
		return "", err
	}
	return marcaDelSellado + "." + b64.EncodeToString(enc) + "." + b64.EncodeToString(cuerpo), nil
}

// AbrirSellado abre con la identidad **de esta bóveda**, que es la personal de quien
// tiene el acceso.
func (b *Boveda) AbrirSellado(sellado string) ([]byte, error) {
	b.mu.Lock()
	if b.llave == nil {
		b.mu.Unlock()
		return nil, ErrCerrada
	}
	guardada := b.cont.Identidad
	b.mu.Unlock()
	if guardada == nil {
		return nil, errSinIdentidad
	}

	partes := strings.Split(sellado, ".")
	if len(partes) != 3 || partes[0] != marcaDelSellado {
		return nil, ErrSinAcceso
	}
	enc, err := b64.DecodeString(partes[1])
	if err != nil {
		return nil, err
	}
	cuerpo, err := b64.DecodeString(partes[2])
	if err != nil {
		return nil, err
	}

	semilla, err := b64.DecodeString(guardada.Semilla)
	if err != nil {
		return nil, err
	}
	cifrado, _, err := derivarIdentidad(semilla)
	if err != nil {
		return nil, err
	}
	mio, err := hpke.NewDHKEMPrivateKey(cifrado)
	if err != nil {
		return nil, err
	}
	receptor, err := hpke.NewRecipient(enc, mio, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), []byte(infoDeAcceso))
	if err != nil {
		return nil, err
	}
	claro, err := receptor.Open([]byte(marcaDelSellado), cuerpo)
	if err != nil {
		// No se dice qué falló: desde fuera, «no es para ti» y «no cuadra» son lo
		// mismo, y distinguirlos solo ayuda a quien prueba sobres ajenos.
		return nil, ErrSinAcceso
	}
	return claro, nil
}

// PonerAcceso le da acceso a una identidad, o se lo renueva.
//
// Renovar es lo que hace rotar la clave: el mismo miembro, el mismo tipo de ranura y
// un contenedor nuevo con la clave nueva dentro.
func (b *Boveda) PonerAcceso(id string, para Identidad) error {
	if id == "" {
		return errors.New("Falta decir a quién se le da el acceso")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	sellado, err := SellarHaciaIdentidad(b.llave, para)
	if err != nil {
		return err
	}
	s := sobre{
		Tipo:         TipoDeAcceso(id),
		Creado:       ahora().UTC().Format(time.RFC3339),
		Contenedor:   sellado,
		Codificacion: CodificacionSellada,
	}
	if i := b.ranura(s.Tipo); i >= 0 {
		b.doc.Sobres[i] = s
	} else {
		b.doc.Sobres = append(b.doc.Sobres, s)
	}
	return b.guardar()
}

// RetirarAcceso le quita el acceso, **dejando la lápida**.
//
// Lo que esto no hace, y por eso al lado va siempre una rotación de la clave: no
// borra lo que esa persona ya se bajó, y no le quita de la cabeza la clave que vio
// mientras tuvo acceso.
func (b *Boveda) RetirarAcceso(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	tipo := TipoDeAcceso(id)
	i := b.ranura(tipo)
	if i < 0 {
		return nil
	}
	if b.doc.Sobres[i].Codificacion == CodificacionRetirada {
		return nil
	}
	b.doc.Sobres[i] = sobre{
		Tipo:         tipo,
		Creado:       ahora().UTC().Format(time.RFC3339),
		Codificacion: CodificacionRetirada,
	}
	return b.guardar()
}

// Accesos son los miembros con ranura viva en esta bóveda, en el orden del fichero.
//
// Las retiradas no salen: están en el documento para que la fusión no las resucite,
// no para que nadie las cuente como acceso.
func (b *Boveda) Accesos() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return accesosDe(b.doc)
}

// AccesosEn es lo mismo **sin abrir el fichero**: es lo que necesita saber la lista
// de proyectos para decir si un proyecto está compartido, con la bóveda cerrada.
func AccesosEn(ruta string) []string {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	doc, err := leerDocumento(datos)
	if err != nil {
		return nil
	}
	return accesosDe(doc)
}

func accesosDe(doc documento) []string {
	var ids []string
	for _, s := range doc.Sobres {
		if s.Codificacion == CodificacionRetirada {
			continue
		}
		if id := IDDeAcceso(s.Tipo); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// AbrirCompartida abre una bóveda a la que te han dado acceso, con **tu** bóveda
// personal: de ella sale la identidad que abre tu ranura.
//
// **Prueba solo esa ranura**, por lo mismo que `AbrirConElSistema` y `AbrirProyecto`:
// probar todas pagaría antes una derivación interactiva entera contra la maestra, que
// es justo el segundo que esto viene a quitar.
func AbrirCompartida(ruta, id string, personal *Boveda) (*Boveda, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	return AbrirCompartidaBytes(ruta, datos, id, personal)
}

// AbrirCompartidaBytes es lo mismo sobre bytes ya leídos, para abrir lo que acaba de
// llegar del servidor sin pasar por el disco.
func AbrirCompartidaBytes(ruta string, datos []byte, id string, personal *Boveda) (*Boveda, error) {
	if personal == nil {
		return nil, ErrCerrada
	}
	doc, err := leerDocumento(datos)
	if err != nil {
		return nil, err
	}
	i := indiceDeRanura(doc, TipoDeAcceso(id))
	if i < 0 || doc.Sobres[i].Codificacion == CodificacionRetirada {
		return nil, ErrSinAcceso
	}
	llave, err := personal.AbrirSellado(doc.Sobres[i].Contenedor)
	if err != nil {
		return nil, err
	}
	return conLlave(ruta, doc, llave, true)
}

// indiceDeRanura es `(*Boveda).ranura` sobre un documento suelto: hace falta antes de
// tener la bóveda, que es el caso de abrir.
func indiceDeRanura(doc documento, tipo string) int {
	for i, s := range doc.Sobres {
		if s.Tipo == tipo {
			return i
		}
	}
	return -1
}

// ------------------------------------------------------------------ el sobre

// Acceso es lo que viaja en un sobre de acceso: **dónde está la bóveda y quién eres
// tú en ella**.
//
// Lo que **no** lleva es la clave, y eso es la mitad de por qué esto es barato: la
// clave ya está dentro del propio fichero de la bóveda, sellada hacia la identidad
// de quien recibe (ver la cabecera de este fichero). Así que este sobre no es un
// secreto que haya que proteger más de lo que ya se protege cualquier otro: quien lo
// interceptara —y va cifrado— no se llevaría nada con lo que abrir nada.
type Acceso struct {
	// Dueno es la cuenta de quien da el acceso, hex de 16 bytes. Con esto y Ref se
	// arma la ruta del servidor, y por eso va aquí: el servidor no tiene ningún
	// índice de «lo que me han compartido», igual que no tiene los nombres.
	Dueno string `json:"dueno"`
	Ref   string `json:"ref"`
	// Nombre es cómo lo llama quien lo comparte. Quien lo recibe puede cambiarlo en
	// su bóveda: es suyo lo que ve, no lo que la otra persona le dice que vea.
	Nombre string `json:"nombre"`
	// Titular es quién eres tú en esa bóveda: el identificador de tu ranura. Lo
	// elige quien da el acceso, y por eso es también lo que le sirve para quitarlo.
	Titular string `json:"titular"`
	// Permiso es "ver" o "editar". **Es informativo**: el que manda es el del
	// servidor. Aquí sirve para no pedirle a nadie que teclee algo que va a acabar
	// en un 403, que es la regla que costó `ExportarLlaves`.
	Permiso string `json:"permiso"`
}

// MandarAcceso prepara el sobre que le dice a alguien que tiene acceso a una bóveda.
//
// Es el mismo sobre de compartir una entrada con otra carga y otra versión, así que
// **no hay ni una línea de criptografía nueva**: la misma cabecera autenticada, la
// misma firma, la misma huella que se enseña antes. Y como `version` va dentro de lo
// autenticado y de lo firmado, un sobre de acceso no se puede hacer pasar por una
// copia de entrada ni al revés.
func (b *Boveda) MandarAcceso(a Acceso, para Identidad) (Envio, error) {
	claro, err := json.Marshal(a)
	if err != nil {
		return Envio{}, err
	}
	return b.sellarSobre(claro, VersionDeAcceso, para)
}

// AbrirAcceso saca el acceso de un sobre dirigido a esta bóveda, con la identidad de
// quien lo manda **ya comprobada**.
func (b *Boveda) AbrirAcceso(s Envio) (Acceso, Identidad, error) {
	claro, de, err := b.abrirSobre(s, VersionDeAcceso)
	if err != nil {
		return Acceso{}, Identidad{}, err
	}
	var a Acceso
	if err := json.Unmarshal(claro, &a); err != nil {
		return Acceso{}, Identidad{}, err
	}
	if a.Dueno == "" || !refValidaEnBoveda(a.Ref) || a.Titular == "" {
		return Acceso{}, Identidad{}, errors.New("Ese acceso no se entiende")
	}
	return a, de, nil
}

// refValidaEnBoveda es la misma regla que el servidor aplica a una referencia, aquí
// para no aceptar un sobre que no podría llevar a ninguna parte.
func refValidaEnBoveda(ref string) bool {
	if len(ref) != 16 {
		return false
	}
	for _, c := range ref {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
