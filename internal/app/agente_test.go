package app

import (
	"bytes"
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
		_, _ = f.Generar(0, 24, "hex")
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

// **El camino de usar una contraseña: se pide, se aprueba, se copia.** Y lo que de
// verdad vigila esta prueba son las tres cosas que podrían salir mal y no se verían:
//
//  1. Que se copie **en vez de devolverse**.
//  2. Que el sí valga **solo para la entrada que se aprobó**.
//  3. Que valga **una sola vez**.
func TestCopiarUnaContrasenaSePideSeApruebaYSeCopia(t *testing.T) {
	a, s, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	elBanco := lista[0].ID
	otra, err := a.BuscarEnBoveda("Correo")
	if err != nil || len(otra) == 0 {
		t.Fatal(err)
	}
	elCorreo := otra[0].ID

	// --- Se pide y **no se copia nada**.
	if _, err := f.CopiarSecreto("Claude Code", elBanco); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatalf("la primera vez contesta %v", err)
	}
	if enElPortapapeles(s) != "" {
		t.Fatal("ha copiado antes de que nadie dijera que sí")
	}
	// Y la ventana se entera de qué se pide, **con el título**: sin él, la pregunta no
	// se puede contestar.
	e := a.EstadoDelAgente()
	if e.Quiere == nil || e.Quiere.Titulo != "Banco" {
		t.Fatalf("la ventana no sabe qué se pide: %+v", e.Quiere)
	}

	// --- Se aprueba, y entonces sí.
	if err := aprobarLoQueSeVe(f.a, false); err != nil {
		t.Fatal(err)
	}
	c, err := f.CopiarSecreto("Claude Code", elBanco)
	if err != nil {
		t.Fatalf("tras aprobarlo: %v", err)
	}
	// **Se ha copiado, no devuelto.**
	if enElPortapapeles(s) != "s3cr3t0" {
		t.Errorf("en el portapapeles hay %q", enElPortapapeles(s))
	}
	crudo, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(crudo), "s3cr3t0") {
		t.Errorf("lo que vuelve lleva la contraseña: %s", crudo)
	}

	// --- **El sí se ha gastado.** Pedirla otra vez vuelve a preguntar.
	if _, err := f.CopiarSecreto("Claude Code", elBanco); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("el mismo sí ha servido dos veces")
	}

	// --- **Y un sí para una entrada no vale para otra**, que es lo que impide que
	// aprobar «la de GitHub» se lleve la del banco.
	if err := aprobarLoQueSeVe(f.a, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CopiarSecreto("Claude Code", elCorreo); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("un sí dado para una entrada ha servido para otra")
	}
}

// **Lo que se le da y lo que se le niega quedan apuntados**, dentro de la bóveda.
func TestLoQueSeLeDaAUnAgenteQuedaApuntado(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	id := lista[0].ID

	// Un no.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := a.DenegarLoQuePideElAgente(); err != nil {
		t.Fatal(err)
	}
	// Y un sí.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := aprobarLoQueSeVe(f.a, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CopiarSecreto("Claude Code", id); err != nil {
		t.Fatal(err)
	}

	// **El registro se lee del estado**, que es por donde lo lee la ventana: pedirlo
	// aparte fallaría con la bóveda cerrada y dejaría un 400 en la consola.
	r := a.EstadoDelAgente().Dado
	if len(r) != 2 {
		t.Fatalf("hay %d apuntes y tenían que ser dos —el no y el sí—: %+v", len(r), r)
	}
	var hechos, negados int
	for _, ap := range r {
		if ap.Titulo != "Banco" || ap.Quien != "Claude Code" {
			t.Errorf("un apunte no dice de qué ni de quién: %+v", ap)
		}
		switch ap.Resultado {
		case boveda.ApunteHecho:
			hechos++
		case boveda.ApunteNegado:
			negados++
		}
	}
	if hechos != 1 || negados != 1 {
		t.Errorf("hay %d hechos y %d negados", hechos, negados)
	}
	// **Y en el registro no hay secretos**, que es lo que lo hace guardable.
	crudo, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(crudo), "s3cr3t0") {
		t.Errorf("el registro lleva la contraseña: %s", crudo)
	}
}

// enElPortapapeles lee lo que el doble tiene puesto, con su cerrojo.
func enElPortapapeles(s *sistemaFalso) string {
	t, _ := s.LeerPortapapeles()
	return t
}

