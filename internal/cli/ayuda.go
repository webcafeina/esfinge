package cli

import "github.com/spf13/cobra"

// castellanizar sustituye los rótulos que cobra trae en inglés y quita el
// comando «completion», que aparece solo y desentona en una ayuda en español.
func castellanizar(raiz *cobra.Command) {
	raiz.CompletionOptions.DisableDefaultCmd = true

	// Una bandera mal escrita es un fallo de uso, no de ejecución, y sale con su
	// propio código para que un script pueda distinguirlos.
	raiz.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return errUso{err}
	})

	raiz.SetHelpCommand(&cobra.Command{
		Use:    "ayuda",
		Short:  "Explica un comando",
		Hidden: false,
		Run: func(cmd *cobra.Command, args []string) {
			objetivo, _, err := raiz.Find(args)
			if err != nil || objetivo == nil {
				_ = raiz.Help()
				return
			}
			_ = objetivo.Help()
		},
	})

	raiz.PersistentFlags().BoolP("help", "h", false, "Muestra esta ayuda")

	// Cobra crea el flag de versión al vuelo con su propio texto en inglés;
	// forzando su creación aquí se puede reescribir antes de que nadie lo lea.
	raiz.InitDefaultVersionFlag()
	if f := raiz.Flags().Lookup("version"); f != nil {
		f.Usage = "Muestra la versión"
	}

	// Sin esto la línea de uso acaba en «[flags]», que es lo único en inglés que
	// queda a la vista.
	raiz.DisableFlagsInUseLine = true
	for _, sub := range raiz.Commands() {
		sub.DisableFlagsInUseLine = true
	}

	raiz.SetUsageTemplate(plantilla)
	raiz.SetVersionTemplate("esfinge {{.Version}}\n")
}

// La plantilla de cobra, con los rótulos traducidos. Las expresiones entre
// llaves son suyas y tienen que quedarse como están.
const plantilla = `Uso:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [comando]{{end}}{{if gt (len .Aliases) 0}}

También:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Ejemplos:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Comandos:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "ayuda"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Opciones:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Opciones generales:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

«{{.CommandPath}} ayuda [comando]» explica cualquiera de ellos.{{end}}
`
