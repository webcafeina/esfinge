package boveda

// La bóveda en varios equipos (ADR 0038, plan en docs/cuentas.md).
//
// El servidor guarda **el mismo documento que hay en el disco**, cifrado igual y
// sin nada que pueda leer. Lo que se añade aquí son tres cosas:
//
//   - PrepararSubida: el documento tal como se sube, sin las ranuras que son solo
//     de este equipo y con la versión del servidor sellada dentro.
//   - Fundir: juntar lo que ha llegado del servidor con lo de aquí, a tres bandas
//     contra la última versión común.
//   - Posesion: la prueba de que se tiene la clave de bóveda, para el servidor,
//     sin darle la clave.
//
// **Un error aquí se copia a todos los equipos de la cuenta**, y por eso la
// fusión prefiere siempre conservar: una edición gana a un borrado, la contraseña
// que pierde un choque se queda en el historial, y una fusión que se llevaría
// media bóveda no se aplica sola.

import (
	"bytes"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// ranurasLocales son las que abren la bóveda **solo en este equipo** —Touch ID,
// Windows Hello, un PIN— y no se suben nunca. Otro equipo no sabría qué hacer con
// ellas, y una ranura de PIN en el servidor sería una forma de abrir la bóveda
// con cuatro cifras.
var ranurasLocales = map[string]bool{"llavero-del-sistema": true, "pin": true}

var (
	// ErrOtraBoveda: lo que llega es una bóveda, pero no ésta.
	ErrOtraBoveda = errors.New("Esa bóveda no es la de esta cuenta")
	// ErrRetroceso: el servidor dice una versión y la bóveda lleva otra dentro, o
	// devuelve una más vieja que la que ya se vio. Puede ser un fallo del servidor
	// o alguien sirviendo una copia vieja; en los dos casos no se funde.
	ErrRetroceso = errors.New("El servidor ha devuelto una versión de la bóveda que no cuadra")
	// ErrMuchosBorrados: la fusión se llevaría más de la mitad de las entradas.
	ErrMuchosBorrados = errors.New("Juntar los cambios de otro equipo borraría más de la mitad de la bóveda")
)

// Fusion cuenta lo que ha hecho Fundir.
type Fusion struct {
	// Traidas son las entradas que han llegado o cambiado desde el otro lado.
	Traidas int
	// Borradas son las entradas vivas aquí que se van porque se borraron allí.
	Borradas int
	// Conflictos son las entradas tocadas en los dos lados a la vez.
	Conflictos int
	// Cambio dice si la bóveda de aquí ha cambiado y se ha guardado.
	Cambio bool
	// Subir dice si lo que queda aquí es distinto de lo que hay en el servidor:
	// entonces hay que subirlo.
	Subir bool
	// Serie es la del fichero justo después de fundir, **leída sin soltar el
	// cerrojo**. Leerla después con Serie() dejaría un hueco por el que un guardado
	// de otro hilo —el navegador guardando una contraseña— se daría por sincronizado
	// sin haberse subido.
	Serie int64
}

// Serie devuelve el contador de guardados de este fichero.
func (b *Boveda) Serie() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.doc.Serie
}

// ID es el identificador de la bóveda, el mismo en todos sus equipos.
func (b *Boveda) ID() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.doc.ID
}

// GuardarEn le da fichero a una bóveda abierta en memoria y la guarda ahí.
func (b *Boveda) GuardarEn(ruta string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ruta = ruta
	b.cuerpoSucio = true
	return b.guardar()
}

// Posesion es la prueba de que se tiene la clave de bóveda.
//
// Se deriva de la clave de bóveda, que **no cambia nunca**: ni al cambiar la
// contraseña maestra ni al rotar la de recuperación. Por eso sirve para demostrar
// al servidor, sin dársela, que quien pide cambiar la contraseña de la cuenta
// tiene la bóveda abierta —con la maestra o con la clave de recuperación—. El
// servidor guarda un HMAC de esto, no esto.
//
// No lleva el identificador de la cuenta, como decía el plan: al darse de alta
// todavía no se sabe, porque lo asigna el servidor en ese mismo paso. Tampoco
// hace falta: la clave de bóveda ya es única de cada bóveda.
func (b *Boveda) Posesion() ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return nil, ErrCerrada
	}
	return PosesionDeLlave(b.llave)
}

// IDDe lee el identificador de una bóveda sin abrirla: para saber si la de este
// equipo y la de una cuenta son la misma antes de pedir ninguna contraseña.
func IDDe(datos []byte) (string, error) {
	doc, err := leerDocumento(datos)
	if err != nil {
		return "", err
	}
	return doc.ID, nil
}

