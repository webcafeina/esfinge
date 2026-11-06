package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// **Una bandera puesta y vacía no es «no se ha dicho»: es un error de uso.**
//
// El camino de la bóveda compartida se elige con `if ob.compartida != ""`, y los de
// la clave y la ruta igual. Así que `--compartida ""` cae en el camino de «no se ha
// dicho» y se trabaja con **la bóveda propia del equipo**, contestando «Guardada» tan
// tranquilo. Pasó aquí el 2026-11-06: se creyó estar escribiendo en la bóveda de un
// cliente y la entrada estaba en la personal del servidor.
//
// Y pasa de verdad fuera de una prueba: `--compartida "$PROYECTO"` con la variable sin
// definir es un despiste de un guion, que es lo que corre en un servidor. Es la familia
// de `.trim()` y del interruptor con un espacio de más — **lo que da la apariencia de
// funcionar es peor que lo que falla**.
//
// Lo que **no** se puede hacer es prohibir el vacío en todas: en `editar`, un
// `--notas ""` es la forma de vaciar las notas, y eso tiene que seguir valiendo. La
// frontera no es el tipo de la bandera, es para qué sirve: las que **eligen dónde o de
// dónde** no admiten vacío, y las que **llevan contenido** sí.
//
// Por eso se marca **donde se declara** y no en una lista aparte: una lista de nombres
// en otro fichero se desalinea sola en cuanto alguien añada una bandera, que es el
// patrón que a este proyecto ya le ha costado tres fallos.
const exigeValorAnotacion = "esfinge-exige-valor"

// exigeValor marca una bandera como de las que no admiten el vacío. Se llama justo
// después de declararla.
func exigeValor(f *pflag.FlagSet, nombres ...string) {
	for _, n := range nombres {
		if err := f.SetAnnotation(n, exigeValorAnotacion, []string{"sí"}); err != nil {
			// Solo falla si la bandera no existe, que sería un error de programación
			// aquí mismo; que se vea al arrancar y no en producción.
			panic(fmt.Sprintf("exigeValor sobre una bandera que no existe: %s (%v)", n, err))
		}
	}
}

// vigilarLasQueExigenValor rechaza las banderas marcadas que se hayan puesto vacías.
//
// Recorre `cmd.Flags()`, que en el comando que se ejecuta incluye las heredadas, así
// que vale igual para una bandera del padre que para una del propio verbo.
func vigilarLasQueExigenValor(cmd *cobra.Command) error {
	var mal *pflag.Flag
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if mal != nil || !f.Changed || f.Annotations[exigeValorAnotacion] == nil {
			return
		}
		if f.Value.String() == "" {
			mal = f
		}
	})
	if mal == nil {
		return nil
	}
	return errUso{fmt.Errorf(
		"--%s está puesta y vacía, y eso no es lo mismo que no decirla: "+
			"dile un valor o quítala. Tal cual, Esfinge trabajaría con otra bóveda sin avisar", mal.Name)}
}
