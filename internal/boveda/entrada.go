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
	// TipoPersonal es un dato personal: un nombre, un correo, un teléfono, una
	// dirección, una fecha de nacimiento (ADR 0047).
	//
	// **Es la primera clase que no guarda un secreto**, y eso cambia lo que
	// Esfinge dice de sí mismo: hasta aquí era «lo que hay dentro no se puede
	// perder ni enseñar», y un teléfono no es eso. Se guarda igual porque es lo
	// que el gestor al que sustituye guardaba, y porque la alternativa realista
	// no era tenerlo fuera: era tenerlo en Dashlane.
	TipoPersonal Tipo = "personal"
	// TipoLlave es una llave de acceso —una passkey— (ADR 0048): la que un sitio
	// acepta en vez de la contraseña.
	//
	// **Es la clase que más pesa de todas**, y por dos razones que no son la
	// criptografía. La primera: lo que guarda **sustituye** a la contraseña en vez
	// de acompañarla, así que perderla no es perder un secreto que se puede
	// restablecer por correo, es perder la cuenta. La segunda: es la única clase
	// cuyo secreto **no se enseña nunca** —no hay nada que copiar ni que leer en
	// voz alta—, así que lo que sale de aquí es una firma y jamás la clave.
	TipoLlave Tipo = "llave"
	// TipoWifi es una red wifi: su nombre, su clave y cómo está protegida (ADR 0049).
	//
	// **Lo que la hace útil no es guardarla, es el código**: la ficha dibuja un QR y
	// un invitado se conecta apuntando el móvil, sin que nadie dicte una contraseña
	// larga en voz alta. Es también la primera clase cuyo secreto **se enseña sin que
	// nadie lo pida** —el código está a la vista en cuanto se abre la ficha—, y eso lo
	// decidió el cliente sabiéndolo: quien fotografíe ese dibujo entra en la red.
	TipoWifi Tipo = "wifi"
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

	// Revision cuenta los cambios de esta entrada: la pone la bóveda al guardar,
	// nunca quien la edita, y sube de uno en uno.
	//
	// Existe para la sincronización (ADR 0038). `Cambiada` no sirve para decidir
	// quién cambió después: tiene resolución de un segundo y depende del reloj de
	// cada equipo. La revisión solo sirve para desempatar cuando dos equipos han
	// tocado lo mismo; qué cambió lo decide comparar con la versión común.
	Revision int64 `json:"revision,omitempty"`

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

	// Dato personal. `NombreCompleto` se comparte con la identidad a propósito:
	// es el mismo dato y buscarlo tiene que encontrar las dos.
	Correo     string `json:"correo,omitempty"`
	Telefono   string `json:"telefono,omitempty"`
	Nacimiento string `json:"nacimiento,omitempty"`

	// La dirección, **en los trozos en que la da un gestor y en que la pide un
	// formulario**. La 2.30.0 la guardaba compuesta en un solo texto y se cambió
	// en la 2.31.0: componerla es de una línea y volver a partirla es adivinar.
	// Para leerla y para copiarla está `Direccion()`.
	Destinatario string `json:"destinatario,omitempty"`
	Calle        string `json:"calle,omitempty"`
	Edificio     string `json:"edificio,omitempty"`
	Piso         string `json:"piso,omitempty"`
	Puerta       string `json:"puerta,omitempty"`
	CodigoPostal string `json:"codigoPostal,omitempty"`
	Ciudad       string `json:"ciudad,omitempty"`
	Provincia    string `json:"provincia,omitempty"`
	Pais         string `json:"pais,omitempty"`

	// Llave de acceso (ADR 0048). Los nombres de WebAuthn se dejan como están
	// —igual que `TOTP`—: son los del protocolo y tienen que poder compararse con
	// lo que dice la especificación sin traducir nada por el camino.
	//
	// **No hay contador de firmas**, y es una decisión, no un olvido: WebAuthn
	// define uno para que un sitio detecte una llave clonada, y una llave
	// sincronizada entre equipos **no puede llevarlo coherente**. Se firma siempre
	// con cero, que es lo que hacen todos los gestores y lo que ningún sitio
	// rechaza. Con ello, esta clase no necesita ninguna regla de fusión propia.
	RPID string `json:"rpId,omitempty"`
	// IDCredencial es lo que el sitio guarda para reconocer esta llave, en
	// base64url. Es lo que la identifica: dos llaves del mismo sitio para la misma
	// persona son dos llaves distintas.
	IDCredencial string `json:"idCredencial,omitempty"`
	// IDUsuario es el identificador opaco que el sitio da a la cuenta, en
	// base64url. No es el usuario que se escribe: eso es `NombreVisible`.
	IDUsuario     string `json:"idUsuario,omitempty"`
	NombreVisible string `json:"nombreVisible,omitempty"`
	// Algoritmo es el de COSE: -7 es ECDSA con P-256 y SHA-256, que es el único
	// que se emite. Se guarda de todos modos porque una llave importada de otro
	// sitio podría traer otro y hay que saber que no se sabe firmarla.
	Algoritmo int `json:"algoritmo,omitempty"`
	// ClavePrivada es la privada en **PKCS#8**, en base64url.
	//
	// La ADR 0048 dijo que sería el escalar de 32 bytes, «porque la parte pública se
	// recalcula», y eso **no se puede hacer**: WebCrypto exige `x` e `y` para importar
	// una P-256 y no expone ninguna forma de multiplicar por el generador, así que
	// `importKey("jwk", {kty:"EC", crv:"P-256", d})` contesta `DataError`. Se vio al
	// implementarlo. PKCS#8 lleva las dos partes dentro y lo entienden los dos lados sin
	// escribir una línea.
	ClavePrivada string `json:"clavePrivada,omitempty"`
	// Confirmada es **cuándo el sitio dijo por primera vez que tiene esta llave**,
	// en RFC3339, o vacío si todavía no lo ha dicho.
	//
	// Existe para distinguir las llaves **huérfanas**: crear guarda en la bóveda
	// antes de entregarle la credencial al sitio —lo contrario dejaría al sitio con
	// una llave que aquí no existe—, así que un registro que falla después deja una
	// llave que no sirve para nada y que **no se distingue de las buenas**. Pasó de
	// verdad el 2026-09-30: cuatro llaves de GitHub y solo una válida.
	//
	// **Lo que la confirma es que el sitio la nombre**, no que se haya firmado con
	// ella: el sitio solo puede pedir por su identificador una credencial que tenga
	// registrada. Firmar no basta —con `allowCredentials` vacío el sitio no dice qué
	// tiene y Esfinge ofrece las suyas, así que una huérfana se firmaría igual y el
	// sitio la rechazaría después—, y esa diferencia es toda la utilidad del campo.
	Confirmada string `json:"confirmada,omitempty"`
	// Usada es **la última vez que se firmó con esta llave**, en RFC3339, o vacío si
	// nunca.
	//
	// Es la otra mitad de `Confirmada`, y van **separadas a propósito** porque no dicen
	// lo mismo: que el sitio la nombre prueba que la tiene registrada; haber firmado con
	// ella solo prueba que alguien la eligió, y en el flujo donde el sitio no nombra
	// ninguna —el «entrar con llave de acceso» de GitHub— se puede firmar con una
	// huérfana y el sitio la rechazará después. Mezclar las dos señales en un campo
	// habría hecho pasar la débil por la fuerte.
	//
	// Hace falta porque **la fuerte casi nunca llega**: el cliente probó la 2.35.0 en su
	// Firefox y su ficha seguía diciendo que el sitio no había pedido la llave, con la
	// llave funcionando. Lo eligió él con las dos señales delante (ADR 0048).
	//
	// **Se actualiza en cada firma**, no solo la primera: lo útil de una fecha de uso es
	// que sea la última. El coste es una escritura en la bóveda por inicio de sesión, y
	// por eso no se reescribe si la fecha no ha cambiado — las fechas tienen resolución
	// de un segundo.
	Usada string `json:"usada,omitempty"`

	// SSID es el nombre de la red, tal cual lo emite el router (ADR 0049).
	//
	// Va aparte del título porque no son lo mismo: el título es cómo se llama la
	// entrada para quien la busca —«Casa», «La oficina»— y el SSID es lo que el móvil
	// tiene que encontrar, con sus mayúsculas y sus espacios exactos. Cuando el fichero
	// importado no trae nombre propio, el título sale de aquí.
	SSID string `json:"ssid,omitempty"`
	// Seguridad es cómo está protegida la red: `wpa`, `wep` o `abierta`.
	//
	// **No se copia de lo que diga el fichero del que salga**, y eso no es desconfianza
	// gratuita: el `wifi.csv` de Dashlane dice `unsecured` en redes que tienen
	// contraseña, y con eso el código saldría marcado como red abierta y el móvil no se
	// conectaría. Lo normaliza `internal/wifi`.
	Seguridad string `json:"seguridad,omitempty"`
	// Oculta dice si la red no anuncia su nombre.
	//
	// Hace falta **para el código**: una red oculta no aparece en la lista del móvil, así
	// que el QR tiene que decirle que la busque. Es el segundo campo de sí o no del
	// formato —el primero fue la papelera— y por eso hubo que arreglar antes el espejo
	// de TypeScript, que escribía todos los booleanos encima de `papelera`.
	Oculta bool `json:"oculta,omitempty"`

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
	// El correo y el teléfono se buscan aunque `vaciarLoSensible` los quite de la
	// lista, y las dos cosas son correctas porque pasan en sitios distintos: esto
	// corre **dentro** de la bóveda, sobre la entrada entera, y lo que se vacía es
	// la copia que sale hacia la ventana. Sin ellos, la única forma de encontrar
	// «Correo electrónico 1» sería acordarse de que se llama así.
	// De una llave de acceso se busca **el sitio y el nombre que enseña**, que es
	// lo único que una persona sabe de ella. Nunca el identificador de credencial:
	// es opaco, nadie lo recuerda, y es lo que la identifica.
	// Y de una red, **el nombre que emite**: es lo que se lee en el móvil y lo que
	// alguien teclea para buscarla, aunque la entrada se llame «La oficina».
	campos := append([]string{e.Titulo, e.Usuario, e.Carpeta, e.NombreCompleto, e.Titular, e.Correo, e.Telefono,
		e.RPID, e.NombreVisible, e.SSID},
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

// Marcas es **lo que se sabe de una entrada sin enseñar nada de ella**: que hay un
// secreto, no cuál es.
type Marcas struct {
	TieneSecreto bool
	TieneCodigo  bool
}

// Marcas se calcula **sobre la entrada entera**, y por eso existe.
//
// `SinSecretos` vacía la semilla del código **sin dejar marca de que la hubiera**, así
// que leer `TOTP != ""` sobre lo que devuelve `Buscar` da siempre falso. Ya pasó una
// vez, al añadir `tieneCodigo` al canal del navegador: **todas las cuentas salían sin
// segundo factor**, y lo cazó una prueba antes de publicar, no la vista.
//
// Llamarla **después** de vaciar no da un error: da `false` en todo. Por eso quien la
// quiera tiene que tener la entrada de dentro, y por eso `BuscarConMarcas` existe.
func (e Entrada) Marcas() Marcas {
	return Marcas{
		// Lo que cuenta como secreto **depende de la clase**: en una credencial es la
		// contraseña, en una nota el texto y en una tarjeta el número. La lista corta
		// de aquí es la de las clases que tienen uno que se pueda usar.
		TieneSecreto: e.Secreto != "" || e.Notas != "" || e.Numero != "" || e.NumeroDocumento != "",
		TieneCodigo:  e.TOTP != "",
	}
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
	// El dato personal (ADR 0047). **El nombre se queda y lo demás no**, y la
	// razón es la misma en los dos sentidos: `tituloDeReserva` saca el título del
	// nombre, así que vaciarlo no protegería nada mientras el título lo repite en
	// la lista y en la papelera. El correo, el teléfono, la dirección y la fecha
	// de nacimiento **no** están en el título, así que vaciarlos sí sirve.
	e.Correo = ""
	e.Telefono = ""
	e.Nacimiento = ""
	e.Destinatario, e.Calle, e.Edificio, e.Piso, e.Puerta = "", "", "", "", ""
	e.CodigoPostal, e.Ciudad, e.Provincia, e.Pais = "", "", "", ""
	// La llave de acceso (ADR 0048). **Solo la clave privada**, y el resto se
	// queda: el sitio y el nombre son lo que la lista tiene que enseñar para que
	// alguien reconozca la llave, y no son secretos. El identificador de
	// credencial tampoco lo es —el sitio ya lo tiene, se lo dio él— pero se va
	// igual: no hace falta en ninguna lista, y lo que no hace falta no viaja.
	e.ClavePrivada = ""
	e.IDCredencial = ""
	e.IDUsuario = ""
}

// Direccion escribe la dirección **en el orden del sobre**, para leerla y para
// copiarla.
//
// El orden importa y no es el de los campos: un gestor los da en el suyo —Dashlane
// pone el país antes que la ciudad— y juntarlos por ahí da «Calle Mayor 1, España,
// Madrid, 28001», que no es una dirección sino una lista de campos.
func (e Entrada) Direccion() string {
	municipio := juntarCon(" ", e.CodigoPostal, e.Ciudad)
	// La provincia solo cuando añade algo: en media España se llama igual que la
	// capital y «Madrid (Madrid)» no informa de nada.
	if p := strings.TrimSpace(e.Provincia); p != "" && !strings.EqualFold(p, strings.TrimSpace(e.Ciudad)) {
		municipio = juntarCon(" ", municipio, "("+p+")")
	}
	return juntarCon("\n",
		strings.TrimSpace(e.Destinatario),
		juntarCon(", ", e.Calle, e.Edificio),
		juntarCon(", ", e.Piso, e.Puerta),
		municipio,
		strings.TrimSpace(e.Pais),
	)
}

// TieneDireccion dice si hay algo que enseñar, que no es lo mismo que que
// `Direccion()` no esté vacía: podría estarlo por tener solo espacios.
func (e Entrada) TieneDireccion() bool { return e.Direccion() != "" }

// ---------------------------------------------------------------------------
// Conservación de campos desconocidos.
//
// La lista de claves conocidas se saca por reflexión de las etiquetas de la
// propia estructura, para que añadir un campo no obligue a acordarse de
// actualizar una lista aparte. Es justo la clase de lista que se queda atrás.

var clavesConocidas = clavesDe(reflect.TypeOf(Entrada{}))

// clavesDe saca los nombres JSON de una estructura. Está aparte porque lo usan
// dos: las entradas y el contenido de la bóveda, que tienen el mismo problema y
// no pueden resolverlo de dos maneras distintas.
func clavesDe(t reflect.Type) map[string]bool {
	m := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		etiqueta := t.Field(i).Tag.Get("json")
		nombre, _, _ := strings.Cut(etiqueta, ",")
		if nombre != "" && nombre != "-" {
			m[nombre] = true
		}
	}
	return m
}

