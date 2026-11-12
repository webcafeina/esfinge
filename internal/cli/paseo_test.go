package cli

// El camino entero de la ADR 0057, con **dos equipos de verdad**: una ventana que
// comparte un proyecto y un servidor que lo recibe y lo usa.
//
// # Por qué existe aparte de todas las demás
//
// Las piezas están probadas una a una —la ranura, `ejecutar`, el alta, la pasada de
// sincronización— y **el camino no lo recorría nadie**. Es la tercera vez que esa
// frase aparece en este proyecto: la primera fue `VolverALaBovedaPersonal`, la
// segunda `AceptarAcceso`, y las dos las encontró alguien recorriendo el camino a
// mano, nunca una prueba.
//
// Aquí el lado que comparte es `internal/app`, o sea **la ventana de verdad**, no un
// ayudante que repita lo que hace: un ayudante que reimplementa producción prueba el
// ayudante. Y el lado del servidor son los comandos de verdad, uno de ellos con el
// argv entero por si alguna bandera estuviera mal conectada.
//
// Se corre con `herramientas/con-servidor.sh`, como el resto de lo de la cuenta.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/app"
	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/llavero"
	"github.com/webcafeina/esfinge/internal/rutas"
	"github.com/webcafeina/esfinge/internal/sincro"
)

// unaVentanaSinPantalla es lo que `internal/app` necesita para funcionar sin
// escritorio. No hace nada porque en este paseo nadie abre un diálogo: lo que se
// ejercita es la cuenta, no los ficheros.
type unaVentanaSinPantalla struct{}

func (unaVentanaSinPantalla) ElegirFicheros(_, _ string, _ bool, _ app.Filtro) ([]string, error) {
	return nil, nil
}
func (unaVentanaSinPantalla) ElegirDondeGuardar(_, _, _ string) (string, error) { return "", nil }
func (unaVentanaSinPantalla) Avisar(string, any)                                {}
func (unaVentanaSinPantalla) Cerrar()                                           {}
func (unaVentanaSinPantalla) PonerEnPortapapeles(string) error                  { return nil }
func (unaVentanaSinPantalla) LeerPortapapeles() (string, error)                 { return "", nil }

// unEquipo es una carpeta de configuración con su programa dentro. Como la carpeta
// sale del entorno, se cambia de equipo volviendo a poner el entorno: por eso
// `usar` se llama **antes de cada bloque**, siempre. Sin eso, el proyecto de la
// ventana acabaría en la carpeta del servidor.
type unEquipo struct {
	carpeta string
	llave   string
}

func (e unEquipo) usar(t *testing.T) {
	t.Helper()
	rutas.ApartarLaCarpeta(e.carpeta, t.Setenv)
	// **La llave del servidor va fuera de la carpeta de las bóvedas**, también aquí.
	t.Setenv(llavero.VariableDeLaLlave, e.llave)
}

func unEquipoNuevo(t *testing.T) unEquipo {
	t.Helper()
	return unEquipo{carpeta: t.TempDir(), llave: filepath.Join(t.TempDir(), "llave")}
}

func elPaso(t *testing.T, n int, que string) {
	t.Helper()
	t.Logf("── %d · %s", n, que)
}

