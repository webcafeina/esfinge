package sincro

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cuenta"
)

const maestra = "una contraseña maestra larga y fuerte"

var ctx = context.Background()

// ------------------------------------------------------------------ servidor en memoria
//
// Solo para lo que el de verdad no deja provocar a voluntad: que otro equipo suba
// justo en medio, que el servidor vuelva atrás. **El recorrido normal se prueba
// contra el servidor de verdad** (TestServidor…), no contra éste.

type enMemoria struct {
	mu      sync.Mutex
	datos   []byte
	version int64
	// antesDeSubir corre justo antes de aceptar una subida: ahí se cuela «otro equipo».
	antesDeSubir func()
	// mentir cambia lo que contesta Bajar.
	mentir func(datos []byte, version int64) ([]byte, int64, error)
}

func (m *enMemoria) Bajar(_ context.Context, _ string, siNoCoincide int64) ([]byte, int64, bool, error) {
	m.mu.Lock()
	datos, version := m.datos, m.version
	mentir := m.mentir
	m.mu.Unlock()
	if mentir != nil {
		return mentirYa(mentir, datos, version)
	}
	if version == 0 {
		return nil, 0, false, cuenta.ErrSinBoveda
	}
	if siNoCoincide == version {
		return nil, version, false, nil
	}
	return datos, version, true, nil
}

func mentirYa(f func([]byte, int64) ([]byte, int64, error), d []byte, v int64) ([]byte, int64, bool, error) {
	datos, version, err := f(d, v)
	return datos, version, err == nil, err
}

