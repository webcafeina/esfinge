//go:build !darwin && !windows

package llavero

// En Linux no hay un llavero del sistema que pida la huella y que Esfinge pueda
// usar sin arrastrar medio escritorio detrás.
//
// **Y eso no es una carencia que haya que tapar.** El Secret Service de
// freedesktop —GNOME Keyring, KWallet— guarda secretos, pero lo abre la sesión y
// no pide nada al leerlos: sería un cerrojo sin cerradura. Donde no hay, se
// teclea la contraseña maestra, y la pantalla lo dice en vez de ofrecer algo que
// no está.
func delSistema() Llavero { return Ninguno{} }
