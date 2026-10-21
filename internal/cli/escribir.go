package cli

// Crear y cambiar entradas desde la línea de comandos (ADR 0058).
//
// Hasta aquí la bóveda en la línea de comandos **solo leía**, y con una razón escrita
// en la cabecera de `boveda.go`: editar podía esperar y consultar no. Lo que cambió es
// la ADR 0057: hay agentes trabajando en un servidor sin ventana, y **el agente que
// rota un testigo tiene que poder guardarlo**. Si no puede, se lo tiene que dar a una
// persona por el chat — o sea que el secreto acaba en la conversación de un modelo,
// que es justo lo que la 0054 y la 0057 existen para evitar.
//
// # Lo que no se aprueba, y por qué eso no contradice a la 0054
//
// La 0054 dice que la válvula no cubre «lo que no se deshace — cambiar una
// contraseña». Mirado de cerca, aquí cae del otro lado:
//
//   - **Dentro de la bóveda sí se deshace**: `CambiarSecreto` empuja el anterior al
//     historial antes de escribir el nuevo.
//   - **Lo irreversible pasó fuera**: si el agente ya rotó el testigo en el
//     proveedor, el viejo está muerto se guarde o no.
//   - Así que en una máquina sin pantalla **no guardarlo es más peligroso que
//     guardarlo**.
//
// En la ventana no cambia nada: allí hay alguien delante y el sí no cuesta nada.
//
// # Cuatro cosas que no son detalles
//
//   - **El secreto no es nunca un argumento.** Es la misma regla que `OrigenClave`:
//     lo que se escribe en la línea queda en el historial del shell y lo lee un `ps`.
//     Hay cuatro orígenes y ninguno es la línea de órdenes.
//   - **`--generar` es el mejor caso**: la contraseña la fabrica Esfinge, se guarda y
//     **no la ve nadie**, ni el agente. Lo que sale por la pantalla es cómo es —largo
//     y alfabeto—, nunca cuál.
//   - **`editar` parchea campo a campo** sobre lo que ya hay, nunca construyendo una
//     entrada nueva: construirla se lleva por delante `Extra` —los campos que escribió
//     una versión más nueva de Esfinge— y todo lo que no se nombre.
//   - **Borrar no está.** En la ventana pide un sí y aquí no hay quién lo dé, y un
//     servidor no tiene ninguna razón para borrar la credencial de un cliente. Es una
//     frontera barata de sostener, que es lo que la hace frontera.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
)

// elSecretoDeFabrica son los bytes de entropía de `--generar` cuando nadie dice un
// largo: los mismos que `esfinge generar` por defecto.
const elSecretoDeFabrica = 24

// origenDelSecreto son las cuatro formas de dar el secreto, y **ninguna es un
// argumento de la línea de órdenes**.
type origenDelSecreto struct {
	stdin      bool
	variable   string
	fichero    string
	generar    bool
	alfabeto   string
	caracteres int
}

// dicho es si alguien ha pedido poner un secreto. Vale para `editar`, donde no
// decirlo significa dejarlo como está.
func (s origenDelSecreto) dicho() bool {
	return s.stdin || s.variable != "" || s.fichero != "" || s.generar
}

// comoEsLoGenerado es lo que se puede decir de una contraseña que nadie ha visto.
// **Cómo es, nunca cuál**: sin esto, quien la pide no puede saber si cumple lo que
// exige el sitio y lo único que le queda es pedir que se la enseñen.
type comoEsLoGenerado struct {
	largo    int
	alfabeto string
}

