package agente

// Lo que un agente de IA puede pedirle a la bóveda, y nada más (ADR 0054).
//
// **Esta lista es a los agentes lo que `LoQueSePuedePedir` es al navegador**, y hay
// una prueba que obliga a que añadir un verbo sea una decisión y no un descuido. La
// diferencia con aquélla, y es la que gobierna todo este paquete:
//
//	al otro lado no hay una persona con una pestaña delante, hay un programa que
//	lee páginas, ficheros y correos, y **cualquiera de esos textos puede decirle
//	qué pedir**. No hace falta atacar a Esfinge: basta con hablarle al que tiene la
//	llave.
//
// De ahí salen las dos reglas que no se tocan:
//
//  1. **El agente actúa sin ver.** La contraseña va al portapapeles del sistema y él
//     recibe «copiado». Lo único que llega a ver es el código de un solo uso, que
//     caduca en treinta segundos y no sirve sin la contraseña.
//  2. **Lo que entrega un secreto se aprueba en la ventana**, cada vez, con la
//     válvula de cinco minutos que el cliente eligió — y que **no cubre el código**,
//     justamente porque es lo único que se le enseña.
const (
	// QueEstado dice si hay bóveda, si está abierta y en cuál se está trabajando.
	//
	// **Lleva el nombre de la bóveda activa**, que en el canal del navegador no
	// hace falta y aquí sí: un agente que no sepa si está dentro de un proyecto
	// puede guardar la cuenta de un cliente en la bóveda de otro. Es un nombre, no
	// un secreto, y solo lo recibe quien ya está emparejado.
	QueEstado = "estado"
	// QueEmparejar pide permiso. Se contesta sin testigo, porque es el verbo que
	// sirve para conseguirlo.
	QueEmparejar = "emparejar"
	// QueBuscar devuelve las entradas que coinciden, **sin secretos**.
	//
	// La búsqueda **no mira los secretos**, como la de la ventana: comparar lo que
	// alguien teclea contra las contraseñas es una forma silenciosa de averiguarlas
	// a base de probar, y con un programa al otro lado eso deja de ser una
	// posibilidad teórica.
	QueBuscar = "buscar"
	// QueVer devuelve una entrada **sin secretos**, con los campos que no lo son.
	QueVer = "ver"
	// QueHigiene cuenta lo que está mal: repetidas, caducadas y sin segundo factor.
	//
	// Es lo que de verdad justifica esta funcionalidad: es trabajo de inventario,
	// aburrido, y **no toca un solo secreto**. Devuelve identificadores y cuentas,
	// nunca contraseñas.
	QueHigiene = "higiene"

	// QueResumen dice **cuántas hay de cada clase, y ni un nombre** (2026-10-08).
	//
	// Nació de verlo usar: a «¿cuántas cuentas tengo?» un agente que solo tiene
	// `buscar` contesta trayéndose las primeras entradas **con sus títulos y sus
	// usuarios** y deduciendo el resto —«las otras nueve no las he visto»—. Con esto
	// la contesta con números, que es a la vez la respuesta más útil y la que menos
	// cuenta de la bóveda.
	QueResumen = "resumen"
	// QueCopiarSecreto pone la contraseña de una entrada en el portapapeles.
	//
	// **El agente no la ve**: copia Esfinge y lo que vuelve es «copiado, se borra en N
	// segundos». Es lo mismo que hace el canal del navegador desde la entrega 1, y aquí
	// vale por una razón de más: lo que el agente recibiera entraría en la conversación
	// de un modelo y se quedaría ahí.
	//
	// **Y hace falta un sí en la ventana, cada vez.** Es lo único que hay entre un
	// agente al que alguien le ha dicho qué pedir y tu bóveda.
	QueCopiarSecreto = "copiar-secreto"
	// QueCodigo devuelve el código de un solo uso de una entrada.
	//
	// **Es lo único que el agente llega a ver**, y es la excepción que el cliente
	// eligió a sabiendas: seis cifras que caducan en treinta segundos y que no sirven
	// sin la contraseña. Dárselas es lo que le deja teclearlas donde hagan falta —un
	// formulario, un script, un `ssh`— sin pasar por el portapapeles.
	//
	// **Y por eso la válvula no lo cubre.** El trato de la válvula es «actúa por mí
	// durante cinco minutos», y actuar es reversible: el portapapeles se borra solo.
	// Enseñar no lo es — en cuanto esas cifras entran en el contexto del modelo están
	// en su transcripción. Lo que puede salir en ráfaga es lo que no sale de este
	// equipo; esto pregunta **siempre**.
	QueCodigo = "codigo"
	// QueCrear guarda una entrada nueva. **Va directo**, como las escrituras del
	// navegador (ADR 0032) y por la misma razón: lo escrito se puede deshacer —la
	// papelera guarda treinta días y la contraseña anterior queda en el historial de
	// la entrada—.
	QueCrear = "crear"
	// QueEditar cambia campos de una entrada.
	//
	// **Cambiar un secreto pregunta; lo demás no.** Las escrituras del navegador están
	// acotadas por el sitio de la pestaña —solo puede guardar una credencial para el
	// sitio donde está la persona— y **un agente no tiene sitio**, así que sin esa
	// puerta podría reescribir la contraseña del banco sin que nadie preguntara.
	QueEditar = "editar"
	// QueBorrar la manda a la papelera, y **pide un sí**: es lo único que quita algo de
	// la vista.
	QueBorrar = "borrar"
	// QueGenerar devuelve una contraseña nueva. **No toca la bóveda**: ni la lee ni
	// escribe en ella, así que no pide nada ni deja rastro.
	//
	// Y sí, lo que devuelve es un secreto que el agente ve — pero **todavía no es
	// de nadie**: no abre ninguna cuenta mientras no se guarde en algún sitio. Lo
	// que el agente haga con ella después es lo que importa, y eso ya pasa por
	// `guardar`.
	QueGenerar = "generar"
)

