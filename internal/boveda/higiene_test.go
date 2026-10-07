package boveda

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func conEntradas(t *testing.T, es ...Entrada) *Boveda {
	t.Helper()
	b, _, err := Crear(t.TempDir()+"/h.esfinge", "una contraseña maestra larga de prueba")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range es {
		if err := b.Poner(e); err != nil {
			t.Fatal(err)
		}
	}
	return b
}

// **La higiene dice qué está mal sin decir ni un secreto**, y lo dice **en el mismo
// orden** cada vez.
//
// Lo del orden no es estética: un mapa de Go se recorre al azar, así que sin ordenar,
// dos llamadas iguales darían respuestas distintas — y a quien pregunta, que es un
// agente comparando, eso le parecería que algo ha cambiado en la bóveda.
func TestLaHigieneNoDiceSecretosYSaleSiempreIgual(t *testing.T) {
	compartida := "LA-MISMA-DE-TRES"
	// **Dos grupos y no uno**, que es lo que hace que esto vigile el orden **entre**
	// grupos: con uno solo, quitar la ordenación de los grupos deja la prueba en verde
	// —comprobado mutándola—, porque lo único que se ejercita es el orden de dentro.
	otraCompartida := "LA-MISMA-DE-DOS"
	b := conEntradas(t,
		Entrada{Tipo: TipoCredencial, Titulo: "A", Secreto: compartida, TOTP: "JBSWY3DPEHPK3PXP"},
		Entrada{Tipo: TipoCredencial, Titulo: "B", Secreto: compartida},
		Entrada{Tipo: TipoCredencial, Titulo: "C", Secreto: compartida},
		Entrada{Tipo: TipoCredencial, Titulo: "E", Secreto: otraCompartida},
		Entrada{Tipo: TipoCredencial, Titulo: "F", Secreto: otraCompartida},
		Entrada{Tipo: TipoCredencial, Titulo: "D", Secreto: "otra distinta"},
		Entrada{Tipo: TipoTarjeta, Titulo: "Vieja", Numero: "4111", Caduca: "2020-01"},
		Entrada{Tipo: TipoTarjeta, Titulo: "Buena", Numero: "4222", Caduca: "2099-01"},
	)
	ahora := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)

	h := b.Higiene(ahora)
	if len(h.Reutilizadas) != 2 {
		t.Fatalf("los grupos de reutilizadas son %+v y tenían que ser dos", h.Reutilizadas)
	}
	// La que no comparte con nadie **no** sale.
	if len(h.SinCodigo) != 5 {
		t.Errorf("sin código deberían ser cinco —A tiene—, y son %d", len(h.SinCodigo))
	}
	if len(h.Caducadas) != 1 {
		t.Errorf("las caducadas son %+v y tenía que ser una", h.Caducadas)
	}

	// **Ni un secreto en lo que sale**, sobre los bytes.
	crudo, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(crudo), compartida) || strings.Contains(string(crudo), otraCompartida) {
		t.Errorf("la higiene lleva la contraseña dentro: %s", crudo)
	}

	// **Y el mismo orden cada vez.** Veinte vueltas, porque el recorrido de un mapa
	// cambia entre llamadas: con una sola, esto pasaría con el orden roto.
	primero, _ := json.Marshal(h)
	for i := 0; i < 20; i++ {
		otro, _ := json.Marshal(b.Higiene(ahora))
		if string(otro) != string(primero) {
			t.Fatalf("la vuelta %d da otro orden:\n  %s\n  %s", i, primero, otro)
		}
	}
}

// Y lo que está en la papelera no cuenta: una contraseña borrada no está reutilizada.
func TestLaHigieneNoMiraLaPapelera(t *testing.T) {
	b := conEntradas(t,
		Entrada{Tipo: TipoCredencial, Titulo: "Viva", Secreto: "la misma"},
		Entrada{Tipo: TipoCredencial, Titulo: "Borrada", Secreto: "la misma"},
	)
	for _, e := range b.Buscar("Borrada") {
		if err := b.Borrar(e.ID); err != nil {
			t.Fatal(err)
		}
	}
	if h := b.Higiene(time.Now()); len(h.Reutilizadas) != 0 {
		t.Errorf("cuenta como reutilizada una que está en la papelera: %+v", h.Reutilizadas)
	}
}

// **Las marcas se calculan sobre la entrada entera**, y calcularlas después de vaciar
// no da error: da `false` en todo. Ésta es la prueba que lo dice.
func TestLasMarcasNecesitanLaEntradaEntera(t *testing.T) {
	e := Entrada{Tipo: TipoCredencial, Titulo: "X", Secreto: "algo", TOTP: "JBSWY3DPEHPK3PXP"}
	if m := e.Marcas(); !m.TieneSecreto || !m.TieneCodigo {
		t.Fatalf("sobre la entrada entera dice %+v", m)
	}
	if m := e.SinSecretos().Marcas(); m.TieneSecreto || m.TieneCodigo {
		t.Errorf("después de vaciar dice %+v, y tenía que decir que no hay nada: "+
			"es justo por eso por lo que BuscarConMarcas existe", m)
	}
}
