package app

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/agente"
	"github.com/webcafeina/esfinge/internal/boveda"
)

// Trabajar en una bóveda que no es la abierta, y volver (ADR 0055).
//
// **El porqué, que es lo que no se puede volver a discutir.** Hasta la 2.55.0 un
// agente actuaba sobre la bóveda abierta y ninguna otra, y conmutar era cosa de la
// persona. Con varias bóvedas eso se convirtió en lo contrario de lo que se quería:
// «es muy molesto tener que ir cambiando de bóveda y que diga todo el rato que no
// puede hacer algo» (el cliente, 2026-10-14).
//
// El fallo de fondo era que **la bóveda era un modo**. Un modo hay que recordarlo, y
// lo que un agente no recuerda entre llamadas lo paga quien le pide las cosas: es la
// misma familia que los arreglos del 13 de octubre —escribir sin mirar dónde— con el
// signo cambiado. Ahora **la bóveda viaja pegada a cada acción**, y aquí no queda
// estado que al agente se le pueda olvidar.
//
// **Sigue habiendo una sola bóveda abierta a la vez** (ADR 0050). Lo que cambia es
// quién puede moverla.

// RespiroDelAgente es lo que se espera antes de devolver la ventana a su sitio.
//
// **No se vuelve tras cada llamada**, y eso es lo que hace esto soportable: tres
// acciones seguidas en la misma bóveda son **una ida y una vuelta** y no seis
// conmutaciones, porque cada una rearranca la sincronización, el goteo de iconos y el
// aviso a la ventana. Lo que lo fija no es el gusto: es cuánto tarda un agente entre
// dos llamadas de la misma tanda, que son segundos.
const RespiroDelAgente = 10 * time.Second

// loDelAgenteEnOtraBoveda es lo que hay que saber para devolver la ventana.
type loDelAgenteEnOtraBoveda struct {
	mu sync.Mutex
	// volver deshace la ida. Nil es que el agente no ha movido nada.
	volver func() error
	// sello es el contador de conmutaciones justo después de la del agente, **y es lo
	// que impide tirar de nadie**: si cuando vence el respiro no coincide, alguien ha
	// cambiado de bóveda por su cuenta y deshacerlo sería peor que no volver.
	//
	// **Un contador y no «en cuál estamos»**, que fue la primera versión y no servía:
	// mirar la bóveda activa no distingue «no se ha movido nadie» de «se fue y volvió
	// al mismo sitio», y en el segundo caso la vuelta le mueve la ventana a quien la
	// estaba usando. Lo dijo la prueba, no la lectura.
	sello uint64
	reloj *time.Timer
}

// unaBoveda es un destino ya resuelto.
type unaBoveda struct {
	personal bool
	dueno    string // vacío en un proyecto mío
	ref      string
	nombre   string
	permiso  string
}

// BovedaParaElAgente es una fila de las que el agente puede nombrar.
//
// **Sin `ref`**: la referencia al azar es un detalle de implementación —existe para
// que un listado de la carpeta no sea la lista de clientes (ADR 0050)— y no le sirve a
// nadie para hablar. Lo que se nombra es el nombre.
type BovedaParaElAgente struct {
	Nombre string `json:"nombre"`
	// EsLaPersonal distingue la de siempre sin que haya que adivinarlo por el nombre.
	EsLaPersonal bool `json:"esLaPersonal"`
	Abierta      bool `json:"abierta"`
	// SeEscribe dice si ahí se puede guardar: una compartida puede ser de solo ver
	// (ADR 0052), y **eso no es un fallo, es un permiso**.
	SeEscribe bool `json:"seEscribe"`
	// Compartida dice de quién es. Informativo, para que el agente no la confunda con
	// un proyecto propio al hablar.
	Compartida bool `json:"compartida,omitempty"`
}

