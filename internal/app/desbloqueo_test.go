package app

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/llavero"
)

const maestraDePrueba = "una contraseña maestra bien larga"

// conLlaveroDeMentira monta la aplicación con un llavero que se puede mandar,
// porque aquí no hay Touch ID ni Windows Hello.
func conLlaveroDeMentira(t *testing.T) (*App, *llavero.DeMentira) {
	t.Helper()
	a, _ := nuevaDePrueba(t)
	l := &llavero.DeMentira{ComoSeLlama: "Touch ID"}
	a.llavero = l
	return a, l
}

// El camino entero: activar con la bóveda abierta, cerrar, abrir con el sistema
// y quitarlo. Y lo que no se puede perder de vista: **la maestra sigue abriendo**.
func TestDesbloquearConElSistema(t *testing.T) {
	a, l := conLlaveroDeMentira(t)

	if e := a.EstadoDelDesbloqueo(); !e.Hay || e.Nombre != "Touch ID" || e.Puesto {
		t.Fatalf("sin bóveda el estado es %+v", e)
	}
	if err := a.ActivarDesbloqueo(); !errors.Is(err, boveda.ErrCerrada) {
		t.Fatalf("se puede activar sin la bóveda abierta: %v", err)
	}

	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{Tipo: boveda.TipoCredencial, Titulo: "Banco", Secreto: "la del banco"}); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivarDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); !e.Puesto {
		t.Fatalf("activado, el estado dice %+v", e)
	}

	a.CerrarBoveda()
	if a.boveda() != nil {
		t.Fatal("la bóveda sigue abierta")
	}
	antes := l.Lecturas
	if err := a.AbrirBovedaConElSistema(); err != nil {
		t.Fatal(err)
	}
	if l.Lecturas != antes+1 {
		t.Fatalf("no ha pedido el secreto al sistema (%d lecturas)", l.Lecturas-antes)
	}
	if e, _ := a.BuscarEnBoveda("Banco"); len(e) != 1 {
		t.Fatalf("abierta con el sistema hay %d entradas", len(e))
	}

	// **Y la maestra sigue abriendo**: la ranura del sistema nunca es la única.
	a.CerrarBoveda()
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatalf("la maestra ha dejado de abrir: %v", err)
	}

	if err := a.QuitarDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); e.Puesto {
		t.Fatal("sigue puesto después de quitarlo")
	}
	a.CerrarBoveda()
	if err := a.AbrirBovedaConElSistema(); !errors.Is(err, boveda.ErrSinRanuraDelSistema) {
		t.Fatalf("quitado, abrir con el sistema dice %v", err)
	}
}

// **Cancelar el diálogo no es un fallo**, y sobre todo no puede dejar la bóveda
// a medias: se vuelve a la contraseña maestra y ahí sigue todo.
func TestSiNoReconoceLaHuellaSeVuelveALaMaestra(t *testing.T) {
	a, l := conLlaveroDeMentira(t)
	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivarDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	a.CerrarBoveda()

	l.DiceQueNo = true
	err := a.AbrirBovedaConElSistema()
	if !errors.Is(err, llavero.ErrNoQuiso) {
		t.Fatalf("cancelar da %v", err)
	}
	if a.boveda() != nil {
		t.Fatal("la bóveda se ha quedado abierta tras cancelar")
	}
	// Y el desbloqueo sigue puesto: cancelar no lo desactiva.
	if e := a.EstadoDelDesbloqueo(); !e.Puesto {
		t.Fatal("cancelar ha quitado el desbloqueo")
	}
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatalf("la maestra no abre después de cancelar: %v", err)
	}
}

// **Si lo que guarda el sistema ya no abre, se limpia y se pide la maestra.**
// Pasa al restaurar la bóveda de una copia anterior: el secreto sigue en el
// llavero y su sobre ya no está.
func TestUnSecretoQueYaNoAbreSeOlvida(t *testing.T) {
	a, l := conLlaveroDeMentira(t)
	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivarDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	// Se le cambia el secreto por la espalda, como si la bóveda fuera otra.
	otro, _ := boveda.SecretoDelSistema()
	if err := l.Guardar("com.webcafeina.esfinge.boveda", otro); err != nil {
		t.Fatal(err)
	}
	a.CerrarBoveda()

	if err := a.AbrirBovedaConElSistema(); !errors.Is(err, boveda.ErrSinRanuraDelSistema) {
		t.Fatalf("con un secreto que no abre dice %v", err)
	}
	// Y lo que quedó suelto en el llavero se ha ido, para no volver a ofrecerlo.
	if _, err := l.Leer("com.webcafeina.esfinge.boveda", ""); !errors.Is(err, llavero.ErrNoEsta) {
		t.Fatalf("el secreto inservible sigue en el llavero: %v", err)
	}
}

