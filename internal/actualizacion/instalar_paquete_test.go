package actualizacion

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// conGestor hace que esta copia parezca instalada con el gestor que se le diga, apartando
// la carpeta donde el paquete lo deja escrito.
func conGestor(t *testing.T, gestor string) {
	t.Helper()
	dir := t.TempDir()
	if gestor != "" {
		if err := os.WriteFile(filepath.Join(dir, "gestor"), []byte(gestor+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	viejo := carpetaDeEsfinge
	carpetaDeEsfinge = dir
	t.Cleanup(func() { carpetaDeEsfinge = viejo })
	// Y que no se cuele la copia de verdad por el camino de «vivo en /usr/bin».
	t.Setenv("APPIMAGE", "")
}

// conProgramas dice qué hay instalado en esta máquina de mentira. Recibir la lista es lo que
// permite comprobar los cuatro gestores desde un VPS que solo tiene uno.
func conProgramas(t *testing.T, cuales ...string) {
	t.Helper()
	hay := map[string]bool{}
	for _, c := range cuales {
		hay[c] = true
	}
	viejo := hayPrograma
	hayPrograma = func(p string) bool { return hay[p] }
	t.Cleanup(func() { hayPrograma = viejo })
}

// La orden sale del gestor **y de lo que haya instalado**, y prefiere el gestor de paquetes
// al instalador crudo: una versión nueva puede traer una dependencia nueva, y `dpkg -i` la
// dejaría a medias donde `apt-get install` la resuelve.
func TestLaOrdenDeInstalarSaleDelGestorYDeLoQueHaya(t *testing.T) {
	casos := []struct {
		nombre   string
		gestor   string
		hay      []string
		esperada string
	}{
		{"deb con apt", "deb", []string{"pkexec", "apt-get", "dpkg"},
			"pkexec apt-get install -y --allow-downgrades /tmp/esfinge.deb"},
		{"deb sin apt, con dpkg", "deb", []string{"pkexec", "dpkg"},
			"pkexec dpkg -i /tmp/esfinge.deb"},
		{"rpm con dnf", "rpm", []string{"pkexec", "dnf", "rpm"},
			"pkexec dnf install -y /tmp/esfinge.deb"},
		{"rpm en openSUSE", "rpm", []string{"pkexec", "zypper", "rpm"},
			"pkexec zypper --non-interactive install --allow-unsigned-rpm /tmp/esfinge.deb"},
		{"rpm a pelo", "rpm", []string{"pkexec", "rpm"},
			"pkexec rpm -U /tmp/esfinge.deb"},

		// Y los que no instalan nada, que importan igual: en Arch lo actualiza pacman y
		// de una AppImage no se instala nada, así que ofrecer «Instalar y reiniciar» sería
		// un botón que no hace lo que dice.
		{"Arch no", "pacman", []string{"pkexec", "pacman"}, ""},
		{"una AppImage no", "appimage", []string{"pkexec", "apt-get"}, ""},
		{"quien se bajó el tar.gz no", "", []string{"pkexec", "apt-get"}, ""},

		// Sin pkexec no hay a quién pedirle permiso sin una terminal delante.
		{"deb sin pkexec", "deb", []string{"apt-get", "dpkg"}, ""},
		{"rpm sin pkexec", "rpm", []string{"dnf"}, ""},

		// Y con pkexec pero sin ninguna herramienta, tampoco se inventa nada.
		{"deb sin dpkg ni apt", "deb", []string{"pkexec"}, ""},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			hay := map[string]bool{}
			for _, h := range c.hay {
				hay[h] = true
			}
			orden := ordenParaInstalar(c.gestor, "/tmp/esfinge.deb", func(p string) bool { return hay[p] })
			vista := strings.Join(orden, " ")
			if vista != c.esperada {
				t.Errorf("con gestor «%s» y %v la orden es «%s» y tenía que ser «%s»",
					c.gestor, c.hay, vista, c.esperada)
			}
		})
	}
}

// Un paquete se instala solo; una AppImage no, aunque la máquina tenga todo lo demás.
//
// El orden de las dos preguntas importa: la AppImage **gana**, porque dentro de una
// AppImage puede haber un `/usr/share/esfinge/gestor` de la copia instalada con el paquete
// y creerle sería instalar encima de otra cosa.
func TestUnPaqueteSeInstalaSoloYUnaAppImageNo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	conProgramas(t, "pkexec", "apt-get", "dpkg")

	conGestor(t, "deb")
	if m := ComoSeInstala(); m != ModoSolo {
		t.Errorf("con un .deb y pkexec el modo es «%s» y tenía que ser «%s»", m, ModoSolo)
	}

	t.Setenv("APPIMAGE", "/home/quien-sea/Aplicaciones/Esfinge-2.68.0-x86_64.AppImage")
	if m := ComoSeInstala(); m != ModoAMano {
		t.Errorf("desde una AppImage el modo es «%s» y tenía que ser «%s»", m, ModoAMano)
	}
}

