package boveda

// Las pruebas cruzadas con la extensión (ADR 0040): la misma bóveda y la misma
// fusión, escritas en Go y en TypeScript, **tienen que dar los mismos bytes**.
//
// La que manda es la de la fusión con escenarios al azar. Dos fusiones «probadas»
// por separado pueden no estar de acuerdo, y entonces la aplicación y la extensión
// se pasarían la bóveda sin fin: cada una vería lo de la otra como un cambio.

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/cruzada"
)

// canonGo es la forma canónica de cualquier cosa, como `canon` pero sin ser una
// entrada: la que escribe `canonico` en la extensión.
func canonGo(t *testing.T, v any) string {
	t.Helper()
	crudo, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(crudo))
	dec.UseNumber()
	var x any
	if err := dec.Decode(&x); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(x); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// canonDeContenido: con las entradas siempre como lista, que es como las escribe
// la extensión; Go escribe `null` cuando no hay ninguna.
func canonDeContenido(t *testing.T, c contenido) string {
	t.Helper()
	if c.Entradas == nil {
		c.Entradas = []Entrada{}
	}
	return canonGo(t, c)
}

func canonDeSobres(t *testing.T, s []sobre) string {
	t.Helper()
	if s == nil {
		s = []sobre{}
	}
	return canonGo(t, s)
}

// Los textos que más fácil rompen una forma canónica: lo que Go escapa y
// JavaScript no, controles, y lo que no cabe en un byte.
var textosRaros = []string{
	"Banco", "a<b>&c", "línea\u2028párrafo\u2029", "tab\tsalto\nfin", "\x01\x1f", "ñandú", "😀", "\"comillas\" y \\", "",
}

func TestCruzadaFormaCanonica(t *testing.T) {
	var entradas []Entrada
	for i, s := range textosRaros {
		e := Entrada{ID: fmt.Sprintf("%032x", i), Tipo: TipoCredencial, Titulo: s, Usuario: s, Secreto: s,
			Creada: "2026-09-22T10:00:00Z", Cambiada: "2026-09-22T10:00:00Z", Revision: int64(i),
			Sitios: []string{s, "x.com"}, Historial: []Antigua{{Secreto: s, Hasta: "2026-01-01T00:00:00Z"}}}
		if i%2 == 0 {
			e.Extra = map[string]json.RawMessage{"campoNuevo": json.RawMessage(`{"z":1,"a":["` + fmt.Sprint(i) + `"],"é":null}`), "Zeta": json.RawMessage(`true`)}
		}
		entradas = append(entradas, e)
	}
	var suyas []string
	cruzada.Pedir(t, map[string]any{"orden": "canon", "entradas": entradas}, &suyas)
	for i, e := range entradas {
		if suyas[i] != canon(e) {
			t.Errorf("entrada %d:\n  Go:        %s\n  extensión: %s", i, canon(e), suyas[i])
		}
	}
}

// ------------------------------------------------------------------ la fusión al azar

type casoDeFusion struct {
	L       contenido  `json:"l"`
	R       contenido  `json:"r"`
	B       *contenido `json:"b"`
	SobresL []sobre    `json:"sobresL"`
	SobresR []sobre    `json:"sobresR"`
	SobresB []sobre    `json:"sobresB"`
	Ahora   string     `json:"ahora"`
}

type resultadoDeFusion struct {
	Contenido  string `json:"contenido"`
	Sobres     string `json:"sobres"`
	Traidas    int    `json:"traidas"`
	Conflictos int    `json:"conflictos"`
}

// generador hace escenarios con **pocas opciones de cada cosa**, para que choquen:
// mismos identificadores en los dos lados, fechas y revisiones que empatan,
// lápidas de entradas que siguen vivas y ranuras cambiadas en el mismo segundo.
type generador struct{ r *rand.Rand }

func (g generador) de(opciones ...string) string { return opciones[g.r.IntN(len(opciones))] }

