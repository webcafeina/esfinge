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
	"io"
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
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/escritura"
	"github.com/webcafeina/esfinge/internal/mcpb"
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
		// la cuenta de un cliente en la bóveda de otro.
		//
		// Y va **con su bandera**, porque antes la personal se decía omitiendo el
		// campo y eso no lo dice: un agente ve que falta, no que sea la de siempre.
		Boveda:       comoSeLlamaLaBoveda(e.NombreDelProyecto),
		EsLaPersonal: e.NombreDelProyecto == "",
		SoloLectura:  e.SoloLectura,
		// **La de la Esfinge que contesta**, que no es la que enseña el cliente MCP en
		// su lista de extensiones: ésa es la del paquete, y el paquete no se actualiza.
		Version: f.a.version,
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

// Resumen son los números de la bóveda, **sin un solo nombre** (2026-10-08).
//
// Nació de verlo usar: a «¿cuántas cuentas tengo?» un agente que solo tiene `buscar`
// contesta trayéndose las primeras entradas **con sus títulos y sus usuarios** y
// deduciendo el resto. Esto la contesta con números, que es a la vez la respuesta más
// útil y la que menos cuenta de lo que hay dentro.
func (f fuenteDelAgente) Resumen() (agente.Recuento, error) {
	b := f.a.boveda()
	if b == nil {
		return agente.Recuento{}, boveda.ErrCerrada
	}
	r := b.Recontar()
	return agente.Recuento{
		Total: r.Total, Papelera: r.Papelera, PorClase: r.PorClase,
		ConCodigo: r.ConCodigo, SinCodigo: r.SinCodigo,
	}, nil
}

func (f fuenteDelAgente) Generar(caracteres, bytes int, alfabeto string) (string, error) {
	return f.a.generarParaElAgente(caracteres, bytes, alfabeto)
}

// generarParaElAgente hace la contraseña que pide el canal, **y es el único sitio donde
// se decide cómo**.
//
// `caracteres` manda sobre `bytes` cuando se piden los dos, porque un largo en caracteres
// es lo que limita un formulario y por eso es lo que alguien pide; los bytes son la
// entropía y están para quien piense en ella. Sin ninguno de los dos, lo de siempre.
//
// **Estaba repetido tres veces** —`crear`, `editar` y `generar` tenían cada uno su `if`—
// y añadir el largo habría hecho cuatro. La misma regla en N sitios se desalinea sola, y
// aquí ya ha costado tres fallos.
func (a *App) generarParaElAgente(caracteres, bytes int, alfabeto string) (string, error) {
	if alfabeto == "" {
		alfabeto = alfabetoDeLoQueGenera
	}
	al, hay := cripto.Alfabetos[alfabeto]
	if !hay {
		return "", fmt.Errorf("El alfabeto «%s» no existe", alfabeto)
	}
	if caracteres > 0 {
		return cripto.GenerarDeLargo(al, caracteres)
	}
	if bytes <= 0 {
		bytes = largoDeLoQueGenera
	}
	return a.GenerarContrasena(bytes, alfabeto)
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
	// Configuracion es el bloque que hay que pegarle a un cliente que se configure con
	// un fichero. **Es lo tosco**, y por eso en la pantalla va plegado.
	Configuracion string `json:"configuracion"`
	// Orden es lo mismo para Claude Code, que se configura con una orden y no con un
	// fichero: una línea que se copia y se pega en una terminal.
	Orden string `json:"orden"`
	// VSCode es el mismo bloque **con otra clave**, porque VS Code no lee `mcpServers`.
	// Ver `ConfiguracionParaVSCode`.
	VSCode string `json:"vscode"`
	// Ruta es dónde está el servidor MCP, suelta, para quien tenga que escribirla a
	// mano en un cliente que no sea ninguno de los de la lista.
	Ruta string `json:"ruta"`
	// Quiere es lo que un agente está pidiendo y hay que contestar, nulo si nada.
	//
	// **Lleva el título de la entrada**, y tiene que llevarlo: «un agente quiere una
	// contraseña» no es una pregunta que se pueda contestar. Lo que no lleva es la
	// contraseña, claro.
	Quiere *loQuePideUnAgente `json:"quiere,omitempty"`
	// Valvula es cómo va el «todo lo de este agente durante un rato», si está abierta.
	Valvula comoVaLaValvula `json:"valvula"`
	// Siempre dice si está puesto el permiso que no caduca, y SiempreQuien a quién se
	// le dio. **La ventana lo tiene que enseñar en todas las pestañas**: un permiso
	// permanente que no se ve es un permiso que se olvida.
	Siempre      bool   `json:"siempre"`
	SiempreQuien string `json:"siempreQuien,omitempty"`

	// Dado es lo último que se le ha dado a un agente, para enseñarlo aquí mismo.
	//
	// **Va dentro del estado y no en una llamada aparte**, y eso es una regla de este
	// proyecto que ya costó dos veces: el registro vive en la bóveda, así que pedirlo
	// por su cuenta **falla con la bóveda cerrada** y deja un 400 en la consola de una
	// pantalla que por lo demás funciona. Lo que la pantalla necesita para dibujar va
	// en lo que ya pide.
	Dado []boveda.Apunte `json:"dado,omitempty"`
}

