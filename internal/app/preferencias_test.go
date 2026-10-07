package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// enConfiguracionDePruebas aparta la carpeta de configuración real, como hace
// nuevaDePrueba con el historial.
//
// **Los cuatro nombres hacen falta y ninguno sobra**, porque `os.UserConfigDir` mira uno
// distinto en cada sistema. Dejarse el de Windows aísla en esta máquina y **no aísla
// allí**, que es donde nadie lo mira: las pruebas se pisan entre ellas y el síntoma es
// una que se queja de encontrarse una bóveda que no ha creado.
func enConfiguracionDePruebas(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux
	t.Setenv("HOME", dir)            // macOS
	t.Setenv("AppData", dir)         // Windows
	t.Setenv("USERPROFILE", dir)
	t.Setenv("LOCALAPPDATA", dir) // y la caché de Windows, por lo mismo
	return dir
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

	a.AnotarComprobacion()
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
	a.AnotarComprobacion()
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

// **Apartar la carpeta de configuración se hace en un solo sitio, y esto lo vigila.**
//
// Cada prueba que escribe en la carpeta del usuario tiene que apartarla, y hacerlo son
// cuatro variables de entorno porque `os.UserConfigDir` mira una distinta en cada
// sistema. Escribirlas a mano **sale bien en esta máquina y mal en Windows**: ahí nadie
// mira, y lo que se ve es una prueba quejándose de encontrarse una bóveda que no ha
// creado — porque la creó otra que corrió antes en la misma carpeta de verdad.
//
// Pasó: `conReloj` tenía su propia lista sin la de Windows, y seis sitios más la
// copiaban. **Tiró una publicación**, y no se vio hasta que el filtro del trabajo de
// Windows juntó dos pruebas que crean bóveda. Aquí no se puede ejecutar Windows, así que
// lo que queda es comprobar que nadie vuelva a escribir la lista por su cuenta.
func TestLaCarpetaDeConfiguracionSeApartaEnUnSoloSitio(t *testing.T) {
	ficheros, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	// El del ayudante, y el del equipo de cuentas, que elige su propia carpeta para
	// poder tener dos equipos a la vez y lo dice en su comentario.
	salvo := map[string]bool{"preferencias_test.go": true, "cuenta_test.go": true}
	for _, f := range ficheros {
		if salvo[f] {
			continue
		}
		crudo, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(crudo), `Setenv("XDG_CONFIG_HOME"`) {
			t.Errorf("%s aparta la carpeta por su cuenta; usa enConfiguracionDePruebas, "+
				"que es donde están los cuatro nombres y por qué hacen falta", f)
		}
	}
}
