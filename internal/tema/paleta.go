package tema

// Tema es la paleta resuelta para un modo de apariencia.
//
// La 1.x vestía a Esfinge con la identidad de ClickHouse, que era una decisión
// para el terminal: allí no hay convenciones que seguir y una identidad fuerte
// ayuda a que la herramienta se reconozca. En una aplicación de escritorio la
// convención sí existe, y saltársela hace que la ventana se sienta ajena al
// sistema. Así que las superficies, la tipografía y las formas siguen a macOS y
// a Windows: gris neutro, controles del sistema, nada inventado.
//
// **Lo que sí es nuestro es el acento** (ADR 0021). Desde la 2.11.0 el color de
// acción no es el azul del sistema sino el oro del tocado de la esfinge, que es
// lo único de la ventana que dice de quién es esto.
//
// Y con el oro hay una regla que no es de gusto sino de contraste medido:
//
//	blanco sobre el oro    1,68:1   imposible
//	piedra sobre el oro    8,38:1   cómodo
//
// De ahí: **el oro rellena, la piedra escribe**. Que resulta ser la gramática
// del propio icono —tocado dorado sobre placa oscura—, no una ocurrencia.
// Oscurecer el oro hasta que el blanco se lea encima lo convierte en un marrón
// (#9a6c1b) y se pierde la marca, así que lo que cambia es la tinta, no el oro.
//
// Lo que no ha cambiado es el mecanismo: el mismo tipo, el mismo cálculo de
// contraste y el mismo test que falla si una pareja no cumple.
type Tema struct {
	Nombre string
	Oscuro bool

	// Superficies, de la más al fondo a la más elevada.
	Lienzo  RGB
	Suave   RGB
	Tarjeta RGB
	Elevada RGB

	// Separaciones.
	Filete       RGB
	FileteFuerte RGB

	// Texto, en tres pesos de presencia.
	Tinta   RGB
	Cuerpo  RGB
	Apagado RGB

	// Acento. Relleno es el fondo de un botón de acción, con SobreAcento encima.
	//
	// **Acento ya no es «el mismo color en versión legible como texto»**, y ese
	// cambio es la mitad de la ADR 0021: con el oro de marca eso daría un bronce,
	// y en la ventana convivirían dos oros distintos. Acento es ahora la tinta
	// fuerte del tema —la piedra en claro, la tinta clara en oscuro—, porque el
	// oro rellena y nunca escribe.
	Relleno     RGB
	RellenoVivo RGB
	SobreAcento RGB
	Acento      RGB

	// Superficies con nombre propio, porque en el sistema no son intercambiables:
	// la barra es translúcida, un campo se hunde y un botón se levanta.
	//
	// BarraEncima es la fila de la barra lateral con el puntero encima, y **es
	// más oscura que la barra, no más clara**. Un botón corriente se levanta al
	// pasar por encima; una fila de barra lateral se hunde, que es lo que hacen
	// las aplicaciones del sistema y lo que se pidió al ver la 2.11.0. No vale
	// reutilizar BotonEncima: en tema oscuro ése aclara.
	Barra       RGB
	BarraEncima RGB
	Campo       RGB
	Boton       RGB
	BotonEncima RGB

	// Estados. Los tres se usan como texto, así que van pasados por AcentoLegible.
	Exito RGB
	Aviso RGB
	Error RGB

	// Los cuadros de las entradas de la bóveda: el fondo de cada uno y la tinta
	// que va encima.
	//
	// **Son el único color propio que entra en una pantalla de trabajo**, y la ADR
	// 0021 dejó dicho que ahí no entra ninguno. Entran porque el problema cambió:
	// con sesenta y cinco entradas dentro, una lista de un solo color no se
	// recorre con el ojo y hay que leerla entera. El tinte lo elige el dominio, así
	// que es estable —el mismo sitio, el mismo color siempre— y sirve de ancla.
	//
	// La condición para que eso no se convierta en una feria: **apagados**, muy
	// por debajo del oro en saturación, y con **todas las parejas medidas**. Son
	// ocho para que dos sitios cualesquiera se distingan sin que el ojo tenga que
	// nombrar el color.
	TintesDeIcono []RGB
	TintaDeIcono  RGB

	// OroLateral es el oro de los iconos de la barra lateral, y **sólo lo usa
	// macOS** (ADR 0068): allí la fila elegida es gris —la gramática de macOS 27— y
	// lo que lleva la marca son los iconos, como el acento del sistema en los suyos.
	// En los demás sistemas la marca está en la fila, rellena de oro, y este color no
	// lo pinta nadie.
	//
	// **No llega a 3:1 en claro, y es a propósito.** Cada icono lleva su nombre
	// escrito al lado, así que no es él quien carga el significado; y el oro
	// oscurecido hasta 3:1 **se vuelve verde oliva** y deja de parecer la marca. El
	// cliente lo eligió así, sabiéndolo, primero en Apolo y aquí el 2026-11-20. La
	// pareja se mide igual, con su umbral escrito, en `contraste_test.go`.
	OroLateral RGB
}

