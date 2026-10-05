package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/llavero"
	"github.com/webcafeina/esfinge/internal/sincro"
)

// conUnProyecto deja la bóveda personal abierta y un proyecto creado.
func conUnProyecto(t *testing.T) (*App, *sistemaFalso, string) {
	t.Helper()
	a, s, _ := conReloj(t)
	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	ref, err := a.CrearProyecto("Acme")
	if err != nil {
		t.Fatal(err)
	}
	return a, s, ref
}

// El camino entero: crear un proyecto, conmutar, y que lo que se ve dentro sea lo
// de ese proyecto y no lo de la bóveda personal.
func TestConmutarEntreLaPersonalYUnProyecto(t *testing.T) {
	a, _, ref := conUnProyecto(t)

	// En la personal hay una cuenta; en el proyecto, otra.
	if err := a.GuardarEnBoveda(boveda.Entrada{Tipo: boveda.TipoCredencial, Titulo: "Mi banco", Secreto: "x"}); err != nil {
		t.Fatal(err)
	}

	lista, err := a.Proyectos()
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 1 || lista[0].Nombre != "Acme" || lista[0].Ref != ref {
		t.Fatalf("la lista de proyectos es %+v", lista)
	}
	if !lista[0].EnEsteEquipo {
		t.Error("el proyecto que se acaba de crear tiene que estar en este equipo")
	}
	if lista[0].Activo {
		t.Error("crear un proyecto no lo abre")
	}

	// Se conmuta.
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoBoveda(); !e.Abierta || e.Proyecto != ref {
		t.Fatalf("tras conmutar, el estado es %+v", e)
	}
	// **Y lo de la personal no está aquí**, que es lo que de verdad separa una
	// bóveda de otra.
	if hay, _ := a.BuscarEnBoveda("banco"); len(hay) != 0 {
		t.Fatalf("en el proyecto se ven %d entradas de la bóveda personal", len(hay))
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{Tipo: boveda.TipoCredencial, Titulo: "Hosting de Acme", Secreto: "y"}); err != nil {
		t.Fatal(err)
	}

	// Y la lista sigue saliendo con un proyecto abierto, que es lo que permite
	// conmutar sin volver a la personal.
	lista, err = a.Proyectos()
	if err != nil {
		t.Fatalf("con un proyecto abierto no se puede listar: %v", err)
	}
	if len(lista) != 1 || !lista[0].Activo {
		t.Fatalf("la lista con el proyecto abierto es %+v", lista)
	}
	if lista[0].Usado == "" {
		t.Error("abrir un proyecto tiene que apuntar cuándo, que es por lo que se ordena la lista")
	}

	// Se vuelve a la personal, y **vuelve abierta y sin pedir la maestra** (2026-10-05):
	// la clave está en memoria y es la misma con la que `conLaPersonal` abre este
	// fichero mientras hay un proyecto delante. Sigue habiendo **una sola bóveda
	// abierta**: ésta se abre después de cerrar la otra, no a la vez.
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoBoveda(); !e.Abierta || e.Proyecto != "" {
		t.Fatalf("al volver, el estado es %+v", e)
	}
	if hay, _ := a.BuscarEnBoveda("banco"); len(hay) != 1 {
		t.Fatalf("de vuelta en la personal se ven %d entradas y había una", len(hay))
	}
	// Y lo del proyecto tampoco está aquí: son dos ficheros.
	if hay, _ := a.BuscarEnBoveda("Hosting"); len(hay) != 0 {
		t.Fatalf("en la personal se ven %d entradas del proyecto", len(hay))
	}
}

// **Bloquear se lleva la clave de la bóveda personal**, no solo la bóveda abierta.
//
// Es la prueba de la relajación que la ADR 0050 acepta: esa clave vive en memoria
// para que conmutar no pida la maestra cada vez, así que si el bloqueo no la
// borrara, el reloj dejaría de significar lo que dice **en todas las demás
// bóvedas**, que es peor que en la que se está mirando.
func TestAlBloquearSeOlvidaLaClaveDeLaPersonal(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}

	// Pasa el plazo sin tocar nada: el de serie son quince minutos.
	a.vig.ultimaActividad = a.vig.ahora().Add(-20 * time.Minute)
	a.repasar()

	if a.boveda() != nil {
		t.Fatal("el proyecto sigue abierto después de bloquear")
	}
	if a.bovedaActiva() != "" {
		t.Error("tras bloquear sigue habiendo una bóveda de proyecto activa")
	}
	// **Y lo que importa:** ya no se puede abrir ningún proyecto sin la maestra.
	if err := a.AbrirProyecto(ref); err == nil {
		t.Fatal("tras bloquear todavía se puede abrir un proyecto: la clave de la personal se ha quedado en memoria")
	}
	if len(a.llavePrincipal) != 0 {
		t.Error("la clave de la bóveda personal sigue en memoria después de bloquear")
	}
}

