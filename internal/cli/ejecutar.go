package cli

// Ejecutar un programa **con el secreto dentro y sin enseñárselo a nadie** (ADR
// 0057).
//
// Es el equivalente del portapapeles de la ADR 0054 en una máquina sin pantalla:
// allí el agente recibe «copiado» y la contraseña va al portapapeles; aquí recibe
// la salida del programa y la contraseña va **al entorno del proceso hijo**. En los
// dos casos lo que se evita es lo mismo y es lo caro: que el secreto pase por la
// conversación de un modelo, donde se queda para siempre.
//
// Cuatro cosas que no son detalles:
//
//   - **Nunca escribe un fichero.** Un `.env` generado por Esfinge es el problema
//     de partida otra vez, con un paso más.
//   - **La bóveda se cierra antes de lanzar el programa.** Lo que hace falta ya se
//     ha sacado, así que mientras el hijo corre —que puede ser media hora— no hay
//     ninguna bóveda abierta ni ninguna clave en memoria.
//   - **Queda apuntado en el registro**, que es lo que hoy no existe en absoluto:
//     con los `.env` en claro nadie puede contestar «¿qué credencial usó ese
//     despliegue?». Se apunta **antes** de lanzar, porque la pregunta que el
//     registro contesta es qué se entregó, y eso se sabe entonces.
//   - Y **esto no es una frontera contra quien tenga terminal en el servidor**: con
//     la llave de este equipo, `esfinge boveda ver` también saca la contraseña. Lo
//     dicen la ADR y `docs/seguridad.md` con todas las letras. Lo que esto hace es
//     que el camino cómodo no se la enseñe y deje rastro.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/boveda"
)

// errDelHijo lleva el código con el que terminó el programa, para devolverlo tal
// cual: dentro de un guion, `esfinge boveda ejecutar -- algo` tiene que fallar
// igual que `algo`, o el guion no se entera.
type errDelHijo struct {
	codigo  int
	comando string
}

func (e errDelHijo) Error() string {
	return fmt.Sprintf("«%s» ha terminado con el código %d", e.comando, e.codigo)
}

func comandoBovedaEjecutar(o *opciones, ob *opcionesBoveda) *cobra.Command {
	var vars []string
	cmd := &cobra.Command{
		Use:   "ejecutar --var VARIABLE=búsqueda -- programa [argumentos...]",
		Short: "Lanza un programa con secretos de la bóveda en su entorno",
		Long: "Saca de la bóveda los secretos que se le digan, los pone en el entorno\n" +
			"del programa y lo lanza. **El secreto no sale por la pantalla ni se\n" +
			"escribe en ningún fichero**: solo lo ve el programa.\n\n" +
			"Para eso está, y es la diferencia con «ver»: un agente que despliega no\n" +
			"necesita leer el testigo, necesita que wrangler lo tenga. Leyéndolo, el\n" +
			"secreto se queda en su conversación para siempre.\n\n" +
			"Cada --var es VARIABLE=búsqueda, y la búsqueda tiene que encajar con una\n" +
			"sola entrada: sacar el secreto equivocado es peor que no sacar ninguno.\n\n" +
			"Queda apuntado en el registro de la bóveda: qué entrada, cuándo y a qué\n" +
			"programa. La bóveda se cierra antes de lanzarlo.",
		Example: "  esfinge boveda ejecutar --var CF_TOKEN=\"Cloudflare API\" -- wrangler deploy\n" +
			"  esfinge boveda ejecutar --var DB_URL=\"Neon Tempero\" --var RESEND=Resend -- ./desplegar.sh\n" +
			"  esfinge boveda ejecutar --compartida Tempero --var CF_TOKEN=Cloudflare -- wrangler deploy",
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			pedidas, err := loQueSePide(vars)
			if err != nil {
				return err
			}
			b, err := abrirBoveda(o, ob)
			if err != nil {
				return err
			}
			avisar := func(frase string) {
				if o.silencio {
					return
				}
				if e, err := estilos(o.tema); err == nil {
					fmt.Fprintln(os.Stderr, e.Ojo(frase))
				}
			}
			return conLosSecretos(b, pedidas, args, avisar, lanzarDeVerdad)
		},
	}
	cmd.Flags().StringArrayVar(&vars, "var", nil,
		"VARIABLE=búsqueda, una vez por cada secreto que necesite el programa")
	return cmd
}

