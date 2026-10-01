package wifi

import "testing"

// Tabla del formato, que es donde están todos los fallos que un móvil no perdona.
func TestElEnlaceDeUnaRed(t *testing.T) {
	casos := []struct {
		nombre    string
		ssid      string
		clave     string
		seguridad string
		oculta    bool
		quiere    string
	}{
		{
			nombre: "lo normal",
			ssid:   "WEBCAFEINA", clave: "secreta", seguridad: "wpa",
			quiere: "WIFI:T:WPA;S:WEBCAFEINA;P:secreta;;",
		},
		{
			// El caso del cliente: Dashlane dice `unsecured` y la red tiene clave.
			nombre: "con clave, lo que diga el fichero no manda",
			ssid:   "WEBCAFEINA", clave: "secreta", seguridad: "unsecured",
			quiere: "WIFI:T:WPA;S:WEBCAFEINA;P:secreta;;",
		},
		{
			nombre: "sin clave es abierta de verdad",
			ssid:   "Invitados", seguridad: "wpa2",
			quiere: "WIFI:T:nopass;S:Invitados;;",
		},
		{
			// Y una clave WEP de diez dígitos es **el caso clásico del hexadecimal**: va
			// entre comillas o el móvil la toma por cinco bytes en vez de por diez
			// caracteres. La primera versión de esta prueba lo esperaba sin comillas y
			// fue el código el que tenía razón.
			nombre: "wep se respeta, y su clave hexadecimal va entre comillas",
			ssid:   "Vieja", clave: "1234567890", seguridad: "WEP",
			quiere: `WIFI:T:WEP;S:Vieja;P:"1234567890";;`,
		},
		{
			nombre: "oculta lo dice",
			ssid:   "Escondida", clave: "x", seguridad: "wpa", oculta: true,
			quiere: "WIFI:T:WPA;S:Escondida;P:x;H:true;;",
		},
		{
			// **Sin comillas, `CAFE` se lee como los bytes 0xCA 0xFE** y el móvil busca
			// una red que no existe.
			nombre: "un nombre hexadecimal va entre comillas",
			ssid:   "CAFE", clave: "x", seguridad: "wpa",
			quiere: `WIFI:T:WPA;S:"CAFE";P:x;;`,
		},
		{
			nombre: "y una clave hexadecimal también",
			ssid:   "Casa", clave: "1234abcd", seguridad: "wpa",
			quiere: `WIFI:T:WPA;S:Casa;P:"1234abcd";;`,
		},
		{
			nombre: "un nombre casi hexadecimal no lleva comillas",
			ssid:   "CAFEs", clave: "x", seguridad: "wpa",
			quiere: "WIFI:T:WPA;S:CAFEs;P:x;;",
		},
		{
			// Los cinco reservados, y la barra **primera**: escapándola al final se
			// escaparían las barras que acaba de poner la propia función.
			nombre: "los cinco caracteres reservados se escapan",
			ssid:   `a;b,c:d"e\f`, clave: `p;q`, seguridad: "wpa",
			quiere: `WIFI:T:WPA;S:a\;b\,c\:d\"e\\f;P:p\;q;;`,
		},
		{
			nombre: "con tildes y espacios, tal cual",
			ssid:   "Café de la Esquina", clave: "ñandú", seguridad: "wpa",
			quiere: "WIFI:T:WPA;S:Café de la Esquina;P:ñandú;;",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			e, err := Enlace(c.ssid, c.clave, c.seguridad, c.oculta)
			if err != nil {
				t.Fatal(err)
			}
			if e != c.quiere {
				t.Errorf("sale  %q\nquiere %q", e, c.quiere)
			}
		})
	}
}

func TestUnaRedSinNombreNoTieneCodigo(t *testing.T) {
	for _, ssid := range []string{"", "   ", "\t"} {
		if _, err := Enlace(ssid, "clave", "wpa", false); err == nil {
			t.Errorf("con el nombre %q tendría que fallar y no dibujar nada", ssid)
		}
	}
}

// La normalización aparte, porque es lo que usa el importador y lo que arregla el dato
// de Dashlane antes de guardarlo.
func TestNormalizarLaSeguridad(t *testing.T) {
	casos := []struct {
		dicho    string
		hayClave bool
		quiere   string
	}{
		{"unsecured", true, WPA},
		{"", true, WPA},
		{"WPA2", true, WPA},
		{"wpa3", true, WPA},
		{"WPA2-Personal", true, WPA},
		{"wep", true, WEP},
		{"  WEP  ", true, WEP},
		{"unsecured", false, Abierta},
		{"wpa2", false, Abierta},
		{"", false, Abierta},
		// Sin clave manda la ausencia de clave, aunque diga WEP.
		{"wep", false, Abierta},
	}
	for _, c := range casos {
		if r := Normalizar(c.dicho, c.hayClave); r != c.quiere {
			t.Errorf("Normalizar(%q, %v) = %q, quiere %q", c.dicho, c.hayClave, r, c.quiere)
		}
	}
}
