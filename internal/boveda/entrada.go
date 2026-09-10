package boveda

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

// Tipo dice qué clase de cosa guarda una entrada.
type Tipo string

const (
	TipoCredencial Tipo = "credencial"
	TipoNota       Tipo = "nota"
	TipoTarjeta    Tipo = "tarjeta"
	TipoIdentidad  Tipo = "identidad"
)

// Antigua es una contraseña que se sustituyó.
//
// Existe por un caso que pasa más de lo que parece: se cambia la contraseña de
// un servicio, el servicio no se entera —o el cambio no llega a guardarse al
// otro lado— y sin la anterior no se entra. Tiene un precio que hay que decir en
// voz alta y está escrito en `docs/seguridad.md`: **un secreto sustituido sigue
// dentro de la bóveda**, así que se puede borrar a mano y hay un tope.
type Antigua struct {
	Secreto string `json:"secreto"`
	Hasta   string `json:"hasta"` // RFC3339, cuándo dejó de valer
}

// maximoHistorial de contraseñas anteriores por entrada. Diez es de sobra para
// el caso real y evita que una entrada crezca sin freno.
const maximoHistorial = 10

// Entrada es una cosa guardada en la bóveda.
//
// Es una estructura ancha con casi todo opcional, en vez de cuatro tipos
// distintos, y es deliberado: los cuatro comparten título, etiquetas, notas y
// fechas, y la mitad del valor de una bóveda está en buscar por encima de todos
// a la vez. Cuatro tipos obligarían a repetir esa búsqueda cuatro veces.
type Entrada struct {
	// ID es inmutable y se pone al crear. Hace falta desde el primer día: sin
	// identidad estable no hay sincronización ni compartir, y ponerlo después
	// sería una migración de datos de verdad.
	ID     string `json:"id"`
	Tipo   Tipo   `json:"tipo"`
	Titulo string `json:"titulo"`

	Notas     string   `json:"notas,omitempty"`
	Etiquetas []string `json:"etiquetas,omitempty"`
	Carpeta   string   `json:"carpeta,omitempty"`

	Creada   string `json:"creada"`
	Cambiada string `json:"cambiada"`

	// Papelera es borrado suave. Barato ahora e imprescindible para sincronizar
	// después: sin él, «borrada aquí» y «nunca existió allí» son indistinguibles.
	Papelera  bool   `json:"papelera,omitempty"`
	BorradaEn string `json:"borradaEn,omitempty"`

	// Credencial.
	Usuario string `json:"usuario,omitempty"`
	Secreto string `json:"secreto,omitempty"`
	// Sitios en plural desde el principio: el autorrelleno va a necesitar varios
	// dominios por entrada, y la misma cuenta se usa en más de un dominio más a
	// menudo de lo que parece.
	Sitios []string `json:"sitios,omitempty"`
	// TOTP es el secreto en base32 tal como lo da el servicio, no el código de
	// seis dígitos. El código se calcula al enseñarlo.
	TOTP      string    `json:"totp,omitempty"`
	Historial []Antigua `json:"historial,omitempty"`

	// Tarjeta.
	Titular      string `json:"titular,omitempty"`
	Numero       string `json:"numero,omitempty"`
	Caduca       string `json:"caduca,omitempty"`
	Verificacion string `json:"verificacion,omitempty"`

	// Identidad.
	NombreCompleto  string `json:"nombreCompleto,omitempty"`
	Documento       string `json:"documento,omitempty"`
	NumeroDocumento string `json:"numeroDocumento,omitempty"`

	// Extra guarda **los campos que esta versión de Esfinge no entiende**.
	//
	// Es lo más subestimado de todo el formato. En cuanto haya dos Esfinges de
	// versiones distintas sobre la misma bóveda —el escenario de la
	// sincronización, y también el de un cliente que no actualiza— la vieja
	// borraría en silencio lo que escribió la nueva. Conservarlos cuesta el
	// Unmarshal/Marshal a medida de abajo; no conservarlos cuesta datos.
	Extra map[string]json.RawMessage `json:"-"`
}

