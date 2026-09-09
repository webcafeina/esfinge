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

const (
	// entreIconos es lo que se espera entre una petición y la siguiente.
	//
	// **Es lento a propósito.** Bajarlos de golpe sería la firma más nítida
	// posible: «esta máquina acaba de abrir una bóveda con estos sesenta y cinco
	// sitios dentro». Espaciado, se confunde con la navegación de cualquiera.
	entreIconos = 20 * time.Second

	// porTanda es cuántos se traen cada vez que se abre la bóveda. Con esto una
	// bóveda de sesenta y cinco tarda unas cuantas sesiones en llenarse, que es
	// exactamente lo que se quiere.
	porTanda = 12

	// alPrincipio es lo que se espera antes de empezar, para no encadenar la
	// primera petición con la apertura de la bóveda.
	alPrincipio = 10 * time.Second
)

// EventoIconos avisa a la ventana de que hay iconos nuevos que enseñar.
const EventoIconos = "iconos"

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
	pendientes := a.loQueFaltaPorMirar()
	if len(pendientes) == 0 {
		return
	}
	// **En orden aleatorio, no alfabético.** Un goteo por orden alfabético es en sí
	// mismo una firma: quien mire la red ve la lista ordenada, que es más fácil de
	// reconocer que la misma lista revuelta.
	rand.Shuffle(len(pendientes), func(i, j int) {
		pendientes[i], pendientes[j] = pendientes[j], pendientes[i]
	})
	if len(pendientes) > porTanda {
		pendientes = pendientes[:porTanda]
	}

	if !esperar(ctx, alPrincipio) {
		return
	}

	d := iconos.Nuevo()
	traidos := map[string]boveda.Icono{}
	for i, anfitrion := range pendientes {
		if i > 0 && !esperar(ctx, entreIconos) {
			break
		}
		// La bóveda puede haberse bloqueado mientras se esperaba. Si es así se
		// abandona **y se tira lo traído**: mantenerla viva para poder guardar sería
		// convertir el bloqueo por inactividad en una promesa a medias.
		if a.bov == nil || !a.bov.Abierta() {
			return
		}

		uri, err := d.De(ctx, anfitrion)
		// Tanto el acierto como el fallo se apuntan: lo segundo es lo que evita
		// volver a preguntar mañana por un sitio que no tiene icono.
		traidos[anfitrion] = boveda.Icono{
			URI:    uri,
			Mirado: time.Now().UTC().Format(time.RFC3339),
		}
		_ = err
	}

	if len(traidos) == 0 || a.bov == nil || !a.bov.Abierta() {
		return
	}
	if err := a.bov.PonerIconos(traidos); err != nil {
		return
	}
	// **Sin llamar a Actividad().** Todo método de bóveda la llama para aplazar el
	// bloqueo, y aquí sería el programa aplazándoselo a sí mismo: una bóveda que no
	// se cierra nunca porque está bajando iconos de fondo. El plazo lo mueve quien
	// está delante, no el trabajo de fondo.
	a.sistema.Avisar(EventoIconos, nil)
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
