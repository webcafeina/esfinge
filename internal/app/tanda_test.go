package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// El tope existe para no pisarse ni comerse la memoria: cada derivación usa ya
// cuatro hilos por dentro y 64 MiB mientras dura.
func TestElTopeDeParalelismoEsRazonable(t *testing.T) {
	n := cuantosALaVez()
	if n < 1 {
		t.Errorf("nunca puede ser menos de uno, y es %d", n)
	}
	if n > 4 {
		t.Errorf("nunca puede pasar de cuatro, y es %d", n)
	}
	if nucleos := runtime.NumCPU(); n > nucleos {
		t.Errorf("hay %d núcleos y quiere %d a la vez", nucleos, n)
	}
}

// Terminan desordenados, pero la lista que se ve tiene que corresponderse con la
// que se soltó. Sin esto, el nombre de un fichero acabaría junto al resultado de
// otro, que es la peor forma de equivocarse aquí.
func TestUnaTandaConservaElOrdenDeEntrada(t *testing.T) {
	a, s := nuevaDePrueba(t)
	dir := t.TempDir()

	const cuantos = 20
	var rutas []string
	for i := 0; i < cuantos; i++ {
		ruta := filepath.Join(dir, fmt.Sprintf("secreto-%02d.env", i))
		if err := os.WriteFile(ruta, []byte(fmt.Sprintf("N=%d\n", i)), 0o600); err != nil {
			t.Fatal(err)
		}
		rutas = append(rutas, ruta)
	}

	out, err := a.CifrarFicheros(rutas, "una clave")
	if err != nil {
		t.Fatalf("CifrarFicheros: %v", err)
	}
	if len(out) != cuantos {
		t.Fatalf("quiero %d resultados, tengo %d", cuantos, len(out))
	}
	for i, r := range out {
		if r.Error != "" {
			t.Errorf("%s: %s", r.Origen, r.Error)
			continue
		}
		if r.Origen != rutas[i] {
			t.Errorf("el hueco %d lleva %q y le tocaba %q", i, r.Origen, rutas[i])
		}
	}

	// Y el progreso llega hasta el final: uno por fichero terminado, más el que
	// cierra la tanda.
	if len(s.avisos) != cuantos+1 {
		t.Errorf("quiero %d avisos, tengo %d", cuantos+1, len(s.avisos))
	}
	visto := make(map[int]bool)
	for _, p := range s.avisos {
		if p.Total != cuantos {
			t.Errorf("un aviso dice que el total es %d", p.Total)
		}
		visto[p.Hechos] = true
	}
	// Llegan desordenados, pero no puede faltar ninguna cuenta por el camino.
	for i := 1; i <= cuantos; i++ {
		if !visto[i] {
			t.Errorf("falta el aviso de haber hecho %d", i)
		}
	}
}