// CambiarSecreto pone una contraseña nueva y guarda la anterior.
func (e *Entrada) CambiarSecreto(nuevo string, ahora time.Time) {
	if e.Secreto != "" && e.Secreto != nuevo {
		e.Historial = append([]Antigua{{
			Secreto: e.Secreto,
			Hasta:   ahora.UTC().Format(time.RFC3339),
		}}, e.Historial...)
		if len(e.Historial) > maximoHistorial {
			e.Historial = e.Historial[:maximoHistorial]
		}
	}
	e.Secreto = nuevo
	e.Cambiada = ahora.UTC().Format(time.RFC3339)
}

// Coincide dice si la entrada encaja con lo que se busca.
//
// Busca por título, usuario, sitios, etiquetas y carpeta. **Nunca por el
// secreto**: teclear en un buscador algo que se compara contra contraseñas es
// una forma silenciosa de averiguarlas a base de probar.
func (e Entrada) Coincide(q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	campos := append([]string{e.Titulo, e.Usuario, e.Carpeta, e.NombreCompleto, e.Titular},
		append(e.Sitios, e.Etiquetas...)...)
	for _, c := range campos {
		if strings.Contains(strings.ToLower(c), q) {
			return true
		}
	}
	return false
}

// SinSecretos devuelve la entrada con lo sensible fuera.
//
// **Es lo que cruza el puente hacia la ventana en la lista.** La contraseña, el
// TOTP y el historial se piden de uno en uno y solo cuando se van a ver: eso
// hace más por que un secreto no acabe en un volcado de memoria que todo el
// borrado de búferes junto, porque en cuanto algo cruza a la interfaz vive en el
// montón del webview, fuera de nuestro alcance.
func (e Entrada) SinSecretos() Entrada {
	e.vaciarLoSensible()
	return e
}

// vaciarLoSensible quita de la entrada todo lo que hay que proteger.
//
// **Está en un solo sitio a propósito**, y lo usan dos caminos que parecen
// distintos y no lo son: lo que viaja en la lista hacia la ventana y lo que
// queda en la papelera al borrar. Con dos listas separadas ya pasó lo que tenía
// que pasar: la de borrar solo quitaba la contraseña, el TOTP y el historial, y
// **una nota segura borrada se quedaba entera dentro del fichero**, igual que el
// número de una tarjeta. Añadir un campo sensible y acordarse de dos sitios es
// una defensa que dura hasta la siguiente prisa.
func (e *Entrada) vaciarLoSensible() {
	e.Secreto = ""
	e.TOTP = ""
	e.Historial = nil
	e.Verificacion = ""
	e.Numero = ""
	e.NumeroDocumento = ""
	e.Notas = ""
}

// ---------------------------------------------------------------------------
// Conservación de campos desconocidos.
//
// La lista de claves conocidas se saca por reflexión de las etiquetas de la
// propia estructura, para que añadir un campo no obligue a acordarse de
// actualizar una lista aparte. Es justo la clase de lista que se queda atrás.

var clavesConocidas = func() map[string]bool {
	m := map[string]bool{}
	t := reflect.TypeOf(Entrada{})
	for i := 0; i < t.NumField(); i++ {
		etiqueta := t.Field(i).Tag.Get("json")
		nombre, _, _ := strings.Cut(etiqueta, ",")
		if nombre != "" && nombre != "-" {
			m[nombre] = true
		}
	}
	return m
}()

type entradaCruda Entrada // sin los métodos, para no entrar en bucle

func (e *Entrada) UnmarshalJSON(b []byte) error {
	var cruda entradaCruda
	if err := json.Unmarshal(b, &cruda); err != nil {
		return err
	}
	*e = Entrada(cruda)

	var todo map[string]json.RawMessage
	if err := json.Unmarshal(b, &todo); err != nil {
		return err
	}
	for k := range todo {
		if clavesConocidas[k] {
			delete(todo, k)
		}
	}
	if len(todo) > 0 {
		e.Extra = todo
	}
	return nil
}

func (e Entrada) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(entradaCruda(e))
	if err != nil {
		return nil, err
	}
	if len(e.Extra) == 0 {
		return b, nil
	}

	var todo map[string]json.RawMessage
	if err := json.Unmarshal(b, &todo); err != nil {
		return nil, err
	}
	for k, v := range e.Extra {
		// Lo conocido manda: un campo que esta versión entiende no se pisa con
		// una copia vieja que venga de Extra.
		if !clavesConocidas[k] {
			todo[k] = v
		}
	}
	return json.Marshal(todo)
}
