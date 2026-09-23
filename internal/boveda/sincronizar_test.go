package boveda

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ------------------------------------------------------------------ el banco
//
// Un «equipo» es una bóveda en su propia carpeta más lo que recuerda del
// servidor: la última versión que vio y sus bytes (la base). El servidor falso
// es solo eso, bytes y versión: lo que hace el de verdad con `If-Match`, sin red.
// Las pruebas contra el servidor de verdad están en internal/sincro.

type servidorFalso struct {
	datos   []byte
	version int64
}

type equipo struct {
	nombre  string
	b       *Boveda
	base    []byte
	version int64
	serie   int64 // la serie del fichero tras la última sincronización
}

// clonar hace «otro equipo» con la misma bóveda, sin volver a derivar la
// contraseña: copia el fichero a otra carpeta y la bóveda abierta en memoria.
func clonar(t *testing.T, b *Boveda, nombre string) *equipo {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	datos, err := os.ReadFile(b.ruta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, datos, 0o600); err != nil {
		t.Fatal(err)
	}
	var cont contenido
	crudo, _ := json.Marshal(b.cont)
	if err := json.Unmarshal(crudo, &cont); err != nil {
		t.Fatal(err)
	}
	otra := &Boveda{ruta: ruta, doc: b.doc, sel: b.sel, cont: cont, llave: append([]byte(nil), b.llave...)}
	otra.doc.Sobres = append([]sobre(nil), b.doc.Sobres...)
	return &equipo{nombre: nombre, b: otra, serie: -1}
}

// sincronizar hace lo que hará internal/sincro, en pequeño: bajar, fundir y, si
// queda algo que el servidor no tiene, subir sobre la versión que se bajó.
func (e *equipo) sincronizar(t *testing.T, s *servidorFalso) (subio bool) {
	t.Helper()
	if s.version == 0 {
		e.subir(t, s)
		return true
	}
	if s.version == e.version {
		if e.b.Serie() == e.serie {
			return false
		}
		e.subir(t, s)
		return true
	}
	f, err := e.b.Fundir(s.datos, s.version, e.base, OpcionesDeFusion{AunqueBorreMucho: true})
	if err != nil {
		t.Fatalf("%s: fundir la %d: %v", e.nombre, s.version, err)
	}
	e.base, e.version, e.serie = s.datos, s.version, f.Serie
	if f.Subir {
		e.subir(t, s)
		return true
	}
	return false
}

func (e *equipo) subir(t *testing.T, s *servidorFalso) {
	t.Helper()
	if s.version != e.version {
		t.Fatalf("%s sube sobre la %d y el servidor va por la %d", e.nombre, e.version, s.version)
	}
	datos, serie, err := e.b.PrepararSubida(s.version + 1)
	if err != nil {
		t.Fatal(err)
	}
	s.datos, s.version = datos, s.version+1
	e.base, e.version, e.serie = datos, s.version, serie
}

func buscar(t *testing.T, b *Boveda, titulo string) Entrada {
	t.Helper()
	for _, e := range b.cont.Entradas {
		if e.Titulo == titulo {
			return e
		}
	}
	t.Fatalf("no está «%s»", titulo)
	return Entrada{}
}

func hay(b *Boveda, titulo string) bool {
	for _, e := range b.cont.Entradas {
		if e.Titulo == titulo {
			return true
		}
	}
	return false
}

// dosEquipos: una bóveda con una entrada, ya subida, y un segundo equipo al día.
func dosEquipos(t *testing.T) (*equipo, *equipo, *servidorFalso) {
	t.Helper()
	b, _, _ := nueva(t)
	if err := b.Poner(Entrada{Titulo: "Banco", Usuario: "ana", Secreto: "uno", Notas: "nota"}); err != nil {
		t.Fatal(err)
	}
	s := &servidorFalso{}
	a := &equipo{nombre: "A", b: b, serie: -1}
	a.sincronizar(t, s)
	otro := clonar(t, b, "B")
	otro.sincronizar(t, s)
	return a, otro, s
}

func mustPoner(t *testing.T, b *Boveda, e Entrada) {
	t.Helper()
	if err := b.Poner(e); err != nil {
		t.Fatal(err)
	}
}

// ------------------------------------------------------------------ lo nuevo del modelo

func TestLaRevisionLaPoneLaBoveda(t *testing.T) {
	b, _, _ := nueva(t)
	mustPoner(t, b, Entrada{Titulo: "X", Revision: 99})
	e := buscar(t, b, "X")
	if e.Revision != 1 {
		t.Fatalf("una entrada nueva empieza en 1, no en %d", e.Revision)
	}
	e.Revision = 50 // una copia vieja o inventada que llega de la ventana
	e.Notas = "otra"
	mustPoner(t, b, e)
	if r := buscar(t, b, "X").Revision; r != 2 {
		t.Fatalf("al editar sube de uno en uno: %d", r)
	}
	if err := b.Borrar(e.ID); err != nil {
		t.Fatal(err)
	}
	if err := b.Restaurar(e.ID); err != nil {
		t.Fatal(err)
	}
	if r := buscar(t, b, "X").Revision; r != 4 {
		t.Fatalf("mandar a la papelera y sacar también son cambios: %d", r)
	}
}

func TestBorrarDelTodoDejaLapidaYLaLapidaCaduca(t *testing.T) {
	b, _, ruta := nueva(t)
	mustPoner(t, b, Entrada{Titulo: "X"})
	id := buscar(t, b, "X").ID
	if err := b.Borrar(id); err != nil {
		t.Fatal(err)
	}
	if err := b.BorrarDelTodo(id); err != nil {
		t.Fatal(err)
	}
	if b.cont.Lapidas[id] == "" {
		t.Fatal("borrar del todo no deja lápida")
	}
	// Vaciar la papelera cuenta entradas, no lápidas.
	mustPoner(t, b, Entrada{Titulo: "Y"})
	idY := buscar(t, b, "Y").ID
	_ = b.Borrar(idY)
	n, err := b.VaciarPapelera()
	if err != nil || n != 1 {
		t.Fatalf("vaciar dice %d, %v", n, err)
	}
	if b.cont.Lapidas[idY] == "" {
		t.Fatal("vaciar la papelera no deja lápida")
	}
	// A los seis meses y un día, las lápidas se van al abrir.
	conReloj(t, func() time.Time { return time.Now().Add(PlazoLapidas + 24*time.Hour) }, func() {
		otra, err := Abrir(ruta, maestra)
		if err != nil {
			t.Fatal(err)
		}
		if len(otra.cont.Lapidas) != 0 {
			t.Fatalf("siguen %d lápidas caducadas", len(otra.cont.Lapidas))
		}
	})
}