func (g generador) fecha() string {
	return g.de("2026-09-01T10:00:00Z", "2026-09-01T10:00:00Z", "2026-09-10T08:30:00Z", "2026-09-20T23:59:59Z", "2026-09-22T12:00:00Z")
}

func (g generador) entrada(id string) Entrada {
	e := Entrada{
		ID: id, Tipo: Tipo(g.de("credencial", "credencial", "nota", "tarjeta")),
		Titulo: g.de(textosRaros...), Creada: g.fecha(), Cambiada: g.fecha(), Revision: int64(g.r.IntN(4)),
	}
	if g.r.IntN(2) == 0 {
		e.Usuario = g.de("yo", "tú", "a<b>&c", "")
	}
	if g.r.IntN(3) > 0 {
		e.Secreto = g.de("uno", "dos", "tres", "línea\u2028párrafo")
	}
	if g.r.IntN(3) == 0 {
		e.Sitios = []string{g.de("x.com", "y.es", "😀.com")}
	}
	if g.r.IntN(4) == 0 {
		e.Notas = g.de("nota", "otra\nnota")
	}
	if g.r.IntN(4) == 0 {
		e.Historial = []Antigua{{Secreto: g.de("viejo", "uno", "dos"), Hasta: g.fecha()}}
	}
	if g.r.IntN(5) == 0 {
		e.Papelera, e.BorradaEn = true, g.fecha()
	}
	if g.r.IntN(5) == 0 {
		e.Extra = map[string]json.RawMessage{"nuevo": json.RawMessage(g.de(`1`, `"x"`, `{"b":[1,2]}`))}
	}
	return e
}

// tocar cambia la copia de un lado como lo haría una persona en ese equipo.
func (g generador) tocar(e Entrada) Entrada {
	switch g.r.IntN(7) {
	case 0:
		e.Titulo = g.de(textosRaros...)
	case 1:
		e.Secreto = g.de("uno", "dos", "cuatro")
	case 2:
		e.Notas = g.de("", "nota", "cambiada")
	case 3:
		e.Sitios = append(append([]string(nil), e.Sitios...), g.de("z.org", "x.com"))
	case 4:
		e.Papelera = !e.Papelera
		if e.Papelera {
			e.BorradaEn = g.fecha()
		} else {
			e.BorradaEn = ""
		}
	case 5:
		e.Extra = map[string]json.RawMessage{"nuevo": json.RawMessage(g.de(`2`, `"y"`))}
	case 6:
		e.Usuario = g.de("yo", "otro")
	}
	e.Revision += int64(g.r.IntN(2))
	e.Cambiada = g.fecha()
	return e
}

func (g generador) lado(base []Entrada, ids []string) contenido {
	var c contenido
	for _, e := range base {
		switch g.r.IntN(6) {
		case 0: // se ha borrado del todo aquí
			if c.Lapidas == nil {
				c.Lapidas = map[string]string{}
			}
			c.Lapidas[e.ID] = g.fecha()
		case 1, 2:
			c.Entradas = append(c.Entradas, g.tocar(e))
		default:
			c.Entradas = append(c.Entradas, e)
		}
	}
	for i := g.r.IntN(3); i > 0; i-- { // nuevas, a veces con un id que también tiene el otro lado
		c.Entradas = append(c.Entradas, g.entrada(ids[g.r.IntN(len(ids))]))
	}
	if g.r.IntN(4) == 0 {
		if c.Lapidas == nil {
			c.Lapidas = map[string]string{}
		}
		c.Lapidas[ids[g.r.IntN(len(ids))]] = g.fecha()
	}
	// Una entrada solo puede estar una vez en cada lado.
	visto := map[string]bool{}
	var unicas []Entrada
	for _, e := range c.Entradas {
		if !visto[e.ID] {
			visto[e.ID] = true
			unicas = append(unicas, e)
		}
	}
	c.Entradas = unicas
	if g.r.IntN(2) == 0 {
		c.SitiosExcluidos = []string{g.de("a.com", "b.com", "c.com")}
	}
	if g.r.IntN(5) == 0 {
		c.Extra = map[string]json.RawMessage{"seccionNueva": json.RawMessage(g.de(`[1]`, `{"x":"<&>"}`))}
	}
	return c
}

