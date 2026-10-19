package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/agente"
	"github.com/webcafeina/esfinge/internal/boveda"
)

// **Lo que se pide en otra bóveda ocurre en otra bóveda, y la ventana vuelve** (ADR 0055).
//
// Esto es el centro de la entrega. Lo que hace que valga la pena escribirlo así —abriendo
// el fichero del proyecto al final en vez de mirar lo que quedó en memoria— es la lección
// de `vaciarLoSensible`: lo que se comprueba sobre el objeto abierto no dice dónde ha
// acabado el byte.
func TestElAgenteTrabajaEnLaBovedaQueSeLeDiceYLaVentanaVuelve(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	f := fuenteDelAgente{a}

	// Se parte de la personal, que es donde `conUnProyecto` deja las cosas.
	if a.bovedaActiva() != "" {
		t.Fatalf("esto empieza en la personal y está en %q", a.bovedaActiva())
	}

	hecho := false
	if err := a.enLaBoveda("Acme", func() error {
		hecho = true
		// **Dentro se está dentro**: si esto no fuera verdad, el verbo escribiría en la
		// bóveda de siempre creyendo que está en la otra, que es el fallo que esto viene
		// a quitar.
		if a.bovedaActiva() != ref {
			t.Errorf("dentro de «Acme» la activa es %q", a.bovedaActiva())
		}
		return a.GuardarEnBoveda(boveda.Entrada{
			Tipo: boveda.TipoCredencial, Titulo: "Figma", Secreto: "x",
		})
	}); err != nil {
		t.Fatal(err)
	}
	if !hecho {
		t.Fatal("no se ha hecho nada dentro")
	}

	// **La vuelta no es inmediata, y eso es a propósito**: tres acciones seguidas en la
	// misma bóveda tienen que ser una ida y una vuelta. Aquí se fuerza el vencimiento en
	// vez de esperar diez segundos de reloj de verdad.
	if a.bovedaActiva() != ref {
		t.Errorf("antes del respiro se sigue dentro, y está en %q", a.bovedaActiva())
	}
	a.volverDeDondeLeDejoElAgente()
	if a.bovedaActiva() != "" {
		t.Errorf("la ventana no ha vuelto a la personal: está en %q", a.bovedaActiva())
	}

	// Y lo guardado está **en el fichero del proyecto**, no en el que se estaba mirando.
	if es, _ := a.BuscarEnBoveda("Figma"); len(es) != 0 {
		t.Errorf("la entrada ha acabado en la personal (%d coincidencias)", len(es))
	}
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if es, _ := a.BuscarEnBoveda("Figma"); len(es) != 1 {
		t.Errorf("la entrada no está en el proyecto: %d coincidencias", len(es))
	}
	_ = f
}

// **Y no se tira de nadie.** Si la persona conmuta mientras tanto, la vuelta no se hace:
// deshacer lo que alguien acaba de hacer con el ratón es peor que dejarlo donde está, y
// además lo dejaría en una bóveda que no eligió.
func TestSiLaPersonaConmutaElAgenteNoLaDevuelve(t *testing.T) {
	a, _, ref := conUnProyecto(t)

	if err := a.enLaBoveda("Acme", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	// La persona se va a su bóveda por su cuenta, antes de que venza el respiro.
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	// Y vuelve al proyecto, también ella.
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}

	a.volverDeDondeLeDejoElAgente()
	if a.bovedaActiva() != ref {
		t.Errorf("le han movido la bóveda por debajo: está en %q y la dejó en el proyecto",
			a.bovedaActiva())
	}
}

// **Lo que hace un agente no cuenta como actividad**, ni siquiera conmutando de bóveda.
//
// Es la regla absoluta de esta casa, la del goteo de iconos y la del código de un solo
// uso: con el agente tocando otra bóveda cada minuto, una bóveda abierta encima de la
// mesa no se cerraría nunca.
func TestConmutarPorUnAgenteNoMantieneLaBovedaAbierta(t *testing.T) {
	a, _, _ := conUnProyecto(t)

	if err := a.GuardarPreferencias(Preferencias{
		MinutosParaBloquear: 1, SegundosDePortapapeles: 5,
	}); err != nil {
		t.Fatal(err)
	}

	// Treinta idas y vueltas con el reloj corriendo, que es lo que haría un agente
	// trabajando en el proyecto de un cliente durante media hora.
	ahora := time.Now()
	a.vig.mu.Lock()
	a.vig.ahora = func() time.Time { return ahora }
	a.vig.ultimaActividad = ahora
	a.vig.mu.Unlock()

	for i := 0; i < 30; i++ {
		ahora = ahora.Add(time.Minute)
		if err := a.enLaBoveda("Acme", func() error { return nil }); err != nil {
			t.Fatal(err)
		}
		a.volverDeDondeLeDejoElAgente()
	}

	if !a.vig.tocaBloquear() {
		t.Error("no toca cerrar: conmutar por un agente ha movido el reloj del bloqueo")
	}
}

