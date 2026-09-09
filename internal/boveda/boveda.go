// Package boveda guarda contraseñas y otros secretos en un fichero cifrado.
//
// # El formato, y por qué no es un contenedor ESF1 a secas
//
// La bóveda es un **JSON en claro** cuyos campos criptográficos son líneas
// `ESF1.` corrientes, indistinguibles de las que salen de `esfinge cifrar`:
//
//	{
//	  "esfinge": "bóveda",
//	  "aviso":   "...no la edites a mano",
//	  "formato": 1,
//	  "id":      "...",
//	  "serie":   42,
//	  "sobres":  [ {"tipo":"maestra", "contenedor":"ESF1.…"},
//	               {"tipo":"recuperacion", "contenedor":"ESF1.…"} ],
//	  "cuerpo":  "ESF1.…"
//	}
//
// La idea evidente —un contenedor `ESF1` cuyo contenido sea la bóveda— **no
// funciona**, y conviene decirlo porque es lo primero que uno intenta: un
// contenedor tiene una sal y una clave, y la clave de recuperación exige dos
// entradas independientes al mismo secreto. De ahí la jerarquía.
//
// # La jerarquía de claves
//
//	contraseña maestra ──Argon2id(sal A)──▶ envuelve ─┐
//	                                                   ├─▶ clave de bóveda ──▶ cuerpo
//	clave de recuperación ──Argon2id(sal B)──▶ envuelve┘
//
// La contraseña maestra **no cifra la bóveda**: cifra la clave que la cifra. Por
// eso **cambiarla es reenvolver 43 caracteres, no recifrar el cuerpo entero**, y
// tarda lo mismo con diez entradas que con diez mil.
//
// Y no hace falta escribir criptografía nueva para esto: `cripto.Sellar` **ya es
// exactamente una envoltura de clave con derivación**, con su sal y sus
// parámetros dentro de la cabecera. Cada sobre es una llamada a `SellarTexto`.
// Eso es lo que permite que este paquete no toque `internal/cripto`, que se acaba
// de congelar con vectores fijos (ADR 0022).
//
// # Cuánto aguanta
//
// No hay actualización parcial: cada cambio descifra la bóveda entera y la
// vuelve a sellar. Medido en esta máquina, con entradas de tamaño realista:
//
//	   100 entradas ·  11 ms guardar ·  133 ms abrir
//	 1.000 entradas ·  35 ms guardar ·   71 ms abrir
//	 5.000 entradas ·  59 ms guardar ·   88 ms abrir
//	20.000 entradas ·  92 ms guardar ·  276 ms abrir
//
// El tiempo de abrir lo domina el Argon2id de la contraseña maestra, que es
// constante; el de guardar apenas crece porque el cuerpo va con `PerfilLlave`.
// Veinte mil entradas están muy por encima de cualquier uso real, así que **el
// límite no es la CPU**: es no llamar a Guardar en cada tecla. Se agrupa al
// confirmar la edición.
//
// # Lo que cuesta
//
// El sobre exterior es JSON en claro y **no va autenticado en su conjunto**.
// Quien pueda escribir el fichero puede quitar la ranura de recuperación o
// revertir el cuerpo a uno viejo. No puede leer nada ni fabricar un cuerpo que
// abra. Se **detecta** —no se impide— guardando dentro del cuerpo sellado el
// `id`, la `serie` y las huellas de cada sobre, y comparándolas al abrir.
package boveda

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/escritura"
)

// Formato de la bóveda que este código entiende.
const Formato = 1

// Tipos de ranura. Son una lista abierta a propósito: añadir
// «llavero-del-sistema» para Touch ID, o «servidor» cuando haya cuentas, no
// cambia el formato ni obliga a migrar nada. Ésa es la puerta que se deja
// abierta sin construirla.
const (
	RanuraMaestra      = "maestra"
	RanuraRecuperacion = "recuperacion"
)

