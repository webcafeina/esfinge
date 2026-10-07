package app

// Lo que un agente de IA puede pedirle a esta aplicación (ADR 0054).
//
// Es el gemelo de `fuenteDelNavegador` y conviene leerlos al lado, porque **lo que
// cambia entre los dos es lo que decide este fichero entero**:
//
//   - Al navegador lo vigila **un dominio**: todo lo que toca la bóveda allí recibe el
//     origen de la pestaña, y una entrada que no sea de ese sitio no sale. Es lo que
//     impide que una página pida las cuentas de otra.
//   - **A un agente no lo vigila nada parecido**, porque no tiene pestaña. Lo que lo
//     vigila es **una persona**: el emparejamiento primero, y la aprobación por uso
//     cuando llegue la A3.
//
// Esa frase es la ADR 0054 entera, y de ella sale que esto sea un canal aparte con su
// socket, su testigo y sus frenos.
//
// **Y la regla que no se rompe, heredada del navegador**: nada de aquí llama a
// `Actividad()`. Si lo hiciera, un agente trabajando mantendría la bóveda abierta para
// siempre y el bloqueo por inactividad dejaría de significar lo que dice. Lo que sí
// cuenta es permitir un agente en la ventana, que es un clic de una persona.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/agente"
	"github.com/webcafeina/esfinge/internal/boveda"
)

// EventoAgentePide avisa a la ventana de que un agente quiere emparejarse.
const EventoAgentePide = "agente-pide"

func rutaAgentes() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "agentes.json")
}

// AgentePermitido es un agente al que se le dijo que sí.
//
// **Vive en su propio fichero y no en las preferencias**, por lo mismo que la lista de
// navegadores: las preferencias cruzan enteras al webview, y un testigo que abre la
// bóveda no tiene por qué pasearse por ahí.
type AgentePermitido struct {
	Testigo string `json:"testigo"`
	// Quien es cómo se llamó a sí mismo. **No se cree**: sirve para que la persona
	// sepa a quién le dijo que sí, no para decidir nada.
	Quien string `json:"quien"`
	Desde string `json:"desde"`
}

// agentesPermitidos lleva la lista y su fichero. Es el gemelo de
// `navegadoresPermitidos`, **duplicado a propósito**: retirar un navegador no puede
// retirar un agente, y la ventana tiene que poder decir cuál es cuál.
type agentesPermitidos struct {
	mu    sync.Mutex
	ruta  string
	lista []AgentePermitido
	// porEntregar es un testigo recién aprobado que todavía no ha recogido nadie.
	// **Se entrega una sola vez**, por lo mismo que el del navegador: si cualquiera
	// pudiera reclamarlo, bastaría pedir «emparejar» justo después de que la persona
	// aprobara el suyo.
	porEntregar string
	pendiente   string
}

func abrirAgentes(ruta string) *agentesPermitidos {
	g := &agentesPermitidos{ruta: ruta}
	datos, err := os.ReadFile(ruta)
	if err == nil {
		_ = json.Unmarshal(datos, &g.lista)
	}
	return g
}

func (g *agentesPermitidos) guardar() error {
	if g.ruta == "" {
		return nil
	}
	datos, err := json.MarshalIndent(g.lista, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(g.ruta), 0o700); err != nil {
		return err
	}
	return os.WriteFile(g.ruta, datos, 0o600)
}

func (g *agentesPermitidos) vale(testigo string) bool {
	if testigo == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, p := range g.lista {
		if p.Testigo == testigo {
			return true
		}
	}
	return false
}

func (g *agentesPermitidos) permitir(ahora time.Time) (AgentePermitido, error) {
	crudo := make([]byte, 32)
	if _, err := rand.Read(crudo); err != nil {
		return AgentePermitido{}, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	p := AgentePermitido{
		Testigo: hex.EncodeToString(crudo),
		Quien:   g.pendiente,
		Desde:   ahora.UTC().Format(time.RFC3339),
	}
	if p.Quien == "" {
		p.Quien = "Un agente"
	}
	g.lista = append(g.lista, p)
	g.porEntregar = p.Testigo
	g.pendiente = ""
	return p, g.guardar()
}

func (g *agentesPermitidos) recoger(quien string) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.porEntregar == "" {
		g.pendiente = quien
		return "", false
	}
	t := g.porEntregar
	g.porEntregar = ""
	return t, true
}

func (g *agentesPermitidos) quienPide() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.pendiente
}

func (g *agentesPermitidos) olvidar(testigo string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	var quedan []AgentePermitido
	for _, p := range g.lista {
		if p.Testigo != testigo {
			quedan = append(quedan, p)
		}
	}
	g.lista = quedan
	return g.guardar()
}

func (g *agentesPermitidos) ver() []AgentePermitido {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]AgentePermitido(nil), g.lista...)
}

// ---------------------------------------------------------------- la fuente

// fuenteDelAgente es lo que el canal de agentes puede pedirle a esta aplicación.
type fuenteDelAgente struct{ a *App }

