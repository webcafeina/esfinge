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
	// QueBovedas son las bóvedas que se pueden nombrar (ADR 0055): la personal, los
	// proyectos que estén en este equipo y las compartidas.
	//
	// **Hace falta para poder decir «en Zeri's»**, y tiene un coste que está dicho en
	// la ADR y en `docs/seguridad.md`: la lista de proyectos es la lista de clientes, y
	// la ADR 0050 la protegía a propósito —los ficheros se llaman por una referencia al
	// azar para que un listado de la carpeta no la delate—. No hay forma de dar lo que
	// se pide sin esto.
	QueBovedas = "bovedas"
	// QueMover lleva una entrada **entera** de una bóveda a otra (ADR 0056), con su
	// contraseña, su segundo factor y su historial.
	//
	// **El secreto no cruza el canal**: lo lee y lo escribe Esfinge por dentro, y el
	// agente sigue sin verlo — exactamente como copiar al portapapeles. Lo pidió el
	// agente del cliente al no poder hacerlo: lo único que tenía era sacar la
	// contraseña al portapapeles y que la pegara una persona a mano, que es **peor que
	// la funcionalidad** y además deja la misma credencial en dos sitios.
	QueMover = "mover"
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
	QueBovedas,
	QueMover,
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

	// Boveda es **en cuál se quiere la acción** (ADR 0055), por su nombre. Vacío es
	// «la que esté abierta», que es como funcionaba todo hasta la 2.55.0.
	//
	// **Va en la petición y no en un verbo `abrir`**, y eso no es estilo. Con un verbo,
	// el agente tendría que acordarse de haber conmutado y de volver, y lo que un
	// agente no recuerda entre llamadas es justo lo que se le olvida: una acción
	// acabaría en la bóveda de otro cliente sin que nada lo dijera. Pegada a la acción
	// no hay estado que perder.
	Boveda string `json:"boveda,omitempty"`

	// Destino es a qué bóveda se lleva una entrada, por su nombre (ADR 0056). Con
	// `boveda` se dice de dónde sale; con éste, a dónde va.
	Destino string `json:"destino,omitempty"`
	// DejarCopia la deja también en el origen. **De fábrica no**: duplicar una
	// credencial es lo que crea el problema de que un día cambie en el sitio, se
	// actualice una y la otra se quede desfasada sin que nada avise.
	DejarCopia bool `json:"dejarCopia,omitempty"`

	// Texto es lo que se busca.
	Texto string `json:"texto,omitempty"`
	// ID es la entrada sobre la que se pide algo.
	ID string `json:"id,omitempty"`
	// Bytes y Alfabeto son para generar.
	Bytes    int    `json:"bytes,omitempty"`
	Alfabeto string `json:"alfabeto,omitempty"`
	// Caracteres es el largo pedido, **y manda sobre Bytes** cuando llegan los dos.
	//
	// Un sitio limita lo que acepta **en caracteres**, no en bits, así que es lo que
	// alguien pide —«una de veinte»— y lo que un formulario rechaza. Hasta el 2026-10-19
	// solo se podían pedir bytes de entropía, y para acortar una contraseña había que
	// generarla aparte y pegarla en `secreto`: o sea **hacerla pasar por la conversación
	// de un modelo**, que es justo lo que `generar` existe para evitar.
	Caracteres int `json:"caracteres,omitempty"`

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
	// Boveda es el nombre de la que está abierta, y EsLaPersonal dice si es la de
	// siempre. Ver [QueEstado]: es lo que evita que un agente escriba en la bóveda de
	// otro cliente creyendo que está en la personal.
	//
	// **Van los dos y sin `omitempty`, y eso es el arreglo.** Antes `Boveda` iba vacío
	// en la personal y se omitía: «vacío es la personal» era un acuerdo que conocía el
	// código y no el agente, que solo veía **un campo ausente** y concluía —con razón—
	// que la herramienta no cumplía lo que su descripción promete. Un dato que hay que
	// saber interpretar para que signifique algo no está dicho. Lo señaló el agente del
	// cliente el 2026-10-13.
	Boveda       string `json:"boveda"`
	EsLaPersonal bool   `json:"esLaPersonal"`
	// SoloLectura dice que aquí no se puede escribir: una compartida de solo ver, o
	// una bóveda de una versión más nueva.
	SoloLectura bool `json:"soloLectura,omitempty"`
	// Version es la de la Esfinge **que está contestando**, y por eso va aquí y no en
	// una herramienta aparte: es el único dato que no se puede averiguar de ninguna
	// otra forma desde dentro de una conversación.
	//
	// Hace falta porque el paquete `.mcpb` **lleva el servidor dentro y no se actualiza
	// con la aplicación**: lo que Claude Desktop enseña en Extensiones es la versión del
	// paquete que se arrastró, y desde la 2.48.0 ése le cede el turno al binario de la
	// Esfinge instalada. O sea que el número de la pantalla y el que de verdad corre
	// **son distintos a propósito**, y hasta ahora no había forma de preguntar por el
	// segundo. Lo preguntó el cliente el 2026-10-14: «¿cómo sé que se ha actualizado?».
	//
	// Y de paso es el diagnóstico del relevo: si algún día no encuentra la aplicación y
	// contesta el binario viejo de dentro del paquete, **aquí se ve**. Sin esto, eso se
	// manifiesta como una herramienta que «no existe», que es un síntoma que no se
	// parece a su causa.
	Version string `json:"version"`
}

