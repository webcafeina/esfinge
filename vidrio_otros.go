//go:build !dev && (!darwin || !cgo)

package main

// ponerElVidrio no hace nada fuera de macOS.
//
// En Windows el vidrio lo pone Mica, que sí funciona con lo que ofrece Wails, y
// en Linux no hay efecto que poner. El remate de AppKit es solo de macOS: ver
// vidrio_darwin.go.
func ponerElVidrio() {}
