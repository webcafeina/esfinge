package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/agente"
)

// **La tubería entera de un agente**, con bytes de MCP por donde entran de verdad.
//
// Las piezas estaban probadas una a una —el catálogo, el servidor, la fuente— y eso ya
// se ha demostrado dos veces en este proyecto que no basta: los iconos sobrevivieron
// dos fallos seguidos con todas sus piezas en verde, y el canal del navegador tiene
// esta misma prueba por lo mismo.
//
// Aquí entra un JSON-RPC por una tubería, cruza el socket, llega a una bóveda de verdad
// y vuelve enmarcado. Lo que comprueba, por orden de lo que importa:
//
//  1. Que **el saludo y la lista se contestan con Esfinge cerrada**, que es la
//     restricción de la ADR 0054 y lo que ninguna otra prueba ve de punta a punta.
//  2. Que sin emparejar **no sale nada**.
//  3. Que emparejado, la búsqueda trae el inventario.
//  4. Y que **en los bytes que salen hacia el modelo no hay ni un secreto**.
func TestLaTuberiaEnteraDeUnAgente(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)

	// El socket, en una carpeta temporal: la ruta de verdad es de la configuración
	// del usuario y aquí no se toca.
	socket := filepath.Join(t.TempDir(), "a.sock")
	srv, err := agente.Servir(socket, fuenteDelAgente{a})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Parar()

	// --- 1. El saludo y la lista, **sin Esfinge**: contra un socket que no existe.
	sinEsfinge := hablarMCP(t, filepath.Join(t.TempDir(), "no-existe.sock"),
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"esfinge_buscar","arguments":{"texto":"Banco"}}}`,
	)
	lista := sinEsfinge[1]["result"].(map[string]any)["tools"].([]any)
	if len(lista) != len(agente.LasHerramientas) {
		t.Fatalf("con Esfinge cerrada la lista trae %d herramientas: un agente que arranque "+
			"así se queda sin ellas toda la sesión", len(lista))
	}
	// Y lo que sí necesita a Esfinge dice que no está, **y dice qué hacer**.
	if r := sinEsfinge[2]["result"].(map[string]any); r["isError"] != true {
		t.Errorf("sin Esfinge, buscar no se marca como error: %+v", r)
	}

	// --- 2. Con Esfinge, pero sin que nadie haya dicho que sí: no sale nada, **y se
	// dice qué hacer**. El testigo lo pide el propio traductor, así que esto recorre el
	// emparejamiento de verdad.
	//
	// **Esto es lo que faltaba y costó encontrar recorriendo el camino a mano**: antes
	// esta prueba hablaba con `Atender` poniendo el testigo ella misma, así que el
	// binario podía no pedirlo nunca —y no lo pedía— sin que nada se pusiera rojo.
	enConfiguracionDePruebas(t) // que el testigo no se mezcle con el de verdad
	rs := hablarMCP(t, socket,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"esfinge_buscar","arguments":{"texto":"Banco"}}}`,
	)
	if texto := primerTexto(t, rs[0]); !strings.Contains(texto, "permite este agente") &&
		!strings.Contains(texto, "Permite este agente") {
		t.Fatalf("sin permiso contesta %q", texto)
	}

	// --- 3. La persona dice que sí en la ventana.
	if err := a.PermitirAgente(); err != nil {
		t.Fatal(err)
	}

	// --- 4. Y ahora la búsqueda trae el inventario, **por la tubería entera**: el
	// traductor recoge el testigo él solo y reintenta.
	rs = hablarMCP(t, socket,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","clientInfo":{"name":"Claude Code"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"esfinge_buscar","arguments":{"texto":"Banco"}}}`,
	)
	elTexto := primerTexto(t, rs[1])
	if strings.Contains(elTexto, "permite este agente") || strings.Contains(elTexto, "Permite este agente") {
		t.Fatalf("tras decir que sí sigue sin emparejarse: %q", elTexto)
	}
	if !strings.Contains(elTexto, "Banco") {
		t.Fatalf("la búsqueda no ha traído el Banco: %q", elTexto)
	}

	// --- 5. **Y el nombre del saludo llega hasta la ventana**, para que diga «Claude
	// Code quiere…» y no «un agente». Se pide algo que haya que aprobar y se mira quién
	// lo pide.
	enLaBoveda, err := a.BuscarEnBoveda("Banco")
	if err != nil || len(enLaBoveda) == 0 {
		t.Fatal(err)
	}
	hablarMCP(t, socket,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","clientInfo":{"name":"Claude Code"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"esfinge_copiar_contrasena","arguments":{"id":"`+enLaBoveda[0].ID+`"}}}`,
	)
	if q := a.EstadoDelAgente().Quiere; q == nil || q.Quien != "Claude Code" {
		t.Errorf("la ventana no sabe quién lo pide: %+v", q)
	}

	// --- 6. Y lo de siempre por el canal, para mirar los bytes de cerca.
	f := fuenteDelAgente{a}
	testigo, err := f.Emparejar("Claude Code")
	if err != nil {
		// Ya lo recogió el traductor: se pide otro permiso.
		if err := a.PermitirAgente(); err != nil {
			t.Fatal(err)
		}
		testigo, err = f.Emparejar("Claude Code")
		if err != nil {
			t.Fatal(err)
		}
	}
	r := srv.Atender(agente.Peticion{
		Version: agente.VersionDelProtocolo, Que: agente.QueBuscar, Testigo: testigo, Texto: "Banco",
	})
	if !r.OK {
		t.Fatalf("la búsqueda ha fallado: %+v", r)
	}
	if len(r.Entradas) == 0 {
		t.Fatal("la búsqueda no ha traído nada")
	}
	hayBanco := false
	for _, e := range r.Entradas {
		if e.Titulo == "Banco" {
			hayBanco = true
			if !e.TieneSecreto || !e.TieneCodigo {
				t.Errorf("las marcas del Banco dicen %+v, y tiene las dos cosas", e)
			}
		}
	}
	if !hayBanco {
		t.Errorf("no está el Banco: %+v", r.Entradas)
	}

	// --- 5. **Y ni un secreto en los bytes.** Sobre lo serializado, no sobre los
	// campos: un campo nuevo con un secreto dentro pasaría mirando los campos.
	crudo, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, secreto := range []string{"s3cr3t0", "GEZDGNBVGY3TQOJQ", "4111111111111111"} {
		if bytes.Contains(crudo, []byte(secreto)) {
			t.Errorf("lo que sale hacia el agente lleva %q dentro: %s", secreto, crudo)
		}
	}
}

// hablarMCP mete esas líneas por el traductor y devuelve lo que sale.
func hablarMCP(t *testing.T, socket string, lineas ...string) []map[string]any {
	t.Helper()
	var sale bytes.Buffer
	if err := agente.Traducir(strings.NewReader(strings.Join(lineas, "\n")+"\n"), &sale, "pruebas", socket); err != nil {
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

func primerTexto(t *testing.T, m map[string]any) string {
	t.Helper()
	r, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("la respuesta no trae resultado: %+v", m)
	}
	c, ok := r["content"].([]any)
	if !ok || len(c) == 0 {
		t.Fatalf("el resultado no trae contenido: %+v", r)
	}
	return fmt.Sprint(c[0].(map[string]any)["text"])
}
