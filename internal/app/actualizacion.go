package app

import (
	"fmt"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/actualizacion"
)

// cadaCuanto se mira si hay versión nueva. Una vez al día: enterarse de una
// versión el mismo día que sale es de sobra, y así abrir y cerrar la ventana no
// son cinco peticiones.
const cadaCuanto = 24 * time.Hour

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

// comprobarAlArrancar sale a mirar en segundo plano, si toca.
//
// La llama Arrancar. No devuelve nada y no bloquea: cuando termina, y solo si
// hay algo que decir, manda EventoNovedad. Va en minúscula porque no es algo que
// la ventana deba poder pedir: todo método exportado de *App cruza el puente.
func (a *App) comprobarAlArrancar() {
	if !a.ajustes.TocaMirar(cadaCuanto) {
		return
	}
	go func() {
		n, err := a.act.comprobador.Mirar()
		if err != nil {
			// Un fallo de red no se le cuenta a nadie: no se ha pedido esto, es
			// una cortesía. Se anota la fecha igual para no reintentar en bucle.
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

// InstalarActualizacion entrega lo descargado al sistema. A partir de ahí manda
// el instalador de cada uno, no Esfinge.
func (a *App) InstalarActualizacion() error {
	a.act.mu.Lock()
	ruta := a.act.descargado
	a.act.mu.Unlock()

	if ruta == "" {
		return fmt.Errorf("Todavía no hay nada descargado")
	}
	return actualizacion.Instalar(ruta)
}

// VerPreferencias son los ajustes tal como están guardados.
func (a *App) VerPreferencias() Preferencias { return a.ajustes.Ver() }

// GuardarPreferencias las cambia. La ventana manda el objeto entero.
func (a *App) GuardarPreferencias(p Preferencias) error { return a.ajustes.Guardar(p) }

// deNovedad recorta lo que sabe el paquete de actualización a lo que necesita la
// ventana. El resumen SHA256 no cruza el puente: no es asunto de la interfaz.
func deNovedad(n actualizacion.Novedad) Novedad {
	return Novedad{
		Hay:     n.Hay,
		Version: n.Version,
		Pagina:  n.Pagina,
		Fichero: n.Fichero,
		Bytes:   n.Bytes,
	}
}
