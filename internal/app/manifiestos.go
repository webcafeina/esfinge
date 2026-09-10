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
	// Carpeta es donde ese navegador busca los manifiestos.
	Carpeta string
	// Familia dice qué campo lleva el manifiesto: «firefox» o «chrome».
	Familia string
}

// dondeMiraCadaNavegador, en este sistema.
func dondeMiraCadaNavegador(casa string) []sitio {
	switch runtime.GOOS {
	case "darwin":
		soporte := filepath.Join(casa, "Library", "Application Support")
		anfitriones := func(base string) string {
			return filepath.Join(base, "NativeMessagingHosts")
		}
		return []sitio{
			// **La señal es el perfil, no la carpeta de destino.**
			{"Firefox", filepath.Join(soporte, "Firefox"), anfitriones(filepath.Join(soporte, "Mozilla")), "firefox"},
			{"Chrome", filepath.Join(soporte, "Google", "Chrome"), anfitriones(filepath.Join(soporte, "Google", "Chrome")), "chrome"},
			{"Chromium", filepath.Join(soporte, "Chromium"), anfitriones(filepath.Join(soporte, "Chromium")), "chrome"},
			{"Edge", filepath.Join(soporte, "Microsoft Edge"), anfitriones(filepath.Join(soporte, "Microsoft Edge")), "chrome"},
			{"Brave", filepath.Join(soporte, "BraveSoftware", "Brave-Browser"), anfitriones(filepath.Join(soporte, "BraveSoftware", "Brave-Browser")), "chrome"},
			{"Vivaldi", filepath.Join(soporte, "Vivaldi"), anfitriones(filepath.Join(soporte, "Vivaldi")), "chrome"},
			{"Opera", filepath.Join(soporte, "com.operasoftware.Opera"), anfitriones(filepath.Join(soporte, "com.operasoftware.Opera")), "chrome"},
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
			{"Firefox", filepath.Join(casa, ".mozilla"), filepath.Join(casa, ".mozilla", "native-messaging-hosts"), "firefox"},
			{"Chrome", filepath.Join(config, "google-chrome"), anfitriones(filepath.Join(config, "google-chrome")), "chrome"},
			{"Chromium", filepath.Join(config, "chromium"), anfitriones(filepath.Join(config, "chromium")), "chrome"},
			{"Edge", filepath.Join(config, "microsoft-edge"), anfitriones(filepath.Join(config, "microsoft-edge")), "chrome"},
			{"Brave", filepath.Join(config, "BraveSoftware", "Brave-Browser"), anfitriones(filepath.Join(config, "BraveSoftware", "Brave-Browser")), "chrome"},
			{"Vivaldi", filepath.Join(config, "vivaldi"), anfitriones(filepath.Join(config, "vivaldi")), "chrome"},
			{"Opera", filepath.Join(config, "opera"), anfitriones(filepath.Join(config, "opera")), "chrome"},
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
		if err := os.WriteFile(filepath.Join(s.Carpeta, nombreDelHost+".json"), datos, 0o644); err != nil {
			fallos = append(fallos, err)
			continue
		}
		avisados = append(avisados, s.Nombre)
	}
	return avisados, fallos
}

// borrarManifiestos los quita de todas partes. Lo llama apagar el canal: un
// interruptor que dejara el manifiesto puesto estaría apagando media puerta.
func borrarManifiestos(casa string) {
	for _, s := range dondeMiraCadaNavegador(casa) {
		_ = os.Remove(filepath.Join(s.Carpeta, nombreDelHost+".json"))
	}
}
