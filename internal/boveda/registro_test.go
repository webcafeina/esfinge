package boveda

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func conRegistro(t *testing.T) *Boveda {
	t.Helper()
	b, _, err := Crear(t.TempDir()+"/r.esfinge", "una contraseña maestra larga de prueba")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Lo que se apunta se lee, **lo último arriba**, y no lleva ni un secreto.
func TestElRegistroApuntaYSeLee(t *testing.T) {
	b := conRegistro(t)
	for _, a := range []Apunte{
		{Quien: "Claude Code", Que: "copiar-secreto", Sobre: "a1", Titulo: "GitHub",
			Resultado: ApunteHecho, Como: ApuntePreguntado, Cuando: "2026-10-07T10:00:00Z"},
		{Quien: "Claude Code", Que: "copiar-secreto", Sobre: "b2", Titulo: "Banco",
			Resultado: ApunteNegado, Cuando: "2026-10-07T11:00:00Z"},
	} {
		if err := b.Apuntar(a); err != nil {
			t.Fatal(err)
		}
	}
	r := b.Registro()
	if len(r) != 2 {
		t.Fatalf("hay %d apuntes", len(r))
	}
	if r[0].Titulo != "Banco" {
		t.Errorf("lo último no está arriba: %+v", r)
	}
	// **Cada apunte con su identificador**, que es lo que deja fundir sin pisar.
	if r[0].ID == "" || r[0].ID == r[1].ID {
		t.Errorf("los identificadores son %q y %q", r[0].ID, r[1].ID)
	}
	// **Y lo negado se apunta**, que es la mitad interesante: «pidió la contraseña del
	// banco y se le dijo que no» es la señal por la que esto existe.
	if r[0].Resultado != ApunteNegado {
		t.Errorf("lo negado no se ha apuntado como tal: %+v", r[0])
	}
}

// **El registro se purga al abrir**, como la papelera y las lápidas, y por lo mismo: una
// bóveda cerrada no ejecuta nada, así que un reloj solo contaría mientras la aplicación
// estuviera puesta y el plazo dependería de cuánto la usa cada uno.
func TestElRegistroSePurgaAlAbrir(t *testing.T) {
	ruta := t.TempDir() + "/r.esfinge"
	b, _, err := Crear(ruta, "una contraseña maestra larga de prueba")
	if err != nil {
		t.Fatal(err)
	}
	viejo := time.Now().Add(-PlazoDelRegistro - 48*time.Hour).UTC().Format(time.RFC3339)
	if err := b.Apuntar(Apunte{Quien: "X", Que: "copiar-secreto", Resultado: ApunteHecho, Cuando: viejo}); err != nil {
		t.Fatal(err)
	}
	if err := b.Apuntar(Apunte{Quien: "X", Que: "copiar-secreto", Resultado: ApunteHecho}); err != nil {
		t.Fatal(err)
	}
	b.Cerrar()

	otra, err := Abrir(ruta, "una contraseña maestra larga de prueba")
	if err != nil {
		t.Fatal(err)
	}
	r := otra.Registro()
	if len(r) != 1 {
		t.Fatalf("al abrir quedan %d apuntes y tenía que quedar uno: %+v", len(r), r)
	}
	if r[0].Cuando == viejo {
		t.Error("el que se ha quedado es el viejo")
	}
}

// El tope: al llegar se va el más viejo, y **el cuerpo no crece sin fin**. Eso importa
// porque el cuerpo se sube entero en cada sincronización.
func TestElRegistroTieneTope(t *testing.T) {
	b := conRegistro(t)
	for i := 0; i < TopeDelRegistro+5; i++ {
		if err := b.Apuntar(Apunte{Quien: "X", Que: "copiar-secreto", Resultado: ApunteHecho}); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(b.Registro()); n != TopeDelRegistro {
		t.Errorf("hay %d apuntes y el tope son %d", n, TopeDelRegistro)
	}
}

// **Fundir no pierde ninguno ni duplica**, y da el mismo orden en los dos lados aunque
// dos apuntes compartan el segundo — que es lo que el desempate por identificador
// resuelve.
func TestFundirElRegistroNoPierdeNiDuplica(t *testing.T) {
	mismoSegundo := "2026-10-07T10:00:00Z"
	uno := []Apunte{
		{ID: "bbbb", Cuando: mismoSegundo, Quien: "A", Que: "copiar-secreto", Resultado: ApunteHecho},
		{ID: "aaaa", Cuando: mismoSegundo, Quien: "A", Que: "borrar", Resultado: ApunteHecho},
	}
	otro := []Apunte{
		{ID: "aaaa", Cuando: mismoSegundo, Quien: "A", Que: "borrar", Resultado: ApunteHecho},
		{ID: "cccc", Cuando: "2026-10-07T11:00:00Z", Quien: "B", Que: "copiar-secreto", Resultado: ApunteNegado},
	}
	r := fundirRegistro(uno, otro)
	if len(r) != 3 {
		t.Fatalf("la fusión da %d apuntes y tenían que ser tres: %+v", len(r), r)
	}
	// **Y al revés da lo mismo**: si no, dos equipos se pasarían la bóveda sin fin.
	alReves := fundirRegistro(otro, uno)
	a, _ := json.Marshal(r)
	bb, _ := json.Marshal(alReves)
	if string(a) != string(bb) {
		t.Errorf("fundir al revés da otra cosa:\n  %s\n  %s", a, bb)
	}
	// Y los dos del mismo segundo salen por identificador.
	if r[0].ID != "aaaa" || r[1].ID != "bbbb" {
		t.Errorf("el desempate del mismo segundo da %q, %q", r[0].ID, r[1].ID)
	}
}

// Y la regla absoluta: **lo del registro no sale de la bóveda**. Esta prueba no puede
// comprobar lo que no pasa, así que comprueba lo que sí: que un apunte con un título
// dentro **viaja cifrado**, o sea que ese título no está en el fichero en claro.
func TestElRegistroViajaDentroDelCuerpoCifrado(t *testing.T) {
	ruta := t.TempDir() + "/r.esfinge"
	b, _, err := Crear(ruta, "una contraseña maestra larga de prueba")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Apuntar(Apunte{
		Quien: "Claude Code", Que: "copiar-secreto", Sobre: "a1",
		Titulo: "HACIENDA-ES-EL-TITULO", Resultado: ApunteHecho,
	}); err != nil {
		t.Fatal(err)
	}
	crudo, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(crudo), "HACIENDA-ES-EL-TITULO") {
		t.Error("el título de lo apuntado está en claro en el fichero")
	}
	if strings.Contains(string(crudo), "Claude Code") {
		t.Error("quién lo pidió está en claro en el fichero")
	}
}
