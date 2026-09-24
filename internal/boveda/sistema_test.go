package boveda

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// La ranura del sistema (fase C): se pone con la bóveda abierta, abre sola, se
// quita, y **no se sube nunca**.
func TestLaRanuraDelSistemaAbreYNoViaja(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	b, _, err := Crear(ruta, "una maestra bien larga para la prueba")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Banco", Secreto: "la del banco"}); err != nil {
		t.Fatal(err)
	}
	if b.TieneRanuraDelSistema() {
		t.Fatal("una bóveda recién creada no tiene desbloqueo del sistema")
	}
	if RanuraDelSistemaEn(ruta) {
		t.Fatal("y el fichero tampoco lo dice")
	}

	secreto, err := SecretoDelSistema()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.PonerRanuraDelSistema(secreto); err != nil {
		t.Fatal(err)
	}
	if !b.TieneRanuraDelSistema() || !RanuraDelSistemaEn(ruta) {
		t.Fatal("la ranura no ha quedado puesta")
	}

	// Abre, y lo que hay dentro es lo de siempre.
	conElSistema, err := AbrirConElSistema(ruta, secreto)
	if err != nil {
		t.Fatal(err)
	}
	if e := conElSistema.Buscar("Banco"); len(e) != 1 {
		t.Fatalf("abierta con el sistema hay %d entradas", len(e))
	}
	// Y la maestra sigue abriendo: **la ranura del sistema nunca es la única**.
	if _, err := Abrir(ruta, "una maestra bien larga para la prueba"); err != nil {
		t.Fatalf("la maestra ha dejado de abrir: %v", err)
	}

	// Otro secreto no abre, y lo dice sin confundirlo con «no la tiene».
	otro, _ := SecretoDelSistema()
	if _, err := AbrirConElSistema(ruta, otro); err == nil {
		t.Fatal("un secreto que no es abre la bóveda")
	}

	// **Lo que no puede pasar nunca**: que esa ranura salga hacia el servidor.
	datos, _, err := b.PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(datos), RanuraDelSistema) {
		t.Fatal("la ranura del sistema viaja en la subida: es de un equipo y solo de uno")
	}
	// Y lo que se sube se abre igual con la maestra, o sea que se volvió a sellar.
	if _, err := AbrirEnMemoria(datos, "una maestra bien larga para la prueba"); err != nil {
		t.Fatalf("lo que se sube ya no abre: %v", err)
	}

	if err := b.QuitarRanuraDelSistema(); err != nil {
		t.Fatal(err)
	}
	if b.TieneRanuraDelSistema() || RanuraDelSistemaEn(ruta) {
		t.Fatal("la ranura sigue ahí después de quitarla")
	}
	if _, err := AbrirConElSistema(ruta, secreto); err == nil {
		t.Fatal("el secreto de antes sigue abriendo cuando ya no hay ranura")
	}
}

// **El secreto no se envuelve con el coste de una contraseña humana.**
//
// Son 32 bytes al azar: no hay diccionario contra el que defenderse, y Argon2id
// interactivo ahí solo sirve para que desbloquear con la huella tarde un segundo,
// que es justo lo que esto viene a quitar.
//
// Se miran **los parámetros que lleva el sobre**, no lo que tarda: medir tiempos
// aquí da un número distinto en cada máquina y además lo tapan el cuerpo y el
// fichero. La cabecera de un contenedor `ESF1` va en claro y su forma está escrita
// en `internal/cripto`: magia(4) · versión(1) · modo(1) · memoria(4) · pasadas(4).
func TestElSobreDelSistemaVaConElPerfilBarato(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	b, _, err := Crear(ruta, "una maestra bien larga para la prueba")
	if err != nil {
		t.Fatal(err)
	}
	secreto, _ := SecretoDelSistema()
	if err := b.PonerRanuraDelSistema(secreto); err != nil {
		t.Fatal(err)
	}

	perfilDe := func(tipo string) (memoria, pasadas uint32) {
		t.Helper()
		i := b.ranura(tipo)
		if i < 0 {
			t.Fatalf("no hay ranura %q", tipo)
		}
		crudo, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(b.doc.Sobres[i].Contenedor, "ESF1."))
		if err != nil {
			t.Fatal(err)
		}
		return binary.BigEndian.Uint32(crudo[6:10]), binary.BigEndian.Uint32(crudo[10:14])
	}

	memSistema, pasSistema := perfilDe(RanuraDelSistema)
	memMaestra, _ := perfilDe(RanuraMaestra)
	t.Logf("sistema %d KiB · maestra %d KiB", memSistema, memMaestra)

	if memSistema != cripto.PerfilLlave.Memoria || pasSistema != cripto.PerfilLlave.Pasadas {
		t.Errorf("el sobre del sistema va con %d KiB y %d pasadas, y le toca el perfil de llave (%d KiB, %d)",
			memSistema, pasSistema, cripto.PerfilLlave.Memoria, cripto.PerfilLlave.Pasadas)
	}
	// Y la maestra sigue con el suyo: esto no puede abaratar lo que protege una
	// contraseña que alguien teclea.
	if memMaestra != cripto.PerfilInteractivo.Memoria {
		t.Errorf("la ranura maestra va con %d KiB y le tocan %d", memMaestra, cripto.PerfilInteractivo.Memoria)
	}
}

// Un fichero que no existe o que no es una bóveda no revienta al preguntarle.
func TestRanuraDelSistemaEnLoQueNoEsUnaBoveda(t *testing.T) {
	dir := t.TempDir()
	if RanuraDelSistemaEn(filepath.Join(dir, "no-existe")) {
		t.Error("dice que sí de un fichero que no está")
	}
	otro := filepath.Join(dir, "cualquiera.txt")
	if err := os.WriteFile(otro, []byte("hola"), 0o600); err != nil {
		t.Fatal(err)
	}
	if RanuraDelSistemaEn(otro) {
		t.Error("dice que sí de algo que no es una bóveda")
	}
}
