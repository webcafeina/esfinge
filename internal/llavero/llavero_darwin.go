//go:build darwin

package llavero

// Touch ID en macOS (fase C, `docs/desbloqueo-del-sistema.md`).
//
// **Este fichero no se puede compilar en la máquina de desarrollo**, como
// `vidrio_darwin.go`: lo compila el trabajo de macOS de la publicación, y la
// lección de aquél es que verlo en verde ahí **solo dice que compila, no que
// arranque**. La 2.9.1 salió con un diagnóstico que compilaba y cerraba la
// aplicación nada más abrirla.
//
// # Las dos piezas, y por qué son dos
//
// Sin firmar la aplicación —decisión del cliente, ADR 0012— **el camino fuerte de
// macOS está cerrado**: guardar una llave con control de acceso biométrico o en
// el Secure Enclave exige el llavero de protección de datos, y ése pide la
// entitlement `keychain-access-groups`, que solo lleva una compilación firmada con
// un perfil de aprovisionamiento. Sin ella, `SecItemAdd` devuelve -34018.
//
// Así que se hace con dos piezas que sí funcionan sin firma:
//
//  1. **El secreto vive en el llavero de inicio de sesión**, el de toda la vida
//     (`kSecClassGenericPassword`, sin `kSecUseDataProtectionKeychain`). No es tan
//     bueno como el Secure Enclave, pero **está cifrado con la contraseña de
//     macOS**: quien copie la carpeta de alguien no se lo lleva. Un fichero suelto
//     junto a la bóveda sería mucho peor, y es la alternativa que se descartó.
//  2. **La huella se pide aparte**, con `LAContext`. Eso devuelve un sí o un no,
//     no una llave: por eso esto es **un cerrojo y no una llave**, y por eso hay
//     que decirlo donde se activa.
//
// # Lo que puede pasar en un Mac de verdad y aquí no se ve
//
//   - **Que el llavero pregunte al actualizar.** La lista de aplicaciones de
//     confianza de un elemento del llavero se ata a la firma de quien lo guardó, y
//     Esfinge no está firmada. Puede que macOS pida permiso la primera vez y otra
//     vez después de cada actualización. Si pasa, se acepta con «Permitir
//     siempre»; si resulta insoportable, está escrito en el documento de la fase.
//   - **Que `LAContext` no funcione sin bundle.** Necesita uno con identificador,
//     porque el diálogo dice «“Esfinge” está intentando…» y lo saca de ahí. La
//     aplicación empaquetada lo tiene; un binario suelto, no.
//
// # Las trampas del Objective-C que ya costaron una versión
//
//   - **Nada de devolver el `UTF8String` de una cadena autoliberada**: deja un
//     puntero colgando en cuanto se vacía la piscina. Lo que sale de aquí hacia Go
//     va con `malloc` y lo libera Go.
//   - **Todo dentro de `@autoreleasepool`**, que aquí no hay bucle de eventos que
//     la vacíe por nosotros.
//   - Y el bloque de `evaluatePolicy` **corre en otra cola**: se espera con un
//     semáforo, y el resultado se escribe en memoria que vive más que el bloque.

