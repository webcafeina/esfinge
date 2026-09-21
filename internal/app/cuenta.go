package app

// La cuenta, vista desde la ventana (ADR 0035, plan en docs/cuentas.md).
//
// Tres caminos llegan a lo mismo —una bóveda en este equipo que se sincroniza con
// la de la cuenta—:
//
//   - **Crear la cuenta** desde cero o con la bóveda que ya hay aquí.
//   - **Entrar** desde otro equipo: correo, contraseña y el código del correo.
//     Si aquí ya había otra bóveda, se pregunta si se junta con la de la cuenta o
//     se aparta, y **nunca se borra**.
//   - **Volver a entrar** cuando la sesión caduca, con la bóveda abierta.
//
// Tres reglas que no se pueden olvidar:
//
//   - **La contraseña de la cuenta es la maestra de la bóveda** (ADR 0037), y con
//     cuenta tiene que ser al menos «Buena».
//   - **La sesión se guarda sellada con la clave de la bóveda**: sin la maestra, un
//     disco robado no habla con el servidor, y solo se sincroniza con la bóveda
//     abierta.
//   - **Nada de la sincronización llama a `Actividad()`**: una bóveda abierta encima
//     de la mesa tiene que cerrarse igual aunque se esté sincronizando.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/cuenta"
	"github.com/webcafeina/esfinge/internal/escritura"
	"github.com/webcafeina/esfinge/internal/sincro"
)

// EventoSincro avisa a la ventana de cómo va la sincronización.
const EventoSincro = "sincro"

// EstadoSincro es lo que se enseña de la sincronización.
type EstadoSincro struct {
	// Estado: apagada · sincronizando · al-dia · sin-conexion · sin-red ·
	// hay-que-entrar · muchos-borrados · error.
	Estado string `json:"estado"`
	// Ultima es cuándo salió bien la última pasada, en RFC3339.
	Ultima string `json:"ultima,omitempty"`
	// Mensaje es lo que hay que enseñar si algo no va bien, tal cual.
	Mensaje string `json:"mensaje,omitempty"`
}

// EstadoCuenta es lo que la ventana necesita saber de la cuenta.
type EstadoCuenta struct {
	// Modo: "" si todavía no se ha elegido —la bienvenida—, «local» o «cuenta».
	Modo     string       `json:"modo"`
	Correo   string       `json:"correo,omitempty"`
	Equipo   string       `json:"equipo,omitempty"`
	Servidor string       `json:"servidor"`
	Sincro   EstadoSincro `json:"sincro"`
}

// ResultadoEntrada dice en qué punto se ha quedado entrar en una cuenta.
type ResultadoEntrada struct {
	// NecesitaCodigo: ha llegado un código al correo; se sigue con ConfirmarEntrada.
	NecesitaCodigo bool `json:"necesitaCodigo"`
	// HayOtraBoveda: en este equipo ya había una bóveda distinta de la de la
	// cuenta; se sigue con ResolverOtraBoveda.
	HayOtraBoveda bool `json:"hayOtraBoveda"`
	// Listo: la bóveda de la cuenta está abierta aquí y sincronizándose.
	Listo bool `json:"listo"`
	// Apartada es dónde ha quedado la bóveda que había aquí, si se apartó.
	Apartada string `json:"apartada,omitempty"`
}

// datosCuenta es lo que se guarda en `cuenta.json`, junto a la bóveda. **No va en
// las preferencias**: ésas se guardan como objeto entero desde la ventana, y un
// guardado a medias cambiaría el modo o apagaría la sincronización en silencio.
type datosCuenta struct {
	Modo         string `json:"modo"`
	Correo       string `json:"correo,omitempty"`
	Cuenta       string `json:"cuenta,omitempty"`
	Equipo       string `json:"equipo,omitempty"`
	NombreEquipo string `json:"nombreEquipo,omitempty"`
	// Sesion y Confianza van **selladas con la clave de la bóveda**.
	Sesion    string `json:"sesion,omitempty"`
	Confianza string `json:"confianza,omitempty"`
}

