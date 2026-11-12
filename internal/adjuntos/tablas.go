package adjuntos

import (
	"fmt"
	"strings"
)

// base es la carpeta de descarga de una publicación.
func base(version string) string {
	return "https://github.com/webcafeina/esfinge/releases/download/v" + version
}

// enlace a un adjunto, con su nombre como texto.
func enlace(f Ficha, version string) string {
	n := f.Nombre(version)
	return fmt.Sprintf("[%s](%s/%s)", n, base(version), n)
}

// TablaDelLeeme es la tabla de descargas del README, entre sus dos marcas.
//
// **Sale del catálogo y no de un heredoc.** Hasta el 2026-11-12 eran dos tablas escritas a
// mano en dos ficheros distintos —el generador del README y el cuerpo de la publicación—
// con `amd64` cableado en las dos: o sea la misma lista en dos sitios, que es como se
// desalinea. Y una tabla desalineada no da error: **da un enlace que contesta 404**, y lo
// descubre quien va a descargar.
//
// Solo salen los adjuntos con `Rotulo` y `Publicado`: así una fase que todavía no fabrica
// su fichero no deja una fila con un enlace roto.
func TablaDelLeeme(version string) string { return tablaDelLeeme(Catalogo(), version) }

// tablaDelLeeme recibe **el catálogo por parámetro**, y eso no es un detalle de estilo.
//
// Con la lista leída por dentro, la prueba solo puede comprobar el mundo de hoy — y hoy
// no hay ni un adjunto de arm64 publicado, así que una mutación que los filtrara pasaría
// en verde. Es la lección de `soloLectura`, escrita en el CLAUDE.md: **si una guarda tiene
// un caso que hoy no puede ocurrir, ese caso no lo prueba nadie**, y una prueba que tenga
// que esperar a la fase siguiente para poder fallar no es una prueba, es una esperanza.
func tablaDelLeeme(catalogo []Ficha, version string) string {
	var b strings.Builder
	b.WriteString("<!-- descargas:inicio · las escribe herramientas/actualizar-descargas.sh -->\n\n")
	b.WriteString("| Sistema | Descarga | Notas |\n|---|---|---|\n")
	for _, f := range catalogo {
		if f.Rotulo == "" || !f.Publicado || f.Clase == ClaseMCPB {
			continue
		}
		fmt.Fprintf(&b, "| **%s** | %s | %s |\n", f.Rotulo, enlace(f, version), f.Nota)
	}
	// La línea de comandos no es un fichero: son dieciocho, y lo que se enlaza es la
	// página. Por eso esta fila no sale del catálogo y se escribe aquí.
	b.WriteString("| **Línea de comandos** | " +
		"[todos los sistemas](https://github.com/webcafeina/esfinge/releases/latest) | " +
		"Seis objetivos, con sus SHA256 |\n")

	b.WriteString("\n**Y para usar la bóveda desde un agente de IA** (Claude Desktop y compañía), " +
		"el paquete de su sistema. Se\ninstala arrastrándolo a la ventana del agente, y necesita " +
		"Esfinge abierta:\n\n| | |\n|---|---|\n")
	for _, f := range catalogo {
		if f.Clase != ClaseMCPB || f.Rotulo == "" || !f.Publicado {
			continue
		}
		fmt.Fprintf(&b, "| **%s** | %s |\n", f.Rotulo, enlace(f, version))
	}
	b.WriteString("\nEn **Claude Code** no hace falta: Ajustes da la orden escrita, con la ruta de tu equipo.\n\n")
	fmt.Fprintf(&b, "Versión **%s**. Las anteriores, en "+
		"[publicaciones](https://github.com/webcafeina/esfinge/releases).\n\n", version)
	b.WriteString("<!-- descargas:fin -->\n")
	return b.String()
}

// CuerpoDeLaPublicacion es el texto que acompaña a los adjuntos en GitHub.
//
// Era la **segunda** tabla cableada, en `publicar.yml`. Sale de aquí por lo mismo.
func CuerpoDeLaPublicacion(version string) string {
	return cuerpoDeLaPublicacion(Catalogo(), version)
}

func cuerpoDeLaPublicacion(catalogo []Ficha, version string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Esfinge %s\n\n", version)
	b.WriteString("| Sistema | Descarga |\n|---|---|\n")
	for _, f := range catalogo {
		if f.Rotulo == "" || !f.Publicado || f.Clase == ClaseMCPB {
			continue
		}
		fmt.Fprintf(&b, "| **%s** | `%s` |\n", f.Rotulo, f.Nombre(version))
	}
	b.WriteString("\nLa línea de comandos, para los seis objetivos, en los adjuntos " +
		"`esfinge-" + version + "-<sistema>-<arquitectura>`.\n\n")
	b.WriteString("Y el paquete para agentes de IA:\n\n")
	for _, f := range catalogo {
		if f.Clase != ClaseMCPB || f.Rotulo == "" || !f.Publicado {
			continue
		}
		fmt.Fprintf(&b, "- **%s**: `%s`\n", f.Rotulo, f.Nombre(version))
	}
	b.WriteString("\nTodos los ficheros llevan su SHA256 en `SHA256SUMS`.\n")
	return b.String()
}