// conservarDesconocidos separa de un JSON las claves que esta versión no
// entiende, para poder devolverlas tal cual al escribir.
func conservarDesconocidos(b []byte, conocidas map[string]bool) (map[string]json.RawMessage, error) {
	var todo map[string]json.RawMessage
	if err := json.Unmarshal(b, &todo); err != nil {
		return nil, err
	}
	for k := range todo {
		if conocidas[k] {
			delete(todo, k)
		}
	}
	if len(todo) == 0 {
		return nil, nil
	}
	return todo, nil
}

// conDesconocidos vuelve a meter lo que se conservó. **Lo conocido manda**: un
// campo que esta versión entiende no se pisa con una copia vieja.
func conDesconocidos(b []byte, extra map[string]json.RawMessage, conocidas map[string]bool) ([]byte, error) {
	if len(extra) == 0 {
		return b, nil
	}
	var todo map[string]json.RawMessage
	if err := json.Unmarshal(b, &todo); err != nil {
		return nil, err
	}
	for k, v := range extra {
		if !conocidas[k] {
			todo[k] = v
		}
	}
	return json.Marshal(todo)
}

type entradaCruda Entrada // sin los métodos, para no entrar en bucle

func (e *Entrada) UnmarshalJSON(b []byte) error {
	var cruda entradaCruda
	if err := json.Unmarshal(b, &cruda); err != nil {
		return err
	}
	*e = Entrada(cruda)

	extra, err := conservarDesconocidos(b, clavesConocidas)
	if err != nil {
		return err
	}
	e.Extra = extra

	// **Lo que escribió la 2.30.0**, que guardaba la dirección compuesta en un solo
	// campo de texto. Se trae a la calle —entera, con sus saltos de línea— en vez de
	// dejarla en `Extra`: ahí se conservaría, pero invisible, que es la clase de
	// pérdida silenciosa que este formato existe para no tener. Partirla en sus
	// trozos sería adivinar, así que eso lo hace quien la mire.
	if crudo, hay := e.Extra["direccion"]; hay {
		var texto string
		if json.Unmarshal(crudo, &texto) == nil && e.Calle == "" {
			e.Calle = texto
		}
		delete(e.Extra, "direccion")
	}
	return nil
}

func (e Entrada) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(entradaCruda(e))
	if err != nil {
		return nil, err
	}
	return conDesconocidos(b, e.Extra, clavesConocidas)
}