// Los colores de la marca, sacados tal cual de build/icono.svg. No son
// aproximaciones: son los mismos valores que pinta el tocado y la placa, para
// que el icono del Dock y el botón de la ventana sean el mismo oro.
var (
	oroMarca = MustParseHex("#f2c14e") // el claro del tocado, arriba del degradado
	oroHondo = MustParseHex("#d99a28") // el hondo, abajo
	piedra   = MustParseHex("#2b2b31") // la placa sobre la que va el tocado
)

// Azules de acción del sistema. **Ya no son el acento** —lo es el oro, ADR
// 0021— pero se quedan aquí y no se borran, por dos razones.
//
// La primera es que son el porqué de RellenoLegible: ni Apple ni Microsoft
// cumplen AA con su azul de botón y texto blanco encima —el #007aff de macOS da
// 3,6:1—, y esa función nació para arreglarlo. Borrar el azul dejaría la función
// sin historia y parecería un adorno.
//
// La segunda es que siguen siendo la referencia contra la que se juzga el
// cambio: el azul ajustado daba 4,65:1 con blanco encima, y el oro da 8,38:1 con
// la piedra. Se mejoró, no se empeoró.
var (
	azulClaro  = MustParseHex("#007aff") // el de macOS
	azulOscuro = MustParseHex("#0a84ff") // su variante para modo oscuro
	blanco     = MustParseHex("#ffffff")

	verdeSistema = MustParseHex("#34c759")
	ambarSistema = MustParseHex("#ff9500")
	rojoSistema  = MustParseHex("#ff3b30")
)

// Y los de Fluent, que no son los mismos (ADR 0064). Microsoft los publica ya
// pensados para leerse como texto —de ahí que el verde sea un verde oscuro y no
// el verde manzana de Apple—, así que al pasarlos por AcentoLegible casi no se
// mueven. Los de oscuro son sus propias variantes, no éstos aclarados.
var (
	verdeWindows = MustParseHex("#0f7b0f")
	ambarWindows = MustParseHex("#9d5d00")
	rojoWindows  = MustParseHex("#c42b1c")

	verdeWindowsOscuro = MustParseHex("#6ccb5f")
	ambarWindowsOscuro = MustParseHex("#fce100")
	rojoWindowsOscuro  = MustParseHex("#ff99a4")
)

