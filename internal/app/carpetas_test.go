package app

import (
	"os"
	"path/filepath"
	"testing"
)

// El diálogo tiene que abrirse donde se quedó la última vez, y abrir y guardar
// se recuerdan por separado: son gestos distintos y casi nunca van al mismo
// sitio.
func TestLosDialogosRecuerdanSuCarpeta(t *testing.T) {
	a, s := nuevaDePrueba(t)

	// La primera vez no hay nada que recordar: manda el sistema.
	s.ficheros = []string{filepath.Join(t.TempDir(), "credenciales.env")}
	if _, err := a.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}
	if s.desdeAbrir != "" {
		t.Errorf("la primera vez no hay carpeta que proponer, y ha propuesto %q", s.desdeAbrir)
	}

	// La segunda se abre donde acabó la primera.
	deDonde := filepath.Dir(s.ficheros[0])
	if _, err := a.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}
	if s.desdeAbrir != deDonde {
		t.Errorf("abrir: quiero %q, tengo %q", deDonde, s.desdeAbrir)
	}

	// Guardar lleva su propia cuenta y no se contagia de la de abrir.
	otra := t.TempDir()
	s.guardaEn = filepath.Join(otra, "secreto.esf")
	if _, err := a.GuardarTexto("secreto.esf", "ESF1.x"); err != nil {
		t.Fatal(err)
	}
	if s.desdeGuardar != "" {
		t.Errorf("guardar aún no tenía carpeta, y ha propuesto %q", s.desdeGuardar)
	}
	if _, err := a.GuardarTexto("otro.esf", "ESF1.y"); err != nil {
		t.Fatal(err)
	}
	if s.desdeGuardar != otra {
		t.Errorf("guardar: quiero %q, tengo %q", otra, s.desdeGuardar)
	}
	if s.desdeAbrir == otra {
		t.Error("la carpeta de guardar se ha colado en la de abrir")
	}
}

// Sobreviven a cerrar la aplicación, que es de lo que se trata.
func TestLasCarpetasSobrevivenAlCierre(t *testing.T) {
	a, s := nuevaDePrueba(t)

	dir := t.TempDir()
	s.ficheros = []string{filepath.Join(dir, "algo.env")}
	if _, err := a.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}

	// Otra aplicación, la misma carpeta de configuración: es lo que pasa al
	// volver a abrir la ventana.
	otra := &sistemaFalso{ficheros: s.ficheros}
	b := Nueva("prueba", otra)
	if _, err := b.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}
	if otra.desdeAbrir != dir {
		t.Errorf("al volver a abrir: quiero %q, tengo %q", dir, otra.desdeAbrir)
	}
}

// Una carpeta que ya no existe no se propone: Wails falla la llamada entera si
// el directorio por defecto no está, así que recordar de más dejaría el diálogo
// sin abrir.
func TestUnaCarpetaQueYaNoEstaNoSePropone(t *testing.T) {
	a, s := nuevaDePrueba(t)

	dir := t.TempDir()
	fuera := filepath.Join(dir, "se-va-a-borrar")
	if err := os.Mkdir(fuera, 0o700); err != nil {
		t.Fatal(err)
	}
	s.ficheros = []string{filepath.Join(fuera, "algo.env")}
	if _, err := a.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(fuera); err != nil {
		t.Fatal(err)
	}

	if _, err := a.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}
	if s.desdeAbrir != "" {
		t.Errorf("la carpeta ya no existe y la propone igual: %q", s.desdeAbrir)
	}
}

// Cancelar no cambia nada: no se ha elegido carpeta ninguna.
func TestCancelarNoCambiaLaCarpeta(t *testing.T) {
	a, s := nuevaDePrueba(t)

	dir := t.TempDir()
	s.ficheros = []string{filepath.Join(dir, "algo.env")}
	if _, err := a.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}

	s.ficheros = nil // como cuando se cierra el diálogo sin elegir
	if _, err := a.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ElegirFicheros(false); err != nil {
		t.Fatal(err)
	}
	if s.desdeAbrir != dir {
		t.Errorf("cancelar ha borrado la carpeta: quiero %q, tengo %q", dir, s.desdeAbrir)
	}
}
