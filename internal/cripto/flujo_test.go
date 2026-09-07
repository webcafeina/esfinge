package cripto

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func idaYVueltaFlujo(t *testing.T, contenido []byte) {
	t.Helper()
	clave := []byte("clave del fichero")

	var cifrado bytes.Buffer
	if err := SellarFlujo(&cifrado, bytes.NewReader(contenido), clave, pruebas); err != nil {
		t.Fatalf("SellarFlujo: %v", err)
	}

	var claro bytes.Buffer
	if err := AbrirFlujo(&claro, bytes.NewReader(cifrado.Bytes()), clave); err != nil {
		t.Fatalf("AbrirFlujo: %v", err)
	}
	if !bytes.Equal(claro.Bytes(), contenido) {
		t.Errorf("el fichero cambió: %d bytes dentro, %d fuera", len(contenido), claro.Len())
	}
}

func TestFlujoIdaYVuelta(t *testing.T) {
	casos := map[string][]byte{
		"vacío":                {},
		"un byte":              {0x00},
		"menos de un segmento": []byte("DATABASE_URL=postgres://u:p@h/db\nAPI_KEY=abc\n"),
		"justo un segmento":    bytes.Repeat([]byte("x"), TamSegmento),
		"segmento y pico":      bytes.Repeat([]byte("y"), TamSegmento+1),
		"dos segmentos":        bytes.Repeat([]byte("z"), TamSegmento*2),
		"varios segmentos":     bytes.Repeat([]byte("abcd"), TamSegmento),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) { idaYVueltaFlujo(t, contenido) })
	}
}

func TestFlujoClaveIncorrecta(t *testing.T) {
	var cifrado bytes.Buffer
	if err := SellarFlujo(&cifrado, strings.NewReader("secreto"), []byte("buena"), pruebas); err != nil {
		t.Fatal(err)
	}

	var claro bytes.Buffer
	err := AbrirFlujo(&claro, bytes.NewReader(cifrado.Bytes()), []byte("mala"))
	if !errors.Is(err, ErrClaveIncorrecta) {
		t.Errorf("quiero ErrClaveIncorrecta, tengo %v", err)
	}
	if claro.Len() != 0 {
		t.Errorf("ha salido contenido sin autenticar: %d bytes", claro.Len())
	}
}

// TestFlujoTruncadoDetectado es la razón de ser de la marca de final. Se corta un
// contenedor de tres segmentos justo por el borde de uno, de modo que lo que queda
// es un prefijo perfectamente válido: todos los segmentos que hay descifran bien.
// Sin la marca, esto se abriría en silencio y el destinatario se quedaría con
// medio fichero creyendo que lo tiene entero.
func TestFlujoTruncadoDetectado(t *testing.T) {
	clave := []byte("clave")
	contenido := bytes.Repeat([]byte("w"), TamSegmento*3)

	var cifrado bytes.Buffer
	if err := SellarFlujo(&cifrado, bytes.NewReader(contenido), clave, pruebas); err != nil {
		t.Fatal(err)
	}

	corte := tamCabecera + (TamSegmento+tamEtiqueta)*2 // dos segmentos enteros
	var claro bytes.Buffer
	err := AbrirFlujo(&claro, bytes.NewReader(cifrado.Bytes()[:corte]), clave)
	if !errors.Is(err, ErrTruncado) {
		t.Errorf("quiero ErrTruncado, tengo %v", err)
	}
}

// TestFlujoCortadoNoCulpaALaClave cubre el caso de verdad: un fichero que llega
// cortado por cualquier sitio, no justo en el borde de un segmento. La clave es
// buena y hay que decirlo, porque si el mensaje dice «la clave no es correcta»
// quien lo recibe se pasa media hora comprobando la clave en vez de pedir que se
// lo manden otra vez.
func TestFlujoCortadoNoCulpaALaClave(t *testing.T) {
	clave := []byte("clave")
	contenido := bytes.Repeat([]byte("w"), TamSegmento*3)

	var cifrado bytes.Buffer
	if err := SellarFlujo(&cifrado, bytes.NewReader(contenido), clave, pruebas); err != nil {
		t.Fatal(err)
	}

	entero := cifrado.Bytes()

	// A partir del segundo segmento sí se puede afirmar que la clave es buena:
	// el primero ya se abrió con ella.
	for _, corte := range []int{
		tamCabecera + TamSegmento + tamEtiqueta + 1000, // dentro del segundo segmento
		len(entero) - 1, // le falta un byte al final
	} {
		var claro bytes.Buffer
		err := AbrirFlujo(&claro, bytes.NewReader(entero[:corte]), clave)
		if err == nil {
			t.Fatalf("un contenedor cortado a %d bytes se abrió", corte)
		}
		if errors.Is(err, ErrClaveIncorrecta) {
			t.Errorf("cortado a %d bytes: culpa a la clave, que es correcta (%v)", corte, err)
		}
	}

	// Dentro del primer segmento no hay nada que hacer: sin haber abierto nada
	// todavía, «la clave está mal» y «el fichero llegó cortado» son la misma
	// etiqueta de autenticación que no cuadra, y son indistinguibles. Lo único
	// exigible es que falle y que el mensaje mencione las dos posibilidades.
	var claro bytes.Buffer
	err := AbrirFlujo(&claro, bytes.NewReader(entero[:tamCabecera+TamSegmento/2]), clave)
	if err == nil {
		t.Fatal("un contenedor cortado en el primer segmento se abrió")
	}
	if !strings.Contains(err.Error(), "cortado") {
		t.Errorf("el mensaje ambiguo debería mencionar que puede estar cortado: %q", err)
	}
}

