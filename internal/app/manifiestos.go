package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Los manifiestos de «native messaging»: cómo se entera el navegador de que el
// puente existe.
//
// Un navegador no lanza cualquier programa. Solo lanza los que estén declarados
// en un fichero suyo, en una carpeta suya, y **solo si la extensión que lo pide
// está en la lista de ese fichero**. Ese fichero lo escribe Esfinge cuando se
// enciende el canal en Ajustes, y lo borra cuando se apaga: el interruptor tiene
// que apagar la puerta entera, no solo el socket.
//
// En Windows es lo mismo con un paso más: el fichero va en la carpeta de Esfinge y
// lo que se escribe donde mira el navegador es **una clave de registro** que apunta
// a él. Encender y apagar el canal escribe y borra las dos cosas.
//
// Es, además, el punto flojo que conviene tener escrito y está en
// `docs/seguridad.md`: **el fichero vive en una carpeta que el usuario puede
// escribir**. Un programa con los permisos de la persona puede cambiar la ruta y
// ponerse en medio. Contra eso no protege ni este fichero ni el emparejamiento;
// protege no tener programas así, que es de lo que habla el apartado «de qué NO
// protege».

// nombreDelHost identifica al puente. Los navegadores solo admiten minúsculas,
// cifras, puntos y guiones bajos, sin dos puntos seguidos.
const nombreDelHost = "com.webcafeina.esfinge"

// sistema es el sistema para el que se calculan las rutas. **Es una variable y no
// `runtime.GOOS` a secas** por lo mismo que `filtrosPara` toma el sistema como
// argumento: la tabla de Windows se prueba entera desde Linux, donde se desarrolla,
// y no hay un Windows a mano en el que ejecutarla.
var sistema = runtime.GOOS

// entorno lee las variables del sistema. Variable, para las pruebas.
var entorno = os.Getenv

// registro es **donde Windows apunta a quién puede lanzar cada navegador**: allí
// el navegador no mira una carpeta, mira una clave de registro cuyo valor por
// defecto es la ruta del manifiesto. Detrás de una interfaz porque el de verdad solo
// existe en Windows (`registro_windows.go`) y en las pruebas se usa uno de mentira.
type registro interface {
	// Poner deja `valor` en el valor por defecto de `clave`, bajo `HKCU`,
	// creándola si no existe.
	Poner(clave, valor string) error
	// Quitar borra la clave. Que no estuviera no es un fallo.
	Quitar(clave string) error
}

// elRegistro es el de Windows en Windows y nulo en los demás.
var elRegistro = registroDelSistema()

// quiénPuedeLlamar: las extensiones que pueden lanzar el puente.
//
// En Firefox el identificador lo elige uno y va en el manifiesto de la extensión
// (`browser_specific_settings.gecko.id`).
//
// **En Chrome lo deriva el navegador de la clave pública que lleve el
// manifiesto**, y ahí está el truco: sin `key`, Chrome se inventa uno distinto en
// cada máquina y en cada instalación, y entonces no hay nada que escribir aquí
// —había que pasarlo a mano por el entorno—. Con `key` puesta, el identificador
// **es el mismo siempre y en todas partes**, así que se puede fijar aquí y la
// extensión funciona nada más cargarla.
//
// Lo que hay en el manifiesto es la **clave pública**: no firma nada y no hay
// secreto que guardar. Solo decide el identificador.
//
// **Y el de la tienda, al lado** (ADR 0033): la Chrome Web Store asigna el suyo,
// sacado de su propia clave, y el paquete que se le sube va sin `key`. Se calculó
// de la clave pública que enseña la consola de la tienda —SHA-256 de la clave,
// los 32 primeros dígitos hexadecimales pasados a las letras de la «a» a la «p»,
// el mismo cálculo que da `jkka…` con la de desarrollo— el 2026-09-15. Los dos
// conviven: la de desarrollo es la de quien carga la extensión a mano.
//
// Y `ESFINGE_EXTENSIONES` sigue existiendo para probar una extensión sin
// publicar. Es una variable de entorno y no un ajuste de la ventana a propósito:
// quien prueba algo sin publicar sabe usar el entorno, y un campo de texto en
// Ajustes donde escribir «quién puede leer mi bóveda» es justo el campo que
// alguien acaba rellenando porque se lo han dicho por teléfono.
var (
	extensionesDeFirefox = []string{"esfinge@webcafeina.com"}
	extensionesDeChrome  = []string{
		"jkkadfdagaojlgffkcboniepfgjkeenk", // la de desarrollo, con la `key` del manifiesto
		"jfkkegampjamnnlopobepjoanebemegp", // la de la Chrome Web Store
	}
)

