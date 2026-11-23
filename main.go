//go:build !dev

// Esfinge, con ventana.
//
// La misma herramienta que la línea de comandos, el mismo núcleo y el mismo
// formato de contenedor; lo único distinto es que aquí hay ratón, diálogos del
// sistema y sitio donde soltar ficheros.
//
// Está en la raíz del proyecto y no en cmd/, que sería lo idiomático en Go,
// porque Wails genera los enlaces con la interfaz buscando el paquete main
// justo donde está wails.json. Con el punto de entrada en cmd/esfinge-gui, la
// compilación falla con «no Go files». La línea de comandos sí vive en cmd/.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	ventanaWails "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/webcafeina/esfinge/internal/app"
	"github.com/webcafeina/esfinge/internal/entorno"
	"github.com/webcafeina/esfinge/internal/interfaz"
	"github.com/webcafeina/esfinge/internal/ventana"
)

// version la pone el enlazador al compilar.
var version = "dev"

func main() {
	// **Y si algo entra en pánico aquí, que al menos lo diga.**
	//
	// Una aplicación con ventana no tiene consola: en Windows, lo que Go escribe al
	// morir **no lo lee nadie**, así que un pánico se ve como medio segundo de cursor
	// y nada más. Le pasó al cliente de la 2.72.0 a la 2.76.0 y costó dos días
	// acotarlo, porque ni siquiera el Visor de eventos apunta nada —un pánico de Go
	// no es un fallo de aplicación de Windows—.
	//
	// Esto no arregla el pánico: lo **deja escrito**, con su pila, al lado de las
	// preferencias. No cubre a quien entre en pánico en otra gorutina —eso no se
	// puede recoger desde aquí— pero sí todo lo que pase en el camino de arranque,
	// que es donde duele.
	defer func() {
		if r := recover(); r != nil {
			apuntarElArranqueFallido(fmt.Errorf("pánico: %v\n%s", r, debug.Stack()))
			panic(r)
		}
	}()

	escritorio := app.NuevoEscritorio()
	aplicacion := app.Nueva(version, escritorio)

	// Doble clic en un .esf: en Windows y en Linux el sistema pasa la ruta como
	// argumento, y con eso la aplicación abre directamente en descifrar. En macOS
	// no llega así sino por un evento de Apple, que se recoge más abajo en
	// Mac.OnFileOpen.
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			aplicacion.AlAbrirCon(arg)
		}
	}

	// El contexto no existe hasta que Wails arranca, y el menú se construye
	// antes: se le pasa una función que lo consulta cuando hace falta, que es
	// siempre después de que la ventana esté abierta.
	var ctx context.Context

	// El vidrio del sistema: macOS lo da con NSVisualEffectView y Windows 11 con
	// Mica. Linux no lo ofrece en Wails, y ahí la ventana se queda opaca.
	//
	// Va acompañado de un fondo transparente, porque si el webview deja pasar la
	// luz el color lo tiene que poner el CSS. Eso es lo que la interfaz consulta
	// con Vidrio(): sin él dejaría la ventana vacía justo donde no hay efecto.
	vidrio := runtime.GOOS == "darwin" || runtime.GOOS == "windows"
	// **Y con `ESFINGE_SIN_VIDRIO` puesto, la ventana se pide opaca.**
	//
	// No es sólo para diagnosticar: una ventana translúcida la compone el sistema,
	// y donde eso no funcione —una máquina virtual sin aceleración, un controlador
	// raro— lo que queda es una ventana que no se ve. Poder pedirla opaca es la
	// salida, y no tenerla deja a alguien sin programa y sin nada que probar.
	if os.Getenv("ESFINGE_SIN_VIDRIO") != "" {
		vidrio = false
	}
	app.MarcarVidrio(aplicacion, vidrio)

	// Y en qué escritorio, que en Linux no lo dice el sistema: GNOME y KDE
	// organizan su ventana de forma distinta y hasta ahora Plasma veía la de
	// GNOME. Se lee **aquí y una vez**, antes de crear la ventana: de esto
	// dependen cosas que se deciden al arrancar, y `XDG_CURRENT_DESKTOP` es de la
	// sesión que lanzó el proceso, así que no cambia en vida.
	app.MarcarEscritorio(aplicacion, entorno.DeAqui())

	err := wails.Run(&options.App{
		Title: "Esfinge",

		// **Sin marco donde la barra de título la dibujamos nosotros** (ADR 0066),
		// que hoy es solo Windows. Es una opción de arranque y no se puede cambiar
		// después, por eso la regla está en Go y se lee aquí.
		//
		// Wails no deja a la ventana desnuda: con `Frameless` llama a
		// `ExtendFrameIntoClientArea`, así que la **sombra y las esquinas de
		// Windows 11 siguen siendo las del sistema**, y conserva el redimensionado
		// por los bordes. Lo que no da es el menú de Snap al pasar por Maximizar;
		// eso va aparte.
		Frameless: app.MarcoDe(runtime.GOOS).Propia,

		// La barra de menús, en español. Ver menu.go: los roles de Wails traen
		// los rótulos en inglés escritos a fuego, así que se construye entera.
		Menu: menuEnEspanol(func() context.Context { return ctx }, aplicacion),

		// Con barra lateral la ventana ya no puede encogerse tanto: 225 px se los
		// lleva la navegación, y al contenido hay que dejarle sitio para el panel
		// de 560 sin que se estruje.
		Width:     980,
		Height:    720,
		MinWidth:  760,
		MinHeight: 520,

		AssetServer: &assetserver.Options{Assets: interfaz.Recursos},

		// **Y con `ESFINGE_DIAGNOSTICO` puesto, Wails cuenta lo que hace.**
		//
		// No está encendido nunca de serie: escribe mucho y no le importa a nadie
		// que no esté arreglando algo. Existe porque hubo un arranque que no se
		// podía mirar desde aquí —la ventana no aparecía, el proceso seguía vivo y
		// no había ni error ni pánico— y lo único que faltaba era saber **por dónde
		// se quedaba**. Es la regla de siempre: antes de razonar sobre la causa,
		// ponerle voz al tramo.
		// **Las dos perillas, y no una.** En una compilación de producción Wails
		// no mira `LogLevel` sino `LogLevelProduction`, que de serie son sólo
		// errores — así que poner la primera dejaba el registro mudo y el fichero
		// sin crear. Se escribió así la primera vez y el diagnóstico no diagnosticó
		// nada: la herramienta de medir también necesita que alguien compruebe que
		// mide.
		Logger:             registroDeDiagnostico(),
		LogLevel:           nivelDeDiagnostico(),
		LogLevelProduction: nivelDeDiagnostico(),

		// Transparente para que se vea el vidrio de detrás. Donde no hay vidrio
		// esto no se llega a poner: ver más abajo.
		BackgroundColour: fondoDeLaVentana(vidrio),

		// EnableFileDrop entrega las rutas absolutas de lo que se suelte, que es
		// lo único que sirve: el arrastrar y soltar del navegador da un objeto sin
		// ruta, y por ahí no se llega al fichero desde Go.
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},

		OnStartup: func(c context.Context) {
			// **Y esto tampoco puede tumbar la ventana en silencio.**
			//
			// Wails llama aquí sin red: lo que entre en pánico dentro mata el
			// proceso, y como una aplicación con ventana no tiene consola, lo que
			// se ve es medio segundo de cursor y nada. Dos días costó acotar uno
			// así. No se recoge para seguir como si nada —lo que falle, falla— sino
			// para **dejarlo escrito** con su pila y que la ventana llegue a
			// abrirse: con la ventana delante se puede contar lo que pasa; sin
			// ella no se puede contar nada.
			defer func() {
				if r := recover(); r != nil {
					apuntarElArranqueFallido(fmt.Errorf("pánico al arrancar: %v\n%s", r, debug.Stack()))
				}
			}()

			ctx = c

			// **Y la ventana se enseña pase lo que pase.**
			//
			// Wails la muestra dentro de `navigationCompleted`, o sea **sólo si la
			// página termina de cargar**. Eso ata que haya ventana a que no falle
			// nada al cargar, y es una dependencia que no se sostiene: el 2026-11-23
			// el cliente estuvo con Esfinge **corriendo sin ventana**, con el
			// WebView2 arrancado y la interfaz ya hablando con Go —se la veía
			// contestar `VerPreferencias` y `Plataforma` en el registro— y aun así
			// nada en pantalla, ni en Alt+Tab, ni en la barra de tareas.
			//
			// Con esto, lo peor que puede pasar es una ventana en blanco: y una
			// ventana en blanco se puede cerrar, se puede contar y se puede
			// diagnosticar. Ninguna de las tres cosas se puede hacer con nada.
			//
			// Si Wails ya la ha enseñado, esto no hace nada.
			go func() {
				time.Sleep(3 * time.Second)
				ventanaWails.WindowShow(c)
			}()

			escritorio.Arrancar(c)
			aplicacion.Arrancar(c)

			// El remate del vidrio en macOS, que Wails deja a medias: la ventana
			// se queda opaca y el material no tiene nada que mezclar. Va aquí
			// porque hasta ahora no existía la ventana. Ver vidrio_darwin.go.
			if vidrio {
				ponerElVidrio()
			}

			// **El menú de Snap**, sobre el botón de maximizar que dibuja la
			// página (ADR 0066). Aquí por lo mismo que el vidrio: hasta ahora no
			// había ventana que encontrar. Fuera de Windows no hace nada.
			//
			// Las medidas salen de `app.MarcoDe`, que es el único sitio donde
			// están: escritas también allí, el día que cambie una, la ventana
			// invisible dejaría de caer sobre el botón. Y el nombre de la clase
			// es el de abajo, que es lo único por lo que Wails deja encontrar su
			// ventana: la v2 no da su identificador por ninguna parte.
			// Con `ESFINGE_SIN_SNAP` no se monta: es Win32 por nombre, y aquí no
			// hay ningún Windows donde ejecutarlo, así que tiene que poder
			// descartarse desde fuera sin recompilar.
			if os.Getenv("ESFINGE_SIN_SNAP") == "" {
				marco := app.MarcoDe(runtime.GOOS)
				ventana.Montar(claseDeLaVentana, marco.Alto, marco.AnchoBoton, func(estado string) {
					escritorio.Avisar(app.EventoBotonMaximizar, estado)
				})
			}
		},

		Bind: []any{aplicacion},

		Mac: &mac.Options{
			// Doble clic en un .esf. macOS no pasa el fichero como argumento
			// —lo entrega por un evento de Apple— y esto es lo que lo recoge.
			OnFileOpen: func(ruta string) {
				aplicacion.AlAbrirCon(ruta)
				escritorio.Avisar(app.EventoFicheroAbierto, ruta)
			},
			// Sin barra de título propia: el contenido llega hasta arriba y los
			// semáforos quedan encima de la barra lateral. Es lo que hacen Finder,
			// Correo y los Ajustes del sistema.
			//
			// **Hidden y no HiddenInset.** El «Inset» activa `UseToolbar`, y
			// entonces macOS dibuja **su propia banda de barra de herramientas**
			// justo donde va nuestro título: se ve un fondo que no cuadra con el
			// resto de la ventana. Aquí la barra de herramientas la dibujamos
			// nosotros, así que no queremos la del sistema.
			//
			// A cambio, **hay que declarar por dónde se arrastra la ventana** o se
			// queda clavada en la pantalla. Eso va en el CSS, con
			// «--wails-draggable» en la barra lateral y la de herramientas.
			TitleBar: mac.TitleBarHidden(),

			// El vidrio de macOS. WebviewIsTransparent deja pasar la luz a través
			// de la página, y WindowIsTranslucent pone detrás la vista de efecto
			// del sistema, que es la que desenfoca el escritorio de verdad.
			WebviewIsTransparent: vidrio,
			WindowIsTranslucent:  vidrio,
			About: &mac.AboutInfo{
				Title:   "Esfinge",
				Message: "Cifra y descifra secretos con una clave.\nDe Webcafeína.",
			},
		},
		Windows: &windows.Options{
			// **Con nombre propio**, que es lo único por lo que se puede volver a
			// encontrar esta ventana desde fuera: hace falta para colgarle encima
			// la del menú de Snap, y Wails v2 no expone su identificador.
			WindowClassName: claseDeLaVentanaSegunElEntorno(),

			// Mica es el equivalente en Windows 11: toma el fondo de escritorio,
			// muy desenfocado y quieto. En Windows 10 no existe y la ventana se
			// queda opaca, sin error y sin aviso, que es lo correcto.
			WebviewIsTransparent: vidrio,
			WindowIsTranslucent:  vidrio,
			BackdropType:         windows.Mica,
		},
	})
	if err != nil {
		apuntarElArranqueFallido(err)
		log.Fatal(err)
	}
}

