package app

import (
	"os"
	"path/filepath"
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

// Los dos plazos valen **desde ya**: quien acaba de bajar el bloqueo a un minuto
// porque se levanta de la mesa no puede tener que reiniciar para que sirva.
func TestLosPlazosDeAjustesValenEnElActo(t *testing.T) {
	a, s, ahora := conReloj(t)

	err := a.GuardarPreferencias(Preferencias{
		BuscarActualizaciones:  true,
		MinutosParaBloquear:    1,
		SegundosDePortapapeles: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	a.Copiar("una contraseña")
	*ahora = ahora.Add(6 * time.Second)
	a.repasar()
	if s.verPortapapeles() != "" {
		t.Errorf("el plazo nuevo del portapapeles no se ha aplicado: %q", s.verPortapapeles())
	}

	*ahora = ahora.Add(2 * time.Minute)
	if !a.vig.tocaBloquear() {
		t.Error("el plazo nuevo del bloqueo no se ha aplicado")
	}
	if m := a.EstadoBoveda().MinutosParaBloquear; m != 1 {
		t.Errorf("la ventana vería %d minutos y no el que se guardó", m)
	}
}

// Con el borrado apagado no se borra **y no se guarda el secreto**: lo que no se
// va a borrar no hace falta recordarlo.
func TestSePuedeApagarElBorradoDelPortapapeles(t *testing.T) {
	a, s, ahora := conReloj(t)

	err := a.GuardarPreferencias(Preferencias{SegundosDePortapapeles: Nunca})
	if err != nil {
		t.Fatal(err)
	}

	a.Copiar("una contraseña")
	if a.vig.loCopiado != "" {
		t.Error("se ha quedado con el secreto en memoria sin necesitarlo")
	}

	*ahora = ahora.Add(time.Hour)
	a.repasar()
	if s.verPortapapeles() != "una contraseña" {
		t.Errorf("lo ha borrado con el borrado apagado: %q", s.verPortapapeles())
	}
}

// **El cero es «no lo he dicho», y esto es lo que lo vigila.**
//
// GuardarPreferencias recibe el objeto entero, así que un guardado a medias
// —cambiar el interruptor de las actualizaciones y mandar solo eso— llega con
// los dos plazos a cero. Si el cero significara «nunca», ese descuido apagaría
// el bloqueo de la bóveda y el borrado del portapapeles sin que nadie lo pidiera
// y sin que se notara. Lo encontró una prueba de interfaz, no ésta.
func TestUnGuardadoAMediasNoApagaLosRelojes(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", casa)
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)

	a := AbrirAjustes()
	if err := a.Guardar(Preferencias{MinutosParaBloquear: 5, SegundosDePortapapeles: 10}); err != nil {
		t.Fatal(err)
	}
	// Y ahora alguien guarda solo el interruptor, como hace la ventana al
	// apagar la comprobación de actualizaciones.
	if err := a.Guardar(Preferencias{BuscarActualizaciones: true}); err != nil {
		t.Fatal(err)
	}

	p := a.Ver()
	if p.MinutosParaBloquear != 5 || p.SegundosDePortapapeles != 10 {
		t.Errorf("un guardado a medias se ha llevado los plazos por delante: %+v", p)
	}
}

// Los plazos llegan de la ventana, así que llega lo que haya escrito quien esté
// al otro lado. Y un fichero de preferencias de antes de la bóveda no tiene los
// campos: ahí no valen los ceros de Go, valen los de siempre.
func TestLosPlazosSeRecortanYLosViejosSeQuedanConLoDeSiempre(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", casa)
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)

	a := AbrirAjustes()
	if p := a.Ver(); p.MinutosParaBloquear != 15 || p.SegundosDePortapapeles != 30 {
		t.Fatalf("de partida: %+v", p)
	}

	// Un fichero escrito por una versión anterior, sin los campos nuevos.
	os.MkdirAll(filepath.Dir(a.ruta), 0o700)
	os.WriteFile(a.ruta, []byte(`{"buscarActualizaciones":false}`), 0o600)
	if p := AbrirAjustes().Ver(); p.MinutosParaBloquear != 15 || p.SegundosDePortapapeles != 30 {
		t.Errorf("un fichero de antes de la bóveda deja la bóveda sin bloqueo: %+v", p)
	}

	for _, c := range []struct {
		mete              Preferencias
		minutos, segundos int // cero: se espera que se conserve lo que había
	}{
		{Preferencias{MinutosParaBloquear: -3}, Nunca, 0},
		{Preferencias{MinutosParaBloquear: 9999}, 480, 0},
		{Preferencias{SegundosDePortapapeles: 1}, 0, 5},
		{Preferencias{SegundosDePortapapeles: 9999}, 0, 600},
		{Preferencias{SegundosDePortapapeles: Nunca}, 0, Nunca},
	} {
		// Se parte de un fichero conocido: lo que no se dice se conserva, así que
		// hay que saber de qué se parte para poder afirmar qué se conservó.
		partida := Preferencias{MinutosParaBloquear: 15, SegundosDePortapapeles: 30}
		a := AbrirAjustes()
		if err := a.Guardar(partida); err != nil {
			t.Fatal(err)
		}
		if err := a.Guardar(c.mete); err != nil {
			t.Fatal(err)
		}

		quiero := partida
		if c.minutos != 0 {
			quiero.MinutosParaBloquear = c.minutos
		}
		if c.segundos != 0 {
			quiero.SegundosDePortapapeles = c.segundos
		}
		p := AbrirAjustes().Ver()
		if p.MinutosParaBloquear != quiero.MinutosParaBloquear ||
			p.SegundosDePortapapeles != quiero.SegundosDePortapapeles {
			t.Errorf("de %+v salió %d min y %d s, y quería %d y %d", c.mete,
				p.MinutosParaBloquear, p.SegundosDePortapapeles,
				quiero.MinutosParaBloquear, quiero.SegundosDePortapapeles)
		}
	}
}