// deDesarrollo son las que se añaden a mano para probar.
func deDesarrollo() []string {
	crudo := strings.TrimSpace(os.Getenv("ESFINGE_EXTENSIONES"))
	if crudo == "" {
		return nil
	}
	var fuera []string
	for _, id := range strings.Split(crudo, ",") {
		if id = strings.TrimSpace(id); id != "" {
			fuera = append(fuera, id)
		}
	}
	return fuera
}

// manifiesto es el fichero que lee el navegador.
type manifiesto struct {
	Nombre      string `json:"name"`
	Descripcion string `json:"description"`
	Ruta        string `json:"path"`
	Tipo        string `json:"type"`
	// Chrome usa «allowed_origins» con direcciones «chrome-extension://…/» y
	// Firefox «allowed_extensions» con identificadores pelados. No es lo mismo con
	// otro nombre: el formato del valor también cambia.
	Origenes    []string `json:"allowed_origins,omitempty"`
	Extensiones []string `json:"allowed_extensions,omitempty"`
}

// sitio es un navegador: **dónde se mira para saber si está y dónde se escribe**.
//
// Los dos no son lo mismo, y confundirlos fue el fallo que dejó la extensión sin
// funcionar en el primer Mac donde se probó. En macOS, Firefox guarda su perfil
// en `~/Library/Application Support/Firefox` y **lee los manifiestos de
// `…/Mozilla/NativeMessagingHosts`**, una carpeta que no existe hasta que alguien
// instala un host nativo. Comprobando la carpeta de destino —que es lo que hacía
// esto— la conclusión era «Firefox no está instalado» en una máquina con Firefox
// abierto, y no se escribía nada. Desde el otro lado se veía como «la extensión
// no hace nada», sin más pista.
type sitio struct {
	// Nombre es como se llama en la ventana. Enseñar a quién se ha avisado es lo
	// que convierte «no funciona» en «ya veo por qué».
	Nombre string
	// Senal es lo que existe si el navegador está instalado.
	Senal string
	// Carpeta es donde se deja el manifiesto: donde lo busca el navegador en macOS y
	// Linux, y la de Esfinge en Windows, donde lo que busca es la clave.
	Carpeta string
	// Fichero es el nombre del manifiesto dentro de la carpeta.
	Fichero string
	// Claves son, **solo en Windows**, las del registro bajo `HKCU` que apuntan al
	// manifiesto. En macOS y Linux ninguna: allí la carpeta es la dirección.
	Claves []string
	// Familia dice qué campo lleva el manifiesto: «firefox» o «chrome».
	Familia string
}

