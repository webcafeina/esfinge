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
	"reflect"
	"sort"
	"strings"
	"sync"
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
	errSinFichero   = errors.New("Esta bóveda todavía no tiene fichero en este equipo")
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
	// Sobres son las huellas del **sobre entero**, no solo de su contenedor
	// (revisión del 2026-09-23).
	//
	// `Huellas` deja fuera `creado` y `codificacion`, y `creado` no es adorno: es
	// lo que decide qué ranura gana al fundir sin base. Un servidor que quisiera
	// hacer daño podía envejecer o rejuvenecer una ranura sin tocar su contenedor,
	// y colarle a un equipo rezagado una contraseña maestra vieja.
	//
	// Va en un campo aparte y no dentro de `Huellas` para que **un fichero de
	// antes se siga abriendo**: si no está, se comprueba lo de siempre. Nadie puede
	// quitarlo para esquivarlo, porque el sello viaja cifrado con la clave de
	// bóveda: sin ella no se puede reescribir.
	Sobres map[string]string `json:"sobres,omitempty"`
	// Sincro es la versión del servidor que tiene o va a tener este documento.
	//
	// Va aquí, **dentro de lo cifrado con la clave de bóveda**, porque es lo que
	// impide que un servidor sirva una versión vieja haciéndola pasar por nueva:
	// la versión que dice la respuesta tiene que coincidir con la de dentro, y la
	// de dentro no la puede escribir nadie que no tenga la clave. Una bóveda que no
	// se ha sincronizado nunca la lleva a cero.
	Sincro int64 `json:"sincro,omitempty"`
	// Cuerpo es la huella del cuerpo, y es lo que ata las dos piezas.
	//
	// Al separar el sello del cuerpo, revertir el cuerpo a uno viejo dejó de
	// detectarse: nada los relacionaba. Con la huella aquí vuelve a detectarse, y
	// **sin renunciar a lo que se había ganado**: si solo cambia una llave, el
	// cuerpo no cambia, así que su huella tampoco y el cuerpo sigue intacto.
	Cuerpo string `json:"cuerpo"`
}

// contenido es lo que va cifrado dentro del cuerpo: hoy, solo las entradas.
//
// **Y conserva lo que no entiende, igual que una entrada.** Esto faltaba, y era
// una trampa con fecha: `Entrada` guarda en `Extra` los campos que una versión no
// conoce —para que una Esfinge vieja no borre en silencio lo que escribió una
// nueva— y el envoltorio de aquí **no hacía nada de eso**. En cuanto alguien
// añadiera una sección nueva al lado de `entradas` —los permisos del navegador de
// la fase 2 son el primer candidato—, cualquier Esfinge anterior que abriera la
// bóveda la habría tirado al guardar.
//
// Se arregla ahora porque después sería una migración de datos: mientras no haya
// nada que conservar no cuesta nada, y en cuanto lo haya, ya es tarde.
type contenido struct {
	Entradas []Entrada `json:"entradas"`

	// SitiosExcluidos son los dominios en los que la extensión del navegador **no
	// ofrece guardar** una contraseña (ADR 0032).
	//
	// **Viven aquí, dentro del cuerpo que se cifra entero, y no en el navegador ni en
	// las preferencias**, y la razón es la misma por la que se cifraron los iconos
	// (ADR 0024): una lista de sitios dice mucho de alguien. En el perfil del
	// navegador estaría en claro, y las preferencias son un JSON sin cifrar que
	// además cruza el puente a la ventana. Una Esfinge anterior a la 2.21.0 no conoce
	// esta sección y la conserva igual, por `Extra`.
	SitiosExcluidos []string `json:"sitiosExcluidos,omitempty"`

	// Lapidas son las entradas borradas del todo: identificador → cuándo.
	//
	// **Sin ellas no se puede sincronizar un borrado.** Una entrada que falta en
	// un lado puede ser una que se borró aquí o una que allí todavía no ha llegado,
	// y confundirlas es resucitar lo borrado o borrar lo nuevo. Solo llevan el
	// identificador y la fecha —nada del contenido—, y duran más que la papelera
	// (PlazoLapidas), para que un equipo que pase semanas sin conectar se entere.
	Lapidas map[string]string `json:"lapidas,omitempty"`

	// Identidad es la de esta bóveda para compartir copias (ADR 0043): una semilla
	// de la que salen las llaves de cifrado y de firma. **Se crea una vez y no
	// cambia**, y una versión que no la conozca la conserva por `Extra`.
	Identidad *identidad `json:"identidad,omitempty"`

	// Envios son las copias que esperan a que quien las recibe tenga cuenta
	// (ADR 0043, entrega B3). Ver `pendiente.go`: son notas, no secretos.
	Envios []Pendiente `json:"envios,omitempty"`

	// Extra son las secciones que esta versión no conoce. Ver Entrada.Extra.
	Extra map[string]json.RawMessage `json:"-"`
}

