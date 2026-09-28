//go:build windows

package llavero

// Windows Hello (fase C, C3, ADR 0044).
//
// # Lo primero, porque decide cómo está escrito todo lo demás
//
// **Esto está escrito a ciegas.** Nadie ha ejecutado nunca Esfinge en un Windows:
// lo único que se ejecuta en uno es la máquina de GitHub al publicar. Aquí se
// compila —es Go puro, sin cgo, así que el cruce de `make comprobar` lo compila
// para los dos objetivos de Windows— y **eso es todo lo que se puede decir**. La
// lección de `vidrio_darwin.go` vale entera: compilar no dice que arranque.
//
// De ahí la regla que gobierna este fichero: **cada paso que falle lo dice, con
// su `HRESULT` en hexadecimal**. No hay nada que tragarse un error, porque el día
// que alguien lo abra en un Windows el mensaje es lo único que va a haber. Es la
// misma lección que costó el fallo mudo de la 2.27.3 en el Mac.
//
// # Las dos piezas, otra vez
//
// Igual que en macOS, porque el problema es el mismo:
//
//  1. **El consentimiento**, con `UserConsentVerifier`: Windows enseña el diálogo
//     de Hello —cara, huella o PIN— y devuelve un sí o un no. En una aplicación
//     de escritorio **no se puede llamar a la versión normal**: hace falta la
//     interfaz de interoperación `IUserConsentVerifierInterop`, que recibe el
//     identificador de la ventana. Ésa es la diferencia con una aplicación de la
//     tienda, y es la que hace que este fichero tenga tanta fontanería.
//  2. **El secreto, en el Administrador de credenciales** (`CredWriteW`), que lo
//     guarda cifrado con las credenciales de la sesión. Un disco copiado no se lo
//     lleva, que es la propiedad que se buscaba en macOS con el llavero.
//
// # Lo que aquí protege menos que en macOS, y hay que decirlo
//
// En macOS, el llavero le pide permiso al usuario cuando otra aplicación quiere
// leer un elemento que no creó. **En Windows no hay nada de eso**: una credencial
// genérica la lee cualquier proceso que corra como tú, sin preguntar. Así que el
// diálogo de Hello es un portero que solo vigila **nuestra** puerta.
//
// Eso **no cambia lo que la ADR 0044 promete** —«protege de quien se siente
// delante de tu ordenador desbloqueado, no de un programa que corra como tú»—,
// pero conviene saber que en Windows el margen es más estrecho todavía. Es
// coherente con lo que Microsoft dice de `KeyCredentialManager`, que tampoco ata
// la credencial a la aplicación sino a la cuenta de usuario.

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	combase  = windows.NewLazySystemDLL("combase.dll")
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")

	roInitialize           = combase.NewProc("RoInitialize")
	roGetActivationFactory = combase.NewProc("RoGetActivationFactory")
	windowsCreateString    = combase.NewProc("WindowsCreateString")
	windowsDeleteString    = combase.NewProc("WindowsDeleteString")

	credWrite  = advapi32.NewProc("CredWriteW")
	credRead   = advapi32.NewProc("CredReadW")
	credDelete = advapi32.NewProc("CredDeleteW")
	credFree   = advapi32.NewProc("CredFree")

	getForegroundWindow = user32.NewProc("GetForegroundWindow")
	findWindow          = user32.NewProc("FindWindowW")
)

