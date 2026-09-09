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
	}
}()