// laCuenta es el estado de la cuenta en memoria. Va detrás de su propio cerrojo:
// la sincronización avisa desde su gorrutina.
type laCuenta struct {
	mu      sync.Mutex
	cliente *cuenta.Cliente
	// sesion es la de ahora, desellada, mientras la bóveda está abierta.
	sesion  string
	estado  EstadoSincro
	entrada *entradaPendiente
	marcha  *sincroEnMarcha
	// retoBorrado es el del código para borrar la cuenta, entre pedirlo y usarlo.
	retoBorrado string
	// espera tras un guardado antes de subir; cero es la de siempre. Solo la
	// tocan las pruebas.
	espera time.Duration
}

// entradaPendiente es lo que hay que recordar entre pedir el código y escribirlo.
// La contraseña maestra se queda en memoria esos minutos porque hace falta para
// abrir la bóveda que baja: no se escribe en ninguna parte.
type entradaPendiente struct {
	correo, maestra, nombre, reto string
	sesion                        *cuenta.Sesion
	remota                        *boveda.Boveda
	// deNuevo: es volver a entrar con la bóveda de la cuenta ya abierta aquí.
	deNuevo bool
}

type sincroEnMarcha struct {
	cancelar context.CancelFunc
	vig      *sincro.Vigilante
	s        *sincro.Sincronizador
}

// vaciarAlCerrar es lo más que se espera al cerrar la bóveda para subir lo que
// quede: al cerrar, el vigilante ya no subiría nada, y lo guardado justo antes se
// quedaría en este equipo hasta volver a abrirla.
const vaciarAlCerrar = 3 * time.Second

// ApuntarCuentasA cambia el servidor de cuentas. Lo usan el servidor de
// desarrollo y las pruebas. **Función y no método**, por lo mismo que
// ApuntarAAPI: la ventana no puede elegir a qué servidor manda la bóveda.
func ApuntarCuentasA(a *App, raiz string) {
	a.cu.mu.Lock()
	defer a.cu.mu.Unlock()
	a.cu.cliente = cuenta.Nuevo(raiz)
	a.cu.cliente.Version = a.version
}

func rutaCuenta() string {
	ruta := rutaBoveda()
	if ruta == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(ruta), "cuenta.json")
}

func leerDatosCuenta() datosCuenta {
	var d datosCuenta
	crudo, err := os.ReadFile(rutaCuenta())
	if err != nil {
		return d
	}
	_ = json.Unmarshal(crudo, &d)
	return d
}

func guardarDatosCuenta(d datosCuenta) error {
	crudo, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return escritura.Atomica(rutaCuenta(), escritura.Opciones{CrearCarpeta: true}, func(w io.Writer) error {
		_, err := w.Write(append(crudo, '\n'))
		return err
	})
}

// nombreDelEquipo es lo que el servidor enseña en la lista de equipos y en el
// correo de «un equipo nuevo ha entrado». El del ordenador, sin el «.local».
func nombreDelEquipo() string {
	n, err := os.Hostname()
	n = strings.TrimSuffix(strings.TrimSpace(n), ".local")
	if err != nil || n == "" {
		return "Equipo sin nombre"
	}
	return n
}

func (a *App) ctxCuenta() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// ------------------------------------------------------------------ estado

// EstadoDeCuenta dice si este equipo trabaja en local o con cuenta, y cómo va.
func (a *App) EstadoDeCuenta() EstadoCuenta {
	d := leerDatosCuenta()
	modo := d.Modo
	if modo == "" {
		// Quien ya tenía bóveda antes de que hubiera cuentas trabaja en local: la
		// bienvenida es para quien no tiene nada.
		if _, err := os.Stat(rutaBoveda()); err == nil {
			modo = "local"
		}
	}
	a.cu.mu.Lock()
	defer a.cu.mu.Unlock()
	e := EstadoCuenta{Modo: modo, Servidor: a.cu.cliente.Raiz(), Sincro: a.cu.estado}
	if modo == "cuenta" {
		e.Correo, e.Equipo = d.Correo, d.NombreEquipo
		if e.Sincro.Estado == "" {
			e.Sincro.Estado = "apagada" // la bóveda está cerrada
		}
	} else {
		e.Sincro = EstadoSincro{Estado: "apagada"}
	}
	return e
}

