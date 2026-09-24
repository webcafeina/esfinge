package tema

import (
	"os"
	"strings"
	"testing"
)

// **Los colores de la extensión del navegador que no son tokens**, medidos.
//
// El panel usa los tokens de la ventana y ésos ya se miden en
// `contraste_test.go`. Pero hay tres sitios de la extensión donde el fondo no es
// nuestro (ADR 0031):
//
//   - **El icono de la barra**, sobre la barra de cada navegador, clara u oscura.
//   - **La insignia del icono**, que la dibuja el navegador con el color que se le
//     pase.
//   - **El aviso «Rellenado por Esfinge»**, dentro de la página de otro.
//
// Una regla de esta máquina es que el contraste se mide siempre, y que el sistema
// de origen cumpla no dice nada de la combinación. Así que se miden aquí, con las
// barras de Chrome y de Firefox como constantes, y **se comprueba que los colores
// medidos son los que usan los ficheros**: si alguien cambia un color en el
// TypeScript o en el SVG sin cambiarlo aquí, esto se pone rojo en vez de seguir
// midiendo el color viejo.
func TestLaExtensionSeVeDondeLaPintanOtros(t *testing.T) {
	var (
		oro          = MustParseHex("#f2c14e")
		piedra       = MustParseHex("#2b2b31")
		grisApagado  = MustParseHex("#a1a1a8")
		blanco       = MustParseHex("#ffffff")
		verdeExito   = MustParseHex("#1d6f31")
		ambarDeAviso = MustParseHex("#8f5300")
	)

	// Las barras de herramientas por defecto de los dos navegadores.
	claras := map[string]RGB{
		"Chrome claro":  MustParseHex("#ffffff"),
		"Firefox claro": MustParseHex("#f9f9fb"),
	}
	oscuras := map[string]RGB{
		"Chrome oscuro":  MustParseHex("#3c3c3c"),
		"Firefox oscuro": MustParseHex("#2b2a33"),
	}

	mide := func(frente, fondo RGB, minimo float64, que string) {
		t.Helper()
		if r := Contraste(frente, fondo); r < minimo {
			t.Errorf("MAL  %s sobre %s = %.2f:1, hace falta %.1f:1 — %s",
				frente.Hex(), fondo.Hex(), r, minimo, que)
		} else {
			t.Logf("ok   %s sobre %s = %5.2f:1 — %s", frente.Hex(), fondo.Hex(), r, que)
		}
	}

	// **La silueta: la piedra la sostiene en las barras claras y el relleno en las
	// oscuras.** No es texto, así que basta 3:1. Sin el contorno, el oro sobre una
	// barra clara no llega, y por eso existe el contorno.
	for nombre, barra := range claras {
		mide(piedra, barra, AAGrande, "contorno de la silueta en la barra, "+nombre)
	}
	for nombre, barra := range oscuras {
		mide(oro, barra, AAGrande, "silueta activa en la barra, "+nombre)
		mide(grisApagado, barra, AAGrande, "silueta apagada en la barra, "+nombre)
	}

	// Las insignias: texto blanco sobre su color.
	mide(blanco, piedra, AANormal, "número de cuentas en la insignia")
	mide(blanco, verdeExito, AANormal, "✓ de rellenado en la insignia")
	mide(blanco, ambarDeAviso, AANormal, "! de aviso en la insignia")
	// El candado va dibujado en el icono, en su placa naranja: la insignia del
	// navegador solo admite texto (2.20.2).
	mide(blanco, MustParseHex("#c2410c"), AANormal, "candado blanco en su placa naranja")

	// El aviso en la página: texto blanco y marca de oro sobre piedra.
	mide(blanco, piedra, AANormal, "«Rellenado por Esfinge»")
	mide(oro, piedra, AAGrande, "el tocado de la esfinge del aviso")
	// La cara, que en la 2.20.0 era un hueco y se veía negra sobre la piedra.
	mide(MustParseHex("#e8dcc4"), piedra, AAGrande, "la cara de la esfinge del aviso")

	// La tarjeta de guardar (ADR 0032): texto blanco y secundario sobre piedra, el
	// botón de oro con la piedra encima, y el campo del título en blanco con piedra.
	mide(blanco, piedra, AANormal, "el texto de la tarjeta de guardar")
	mide(MustParseHex("#d8d8de"), piedra, AANormal, "el texto secundario de la tarjeta")
	mide(piedra, oro, AANormal, "el botón «Guardar» de la tarjeta")
	mide(piedra, blanco, AANormal, "lo que se escribe en el título de la tarjeta")
	mide(oro, piedra, AAGrande, "el foco de los botones de la tarjeta")

	// Y que lo medido es lo que se usa.
	usan := map[string][]string{
		"../../navegador/src/insignia.ts":     {"#2b2b31", "#1d6f31", "#8f5300", "#ffffff"},
		"../../navegador/src/tarjeta.ts":      {"#2b2b31", "#ffffff", "#f2c14e", "#d8d8de"},
		"../../navegador/src/marcas.ts":       {"#f2c14e", "#2b2b31", "#ffffff"},
		"../../build/icono-barra.svg":         {"#f2c14e", "#2b2b31", "#e8dcc4"},
		"../../build/icono-barra-apagado.svg": {"#a1a1a8", "#2b2b31"},
		"../../build/icono-barra-cerrado.svg": {"#a1a1a8", "#2b2b31", "#ffffff", "#c2410c"},
	}
	for fichero, colores := range usan {
		datos, err := os.ReadFile(fichero)
		if err != nil {
			t.Errorf("no se puede leer %s: %v", fichero, err)
			continue
		}
		for _, c := range colores {
			if !strings.Contains(strings.ToLower(string(datos)), c) {
				t.Errorf("%s ya no usa %s, que es lo que se mide aquí: mide el color nuevo", fichero, c)
			}
		}
	}
}

