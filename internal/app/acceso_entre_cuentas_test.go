package app

// El camino entero de dar acceso a una bóveda de proyecto (ADR 0052), con dos cuentas
// contra el Worker de verdad.
//
// # Por qué esta prueba existe aparte de todas las demás
//
// Las piezas estaban probadas una a una —el núcleo, el servidor, la sincronización, la
// pantalla— y **el camino no lo recorría nadie**. La primera vez que se recorrió encontró
// lo que ninguna de ellas podía ver: un acceso llegaba al buzón, se intentaba abrir como
// la copia de una entrada, fallaba, y la ventana lo enseñaba como «No se puede abrir ·
// Este envío no es para esta bóveda» con un único botón para descartarlo.
// `AceptarAcceso` estaba escrita en Go y en el puente **y no la llamaba nadie**.
//
// Es la segunda vez que pasa lo mismo —antes fue `VolverALaBovedaPersonal`— y por eso
// esto se queda: es la prueba de la tubería que a esta funcionalidad le faltaba.
//
// Con el Worker **local**, los códigos del alta los lee ella del buzón. Para recorrerlo
// contra el **desplegado** de pruebas, donde el buzón está detrás de Cloudflare Access y
// los códigos se copian a mano:
//
//	ESFINGE_SERVIDOR_PRUEBAS=https://esfinge-cuentas-pruebas.webcafe-na.workers.dev \
//	PASEO_CORREO_A=… PASEO_CODIGO_A=… PASEO_CORREO_B=… PASEO_CODIGO_B=… \
//	go test ./internal/app -run TestPaseoDeUnAccesoEntreDosCuentas -v -count=1
//
// Las dos altas se piden antes con `POST /v1/registro/inicio`, y el código **caduca a los
// diez minutos**.

import (
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
)

func delEntorno(t *testing.T, nombre string) string {
	t.Helper()
	v := strings.TrimSpace(os.Getenv(nombre))
	if v == "" {
		t.Fatalf("falta %s", nombre)
	}
	return v
}

// cuentaDelPaseo crea la cuenta con el código que se le pase, y **si no se le pasa
// ninguno, pidiendo el alta y leyendo el buzón por HTTP**.
//
// Lo segundo es lo que permite correr este mismo paseo contra el Worker **local**, donde
// el buzón no tiene Access delante. Y hace falta por una razón de método: con el
// desplegado fallando al primer paso, lo primero que hay que saber es si falla el paseo o
// falla el código que alguien copió a mano — y eso solo lo contesta correr lo mismo donde
// los códigos no los copia nadie.
func cuentaDelPaseo(t *testing.T, raiz string, e *equipoDePrueba, correo, codigo, maestra string) {
	t.Helper()
	if codigo == "" {
		if err := e.a.EmpezarRegistro(correo); err != nil {
			t.Fatal(err)
		}
		codigo = codigoDelBuzon(t, raiz, correo)
	}
	if _, err := e.a.TerminarRegistro(correo, codigo, maestra, ""); err != nil {
		t.Fatalf("no se ha podido crear la cuenta de %s: %v", correo, err)
	}
}

// paso escribe por dónde va, porque esto se lee a mano y lo que importa es **en qué
// tramo se queda** si falla. Es la misma idea que ponerle voz a cada tramo de un fallo
// mudo: sin esto, un error a mitad no dice de quién era la bóveda ni qué se intentaba.
func paso(t *testing.T, n int, que string) {
	t.Helper()
	t.Logf("── %d · %s", n, que)
}