// SellarSecreto cifra algo pequeño con la clave de bóveda: la sesión de la
// cuenta, que se guarda en el disco pero **solo sirve con la bóveda abierta**.
// Un disco robado sin la contraseña maestra no habla con el servidor.
func (b *Boveda) SellarSecreto(claro []byte) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return "", ErrCerrada
	}
	return cripto.SellarTexto(claro, b.llave, cripto.PerfilLlave)
}

// AbrirSecreto es lo contrario de SellarSecreto.
func (b *Boveda) AbrirSecreto(sellado string) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return nil, ErrCerrada
	}
	return cripto.AbrirTexto(sellado, b.llave)
}

// Traer mete en esta bóveda las entradas de otra —la que había en un equipo antes
// de entrar en la cuenta, cuando se elige «Juntar»— **con sus identificadores**,
// las de la papelera incluidas, y sus sitios excluidos. Lo que ya está no se
// toca. Devuelve cuántas ha traído y guarda.
//
// **Lo que ya está es también la misma cuenta con otro identificador** (ver
// repetidas.go): dos bóvedas importadas por separado del mismo gestor tienen las
// mismas cuentas con identificadores distintos, y juntarlas comparando solo el
// identificador las dejaba todas dos veces. La que llega se junta en la que hay
// —lo que traiga de más, sin cambiar nada de lo que había— y no se trae.
func (b *Boveda) Traer(otra *Boveda) (int, error) {
	otra.mu.Lock()
	suyas := append([]Entrada(nil), otra.cont.Entradas...)
	excluidos := append([]string(nil), otra.cont.SitiosExcluidos...)
	otra.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return 0, ErrCerrada
	}
	hay := map[string]bool{}
	cuenta := map[string]int{} // claveDeCuenta → posición en b.cont.Entradas
	for i, e := range b.cont.Entradas {
		hay[e.ID] = true
		if !e.Papelera {
			cuenta[claveDeCuenta(e)] = i
		}
	}
	cuando := ahora().UTC().Format(time.RFC3339)
	traidas := 0
	for _, e := range suyas {
		if hay[e.ID] {
			continue
		}
		if i, esta := cuenta[claveDeCuenta(e)]; esta && !e.Papelera && !chocan(b.cont.Entradas[i], e) {
			antes := contenidoDe(b.cont.Entradas[i])
			juntarEn(&b.cont.Entradas[i], e)
			if contenidoDe(b.cont.Entradas[i]) != antes {
				b.cont.Entradas[i].Cambiada = cuando
				b.cont.Entradas[i].Revision++
			}
			continue
		}
		b.cont.Entradas = append(b.cont.Entradas, e)
		hay[e.ID] = true
		if !e.Papelera {
			cuenta[claveDeCuenta(e)] = len(b.cont.Entradas) - 1
		}
		traidas++
	}
	b.cont.SitiosExcluidos = fundirConjunto(b.cont.SitiosExcluidos, excluidos, nil, false)
	b.cuerpoSucio = true
	if b.ruta == "" {
		return traidas, nil // se guarda al darle fichero
	}
	return traidas, b.guardar()
}

// PrepararSubida devuelve la bóveda tal como se sube al servidor como `version`.
//
// Es la misma que hay en el disco con dos diferencias: **sin las ranuras de este
// equipo**, y con la versión sellada dentro para que nadie pueda servirla después
// como otra. Quitar una ranura obliga a volver a sellar —si no, el sello del otro
// equipo echaría en falta su huella y diría que está manipulada—, así que el sello
// se hace de nuevo. **No toca el fichero de aquí.**
//
// Devuelve también la serie del fichero que lleva dentro, por la misma razón que
// Fusion.Serie.
func (b *Boveda) PrepararSubida(version int64) ([]byte, int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.prepararSubida(version, nil)
}

