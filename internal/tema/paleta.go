package tema

// Tema es la paleta resuelta para un modo de apariencia.
//
// La 1.x vestía a Esfinge con la identidad de ClickHouse, que era una decisión
// para el terminal: allí no hay convenciones que seguir y una identidad fuerte
// ayuda a que la herramienta se reconozca. En una aplicación de escritorio la
// convención sí existe, y saltársela hace que la ventana se sienta ajena al
// sistema. Así que estos valores siguen las apariencias de macOS y Windows —gris
// neutro, azul de acción, tipografía del sistema— y la marca queda en el icono.
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

	// Acento. Relleno es el fondo de un botón de acción, con SobreAcento encima;
	// Acento es el mismo color en su versión legible como texto.
	Relleno     RGB
	RellenoVivo RGB
	SobreAcento RGB
	Acento      RGB

	// Superficies con nombre propio, porque en el sistema no son intercambiables:
	// la barra es translúcida, un campo se hunde y un botón se levanta.
	Barra       RGB
	Campo       RGB
	Boton       RGB
	BotonEncima RGB

	// Estados. Los tres se usan como texto, así que van pasados por AcentoLegible.
	Exito RGB
	Aviso RGB
	Error RGB
}

// Azules de acción del sistema, de donde parte el acento antes de ajustarlo.
//
// Ni Apple ni Microsoft cumplen AA con su azul de botón y texto blanco encima:
// el #007aff de macOS da 3,6:1. Aquí se oscurece hasta cumplir, con
// RellenoLegible, porque un botón que no se lee no es un botón bonito, es un
// botón roto.
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
		Relleno:      RellenoLegible(azulClaro, blanco, AANormal),
		RellenoVivo:  Oscurecer(RellenoLegible(azulClaro, blanco, AANormal), 0.85),
		SobreAcento:  blanco,
		Acento:       AcentoLegible(azulClaro, masOscura, AANormal),
		Barra:        MustParseHex("#f6f6f8"),
		Campo:        lienzo,
		Boton:        lienzo,
		BotonEncima:  MustParseHex("#f2f2f5"),
		Exito:        AcentoLegible(verdeSistema, masOscura, AANormal),
		Aviso:        AcentoLegible(ambarSistema, masOscura, AANormal),
		Error:        AcentoLegible(rojoSistema, masOscura, AANormal),
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
		Relleno:      RellenoLegible(azulOscuro, blanco, AANormal),
		RellenoVivo:  Oscurecer(RellenoLegible(azulOscuro, blanco, AANormal), 0.85),
		SobreAcento:  blanco,
		Barra:        MustParseHex("#242426"),
		Campo:        MustParseHex("#1a1a1c"),
		Boton:        MustParseHex("#3a3a3c"),
		BotonEncima:  MustParseHex("#48484a"),
		// Sobre fondo oscuro el azul del sistema ya se lee, así que se queda.
		Acento: AcentoLegible(azulOscuro, MustParseHex("#3a3a3c"), AANormal),
		Exito:  AcentoLegible(verdeSistema, MustParseHex("#3a3a3c"), AANormal),
		Aviso:  AcentoLegible(ambarSistema, MustParseHex("#3a3a3c"), AANormal),
		Error:  AcentoLegible(rojoSistema, MustParseHex("#3a3a3c"), AANormal),
	}
}()
