package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/codigos"
	"github.com/webcafeina/esfinge/internal/navegador"
)

// El canal con el navegador, visto desde la aplicación.
//
// Aquí hay dos cosas y conviene no mezclarlas:
//
//   - **Quién contesta** (`fuenteDelNavegador`), que es la única parte de la
//     bóveda que el navegador alcanza. Es corta a propósito y no puede crecer sin
//     tocar `navegador.Fuente`, que lleva su lista.
//   - **Quién tiene permiso** (`navegadoresPermitidos`), que se guarda aparte de
//     las preferencias por un motivo concreto: las preferencias **cruzan el puente
//     hacia la ventana**, y un testigo de emparejamiento no tiene nada que hacer
//     dentro del webview.

// EventoNavegadorPide avisa a la ventana de que un navegador quiere conectarse.
const EventoNavegadorPide = "navegador-pide"

// rutaNavegadores es donde se apunta a quién se le ha dado permiso.
func rutaNavegadores() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "navegadores.json")
}

// RutaDelCanal es el socket por el que habla la extensión.
//
// **Corto a propósito.** La ruta de un socket de dominio unix no puede pasar de
// 104 caracteres en macOS, y allí la carpeta de configuración ya es
// `~/Library/Application Support`: cada letra de más cuenta.
func RutaDelCanal() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "puente.sock")
}

// NavegadorPermitido es un navegador al que se le dijo que sí.
type NavegadorPermitido struct {
	// Testigo es lo que la extensión presenta después. No sale hacia la ventana.
	Testigo string `json:"testigo"`
	Quien   string `json:"quien"`
	Desde   string `json:"desde"`
}

// navegadoresPermitidos lleva la lista y su fichero.
type navegadoresPermitidos struct {
	mu    sync.Mutex
	ruta  string
	lista []NavegadorPermitido
	// porEntregar es un testigo recién aprobado que todavía no ha recogido nadie.
	//
	// **Se entrega una sola vez.** Si el permiso se pudiera reclamar cuando
	// quisiera cualquiera que preguntase, el emparejamiento no valdría nada: le
	// bastaría a un programa con pedir «emparejar» después de que la persona
	// hubiera permitido su navegador.
	porEntregar string
	// pendiente es quién ha pedido permiso y todavía no lo tiene.
	pendiente string
}

func abrirNavegadores(ruta string) *navegadoresPermitidos {
	n := &navegadoresPermitidos{ruta: ruta}
	datos, err := os.ReadFile(ruta)
	if err == nil {
		_ = json.Unmarshal(datos, &n.lista)
	}
	return n
}

func (n *navegadoresPermitidos) guardar() error {
	if n.ruta == "" {
		return nil
	}
	datos, err := json.MarshalIndent(n.lista, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(n.ruta), 0o700); err != nil {
		return err
	}
	return os.WriteFile(n.ruta, datos, 0o600)
}

func (n *navegadoresPermitidos) vale(testigo string) bool {
	if testigo == "" {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, p := range n.lista {
		if p.Testigo == testigo {
			return true
		}
	}
	return false
}

// permitir crea el testigo y lo deja listo para que lo recoja quien lo pidió.
func (n *navegadoresPermitidos) permitir(ahora time.Time) (NavegadorPermitido, error) {
	crudo := make([]byte, 32)
	if _, err := rand.Read(crudo); err != nil {
		return NavegadorPermitido{}, err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	p := NavegadorPermitido{
		Testigo: hex.EncodeToString(crudo),
		Quien:   n.pendiente,
		Desde:   ahora.UTC().Format(time.RFC3339),
	}
	if p.Quien == "" {
		p.Quien = "Un navegador"
	}
	n.lista = append(n.lista, p)
	n.porEntregar = p.Testigo
	n.pendiente = ""
	return p, n.guardar()
}

// recoger devuelve el testigo aprobado, **una sola vez**.
func (n *navegadoresPermitidos) recoger(quien string) (string, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.porEntregar == "" {
		n.pendiente = quien
		return "", false
	}
	t := n.porEntregar
	n.porEntregar = ""
	return t, true
}

func (n *navegadoresPermitidos) quienPide() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.pendiente
}

func (n *navegadoresPermitidos) olvidar(testigo string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	var quedan []NavegadorPermitido
	for _, p := range n.lista {
		if p.Testigo != testigo {
			quedan = append(quedan, p)
		}
	}
	n.lista = quedan
	return n.guardar()
}

func (n *navegadoresPermitidos) ver() []NavegadorPermitido {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]NavegadorPermitido(nil), n.lista...)
}

