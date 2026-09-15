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
	deCarpetas(t)
	casa := t.TempDir()
	fingirFirefox(t, casa)

	if _, fallos := escribirManifiestos(casa, "/donde/sea/esfinge-puente"); len(fallos) > 0 {
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

// Sin nadie a quien autorizar **no se escribe nada**: un manifiesto con la lista
// vacía no sirve para nada y encima parece que sí.
//
// La lista se vacía aquí a mano porque hoy no está vacía —Chrome tiene su
// identificador fijo desde que el manifiesto lleva `key`—, y lo que hay que
// comprobar es **la regla**, no el estado de una lista que va a cambiar.
func TestSinExtensionesNoSeEscribeManifiesto(t *testing.T) {
	so := deCarpetas(t)
	antes := extensionesDeChrome
	extensionesDeChrome = nil
	t.Cleanup(func() { extensionesDeChrome = antes })

	casa := t.TempDir()
	base := filepath.Join(casa, ".config", "google-chrome")
	if so == "darwin" {
		base = filepath.Join(casa, "Library", "Application Support", "Google", "Chrome")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _ = escribirManifiestos(casa, "/donde/sea/esfinge-puente")
	if _, err := os.Stat(filepath.Join(base, "NativeMessagingHosts", nombreDelHost+".json")); err == nil {
		t.Error("ha escrito un manifiesto de Chrome sin ninguna extensión que autorizar")
	}

	// Y con una de desarrollo puesta a mano, sí.
	t.Setenv("ESFINGE_EXTENSIONES", "abcdefghijklmnopabcdefghijklmnop")
	_, _ = escribirManifiestos(casa, "/donde/sea/esfinge-puente")
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
	so := deCarpetas(t)
	t.Setenv("ESFINGE_EXTENSIONES", "abcdefghijklmnopabcdefghijklmnop")
	casa := t.TempDir()

	// Cada uno con su carpeta base, como si estuvieran los seis instalados.
	var bases []string
	if so == "darwin" {
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

	_, _ = escribirManifiestos(casa, "/donde/sea/esfinge-puente")
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
	deCarpetas(t)
	casa := t.TempDir()
	_, _ = escribirManifiestos(casa, "/donde/sea/esfinge-puente")

	entradas, err := os.ReadDir(casa)
	if err != nil {
		t.Fatal(err)
	}
	if len(entradas) != 0 {
		t.Errorf("ha creado cosas sin que hubiera ningún navegador: %v", entradas)
	}
}

// fingirFirefox crea **la señal de que Firefox está instalado**, que es su perfil
// y no la carpeta de los manifiestos.
//
// **Ésa fue la confusión que dejó la extensión muerta en el primer Mac.** En
// macOS el perfil vive en «…/Application Support/Firefox» y los manifiestos en
// «…/Application Support/Mozilla/NativeMessagingHosts», que no existe hasta que
// alguien instala un host nativo. Mirando la de destino, la conclusión era
// «Firefox no está» con Firefox abierto delante.
func fingirFirefox(t *testing.T, casa string) {
	t.Helper()
	perfil := filepath.Join(casa, ".mozilla")
	if sistema == "darwin" {
		perfil = filepath.Join(casa, "Library", "Application Support", "Firefox")
	}
	if err := os.MkdirAll(perfil, 0o755); err != nil {
		t.Fatal(err)
	}
}

// **La prueba que habría evitado el viaje.** Con Firefox instalado y sin que
// exista todavía la carpeta de los manifiestos, el manifiesto tiene que
// escribirse igual: esa carpeta la crea quien instala un host nativo, y el
// primero somos nosotros.
func TestElManifiestoSeEscribeAunqueSuCarpetaNoExista(t *testing.T) {
	deCarpetas(t)
	casa := t.TempDir()
	fingirFirefox(t, casa)

	// Y se comprueba que de verdad no existe, para que la prueba no pase por el
	// motivo equivocado el día que alguien cambie el ayudante de arriba.
	destino := filepath.Dir(manifiestoDeFirefox(casa))
	if _, err := os.Stat(destino); err == nil {
		t.Fatalf("la carpeta de destino ya existía: %s", destino)
	}

	if _, fallos := escribirManifiestos(casa, "/donde/sea/esfinge-puente"); len(fallos) > 0 {
		t.Fatalf("no ha podido escribirlo: %v", fallos)
	}
	if _, err := os.Stat(manifiestoDeFirefox(casa)); err != nil {
		t.Errorf("no ha escrito el manifiesto de Firefox: %v", err)
	}
}

func manifiestoDeFirefox(casa string) string {
	if sistema == "darwin" {
		return filepath.Join(casa, "Library", "Application Support", "Mozilla",
			"NativeMessagingHosts", nombreDelHost+".json")
	}
	return filepath.Join(casa, ".mozilla", "native-messaging-hosts", nombreDelHost+".json")
}

// enSistema hace que las rutas se calculen para ese sistema mientras dura la prueba.
func enSistema(t *testing.T, s string) {
	t.Helper()
	antes := sistema
	sistema = s
	t.Cleanup(func() { sistema = antes })
}

// deCarpetas pone la prueba en uno de los sistemas donde el navegador mira una
// carpeta: el de verdad, o Linux si la prueba corre en Windows. Así estas pruebas
// también corren en la máquina Windows de GitHub, en vez de saltarse.
func deCarpetas(t *testing.T) string {
	t.Helper()
	s := runtime.GOOS
	if s == "windows" {
		s = "linux"
	}
	enSistema(t, s)
	return s
}

/* ---------------------------------------------------------------- Windows */

// registroDeMentira apunta lo que se escribiría en el registro de Windows.
type registroDeMentira map[string]string

func (r registroDeMentira) Poner(clave, valor string) error {
	r[clave] = valor
	return nil
}

func (r registroDeMentira) Quitar(clave string) error {
	delete(r, clave)
	return nil
}

// enWindows calcula las rutas como en Windows, con un registro de mentira y las dos
// carpetas de datos dentro de una carpeta temporal.
func enWindows(t *testing.T) (string, registroDeMentira) {
	t.Helper()
	enSistema(t, "windows")
	casa := t.TempDir()
	reg := registroDeMentira{}
	antesReg, antesEntorno := elRegistro, entorno
	elRegistro = reg
	entorno = func(v string) string {
		switch v {
		case "LOCALAPPDATA":
			return filepath.Join(casa, "Local")
		case "APPDATA":
			return filepath.Join(casa, "Roaming")
		}
		return ""
	}
	t.Cleanup(func() { elRegistro, entorno = antesReg, antesEntorno })
	return casa, reg
}

func crear(t *testing.T, carpetas ...string) {
	t.Helper()
	for _, c := range carpetas {
		if err := os.MkdirAll(c, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

const (
	claveDeChrome  = `Software\Google\Chrome\NativeMessagingHosts\com.webcafeina.esfinge`
	claveDeFirefox = `Software\Mozilla\NativeMessagingHosts\com.webcafeina.esfinge`
)

// **En Windows el manifiesto no basta: hace falta la clave que apunta a él.** Y cada
// familia su fichero, porque Firefox lleva «allowed_extensions» y Chrome
// «allowed_origins».
func TestEnWindowsElManifiestoVaAlRegistro(t *testing.T) {
	casa, reg := enWindows(t)
	crear(t,
		filepath.Join(casa, "Roaming", "Mozilla", "Firefox"),
		filepath.Join(casa, "Local", "Google", "Chrome", "User Data"),
	)

	avisados, fallos := escribirManifiestos(casa, `C:\Program Files\Webcafeína\Esfinge\esfinge-puente.exe`)
	if len(fallos) > 0 {
		t.Fatalf("no ha podido: %v", fallos)
	}
	if strings.Join(avisados, ",") != "Firefox,Chrome" {
		t.Errorf("avisados: %v", avisados)
	}
	if len(reg) != 2 {
		t.Errorf("tenía que haber dos claves y hay %d: %v", len(reg), reg)
	}

	carpeta := filepath.Join(casa, "Roaming", "Esfinge", "NativeMessagingHosts")
	for clave, familia := range map[string]string{claveDeFirefox: "firefox", claveDeChrome: "chrome"} {
		ruta, esta := reg[clave]
		if !esta {
			t.Errorf("falta la clave %s", clave)
			continue
		}
		if ruta != filepath.Join(carpeta, nombreDelHost+"."+familia+".json") || !filepath.IsAbs(ruta) {
			t.Errorf("la clave de %s apunta a %q", familia, ruta)
		}
		datos, err := os.ReadFile(ruta)
		if err != nil {
			t.Errorf("la clave apunta a un fichero que no está: %v", err)
			continue
		}
		var m manifiesto
		if err := json.Unmarshal(datos, &m); err != nil {
			t.Fatal(err)
		}
		if m.Ruta != `C:\Program Files\Webcafeína\Esfinge\esfinge-puente.exe` {
			t.Errorf("el manifiesto de %s lanza %q", familia, m.Ruta)
		}
		if familia == "firefox" && (len(m.Extensiones) != 1 || len(m.Origenes) != 0) {
			t.Errorf("el de Firefox deja entrar a %v / %v", m.Extensiones, m.Origenes)
		}
		if familia == "chrome" && (len(m.Origenes) == 0 || len(m.Extensiones) != 0) {
			t.Errorf("el de Chrome deja entrar a %v / %v", m.Origenes, m.Extensiones)
		}
	}

	// Apagar el canal se lleva las claves y los ficheros.
	borrarManifiestos(casa)
	if len(reg) != 0 {
		t.Errorf("quedan claves después de apagar el canal: %v", reg)
	}
	if entradas, _ := os.ReadDir(carpeta); len(entradas) != 0 {
		t.Errorf("quedan manifiestos después de apagar el canal: %v", entradas)
	}
}

// **La tabla de Windows entera**, con los siete navegadores instalados. Es la clase
// de lista que se copia mal, y aquí además la mitad no está documentada: Brave va en
// su clave y en la de Chrome, y Vivaldi y Opera en la de Chrome.
func TestLaTablaDeWindows(t *testing.T) {
	casa, reg := enWindows(t)
	local, itinerante := filepath.Join(casa, "Local"), filepath.Join(casa, "Roaming")
	crear(t,
		filepath.Join(itinerante, "Mozilla", "Firefox"),
		filepath.Join(local, "Google", "Chrome", "User Data"),
		filepath.Join(local, "Chromium", "User Data"),
		filepath.Join(local, "Microsoft", "Edge", "User Data"),
		filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data"),
		filepath.Join(local, "Vivaldi", "User Data"),
		filepath.Join(itinerante, "Opera Software", "Opera Stable"),
	)

	avisados, fallos := escribirManifiestos(casa, `C:\esfinge-puente.exe`)
	if len(fallos) > 0 {
		t.Fatalf("no ha podido: %v", fallos)
	}
	if len(avisados) != 7 {
		t.Errorf("tenía que avisar a los siete y ha avisado a %v", avisados)
	}
	esperadas := []string{
		claveDeFirefox,
		claveDeChrome,
		`Software\Chromium\NativeMessagingHosts\com.webcafeina.esfinge`,
		`Software\Microsoft\Edge\NativeMessagingHosts\com.webcafeina.esfinge`,
		`Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\com.webcafeina.esfinge`,
	}
	for _, c := range esperadas {
		if _, esta := reg[c]; !esta {
			t.Errorf("falta la clave %s", c)
		}
	}
	if len(reg) != len(esperadas) {
		t.Errorf("hay %d claves y tenían que ser %d: %v", len(reg), len(esperadas), reg)
	}
}

// Brave solo, sin Chrome: **va en las dos claves**, porque no se sabe cuál lee.
func TestEnWindowsBraveVaEnLasDosClaves(t *testing.T) {
	casa, reg := enWindows(t)
	crear(t, filepath.Join(casa, "Local", "BraveSoftware", "Brave-Browser", "User Data"))
	_, _ = escribirManifiestos(casa, `C:\esfinge-puente.exe`)
	if _, esta := reg[claveDeChrome]; !esta {
		t.Error("Brave no ha dejado la clave de Chrome")
	}
	if _, esta := reg[`Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\com.webcafeina.esfinge`]; !esta {
		t.Error("Brave no ha dejado la suya")
	}
}

// Sin las variables de entorno, las carpetas de siempre dentro del perfil.
func TestEnWindowsSinVariablesSeMiraEnElPerfil(t *testing.T) {
	casa, reg := enWindows(t)
	entorno = func(string) string { return "" }
	crear(t, filepath.Join(casa, "AppData", "Roaming", "Mozilla", "Firefox"))
	_, _ = escribirManifiestos(casa, `C:\esfinge-puente.exe`)
	ruta, esta := reg[claveDeFirefox]
	if !esta || ruta != filepath.Join(casa, "AppData", "Roaming", "Esfinge", "NativeMessagingHosts", nombreDelHost+".firefox.json") {
		t.Errorf("sin variables apunta a %q", ruta)
	}
}

// Sin registro no se da por avisado a nadie: un manifiesto al que no apunta ninguna
// clave no lo encuentra el navegador, y decir «avisado» sería mentir.
func TestEnWindowsSinRegistroNoSeAvisaANadie(t *testing.T) {
	casa, _ := enWindows(t)
	elRegistro = nil
	crear(t, filepath.Join(casa, "Roaming", "Mozilla", "Firefox"))
	avisados, fallos := escribirManifiestos(casa, `C:\esfinge-puente.exe`)
	if len(avisados) != 0 || len(fallos) == 0 {
		t.Errorf("sin registro: avisados %v, fallos %v", avisados, fallos)
	}
}

// **Las dos de Chrome en el manifiesto**: la de desarrollo, para quien carga la
// extensión a mano, y la que asignó la tienda. Si falta la de la tienda, la extensión
// instalada desde la Chrome Web Store no puede lanzar el puente y no hace nada.
func TestChromeDejaEntrarALaDeLaTiendaYALaDeDesarrollo(t *testing.T) {
	deCarpetas(t)
	t.Setenv("ESFINGE_EXTENSIONES", "")
	casa := t.TempDir()
	base := filepath.Join(casa, ".config", "google-chrome")
	if sistema == "darwin" {
		base = filepath.Join(casa, "Library", "Application Support", "Google", "Chrome")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, fallos := escribirManifiestos(casa, "/donde/sea/esfinge-puente"); len(fallos) > 0 {
		t.Fatal(fallos)
	}
	datos, err := os.ReadFile(filepath.Join(base, "NativeMessagingHosts", nombreDelHost+".json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"jkkadfdagaojlgffkcboniepfgjkeenk", "jfkkegampjamnnlopobepjoanebemegp"} {
		if !strings.Contains(string(datos), "chrome-extension://"+id+"/") {
			t.Errorf("el manifiesto de Chrome no deja entrar a %s: %s", id, datos)
		}
	}
}
