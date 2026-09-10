package app

import (
	"errors"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
)

// La semilla de los vectores del RFC, para poder comprobar el código contra un
// número que no lo ha calculado este programa.
const semillaDelRFC = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

// El recorrido entero: una entrada con semilla dentro de una bóveda de verdad,
// pedida por el mismo método que usa la ventana, y el código comparado con el
// del RFC. No prueba el algoritmo —eso es cosa de internal/codigos— sino **que
// están conectados**: que lo que se guardó es lo que se lee y que el reloj que
// se usa es el que se cree.
func TestElCodigoDeUnaEntradaDeLaBoveda(t *testing.T) {
	a, _, ahora := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Titulo: "Banco", Usuario: "yo@ejemplo.es", TOTP: semillaDelRFC,
	}); err != nil {
		t.Fatal(err)
	}
	lista, err := a.BuscarEnBoveda("banco")
	if err != nil || len(lista) != 1 {
		t.Fatalf("la entrada no está: %v, %d", err, len(lista))
	}
	id := lista[0].ID

	// El segundo 59 es el primer vector de RFC 6238; con seis cifras, el código
	// es el mismo número cortado: 287082.
	*ahora = time.Unix(59, 0)
	c, err := a.CodigoDeBoveda(id)
	if err != nil {
		t.Fatal(err)
	}
	if c.Codigo != "287082" {
		t.Errorf("da %s y el RFC dice 287082", c.Codigo)
	}
	if c.Periodo != 30 || c.Quedan != 1 {
		t.Errorf("la cuenta atrás dice %d de %d", c.Quedan, c.Periodo)
	}

	// Un segundo después ya es otro intervalo, y otro código.
	*ahora = time.Unix(60, 0)
	otro, err := a.CodigoDeBoveda(id)
	if err != nil {
		t.Fatal(err)
	}
	if otro.Codigo == c.Codigo {
		t.Error("al saltar de intervalo sigue dando el mismo código")
	}
	if otro.Quedan != 30 {
		t.Errorf("al empezar el intervalo quedan %d segundos", otro.Quedan)
	}
}

// **Pedir el código no cuenta como actividad**, y es lo que separa un bloqueo
// por inactividad de un adorno: la ventana vuelve a pedirlo sola cada treinta
// segundos mientras la entrada esté abierta, así que si contara, una bóveda
// abierta encima de una mesa no se cerraría nunca.
func TestPedirElCodigoNoMantieneLaBovedaAbierta(t *testing.T) {
	a, _, ahora := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{Titulo: "Banco", TOTP: semillaDelRFC}); err != nil {
		t.Fatal(err)
	}
	lista, _ := a.BuscarEnBoveda("banco")
	id := lista[0].ID

	// Se pide el código una vez por minuto durante media hora, que es lo que
	// hace la ventana con una entrada delante. Que a mitad de camino empiece a
	// decir que está cerrada es justo lo que se quiere.
	for i := 0; i < 30; i++ {
		*ahora = ahora.Add(time.Minute)
		if _, err := a.CodigoDeBoveda(id); err != nil && !errors.Is(err, boveda.ErrCerrada) {
			t.Fatalf("vuelta %d: %v", i, err)
		}
		a.repasar()
	}
	if a.EstadoBoveda().Abierta {
		t.Error("la bóveda sigue abierta después de media hora sin nadie: " +
			"pedir el código está tocando el reloj del bloqueo")
	}
}

// Los tres noes: sin bóveda abierta, sin semilla y con una semilla rota. Los
// tres tienen que decir **cuál de los tres es**, porque desde la ventana no se
// distinguen y el arreglo de cada uno es distinto.
func TestLoQueNoTieneCodigo(t *testing.T) {
	a, _, _ := conReloj(t)

	if _, err := a.CodigoDeBoveda("lo que sea"); !errors.Is(err, boveda.ErrCerrada) {
		t.Errorf("con la bóveda cerrada dice: %v", err)
	}

	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	a.GuardarEnBoveda(boveda.Entrada{Titulo: "Sin semilla"})
	// Con un 0 y un 1 dentro, que son las dos cifras que el alfabeto base32 no
	// tiene. Una cadena de letras cualquiera **sí** es base32 válida, y ése es
	// precisamente el motivo por el que una semilla mal copiada no se detecta
	// aquí sino en la pantalla del servicio.
	a.GuardarEnBoveda(boveda.Entrada{Titulo: "Rota", TOTP: "10 no es base32"})

	lista, _ := a.BuscarEnBoveda("")
	porTitulo := map[string]string{}
	for _, e := range lista {
		porTitulo[e.Titulo] = e.ID
	}

	if _, err := a.CodigoDeBoveda(porTitulo["Sin semilla"]); err == nil {
		t.Error("una entrada sin semilla ha devuelto un código")
	}
	if _, err := a.CodigoDeBoveda(porTitulo["Rota"]); err == nil {
		t.Error("una semilla que no es base32 ha devuelto un código")
	}
	if _, err := a.CodigoDeBoveda("una entrada que no existe"); err == nil {
		t.Error("un identificador inventado ha devuelto un código")
	}
}