// TestFlujoSegmentosReordenados: cambiar dos segmentos de sitio tiene que romper,
// porque el número de segmento va dentro del nonce.
func TestFlujoSegmentosReordenados(t *testing.T) {
	clave := []byte("clave")
	contenido := bytes.Repeat([]byte("v"), TamSegmento*2+10)

	var cifrado bytes.Buffer
	if err := SellarFlujo(&cifrado, bytes.NewReader(contenido), clave, pruebas); err != nil {
		t.Fatal(err)
	}

	b := bytes.Clone(cifrado.Bytes())
	tam := TamSegmento + tamEtiqueta
	uno := b[tamCabecera : tamCabecera+tam]
	dos := b[tamCabecera+tam : tamCabecera+tam*2]
	intercambio := bytes.Clone(uno)
	copy(uno, dos)
	copy(dos, intercambio)

	var claro bytes.Buffer
	if err := AbrirFlujo(&claro, bytes.NewReader(b), clave); err == nil {
		t.Error("reordenar los segmentos pasó desapercibido")
	}
}

func TestFlujoManipulacionDetectada(t *testing.T) {
	clave := []byte("clave")
	var cifrado bytes.Buffer
	if err := SellarFlujo(&cifrado, strings.NewReader("un secreto en un fichero"), clave, pruebas); err != nil {
		t.Fatal(err)
	}

	for i := range cifrado.Bytes() {
		alterado := bytes.Clone(cifrado.Bytes())
		alterado[i] ^= 0x01

		var claro bytes.Buffer
		if err := AbrirFlujo(&claro, bytes.NewReader(alterado), clave); err == nil {
			t.Fatalf("alterar el byte %d pasó desapercibido", i)
		}
	}
}

// Los dos modos no se confunden entre sí: cada uno dice qué hay que usar.
func TestModosNoSeMezclan(t *testing.T) {
	clave := []byte("clave")

	unico, err := Sellar([]byte("x"), clave, pruebas)
	if err != nil {
		t.Fatal(err)
	}
	var claro bytes.Buffer
	if err := AbrirFlujo(&claro, bytes.NewReader(unico), clave); err == nil {
		t.Error("AbrirFlujo aceptó un contenedor de modo único")
	}

	var flujo bytes.Buffer
	if err := SellarFlujo(&flujo, strings.NewReader("x"), clave, pruebas); err != nil {
		t.Fatal(err)
	}
	if _, err := Abrir(flujo.Bytes(), clave); err == nil {
		t.Error("Abrir aceptó un contenedor de modo flujo")
	}
}

func TestGenerar(t *testing.T) {
	for _, a := range []Alfabeto{AlfHex, AlfAlnum, AlfSimbolos} {
		t.Run(a.Nombre, func(t *testing.T) {
			visto := map[string]bool{}
			for i := 0; i < 50; i++ {
				p, err := Generar(a, 24)
				if err != nil {
					t.Fatal(err)
				}
				if visto[p] {
					t.Fatal("contraseña repetida")
				}
				visto[p] = true

				if a.SeguroURL && strings.ContainsAny(p, "/+@:#?&=%") {
					t.Errorf("%s dice ser seguro en URL y ha dado %q", a.Nombre, p)
				}
				// 24 bytes = 192 bits; nunca menos caracteres de los necesarios.
				if len([]rune(p)) < 24 {
					t.Errorf("%q es demasiado corta para 192 bits", p)
				}
			}
		})
	}
}

func TestGenerarHexEsHexadecimal(t *testing.T) {
	p, err := Generar(AlfHex, 24)
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 48 {
		t.Errorf("24 bytes en hexadecimal son 48 caracteres, tengo %d", len(p))
	}
	if strings.Trim(p, "0123456789abcdef") != "" {
		t.Errorf("%q no es hexadecimal", p)
	}
}

func TestGenerarLimites(t *testing.T) {
	if _, err := Generar(AlfHex, 4); err == nil {
		t.Error("4 bytes debería rechazarse")
	}
	if _, err := Generar(AlfHex, 1000); err == nil {
		t.Error("1000 bytes debería rechazarse")
	}
}

// Las dos formas de medir una contraseña tienen que cuadrar entre sí: pedir por
// caracteres y volver a contarlos no puede dar otra cifra.
func TestMedirEnCaracteresYEnBytesCuadran(t *testing.T) {
	for _, a := range []Alfabeto{AlfHex, AlfAlnum, AlfSimbolos} {
		t.Run(a.Nombre, func(t *testing.T) {
			for caracteres := 16; caracteres <= 96; caracteres += 4 {
				bytes := BytesParaCaracteres(a, caracteres)
				salen := Caracteres(a, bytes)

				// Se acepta perder algún carácter por el redondeo —los bits no son
				// divisibles a voluntad—, pero no alejarse.
				if salen < caracteres-2 || salen > caracteres+2 {
					t.Errorf("pedir %d caracteres da %d bytes, que producen %d",
						caracteres, bytes, salen)
				}

				p, err := Generar(a, bytes)
				if err != nil {
					t.Fatalf("con %d bytes: %v", bytes, err)
				}
				if n := len([]rune(p)); n != salen {
					t.Errorf("Caracteres dice %d y la contraseña tiene %d", salen, n)
				}
			}
		})
	}
}
