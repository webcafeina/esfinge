package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/agente"
	"github.com/webcafeina/esfinge/internal/mcpb"
)

// **Las tres formas de instalarlo tienen que nombrar el mismo binario.**
//
// Hay varios caminos para que un agente encuentre a Esfinge —el paquete que se arrastra
// a Claude Desktop, la orden de Claude Code, el bloque de Cursor y el de VS Code— y se
// escriben en sitios distintos. **Cambiar el nombre del binario en uno y no en los
// otros no da ningún error**: el paquete se instala igual y el servidor no arranca, o
// la orden añade un `esfinge` que apunta a nada.
func TestLasTresFormasDeInstalarloNombranLoMismo(t *testing.T) {
	ruta, err := rutaDelServidorMCP()
	if err != nil {
		t.Skipf("aquí no se sabe dónde está el binario: %v", err)
	}
	if !filepath.IsAbs(ruta) {
		t.Fatalf("la ruta tendría que ser absoluta y es %q", ruta)
	}

	// **Con `--scope user`**: de fábrica el ámbito es `local`, o sea solo la carpeta
	// donde se pegue la orden, y entonces Esfinge no está en ningún otro proyecto. Lo
	// vio el cliente el 2026-10-08 con el conector recién instalado.
	orden := OrdenParaClaudeCode()
	if orden != "claude mcp add --scope user esfinge "+ruta {
		t.Errorf("la orden de Claude Code es %q y la ruta es %q", orden, ruta)
	}
	for que, bloque := range map[string]string{
		"Claude y Cursor": ConfiguracionParaElCliente(),
		"VS Code":         ConfiguracionParaVSCode(),
	} {
		if !bytes.Contains([]byte(bloque), []byte(ruta)) {
			t.Errorf("el bloque de %s no lleva %q:\n%s", que, ruta, bloque)
		}
	}
	// Y el paquete arranca ese mismo nombre.
	if dentro := mcpb.NombreDeDentro(ruta); filepath.Base(dentro) != filepath.Base(ruta) {
		t.Errorf("la ventana apunta a %q y el paquete arranca %q",
			filepath.Base(ruta), filepath.Base(dentro))
	}
}

// **VS Code lee `servers`, no `mcpServers`, y darle el bloque equivocado no da error.**
//
// Es la trampa de siempre con otra cara: Claude Desktop y Cursor leen `mcpServers` y
// VS Code lee `servers`, así que pegarle a VS Code el bloque de Claude **no falla, no
// carga nada**, y desde fuera parece que Esfinge no funciona ahí. Por eso la pantalla
// enseña los dos con su nombre en vez de uno «genérico», y por eso esto lo vigila: el
// día que alguien los unifique «porque son iguales», se pone rojo.
func TestElBloqueDeVSCodeNoEsElDeClaude(t *testing.T) {
	var vsc struct {
		Servers map[string]struct {
			Tipo    string `json:"type"`
			Comando string `json:"command"`
		} `json:"servers"`
		MCPServers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(ConfiguracionParaVSCode()), &vsc); err != nil {
		t.Fatalf("el bloque de VS Code no se entiende: %v", err)
	}
	if len(vsc.MCPServers) > 0 {
		t.Error("el bloque de VS Code lleva «mcpServers», que es el de Claude: ahí no carga nada")
	}
	e, hay := vsc.Servers["esfinge"]
	if !hay {
		t.Fatalf("el bloque de VS Code no declara «esfinge» bajo «servers»:\n%s", ConfiguracionParaVSCode())
	}
	if e.Tipo != "stdio" {
		t.Errorf("VS Code quiere type=stdio y pone %q", e.Tipo)
	}

	var claude struct {
		MCPServers map[string]struct {
			Comando string `json:"command"`
		} `json:"mcpServers"`
		Servers map[string]any `json:"servers"`
	}
	if err := json.Unmarshal([]byte(ConfiguracionParaElCliente()), &claude); err != nil {
		t.Fatalf("el bloque de Claude no se entiende: %v", err)
	}
	if len(claude.Servers) > 0 {
		t.Error("el bloque de Claude lleva «servers», que es el de VS Code")
	}
	if claude.MCPServers["esfinge"].Comando != e.Comando {
		t.Errorf("los dos bloques apuntan a sitios distintos: %q y %q",
			claude.MCPServers["esfinge"].Comando, e.Comando)
	}
}

// **El botón de guardar el paquete no abre el diálogo si no hay nada que guardar.**
//
// Es la lección de `ExportarLlaves`, que pedía una clave y un sitio para un fichero que
// no iba a existir: **lo que hace falta para decidir se mira antes de pedirle a alguien
// que decida**. Aquí lo que puede faltar es el servidor MCP al lado de Esfinge — y pasa
// de verdad, porque en desarrollo el binario que corre es otro.
//
// El doble **cuenta las llamadas**, que es lo que distingue «no me han llamado» de «me
// han llamado sin carpeta».
func TestGuardarElPaqueteNoPreguntaSiNoHayServidor(t *testing.T) {
	a, s := nuevaDePrueba(t)
	if _, err := a.GuardarPaqueteMCP(); err == nil {
		t.Fatal("sin servidor MCP al lado tendría que negarse")
	}
	if s.vecesGuardar != 0 {
		t.Errorf("ha abierto el diálogo %d veces y no tenía nada que guardar",
			s.vecesGuardar)
	}
}