// Las tres interfaces que hacen falta, y **una sola de sus GUID es discutible**.
//
// `IAsyncInfo` y las dos de `UserConsentVerifier` son fijas y están publicadas.
// La cuarta, `IAsyncOperation<UserConsentVerificationResult>`, es de un tipo
// **parametrizado**: su identificador sale de una fórmula sobre el tipo genérico y
// el de dentro, no de una tabla. Va escrita a mano con el valor conocido, y **si
// algo de este fichero está mal, empieza por mirar ésta**: la llamada al interop
// fallaría con `E_NOINTERFACE` (0x80004002), que es lo que dirá el mensaje.
var (
	iidAsyncInfo             = windows.GUID{Data1: 0x00000036, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidConsentInterop        = windows.GUID{Data1: 0x39E050C3, Data2: 0x4E74, Data3: 0x441A, Data4: [8]byte{0x8D, 0xC0, 0xB8, 0x11, 0x04, 0xDF, 0x94, 0x9C}}
	iidConsentStatics        = windows.GUID{Data1: 0xAF4F3F91, Data2: 0x564C, Data3: 0x4DDC, Data4: [8]byte{0xB8, 0xB5, 0x97, 0x34, 0x47, 0x62, 0x7C, 0x65}}
	iidAsyncOpDeVerificacion = windows.GUID{Data1: 0xD26B03C3, Data2: 0xCB57, Data3: 0x53B3, Data4: [8]byte{0xB8, 0xFD, 0x6A, 0x0E, 0x7C, 0x21, 0xC5, 0xC0}}
)

const (
	claseDelVerificador = "Windows.Security.Credentials.UI.UserConsentVerifier"

	roInitMultihilo = 1

	// Estados de un `IAsyncInfo`. Lo que se espera es dejar de ver `empezado`.
	asyncEmpezado  = 0
	asyncTerminado = 1

	// `UserConsentVerificationResult`: 0 es que sí, y lo demás son maneras de que
	// no. No se distinguen una por una a propósito — para esta pantalla, «no te ha
	// reconocido» y «has cancelado» acaban en el mismo sitio: la maestra.
	consentimientoVerificado = 0

	// `UserConsentVerifierAvailability`: 0 es que hay Hello configurado.
	helloDisponible = 0

	credTipoGenerico  = 1
	credPersisteLocal = 2
	errorNoEncontrado = 1168 // ERROR_NOT_FOUND
)

// destino es cómo sale Esfinge en el Administrador de credenciales de Windows.
// Se ve —«Panel de control → Administrador de credenciales»—, así que se escribe
// para quien lo lea, igual que el nombre del elemento del llavero en macOS.
const destino = "Esfinge (bóveda)"

type deWindows struct{}

func delSistema() Llavero { return deWindows{} }

func (deWindows) Nombre() string {
	if hayHello() {
		return "Windows Hello"
	}
	return ""
}

func (l deWindows) Hay() bool { return hayHello() }

// hayHello pregunta si este equipo tiene Hello configurado.
//
// **Configurado, no soportado**: un Windows con lector de huella pero sin PIN ni
// huella dados de alta contesta que no, que es justo lo que hay que saber para no
// ofrecer un botón muerto.
func hayHello() bool {
	var disponible int32
	if err := enWinRT(func() error {
		fabrica, err := fabricaDe(claseDelVerificador, &iidConsentStatics)
		if err != nil {
			return err
		}
		defer soltar(fabrica)

		vt := (*vtblConsentStatics)(*(*unsafe.Pointer)(fabrica))
		var op unsafe.Pointer
		if hr, _, _ := syscall.SyscallN(vt.CheckAvailabilityAsync, uintptr(fabrica), uintptr(unsafe.Pointer(&op))); hr != 0 {
			return fallo("CheckAvailabilityAsync", hr)
		}
		defer soltar(op)
		return esperarYLeer(op, &disponible)
	}); err != nil {
		return false
	}
	return disponible == helloDisponible
}

func (deWindows) Guardar(id string, secreto []byte) error {
	if len(secreto) == 0 {
		return fmt.Errorf("No hay nada que guardar en el Administrador de credenciales")
	}
	nombre, err := windows.UTF16PtrFromString(destino + " · " + id)
	if err != nil {
		return err
	}
	comentario, err := windows.UTF16PtrFromString("Llave para abrir la bóveda con Windows Hello")
	if err != nil {
		return err
	}
	usuario, err := windows.UTF16PtrFromString("Esfinge")
	if err != nil {
		return err
	}

	c := credencial{
		Type:               credTipoGenerico,
		TargetName:         nombre,
		Comment:            comentario,
		CredentialBlobSize: uint32(len(secreto)),
		CredentialBlob:     &secreto[0],
		Persist:            credPersisteLocal,
		UserName:           usuario,
	}
	r, _, e := credWrite.Call(uintptr(unsafe.Pointer(&c)), 0)
	// `CredWriteW` devuelve TRUE/FALSE, no un HRESULT: el porqué está en
	// GetLastError, que es lo que trae `e`.
	runtime.KeepAlive(secreto)
	if r == 0 {
		return fmt.Errorf("El Administrador de credenciales no ha guardado la llave (%v)", e)
	}
	return nil
}

func (deWindows) Leer(id, motivo string) ([]byte, error) {
	// **Primero Hello y después la credencial**, como en macOS: al revés, un fallo
	// del almacén se vería después de haber hecho poner la cara para nada.
	vale, err := pedirHello(motivo)
	if err != nil {
		return nil, err
	}
	if !vale {
		return nil, ErrNoQuiso
	}

	nombre, err := windows.UTF16PtrFromString(destino + " · " + id)
	if err != nil {
		return nil, err
	}
	var puntero unsafe.Pointer
	r, _, e := credRead.Call(uintptr(unsafe.Pointer(nombre)), credTipoGenerico, 0, uintptr(unsafe.Pointer(&puntero)))
	if r == 0 {
		if errno, vale := e.(syscall.Errno); vale && errno == errorNoEncontrado {
			return nil, ErrNoEsta
		}
		return nil, fmt.Errorf("El Administrador de credenciales no ha devuelto la llave (%v)", e)
	}
	defer credFree.Call(uintptr(puntero))

	c := (*credencial)(puntero)
	if c.CredentialBlobSize == 0 || c.CredentialBlob == nil {
		return nil, ErrNoEsta
	}
	// Copia propia: lo de dentro lo libera `CredFree` en cuanto salgamos.
	return append([]byte(nil), unsafe.Slice(c.CredentialBlob, c.CredentialBlobSize)...), nil
}

func (deWindows) Borrar(id string) error {
	nombre, err := windows.UTF16PtrFromString(destino + " · " + id)
	if err != nil {
		return err
	}
	r, _, e := credDelete.Call(uintptr(unsafe.Pointer(nombre)), credTipoGenerico, 0)
	if r == 0 {
		if errno, vale := e.(syscall.Errno); vale && errno == errorNoEncontrado {
			return nil // que no estuviera no es un error
		}
		return fmt.Errorf("El Administrador de credenciales no ha borrado la llave (%v)", e)
	}
	return nil
}

// pedirHello enseña el diálogo del sistema y espera.
//
// **Necesita el identificador de una ventana**, y ahí está la otra cosa que solo
// dirá un Windows: se usa la que esté delante, que es la nuestra porque esto solo
// ocurre justo después de que alguien pulse en ella. Si no la hubiera, se busca
// por el título. Sin ventana, Windows no sabe dónde poner el diálogo y la llamada
// falla en vez de enseñar nada.
func pedirHello(motivo string) (bool, error) {
	var resultado int32
	err := enWinRT(func() error {
		ventana := ventanaDeEsfinge()
		if ventana == 0 {
			return fmt.Errorf("No se ha encontrado la ventana de Esfinge para enseñar Windows Hello")
		}
		fabrica, err := fabricaDe(claseDelVerificador, &iidConsentInterop)
		if err != nil {
			return err
		}
		defer soltar(fabrica)

		mensaje, borrar, err := cadenaWinRT(motivo)
		if err != nil {
			return err
		}
		defer borrar()

		vt := (*vtblConsentInterop)(*(*unsafe.Pointer)(fabrica))
		var op unsafe.Pointer
		hr, _, _ := syscall.SyscallN(vt.RequestVerificationForWindowAsync,
			uintptr(fabrica), ventana, mensaje,
			uintptr(unsafe.Pointer(&iidAsyncOpDeVerificacion)), uintptr(unsafe.Pointer(&op)))
		if hr != 0 {
			return fallo("RequestVerificationForWindowAsync", hr)
		}
		defer soltar(op)
		return esperarYLeer(op, &resultado)
	})
	if err != nil {
		return false, err
	}
	return resultado == consentimientoVerificado, nil
}

func ventanaDeEsfinge() uintptr {
	if h, _, _ := getForegroundWindow.Call(); h != 0 {
		return h
	}
	titulo, err := windows.UTF16PtrFromString("Esfinge")
	if err != nil {
		return 0
	}
	h, _, _ := findWindow.Call(0, uintptr(unsafe.Pointer(titulo)))
	return h
}

// ---------------------------------------------------------------- fontanería

// enWinRT corre algo con WinRT arrancado **en un hilo fijo**.
//
// `RoInitialize` vale para el hilo que la llama, y una gorrutina de Go salta de
// hilo entre dos llamadas: sin `LockOSThread` esto funcionaría a veces, que es la
// peor de las opciones.
func enWinRT(hacer func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// S_FALSE (1) es «ya estaba arrancado en este hilo», y no es un error. Wails
	// puede haberlo hecho antes que nosotros.
	if hr, _, _ := roInitialize.Call(roInitMultihilo); hr != 0 && hr != 1 && hr != 0x80010106 {
		return fallo("RoInitialize", hr)
	}
	return hacer()
}

func cadenaWinRT(s string) (uintptr, func(), error) {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return 0, func() {}, err
	}
	var h uintptr
	// El largo va **sin el cero final**, que WinRT lo pone por su cuenta.
	hr, _, _ := windowsCreateString.Call(uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&h)))
	if hr != 0 {
		return 0, func() {}, fallo("WindowsCreateString", hr)
	}
	return h, func() { windowsDeleteString.Call(h) }, nil
}

