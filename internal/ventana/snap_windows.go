//go:build windows

// Package ventana es lo que hay que hacer con la ventana y no cabe en la
// interfaz: hoy, el menú de Snap de Windows.
package ventana

import (
	"log"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// El menú de Snap de Windows 11 sobre el botón de maximizar que dibuja la
// interfaz (ADR 0066).
//
// # Por qué hace falta una ventana invisible
//
// Windows enseña el menú de Snap —las mitades y los cuadrantes— cuando la
// ventana contesta `HTMAXBUTTON` a `WM_NCHITTEST` sobre el punto del ratón.
// Pero nuestra barra de título la dibuja la página, así que ahí **quien tiene
// el ratón es la ventana hija de WebView2**, que no sabe contestar eso y ni
// siquiera deja que el mensaje llegue a la de arriba.
//
// La única salida —la misma que usan todos los complementos que lo resuelven—
// es **otra ventana hija, invisible, justo encima del botón**, que conteste
// `HTMAXBUTTON`. Como se queda con el ratón en ese recuadro, tiene que hacer
// además dos cosas que ya no puede hacer la página: avisar de cuándo está
// encima o pulsada —para que el botón se ilumine igual que los otros dos— y
// maximizar ella misma al soltar el clic.
//
// Invisible **porque no pinta** (pincel nulo), no por ser transparente:
// `WS_EX_LAYERED` le quitaría el ratón, que es justo lo que viene a recoger. Y
// `WS_CLIPSIBLINGS` es lo que la hace ganar a WebView2, que es su hermana.
//
// # Nada de esto se puede probar en el servidor de desarrollo
//
// Es Win32: compila aquí —cruzando a Windows sin cgo— y **solo se ejecuta en
// Windows**. Por eso está escrito entero en negativo: si no aparece la ventana,
// si no se registra la clase, si no se crea la hija, **se vuelve sin hacer
// nada** y Esfinge se queda exactamente como estaba. Lo peor que puede pasar es
// que no salga el menú de Snap; nunca que la ventana deje de funcionar. Es la
// lección de `vidrio_darwin.go`, que compiló en la máquina de la publicación y
// cerraba la aplicación al arrancar.
const (
	wsChild        = 0x40000000
	wsVisible      = 0x10000000
	wsClipSiblings = 0x04000000

	htMaxButton = 9

	wmSize          = 0x0005
	wmNCHitTest     = 0x0084
	wmNCMouseMove   = 0x00A0
	wmNCLButtonDown = 0x00A1
	wmNCLButtonUp   = 0x00A2
	wmNCMouseLeave  = 0x02A2
	wmDPIChanged    = 0x02E0

	tmeLeave      = 0x00000002
	tmeNonClient  = 0x00000010
	swMaximize    = 3
	swRestore     = 9
	swpAsync      = 0x4000
	swpShowWindow = 0x0040
	nullBrush     = 5
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	// **`comctl32` va por `NewLazyDLL` y no por `NewLazySystemDLL`, y en esa
	// palabra se fue una versión entera.**
	//
	// `SetWindowSubclass` y `DefSubclassProc` **sólo se exportan por su nombre en
	// la versión 6** de esa biblioteca; en la 5.82 están únicamente por número. La
	// versión 6 se consigue por el manifiesto —Wails pide los controles comunes— y
	// esa redirección la resuelve el contexto de activación **al cargar por
	// nombre**. `NewLazySystemDLL` carga con `LOAD_LIBRARY_SEARCH_SYSTEM32`, que
	// **se salta ese contexto** y trae la 5.82 de `System32`; entonces el `.Call()`
	// no encuentra la función y **entra en pánico**.
	//
	// Lo que se ve desde fuera es lo peor: la aplicación no abre, no sale ninguna
	// ventana y **el Visor de eventos no apunta nada**, porque un pánico de Go no
	// es un fallo de aplicación de Windows. Medio segundo de cursor y nada. Así
	// estuvo de la 2.72.0 a la 2.76.0, en todos los Windows: aquí no se puede
	// ejecutar ninguno y las puertas sólo dicen que enlaza.
	//
	// Las otras tres sí van por `System32` a propósito: lo que se pide de ellas
	// está en todas las versiones y cargarlas de ahí es lo seguro.
	comctl32 = windows.NewLazyDLL("comctl32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	findWindowW       = user32.NewProc("FindWindowW")
	registerClassExW  = user32.NewProc("RegisterClassExW")
	createWindowExW   = user32.NewProc("CreateWindowExW")
	defWindowProcW    = user32.NewProc("DefWindowProcW")
	getClientRect     = user32.NewProc("GetClientRect")
	setWindowPos      = user32.NewProc("SetWindowPos")
	showWindow        = user32.NewProc("ShowWindow")
	isZoomed          = user32.NewProc("IsZoomed")
	trackMouseEvent   = user32.NewProc("TrackMouseEvent")
	getDpiForWindow   = user32.NewProc("GetDpiForWindow")
	getStockObject    = gdi32.NewProc("GetStockObject")
	setWindowSubclass = comctl32.NewProc("SetWindowSubclass")
	defSubclassProc   = comctl32.NewProc("DefSubclassProc")
	getModuleHandleW  = kernel32.NewProc("GetModuleHandleW")
)

type rect struct{ izq, arriba, der, abajo int32 }

type wndClassExW struct {
	tamano     uint32
	estilo     uint32
	proc       uintptr
	extraClase int32
	extraVent  int32
	instancia  windows.Handle
	icono      windows.Handle
	cursor     windows.Handle
	fondo      windows.Handle
	menu       *uint16
	clase      *uint16
	iconoChico windows.Handle
}

type trackMouse struct {
	tamano  uint32
	banales uint32
	ventana windows.HWND
	espera  uint32
}

var (
	padre      atomic.Uintptr
	hija       atomic.Uintptr
	encima     atomic.Bool
	pulsado    atomic.Bool
	altoBarra  atomic.Int32
	anchoBoton atomic.Int32
	avisar     atomic.Pointer[func(string)]
)

// Montar pone la ventana invisible sobre el botón de maximizar.
//
// `clase` es el nombre de clase que se le dio a la ventana en `wails.Run`, que
// es lo único por lo que se la puede encontrar: Wails v2 no da el identificador
// de su ventana por ninguna parte. `alto` y `ancho` son las medidas de la barra
// y de un botón, **y vienen de `app.MarcoDe`**, que es el único sitio donde
// están: con ellas escritas también aquí, el día que cambie una la ventana de
// Snap dejaría de caer sobre el botón.
//
// `alAvisar` recibe «encima», «pulsado» o «fuera» para que la interfaz ilumine
// el botón, porque desde que hay una ventana encima la página ya no ve el ratón.
func Montar(clase string, alto, ancho int, alAvisar func(string)) {
	// **Y pase lo que pase aquí dentro, la ventana se abre.**
	//
	// Esto es una comodidad: el menú de Snap al pasar por encima de «Maximizar».
	// Lo que hay debajo son llamadas a Win32 por nombre, y una función que no esté
	// **entra en pánico** — que es exactamente lo que dejó a Esfinge sin abrirse
	// durante cinco versiones. Una conveniencia no puede tener la capacidad de
	// impedir que el programa arranque, y menos una que aquí no se puede ejecutar.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("esfinge: el menú de Snap no se ha podido montar: %v", r)
		}
	}()

	if clase == "" || alto <= 0 || ancho <= 0 || alAvisar == nil {
		return
	}
	nombre, err := windows.UTF16PtrFromString(clase)
	if err != nil {
		return
	}
	h, _, _ := findWindowW.Call(uintptr(unsafe.Pointer(nombre)), 0)
	if h == 0 {
		return
	}

	mia, err := windows.UTF16PtrFromString(clase + "Snap")
	if err != nil {
		return
	}
	instancia, _, _ := getModuleHandleW.Call(0)
	pincel, _, _ := getStockObject.Call(nullBrush)
	registro := wndClassExW{
		tamano:    uint32(unsafe.Sizeof(wndClassExW{})),
		proc:      windows.NewCallback(deLaHija),
		instancia: windows.Handle(instancia),
		fondo:     windows.Handle(pincel),
		clase:     mia,
	}
	// Si ya estaba registrada —no debería, esto se llama una vez— se sigue: lo
	// que importa es que exista al crear la ventana.
	registerClassExW.Call(uintptr(unsafe.Pointer(&registro)))

	nueva, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(mia)),
		uintptr(unsafe.Pointer(mia)),
		wsChild|wsVisible|wsClipSiblings,
		0, 0, 0, 0,
		h, 0, instancia, 0,
	)
	if nueva == 0 {
		return
	}

	padre.Store(h)
	hija.Store(nueva)
	altoBarra.Store(int32(alto))
	anchoBoton.Store(int32(ancho))
	avisar.Store(&alAvisar)

	// Para recolocarla cuando la ventana cambie de tamaño o de pantalla.
	setWindowSubclass.Call(h, windows.NewCallback(delPadre), 1, 0)
	colocar(h)
}

