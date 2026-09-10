package app

import (
	"context"
	"sync"
	"time"
)

// El bloqueo por inactividad y el borrado del portapapeles.
//
// Las dos cosas son relojes, y las dos están escritas con la misma lección
// aprendida a base de equivocarse en este mismo proyecto: **un temporizador
// único no vale**.
//
// La comprobación de actualizaciones se hizo así y estuvo rota desde la 2.1.0
// hasta la 2.10.4 sin que nadie lo notara. Aquí sería peor: un portátil que se
// suspende ocho horas tiene que aparecer **bloqueado** al despertar, y un
// `time.AfterFunc` a quince minutos no garantiza eso en ningún sistema. Lo que
// funciona es un tic corto que compara la hora contra la última actividad.

const (
	// cadaCuantoSeMira es el tic del reloj. Corto a propósito: lo que decide no
	// es el tic sino la comparación de horas, y así un despertar tras suspender
	// se nota enseguida.
	cadaCuantoSeMira = 15 * time.Second

	// bloqueoPorDefecto son los minutos sin tocar nada antes de cerrar la bóveda.
	// Lo de verdad lo dicen las preferencias; esto es con lo que se arranca antes
	// de leerlas y lo que usan las pruebas que no las tocan.
	bloqueoPorDefecto = minutosBloqueoPorDefecto * time.Minute

	// portapapelesPorDefecto es lo que tarda en borrarse un secreto copiado.
	portapapelesPorDefecto = segundosPortapapelesPorDefecto * time.Second
)

// EventoBloqueada avisa a la ventana de que la bóveda se ha cerrado sola.
const EventoBloqueada = "boveda-bloqueada"

// EventoPortapapeles dice cuántos segundos quedan para que se borre, o cero
// cuando ya se ha borrado. La cuenta atrás se enseña: un secreto en el
// portapapeles sin decir cuánto va a estar ahí es peor que no borrarlo.
const EventoPortapapeles = "portapapeles"

// vigilante guarda el estado de los dos relojes.
type vigilante struct {
	mu sync.Mutex

	ultimaActividad time.Time
	espera          time.Duration
	esperaCopiado   time.Duration

	// loCopiado es lo último que Esfinge puso en el portapapeles, y cuándo caduca.
	loCopiado string
	caducaEn  time.Time

	// ahora se puede sustituir en las pruebas. Sin esto habría que esperar
	// quince minutos de verdad para comprobar que bloquea.
	ahora func() time.Time
}

func nuevoVigilante() *vigilante {
	return &vigilante{
		espera:          bloqueoPorDefecto,
		esperaCopiado:   portapapelesPorDefecto,
		ahora:           time.Now,
		ultimaActividad: time.Now(),
	}
}

// aplicarPreferencias pone los dos relojes en hora.
//
// Se llama al arrancar y cada vez que se guardan los ajustes, porque los dos
// plazos tienen que valer **desde ya**: quien acaba de bajar el bloqueo a un
// minuto porque se va de la mesa no puede tener que reiniciar para que sirva.
func (a *App) aplicarPreferencias(p Preferencias) {
	a.vig.mu.Lock()
	defer a.vig.mu.Unlock()
	// «Nunca» viaja como -1 y aquí es una duración de cero, que es lo que los dos
	// relojes entienden como «no cuentes».
	a.vig.espera = duracion(p.MinutosParaBloquear, time.Minute)
	a.vig.esperaCopiado = duracion(p.SegundosDePortapapeles, time.Second)
}

func duracion(cuantos int, unidad time.Duration) time.Duration {
	if cuantos <= 0 {
		return 0
	}
	return time.Duration(cuantos) * unidad
}

// Actividad la llama la interfaz cuando alguien está usando la aplicación.
//
// Se llama con moderación desde JavaScript —una vez cada diez segundos como
// mucho— porque cruzar el puente en cada movimiento del ratón sería gastar por
// gastar.
func (a *App) Actividad() {
	a.vig.mu.Lock()
	a.vig.ultimaActividad = a.vig.ahora()
	a.vig.mu.Unlock()
}