// Los ocho tintes de los cuadros de la bóveda, recorriendo el círculo de color a
// saltos parejos para que dos cualesquiera se distingan.
//
// **Van muy desaturados a propósito.** El oro de la marca es lo único que puede
// llamar la atención en la ventana, y ocho colores vivos al lado lo apagarían.
// Éstos son lavados: dicen «esta fila no es aquella» y no dicen nada más.
//
// En claro son casi blancos, para que escriba la piedra; en oscuro son casi
// negros, para que escriba la tinta clara. Los dos juegos están medidos en
// contraste_test.go, uno a uno.
var (
	tintesClaros = []RGB{
		MustParseHex("#d6e2f0"), // azulado
		MustParseHex("#d3e8e2"), // verde agua
		MustParseHex("#dde8d2"), // oliva
		MustParseHex("#f0e6cf"), // arena
		MustParseHex("#f2ddd4"), // terracota
		MustParseHex("#f0dae2"), // rosa
		MustParseHex("#e4dcf0"), // malva
		MustParseHex("#d3e4ec"), // turquesa
	}

	tintesOscuros = []RGB{
		MustParseHex("#2d3b4d"), // azulado
		MustParseHex("#2a423b"), // verde agua
		MustParseHex("#38442e"), // oliva
		MustParseHex("#493d2b"), // arena
		MustParseHex("#4a342b"), // terracota
		MustParseHex("#48303c"), // rosa
		MustParseHex("#3a3350"), // malva
		MustParseHex("#2b4149"), // turquesa
	}
)

// TemaClaro sigue la apariencia clara del sistema.
var TemaClaro = func() Tema {
	lienzo := MustParseHex("#ffffff")
	// Los colores de texto se derivan contra la superficie de menos contraste, no
	// contra el lienzo: un acento calculado contra blanco cumple sobre el lienzo y
	// se queda corto en cuanto cae dentro de una tarjeta, que es donde más se usa.
	masOscura := MustParseHex("#e8e8ed")

	return Tema{
		Nombre:       "claro",
		Lienzo:       lienzo,
		Suave:        MustParseHex("#f7f7f9"),
		Tarjeta:      MustParseHex("#f2f2f7"),
		Elevada:      masOscura,
		Filete:       MustParseHex("#d8d8de"),
		FileteFuerte: MustParseHex("#8e8e93"),
		Tinta:        MustParseHex("#1c1c1e"),
		Cuerpo:       MustParseHex("#3c3c43"),
		Apagado:      MustParseHex("#6c6c72"),
		// El oro pasa por RellenoLegible igual que pasaba el azul, aunque con la
		// piedra encima no tenga nada que corregir: la red se deja puesta, no se
		// quita porque hoy no haga falta.
		Relleno:     RellenoLegible(oroMarca, piedra, AANormal),
		RellenoVivo: Oscurecer(RellenoLegible(oroMarca, piedra, AANormal), 0.85),
		SobreAcento: piedra,
		// La piedra del icono, no un oro oscurecido: el oro rellena y no escribe.
		Acento: piedra,
		Barra:  MustParseHex("#f6f6f8"),
		// Un 6 % más oscura que la barra: se nota sin llamar la atención.
		BarraEncima: Oscurecer(MustParseHex("#f6f6f8"), 0.94),
		Campo:       lienzo,
		Boton:       lienzo,
		BotonEncima: MustParseHex("#f2f2f5"),
		Exito:       AcentoLegible(verdeSistema, masOscura, AANormal),
		Aviso:       AcentoLegible(ambarSistema, masOscura, AANormal),
		Error:       AcentoLegible(rojoSistema, masOscura, AANormal),
		// La misma piedra que escribe sobre el oro: en la ventana solo hay una
		// tinta oscura, y los cuadros no son una excepción.
		TintesDeIcono: tintesClaros,
		TintaDeIcono:  piedra,
		OroLateral:    AcentoLegible(oroMarca, masOscura, AAGrande),
	}
}()

