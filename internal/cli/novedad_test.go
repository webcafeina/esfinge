package cli

import (
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/actualizacion"
	"github.com/webcafeina/esfinge/internal/salida"
	"github.com/webcafeina/esfinge/internal/tema"
)

// Lo que más importa de esta parte: que un binario que se usa en tuberías no se
// ponga a hacer peticiones por su cuenta.
//
// Los tests corren con la salida de error redirigida, así que este caso es
// exactamente el de un script o un cron. Si «vigilar» devolviera algo aquí,
// estaría saliendo a la red donde nadie se lo ha pedido.
func TestFueraDeUnTerminalNoSeSaleALaRed(t *testing.T) {
	if v := vigilar("2.0.3"); v != nil {
		t.Error("la salida de error no es un terminal y ha arrancado la consulta igual")
	}
}

func TestConEsfingeSinRedNoSeSaleNunca(t *testing.T) {
	t.Setenv("ESFINGE_SIN_RED", "1")
	if v := vigilar("2.0.3"); v != nil {
		t.Error("ESFINGE_SIN_RED=1 y ha arrancado la consulta")
	}
}

// contar sobre un vigilante nulo tiene que ser inofensivo: es el caso de todas
// las veces que no toca mirar, que son casi todas.
func TestContarSinVigilanteNoRevienta(t *testing.T) {
	var v *vigilante
	v.contar(salida.NuevosEstilos(tema.TemaOscuro))
}

// Si la red no contesta, el comando no se queda esperando: el plazo de cortesía
// es lo que separa «te aviso si puedo» de «te retengo».
func TestSiLaRedNoContestaNoSeEsperaIndefinidamente(t *testing.T) {
	v := &vigilante{hecho: make(chan actualizacion.Novedad)} // nadie va a escribir aquí

	empezo := time.Now()
	v.contar(salida.NuevosEstilos(tema.TemaOscuro))

	if tardo := time.Since(empezo); tardo > 3*plazoDeCortesia {
		t.Errorf("ha esperado %v, y el plazo es %v", tardo, plazoDeCortesia)
	}
}