// prepararSubida arma el documento que se sube, con el cerrojo cogido. Con
// `cambio`, esa ranura sustituye a la de su tipo **solo en lo que se sube**: el
// fichero de aquí no se toca.
func (b *Boveda) prepararSubida(version int64, cambio *sobre) ([]byte, int64, error) {
	if b.llave == nil {
		return nil, 0, ErrCerrada
	}
	if b.cuerpoSucio {
		return nil, 0, errors.New("La bóveda tiene cambios sin guardar")
	}
	doc := b.doc
	doc.Sobres = nil
	sel := sello{
		ID: doc.ID, Serie: doc.Serie, Sincro: version, Cuerpo: huellaDe(doc.Cuerpo),
		Huellas: map[string]string{}, Sobres: map[string]string{},
	}
	puesto := false
	for _, s := range b.doc.Sobres {
		if ranurasLocales[s.Tipo] {
			continue
		}
		if cambio != nil && s.Tipo == cambio.Tipo {
			s, puesto = *cambio, true
		}
		doc.Sobres = append(doc.Sobres, s)
		sel.Huellas[s.Tipo] = huellaDe(s.Contenedor)
		sel.Sobres[s.Tipo] = huellaDeSobre(s)
	}
	if cambio != nil && !puesto {
		doc.Sobres = append(doc.Sobres, *cambio)
		sel.Huellas[cambio.Tipo] = huellaDe(cambio.Contenedor)
		sel.Sobres[cambio.Tipo] = huellaDeSobre(*cambio)
	}
	crudo, err := json.Marshal(sel)
	if err != nil {
		return nil, 0, err
	}
	doc.Sello, err = cripto.SellarTexto(crudo, b.llave, cripto.PerfilLlave)
	if err != nil {
		return nil, 0, err
	}
	fuera, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, 0, err
	}
	return append(fuera, '\n'), doc.Serie, nil
}

// MaestraNueva es una ranura de contraseña maestra preparada y **todavía sin
// poner** en la bóveda: la de una cuenta, que primero tiene que aceptar el
// servidor.
type MaestraNueva struct{ s sobre }

// SubidaConMaestra prepara el cambio de contraseña de una cuenta: el documento que
// se sube como `version`, ya con la ranura de `nueva`, y esa ranura para ponerla
// aquí **solo cuando el servidor diga que sí** (PonerMaestra). Si el servidor dice
// que no, aquí no ha cambiado nada y las dos contraseñas siguen siendo la misma.
func (b *Boveda) SubidaConMaestra(nueva string, version int64) ([]byte, MaestraNueva, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return nil, MaestraNueva{}, ErrCerrada
	}
	if strings.TrimSpace(nueva) == "" {
		return nil, MaestraNueva{}, errors.New("La contraseña maestra no puede estar vacía")
	}
	s, err := envolver(RanuraMaestra, nueva, b.llave, ahora().UTC().Format(time.RFC3339))
	if err != nil {
		return nil, MaestraNueva{}, err
	}
	datos, _, err := b.prepararSubida(version, &s)
	return datos, MaestraNueva{s}, err
}

// PonerMaestra pone aquí la ranura que ya aceptó el servidor, y guarda. Devuelve
// la serie del fichero tras guardar, leída sin soltar el cerrojo.
func (b *Boveda) PonerMaestra(m MaestraNueva) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return 0, ErrCerrada
	}
	if i := b.ranura(RanuraMaestra); i >= 0 {
		b.doc.Sobres[i] = m.s
	} else {
		b.doc.Sobres = append(b.doc.Sobres, m.s)
	}
	if err := b.guardar(); err != nil {
		return 0, err
	}
	return b.doc.Serie, nil
}

// LlaveDeRecuperacion abre el sobre de recuperación que entrega el servidor al
// recuperar una cuenta y devuelve la clave de la bóveda. Solo sirve para sacar la
// prueba de posesión (PosesionDeLlave): la bóveda se abre después, entera.
func LlaveDeRecuperacion(contenedor, clave string) ([]byte, error) {
	norm, err := Normalizar(clave)
	if err != nil {
		return nil, err
	}
	llave, err := cripto.AbrirTexto(contenedor, []byte(norm))
	if err != nil {
		return nil, errors.New("Esa clave de recuperación no es la de esta cuenta")
	}
	return llave, nil
}

// PosesionDeLlave es Posesion a partir de la clave de bóveda suelta.
func PosesionDeLlave(llave []byte) ([]byte, error) {
	clave, err := base64.RawURLEncoding.DecodeString(string(llave))
	if err != nil {
		return nil, err
	}
	defer cripto.Borrar(clave)
	return hkdf.Key(sha256.New, clave, nil, "esfinge/cuenta/posesion/v1", 32)
}