// Nombrar una bóveda que no existe, o nombrar una que está dormida, **no son el mismo
// problema y no se contestan igual**: una se arregla diciendo otro nombre y la otra
// abriéndola una vez en Esfinge, que el agente no puede hacer.
func TestLosNombresDeBovedaQueNoLlevanANingunaSeDistinguen(t *testing.T) {
	a, _, _ := conUnProyecto(t)

	err := a.enLaBoveda("La que no existe", func() error { return nil })
	if !errors.Is(err, agente.ErrNoEsaBoveda) {
		t.Errorf("una bóveda que no existe tendría que decirlo con su motivo: %v", err)
	}

	// Y la que sí existe se encuentra por su nombre, con mayúsculas o sin ellas: quien
	// habla con un agente escribe «acme», no la referencia al azar.
	if err := a.enLaBoveda("acme", func() error { return nil }); err != nil {
		t.Errorf("no ha encontrado «acme»: %v", err)
	}
}

// La lista que ve el agente: la personal, los proyectos de este equipo y las
// compartidas. **Lo archivado no sale**, porque archivar es decir «con esto ya no se
// trabaja» y ofrecérselo sería deshacer esa decisión por la puerta de atrás.
func TestLaListaDeBovedasDelAgenteDiceCualEstaAbiertaYDondeSeEscribe(t *testing.T) {
	a, _, ref := conUnProyecto(t)

	lista, err := a.bovedasParaElAgente()
	if err != nil {
		t.Fatal(err)
	}
	porNombre := map[string]BovedaParaElAgente{}
	for _, b := range lista {
		porNombre[b.Nombre] = b
	}
	if p, hay := porNombre["Personal"]; !hay || !p.EsLaPersonal || !p.Abierta {
		t.Errorf("la personal tiene que estar, decir que lo es y decir que está abierta: %+v", p)
	}
	if p, hay := porNombre["Acme"]; !hay || p.Abierta || !p.SeEscribe {
		t.Errorf("el proyecto tiene que estar, cerrado y con escritura: %+v", p)
	}

	// Con el proyecto abierto, la lista lo dice. Si esto no cambiara, el agente no
	// podría saber nunca dónde está trabajando.
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	lista, _ = a.bovedasParaElAgente()
	for _, b := range lista {
		if b.Nombre == "Acme" && !b.Abierta {
			t.Error("con el proyecto abierto, la lista sigue diciendo que no lo está")
		}
		if b.EsLaPersonal && b.Abierta {
			t.Error("la personal no está abierta y la lista dice que sí")
		}
	}
}

// **Una bóveda que existe y no está en este equipo tiene motivo propio.**
//
// No es «no existe» y el arreglo no es el mismo: ahí no hay nada que el agente pueda
// hacer —hay que abrirla una vez en Esfinge para que se baje—, y contestarle «no hay
// ninguna bóveda con ese nombre» le haría buscar un nombre que ya tenía bien.
func TestUnaBovedaDormidaSeDiceComoTalYNoComoQueNoExiste(t *testing.T) {
	a, _, ref := conUnProyecto(t)

	// Se le quita el fichero de debajo: es exactamente lo que ve un equipo que todavía
	// no la ha bajado.
	if err := os.Remove(rutaDeProyecto(ref)); err != nil {
		t.Fatal(err)
	}

	err := a.enLaBoveda("Acme", func() error {
		t.Error("no tendría que haber llegado a hacer nada")
		return nil
	})
	if !errors.Is(err, agente.ErrBovedaDormida) {
		t.Errorf("una bóveda dormida tiene que decirlo con su motivo, y dice: %v", err)
	}
	if errors.Is(err, agente.ErrNoEsaBoveda) {
		t.Error("se está diciendo que no existe, y existe: lo que pasa es que no está aquí")
	}

	// Y tampoco sale en la lista: ofrecer algo que no se puede abrir es ofrecer un fallo.
	lista, err := a.bovedasParaElAgente()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range lista {
		if b.Nombre == "Acme" {
			t.Error("una bóveda que no está en este equipo no se ofrece para nombrarla")
		}
	}
}