// LoQueSePuedePedir son los verbos que existen. **Añadir uno es una decisión**, y
// hay una prueba que lo exige: recorre esta lista y ninguno puede contestar que no
// se entiende.
//
// Lo que **no** está aquí y no va a estar: nada de administrar. Ni exportar, ni
// importar, ni la contraseña maestra, ni la recuperación, ni borrar la bóveda, ni
// cuentas, ni crear o entregar proyectos, ni compartir. Un agente trabaja **dentro**
// de la bóveda que ya está abierta; no decide cuál es ni se la lleva.
var LoQueSePuedePedir = []string{
	QueEstado,
	QueEmparejar,
	QueBuscar,
	QueVer,
	QueHigiene,
	QueResumen,
	QueGenerar,
	QueCopiarSecreto,
	QueCodigo,
	QueCrear,
	QueEditar,
	QueBorrar,
}

// TopeDeResultados es cuántas entradas vuelven como mucho de una búsqueda.
//
// **No es una optimización: es la segunda mitad de lo que la ADR 0024 protege.** Una
// bóveda de dos mil entradas volcada entera al contexto de un modelo es la lista
// completa de sitios y usuarios de una persona —exactamente lo que se decidió cifrar en
// el disco— y ahí ya no la protege nadie. Con un tope, una búsqueda sin filtro da una
// muestra **y el total**, que es lo que hace falta para decir «tienes 1.843 cuentas»
// sin enumerarlas.
const TopeDeResultados = 25

// VersionDelProtocolo la mandan los dos lados en cada petición. Sirve para que una
// versión vieja del binario y una nueva de Esfinge no se entiendan a medias: o se
// entienden o se dice que no.
const VersionDelProtocolo = 1

