package actualizacion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/webcafeina/esfinge/internal/adjuntos"
)

// TestConQueGestorSeInstaloEstaCopia cubre lo que el paquete deja escrito y los dos
// caminos de cuando no hay nada escrito.
//
// Lo que esto vigila de verdad: **que un valor que no se reconoce no se crea**. Un
// fichero con basura dentro —o de una versión futura que invente un gestor nuevo— no
// puede hacer que a nadie se le ofrezca el paquete de otro.
func TestConQueGestorSeInstaloEstaCopia(t *testing.T) {
	apartar := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		viejo := carpetaDeEsfinge
		carpetaDeEsfinge = dir
		t.Cleanup(func() { carpetaDeEsfinge = viejo })
		return dir
	}
	poner := func(t *testing.T, dir, que string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "gestor"), []byte(que), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	for _, g := range []string{adjuntos.GestorDeb, adjuntos.GestorRPM, adjuntos.GestorPacman} {
		dir := apartar(t)
		poner(t, dir, g+"\n") // con su salto de línea, como lo escribe el postinstall
		if hay := gestorQueLoInstalo(); hay != g {
			t.Errorf("con «%s» escrito, dice %q", g, hay)
		}
	}

	// **Basura dentro: no se cree.** Se cae al razonamiento de la ruta, y lo que no
	// puede pasar es que devuelva lo que diga el fichero.
	dir := apartar(t)
	poner(t, dir, "pacman-pero-mal")
	if hay := gestorQueLoInstalo(); hay == "pacman-pero-mal" {
		t.Error("se ha creído un gestor que no existe")
	}

	// Sin fichero, lo de siempre: el ejecutable de las pruebas no vive en /usr/bin, así
	// que esto es «me pusieron a mano», que es lo que se quiere para el tar.gz.
	_ = apartar(t)
	if hay := gestorQueLoInstalo(); hay != adjuntos.SinGestor {
		t.Errorf("sin fichero y fuera de /usr/bin dice %q", hay)
	}

	// Y una AppImage se anuncia por el entorno, que manda sobre todo lo demás.
	dir = apartar(t)
	poner(t, dir, adjuntos.GestorDeb)
	t.Setenv("APPIMAGE", "/tmp/Esfinge-x86_64.AppImage")
	if hay := gestorQueLoInstalo(); hay != adjuntos.GestorAppImage {
		t.Errorf("dentro de una AppImage dice %q", hay)
	}
}