var clavesDelContenido = clavesDe(reflect.TypeOf(contenido{}))

type contenidoCrudo contenido // sin los métodos, para no entrar en bucle

func (c *contenido) UnmarshalJSON(b []byte) error {
	var crudo contenidoCrudo
	if err := json.Unmarshal(b, &crudo); err != nil {
		return err
	}
	*c = contenido(crudo)

	extra, err := conservarDesconocidos(b, clavesDelContenido)
	if err != nil {
		return err
	}
	c.Extra = extra
	return nil
}

func (c contenido) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(contenidoCrudo(c))
	if err != nil {
		return nil, err
	}
	return conDesconocidos(b, c.Extra, clavesDelContenido)
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
	// mu guarda todo lo de abajo.
	//
	// **No estaba, y hacía falta desde antes de que se notara.** Ya había dos
	// gorrutinas tocando esto: la que atiende a la ventana y el tic del bloqueo por
	// inactividad, que llama a `Cerrar()` —y `Cerrar` pone `llave` a nil y vacía el
	// contenido justo mientras `Guardar` puede estar serializándolo—. Era una
	// ventana estrecha que nadie había pillado; con la descarga de iconos
	// trabajando de fondo se vuelve ancha y reproducible.
	mu sync.Mutex

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
	// alGuardar se llama tras cada guardado que sale bien, **en su propia
	// gorrutina**: se guarda con el cerrojo cogido, y quien escucha —la
	// sincronización— no puede esperar a que se suelte ni tomarlo.
	alGuardar func()
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
	return crear(ruta, maestra)
}

// CrearEnMemoria hace una bóveda nueva **sin escribirla**: la que se crea al dar
// de alta una cuenta, que solo se guarda si el servidor dice que sí. Si se
// guardara antes y el alta fallara, quedaría una bóveda cuya clave de
// recuperación no ha visto nadie. Se le da fichero con GuardarEn.
func CrearEnMemoria(maestra string) (*Boveda, string, error) {
	return crear("", maestra)
}

