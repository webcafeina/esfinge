// Package rutas dice dónde vive cada bóveda en este equipo.
//
// Es un paquete de cuatro funciones y existe por una razón: **desde la ADR 0057
// esas rutas las leen dos programas**, la ventana y la línea de comandos, y el
// mismo dato escrito en dos sitios se desalinea solo. Aquí el fallo sería mudo y de
// los peores: dos sitios donde buscar una bóveda compartida, cada programa
// trabajando sobre su copia y la sincronización peleándose con ella.
//
// Antes vivía en `internal/app`, y la línea de comandos **repetía la ruta de la
// bóveda principal a propósito** para no depender del paquete de la ventana —que
// arrastra Wails y no cruza de plataforma—. Esa razón sigue en pie; lo que cambia
// es que ahora hay un sitio al que los dos pueden preguntar.
//
// Lo que este paquete **no** hace: no crea carpetas, no lee ficheros y no sabe qué
// hay dentro. Son rutas.
package rutas

import (
	"os"
	"path/filepath"
)

// Los nombres de las dos subcarpetas. **Una bóveda de proyecto se llama por una
// referencia al azar** (ADR 0050): un listado de la carpeta no puede ser la lista de
// clientes.
const (
	carpetaDeProyectos   = "proyectos"
	carpetaDeCompartidas = "compartidas"
)

// DeEsfinge es la carpeta de configuración de Esfinge en este equipo. Vacío si el
// sistema no sabe decir cuál es.
func DeEsfinge() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge")
}

// BovedaPrincipal es la bóveda personal: la que abre todas las demás (ADR 0050).
func BovedaPrincipal() string { return en("boveda.esfinge") }

// Proyecto es el fichero de una bóveda de proyecto, por su referencia.
func Proyecto(ref string) string { return en(carpetaDeProyectos, ref+".esfinge") }

// Compartida es el fichero de una bóveda de otra persona a la que me han dado
// acceso (ADR 0052). Lleva dentro de quién es, porque dos personas pueden tener
// bóvedas con la misma referencia.
func Compartida(dueno, ref string) string {
	return en(carpetaDeCompartidas, dueno+"-"+ref+".esfinge")
}

// CarpetaDeProyectos y CarpetaDeCompartidas son donde van todos, para poder
// recorrerlos. La segunda **separa lo ajeno de lo propio en el disco**, que es lo que
// hace que una no se pueda confundir con la otra al mirar la carpeta o al borrar.
func CarpetaDeProyectos() string   { return en(carpetaDeProyectos) }
func CarpetaDeCompartidas() string { return en(carpetaDeCompartidas) }

// RelativaDeProyecto es el fichero **relativo a la carpeta de Esfinge**, que es como
// lo guarda el índice de bóvedas: así el índice sobrevive a que la carpeta se mueva.
func RelativaDeProyecto(ref string) string {
	return filepath.Join(carpetaDeProyectos, ref+".esfinge")
}

func en(trozos ...string) string {
	c := DeEsfinge()
	if c == "" {
		return ""
	}
	return filepath.Join(append([]string{c}, trozos...)...)
}

// ApartarLaCarpeta manda la carpeta de configuración a otro sitio, para una prueba.
//
// **Son cinco variables porque `os.UserConfigDir` mira una distinta en cada
// sistema**, y escribirlas a mano sale bien en esta máquina y mal en Windows: allí
// nadie mira, y lo que se ve es una prueba quejándose de encontrarse una bóveda que
// no ha creado, porque la creó otra que corrió antes en la carpeta de verdad. Pasó
// —`conReloj` tenía su lista sin la de Windows y seis sitios la copiaban— y **tiró
// una publicación**.
//
// Vive aquí y no en un ayudante de pruebas por eso: desde la ADR 0057 hay dos
// paquetes que apartan la carpeta, y la lista tiene que ser **una**. No importa
// `testing` a propósito —esto es código de producción— así que recibe el `t.Setenv`
// que lo hace. Lo vigila `TestLaCarpetaDeConfiguracionSeApartaEnUnSoloSitio`.
func ApartarLaCarpeta(dir string, poner func(nombre, valor string)) {
	for _, v := range VariablesDeLaCarpeta {
		poner(v, dir)
	}
}

// VariablesDeLaCarpeta son las que miran los tres sistemas para decir dónde va la
// configuración. Ver `ApartarLaCarpeta`.
var VariablesDeLaCarpeta = []string{
	"XDG_CONFIG_HOME", // Linux
	"HOME",            // macOS
	"AppData",         // Windows
	"USERPROFILE",
	"LOCALAPPDATA", // y la caché de Windows, por lo mismo
}