// Cerrar a mano hace lo mismo que bloquear: cierra todo, no solo lo que se mira.
func TestCerrarAManoSeLlevaLaClaveDeLaPersonal(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	a.CerrarBoveda()
	if err := a.AbrirProyecto(ref); err == nil {
		t.Fatal("tras cerrar a mano todavía se puede abrir un proyecto")
	}
}

// **El desbloqueo con el sistema es de la bóveda personal, y solo de ella**
// (ADR 0050).
//
// Dos fallos que esto vigila, y los dos salen de que `EstadoDelDesbloqueo` y
// `ActivarDesbloqueo` preguntaban por «la bóveda abierta»:
//
//   - con un proyecto abierto, la pantalla ofrecería Touch ID **para el proyecto**,
//     que es la forma exacta del fallo que arregló la ADR 0044;
//   - y activarlo pondría la ranura del sistema en el proyecto, lo que convertiría
//     «una entrada en el llavero» en una por bóveda y, con ella, un diálogo del
//     sistema por proyecto tras cada actualización.
//
// Y el escenario está elegido para que distinga: **la personal sin desbloqueo
// puesto y con «ahora no» contestado**.
//
// La primera versión de esta prueba activaba el desbloqueo antes y comprobaba que
// no se sugería — y pasaba igual con el fallo dentro, porque `Sugerir` ya era falso
// por estar puesto. Lo dijo mutar la línea, no leerla: una prueba que no distingue
// los dos casos no está comprobando nada.
func TestConUnProyectoAbiertoNoSeOfreceTouchIDParaEl(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	a.llavero = &llavero.DeMentira{ComoSeLlama: "Touch ID"}

	// A la personal se le ofreció y contestó «ahora no»: no se le vuelve a ofrecer.
	if e := a.EstadoDelDesbloqueo(); !e.Sugerir {
		t.Fatalf("de partida tenía que ofrecerlo: %+v", e)
	}
	if err := a.NoOfrecerElDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); e.Sugerir {
		t.Fatalf("se contestó y sigue ofreciéndolo: %+v", e)
	}

	// Y ahora se abre un proyecto. Preguntándole a la bóveda abierta, el
	// identificador sería el del proyecto —al que nunca se le ha ofrecido nada— y la
	// tarjeta volvería a salir, ofreciendo Touch ID **para el proyecto**: la forma
	// exacta del fallo que arregló la ADR 0044, con otra cara.
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); e.Sugerir {
		t.Errorf("con un proyecto abierto se ofrece Touch ID, y esa ranura es de la bóveda personal: %+v", e)
	}
}