func (f fuenteDelAgente) Estado() agente.Estado {
	e := f.a.EstadoBoveda()
	return agente.Estado{
		Existe:  e.Existe,
		Abierta: e.Abierta,
		// **El nombre de la bóveda activa sí va**, al revés que en el canal del
		// navegador: un agente que no sepa si está dentro de un proyecto puede guardar
		// la cuenta de un cliente en la bóveda de otro. Vacío es la personal.
		Boveda:      e.NombreDelProyecto,
		SoloLectura: e.SoloLectura,
	}
}

func (f fuenteDelAgente) Buscar(texto string) ([]agente.Entrada, int, error) {
	b := f.a.boveda()
	if b == nil {
		return nil, 0, boveda.ErrCerrada
	}
	// **Una sola pasada.** Lo evidente —buscar y pedir cada entrada con `Ver` para
	// mirar sus marcas— recorre la bóveda una vez por resultado, y eso es cuadrático:
	// con dos mil entradas, cuatro millones de comparaciones por búsqueda. Y el freno
	// deja hacer veinte búsquedas por minuto.
	todas := b.BuscarConMarcas(texto)
	out := make([]agente.Entrada, 0, len(todas))
	for _, c := range todas {
		if len(out) >= agente.TopeDeResultados {
			break
		}
		out = append(out, laEntrada(c))
	}
	return out, len(todas), nil
}

func (f fuenteDelAgente) Ver(id string) (agente.Entrada, error) {
	b := f.a.boveda()
	if b == nil {
		return agente.Entrada{}, boveda.ErrCerrada
	}
	entera, hay := b.Ver(id)
	// **Lo que está en la papelera no está**, aunque `Ver` lo encuentre: para el
	// agente, una entrada borrada es una entrada que no existe. Restaurarla es de la
	// ventana, que es donde se ve lo que se está recuperando.
	if !hay || entera.Papelera {
		return agente.Entrada{}, agente.ErrNoEsta
	}
	return laEntrada(boveda.ConMarcas{Entrada: entera.SinSecretos(), Marcas: entera.Marcas()}), nil
}

// laEntrada pasa una entrada de la bóveda a lo que el agente ve.
//
// **La que entra ya viene sin secretos** —`Buscar` las devuelve pasadas por
// `SinSecretos`—, y por eso las dos banderas se calculan mirando la entrada entera con
// `Ver`: `SinSecretos` vacía la semilla del código **sin dejar marca de que la
// hubiera**, así que leer `TOTP != ""` sobre lo que llega aquí daría siempre falso.
// Ya pasó una vez, al añadir `tieneCodigo` al canal del navegador: **todas las cuentas
// salían sin segundo factor** y lo cazó una prueba antes de publicar, no la vista.
func laEntrada(c boveda.ConMarcas) agente.Entrada {
	e := c.Entrada
	x := agente.Entrada{
		ID:        e.ID,
		Tipo:      string(e.Tipo),
		Titulo:    e.Titulo,
		Usuario:   e.Usuario,
		Sitios:    e.Sitios,
		Carpeta:   e.Carpeta,
		Etiquetas: e.Etiquetas,
		Cambiada:  e.Cambiada,
	}
	x.TieneSecreto = c.Marcas.TieneSecreto
	x.TieneCodigo = c.Marcas.TieneCodigo
	return x
}

func (f fuenteDelAgente) Higiene() (agente.Higiene, error) {
	b := f.a.boveda()
	if b == nil {
		return agente.Higiene{}, boveda.ErrCerrada
	}
	// **Lo calcula la bóveda, no esto.** Para saber qué contraseñas están reutilizadas
	// hay que compararlas, y comparándolas aquí la bóveda entera acabaría en claro en
	// el montón de este proceso. Lo que vuelve son identificadores.
	h := b.Higiene(time.Now())
	return agente.Higiene{
		Repetidas: h.Reutilizadas,
		SinCodigo: h.SinCodigo,
		Caducadas: h.Caducadas,
	}, nil
}

func (f fuenteDelAgente) Generar(bytes int, alfabeto string) (string, error) {
	if bytes <= 0 {
		bytes = 24
	}
	if alfabeto == "" {
		alfabeto = "hex"
	}
	return f.a.GenerarContrasena(bytes, alfabeto)
}

// Emparejar le pregunta a la persona, en la ventana. **No espera a nadie**: al otro
// lado hay un proceso que el cliente MCP puede matar en cualquier momento, así que se
// avisa, se contesta que todavía no, y el agente lo vuelve a pedir.
func (f fuenteDelAgente) Emparejar(quien string) (string, error) {
	if t, hay := f.a.agentes.recoger(quien); hay {
		return t, nil
	}
	f.a.sistema.Avisar(EventoAgentePide, quien)
	return "", errors.New("Permite este agente en la ventana de Esfinge")
}

func (f fuenteDelAgente) Emparejado(testigo string) bool { return f.a.agentes.vale(testigo) }

// ------------------------------------------------------- encender y apagar

