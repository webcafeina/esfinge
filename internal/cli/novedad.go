package cli

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"

	"github.com/webcafeina/esfinge/internal/actualizacion"
	"github.com/webcafeina/esfinge/internal/app"
	"github.com/webcafeina/esfinge/internal/red"
	"github.com/webcafeina/esfinge/internal/salida"
)

// cadaCuanto se pregunta por versiones nuevas desde la línea de comandos. Es el
// mismo plazo que usa la ventana y se lleva la cuenta en el mismo fichero: si
// una acaba de mirar, la otra no vuelve a salir.
const cadaCuanto = 24 * time.Hour

// plazoDeCortesia es lo máximo que se espera al final por una respuesta que no
// se ha pedido. Pasado eso, se sale sin decir nada.
const plazoDeCortesia = 700 * time.Millisecond

// vigilante mira si hay versión nueva mientras el comando hace lo suyo.
//
// Cuatro cautelas, porque esto se mete en tuberías y en cron:
//
//   - Solo si la salida de error es un terminal. Redirigida a un fichero, nadie
//     va a leer el aviso y sí va a ensuciar un registro.
//   - Nunca por la salida estándar, que es la que se encadena.
//   - Nunca cambia el código de salida ni retrasa el resultado más allá del
//     plazo de cortesía.
//   - Una vez al día, y apagable.
type vigilante struct {
	hecho chan actualizacion.Novedad
	ajus  *app.Ajustes
}

// vigilar arranca la consulta en segundo plano, si procede. Devuelve nil cuando
// no hay nada que vigilar, y entonces contar() no hace nada.
func vigilar(version string) *vigilante {
	// ESFINGE_SIN_RED la apaga entera, para quien no quiera ni la posibilidad.
	// Es lo que se pone en una imagen de contenedor o en un servidor de compilación.
	//
	// Va por internal/red y no por un os.Getenv aquí: **antes esto era lo único que
	// la miraba**, así que quien la ponía creyendo que apagaba la red apagaba la
	// mitad —la ventana no la consultaba nunca—. Con dos salidas a la red eso deja
	// de ser un descuido pequeño.
	if red.SinRed() {
		return nil
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return nil
	}

	ajus := app.AbrirAjustes()
	if !ajus.TocaMirar(cadaCuanto) {
		return nil
	}

	v := &vigilante{hecho: make(chan actualizacion.Novedad, 1), ajus: ajus}
	go func() {
		n, err := actualizacion.Nuevo(version).Mirar()
		if err != nil {
			ajus.AnotarComprobacion("")
			v.hecho <- actualizacion.Novedad{}
			return
		}
		ajus.AnotarComprobacion(n.Version)
		v.hecho <- n
	}()
	return v
}

// contar escribe el aviso, si ha llegado a tiempo y hay algo que decir.
func (v *vigilante) contar(e salida.Estilos) {
	if v == nil {
		return
	}
	select {
	case n := <-v.hecho:
		if !n.Hay {
			return
		}
		fmt.Fprintln(os.Stderr, e.Info(fmt.Sprintf(
			"Hay una versión nueva de Esfinge, la %s: %s", n.Version, n.Pagina)))
	case <-time.After(plazoDeCortesia):
		// Que no conteste la red no puede hacer esperar a quien pidió cifrar algo.
	}
}