// bovedasParaElAgente son las que puede nombrar: la personal, los proyectos **no
// archivados y que estén en este equipo**, y las compartidas.
//
// Lo archivado se queda fuera a propósito: archivar es decir «con esto ya no se
// trabaja», y ofrecérselo al agente sería deshacer esa decisión por la puerta de
// atrás. Lo dormido también, porque no se puede abrir sin bajarlo antes y eso es
// administrar.
func (a *App) bovedasParaElAgente() ([]BovedaParaElAgente, error) {
	activa := a.bovedaActiva()
	out := []BovedaParaElAgente{{
		Nombre: comoSeLlamaLaBoveda(""), EsLaPersonal: true,
		Abierta: activa == "", SeEscribe: true,
	}}

	proyectos, err := a.Proyectos()
	if err != nil {
		return nil, err
	}
	for _, p := range proyectos {
		if p.Archivado || !p.EnEsteEquipo {
			continue
		}
		out = append(out, BovedaParaElAgente{
			Nombre: p.Nombre, Abierta: activa == p.Ref, SeEscribe: true,
		})
	}

	// Las compartidas. Una retirada (ADR 0053) **no es una bóveda**: es el aviso de
	// una que se fue, y no se ofrece.
	compartidas, err := a.Compartidas()
	if err != nil {
		return out, nil // la lista de proyectos ya vale; no se cae todo por esto
	}
	for _, c := range compartidas {
		if c.Retirada != "" || !c.EnEsteEquipo {
			continue
		}
		out = append(out, BovedaParaElAgente{
			Nombre: c.Nombre, Abierta: activa == c.Ref,
			SeEscribe: c.Permiso == "editar", Compartida: true,
		})
	}
	return out, nil
}

// buscarLaBoveda resuelve un nombre.
//
// **Por nombre y no por referencia**, porque es lo que una persona le dice al agente.
// Si encajan dos, **no se elige**: con bóvedas de clientes, acertar por sorteo es el
// peor resultado posible.
func (a *App) buscarLaBoveda(nombre string) (unaBoveda, error) {
	quiero := strings.TrimSpace(nombre)
	if quiero == "" {
		return unaBoveda{}, fmt.Errorf("%w: no has dicho qué bóveda", agente.ErrNoEsaBoveda)
	}
	if strings.EqualFold(quiero, comoSeLlamaLaBoveda("")) {
		return unaBoveda{personal: true, nombre: comoSeLlamaLaBoveda("")}, nil
	}

	var encajan []unaBoveda
	proyectos, err := a.Proyectos()
	if err != nil {
		return unaBoveda{}, err
	}
	for _, p := range proyectos {
		if p.Archivado || !strings.EqualFold(p.Nombre, quiero) {
			continue
		}
		if !p.EnEsteEquipo {
			return unaBoveda{}, fmt.Errorf(
				"%w: «%s» todavía no está en este equipo. Ábrela una vez en Esfinge y se baja",
				agente.ErrBovedaDormida, p.Nombre)
		}
		encajan = append(encajan, unaBoveda{ref: p.Ref, nombre: p.Nombre})
	}
	if compartidas, err := a.Compartidas(); err == nil {
		for _, c := range compartidas {
			if c.Retirada != "" || !strings.EqualFold(c.Nombre, quiero) {
				continue
			}
			if !c.EnEsteEquipo {
				return unaBoveda{}, fmt.Errorf(
					"%w: «%s» todavía no está en este equipo. Ábrela una vez en Esfinge y se baja",
					agente.ErrBovedaDormida, c.Nombre)
			}
			encajan = append(encajan, unaBoveda{
				dueno: c.Dueno, ref: c.Ref, nombre: c.Nombre, permiso: c.Permiso,
			})
		}
	}

	switch len(encajan) {
	case 0:
		return unaBoveda{}, fmt.Errorf("%w: aquí no hay ninguna bóveda que se llame «%s»",
			agente.ErrNoEsaBoveda, quiero)
	case 1:
		return encajan[0], nil
	default:
		return unaBoveda{}, fmt.Errorf(
			"%w: hay %d bóvedas que se llaman «%s». Dilo de otra forma o renómbralas en Esfinge",
			agente.ErrNoEsaBoveda, len(encajan), quiero)
	}
}