// **«Salir del proyecto» y «Cerrar la bóveda» no son lo mismo**, y esta prueba existe
// porque durante un tiempo sí lo fueron: las dos cerraban todo, olvidaban la clave de
// la personal y dejaban la pantalla de desbloquear. Dos botones con dos nombres para
// una sola cosa. Lo vio el cliente preguntando lo evidente —«¿no vuelven los dos a mi
// bóveda?»— y no lo había dicho ninguna prueba, porque cada una comprobaba lo suyo y
// ninguna las comparaba.
//
// Lo que las separa, y es lo que se comprueba aquí: salir **devuelve la personal
// abierta**; cerrar **cierra todo y olvida la clave**.
func TestSalirDeUnProyectoNoEsCerrarLaBoveda(t *testing.T) {
	a, _, ref := conUnProyecto(t)

	// Cerrar: no queda nada abierto, y **tampoco la clave de la personal en memoria**,
	// que es lo que hace que el reloj del bloqueo siga siendo verdad.
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	a.CerrarBoveda()
	if e := a.EstadoBoveda(); e.Abierta {
		t.Fatalf("cerrar la bóveda ha dejado algo abierto: %+v", e)
	}
	if len(a.llaveDeLaPrincipal()) != 0 {
		t.Error("cerrar la bóveda ha dejado la clave de la personal en memoria")
	}

	// Salir: la personal vuelve **abierta**, y además se puede leer lo suyo sin
	// teclear nada, que es lo que de verdad lo distingue de cerrar.
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{Tipo: boveda.TipoCredencial, Titulo: "Mi banco", Secreto: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	e := a.EstadoBoveda()
	if !e.Abierta || e.Proyecto != "" || e.NombreDelProyecto != "" {
		t.Fatalf("salir del proyecto tenía que dejar la personal abierta: %+v", e)
	}
	hay, err := a.BuscarEnBoveda("banco")
	if err != nil || len(hay) != 1 {
		t.Fatalf("de vuelta en la personal se ven %d entradas y había una (%v)", len(hay), err)
	}
}

// Y el caso que la de arriba **no** cubría, porque parte de una personal que ya
// contestó: una personal **a la que no se le ha ofrecido nunca**.
//
// Ahí el identificador que mira `Sugerir` es el bueno —el de la personal— y aun así
// la tarjeta salía, **encima de la pantalla de un proyecto**: «Puedes abrir esta
// bóveda con Touch ID» diciendo «esta» sobre otra. Y lo peor no es el texto, es que
// **activar con un proyecto abierto está prohibido**, así que el botón de la tarjeta
// solo podía dar un error. Es la regla que costó `ExportarLlaves`.
//
// Se vio **mirando una captura**, no leyendo el código ni con una prueba: la de
// arriba pasaba en verde con esto dentro.
func TestConUnProyectoAbiertoNoSeOfreceNadaAunqueLaPersonalNoHayaContestado(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	a.llavero = &llavero.DeMentira{ComoSeLlama: "Touch ID"}

	// Nadie ha contestado nada: en la personal se ofrece, que es lo que hace que esta
	// prueba distinga de la de arriba.
	if e := a.EstadoDelDesbloqueo(); !e.Sugerir {
		t.Fatalf("en la personal, sin contestar, tenía que ofrecerlo: %+v", e)
	}

	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); e.Sugerir {
		t.Errorf("con un proyecto abierto ofrece algo que al pulsarlo falla: %+v", e)
	}

	// Y al volver a la personal vuelve a ofrecerse: no se pierde, se mueve a donde se
	// puede aceptar. Sin esto, «no ofrecer» podría estar apagándolo para siempre.
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); !e.Sugerir {
		t.Errorf("de vuelta en la personal ya no se ofrece, o sea que se perdió: %+v", e)
	}
}

// Y activarlo sobre un proyecto pondría una ranura del sistema por bóveda, lo que
// convierte **un** diálogo del sistema tras cada actualización en uno por proyecto.
// Además esa ranura no se sube, así que no serviría en el otro equipo.
func TestNoSeActivaElDesbloqueoSobreUnProyecto(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	a.llavero = &llavero.DeMentira{ComoSeLlama: "Touch ID"}

	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivarDesbloqueo(); err == nil {
		t.Fatal("se ha activado el desbloqueo del sistema sobre una bóveda de proyecto")
	}
	// Se mira **el fichero** y no la memoria, que es donde se vería el daño.
	if boveda.RanuraDelSistemaEn(rutaDeProyecto(ref)) {
		t.Error("el proyecto tiene ranura del sistema: una entrada más en el llavero y un diálogo más por actualización")
	}

	// Y en la personal sí se puede, que es la otra mitad: la guarda no puede dejar
	// el desbloqueo inservible.
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivarDesbloqueo(); err != nil {
		t.Fatalf("en la bóveda personal tiene que poder activarse: %v", err)
	}
	if !boveda.RanuraDelSistemaEn(rutaBovedaPrincipal()) {
		t.Error("la ranura no ha quedado en la bóveda personal")
	}
}