// Y cuando sí está, lo escribe entero y se puede abrir.
func TestGuardarElPaqueteEscribeUnZipQueSeAbre(t *testing.T) {
	a, s := nuevaDePrueba(t)

	// El servidor «instalado» al lado del binario que corre esta prueba.
	ruta, err := rutaDelServidorMCP()
	if err != nil {
		t.Skipf("aquí no se sabe dónde está el binario: %v", err)
	}
	if err := os.WriteFile(ruta, []byte("un servidor de mentira"), 0o755); err != nil {
		t.Skipf("no se puede dejar un servidor de mentira ahí: %v", err)
	}
	defer os.Remove(ruta)

	destino := filepath.Join(t.TempDir(), "Esfinge.mcpb")
	s.guardaEn = destino
	hecho, err := a.GuardarPaqueteMCP()
	if err != nil {
		t.Fatal(err)
	}
	if hecho != destino {
		t.Errorf("dice que lo ha dejado en %q y se pidió %q", hecho, destino)
	}
	z, err := zip.OpenReader(destino)
	if err != nil {
		t.Fatalf("lo guardado no es un zip: %v", err)
	}
	defer z.Close()
	dentro := map[string]bool{}
	for _, f := range z.File {
		dentro[f.Name] = true
	}
	for _, hace := range []string{"manifest.json", "icon.png", mcpb.NombreDeDentro(ruta)} {
		if !dentro[hace] {
			t.Errorf("al paquete guardado le falta %q", hace)
		}
	}
}

// **Lo que el núcleo sabe aplicar tiene que estar declarado en el esquema**, porque el
// esquema es lo único que el modelo ve.
//
// `crear` y `editar` aceptaban `sitios` y `etiquetas` desde el principio —el código los
// aplica— y **no estaban en el esquema**, así que para el agente no existían: creaba
// credenciales sin sitio, que son credenciales que no se rellenan solas, que es para lo
// que existe la bóveda. Lo dijo Claude al crear la primera entrada de verdad, el
// 2026-10-08: «desde aquí no puedo rellenar el campo de sitios».
//
// Esto se lee del propio fuente, como hace el vigilante del puente con `puente.ts`: la
// lista de campos que el código entiende no se puede sacar por reflexión —es un `switch`
// y unos `p.Campos["x"]`— y mantenerla a mano en dos sitios es exactamente el fallo que
// se está arreglando.
func TestLoQueElNucleoAplicaEstaEnElEsquema(t *testing.T) {
	crudo, err := os.ReadFile("agente.go")
	if err != nil {
		t.Fatal(err)
	}
	fuente := string(crudo)

	// **Los campos de cada verbo, no los del fichero entero.** La primera versión de
	// esto juntaba todos y le exigía a `editar` el `tipo` que solo usa `crear`: una
	// prueba que falla por lo que no es manda a arreglar donde no hay nada roto.
	cuerpoDe := func(nombre string) string {
		i := strings.Index(fuente, ") "+nombre+"(")
		if i < 0 {
			t.Fatalf("no se encuentra %s en agente.go: el patrón ya no vale", nombre)
		}
		j := strings.Index(fuente[i:], "\nfunc ")
		if j < 0 {
			return fuente[i:]
		}
		return fuente[i : i+j]
	}
	camposDe := func(cuerpo string) map[string]bool {
		out := map[string]bool{}
		for _, m := range regexp.MustCompile(`p\.Campos\["([a-zñáéíóú]+)"\]`).FindAllStringSubmatch(cuerpo, -1) {
			out[m[1]] = true
		}
		for _, m := range regexp.MustCompile(`case "([a-zñáéíóú]+)":`).FindAllStringSubmatch(cuerpo, -1) {
			out[m[1]] = true
		}
		// `totp` se puede cambiar y **no se ofrece a propósito**: una semilla mal
		// puesta deja una cuenta sin segundo factor y el agente no puede comprobarla.
		delete(out, "totp")
		// `generar` no es un campo de la entrada, es cómo se hace el secreto.
		delete(out, "generar")
		return out
	}

	lee := map[string]map[string]bool{
		agente.QueCrear:  camposDe(cuerpoDe("Crear")),
		agente.QueEditar: camposDe(cuerpoDe("Editar")),
	}
	for verbo, campos := range lee {
		if len(campos) < 3 {
			t.Fatalf("%s: solo se han encontrado %v en el fuente; el patrón ya no vale", verbo, campos)
		}
	}

	// **Cada herramienta por su cuenta.** Juntarlas hace que baste con que un campo
	// esté en una de las dos, y entonces quitarlo de `crear` —que es exactamente el
	// fallo que se está arreglando— no pone nada rojo. Comprobado mutándolo.
	for _, h := range agente.LasHerramientas {
		campos, mira := lee[h.Verbo]
		if !mira {
			continue
		}
		declara := map[string]bool{}
		for campo := range h.Esquema.Propiedades {
			declara[campo] = true
		}
		for campo := range campos {
			if !declara[campo] {
				t.Errorf("%s: el núcleo aplica %q y el esquema no lo declara: para el modelo no existe",
					h.Nombre, campo)
			}
		}
		// Y las dos listas, que no van por el mapa y faltaban en las dos.
		for _, campo := range []string{"sitios", "etiquetas"} {
			if !declara[campo] {
				t.Errorf("%s: falta %q; sin eso una credencial nace sin poder rellenarse sola",
					h.Nombre, campo)
			}
		}
	}
}