// enLaBoveda hace algo en la bóveda que se nombra y deja programada la vuelta.
//
// **Va en fila de uno.** Dos acciones a la vez conmutarían una encima de la otra y la
// segunda trabajaría donde no cree estar, que es exactamente lo que esto viene a
// quitar.
func (a *App) enLaBoveda(nombre string, hacer func() error) error {
	destino, err := a.buscarLaBoveda(nombre)
	if err != nil {
		return err
	}

	a.ag.mu.Lock()
	defer a.ag.mu.Unlock()

	// Ya estamos ahí: ni se conmuta ni se programa vuelta ninguna. Es el caso normal
	// cuando alguien nombra la bóveda en la que ya está trabajando.
	if a.bovedaActiva() == destino.ref {
		return hacer()
	}

	// **A dónde volver se apunta la primera vez y no en cada salto.** Si el agente
	// encadena Zeri's y luego Beta, la vuelta sigue siendo a donde estaba la persona.
	if a.ag.volver == nil {
		a.ag.volver = a.comoVolverAqui()
	}
	if err := a.irA(destino, false); err != nil {
		// Si no se ha podido ir, no hay nada que deshacer **y la vuelta apuntada se
		// queda**: puede haber un salto anterior pendiente de deshacer.
		return err
	}
	a.ag.sello = a.cuantasConmutaciones()

	errDeLaAccion := hacer()
	a.programarLaVuelta()
	return errDeLaAccion
}

// comoVolverAqui devuelve qué habría que hacer para dejar esto como está ahora.
func (a *App) comoVolverAqui() func() error {
	a.mu.Lock()
	dueno, ref := a.duenoActivo, a.activa
	a.mu.Unlock()
	switch {
	case ref == "":
		return func() error { return a.volverALaPersonal(false) }
	case dueno != "":
		return func() error { return a.abrirCompartida(dueno, ref, false) }
	default:
		return func() error { return a.abrirProyecto(ref, false) }
	}
}

// irA conmuta de verdad, por el camino que ya existe.
func (a *App) irA(b unaBoveda, deUnaPersona bool) error {
	switch {
	case b.personal:
		return a.volverALaPersonal(deUnaPersona)
	case b.dueno != "":
		return a.abrirCompartida(b.dueno, b.ref, deUnaPersona)
	default:
		return a.abrirProyecto(b.ref, deUnaPersona)
	}
}

// programarLaVuelta arma el reloj del respiro, o lo retrasa si ya estaba armado.
//
// Se llama con `a.ag.mu` cogido.
func (a *App) programarLaVuelta() {
	if a.ag.reloj != nil {
		a.ag.reloj.Stop()
	}
	a.ag.reloj = time.AfterFunc(RespiroDelAgente, a.volverDeDondeLeDejoElAgente)
}

// volverDeDondeLeDejoElAgente devuelve la ventana a donde estaba, **si nadie la ha
// movido mientras tanto**.
func (a *App) volverDeDondeLeDejoElAgente() {
	a.ag.mu.Lock()
	defer a.ag.mu.Unlock()
	volver, sello := a.ag.volver, a.ag.sello
	a.ag.volver, a.ag.sello = nil, 0
	if volver == nil {
		return
	}
	// **Si alguien ha conmutado, no se toca nada.** Deshacer lo que una persona acaba
	// de hacer con el ratón es peor que dejarlo donde está, y además la dejaría en una
	// bóveda que no eligió.
	if a.cuantasConmutaciones() != sello {
		return
	}
	// Y con la bóveda cerrada no hay nada que devolver: el reloj de inactividad ya ha
	// hecho lo suyo, que es más de lo que íbamos a hacer aquí.
	if a.boveda() == nil {
		return
	}
	if err := volver(); err != nil {
		log.Printf("esfinge: no se ha podido devolver la ventana a su bóveda: %v", err)
	}
}

