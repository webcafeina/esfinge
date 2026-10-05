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
	"sort"
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

// ------------------------------------------------------ lo que me han compartido

// Compartida es **una bóveda de otra persona a la que tengo acceso** (ADR 0052).
//
// Vive en el cuerpo cifrado de mi bóveda personal, hermana de `Proyectos`, y por lo
// mismo: con quién trabajo y cómo se llama cada cosa es tan revelador como la lista
// de proyectos propios, y aquí dentro se sincroniza gratis a mis otros equipos **sin
// una ruta nueva en el servidor y sin que el servidor sepa un solo nombre**.
//
// **No es un `Proyecto` con un campo más**, y tienta serlo porque se parecen. No:
// `Ref` ahí es el nombre del fichero y aquí es la referencia **en la cuenta de otro**;
// `Archivado` significa algo distinto; y una compartida no se entrega, ni se borra
// del servidor, ni se renombra para los demás. Mezclarlas obligaría a un `if` de
// dueño en cada una de esas acciones.
//
// Lo que **no** guarda es la clave de la bóveda: ésa está en la ranura de su propio
// fichero, sellada hacia mi identidad. Guardarla aquí además sería tener la llave en
// dos sitios sin ganar un solo caso de uso — el mismo razonamiento que la ADR 0050
// hizo con los proyectos.
type Compartida struct {
	// Dueno es la cuenta de quien la comparte, hex de 16 bytes, y Ref la referencia
	// **en esa cuenta**. Las dos juntas son la dirección en el servidor.
	Dueno string `json:"dueno"`
	Ref   string `json:"ref"`
	// Nombre es lo que leo yo. Llega el que puso quien la comparte y **lo puedo
	// cambiar**: lo que veo es mío, no lo que la otra persona me dice que vea.
	Nombre string `json:"nombre"`
	// Titular es quién soy yo en esa bóveda: el identificador de mi ranura.
	Titular string `json:"titular"`
	// Permiso es lo que me dijeron que tengo. **Es informativo**: el que manda es el
	// del servidor, y éste solo sirve para no pedirme que teclee algo que va a acabar
	// en un 403.
	Permiso string `json:"permiso"`
	// Huella es la de quien me la compartió, **la que comparé al aceptar**. Se
	// guarda para poder decir después de quién es esto sin preguntarle al servidor.
	Huella string `json:"huella,omitempty"`
	Desde  string `json:"desde"`
	Usado  string `json:"usado,omitempty"`
}

// Compartidas son las que tengo, la última usada arriba.
func (b *Boveda) Compartidas() []Compartida {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := append([]Compartida(nil), b.cont.Compartidas...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Usado > out[j].Usado })
	return out
}

// LaCompartida devuelve una por su dirección.
func (b *Boveda) LaCompartida(dueno, ref string) (Compartida, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, c := range b.cont.Compartidas {
		if c.Dueno == dueno && c.Ref == ref {
			return c, true
		}
	}
	return Compartida{}, false
}

// PonerCompartida la añade o la actualiza por su dirección.
func (b *Boveda) PonerCompartida(c Compartida) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	if c.Dueno == "" || c.Ref == "" {
		return errors.New("Una bóveda compartida sin dirección no se puede guardar")
	}
	if c.Desde == "" {
		c.Desde = ahora().UTC().Format(time.RFC3339)
	}
	for i, v := range b.cont.Compartidas {
		if v.Dueno == c.Dueno && v.Ref == c.Ref {
			b.cont.Compartidas[i] = c
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	b.cont.Compartidas = append(b.cont.Compartidas, c)
	b.cuerpoSucio = true
	return b.guardar()
}

// OlvidarCompartida la saca de mi lista: es como se deja de ver algo que me
// compartieron. **No toca nada de la otra persona**, que no es mía.
func (b *Boveda) OlvidarCompartida(dueno, ref string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	for i, v := range b.cont.Compartidas {
		if v.Dueno == dueno && v.Ref == ref {
			b.cont.Compartidas = append(b.cont.Compartidas[:i], b.cont.Compartidas[i+1:]...)
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	return nil
}

// fundirCompartidas es `fundirProyectos` con otra clave: conjunto por dueño+ref, a
// tres bandas contra la base, para que dejar de ver una aquí no la devuelva el otro
// equipo y aceptar una allí llegue aquí.
func fundirCompartidas(l, r, b []Compartida, hayBase bool) []Compartida {
	clave := func(c Compartida) string { return c.Dueno + "/" + c.Ref }
	en := func(lista []Compartida) map[string]Compartida {
		m := map[string]Compartida{}
		for _, c := range lista {
			m[clave(c)] = c
		}
		return m
	}
	ml, mr, mb := en(l), en(r), en(b)
	out := make([]Compartida, 0, len(ml)+len(mr))
	visto := map[string]bool{}
	for _, m := range []map[string]Compartida{ml, mr} {
		for k := range m {
			if visto[k] {
				continue
			}
			visto[k] = true
			cl, enL := ml[k]
			cr, enR := mr[k]
			cb, enB := mb[k]
			if !((enL && enR) || !hayBase || (enL && !enB) || (enR && !enB)) {
				continue
			}
			switch {
			case !enL:
				out = append(out, cr)
			case !enR:
				out = append(out, cl)
			default:
				out = append(out, unaCompartida(cl, cr, cb, enB))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return clave(out[i]) < clave(out[j]) })
	return out
}

func unaCompartida(l, r, b Compartida, hayBase bool) Compartida {
	out := r
	// Cuándo se abrió por última vez: las dos son verdad, gana la mayor.
	if l.Usado > out.Usado {
		out.Usado = l.Usado
	}
	// El nombre lo decide la base: gana el lado que lo cambió, y si lo cambiaron los
	// dos, el mayor por cadena — arbitrario pero **igual en los dos equipos**.
	switch {
	case !hayBase:
		if l.Nombre > r.Nombre {
			out.Nombre = l.Nombre
		}
	case l.Nombre != b.Nombre && r.Nombre == b.Nombre:
		out.Nombre = l.Nombre
	case l.Nombre != b.Nombre && r.Nombre != b.Nombre && l.Nombre > r.Nombre:
		out.Nombre = l.Nombre
	}
	// **Y el permiso no lo deciden mis equipos: lo decide el servidor.** Aquí se
	// queda el más nuevo que haya llegado, que es el del lado que lo cambió; si
	// cambió en los dos, el más estrecho, porque equivocarse hacia «ver» cuesta un
	// 403 que se explica y equivocarse hacia «editar» cuesta pedirle a alguien que
	// teclee algo que va a acabar rechazado.
	switch {
	case !hayBase:
		if l.Permiso == "ver" || r.Permiso == "ver" {
			out.Permiso = "ver"
		}
	case l.Permiso != b.Permiso && r.Permiso == b.Permiso:
		out.Permiso = l.Permiso
	case l.Permiso != b.Permiso && r.Permiso != b.Permiso:
		out.Permiso = "ver"
	}
	return out
}
