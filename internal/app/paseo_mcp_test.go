package app

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/agente"
	"github.com/webcafeina/esfinge/internal/boveda"
)

// Un paseo a mano por el servidor MCP, para **leer la conversación tal cual la vería un
// agente**. No afirma casi nada: lo que hace es enseñarla, con
//
//	go test ./internal/app -run TestPaseoMCPAMano -v
//
// **Y existe porque encontró dos cosas que ninguna prueba veía** (2026-10-07): que el
// binario **no se emparejaba nunca** —así que un agente solo podía preguntar el estado—
// y que el nombre del cliente no llegaba a la ventana. Las dos estaban cubiertas «por
// pruebas» que ponían el testigo a mano, incluida la de la tubería. Leer la conversación
// entera es lo que las destapó, y por eso esto se queda: lo que una tabla de casos no
// enseña es **cómo se lee todo junto**.
func TestPaseoMCPAMano(t *testing.T) {
	if testing.Short() {
		t.Skip("es para leerlo, no para la puerta")
	}
	a, _, _, _, _ := conBoveda(t)
	for _, e := range []boveda.Entrada{
		{Tipo: boveda.TipoCredencial, Titulo: "Hacienda", Usuario: "nif@webcafeina.com",
			Secreto: "la misma de siempre", Sitios: []string{"https://agenciatributaria.gob.es"}},
		{Tipo: boveda.TipoCredencial, Titulo: "Brevo", Usuario: "info@webcafeina.com",
			Secreto: "la misma de siempre", Sitios: []string{"https://brevo.com"}},
	} {
		if err := a.GuardarEnBoveda(e); err != nil {
			t.Fatal(err)
		}
	}
	socket := filepath.Join(t.TempDir(), "a.sock")
	srv, err := agente.Servir(socket, fuenteDelAgente{a})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Parar()
	if err := a.PermitirAgente(); err != nil {
		t.Fatal(err)
	}

	decir := func(titulo string, lineas ...string) []map[string]any {
		t.Helper()
		var sale bytes.Buffer
		if err := agente.Traducir(strings.NewReader(strings.Join(lineas, "\n")+"\n"), &sale, "2.44.0", socket); err != nil {
			t.Fatal(err)
		}
		t.Logf("\n━━━ %s ━━━", titulo)
		var out []map[string]any
		for i, l := range strings.Split(strings.TrimSpace(sale.String()), "\n") {
			if l == "" {
				continue
			}
			var m map[string]any
			_ = json.Unmarshal([]byte(l), &m)
			out = append(out, m)
			if r, ok := m["result"].(map[string]any); ok {
				if c, ok := r["content"].([]any); ok && len(c) > 0 {
					esError := ""
					if r["isError"] == true {
						esError = "  [isError]"
					}
					t.Logf("  → %v%s", c[0].(map[string]any)["text"], esError)
					continue
				}
			}
			t.Logf("  → %s", recortar(l, 300))
			_ = i
		}
		return out
	}

	decir("1 · el saludo",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","clientInfo":{"name":"claude-code"}}}`)

	// Las descripciones, que es lo único que el agente lee para decidir.
	t.Log("\n━━━ 2 · las herramientas, como las lee un agente ━━━")
	for _, h := range agente.LasHerramientas {
		marcas := []string{}
		if h.PideAprobacion {
			marcas = append(marcas, "pide un sí")
		}
		if h.DevuelveSecreto {
			marcas = append(marcas, "SE LE ENSEÑA")
		}
		if h.Escribe {
			marcas = append(marcas, "escribe")
		}
		etiqueta := ""
		if len(marcas) > 0 {
			etiqueta = "   [" + strings.Join(marcas, " · ") + "]"
		}
		t.Logf("\n  %s%s\n    %s", h.Nombre, etiqueta, h.Descripcion)
	}

	decir("3 · estado", peticion(1, "esfinge_estado", nil))
	decir("4 · buscar «hacienda»", peticion(2, "esfinge_buscar", map[string]any{"texto": "hacienda"}))
	decir("5 · la higiene", peticion(3, "esfinge_higiene", nil))

	lista, _ := a.BuscarEnBoveda("Hacienda")
	id := lista[0].ID
	decir("6 · copiar la contraseña, la primera vez", peticion(4, "esfinge_copiar_contrasena", map[string]any{"id": id}))

	t.Logf("\n━━━ 7 · la persona dice que sí en la ventana ━━━\n  (Ajustes → «Claude Code quiere la contraseña de «Hacienda»» → Solo ésta)")
	if err := a.AprobarLoQuePideElAgente(false); err != nil {
		t.Fatal(err)
	}

	decir("8 · y ahora sí", peticion(5, "esfinge_copiar_contrasena", map[string]any{"id": id}))
	decir("9 · crear una cuenta con contraseña generada",
		peticion(6, "esfinge_crear", map[string]any{"titulo": "Cliente nuevo", "usuario": "yo@ahí.com", "generar": true}))

	t.Logf("\n━━━ 10 · y lo que queda apuntado en la bóveda ━━━")
	r := a.EstadoDelAgente().Dado
	for _, ap := range r {
		t.Logf("  %s · %s · %q · %s (%s)", ap.Cuando, ap.Quien, ap.Titulo, ap.Resultado, ap.Como)
	}
}

func peticion(id int, nombre string, args map[string]any) string {
	p := map[string]any{"name": nombre}
	if args != nil {
		p["arguments"] = args
	}
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": p})
	return string(b)
}

func recortar(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
