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
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
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
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
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
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
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
		if err := a.AprobarLoQuePideElAgente(true); err != nil {
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
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CopiarSecreto("Claude Code", id); err != nil {
		t.Fatal(err)
	}
	// Y uno por la válvula.
	if _, err := f.CopiarSecreto("Claude Code", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatal(err)
	}
	if err := a.AprobarLoQuePideElAgente(true); err != nil {
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
	if err := a.AprobarLoQuePideElAgente(true); err != nil {
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
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
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
	if err := a.AprobarLoQuePideElAgente(true); err != nil {
		t.Fatal(err)
	}
	_, err = f.Editar("Claude Code", agente.Peticion{ID: id, Campos: map[string]string{"secreto": "otra"}})
	if !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("la válvula ha dejado cambiar una contraseña sin preguntar")
	}

	// Aprobándolo sí, **y la anterior queda en el historial**.
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
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
