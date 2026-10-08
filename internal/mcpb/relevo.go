package mcpb

// El relevo hacia el servidor de la Esfinge instalada (2026-10-08).
//
// **El paquete `.mcpb` lleva el servidor dentro**, que es lo que hace que no se rompa
// al mover Esfinge de carpeta. El precio salió al usarlo: actualizar Esfinge **no
// actualiza ese servidor**, y como los clientes MCP se quedan con la lista de
// herramientas al conectar, una herramienta nueva no aparece ni reiniciando la
// conversación. El cliente pidió crear una cuenta con su sitio y el agente contestó
// que su versión no tenía ese campo — y tenía razón: era la de antes.
//
// La salida es que el binario del paquete, al arrancar, **le ceda el turno al de la
// aplicación instalada**. Y manda ése, no el más nuevo de los dos: es con **esa**
// Esfinge con la que se va a hablar, así que su servidor es el que entiende su canal.
//
// **Todo está escrito para no romperse**: si no hay aplicación, si la ruta no es
// ejecutable o si el relevo falla, sigue el de dentro del paquete, que es lo que hay
// hoy. Lo peor que puede pasar es quedarse como estaba.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DondeBuscarElServidor son los sitios donde puede estar la Esfinge instalada.
//
// **Es una lista y las listas se quedan cortas**, así que lo que cuelga de ella es una
// mejora y nunca lo que hace que algo funcione: no encontrar nada deja las cosas como
// estaban. `casa` es el directorio del usuario, que se pasa para poder probar los tres
// sistemas desde cualquiera.
func DondeBuscarElServidor(sistema, casa string) []string {
	nombre := "esfinge-mcp"
	if sistema == "windows" {
		nombre += ".exe"
	}
	// **Se unen con el separador del sistema de destino, no con el de esta máquina.**
	// Es la regla que este proyecto ya tiene escrita para los manifiestos del
	// navegador: `filepath.Join` usa el de donde corre, así que armando la ruta de
	// Windows desde Linux sale `C:\Program Files/Esfinge/...` y **la tabla de Windows
	// dejaría de ser comprobable desde aquí**, que es justo donde se comprueba.
	var sitios [][]string
	switch sistema {
	case "darwin":
		sitios = [][]string{
			{"/Applications", "Esfinge.app", "Contents", "MacOS", nombre},
			{casa, "Applications", "Esfinge.app", "Contents", "MacOS", nombre},
		}
	case "windows":
		// Donde deja las cosas el instalador, y la carpeta por usuario que ofrece.
		sitios = [][]string{
			{`C:\Program Files`, "Esfinge", nombre},
			{casa, "AppData", "Local", "Programs", "Esfinge", nombre},
		}
	default:
		// El `.deb` y el tar.gz.
		sitios = [][]string{
			{"/usr/bin", nombre},
			{"/usr/local/bin", nombre},
			{casa, ".local", "bin", nombre},
		}
	}
	out := make([]string, 0, len(sitios))
	for _, partes := range sitios {
		out = append(out, unir(sistema, partes...))
	}
	return out
}

// unir arma una ruta para `sistema`, con su separador y no con el de esta máquina.
func unir(sistema string, partes ...string) string {
	sep := "/"
	if sistema == "windows" {
		sep = `\`
	}
	var out string
	for _, p := range partes {
		if p == "" {
			continue
		}
		if out == "" {
			out = strings.TrimSuffix(p, sep)
			continue
		}
		out += sep + strings.Trim(p, sep)
	}
	return out
}

// AQuienCederle devuelve el servidor instalado al que pasarle el turno, o vacío si hay
// que seguir.
//
// `yo` es la ruta de quien pregunta: **no se cede a uno mismo**, que es lo que pasaría
// cuando este binario *es* el de la aplicación —el caso de Claude Code, donde la orden
// ya apunta ahí— y daría una cadena infinita de procesos.
//
// `existe` se inyecta para poder probar los tres sistemas desde cualquiera; en
// producción mira el disco y comprueba que se puede ejecutar.
func AQuienCederle(sistema, casa, yo string, existe func(string) bool) string {
	yo = limpia(yo)
	for _, sitio := range DondeBuscarElServidor(sistema, casa) {
		if limpia(sitio) == yo {
			return ""
		}
		if existe(sitio) {
			return sitio
		}
	}
	return ""
}

// limpia deja una ruta comparable: absoluta y sin enlaces, y si algo de eso falla se
// queda con lo que había. **Comparar mal aquí no puede impedir arrancar.**
func limpia(ruta string) string {
	if ruta == "" {
		return ""
	}
	if abs, err := filepath.Abs(ruta); err == nil {
		ruta = abs
	}
	if real, err := filepath.EvalSymlinks(ruta); err == nil {
		ruta = real
	}
	return ruta
}

// SePuedeEjecutar dice si ahí hay un fichero que se puede lanzar.
func SePuedeEjecutar(ruta string) bool {
	fi, err := os.Stat(ruta)
	if err != nil || fi.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return fi.Mode().Perm()&0o111 != 0
}