// En un equipo sin biometría no se ofrece nada, y activarlo lo dice claro en vez
// de dejar un botón que no hace nada.
func TestSinBiometriaNoSeOfreceNada(t *testing.T) {
	a, l := conLlaveroDeMentira(t)
	l.NoHay = true
	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); e.Hay {
		t.Fatalf("dice que hay biometría donde no la hay: %+v", e)
	}
	if err := a.ActivarDesbloqueo(); !errors.Is(err, llavero.ErrNoHay) {
		t.Fatalf("activar sin biometría dice %v", err)
	}
	if strings.TrimSpace(llavero.ErrNoHay.Error()) == "" {
		t.Fatal("y el mensaje no puede estar vacío: lo lee alguien")
	}
}

// **La ranura del sistema no se sube nunca**, y esto lo mira desde la aplicación
// entera y no solo desde la bóveda: es la pieza que, si se rompiera, le daría al
// servidor una segunda puerta a la bóveda de todos los equipos.
func TestLaRanuraDelSistemaNoSaleDeEsteEquipo(t *testing.T) {
	a, _ := conLlaveroDeMentira(t)
	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivarDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	datos, _, err := a.boveda().PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(datos), boveda.RanuraDelSistema) {
		t.Fatal("la ranura del sistema viaja hacia el servidor")
	}
}

// **El aviso de «acabas de actualizar» sale una vez por versión, y solo esa vez.**
//
// Sin firmar, cada versión de Esfinge es un binario nuevo y el llavero de macOS
// vuelve a pedir la contraseña del equipo la primera vez que se pone el dedo
// (ADR 0044, decisión 3, contestada en el Mac del cliente el 2026-09-25). La
// pantalla lo avisa antes de que salga, y para eso tiene que saber **si esta
// versión ya consiguió abrir**: no basta con mirar qué versión corre.
//
// Aquí no hay Touch ID, así que lo que se comprueba es la contabilidad: cuándo se
// enciende la marca, cuándo se apaga, y que no se encienda donde no hay nada que
// avisar.
func TestElAvisoDeActualizarSaleUnaVezPorVersion(t *testing.T) {
	a, _ := conLlaveroDeMentira(t)
	a.version = "2.27.6"

	// Sin la ranura puesta no hay huella que vaya a salir, así que tampoco hay
	// diálogo del que avisar.
	if e := a.EstadoDelDesbloqueo(); e.TrasActualizar {
		t.Fatalf("avisa sin tener el desbloqueo puesto: %+v", e)
	}

	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivarDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	a.CerrarBoveda()

	// Puesto y sin haber abierto nunca con esta versión: se avisa.
	if e := a.EstadoDelDesbloqueo(); !e.TrasActualizar {
		t.Fatalf("no avisa con el desbloqueo recién puesto: %+v", e)
	}

	// Abrir con el sistema es lo que demuestra que el permiso está dado.
	if err := a.AbrirBovedaConElSistema(); err != nil {
		t.Fatal(err)
	}
	a.CerrarBoveda()
	if e := a.EstadoDelDesbloqueo(); e.TrasActualizar {
		t.Fatalf("sigue avisando después de abrir con el sistema: %+v", e)
	}

	// Y al actualizar vuelve, que es justo lo que pasa en un Mac: el permiso va
	// atado al binario y el binario es otro.
	a.version = "2.28.0"
	if e := a.EstadoDelDesbloqueo(); !e.TrasActualizar {
		t.Fatalf("no vuelve a avisar tras actualizar: %+v", e)
	}
}

