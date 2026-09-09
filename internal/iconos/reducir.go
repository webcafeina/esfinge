package iconos

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
)

var base64Std = base64.StdEncoding

// reducir deja la imagen en un cuadrado de `lado` píxeles, promediando.
//
// **Está escrito a mano y son cuarenta líneas, en vez de traerse un módulo.**
// La biblioteca estándar de Go no sabe escalar —`image/draw` copia píxel a
// píxel— y el que sí sabe, `golang.org/x/image/draw`, sería **una dependencia
// nueva de verdad** en el binario que guarda las contraseñas de una empresa.
// Este proyecto tiene cuatro dependencias directas; añadir la quinta para
// encoger un icono no es un intercambio que salga a cuenta.
//
// Y hace falta hacerlo, no es un lujo: un `apple-touch-icon` viene a 180×180 y
// pesa entre diez y cuarenta kilobytes. Sesenta y cinco de ésos son megabytes;
// sesenta y cinco de 32×32 son ciento treinta kilobytes. La diferencia entre que
// esto quepa en la bóveda y que no, es esta función.
//
// El filtro es de caja: cada píxel de salida es el promedio de los de entrada que
// le corresponden. Para reducir es lo que hay que hacer —quedarse con uno de cada
// n deja los bordes dentados y las letras rotas— y para un icono da un resultado
// perfectamente bueno.
func reducir(origen image.Image, lado int) image.Image {
	caja := origen.Bounds()
	if caja.Dx() <= 0 || caja.Dy() <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, lado, lado))
	}

	// Lo que ya es igual o más pequeño no se toca: agrandar un icono de 16×16 a
	// 32×32 promediando solo lo emborrona.
	if caja.Dx() <= lado && caja.Dy() <= lado {
		fuera := image.NewNRGBA(image.Rect(0, 0, caja.Dx(), caja.Dy()))
		draw.Draw(fuera, fuera.Bounds(), origen, caja.Min, draw.Src)
		return fuera
	}

	fuera := image.NewNRGBA(image.Rect(0, 0, lado, lado))
	for y := 0; y < lado; y++ {
		// El trozo de la imagen de entrada que cae en esta fila de salida.
		desdeY := caja.Min.Y + y*caja.Dy()/lado
		hastaY := caja.Min.Y + (y+1)*caja.Dy()/lado
		if hastaY <= desdeY {
			hastaY = desdeY + 1
		}

		for x := 0; x < lado; x++ {
			desdeX := caja.Min.X + x*caja.Dx()/lado
			hastaX := caja.Min.X + (x+1)*caja.Dx()/lado
			if hastaX <= desdeX {
				hastaX = desdeX + 1
			}

			var r, g, b, a, n uint64
			for py := desdeY; py < hastaY; py++ {
				for px := desdeX; px < hastaX; px++ {
					// **Sin premultiplicar por el alfa**: un icono con transparencia
					// promediado sobre valores premultiplicados se oscurece por los
					// bordes. NRGBA da los componentes tal cual.
					c := color.NRGBAModel.Convert(origen.At(px, py)).(color.NRGBA)
					r += uint64(c.R)
					g += uint64(c.G)
					b += uint64(c.B)
					a += uint64(c.A)
					n++
				}
			}
			if n == 0 {
				continue
			}
			fuera.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n),
			})
		}
	}
	return fuera
}