// conLosSecretos es el orden que importa: sacar, apuntar, **cerrar** y lanzar.
//
// `lanzar` entra por aquí y no está escrito dentro a propósito. No es una
// comodidad: lo único que hay que comprobar de esta función es que **cuando se
// lanza el programa la bóveda ya está cerrada**, y eso no se ve desde fuera del
// proceso. Es la misma costura que `Sistema` en la ventana y que `azar` en
// `cripto`.
//
// Cierra la bóveda siempre, salga como salga: una bóveda abierta a la que ya nadie
// va a volver es justo lo que el reloj del bloqueo existe para que no pase, y aquí
// no hay reloj.
func conLosSecretos(b *boveda.Boveda, pedidas [][2]string, args []string,
	avisar func(string), lanzar func(entorno []string, args []string) error) error {
	entorno, apuntes, err := sacarLosSecretos(b, pedidas)
	if err != nil {
		b.Cerrar()
		return err
	}

	// El rastro, antes de lanzar nada y con la bóveda todavía abierta.
	programa := strings.Join(args, " ")
	sinRastro := false
	for _, a := range apuntes {
		a.Quien = programa
		a.Que = "ejecutar"
		a.Resultado = boveda.ApunteHecho
		a.Como = boveda.ApunteLlaveDelEquipo
		if err := b.Apuntar(a); err != nil {
			sinRastro = true
		}
	}
	// **Si no se ha podido apuntar, se dice**: una medida de seguridad que no se
	// aplica en silencio es peor que no tenerla, y lo dice el propio `registro.go`.
	// Pasa con una bóveda compartida de solo ver, que es justamente la clase de
	// bóveda que un servidor va a tener.
	if sinRastro {
		avisar("Esto no ha quedado apuntado en el registro de la bóveda.")
	}

	// **Y se cierra antes de lanzar.** Lo que hacía falta ya está fuera, y el
	// programa puede tardar media hora.
	b.Cerrar()
	return lanzar(entorno, args)
}

func lanzarDeVerdad(entorno []string, args []string) error {
	hijo := exec.Command(args[0], args[1:]...)
	hijo.Env = append(os.Environ(), entorno...)
	hijo.Stdin, hijo.Stdout, hijo.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := hijo.Run(); err != nil {
		var salida *exec.ExitError
		if errors.As(err, &salida) {
			return errDelHijo{codigo: salida.ExitCode(), comando: args[0]}
		}
		return fmt.Errorf("No he podido lanzar «%s»: %w", args[0], err)
	}
	return nil
}

// loQueSePide parte los --var y comprueba los nombres.
//
// **Un nombre de variable mal puesto no da error en el hijo, da una variable que no
// está**, y entonces el programa falla por otra cosa y en otro sitio. Se comprueba
// aquí.
func loQueSePide(vars []string) ([][2]string, error) {
	if len(vars) == 0 {
		return nil, errUso{errors.New(
			"Dime qué secretos necesita, con --var VARIABLE=búsqueda")}
	}
	var out [][2]string
	vistas := map[string]bool{}
	for _, v := range vars {
		nombre, busqueda, hay := strings.Cut(v, "=")
		if !hay {
			return nil, errUso{fmt.Errorf("«%s» no tiene la forma VARIABLE=búsqueda", v)}
		}
		nombre, busqueda = strings.TrimSpace(nombre), strings.TrimSpace(busqueda)
		if err := nombreDeVariable(nombre); err != nil {
			return nil, errUso{err}
		}
		if busqueda == "" {
			return nil, errUso{fmt.Errorf("%s no dice qué entrada de la bóveda es", nombre)}
		}
		if vistas[nombre] {
			return nil, errUso{fmt.Errorf("%s se pide dos veces, y solo puede valer una cosa", nombre)}
		}
		vistas[nombre] = true
		out = append(out, [2]string{nombre, busqueda})
	}
	return out, nil
}

func nombreDeVariable(n string) error {
	if n == "" {
		return errors.New("Falta el nombre de la variable antes del «=»")
	}
	for i, r := range n {
		letra := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		cifra := r >= '0' && r <= '9'
		if letra || (cifra && i > 0) {
			continue
		}
		return fmt.Errorf("«%s» no es un nombre de variable de entorno; valen letras sin acento, cifras y «_»", n)
	}
	return nil
}

// sacarLosSecretos busca cada entrada y arma el entorno. Devuelve también los
// apuntes a medio escribir, para que el registro diga **de qué entrada** se trata y
// no solo que pasó algo.
//
// **Se saca todo antes de lanzar nada**: si una de las tres búsquedas no encaja, lo
// que hay que hacer es decirlo, no desplegar con dos credenciales de tres.
func sacarLosSecretos(b *boveda.Boveda, pedidas [][2]string) ([]string, []boveda.Apunte, error) {
	var entorno []string
	var apuntes []boveda.Apunte
	for _, p := range pedidas {
		nombre, busqueda := p[0], p[1]
		e, err := unaSola(b, busqueda)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", nombre, err)
		}
		if e.Secreto == "" {
			return nil, nil, fmt.Errorf("%s: «%s» no tiene ningún secreto guardado", nombre, e.Titulo)
		}
		entorno = append(entorno, nombre+"="+e.Secreto)
		apuntes = append(apuntes, boveda.Apunte{Sobre: e.ID, Titulo: e.Titulo})
	}
	return entorno, apuntes, nil
}
