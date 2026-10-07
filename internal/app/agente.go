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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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
		Reutilizadas: h.Reutilizadas,
		SinCodigo:    h.SinCodigo,
		Caducadas:    h.Caducadas,
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
	// Quiere es lo que un agente está pidiendo y hay que contestar, nulo si nada.
	//
	// **Lleva el título de la entrada**, y tiene que llevarlo: «un agente quiere una
	// contraseña» no es una pregunta que se pueda contestar. Lo que no lleva es la
	// contraseña, claro.
	Quiere *loQuePideUnAgente `json:"quiere,omitempty"`
	// Valvula es cómo va el «todo lo de este agente durante un rato», si está abierta.
	Valvula comoVaLaValvula `json:"valvula"`
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
		Quiere:        a.permisos.loPendiente(),
		Valvula:       a.permisos.comoVa(time.Now()),
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
			// Retirar a un agente **cierra también lo que tuviera concedido**: si no,
			// seguiría pudiendo durante lo que quedara de válvula, que es justo lo que
			// quien pulsa «Retirar» está intentando que no pase.
			a.permisos.cerrarLaValvula()
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

// ---------------------------------------------- usar un secreto, con permiso

// PlazoDelPermiso es lo que dura un sí sin recoger.
//
// Corto a propósito: un permiso que se queda flotando es un permiso que alguien usa
// media hora después de que la persona lo diera pensando en otra cosa.
const PlazoDelPermiso = 2 * time.Minute

// EventoAgenteQuiere avisa a la ventana de que un agente pide algo que hay que aprobar.
const EventoAgenteQuiere = "agente-quiere"

// loQuePideUnAgente es una petición esperando un sí.
//
// **El sí va atado a la entrada**, no al agente: aprobar «la contraseña de GitHub» no
// puede servir para que el siguiente intento se lleve la del banco. Es la misma idea que
// el testigo de emparejamiento, que se entrega una sola vez y para lo que se pidió.
type loQuePideUnAgente struct {
	Quien  string `json:"quien"`
	Que    string `json:"que"`
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	Cuando string `json:"cuando"`
}

// PlazoDeLaValvula es lo que dura un «todo lo de este agente durante un rato».
//
// **Cinco minutos desde el clic, no desde el último uso.** Un plazo que se renueva con
// el uso es una válvula permanente para un agente ocupado, que es justo lo contrario de
// lo que se está concediendo.
const PlazoDeLaValvula = 5 * time.Minute

// TopeDeLaValvula es cuántas veces puede usarla antes de que se cierre sola.
//
// **Sin un tope, «cinco minutos» es un cheque en blanco**: un agente en un bucle puede
// pedir cientos de veces en ese rato. Al llegar aquí la válvula se cierra y la
// siguiente petición vuelve a preguntar, que es lo que devuelve a la persona al bucle —
// y lo que un agente desbocado necesita que pase.
const TopeDeLaValvula = 20

type permisosDelAgente struct {
	mu sync.Mutex
	// pendiente es lo último que se ha pedido y nadie ha contestado.
	pendiente *loQuePideUnAgente
	// concedidoPara es la entrada para la que hay un sí, y hasta cuándo vale.
	concedidoPara string
	hasta         time.Time

	// valvulaHasta es hasta cuándo vale el «todo lo de este agente», y usadas cuántas
	// veces se ha usado ya. Ver [PlazoDeLaValvula].
	valvulaHasta time.Time
	usadas       int
	// loUltimo son los títulos de lo que se le ha ido dando, para que la ventana pueda
	// enseñarlo mientras pasa. **Sin secretos**, como todo lo demás.
	loUltimo []string
}

