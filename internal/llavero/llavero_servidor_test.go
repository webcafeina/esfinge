package llavero

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func enUnFicheroDePruebas(t *testing.T) EnFichero {
	t.Helper()
	// Por la variable, que es como se le dice al servidor que la llave está en otro
	// sitio: así la prueba ejercita de paso el camino que usa quien la mueve.
	ruta := filepath.Join(t.TempDir(), "llave-del-servidor")
	t.Setenv(VariableDeLaLlave, ruta)
	f := DelServidor()
	if f.Ruta != ruta {
		t.Fatalf("la ruta es %q y la variable decía %q", f.Ruta, ruta)
	}
	return f
}

func TestLaLlaveDelServidorSeGuardaSeLeeYSeRota(t *testing.T) {
	f := enUnFicheroDePruebas(t)

	if f.Hay() {
		t.Fatal("dice que hay llave antes de guardar ninguna")
	}
	if _, err := f.Leer("una", "da igual"); err != ErrNoHay {
		t.Fatalf("sin fichero, leer devuelve %v", err)
	}

	secreto := bytes.Repeat([]byte{0xab}, 32)
	if err := f.Guardar("una", secreto); err != nil {
		t.Fatal(err)
	}
	if !f.Hay() {
		t.Fatal("no dice que haya llave después de guardarla")
	}
	leido, err := f.Leer("una", "da igual")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leido, secreto) {
		t.Fatalf("lo leído no es lo guardado: %x", leido)
	}
	// Otro identificador no es el mismo secreto, y lo dice distinto de «no hay
	// fichero»: una llave de otra bóveda no puede leerse como la de ésta.
	if _, err := f.Leer("otra", "da igual"); err != ErrNoEsta {
		t.Fatalf("con otro identificador devuelve %v", err)
	}

	// **0600**: lo que guarda esto abre una bóveda entera.
	info, err := os.Stat(f.Ruta)
	if err != nil {
		t.Fatal(err)
	}
	if p := info.Mode().Perm(); p != 0o600 {
		t.Fatalf("el fichero de la llave está a %o", p)
	}

	// Rotar es volver a guardar, y la de antes deja de estar.
	nuevo := bytes.Repeat([]byte{0xcd}, 32)
	if err := f.Guardar("una", nuevo); err != nil {
		t.Fatal(err)
	}
	leido, _ = f.Leer("una", "")
	if !bytes.Equal(leido, nuevo) {
		t.Fatalf("después de rotar sale %x", leido)
	}

	// Y al borrar la última se va el fichero: uno vacío con ese nombre haría creer
	// que ahí hay una llave.
	if err := f.Borrar("una"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.Ruta); !os.IsNotExist(err) {
		t.Fatalf("el fichero sigue ahí sin ninguna llave dentro: %v", err)
	}
	if f.Hay() {
		t.Fatal("dice que hay llave después de borrarla")
	}
}

// **La llave del servidor no la ofrece `Del()`.**
//
// Es la mitad de la ADR 0057 que no está en el código del fichero sino en que
// nadie lo enchufe: en un Linux de escritorio sigue sin haber llavero —con el
// porqué escrito en `llavero_otros.go`— y un fichero en el disco que apareciera
// solo como «desbloqueo del sistema» sería el cerrojo sin cerradura que allí se
// descartó. Hay que pedirlo, con una orden, una vez.
func TestElLlaveroDelSistemaNoEsElDelServidor(t *testing.T) {
	f := enUnFicheroDePruebas(t)
	if err := f.Guardar("una", bytes.Repeat([]byte{1}, 32)); err != nil {
		t.Fatal(err)
	}

	l := Del()
	if l.Hay() {
		// En un Mac o en un Windows esto puede ser verdad por su cuenta; lo que no
		// puede ser es que lo sea **por el fichero**.
		if _, err := l.Leer("una", "da igual"); err == nil {
			t.Fatal("el llavero del sistema está leyendo la llave del servidor")
		}
		t.Skip("este equipo tiene llavero del sistema de verdad")
	}
	if _, err := l.Leer("una", "da igual"); err != ErrNoHay {
		t.Fatalf("con la llave del servidor puesta, el llavero del sistema devuelve %v", err)
	}
}

// Un fichero roto cuenta como que no hay llave, y no revienta: de ahí no sale
// ningún secreto, y lo que toca es volver a activarlo.
func TestUnFicheroDeLlaveRotoNoRevienta(t *testing.T) {
	f := enUnFicheroDePruebas(t)
	if err := os.WriteFile(f.Ruta, []byte("esto no es JSON"), 0o600); err != nil {
		t.Fatal(err)
	}
	if f.Hay() {
		t.Fatal("un fichero roto cuenta como llave")
	}
	if _, err := f.Leer("una", ""); err != ErrNoHay {
		t.Fatalf("leer de un fichero roto devuelve %v", err)
	}
	// Y guardar encima funciona: lo roto se reemplaza, que ahí no hay nada que
	// conservar.
	if err := f.Guardar("una", bytes.Repeat([]byte{2}, 32)); err != nil {
		t.Fatal(err)
	}
	var llaves map[string]string
	datos, _ := os.ReadFile(f.Ruta)
	if err := json.Unmarshal(datos, &llaves); err != nil {
		t.Fatalf("lo escrito encima no es JSON: %v", err)
	}
	if len(llaves) != 1 {
		t.Fatalf("hay %d llaves", len(llaves))
	}
}