func crear(ruta, maestra string) (*Boveda, string, error) {
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

	if ruta == "" {
		return b, recuperacion, nil
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

// AbrirBytes es lo mismo sin leer el disco. Existe para poder probarlo y para
// abrir una bóveda que venga de otro sitio.
//
// **Puede escribir en `ruta`**: si al abrir hay papelera o lápidas caducadas, se
// guardan ya purgadas. Para mirar una bóveda que no es la de este equipo sin
// tocar nada, AbrirEnMemoria.
func AbrirBytes(ruta string, datos []byte, llaveTecleada string) (*Boveda, error) {
	return abrir(ruta, datos, llaveTecleada, true)
}

// AbrirEnMemoria abre una bóveda que no tiene fichero en este equipo —la que baja
// del servidor en un equipo nuevo— **sin escribir nada en ninguna parte**. La que
// devuelve no se puede guardar hasta que se le dé una ruta con GuardarEn.
func AbrirEnMemoria(datos []byte, llaveTecleada string) (*Boveda, error) {
	return abrir("", datos, llaveTecleada, false)
}

func abrir(ruta string, datos []byte, llaveTecleada string, purgar bool) (*Boveda, error) {
	doc, err := leerDocumento(datos)
	if err != nil {
		return nil, err
	}

	// Si parece una clave de recuperación, se normaliza: así se acepta lo que la
	// gente escribe de verdad —minúsculas, sin guiones, con «O» donde va un cero—
	// y la suma de control queda comprobada antes de derivar nada.
	candidatas := []string{llaveTecleada}
	norm, errRecuperacion := Normalizar(llaveTecleada)
	if errRecuperacion == nil {
		candidatas = append([]string{norm}, candidatas...)
	}

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

	return conLlave(ruta, doc, llave, purgar)
}

// conLlave arma la bóveda a partir de la clave que ya se ha sacado de un sobre,
// y purga lo caducado si toca. Lo comparten abrir —la contraseña tecleada— y
// `AbrirConElSistema`, que prueba una sola ranura.
func conLlave(ruta string, doc documento, llave []byte, purgar bool) (*Boveda, error) {
	// Que la llave abriera un sobre y esto no se deje abrir significa que el
	// fichero está mezclado, no que la clave esté mal.
	sel, cont, err := desempaquetar(doc, llave)
	if err != nil {
		return nil, err
	}
	b := &Boveda{ruta: ruta, doc: doc, sel: sel, cont: cont, llave: llave, soloLectura: doc.Formato < Formato}

	if !purgar {
		return b, nil
	}
	// **La papelera se vacía sola al abrir**, y aquí y no con un reloj a
	// propósito: una bóveda cerrada no ejecuta nada, así que un reloj solo
	// contaría mientras la aplicación estuviera puesta y el plazo dependería de
	// cuánto la usa cada uno. Al abrir se sabe qué día es y se puede decidir de
	// una vez. Y lo mismo las lápidas que ya han cumplido su plazo.
	purgados := b.purgarPendientes(ahora().Add(-PlazoPendientes))
	if b.purgarPapelera(ahora().Add(-PlazoPapelera))+b.purgarLapidas(ahora().Add(-PlazoLapidas)) > 0 || purgados {
		b.cuerpoSucio = true
		if err := b.guardar(); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// leerDocumento lee el JSON de fuera y comprueba que es una bóveda que se entiende.
func leerDocumento(datos []byte) (documento, error) {
	var doc documento
	if err := json.Unmarshal(datos, &doc); err != nil || doc.Esfinge != marca {
		return documento{}, ErrNoEsBoveda
	}
	if doc.Formato > Formato {
		return documento{}, ErrFormatoNuevo
	}
	return doc, nil
}

// desempaquetar abre el sello y el cuerpo con la clave de bóveda y comprueba que
// cuadran con lo de fuera. No escribe nada.
func desempaquetar(doc documento, llave []byte) (sello, contenido, error) {
	var sel sello
	var cont contenido
	crudoSello, err := cripto.AbrirTexto(doc.Sello, llave)
	if err != nil {
		return sel, cont, ErrManipulada
	}
	if err := json.Unmarshal(crudoSello, &sel); err != nil {
		return sel, cont, ErrManipulada
	}
	claro, err := cripto.AbrirTexto(doc.Cuerpo, llave)
	if err != nil {
		return sel, cont, ErrManipulada
	}
	defer cripto.Borrar(claro)
	if err := json.Unmarshal(claro, &cont); err != nil {
		return sel, cont, ErrManipulada
	}
	if err := coherente(doc, sel); err != nil {
		return sel, cont, err
	}
	return sel, cont, nil
}

// comprobarCoherencia compara lo de dentro con lo de fuera.
func (b *Boveda) comprobarCoherencia() error { return coherente(b.doc, b.sel) }

// coherente compara lo de dentro con lo de fuera.
//
// El JSON exterior no va autenticado, así que quien pueda escribir el fichero
// puede quitar la ranura de recuperación o revertir el cuerpo a uno viejo. No
// puede leerlo ni fabricar uno que abra, pero sí estropearlo sin que se note. Lo
// que hay aquí no lo impide: lo **detecta**, y es lo mismo que protege lo que
// baja del servidor, que tampoco puede fabricar una ranura.
func coherente(doc documento, sel sello) error {
	if sel.ID != doc.ID {
		return ErrManipulada
	}
	if sel.Serie != doc.Serie {
		return ErrManipulada
	}
	if sel.Cuerpo != huellaDe(doc.Cuerpo) {
		return ErrManipulada
	}
	tiene := map[string]bool{}
	for _, s := range doc.Sobres {
		tiene[s.Tipo] = true
		esperada, hay := sel.Huellas[s.Tipo]
		if !hay {
			// Una ranura que el sello no conoce: sobra o es de una versión que
			// añade tipos nuevos. No es motivo para no abrir.
			continue
		}
		if huellaDe(s.Contenedor) != esperada {
			return ErrManipulada
		}
		// Y si el sello es de los que cubren el sobre entero, también la fecha y
		// la codificación. Un fichero de antes no lo trae, y eso no es motivo para
		// no abrirlo.
		if h, hay := sel.Sobres[s.Tipo]; hay && h != huellaDeSobre(s) {
			return ErrManipulada
		}
	}
	// Y al revés: una ranura que el sello conoce y ya no está en el fichero.
	for tipo := range sel.Huellas {
		if !tiene[tipo] {
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

// huellaDeSobre cubre el sobre entero. Los campos van separados por saltos de
// línea, que no pueden aparecer dentro de ninguno de ellos —un tipo, una fecha
// RFC 3339, una palabra y un contenedor en base64—, así que dos sobres distintos
// no pueden dar la misma cadena. Es a propósito **una cadena y no JSON**: la
// tiene que escribir igual la extensión, en TypeScript (docs/formato-boveda.md).
func huellaDeSobre(s sobre) string {
	return huellaDe(strings.Join([]string{s.Tipo, s.Creado, s.Codificacion, s.Contenedor}, "\n"))
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
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.guardar()
}

// guardar es lo mismo con el cerrojo ya cogido, para los métodos de aquí que
// terminan guardando sin soltarlo.
func (b *Boveda) guardar() error {
	if b.llave == nil {
		return ErrCerrada
	}
	if b.soloLectura {
		return ErrFormatoNuevo
	}
	if b.ruta == "" {
		return errSinFichero
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
	b.sel.Sobres = map[string]string{}
	for _, s := range b.doc.Sobres {
		b.sel.Huellas[s.Tipo] = huellaDe(s.Contenedor)
		b.sel.Sobres[s.Tipo] = huellaDeSobre(s)
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
	if b.alGuardar != nil {
		go b.alGuardar()
	}
	return nil
}

// AlGuardar pide que se avise tras cada guardado. Vale para cualquiera que
// escriba: la ventana, el navegador, el importador y la propia fusión.
func (b *Boveda) AlGuardar(f func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.alGuardar = f
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
	b.mu.Lock()
	defer b.mu.Unlock()
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
	return b.guardar()
}

// RotarRecuperacion genera una clave de recuperación nueva y deja la anterior
// inservible. Exige la bóveda abierta, que es lo que la hace segura.
func (b *Boveda) RotarRecuperacion() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
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
	return nueva, b.guardar()
}

// Cerrar borra de memoria lo que se pueda.
func (b *Boveda) Cerrar() {
	b.mu.Lock()
	defer b.mu.Unlock()
	cripto.Borrar(b.llave)
	b.llave = nil
	b.cont = contenido{}
}

// Abierta dice si se puede trabajar.
func (b *Boveda) Abierta() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.llave != nil
}

// SoloLectura dice si la bóveda viene de un formato que no entendemos del todo.
func (b *Boveda) SoloLectura() bool { return b.soloLectura }

// Buscar devuelve las entradas que encajan, **sin secretos**.
func (b *Boveda) Buscar(q string) []Entrada {
	b.mu.Lock()
	defer b.mu.Unlock()
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
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, e := range b.cont.Entradas {
		if e.ID == id {
			return e, true
		}
	}
	return Entrada{}, false
}

// Poner añade o sustituye una entrada y guarda.
func (b *Boveda) Poner(e Entrada) error {
	b.mu.Lock()
	defer b.mu.Unlock()
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
	// La revisión la pone la bóveda, no quien edita: lo que llegue de la ventana o
	// del navegador puede ser una copia vieja de la entrada.
	e.Revision = 1

	for i, v := range b.cont.Entradas {
		if v.ID == e.ID {
			if e.Creada == "" {
				e.Creada = v.Creada
			}
			e.Revision = v.Revision + 1
			b.cont.Entradas[i] = e
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	b.cont.Entradas = append(b.cont.Entradas, e)
	b.cuerpoSucio = true
	return b.guardar()
}

// PlazoPapelera es lo que sobrevive una entrada borrada.
//
// Treinta días es lo que usa todo el mundo, y por una razón que no es la
// costumbre: es lo que tarda alguien en darse cuenta de que borró lo que no era
// —normalmente cuando va a entrar en el sitio— sin que la papelera se convierta
// en un almacén paralelo de contraseñas que nadie mira.
const PlazoPapelera = 30 * 24 * time.Hour

// ahora es una variable para poder parar el reloj en las pruebas.
//
// Es la misma costura que `azar` en internal/cripto, y hace falta por lo mismo:
// sin ella, comprobar que la papelera se vacía a los treinta días exige esperar
// treinta días o fabricar a mano un fichero con fechas viejas dentro.
var ahora = time.Now

// Borrar manda una entrada a la papelera.
//
// **Lo borrado se guarda entero**, con su contraseña, hasta que la papelera se
// vacíe (ADR 0026). Es lo contrario de lo que hacía esto por la mañana, y el
// cambio es deliberado: mientras no había forma de vaciarla, guardar el secreto
// era dejarlo dentro del fichero para siempre; con una papelera que se vacía —a
// mano o sola a los treinta días— lo que se compra a cambio es que **un clic mal
// dado deje de perder una contraseña para siempre**.
//
// El borrado sigue siendo suave por lo de siempre, además: sin rastro, «borrada
// aquí» y «nunca existió allí» son indistinguibles al sincronizar.
func (b *Boveda) Borrar(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	for i, e := range b.cont.Entradas {
		if e.ID == id && !e.Papelera {
			cuando := ahora().UTC().Format(time.RFC3339)
			b.cont.Entradas[i].Papelera = true
			b.cont.Entradas[i].BorradaEn = cuando
			// **Mandar a la papelera y sacar de ella son cambios, y la fecha lo
			// tiene que decir** (revisión del 2026-09-23). Una entrada sobrevive a
			// la lápida de otro equipo si se cambió después de que se borrara
			// (`p.Cambiada >= lapidaDelOtro`): sin tocar la fecha, restaurar una
			// entrada aquí perdía contra la purga de allí y se volvía a ir.
			b.cont.Entradas[i].Cambiada = cuando
			b.cont.Entradas[i].Revision++
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	return nil
}

// Papelera devuelve lo borrado que todavía se puede recuperar, **sin secretos**
// y con lo último borrado arriba.
//
// Sin secretos como cualquier otra lista: que una entrada esté en la papelera no
// la hace menos secreta, y quien quiera verla la restaura primero.
func (b *Boveda) Papelera() []Entrada {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []Entrada
	for _, e := range b.cont.Entradas {
		if e.Papelera {
			out = append(out, e.SinSecretos())
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].BorradaEn > out[j].BorradaEn // RFC3339 ordena como texto
	})
	return out
}

// Restaurar saca una entrada de la papelera y la devuelve a la lista, entera.
func (b *Boveda) Restaurar(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	for i, e := range b.cont.Entradas {
		if e.ID == id && e.Papelera {
			b.cont.Entradas[i].Papelera = false
			b.cont.Entradas[i].BorradaEn = ""
			b.cont.Entradas[i].Cambiada = ahora().UTC().Format(time.RFC3339)
			b.cont.Entradas[i].Revision++
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	return errors.New("Esa entrada ya no está en la papelera")
}

// BorrarDelTodo quita una entrada de la papelera y de la bóveda. **No hay vuelta
// atrás**, y esta vez de verdad.
func (b *Boveda) BorrarDelTodo(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	for i, e := range b.cont.Entradas {
		// **Solo desde la papelera.** Una entrada viva se borra en dos pasos, y
		// saltárselos por tener el identificador a mano sería quitarle el sentido
		// al primero.
		if e.ID == id && e.Papelera {
			b.cont.Entradas = append(b.cont.Entradas[:i], b.cont.Entradas[i+1:]...)
			b.enterrar(id, ahora())
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	return errors.New("Esa entrada no está en la papelera")
}

// VaciarPapelera se lleva todo lo borrado y dice cuánto era.
func (b *Boveda) VaciarPapelera() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return 0, ErrCerrada
	}
	// Con un plazo en el futuro entra todo, sea de cuando sea.
	cuantas := b.purgarPapelera(ahora().Add(time.Hour))
	if cuantas == 0 {
		return 0, nil // no hay nada que guardar, y guardar de más reescribe la bóveda
	}
	b.cuerpoSucio = true
	return cuantas, b.guardar()
}

// purgarPapelera quita lo borrado antes de esa fecha y devuelve cuánto quitó.
// **No guarda**: quien llame decide, porque uno de los dos sitios que la usan
// está a mitad de abrir el fichero.
//
// Cada entrada que se va deja su lápida.
func (b *Boveda) purgarPapelera(limite time.Time) int {
	corte := limite.UTC().Format(time.RFC3339)
	vivas := b.cont.Entradas[:0]
	quitadas := 0
	for _, e := range b.cont.Entradas {
		// Una entrada en la papelera **sin fecha** no se toca. Solo puede venir de
		// una versión que no la escribía, y tirar datos de alguien por no saber
		// cuándo los borró es exactamente lo que no hay que hacer.
		if e.Papelera && e.BorradaEn != "" && e.BorradaEn < corte {
			quitadas++
			b.enterrar(e.ID, ahora())
			continue
		}
		vivas = append(vivas, e)
	}
	b.cont.Entradas = vivas
	return quitadas
}

// PlazoLapidas es lo que dura el rastro de una entrada borrada del todo.
//
// Seis meses, bastante más que la papelera: es el tiempo que un equipo puede
// pasar sin conectarse y aun así enterarse de que algo se borró, en vez de
// devolverlo a la vida al sincronizar. Pasado ese plazo, un equipo que vuelva
// con la entrada la subirá como si fuera nueva: es el coste, y está dicho en la
// ADR 0038.
const PlazoLapidas = 180 * 24 * time.Hour

// enterrar apunta que una entrada se ha borrado del todo.
func (b *Boveda) enterrar(id string, cuando time.Time) {
	if b.cont.Lapidas == nil {
		b.cont.Lapidas = map[string]string{}
	}
	b.cont.Lapidas[id] = cuando.UTC().Format(time.RFC3339)
}

// purgarLapidas quita las lápidas anteriores a esa fecha y dice cuántas.
func (b *Boveda) purgarLapidas(limite time.Time) int {
	corte := limite.UTC().Format(time.RFC3339)
	n := 0
	for id, cuando := range b.cont.Lapidas {
		if cuando < corte {
			delete(b.cont.Lapidas, id)
			n++
		}
	}
	if len(b.cont.Lapidas) == 0 {
		b.cont.Lapidas = nil
	}
	return n
}

// EnLaPapelera cuenta lo borrado que todavía se puede recuperar.
func (b *Boveda) EnLaPapelera() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, e := range b.cont.Entradas {
		if e.Papelera {
			n++
		}
	}
	return n
}

// Cuantas devuelve el número de entradas vivas.
func (b *Boveda) Cuantas() int {
	b.mu.Lock()
	defer b.mu.Unlock()
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

// ------------------------------------------- los sitios en los que no se ofrece

// Excluir apunta un dominio en el que la extensión no ofrecerá guardar. Repetirlo
// no hace nada.
func (b *Boveda) Excluir(dominio string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	dominio = strings.ToLower(strings.TrimSpace(dominio))
	if dominio == "" {
		return errors.New("Hace falta un sitio que excluir")
	}
	for _, d := range b.cont.SitiosExcluidos {
		if d == dominio {
			return nil
		}
	}
	b.cont.SitiosExcluidos = append(b.cont.SitiosExcluidos, dominio)
	sort.Strings(b.cont.SitiosExcluidos)
	b.cuerpoSucio = true
	return b.guardar()
}

// QuitarExclusion vuelve a dejar que se ofrezca guardar en ese dominio.
func (b *Boveda) QuitarExclusion(dominio string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	dominio = strings.ToLower(strings.TrimSpace(dominio))
	var quedan []string
	for _, d := range b.cont.SitiosExcluidos {
		if d != dominio {
			quedan = append(quedan, d)
		}
	}
	if len(quedan) == len(b.cont.SitiosExcluidos) {
		return errors.New("Ese sitio no estaba excluido")
	}
	b.cont.SitiosExcluidos = quedan
	b.cuerpoSucio = true
	return b.guardar()
}

// Excluido dice si en ese dominio no se ofrece guardar.
func (b *Boveda) Excluido(dominio string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	dominio = strings.ToLower(strings.TrimSpace(dominio))
	for _, d := range b.cont.SitiosExcluidos {
		if d == dominio {
			return true
		}
	}
	return false
}

// Excluidos es la lista, en orden, para enseñarla en Ajustes.
func (b *Boveda) Excluidos() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.cont.SitiosExcluidos...)
}