// TemaOscuro sigue la apariencia oscura del sistema.
var TemaOscuro = func() Tema {
	lienzo := MustParseHex("#1e1e1e")

	return Tema{
		Nombre:       "oscuro",
		Oscuro:       true,
		Lienzo:       lienzo,
		Suave:        MustParseHex("#252527"),
		Tarjeta:      MustParseHex("#2c2c2e"),
		Elevada:      MustParseHex("#3a3a3c"),
		Filete:       MustParseHex("#3f3f42"),
		FileteFuerte: MustParseHex("#5a5a5f"),
		Tinta:        MustParseHex("#ffffff"),
		Cuerpo:       MustParseHex("#e3e3e6"),
		Apagado:      MustParseHex("#a1a1a8"),
		// El mismo oro que en claro, y a propósito: el icono del Dock no cambia
		// con la apariencia del sistema, así que el acento tampoco.
		Relleno:     RellenoLegible(oroMarca, piedra, AANormal),
		RellenoVivo: Oscurecer(RellenoLegible(oroMarca, piedra, AANormal), 0.85),
		SobreAcento: piedra,
		Barra:       MustParseHex("#242426"),
		// En oscuro hay menos recorrido hacia abajo —la barra ya está cerca del
		// negro— así que el paso es mayor para que se llegue a ver.
		BarraEncima: Oscurecer(MustParseHex("#242426"), 0.7),
		Campo:       MustParseHex("#1a1a1c"),
		Boton:       MustParseHex("#3a3a3c"),
		BotonEncima: MustParseHex("#48484a"),
		// Aquí el oro sí se leería como texto —da 9,93:1 sobre el lienzo oscuro—,
		// y aun así no se usa: la regla es una sola en los dos temas o deja de ser
		// una regla. Lo que escribe es la tinta, que en oscuro es la clara.
		Acento: blanco,
		Exito:  AcentoLegible(verdeSistema, MustParseHex("#3a3a3c"), AANormal),
		Aviso:  AcentoLegible(ambarSistema, MustParseHex("#3a3a3c"), AANormal),
		Error:  AcentoLegible(rojoSistema, MustParseHex("#3a3a3c"), AANormal),
		// No el blanco puro: una inicial en blanco sobre ocho cuadros oscuros es
		// ocho destellos en una lista que se recorre de arriba abajo.
		TintesDeIcono: tintesOscuros,
		TintaDeIcono:  MustParseHex("#ebebf0"),
		OroLateral:    AcentoLegible(oroMarca, MustParseHex("#3a3a3c"), AAGrande),
	}
}()

// WindowsClaro es el modo claro en Windows: los neutros de WinUI 3 (ADR 0064).
//
// # De dónde salen
//
// De la paleta de Fluent 2 —`SolidBackgroundFillColorBase` y familia— aplanada
// sobre el gris de Mica, que es lo que se ve de verdad detrás de una ventana de
// Windows 11. **No son éstos retocados a ojo**: son los mismos que Apolo usa
// desde su v0.13.0, ya derivados y vistos en una máquina.
//
// Lo que no cambia del tema base es el oro: el acento es nuestro en los cuatro
// sistemas (ADR 0021), y la piedra es lo que escribe encima.
var WindowsClaro = func() Tema {
	lienzo := MustParseHex("#f9f9f9")
	tarjeta := MustParseHex("#ffffff")
	// La superficie de menos contraste, contra la que se derivan los textos. En
	// Fluent no es el lienzo: es la fila con el puntero encima, que es donde un
	// texto apagado se queda corto antes que en ningún otro sitio.
	peor := MustParseHex("#ebebeb")
	relleno := RellenoLegible(oroMarca, piedra, AANormal)

	return Tema{
		Nombre:  "windows-claro",
		Lienzo:  lienzo,
		Suave:   MustParseHex("#f3f3f3"),
		Tarjeta: tarjeta,
		Elevada: peor,
		// **Derivado, no copiado, y aquí se vio por qué.** El divisor de Fluent
		// —negro al 8 % sobre la superficie, o sea este #e5e5e5— da **1,1996:1**
		// sobre su lienzo, y el suelo de esta casa para una línea de separación es
		// 1,2:1. Son cuatro diezmilésimas, y da igual: lo que no se baja es el
		// umbral. Así que se parte del color de Fluent y se oscurece hasta el
		// suelo, que es lo mismo que se hace con los textos y con los estados.
		Filete:       AcentoLegible(MustParseHex("#e5e5e5"), lienzo, 1.2),
		FileteFuerte: AcentoLegible(MustParseHex("#8a8a8a"), peor, AAGrande),
		Tinta:        MustParseHex("#1b1b1b"),
		Cuerpo:       MustParseHex("#3d3d3d"),
		Apagado:      AcentoLegible(MustParseHex("#616161"), peor, AANormal),
		Relleno:      relleno,
		// Más suave que en el tema base (0,85): el estado pulsado de Fluent apenas
		// se mueve, y un salto grande ahí se lee como otro color.
		RellenoVivo: Oscurecer(relleno, 0.92),
		SobreAcento: piedra,
		Acento:      piedra,
		Barra:       MustParseHex("#f3f3f3"),
		BarraEncima: MustParseHex("#eaeaea"),
		Campo:       tarjeta,
		Boton:       MustParseHex("#fbfbfb"),
		BotonEncima: MustParseHex("#f6f6f6"),
		Exito:       AcentoLegible(verdeWindows, peor, AANormal),
		Aviso:       AcentoLegible(ambarWindows, peor, AANormal),
		Error:       AcentoLegible(rojoWindows, peor, AANormal),
		// Los ocho cuadros de la bóveda no cambian de sistema: el tinte lo elige el
		// dominio y tiene que dar lo mismo en todas partes, que es lo que lo hace
		// servir de ancla. Lo que cambia es la superficie contra la que se miden.
		TintesDeIcono: tintesClaros,
		TintaDeIcono:  piedra,
		OroLateral:    AcentoLegible(oroMarca, peor, AAGrande),
	}
}()