// LoUltimoQueSeEnsena son los apuntes que caben en la pantalla. El resto está en la
// bóveda y se purga solo a los noventa días.
const LoUltimoQueSeEnsena = 20

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
		Orden:         OrdenParaClaudeCode(),
		VSCode:        ConfiguracionParaVSCode(),
		Ruta:          rutaDelServidorMCPOEsfingeMCP(),
		Quiere:        a.permisos.loPendiente(),
		Valvula:       a.permisos.comoVa(time.Now()),
		Siempre:       a.ajustes.Ver().SiempreAlAgente,
		SiempreQuien:  a.ajustes.Ver().SiempreAlAgenteQuien,
	}
	// **Con la bóveda cerrada no hay nada que enseñar, y eso no es un error**: el
	// registro vive dentro de ella.
	if b := a.boveda(); b != nil {
		todo := b.Registro()
		if len(todo) > LoUltimoQueSeEnsena {
			todo = todo[:LoUltimoQueSeEnsena]
		}
		e.Dado = todo
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
			// Retirar a un agente **se lleva todo lo que tuviera concedido**: si no,
			// seguiría pudiendo durante lo que quedara de válvula, que es justo lo que
			// quien pulsa «Retirar» está intentando que no pase.
			a.permisos.retirarLoConcedido()
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
	ruta := rutaDelServidorMCPOEsfingeMCP()
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

// noEraEsoQueAprobabas es lo que se contesta cuando lo que se pulsó ya no es lo que
// está esperando: o lo contestó alguien, o caducó, o llegó otra cosa.
//
// **Dice las tres, porque desde la ventana no se distinguen** y lo que hay que hacer es
// el mismo en los tres casos: mirar qué hay ahora y contestar eso.
const noEraEsoQueAprobabas = "Eso ya no es lo que está esperando: míralo otra vez antes de contestar"

// PlazoParaContestar es lo que una petición espera en la ventana antes de darse por
// abandonada.
//
// **Existe porque desde el 2026-10-19 la segunda petición espera su turno** en vez de
// pisar la primera: sin plazo, una que nadie contesta dejaría a todos los agentes
// esperando para siempre. Dos minutos es lo que tarda alguien en ver la banda de arriba y
// decidir; pasados, la siguiente ocupa su sitio y la vieja se da por caída.
const PlazoParaContestar = 2 * time.Minute

// refDePeticion es la referencia con la que la ventana dice **qué** está aprobando.
//
// Al azar y no un contador, por lo mismo que las referencias de los ficheros de proyecto:
// un número que sube cuenta cuántas veces se le ha pedido algo a esta persona, y eso no
// hace falta para nada.
func refDePeticion() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Sin azar no se puede identificar la petición, y lo que **no** se hace es
		// devolver una vacía: eso haría que la ventana aprobara «lo que haya», que es
		// justo lo que esto viene a quitar.
		return "sin-referencia"
	}
	return hex.EncodeToString(b[:])
}

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
	// Destino es a dónde se lleva, cuando lo que se pide es mover (ADR 0056). Vacío en
	// todo lo demás.
	Destino string `json:"destino,omitempty"`
	// Ref identifica **esta** petición, para que la ventana pueda decir qué está
	// aprobando y no «lo que haya».
	//
	// Sin esto, `AprobarLoQuePideElAgente` concedía lo pendiente a ciegas: si entre que
	// alguien leía la tarjeta y pulsaba llegaba otra petición, **aprobaba la nueva
	// creyendo que aprobaba la que vio**. Es la misma regla que obligó a poner la bóveda
	// aquí: aprobar sin saber qué es aprobar a ciegas.
	Ref string `json:"ref"`
	// PorValvula dice si **este verbo** puede ir por la válvula de cinco minutos y por
	// el permiso que no caduca, y existe para que **la ventana no tenga su propia
	// lista**: los dos botones anchos de la tarjeta cuelgan de aquí.
	//
	// Tenía una, escrita como `que !== "codigo"`, y el día que `mover` entró en el grupo
	// de los que no pasan se quedó corta: la tarjeta ofrecía dos botones que no hacían lo
	// que decían —y su propio comentario advertía de eso mismo—. Ver [PasaPorLaValvula].
	PorValvula bool `json:"porValvula"`
	// Boveda es en cuál está pasando esto (ADR 0055).
	//
	// **Desde que la acción puede ocurrir en una bóveda que no estás mirando, aprobar
	// sin saber dónde es aprobar a ciegas**: «la contraseña de GitHub» no es la misma
	// pregunta en tu bóveda que en la de un cliente. Va sin `omitempty` por lo mismo
	// que en `Estado`: un campo ausente no dice «la personal», no dice nada.
	Boveda string `json:"boveda"`
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
	// pedidaEn es cuándo llegó lo pendiente, **para que caduque**.
	//
	// Hace falta desde que una segunda petición espera su turno en vez de pisar: sin
	// plazo, una que nadie contesta bloquearía a todos los agentes para siempre.
	pedidaEn time.Time
	// concedido es la petición para la que hay un sí, y hasta cuándo vale.
	//
	// **Se guarda entera y se compara entera** (`mismaPeticion`), no por el
	// identificador de la entrada. Eso es lo que permite que **nada ajeno tenga que
	// borrarla**: un sí para «mover DeVuelta de Acme a Personal» no se puede gastar en
	// otro verbo, ni en otra entrada, ni en otra bóveda, ni hacia otro destino, así que
	// conmutar de bóveda no hace falta que lo tire — y tirarlo era el bucle de
	// aprobaciones que vio el cliente. Ver [permisosDelAgente.cerrarLaValvula].
	concedido *loQuePideUnAgente
	hasta     time.Time

	// valvulaHasta es hasta cuándo vale el «todo lo de este agente», y usadas cuántas
	// veces se ha usado ya. Ver [PlazoDeLaValvula].
	valvulaHasta time.Time
	usadas       int
	// loUltimo son los títulos de lo que se le ha ido dando, para que la ventana pueda
	// enseñarlo mientras pasa. **Sin secretos**, como todo lo demás.
	loUltimo []string

	// siempre es el permiso que no caduca (2026-10-08). Vive en las preferencias, no
	// aquí: esto es la copia que se consulta en caliente, y la pone `alArrancar` y
	// cada vez que se enciende o se quita.
	//
	// **`cerrarLaValvula` no lo toca**, y eso es a propósito aunque incomode: cerrar
	// la bóveda cierra la válvula porque fue un permiso para «esta bóveda, ahora», y
	// esto es justo lo contrario. Lo que impide que se olvide es que la ventana lo
	// diga **en todas las pestañas** mientras esté puesto.
	siempre bool
}