// pedir deja apuntado lo que se quiere y dice si ya había permiso **para eso mismo**.
//
// Si lo había, **se gasta**: un sí vale para una vez. Si el agente quiere dos, pregunta
// dos veces, que es exactamente lo que el cliente eligió.
// pedir dice si se puede hacer ya, y si no, lo deja pedido.
//
// `porValvula` dice si este verbo puede pasar por ella: **lo que se le enseña al agente
// no pasa nunca**, y ésa es la frontera entera de la válvula. Ver [conceder].
func (p *permisosDelAgente) pedir(q loQuePideUnAgente, porValvula bool, ahora time.Time) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Un sí para esta entrada, que **se gasta**: un sí vale para una vez.
	if p.concedidoPara != "" && p.concedidoPara == q.ID && ahora.Before(p.hasta) {
		p.concedidoPara = ""
		return true, boveda.ApuntePreguntado
	}
	// O la válvula, si está abierta y si este verbo puede pasar por ella.
	if porValvula && ahora.Before(p.valvulaHasta) {
		p.usadas++
		p.loUltimo = append(p.loUltimo, q.Titulo)
		// **Y al llegar al tope se cierra**, en el mismo momento en que se usa la
		// última: la siguiente vuelve a preguntar.
		if p.usadas >= TopeDeLaValvula {
			p.valvulaHasta = time.Time{}
		}
		return true, boveda.ApunteValvula
	}
	p.pendiente = &q
	return false, ""
}

// cerrarLaValvula la cierra ya, y olvida la cuenta.
//
// La llaman el botón de cortar **y todo lo que cambia el suelo bajo los pies**: cerrar
// la bóveda, el bloqueo por inactividad, cambiar de bóveda y retirar al agente. Un
// permiso dado para «esta bóveda, ahora» no puede sobrevivir a ninguna de esas cosas.
func (p *permisosDelAgente) cerrarLaValvula() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.valvulaHasta = time.Time{}
	p.concedidoPara = ""
	p.usadas = 0
	p.loUltimo = nil
}

// conceder dice que sí a lo que estuviera pendiente, y devuelve para qué era.
// conceder dice que sí a lo que estuviera pendiente. Con `unRato`, además abre la
// válvula.
func (p *permisosDelAgente) conceder(unRato bool, ahora time.Time) (loQuePideUnAgente, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pendiente == nil {
		return loQuePideUnAgente{}, false
	}
	q := *p.pendiente
	p.pendiente = nil
	if unRato {
		// **Y entonces no se pone el permiso de una vez**, que es lo que parecía
		// inofensivo y rompía la válvula entera: con los dos puestos, la petición
		// siguiente consumía el individual, la válvula no se usaba nunca, su contador
		// se quedaba a cero y **el tope no llegaba jamás**. Lo dijo la prueba del tope,
		// no la lectura.
		p.valvulaHasta = ahora.Add(PlazoDeLaValvula)
		p.usadas = 0
		p.loUltimo = nil
		return q, true
	}
	p.concedidoPara = q.ID
	p.hasta = ahora.Add(PlazoDelPermiso)
	return q, true
}

// comoVaLaValvula es lo que la ventana enseña mientras está abierta.
type comoVaLaValvula struct {
	Abierta bool     `json:"abierta"`
	Quedan  int      `json:"quedan"`
	Usadas  int      `json:"usadas"`
	Tope    int      `json:"tope"`
	Ultimos []string `json:"ultimos,omitempty"`
}

func (p *permisosDelAgente) comoVa(ahora time.Time) comoVaLaValvula {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !ahora.Before(p.valvulaHasta) {
		return comoVaLaValvula{}
	}
	return comoVaLaValvula{
		Abierta: true,
		Quedan:  int(p.valvulaHasta.Sub(ahora).Seconds()),
		Usadas:  p.usadas,
		Tope:    TopeDeLaValvula,
		Ultimos: append([]string(nil), p.loUltimo...),
	}
}

// denegar lo quita sin conceder nada.
func (p *permisosDelAgente) denegar() (loQuePideUnAgente, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pendiente == nil {
		return loQuePideUnAgente{}, false
	}
	q := *p.pendiente
	p.pendiente = nil
	return q, true
}

func (p *permisosDelAgente) loPendiente() *loQuePideUnAgente {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pendiente == nil {
		return nil
	}
	q := *p.pendiente
	return &q
}

