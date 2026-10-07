package app

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
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
	// **Cruza una clave, y por eso está dicho aquí**: la del fichero cifrado de las
	// llaves de acceso (ADR 0048), que no es la maestra y se pide aparte. Lo que
	// **no** cruza en ningún sentido es la clave privada de una llave: sale
	// cifrada dentro del fichero y nunca por el puente.
	"ExportarLlaves",
	// **Devuelve la contraseña de una red, aunque no lo parezca** (ADR 0049): un código
	// QR es su clave escrita de otra forma, y quien reciba esa matriz puede sacarla sin
	// pedir nada más. Está aquí porque es exactamente lo que tiene que hacer —la ficha
	// lo enseña a propósito, para que un invitado se conecte sin que nadie dicte nada—,
	// y conviene que quien lea esta lista lo sepa: no es un dato de dibujo.
	"CodigoDeWifi",
	// La papelera. `PapeleraDeBoveda` devuelve la lista **sin secretos**, como
	// cualquier otra lista: estar borrada no hace a una entrada menos secreta.
	"PapeleraDeBoveda", "RestaurarDeBoveda", "BorrarDelTodoDeBoveda",
	"VaciarPapeleraDeBoveda",
	// Los sitios en los que la extensión no ofrece guardar (ADR 0032): una lista de
	// dominios, sin secretos, y quitar uno.
	"SitiosExcluidos", "QuitarSitioExcluido",
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

	// El canal con los agentes de IA (ADR 0054). **Lo mismo y por lo mismo**: por
	// aquí cruzan el ajuste, los permisos y el bloque de configuración que hay que
	// pegarle al cliente; lo que el agente pregunta va por su propio socket y tiene
	// su propia lista, `agente.LoQueSePuedePedir`. Los testigos tampoco salen:
	// `EstadoDelAgente` los vacía, y por eso retirar uno se hace **por su fecha**.
	"EstadoDelAgente", "PermitirAgente", "OlvidarAgente",
	// Y lo que un agente pide y hay que contestar (ADR 0054). **El sí y el no son de
	// la ventana y de ningún otro sitio**: es lo único que hay entre un agente al que
	// alguien le ha dicho qué pedir y la bóveda. `RegistroDelAgente` es de lectura y
	// **no se le da al agente**: existe para que lo lea la persona.
	"AprobarLoQuePideElAgente", "DenegarLoQuePideElAgente", "RegistroDelAgente",

	// La cuenta (ADR 0035). Por aquí viaja **la contraseña maestra hacia Go**, igual
	// que al abrir la bóveda, y nunca de vuelta: lo que sale es el estado, sin
	// sesión ni testigos, y la clave de recuperación una sola vez al crear la
	// cuenta con una bóveda nueva. A qué servidor se habla lo decide
	// `ApuntarCuentasA`, que es función y no método.
	"EstadoDeCuenta", "ElegirModoLocal", "SincronizarAhora", "SincronizarAunqueBorre",
	"EmpezarRegistro", "TerminarRegistro",
	"EntrarEnCuenta", "ConfirmarEntrada", "ResolverOtraBoveda", "SalirDeCuenta",
	// La A3. Recuperar recibe la clave de recuperación y la contraseña nueva, y no
	// devuelve ninguna de las dos; los equipos salen sin sesiones ni testigos, y la
	// exportación va a un fichero, no por el puente.
	"EmpezarRecuperacion", "TerminarRecuperacion",
	"DispositivosDeCuenta", "OlvidarDispositivo",
	"PedirCodigoParaBorrarCuenta", "BorrarCuenta", "ExportarDatosDeCuenta",
	"RepetidasEnBoveda", "QuitarRepetidasDeBoveda",

	// La B: compartir copias (ADR 0043). **Lo que cruza de la identidad es su
	// huella, nunca la semilla ni las llaves**, y del buzón, de quién viene y qué
	// es; el secreto solo entra en la bóveda al aceptarlo. Mandar recibe el
	// identificador de una entrada y un correo, y devuelve lo que devuelve el
	// servidor: nada.
	"MiIdentidad", "HuellaDe", "MandarCopia", "EnviosPendientes", "Buzon", "AceptarDelBuzon", "TirarDelBuzon",

	// La C: desbloquear con el sistema, Touch ID o Windows Hello
	// (`docs/desbloqueo-del-sistema.md`). El que hay que mirar dos veces es
	// **`AbrirBovedaConElSistema`: abre la bóveda sin la contraseña maestra**, y
	// por eso está aquí a conciencia y no de paso. Lo que lo sostiene es que el
	// secreto no está en Esfinge sino en el llavero del sistema, y que para
	// sacarlo de ahí el sistema pide la huella. Ninguno de los cuatro recibe ni
	// devuelve el secreto: entra y sale de `internal/llavero` sin cruzar nada.
	"EstadoDelDesbloqueo", "ActivarDesbloqueo", "QuitarDesbloqueo", "AbrirBovedaConElSistema",
	"NoOfrecerElDesbloqueo",

	// Las bóvedas de proyecto (ADR 0050). El que hay que mirar dos veces es
	// **`AbrirProyecto`: abre una bóveda sin la contraseña maestra**, con la clave
	// de la personal. Está aquí a conciencia, y lo que lo sostiene es que esa clave
	// solo existe en memoria **porque la personal ya se abrió con su maestra en esta
	// sesión**, y que el reloj del bloqueo la borra. No hay camino desde la ventana
	// para conseguirla: ninguno de los cinco recibe ni devuelve ninguna clave, y
	// `LlaveParaProyectos` no es método de `App`.
	//
	// Lo que **no** está aquí, y no por olvido: no hay forma de pedir desde la
	// ventana la clave de un proyecto, ni de crear uno con una contraseña elegida, ni
	// de poner la ranura del sistema en uno.
	"Proyectos", "CrearProyecto", "AbrirProyecto", "VolverALaBovedaPersonal", "RenombrarProyecto",
	"LlevarAOtraBoveda", "BajarProyecto",
	"EntregarProyecto", "ArchivarProyecto", "BorrarProyecto",

	// Las bóvedas a las que me han dado acceso (ADR 0052). Abrirla sí se pide desde
	// la ventana, como cualquier otra. Lo que **no** está, y tampoco por olvido: nada
	// que selle una ranura hacia una identidad ni que lea la clave de una bóveda
	// ajena. Dar y quitar el acceso llegarán con su propia entrega, y entonces habrá
	// que volver a pensar esta línea, no ampliarla de paso.
	"AbrirCompartida", "BajarCompartida",
	"DarAcceso", "QuitarAcceso", "QuienTiene", "AceptarAcceso",
	"Compartidas", "DejarDeVerCompartida",
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