// **Mover lleva la entrada entera, con su secreto, y no deja nada detrás** (ADR 0056).
//
// Es el centro de la entrega, y se comprueba **abriendo los dos ficheros** en vez de
// mirar lo que quedó en memoria: es la lección de `vaciarLoSensible`, y aquí importa más
// porque lo que tiene que viajar es justo lo que no se puede ver desde fuera.
func TestMoverLlevaLaEntradaEnteraYNoDejaNadaDetras(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	f := fuenteDelAgente{a}

	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Brevo", Usuario: "yo@ejemplo.com",
		Secreto: "la-contrasena-de-brevo", TOTP: "JBSWY3DPEHPK3PXP",
		Sitios: []string{"https://login.brevo.com"},
	}); err != nil {
		t.Fatal(err)
	}
	es, _ := a.BuscarEnBoveda("Brevo")
	if len(es) != 1 {
		t.Fatalf("hacen falta una entrada para mover y hay %d", len(es))
	}
	id := es[0].ID

	// **Lo primero que tiene que pasar es que pida permiso**, porque esto quita algo de
	// la bóveda — y eso es más de lo que hace `borrar`, que lo manda a la papelera y
	// aun así lo pregunta. Lo decidió el cliente el 2026-10-15 cuando su agente señaló
	// que la acción más destructiva del canal era la única que no pasaba por la ventana.
	if _, err := f.Mover("Claude", agente.Peticion{ID: id, Destino: "Acme"}); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatalf("mover tendría que pedir aprobación y ha contestado: %v", err)
	}
	if es, _ := a.BuscarEnBoveda("Brevo"); len(es) != 1 {
		t.Fatal("ha tocado la entrada sin que nadie lo aprobara")
	}
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
		t.Fatal(err)
	}

	m, err := f.Mover("Claude", agente.Peticion{ID: id, Destino: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Desde != "Personal" || m.Hasta != "Acme" {
		t.Errorf("no dice de dónde a dónde: %+v", m)
	}
	if !m.ConSecreto {
		t.Error("llevaba contraseña y dice que no: quien lo pidió no puede saber que ha movido algo sensible")
	}
	// **Lo que vuelve no lleva el secreto**, que es la promesa de todo el canal.
	crudo, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(crudo, []byte("la-contrasena-de-brevo")) {
		t.Errorf("la contraseña ha salido por el canal: %s", crudo)
	}

	// La ventana vuelve a donde estaba, como en cualquier otra acción del agente.
	a.volverDeDondeLeDejoElAgente()
	if a.bovedaActiva() != "" {
		t.Errorf("la ventana se ha quedado en %q", a.bovedaActiva())
	}

	// **En la personal no queda nada, ni en la papelera.** Si quedara, la contraseña
	// del cliente seguiría treinta días en la bóveda de la que se quería sacar.
	if es, _ := a.BuscarEnBoveda("Brevo"); len(es) != 0 {
		t.Errorf("sigue en la personal: %d coincidencias", len(es))
	}
	if p, err := a.PapeleraDeBoveda(); err != nil {
		t.Fatal(err)
	} else {
		for _, e := range p {
			if e.Titulo == "Brevo" {
				t.Error("se ha quedado en la papelera de la personal, con su contraseña dentro")
			}
		}
	}

	// Y en el proyecto está **entera**: contraseña, segundo factor y sitio.
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	llegadas, _ := a.BuscarEnBoveda("Brevo")
	if len(llegadas) != 1 {
		t.Fatalf("no ha llegado al proyecto: %d coincidencias", len(llegadas))
	}
	entera, hay := a.boveda().Ver(llegadas[0].ID)
	if !hay {
		t.Fatal("no se encuentra la que ha llegado")
	}
	if entera.Secreto != "la-contrasena-de-brevo" {
		t.Errorf("la contraseña no ha viajado: %q", entera.Secreto)
	}
	if entera.TOTP != "JBSWY3DPEHPK3PXP" {
		t.Errorf("el segundo factor no ha viajado: %q", entera.TOTP)
	}
	if len(entera.Sitios) != 1 || entera.Sitios[0] != "https://login.brevo.com" {
		t.Errorf("el sitio no ha viajado: %v", entera.Sitios)
	}
	// **Identificador nuevo**: los de una bóveda son suyos.
	if entera.ID == id {
		t.Error("se ha llevado el identificador de la otra bóveda")
	}
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
}

// **Y con «déjala», se queda en las dos.** Es lo que pidió el cliente para una cuenta
// compartida, y lo que no puede pasar es que se mueva cuando se dijo que se copiara.
func TestMoverConDejarCopiaLaDejaEnLasDos(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	f := fuenteDelAgente{a}

	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Compartida", Secreto: "x",
	}); err != nil {
		t.Fatal(err)
	}
	es, _ := a.BuscarEnBoveda("Compartida")
	// **Con «déjala» no pide nada**, y ésa es la frontera exacta que eligió el cliente:
	// lo que se aprueba es **quitar**, no copiar. Si esto pidiera aprobación, la regla
	// sería otra y habría que decirlo.
	m, err := f.Mover("Claude", agente.Peticion{ID: es[0].ID, Destino: "Acme", DejarCopia: true})
	if err != nil {
		t.Fatalf("copiar no tendría que pedir nada, y ha contestado: %v", err)
	}
	if !m.SeQuedaCopia {
		t.Error("no dice que se queda copia, y se queda")
	}
	if es, _ := a.BuscarEnBoveda("Compartida"); len(es) != 1 {
		t.Errorf("se ha movido cuando se pidió copiar: %d en la personal", len(es))
	}
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if es, _ := a.BuscarEnBoveda("Compartida"); len(es) != 1 {
		t.Errorf("no ha llegado al proyecto: %d", len(es))
	}
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
}