// El registro es una caché para encontrar ficheros, **y no lleva nombres**: un
// `bovedas.json` con «Acme» dentro sería la lista de clientes de Webcafeína en
// claro en el disco.
func TestElRegistroNoLlevaNombresYLosFicherosVanPorReferencia(t *testing.T) {
	_, _, ref := conUnProyecto(t)

	datos, err := os.ReadFile(rutaDelRegistro())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(datos), "Acme") {
		t.Error("el nombre del proyecto está en claro en el registro")
	}
	if !strings.Contains(string(datos), ref) {
		t.Error("el registro no apunta la referencia del proyecto")
	}

	// Y el fichero se llama por la referencia, no por el proyecto.
	entradas, err := os.ReadDir(filepath.Join(carpetaDeEsfinge(), carpetaDeProyectos))
	if err != nil {
		t.Fatal(err)
	}
	if len(entradas) != 1 {
		t.Fatalf("en la carpeta de proyectos hay %d ficheros", len(entradas))
	}
	if entradas[0].Name() != ref+".esfinge" {
		t.Errorf("el fichero se llama %q y tenía que llamarse por la referencia", entradas[0].Name())
	}
}

// Lo de la cuenta es de la bóveda personal, **esté abierta o no**, y por eso su
// fichero no puede colgar de la bóveda activa: con un proyecto abierto se buscaría
// la sesión al lado del proyecto y se perdería.
func TestLaCuentaNoCuelgaDeLaBovedaActiva(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	antes := rutaCuenta()
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if ahora := rutaCuenta(); ahora != antes {
		t.Errorf("con un proyecto abierto, lo de la cuenta se busca en %q y antes era %q", ahora, antes)
	}
	if strings.Contains(antes, ref) {
		t.Error("el fichero de la cuenta cuelga de una bóveda de proyecto")
	}
}

// **Crear y renombrar funcionan también desde dentro de otro proyecto**, y eso no
// es una concesión: la lista vive en la bóveda personal, y lo que llega a ella es la
// clave que se guardó al conmutar. Obligar a volver a la personal sería obligar a
// teclear la contraseña maestra para ponerle nombre a algo.
func TestCrearYRenombrarFuncionanDesdeCualquierBoveda(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	if err := a.RenombrarProyecto(ref, "Acme S. A."); err != nil {
		t.Fatal(err)
	}
	lista, _ := a.Proyectos()
	if len(lista) != 1 || lista[0].Nombre != "Acme S. A." {
		t.Fatalf("tras renombrar: %+v", lista)
	}

	// Y ahora lo mismo con un proyecto abierto y la personal cerrada.
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	otra, err := a.CrearProyecto("Beta")
	if err != nil {
		t.Fatalf("no se puede crear un proyecto desde dentro de otro: %v", err)
	}
	if err := a.RenombrarProyecto(ref, "Acme, S. A."); err != nil {
		t.Fatalf("no se puede renombrar desde dentro de otro: %v", err)
	}

	// Y los dos cambios han llegado a la bóveda personal de verdad, no a una copia
	// en memoria: se comprueba volviendo a abrirla con su contraseña.
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	lista, _ = a.Proyectos()
	if len(lista) != 2 {
		t.Fatalf("en la personal hay %d proyectos y tenían que ser 2: %+v", len(lista), lista)
	}
	nombres := map[string]bool{}
	for _, p := range lista {
		nombres[p.Nombre] = true
	}
	if !nombres["Acme, S. A."] || !nombres["Beta"] {
		t.Errorf("los nombres que han llegado son %+v", nombres)
	}
	if _, hay := nombres[""]; hay {
		t.Error("hay un proyecto sin nombre")
	}
	_ = otra
}

// Un proyecto que no está en este equipo es un estado, no un fallo, y se dice sin
// crear nada: **nunca se fabrica una bóveda vacía** para tapar el hueco.
func TestUnProyectoQueNoEstaAquiNoSeCrea(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	ruta := rutaDeProyecto(ref)
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}

	lista, err := a.Proyectos()
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 1 {
		t.Fatalf("el proyecto ha desaparecido de la lista: %+v", lista)
	}
	if lista[0].EnEsteEquipo {
		t.Error("dice que está en este equipo y su fichero no está")
	}
	if err := a.AbrirProyecto(ref); err == nil {
		t.Fatal("ha abierto un proyecto que no está en este equipo")
	}
	if _, err := os.Stat(ruta); err == nil {
		t.Fatal("abrir un proyecto que no está lo ha creado vacío")
	}
}

