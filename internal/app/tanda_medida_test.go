package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Cuánto cuesta una tanda de veinte. Sirve para decir un número al cambiar el
// reparto, no para vigilar nada: se ejecuta a mano.
func TestMedirUnaTandaDeVeinte(t *testing.T) {
	if os.Getenv("MEDIR") == "" {
		t.Skip("solo a mano: MEDIR=1 go test -run TestMedirUnaTandaDeVeinte ./internal/app/")
	}
	a, _ := nuevaDePrueba(t)
	dir := t.TempDir()

	var rutas []string
	for i := 0; i < 20; i++ {
		ruta := filepath.Join(dir, fmt.Sprintf("secreto-%02d.env", i))
		if err := os.WriteFile(ruta, []byte("API_KEY=abc123\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		rutas = append(rutas, ruta)
	}

	empezo := time.Now()
	if _, err := a.CifrarFicheros(rutas, "una clave"); err != nil {
		t.Fatal(err)
	}
	t.Logf("veinte ficheros en %v", time.Since(empezo).Round(time.Millisecond))
}
