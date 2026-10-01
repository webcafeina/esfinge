package qr

import (
	"os"
	"strings"
	"testing"
)

// Las pruebas de este paquete se dividen en dos clases, y la diferencia importa más que
// de costumbre.
//
// **Las independientes** comprueban lo escrito a mano contra algo que no sale de este
// paquete: las constantes de formato y de versión se recalculan con su BCH, las tablas
// de capacidad se comprueban unas contra otras, y la corrección de errores se verifica
// con la propiedad matemática que la define —el polinomio resultante es divisible por el
// generador—. Si una de esas tablas está mal copiada, salta aquí.
//
// **Y la del espejo**: el descodificador del final lee lo que escribe el codificador.
// Ésa **puede compartir un malentendido** —si el zigzag está mal en los dos sitios, pasa
// en verde— y por eso no es la que cierra el asunto. Lo que lo cierra es un móvil
// leyendo el código.

// ------------------------------------------------- independientes

func TestElLadoCrecePorCuatro(t *testing.T) {
	quiere := []int{21, 25, 29, 33, 37, 41, 45, 49, 53, 57}
	for v := 1; v <= 10; v++ {
		if l := lado(v); l != quiere[v-1] {
			t.Errorf("la versión %d mide %d y tiene que medir %d", v, l, quiere[v-1])
		}
	}
}

// Las dos tablas tienen que cuadrar entre sí: las palabras de datos de una versión son
// la suma de sus bloques, y la capacidad en bytes es eso menos la cabecera.
func TestLasTablasCuadranEntreSi(t *testing.T) {
	for v := 1; v <= 10; v++ {
		x := tablaM[v-1]
		suma := x.bloques1*x.porBloque1 + x.bloques2*(x.porBloque1+1)
		if suma != x.datos {
			t.Errorf("versión %d: los bloques suman %d palabras y la tabla dice %d", v, suma, x.datos)
		}
		// La cabecera son 4 bits de modo más el indicador de longitud: 8 bits hasta la
		// versión 9 y 16 desde la 10. O sea 2 bytes de cabecera, o 3 desde la 10.
		cabecera := 2
		if v >= 10 {
			cabecera = 3
		}
		if capacidadM[v-1] != x.datos-cabecera {
			t.Errorf("versión %d: caben %d bytes y la tabla de capacidad dice %d",
				v, x.datos-cabecera, capacidadM[v-1])
		}
	}
}

// bch calcula el resto de meter unos datos por un polinomio generador, que es lo que
// protege la información de formato y la de versión de un módulo mal leído.
func bch(datos uint, bitsDeDatos int, generador uint, bitsDelGenerador int) uint {
	v := datos << uint(bitsDelGenerador-1)
	for i := bitsDeDatos + bitsDelGenerador - 2; i >= bitsDelGenerador-1; i-- {
		if v>>uint(i)&1 == 1 {
			v ^= generador << uint(i-(bitsDelGenerador-1))
		}
	}
	return v
}

// **Las quince secuencias de formato, recalculadas.** Es la prueba que caza una
// constante mal transcrita, que es el fallo más probable de todo el fichero: una sola
// cifra cambiada da un código que ningún lector entiende y el dibujo se ve igual de
// bien.
func TestLaInformacionDeFormatoSaleDelBCH(t *testing.T) {
	// Nivel M son los dos bits `00` delante de los tres de la máscara.
	for mascara := 0; mascara < 8; mascara++ {
		datos := uint(nivelM-1)<<3 | uint(mascara) // nivelM es 1, así que los dos bits van a 0
		resto := bch(datos, 5, 0b10100110111, 11)
		esperado := (datos<<10 | resto) ^ 0b101010000010010
		if formatoM[mascara] != esperado {
			t.Errorf("máscara %d: la tabla dice %015b y el BCH da %015b", mascara, formatoM[mascara], esperado)
		}
	}
}

// Y la de versión, igual: seis bits de versión más doce de BCH, sin XOR.
func TestLaInformacionDeVersionSaleDelBCH(t *testing.T) {
	for v := 7; v <= 10; v++ {
		resto := bch(uint(v), 6, 0b1111100100101, 13)
		esperado := uint(v)<<12 | resto
		if versionBits[v] != esperado {
			t.Errorf("versión %d: la tabla dice %018b y el BCH da %018b", v, versionBits[v], esperado)
		}
	}
}

