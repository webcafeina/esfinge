package app

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/agente"
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
