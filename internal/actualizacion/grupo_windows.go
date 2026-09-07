//go:build windows

package actualizacion

import (
	"os/exec"
	"syscall"
)

// ponerEnSuPropioGrupo desata el guion de Esfinge.
//
// En Windows no hay sesiones como en Unix: lo que hace falta es que el proceso
// no herede la consola y sobreviva por su cuenta.
func ponerEnSuPropioGrupo(orden *exec.Cmd) {
	orden.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008, // DETACHED_PROCESS
		HideWindow:    true,
	}
}
