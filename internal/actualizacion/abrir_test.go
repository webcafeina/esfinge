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

// Y la tabla entera, **con el mundo por parámetro**: los cinco gestores y los dos sitios donde
// se mira, comprobados desde un VPS que solo conoce uno.
//
// Es la misma razón por la que las rutas de los manifiestos del navegador se calculan para un
// sistema dado: de lo contrario, lo de Windows no lo probaría nadie hasta que una máquina de
// GitHub lo ejecutara en medio de una publicación.
func TestDeQueGestorVieneCadaCopia(t *testing.T) {
	const programa = `C:\Users\quien-sea\AppData\Local\Programs\esfinge\esfinge.exe`
	const carpeta = "/usr/share/esfinge"

	casos := []struct {
		nombre          string
		sistema         string
		appimage        string
		ejecutable      string
		ficheros        map[string]string
		espera          string
		loActualizaOtro bool
	}{
		{"Debian con su .deb", "linux", "", "/usr/bin/esfinge-gui",
			map[string]string{carpeta + "/gestor": "deb\n"}, adjuntos.GestorDeb, false},
		{"Fedora con su .rpm", "linux", "", "/usr/bin/esfinge-gui",
			map[string]string{carpeta + "/gestor": "rpm\n"}, adjuntos.GestorRPM, false},
		{"Arch con pacman", "linux", "", "/usr/bin/esfinge-gui",
			map[string]string{carpeta + "/gestor": "pacman\n"}, adjuntos.GestorPacman, true},
		{"una AppImage, que se anuncia sola", "linux", "/home/quien-sea/Esfinge.AppImage", "/tmp/.mount_x/usr/bin/esfinge-gui",
			map[string]string{carpeta + "/gestor": "deb\n"}, adjuntos.GestorAppImage, false},
		{"el tar.gz puesto a mano", "linux", "", "/home/quien-sea/bin/esfinge-gui",
			nil, adjuntos.SinGestor, false},
		{"un .deb viejo, de antes del fichero", "linux", "", "/usr/bin/esfinge-gui",
			nil, adjuntos.GestorDeb, false},
		{"basura en el fichero no se cree", "linux", "", "/home/quien-sea/bin/esfinge-gui",
			map[string]string{carpeta + "/gestor": "lo-que-sea"}, adjuntos.SinGestor, false},

		// Windows: al lado del programa, y sólo si el paquete lo pidió.
		{"Windows con Chocolatey", "windows", "", programa,
			map[string]string{`C:\Users\quien-sea\AppData\Local\Programs\esfinge\gestor`: "chocolatey"},
			adjuntos.GestorChocolatey, true},
		{"Windows con winget", "windows", "", programa,
			map[string]string{`C:\Users\quien-sea\AppData\Local\Programs\esfinge\gestor`: "winget"},
			adjuntos.GestorWinget, true},
		{"Windows con el instalador a mano", "windows", "", programa,
			nil, adjuntos.SinGestor, false},
		// Y lo que **no** puede pasar: que en Windows se cuele el razonamiento de Linux. Un
		// programa en «/usr/bin» allí no existe, pero si alguien reordenara las guardas, esto
		// contestaría «deb» y a un Windows se le ofrecería un paquete de Debian.
		{"Windows no cae en el camino de Linux", "windows", "", "/usr/bin/esfinge-gui",
			nil, adjuntos.SinGestor, false},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			leer := func(f string) string { return c.ficheros[f] }
			if hay := gestorSegun(c.sistema, c.appimage, carpeta, c.ejecutable, leer); hay != c.espera {
				t.Errorf("dice %q y tenía que decir %q", hay, c.espera)
			}
		})
	}
}

// Y de ahí sale lo que la ventana enseña: a quién se le dice «lo actualiza tu gestor».
//
// Los tres son los que **no pueden recibir nada nuestro**: Arch, porque no publicamos paquete
// para pacman; Chocolatey y winget, porque su paquete **es** el instalador publicado y
// adelantarnos dejaría al gestor viendo una versión vieja. Los demás sí se actualizan desde la
// banda, y por eso están aquí abajo.
func TestAQuienSeLeDiceQueLoActualizaSuGestor(t *testing.T) {
	casos := map[string]bool{
		adjuntos.GestorPacman:     true,
		adjuntos.GestorChocolatey: true,
		adjuntos.GestorWinget:     true,
		adjuntos.GestorDeb:        false,
		adjuntos.GestorRPM:        false,
		adjuntos.GestorAppImage:   false,
		adjuntos.SinGestor:        false,
	}
	for gestor, espera := range casos {
		dir := t.TempDir()
		if gestor != adjuntos.SinGestor {
			if err := os.WriteFile(filepath.Join(dir, "gestor"), []byte(gestor+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		viejo := carpetaDeEsfinge
		carpetaDeEsfinge = dir
		t.Setenv("APPIMAGE", "")
		hay := LoActualizaSuGestor()
		carpetaDeEsfinge = viejo
		if hay != espera {
			t.Errorf("con «%s» dice %v y tenía que decir %v", gestor, hay, espera)
		}
	}
}
