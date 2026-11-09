package cli

// El modo servidor: abrir la bóveda **donde no hay nadie que teclee la
// contraseña maestra** (ADR 0057).
//
// Lo que esto resuelve es lo que hoy son ficheros `.env` en claro en el VPS, que
// cualquiera que entre lee y que ningún agente puede usar sin metérselos en su
// conversación. Lo que cuesta está dicho en `activar`, en la pantalla, y no solo
// en la documentación: **quien entre en el servidor abre la bóveda**.
//
// Tres cosas de aquí que no son detalles:
//
//   - **Esto es lo único de la bóveda que escribe desde la línea de comandos.** El
//     resto solo lee, y la cabecera de `boveda.go` lo dice. No es una excepción
//     caprichosa: activar exige poner una ranura, y poner una ranura exige la
//     bóveda abierta, que es lo que impide hacerlo sin saber la maestra.
//   - **La llave se guarda por el identificador de la bóveda**, no por una
//     constante como en la ventana (`idEnElLlavero`). Allí es una por máquina para
//     que tras cada actualización salga **un** diálogo del llavero y no uno por
//     bóveda; aquí no hay diálogo ninguno, así que no se paga nada por que cada
//     bóveda tenga la suya — y se gana que la llave de una no se lea como la de
//     otra.
//   - **La ranura va solo en la bóveda personal**, nunca en una de proyecto: un
//     proyecto se abre con la clave de bóveda de la personal (ADR 0050), así que
//     ésta es la única llave en disco y lo demás cuelga de ella. Se comprueba
//     **antes** de pedir la maestra, que es la regla de `ExportarLlaves`: lo que
//     hace falta para decidir se mira antes de pedirle a alguien que decida.

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/llavero"
	"github.com/webcafeina/esfinge/internal/rutas"
	"github.com/webcafeina/esfinge/internal/salida"
)

// elAvisoDelServidor es lo que se lee antes de activar, y **se dice con las
// palabras de Touch ID**: lo que no protege, primero.
const elAvisoDelServidor = "La llave queda en un fichero de este equipo, así que quien entre en él " +
	"abre esta bóveda. No protege de eso: protege de que las contraseñas estén en " +
	"claro en un .env, y deja rastro de lo que se usa."

func comandoBovedaServidor(o *opciones, ob *opcionesBoveda) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "servidor",
		Short: "Abrir esta bóveda en un servidor, sin contraseña",
		Long: "Deja que esta bóveda se abra en este equipo sin teclear la contraseña\n" +
			"maestra, con una llave que queda en un fichero.\n\n" +
			"Es para un servidor, donde no hay nadie delante: lo que hace es que los\n" +
			"secretos de un proyecto dejen de vivir en ficheros .env en claro.\n\n" +
			"No tiene nada que ver con el servidor de cuentas de Esfinge, que es lo\n" +
			"que sincroniza las bóvedas entre equipos.",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		comandoServidorAlta(o, ob),
		comandoServidorEntrar(o, ob),
		comandoServidorActivar(o, ob),
		comandoServidorQuitar(o, ob),
		comandoServidorRecibir(o, ob),
		comandoServidorSincronizar(o, ob),
		comandoServidorEstado(o, ob),
		comandoServidorHuella(o, ob),
	)
	return cmd
}

