package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Accion es lo que se hizo.
type Accion string

const (
	AccionCifrar    Accion = "cifrar"
	AccionDescifrar Accion = "descifrar"
)

// Entrada es una línea del historial.
//
// Lo que hay aquí es deliberadamente pobre: qué se hizo, con qué nombre de
// fichero y cuándo. Ni el contenido, ni la clave, ni el contenedor cifrado.
// El historial sirve para reencontrar un fichero, no para recuperar un secreto,
// y guardar de más convertiría un fichero de conveniencia en un objetivo.
type Entrada struct {
	Accion  Accion `json:"accion"`
	Nombre  string `json:"nombre"`
	Destino string `json:"destino"`
	// Cuando va como texto ISO y no como time.Time: al otro lado del puente no
	// existe ese tipo, y el generador de Wails no sabe qué hacer con él.
	Cuando string `json:"cuando"`
}

// maximo de entradas que se conservan. Pasado eso se tiran las más viejas: un
// historial sin tope acaba siendo un registro de toda la actividad de años.
const maximo = 200

// Historial guarda lo hecho últimamente en la carpeta de configuración.
type Historial struct {
	mu       sync.Mutex
	ruta     string
	entradas []Entrada
}

// AbrirHistorial lee el de disco, si lo hay.
//
// Un historial ilegible o corrupto no es motivo para no arrancar: se empieza uno
// nuevo y se sigue. Lo que no puede pasar es que la aplicación no abra porque un
// fichero de conveniencia tenga una llave de más.
func AbrirHistorial() *Historial {
	h := &Historial{ruta: rutaHistorial()}
	if h.ruta == "" {
		return h
	}
	datos, err := os.ReadFile(h.ruta)
	if err != nil {
		return h
	}
	_ = json.Unmarshal(datos, &h.entradas)
	return h
}

func rutaHistorial() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "historial.json")
}

// Ruta dice dónde vive el fichero, para poder enseñarlo en la interfaz.
func (h *Historial) Ruta() string { return h.ruta }

// Anotar añade una entrada y la guarda.
func (h *Historial) Anotar(accion Accion, nombre, destino string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entradas = append([]Entrada{{
		Accion:  accion,
		Nombre:  nombre,
		Destino: destino,
		Cuando:  time.Now().Format(time.RFC3339),
	}}, h.entradas...)

	if len(h.entradas) > maximo {
		h.entradas = h.entradas[:maximo]
	}
	h.guardar()
}

// Entradas devuelve lo anotado, de lo más reciente a lo más antiguo.
func (h *Historial) Entradas() []Entrada {
	h.mu.Lock()
	defer h.mu.Unlock()

	out := make([]Entrada, len(h.entradas))
	copy(out, h.entradas)
	return out
}

// Vaciar borra el historial de memoria y de disco.
func (h *Historial) Vaciar() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entradas = nil
	if h.ruta == "" {
		return nil
	}
	if err := os.Remove(h.ruta); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// guardar escribe el fichero. Se llama con el candado ya cogido.
//
// Los fallos se tragan a propósito: no poder escribir el historial —un disco
// lleno, una carpeta sin permisos— no puede impedir que se cifre. La operación
// de verdad ya ha terminado bien cuando se llega aquí.
func (h *Historial) guardar() {
	if h.ruta == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(h.ruta), 0o700); err != nil {
		return
	}
	datos, err := json.MarshalIndent(h.entradas, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(h.ruta, datos, 0o600)
}
