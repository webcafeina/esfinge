package cuenta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"
)

// ------------------------------------------------------------------ sin servidor

func TestLaClaveDeAccesoNoSeDerivaConPocoCoste(t *testing.T) {
	sal := bytes.Repeat([]byte{1}, 16)
	for _, p := range []Parametros{
		{Memoria: 8 * 1024, Pasadas: 1, Paralelismo: 1},  // el de sellar el cuerpo: no vale aquí
		{Memoria: 64 * 1024, Pasadas: 2, Paralelismo: 4}, // una pasada de menos
		{Memoria: 32 * 1024, Pasadas: 3, Paralelismo: 4}, // media memoria
		{Memoria: 64 * 1024, Pasadas: 3, Paralelismo: 0}, // sin hilos
	} {
		if _, err := DerivarAcceso("maestra", sal, p); !errors.Is(err, ErrCosteBajo) {
			t.Errorf("deriva con %+v: %v", p, err)
		}
	}
	// Ni 4 GiB ni medio: el tope es el mismo que al abrir un sobre, 256 MiB.
	for _, m := range []uint32{512 * 1024, 4 * 1024 * 1024} {
		if _, err := DerivarAcceso("maestra", sal, Parametros{Memoria: m, Pasadas: 3, Paralelismo: 4}); err == nil {
			t.Errorf("deriva pidiendo %d KiB", m)
		}
	}
}

func TestLaClaveDeAccesoEsEstableYDependeDeTodo(t *testing.T) {
	sal := bytes.Repeat([]byte{7}, 16)
	una, err := DerivarAcceso("maestra", sal, PorDefecto)
	if err != nil {
		t.Fatal(err)
	}
	otra, _ := DerivarAcceso("maestra", sal, PorDefecto)
	if !bytes.Equal(una, otra) || len(una) != 32 {
		t.Fatal("la misma contraseña con la misma sal no da la misma clave")
	}
	conOtraSal, _ := DerivarAcceso("maestra", bytes.Repeat([]byte{8}, 16), PorDefecto)
	conOtraMaestra, _ := DerivarAcceso("maestro", sal, PorDefecto)
	if bytes.Equal(una, conOtraSal) || bytes.Equal(una, conOtraMaestra) {
		t.Fatal("la clave no depende de la sal o de la contraseña")
	}
}

func TestElCorreoSeNormalizaComoEnElServidor(t *testing.T) {
	for entra, sale := range map[string]string{
		"  Ana@Ejemplo.COM ":    "ana@ejemplo.com",
		"ana+esfinge@gmail.com": "ana+esfinge@gmail.com", // el «+» se queda: no es cosa nuestra
		"a.n.a@gmail.com":       "a.n.a@gmail.com",
	} {
		if got, err := NormalizarCorreo(entra); err != nil || got != sale {
			t.Errorf("%q → %q, %v", entra, got, err)
		}
	}
	for _, malo := range []string{"", "ana", "ana@", "@ejemplo.com", "ana @ejemplo.com"} {
		if _, err := NormalizarCorreo(malo); err == nil {
			t.Errorf("acepta %q", malo)
		}
	}
}

func TestLaEtiquetaDebilTambienSeLee(t *testing.T) {
	for v, n := range map[string]int64{`"17"`: 17, `W/"17"`: 17, `17`: 17, ` "3" `: 3} {
		if got, ok := LeerEtiqueta(v); !ok || got != n {
			t.Errorf("%s → %d, %v", v, got, ok)
		}
	}
	for _, v := range []string{"", `"abc"`, `W/"-1"`} {
		if _, ok := LeerEtiqueta(v); ok {
			t.Errorf("lee %q", v)
		}
	}
}

// ------------------------------------------------------------------ contra el servidor de verdad
//
// Solo con ESFINGE_SERVIDOR_PRUEBAS, que pone herramientas/con-servidor.sh al
// levantar el Worker en local. `make comprobar` lo hace; un `go test` suelto se
// las salta.

func servidor(t *testing.T) *Cliente {
	t.Helper()
	raiz := os.Getenv("ESFINGE_SERVIDOR_PRUEBAS")
	if raiz == "" {
		t.Skip("sin servidor de cuentas: se corren con herramientas/con-servidor.sh (make comprobar)")
	}
	return Nuevo(raiz)
}

var ctx = context.Background()