// **Y lo que de verdad cierra la otra mitad: un método del puente que no llama nadie.**
//
// La lista de arriba vigila que no cruce el puente lo que no debe. Esto vigila lo
// contrario, que ha costado dos veces lo mismo:
//
//   - `VolverALaBovedaPersonal` (2026-10-02) existía en Go y en el puente, y la única
//     forma de salir de un proyecto era bloquear la bóveda y volver a desbloquear.
//   - `AceptarAcceso` (2026-10-05) existía en Go y en el puente, y un acceso que llegaba
//     al buzón se enseñaba como un sobre roto que solo se podía descartar.
//
// Las dos veces el código estaba escrito, probado y **muerto**, y las dos veces lo
// encontró alguien recorriendo el camino a mano. No es un olvido que se arregle
// acordándose: es que **escribir el puente parece terminar el trabajo**.
//
// Lo que esto **no** puede decir, y por eso no basta solo: que lo que se llama se llame
// en el sitio bueno, ni que la pantalla haga algo útil con ello. Para eso están las
// pruebas de la interfaz y la de la tubería entre dos cuentas.
func TestLoQueEstaEnElPuenteLoLlamaLaVentana(t *testing.T) {
	raiz := raizDelRepo(t)
	puente := filepath.Join(raiz, "frontend", "src", "puente.ts")
	fuente, err := os.ReadFile(puente)
	if err != nil {
		t.Fatal(err)
	}

	// Los métodos del objeto `esfinge`: dos espacios de sangría y abriendo paréntesis,
	// que es como está escrito el fichero entero.
	dentro := string(fuente)
	if i := strings.Index(dentro, "export const esfinge = {"); i >= 0 {
		dentro = dentro[i:]
	} else {
		t.Fatal("no se encuentra el objeto `esfinge` en puente.ts")
	}
	metodos := regexp.MustCompile(`(?m)^  (\w+): *\(`).FindAllStringSubmatch(dentro, -1)
	if len(metodos) < 50 {
		t.Fatalf("solo se han encontrado %d métodos en el puente: ¿ha cambiado cómo está escrito?", len(metodos))
	}

	// Y todo lo demás de la interfaz, que es donde tienen que usarse.
	otros, err := filepath.Glob(filepath.Join(raiz, "frontend", "src", "*"))
	if err != nil {
		t.Fatal(err)
	}
	var todo strings.Builder
	for _, f := range otros {
		if filepath.Base(f) == "puente.ts" {
			continue
		}
		datos, err := os.ReadFile(f)
		if err != nil {
			continue // carpetas y demás
		}
		todo.Write(datos)
	}
	usado := todo.String()

	// **Se busca el nombre, no la llamada.** Pidiendo `.nombre(` se escapan los que se
	// pasan como referencia —`hacer(esfinge.vaciarPapeleraDeBoveda)`—, que es uso
	// legítimo y daba un falso positivo. Un vigilante que señala lo que está bien se
	// deja de leer, que es lo que a este proyecto le costó seis versiones.
	suelto := regexp.MustCompile(`\.(\w+)`)
	usados := map[string]bool{}
	for _, m := range suelto.FindAllStringSubmatch(usado, -1) {
		usados[m[1]] = true
	}
	var muertos []string
	for _, m := range metodos {
		if !usados[m[1]] {
			muertos = append(muertos, m[1])
		}
	}
	if len(muertos) > 0 {
		t.Errorf("estos métodos del puente no los llama nadie en la ventana: %v\n"+
			"O les falta la pantalla que los use —que es lo que ha pasado dos veces— o sobran y hay que "+
			"quitarlos de aquí y de la lista de arriba.", muertos)
	}
}

