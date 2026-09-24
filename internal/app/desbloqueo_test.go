package app

import (
	"errors"
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