// leer saca el secreto de donde se haya dicho.
//
// **Solo de un sitio**: dos orígenes a la vez no es una preferencia, es que alguien se
// ha equivocado escribiendo el guion y adivinar cuál quería es cómo se guarda el
// secreto de otra cosa.
func (s origenDelSecreto) leer() (string, *comoEsLoGenerado, error) {
	cuantos := 0
	for _, hay := range []bool{s.stdin, s.variable != "", s.fichero != "", s.generar} {
		if hay {
			cuantos++
		}
	}
	if cuantos == 0 {
		return "", nil, errUso{errors.New(
			"Dime de dónde sale el secreto: --secreto-stdin, --secreto-env, --secreto-fichero o --generar")}
	}
	if cuantos > 1 {
		return "", nil, errUso{errors.New(
			"Dos sitios distintos de donde sacar el secreto; deja solo uno")}
	}

	switch {
	case s.generar:
		a, ok := cripto.Alfabetos[s.alfabeto]
		if !ok {
			return "", nil, errUso{fmt.Errorf("El alfabeto «%s» no existe: %s",
				s.alfabeto, alfabetosDisponibles())}
		}
		var clave string
		var err error
		if s.caracteres > 0 {
			clave, err = cripto.GenerarDeLargo(a, s.caracteres)
		} else {
			clave, err = cripto.Generar(a, elSecretoDeFabrica)
		}
		if err != nil {
			return "", nil, err
		}
		return clave, &comoEsLoGenerado{largo: len([]rune(clave)), alfabeto: a.Nombre}, nil

	case s.variable != "":
		v, hay := os.LookupEnv(s.variable)
		if !hay {
			return "", nil, fmt.Errorf("La variable de entorno %s no existe", s.variable)
		}
		if v == "" {
			return "", nil, fmt.Errorf("La variable de entorno %s está vacía", s.variable)
		}
		return v, nil, nil

	case s.fichero != "":
		b, err := os.ReadFile(s.fichero)
		if err != nil {
			return "", nil, fmt.Errorf("No puedo leer el secreto de %s: %w", s.fichero, err)
		}
		// Igual que un fichero de clave: el salto final no lo quiso poner nadie.
		v := strings.TrimRight(string(b), "\r\n")
		if v == "" {
			return "", nil, fmt.Errorf("El fichero %s está vacío", s.fichero)
		}
		return v, nil, nil
	}

	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", nil, fmt.Errorf("No he podido leer el secreto de la entrada estándar: %w", err)
	}
	v := strings.TrimRight(string(b), "\r\n")
	if v == "" {
		return "", nil, errors.New("Por la entrada estándar no ha llegado nada")
	}
	return v, nil, nil
}

// losCamposQueSePueden son los que estos dos verbos saben poner, con su bandera.
//
// Están aquí y no repartidos por los dos comandos porque la lista es la misma, y una
// lista en dos sitios se desalinea sola: en este proyecto ya ha costado tres fallos.
type losCampos struct {
	titulo, usuario, notas, carpeta, tipo string
	sitios, etiquetas                     []string
}

func banderasDeLosCampos(cmd *cobra.Command, c *losCampos) {
	f := cmd.Flags()
	f.StringVar(&c.titulo, "titulo", "", "Cómo se llama la entrada")
	f.StringVar(&c.usuario, "usuario", "", "El usuario o el correo con el que se entra")
	f.StringVar(&c.notas, "notas", "", "Notas, que van cifradas como todo lo demás")
	f.StringVar(&c.carpeta, "carpeta", "", "La carpeta donde ponerla")
	f.StringArrayVar(&c.sitios, "sitio", nil,
		"Un dominio donde vale esta credencial; se puede repetir")
	f.StringArrayVar(&c.etiquetas, "etiqueta", nil, "Una etiqueta; se puede repetir")
}

// ------------------------------------------------------------------ crear