// PasaPorLaValvula dice si un verbo puede ir por la válvula de cinco minutos y por el
// permiso que no caduca. **Es el único sitio donde eso se decide**, y por eso es una
// función del verbo y no un argumento.
//
// Antes era un literal en cada llamada a `pedir` y, en la ventana, un `que !== "codigo"`
// escrito a mano. O sea **la misma regla en tres listas**, y el 2026-10-19 estaban
// desalineadas: Go decía que el código, editar y mover no pasan; la ventana solo excluía
// el código; y `concederComo` daba por hecho que todos pasaban. Resultado: el cliente
// aprobó un movimiento con «No volver a preguntar», **no se concedió nada** —bucle
// infinito, no la carrera del día 16— y de paso **se encendió el permiso permanente de
// copiar contraseñas**, que era lo grave. Es el patrón de `vaciarLoSensible` y de
// `enConfiguracionDePruebas` por tercera vez: la regla en un sitio, y quien la necesite
// que pregunte.
//
// La frontera es la de la ADR 0054, dicha entera: **no pasa lo que se le enseña al
// agente** —el código de un solo uso— **ni lo que no se deshace** —cambiar una
// contraseña o una semilla, y mover, que no va a la papelera—.
func PasaPorLaValvula(que string) bool {
	switch que {
	case agente.QueCodigo, agente.QueEditar, agente.QueMover:
		return false
	default:
		return true
	}
}

// pedir deja apuntado lo que se quiere y dice si ya había permiso **para eso mismo**.
//
// Si lo había, **se gasta**: un sí vale para una vez. Si el agente quiere dos, pregunta
// dos veces, que es exactamente lo que el cliente eligió. Si no lo había, lo deja pedido.
//
// Si el verbo pasa por la válvula lo dice [PasaPorLaValvula], y no quien llama: era un
// argumento y eso lo convertía en una lista repetida.
func (p *permisosDelAgente) pedir(q loQuePideUnAgente, ahora time.Time) (bool, string, error) {
	porValvula := PasaPorLaValvula(q.Que)
	p.mu.Lock()
	defer p.mu.Unlock()

	// Un sí para esto mismo, que **se gasta**: un sí vale para una vez.
	if p.concedido != nil && mismaPeticion(*p.concedido, q) && ahora.Before(p.hasta) {
		p.concedido = nil
		return true, boveda.ApuntePreguntado, nil
	}
	// O el «siempre», si lo hay y si este verbo puede pasar por él.
	//
	// **El mismo alcance que la válvula y ni un verbo más**: `porValvula` es la
	// frontera —lo que se le *enseña* al agente y lo que no se deshace no pasan— y
	// aquí se reutiliza a propósito, para que no haya dos listas que puedan
	// separarse. Lo eligió así el cliente teniendo delante la alternativa de dejar
	// borrar fuera: «de todos modos iría a la papelera», y una frontera es más fácil
	// de sostener que dos parecidas.
	if porValvula && p.siempre {
		return true, boveda.ApunteSiempre, nil
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
		return true, boveda.ApunteValvula, nil
	}
	// **Se rellena aquí y no en cada verbo**, que es el sitio por donde pasan todas:
	// ponerlo en los cinco sitios que piden sería la cuarta lista de lo mismo.
	q.PorValvula = porValvula

	// **Y no se pisa lo que otro agente esté esperando.**
	//
	// Con dos a la vez —Claude Desktop y Claude Code— esto pasa de verdad: el segundo
	// llegaba y cambiaba la tarjeta debajo de quien la estaba leyendo. Se le dice que
	// espere su turno, que es lo honesto cuando hay una persona y una ventana.
	//
	// **Entre agentes y no dentro de uno**, y esa frontera es la decisión fina de aquí.
	// El mismo agente pidiendo otra cosa es el mismo interlocutor cambiando de tema:
	// bloquearlo contra sí mismo lo dejaría dos minutos sin poder trabajar porque nadie
	// contestó lo de antes. Lo que protege entonces a quien está leyendo no es esta
	// puerta, es la referencia: si la tarjeta cambió, el clic no concede nada y lo dice.
	//
	// Y si es **lo mismo otra vez** —el reintento de siempre— no es una segunda petición
	// y se deja como está, con su referencia, para que el sí siga valiendo.
	if p.pendiente != nil && p.pendiente.Quien != q.Quien &&
		!mismaPeticion(*p.pendiente, q) &&
		ahora.Before(p.pedidaEn.Add(PlazoParaContestar)) {
		return false, "", fmt.Errorf("%w: «%s», de %s", agente.ErrOtraEsperando,
			p.pendiente.Titulo, p.pendiente.Quien)
	}
	if p.pendiente == nil || !mismaPeticion(*p.pendiente, q) {
		q.Ref = refDePeticion()
		p.pendiente, p.pedidaEn = &q, ahora
	}
	return false, "", nil
}

