package llavero

import (
	"testing"
	"unsafe"
)

// TestLaCredencialDeWindowsCaeDondeWindowsLaEspera compara la estructura con la
// `CREDENTIALW` de la cabecera `wincred.h`, campo a campo.
//
// **Es la única parte de Windows Hello que se comprueba aquí**, y por eso existe:
// el resto de la C3 se escribió a ciegas y lo único que se puede decir de ello es
// que compila. Esto no. Los desplazamientos de una estructura de C son aritmética,
// y la aritmética se puede mirar desde cualquier sistema: los tipos son todos de
// tamaño fijo o de tamaño de puntero, así que en amd64 salen los mismos números
// aquí que en Windows.
//
// Si esta prueba se pone roja, **no se toca**: lo que hay mal es la estructura, y
// el síntoma en un Windows sería que `CredWriteW` guardara cualquier cosa.
func TestLaCredencialDeWindowsCaeDondeWindowsLaEspera(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("los desplazamientos de referencia son los de 64 bits")
	}

	quiero := []struct {
		campo string
		donde uintptr
	}{
		{"Flags", 0},
		{"Type", 4},
		{"TargetName", 8},
		{"Comment", 16},
		{"LastWritten", 24},
		{"CredentialBlobSize", 32},
		// **El de los 40 es el que importa**: entre el tamaño del secreto y su
		// puntero, C mete cuatro bytes de relleno para alinear a ocho. Si Go no los
		// metiera —o los metiera donde no toca—, Windows leería el puntero cuatro
		// bytes antes y escribiría una llave que no es.
		{"CredentialBlob", 40},
		{"Persist", 48},
		{"AttributeCount", 52},
		{"Attributes", 56},
		{"TargetAlias", 64},
		{"UserName", 72},
	}
	var c credencial
	tengo := map[string]uintptr{
		"Flags":              unsafe.Offsetof(c.Flags),
		"Type":               unsafe.Offsetof(c.Type),
		"TargetName":         unsafe.Offsetof(c.TargetName),
		"Comment":            unsafe.Offsetof(c.Comment),
		"LastWritten":        unsafe.Offsetof(c.LastWrittenBajo),
		"CredentialBlobSize": unsafe.Offsetof(c.CredentialBlobSize),
		"CredentialBlob":     unsafe.Offsetof(c.CredentialBlob),
		"Persist":            unsafe.Offsetof(c.Persist),
		"AttributeCount":     unsafe.Offsetof(c.AttributeCount),
		"Attributes":         unsafe.Offsetof(c.Attributes),
		"TargetAlias":        unsafe.Offsetof(c.TargetAlias),
		"UserName":           unsafe.Offsetof(c.UserName),
	}
	for _, q := range quiero {
		if tengo[q.campo] != q.donde {
			t.Errorf("%s cae en %d y wincred.h lo espera en %d", q.campo, tengo[q.campo], q.donde)
		}
	}
	if n := unsafe.Sizeof(c); n != 80 {
		t.Errorf("la estructura mide %d bytes y CREDENTIALW mide 80", n)
	}
}
