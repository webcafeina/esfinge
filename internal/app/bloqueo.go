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
	bloqueoPorDefecto = 15 * time.Minute

	// portapapelesPorDefecto es lo que tarda en borrarse un secreto copiado.
	portapapelesPorDefecto = 30 * time.Second
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
		ahora:           time.Now,
		ultimaActividad: time.Now(),
	}
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
	a.vig.loCopiado = texto
	a.vig.caducaEn = a.vig.ahora().Add(a.esperaDePortapapeles())
	a.vig.ultimaActividad = a.vig.ahora()
	a.vig.mu.Unlock()
	return nil
}

func (a *App) esperaDePortapapeles() time.Duration { return portapapelesPorDefecto }

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
	if a.bov != nil && a.bov.Abierta() && a.vig.tocaBloquear() {
		a.bov.Cerrar()
		a.sistema.Avisar(EventoBloqueada, nil)
	}
}
