// Package adjuntos es **el único sitio** donde se dice cómo se llama cada fichero que
// cuelga de una publicación de Esfinge, para qué equipo es y quién lo fabrica.
//
// # Por qué existe
//
// Esos nombres vivían en siete sitios: el `Makefile`, `publicar.yml`, el generador de la
// tabla del README, el cuerpo de la publicación, `empaquetado/linux/instalar.sh`, el que
// elige la descarga (`internal/actualizacion`) y cinco ficheros de prueba. En esta casa
// **una regla escrita en N sitios ya ha costado cinco fallos** —`vaciarLoSensible`,
// `enConfiguracionDePruebas` (que tiró una publicación), `PasaPorLaValvula`, el sello en
// dos mitades y el gancho de la cadena—, y el día que la publicación pase de 28 adjuntos
// a 37 esa cuenta no se sostiene con acordarse.
//
// # Por qué en Go y no en un YAML o en Python
//
// Por la lección de `zip`: **una herramienta de línea de órdenes que está en los tres
// sistemas de desarrollo puede no estar en los tres de la publicación**. Lo único
// garantizado en los cinco trabajos del flujo —incluidas las dos imágenes ARM que nadie
// de esta casa ha visto por dentro— es **Go, porque los cinco compilan Esfinge**. Es el
// mismo argumento que decidió que `herramientas/armar-mcpb` fuera un programa de Go y no
// un `zip`.
package adjuntos

import (
	"fmt"
	"sort"
	"strings"
)

// Las clases de adjunto. La clase dice **qué es**, no para quién.
const (
	ClaseApp       = "app"       // la aplicación con ventana, empaquetada para instalar
	ClasePaquete   = "paquete"   // lo mismo, pero para un gestor de paquetes
	ClaseCLI       = "cli"       // un binario suelto de la línea de comandos
	ClaseMCPB      = "mcpb"      // el paquete que se arrastra a un cliente de agentes
	ClaseExtension = "extension" // la extensión del navegador
	ClaseSumas     = "sumas"     // el SHA256SUMS
)

// Los gestores de paquetes de los que puede venir una copia instalada.
//
// **Esto no es la clase del fichero: es de dónde vino quien pregunta.** Y es la mitad del
// porqué de este paquete: hasta el 2026-11-12 el que elegía la descarga preguntaba «¿vengo
// de un paquete?» —`instaladoConPaquete()`— cuando lo que necesita saber es **de qué
// gestor**. Con un solo paquete en el mundo las dos preguntas valían lo mismo; con `.deb`,
// `.rpm` y Arch, la primera ofrece un `.deb` a una Fedora. Es la familia «una función cuyo
// nombre contesta *casi* lo que quieres».
const (
	GestorDeb      = "deb"
	GestorRPM      = "rpm"
	GestorPacman   = "pacman"
	GestorAppImage = "appimage"
	// SinGestor es quien se bajó el tar.gz y lo puso a mano.
	SinGestor = ""
)

// Ficha es un adjunto de la publicación.
type Ficha struct {
	// Clave lo nombra aquí dentro, y es lo que piden los demás. Estable: cambiar una
	// clave es cambiar lo que pregunta otro fichero.
	Clave string
	// Sistema y Arq son para quién es. Arq puede ser "universal" (macOS) o estar vacío
	// (lo que no depende de la máquina, como el SHA256SUMS).
	Sistema, Arq string
	// Clase es qué es.
	Clase string
	// Gestor, solo en los paquetes: con qué se instala.
	Gestor string
	// Plantilla es el nombre, con {v} donde va la versión.
	Plantilla string
	// Rotulo y Nota son cómo sale en la tabla del README. Rotulo vacío = no sale.
	Rotulo, Nota string
	// Publicado dice si **hoy** hay un trabajo del flujo que lo fabrique.
	//
	// Es lo que hace que las fases se puedan publicar por separado: el catálogo puede
	// nombrar un adjunto antes de que exista —para que el que elige la descarga lo
	// acepte el día que aparezca— mientras la puerta de la publicación solo exige los
	// que ya tienen quien los produzca. Cada fase es un `true` más el trabajo que lo
	// fabrica, y nunca al revés.
	Publicado bool
}

// Nombre de este adjunto para esa versión.
func (f Ficha) Nombre(version string) string {
	return strings.ReplaceAll(f.Plantilla, "{v}", version)
}

