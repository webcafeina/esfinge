package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/actualizacion"
)

// cadaCuanto se mira si hay versión nueva. Una vez al día: enterarse de una
// versión el mismo día que sale es de sobra, y así abrir y cerrar la ventana no
// son cinco peticiones.
const cadaCuanto = 24 * time.Hour

// cadaCuantoSeAsoma es cada cuánto el reloj pregunta **si toca** mirar. No es lo
// mismo que cadaCuanto y no hay que confundirlos: el reloj se asoma a menudo y
// quien decide sigue siendo la puerta de las 24 horas, así que asomarse más no
// significa pedir más.
//
// Hace falta porque hasta la 2.10.3 la comprobación ocurría **solo al arrancar**:
// quien dejaba Esfinge abierta no volvía a enterarse de nada, ni al día siguiente
// ni a la semana, mientras la portada prometía «una vez al día». Lo dijo el
// cliente, que nunca había visto la banda de aviso.
const cadaCuantoSeAsoma = time.Hour

// Nombres de los eventos de la actualización que viajan hasta la ventana.
const (
	// EventoNovedad llega cuando la comprobación del arranque ha terminado y
	// hay algo que contar. La comprobación va en su propia gorrutina para no
	// retrasar la ventana: si la red no contesta, no se nota.
	EventoNovedad = "novedad"
	// EventoDescarga lleva el avance de la descarga.
	EventoDescarga = "descarga"
)

// Novedad es lo que se le enseña a quien mira: hay una versión nueva, cuál, y
// qué se bajaría en esta máquina.
type Novedad struct {
	Hay     bool   `json:"hay"`
	Version string `json:"version"`
	Pagina  string `json:"pagina"`
	Fichero string `json:"fichero"`
	Bytes   int64  `json:"bytes"`
	// ComoSeInstala vale «sola» cuando Esfinge puede reemplazarse y reiniciarse
	// sin que nadie arrastre nada, e «instalador» cuando hace falta el del
	// sistema. El botón dice una cosa u otra según esto.
	ComoSeInstala string `json:"comoSeInstala"`
}

// actualizador reúne el estado de la comprobación, que vive aparte del resto.
type actualizador struct {
	mu          sync.Mutex
	comprobador *actualizacion.Comprobador
	// encontrada es lo último que se vio, para que la ventana pueda preguntarlo
	// aunque llegue tarde al evento.
	encontrada actualizacion.Novedad
	descargado string
}

// vigilar deja un reloj mirando si toca comprobar, mientras la ventana viva.
//
// **No basta con comprobar al arrancar**, que es lo que se hacía hasta la
// 2.10.3: una herramienta como ésta se deja abierta, y así no se enteraba nunca.
// El reloj se asoma cada poco y la puerta de las 24 horas es la que decide, de
// modo que esto no aumenta las peticiones: solo hace que la de cada día llegue
// también a quien no cierra la ventana.
//
// Se va con el contexto de Wails, que es el de la ventana.
func (a *App) vigilar(ctx context.Context, cada time.Duration) {
	go func() {
		reloj := time.NewTicker(cada)
		defer reloj.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-reloj.C:
				// Y se vuelve a preguntar: cuando las dos ramas están listas,
				// «select» elige al azar, así que el cierre de la ventana puede
				// perder varias veces seguidas frente al reloj. Con un reloj de
				// una hora no se notaría nunca; el código no debe depender de eso.
				if ctx.Err() != nil {
					return
				}
				a.mirarSiToca()
			}
		}
	}()
}

