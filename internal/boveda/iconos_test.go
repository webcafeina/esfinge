package boveda

import (
	"os"
	"strings"
	"testing"
	"time"
)

// La caché de iconos va cifrada, y eso no es celo: **el fichero en claro sería
// la lista de sitios de la bóveda**, que es justo lo que la bóveda oculta.
func TestLaCacheDeIconosNoDiceQueSitiosHay(t *testing.T) {
	b, _, ruta := nueva(t)

	if err := b.PonerIconos(map[string]Icono{
		"banco-secreto.es":    {URI: "data:image/png;base64,Zq9wXk7", Mirado: ahoraRFC()},
		"hacienda.gob.es":     {URI: "data:image/png;base64,Vt3mLp2", Mirado: ahoraRFC()},
		"sitio-sin-icono.com": {Mirado: ahoraRFC()},
	}); err != nil {
		t.Fatal(err)
	}

	crudo, err := os.ReadFile(RutaDeIconos(ruta))
	if err != nil {
		t.Fatal(err)
	}
	// Los dominios son lo que de verdad hay que ocultar; el contenido del icono es
	// público. Se comprueban los dos de todas formas, con marcadores que no puedan
	// salir por casualidad en un base64 —«AAAA» sale a cada rato y daba un falso
	// positivo—.
	for _, secreto := range []string{"banco-secreto", "hacienda", "sitio-sin-icono", "Zq9wXk7"} {
		if strings.Contains(string(crudo), secreto) {
			t.Errorf("«%s» se lee en claro en el fichero de iconos", secreto)
		}
	}
	if !strings.HasPrefix(string(crudo), "ESF1.") {
		t.Errorf("no es un contenedor de Esfinge: %.20s", crudo)
	}

	// Y se vuelve a leer entero.
	todos := b.Iconos()
	if len(todos) != 3 {
		t.Fatalf("han vuelto %d de 3", len(todos))
	}
	if todos["banco-secreto.es"].URI != "data:image/png;base64,Zq9wXk7" {
		t.Errorf("no ha vuelto igual: %+v", todos["banco-secreto.es"])
	}
}

// Se mezcla con lo que hubiera en vez de sustituirlo: una tanda nueva no puede
// llevarse por delante lo que trajo la anterior.
func TestPonerIconosNoBorraLosDeAntes(t *testing.T) {
	b, _, _ := nueva(t)
	if err := b.PonerIconos(map[string]Icono{"uno.es": {URI: "x", Mirado: ahoraRFC()}}); err != nil {
		t.Fatal(err)
	}
	if err := b.PonerIconos(map[string]Icono{"dos.es": {URI: "y", Mirado: ahoraRFC()}}); err != nil {
		t.Fatal(err)
	}
	todos := b.Iconos()
	if len(todos) != 2 || todos["uno.es"].URI != "x" || todos["dos.es"].URI != "y" {
		t.Errorf("%+v", todos)
	}
}

// **Recordar el fracaso es la mitad del diseño.** Sin esto, los sitios sin icono
// se vuelven a preguntar en cada arranque, para siempre.
func TestUnSitioSinIconoNoSePreguntaCadaVez(t *testing.T) {
	ahora := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

	// Recién preguntado y sin icono: no se vuelve a preguntar.
	reciente := Icono{Mirado: ahora.Add(-24 * time.Hour).Format(time.RFC3339)}
	if TocaMirar(reciente, ahora) {
		t.Error("vuelve a preguntar por un sitio que ayer no tenía icono")
	}
	// Pasado el plazo, sí: un sitio puede estrenar icono.
	viejo := Icono{Mirado: ahora.Add(-60 * 24 * time.Hour).Format(time.RFC3339)}
	if !TocaMirar(viejo, ahora) {
		t.Error("no vuelve a preguntar ni después de dos meses")
	}
	// Con icono, nunca.
	if TocaMirar(Icono{URI: "x", Mirado: viejo.Mirado}, ahora) {
		t.Error("vuelve a preguntar por uno que ya tiene icono")
	}
	// Y por uno que no se ha mirado nunca, sí.
	if !TocaMirar(Icono{}, ahora) {
		t.Error("no pregunta por uno que no se ha mirado nunca")
	}
}

// La caché es prescindible: si no se puede leer, se empieza de cero y no pasa
// nada. Nada de lo que hay dentro es insustituible.
func TestUnaCacheRotaNoRompeNada(t *testing.T) {
	b, _, ruta := nueva(t)
	if err := b.PonerIconos(map[string]Icono{"uno.es": {URI: "x", Mirado: ahoraRFC()}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(RutaDeIconos(ruta), []byte("esto no es nada"), 0o600); err != nil {
		t.Fatal(err)
	}
	if todos := b.Iconos(); len(todos) != 0 {
		t.Errorf("una caché rota ha devuelto %d iconos", len(todos))
	}
	// Y se puede volver a llenar encima.
	if err := b.PonerIconos(map[string]Icono{"dos.es": {URI: "y", Mirado: ahoraRFC()}}); err != nil {
		t.Fatal(err)
	}
	if b.Iconos()["dos.es"].URI != "y" {
		t.Error("no se ha podido volver a llenar")
	}
}

func ahoraRFC() string { return time.Now().UTC().Format(time.RFC3339) }
