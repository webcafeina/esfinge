// Rellena una plantilla de empaquetado con la versión, los nombres de los adjuntos y sus sumas.
//
//	rellenar <versión> <plantilla> <SHA256SUMS>
//
// Escribe el resultado por la salida estándar.
//
// # Qué sustituye
//
//	{{VERSION}}            la versión, sin la «v»
//	{{NOMBRE:deb-arm64}}   el nombre del adjunto de esa clave, según el catálogo
//	{{SHA:deb-arm64}}      su SHA-256, leído del SHA256SUMS de la publicación
//
// # Por qué los marcadores nombran la clave del catálogo
//
// Apolo hace esto mismo con marcadores escritos a mano —`{{SHA_DEB_AMD64}}`— y una tabla en su
// guion que dice qué fichero es cada uno. Eso es **una segunda lista de nombres**, y en esta casa
// una regla en dos listas ya ha costado seis fallos. Aquí el marcador **es** la clave del catálogo
// (`internal/adjuntos`), así que no hay nada que mantener al lado: si el nombre de un fichero
// cambia, cambia en un solo sitio y las tres recetas lo siguen.
//
// # Y si queda algún marcador sin sustituir, esto falla
//
// No es una cortesía. Un `{{SHA:deb-arm64}}` mal escrito que se cuela en un PKGBUILD **se publica
// en el AUR**, y entonces `makepkg` se cae en el equipo de quien instala con un error de sintaxis
// que no dice nada de nosotros. Es la familia de «un aviso que no detiene nada deja de leerse»: lo
// que hay que respetar tiene que parar algo.
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/webcafeina/esfinge/internal/adjuntos"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Uso: rellenar <versión> <plantilla> <SHA256SUMS>")
		os.Exit(2)
	}
	version, plantilla, sumas := os.Args[1], os.Args[2], os.Args[3]

	crudo, err := os.ReadFile(plantilla)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✕", err)
		os.Exit(1)
	}
	porFichero, err := leerSumas(sumas)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✕", err)
		os.Exit(1)
	}

	salida, problemas := Rellenar(string(crudo), version, porFichero)
	if len(problemas) > 0 {
		fmt.Fprintf(os.Stderr, "✕ La plantilla %s no se ha podido rellenar:\n", plantilla)
		for _, p := range problemas {
			fmt.Fprintln(os.Stderr, "   ", p)
		}
		os.Exit(1)
	}
	fmt.Print(salida)
}

// elMarcador casa cualquier `{{…}}`, también los que no se reconocen: lo que no se sustituye hay
// que poder nombrarlo en el error.
var elMarcador = regexp.MustCompile(`\{\{([A-Za-z]+)(?::([a-z0-9-]+))?\}\}`)

// Rellenar sustituye los marcadores y devuelve lo que no ha podido.
//
// `porFichero` son las sumas por nombre de fichero, que es como las escribe `sha256sum`.
func Rellenar(plantilla, version string, porFichero map[string]string) (string, []string) {
	var problemas []string
	salida := elMarcador.ReplaceAllStringFunc(plantilla, func(m string) string {
		partes := elMarcador.FindStringSubmatch(m)
		que, clave := partes[1], partes[2]
		switch que {
		case "VERSION":
			if clave != "" {
				problemas = append(problemas, m+": «VERSION» no lleva clave")
				return m
			}
			return version
		case "NOMBRE", "SHA":
			if clave == "" {
				problemas = append(problemas, m+": falta la clave del catálogo, como {{"+que+":deb-arm64}}")
				return m
			}
			f, hay := adjuntos.Por(clave)
			if !hay {
				problemas = append(problemas, m+": el catálogo no conoce «"+clave+"»")
				return m
			}
			nombre := f.Nombre(version)
			if !f.Publicado {
				problemas = append(problemas, m+": «"+clave+"» todavía no se publica, así que no hay fichero que nombrar")
				return m
			}
			if que == "NOMBRE" {
				return nombre
			}
			suma, hay := porFichero[nombre]
			if !hay {
				problemas = append(problemas, m+": «"+nombre+"» no está en el SHA256SUMS de la publicación")
				return m
			}
			return suma
		default:
			problemas = append(problemas, m+": no sé qué es «"+que+"»")
			return m
		}
	})
	// Y lo que no casa ni el patrón: unas llaves mal cerradas no se sustituyen y tampoco se ven.
	if i := strings.Index(salida, "{{"); i >= 0 {
		hasta := i + 40
		if hasta > len(salida) {
			hasta = len(salida)
		}
		problemas = append(problemas, "queda un marcador sin cerrar o mal escrito: "+salida[i:hasta])
	}
	return salida, problemas
}

// leerSumas lee un fichero al estilo de coreutils: «<suma>  <nombre>», con el «*» opcional que
// marca lo leído en binario.
func leerSumas(ruta string) (map[string]string, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		campos := strings.Fields(s.Text())
		if len(campos) != 2 {
			continue
		}
		out[strings.TrimPrefix(campos[1], "*")] = campos[0]
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s no tiene ninguna línea con la forma «<suma>  <nombre>»", ruta)
	}
	return out, nil
}