// AbrirConLaLlaveDe abre el fichero de este equipo con la clave de otra bóveda
// abierta —la misma bóveda, bajada del servidor—. Es el equipo que se quedó con la
// contraseña de antes: la nueva no abre su fichero, pero la clave de bóveda es la
// misma, y así se puede fundir lo de aquí con lo del servidor sin perder nada.
// **No escribe nada** al abrir.
func AbrirConLaLlaveDe(ruta string, otra *Boveda) (*Boveda, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	doc, err := leerDocumento(datos)
	if err != nil {
		return nil, err
	}
	otra.mu.Lock()
	llave := append([]byte(nil), otra.llave...)
	id := otra.doc.ID
	otra.mu.Unlock()
	if doc.ID != id {
		return nil, ErrOtraBoveda
	}
	sel, cont, err := desempaquetar(doc, llave)
	if err != nil {
		return nil, err
	}
	return &Boveda{ruta: ruta, doc: doc, sel: sel, cont: cont, llave: llave, soloLectura: doc.Formato < Formato}, nil
}

// OpcionesDeFusion cambian lo que Fundir se atreve a hacer solo.
type OpcionesDeFusion struct {
	// AunqueBorreMucho aplica la fusión aunque se lleve más de la mitad de las
	// entradas. Solo cuando una persona lo ha visto y ha dicho que sí.
	AunqueBorreMucho bool
}

// Fundir junta en esta bóveda lo que ha llegado del servidor.
//
// `remoto` es la bóveda del servidor en su `version`; `base`, la última versión
// del servidor que vio este equipo —la común a los dos lados—, o nada si no se
// tiene. Con base, cada lado aporta lo que cambió desde ella; sin base no se
// puede saber quién cambió qué, y se hace lo más prudente: unir.
//
// Si algo cambia aquí, **se guarda**. Si la fusión borra entradas, antes se deja
// una copia del fichero como estaba en `<ruta>.antes-de-fundir`.
func (b *Boveda) Fundir(remoto []byte, version int64, base []byte, o OpcionesDeFusion) (Fusion, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return Fusion{}, ErrCerrada
	}
	if b.soloLectura {
		return Fusion{}, ErrFormatoNuevo
	}

	docR, err := leerDocumento(remoto)
	if err != nil {
		return Fusion{}, err
	}
	if docR.ID != b.doc.ID {
		return Fusion{}, ErrOtraBoveda
	}
	selR, contR, err := desempaquetar(docR, b.llave)
	if err != nil {
		return Fusion{}, err
	}
	if selR.Sincro != version {
		return Fusion{}, fmt.Errorf("%w (dice la %d y lleva dentro la %d)", ErrRetroceso, version, selR.Sincro)
	}

	// La base es una ayuda, no una condición: si no se puede leer, se funde sin ella.
	var contB *contenido
	var sobresB []sobre
	if len(base) > 0 {
		if docB, err := leerDocumento(base); err == nil && docB.ID == b.doc.ID {
			if _, c, err := desempaquetar(docB, b.llave); err == nil {
				contB = &c
				sobresB = docB.Sobres
			}
		}
	}

	ahoraT := ahora().UTC()
	var f Fusion
	cont := fundirContenido(b.cont, contR, contB, ahoraT, &f)
	sobres := fundirSobres(b.doc.Sobres, docR.Sobres, sobresB, contB != nil)

	f.Cambio = !igualJSON(normal(cont), normal(b.cont)) || !mismasRanuras(sobres, b.doc.Sobres)
	f.Subir = !igualJSON(normal(cont), normal(contR)) || !mismasRanuras(sinLocales(sobres), sinLocales(docR.Sobres))

	// **Perdida es la que desaparece, no la que va a la papelera** (revisión del
	// 2026-09-23). Contando la papelera, mandar tres de cuatro entradas a ella en un
	// equipo —un gesto normal y reversible— era «media bóveda borrada» para todos
	// los demás, que se paraban en seco y además dejaban de subir lo suyo. La
	// papelera es justo el seguro contra ese clic: lo borrado se ve, se restaura y
	// dura treinta días. Lo que este freno tiene que cazar es una fusión que **se
	// lleve** entradas del fichero.
	vivasAntes := vivas(b.cont.Entradas)
	perdidas := 0
	quedan := map[string]bool{}
	for _, e := range cont.Entradas {
		quedan[e.ID] = true
	}
	for id := range vivasAntes {
		if !quedan[id] {
			perdidas++
		}
	}
	f.Borradas = perdidas
	f.Serie = b.doc.Serie
	if !o.AunqueBorreMucho && len(vivasAntes) >= 4 && perdidas*2 > len(vivasAntes) {
		return f, ErrMuchosBorrados
	}
	if !f.Cambio {
		return f, nil
	}

	if perdidas > 0 && b.ruta != "" {
		if antes, err := os.ReadFile(b.ruta); err == nil {
			if err := os.WriteFile(b.ruta+".antes-de-fundir", antes, 0o600); err != nil {
				return f, err
			}
		}
	}
	// **Si el guardado no sale bien, en memoria se queda lo que había** (revisión
	// del 2026-09-23). Guardar falla de verdad: `ErrCambiada` cuando otro Esfinge ha
	// tocado el fichero entre medias, o el disco. Dejando puesto el resultado de la
	// fusión, la ventana enseñaría una bóveda que no está en ninguna parte y el
	// siguiente guardado —cambiar cualquier cosa— la escribiría sin que nadie
	// hubiera decidido eso. Con el error, quien llama vuelve a sincronizar, que es
	// justo lo que hay que hacer.
	contAntes, sobresAntes, sucioAntes := b.cont, b.doc.Sobres, b.cuerpoSucio
	b.cont = cont
	b.doc.Sobres = sobres
	b.cuerpoSucio = true
	if err := b.guardar(); err != nil {
		b.cont, b.doc.Sobres, b.cuerpoSucio = contAntes, sobresAntes, sucioAntes
		f.Cambio, f.Subir = false, false
		return f, err
	}
	f.Serie = b.doc.Serie
	return f, nil
}