// **Mover una entrada de una bóveda a otra no la pierde**, que es lo único que de
// verdad importa de esta operación: lo que se mueve es una contraseña que a lo
// mejor no está en ningún otro sitio.
//
// Se comprueban las dos direcciones —de la personal a un proyecto y al revés— y
// que el secreto llega entero, porque mover algo sin su contraseña es perderla con
// más pasos.
func TestLlevarUnaEntradaAOtraBoveda(t *testing.T) {
	a, _, ref := conUnProyecto(t)

	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Hosting de Acme",
		Usuario: "yo@acme.com", Secreto: "la de acme", Notas: "la nota",
	}); err != nil {
		t.Fatal(err)
	}
	lista, _ := a.BuscarEnBoveda("Acme")
	if len(lista) != 1 {
		t.Fatalf("en la personal hay %d entradas", len(lista))
	}
	id := lista[0].ID

	if err := a.LlevarAOtraBoveda(id, ref, false); err != nil {
		t.Fatal(err)
	}

	// En el origen ya no está viva, **y está en la papelera**: borrar no es para
	// siempre (ADR 0026), y menos cuando acaba de salir de aquí.
	if l, _ := a.BuscarEnBoveda("Acme"); len(l) != 0 {
		t.Errorf("sigue en la bóveda de origen: %d", len(l))
	}
	if n := a.EstadoBoveda().EnLaPapelera; n != 1 {
		t.Errorf("en la papelera del origen hay %d y tenía que haber 1", n)
	}

	// Y en el destino está entera, con su secreto.
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	l, _ := a.BuscarEnBoveda("Acme")
	if len(l) != 1 {
		t.Fatalf("en el proyecto hay %d entradas", len(l))
	}
	if l[0].ID == id {
		t.Error("ha llegado con el mismo identificador: es una copia, no la misma entrada en dos sitios")
	}
	entera, err := a.VerDeBoveda(l[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if entera.Secreto != "la de acme" {
		t.Errorf("el secreto ha llegado como %q", entera.Secreto)
	}
	if entera.Usuario != "yo@acme.com" || entera.Notas != "la nota" {
		t.Errorf("no ha llegado entera: %+v", entera)
	}

	// Y de vuelta, que es el camino que necesita abrir la personal con la clave
	// guardada en vez de con la contraseña maestra.
	if err := a.LlevarAOtraBoveda(l[0].ID, "", false); err != nil {
		t.Fatal(err)
	}
	if l, _ := a.BuscarEnBoveda("Acme"); len(l) != 0 {
		t.Errorf("sigue en el proyecto: %d", len(l))
	}
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if l, _ := a.BuscarEnBoveda("Acme"); len(l) != 1 {
		t.Fatalf("de vuelta en la personal hay %d entradas", len(l))
	}
}

// **Si el destino falla, la entrada sigue en el origen.** Es la rama que justifica
// el orden —primero existe allí, después desaparece de aquí— y la que no se ve
// nunca, porque solo ocurre cuando algo va mal.
func TestSiElDestinoFallaLaEntradaNoSePierde(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Hosting de Acme", Secreto: "la de acme",
	}); err != nil {
		t.Fatal(err)
	}
	lista, _ := a.BuscarEnBoveda("Acme")
	id := lista[0].ID

	// El destino deja de poder abrirse: es lo que pasa con un fichero borrado, un
	// disco lleno o una bóveda que todavía no se ha bajado a este equipo.
	if err := os.Remove(rutaDeProyecto(ref)); err != nil {
		t.Fatal(err)
	}

	if err := a.LlevarAOtraBoveda(id, ref, false); err == nil {
		t.Fatal("ha dicho que sí con el destino inservible")
	}
	// **Y lo que importa: sigue aquí, viva y entera.**
	l, _ := a.BuscarEnBoveda("Acme")
	if len(l) != 1 {
		t.Fatalf("la entrada se ha perdido: quedan %d", len(l))
	}
	entera, err := a.VerDeBoveda(l[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if entera.Secreto != "la de acme" {
		t.Errorf("sigue aquí pero sin su secreto: %q", entera.Secreto)
	}
	if n := a.EstadoBoveda().EnLaPapelera; n != 0 {
		t.Errorf("la ha mandado a la papelera aunque el destino falló: %d", n)
	}
}

// Copiar deja las dos, que es lo que se pide cuando una cuenta la usan el cliente y
// la casa.
func TestCopiarAOtraBovedaDejaLasDos(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Hosting de Acme", Secreto: "la de acme",
	}); err != nil {
		t.Fatal(err)
	}
	lista, _ := a.BuscarEnBoveda("Acme")

	if err := a.LlevarAOtraBoveda(lista[0].ID, ref, true); err != nil {
		t.Fatal(err)
	}
	if l, _ := a.BuscarEnBoveda("Acme"); len(l) != 1 {
		t.Errorf("copiar se ha llevado la original: quedan %d", len(l))
	}
	if n := a.EstadoBoveda().EnLaPapelera; n != 0 {
		t.Errorf("copiar ha mandado algo a la papelera: %d", n)
	}
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if l, _ := a.BuscarEnBoveda("Acme"); len(l) != 1 {
		t.Errorf("la copia no ha llegado: %d", len(l))
	}
}

