//go:build darwin && !cgo

package llavero

// **Un macOS sin cgo no tiene Touch ID, y hace falta decirlo en un fichero.**
//
// `llavero_darwin.go` es cgo, y un fichero cgo **no entra en la compilación
// cuando `CGO_ENABLED=0`** — que es justo como se compila la línea de comandos
// para los seis objetivos (`make publicar`), porque así cruza de plataforma desde
// esta máquina. Sin esto, los dos objetivos de Mac se caían con «undefined:
// delSistema», y no aquí sino en medio de una publicación.
//
// La línea de comandos no abre la bóveda con la huella —es para tuberías y
// scripts—, así que lo que pierde con esto es nada. La aplicación con ventana sí
// lleva cgo, y ahí manda `llavero_darwin.go`.

func delSistema() Llavero { return Ninguno{} }
