package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/codigos"
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/qr"
	"github.com/webcafeina/esfinge/internal/rutas"
	"github.com/webcafeina/esfinge/internal/wifi"
)

// La bóveda desde la línea de comandos: **solo leer**, con una excepción que
// está en `servidor.go`.
//
// Editar desde aquí puede esperar; consultar, no. Lo que un script necesita es
// sacar una contraseña para metérsela a otra cosa, y para eso hacen falta dos
// piezas que ya existen —la clave nunca por argumento, y una salida limpia por
// la estándar— más una que no existía: **salida estructurada**.
//
// El `--json` no es un adorno. Es la costura de la que va a colgar el
// autorrelleno del navegador cuando llegue: la extensión hablará con esto, no
// con la ventana.
//
// Y la excepción: `servidor activar` **escribe**, porque poner una ranura es lo
// único que puede dejar esta bóveda abriéndose sin contraseña en un servidor, y
// poner una ranura exige la bóveda abierta —o sea saber la maestra— (ADR 0057).

type opcionesBoveda struct {
	ruta  string
	json  bool
	clave OrigenClave
	// sinSincronizar se salta la pasada que se hace al abrir en un servidor. Lo pide
	// quien tiene prisa y sabe lo que hace: lo que se lee puede estar viejo.
	sinSincronizar bool
	// compartida es el nombre de una bóveda que le han compartido a este equipo
	// (ADR 0057). **Por su nombre y no por su ruta**, porque el fichero se llama por
	// una referencia al azar y lo que alguien escribe en un guion es «Tempero».
	compartida string
}

func comandoBoveda(o *opciones) *cobra.Command {
	var ob opcionesBoveda

	cmd := &cobra.Command{
		Use:   "boveda",
		Short: "Consulta la bóveda de contraseñas",
		Long: "Lee la bóveda que gestiona la aplicación con ventana.\n\n" +
			"Solo lee: añadir y cambiar entradas se hace desde la ventana. Lo que\n" +
			"hay aquí es lo que necesita un script —sacar una contraseña para dársela\n" +
			"a otra cosa— y la forma de salir si algún día la ventana no arranca.\n\n" +
			"La clave se pide por el terminal, o con --clave-env o --clave-fichero.\n" +
			"Vale tanto la contraseña maestra como la clave de recuperación.\n\n" +
			"En un servidor puede no pedir nada, y se trabaja con una bóveda que le\n" +
			"hayan compartido diciendo su nombre con --compartida. Ver «servidor».",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}

	f := cmd.PersistentFlags()
	f.StringVar(&ob.ruta, "boveda", "", "Ruta de la bóveda (por defecto, la de la aplicación)")
	f.StringVar(&ob.clave.Variable, "clave-env", "", "Leer la clave de una variable de entorno")
	f.StringVar(&ob.clave.Fichero, "clave-fichero", "", "Leer la clave de un fichero")
	f.BoolVar(&ob.json, "json", false, "Sacar los datos en JSON, para tuberías")
	f.BoolVar(&ob.sinSincronizar, "sin-sincronizar", false,
		"En un servidor, no ponerse al día con la cuenta antes de leer")
	f.StringVar(&ob.compartida, "compartida", "",
		"Trabajar con una bóveda que le hayan compartido a este equipo, por su nombre")

	cmd.AddCommand(
		comandoBovedaListar(o, &ob),
		comandoBovedaVer(o, &ob),
		comandoBovedaCodigo(o, &ob),
		comandoBovedaWifi(o, &ob),
		comandoBovedaExportar(o, &ob),
		comandoBovedaEjecutar(o, &ob),
		comandoBovedaServidor(o, &ob),
	)
	return cmd
}