// **La válvula: lo que deja hacer y, sobre todo, lo que no.**
//
// Es la única pieza de todo esto que **resta** seguridad, así que lo que esta prueba
// vigila son sus límites, uno por caso:
//
//  1. Que mientras está abierta no pregunte otra vez.
//  2. Que **se cierre sola al llegar al tope**, que es lo que impide que «cinco
//     minutos» sea un cheque en blanco.
//  3. Que **cortarla la cierre en el acto**.
//  4. Que **cambiar de bóveda la cierre**: se dio mirando una, no otra.
func TestLaValvulaYSusLimites(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	id := lista[0].ID

	abrirLaValvula := func() {
		t.Helper()
		if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
			t.Fatalf("no ha pedido aprobación: %v", err)
		}
		if err := aprobar(t, a, true); err != nil {
			t.Fatal(err)
		}
	}

	// --- 1. Abierta, no vuelve a preguntar.
	abrirLaValvula()
	for i := 0; i < 3; i++ {
		if _, err := f.CopiarSecreto("Claude Code", id); err != nil {
			t.Fatalf("con la válvula abierta, la vuelta %d pide aprobación: %v", i, err)
		}
	}
	if v := a.EstadoDelAgente().Valvula; !v.Abierta || v.Usadas != 3 {
		t.Errorf("la ventana dice %+v", v)
	}

	// --- 2. **Se cierra al llegar al tope.**
	for i := 3; i < TopeDeLaValvula; i++ {
		if _, err := f.CopiarSecreto("Claude Code", id); err != nil {
			t.Fatalf("la vuelta %d ha fallado antes del tope: %v", i, err)
		}
	}
	if v := a.EstadoDelAgente().Valvula; v.Abierta {
		t.Error("la válvula sigue abierta pasado el tope: «cinco minutos» sería un cheque en blanco")
	}
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("pasado el tope no vuelve a preguntar")
	}

	// --- 3. **Cortar la cierra en el acto.**
	abrirLaValvula()
	if err := a.CortarAlAgente(); err != nil {
		t.Fatal(err)
	}
	if v := a.EstadoDelAgente().Valvula; v.Abierta {
		t.Error("cortar no la ha cerrado")
	}
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("después de cortar sigue dando sin preguntar")
	}

	// --- 4. **Cambiar de bóveda la cierra**: lo concedido se dio mirando una.
	abrirLaValvula()
	a.CerrarBoveda()
	if v := a.EstadoDelAgente().Valvula; v.Abierta {
		t.Error("cerrar la bóveda no ha cerrado la válvula")
	}
}

// **Y lo que la válvula no cubre queda apuntado como tal.** Poder distinguir después lo
// que se aprobó mirando de lo que pasó en bloque es media razón de que el registro
// exista.
func TestElRegistroDistingueLaValvula(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	id := lista[0].ID

	// Uno preguntado.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := aprobarLoQueSeVe(f.a, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CopiarSecreto("Claude Code", id); err != nil {
		t.Fatal(err)
	}
	// Y uno por la válvula.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := aprobar(t, a, true); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CopiarSecreto("Claude Code", id); err != nil {
		t.Fatal(err)
	}

	// **El registro se lee del estado**, que es por donde lo lee la ventana: pedirlo
	// aparte fallaría con la bóveda cerrada y dejaría un 400 en la consola.
	r := a.EstadoDelAgente().Dado
	var preguntados, porValvula int
	for _, ap := range r {
		switch ap.Como {
		case boveda.ApuntePreguntado:
			preguntados++
		case boveda.ApunteValvula:
			porValvula++
		}
	}
	// Uno y uno: **pedir aprobación no deja apunte**, solo hacerlo o negarlo. Lo que
	// esto vigila es que los dos caminos se distingan después, no cuántos hay.
	if preguntados != 1 || porValvula != 1 {
		t.Errorf("el registro dice %d preguntados y %d por la válvula: %+v", preguntados, porValvula, r)
	}
}

// **La válvula no cubre el código de un solo uso**, y ésta es la prueba que lo dice.
//
// Es la línea entera de la ADR 0054: la válvula vale para **actuar** —el secreto se
// queda en este equipo y el portapapeles se borra solo— y no para **enseñar**, porque
// enseñar no es reversible: en cuanto las seis cifras entran en el contexto del modelo
// están en su transcripción.
//
// Sin esta prueba, bastaría con que alguien pusiera `true` donde hay un `false` para
// que la válvula empezara a soltar códigos en ráfaga, y nada se pondría rojo.
func TestLaValvulaNoCubreElCodigo(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	id := lista[0].ID // el Banco tiene TOTP

	// Se abre la válvula del todo, con una copia.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := aprobar(t, a, true); err != nil {
		t.Fatal(err)
	}
	// Copiar ya no pregunta...
	if _, err := f.CopiarSecreto("Claude Code", id); err != nil {
		t.Fatalf("con la válvula abierta, copiar pide aprobación: %v", err)
	}
	// ...**y el código sí**.
	if _, err := f.Codigo("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal("la válvula ha soltado un código: lo que se le enseña tiene que preguntar siempre")
	}

	// Y aprobándolo, llega — con sus segundos.
	if err := aprobarLoQueSeVe(f.a, false); err != nil {
		t.Fatal(err)
	}
	c, err := f.Codigo("Claude Code", id)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Codigo) != 6 {
		t.Errorf("el código es %q", c.Codigo)
	}
	if c.Quedan <= 0 {
		t.Errorf("no dice cuánto le queda: %d", c.Quedan)
	}

	// **Y el siguiente vuelve a preguntar**, aunque la válvula siga abierta.
	if _, err := f.Codigo("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("el segundo código ha salido sin preguntar")
	}
}

// Y una entrada sin segundo factor lo dice, en vez de dar un código inventado.
func TestSinSegundoFactorNoHayCodigo(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	lista, err := a.BuscarEnBoveda("Correo") // ésta no tiene TOTP
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	if _, err := (fuenteDelAgente{a}).Codigo("Claude Code", lista[0].ID); err == nil {
		t.Fatal("ha dado un código de una entrada que no tiene")
	} else if !strings.Contains(err.Error(), "no tiene código") {
		t.Errorf("lo que dice no ayuda: %v", err)
	}
}