// ------------------------------------------------------------------ piezas

func vivas(es []Entrada) map[string]bool {
	m := map[string]bool{}
	for _, e := range es {
		if !e.Papelera {
			m[e.ID] = true
		}
	}
	return m
}

// mismasRanuras compara ranuras sin mirar el orden: dos equipos que las listen
// distinto no han cambiado nada, y si contara se pasarían la bóveda sin fin.
func mismasRanuras(a, b []sobre) bool {
	porTipo := func(ss []sobre) map[string]sobre {
		m := map[string]sobre{}
		for _, s := range ss {
			m[s.Tipo] = s
		}
		return m
	}
	return igualJSON(porTipo(a), porTipo(b))
}

func sinLocales(ss []sobre) []sobre {
	var out []sobre
	for _, s := range ss {
		if !ranurasLocales[s.Tipo] {
			out = append(out, s)
		}
	}
	return out
}

// igualJSON compara dos cosas por su forma en JSON, que es lo que acaba en el
// fichero. `json.Marshal` ordena las claves de los mapas, así que es estable.
func igualJSON(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	return errA == nil && errB == nil && bytes.Equal(ja, jb)
}

// normal deja vacío lo que está vacío, para comparar: una lista sin nada y una
// que no está se escriben distinto en JSON (`[]` y `null`) y significan lo mismo.
// Sin esto, una bóveda vacía parecería cambiar en cada fusión.
func normal(c contenido) contenido {
	if len(c.Entradas) == 0 {
		c.Entradas = nil
	}
	if len(c.SitiosExcluidos) == 0 {
		c.SitiosExcluidos = nil
	}
	if len(c.Lapidas) == 0 {
		c.Lapidas = nil
	}
	if len(c.Extra) == 0 {
		c.Extra = nil
	}
	return c
}