// ponerElSiempre refleja en caliente lo que digan las preferencias.
func (p *permisosDelAgente) ponerElSiempre(v bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.siempre = v
}

// mismaPeticion dice si dos peticiones son la misma cosa, para gastar un sí.
//
// **Compara lo que define la acción y nada más.** `Cuando` es el instante y nunca
// coincide; `Titulo` sale de la entrada y es informativo. Lo demás sí: quién lo pide,
// qué verbo, qué entrada, en qué bóveda y —si mueve— hacia dónde.
//
// Esto es lo que hace que un sí pueda vivir sin que nadie lo vigile: **no hay forma de
// gastarlo en otra cosa de la que se aprobó.** Antes solo se comparaba el identificador
// de la entrada, y entonces el sí era genérico de más y había que borrarlo cada vez que
// el suelo cambiaba — que es de donde salió el bucle.
func mismaPeticion(a, b loQuePideUnAgente) bool {
	return a.Quien == b.Quien && a.Que == b.Que && a.ID == b.ID &&
		a.Boveda == b.Boveda && a.Destino == b.Destino
}

// cerrarLaValvula cierra el «todo lo de este agente durante un rato», y olvida la
// cuenta.
//
// La llaman **todo lo que cambia el suelo bajo los pies**: cerrar la bóveda, el bloqueo
// por inactividad y cambiar de bóveda. Un «durante cinco minutos» se dio mirando una
// bóveda y no puede sobrevivir a ninguna de esas cosas, porque **no nombra nada**: vale
// para lo que el agente pida después, y lo que pida después puede estar en otro sitio.
//
// **Y no toca el sí concreto, que es lo contrario de lo que hacía hasta la 2.58.1.**
// Ahí está el bucle de aprobaciones que el cliente vio seis veces seguidas: la vuelta
// automática de la ventana —el respiro de la ADR 0055— pasa por aquí, así que aprobar en
// menos de diez segundos borraba el sí **antes** de que el agente reintentara, y Esfinge
// volvía a preguntar. Para siempre, y solo si se aprueba deprisa.
//
// Lo que protege al sí ya no es que algo lo borre, es que **nombra entera la acción que
// se aprobó** (`mismaPeticion`): verbo, entrada, bóveda y destino. Un permiso así no se
// puede gastar en otra parte, así que conmutar no tiene de qué protegerlo. Quien lo
// retira es quien lo quiso retirar: ver [permisosDelAgente.retirarLoConcedido].
//
// El coste, dicho en voz alta: **un sí puede sobrevivir a cerrar la bóveda**, hasta los
// dos minutos de [PlazoDelPermiso]. No abre ninguna puerta —con la bóveda cerrada el
// agente no puede hacer nada, y para reabrirla hace falta la maestra o el dedo— y lo que
// se gastaría es exactamente la acción que esa persona aprobó hace menos de dos minutos.
func (p *permisosDelAgente) cerrarLaValvula() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.valvulaHasta = time.Time{}
	p.usadas = 0
	p.loUltimo = nil
}

// retirarLoConcedido borra **todo** lo que un agente tenga en la mano: la válvula, el sí
// concreto y lo que estuviera pidiendo.
//
// La llaman los dos gestos con los que una persona dice «retírale lo que le has dado»:
// el botón de cortar y retirar al agente de la lista. Ahí sí hay que llevarse el sí
// concreto por delante, porque quien pulsa está intentando justo eso — y dejar vivo un
// permiso de hace un minuto sería desoír el clic.
func (p *permisosDelAgente) retirarLoConcedido() {
	p.cerrarLaValvula()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.concedido = nil
	p.pendiente = nil
}

// conceder dice que sí a lo que estuviera pendiente. Con `unRato`, además abre la
// válvula.
func (p *permisosDelAgente) conceder(ref string, unRato bool, ahora time.Time) (loQuePideUnAgente, bool) {
	return p.concederComo(ref, unRato, false, ahora)
}