// Y no se lleva nada a la bóveda en la que ya está: sin esto, «mover a la misma»
// borraría la original después de guardar un duplicado al lado.
func TestNoSeLlevaUnaEntradaALaBovedaEnLaQueYaEsta(t *testing.T) {
	a, _, _ := conUnProyecto(t)
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Hosting de Acme", Secreto: "x",
	}); err != nil {
		t.Fatal(err)
	}
	lista, _ := a.BuscarEnBoveda("Acme")
	if err := a.LlevarAOtraBoveda(lista[0].ID, "", false); err == nil {
		t.Fatal("ha movido una entrada a la bóveda en la que ya estaba")
	}
	if l, _ := a.BuscarEnBoveda("Acme"); len(l) != 1 {
		t.Errorf("y encima se ha perdido: quedan %d", len(l))
	}
}

// **Un error que no es de red no se cuenta como falta de red.**
//
// `ErrOtraBoveda` —lo que baja del servidor no es esta bóveda— caía en el cajón de
// «Sin conexión con el servidor de cuentas», que manda a mirar el wifi cuando lo
// que pasa es otra cosa. Es la lección del mensaje impreciso: no es impreciso,
// **señala a otro sitio**. Lo destapó mutar la ruta de un proyecto para que subiera
// a la de la bóveda personal.
func TestLosErroresDeLaSincronizacionSeDistinguen(t *testing.T) {
	a, _, _ := conReloj(t)
	for _, c := range []struct {
		err    error
		estado string
	}{
		{boveda.ErrOtraBoveda, "error"},
		{boveda.ErrMuchosBorrados, "muchos-borrados"},
		{errors.New("algo que nadie ha visto"), "sin-conexion"},
	} {
		a.alSincronizar(sincro.Resultado{}, c.err)
		// Se mira el estado que se guarda, no el que enseña `EstadoDeCuenta`: ése
		// dice «apagada» mientras no haya cuenta, y aquí lo que se comprueba es la
		// clasificación del error.
		a.cu.mu.Lock()
		guardado := a.cu.estado.Estado
		a.cu.mu.Unlock()
		if e := guardado; e != c.estado {
			t.Errorf("con %v el estado es %q y tenía que ser %q", c.err, e, c.estado)
		}
	}
}

// El estado de la bóveda dice **en cuál se está trabajando**, con su nombre: es lo
// que la barra de herramientas enseña, y con la referencia sola la ventana tendría
// que pedir la lista para traducirla —y pedirla con la bóveda cerrada es un 400—.
func TestElEstadoDiceEnQueBovedaSeTrabaja(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	if e := a.EstadoBoveda(); e.Proyecto != "" || e.NombreDelProyecto != "" {
		t.Fatalf("en la bóveda personal el estado dice %+v", e)
	}
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	e := a.EstadoBoveda()
	if e.Proyecto != ref {
		t.Errorf("la referencia activa es %q y tenía que ser %q", e.Proyecto, ref)
	}
	if e.NombreDelProyecto != "Acme" {
		t.Errorf("el nombre que se enseña es %q y tenía que ser «Acme»", e.NombreDelProyecto)
	}
}

