package boveda

import (
	"strings"
	"testing"
)

// La identidad nace una vez y no cambia: es lo que sostiene que lo que te
// mandaron ayer se siga abriendo mañana (ADR 0043).
func TestLaIdentidadSeCreaUnaVezYSobreviveAlDisco(t *testing.T) {
	b, _, ruta := nueva(t)
	una, err := b.Identidad()
	if err != nil {
		t.Fatal(err)
	}
	otra, err := b.Identidad()
	if err != nil {
		t.Fatal(err)
	}
	if una.Huella != otra.Huella {
		t.Fatalf("pedirla dos veces da dos identidades: %s y %s", una.Huella, otra.Huella)
	}
	if len(una.Cifrado) != 32 || len(una.Firma) != 32 {
		t.Fatalf("llaves de %d y %d bytes", len(una.Cifrado), len(una.Firma))
	}
	if una.Suite != Suite {
		t.Fatalf("suite %q", una.Suite)
	}
	b.Cerrar()

	// Y al volver a abrir el fichero es la misma, que es lo que importa de verdad.
	otraVez, err := Abrir(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	defer otraVez.Cerrar()
	vuelta, err := otraVez.Identidad()
	if err != nil {
		t.Fatal(err)
	}
	if vuelta.Huella != una.Huella {
		t.Fatalf("tras cerrar y abrir, otra identidad: %s ≠ %s", vuelta.Huella, una.Huella)
	}
}

// La huella es lo único que protege el primer envío, así que tiene que cambiar
// con cualquiera de las tres cosas que cubre y leerse sin confundir caracteres.
func TestLaHuellaSePuedeLeerEnVozAltaYCubreLoQueDice(t *testing.T) {
	cifrado := make([]byte, 32)
	firma := make([]byte, 32)
	base := HuellaDeIdentidad(Suite, cifrado, firma)

	if n := len(strings.ReplaceAll(base, "-", "")); n != 28 {
		t.Fatalf("la huella tiene %d símbolos", n)
	}
	for _, r := range strings.ReplaceAll(base, "-", "") {
		if strings.ContainsRune("ILOU", r) {
			t.Fatalf("la huella lleva un carácter que se confunde al dictarla: %q en %s", r, base)
		}
		if !strings.ContainsRune(alfabeto, r) {
			t.Fatalf("carácter fuera del alfabeto: %q", r)
		}
	}

	otraFirma := make([]byte, 32)
	otraFirma[31] = 1
	if HuellaDeIdentidad(Suite, cifrado, otraFirma) == base {
		t.Fatal("cambiar la llave de firma no cambia la huella")
	}
	otroCifrado := make([]byte, 32)
	otroCifrado[0] = 1
	if HuellaDeIdentidad(Suite, otroCifrado, firma) == base {
		t.Fatal("cambiar la llave de cifrado no cambia la huella")
	}
	if HuellaDeIdentidad("otra suite", cifrado, firma) == base {
		t.Fatal("cambiar la suite no cambia la huella")
	}
}

// **Y si dos equipos crean una cada uno antes de verse, los dos se quedan con la
// misma.** Con una identidad por equipo, lo que a uno le mandaran no lo abriría el
// otro.
func TestDosIdentidadesSeFundenSiempreALaMisma(t *testing.T) {
	vieja := &identidad{Semilla: "AAAA", Creada: "2026-01-01T00:00:00Z", Suite: Suite}
	nueva := &identidad{Semilla: "BBBB", Creada: "2026-06-01T00:00:00Z", Suite: Suite}

	if fundirIdentidad(vieja, nueva) != vieja || fundirIdentidad(nueva, vieja) != vieja {
		t.Fatal("no gana la más antigua, o no gana la misma en los dos sentidos")
	}
	// Empatadas en fecha, decide la semilla, y también en los dos sentidos.
	a := &identidad{Semilla: "AAAA", Creada: "2026-01-01T00:00:00Z"}
	z := &identidad{Semilla: "ZZZZ", Creada: "2026-01-01T00:00:00Z"}
	if fundirIdentidad(a, z) != a || fundirIdentidad(z, a) != a {
		t.Fatal("el desempate por semilla no es simétrico")
	}
	if fundirIdentidad(nil, z) != z || fundirIdentidad(z, nil) != z {
		t.Fatal("con una sola identidad tiene que quedar ésa")
	}
}

// Y la prueba que de verdad importa de la fusión: dos equipos de la misma cuenta
// acaban con la misma identidad aunque cada uno se la haya creado por su cuenta.
func TestDosEquiposAcabanConLaMismaIdentidad(t *testing.T) {
	a, b, s := dosEquipos(t)
	suya, err := a.b.Identidad()
	if err != nil {
		t.Fatal(err)
	}
	laOtra, err := b.b.Identidad()
	if err != nil {
		t.Fatal(err)
	}
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	a.sincronizar(t, s)

	enA, _ := a.b.Identidad()
	enB, _ := b.b.Identidad()
	if enA.Huella != enB.Huella {
		t.Fatalf("cada equipo con su identidad: %s ≠ %s", enA.Huella, enB.Huella)
	}
	if enA.Huella != suya.Huella && enA.Huella != laOtra.Huella {
		t.Fatal("la identidad que queda no es ninguna de las dos que había")
	}
}