// apuntarElArranqueFallido deja por escrito por qué no se ha podido abrir.
//
// **Un fallo mudo no se depura, y éste era mudo del todo.** Una aplicación con
// ventana no tiene consola: en Windows, `log.Fatal` escribe en una salida de error
// que no lee nadie, así que si `wails.Run` falla lo que se ve es **el cursor de
// espera medio segundo y nada más**. Le pasó al cliente con la 2.76.0 y desde aquí
// no hay forma de saber qué dijo, porque esta parte sólo corre en su máquina.
//
// Se escribe al lado de las preferencias, que es donde ya mira quien arregla, y se
// **añade** en vez de pisar: dos arranques fallidos seguidos dicen más que uno.
// Nada de lo que va aquí es secreto —es el error de la ventana, no de la bóveda— y
// si no se puede escribir, se calla: lo último que debe hacer un fallo de arranque
// es provocar otro.
func apuntarElArranqueFallido(err error) {
	dir, e := os.UserConfigDir()
	if e != nil {
		return
	}
	carpeta := filepath.Join(dir, "Esfinge")
	if e := os.MkdirAll(carpeta, 0o700); e != nil {
		return
	}
	f, e := os.OpenFile(filepath.Join(carpeta, "arranque.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if e != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s · Esfinge %s · %s/%s · no se ha podido abrir la ventana: %v\n",
		time.Now().Format(time.RFC3339), version, runtime.GOOS, runtime.GOARCH, err)
}

// fondoDeLaVentana es transparente cuando hay vidrio y nada cuando no lo hay.
//
// Devolver nil deja que Wails ponga el fondo de siempre, que es justo lo que
// tiene que pasar en Linux: un fondo transparente sin vidrio detrás no enseña el
// escritorio, enseña un agujero.
// El nombre de clase de la ventana. Ver `WindowClassName` y `ventana.Montar`.
const claseDeLaVentana = "EsfingeVentana"

func fondoDeLaVentana(vidrio bool) *options.RGBA {
	if !vidrio {
		return nil
	}
	return options.NewRGBA(0, 0, 0, 0)
}

// registroDeDiagnostico devuelve a dónde escribe Wails lo que hace.
//
// Sin la variable puesta, al sitio de siempre —que en una aplicación con ventana
// es ninguna parte—. Con ella, a un fichero al lado de las preferencias, que es
// donde ya mira quien arregla.
func registroDeDiagnostico() logger.Logger {
	if os.Getenv("ESFINGE_DIAGNOSTICO") == "" {
		return nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	carpeta := filepath.Join(dir, "Esfinge")
	if err := os.MkdirAll(carpeta, 0o700); err != nil {
		return nil
	}
	l := logger.NewFileLogger(filepath.Join(carpeta, "diagnostico.log"))

	// **Y se escribe una línea aquí mismo, antes de nada.**
	//
	// Para que «no hay fichero» signifique una sola cosa. Sin ella significa dos
	// —o el programa no llegó aquí, o llegó y no tenía nada que contar— y se tarda
	// una vuelta entera en saber cuál, que es lo que acaba de pasar.
	l.Info("Esfinge " + version + " · arrancando · " + runtime.GOOS + "/" + runtime.GOARCH)
	return l
}

func nivelDeDiagnostico() logger.LogLevel {
	if os.Getenv("ESFINGE_DIAGNOSTICO") == "" {
		return 0
	}
	return logger.TRACE
}

// claseDeLaVentanaSegunElEntorno deja quitar el nombre propio de la clase.
//
// Con `ESFINGE_SIN_CLASE` puesto, Wails usa el suyo de siempre. Sólo sirve para
// descartar: sin nombre propio no se puede encontrar la ventana desde fuera y el
// menú de Snap deja de funcionar.
func claseDeLaVentanaSegunElEntorno() string {
	if os.Getenv("ESFINGE_SIN_CLASE") != "" {
		return ""
	}
	return claseDeLaVentana
}
