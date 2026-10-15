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
	// Items es de qué son los elementos cuando `Tipo` es `array`. Sin esto el
	// esquema no dice de qué va la lista y el cliente no manda nada.
	Items *Campo `json:"items,omitempty"`
}

// lista es un campo de texto repetido, que es la única forma de lista que hay aquí.
func lista(desc string) Campo {
	return Campo{Tipo: "array", Descripcion: desc, Items: &Campo{Tipo: "string"}}
}

func objeto(props map[string]Campo, obligatorios ...string) Esquema {
	return Esquema{Tipo: "object", Propiedades: props, Obligatorios: obligatorios}
}

// campoDeBoveda es **en cuál se quiere la acción** (ADR 0055).
//
// Lo lleva todo lo que toca la bóveda, y eso es el diseño entero: la bóveda es un dato
// de cada acción y no un modo. Un modo hay que recordarlo entre llamadas, y lo que un
// agente no recuerda acaba escrito donde no iba.
var campoDeBoveda = Campo{
	Tipo: "string",
	Descripcion: "En qué bóveda quieres que se haga, por su nombre —`esfinge_bovedas` las lista—. " +
		"Vacío es la que esté abierta. Esfinge conmuta sola y devuelve la ventana a donde estaba.",
}

// enLaQueEsteAbierta es lo que llevan las herramientas que escriben.
//
// **Decirlo aquí es la diferencia entre esperarlo y pedirlo.** A un agente al que le
// dicen «guarda esto en el proyecto de tal cliente» no le falta nada para hacerlo mal:
// escribe en la bóveda que esté abierta, que puede ser otra, y nadie se entera hasta que
// alguien busca esa cuenta donde no está.
//
// **Y se mira siempre, no solo cuando alguien nombra otra bóveda**, que es como estaba
// escrito al principio y por eso no servía: el caso que de verdad duele es justo el que
// **no** la nombra —tienes abierto un proyecto, pides guardar algo sin pensar en dónde, y
// acaba en la bóveda de un cliente—. Pedirlo solo al nombrarla deja al agente contándolo
// **después**, que es cuando ya está escrito y sincronizado. Lo señaló el agente del
// cliente el 2026-10-13, probando exactamente ese caso.
//
// Lo que **no** se hace, y es decisión: pedir confirmación en la ventana al escribir
// fuera de la personal. Crear y editar van directos desde la ADR 0032, y con un proyecto
// abierto a propósito —que es el caso normal— preguntar cada vez es ruido. Lo que falta
// no es una puerta más: es que el agente mire antes y lo diga.
//
// **Y desde la ADR 0055 la bóveda se puede nombrar**, lo que cambia la mitad de esta
// frase y no la otra. Lo que decía al final —«si te nombra otra bóveda, pídele que la
// abra él»— era justo lo molesto: «es muy molesto tener que ir cambiando de bóveda y que
// diga todo el rato que no puede hacer algo» (el cliente, 2026-10-14). Ahora se nombra en
// `boveda` y ya está. Lo que **no** cambia es el caso sin nombrar, que es el que duele:
// ahí se mira y se pregunta, igual que antes.
const enLaQueEsteAbierta = "**Di en `boveda` en cuál quieres que se haga**, con su nombre, y se hará " +
	"ahí: no hace falta que nadie cambie de bóveda a mano. `esfinge_bovedas` las lista. " +
	"**Y si no te han dicho dónde**, mira `esfinge_estado` —antes de escribir, siempre—, di en voz alta " +
	"en qué bóveda vas a guardar, y si es la de un proyecto y quien te lo pide no lo ha nombrado, " +
	"pregúntale antes de hacerlo. "

