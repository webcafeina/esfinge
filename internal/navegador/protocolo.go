package navegador

// Lo que se puede pedir por el canal, y nada más.
//
// **Esta lista es a la fase 2 lo que `loQuePuedeCruzarElPuente` es a la ventana**,
// y por la misma razón: es una superficie exportada que crece sola si nadie la
// vigila. La diferencia es que aquí al otro lado no hay una ventana nuestra sino
// un navegador, así que la lista importa más.
//
// La regla que la gobierna, heredada de `internal/app/boveda.go`:
//
//	los secretos salen de uno en uno, solo cuando se piden, y siempre con un
//	origen que diga para qué sitio se piden.
const (
	// QueEstado dice si hay bóveda y si está abierta. Nada más: ni cuántas
	// entradas hay, que ya sería contar algo de dentro.
	QueEstado = "estado"
	// QueCuentas devuelve las cuentas que encajan con un origen, **sin secretos**.
	QueCuentas = "cuentas"
	// QueCopiarSecreto pone la contraseña de una entrada en el portapapeles.
	//
	// **Copia Esfinge, y el secreto no cruza el canal.** Es la decisión que hace
	// que en esta entrega **no salga ni un secreto hacia el navegador**, y encima
	// no cuesta nada: el borrado del portapapeles ya existe desde la 2.12.0, así
	// que lo copiado se va solo pasado el plazo de Ajustes. Copiándolo la
	// extensión se quedaría ahí para siempre, que es justo el agujero que aquella
	// versión vino a tapar.
	//
	// Cuando llegue el relleno hará falta un verbo que **sí** devuelva la
	// contraseña, porque para escribirla en un formulario hay que tenerla. Ese día
	// tendrá que justificarse solo; hoy no hace falta y no está.
	QueCopiarSecreto = "copiar-secreto"
	// QueCopiarCodigo hace lo mismo con el código de un solo uso.
	//
	// **Lleva origen, como todo lo demás.** En el primer borrador no lo llevaba, y
	// era un agujero de los que se cuelan por parecer un detalle: los
	// identificadores se pueden enumerar preguntando por cuentas, así que un
	// `codigo(id)` sin origen es el segundo factor entero saliendo por ahí sin que
	// nadie diga para qué sitio.
	QueCopiarCodigo = "copiar-codigo"
	// QueEmparejar pide permiso para hablar con esta bóveda. Lo contesta una
	// persona en la ventana de Esfinge, y devuelve un testigo que la extensión
	// guarda y presenta después.
	//
	// **Se pide una vez en la vida, no una vez por conexión.** El trabajador de
	// una extensión MV3 se muere cada pocos minutos con un puerto nativo abierto,
	// así que el proceso se relanza decenas de veces por sesión: un saludo caro
	// —una derivación, un diálogo— se pagaría todo el rato.
	QueEmparejar = "emparejar"
)

// LoQueSePuedePedir es la lista, en un sitio, para que añadir algo sea una
// decisión y no un efecto de haber escrito un `case` más.
var LoQueSePuedePedir = []string{
	QueEstado, QueEmparejar, QueCuentas, QueCopiarSecreto, QueCopiarCodigo,
}

// VersionDelProtocolo la manda la extensión en cada petición.
//
// Existe desde el primer día porque la extensión y la aplicación **se actualizan
// por caminos distintos y a velocidades distintas**: una por la tienda, con días
// de revisión, y la otra empujando una etiqueta. Van a estar descompasadas casi
// siempre, así que hay que poder decir «esa versión no la entiendo» en vez de
// fallar de formas raras.
const VersionDelProtocolo = 1

// Peticion es lo que llega de la extensión.
type Peticion struct {
	Version int    `json:"version"`
	Que     string `json:"que"`
	// Origen es la dirección de la pestaña **tal como la da el navegador**, no la
	// página. Obligatoria en todo lo que toque una entrada.
	Origen string `json:"origen,omitempty"`
	ID     string `json:"id,omitempty"`
	// Testigo es lo que se le dio a esta extensión al emparejarla.
	Testigo string `json:"testigo,omitempty"`
	// Quien es el nombre que la extensión da de sí misma, para poder enseñarlo al
	// emparejar. **No se cree**: sirve para escribir «Chrome» en un diálogo, no
	// para decidir nada.
	Quien string `json:"quien,omitempty"`
}

// Cuenta es lo que sale hacia el navegador cuando se pregunta qué hay para un
// sitio. **Sin secretos**: ni contraseña, ni semilla, ni notas.
type Cuenta struct {
	ID      string `json:"id"`
	Titulo  string `json:"titulo"`
	Usuario string `json:"usuario"`
}

// Estado es lo poco que se puede saber sin haber abierto nada.
type Estado struct {
	Existe  bool `json:"existe"`
	Abierta bool `json:"abierta"`
}

// Copiado es lo que se sabe después de copiar algo. **No lleva lo copiado.**
type Copiado struct {
	// Portapapeles son los segundos que tardará Esfinge en borrarlo, o cero si el
	// borrado está apagado en Ajustes.
	Portapapeles int `json:"portapapeles"`
	// Quedan son los segundos de vida que le quedan al código de un solo uso.
	// Solo lo lleva `copiar-codigo`, y sirve para no copiar uno que va a caducar
	// antes de que dé tiempo a pegarlo.
	Quedan int `json:"quedan,omitempty"`
}

// Respuesta es lo que vuelve. Siempre lleva `ok`, y cuando es falso lleva un
// motivo que se puede enseñar tal cual: los errores de este proyecto están
// escritos para leerse.
type Respuesta struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	// Motivo es una etiqueta estable para que la extensión pueda decidir sin
	// mirar el texto. Los textos cambian; esto no.
	Motivo string `json:"motivo,omitempty"`

	Estado  *Estado  `json:"estado,omitempty"`
	Cuentas []Cuenta `json:"cuentas,omitempty"`
	Copiado *Copiado `json:"copiado,omitempty"`
	// Testigo solo vuelve al emparejar, y una sola vez.
	Testigo string `json:"testigo,omitempty"`
}

// Los motivos, que son los que la extensión mira para saber qué enseñar.
const (
	MotivoCerrada        = "cerrada"       // hay bóveda, pero está cerrada
	MotivoSinBoveda      = "sin-boveda"    // no hay ninguna
	MotivoSinEmparejar   = "sin-emparejar" // hace falta permitirlo en la ventana
	MotivoOrigenInvalido = "origen"        // no se rellena ahí, y por qué
	MotivoNoEncaja       = "no-encaja"     // esa entrada no es de ese sitio
	MotivoDemasiado      = "demasiado"     // demasiadas preguntas seguidas
	MotivoNoEntiendo     = "no-entiendo"   // versión o petición desconocida
)

func mal(motivo, texto string) Respuesta {
	return Respuesta{OK: false, Motivo: motivo, Error: texto}
}
