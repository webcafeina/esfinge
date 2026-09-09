package app

import (
	"context"
	"math/rand"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/iconos"
	"github.com/webcafeina/esfinge/internal/red"
)

// Los iconos de los sitios: cuándo se piden, a qué ritmo y qué se hace con ellos.
//
// **Esta es la segunda salida a la red del programa**, y la primera a sitios que
// no elegimos. El ritmo de aquí importa tanto como el filtro de `internal/iconos`:
// sesenta y cinco peticiones seguidas dibujan un pico muy reconocible para quien
// mire la red, y no hay ninguna prisa. Los iconos van apareciendo.

// Los tres números del goteo. Son campos y no constantes para poder acortarlos
// en las pruebas: sin eso, comprobar el camino entero costaría minutos y no se
// comprobaría nunca. Que es justo lo que pasó.
type ritmo struct {
	// entre es lo que se espera de una petición a la siguiente.
	//
	// **Espaciado a propósito**: bajarlos de golpe sería la firma más nítida
	// posible —«esta máquina acaba de abrir una bóveda con estos sesenta y cinco
	// sitios dentro»—. Pero no tanto como para que no se vea nada: con veinte
	// segundos, y guardando solo al final, el primer icono tardaba casi cuatro
	// minutos en aparecer y quien cerraba antes no se llevaba ninguno.
	entre time.Duration
	// alPrincipio, para no encadenar la primera petición con la apertura.
	alPrincipio time.Duration
	// cuantos por sesión. Una bóveda de sesenta y cinco se llena en tres.
	cuantos int
}

func ritmoNormal() ritmo {
	return ritmo{entre: 5 * time.Second, alPrincipio: 3 * time.Second, cuantos: 25}
}

// EventoIconos avisa a la ventana de que hay iconos nuevos que enseñar.
const EventoIconos = "iconos"

// ApuntarIconosA cambia quién baja los iconos. Lo usan las pruebas, que levantan
// un sitio de mentira.
//
// Función y no método, por lo de siempre: todo método exportado de *App cruza el
// puente, y dejar que la ventana eligiera a quién se le piden los iconos sería
// abrir justo la puerta que este paquete cierra.
func ApuntarIconosA(a *App, d *iconos.Descargador, r ritmo) {
	a.descargador = d
	a.ritmo = r
}

// IconosDeBoveda devuelve lo que hay, por anfitrión.
//
// **Va por su propio método y no dentro de `BuscarEnBoveda`**, y no es un
// capricho: la lista se vuelve a pedir en cada tecla del buscador, así que meter
// los iconos ahí sería mandarlos todos por el puente en cada pulsación. Aquí se
// piden una vez al abrir la lista.
func (a *App) IconosDeBoveda() (map[string]string, error) {
	if a.bov == nil || !a.bov.Abierta() {
		return nil, boveda.ErrCerrada
	}
	fuera := map[string]string{}
	for anfitrion, i := range a.bov.Iconos() {
		if i.URI != "" {
			fuera[anfitrion] = i.URI
		}
	}
	return fuera, nil
}

// buscarIconosSiProcede arranca el goteo, si toca.
//
// Se llama al abrir la bóveda y al importar. Devuelve enseguida: el trabajo va en
// su propia gorrutina y se cancela solo cuando la ventana se cierra o la bóveda
// se bloquea.
func (a *App) buscarIconosSiProcede(ctx context.Context) {
	// **Sin contexto no hay ventana, y sin ventana no hay a quién enseñarle un
	// icono.** El contexto es el de Wails: llega en Arrancar y se cancela al
	// cerrar. Trabajo de fondo que no se pueda cancelar no se empieza.
	if ctx == nil {
		return
	}
	// Los tres frenos, en el orden en que hay que mirarlos.
	if red.SinRed() || !a.ajustes.Ver().DescargarIconos {
		return
	}
	if a.bov == nil || !a.bov.Abierta() {
		return
	}
	go a.gotearIconos(ctx)
}

func (a *App) gotearIconos(ctx context.Context) {
	a.gotearIconosDesde(ctx, a.loQueFaltaPorMirar())
}

// gotearIconosDesde es lo mismo con la lista dada, que es como se puede probar el
// camino entero sin depender de lo que haya en la bóveda.
func (a *App) gotearIconosDesde(ctx context.Context, pendientes []string) {
	if len(pendientes) == 0 {
		return
	}
	// **En orden aleatorio, no alfabético.** Un goteo por orden alfabético es en sí
	// mismo una firma: quien mire la red ve la lista ordenada, que es más fácil de
	// reconocer que la misma lista revuelta.
	rand.Shuffle(len(pendientes), func(i, j int) {
		pendientes[i], pendientes[j] = pendientes[j], pendientes[i]
	})
	if len(pendientes) > a.ritmo.cuantos {
		pendientes = pendientes[:a.ritmo.cuantos]
	}

	if !esperar(ctx, a.ritmo.alPrincipio) {
		return
	}

	d := a.descargador
	if d == nil {
		d = iconos.Nuevo()
	}
	for i, anfitrion := range pendientes {
		if i > 0 && !esperar(ctx, a.ritmo.entre) {
			return
		}
		// La bóveda puede haberse bloqueado mientras se esperaba. Si es así se
		// abandona **y se tira lo traído**: mantenerla viva para poder guardar sería
		// convertir el bloqueo por inactividad en una promesa a medias.
		if a.bov == nil || !a.bov.Abierta() {
			return
		}

		uri, _ := d.De(ctx, anfitrion)

		// **Se guarda y se avisa uno a uno, no al terminar la tanda.**
		//
		// Guardando al final, el primer icono no aparecía hasta que habían caído los
		// doce —minutos— y quien cerraba la bóveda antes no se llevaba ninguno, ni
		// siquiera los que ya se habían bajado. Uno a uno, cada icono que llega se
		// queda, se ve, y el trabajo hecho no depende de llegar al final.
		//
		// Tanto el acierto como el fallo se apuntan: lo segundo es lo que evita
		// volver a preguntar mañana por un sitio que no tiene icono.
		if err := a.bov.PonerIconos(map[string]boveda.Icono{
			anfitrion: {URI: uri, Mirado: time.Now().UTC().Format(time.RFC3339)},
		}); err != nil {
			return
		}
		if uri != "" {
			// **Sin llamar a Actividad().** Todo método de bóveda la llama para
			// aplazar el bloqueo, y aquí sería el programa aplazándoselo a sí mismo:
			// una bóveda que no se cierra nunca porque está bajando iconos de fondo.
			// El plazo lo mueve quien está delante, no el trabajo de fondo.
			a.sistema.Avisar(EventoIconos, nil)
		}
	}
}

// loQueFaltaPorMirar son los anfitriones de la bóveda que no tienen icono ni un
// intento reciente.
func (a *App) loQueFaltaPorMirar() []string {
	sabidos := a.bov.Iconos()
	ahora := time.Now()

	visto := map[string]bool{}
	var faltan []string
	for _, e := range a.bov.Buscar("") {
		for _, sitio := range e.Sitios {
			anfitrion := iconos.Anfitrion(sitio)
			if anfitrion == "" || visto[anfitrion] {
				continue
			}
			visto[anfitrion] = true
			if boveda.TocaMirar(sabidos[anfitrion], ahora) {
				faltan = append(faltan, anfitrion)
			}
		}
	}
	return faltan
}

// esperar duerme lo que se le diga y dice si se puede seguir.
func esperar(ctx context.Context, cuanto time.Duration) bool {
	reloj := time.NewTimer(cuanto)
	defer reloj.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-reloj.C:
		return true
	}
}