// **La corrección, comprobada por lo que la define y no por un vector.** Un bloque
// seguido de sus palabras de corrección es, por construcción, divisible por el polinomio
// generador: el resto tiene que ser cero. Eso verifica a la vez las tablas de GF(256), el
// generador y la división, sin que ninguna de las tres tenga que creerse a las otras.
func TestLaCorreccionEsDivisiblePorElGenerador(t *testing.T) {
	for _, n := range []int{10, 16, 18, 22, 24, 26} {
		datos := make([]byte, 20)
		for i := range datos {
			datos[i] = byte(i*7 + 1)
		}
		c := reedSolomon(datos, n)
		if len(c) != n {
			t.Fatalf("pedí %d palabras de corrección y salieron %d", n, len(c))
		}
		// Dividir datos||corrección por el generador: el resto tiene que ser todo ceros.
		junto := append(append([]byte{}, datos...), c...)
		g := generador(n)
		for i := 0; i < len(datos); i++ {
			coef := junto[i]
			if coef == 0 {
				continue
			}
			for j, gc := range g {
				junto[i+j] ^= multiplicar(gc, coef)
			}
		}
		for i, v := range junto[len(datos):] {
			if v != 0 {
				t.Fatalf("con %d palabras, el resto no es cero en la posición %d: %d", n, i, v)
			}
		}
	}
}

// Las tablas de logaritmos, por su propia definición.
func TestElCuerpoFinitoEsCoherente(t *testing.T) {
	for i := 1; i < 256; i++ {
		if exp[log[byte(i)]] != byte(i) {
			t.Fatalf("exp y log no se deshacen en %d", i)
		}
	}
	// Y la multiplicación, conmutativa y con su neutro.
	for _, a := range []byte{1, 2, 3, 100, 255} {
		for _, b := range []byte{1, 5, 17, 200} {
			if multiplicar(a, b) != multiplicar(b, a) {
				t.Errorf("multiplicar no es conmutativa en %d y %d", a, b)
			}
		}
		if multiplicar(a, 1) != a {
			t.Errorf("multiplicar por uno cambia %d", a)
		}
		if multiplicar(a, 0) != 0 {
			t.Errorf("multiplicar por cero no da cero en %d", a)
		}
	}
}

// Los patrones fijos, contra lo que dice la especificación y no contra lo que escribimos.
func TestLosPatronesFijos(t *testing.T) {
	c, err := Nuevo("WIFI:T:WPA;S:Casa;P:secreta;;")
	if err != nil {
		t.Fatal(err)
	}
	n := c.Lado()

	// Los tres buscadores: anillo oscuro de 7×7 con el centro de 3×3 oscuro y el aro
	// intermedio claro.
	for _, e := range [][2]int{{0, 0}, {n - 7, 0}, {0, n - 7}} {
		for y := 0; y < 7; y++ {
			for x := 0; x < 7; x++ {
				borde := x == 0 || x == 6 || y == 0 || y == 6
				centro := x >= 2 && x <= 4 && y >= 2 && y <= 4
				if c.Oscuro(e[0]+x, e[1]+y) != (borde || centro) {
					t.Fatalf("el buscador de (%d,%d) está mal en (%d,%d)", e[0], e[1], x, y)
				}
			}
		}
	}

	// El separador: la fila y la columna que rodean al buscador, claras.
	for i := 0; i <= 7; i++ {
		if c.Oscuro(i, 7) || c.Oscuro(7, i) {
			t.Fatalf("el separador del primer buscador no está claro en %d", i)
		}
	}

	// La temporización, alterna y empezando en oscuro.
	for i := 8; i < n-8; i++ {
		if c.Oscuro(i, 6) != (i%2 == 0) || c.Oscuro(6, i) != (i%2 == 0) {
			t.Fatalf("la temporización está mal en %d", i)
		}
	}

	// Y el módulo oscuro de siempre.
	if !c.Oscuro(8, n-8) {
		t.Error("falta el módulo oscuro de (8, lado-8)")
	}
}