// Y sin pkexec se vuelve a lo de siempre: bajar el paquete y dárselo al sistema.
func TestSinPkexecElPaqueteLoInstalaElSistema(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	conGestor(t, "deb")
	conProgramas(t, "apt-get", "dpkg")
	if m := ComoSeInstala(); m != ModoInstalador {
		t.Errorf("sin pkexec el modo es «%s» y tenía que ser «%s»", m, ModoInstalador)
	}
}

// Instalar llama al gestor **con el fichero recién descargado** y luego pide el relevo.
//
// Las dos mitades se miran: una orden correcta que no reabra deja a la persona con la
// versión vieja en pantalla y la nueva en el disco, sin que nada lo diga.
func TestInstalarUnPaqueteLlamaAlGestorYLuegoReabre(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	conGestor(t, "deb")
	conProgramas(t, "pkexec", "apt-get")

	bajado := filepath.Join(t.TempDir(), "esfinge_2.68.0_amd64.deb")
	if err := os.WriteFile(bajado, []byte("el paquete"), 0o644); err != nil {
		t.Fatal(err)
	}

	var ordenes [][]string
	viejoEjecutar := ejecutar
	ejecutar = func(o []string) error { ordenes = append(ordenes, o); return nil }
	t.Cleanup(func() { ejecutar = viejoEjecutar })

	reabierta := 0
	viejoReabrir := reabrir
	reabrir = func() error { reabierta++; return nil }
	t.Cleanup(func() { reabrir = viejoReabrir })

	if err := Instalar(bajado); err != nil {
		t.Fatalf("instalar: %v", err)
	}

	if len(ordenes) != 1 {
		t.Fatalf("se ha llamado al gestor %d veces y tenía que ser una", len(ordenes))
	}
	if ultimo := ordenes[0][len(ordenes[0])-1]; ultimo != bajado {
		t.Errorf("se ha instalado «%s» y lo descargado era «%s»", ultimo, bajado)
	}
	if ordenes[0][0] != "pkexec" {
		t.Errorf("la orden empieza por «%s» y tenía que pedir permiso con pkexec", ordenes[0][0])
	}
	if reabierta != 1 {
		t.Errorf("se ha pedido reabrir %d veces y tenía que ser una", reabierta)
	}
}

// Cancelar el diálogo de la contraseña no es un fallo del programa, y tiene que decirse con
// palabras: «exit status 126» no le dice nada a nadie.
func TestSiNoSeAutorizaSeDiceConPalabras(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	conGestor(t, "deb")
	conProgramas(t, "pkexec", "apt-get")

	bajado := filepath.Join(t.TempDir(), "esfinge_2.68.0_amd64.deb")
	if err := os.WriteFile(bajado, []byte("el paquete"), 0o644); err != nil {
		t.Fatal(err)
	}

	viejoEjecutar := ejecutar
	// `false` devuelve 1; lo que hace falta es un 126 de verdad, que es lo que contesta
	// pkexec cuando nadie autoriza. Se consigue con un intérprete que salga con ese código.
	ejecutar = func([]string) error { return salidaCon(126) }
	t.Cleanup(func() { ejecutar = viejoEjecutar })

	reabierta := 0
	viejoReabrir := reabrir
	reabrir = func() error { reabierta++; return nil }
	t.Cleanup(func() { reabrir = viejoReabrir })

	err := Instalar(bajado)
	if err == nil {
		t.Fatal("cancelar la autorización tiene que dar error")
	}
	if !strings.Contains(err.Error(), "autorización") {
		t.Errorf("el error dice «%v» y tenía que hablar de la autorización", err)
	}
	if reabierta != 0 {
		t.Error("no se puede reabrir Esfinge si la instalación no se ha hecho")
	}
}

// salidaCon devuelve el error que devolvería una orden que termina con ese código, que es
// un `*exec.ExitError` de verdad y no uno fabricado: lo que se está comprobando es que el
// código se lee bien, y con un error inventado se comprobaría el invento.
func salidaCon(codigo int) error {
	err := exec.Command("sh", "-c", fmt.Sprintf("exit %d", codigo)).Run()
	if err == nil {
		panic("se esperaba un fallo")
	}
	return err
}