// losObjetivos son los seis de `PLATAFORMAS` del Makefile, en el mismo orden.
var losObjetivos = []struct{ Sistema, Arq string }{
	{"linux", "amd64"}, {"linux", "arm64"},
	{"darwin", "amd64"}, {"darwin", "arm64"},
	{"windows", "amd64"}, {"windows", "arm64"},
}

// losBinarios son los tres que se cruzan a los seis objetivos.
var losBinarios = []string{"esfinge", "esfinge-puente", "esfinge-mcp"}

// Catalogo devuelve todos los adjuntos que Esfinge sabe nombrar, publicados o no.
//
// **El vocabulario es `amd64`/`arm64`**, como `PLATAFORMAS` del Makefile, y no el
// `x64` de otros proyectos de la casa: mezclar dos vocabularios es exactamente cómo se
// desalinea esto. Las dos excepciones son donde el nombre lo pone la herramienta y no
// nosotros: el `.rpm` y la AppImage, que hablan de `x86_64` y `aarch64`.
func Catalogo() []Ficha {
	fichas := []Ficha{
		// ---------------------------------------------------------------- macOS
		{
			Clave: "app-darwin", Sistema: "darwin", Arq: "universal", Clase: ClaseApp,
			Plantilla: "Esfinge-{v}.dmg", Publicado: true,
			Rotulo: "macOS",
			Nota:   "Universal: Apple Silicon e Intel",
		},
		{
			Clave: "mcpb-darwin", Sistema: "darwin", Arq: "universal", Clase: ClaseMCPB,
			Plantilla: "Esfinge-{v}-macos.mcpb", Publicado: true, Rotulo: "macOS",
		},

		// -------------------------------------------------------------- Windows
		//
		// **El nombre viejo se queda en el catálogo a propósito**, con su clave propia.
		// El que elige la descarga busca por sufijo justamente para que renombrar no
		// rompa a quien ya lo tiene instalado, así que mientras haya copias que buscan
		// `-windows-instalador.exe` el catálogo tiene que saber qué es eso.
		{
			Clave: "app-windows-amd64-viejo", Sistema: "windows", Arq: "amd64", Clase: ClaseApp,
			Plantilla: "Esfinge-{v}-windows-instalador.exe",
			// Lleva rótulo porque **hoy es el que sale en la tabla**. Al flipear
			// `Publicado` en la fase 2, las filas se cambian solas: la tabla filtra por
			// lo que de verdad está publicado, así que nunca enseña un enlace que no
			// existe ni se queda sin la fila de un sistema que sí tiene fichero.
			Rotulo: "Windows", Nota: "Asistente de instalación",
		},
		{
			Clave: "app-windows-amd64", Sistema: "windows", Arq: "amd64", Clase: ClaseApp,
			Plantilla: "Esfinge-{v}-windows-amd64-instalador.exe", Publicado: true,
			Rotulo: "Windows · Intel y AMD", Nota: "Asistente de instalación",
		},
		{
			Clave: "app-windows-arm64", Sistema: "windows", Arq: "arm64", Clase: ClaseApp,
			Plantilla: "Esfinge-{v}-windows-arm64-instalador.exe", Publicado: true,
			Rotulo: "Windows · ARM", Nota: "Asistente de instalación",
		},
		{
			Clave: "mcpb-windows-viejo", Sistema: "windows", Arq: "amd64", Clase: ClaseMCPB,
			Plantilla: "Esfinge-{v}-windows.mcpb", Rotulo: "Windows",
		},
		{
			Clave: "mcpb-windows-amd64", Sistema: "windows", Arq: "amd64", Clase: ClaseMCPB,
			Plantilla: "Esfinge-{v}-windows-amd64.mcpb", Publicado: true, Rotulo: "Windows · Intel y AMD",
		},
		{
			Clave: "mcpb-windows-arm64", Sistema: "windows", Arq: "arm64", Clase: ClaseMCPB,
			Plantilla: "Esfinge-{v}-windows-arm64.mcpb", Publicado: true, Rotulo: "Windows · ARM",
		},

		// ---------------------------------------------------------------- Linux
		{
			Clave: "deb-amd64", Sistema: "linux", Arq: "amd64", Clase: ClasePaquete,
			Gestor: GestorDeb, Plantilla: "esfinge_{v}_amd64.deb", Publicado: true,
			Rotulo: "Linux · Debian y Ubuntu", Nota: "Aplicación y línea de comandos",
		},
		{
			Clave: "deb-arm64", Sistema: "linux", Arq: "arm64", Clase: ClasePaquete,
			Gestor: GestorDeb, Plantilla: "esfinge_{v}_arm64.deb", Publicado: true,
			Rotulo: "Linux · Debian y Ubuntu (ARM)", Nota: "Aplicación y línea de comandos",
		},
		{
			Clave: "rpm-amd64", Sistema: "linux", Arq: "amd64", Clase: ClasePaquete,
			Gestor: GestorRPM, Plantilla: "esfinge-{v}-1.x86_64.rpm", Publicado: true,
			Rotulo: "Linux · Fedora y RHEL", Nota: "Aplicación y línea de comandos",
		},
		{
			Clave: "rpm-arm64", Sistema: "linux", Arq: "arm64", Clase: ClasePaquete,
			Gestor: GestorRPM, Plantilla: "esfinge-{v}-1.aarch64.rpm", Publicado: true,
			Rotulo: "Linux · Fedora y RHEL (ARM)", Nota: "Aplicación y línea de comandos",
		},
		{
			Clave: "appimage-amd64", Sistema: "linux", Arq: "amd64", Clase: ClasePaquete,
			Gestor: GestorAppImage, Plantilla: "Esfinge-{v}-x86_64.AppImage",
			Publicado: true,
			Rotulo:    "Linux · cualquiera (AppImage)", Nota: "Se ejecuta sin instalar nada",
		},
		{
			Clave: "appimage-arm64", Sistema: "linux", Arq: "arm64", Clase: ClasePaquete,
			Gestor: GestorAppImage, Plantilla: "Esfinge-{v}-aarch64.AppImage",
			Publicado: true,
			Rotulo:    "Linux · cualquiera (AppImage, ARM)", Nota: "Se ejecuta sin instalar nada",
		},
		{
			Clave: "tar-amd64", Sistema: "linux", Arq: "amd64", Clase: ClasePaquete,
			Plantilla: "esfinge-{v}-linux-amd64.tar.gz", Publicado: true,
			// **El rótulo es el que ya lee el cliente**, y no se cambia sin que lo pida:
			// la prueba que compara el README con el catálogo cazó que lo había tocado.
			// El de la AppImage se decidirá en su fase, con él delante.
			Rotulo: "Linux · cualquiera", Nota: "Los binarios sueltos",
		},
		{
			Clave: "tar-arm64", Sistema: "linux", Arq: "arm64", Clase: ClasePaquete,
			Plantilla: "esfinge-{v}-linux-arm64.tar.gz", Publicado: true,
			Rotulo: "Linux · cualquiera (ARM)", Nota: "Los binarios sueltos",
		},
		{
			Clave: "mcpb-linux-viejo", Sistema: "linux", Arq: "amd64", Clase: ClaseMCPB,
			Plantilla: "Esfinge-{v}-linux.mcpb", Rotulo: "Linux",
		},
		{
			Clave: "mcpb-linux-amd64", Sistema: "linux", Arq: "amd64", Clase: ClaseMCPB,
			Plantilla: "Esfinge-{v}-linux-amd64.mcpb", Publicado: true, Rotulo: "Linux · Intel y AMD",
		},
		{
			Clave: "mcpb-linux-arm64", Sistema: "linux", Arq: "arm64", Clase: ClaseMCPB,
			Plantilla: "Esfinge-{v}-linux-arm64.mcpb", Publicado: true, Rotulo: "Linux · ARM",
		},

		// ------------------------------------------------------------ extensión
		{
			Clave: "extension-chrome-tienda", Clase: ClaseExtension,
			Plantilla: "esfinge-extension-{v}-chrome-tienda.zip", Publicado: true,
		},
		{
			Clave: "extension-fuente", Clase: ClaseExtension,
			Plantilla: "esfinge-extension-{v}-fuente.zip", Publicado: true,
		},

		// ---------------------------------------------------------------- sumas
		{Clave: "sumas", Clase: ClaseSumas, Plantilla: "SHA256SUMS", Publicado: true},
	}

	// Y los dieciocho de la línea de comandos, que ya cruzan los seis objetivos desde
	// antes de que esto existiera. Se generan en vez de escribirse: dieciocho filas a
	// mano son dieciocho sitios donde equivocarse con un guion.
	for _, o := range losObjetivos {
		for _, b := range losBinarios {
			ext := ""
			if o.Sistema == "windows" {
				ext = ".exe"
			}
			clave := strings.TrimPrefix(b, "esfinge")
			clave = strings.TrimPrefix(clave, "-")
			if clave == "" {
				clave = "cli"
			}
			fichas = append(fichas, Ficha{
				Clave:     clave + "-" + o.Sistema + "-" + o.Arq,
				Sistema:   o.Sistema,
				Arq:       o.Arq,
				Clase:     ClaseCLI,
				Plantilla: b + "-{v}-" + o.Sistema + "-" + o.Arq + ext,
				Publicado: true,
			})
		}
	}
	return fichas
}

