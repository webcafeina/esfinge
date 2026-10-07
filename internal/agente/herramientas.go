package agente

// El catálogo de herramientas que se le ofrece a un agente (ADR 0054).
//
// # Por qué esto son datos y vive aquí
//
// **`tools/list` tiene que contestarse con Esfinge cerrada.** Claude Code pide la lista
// al abrir la sesión y **se la queda**: si en ese momento no hay nadie escuchando y se
// contesta una lista vacía, el agente se queda sin herramientas **toda la sesión**,
// aunque la ventana se abra dos minutos después.
//
// Por eso el catálogo es una tabla sin bóveda dentro, en el paquete que importa el
// binario pequeño. Si alguien lo mueve a `internal/app` «para ordenar», esto se rompe
// de una forma que **no ve ninguna prueba** salvo la de la tubería con Esfinge cerrada.
// Esa prueba está por esto.
//
// # Los nombres
//
// **Sin `ñ` y sin acentos**: los clientes validan los nombres de herramienta contra
// `^[a-zA-Z0-9_-]{1,128}$`, así que `copiar_contrasena` va sin tilde aunque duela. Y
// con el prefijo `esfinge_` porque los clientes aplanan los nombres de todos los
// servidores juntos.
//
// **Las descripciones están en español** (ADR 0005) y son lo más parecido a código que
// no compila nadie: una mal escrita no pone nada en rojo, simplemente hace que el
// agente haga lo que no debía. Se revisan como se revisa un texto de la interfaz.

// Herramienta es lo que el agente ve de cada una, y las banderas con las que este
// paquete decide qué hacer con ella.
type Herramienta struct {
	Nombre      string  `json:"name"`
	Descripcion string  `json:"description"`
	Esquema     Esquema `json:"inputSchema"`

	// Verbo es lo que se manda por el canal. No sale hacia el agente.
	Verbo string `json:"-"`
	// PideAprobacion: hace falta un sí en la ventana. No sale hacia el agente.
	PideAprobacion bool `json:"-"`
	// DevuelveSecreto: lo que contesta **lo ve el modelo**, así que la válvula no lo
	// cubre nunca. No sale hacia el agente.
	DevuelveSecreto bool `json:"-"`
	// Escribe: cuenta del freno de escrituras. No sale hacia el agente.
	Escribe bool `json:"-"`
}

// Esquema es el JSON Schema de los argumentos, escrito a mano porque son cuatro.
type Esquema struct {
	Tipo         string           `json:"type"`
	Propiedades  map[string]Campo `json:"properties,omitempty"`
	Obligatorios []string         `json:"required,omitempty"`
}

