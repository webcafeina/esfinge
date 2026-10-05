package boveda

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// relojQueAvanza para el reloj de la bóveda donde se diga y devuelve con qué
// moverlo.
//
// Hace falta porque **las fechas del formato van a segundos**: dar un acceso y
// quitarlo en la misma prueba cae en el mismo segundo, y entonces lo que se estaría
// probando es el empate de `fundirSobres` y no el orden. Las dos cosas se prueban,
// pero cada una en la suya.
func relojQueAvanza(t *testing.T) func(segundos int) {
	t.Helper()
	antes := ahora
	actual := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	ahora = func() time.Time { return actual }
	t.Cleanup(func() { ahora = antes })
	return func(segundos int) { actual = actual.Add(time.Duration(segundos) * time.Second) }
}

// unaPersonal hace la bóveda personal de otra persona, con su identidad ya creada:
// es lo que hace falta para darle acceso a algo.
func unaPersonal(t *testing.T, nombre string) (*Boveda, Identidad) {
	t.Helper()
	b, _, err := Crear(filepath.Join(t.TempDir(), nombre+".esfinge"), maestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	id, err := b.Identidad()
	if err != nil {
		t.Fatal(err)
	}
	return b, id
}

// Lo primero: que la ranura sellada abra, y que **solo la abra quien debe**.
func TestUnAccesoLoAbreSuDuenoYNadieMas(t *testing.T) {
	_, ruta, llave := personalYProyecto(t)
	ana, idAna := unaPersonal(t, "ana")
	beto, _ := unaPersonal(t, "beto")

	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PonerAcceso("1111222233334444", idAna); err != nil {
		t.Fatal(err)
	}

	// Ana abre con su personal, sin saber ninguna contraseña de esta bóveda.
	deAna, err := AbrirCompartida(ruta, "1111222233334444", ana)
	if err != nil {
		t.Fatalf("quien tiene acceso no abre: %v", err)
	}
	if e := deAna.Buscar("Acme"); len(e) != 1 {
		t.Fatalf("Ana ve %d entradas y hay una", len(e))
	}

	// Beto no, y es la mitad que importa: la ranura está en el fichero que él
	// también podría tener, y lo que lo para es que está sellada hacia Ana.
	if _, err := AbrirCompartida(ruta, "1111222233334444", beto); err == nil {
		t.Fatal("la ranura de Ana la ha abierto Beto")
	}

	// Y el dueño sigue abriendo por la suya: dar acceso no se lleva nada.
	if _, err := AbrirProyecto(ruta, llave); err != nil {
		t.Fatalf("el dueño ha dejado de abrir su propio proyecto: %v", err)
	}
}

// **Tres accesos a la vez**, que es lo que demuestra que el tipo por titular no es
// estilo: con un tipo fijo, el sello indexa por tipo y la bóveda deja de abrir.
func TestVariosAccesosConvivenEnLaMismaBoveda(t *testing.T) {
	_, ruta, llave := personalYProyecto(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}

	gente := map[string]*Boveda{}
	for _, n := range []string{"aaaa0000aaaa0000", "bbbb1111bbbb1111", "cccc2222cccc2222"} {
		b, id := unaPersonal(t, n)
		if err := p.PonerAcceso(n, id); err != nil {
			t.Fatal(err)
		}
		gente[n] = b
	}

	if ids := p.Accesos(); len(ids) != 3 {
		t.Fatalf("la bóveda dice tener %d accesos y se le han puesto tres: %v", len(ids), ids)
	}
	for n, b := range gente {
		if _, err := AbrirCompartida(ruta, n, b); err != nil {
			t.Errorf("%s no puede abrir: %v", n, err)
		}
	}
	// Y sin abrirla, que es lo que necesita la lista de proyectos.
	if ids := AccesosEn(ruta); len(ids) != 3 {
		t.Errorf("sin abrir el fichero se ven %d accesos: %v", len(ids), ids)
	}
}

// Quitar el acceso: deja de abrir, y **la ranura se queda como lápida**.
func TestQuitarElAccesoDejaLapidaYCierraLaPuerta(t *testing.T) {
	_, ruta, llave := personalYProyecto(t)
	ana, idAna := unaPersonal(t, "ana")
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PonerAcceso("1111222233334444", idAna); err != nil {
		t.Fatal(err)
	}
	if err := p.RetirarAcceso("1111222233334444"); err != nil {
		t.Fatal(err)
	}

	if _, err := AbrirCompartida(ruta, "1111222233334444", ana); err == nil {
		t.Fatal("se le ha quitado el acceso y sigue abriendo")
	}
	if ids := p.Accesos(); len(ids) != 0 {
		t.Errorf("la bóveda sigue contando %v como acceso", ids)
	}

	// **Y la lápida está en el fichero.** Es lo que no se ve mirando la pantalla y
	// lo que hace que quitar el acceso no se deshaga solo: el sobre no se borra.
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(datos), TipoDeAcceso("1111222233334444")) {
		t.Fatal("la ranura ha desaparecido del fichero en vez de quedarse como lápida")
	}
	if !strings.Contains(string(datos), CodificacionRetirada) {
		t.Fatal("la ranura sigue en el fichero pero no está marcada como retirada")
	}
}

