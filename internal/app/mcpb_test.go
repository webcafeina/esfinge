package app

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// **El paquete instalable de Claude Desktop** (`.mcpb`), comprobado de verdad.
//
// Existe porque un paquete mal formado **no da ningún error**: Claude Desktop
// simplemente no lo instala, o lo instala y no arranca, y eso no se descubre hasta que
// alguien lo intenta en su Mac. Lo que se puede comprobar desde aquí es la forma, y es
// justo lo que más fácil se rompe al tocar el guion: que el zip lleve lo que tiene que
// llevar, que el manifiesto se entienda, y que **la versión sea la de la publicación** y
// no la que quedó escrita en el fichero.
func TestElPaqueteDeClaudeDesktopTieneLaForma(t *testing.T) {
	raiz := raizDelRepo(t)
	binario := filepath.Join(t.TempDir(), "esfinge-mcp")
	compilar := exec.Command("go", "build", "-o", binario, "./cmd/esfinge-mcp")
	compilar.Dir = raiz
	if salida, err := compilar.CombinedOutput(); err != nil {
		t.Fatalf("no se ha podido compilar el servidor: %v\n%s", err, salida)
	}

	paquete := filepath.Join(t.TempDir(), "esfinge.mcpb")
	armar := exec.Command("go", "run", "./herramientas/armar-mcpb", "9.9.9", binario, paquete)
	armar.Dir = raiz
	if salida, err := armar.CombinedOutput(); err != nil {
		t.Fatalf("no se ha podido armar el paquete: %v\n%s", err, salida)
	}

	z, err := zip.OpenReader(paquete)
	if err != nil {
		t.Fatalf("el paquete no es un zip: %v", err)
	}
	defer z.Close()

	dentro := map[string]bool{}
	modos := map[string]os.FileMode{}
	for _, f := range z.File {
		dentro[f.Name] = true
		modos[f.Name] = f.Mode()
	}
	// **Las tres cosas que el formato exige.** Sin el manifiesto no instala; sin el
	// binario en su sitio, instala y no arranca.
	for _, hace := range []string{"manifest.json", "icon.png", "server/esfinge-mcp"} {
		if !dentro[hace] {
			t.Errorf("al paquete le falta %q: lleva %v", hace, claves(dentro))
		}
	}

	crudo := leerDelZip(t, &z.Reader, "manifest.json")
	var m struct {
		ManifestVersion string `json:"manifest_version"`
		Nombre          string `json:"name"`
		Version         string `json:"version"`
		Servidor        struct {
			Tipo      string `json:"type"`
			Entrada   string `json:"entry_point"`
			MCPConfig struct {
				Comando string `json:"command"`
			} `json:"mcp_config"`
		} `json:"server"`
	}
	if err := json.Unmarshal(crudo, &m); err != nil {
		t.Fatalf("el manifiesto no se entiende: %v", err)
	}
	// **La versión es la de la publicación.** Escrita a mano se quedaría vieja sin que
	// nadie lo notara, y entonces Claude Desktop no vería que hay una nueva.
	if m.Version != "9.9.9" {
		t.Errorf("la versión del manifiesto es %q y tenía que ser la que se le pasa", m.Version)
	}
	if m.ManifestVersion == "" || m.Nombre == "" {
		t.Errorf("al manifiesto le falta lo básico: %+v", m)
	}
	if m.Servidor.Tipo != "binary" {
		t.Errorf("el tipo de servidor es %q y aquí es un binario compilado", m.Servidor.Tipo)
	}
	// **`${__dirname}` no es opcional**: es la carpeta donde Claude Desktop lo instale,
	// y sin eso no hay forma de escribir la ruta del comando, porque no se sabe.
	if !strings.Contains(m.Servidor.MCPConfig.Comando, "${__dirname}") {
		t.Errorf("el comando no usa ${__dirname}: %q", m.Servidor.MCPConfig.Comando)
	}
	if !dentro[m.Servidor.Entrada] {
		t.Errorf("el manifiesto dice que el servidor está en %q y ahí no hay nada", m.Servidor.Entrada)
	}
	// **Y el servidor sale ejecutable del zip.** Lo que se instala en macOS y en Linux
	// es esto, y sin el bit puesto Claude Desktop lo instala y no arranca — un fallo
	// que no se ve desde aquí y que el `zip` de antes daba por hecho.
	if modo := modos[m.Servidor.Entrada]; modo&0o111 == 0 {
		t.Errorf("el servidor va en el zip con permisos %v y no se podrá ejecutar", modo)
	}
}

