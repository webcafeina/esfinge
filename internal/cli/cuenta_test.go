package cli

// La cuenta de un servidor, contra **el servidor de cuentas de verdad** levantado
// en local por `herramientas/con-servidor.sh` (ADR 0057).
//
// Un `go test` suelto se las salta. Las corre `make comprobar` y la puerta de
// publicación, por la razón de siempre: un servidor de mentira escrito en Go
// probaría el servidor de mentira.
//
// **Y recorren el camino de verdad, prompt incluido.** El código del correo se le
// escribe por la entrada estándar mientras el comando espera, que es lo que hace
// una persona: lo de ponerle el estado a mano —un testigo, una sesión— es
// exactamente lo que dejó sin probar el emparejamiento del servidor MCP.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cuenta"
	"github.com/webcafeina/esfinge/internal/llavero"
)

const laMaestraDelServidor = "una maestra larga para las pruebas del servidor"

func elServidorDeCuentas(t *testing.T) string {
	t.Helper()
	raiz := os.Getenv("ESFINGE_SERVIDOR_PRUEBAS")
	if raiz == "" {
		t.Skip("sin servidor de cuentas: se corren con herramientas/con-servidor.sh (make comprobar)")
	}
	viejo := raizDeCuentas
	raizDeCuentas = raiz
	t.Cleanup(func() { raizDeCuentas = viejo })
	return raiz
}

// unCorreoNuevo, porque el buzón de pruebas **acumula mensajes y solo vale el más
// reciente**: con una dirección por prueba no hay forma de coger el de otra.
func unCorreoNuevo(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("servidor-%d@ejemplo.com", time.Now().UnixNano())
}

