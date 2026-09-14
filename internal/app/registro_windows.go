//go:build windows

package app

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// El registro de Windows, de verdad (ver `registro` en manifiestos.go).
//
// **Solo `HKCU`**: Esfinge corre como la persona, y lo que apunta es para sus
// navegadores, igual que en macOS y Linux escribe en su carpeta. Escribir en `HKLM`
// pediría permisos de administrador a una aplicación que no los tiene ni los quiere.
type registroDeWindows struct{}

func registroDelSistema() registro { return registroDeWindows{} }

func (registroDeWindows) Poner(clave, valor string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, clave, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue("", valor)
}

// Quitar borra la clave. Que no estuviera no es un fallo: apagar el canal dos veces
// tiene que dar lo mismo que apagarlo una.
func (registroDeWindows) Quitar(clave string) error {
	err := registry.DeleteKey(registry.CURRENT_USER, clave)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}
