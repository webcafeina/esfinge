// Package cli es la cara de línea de comandos: silenciosa, pipeable y con un
// código de salida distinto por cada cosa que puede ir mal.
package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/escritura"
	"github.com/webcafeina/esfinge/internal/salida"
	"github.com/webcafeina/esfinge/internal/tema"
)

// Códigos de salida. Que «la clave es incorrecta» y «esto no es un contenedor»
// salgan distintos no es cosmética: dentro de un script, el primero se arregla
// reintentando y el segundo buscando otro fichero.
const (
	SalidaOK       = 0
	SalidaError    = 1
	SalidaUso      = 2
	SalidaClave    = 3
	SalidaCorrupto = 4
)

// opciones comunes a todos los comandos.
type opciones struct {
	entrada  string
	salida   string
	tema     string
	clave    OrigenClave
	forzar   bool
	silencio bool
}

// Ejecutar corre la línea de comandos y devuelve el código de salida.
func Ejecutar(version string) int {
	var o opciones

	raiz := &cobra.Command{
		Use:   "esfinge",
		Short: "Cifra y descifra secretos con una clave",
		Long: "Esfinge cifra contraseñas, ficheros de credenciales y cualquier otro\n" +
			"secreto con una clave que solo conocen las dos partes.\n\n" +
			"Esto es la línea de comandos, pensada para tuberías y scripts. Si lo\n" +
			"que buscas son menús y ratón, abre la aplicación Esfinge.\n\n" +
			"Una vez al día comprueba si hay una versión nueva y lo dice por la\n" +
			"salida de error, nunca por la estándar, y solo si estás en un terminal.\n" +
			"Con ESFINGE_SIN_RED=1 no sale a la red en ningún caso.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		Args:          soloArgumentosConocidos,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Sin subcomando no hay nada que ejecutar: se enseña la ayuda, que es
			// lo que espera quien escribe el nombre del programa a secas.
			return cmd.Help()
		},
	}
	raiz.PersistentFlags().StringVar(&o.tema, "tema", "auto",
		"Paleta: claro, oscuro o auto")

	raiz.AddCommand(
		comandoCifrar(&o),
		comandoDescifrar(&o),
		comandoGenerar(&o),
		comandoBoveda(&o),
	)

	// Cobra habla inglés de fábrica. La herramienta es para un cliente que la va
	// a abrir en español, y una ayuda medio traducida se lee peor que ninguna.
	// Va después de AddCommand porque también reescribe los subcomandos.
	castellanizar(raiz)

	// La consulta de versiones nuevas sale ya, para que le dé tiempo mientras se
	// cifra. Lo que no hace nunca es retrasar el resultado: ver novedad.go.
	v := vigilar(version)

	if err := raiz.Execute(); err != nil {
		e, _ := estilos(o.tema)
		fmt.Fprintln(os.Stderr, e.Mal(err.Error()))
		v.contar(e)
		return codigoDe(err)
	}

	e, _ := estilos(o.tema)
	v.contar(e)
	return SalidaOK
}

// errUso marca los fallos de escritura del propio comando —un subcomando que no
// existe, una bandera inventada, argumentos de más— para que salgan con su
// código y no confundidos con un fallo de ejecución.
type errUso struct{ err error }

func (e errUso) Error() string { return e.err.Error() }
func (e errUso) Unwrap() error { return e.err }

// soloArgumentosConocidos rechaza lo que no sea un subcomando, con un mensaje
// que dice adónde ir en vez del «unknown command» de cobra.
func soloArgumentosConocidos(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return errUso{fmt.Errorf("No conozco el comando «%s»; prueba con «esfinge ayuda»", args[0])}
}

func codigoDe(err error) int {
	var uso errUso
	switch {
	case errors.As(err, &uso):
		return SalidaUso
	case errors.Is(err, cripto.ErrClaveIncorrecta):
		return SalidaClave
	case errors.Is(err, cripto.ErrFormato),
		errors.Is(err, cripto.ErrTruncado),
		errors.Is(err, cripto.ErrDanado),
		errors.Is(err, cripto.ErrVersion):
		return SalidaCorrupto
	default:
		return SalidaError
	}
}

func estilos(nombre string) (salida.Estilos, error) {
	t, err := salida.TemaPorNombre(nombre)
	if err != nil {
		return salida.NuevosEstilos(tema.TemaOscuro), err
	}
	return salida.NuevosEstilos(t), nil
}

