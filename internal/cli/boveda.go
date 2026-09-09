package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
)

// La bóveda desde la línea de comandos: **solo leer**.
//
// Editar desde aquí puede esperar; consultar, no. Lo que un script necesita es
// sacar una contraseña para metérsela a otra cosa, y para eso hacen falta dos
// piezas que ya existen —la clave nunca por argumento, y una salida limpia por
// la estándar— más una que no existía: **salida estructurada**.
//
// El `--json` no es un adorno. Es la costura de la que va a colgar el
// autorrelleno del navegador cuando llegue: la extensión hablará con esto, no
// con la ventana.

type opcionesBoveda struct {
	ruta  string
	json  bool
	clave OrigenClave
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
			"Vale tanto la contraseña maestra como la clave de recuperación.",
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

	cmd.AddCommand(
		comandoBovedaListar(o, &ob),
		comandoBovedaVer(o, &ob),
		comandoBovedaExportar(o, &ob),
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
			candidatas := b.Buscar(args[0])
			switch len(candidatas) {
			case 0:
				return fmt.Errorf("No hay nada en la bóveda que encaje con «%s»", args[0])
			case 1:
			default:
				var nombres []string
				for _, e := range candidatas {
					nombres = append(nombres, e.Titulo)
				}
				return fmt.Errorf("«%s» encaja con %d entradas (%s); afina la búsqueda",
					args[0], len(candidatas), strings.Join(nombres, ", "))
			}

			entera, _ := b.Ver(candidatas[0].ID)
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
func abrirBoveda(o *opciones, ob *opcionesBoveda) (*boveda.Boveda, error) {
	e, err := estilos(o.tema)
	if err != nil {
		return nil, err
	}
	ruta := ob.ruta
	if ruta == "" {
		ruta = rutaDeLaBoveda()
	}
	if ruta == "" {
		return nil, fmt.Errorf("No sé dónde está la bóveda en este sistema; dila con --boveda")
	}
	if _, err := os.Stat(ruta); err != nil {
		return nil, fmt.Errorf("No hay ninguna bóveda en %s; créala desde la aplicación", ruta)
	}

	// Sin confirmación: aquí se abre algo que ya existe, no se crea nada.
	clave, err := ob.clave.Leer(e, false)
	if err != nil {
		return nil, err
	}
	defer cripto.Borrar(clave)

	return boveda.Abrir(ruta, string(clave))
}

// rutaDeLaBoveda es la misma que usa la ventana. Se repite aquí y no se importa
// de internal/app a propósito: la línea de comandos no depende del paquete de la
// ventana, y meterle esa dependencia por una ruta sería pagar caro.
func rutaDeLaBoveda() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "boveda.esfinge")
}

// escribirJSON saca los datos con sangría, que es lo que espera quien va a
// mirarlos, y sigue siendo válido para quien los vaya a procesar.
func escribirJSON(v any) error {
	c := json.NewEncoder(os.Stdout)
	c.SetIndent("", "  ")
	return c.Encode(v)
}