func comandoBovedaCrear(o *opciones, ob *opcionesBoveda) *cobra.Command {
	var (
		c     losCampos
		s     origenDelSecreto
		quien string
	)
	cmd := &cobra.Command{
		Use:   "crear --titulo NOMBRE (--secreto-stdin | --generar)",
		Short: "Guarda una entrada nueva en la bóveda",
		Long: "Crea una entrada con el secreto que se le diga, o con uno que fabrica\n" +
			"Esfinge y **que no ve nadie**.\n\n" +
			"**El secreto no se pasa como argumento**, nunca: lo que se escribe en la\n" +
			"línea queda en el historial del shell y lo lee cualquiera con un «ps».\n" +
			"Sale de --secreto-stdin, --secreto-env VARIABLE, --secreto-fichero RUTA\n" +
			"o de --generar, que es el mejor caso porque no pasa por ninguna parte.\n\n" +
			"Con --generar salen 48 caracteres en hexadecimal; --caracteres los cambia\n" +
			"y --alfabeto elige hex, alnum o simbolos. Lo que se dice por pantalla es\n" +
			"**cómo es** la contraseña, nunca cuál.\n\n" +
			"Pon --sitio con el dominio donde vale: sin eso no se rellena sola en el\n" +
			"navegador, que es para lo que existe la bóveda.\n\n" +
			"Queda apuntado en el registro de la bóveda. Y ojo con una cosa que no se\n" +
			"ve: si en este mismo equipo hay una ventana de Esfinge abierta, su lista\n" +
			"se queda con lo de antes hasta que algo la refresque. Lo guardado está\n" +
			"guardado y sube igual; lo que no se entera es la pantalla.",
		Example: "  printf '%s' \"$TOKEN\" | esfinge boveda crear --compartida Tempero \\\n" +
			"      --titulo \"Cloudflare API\" --usuario zeri --sitio dash.cloudflare.com --secreto-stdin\n" +
			"  esfinge boveda crear --titulo Resend --generar --alfabeto alnum --caracteres 40",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(c.titulo) == "" {
				return errUso{errors.New("Hace falta un título, con --titulo")}
			}
			tipo, err := elTipoQueSePide(c.tipo)
			if err != nil {
				return err
			}
			secreto, comoEs, err := s.leer()
			if err != nil {
				return err
			}

			d, err := abrirParaEscribir(o, ob)
			if err != nil {
				return err
			}
			defer d.cerrar()
			if porque := d.porQueNoSeEscribe(); porque != "" {
				return errors.New(porque)
			}

			e := boveda.Entrada{
				Tipo: tipo, Titulo: c.titulo, Usuario: c.usuario,
				Notas: c.notas, Carpeta: c.carpeta,
				Sitios: c.sitios, Etiquetas: c.etiquetas,
				Secreto: secreto,
			}
			if err := d.b.Poner(e); err != nil {
				return err
			}
			// El identificador lo pone la bóveda al guardar, así que se busca por el
			// título — lo mismo que hace el canal de los agentes.
			id := ""
			for _, x := range d.b.Buscar(c.titulo) {
				if x.Titulo == c.titulo {
					id = x.ID
				}
			}

			d.apuntar(o, boveda.Apunte{
				Quien: quienDice(quien), Que: "crear", Sobre: id, Titulo: c.titulo,
			})
			d.decir(o, loQuePaso{
				que: "crear", titulo: c.titulo, id: id, generada: comoEs,
			})
			return d.subir(o)
		},
	}
	banderasDeLosCampos(cmd, &c)
	cmd.Flags().StringVar(&c.tipo, "tipo", "",
		"credencial, nota, tarjeta, identidad, personal o wifi (por defecto, credencial)")
	banderasDelSecreto(cmd, &s)
	banderasDeQuien(cmd, &quien)
	return cmd
}

// ------------------------------------------------------------------ editar

