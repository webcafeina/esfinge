//go:build darwin && cgo && !dev

package main

// Termina de poner el vidrio de macOS, que Wails deja a medias.
//
// **El problema.** Wails crea el NSVisualEffectView detrás de la ventana y le da
// mezcla «BehindWindow», que es lo correcto. Pero **nunca pone la ventana como no
// opaca**: le cambia el color de fondo a transparente y ya. Y una NSWindow con
// `opaque = YES` compone como opaca por mucho que su color tenga alfa cero, así
// que el material no tiene nada detrás que mezclar y se dibuja como un gris
// plano. Es justo lo que se veía: vidrio puesto, efecto ninguno.
//
// **Y hay una segunda capa.** Desde macOS 12, WKWebView pinta su
// `underPageBackgroundColor` por debajo de la página aunque `drawsBackground`
// esté a NO. Si no se aclara, tapa el material igual.
//
// Las dos cosas son una línea de AppKit cada una, pero no hay forma de pedirlas
// desde la API de Wails, así que se hacen aquí. Si algún día Wails las hace, esto
// sobra entero.

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

// aclararWebviews busca el WKWebView y le quita el fondo que pinta por debajo de
// la página. Va en profundidad porque Wails lo cuelga dentro de la vista de
// contenido, no directamente de la ventana.
static void aclararWebviews(NSView *vista) {
	if (vista == nil) {
		return;
	}
	for (NSView *hija in [vista subviews]) {
		if ([hija isKindOfClass:[WKWebView class]]) {
			WKWebView *web = (WKWebView *)hija;
			// Solo existe desde macOS 12: en versiones anteriores no hace falta y
			// preguntar evita el fallo.
			if ([web respondsToSelector:@selector(setUnderPageBackgroundColor:)]) {
				[web setUnderPageBackgroundColor:[NSColor clearColor]];
			}
		}
		aclararWebviews(hija);
	}
}

// EstadoDelVidrio mira **lo que de verdad hay puesto**, no lo que se pidió.
//
// Existe porque el vidrio no se veía y las tres primeras explicaciones fueron
// razonamientos, no medidas. Devuelve una cadena corta con los cuatro hechos que
// deciden si el material puede componer: si la ventana es opaca, si su color de
// fondo tiene alfa, si detrás hay una vista de efecto, y si el webview pinta su
// propio fondo. Con eso se sabe dónde se corta la cadena sin adivinar.
static const char* EstadoDelVidrio(void) {
	__block NSString *fuera = @"sin ventana";

	void (^mirar)(void) = ^{
		for (NSWindow *v in [NSApp windows]) {
			if ([v contentView] == nil) {
				continue;
			}
			BOOL hayEfecto = NO;
			BOOL webviewOpaco = NO;
			for (NSView *hija in [[v contentView] subviews]) {
				if ([hija isKindOfClass:[NSVisualEffectView class]]) {
					hayEfecto = YES;
				}
			}
			for (NSView *hija in [[v contentView] subviews]) {
				if ([hija isKindOfClass:[WKWebView class]]) {
					id dibuja = [hija valueForKey:@"drawsBackground"];
					webviewOpaco = [dibuja boolValue];
				}
			}
			fuera = [NSString stringWithFormat:@"opaca=%@ alfa=%.2f efecto=%@ webview=%@",
			         [v isOpaque] ? @"si" : @"no",
			         [[v backgroundColor] alphaComponent],
			         hayEfecto ? @"si" : @"no",
			         webviewOpaco ? @"con fondo" : @"transparente"];
			break;
		}
	};

	if ([NSThread isMainThread]) {
		mirar();
	} else {
		dispatch_sync(dispatch_get_main_queue(), mirar);
	}
	return [fuera UTF8String];
}

// PonerElVidrio deja pasar de verdad lo que hay detrás de la ventana.
static void PonerElVidrio(void) {
	// AppKit solo se toca desde el hilo principal, y esto se llama desde una
	// gorrutina de Go que puede estar en cualquier otro.
	dispatch_async(dispatch_get_main_queue(), ^{
		for (NSWindow *ventana in [NSApp windows]) {
			[ventana setOpaque:NO];
			[ventana setBackgroundColor:[NSColor clearColor]];
			aclararWebviews([ventana contentView]);
		}
	});
}
*/
import "C"

func ponerElVidrio() { C.PonerElVidrio() }

// estadoDelVidrio devuelve lo que AppKit dice de la ventana ahora mismo.
func estadoDelVidrio() string { return C.GoString(C.EstadoDelVidrio()) }
