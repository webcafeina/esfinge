package qr

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// Temporal: dibuja un QR para escanearlo con un móvil de verdad.
func TestMirarlo(t *testing.T) {
	destino := os.Getenv("ESFINGE_QR_PNG")
	if destino == "" {
		t.Skip("sin ESFINGE_QR_PNG")
	}
	enlace := "WIFI:T:WPA;S:Esfinge de prueba;P:clave-de-prueba-1234;;"
	c, err := Nuevo(enlace)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("contenido:", enlace)
	fmt.Println("lado:", c.Lado())

	// Con los colores puestos a mano, que si no depende del tema de la terminal.
	const margen = 4
	for y := -margen; y < c.Lado()+margen; y++ {
		linea := ""
		for x := -margen; x < c.Lado()+margen; x++ {
			if c.Oscuro(x, y) {
				linea += "\033[40m  "
			} else {
				linea += "\033[107m  "
			}
		}
		fmt.Println(linea + "\033[0m")
	}

	const escala = 10
	lado := (c.Lado() + margen*2) * escala
	img := image.NewGray(image.Rect(0, 0, lado, lado))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	for y := 0; y < c.Lado(); y++ {
		for x := 0; x < c.Lado(); x++ {
			if !c.Oscuro(x, y) {
				continue
			}
			for dy := 0; dy < escala; dy++ {
				for dx := 0; dx < escala; dx++ {
					img.Set((x+margen)*escala+dx, (y+margen)*escala+dy, color.Gray{Y: 0})
				}
			}
		}
	}
	f, err := os.Create(destino)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	fmt.Println("escrito:", destino)
}