// WindowsOscuro es el modo oscuro en Windows: los neutros de WinUI 3 (ADR 0064).
//
// **Y aquí la fila de la barra lateral aclara al pasar por encima**, al
// contrario que en el tema base. No es un descuido: la regla de la casa —«la
// barra lateral se hunde, no se levanta»— se escribió mirando macOS, y su razón
// era «es lo que hacen las aplicaciones del sistema». En Fluent lo que hace el
// sistema es lo contrario, porque su estado sutil es blanco al 6 %. Se sigue la
// razón, no la letra; en claro las dos coinciden y la fila se oscurece igual.
var WindowsOscuro = func() Tema {
	// La superficie más clara del tema, que es el botón con el puntero encima.
	peor := MustParseHex("#3a3a3a")
	relleno := RellenoLegible(oroMarca, piedra, AANormal)

	return Tema{
		Nombre:        "windows-oscuro",
		Oscuro:        true,
		Lienzo:        MustParseHex("#272727"),
		Suave:         MustParseHex("#1c1c1c"),
		Tarjeta:       MustParseHex("#2b2b2b"),
		Elevada:       MustParseHex("#383838"),
		Filete:        MustParseHex("#3a3a3a"),
		FileteFuerte:  AcentoLegible(MustParseHex("#9a9a9a"), peor, AAGrande),
		Tinta:         MustParseHex("#ffffff"),
		Cuerpo:        MustParseHex("#e4e4e4"),
		Apagado:       AcentoLegible(MustParseHex("#9d9d9d"), peor, AANormal),
		Relleno:       relleno,
		RellenoVivo:   Oscurecer(relleno, 0.92),
		SobreAcento:   piedra,
		Acento:        blanco,
		Barra:         MustParseHex("#202020"),
		BarraEncima:   MustParseHex("#2d2d2d"),
		Campo:         MustParseHex("#1e1e1e"),
		Boton:         MustParseHex("#2d2d2d"),
		BotonEncima:   MustParseHex("#323232"),
		Exito:         AcentoLegible(verdeWindowsOscuro, peor, AANormal),
		Aviso:         AcentoLegible(ambarWindowsOscuro, peor, AANormal),
		Error:         AcentoLegible(rojoWindowsOscuro, peor, AANormal),
		TintesDeIcono: tintesOscuros,
		TintaDeIcono:  MustParseHex("#ebebf0"),
		OroLateral:    AcentoLegible(oroMarca, peor, AAGrande),
	}
}()