// CopiarSecreto pone la contraseña en el portapapeles **si hay un sí para esa entrada**.
//
// **No espera a nadie**, y eso es lo que decide cómo se siente esto: al otro lado hay un
// proceso al que su cliente puede cortar en cualquier momento, y una ventana que puede
// estar detrás de todo. Bloquearse treinta segundos esperando un clic sería agotar el
// plazo del cliente MCP las más de las veces. Es la misma forma que `Emparejar`, con su
// porqué escrito desde la ADR 0027.
//
// Así que: se pide, se avisa a la ventana, y **quien llama vuelve a intentarlo**. Lo
// dice la descripción de la herramienta, para que el modelo sepa que tiene que hacerlo.
func (f fuenteDelAgente) CopiarSecreto(quien, id string) (agente.Copiado, error) {
	b := f.a.boveda()
	if b == nil {
		return agente.Copiado{}, boveda.ErrCerrada
	}
	e, hay := b.Ver(id)
	if !hay || e.Papelera {
		return agente.Copiado{}, agente.ErrNoEsta
	}
	if e.Secreto == "" {
		return agente.Copiado{}, errors.New("Esa entrada no tiene contraseña")
	}
	if quien == "" {
		quien = "Un agente"
	}

	ahora := time.Now()
	q := loQuePideUnAgente{
		Quien: quien, Que: agente.QueCopiarSecreto, ID: id, Titulo: e.Titulo,
		Cuando: ahora.UTC().Format(time.RFC3339),
	}
	// **Copiar puede pasar por la válvula**, y ésa es exactamente la línea: lo que la
	// válvula cubre es **actuar** —el secreto se queda en este equipo y el portapapeles
	// se borra solo—, no enseñar. Lo que se le enseña al agente pregunta siempre.
	vale, como := f.a.permisos.pedir(q, true, ahora)
	if !vale {
		// **La ventana no se trae al frente** (ADR 0027): hacerlo dejaría que
		// cualquier programa de esta máquina hiciera aparecer la ventana de un gestor
		// de contraseñas cuando quisiera.
		f.a.sistema.Avisar(EventoAgenteQuiere, q)
		return agente.Copiado{}, agente.ErrPideAprobacion
	}

	// **Copia Esfinge, y no cuenta como actividad**: lo pide un programa, no una
	// persona. Lo que sí contó fue el clic de aprobarlo.
	segundos, err := f.a.copiar(e.Secreto, false)
	if err != nil {
		f.a.apuntarComo(b, q, boveda.ApunteNegado, como)
		return agente.Copiado{}, err
	}
	f.a.apuntarComo(b, q, boveda.ApunteHecho, como)
	return agente.Copiado{Portapapeles: segundos, Titulo: e.Titulo}, nil
}

// apuntar deja constancia en el registro de la bóveda (ADR 0054).
//
// **Que falle no deshace lo hecho**, así que no se devuelve el error: se registra. Y en
// una bóveda que no se puede escribir —de solo lectura, o compartida de solo ver— no hay
// dónde apuntar, y eso **la ventana lo dice**: una medida que no se aplica en silencio es
// peor que no tenerla.
func (a *App) apuntar(b *boveda.Boveda, q loQuePideUnAgente, resultado string) {
	a.apuntarComo(b, q, resultado, boveda.ApuntePreguntado)
}

// apuntarComo es lo mismo diciendo **cómo se autorizó**: preguntando una por una o por
// la válvula. Esa distinción es lo que deja leer el registro después y saber qué se dio
// mirándolo y qué se dio en bloque.
func (a *App) apuntarComo(b *boveda.Boveda, q loQuePideUnAgente, resultado, como string) {
	err := b.Apuntar(boveda.Apunte{
		Quien: q.Quien, Que: q.Que, Sobre: q.ID, Titulo: q.Titulo,
		Resultado: resultado, Como: como,
	})
	if err != nil {
		log.Printf("esfinge: no se ha podido apuntar lo que se le dio a un agente: %v", err)
	}
}

