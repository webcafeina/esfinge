package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/qr"
	"github.com/webcafeina/esfinge/internal/wifi"
)

// **Cuando la búsqueda encaja con varias, no se adivina.** Es la regla que
// separa una herramienta de tubería de una trampa: sacar la contraseña
// equivocada por la salida estándar es peor que no sacar ninguna, porque el
// fallo aparece al otro lado y sin nada que lo explique.
//
// Vale para «ver» y para «codigo», que salen las dos de aquí.
func TestLaBusquedaDeLaLineaDeComandosNoAdivina(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	b, _, err := boveda.Crear(ruta, "una contraseña maestra larga")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []boveda.Entrada{
		{Titulo: "Banco del norte", Secreto: "una"},
		{Titulo: "Banco del sur", Secreto: "otra"},
		{Titulo: "Correo", Secreto: "tercera", TOTP: "GEZDGNBVGY3TQOJQ"},
	} {
		if err := b.Poner(e); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := unaSola(b, "banco"); err == nil {
		t.Error("con dos candidatas ha elegido una")
	} else if !strings.Contains(err.Error(), "afina la búsqueda") {
		t.Errorf("no dice qué hacer: %v", err)
	}

	if _, err := unaSola(b, "esto no está"); err == nil {
		t.Error("ha encontrado algo que no existe")
	}

	// Y con una sola, la entrada **entera**: la búsqueda devuelve la lista sin
	// secretos, así que si no se pidiera después la entrada por su identificador
	// esto sacaría una contraseña vacía por la tubería sin decir nada.
	e, err := unaSola(b, "correo")
	if err != nil {
		t.Fatal(err)
	}
	if e.Secreto != "tercera" || e.TOTP == "" {
		t.Errorf("la entrada llega sin sus secretos: %+v", e)
	}
}

// **El código de una red se dibuja desde la línea de comandos** (ADR 0049), y lo que se
// comprueba aquí es lo que puede fallar sin que se note: que la cadena que va dentro del
// código sea la de esa red, y que una entrada que no es una red lo diga en vez de
// dibujar cualquier cosa.
//
// Que el dibujo se pueda escanear no lo dice esta prueba ni ninguna de aquí: eso lo
// cerró un móvil contra `internal/qr`.
func TestElCodigoDeUnaRedSaleDeSusCampos(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	b, _, err := boveda.Crear(ruta, "una contraseña maestra larga")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []boveda.Entrada{
		{Titulo: "La oficina", Tipo: boveda.TipoWifi, SSID: "WEBCAFEINA",
			Secreto: "la-clave", Seguridad: "wpa", Oculta: true},
		{Titulo: "Un banco", Secreto: "nada que ver"},
	} {
		if err := b.Poner(e); err != nil {
			t.Fatal(err)
		}
	}

	red, err := unaSola(b, "oficina")
	if err != nil {
		t.Fatal(err)
	}
	enlace, err := wifi.Enlace(red.SSID, red.Secreto, red.Seguridad, red.Oculta)
	if err != nil {
		t.Fatal(err)
	}
	// Lo que va dentro del código, con la red oculta dicha: sin esto el móvil no la
	// busca, porque una red oculta no sale en su lista.
	if enlace != "WIFI:T:WPA;S:WEBCAFEINA;P:la-clave;H:true;;" {
		t.Errorf("el código llevaría %q", enlace)
	}
	c, err := qr.Nuevo(enlace)
	if err != nil {
		t.Fatal(err)
	}
	if c.Lado() < 21 || len(c.Filas()) != c.Lado() {
		t.Errorf("el código mide %d y trae %d filas", c.Lado(), len(c.Filas()))
	}

	// Y una entrada que no es una red no tiene código: el comando lo dice en vez de
	// dibujar el de una contraseña suelta, que sería un QR con un secreto dentro y sin
	// nada que lo explique.
	otra, err := unaSola(b, "banco")
	if err != nil {
		t.Fatal(err)
	}
	if otra.Tipo == boveda.TipoWifi {
		t.Error("una credencial no puede ser una red")
	}
}