// Peticion es lo que llega por el canal.
type Peticion struct {
	Version int    `json:"version"`
	Testigo string `json:"testigo,omitempty"`
	// Quien es cómo se llama el agente, para poder escribirlo en el diálogo de la
	// ventana. **No se cree**: sirve para que la persona sepa a quién está diciendo
	// que sí, no para decidir nada.
	Quien string `json:"quien,omitempty"`
	Que   string `json:"que"`

	// Texto es lo que se busca.
	Texto string `json:"texto,omitempty"`
	// ID es la entrada sobre la que se pide algo.
	ID string `json:"id,omitempty"`
	// Bytes y Alfabeto son para generar.
	Bytes    int    `json:"bytes,omitempty"`
	Alfabeto string `json:"alfabeto,omitempty"`

	// Campos es lo que se escribe, al crear y al editar.
	//
	// **Un mapa y no una estructura**, a propósito: al editar, lo que decide qué se
	// toca es **qué claves vienen**, no qué valores. Con una estructura no habría forma
	// de distinguir «no lo toques» de «déjalo vacío», y con el mapa una clave con
	// cadena vacía vacía el campo porque alguien lo ha pedido.
	Campos map[string]string `json:"campos,omitempty"`
	// Sitios y Etiquetas van aparte porque son listas. Nulas quiere decir «no las
	// toques»; una lista vacía, «déjalas vacías».
	Sitios    []string `json:"sitios,omitempty"`
	Etiquetas []string `json:"etiquetas,omitempty"`
	// Generar pide que la contraseña la haga Esfinge. **Entonces no vuelve**: el
	// agente crea una cuenta con una contraseña que nunca ha visto, que es
	// estrictamente mejor que una que se invente él.
	Generar bool `json:"generar,omitempty"`
}

// Estado es lo poco que se dice sin haber pedido nada.
type Estado struct {
	Existe  bool `json:"existe"`
	Abierta bool `json:"abierta"`
	// Boveda es el nombre de la que está abierta, vacío si es la personal. Ver
	// [QueEstado]: es lo que evita que un agente escriba en la bóveda de otro
	// cliente creyendo que está en la personal.
	Boveda string `json:"boveda,omitempty"`
	// SoloLectura dice que aquí no se puede escribir: una compartida de solo ver, o
	// una bóveda de una versión más nueva.
	SoloLectura bool `json:"soloLectura,omitempty"`
}

// Entrada es lo que el agente ve de una entrada de la bóveda. **No lleva ni un
// secreto**, y los dos campos de abajo existen por una razón que ya costó una vez.
type Entrada struct {
	ID        string   `json:"id"`
	Tipo      string   `json:"tipo"`
	Titulo    string   `json:"titulo"`
	Usuario   string   `json:"usuario,omitempty"`
	Sitios    []string `json:"sitios,omitempty"`
	Carpeta   string   `json:"carpeta,omitempty"`
	Etiquetas []string `json:"etiquetas,omitempty"`
	Cambiada  string   `json:"cambiada,omitempty"`

	// TieneSecreto y TieneCodigo dicen **si los hay**, no cuáles son.
	//
	// Hacen falta porque lo que sale de la bóveda pasa por `SinSecretos`, que vacía
	// la semilla del código **sin dejar marca de que la hubiera**: leer `TOTP != ""`
	// sobre eso da siempre falso. Ya pasó al añadir `tieneCodigo` al canal del
	// navegador, y **todas las cuentas salían sin segundo factor**. Se calculan
	// dentro de la bóveda, mirando la entrada entera.
	TieneSecreto bool `json:"tieneSecreto"`
	TieneCodigo  bool `json:"tieneCodigo"`
}

// Higiene es lo que está mal en la bóveda, por identificador. **Sin secretos**: lo
// que dice es «estas tres comparten contraseña», no cuál es.
type Higiene struct {
	// Reutilizadas son los grupos de entradas que **comparten contraseña**.
	//
	// **Se llama así y no «repetidas» a propósito.** En este repo `Repetidas` es otra
	// cosa —«la misma cuenta dos veces», un ayudante para importar— y publicar esa
	// palabra aquí sería contestar la pregunta equivocada con toda confianza, a un
	// modelo que no tiene cómo saberlo.
	Reutilizadas [][]string `json:"reutilizadas,omitempty"`
	// SinCodigo son las credenciales sin segundo factor.
	SinCodigo []string `json:"sinCodigo,omitempty"`
	// Caducadas son las tarjetas y documentos que ya han caducado.
	Caducadas []string `json:"caducadas,omitempty"`
}

// Copiado es lo que se contesta tras poner algo en el portapapeles.
//
// **No lleva lo copiado**, y el tipo se llama así para que buscar quién toca una
// contraseña en este paquete sea buscar un nombre. Es el mismo de `internal/navegador`,
// duplicado a propósito como todo lo demás del protocolo.
type Copiado struct {
	// Portapapeles son los segundos que tardará en borrarse solo, 0 si no se borra.
	Portapapeles int `json:"portapapeles"`
	// Titulo es de qué entrada se ha copiado, para que el agente pueda decirlo.
	Titulo string `json:"titulo,omitempty"`
}

