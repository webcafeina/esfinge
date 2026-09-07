package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Preferencias es lo poco que Esfinge recuerda entre arranques además del
// historial.
//
// Hay una sola, y existe por una razón concreta: la comprobación de
// actualizaciones sale a la red, y eso tiene que poder apagarse y tiene que
// poder espaciarse. Lo demás del programa sigue sin guardar estado.
type Preferencias struct {
	// BuscarActualizaciones vale true mientras no se diga lo contrario. Es la
	// decisión que se tomó: avisar por defecto, contarlo en Ajustes y dejar
	// apagarlo, en vez de esconderlo detrás de un botón que nadie pulsa.
	BuscarActualizaciones bool `json:"buscarActualizaciones"`
	// UltimaComprobacion en RFC3339, como las entradas del historial: al otro
	// lado del puente no existe time.Time.
	UltimaComprobacion string `json:"ultimaComprobacion"`
	// VersionVista es la última que se ofreció. Sirve para no repetir el mismo
	// aviso en cada arranque cuando ya se dijo «ahora no».
	VersionVista string `json:"versionVista"`
	// CarpetaAbrir y CarpetaGuardar son las últimas que se usaron en cada
	// diálogo. Van separadas porque son gestos distintos: se abre de donde están
	// los ficheros y se guarda donde va el resultado.
	CarpetaAbrir   string `json:"carpetaAbrir"`
	CarpetaGuardar string `json:"carpetaGuardar"`
}

// Ajustes guarda las preferencias en la carpeta de configuración.
type Ajustes struct {
	mu   sync.Mutex
	ruta string
	p    Preferencias
}

// AbrirAjustes lee los de disco, si los hay.
//
// Como con el historial: un fichero ilegible no impide arrancar. Se empieza con
// los valores de siempre y se sigue.
func AbrirAjustes() *Ajustes {
	a := &Ajustes{
		ruta: rutaPreferencias(),
		p:    Preferencias{BuscarActualizaciones: true},
	}
	if a.ruta == "" {
		return a
	}
	datos, err := os.ReadFile(a.ruta)
	if err != nil {
		return a
	}
	_ = json.Unmarshal(datos, &a.p)
	return a
}

func rutaPreferencias() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "preferencias.json")
}

// Ver devuelve una copia de lo guardado.
func (a *Ajustes) Ver() Preferencias {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.p
}

// Guardar sustituye las preferencias y las escribe.
//
// Las marcas de tiempo no se dejan en manos de quien llama desde la ventana: se
// conservan las que ya había, porque son cuentas internas y no ajustes.
func (a *Ajustes) Guardar(p Preferencias) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	p.UltimaComprobacion = a.p.UltimaComprobacion
	p.VersionVista = a.p.VersionVista
	p.CarpetaAbrir = a.p.CarpetaAbrir
	p.CarpetaGuardar = a.p.CarpetaGuardar
	a.p = p
	return a.guardar()
}

// TocaMirar dice si ha pasado bastante desde la última consulta.
//
// Una vez al día basta para enterarse de una versión nueva, y evita que abrir y
// cerrar la ventana cinco veces sean cinco peticiones. La cuenta la comparten la
// ventana y la línea de comandos, que leen el mismo fichero.
func (a *Ajustes) TocaMirar(cada time.Duration) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.p.BuscarActualizaciones {
		return false
	}
	if a.p.UltimaComprobacion == "" {
		return true
	}
	cuando, err := time.Parse(time.RFC3339, a.p.UltimaComprobacion)
	if err != nil {
		return true
	}
	return time.Since(cuando) >= cada
}

// AnotarComprobacion deja constancia de que se acaba de mirar, y de qué versión
// se vio.
func (a *Ajustes) AnotarComprobacion(version string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.p.UltimaComprobacion = time.Now().Format(time.RFC3339)
	if version != "" {
		a.p.VersionVista = version
	}
	_ = a.guardar()
}

// CarpetaDeAbrir y CarpetaDeGuardar devuelven la última que se usó, **si todavía
// existe**.
//
// La comprobación no es cosmética: Wails **falla la llamada entera** si el
// directorio por defecto no existe, así que una carpeta borrada o en un disco
// desconectado dejaría el diálogo sin abrir. Recordar de más no puede salir más
// caro que no recordar nada.
func (a *Ajustes) CarpetaDeAbrir() string   { return siSigueAhi(a.Ver().CarpetaAbrir) }
func (a *Ajustes) CarpetaDeGuardar() string { return siSigueAhi(a.Ver().CarpetaGuardar) }

func siSigueAhi(carpeta string) string {
	if carpeta == "" {
		return ""
	}
	if info, err := os.Stat(carpeta); err != nil || !info.IsDir() {
		return ""
	}
	return carpeta
}

// RecordarCarpetaDeAbrir y RecordarCarpetaDeGuardar anotan dónde se quedó el
// diálogo. No devuelven error: no poder escribir una comodidad no puede
// estropear la operación que acaba de salir bien.
func (a *Ajustes) RecordarCarpetaDeAbrir(carpeta string) {
	a.recordar(&a.p.CarpetaAbrir, carpeta)
}

func (a *Ajustes) RecordarCarpetaDeGuardar(carpeta string) {
	a.recordar(&a.p.CarpetaGuardar, carpeta)
}

func (a *Ajustes) recordar(donde *string, carpeta string) {
	if carpeta == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if *donde == carpeta {
		return
	}
	*donde = carpeta
	_ = a.guardar()
}

// guardar escribe el fichero. Se llama con el candado ya cogido.
//
// A diferencia del historial, aquí el fallo sí se devuelve: si alguien apaga la
// comprobación de actualizaciones y no se puede escribir, tiene que enterarse.
func (a *Ajustes) guardar() error {
	if a.ruta == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(a.ruta), 0o700); err != nil {
		return err
	}
	datos, err := json.MarshalIndent(a.p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.ruta, datos, 0o600)
}