func comandoServidorActivar(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "activar",
		Short: "Pone la llave de este servidor en la bóveda",
		Long: "Fabrica una llave al azar, la guarda en un fichero de este equipo y\n" +
			"añade a la bóveda la ranura que esa llave abre.\n\n" +
			"Pide la contraseña maestra: es lo que impide activarlo sin saberla.\n\n" +
			"**" + elAvisoDelServidor + "**\n\n" +
			"La contraseña maestra sigue abriendo, y la clave de recuperación\n" +
			"también. Volver a activarlo cambia la llave, y la de antes deja de\n" +
			"servir: es como se rota el día que haya que rotarla.\n\n" +
			"La llave no se sube a ninguna parte: es de este equipo y de nadie más.",
		Example:       "  esfinge boveda servidor activar",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			e, err := estilos(o.tema)
			if err != nil {
				return err
			}
			ruta, err := rutaElegida(ob)
			if err != nil {
				return err
			}
			// Antes de pedir la maestra: lo que haría que esto no se pueda hacer.
			if boveda.RanuraPrincipalEn(ruta) {
				return fmt.Errorf("Ésa es la bóveda de un proyecto, y la llave del servidor va en la personal; " +
					"con la personal activada, los proyectos se abren a través de ella")
			}
			f := llavero.DelServidor()
			if f.Ruta == "" {
				return fmt.Errorf("No sé dónde guardar la llave en este sistema; dilo con %s", llavero.VariableDeLaLlave)
			}
			// Y se dice lo que cuesta **antes** de pedir la contraseña, no después.
			// Esto no se calla con --silencio, al contrario que los mensajes de
			// cortesía de abajo: callar lo que no protege sería venderlo.
			fmt.Fprintln(os.Stderr, e.Ojo(elAvisoDelServidor))

			b, err := abrirLaBoveda(o, ob)
			if err != nil {
				return err
			}
			defer b.Cerrar()

			llave, err := boveda.SecretoDelServidor()
			if err != nil {
				return err
			}
			// **Primero el fichero y después la ranura.** Al revés, un fallo al
			// escribir dejaría una ranura que no abre nadie.
			if err := f.Guardar(b.ID(), llave); err != nil {
				return err
			}
			if err := b.PonerRanuraDelServidor(llave); err != nil {
				// Se deshace lo de fuera: una llave que no abre nada no hace daño,
				// pero tampoco tiene por qué quedarse en el disco.
				_ = f.Borrar(b.ID())
				return err
			}

			if !o.silencio {
				fmt.Fprintln(os.Stderr, e.Ok("Esta bóveda ya se abre en este equipo sin contraseña."))
				fmt.Fprintln(os.Stderr, e.Info("La llave está en "+f.Ruta+", con permisos 600. No la copies a ninguna parte."))
			}
			return nil
		},
	}
}

func comandoServidorQuitar(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "quitar",
		Short: "Deja de abrir esta bóveda sin contraseña",
		Long: "Quita la ranura de la bóveda y borra la llave de este equipo. Después\n" +
			"solo abre la contraseña maestra.\n\n" +
			"Se puede hacer con la llave que hay —no hace falta la maestra—, porque\n" +
			"esto solo cierra una puerta y quien está en el servidor ya la tenía\n" +
			"abierta.",
		Example:       "  esfinge boveda servidor quitar",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			e, err := estilos(o.tema)
			if err != nil {
				return err
			}
			b, err := abrirLaBoveda(o, ob)
			if err != nil {
				return err
			}
			defer b.Cerrar()
			if !b.TieneRanuraDelServidor() {
				if !o.silencio {
					fmt.Fprintln(os.Stderr, e.Info("Esta bóveda ya no se abría sin contraseña en este equipo."))
				}
				return nil
			}
			// **La ranura se quita aunque falle borrar la llave**: sin sobre, esa
			// llave ya no abre nada, y lo que importa es que la puerta se cierre.
			if err := b.QuitarRanuraDelServidor(); err != nil {
				return err
			}
			_ = llavero.DelServidor().Borrar(b.ID())
			if !o.silencio {
				fmt.Fprintln(os.Stderr, e.Ok("Quitado. Esta bóveda vuelve a pedir la contraseña maestra."))
			}
			return nil
		},
	}
}

