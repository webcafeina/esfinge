package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/llavero"
)

// unServidorDePruebas deja una bóveda con una entrada, la llave del servidor
// apuntando a un sitio de la prueba, y la maestra en un fichero para poder dársela
// a los comandos que la piden.
func unServidorDePruebas(t *testing.T) (ruta, conLaMaestra string) {
	t.Helper()
	const maestra = "una contraseña maestra bien larga"
	dir := t.TempDir()
	ruta = filepath.Join(dir, "boveda.esfinge")
	b, _, err := boveda.Crear(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Poner(boveda.Entrada{Tipo: boveda.TipoCredencial, Titulo: "Cloudflare", Secreto: "el testigo"}); err != nil {
		t.Fatal(err)
	}
	b.Cerrar()

	// **La llave fuera de la carpeta de la bóveda**, que es la regla de la ADR 0044
	// y aquí también se cumple en las pruebas: `t.TempDir()` da una carpeta nueva
	// cada vez.
	t.Setenv(llavero.VariableDeLaLlave, filepath.Join(t.TempDir(), "llave"))

	conLaMaestra = filepath.Join(dir, "maestra.txt")
	if err := os.WriteFile(conLaMaestra, []byte(maestra), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta, conLaMaestra
}

// loQueSaleAlError recoge lo que un comando escribe por la salida de error, que es
// por donde va todo lo que se le dice a quien mira.
func loQueSaleAlError(t *testing.T, hacer func()) string {
	t.Helper()
	antes := os.Stderr
	leer, escribir, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = escribir
	hecho := make(chan string, 1)
	go func() {
		todo, _ := io.ReadAll(leer)
		hecho <- string(todo)
	}()
	hacer()
	os.Stderr = antes
	_ = escribir.Close()
	salida := <-hecho
	_ = leer.Close()
	return salida
}

// La bóveda se abre en un servidor sin contraseña, **y solo cuando se ha pedido**
// (ADR 0057): se activa, abre sola, y al quitarlo vuelve a pedir la maestra.
//
// Lo que de verdad vigila es el primer paso y el último: **antes de activar no abre
// nada**. Una prueba que solo mire el medio pasaría igual con esto encendido de
// fábrica, que es lo contrario de lo que dice la ADR.
func TestEnUnServidorLaBovedaSeAbreSinContrasenaYSoloSiSePide(t *testing.T) {
	ruta, conLaMaestra := unServidorDePruebas(t)
	o := &opciones{tema: "oscuro", silencio: true}

	sinClave := &opcionesBoveda{ruta: ruta}
	if b, err := abrirBoveda(o, sinClave); err == nil {
		b.Cerrar()
		t.Fatal("la bóveda se abre sin contraseña sin que nadie lo haya activado")
	}

	conClave := &opcionesBoveda{ruta: ruta}
	conClave.clave.Fichero = conLaMaestra
	dicho := loQueSaleAlError(t, func() {
		if err := comandoServidorActivar(o, conClave).RunE(nil, nil); err != nil {
			t.Fatal(err)
		}
	})
	// **Se dice lo que cuesta donde se activa**, no solo en la documentación.
	if !strings.Contains(dicho, "quien entre en él") {
		t.Errorf("al activar no se dice qué no protege:\n%s", dicho)
	}

	// Y ahora abre sin que nadie teclee nada, con lo de dentro intacto.
	b, err := abrirBoveda(o, sinClave)
	if err != nil {
		t.Fatalf("después de activar sigue pidiendo contraseña: %v", err)
	}
	if e := b.Buscar("Cloudflare"); len(e) != 1 {
		t.Fatalf("abierta con la llave del servidor hay %d entradas", len(e))
	}
	if !b.TieneRanuraDelServidor() {
		t.Fatal("la bóveda abierta no dice tener la ranura del servidor")
	}
	b.Cerrar()

	// La maestra sigue abriendo: esta ranura nunca es la única.
	if b, err := abrirBoveda(o, conClave); err != nil {
		t.Fatalf("la maestra ha dejado de abrir: %v", err)
	} else {
		b.Cerrar()
	}

	// Quitar **no pide la maestra**: lo hace con la llave que ya hay, porque solo
	// cierra una puerta que en el servidor ya estaba abierta.
	if err := comandoServidorQuitar(o, sinClave).RunE(nil, nil); err != nil {
		t.Fatal(err)
	}
	if b, err := abrirBoveda(o, sinClave); err == nil {
		b.Cerrar()
		t.Fatal("sigue abriendo sin contraseña después de quitarlo")
	}
	if boveda.RanuraDelServidorEn(ruta) {
		t.Fatal("la ranura sigue en el fichero")
	}
	// Y la llave se va del disco: una llave que no abre nada no tiene por qué
	// quedarse ahí.
	if llavero.DelServidor().Hay() {
		t.Error("la llave sigue en el disco después de quitarla")
	}
}

// **Una clave dicha a mano manda sobre la llave del servidor.** Quien pasa
// --clave-fichero está diciendo con qué quiere abrir, y lo contrario haría que en
// un servidor no se pudiera comprobar una contraseña nunca.
func TestLaClaveDichaAManoGanaALaLlaveDelServidor(t *testing.T) {
	ruta, conLaMaestra := unServidorDePruebas(t)
	o := &opciones{tema: "oscuro", silencio: true}
	conClave := &opcionesBoveda{ruta: ruta}
	conClave.clave.Fichero = conLaMaestra
	if err := comandoServidorActivar(o, conClave).RunE(nil, nil); err != nil {
		t.Fatal(err)
	}

	malaClave := filepath.Join(t.TempDir(), "mala.txt")
	if err := os.WriteFile(malaClave, []byte("ésta no es la maestra"), 0o600); err != nil {
		t.Fatal(err)
	}
	conOtra := &opcionesBoveda{ruta: ruta}
	conOtra.clave.Fichero = malaClave
	if b, err := abrirBoveda(o, conOtra); err == nil {
		b.Cerrar()
		t.Fatal("con una clave equivocada ha abierto con la llave del servidor, así que esa clave no se comprueba nunca")
	}
}

// **La llave del servidor va en la bóveda personal, no en una de proyecto** (ADR
// 0050), y eso se comprueba **antes de pedir la contraseña**.
//
// Es la regla de `ExportarLlaves`: lo que hace falta para decidir se mira antes de
// pedirle a alguien que decida. La prueba lo fuerza no dándole ninguna clave: si el
// orden fuera el otro, lo que saldría es que no hay clave, y nadie se enteraría de
// cuál era el problema de verdad.
func TestActivarEnUnProyectoSeNiegaAntesDePedirNada(t *testing.T) {
	dir := t.TempDir()
	personal, _, err := boveda.Crear(filepath.Join(dir, "boveda.esfinge"), "una contraseña maestra bien larga")
	if err != nil {
		t.Fatal(err)
	}
	rutaProyecto := filepath.Join(dir, "abcdef0123456789.esfinge")
	p, err := boveda.CrearProyecto(rutaProyecto, personal.LlaveParaProyectos())
	if err != nil {
		t.Fatal(err)
	}
	p.Cerrar()
	t.Setenv(llavero.VariableDeLaLlave, filepath.Join(t.TempDir(), "llave"))

	o := &opciones{tema: "oscuro", silencio: true}
	err = comandoServidorActivar(o, &opcionesBoveda{ruta: rutaProyecto}).RunE(nil, nil)
	if err == nil {
		t.Fatal("ha activado la llave del servidor en la bóveda de un proyecto")
	}
	if !strings.Contains(err.Error(), "proyecto") {
		t.Fatalf("se ha quejado de otra cosa, así que el orden es el contrario: %v", err)
	}
	if llavero.DelServidor().Hay() {
		t.Error("ha dejado una llave en el disco antes de negarse")
	}
}
