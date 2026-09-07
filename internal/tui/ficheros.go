package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// LimpiarRuta arregla lo que un terminal deja caer cuando se le arrastra un
// fichero encima.
//
// Arrastrar es la forma cómoda de dar una ruta, y ningún terminal la pega
// limpia: unos la rodean de comillas, otros escapan los espacios con barras, y
// casi todos añaden un espacio al final. Sin esto, una carpeta con un espacio en
// el nombre —«Mis documentos»— hace que el fichero no aparezca por ninguna parte.
func LimpiarRuta(s string) string {
	s = strings.TrimSpace(s)

	// Comillas alrededor, de las tres clases que usan los distintos terminales.
	for _, par := range [][2]string{{`"`, `"`}, {`'`, `'`}, {"`", "`"}} {
		if len(s) >= 2 && strings.HasPrefix(s, par[0]) && strings.HasSuffix(s, par[1]) {
			s = s[1 : len(s)-1]
			break
		}
	}

	// Escapes de barra invertida: «\ » por un espacio, y lo mismo con los otros
	// caracteres que un shell protege.
	if strings.Contains(s, `\`) && !esRutaDeWindows(s) {
		reemplazos := []string{`\ `, " ", `\(`, "(", `\)`, ")", `\'`, "'", `\"`, `"`,
			`\&`, "&", `\;`, ";", `\$`, "$", `\\`, `\`}
		s = strings.NewReplacer(reemplazos...).Replace(s)
	}

	s = strings.TrimSpace(s)

	// La virgulilla la expande el shell, no el programa: si llega escrita es
	// porque nadie la ha tocado.
	if s == "~" || strings.HasPrefix(s, "~/") {
		if casa, err := os.UserHomeDir(); err == nil {
			s = filepath.Join(casa, strings.TrimPrefix(strings.TrimPrefix(s, "~"), "/"))
		}
	}
	return s
}

// esRutaDeWindows distingue «C:\Users\...» de una ruta con escapes. En Windows
// la barra invertida es el separador y no hay nada que desescapar.
func esRutaDeWindows(s string) bool {
	if len(s) >= 3 && s[1] == ':' && (s[2] == '\\' || s[2] == '/') {
		return true
	}
	return strings.HasPrefix(s, `\\`) // recurso de red
}

// CifrarFichero sella el fichero de origen y devuelve dónde ha quedado.
func CifrarFichero(origen string, clave []byte) (string, error) {
	origen = LimpiarRuta(origen)
	if err := comprobarLegible(origen); err != nil {
		return "", err
	}

	destino, err := nombreLibre(origen + ".esf")
	if err != nil {
		return "", err
	}

	entrada, err := os.Open(origen)
	if err != nil {
		return "", fmt.Errorf("No puedo abrir %s: %w", filepath.Base(origen), err)
	}
	defer entrada.Close()

	salida, err := os.OpenFile(destino, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("No puedo escribir el fichero cifrado: %w", err)
	}
	defer salida.Close()

	w := bufio.NewWriter(salida)
	if err := cripto.SellarFlujo(w, bufio.NewReader(entrada), clave, cripto.PerfilInteractivo); err != nil {
		os.Remove(destino)
		return "", err
	}
	if err := w.Flush(); err != nil {
		os.Remove(destino)
		return "", err
	}
	return destino, nil
}

// DescifrarFichero abre un contenedor y devuelve dónde ha dejado el original.
//
// Acepta tanto un fichero binario como uno que guarde el texto ESF1.… de una
// línea, porque quien lo recibe no tiene por qué saber cuál de los dos le han
// mandado.
func DescifrarFichero(origen string, clave []byte) (string, error) {
	origen = LimpiarRuta(origen)
	if err := comprobarLegible(origen); err != nil {
		return "", err
	}

	entrada, err := os.Open(origen)
	if err != nil {
		return "", fmt.Errorf("No puedo abrir %s: %w", filepath.Base(origen), err)
	}
	defer entrada.Close()

	buf := bufio.NewReader(entrada)
	destino, err := nombreLibre(sinExtensionEsf(origen))
	if err != nil {
		return "", err
	}

	// El quinto byte distingue los dos formatos: en el de texto es el punto de
	// «ESF1.» y en el binario es el número de versión.
	cabeza, _ := buf.Peek(5)
	if len(cabeza) < 5 {
		return "", cripto.ErrFormato
	}

	if cabeza[4] == '.' {
		crudo, err := os.ReadFile(origen)
		if err != nil {
			return "", err
		}
		datos, err := cripto.AbrirTexto(string(crudo), clave)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(destino, datos, 0o600); err != nil {
			return "", err
		}
		return destino, nil
	}

	salida, err := os.OpenFile(destino, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("No puedo escribir el resultado: %w", err)
	}
	defer salida.Close()

	w := bufio.NewWriter(salida)
	if err := cripto.AbrirFlujo(w, buf, clave); err != nil {
		os.Remove(destino)
		return "", err
	}
	if err := w.Flush(); err != nil {
		os.Remove(destino)
		return "", err
	}
	return destino, nil
}

func comprobarLegible(ruta string) error {
	if strings.TrimSpace(ruta) == "" {
		return fmt.Errorf("Falta la ruta del fichero")
	}
	info, err := os.Stat(ruta)
	if os.IsNotExist(err) {
		return fmt.Errorf("No existe ningún fichero en %s", ruta)
	}
	if err != nil {
		return fmt.Errorf("No puedo leer %s: %w", ruta, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s es una carpeta, no un fichero", filepath.Base(ruta))
	}
	return nil
}

func sinExtensionEsf(ruta string) string {
	if strings.HasSuffix(ruta, ".esf") {
		return strings.TrimSuffix(ruta, ".esf")
	}
	return ruta + ".claro"
}

// nombreLibre busca un nombre que no exista todavía, para no pisar nada de nadie.
func nombreLibre(preferido string) (string, error) {
	if _, err := os.Stat(preferido); os.IsNotExist(err) {
		return preferido, nil
	}

	ext := filepath.Ext(preferido)
	base := strings.TrimSuffix(preferido, ext)
	for i := 2; i < 1000; i++ {
		intento := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(intento); os.IsNotExist(err) {
			return intento, nil
		}
	}
	return "", fmt.Errorf("Ya hay demasiados ficheros parecidos a %s", filepath.Base(preferido))
}
