package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/rutas"
	"github.com/webcafeina/esfinge/internal/wifi"
)

const laMaestraDePrueba = "una contraseña maestra bien larga"

// conLaEntrada le pone a la orden lo que vendría por una tubería, que es como un
// guion de despliegue le pasa un secreto: `printf '%s' "$TOKEN" | esfinge …`.
func conLaEntrada(t *testing.T, texto string) {
	t.Helper()
	leer, escribir, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	antes := os.Stdin
	os.Stdin = leer
	t.Cleanup(func() { os.Stdin = antes; _ = leer.Close() })
	go func() {
		_, _ = escribir.WriteString(texto)
		_ = escribir.Close()
	}()
}

// lasOpciones son las de siempre para estas pruebas: una bóveda recién hecha y la
// maestra en un fichero, que es el único origen que no necesita terminal.
func lasOpciones(t *testing.T) (*opciones, *opcionesBoveda, string) {
	t.Helper()
	ruta, conLaMaestra := unaBovedaConSecretos(t)
	o := &opciones{tema: "oscuro", silencio: true}
	ob := &opcionesBoveda{ruta: ruta}
	ob.clave.Fichero = conLaMaestra
	return o, ob, ruta
}

// laEntradaGuardada vuelve a abrir el fichero y saca la entrada entera: **lo que
// quede en memoria no lo lee nadie después.**
func laEntradaGuardada(t *testing.T, ruta, titulo string) boveda.Entrada {
	t.Helper()
	b, err := boveda.Abrir(ruta, laMaestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	for _, x := range b.Buscar(titulo) {
		if x.Titulo == titulo {
			entera, _ := b.Ver(x.ID)
			return entera
		}
	}
	t.Fatalf("«%s» no está en la bóveda", titulo)
	return boveda.Entrada{}
}

func correr(t *testing.T, cmd *cobra.Command, banderas map[string]string, args ...string) error {
	t.Helper()
	for k, v := range banderas {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatalf("la bandera --%s no se puede poner: %v", k, err)
		}
	}
	return cmd.RunE(cmd, args)
}

// **El secreto no es un argumento, y hay cuatro sitios de donde sale.**
//
// Es la misma regla que `OrigenClave`: lo que se escribe en la línea queda en el
// historial del shell y lo lee cualquiera con un «ps». Se comprueba pidiéndole a cobra
// las banderas que **no** tienen que existir, que es lo único que lo vigila — una
// prueba de comportamiento pasaría igual con un `--secreto` añadido al lado.
func TestElSecretoNoSePasaComoArgumento(t *testing.T) {
	o, ob, _ := lasOpciones(t)
	for nombre, cmd := range map[string]*cobra.Command{
		"crear":  comandoBovedaCrear(o, ob),
		"editar": comandoBovedaEditar(o, ob),
	} {
		for _, prohibida := range []string{"secreto", "contrasena", "password", "clave"} {
			if cmd.Flags().Lookup(prohibida) != nil {
				t.Errorf("«%s» acepta --%s, y un secreto no puede ir en la línea de órdenes",
					nombre, prohibida)
			}
		}
		for _, hace := range []string{"secreto-stdin", "secreto-env", "secreto-fichero", "generar"} {
			if cmd.Flags().Lookup(hace) == nil {
				t.Errorf("«%s» no tiene --%s, que es uno de los orígenes del secreto", nombre, hace)
			}
		}
	}
}