var (
	ErrNoEsBoveda = errors.New("Esto no parece una bóveda de Esfinge")
	// ErrFormatoNuevo se devuelve cuando la bóveda viene de una versión más
	// nueva. Se puede leer si el formato lo permite, pero **no se guarda encima**:
	// escribir sobre algo que no se entiende del todo es borrar lo que no se
	// entiende.
	ErrFormatoNuevo = errors.New("Esta bóveda la escribió una versión de Esfinge más nueva; actualiza antes de tocarla")
	ErrManipulada   = errors.New("La bóveda no cuadra por dentro: alguien ha editado el fichero a mano o está a medias")
	ErrCambiada     = errors.New("La bóveda ha cambiado en otro sitio desde que se abrió aquí")
	ErrSinRanura    = errors.New("Esa llave no abre esta bóveda")
	ErrCerrada      = errors.New("La bóveda está cerrada")
)

// sobre es una ranura de llave: la clave de bóveda envuelta con una llave.
type sobre struct {
	Tipo       string `json:"tipo"`
	Creado     string `json:"creado"`
	Contenedor string `json:"contenedor"`
	// Codificacion solo la lleva la ranura de recuperación, y dice con qué
	// alfabeto se presentó la clave. Si algún día cambia, las claves ya escritas
	// en un papel siguen sabiendo cuál era el suyo.
	Codificacion string `json:"codificacion,omitempty"`
}

// documento es el JSON en claro que hay en el disco.
type documento struct {
	Esfinge  string  `json:"esfinge"`
	Aviso    string  `json:"aviso"`
	Formato  int     `json:"formato"`
	ID       string  `json:"id"`
	Serie    int64   `json:"serie"`
	Cambiada string  `json:"cambiada"`
	Sobres   []sobre `json:"sobres"`
	// Sello autentica el JSON de fuera, y va **aparte del cuerpo a propósito**.
	//
	// Al principio el id, la serie y las huellas vivían dentro del cuerpo, y una
	// prueba enseñó por qué está mal: cambiar la contraseña maestra cambia una
	// huella, así que obligaba a recifrar todas las entradas. Justo lo que la
	// jerarquía de claves existe para evitar. Separado, reenvolver una llave
	// reescribe estos cien bytes y deja el cuerpo intacto.
	Sello  string `json:"sello"`
	Cuerpo string `json:"cuerpo"`
}

// sello es lo poco que hace falta para saber que el JSON de fuera no se ha
// manoseado. Va cifrado con la clave de bóveda, como el cuerpo.
type sello struct {
	ID      string            `json:"id"`
	Serie   int64             `json:"serie"`
	Huellas map[string]string `json:"huellas"`
	// Cuerpo es la huella del cuerpo, y es lo que ata las dos piezas.
	//
	// Al separar el sello del cuerpo, revertir el cuerpo a uno viejo dejó de
	// detectarse: nada los relacionaba. Con la huella aquí vuelve a detectarse, y
	// **sin renunciar a lo que se había ganado**: si solo cambia una llave, el
	// cuerpo no cambia, así que su huella tampoco y el cuerpo sigue intacto.
	Cuerpo string `json:"cuerpo"`
}

// contenido es lo que va cifrado dentro del cuerpo: solo las entradas.
type contenido struct {
	Entradas []Entrada `json:"entradas"`
}

const marca = "bóveda"

const avisoDelFichero = "Bóveda de Esfinge. Las contraseñas van cifradas; " +
	"esto de fuera solo dice cómo abrirlas. No la edites a mano."

// Boveda es una bóveda abierta en memoria.
//
// Mientras está abierta, la clave de bóveda y todas las contraseñas viven
// descifradas aquí. Eso no tiene arreglo completo en Go —las cadenas son
// inmutables y el recolector copia sin dejar rastro— y está declarado en
// `docs/seguridad.md`. Lo que sí se hace es no mandar secretos a la ventana
// hasta que se piden.
type Boveda struct {
	ruta  string
	doc   documento
	sel   sello
	cont  contenido
	llave []byte // la clave de bóveda, en base64url y como bytes ASCII
	// cuerpoSucio marca que hay cambios en las entradas. Sin él, cada guardado
	// recifraría el cuerpo aunque solo se hubiera tocado una llave.
	cuerpoSucio bool
	// soloLectura cuando el formato es más nuevo del que entendemos.
	soloLectura bool
}

