package app

// La huella del servidor de esta casa, que la ADR 0058 hace obligatoria antes de
// marcarle una bóveda (sobre la regla de la 0043).
//
// Lo que esto vigila es la mitad que **no** necesita servidor de cuentas: que lo
// confirmado se caiga cuando deja de significar lo que decía, y que sin ello el
// interruptor no pueda dar ningún acceso. Que la huella se compare contra la que
// publica el servidor en el momento de sellar lo recorre el paseo de dos cuentas.

import (
	"strings"
	"testing"
)

// **Lo que se guarda es cuál es la huella, no un sí/no, y esto es el porqué.**
//
// Con un booleano, cambiar el correo a otro servidor dejaría la marca puesta y el
// interruptor sellaría la clave de la bóveda hacia unas llaves que nadie ha visto en su
// vida, **sin que nadie se lo mencionara**. Es exactamente la lección de la ranura de
// Touch ID (ADR 0044), donde «una vez» tuvo que pasar a ser «una vez por bóveda».
func TestAlCambiarElCorreoDelServidorSeCaeLaHuellaConfirmada(t *testing.T) {
	a, _, _ := conReloj(t)

	p := a.VerPreferencias()
	p.CorreoDelServidor = "servidor@zeri.es"
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	if err := a.ajustes.PonerHuellaDelServidor("AAAA-BBBB", "servidor@zeri.es"); err != nil {
		t.Fatal(err)
	}
	if a.VerPreferencias().HuellaDelServidor != "AAAA-BBBB" {
		t.Fatal("no ha guardado la huella confirmada")
	}

	// Otro servidor: lo confirmado deja de valer.
	p = a.VerPreferencias()
	p.CorreoDelServidor = "otro@zeri.es"
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	if h := a.VerPreferencias().HuellaDelServidor; h != "" {
		t.Errorf("la huella del servidor de antes sigue puesta: %q", h)
	}
}

// Y la ventana **no manda la huella**, como no manda las carpetas de los diálogos: un
// guardado a medias no puede borrar lo que alguien comparó, y tampoco darlo por bueno.
//
// Esto es la otra cara de la regla del cero: ahí el cero conserva, y aquí la cadena
// vacía también — mientras el correo no cambie.
func TestLaVentanaNoMandaLaHuellaDelServidor(t *testing.T) {
	a, _, _ := conReloj(t)

	p := a.VerPreferencias()
	p.CorreoDelServidor = "servidor@zeri.es"
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	if err := a.ajustes.PonerHuellaDelServidor("AAAA-BBBB", "servidor@zeri.es"); err != nil {
		t.Fatal(err)
	}

	// Un guardado como el de la ventana, con la huella vacía porque no la manda.
	p = a.VerPreferencias()
	p.HuellaDelServidor = ""
	p.MinutosParaBloquear = 5
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	if h := a.VerPreferencias().HuellaDelServidor; h != "AAAA-BBBB" {
		t.Errorf("un guardado de la ventana se ha llevado la huella comparada: %q", h)
	}

	// Y tampoco la puede poner, que es lo que impide darla por buena sin comprobarla.
	p = a.VerPreferencias()
	p.HuellaDelServidor = "ME-LA-INVENTO"
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	if h := a.VerPreferencias().HuellaDelServidor; h != "AAAA-BBBB" {
		t.Errorf("la ventana ha podido escribir una huella: %q", h)
	}
}

// **Sin huella comparada no se da ningún acceso**, y se dice adónde ir.
//
// Y se niega **antes de salir a la red**: lo que hace falta para decidir se mira antes
// de pedirle nada a nadie, que es la regla de `ExportarLlaves`. Por eso esta prueba no
// necesita servidor de cuentas — y si alguien moviera la comprobación detrás de la
// llamada, se pondría roja por el error que no es.
func TestSinHuellaComparadaNoSeLeDaAccesoAlServidor(t *testing.T) {
	a, _, _ := conReloj(t)

	err := a.DarAccesoAlServidor()
	if err == nil {
		t.Fatal("ha dado acceso sin haber ningún servidor puesto")
	}
	if !strings.Contains(err.Error(), "Ajustes") {
		t.Errorf("sin servidor puesto, el error no manda a ninguna parte: %v", err)
	}

	p := a.VerPreferencias()
	p.CorreoDelServidor = "servidor@zeri.es"
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	err = a.DarAccesoAlServidor()
	if err == nil {
		t.Fatal("ha dado acceso con la huella sin comparar")
	}
	if !strings.Contains(err.Error(), "huella") || !strings.Contains(err.Error(), "Ajustes") {
		t.Errorf("el error no dice qué falta ni dónde se arregla: %v", err)
	}
}

// Y la huella se apunta **contra el correo que había cuando se miró**: entre que la
// pantalla la enseña y alguien pulsa «coincide» caben segundos, y en ese rato se puede
// haber cambiado el correo en la fila de arriba. Sin esto se apuntaría la huella de un
// servidor como si fuera la del otro.
func TestLaHuellaSeApuntaContraElCorreoQueHabia(t *testing.T) {
	a, _, _ := conReloj(t)

	p := a.VerPreferencias()
	p.CorreoDelServidor = "servidor@zeri.es"
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	if err := a.ajustes.PonerHuellaDelServidor("AAAA-BBBB", "elDeAntes@zeri.es"); err == nil {
		t.Fatal("ha apuntado la huella de un servidor contra el correo de otro")
	}
	if h := a.VerPreferencias().HuellaDelServidor; h != "" {
		t.Errorf("ha guardado algo: %q", h)
	}
}
