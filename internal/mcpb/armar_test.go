package mcpb

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// **El icono del paquete es una copia, y una copia a ciegas ya costó una versión.**
//
// `go:embed` no puede salir del directorio de su paquete, así que `icono.png` tiene
// que estar aquí. La vez anterior que un icono se copió sin vigilancia —`build/esf.png`
// era `build/appicon.png` byte a byte— se publicaron varias versiones con el documento
// `.esf` luciendo el icono de la aplicación y nadie lo vio. Aquí la copia es a
// propósito, pero **se compara**: si alguien cambia la marca, los dos cambian o esto se
// pone rojo.
func TestElIconoEsElDeLaAplicacion(t *testing.T) {
	deLaApp, err := os.ReadFile(filepath.Join("..", "..", "build", "appicon.png"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(icono, deLaApp) {
		t.Errorf("internal/mcpb/icono.png (%d bytes) ya no es build/appicon.png (%d bytes): "+
			"cópialo otra vez, que es de donde sale", len(icono), len(deLaApp))
	}
}

// **El manifiesto del repositorio nunca se publica tal cual.** Lleva una versión de
// mentira para que armar sin ponerla sea imposible —`Armar` falla si no la encuentra—,
// y ese acuerdo entre el fichero y el código no lo comprueba nadie más.
func TestElManifiestoDelRepositorioLlevaLaVersionDeMentira(t *testing.T) {
	var m struct {
		Version  string `json:"version"`
		Servidor struct {
			Entrada string `json:"entry_point"`
		} `json:"server"`
	}
	if err := json.Unmarshal(manifiesto, &m); err != nil {
		t.Fatalf("el manifiesto embebido no se entiende: %v", err)
	}
	if m.Version != LaVersionDePrueba {
		t.Errorf("el manifiesto dice %q y tendría que decir %q, que es lo que Armar sustituye",
			m.Version, LaVersionDePrueba)
	}
	// Y el nombre de dentro: el manifiesto dice dónde está el servidor y `Armar` lo
	// escribe ahí. Si se separan, el paquete se instala y no arranca.
	if m.Servidor.Entrada != NombreDeDentro("esfinge-mcp") {
		t.Errorf("el manifiesto arranca %q y Armar escribe %q",
			m.Servidor.Entrada, NombreDeDentro("esfinge-mcp"))
	}
}

// **En Windows el de dentro lleva `.exe`, y en ningún otro sitio.**
func TestElNombreDeDentro(t *testing.T) {
	for _, c := range []struct{ binario, quiero string }{
		{"esfinge-mcp", "server/esfinge-mcp"},
		{"/Applications/Esfinge.app/Contents/MacOS/esfinge-mcp", "server/esfinge-mcp"},
		{`C:\Esfinge\esfinge-mcp.exe`, "server/esfinge-mcp.exe"},
		{"esfinge-mcp.EXE", "server/esfinge-mcp.exe"},
	} {
		if hay := NombreDeDentro(c.binario); hay != c.quiero {
			t.Errorf("con %q sale %q y esperaba %q", c.binario, hay, c.quiero)
		}
	}
}

// Y el paquete armado: las tres cosas dentro, la versión puesta y el bit de ejecución.
func TestElPaqueteLlevaLoQueTieneQueLlevar(t *testing.T) {
	var buf bytes.Buffer
	if err := ArmarCon("9.9.9", "server/esfinge-mcp", []byte("binario de mentira"), &buf); err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("no es un zip: %v", err)
	}
	modos := map[string]os.FileMode{}
	for _, f := range z.File {
		modos[f.Name] = f.Mode()
	}
	for _, hace := range []string{"manifest.json", "icon.png", "server/esfinge-mcp"} {
		if _, hay := modos[hace]; !hay {
			t.Errorf("falta %q en el paquete", hace)
		}
	}
	if modos["server/esfinge-mcp"]&0o111 == 0 {
		t.Errorf("el servidor va con %v y no se podrá ejecutar", modos["server/esfinge-mcp"])
	}
	f, _ := z.Open("manifest.json")
	crudo := make([]byte, 4096)
	n, _ := f.Read(crudo)
	if !bytes.Contains(crudo[:n], []byte(`"9.9.9"`)) {
		t.Errorf("la versión no se ha puesto:\n%s", crudo[:n])
	}
}
