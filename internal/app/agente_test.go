package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/agente"
	"github.com/webcafeina/esfinge/internal/boveda"
)

// **Viene apagado de fábrica, y apagarlo lo apaga de verdad.**
//
// Lo segundo es lo que no es obvio: un interruptor que deja el socket escuchando hasta
// el siguiente arranque es un interruptor que miente, y en una puerta hacia la bóveda
// eso no se puede permitir.
func TestElCanalDeAgentesSeEnciendeYSeApagaDesdeAjustes(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)

	if e := a.EstadoDelAgente(); e.Encendido || e.Escuchando {
		t.Fatalf("viene encendido de fábrica: %+v", e)
	}

	p := a.VerPreferencias()
	p.CanalDeAgentes = true
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	e := a.EstadoDelAgente()
	if !e.Encendido || !e.Escuchando {
		t.Fatalf("no ha arrancado: %+v", e)
	}
	if !strings.HasSuffix(e.Donde, "agentes.sock") {
		t.Errorf("no dice dónde escucha: %q", e.Donde)
	}
	// **Y es otro socket que el del navegador**, que es media ADR 0054: con el mismo,
	// emparejar un agente le daría también los verbos de la extensión.
	if strings.HasSuffix(e.Donde, "puente.sock") {
		t.Error("los agentes están escuchando por el socket del navegador")
	}

	p.CanalDeAgentes = false
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelAgente(); e.Encendido || e.Escuchando {
		t.Errorf("sigue escuchando después de apagarlo: %+v", e)
	}
}

// El emparejamiento **se pide, se concede una vez, y el testigo no cruza el puente**.
func TestElAgenteSeEmparejaUnaVezYSuTestigoNoCruza(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}

	// Lo primero que pasa es que no hay permiso y se avisa a la ventana.
	if _, err := f.Emparejar("Claude Code"); err == nil {
		t.Fatal("ha emparejado sin que nadie dijera que sí")
	}
	if e := a.EstadoDelAgente(); e.Pide != "Claude Code" {
		t.Fatalf("la ventana no se entera de quién pide: %+v", e)
	}

	if err := a.PermitirAgente(); err != nil {
		t.Fatal(err)
	}
	testigo, err := f.Emparejar("Claude Code")
	if err != nil || testigo == "" {
		t.Fatalf("no ha recogido el testigo: %v", err)
	}
	if !f.Emparejado(testigo) {
		t.Error("el testigo que acaba de dar no vale")
	}

	// **Una sola vez.** Si se pudiera recoger cuando quisiera cualquiera, bastaría
	// pedir «emparejar» justo después de que la persona permitiera el suyo.
	if otro, err := f.Emparejar("Otro cualquiera"); err == nil {
		t.Errorf("el mismo permiso se ha podido recoger dos veces: %q", otro)
	}

	// **Y el testigo no cruza el puente**: es lo que abre la bóveda.
	e := a.EstadoDelAgente()
	if len(e.Permitidos) != 1 {
		t.Fatalf("la lista tiene %d: %+v", len(e.Permitidos), e.Permitidos)
	}
	if e.Permitidos[0].Testigo != "" {
		t.Error("el testigo ha cruzado hacia la ventana")
	}
	// Y se retira por su fecha, porque el testigo no está.
	if err := a.OlvidarAgente(e.Permitidos[0].Desde); err != nil {
		t.Fatal(err)
	}
	if f.Emparejado(testigo) {
		t.Error("sigue valiendo después de retirarlo")
	}
}

// **El agente no mantiene la bóveda abierta**, y es la regla que no se rompe: si lo que
// pide contara como actividad, un agente trabajando dejaría la bóveda abierta para
// siempre y el bloqueo por inactividad dejaría de significar lo que dice.
func TestElAgenteNoMantieneLaBovedaAbierta(t *testing.T) {
	a, _, ahora, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}

	// **El reloj se adelanta antes de pedir**, que es lo que hace que esto vigile algo:
	// si uno de estos métodos tocara el reloj, lo pondría en el minuto nuevo y la
	// bóveda no se cerraría nunca. Escrito al revés —adelantando después— la prueba
	// pasa **con la regla rota**, y se comprobó mutándola.
	for i := 0; i < 30; i++ {
		*ahora = ahora.Add(time.Minute)
		_ = f.Estado()
		_, _, _ = f.Buscar("")
		_, _ = f.Ver("lo-que-sea")
		_, _ = f.Higiene()
		_, _ = f.Generar(24, "hex")
		a.repasar()
	}
	if a.EstadoBoveda().Abierta {
		t.Error("la bóveda sigue abierta después de media hora sin nadie delante: " +
			"lo que pide un agente está tocando el reloj del bloqueo")
	}
}

