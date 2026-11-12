package actualizacion

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Lo que decide si Esfinge puede reemplazarse sola no es el sistema, es si se
// puede escribir donde vive. Y eso solo lo responde de verdad escribir.
func TestSePuedeEscribirDondeSePuede(t *testing.T) {
	if !sePuedeEscribirEn(t.TempDir()) {
		t.Error("en una carpeta temporal propia sí se puede escribir")
	}
	if sePuedeEscribirEn(filepath.Join(t.TempDir(), "no-existe")) {
		t.Error("dice que se puede escribir en una carpeta que no existe")
	}

	// Una carpeta sin permiso de escritura. Como root no vale, que puede con
	// todo, y en Windows los permisos POSIX no significan lo mismo.
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("los permisos POSIX no deciden aquí")
	}
	cerrada := filepath.Join(t.TempDir(), "cerrada")
	if err := os.Mkdir(cerrada, 0o500); err != nil {
		t.Fatal(err)
	}
	if sePuedeEscribirEn(cerrada) {
		t.Error("dice que se puede escribir en una carpeta de solo lectura")
	}
}

// Fuera de un paquete .app no hay nada que reemplazar, y hay que decirlo en vez
// de intentar el cambiazo sobre una ruta cualquiera.
func TestSinPaqueteNoHayNadaQueReemplazar(t *testing.T) {
	if runtime.GOOS != "darwin" {
		// paqueteDeLaApp mira dónde está este ejecutable; fuera de macOS el
		// binario de los tests nunca vive en un .app, que es justo el caso.
		if _, err := paqueteDeLaApp(); err == nil {
			t.Error("ha creído estar dentro de un paquete .app")
		}
	}
}

// En Linux se instala con el gestor de paquetes: pedir escribir en /usr/bin
// desde la aplicación sería pedir la contraseña de root para algo que el sistema
// ya sabe hacer.
func TestEnLinuxSeUsaElInstaladorDelSistema(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	if ComoSeInstala() != ModoInstalador {
		t.Error("en Linux la actualización pasa por el instalador del sistema")
	}
}

// Y desde una AppImage, no la instala nadie.
//
// **No es una variante del instalador, y las palabras importan**: lo que se descarga es la
// aplicación entera, así que el botón que dijera «abrir el instalador» abriría una segunda
// Esfinge y la que se está usando seguiría siendo la vieja. Lo único que se puede hacer es
// enseñar dónde ha quedado el fichero.
//
// La variable es la misma por la que se decide qué paquete ofrecer: una AppImage se anuncia
// ella misma, y lo que haya escrito dentro del montaje no se puede creer.
func TestDesdeUnaAppImageLaActualizacionSePoneAMano(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	t.Setenv("APPIMAGE", "/home/quien-sea/Aplicaciones/Esfinge-2.65.0-x86_64.AppImage")
	if m := ComoSeInstala(); m != ModoAMano {
		t.Errorf("desde una AppImage el modo es «%s» y tiene que ser «%s»", m, ModoAMano)
	}

	// Y sin esa variable vuelve a ser el del gestor de paquetes: lo que distingue los dos
	// casos es de dónde viene la copia, no el sistema.
	t.Setenv("APPIMAGE", "")
	if m := ComoSeInstala(); m != ModoInstalador {
		t.Errorf("sin AppImage el modo es «%s» y tiene que ser «%s»", m, ModoInstalador)
	}
}

// Y lo que se abre desde una AppImage es la carpeta, no el fichero.
//
// La diferencia no es cosmética: abriendo el fichero se arranca **una segunda Esfinge** y la
// que la persona está usando sigue siendo la vieja, sin que nada lo diga. Esto solo se puede
// afirmar con la costura de `abrir`, porque lo que hay debajo es un `xdg-open` que aquí no
// se puede ejecutar.
func TestDesdeUnaAppImageSeAbreLaCarpetaYNoElFichero(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	bajado := filepath.Join(t.TempDir(), "Esfinge-2.65.0-x86_64.AppImage")
	if err := os.WriteFile(bajado, []byte("la nueva"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPIMAGE", "/home/quien-sea/Aplicaciones/Esfinge-2.64.0-x86_64.AppImage")

	var abierto string
	var veces int
	antes := abrir
	abrir = func(ruta string) error { abierto = ruta; veces++; return nil }
	t.Cleanup(func() { abrir = antes })

	if err := Instalar(bajado); err != nil {
		t.Fatal(err)
	}
	// **Se cuentan las veces, no solo se apunta el argumento**: con un doble que solo
	// guarda lo que recibe, «no me han llamado» y «me han llamado con la carpeta» no se
	// distinguen el día que la carpeta sea la cadena vacía.
	if veces != 1 {
		t.Fatalf("se ha abierto %d veces", veces)
	}
	if abierto != filepath.Dir(bajado) {
		t.Errorf("ha abierto «%s» y tenía que abrir la carpeta «%s»", abierto, filepath.Dir(bajado))
	}
	if abierto == bajado {
		t.Error("ha abierto la propia AppImage: eso arranca una segunda Esfinge")
	}
}
