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
	// **Y el de Brevo se coge ahora**, con la personal abierta. Más abajo ya no se puede:
	// después de trabajar en una bóveda nombrada la ventana se queda dentro durante el
	// respiro, y buscar «en la personal» busca en la del cliente. Lo enseñó este paseo a
	// la primera.
	idBrevo := idDe(t, a, "Brevo")
	decir("6 · copiar la contraseña, la primera vez", peticion(4, "esfinge_copiar_contrasena", map[string]any{"id": id}))

	t.Logf("\n━━━ 7 · la persona dice que sí en la ventana ━━━" +
		"\n  (la banda de arriba, en cualquier pantalla: «Claude Code quiere la contraseña de" +
		" «Hacienda»» → Solo ésta)")
	if err := aprobar(t, a, false); err != nil {
		t.Fatal(err)
	}

	decir("8 · y ahora sí", peticion(5, "esfinge_copiar_contrasena", map[string]any{"id": id}))
	decir("9 · crear una cuenta con contraseña generada",
		peticion(6, "esfinge_crear", map[string]any{"titulo": "Cliente nuevo", "usuario": "yo@ahí.com", "generar": true}))

	// ---- y el tramo de las bóvedas, que es lo que ha costado dos versiones ----
	//
	// **Va aquí porque lo de las bóvedas se ve entero o no se ve.** El bucle de
	// aprobaciones del 15 y el 16 de octubre vivía justo en esta secuencia —nombrar una
	// bóveda, que pida permiso, aprobar deprisa y reintentar— y el paseo llegaba solo
	// hasta el paso 9, con todo en la personal.
	ref, err := a.CrearProyecto("Acme")
	if err != nil {
		t.Fatal(err)
	}
	decir("11 · ¿qué bóvedas hay?", peticion(7, "esfinge_bovedas", nil))
	decir("12 · crear en la bóveda del cliente, sin salir de la mía",
		peticion(8, "esfinge_crear", map[string]any{
			"boveda": "Acme", "titulo": "Panel de Acme", "usuario": "admin@acme.com", "generar": true}))
	decir("13 · y buscarlo ahí", peticion(9, "esfinge_buscar", map[string]any{"boveda": "Acme", "texto": "Panel"}))

	// El agente deja de nombrar bóveda, así que el respiro se deshace y «sin nombrar»
	// vuelve a querer decir «la mía». Es lo que hace cada llamada sin `boveda`.
	a.volverDeDondeLeDejoElAgente()
	decir("14 · traerse «Brevo» de la mía a la bóveda del cliente", peticion(10, "esfinge_mover",
		map[string]any{"id": idBrevo, "destino": "Acme"}))
	t.Logf("\n━━━ 15 · la persona aprueba **deprisa**, y el respiro vence después ━━━" +
		"\n  (es el orden que daba el bucle: aprobar en menos de diez segundos)")
	if err := aprobar(t, a, false); err != nil {
		t.Fatal(err)
	}
	a.volverDeDondeLeDejoElAgente()
	decir("16 · el reintento, con los mismos argumentos", peticion(11, "esfinge_mover",
		map[string]any{"id": idBrevo, "destino": "Acme"}))

	a.volverDeDondeLeDejoElAgente()
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	llegadas, _ := a.BuscarEnBoveda("Brevo")
	t.Logf("\n━━━ 17 · y «Brevo» está en la bóveda del cliente: %d ━━━", len(llegadas))
	if len(llegadas) != 1 {
		t.Errorf("el movimiento no ha llegado al destino: %d", len(llegadas))
	}
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}

	// **Y aquí dirá «Un agente» y no «Claude Code», y eso no es el fallo del 2026-10-07.**
	// Es de este paseo: `decir` abre una conexión por paso, así que el saludo con el
	// nombre solo entra en la primera y el traductor de las demás nace sin él. En
	// producción el cliente MCP arranca el binario una vez y le habla por el mismo
	// `stdin`. Lo que vigila que el nombre llegue de verdad es la prueba de la tubería,
	// que manda el saludo y la llamada **en la misma conexión**.
	t.Logf("\n━━━ 18 · y lo que queda apuntado en la bóveda ━━━")
	r := a.EstadoDelAgente().Dado
	for _, ap := range r {
		t.Logf("  %s · %s · %q · %s (%s)", ap.Cuando, ap.Quien, ap.Titulo, ap.Resultado, ap.Como)
	}
}

// idDe busca una entrada por su título en la bóveda que esté abierta.
func idDe(t *testing.T, a *App, titulo string) string {
	t.Helper()
	es, err := a.BuscarEnBoveda(titulo)
	if err != nil || len(es) == 0 {
		t.Fatalf("no está «%s» en la bóveda abierta: %v", titulo, err)
	}
	return es[0].ID
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