func fabricaDe(clase string, iid *windows.GUID) (unsafe.Pointer, error) {
	nombre, borrar, err := cadenaWinRT(clase)
	if err != nil {
		return nil, err
	}
	defer borrar()

	var fabrica unsafe.Pointer
	hr, _, _ := roGetActivationFactory.Call(nombre, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&fabrica)))
	if hr != 0 {
		return nil, fallo("RoGetActivationFactory("+clase+")", hr)
	}
	return fabrica, nil
}

// esperarYLeer espera a que termine una operación asíncrona y saca su resultado.
//
// **Se espera dando vueltas, no con una retrollamada**, y es a propósito: una
// retrollamada exige construir una tabla de métodos de COM desde Go y dársela a
// Windows para que la llame en otro hilo. Aquí, donde no se puede ejecutar nada
// para comprobarlo, eso es mucho más de lo que se puede sostener; dar vueltas
// cada pocos milisegundos mientras alguien mira un diálogo no le cuesta nada a
// nadie.
func esperarYLeer(op unsafe.Pointer, fuera *int32) error {
	info, err := comoInfo(op)
	if err != nil {
		return err
	}
	defer soltar(info)

	vtInfo := (*vtblAsyncInfo)(*(*unsafe.Pointer)(info))
	for {
		var estado int32
		if hr, _, _ := syscall.SyscallN(vtInfo.GetStatus, uintptr(info), uintptr(unsafe.Pointer(&estado))); hr != 0 {
			return fallo("IAsyncInfo::get_Status", hr)
		}
		if estado != asyncEmpezado {
			if estado != asyncTerminado {
				// Cancelada o con error: se mira el porqué, que es lo único que
				// va a quedar cuando esto falle en una máquina que no tenemos.
				var codigo int32
				syscall.SyscallN(vtInfo.GetErrorCode, uintptr(info), uintptr(unsafe.Pointer(&codigo)))
				return fmt.Errorf("Windows Hello no ha terminado (estado %d, error 0x%08X)", estado, uint32(codigo))
			}
			break
		}
		windows.SleepEx(20, false)
	}

	vtOp := (*vtblAsyncOperation)(*(*unsafe.Pointer)(op))
	if hr, _, _ := syscall.SyscallN(vtOp.GetResults, uintptr(op), uintptr(unsafe.Pointer(fuera))); hr != 0 {
		return fallo("GetResults", hr)
	}
	return nil
}

