package tema

import (
	"math"
	"testing"
)

// pareja es una combinación de colores que la interfaz usa de verdad, con el
// mínimo que le toca y para qué sirve. La lista es explícita a propósito: medir
// todas las combinaciones posibles de la paleta daría fallos en parejas que nunca
// se pintan juntas, y acabaría enseñando a ignorar el resultado.
//
// Mismo modelo que apps/web/scripts/contraste.mjs de Tempero, que corre en cada
// lint. Aquí corre en cada `go test`, así que una paleta que no cumple no llega a
// compilar un binario.
type pareja struct {
	frente, fondo RGB
	minimo        float64
	proposito     string
}

func parejasDe(t Tema) []pareja {
	return []pareja{
		// Texto sobre las cuatro superficies.
		{t.Tinta, t.Lienzo, AANormal, "títulos sobre el lienzo"},
		{t.Cuerpo, t.Lienzo, AANormal, "texto corriente sobre el lienzo"},
		{t.Apagado, t.Lienzo, AANormal, "ayudas y pies sobre el lienzo"},
		{t.Tinta, t.Tarjeta, AANormal, "títulos dentro de un panel"},
		{t.Cuerpo, t.Tarjeta, AANormal, "texto dentro de un panel"},
		{t.Apagado, t.Tarjeta, AANormal, "etiquetas dentro de un panel"},
		{t.Cuerpo, t.Suave, AANormal, "texto sobre la superficie suave"},
		{t.Tinta, t.Elevada, AANormal, "texto sobre la superficie elevada"},
		{t.Cuerpo, t.Elevada, AANormal, "texto secundario sobre la elevada"},

		// El acento como texto: el resultado cifrado, el foco, los enlaces.
		{t.Acento, t.Lienzo, AANormal, "el resultado cifrado sobre el lienzo"},
		{t.Acento, t.Tarjeta, AANormal, "el acento dentro de un panel"},

		// El acento como relleno: botón activo, cabecera de marca.
		{t.SobreAcento, t.Relleno, AANormal, "texto sobre el relleno de acento"},
		{t.SobreAcento, t.RellenoVivo, AANormal, "texto sobre el acento pulsado"},

		// Estados, que siempre se pintan como texto.
		{t.Exito, t.Lienzo, AANormal, "«cifrado» sobre el lienzo"},
		{t.Aviso, t.Lienzo, AANormal, "avisos sobre el lienzo"},
		{t.Error, t.Lienzo, AANormal, "«clave incorrecta» sobre el lienzo"},
		{t.Exito, t.Tarjeta, AANormal, "estado correcto dentro de un panel"},
		{t.Aviso, t.Tarjeta, AANormal, "aviso dentro de un panel"},
		{t.Error, t.Tarjeta, AANormal, "error dentro de un panel"},

		// Botones. El primario es el amarillo como fondo, que es para lo que sirve;
		// el secundario vive sobre la superficie de tarjeta, y al pasar el puntero
		// sube un escalón en la escala de superficies en vez de estrenar un color.
		{t.SobreAcento, t.Relleno, AANormal, "botón primario"},
		{t.SobreAcento, t.RellenoVivo, AANormal, "botón primario con el puntero encima"},
		{t.Cuerpo, t.Tarjeta, AANormal, "botón secundario"},
		{t.Tinta, t.Elevada, AANormal, "botón secundario con el puntero encima"},
		{t.Acento, t.Tarjeta, AANormal, "botón secundario en la pantalla activa"},

		// Teclas dibujadas como teclas en el pie de ayuda.
		{t.Cuerpo, t.Elevada, AANormal, "nombre de una tecla"},

		// Migas de pan: el tramo actual en acento, los anteriores apagados.
		{t.Apagado, t.Lienzo, AANormal, "tramo anterior de las migas"},
		{t.Acento, t.Lienzo, AANormal, "tramo actual de las migas"},

		// Elementos que no son texto: les basta 3:1.
		//
		// El foco de un campo lo marca el acento y no un gris más fuerte. Es lo que
		// hace ClickHouse, y además es lo único que funciona: su filete fuerte
		// #3a3a3a sobre el lienzo da 1,74:1, invisible como señal de foco.
		{t.Acento, t.Lienzo, AAGrande, "borde de un campo con el foco"},
		{t.Acento, t.Tarjeta, AAGrande, "borde con foco dentro de un panel"},
		{t.Acento, t.Lienzo, AAGrande, "barra de marca ▍ y medidores"},
		{t.Filete, t.Lienzo, 1.2, "filete de separación sobre el lienzo"},
		{t.FileteFuerte, t.Lienzo, 1.5, "separador de sección, más presente que el filete"},
	}
}