// EnLaPersonaLaDejo deshace ya una conmutación del agente, sin esperar al respiro.
//
// **Es lo que hace que «sin nombrar bóveda» signifique siempre lo mismo.** El respiro
// de la ADR 0055 deja la bóveda nombrada activa diez segundos para no conmutar seis
// veces en una tanda, y eso convertía la llamada siguiente sin `boveda` en una llamada
// sobre la bóveda de antes. Lo encontró el agente del cliente el 2026-10-15.
//
// **Respeta la misma regla que la vuelta por respiro**: si quien conmutó fue la
// persona, aquí no se toca nada.
func (f fuenteDelAgente) EnLaPersonaLaDejo() { f.a.volverDeDondeLeDejoElAgente() }

// EnLaBoveda es lo que el canal llama. Ver [App.enLaBoveda].
func (f fuenteDelAgente) EnLaBoveda(nombre string, hacer func() error) error {
	return f.a.enLaBoveda(nombre, hacer)
}

// Bovedas son las que el agente puede nombrar.
func (f fuenteDelAgente) Bovedas() ([]agente.UnaBoveda, error) {
	lista, err := f.a.bovedasParaElAgente()
	if err != nil {
		return nil, err
	}
	out := make([]agente.UnaBoveda, 0, len(lista))
	for _, b := range lista {
		out = append(out, agente.UnaBoveda{
			Nombre: b.Nombre, EsLaPersonal: b.EsLaPersonal, Abierta: b.Abierta,
			SeEscribe: b.SeEscribe, Compartida: b.Compartida,
		})
	}
	return out, nil
}

// ---------------------------------------------- llevar una entrada a otra bóveda

// Mover lleva una entrada entera de una bóveda a otra (ADR 0056).
//
// **El secreto no cruza el canal**: se lee y se escribe aquí dentro, y lo que vuelve
// dice qué se movió y a dónde, nunca qué llevaba. Es lo mismo que hace copiar al
// portapapeles, con la diferencia de que aquí ni siquiera sale del programa.
//
// Lo pidió el agente del cliente **al no poder hacerlo** (2026-10-15): lo único que
// tenía era crear la entrada a medias, sacar la contraseña al portapapeles y pedirle a
// una persona que la pegara. Eso es peor que la funcionalidad —obliga a manejar el
// secreto a mano y deja la misma credencial en dos bóvedas— y él mismo lo señaló.
func (f fuenteDelAgente) Mover(quien string, p agente.Peticion) (agente.Movido, error) {
	return f.a.moverEntreBovedas(siNoDice(quien), p)
}