// Mover a una bóveda que no existe **no mueve nada**: ni la deja a medias ni la guarda
// en la que estuviera abierta.
func TestMoverAUnaBovedaQueNoExisteNoTocaNada(t *testing.T) {
	a, _, _ := conUnProyecto(t)
	f := fuenteDelAgente{a}

	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Quieta", Secreto: "x",
	}); err != nil {
		t.Fatal(err)
	}
	es, _ := a.BuscarEnBoveda("Quieta")

	_, err := f.Mover("Claude", agente.Peticion{ID: es[0].ID, Destino: "La que no existe"})
	if !errors.Is(err, agente.ErrNoEsaBoveda) {
		t.Errorf("tendría que decir que esa bóveda no existe, y dice: %v", err)
	}
	if es, _ := a.BuscarEnBoveda("Quieta"); len(es) != 1 {
		t.Error("ha tocado la entrada cuando no tenía que mover nada")
	}
}

// **Mover desde una bóveda que no es la abierta no entra en bucle** (2026-10-15).
//
// Lo vio el cliente moviendo una credencial **de vuelta** a su bóveda: aprobaba en la
// ventana, el agente reintentaba, y Esfinge volvía a pedir permiso. Seis veces.
//
// La causa: `cambiarBoveda` cierra la válvula y **con ella el sí concedido**, que es lo
// correcto cuando el suelo cambia por cualquier otro motivo. Pero mover **tiene que
// conmutar al origen**, o sea que el reintento borraba su propia aprobación antes de
// mirarla. Una operación no puede invalidarse a sí misma.
//
// **Y la prueba que había no lo veía porque solo ejercitaba la dirección que funciona**:
// de la personal a un proyecto con la personal abierta no hay que conmutar al origen, y
// el sí sobrevive. Es la lección de siempre: probar el camino real, no el fácil.
func TestMoverDesdeUnaBovedaQueNoEsLaAbiertaNoPideAprobacionSinFin(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	f := fuenteDelAgente{a}

	// La entrada nace en el proyecto…
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "DeVuelta", Secreto: "x",
	}); err != nil {
		t.Fatal(err)
	}
	es, _ := a.BuscarEnBoveda("DeVuelta")
	id := es[0].ID

	// …y se trabaja desde la personal, que es como estaba el cliente.
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}

	pide := agente.Peticion{ID: id, Boveda: "Acme", Destino: "Personal"}
	if _, err := f.Mover("Claude", pide); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatalf("la primera vez tiene que pedir aprobación: %v", err)
	}
	// **Se aprueba primero y el respiro pasa después**, que es el orden de quien lo
	// vive: la tarjeta sale, se lee, se pulsa —y eso son segundos, no diez—, y el
	// reloj de la vuelta sigue corriendo por detrás.
	//
	// **La segunda versión de esta prueba lo hacía al revés** —respiro y luego
	// aprobar— y ése es el único orden en el que el sí sobrevive, así que pasaba en
	// verde con el bucle dentro. Por eso al cliente le fallaba siempre: aprueba
	// deprisa.
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
		t.Fatal(err)
	}
	a.volverDeDondeLeDejoElAgente()
	if a.bovedaActiva() != "" {
		t.Fatalf("el respiro tendría que haber devuelto la ventana y está en %q", a.bovedaActiva())
	}

	// **Y al reintentar tiene que valer.** Aquí es donde se iba en bucle.
	m, err := f.Mover("Claude", pide)
	if err != nil {
		t.Fatalf("con el sí dado, el reintento tendría que funcionar y dice: %v", err)
	}
	if m.Desde != "Acme" || m.Hasta != "Personal" {
		t.Errorf("no dice de dónde a dónde: %+v", m)
	}

	a.volverDeDondeLeDejoElAgente()
	if llegadas, _ := a.BuscarEnBoveda("DeVuelta"); len(llegadas) != 1 {
		t.Errorf("no ha llegado a la personal: %d", len(llegadas))
	}
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if quedan, _ := a.BuscarEnBoveda("DeVuelta"); len(quedan) != 0 {
		t.Errorf("sigue en el proyecto: %d", len(quedan))
	}
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
}