// **Editar parchea sobre lo que hay, y no se lleva por delante lo que no entiende.**
//
// Es la trampa que `ActualizarCuenta` esquiva desde la fase 2: construir una `Entrada`
// nueva con lo que mandan borra `Extra` —los campos que escribió una versión más nueva
// de Esfinge— y todo lo que no se nombre. Una Esfinge vieja editando el título de una
// entrada dejaría la bóveda sin lo que la nueva guardó, **sin un error en ninguna
// parte**.
func TestEditarParcheaYNoPierdeLoQueNoEntiende(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Correo")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	id := lista[0].ID

	// Se le mete a mano un campo que esta versión no conoce, como haría una más nueva.
	b := a.boveda()
	e, _ := b.Ver(id)
	e.Extra = map[string]json.RawMessage{"loQueVieneDespues": json.RawMessage(`"no se toca"`)}
	if err := b.Poner(e); err != nil {
		t.Fatal(err)
	}

	// Y se edita **solo el título**.
	esc, err := f.Editar("Claude Code", agente.Peticion{ID: id, Campos: map[string]string{"titulo": "Correo nuevo"}})
	if err != nil {
		t.Fatal(err)
	}
	if esc.Titulo != "Correo nuevo" {
		t.Errorf("no ha cambiado el título: %+v", esc)
	}

	despues, hay := b.Ver(id)
	if !hay {
		t.Fatal("la entrada ha desaparecido")
	}
	// **Lo que no se nombró sigue ahí**: el usuario, el secreto y lo que no entendemos.
	if despues.Usuario != "otro@ejemplo.es" {
		t.Errorf("se ha perdido el usuario: %q", despues.Usuario)
	}
	if despues.Secreto != "otra clave" {
		t.Errorf("se ha perdido la contraseña: %q", despues.Secreto)
	}
	if string(despues.Extra["loQueVieneDespues"]) != `"no se toca"` {
		t.Errorf("se ha perdido lo que esta versión no entiende: %+v", despues.Extra)
	}
}

// **Cambiar un secreto pregunta; cambiar el título, no.**
//
// Las escrituras del navegador están acotadas por el sitio de la pestaña, y un agente no
// tiene sitio: sin esta puerta podría reescribir la contraseña del banco sin que nadie
// preguntara.
func TestCambiarUnSecretoPreguntaYElTituloNo(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	id := lista[0].ID

	// El título va directo.
	if _, err := f.Editar("Claude Code", agente.Peticion{ID: id, Campos: map[string]string{"titulo": "Mi banco"}}); err != nil {
		t.Fatalf("cambiar el título ha pedido aprobación: %v", err)
	}
	// La contraseña, no.
	_, err = f.Editar("Claude Code", agente.Peticion{ID: id, Campos: map[string]string{"secreto": "otra"}})
	if !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatalf("cambiar la contraseña no ha pedido aprobación: %v", err)
	}
	// **Y la válvula tampoco lo cubre**, por lo mismo que el código: dejar una cuenta
	// sin forma de entrar no se deshace como una copia.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := aprobar(t, a, true); err != nil {
		t.Fatal(err)
	}
	_, err = f.Editar("Claude Code", agente.Peticion{ID: id, Campos: map[string]string{"secreto": "otra"}})
	if !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("la válvula ha dejado cambiar una contraseña sin preguntar")
	}

	// Aprobándolo sí, **y la anterior queda en el historial**.
	if err := aprobarLoQueSeVe(f.a, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Editar("Claude Code", agente.Peticion{ID: id, Campos: map[string]string{"secreto": "otra"}}); err != nil {
		t.Fatal(err)
	}
	e, _ := a.boveda().Ver(id)
	if e.Secreto != "otra" {
		t.Errorf("no la ha cambiado: %q", e.Secreto)
	}
	if len(e.Historial) == 0 {
		t.Error("la contraseña anterior no ha caído al historial")
	}
}

