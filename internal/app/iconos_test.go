package app

import (
	"testing"

	"github.com/webcafeina/esfinge/internal/boveda"
)

// **Con la red apagada no se abre ni un socket**, y da igual lo que digan los
// ajustes: ESFINGE_SIN_RED es un freno, no una preferencia más.
//
// La prueba mira lo único que se puede mirar sin red: que el goteo ni siquiera
// arranca. Si arrancara, lo siguiente sería una petición.
func TestConLaRedApagadaNoSeBuscanIconos(t *testing.T) {
	t.Setenv("ESFINGE_SIN_RED", "1")
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Titulo: "Banco", Sitios: []string{"https://banco.es"},
	}); err != nil {
		t.Fatal(err)
	}

	// Con la red apagada no hay nada que hacer, así que no se toca la caché.
	a.buscarIconosSiProcede(t.Context())
	if len(a.bov.Iconos()) != 0 {
		t.Error("ha guardado algo con la red apagada")
	}
}

// Y con el ajuste apagado, tampoco.
func TestConElAjusteApagadoNoSeBuscanIconos(t *testing.T) {
	a, _, _ := conReloj(t)
	if err := a.GuardarPreferencias(Preferencias{DescargarIconos: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	a.buscarIconosSiProcede(t.Context())
	if len(a.bov.Iconos()) != 0 {
		t.Error("ha guardado algo con el ajuste apagado")
	}
}

// De la bóveda salen los anfitriones a los que preguntar, y **solo los que se
// pueden visitar**: una intranet o una dirección de la red de casa no se tocan.
func TestLoQueFaltaPorMirarDejaFueraLoQueNoSeVisita(t *testing.T) {
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	for _, sitio := range []string{
		"https://banco.es/login",
		"https://www.banco.es/otra",     // el mismo anfitrión no, el mismo no: www. cuenta aparte
		"https://intranet.empresa.local", // no se toca
		"http://192.168.1.1",             // tampoco
		"",                               // ni esto
	} {
		if err := a.GuardarEnBoveda(boveda.Entrada{
			Titulo: "x " + sitio, Sitios: []string{sitio},
		}); err != nil {
			t.Fatal(err)
		}
	}

	faltan := a.loQueFaltaPorMirar()
	for _, a := range faltan {
		if a == "intranet.empresa.local" || a == "192.168.1.1" {
			t.Errorf("va a preguntarle a %q", a)
		}
	}
	if len(faltan) != 2 {
		t.Errorf("esperaba banco.es y www.banco.es, y tengo %v", faltan)
	}
}