// comandoServidorHuella enseña la huella de identidad de esta bóveda, que es lo que
// alguien tiene que comparar en el otro equipo antes de darle acceso (ADR 0043).
//
// **Existe porque sin él la ceremonia no se puede hacer.** La ventana enseña la huella
// que el servidor de cuentas publica para esa dirección, y compararla exige verla
// también **aquí**, dicha por este equipo y no por el servidor — que es el que se
// quiere descartar. Hasta la 0058 no había forma: la primera vez se sacó con un
// programa de un solo uso, y la pantalla mandaba a compararla con algo que no se podía
// leer. Es la trampa de `ExportarLlaves` otra vez: lo que hace falta para decidir se
// mira antes de pedirle a alguien que decida.
//
// **Una huella es pública** —es un resumen de las llaves públicas—, así que esto no
// enseña ningún secreto y se puede pegar en un chat sin pensarlo. Lo que protege es
// justo eso: que las dos partes digan la misma en voz alta.
func comandoServidorHuella(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "huella",
		Short: "Enseña la huella de identidad de este equipo, para compararla",
		Long: "Escribe la huella de identidad de esta bóveda.\n\n" +
			"Es lo que hay que comparar con lo que enseña la ventana del otro equipo\n" +
			"antes de darle acceso a una bóveda de proyecto: si las dos coinciden, lo\n" +
			"que se comparta se cifra hacia este equipo y no hacia nadie más.\n\n" +
			"No es un secreto: es un resumen de las llaves públicas, y se puede decir\n" +
			"en voz alta o pegar en un chat.",
		Example:       "  esfinge boveda servidor huella",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			// **Sin pasada de sincronización**: la identidad es de este fichero y no
			// cambia nunca, así que salir a la red aquí solo añade formas de fallar.
			b, err := abrirLaBoveda(o, ob)
			if err != nil {
				return err
			}
			defer b.Cerrar()
			i, err := b.Identidad()
			if err != nil {
				return err
			}
			if ob.json {
				return escribirJSON(map[string]any{"huella": i.Huella})
			}
			// Por la salida estándar y sin adornos, que esto se copia y se pega.
			fmt.Print(i.Huella)
			if aTerminal(os.Stdout) {
				fmt.Println()
				if e, err := estilos(o.tema); err == nil && !o.silencio {
					fmt.Fprintln(os.Stderr, e.Info(
						"Compárala con la que enseña la ventana antes de aceptar ningún acceso."))
				}
			}
			return nil
		},
	}
}