// colocar la pone encima del segundo botón por la derecha —entre «Maximizar» y
// el borde solo está «Cerrar»—, a la escala de la pantalla en la que esté.
func colocar(h uintptr) {
	n := hija.Load()
	if n == 0 {
		return
	}
	var r rect
	if ok, _, _ := getClientRect.Call(h, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return
	}
	ppp, _, _ := getDpiForWindow.Call(h)
	if ppp == 0 {
		ppp = 96
	}
	escala := func(v int32) int32 { return int32((uint32(v)*uint32(ppp) + 48) / 96) }
	ancho := escala(anchoBoton.Load())
	setWindowPos.Call(
		n, 0,
		uintptr(r.der-ancho*2), 0,
		uintptr(ancho), uintptr(escala(altoBarra.Load())),
		swpAsync|swpShowWindow,
	)
}

func decir(que string) {
	if f := avisar.Load(); f != nil {
		(*f)(que)
	}
}

func delPadre(h uintptr, msg uint32, wp, lp uintptr, id, datos uintptr) uintptr {
	if msg == wmSize || msg == wmDPIChanged {
		colocar(h)
	}
	r, _, _ := defSubclassProc.Call(h, uintptr(msg), wp, lp)
	return r
}

func deLaHija(h uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case wmNCHitTest:
		// Lo único que hace salir el menú de Snap.
		return htMaxButton

	case wmNCMouseMove:
		if !encima.Swap(true) {
			decir("encima")
		}
		// Sin esto no llega el aviso de cuándo se va.
		seguir := trackMouse{
			tamano:  uint32(unsafe.Sizeof(trackMouse{})),
			banales: tmeLeave | tmeNonClient,
			ventana: windows.HWND(h),
		}
		trackMouseEvent.Call(uintptr(unsafe.Pointer(&seguir)))
		return 0

	case wmNCMouseLeave:
		encima.Store(false)
		pulsado.Store(false)
		decir("fuera")
		return 0

	case wmNCLButtonDown:
		pulsado.Store(true)
		decir("pulsado")
		return 0

	case wmNCLButtonUp:
		if pulsado.Swap(false) {
			p := padre.Load()
			z, _, _ := isZoomed.Call(p)
			if z != 0 {
				showWindow.Call(p, swRestore)
			} else {
				showWindow.Call(p, swMaximize)
			}
		}
		if encima.Load() {
			decir("encima")
		} else {
			decir("fuera")
		}
		return 0
	}
	r, _, _ := defWindowProcW.Call(h, uintptr(msg), wp, lp)
	return r
}
