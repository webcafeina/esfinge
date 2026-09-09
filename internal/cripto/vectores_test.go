package cripto

// Los vectores fijos: contenedores grabados una vez y **nunca regenerados**.
//
// **Por qué hacían falta.** Hasta aquí, el único test que decía congelar el
// formato —`TestCabeceraCongelada`— sella y abre en el momento: comprueba el
// código contra sí mismo. Un cambio coherente en los dos sentidos —el orden de
// bytes de los parámetros, el offset del contador de segmento en el nonce, qué
// entra en el AAD— pasaría en verde y dejaría de abrir los `.esf` ya emitidos,
// **en silencio**. La ADR 0002 afirma que un contenedor de la 1.0 se abre con la
// 2.x y nada lo verificaba.
//
// Aquí se comprueban dos cosas distintas, y hacen falta las dos:
//
//   - que los ficheros grabados **se siguen abriendo** (el camino de lectura);
//   - que sellar el mismo claro con la misma sal y el mismo nonce **da los
//     mismos bytes** (el camino de escritura). Esto es lo que atrapa el cambio
//     coherente, y es la razón de que `azar` sea una variable.
//
// Si un cambio pone esto en rojo: o el cambio está mal, o toca subir la versión
// del contenedor y grabar vectores nuevos **al lado**, sin tocar los viejos.
// Regenerarlos para que el test vuelva a verde es borrar la única red que hay.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type vectorFijo struct {
	Fichero    string     `json:"fichero"`
	Clave      string     `json:"clave"`
	Modo       string     `json:"modo"`
	Parametros Parametros `json:"parametros"`
	Sal        string     `json:"sal"`
	Nonce      string     `json:"nonce"`
	Claro      string     `json:"claro"`
	ClaroBytes int        `json:"claroBytes"`
	SHA256     string     `json:"sha256"`
}

type vectorRoto struct {
	Fichero string `json:"fichero"`
	Clave   string `json:"clave"`
	Error   string `json:"error"`
	Porque  string `json:"porque"`
}

type manifiesto struct {
	Vectores []vectorFijo `json:"vectores"`
	Rotos    []vectorRoto `json:"rotos"`
}

// patronDe reconstruye el claro de los vectores grandes. Guardar 130 KiB de
// claro al lado de 130 KiB de cifrado sería duplicar el peso del repositorio
// para nada: la fórmula es fija y basta con no tocarla.
func patronDe(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*31 + 7)
	}
	return b
}

func leerManifiesto(t *testing.T) manifiesto {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "vectores.json"))
	if err != nil {
		t.Fatalf("sin manifiesto de vectores: %v", err)
	}
	var m manifiesto
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("manifiesto ilegible: %v", err)
	}
	if len(m.Vectores) == 0 {
		t.Fatal("el manifiesto no tiene vectores")
	}
	return m
}

func claroDe(v vectorFijo) []byte {
	if v.ClaroBytes > 0 {
		return patronDe(v.ClaroBytes)
	}
	return []byte(v.Claro)
}

// El camino de lectura: lo grabado hace meses sigue abriéndose.
func TestLosVectoresFijosSeAbren(t *testing.T) {
	for _, v := range leerManifiesto(t).Vectores {
		t.Run(v.Fichero, func(t *testing.T) {
			bruto, err := os.ReadFile(filepath.Join("testdata", v.Fichero))
			if err != nil {
				t.Fatal(err)
			}

			// El fichero no ha cambiado desde que se grabó.
			if h := hex.EncodeToString(sliceSHA(bruto)); h != v.SHA256 {
				t.Fatalf("el vector ha cambiado en disco: %s, se esperaba %s", h, v.SHA256)
			}

			var salido []byte
			switch v.Modo {
			case "unico":
				salido, err = Abrir(bruto, []byte(v.Clave))
			case "texto":
				salido, err = AbrirTexto(string(bruto), []byte(v.Clave))
			case "flujo":
				var w bytes.Buffer
				err = AbrirFlujo(&w, bytes.NewReader(bruto), []byte(v.Clave))
				salido = w.Bytes()
			default:
				t.Fatalf("modo desconocido: %q", v.Modo)
			}
			if err != nil {
				t.Fatalf("no abre: %v", err)
			}
			if quiero := claroDe(v); !bytes.Equal(salido, quiero) {
				t.Errorf("el claro no coincide: %d bytes, se esperaban %d", len(salido), len(quiero))
			}
		})
	}
}

