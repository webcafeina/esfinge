// Package interfaz lleva dentro la interfaz ya construida.
//
// El contenido de dist/ lo pone «make frontend» copiando lo que produce Vite.
// Está aquí y no en frontend/ porque go:embed no puede salir del directorio del
// paquete, y tener el punto de entrada en la raíz del proyecto solo para eso
// obligaría a romper la organización del resto.
package interfaz

import "embed"

//go:embed all:dist
var Recursos embed.FS
