package boveda

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/cripto"
)

const maestraDePrueba = "una maestra bien larga para la prueba"

// personalYProyecto deja una bóveda personal abierta y un proyecto suyo con una
// entrada dentro.
func personalYProyecto(t *testing.T) (personal *Boveda, rutaProyecto string, llave []byte) {
	t.Helper()
	dir := t.TempDir()
	personal, _, err := Crear(filepath.Join(dir, "boveda.esfinge"), maestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	llave = personal.LlaveParaProyectos()
	if len(llave) == 0 {
		t.Fatal("una bóveda abierta tiene que dar su clave para los proyectos")
	}
	rutaProyecto = filepath.Join(dir, "proyectos", "a1b2c3d4e5f60718.esfinge")
	if err := os.MkdirAll(filepath.Dir(rutaProyecto), 0o700); err != nil {
		t.Fatal(err)
	}
	p, err := CrearProyecto(rutaProyecto, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Hosting de Acme", Secreto: "la de Acme"}); err != nil {
		t.Fatal(err)
	}
	return personal, rutaProyecto, llave
}

// Un proyecto se abre con la bóveda personal, **y con nada más**: no tiene ranura
// maestra ni de recuperación, así que su propia contraseña no existe.
func TestUnProyectoSeAbreConLaPersonalYNoConSuMaestra(t *testing.T) {
	_, ruta, llave := personalYProyecto(t)

	if !RanuraPrincipalEn(ruta) {
		t.Fatal("el fichero de un proyecto tiene que decir que lo es, sin abrirlo")
	}

	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if e := p.Buscar("Acme"); len(e) != 1 {
		t.Fatalf("en el proyecto hay %d entradas", len(e))
	}
	if !p.TieneRanuraPrincipal() {
		t.Fatal("y lo dice también con la bóveda abierta")
	}

	// **La maestra de la personal no abre el proyecto.** Es lo que hace que perder
	// la contraseña no sea perder el diseño: lo que abre es la clave de bóveda.
	if _, err := Abrir(ruta, maestraDePrueba); err == nil {
		t.Fatal("la contraseña maestra abre un proyecto, y no debería: ahí no hay ranura maestra")
	}

	// Y la clave de otra bóveda personal tampoco, con un error que no dice «roto».
	otra, _, _ := Crear(filepath.Join(t.TempDir(), "otra.esfinge"), "otra maestra bien larga de prueba")
	if _, err := AbrirProyecto(ruta, otra.LlaveParaProyectos()); err != ErrSinRanuraPrincipal {
		t.Fatalf("con otra personal el error es %v, y tenía que ser ErrSinRanuraPrincipal", err)
	}
}

// **La ranura del proyecto SÍ se sube**, al contrario que la del sistema.
//
// Es el espejo exacto de `TestLaRanuraDelSistemaAbreYNoViaja`, y existe porque el
// fallo que vigila es mudo y tardío: si alguien mete `boveda-principal` en
// `ranurasLocales` —que es media línea y parece lo prudente— los proyectos suben
// sin la única ranura que los abre, y **no se nota hasta el segundo equipo**, que
// se baja una bóveda que no puede abrir nadie.
func TestLaRanuraPrincipalSeSube(t *testing.T) {
	_, ruta, llave := personalYProyecto(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}

	datos, _, err := p.PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(datos), RanuraPrincipal) {
		t.Fatal("la ranura del proyecto no viaja en la subida: el otro equipo no lo podría abrir nunca")
	}
	// Y lo que se sube abre de verdad, que es la mitad que de verdad importa: que
	// el nombre esté en el JSON no dice que el sobre siga sellando lo mismo.
	otroEquipo, err := AbrirProyectoBytes("", datos, llave)
	if err != nil {
		t.Fatalf("lo que se sube no lo abre el otro equipo: %v", err)
	}
	if e := otroEquipo.Buscar("Acme"); len(e) != 1 {
		t.Fatalf("en el otro equipo hay %d entradas", len(e))
	}
}

// **Cambiar la contraseña maestra no toca ningún proyecto**, y ésa es la razón de
// que la ranura se envuelva con la clave de bóveda y no con la maestra.
func TestCambiarLaMaestraNoTocaLosProyectos(t *testing.T) {
	personal, ruta, llave := personalYProyecto(t)

	antes, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if err := personal.CambiarMaestra("otra maestra igual de larga que la primera"); err != nil {
		t.Fatal(err)
	}

	// El fichero del proyecto no se ha tocado: ni un byte.
	despues, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if string(antes) != string(despues) {
		t.Fatal("cambiar la maestra ha reescrito el fichero del proyecto")
	}
	// Y la clave de la personal sigue siendo la misma, así que sigue abriéndolo.
	if ahora := personal.LlaveParaProyectos(); string(ahora) != string(llave) {
		t.Fatal("la clave de bóveda ha cambiado al cambiar la maestra; con eso se perderían los proyectos")
	}
	if _, err := AbrirProyecto(ruta, personal.LlaveParaProyectos()); err != nil {
		t.Fatalf("tras cambiar la maestra el proyecto ya no abre: %v", err)
	}
}