func leerDelZip(t *testing.T, z *zip.Reader, nombre string) []byte {
	t.Helper()
	f, err := z.Open(nombre)
	if err != nil {
		t.Fatalf("no está %q en el paquete", nombre)
	}
	defer f.Close()
	crudo, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return crudo
}

// **Las tres formas de instalarlo tienen que nombrar el mismo binario.**
//
// Hay tres caminos para que un agente encuentre a Esfinge —el paquete que se arrastra a
// Claude Desktop, la orden de Claude Code y el bloque de JSON para lo demás— y los tres
// se escriben en sitios distintos: el manifiesto del paquete, el guion que lo arma y este
// Go. **Cambiar el nombre del binario en uno y no en los otros no da ningún error**: el
// paquete se instala igual y el servidor no arranca, o la orden añade un `esfinge` que
// apunta a nada.
//
// Y las dos que salen de aquí llevan **la misma ruta absoluta**, que es lo que de verdad
// se rompe: la del JSON se escribió primero y la orden se añadió después copiándola.
func TestLasTresFormasDeInstalarloNombranLoMismo(t *testing.T) {
	manifiesto := filepath.Join(raizDelRepo(t), "empaquetado", "mcpb", "manifest.json")
	crudo, err := os.ReadFile(manifiesto)
	if err != nil {
		t.Fatalf("no se puede leer el manifiesto: %v", err)
	}
	var m struct {
		Server struct {
			EntryPoint string `json:"entry_point"`
			MCPConfig  struct {
				Command string `json:"command"`
			} `json:"mcp_config"`
		} `json:"server"`
	}
	if err := json.Unmarshal(crudo, &m); err != nil {
		t.Fatalf("el manifiesto no se entiende: %v", err)
	}

	// En el paquete el binario va dentro, así que lo que se compara es el nombre.
	nombre := filepath.Base(m.Server.EntryPoint)
	if nombre != "esfinge-mcp" {
		t.Errorf("el manifiesto arranca %q y el binario se llama esfinge-mcp", nombre)
	}
	if filepath.Base(m.Server.MCPConfig.Command) != nombre {
		t.Errorf("el manifiesto dice %q en entry_point y %q en command",
			m.Server.EntryPoint, m.Server.MCPConfig.Command)
	}

	// Y lo que enseña la ventana: la orden y el JSON, con la misma ruta entera.
	ruta, err := rutaDelServidorMCP()
	if err != nil {
		t.Skipf("aquí no se sabe dónde está el binario: %v", err)
	}
	if !filepath.IsAbs(ruta) {
		t.Fatalf("la ruta tendría que ser absoluta y es %q", ruta)
	}
	orden := OrdenParaClaudeCode()
	if orden != "claude mcp add esfinge "+ruta {
		t.Errorf("la orden de Claude Code es %q y la ruta es %q", orden, ruta)
	}
	if !strings.Contains(ConfiguracionParaElCliente(), ruta) {
		t.Errorf("el bloque de configuración no lleva %q:\n%s", ruta, ConfiguracionParaElCliente())
	}
	if filepath.Base(ruta) != nombre && filepath.Base(ruta) != nombre+".exe" {
		t.Errorf("la ventana apunta a %q y el paquete arranca %q", filepath.Base(ruta), nombre)
	}
}
