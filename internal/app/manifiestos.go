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
// Es, además, el punto flojo que conviene tener escrito y está en
// `docs/seguridad.md`: **el fichero vive en una carpeta que el usuario puede
// escribir**. Un programa con los permisos de la persona puede cambiar la ruta y
// ponerse en medio. Contra eso no protege ni este fichero ni el emparejamiento;
// protege no tener programas así, que es de lo que habla el apartado «de qué NO
// protege».

// nombreDelHost identifica al puente. Los navegadores solo admiten minúsculas,
// cifras, puntos y guiones bajos, sin dos puntos seguidos.
const nombreDelHost = "com.webcafeina.esfinge"

// quiénPuedeLlamar: las extensiones que pueden lanzar el puente.
//
// **Firefox ya está y Chrome todavía no**, y la diferencia no es descuido: en
// Firefox el identificador lo elige uno y va en el manifiesto de la extensión
// (`browser_specific_settings.gecko.id`); en Chrome **lo asigna la tienda** al
// subirla por primera vez, así que hasta que exista no hay nada que escribir aquí.
//
// Para probar con una extensión cargada a mano —que en Chrome recibe un
// identificador distinto cada vez que se instala— está `ESFINGE_EXTENSIONES`, con
// los identificadores separados por comas. Es una variable de entorno y no un
// ajuste de la ventana a propósito: quien está probando una extensión sin
// publicar sabe usar el entorno, y un campo de texto en Ajustes donde escribir
// «quién puede leer mi bóveda» es justo el campo que alguien acaba rellenando
// porque se lo han dicho por teléfono.
var (
	extensionesDeFirefox = []string{"esfinge@webcafeina.com"}
	extensionesDeChrome  []string
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

// carpetasDeManifiestos dice dónde busca cada navegador, en este sistema.
//
// Son rutas por usuario, no del sistema: Esfinge se instala sin permisos de
// administrador y no tiene por qué tocar nada de fuera de la carpeta de quien la
// usa.
//
// **Los seis de la familia de Chromium comparten formato y comparten lista**: el
// mismo manifiesto vale para Chrome, Chromium, Edge, Brave, Vivaldi y Opera,
// porque todos leen «allowed_origins» con direcciones «chrome-extension://…». Lo
// único que cambia es dónde lo buscan, y por eso esto es una lista de carpetas y
// no seis casos.
//
// Firefox va aparte de verdad: su campo es «allowed_extensions» y el valor es el
// identificador pelado, no una dirección.
func carpetasDeManifiestos(casa string) map[string][]string {
	switch runtime.GOOS {
	case "darwin":
		soporte := filepath.Join(casa, "Library", "Application Support")
		return map[string][]string{
			"chrome": {
				filepath.Join(soporte, "Google", "Chrome", "NativeMessagingHosts"),
				filepath.Join(soporte, "Chromium", "NativeMessagingHosts"),
				filepath.Join(soporte, "Microsoft Edge", "NativeMessagingHosts"),
				filepath.Join(soporte, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts"),
				filepath.Join(soporte, "Vivaldi", "NativeMessagingHosts"),
				filepath.Join(soporte, "com.operasoftware.Opera", "NativeMessagingHosts"),
			},
			"firefox": {filepath.Join(soporte, "Mozilla", "NativeMessagingHosts")},
		}
	case "linux":
		config := filepath.Join(casa, ".config")
		return map[string][]string{
			"chrome": {
				filepath.Join(config, "google-chrome", "NativeMessagingHosts"),
				filepath.Join(config, "chromium", "NativeMessagingHosts"),
				filepath.Join(config, "microsoft-edge", "NativeMessagingHosts"),
				filepath.Join(config, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts"),
				filepath.Join(config, "vivaldi", "NativeMessagingHosts"),
				filepath.Join(config, "opera", "NativeMessagingHosts"),
			},
			"firefox": {filepath.Join(casa, ".mozilla", "native-messaging-hosts")},
		}
	}
	// En Windows no van en carpetas sino en el registro, y eso es otra historia
	// que se escribirá cuando toque empaquetar. Está en docs/deuda.md.
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

// escribirManifiestos deja el fichero en cada carpeta de navegador que exista.
//
// **Solo donde el navegador ya está instalado.** Crear la carpeta de un navegador
// que no está sería dejar un fichero suelto en el perfil de alguien para siempre.
func escribirManifiestos(casa, puente string) []error {
	var fallos []error
	for familia, carpetas := range carpetasDeManifiestos(casa) {
		m := manifiesto{
			Nombre:      nombreDelHost,
			Descripcion: "El puente entre la extensión de Esfinge y la bóveda",
			Ruta:        puente,
			Tipo:        "stdio",
		}
		switch familia {
		case "firefox":
			m.Extensiones = append(append([]string{}, extensionesDeFirefox...), deDesarrollo()...)
		default:
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
		for _, carpeta := range carpetas {
			if _, err := os.Stat(filepath.Dir(carpeta)); err != nil {
				continue // ese navegador no está instalado
			}
			if err := os.MkdirAll(carpeta, 0o755); err != nil {
				fallos = append(fallos, err)
				continue
			}
			destino := filepath.Join(carpeta, nombreDelHost+".json")
			if err := os.WriteFile(destino, datos, 0o644); err != nil {
				fallos = append(fallos, err)
			}
		}
	}
	return fallos
}

// borrarManifiestos los quita de todas partes. Lo llama apagar el canal: un
// interruptor que dejara el manifiesto puesto estaría apagando media puerta.
func borrarManifiestos(casa string) {
	for _, carpetas := range carpetasDeManifiestos(casa) {
		for _, carpeta := range carpetas {
			_ = os.Remove(filepath.Join(carpeta, nombreDelHost+".json"))
		}
	}
}