// **Recuperar la bóveda personal recupera todos los proyectos.** Es lo que
// sostiene la decisión de que un proyecto no tenga clave de recuperación propia.
func TestRecuperarLaPersonalAbreLosProyectos(t *testing.T) {
	dir := t.TempDir()
	rutaPersonal := filepath.Join(dir, "boveda.esfinge")
	personal, recuperacion, err := Crear(rutaPersonal, maestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	llave := personal.LlaveParaProyectos()
	rutaProyecto := filepath.Join(dir, "proyecto.esfinge")
	if _, err := CrearProyecto(rutaProyecto, llave); err != nil {
		t.Fatal(err)
	}

	// Se olvida la maestra. Lo único que queda es la clave de recuperación.
	conRecuperacion, err := Abrir(rutaPersonal, recuperacion)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AbrirProyecto(rutaProyecto, conRecuperacion.LlaveParaProyectos()); err != nil {
		t.Fatalf("tras recuperar la personal, el proyecto no abre: %v", err)
	}

	// Y poniendo una maestra nueva sigue abriendo, que es el camino entero de
	// «he perdido la contraseña»: recuperar, poner otra y seguir trabajando.
	if err := conRecuperacion.CambiarMaestra("la tercera maestra, también larga"); err != nil {
		t.Fatal(err)
	}
	if _, err := AbrirProyecto(rutaProyecto, conRecuperacion.LlaveParaProyectos()); err != nil {
		t.Fatalf("tras poner una maestra nueva, el proyecto no abre: %v", err)
	}
}

// Quitar la ranura corta el lazo —es el paso 2 de entregar— y ponerla es cómo se
// adopta una bóveda que te han entregado.
func TestQuitarLaRanuraPrincipalCortaElLazoYPonerlaAdopta(t *testing.T) {
	_, ruta, llave := personalYProyecto(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}

	if err := p.QuitarRanuraPrincipal(); err != nil {
		t.Fatal(err)
	}
	if p.TieneRanuraPrincipal() || RanuraPrincipalEn(ruta) {
		t.Fatal("la ranura sigue puesta después de quitarla")
	}
	if _, err := AbrirProyecto(ruta, llave); err != ErrSinRanuraPrincipal {
		t.Fatalf("la personal sigue abriendo una bóveda desprendida (%v)", err)
	}

	// Y otra persona la adopta: se abre con lo que tenga y se le pone su ranura.
	otra, _, _ := Crear(filepath.Join(t.TempDir(), "otra.esfinge"), "la maestra de la otra persona")
	suya := otra.LlaveParaProyectos()
	if err := p.PonerRanuraPrincipal(suya); err != nil {
		t.Fatal(err)
	}
	adoptada, err := AbrirProyecto(ruta, suya)
	if err != nil {
		t.Fatalf("la adoptada no abre con la bóveda que la adoptó: %v", err)
	}
	if e := adoptada.Buscar("Acme"); len(e) != 1 {
		t.Fatalf("la adoptada tiene %d entradas", len(e))
	}
	// Y la de antes ya no abre: la ranura se reemplaza, no se acumula.
	if _, err := AbrirProyecto(ruta, llave); err != ErrSinRanuraPrincipal {
		t.Fatalf("la personal de antes sigue abriendo la adoptada (%v)", err)
	}
}

// El sobre del proyecto va con el perfil barato, por lo mismo que el del sistema:
// el secreto son los 43 bytes de la clave de la personal, no algo que se teclee, y
// Argon2id interactivo ahí solo haría que conmutar de proyecto tardara un segundo.
//
// Se miran **los parámetros que lleva el sobre**, como en el del sistema: medir
// tiempos da un número distinto en cada máquina.
func TestElSobreDelProyectoVaConElPerfilBarato(t *testing.T) {
	_, ruta, llave := personalYProyecto(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	i := p.ranura(RanuraPrincipal)
	if i < 0 {
		t.Fatal("no hay ranura principal")
	}
	crudo, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(p.doc.Sobres[i].Contenedor, "ESF1."))
	if err != nil {
		t.Fatal(err)
	}
	memoria := binary.BigEndian.Uint32(crudo[6:10])
	pasadas := binary.BigEndian.Uint32(crudo[10:14])
	if memoria != cripto.PerfilLlave.Memoria || pasadas != cripto.PerfilLlave.Pasadas {
		t.Errorf("el sobre del proyecto va con %d KiB y %d pasadas, y le toca el perfil de llave (%d KiB, %d)",
			memoria, pasadas, cripto.PerfilLlave.Memoria, cripto.PerfilLlave.Pasadas)
	}
}

// Un proyecto es una bóveda cualquiera: su identificador es suyo y no el de la
// personal. Si lo compartieran, la sincronización los daría por la misma bóveda y
// se pisarían en el servidor.
func TestUnProyectoTieneSuPropioIdentificador(t *testing.T) {
	personal, ruta, llave := personalYProyecto(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID() == personal.ID() {
		t.Fatal("el proyecto tiene el identificador de la personal")
	}
	if p.ID() == "" {
		t.Fatal("el proyecto no tiene identificador")
	}
}

// **La lista de proyectos sobrevive a la sincronización**, y esta prueba existe
// porque el fallo que vigila ya estaba escrito: `fundirContenido` arma el contenido
// **campo a campo**, así que una sección nueva que nadie añada ahí no se funde —se
// pierde—. No es la trampa de `Extra`, que conserva lo que no se entiende: aquí lo
// entendemos y lo tiramos, y el síntoma sería que la lista de clientes desaparece
// en la primera pasada de la sincronización.
//
// Se comprueba lo que de verdad tiene que pasar: lo que crea cada equipo llega al
// otro, lo que uno olvida se va de los dos, y el nombre que uno cambia gana.
func TestLaFusionConservaYFundeLosProyectos(t *testing.T) {
	a, b, s := dosEquipos(t)

	if err := a.b.PonerProyecto(Proyecto{Ref: "aaaa000000000001", Nombre: "Acme", Usado: "2026-10-02T09:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := b.b.PonerProyecto(Proyecto{Ref: "bbbb000000000002", Nombre: "Beta", Usado: "2026-10-02T09:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	alDia(t, a, b, s)

	// Los dos equipos tienen los dos proyectos.
	for _, e := range []*equipo{a, b} {
		if n := len(e.b.Proyectos()); n != 2 {
			t.Fatalf("%s tiene %d proyectos y tenía que tener 2: %+v", e.nombre, n, e.b.Proyectos())
		}
	}

	// A le cambia el nombre a uno; B lo abre, o sea que le toca la fecha de uso.
	acme, _ := a.b.Proyecto("aaaa000000000001")
	acme.Nombre = "Acme S. A."
	if err := a.b.PonerProyecto(acme); err != nil {
		t.Fatal(err)
	}
	suyo, _ := b.b.Proyecto("aaaa000000000001")
	suyo.Usado = "2026-10-02T11:00:00Z"
	if err := b.b.PonerProyecto(suyo); err != nil {
		t.Fatal(err)
	}
	alDia(t, a, b, s)

	for _, e := range []*equipo{a, b} {
		p, hay := e.b.Proyecto("aaaa000000000001")
		if !hay {
			t.Fatalf("%s ha perdido el proyecto", e.nombre)
		}
		if p.Nombre != "Acme S. A." {
			t.Errorf("%s ve el nombre %q y el que se cambió es «Acme S. A.»", e.nombre, p.Nombre)
		}
		// Las dos cosas son verdad a la vez: el nombre nuevo y la última apertura.
		if p.Usado != "2026-10-02T11:00:00Z" {
			t.Errorf("%s ve usado %q y la última vez fue a las 11", e.nombre, p.Usado)
		}
	}

	// Y lo que un equipo olvida se va de los dos, que es lo que distingue una
	// fusión a tres bandas de juntar dos listas.
	if err := a.b.OlvidarProyecto("bbbb000000000002"); err != nil {
		t.Fatal(err)
	}
	alDia(t, a, b, s)
	for _, e := range []*equipo{a, b} {
		if _, hay := e.b.Proyecto("bbbb000000000002"); hay {
			t.Errorf("%s resucita un proyecto olvidado", e.nombre)
		}
		if n := len(e.b.Proyectos()); n != 1 {
			t.Errorf("%s se queda con %d proyectos y tenía que quedar 1", e.nombre, n)
		}
	}
}

// alDia pasa la sincronización hasta que los dos equipos dejan de tener nada que
// decir. Dos pasadas por equipo bastan —subir lo propio y bajar lo del otro— y se
// repite por si la fusión deja algo que subir.
func alDia(t *testing.T, a, b *equipo, s *servidorFalso) {
	t.Helper()
	for i := 0; i < 4; i++ {
		a.sincronizar(t, s)
		b.sincronizar(t, s)
	}
}

// Sin bóveda personal abierta no se crea ni se abre nada, y lo dice con ErrCerrada
// en vez de intentarlo con una clave vacía.
func TestSinLaPersonalNoHayProyecto(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "p.esfinge")
	if _, err := CrearProyecto(ruta, nil); err != ErrCerrada {
		t.Fatalf("crear sin la personal da %v", err)
	}
	if _, err := AbrirProyecto(ruta, nil); err != ErrCerrada {
		t.Fatalf("abrir sin la personal da %v", err)
	}
}

// **Entregar una bóveda: lo que se lleva y, sobre todo, lo que no** (ADR 0051).
//
// Cada aserción de aquí es un paso que, si se olvida, entrega algo que no debía
// salir. La que más duele es la identidad: es la semilla con la que se firman los
// envíos compartidos, así que regalarla es regalar la firma de quien entrega.
func TestDesprenderUnaBovedaSeLlevaLoDeDentroYNadaMas(t *testing.T) {
	personal, ruta, llave := personalYProyecto(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	// Se le pone de todo lo que no debe salir: identidad, una copia esperando y
	// una lápida de algo borrado.
	if _, err := p.Identidad(); err != nil {
		t.Fatal(err)
	}
	e := p.Buscar("Acme")[0]
	if err := p.Borrar(e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := p.VaciarPapelera(); err != nil {
		t.Fatal(err)
	}
	if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Correo de Acme", Secreto: "la del correo"}); err != nil {
		t.Fatal(err)
	}
	if err := p.AnotarPendiente("x", "a@b.c", "AAAA-BBBB"); err != nil {
		t.Fatal(err)
	}

	const nueva = "la maestra del cliente, bien larga"
	copia, recuperacion, err := p.Desprender(nueva)
	if err != nil {
		t.Fatal(err)
	}

	// **Lo que se lleva**: las entradas con su secreto. Es para lo que se entrega.
	if l := copia.Buscar("Correo"); len(l) != 1 {
		t.Fatalf("la copia tiene %d entradas", len(l))
	}
	entera, _ := copia.Ver(copia.Buscar("Correo")[0].ID)
	if entera.Secreto != "la del correo" {
		t.Errorf("el secreto ha llegado como %q", entera.Secreto)
	}

	// **Y lo que no:**
	if _, err := copia.Identidad(); err == nil {
		t.Error("la copia se lleva la identidad: eso es regalar la firma de quien la entrega")
	}
	if n := len(copia.Pendientes()); n != 0 {
		t.Errorf("la copia se lleva %d copias que esperaban", n)
	}
	if copia.ID() == p.ID() {
		t.Error("la copia tiene el identificador de la original: la sincronización las confundiría")
	}
	if copia.TieneRanuraPrincipal() {
		t.Error("la copia sigue abriéndose con la bóveda personal de quien la entrega")
	}
	if _, err := AbrirProyecto(rutaDeLaCopia(t, copia), llave); err != ErrSinRanuraPrincipal {
		t.Error("y el fichero entregado se abre con la bóveda personal de quien lo entregó")
	}

	// **Se abre con su maestra nueva y con su recuperación**, y con nada más.
	fichero := rutaDeLaCopia(t, copia)
	if _, err := Abrir(fichero, nueva); err != nil {
		t.Fatalf("la copia no abre con su contraseña nueva: %v", err)
	}
	if _, err := Abrir(fichero, recuperacion); err != nil {
		t.Fatalf("la copia no abre con su clave de recuperación: %v", err)
	}
	if _, err := Abrir(fichero, maestraDePrueba); err == nil {
		t.Error("la copia abre con la contraseña de quien la entregó")
	}

	// Y **la original no se ha tocado**: sigue con lo suyo y con su identidad.
	if _, err := p.Identidad(); err != nil {
		t.Errorf("desprender se ha llevado la identidad de la original: %v", err)
	}
	if n := len(p.Pendientes()); n != 1 {
		t.Errorf("la original tiene %d copias esperando y tenía una", n)
	}
	_ = personal
}

func rutaDeLaCopia(t *testing.T, b *Boveda) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "entregada.esfinge")
	if err := b.GuardarEn(ruta); err != nil {
		t.Fatal(err)
	}
	return ruta
}