// **Y el bucle no era de `mover`: era de cualquier cosa que se apruebe en una bóveda
// nombrada.**
//
// Esto es lo que la 2.58.1 no vio. Se parcheó la conmutación de `mover` —guardando el sí
// y reponiéndolo a mano— y el fallo estaba un piso más abajo: **la vuelta automática de
// la ventana** cierra la válvula igual, así que pedir la contraseña de una entrada de un
// proyecto desde la personal entraba en el mismo bucle sin que nadie lo hubiera probado.
//
// El orden es el de quien lo vive: se pide, se aprueba deprisa, y el respiro vence
// **después**.
func TestPedirUnSecretoEnUnaBovedaNombradaNoPideAprobacionSinFin(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	f := fuenteDelAgente{a}

	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Brevo", Secreto: "s3creto",
	}); err != nil {
		t.Fatal(err)
	}
	es, _ := a.BuscarEnBoveda("Brevo")
	id := es[0].ID
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}

	// El agente nombra la bóveda, se conmuta y se pide la contraseña de dentro.
	pedir := func() error {
		var errDentro error
		if err := a.enLaBoveda("Acme", func() error {
			_, errDentro = f.CopiarSecreto("Claude", id)
			return nil
		}); err != nil {
			return err
		}
		return errDentro
	}
	if err := pedir(); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Fatalf("la primera vez tiene que pedir aprobación: %v", err)
	}

	// Se aprueba **antes** de que venza el respiro, y el respiro vence después.
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
		t.Fatal(err)
	}
	a.volverDeDondeLeDejoElAgente()
	if a.bovedaActiva() != "" {
		t.Fatalf("el respiro tendría que haber devuelto la ventana y está en %q", a.bovedaActiva())
	}

	// Y el reintento tiene que llevarse la contraseña, no volver a preguntar.
	if err := pedir(); err != nil {
		t.Fatalf("con el sí dado, el reintento tendría que funcionar y dice: %v", err)
	}
	a.volverDeDondeLeDejoElAgente()
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
}

// **Y el registro de un movimiento dice cómo se autorizó**, que es la pregunta para la
// que existe.
//
// Decía solo «hecho», de cuando mover no pedía nada; desde la 2.58.0 sí pide, y un apunte
// que no distingue lo aprobado de lo que fue directo no contesta «¿qué ha hecho un agente
// con mis secretos?». **Lo vio el paseo a mano** leyendo el registro al final, no una
// prueba: ahí salía `"Brevo" · hecho ()` al lado de `(preguntado)` y `(directo)`.
func TestElRegistroDeUnMovimientoDiceComoSeAutorizo(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	f := fuenteDelAgente{a}

	pon := func(titulo string) string {
		if err := a.GuardarEnBoveda(boveda.Entrada{
			Tipo: boveda.TipoCredencial, Titulo: titulo, Secreto: "x",
		}); err != nil {
			t.Fatal(err)
		}
		es, _ := a.BuscarEnBoveda(titulo)
		return es[0].ID
	}
	quitando, dejandoCopia := pon("Quitando"), pon("ConCopia")

	// Uno que quita del origen: lo aprueba la persona.
	if _, err := f.Mover("Claude", agente.Peticion{ID: quitando, Destino: "Acme"}); !errors.Is(
		err, agente.ErrPideAprobacion) {
		t.Fatalf("tendría que pedir aprobación: %v", err)
	}
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Mover("Claude", agente.Peticion{ID: quitando, Destino: "Acme"}); err != nil {
		t.Fatal(err)
	}
	// Y uno con copia, que va directo.
	if _, err := f.Mover("Claude", agente.Peticion{
		ID: dejandoCopia, Destino: "Acme", DejarCopia: true}); err != nil {
		t.Fatal(err)
	}
	a.volverDeDondeLeDejoElAgente()

	var preguntado, directo int
	for _, ap := range a.EstadoDelAgente().Dado {
		if !strings.HasPrefix(ap.Que, agente.QueMover) {
			continue
		}
		switch ap.Como {
		case boveda.ApuntePreguntado:
			preguntado++
		case boveda.ApunteDirecto:
			directo++
		}
	}
	if preguntado != 1 || directo != 1 {
		t.Errorf("el registro dice %d aprobados y %d directos: %+v",
			preguntado, directo, a.EstadoDelAgente().Dado)
	}
	_ = ref
}