func (g generador) sobres() []sobre {
	var out []sobre
	for _, tipo := range []string{"maestra", "recuperacion", "pin"} {
		if tipo == "pin" && g.r.IntN(3) > 0 {
			continue
		}
		s := sobre{Tipo: tipo, Creado: g.fecha(), Contenedor: "ESF1." + g.de("aaa", "bbb", "ccc")}
		if tipo == "recuperacion" {
			s.Codificacion = "crockford32-v1"
		}
		out = append(out, s)
	}
	return out
}

func (g generador) caso() casoDeFusion {
	ids := []string{"0a", "0b", "0c", "0d", "0e", "0f"}
	var base []Entrada
	for _, id := range ids[:g.r.IntN(len(ids))] {
		base = append(base, g.entrada(id))
	}
	c := casoDeFusion{
		L: g.lado(base, ids), R: g.lado(base, ids),
		SobresL: g.sobres(), SobresR: g.sobres(),
		Ahora: "2026-09-22T13:14:15Z",
	}
	if g.r.IntN(5) > 0 {
		b := contenido{Entradas: base}
		if g.r.IntN(3) == 0 {
			b.Lapidas = map[string]string{ids[g.r.IntN(len(ids))]: g.fecha()}
		}
		if g.r.IntN(3) == 0 {
			b.SitiosExcluidos = []string{g.de("a.com", "b.com")}
		}
		c.B = &b
		c.SobresB = g.sobres()
		if g.r.IntN(2) == 0 {
			c.SobresB = c.SobresL
		}
	}
	return c
}

func TestCruzadaFusionAlAzar(t *testing.T) {
	cruzada.Activa(t)
	semilla := uint64(time.Now().UnixNano())
	if s := os.Getenv("ESFINGE_SEMILLA"); s != "" {
		fmt.Sscan(s, &semilla)
	}
	t.Logf("semilla %d (para repetirla: ESFINGE_SEMILLA=%d)", semilla, semilla)
	g := generador{rand.New(rand.NewPCG(semilla, 7))}
	var casos []casoDeFusion
	for i := 0; i < 400; i++ {
		casos = append(casos, g.caso())
	}
	// Y unos fijos, que no dependen del azar: vacío contra vacío y sin base.
	casos = append(casos, casoDeFusion{Ahora: "2026-09-22T13:14:15Z"})

	var suyos []resultadoDeFusion
	cruzada.Pedir(t, map[string]any{"orden": "fundir", "casos": casos}, &suyos)
	if len(suyos) != len(casos) {
		t.Fatalf("%d casos y %d respuestas", len(casos), len(suyos))
	}
	fallos := 0
	for i, c := range casos {
		ahoraT, _ := time.Parse(time.RFC3339, c.Ahora)
		var f Fusion
		// Go trabaja sobre copias: fundir toca las lápidas de lo que se le da.
		l, r := copiaDe(t, c.L), copiaDe(t, c.R)
		var b *contenido
		if c.B != nil {
			x := copiaDe(t, *c.B)
			b = &x
		}
		cont := fundirContenido(l, r, b, ahoraT, &f)
		sobres := fundirSobres(c.SobresL, c.SobresR, c.SobresB, c.B != nil)
		quiero := resultadoDeFusion{canonDeContenido(t, cont), canonDeSobres(t, sobres), f.Traidas, f.Conflictos}
		if suyos[i] != quiero {
			fallos++
			if fallos <= 3 {
				caso, _ := json.MarshalIndent(c, "", " ")
				t.Errorf("caso %d: Go y la extensión funden distinto\n  Go:        %+v\n  extensión: %+v\n  caso: %s", i, quiero, suyos[i], caso)
			}
		}
	}
	if fallos > 0 {
		t.Fatalf("%d de %d casos funden distinto (semilla %d)", fallos, len(casos), semilla)
	}
}