// raizDelRepo es la carpeta con el `go.mod`.
func raizDelRepo(t *testing.T) string {
	t.Helper()
	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		arriba := filepath.Dir(d)
		if arriba == d {
			t.Fatal("no se encuentra la raíz del repositorio")
		}
		d = arriba
	}
}

// **Un estado que la ventana no conoce no da error: da el cajón de sastre.**
//
// `EstadoSincro.Estado` es una cadena, así que Go puede emitir uno nuevo y la ventana
// lo deja caer en el `default` de `frase()` sin que nada se queje. Pasó con
// `sin-acceso` (ADR 0052): a quien le acababan de quitar el acceso a una bóveda
// compartida, la ventana le decía **«Sin sincronizar»** —con el mensaje bueno llegando
// de Go y sin enseñarlo— y el botón de sincronizar giraba los veinte segundos del
// plazo de seguridad, porque tampoco contaba como pasada terminada. **Lo vio el cliente
// en su segundo Mac, recorriendo el último tramo de la ADR 0052**, y no lo dijo ninguna
// de las 118 pruebas de interfaz: todas sincronizan bien.
//
// Es la trampa del puente —escribir el lado de Go parece terminar el trabajo— con otra
// cara: aquí lo que falta no es un método, es **un valor**.
//
// Se compara contra las dos listas que tiene la ventana, porque fallan por separado: el
// tipo de `puente.ts` —que es lo que haría que TypeScript avisara en el `switch`— y los
// `case` de `frase()`. Con el estado en el tipo y sin su `case`, el fallo es
// exactamente el que se vio.
func TestLosEstadosDeLaSincroLosEnsenaLaVentana(t *testing.T) {
	raiz := raizDelRepo(t)
	leer := func(partes ...string) string {
		datos, err := os.ReadFile(filepath.Join(append([]string{raiz}, partes...)...))
		if err != nil {
			t.Fatal(err)
		}
		return string(datos)
	}

	// Los que Go emite, sacados de donde se construyen.
	deGo := map[string]bool{}
	for _, m := range regexp.MustCompile(`EstadoSincro\{Estado: "([a-z-]+)"`).
		FindAllStringSubmatch(leer("internal", "app", "cuenta.go"), -1) {
		deGo[m[1]] = true
	}
	if len(deGo) < 8 {
		t.Fatalf("solo se han encontrado %d estados en cuenta.go (%v): ¿ha cambiado cómo se escriben?", len(deGo), deGo)
	}

	// El tipo del puente, y los `case` de la frase que los traduce.
	tipo := leer("frontend", "src", "puente.ts")
	i := strings.Index(tipo, "export type EstadoSincro = {")
	if i < 0 {
		t.Fatal("no se encuentra el tipo EstadoSincro en puente.ts")
	}
	tipo = tipo[i:]
	if j := strings.Index(tipo, "};"); j > 0 {
		tipo = tipo[:j]
	}
	frases := leer("frontend", "src", "cuenta.tsx")
	if k := strings.Index(frases, "export function frase("); k >= 0 {
		frases = frases[k:]
		if j := strings.Index(frases, "\n}\n"); j > 0 {
			frases = frases[:j]
		}
	} else {
		t.Fatal("no se encuentra frase() en cuenta.tsx")
	}

	var sinTipo, sinFrase []string
	for estado := range deGo {
		if !strings.Contains(tipo, `"`+estado+`"`) {
			sinTipo = append(sinTipo, estado)
		}
		if !strings.Contains(frases, `case "`+estado+`":`) {
			sinFrase = append(sinFrase, estado)
		}
	}
	sort.Strings(sinTipo)
	sort.Strings(sinFrase)
	if len(sinTipo) > 0 {
		t.Errorf("estos estados los emite Go y no están en el tipo EstadoSincro de puente.ts: %v\n"+
			"Sin estar en el tipo, TypeScript no puede avisar de que falta su rama.", sinTipo)
	}
	if len(sinFrase) > 0 {
		t.Errorf("estos estados los emite Go y frase() no los traduce: %v\n"+
			"Caen en el `default` y la ventana dice «Sin sincronizar», que es lo que le pasó a `sin-acceso`: "+
			"el mensaje de Go llega y no se enseña. Cada estado va con su `case`, aunque la frase se repita.", sinFrase)
	}
}