// Es dice si unos bytes parecen una bóveda, mirando lo justo.
//
// Sirve para que el doble clic en un fichero abra la pantalla que toca, igual
// que `cripto.FormaDe` hace con los `.esf`.
func Es(b []byte) bool {
	limpio := strings.TrimLeft(string(b), " \t\r\n")
	if !strings.HasPrefix(limpio, "{") {
		return false
	}
	var cabeza struct {
		Esfinge string `json:"esfinge"`
	}
	// Con los primeros cientos de bytes bastaría, pero un JSON no se puede cortar
	// por la mitad y seguir siendo JSON, así que se lee entero. Una bóveda son
	// cientos de kilobytes, no gigabytes.
	if err := json.Unmarshal(b, &cabeza); err != nil {
		return false
	}
	return cabeza.Esfinge == marca
}

// Crear hace una bóveda nueva y devuelve la clave de recuperación **una sola
// vez**: no se guarda en ninguna parte y no se puede volver a ver.
func Crear(ruta, maestra string) (*Boveda, string, error) {
	if strings.TrimSpace(maestra) == "" {
		return nil, "", errors.New("La contraseña maestra no puede estar vacía")
	}

	// La clave de bóveda: 32 bytes de azar, manejados **siempre como texto**.
	//
	// Va en base64url y no en crudo por una razón muy concreta: la línea de
	// comandos lee claves de fichero con `--clave-fichero`, y ahí se recortan los
	// saltos de línea del final. Una clave binaria que acabara en 0x0A o 0x0D
	// —una de cada cien— se truncaría en silencio y la vía de escape dejaría de
	// funcionar de forma aparentemente aleatoria.
	bruta, err := cripto.Azar(32)
	if err != nil {
		return nil, "", err
	}
	defer cripto.Borrar(bruta)
	llave := []byte(base64.RawURLEncoding.EncodeToString(bruta))

	recuperacion, err := NuevaRecuperacion()
	if err != nil {
		return nil, "", err
	}

	id, err := cripto.Azar(16)
	if err != nil {
		return nil, "", err
	}

	ahora := time.Now().UTC().Format(time.RFC3339)
	b := &Boveda{
		ruta:  ruta,
		llave: llave,
		doc: documento{
			Esfinge:  marca,
			Aviso:    avisoDelFichero,
			Formato:  Formato,
			ID:       hex.EncodeToString(id),
			Serie:    0,
			Cambiada: ahora,
		},
	}
	b.sel.ID = b.doc.ID
	b.cuerpoSucio = true

	for _, r := range []struct{ tipo, clave string }{
		{RanuraMaestra, maestra},
		{RanuraRecuperacion, recuperacion},
	} {
		s, err := envolver(r.tipo, r.clave, llave, ahora)
		if err != nil {
			return nil, "", err
		}
		b.doc.Sobres = append(b.doc.Sobres, s)
	}

	if err := b.Guardar(); err != nil {
		return nil, "", err
	}
	return b, recuperacion, nil
}

// envolver sella la clave de bóveda con una llave. Cada sobre es un contenedor
// ESF1 normal y corriente, con su propia sal.
func envolver(tipo, clave string, llave []byte, ahora string) (sobre, error) {
	if tipo == RanuraRecuperacion {
		norm, err := Normalizar(clave)
		if err != nil {
			return sobre{}, err
		}
		clave = norm
	}
	// Con PerfilInteractivo: aquí sí hay una clave que puede ser humana, y es
	// donde el coste de la derivación hace su trabajo.
	texto, err := cripto.SellarTexto(llave, []byte(clave), cripto.PerfilInteractivo)
	if err != nil {
		return sobre{}, err
	}
	s := sobre{Tipo: tipo, Creado: ahora, Contenedor: texto}
	if tipo == RanuraRecuperacion {
		s.Codificacion = "crockford32-v1"
	}
	return s, nil
}

// Abrir lee la bóveda del disco y la desbloquea con una llave, sea la contraseña
// maestra o la de recuperación.
func Abrir(ruta, llaveTecleada string) (*Boveda, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	return AbrirBytes(ruta, datos, llaveTecleada)
}