// mirarSiToca sale a mirar en segundo plano, si la puerta de las 24 horas deja.
//
// La llaman Arrancar y el reloj de vigilar. No devuelve nada y no bloquea:
// cuando termina, y solo si hay algo que decir, manda EventoNovedad. Va en
// minúscula porque no es algo que la ventana deba poder pedir: todo método
// exportado de *App cruza el puente.
func (a *App) mirarSiToca() {
	// Reservar y no solo preguntar: ver ReservarComprobacion. Preguntando, dos
	// vueltas del reloj pueden colarse las dos mientras la primera está en la red.
	if !a.ajustes.ReservarComprobacion(cadaCuanto) {
		return
	}
	go func() {
		n, err := a.act.comprobador.Mirar()
		if err != nil {
			// Un fallo de red no se le cuenta a nadie: no se ha pedido esto, es
			// una cortesía. Se anota la fecha igual para no reintentar en bucle
			// —con el reloj puesto, eso sería una petición cada hora—.
			a.ajustes.AnotarComprobacion("")
			return
		}
		a.ajustes.AnotarComprobacion(n.Version)
		if !n.Hay {
			return
		}
		a.act.mu.Lock()
		a.act.encontrada = n
		a.act.mu.Unlock()
		a.sistema.Avisar(EventoNovedad, deNovedad(n))
	}()
}

// ComprobarActualizacion mira ahora mismo, lo diga la fecha o no. Es el botón
// «Buscar ahora» de Ajustes.
func (a *App) ComprobarActualizacion() (Novedad, error) {
	n, err := a.act.comprobador.Mirar()
	if err != nil {
		return Novedad{}, fmt.Errorf("No se ha podido preguntar a GitHub: %w", err)
	}
	a.ajustes.AnotarComprobacion(n.Version)

	a.act.mu.Lock()
	a.act.encontrada = n
	a.act.mu.Unlock()
	return deNovedad(n), nil
}

// NovedadPendiente devuelve lo que encontró la comprobación del arranque, para
// que la ventana pueda preguntarlo al montarse sin depender de haber estado
// escuchando el evento.
func (a *App) NovedadPendiente() Novedad {
	a.act.mu.Lock()
	defer a.act.mu.Unlock()
	return deNovedad(a.act.encontrada)
}

// DescargarActualizacion trae el fichero y devuelve dónde ha quedado. El avance
// va por EventoDescarga.
func (a *App) DescargarActualizacion() (string, error) {
	a.act.mu.Lock()
	n := a.act.encontrada
	a.act.mu.Unlock()

	if !n.Hay {
		return "", fmt.Errorf("No hay ninguna versión nueva que descargar")
	}

	ruta, err := a.act.comprobador.Descargar(n, func(av actualizacion.Avance) {
		a.sistema.Avisar(EventoDescarga, av)
	})
	if err != nil {
		return "", err
	}

	a.act.mu.Lock()
	a.act.descargado = ruta
	a.act.mu.Unlock()
	return ruta, nil
}

// InstalarActualizacion pone la versión descargada en su sitio.
//
// Donde Esfinge puede reemplazarse sola, esto **cierra la ventana**: el cambiazo
// lo da un guion que espera a que este proceso muera y luego vuelve a abrir la
// aplicación. El cierre va con un respiro para que la llamada llegue a
// contestar; si se cerrara aquí mismo, la interfaz vería un error de puente roto
// en lugar de una actualización que va bien.
func (a *App) InstalarActualizacion() error {
	a.act.mu.Lock()
	ruta := a.act.descargado
	a.act.mu.Unlock()

	if ruta == "" {
		return fmt.Errorf("Todavía no hay nada descargado")
	}
	if err := actualizacion.Instalar(ruta); err != nil {
		return err
	}

	if actualizacion.ComoSeInstala() == actualizacion.ModoSolo {
		go func() {
			time.Sleep(400 * time.Millisecond)
			a.sistema.Cerrar()
		}()
	}
	return nil
}

// VerPreferencias son los ajustes tal como están guardados.
func (a *App) VerPreferencias() Preferencias { return a.ajustes.Ver() }

// GuardarPreferencias las cambia. La ventana manda el objeto entero.
func (a *App) GuardarPreferencias(p Preferencias) error { return a.ajustes.Guardar(p) }

// deNovedad recorta lo que sabe el paquete de actualización a lo que necesita la
// ventana. El resumen SHA256 no cruza el puente: no es asunto de la interfaz.
func deNovedad(n actualizacion.Novedad) Novedad {
	return Novedad{
		Hay:           n.Hay,
		Version:       n.Version,
		Pagina:        n.Pagina,
		Fichero:       n.Fichero,
		Bytes:         n.Bytes,
		ComoSeInstala: string(actualizacion.ComoSeInstala()),
	}
}