func TestLaMascaraElegidaEsLaDeMenorPuntuacion(t *testing.T) {
	texto := "WIFI:T:WPA;S:WEBCAFEINA;P:una-clave-larga-de-prueba;;"
	c, err := Nuevo(texto)
	if err != nil {
		t.Fatal(err)
	}
	v, err := versionPara(len(texto))
	if err != nil {
		t.Fatal(err)
	}
	palabras := conCorreccion(cuerpo([]byte(texto), v), v)
	mejor := -1
	for m := 0; m < 8; m++ {
		if p := penalizacion(dibujar(v, palabras, m)); mejor == -1 || p < mejor {
			mejor = p
		}
	}
	if p := penalizacion(c); p != mejor {
		t.Errorf("el código elegido puntúa %d y la mejor máscara puntúa %d", p, mejor)
	}
}

func TestLasCuatroReglasDePenalizacion(t *testing.T) {
	// Un código entero de un color: las cuatro reglas tienen que protestar.
	todoOscuro := &Codigo{lado: 21, modulos: make([]bool, 21*21)}
	for i := range todoOscuro.modulos {
		todoOscuro.modulos[i] = true
	}
	if seguidas(todoOscuro) == 0 {
		t.Error("una fila entera del mismo color no penaliza")
	}
	if bloques(todoOscuro) == 0 {
		t.Error("un cuadrado entero del mismo color no penaliza")
	}
	if desequilibrio(todoOscuro) == 0 {
		t.Error("el 100% de módulos oscuros no penaliza")
	}
	// Y el patrón del buscador en medio de los datos.
	conFalso := &Codigo{lado: 21, modulos: make([]bool, 21*21)}
	patron := []bool{true, false, true, true, true, false, true, false, false, false, false}
	for x, v := range patron {
		conFalso.modulos[10*21+x] = v
	}
	if falsosBuscadores(conFalso) == 0 {
		t.Error("un patrón de buscador falso no penaliza")
	}
}

func TestLoQueNoCabeLoDice(t *testing.T) {
	if _, err := Nuevo(""); err == nil {
		t.Error("un texto vacío tendría que dar error")
	}
	if _, err := Nuevo(strings.Repeat("x", 214)); err == nil {
		t.Error("214 caracteres no caben en la versión 10 y tendría que decirlo")
	}
	if _, err := Nuevo(strings.Repeat("x", 213)); err != nil {
		t.Errorf("213 caracteres sí caben: %v", err)
	}
}

func TestLaVersionCreceConElTexto(t *testing.T) {
	// Catorce caracteres son los que caben en la versión 1 con nivel M; «WIFI:T:nopass;S:x;;»
	// son diecinueve y ya pide la 2, que es lo que esta prueba daba por hecho al revés.
	corto, err := Nuevo("WIFI:S:x;;")
	if err != nil {
		t.Fatal(err)
	}
	largo, err := Nuevo("WIFI:T:WPA;S:" + strings.Repeat("a", 80) + ";P:" + strings.Repeat("b", 60) + ";;")
	if err != nil {
		t.Fatal(err)
	}
	if corto.Lado() >= largo.Lado() {
		t.Errorf("el código corto mide %d y el largo %d", corto.Lado(), largo.Lado())
	}
	if corto.Lado() != 21 {
		t.Errorf("un texto de diez caracteres cabe en la versión 1 y mide %d", corto.Lado())
	}
}

func TestFilasDibujaLaMatriz(t *testing.T) {
	c, err := Nuevo("hola")
	if err != nil {
		t.Fatal(err)
	}
	filas := c.Filas()
	if len(filas) != c.Lado() {
		t.Fatalf("salen %d filas y el lado es %d", len(filas), c.Lado())
	}
	for y, f := range filas {
		if len(f) != c.Lado() {
			t.Fatalf("la fila %d mide %d", y, len(f))
		}
		for x, r := range f {
			if (r == '1') != c.Oscuro(x, y) {
				t.Fatalf("la fila %d no cuadra con la matriz en %d", y, x)
			}
		}
	}
}

// ------------------------------------------------- la del espejo