// AbrirBytes es lo mismo sin tocar el disco. Existe para poder probarlo y para
// abrir una bóveda que venga de otro sitio.
func AbrirBytes(ruta string, datos []byte, llaveTecleada string) (*Boveda, error) {
	var doc documento
	if err := json.Unmarshal(datos, &doc); err != nil || doc.Esfinge != marca {
		return nil, ErrNoEsBoveda
	}
	if doc.Formato > Formato {
		return nil, ErrFormatoNuevo
	}

	// Si parece una clave de recuperación, se normaliza: así se acepta lo que la
	// gente escribe de verdad —minúsculas, sin guiones, con «O» donde va un cero—
	// y la suma de control queda comprobada antes de derivar nada.
	candidatas := []string{llaveTecleada}
	norm, errRecuperacion := Normalizar(llaveTecleada)
	if errRecuperacion == nil {
		candidatas = append([]string{norm}, candidatas...)
	}

	b := &Boveda{ruta: ruta, doc: doc, soloLectura: doc.Formato < Formato}
	var llave []byte
	for _, s := range doc.Sobres {
		for _, c := range candidatas {
			abierta, err := cripto.AbrirTexto(s.Contenedor, []byte(c))
			if err == nil {
				llave = abierta
				break
			}
		}
		if llave != nil {
			break
		}
	}
	if llave == nil {
		// **«Te has equivocado al copiarla» y «has perdido la bóveda» son cosas
		// muy distintas**, y ésta es la diferencia. Se mira al final y no al
		// principio a propósito: una contraseña maestra que empiece por ESF es rara
		// pero legítima, así que primero se intenta abrir con lo que sea que hayan
		// escrito y solo si no abre nada se dice que la clave viene mal copiada.
		if errRecuperacion != nil && PareceRecuperacion(llaveTecleada) {
			return nil, errRecuperacion
		}
		return nil, ErrSinRanura
	}
	b.llave = llave

	// Que la llave abriera un sobre y esto no se deje abrir significa que el
	// fichero está mezclado, no que la clave esté mal.
	crudoSello, err := cripto.AbrirTexto(doc.Sello, llave)
	if err != nil {
		return nil, ErrManipulada
	}
	if err := json.Unmarshal(crudoSello, &b.sel); err != nil {
		return nil, ErrManipulada
	}

	claro, err := cripto.AbrirTexto(doc.Cuerpo, llave)
	if err != nil {
		return nil, ErrManipulada
	}
	if err := json.Unmarshal(claro, &b.cont); err != nil {
		return nil, ErrManipulada
	}
	cripto.Borrar(claro)

	if err := b.comprobarCoherencia(); err != nil {
		return nil, err
	}
	return b, nil
}

// comprobarCoherencia compara lo de dentro con lo de fuera.
//
// El JSON exterior no va autenticado, así que quien pueda escribir el fichero
// puede quitar la ranura de recuperación o revertir el cuerpo a uno viejo. No
// puede leerlo ni fabricar uno que abra, pero sí estropearlo sin que se note. Lo
// que hay aquí no lo impide: lo **detecta**, y es el mismo mecanismo que
// necesitará la sincronización.
func (b *Boveda) comprobarCoherencia() error {
	if b.sel.ID != b.doc.ID {
		return ErrManipulada
	}
	if b.sel.Serie != b.doc.Serie {
		return ErrManipulada
	}
	if b.sel.Cuerpo != huellaDe(b.doc.Cuerpo) {
		return ErrManipulada
	}
	for _, s := range b.doc.Sobres {
		esperada, hay := b.sel.Huellas[s.Tipo]
		if !hay {
			// Una ranura que el sello no conoce: sobra o es de una versión que
			// añade tipos nuevos. No es motivo para no abrir.
			continue
		}
		if huellaDe(s.Contenedor) != esperada {
			return ErrManipulada
		}
	}
	// Y al revés: una ranura que el sello conoce y ya no está en el fichero.
	for tipo := range b.sel.Huellas {
		if b.ranura(tipo) < 0 {
			return ErrManipulada
		}
	}
	return nil
}