// Crear con `generar`: **la contraseña la hace Esfinge y no vuelve**.
func TestCrearConGenerarNoDevuelveLaContrasena(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	esc, err := fuenteDelAgente{a}.Crear("Claude Code", agente.Peticion{
		Campos: map[string]string{"titulo": "Cuenta nueva", "usuario": "yo"}, Generar: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	crudo, err := json.Marshal(esc)
	if err != nil {
		t.Fatal(err)
	}
	e, hay := a.boveda().Ver(esc.ID)
	if !hay || e.Secreto == "" {
		t.Fatal("no ha guardado ninguna contraseña")
	}
	if strings.Contains(string(crudo), e.Secreto) {
		t.Errorf("la contraseña generada ha vuelto hacia el agente: %s", crudo)
	}
}

// **Una llave de acceso no se crea a mano** (ADR 0048): la emite el sitio, y lo que se
// guardara aquí no abriría ninguna cuenta.
func TestUnAgenteNoCreaLlavesDeAcceso(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	_, err := fuenteDelAgente{a}.Crear("Claude Code", agente.Peticion{
		Campos: map[string]string{"titulo": "Inventada", "tipo": "llave"},
	})
	if !errors.Is(err, agente.ErrNoSeEscribe) {
		t.Fatalf("ha dejado crear una llave: %v", err)
	}
}

// **Apuntar algo en el registro avisa a la ventana**, y sin eso el registro no se ve.
//
// El apunte no se escribe cuando se aprueba: se escribe cuando el agente **vuelve a
// pedirlo**, un instante después, y para entonces la ventana ya ha preguntado por el
// estado. Y como las secciones no se desmontan, ni cambiar de pestaña ni salir de
// Ajustes vuelven a preguntar: «Lo que les has dado» se quedaba vacío hasta reiniciar
// Esfinge. Lo vio el cliente con la 2.46.0, dando la primera contraseña de verdad.
//
// Es la regla de la casa con otra cara: **avisar a la ventana es que la ventana va a
// preguntar**, y lo que no se avisa no se pregunta nunca.
func TestDarleAlgoAUnAgenteAvisaALaVentana(t *testing.T) {
	a, s, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}

	if _, err := f.CopiarSecreto("Claude", lista[0].ID); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := aprobarLoQueSeVe(f.a, false); err != nil {
		t.Fatal(err)
	}
	// **Aprobar no apunta nada**, y eso es justo lo que hacía que no se viera.
	if s.hanAvisadoDe(EventoAgenteApuntado) {
		t.Fatal("ha avisado de un apunte y todavía no se le ha dado nada")
	}

	if _, err := f.CopiarSecreto("Claude", lista[0].ID); err != nil {
		t.Fatal(err)
	}
	if len(a.EstadoDelAgente().Dado) == 0 {
		t.Fatal("no se ha apuntado lo que se le dio")
	}
	if !s.hanAvisadoDe(EventoAgenteApuntado) {
		t.Error("se apuntó y no se avisó a la ventana: el registro no se vería hasta reiniciar")
	}
}

// **El «siempre» cubre lo mismo que la válvula y ni un verbo más** (2026-10-08).
//
// Es el tercer botón, y la línea que no puede moverse: lo reversible pasa —copiar una
// contraseña, o borrar, que va a la papelera— y **el código de un solo uso y cambiar un
// secreto siguen preguntando uno a uno**, porque enseñar no se deshace y dejar una
// cuenta sin forma de entrar tampoco.
//
// El cliente eligió ese alcance con la alternativa delante. Sin esta prueba, bastaría
// con que alguien pasara `true` donde va `porValvula` para que un permiso permanente
// empezara a soltar códigos, y nada se pondría rojo.
func TestElSiempreNoCubreLoQueNoSeDeshace(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(lista) == 0 {
		t.Fatal(err)
	}
	id := lista[0].ID

	// Se pide una contraseña y se concede «para siempre».
	if _, err := f.CopiarSecreto("Claude", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := aprobarSiempre(t, a); err != nil {
		t.Fatal(err)
	}

	// **Las contraseñas pasan solas**, y más de una vez: eso es lo que se pidió.
	for i := 0; i < 3; i++ {
		if _, err := f.CopiarSecreto("Claude", id); err != nil {
			t.Fatalf("con el «siempre» puesto, copiar la vez %d tenía que pasar: %v", i+1, err)
		}
	}

	// **Y el código sigue preguntando.** Es la frontera entera.
	if _, err := f.Codigo("Claude", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Errorf("el «siempre» ha soltado un código sin preguntar: %v", err)
	}
	a.DenegarLoQuePideElAgente()

	// **Y borrar sí pasa**, igual que con la válvula: lo eligió el cliente con la
	// alternativa delante, porque la entrada va a la papelera y se puede recuperar
	// treinta días. Lo que decide la frontera no es si molesta, es si se deshace.
	if _, err := f.Borrar("Claude", id); err != nil {
		t.Errorf("borrar tenía que pasar con el «siempre», igual que con la válvula: %v", err)
	}

	// Y el registro distingue de dónde salió el permiso, que es lo único que queda
	// cuando nadie mira en el momento.
	var porSiempre int
	for _, ap := range a.EstadoDelAgente().Dado {
		if ap.Como == boveda.ApunteSiempre {
			porSiempre++
		}
	}
	if porSiempre < 3 {
		t.Errorf("el registro apunta %d por el «siempre» y hubo al menos 3", porSiempre)
	}
}

// **Y se puede quitar, que es lo que hace aceptable darlo.**
//
// Un permiso permanente sin forma de revocarlo no se da. Y tiene que valer **desde
// ya**: quitarlo y que siguiera abierto hasta reiniciar sería un botón que miente, que
// es justo lo que esta casa no deja pasar con los interruptores.
func TestElSiempreSeQuitaYValeDesdeYa(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	lista, _ := a.BuscarEnBoveda("Banco")
	id := lista[0].ID

	if _, err := f.CopiarSecreto("Claude", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := aprobarSiempre(t, a); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CopiarSecreto("Claude", id); err != nil {
		t.Fatal(err)
	}
	// **La ventana lo tiene que poder decir**: sin esto no hay nada que enseñar y el
	// permiso se olvida.
	e := a.EstadoDelAgente()
	if !e.Siempre || e.SiempreQuien == "" {
		t.Errorf("el estado no dice que hay un permiso permanente: %+v", e)
	}

	if err := a.QuitarElSiempreAlAgente(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CopiarSecreto("Claude", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Errorf("se quitó el «siempre» y sigue dando contraseñas: %v", err)
	}
	if a.EstadoDelAgente().Siempre {
		t.Error("el estado sigue diciendo que está puesto")
	}
}

// **El estado dice en qué bóveda se trabaja, también cuando es la personal.**
//
// Antes se decía **omitiendo el campo**, y eso no lo dice: un agente ve que falta, no
// que sea la de siempre. La descripción de la herramienta prometía decirlo, así que
// desde fuera parecía incumplida — lo señaló el agente del cliente el 2026-10-13.
//
// Y lo que esto protege de verdad: un agente que no sepa si está dentro de un proyecto
// puede guardar la cuenta de un cliente en la bóveda de otro.
// **Y dice con qué Esfinge se está hablando**, que es lo único que un agente no puede
// averiguar de ninguna otra forma.
//
// El paquete `.mcpb` lleva el servidor dentro y **no se actualiza con la aplicación**:
// lo que el cliente MCP enseña en su lista de extensiones es la versión del paquete que
// alguien arrastró, y desde la 2.48.0 ése le cede el turno al binario de la Esfinge
// instalada. Así que el número de la pantalla y el que de verdad contesta son distintos
// a propósito, y hasta la 2.55.0 **no había forma de preguntar por el segundo**. Lo
// preguntó el cliente el 2026-10-14: «¿cómo sé que se ha actualizado?».
//
// Y de paso es el diagnóstico del relevo: si no encuentra la aplicación y contesta el
// binario viejo de dentro del paquete, aquí se ve — en vez de manifestarse como una
// herramienta que «no existe», que es un síntoma que no se parece a su causa.
func TestElEstadoDelAgenteDiceQueVersionContesta(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}

	// La de **esta** aplicación, no una constante del paquete del canal: con una
	// constante diría siempre lo mismo y no distinguiría quién contesta, que es justo
	// para lo que existe.
	if hace, quiero := f.Estado().Version, a.Version(); hace != quiero {
		t.Errorf("dice la versión %q y esta Esfinge es la %q", hace, quiero)
	}

	crudo, err := json.Marshal(f.Estado())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(crudo, []byte(`"version":"2.0.0"`)) {
		t.Errorf("la versión no sale en lo que recibe el agente: %s", crudo)
	}

	// **Y sale aunque esté vacía, que es cuando de verdad hace falta.**
	//
	// Es la misma trampa que ya se tragó `boveda`: con `omitempty` el campo desaparece
	// justo en el caso raro, y lo que el agente ve entonces no es «no se sabe», es
	// **nada** — y de una ausencia no se puede concluir nada, que es literalmente lo
	// que el agente del cliente dijo el 2026-10-13.
	//
	// Y esta aserción hay que escribirla así: con la versión puesta, `omitempty` no
	// borra nada y **la prueba pasa con él**. Comprobado poniéndolo.
	enConfiguracionDePruebas(t)
	sinVersion := fuenteDelAgente{Nueva("", &sistemaFalso{})}
	crudo, err = json.Marshal(sinVersion.Estado())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(crudo, []byte(`"version":""`)) {
		t.Errorf("una versión vacía tiene que decirse vacía, no omitirse: %s", crudo)
	}
}

func TestElEstadoDelAgenteDiceEnQueBovedaSeTrabaja(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}

	e := f.Estado()
	if !e.Abierta {
		t.Fatal("la bóveda tendría que estar abierta")
	}
	if !e.EsLaPersonal {
		t.Error("en la personal tiene que decir que lo es, no callarse")
	}

	// **Y sale por el canal**, que es lo que de verdad falló: el campo existía y
	// `omitempty` lo borraba al serializar.
	crudo, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, hace := range []string{`"boveda"`, `"esLaPersonal"`} {
		if !bytes.Contains(crudo, []byte(hace)) {
			t.Errorf("falta %s en lo que sale: %s", hace, crudo)
		}
	}
}

// **Al generar se dice cómo es la contraseña, nunca cuál.**
//
// Sin esto, un agente que pide `generar: true` no puede decir si lo guardado cumple lo
// que exige el sitio, y lo único que le queda es pedir que se la copien al portapapeles
// para mirarla — justo lo que este canal existe para evitar.
//
// Y la prueba mira **los bytes**: lo que no puede pasar es que por querer describirla
// se cuele la contraseña.
func TestAlGenerarSeDiceComoEsYNoCual(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}

	hecho, err := f.Crear("Claude", agente.Peticion{
		Campos:  map[string]string{"titulo": "prueba-generar.ejemplo.com"},
		Generar: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hecho.Generada == nil {
		t.Fatal("se generó la contraseña y no se dice cómo es")
	}
	if hecho.Generada.Alfabeto != "hex" {
		t.Errorf("el alfabeto es %q y tiene que ser hex: una contraseña con «/» dentro de "+
			"una cadena de conexión la parte", hecho.Generada.Alfabeto)
	}
	if hecho.Generada.Largo < 32 {
		t.Errorf("dice que mide %d, que no llega para la política de ningún sitio", hecho.Generada.Largo)
	}

	// Y lo guardado de verdad mide eso: decirlo sin que sea cierto es peor que callar.
	b := a.boveda()
	entera, hay := b.Ver(hecho.ID)
	if !hay {
		t.Fatal("no se ha guardado")
	}
	if len(entera.Secreto) != hecho.Generada.Largo {
		t.Errorf("dice %d y la contraseña mide %d", hecho.Generada.Largo, len(entera.Secreto))
	}
	crudo, err := json.Marshal(hecho)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(crudo, []byte(entera.Secreto)) {
		t.Error("la contraseña ha salido en lo que se le devuelve al agente")
	}
}

// **Lo que un agente escribe en la bóveda avisa a la ventana**, como lo que escribe el
// navegador.
//
// Sin esto la bóveda queda bien y **la pantalla sigue enseñando lo de antes**: el
// cliente guardó una entrada en la bóveda de un proyecto, el agente dijo la verdad —que
// estaba guardada— y él no la veía. Es el fallo del registro otra vez con otra cara, y
// el evento existía desde el canal del navegador, que lo manda en ocho sitios.
//
// Se prueban **los tres verbos que escriben**: olvidarse de uno es tan fácil como
// olvidarse de los tres, y la última vez se olvidó justo el que no se había pensado.
func TestLoQueEscribeUnAgenteAvisaALaVentana(t *testing.T) {
	for _, c := range []struct {
		que   string
		hacer func(f fuenteDelAgente, id string) error
	}{
		{"crear", func(f fuenteDelAgente, _ string) error {
			_, err := f.Crear("Claude", agente.Peticion{Campos: map[string]string{"titulo": "Nueva"}})
			return err
		}},
		{"editar", func(f fuenteDelAgente, id string) error {
			_, err := f.Editar("Claude", agente.Peticion{ID: id, Campos: map[string]string{"notas": "x"}})
			return err
		}},
		{"borrar", func(f fuenteDelAgente, id string) error {
			if _, err := f.Borrar("Claude", id); !errors.Is(err, agente.ErrPideAprobacion) {
				return err
			}
			if err := aprobarLoQueSeVe(f.a, false); err != nil {
				return err
			}
			_, err := f.Borrar("Claude", id)
			return err
		}},
	} {
		t.Run(c.que, func(t *testing.T) {
			a, s, _, _, _ := conBoveda(t)
			f := fuenteDelAgente{a}
			lista, err := a.BuscarEnBoveda("Banco")
			if err != nil || len(lista) == 0 {
				t.Fatal(err)
			}
			if s.hanAvisadoDe(EventoBovedaCambiada) {
				t.Fatal("ha avisado antes de que nadie escriba")
			}
			if err := c.hacer(f, lista[0].ID); err != nil {
				t.Fatal(err)
			}
			if !s.hanAvisadoDe(EventoBovedaCambiada) {
				t.Errorf("%s no avisa: la bóveda cambia y la pantalla sigue con lo de antes", c.que)
			}
		})
	}
}

// **Al crear se puede pedir el alfabeto, y sin pedirlo sale hexadecimal.**
//
// Lo señaló el agente del cliente: `esfinge_generar` dejaba elegirlo y `crear` no, así
// que para un sitio que exija símbolos había que generar la contraseña aparte y pasarla
// — o sea, hacerla cruzar el chat, que es justo lo que este canal evita.
//
// Y el defecto se queda en hexadecimal, que no es una preferencia: una contraseña con
// `/`, `+` o `@` dentro de una cadena de conexión la parte, y el error que sale no
// menciona la contraseña por ninguna parte.
func TestAlCrearSePuedeElegirElAlfabetoYPorDefectoEsHex(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}

	sinDecir, err := f.Crear("Claude", agente.Peticion{
		Campos: map[string]string{"titulo": "sin-decir.ejemplo.com"}, Generar: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sinDecir.Generada == nil || sinDecir.Generada.Alfabeto != "hex" {
		t.Errorf("sin pedir alfabeto tiene que salir hex: %+v", sinDecir.Generada)
	}

	conSimbolos, err := f.Crear("Claude", agente.Peticion{
		Campos:  map[string]string{"titulo": "con-simbolos.ejemplo.com"},
		Generar: true, Alfabeto: "simbolos",
	})
	if err != nil {
		t.Fatal(err)
	}
	if conSimbolos.Generada == nil || conSimbolos.Generada.Alfabeto != "simbolos" {
		t.Fatalf("se pidió «simbolos» y dice %+v", conSimbolos.Generada)
	}
	// Y lo guardado sale de ese alfabeto de verdad: decirlo sin que lo sea es peor que
	// no decirlo.
	b := a.boveda()
	entera, hay := b.Ver(conSimbolos.ID)
	if !hay {
		t.Fatal("no se ha guardado")
	}
	soloHex := true
	for _, r := range entera.Secreto {
		if !strings.ContainsRune("0123456789abcdef", r) {
			soloHex = false
		}
	}
	if soloHex {
		t.Errorf("se pidió «simbolos» y la contraseña guardada solo tiene cifras hexadecimales")
	}
}

// **Y la bóveda se dice con un nombre, también la personal.**
//
// La cadena vacía no es un nombre: obligaba a mirar la bandera para saber qué se estaba
// diciendo. Los dos campos siguen, que es la separación de siempre en este protocolo —
// **el nombre es para leer y la bandera es para decidir**, porque un proyecto podría
// llamarse «Personal».
func TestLaBovedaSeDiceConUnNombreTambienLaPersonal(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	if e := f.Estado(); e.Boveda != "Personal" || !e.EsLaPersonal {
		t.Errorf("en la personal el estado dice %+v", e)
	}
	hecho, err := f.Crear("Claude", agente.Peticion{Campos: map[string]string{"titulo": "X"}})
	if err != nil {
		t.Fatal(err)
	}
	if hecho.Boveda != "Personal" || !hecho.EsLaPersonal {
		t.Errorf("lo escrito dice %q / %v", hecho.Boveda, hecho.EsLaPersonal)
	}
}

// **El largo que el esquema promete es el que sale de verdad.**
//
// Acabo de escribir ese número en dos sitios —`largoDeLoQueGenera` en el código y la frase
// del esquema que el modelo lee— y eso es la familia de fallos que ya ha costado tres
// veces. Si alguien toca el uno sin el otro, el modelo promete un largo que no es.
//
// Lo que no se puede hacer es comparar la frase con otra frase: hay que **generar de
// verdad** con cada alfabeto y contar.
func TestElLargoQueElEsquemaPrometeEsElQueSale(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	for _, c := range []struct {
		alfabeto string
		dice     int
	}{
		{"hex", 48}, {"alnum", 33}, {"simbolos", 30},
	} {
		clave, err := a.GenerarContrasena(largoDeLoQueGenera, c.alfabeto)
		if err != nil {
			t.Fatal(err)
		}
		if len(clave) != c.dice {
			t.Errorf("con %s salen %d caracteres y el esquema dice %d",
				c.alfabeto, len(clave), c.dice)
		}
		// Y el número tiene que estar escrito en la frase que lee el modelo.
		if !strings.Contains(agente.LargoDeLoGenerado(), fmt.Sprint(c.dice)) {
			t.Errorf("el esquema no dice los %d caracteres de %s", c.dice, c.alfabeto)
		}
	}
}

// **Una petición nueva no puede pisar la que alguien está leyendo** (2026-10-19).
//
// `pedir` hacía `p.pendiente = &q` sin mirar si ya había una esperando, y
// `AprobarLoQuePideElAgente` concedía **lo que hubiera**, sin decir qué. Entre las dos
// cosas, si entre que lees la tarjeta y pulsas llegaba otra petición, **aprobabas la
// nueva creyendo que aprobabas la que viste**.
//
// Importa más desde que puede haber dos agentes a la vez —Claude Desktop y Claude Code—,
// y es la misma regla que ya obligó a poner la bóveda en la tarjeta: **aprobar sin saber
// qué es aprobar a ciegas**.
func TestUnaPeticionNuevaNoPisaLaQueSeEstaLeyendo(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	es, _ := a.BuscarEnBoveda("Banco")
	banco := es[0].ID
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Hacienda", Secreto: "otra",
	}); err != nil {
		t.Fatal(err)
	}
	otras, _ := a.BuscarEnBoveda("Hacienda")
	hacienda := otras[0].ID

	// El primer agente pide la del Banco, y eso es lo que la ventana enseña.
	if _, err := f.CopiarSecreto("Claude Desktop", banco); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal("tendría que pedir aprobación")
	}
	loQueSeVe := a.EstadoDelAgente().Quiere
	if loQueSeVe == nil || loQueSeVe.Titulo != "Banco" {
		t.Fatalf("la ventana no está enseñando la del Banco: %+v", loQueSeVe)
	}

	// Y mientras alguien la lee, **otro** agente pide otra cosa: espera su turno.
	_, err := f.CopiarSecreto("Claude Code", hacienda)
	if !errors.Is(err, agente.ErrOtraEsperando) {
		t.Fatalf("el segundo agente tendría que esperar su turno, y dice: %v", err)
	}

	// **Lo que se enseña sigue siendo lo que se enseñaba.** Si ha cambiado, la persona
	// está mirando una cosa y el botón hará otra.
	ahora := a.EstadoDelAgente().Quiere
	if ahora == nil || ahora.Titulo != "Banco" {
		t.Errorf("la tarjeta ha cambiado debajo: enseñaba «Banco» y ahora %+v", ahora)
	}

	// Y al aprobar se concede **lo que se estaba viendo**, no lo último que llegó.
	if err := aprobarLoQueSeVe(f.a, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CopiarSecreto("Claude Desktop", banco); err != nil {
		t.Errorf("se aprobó la del Banco y no se puede copiar: %v", err)
	}
	// La de Hacienda, en cambio, sigue sin permiso: nadie la aprobó.
	if _, err := f.CopiarSecreto("Claude Code", hacienda); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("ha copiado la de Hacienda, que nadie aprobó")
	}
}

// **Pero un agente no espera turno contra sí mismo**, y esa frontera es la decisión fina
// del arreglo de arriba.
//
// Un agente que pide otra cosa es el mismo interlocutor cambiando de tema; bloquearlo
// lo dejaría dos minutos sin poder trabajar porque nadie contestó lo de antes. A quien
// está leyendo lo protege la referencia, no esta puerta: si la tarjeta cambió, el clic no
// concede nada.
func TestUnAgenteNoEsperaTurnoContraSiMismo(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	es, _ := a.BuscarEnBoveda("Banco")
	id := es[0].ID

	// Pide cambiar la contraseña y nadie contesta…
	if _, err := f.Editar("Claude Code", agente.Peticion{
		ID: id, Campos: map[string]string{"secreto": "otra"},
	}); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatalf("cambiar un secreto tiene que pedir aprobación: %v", err)
	}
	// …y cambia de tema. **Eso no es esperar turno**: es el mismo que pregunta.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Errorf("un agente se ha bloqueado contra sí mismo: %v", err)
	}
	// Y lo que la ventana enseña es lo último que pidió, con su referencia nueva.
	q := a.EstadoDelAgente().Quiere
	if q == nil || q.Que != agente.QueCopiarSecreto {
		t.Fatalf("la tarjeta no enseña lo último que pidió: %+v", q)
	}
	// **Y un sí para eso no vale para lo de antes.**
	if err := aprobar(t, a, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Editar("Claude Code", agente.Peticion{
		ID: id, Campos: map[string]string{"secreto": "otra"},
	}); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("el sí de copiar ha servido para cambiar la contraseña")
	}
}