// **Una ventana comparte un proyecto, un servidor lo recibe y lo usa sin ver el
// secreto.**
func TestPaseoDeUnProyectoHastaUnServidor(t *testing.T) {
	raiz := elServidorDeCuentas(t)
	const maestra = "una contraseña maestra bien larga para el paseo"

	ventana := unEquipoNuevo(t)
	servidor := unEquipoNuevo(t)
	conLaMaestra := filepath.Join(t.TempDir(), "maestra.txt")
	if err := os.WriteFile(conLaMaestra, []byte(maestra), 0o600); err != nil {
		t.Fatal(err)
	}

	elPaso(t, 1, "la ventana crea su cuenta y un proyecto con una credencial dentro")
	ventana.usar(t)
	correoVentana := unCorreoNuevo(t)
	a := app.Nueva("paseo", unaVentanaSinPantalla{})
	app.ApuntarCuentasA(a, raiz)
	t.Cleanup(a.CerrarBoveda)
	if err := a.EmpezarRegistro(correoVentana); err != nil {
		t.Fatal(err)
	}
	if _, err := a.TerminarRegistro(correoVentana, elCodigoQueLlegue(t, raiz, correoVentana), maestra, ""); err != nil {
		t.Fatalf("la ventana no ha podido crear su cuenta: %v", err)
	}
	ref, err := a.CrearProyecto("Zeri's Coffee")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AbrirProyecto(ref); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Cloudflare de Zeri", Usuario: "zeri",
		Secreto: "el-testigo-de-zeri-que-no-puede-salir",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.SincronizarAhora(); err != nil {
		t.Fatal(err)
	}
	laVentana, err := a.MiIdentidad()
	if err != nil {
		t.Fatal(err)
	}

	elPaso(t, 2, "el servidor se da de alta con su propia cuenta y publica sus llaves")
	servidor.usar(t)
	correoServidor := unCorreoNuevo(t)
	o := &opciones{tema: "oscuro", silencio: true}
	ob := &opcionesBoveda{}
	ob.clave.Fichero = conLaMaestra
	alta := comandoServidorAlta(o, ob)
	if err := alta.Flags().Set("correo", correoServidor); err != nil {
		t.Fatal(err)
	}
	contestandoElCodigo(t, raiz, correoServidor, func() error { return alta.RunE(alta, nil) })

	elPaso(t, 3, "y activa la llave de este equipo, para abrir sin contraseña")
	if err := comandoServidorActivar(o, ob).RunE(nil, nil); err != nil {
		t.Fatal(err)
	}

	elPaso(t, 4, "la ventana le da acceso de solo ver, que es lo que hace el selector")
	ventana.usar(t)
	// **La huella se mira antes**, que es lo único que protege del servidor en el
	// primer envío (ADR 0043): sin llaves publicadas, el servidor daría unas
	// inventadas y el sobre llegaría cifrado hacia nadie.
	vista, err := a.HuellaDe(correoServidor)
	if err != nil {
		t.Fatalf("la ventana no ve la identidad del servidor: %v", err)
	}
	if vista.Huella == "" {
		t.Fatal("el servidor no ha publicado sus llaves, así que el sobre iría cifrado hacia nadie")
	}
	if err := a.DarAcceso(correoServidor, "ver"); err != nil {
		t.Fatalf("no ha podido dar acceso al servidor: %v", err)
	}
	quien, err := a.QuienTiene()
	if err != nil {
		t.Fatal(err)
	}
	if len(quien) != 1 || quien[0].Correo != correoServidor || quien[0].Permiso != "ver" {
		t.Fatalf("quién tiene acceso dice %+v", quien)
	}
	// **Y la ranura hay que subirla.** Dar acceso cambia el fichero del proyecto, y lo
	// que lo sube es la sincronización de la bóveda abierta: sin esto, el servidor se
	// baja una bóveda que todavía no tiene su ranura dentro. Es exactamente lo que le
	// va a pasar a quien pulse el selector y mire el servidor en el mismo segundo.
	if err := a.SincronizarAhora(); err != nil {
		t.Fatal(err)
	}

	elPaso(t, 5, "el servidor recibe el acceso y se trae la bóveda")
	servidor.usar(t)
	nuevas, _, err := recibirDesdeElComando(t, o, ob)
	if err != nil {
		t.Fatalf("el servidor no ha podido recibir: %v", err)
	}
	if len(nuevas) != 1 || !strings.HasPrefix(nuevas[0], "Zeri's Coffee") {
		t.Fatalf("lo aceptado es %v", nuevas)
	}
	// **Y dice de quién viene**: la huella de **quien lo comparte**, que es la de la
	// ventana y no la del servidor. Aquí no hay nadie que la compare, y aun así es lo
	// único que distingue un acceso de quien dices de uno de cualquiera que sepa el
	// correo de este servidor (ADR 0043).
	if !strings.Contains(nuevas[0], laVentana.Huella) {
		t.Errorf("lo aceptado dice venir de otro: %q, y la ventana es %s", nuevas[0], laVentana.Huella)
	}

	elPaso(t, 6, "y le da el secreto a un programa, sin verlo, por el argv de verdad")
	donde := filepath.Join(t.TempDir(), "lo-que-vio-el-programa")
	t.Setenv("DONDE", donde)
	if codigo := conArgv(t, "esfinge", "boveda", "ejecutar",
		"--compartida", "Zeri's Coffee",
		"--var", "CF_TOKEN=Cloudflare de Zeri",
		"--", "sh", "-c", `printf '%s' "$CF_TOKEN" > "$DONDE"`); codigo != SalidaOK {
		t.Fatalf("«ejecutar» ha salido con %d", codigo)
	}
	visto, err := os.ReadFile(donde)
	if err != nil {
		t.Fatalf("el programa no ha escrito nada: %v", err)
	}
	if string(visto) != "el-testigo-de-zeri-que-no-puede-salir" {
		t.Fatalf("al programa le ha llegado %q", visto)
	}

	elPaso(t, 7, "la ventana cambia la contraseña y el servidor usa la nueva")
	ventana.usar(t)
	entradas := buscarEnLaVentana(t, a, "Cloudflare")
	if len(entradas) != 1 {
		t.Fatalf("en el proyecto hay %d entradas", len(entradas))
	}
	entrada := entradas[0]
	entrada.Secreto = "el-testigo-nuevo-de-zeri"
	if err := a.GuardarEnBoveda(entrada); err != nil {
		t.Fatal(err)
	}
	if err := a.SincronizarAhora(); err != nil {
		t.Fatal(err)
	}

	servidor.usar(t)
	// Y aquí está lo que de verdad se compara con entregar una copia: **no hay que
	// volver a entregar nada**. La pasada va por delante de abrir, así que lo que el
	// programa recibe es lo de ahora.
	if codigo := conArgv(t, "esfinge", "boveda", "ejecutar",
		"--compartida", "Zeri's",
		"--var", "CF_TOKEN=Cloudflare de Zeri",
		"--", "sh", "-c", `printf '%s' "$CF_TOKEN" > "$DONDE"`); codigo != SalidaOK {
		t.Fatalf("«ejecutar» ha salido con %d la segunda vez", codigo)
	}
	visto, err = os.ReadFile(donde)
	if err != nil {
		t.Fatal(err)
	}
	if string(visto) != "el-testigo-nuevo-de-zeri" {
		t.Fatalf("el servidor sigue usando la contraseña vieja: %q", visto)
	}

	elPaso(t, 8, "y con permiso de solo ver, lo que escriba aquí no sube")
	// Lo impone el servidor de cuentas y no el cifrado, así que lo que se comprueba es
	// que el servidor dice que no, no que aquí no se pueda escribir.
	comp := laCompartidaDelServidor(t, o, ob, "Zeri's")
	b, err := boveda.AbrirCompartida(rutas.Compartida(comp.Dueno, comp.Ref), comp.Titular, laPersonalDelServidor(t, o, ob))
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Poner(boveda.Entrada{Tipo: boveda.TipoCredencial, Titulo: "Algo que el servidor no puede subir", Secreto: "x"}); err != nil {
		t.Fatal(err)
	}
	err = intentarSubirLaCompartida(t, o, ob, b, comp)
	if err == nil {
		t.Fatal("con permiso de solo ver ha podido subir")
	}
	t.Logf("   el servidor contesta: %v", err)
	b.Cerrar()

	elPaso(t, 9, "la ventana le sube el permiso a editar, y el servidor se entera solo")
	// **Esto es la ADR 0058 entera, y sin esto el interruptor sería mentira.** El
	// permiso que guarda quien recibe solo se escribía al aceptar el sobre del buzón, así
	// que subírselo no le llegaba nunca: el servidor empezaba a aceptarle subidas y su
	// Esfinge seguía negándose a escribir. Ahora el servidor lo dice **al bajar**, y la
	// pasada que va por delante de abrir lo recoge.
	//
	// Y se comprueba **sin volver a `recibir`**: es lo que distingue cambiar el permiso
	// en el sitio de quitar y volver a dar.
	ventana.usar(t)
	quien, err = a.QuienTiene()
	if err != nil {
		t.Fatal(err)
	}
	if len(quien) != 1 {
		t.Fatalf("quién tiene acceso dice %+v", quien)
	}
	if err := a.CambiarPermiso(quien[0].Titular, "editar"); err != nil {
		t.Fatalf("no ha podido subirle el permiso: %v", err)
	}
	if err := a.SincronizarAhora(); err != nil {
		t.Fatal(err)
	}

	servidor.usar(t)
	// **Por una variable de entorno y no por la línea**: lo que se escribe en la línea
	// queda en el historial del shell y lo lee un `ps`, y eso vale también aquí.
	t.Setenv("EL_TESTIGO_NUEVO", "el-testigo-que-ha-rotado-el-agente")
	// Y ahora **sí** escribe, por el argv de verdad: la pasada de abrir trae el permiso
	// nuevo, la fila se vuelve a leer con él y la guarda deja pasar.
	if codigo := conArgv(t, "esfinge", "boveda", "crear",
		"--compartida", "Zeri's",
		"--titulo", "Lo que el servidor ya puede guardar",
		"--secreto-env", "EL_TESTIGO_NUEVO",
		"--quien", "Serafin",
		"-q"); codigo != SalidaOK {
		t.Fatal("con permiso de editar, el servidor no ha podido crear una entrada")
	}

	// Y lo escrito **sube**, que es la otra mitad: guardarlo aquí y que no salga de este
	// equipo sería lo mismo que no poder escribir, con peor cara.
	ventana.usar(t)
	esperarEnLaVentana(t, a, "Lo que el servidor ya puede guardar")

	elPaso(t, 10, "la ventana cambia la clave de la bóveda y el servidor sigue trabajando")
	// **Esto es la ADR 0059 en la cabeza que no tiene pantalla, y es lo que faltaba.**
	// Rotar deja la cadena de un titular corta en el acto, y la nueva solo la abre la
	// identidad hacia la que se selló su ranura, que vive en su bóveda personal. La
	// ventana y la extensión recogen esa cadena en su pasada; **la línea de comandos no lo
	// hacía**, así que el VPS se quedó sordo con la primera rotación de verdad
	// (2026-11-12): sincronizaba, decía «esto puede estar viejo» y servía la copia de
	// antes, para siempre.
	//
	// Y la razón de que esto vaya **aquí** y no en una prueba de `internal/boveda`: el
	// fallo solo existe cuando el servidor **ya tiene una copia anterior a la rotación**,
	// que es lo que los nueve pasos de arriba han construido. Bajando la bóveda por
	// primera vez después de rotar no hay nada que recoger y todo funciona — que es
	// exactamente por lo que la prueba de la ventana (`acceso_entre_cuentas_test.go`,
	// paso 9) pasaba en verde con este agujero dentro.
	ventana.usar(t)
	if err := a.RotarLaClave(); err != nil {
		t.Fatalf("la ventana no ha podido cambiar la clave de su proyecto: %v", err)
	}
	// Y se cambia el secreto **después** de rotar: así lo que el servidor tiene que
	// entregar solo se puede leer con la clave nueva, y servir la copia de antes deja de
	// ser un aviso para convertirse en una respuesta equivocada.
	entradas = buscarEnLaVentana(t, a, "Cloudflare")
	if len(entradas) != 1 {
		t.Fatalf("en el proyecto hay %d entradas", len(entradas))
	}
	entrada = entradas[0]
	entrada.Secreto = "el-testigo-de-despues-de-la-clave-nueva"
	if err := a.GuardarEnBoveda(entrada); err != nil {
		t.Fatal(err)
	}
	if err := a.SincronizarAhora(); err != nil {
		t.Fatal(err)
	}

	servidor.usar(t)
	if codigo := conArgv(t, "esfinge", "boveda", "ejecutar",
		"--compartida", "Zeri's",
		"--var", "CF_TOKEN=Cloudflare de Zeri",
		"--", "sh", "-c", `printf '%s' "$CF_TOKEN" > "$DONDE"`); codigo != SalidaOK {
		t.Fatalf("«ejecutar» ha salido con %d tras la rotación", codigo)
	}
	visto, err = os.ReadFile(donde)
	if err != nil {
		t.Fatal(err)
	}
	if string(visto) != "el-testigo-de-despues-de-la-clave-nueva" {
		t.Fatalf("tras cambiar la clave, al programa le ha llegado %q: la rotación no ha llegado a este equipo", visto)
	}
	// Y el fichero de aquí **ha pasado de época**, que es lo que dice que ha recogido la
	// cadena y no que haya acertado por casualidad. Se mira la cabecera en claro, que va
	// fuera del cifrado justamente para poder elegir la clave antes de descifrar.
	laCabecera := struct {
		Formato int    `json:"formato"`
		Epoca   string `json:"epoca"`
	}{}
	crudo, err := os.ReadFile(rutas.Compartida(comp.Dueno, comp.Ref))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(crudo, &laCabecera); err != nil {
		t.Fatal(err)
	}
	if laCabecera.Formato != boveda.FormatoConClavesDeContenido || laCabecera.Epoca == "" {
		t.Errorf("la copia del servidor sigue en formato %d con época %q", laCabecera.Formato, laCabecera.Epoca)
	}
	// Y **sigue pudiendo escribir**, que es la otra mitad de seguir dentro: su ranura se
	// reselló con la cadena nueva, así que el permiso de editar no se lo lleva la rotación.
	t.Setenv("EL_TESTIGO_DE_DESPUES", "lo-que-el-servidor-guarda-tras-la-clave-nueva")
	if codigo := conArgv(t, "esfinge", "boveda", "crear",
		"--compartida", "Zeri's",
		"--titulo", "Lo que el servidor guarda tras la clave nueva",
		"--secreto-env", "EL_TESTIGO_DE_DESPUES",
		"--quien", "Serafin",
		"-q"); codigo != SalidaOK {
		t.Fatal("tras la rotación, el servidor no ha podido crear una entrada")
	}
	// **Y lee el cuerpo entero, que es lo que destapó probar las capacidades a mano**
	// (2026-11-12). `exportar` preguntaba `b.llave == nil` para saber si la bóveda estaba
	// abierta, y un titular del formato 2 **no tiene esa clave**: contestaba «La bóveda
	// está cerrada» con la bóveda abierta y los siete secretos legibles de uno en uno.
	// Va aquí y no solo en `internal/boveda` porque esto es el camino por el que se vio:
	// la línea de comandos, sobre una bóveda que acaba de rotar.
	elCSV := filepath.Join(t.TempDir(), "copia.csv")
	if codigo := conArgv(t, "esfinge", "boveda", "exportar",
		"--compartida", "Zeri's", "-o", elCSV, "-q"); codigo != SalidaOK {
		t.Fatal("tras la rotación, el servidor no ha podido exportar la bóveda")
	}
	if csv, err := os.ReadFile(elCSV); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(string(csv), "Cloudflare de Zeri") {
		t.Error("el CSV sale sin las entradas")
	}

	ventana.usar(t)
	esperarEnLaVentana(t, a, "Lo que el servidor guarda tras la clave nueva")

	elPaso(t, 11, "y al quitarle el acceso, el servidor se va y borra su copia")
	// **Esto es lo único que llega al disco del servidor.** El 403 corta lo que baja
	// y la rotación de la clave que la ADR 0052 prometía **no existe** (ver su
	// apartado del 2026-10-21), así que sin este borrado la bóveda de un cliente con
	// el que ya no se trabaja se quedaría en el VPS, legible por quien entre, para
	// siempre.
	ventana.usar(t)
	quien, err = a.QuienTiene()
	if err != nil {
		t.Fatal(err)
	}
	if len(quien) != 1 {
		t.Fatalf("quién tiene acceso dice %+v", quien)
	}
	if err := a.QuitarAcceso(quien[0].Titular); err != nil {
		t.Fatal(err)
	}
	if err := a.SincronizarAhora(); err != nil {
		t.Fatal(err)
	}

	servidor.usar(t)
	laRuta := rutas.Compartida(comp.Dueno, comp.Ref)
	if _, err := os.Stat(laRuta); err != nil {
		t.Fatalf("la copia del servidor ya no estaba antes de quitarle el acceso: %v", err)
	}
	_, err = abrirLaCompartidaPorNombre(o, ob, "Zeri's")
	if err == nil {
		t.Fatal("el servidor sigue abriendo una bóveda a la que le han quitado el acceso")
	}
	if !strings.Contains(err.Error(), "Ya no tienes acceso") {
		t.Fatalf("el error no dice qué ha pasado: %v", err)
	}
	if _, err := os.Stat(laRuta); !os.IsNotExist(err) {
		t.Fatalf("la copia sigue en el disco del servidor: %v", err)
	}
	// Y la lápida, que es lo que impide que vuelva en la siguiente pasada.
	if _, err := laCompartidaQueSeLlama(laPersonalDelServidor(t, o, ob), "Zeri's"); err == nil {
		t.Error("la bóveda retirada sigue en la lista del servidor")
	}
}

