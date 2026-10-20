package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/boveda"
)

const elSecretoDePrueba = "testigo-de-cloudflare-que-no-puede-salir"

func unaBovedaConSecretos(t *testing.T) (ruta, conLaMaestra string) {
	t.Helper()
	const maestra = "una contraseña maestra bien larga"
	dir := t.TempDir()
	ruta = filepath.Join(dir, "boveda.esfinge")
	b, _, err := boveda.Crear(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []boveda.Entrada{
		{Tipo: boveda.TipoCredencial, Titulo: "Cloudflare API", Secreto: elSecretoDePrueba},
		{Tipo: boveda.TipoCredencial, Titulo: "Neon Tempero", Secreto: "postgres://x"},
		{Tipo: boveda.TipoCredencial, Titulo: "Banco del norte", Secreto: "una"},
		{Tipo: boveda.TipoCredencial, Titulo: "Banco del sur", Secreto: "otra"},
		{Tipo: boveda.TipoNota, Titulo: "Sin nada dentro"},
	} {
		if err := b.Poner(e); err != nil {
			t.Fatal(err)
		}
	}
	b.Cerrar()

	conLaMaestra = filepath.Join(dir, "maestra.txt")
	if err := os.WriteFile(conLaMaestra, []byte(maestra), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta, conLaMaestra
}

// **El secreto llega al programa y no a la salida de Esfinge** (ADR 0057), que es
// las dos mitades de para qué existe esto.
//
// Lo del hijo se comprueba haciéndole escribir su entorno en un fichero —no en su
// salida, que es del programa y puede enseñar lo que quiera—, y lo de Esfinge
// buscando el secreto **en los bytes** de lo que escribe, como ya se hace con el
// canal de los agentes.
func TestEjecutarLeDaElSecretoAlProgramaYNoALaSalida(t *testing.T) {
	ruta, conLaMaestra := unaBovedaConSecretos(t)
	donde := filepath.Join(t.TempDir(), "lo-que-vio-el-hijo")
	// Por el entorno de aquí, que de paso comprueba que el hijo **hereda** lo que
	// hubiera: un programa lanzado sin su PATH ni su HOME no desplegaría nada.
	t.Setenv("DONDE", donde)

	o := &opciones{tema: "oscuro", silencio: true}
	ob := &opcionesBoveda{ruta: ruta}
	ob.clave.Fichero = conLaMaestra

	cmd := comandoBovedaEjecutar(o, ob)
	if err := cmd.Flags().Set("var", "CF_TOKEN=Cloudflare API"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("var", "DB_URL=Neon Tempero"); err != nil {
		t.Fatal(err)
	}
	dicho := loQueSaleAlError(t, func() {
		err := cmd.RunE(cmd, []string{"sh", "-c", `printf '%s\n%s\n' "$CF_TOKEN" "$DB_URL" > "$DONDE"`})
		if err != nil {
			t.Fatal(err)
		}
	})

	visto, err := os.ReadFile(donde)
	if err != nil {
		t.Fatalf("el programa no ha podido escribir lo que vio: %v", err)
	}
	if !strings.Contains(string(visto), elSecretoDePrueba) {
		t.Fatalf("al programa no le ha llegado el secreto: %q", visto)
	}
	if !strings.Contains(string(visto), "postgres://x") {
		t.Fatalf("al programa le ha llegado una de las dos variables y no la otra: %q", visto)
	}
	// Y lo que Esfinge dice no lleva el secreto dentro, se busque como se busque.
	if bytes.Contains([]byte(dicho), []byte(elSecretoDePrueba)) {
		t.Fatalf("el secreto sale por la salida de error de Esfinge:\n%s", dicho)
	}
}

// Y queda apuntado **de qué entrada** se trata, a qué programa y sin el secreto
// dentro: es la pregunta que hoy no se puede contestar con un `.env`.
func TestEjecutarDejaRastroEnElRegistro(t *testing.T) {
	ruta, conLaMaestra := unaBovedaConSecretos(t)
	o := &opciones{tema: "oscuro", silencio: true}
	ob := &opcionesBoveda{ruta: ruta}
	ob.clave.Fichero = conLaMaestra

	cmd := comandoBovedaEjecutar(o, ob)
	if err := cmd.Flags().Set("var", "CF_TOKEN=Cloudflare API"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"true"}); err != nil {
		t.Fatal(err)
	}

	// Se vuelve a abrir el fichero, que es donde tiene que haber quedado: lo que
	// quede en memoria no lo lee nadie después.
	b, err := boveda.Abrir(ruta, "una contraseña maestra bien larga")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	reg := b.Registro()
	if len(reg) != 1 {
		t.Fatalf("hay %d apuntes en el registro", len(reg))
	}
	a := reg[0]
	if a.Titulo != "Cloudflare API" || a.Que != "ejecutar" || a.Quien != "true" {
		t.Errorf("el apunte no dice qué pasó: %+v", a)
	}
	if a.Como != boveda.ApunteLlaveDelEquipo {
		t.Errorf("el apunte dice que lo aprobó alguien: %q", a.Como)
	}
	if a.Resultado != boveda.ApunteHecho {
		t.Errorf("resultado %q", a.Resultado)
	}
	// Y nunca el secreto: el registro vive dentro del cuerpo cifrado, pero eso no es
	// excusa para escribir una contraseña en él.
	crudo, _ := os.ReadFile(ruta)
	if bytes.Contains(crudo, []byte(elSecretoDePrueba)) {
		t.Error("el fichero lleva el secreto en claro")
	}
}

// **El código del programa se devuelve tal cual.** Dentro de un guion, esto tiene
// que fallar igual que fallaría el programa a pelo; con un 1 de oficio, un
// despliegue que devuelve 7 pasaría por un fallo cualquiera de Esfinge.
func TestEjecutarDevuelveElCodigoDelPrograma(t *testing.T) {
	ruta, conLaMaestra := unaBovedaConSecretos(t)
	o := &opciones{tema: "oscuro", silencio: true}
	ob := &opcionesBoveda{ruta: ruta}
	ob.clave.Fichero = conLaMaestra

	cmd := comandoBovedaEjecutar(o, ob)
	if err := cmd.Flags().Set("var", "X=Cloudflare API"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"sh", "-c", "exit 7"})
	if err == nil {
		t.Fatal("un programa que falla no ha dado error")
	}
	var hijo errDelHijo
	if !errors.As(err, &hijo) || hijo.codigo != 7 {
		t.Fatalf("el error no lleva el código del programa: %v", err)
	}
	if codigoDe(err) != 7 {
		t.Fatalf("la línea de comandos saldría con %d", codigoDe(err))
	}
}

// **O todo o nada**: con tres variables pedidas y una búsqueda que no encaja, no se
// lanza nada. Desplegar con dos credenciales de tres es un fallo que aparece lejos
// de aquí.
func TestEjecutarNoLanzaNadaSiFaltaUnSecreto(t *testing.T) {
	ruta, conLaMaestra := unaBovedaConSecretos(t)
	testigo := filepath.Join(t.TempDir(), "se-ha-lanzado")

	o := &opciones{tema: "oscuro", silencio: true}
	ob := &opcionesBoveda{ruta: ruta}
	ob.clave.Fichero = conLaMaestra

	for _, caso := range []struct {
		nombre, pide, dice string
	}{
		{"la búsqueda no encaja con nada", "X=esto no está", "No hay nada en la bóveda"},
		{"la búsqueda encaja con varias", "X=banco", "afina la búsqueda"},
		{"la entrada no tiene secreto", "X=Sin nada dentro", "no tiene ningún secreto"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			cmd := comandoBovedaEjecutar(o, ob)
			if err := cmd.Flags().Set("var", caso.pide); err != nil {
				t.Fatal(err)
			}
			// Se pide también una que sí está, para que lo que se comprueba sea que
			// no lanza nada **teniendo parte de lo que hace falta**.
			if err := cmd.Flags().Set("var", "CF_TOKEN=Cloudflare API"); err != nil {
				t.Fatal(err)
			}
			err := cmd.RunE(cmd, []string{"sh", "-c", "touch " + testigo})
			if err == nil {
				t.Fatal("ha lanzado el programa sin tener todos los secretos")
			}
			if !strings.Contains(err.Error(), caso.dice) {
				t.Errorf("el error no dice qué pasa: %v", err)
			}
			if !strings.Contains(err.Error(), "X") {
				t.Errorf("el error no dice de qué variable se queja: %v", err)
			}
			if _, err := os.Stat(testigo); err == nil {
				t.Fatal("el programa se ha lanzado")
			}
		})
	}
}

