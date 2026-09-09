package iconos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
)

// El `.ico` de toda la vida, que es donde muchos sitios tienen su único icono.
//
// **Se descartó al principio y los datos dijeron lo contrario.** El argumento
// para dejarlo fuera era bueno —Go no sabe decodificar ICO, y analizar a mano un
// formato de los noventa sobre datos de un tercero no compensa— pero al medir
// contra sitios de verdad solo cuatro de doce daban icono, y varios de los que
// faltaban tenían el suyo justo ahí.
//
// Lo que se hace es lo barato y lo seguro: **ICO es un contenedor**, y desde hace
// años casi todos llevan un PNG dentro. Se lee el índice —que son seis bytes de
// cabecera y dieciséis por entrada—, se busca la imagen más grande, y **si empieza
// por la firma de un PNG se le pasa al decodificador de siempre**. Si lleva un
// mapa de bits de los antiguos, se deja: descodificar BMP a mano es exactamente el
// trabajo que no queríamos hacer.

var firmaPNG = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

var errNoEsICO = errors.New("no es un ICO con PNG dentro")

// deUnICO saca la imagen más grande de un ICO, venga como venga.
//
// Devuelve o bien bytes de un PNG —para que los decodifique quien sabe— o bien
// una imagen ya montada, cuando dentro había un mapa de bits de 32 bits.
func deUnICO(datos []byte) ([]byte, image.Image, error) {
	trozo, err := elMayorDeUnICO(datos)
	if err != nil {
		return nil, nil, err
	}
	if bytes.HasPrefix(trozo, firmaPNG) {
		return trozo, nil, nil
	}
	img, err := elBMPde32DeUnICO(trozo)
	if err != nil {
		return nil, nil, err
	}
	return nil, img, nil
}

// elMayorDeUnICO devuelve el trozo de la entrada más grande del índice.
func elMayorDeUnICO(datos []byte) ([]byte, error) {
	const cabecera = 6
	const porEntrada = 16
	if len(datos) < cabecera {
		return nil, errNoEsICO
	}
	// Reservado 0, tipo 1 (icono). Cualquier otra cosa no es un ICO.
	if binary.LittleEndian.Uint16(datos[0:2]) != 0 || binary.LittleEndian.Uint16(datos[2:4]) != 1 {
		return nil, errNoEsICO
	}
	cuantas := int(binary.LittleEndian.Uint16(datos[4:6]))
	if cuantas == 0 || len(datos) < cabecera+cuantas*porEntrada {
		return nil, errNoEsICO
	}

	var mejor []byte
	var mejorArea int
	for i := 0; i < cuantas; i++ {
		e := datos[cabecera+i*porEntrada:]
		// En un ICO, el cero significa 256: no cabía en un byte.
		ancho, alto := int(e[0]), int(e[1])
		if ancho == 0 {
			ancho = 256
		}
		if alto == 0 {
			alto = 256
		}
		largo := int(binary.LittleEndian.Uint32(e[8:12]))
		desde := int(binary.LittleEndian.Uint32(e[12:16]))

		// Todo lo que venga del fichero se comprueba contra su tamaño real antes de
		// usarlo como índice: es un fichero de un tercero, y aquí es donde un
		// desplazamiento inventado tumbaría el programa.
		if largo <= 0 || desde < 0 || desde+largo > len(datos) {
			continue
		}
		if area := ancho * alto; area > mejorArea {
			mejor, mejorArea = datos[desde:desde+largo], area
		}
	}
	if mejor == nil {
		return nil, errNoEsICO
	}
	return mejor, nil
}

// Y el otro contenido posible de un ICO: un mapa de bits de los de Windows, sin
// su cabecera de fichero.
//
// **También estaba descartado, y también lo desmintieron los datos.** Al medir,
// los tres sitios grandes que seguían sin icono —Google, Amazon y Netflix—
// llevaban exactamente esto. Lo que se hace es **solo el caso de 32 bits sin
// comprimir**, que es el que usan todos ellos y el único que se puede leer sin
// paletas, sin descompresión y sin ambigüedad: cabecera de cuarenta bytes, cuatro
// bytes por píxel, filas de abajo arriba. Cualquier otra profundidad se deja
// pasar y la entrada se queda con su cuadro de color.
//
// La aritmética se comprueba contra el tamaño real del trozo antes de tocar un
// solo byte: esto son datos de un tercero, y aquí es donde una multiplicación que
// se desborda se convierte en una lectura fuera de sitio.
func elBMPde32DeUnICO(datos []byte) (image.Image, error) {
	const cabeceraDIB = 40
	if len(datos) < cabeceraDIB {
		return nil, errNoEsICO
	}
	if binary.LittleEndian.Uint32(datos[0:4]) != cabeceraDIB {
		return nil, errNoEsICO // no es un BITMAPINFOHEADER
	}
	ancho := int(int32(binary.LittleEndian.Uint32(datos[4:8])))
	// En un ICO el alto viene **doblado**: la imagen y su máscara van juntas.
	alto := int(int32(binary.LittleEndian.Uint32(datos[8:12]))) / 2
	bits := int(binary.LittleEndian.Uint16(datos[14:16]))
	comprimido := binary.LittleEndian.Uint32(datos[16:20])

	if bits != 32 || comprimido != 0 {
		return nil, errNoEsICO
	}
	if ancho <= 0 || alto <= 0 || ancho > ladoMaximo || alto > ladoMaximo {
		return nil, errNoEsICO
	}
	necesarios := ancho * alto * 4
	if len(datos) < cabeceraDIB+necesarios {
		return nil, errNoEsICO
	}

	img := image.NewNRGBA(image.Rect(0, 0, ancho, alto))
	pixeles := datos[cabeceraDIB:]
	for y := 0; y < alto; y++ {
		// De abajo arriba, que es como guarda las filas este formato.
		fila := pixeles[(alto-1-y)*ancho*4:]
		for x := 0; x < ancho; x++ {
			p := fila[x*4:]
			// Y en orden azul, verde, rojo, alfa.
			img.SetNRGBA(x, y, color.NRGBA{R: p[2], G: p[1], B: p[0], A: p[3]})
		}
	}
	return img, nil
}
