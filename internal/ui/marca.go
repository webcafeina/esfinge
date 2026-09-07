package ui

import "strings"

// Marca es la identidad de Webcafeína en el terminal.
//
// El logotipo real de la casa es una brújula lima y no hay arte ASCII de él en
// ningún sitio: la única cabecera de marca que existe en terminal es la barra ▍
// en lima con el wordmark en minúsculas, de apps/cli/src/ui/branding.ts. Aquí se
// extiende esa idea, con el amarillo de ClickHouse en lugar del lima porque la
// identidad elegida para esta herramienta es la suya.
const (
	Nombre    = "esfinge"
	Wordmark  = "webcafeína"
	Descripcion = "Cifra y descifra secretos con una clave"
)

// letras del banner, de cinco filas. Ancho fijo de cinco columnas por letra, que
// deja el banner en 41 columnas y cabe en un terminal de 80 sin partirse.
var letras = map[rune][5]string{
	'E': {"█████", "█    ", "████ ", "█    ", "█████"},
	'S': {"█████", "█    ", "█████", "    █", "█████"},
	'F': {"█████", "█    ", "████ ", "█    ", "█    "},
	'I': {"█████", "  █  ", "  █  ", "  █  ", "█████"},
	'N': {"█   █", "██  █", "█ █ █", "█  ██", "█   █"},
	'G': {"█████", "█    ", "█  ██", "█   █", "█████"},
}

// Banner devuelve el rótulo de arranque: el nombre en bloques con el acento del
// tema y, debajo, lo que hace el programa.
//
// El wordmark no se repite aquí: ya está en la barra de arriba, y ponerlo dos
// veces en la misma pantalla no es más marca, es ruido.
func Banner(e Estilos) string {
	var filas [5]strings.Builder
	for _, r := range strings.ToUpper(Nombre) {
		bloque, ok := letras[r]
		if !ok {
			continue
		}
		for i := range filas {
			filas[i].WriteString(bloque[i])
			filas[i].WriteByte(' ')
		}
	}

	var b strings.Builder
	for i := range filas {
		b.WriteString(e.Acento.Render(strings.TrimRight(filas[i].String(), " ")))
		b.WriteByte('\n')
	}
	b.WriteString(e.Apagado.Render(Descripcion))
	return b.String()
}

// Firma es la línea de marca: barra de acento, wordmark y descripción. Es la
// versión corta, la que va al pie de los paneles y en las cabeceras estrechas.
func Firma(e Estilos) string {
	return e.Acento.Render(GlifoBarra) + " " +
		e.Titulo.Render(Wordmark) + " " +
		e.Apagado.Render(GlifoInfo+" "+Descripcion)
}

// Pie es la firma con la versión, para el borde inferior de la interfaz.
func Pie(e Estilos, version string) string {
	return e.Acento.Render(GlifoBarra) + " " +
		e.Apagado.Render(Wordmark+" "+GlifoInfo+" "+Nombre+" "+version)
}

// FirmaPlana es la marca sin color ni estilo, para cuando la salida no es un
// terminal pero aun así toca firmar —un fichero cifrado que se guarda con una
// cabecera de cortesía, por ejemplo—.
func FirmaPlana(version string) string {
	return Wordmark + " " + GlifoInfo + " " + Nombre + " " + version
}