// El camino de escritura, que es el que atrapa un cambio coherente en los dos
// sentidos: con la misma sal y el mismo nonce, sellar tiene que dar los mismos
// bytes que hace meses.
func TestLosVectoresFijosSeSellanIgual(t *testing.T) {
	for _, v := range leerManifiesto(t).Vectores {
		t.Run(v.Fichero, func(t *testing.T) {
			sal, err := hex.DecodeString(v.Sal)
			if err != nil {
				t.Fatal(err)
			}
			nonce, err := hex.DecodeString(v.Nonce)
			if err != nil {
				t.Fatal(err)
			}

			antes := azar
			n := 0
			azar = func(int) ([]byte, error) {
				n++
				if n == 1 {
					return append([]byte(nil), sal...), nil
				}
				return append([]byte(nil), nonce...), nil
			}
			defer func() { azar = antes }()

			claro := claroDe(v)
			var hecho []byte
			switch v.Modo {
			case "unico":
				hecho, err = Sellar(claro, []byte(v.Clave), v.Parametros)
			case "texto":
				var s string
				s, err = SellarTexto(claro, []byte(v.Clave), v.Parametros)
				hecho = []byte(s)
			case "flujo":
				var w bytes.Buffer
				err = SellarFlujo(&w, bytes.NewReader(claro), []byte(v.Clave), v.Parametros)
				hecho = w.Bytes()
			}
			if err != nil {
				t.Fatal(err)
			}

			esperado, err := os.ReadFile(filepath.Join("testdata", v.Fichero))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(hecho, esperado) {
				t.Errorf("el formato ha cambiado: sellar da %d bytes y el vector tiene %d.\n"+
					"Si el cambio es intencionado, sube la versión del contenedor y graba vectores "+
					"nuevos al lado. No regeneres éstos.", len(hecho), len(esperado))
			}
		})
	}
}

// No basta con que un contenedor roto falle: tiene que fallar **con el error que
// toca**. De esa clasificación salen los códigos de salida 3 y 4 de la línea de
// comandos, y un script que reintenta con otra clave depende de ella.
func TestLosVectoresRotosDanElErrorQueToca(t *testing.T) {
	porNombre := map[string]error{
		"ErrClaveIncorrecta": ErrClaveIncorrecta,
		"ErrFormato":         ErrFormato,
		"ErrTruncado":        ErrTruncado,
		"ErrDanado":          ErrDanado,
		"ErrVersion":         ErrVersion,
	}
	for _, r := range leerManifiesto(t).Rotos {
		t.Run(r.Fichero, func(t *testing.T) {
			bruto, err := os.ReadFile(filepath.Join("testdata", r.Fichero))
			if err != nil {
				t.Fatal(err)
			}
			quiero, vale := porNombre[r.Error]
			if !vale {
				t.Fatalf("el manifiesto nombra un error que no existe: %q", r.Error)
			}
			_, err = Abrir(bruto, []byte(r.Clave))
			if !errors.Is(err, quiero) {
				t.Errorf("%s: quiero %v, tengo %v", r.Porque, quiero, err)
			}
		})
	}
}

// Subir el coste de la derivación es una decisión, no un descuido. Los
// parámetros viajan dentro del contenedor, así que subirlos no rompe nada ya
// cifrado —y por eso es fácil hacerlo sin querer—. Esto pone un test en rojo
// delante de quien lo haga.
func TestElPerfilInteractivoNoHaCambiado(t *testing.T) {
	if PerfilInteractivo.Memoria != 64*1024 {
		t.Errorf("memoria: %d KiB, se esperaban 65536", PerfilInteractivo.Memoria)
	}
	if PerfilInteractivo.Pasadas != 3 {
		t.Errorf("pasadas: %d, se esperaban 3", PerfilInteractivo.Pasadas)
	}
	if PerfilInteractivo.Paralelismo != 4 {
		t.Errorf("paralelismo: %d, se esperaban 4", PerfilInteractivo.Paralelismo)
	}
}

// El byte de versión no puede valer nunca 46, que es el punto.
//
// Hoy es trivialmente cierto —la versión es 1— y por eso hay que escribirlo
// ahora: FormaDe distingue el contenedor binario del de texto mirando el quinto
// byte, y `ESF1.` es justamente la forma de texto. Quien suba la versión dentro
// de unos años pasará por la 45 y por la 47 sin enterarse de que la 46 rompe la
// detección de forma.
func TestElByteDeVersionNuncaPuedeSerUnPunto(t *testing.T) {
	if version == '.' {
		t.Fatal("la versión vale 46: FormaDe confundiría el binario con la forma de texto")
	}
	falso := make([]byte, 8)
	copy(falso, magia)
	falso[4] = '.'
	if f := FormaDe(falso); f != FormaTexto {
		t.Errorf("con el quinto byte a punto, FormaDe dice %v: la trampa ya no es la que se creía", f)
	}
}

func sliceSHA(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}
