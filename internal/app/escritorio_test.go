//go:build !dev

package app

import (
	"strings"
	"testing"
)

// **Un filtro de fichero no es lo mismo en los tres sistemas, y confundirlo deja
// el diálogo sin dejar elegir nada.**
//
// En Windows y en GTK el filtro es una lista desplegable: proponer los `.esf`
// primero es una comodidad y no le quita a nadie la posibilidad de elegir otra
// cosa. En macOS **no**: lo que se manda es la única lista de extensiones que el
// panel deja seleccionar, y todo lo demás sale en gris.
//
// Encima Wails les quita el `*.` por delante antes de dárselos al panel, así que
// el `*.*` de «todos los ficheros» —idiomático en Windows, donde funciona—
// llegaba a macOS convertido en una extensión llamada literalmente `*`, que no
// tiene ningún fichero del mundo. El panel se abría y no se podía elegir nada.
//
// Estuvo así desde que existen los diálogos y no se vio nunca, porque en macOS
// los ficheros se arrastran a la ventana y ese camino no pasa por aquí. Lo
// encontró el cliente al ir a importar de Dashlane, que es lo primero que no
// tiene arrastrar y soltar.
func TestElFiltroDeCadaSistema(t *testing.T) {
	// macOS: ninguno. Sin filtros, Wails llama a setAllowsOtherFileTypes:true y
	// el panel acepta lo que sea, que es lo único que hace falta que haga.
	for _, f := range []Filtro{FiltroCualquiera, FiltroCifrados, FiltroTablas} {
		if got := filtrosPara("darwin", f); got != nil {
			t.Errorf("macOS con el filtro %d: %+v, y ahí no puede ir ninguno", f, got)
		}
	}

	// En los otros dos, «todos los ficheros» tiene que estar **y estar el último**,
	// que es como se lee una lista de sugerencias.
	for _, sistema := range []string{"windows", "linux"} {
		for _, f := range []Filtro{FiltroCualquiera, FiltroCifrados, FiltroTablas} {
			filtros := filtrosPara(sistema, f)
			if len(filtros) == 0 {
				t.Fatalf("%s con el filtro %d se ha quedado sin ninguno", sistema, f)
			}
			ultimo := filtros[len(filtros)-1]
			if !strings.Contains(ultimo.DisplayName, "Todos") {
				t.Errorf("%s con el filtro %d acaba en %q y no en «todos los ficheros»",
					sistema, f, ultimo.DisplayName)
			}
		}
	}

	// Y el patrón de «todos» tampoco es el mismo: `*.*` es lo idiomático en
	// Windows, y en GTK deja fuera los ficheros sin extensión.
	if p := filtrosPara("windows", FiltroCualquiera)[0].Pattern; p != "*.*" {
		t.Errorf("Windows quiere *.* y tiene %q", p)
	}
	if p := filtrosPara("linux", FiltroCualquiera)[0].Pattern; p != "*" {
		t.Errorf("GTK quiere * y tiene %q", p)
	}

	// Lo que se propone primero, donde se puede proponer algo.
	if p := filtrosPara("linux", FiltroCifrados)[0].Pattern; p != "*.esf" {
		t.Errorf("al descifrar se proponen los .esf y se propone %q", p)
	}
	if p := filtrosPara("linux", FiltroTablas)[0].Pattern; p != "*.csv" {
		t.Errorf("al importar se proponen los .csv y se propone %q", p)
	}
}

// La otra mitad de la lección: **un filtro nunca puede impedir elegir**, ni
// siquiera donde hay lista desplegable.
//
// Un `.esf` puede ser un fichero cifrado o un `.txt` en el que alguien guardó la
// línea `ESF1.…` que salió de cifrar un texto —lo hace «Guardar como…», y Esfinge
// lo abre igual—. Y una exportación de contraseñas llega con la extensión que le
// dé la gana al gestor que la escribió.
func TestNingunFiltroSeQuedaSoloConLoQueEsperabamos(t *testing.T) {
	for _, sistema := range []string{"windows", "linux"} {
		for _, f := range []Filtro{FiltroCifrados, FiltroTablas} {
			var abierto bool
			for _, filtro := range filtrosPara(sistema, f) {
				if filtro.Pattern == "*" || filtro.Pattern == "*.*" {
					abierto = true
				}
			}
			if !abierto {
				t.Errorf("%s con el filtro %d solo deja elegir lo que esperábamos", sistema, f)
			}
		}
	}
}