// dondeMiraCadaNavegador, en este sistema.
func dondeMiraCadaNavegador(casa string) []sitio {
	fichero := nombreDelHost + ".json"
	switch sistema {
	case "darwin":
		soporte := filepath.Join(casa, "Library", "Application Support")
		anfitriones := func(base string) string {
			return filepath.Join(base, "NativeMessagingHosts")
		}
		return []sitio{
			// **La señal es el perfil, no la carpeta de destino.**
			{"Firefox", filepath.Join(soporte, "Firefox"), anfitriones(filepath.Join(soporte, "Mozilla")), fichero, nil, "firefox"},
			{"Chrome", filepath.Join(soporte, "Google", "Chrome"), anfitriones(filepath.Join(soporte, "Google", "Chrome")), fichero, nil, "chrome"},
			{"Chromium", filepath.Join(soporte, "Chromium"), anfitriones(filepath.Join(soporte, "Chromium")), fichero, nil, "chrome"},
			{"Edge", filepath.Join(soporte, "Microsoft Edge"), anfitriones(filepath.Join(soporte, "Microsoft Edge")), fichero, nil, "chrome"},
			{"Brave", filepath.Join(soporte, "BraveSoftware", "Brave-Browser"), anfitriones(filepath.Join(soporte, "BraveSoftware", "Brave-Browser")), fichero, nil, "chrome"},
			{"Vivaldi", filepath.Join(soporte, "Vivaldi"), anfitriones(filepath.Join(soporte, "Vivaldi")), fichero, nil, "chrome"},
			{"Opera", filepath.Join(soporte, "com.operasoftware.Opera"), anfitriones(filepath.Join(soporte, "com.operasoftware.Opera")), fichero, nil, "chrome"},
		}
	case "linux":
		config := filepath.Join(casa, ".config")
		anfitriones := func(base string) string {
			return filepath.Join(base, "NativeMessagingHosts")
		}
		return []sitio{
			// En Linux sí coinciden: el perfil vive en `~/.mozilla/firefox` y los
			// manifiestos en `~/.mozilla/native-messaging-hosts`, las dos bajo la
			// misma carpeta. Aun así se dice cuál es cuál, que es lo que evita
			// volver a mezclarlas.
			{"Firefox", filepath.Join(casa, ".mozilla"), filepath.Join(casa, ".mozilla", "native-messaging-hosts"), fichero, nil, "firefox"},
			{"Chrome", filepath.Join(config, "google-chrome"), anfitriones(filepath.Join(config, "google-chrome")), fichero, nil, "chrome"},
			{"Chromium", filepath.Join(config, "chromium"), anfitriones(filepath.Join(config, "chromium")), fichero, nil, "chrome"},
			{"Edge", filepath.Join(config, "microsoft-edge"), anfitriones(filepath.Join(config, "microsoft-edge")), fichero, nil, "chrome"},
			{"Brave", filepath.Join(config, "BraveSoftware", "Brave-Browser"), anfitriones(filepath.Join(config, "BraveSoftware", "Brave-Browser")), fichero, nil, "chrome"},
			{"Vivaldi", filepath.Join(config, "vivaldi"), anfitriones(filepath.Join(config, "vivaldi")), fichero, nil, "chrome"},
			{"Opera", filepath.Join(config, "opera"), anfitriones(filepath.Join(config, "opera")), fichero, nil, "chrome"},
		}
	case "windows":
		// **En Windows el navegador mira el registro, no una carpeta** (documentación
		// de Chrome, Edge y MDN): una clave bajo `HKCU` cuyo valor por defecto es la
		// ruta absoluta del manifiesto, que puede estar en cualquier sitio. Van en la
		// carpeta de Esfinge, **uno por familia**, porque llevan campos distintos.
		local := entorno("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(casa, "AppData", "Local")
		}
		itinerante := entorno("APPDATA")
		if itinerante == "" {
			itinerante = filepath.Join(casa, "AppData", "Roaming")
		}
		carpeta := filepath.Join(itinerante, "Esfinge", "NativeMessagingHosts")
		clave := func(base string) string {
			return `Software\` + base + `\NativeMessagingHosts\` + nombreDelHost
		}
		deChrome := nombreDelHost + ".chrome.json"
		deGoogle := clave(`Google\Chrome`)
		return []sitio{
			{"Firefox", filepath.Join(itinerante, "Mozilla", "Firefox"), carpeta, nombreDelHost + ".firefox.json", []string{clave("Mozilla")}, "firefox"},
			{"Chrome", filepath.Join(local, "Google", "Chrome", "User Data"), carpeta, deChrome, []string{deGoogle}, "chrome"},
			{"Chromium", filepath.Join(local, "Chromium", "User Data"), carpeta, deChrome, []string{clave("Chromium")}, "chrome"},
			// Edge busca primero la suya y, si no está, la de Chromium y la de Chrome.
			{"Edge", filepath.Join(local, "Microsoft", "Edge", "User Data"), carpeta, deChrome, []string{clave(`Microsoft\Edge`)}, "chrome"},
			// **Brave, Vivaldi y Opera no documentan qué clave leen**, y lo que se
			// encuentra se contradice: hay quien dice que Brave solo lee la suya y quien
			// dice que comparte la de Chrome. Brave va en las dos; Vivaldi y Opera, en la
			// de Chrome, que es lo que hace KeePassXC hoy. Sin comprobar en un Windows.
			{"Brave", filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data"), carpeta, deChrome, []string{clave(`BraveSoftware\Brave-Browser`), deGoogle}, "chrome"},
			{"Vivaldi", filepath.Join(local, "Vivaldi", "User Data"), carpeta, deChrome, []string{deGoogle}, "chrome"},
			{"Opera", filepath.Join(itinerante, "Opera Software", "Opera Stable"), carpeta, deChrome, []string{deGoogle}, "chrome"},
		}
	}
	return nil
}

// rutaDelPuente es el binario que el navegador va a lanzar.
//
// Se busca **al lado del ejecutable que está corriendo**, que es donde queda
// instalado en los tres sistemas: dentro del paquete en macOS, en `/usr/bin` en
// Linux y junto a la aplicación en Windows. Buscarlo en el `PATH` sería peor: el
// navegador lanza lo que diga este fichero, así que la ruta tiene que ser la que
// se sabe, no la que se encuentre.
func rutaDelPuente() (string, error) {
	yo, err := os.Executable()
	if err != nil {
		return "", err
	}
	yo, err = filepath.EvalSymlinks(yo)
	if err != nil {
		return "", err
	}
	nombre := "esfinge-puente"
	if runtime.GOOS == "windows" {
		nombre += ".exe"
	}
	ruta := filepath.Join(filepath.Dir(yo), nombre)
	if _, err := os.Stat(ruta); err != nil {
		return "", errors.New("No encuentro el puente del navegador junto a Esfinge")
	}
	return ruta, nil
}

// escribirManifiestos deja el fichero donde lo busca cada navegador **que esté
// instalado**.
//
// Lo de «que esté instalado» es para no dejar ficheros sueltos en el perfil de
// alguien para siempre; lo que se mira para decidirlo es la **señal** de cada
// navegador, no la carpeta de destino, que puede no existir todavía.
func escribirManifiestos(casa, puente string) ([]string, []error) {
	var avisados []string
	var fallos []error
	for _, s := range dondeMiraCadaNavegador(casa) {
		if _, err := os.Stat(s.Senal); err != nil {
			continue // ese navegador no está
		}
		m := manifiesto{
			Nombre:      nombreDelHost,
			Descripcion: "El puente entre la extensión de Esfinge y la bóveda",
			Ruta:        puente,
			Tipo:        "stdio",
		}
		if s.Familia == "firefox" {
			m.Extensiones = append(append([]string{}, extensionesDeFirefox...), deDesarrollo()...)
		} else {
			for _, id := range append(append([]string{}, extensionesDeChrome...), deDesarrollo()...) {
				m.Origenes = append(m.Origenes, "chrome-extension://"+id+"/")
			}
		}
		// Sin extensiones que autorizar no se escribe nada: un manifiesto con la
		// lista vacía no sirve para nada y encima parece que sí.
		if len(m.Extensiones) == 0 && len(m.Origenes) == 0 {
			continue
		}

		datos, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			fallos = append(fallos, err)
			continue
		}
		if err := os.MkdirAll(s.Carpeta, 0o755); err != nil {
			fallos = append(fallos, err)
			continue
		}
		ruta := filepath.Join(s.Carpeta, s.Fichero)
		if err := os.WriteFile(ruta, datos, 0o644); err != nil {
			fallos = append(fallos, err)
			continue
		}
		// Y en Windows, la clave que dice dónde está, **con la ruta absoluta**: en el
		// registro Chrome no acepta una relativa.
		if len(s.Claves) > 0 {
			if err := apuntarEnElRegistro(s.Claves, ruta); err != nil {
				fallos = append(fallos, err)
				continue
			}
		}
		avisados = append(avisados, s.Nombre)
	}
	return avisados, fallos
}

// borrarManifiestos los quita de todas partes. Lo llama apagar el canal: un
// interruptor que dejara el manifiesto puesto estaría apagando media puerta.
func borrarManifiestos(casa string) {
	for _, s := range dondeMiraCadaNavegador(casa) {
		_ = os.Remove(filepath.Join(s.Carpeta, s.Fichero))
		if elRegistro == nil {
			continue
		}
		for _, c := range s.Claves {
			_ = elRegistro.Quitar(c)
		}
	}
}

// apuntarEnElRegistro pone en cada clave la ruta absoluta del manifiesto.
func apuntarEnElRegistro(claves []string, ruta string) error {
	if elRegistro == nil {
		return errors.New("No hay registro de Windows donde apuntar el manifiesto")
	}
	absoluta, err := filepath.Abs(ruta)
	if err != nil {
		return err
	}
	for _, c := range claves {
		if err := elRegistro.Poner(c, absoluta); err != nil {
			return err
		}
	}
	return nil
}