// aplicarCanalDeAgentes abre o cierra la puerta según el ajuste.
//
// **Se mira en cada guardado y al arrancar**, como la del navegador: apagarla en
// Ajustes y que siguiera escuchando hasta el siguiente arranque sería un interruptor
// que miente.
func (a *App) aplicarCanalDeAgentes(p Preferencias) {
	a.mu.Lock()
	yaEsta := a.canalDeAgentes != nil
	a.mu.Unlock()

	if p.CanalDeAgentes == yaEsta {
		return
	}
	if !p.CanalDeAgentes {
		a.pararCanalDeAgentes()
		return
	}

	srv, err := agente.Servir(agente.RutaDelCanal(), fuenteDelAgente{a})
	fallo := ""
	if err != nil {
		fallo = err.Error()
	}
	a.mu.Lock()
	a.canalDeAgentes = srv
	a.agentesFallo = fallo
	a.mu.Unlock()
}

func (a *App) pararCanalDeAgentes() {
	a.mu.Lock()
	srv := a.canalDeAgentes
	a.canalDeAgentes = nil
	a.agentesFallo = ""
	a.mu.Unlock()
	if srv != nil {
		_ = srv.Parar()
	}
}

// ------------------------------------------------------- lo que ve la ventana

// EstadoDelAgente es lo que Ajustes enseña del canal de agentes.
type EstadoDelAgente struct {
	Encendido  bool   `json:"encendido"`
	Escuchando bool   `json:"escuchando"`
	Donde      string `json:"donde"`
	Error      string `json:"error,omitempty"`
	// Pide es quién está esperando permiso, vacío si no hay nadie.
	Pide string `json:"pide,omitempty"`
	// Permitidos son los agentes a los que se les dijo que sí. **Sin el testigo**.
	Permitidos []AgentePermitido `json:"permitidos"`
	// Configuracion es el bloque que hay que pegarle al cliente MCP.
	Configuracion string `json:"configuracion"`
}

// EstadoDelAgente dice cómo está la puerta de los agentes.
func (a *App) EstadoDelAgente() EstadoDelAgente {
	a.mu.Lock()
	srv := a.canalDeAgentes
	fallo := a.agentesFallo
	a.mu.Unlock()

	e := EstadoDelAgente{
		Encendido:     a.ajustes.Ver().CanalDeAgentes,
		Escuchando:    srv != nil,
		Error:         fallo,
		Pide:          a.agentes.quienPide(),
		Permitidos:    a.agentes.ver(),
		Configuracion: ConfiguracionParaElCliente(),
	}
	if srv != nil {
		e.Donde = srv.Donde()
	}
	// **El testigo no cruza el puente.** Es lo que abre la bóveda, y dentro del
	// webview no pinta nada: la ventana identifica a cada agente por su fecha, que es
	// lo mismo que hace con los navegadores.
	for i := range e.Permitidos {
		e.Permitidos[i].Testigo = ""
	}
	return e
}

// PermitirAgente le dice que sí al que está esperando.
func (a *App) PermitirAgente() error {
	_, err := a.agentes.permitir(time.Now())
	// **Esto sí cuenta como actividad**: es un clic de una persona, al revés que todo
	// lo que pide el agente.
	a.Actividad()
	return err
}

// OlvidarAgente retira a uno, por su fecha. **Por la fecha y no por el testigo**,
// porque el testigo no cruza el puente.
func (a *App) OlvidarAgente(desde string) error {
	for _, p := range a.agentes.ver() {
		if p.Desde == desde {
			a.Actividad()
			return a.agentes.olvidar(p.Testigo)
		}
	}
	return errors.New("Ese agente ya no está en la lista")
}

// ConfiguracionParaElCliente es el bloque que hay que pegar en el cliente MCP.
//
// **Se enseña, no se escribe.** El native messaging obliga a escribir manifiestos en
// carpetas ajenas porque no hay otra forma de que el navegador sepa que existimos;
// aquí sí la hay, y los clientes MCP son muchos y cambian. Escribir en el fichero de
// configuración de otro programa es algo que hay que hacer solo cuando no queda más
// remedio.
//
// **La ruta es absoluta y eso no es un detalle**: Claude Desktop arranca los servidores
// desde un directorio indefinido, así que una ruta relativa no encuentra nada.
func ConfiguracionParaElCliente() string {
	ruta, err := rutaDelServidorMCP()
	if err != nil || ruta == "" {
		ruta = "esfinge-mcp"
	}
	b, err := json.MarshalIndent(map[string]any{
		"mcpServers": map[string]any{
			"esfinge": map[string]any{"command": ruta},
		},
	}, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

// rutaDelServidorMCP es dónde está el binario, al lado del que se esté ejecutando.
func rutaDelServidorMCP() (string, error) {
	yo, err := os.Executable()
	if err != nil {
		return "", err
	}
	yo, err = filepath.EvalSymlinks(yo)
	if err != nil {
		return "", err
	}
	nombre := "esfinge-mcp"
	if runtime.GOOS == "windows" {
		nombre += ".exe"
	}
	return filepath.Join(filepath.Dir(yo), nombre), nil
}