// codigo lee el último código que ha llegado a un buzón del servidor de pruebas.
func codigo(t *testing.T, c *Cliente, correo string) string {
	t.Helper()
	resp, err := http.Get(c.raiz + "/_pruebas/buzon?correo=" + correo)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r struct {
		Mensajes []struct{ Cuerpo string } `json:"mensajes"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&r)
	if len(r.Mensajes) == 0 {
		t.Fatalf("no ha llegado nada a %s", correo)
	}
	m := regexp.MustCompile(`(?m)^\s+(\d{6})$`).FindStringSubmatch(r.Mensajes[0].Cuerpo)
	if m == nil {
		t.Fatalf("el último correo a %s no trae código", correo)
	}
	return m[1]
}

func correoNuevo(t *testing.T) string {
	return fmt.Sprintf("go-%d@ejemplo.com", time.Now().UnixNano())
}

// alta hace lo que hará Esfinge al crear la cuenta, con la derivación de verdad.
func alta(t *testing.T, c *Cliente, correo, maestra string, posesion []byte) Sesion {
	t.Helper()
	if err := c.EmpezarAlta(ctx, correo); err != nil {
		t.Fatal(err)
	}
	sal := bytes.Repeat([]byte{3}, 16)
	clave, err := DerivarAcceso(maestra, sal, PorDefecto)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.TerminarAlta(ctx, Alta{
		Correo: correo, Codigo: codigo(t, c, correo), Sal: sal, Argon2: PorDefecto,
		ClaveDeAcceso: clave, Posesion: posesion, Dispositivo: "Pruebas de Go", Confiar: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestServidorEntrarConLaClaveDerivada(t *testing.T) {
	c := servidor(t)
	if s, err := c.Salud(ctx); err != nil || s.Protocolo != 1 {
		t.Fatalf("salud: %+v, %v", s, err)
	}
	correo := correoNuevo(t)
	s := alta(t, c, correo, "la maestra de las pruebas", bytes.Repeat([]byte{9}, 32))

	// Otro equipo: pre-entrada, derivar, entrar con código.
	pre, err := c.Prelogin(ctx, correo)
	if err != nil {
		t.Fatal(err)
	}
	clave, _ := DerivarAcceso("la maestra de las pruebas", pre.Sal, pre.Argon2)
	sesion, reto, err := c.Entrar(ctx, correo, clave, "Otro equipo", "")
	if err != nil || sesion != nil || reto == "" {
		t.Fatalf("entrar desde un equipo nuevo tiene que pedir código: %v %v %q", sesion, err, reto)
	}
	otra, err := c.ConfirmarEntrada(ctx, reto, codigo(t, c, correo), false)
	if err != nil || otra.Token == "" {
		t.Fatalf("confirmar: %v", err)
	}
	// Con el testigo de confianza del primero, sin código.
	directa, _, err := c.Entrar(ctx, correo, clave, "Pruebas de Go", s.Confianza)
	if err != nil || directa == nil {
		t.Fatalf("un equipo de confianza pide código: %v", err)
	}
	// Con otra contraseña, no; y el mensaje se puede enseñar tal cual.
	mala, _ := DerivarAcceso("otra", pre.Sal, pre.Argon2)
	_, _, err = c.Entrar(ctx, correo, mala, "x", "")
	var e *ErrorDelServidor
	if !errors.As(err, &e) || e.Estado != 401 || e.Mensaje == "" {
		t.Fatalf("una contraseña mala: %v", err)
	}
	// Cerrar la sesión la deja sin valor.
	if err := c.CerrarSesion(ctx, otra.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Equipos(ctx, otra.Token); !SesionCaducada(err) {
		t.Fatalf("una sesión cerrada sigue valiendo: %v", err)
	}
}

func TestServidorSubirYBajarConVersiones(t *testing.T) {
	c := servidor(t)
	s := alta(t, c, correoNuevo(t), "maestra", bytes.Repeat([]byte{1}, 32))
	if _, _, _, err := c.Bajar(ctx, s.Token, 0); !errors.Is(err, ErrSinBoveda) {
		t.Fatalf("sin bóveda: %v", err)
	}
	doc := []byte(`{"esfinge":"bóveda","formato":1,"id":"0123456789abcdef0123456789abcdef","sobres":[],"cuerpo":"ESF1.x"}`)
	v, err := c.Subir(ctx, s.Token, 0, doc)
	if err != nil || v != 1 {
		t.Fatalf("subir: %d %v", v, err)
	}
	datos, version, cambio, err := c.Bajar(ctx, s.Token, 0)
	if err != nil || !cambio || version != 1 || !bytes.Equal(datos, doc) {
		t.Fatalf("bajar: %d %v %v", version, cambio, err)
	}
	if _, _, cambio, err := c.Bajar(ctx, s.Token, 1); err != nil || cambio {
		t.Fatalf("sin cambios tiene que contestar que no hay nada: %v %v", cambio, err)
	}
	_, err = c.Subir(ctx, s.Token, 0, doc)
	if actual, ok := Conflicto(err); !ok || actual != 1 {
		t.Fatalf("subir sobre una versión vieja: %v", err)
	}
	vs, err := c.Versiones(ctx, s.Token)
	if err != nil || len(vs) != 1 || vs[0].Version != 1 {
		t.Fatalf("versiones: %+v %v", vs, err)
	}
	una, err := c.UnaVersion(ctx, s.Token, 1)
	if err != nil || !bytes.Equal(una, doc) {
		t.Fatalf("una versión: %v", err)
	}
}

func TestServidorRecuperarCambiarClaveYBorrar(t *testing.T) {
	c := servidor(t)
	correo := correoNuevo(t)
	posesion := bytes.Repeat([]byte{5}, 32)
	s := alta(t, c, correo, "maestra vieja", posesion)
	doc := []byte(`{"esfinge":"bóveda","formato":1,"id":"0123456789abcdef0123456789abcdef","sobres":[{"tipo":"recuperacion","contenedor":"ESF1.rec","codificacion":"crockford32-v1"}],"cuerpo":"ESF1.x"}`)
	if _, err := c.Subir(ctx, s.Token, 0, doc); err != nil {
		t.Fatal(err)
	}

	if err := c.EmpezarRecuperacion(ctx, correo); err != nil {
		t.Fatal(err)
	}
	reto, sobre, err := c.ComprobarRecuperacion(ctx, correo, codigo(t, c, correo))
	if err != nil || sobre.Contenedor != "ESF1.rec" || sobre.Codificacion != "crockford32-v1" {
		t.Fatalf("recuperar: %+v %v", sobre, err)
	}
	restringida, err := c.TerminarRecuperacion(ctx, reto, posesion, "Equipo recuperado")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Equipos(ctx, restringida.Token); err == nil {
		t.Fatal("la sesión de recuperación sirve para algo más que cambiar la contraseña")
	}

	sal := bytes.Repeat([]byte{4}, 16)
	nueva, _ := DerivarAcceso("maestra nueva", sal, PorDefecto)
	v, normal, err := c.CambiarClave(ctx, restringida.Token, CambioDeClave{
		Posesion: posesion, Sal: sal, Argon2: PorDefecto, ClaveDeAcceso: nueva, Version: 1, Documento: doc,
	})
	if err != nil || v != 2 || normal == "" {
		t.Fatalf("cambiar la clave: %d %q %v", v, normal, err)
	}
	pre, _ := c.Prelogin(ctx, correo)
	if !bytes.Equal(pre.Sal, sal) {
		t.Fatal("la sal no ha cambiado con la contraseña")
	}
	if _, err := c.Equipos(ctx, s.Token); !SesionCaducada(err) {
		t.Fatal("la sesión del otro equipo sigue viva tras cambiar la contraseña")
	}

	reto, err = c.PedirBorrado(ctx, normal)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.BorrarCuenta(ctx, normal, nueva, reto, codigo(t, c, correo)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Equipos(ctx, normal); !SesionCaducada(err) {
		t.Fatal("tras borrar la cuenta, la sesión sigue")
	}
}

// **Con ESFINGE_SIN_RED no sale ni una petición**, también hacia la cuenta: es la
// tercera salida a la red y la regla de internal/red es que las apaga todas.
//
// Se prueba en un proceso aparte porque `red.SinRed` lee la variable una sola vez:
// dentro de este proceso ya la ha leído otra prueba.
func TestConLaRedApagadaNoSaleNada(t *testing.T) {
	if os.Getenv("ESFINGE_SIN_RED") != "" {
		llamadas := 0
		falso := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			llamadas++
		}))
		defer falso.Close()
		c := Nuevo(falso.URL)
		if _, err := c.Salud(ctx); !errors.Is(err, ErrSinRed) {
			t.Fatalf("con la red apagada: %v", err)
		}
		if _, _, _, err := c.Bajar(ctx, "s1.x.y", 0); !errors.Is(err, ErrSinRed) {
			t.Fatalf("bajar con la red apagada: %v", err)
		}
		if llamadas != 0 {
			t.Fatalf("han salido %d peticiones con la red apagada", llamadas)
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestConLaRedApagadaNoSaleNada$", "-test.count=1")
	cmd.Env = append(os.Environ(), "ESFINGE_SIN_RED=1")
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, salida)
	}
}
