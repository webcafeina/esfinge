// Package red dice si Esfinge tiene permiso para salir de esta máquina.
//
// **Existe porque el freno que había era mentira a medias.** `ESFINGE_SIN_RED`
// se comprobaba en un solo `os.Getenv` dentro de la línea de comandos, y la
// ventana no lo miraba nunca: quien la ponía en su entorno creyendo que apagaba
// la red apagaba media. Con una sola conexión —la de versiones, que la ventana
// además deja apagar en Ajustes— eso era un descuido pequeño. Con dos deja de
// serlo.
//
// La regla que este paquete sostiene, y que docs/seguridad.md puede prometer
// porque hay una prueba por cada salida:
//
//	ESFINGE_SIN_RED apaga TODAS las salidas a la red, sin excepción y sin ajuste
//	que las reviva.
//
// Es lo que se pone en una imagen de contenedor, en un servidor de compilación o
// en una máquina que no debe hablar con nadie.
package red

import (
	"os"
	"sync"
)

// laVariable es el nombre, en un sitio, para que no se escriba a mano en cinco.
const laVariable = "ESFINGE_SIN_RED"

var (
	unaVez sync.Once
	sinRed bool
)

// SinRed dice si está prohibido salir a la red.
//
// **Se lee una sola vez**, la primera. Un programa que cambia de opinión sobre
// si puede usar la red según cuándo se le pregunte es peor que cualquiera de las
// dos respuestas: lo que decide es el entorno con el que arrancó.
func SinRed() bool {
	unaVez.Do(func() { sinRed = os.Getenv(laVariable) != "" })
	return sinRed
}

// olvidar deshace la memoria. Solo para las pruebas, que necesitan preguntar con
// la variable puesta y quitada dentro del mismo proceso.
func olvidar() {
	unaVez = sync.Once{}
	sinRed = false
}
