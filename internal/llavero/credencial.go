package llavero

// credencial es la `CREDENTIALW` de Windows, la estructura que se le pasa a
// `CredWriteW` y que devuelve `CredReadW`.
//
// **Vive en un fichero sin etiqueta de compilación a propósito**, y no junto al
// código que la usa. La razón es que es lo único de toda la C3 que se puede
// comprobar de verdad en esta máquina: Windows lee estos campos **por su sitio en
// memoria**, no por su nombre, así que un relleno de más o de menos hace que
// `CredentialBlob` caiga donde no toca y la llamada devuelva basura o reviente
// —en un ordenador que aquí no hay—. Compilándola en todas partes, una prueba
// normal puede mirar los desplazamientos, y mira **esta** estructura y no una
// copia suya, que probar una copia es probar la copia.
//
// Los tipos son todos de tamaño fijo o del tamaño de un puntero, así que en
// linux/amd64 se alinean igual que en windows/amd64 y en windows/arm64. Por eso
// `LastWritten` son dos `uint32` en vez de `windows.Filetime`: el paquete de
// Windows no compila fuera de Windows, y aquí lo que hace falta es que esto sí.
type credencial struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWrittenBajo    uint32
	LastWrittenAlto    uint32
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}