// Los nombres de variable se comprueban aquí, porque un nombre mal puesto **no da
// error en el hijo, da una variable que no está**: el programa falla por otra cosa
// y en otro sitio.
func TestLoQueSePideSeComprueba(t *testing.T) {
	for _, caso := range []struct{ pide, dice string }{
		{"CF_TOKEN", "la forma VARIABLE=búsqueda"},
		{"=algo", "Falta el nombre"},
		{"CF-TOKEN=algo", "no es un nombre de variable"},
		{"2CF=algo", "no es un nombre de variable"},
		{"CF_TOKEN=", "no dice qué entrada"},
	} {
		if _, err := loQueSePide([]string{caso.pide}); err == nil {
			t.Errorf("«%s» ha pasado por bueno", caso.pide)
		} else if !strings.Contains(err.Error(), caso.dice) {
			t.Errorf("«%s» se queja de otra cosa: %v", caso.pide, err)
		}
	}
	// Sin ninguna no hay nada que hacer, y lo dice en vez de lanzar el programa a
	// secas: quien escribe esto espera que lleve un secreto dentro.
	if _, err := loQueSePide(nil); err == nil {
		t.Error("sin ninguna variable no se ha quejado")
	}
	// La misma dos veces solo puede valer una cosa.
	if _, err := loQueSePide([]string{"X=uno", "X=otro"}); err == nil {
		t.Error("ha aceptado la misma variable dos veces")
	}
	// Y lo que está bien pasa, con los espacios recortados.
	ok, err := loQueSePide([]string{" CF_TOKEN = Cloudflare API ", "_X2=otra"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ok) != 2 || ok[0] != [2]string{"CF_TOKEN", "Cloudflare API"} || ok[1][0] != "_X2" {
		t.Fatalf("lo que ha salido es %+v", ok)
	}
}

// **Cuando el programa arranca, la bóveda ya está cerrada.**
//
// Es lo que dice el comentario de `conLosSecretos` y lo único que hay ahí que no se
// puede ver desde fuera del proceso: un despliegue puede tardar media hora, y
// dejarla abierta es justo lo que el reloj del bloqueo existe para que no pase. Para
// eso `lanzar` entra por una costura.
func TestCuandoArrancaElProgramaLaBovedaYaEstaCerrada(t *testing.T) {
	ruta, _ := unaBovedaConSecretos(t)
	b, err := boveda.Abrir(ruta, "una contraseña maestra bien larga")
	if err != nil {
		t.Fatal(err)
	}
	pedidas, err := loQueSePide([]string{"CF_TOKEN=Cloudflare API"})
	if err != nil {
		t.Fatal(err)
	}

	lanzado := false
	err = conLosSecretos(b, pedidas, []string{"true"}, func(string) {},
		func(entorno []string, _ []string) error {
			lanzado = true
			if b.Abierta() {
				t.Error("el programa arranca con la bóveda todavía abierta")
			}
			// Y el secreto va con él: cerrar antes no puede dejarlo sin nada.
			if len(entorno) != 1 || !strings.Contains(entorno[0], elSecretoDePrueba) {
				t.Errorf("el entorno que recibe es %v", len(entorno))
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if !lanzado {
		t.Fatal("no ha lanzado nada")
	}
}

// Y si una búsqueda no encaja, **tampoco se queda abierta**: ahí no se lanza nada y
// nadie va a volver a esa bóveda.
func TestSiNoSeLanzaNadaLaBovedaTampocoSeQuedaAbierta(t *testing.T) {
	ruta, _ := unaBovedaConSecretos(t)
	b, err := boveda.Abrir(ruta, "una contraseña maestra bien larga")
	if err != nil {
		t.Fatal(err)
	}
	pedidas, _ := loQueSePide([]string{"X=esto no está"})
	if err := conLosSecretos(b, pedidas, []string{"true"}, func(string) {},
		func([]string, []string) error {
			t.Fatal("ha lanzado el programa")
			return nil
		}); err == nil {
		t.Fatal("no ha dado error")
	}
	if b.Abierta() {
		t.Error("la bóveda se queda abierta cuando no se lanza nada")
	}
}

// **Si no se ha podido apuntar, se dice.** Lo pide `registro.go` con esas palabras:
// una medida de seguridad que no se aplica en silencio es peor que no tenerla.
//
// Y no es un caso raro: pasa con una bóveda compartida de solo ver, que es la clase
// de bóveda que un servidor va a tener. Aquí se provoca dejando la carpeta sin
// permiso de escritura, que es lo que ve `escritura.Atomica` al ir a renombrar.
func TestSiNoSePuedeApuntarSeDice(t *testing.T) {
	ruta, _ := unaBovedaConSecretos(t)
	b, err := boveda.Abrir(ruta, "una contraseña maestra bien larga")
	if err != nil {
		t.Fatal(err)
	}
	carpeta := filepath.Dir(ruta)
	if err := os.Chmod(carpeta, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(carpeta, 0o700) })

	pedidas, _ := loQueSePide([]string{"CF_TOKEN=Cloudflare API"})
	var dicho []string
	lanzado := false
	err = conLosSecretos(b, pedidas, []string{"true"},
		func(frase string) { dicho = append(dicho, frase) },
		func([]string, []string) error { lanzado = true; return nil })
	if err != nil {
		t.Fatal(err)
	}
	// **El programa se lanza igual**: lo que se estaba haciendo ya se ha hecho, y no
	// poder apuntarlo no puede deshacerlo ni esconderlo.
	if !lanzado {
		t.Error("no ha lanzado el programa por no poder apuntar")
	}
	if len(dicho) != 1 || !strings.Contains(dicho[0], "registro") {
		t.Errorf("no ha dicho que no ha quedado apuntado: %v", dicho)
	}
}
