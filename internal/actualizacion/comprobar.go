// Package actualizacion mira si hay una versión más nueva publicada, se la trae
// y se la entrega al sistema para que la instale.
//
// Es una de las dos partes de Esfinge que salen a la red —la otra es
// internal/iconos, desde la 2.14.0— y la única que habla con un destino fijo. Lo
// hace de una forma, y lo hace de una forma: una
// petición GET a la API de GitHub que no lleva nada más que la versión instalada
// en el «User-Agent». Ni identificadores, ni contadores, ni nada de lo que se
// cifra. Está contado en docs/adr/0014-comprobacion-de-actualizaciones.md.
//
// No sabe nada de Wails ni de la interfaz a propósito: así lo usan igual la
// ventana y la línea de comandos, y se puede probar entero contra un servidor de
// mentira, sin tocar internet.
package actualizacion

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// APIPorDefecto es la de GitHub. Se puede cambiar al construir el comprobador,
// que es lo que hacen los tests y el servidor de desarrollo.
const APIPorDefecto = "https://api.github.com"

// repositorio del que se sacan las publicaciones.
const repositorio = "webcafeina/esfinge"

// Comprobador consulta las publicaciones.
type Comprobador struct {
	// Version es la instalada, la que se compara con la publicada.
	Version string
	// API es la raíz de la API. Vacía significa la de GitHub.
	API string
	// Cliente propio y no http.DefaultClient: el de por defecto no tiene plazo,
	// y una comprobación de cortesía no puede quedarse colgada esperando a una
	// red que no contesta.
	Cliente *http.Client
}

// Nuevo construye un comprobador con los valores de siempre.
func Nuevo(version string) *Comprobador {
	return &Comprobador{
		Version: version,
		API:     APIPorDefecto,
		Cliente: &http.Client{Timeout: 10 * time.Second},
	}
}

// Adjunto es un fichero colgado de una publicación.
type Adjunto struct {
	Nombre string `json:"name"`
	URL    string `json:"browser_download_url"`
	Bytes  int64  `json:"size"`
}

// publicacion es lo que devuelve la API, recortado a lo que hace falta.
type publicacion struct {
	Etiqueta string    `json:"tag_name"`
	Pagina   string    `json:"html_url"`
	Borrador bool      `json:"draft"`
	Previa   bool      `json:"prerelease"`
	Adjuntos []Adjunto `json:"assets"`
}

// Novedad es el resultado de mirar: si hay algo más nuevo y qué habría que
// bajarse en esta máquina.
//
// Los campos son planos y con etiqueta json porque esto cruza el puente hasta la
// ventana tal cual.
type Novedad struct {
	Hay     bool   `json:"hay"`
	Version string `json:"version"`
	Pagina  string `json:"pagina"`
	// Fichero es el adjunto que le toca a este sistema. Puede venir vacío aunque
	// haya versión nueva: entonces se avisa, pero se manda a la página.
	Fichero string `json:"fichero"`
	URL     string `json:"url"`
	Bytes   int64  `json:"bytes"`
	// Resumen es la línea del SHA256SUMS publicado que corresponde al fichero.
	Resumen string `json:"-"`
}

// Mirar consulta la última publicación y dice si hay algo más nuevo.
func (c *Comprobador) Mirar() (Novedad, error) {
	pub, err := c.ultima()
	if err != nil {
		return Novedad{}, err
	}

	publicada := strings.TrimPrefix(pub.Etiqueta, "v")
	mas, err := EsMasNueva(publicada, c.Version)
	if err != nil || !mas {
		// Un error aquí es «no sé comparar», no «va mal»: pasa con las versiones
		// de trabajo, y ahí lo correcto es callarse.
		return Novedad{}, nil
	}

	n := Novedad{Hay: true, Version: publicada, Pagina: pub.Pagina}
	if a, vale := ParaEsteSistema(pub.Adjuntos); vale {
		n.Fichero, n.URL, n.Bytes = a.Nombre, a.URL, a.Bytes
		n.Resumen = c.resumenDe(pub.Adjuntos, a.Nombre)
	}
	return n, nil
}