func comandoBovedaEditar(o *opciones, ob *opcionesBoveda) *cobra.Command {
	var (
		c     losCampos
		s     origenDelSecreto
		quien string
	)
	cmd := &cobra.Command{
		Use:   "editar <búsqueda> [--secreto-stdin | --generar | --campo VALOR]",
		Short: "Cambia una entrada que ya está en la bóveda",
		Long: "Cambia **solo lo que se nombre** de una entrada que ya existe. Lo que no\n" +
			"se diga se queda como está.\n\n" +
			"Es lo que hace falta para rotar un secreto: el anterior no se pierde, cae\n" +
			"al historial de esa entrada, que es lo que deja volver atrás si el sitio\n" +
			"no acepta el nuevo.\n\n" +
			"El secreto no se pasa como argumento, igual que en «crear»: sale de\n" +
			"--secreto-stdin, --secreto-env, --secreto-fichero o --generar.\n\n" +
			"Si la búsqueda encaja con más de una entrada, no adivina: las enseña y\n" +
			"pide que se afine. Cambiar la contraseña equivocada es peor que no\n" +
			"cambiar ninguna.\n\n" +
			"Queda apuntado en el registro de la bóveda. Y con una ventana de Esfinge\n" +
			"abierta en este mismo equipo, su lista se queda con lo de antes hasta que\n" +
			"algo la refresque: lo guardado sube igual, pero la pantalla no se entera.",
		Example: "  printf '%s' \"$NUEVO\" | esfinge boveda editar --compartida Tempero \\\n" +
			"      \"Cloudflare API\" --secreto-stdin\n" +
			"  esfinge boveda editar Resend --generar --caracteres 32",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// **El secreto se lee antes de abrir nada**, por lo mismo que
			// `ExportarLlaves` mira si hay llaves antes de pedir un sitio donde
			// guardarlas: lo que hace falta para decidir se mira antes de pedirle a
			// nadie que decida. Con `--secreto-stdin` y una tubería vacía, enterarse al
			// final es haber abierto la bóveda para nada.
			var secreto string
			var comoEs *comoEsLoGenerado
			if s.dicho() {
				var err error
				secreto, comoEs, err = s.leer()
				if err != nil {
					return err
				}
			}
			cambiados := loQueSeNombro(cmd, s.dicho())
			if len(cambiados) == 0 {
				return errUso{errors.New(
					"No has dicho qué cambiar; mira «esfinge boveda editar --ayuda»")}
			}

			d, err := abrirParaEscribir(o, ob)
			if err != nil {
				return err
			}
			defer d.cerrar()
			if porque := d.porQueNoSeEscribe(); porque != "" {
				return errors.New(porque)
			}

			e, err := unaSola(d.b, args[0])
			if err != nil {
				return err
			}

			// **El parche, campo a campo sobre lo que ya hay.**
			for _, campo := range cambiados {
				switch campo {
				case "titulo":
					e.Titulo = c.titulo
				case "usuario":
					e.Usuario = c.usuario
				case "notas":
					e.Notas = c.notas
				case "carpeta":
					e.Carpeta = c.carpeta
				case "sitio":
					e.Sitios = c.sitios
				case "etiqueta":
					e.Etiquetas = c.etiquetas
				case "secreto":
					// Por el camino de siempre, para que el anterior caiga al historial.
					e.CambiarSecreto(secreto, time.Now())
				}
			}
			if err := d.b.Poner(e); err != nil {
				return err
			}

			d.apuntar(o, boveda.Apunte{
				Quien: quienDice(quien), Que: "editar", Sobre: e.ID, Titulo: e.Titulo,
			})
			d.decir(o, loQuePaso{
				que: "editar", titulo: e.Titulo, id: e.ID,
				campos: cambiados, generada: comoEs,
			})
			return d.subir(o)
		},
	}
	banderasDeLosCampos(cmd, &c)
	banderasDelSecreto(cmd, &s)
	banderasDeQuien(cmd, &quien)
	return cmd
}

// loQueSeNombro son los campos que alguien ha dicho en esta orden, **mirando si la
// bandera se usó y no si trae valor**.
//
// Esa diferencia es la mitad de `editar`: `--notas ""` es «bórrame las notas» y no
// decir `--notas` es «déjalas». Con el valor vacío las dos cosas se ven igual.
func loQueSeNombro(cmd *cobra.Command, hayUnSecreto bool) []string {
	var out []string
	for _, n := range []string{"titulo", "usuario", "notas", "carpeta", "sitio", "etiqueta"} {
		if cmd.Flags().Changed(n) {
			out = append(out, n)
		}
	}
	if hayUnSecreto {
		out = append(out, "secreto")
	}
	return out
}