// concederComo es lo mismo con el tercer botón: `paraSiempre` deja el permiso puesto
// hasta que alguien lo quite. Quien lo persiste es quien llama, en las preferencias.
func (p *permisosDelAgente) concederComo(ref string, unRato, paraSiempre bool, ahora time.Time) (loQuePideUnAgente, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pendiente == nil {
		return loQuePideUnAgente{}, false
	}
	// **Se concede lo que la ventana dice estar aprobando, no «lo que haya».**
	//
	// `ref` es la referencia de la petición que se enseñó. Si ya no es la pendiente,
	// aquí no se concede nada: quien pulsó estaba mirando otra cosa.
	//
	// **Y se exige, no se acepta vacía.** Dejar pasar «lo que haya» para las llamadas
	// que no la manden sería dejar abierta justo la puerta que esto cierra — y haría que
	// las pruebas se saltaran el paso que la ventana sí hace, que es la trampa del
	// testigo puesto a mano.
	if p.pendiente.Ref != ref {
		return loQuePideUnAgente{}, false
	}
	q := *p.pendiente
	p.pendiente = nil

	// **Un alcance amplio sobre un verbo que no entra en él concede solo esta vez.**
	//
	// Es lo que faltaba y lo que el cliente vio el 2026-10-19: los dos botones anchos
	// ponían la válvula o el «siempre» **y nada más**, dando por hecho que el verbo
	// pasaría por ahí. Para los tres que no pasan —el código, editar lo que no se
	// deshace y mover cuando quita— eso **no concedía nada en absoluto**, y el reintento
	// volvía a preguntar para siempre.
	//
	// Y la otra mitad, que es la de seguridad: **no se enciende un alcance que esta
	// petición no puede usar**. Aprobar un movimiento encendía el permiso permanente de
	// copiar contraseñas y de borrar, o sea **concedía sobre otros verbos** algo que la
	// persona no estaba mirando, y encima sin hacer lo que había pedido. Lo que ve en la
	// pantalla es un movimiento; lo que se concede no puede ser otra cosa.
	//
	// La ventana no debería llegar aquí con esos botones —cuelgan de `PorValvula`, que
	// viaja en la petición—, y esto es el cinturón de debajo: si las dos listas se vuelven
	// a desalinear, lo peor que pasa es que el alcance amplio valga una vez.
	if !PasaPorLaValvula(q.Que) {
		unRato, paraSiempre = false, false
	}

	if paraSiempre {
		// **Y aquí tampoco se pone el permiso de una vez**, por lo mismo que con la
		// válvula: con los dos puestos, la petición siguiente consumiría el
		// individual y el camino del «siempre» no se ejercitaría nunca.
		p.siempre = true
		return q, true
	}
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
	p.concedido = &q
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

// loPendiente es lo que la ventana enseña, **y nada si ya ha caducado**.
//
// Que caduque aquí y no con un reloj es lo que hace que la tarjeta se caiga sola: la
// ventana pregunta por el estado a menudo, así que una petición que nadie contestó en
// [PlazoParaContestar] desaparece sin que haga falta vigilarla. Antes se quedaba puesta
// para siempre, y desde que la segunda espera su turno eso bloquearía a los demás.
func (p *permisosDelAgente) loPendiente() *loQuePideUnAgente {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pendiente == nil {
		return nil
	}
	if time.Now().After(p.pedidaEn.Add(PlazoParaContestar)) {
		p.pendiente = nil
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
		Boveda: f.dondeEstamos(),
	}
	// **Copiar puede pasar por la válvula**, y ésa es exactamente la línea: lo que la
	// válvula cubre es **actuar** —el secreto se queda en este equipo y el portapapeles
	// se borra solo—, no enseñar. Lo que se le enseña al agente pregunta siempre.
	vale, como, err := f.a.permisos.pedir(q, ahora)
	if err != nil {
		return agente.Copiado{}, err
	}
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
		return
	}
	// **Y se avisa, porque si no la ventana no se entera nunca.**
	//
	// El apunte no se escribe al aprobar: se escribe cuando el agente **vuelve a
	// pedirlo**, un instante después, y para entonces la ventana ya ha mirado. Como
	// las secciones no se desmontan, ni cambiar de pestaña ni salir de Ajustes vuelven
	// a preguntar: el registro no aparecía hasta reiniciar Esfinge. Lo vio el cliente
	// con la 2.46.0, probando la primera contraseña de verdad.
	//
	// Es la regla de siempre en esta casa, con otra cara: **avisar a la ventana es que
	// la ventana va a preguntar**, y lo que no se avisa no se pregunta.
	a.sistema.Avisar(EventoAgenteApuntado, q.Quien)
}

// EventoAgenteApuntado avisa de que hay una línea nueva en el registro.
//
// Va aparte de `EventoAgenteQuiere` a propósito: aquél dice «hay algo que contestar» y
// la ventana saca una tarjeta; éste dice «ya está hecho», y lo único que cambia es una
// lista. Mezclarlos haría que terminar de dar una contraseña pareciera otra petición.
const EventoAgenteApuntado = "agente-apuntado"