// AprobarLoQuePideElAgente es el «sí» de la persona.
func (a *App) AprobarLoQuePideElAgente(unRato bool) error {
	if _, hay := a.permisos.conceder(unRato, time.Now()); !hay {
		return errors.New("Ya no hay nada que aprobar")
	}
	// **Esto sí cuenta como actividad**: es un clic de una persona.
	a.Actividad()
	return nil
}

// DenegarLoQuePideElAgente es el «no», y **queda apuntado**: «pidió la contraseña del
// banco y se le dijo que no» es la señal por la que el registro existe.
func (a *App) DenegarLoQuePideElAgente() error {
	q, hay := a.permisos.denegar()
	if !hay {
		return errors.New("Ya no hay nada que denegar")
	}
	if b := a.boveda(); b != nil {
		a.apuntar(b, q, boveda.ApunteNegado)
	}
	a.Actividad()
	return nil
}

// RegistroDelAgente es lo que se le ha dado, para enseñarlo en la ventana.
func (a *App) RegistroDelAgente() ([]boveda.Apunte, error) {
	b := a.boveda()
	if b == nil {
		return nil, boveda.ErrCerrada
	}
	return b.Registro(), nil
}

// CortarAlAgente cierra la válvula en el acto.
//
// Es el botón que hace que «durante cinco minutos» sea soportable: lo que se concede se
// puede retirar sin esperar a que caduque.
func (a *App) CortarAlAgente() error {
	a.permisos.cerrarLaValvula()
	a.Actividad()
	return nil
}

// Codigo devuelve el de un solo uso, **y es lo único que el agente llega a ver**.
//
// Pide un sí **cada vez, sin excepción**: la válvula no lo cubre, y eso no es una
// omisión sino la línea entera. El trato de la válvula es «actúa por mí durante cinco
// minutos», y actuar es reversible —el portapapeles se borra solo, una copia es un
// pegado—. **Enseñar no lo es**: en cuanto esas seis cifras entran en el contexto del
// modelo están en su transcripción, en su disco y camino de un servidor. Lo que puede
// salir en ráfaga es lo que de todos modos no sale de este equipo.
func (f fuenteDelAgente) Codigo(quien, id string) (agente.Codigo, error) {
	b := f.a.boveda()
	if b == nil {
		return agente.Codigo{}, boveda.ErrCerrada
	}
	e, hay := b.Ver(id)
	if !hay || e.Papelera {
		return agente.Codigo{}, agente.ErrNoEsta
	}
	if e.TOTP == "" {
		return agente.Codigo{}, errors.New("Esa entrada no tiene código de un solo uso")
	}
	if quien == "" {
		quien = "Un agente"
	}

	ahora := time.Now()
	q := loQuePideUnAgente{
		Quien: quien, Que: agente.QueCodigo, ID: id, Titulo: e.Titulo,
		Cuando: ahora.UTC().Format(time.RFC3339),
	}
	// **`false`**: por la válvula no pasa. Es el único sitio donde se dice, y es lo que
	// hace que la regla sea una regla y no una excepción.
	vale, _ := f.a.permisos.pedir(q, false, ahora)
	if !vale {
		f.a.sistema.Avisar(EventoAgenteQuiere, q)
		return agente.Codigo{}, agente.ErrPideAprobacion
	}

	// Se calcula con lo que ya existe, que además es lo que la ventana enseña: así el
	// agente y la pantalla no pueden dar códigos distintos.
	c, err := f.a.CodigoDeBoveda(id)
	if err != nil {
		f.a.apuntarComo(b, q, boveda.ApunteNegado, boveda.ApuntePreguntado)
		return agente.Codigo{}, err
	}
	f.a.apuntarComo(b, q, boveda.ApunteHecho, boveda.ApuntePreguntado)
	return agente.Codigo{Codigo: c.Codigo, Quedan: c.Quedan}, nil
}

// ------------------------------------------------------------- escribir

