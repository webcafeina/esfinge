package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
)

// Las pruebas de la cuenta hablan con **el servidor de verdad**, levantado en
// local por herramientas/con-servidor.sh (make comprobar). Un `go test` suelto
// se las salta.
//
// Cada «equipo» es una App con su propia carpeta de configuración. Como la
// carpeta sale del entorno (os.UserConfigDir), se cambia de equipo cambiando el
// entorno, y por eso aquí se trabaja **en uno cada vez**: lo que corre de fondo
// —la sincronización— ya sabe su ruta y no vuelve a preguntar.

const maestraFuerte = "una maestra larga para las pruebas de la cuenta"

func servidorDeCuentas(t *testing.T) string {
	t.Helper()
	raiz := os.Getenv("ESFINGE_SERVIDOR_PRUEBAS")
	if raiz == "" {
		t.Skip("sin servidor de cuentas: se corren con herramientas/con-servidor.sh (make comprobar)")
	}
	return raiz
}

// equipoDePrueba es una App en su propia carpeta.
type equipoDePrueba struct {
	a       *App
	s       *sistemaFalso
	carpeta string
}

func (e *equipoDePrueba) usar(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", e.carpeta)
	t.Setenv("HOME", e.carpeta)
	t.Setenv("USERPROFILE", e.carpeta)
	t.Setenv("APPDATA", e.carpeta)
}

func nuevoEquipo(t *testing.T, raiz string) *equipoDePrueba {
	t.Helper()
	e := &equipoDePrueba{s: &sistemaFalso{}, carpeta: t.TempDir()}
	e.usar(t)
	e.a = Nueva("2.23.0", e.s)
	ApuntarCuentasA(e.a, raiz)
	e.a.cu.espera = 50 * time.Millisecond
	t.Cleanup(func() { e.a.alCerrarLaBoveda() })
	return e
}

