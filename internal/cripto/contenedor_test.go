package cripto

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// pruebas usa un coste ridículo a propósito: los tests comprueban el formato y
// las garantías, no lo cara que es la derivación.
var pruebas = Parametros{Memoria: 8 * 1024, Pasadas: 1, Paralelismo: 1}

func TestIdaYVuelta(t *testing.T) {
	casos := map[string]string{
		"secreto corto": "hunter2",
		"vacío":         "",
		"con acentos":   "contraseña ñandú · 1€",
		"largo":         strings.Repeat("a", 10000),
	}
	clave := []byte("la clave maestra")

	for nombre, secreto := range casos {
		t.Run(nombre, func(t *testing.T) {
			sellado, err := Sellar([]byte(secreto), clave, pruebas)
			if err != nil {
				t.Fatalf("Sellar: %v", err)
			}
			abierto, err := Abrir(sellado, clave)
			if err != nil {
				t.Fatalf("Abrir: %v", err)
			}
			if string(abierto) != secreto {
				t.Errorf("ida y vuelta cambió el contenido:\nquiero %q\ntengo  %q", secreto, abierto)
			}
		})
	}
}

func TestClaveIncorrecta(t *testing.T) {
	sellado, err := Sellar([]byte("secreto"), []byte("buena"), pruebas)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Abrir(sellado, []byte("mala")); !errors.Is(err, ErrClaveIncorrecta) {
		t.Errorf("con la clave equivocada quiero ErrClaveIncorrecta, tengo %v", err)
	}
}

func TestManipulacionDetectada(t *testing.T) {
	clave := []byte("clave")
	sellado, err := Sellar([]byte("transferir 100 euros"), clave, pruebas)
	if err != nil {
		t.Fatal(err)
	}

	// Cada byte del contenedor, uno a uno: cambiar cualquiera tiene que romper la
	// apertura. Incluye la cabecera, que va en claro pero autenticada.
	for i := range sellado {
		alterado := bytes.Clone(sellado)
		alterado[i] ^= 0x01

		if _, err := Abrir(alterado, clave); err == nil {
			t.Fatalf("alterar el byte %d de %d pasó desapercibido", i, len(sellado))
		}
	}
}

func TestTruncadoDetectado(t *testing.T) {
	clave := []byte("clave")
	sellado, err := Sellar([]byte("secreto"), clave, pruebas)
	if err != nil {
		t.Fatal(err)
	}

	for n := 0; n < len(sellado); n++ {
		if _, err := Abrir(sellado[:n], clave); err == nil {
			t.Fatalf("un contenedor cortado a %d de %d bytes se abrió", n, len(sellado))
		}
	}
}

func TestFormatoTexto(t *testing.T) {
	clave := []byte("clave")
	texto, err := SellarTexto([]byte("postgres://u:p@h/db"), clave, pruebas)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(texto, Prefijo) {
		t.Errorf("falta el prefijo %q en %q", Prefijo, texto)
	}
	if strings.ContainsAny(texto, "/+=") {
		t.Errorf("el texto cifrado tiene caracteres que rompen una URL: %q", texto)
	}
	if strings.ContainsAny(texto, " \n\r\t") {
		t.Errorf("el texto cifrado tiene espacios: %q", texto)
	}

	abierto, err := AbrirTexto(texto, clave)
	if err != nil {
		t.Fatalf("AbrirTexto: %v", err)
	}
	if string(abierto) != "postgres://u:p@h/db" {
		t.Errorf("tengo %q", abierto)
	}
}

func TestAbrirTextoTolerante(t *testing.T) {
	clave := []byte("clave")
	texto, err := SellarTexto([]byte("secreto"), clave, pruebas)
	if err != nil {
		t.Fatal(err)
	}

	// Lo que le pasa de verdad a un texto pegado desde un correo o un chat.
	casos := map[string]string{
		"con saltos":     "  " + texto + "\n",
		"partido":        texto[:20] + "\n" + texto[20:],
		"sin el prefijo": strings.TrimPrefix(texto, Prefijo),
	}
	for nombre, variante := range casos {
		t.Run(nombre, func(t *testing.T) {
			abierto, err := AbrirTexto(variante, clave)
			if err != nil {
				t.Fatalf("%v", err)
			}
			if string(abierto) != "secreto" {
				t.Errorf("tengo %q", abierto)
			}
		})
	}
}