// **Un sí nombra entera la acción que se aprobó, y no se gasta en otra.**
//
// Es lo que sustituye a que conmutar borrara el permiso: si el sí solo dijera «la entrada
// X», sería genérico de más y habría que tirarlo cada vez que el suelo cambia. Diciendo
// verbo, entrada, bóveda y destino, **no hay dónde gastarlo mal** — y entonces conmutar
// no tiene de qué protegerlo.
//
// Lo que esto cierra es lo que el permiso viejo dejaba abierto: aprobar una cosa y que el
// agente se lleve otra con el mismo identificador de entrada.
func TestUnSiNoSeGastaEnOtraCosa(t *testing.T) {
	ahora := time.Now()
	aprobado := loQuePideUnAgente{
		Quien: "Claude", Que: agente.QueMover, ID: "abc",
		Boveda: "Acme", Destino: "Personal",
	}

	otras := []struct {
		por string
		q   loQuePideUnAgente
	}{
		{"el verbo", loQuePideUnAgente{
			Quien: "Claude", Que: agente.QueCopiarSecreto, ID: "abc",
			Boveda: "Acme", Destino: "Personal"}},
		{"la bóveda", loQuePideUnAgente{
			Quien: "Claude", Que: agente.QueMover, ID: "abc",
			Boveda: "Beta", Destino: "Personal"}},
		{"el destino", loQuePideUnAgente{
			Quien: "Claude", Que: agente.QueMover, ID: "abc",
			Boveda: "Acme", Destino: "Beta"}},
		{"la entrada", loQuePideUnAgente{
			Quien: "Claude", Que: agente.QueMover, ID: "otra",
			Boveda: "Acme", Destino: "Personal"}},
		{"quién lo pide", loQuePideUnAgente{
			Quien: "Otro", Que: agente.QueMover, ID: "abc",
			Boveda: "Acme", Destino: "Personal"}},
	}
	for _, c := range otras {
		var p permisosDelAgente
		p.pendiente = &aprobado
		if _, ok := p.conceder(false, ahora); !ok {
			t.Fatal("no ha concedido")
		}
		if vale, _ := p.pedir(c.q, ahora); vale {
			t.Errorf("el sí se ha gastado en algo que cambia %s", c.por)
		}
	}

	// Y lo mismo, exacto, sí vale: que esto no sea una prueba que lo rechaza todo.
	var p permisosDelAgente
	p.pendiente = &aprobado
	if _, ok := p.conceder(false, ahora); !ok {
		t.Fatal("no ha concedido")
	}
	if vale, _ := p.pedir(aprobado, ahora); !vale {
		t.Error("la misma petición no ha podido gastar su propio sí")
	}
}

// **Y lo que la persona retira a mano se va de verdad**, que es lo único que conmutar ya
// no toca.
//
// Si esto se pone rojo, el botón de cortar ha dejado de significar lo que dice: diría que
// ha retirado el permiso y el agente podría gastarlo en el reintento siguiente.
func TestCortarAlAgenteSeLlevaElSiConcedido(t *testing.T) {
	a, _, _ := conUnProyecto(t)
	ahora := time.Now()
	q := loQuePideUnAgente{
		Quien: "Claude", Que: agente.QueCopiarSecreto, ID: "abc", Boveda: "Personal",
	}

	a.permisos.pendiente = &q
	if _, ok := a.permisos.conceder(false, ahora); !ok {
		t.Fatal("no ha concedido")
	}
	if err := a.CortarAlAgente(); err != nil {
		t.Fatal(err)
	}
	if vale, _ := a.permisos.pedir(q, ahora); vale {
		t.Error("el sí ha sobrevivido al botón de cortar")
	}

	// Y retirar al agente de la lista, igual.
	a.permisos.pendiente = &q
	if _, ok := a.permisos.conceder(false, ahora); !ok {
		t.Fatal("no ha concedido")
	}
	a.permisos.retirarLoConcedido()
	if vale, _ := a.permisos.pedir(q, ahora); vale {
		t.Error("el sí ha sobrevivido a retirar al agente")
	}
}

