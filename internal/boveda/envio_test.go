package boveda

import (
	"errors"
	"strings"
	"testing"
)

// Dos bóvedas: una manda y la otra abre. Es la prueba de ida y vuelta del sobre.
func dosBovedas(t *testing.T) (*Boveda, *Boveda) {
	t.Helper()
	a, _, _ := nueva(t)
	b, _, _ := nueva(t)
	if _, err := a.Identidad(); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Identidad(); err != nil {
		t.Fatal(err)
	}
	return a, b
}

func TestUnEnvioLoAbreSuDestinatarioYNadieMas(t *testing.T) {
	a, b := dosBovedas(t)
	c, _, _ := nueva(t)
	if _, err := c.Identidad(); err != nil {
		t.Fatal(err)
	}
	suya, _ := b.Identidad()
	mia, _ := a.Identidad()

	e := Entrada{ID: "0123456789abcdef0123456789abcdef", Tipo: TipoCredencial, Titulo: "Banco", Usuario: "ana",
		Secreto: "la contraseña", Sitios: []string{"banco.com"}, Historial: []Antigua{{Secreto: "la vieja", Hasta: "2026-01-01T00:00:00Z"}}}
	sobre, err := a.MandarEntrada(e, suya)
	if err != nil {
		t.Fatal(err)
	}

	recibida, de, err := b.AbrirEnvio(sobre)
	if err != nil {
		t.Fatal(err)
	}
	if recibida.Secreto != "la contraseña" || recibida.Titulo != "Banco" {
		t.Fatalf("lo recibido no es lo mandado: %+v", recibida)
	}
	// **Copia, no la misma entrada**: sin identificador y sin historial.
	if recibida.ID != "" {
		t.Fatalf("el envío lleva el identificador de la entrada del que manda: %q", recibida.ID)
	}
	if len(recibida.Historial) != 0 {
		t.Fatal("el envío lleva el historial de contraseñas de quien manda")
	}
	// Y se sabe de quién es, con su huella para comparar.
	if de.Huella != mia.Huella {
		t.Fatalf("la huella del que manda es %s y se esperaba %s", de.Huella, mia.Huella)
	}

	// Un tercero con el sobre en la mano no saca nada.
	if _, _, err := c.AbrirEnvio(sobre); !errors.Is(err, ErrSobreDeOtro) {
		t.Fatalf("otra bóveda abre el sobre, o falla por otra cosa: %v", err)
	}
}

// Tocar el sobre por fuera tiene que romperlo: va como datos autenticados y
// además firmado.
func TestUnEnvioManipuladoNoSeAbre(t *testing.T) {
	a, b := dosBovedas(t)
	suya, _ := b.Identidad()
	sobre, err := a.MandarEntrada(Entrada{Titulo: "X", Secreto: "s"}, suya)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("el cuerpo", func(t *testing.T) {
		malo := sobre
		malo.Cuerpo = append([]byte(nil), sobre.Cuerpo...)
		malo.Cuerpo[0] ^= 1
		if _, _, err := b.AbrirEnvio(malo); err == nil {
			t.Fatal("se abre con el cuerpo cambiado")
		}
	})
	t.Run("de quién dice venir", func(t *testing.T) {
		malo := sobre
		otra, _, _ := nueva(t)
		suplantador, _ := otra.Identidad()
		malo.De = EnvioDe{Cifrado: suplantador.Cifrado, Firma: suplantador.Firma}
		if _, _, err := b.AbrirEnvio(malo); !errors.Is(err, ErrFirmaDelEnvio) {
			t.Fatalf("se acepta un remitente cambiado: %v", err)
		}
	})
	t.Run("la firma", func(t *testing.T) {
		malo := sobre
		malo.Firma = append([]byte(nil), sobre.Firma...)
		malo.Firma[0] ^= 1
		if _, _, err := b.AbrirEnvio(malo); !errors.Is(err, ErrFirmaDelEnvio) {
			t.Fatalf("se acepta una firma que no cuadra: %v", err)
		}
	})
	t.Run("una versión de mañana", func(t *testing.T) {
		malo := sobre
		malo.Version = VersionDeEnvio + 1
		if _, _, err := b.AbrirEnvio(malo); !errors.Is(err, ErrEnvioNuevo) {
			t.Fatalf("no avisa de que viene de una versión más nueva: %v", err)
		}
	})
}

// Y sin identidad no se manda: mejor decirlo que mandar algo que nadie puede
// atribuir.
func TestSinIdentidadNoSeManda(t *testing.T) {
	a, _, _ := nueva(t)
	b, _, _ := nueva(t)
	suya, err := b.Identidad()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.MandarEntrada(Entrada{Titulo: "X"}, suya); !errors.Is(err, errSinIdentidad) {
		t.Fatalf("manda sin identidad: %v", err)
	}
	if _, err := a.Identidad(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.MandarEntrada(Entrada{Titulo: "X"}, suya); err != nil {
		t.Fatalf("con identidad tendría que mandar: %v", err)
	}
}

// El sobre no puede llevar la contraseña a la vista en ninguno de sus campos de
// fuera, que son los que el servidor ve.
func TestElSobreNoEnsenaNadaPorFuera(t *testing.T) {
	a, b := dosBovedas(t)
	suya, _ := b.Identidad()
	sobre, err := a.MandarEntrada(Entrada{Titulo: "Banco secreto", Usuario: "ana@ejemplo.com", Secreto: "contraseña-larguísima"}, suya)
	if err != nil {
		t.Fatal(err)
	}
	fuera := string(loQueSeFirma(sobre))
	for _, prohibido := range []string{"contraseña-larguísima", "Banco secreto", "ana@ejemplo.com"} {
		if strings.Contains(fuera, prohibido) {
			t.Fatalf("el sobre enseña %q por fuera: %s", prohibido, fuera)
		}
	}
}