// **Entregar un proyecto deja una bóveda que ya no depende de nada de aquí**
// (ADR 0051).
//
// Es la mitigación del coste que la ADR 0050 acepta —perder la bóveda personal es
// perder todos los proyectos—: lo entregado sobrevive a eso porque tiene su propia
// contraseña y su propia clave de recuperación.
func TestEntregarUnProyecto(t *testing.T) {
	a, s, ref := conUnProyecto(t)
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Hosting de Acme", Secreto: "la de acme",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}

	destino := filepath.Join(t.TempDir(), "Acme.esfinge")
	s.mu.Lock()
	s.guardaEn = destino
	s.mu.Unlock()

	const suya = "la contraseña del cliente, larga"
	recuperacion, err := a.EntregarProyecto(ref, suya)
	if err != nil {
		t.Fatal(err)
	}
	if recuperacion == "" {
		t.Fatal("entregar sin clave de recuperación: quien la recibe se queda sin segunda puerta")
	}

	// Abre con lo suyo, con lo de dentro dentro.
	entregada, err := boveda.Abrir(destino, suya)
	if err != nil {
		t.Fatalf("la bóveda entregada no abre con su contraseña: %v", err)
	}
	if l := entregada.Buscar("Acme"); len(l) != 1 {
		t.Fatalf("la entregada tiene %d entradas", len(l))
	}
	// Y **no abre con la de quien la entregó**, ni con su bóveda personal.
	if _, err := boveda.Abrir(destino, maestraDePrueba); err == nil {
		t.Error("la bóveda entregada abre con la contraseña de quien la entregó")
	}
	if boveda.RanuraPrincipalEn(destino) {
		t.Error("la bóveda entregada sigue abriéndose con la bóveda personal de quien la entregó")
	}

	// **Y el proyecto sigue aquí**: entregar es dar una copia, no desprenderse.
	lista, _ := a.Proyectos()
	if len(lista) != 1 {
		t.Fatalf("entregar se ha llevado el proyecto: quedan %+v", lista)
	}
	if _, err := os.Stat(rutaDeProyecto(ref)); err != nil {
		t.Error("entregar ha borrado el fichero del proyecto")
	}
}

// **Y no se abre el diálogo del sistema para un fichero que no se va a escribir.**
//
// Es la regla que costó `ExportarLlaves`: lo que hace falta para decidir se mira
// antes de pedirle a alguien que decida. El doble **cuenta las veces**, porque
// mirar el argumento no distingue «no me han llamado» de «me han llamado sin
// carpeta».
func TestEntregarMiraAntesDeAbrirElDialogo(t *testing.T) {
	a, s, ref := conUnProyecto(t)

	// Con la bóveda cerrada no hay con qué abrir el proyecto.
	a.CerrarBoveda()
	if _, err := a.EntregarProyecto(ref, "una contraseña larga de prueba"); err == nil {
		t.Fatal("ha entregado con la bóveda cerrada")
	}
	if n := s.vecesQueHaPreguntadoDondeGuardar(); n != 0 {
		t.Errorf("ha abierto el diálogo del sistema %d veces con la bóveda cerrada", n)
	}

	// Y sin contraseña para quien la recibe tampoco.
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if _, err := a.EntregarProyecto(ref, "   "); err == nil {
		t.Fatal("ha entregado sin contraseña")
	}
	if n := s.vecesQueHaPreguntadoDondeGuardar(); n != 0 {
		t.Errorf("ha abierto el diálogo del sistema %d veces sin contraseña", n)
	}
}

// Archivar **borra el fichero de este equipo** y lo deja en el servidor, que es lo
// que hace que un proyecto terminado deje de ocupar sitio y de estar en el disco.
func TestArchivarUnProyectoSeLlevaElFicheroDeAqui(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	ruta := rutaDeProyecto(ref)

	if err := a.ArchivarProyecto(ref, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ruta); err == nil {
		t.Error("archivar ha dejado el fichero en este equipo")
	}
	// Sigue en la lista, marcado: archivar no es borrar.
	lista, _ := a.Proyectos()
	if len(lista) != 1 || !lista[0].Archivado {
		t.Fatalf("tras archivar: %+v", lista)
	}
	// Y la lista dice lo mismo que el disco. **El mensaje no da por hecho cuál de
	// los dos está mal**: saltó al mutar archivar para que no borrara el fichero, y
	// decía «su fichero no está» cuando lo que pasaba era justo lo contrario.
	if lista[0].EnEsteEquipo {
		t.Error("la lista dice que el proyecto archivado sigue en este equipo")
	}

	// Y desarchivar lo devuelve a la lista del día a día —el fichero se baja
	// aparte, con `BajarProyecto`—.
	if err := a.ArchivarProyecto(ref, false); err != nil {
		t.Fatal(err)
	}
	lista, _ = a.Proyectos()
	if lista[0].Archivado {
		t.Error("desarchivar no lo ha devuelto a la lista")
	}
}