// MacosClaro es el modo claro en macOS: los neutros de macOS 27 Golden Gate.
//
// # De dónde salen
//
// Del kit de diseño de Apple para macOS 27 en Figma —`Window Background`,
// `Fills - Vibrant` y `Labels`—, y **no de ahí a pelo**: son los mismos que Apolo
// publica desde su v0.14.0, o sea ya derivados, ya medidos y **ya vistos en un Mac
// de verdad**, con las correcciones que salieron de mirarlos allí. Volver al kit a
// leerlos otra vez habría sido gastar el asiento de Figma en releer lo que ya
// estaba leído y corregido.
//
// El contenido va en blanco y los grupos, los botones y la barra lateral en los
// rellenos grises del sistema — al contrario que en Fluent, donde el lienzo es
// gris y la tarjeta blanca.
//
// Lo que no cambia es el oro: el acento es nuestro en los cuatro sistemas
// (ADR 0021) y la piedra es lo que escribe encima.
var MacosClaro = func() Tema {
	lienzo := MustParseHex("#ffffff")
	// La superficie de menos contraste, contra la que se derivan los textos. En
	// macOS es **la fila elegida de la barra lateral**, que es el gris más oscuro
	// de la ventana; un texto apagado se queda corto ahí antes que en el lienzo.
	peor := MustParseHex("#d9d9d9")
	relleno := RellenoLegible(oroMarca, piedra, AANormal)

	return Tema{
		Nombre:  "macos-claro",
		Lienzo:  lienzo,
		Suave:   MustParseHex("#f7f7f7"),
		Tarjeta: MustParseHex("#f2f2f2"),
		Elevada: MustParseHex("#e6e6e6"),
		// Derivado y no copiado, por lo mismo que en Fluent: el separador del kit
		// puede quedarse por debajo del suelo de 1,2:1 de esta casa, y lo que no se
		// baja nunca es el umbral.
		Filete:       AcentoLegible(MustParseHex("#e5e5e5"), lienzo, 1.2),
		FileteFuerte: AcentoLegible(MustParseHex("#8c8c8c"), peor, AAGrande),
		Tinta:        MustParseHex("#1a1a1a"),
		Cuerpo:       MustParseHex("#3b3b3b"),
		Apagado:      AcentoLegible(MustParseHex("#727272"), peor, AANormal),
		Relleno:      relleno,
		RellenoVivo:  Oscurecer(relleno, 0.92),
		SobreAcento:  piedra,
		Acento:       piedra,
		Barra:        MustParseHex("#e6e6e6"),
		BarraEncima:  peor,
		Campo:        lienzo,
		Boton:        MustParseHex("#ebebeb"),
		BotonEncima:  MustParseHex("#e1e1e1"),
		Exito:        AcentoLegible(verdeSistema, peor, AANormal),
		Aviso:        AcentoLegible(ambarSistema, peor, AANormal),
		Error:        AcentoLegible(rojoSistema, peor, AANormal),
		// Los ocho cuadros de la bóveda no cambian de sistema: el tinte lo elige el
		// dominio y tiene que dar lo mismo en todas partes. Lo que cambia es la
		// superficie contra la que se miden.
		TintesDeIcono: tintesClaros,
		TintaDeIcono:  piedra,
		OroLateral:    MustParseHex("#d99a00"),
	}
}()

