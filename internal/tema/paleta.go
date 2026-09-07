package tema

// Tema es la paleta resuelta para un fondo de terminal concreto.
//
// Los valores del tema oscuro salen tal cual de
// ~/sistemas-diseno-empresas/sistemas/clickhouse/variables.css. ClickHouse es un
// sistema declarado «dark» y no trae variante clara, así que la de aquí está
// derivada: se conserva su gesto —lienzo casi neutro, tres superficies, filete de
// 1px, un único acento eléctrico— y se recalculan los colores de texto para que
// cumplan AA sobre fondo claro.
type Tema struct {
	Nombre string
	Oscuro bool

	// Superficies, de la más al fondo a la más elevada.
	Lienzo    RGB
	Suave     RGB
	Tarjeta   RGB
	Elevada   RGB

	// Filetes: la única separación que usa ClickHouse.
	Filete       RGB
	FileteFuerte RGB

	// Texto, en tres pesos de presencia.
	Tinta   RGB
	Cuerpo  RGB
	Apagado RGB

	// Acento. Relleno es el amarillo de ClickHouse intacto y solo vale como fondo,
	// con SobreAcento encima. Acento es la variante legible: en tema oscuro es el
	// mismo amarillo, y en claro una versión oscurecida hasta cumplir AA.
	//
	// Todo lo que se dibuja en primer plano —el foco de un campo, la barra ▍ de la
	// marca, los medidores, el resultado cifrado— usa Acento, nunca Relleno. Sobre
	// blanco el amarillo da 1,07:1 y la barra de marca desaparecería de la pantalla.
	Relleno     RGB
	RellenoVivo RGB
	SobreAcento RGB
	Acento      RGB

	// Estados. Los tres se usan como texto, así que van pasados por AcentoLegible.
	Exito RGB
	Aviso RGB
	Error RGB
}

// Colores de ClickHouse, con el nombre del token de origen al lado.
var (
	chPrimario     = MustParseHex("#faff69") // --color-primary
	chPrimarioVivo = MustParseHex("#e6eb52") // --color-primary-active
	chTinta        = MustParseHex("#ffffff") // --color-ink
	chCuerpo       = MustParseHex("#cccccc") // --color-body
	chApagado      = MustParseHex("#888888") // --color-muted
	chFilete       = MustParseHex("#2a2a2a") // --color-hairline
	chFileteFuerte = MustParseHex("#3a3a3a") // --color-hairline-strong
	chLienzo       = MustParseHex("#0a0a0a") // --color-canvas
	chSuave        = MustParseHex("#121212") // --color-surface-soft
	chTarjeta      = MustParseHex("#1a1a1a") // --color-surface-card
	chElevada      = MustParseHex("#242424") // --color-surface-elevated
	chSobreAcento  = MustParseHex("#0a0a0a") // --color-on-primary
	chExito        = MustParseHex("#22c55e") // --color-success
	chAviso        = MustParseHex("#f59e0b") // --color-warning
	chError        = MustParseHex("#ef4444") // --color-error
)

// TemaOscuro es ClickHouse sin tocar: sobre su lienzo #0a0a0a los tres colores de
// estado y el amarillo pasan AA de sobra, así que no hay nada que corregir.
var TemaOscuro = Tema{
	Nombre:       "oscuro",
	Oscuro:       true,
	Lienzo:       chLienzo,
	Suave:        chSuave,
	Tarjeta:      chTarjeta,
	Elevada:      chElevada,
	Filete:       chFilete,
	FileteFuerte: chFileteFuerte,
	Tinta:        chTinta,
	Cuerpo:       chCuerpo,
	Apagado:      chApagado,
	Relleno:      chPrimario,
	RellenoVivo:  chPrimarioVivo,
	SobreAcento:  chSobreAcento,
	Acento:       chPrimario,
	Exito:        chExito,
	Aviso:        chAviso,
	Error:        chError,
}

// claroMasOscuro es la superficie de menos contraste del tema claro. Los colores
// de texto se derivan contra ella y no contra el blanco: un acento calculado
// contra #ffffff cumple sobre el lienzo y se queda corto en cuanto cae dentro de
// un panel, que es donde más se usa.
var claroMasOscuro = MustParseHex("#e6e6e6")

// TemaClaro invierte la escala de superficies y recalcula los colores de texto.
//
// Sobre blanco, el amarillo de ClickHouse da 1,07:1, el naranja de aviso 2,2:1 y
// el verde de éxito 2,3:1: los tres son ilegibles y los tres se arreglan igual,
// oscureciéndolos hasta AA. El amarillo original sigue vivo en Relleno, que es
// donde funciona: como fondo, con la tinta encima.
var TemaClaro = Tema{
	Nombre:       "claro",
	Oscuro:       false,
	Lienzo:       MustParseHex("#ffffff"),
	Suave:        MustParseHex("#f7f7f7"),
	Tarjeta:      MustParseHex("#f0f0f0"),
	Elevada:      MustParseHex("#e6e6e6"),
	Filete:       MustParseHex("#d4d4d4"),
	FileteFuerte: MustParseHex("#a8a8a8"),
	Tinta:        MustParseHex("#0a0a0a"),
	Cuerpo:       MustParseHex("#3a3a3a"),
	Apagado:      MustParseHex("#6b6b6b"),
	Relleno:      chPrimario,
	RellenoVivo:  chPrimarioVivo,
	SobreAcento:  chSobreAcento,
	Acento:       AcentoLegible(chPrimario, claroMasOscuro, AANormal),
	Exito:        AcentoLegible(chExito, claroMasOscuro, AANormal),
	Aviso:        AcentoLegible(chAviso, claroMasOscuro, AANormal),
	Error:        AcentoLegible(chError, claroMasOscuro, AANormal),
}

// Glifos de estado. Se imprimen siempre, con color y sin él: cuando el terminal
// no tiene color —NO_COLOR, una tubería, un ConHost antiguo— el glifo es lo
// único que distingue un acierto de un fallo.
const (
	GlifoExito = "✓"
	GlifoAviso = "!"
	GlifoError = "✕"
	GlifoInfo  = "·"
	GlifoBarra = "▍"
)