// Crear guarda una entrada nueva. **Va directo**, como las escrituras del navegador
// (ADR 0032): lo escrito se puede deshacer —la papelera guarda treinta días— y pedir un
// sí por cada una convertiría «ordéname la bóveda» en cuarenta diálogos, que es como se
// aprende a aprobar sin leer.
func (f fuenteDelAgente) Crear(quien string, p agente.Peticion) (agente.Escrito, error) {
	b, err := f.a.bovedaParaEscribirAgente()
	if err != nil {
		return agente.Escrito{}, err
	}
	titulo := p.Campos["titulo"]
	if strings.TrimSpace(titulo) == "" {
		return agente.Escrito{}, errors.New("Hace falta un título")
	}
	tipo := boveda.Tipo(p.Campos["tipo"])
	if tipo == "" {
		tipo = boveda.TipoCredencial
	}
	// **Una llave de acceso no se crea a mano** (ADR 0048): la emite el sitio, y lo que
	// se guardaría aquí no abriría ninguna cuenta.
	if tipo == boveda.TipoLlave {
		return agente.Escrito{}, fmt.Errorf("%w: una llave de acceso la emite el sitio, no se escribe a mano", agente.ErrNoSeEscribe)
	}

	e := boveda.Entrada{
		Tipo: tipo, Titulo: titulo,
		Usuario: p.Campos["usuario"], Notas: p.Campos["notas"],
		Carpeta: p.Campos["carpeta"], Sitios: p.Sitios, Etiquetas: p.Etiquetas,
	}
	e.Secreto = p.Campos["secreto"]
	if p.Generar {
		// **La hace Esfinge y no vuelve.** El agente crea una cuenta con una
		// contraseña que no ha visto, que es estrictamente mejor que una que se
		// invente él — y eso lo dice la descripción de la herramienta, para que la
		// prefiera.
		clave, err := f.a.GenerarContrasena(24, "hex")
		if err != nil {
			return agente.Escrito{}, err
		}
		e.Secreto = clave
	}
	if err := f.a.GuardarEnBoveda(e); err != nil {
		return agente.Escrito{}, err
	}
	// El identificador lo pone la bóveda al guardar, así que se busca por el título.
	id := ""
	for _, x := range b.Buscar(titulo) {
		if x.Titulo == titulo {
			id = x.ID
		}
	}
	f.a.apuntarComo(b, loQuePideUnAgente{Quien: siNoDice(quien), Que: agente.QueCrear, ID: id, Titulo: titulo},
		boveda.ApunteHecho, boveda.ApunteDirecto)
	return agente.Escrito{ID: id, Titulo: titulo}, nil
}

// losSecretos son los campos cuyo cambio **tiene que aprobar una persona**.
//
// Las escrituras del navegador están acotadas por el sitio de la pestaña: solo puede
// guardar una credencial para el sitio donde está quien navega. **Un agente no tiene
// sitio**, así que sin esta puerta podría reescribir la contraseña del banco sin que
// nadie preguntara — y aunque la anterior quede en el historial, nadie se habría
// enterado.
var losSecretos = map[string]bool{"secreto": true, "totp": true}

