package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// La ruta del servidor MCP es lo que una persona pega en la configuración de otro programa,
// así que lo que se comprueba aquí no es que la función devuelva algo: es que **lo que
// devuelve siga valiendo mañana**, y que cuando no pueda lo diga.

func TestFueraDeUnaAppImageLaRutaEsLaDeAlLadoYNoSeCopiaNada(t *testing.T) {
	casa := t.TempDir()
	ruta, problema := laRutaUsable(
		func() (string, error) { return "/usr/bin/esfinge-mcp", nil },
		false,
		func() (string, error) { return casa, nil },
	)
	if ruta != "/usr/bin/esfinge-mcp" {
		t.Fatalf("instalada de siempre, la ruta es la de al lado y no «%s»", ruta)
	}
	if problema != "" {
		t.Fatalf("y no hay nada que avisar, pero dice: %s", problema)
	}
	// Y no se toca la carpeta de nadie: copiar en ~/.local/bin a quien no lo necesita
	// sería dejarle un binario suelto en el PATH sin que lo haya pedido.
	if _, err := os.Stat(filepath.Join(casa, ".local", "bin", "esfinge-mcp")); err == nil {
		t.Fatal("ha dejado una copia en ~/.local/bin sin hacer falta")
	}
}

func TestEnUnaAppImageLaRutaEsLaCopiaEstableYEsEjecutable(t *testing.T) {
	casa := t.TempDir()
	dentro := filepath.Join(t.TempDir(), "esfinge-mcp")
	if err := os.WriteFile(dentro, []byte("#!/bin/sh\necho yo soy el servidor\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	ruta, problema := laRutaUsable(
		func() (string, error) { return dentro, nil },
		true,
		func() (string, error) { return casa, nil },
	)
	if problema != "" {
		t.Fatalf("debería haber podido, y dice: %s", problema)
	}
	quiero := filepath.Join(casa, ".local", "bin", "esfinge-mcp")
	if ruta != quiero {
		t.Fatalf("la ruta estable es %s, no «%s»", quiero, ruta)
	}

	// **Que exista no basta**: si no se puede ejecutar, el cliente MCP dice «no arranca»
	// y eso no se parece a la causa. Es el mismo fallo que el bit de ejecución del zip de
	// los .mcpb.
	fi, err := os.Stat(ruta)
	if err != nil {
		t.Fatalf("no ha dejado la copia: %v", err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Fatalf("la copia no se puede ejecutar (%v)", fi.Mode().Perm())
	}
	if b, err := os.ReadFile(ruta); err != nil || !strings.Contains(string(b), "yo soy el servidor") {
		t.Fatalf("la copia no es el binario de dentro: %v", err)
	}

	// Y la ruta que va dentro de los bloques que se pegan **tiene que ser ésta**, no la
	// del montaje: es lo que hace que la configuración de Claude Desktop siga valiendo.
	if !strings.HasPrefix(ruta, casa) {
		t.Fatal("la ruta que se pega apunta fuera de la carpeta personal")
	}
}

func TestLaCopiaNoSeRepiteSiYaEsLaMismaYSeRefrescaSiCambia(t *testing.T) {
	casa := t.TempDir()
	dentro := filepath.Join(t.TempDir(), "esfinge-mcp")
	if err := os.WriteFile(dentro, []byte("la 2.65.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	pedir := func() string {
		r, p := laRutaUsable(func() (string, error) { return dentro, nil }, true,
			func() (string, error) { return casa, nil })
		if p != "" {
			t.Fatalf("no debería quejarse: %s", p)
		}
		return r
	}

	ruta := pedir()
	antes, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}

	// Segunda vez: **no se vuelve a copiar**. Esto se llama cada vez que alguien mira
	// Ajustes y el binario son diez megas.
	//
	// Se comprueba por el inodo y no por la fecha: la copia nueva lleva a propósito la
	// fecha del origen, así que mirar la fecha no distinguiría «no ha hecho nada» de «lo
	// ha copiado otra vez», que es justo lo que se quiere separar. Es la trampa del doble
	// que apunta el argumento y no dice si le han llamado.
	pedir()
	despues, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if !mismoFichero(t, antes, despues) {
		t.Fatal("ha vuelto a copiar un binario que ya estaba igual")
	}

	// Y al actualizar Esfinge, la copia tiene que ponerse al día: si no, quien use la
	// AppImage se queda con el servidor de la versión de antes y con su lista de
	// herramientas, que es el fallo que el relevo del .mcpb vino a arreglar.
	//
	// El contenido tiene otro tamaño **y** otra fecha a propósito: con el mismo tamaño y
	// la fecha de antes, esto no puede distinguirse y es una limitación escrita.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(dentro, []byte("la 2.66.0, que trae una herramienta más"), 0o755); err != nil {
		t.Fatal(err)
	}
	pedir()
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "la 2.66.0, que trae una herramienta más" {
		t.Fatalf("la copia se ha quedado en la versión de antes: %q", string(b))
	}
}

func TestSiNoSePuedeDejarEnUnSitioFijoSeDiceYNoSeCalla(t *testing.T) {
	// Lo que falla es el caso real: la AppImage se desmontó, o el binario no está donde
	// se esperaba. **No se puede devolver la ruta del montaje en silencio**: el programa
	// de al lado diría «el servidor no arranca» y nadie lo relacionaría con esto.
	ruta, problema := laRutaUsable(
		func() (string, error) { return "/tmp/.mount_Esfingabc/usr/bin/esfinge-mcp", nil },
		true,
		func() (string, error) { return t.TempDir(), nil },
	)
	if problema == "" {
		t.Fatal("no ha podido dejar la copia y no lo dice")
	}
	if !strings.Contains(problema, "AppImage") {
		t.Fatalf("el aviso tiene que decir por qué pasa esto: %q", problema)
	}
	if ruta == "" {
		t.Fatal("y aun así hay que dar algo, que es mejor que un hueco")
	}

	// Y sin carpeta personal, lo mismo.
	_, problema = laRutaUsable(
		func() (string, error) { return "/x/esfinge-mcp", nil },
		true,
		func() (string, error) { return "", errors.New("no hay HOME") },
	)
	if problema == "" {
		t.Fatal("sin carpeta personal tampoco se puede, y tampoco se dice")
	}
}

func TestSinSaberDondeEstoyLaRutaEsElNombreASecas(t *testing.T) {
	ruta, problema := laRutaUsable(
		func() (string, error) { return "", errors.New("no sé quién soy") },
		true,
		func() (string, error) { return t.TempDir(), nil },
	)
	if ruta != "esfinge-mcp" {
		t.Fatalf("lo honrado es el nombre a secas, no «%s»", ruta)
	}
	if problema != "" {
		t.Fatalf("y no se inventa un aviso sobre la AppImage: %q", problema)
	}
}

// TestLosBloquesQueSePeganLlevanLaMismaRutaQueSeEnsena comprueba que lo que llevan dentro
// los tres bloques de configuración y la orden de Claude Code es la ruta estable, la misma
// que el estado enseña suelta.
//
// Es la trampa de siempre en esta casa: **lo que se escribe por separado se desalinea
// solo**. Si alguien arregla la ruta del campo `Ruta` y deja un bloque llamando a la de al
// lado, la pantalla enseñaría una cosa y lo que se pega sería otra —y en una AppImage eso
// es una configuración que deja de valer al cerrar Esfinge—.
//
// # Por qué esto lee el fuente y no compara los valores
//
// La primera versión comparaba lo que devuelven las cuatro funciones, y **pasaba en verde
// con el fallo dentro**: en el binario de pruebas no hay `APPIMAGE`, así que la ruta estable
// y la de al lado son la misma y comparar no distingue nada. Es exactamente la familia de
// `soloLectura` —una guarda cuyo caso límite no puede ocurrir donde corre la prueba— y lo
// dijo mutarlo, no leerlo.
//
// Comprobado al escribirla: devolviendo `rutaDelServidorMCP()` en vez de la estable, se
// pone roja.
func TestLosBloquesQueSePeganLlevanLaMismaRutaQueSeEnsena(t *testing.T) {
	crudo, err := os.ReadFile("agente.go")
	if err != nil {
		t.Fatal(err)
	}
	fuente := string(crudo)

	// **Un solo sitio del que los demás preguntan.** `rutaDelServidorMCPOEsfingeMCP` tiene
	// que salir de `rutaUsableDelServidorMCP` y de ninguna otra parte.
	cuerpo := entre(t, fuente, "func rutaDelServidorMCPOEsfingeMCP() string {", "\n}")
	if !strings.Contains(cuerpo, "rutaUsableDelServidorMCP()") {
		t.Fatalf("la ruta que se pega ya no sale de rutaUsableDelServidorMCP:\n%s", cuerpo)
	}
	if strings.Contains(cuerpo, "rutaDelServidorMCP()") {
		t.Fatalf("vuelve a dar la ruta de al lado, que en una AppImage caduca al cerrar:\n%s", cuerpo)
	}

	// Y los tres bloques y el estado piden **esa** y no la de al lado.
	for _, quien := range []string{
		"func ConfiguracionParaElCliente() string {",
		"func ConfiguracionParaVSCode() string {",
		"func OrdenParaClaudeCode() string {",
	} {
		cuerpo := entre(t, fuente, quien, "\n}")
		if !strings.Contains(cuerpo, "rutaDelServidorMCPOEsfingeMCP()") {
			t.Fatalf("%s no pide la ruta por el único sitio que la sabe:\n%s", quien, cuerpo)
		}
	}

	// Lo de los valores se queda **además**, que no estorba y cazaría una divergencia el
	// día que una prueba corra con APPIMAGE puesto.
	ruta := rutaDelServidorMCPOEsfingeMCP()
	for nombre, bloque := range map[string]string{
		"Cursor":      ConfiguracionParaElCliente(),
		"VS Code":     ConfiguracionParaVSCode(),
		"Claude Code": OrdenParaClaudeCode(),
	} {
		if !strings.Contains(bloque, ruta) {
			t.Fatalf("el bloque de %s no lleva la ruta que se enseña (%q):\n%s", nombre, ruta, bloque)
		}
	}
}

// entre saca lo que hay entre dos marcas, y **falla si no están**: una prueba que lee el
// fuente y no encuentra lo que busca tiene que ponerse roja, no pasar por vacío.
func entre(t *testing.T, texto, desde, hasta string) string {
	t.Helper()
	i := strings.Index(texto, desde)
	if i < 0 {
		t.Fatalf("no encuentro «%s» en el fuente: si se ha renombrado, esta prueba hay que ponerla al día", desde)
	}
	resto := texto[i+len(desde):]
	j := strings.Index(resto, hasta)
	if j < 0 {
		t.Fatalf("no encuentro el final de «%s»", desde)
	}
	return resto[:j]
}

func mismoFichero(t *testing.T, a, b os.FileInfo) bool {
	t.Helper()
	ia, oka := inodo(a)
	ib, okb := inodo(b)
	if !oka || !okb {
		t.Skip("este sistema no da el inodo, así que esto no se puede distinguir aquí")
	}
	return ia == ib
}
