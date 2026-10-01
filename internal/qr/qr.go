// Package qr dibuja un código QR, y solo el que hace falta aquí.
//
// Existe para una cosa: el código de una red wifi, que son unas decenas de bytes. Así
// que está acotado a propósito y **no es un codificador de QR completo**: modo de
// bytes, nivel de corrección M y versiones 1 a 10. Lo que queda fuera —los modos
// numérico y alfanumérico, el kanji, los niveles L, Q y H, las versiones grandes y el
// modo structured append— no se necesita para esto, y cada pieza de más sería código
// sin usar en el binario que guarda las contraseñas de una empresa.
//
// **Por qué no una dependencia.** Es lo que este proyecto ya decidió dos veces por
// escrito: `internal/codigos` dice que cuarenta líneas de biblioteca estándar valen más
// que un paquete nuevo, y `internal/iconos` que `golang.org/x/image/draw` «sería una
// dependencia nueva de verdad en el binario que guarda las contraseñas de una
// empresa». Aquí pesa además que **el generador recibe la contraseña del wifi en
// claro** para meterla en el dibujo.
//
// **Lo que este paquete no puede comprobar de sí mismo.** En la máquina donde se
// escribió no hay `qrencode`, ni `zbarimg`, ni lector de códigos en el navegador de las
// pruebas, así que no existe una segunda implementación con la que comparar. Lo que hay
// es: vectores fijos, los patrones fijos contrastados con la especificación, la
// puntuación de las ocho máscaras, y un descodificador escrito en el fichero de pruebas
// — que **puede compartir un malentendido con este fichero** y entonces las dos mitades
// se darían la razón. Lo que cierra el asunto es un móvil leyendo el código, igual que
// con los códigos de un solo uso fue tener Dashlane al lado.
package qr

import (
	"errors"
	"fmt"
)

// Nivel de corrección M: recupera un 15% del código dañado. Es el que usan los
// generadores de QR de wifi y el equilibrio habitual entre tamaño y tolerancia a una
// pantalla sucia o a una foto torcida.
const nivelM = 1

// Codigo es un código QR ya resuelto: la matriz de módulos y su lado.
type Codigo struct {
	lado    int
	modulos []bool // lado*lado, true = oscuro
}

// Lado dice cuántos módulos tiene de ancho, **sin el margen blanco**.
//
// El margen —cuatro módulos por lado, la «zona tranquila»— no está en la matriz porque
// no es parte del código: lo pone quien dibuja, y cada medio lo pone a su manera. Sin
// él, muchos lectores no encuentran el código.
func (c *Codigo) Lado() int { return c.lado }

// Oscuro dice si el módulo de esa columna y fila va pintado. Fuera de la matriz,
// devuelve false: así quien dibuje el margen no tiene que comprobar los bordes.
func (c *Codigo) Oscuro(x, y int) bool {
	if x < 0 || y < 0 || x >= c.lado || y >= c.lado {
		return false
	}
	return c.modulos[y*c.lado+x]
}

// Filas devuelve la matriz como una cadena de `0` y `1` por fila.
//
// Es lo que cruza el puente hacia la ventana: una lista de cadenas pesa menos que una
// lista de listas de booleanos en JSON y se lee igual de bien desde TypeScript.
func (c *Codigo) Filas() []string {
	filas := make([]string, c.lado)
	for y := 0; y < c.lado; y++ {
		b := make([]byte, c.lado)
		for x := 0; x < c.lado; x++ {
			b[x] = '0'
			if c.Oscuro(x, y) {
				b[x] = '1'
			}
		}
		filas[y] = string(b)
	}
	return filas
}

// Nuevo codifica un texto en el código más pequeño que lo admita.
func Nuevo(texto string) (*Codigo, error) {
	datos := []byte(texto)
	if len(datos) == 0 {
		return nil, errors.New("Un código vacío no lleva a ninguna parte")
	}
	version, err := versionPara(len(datos))
	if err != nil {
		return nil, err
	}

	bits := cuerpo(datos, version)
	palabras := conCorreccion(bits, version)

	// Las ocho máscaras se prueban todas y gana la de menor penalización, que es lo que
	// manda la especificación y no una elección nuestra: el objetivo es que no se formen
	// manchas ni falsos patrones de búsqueda que confundan al lector.
	var mejor *Codigo
	mejorPuntos := -1
	for mascara := 0; mascara < 8; mascara++ {
		c := dibujar(version, palabras, mascara)
		p := penalizacion(c)
		if mejorPuntos == -1 || p < mejorPuntos {
			mejor, mejorPuntos = c, p
		}
	}
	return mejor, nil
}

// ---------------------------------------------------------------- tamaños

// capacidad en bytes de cada versión con nivel M, de la tabla de la especificación.
// El índice es la versión menos uno.
var capacidadM = [10]int{14, 26, 42, 62, 84, 106, 122, 152, 180, 213}

// palabras de corrección por bloque y número de bloques, nivel M, versiones 1 a 10.
// {palabras de datos totales, palabras de corrección por bloque, bloques del grupo 1,
// palabras de datos por bloque del grupo 1, bloques del grupo 2}
var tablaM = [10]struct {
	datos      int
	correccion int
	bloques1   int
	porBloque1 int
	bloques2   int
}{
	{16, 10, 1, 16, 0},
	{28, 16, 1, 28, 0},
	{44, 26, 1, 44, 0},
	{64, 18, 2, 32, 0},
	{86, 24, 2, 43, 0},
	{108, 16, 4, 27, 0},
	{124, 18, 4, 31, 0},
	{154, 22, 2, 38, 2},
	{182, 22, 3, 36, 2},
	{216, 26, 4, 43, 1},
}