func TestContrasteDeLosDosTemas(t *testing.T) {
	for _, tema := range []Tema{TemaOscuro, TemaClaro} {
		t.Run(tema.Nombre, func(t *testing.T) {
			for _, p := range parejasDe(tema) {
				ratio := Contraste(p.frente, p.fondo)
				if ratio < p.minimo {
					t.Errorf("MAL  %s sobre %s = %.2f:1, hace falta %.1f:1 — %s",
						p.frente.Hex(), p.fondo.Hex(), ratio, p.minimo, p.proposito)
					continue
				}
				t.Logf("ok   %s sobre %s = %5.2f:1 — %s",
					p.frente.Hex(), p.fondo.Hex(), ratio, p.proposito)
			}
		})
	}
}

// El azul de botón del sistema no cumple AA con texto blanco encima: hay que
// oscurecerlo. Si alguien «corrige» el relleno poniendo el azul original, esto
// salta.
func TestElRellenoDeAccionSeLee(t *testing.T) {
	for _, tm := range []Tema{TemaClaro, TemaOscuro} {
		if r := Contraste(tm.SobreAcento, tm.Relleno); r < AANormal {
			t.Errorf("tema %s: el texto del botón de acción da %.2f:1", tm.Nombre, r)
		}
	}

	// Y el supuesto de partida: los azules del sistema, tal cual, no llegan.
	if r := Contraste(MustParseHex("#ffffff"), azulClaro); r >= AANormal {
		t.Errorf("el supuesto ha cambiado: blanco sobre #007aff da %.2f:1", r)
	}
}

func TestRellenoLegible(t *testing.T) {
	blanco := RGB{255, 255, 255}
	for _, base := range []string{"#007aff", "#0a84ff", "#34c759", "#ff3b30"} {
		fondo := RellenoLegible(MustParseHex(base), blanco, AANormal)
		if r := Contraste(blanco, fondo); r < AANormal {
			t.Errorf("sobre %s ajustado a %s, el blanco da %.2f:1", base, fondo.Hex(), r)
		}
	}
}

func TestAcentoLegible(t *testing.T) {
	blanco, negro := RGB{255, 255, 255}, RGB{10, 10, 10}

	casos := []struct {
		acento, fondo RGB
		nombre        string
	}{
		{MustParseHex("#faff69"), blanco, "un amarillo sobre blanco"},
		{MustParseHex("#b1f100"), blanco, "lima de Webcafeína sobre blanco"},
		{rojoSistema, blanco, "el rojo del sistema sobre blanco"},
		{azulClaro, blanco, "el azul del sistema sobre blanco"},
		{MustParseHex("#5a3519"), negro, "marrón de Webcafeína sobre negro"},
		{MustParseHex("#171009"), negro, "tinta de Webcafeína sobre negro"},
	}
	for _, c := range casos {
		got := AcentoLegible(c.acento, c.fondo, AANormal)
		if r := Contraste(got, c.fondo); r < AANormal {
			t.Errorf("%s: AcentoLegible dio %s, que da %.2f:1", c.nombre, got.Hex(), r)
		}
	}
}

// Un color que ya cumple no se toca: corregir de más aguaría la identidad del
// sistema sin motivo.
func TestAcentoLegibleNoTocaLoQueYaCumple(t *testing.T) {
	amarillo := MustParseHex("#faff69")
	if got := AcentoLegible(amarillo, MustParseHex("#0a0a0a"), AANormal); got != amarillo {
		t.Errorf("un amarillo sobre negro ya cumple y lo ha cambiado a %s", got.Hex())
	}
}

func TestLuminanciaRelativa(t *testing.T) {
	casos := []struct {
		hex    string
		quiero float64
	}{
		{"#000000", 0},
		{"#ffffff", 1},
		{"#808080", 0.2159},
	}
	for _, c := range casos {
		got := LuminanciaRelativa(MustParseHex(c.hex))
		if math.Abs(got-c.quiero) > 0.001 {
			t.Errorf("LuminanciaRelativa(%s) = %.4f, quiero %.4f", c.hex, got, c.quiero)
		}
	}

	// Los extremos de la escala, que es donde se ve si la fórmula está bien.
	if r := Contraste(RGB{0, 0, 0}, RGB{255, 255, 255}); math.Abs(r-21) > 0.01 {
		t.Errorf("negro sobre blanco = %.2f:1, tiene que ser 21", r)
	}
}

func TestParseHex(t *testing.T) {
	for _, s := range []string{"#faff69", "faff69", "  #FAFF69  "} {
		if c, err := ParseHex(s); err != nil || c != MustParseHex("#faff69") {
			t.Errorf("ParseHex(%q) = %v, %v", s, c, err)
		}
	}
	for _, s := range []string{"", "#fff", "#gggggg", "faff6"} {
		if _, err := ParseHex(s); err == nil {
			t.Errorf("ParseHex(%q) debería fallar", s)
		}
	}
}
