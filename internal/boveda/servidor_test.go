package boveda

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// La ranura del servidor (ADR 0057): se pone con la bóveda abierta, abre sola, se
// rota, se quita, y **no se sube nunca**. Es el espejo de
// `TestLaRanuraDelSistemaAbreYNoViaja`, que es la otra cara de la misma máquina.
func TestLaRanuraDelServidorAbreYNoViaja(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	b, _, err := Crear(ruta, "una maestra bien larga para la prueba")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Cloudflare", Secreto: "el testigo"}); err != nil {
		t.Fatal(err)
	}
	// **No se activa sola**: una bóveda recién creada no abre sin contraseña en
	// ningún sitio.
	if b.TieneRanuraDelServidor() || RanuraDelServidorEn(ruta) {
		t.Fatal("una bóveda recién creada se abre con la llave de un servidor")
	}

	llave, err := SecretoDelServidor()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.PonerRanuraDelServidor(llave); err != nil {
		t.Fatal(err)
	}
	if !b.TieneRanuraDelServidor() || !RanuraDelServidorEn(ruta) {
		t.Fatal("la ranura no ha quedado puesta")
	}

	enElServidor, err := AbrirConElServidor(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if e := enElServidor.Buscar("Cloudflare"); len(e) != 1 {
		t.Fatalf("abierta con la llave del servidor hay %d entradas", len(e))
	}
	// Y la maestra sigue abriendo: esta ranura nunca es la única.
	if _, err := Abrir(ruta, "una maestra bien larga para la prueba"); err != nil {
		t.Fatalf("la maestra ha dejado de abrir: %v", err)
	}
	// Otra llave no abre, y no se confunde con «no la tiene»: las dos cosas se
	// cuentan igual a propósito.
	otra, _ := SecretoDelServidor()
	if _, err := AbrirConElServidor(ruta, otra); err != ErrSinRanuraDelServidor {
		t.Fatalf("con otra llave el error es %v", err)
	}
	// Y la del sistema tampoco: son dos ranuras distintas y cada una mira la suya.
	if _, err := AbrirConElSistema(ruta, llave); err != ErrSinRanuraDelSistema {
		t.Fatalf("la llave del servidor abre por la ranura del sistema: %v", err)
	}

	// **Lo que no puede pasar nunca**: que esa llave salga hacia el servidor de
	// cuentas. Es de un equipo y de uno solo.
	datos, _, err := b.PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(datos), RanuraDelServidor) {
		t.Fatal("la ranura del servidor viaja en la subida: le daría al servidor una segunda puerta")
	}
	if _, err := AbrirEnMemoria(datos, "una maestra bien larga para la prueba"); err != nil {
		t.Fatalf("lo que se sube ya no abre: %v", err)
	}

	// **Rotarla es ponerla otra vez**, y entonces la de antes deja de abrir. Es lo
	// que hay que poder hacer el día que se sospeche del servidor.
	nueva, _ := SecretoDelServidor()
	if err := b.PonerRanuraDelServidor(nueva); err != nil {
		t.Fatal(err)
	}
	if _, err := AbrirConElServidor(ruta, nueva); err != nil {
		t.Fatalf("la llave nueva no abre: %v", err)
	}
	if _, err := AbrirConElServidor(ruta, llave); err != ErrSinRanuraDelServidor {
		t.Fatalf("la llave vieja sigue abriendo después de rotar: %v", err)
	}

	if err := b.QuitarRanuraDelServidor(); err != nil {
		t.Fatal(err)
	}
	if b.TieneRanuraDelServidor() || RanuraDelServidorEn(ruta) {
		t.Fatal("la ranura sigue ahí después de quitarla")
	}
	if _, err := AbrirConElServidor(ruta, nueva); err != ErrSinRanuraDelServidor {
		t.Fatal("la llave de antes sigue abriendo cuando ya no hay ranura")
	}
}

// **Con una copia de la carpeta de las bóvedas no se abre nada.**
//
// Es la regla de la ADR 0044 —la llave nunca al lado de la bóveda— comprobada por
// donde se incumpliría: se copia la carpeta entera, como haría una copia de
// seguridad o quien se lleve el disco, y lo que hay ahí dentro **no** abre.
//
// Lo que sostiene esta prueba no es el código de aquí, que no sabe dónde vive la
// llave, sino que nadie la escriba en esa carpeta: por eso se comprueba mirando
// los bytes de todo lo que haya.
func TestLaCarpetaDeLaBovedaNoLlevaLaLlaveDelServidor(t *testing.T) {
	carpeta := t.TempDir()
	ruta := filepath.Join(carpeta, "boveda.esfinge")
	b, _, err := Crear(ruta, "una maestra bien larga para la prueba")
	if err != nil {
		t.Fatal(err)
	}
	llave, _ := SecretoDelServidor()
	if err := b.PonerRanuraDelServidor(llave); err != nil {
		t.Fatal(err)
	}

	entradas, err := os.ReadDir(carpeta)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		datos, err := os.ReadFile(filepath.Join(carpeta, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		// La llave son 32 bytes al azar: si estuviera ahí, en claro o en hex.
		if contieneBytes(datos, llave) || strings.Contains(strings.ToLower(string(datos)), enHex(llave)) {
			t.Fatalf("la llave del servidor está dentro de %s, al lado de la bóveda que abre", e.Name())
		}
	}
}

func contieneBytes(donde, que []byte) bool {
	return len(que) > 0 && strings.Contains(string(donde), string(que))
}

func enHex(b []byte) string {
	const d = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, x := range b {
		out = append(out, d[x>>4], d[x&0xf])
	}
	return string(out)
}
