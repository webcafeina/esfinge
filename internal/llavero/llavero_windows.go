//go:build windows

package llavero

// Windows Hello todavía no está: es la **C3**, y hasta entonces un Windows se
// comporta como un Linux —la pantalla no ofrece el botón y se teclea la maestra—.
//
// Este fichero existe **porque tiene que existir**, y eso costó encontrarlo: al
// escribir la C1 se le puso a `llavero_otros.go` la etiqueta `!darwin && !windows`
// dando por hecho que los dos ficheros propios llegarían enseguida, y **los dos
// objetivos de Windows se quedaron sin `delSistema`**. No lo vio nada: `make
// comprobar` solo miraba esta máquina, y la línea de comandos se compila para los
// seis objetivos **en `make publicar`**, o sea en medio de una publicación. Ahora
// `make comprobar` cruza los seis, que es la lección: una etiqueta de compilación
// que excluye un sistema es una promesa de escribir ese fichero, y hasta que se
// escriba hay que dejarlo dicho en Go y no en la cabeza de nadie.

func delSistema() Llavero { return Ninguno{} }
