package app

import "testing"

// **Un guardado a medias no puede cambiar el aspecto de la ventana.**
//
// `GuardarPreferencias` recibe el objeto entero, así que un `{"buscarActualizaciones":
// true}` —que es lo que manda una prueba a propósito— llega con el tema en blanco. Si
// el blanco valiera como «el del sistema», ese descuido devolvería al aspecto del
// sistema a quien hubiera elegido el oscuro, **en silencio**. Es la regla del cero de
// los dos relojes, y aquí se cumple con la cadena vacía.
func TestElTemaSobreviveAUnGuardadoAMedias(t *testing.T) {
	antes := Preferencias{Tema: "oscuro", MinutosParaBloquear: 15}
	despues := fundir(antes, Preferencias{BuscarActualizaciones: true})
	if despues.Tema != "oscuro" {
		t.Fatalf("un guardado a medias se ha llevado el tema: %q", despues.Tema)
	}
}

// Y lo que no existe tampoco lo cambia: la ventana puede mandar cualquier cosa.
func TestUnTemaQueNoExisteNoCambiaNada(t *testing.T) {
	for _, malo := range []string{"azul", "SISTEMA", "dark", " claro"} {
		if got := fundir(Preferencias{Tema: "oscuro"}, Preferencias{Tema: malo}).Tema; got != "oscuro" {
			t.Errorf("con %q el tema ha pasado a %q", malo, got)
		}
	}
	for _, bueno := range TemasQueHay {
		if got := fundir(Preferencias{Tema: "oscuro"}, Preferencias{Tema: bueno}).Tema; got != bueno {
			t.Errorf("%q no se ha podido elegir: %q", bueno, got)
		}
	}
}

// Y sin nada guardado, el del sistema.
func TestSinNadaGuardadoElTemaEsElDelSistema(t *testing.T) {
	if got := fundir(Preferencias{}, Preferencias{}).Tema; got != "sistema" {
		t.Fatalf("de fábrica el tema es %q", got)
	}
}