// ---------------------------------------------------------------- la fuente

// fuenteDelNavegador es lo que el canal puede pedirle a esta aplicación.
//
// **Nada de aquí llama a `Actividad()`, y ésa es la regla que hay que no
// romper.** Una extensión pregunta sola: al cambiar de pestaña, al cargar una
// página, cada vez que el trabajador de MV3 revive. Si eso moviera el reloj del
// bloqueo, navegar mantendría la bóveda abierta para siempre y el «se cierra a
// los quince minutos» dejaría de ser verdad. Es la misma regla que el goteo de
// iconos y que el código de un solo uso, y ya van tres.
type fuenteDelNavegador struct{ a *App }

func (f fuenteDelNavegador) Estado() navegador.Estado {
	e := navegador.Estado{}
	if ruta := rutaBoveda(); ruta != "" {
		if _, err := os.Stat(ruta); err == nil {
			e.Existe = true
		}
	}
	e.Abierta = f.a.boveda() != nil
	return e
}

func (f fuenteDelNavegador) CuentasDe(dominio string) ([]navegador.Cuenta, error) {
	b := f.a.boveda()
	if b == nil {
		return nil, boveda.ErrCerrada
	}
	var out []navegador.Cuenta
	for _, e := range b.Buscar("") {
		// Solo credenciales: una tarjeta o una nota no se rellenan en un formulario
		// de inicio de sesión, y ofrecerlas sería enseñar títulos por gusto.
		if e.Tipo != boveda.TipoCredencial || !leEncaja(e, dominio) {
			continue
		}
		out = append(out, navegador.Cuenta{ID: e.ID, Titulo: e.Titulo, Usuario: e.Usuario})
	}
	return out, nil
}

func (f fuenteDelNavegador) Secreto(id, dominio string) (string, error) {
	e, err := f.entradaDe(id, dominio)
	if err != nil {
		return "", err
	}
	if e.Secreto == "" {
		return "", errors.New("Esa entrada no tiene contraseña")
	}
	return e.Secreto, nil
}

func (f fuenteDelNavegador) Codigo(id, dominio string) (navegador.Codigo, error) {
	e, err := f.entradaDe(id, dominio)
	if err != nil {
		return navegador.Codigo{}, err
	}
	if e.TOTP == "" {
		return navegador.Codigo{}, errors.New("Esa entrada no tiene código de un solo uso")
	}
	s, err := codigos.Leer(e.TOTP)
	if err != nil {
		return navegador.Codigo{}, err
	}
	ahora := time.Now()
	codigo, err := s.En(ahora)
	if err != nil {
		return navegador.Codigo{}, err
	}
	return navegador.Codigo{
		Codigo: codigo, Quedan: int(s.Quedan(ahora).Seconds()),
		Periodo: int(s.Periodo.Seconds()),
	}, nil
}

// entradaDe busca una entrada **y comprueba que es de ese sitio**.
//
// La comprobación va aquí y no en quien llama, a propósito: es el único sitio por
// el que sale un secreto hacia el navegador, así que es donde tiene que estar la
// pregunta «¿esto es de ese sitio?». Con el identificador a mano y sin esto,
// pedir la contraseña del banco desde cualquier página sería una línea de código.
func (f fuenteDelNavegador) entradaDe(id, dominio string) (boveda.Entrada, error) {
	b := f.a.boveda()
	if b == nil {
		return boveda.Entrada{}, boveda.ErrCerrada
	}
	e, hay := b.Ver(id)
	if !hay || e.Papelera {
		return boveda.Entrada{}, errors.New("Esa entrada ya no está en la bóveda")
	}
	if !leEncaja(e, dominio) {
		return boveda.Entrada{}, errors.New("Esa entrada no es de ese sitio")
	}
	return e, nil
}

