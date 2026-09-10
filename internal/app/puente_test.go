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

	// La bóveda propiamente dicha. **Aquí es donde esta lista gana su sueldo**:
	// por estos métodos viajan contraseñas, y la regla es que los secretos salen
	// de uno en uno y solo cuando se piden. VerDeBoveda devuelve una entrada;
	// BuscarEnBoveda devuelve la lista sin secretos.
	"EstadoBoveda", "CrearBoveda", "AbrirBoveda", "CerrarBoveda",
	"BuscarEnBoveda", "VerDeBoveda", "GuardarEnBoveda", "BorrarDeBoveda",
	"CambiarMaestraDeBoveda", "RotarRecuperacionDeBoveda", "BorrarBoveda",
	"ImportarEnBoveda", "ExportarBoveda", "BorrarElCSVImportado",
	// La papelera. `PapeleraDeBoveda` devuelve la lista **sin secretos**, como
	// cualquier otra lista: estar borrada no hace a una entrada menos secreta.
	"PapeleraDeBoveda", "RestaurarDeBoveda", "BorrarDelTodoDeBoveda",
	"VaciarPapeleraDeBoveda",
	// Los iconos van por su propio método y no dentro de la lista de entradas: la
	// lista se vuelve a pedir en cada tecla del buscador, y meterlos ahí sería
	// mandarlos todos por el puente en cada pulsación.
	"IconosDeBoveda",
	// El código de un solo uso se calcula en Go y cruza **ya calculado**: lo que
	// sale por aquí son seis cifras que caducan en treinta segundos, no la
	// semilla, que es el segundo factor entero.
	"CodigoDeBoveda",

	// El canal con el navegador (fase 2). Lo que cruza por aquí es **el ajuste y
	// los permisos**, no los datos: lo que el navegador pregunta va por su propio
	// socket y tiene su propia lista, `navegador.LoQueSePuedePedir`. Y los
	// testigos de emparejamiento no salen: `EstadoDelNavegador` los quita antes de
	// devolver la lista, porque no pintan nada dentro del webview.
	"EstadoDelNavegador", "PermitirNavegador", "OlvidarNavegador",
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