// Y no se archiva la que se está usando: dejaría la bóveda abierta sin fichero, y
// el siguiente guardado la escribiría otra vez.
func TestNoSeArchivaLaBovedaQueSeEstaUsando(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if err := a.ArchivarProyecto(ref, true); err == nil {
		t.Fatal("ha archivado la bóveda que estaba abierta")
	}
	if _, err := os.Stat(rutaDeProyecto(ref)); err != nil {
		t.Error("y encima se ha llevado su fichero")
	}
}

// **Borrar pide la contraseña maestra**, como borrar la bóveda personal: es lo
// único irreversible que hay aquí y lo que se lleva son las contraseñas de un
// cliente entero.
func TestBorrarUnProyectoPideLaMaestra(t *testing.T) {
	a, _, ref := conUnProyecto(t)
	ruta := rutaDeProyecto(ref)

	if err := a.BorrarProyecto(ref, "esa no es"); err == nil {
		t.Fatal("ha borrado con una contraseña que no es")
	}
	if _, err := os.Stat(ruta); err != nil {
		t.Fatal("y aun así se ha llevado el fichero")
	}
	if l, _ := a.Proyectos(); len(l) != 1 {
		t.Fatal("y lo ha quitado de la lista")
	}

	if err := a.BorrarProyecto(ref, maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ruta); err == nil {
		t.Error("el fichero sigue ahí después de borrar")
	}
	if l, _ := a.Proyectos(); len(l) != 0 {
		t.Errorf("sigue en la lista: %+v", l)
	}
	// Y el registro tampoco lo nombra: lo que queda ahí sale como «huérfano».
	datos, _ := os.ReadFile(rutaDelRegistro())
	if strings.Contains(string(datos), ref) {
		t.Error("el registro sigue apuntando una bóveda que ya no existe")
	}
}

// **La identidad para compartir es la de la bóveda personal, también con un
// proyecto abierto** (ADR 0052, al planificarla).
//
// Hasta la 2.39.2 no lo era: `MiIdentidad` preguntaba a la bóveda **activa**, que
// con un proyecto delante es el proyecto. Y `Identidad()` crea la identidad si no
// hay, así que no era leer de más: **fabricaba una identidad dentro del proyecto y
// la publicaba como las llaves de la cuenta**. A partir de ahí, lo que te mandaran
// llegaba cifrado hacia una bóveda que puedes tener cerrada.
//
// Es la forma exacta del fallo que la ADR 0050 dejó avisado en `arrancarSincro` —y
// que allí sí se respeta— en el único sitio que no lo miraba. Lo encontró leer el
// código, no una prueba, y por eso esta prueba existe.
func TestLaIdentidadParaCompartirEsSiempreLaDeLaPersonal(t *testing.T) {
	a, _, ref := conUnProyecto(t)

	mia, err := a.MiIdentidad()
	if err != nil {
		t.Fatal(err)
	}
	if mia.Huella == "" {
		t.Fatal("la bóveda personal tiene que dar una huella")
	}

	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	conElProyecto, err := a.MiIdentidad()
	if err != nil {
		t.Fatal(err)
	}
	if conElProyecto.Huella != mia.Huella {
		t.Fatalf("con un proyecto abierto la huella es %s y la de la cuenta es %s: se está enseñando la identidad del proyecto",
			conElProyecto.Huella, mia.Huella)
	}

	// Y al volver sigue siendo la misma: lo de arriba no puede haber pasado por
	// haberle puesto a la personal la identidad del proyecto.
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	deVuelta, err := a.MiIdentidad()
	if err != nil {
		t.Fatal(err)
	}
	if deVuelta.Huella != mia.Huella {
		t.Fatalf("al volver, la huella de la cuenta ha cambiado a %s", deVuelta.Huella)
	}
}
