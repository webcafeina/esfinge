//go:build unix

package app

import (
	"os"
	"syscall"
)

// inodo saca el número de inodo, que es lo que distingue «no ha hecho nada» de «ha vuelto a
// copiar»: la copia nueva lleva la fecha del origen a propósito, así que por la fecha son
// iguales las dos.
func inodo(fi os.FileInfo) (uint64, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return st.Ino, true
}