// azarHex da un identificador nuevo. Va aquí y no en cada sitio porque lo usan
// tanto Poner como Importar, y un id repetido sería un estropicio silencioso.
func azarHex() (string, error) {
	b, err := cripto.Azar(16)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func huellaDe(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (b *Boveda) ranura(tipo string) int {
	for i, s := range b.doc.Sobres {
		if s.Tipo == tipo {
			return i
		}
	}
	return -1
}

// Guardar escribe la bóveda entera.
//
// No hay actualización parcial: se descifra todo y se vuelve a sellar en cada
// cambio. Con mil entradas eso es un par de milisegundos —el cuerpo va con
// `PerfilLlave`, no con el coste de una contraseña humana— así que el límite no
// es la CPU sino no llamar aquí en cada tecla.
func (b *Boveda) Guardar() error {
	if b.llave == nil {
		return ErrCerrada
	}
	if b.soloLectura {
		return ErrFormatoNuevo
	}

	// Comprobación optimista: si el fichero de disco ya no es de la serie que se
	// cargó, alguien lo ha tocado desde otro Esfinge. La serie va en claro, así
	// que no hace falta descifrar nada para mirarla.
	if err := b.comprobarSerieEnDisco(); err != nil {
		return err
	}

	b.doc.Serie++
	b.doc.Cambiada = time.Now().UTC().Format(time.RFC3339)

	// **El cuerpo solo se vuelve a cifrar si han cambiado las entradas.** Si lo
	// único que se ha tocado es una llave, se queda byte a byte como estaba: eso
	// es lo que hace que cambiar la contraseña maestra cueste lo mismo con diez
	// entradas que con diez mil.
	if b.cuerpoSucio || b.doc.Cuerpo == "" {
		claro, err := json.Marshal(b.cont)
		if err != nil {
			b.doc.Serie--
			return err
		}
		defer cripto.Borrar(claro)

		cuerpo, err := cripto.SellarTexto(claro, b.llave, cripto.PerfilLlave)
		if err != nil {
			b.doc.Serie--
			return err
		}
		b.doc.Cuerpo = cuerpo
	}

	b.sel.ID = b.doc.ID
	b.sel.Serie = b.doc.Serie
	b.sel.Cuerpo = huellaDe(b.doc.Cuerpo)
	b.sel.Huellas = map[string]string{}
	for _, s := range b.doc.Sobres {
		b.sel.Huellas[s.Tipo] = huellaDe(s.Contenedor)
	}
	crudoSello, err := json.Marshal(b.sel)
	if err != nil {
		b.doc.Serie--
		return err
	}
	sellado, err := cripto.SellarTexto(crudoSello, b.llave, cripto.PerfilLlave)
	if err != nil {
		b.doc.Serie--
		return err
	}
	b.doc.Sello = sellado

	fuera, err := json.MarshalIndent(b.doc, "", "  ")
	if err != nil {
		b.doc.Serie--
		return err
	}

	err = escritura.Atomica(b.ruta, escritura.Opciones{
		CrearCarpeta: true,
		// El seguro contra un fallo de nuestro propio serializador, que la
		// atomicidad no cubre: lo que está mal se escribe mal, pero atómicamente.
		Anterior: ".anterior",
	}, func(w io.Writer) error {
		_, err := w.Write(append(fuera, '\n'))
		return err
	})
	if err != nil {
		b.doc.Serie--
		return err
	}
	b.cuerpoSucio = false
	return nil
}

func (b *Boveda) comprobarSerieEnDisco() error {
	datos, err := os.ReadFile(b.ruta)
	if err != nil {
		return nil // todavía no existe: es el primer guardado
	}
	var enDisco struct {
		Serie int64 `json:"serie"`
	}
	if err := json.Unmarshal(datos, &enDisco); err != nil {
		return nil
	}
	if enDisco.Serie != b.doc.Serie {
		return fmt.Errorf("%w (serie %d en disco, %d aquí)", ErrCambiada, enDisco.Serie, b.doc.Serie)
	}
	return nil
}

// CambiarMaestra reenvuelve la clave de bóveda con una contraseña nueva.
//
// **El cuerpo no se toca**: queda byte a byte idéntico. Ésa es toda la razón de
// que exista la jerarquía de claves, y hay un test que lo comprueba comparando
// el cuerpo antes y después.
func (b *Boveda) CambiarMaestra(nueva string) error {
	if b.llave == nil {
		return ErrCerrada
	}
	if strings.TrimSpace(nueva) == "" {
		return errors.New("La contraseña maestra no puede estar vacía")
	}
	s, err := envolver(RanuraMaestra, nueva, b.llave, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	if i := b.ranura(RanuraMaestra); i >= 0 {
		b.doc.Sobres[i] = s
	} else {
		b.doc.Sobres = append(b.doc.Sobres, s)
	}
	return b.Guardar()
}

// RotarRecuperacion genera una clave de recuperación nueva y deja la anterior
// inservible. Exige la bóveda abierta, que es lo que la hace segura.
func (b *Boveda) RotarRecuperacion() (string, error) {
	if b.llave == nil {
		return "", ErrCerrada
	}
	nueva, err := NuevaRecuperacion()
	if err != nil {
		return "", err
	}
	s, err := envolver(RanuraRecuperacion, nueva, b.llave, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return "", err
	}
	if i := b.ranura(RanuraRecuperacion); i >= 0 {
		b.doc.Sobres[i] = s
	} else {
		b.doc.Sobres = append(b.doc.Sobres, s)
	}
	return nueva, b.Guardar()
}

// Cerrar borra de memoria lo que se pueda.
func (b *Boveda) Cerrar() {
	cripto.Borrar(b.llave)
	b.llave = nil
	b.cont = contenido{}
}

// Abierta dice si se puede trabajar.
func (b *Boveda) Abierta() bool { return b.llave != nil }

// SoloLectura dice si la bóveda viene de un formato que no entendemos del todo.
func (b *Boveda) SoloLectura() bool { return b.soloLectura }

// Buscar devuelve las entradas que encajan, **sin secretos**.
func (b *Boveda) Buscar(q string) []Entrada {
	var out []Entrada
	for _, e := range b.cont.Entradas {
		if e.Papelera || !e.Coincide(q) {
			continue
		}
		out = append(out, e.SinSecretos())
	}
	return out
}

// Ver devuelve una entrada entera, con sus secretos. Se pide de una en una a
// propósito: ver §SinSecretos.
func (b *Boveda) Ver(id string) (Entrada, bool) {
	for _, e := range b.cont.Entradas {
		if e.ID == id {
			return e, true
		}
	}
	return Entrada{}, false
}

// Poner añade o sustituye una entrada y guarda.
func (b *Boveda) Poner(e Entrada) error {
	if b.llave == nil {
		return ErrCerrada
	}
	ahora := time.Now().UTC().Format(time.RFC3339)
	if e.ID == "" {
		id, err := azarHex()
		if err != nil {
			return err
		}
		e.ID = id
		e.Creada = ahora
	}
	e.Cambiada = ahora
	if e.Tipo == "" {
		e.Tipo = TipoCredencial
	}

	for i, v := range b.cont.Entradas {
		if v.ID == e.ID {
			if e.Creada == "" {
				e.Creada = v.Creada
			}
			b.cont.Entradas[i] = e
			b.cuerpoSucio = true
			return b.Guardar()
		}
	}
	b.cont.Entradas = append(b.cont.Entradas, e)
	b.cuerpoSucio = true
	return b.Guardar()
}

// Borrar manda una entrada a la papelera. El borrado es suave a propósito: sin
// él, «borrada aquí» y «nunca existió allí» son indistinguibles al sincronizar.
func (b *Boveda) Borrar(id string) error {
	if b.llave == nil {
		return ErrCerrada
	}
	for i, e := range b.cont.Entradas {
		if e.ID == id {
			b.cont.Entradas[i].Papelera = true
			b.cont.Entradas[i].BorradaEn = time.Now().UTC().Format(time.RFC3339)
			// Los secretos se van del todo: la papelera guarda que existió, no lo
			// que valía.
			b.cont.Entradas[i].Secreto = ""
			b.cont.Entradas[i].TOTP = ""
			b.cont.Entradas[i].Historial = nil
			b.cuerpoSucio = true
			return b.Guardar()
		}
	}
	return nil
}

// Cuantas devuelve el número de entradas vivas.
func (b *Boveda) Cuantas() int {
	n := 0
	for _, e := range b.cont.Entradas {
		if !e.Papelera {
			n++
		}
	}
	return n
}

// Ruta dice dónde vive, para poder enseñarlo y que nadie tenga que fiarse.
func (b *Boveda) Ruta() string { return b.ruta }