// AprobarLoQuePideElAgente es el «sí» de la persona.
func (a *App) AprobarLoQuePideElAgente(ref string, unRato bool) error {
	if _, hay := a.permisos.conceder(ref, unRato, time.Now()); !hay {
		return errors.New(noEraEsoQueAprobabas)
	}
	// **Esto sí cuenta como actividad**: es un clic de una persona.
	a.Actividad()
	return nil
}

// AprobarSiempreAlAgente es el tercer botón: no volver a preguntar (2026-10-08).
//
// **Cubre lo mismo que la válvula y ni un verbo más** —copiar contraseñas sí; el
// código, cambiar un secreto y borrar siguen preguntando— y **no caduca**, que es lo
// que lo distingue de los otros dos. Lo pidió el cliente al usarlo de verdad: aprobar
// una por una cansa, y lo que cansa se aprueba sin mirar, que es peor que no preguntar
// porque da la apariencia de control sin el control.
//
// Se guarda en las preferencias **antes** de concederlo: si el disco falla, lo que no
// puede pasar es que el permiso quede puesto en memoria y no se vea en la pantalla
// que lo tiene que anunciar.
func (a *App) AprobarSiempreAlAgente(ref string) error {
	q := a.permisos.loPendiente()
	if q == nil {
		return errors.New("Ya no hay nada que aprobar")
	}
	// **Y no se enciende para un verbo que no lo puede usar.** Esto es lo que le pasó al
	// cliente el 2026-10-19: aprobó un movimiento con este botón y lo que se guardó fue
	// el permiso permanente de **copiar contraseñas y borrar**, que no era lo que estaba
	// mirando — y el movimiento no se hizo. Un permiso se concede sobre lo que la
	// pantalla enseña, nunca sobre lo de al lado. `concederComo` lo repite por su lado:
	// la pregunta es la misma y no puede contestarse distinto en dos sitios.
	if !PasaPorLaValvula(q.Que) {
		if _, hay := a.permisos.concederComo(ref, false, false, time.Now()); !hay {
			return errors.New(noEraEsoQueAprobabas)
		}
		a.Actividad()
		return nil
	}
	if err := a.guardarElSiempre(true, q.Quien); err != nil {
		return err
	}
	if _, hay := a.permisos.concederComo(ref, false, true, time.Now()); !hay {
		return errors.New(noEraEsoQueAprobabas)
	}
	a.Actividad()
	return nil
}

// QuitarElSiempreAlAgente lo revoca. Es la mitad que hace que el botón de arriba sea
// aceptable: un permiso que no se puede quitar no se da.
func (a *App) QuitarElSiempreAlAgente() error {
	if err := a.guardarElSiempre(false, ""); err != nil {
		return err
	}
	a.Actividad()
	return nil
}

func (a *App) guardarElSiempre(v bool, quien string) error {
	p := a.ajustes.Ver()
	p.SiempreAlAgente = v
	p.SiempreAlAgenteQuien = quien
	if err := a.ajustes.Guardar(p); err != nil {
		return err
	}
	a.permisos.ponerElSiempre(v)
	a.avisarDelAgente()
	return nil
}

