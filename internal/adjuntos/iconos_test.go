package adjuntos

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLosIconosQueSeEmpaquetanNoSonUnCuadradoPlano, que nace de haberlo roto.
//
// # Qué pasó
//
// Probando el guion de la AppImage, el primer contenedor tenía el repositorio montado **con
// escritura** y su guion de prueba escribió ocho PNG de relleno —un cuadrado de un solo color— justo
// encima de los iconos de verdad. `git add -A` se los llevó, y **Esfinge salió dos versiones con un
// cuadrado mostaza por icono** en el menú de aplicaciones de Linux. Lo vio el cliente en su VM.
//
// Es la misma causa que el fichero `esfinge-mcp` suelto en la raíz, con el agravante de que esto
// **no se ve en ningún diff**: `git status` dice «Bin 2713 -> 209 bytes» y eso no parece nada.
//
// # Por qué esto y no otra cosa
//
// Un icono es invisible para todas las demás pruebas: compila, se empaqueta, se instala y no rompe
// nada. Lo único que lo delata es **mirarlo**, y lo que se puede afirmar sin ojos es que **no es un
// color plano**: cualquier icono de verdad tiene decenas de colores por el antialias, y el más
// pequeño de los nuestros —16×16— tiene 132. Un relleno tiene uno.
//
// No comprueba que el dibujo sea el bueno. Comprueba que hay un dibujo.
//
// # Y los mira por lo que el paquete declara, no por una lista
//
// Los PNG salen de los `src:` de `nfpm.yaml`, que es lo que de verdad se empaqueta: así un icono
// nuevo queda cubierto sin que nadie se acuerde de añadirlo aquí.
func TestLosIconosQueSeEmpaquetanNoSonUnCuadradoPlano(t *testing.T) {
	crudo, err := os.ReadFile("../../empaquetado/linux/nfpm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// `- src: build/linux/hicolor/64x64.png`
	src := regexp.MustCompile(`(?m)^\s*-?\s*src:\s*(\S+\.png)\s*$`)
	encontrados := src.FindAllStringSubmatch(string(crudo), -1)
	if len(encontrados) < 8 {
		t.Fatalf("esperaba al menos los ocho tamaños del icono en nfpm.yaml y he encontrado %d: o el fichero ha cambiado de forma o esta prueba no comprueba nada", len(encontrados))
	}

	for _, m := range encontrados {
		ruta := filepath.Join("../..", m[1])
		f, err := os.Open(ruta)
		if err != nil {
			t.Errorf("%s está en nfpm.yaml y no existe: el paquete no se podría armar", m[1])
			continue
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Errorf("%s no es un PNG legible: %v", m[1], err)
			continue
		}
		if n := colores(img, 40); n < 20 {
			t.Errorf("%s tiene %d color(es): eso no es un icono, es un relleno — y así salieron dos versiones, con un cuadrado mostaza en el menú de aplicaciones",
				m[1], n)
		}
	}
}

// colores cuenta cuántos distintos hay, parando en `tope` para no recorrer un PNG de 512×512
// entero cuando con los primeros ya está contestada la pregunta.
func colores(img image.Image, tope int) int {
	vistos := map[uint64]bool{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			vistos[uint64(r)<<48|uint64(g)<<32|uint64(bl)<<16|uint64(a)] = true
			if len(vistos) >= tope {
				return len(vistos)
			}
		}
	}
	return len(vistos)
}

// Y el del tipo de fichero no es el de la aplicación, que es un error que este proyecto **ya
// cometió dos veces**: `build/esf.png` fue una copia byte a byte de `build/appicon.png` hasta la
// 2.10.1, y en Linux el `.deb` apuntaba al icono de la aplicación hasta la 2.10.3.
//
// Lo que dice «documento» es el papel, no el emblema: si los dos ficheros vuelven a ser iguales, en
// el explorador un `.esf` y Esfinge se ven igual.
func TestElIconoDelTipoDeFicheroNoEsElDeLaAplicacion(t *testing.T) {
	var app, tipo []string
	src := regexp.MustCompile(`(?m)^\s*-?\s*src:\s*(\S+\.png)\s*$`)
	dst := regexp.MustCompile(`(?m)^\s*dst:\s*(\S+)\s*$`)
	crudo, err := os.ReadFile("../../empaquetado/linux/nfpm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	lineas := strings.Split(string(crudo), "\n")
	for i, l := range lineas {
		m := src.FindStringSubmatch(l)
		if m == nil || i+1 >= len(lineas) {
			continue
		}
		d := dst.FindStringSubmatch(lineas[i+1])
		if d == nil {
			continue
		}
		switch {
		case strings.Contains(d[1], "/apps/"):
			app = append(app, m[1])
		case strings.Contains(d[1], "/mimetypes/"):
			tipo = append(tipo, m[1])
		}
	}
	if len(app) == 0 || len(tipo) == 0 {
		t.Fatalf("no he sabido separar los iconos de la aplicación (%d) de los del tipo de fichero (%d)", len(app), len(tipo))
	}
	for _, a := range app {
		for _, b := range tipo {
			ba, err1 := os.ReadFile(filepath.Join("../..", a))
			bb, err2 := os.ReadFile(filepath.Join("../..", b))
			if err1 != nil || err2 != nil {
				continue
			}
			if string(ba) == string(bb) {
				t.Errorf("%s y %s son el mismo fichero: en el explorador, un .esf se vería igual que Esfinge", a, b)
			}
		}
	}
}