func comandoBovedaListar(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "listar [búsqueda]",
		Short: "Enseña lo que hay dentro, sin las contraseñas",
		Long: "Lista las entradas que encajen con la búsqueda.\n\n" +
			"**No saca ninguna contraseña**: para eso está «ver», que saca una sola y\n" +
			"hay que pedirla por su nombre.",
		Example:       "  esfinge boveda listar\n  esfinge boveda listar banco --json",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			b, err := abrirBoveda(o, ob)
			if err != nil {
				return err
			}
			q := ""
			if len(args) > 0 {
				q = args[0]
			}
			entradas := b.Buscar(q)

			if ob.json {
				return escribirJSON(entradas)
			}
			for _, e := range entradas {
				sitio := ""
				if len(e.Sitios) > 0 {
					sitio = "  " + e.Sitios[0]
				}
				fmt.Printf("%s\t%s%s\n", e.ID[:8], e.Titulo, sitio)
			}
			return nil
		},
	}
}

func comandoBovedaVer(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "ver <búsqueda>",
		Short: "Saca la contraseña de una entrada",
		Long: "Escribe la contraseña por la salida estándar, sin nada más, para\n" +
			"poder metérsela a otra cosa por una tubería.\n\n" +
			"Si la búsqueda encaja con más de una entrada, no adivina: las enseña y\n" +
			"pide que se afine. Sacar la contraseña equivocada es peor que no sacar\n" +
			"ninguna.",
		Example:       "  esfinge boveda ver banco\n  esfinge boveda ver banco --json",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			b, err := abrirBoveda(o, ob)
			if err != nil {
				return err
			}
			entera, err := unaSola(b, args[0])
			if err != nil {
				return err
			}
			if ob.json {
				return escribirJSON(entera)
			}
			// Sin salto de línea decorativo ni marca: esto va a una tubería.
			fmt.Print(entera.Secreto)
			if aTerminal(os.Stdout) {
				fmt.Println()
			}
			return nil
		},
	}
}

// comandoBovedaCodigo saca el código de un solo uso, que es lo que ata a un
// gestor de contraseñas cuando todo lo demás ya se puede llevar uno.
//
// Aquí gana el mismo sueldo que en la ventana y uno más: en un script, el
// segundo factor era lo único que obligaba a coger el teléfono.
func comandoBovedaCodigo(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "codigo <búsqueda>",
		Short: "Calcula el código de un solo uso de una entrada",
		Long: "Escribe las seis cifras por la salida estándar, sin nada más.\n\n" +
			"Se calculan aquí a partir de la semilla que guarda la bóveda: no se\n" +
			"pregunta a nadie ni hace falta red. Si la máquina lleva el reloj\n" +
			"desviado más de medio minuto, el código no valdrá y eso no hay forma\n" +
			"de detectarlo desde este lado.",
		Example:       "  esfinge boveda codigo banco",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			b, err := abrirBoveda(o, ob)
			if err != nil {
				return err
			}
			entera, err := unaSola(b, args[0])
			if err != nil {
				return err
			}
			if entera.TOTP == "" {
				return fmt.Errorf("«%s» no tiene código de un solo uso guardado", entera.Titulo)
			}
			s, err := codigos.Leer(entera.TOTP)
			if err != nil {
				return err
			}
			ahora := time.Now()
			codigo, err := s.En(ahora)
			if err != nil {
				return err
			}

			if ob.json {
				return escribirJSON(map[string]any{
					"codigo":  codigo,
					"quedan":  int(s.Quedan(ahora).Seconds()),
					"periodo": int(s.Periodo.Seconds()),
				})
			}
			// Sin salto ni adornos: esto va a una tubería.
			fmt.Print(codigo)
			if aTerminal(os.Stdout) {
				fmt.Println()
				// Y cuando hay alguien mirando, cuánto le queda de vida. Un código
				// con dos segundos por delante y sin avisar es un viaje en balde a
				// la pantalla del servicio.
				e, err := estilos(o.tema)
				if err == nil && !o.silencio {
					fmt.Fprintln(os.Stderr, e.Info(fmt.Sprintf("Vale %d segundos más.",
						int(s.Quedan(ahora).Seconds()))))
				}
			}
			return nil
		},
	}
}