func (c *Comprobador) ultima() (publicacion, error) {
	raiz := c.API
	if raiz == "" {
		raiz = APIPorDefecto
	}

	url := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimSuffix(raiz, "/"), repositorio)
	pet, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return publicacion{}, err
	}
	// La API de GitHub rechaza las peticiones sin User-Agent. Va la versión
	// porque es lo que se está comparando; nada más.
	pet.Header.Set("User-Agent", "Esfinge/"+c.Version)
	pet.Header.Set("Accept", "application/vnd.github+json")

	cliente := c.Cliente
	if cliente == nil {
		cliente = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := cliente.Do(pet)
	if err != nil {
		return publicacion{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return publicacion{}, fmt.Errorf("GitHub contestó %s", resp.Status)
	}

	var pub publicacion
	if err := json.NewDecoder(resp.Body).Decode(&pub); err != nil {
		return publicacion{}, fmt.Errorf("No se entiende lo que contestó GitHub: %w", err)
	}
	if pub.Borrador || pub.Previa {
		return publicacion{}, fmt.Errorf("La última publicación no es definitiva")
	}
	return pub, nil
}

// ParaEsteSistema elige, de todo lo colgado de la publicación, el fichero que le
// toca a esta máquina.
//
// Se busca por sufijo dentro de lo que devuelve la API en vez de componer el
// nombre a mano: si algún día cambia cómo se llaman los ficheros, las versiones
// ya instaladas seguirán encontrando la suya.
func ParaEsteSistema(adjuntos []Adjunto) (Adjunto, bool) {
	var sufijos []string
	switch runtime.GOOS {
	case "darwin":
		// Una sola imagen, universal para Intel y Apple Silicon.
		sufijos = []string{".dmg"}
	case "windows":
		sufijos = []string{"-windows-instalador.exe"}
	case "linux":
		// Quien lo instaló con el paquete recibe un paquete; quien se bajó el
		// tar.gz y lo puso a mano no quiere que le metan un .deb por la puerta.
		if instaladoConPaquete() {
			sufijos = []string{"_" + runtime.GOARCH + ".deb", ".deb"}
		}
		sufijos = append(sufijos, "-linux-"+runtime.GOARCH+".tar.gz")
	default:
		return Adjunto{}, false
	}

	for _, sufijo := range sufijos {
		for _, a := range adjuntos {
			if strings.HasSuffix(a.Nombre, sufijo) {
				return a, true
			}
		}
	}
	return Adjunto{}, false
}

// resumenDe baja el SHA256SUMS de la publicación y saca la línea del fichero.
//
// Si no se puede, se devuelve vacío y la descarga seguirá adelante sin
// comprobar: es peor dejar a alguien sin poder actualizarse que dejarle
// actualizarse sin el resumen, que en todo caso solo protege de una descarga
// rota, no de una publicación manipulada.
func (c *Comprobador) resumenDe(adjuntos []Adjunto, fichero string) string {
	var url string
	for _, a := range adjuntos {
		if a.Nombre == "SHA256SUMS" {
			url = a.URL
			break
		}
	}
	if url == "" {
		return ""
	}

	cliente := c.Cliente
	if cliente == nil {
		cliente = &http.Client{Timeout: 10 * time.Second}
	}
	pet, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	pet.Header.Set("User-Agent", "Esfinge/"+c.Version)

	resp, err := cliente.Do(pet)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// Con tope: un SHA256SUMS son unas líneas, y lo que venga de la red no se
	// lee entero sin mirar cuánto es.
	datos, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}
	return ResumenEn(string(datos), fichero)
}

// ResumenEn busca en un SHA256SUMS la línea de un fichero y devuelve su resumen.
//
// El formato es el de coreutils: «<resumen>  <nombre>», con dos espacios.
func ResumenEn(sumas, fichero string) string {
	for _, linea := range strings.Split(sumas, "\n") {
		campos := strings.Fields(linea)
		if len(campos) != 2 {
			continue
		}
		// El nombre puede venir con «*» delante, que es como marca coreutils los
		// ficheros leídos en binario.
		if strings.TrimPrefix(campos[1], "*") == fichero {
			return campos[0]
		}
	}
	return ""
}

// EsMasNueva compara dos versiones de tres números.
//
// Devuelve error si alguna no se puede leer, que es lo que pasa con las
// compilaciones de trabajo: «dev», o el «2.0.3-3-gabc1234» que sale de
// «git describe». Ante la duda no se avisa de nada: quien compila en su máquina
// no necesita que le ofrezcan descargarse una versión.
func EsMasNueva(candidata, actual string) (bool, error) {
	a, err := trocear(candidata)
	if err != nil {
		return false, err
	}
	b, err := trocear(actual)
	if err != nil {
		return false, err
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i], nil
		}
	}
	return false, nil
}

func trocear(v string) ([3]int, error) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	partes := strings.Split(v, ".")
	if len(partes) != 3 {
		return out, fmt.Errorf("«%s» no es una versión de tres números", v)
	}
	for i, p := range partes {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, fmt.Errorf("«%s» no es una versión de tres números", v)
		}
		out[i] = n
	}
	return out, nil
}
