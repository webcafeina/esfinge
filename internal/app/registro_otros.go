//go:build !windows

package app

// Fuera de Windows no hay registro: los navegadores miran una carpeta.
func registroDelSistema() registro { return nil }