func leEncaja(e boveda.Entrada, dominio string) bool {
	for _, sitio := range e.Sitios {
		if navegador.Encaja(sitio, dominio) {
			return true
		}
	}
	return false
}

func (f fuenteDelNavegador) Emparejado(testigo string) bool {
	return f.a.navegadores.vale(testigo)
}

// Emparejar deja constancia de quién pide permiso y avisa a la ventana.
//
// **No espera a que alguien conteste**, y eso no es pereza: al otro lado hay un
// proceso que el navegador mata y relanza cada pocos minutos, así que un saludo
// que se queda esperando a una persona se cancela solo. Se pide, se avisa, y la
// extensión vuelve a preguntar cuando la persona ya haya contestado.
func (f fuenteDelNavegador) Emparejar(quien string) (string, error) {
	if testigo, hay := f.a.navegadores.recoger(quien); hay {
		return testigo, nil
	}
	f.a.sistema.Avisar(EventoNavegadorPide, quien)
	return "", errors.New("Permite este navegador en la ventana de Esfinge")
}

// ------------------------------------------------------- encender y apagar

// EstadoDelNavegador es lo que la ventana necesita saber del canal.
type EstadoDelNavegador struct {
	Encendido bool `json:"encendido"`
	// Escuchando dice si de verdad hay un socket abierto: el ajuste puede estar
	// encendido y el canal no haber podido arrancar, y eso hay que verlo.
	Escuchando bool   `json:"escuchando"`
	Donde      string `json:"donde"`
	Error      string `json:"error,omitempty"`
	// Pide es el nombre del navegador que está esperando permiso, si hay alguno.
	Pide       string               `json:"pide,omitempty"`
	Permitidos []NavegadorPermitido `json:"permitidos"`
}

// EstadoDelNavegador dice cómo está el canal. **Sin los testigos**: la ventana
// no los necesita para nada y no tienen por qué vivir en el webview.
func (a *App) EstadoDelNavegador() EstadoDelNavegador {
	a.mu.Lock()
	srv, fallo := a.canal, a.canalFallo
	a.mu.Unlock()

	e := EstadoDelNavegador{
		Encendido:  a.ajustes.Ver().PuenteDelNavegador,
		Escuchando: srv != nil,
		Donde:      RutaDelCanal(),
		Error:      fallo,
		Pide:       a.navegadores.quienPide(),
	}
	for _, p := range a.navegadores.ver() {
		p.Testigo = ""
		e.Permitidos = append(e.Permitidos, p)
	}
	return e
}

// PermitirNavegador es el «sí» de la persona. Crea el testigo, que recogerá la
// extensión la próxima vez que pregunte.
func (a *App) PermitirNavegador() error {
	a.Actividad()
	_, err := a.navegadores.permitir(time.Now())
	return err
}

// OlvidarNavegador retira un permiso dado. Se identifica por cuándo se dio,
// porque es lo único que la ventana conoce: el testigo no cruza el puente.
func (a *App) OlvidarNavegador(desde string) error {
	a.Actividad()
	for _, p := range a.navegadores.ver() {
		if p.Desde == desde {
			return a.navegadores.olvidar(p.Testigo)
		}
	}
	return errors.New("Ese navegador ya no estaba permitido")
}

// aplicarCanal enciende o apaga el canal según el ajuste. Se llama al arrancar y
// cada vez que se guardan las preferencias, como los dos relojes de la bóveda.
func (a *App) aplicarCanal(p Preferencias) {
	a.mu.Lock()
	yaEsta := a.canal != nil
	a.mu.Unlock()

	if p.PuenteDelNavegador == yaEsta {
		return
	}
	if !p.PuenteDelNavegador {
		a.pararCanal()
		return
	}

	ruta := RutaDelCanal()
	srv, err := navegador.Servir(ruta, fuenteDelNavegador{a})
	a.mu.Lock()
	a.canal = srv
	a.canalFallo = ""
	if err != nil {
		a.canalFallo = err.Error()
	}
	a.mu.Unlock()
}

func (a *App) pararCanal() {
	a.mu.Lock()
	srv := a.canal
	a.canal = nil
	a.canalFallo = ""
	a.mu.Unlock()
	if srv != nil {
		_ = srv.Parar()
	}
}