// Editar cambia campos **sobre la entrada que ya está**, nunca construyendo una nueva.
//
// Eso no es estilo: construirla se lleva por delante `Extra` —los campos que escribió
// una versión más nueva de Esfinge— y todo lo que no se nombre. Es la trampa que
// `ActualizarCuenta` esquiva desde la fase 2.
func (f fuenteDelAgente) Editar(quien string, p agente.Peticion) (agente.Escrito, error) {
	b, err := f.a.bovedaParaEscribirAgente()
	if err != nil {
		return agente.Escrito{}, err
	}
	e, hay := b.Ver(p.ID)
	if !hay || e.Papelera {
		return agente.Escrito{}, agente.ErrNoEsta
	}

	tocaUnSecreto := false
	var cambiados []string
	for campo := range p.Campos {
		if losSecretos[campo] {
			tocaUnSecreto = true
		}
		cambiados = append(cambiados, campo)
	}
	sort.Strings(cambiados)

	if tocaUnSecreto {
		ahora := time.Now()
		q := loQuePideUnAgente{
			Quien: siNoDice(quien), Que: agente.QueEditar, ID: p.ID, Titulo: e.Titulo,
			Cuando: ahora.UTC().Format(time.RFC3339),
		}
		// **Y la válvula no lo cubre**, por lo mismo que el código: dejar una cuenta
		// sin forma de entrar no es reversible de la forma en que lo es una copia.
		vale, _ := f.a.permisos.pedir(q, false, ahora)
		if !vale {
			f.a.sistema.Avisar(EventoAgenteQuiere, q)
			return agente.Escrito{}, agente.ErrPideAprobacion
		}
	}

	// **El parche, campo a campo sobre lo que ya hay.**
	for campo, valor := range p.Campos {
		switch campo {
		case "titulo":
			e.Titulo = valor
		case "usuario":
			e.Usuario = valor
		case "notas":
			e.Notas = valor
		case "carpeta":
			e.Carpeta = valor
		case "secreto":
			// Por el camino de siempre, para que la anterior caiga al historial.
			e.CambiarSecreto(valor, time.Now())
		case "totp":
			e.TOTP = valor
		}
	}
	if p.Sitios != nil {
		e.Sitios = p.Sitios
	}
	if p.Etiquetas != nil {
		e.Etiquetas = p.Etiquetas
	}
	if err := f.a.GuardarEnBoveda(e); err != nil {
		return agente.Escrito{}, err
	}
	como := boveda.ApunteDirecto
	if tocaUnSecreto {
		como = boveda.ApuntePreguntado
	}
	f.a.apuntarComo(b, loQuePideUnAgente{Quien: siNoDice(quien), Que: agente.QueEditar, ID: p.ID, Titulo: e.Titulo},
		boveda.ApunteHecho, como)
	return agente.Escrito{ID: p.ID, Titulo: e.Titulo, Cambiados: cambiados}, nil
}

// Borrar la manda a la papelera, **con un sí**. Es lo único que quita algo de la vista,
// y la pregunta **dice que son treinta días**: sin eso se estaría contestando algo más
// grave de lo que pasa.
func (f fuenteDelAgente) Borrar(quien, id string) (agente.Escrito, error) {
	b, err := f.a.bovedaParaEscribirAgente()
	if err != nil {
		return agente.Escrito{}, err
	}
	e, hay := b.Ver(id)
	if !hay || e.Papelera {
		return agente.Escrito{}, agente.ErrNoEsta
	}

	ahora := time.Now()
	q := loQuePideUnAgente{
		Quien: siNoDice(quien), Que: agente.QueBorrar, ID: id, Titulo: e.Titulo,
		Cuando: ahora.UTC().Format(time.RFC3339),
	}
	vale, como := f.a.permisos.pedir(q, true, ahora)
	if !vale {
		f.a.sistema.Avisar(EventoAgenteQuiere, q)
		return agente.Escrito{}, agente.ErrPideAprobacion
	}
	if err := f.a.BorrarDeBoveda(id); err != nil {
		f.a.apuntarComo(b, q, boveda.ApunteNegado, como)
		return agente.Escrito{}, err
	}
	f.a.apuntarComo(b, q, boveda.ApunteHecho, como)
	return agente.Escrito{ID: id, Titulo: e.Titulo, ALaPapelera: true, Dias: 30}, nil
}

// bovedaParaEscribir dice si aquí se puede escribir, **con el porqué**: una bóveda de
// una versión más nueva o una compartida de solo ver (ADR 0052).
func (a *App) bovedaParaEscribirAgente() (*boveda.Boveda, error) {
	b := a.boveda()
	if b == nil {
		return nil, boveda.ErrCerrada
	}
	// **La misma razón que para el navegador**, y por eso se pide la suya en vez de
	// escribir otra: si un día se añade un caso, se añade para los dos.
	if porque := (fuenteDelNavegador{a}).porQueNoSeEscribe(b); porque != "" {
		return nil, fmt.Errorf("%w: %s", agente.ErrNoSeEscribe, porque)
	}
	return b, nil
}

func siNoDice(quien string) string {
	if quien == "" {
		return "Un agente"
	}
	return quien
}