// comandoBovedaWifi dibuja el código QR de una red en la terminal (ADR 0049).
//
// **Es el primer comando de la bóveda que enseña un secreto en la pantalla.** `ver`
// saca la contraseña por la salida estándar para que se la lleve una tubería, y eso es
// distinto: ahí el secreto va a donde se le mande. Un código QR está para mirarlo, así
// que se dice en la ayuda y se dice al dibujarlo.
//
// Se dibuja con el color puesto a mano y no con el del tema: una terminal clara con el
// código invertido no la lee ninguna cámara, y aquí no hay forma de saber de qué color
// es el fondo de quien mira.
func comandoBovedaWifi(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "wifi <búsqueda>",
		Short: "Dibuja el código QR de una red para conectar un móvil",
		Long: "Dibuja en la terminal el código de una red wifi guardada. Apuntando con\n" +
			"la cámara de un móvil, se conecta sin que nadie teclee la contraseña.\n\n" +
			"El dibujo es la contraseña: quien lo fotografíe entra en la red. Con\n" +
			"--json salen las filas en ceros y unos, para dibujarlo en otro sitio.",
		Example:       "  esfinge boveda wifi oficina",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			b, err := abrirBoveda(o, ob)
			if err != nil {
				return err
			}
			entera, err := unaSola(b, args[0])
			if err != nil {
				return err
			}
			if entera.Tipo != boveda.TipoWifi {
				return fmt.Errorf("«%s» no es una red wifi", entera.Titulo)
			}
			enlace, err := wifi.Enlace(entera.SSID, entera.Secreto, entera.Seguridad, entera.Oculta)
			if err != nil {
				return err
			}
			c, err := qr.Nuevo(enlace)
			if err != nil {
				return err
			}

			if ob.json {
				return escribirJSON(map[string]any{
					"red":   entera.SSID,
					"lado":  c.Lado(),
					"filas": c.Filas(),
				})
			}

			// La zona tranquila, que no es parte del código y sin ella muchos lectores
			// no lo encuentran. Y dos caracteres por módulo, o sale un rectángulo:
			// una celda de terminal es el doble de alta que de ancha.
			const margen = 4
			var b2 strings.Builder
			for y := -margen; y < c.Lado()+margen; y++ {
				for x := -margen; x < c.Lado()+margen; x++ {
					if c.Oscuro(x, y) {
						b2.WriteString("\033[40m  ")
					} else {
						b2.WriteString("\033[107m  ")
					}
				}
				b2.WriteString("\033[0m\n")
			}
			fmt.Print(b2.String())
			if aTerminal(os.Stdout) && !o.silencio {
				e, err := estilos(o.tema)
				if err == nil {
					fmt.Fprintln(os.Stderr, e.Info(fmt.Sprintf(
						"Red «%s». Este dibujo es la contraseña: quien lo fotografíe entra.", entera.SSID)))
				}
			}
			return nil
		},
	}
}

// unaSola encuentra la entrada que se pide, o dice por qué no.
//
// **No adivina cuando hay varias**, y es deliberado: sacar la contraseña
// equivocada por una tubería es peor que no sacar ninguna, porque el fallo
// aparece al otro lado y sin nada que lo explique.
func unaSola(b *boveda.Boveda, q string) (boveda.Entrada, error) {
	candidatas := b.Buscar(q)
	switch len(candidatas) {
	case 0:
		return boveda.Entrada{}, fmt.Errorf("No hay nada en la bóveda que encaje con «%s»", q)
	case 1:
		entera, _ := b.Ver(candidatas[0].ID)
		return entera, nil
	default:
		var nombres []string
		for _, e := range candidatas {
			nombres = append(nombres, e.Titulo)
		}
		return boveda.Entrada{}, fmt.Errorf("«%s» encaja con %d entradas (%s); afina la búsqueda",
			q, len(candidatas), strings.Join(nombres, ", "))
	}
}

func comandoBovedaExportar(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "exportar",
		Short: "Saca la bóveda entera en CSV, sin cifrar",
		Long: "Escribe todas las entradas en CSV **sin cifrar**, por la salida\n" +
			"estándar o al fichero que diga -o.\n\n" +
			"Existe porque una bóveda de la que no se puede salir es una trampa. Lo\n" +
			"que sale de aquí es una lista de contraseñas en claro: piensa dónde la\n" +
			"dejas y bórrala cuando termines.",
		Example:       "  esfinge boveda exportar -o copia.csv",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			b, err := abrirBoveda(o, ob)
			if err != nil {
				return err
			}
			e, err := estilos(o.tema)
			if err != nil {
				return err
			}
			if !o.silencio {
				fmt.Fprintln(os.Stderr, e.Mal("Esto sale SIN CIFRAR: es una lista de contraseñas en claro."))
			}
			return conSalida(e, o, o.salida, b.Exportar)
		},
	}
}

