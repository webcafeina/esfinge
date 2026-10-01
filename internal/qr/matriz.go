package qr

// El dibujo: los patrones fijos, los datos en zigzag, la máscara y la puntuación.
//
// Todo lo de este fichero está fijado por la especificación y **nada de ello es una
// elección nuestra**. Conviene saberlo antes de «mejorar» una constante: un patrón de
// búsqueda de otro tamaño o una máscara distinta no dan un código peor, dan un código
// que no es un código.

type lienzo struct {
	lado   int
	m      []bool // oscuro
	puesto []bool // si ese módulo ya lo ocupa un patrón o un dato
}

func nuevoLienzo(lado int) *lienzo {
	return &lienzo{lado: lado, m: make([]bool, lado*lado), puesto: make([]bool, lado*lado)}
}

func (l *lienzo) poner(x, y int, oscuro bool) {
	if x < 0 || y < 0 || x >= l.lado || y >= l.lado {
		return
	}
	l.m[y*l.lado+x] = oscuro
	l.puesto[y*l.lado+x] = true
}

func (l *lienzo) libre(x, y int) bool {
	return !l.puesto[y*l.lado+x]
}

func (l *lienzo) oscuro(x, y int) bool {
	return l.m[y*l.lado+x]
}

// dibujar arma el código entero con una máscara concreta.
func dibujar(version int, palabras []byte, mascara int) *Codigo {
	n := lado(version)
	l := nuevoLienzo(n)

	buscadores(l)
	temporizacion(l)
	alineacion(l, version)
	// Las zonas de la información de formato y de versión se reservan antes de meter
	// datos: si no, los datos se colarían en ellas y luego se sobreescribirían.
	reservarFormato(l)
	// **Y el módulo oscuro después de reservar, no antes.** Está en (8, lado-8), justo
	// donde acaba la segunda copia del formato, y reservándolo después se quedaba claro:
	// el código salía sin él y ningún lector lo habría aceptado. Lo cazó la prueba de los
	// patrones fijos, que es para esto.
	l.poner(8, n-8, true)
	if version >= 7 {
		reservarVersion(l)
	}

	datos(l, palabras, mascara)
	formato(l, mascara)
	if version >= 7 {
		infoVersion(l, version)
	}

	return &Codigo{lado: n, modulos: l.m}
}

// buscadores pone los tres cuadrados de las esquinas con su separador.
func buscadores(l *lienzo) {
	esquinas := [][2]int{{0, 0}, {l.lado - 7, 0}, {0, l.lado - 7}}
	for _, e := range esquinas {
		ox, oy := e[0], e[1]
		for y := -1; y <= 7; y++ {
			for x := -1; x <= 7; x++ {
				// El anillo exterior y el cuadrado interior, oscuros; el resto, claro.
				dentro := x >= 0 && x <= 6 && y >= 0 && y <= 6
				borde := dentro && (x == 0 || x == 6 || y == 0 || y == 6)
				centro := x >= 2 && x <= 4 && y >= 2 && y <= 4
				l.poner(ox+x, oy+y, borde || centro)
			}
		}
	}
}

// temporizacion son las dos líneas alternas que dan la escala al lector.
func temporizacion(l *lienzo) {
	for i := 8; i < l.lado-8; i++ {
		oscuro := i%2 == 0
		l.poner(i, 6, oscuro)
		l.poner(6, i, oscuro)
	}
}

// Centros de los patrones de alineación por versión, de la tabla de la especificación.
var centrosAlineacion = [11][]int{
	{},          // sin versión 0
	{},          // la versión 1 no tiene
	{6, 18},     // 2
	{6, 22},     // 3
	{6, 26},     // 4
	{6, 30},     // 5
	{6, 34},     // 6
	{6, 22, 38}, // 7
	{6, 24, 42}, // 8
	{6, 26, 46}, // 9
	{6, 28, 50}, // 10
}

func alineacion(l *lienzo, version int) {
	c := centrosAlineacion[version]
	for _, y := range c {
		for _, x := range c {
			// Las tres esquinas las ocupan los buscadores.
			if (x == 6 && y == 6) || (x == 6 && y == c[len(c)-1]) || (x == c[len(c)-1] && y == 6) {
				continue
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					borde := dx == -2 || dx == 2 || dy == -2 || dy == 2
					centro := dx == 0 && dy == 0
					l.poner(x+dx, y+dy, borde || centro)
				}
			}
		}
	}
}

