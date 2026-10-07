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
	QueGenerar,
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
	// Repetidas son los grupos de entradas que comparten contraseña. Cada grupo es
	// una lista de identificadores.
	Repetidas [][]string `json:"repetidas,omitempty"`
	// SinCodigo son las credenciales sin segundo factor.
	SinCodigo []string `json:"sinCodigo,omitempty"`
	// Caducadas son las tarjetas y documentos que ya han caducado.
	Caducadas []string `json:"caducadas,omitempty"`
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
	Clave   string   `json:"clave,omitempty"`
}

// Los motivos, que son etiquetas estables y no frases.
const (
	MotivoCerrada      = "cerrada"
	MotivoSinBoveda    = "sin-boveda"
	MotivoSinEmparejar = "sin-emparejar"
	MotivoDemasiado    = "demasiado"
	MotivoNoEntiendo   = "no-entiendo"
	MotivoNoEsta       = "no-esta"
	// MotivoSinEsfinge lo pone el binario cuando no hay nadie escuchando: Esfinge
	// no está abierta, o el canal está apagado en Ajustes.
	MotivoSinEsfinge = "sin-esfinge"
)

func bien() Respuesta { return Respuesta{OK: true} }

func mal(motivo, texto string) Respuesta {
	return Respuesta{OK: false, Motivo: motivo, Error: texto}
}
