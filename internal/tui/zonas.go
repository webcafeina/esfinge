package tui

import "strings"

// Una zona es un trozo de pantalla que responde al ratón.
type zona struct {
	id   string
	fila int
	col1 int
	col2 int
}

// registro guarda las zonas de la última pantalla dibujada, ya en coordenadas de
// la ventana.
//
// Vive detrás de un puntero a propósito. Bubble Tea llama a View() sobre una
// copia del modelo, así que lo que View apunte en el modelo se pierde; a través
// del puntero, en cambio, las zonas sobreviven hasta el siguiente Update, que es
// justo cuando hace falta consultarlas para saber dónde ha caído un clic.
type registro struct {
	zonas []zona

	// osc guarda un texto pendiente de mandar al portapapeles del terminal por
	// OSC 52. Vive aquí y no en el modelo por el mismo motivo que las zonas: View
	// trabaja sobre una copia, así que necesita un sitio compartido donde marcar
	// que ya lo ha emitido y no repetirlo en cada redibujado.
	osc string

	// oculto dice cuántas líneas de contenido se han quedado fuera por arriba y
	// por abajo, para poder avisar de que hay más.
	arriba, abajo int
}

// lienzo construye un trozo de pantalla línea a línea y va anotando por dónde
// cae cada cosa pulsable. Las filas son locales: al componer la pantalla se
// trasladan a su sitio definitivo.
type lienzo struct {
	lineas []string
	zonas  []zona

	// ancla es la línea que no puede quedarse fuera si hay que desplazar el
	// contenido. Es el campo que tiene el foco: escribir en algo que no se ve
	// sería peor que cualquier recorte.
	ancla int
}

func nuevoLienzo(_ ...*registro) *lienzo { return &lienzo{ancla: -1} }

// escribe añade una línea, que puede traer saltos dentro: un panel de lipgloss
// llega como un bloque de varias líneas y hay que contarlas todas para no
// descuadrar las filas de lo que venga detrás.
func (l *lienzo) escribe(s string) {
	l.lineas = append(l.lineas, strings.Split(s, "\n")...)
}

func (l *lienzo) blanco() { l.lineas = append(l.lineas, "") }

// fila es el número de la próxima línea que se escriba.
func (l *lienzo) fila() int { return len(l.lineas) }

// marca apunta que en la última línea escrita, entre esas dos columnas, hay algo
// pulsable.
func (l *lienzo) marca(id string, col1, col2 int) {
	l.marcaEn(len(l.lineas)-1, id, col1, col2)
}

func (l *lienzo) marcaEn(fila int, id string, col1, col2 int) {
	l.zonas = append(l.zonas, zona{id: id, fila: fila, col1: col1, col2: col2})
}

// marcaFila apunta la línea entera, para las entradas de un menú.
func (l *lienzo) marcaFila(id string) { l.marca(id, 0, 9999) }

func (l *lienzo) alto() int { return len(l.lineas) }

// ventana devuelve el trozo de contenido que se va a ver y desde qué línea
// empieza.
//
// Cuando el contenido no cabe hay que elegir qué parte se enseña, y la respuesta
// es siempre la misma: la que contiene el ancla. Lo que no puede pasar es lo que
// pasaba antes —dibujar más líneas de las que caben y dejar que el terminal las
// desplace por su cuenta—, porque entonces las coordenadas del ratón dejan de
// corresponderse con lo que se ve.
func (l *lienzo) ventana(alto, pedido int, aMano bool) (desde int) {
	if alto <= 0 || l.alto() <= alto {
		return 0
	}

	maximo := l.alto() - alto
	desde = pedido
	if desde > maximo {
		desde = maximo
	}
	if desde < 0 {
		desde = 0
	}

	// El ancla manda sobre el desplazamiento pedido: si el campo con el foco se
	// ha quedado fuera, se trae a la vista. Pero deja de mandar en cuanto alguien
	// mueve la rueda: quien está mirando el final de un formulario no quiere que
	// la pantalla le salte de vuelta al campo enfocado.
	if l.ancla >= 0 && !aMano {
		if l.ancla < desde {
			desde = l.ancla
		}
		// El ancla es la etiqueta del campo, y debajo van las tres líneas del
		// recuadro. Se reservan las cuatro: con menos, el borde de abajo se queda
		// fuera y el campo parece abierto por la base.
		const altoCampo = 4
		if l.ancla+altoCampo > desde+alto {
			desde = l.ancla + altoCampo - alto
		}
		if desde > maximo {
			desde = maximo
		}
		if desde < 0 {
			desde = 0
		}
	}
	return desde
}

// componer pega los trozos en orden, recortando el del medio a lo que quepa, y
// deja el registro con las zonas ya en coordenadas de la ventana.
func componer(reg *registro, alto, desplazado int, aMano bool, arriba, medio, abajo *lienzo) (string, int) {
	reg.zonas = reg.zonas[:0]
	reg.arriba, reg.abajo = 0, 0

	altoMedio := alto - arriba.alto() - abajo.alto()
	if altoMedio < 1 {
		altoMedio = 1
	}

	desde := medio.ventana(altoMedio, desplazado, aMano)
	hasta := desde + altoMedio
	if hasta > medio.alto() {
		hasta = medio.alto()
	}
	reg.arriba = desde
	reg.abajo = medio.alto() - hasta

	var lineas []string
	lineas = append(lineas, arriba.lineas...)
	for _, z := range arriba.zonas {
		reg.zonas = append(reg.zonas, z)
	}

	base := arriba.alto()
	lineas = append(lineas, medio.lineas[desde:hasta]...)
	for _, z := range medio.zonas {
		if z.fila < desde || z.fila >= hasta {
			continue // se ha quedado fuera de la ventana
		}
		z.fila = z.fila - desde + base
		reg.zonas = append(reg.zonas, z)
	}

	// El pie se pega abajo del todo aunque el contenido no llene la pantalla, así
	// los botones y los atajos están siempre en el mismo sitio.
	for len(lineas) < alto-abajo.alto() {
		lineas = append(lineas, "")
	}
	base = len(lineas)
	lineas = append(lineas, abajo.lineas...)
	for _, z := range abajo.zonas {
		z.fila += base
		reg.zonas = append(reg.zonas, z)
	}

	if alto > 0 && len(lineas) > alto {
		lineas = lineas[:alto]
	}
	return strings.Join(lineas, "\n"), desde
}

// en devuelve el identificador de lo que haya en esa posición de la pantalla.
func (r *registro) en(x, y int) string {
	// De atrás hacia delante: si dos zonas se solapan, gana la que se dibujó
	// encima.
	for i := len(r.zonas) - 1; i >= 0; i-- {
		z := r.zonas[i]
		if z.fila == y && x >= z.col1 && x <= z.col2 {
			return z.id
		}
	}
	return ""
}