// leer descodifica un código de este paquete: encuentra la máscara en la información de
// formato, la deshace, recorre el zigzag y saca los bytes.
//
// **No corrige errores y no le hace falta**: la matriz viene de aquí y está intacta. Lo
// que comprueba es que lo escrito se puede volver a leer, y **lo que no puede comprobar
// es el orden del zigzag ni el modo de bytes**, porque de eso tiene la misma idea que el
// codificador. Si esa idea está mal, estas pruebas pasan y el móvil no lee nada.
func leer(t *testing.T, c *Codigo) string {
	t.Helper()
	// La máscara, de la primera copia de la información de formato.
	var bits uint
	// El bit 0 va el primero, igual que al escribirlo.
	cuantos := 0
	leerBit := func(x, y int) {
		if c.Oscuro(x, y) {
			bits |= 1 << uint(cuantos)
		}
		cuantos++
	}
	for i := 0; i <= 5; i++ {
		leerBit(8, i)
	}
	leerBit(8, 7)
	leerBit(8, 8)
	leerBit(7, 8)
	for i := 9; i <= 14; i++ {
		leerBit(14-i, 8)
	}
	mascara := -1
	for m, v := range formatoM {
		if v == bits {
			mascara = m
		}
	}
	if mascara == -1 {
		t.Fatalf("la información de formato no es ninguna de las conocidas: %015b", bits)
	}

	// La versión, por el lado.
	version := (c.Lado() - 17) / 4

	// El mismo recorrido que al escribir, pero marcando los módulos que ocupan los
	// patrones con un lienzo sin datos.
	l := nuevoLienzo(c.Lado())
	buscadores(l)
	temporizacion(l)
	alineacion(l, version)
	l.poner(8, c.Lado()-8, true)
	reservarFormato(l)
	if version >= 7 {
		reservarVersion(l)
	}

	var datosBits []bool
	arriba := true
	for col := c.Lado() - 1; col > 0; col -= 2 {
		if col == 6 {
			col--
		}
		for i := 0; i < c.Lado(); i++ {
			y := i
			if arriba {
				y = c.Lado() - 1 - i
			}
			for _, x := range []int{col, col - 1} {
				if !l.libre(x, y) {
					continue
				}
				v := c.Oscuro(x, y)
				if enmascarar(mascara, x, y) {
					v = !v
				}
				datosBits = append(datosBits, v)
			}
		}
		arriba = !arriba
	}

	palabras := aBytes(datosBits[:len(datosBits)/8*8])

	// Deshacer el entrelazado de bloques para recuperar los datos en orden.
	tb := tablaM[version-1]
	nBloques := tb.bloques1 + tb.bloques2
	largos := make([]int, nBloques)
	for i := 0; i < tb.bloques1; i++ {
		largos[i] = tb.porBloque1
	}
	for i := tb.bloques1; i < nBloques; i++ {
		largos[i] = tb.porBloque1 + 1
	}
	bloques := make([][]byte, nBloques)
	p := 0
	for col := 0; col <= tb.porBloque1; col++ {
		for b := 0; b < nBloques; b++ {
			if col < largos[b] {
				bloques[b] = append(bloques[b], palabras[p])
				p++
			}
		}
	}
	var datos []byte
	for _, b := range bloques {
		datos = append(datos, b...)
	}

	// Y la cabecera: modo de bytes, la cuenta y el texto.
	f := &flujo{}
	for _, b := range datos {
		f.añadir(uint(b), 8)
	}
	tomar := func(desde, cuantos int) uint {
		var v uint
		for i := 0; i < cuantos; i++ {
			v <<= 1
			if f.bits[desde+i] {
				v |= 1
			}
		}
		return v
	}
	if modo := tomar(0, 4); modo != 0b0100 {
		t.Fatalf("el modo no es el de bytes: %04b", modo)
	}
	bitsDeCuenta := 8
	if version >= 10 {
		bitsDeCuenta = 16
	}
	n := int(tomar(4, bitsDeCuenta))
	salida := make([]byte, n)
	for i := 0; i < n; i++ {
		salida[i] = byte(tomar(4+bitsDeCuenta+i*8, 8))
	}
	return string(salida)
}

