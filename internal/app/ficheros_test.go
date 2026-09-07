package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/cripto"
)

func TestLimpiarRuta(t *testing.T) {
	casa, _ := os.UserHomeDir()

	casos := []struct {
		nombre, entra, sale string
	}{
		{"tal cual", "/tmp/fichero.env", "/tmp/fichero.env"},
		{"con espacios alrededor", "  /tmp/fichero.env  ", "/tmp/fichero.env"},
		{"comillas dobles", `"/tmp/mis cosas/fichero.env"`, "/tmp/mis cosas/fichero.env"},
		{"comillas simples", `'/tmp/mis cosas/fichero.env'`, "/tmp/mis cosas/fichero.env"},
		{"espacios escapados", `/tmp/mis\ cosas/fichero.env`, "/tmp/mis cosas/fichero.env"},
		{"paréntesis escapados", `/tmp/copia\ \(1\).env`, "/tmp/copia (1).env"},
		{"arrastrado con salto", "/tmp/fichero.env\n", "/tmp/fichero.env"},
		{"virgulilla", "~/fichero.env", filepath.Join(casa, "fichero.env")},
		{"ruta de Windows", `C:\Users\ana\fichero.env`, `C:\Users\ana\fichero.env`},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := LimpiarRuta(c.entra); got != c.sale {
				t.Errorf("LimpiarRuta(%q) = %q, quiero %q", c.entra, got, c.sale)
			}
		})
	}
}

func TestCifrarYDescifrarFichero(t *testing.T) {
	dir := t.TempDir()
	origen := filepath.Join(dir, "credenciales.env")
	contenido := []byte("DATABASE_URL=postgres://u:p@h/db\nAPI_KEY=abc123\n")
	if err := os.WriteFile(origen, contenido, 0o644); err != nil {
		t.Fatal(err)
	}
	clave := []byte("la clave del fichero")

	cifrado, err := CifrarFichero(origen, clave)
	if err != nil {
		t.Fatalf("CifrarFichero: %v", err)
	}
	if cifrado != origen+".esf" {
		t.Errorf("el fichero cifrado salió en %q", cifrado)
	}

	// El original no se toca: perder el de partida por cifrarlo sería un
	// desastre silencioso.
	sigue, err := os.ReadFile(origen)
	if err != nil || !bytes.Equal(sigue, contenido) {
		t.Error("el fichero original ha cambiado")
	}

	// Y el cifrado no puede tener el secreto a la vista.
	crudo, _ := os.ReadFile(cifrado)
	if bytes.Contains(crudo, []byte("postgres")) {
		t.Error("el contenido está en claro dentro del fichero cifrado")
	}
	if info, _ := os.Stat(cifrado); info.Mode().Perm() != 0o600 {
		t.Errorf("permisos %o, esperaba 600", info.Mode().Perm())
	}

	// La vuelta. Como «credenciales.env» ya existe, tiene que buscarse otro
	// nombre en vez de pisarlo.
	claro, err := DescifrarFichero(cifrado, clave)
	if err != nil {
		t.Fatalf("DescifrarFichero: %v", err)
	}
	if claro == origen {
		t.Fatal("ha pisado el fichero original")
	}
	vuelta, err := os.ReadFile(claro)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(vuelta, contenido) {
		t.Errorf("la vuelta dio %q", vuelta)
	}
}

func TestDescifrarFicheroConClaveIncorrecta(t *testing.T) {
	dir := t.TempDir()
	origen := filepath.Join(dir, "secreto.txt")
	os.WriteFile(origen, []byte("contenido"), 0o644)

	cifrado, err := CifrarFichero(origen, []byte("buena"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DescifrarFichero(cifrado, []byte("mala")); err == nil {
		t.Fatal("ha descifrado con la clave equivocada")
	}

	// Y no puede quedar un fichero a medias por el camino.
	restos, _ := filepath.Glob(filepath.Join(dir, "secreto.txt*"))
	for _, r := range restos {
		if r != origen && r != cifrado {
			t.Errorf("ha dejado un fichero suelto: %s", r)
		}
	}
}

// Un fichero que guarda el texto ESF1.… de una línea también se descifra: quien
// lo recibe no tiene por qué saber en qué formato se lo han mandado.
func TestDescifrarFicheroDeTexto(t *testing.T) {
	dir := t.TempDir()
	clave := []byte("clave")

	// Un contenedor de texto, como el que sale de «esfinge cifrar» por la salida
	// estándar y alguien guarda en un fichero.
	texto, err := cripto.SellarTexto([]byte("el secreto"), clave, cripto.PerfilInteractivo)
	if err != nil {
		t.Fatal(err)
	}

	comoTexto := filepath.Join(dir, "recibido.esf")
	os.WriteFile(comoTexto, []byte(texto+"\n"), 0o644)

	claro, err := DescifrarFichero(comoTexto, clave)
	if err != nil {
		t.Fatalf("DescifrarFichero sobre un contenedor de texto: %v", err)
	}
	datos, _ := os.ReadFile(claro)
	if string(datos) != "el secreto" {
		t.Errorf("tengo %q", datos)
	}
}

func TestFicheroQueNoExiste(t *testing.T) {
	_, err := CifrarFichero("/no/existe/esto.env", []byte("clave"))
	if err == nil {
		t.Fatal("ha aceptado un fichero inexistente")
	}
	if !strings.Contains(err.Error(), "No existe") {
		t.Errorf("el mensaje no dice qué pasa: %v", err)
	}
}

func TestCarpetaEnVezDeFichero(t *testing.T) {
	dir := t.TempDir()
	_, err := CifrarFichero(dir, []byte("clave"))
	if err == nil {
		t.Fatal("ha aceptado una carpeta")
	}
	if !strings.Contains(err.Error(), "carpeta") {
		t.Errorf("el mensaje no dice que es una carpeta: %v", err)
	}
}

// Cifrar dos veces el mismo fichero no puede pisar el primer cifrado.
func TestCifrarDosVecesNoPisa(t *testing.T) {
	dir := t.TempDir()
	origen := filepath.Join(dir, "datos.txt")
	os.WriteFile(origen, []byte("x"), 0o644)

	uno, err := CifrarFichero(origen, []byte("clave"))
	if err != nil {
		t.Fatal(err)
	}
	dos, err := CifrarFichero(origen, []byte("clave"))
	if err != nil {
		t.Fatal(err)
	}
	if uno == dos {
		t.Errorf("los dos cifrados fueron al mismo sitio: %s", uno)
	}
}