// abrirBoveda pide la clave y desbloquea. Vale tanto la contraseña maestra como
// la de recuperación: la bóveda prueba las dos ranuras y no hace falta decirlo.
//
// **Y en un servidor puede no pedir nada** (ADR 0057): si esta bóveda lleva la
// ranura del servidor y la llave está en este equipo, abre con ella. Lo decide el
// fichero de la bóveda, así que donde nadie lo haya activado esto no cambia nada.
//
// Y **en un servidor se pone al día antes de dejar leer nada**, que es lo que hace
// que una credencial cambiada en el Mac llegue aquí.
func abrirBoveda(o *opciones, ob *opcionesBoveda) (*boveda.Boveda, error) {
	if ob.compartida != "" {
		if ob.ruta != "" {
			return nil, errUso{errors.New("--boveda y --compartida dicen dos bóvedas distintas; elige una")}
		}
		// Ésa se pone al día por su cuenta, contra la cuenta de su dueño.
		return abrirLaCompartidaPorNombre(o, ob, ob.compartida)
	}
	b, err := abrirLaBoveda(o, ob)
	if err != nil {
		return nil, err
	}
	ponerseAlDia(o, ob, b)
	return b, nil
}

// abrirLaBoveda abre y nada más: lo usan los comandos que no quieren una pasada de
// sincronización por delante —los de montar el servidor, y el que la pide él mismo—.
func abrirLaBoveda(o *opciones, ob *opcionesBoveda) (*boveda.Boveda, error) {
	e, err := estilos(o.tema)
	if err != nil {
		return nil, err
	}
	ruta, err := rutaElegida(ob)
	if err != nil {
		return nil, err
	}

	if b, err := abrirConLaLlaveDelServidor(e, o.silencio, ob, ruta); b != nil || err != nil {
		return b, err
	}

	// Sin confirmación: aquí se abre algo que ya existe, no se crea nada.
	clave, err := ob.clave.Leer(e, false)
	if err != nil {
		return nil, err
	}
	defer cripto.Borrar(clave)

	return boveda.Abrir(ruta, string(clave))
}

// rutaElegida es con qué bóveda se va a trabajar, y comprueba que está ahí.
//
// Va aparte de abrirBoveda porque hay cosas que se saben **sin abrir nada** y que
// hay que mirar antes de pedirle a nadie la contraseña: si esta bóveda es la de un
// proyecto, o si se abre sola en este equipo.
func rutaElegida(ob *opcionesBoveda) (string, error) {
	ruta := ob.ruta
	if ruta == "" {
		ruta = rutaDeLaBoveda()
	}
	if ruta == "" {
		return "", fmt.Errorf("No sé dónde está la bóveda en este sistema; dila con --boveda")
	}
	if _, err := os.Stat(ruta); err != nil {
		return "", fmt.Errorf("No hay ninguna bóveda en %s; créala desde la aplicación", ruta)
	}
	return ruta, nil
}

// rutaDeLaBoveda es la misma que usa la ventana, **y desde la ADR 0057 sale del
// mismo sitio**: `internal/rutas`.
//
// Antes se repetía aquí a propósito, para no depender del paquete de la ventana
// —que arrastra Wails y no cruza de plataforma—. Esa razón sigue en pie; lo que
// cambió es que ahora la línea de comandos también busca bóvedas de proyecto y
// compartidas, y tres rutas copiadas se desalinean solas: el fallo sería dos sitios
// donde buscar la misma bóveda, cada programa sobre su copia.
func rutaDeLaBoveda() string { return rutas.BovedaPrincipal() }

// escribirJSON saca los datos con sangría, que es lo que espera quien va a
// mirarlos, y sigue siendo válido para quien los vaya a procesar.
func escribirJSON(v any) error {
	c := json.NewEncoder(os.Stdout)
	c.SetIndent("", "  ")
	return c.Encode(v)
}