// MacosOscuro es el modo oscuro en macOS, con los neutros de macOS 27.
var MacosOscuro = func() Tema {
	// La superficie más clara del tema, que es el botón con el puntero encima.
	peor := MustParseHex("#404040")
	relleno := RellenoLegible(oroMarca, piedra, AANormal)

	return Tema{
		Nombre:        "macos-oscuro",
		Oscuro:        true,
		Lienzo:        MustParseHex("#1e1e1e"),
		Suave:         MustParseHex("#191919"),
		Tarjeta:       MustParseHex("#282828"),
		Elevada:       MustParseHex("#323232"),
		Filete:        AcentoLegible(MustParseHex("#363636"), MustParseHex("#1e1e1e"), 1.2),
		FileteFuerte:  AcentoLegible(MustParseHex("#8a8a8a"), peor, AAGrande),
		Tinta:         MustParseHex("#f5f5f5"),
		Cuerpo:        MustParseHex("#dcdcdc"),
		Apagado:       AcentoLegible(MustParseHex("#9a9a9a"), peor, AANormal),
		Relleno:       relleno,
		RellenoVivo:   Oscurecer(relleno, 0.92),
		SobreAcento:   piedra,
		Acento:        blanco,
		Barra:         MustParseHex("#161616"),
		BarraEncima:   MustParseHex("#2a2a2a"),
		Campo:         MustParseHex("#262626"),
		Boton:         MustParseHex("#383838"),
		BotonEncima:   peor,
		Exito:         AcentoLegible(verdeSistema, peor, AANormal),
		Aviso:         AcentoLegible(ambarSistema, peor, AANormal),
		Error:         AcentoLegible(rojoSistema, peor, AANormal),
		TintesDeIcono: tintesOscuros,
		TintaDeIcono:  MustParseHex("#ebebf0"),
		OroLateral:    AcentoLegible(oroMarca, peor, AAGrande),
	}
}()

// Los colores con significado de Yaru, que son los de Ubuntu y no los de Apple.
//
// Van aparte igual que los de Fluent: un verde de éxito es parte de la identidad
// del escritorio, y en Ubuntu es el verde de Canonical. Los de abajo son los
// canónicos de Yaru en claro y los de la paleta de Ubuntu en oscuro, que es lo
// que Apolo publica desde su v0.13.0; `AcentoLegible` los sube o los baja hasta
// que cumplen contra la peor superficie de su paleta.
var (
	verdeYaru = MustParseHex("#219e34")
	ambarYaru = MustParseHex("#c88800")
	rojoYaru  = MustParseHex("#d3273b")

	verdeYaruOscuro = MustParseHex("#0e8420")
	ambarYaruOscuro = MustParseHex("#cd9309")
	rojoYaruOscuro  = MustParseHex("#c7162b")
)

// LinuxClaro es el modo claro en GNOME: los neutros de Yaru (ADR 0070).
//
// # De dónde salen
//
// De libadwaita 1.10 con el tema Yaru, que es lo que trae Ubuntu 26.10 — Ubuntu
// no cambia ninguna medida de libadwaita, sólo los colores—. **No son los de la
// hoja de libadwaita a pelo**: son los mismos que Apolo usa desde su v0.13.0, o
// sea ya derivados y **ya vistos en una Ubuntu de verdad**.
//
// Al contrario que en Fluent, aquí el lienzo es casi blanco y las tarjetas son
// **blancas**: una `boxed list` de Ubuntu se levanta sobre el fondo de la
// ventana, no se hunde en él.
//
// Lo que no cambia del tema base es el oro: el acento es nuestro en los cuatro
// sistemas (ADR 0021), y la piedra es lo que escribe encima.
var LinuxClaro = func() Tema {
	lienzo := MustParseHex("#fafafa")
	// La superficie de menos contraste, contra la que se derivan los textos. En
	// GNOME es **la fila elegida de la barra lateral**, que es el gris más oscuro
	// de la ventana: un texto apagado se queda corto ahí antes que en el lienzo.
	peor := MustParseHex("#d8d8d8")
	relleno := RellenoLegible(oroMarca, piedra, AANormal)

	return Tema{
		Nombre:  "linux-claro",
		Lienzo:  lienzo,
		Suave:   MustParseHex("#f0f0f0"),
		Tarjeta: MustParseHex("#ffffff"),
		Elevada: MustParseHex("#e7e7e7"),
		// Derivado y no copiado, por lo mismo que en Fluent y en macOS: el divisor
		// de libadwaita puede quedarse por debajo del suelo de 1,2:1 de esta casa,
		// y lo que no se baja nunca es el umbral.
		Filete:        AcentoLegible(MustParseHex("#e0e0e0"), lienzo, 1.2),
		FileteFuerte:  AcentoLegible(MustParseHex("#8a8a8a"), peor, AAGrande),
		Tinta:         MustParseHex("#303030"),
		Cuerpo:        MustParseHex("#3d3d3d"),
		Apagado:       AcentoLegible(MustParseHex("#737373"), peor, AANormal),
		Relleno:       relleno,
		RellenoVivo:   Oscurecer(relleno, 0.92),
		SobreAcento:   piedra,
		Acento:        piedra,
		Barra:         MustParseHex("#ebebeb"),
		BarraEncima:   peor,
		Campo:         MustParseHex("#ebebeb"),
		Boton:         MustParseHex("#e7e7e7"),
		BotonEncima:   MustParseHex("#dedede"),
		Exito:         AcentoLegible(verdeYaru, peor, AANormal),
		Aviso:         AcentoLegible(ambarYaru, peor, AANormal),
		Error:         AcentoLegible(rojoYaru, peor, AANormal),
		TintesDeIcono: tintesClaros,
		TintaDeIcono:  piedra,
		OroLateral:    AcentoLegible(oroMarca, peor, AAGrande),
	}
}()