// **La oferta es una por bóveda, no una por equipo.**
//
// La primera versión guardaba un sí/no y se razonó entre equipos: haberlo
// descartado en el portátil no dice nada del de la oficina. Le faltaba el otro
// caso, y el cliente lo encontró dos veces el mismo día (2026-09-28) —creando una
// cuenta nueva, y luego entrando en la suya—: **la ranura del sistema es de cada
// bóveda**, así que con un sí/no una bóveda distinta en el mismo equipo nacía sin
// desbloqueo y **sin que nadie volviera a mencionarlo**.
//
// Entrar en una cuenta es el caso que más importa, porque **trae otra bóveda** y
// la ranura no viaja con ella: quien tenía Touch ID se quedaba sin él y sin
// explicación.
func TestLaOfertaDelDesbloqueoEsUnaPorBoveda(t *testing.T) {
	a, _ := conLlaveroDeMentira(t)

	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); !e.Sugerir {
		t.Fatalf("una bóveda recién creada tiene que ofrecerlo: %+v", e)
	}

	// Se contesta «ahora no»: para ésta ya no se vuelve a ofrecer.
	if err := a.NoOfrecerElDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); e.Sugerir {
		t.Fatalf("se contestó y sigue ofreciéndolo: %+v", e)
	}

	// **Y ahora llega otra bóveda al mismo equipo**, que es lo que pasa al entrar
	// en una cuenta. Se simula como lo hace la aplicación: la de antes se aparta y
	// en su sitio queda una nueva.
	a.CerrarBoveda()
	ruta := rutaBovedaPrincipal()
	if err := os.Rename(ruta, ruta+".apartada"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); !e.Sugerir {
		t.Fatalf("es otra bóveda y no lo ofrece: %+v", e)
	}

	// Y activarlo también cuenta como contestar, sin pasar por «ahora no».
	if err := a.ActivarDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelDesbloqueo(); e.Sugerir {
		t.Fatalf("está puesto y sigue ofreciéndolo: %+v", e)
	}
}

// **La ranura se queda sin su llave, y eso se arregla solo en cuanto se abre.**
//
// Pasa de verdad y no hace falta que nadie toque el llavero por fuera: el secreto es
// **uno por máquina** y las bóvedas son varias, así que basta con entrar en una cuenta
// en este equipo —la bóveda que hubiera se aparta (ADR 0039), la nueva no abre con ese
// secreto, y el caso de arriba lo borra para no ofrecer algo que no funciona— y volver
// luego a la primera. **Lo encontró el cliente** al devolver su segundo Mac tras las
// pruebas de la ADR 0052 (2026-10-07): su bóveda volvió entera y la huella decía «El
// sistema ya no guarda esa llave», con la ranura dentro y sin forma de salir de ahí más
// que yendo a Ajustes a apagarlo y encenderlo.
//
// Lo que se comprueba es lo que le faltaba: que **lo que se dice sirva para algo**, que
// la ranura inservible **se vaya** en vez de volver a fallar en cada intento, y que **se
// vuelva a ofrecer**, porque quitarla en silencio deja a quien tenía Touch ID sin él y
// sin que nadie se lo mencione.
func TestLaRanuraQueSeQuedaSinLlaveSeLimpiaAlAbrir(t *testing.T) {
	a, l := conLlaveroDeMentira(t)
	if _, err := a.CrearBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivarDesbloqueo(); err != nil {
		t.Fatal(err)
	}
	b := a.boveda()
	if b == nil {
		t.Fatal("la bóveda tenía que estar abierta")
	}
	id := b.ID()
	// Se le quita la llave por la espalda, que es lo que hace otra bóveda del mismo
	// equipo al pasar por ahí.
	if err := l.Borrar(idEnElLlavero); err != nil {
		t.Fatal(err)
	}
	a.CerrarBoveda()

	// **Lo que se dice tiene que decir qué hacer.**
	err := a.AbrirBovedaConElSistema()
	if !errors.Is(err, ErrLlaveQueYaNoEsta) {
		t.Fatalf("con la ranura sin su llave dice %v", err)
	}
	if !strings.Contains(err.Error(), "contraseña maestra") {
		t.Errorf("el mensaje no dice qué hacer: %q", err)
	}

	// Y la ranura sigue ahí, porque con la bóveda cerrada no se puede tocar.
	if !boveda.RanuraDelSistemaEn(rutaBovedaPrincipal()) {
		t.Fatal("la ranura se ha quitado con la bóveda cerrada, que es lo que no se puede hacer")
	}

	// **Al abrir con la maestra se limpia.**
	if err := a.AbrirBoveda(maestraDePrueba); err != nil {
		t.Fatal(err)
	}
	if boveda.RanuraDelSistemaEn(rutaBovedaPrincipal()) {
		t.Fatal("la ranura sin llave sigue puesta tras abrir: volvería a fallar en cada intento")
	}
	// **Y se vuelve a ofrecer**, que es la mitad que importa: quitarla en silencio deja
	// sin Touch ID a quien lo tenía sin que nadie se lo mencione.
	if p := a.VerPreferencias(); p.DesbloqueoSugeridoPara == id {
		t.Error("la bóveda se ha quedado marcada como «ya se le ofreció», así que no se vuelve a ofrecer")
	}
}