func banderasDelSecreto(cmd *cobra.Command, s *origenDelSecreto) {
	f := cmd.Flags()
	f.BoolVar(&s.stdin, "secreto-stdin", false, "Leer el secreto de la entrada estándar")
	f.StringVar(&s.variable, "secreto-env", "", "Leer el secreto de una variable de entorno")
	f.StringVar(&s.fichero, "secreto-fichero", "", "Leer el secreto de un fichero")
	f.BoolVar(&s.generar, "generar", false,
		"Que lo fabrique Esfinge y no lo vea nadie, ni quien lo pide")
	f.StringVar(&s.alfabeto, "alfabeto", cripto.AlfHex.Nombre, "hex, alnum o simbolos")
	f.IntVar(&s.caracteres, "caracteres", 0,
		"Cuántos caracteres con --generar (por defecto, 48 en hexadecimal)")
}

// elTipoQueSePide comprueba el `--tipo`, **contra la lista y no contra nada**.
//
// Sin esto, un tipo mal escrito crea una entrada que la ventana no sabe dibujar y que
// en pantalla sale sin formulario: un fallo mudo que aparece en otro programa.
func elTipoQueSePide(dicho string) (boveda.Tipo, error) {
	if strings.TrimSpace(dicho) == "" {
		return boveda.TipoCredencial, nil
	}
	tipo := boveda.Tipo(strings.ToLower(strings.TrimSpace(dicho)))
	// **Una llave de acceso no se crea a mano** (ADR 0048): la emite el sitio, y lo que
	// se guardara aquí no abriría ninguna cuenta. Se dice aparte de «no existe» porque
	// el tipo sí existe y la razón es otra.
	if tipo == boveda.TipoLlave {
		return "", errUso{errors.New("Una llave de acceso la emite el sitio, no se escribe a mano")}
	}
	for _, t := range []boveda.Tipo{
		boveda.TipoCredencial, boveda.TipoNota, boveda.TipoTarjeta,
		boveda.TipoIdentidad, boveda.TipoPersonal, boveda.TipoWifi,
	} {
		if tipo == t {
			return tipo, nil
		}
	}
	return "", errUso{fmt.Errorf(
		"El tipo «%s» no existe: credencial, nota, tarjeta, identidad, personal o wifi", dicho)}
}

// banderasDeQuien declara `--quien`, que **no autentica nada** y se dice así.
func banderasDeQuien(cmd *cobra.Command, quien *string) {
	cmd.Flags().StringVar(quien, "quien", "",
		"Quién lo pide, para el registro; no se comprueba, es lo que diga quien lo escribe")
}

// quienDice es lo que va al registro. Sin `--quien`, el nombre del programa: es lo
// único cierto que se sabe.
//
// **No se comprueba**, exactamente igual que el nombre que un cliente MCP dice de sí
// mismo en el Mac. Sirve para leer el registro después, no para dar permisos.
func quienDice(quien string) string {
	if q := strings.TrimSpace(quien); q != "" {
		return q
	}
	return "esfinge (línea de comandos)"
}

// ------------------------------------------------- dónde se escribe, y cómo sube

// dondeEscribir es la bóveda que se va a escribir y lo que hace falta para subir lo
// escrito.
//
// `mia` es la personal, y solo está cuando lo que se escribe es una bóveda compartida:
// **la sesión de la cuenta vive sellada dentro de la personal**, así que subir exige
// tener las dos abiertas a la vez. Para leer no hacía falta y se cerraba en el acto.
type dondeEscribir struct {
	b    *boveda.Boveda
	mia  *boveda.Boveda
	comp boveda.Compartida
}

func abrirParaEscribir(o *opciones, ob *opcionesBoveda) (*dondeEscribir, error) {
	if ob.compartida != "" {
		if ob.ruta != "" {
			return nil, errUso{errors.New("--boveda y --compartida dicen dos bóvedas distintas; elige una")}
		}
		b, mia, comp, err := abrirLaCompartidaConLaPersonal(o, ob, ob.compartida)
		if err != nil {
			return nil, err
		}
		return &dondeEscribir{b: b, mia: mia, comp: comp}, nil
	}
	b, err := abrirLaBoveda(o, ob)
	if err != nil {
		return nil, err
	}
	ponerseAlDia(o, ob, b)
	return &dondeEscribir{b: b}, nil
}