func copiaDe(t *testing.T, c contenido) contenido {
	t.Helper()
	crudo, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var out contenido
	if err := json.Unmarshal(crudo, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// ------------------------------------------------------------------ la bóveda entera

type resumenDeBoveda struct {
	Texto        string   `json:"texto"`
	Recuperacion string   `json:"recuperacion"`
	ID           string   `json:"id"`
	Serie        int64    `json:"serie"`
	Entradas     []string `json:"entradas"`
	Excluidos    []string `json:"excluidos"`
	Posesion     string   `json:"posesion"`
}

// resumenDe es lo mismo que calcula la extensión: vivas, luego la papelera en su
// orden, cada una en forma canónica.
func resumenDe(t *testing.T, b *Boveda) resumenDeBoveda {
	t.Helper()
	var r resumenDeBoveda
	r.ID, r.Serie = b.ID(), b.Serie()
	for _, e := range b.Buscar("") {
		v, _ := b.Ver(e.ID)
		r.Entradas = append(r.Entradas, canon(v))
	}
	for _, e := range b.Papelera() {
		v, _ := b.Ver(e.ID)
		r.Entradas = append(r.Entradas, canon(v))
	}
	r.Excluidos = b.Excluidos()
	p, err := b.Posesion()
	if err != nil {
		t.Fatal(err)
	}
	r.Posesion = fmt.Sprintf("%x", p)
	return r
}

func mismoResumen(t *testing.T, donde string, go_, ext resumenDeBoveda) {
	t.Helper()
	if go_.ID != ext.ID || go_.Posesion != ext.Posesion || strings.Join(go_.Entradas, "\n") != strings.Join(ext.Entradas, "\n") ||
		strings.Join(go_.Excluidos, ",") != strings.Join(ext.Excluidos, ",") {
		t.Fatalf("%s: Go y la extensión no ven la misma bóveda\n  Go:        %+v\n  extensión: %+v", donde, go_, ext)
	}
}

func entradasRaras() []Entrada {
	var out []Entrada
	for i, s := range textosRaros[:5] {
		e := Entrada{Titulo: "Entrada " + s, Usuario: s, Secreto: "secreto " + s, Sitios: []string{"x.com"}}
		if i == 1 {
			e.Tipo, e.Notas, e.Secreto = TipoNota, "una nota con <html> & \u2028", ""
		}
		if i == 2 {
			e.Extra = map[string]json.RawMessage{"deUnaVersionNueva": json.RawMessage(`{"a":[1,"b"]}`)}
		}
		out = append(out, e)
	}
	return out
}

// Una bóveda de Go la abre la extensión con la maestra y con la clave de
// recuperación tecleada de cualquier manera, la cambia, y Go abre lo que escribió.
func TestCruzadaBovedaDeGoEnLaExtension(t *testing.T) {
	cruzada.Activa(t)
	b, rec, ruta := nueva(t)
	for _, e := range entradasRaras() {
		if err := b.Poner(e); err != nil {
			t.Fatal(err)
		}
	}
	borrar := b.Buscar("")[0].ID
	if err := b.Excluir("nunca.com"); err != nil {
		t.Fatal(err)
	}
	texto, _ := os.ReadFile(ruta)
	suya := resumenDeBoveda{}
	for _, llave := range []string{maestra, strings.ToLower(strings.ReplaceAll(rec, "-", ""))} {
		cruzada.Pedir(t, map[string]any{"orden": "abrir", "texto": string(texto), "llave": llave}, &suya)
		mismoResumen(t, "la extensión abre la de Go", resumenDe(t, b), suya)
	}

	cruzada.Pedir(t, map[string]any{
		"orden": "modificar", "texto": string(texto), "llave": maestra,
		"poner":  []Entrada{{Titulo: "Desde el navegador", Usuario: "yo", Secreto: "nueva<&>", Sitios: []string{"y.es"}}},
		"borrar": []string{borrar}, "excluir": []string{"otro.es"},
	}, &suya)
	for _, llave := range []string{maestra, rec} {
		otra, err := AbrirEnMemoria([]byte(suya.Texto), llave)
		if err != nil {
			t.Fatalf("Go no abre lo que escribió la extensión con %q: %v", llave, err)
		}
		mismoResumen(t, "Go abre lo que escribió la extensión", resumenDe(t, otra), suya)
		if otra.Cuantas() != 5 || otra.EnLaPapelera() != 1 || !otra.Excluido("otro.es") {
			t.Fatalf("no llega lo que hizo la extensión: %d vivas, %d en la papelera", otra.Cuantas(), otra.EnLaPapelera())
		}
	}
}

// Una bóveda nueva de la extensión se abre en Go con las dos llaves.
func TestCruzadaBovedaDeLaExtensionEnGo(t *testing.T) {
	var suya resumenDeBoveda
	cruzada.Pedir(t, map[string]any{"orden": "crear", "maestra": maestra, "poner": entradasRaras()}, &suya)
	for _, llave := range []string{maestra, suya.Recuperacion} {
		b, err := AbrirEnMemoria([]byte(suya.Texto), llave)
		if err != nil {
			t.Fatalf("Go no abre la bóveda de la extensión con %q: %v", llave, err)
		}
		mismoResumen(t, "bóveda nueva de la extensión", resumenDe(t, b), suya)
	}
}

// Lo que sube uno lo funde el otro, y la versión sellada dentro se respeta en los
// dos sentidos: una que no cuadra no se funde en ninguno.
func TestCruzadaSubidaYFusion(t *testing.T) {
	cruzada.Activa(t)
	b, _, ruta := nueva(t)
	for _, e := range entradasRaras()[:3] {
		if err := b.Poner(e); err != nil {
			t.Fatal(err)
		}
	}
	base, _, err := b.PrepararSubida(4)
	if err != nil {
		t.Fatal(err)
	}
	texto, _ := os.ReadFile(ruta)

	// La extensión cambia su copia y la sube como la 5.
	var suya resumenDeBoveda
	cruzada.Pedir(t, map[string]any{"orden": "modificar", "texto": string(texto), "llave": maestra,
		"poner": []Entrada{{Titulo: "Del navegador", Secreto: "x"}}}, &suya)
	var subida struct {
		Texto string `json:"texto"`
	}
	cruzada.Pedir(t, map[string]any{"orden": "subida", "texto": suya.Texto, "llave": maestra, "version": 5}, &subida)

	// Go la funde contra la 4: le llega la entrada nueva.
	if err := b.Poner(Entrada{Titulo: "De la aplicación", Secreto: "y"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Fundir([]byte(subida.Texto), 6, base, OpcionesDeFusion{}); err == nil {
		t.Fatal("Go funde una subida de la extensión que dice otra versión")
	}
	f, err := b.Fundir([]byte(subida.Texto), 5, base, OpcionesDeFusion{})
	if err != nil {
		t.Fatalf("Go no funde lo que subió la extensión: %v", err)
	}
	if b.Cuantas() != 5 || !f.Subir {
		t.Fatalf("tras fundir: %d entradas y subir=%v", b.Cuantas(), f.Subir)
	}

	// Y al revés: la extensión funde lo que sube Go como la 6, con la misma hora.
	deGo, _, err := b.PrepararSubida(6)
	if err != nil {
		t.Fatal(err)
	}
	var fundida resumenDeBoveda
	cruzada.Pedir(t, map[string]any{"orden": "fundirBoveda", "local": suya.Texto, "remoto": string(deGo), "version": 6,
		"base": string(base), "llave": maestra, "ahora": "2026-09-22T13:14:15Z"}, &fundida)
	mismoResumen(t, "la extensión funde lo que subió Go", resumenDe(t, b), fundida)
}

// La identidad para compartir, igual en los dos (ADR 0043).
//
// **Si las llaves no coinciden, lo que se manda desde la ventana no lo abre el
// navegador**, y al revés. Y si la huella no coincide, dos personas comparándola
// por teléfono creerían que están hablando de identidades distintas.
func TestCruzadaIdentidad(t *testing.T) {
	cruzada.Activa(t)
	semillas := []string{
		"0000000000000000000000000000000000000000000000000000000000000000",
		"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}
	var suyas []struct {
		Cifrado string `json:"cifrado"`
		Firma   string `json:"firma"`
		Huella  string `json:"huella"`
		Suite   string `json:"suite"`
	}
	cruzada.Pedir(t, map[string]any{"orden": "identidad", "semillas": semillas}, &suyas)
	if len(suyas) != len(semillas) {
		t.Fatalf("la extensión ha devuelto %d identidades de %d semillas", len(suyas), len(semillas))
	}
	for i, s := range semillas {
		semilla, _ := hex.DecodeString(s)
		mia, err := publicaDe(&identidad{Semilla: b64.EncodeToString(semilla), Suite: Suite})
		if err != nil {
			t.Fatal(err)
		}
		if suyas[i].Suite != mia.Suite {
			t.Errorf("semilla %s: suites distintas, %q y %q", s[:8], mia.Suite, suyas[i].Suite)
		}
		if suyas[i].Cifrado != hex.EncodeToString(mia.Cifrado) {
			t.Errorf("semilla %s: la llave de cifrado de Go es %x y la de la extensión %s", s[:8], mia.Cifrado, suyas[i].Cifrado)
		}
		if suyas[i].Firma != hex.EncodeToString(mia.Firma) {
			t.Errorf("semilla %s: la llave de firma de Go es %x y la de la extensión %s", s[:8], mia.Firma, suyas[i].Firma)
		}
		if suyas[i].Huella != mia.Huella {
			t.Errorf("semilla %s: la huella de Go es %s y la de la extensión %s", s[:8], mia.Huella, suyas[i].Huella)
		}
	}
}

// **Y al fundir, los dos eligen la misma identidad.** Como sección desconocida
// ganaría la del servidor y cada lado podría quedarse con una distinta; entonces
// lo que le mandaran a uno no lo abriría el otro.
func TestCruzadaFundirIdentidad(t *testing.T) {
	cruzada.Activa(t)
	conIdentidad := func(semilla, creada string) contenido {
		return contenido{
			Entradas:  []Entrada{{ID: "a", Tipo: TipoCredencial, Titulo: "Uno", Creada: "2026-09-01T00:00:00Z", Cambiada: "2026-09-01T00:00:00Z", Revision: 1}},
			Identidad: &identidad{Semilla: semilla, Creada: creada, Suite: Suite},
		}
	}
	// La de la izquierda es más antigua: tiene que ganar la mire quien la mire.
	l := conIdentidad("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "2026-01-01T00:00:00Z")
	r := conIdentidad("BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", "2026-06-01T00:00:00Z")

	var f Fusion
	mio := fundirContenido(l, r, nil, time.Unix(0, 0).UTC(), &f)
	if mio.Identidad == nil || mio.Identidad.Semilla != l.Identidad.Semilla {
		t.Fatalf("en Go no gana la más antigua: %+v", mio.Identidad)
	}

	// Y la extensión, con el mismo caso, tiene que dar el mismo contenido: se
	// compara la forma canónica entera, no solo la identidad.
	var suyos []struct {
		Contenido string `json:"contenido"`
	}
	cruzada.Pedir(t, map[string]any{
		"orden": "fundir",
		"casos": []map[string]any{{
			"l": l, "r": r, "b": nil,
			"sobresL": []sobre{}, "sobresR": []sobre{}, "sobresB": []sobre{},
			"ahora": "1970-01-01T00:00:00Z",
		}},
	}, &suyos)
	if len(suyos) != 1 {
		t.Fatalf("la extensión ha devuelto %d resultados", len(suyos))
	}
	if suyos[0].Contenido != canonDeContenido(t, mio) {
		t.Fatalf("funden distinto:\nGo:        %s\nextensión: %s", canonDeContenido(t, mio), suyos[0].Contenido)
	}
}
