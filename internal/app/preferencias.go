package app

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/escritura"
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

	// Los dos relojes de la bóveda. **«Nunca» es -1 y no 0**, y eso es lo único
	// importante de este par de campos.
	//
	// GuardarPreferencias recibe el objeto entero, así que el cero es lo que llega
	// cuando alguien manda un objeto a medias: un `{"buscarActualizaciones":true}`
	// deja los dos campos a cero al deserializar. Si el cero significara «nunca»,
	// ese guardado apagaría el bloqueo de la bóveda y el borrado del portapapeles
	// **en silencio y sin que nadie lo haya pedido**. Con el cero significando «no
	// lo he dicho, deja lo que había», el descuido es inofensivo. Ver fundir.
	MinutosParaBloquear    int `json:"minutosParaBloquear"`
	SegundosDePortapapeles int `json:"segundosDePortapapeles"`
}

// Nunca es lo que se manda para apagar uno de los dos relojes.
const Nunca = -1

// Lo que valen los dos relojes mientras nadie diga otra cosa, y hasta dónde se
// dejan mover. El tope de arriba no es desconfianza: una bóveda que no se cierra
// en ocho horas es una bóveda abierta, y ahí ya vale más «nunca», que al menos
// se lee como lo que es.
const (
	minutosBloqueoPorDefecto = 15
	minutosBloqueoMaximo     = 480

	segundosPortapapelesPorDefecto = 30
	segundosPortapapelesMinimo     = 5
	segundosPortapapelesMaximo     = 600
)

// fundir mezcla lo que llega con lo que había y deja los plazos en su sitio.
//
// La regla, escrita una sola vez y aplicada a los dos: **cero es «no lo he
// dicho»**, cualquier negativo es «nunca», y lo demás se recorta a lo que tiene
// sentido.
func fundir(anterior, nuevo Preferencias) Preferencias {
	nuevo.MinutosParaBloquear = plazo(anterior.MinutosParaBloquear, nuevo.MinutosParaBloquear,
		0, minutosBloqueoMaximo)
	nuevo.SegundosDePortapapeles = plazo(anterior.SegundosDePortapapeles, nuevo.SegundosDePortapapeles,
		segundosPortapapelesMinimo, segundosPortapapelesMaximo)
	return nuevo
}

func plazo(anterior, nuevo, minimo, maximo int) int {
	switch {
	case nuevo == 0:
		return anterior
	case nuevo < 0:
		return Nunca
	case nuevo < minimo:
		return minimo
	case nuevo > maximo:
		return maximo
	default:
		return nuevo
	}
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
		p: Preferencias{
			BuscarActualizaciones:  true,
			MinutosParaBloquear:    minutosBloqueoPorDefecto,
			SegundosDePortapapeles: segundosPortapapelesPorDefecto,
		},
	}
	if a.ruta == "" {
		return a
	}
	datos, err := os.ReadFile(a.ruta)
	if err != nil {
		return a
	}
	// Los valores por defecto están puestos arriba, antes de leer: json.Unmarshal
	// solo pisa lo que viene en el fichero, así que un fichero de antes de la
	// bóveda —que no lleva estos campos— se queda con ellos.
	_ = json.Unmarshal(datos, &a.p)
	// Y si el fichero trae un cero a pelo —que ya no lo escribe nadie, pero un
	// fichero es un fichero— se queda con lo de siempre en vez de con un plazo
	// que no significa nada.
	a.p = fundir(Preferencias{
		MinutosParaBloquear:    minutosBloqueoPorDefecto,
		SegundosDePortapapeles: segundosPortapapelesPorDefecto,
	}, a.p)
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
	a.p = fundir(a.p, p)
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
	return a.tocaMirar(cada)
}

// tocaMirar es lo mismo, con el cerrojo ya tomado: lo comparten TocaMirar y
// ReservarComprobacion, que necesita decidir y anotar sin soltarlo.
func (a *Ajustes) tocaMirar(cada time.Duration) bool {
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

// ReservarComprobacion dice si toca mirar y, si toca, **se queda el turno en el
// mismo cerrojo**: anota la fecha antes de que nadie salga a la red.
//
// Esa unión es el detalle que importa. Preguntar con TocaMirar y anotar al
// volver deja en medio toda la ida y vuelta a GitHub, y por ese hueco pasan
// varias comprobaciones a la vez. Con la comprobación solo al arrancar daba
// igual, porque no había dos; con el reloj de vigilar sí las hay, y una prueba
// lo pilló haciendo cuatro peticiones donde debía hacer una.
//
// Anotar antes de saber el resultado es a propósito: si la red falla, el turno
// se ha gastado igual. Es lo mismo que ya hacía el camino de error, y evita
// reintentar en bucle.
func (a *Ajustes) ReservarComprobacion(cada time.Duration) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.tocaMirar(cada) {
		return false
	}
	a.p.UltimaComprobacion = time.Now().Format(time.RFC3339)
	_ = a.guardar()
	return true
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
	datos, err := json.MarshalIndent(a.p, "", "  ")
	if err != nil {
		return err
	}
	// Igual que el historial: atómica, para que una caída a media escritura no
	// deje las preferencias truncadas y por tanto ilegibles.
	return escritura.Atomica(a.ruta, escritura.Opciones{CrearCarpeta: true},
		func(w io.Writer) error {
			_, err := w.Write(datos)
			return err
		})
}
