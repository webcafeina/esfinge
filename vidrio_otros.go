//go:build !dev && (!darwin || !cgo)

package main

// ponerElVidrio no hace nada fuera de macOS.
//
// En Windows el vidrio lo pone Mica, que sí funciona con lo que ofrece Wails, y
// en Linux no hay efecto que poner. El remate de AppKit es solo de macOS: ver
// vidrio_darwin.go.
func ponerElVidrio() {}

// estadoDelVidrio no dice nada fuera de macOS: el vidrio de Windows lo pone Mica
// y no hay nada equivalente que mirar.
func estadoDelVidrio() string { return "" }