func TestAbrirEnMemoriaNoEscribeNada(t *testing.T) {
	b, _, ruta := nueva(t)
	mustPoner(t, b, Entrada{Titulo: "X"})
	_ = b.Borrar(buscar(t, b, "X").ID)
	datos, _ := os.ReadFile(ruta)
	_ = os.Remove(ruta)
	// Con la papelera caducada, abrir del disco la vaciaría y guardaría; en memoria, no.
	conReloj(t, func() time.Time { return time.Now().Add(PlazoPapelera + 24*time.Hour) }, func() {
		m, err := AbrirEnMemoria(datos, maestra)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(ruta); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("abrir en memoria ha escrito el fichero")
		}
		if err := m.Poner(Entrada{Titulo: "Y"}); !errors.Is(err, errSinFichero) {
			t.Fatalf("sin fichero no se puede guardar, y dice: %v", err)
		}
		otra := filepath.Join(t.TempDir(), "boveda.esfinge")
		if err := m.GuardarEn(otra); err != nil {
			t.Fatal(err)
		}
		if _, err := Abrir(otra, maestra); err != nil {
			t.Fatalf("lo guardado con GuardarEn no se abre: %v", err)
		}
	})
}

func TestLaSubidaNoLlevaLasRanurasDeEsteEquipo(t *testing.T) {
	b, _, _ := nueva(t)
	// Una ranura de PIN, que solo tiene sentido en este equipo.
	pin, err := envolver("pin", "1234", b.llave, "2026-09-18T10:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	b.doc.Sobres = append(b.doc.Sobres, pin)
	if err := b.Guardar(); err != nil {
		t.Fatal(err)
	}
	subida, _, err := b.PrepararSubida(7)
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := leerDocumento(subida)
	for _, s := range doc.Sobres {
		if s.Tipo == "pin" {
			t.Fatal("la ranura del PIN va al servidor")
		}
	}
	// Y lo subido cuadra por dentro: el otro equipo lo abre y funde sin ErrManipulada.
	otro := clonar(t, b, "B")
	otro.b.doc.Sobres = sinLocales(otro.b.doc.Sobres)
	if _, err := otro.b.Fundir(subida, 7, nil, OpcionesDeFusion{}); err != nil {
		t.Fatalf("lo subido no se deja fundir: %v", err)
	}
	// Con el PIN tampoco se abre lo subido.
	if _, err := AbrirEnMemoria(subida, "1234"); !errors.Is(err, ErrSinRanura) {
		t.Fatalf("el PIN abre lo del servidor: %v", err)
	}
}

func TestUnaVersionQueNoCuadraNoSeFunde(t *testing.T) {
	a, b, s := dosEquipos(t)
	mustPoner(t, a.b, Entrada{Titulo: "Nueva"})
	a.sincronizar(t, s)
	// El servidor devuelve la bóveda de la versión 2 diciendo que es la 3.
	if _, err := b.b.Fundir(s.datos, s.version+1, b.base, OpcionesDeFusion{}); !errors.Is(err, ErrRetroceso) {
		t.Fatalf("funde una versión que miente: %v", err)
	}
}

func TestOtraBovedaNoSeFunde(t *testing.T) {
	a, _, _ := dosEquipos(t)
	otra, _, _ := nueva(t)
	datos, _, _ := otra.PrepararSubida(1)
	if _, err := a.b.Fundir(datos, 1, nil, OpcionesDeFusion{}); !errors.Is(err, ErrOtraBoveda) {
		t.Fatalf("funde una bóveda que no es la suya: %v", err)
	}
}

func TestLaPosesionNoCambiaConLaMaestra(t *testing.T) {
	b, _, _ := nueva(t)
	antes, err := b.Posesion()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.CambiarMaestra("otra contraseña maestra"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.RotarRecuperacion(); err != nil {
		t.Fatal(err)
	}
	despues, _ := b.Posesion()
	otra, _, _ := nueva(t)
	deOtra, _ := otra.Posesion()
	if string(antes) != string(despues) {
		t.Fatal("la posesión cambia al cambiar la maestra o la de recuperación")
	}
	if string(antes) == string(deOtra) || len(antes) != 32 {
		t.Fatal("dos bóvedas dan la misma posesión, o no mide 32 bytes")
	}
}

// ------------------------------------------------------------------ la fusión, caso a caso

func TestLoQueCambiaEnUnEquipoLlegaAlOtro(t *testing.T) {
	a, b, s := dosEquipos(t)
	e := buscar(t, a.b, "Banco")
	e.Notas = "cambiada en A"
	mustPoner(t, a.b, e)
	mustPoner(t, a.b, Entrada{Titulo: "Nueva en A"})
	a.sincronizar(t, s)
	if subio := b.sincronizar(t, s); subio {
		t.Fatal("B sube sin tener nada suyo")
	}
	if buscar(t, b.b, "Banco").Notas != "cambiada en A" || !hay(b.b, "Nueva en A") {
		t.Fatal("B no tiene lo de A")
	}
}

func TestCamposDistintosEnCadaEquipoSeJuntan(t *testing.T) {
	a, b, s := dosEquipos(t)
	ea := buscar(t, a.b, "Banco")
	ea.Notas = "nota de A"
	mustPoner(t, a.b, ea)
	eb := buscar(t, b.b, "Banco")
	eb.Usuario = "usuario de B"
	mustPoner(t, b.b, eb)
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	a.sincronizar(t, s)
	for _, x := range []*equipo{a, b} {
		e := buscar(t, x.b, "Banco")
		if e.Notas != "nota de A" || e.Usuario != "usuario de B" {
			t.Fatalf("%s: %q / %q", x.nombre, e.Notas, e.Usuario)
		}
	}
}

func TestDosContrasenasAlaVezNoSePierdeNinguna(t *testing.T) {
	a, b, s := dosEquipos(t)
	ea := buscar(t, a.b, "Banco")
	ea.CambiarSecreto("de A", time.Now())
	mustPoner(t, a.b, ea)
	eb := buscar(t, b.b, "Banco")
	eb.CambiarSecreto("de B", time.Now())
	mustPoner(t, b.b, eb)
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	a.sincronizar(t, s)
	fa, fb := buscar(t, a.b, "Banco"), buscar(t, b.b, "Banco")
	if canon(fa) != canon(fb) {
		t.Fatalf("los dos equipos no han quedado igual:\n%s\n%s", canon(fa), canon(fb))
	}
	todas := map[string]bool{fa.Secreto: true}
	for _, h := range fa.Historial {
		todas[h.Secreto] = true
	}
	for _, x := range []string{"uno", "de A", "de B"} {
		if !todas[x] {
			t.Fatalf("se ha perdido la contraseña «%s»: %s", x, canon(fa))
		}
	}
}

func TestElDesempateDaLoMismoLoMireQuienLoMire(t *testing.T) {
	base := Entrada{ID: "x", Titulo: "T", Secreto: "0", Revision: 3, Cambiada: "2026-01-01T00:00:00Z"}
	l, r := base, base
	l.Secreto, l.Revision = "L", 4
	r.Secreto, r.Revision = "R", 4
	r.Cambiada = "2026-01-01T00:00:00Z"
	una := fundirCampos(l, r, &base, time.Unix(0, 0).UTC())
	otra := fundirCampos(r, l, &base, time.Unix(0, 0).UTC())
	if canon(una) != canon(otra) {
		t.Fatalf("fundir A con B no da lo mismo que B con A:\n%s\n%s", canon(una), canon(otra))
	}
}

func TestBorradoContraEdicionGanaLaEdicion(t *testing.T) {
	a, b, s := dosEquipos(t)
	id := buscar(t, a.b, "Banco").ID
	eb := buscar(t, b.b, "Banco")
	eb.Notas = "la toqué en B"
	mustPoner(t, b.b, eb)
	// Y A la borra **una hora después** de que B la editara. Gana igual la edición:
	// B no sabía nada del borrado. Con las dos cosas en el mismo segundo esto
	// pasaba por el empate de fechas y no ejercitaba la regla de verdad.
	conReloj(t, func() time.Time { return time.Now().Add(time.Hour) }, func() {
		_ = a.b.Borrar(id)
		if err := a.b.BorrarDelTodo(id); err != nil {
			t.Fatal(err)
		}
	})
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	a.sincronizar(t, s)
	for _, x := range []*equipo{a, b} {
		if !hay(x.b, "Banco") || buscar(t, x.b, "Banco").Notas != "la toqué en B" {
			t.Fatalf("%s ha perdido la entrada editada", x.nombre)
		}
		if x.b.cont.Lapidas[id] != "" {
			t.Fatalf("%s conserva la lápida de una entrada viva", x.nombre)
		}
	}
}

func TestLoBorradoEnUnEquipoSeBorraEnElOtro(t *testing.T) {
	a, b, s := dosEquipos(t)
	mustPoner(t, a.b, Entrada{Titulo: "Otra"})
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	id := buscar(t, a.b, "Banco").ID
	_ = a.b.Borrar(id)
	_ = a.b.BorrarDelTodo(id)
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	if hay(b.b, "Banco") {
		t.Fatal("lo borrado del todo en A sigue en B")
	}
	if b.b.cont.Lapidas[id] == "" {
		t.Fatal("B no se queda la lápida, y la resucitaría otro equipo")
	}
	// La papelera también viaja: mandar a la papelera en B llega a A.
	idOtra := buscar(t, b.b, "Otra").ID
	_ = b.b.Borrar(idOtra)
	b.sincronizar(t, s)
	a.sincronizar(t, s)
	if !buscar(t, a.b, "Otra").Papelera {
		t.Fatal("la papelera de B no llega a A")
	}
}

func TestSinBaseSeUneYLaLapidaMandaSobreLoViejo(t *testing.T) {
	a, b, s := dosEquipos(t)
	mustPoner(t, b.b, Entrada{Titulo: "Solo en B"})
	id := buscar(t, a.b, "Banco").ID
	// El borrado, una hora después: con fechas de un segundo, borrar en el mismo
	// segundo en que se tocó empata, y en el empate la entrada vive.
	conReloj(t, func() time.Time { return time.Now().Add(time.Hour) }, func() {
		_ = a.b.Borrar(id)
		_ = a.b.BorrarDelTodo(id)
	})
	a.sincronizar(t, s)
	b.base = nil // B ha perdido su base
	b.sincronizar(t, s)
	if !hay(b.b, "Solo en B") {
		t.Fatal("sin base se pierde lo nuevo de B")
	}
	if hay(b.b, "Banco") {
		t.Fatal("sin base, una entrada sin tocar desde antes de su lápida resucita")
	}
}

func TestLosSitiosExcluidosSeFundenComoConjunto(t *testing.T) {
	a, b, s := dosEquipos(t)
	_ = a.b.Excluir("uno.com")
	_ = a.b.Excluir("dos.com")
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	_ = a.b.QuitarExclusion("uno.com")
	_ = b.b.Excluir("tres.com")
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	a.sincronizar(t, s)
	for _, x := range []*equipo{a, b} {
		if got := fmt.Sprint(x.b.Excluidos()); got != "[dos.com tres.com]" {
			t.Fatalf("%s: %s", x.nombre, got)
		}
	}
}

func TestUnaSeccionDesconocidaViajaEntera(t *testing.T) {
	a, b, s := dosEquipos(t)
	a.b.cont.Extra = map[string]json.RawMessage{"identidad": json.RawMessage(`{"semilla":"abc"}`)}
	a.b.cuerpoSucio = true
	if err := a.b.Guardar(); err != nil {
		t.Fatal(err)
	}
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	if string(b.b.cont.Extra["identidad"]) != `{"semilla":"abc"}` {
		t.Fatalf("la sección que esta versión no conoce no llega: %v", b.b.cont.Extra)
	}
}

func TestLaContrasenaCambiadaEnOtroEquipoLlegaConSuRanura(t *testing.T) {
	a, b, s := dosEquipos(t)
	if err := a.b.CambiarMaestra("la maestra nueva de A"); err != nil {
		t.Fatal(err)
	}
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	if _, err := Abrir(b.b.ruta, "la maestra nueva de A"); err != nil {
		t.Fatalf("en B no abre la contraseña nueva: %v", err)
	}
	if _, err := Abrir(b.b.ruta, maestra); !errors.Is(err, ErrSinRanura) {
		t.Fatalf("en B sigue abriendo la vieja: %v", err)
	}
}

// **La papelera no es un borrado, a estos efectos** (revisión del 2026-09-23).
// Mandar a la papelera casi todo en un equipo es un gesto normal y reversible, y
// hasta ahora paraba en seco la sincronización de todos los demás —que además
// dejaban de subir lo suyo— sin ninguna forma de decir que sí. El freno es para
// una fusión que **se lleve** entradas del fichero.
func TestMandarALaPapeleraNoParaLaSincronizacion(t *testing.T) {
	a, b, s := dosEquipos(t)
	for i := range 5 {
		mustPoner(t, a.b, Entrada{Titulo: fmt.Sprint("E", i)})
	}
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	for _, e := range append([]Entrada(nil), a.b.cont.Entradas...) {
		if e.Titulo != "E0" {
			_ = a.b.Borrar(e.ID) // a la papelera, sin borrar del todo
		}
	}
	a.sincronizar(t, s)

	f, err := b.b.Fundir(s.datos, s.version, b.base, OpcionesDeFusion{})
	if err != nil {
		t.Fatalf("la papelera de otro equipo para la sincronización: %v", err)
	}
	if f.Borradas != 0 {
		t.Fatalf("cuenta %d como borradas y solo están en la papelera", f.Borradas)
	}
	if b.b.Cuantas() != 1 || b.b.EnLaPapelera() != 5 {
		t.Fatalf("en B quedan %d vivas y %d en la papelera", b.b.Cuantas(), b.b.EnLaPapelera())
	}
}

func TestUnaFusionQueSeLlevaMediaBovedaNoSeAplicaSola(t *testing.T) {
	a, b, s := dosEquipos(t)
	for i := range 5 {
		mustPoner(t, a.b, Entrada{Titulo: fmt.Sprint("E", i)})
	}
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	for _, e := range append([]Entrada(nil), a.b.cont.Entradas...) {
		if e.Titulo != "E0" && e.Titulo != "E1" {
			_ = a.b.Borrar(e.ID)
			_ = a.b.BorrarDelTodo(e.ID)
		}
	}
	a.sincronizar(t, s)
	antes, _ := os.ReadFile(b.b.ruta)
	f, err := b.b.Fundir(s.datos, s.version, b.base, OpcionesDeFusion{})
	if !errors.Is(err, ErrMuchosBorrados) {
		t.Fatalf("se lleva 4 de 6 sin preguntar: %v", err)
	}
	if f.Borradas != 4 {
		t.Fatalf("dice que borraría %d", f.Borradas)
	}
	despues, _ := os.ReadFile(b.b.ruta)
	if string(antes) != string(despues) {
		t.Fatal("sin permiso ha tocado el fichero")
	}
	if _, err := b.b.Fundir(s.datos, s.version, b.base, OpcionesDeFusion{AunqueBorreMucho: true}); err != nil {
		t.Fatal(err)
	}
	copia, err := os.ReadFile(b.b.ruta + ".antes-de-fundir")
	if err != nil || string(copia) != string(antes) {
		t.Fatal("no deja la copia de antes de fundir")
	}
}

// ------------------------------------------------------------------ la prueba grande

// Tres equipos hacen cosas al azar y se sincronizan cuando les toca; al final,
// sin tocar nada más, todos tienen que acabar igual que el servidor, y ninguna
// contraseña puede haberse perdido salvo las de entradas que alguien borró del
// todo a propósito.
//
// Es la prueba que vigila el riesgo número uno del plan: una fusión mala se copia
// a todos los equipos.
func TestTresEquiposConvergenYNoPierdenContrasenas(t *testing.T) {
	semillas := 16
	if testing.Short() {
		semillas = 3
	}
	for semilla := range semillas {
		t.Run(fmt.Sprint("semilla ", semilla), func(t *testing.T) {
			// En paralelo entre ellas: cada guardado sella con Argon2id, y en fila
			// son medio minuto. No tocan el reloj de mentira (`ahora`).
			t.Parallel()
			convergen(t, uint64(semilla))
		})
	}
}

func convergen(t *testing.T, semilla uint64) {
	azar := rand.New(rand.NewPCG(semilla, 42))
	origen, _, _ := nueva(t)
	s := &servidorFalso{}
	equipos := []*equipo{{nombre: "A", b: origen, serie: -1}}
	equipos[0].sincronizar(t, s)
	for _, n := range []string{"B", "C"} {
		e := clonar(t, origen, n)
		e.sincronizar(t, s)
		equipos = append(equipos, e)
	}

	secretos := map[string]map[string]bool{} // id → contraseñas que ha tenido
	borradaDelTodo := map[string]bool{}
	cuenta := 0

	for paso := range 40 {
		e := equipos[azar.IntN(len(equipos))]
		entradas := e.b.cont.Entradas
		switch op := azar.IntN(10); {
		case op < 2 || len(entradas) == 0:
			cuenta++
			secreto := fmt.Sprintf("s%d-%s-%d", cuenta, e.nombre, paso)
			mustPoner(t, e.b, Entrada{Titulo: fmt.Sprint("T", cuenta), Secreto: secreto})
			id := e.b.cont.Entradas[len(e.b.cont.Entradas)-1].ID
			secretos[id] = map[string]bool{secreto: true}
		case op < 4:
			x := entradas[azar.IntN(len(entradas))]
			secreto := fmt.Sprintf("s-%s-%d", e.nombre, paso)
			x.CambiarSecreto(secreto, time.Now())
			mustPoner(t, e.b, x)
			if secretos[x.ID] == nil {
				secretos[x.ID] = map[string]bool{}
			}
			secretos[x.ID][secreto] = true
		case op < 5:
			x := entradas[azar.IntN(len(entradas))]
			x.Notas = fmt.Sprintf("nota %s %d", e.nombre, paso)
			mustPoner(t, e.b, x)
		case op < 6:
			x := entradas[azar.IntN(len(entradas))]
			if x.Papelera {
				if azar.IntN(2) == 0 {
					_ = e.b.Restaurar(x.ID)
				} else {
					_ = e.b.BorrarDelTodo(x.ID)
					borradaDelTodo[x.ID] = true
				}
			} else {
				_ = e.b.Borrar(x.ID)
			}
		case op < 7:
			sitio := fmt.Sprintf("sitio%d.com", azar.IntN(4))
			if e.b.Excluido(sitio) {
				_ = e.b.QuitarExclusion(sitio)
			} else {
				_ = e.b.Excluir(sitio)
			}
		default:
			e.sincronizar(t, s)
		}
	}

	// Sin tocar nada más, unas vueltas hasta que nadie tenga nada que subir.
	quieto := false
	for vuelta := 0; vuelta < 6 && !quieto; vuelta++ {
		quieto = true
		for _, e := range equipos {
			if e.sincronizar(t, s) {
				quieto = false
			}
		}
	}
	if !quieto {
		t.Fatal("tras seis vueltas sin cambios, los equipos siguen subiendo: se pasan la bóveda sin fin")
	}
	for _, e := range equipos {
		e.sincronizar(t, s) // el último que subió ya lo tiene; los demás lo bajan
	}

	docS, _ := leerDocumento(s.datos)
	_, contS, err := desempaquetar(docS, origen.llave)
	if err != nil {
		t.Fatal(err)
	}
	referencia, _ := json.Marshal(normal(contS))
	for _, e := range equipos {
		suyo, _ := json.Marshal(normal(e.b.cont))
		if string(suyo) != string(referencia) {
			t.Fatalf("%s no ha acabado igual que el servidor:\n%s\n%s", e.nombre, suyo, referencia)
		}
	}

	for id, todas := range secretos {
		if borradaDelTodo[id] || len(todas) > maximoHistorial {
			continue
		}
		var la *Entrada
		for i := range contS.Entradas {
			if contS.Entradas[i].ID == id {
				la = &contS.Entradas[i]
			}
		}
		if la == nil {
			t.Fatalf("la entrada %s ha desaparecido sin que nadie la borrara del todo", id)
		}
		tiene := map[string]bool{la.Secreto: true}
		for _, h := range la.Historial {
			tiene[h.Secreto] = true
		}
		for x := range todas {
			if !tiene[x] {
				t.Fatalf("se ha perdido la contraseña %q de %s: %s", x, id, canon(*la))
			}
		}
	}
}

// ------------------------------------------------------------------ lo que ya existe

// Una bóveda escrita por la 2.22.2 —con el código de entonces, no con éste— se
// abre, conserva todo y se sincroniza. El fichero se generó una vez y **no se
// regenera**: es la prueba de que lo que tienen hoy los equipos del cliente sigue
// sirviendo, igual que los vectores fijos del formato (ADR 0022).
func TestUnaBovedaDeLa2222SeAbreYSeSincroniza(t *testing.T) {
	datos, err := os.ReadFile("testdata/boveda-2.22.2.esfinge")
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	if err := os.WriteFile(ruta, datos, 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := Abrir(ruta, "la maestra de la 2.22.2")
	if err != nil {
		t.Fatal(err)
	}
	banco := buscar(t, b, "Banco")
	if banco.Secreto != "primera" || banco.TOTP == "" || len(banco.Historial) != 1 || banco.Revision != 0 {
		t.Fatalf("la entrada vieja no llega entera: %s", canon(banco))
	}
	if string(banco.Extra["campoDelFuturo"]) != `{"x":1}` {
		t.Fatal("se pierde el campo que la 2.22.2 conservaba sin entenderlo")
	}
	if buscar(t, b, "Visa").Numero != "4111111111111111" || !buscar(t, b, "En la papelera").Papelera {
		t.Fatal("la tarjeta o la papelera no llegan enteras")
	}
	if !b.Excluido("nunca.com") {
		t.Fatal("se pierden los sitios excluidos")
	}

	// Se sube, la baja otro equipo, y un cambio en cada uno llega al otro.
	s := &servidorFalso{}
	a := &equipo{nombre: "A", b: b, serie: -1}
	a.sincronizar(t, s)
	otro := clonar(t, b, "B")
	otro.sincronizar(t, s)
	banco.Notas = "desde A"
	mustPoner(t, a.b, banco)
	if r := buscar(t, a.b, "Banco").Revision; r != 1 {
		t.Fatalf("la primera edición de una entrada vieja deja la revisión en %d", r)
	}
	mustPoner(t, otro.b, Entrada{Titulo: "Desde B"})
	a.sincronizar(t, s)
	otro.sincronizar(t, s)
	a.sincronizar(t, s)
	if buscar(t, otro.b, "Banco").Notas != "desde A" || !hay(a.b, "Desde B") {
		t.Fatal("la bóveda de la 2.22.2 no se sincroniza")
	}
}

// La forma canónica tiene que poder sacarla igual la extensión en TypeScript:
// claves en orden a todos los niveles y sin el escape de HTML que Go pone solo.
func TestLaFormaCanonicaEsLaDeLaEspecificacion(t *testing.T) {
	e := Entrada{ID: "a", Tipo: TipoCredencial, Titulo: "<Tom & Jerry>", Revision: 2,
		Sitios: []string{"b.com", "a.com"}, Historial: []Antigua{{Secreto: "x", Hasta: "2026"}},
		Extra: map[string]json.RawMessage{"zeta": json.RawMessage(`{"b":1,"a":[2,1]}`)}}
	quiere := `{"cambiada":"","creada":"","historial":[{"hasta":"2026","secreto":"x"}],"id":"a","revision":2,` +
		`"sitios":["b.com","a.com"],"tipo":"credencial","titulo":"<Tom & Jerry>","zeta":{"a":[2,1],"b":1}}`
	if got := canon(e); got != quiere {
		t.Fatalf("\n%s\n%s", got, quiere)
	}
}

// ------------------------------------------------------------------ piezas de la cuenta

func TestCrearEnMemoriaNoEscribeHastaDarleFichero(t *testing.T) {
	b, rec, err := CrearEnMemoria(maestra)
	if err != nil || rec == "" {
		t.Fatalf("%v, %q", err, rec)
	}
	if err := b.Poner(Entrada{Titulo: "X"}); !errors.Is(err, errSinFichero) {
		t.Fatalf("sin fichero no se guarda: %v", err)
	}
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	if err := b.GuardarEn(ruta); err != nil {
		t.Fatal(err)
	}
	for _, llave := range []string{maestra, rec} {
		if _, err := Abrir(ruta, llave); err != nil {
			t.Fatalf("no abre con %q: %v", llave, err)
		}
	}
}

func TestAlGuardarAvisaDeCadaGuardado(t *testing.T) {
	b, _, _ := nueva(t)
	avisos := make(chan struct{}, 10)
	b.AlGuardar(func() { avisos <- struct{}{} })
	mustPoner(t, b, Entrada{Titulo: "X"})
	_ = b.Excluir("a.com")
	for range 2 {
		select {
		case <-avisos:
		case <-time.After(2 * time.Second):
			t.Fatal("un guardado no ha avisado")
		}
	}
}

func TestUnSecretoSelladoSoloSeAbreConLaBoveda(t *testing.T) {
	b, _, _ := nueva(t)
	sellado, err := b.SellarSecreto([]byte("s1.cuenta.sesion"))
	if err != nil {
		t.Fatal(err)
	}
	claro, err := b.AbrirSecreto(sellado)
	if err != nil || string(claro) != "s1.cuenta.sesion" {
		t.Fatalf("%q %v", claro, err)
	}
	otra, _, _ := nueva(t)
	if _, err := otra.AbrirSecreto(sellado); err == nil {
		t.Fatal("otra bóveda abre el secreto")
	}
	b.Cerrar()
	if _, err := b.AbrirSecreto(sellado); !errors.Is(err, ErrCerrada) {
		t.Fatalf("con la bóveda cerrada: %v", err)
	}
}

func TestTraerJuntaDosBovedasSinRepetir(t *testing.T) {
	a, _, _ := nueva(t)
	mustPoner(t, a, Entrada{Titulo: "De la cuenta"})
	otra, _, _ := nueva(t)
	mustPoner(t, otra, Entrada{Titulo: "De este equipo"})
	mustPoner(t, otra, Entrada{Titulo: "Borrada aquí"})
	_ = otra.Borrar(buscar(t, otra, "Borrada aquí").ID)
	_ = otra.Excluir("nunca.com")
	n, err := a.Traer(otra)
	if err != nil || n != 2 {
		t.Fatalf("%d, %v", n, err)
	}
	if n, _ := a.Traer(otra); n != 0 {
		t.Fatalf("traer dos veces repite %d", n)
	}
	if !hay(a, "De este equipo") || !buscar(t, a, "Borrada aquí").Papelera || !a.Excluido("nunca.com") {
		t.Fatal("no llega todo, o la papelera no se respeta")
	}
	if id, _ := IDDe(mustLeer(t, a.ruta)); id != a.ID() {
		t.Fatal("IDDe no lee el identificador")
	}
}

func mustLeer(t *testing.T, ruta string) []byte {
	t.Helper()
	d, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Dos bóvedas importadas por separado del mismo gestor: las mismas cuentas con
// identificadores distintos. Juntarlas no las repite, y lo que difiere en algo sí
// llega, porque no se sabe cuál es la buena. Lo vio el cliente en sus dos Macs.
func TestTraerNoRepiteLoImportadoDosVeces(t *testing.T) {
	a, _, _ := nueva(t)
	otra, _, _ := nueva(t)
	for _, b := range []*Boveda{a, otra} {
		mustPoner(t, b, Entrada{Titulo: "Correo", Usuario: "yo@x.com", Secreto: "uno", Sitios: []string{"x.com"}})
		mustPoner(t, b, Entrada{Titulo: "Banco", Usuario: "yo", Secreto: "dos", Sitios: []string{"banco.es"}})
	}
	mustPoner(t, otra, Entrada{Titulo: "Correo", Usuario: "yo@x.com", Secreto: "la otra", Sitios: []string{"x.com"}})
	n, err := a.Traer(otra)
	if err != nil || n != 1 {
		t.Fatalf("trae %d (quiero solo la distinta), %v", n, err)
	}
	if c, _ := a.Repetidas(); c != 0 {
		t.Fatalf("quedan %d repetidas", c)
	}
}

// Lo que ya se juntó dos veces se limpia: sobra una de cada pareja idéntica, va a
// la papelera y no se borra, y de cada grupo se queda la misma sea quien sea quien
// limpie —si no, dos equipos limpiando a la vez se quedarían sin ninguna—.
func TestQuitarRepetidas(t *testing.T) {
	a, _, _ := nueva(t)
	mustPoner(t, a, Entrada{Titulo: "Correo", Usuario: "yo@x.com", Secreto: "uno", Sitios: []string{"x.com"}})
	mustPoner(t, a, Entrada{Titulo: "Banco", Usuario: "yo", Secreto: "dos"})
	mustPoner(t, a, Entrada{Titulo: "Banco", Usuario: "yo", Secreto: "dos"})
	mustPoner(t, a, Entrada{Titulo: "Banco", Usuario: "yo", Secreto: "dos"})
	mustPoner(t, a, Entrada{Titulo: "Banco", Usuario: "yo", Secreto: "otra"})
	if c, err := a.Repetidas(); err != nil || c != 2 {
		t.Fatalf("repetidas: %d, %v", c, err)
	}
	var menor string
	for _, e := range a.cont.Entradas {
		if e.Secreto == "dos" && (menor == "" || e.ID < menor) {
			menor = e.ID
		}
	}
	n, err := a.QuitarRepetidas()
	if err != nil || n != 2 {
		t.Fatalf("quita %d, %v", n, err)
	}
	vivas := map[string]int{}
	for _, e := range a.cont.Entradas {
		if !e.Papelera {
			vivas[e.Secreto]++
			if e.Secreto == "dos" && e.ID != menor {
				t.Fatal("no se queda la de identificador menor")
			}
		}
	}
	if vivas["uno"] != 1 || vivas["dos"] != 1 || vivas["otra"] != 1 {
		t.Fatalf("quedan %v", vivas)
	}
	if len(a.Papelera()) != 2 {
		t.Fatal("las que sobran no están en la papelera")
	}
	if c, _ := a.Repetidas(); c != 0 {
		t.Fatalf("siguen %d", c)
	}
}

// Lo que vio el cliente: parejas idénticas a la vista que la primera versión no
// contaba, porque se diferenciaban en lo que la ventana no enseña. Se juntan en la
// de identificador menor sin perder nada; lo que choca de verdad se queda.
func TestRepetidasConDiferenciasQueNoSeVen(t *testing.T) {
	a, _, _ := nueva(t)
	base := Entrada{Tipo: TipoCredencial, Titulo: "Correo", Usuario: "yo@x.com", Secreto: "uno", Sitios: []string{"x.com"}}
	una := base
	una.Etiquetas = []string{"trabajo"}
	otra := base
	otra.Titulo = "Correo "
	otra.Sitios = []string{"https://x.com", "mail.x.com"}
	otra.Carpeta = "Dashlane"
	otra.Notas = "la nota que solo tenía una"
	otra.Historial = []Antigua{{Secreto: "cero", Hasta: "2026-01-01T00:00:00Z"}}
	otra.Extra = map[string]json.RawMessage{"categoria": json.RawMessage(`"Email"`)}
	distinta := base
	distinta.Secreto = "dos"
	for _, e := range []Entrada{una, otra, distinta} {
		mustPoner(t, a, e)
	}
	// Las dos primeras son la misma cuenta; la de otra contraseña es otra cuenta.
	if c, err := a.Repetidas(); err != nil || c != 1 {
		t.Fatalf("repetidas: %d, %v", c, err)
	}
	if n, err := a.QuitarRepetidas(); err != nil || n != 1 {
		t.Fatalf("quita %d, %v", n, err)
	}
	var vivas []Entrada
	for _, e := range a.cont.Entradas {
		if !e.Papelera {
			vivas = append(vivas, e)
		}
	}
	if len(vivas) != 2 {
		t.Fatalf("quedan %d vivas", len(vivas))
	}
	var junta *Entrada
	for i := range vivas {
		if vivas[i].Secreto == "uno" {
			junta = &vivas[i]
		}
	}
	if junta == nil || junta.Notas != "la nota que solo tenía una" || len(junta.Historial) != 1 ||
		strings.Join(junta.Etiquetas, ",") != "trabajo" || len(junta.Sitios) != 3 || junta.Carpeta != "Dashlane" ||
		junta.Extra["categoria"] == nil {
		t.Fatalf("la que se queda no lleva lo de las dos: %+v", junta)
	}

	// Lo que choca se queda: dos notas distintas, una de las dos es la buena.
	b, _, _ := nueva(t)
	conNota := base
	conNota.Notas = "una nota"
	otraNota := base
	otraNota.Notas = "otra nota"
	mustPoner(t, b, conNota)
	mustPoner(t, b, otraNota)
	if c, _ := b.Repetidas(); c != 0 {
		t.Fatalf("junta dos notas distintas: %d", c)
	}
}

// Dos equipos que limpian a la vez llegan a lo mismo: si no, cada uno mandaría a
// la papelera una copia distinta y la cuenta desaparecería de los dos.
func TestQuitarRepetidasEsLoMismoEnDosEquipos(t *testing.T) {
	uno, _, _ := nueva(t)
	for i := 0; i < 4; i++ {
		e := Entrada{Tipo: TipoCredencial, Titulo: "Banco", Usuario: "yo", Secreto: "s"}
		if i%2 == 1 {
			e.Carpeta = "Finanzas"
		}
		mustPoner(t, uno, e)
	}
	// El otro equipo tiene las mismas entradas en otro orden.
	otro, _, _ := nueva(t)
	otro.cont.Entradas = append([]Entrada(nil), uno.cont.Entradas...)
	for i, j := 0, len(otro.cont.Entradas)-1; i < j; i, j = i+1, j-1 {
		otro.cont.Entradas[i], otro.cont.Entradas[j] = otro.cont.Entradas[j], otro.cont.Entradas[i]
	}
	for _, b := range []*Boveda{uno, otro} {
		if n, err := b.QuitarRepetidas(); err != nil || n != 3 {
			t.Fatalf("quita %d, %v", n, err)
		}
	}
	vivaDe := func(b *Boveda) Entrada {
		for _, e := range b.cont.Entradas {
			if !e.Papelera {
				return e
			}
		}
		return Entrada{}
	}
	u, o := vivaDe(uno), vivaDe(otro)
	if u.ID == "" || u.ID != o.ID || u.Carpeta != "Finanzas" || contenidoDe(u) != contenidoDe(o) {
		t.Fatalf("cada equipo se queda con otra: %+v / %+v", u, o)
	}
}

// Juntar al entrar en una cuenta: la misma cuenta con lo que la ventana no enseña
// distinto no se trae dos veces; se junta en la que hay.
func TestTraerJuntaLaMismaCuentaConDiferenciasQueNoSeVen(t *testing.T) {
	a, _, _ := nueva(t)
	otra, _, _ := nueva(t)
	mustPoner(t, a, Entrada{Tipo: TipoCredencial, Titulo: "Correo", Usuario: "yo", Secreto: "s", Sitios: []string{"x.com"}})
	mustPoner(t, otra, Entrada{Tipo: TipoCredencial, Titulo: "Correo", Usuario: "yo", Secreto: "s", Sitios: []string{"https://x.com"}, Carpeta: "Email"})
	n, err := a.Traer(otra)
	if err != nil || n != 0 {
		t.Fatalf("trae %d, %v", n, err)
	}
	if len(a.cont.Entradas) != 1 || a.cont.Entradas[0].Carpeta != "Email" || len(a.cont.Entradas[0].Sitios) != 2 {
		t.Fatalf("no se junta: %+v", a.cont.Entradas)
	}
}

// La carpeta y los campos del gestor guardados aparte no impiden juntar: son
// ordenar, no secretos, y así llegaban las del cliente de dos importaciones.
func TestRepetidasConCarpetasDistintas(t *testing.T) {
	a, _, _ := nueva(t)
	for _, c := range []string{"", "Dashlane", "Email"} {
		mustPoner(t, a, Entrada{Tipo: TipoCredencial, Titulo: "Banco", Usuario: "yo", Secreto: "s", Carpeta: c,
			Extra: map[string]json.RawMessage{"categoria": json.RawMessage(`"` + c + `"`)}})
	}
	if n, err := a.QuitarRepetidas(); err != nil || n != 2 {
		t.Fatalf("quita %d, %v", n, err)
	}
}

// Contar no cambia nada: juntar trabaja sobre una copia.
func TestContarRepetidasNoTocaLaBoveda(t *testing.T) {
	a, _, _ := nueva(t)
	mustPoner(t, a, Entrada{Tipo: TipoCredencial, Titulo: "B", Usuario: "yo", Secreto: "s", Sitios: []string{"a.com"},
		Extra: map[string]json.RawMessage{"x": json.RawMessage(`1`)}})
	mustPoner(t, a, Entrada{Tipo: TipoCredencial, Titulo: "B", Usuario: "yo", Secreto: "s", Sitios: []string{"b.com"},
		Extra: map[string]json.RawMessage{"y": json.RawMessage(`2`)}})
	antes := []string{contenidoDe(a.cont.Entradas[0]), contenidoDe(a.cont.Entradas[1])}
	if c, _ := a.Repetidas(); c != 1 {
		t.Fatalf("repetidas: %d", c)
	}
	for i := range antes {
		if contenidoDe(a.cont.Entradas[i]) != antes[i] {
			t.Fatalf("contar ha cambiado la entrada %d", i)
		}
	}
}

// **La edición gana al borrado también cuando el borrado es suave** (revisión del
// 2026-09-23). Antes, con la papelera, el reparto campo a campo se quedaba con las
// dos cosas: la entrada acababa en la papelera **con la contraseña nueva**, fuera
// de la lista, y a los treinta días se purgaba.
// **Una fusión que no se guarda no cambia nada.**
//
// Guardar falla de verdad: basta con que otro Esfinge haya tocado el fichero desde
// que se abrió éste. Si la fusión ya hubiera sustituido el contenido en memoria,
// la ventana enseñaría una bóveda que no está en ningún sitio y el siguiente
// guardado la escribiría sin que nadie lo decidiera.
func TestUnaFusionQueNoSeGuardaNoDejaRastroEnMemoria(t *testing.T) {
	a, b, s := dosEquipos(t)
	mustPoner(t, b.b, Entrada{Titulo: "Solo de B", Usuario: "yo", Secreto: "x"})
	b.sincronizar(t, s)

	// Entre medias, otro Esfinge escribe en el fichero de A: la serie del disco deja
	// de ser la que A tiene cargada.
	crudo, err := os.ReadFile(a.b.ruta)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(crudo, &doc); err != nil {
		t.Fatal(err)
	}
	doc["serie"] = doc["serie"].(float64) + 1
	otro, _ := json.Marshal(doc)
	if err := os.WriteFile(a.b.ruta, otro, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := a.b.Fundir(s.datos, s.version, a.base, OpcionesDeFusion{}); !errors.Is(err, ErrCambiada) {
		t.Fatalf("quiero ErrCambiada, tengo %v", err)
	}
	for _, e := range a.b.cont.Entradas {
		if e.Titulo == "Solo de B" {
			t.Fatal("la fusión que no se guardó se ha quedado en memoria")
		}
	}
}

// **Restaurar es un cambio, y sin base lo único que lo dice es la fecha.**
//
// Con base, sacar una entrada de la papelera se ve porque la entrada ya no es la
// que había; sin base —un equipo que entra en la cuenta sin haberse sincronizado
// nunca, o al que se le ha perdido el fichero `.base`— la única regla que queda es
// «vive si se cambió después de borrarse». Restaurar no tocaba la fecha de cambio,
// así que la entrada rescatada aquí perdía contra la purga de allí y se volvía a
// ir, en silencio y para siempre: la papelera de la otra ya estaba vacía.
func TestRestaurarGanaALaPurgaDeOtroEquipoAunqueSeFundaSinBase(t *testing.T) {
	a, b, s := dosEquipos(t)
	id := buscar(t, a.b, "Banco").ID
	_ = a.b.Borrar(id)
	a.sincronizar(t, s)
	b.sincronizar(t, s)

	// Una hora después, A vacía la papelera: deja su lápida y sube.
	conReloj(t, func() time.Time { return time.Now().Add(time.Hour) }, func() {
		if _, err := a.b.VaciarPapelera(); err != nil {
			t.Fatal(err)
		}
	})
	a.sincronizar(t, s)

	// Y dos horas después B la rescata de su papelera, sin saber nada de aquello.
	conReloj(t, func() time.Time { return time.Now().Add(2 * time.Hour) }, func() {
		if err := b.b.Restaurar(id); err != nil {
			t.Fatal(err)
		}
	})
	b.base = nil // el equipo que funde sin base
	b.sincronizar(t, s)

	viva, hay := b.b.Ver(id)
	if !hay || viva.Papelera {
		t.Fatalf("lo que B acababa de restaurar se ha ido con la purga de A: hay=%v %+v", hay, viva)
	}
	a.sincronizar(t, s)
	if _, hay := a.b.Ver(id); !hay {
		t.Fatal("A no ha recuperado la entrada que B rescató")
	}
}

func TestLaPapeleraDeUnEquipoNoSeLlevaLoQueOtroAcabaDeCambiar(t *testing.T) {
	a, b, s := dosEquipos(t)
	mustPoner(t, a.b, Entrada{Titulo: "Banco", Usuario: "yo", Secreto: "la vieja"})
	a.sincronizar(t, s)
	b.sincronizar(t, s)
	id := buscar(t, a.b, "Banco").ID

	// A la manda a la papelera; B le cambia la contraseña, en su equipo.
	_ = a.b.Borrar(id)
	e, _ := b.b.Ver(id)
	e.CambiarSecreto("la nueva", ahora())
	mustPoner(t, b.b, e)
	a.sincronizar(t, s)
	b.sincronizar(t, s)

	viva, hay := b.b.Ver(id)
	if !hay || viva.Papelera {
		t.Fatalf("la entrada que B acababa de cambiar se ha ido a la papelera: %+v", viva)
	}
	if viva.Secreto != "la nueva" {
		t.Fatalf("la contraseña es %q", viva.Secreto)
	}
	// Y converge: A ve lo mismo.
	a.sincronizar(t, s)
	enA, _ := a.b.Ver(id)
	if enA.Papelera || enA.Secreto != "la nueva" {
		t.Fatalf("en A: papelera=%v secreto=%q", enA.Papelera, enA.Secreto)
	}
}
