package app

import (
	"errors"
	"math"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/codigos"
)

// El código de un solo uso, calculado aquí y no en la ventana.
//
// Podría calcularlo el JavaScript: el algoritmo cabe en cuarenta líneas y la
// semilla ya cruza el puente cuando se abre una entrada. Se hace en Go por dos
// razones que conviene dejar escritas, porque la tentación de moverlo va a
// volver:
//
//   - **La semilla es el segundo factor entero.** Cuanto menos viva en el montón
//     del webview, mejor; y si algún día `VerDeBoveda` deja de mandarla —que
//     sería lo suyo—, este método ya está en su sitio y no hay nada que rehacer.
//   - **La línea de comandos también lo necesita**, y ahí no hay ventana. Escrito
//     una vez en `internal/codigos`, las dos caras dan el mismo código; escrito en
//     la interfaz, la segunda copia se escribiría en otro sitio y divergiría.

// CodigoDeUnSoloUso es el código que vale ahora mismo para una entrada.
type CodigoDeUnSoloUso struct {
	Codigo string `json:"codigo"`
	// Quedan son los segundos que le sobran de vida, para poder enseñar la
	// cuenta atrás sin volver a preguntar cada segundo.
	Quedan int `json:"quedan"`
	// Periodo es cuánto dura entero, que es lo que hace falta para dibujar la
	// barra: sin él no se sabe qué fracción queda.
	Periodo int `json:"periodo"`
}

// CodigoDeBoveda calcula el código de un solo uso de una entrada.
//
// **No cuenta como actividad, y eso no es un olvido.** La ventana lo vuelve a
// pedir sola cada vez que el código caduca, para que en pantalla siempre esté el
// que vale; si eso tocara el reloj del bloqueo, una entrada abierta encima de la
// mesa mantendría la bóveda abierta para siempre y el «se cierra a los quince
// minutos» dejaría de ser verdad. Es la misma regla que el goteo de los iconos.
// Quien la abrió ya avisó de que estaba ahí al pedir la entrada.
func (a *App) CodigoDeBoveda(id string) (CodigoDeUnSoloUso, error) {
	b := a.boveda()
	if b == nil {
		return CodigoDeUnSoloUso{}, boveda.ErrCerrada
	}
	e, hay := b.Ver(id)
	if !hay {
		return CodigoDeUnSoloUso{}, errors.New("Esa entrada ya no está en la bóveda")
	}
	if e.TOTP == "" {
		return CodigoDeUnSoloUso{}, errors.New("Esta entrada no tiene código de un solo uso")
	}

	s, err := codigos.Leer(e.TOTP)
	if err != nil {
		return CodigoDeUnSoloUso{}, err
	}

	// El reloj sale del vigilante y no de time.Now para que las pruebas puedan
	// pararlo: un código que cambia cada treinta segundos no se comprueba contra
	// un valor fijo si el reloj es el de la máquina.
	a.vig.mu.Lock()
	ahora := a.vig.ahora()
	a.vig.mu.Unlock()

	codigo, err := s.En(ahora)
	if err != nil {
		return CodigoDeUnSoloUso{}, err
	}
	// Los segundos que quedan se redondean **hacia arriba**: hacia abajo, medio
	// segundo de vida se convierte en un cero, la ventana se cree que ya ha
	// caducado y vuelve a preguntar en el acto, una y otra vez, hasta que el
	// reloj cruce el segundo. Un cero aquí es un bucle allí.
	return CodigoDeUnSoloUso{
		Codigo:  codigo,
		Quedan:  int(math.Ceil(s.Quedan(ahora).Seconds())),
		Periodo: int(s.Periodo.Seconds()),
	}, nil
}