func comandoServidorEstado(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "estado",
		Short: "Dice si esta bóveda se abre sin contraseña en este equipo",
		Long: "Enseña si la bóveda lleva la ranura del servidor y si la llave sigue\n" +
			"estando. **No pide la contraseña**: las dos cosas se pueden saber sin\n" +
			"abrir nada.\n\n" +
			"Y si se abre sin contraseña, dice además **qué bóvedas le han\n" +
			"compartido y con qué permiso**, que es lo que hace falta para trabajar\n" +
			"con ellas: el nombre solo se anuncia al aceptarlas con «recibir».",
		Example:       "  esfinge boveda servidor estado --json",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			e, err := estilos(o.tema)
			if err != nil {
				return err
			}
			ruta, err := rutaElegida(ob)
			if err != nil {
				return err
			}
			f := llavero.DelServidor()
			ranura := boveda.RanuraDelServidorEn(ruta)
			id := boveda.IDEn(ruta)
			hayLlave := false
			if id != "" {
				_, err := f.Leer(id, "")
				hayLlave = err == nil
			}
			// Los permisos del fichero se dicen tal cual: nadie los va a mirar por su
			// cuenta, y un 644 ahí es la bóveda abierta para cualquiera que tenga
			// cuenta en la máquina.
			permisos := ""
			if info, err := os.Stat(f.Ruta); err == nil {
				permisos = fmt.Sprintf("%o", info.Mode().Perm())
			}

			// **Las bóvedas compartidas, cuando se pueden saber sin contraseña.**
			//
			// Este comando prometía no abrir nada, y lo sigue cumpliendo donde eso es
			// lo único posible: en un portátil no hay ranura y aquí no se pregunta
			// nada. Pero en un servidor la bóveda **se abre sola**, y entonces negarse
			// a mirar dentro no protege de nada y deja al que trabaja ahí sin saber
			// qué tiene. Hasta hoy la única forma de averiguarlo era un programa de un
			// solo uso.
			compartidas, pudeMirar := lasCompartidasDeAqui(e, ob)

			if ob.json {
				return escribirJSON(map[string]any{
					"boveda":      ruta,
					"ranura":      ranura,
					"llave":       hayLlave,
					"fichero":     f.Ruta,
					"permisos":    permisos,
					"abre":        ranura && hayLlave,
					"version":     o.version,
					"compartidas": compartidas,
				})
			}
			switch {
			case ranura && hayLlave:
				fmt.Fprintln(os.Stderr, e.Ok("Esta bóveda se abre en este equipo sin contraseña."))
				fmt.Fprintln(os.Stderr, e.Info("La llave está en "+f.Ruta+", con permisos "+permisos+"."))
				if permisos != "600" {
					fmt.Fprintln(os.Stderr, e.Ojo("Esos permisos dejan leer la llave a quien tenga cuenta en esta máquina."))
				}
			case ranura:
				fmt.Fprintln(os.Stderr, e.Ojo("La bóveda lleva la ranura del servidor y la llave de este equipo ya no está."))
				fmt.Fprintln(os.Stderr, e.Info("Ábrela con la contraseña maestra y vuelve a activarlo."))
			case hayLlave:
				fmt.Fprintln(os.Stderr, e.Ojo("Hay una llave de este equipo y la bóveda ya no la usa."))
				fmt.Fprintln(os.Stderr, e.Info("No abre nada; se puede borrar con «esfinge boveda servidor quitar»."))
			default:
				fmt.Fprintln(os.Stderr, e.Info("Esta bóveda solo se abre con la contraseña maestra."))
			}
			// **Y solo se dice algo si se ha podido mirar.** No es lo mismo «no tiene
			// ninguna» que «no he podido abrir para verlo»: con `--clave-env` puesto, o
			// sin la llave de este equipo, decir lo primero sería mentir.
			if pudeMirar {
				if len(compartidas) == 0 {
					fmt.Fprintln(os.Stderr, e.Info(
						"Todavía no le han compartido ninguna bóveda; tráelas con «esfinge boveda servidor recibir»."))
				} else {
					fmt.Fprintln(os.Stderr, e.Info("Bóvedas compartidas con este servidor:"))
					for _, c := range compartidas {
						linea := "    " + c.Nombre + " · " + fraseDelPermiso(c.Permiso)
						if !c.Aqui {
							linea += " · todavía no está aquí («servidor recibir»)"
						}
						fmt.Fprintln(os.Stderr, linea)
					}
				}
			}
			return nil
		},
	}
}

// compartidaDeAqui es lo que `estado` dice de cada bóveda compartida. Lleva **menos
// de lo que hay** a propósito: el nombre y el permiso son para trabajar, y de quién
// es o desde cuándo se leen en el JSON, que es donde mira un programa.
type compartidaDeAqui struct {
	Nombre  string `json:"nombre"`
	Permiso string `json:"permiso"`
	// Aqui dice si el fichero está de verdad en este equipo. Una fila sin fichero no
	// es un error: es una bóveda que han compartido y que todavía no se ha aceptado
	// con «recibir», y decirlo es la diferencia entre entenderlo y pelearse con un
	// «no está todavía en este equipo» al ir a usarla.
	Aqui   bool   `json:"aqui"`
	Dueno  string `json:"dueno"`
	Huella string `json:"huella,omitempty"`
	Desde  string `json:"desde,omitempty"`
}