func TestLoQueSeEscribeSePuedeLeer(t *testing.T) {
	casos := []string{
		"hola",
		"WIFI:T:nopass;S:Invitados;;",
		"WIFI:T:WPA;S:WEBCAFEINA;P:una-clave;;",
		`WIFI:T:WPA;S:a\;b\,c;P:p\\q;H:true;;`,
		"WIFI:T:WPA;S:Café de la Esquina;P:ñandú-con-tildes;;",
		// Uno de cada tamaño que nos importa, para cruzar todas las tablas de bloques.
		strings.Repeat("a", 14),
		strings.Repeat("b", 26),
		strings.Repeat("c", 62),
		strings.Repeat("d", 106),
		strings.Repeat("e", 152),
		strings.Repeat("f", 180),
		strings.Repeat("g", 213),
	}
	for _, texto := range casos {
		c, err := Nuevo(texto)
		if err != nil {
			t.Fatalf("%q: %v", texto, err)
		}
		if vuelta := leer(t, c); vuelta != texto {
			t.Errorf("escribí %q y leo %q (lado %d)", texto, vuelta, c.Lado())
		}
	}
}

// ------------------------------------------------- los vectores

// **El vector que leyó un móvil de verdad**, grabado el 2026-10-01.
//
// Es lo mismo que hacen los vectores del formato `ESF1` y por la misma razón: aquí no
// hay con qué comprobar que un QR es correcto —ni `qrencode`, ni `zbarimg`, ni lector en
// el navegador de las pruebas— así que lo que tiene valor es **congelar el que se sabe
// que funciona**. Este contenido exacto se enseñó en la pantalla, el cliente lo escaneó
// con su móvil y le ofreció unirse a «Esfinge de prueba».
//
// Si esta prueba se pone roja, el código ha cambiado y **lo que un móvil leía ya no se
// sabe si se lee**. No se regenera el fichero para que vuelva a pasar: o el cambio está
// mal, o hay que volver a escanearlo y grabarlo de nuevo. El programa que lo generó no se
// queda, igual que con los vectores del contenedor.
//
// Lo que este vector **no** dice: que cualquier otro texto salga bien. Para eso están las
// pruebas de arriba, que comprueban las tablas contra sus propias definiciones.
func TestElVectorQueLeyoUnMovil(t *testing.T) {
	bruto, err := os.ReadFile("testdata/vectores.txt")
	if err != nil {
		t.Fatal(err)
	}
	bloques := strings.Split(strings.TrimSpace(string(bruto)), "\n\n")
	if len(bloques) == 0 {
		t.Fatal("no hay vectores")
	}
	for _, b := range bloques {
		lineas := strings.Split(strings.TrimSpace(b), "\n")
		texto, quiere := lineas[0], lineas[1:]
		c, err := Nuevo(texto)
		if err != nil {
			t.Fatalf("%q: %v", texto, err)
		}
		salen := c.Filas()
		if len(salen) != len(quiere) {
			t.Fatalf("%q: salen %d filas y el vector tiene %d", texto, len(salen), len(quiere))
		}
		for i := range quiere {
			if salen[i] != quiere[i] {
				t.Fatalf("%q, fila %d:\nsale   %s\nquiere %s", texto, i, salen[i], quiere[i])
			}
		}
	}
}

