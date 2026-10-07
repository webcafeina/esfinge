package agente

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// hablar mete esas líneas por la entrada y devuelve lo que sale, una respuesta por
// línea.
func hablar(t *testing.T, llamar Llamar, lineas ...string) []map[string]any {
	t.Helper()
	var sale bytes.Buffer
	if err := Hablar(strings.NewReader(strings.Join(lineas, "\n")+"\n"), &sale, "2.44.0", llamar); err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(sale.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("lo que ha salido no es JSON: %q", l)
		}
		out = append(out, m)
	}
	return out
}

// **El saludo y la lista se contestan sin Esfinge**, y ésta es la prueba que vigila esa
// restricción (ADR 0054).
//
// Claude Code pide `tools/list` al abrir la sesión y **se queda con lo que le demos**:
// si esto dependiera del socket, un arranque con la ventana cerrada dejaría al agente
// sin herramientas toda la sesión, aunque se abriera dos minutos después. Por eso el
// catálogo son datos de este paquete, y por eso la llamada de abajo **falla a
// propósito**: si alguien mueve la tabla a `internal/app`, esto se pone rojo.
func TestElSaludoYLaListaSeContestanSinEsfinge(t *testing.T) {
	sinNadieAlOtroLado := func(Herramienta, map[string]any) Resultado {
		t.Error("tools/list no puede necesitar a Esfinge")
		return fallo("no")
	}
	rs := hablar(t, sinNadieAlOtroLado,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	if len(rs) != 2 {
		t.Fatalf("han salido %d respuestas: %+v", len(rs), rs)
	}

	saludo := rs[0]["result"].(map[string]any)
	if saludo["protocolVersion"] != RevisionDeMCP {
		t.Errorf("el saludo dice la revisión %v", saludo["protocolVersion"])
	}
	if si := saludo["serverInfo"].(map[string]any); si["name"] != NombreDelServidor || si["version"] != "2.44.0" {
		t.Errorf("el saludo dice %+v", si)
	}

	lista := rs[1]["result"].(map[string]any)["tools"].([]any)
	if len(lista) != len(LasHerramientas) {
		t.Fatalf("la lista trae %d herramientas y el catálogo tiene %d", len(lista), len(LasHerramientas))
	}
}

// **Una notificación no se contesta.** Mandar algo a un mensaje sin identificador rompe
// a los clientes estrictos, y es la clase de cosa que solo se ve con un cliente de
// verdad delante.
func TestUnaNotificacionNoSeContesta(t *testing.T) {
	rs := hablar(t, nil,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":7,"method":"ping"}`,
	)
	if len(rs) != 1 {
		t.Fatalf("han salido %d respuestas y tenía que salir una: %+v", len(rs), rs)
	}
	if rs[0]["id"] != float64(7) {
		t.Errorf("la respuesta que ha salido no es la del ping: %+v", rs[0])
	}
}

// Lo que no se puede atender va **como resultado con `isError`**, no como error de
// JSON-RPC: es la convención de MCP para lo que el modelo tiene que leer y explicar.
func TestUnaHerramientaQueNoExisteSaleComoResultado(t *testing.T) {
	rs := hablar(t, nil, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"esfinge_inventada"}}`)
	r := rs[0]["result"].(map[string]any)
	if r["isError"] != true {
		t.Fatalf("no viene marcado como error: %+v", rs[0])
	}
	if rs[0]["error"] != nil {
		t.Error("ha salido además como error de protocolo, y eso es lo que no hay que hacer")
	}
}

// Y el catálogo tiene que cumplir lo que los clientes exigen y lo que esta casa exige.
func TestElCatalogoEstaBienEscrito(t *testing.T) {
	// **Sin `ñ` ni acentos**: los clientes validan contra esto y una tilde deja la
	// herramienta fuera sin decir por qué.
	comoLoQuierenLosClientes := regexp.MustCompile(`^esfinge_[a-z0-9_]+$`)
	vistos := map[string]bool{}
	for _, h := range LasHerramientas {
		if !comoLoQuierenLosClientes.MatchString(h.Nombre) {
			t.Errorf("el nombre %q no vale: los clientes solo admiten letras sin acentos, cifras y guiones bajos", h.Nombre)
		}
		if vistos[h.Nombre] {
			t.Errorf("la herramienta %q está dos veces", h.Nombre)
		}
		vistos[h.Nombre] = true
		if strings.TrimSpace(h.Descripcion) == "" {
			t.Errorf("la herramienta %q no tiene descripción, que es lo único que el modelo lee para decidir", h.Nombre)
		}
		if h.Verbo == "" {
			t.Errorf("la herramienta %q no dice qué verbo pide", h.Nombre)
		}
	}
	// Y toda herramienta pide un verbo **que existe**: una que pidiera uno que no está
	// en la lista contestaría «no entiendo» a quien la llamara, y eso no lo ve nadie
	// hasta que un agente lo intenta.
	for _, h := range LasHerramientas {
		hay := false
		for _, q := range LoQueSePuedePedir {
			if q == h.Verbo {
				hay = true
			}
		}
		if !hay {
			t.Errorf("la herramienta %q pide el verbo %q, que no está en la lista", h.Nombre, h.Verbo)
		}
	}
}

// **Sin Esfinge al otro lado se contesta, y se dice qué hacer.** Un servidor MCP que se
// muere cuando la aplicación está cerrada deja al agente sin saber por qué.
func TestSinEsfingeSeContestaIgual(t *testing.T) {
	var sale bytes.Buffer
	err := Traducir(
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"esfinge_buscar","arguments":{"texto":"x"}}}`+"\n"),
		&sale, "2.44.0", "/no/existe/este.sock")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sale.String(), "Esfinge no está abierta") {
		t.Fatalf("no dice qué pasa: %s", sale.String())
	}
	if !strings.Contains(sale.String(), `"isError":true`) {
		t.Errorf("no viene marcado como error: %s", sale.String())
	}
}