// **Lo que vio el cliente el 2026-10-19 con la 2.58.2: aprobó con «No volver a
// preguntar» y el bucle siguió igual.**
//
// No es la carrera del 16 —ésa está arreglada— sino otra cosa y **determinista**: los dos
// botones de permiso amplio de la tarjeta (`unRato` y `paraSiempre`) **no conceden el sí
// individual**, a propósito, dando por hecho que el verbo pasará por la válvula. Y hay
// tres verbos que **no pasan** por la válvula: el código de un solo uso, editar lo que no
// se deshace, y mover cuando quita. Para ésos, pulsar cualquiera de los dos **no concede
// nada en absoluto** y el reintento vuelve a preguntar. Para siempre.
//
// Esta prueba recorre los tres botones por los dos caminos. Si solo pasa el primero, la
// tarjeta está ofreciendo botones que no hacen lo que dicen.
func TestLosTresBotonesDeLaTarjetaValenParaLoQueNoPasaPorLaValvula(t *testing.T) {
	porCadaBoton := []struct {
		nombre  string
		aprobar func(a *App) error
	}{
		{"Solo ésta", func(a *App) error { return a.AprobarLoQuePideElAgente(false) }},
		{"Todo lo suyo, 5 minutos", func(a *App) error { return a.AprobarLoQuePideElAgente(true) }},
		{"No volver a preguntar", func(a *App) error { return a.AprobarSiempreAlAgente() }},
	}
	for _, boton := range porCadaBoton {
		t.Run("mover con «"+boton.nombre+"»", func(t *testing.T) {
			a, _, _ := conUnProyecto(t)
			f := fuenteDelAgente{a}
			if err := a.GuardarEnBoveda(boveda.Entrada{
				Tipo: boveda.TipoCredencial, Titulo: "Baristologia", Secreto: "x",
			}); err != nil {
				t.Fatal(err)
			}
			es, _ := a.BuscarEnBoveda("Baristologia")
			pide := agente.Peticion{ID: es[0].ID, Destino: "Acme"}

			if _, err := f.Mover("Claude", pide); !errors.Is(err, agente.ErrPideAprobacion) {
				t.Fatalf("la primera vez tiene que pedir aprobación: %v", err)
			}
			if err := boton.aprobar(a); err != nil {
				t.Fatal(err)
			}
			if _, err := f.Mover("Claude", pide); err != nil {
				t.Fatalf("aprobado con «%s», el reintento tendría que mover y dice: %v",
					boton.nombre, err)
			}
			a.volverDeDondeLeDejoElAgente()
		})

		t.Run("el código con «"+boton.nombre+"»", func(t *testing.T) {
			a, _, _, _, _ := conBoveda(t)
			f := fuenteDelAgente{a}
			if err := a.GuardarEnBoveda(boveda.Entrada{
				Tipo: boveda.TipoCredencial, Titulo: "ConCodigo", Secreto: "x",
				TOTP: "JBSWY3DPEHPK3PXP",
			}); err != nil {
				t.Fatal(err)
			}
			es, _ := a.BuscarEnBoveda("ConCodigo")
			id := es[0].ID

			if _, err := f.Codigo("Claude", id); !errors.Is(err, agente.ErrPideAprobacion) {
				t.Fatalf("la primera vez tiene que pedir aprobación: %v", err)
			}
			if err := boton.aprobar(a); err != nil {
				t.Fatal(err)
			}
			if _, err := f.Codigo("Claude", id); err != nil {
				t.Fatalf("aprobado con «%s», el reintento tendría que dar el código y dice: %v",
					boton.nombre, err)
			}
		})
	}
}

// **Y lo que de verdad importa de ese fallo: concede más de lo que la persona cree.**
//
// «No volver a preguntar» en una tarjeta de mover **enciende el permiso permanente** —que
// cubre copiar contraseñas y borrar entradas, y vive en las preferencias— **y el
// movimiento no se hace**. O sea: la persona cree estar aprobando un movimiento y lo que
// concede es un permiso que no caduca sobre *otros* verbos, sin obtener lo que pedía.
//
// Eso es lo contrario de lo que promete la ADR 0054, y por eso esto es un fallo de
// seguridad y no solo del bucle. Le pasó al cliente el 2026-10-19.
func TestAprobarUnMovimientoNoPuedeEncenderElPermisoDeOtrosVerbos(t *testing.T) {
	a, _, _ := conUnProyecto(t)
	f := fuenteDelAgente{a}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Baristologia", Secreto: "x",
	}); err != nil {
		t.Fatal(err)
	}
	es, _ := a.BuscarEnBoveda("Baristologia")
	id := es[0].ID

	// Se pide un movimiento y se aprueba con el botón que no caduca.
	if _, err := f.Mover("Claude", agente.Peticion{ID: id, Destino: "Acme"}); !errors.Is(
		err, agente.ErrPideAprobacion) {
		t.Fatal("tendría que pedir aprobación")
	}
	if err := a.AprobarSiempreAlAgente(); err != nil {
		t.Fatal(err)
	}
	a.volverDeDondeLeDejoElAgente()

	// **Y ahora copiar una contraseña no puede ir directo.** Si va, lo que se encendió
	// aprobando un movimiento es un permiso sobre otra cosa.
	if _, err := f.CopiarSecreto("Claude", id); !errors.Is(err, agente.ErrPideAprobacion) {
		t.Error("aprobar un movimiento ha encendido el permiso permanente de copiar contraseñas")
	}
}

