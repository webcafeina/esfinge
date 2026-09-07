package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// CarpetaDeDescargas devuelve dónde dejar los ficheros que genera la interfaz.
//
// Descargas es el sitio que todo el mundo sabe abrir, sale en la barra lateral
// del Finder y del explorador, y no depende de desde dónde se haya arrancado
// Esfinge. Antes se escribía en el directorio de trabajo, que con doble clic es
// la carpeta personal: el fichero aparecía, pero en un sitio que nadie miraba.
//
// Se busca en este orden:
//
//  1. XDG_DOWNLOAD_DIR, que es lo que respeta un Linux configurado.
//  2. La carpeta de descargas del usuario, en inglés o en español, porque en
//     macOS la ruta real es «Downloads» aunque el Finder la enseñe traducida.
//  3. La carpeta personal, si no hay ninguna de descargas.
//  4. El directorio actual, como último recurso.
func CarpetaDeDescargas() string {
	if d := desdeXDG(); d != "" {
		return d
	}

	casa, err := os.UserHomeDir()
	if err == nil {
		for _, nombre := range []string{"Downloads", "Descargas", "Baixades", "Deskargak", "Descargas"} {
			ruta := filepath.Join(casa, nombre)
			if info, err := os.Stat(ruta); err == nil && info.IsDir() {
				return ruta
			}
		}
		return casa
	}
	return "."
}

// desdeXDG lee la carpeta de descargas del fichero de configuración de usuario
// que usan los escritorios de Linux.
func desdeXDG() string {
	if d := os.Getenv("XDG_DOWNLOAD_DIR"); d != "" {
		return expandirCasa(d)
	}

	casa, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	f, err := os.Open(filepath.Join(casa, ".config", "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		linea := strings.TrimSpace(s.Text())
		if !strings.HasPrefix(linea, "XDG_DOWNLOAD_DIR=") {
			continue
		}
		valor := strings.Trim(strings.TrimPrefix(linea, "XDG_DOWNLOAD_DIR="), `"`)
		valor = strings.ReplaceAll(valor, "$HOME", casa)
		if info, err := os.Stat(valor); err == nil && info.IsDir() {
			return valor
		}
	}
	return ""
}

func expandirCasa(ruta string) string {
	if !strings.HasPrefix(ruta, "~") {
		return ruta
	}
	casa, err := os.UserHomeDir()
	if err != nil {
		return ruta
	}
	return filepath.Join(casa, strings.TrimPrefix(strings.TrimPrefix(ruta, "~"), "/"))
}