// ElegirModoLocal apunta que en este equipo se trabaja sin cuenta: lo que se elige
// en la bienvenida. Se puede cambiar de idea en Ajustes.
func (a *App) ElegirModoLocal() error {
	d := leerDatosCuenta()
	if d.Modo == "cuenta" {
		return errors.New("Este equipo está en una cuenta")
	}
	d.Modo = "local"
	return guardarDatosCuenta(d)
}

// SalirDeCuenta deja la cuenta **en este equipo**: la bóveda se queda aquí tal
// como está y deja de sincronizarse. La cuenta sigue en el servidor, y los otros
// equipos no cambian. Pide la contraseña maestra, por lo mismo que borrar la
// bóveda: no es algo que deba poder hacer quien pase por delante.
func (a *App) SalirDeCuenta(maestra string) error {
	d := leerDatosCuenta()
	if d.Modo != "cuenta" {
		return errors.New("Este equipo no está en ninguna cuenta")
	}
	ruta := rutaBoveda()
	if _, err := boveda.Abrir(ruta, maestra); err != nil {
		return errors.New("Esa no es la contraseña de esta bóveda")
	}
	// Primero se sube lo que quede, para que no se pierda en el camino, y se cierra
	// la sesión en el servidor: sin eso seguiría viva hasta caducar. Si no hay red,
	// se sale igual: la sesión sellada se borra de aquí y ya no sirve.
	a.cu.mu.Lock()
	m, token := a.cu.marcha, a.cu.sesion
	a.cu.mu.Unlock()
	if m != nil {
		m.cancelar()
		_ = m.s.Vaciar(vaciarAlCerrar)
	}
	if token != "" {
		_ = a.cliente().CerrarSesion(a.ctxCuenta(), token)
	}
	return a.olvidarLaCuentaAqui()
}

// olvidarLaCuentaAqui deja este equipo en local: la bóveda como está, sin
// sincronizar y sin nada de la cuenta en el disco.
func (a *App) olvidarLaCuentaAqui() error {
	a.alCerrarLaBoveda()
	a.cu.mu.Lock()
	a.cu.estado = EstadoSincro{}
	a.cu.mu.Unlock()
	if b := a.boveda(); b != nil {
		b.AlGuardar(nil)
	}
	if err := (sincro.JuntoALaBoveda{Ruta: rutaBoveda()}).Olvidar(); err != nil {
		return err
	}
	return guardarDatosCuenta(datosCuenta{Modo: "local"})
}

// SincronizarAhora pide una pasada sin esperar al reloj.
func (a *App) SincronizarAhora() error {
	a.cu.mu.Lock()
	m := a.cu.marcha
	a.cu.mu.Unlock()
	if m == nil {
		return errors.New("Aquí no se está sincronizando: abre la bóveda de la cuenta")
	}
	m.vig.Pedir()
	return nil
}

// ------------------------------------------------------------------ crear la cuenta

// EmpezarRegistro pide el código para crear la cuenta.
func (a *App) EmpezarRegistro(correo string) error {
	if leerDatosCuenta().Modo == "cuenta" {
		return errors.New("Este equipo ya está en una cuenta")
	}
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return err
	}
	return a.cliente().EmpezarAlta(a.ctxCuenta(), c)
}

// MaestraSirveParaCuenta dice si una contraseña vale como contraseña de una
// cuenta: con cuenta, la bóveda cifrada vive también en el servidor, y si alguien
// la robara de allí la única defensa sería la contraseña (ADR 0035).
func maestraSirveParaCuenta(maestra string) error {
	if cripto.Evaluar(maestra).Nivel < 3 {
		return errors.New("Con cuenta, la contraseña maestra tiene que ser al menos «Buena»: alárgala o usa varias palabras sin relación")
	}
	return nil
}

