package codigos

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

// La semilla de las dos especificaciones: los veinte bytes «12345678901234567890»
// en base32, escritos a mano. Escrita y no calculada a propósito: así el
// descifrado del alfabeto también queda comprobado contra algo de fuera.
const semillaDelRFC = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

// Los vectores de RFC 4226, apéndice D. Son HOTP —el contador va de 0 a 9— y
// aquí se ejercitan como TOTP poniendo el reloj en el segundo que da ese mismo
// contador, que es exactamente lo que TOTP hace por dentro.
func TestLosVectoresDelRFC4226(t *testing.T) {
	esperados := []string{
		"755224", "287082", "359152", "969429", "338314",
		"254676", "287922", "162583", "399871", "520489",
	}
	s, err := Leer(semillaDelRFC)
	if err != nil {
		t.Fatalf("no lee la semilla del RFC: %v", err)
	}
	for contador, quiero := range esperados {
		momento := time.Unix(int64(contador)*30, 0)
		tengo, err := s.En(momento)
		if err != nil {
			t.Fatalf("contador %d: %v", contador, err)
		}
		if tengo != quiero {
			t.Errorf("contador %d: da %s y el RFC dice %s", contador, tengo, quiero)
		}
	}
}

// Los vectores de RFC 6238, apéndice B. Son de ocho cifras, y **cada algoritmo
// lleva su propia semilla**: la tabla del RFC alarga la de SHA-1 repitiéndola
// hasta el tamaño del bloque de cada picadora, cosa que no se dice en la tabla y
// que ha costado horas a mucha gente.
func TestLosVectoresDelRFC6238(t *testing.T) {
	enBase32 := func(s string) string {
		return base32.StdEncoding.EncodeToString([]byte(s))
	}
	sha1 := enBase32("12345678901234567890")
	sha256 := enBase32("12345678901234567890123456789012")
	sha512 := enBase32("1234567890123456789012345678901234567890" +
		"123456789012345678901234")

	casos := []struct {
		segundo   int64
		algoritmo string
		semilla   string
		quiero    string
	}{
		{59, "SHA1", sha1, "94287082"},
		{59, "SHA256", sha256, "46119246"},
		{59, "SHA512", sha512, "90693936"},
		{1111111109, "SHA1", sha1, "07081804"},
		{1111111111, "SHA1", sha1, "14050471"},
		{1234567890, "SHA1", sha1, "89005924"},
		{2000000000, "SHA1", sha1, "69279037"},
		{20000000000, "SHA1", sha1, "65353130"},
		{20000000000, "SHA256", sha256, "77737706"},
		{20000000000, "SHA512", sha512, "47863826"},
	}
	for _, c := range casos {
		s := Semilla{
			Clave: debeLeer(t, c.semilla).Clave, Digitos: 8,
			Periodo: 30 * time.Second, Algoritmo: c.algoritmo,
		}
		tengo, err := s.En(time.Unix(c.segundo, 0))
		if err != nil {
			t.Fatalf("%s en %d: %v", c.algoritmo, c.segundo, err)
		}
		if tengo != c.quiero {
			t.Errorf("%s en %d: da %s y el RFC dice %s",
				c.algoritmo, c.segundo, tengo, c.quiero)
		}
	}
}