func reservarFormato(l *lienzo) {
	for i := 0; i <= 8; i++ {
		if i != 6 {
			l.poner(i, 8, false)
			l.poner(8, i, false)
		}
	}
	// **Siete en la columna y ocho en la fila, no ocho y ocho**: el octavo de la columna
	// sería (8, lado-8), que es el módulo oscuro y no parte del formato.
	for i := 0; i < 7; i++ {
		l.poner(8, l.lado-1-i, false)
	}
	for i := 0; i < 8; i++ {
		l.poner(l.lado-1-i, 8, false)
	}
}

func reservarVersion(l *lienzo) {
	for i := 0; i < 6; i++ {
		for j := 0; j < 3; j++ {
			l.poner(i, l.lado-11+j, false)
			l.poner(l.lado-11+j, i, false)
		}
	}
}

// datos recorre la matriz en zigzag de abajo a la derecha hacia arriba, en columnas de
// dos, saltando la columna de temporización, y va poniendo los bits ya enmascarados.
func datos(l *lienzo, palabras []byte, mascara int) {
	bit := 0
	siguiente := func() bool {
		if bit >= len(palabras)*8 {
			// Los códigos admiten unos bits de más a cero al final; no es un error.
			return false
		}
		v := palabras[bit/8]>>(7-uint(bit%8))&1 == 1
		bit++
		return v
	}

	arriba := true
	for col := l.lado - 1; col > 0; col -= 2 {
		if col == 6 {
			// La columna de temporización no cuenta como columna de datos.
			col--
		}
		for i := 0; i < l.lado; i++ {
			y := i
			if arriba {
				y = l.lado - 1 - i
			}
			for _, x := range []int{col, col - 1} {
				if !l.libre(x, y) {
					continue
				}
				v := siguiente()
				if enmascarar(mascara, x, y) {
					v = !v
				}
				l.poner(x, y, v)
			}
		}
		arriba = !arriba
	}
}

// enmascarar dice si ese módulo se invierte, con las ocho fórmulas de la
// especificación. Van escritas tal cual: cualquier «simplificación» cambia el código.
func enmascarar(mascara, x, y int) bool {
	switch mascara {
	case 0:
		return (y+x)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (y+x)%3 == 0
	case 4:
		return (y/2+x/3)%2 == 0
	case 5:
		return (y*x)%2+(y*x)%3 == 0
	case 6:
		return ((y*x)%2+(y*x)%3)%2 == 0
	case 7:
		return ((y+x)%2+(y*x)%3)%2 == 0
	}
	return false
}

// Las quince secuencias de información de formato para el nivel M y cada máscara, ya
// con su BCH y su XOR aplicados. Son constantes de la especificación.
var formatoM = [8]uint{
	0b101010000010010,
	0b101000100100101,
	0b101111001111100,
	0b101101101001011,
	0b100010111111001,
	0b100000011001110,
	0b100111110010111,
	0b100101010100000,
}

// formato escribe la información de formato en sus dos sitios: dos copias a propósito,
// para que el código siga leyéndose si una esquina se estropea.
func formato(l *lienzo, mascara int) {
	bits := formatoM[mascara]
	// **El bit 0 es el menos significativo, y va el primero.** La primera versión de esto
	// colocaba la secuencia al revés —el bit más significativo primero— y el resultado era
	// un código que se dibuja bien, que un lector escrito con la misma idea descodifica sin
	// problema, y que **ningún móvil reconoce**: busca una información de formato válida,
	// no la encuentra y ni llega a mirar los datos. Lo que lo acotó fue un diagnóstico que
	// sacaba los datos probando las ocho máscaras **sin mirar el formato**: el texto salía
	// entero, así que el zigzag, el entrelazado y la corrección estaban bien y solo
	// quedaba esto.
	leer := func(i int) bool { return bits>>uint(i)&1 == 1 }

	// Primera copia: alrededor del buscador de arriba a la izquierda.
	for i := 0; i <= 5; i++ {
		l.poner(8, i, leer(i))
	}
	l.poner(8, 7, leer(6))
	l.poner(8, 8, leer(7))
	l.poner(7, 8, leer(8))
	for i := 9; i <= 14; i++ {
		l.poner(14-i, 8, leer(i))
	}

	// Segunda copia: debajo del de arriba a la derecha y a la derecha del de abajo.
	for i := 0; i <= 7; i++ {
		l.poner(l.lado-1-i, 8, leer(i))
	}
	for i := 8; i <= 14; i++ {
		l.poner(8, l.lado-15+i, leer(i))
	}
}

