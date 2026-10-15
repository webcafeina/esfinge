package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
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
