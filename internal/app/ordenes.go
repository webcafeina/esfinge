package app

// EventoOrden lleva a la ventana lo que se ha pedido desde el menú del sistema.
//
// El menú vive en Go —lo dibuja el sistema, no la interfaz— pero casi todo lo
// que se puede pedir desde él ocurre dentro de la ventana: cambiar de pestaña,
// copiar lo que hay seleccionado, pegar en el campo que tiene el foco. Así que
// el menú no hace: pide, y la interfaz hace.
const EventoOrden = "orden"

// Orden es lo que viaja en ese evento.
type Orden struct {
	Que string `json:"que"`
	// Texto solo lo lleva «editar:pegar»: el portapapeles del sistema lo lee Go,
	// porque el navegador no deja leerlo sin permiso y dentro de una ventana de
	// escritorio ese permiso no lo puede dar nadie.
	Texto string `json:"texto"`
}

// Órdenes que entiende la interfaz. Están aquí y no sueltas en el menú para que
// las dos puntas del cable se escriban en el mismo sitio.
const (
	OrdenIrACifrar    = "ir:cifrar"
	OrdenIrADescifrar = "ir:descifrar"
	OrdenIrAGenerar   = "ir:generar"
	OrdenIrAHistorial = "ir:historial"
	OrdenIrAAjustes   = "ir:ajustes"

	OrdenDeshacer        = "editar:deshacer"
	OrdenRehacer         = "editar:rehacer"
	OrdenCortar          = "editar:cortar"
	OrdenCopiar          = "editar:copiar"
	OrdenPegar           = "editar:pegar"
	OrdenSeleccionarTodo = "editar:seleccionar-todo"
	OrdenBuscarVersion   = "actualizar:buscar"
)

// Ordenar manda una orden sin datos a la ventana.
//
// La usa el menú del sistema. Está exportada —y por tanto cruza el puente— a
// propósito: es lo que permite ejercitar desde las pruebas de interfaz lo que en
// la aplicación de verdad dispara el menú, que no se puede pulsar desde un
// navegador. Pedir cambiar de pestaña no da acceso a nada.
func (a *App) Ordenar(que string) { a.sistema.Avisar(EventoOrden, Orden{Que: que}) }

// OrdenarPegar manda el texto del portapapeles, que solo Go puede leer.
func (a *App) OrdenarPegar(texto string) {
	a.sistema.Avisar(EventoOrden, Orden{Que: OrdenPegar, Texto: texto})
}
