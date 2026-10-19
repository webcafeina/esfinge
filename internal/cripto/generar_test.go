package cripto

import (
	"strings"
	"testing"
)

// **Una contraseña de un largo pedido sale de ese largo exacto**, con los tres alfabetos
// y en todo el rango que se acepta.
//
// Un carácter de menos no se nota al mirarla y es una contraseña más débil de la que
// alguien cree tener; uno de más lo rechaza el formulario que puso el límite.
func TestGenerarDeLargoDaElLargoPedido(t *testing.T) {
	for _, a := range []Alfabeto{AlfHex, AlfAlnum, AlfSimbolos} {
		min, max := MinimoDeCaracteres(a), MaximoDeCaracteres(a)
		// Los extremos y unos cuantos de en medio, incluidos impares: con hexadecimal
		// salen dos caracteres por byte, así que un largo impar es el que obliga a
		// recortar.
		for _, n := range []int{min, min + 1, 20, 21, 32, 33, 64, max - 1, max} {
			if n < min || n > max {
				continue
			}
			clave, err := GenerarDeLargo(a, n)
			if err != nil {
				t.Fatalf("%s con %d: %v", a.Nombre, n, err)
			}
			if got := len([]rune(clave)); got != n {
				t.Errorf("%s: pedidos %d caracteres y salen %d", a.Nombre, n, got)
			}
			// Y todo lo que sale es del alfabeto, que es lo que hace que el sitio la acepte.
			if a.Runas != "" {
				for _, r := range clave {
					if !strings.ContainsRune(a.Runas, r) {
						t.Fatalf("%s: ha salido %q, que no es del alfabeto", a.Nombre, r)
					}
				}
			}
		}
	}
}

// **Y por debajo del suelo de entropía dice que no, en vez de dar algo flojo.**
//
// El mínimo no es un número redondo: son los 64 bits de `Generar` traducidos a caracteres
// de cada alfabeto, que es por lo que en hexadecimal son 16 y con símbolos 10. Un gestor
// de contraseñas que acepte «dame una de ocho» y la dé no está protegiendo a nadie.
func TestGenerarDeLargoNoBajaDelSueloDeEntropia(t *testing.T) {
	for _, a := range []Alfabeto{AlfHex, AlfAlnum, AlfSimbolos} {
		min := MinimoDeCaracteres(a)
		if _, err := GenerarDeLargo(a, min-1); err == nil {
			t.Errorf("%s: ha aceptado %d caracteres, uno por debajo del mínimo", a.Nombre, min-1)
		}
		if _, err := GenerarDeLargo(a, MaximoDeCaracteres(a)+1); err == nil {
			t.Errorf("%s: ha aceptado pasarse del máximo", a.Nombre)
		}
		// Y el mínimo de verdad son 64 bits, dicho por el otro camino.
		if b := BytesParaCaracteres(a, min); b < 8 {
			t.Errorf("%s: el mínimo de %d caracteres son %d bytes, menos de 8", a.Nombre, min, b)
		}
	}
}

// **Y el recorte no sesga**: cada carácter sigue saliendo de un sorteo uniforme.
//
// Si recortar inclinara el resultado, se vería en que unos caracteres del alfabeto salen
// mucho más que otros. Con 62 caracteres y 20.000 sorteos, lo esperado son ~320 de cada
// uno; se deja un margen amplio porque esto es azar de verdad y no un generador fijado:
// lo que se busca es un sesgo gordo, no ruido.
func TestElRecorteNoSesgaElAlfabeto(t *testing.T) {
	cuenta := map[rune]int{}
	total := 0
	for i := 0; i < 1000; i++ {
		clave, err := GenerarDeLargo(AlfAlnum, 20)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range clave {
			cuenta[r]++
			total++
		}
	}
	esperado := float64(total) / float64(len([]rune(AlfAlnum.Runas)))
	for _, r := range AlfAlnum.Runas {
		if v := float64(cuenta[r]); v < esperado*0.5 || v > esperado*1.5 {
			t.Errorf("%q sale %v veces y lo esperado era ~%.0f", r, v, esperado)
		}
	}
}