// **Y los datos se leen sin mirar la información de formato.**
//
// Es la prueba que acotó el único fallo que tuvo este paquete: el código se dibujaba
// bien, el descodificador de aquí lo leía y ningún móvil lo reconocía. Sacando los datos
// con las ocho máscaras **a ciegas** se vio que el texto salía entero, y con eso el fallo
// quedó encerrado en los quince bits del formato — que estaban al revés.
//
// Se queda porque separa dos cosas que de otro modo fallan juntas: si algún día esto pasa
// y `TestElVectorQueLeyoUnMovil` falla, el problema es el formato; si falla esto, es el
// zigzag, el entrelazado o la corrección.
func TestLosDatosSeLeenSinMirarElFormato(t *testing.T) {
	texto := "WIFI:T:WPA;S:Esfinge de prueba;P:clave-de-prueba-1234;;"
	c, err := Nuevo(texto)
	if err != nil {
		t.Fatal(err)
	}
	version := (c.Lado() - 17) / 4

	encontrado := ""
	for mascara := 0; mascara < 8; mascara++ {
		l := nuevoLienzo(c.Lado())
		buscadores(l)
		temporizacion(l)
		alineacion(l, version)
		reservarFormato(l)
		if version >= 7 {
			reservarVersion(l)
		}
		l.poner(8, c.Lado()-8, true)

		var bits []bool
		arriba := true
		for col := c.Lado() - 1; col > 0; col -= 2 {
			if col == 6 {
				col--
			}
			for i := 0; i < c.Lado(); i++ {
				y := i
				if arriba {
					y = c.Lado() - 1 - i
				}
				for _, x := range []int{col, col - 1} {
					if !l.libre(x, y) {
						continue
					}
					v := c.Oscuro(x, y)
					if enmascarar(mascara, x, y) {
						v = !v
					}
					bits = append(bits, v)
				}
			}
			arriba = !arriba
		}
		palabras := aBytes(bits[:len(bits)/8*8])
		tb := tablaM[version-1]
		nBloques := tb.bloques1 + tb.bloques2
		largos := make([]int, nBloques)
		for i := 0; i < tb.bloques1; i++ {
			largos[i] = tb.porBloque1
		}
		for i := tb.bloques1; i < nBloques; i++ {
			largos[i] = tb.porBloque1 + 1
		}
		bl := make([][]byte, nBloques)
		p := 0
		for col := 0; col <= tb.porBloque1; col++ {
			for b := 0; b < nBloques; b++ {
				if col < largos[b] && p < len(palabras) {
					bl[b] = append(bl[b], palabras[p])
					p++
				}
			}
		}
		var datos []byte
		for _, b := range bl {
			datos = append(datos, b...)
		}
		if len(datos) < 2 || datos[0]>>4 != 0b0100 {
			continue
		}
		largo := int(datos[0]&0x0f)<<4 | int(datos[1]>>4)
		if largo != len(texto) || len(datos) < 2+largo {
			continue
		}
		salida := make([]byte, largo)
		for i := 0; i < largo; i++ {
			salida[i] = datos[1+i]<<4 | datos[2+i]>>4
		}
		encontrado = string(salida)
	}
	if encontrado != texto {
		t.Errorf("probando las ocho máscaras a ciegas sale %q y tenía que salir %q", encontrado, texto)
	}
}

// **Y el bit 0 del formato va en (8,0), no el 14.** Es la convención que un lector espera
// y la que estuvo invertida: se escribe aquí como prueba para que invertirla otra vez no
// dependa de que alguien se acuerde.
func TestElFormatoEmpiezaPorElBitMenosSignificativo(t *testing.T) {
	c, err := Nuevo("hola")
	if err != nil {
		t.Fatal(err)
	}
	// Se lee la secuencia entera **en el orden documentado** —el bit 0 en (8,0), subiendo
	// hasta el bit 14 en (0,8)— y tiene que ser una de las quince conocidas. Mirando dos
	// bits sueltos no vale: coinciden varias máscaras a la vez, que fue lo que falló en la
	// primera versión de esta prueba.
	var leido uint
	cuantos := 0
	tomar := func(x, y int) {
		if c.Oscuro(x, y) {
			leido |= 1 << uint(cuantos)
		}
		cuantos++
	}
	for i := 0; i <= 5; i++ {
		tomar(8, i)
	}
	tomar(8, 7)
	tomar(8, 8)
	tomar(7, 8)
	for i := 9; i <= 14; i++ {
		tomar(14-i, 8)
	}
	cual := -1
	for m, v := range formatoM {
		if v == leido {
			cual = m
		}
	}
	if cual == -1 {
		t.Fatalf("lo que hay escrito en la zona de formato no es ninguna secuencia válida: %015b", leido)
	}
	// Y esa máscara tiene que ser la que de verdad se usó: se comprueba volviendo a
	// dibujar con ella y exigiendo la misma matriz.
	v, err := versionPara(4)
	if err != nil {
		t.Fatal(err)
	}
	igual := dibujar(v, conCorreccion(cuerpo([]byte("hola"), v), v), cual)
	for y := 0; y < c.Lado(); y++ {
		for x := 0; x < c.Lado(); x++ {
			if c.Oscuro(x, y) != igual.Oscuro(x, y) {
				t.Fatalf("el formato dice la máscara %d y la matriz no es la de esa máscara (%d,%d)", cual, x, y)
			}
		}
	}
}