// aprobar contesta lo que la ventana esté enseñando, **leyendo su referencia del estado
// como lo hace la ventana**.
//
// No se le pasa a mano: la referencia la pone Go al dejar la petición pendiente y la
// ventana la devuelve al pulsar. Una prueba que se la inventara —o que mandara una vacía—
// se saltaría ese paso, que es la trampa del testigo puesto a mano: el ayudante haría lo
// que en producción hace el programa, y entonces nadie prueba esa parte.
func aprobar(t *testing.T, a *App, unRato bool) error {
	t.Helper()
	if a.EstadoDelAgente().Quiere == nil {
		t.Fatal("no hay nada esperando aprobación")
	}
	return aprobarLoQueSeVe(a, unRato)
}

// aprobarLoQueSeVe es lo mismo sin `t`, para los casos de una tabla que se declaran
// antes de que exista la `App` y solo pueden devolver el error.
func aprobarLoQueSeVe(a *App, unRato bool) error {
	q := a.EstadoDelAgente().Quiere
	if q == nil {
		return errors.New("no hay nada esperando aprobación")
	}
	return a.AprobarLoQuePideElAgente(q.Ref, unRato)
}

// aprobarSiempre es lo mismo con el tercer botón.
func aprobarSiempre(t *testing.T, a *App) error {
	t.Helper()
	q := a.EstadoDelAgente().Quiere
	if q == nil {
		t.Fatal("no hay nada esperando aprobación")
	}
	return a.AprobarSiempreAlAgente(q.Ref)
}