// TerminarRegistro crea la cuenta con el código del correo.
//
// Si en este equipo no hay bóveda, se crea una con `maestra` y **se devuelve su
// clave de recuperación**, una sola vez, como al crearla en local. La bóveda nueva
// se crea en memoria y **solo se guarda si el servidor dice que sí**: guardada
// antes, un código mal escrito dejaría una bóveda cuya clave de recuperación no ha
// visto nadie.
//
// Si ya la hay, `maestra` es la suya —si está cerrada, se abre con ella: pedir que
// se abriera antes obligaba a salir del asistente y volver a escribirlo todo—. La
// contraseña de la cuenta es la maestra de la bóveda, y con cuenta tiene que ser al
// menos «Buena». **Si la de ahora no llega, `nueva` es la que la sustituye**, y se
// cambia en la bóveda después de que el servidor haya dicho que sí: así un código
// mal escrito no deja la bóveda con otra contraseña y sin cuenta.
func (a *App) TerminarRegistro(correo, codigo, maestra, nueva string) (string, error) {
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return "", err
	}
	ruta := rutaBoveda()
	var b *boveda.Boveda
	recuperacion := ""
	creada := false
	efectiva := maestra
	if _, err := os.Stat(ruta); err == nil {
		abierta, err := boveda.Abrir(ruta, maestra)
		if err != nil {
			return "", errors.New("La contraseña maestra de tu bóveda no es ésa")
		}
		if nueva != "" {
			if err := maestraSirveParaCuenta(nueva); err != nil {
				abierta.Cerrar()
				return "", err
			}
			if nueva == maestra {
				abierta.Cerrar()
				return "", errors.New("La contraseña nueva tiene que ser distinta de la de ahora")
			}
			efectiva = nueva
		} else if err := maestraSirveParaCuenta(maestra); err != nil {
			abierta.Cerrar()
			return "", errors.New("Tu contraseña maestra de ahora no llega a «Buena», y con cuenta hace falta: pon una nueva")
		}
		if b = a.boveda(); b == nil {
			b = abierta
			a.cambiarBoveda(b)
			a.Actividad()
		} else {
			abierta.Cerrar()
		}
	} else {
		if err := maestraSirveParaCuenta(maestra); err != nil {
			return "", err
		}
		if b, recuperacion, err = boveda.CrearEnMemoria(maestra); err != nil {
			return "", err
		}
		creada = true
	}

	sal, err := cripto.Azar(16)
	if err != nil {
		return "", err
	}
	clave, err := cuenta.DerivarAcceso(efectiva, sal, cuenta.PorDefecto)
	if err != nil {
		return "", err
	}
	posesion, err := b.Posesion()
	if err != nil {
		return "", err
	}
	nombre := nombreDelEquipo()
	s, err := a.cliente().TerminarAlta(a.ctxCuenta(), cuenta.Alta{
		Correo: c, Codigo: codigo, Sal: sal, Argon2: cuenta.PorDefecto,
		ClaveDeAcceso: clave, Posesion: posesion, Dispositivo: nombre, Confiar: true,
	})
	if err != nil {
		return "", err
	}
	if creada {
		if err := b.GuardarEn(ruta); err != nil {
			return "", err
		}
		a.ponerBoveda(b)
		a.Actividad()
	}
	// La contraseña nueva, en la bóveda, antes de que empiece a sincronizarse: lo
	// que suba tiene que llevar ya la ranura que abre la cuenta.
	if efectiva != maestra {
		if err := b.CambiarMaestra(efectiva); err != nil {
			return "", err
		}
	}
	if err := a.quedarseCon(b, c, nombre, s); err != nil {
		return "", err
	}
	return recuperacion, nil
}

// ------------------------------------------------------------------ entrar