// recibirDesdeElComando corre «servidor recibir» de verdad y devuelve lo que aceptó.
func recibirDesdeElComando(t *testing.T, o *opciones, ob *opcionesBoveda) ([]string, []string, error) {
	t.Helper()
	b, err := abrirLaBoveda(o, ob)
	if err != nil {
		return nil, nil, err
	}
	defer b.Cerrar()
	return recibirLoCompartido(b)
}

func laPersonalDelServidor(t *testing.T, o *opciones, ob *opcionesBoveda) *boveda.Boveda {
	t.Helper()
	b, err := abrirLaBoveda(o, ob)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Cerrar)
	return b
}

func laCompartidaDelServidor(t *testing.T, o *opciones, ob *opcionesBoveda, nombre string) boveda.Compartida {
	t.Helper()
	comp, err := laCompartidaQueSeLlama(laPersonalDelServidor(t, o, ob), nombre)
	if err != nil {
		t.Fatal(err)
	}
	return comp
}

// esperarEnLaVentana pide pasadas hasta que ese título aparece.
//
// **Se espera al contenido y no a que la sincronización diga «al día»**: `alDia` se
// conforma con una pasada anterior, y la de aquí puede haber terminado antes de que el
// servidor subiera. Esperar al estado era decir que no llega nada cuando lo que pasa es
// que todavía no se ha vuelto a mirar.
func esperarEnLaVentana(t *testing.T, a *app.App, titulo string) {
	t.Helper()
	for i := 0; i < 30; i++ {
		if err := a.SincronizarAhora(); err == nil {
			for _, e := range buscarEnLaVentana(t, a, titulo) {
				if e.Titulo == titulo {
					return
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("«%s» no ha llegado a la ventana", titulo)
}

// buscarEnLaVentana saca la entrada entera del proyecto abierto en la ventana.
func buscarEnLaVentana(t *testing.T, a *app.App, q string) []boveda.Entrada {
	t.Helper()
	encajan, err := a.BuscarEnBoveda(q)
	if err != nil {
		t.Fatal(err)
	}
	var out []boveda.Entrada
	for _, e := range encajan {
		entera, err := a.VerDeBoveda(e.ID)
		if err != nil {
			t.Fatalf("la entrada %s ya no está: %v", e.ID, err)
		}
		out = append(out, entera)
	}
	return out
}

// conArgv corre la línea de comandos **como la corre el binario**: con su argv, su
// cobra y su código de salida. Es lo que caza una bandera mal conectada, que es un
// fallo que ninguna prueba que toque la estructura por dentro puede ver.
func conArgv(t *testing.T, args ...string) int {
	t.Helper()
	antes := os.Args
	os.Args = args
	defer func() { os.Args = antes }()
	return Ejecutar("paseo")
}

// intentarSubirLaCompartida es lo que el permiso de solo ver tiene que impedir, y lo
// impide **el servidor**: aquí se escribe sin problema y lo que falla es la subida.
func intentarSubirLaCompartida(t *testing.T, o *opciones, ob *opcionesBoveda,
	b *boveda.Boveda, comp boveda.Compartida) error {
	t.Helper()
	mia := laPersonalDelServidor(t, o, ob)
	s, err := abrirLaCuenta(mia, mia.Ruta())
	if err != nil {
		return err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), elPlazoDeLaRed)
	defer cancelar()
	return s.conLaSesion(ctx, func(token string) error {
		sin := &sincro.Sincronizador{
			Servidor: &laCompartidaRemota{c: s.c, dueno: comp.Dueno, ref: comp.Ref},
			Boveda:   b,
			Memoria:  sincro.JuntoALaBoveda{Ruta: b.Ruta()},
			Token:    func() string { return token },
		}
		_, err := sin.Sincronizar(ctx)
		return err
	})
}