func TestNoEsUnContenedor(t *testing.T) {
	casos := map[string]string{
		"texto suelto": "esto no es nada",
		"vacío":        "",
		"otra magia":   "XXX1.AAAA",
	}
	for nombre, basura := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := AbrirTexto(basura, []byte("clave"))
			if !errors.Is(err, ErrFormato) {
				t.Errorf("quiero ErrFormato, tengo %v", err)
			}
		})
	}
}

// TestCabeceraCongelada fija el formato ESF1 sobre el papel. Si alguien cambia el
// orden o el tamaño de un campo, este test se entera y hay que subir la versión
// del contenedor en vez de romper los mensajes ya emitidos.
func TestCabeceraCongelada(t *testing.T) {
	if tamCabecera != 55 {
		t.Errorf("la cabecera mide %d bytes y el formato dice 55", tamCabecera)
	}

	sellado, err := Sellar([]byte("x"), []byte("clave"), pruebas)
	if err != nil {
		t.Fatal(err)
	}

	if string(sellado[:4]) != "ESF1" {
		t.Errorf("magia %q", sellado[:4])
	}
	if sellado[4] != version {
		t.Errorf("versión %d", sellado[4])
	}
	if Modo(sellado[5]) != ModoUnico {
		t.Errorf("modo %d", sellado[5])
	}
	if len(sellado) != tamCabecera+1+tamEtiqueta {
		t.Errorf("un secreto de 1 byte da un contenedor de %d bytes", len(sellado))
	}

	cab, err := leerCabecera(sellado)
	if err != nil {
		t.Fatal(err)
	}
	if cab.Par != pruebas {
		t.Errorf("los parámetros no sobreviven a la ida y vuelta: %+v", cab.Par)
	}
}

// TestCabeceraAbusiva comprueba que una cabecera que pide una barbaridad de
// memoria se rechaza antes de intentar reservarla. Sin esto, un fichero de 55
// bytes tumba el proceso.
func TestCabeceraAbusiva(t *testing.T) {
	sellado, err := Sellar([]byte("x"), []byte("clave"), pruebas)
	if err != nil {
		t.Fatal(err)
	}

	abusivo := bytes.Clone(sellado)
	abusivo[6], abusivo[7], abusivo[8], abusivo[9] = 0xff, 0xff, 0xff, 0xff // memoria

	if _, err := Abrir(abusivo, []byte("clave")); !errors.Is(err, ErrFormato) {
		t.Errorf("quiero ErrFormato ante una memoria imposible, tengo %v", err)
	}
}

func TestVersionFutura(t *testing.T) {
	sellado, err := Sellar([]byte("x"), []byte("clave"), pruebas)
	if err != nil {
		t.Fatal(err)
	}
	futuro := bytes.Clone(sellado)
	futuro[4] = 99

	if _, err := Abrir(futuro, []byte("clave")); !errors.Is(err, ErrVersion) {
		t.Errorf("quiero ErrVersion, tengo %v", err)
	}
}

func TestSalYNonceCambianSiempre(t *testing.T) {
	clave := []byte("clave")
	visto := map[string]bool{}
	for i := 0; i < 20; i++ {
		sellado, err := Sellar([]byte("el mismo secreto"), clave, pruebas)
		if err != nil {
			t.Fatal(err)
		}
		llave := string(sellado[15:tamCabecera]) // sal + nonce
		if visto[llave] {
			t.Fatal("se ha repetido la pareja sal+nonce: el azar no está funcionando")
		}
		visto[llave] = true

		if bytes.Equal(sellado[tamCabecera:], []byte("el mismo secreto")) {
			t.Fatal("el contenido ha salido en claro")
		}
	}
}

func TestEvaluar(t *testing.T) {
	casos := []struct {
		clave     string
		nivelMax  int
		nivelMin  int
	}{
		{"", 0, 0},
		{"1234", 0, 0},
		{"hunter2", 1, 0},
		{"Tr0ub4dor&3", 2, 1},
		{"caballo grapa batería correcto", 4, 3},
	}
	for _, c := range casos {
		f := Evaluar(c.clave)
		if f.Nivel < c.nivelMin || f.Nivel > c.nivelMax {
			t.Errorf("Evaluar(%q) = nivel %d (%s, %.0f bits), esperaba entre %d y %d",
				c.clave, f.Nivel, f.Etiqueta, f.Bits, c.nivelMin, c.nivelMax)
		}
		if f.Etiqueta == "" {
			t.Errorf("Evaluar(%q) no trae etiqueta", c.clave)
		}
	}
}