// Campo es una propiedad del esquema.
type Campo struct {
	Tipo        string   `json:"type"`
	Descripcion string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

func objeto(props map[string]Campo, obligatorios ...string) Esquema {
	return Esquema{Tipo: "object", Propiedades: props, Obligatorios: obligatorios}
}

// LasHerramientas es el catálogo. **Cada una con sus tres banderas puestas
// explícitamente**, aunque sean falsas: una bandera que falta es una decisión que nadie
// tomó, y hay una prueba que lo exige.
var LasHerramientas = []Herramienta{
	{
		Nombre: "esfinge_estado",
		Descripcion: "Dice si hay una bóveda de Esfinge en este equipo, si está abierta y en cuál se está " +
			"trabajando. Todas las demás herramientas actúan sobre esa bóveda y ninguna otra, así que " +
			"conviene mirar esto antes de guardar algo. Si está cerrada, la persona tiene que abrirla en " +
			"Esfinge: no se puede abrir desde aquí.",
		Esquema:         objeto(nil),
		Verbo:           QueEstado,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_buscar",
		Descripcion: "Busca entradas en la bóveda abierta y devuelve lo que no es secreto: título, usuario, " +
			"sitios, tipo y si tienen contraseña y segundo factor. **Nunca devuelve contraseñas.** La " +
			"búsqueda no mira los secretos, solo los títulos, usuarios y sitios.",
		Esquema: objeto(map[string]Campo{
			"texto": {Tipo: "string", Descripcion: "Lo que se busca. Vacío devuelve las primeras."},
		}),
		Verbo:           QueBuscar,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_ver",
		Descripcion: "Devuelve una entrada por su identificador, **sin sus secretos**: ni la contraseña, ni la " +
			"semilla del código, ni el número de una tarjeta, ni el texto de una nota. Para usar la " +
			"contraseña está esfinge_copiar_contrasena.",
		Esquema: objeto(map[string]Campo{
			"id": {Tipo: "string", Descripcion: "El identificador que devuelve esfinge_buscar."},
		}, "id"),
		Verbo:           QueVer,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_higiene",
		Descripcion: "Lo que está mal en la bóveda, por identificador y sin secretos: contraseñas reutilizadas " +
			"en varios sitios, entradas duplicadas, credenciales sin segundo factor y tarjetas caducadas. " +
			"Dice cuáles son, nunca cuáles son las contraseñas.",
		Esquema:         objeto(nil),
		Verbo:           QueHigiene,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_copiar_contrasena",
		Descripcion: "Pone la contraseña de una entrada en el portapapeles de este ordenador, para que la " +
			"persona la pegue. **No te la devuelve a ti**: tú recibes solo la confirmación. La primera vez " +
			"que la pidas, Esfinge preguntará en su ventana; si contesta que hace falta aprobarlo, dilo y " +
			"vuelve a pedirlo cuando la persona diga que sí. Se borra sola del portapapeles pasado un rato.",
		Esquema: objeto(map[string]Campo{
			"id": {Tipo: "string", Descripcion: "El identificador que devuelve esfinge_buscar."},
		}, "id"),
		Verbo:          QueCopiarSecreto,
		PideAprobacion: true,
		// **No devuelve el secreto**, que es justo lo que la hace aceptable: lo que
		// vuelve es «copiado», y la contraseña se queda en este equipo.
		DevuelveSecreto: false,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_codigo",
		Descripcion: "Da el código de un solo uso —seis cifras— de una entrada que tenga segundo factor, con " +
			"los segundos que le quedan de vida. **Éste sí te lo devuelve a ti**, porque caduca en medio " +
			"minuto y no sirve sin la contraseña. Hace falta que la persona lo apruebe en la ventana de " +
			"Esfinge **cada vez**, aunque te haya dado permiso para otras cosas.",
		Esquema: objeto(map[string]Campo{
			"id": {Tipo: "string", Descripcion: "El identificador que devuelve esfinge_buscar."},
		}, "id"),
		Verbo:          QueCodigo,
		PideAprobacion: true,
		// **Sí**, y es la única con esta bandera que entrega algo de la bóveda. Lo que
		// esa bandera decide es que **la válvula no la cubre**.
		DevuelveSecreto: true,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_crear",
		Descripcion: "Guarda una entrada nueva en la bóveda abierta. Si pones generar:true, **Esfinge hace la " +
			"contraseña y no te la devuelve** — eso es mejor que inventarla tú, y luego se puede copiar con " +
			"esfinge_copiar_contrasena. Tipos: credencial, nota, tarjeta, identidad, personal, wifi. **Las " +
			"llaves de acceso no se crean aquí**: las emite el sitio.",
		Esquema: objeto(map[string]Campo{
			"tipo":    {Tipo: "string", Descripcion: "credencial si no se dice nada.", Enum: []string{"credencial", "nota", "tarjeta", "identidad", "personal", "wifi"}},
			"titulo":  {Tipo: "string", Descripcion: "Cómo se llama. Obligatorio."},
			"usuario": {Tipo: "string"},
			"secreto": {Tipo: "string", Descripcion: "La contraseña. Mejor no ponerla y usar generar."},
			"notas":   {Tipo: "string"},
			"generar": {Tipo: "boolean", Descripcion: "Que la contraseña la haga Esfinge y no salga de ahí."},
		}, "titulo"),
		Verbo:           QueCrear,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         true,
	},
	{
		Nombre: "esfinge_editar",
		Descripcion: "Cambia campos de una entrada. Lo que no mandes **no se toca**. Cambiar el título, las " +
			"notas o la carpeta va directo; **cambiar la contraseña o el código tiene que aprobarlo la " +
			"persona**, porque eso puede dejar una cuenta sin forma de entrar. La contraseña anterior se " +
			"guarda en el historial de la entrada.",
		Esquema: objeto(map[string]Campo{
			"id":      {Tipo: "string", Descripcion: "El identificador que devuelve esfinge_buscar."},
			"titulo":  {Tipo: "string"},
			"usuario": {Tipo: "string"},
			"secreto": {Tipo: "string", Descripcion: "La contraseña nueva. Esto pide aprobación."},
			"notas":   {Tipo: "string"},
			"carpeta": {Tipo: "string"},
		}, "id"),
		Verbo:           QueEditar,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         true,
	},
	{
		Nombre: "esfinge_borrar",
		Descripcion: "Manda una entrada a la papelera, donde se puede recuperar durante treinta días. **Tiene " +
			"que aprobarlo la persona**: es lo único que quita algo de su vista.",
		Esquema: objeto(map[string]Campo{
			"id": {Tipo: "string", Descripcion: "El identificador que devuelve esfinge_buscar."},
		}, "id"),
		Verbo:           QueBorrar,
		PideAprobacion:  true,
		DevuelveSecreto: false,
		Escribe:         true,
	},
	{
		Nombre: "esfinge_generar",
		Descripcion: "Genera una contraseña nueva al azar. **No toca la bóveda**: no la lee ni guarda nada. " +
			"Para crear una cuenta con una contraseña nueva es mejor pedírselo a esfinge_crear, que la " +
			"genera y la guarda sin que nadie llegue a verla.",
		Esquema: objeto(map[string]Campo{
			"bytes":    {Tipo: "integer", Descripcion: "Cuánta aleatoriedad. 24 si no se dice nada."},
			"alfabeto": {Tipo: "string", Descripcion: "hex, alnum o simbolos. hex si no se dice nada, que es lo que no rompe una URL.", Enum: []string{"hex", "alnum", "simbolos"}},
		}),
		Verbo:          QueGenerar,
		PideAprobacion: false,
		// **Sí devuelve un secreto**, y por eso la bandera está puesta: lo que sale lo
		// ve el modelo. Lo que lo hace aceptable es que **todavía no es de nadie** —no
		// abre ninguna cuenta mientras no se guarde—, y que el camino bueno para crear
		// una cuenta es `esfinge_crear`, que la genera sin enseñarla.
		DevuelveSecreto: true,
		Escribe:         false,
	},
}

// PorNombre busca una herramienta del catálogo.
func PorNombre(nombre string) (Herramienta, bool) {
	for _, h := range LasHerramientas {
		if h.Nombre == nombre {
			return h, true
		}
	}
	return Herramienta{}, false
}