func codigoDelBuzon(t *testing.T, raiz, correo string) string {
	t.Helper()
	resp, err := http.Get(raiz + "/_pruebas/buzon?correo=" + correo)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r struct {
		Mensajes []struct{ Cuerpo string } `json:"mensajes"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&r)
	if len(r.Mensajes) == 0 {
		t.Fatalf("no ha llegado nada a %s", correo)
	}
	m := regexp.MustCompile(`(?m)^\s+(\d{6})$`).FindStringSubmatch(r.Mensajes[0].Cuerpo)
	if m == nil {
		t.Fatal("el correo no trae código")
	}
	return m[1]
}

// alDia espera a que la sincronización esté al día **con todo lo de aquí subido**.
//
// No basta con mirar la hora de la última pasada: tiene resolución de un segundo,
// y una pasada anterior en el mismo segundo que un guardado la daba por buena sin
// que el guardado hubiera subido. Así se escondió que cerrar no subía lo pendiente.
func alDia(t *testing.T, a *App, _ time.Time) {
	t.Helper()
	limite := time.Now().Add(20 * time.Second)
	for time.Now().Before(limite) {
		e := a.EstadoDeCuenta().Sincro
		a.cu.mu.Lock()
		m := a.cu.marcha
		a.cu.mu.Unlock()
		if e.Estado == "al-dia" && m != nil && !m.s.Pendiente() {
			return
		}
		if e.Estado == "error" || e.Estado == "hay-que-entrar" || e.Estado == "muchos-borrados" {
			t.Fatalf("la sincronización dice %s: %s", e.Estado, e.Mensaje)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("no se ha puesto al día: %+v", a.EstadoDeCuenta().Sincro)
}

func titulosDe(t *testing.T, a *App) string {
	t.Helper()
	l, err := a.BuscarEnBoveda("")
	if err != nil {
		t.Fatal(err)
	}
	var ts []string
	for _, e := range l {
		ts = append(ts, e.Titulo)
	}
	return strings.Join(ts, ", ")
}

func correoDePrueba() string { return fmt.Sprintf("app-%d@ejemplo.com", time.Now().UnixNano()) }

// crearCuenta hace lo que la bienvenida: pedir el código y crear la cuenta.
func crearCuenta(t *testing.T, raiz string, e *equipoDePrueba, correo, maestra string) string {
	t.Helper()
	if err := e.a.EmpezarRegistro(correo); err != nil {
		t.Fatal(err)
	}
	rec, err := e.a.TerminarRegistro(correo, codigoDelBuzon(t, raiz, correo), maestra, "")
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// ------------------------------------------------------------------ las pruebas

// La tubería entera, de la ventana de un equipo a la de otro: crear la cuenta en
// uno, entrar desde otro con el código, trabajar en los dos y que todo llegue.
func TestCuentaDeUnEquipoAOtro(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()

	a := nuevoEquipo(t, raiz)
	if m := a.a.EstadoDeCuenta().Modo; m != "" {
		t.Fatalf("un equipo sin nada tiene que ver la bienvenida, y el modo es %q", m)
	}
	if err := a.a.EmpezarRegistro(correo); err != nil {
		t.Fatal(err)
	}
	codigo := codigoDelBuzon(t, raiz, correo)
	if _, err := a.a.TerminarRegistro(correo, codigo, "corta", ""); err == nil {
		t.Fatal("crea la cuenta con una contraseña débil")
	}
	if _, err := os.Stat(rutaBoveda()); err == nil {
		t.Fatal("un alta que no ha salido deja una bóveda en el disco")
	}
	desde := time.Now()
	rec, err := a.a.TerminarRegistro(correo, codigo, maestraFuerte, "")
	if err != nil {
		t.Fatal(err)
	}
	if rec == "" {
		t.Fatal("una bóveda nueva sin clave de recuperación")
	}
	if e := a.a.EstadoDeCuenta(); e.Modo != "cuenta" || e.Correo != correo {
		t.Fatalf("%+v", e)
	}
	alDia(t, a.a, desde)
	// Guarda y cierra **en el acto**, sin esperar a los segundos del vigilante: lo
	// pendiente tiene que subir al cerrar. Si no, se queda en este equipo.
	if err := a.a.GuardarEnBoveda(boveda.Entrada{Titulo: "Banco", Secreto: "uno"}); err != nil {
		t.Fatal(err)
	}
	a.a.CerrarBoveda()

	// Otro equipo, sin nada: correo, contraseña y el código del correo.
	b := nuevoEquipo(t, raiz)
	r, err := b.a.EntrarEnCuenta(correo, maestraFuerte)
	if err != nil || !r.NecesitaCodigo {
		t.Fatalf("%+v %v", r, err)
	}
	r, err = b.a.ConfirmarEntrada(codigoDelBuzon(t, raiz, correo))
	if err != nil || !r.Listo {
		t.Fatalf("%+v %v", r, err)
	}
	if got := titulosDe(t, b.a); got != "Banco" {
		t.Fatalf("en el equipo nuevo hay %q", got)
	}
	desde = time.Now()
	if err := b.a.GuardarEnBoveda(boveda.Entrada{Titulo: "Desde B"}); err != nil {
		t.Fatal(err)
	}
	alDia(t, b.a, desde)
	b.a.CerrarBoveda()

	// Y el primero, al abrir, se entera.
	a.usar(t)
	desde = time.Now()
	if err := a.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	alDia(t, a.a, desde)
	if got := titulosDe(t, a.a); got != "Banco, Desde B" {
		t.Fatalf("en el primer equipo hay %q", got)
	}
	if !a.s.hanAvisadoDe(EventoBovedaCambiada) {
		t.Fatal("lo que llega de otro equipo no avisa a la ventana para que vuelva a pedir la lista")
	}
}

func TestCuentaConLaBovedaQueYaHabia(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	e := nuevoEquipo(t, raiz)
	if _, err := e.a.CrearBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	if m := e.a.EstadoDeCuenta().Modo; m != "local" {
		t.Fatalf("crear la bóveda aquí es elegir local, y el modo es %q", m)
	}
	_ = e.a.GuardarEnBoveda(boveda.Entrada{Titulo: "Ya estaba"})
	if err := e.a.EmpezarRegistro(correo); err != nil {
		t.Fatal(err)
	}
	codigo := codigoDelBuzon(t, raiz, correo)
	if _, err := e.a.TerminarRegistro(correo, codigo, "otra maestra larga que no es la de esta bóveda", ""); err == nil {
		t.Fatal("crea la cuenta con una contraseña que no es la de la bóveda")
	}
	desde := time.Now()
	rec, err := e.a.TerminarRegistro(correo, codigo, maestraFuerte, "")
	if err != nil {
		t.Fatal(err)
	}
	if rec != "" {
		t.Fatal("con la bóveda de siempre no hay clave de recuperación nueva: la de siempre sigue valiendo")
	}
	alDia(t, e.a, desde)

	// Otro equipo la baja entera.
	e.a.CerrarBoveda()
	otro := nuevoEquipo(t, raiz)
	r, _ := otro.a.EntrarEnCuenta(correo, maestraFuerte)
	if r.NecesitaCodigo {
		r, err = otro.a.ConfirmarEntrada(codigoDelBuzon(t, raiz, correo))
	}
	if err != nil || !r.Listo || titulosDe(t, otro.a) != "Ya estaba" {
		t.Fatalf("%+v %v %q", r, err, titulosDe(t, otro.a))
	}
}

// En un equipo que ya tiene su bóveda, entrar en la cuenta pregunta; juntar trae
// lo de aquí a la cuenta, y la bóveda de aquí **se aparta, no se borra**.
func TestEntrarEnUnEquipoConOtraBovedaYJuntar(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	a := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, a, correo, maestraFuerte)
	_ = a.a.GuardarEnBoveda(boveda.Entrada{Titulo: "De la cuenta"})
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	b := nuevoEquipo(t, raiz)
	const deAqui = "la contraseña de la bóveda de este otro equipo"
	if _, err := b.a.CrearBoveda(deAqui); err != nil {
		t.Fatal(err)
	}
	_ = b.a.GuardarEnBoveda(boveda.Entrada{Titulo: "De este equipo"})
	r, err := b.a.EntrarEnCuenta(correo, maestraFuerte)
	if err == nil && r.NecesitaCodigo {
		r, err = b.a.ConfirmarEntrada(codigoDelBuzon(t, raiz, correo))
	}
	if err != nil || !r.HayOtraBoveda {
		t.Fatalf("no pregunta qué hacer con la bóveda de aquí: %+v %v", r, err)
	}
	if _, err := b.a.ResolverOtraBoveda(true, "no es ésta"); err == nil {
		t.Fatal("junta sin poder abrir la bóveda de aquí")
	}
	desde := time.Now()
	r, err = b.a.ResolverOtraBoveda(true, deAqui)
	if err != nil || !r.Listo || r.Apartada == "" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := os.Stat(r.Apartada); err != nil {
		t.Fatal("la bóveda de aquí no se ha apartado: se ha perdido")
	}
	if _, err := boveda.Abrir(r.Apartada, deAqui); err != nil {
		t.Fatalf("la apartada no se abre con su contraseña: %v", err)
	}
	alDia(t, b.a, desde)
	if got := titulosDe(t, b.a); got != "De la cuenta, De este equipo" {
		t.Fatalf("tras juntar hay %q", got)
	}
	// Y ahora la contraseña de este equipo es la de la cuenta.
	b.a.CerrarBoveda()
	if err := b.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatalf("la de la cuenta no abre la bóveda de aquí: %v", err)
	}
}

// Sincronizar de fondo **no mantiene abierta** una bóveda que nadie usa.
func TestSincronizarNoCuentaComoActividad(t *testing.T) {
	raiz := servidorDeCuentas(t)
	e := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, e, correoDePrueba(), maestraFuerte)
	alDia(t, e.a, time.Now().Add(-time.Minute))

	antes := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e.a.vig.mu.Lock()
	e.a.vig.ultimaActividad = antes
	e.a.vig.mu.Unlock()
	for range 3 {
		desde := time.Now()
		if err := e.a.SincronizarAhora(); err != nil {
			t.Fatal(err)
		}
		alDia(t, e.a, desde)
	}
	e.a.vig.mu.Lock()
	despues := e.a.vig.ultimaActividad
	e.a.vig.mu.Unlock()
	if !despues.Equal(antes) {
		t.Fatal("sincronizar ha contado como actividad: la bóveda no se cerraría nunca")
	}
}

// Borrar la bóveda se lleva también lo de la sincronización —la base es una copia
// entera— y la cuenta de este equipo.
func TestBorrarLaBovedaSeLlevaLaBaseYLaCuenta(t *testing.T) {
	raiz := servidorDeCuentas(t)
	e := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, e, correoDePrueba(), maestraFuerte)
	alDia(t, e.a, time.Now().Add(-time.Minute))
	ruta := rutaBoveda()
	for _, f := range []string{ruta + ".base", ruta + ".sincro", rutaCuenta()} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("antes de borrar falta %s", filepath.Base(f))
		}
	}
	if err := e.a.BorrarBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{ruta + ".base", ruta + ".sincro", rutaCuenta()} {
		if _, err := os.Stat(f); err == nil {
			t.Fatalf("borrar la bóveda deja %s", filepath.Base(f))
		}
	}
	if m := e.a.EstadoDeCuenta().Modo; m != "" {
		t.Fatalf("tras borrarlo todo vuelve la bienvenida, y el modo es %q", m)
	}
}

// La sesión va sellada con la clave de la bóveda: en el disco no está en claro.
func TestLaSesionNoQuedaEnClaroEnElDisco(t *testing.T) {
	raiz := servidorDeCuentas(t)
	e := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, e, correoDePrueba(), maestraFuerte)
	crudo, err := os.ReadFile(rutaCuenta())
	if err != nil {
		t.Fatal(err)
	}
	e.a.cu.mu.Lock()
	sesion := e.a.cu.sesion
	e.a.cu.mu.Unlock()
	if sesion == "" || strings.Contains(string(crudo), strings.Split(sesion, ".")[2]) {
		t.Fatal("la sesión está en claro en cuenta.json")
	}
	if info, _ := os.Stat(rutaCuenta()); info.Mode().Perm() != 0o600 {
		t.Fatalf("cuenta.json con permisos %v", info.Mode().Perm())
	}
	// Y al cerrar la bóveda se olvida la de memoria: cerrada no sincroniza.
	e.a.CerrarBoveda()
	e.a.cu.mu.Lock()
	sesion = e.a.cu.sesion
	e.a.cu.mu.Unlock()
	if sesion != "" {
		t.Fatal("con la bóveda cerrada sigue la sesión en memoria")
	}
}

// Dejar la cuenta en este equipo: la bóveda se queda como está, deja de
// sincronizarse, y la cuenta sigue viva para los demás equipos.
func TestSalirDeLaCuentaEnEsteEquipo(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	e := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, e, correo, maestraFuerte)
	_ = e.a.GuardarEnBoveda(boveda.Entrada{Titulo: "Se queda aquí"})
	alDia(t, e.a, time.Now())
	e.a.cu.mu.Lock()
	sesion := e.a.cu.sesion
	e.a.cu.mu.Unlock()

	if err := e.a.SalirDeCuenta("no es ésta"); err == nil {
		t.Fatal("sale de la cuenta sin la contraseña")
	}
	if err := e.a.SalirDeCuenta(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	if m := e.a.EstadoDeCuenta().Modo; m != "local" {
		t.Fatalf("tras salir, el modo es %q", m)
	}
	if got := titulosDe(t, e.a); got != "Se queda aquí" {
		t.Fatalf("la bóveda de aquí ha cambiado: %q", got)
	}
	if _, err := os.Stat(rutaBoveda() + ".base"); err == nil {
		t.Fatal("se queda la base de la sincronización")
	}
	if err := e.a.SincronizarAhora(); err == nil {
		t.Fatal("sigue sincronizando")
	}
	// La sesión de este equipo ya no vale en el servidor.
	if _, err := e.a.cliente().Equipos(e.a.ctxCuenta(), sesion); err == nil {
		t.Fatal("la sesión sigue viva en el servidor")
	}
	// Y la cuenta sigue: se puede volver a entrar desde otro equipo.
	otro := nuevoEquipo(t, raiz)
	r, err := otro.a.EntrarEnCuenta(correo, maestraFuerte)
	if err == nil && r.NecesitaCodigo {
		r, err = otro.a.ConfirmarEntrada(codigoDelBuzon(t, raiz, correo))
	}
	if err != nil || !r.Listo {
		t.Fatalf("la cuenta ya no existe: %+v %v", r, err)
	}
}

// Con la bóveda cerrada, crear la cuenta la abre con la contraseña que se acaba de
// escribir: pedir que se abra antes obligaba a salir del asistente y empezar otra vez.
func TestCrearLaCuentaConLaBovedaCerrada(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	e := nuevoEquipo(t, raiz)
	if _, err := e.a.CrearBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	_ = e.a.GuardarEnBoveda(boveda.Entrada{Titulo: "Ya estaba"})
	e.a.CerrarBoveda()

	if err := e.a.EmpezarRegistro(correo); err != nil {
		t.Fatal(err)
	}
	codigo := codigoDelBuzon(t, raiz, correo)
	if _, err := e.a.TerminarRegistro(correo, codigo, "una contraseña larga que no es la de la bóveda", ""); err == nil {
		t.Fatal("crea la cuenta con una contraseña que no abre la bóveda")
	}
	if _, err := e.a.TerminarRegistro(correo, codigo, maestraFuerte, ""); err != nil {
		t.Fatalf("con la bóveda cerrada no crea la cuenta: %v", err)
	}
	if !e.a.EstadoBoveda().Abierta || titulosDe(t, e.a) != "Ya estaba" {
		t.Fatal("la bóveda no ha quedado abierta con lo suyo")
	}
	alDia(t, e.a, time.Now())
}

// Una bóveda con una contraseña que no llega a «Buena»: sin nueva, no se crea la
// cuenta; con nueva, se crea, la bóveda pasa a abrirse con la nueva, y otro equipo
// entra con ella.
func TestCrearLaCuentaCambiandoUnaMaestraFloja(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	const floja = "contrasena1"
	e := nuevoEquipo(t, raiz)
	if _, err := e.a.CrearBoveda(floja); err != nil {
		t.Fatal(err)
	}
	_ = e.a.GuardarEnBoveda(boveda.Entrada{Titulo: "Ya estaba"})
	e.a.CerrarBoveda()
	if err := e.a.EmpezarRegistro(correo); err != nil {
		t.Fatal(err)
	}
	codigo := codigoDelBuzon(t, raiz, correo)
	if _, err := e.a.TerminarRegistro(correo, codigo, floja, ""); err == nil || !strings.Contains(err.Error(), "pon una nueva") {
		t.Fatalf("con una maestra floja y sin nueva: %v", err)
	}
	if _, err := e.a.TerminarRegistro(correo, codigo, floja, "corta"); err == nil {
		t.Fatal("acepta una nueva que tampoco es buena")
	}
	if _, err := e.a.TerminarRegistro(correo, codigo, floja, maestraFuerte); err != nil {
		t.Fatal(err)
	}
	alDia(t, e.a, time.Now())
	e.a.CerrarBoveda()
	if err := e.a.AbrirBoveda(floja); err == nil {
		t.Fatal("la contraseña de antes sigue abriendo la bóveda")
	}
	if err := e.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatalf("la nueva no abre la bóveda: %v", err)
	}
	e.a.CerrarBoveda()

	otro := nuevoEquipo(t, raiz)
	r, err := otro.a.EntrarEnCuenta(correo, maestraFuerte)
	if err == nil && r.NecesitaCodigo {
		r, err = otro.a.ConfirmarEntrada(codigoDelBuzon(t, raiz, correo))
	}
	if err != nil || !r.Listo || titulosDe(t, otro.a) != "Ya estaba" {
		t.Fatalf("otro equipo no entra con la nueva: %+v %v", r, err)
	}
}

// entrarDesde hace que un equipo entre en la cuenta, código incluido.
func entrarDesde(t *testing.T, raiz string, e *equipoDePrueba, correo, maestra string) ResultadoEntrada {
	t.Helper()
	e.usar(t)
	r, err := e.a.EntrarEnCuenta(correo, maestra)
	if err == nil && r.NecesitaCodigo {
		r, err = e.a.ConfirmarEntrada(codigoDelBuzon(t, raiz, correo))
	}
	if err != nil {
		t.Fatalf("entrar: %v", err)
	}
	return r
}

// Cambiar la contraseña con cuenta: cambia en el servidor y aquí a la vez. El otro
// equipo, con la de antes, entra con la nueva y **no pierde lo que tenía sin
// subir**; y un equipo nuevo entra con la nueva.
func TestCambiarLaContrasenaDeLaCuenta(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	const nueva = "la contraseña nueva de la cuenta, larga y buena"

	b := nuevoEquipo(t, raiz)
	a := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, a, correo, maestraFuerte)
	_ = a.a.GuardarEnBoveda(boveda.Entrada{Titulo: "De A"})
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	entrarDesde(t, raiz, b, correo, maestraFuerte)
	alDia(t, b.a, time.Now())
	b.a.CerrarBoveda()

	// A cambia la contraseña.
	a.usar(t)
	if err := a.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	if err := a.a.CambiarMaestraDeBoveda(maestraFuerte, "corta"); err == nil {
		t.Fatal("con cuenta acepta una contraseña que no llega a «Buena»")
	}
	if err := a.a.CambiarMaestraDeBoveda("no es ésta", nueva); err == nil {
		t.Fatal("cambia sin la contraseña de ahora")
	}
	if err := a.a.CambiarMaestraDeBoveda(maestraFuerte, nueva); err != nil {
		t.Fatal(err)
	}
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()
	if err := a.a.AbrirBoveda(maestraFuerte); err == nil {
		t.Fatal("la de antes sigue abriendo la bóveda en A")
	}
	if err := a.a.AbrirBoveda(nueva); err != nil {
		t.Fatalf("la nueva no abre la bóveda en A: %v", err)
	}
	a.a.CerrarBoveda()

	// B tiene la de antes: abre con ella, guarda algo sin poder subirlo —su sesión
	// ya no vale—, y al entrar con la nueva se pone al día sin perderlo.
	b.usar(t)
	if err := b.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	_ = b.a.GuardarEnBoveda(boveda.Entrada{Titulo: "De B sin subir"})
	b.a.CerrarBoveda()
	if r := entrarDesde(t, raiz, b, correo, nueva); !r.Listo || r.Apartada != "" {
		t.Fatalf("B no entra con la nueva, o aparta su bóveda: %+v", r)
	}
	alDia(t, b.a, time.Now())
	if got := titulosDe(t, b.a); got != "De A, De B sin subir" {
		t.Fatalf("en B hay %q", got)
	}
	b.a.CerrarBoveda()
	if err := b.a.AbrirBoveda(nueva); err != nil {
		t.Fatalf("en B la nueva no abre la bóveda: %v", err)
	}
	b.a.CerrarBoveda()

	// Y un equipo nuevo entra con la nueva, y no con la vieja.
	c := nuevoEquipo(t, raiz)
	if _, err := c.a.EntrarEnCuenta(correo, maestraFuerte); err == nil {
		t.Fatal("la contraseña de antes entra en la cuenta")
	}
	entrarDesde(t, raiz, c, correo, nueva)
	if got := titulosDe(t, c.a); got != "De A, De B sin subir" {
		t.Fatalf("en el equipo nuevo hay %q", got)
	}
}

// El otro equipo, cerrado, se abre con solo escribir la contraseña nueva en
// «Abrir la bóveda»: es de confianza, así que entra en la cuenta sin código, se
// pone al día y no pierde lo que tuviera sin subir. Lo pidió el cliente con la
// 2.24.0: «si la cambio en un ordenador, ¿por qué tengo que indicarlo en el otro?».
func TestElOtroEquipoAbreConLaContrasenaNueva(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	const nueva = "la contraseña nueva de la cuenta, larga y buena"

	a := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, a, correo, maestraFuerte)
	_ = a.a.GuardarEnBoveda(boveda.Entrada{Titulo: "De A"})
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	b := nuevoEquipo(t, raiz)
	entrarDesde(t, raiz, b, correo, maestraFuerte)
	alDia(t, b.a, time.Now())
	b.a.CerrarBoveda()
	if d := leerDatosCuenta(); d.Confianza == "" || strings.HasPrefix(d.Confianza, "ESF1.") {
		t.Fatalf("el testigo de confianza no queda en claro: %q", d.Confianza)
	}

	a.usar(t)
	if err := a.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	if err := a.a.CambiarMaestraDeBoveda(maestraFuerte, nueva); err != nil {
		t.Fatal(err)
	}
	_ = a.a.GuardarEnBoveda(boveda.Entrada{Titulo: "De A con la nueva"})
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	// B, con la de antes, todavía abre su copia y guarda algo que no puede subir:
	// su sesión ya no vale.
	b.usar(t)
	if err := b.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	_ = b.a.GuardarEnBoveda(boveda.Entrada{Titulo: "De B sin subir"})
	b.a.CerrarBoveda()

	if err := b.a.AbrirBoveda("una que no es ninguna de las dos"); !errors.Is(err, boveda.ErrSinRanura) {
		t.Fatalf("una contraseña mala: quiero ErrSinRanura, tengo %v", err)
	}
	if err := b.a.AbrirBoveda(nueva); err != nil {
		t.Fatalf("B no abre con la nueva: %v", err)
	}
	alDia(t, b.a, time.Now())
	if got := titulosDe(t, b.a); got != "De A, De A con la nueva, De B sin subir" {
		t.Fatalf("en B hay %q", got)
	}
	if e := b.a.EstadoDeCuenta(); e.Modo != "cuenta" {
		t.Fatalf("B ha salido de la cuenta: %+v", e)
	}
	b.a.CerrarBoveda()
	if err := b.a.AbrirBoveda(maestraFuerte); err == nil {
		t.Fatal("en B la de antes sigue abriendo")
	}
	if err := b.a.AbrirBoveda(nueva); err != nil {
		t.Fatalf("en B la nueva ya no abre a la segunda: %v", err)
	}
	b.a.CerrarBoveda()

	// Y lo de B ha llegado a A.
	a.usar(t)
	if err := a.a.AbrirBoveda(nueva); err != nil {
		t.Fatal(err)
	}
	alDia(t, a.a, time.Now())
	if got := titulosDe(t, a.a); got != "De A, De A con la nueva, De B sin subir" {
		t.Fatalf("en A hay %q", got)
	}
}

// Sin testigo de confianza —caducado, o sellado de antes de la 2.24.1, que es lo
// que le pasó al cliente con su segundo Mac— la contraseña nueva **no se toma por
// mala**: se reconoce y se pide el código del correo, y con él se abre.
func TestLaContrasenaNuevaSinTestigoPideElCodigo(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	const nueva = "la contraseña nueva de la cuenta, larga y buena"

	a := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, a, correo, maestraFuerte)
	_ = a.a.GuardarEnBoveda(boveda.Entrada{Titulo: "De A"})
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	b := nuevoEquipo(t, raiz)
	entrarDesde(t, raiz, b, correo, maestraFuerte)
	alDia(t, b.a, time.Now())
	b.a.CerrarBoveda()
	d := leerDatosCuenta()
	d.Confianza = "ESF1.sellado-con-la-clave-de-la-boveda"
	if err := guardarDatosCuenta(d); err != nil {
		t.Fatal(err)
	}

	a.usar(t)
	if err := a.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	if err := a.a.CambiarMaestraDeBoveda(maestraFuerte, nueva); err != nil {
		t.Fatal(err)
	}
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	b.usar(t)
	if err := b.a.AbrirBoveda("una que no es ninguna de las dos"); !errors.Is(err, boveda.ErrSinRanura) {
		t.Fatalf("una contraseña mala: quiero ErrSinRanura, tengo %v", err)
	}
	if b.a.EstadoDeCuenta().CodigoPendiente {
		t.Fatal("una contraseña mala deja un código pendiente")
	}
	if err := b.a.AbrirBoveda(nueva); !errors.Is(err, ErrFaltaElCodigo) {
		t.Fatalf("con la nueva y sin testigo: quiero ErrFaltaElCodigo, tengo %v", err)
	}
	if !b.a.EstadoDeCuenta().CodigoPendiente {
		t.Fatal("la ventana no sabe que falta el código")
	}
	r, err := b.a.ConfirmarEntrada(codigoDelBuzon(t, raiz, correo))
	if err != nil || !r.Listo || r.Apartada != "" {
		t.Fatalf("con el código: %+v, %v", r, err)
	}
	alDia(t, b.a, time.Now())
	if got := titulosDe(t, b.a); got != "De A" {
		t.Fatalf("en B hay %q", got)
	}
	if b.a.EstadoDeCuenta().CodigoPendiente {
		t.Fatal("queda un código pendiente después de entrar")
	}
	b.a.CerrarBoveda()
	if err := b.a.AbrirBoveda(nueva); err != nil {
		t.Fatalf("la nueva no abre a la segunda: %v", err)
	}
}

// El camino que siguió el cliente con la 2.24.1: cambiada en A, en B abre con la
// de antes, ve «Vuelve a entrar» y entra con la nueva **con la bóveda abierta**.
func TestVolverAEntrarConLaNuevaConLaBovedaAbierta(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	const nueva = "la contraseña nueva de la cuenta, larga y buena"

	a := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, a, correo, maestraFuerte)
	for _, t2 := range []string{"Uno", "Dos", "Tres"} {
		_ = a.a.GuardarEnBoveda(boveda.Entrada{Titulo: t2, Usuario: "yo", Secreto: t2})
	}
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	b := nuevoEquipo(t, raiz)
	entrarDesde(t, raiz, b, correo, maestraFuerte)
	alDia(t, b.a, time.Now())
	b.a.CerrarBoveda()

	a.usar(t)
	if err := a.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	if err := a.a.CambiarMaestraDeBoveda(maestraFuerte, nueva); err != nil {
		t.Fatal(err)
	}
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	b.usar(t)
	if err := b.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	_ = b.a.SincronizarAhora()
	time.Sleep(500 * time.Millisecond)
	t.Logf("estado en B: %+v", b.a.EstadoDeCuenta().Sincro)
	r := entrarDesde(t, raiz, b, correo, nueva)
	t.Logf("entrar: %+v", r)
	alDia(t, b.a, time.Now())
	t.Logf("en B: %q", titulosDe(t, b.a))
	b.a.CerrarBoveda()
	if err := b.a.AbrirBoveda(nueva); err != nil {
		t.Logf("en B la nueva no abre: %v", err)
		_ = b.a.AbrirBoveda(maestraFuerte)
	}
	t.Logf("en B al reabrir: %q", titulosDe(t, b.a))
	alDia(t, b.a, time.Now())
	b.a.CerrarBoveda()

	a.usar(t)
	if err := a.a.AbrirBoveda(nueva); err != nil {
		t.Fatal(err)
	}
	alDia(t, a.a, time.Now())
	if got := titulosDe(t, a.a); got != "Dos, Tres, Uno" && got != "Uno, Dos, Tres" {
		t.Fatalf("en A hay %q", got)
	}
}

// Recuperar la cuenta sin ningún equipo a mano: código, clave de recuperación y
// contraseña nueva, en un equipo vacío.
func TestRecuperarLaCuentaSinNingunEquipo(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	const nueva = "la contraseña de después de recuperar la cuenta"
	a := nuevoEquipo(t, raiz)
	rec := crearCuenta(t, raiz, a, correo, maestraFuerte)
	_ = a.a.GuardarEnBoveda(boveda.Entrada{Titulo: "Lo que había"})
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()

	d := nuevoEquipo(t, raiz)
	if err := d.a.EmpezarRecuperacion(correo); err != nil {
		t.Fatal(err)
	}
	codigo := codigoDelBuzon(t, raiz, correo)
	if _, err := d.a.TerminarRecuperacion(correo, codigo, rec, "corta"); err == nil {
		t.Fatal("recupera con una contraseña nueva floja")
	}
	r, err := d.a.TerminarRecuperacion(correo, codigo, rec, nueva)
	if err != nil || !r.Listo {
		t.Fatalf("%+v %v", r, err)
	}
	if got := titulosDe(t, d.a); got != "Lo que había" {
		t.Fatalf("tras recuperar hay %q", got)
	}
	alDia(t, d.a, time.Now())
	d.a.CerrarBoveda()
	if err := d.a.AbrirBoveda(nueva); err != nil {
		t.Fatalf("la nueva no abre: %v", err)
	}
	d.a.CerrarBoveda()
	// Con otra clave de recuperación, no.
	if err := d.a.EmpezarRecuperacion(correo); err != nil {
		t.Fatal(err)
	}
	if _, err := d.a.TerminarRecuperacion(correo, codigoDelBuzon(t, raiz, correo), "ESF-0000-0000-0000-0000-0000-0000-0000-0000-0000-0000-0000-0000-0000", nueva); err == nil {
		t.Fatal("recupera con una clave de recuperación que no es la suya")
	}
	// Y el equipo de antes entra con la nueva.
	if r := entrarDesde(t, raiz, a, correo, nueva); !r.Listo {
		t.Fatalf("%+v", r)
	}
}

func TestLosEquiposSeVenYSeOlvidan(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	b := nuevoEquipo(t, raiz)
	a := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, a, correo, maestraFuerte)
	alDia(t, a.a, time.Now())
	a.a.CerrarBoveda()
	entrarDesde(t, raiz, b, correo, maestraFuerte)
	es, err := b.a.DispositivosDeCuenta()
	if err != nil || len(es) != 2 {
		t.Fatalf("%+v %v", es, err)
	}
	var otro string
	actuales := 0
	for _, e := range es {
		if e.Actual {
			actuales++
		} else {
			otro = e.ID
		}
	}
	if actuales != 1 || otro == "" {
		t.Fatalf("no se marca bien el de ahora: %+v", es)
	}
	if err := b.a.OlvidarDispositivo(otro); err != nil {
		t.Fatal(err)
	}
	if es, _ := b.a.DispositivosDeCuenta(); len(es) != 1 {
		t.Fatalf("sigue el olvidado: %+v", es)
	}
	// A, al abrir, se entera de que tiene que volver a entrar.
	a.usar(t)
	if err := a.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatal(err)
	}
	limite := time.Now().Add(20 * time.Second)
	for a.a.EstadoDeCuenta().Sincro.Estado != "hay-que-entrar" {
		if time.Now().After(limite) {
			t.Fatalf("el equipo olvidado sigue: %+v", a.a.EstadoDeCuenta().Sincro)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestExportarYBorrarLaCuenta(t *testing.T) {
	raiz := servidorDeCuentas(t)
	correo := correoDePrueba()
	e := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, e, correo, maestraFuerte)
	_ = e.a.GuardarEnBoveda(boveda.Entrada{Titulo: "Se queda aquí", Secreto: "s3cr3t0"})
	alDia(t, e.a, time.Now())

	e.s.guardaEn = filepath.Join(t.TempDir(), "esfinge-cuenta.json")
	donde, err := e.a.ExportarDatosDeCuenta()
	if err != nil || donde != e.s.guardaEn {
		t.Fatalf("%q %v", donde, err)
	}
	crudo, _ := os.ReadFile(donde)
	if !strings.Contains(string(crudo), correo) || strings.Contains(string(crudo), "s3cr3t0") || strings.Contains(string(crudo), "Se queda aquí") {
		t.Fatal("la exportación no trae la cuenta, o trae algo en claro")
	}

	if err := e.a.BorrarCuenta(maestraFuerte, "123456"); err == nil {
		t.Fatal("borra sin haber pedido el código")
	}
	if err := e.a.PedirCodigoParaBorrarCuenta(); err != nil {
		t.Fatal(err)
	}
	codigo := codigoDelBuzon(t, raiz, correo)
	if err := e.a.BorrarCuenta("no es ésta", codigo); err == nil {
		t.Fatal("borra con otra contraseña")
	}
	if err := e.a.BorrarCuenta(maestraFuerte, codigo); err != nil {
		t.Fatal(err)
	}
	if m := e.a.EstadoDeCuenta().Modo; m != "local" {
		t.Fatalf("tras borrar la cuenta, el modo es %q", m)
	}
	if got := titulosDe(t, e.a); got != "Se queda aquí" {
		t.Fatalf("la bóveda de aquí ha cambiado: %q", got)
	}
	if _, err := e.a.cliente().Prelogin(e.a.ctxCuenta(), correo); err != nil {
		t.Fatal(err)
	}
	otro := nuevoEquipo(t, raiz)
	if _, err := otro.a.EntrarEnCuenta(correo, maestraFuerte); err == nil {
		t.Fatal("la cuenta borrada deja entrar")
	}
}