// EntrarEnCuenta empieza a entrar en una cuenta desde este equipo. Casi siempre
// pide un código, que llega al correo; con el equipo ya de confianza, no.
//
// Con la bóveda de la cuenta ya abierta aquí es **volver a entrar**, cuando la
// sesión ha caducado.
func (a *App) EntrarEnCuenta(correo, maestra string) (ResultadoEntrada, error) {
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	d := leerDatosCuenta()
	p := &entradaPendiente{correo: c, maestra: maestra, nombre: nombreDelEquipo()}
	confianza := ""
	if d.Modo == "cuenta" && d.Correo != c {
		return ResultadoEntrada{}, errors.New("Este equipo ya está en otra cuenta")
	}
	// Con la bóveda de la cuenta abierta es volver a entrar; cerrada, es entrar sin
	// más —el caso de quien cambió la contraseña en otro equipo y aquí no abre con
	// la nueva—, y terminarEntrada se encarga de ponerla al día.
	if b := a.boveda(); d.Modo == "cuenta" && b != nil {
		p.deNuevo = true
		if d.NombreEquipo != "" {
			p.nombre = d.NombreEquipo
		}
		if d.Confianza != "" {
			if claro, err := b.AbrirSecreto(d.Confianza); err == nil {
				confianza = string(claro)
			}
		}
	}

	pre, err := a.cliente().Prelogin(a.ctxCuenta(), c)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	clave, err := cuenta.DerivarAcceso(maestra, pre.Sal, pre.Argon2)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	s, reto, err := a.cliente().Entrar(a.ctxCuenta(), c, clave, p.nombre, confianza)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	p.reto = reto
	a.cu.mu.Lock()
	a.cu.entrada = p
	a.cu.mu.Unlock()
	if s != nil {
		return a.terminarEntrada(p, *s)
	}
	return ResultadoEntrada{NecesitaCodigo: true}, nil
}

// ConfirmarEntrada termina de entrar con el código que ha llegado al correo.
func (a *App) ConfirmarEntrada(codigo string) (ResultadoEntrada, error) {
	a.cu.mu.Lock()
	p := a.cu.entrada
	a.cu.mu.Unlock()
	if p == nil || p.reto == "" {
		return ResultadoEntrada{}, errors.New("Empieza otra vez: no hay ninguna entrada a medias")
	}
	s, err := a.cliente().ConfirmarEntrada(a.ctxCuenta(), p.reto, codigo, true)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	return a.terminarEntrada(p, s)
}

