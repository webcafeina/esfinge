package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// La ruta que se le da a otro programa para que arranque el servidor MCP.
//
// # El problema, que solo lo tiene la AppImage
//
// Lo que Ajustes enseña se pega en la configuración de Claude Desktop, de Claude Code, de
// Cursor o de VS Code, y **tiene que seguir valiendo la semana que viene**: esos programas
// arrancan los servidores desde un directorio indefinido, así que la ruta es absoluta y la
// escribe una persona una vez.
//
// Instalada de cualquier otra forma, el binario está al lado del que se ejecuta y ahí se
// queda: /usr/bin, Program Files, el paquete .app. **Una AppImage no**: se monta en
// /tmp/.mount_EsfingeXXXXXX, con un nombre distinto en cada arranque, que desaparece al
// cerrar. Pegar esa ruta es pegar algo que funciona hoy y no mañana, **y el fallo es mudo**:
// el cliente MCP dice que el servidor no arranca y nadie relaciona eso con haber cerrado
// Esfinge.
//
// # La salida, y por qué ~/.local/bin y no nuestra carpeta
//
// Se deja una copia en ~/.local/bin/esfinge-mcp y se da **esa** ruta.
//
// Lo primero que pensé fue dejarla en nuestra carpeta de configuración, que es nuestra y no
// mete nada en el PATH de nadie. Está mal, y lo dice el código: **`mcpb.DondeBuscarElServidor`
// ya busca en ~/.local/bin**. El relevo del paquete .mcpb —el binario de dentro le cede el
// turno al de la Esfinge instalada, para que una herramienta nueva aparezca al actualizar—
// solo encuentra lo que esté en esa lista. Dejándola en otro sitio, quien use la AppImage se
// quedaría con la lista de herramientas congelada del día que instaló el paquete, que es
// exactamente el fallo que el relevo vino a arreglar.
//
// Así que la copia va donde el resto del programa ya mira. Y de paso le arregla el relevo a
// quien tenga el .mcpb instalado sin haber abierto Ajustes nunca.
//
// # Lo que no se hace
//
// No se copia al arrancar: se copia **cuando alguien pide la ruta**, que es cuando hace
// falta. Y si no se puede, **se dice** —ésa es la lección del `catch` vacío de la casilla de
// Touch ID—: dar una ruta que va a dejar de valer sería peor que no dar ninguna.

// desdeDondeCorro dice si esta copia se está ejecutando desde una AppImage.
//
// La AppImage se anuncia ella misma por el entorno, igual que para decidir qué paquete
// ofrecer al actualizarse (`gestorQueLoInstalo`).
func enUnaAppImage() bool { return os.Getenv("APPIMAGE") != "" }

// rutaUsableDelServidorMCP es la ruta que se puede pegar en otro programa.
//
// Devuelve además el aviso que hay que enseñar cuando no se ha podido dejar en un sitio
// estable, que es lo único que la ventana puede contar.
func rutaUsableDelServidorMCP() (string, string) {
	return laRutaUsable(rutaDelServidorMCP, enUnaAppImage(), os.UserHomeDir)
}

// laRutaUsable es la de arriba con el mundo por parámetro, **que es lo que la hace
// comprobable**: desde aquí no se puede cambiar lo que devuelve `os.Executable`, así que
// sin esto el único caso que una prueba podría ejercitar es el de la máquina donde corre
// —el mismo motivo por el que `elDeEsteEquipo` recibe el sistema y la arquitectura—.
func laRutaUsable(alLadoDe func() (string, error), appimage bool, casaDe func() (string, error)) (string, string) {
	alLado, err := alLadoDe()
	if err != nil || alLado == "" {
		// Sin saber dónde estoy, lo único honrado es el nombre a secas, que es lo que
		// había antes de todo esto.
		return "esfinge-mcp", ""
	}
	if !appimage {
		return alLado, ""
	}

	casa, err := casaDe()
	if err != nil {
		return alLado, "Esto corre desde una AppImage y no encuentro tu carpeta personal, así que la ruta de abajo deja de valer al cerrar Esfinge."
	}
	estable, err := dejarElServidorMCPAMano(alLado, casa)
	if err != nil {
		return alLado, "Esto corre desde una AppImage y no he podido dejar el servidor en una ruta fija (" + err.Error() + "), así que la de abajo deja de valer al cerrar Esfinge."
	}
	return estable, ""
}

// dejarElServidorMCPAMano pone el binario en ~/.local/bin y devuelve su ruta.
//
// `casa` por parámetro para poder probarlo, que es la regla de esta casa desde que una
// prueba solo podía comprobar la máquina donde corría.
func dejarElServidorMCPAMano(deDentro, casa string) (string, error) {
	nombre := "esfinge-mcp"
	if runtime.GOOS == "windows" {
		nombre += ".exe"
	}
	carpeta := filepath.Join(casa, ".local", "bin")
	destino := filepath.Join(carpeta, nombre)

	origen, err := os.Stat(deDentro)
	if err != nil {
		return "", fmt.Errorf("no encuentro el servidor dentro de la AppImage")
	}
	// **Se compara tamaño y fecha, no el contenido.** Esto se llama cada vez que alguien
	// mira Ajustes y el binario son diez megas: leerlos para decidir que no hay nada que
	// hacer sería pagar la copia siempre. La fecha se copia a propósito justo para que
	// esta comparación signifique algo.
	if ya, err := os.Stat(destino); err == nil &&
		ya.Size() == origen.Size() && ya.ModTime().Equal(origen.ModTime()) {
		return destino, nil
	}

	if err := os.MkdirAll(carpeta, 0o755); err != nil {
		return "", err
	}
	// Se escribe al lado y se mueve encima, que es lo único que funciona si el de antes
	// está corriendo: escribir sobre un ejecutable en uso da «text file busy», y un
	// renombrado solo cambia la entrada del directorio.
	tmp := destino + ".nuevo"
	if err := copiarFichero(deDentro, tmp); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Chtimes(tmp, origen.ModTime(), origen.ModTime()); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, destino); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return destino, nil
}

func copiarFichero(origen, destino string) error {
	o, err := os.Open(origen)
	if err != nil {
		return err
	}
	defer o.Close()
	d, err := os.OpenFile(destino, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, o); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}