// moverEntreBovedas es el camino entero, **con el cerrojo del agente cogido una sola
// vez**.
//
// No usa `enLaBoveda` y eso no es duplicar código: aquél conmuta una vez y programa la
// vuelta, y esto conmuta dos —al destino y de vuelta al origen— dentro del mismo
// cerrojo. Llamándolo se bloquearía contra sí mismo.
func (a *App) moverEntreBovedas(quien string, p agente.Peticion) (agente.Movido, error) {
	destino, err := a.buscarLaBoveda(p.Destino)
	if err != nil {
		return agente.Movido{}, err
	}

	a.ag.mu.Lock()
	defer a.ag.mu.Unlock()

	// **A dónde volver se apunta antes de tocar nada**, igual que en `enLaBoveda`: si
	// esto empieza con la persona en su bóveda, ahí tiene que acabar.
	if a.ag.volver == nil {
		a.ag.volver = a.comoVolverAqui()
	}

	// 1 · En el origen, que es donde `Atender` nos haya dejado o la que diga `boveda`.
	if p.Boveda != "" {
		otro, err := a.buscarLaBoveda(p.Boveda)
		if err != nil {
			return agente.Movido{}, err
		}
		// **Ya estar ahí no es conmutar.** Sin esto se reabría la misma bóveda, y cada
		// reapertura cierra la válvula y borra el sí concedido.
		if a.bovedaActiva() != otro.ref {
			// **Y la conmutación no puede tirar la aprobación de esta misma
			// operación.** Ver `elSiEnLaMano`: sin esto, mover desde una bóveda que no
			// es la abierta entra en bucle pidiendo permiso.
			para, hasta := a.permisos.elSiEnLaMano()
			if err := a.irA(otro, false); err != nil {
				return agente.Movido{}, err
			}
			a.permisos.devolverElSi(para, hasta)
		}
	}
	origen := a.comoVolverAqui()
	nombreOrigen := comoSeLlamaLaBoveda(a.EstadoBoveda().NombreDelProyecto)
	if a.bovedaActiva() == destino.ref {
		return agente.Movido{}, fmt.Errorf("«%s» ya está en esa bóveda", nombreOrigen)
	}

	b, err := a.bovedaParaEscribirAgente()
	if err != nil {
		return agente.Movido{}, err
	}
	// **La entrada entera, con su secreto.** `b.Ver` devuelve el original; lo que lo
	// recorta es la capa del agente, y aquí no se pasa por ella.
	e, hay := b.Ver(p.ID)
	if !hay || e.Papelera {
		return agente.Movido{}, agente.ErrNoEsta
	}
	movida := e.Copia()
	conSecreto := e.Secreto != ""

	// **Y si va a quitarla del origen, lo aprueba la persona en la ventana.**
	//
	// La frontera es «lo que quita algo de tu bóveda», que es la misma de `borrar`: lo
	// decidió el cliente el 2026-10-15 cuando su agente señaló la inconsistencia —mover
	// es **más** destructiva que borrar, porque no pasa por la papelera, y era la única
	// acción del canal que no pedía nada—. Con `dejar_copia` no quita nada y va directo.
	//
	// **Y no pasa por la válvula ni por «No volver a preguntar»**, al revés que borrar.
	// Eso no es una excepción inventada: la razón por la que borrar sí pasa fue suya
	// —«de todos modos iría a la papelera»— y aquí no va, así que la razón no vale. Es
	// la misma regla que el código de un solo uso y que cambiar una contraseña: lo que
	// no se deshace no pasa por la válvula.
	if !p.DejarCopia {
		ahora := time.Now()
		q := loQuePideUnAgente{
			Quien: quien, Que: agente.QueMover, ID: p.ID, Titulo: e.Titulo,
			Cuando: ahora.UTC().Format(time.RFC3339),
			Boveda: nombreOrigen, Destino: destino.nombre,
		}
		vale, _ := a.permisos.pedir(q, false, ahora)
		if !vale {
			a.sistema.Avisar(EventoAgenteQuiere, q)
			a.ag.sello = a.cuantasConmutaciones()
			a.programarLaVuelta()
			return agente.Movido{}, agente.ErrPideAprobacion
		}
	}

	// 2 · En el destino, **y primero**: si esto falla, en el origen no se ha tocado
	// nada. Al revés —quitar y luego escribir— un fallo en medio perdería la entrada.
	if err := a.irA(destino, false); err != nil {
		return agente.Movido{}, err
	}
	dest, err := a.bovedaParaEscribirAgente()
	if err != nil {
		// De vuelta al origen antes de rendirse: dejarlo en la bóveda de otro porque
		// no se podía escribir ahí sería cambiar de sitio a quien está mirando.
		_ = origen()
		return agente.Movido{}, err
	}
	// **Identificador nuevo**: los de una bóveda son suyos, y llevarse el de la otra
	// podría chocar con uno de aquí.
	nuevoID, err := dest.PonerNueva(movida)
	if err != nil {
		_ = origen()
		return agente.Movido{}, err
	}
	a.avisarDeQueAlguienHaEscrito()

	// 3 · De vuelta al origen, a quitarla.
	//
	// **Y si esto falla, se dice.** La entrada está en el destino y sigue aquí: eso es
	// una credencial duplicada, que es justo lo que este verbo viene a evitar, y
	// callarlo dejaría a alguien creyendo que la ha sacado de su bóveda personal.
	if err := origen(); err != nil {
		return agente.Movido{}, fmt.Errorf(
			"se ha copiado a «%s» y no se ha podido quitar de «%s», así que está en las dos: %w",
			destino.nombre, nombreOrigen, err)
	}
	if !p.DejarCopia {
		if err := a.quitarDelTodo(p.ID); err != nil {
			return agente.Movido{}, fmt.Errorf(
				"se ha copiado a «%s» y no se ha podido quitar de «%s», así que está en las dos: %w",
				destino.nombre, nombreOrigen, err)
		}
		a.avisarDeQueAlguienHaEscrito()
	}

	// **Al registro de la bóveda de origen**, que es de Go y no depende de que el
	// modelo se acuerde de contarlo (ADR 0056).
	a.apuntarMovimiento(quien, e.Titulo, nombreOrigen, destino.nombre, conSecreto)

	a.ag.sello = a.cuantasConmutaciones()
	a.programarLaVuelta()
	return agente.Movido{
		ID: nuevoID, Titulo: e.Titulo,
		Desde: nombreOrigen, Hasta: destino.nombre,
		SeQuedaCopia: p.DejarCopia, ConSecreto: conSecreto,
	}, nil
}