func (a *App) terminarEntrada(p *entradaPendiente, s cuenta.Sesion) (ResultadoEntrada, error) {
	if p.deNuevo {
		b := a.boveda()
		if b == nil {
			return ResultadoEntrada{}, boveda.ErrCerrada
		}
		a.olvidarEntrada()
		return ResultadoEntrada{Listo: true}, a.quedarseCon(b, p.correo, p.nombre, s)
	}

	datos, version, _, err := a.cliente().Bajar(a.ctxCuenta(), s.Token, 0)
	var remota *boveda.Boveda
	switch {
	case errors.Is(err, cuenta.ErrSinBoveda):
		// Una cuenta sin bóveda: la de aquí será la primera.
	case err != nil:
		return ResultadoEntrada{}, err
	default:
		if remota, err = boveda.AbrirEnMemoria(datos, p.maestra); err != nil {
			return ResultadoEntrada{}, errors.New("La contraseña de la cuenta no abre la bóveda que hay en ella")
		}
	}

	ruta := rutaBoveda()
	local, errLocal := os.ReadFile(ruta)
	if errLocal != nil {
		// Nada en este equipo: la bóveda de la cuenta es la de aquí.
		if remota == nil {
			return ResultadoEntrada{}, errors.New("Esta cuenta todavía no tiene bóveda, y en este equipo tampoco hay ninguna")
		}
		if err := remota.GuardarEn(ruta); err != nil {
			return ResultadoEntrada{}, err
		}
		a.cambiarBoveda(remota)
		a.olvidarEntrada()
		return ResultadoEntrada{Listo: true}, a.quedarseCon(remota, p.correo, p.nombre, s)
	}

	idLocal, _ := boveda.IDDe(local)
	if remota == nil || idLocal == remota.ID() {
		// La misma bóveda —este equipo ya estuvo en la cuenta— o una cuenta sin
		// bóveda que va a recibir la de aquí. Se abre la de aquí con la contraseña.
		b, err := boveda.Abrir(ruta, p.maestra)
		if err != nil && remota != nil {
			// La de aquí tiene la contraseña de antes: se cambió en otro equipo. La
			// clave de la bóveda es la misma, así que se abre la de aquí con la de la
			// cuenta y **se funde** con lo del servidor: trae la ranura nueva y no se
			// pierde lo que hubiera aquí sin subir.
			if local, err := boveda.AbrirConLaLlaveDe(ruta, remota); err == nil {
				_, base, _ := (sincro.JuntoALaBoveda{Ruta: ruta}).Cargar()
				if _, err := local.Fundir(datos, version, base, boveda.OpcionesDeFusion{}); err == nil {
					remota.Cerrar()
					a.cambiarBoveda(local)
					a.Actividad()
					a.olvidarEntrada()
					return ResultadoEntrada{Listo: true}, a.quedarseCon(local, p.correo, p.nombre, s)
				}
				local.Cerrar()
			}
			// Si no se puede fundir, manda la de la cuenta, y la de aquí se aparta en
			// vez de pisarse.
			apartada, err := apartar(ruta)
			if err != nil {
				return ResultadoEntrada{}, err
			}
			if err := remota.GuardarEn(ruta); err != nil {
				return ResultadoEntrada{}, err
			}
			a.cambiarBoveda(remota)
			a.olvidarEntrada()
			return ResultadoEntrada{Listo: true, Apartada: apartada}, a.quedarseCon(remota, p.correo, p.nombre, s)
		}
		if err != nil {
			return ResultadoEntrada{}, errors.New("Esa contraseña no abre la bóveda de este equipo")
		}
		if remota == nil {
			if err := maestraSirveParaCuenta(p.maestra); err != nil {
				return ResultadoEntrada{}, err
			}
		}
		a.cambiarBoveda(b)
		a.Actividad()
		a.olvidarEntrada()
		return ResultadoEntrada{Listo: true}, a.quedarseCon(b, p.correo, p.nombre, s)
	}

	// Aquí hay otra bóveda: se pregunta antes de tocar nada.
	a.cu.mu.Lock()
	p.sesion, p.remota = &s, remota
	a.cu.mu.Unlock()
	return ResultadoEntrada{HayOtraBoveda: true}, nil
}

// ResolverOtraBoveda decide qué pasa con la bóveda que ya había en este equipo al
// entrar en la cuenta. Con `juntar`, sus entradas pasan a la de la cuenta; sin él,
// no. **En los dos casos la de aquí se aparta y no se borra**: queda en su carpeta
// con otro nombre. Para juntar hace falta abrirla: con `maestraLocal`, o con la
// de la cuenta si está vacía.
func (a *App) ResolverOtraBoveda(juntar bool, maestraLocal string) (ResultadoEntrada, error) {
	a.cu.mu.Lock()
	p := a.cu.entrada
	a.cu.mu.Unlock()
	if p == nil || p.remota == nil || p.sesion == nil {
		return ResultadoEntrada{}, errors.New("Empieza otra vez: no hay ninguna entrada a medias")
	}
	ruta := rutaBoveda()
	if juntar {
		llave := maestraLocal
		if llave == "" {
			llave = p.maestra
		}
		local, err := boveda.Abrir(ruta, llave)
		if err != nil {
			return ResultadoEntrada{}, errors.New("Esa contraseña no abre la bóveda de este equipo")
		}
		if _, err := p.remota.Traer(local); err != nil {
			return ResultadoEntrada{}, err
		}
		local.Cerrar()
	}
	apartada, err := apartar(ruta)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	if err := p.remota.GuardarEn(ruta); err != nil {
		return ResultadoEntrada{}, err
	}
	a.cambiarBoveda(p.remota)
	a.Actividad()
	a.olvidarEntrada()
	return ResultadoEntrada{Listo: true, Apartada: apartada}, a.quedarseCon(p.remota, p.correo, p.nombre, *p.sesion)
}