func banderasComunes(cmd *cobra.Command, o *opciones) {
	f := cmd.Flags()
	f.StringVarP(&o.entrada, "entrada", "i", "", "Fichero de entrada (por defecto, la entrada estándar)")
	f.StringVarP(&o.salida, "salida", "o", "", "Fichero de salida (por defecto, la salida estándar)")
	f.StringVar(&o.clave.Variable, "clave-env", "", "Leer la clave de una variable de entorno")
	f.StringVar(&o.clave.Fichero, "clave-fichero", "", "Leer la clave de un fichero")
	f.BoolVar(&o.forzar, "forzar", false, "Sobrescribir el fichero de salida si ya existe")
	f.BoolVarP(&o.silencio, "silencio", "q", false, "Sin marca ni mensajes de cortesía")
}

func comandoCifrar(o *opciones) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cifrar",
		Short: "Cifra un secreto o un fichero",
		Long: "Cifra lo que reciba y devuelve un contenedor ESF1.\n\n" +
			"Desde la entrada estándar produce una línea de texto ESF1.… que se puede\n" +
			"pegar en un correo, en un .env o en una URL sin escapar nada. Desde un\n" +
			"fichero (-i) produce un contenedor binario troceado, que no carga el\n" +
			"fichero entero en memoria por grande que sea.",
		Example: "  esfinge cifrar\n" +
			"  echo -n 'secreto' | esfinge cifrar --clave-env CLAVE\n" +
			"  esfinge cifrar -i credenciales.env -o credenciales.env.esf",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cifrar(o)
		},
	}
	banderasComunes(cmd, o)
	return cmd
}

func comandoDescifrar(o *opciones) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "descifrar",
		Short: "Descifra un contenedor de Esfinge",
		Long: "Descifra un contenedor ESF1, sea el texto de una línea o un fichero\n" +
			"binario. No hay que decirle cuál de los dos es: lo distingue solo.",
		Example: "  esfinge descifrar\n" +
			"  esfinge descifrar -i credenciales.env.esf -o credenciales.env",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return descifrar(o)
		},
	}
	banderasComunes(cmd, o)
	return cmd
}

func cifrar(o *opciones) error {
	e, err := estilos(o.tema)
	if err != nil {
		return err
	}

	clave, err := o.clave.Leer(e, true)
	if err != nil {
		return err
	}
	defer cripto.Borrar(clave)

	// Desde un fichero se usa el modo por segmentos, que es el que aguanta un
	// fichero de cualquier tamaño sin comérselo entero.
	if o.entrada != "" {
		return cifrarFichero(e, o, clave)
	}

	datos, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("No puedo leer la entrada: %w", err)
	}
	datos = bytes.TrimRight(datos, "\r\n")

	texto, err := cripto.SellarTexto(datos, clave, cripto.PerfilInteractivo)
	if err != nil {
		return err
	}
	return escribirTexto(e, o, texto)
}

func cifrarFichero(e salida.Estilos, o *opciones, clave []byte) error {
	entrada, err := os.Open(o.entrada)
	if err != nil {
		return fmt.Errorf("No puedo abrir %s: %w", o.entrada, err)
	}
	defer entrada.Close()

	destino := o.salida
	if destino == "" {
		destino = o.entrada + ".esf"
	}

	return conSalida(e, o, destino, func(w io.Writer) error {
		return cripto.SellarFlujo(w, bufio.NewReader(entrada), clave, cripto.PerfilInteractivo)
	})
}

func descifrar(o *opciones) error {
	e, err := estilos(o.tema)
	if err != nil {
		return err
	}

	var lector io.Reader = os.Stdin
	if o.entrada != "" {
		f, err := os.Open(o.entrada)
		if err != nil {
			return fmt.Errorf("No puedo abrir %s: %w", o.entrada, err)
		}
		defer f.Close()
		lector = f
	}

	// Se espían los primeros bytes para saber con qué se trata sin consumirlos.
	// El quinto byte lo dice todo: en el formato de texto es el punto de «ESF1.»
	// y en el binario es el número de versión.
	buf := bufio.NewReader(lector)
	cabeza, err := buf.Peek(5)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("No puedo leer la entrada: %w", err)
	}
	esTexto := len(cabeza) < 5 || cabeza[4] == '.'

	clave, err := o.clave.Leer(e, false)
	if err != nil {
		return err
	}
	defer cripto.Borrar(clave)

	if esTexto {
		crudo, err := io.ReadAll(buf)
		if err != nil {
			return fmt.Errorf("No puedo leer la entrada: %w", err)
		}
		datos, err := cripto.AbrirTexto(string(crudo), clave)
		if err != nil {
			return err
		}
		return escribirDatos(e, o, o.salida, datos)
	}

	destino := o.salida
	if destino == "" && o.entrada != "" {
		destino = strings.TrimSuffix(o.entrada, ".esf")
		if destino == o.entrada {
			destino = o.entrada + ".claro"
		}
	}
	return conSalida(e, o, destino, func(w io.Writer) error {
		return cripto.AbrirFlujo(w, buf, clave)
	})
}