func (m *enMemoria) Subir(_ context.Context, _ string, siCoincide int64, datos []byte) (int64, error) {
	if m.antesDeSubir != nil {
		hacer := m.antesDeSubir
		m.antesDeSubir = nil
		hacer()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if siCoincide != m.version {
		return 0, &cuenta.ErrorDelServidor{Estado: http.StatusPreconditionFailed, Mensaje: "Ha cambiado", Version: m.version}
	}
	m.datos, m.version = datos, m.version+1
	return m.version, nil
}

// equipo: una bóveda en su carpeta y su sincronizador.
type equipo struct {
	b *boveda.Boveda
	s *Sincronizador
}

func nuevoEquipo(t *testing.T, srv Servidor, b *boveda.Boveda) *equipo {
	return &equipo{b: b, s: &Sincronizador{
		Servidor: srv, Boveda: b, Memoria: JuntoALaBoveda{Ruta: b.Ruta()},
		Token: func() string { return "s1.x.y" },
	}}
}

// otroEquipo copia el fichero de la bóveda a otra carpeta y lo abre: la misma
// bóveda en otro ordenador, antes de haberse sincronizado nunca.
func otroEquipo(t *testing.T, srv Servidor, de *boveda.Boveda) *equipo {
	t.Helper()
	datos, err := os.ReadFile(de.Ruta())
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	if err := os.WriteFile(ruta, datos, 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := boveda.Abrir(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	return nuevoEquipo(t, srv, b)
}

func crear(t *testing.T) *boveda.Boveda {
	t.Helper()
	b, _, err := boveda.Crear(filepath.Join(t.TempDir(), "boveda.esfinge"), maestra)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (e *equipo) sincronizar(t *testing.T) Resultado {
	t.Helper()
	r, err := e.s.Sincronizar(ctx)
	if err != nil {
		t.Fatalf("sincronizar: %v", err)
	}
	return r
}

func titulos(b *boveda.Boveda) string {
	var ts []string
	for _, e := range b.Buscar("") {
		ts = append(ts, e.Titulo)
	}
	return fmt.Sprint(ts)
}

// ------------------------------------------------------------------ las piezas

func TestSiOtroEquipoSubeEnMedioSeVuelveABajar(t *testing.T) {
	srv := &enMemoria{}
	a := nuevoEquipo(t, srv, crear(t))
	a.sincronizar(t)
	b := otroEquipo(t, srv, a.b)
	b.sincronizar(t)

	_ = a.b.Poner(boveda.Entrada{Titulo: "De A"})
	_ = b.b.Poner(boveda.Entrada{Titulo: "De B"})
	// B sube justo cuando A va a subir: la subida de A llega tarde y tiene que
	// bajar lo de B, fundir y volver a subir.
	srv.antesDeSubir = func() { b.sincronizar(t) }
	r := a.sincronizar(t)
	if r.Intentos != 2 || !r.Subio {
		t.Fatalf("no ha vuelto a intentarlo: %+v", r)
	}
	b.sincronizar(t)
	if titulos(a.b) != titulos(b.b) || titulos(a.b) != "[De B De A]" {
		t.Fatalf("A %s, B %s", titulos(a.b), titulos(b.b))
	}
}

func TestUnServidorQueVuelveAtrasNoSeFundeNiSePisa(t *testing.T) {
	srv := &enMemoria{}
	a := nuevoEquipo(t, srv, crear(t))
	_ = a.b.Poner(boveda.Entrada{Titulo: "Uno"})
	a.sincronizar(t)
	vieja := srv.datos
	_ = a.b.Poner(boveda.Entrada{Titulo: "Dos"})
	a.sincronizar(t) // el servidor va por la 2

	// Devuelve la 1 diciendo que es la 1: más vieja que lo ya visto.
	srv.mentir = func([]byte, int64) ([]byte, int64, error) { return vieja, 1, nil }
	if _, err := a.s.Sincronizar(ctx); !errors.Is(err, ErrRetroceso) {
		t.Fatalf("acepta volver a la 1: %v", err)
	}
	// La 1 diciendo que es la 3: lo que lleva sellado dentro no cuadra.
	srv.mentir = func([]byte, int64) ([]byte, int64, error) { return vieja, 3, nil }
	if _, err := a.s.Sincronizar(ctx); !errors.Is(err, boveda.ErrRetroceso) {
		t.Fatalf("acepta la 1 disfrazada de 3: %v", err)
	}
	// Y ninguna, después de haber visto la 2: no se sube encima como si nada.
	srv.mentir = func([]byte, int64) ([]byte, int64, error) { return nil, 0, cuenta.ErrSinBoveda }
	if _, err := a.s.Sincronizar(ctx); !errors.Is(err, ErrRetroceso) {
		t.Fatalf("sube sobre una cuenta que ha perdido la bóveda: %v", err)
	}
	if titulos(a.b) != "[Uno Dos]" {
		t.Fatalf("la bóveda de aquí ha cambiado: %s", titulos(a.b))
	}
}

// conGuardadoEnMedio guarda una entrada justo después de preparar la subida,
// como haría el navegador guardando una contraseña a la vez.
type conGuardadoEnMedio struct {
	*boveda.Boveda
	colar func()
}

func (c conGuardadoEnMedio) PrepararSubida(v int64) ([]byte, int64, error) {
	d, s, err := c.Boveda.PrepararSubida(v)
	if c.colar != nil {
		c.colar()
	}
	return d, s, err
}

// Un guardado que se cuela entre preparar la subida y apuntarla **no se da por
// subido**: la pasada siguiente lo sube. Es la razón de que la serie salga del
// mismo cerrojo que la subida.
func TestUnGuardadoQueSeCuelaNoSeDaPorSubido(t *testing.T) {
	srv := &enMemoria{}
	b := crear(t)
	e := nuevoEquipo(t, srv, b)
	una := true
	e.s.Boveda = conGuardadoEnMedio{Boveda: b, colar: func() {
		if una {
			una = false
			_ = b.Poner(boveda.Entrada{Titulo: "Colada"})
		}
	}}
	e.sincronizar(t)
	r := e.sincronizar(t)
	if !r.Subio {
		t.Fatal("la entrada colada se ha dado por subida y no sube nunca")
	}
	// Se mira lo que hay en el servidor, abriéndolo:
	m, err := boveda.AbrirEnMemoria(srv.datos, maestra)
	if err != nil {
		t.Fatal(err)
	}
	if titulos(m) != "[Colada]" {
		t.Fatalf("en el servidor hay %s", titulos(m))
	}
}

// El turno se reserva: dos pasadas a la vez no suben dos veces sobre la misma
// versión, y la que llega tarde no se pierde.
func TestDosPasadasALaVezNoSePisan(t *testing.T) {
	srv := &enMemoria{}
	e := nuevoEquipo(t, srv, crear(t))
	e.sincronizar(t)
	dentro := make(chan struct{})
	seguir := make(chan struct{})
	srv.antesDeSubir = func() { close(dentro); <-seguir }
	_ = e.b.Poner(boveda.Entrada{Titulo: "Una"})

	var primera error
	hecho := make(chan struct{})
	go func() { _, primera = e.s.Sincronizar(ctx); close(hecho) }()
	<-dentro
	_ = e.b.Poner(boveda.Entrada{Titulo: "Otra"})
	if _, err := e.s.Sincronizar(ctx); !errors.Is(err, ErrOcupado) {
		t.Fatalf("la segunda pasada no espera su turno: %v", err)
	}
	close(seguir)
	<-hecho
	if primera != nil {
		t.Fatal(primera)
	}
	m, _ := boveda.AbrirEnMemoria(srv.datos, maestra)
	if titulos(m) != "[Una Otra]" {
		t.Fatalf("lo pedido mientras tanto no ha subido: %s", titulos(m))
	}
}

func TestVigilarSubeTrasGuardarYParaAlCancelar(t *testing.T) {
	srv := &enMemoria{}
	e := nuevoEquipo(t, srv, crear(t))
	pasadas := make(chan Resultado, 10)
	v := &Vigilante{S: e.s, Espera: 20 * time.Millisecond, Avisar: func(r Resultado, err error) {
		if err != nil {
			t.Error(err)
		}
		pasadas <- r
	}}
	c, cancelar := context.WithCancel(ctx)
	terminado := make(chan struct{})
	go func() { v.Vigilar(c); close(terminado) }()

	<-pasadas // la del arranque sube la bóveda vacía
	_ = e.b.Poner(boveda.Entrada{Titulo: "Guardada"})
	v.Pedir()
	v.Pedir() // dos pedidos seguidos, una pasada
	select {
	case r := <-pasadas:
		if !r.Subio {
			t.Fatalf("tras guardar no sube: %+v", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("tras pedirlo no ha habido pasada")
	}
	cancelar()
	select {
	case <-terminado:
	case <-time.After(5 * time.Second):
		t.Fatal("no para al cancelar")
	}
}

// «Sincronizar ahora» es ahora: sin la espera de después de guardar, que en un
// botón se lee como que no ha hecho nada (2.25.2).
func TestYaNoEsperaLoDeDespuesDeGuardar(t *testing.T) {
	srv := &enMemoria{}
	e := nuevoEquipo(t, srv, crear(t))
	pasadas := make(chan Resultado, 10)
	v := &Vigilante{S: e.s, Espera: 10 * time.Second, Avisar: func(r Resultado, err error) {
		if err != nil {
			t.Error(err)
		}
		pasadas <- r
	}}
	c, cancelar := context.WithCancel(ctx)
	defer cancelar()
	go v.Vigilar(c)
	<-pasadas // la del arranque

	_ = e.b.Poner(boveda.Entrada{Titulo: "Con prisa"})
	v.Pedir() // un guardado: esperaría diez segundos
	v.Ya()    // un botón: no
	select {
	case r := <-pasadas:
		if !r.Subio {
			t.Fatalf("no sube: %+v", r)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("«ya» ha esperado lo de después de guardar")
	}
}

func TestLoQueSeRecuerdaSeGuardaYSeOlvida(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	m := JuntoALaBoveda{Ruta: ruta}
	if r, base, err := m.Cargar(); err != nil || r.Version != 0 || base != nil {
		t.Fatalf("sin nada guardado: %+v %v", r, err)
	}
	if err := m.Guardar(Recuerdo{Version: 7, Serie: 3}, []byte("base")); err != nil {
		t.Fatal(err)
	}
	r, base, _ := m.Cargar()
	if r.Version != 7 || r.Serie != 3 || string(base) != "base" {
		t.Fatalf("%+v %q", r, base)
	}
	for _, f := range []string{".base", ".sincro"} {
		if info, _ := os.Stat(ruta + f); info.Mode().Perm() != 0o600 {
			t.Fatalf("%s con permisos %v", f, info.Mode().Perm())
		}
	}
	if err := m.Olvidar(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ruta + ".base"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("olvidar deja la base")
	}
}

// ------------------------------------------------------------------ la tubería entera
//
// Contra el servidor de verdad, levantado por herramientas/con-servidor.sh: dos
// equipos, una cuenta, y lo que hará Esfinge de principio a fin —crear la
// cuenta, entrar desde otro equipo con su código, trabajar en los dos a la vez y
// acabar igual—. Es la prueba que faltaba en los iconos (CLAUDE.md): las piezas
// sueltas en verde no dicen que estén conectadas.

func TestServidorDosEquiposDeUnaCuenta(t *testing.T) {
	raiz := os.Getenv("ESFINGE_SERVIDOR_PRUEBAS")
	if raiz == "" {
		t.Skip("sin servidor de cuentas: se corren con herramientas/con-servidor.sh (make comprobar)")
	}
	cli := cuenta.Nuevo(raiz)
	correo := fmt.Sprintf("tuberia-%d@ejemplo.com", time.Now().UnixNano())

	// Equipo A: tiene su bóveda y crea la cuenta con ella.
	bA := crear(t)
	_ = bA.Poner(boveda.Entrada{Titulo: "Banco", Usuario: "ana", Secreto: "uno"})
	if err := cli.EmpezarAlta(ctx, correo); err != nil {
		t.Fatal(err)
	}
	sal := bytes.Repeat([]byte{2}, 16)
	clave, err := cuenta.DerivarAcceso(maestra, sal, cuenta.PorDefecto)
	if err != nil {
		t.Fatal(err)
	}
	posesion, err := bA.Posesion()
	if err != nil {
		t.Fatal(err)
	}
	sesA, err := cli.TerminarAlta(ctx, cuenta.Alta{
		Correo: correo, Codigo: codigoDe(t, raiz, correo), Sal: sal, Argon2: cuenta.PorDefecto,
		ClaveDeAcceso: clave, Posesion: posesion, Dispositivo: "Equipo A", Confiar: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	a := &equipo{b: bA, s: &Sincronizador{Servidor: cli, Boveda: bA, Memoria: JuntoALaBoveda{Ruta: bA.Ruta()}, Token: func() string { return sesA.Token }}}
	if r := a.sincronizar(t); !r.Subio || r.Version != 1 {
		t.Fatalf("la primera subida: %+v", r)
	}

	// Equipo B: no tiene nada. Entra con correo, contraseña y código, baja la
	// bóveda, la abre con la contraseña en memoria y la guarda en su carpeta.
	pre, err := cli.Prelogin(ctx, correo)
	if err != nil {
		t.Fatal(err)
	}
	claveB, _ := cuenta.DerivarAcceso(maestra, pre.Sal, pre.Argon2)
	_, reto, err := cli.Entrar(ctx, correo, claveB, "Equipo B", "")
	if err != nil {
		t.Fatal(err)
	}
	sesB, err := cli.ConfirmarEntrada(ctx, reto, codigoDe(t, raiz, correo), true)
	if err != nil {
		t.Fatal(err)
	}
	datos, _, _, err := cli.Bajar(ctx, sesB.Token, 0)
	if err != nil {
		t.Fatal(err)
	}
	bB, err := boveda.AbrirEnMemoria(datos, maestra)
	if err != nil {
		t.Fatalf("B no abre lo que bajó con su contraseña: %v", err)
	}
	if err := bB.GuardarEn(filepath.Join(t.TempDir(), "boveda.esfinge")); err != nil {
		t.Fatal(err)
	}
	b := &equipo{b: bB, s: &Sincronizador{Servidor: cli, Boveda: bB, Memoria: JuntoALaBoveda{Ruta: bB.Ruta()}, Token: func() string { return sesB.Token }}}
	b.sincronizar(t)

	// Los dos trabajan a la vez, en la misma entrada y en otras.
	banco := func(bv *boveda.Boveda) boveda.Entrada {
		for _, e := range bv.Buscar("Banco") {
			x, _ := bv.Ver(e.ID)
			return x
		}
		t.Fatal("no está el banco")
		return boveda.Entrada{}
	}
	ea := banco(bA)
	ea.CambiarSecreto("de A", time.Now())
	_ = bA.Poner(ea)
	eb := banco(bB)
	eb.Notas = "nota de B"
	_ = bB.Poner(eb)
	_ = bA.Poner(boveda.Entrada{Titulo: "Solo en A"})
	_ = bB.Poner(boveda.Entrada{Titulo: "Solo en B"})
	a.sincronizar(t)
	b.sincronizar(t)
	a.sincronizar(t)
	b.sincronizar(t)

	fa, fb := banco(bA), banco(bB)
	if fa.Secreto != "de A" || fa.Notas != "nota de B" || fb.Secreto != "de A" || fb.Notas != "nota de B" {
		t.Fatalf("no se han juntado los cambios de los dos equipos:\nA %+v\nB %+v", fa, fb)
	}
	if titulos(bA) != titulos(bB) {
		t.Fatalf("A %s, B %s", titulos(bA), titulos(bB))
	}
	// Y lo que hay en el servidor es lo mismo, sin que el servidor pueda leerlo:
	// solo se abre con la contraseña.
	final, _, _, _ := cli.Bajar(ctx, sesA.Token, 0)
	m, err := boveda.AbrirEnMemoria(final, maestra)
	if err != nil || titulos(m) != titulos(bA) {
		t.Fatalf("en el servidor: %s, %v", titulos(m), err)
	}
	if bytes.Contains(final, []byte("de A")) || bytes.Contains(final, []byte("Banco")) {
		t.Fatal("la bóveda viaja con algo en claro")
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(final, &doc); err != nil || doc["cuerpo"] == nil {
		t.Fatal("lo del servidor no es una bóveda")
	}
}

func codigoDe(t *testing.T, raiz, correo string) string {
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

// Al cerrar se sube lo pendiente, sin esperar a los tres segundos del vigilante.
func TestVaciarSubeLoPendiente(t *testing.T) {
	srv := &enMemoria{}
	e := nuevoEquipo(t, srv, crear(t))
	e.sincronizar(t)
	if e.s.Pendiente() {
		t.Fatal("recién sincronizada dice que hay algo pendiente")
	}
	_ = e.b.Poner(boveda.Entrada{Titulo: "Antes de cerrar"})
	if !e.s.Pendiente() {
		t.Fatal("con un cambio sin subir dice que no hay nada pendiente")
	}
	if err := e.s.Vaciar(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	m, _ := boveda.AbrirEnMemoria(srv.datos, maestra)
	if titulos(m) != "[Antes de cerrar]" {
		t.Fatalf("en el servidor hay %s", titulos(m))
	}
}

// Una pasada que se encuentra con otra en marcha **no puede dejar al vigilante
// dormido**. Con el `continue` de antes se saltaba el rearme del reloj: no volvía
// a haber pasadas cada minuto y la ventana se quedaba en «Sincronizando…» hasta el
// siguiente guardado. Pasa al abrir la bóveda mientras corre la pasada del
// vigilante anterior —entrar con la contraseña nueva—, y lo cazó la puerta de
// publicación de la 2.25.4, no esta máquina.
func TestElVigilanteSigueDespuesDeEncontrarseOcupado(t *testing.T) {
	srv := &enMemoria{}
	e := nuevoEquipo(t, srv, crear(t))

	// Una pasada de fuera, parada a mitad: el turno está cogido.
	dentro := make(chan struct{})
	seguir := make(chan struct{})
	srv.antesDeSubir = func() { close(dentro); <-seguir }
	_ = e.b.Poner(boveda.Entrada{Titulo: "De la otra pasada"})
	go func() { _, _ = e.s.Sincronizar(ctx) }()
	<-dentro
	srv.antesDeSubir = nil

	pasadas := make(chan Resultado, 10)
	v := &Vigilante{S: e.s, Espera: 20 * time.Millisecond, Avisar: func(r Resultado, _ error) { pasadas <- r }}
	c, cancelar := context.WithCancel(ctx)
	defer cancelar()
	go v.Vigilar(c)

	// La del arranque se encuentra el turno cogido y no cuenta nada.
	select {
	case r := <-pasadas:
		t.Fatalf("ha contado una pasada que no pudo hacer: %+v", r)
	case <-time.After(300 * time.Millisecond):
	}

	// Al soltarse el turno, el vigilante tiene que volver **solo**: nadie le pide
	// nada y la pasada de cada minuto no llega en lo que dura esta prueba.
	close(seguir)
	select {
	case <-pasadas:
	case <-time.After(10 * time.Second):
		t.Fatal("el vigilante se ha quedado dormido tras encontrarse el turno cogido")
	}
}

// **La parada por muchos borrados tiene salida** (revisión del 2026-09-23): un
// permiso para **una sola pasada**, que da una persona viendo lo que va a pasar.
// Antes, la opción existía en el código y no la ponía nadie: la bóveda se quedaba
// sin sincronizar y sin subir lo suyo, y lo único que se leía era que estaba
// parada.
func TestJuntarloIgualValeUnaVezYSoloUna(t *testing.T) {
	srv := &enMemoria{}
	a := nuevoEquipo(t, srv, crear(t))
	for i := range 5 {
		_ = a.b.Poner(boveda.Entrada{Titulo: fmt.Sprint("E", i)})
	}
	a.sincronizar(t)
	b := otroEquipo(t, srv, a.b)
	b.sincronizar(t)

	// A se lleva por delante cuatro de las cinco, del todo.
	for _, e := range a.b.Buscar("") {
		if e.Titulo != "E0" {
			_ = a.b.Borrar(e.ID)
			_ = a.b.BorrarDelTodo(e.ID)
		}
	}
	a.sincronizar(t)

	if _, err := b.s.Sincronizar(ctx); !errors.Is(err, boveda.ErrMuchosBorrados) {
		t.Fatalf("se las lleva sin preguntar: %v", err)
	}
	b.s.UnaVezAunqueBorre()
	if _, err := b.s.Sincronizar(ctx); err != nil {
		t.Fatalf("con el permiso no funde: %v", err)
	}
	if b.b.Cuantas() != 1 {
		t.Fatalf("en B quedan %d", b.b.Cuantas())
	}

	// Y el permiso **se ha gastado**: la vez siguiente vuelve a pararse. B repuebla
	// la bóveda, A se lleva otra vez casi todo, y B se para.
	for i := range 5 {
		_ = b.b.Poner(boveda.Entrada{Titulo: fmt.Sprint("De B ", i)})
	}
	b.sincronizar(t)
	a.sincronizar(t)
	for _, e := range a.b.Buscar("") {
		if e.Titulo != "E0" {
			_ = a.b.Borrar(e.ID)
			_ = a.b.BorrarDelTodo(e.ID)
		}
	}
	a.sincronizar(t)
	if _, err := b.s.Sincronizar(ctx); !errors.Is(err, boveda.ErrMuchosBorrados) {
		t.Fatalf("el permiso se ha quedado puesto: %v", err)
	}
}
