// Package tema es la fuente de verdad del color de Esfinge: la paleta, el
// cálculo de contraste y la generación de los tokens que consume la interfaz.
//
// No depende de ninguna tecnología de presentación. Nació sirviendo a una
// interfaz de terminal y ahora sirve a una de escritorio sin cambiar una línea,
// que es exactamente lo que se le pide a esta capa.
package tema

import (
	"fmt"
	"math"
	"strings"
)

// Umbrales de la WCAG. AANormal es el mínimo para texto corriente; AAGrande vale
// para texto grande, que en un terminal no existe, y para elementos de interfaz
// que no son texto: bordes, separadores, indicadores.
const (
	AANormal = 4.5
	AAGrande = 3.0
)

// RGB es un color en el espacio de 8 bits por canal.
type RGB struct{ R, G, B uint8 }

// ParseHex acepta «#rrggbb» y «rrggbb».
func ParseHex(s string) (RGB, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return RGB{}, fmt.Errorf("El color %q: hacen falta seis dígitos hexadecimales", s)
	}
	var c RGB
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &c.R, &c.G, &c.B); err != nil {
		return RGB{}, fmt.Errorf("El color %q no es hexadecimal", s)
	}
	return c, nil
}

// MustParseHex es para las constantes de la paleta, que son literales del código
// y no entrada de nadie: si una está mal, el programa no debe arrancar.
func MustParseHex(s string) RGB {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Hex devuelve la forma «#rrggbb», que es lo que entiende lipgloss.
func (c RGB) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// canalLineal deshace la corrección gamma de sRGB.
func canalLineal(v uint8) float64 {
	f := float64(v) / 255
	if f <= 0.04045 {
		return f / 12.92
	}
	return math.Pow((f+0.055)/1.055, 2.4)
}

// LuminanciaRelativa según la definición de la WCAG.
func LuminanciaRelativa(c RGB) float64 {
	return 0.2126*canalLineal(c.R) + 0.7152*canalLineal(c.G) + 0.0722*canalLineal(c.B)
}

// Contraste devuelve la razón entre dos colores, de 1 a 21.
func Contraste(a, b RGB) float64 {
	la, lb := LuminanciaRelativa(a), LuminanciaRelativa(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// AcentoLegible oscurece —o aclara, si el fondo es oscuro— un color de marca en
// pasos del 8 % hasta que alcanza el mínimo pedido sobre ese fondo.
//
// Es el mismo procedimiento que readableAccent en el paquete design-tokens de
// Webcafeína, y hace falta por el mismo motivo: los amarillos y los limas de
// marca son estupendos como relleno y catastróficos como texto sobre claro. El
// #faff69 de ClickHouse sobre blanco da 1,1:1, igual que el #B1F100 de la casa.
// Ninguno de los dos se toca; se usa esta variante para el texto y el original
// para los rellenos.
func AcentoLegible(acento, fondo RGB, minimo float64) RGB {
	if Contraste(acento, fondo) >= minimo {
		return acento
	}

	// Sobre un fondo claro hay que ir a negro; sobre uno oscuro, a blanco.
	haciaNegro := LuminanciaRelativa(fondo) > 0.5

	c := acento
	for i := 0; i < 24; i++ {
		if haciaNegro {
			c = RGB{escalar(c.R, 0.92), escalar(c.G, 0.92), escalar(c.B, 0.92)}
		} else {
			c = RGB{aclarar(c.R), aclarar(c.G), aclarar(c.B)}
		}
		if Contraste(c, fondo) >= minimo {
			return c
		}
	}
	// 24 pasos del 8 % llegan a negro o a blanco puro; si aun así no cumple, es
	// que el fondo es de un gris medio imposible y más vale decirlo con el
	// extremo que con un color a medias.
	if haciaNegro {
		return RGB{0, 0, 0}
	}
	return RGB{255, 255, 255}
}

// RellenoLegible oscurece un color de fondo hasta que el texto que va encima se
// lee sobre él.
//
// Es el reverso de AcentoLegible: allí se mueve la tinta, aquí el fondo. Hace
// falta para los botones de acción, donde el color viene dado por la convención
// del sistema y lo que se puede mover es el fondo. El azul de botón de macOS con
// texto blanco da 3,6:1; para cumplir AA hay que bajarlo un par de escalones, y
// el resultado sigue leyéndose como el azul del sistema.
func RellenoLegible(fondo, encima RGB, minimo float64) RGB {
	c := fondo
	for i := 0; i < 24 && Contraste(encima, c) < minimo; i++ {
		c = Oscurecer(c, 0.92)
	}
	return c
}

// Oscurecer multiplica los tres canales por un factor. Con 0,85 se obtiene el
// estado «pulsado» de un botón, que es como lo resuelven los dos sistemas.
func Oscurecer(c RGB, factor float64) RGB {
	return RGB{escalar(c.R, factor), escalar(c.G, factor), escalar(c.B, factor)}
}

func escalar(v uint8, f float64) uint8 {
	return uint8(math.Round(float64(v) * f))
}

func aclarar(v uint8) uint8 {
	return uint8(math.Round(float64(v) + (255-float64(v))*0.08))
}