// avisarDeQueAlguienHaEscrito es lo que ya hace `escrito` para los otros verbos: sin
// esto, la bóveda queda bien y **la pantalla sigue con lo de antes**.
//
// Aquí se manda **dos veces**, una por bóveda tocada, y no es de más: la ventana puede
// estar mirando cualquiera de las dos.
func (a *App) avisarDeQueAlguienHaEscrito() {
	a.sistema.Avisar(EventoBovedaCambiada, nil)
}

// apuntarMovimiento deja constancia en el registro de la bóveda de origen (ADR 0056).
//
// **Es la mitad que no depende del modelo.** La tarjeta de la ventana no se usa aquí
// —lo decidió el cliente: que pregunte el agente en la conversación—, y una frase en una
// descripción la lee el modelo y puede saltársela. Esto lo escribe Go, así que la
// pregunta «¿qué ha hecho un agente con mis secretos?» se sigue pudiendo contestar.
func (a *App) apuntarMovimiento(quien, titulo, desde, hasta string, conSecreto bool) {
	b := a.boveda()
	if b == nil {
		return
	}
	que := agente.QueMover
	if !conSecreto {
		// Sin secreto dentro, mover es mover papeles. Se apunta igual —lo que se mueve
		// y a dónde es lo que importa— pero se distingue de lo que lleva una llave.
		que = agente.QueMover + ":sin-secreto"
	}
	if err := b.Apuntar(boveda.Apunte{
		Quien: quien, Que: que, Titulo: titulo,
		Sobre: desde + " → " + hasta, Resultado: boveda.ApunteHecho,
	}); err != nil {
		log.Printf("esfinge: no se ha podido apuntar un movimiento entre bóvedas: %v", err)
	}
}

// quitarDelTodo manda la entrada a la papelera y la saca de ella, que es la única
// forma de borrar de verdad **dejando lápida**.
//
// La lápida no es un detalle: sin ella, el equipo de al lado tiene la entrada viva y en
// la siguiente sincronización la resucita, con lo que mover dejaría la credencial del
// cliente de vuelta en la bóveda personal sin que nadie lo hubiera pedido.
//
// Y los dos pasos son a propósito en `BorrarDelTodo` —«una entrada viva se borra en dos
// pasos»—; aquí se dan los dos de golpe porque **no se pierde nada**: la entrada está en
// el destino, y deshacerlo es moverla de vuelta.
func (a *App) quitarDelTodo(id string) error {
	b := a.boveda()
	if b == nil {
		return boveda.ErrCerrada
	}
	if err := b.Borrar(id); err != nil {
		return err
	}
	return b.BorrarDelTodo(id)
}
