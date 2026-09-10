package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// El manifiesto es **la otra mitad de la puerta**: sin él, el socket está abierto
// y ningún navegador sabe que existe. Lo que se comprueba aquí es que dice lo que
// tiene que decir y, sobre todo, **a quién deja entrar**.
func TestElManifiestoDiceQuienPuedeLlamar(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("en Windows esto va al registro, y todavía no está")
	}
	casa := t.TempDir()
	// Solo se escribe donde el navegador ya está instalado, así que se finge que
	// Firefox lo está.
	if err := os.MkdirAll(filepath.Join(casa, ".mozilla"), 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		os.MkdirAll(filepath.Join(casa, "Library", "Application Support", "Mozilla"), 0o755)
	}

	if fallos := escribirManifiestos(casa, "/donde/sea/esfinge-puente"); len(fallos) > 0 {
		t.Fatalf("no ha podido escribirlo: %v", fallos)
	}

	ruta := manifiestoDeFirefox(casa)
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no está el manifiesto de Firefox: %v", err)
	}
	var m manifiesto
	if err := json.Unmarshal(datos, &m); err != nil {
		t.Fatal(err)
	}
	if m.Nombre != nombreDelHost || m.Tipo != "stdio" {
		t.Errorf("manifiesto raro: %+v", m)
	}
	if m.Ruta != "/donde/sea/esfinge-puente" {
		t.Errorf("apunta a %q", m.Ruta)
	}
	if len(m.Extensiones) != 1 || m.Extensiones[0] != "esfinge@webcafeina.com" {
		t.Errorf("deja entrar a %v", m.Extensiones)
	}
	// **Firefox lleva «allowed_extensions» y Chrome «allowed_origins», y no es el
	// mismo campo con otro nombre**: el formato del valor también cambia.
	if len(m.Origenes) != 0 {
		t.Errorf("el manifiesto de Firefox lleva orígenes de Chrome: %v", m.Origenes)
	}

	// Apagar el canal tiene que llevarse el fichero: dejarlo puesto sería apagar
	// media puerta.
	borrarManifiestos(casa)
	if _, err := os.Stat(ruta); err == nil {
		t.Error("el manifiesto sigue ahí después de apagar el canal")
	}
}

// Sin nadie a quien autorizar **no se escribe nada**. Es el caso de Chrome hoy:
// su identificador lo asigna la tienda y todavía no existe, y un manifiesto con
// la lista vacía no sirve para nada y encima parece que sí.
func TestSinExtensionesNoSeEscribeManifiesto(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("en Windows esto va al registro, y todavía no está")
	}
	casa := t.TempDir()
	base := filepath.Join(casa, ".config", "google-chrome")
	if runtime.GOOS == "darwin" {
		base = filepath.Join(casa, "Library", "Application Support", "Google", "Chrome")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}

	escribirManifiestos(casa, "/donde/sea/esfinge-puente")
	if _, err := os.Stat(filepath.Join(base, "NativeMessagingHosts", nombreDelHost+".json")); err == nil {
		t.Error("ha escrito un manifiesto de Chrome sin ninguna extensión que autorizar")
	}

	// Y con una de desarrollo puesta a mano, sí.
	t.Setenv("ESFINGE_EXTENSIONES", "abcdefghijklmnopabcdefghijklmnop")
	escribirManifiestos(casa, "/donde/sea/esfinge-puente")
	datos, err := os.ReadFile(filepath.Join(base, "NativeMessagingHosts", nombreDelHost+".json"))
	if err != nil {
		t.Fatalf("con una extensión de desarrollo tampoco lo escribe: %v", err)
	}
	if !strings.Contains(string(datos), "chrome-extension://abcdefghijklmnopabcdefghijklmnop/") {
		t.Errorf("no la ha puesto como origen: %s", datos)
	}
}

// Los seis de la familia de Chromium comparten el mismo manifiesto y lo buscan en
// seis sitios distintos. Esta prueba existe porque **la lista de carpetas es la
// clase de cosa que se copia mal**: una ruta con una mayúscula de menos deja a un
// navegador fuera sin que nada falle.
func TestLosSeisDeLaFamiliaDeChromium(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("en Windows esto va al registro, y todavía no está")
	}
	t.Setenv("ESFINGE_EXTENSIONES", "abcdefghijklmnopabcdefghijklmnop")
	casa := t.TempDir()

	// Cada uno con su carpeta base, como si estuvieran los seis instalados.
	var bases []string
	if runtime.GOOS == "darwin" {
		soporte := filepath.Join(casa, "Library", "Application Support")
		bases = []string{
			filepath.Join(soporte, "Google", "Chrome"),
			filepath.Join(soporte, "Chromium"),
			filepath.Join(soporte, "Microsoft Edge"),
			filepath.Join(soporte, "BraveSoftware", "Brave-Browser"),
			filepath.Join(soporte, "Vivaldi"),
			filepath.Join(soporte, "com.operasoftware.Opera"),
		}
	} else {
		config := filepath.Join(casa, ".config")
		bases = []string{
			filepath.Join(config, "google-chrome"),
			filepath.Join(config, "chromium"),
			filepath.Join(config, "microsoft-edge"),
			filepath.Join(config, "BraveSoftware", "Brave-Browser"),
			filepath.Join(config, "vivaldi"),
			filepath.Join(config, "opera"),
		}
	}
	for _, b := range bases {
		if err := os.MkdirAll(b, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	escribirManifiestos(casa, "/donde/sea/esfinge-puente")
	for _, b := range bases {
		ruta := filepath.Join(b, "NativeMessagingHosts", nombreDelHost+".json")
		if _, err := os.Stat(ruta); err != nil {
			t.Errorf("falta el manifiesto en %s", b)
		}
	}

	borrarManifiestos(casa)
	for _, b := range bases {
		ruta := filepath.Join(b, "NativeMessagingHosts", nombreDelHost+".json")
		if _, err := os.Stat(ruta); err == nil {
			t.Errorf("sigue el manifiesto en %s después de apagar el canal", b)
		}
	}
}

// No se crea la carpeta de un navegador que no está instalado: sería dejar un
// fichero suelto en el perfil de alguien para siempre.
func TestNoSeTocaElPerfilDeUnNavegadorQueNoEsta(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("en Windows esto va al registro, y todavía no está")
	}
	casa := t.TempDir()
	escribirManifiestos(casa, "/donde/sea/esfinge-puente")

	entradas, err := os.ReadDir(casa)
	if err != nil {
		t.Fatal(err)
	}
	if len(entradas) != 0 {
		t.Errorf("ha creado cosas sin que hubiera ningún navegador: %v", entradas)
	}
}

func manifiestoDeFirefox(casa string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(casa, "Library", "Application Support", "Mozilla",
			"NativeMessagingHosts", nombreDelHost+".json")
	}
	return filepath.Join(casa, ".mozilla", "native-messaging-hosts", nombreDelHost+".json")
}
