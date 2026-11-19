package entorno

import (
	"runtime"
	"testing"
)

// Las cadenas de verdad de cada escritorio, que es lo que esto tiene que
// entender. No son inventadas: son las que publican Ubuntu, Fedora KDE, Budgie,
// Cinnamon y una sesión pelada.
func TestQueEscritorioEsCadaLista(t *testing.T) {
	casos := []struct {
		lista    string
		esperado string
	}{
		// Lo que de verdad hay en las máquinas del cliente.
		{"ubuntu:GNOME", GNOME},
		{"GNOME", GNOME},
		{"KDE", KDE},
		{"plasma:KDE", KDE},

		// Mayúsculas y espacios: la lista la escribe cada distribución.
		{"gnome", GNOME},
		{"kde", KDE},
		{" KDE ", KDE},

		// Lo que se ve como GNOME a propósito.
		{"X-Cinnamon", Otro},
		{"XFCE", Otro},
		{"COSMIC", Otro},
		{"", Otro},

		// Budgie se declara GNOME, y lo es: lo dibuja GTK.
		{"Budgie:GNOME", GNOME},

		// **Y el caso que decide el orden de las preguntas**: una sesión que se
		// declara de los dos es KDE, porque quien dibuja la ventana es KWin.
		{"KDE:GNOME", KDE},
		{"GNOME:KDE", KDE},

		// Un nombre que contiene otro no cuenta: se compara trozo a trozo, no por
		// subcadena. Sin eso, «GNOME-Flashback» o «kdeconnect» acertarían por
		// casualidad y algún día una lista nueva fallaría sin que se entienda.
		{"GNOMEISH", Otro},
		{"notkde", Otro},
	}

	for _, c := range casos {
		if hay := Cual(c.lista); hay != c.esperado {
			t.Errorf("con «%s» el escritorio es «%s» y tenía que ser «%s»", c.lista, hay, c.esperado)
		}
	}
}

// Fuera de Linux no se pregunta nada, y eso se dice con el vacío.
//
// Devolver «otro» en un Mac sería un dato que parece significar algo —e invita a
// que alguien cuelgue CSS de él— cuando ahí el escritorio no existe como
// pregunta.
func TestFueraDeLinuxNoHayEscritorio(t *testing.T) {
	// En esta máquina (Linux) tiene que contestar algo de la lista; en macOS y en
	// Windows, vacío. La prueba vale en los tres porque mira lo que toca según
	// dónde corre, que es lo mismo que hace la función.
	hay := DeAqui()
	if runtime.GOOS == "linux" {
		if hay != GNOME && hay != KDE && hay != Otro {
			t.Errorf("en Linux el escritorio es «%s», que no es ninguno de los tres", hay)
		}
		return
	}
	if hay != "" {
		t.Errorf("en %s el escritorio tendría que venir vacío y viene «%s»", runtime.GOOS, hay)
	}
}
