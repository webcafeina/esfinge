package app

import (
	"testing"
	"time"
)

// conReloj monta una aplicación con el tiempo en la mano.
//
// Sin esto habría que esperar quince minutos de verdad para comprobar que
// bloquea, y una prueba que tarda quince minutos no se ejecuta nunca.
func conReloj(t *testing.T) (*App, *sistemaFalso, *time.Time) {
	t.Helper()
	casa := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", casa)
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)

	s := &sistemaFalso{}
	a := Nueva("2.0.0", s)
	ahora := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	a.vig.ahora = func() time.Time { return ahora }
	a.vig.ultimaActividad = ahora
	return a, s, &ahora
}

func TestElPortapapelesSeBorraSolo(t *testing.T) {
	a, s, ahora := conReloj(t)

	if err := a.Copiar("una contraseña"); err != nil {
		t.Fatal(err)
	}
	if s.verPortapapeles() != "una contraseña" {
		t.Fatalf("no ha copiado: %q", s.verPortapapeles())
	}

	// Antes del plazo no se toca.
	*ahora = ahora.Add(10 * time.Second)
	a.repasar()
	if s.verPortapapeles() != "una contraseña" {
		t.Error("lo ha borrado antes de tiempo")
	}

	// Pasado el plazo, fuera.
	*ahora = ahora.Add(portapapelesPorDefecto)
	a.repasar()
	if s.verPortapapeles() != "" {
		t.Errorf("sigue ahí: %q", s.verPortapapeles())
	}
}

// **El detalle que hace esto aceptable.** Si la persona ha copiado otra cosa
// después, borrar sería quitarle lo suyo por proteger un secreto que ya no está.
func TestNoPisaLoQueSeHayaCopiadoDespues(t *testing.T) {
	a, s, ahora := conReloj(t)

	a.Copiar("una contraseña")
	// Alguien copia otra cosa, de otro programa.
	s.PonerEnPortapapeles("la dirección de una web")

	*ahora = ahora.Add(portapapelesPorDefecto + time.Second)
	a.repasar()

	if s.verPortapapeles() != "la dirección de una web" {
		t.Errorf("ha borrado lo que había copiado la persona: %q", s.verPortapapeles())
	}
}

// Si no se puede leer el portapapeles, no se puede saber si sigue siendo
// nuestro, así que no se borra: equivocarse aquí es quitarle a alguien lo suyo.
func TestSiNoSePuedeLeerNoSeBorraNada(t *testing.T) {
	a, s, ahora := conReloj(t)
	a.Copiar("una contraseña")
	s.mu.Lock()
	s.leerFalla = true
	s.mu.Unlock()

	*ahora = ahora.Add(portapapelesPorDefecto + time.Second)
	a.repasar()

	if s.verPortapapeles() != "una contraseña" {
		t.Error("ha borrado sin poder comprobar que era suyo")
	}
}

// La lección que este proyecto ya pagó con la comprobación de actualizaciones:
// **un temporizador único no vale**. Un portátil suspendido ocho horas tiene que
// aparecer bloqueado al despertar.
func TestBloqueaTrasSuspenderElPortatil(t *testing.T) {
	a, _, ahora := conReloj(t)

	// Antes del plazo, nada.
	*ahora = ahora.Add(bloqueoPorDefecto - time.Minute)
	if a.vig.tocaBloquear() {
		t.Error("ha bloqueado antes de tiempo")
	}

	// Un salto de ocho horas, como al despertar de suspender.
	*ahora = ahora.Add(8 * time.Hour)
	if !a.vig.tocaBloquear() {
		t.Error("tras ocho horas suspendido sigue sin bloquear")
	}
}

func TestLaActividadAplazaElBloqueo(t *testing.T) {
	a, _, ahora := conReloj(t)

	*ahora = ahora.Add(bloqueoPorDefecto - time.Minute)
	a.Actividad()

	*ahora = ahora.Add(2 * time.Minute)
	if a.vig.tocaBloquear() {
		t.Error("la actividad no ha aplazado el bloqueo")
	}

	*ahora = ahora.Add(bloqueoPorDefecto)
	if !a.vig.tocaBloquear() {
		t.Error("y pasado el plazo desde la última actividad, no bloquea")
	}
}

// «Nunca» es una opción legítima de Ajustes, y tiene que significar nunca.
func TestSePuedeApagarElBloqueo(t *testing.T) {
	a, _, ahora := conReloj(t)
	a.vig.espera = 0

	*ahora = ahora.Add(30 * 24 * time.Hour)
	if a.vig.tocaBloquear() {
		t.Error("con el bloqueo apagado ha bloqueado igual")
	}
}