// **Y lo que de verdad cierra esta familia: que la ventana no tenga su propia lista.**
//
// El fallo del 2026-10-19 no fue de lógica, fue de **la misma regla escrita en tres
// sitios**: el argumento de `pedir` en Go, un `que !== "codigo"` en la tarjeta, y lo que
// `concederComo` daba por hecho. Es el patrón de `vaciarLoSensible` y de
// `enConfiguracionDePruebas`, que ya costaron lo suyo.
//
// Esta prueba **lee el fuente de la ventana** y exige dos cosas: que los botones anchos
// cuelguen del campo que manda Go, y que no haya quedado ninguna lista escrita a mano.
// Si alguien vuelve a poner `que !== "<verbo>"` ahí, se pone roja.
func TestLaTarjetaNoTieneSuPropiaListaDeLoQuePasaPorLaValvula(t *testing.T) {
	fuente, err := os.ReadFile("../../frontend/src/App.tsx")
	if err != nil {
		t.Fatal(err)
	}
	texto := string(fuente)

	// Los dos botones anchos preguntan a Go.
	if n := strings.Count(texto, "elAgente.quiere.porValvula &&"); n != 2 {
		t.Errorf("los botones anchos tendrían que colgar de `porValvula` y hay %d, no 2", n)
	}
	// Y nadie decide esto comparando el verbo a mano.
	for _, lista := range []string{
		`elAgente.quiere.que !== "codigo"`,
		`elAgente.quiere.que !== "mover"`,
		`elAgente.quiere.que !== "editar"`,
	} {
		if strings.Contains(texto, lista) {
			t.Errorf("la tarjeta ha vuelto a escribir su propia lista: %s\n"+
				"Lo que pasa por la válvula lo dice app.PasaPorLaValvula, y viaja en la petición.",
				lista)
		}
	}
}

// Y la gemela, por el otro lado: **los tres verbos que no pasan son los tres**, dicho con
// sus nombres, para que añadir uno nuevo obligue a pasar por aquí.
//
// Si esta se pone roja, la pregunta no es cuál arreglar: es si el verbo nuevo **se le
// enseña al agente** o **no se deshace**, que es la frontera de la ADR 0054.
func TestLosVerbosQueNoPasanPorLaValvulaSonLosQueSeSabe(t *testing.T) {
	noPasan := map[string]bool{
		agente.QueCodigo: true, agente.QueEditar: true, agente.QueMover: true,
	}
	for _, h := range agente.LasHerramientas {
		if h.Verbo == "" {
			continue
		}
		if PasaPorLaValvula(h.Verbo) == noPasan[h.Verbo] {
			t.Errorf("«%s» (%s): PasaPorLaValvula dice %v y la lista de esta prueba dice que %s",
				h.Nombre, h.Verbo, PasaPorLaValvula(h.Verbo),
				map[bool]string{true: "no pasa", false: "sí pasa"}[noPasan[h.Verbo]])
		}
	}
	// Y lo que se le enseña al agente **y pide aprobación** nunca pasa, dicho por las
	// banderas y no por su nombre: es la frontera entera de la válvula.
	//
	// **Las dos banderas, no solo una**: `esfinge_generar` devuelve una contraseña que el
	// modelo ve y aun así pasa, porque **no toca la bóveda y no pide nada** — no hay
	// permiso que ampliar. La válvula habla de lo que se aprueba.
	for _, h := range agente.LasHerramientas {
		if h.DevuelveSecreto && h.PideAprobacion && PasaPorLaValvula(h.Verbo) {
			t.Errorf("«%s» le enseña un secreto al agente, lo aprueba una persona, "+
				"y aun así pasa por la válvula", h.Nombre)
		}
	}
}
