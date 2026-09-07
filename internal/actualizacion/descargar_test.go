package actualizacion

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// servidorDeFicheros sirve un contenido cualquiera como si fuera el instalador.
func servidorDeFicheros(t *testing.T, contenido []byte) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(contenido)
	}))
	t.Cleanup(s.Close)
	return s
}

// enCarpetaDePruebas aparta la caché real: los tests no dejan ficheros en la del
// usuario ni se pisan entre ellos.
func enCarpetaDePruebas(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir) // Linux
	t.Setenv("HOME", dir)           // macOS
	t.Setenv("LocalAppData", dir)   // Windows
	return dir
}

func TestDescargarComprobandoElResumen(t *testing.T) {
	enCarpetaDePruebas(t)

	contenido := []byte("esto hace de instalador")
	suma := sha256.Sum256(contenido)
	s := servidorDeFicheros(t, contenido)

	c := Nuevo("2.0.3")
	n := Novedad{
		Hay: true, Version: "2.1.0",
		Fichero: "Esfinge-2.1.0.dmg", URL: s.URL,
		Bytes: int64(len(contenido)), Resumen: hex.EncodeToString(suma[:]),
	}

	var ultimo Avance
	ruta, err := c.Descargar(n, func(a Avance) { ultimo = a })
	if err != nil {
		t.Fatalf("descargar: %v", err)
	}

	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer lo descargado: %v", err)
	}
	if string(datos) != string(contenido) {
		t.Errorf("contenido:\nquiero %q\ntengo  %q", contenido, datos)
	}
	if filepath.Base(ruta) != n.Fichero {
		t.Errorf("nombre: quiero %q, tengo %q", n.Fichero, filepath.Base(ruta))
	}
	if !ultimo.Hecho {
		t.Error("el último aviso tiene que decir que ha terminado")
	}
}

// Lo que de verdad importa de todo esto: si lo que llega no es lo publicado, no
// se queda en disco. Un instalador a medias que alguien abre luego es peor que
// no haberlo descargado.
func TestUnaDescargaQueNoCuadraSeBorra(t *testing.T) {
	enCarpetaDePruebas(t)

	s := servidorDeFicheros(t, []byte("esto es otra cosa"))
	c := Nuevo("2.0.3")
	n := Novedad{
		Hay: true, Version: "2.1.0",
		Fichero: "Esfinge-2.1.0.dmg", URL: s.URL,
		Resumen: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	if _, err := c.Descargar(n, nil); err == nil {
		t.Fatal("el resumen no cuadra y lo ha dado por bueno")
	}

	carpeta, err := Carpeta()
	if err != nil {
		t.Fatalf("carpeta: %v", err)
	}
	entradas, err := os.ReadDir(carpeta)
	if err != nil {
		t.Fatalf("leer la carpeta: %v", err)
	}
	if len(entradas) != 0 {
		t.Errorf("ha dejado %d ficheros donde no debería quedar ninguno", len(entradas))
	}
}

func TestSinFicheroParaEsteSistemaNoDescargaNada(t *testing.T) {
	enCarpetaDePruebas(t)

	c := Nuevo("2.0.3")
	if _, err := c.Descargar(Novedad{Hay: true, Version: "2.1.0"}, nil); err == nil {
		t.Error("sin URL no hay nada que descargar, y no ha protestado")
	}
}

// La carpeta de descargas no es un almacén: cada versión nueva se lleva por
// delante lo anterior, que son megas de algo que ya no sirve.
func TestLaDescargaAnteriorSeTira(t *testing.T) {
	enCarpetaDePruebas(t)

	carpeta, err := Carpeta()
	if err != nil {
		t.Fatalf("carpeta: %v", err)
	}
	viejo := filepath.Join(carpeta, "Esfinge-2.0.0.dmg")
	if err := os.WriteFile(viejo, []byte("una versión de hace tiempo"), 0o600); err != nil {
		t.Fatalf("preparar: %v", err)
	}

	contenido := []byte("la nueva")
	suma := sha256.Sum256(contenido)
	s := servidorDeFicheros(t, contenido)

	c := Nuevo("2.0.3")
	_, err = c.Descargar(Novedad{
		Hay: true, Fichero: "Esfinge-2.1.0.dmg", URL: s.URL,
		Resumen: hex.EncodeToString(suma[:]),
	}, nil)
	if err != nil {
		t.Fatalf("descargar: %v", err)
	}

	if _, err := os.Stat(viejo); !os.IsNotExist(err) {
		t.Error("la descarga anterior sigue ahí")
	}
}
