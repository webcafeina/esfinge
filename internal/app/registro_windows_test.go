//go:build windows

package app

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

// **El registro de Windows de verdad**, en una clave de pruebas que se borra al
// acabar. Es lo único del canal en Windows que se puede ejecutar sin que nadie lo
// instale: corre en la máquina Windows de GitHub, en cada publicación.
func TestElRegistroDeWindowsDeVerdad(t *testing.T) {
	const raiz = `Software\Webcafeina\EsfingePruebas`
	clave := raiz + `\NativeMessagingHosts\` + nombreDelHost
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, clave)
		_ = registry.DeleteKey(registry.CURRENT_USER, raiz+`\NativeMessagingHosts`)
		_ = registry.DeleteKey(registry.CURRENT_USER, raiz)
	})

	r := registroDeWindows{}
	const ruta = `C:\Users\alguien\AppData\Roaming\Esfinge\NativeMessagingHosts\com.webcafeina.esfinge.chrome.json`
	if err := r.Poner(clave, ruta); err != nil {
		t.Fatalf("no ha podido escribir la clave: %v", err)
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, clave, registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("la clave no está: %v", err)
	}
	valor, _, err := k.GetStringValue("")
	k.Close()
	if err != nil || valor != ruta {
		t.Errorf("el valor por defecto es %q (%v), y tenía que ser la ruta del manifiesto", valor, err)
	}

	if err := r.Quitar(clave); err != nil {
		t.Fatalf("no ha podido borrarla: %v", err)
	}
	if k, err := registry.OpenKey(registry.CURRENT_USER, clave, registry.QUERY_VALUE); err == nil {
		k.Close()
		t.Error("la clave sigue ahí después de quitarla")
	}
	// Apagar el canal dos veces tiene que dar lo mismo que una.
	if err := r.Quitar(clave); err != nil {
		t.Errorf("quitar una clave que ya no está falla: %v", err)
	}
}