// Una semilla se copia de una pantalla, así que llega con espacios, en
// minúsculas y a veces con guiones. Todas esas formas son la misma semilla.
func TestLaSemillaLlegaComoLaCopiaUnHumano(t *testing.T) {
	formas := []string{
		semillaDelRFC,
		strings.ToLower(semillaDelRFC),
		"gezd gnbv gy3t qojq gezd gnbv gy3t qojq",
		"GEZD-GNBV-GY3T-QOJQ-GEZD-GNBV-GY3T-QOJQ",
		semillaDelRFC + "======",
		"  " + semillaDelRFC + "\n",
	}
	quiero, err := debeLeer(t, semillaDelRFC).En(time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range formas {
		s, err := Leer(f)
		if err != nil {
			t.Errorf("«%s» no se entiende: %v", f, err)
			continue
		}
		tengo, err := s.En(time.Unix(0, 0))
		if err != nil {
			t.Errorf("«%s»: %v", f, err)
			continue
		}
		if tengo != quiero {
			t.Errorf("«%s» da %s y debería dar %s", f, tengo, quiero)
		}
	}
}

// Y lo contrario: una semilla a la que le falta o le sobra algo **no se
// adivina**. Un carácter de más que se ignorara en silencio daría códigos que no
// valen, y el fallo aparecería en la pantalla del servicio, lejos de aquí.
func TestUnaSemillaRotaNoSeAdivina(t *testing.T) {
	malas := []string{
		"", "   ", "=====",
		"GEZDGNBV!GY3TQOJQ",                        // un carácter que no es del alfabeto
		"GEZDGNBV0GY3TQOJQ",                        // el cero no está en base32
		"GEZDGNBV1GY3TQOJQ",                        // el uno tampoco
		"GEZDGNBVGY3TQOJQG",                        // sobra media letra
		"otpauth://hotp/x?secret=" + semillaDelRFC, // contador, no reloj
		"otpauth://totp/x?secret=" + semillaDelRFC + "&digits=4",
		"otpauth://totp/x?secret=" + semillaDelRFC + "&digits=20",
		"otpauth://totp/x?secret=" + semillaDelRFC + "&period=0",
		"otpauth://totp/x?secret=" + semillaDelRFC + "&algorithm=MD5",
		"otpauth://totp/x?secret=00000000",
		"otpauth://totp/x",
	}
	for _, m := range malas {
		if s, err := Leer(m); err == nil {
			t.Errorf("«%s» se ha tomado por buena: %+v", m, s)
		} else if err.Error() == "" || err.Error()[:1] != strings.ToUpper(err.Error()[:1]) {
			t.Errorf("«%s»: el error no empieza en mayúscula: %q", m, err)
		}
	}
}

// La URI que exportan varios gestores, con sus parámetros. Lo que no venga toma
// el valor por defecto del RFC.
func TestLaURIDeOtroGestor(t *testing.T) {
	s, err := Leer("otpauth://totp/Banco:yo%40ejemplo.es?secret=" + semillaDelRFC +
		"&issuer=Banco&algorithm=SHA256&digits=8&period=60")
	if err != nil {
		t.Fatal(err)
	}
	if s.Digitos != 8 || s.Periodo != 60*time.Second || s.Algoritmo != "SHA256" {
		t.Errorf("no ha leído los parámetros: %+v", s)
	}
	if s.Cuenta != "Banco:yo@ejemplo.es" {
		t.Errorf("la etiqueta es %q", s.Cuenta)
	}

	// Y sin parámetros, los de siempre.
	s, err = Leer("otpauth://totp/x?secret=" + semillaDelRFC)
	if err != nil {
		t.Fatal(err)
	}
	if s.Digitos != 6 || s.Periodo != 30*time.Second || s.Algoritmo != "SHA1" {
		t.Errorf("los valores por defecto no son los del RFC: %+v", s)
	}
	// La misma semilla por los dos caminos tiene que dar el mismo código.
	porURI, _ := s.En(time.Unix(0, 0))
	pelada, _ := debeLeer(t, semillaDelRFC).En(time.Unix(0, 0))
	if porURI != pelada {
		t.Errorf("la URI da %s y la semilla pelada %s", porURI, pelada)
	}
}

// Lo que le queda de vida al código se cuenta contra el reloj, no contra cuándo
// se preguntó: dos códigos mirados con un segundo de diferencia caducan a la
// vez, porque los intervalos son fijos desde 1970.
func TestLoQueLeQuedaAlCodigoVaConElReloj(t *testing.T) {
	s := debeLeer(t, semillaDelRFC)
	casos := []struct {
		segundo int64
		quedan  time.Duration
	}{
		{0, 30 * time.Second},
		{1, 29 * time.Second},
		{29, 1 * time.Second},
		{30, 30 * time.Second},
		{1111111109, 1 * time.Second},
	}
	for _, c := range casos {
		if tengo := s.Quedan(time.Unix(c.segundo, 0)); tengo != c.quedan {
			t.Errorf("en el segundo %d quedan %v y deberían quedar %v",
				c.segundo, tengo, c.quedan)
		}
	}

	// Y con un periodo que no sea el de siempre, también.
	s.Periodo = 60 * time.Second
	if q := s.Quedan(time.Unix(1111111109, 0)); q != 31*time.Second {
		t.Errorf("con periodo de 60 quedan %v", q)
	}
}

// El código no cambia dentro de su intervalo y sí cambia al saltar al siguiente.
// Parece obvio; es justo lo que rompe un error de una unidad en la división.
func TestElCodigoDuraSuIntervaloYNiUnSegundoMas(t *testing.T) {
	s := debeLeer(t, semillaDelRFC)
	dentro, _ := s.En(time.Unix(1111111109, 0))
	mismo, _ := s.En(time.Unix(1111111080, 0)) // el primer segundo del intervalo
	siguiente, _ := s.En(time.Unix(1111111110, 0))
	if dentro != mismo {
		t.Errorf("dentro del mismo intervalo da dos códigos: %s y %s", mismo, dentro)
	}
	if dentro == siguiente {
		t.Errorf("al saltar de intervalo sigue dando %s", dentro)
	}
}

func debeLeer(t *testing.T, texto string) Semilla {
	t.Helper()
	s, err := Leer(texto)
	if err != nil {
		t.Fatalf("no lee «%s»: %v", texto, err)
	}
	return s
}