// Información de versión, de la 7 a la 10. Antes de la 7 no existe.
var versionBits = map[int]uint{
	7:  0b000111110010010100,
	8:  0b001000010110111100,
	9:  0b001001101010011001,
	10: 0b001010010011010011,
}

func infoVersion(l *lienzo, version int) {
	bits := versionBits[version]
	for i := 0; i < 18; i++ {
		v := bits>>uint(i)&1 == 1
		l.poner(i/3, l.lado-11+i%3, v)
		l.poner(l.lado-11+i%3, i/3, v)
	}
}

// ---------------------------------------------------------------- penalización

// penalizacion puntúa un código con las cuatro reglas de la especificación. Gana el de
// menor puntuación, y el objetivo de todas es el mismo: que no haya zonas grandes de un
// color ni dibujos que un lector pueda confundir con un patrón de búsqueda.
func penalizacion(c *Codigo) int {
	return seguidas(c) + bloques(c) + falsosBuscadores(c) + desequilibrio(c)
}

// Regla 1: cinco módulos iguales seguidos valen 3, y cada uno de más suma 1.
func seguidas(c *Codigo) int {
	total := 0
	mirar := func(lee func(i int) bool) {
		n, anterior := 0, false
		for i := 0; i < c.lado; i++ {
			v := lee(i)
			if i > 0 && v == anterior {
				n++
			} else {
				n = 1
			}
			anterior = v
			if n == 5 {
				total += 3
			} else if n > 5 {
				total++
			}
		}
	}
	for y := 0; y < c.lado; y++ {
		fila := y
		mirar(func(x int) bool { return c.Oscuro(x, fila) })
	}
	for x := 0; x < c.lado; x++ {
		col := x
		mirar(func(y int) bool { return c.Oscuro(col, y) })
	}
	return total
}

// Regla 2: cada cuadrado de 2×2 del mismo color vale 3.
func bloques(c *Codigo) int {
	total := 0
	for y := 0; y < c.lado-1; y++ {
		for x := 0; x < c.lado-1; x++ {
			v := c.Oscuro(x, y)
			if c.Oscuro(x+1, y) == v && c.Oscuro(x, y+1) == v && c.Oscuro(x+1, y+1) == v {
				total += 3
			}
		}
	}
	return total
}

// Regla 3: el patrón 1:1:3:1:1 con cuatro claros a un lado vale 40. Es el dibujo del
// buscador, y encontrarlo en medio de los datos es lo que descoloca a un lector.
func falsosBuscadores(c *Codigo) int {
	patron := []bool{true, false, true, true, true, false, true, false, false, false, false}
	alReves := []bool{false, false, false, false, true, false, true, true, true, false, true}
	total := 0
	coincide := func(lee func(i int) bool, desde int, p []bool) bool {
		for i, v := range p {
			if lee(desde+i) != v {
				return false
			}
		}
		return true
	}
	for y := 0; y < c.lado; y++ {
		fila := y
		lee := func(x int) bool { return c.Oscuro(x, fila) }
		for x := 0; x+11 <= c.lado; x++ {
			if coincide(lee, x, patron) || coincide(lee, x, alReves) {
				total += 40
			}
		}
	}
	for x := 0; x < c.lado; x++ {
		col := x
		lee := func(y int) bool { return c.Oscuro(col, y) }
		for y := 0; y+11 <= c.lado; y++ {
			if coincide(lee, y, patron) || coincide(lee, y, alReves) {
				total += 40
			}
		}
	}
	return total
}

// Regla 4: cuánto se desvía del 50% de módulos oscuros, en tramos de un 5%, por 10.
func desequilibrio(c *Codigo) int {
	oscuros := 0
	for _, v := range c.modulos {
		if v {
			oscuros++
		}
	}
	porcentaje := oscuros * 100 / len(c.modulos)
	desvio := porcentaje - 50
	if desvio < 0 {
		desvio = -desvio
	}
	return (desvio / 5) * 10
}
