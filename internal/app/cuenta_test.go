package app

import (
	"encoding/json"
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
	rec, err := e.a.TerminarRegistro(correo, codigoDelBuzon(t, raiz, correo), maestra)
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
	if _, err := a.a.TerminarRegistro(correo, codigo, "corta"); err == nil {
		t.Fatal("crea la cuenta con una contraseña débil")
	}
	if _, err := os.Stat(rutaBoveda()); err == nil {
		t.Fatal("un alta que no ha salido deja una bóveda en el disco")
	}
	desde := time.Now()
	rec, err := a.a.TerminarRegistro(correo, codigo, maestraFuerte)
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
	if _, err := e.a.TerminarRegistro(correo, codigo, "otra maestra larga que no es la de esta bóveda"); err == nil {
		t.Fatal("crea la cuenta con una contraseña que no es la de la bóveda")
	}
	desde := time.Now()
	rec, err := e.a.TerminarRegistro(correo, codigo, maestraFuerte)
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

// Con cuenta, cambiar solo la contraseña de la bóveda dejaría a los equipos nuevos
// sin poder entrar. Hasta que se pueda cambiar en los dos sitios a la vez, no se deja.
func TestConCuentaLaMaestraNoSeCambiaTodavia(t *testing.T) {
	raiz := servidorDeCuentas(t)
	e := nuevoEquipo(t, raiz)
	crearCuenta(t, raiz, e, correoDePrueba(), maestraFuerte)
	if err := e.a.CambiarMaestraDeBoveda(maestraFuerte, "otra maestra larga que sería la nueva"); err == nil {
		t.Fatal("con cuenta deja cambiar la contraseña solo en la bóveda")
	}
	e.a.CerrarBoveda()
	if err := e.a.AbrirBoveda(maestraFuerte); err != nil {
		t.Fatalf("la de siempre ya no abre: %v", err)
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
	if _, err := e.a.TerminarRegistro(correo, codigo, "una contraseña larga que no es la de la bóveda"); err == nil {
		t.Fatal("crea la cuenta con una contraseña que no abre la bóveda")
	}
	if _, err := e.a.TerminarRegistro(correo, codigo, maestraFuerte); err != nil {
		t.Fatalf("con la bóveda cerrada no crea la cuenta: %v", err)
	}
	if !e.a.EstadoBoveda().Abierta || titulosDe(t, e.a) != "Ya estaba" {
		t.Fatal("la bóveda no ha quedado abierta con lo suyo")
	}
	alDia(t, e.a, time.Now())
}