// **Y una petición que nadie contesta caduca, o bloquearía a los demás para siempre.**
//
// Es la otra mitad de que la segunda espere su turno: sin plazo, un agente que pide algo
// y se va —su cliente lo mató, la conversación se cerró— deja a todos los demás fuera.
// Caduca al mirarla, no con un reloj aparte: la ventana pregunta por el estado a menudo,
// así que la tarjeta **se cae sola** sin nada que vigilarla.
func TestLoQuePideUnAgenteCaducaSiNadieContesta(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	es, _ := a.BuscarEnBoveda("Banco")
	id := es[0].ID

	if _, err := f.CopiarSecreto("Claude Desktop", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal("tendría que pedir aprobación")
	}
	if a.EstadoDelAgente().Quiere == nil {
		t.Fatal("la ventana no está enseñando nada")
	}

	// Pasa el plazo sin que nadie conteste. **Se mueve el instante en que se pidió**, que
	// es lo que hay, en vez de esperar dos minutos de reloj.
	a.permisos.mu.Lock()
	a.permisos.pedidaEn = a.permisos.pedidaEn.Add(-PlazoParaContestar - time.Minute)
	a.permisos.mu.Unlock()

	// La tarjeta se ha caído sola.
	if q := a.EstadoDelAgente().Quiere; q != nil {
		t.Errorf("la petición caducada sigue en la ventana: %+v", q)
	}
	// Y otro agente puede pedir lo suyo, que es para lo que existe el plazo.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Errorf("el segundo sigue bloqueado por una petición caducada: %v", err)
	}
}

