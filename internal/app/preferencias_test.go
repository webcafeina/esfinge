package app

import (
	"os"
	"testing"
	"time"
)

// enConfiguracionDePruebas aparta la carpeta de configuración real, como hace
// nuevaDePrueba con el historial.
func enConfiguracionDePruebas(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux
	t.Setenv("HOME", dir)            // macOS
	t.Setenv("AppData", dir)         // Windows
	t.Setenv("USERPROFILE", dir)
}

func TestLasPreferenciasVienenEncendidasYSeGuardan(t *testing.T) {
	enConfiguracionDePruebas(t)

	a := AbrirAjustes()
	if !a.Ver().BuscarActualizaciones {
		t.Fatal("de fábrica se busca; ha salido apagado")
	}

	if err := a.Guardar(Preferencias{BuscarActualizaciones: false}); err != nil {
		t.Fatalf("guardar: %v", err)
	}

	// Se relee de disco, que es lo que pasa al arrancar la siguiente vez.
	if otro := AbrirAjustes(); otro.Ver().BuscarActualizaciones {
		t.Error("se apagó, y al releerlo vuelve encendido")
	}
}

// El fichero está en la misma carpeta que el historial, y ahí no entra nadie más
// que su dueño.
func TestElFicheroDePreferenciasEsSoloDelDueno(t *testing.T) {
	enConfiguracionDePruebas(t)

	a := AbrirAjustes()
	if err := a.Guardar(Preferencias{BuscarActualizaciones: true}); err != nil {
		t.Fatalf("guardar: %v", err)
	}

	info, err := os.Stat(rutaPreferencias())
	if err != nil {
		t.Fatalf("no se ha escrito el fichero: %v", err)
	}
	if permisos := info.Mode().Perm(); permisos != 0o600 {
		t.Errorf("permisos: quiero 0600, tengo %04o", permisos)
	}
}

func TestNoSeSaleALaRedDosVecesElMismoDia(t *testing.T) {
	enConfiguracionDePruebas(t)

	a := AbrirAjustes()
	if !a.TocaMirar(24 * time.Hour) {
		t.Fatal("la primera vez siempre toca mirar")
	}

	a.AnotarComprobacion("2.1.0")
	if a.TocaMirar(24 * time.Hour) {
		t.Error("se acaba de mirar y dice que toca otra vez")
	}
	if !a.TocaMirar(time.Nanosecond) {
		t.Error("pasado el plazo tiene que volver a tocar")
	}
}

func TestApagarlaLaApaga(t *testing.T) {
	enConfiguracionDePruebas(t)

	a := AbrirAjustes()
	if err := a.Guardar(Preferencias{BuscarActualizaciones: false}); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	if a.TocaMirar(24 * time.Hour) {
		t.Error("está apagada y seguiría saliendo a la red")
	}
}

// Guardar preferencias viene de la ventana, y la ventana no tiene por qué saber
// de las cuentas internas: si mandara la fecha en blanco borraría el plazo y la
// comprobación pasaría a hacerse en cada arranque.
func TestGuardarNoBorraLaFechaDeLaUltimaComprobacion(t *testing.T) {
	enConfiguracionDePruebas(t)

	a := AbrirAjustes()
	a.AnotarComprobacion("2.1.0")
	antes := a.Ver().UltimaComprobacion

	if err := a.Guardar(Preferencias{BuscarActualizaciones: true}); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	if despues := a.Ver().UltimaComprobacion; despues != antes {
		t.Errorf("la fecha: quiero %q, tengo %q", antes, despues)
	}
}

// Un fichero corrupto no puede impedir que la aplicación abra, igual que con el
// historial.
func TestUnFicheroIlegibleNoImpideArrancar(t *testing.T) {
	enConfiguracionDePruebas(t)

	a := AbrirAjustes()
	if err := a.Guardar(Preferencias{BuscarActualizaciones: true}); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	if err := os.WriteFile(rutaPreferencias(), []byte("{ esto no es json"), 0o600); err != nil {
		t.Fatalf("estropear: %v", err)
	}

	if otro := AbrirAjustes(); !otro.Ver().BuscarActualizaciones {
		t.Error("con el fichero roto tiene que volver a los valores de fábrica")
	}
}