// Y los tres que traen el secreto de fuera lo guardan igual, porque la diferencia
// entre ellos es de dónde se lee y nada más.
func TestLosTresOrigenesDelSecretoGuardanLoMismo(t *testing.T) {
	const nuevo = "el-testigo-que-llega-de-fuera"

	t.Run("por la entrada estándar", func(t *testing.T) {
		o, ob, ruta := lasOpciones(t)
		conLaEntrada(t, nuevo+"\n") // con el salto que deja cualquier `echo`
		cmd := comandoBovedaCrear(o, ob)
		if err := correr(t, cmd, map[string]string{
			"titulo": "Resend", "secreto-stdin": "true",
		}); err != nil {
			t.Fatal(err)
		}
		if e := laEntradaGuardada(t, ruta, "Resend"); e.Secreto != nuevo {
			t.Errorf("ha guardado %q", e.Secreto)
		}
	})

	t.Run("de una variable de entorno", func(t *testing.T) {
		o, ob, ruta := lasOpciones(t)
		t.Setenv("EL_TESTIGO", nuevo)
		cmd := comandoBovedaCrear(o, ob)
		if err := correr(t, cmd, map[string]string{
			"titulo": "Resend", "secreto-env": "EL_TESTIGO",
		}); err != nil {
			t.Fatal(err)
		}
		if e := laEntradaGuardada(t, ruta, "Resend"); e.Secreto != nuevo {
			t.Errorf("ha guardado %q", e.Secreto)
		}
	})

	t.Run("de un fichero", func(t *testing.T) {
		o, ob, ruta := lasOpciones(t)
		fichero := filepath.Join(t.TempDir(), "testigo.txt")
		if err := os.WriteFile(fichero, []byte(nuevo+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := comandoBovedaCrear(o, ob)
		if err := correr(t, cmd, map[string]string{
			"titulo": "Resend", "secreto-fichero": fichero,
		}); err != nil {
			t.Fatal(err)
		}
		if e := laEntradaGuardada(t, ruta, "Resend"); e.Secreto != nuevo {
			t.Errorf("ha guardado %q", e.Secreto)
		}
	})
}

// **Dos orígenes a la vez no es una preferencia: es que alguien se ha equivocado
// escribiendo el guion**, y adivinar cuál quería es cómo se guarda el secreto de otra
// cosa en la credencial de un cliente. Y ninguno tampoco vale, que si no la entrada
// nacería sin contraseña y pareciendo que se guardó bien.
func TestCrearSeNiegaSinUnSoloOrigenDelSecreto(t *testing.T) {
	o, ob, ruta := lasOpciones(t)

	t.Run("sin ninguno", func(t *testing.T) {
		cmd := comandoBovedaCrear(o, ob)
		err := correr(t, cmd, map[string]string{"titulo": "Resend"})
		if err == nil {
			t.Fatal("ha creado una entrada sin decir de dónde sale el secreto")
		}
		if !strings.Contains(err.Error(), "secreto-stdin") {
			t.Errorf("el error no dice de dónde puede salir: %v", err)
		}
	})

	t.Run("con dos", func(t *testing.T) {
		t.Setenv("EL_TESTIGO", "x")
		cmd := comandoBovedaCrear(o, ob)
		err := correr(t, cmd, map[string]string{
			"titulo": "Resend", "secreto-env": "EL_TESTIGO", "generar": "true",
		})
		if err == nil {
			t.Fatal("ha aceptado dos sitios de donde sacar el secreto")
		}
	})

	// Y sin título tampoco: una entrada que no se puede nombrar no se puede volver a
	// encontrar, y lo que la línea de comandos escribe no lo mira nadie después.
	t.Run("sin título", func(t *testing.T) {
		conLaEntrada(t, "x")
		cmd := comandoBovedaCrear(o, ob)
		if err := correr(t, cmd, map[string]string{"secreto-stdin": "true"}); err == nil {
			t.Fatal("ha creado una entrada sin título")
		}
	})

	// En los tres casos, **nada ha entrado en la bóveda**.
	b, err := boveda.Abrir(ruta, laMaestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	if hay := b.Buscar("Resend"); len(hay) != 0 {
		t.Errorf("ha quedado algo guardado: %d entradas", len(hay))
	}
}

// **`--generar` es el mejor caso de todos: la contraseña no la ve nadie**, ni quien la
// pide. Lo que sale por la pantalla es **cómo es** —largo y alfabeto—, que es lo que
// hace falta para saber si cumple lo que exige el sitio, y nunca cuál.
//
// Se busca el secreto **en los bytes** de lo que Esfinge escribe, igual que en las
// pruebas del canal de los agentes: comprobar que la frase «parece» correcta no
// distingue decir el largo de decir la contraseña.
func TestGenerarNoEnsenaLaContrasenaYSiComoEs(t *testing.T) {
	o, ob, ruta := lasOpciones(t)
	o.silencio = false // que hable, que es lo que se está mirando

	cmd := comandoBovedaCrear(o, ob)
	dicho := loQueSaleAlError(t, func() {
		if err := correr(t, cmd, map[string]string{
			"titulo": "Resend", "generar": "true", "alfabeto": "alnum", "caracteres": "40",
		}); err != nil {
			t.Fatal(err)
		}
	})

	e := laEntradaGuardada(t, ruta, "Resend")
	if n := len([]rune(e.Secreto)); n != 40 {
		t.Fatalf("ha guardado una contraseña de %d caracteres", n)
	}
	if strings.Contains(dicho, e.Secreto) {
		t.Error("la contraseña generada sale por la pantalla")
	}
	for _, trozo := range []string{"40", "alnum"} {
		if !strings.Contains(dicho, trozo) {
			t.Errorf("no dice cómo es la contraseña, falta %q en %q", trozo, dicho)
		}
	}
}

// **Editar cambia solo lo que se nombra.** Y eso no se decide por el valor sino por si
// la bandera se usó: `--notas ""` es «bórrame las notas» y no decir `--notas` es
// «déjalas». Con el valor vacío las dos cosas se ven igual, y entonces editar el
// usuario de una credencial le borraría las notas, el sitio y la carpeta de paso.
func TestEditarCambiaSoloLoQueSeNombra(t *testing.T) {
	o, ob, ruta := lasOpciones(t)

	// Una entrada con todo puesto, para que se note lo que se pierde.
	b, err := boveda.Abrir(ruta, laMaestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Poner(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Brevo", Usuario: "antiguo@zeri.es",
		Notas: "la cuenta de los envíos", Carpeta: "Zeri", Sitios: []string{"app.brevo.com"},
		Etiquetas: []string{"correo"}, Secreto: "la-de-antes",
	}); err != nil {
		t.Fatal(err)
	}
	b.Cerrar()

	cmd := comandoBovedaEditar(o, ob)
	if err := correr(t, cmd, map[string]string{"usuario": "nuevo@zeri.es"}, "Brevo"); err != nil {
		t.Fatal(err)
	}

	e := laEntradaGuardada(t, ruta, "Brevo")
	if e.Usuario != "nuevo@zeri.es" {
		t.Errorf("no ha cambiado el usuario: %q", e.Usuario)
	}
	if e.Notas != "la cuenta de los envíos" {
		t.Errorf("se ha llevado las notas: %q", e.Notas)
	}
	if e.Carpeta != "Zeri" {
		t.Errorf("se ha llevado la carpeta: %q", e.Carpeta)
	}
	if len(e.Sitios) != 1 || e.Sitios[0] != "app.brevo.com" {
		t.Errorf("se ha llevado los sitios: %v", e.Sitios)
	}
	if len(e.Etiquetas) != 1 {
		t.Errorf("se ha llevado las etiquetas: %v", e.Etiquetas)
	}
	if e.Secreto != "la-de-antes" {
		t.Errorf("ha tocado el secreto sin que nadie se lo pidiera: %q", e.Secreto)
	}

	// Y lo nombrado vacío **sí** borra, que es la otra mitad de la misma regla.
	cmd = comandoBovedaEditar(o, ob)
	if err := correr(t, cmd, map[string]string{"notas": ""}, "Brevo"); err != nil {
		t.Fatal(err)
	}
	if e := laEntradaGuardada(t, ruta, "Brevo"); e.Notas != "" {
		t.Errorf("no ha borrado las notas que se le nombraron vacías: %q", e.Notas)
	}

	// Y sin nombrar nada no hay nada que hacer, y se dice en vez de contestar que sí.
	cmd = comandoBovedaEditar(o, ob)
	if err := correr(t, cmd, nil, "Brevo"); err == nil {
		t.Error("ha contestado que ha cambiado algo sin que nadie dijera qué")
	}
}

// **Pisar un secreto deja el anterior en el historial de esa entrada**, y eso es el
// argumento con el que la ADR 0058 se sostiene: dentro de la bóveda esto se deshace,
// así que escribir sin que nadie apruebe no pierde nada que no estuviera perdido ya
// —lo irreversible pasó en el proveedor—.
//
// Pasa por `CambiarSecreto` y no por `e.Secreto = …`, que es lo único que lo consigue.
func TestEditarDejaElSecretoAnteriorEnElHistorial(t *testing.T) {
	o, ob, ruta := lasOpciones(t)
	conLaEntrada(t, "el-testigo-nuevo")

	cmd := comandoBovedaEditar(o, ob)
	if err := correr(t, cmd, map[string]string{"secreto-stdin": "true"}, "Cloudflare API"); err != nil {
		t.Fatal(err)
	}

	e := laEntradaGuardada(t, ruta, "Cloudflare API")
	if e.Secreto != "el-testigo-nuevo" {
		t.Fatalf("no ha cambiado el secreto: %q", e.Secreto)
	}
	if len(e.Historial) != 1 {
		t.Fatalf("el historial tiene %d entradas y tendría que tener la de antes", len(e.Historial))
	}
	if e.Historial[0].Secreto != elSecretoDePrueba {
		t.Errorf("en el historial no está el secreto anterior: %q", e.Historial[0].Secreto)
	}
	if e.Historial[0].Hasta == "" {
		t.Error("el historial no dice hasta cuándo valió")
	}
}

// **Queda apuntado, y dice quién lo pidió.** `--quien` no autentica nada —es lo que
// dice de sí mismo quien escribe el guion, igual que el nombre de un cliente MCP— y
// por eso sirve para leer el registro después, no para dar permisos.
//
// Y `Como` dice `llave-del-equipo`, que aquí significa lo que dice su comentario: **no
// había a quién preguntar**. Sin ese campo, los apuntes de la línea de comandos se
// leerían al lado de los `(preguntado)` del Mac como si alguien los hubiera aprobado.
func TestLoEscritoQuedaApuntadoConQuienLoPidio(t *testing.T) {
	o, ob, ruta := lasOpciones(t)
	const secreto = "el-testigo-de-resend-que-no-puede-salir"
	conLaEntrada(t, secreto)

	cmd := comandoBovedaCrear(o, ob)
	if err := correr(t, cmd, map[string]string{
		"titulo": "Resend", "secreto-stdin": "true", "quien": "Serafin",
	}); err != nil {
		t.Fatal(err)
	}

	b, err := boveda.Abrir(ruta, laMaestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	reg := b.Registro()
	if len(reg) != 1 {
		t.Fatalf("hay %d apuntes en el registro", len(reg))
	}
	a := reg[0]
	if a.Quien != "Serafin" {
		t.Errorf("el apunte no dice quién lo pidió: %q", a.Quien)
	}
	if a.Que != "crear" || a.Titulo != "Resend" {
		t.Errorf("el apunte no dice qué pasó: %+v", a)
	}
	if a.Como != boveda.ApunteLlaveDelEquipo {
		t.Errorf("el apunte dice que lo aprobó alguien: %q", a.Como)
	}
	if a.Resultado != boveda.ApunteHecho {
		t.Errorf("resultado %q", a.Resultado)
	}
	if a.Sobre == "" {
		t.Error("el apunte no dice de qué entrada se trata")
	}
	// Y el secreto no está en claro en ninguna parte del fichero: el apunte no lo
	// lleva y el cuerpo va cifrado. Se busca **en los bytes**, que es lo único que
	// distingue «no lo escribe» de «lo escribe donde no he mirado».
	crudo, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(crudo), secreto) {
		t.Error("el fichero lleva el secreto en claro")
	}
}

// Sin `--quien`, el apunte lleva el nombre del programa: es lo único cierto que se
// sabe, y dejarlo vacío haría un registro que no contesta la pregunta para la que
// existe.
func TestSinQuienElApunteDiceQueFueLaLineaDeComandos(t *testing.T) {
	o, ob, ruta := lasOpciones(t)
	conLaEntrada(t, "x")

	cmd := comandoBovedaCrear(o, ob)
	if err := correr(t, cmd, map[string]string{"titulo": "Resend", "secreto-stdin": "true"}); err != nil {
		t.Fatal(err)
	}
	b, err := boveda.Abrir(ruta, laMaestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	if reg := b.Registro(); len(reg) != 1 || reg[0].Quien == "" {
		t.Fatalf("el apunte no dice quién: %+v", reg)
	}
}

// **El tipo se comprueba contra la lista.** Un tipo mal escrito crea una entrada que la
// ventana no sabe dibujar y que en pantalla sale sin formulario: un fallo mudo que
// aparece en otro programa y no aquí.
//
// Y **una llave de acceso no se crea a mano** (ADR 0048): la emite el sitio, y lo que
// se guardara aquí no abriría ninguna cuenta. Se dice aparte de «no existe» porque el
// tipo sí existe y la razón es otra.
func TestElTipoSeComprueba(t *testing.T) {
	for _, caso := range []struct {
		tipo string
		vale bool
		dice string
	}{
		{tipo: "", vale: true},
		{tipo: "credencial", vale: true},
		{tipo: "wifi", vale: true},
		{tipo: "IDENTIDAD", vale: true}, // da igual cómo se escriba
		{tipo: "llave", vale: false, dice: "la emite el sitio"},
		{tipo: "credencia", vale: false, dice: "no existe"},
		{tipo: "documento", vale: false, dice: "no existe"}, // se llama «identidad»
	} {
		_, err := elTipoQueSePide(caso.tipo)
		if caso.vale && err != nil {
			t.Errorf("el tipo %q tendría que valer: %v", caso.tipo, err)
			continue
		}
		if !caso.vale {
			if err == nil {
				t.Errorf("el tipo %q ha pasado", caso.tipo)
				continue
			}
			if !strings.Contains(err.Error(), caso.dice) {
				t.Errorf("el tipo %q da un error que no explica por qué: %v", caso.tipo, err)
			}
		}
	}
}

// ---------------------------------------------- en una compartida de solo ver

// unaCompartidaEnEsteEquipo monta **sin ningún servidor** lo que en producción deja un
// acceso ya aceptado: la bóveda personal de este equipo, una bóveda ajena cuya ranura
// abre la identidad de la personal, su fichero donde la línea de comandos lo busca, y
// la fila en la lista con el permiso que le dieron.
//
// Es la única forma de ejercitar `--compartida` desde aquí: el paseo de dos equipos
// hace esto mismo contra el servidor de verdad y por eso tarda un minuto.
func unaCompartidaEnEsteEquipo(t *testing.T, permiso string) (o *opciones, ob *opcionesBoveda, ajena string) {
	t.Helper()
	rutas.ApartarLaCarpeta(t.TempDir(), t.Setenv)

	mia, _, err := boveda.Crear(rutas.BovedaPrincipal(), laMaestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	yo, err := mia.Identidad()
	if err != nil {
		t.Fatal(err)
	}

	const dueno, ref, titular = "aaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbb", "cccccccc"
	ajena = rutas.Compartida(dueno, ref)
	otra, _, err := boveda.Crear(ajena, "la maestra de la otra persona, bien larga")
	if err != nil {
		t.Fatal(err)
	}
	if err := otra.Poner(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Cloudflare de Zeri", Secreto: "el-testigo-de-zeri",
	}); err != nil {
		t.Fatal(err)
	}
	// La ranura **sellada hacia mi identidad pública**, que es la decisión entera de la
	// ADR 0052 y lo que hace que esto se pueda montar sin tener a nadie delante.
	if err := otra.PonerAcceso(titular, yo); err != nil {
		t.Fatal(err)
	}
	otra.Cerrar()

	if err := mia.PonerCompartida(boveda.Compartida{
		Dueno: dueno, Ref: ref, Nombre: "Zeri's Coffee",
		Titular: titular, Permiso: permiso, Desde: ahoraRFC3339(),
	}); err != nil {
		t.Fatal(err)
	}
	mia.Cerrar()

	conLaMaestra := filepath.Join(t.TempDir(), "maestra.txt")
	if err := os.WriteFile(conLaMaestra, []byte(laMaestraDePrueba), 0o600); err != nil {
		t.Fatal(err)
	}
	o = &opciones{tema: "oscuro", silencio: true}
	ob = &opcionesBoveda{compartida: "Zeri's", sinSincronizar: true}
	ob.clave.Fichero = conLaMaestra
	return o, ob, ajena
}

// **Con permiso de solo ver, los dos verbos se niegan y dicen por qué.**
//
// Y el porqué importa: la clave para descifrarla **sí está** —escribir funcionaría— y
// lo que no se puede es subirlo, porque el servidor contesta 403. Decir «no se puede
// escribir» a secas mandaría a buscar un permiso de fichero.
//
// Lo que esto vigila de verdad es que la regla de `boveda.PorQueNoSeEscribe` la
// pregunte **también** la línea de comandos: la ventana y el navegador ya la
// preguntaban, y una regla en dos listas se desalinea sola.
func TestEnUnaCompartidaDeSoloVerNoSeEscribe(t *testing.T) {
	for _, caso := range []struct{ verbo string }{{"crear"}, {"editar"}} {
		t.Run(caso.verbo, func(t *testing.T) {
			o, ob, ajena := unaCompartidaEnEsteEquipo(t, "ver")
			conLaEntrada(t, "lo-que-no-tendria-que-entrar")

			var err error
			if caso.verbo == "crear" {
				err = correr(t, comandoBovedaCrear(o, ob), map[string]string{
					"titulo": "Lo que el servidor no puede guardar", "secreto-stdin": "true",
				})
			} else {
				err = correr(t, comandoBovedaEditar(o, ob), map[string]string{
					"secreto-stdin": "true",
				}, "Cloudflare de Zeri")
			}
			if err == nil {
				t.Fatal("ha escrito en una bóveda compartida de solo ver")
			}
			if !strings.Contains(err.Error(), "Solo puedes ver") {
				t.Errorf("el error no dice por qué: %v", err)
			}

			// Y **no ha escrito nada**, que es lo que de verdad se protege: el mensaje
			// correcto con la entrada dentro sería peor que un mensaje malo.
			seGuardoAlgo(t, ajena, "Lo que el servidor no puede guardar", "lo-que-no-tendria-que-entrar")
		})
	}
}

// Y con permiso de editar, la misma bóveda sí se escribe: sin esto, la prueba de
// arriba pasaría igual con la escritura rota del todo.
func TestEnUnaCompartidaDeEditarSiSeEscribe(t *testing.T) {
	o, ob, ajena := unaCompartidaEnEsteEquipo(t, "editar")
	conLaEntrada(t, "el-testigo-nuevo-de-zeri")

	if err := correr(t, comandoBovedaEditar(o, ob), map[string]string{
		"secreto-stdin": "true",
	}, "Cloudflare de Zeri"); err != nil {
		t.Fatalf("con permiso de editar no ha podido escribir: %v", err)
	}

	b, err := laAjena(t, ajena)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	hay := b.Buscar("Cloudflare de Zeri")
	if len(hay) != 1 {
		t.Fatalf("en la bóveda ajena hay %d entradas que encajen", len(hay))
	}
	e, _ := b.Ver(hay[0].ID)
	if e.Secreto != "el-testigo-nuevo-de-zeri" {
		t.Errorf("no ha guardado el secreto nuevo: %q", e.Secreto)
	}
	if len(e.Historial) != 1 || e.Historial[0].Secreto != "el-testigo-de-zeri" {
		t.Errorf("el anterior no ha caído al historial: %+v", e.Historial)
	}
}

// laAjena abre la bóveda compartida como lo hace la línea de comandos: **a través de
// la personal**, que es de donde sale la identidad que abre su ranura.
func laAjena(t *testing.T, ruta string) (*boveda.Boveda, error) {
	t.Helper()
	mia, err := boveda.Abrir(rutas.BovedaPrincipal(), laMaestraDePrueba)
	if err != nil {
		return nil, err
	}
	t.Cleanup(mia.Cerrar)
	comp, err := laCompartidaQueSeLlama(mia, "Zeri's")
	if err != nil {
		return nil, err
	}
	return boveda.AbrirCompartida(ruta, comp.Titular, mia)
}

// seGuardoAlgo se queja si en la bóveda ajena ha entrado el título o el secreto que no
// tenían que entrar.
func seGuardoAlgo(t *testing.T, ruta, titulo, secreto string) {
	t.Helper()
	b, err := laAjena(t, ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	if hay := b.Buscar(titulo); len(hay) != 0 {
		t.Errorf("ha entrado «%s» en la bóveda de solo ver", titulo)
	}
	for _, x := range b.Buscar("") {
		e, _ := b.Ver(x.ID)
		if e.Secreto == secreto {
			t.Errorf("el secreto ha entrado en «%s», que es de solo ver", e.Titulo)
		}
	}
}

// **`-q` existe en «boveda».** Lo declaran `cifrar` y `descifrar` y aquí faltaba, así
// que `esfinge boveda ver x -q` moría con «unknown flag» justo en el programa que más
// se mete en un guion de despliegue, y donde las frases de cortesía molestan más.
//
// Se mira en el comando de verdad, con sus hijos, porque lo que falló fue el cableado.
func TestLaBovedaAceptaSilencio(t *testing.T) {
	o := &opciones{tema: "oscuro"}
	cmd := comandoBoveda(o)
	if f := cmd.PersistentFlags().Lookup("silencio"); f == nil {
		t.Fatal("«boveda» no declara --silencio")
	}
	if f := cmd.PersistentFlags().ShorthandLookup("q"); f == nil {
		t.Error("«boveda» no declara -q")
	}
	// Y lo heredan los hijos, que es por donde se usa.
	for _, hijo := range cmd.Commands() {
		if hijo.InheritedFlags().Lookup("silencio") == nil {
			t.Errorf("«boveda %s» no hereda --silencio", hijo.Name())
		}
	}
}

// TestUnaRedWifiSeCreaEnteraONoSeCrea cubre los tres campos de la red (ADR 0049), que
// hasta el 2026-11-12 no se podían poner desde aquí.
//
// # Por qué existen en la línea de comandos
//
// Parecía que no hacían falta —un servidor no se conecta a ningún wifi— y el cliente
// señaló el caso que lo tira: **una bóveda de proyecto se gestiona donde se gestione el
// proyecto**, y eso puede ser el VPS, mientras el código QR se enseña en el equipo de
// quien lo necesita. El que crea la red y el que la usa no son el mismo.
//
// Y lo que de verdad se arregla no es que faltaran banderas: es que **sin ellas se podía
// crear una red que no sirve** —sin SSID no hay QR— y nada lo decía al crearla. Es la
// lección de `ExportarLlaves`.
func TestUnaRedWifiSeCreaEnteraONoSeCrea(t *testing.T) {
	o, ob, ruta := lasOpciones(t)

	// **Sin SSID no se guarda nada**, y el mensaje dice qué falta y por qué no es el título.
	err := correr(t, comandoBovedaCrear(o, ob), map[string]string{
		"tipo": "wifi", "titulo": "Sin nombre", "generar": "true",
	})
	if err == nil {
		t.Fatal("una red sin SSID se ha guardado: el QR no se podría dibujar nunca")
	}
	if !strings.Contains(err.Error(), "--ssid") {
		t.Errorf("el error no dice qué falta: %v", err)
	}

	// Con SSID, la red queda **entera**: su nombre, y la seguridad deducida de que hay
	// contraseña —no de lo que diga nadie, que es la regla de la 0049—.
	if err := correr(t, comandoBovedaCrear(o, ob), map[string]string{
		"tipo": "wifi", "titulo": "La oficina", "ssid": "ZeriGuest", "generar": "true",
	}); err != nil {
		t.Fatalf("no se ha podido crear la red: %v", err)
	}
	red := laEntradaGuardada(t, ruta, "La oficina")
	if red.Tipo != boveda.TipoWifi {
		t.Errorf("la entrada es %s", red.Tipo)
	}
	if red.SSID != "ZeriGuest" {
		t.Errorf("el SSID es %q", red.SSID)
	}
	if red.Seguridad != "wpa" {
		t.Errorf("con contraseña, la seguridad tiene que ser wpa y es %q", red.Seguridad)
	}
	if red.Secreto == "" {
		t.Error("la red se ha guardado sin contraseña")
	}

	// **Y el QR se puede dibujar**, que es lo único que prueba que la red sirve: el
	// comando que la consume es el que antes la rechazaba.
	if _, err := wifi.Enlace(red.SSID, red.Secreto, red.Seguridad, red.Oculta); err != nil {
		t.Errorf("la red creada aquí no se puede poner en un código: %v", err)
	}

	// **Un campo de una clase no entra en otra.** Sin esto, el SSID se guardaba en una
	// credencial, donde no se lee y nadie lo vería nunca.
	err = correr(t, comandoBovedaCrear(o, ob), map[string]string{
		"titulo": "Una credencial", "ssid": "loquesea", "generar": "true",
	})
	if err == nil || !strings.Contains(err.Error(), "red wifi") {
		t.Errorf("el SSID en una credencial contesta %v", err)
	}
}

// TestLosCamposQueSePuedenSeAplicanTodos recorre la tabla de campos y **ejercita cada
// uno por `editar`**, comparándola con una lista de casos escrita aquí.
//
// # Por qué se compara contra otra lista
//
// La tabla nació juntando tres —las banderas, el «qué se nombró» y el `switch` que
// parcheaba la entrada—, porque olvidar uno de los tres **no da error**: la bandera se
// declara, nadie la aplica y `editar --ssid` contesta «no has dicho qué cambiar». Con una
// sola lista eso ya no puede pasar por descuido, pero sí puede pasar que alguien añada un
// campo y nadie lo pruebe. Esto se pone rojo entonces: es la regla de la casa de comparar
// las dos listas entre sí, no cada una contra lo que parezca correcto.
func TestLosCamposQueSePuedenSeAplicanTodos(t *testing.T) {
	casos := map[string]struct {
		valor  string
		tipo   boveda.Tipo
		puesto func(boveda.Entrada) bool
	}{
		"titulo":    {"Otro título", "", func(e boveda.Entrada) bool { return e.Titulo == "Otro título" }},
		"usuario":   {"alguien@ejemplo.com", "", func(e boveda.Entrada) bool { return e.Usuario == "alguien@ejemplo.com" }},
		"notas":     {"unas notas", "", func(e boveda.Entrada) bool { return e.Notas == "unas notas" }},
		"carpeta":   {"Clientes", "", func(e boveda.Entrada) bool { return e.Carpeta == "Clientes" }},
		"sitio":     {"ejemplo.com", "", func(e boveda.Entrada) bool { return len(e.Sitios) == 1 && e.Sitios[0] == "ejemplo.com" }},
		"etiqueta":  {"una", "", func(e boveda.Entrada) bool { return len(e.Etiquetas) == 1 && e.Etiquetas[0] == "una" }},
		"ssid":      {"OtraRed", boveda.TipoWifi, func(e boveda.Entrada) bool { return e.SSID == "OtraRed" }},
		"seguridad": {"wep", boveda.TipoWifi, func(e boveda.Entrada) bool { return e.Seguridad == "wep" }},
		"oculta":    {"true", boveda.TipoWifi, func(e boveda.Entrada) bool { return e.Oculta }},
	}

	// Las dos listas, una contra otra y en los dos sentidos.
	for _, campo := range losCamposQueSePueden {
		if _, hay := casos[campo.bandera]; !hay {
			t.Errorf("el campo --%s está en la tabla y no lo prueba nadie", campo.bandera)
		}
	}
	for bandera := range casos {
		visto := false
		for _, campo := range losCamposQueSePueden {
			if campo.bandera == bandera {
				visto = true
			}
		}
		if !visto {
			t.Errorf("se prueba --%s y ya no está en la tabla", bandera)
		}
	}

	for _, campo := range losCamposQueSePueden {
		caso, hay := casos[campo.bandera]
		if !hay {
			continue
		}
		t.Run(campo.bandera, func(t *testing.T) {
			o, ob, ruta := lasOpciones(t)
			titulo := "Para " + campo.bandera
			nace := map[string]string{"titulo": titulo, "generar": "true"}
			if caso.tipo == boveda.TipoWifi {
				nace["tipo"] = "wifi"
				nace["ssid"] = "LaDeAntes"
			}
			if err := correr(t, comandoBovedaCrear(o, ob), nace); err != nil {
				t.Fatalf("crear: %v", err)
			}
			if err := correr(t, comandoBovedaEditar(o, ob), map[string]string{campo.bandera: caso.valor}, titulo); err != nil {
				t.Fatalf("editar --%s: %v", campo.bandera, err)
			}
			buscar := titulo
			if campo.bandera == "titulo" {
				buscar = caso.valor
			}
			if e := laEntradaGuardada(t, ruta, buscar); !caso.puesto(e) {
				t.Errorf("--%s se declara y no se aplica: la entrada guardada no lo lleva", campo.bandera)
			}
		})
	}
}