// canon es la forma canónica de una entrada: JSON con las claves en orden
// alfabético a todos los niveles, sin espacios y **sin escapar `<`, `>` ni `&`**,
// que Go escapa por defecto y JavaScript no.
//
// Sirve para comparar y para el último desempate, que hace una huella de esto: la
// extensión, que fundirá en TypeScript, tiene que sacar exactamente los mismos
// bytes (docs/formato-boveda.md).
func canon(e Entrada) string {
	crudo, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	dec := json.NewDecoder(bytes.NewReader(crudo))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return ""
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return ""
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// fundirSobres decide, de cada tipo de ranura, cuál queda.
//
// A tres bandas, como las entradas: si una ranura solo cambió en un lado —la
// contraseña maestra cambiada en otro equipo—, gana ese lado. Si cambió en los
// dos, o no hay base, gana la más reciente, y si empatan, la de contenedor mayor:
// las fechas tienen resolución de un segundo, y cambiar la contraseña en el mismo
// segundo en que se creó la bóveda no puede dejar la vieja.
//
// Se puede fiar de lo que llega porque el sello del documento remoto —cifrado
// con la clave de bóveda— lleva la huella de cada ranura: el servidor no puede
// meter una suya. Las ranuras de este equipo se quedan como están y las de otros
// equipos no entran nunca.
func fundirSobres(l, r, b []sobre, hayBase bool) []sobre {
	porTipo := func(ss []sobre) map[string]sobre {
		m := map[string]sobre{}
		for _, s := range ss {
			m[s.Tipo] = s
		}
		return m
	}
	mL, mR, mB := porTipo(l), porTipo(r), porTipo(b)
	var orden []string
	for _, s := range l {
		orden = append(orden, s.Tipo)
	}
	for _, s := range r {
		if _, hay := mL[s.Tipo]; !hay && !ranurasLocales[s.Tipo] {
			orden = append(orden, s.Tipo)
		}
	}
	out := make([]sobre, 0, len(orden))
	for _, tipo := range orden {
		sl, enL := mL[tipo]
		sr, enR := mR[tipo]
		sb, enB := mB[tipo]
		switch {
		case ranurasLocales[tipo] || !enR:
			out = append(out, sl)
		case !enL:
			out = append(out, sr)
		case sl == sr:
			out = append(out, sl)
		case hayBase && enB && sl == sb:
			out = append(out, sr)
		case hayBase && enB && sr == sb:
			out = append(out, sl)
		case sr.Creado > sl.Creado || (sr.Creado == sl.Creado && sr.Contenedor > sl.Contenedor):
			out = append(out, sr)
		default:
			out = append(out, sl)
		}
	}
	return out
}

func fundirContenido(l, r contenido, b *contenido, ahora time.Time, f *Fusion) contenido {
	var out contenido
	lapidas := unirLapidas(l.Lapidas, r.Lapidas)
	out.Entradas = fundirEntradas(l, r, b, lapidas, ahora, f)
	if len(lapidas) > 0 {
		out.Lapidas = lapidas
	}
	var excluidosB []string
	if b != nil {
		excluidosB = b.SitiosExcluidos
	}
	out.SitiosExcluidos = fundirConjunto(l.SitiosExcluidos, r.SitiosExcluidos, excluidosB, b != nil)
	// **La identidad no se funde campo a campo: se elige una** (ADR 0043). No cambia
	// nunca, así que solo puede haber dos si dos equipos crearon la suya antes de
	// verse, y entonces hace falta que los dos elijan la misma.
	out.Identidad = fundirIdentidad(l.Identidad, r.Identidad)
	var extraB map[string]json.RawMessage
	if b != nil {
		extraB = b.Extra
	}
	out.Extra = fundirSecciones(l.Extra, r.Extra, extraB)
	return out
}

func unirLapidas(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range []map[string]string{a, b} {
		for id, cuando := range m {
			if cuando > out[id] {
				out[id] = cuando
			}
		}
	}
	return out
}

// fundirEntradas decide, entrada a entrada, qué queda.
//
// Con la versión común (base) a mano:
//
//   - igual en los dos lados, o cambiada solo en uno: gana el que cambió;
//   - cambiada en los dos: se funde campo a campo (fundirCampos);
//   - borrada en un lado y cambiada en el otro: **se queda la cambiada**;
//   - borrada en un lado y sin tocar en el otro: se borra.
//
// Sin base, una entrada que solo está en un lado es nueva, salvo que el otro
// tenga su lápida y la entrada no haya cambiado desde entonces.
//
// **El orden es el del servidor**, con lo nuevo de aquí al final. Si cada equipo
// conservara el suyo, la bóveda fundida nunca sería igual a la del servidor y
// los equipos se la pasarían sin fin: lo encontró la prueba de los tres equipos.
func fundirEntradas(l, r contenido, b *contenido, lapidas map[string]string, ahora time.Time, f *Fusion) []Entrada {
	indice := func(es []Entrada) map[string]Entrada {
		m := map[string]Entrada{}
		for _, e := range es {
			m[e.ID] = e
		}
		return m
	}
	mL, mR := indice(l.Entradas), indice(r.Entradas)
	var mB map[string]Entrada
	var lapidasB map[string]string
	if b != nil {
		mB = indice(b.Entradas)
		lapidasB = b.Lapidas
	}

	var orden []string
	visto := map[string]bool{}
	for _, lista := range [][]Entrada{r.Entradas, l.Entradas} {
		for _, e := range lista {
			if !visto[e.ID] {
				visto[e.ID] = true
				orden = append(orden, e.ID)
			}
		}
	}

	// sigue decide si una entrada que solo está en un lado se queda. `p` es la que
	// está; `lapidaAqui` y `lapidaAlli`, las lápidas del lado donde está y del otro.
	sigue := func(p Entrada, eb Entrada, enB bool, lapidaDelOtro, lapidaPropia string) bool {
		if enB {
			// Estaba en la base y el otro lado la ha borrado: sobrevive si se ha
			// tocado desde entonces. La edición gana al borrado.
			return canon(p) != canon(eb)
		}
		if lapidaDelOtro == "" {
			return true // nueva de este lado
		}
		// El otro lado tiene su lápida. Si esa lápida ya estaba en la base y este
		// lado la ha quitado, es que este lado decidió que vive —porque se editó
		// en otro equipo después de borrarla— y esa decisión ya está tomada.
		if b != nil && lapidasB[p.ID] != "" && lapidaPropia == "" {
			return true
		}
		// Sin más que mirar: vive si se cambió después de borrarse.
		return p.Cambiada >= lapidaDelOtro
	}

	var out []Entrada
	for _, id := range orden {
		el, enL := mL[id]
		er, enR := mR[id]
		eb, enB := mB[id]
		switch {
		case enL && enR:
			delete(lapidas, id)
			switch {
			case canon(el) == canon(er):
				out = append(out, er)
			case enB && canon(el) == canon(eb):
				out = append(out, er)
				f.Traidas++
			case enB && canon(er) == canon(eb):
				out = append(out, el)
			default:
				var base *Entrada
				if enB {
					base = &eb
				}
				out = append(out, fundirCampos(el, er, base, ahora))
				f.Conflictos++
			}
		case enL:
			if sigue(el, eb, enB, r.Lapidas[id], l.Lapidas[id]) {
				delete(lapidas, id)
				out = append(out, el)
				continue
			}
			if lapidas[id] == "" {
				lapidas[id] = ahora.Format(time.RFC3339)
			}
		case enR:
			if sigue(er, eb, enB, l.Lapidas[id], r.Lapidas[id]) {
				delete(lapidas, id)
				out = append(out, er)
				f.Traidas++
				continue
			}
			if lapidas[id] == "" {
				lapidas[id] = ahora.Format(time.RFC3339)
			}
		}
	}
	return out
}

// fundirCampos junta una entrada tocada en los dos lados.
//
// Con base, campo a campo: si un campo solo cambió en un lado, gana ese lado;
// si cambió en los dos, gana la entrada «mayor» —más revisiones, luego la fecha,
// luego su huella—, que es un desempate **que da lo mismo lo mire quien lo mire**:
// si cada equipo eligiera el suyo, se pasarían los cambios el uno al otro para
// siempre. Sin base, gana entera la mayor.
//
// Y pase lo que pase, **ninguna contraseña se pierde**: la que no gana va al
// historial de la entrada, junto con los historiales de los dos lados.
func fundirCampos(l, r Entrada, b *Entrada, ahora time.Time) Entrada {
	ganaL := mayor(l, r)
	ganadora, perdedora := r, l
	if ganaL {
		ganadora, perdedora = l, r
	}

	var out Entrada
	if b == nil {
		out = ganadora
	} else {
		ml, mr, mb := aMapa(l), aMapa(r), aMapa(*b)
		fundido := map[string]json.RawMessage{}
		claves := map[string]bool{}
		for _, m := range []map[string]json.RawMessage{ml, mr} {
			for k := range m {
				claves[k] = true
			}
		}
		for k := range claves {
			vl, vr, vb := ml[k], mr[k], mb[k]
			var v json.RawMessage
			switch {
			case bytes.Equal(vl, vr):
				v = vl
			case bytes.Equal(vl, vb):
				v = vr
			case bytes.Equal(vr, vb):
				v = vl
			case ganaL:
				v = vl
			default:
				v = vr
			}
			if v != nil {
				fundido[k] = v
			}
		}
		crudo, _ := json.Marshal(fundido)
		if err := json.Unmarshal(crudo, &out); err != nil {
			out = ganadora
		}
	}

	// **Papelera contra edición: gana la edición** (revisión del 2026-09-23). La
	// regla de la ADR 0038 valía solo para el borrado definitivo: con la papelera,
	// un lado que solo la manda a la papelera y otro que le cambia la contraseña
	// juntaban las dos cosas campo a campo y la entrada acababa **en la papelera con
	// la contraseña nueva**, fuera de la lista y purgada a los treinta días.
	if b != nil {
		soloL, soloR := soloALaPapelera(l, *b), soloALaPapelera(r, *b)
		if soloL != soloR {
			conContenido := r
			if soloR {
				conContenido = l
			}
			out.Papelera, out.BorradaEn = conContenido.Papelera, conContenido.BorradaEn
		}
	}

	var historial [][]Antigua
	historial = append(historial, l.Historial, r.Historial)
	if perdedora.Secreto != "" && perdedora.Secreto != out.Secreto {
		historial = append(historial, []Antigua{{Secreto: perdedora.Secreto, Hasta: ahora.Format(time.RFC3339)}})
	}
	out.Historial = unirHistorial(out.Secreto, historial...)
	out.Revision = max(l.Revision, r.Revision) + 1
	out.Cambiada = max(l.Cambiada, r.Cambiada)
	return out
}

// soloALaPapelera dice si este lado, respecto a la versión común, **no ha hecho
// más que mandarla a la papelera**: ni contraseña, ni título, ni nada.
func soloALaPapelera(x, b Entrada) bool {
	if !x.Papelera || x.Papelera == b.Papelera {
		return false
	}
	sinPapelera := func(e Entrada) string {
		// `Cambiada` también, porque borrar la toca: si no, cualquier borrado
		// parecería un cambio de contenido y esto no diría nunca que sí.
		e.Papelera, e.BorradaEn, e.Revision, e.Cambiada = false, "", 0, ""
		return canon(e)
	}
	return sinPapelera(x) == sinPapelera(b)
}

// mayor dice si `a` gana a `b` en un choque. El orden es total: dos entradas
// distintas nunca empatan, y el resultado no depende de cuál sea la de aquí.
func mayor(a, b Entrada) bool {
	if a.Revision != b.Revision {
		return a.Revision > b.Revision
	}
	if a.Cambiada != b.Cambiada {
		return a.Cambiada > b.Cambiada
	}
	ha, hb := sha256.Sum256([]byte(canon(a))), sha256.Sum256([]byte(canon(b)))
	return hex.EncodeToString(ha[:]) >= hex.EncodeToString(hb[:])
}

func aMapa(e Entrada) map[string]json.RawMessage {
	crudo, _ := json.Marshal(e)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(crudo, &m)
	return m
}

// unirHistorial junta historiales sin repetir contraseñas, lo más reciente
// primero, sin la contraseña de ahora y con el tope de siempre.
func unirHistorial(actual string, listas ...[]Antigua) []Antigua {
	porSecreto := map[string]Antigua{}
	for _, lista := range listas {
		for _, a := range lista {
			if a.Secreto == "" || a.Secreto == actual {
				continue
			}
			if v, hay := porSecreto[a.Secreto]; !hay || a.Hasta > v.Hasta {
				porSecreto[a.Secreto] = a
			}
		}
	}
	out := make([]Antigua, 0, len(porSecreto))
	for _, a := range porSecreto {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Hasta != out[j].Hasta {
			return out[i].Hasta > out[j].Hasta
		}
		return out[i].Secreto < out[j].Secreto
	})
	if len(out) > maximoHistorial {
		out = out[:maximoHistorial]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// fundirConjunto junta dos listas de cosas sueltas —los sitios excluidos—.
// Con base, algo está si está en los dos lados o si lo añadió uno de ellos; sin
// base, se unen.
func fundirConjunto(l, r, b []string, hayBase bool) []string {
	en := func(lista []string) map[string]bool {
		m := map[string]bool{}
		for _, x := range lista {
			m[x] = true
		}
		return m
	}
	sl, sr, sb := en(l), en(r), en(b)
	todos := map[string]bool{}
	for _, m := range []map[string]bool{sl, sr} {
		for x := range m {
			todos[x] = true
		}
	}
	var out []string
	for x := range todos {
		queda := (sl[x] && sr[x]) || !hayBase || (sl[x] && !sb[x]) || (sr[x] && !sb[x])
		if queda {
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

// fundirSecciones junta las secciones del contenido que esta versión no conoce.
// Sin poder entenderlas, se funden enteras: si una cambió en un lado, gana ese
// lado; si en los dos, gana la del servidor.
func fundirSecciones(l, r, b map[string]json.RawMessage) map[string]json.RawMessage {
	claves := map[string]bool{}
	for _, m := range []map[string]json.RawMessage{l, r} {
		for k := range m {
			claves[k] = true
		}
	}
	out := map[string]json.RawMessage{}
	for k := range claves {
		vl, vr, vb := l[k], r[k], b[k]
		var v json.RawMessage
		switch {
		case bytes.Equal(vl, vr):
			v = vl
		case bytes.Equal(vl, vb):
			v = vr
		case bytes.Equal(vr, vb):
			v = vl
		default:
			v = vr
		}
		if v != nil {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
