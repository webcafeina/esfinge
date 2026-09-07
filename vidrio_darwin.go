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
// **Y una tercera, que es la que costó tres versiones encontrar.** Wails crea el
// NSVisualEffectView, le pone la mezcla y el estado, y **nunca le pone el
// material** (se comprobó leyendo su WailsContext.m: solo hay `setBlendingMode`
// y `setState`). Se queda con el de por defecto,
// NSVisualEffectMaterialAppearanceBased, que Apple dejó **obsoleto en macOS
// 10.14** y que en macOS moderno se dibuja como una superficie plana. De ahí el
// «gris» de siempre: el vidrio estaba puesto y desenfocando nada.
//
// Las tres cosas son una línea de AppKit cada una, pero no hay forma de pedirlas
// desde la API de Wails, así que se hacen aquí. Si algún día Wails las hace, esto
// sobra entero.
//
// **Cómo se escribe esto después de romperlo una vez.** La 2.9.1 añadió aquí un
// diagnóstico que compilaba en verde y **cerraba la aplicación al arrancar**. Las
// reglas que salieron de aquello, y que este fichero cumple: nada de devolver
// cadenas a Go (un `UTF8String` autoliberado deja un puntero colgando), nada de
// `valueForKey:` (se puede escribir una propiedad que no se deja leer), nada de
// `alphaComponent` (lanza excepción sobre un color de patrón). Solo asignaciones
// a propiedades públicas, y preguntando antes si existen.

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

// prepararVistas recorre el árbol y arregla las dos vistas que hay que tocar: el
// WKWebView, para que no pinte por debajo de la página, y el
// NSVisualEffectView, para darle el material que Wails no le da.
//
// Va en profundidad porque Wails las cuelga dentro de la vista de contenido, no
// directamente de la ventana.
static void prepararVistas(NSView *vista) {
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
		if ([hija isKindOfClass:[NSVisualEffectView class]]) {
			NSVisualEffectView *efecto = (NSVisualEffectView *)hija;
			// El material de las barras laterales del sistema: es el que usan el
			// Finder, Correo y los Ajustes, y el que corresponde aquí porque la
			// única zona que deja ver el material es la barra lateral —la columna
			// de trabajo la pinta opaca el CSS.
			//
			// Existe desde macOS 10.11, muy por debajo de lo que exige Wails, pero
			// se pregunta igual: es gratis y este fichero no se puede probar aquí.
			if ([efecto respondsToSelector:@selector(setMaterial:)]) {
				[efecto setMaterial:NSVisualEffectMaterialSidebar];
			}
			// «Active» pase lo que pase: si no, el material se apaga cuando la
			// ventana pierde el foco y el efecto va y viene.
			[efecto setState:NSVisualEffectStateActive];
		}
		prepararVistas(hija);
	}
}

// PonerElVidrio deja pasar de verdad lo que hay detrás de la ventana.
static void PonerElVidrio(void) {
	// AppKit solo se toca desde el hilo principal, y esto se llama desde una
	// gorrutina de Go que puede estar en cualquier otro.
	dispatch_async(dispatch_get_main_queue(), ^{
		for (NSWindow *ventana in [NSApp windows]) {
			[ventana setOpaque:NO];
			[ventana setBackgroundColor:[NSColor clearColor]];
			prepararVistas([ventana contentView]);
		}
	});
}
*/
import "C"

func ponerElVidrio() { C.PonerElVidrio() }