// Por devuelve la ficha de esa clave.
func Por(clave string) (Ficha, bool) {
	for _, f := range Catalogo() {
		if f.Clave == clave {
			return f, true
		}
	}
	return Ficha{}, false
}

// Nombre es el del adjunto de esa clave, o error si la clave no existe: un nombre
// inventado en silencio es un enlace roto en la tabla de descargas.
func Nombre(clave, version string) (string, error) {
	f, hay := Por(clave)
	if !hay {
		return "", fmt.Errorf("no hay ningún adjunto que se llame «%s» en el catálogo", clave)
	}
	return f.Nombre(version), nil
}

// Publicados son los nombres que **tienen que estar** colgados de la publicación de esa
// versión, ordenados.
//
// Es lo que la puerta del flujo compara con lo que hay, **en los dos sentidos**: que no
// falte ninguno y que no sobre ninguno. Sin el segundo sentido, un adjunto que se cuela
// con el nombre mal escrito no lo ve nadie hasta que alguien pulsa el enlace.
func Publicados(version string) []string {
	var out []string
	for _, f := range Catalogo() {
		if f.Publicado {
			out = append(out, f.Nombre(version))
		}
	}
	sort.Strings(out)
	return out
}

// De son todos los nombres por su clave, para quien tenga que construir una lista.
func De(version string) map[string]string {
	out := map[string]string{}
	for _, f := range Catalogo() {
		out[f.Clave] = f.Nombre(version)
	}
	return out
}