// apartar le cambia el nombre a la bóveda de este equipo —y a lo que va con ella—
// para dejar sitio a la de la cuenta. Devuelve dónde ha quedado.
func apartar(ruta string) (string, error) {
	destino := fmt.Sprintf("%s.apartada-%s", ruta, time.Now().Format("2006-01-02-150405"))
	if err := os.Rename(ruta, destino); err != nil {
		return "", err
	}
	_ = os.Rename(ruta+".anterior", destino+".anterior")
	_ = os.Rename(boveda.RutaDeIconos(ruta), boveda.RutaDeIconos(destino))
	_ = (sincro.JuntoALaBoveda{Ruta: ruta}).Olvidar()
	return destino, nil
}

func (a *App) olvidarEntrada() {
	a.cu.mu.Lock()
	a.cu.entrada = nil
	a.cu.mu.Unlock()
}

// cambiarBoveda deja `b` como la bóveda abierta, cerrando la que hubiera.
func (a *App) cambiarBoveda(b *boveda.Boveda) {
	a.pararSincro()
	a.mu.Lock()
	antes := a.bov
	a.bov = b
	a.mu.Unlock()
	if antes != nil && antes != b {
		antes.Cerrar()
	}
}

// quedarseCon apunta la cuenta en este equipo, con la sesión sellada con la clave
// de la bóveda, y arranca la sincronización.
func (a *App) quedarseCon(b *boveda.Boveda, correo, nombre string, s cuenta.Sesion) error {
	d := leerDatosCuenta()
	sesion, err := b.SellarSecreto([]byte(s.Token))
	if err != nil {
		return err
	}
	d.Modo, d.Correo, d.NombreEquipo, d.Sesion = "cuenta", correo, nombre, sesion
	if s.Cuenta != "" {
		d.Cuenta = s.Cuenta
	}
	if s.Dispositivo != "" {
		d.Equipo = s.Dispositivo
	}
	if s.Confianza != "" {
		if d.Confianza, err = b.SellarSecreto([]byte(s.Confianza)); err != nil {
			return err
		}
	}
	if err := guardarDatosCuenta(d); err != nil {
		return err
	}
	a.cu.mu.Lock()
	a.cu.sesion = s.Token
	a.cu.mu.Unlock()
	a.arrancarSincro(b)
	return nil
}

// ------------------------------------------------------------------ la sincronización

func (a *App) cliente() *cuenta.Cliente {
	a.cu.mu.Lock()
	defer a.cu.mu.Unlock()
	return a.cu.cliente
}

// alAbrirLaBoveda arranca la sincronización si este equipo está en una cuenta.
// La llaman AbrirBoveda y quien deje una bóveda abierta.
func (a *App) alAbrirLaBoveda(b *boveda.Boveda) {
	d := leerDatosCuenta()
	if d.Modo != "cuenta" {
		return
	}
	token := ""
	if d.Sesion != "" {
		if claro, err := b.AbrirSecreto(d.Sesion); err == nil {
			token = string(claro)
		}
	}
	a.cu.mu.Lock()
	a.cu.sesion = token
	a.cu.mu.Unlock()
	if token == "" {
		a.ponerEstado(EstadoSincro{Estado: "hay-que-entrar", Mensaje: "Vuelve a entrar en la cuenta para sincronizar"})
		return
	}
	a.arrancarSincro(b)
}

func (a *App) nuevoSincronizador(b *boveda.Boveda) *sincro.Sincronizador {
	return &sincro.Sincronizador{
		Servidor: a.cliente(),
		Boveda:   b,
		Memoria:  sincro.JuntoALaBoveda{Ruta: b.Ruta()},
		Token: func() string {
			a.cu.mu.Lock()
			defer a.cu.mu.Unlock()
			return a.cu.sesion
		},
	}
}