// Codigo es el de un solo uso, ya calculado. **La semilla no sale nunca**: lo que se
// da son las seis cifras de ahora, igual que al navegador.
type Codigo struct {
	Codigo string `json:"codigo"`
	// Quedan son los segundos que le sobran de vida, para que el agente sepa si le da
	// tiempo a usarlo o tiene que pedir otro.
	Quedan int `json:"quedan"`
}

// Escrito es lo que se contesta tras crear o cambiar algo. **Nunca lleva el secreto**,
// ni siquiera cuando lo acaba de generar Esfinge.
type Escrito struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	// Cambiados son los campos que se han tocado, para que el agente pueda decir qué
	// ha hecho sin tener que volver a leerlos.
	Cambiados []string `json:"cambiados,omitempty"`
	// ALaPapelera y Dias, al borrar: **decir que son treinta días es parte de la
	// pregunta**, porque si no se está contestando algo más grave de lo que pasa.
	ALaPapelera bool `json:"alaPapelera,omitempty"`
	Dias        int  `json:"dias,omitempty"`
}

// Respuesta es lo que se contesta. Siempre lleva `ok`.
type Respuesta struct {
	OK bool `json:"ok"`
	// Error es para leer; **Motivo es para decidir**. La misma separación que en el
	// canal del navegador: un cliente que mire el texto se rompe en cuanto alguien
	// mejore una frase.
	Error  string `json:"error,omitempty"`
	Motivo string `json:"motivo,omitempty"`

	Estado   *Estado   `json:"estado,omitempty"`
	Testigo  string    `json:"testigo,omitempty"`
	Entradas []Entrada `json:"entradas,omitempty"`
	// Cuantas son las que coinciden **en total**, que pueden ser más de las que
	// vienen: ver [TopeDeResultados].
	Cuantas int      `json:"cuantas,omitempty"`
	Entrada *Entrada `json:"entrada,omitempty"`
	Higiene *Higiene `json:"higiene,omitempty"`
	// Resumen son los números de la bóveda, sin un solo nombre dentro.
	Resumen *Recuento `json:"resumen,omitempty"`
	Copiado *Copiado  `json:"copiado,omitempty"`
	Codigo  *Codigo   `json:"codigo,omitempty"`
	Escrito *Escrito  `json:"escrito,omitempty"`
	Clave   string    `json:"clave,omitempty"`
}

// Los motivos, que son etiquetas estables y no frases.
const (
	MotivoCerrada      = "cerrada"
	MotivoSinBoveda    = "sin-boveda"
	MotivoSinEmparejar = "sin-emparejar"
	MotivoDemasiado    = "demasiado"
	MotivoNoEntiendo   = "no-entiendo"
	MotivoNoEsta       = "no-esta"
	// MotivoPideAprobacion: hace falta un sí en la ventana. **No es un error**: es el
	// camino normal la primera vez que se pide algo, y lo que hay que hacer es
	// aprobarlo y **volver a pedirlo**.
	MotivoPideAprobacion = "pide-aprobacion"
	// MotivoNoSeEscribe: aquí no se puede escribir —una bóveda de solo lectura, o una
	// compartida de solo ver—, o lo que se pide no se puede crear a mano.
	MotivoNoSeEscribe = "no-se-escribe"
	// MotivoSinEsfinge lo pone el binario cuando no hay nadie escuchando: Esfinge
	// no está abierta, o el canal está apagado en Ajustes.
	MotivoSinEsfinge = "sin-esfinge"
)

func bien() Respuesta { return Respuesta{OK: true} }

func mal(motivo, texto string) Respuesta {
	return Respuesta{OK: false, Motivo: motivo, Error: texto}
}

// Recuento es lo que hay en la bóveda **en números**.
//
// Es el espejo del de `internal/boveda` y vive aquí porque este paquete no importa la
// bóveda: el binario que habla MCP no sabe abrir ninguna, y así sigue.
type Recuento struct {
	Total     int            `json:"total"`
	Papelera  int            `json:"papelera"`
	PorClase  map[string]int `json:"porClase"`
	ConCodigo int            `json:"conCodigo"`
	SinCodigo int            `json:"sinCodigo"`
}