// ParaSistema devuelve las claves que le valen a ese equipo, **en orden de preferencia**.
//
// Los tres datos entran por parámetro y no se leen de `runtime`, y eso es la mitad del
// valor de este paquete: así los seis objetivos y los cuatro gestores se comprueban desde
// una sola máquina. La prueba de antes solo podía comprobar el sistema donde corría —lo
// decía su propio comentario— y por eso nadie vio que en una Fedora se iba a ofrecer un
// `.deb`.
//
// Vacío significa **no hay nada que descargarle a este equipo**, que no es un fallo: en
// Arch se actualiza con `pacman`, y a quien venga de ahí hay que decírselo en vez de
// bajarle un fichero que su gestor no conoce.
func ParaSistema(sistema, arq, gestor string) []string {
	switch sistema {
	case "darwin":
		// Una sola imagen, universal para Intel y Apple Silicon.
		return []string{"app-darwin"}

	case "windows":
		claves := []string{"app-windows-" + arq}
		// **El nombre viejo, y solo para amd64.** Es lo que deja que una copia
		// instalada hoy siga actualizándose cuando el fichero se renombre; ofrecérselo
		// a una máquina ARM sería darle el binario de la otra arquitectura, que es
		// justo lo que el renombrado viene a arreglar.
		if arq == "amd64" {
			claves = append(claves, "app-windows-amd64-viejo")
		}
		return claves

	case "linux":
		// **Se le da lo de su gestor y nada más.** Quien instaló con `rpm` no quiere un
		// `.deb` —no lo instalaría—, y quien se bajó el tar.gz a su carpeta no quiere
		// que le metan un paquete por la puerta, que le pediría contraseña para escribir
		// donde no tiene nada.
		switch gestor {
		case GestorDeb:
			return []string{"deb-" + arq}
		case GestorRPM:
			return []string{"rpm-" + arq}
		case GestorAppImage:
			return []string{"appimage-" + arq}
		case GestorPacman:
			// Nada: lo actualiza `pacman`. Lo dice la pantalla, no se descarga.
			return nil
		default:
			return []string{"tar-" + arq}
		}
	}
	return nil
}

// Sufijo es la parte del nombre que no cambia con la versión.
//
// Es lo que busca el que elige la descarga, y por una razón que lleva escrita desde la
// 2.0: **una copia ya instalada tiene que encontrar su fichero aunque el nombre cambie**.
// Sale de la plantilla y no de una lista aparte, así que no puede desalinearse con el
// nombre que de verdad se publica.
func (f Ficha) Sufijo() string {
	if i := strings.Index(f.Plantilla, "{v}"); i >= 0 {
		return f.Plantilla[i+len("{v}"):]
	}
	return f.Plantilla
}