// UnaBoveda es una de las que el agente puede nombrar (ADR 0055).
//
// **Sin referencia**: el identificador al azar es un detalle de implementación y no le
// sirve a nadie para hablar. Lo que se nombra es el nombre.
type UnaBoveda struct {
	Nombre       string `json:"nombre"`
	EsLaPersonal bool   `json:"esLaPersonal"`
	Abierta      bool   `json:"abierta"`
	// SeEscribe: en una compartida de solo ver, no (ADR 0052). **Es un permiso y no un
	// fallo**, y se dice antes para que el agente no lo descubra al ir a guardar.
	SeEscribe bool `json:"seEscribe"`
	// Compartida dice que es de otra persona, para que el agente no la confunda con un
	// proyecto propio al hablar.
	Compartida bool `json:"compartida,omitempty"`
}

// Movido es el resultado de llevar una entrada de una bóveda a otra (ADR 0056).
//
// **Dice las dos bóvedas por su nombre**, por lo mismo que `Escrito`: un agente que no
// puede decir de dónde a dónde ha movido algo no puede contarlo, y quien se lo pidió no
// puede comprobarlo.
type Movido struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	Desde  string `json:"desde"`
	Hasta  string `json:"hasta"`
	// SeQuedaCopia dice si la del origen sigue ahí.
	SeQuedaCopia bool `json:"seQuedaCopia"`
	// ConSecreto dice si lo que se ha movido llevaba contraseña. **No dice cuál**: es
	// para que el agente pueda avisar de que ha movido algo sensible.
	ConSecreto bool `json:"conSecreto"`
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
	// Boveda es **dónde ha ido**, y EsLaPersonal si era la de siempre.
	//
	// **Va en la respuesta y no solo en `estado`, y eso es lo que cierra el caso.**
	// Mirar antes de escribir depende de que el modelo se acuerde; esto no depende de
	// nada: aunque guarde sin comprobar, al terminar tiene el dato delante y lo dice.
	// Y si no era la bóveda que la persona tenía en la cabeza, se ve **en el momento**
	// y no tres semanas después buscando una cuenta donde no está.
	Boveda       string `json:"boveda"`
	EsLaPersonal bool   `json:"esLaPersonal"`
	// Generada dice **cómo es** la contraseña que hizo Esfinge, nunca cuál.
	//
	// Sin esto, un agente que pide `generar: true` no puede decir si lo guardado
	// cumple lo que exige el sitio —largo mínimo, qué caracteres admite— y lo único
	// que le queda es pedir que se la copien al portapapeles para mirarla, que es
	// justo lo que este canal evita. Lo pidió el agente del cliente el 2026-10-13.
	Generada *ComoEs `json:"generada,omitempty"`
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
	// Bovedas son las que se pueden nombrar (ADR 0055).
	Bovedas []UnaBoveda `json:"bovedas,omitempty"`
	// Movido es lo que se ha llevado de una bóveda a otra (ADR 0056).
	Movido *Movido `json:"movido,omitempty"`
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
	// MotivoNoEsaBoveda: el nombre de bóveda que se ha dicho no lleva a ninguna, o
	// lleva a varias. **No es lo mismo que `no-esta`**, que habla de una entrada: aquí
	// el arreglo es decir otro nombre, no buscar otra cosa.
	MotivoNoEsaBoveda = "no-esa-boveda"
	// MotivoBovedaDormida: esa bóveda existe y **no está en este equipo**. Tiene motivo
	// propio porque el arreglo es distinto y no lo puede hacer el agente: hay que
	// abrirla una vez en Esfinge para que se baje.
	MotivoBovedaDormida = "boveda-dormida"
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

// ComoEs describe una contraseña **sin decirla**: cuánto mide y de qué alfabeto sale.
//
// **De fábrica en hexadecimal**, y eso no es un detalle del canal sino una regla de la
// casa: una contraseña con `/`, `+` o `@` dentro de una cadena de conexión la parte, y el
// error que sale no menciona la contraseña por ninguna parte. Se puede pedir otro
// alfabeto —hay sitios que exigen una mayúscula o un símbolo— y entonces esto es lo que
// deja decir si lo guardado cumple lo que pedía el sitio, **sin enseñarlo**.
type ComoEs struct {
	Largo    int    `json:"largo"`
	Alfabeto string `json:"alfabeto"`
}