// LasHerramientas es el catálogo. **Cada una con sus tres banderas puestas
// explícitamente**, aunque sean falsas: una bandera que falta es una decisión que nadie
// tomó, y hay una prueba que lo exige.
var LasHerramientas = []Herramienta{
	{
		Nombre: "esfinge_estado",
		Descripcion: "Dice si hay una bóveda de Esfinge en este equipo, si está abierta y en cuál se está " +
			"trabajando. Todas las demás herramientas actúan sobre esa bóveda y ninguna otra, así que " +
			"conviene mirar esto antes de guardar algo. Si está cerrada, la persona tiene que abrirla en " +
			"Esfinge: no se puede abrir desde aquí. **Y dice la versión de la Esfinge que contesta**, que " +
			"es la respuesta a «¿se ha actualizado?»: el número que el cliente enseña en su lista de " +
			"extensiones es el del paquete instalado y no cambia al actualizar Esfinge, porque el paquete " +
			"le cede el turno a la aplicación. Si no coinciden, manda éste.",
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
			"boveda": campoDeBoveda,
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
			"boveda": campoDeBoveda,
			"id": {Tipo: "string", Descripcion: "El identificador que devuelve esfinge_buscar."},
		}, "id"),
		Verbo:           QueVer,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_bovedas",
		Descripcion: "Las bóvedas que hay en este equipo y que puedes nombrar en `boveda`: la personal, " +
			"los proyectos y las que te han compartido. Dice cuál está abierta y **en cuál se puede " +
			"escribir**: una compartida puede ser de solo ver, y eso es un permiso y no un fallo. " +
			"Úsala cuando quien te habla nombre una bóveda y quieras asegurarte de cuál es, o cuando " +
			"te pregunte qué bóvedas tiene.",
		Esquema:         objeto(nil),
		Verbo:           QueBovedas,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_resumen",
		Descripcion: "Cuántas entradas hay en la bóveda y de qué clase, **en números y sin ningún " +
			"nombre**. Es lo que hay que usar para «¿qué tengo guardado?»: esfinge_buscar devuelve " +
			"solo las primeras y obliga a adivinar el resto.",
		Esquema:         objeto(map[string]Campo{"boveda": campoDeBoveda}),
		Verbo:           QueResumen,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         false,
	},
	{
		Nombre: "esfinge_higiene",
		Descripcion: "Lo que está mal en la bóveda, por identificador y sin secretos: contraseñas reutilizadas " +
			"en varios sitios, entradas duplicadas, credenciales sin segundo factor y tarjetas caducadas. " +
			"Dice cuáles son, nunca cuáles son las contraseñas.",
		Esquema:         objeto(map[string]Campo{"boveda": campoDeBoveda}),
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
			"boveda": campoDeBoveda,
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
			"boveda": campoDeBoveda,
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
		Descripcion: "Guarda una entrada nueva. " + enLaQueEsteAbierta +
			"Si pones generar:true, **Esfinge hace la contraseña y no te la devuelve** — eso es mejor " +
			"que inventarla tú, y luego se puede copiar con esfinge_copiar_contrasena; lo que sí te dice " +
			"es cuánto mide y de qué alfabeto sale. Tipos: credencial, nota, tarjeta, identidad, " +
			"personal, wifi. **Las llaves de acceso no se crean aquí**: las emite el sitio.",
		Esquema: objeto(map[string]Campo{
			"boveda": campoDeBoveda,
			"tipo":    {Tipo: "string", Descripcion: "credencial si no se dice nada.", Enum: []string{"credencial", "nota", "tarjeta", "identidad", "personal", "wifi"}},
			"titulo":  {Tipo: "string", Descripcion: "Cómo se llama. Obligatorio."},
			"usuario": {Tipo: "string"},
			"secreto": {Tipo: "string", Descripcion: "La contraseña. Mejor no ponerla y usar generar."},
			"notas":   {Tipo: "string"},
			"generar": {Tipo: "boolean", Descripcion: "Que la contraseña la haga Esfinge y no salga de ahí."},
			"carpeta": {Tipo: "string"},
			// **Con `generar`, qué alfabeto.** Por defecto hexadecimal, que es lo que
			// no rompe una URL; un sitio que exija símbolos necesita otro, y sin esto
			// había que generarla aparte y pegarla — o sea, hacerla pasar por el chat.
			"alfabeto": {Tipo: "string", Descripcion: "Con generar:true. hex, alnum o simbolos. " +
				"hex si no se dice nada, que es lo que no rompe una URL.",
				Enum: []string{"hex", "alnum", "simbolos"}},
			// **Sin sitios, una credencial nace sin poder rellenarse sola**, que es
			// para lo que existe la bóveda. El núcleo siempre supo aplicarlos; lo que
			// faltaba era decirlo aquí, y lo que no está en el esquema **no existe**
			// para el modelo. Lo dijo Claude al crear la primera entrada de verdad:
			// «desde aquí no puedo rellenar el campo de sitios».
			"sitios":    lista("Las direcciones donde se usa, p. ej. https://asana.com. Sin esto no se rellenará sola."),
			"etiquetas": lista("Para agrupar."),
		}, "titulo"),
		Verbo:           QueCrear,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         true,
	},
	{
		Nombre: "esfinge_editar",
		Descripcion: "Cambia campos de una entrada. " + enLaQueEsteAbierta +
			"Lo que no mandes **no se toca**. Cambiar el título, las notas o la carpeta va directo; " +
			"**cambiar la contraseña o el código tiene que aprobarlo la persona**, porque eso puede " +
			"dejar una cuenta sin forma de entrar. La contraseña anterior se guarda en el historial " +
			"de la entrada.",
		Esquema: objeto(map[string]Campo{
			"boveda": campoDeBoveda,
			"id":        {Tipo: "string", Descripcion: "El identificador que devuelve esfinge_buscar."},
			"titulo":    {Tipo: "string"},
			"usuario":   {Tipo: "string"},
			"secreto":   {Tipo: "string", Descripcion: "La contraseña nueva. Esto pide aprobación."},
			"notas":     {Tipo: "string"},
			"carpeta":   {Tipo: "string"},
			"sitios":    lista("Sustituye la lista entera, no añade."),
			"etiquetas": lista("Sustituye la lista entera, no añade."),
		}, "id"),
		Verbo:           QueEditar,
		PideAprobacion:  false,
		DevuelveSecreto: false,
		Escribe:         true,
	},
	{
		Nombre: "esfinge_borrar",
		Descripcion: "Manda una entrada a la papelera, donde se puede recuperar durante treinta días. " +
			enLaQueEsteAbierta +
			"**Tiene que aprobarlo la persona**: es lo único que quita algo de su vista.",
		Esquema: objeto(map[string]Campo{
			"boveda": campoDeBoveda,
			"id": {Tipo: "string", Descripcion: "El identificador que devuelve esfinge_buscar."},
		}, "id"),
		Verbo:           QueBorrar,
		PideAprobacion:  true,
		DevuelveSecreto: false,
		Escribe:         true,
	},
	{
		Nombre: "esfinge_mover",
		Descripcion: "Lleva una entrada **entera** de una bóveda a otra: su contraseña, su segundo factor y " +
			"su historial. **Tú no ves nada de eso**: lo lee y lo escribe Esfinge por dentro, igual que al " +
			"copiar al portapapeles. Es lo que hay que usar para «pasa esta cuenta a la bóveda de tal " +
			"cliente»: crear una a mano y pedirle a la persona que pegue la contraseña deja la credencial en " +
			"dos sitios y la obliga a manejarla.\n\n" +
			"De fábrica **mueve**: la entrada deja de estar en el origen, y **se va del todo, no a la " +
			"papelera** —en un movimiento no se pierde nada, porque está en el destino—. Con `dejar_copia` " +
			"se queda también en el origen; dilo solo si te lo piden, porque la misma credencial en dos " +
			"bóvedas se desfasa el día que una cambie y nada avisa.\n\n" +
			"Con `boveda` dices **de dónde sale** y con `destino` **a dónde va**. Si no dices el origen se " +
			"saca de la bóveda abierta, así que mira `esfinge_estado` antes: con un proyecto abierto, " +
			"mover «la cuenta de Brevo» se llevaría la del proyecto y no la tuya.\n\n" +
			"**Pregunta antes de mover, siempre**, diciendo de qué bóveda a cuál. Y si el destino es una " +
			"bóveda compartida, dilo con todas las letras: lo que muevas ahí lo verá **todo el que tenga " +
			"acceso**, y quitarlo después no deshace que lo hayan visto.",
		Esquema: objeto(map[string]Campo{
			"boveda": campoDeBoveda,
			"id":     {Tipo: "string", Descripcion: "La entrada que se lleva, por su identificador."},
			"destino": {
				Tipo:        "string",
				Descripcion: "A qué bóveda se lleva, por su nombre. `esfinge_bovedas` las lista.",
			},
			"dejar_copia": {
				Tipo:        "boolean",
				Descripcion: "Dejarla también en el origen. De fábrica no: se mueve.",
			},
		}, "id", "destino"),
		Verbo:           QueMover,
		PideAprobacion:  false,
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