// avisarDelAgente le dice a la ventana que vuelva a pedir el estado.
func (a *App) avisarDelAgente() {
	a.sistema.Avisar(EventoAgenteApuntado, "")
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

// CortarAlAgente le retira en el acto lo que tenga concedido.
//
// Es el botón que hace que «durante cinco minutos» sea soportable: lo que se concede se
// puede retirar sin esperar a que caduque. **Y se lleva también el sí concreto**, que es
// lo único que conmutar de bóveda ya no toca.
func (a *App) CortarAlAgente() error {
	a.permisos.retirarLoConcedido()
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
		Boveda: f.dondeEstamos(),
	}
	// **`false`**: por la válvula no pasa. Es el único sitio donde se dice, y es lo que
	// hace que la regla sea una regla y no una excepción.
	vale, _, err := f.a.permisos.pedir(q, ahora)
	if err != nil {
		return agente.Codigo{}, err
	}
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
	var comoEs *agente.ComoEs
	if p.Generar {
		// **La hace Esfinge y no vuelve.** El agente crea una cuenta con una
		// contraseña que no ha visto, que es estrictamente mejor que una que se
		// invente él — y eso lo dice la descripción de la herramienta, para que la
		// prefiera.
		// **El alfabeto se puede pedir, y si no se pide es hexadecimal.** Lo señaló el
		// agente del cliente: `esfinge_generar` dejaba elegirlo y `crear` no, así que
		// para un sitio que exija símbolos había que generar aparte y pegarla — o sea,
		// hacerla pasar por el chat, que es lo que este canal evita.
		alfabeto := p.Alfabeto
		if alfabeto == "" {
			alfabeto = alfabetoDeLoQueGenera
		}
		clave, err := f.a.generarParaElAgente(p.Caracteres, 0, alfabeto)
		if err != nil {
			return agente.Escrito{}, err
		}
		e.Secreto = clave
		// **Cómo es, nunca cuál.** Sin esto el agente no puede decir si lo guardado
		// cumple lo que pide el sitio, y lo único que le queda es pedir que se la
		// copien para mirarla — justo lo que este canal evita.
		comoEs = &agente.ComoEs{Largo: len(clave), Alfabeto: alfabeto}
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
	return f.escrito(agente.Escrito{ID: id, Titulo: titulo, Generada: comoEs}), nil
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

	// **Rotar sin escribirla.** Sin esto, cambiar una contraseña obliga a pasarla en
	// claro por la conversación —y ahí se queda para siempre—, que es justo lo que este
	// canal existe para evitar. `crear` lo tenía desde el principio y `editar` no: lo
	// señaló el agente del cliente el 2026-10-15 auditando los esquemas, y tenía razón.
	if p.Generar {
		alfabeto := p.Alfabeto
		if alfabeto == "" {
			alfabeto = alfabetoDeLoQueGenera
		}
		// **Y el largo sí se ofrece** desde el 2026-10-19: un sitio limita lo que
		// acepta en caracteres, y sin esto acortarla obligaba a generarla aparte y
		// pegarla, o sea a hacerla pasar por la conversación de un modelo. Lo destapó
		// el agente del cliente prometiéndole que podía hacerla más corta.
		clave, err := f.a.generarParaElAgente(p.Caracteres, 0, alfabeto)
		if err != nil {
			return agente.Escrito{}, err
		}
		if p.Campos == nil {
			p.Campos = map[string]string{}
		}
		p.Campos["secreto"] = clave
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
			Boveda: f.dondeEstamos(),
		}
		// **Y la válvula no lo cubre**, por lo mismo que el código: dejar una cuenta
		// sin forma de entrar no es reversible de la forma en que lo es una copia.
		vale, _, err := f.a.permisos.pedir(q, ahora)
		if err != nil {
			return agente.Escrito{}, err
		}
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
	return f.escrito(agente.Escrito{ID: p.ID, Titulo: e.Titulo, Cambiados: cambiados}), nil
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
		Boveda: f.dondeEstamos(),
	}
	vale, como, err := f.a.permisos.pedir(q, ahora)
	if err != nil {
		return agente.Escrito{}, err
	}
	if !vale {
		f.a.sistema.Avisar(EventoAgenteQuiere, q)
		return agente.Escrito{}, agente.ErrPideAprobacion
	}
	if err := f.a.BorrarDeBoveda(id); err != nil {
		f.a.apuntarComo(b, q, boveda.ApunteNegado, como)
		return agente.Escrito{}, err
	}
	f.a.apuntarComo(b, q, boveda.ApunteHecho, como)
	return f.escrito(agente.Escrito{ID: id, Titulo: e.Titulo, ALaPapelera: true, Dias: 30}), nil
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

// OrdenParaClaudeCode es cómo se añade Esfinge ahí: una orden, no un fichero.
//
// **La ruta va entera**, por lo mismo que en el bloque de configuración: lo que lo
// arranca puede hacerlo desde cualquier carpeta.
//
// **Y `--scope user` no es un adorno.** El ámbito de `claude mcp add` es `local` de
// fábrica, o sea **solo la carpeta donde se pegó la orden**: quien la pega en un sitio y
// abre Claude Code en otro proyecto se encuentra con que Esfinge no está, y sin ningún
// error que lo explique. Aquí la bóveda es una por persona, no una por repositorio. Lo
// vio el cliente el 2026-10-08, con el conector recién instalado.
func OrdenParaClaudeCode() string {
	return "claude mcp add --scope user esfinge " + rutaDelServidorMCPOEsfingeMCP()
}

// escrito es lo que hay que hacer **después** de cambiar la bóveda desde un agente:
// decir dónde ha ido y **avisar a la ventana**.
//
// Lo segundo faltaba, y es el mismo fallo que el del registro con otra cara: el agente
// escribía, la bóveda quedaba bien y **la pantalla seguía enseñando lo de antes**. El
// cliente guardó una entrada en la bóveda de un proyecto, el agente dijo la verdad —que
// estaba guardada— y él no la veía. El evento existía desde el canal del navegador, que
// lo manda en ocho sitios, y su comentario dice literalmente «alguien de fuera ha
// escrito en la bóveda»: un agente es exactamente eso.
//
// Por eso las dos cosas van juntas en una función y no sueltas en cada verbo: lo que se
// escribe por separado se olvida por separado.
func (f fuenteDelAgente) escrito(e agente.Escrito) agente.Escrito {
	f.a.sistema.Avisar(EventoBovedaCambiada, nil)
	return f.donde(e)
}

// donde le pone a lo escrito **en qué bóveda ha ido**.
//
// Va en la respuesta y no solo en `estado` porque eso es lo que no depende de nada: un
// agente que guarde sin comprobar antes tiene el dato delante al terminar, y lo dice. Si
// no era la bóveda que la persona tenía en la cabeza, se ve en el momento.
func (f fuenteDelAgente) donde(e agente.Escrito) agente.Escrito {
	est := f.a.EstadoBoveda()
	e.Boveda = comoSeLlamaLaBoveda(est.NombreDelProyecto)
	e.EsLaPersonal = est.NombreDelProyecto == ""
	return e
}

// comoSeLlamaLaBoveda da un nombre que se pueda leer, también a la personal.
//
// **La cadena vacía no es un nombre**, y dejarla obligaba a mirar la bandera para saber
// qué se estaba diciendo — que es justo lo que se arregló hace un rato en el estado y se
// quedó a medias aquí. Lo señaló el agente del cliente: «el campo debería devolver su
// nombre y no una cadena vacía».
//
// Y los dos campos siguen, que es la misma separación que `Error` y `Motivo` en el resto
// del protocolo: **el nombre es para leer y la bandera es para decidir**. Un proyecto
// podría llamarse «Personal», así que quien tenga que elegir un camino mira
// `esLaPersonal` y no el texto.
// dondeEstamos es el nombre de la bóveda en la que se está trabajando ahora mismo.
//
// Lo usan las tarjetas de aprobación: desde la ADR 0055 la acción puede estar pasando en
// una bóveda que no estás mirando, y **aprobar sin saber dónde es aprobar a ciegas**.
func (f fuenteDelAgente) dondeEstamos() string {
	return comoSeLlamaLaBoveda(f.a.EstadoBoveda().NombreDelProyecto)
}

func comoSeLlamaLaBoveda(proyecto string) string {
	if proyecto == "" {
		return "Personal"
	}
	return proyecto
}

// Cómo es la contraseña que Esfinge genera para un agente.
//
// **Hexadecimal, y eso no es una elección del canal.** Es la regla de la casa: una
// contraseña con `/`, `+` o `@` dentro de una cadena de conexión la parte, y el error
// que sale —`TypeError: Invalid URL`, o un fallo de autenticación que no cuadra— **no
// menciona la contraseña por ninguna parte**. Un agente que la guarda para un servicio
// no sabe dónde va a acabar pegada.
//
// Veinticuatro bytes son 48 caracteres: de sobra para cualquier política de sitio, que
// es lo que el agente necesita poder decir.
const (
	largoDeLoQueGenera    = 24
	alfabetoDeLoQueGenera = "hex"
)

// GuardarPaqueteMCP escribe el `.mcpb` de Claude Desktop donde diga la persona.
//
// **Existe para no mandar a nadie a la página de descargas.** El servidor MCP ya está
// dentro de esta aplicación —al lado del binario que se está ejecutando—, así que
// pedirle a alguien que vaya a la web, acierte con su sistema y se descargue la versión
// que coincida es trabajo que la ventana puede hacer sola y mejor: así sale **siempre
// el de la versión que tiene puesta**.
//
// **Se comprueba que el servidor está antes de abrir el diálogo**, que es la lección de
// `ExportarLlaves`: preguntar dónde guardar algo que no existe es pedirle a alguien que
// decida sobre nada, y el fallo aparece al final, cuando ya ha elegido.
func (a *App) GuardarPaqueteMCP() (string, error) {
	ruta, err := rutaDelServidorMCP()
	if err != nil || ruta == "" {
		return "", errors.New("No se encuentra el servidor MCP de esta instalación")
	}
	servidor, err := os.ReadFile(ruta)
	if err != nil {
		return "", errors.New("No se encuentra el servidor MCP al lado de Esfinge: " +
			"vuelve a instalarla desde el paquete de tu sistema")
	}

	nombre := "Esfinge-" + a.version + ".mcpb"
	destino, err := a.sistema.ElegirDondeGuardar("Guardar el paquete para Claude Desktop",
		nombre, a.ajustes.CarpetaDeGuardar())
	if err != nil || destino == "" {
		return "", err
	}
	a.ajustes.RecordarCarpetaDeGuardar(filepath.Dir(destino))

	err = escritura.Atomica(destino, escritura.Opciones{}, func(w io.Writer) error {
		return mcpb.ArmarCon(a.version, mcpb.NombreDeDentro(ruta), servidor, w)
	})
	if err != nil {
		return "", err
	}
	return destino, nil
}

// ConfiguracionParaVSCode es el mismo bloque con **otra clave**, y eso no es un detalle
// de estilo.
//
// VS Code lee `servers`; Claude Desktop y Cursor leen `mcpServers`. Pegarle a VS Code
// el bloque de Claude **no da ningún error: no carga nada**, que es exactamente la
// clase de fallo mudo que esta casa ya ha pagado varias veces. Así que se enseñan los
// dos, cada uno con su nombre, en vez de uno «genérico» que falla en la mitad.
func ConfiguracionParaVSCode() string {
	b, err := json.MarshalIndent(map[string]any{
		"servers": map[string]any{
			"esfinge": map[string]any{
				"type":    "stdio",
				"command": rutaDelServidorMCPOEsfingeMCP(),
			},
		},
	}, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

// rutaDelServidorMCPOEsfingeMCP es la ruta, o el nombre a secas si no se sabe dónde
// está — que es lo que hacía cada sitio por su cuenta hasta que fueron cuatro.
func rutaDelServidorMCPOEsfingeMCP() string {
	ruta, err := rutaDelServidorMCP()
	if err != nil || ruta == "" {
		return "esfinge-mcp"
	}
	return ruta
}
