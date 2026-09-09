package app

import (
	"reflect"
	"sort"
	"testing"
)

// **Todo método exportado de *App queda expuesto a la interfaz.** Wails los
// enlaza con Bind y el servidor de desarrollo los publica por reflexión, sin
// listas que mantener — que es cómodo hasta que se exporta algo que no debería
// poder pedirse desde la ventana.
//
// `CLAUDE.md` le dedica un párrafo entero a ese miedo desde hace versiones, y
// hasta ahora la única defensa era acordarse. Con la bóveda dentro eso pasa de
// incómodo a grave: un método exportado de más puede ser la clave maestra
// saliendo por el puente.
//
// Esta lista es la puerta. Añadir algo aquí tiene que ser una decisión, no un
// efecto colateral de haber puesto una mayúscula.
var loQuePuedeCruzarElPuente = []string{
	// Lo que la ventana necesita saber de sí misma.
	"Plataforma", "Version", "Vidrio", "Arrancar",

	// Cifrar y descifrar.
	"CifrarTexto", "DescifrarTexto", "CifrarFicheros", "DescifrarFicheros",
	"ElegirFicheros", "ElegirCifrados", "GuardarTexto",

	// Generar contraseñas.
	"GenerarContrasena", "EvaluarClave", "Alfabetos",
	"MedirPorCaracteres",

	// Historial y ajustes.
	"VerHistorial", "VaciarHistorial", "DondeVive",
	"VerPreferencias", "GuardarPreferencias",

	// Actualizaciones.
	"NovedadPendiente", "ComprobarActualizacion",
	"DescargarActualizacion", "InstalarActualizacion",

	// Ficheros que manda el sistema, y órdenes del menú.
	"AlAbrirCon", "AperturaDeArranque", "Ordenar", "OrdenarPegar",

	// La bóveda: el portapapeles con borrado y el aviso de que hay alguien ahí.
	"Copiar", "Actividad",
}

func TestLoQueCruzaElPuenteEstaEnLaLista(t *testing.T) {
	permitidos := map[string]bool{}
	for _, n := range loQuePuedeCruzarElPuente {
		permitidos[n] = true
	}

	tipo := reflect.TypeOf(&App{})
	var sobran, faltan []string

	presentes := map[string]bool{}
	for i := 0; i < tipo.NumMethod(); i++ {
		nombre := tipo.Method(i).Name
		presentes[nombre] = true
		if !permitidos[nombre] {
			sobran = append(sobran, nombre)
		}
	}
	for n := range permitidos {
		if !presentes[n] {
			faltan = append(faltan, n)
		}
	}
	sort.Strings(sobran)
	sort.Strings(faltan)

	if len(sobran) > 0 {
		t.Errorf("estos métodos cruzan el puente y no están en la lista: %v\n"+
			"Si tienen que poder pedirse desde la ventana, añádelos a la lista a conciencia. "+
			"Si no, ponlos en minúscula o sácalos a una función suelta, como ApuntarAAPI.", sobran)
	}
	if len(faltan) > 0 {
		t.Errorf("la lista nombra métodos que ya no existen: %v", faltan)
	}
}
