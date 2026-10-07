package agente

// MCP por entrada y salida estándar, escrito a mano (ADR 0054).
//
// # Por qué a mano
//
// Lo que hace falta son cinco métodos de JSON-RPC 2.0 —`initialize`,
// `notifications/initialized`, `ping`, `tools/list` y `tools/call`— y es menos código
// que el CBOR de las llaves de acceso, que también está escrito aquí. Pesa además algo
// que no es gusto: **sería la primera dependencia dentro de la frontera de seguridad**,
// en el proceso que recibe lo que pide un agente, y con un árbol transitivo que aquí no
// audita nadie en cada actualización.
//
// Lo que cuesta, dicho antes de pagarlo: hay que seguir las revisiones de la
// especificación a mano. Lo amortigua `docs/mcp.md`, que dice a qué revisión se apunta,
// y una prueba de bytes para que un cambio sea un diff y no un misterio.
//
// # El encuadre
//
// **Un mensaje por línea**, no las cabeceras `Content-Length` de LSP —eso es otra cosa
// y MCP lo quitó—. Así que lo que se escribe no puede llevar saltos de línea dentro:
// el codificador va sin sangrar, siempre.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RevisionDeMCP es la de la especificación a la que esto apunta. Si un cliente pide
// otra, se le contesta ésta: es lo que dice la propia especificación que hay que hacer.
const RevisionDeMCP = "2025-06-18"

// NombreDelServidor y VersionDelServidor son lo que se dice de uno mismo en el saludo.
const NombreDelServidor = "esfinge"

type mensaje struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Metodo  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type respuestaRPC struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *errorRPC       `json:"error,omitempty"`
}

type errorRPC struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Contenido es un trozo de lo que una herramienta devuelve.
//
// **El texto tiene que bastarse solo**: muchos clientes ignoran lo estructurado, y es
// el texto el que acaba en la transcripción. De ahí la regla que gobierna todas las
// herramientas: **el texto no puede llevar un secreto**, con la única excepción del
// código de un solo uso, que es justamente el punto.
type Contenido struct {
	Tipo  string `json:"type"`
	Texto string `json:"text"`
}

// Resultado es lo que contesta `tools/call`.
type Resultado struct {
	Contenido []Contenido `json:"content"`
	// EsError marca un fallo **que el modelo tiene que leer y explicar**, no un error
	// de protocolo: la convención de MCP es que lo primero va como resultado y lo
	// segundo como error de JSON-RPC. «Esfinge no está abierta» es de los primeros.
	EsError bool `json:"isError,omitempty"`
}

func texto(s string) Resultado {
	return Resultado{Contenido: []Contenido{{Tipo: "text", Texto: s}}}
}

func fallo(s string) Resultado {
	return Resultado{Contenido: []Contenido{{Tipo: "text", Texto: s}}, EsError: true}
}

// Llamar es lo que atiende una herramienta. Lo pone quien monta el servidor MCP: el
// binario lo apunta al socket, y las pruebas a donde quieran.
type Llamar func(h Herramienta, args map[string]any) Resultado

// Hablar atiende MCP sobre esas dos tuberías hasta que la entrada se acaba.
//
// **`initialize` y `tools/list` se contestan aquí**, sin preguntarle a nadie, y es una
// restricción y no una comodidad: Claude Code pide la lista al abrir la sesión y se la
// queda, así que contestar una lista vacía porque Esfinge no esté abierta deja al
// agente sin herramientas **toda la sesión**.
func Hablar(entra io.Reader, sale io.Writer, version string, llamar Llamar) error {
	lector := bufio.NewScanner(entra)
	// Un mensaje puede ser largo —un `tools/call` con argumentos— pero no enorme.
	lector.Buffer(make([]byte, 0, 64*1024), 1<<20)
	enc := json.NewEncoder(sale)

	for lector.Scan() {
		linea := strings.TrimSpace(lector.Text())
		if linea == "" {
			continue
		}
		var m mensaje
		if err := json.Unmarshal([]byte(linea), &m); err != nil {
			continue // lo que no es JSON no se contesta: no hay a qué identificador contestar
		}
		// **Sin identificador es una notificación**, y a una notificación no se
		// contesta nunca. Mandar algo rompe a clientes estrictos.
		if len(m.ID) == 0 {
			continue
		}
		r := respuestaRPC{JSONRPC: "2.0", ID: m.ID}
		switch m.Metodo {
		case "initialize":
			r.Result = map[string]any{
				"protocolVersion": RevisionDeMCP,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": NombreDelServidor, "version": version},
			}
		case "ping":
			r.Result = map[string]any{}
		case "tools/list":
			r.Result = map[string]any{"tools": LasHerramientas}
		case "tools/call":
			r.Result = atenderLlamada(m.Params, llamar)
		default:
			r.Error = &errorRPC{Code: -32601, Message: fmt.Sprintf("Esfinge no entiende %q", m.Metodo)}
		}
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return lector.Err()
}

func atenderLlamada(params json.RawMessage, llamar Llamar) Resultado {
	var p struct {
		Nombre string         `json:"name"`
		Args   map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return fallo("Esfinge no ha entendido los argumentos de esa herramienta")
	}
	h, hay := PorNombre(p.Nombre)
	if !hay {
		// Como resultado y no como error de protocolo: es algo que el modelo tiene que
		// leer y contarle a quien le habla.
		return fallo(fmt.Sprintf("Esfinge no tiene ninguna herramienta llamada %q", p.Nombre))
	}
	return llamar(h, p.Args)
}