func (d *dondeEscribir) cerrar() {
	if d.b != nil {
		d.b.Cerrar()
	}
	if d.mia != nil {
		d.mia.Cerrar()
	}
}

// porQueNoSeEscribe pregunta **la regla de siempre**, que vive en `internal/boveda`
// desde la ADR 0058 porque la comparten la ventana, el navegador, los agentes y esto.
//
// El dueño va vacío cuando la bóveda es propia, que es lo que `PorQueNoSeEscribe`
// espera.
func (d *dondeEscribir) porQueNoSeEscribe() string {
	return boveda.PorQueNoSeEscribe(d.b, d.comp.Dueno, d.comp.Permiso)
}

// apuntar deja el rastro, **y dice cuando no ha podido**.
//
// Una medida de seguridad que no se aplica en silencio es peor que no tenerla, y aquí
// pasa de verdad: en una bóveda compartida de solo ver no se puede escribir el
// registro — aunque ahí tampoco se llega, porque escribir se ha rechazado antes.
func (d *dondeEscribir) apuntar(o *opciones, a boveda.Apunte) {
	a.Resultado = boveda.ApunteHecho
	a.Como = boveda.ApunteLlaveDelEquipo
	if err := d.b.Apuntar(a); err != nil {
		avisar(o, "Esto no ha quedado apuntado en el registro de la bóveda.")
	}
}

// subir es la pasada de después de escribir.
//
// `ponerseAlDia` sincroniza **al abrir**, así que escribir y salir dejaría el cambio
// sin subir hasta la orden siguiente — y en un servidor la siguiente puede no llegar
// nunca.
//
// **Si falla, se dice y no se calla**: lo escrito está guardado aquí y subirá en la
// próxima pasada, así que no es un error de la orden; lo que sí sería un fallo es que
// nadie se enterara.
func (d *dondeEscribir) subir(o *opciones) error {
	if d.mia != nil {
		if err := unaPasadaDeLaCompartida(d.mia, d.b, d.comp); err != nil {
			avisar(o, "Guardado aquí, pero no he podido subirlo: "+err.Error())
		}
		return nil
	}
	if sinCuentaDetras(d.b) {
		return nil
	}
	if _, err := unaPasada(d.b); err != nil {
		avisar(o, "Guardado aquí, pero no he podido subirlo: "+err.Error())
	}
	return nil
}

// sinCuentaDetras dice si esta bóveda no tiene cuenta a la que subir, que es el caso
// de un equipo que trabaja en local: entonces no hay pasada ni hay nada que avisar.
// Es la misma puerta que mira `ponerseAlDia` antes de salir a la red.
func sinCuentaDetras(b *boveda.Boveda) bool {
	return leerDatosDelServidor(b.Ruta()).Acceso == ""
}

// loQuePaso es lo que se cuenta al terminar. **Nunca el secreto**: de una contraseña
// generada se dice cómo es —largo y alfabeto— y de una dada, nada.
type loQuePaso struct {
	que, titulo, id string
	campos          []string
	generada        *comoEsLoGenerado
}

func (d *dondeEscribir) decir(o *opciones, p loQuePaso) {
	if o.silencio {
		return
	}
	e, err := estilos(o.tema)
	if err != nil {
		return
	}
	frase := "Guardada «" + p.titulo + "»."
	if p.que == "editar" {
		frase = "Cambiada «" + p.titulo + "»: " + strings.Join(p.campos, ", ") + "."
	}
	if p.generada != nil {
		frase += fmt.Sprintf(" La contraseña la ha hecho Esfinge: %d caracteres, %s, y no la ha visto nadie.",
			p.generada.largo, p.generada.alfabeto)
	}
	fmt.Fprintln(os.Stderr, e.Ok(frase))
}

// avisar dice algo por la salida de error, que es donde una persona lo sigue viendo
// aunque la salida esté redirigida.
func avisar(o *opciones, frase string) {
	if o.silencio {
		return
	}
	if e, err := estilos(o.tema); err == nil {
		fmt.Fprintln(os.Stderr, e.Ojo(frase))
	}
}