// **Pulsar sobre una tarjeta que ya ha cambiado no concede nada**, y esto es la mitad del
// arreglo que no se ve: la que protege a quien estaba leyendo.
//
// El caso es exacto: lees la tarjeta de A, llega B, y pulsas. El botón manda la
// referencia de **A**, que es lo que estaba pintado; Go ve que lo pendiente es B y no
// concede nada. Sin esta comprobación se aprobaba «lo que hubiera», o sea B — la cosa que
// no has leído.
//
// La espera de turno tapa casi todos los casos, y por eso esto hace falta igual: **dentro
// de un mismo agente la tarjeta sí puede cambiar**, que es la frontera que se eligió.
func TestPulsarSobreUnaTarjetaQueYaHaCambiadoNoConcedeNada(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)
	f := fuenteDelAgente{a}
	es, _ := a.BuscarEnBoveda("Banco")
	banco := es[0].ID
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Hacienda", Secreto: "otra",
	}); err != nil {
		t.Fatal(err)
	}
	otras, _ := a.BuscarEnBoveda("Hacienda")
	hacienda := otras[0].ID

	// Se pide la del Banco y **se lee la referencia de lo que se está viendo**.
	if _, err := f.CopiarSecreto("Claude Code", banco); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal("tendría que pedir aprobación")
	}
	loQueLei := a.EstadoDelAgente().Quiere
	if loQueLei == nil || loQueLei.Titulo != "Banco" {
		t.Fatalf("la ventana no enseña la del Banco: %+v", loQueLei)
	}

	// El mismo agente cambia de tema antes de que nadie pulse.
	if _, err := f.CopiarSecreto("Claude Code", hacienda); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal("la segunda tendría que pedir aprobación")
	}

	// Y ahora se pulsa con la referencia de lo que se leyó. **No concede nada y lo dice.**
	err := a.AprobarLoQuePideElAgente(loQueLei.Ref, false)
	if err == nil {
		t.Fatal("ha aceptado un sí sobre una tarjeta que ya había cambiado")
	}
	if !strings.Contains(err.Error(), "ya no es lo que está esperando") {
		t.Errorf("el mensaje no dice qué ha pasado: %v", err)
	}

	// Y lo importante: **no se ha concedido ninguna de las dos**.
	if _, err := f.CopiarSecreto("Claude Code", banco); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("se ha concedido la del Banco, que no era lo pendiente")
	}
	if _, err := f.CopiarSecreto("Claude Code", hacienda); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("se ha concedido la de Hacienda, que nadie aprobó")
	}
}