func (a *App) arrancarSincro(b *boveda.Boveda) {
	a.pararSincro()
	s := a.nuevoSincronizador(b)
	a.cu.mu.Lock()
	espera := a.cu.espera
	a.cu.mu.Unlock()
	vig := &sincro.Vigilante{S: s, Avisar: a.alSincronizar, Espera: espera}
	ctx, cancelar := context.WithCancel(a.ctxCuenta())
	// Cada guardado —de la ventana, del navegador, del importador— pide una pasada.
	b.AlGuardar(vig.Pedir)
	a.cu.mu.Lock()
	a.cu.marcha = &sincroEnMarcha{cancelar: cancelar, vig: vig, s: s}
	a.cu.mu.Unlock()
	a.ponerEstado(EstadoSincro{Estado: "sincronizando"})
	go vig.Vigilar(ctx)
}

// pararSincro para la sincronización, sin olvidar la sesión: la llama también
// arrancarSincro antes de empezar otra.
func (a *App) pararSincro() {
	a.cu.mu.Lock()
	m := a.cu.marcha
	a.cu.marcha = nil
	if a.cu.estado.Estado != "" {
		a.cu.estado.Estado = "apagada"
	}
	a.cu.mu.Unlock()
	if m != nil {
		m.cancelar()
	}
}

// alCerrarLaBoveda para la sincronización y olvida la sesión desellada. La llaman
// todos los sitios donde la bóveda se cierra: a mano, por el reloj del bloqueo y
// al borrarla. **Una bóveda cerrada no sincroniza**: la sesión solo existe en
// claro mientras está abierta.
func (a *App) alCerrarLaBoveda() {
	a.cu.mu.Lock()
	m := a.cu.marcha
	a.cu.mu.Unlock()
	if m != nil {
		// Primero el vigilante, para que no empiece otra pasada; después se sube lo
		// que quede, con su tope.
		m.cancelar()
		_ = m.s.Vaciar(vaciarAlCerrar)
	}
	a.pararSincro()
	a.cu.mu.Lock()
	a.cu.sesion = ""
	a.cu.mu.Unlock()
}

// alSincronizar recoge lo que ha hecho cada pasada. **No llama a `Actividad()`**.
func (a *App) alSincronizar(r sincro.Resultado, err error) {
	var e EstadoSincro
	switch {
	case err == nil:
		e = EstadoSincro{Estado: "al-dia", Ultima: time.Now().UTC().Format(time.RFC3339)}
		if r.Bajo {
			// Ha llegado algo de otro equipo: la lista se vuelve a pedir.
			a.sistema.Avisar(EventoBovedaCambiada, nil)
		}
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, cuenta.ErrSinRed):
		e = EstadoSincro{Estado: "sin-red", Mensaje: err.Error()}
	case cuenta.SesionCaducada(err):
		e = EstadoSincro{Estado: "hay-que-entrar", Mensaje: "La sesión ha caducado: vuelve a entrar en la cuenta para sincronizar"}
	case errors.Is(err, boveda.ErrMuchosBorrados):
		e = EstadoSincro{Estado: "muchos-borrados", Mensaje: err.Error()}
	default:
		var delServidor *cuenta.ErrorDelServidor
		if errors.As(err, &delServidor) || errors.Is(err, sincro.ErrRetroceso) || errors.Is(err, boveda.ErrRetroceso) {
			e = EstadoSincro{Estado: "error", Mensaje: err.Error()}
		} else {
			e = EstadoSincro{Estado: "sin-conexion", Mensaje: "Sin conexión con el servidor de cuentas: se sigue trabajando aquí y se sube al volver"}
		}
	}
	a.cu.mu.Lock()
	if e.Ultima == "" {
		e.Ultima = a.cu.estado.Ultima
	}
	a.cu.mu.Unlock()
	a.ponerEstado(e)
}

func (a *App) ponerEstado(e EstadoSincro) {
	a.cu.mu.Lock()
	if e.Ultima == "" {
		e.Ultima = a.cu.estado.Ultima
	}
	a.cu.estado = e
	a.cu.mu.Unlock()
	a.sistema.Avisar(EventoSincro, e)
}