/*
#cgo CFLAGS: -x objective-c -fmodules -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework Security -framework LocalAuthentication

#import <Foundation/Foundation.h>
#import <Security/Security.h>
#import <LocalAuthentication/LocalAuthentication.h>
#include <stdlib.h>
#include <string.h>

// esfingeBiometria: 0 si no hay, 1 Touch ID, 2 Face ID.
int esfingeBiometria(void) {
	@autoreleasepool {
		LAContext *c = [[LAContext alloc] init];
		NSError *err = nil;
		if (![c canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:&err]) {
			return 0;
		}
		// `biometryType` solo vale después de preguntar por la política, que es lo
		// que se acaba de hacer. Preguntarlo antes devuelve «ninguna».
		if (c.biometryType == LABiometryTypeFaceID) return 2;
		if (c.biometryType == LABiometryTypeTouchID) return 1;
		return 0;
	}
}

// esfingeHuella pide identificarse. 1 si sí; 0 si no o si se ha cancelado.
//
// **Se espera para siempre a propósito.** El diálogo del sistema siempre termina
// —la persona acepta, cancela, o se agotan los intentos—, y poner aquí un plazo
// obligaría a que el bloque escribiera en memoria que ya no existe.
int esfingeHuella(const char *motivo) {
	@autoreleasepool {
		LAContext *c = [[LAContext alloc] init];
		// Sin «Introducir contraseña»: lo que esto promete es la huella. Quien no
		// pueda usarla tiene la contraseña maestra de Esfinge, que es la de verdad.
		c.localizedFallbackTitle = @"";
		NSString *razon = [NSString stringWithUTF8String:motivo];

		__block int vale = 0;
		dispatch_semaphore_t listo = dispatch_semaphore_create(0);
		[c evaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics
			  localizedReason:razon
						reply:^(BOOL ok, NSError *_Nullable error) {
							vale = ok ? 1 : 0;
							dispatch_semaphore_signal(listo);
						}];
		dispatch_semaphore_wait(listo, DISPATCH_TIME_FOREVER);
		return vale;
	}
}

static NSDictionary *esfingeBusqueda(const char *servicio, const char *cuenta) {
	return @{
		(__bridge id)kSecClass : (__bridge id)kSecClassGenericPassword,
		(__bridge id)kSecAttrService : [NSString stringWithUTF8String:servicio],
		(__bridge id)kSecAttrAccount : [NSString stringWithUTF8String:cuenta],
	};
}

// esfingeGuardar deja el secreto en el llavero de inicio de sesión. 0 si bien.
int esfingeGuardar(const char *servicio, const char *cuenta, const void *datos, int n) {
	@autoreleasepool {
		NSData *valor = [NSData dataWithBytes:datos length:(NSUInteger)n];
		NSMutableDictionary *q = [esfingeBusqueda(servicio, cuenta) mutableCopy];
		// Primero se intenta actualizar: añadir encima de uno que ya está devuelve
		// errSecDuplicateItem, y entonces el secreto viejo se quedaría.
		OSStatus st = SecItemUpdate((__bridge CFDictionaryRef)q,
									(__bridge CFDictionaryRef) @{(__bridge id)kSecValueData : valor});
		if (st == errSecItemNotFound) {
			q[(__bridge id)kSecValueData] = valor;
			q[(__bridge id)kSecAttrLabel] = @"Esfinge";
			q[(__bridge id)kSecAttrDescription] = @"Llave para abrir la bóveda con Touch ID";
			st = SecItemAdd((__bridge CFDictionaryRef)q, NULL);
		}
		return (int)st;
	}
}

// esfingeLeer devuelve el secreto con malloc. Quien llame lo libera.
int esfingeLeer(const char *servicio, const char *cuenta, void **fuera, int *n) {
	@autoreleasepool {
		NSMutableDictionary *q = [esfingeBusqueda(servicio, cuenta) mutableCopy];
		q[(__bridge id)kSecReturnData] = @YES;
		q[(__bridge id)kSecMatchLimit] = (__bridge id)kSecMatchLimitOne;
		CFTypeRef salida = NULL;
		OSStatus st = SecItemCopyMatching((__bridge CFDictionaryRef)q, &salida);
		if (st != errSecSuccess) return (int)st;

		NSData *datos = (__bridge_transfer NSData *)salida;
		// **Copia con malloc y no el puntero de la NSData**: al vaciarse la piscina
		// ese puntero deja de valer, y eso no se nota hasta que se nota.
		//
		// El `+ 1` es por si algún día viniera vacío: `malloc(0)` puede devolver NULL
		// sin que haya pasado nada malo, y eso se leería aquí como un fallo.
		void *copia = malloc(datos.length + 1);
		if (copia == NULL) return -1;
		memcpy(copia, datos.bytes, datos.length);
		*fuera = copia;
		*n = (int)datos.length;
		return 0;
	}
}

int esfingeBorrar(const char *servicio, const char *cuenta) {
	@autoreleasepool {
		OSStatus st = SecItemDelete((__bridge CFDictionaryRef)esfingeBusqueda(servicio, cuenta));
		if (st == errSecItemNotFound) return 0; // que no estuviera no es un error
		return (int)st;
	}
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// servicio es cómo sale Esfinge en Acceso a Llaveros. Se ve, así que se escribe
// para quien lo lea.
const servicio = "Esfinge (bóveda)"

type deMacOS struct{}

func delSistema() Llavero { return deMacOS{} }

func (deMacOS) Nombre() string {
	switch C.esfingeBiometria() {
	case 1:
		return "Touch ID"
	case 2:
		return "Face ID"
	default:
		return ""
	}
}

func (l deMacOS) Hay() bool { return l.Nombre() != "" }

func (deMacOS) Guardar(id string, secreto []byte) error {
	if len(secreto) == 0 {
		return fmt.Errorf("No hay nada que guardar en el llavero")
	}
	s, c := C.CString(servicio), C.CString(id)
	defer C.free(unsafe.Pointer(s))
	defer C.free(unsafe.Pointer(c))
	st := C.esfingeGuardar(s, c, unsafe.Pointer(&secreto[0]), C.int(len(secreto)))
	if st != 0 {
		// **El número sale tal cual a propósito.** Aquí no se puede reproducir
		// ninguno de estos fallos, así que quien se encuentre uno en un Mac
		// necesita poder buscarlo: -25299 es «ya existe», -34018 «falta una
		// entitlement», -25293 «no se ha podido autenticar con el llavero».
		return fmt.Errorf("El llavero de macOS no ha guardado la llave (%d)", int(st))
	}
	return nil
}

func (deMacOS) Leer(id, motivo string) ([]byte, error) {
	m := C.CString(motivo)
	defer C.free(unsafe.Pointer(m))
	// **Primero la huella y después el llavero.** Al revés, un fallo del llavero
	// se vería después de haber hecho poner el dedo para nada.
	if C.esfingeHuella(m) != 1 {
		return nil, ErrNoQuiso
	}

	s, c := C.CString(servicio), C.CString(id)
	defer C.free(unsafe.Pointer(s))
	defer C.free(unsafe.Pointer(c))
	var datos unsafe.Pointer
	var n C.int
	switch st := C.esfingeLeer(s, c, &datos, &n); {
	case st == 0:
	case int(st) == -25300: // errSecItemNotFound
		return nil, ErrNoEsta
	default:
		return nil, fmt.Errorf("El llavero de macOS no ha devuelto la llave (%d)", int(st))
	}
	defer C.free(datos)
	return C.GoBytes(datos, n), nil
}

func (deMacOS) Borrar(id string) error {
	s, c := C.CString(servicio), C.CString(id)
	defer C.free(unsafe.Pointer(s))
	defer C.free(unsafe.Pointer(c))
	if st := C.esfingeBorrar(s, c); st != 0 {
		return fmt.Errorf("El llavero de macOS no ha borrado la llave (%d)", int(st))
	}
	return nil
}
