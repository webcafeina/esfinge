package tui

import (
	"strings"

	"github.com/webcafeina/esfinge/internal/ui"
)

// Medidas del diagrama. Están aquí arriba y no repartidas por el código porque
// las líneas tienen que cuadrar columna a columna: basta con que una caja mida
// un carácter de más para que la flecha de vuelta apunte a cualquier sitio.
const (
	anchoIzq = 14 // interior de la caja de la izquierda
	anchoDer = 18 // interior de la caja de la derecha
	hueco    = 11 // separación entre las dos cajas
)

// verAyuda explica lo que hace Esfinge con un dibujo en vez de con un párrafo.
//
// Antes era un bloque de texto dentro del panel de resultados, y un bloque de
// texto sobre criptografía no lo lee nadie. Lo que hay que entender son dos
// cosas —que la clave es lo único que abre lo cifrado, y que esa clave no está
// guardada en ningún sitio—, y las dos se ven de un vistazo en el diagrama.
func (m modelo) verAyuda(cabeza, l, pie *lienzo) {
	e := m.e

	cabeza.escribe(sangria + e.Migas("Menú", "Ayuda"))
	cabeza.blanco()

	// relleno devuelve los espacios que faltan para completar el interior de una
	// caja. Se cuenta en runas y no en bytes: «…» ocupa tres bytes y una columna,
	// y contando bytes la caja saldría torcida justo en la línea del ejemplo.
	relleno := func(texto string, ancho int) string {
		n := ancho - 1 - len([]rune(texto))
		if n < 0 {
			n = 0
		}
		return strings.Repeat(" ", n)
	}
	techo := func(ancho int) string {
		return "┌" + strings.Repeat("─", ancho) + "┐"
	}
	suelo := func(ancho int) string {
		return "└" + strings.Repeat("─", ancho) + "┘"
	}

	blancoHueco := strings.Repeat(" ", hueco)
	flecha := "  " + "──────►" + "  " // los mismos once caracteres del hueco

	l.escribe(sangria + e.Apagado.Render("Lo tuyo") +
		strings.Repeat(" ", anchoIzq+2+hueco-len("Lo tuyo")) +
		e.Apagado.Render("Lo que puedes mandar"))

	l.escribe(sangria + e.Filete.Render(techo(anchoIzq)+blancoHueco+techo(anchoDer)))

	fila := func(izquierda, derecha, medio string) string {
		return sangria +
			e.Filete.Render("│ ") + e.Cuerpo.Render(izquierda) + relleno(izquierda, anchoIzq) + e.Filete.Render("│") +
			medio +
			e.Filete.Render("│ ") + e.Codigo.Render(derecha) + relleno(derecha, anchoDer) + e.Filete.Render("│")
	}
	l.escribe(fila("hunter2", "ESF1.RVNGMQEA…", e.Acento.Render(flecha)))
	l.escribe(fila("un .env", "un.env.esf", blancoHueco))

	l.escribe(sangria + e.Filete.Render(suelo(anchoIzq)+blancoHueco+suelo(anchoDer)))

	// La flecha de vuelta sale del centro de una caja y entra en el centro de la
	// otra. Es lo que cuenta que la clave que cierra es la misma que abre.
	centroIzq := 1 + anchoIzq/2
	centroDer := anchoIzq + 2 + hueco + 1 + anchoDer/2
	entre := centroDer - centroIzq - 1

	l.escribe(sangria + strings.Repeat(" ", centroIzq) + e.Filete.Render("▲") +
		strings.Repeat(" ", entre) + e.Filete.Render("│"))

	etiqueta := " La misma clave "
	sobra := entre - len([]rune(etiqueta))
	izq := sobra / 2
	l.escribe(sangria + strings.Repeat(" ", centroIzq) +
		e.Filete.Render("└"+strings.Repeat("─", izq)) +
		e.Acento.Render(etiqueta) +
		e.Filete.Render(strings.Repeat("─", sobra-izq)+"┘"))

	l.blanco()
	l.escribe(sangria + e.Ojo("La clave no se guarda en ninguna parte."))
	l.escribe(sangria + "  " + e.Apagado.Render("Si se pierde, se pierde el contenido: no hay «he olvidado mi contraseña»."))

	l.blanco()
	l.escribe(sangria + e.Etiq("Cómo se usa"))
	l.blanco()
	{
		for _, p := range [][2]string{
			{"1", "Elige Cifrar y escribe el secreto, o pulsa Fichero y arrastra uno"},
			{"2", "Pon una clave larga: cuatro palabras sin relación van bien"},
			{"3", "El resultado se copia solo; también puedes guardarlo en un fichero"},
			{"4", "Manda el resultado y la clave por caminos distintos"},
		} {
			l.escribe(sangria + e.Chip(p[0]) + " " + e.Cuerpo.Render(p[1]))
		}
	}

	m.botonesEn(pie, []boton{{zonaVolver, "Volver", true}})
}

// El texto de la ayuda para quien la quiere en un fichero, con la tecla de
// guardar. Es el mismo contenido del dibujo, escrito de corrido.
var textoAyuda = ui.Nombre + " · " + ui.Descripcion + `

Esfinge cifra un secreto con una clave. Quien tenga el texto cifrado y la
clave lo recupera; quien tenga solo el texto, no.

La clave no se guarda en ninguna parte. Si se pierde, se pierde el
contenido: no es una cuenta con «he olvidado mi contraseña».

Cómo se usa:

  1  Elige Cifrar y escribe el secreto, o pulsa Fichero y arrastra uno.
  2  Pon una clave larga: cuatro palabras sin relación van bien.
  3  El resultado se copia solo; también puedes guardarlo en un fichero.
  4  Manda el resultado y la clave por caminos distintos.

Desde la terminal, sin abrir los menús:

  esfinge cifrar -i credenciales.env -o credenciales.env.esf
  esfinge descifrar -i credenciales.env.esf
  esfinge generar --bytes 32
  esfinge --help

El cifrado es XChaCha20-Poly1305 y la clave se deriva con Argon2id.
`