// escribirTexto saca el contenedor de texto. Si va a un terminal se presenta en
// un panel; si va a una tubería sale desnudo, sin una sola secuencia de color.
func escribirTexto(e salida.Estilos, o *opciones, texto string) error {
	if o.salida != "" {
		return escribirDatos(e, o, o.salida, []byte(texto+"\n"))
	}

	if !aTerminal(os.Stdout) {
		fmt.Println(texto)
		marcaEnStderr(e, o)
		return nil
	}

	fmt.Println()
	fmt.Println(e.Panel.Render(
		e.Etiq("cifrado") + "\n\n" + e.Codigo.Render(texto)))
	fmt.Println("  " + e.Info("Guárdalo entero: sin el prefijo ESF1. no se puede descifrar"))
	marca(e, o, os.Stdout)
	return nil
}

func escribirDatos(e salida.Estilos, o *opciones, destino string, datos []byte) error {
	if destino == "" {
		if _, err := os.Stdout.Write(datos); err != nil {
			return err
		}
		if aTerminal(os.Stdout) && !bytes.HasSuffix(datos, []byte("\n")) {
			fmt.Println()
		}
		marcaEnStderr(e, o)
		return nil
	}
	return conSalida(e, o, destino, func(w io.Writer) error {
		_, err := w.Write(datos)
		return err
	})
}

// conSalida escribe a un fichero temporal en el mismo directorio y lo renombra
// al terminar. Así una interrupción a mitad no deja un fichero a medias con el
// nombre del bueno, que en un fichero de credenciales es la diferencia entre un
// susto y una pérdida.
func conSalida(e salida.Estilos, o *opciones, destino string, escribir func(io.Writer) error) error {
	if destino == "" || destino == "-" {
		if err := escribir(os.Stdout); err != nil {
			return err
		}
		marcaEnStderr(e, o)
		return nil
	}

	if info, err := os.Stat(destino); err == nil {
		// /dev/null y compañía: escribir un temporal al lado y renombrar encima no
		// tiene sentido —ni permiso— sobre un fichero que no es un fichero.
		if !info.Mode().IsRegular() {
			f, err := os.OpenFile(destino, os.O_WRONLY, 0o600)
			if err != nil {
				return fmt.Errorf("No puedo escribir en %s: %w", destino, err)
			}
			defer f.Close()

			w := bufio.NewWriter(f)
			if err := escribir(w); err != nil {
				return err
			}
			return w.Flush()
		}
		if !o.forzar {
			return fmt.Errorf("%s ya existe; usa --forzar para sobrescribirlo", destino)
		}
	}

	// La receta vive ahora en internal/escritura, porque la bóveda necesita
	// exactamente la misma y con más motivo. Allí gana además el «Sync» antes de
	// cerrar, que aquí faltaba: sin él, el renombrado puede llegar al disco antes
	// que los datos y un corte de luz deja un fichero de cero bytes con el nombre
	// del bueno.
	if err := escritura.Atomica(destino, escritura.Opciones{}, escribir); err != nil {
		return err
	}

	if !o.silencio {
		info, err := os.Stat(destino)
		tam := ""
		if err == nil {
			tam = fmt.Sprintf(" (%s)", tamanoLegible(info.Size()))
		}
		fmt.Fprintln(os.Stderr, e.Ok("Escrito en "+destino+tam))
		marcaEnStderr(e, o)
	}
	return nil
}

func tamanoLegible(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1024*1024))
	}
}

func aTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// marca imprime la firma de la casa. Dentro de una tubería no se imprime en la
// salida —un banner ahí contamina el dato— pero sí en stderr, que la persona
// sigue viendo. Así la marca está en todas partes sin estorbar en ninguna.
func marca(e salida.Estilos, o *opciones, f *os.File) {
	if o.silencio || !aTerminal(f) {
		return
	}
	fmt.Fprintln(f, "\n"+salida.Firma(e))
}

func marcaEnStderr(e salida.Estilos, o *opciones) {
	if o.silencio || !aTerminal(os.Stderr) {
		return
	}
	fmt.Fprintln(os.Stderr, salida.Firma(e))
}