// tocaBloquear dice si ha pasado el plazo sin actividad.
func (v *vigilante) tocaBloquear() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.espera <= 0 {
		return false // «nunca» es una opción legítima de Ajustes
	}
	return v.ahora().Sub(v.ultimaActividad) >= v.espera
}

// tocaBorrarPortapapeles dice si el secreto copiado ya ha caducado.
func (v *vigilante) tocaBorrarPortapapeles() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.loCopiado != "" && !v.ahora().Before(v.caducaEn)
}

// Copiar pone algo en el portapapeles y arma el borrado.
//
// **Arregla de paso un agujero que ya existía**: desde la 2.8.0, «Usar como
// clave» copia una contraseña generada en claro y nunca se borraba, cosa que
// `docs/seguridad.md` reconocía sin resolver.
func (a *App) Copiar(texto string) error {
	if err := a.sistema.PonerEnPortapapeles(texto); err != nil {
		return err
	}

	a.vig.mu.Lock()
	espera := a.vig.esperaCopiado
	ahora := a.vig.ahora()
	a.vig.ultimaActividad = ahora
	// Con el borrado apagado no se apunta lo copiado: lo que no se va a borrar no
	// hace falta recordarlo, y guardar el secreto en memoria «por si acaso» sería
	// justo lo contrario de lo que hace este fichero.
	if espera > 0 {
		a.vig.loCopiado = texto
		a.vig.caducaEn = ahora.Add(espera)
	} else {
		a.vig.loCopiado = ""
	}
	a.vig.mu.Unlock()

	// La cuenta atrás se enseña. Un secreto en el portapapeles sin decir cuánto
	// va a estar ahí es peor que no borrarlo: quien no lo sabe, no pega a tiempo.
	a.sistema.Avisar(EventoPortapapeles, int(espera/time.Second))
	return nil
}

// borrarPortapapelesSiSigueSiendoNuestro es el detalle que hace esto aceptable.
//
// **Nunca se pisa algo que la persona haya copiado después.** Si el portapapeles
// ya no contiene lo que pusimos, es que hay algo más reciente ahí, y borrarlo
// sería quitarle a alguien lo que acababa de copiar por proteger un secreto que
// ya no está.
func (a *App) borrarPortapapelesSiSigueSiendoNuestro() {
	a.vig.mu.Lock()
	nuestro := a.vig.loCopiado
	a.vig.mu.Unlock()
	if nuestro == "" {
		return
	}

	hay, err := a.sistema.LeerPortapapeles()
	if err == nil && hay == nuestro {
		_ = a.sistema.PonerEnPortapapeles("")
	}

	a.vig.mu.Lock()
	a.vig.loCopiado = ""
	a.vig.mu.Unlock()
	a.sistema.Avisar(EventoPortapapeles, 0)
}

// vigilarBoveda deja los dos relojes andando mientras viva la ventana.
//
// Igual que el de las actualizaciones: se vuelve a preguntar por el contexto
// dentro del tic, porque cuando las dos ramas de un `select` están listas Go
// elige al azar y el cierre puede perder varias veces seguidas.
func (a *App) vigilarBoveda(ctx context.Context, cada time.Duration) {
	go func() {
		reloj := time.NewTicker(cada)
		defer reloj.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-reloj.C:
				if ctx.Err() != nil {
					return
				}
				a.repasar()
			}
		}
	}()
}

// repasar es lo que hace el tic: mirar las dos cuentas y actuar.
func (a *App) repasar() {
	if a.vig.tocaBorrarPortapapeles() {
		a.borrarPortapapelesSiSigueSiendoNuestro()
	}
	// La bóveda se pide con `boveda()`, que la coge detrás del cerrojo: **esta
	// gorrutina no es la de la ventana**, y era justo el otro lado de la carrera.
	if b := a.boveda(); b != nil && a.vig.tocaBloquear() {
		b.Cerrar()
		a.sistema.Avisar(EventoBloqueada, nil)
	}
}