// esperarA pide pasadas hasta que ese título aparece en la bóveda abierta.
//
// Hace falta porque «en vivo» aquí es **al abrir y cada minuto** (ADR 0052): lo que se
// comprueba no es que llegue al instante, es que llegue. Y el plazo es generoso a
// propósito, que esto habla con un servidor de verdad.
func esperarA(t *testing.T, a *App, titulo string) {
	t.Helper()
	limite := time.Now().Add(60 * time.Second)
	var visto string
	for time.Now().Before(limite) {
		if err := a.SincronizarAhora(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(300 * time.Millisecond)
		visto = titulosDe(t, a)
		if strings.Contains(visto, titulo) {
			return
		}
		if e := a.EstadoDeCuenta().Sincro; e.Estado == "error" || e.Estado == "hay-que-entrar" {
			t.Fatalf("la sincronización dice %s: %s", e.Estado, e.Mensaje)
		}
	}
	t.Fatalf("no ha llegado %q; en la bóveda hay: %s", titulo, visto)
}

func TestPaseoDeUnAccesoEntreDosCuentas(t *testing.T) {
	raiz := servidorDeCuentas(t)
	// Sin correos ni códigos, se inventan y se leen del buzón: es el modo local.
	correoAna := os.Getenv("PASEO_CORREO_A")
	correoBeto := os.Getenv("PASEO_CORREO_B")
	if correoAna == "" {
		correoAna, correoBeto = correoDePrueba(), correoDePrueba()
	}
	codigoAna := strings.TrimSpace(os.Getenv("PASEO_CODIGO_A"))
	codigoBeto := strings.TrimSpace(os.Getenv("PASEO_CODIGO_B"))
	const maestra = "una contraseña maestra bien larga para el paseo"

	ana := nuevoEquipo(t, raiz)
	beto := nuevoEquipo(t, raiz)

	// **`usar` antes de cada bloque, siempre.** Las pruebas de dos cuentas que ya
	// había no lo necesitan porque solo tocan bóvedas ya abiertas, que llevan su ruta
	// dentro; aquí se crean proyectos y se bajan bóvedas ajenas, y eso **calcula rutas
	// desde el entorno**. Sin esto, el proyecto de Ana acabaría en la carpeta de Beto.

	paso(t, 1, "Ana crea su cuenta")
	ana.usar(t)
	cuentaDelPaseo(t, raiz, ana, correoAna, codigoAna, maestra)
	mia, err := ana.a.MiIdentidad()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("   Ana es %s", mia.Huella)

	// **Y se le borra el identificador de la cuenta, a propósito**, para que Ana sea una
	// cuenta **de las de antes de la 2.40.0**.
	//
	// Es el caso que ninguna prueba veía y que el cliente se encontró al dar el primer
	// acceso de verdad: `Cuenta` solo se escribía **al entrar**, así que quien ya tenía
	// cuenta lo tenía vacío y `DarAcceso` contestaba «Para dar acceso hace falta una
	// cuenta» en un equipo que llevaba semanas con una. Aquí todas las cuentas se crean
	// en el momento, así que el campo siempre estaba y el fallo era invisible.
	//
	// Quitarlo aquí es lo que hace que esta prueba ejercite a quien ya estaba, que es
	// **todo el mundo** salvo quien estrene Esfinge hoy.
	deAna := leerDatosCuenta()
	deAna.Cuenta = ""
	if err := guardarDatosCuenta(deAna); err != nil {
		t.Fatal(err)
	}

	paso(t, 2, "Beto crea la suya y publica sus llaves")
	beto.usar(t)
	cuentaDelPaseo(t, raiz, beto, correoBeto, codigoBeto, maestra)
	// **Publicar es lo que permite recibir**, y por eso se mira la huella aquí: sin
	// llaves publicadas, el servidor le da a Ana unas inventadas y el sobre llega
	// cifrado hacia nadie.
	suya, err := beto.a.MiIdentidad()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("   Beto es %s", suya.Huella)

	paso(t, 3, "Ana crea el proyecto y pone algo dentro")
	ana.usar(t)
	ref, err := ana.a.CrearProyecto("Zeri's Coffee")
	if err != nil {
		t.Fatal(err)
	}
	if err := ana.a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if err := ana.a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Hosting de Zeri", Usuario: "zeri", Secreto: "la del hosting",
	}); err != nil {
		t.Fatal(err)
	}
	alDia(t, ana.a, time.Now())

	paso(t, 4, "Ana comprueba la huella de Beto y le da acceso para editar")
	vista, err := ana.a.HuellaDe(correoBeto)
	if err != nil {
		t.Fatal(err)
	}
	if vista.Huella != suya.Huella {
		t.Fatalf("la huella que ve Ana (%s) no es la de Beto (%s): el servidor le está dando unas inventadas", vista.Huella, suya.Huella)
	}
	if err := ana.a.DarAcceso(correoBeto, "editar"); err != nil {
		t.Fatalf("Ana no ha podido dar acceso: %v", err)
	}
	quien, err := ana.a.QuienTiene()
	if err != nil {
		t.Fatal(err)
	}
	if len(quien) != 1 || quien[0].Correo != correoBeto || !quien[0].EnElServidor {
		t.Fatalf("quién tiene acceso dice %+v", quien)
	}
	titular := quien[0].Titular
	t.Logf("   Beto es el titular %s", titular)

	paso(t, 5, "A Beto le espera en el buzón y lo acepta")
	beto.usar(t)
	esperando, err := beto.a.Buzon()
	if err != nil {
		t.Fatal(err)
	}
	if len(esperando) != 1 {
		t.Fatalf("en el buzón de Beto hay %d cosas: %+v", len(esperando), esperando)
	}
	t.Logf("   el buzón enseña: acceso=%v titulo=%q huella=%q permiso=%q error=%q",
		esperando[0].Acceso, esperando[0].Titulo, esperando[0].Huella, esperando[0].Permiso, esperando[0].Error)
	// **El buzón tiene que decir que esto es un acceso, no enseñarlo roto.**
	//
	// Es lo que encontró este paseo y lo que lo justifica entero: hasta aquí, un acceso
	// llegaba al buzón, fallaba al abrirlo como copia y se enseñaba como «No se puede
	// abrir · Este envío no es para esta bóveda» —que además de inútil es mentira—, con
	// un solo botón: descartar. `AceptarAcceso` estaba en Go y en el puente **y no la
	// llamaba nadie**.
	if esperando[0].Error != "" {
		t.Fatalf("el acceso llega al buzón como un sobre roto: %s", esperando[0].Error)
	}
	if !esperando[0].Acceso {
		t.Fatal("el buzón no dice que esto es un acceso, así que la ventana ofrecerá guardarlo como una copia")
	}
	if esperando[0].Titulo != "Zeri's Coffee" || esperando[0].Permiso != "editar" {
		t.Fatalf("el buzón no dice de qué bóveda es ni qué se podrá hacer: %+v", esperando[0])
	}
	if esperando[0].Huella != mia.Huella {
		t.Fatalf("el buzón dice que viene de %s y Ana es %s", esperando[0].Huella, mia.Huella)
	}
	if err := beto.a.AceptarAcceso(esperando[0].ID); err != nil {
		t.Fatalf("Beto no ha podido aceptar el acceso: %v", err)
	}

	paso(t, 6, "Beto la abre y ve lo de Ana")
	suyas, err := beto.a.Compartidas()
	if err != nil {
		t.Fatal(err)
	}
	if len(suyas) != 1 || suyas[0].Nombre != "Zeri's Coffee" || suyas[0].Permiso != "editar" || !suyas[0].EnEsteEquipo {
		t.Fatalf("la lista de compartidas de Beto es %+v", suyas)
	}
	if err := beto.a.AbrirCompartida(suyas[0].Dueno, suyas[0].Ref); err != nil {
		t.Fatalf("Beto no abre la bóveda compartida: %v", err)
	}
	if v := titulosDe(t, beto.a); !strings.Contains(v, "Hosting de Zeri") {
		t.Fatalf("Beto no ve lo de Ana dentro de la compartida: %s", v)
	}

	paso(t, 7, "Beto escribe dentro y lo sube")
	if err := beto.a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Correo de Zeri", Usuario: "hola@zeri", Secreto: "la del correo",
	}); err != nil {
		t.Fatalf("Beto no ha podido escribir con permiso de editar: %v", err)
	}
	alDia(t, beto.a, time.Now())

	paso(t, 8, "y a Ana le llega")
	ana.usar(t)
	// **Se espera al contenido, no a que diga «al día».** `alDia` se conforma con una
	// pasada anterior, y la de Ana ya había terminado **antes** de que Beto subiera: con
	// ella, esto decía que no le llegaba nada cuando lo que pasaba es que todavía no
	// había vuelto a mirar. Es el mismo error que esperar a un estado en vez de al hecho.
	esperarA(t, ana.a, "Correo de Zeri")

	paso(t, 9, "Ana le quita el acceso, que rota la clave")
	if err := ana.a.QuitarAcceso(titular); err != nil {
		t.Fatalf("Ana no ha podido quitar el acceso: %v", err)
	}
	quien, err = ana.a.QuienTiene()
	if err != nil {
		t.Fatal(err)
	}
	if len(quien) != 0 {
		t.Fatalf("tras quitarlo, quién tiene acceso dice %+v", quien)
	}
	// Y Ana sigue trabajando en lo suyo, que es lo que la rotación no puede romper.
	if err := ana.a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Después de rotar", Secreto: "x",
	}); err != nil {
		t.Fatalf("Ana no puede escribir en su propio proyecto tras rotar: %v", err)
	}
	alDia(t, ana.a, time.Now())

	paso(t, 10, "y Beto se queda fuera: se entera, sale y la bóveda se va de su equipo")
	beto.usar(t)
	// **Se espera al hecho, no al estado** — y aquí eso no es una preferencia de estilo:
	// desde la ADR 0053, `sin-acceso` es **transitorio**. Beto se entera, sale a su
	// bóveda personal y arranca la sincronización de ésa, que va perfectamente, así que
	// medio segundo después el estado vuelve a ser «al-dia». Esperar a leer «sin-acceso»
	// era esperar a ganar una carrera: esta prueba se puso roja al escribir el borrado,
	// con el borrado funcionando.
	//
	// Lo que sí es estable, y es lo que de verdad hay que comprobar, son las tres cosas
	// de abajo: que la fila quede tachada, que el fichero no esté, y que **la bóveda
	// propia de Beto siga abierta**.
	limite := time.Now().Add(30 * time.Second)
	var fila CompartidaEnLaLista
	vistos := map[string]bool{}
	for time.Now().Before(limite) {
		// **Que esto falle aquí no es un fallo: es lo que se está comprobando.**
		//
		// Al enterarse del 403, Beto sale de la bóveda retirada y vuelve a la personal,
		// y entre que se para una sincronización y arranca la otra hay un instante en
		// que no hay ninguna — y entonces esto contesta «aquí no se está
		// sincronizando». Con `t.Fatal`, la prueba moría **justo por haber funcionado**,
		// y además de forma intermitente: depende de en qué milisegundo caiga la vuelta.
		//
		// Se sigue intentando, que es lo que haría una persona pulsando el botón, y lo
		// que decide sigue siendo el hecho de abajo.
		if err := beto.a.SincronizarAhora(); err != nil {
			vistos["no-sincroniza: "+err.Error()] = true
		}
		time.Sleep(500 * time.Millisecond)
		// Se apuntan los estados que se han llegado a ver: si esto falla, lo primero que
		// hay que saber es si el 403 llegó siquiera.
		vistos[beto.a.EstadoDeCuenta().Sincro.Estado] = true
		lista, err := beto.a.Compartidas()
		if err != nil {
			t.Fatalf("Beto no puede leer su lista de compartidas: %v", err)
		}
		if len(lista) != 1 {
			t.Fatalf("Beto tiene %d compartidas y tenía que tener una tachada: %+v", len(lista), lista)
		}
		fila = lista[0]
		if fila.Retirada != "" && !fila.EnEsteEquipo {
			break
		}
	}
	if fila.Retirada == "" {
		t.Fatalf("la compartida de Beto no se ha tachado: la lista sigue diciendo que es una bóveda suya "+
			"(estados vistos: %v)", claves(vistos))
	}
	if fila.EnEsteEquipo {
		t.Fatalf("la bóveda retirada sigue en el disco de Beto (estados vistos: %v)", claves(vistos))
	}
	// **Y la fila se queda**, que es lo que permite decir cuál se fue: sin ella, la
	// bóveda de un cliente desaparece de la pantalla sin que nadie sepa cuál era.
	if fila.Nombre == "" {
		t.Fatal("la fila tachada no dice de qué bóveda era")
	}
	t.Logf("   tachada %q el %s · estados vistos: %v", fila.Nombre, fila.Retirada, claves(vistos))

	// **El 403 no es el 401.** Lo contrario sería cerrarle a Beto su propia bóveda
	// porque otra persona le quitó el acceso a la de ella.
	if e := beto.a.EstadoBoveda(); !e.Abierta {
		t.Fatal("quitarle el acceso a una bóveda ajena le ha cerrado la bóveda a Beto")
	}
	// Y ha vuelto a la suya, que es lo que hace que lo de arriba se pueda decir: quedarse
	// «dentro» de una bóveda cuyo fichero ya no está no es un sitio donde se pueda estar.
	if ref, dueno := beto.a.bovedaActiva(), beto.a.duenoDeLaActiva(); ref != "" || dueno != "" {
		t.Fatalf("Beto se ha quedado dentro de la bóveda retirada: ref=%q dueno=%q", ref, dueno)
	}
	t.Log("── el paseo ha llegado hasta el final")
}

// claves saca las de un conjunto, ordenadas, para poder decir qué se ha visto.
func claves(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