// lasCompartidasDeAqui las lee **sin pedir ninguna contraseña**, y dice además si ha
// podido mirar: quien llama necesita distinguir «no tiene ninguna» de «no he podido
// abrir para verlo», porque decir lo primero cuando pasa lo segundo es mentir.
//
// **Quien se niega es `abrirConLaLlaveDelServidor`**, que devuelve `nil, nil` cuando
// aquí no hay con qué abrir: sin ranura, sin llave, o con una clave dicha a mano, que
// manda. Comprobar aparte si la bóveda «abre sola» sería la misma regla en dos sitios —
// y se vio que no vigilaba nada al mutarla.
//
// Cualquier fallo al abrir se traga: esto es un informe, y un informe que revienta por
// no poder contar una de sus líneas es peor que uno que cuenta las demás.
func lasCompartidasDeAqui(e salida.Estilos, ob *opcionesBoveda) ([]compartidaDeAqui, bool) {
	ruta, err := rutaElegida(ob)
	if err != nil {
		return nil, false
	}
	b, err := abrirConLaLlaveDelServidor(e, true, ob, ruta)
	if err != nil || b == nil {
		return nil, false
	}
	defer b.Cerrar()
	var out []compartidaDeAqui
	for _, c := range lasCompartidasVivas(b) {
		_, err := os.Stat(rutas.Compartida(c.Dueno, c.Ref))
		out = append(out, compartidaDeAqui{
			Nombre: c.Nombre, Permiso: c.Permiso, Aqui: err == nil,
			Dueno: c.Dueno, Huella: c.Huella, Desde: c.Desde,
		})
	}
	return out, true
}

// fraseDelPermiso lo dice en palabras, y **el vacío no se calla**: una bóveda
// aceptada antes de la ADR 0058 no lleva permiso apuntado, y eso no es «no puede»
// sino «todavía no se sabe» — se sabrá en la primera pasada, porque lo dice el
// servidor al bajar.
func fraseDelPermiso(p string) string {
	switch p {
	case "editar":
		return "puede ver y escribir"
	case "ver":
		return "solo puede ver"
	default:
		return "permiso todavía sin saber"
	}
}

// abrirConLaLlaveDelServidor abre sin preguntar nada **cuando aquí hay con qué**, y
// devuelve `nil, nil` cuando no, para que se pida la clave como siempre.
//
// Lo que decide es el fichero de la bóveda, no una preferencia: si no lleva la
// ranura, aquí no hay nada que probar. En un portátil nunca la lleva, así que esto
// no cambia nada de lo de siempre.
func abrirConLaLlaveDelServidor(e salida.Estilos, silencio bool, ob *opcionesBoveda, ruta string) (*boveda.Boveda, error) {
	// Una clave dicha a mano manda: quien pasa --clave-env está diciendo con qué
	// quiere abrir.
	if ob.clave.Variable != "" || ob.clave.Fichero != "" {
		return nil, nil
	}
	if !boveda.RanuraDelServidorEn(ruta) {
		return nil, nil
	}
	id := boveda.IDEn(ruta)
	if id == "" {
		return nil, nil
	}
	llave, err := llavero.DelServidor().Leer(id, "")
	if err != nil {
		// **La ranura está y la llave no.** Pasa si alguien borró el fichero, o si
		// esta bóveda llegó de otro equipo que sí lo tenía activado. No se puede
		// arreglar desde aquí —quitar la ranura exige la bóveda abierta—, así que se
		// dice y se sigue por la contraseña, que es la salida que queda.
		if !silencio {
			fmt.Fprintln(os.Stderr, e.Ojo("Esta bóveda dice abrirse sin contraseña en un equipo y aquí no está esa llave."))
		}
		return nil, nil
	}
	b, err := boveda.AbrirConElServidor(ruta, llave)
	if err != nil {
		// Y aquí **no se vuelve a la contraseña**: la llave está, no abre, y eso es
		// una bóveda restaurada de una copia anterior. Decirlo es lo útil; pedir una
		// contraseña que en un servidor nadie va a teclear, no.
		return nil, fmt.Errorf("La llave de este servidor ya no abre esta bóveda; "+
			"ábrela con la contraseña maestra y vuelve a activarlo (%w)", err)
	}
	return b, nil
}
