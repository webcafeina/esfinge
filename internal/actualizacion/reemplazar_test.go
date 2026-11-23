package actualizacion

import (
	"regexp"
	"strings"
	"testing"
)

// **El guion de Windows no puede llamar a nada por su nombre a secas.**
//
// Esta prueba existe por un fallo que llegó a los equipos de los clientes: el
// guion filtraba la salida de `tasklist` con `find`, y en un Windows con Git
// instalado el `find` que gana en el PATH es el de GNU. Ése no filtra: busca un
// directorio con ese nombre, falla, y el `||` salta a instalar **en la primera
// vuelta** — con Esfinge viva y sus ficheros bloqueados. La ventana se cerraba,
// el instalador no podía reemplazar nada y la aplicación no volvía a abrirse.
//
// Aquí no hay ningún Windows donde ejecutarlo, así que **lo único que se puede
// comprobar es lo que el guion dice**. Lo que lo cierra es fijar el PATH, y lo
// que esta prueba vigila es que siga fijado y antes de la primera llamada.
func TestElGuionDeWindowsNoSeFiaDelPATH(t *testing.T) {
	g := GuionDeWindows(`C:\Temp\Esfinge-instalador.exe`, `C:\Program Files\Esfinge\Esfinge.exe`, 2228)

	fija := regexp.MustCompile(`(?mi)^\s*set\s+"PATH=%SystemRoot%\\System32;`)
	sitio := fija.FindStringIndex(g)
	if sitio == nil {
		t.Fatal("el guion no fija el PATH al del sistema, así que un «find» de GNU en el PATH lo rompe")
	}

	// Y **antes de la primera llamada**: fijarlo después no protege a nadie.
	//
	// Se mira sobre el guion **sin sus comentarios**, y eso no es un detalle: la
	// primera versión de esta guarda buscaba las órdenes en el texto entero y se
	// encontraba con el comentario que explica justo esto —que nombra las tres—,
	// así que daba por roto un guion correcto. Es la misma trampa que la guarda de
	// `rellenar`, que preguntaba si el marcador aparecía en el fichero y aparecía en
	// el comentario de al lado: **una guarda pregunta por lo que se ejecuta.**
	sinComentarios := g
	for _, linea := range strings.Split(g, "\n") {
		if strings.HasPrefix(strings.TrimSpace(strings.ToLower(linea)), "rem") {
			sinComentarios = strings.Replace(sinComentarios, linea, "", 1)
		}
	}
	donde := regexp.MustCompile(`(?mi)^\s*set\s+"PATH=`).FindStringIndex(sinComentarios)
	if donde == nil {
		t.Fatal("el PATH se fija sólo dentro de un comentario")
	}
	for _, orden := range []string{"tasklist", "find ", "timeout "} {
		if i := strings.Index(sinComentarios, orden); i >= 0 && i < donde[0] {
			t.Errorf("%q se llama antes de fijar el PATH", strings.TrimSpace(orden))
		}
	}
}

// **Y vuelve a abrir la aplicación pase lo que pase.**
//
// Es la promesa que encabeza este fichero —«lo que no puede pasar nunca es
// quedarse sin aplicación»— y la que el fallo de arriba rompió. Si el instalador
// no ha podido hacer su trabajo, en disco sigue estando la versión de antes, que
// funciona: lo que no vale es condicionar el arranque al éxito de la instalación.
func TestElGuionDeWindowsVuelveAAbrirAunqueFalleElInstalador(t *testing.T) {
	g := GuionDeWindows(`C:\Temp\inst.exe`, `C:\Archivos\Esfinge.exe`, 7)

	instala := strings.Index(g, `"C:\Temp\inst.exe" /S`)
	abre := strings.Index(g, `start "" "C:\Archivos\Esfinge.exe"`)
	if instala < 0 {
		t.Fatal("el guion no lanza el instalador en silencio")
	}
	if abre < 0 {
		t.Fatal("el guion no vuelve a abrir Esfinge")
	}
	if abre < instala {
		t.Error("abre antes de instalar")
	}

	// Sin condición de por medio: ni un «if errorlevel», ni un «||», ni un salto.
	entre := g[instala:abre]
	for _, condicion := range []string{"errorlevel", "||", "goto"} {
		if strings.Contains(strings.ToLower(entre), condicion) {
			t.Errorf("entre instalar y abrir hay un %q: si el instalador falla, nadie recupera la aplicación", condicion)
		}
	}
}

// El guion se escribe entero con lo que se le pasa, sin dejar ningún verbo de
// formato sin sustituir. Una errata ahí da un guion que no hace nada y que nadie
// vería hasta estar publicado.
func TestElGuionDeWindowsNoDejaNadaSinRellenar(t *testing.T) {
	g := GuionDeWindows(`C:\i.exe`, `C:\a.exe`, 99)
	for _, resto := range []string{"%!", "%[1]", "%[2]", "%[3]", "MISSING", "EXTRA"} {
		if strings.Contains(g, resto) {
			t.Errorf("el guion sale con %q dentro:\n%s", resto, g)
		}
	}
	for _, loSuyo := range []string{`C:\i.exe`, `C:\a.exe`, "PID eq 99"} {
		if !strings.Contains(g, loSuyo) {
			t.Errorf("el guion no lleva %q", loSuyo)
		}
	}
}