// LinuxOscuro es el modo oscuro en GNOME, con los neutros de Yaru.
var LinuxOscuro = func() Tema {
	// La superficie más clara del tema, que es el botón con el puntero encima.
	peor := MustParseHex("#4d4d4d")
	relleno := RellenoLegible(oroMarca, piedra, AANormal)

	return Tema{
		Nombre:        "linux-oscuro",
		Oscuro:        true,
		Lienzo:        MustParseHex("#2c2c2c"),
		Suave:         MustParseHex("#323232"),
		Tarjeta:       MustParseHex("#3a3a3a"),
		Elevada:       MustParseHex("#414141"),
		Filete:        AcentoLegible(MustParseHex("#474747"), MustParseHex("#2c2c2c"), 1.2),
		FileteFuerte:  AcentoLegible(MustParseHex("#8f8f8f"), peor, AAGrande),
		Tinta:         MustParseHex("#f7f7f7"),
		Cuerpo:        MustParseHex("#e4e4e4"),
		Apagado:       AcentoLegible(MustParseHex("#a6a6a6"), peor, AANormal),
		Relleno:       relleno,
		RellenoVivo:   Oscurecer(relleno, 0.92),
		SobreAcento:   piedra,
		Acento:        blanco,
		Barra:         MustParseHex("#393939"),
		BarraEncima:   MustParseHex("#4a4a4a"),
		Campo:         MustParseHex("#3e3e3e"),
		Boton:         MustParseHex("#414141"),
		BotonEncima:   peor,
		Exito:         AcentoLegible(verdeYaruOscuro, peor, AANormal),
		Aviso:         AcentoLegible(ambarYaruOscuro, peor, AANormal),
		Error:         AcentoLegible(rojoYaruOscuro, peor, AANormal),
		TintesDeIcono: tintesOscuros,
		TintaDeIcono:  MustParseHex("#ebebf0"),
		OroLateral:    AcentoLegible(oroMarca, peor, AAGrande),
	}
}()

// Temas son todas las paletas que se publican, para medirlas todas.
//
// Es la lista que recorre el test de contraste: una paleta que no esté aquí no
// la mide nadie, y una paleta sin medir es justo lo que este proyecto no puede
// permitirse —el oro no vale como línea en ningún sitio, y eso hay que
// comprobarlo en cada juego de neutros, no en uno—.
func Temas() []Tema {
	return []Tema{
		TemaClaro, TemaOscuro,
		WindowsClaro, WindowsOscuro,
		MacosClaro, MacosOscuro,
		LinuxClaro, LinuxOscuro,
	}
}