// **Y el correo de invitación, que lo pinta el cliente de correo de otro** (ADR
// 0043, entrega B3). Es el único correo con formato, y va con los colores
// escritos a mano: en un correo no hay hoja de estilos, ni variables, ni clases
// que sobrevivan.
//
// Aquí se mide lo mismo que en la ventana y por la misma regla —el contraste se
// mide siempre, y que el sistema de origen cumpla no dice nada de la combinación
// resultante—, y se comprueba que los colores medidos son los que están en el
// fichero: si alguien cambia uno allí y no aquí, esto se pone rojo en vez de
// seguir midiendo el color viejo.
func TestElCorreoDeInvitacionSeLee(t *testing.T) {
	var (
		oro    = MustParseHex("#f2c14e")
		piedra = MustParseHex("#2b2b31")
		tinta  = MustParseHex("#1c1c1e")
		cuerpo = MustParseHex("#3c3c43")
		lienzo = MustParseHex("#ffffff")
		fondo  = MustParseHex("#f2f2f7")
		filete = MustParseHex("#d8d8de")
	)

	mide := func(frente, atras RGB, minimo float64, que string) {
		t.Helper()
		if r := Contraste(frente, atras); r < minimo {
			t.Errorf("MAL  %s sobre %s = %.2f:1, hace falta %.1f:1 — %s", frente.Hex(), atras.Hex(), r, minimo, que)
		} else {
			t.Logf("ok   %s sobre %s = %5.2f:1 — %s", frente.Hex(), atras.Hex(), r, que)
		}
	}

	// La banda de marca de arriba: el icono y, al lado, el nombre escrito.
	mide(lienzo, piedra, AANormal, "«Esfinge» en la banda de la invitación")
	mide(tinta, lienzo, AANormal, "el titular de la invitación")
	mide(cuerpo, lienzo, AANormal, "el texto de la invitación")
	// **El oro rellena y la piedra escribe**, como en toda la aplicación: sobre el
	// oro, el blanco daría 1,68:1 (ADR 0021).
	mide(piedra, oro, AANormal, "«Crear mi cuenta de Esfinge», el botón")
	mide(filete, fondo, 1.0, "la línea de la tarjeta sobre el fondo del correo")
	mide(lienzo, fondo, 1.0, "la tarjeta sobre el fondo del correo")

	datos, err := os.ReadFile("../../servidor/src/correo.ts")
	if err != nil {
		t.Fatalf("no se puede leer el correo: %v", err)
	}
	texto := strings.ToLower(string(datos))
	for _, c := range []string{"#f2c14e", "#2b2b31", "#1c1c1e", "#3c3c43", "#ffffff", "#f2f2f7", "#d8d8de"} {
		if !strings.Contains(texto, c) {
			t.Errorf("el correo ya no usa %s, que es lo que se mide aquí: mide el color nuevo", c)
		}
	}
	// **El icono de la cabecera puede faltar, y la cabecera tiene que seguir ahí.**
	// Lo eligió el cliente sabiendo lo que cuesta (2026-09-24), y lo que no se
	// negocia es que el correo dependa de una imagen: casi ningún cliente las enseña
	// de entrada, así que el nombre va escrito al lado.
	if strings.Contains(texto, "<img") && !strings.Contains(texto, `>esfinge</td>`) {
		t.Error("la cabecera del correo es solo la imagen: con las imágenes bloqueadas queda un hueco")
	}
	// **El botón va en una celda con `bgcolor`, no en un enlace con fondo.** Con un
	// `<a style="background:…">` Gmail no lo pinta, y eso solo se vio mandándolo a un
	// buzón de verdad (2026-09-24).
	if !strings.Contains(texto, `bgcolor="#f2c14e"`) {
		t.Error("el botón de la invitación ya no lleva bgcolor: en Gmail se queda sin fondo")
	}
	if strings.Contains(texto, "background:#") {
		t.Error("el correo usa la forma abreviada `background:`, que Gmail tira: va `background-color:`")
	}
}