// elCodigoQueLlegue espera **al contenido** del buzón, no a un estado: esperar a que
// algo diga «ya» es lo que costó creer que una bóveda compartida no se sincronizaba
// cuando se sincronizaba perfectamente.
func elCodigoQueLlegue(t *testing.T, raiz, correo string) string {
	t.Helper()
	limite := time.Now().Add(20 * time.Second)
	for time.Now().Before(limite) {
		if c := miraElBuzon(t, raiz, correo); c != "" {
			return c
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no ha llegado ningún código a %s", correo)
	return ""
}

func miraElBuzon(t *testing.T, raiz, correo string) string {
	t.Helper()
	resp, err := http.Get(raiz + "/_pruebas/buzon?correo=" + correo)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var r struct {
		Mensajes []struct{ Cuerpo string } `json:"mensajes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil || len(r.Mensajes) == 0 {
		return ""
	}
	m := regexp.MustCompile(`(?m)^\s+(\d{6})$`).FindStringSubmatch(r.Mensajes[0].Cuerpo)
	if m == nil {
		return ""
	}
	return m[1]
}

// contestandoElCodigo corre el comando y, mientras espera, le escribe por la entrada
// estándar el código que llegue al correo. Devuelve lo que el comando saque por la
// salida estándar.
func contestandoElCodigo(t *testing.T, raiz, correo string, correr func() error) string {
	t.Helper()
	leer, escribir, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	leerSalida, escribirSalida, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	antesIn, antesOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = leer, escribirSalida
	recogido := make(chan string, 1)
	go func() {
		todo, _ := io.ReadAll(leerSalida)
		recogido <- string(todo)
	}()

	hecho := make(chan error, 1)
	go func() { hecho <- correr() }()

	// El código solo existe después de que el comando lo haya pedido, así que se
	// espera a que llegue al buzón y entonces se teclea.
	codigo := elCodigoQueLlegue(t, raiz, correo)
	fmt.Fprintln(escribir, codigo)
	err = <-hecho

	os.Stdin, os.Stdout = antesIn, antesOut
	_ = escribir.Close()
	_ = leer.Close()
	_ = escribirSalida.Close()
	salida := <-recogido
	_ = leerSalida.Close()
	if err != nil {
		t.Fatalf("el comando ha fallado: %v", err)
	}
	return salida
}

// un servidor recién montado: cuenta nueva, bóveda nueva y la llave de este equipo.
func unServidorMontado(t *testing.T, raiz string) (ruta, correo, conLaMaestra string) {
	t.Helper()
	dir := t.TempDir()
	ruta = filepath.Join(dir, "boveda.esfinge")
	correo = unCorreoNuevo(t)
	conLaMaestra = filepath.Join(dir, "maestra.txt")
	if err := os.WriteFile(conLaMaestra, []byte(laMaestraDelServidor), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(llavero.VariableDeLaLlave, filepath.Join(t.TempDir(), "llave"))

	o := &opciones{tema: "oscuro", silencio: true}
	ob := &opcionesBoveda{ruta: ruta}
	ob.clave.Fichero = conLaMaestra

	alta := comandoServidorAlta(o, ob)
	if err := alta.Flags().Set("correo", correo); err != nil {
		t.Fatal(err)
	}
	recuperacion := strings.TrimSpace(contestandoElCodigo(t, raiz, correo, func() error {
		return alta.RunE(alta, nil)
	}))
	// La clave de recuperación sale **una sola vez y por la salida estándar**, para
	// poder redirigirla sin llevarse los mensajes de cortesía por delante.
	if !strings.HasPrefix(recuperacion, "ESF-") || len(strings.Split(recuperacion, "-")) < 5 {
		t.Fatalf("la clave de recuperación no parece una: %q", recuperacion)
	}
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("no ha quedado bóveda en %s: %v", ruta, err)
	}
	return ruta, correo, conLaMaestra
}

// **Un servidor se da de alta, entra, abre sin contraseña y se sincroniza.**
//
// Es la tubería entera de la ADR 0057 por el camino de verdad: las piezas en verde
// una a una no dicen que estén conectadas.
func TestUnServidorSeDaDeAltaAbreSinContrasenaYSeSincroniza(t *testing.T) {
	raiz := elServidorDeCuentas(t)
	ruta, correo, conLaMaestra := unServidorMontado(t, raiz)

	o := &opciones{tema: "oscuro", silencio: true}
	conClave := &opcionesBoveda{ruta: ruta}
	conClave.clave.Fichero = conLaMaestra
	sinClave := &opcionesBoveda{ruta: ruta}

	// Lo que el alta ha dejado escrito: la cuenta, su clave de acceso sellada y la
	// confianza, que es lo que permite volver a entrar sin que nadie lea un correo.
	d := leerDatosDelServidor(ruta)
	if d.Correo != correo || d.Acceso == "" || d.Sesion == "" || d.Confianza == "" {
		t.Fatalf("los datos del servidor están a medias: %+v", d)
	}
	// **Y van sellados**, no en claro: un `ESF1.` delante es lo que dice que cuelgan
	// de la llave de este equipo.
	for nombre, v := range map[string]string{"sesion": d.Sesion, "acceso": d.Acceso, "confianza": d.Confianza} {
		if !strings.HasPrefix(v, "ESF1.") {
			t.Errorf("%s no está sellado: %q", nombre, v)
		}
	}

	// Sin activar la llave del equipo, esto no abre nada solo.
	if b, err := abrirBoveda(o, sinClave); err == nil {
		b.Cerrar()
		t.Fatal("la bóveda del servidor se abre sin contraseña sin haberlo activado")
	}
	if err := comandoServidorActivar(o, conClave).RunE(nil, nil); err != nil {
		t.Fatal(err)
	}

	// Y ahora: abre sin contraseña **y de paso se pone al día**, que es lo que hace
	// que una credencial cambiada en el Mac llegue aquí.
	b, err := abrirBoveda(o, sinClave)
	if err != nil {
		t.Fatalf("no abre con la llave de este equipo: %v", err)
	}
	if err := b.Poner(boveda.Entrada{Tipo: boveda.TipoCredencial, Titulo: "Cloudflare API", Secreto: "el testigo"}); err != nil {
		t.Fatal(err)
	}
	r, err := unaPasada(b)
	if err != nil {
		t.Fatalf("no se ha podido sincronizar: %v", err)
	}
	if !r.Subio {
		t.Error("la pasada no ha subido lo de aquí")
	}
	b.Cerrar()

	// Y lo subido está en el servidor de verdad, cifrado: se baja con el mismo
	// cliente y se abre con la maestra.
	cli := cuenta.Nuevo(raiz)
	token := elTestigoDe(t, ruta)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	datos, version, _, err := cli.Bajar(ctx, token, 0)
	if err != nil {
		t.Fatalf("no se ha podido bajar lo que subió: %v", err)
	}
	if version == 0 || len(datos) == 0 {
		t.Fatalf("el servidor dice versión %d y %d bytes", version, len(datos))
	}
	otroEquipo, err := boveda.AbrirEnMemoria(datos, laMaestraDelServidor)
	if err != nil {
		t.Fatalf("lo que hay en el servidor no abre: %v", err)
	}
	if e := otroEquipo.Buscar("Cloudflare"); len(e) != 1 {
		t.Fatalf("en el servidor hay %d entradas", len(e))
	}
	// **Y la llave de este equipo no ha viajado**: es de un equipo y de uno solo.
	if strings.Contains(string(datos), boveda.RanuraDelServidor) {
		t.Fatal("la ranura del servidor ha subido a la cuenta")
	}
}

// **La sesión caduca, y el servidor vuelve a entrar él solo.**
//
// Es la razón entera de guardar la clave de acceso, y lo que se comprueba aquí es
// que el camino existe: se tira la sesión guardada y la siguiente pasada funciona
// igual, sin que nadie teclee nada ni lea ningún correo.
//
// La sesión no se puede caducar de verdad desde aquí —son treinta días— así que se
// hace lo que el servidor haría: dejarla inservible. Lo que se ejercita es
// `conLaSesion`, que es quien tiene que notarlo.
func TestSiLaSesionNoValeElServidorVuelveAEntrarSolo(t *testing.T) {
	raiz := elServidorDeCuentas(t)
	ruta, _, conLaMaestra := unServidorMontado(t, raiz)

	o := &opciones{tema: "oscuro", silencio: true}
	conClave := &opcionesBoveda{ruta: ruta}
	conClave.clave.Fichero = conLaMaestra
	if err := comandoServidorActivar(o, conClave).RunE(nil, nil); err != nil {
		t.Fatal(err)
	}

	// La sesión que había, cambiada por una que el servidor no conoce.
	b, err := abrirLaBoveda(o, &opcionesBoveda{ruta: ruta})
	if err != nil {
		t.Fatal(err)
	}
	d := leerDatosDelServidor(ruta)
	rota, err := b.SellarSecreto([]byte("este-testigo-no-existe-en-el-servidor"))
	if err != nil {
		t.Fatal(err)
	}
	d.Sesion = rota
	if err := guardarDatosDelServidor(ruta, d); err != nil {
		t.Fatal(err)
	}

	if _, err := unaPasada(b); err != nil {
		t.Fatalf("con la sesión inservible no ha podido volver a entrar: %v", err)
	}
	b.Cerrar()

	// Y la sesión guardada es otra: ha entrado de verdad, no ha seguido con la mala.
	if leerDatosDelServidor(ruta).Sesion == rota {
		t.Error("ha sincronizado y no ha guardado la sesión nueva")
	}
}

func elTestigoDe(t *testing.T, ruta string) string {
	t.Helper()
	b, err := boveda.Abrir(ruta, laMaestraDelServidor)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	claro, err := b.AbrirSecreto(leerDatosDelServidor(ruta).Sesion)
	if err != nil {
		t.Fatal(err)
	}
	return string(claro)
}