func versionPara(n int) (int, error) {
	for v, cap := range capacidadM {
		if n <= cap {
			return v + 1, nil
		}
	}
	return 0, fmt.Errorf("Eso son %d caracteres y en un código de este tamaño caben %d", n, capacidadM[9])
}

func lado(version int) int { return 17 + 4*version }

// ---------------------------------------------------------------- bits

type flujo struct {
	bits []bool
}

func (f *flujo) añadir(v uint, cuantos int) {
	for i := cuantos - 1; i >= 0; i-- {
		f.bits = append(f.bits, (v>>uint(i))&1 == 1)
	}
}

// cuerpo arma los bits de datos: cabecera de modo, cuenta, los bytes, el terminador y
// el relleno.
func cuerpo(datos []byte, version int) []bool {
	f := &flujo{}
	// Modo de bytes: 0100. Y la cuenta en ocho bits, que es lo que mide el indicador de
	// longitud del modo de bytes **hasta la versión 9**; de la 10 en adelante son
	// dieciséis, y ésa es la única diferencia que este paquete tiene que conocer.
	f.añadir(0b0100, 4)
	if version < 10 {
		f.añadir(uint(len(datos)), 8)
	} else {
		f.añadir(uint(len(datos)), 16)
	}
	for _, b := range datos {
		f.añadir(uint(b), 8)
	}

	capacidadBits := tablaM[version-1].datos * 8
	// Terminador: hasta cuatro ceros, y menos si no caben.
	sobra := capacidadBits - len(f.bits)
	if sobra > 4 {
		sobra = 4
	}
	for i := 0; i < sobra; i++ {
		f.bits = append(f.bits, false)
	}
	// Hasta completar el byte.
	for len(f.bits)%8 != 0 {
		f.bits = append(f.bits, false)
	}
	// Y el relleno, que son dos bytes alternándose. Están fijados por la
	// especificación: no es azar ni ceros.
	relleno := []uint{0b11101100, 0b00010001}
	for i := 0; len(f.bits) < capacidadBits; i++ {
		f.añadir(relleno[i%2], 8)
	}
	return f.bits
}

func aBytes(bits []bool) []byte {
	b := make([]byte, len(bits)/8)
	for i, v := range bits {
		if v {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}

// conCorreccion parte los datos en bloques, calcula la corrección de cada uno y los
// vuelve a entrelazar en el orden que manda la especificación.
//
// **El entrelazado no es un detalle de formato: es lo que hace que un borrón tape un
// trozo de varios bloques en vez de reventar uno entero**, y con él la corrección
// funciona. Al revés —bloque tras bloque— el código se dibuja igual y deja de leerse en
// cuanto se mancha.
func conCorreccion(bits []bool, version int) []byte {
	t := tablaM[version-1]
	datos := aBytes(bits)

	bloques := make([][]byte, 0, t.bloques1+t.bloques2)
	i := 0
	for n := 0; n < t.bloques1; n++ {
		bloques = append(bloques, datos[i:i+t.porBloque1])
		i += t.porBloque1
	}
	for n := 0; n < t.bloques2; n++ {
		bloques = append(bloques, datos[i:i+t.porBloque1+1])
		i += t.porBloque1 + 1
	}

	correcciones := make([][]byte, len(bloques))
	for n, b := range bloques {
		correcciones[n] = reedSolomon(b, t.correccion)
	}

	var salida []byte
	// Primero los datos, columna a columna entre bloques.
	for col := 0; col <= t.porBloque1; col++ {
		for _, b := range bloques {
			if col < len(b) {
				salida = append(salida, b[col])
			}
		}
	}
	// Y después la corrección, igual.
	for col := 0; col < t.correccion; col++ {
		for _, c := range correcciones {
			salida = append(salida, c[col])
		}
	}
	return salida
}

// ---------------------------------------------------------------- Reed-Solomon

// Las tablas de logaritmos de GF(256) con el polinomio 0x11d, que es el que usa QR.
var (
	exp [512]byte
	log [256]byte
)

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		exp[i] = byte(x)
		log[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11d
		}
	}
	// Duplicada para poder sumar logaritmos sin pensar en el módulo.
	for i := 255; i < 512; i++ {
		exp[i] = exp[i-255]
	}
}

func multiplicar(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return exp[int(log[a])+int(log[b])]
}

// generador devuelve el polinomio generador de n palabras de corrección.
func generador(n int) []byte {
	g := []byte{1}
	for i := 0; i < n; i++ {
		// Multiplicar por (x - α^i).
		nuevo := make([]byte, len(g)+1)
		for j, c := range g {
			nuevo[j] ^= c
			nuevo[j+1] ^= multiplicar(c, exp[i])
		}
		g = nuevo
	}
	return g
}

// reedSolomon calcula las n palabras de corrección de un bloque: el resto de dividir
// el bloque (desplazado) por el polinomio generador.
func reedSolomon(datos []byte, n int) []byte {
	g := generador(n)
	resto := make([]byte, len(datos)+n)
	copy(resto, datos)
	for i := 0; i < len(datos); i++ {
		c := resto[i]
		if c == 0 {
			continue
		}
		for j, gc := range g {
			resto[i+j] ^= multiplicar(gc, c)
		}
	}
	return resto[len(datos):]
}
