//go:build !windows

package actualizacion

import (
	"os/exec"
	"syscall"
)

// ponerEnSuPropioGrupo desata el guion de Esfinge.
//
// Sin esto el hijo comparte grupo de procesos con la ventana, y cerrarla podría
// llevárselo por delante justo cuando está a mitad del cambiazo.
func ponerEnSuPropioGrupo(orden *exec.Cmd) {
	orden.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