// Lo que el agente ve de una entrada **no lleva secretos**, y las dos banderas dicen
// que los hay sin decir cuáles son.
//
// La de `TieneCodigo` existe por una trampa que ya costó una vez: `SinSecretos` vacía
// la semilla **sin dejar marca**, así que leerla sobre lo que devuelve `Buscar` da
// siempre falso, y todas las cuentas salían sin segundo factor.
func TestLoQueElAgenteVeDeUnaEntradaNoLlevaSecretos(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "GitHub", Usuario: "zeri",
		Secreto: "LA-CONTRASENA", TOTP: "JBSWY3DPEHPK3PXP",
	}); err != nil {
		t.Fatal(err)
	}
	f := fuenteDelAgente{a}
	es, cuantas, err := f.Buscar("GitHub")
	if err != nil || len(es) != 1 || cuantas != 1 {
		t.Fatalf("la búsqueda ha dado %d entradas: %v", len(es), err)
	}
	e := es[0]
	if !e.TieneSecreto || !e.TieneCodigo {
		t.Errorf("las banderas no dicen lo que hay: %+v", e)
	}
	// Y lo que sale, por los bytes.
	if contieneSecreto(t, e, "LA-CONTRASENA") || contieneSecreto(t, e, "JBSWY3DPEHPK3PXP") {
		t.Error("la entrada que ve el agente lleva un secreto dentro")
	}
}

// contieneSecreto mira **los bytes** de lo que saldría por el canal, no los campos:
// sobre los campos, uno nuevo con un secreto dentro pasaría sin que nadie lo viera.
func contieneSecreto(t *testing.T, e agente.Entrada, secreto string) bool {
	t.Helper()
	crudo, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Contains(string(crudo), secreto)
}

// **Una búsqueda sin filtro no vuelca la bóveda entera**, y dice cuántas hay.
//
// No es una optimización: una bóveda de dos mil entradas en el contexto de un modelo es
// la lista completa de sitios y usuarios de una persona —justo lo que la ADR 0024
// decidió cifrar en el disco— y ahí ya no la protege nadie. Lo que vuelve es una
// muestra **y el total**, que es lo que hace falta para poder decir cuántas hay sin
// enumerarlas.
func TestUnaBusquedaSinFiltroNoVuelcaLaBovedaEntera(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	cuantas := agente.TopeDeResultados + 12
	for i := 0; i < cuantas; i++ {
		if err := a.GuardarEnBoveda(boveda.Entrada{
			Tipo: boveda.TipoCredencial, Titulo: fmt.Sprintf("Cuenta %02d", i), Secreto: "x",
		}); err != nil {
			t.Fatal(err)
		}
	}
	es, total, err := fuenteDelAgente{a}.Buscar("Cuenta")
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != agente.TopeDeResultados {
		t.Errorf("han vuelto %d entradas y el tope son %d", len(es), agente.TopeDeResultados)
	}
	// **Y el total es el de verdad**, no el de lo que ha vuelto: sin esto, un agente
	// que recibe veinticinco creería que la bóveda tiene veinticinco.
	if total != cuantas {
		t.Errorf("dice que hay %d y hay %d", total, cuantas)
	}
}

// Lo que está en la papelera **no está** para el agente, aunque la bóveda lo encuentre.
func TestElAgenteNoVeLoQueEstaEnLaPapelera(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Para borrar", Secreto: "x",
	}); err != nil {
		t.Fatal(err)
	}
	lista, err := a.BuscarEnBoveda("Para borrar")
	if err != nil || len(lista) != 1 {
		t.Fatalf("no se ha guardado: %v", err)
	}
	id := lista[0].ID
	if err := a.BorrarDeBoveda(id); err != nil {
		t.Fatal(err)
	}
	f := fuenteDelAgente{a}
	if _, err := f.Ver(id); !errors.Is(err, agente.ErrNoEsta) {
		t.Errorf("una entrada de la papelera se puede ver: %v", err)
	}
	es, _, err := f.Buscar("Para borrar")
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 0 {
		t.Errorf("la búsqueda trae lo que está en la papelera: %+v", es)
	}
}