// **La prueba que no existiría sin haber leído `fundirSobres`.**
//
// Quitar una ranura no es representable en la fusión: une por tipo y conserva la que
// está en un solo lado. Así que sin lápida, el dueño quita el acceso, sube, y el
// primer equipo con una copia de antes **lo devuelve**.
func TestQuitarElAccesoAguantaUnaCopiaDeAntes(t *testing.T) {
	avanzar := relojQueAvanza(t)
	_, ruta, llave := personalYProyecto(t)
	ana, idAna := unaPersonal(t, "ana")
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PonerAcceso("1111222233334444", idAna); err != nil {
		t.Fatal(err)
	}

	// La copia de antes: lo que tiene el equipo que lleva días sin abrir Esfinge.
	deAntes, _, err := p.PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}

	// Pasa un rato y se le quita el acceso.
	avanzar(60)
	if err := p.RetirarAcceso("1111222233334444"); err != nil {
		t.Fatal(err)
	}

	// Y ahora llega la copia de antes, como llega cualquier cosa del servidor.
	if _, err := p.Fundir(deAntes, 1, nil, OpcionesDeFusion{}); err != nil {
		t.Fatalf("fundir con la copia de antes: %v", err)
	}

	if ids := p.Accesos(); len(ids) != 0 {
		t.Fatalf("la ranura del revocado ha vuelto al fundir con una copia de antes: %v", ids)
	}
	if _, err := AbrirCompartida(ruta, "1111222233334444", ana); err == nil {
		t.Fatal("tras fundir con una copia de antes, el revocado vuelve a abrir")
	}
}

// Y la vuelta: **volver a dar el acceso después de quitarlo manda**, o la lápida
// sería una puerta cerrada para siempre.
func TestVolverADarElAccesoGanaALaLapida(t *testing.T) {
	avanzar := relojQueAvanza(t)
	_, ruta, llave := personalYProyecto(t)
	ana, idAna := unaPersonal(t, "ana")
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PonerAcceso("1111222233334444", idAna); err != nil {
		t.Fatal(err)
	}
	if err := p.RetirarAcceso("1111222233334444"); err != nil {
		t.Fatal(err)
	}
	conLaLapida, _, err := p.PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}

	avanzar(60)
	if err := p.PonerAcceso("1111222233334444", idAna); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Fundir(conLaLapida, 1, nil, OpcionesDeFusion{}); err != nil {
		t.Fatalf("fundir con la versión que traía la lápida: %v", err)
	}

	if _, err := AbrirCompartida(ruta, "1111222233334444", ana); err != nil {
		t.Fatalf("se le ha vuelto a dar el acceso y la lápida se lo come: %v", err)
	}
}

// Lo que se sube lleva las ranuras de acceso, por lo mismo que lleva la del dueño:
// sin ellas, quien tiene acceso se baja una bóveda que no puede abrir.
func TestLasRanurasDeAccesoSeSuben(t *testing.T) {
	_, ruta, llave := personalYProyecto(t)
	ana, idAna := unaPersonal(t, "ana")
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PonerAcceso("1111222233334444", idAna); err != nil {
		t.Fatal(err)
	}

	datos, _, err := p.PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}
	// Y lo que se sube **abre de verdad**: que el nombre esté en el JSON no dice que
	// el sobre siga sellando lo mismo.
	if _, err := AbrirCompartidaBytes("", datos, "1111222233334444", ana); err != nil {
		t.Fatalf("lo que se sube no lo abre quien tiene acceso: %v", err)
	}
}

// Sellar y abrir, suelto: el secreto vuelve entero y **no lo abre otra identidad**.
func TestSellarHaciaUnaIdentidadYAbrirlo(t *testing.T) {
	ana, idAna := unaPersonal(t, "ana")
	beto, _ := unaPersonal(t, "beto")

	sellado, err := SellarHaciaIdentidad([]byte("esto es una clave de bóveda"), idAna)
	if err != nil {
		t.Fatal(err)
	}
	// No se parece a un contenedor ESF1 ni se puede confundir con uno.
	if !strings.HasPrefix(sellado, marcaDelSellado+".") {
		t.Fatalf("un sellado tiene que decir qué es: %.20s", sellado)
	}

	claro, err := ana.AbrirSellado(sellado)
	if err != nil {
		t.Fatal(err)
	}
	if string(claro) != "esto es una clave de bóveda" {
		t.Fatalf("lo que vuelve es %q", claro)
	}
	if _, err := beto.AbrirSellado(sellado); err == nil {
		t.Fatal("un sellado para Ana lo ha abierto Beto")
	}
}