func comoInfo(op unsafe.Pointer) (unsafe.Pointer, error) {
	vt := (*vtblInspectable)(*(*unsafe.Pointer)(op))
	var info unsafe.Pointer
	hr, _, _ := syscall.SyscallN(vt.QueryInterface, uintptr(op), uintptr(unsafe.Pointer(&iidAsyncInfo)), uintptr(unsafe.Pointer(&info)))
	if hr != 0 {
		return nil, fallo("QueryInterface(IAsyncInfo)", hr)
	}
	return info, nil
}

// soltar suelta una referencia de COM. **Los punteros de COM viajan como
// `unsafe.Pointer` y no como `uintptr` en todo este fichero**, y no es una manía:
// `go vet` marca cada vuelta de `uintptr` a puntero —«possible misuse»— y tiene
// razón, aunque aquí lo apuntado sea memoria de Windows y no del recolector de
// Go. La puerta de la publicación corre `GOOS=windows go vet`, así que escribirlo
// de la otra forma no habría llegado ni a compilarse en la máquina de Windows.
func soltar(p unsafe.Pointer) {
	if p == nil {
		return
	}
	vt := (*vtblInspectable)(*(*unsafe.Pointer)(p))
	syscall.SyscallN(vt.Release, uintptr(p))
}

// fallo escribe el HRESULT en hexadecimal, que es como se busca.
func fallo(donde string, hr uintptr) error {
	return fmt.Errorf("Windows Hello ha fallado en %s (0x%08X)", donde, uint32(hr))
}

// --------------------------------------------------------- tablas de métodos

// El orden **es** la interfaz: COM llama por posición, no por nombre. Una entrada
// de más o de menos no da un error de compilación, da una llamada a la función
// equivocada. Cada tabla empieza por la de `IInspectable`, que a su vez empieza
// por la de `IUnknown`.
type vtblInspectable struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	GetIids             uintptr
	GetRuntimeClassName uintptr
	GetTrustLevel       uintptr
}

type vtblConsentInterop struct {
	vtblInspectable
	RequestVerificationForWindowAsync uintptr
}

type vtblConsentStatics struct {
	vtblInspectable
	CheckAvailabilityAsync   uintptr
	RequestVerificationAsync uintptr
}

type vtblAsyncInfo struct {
	vtblInspectable
	GetID        uintptr
	GetStatus    uintptr
	GetErrorCode uintptr
	Cancel       uintptr
	Close        uintptr
}

type vtblAsyncOperation struct {
	vtblInspectable
	PutCompleted uintptr
	GetCompleted uintptr
	GetResults   uintptr
}
