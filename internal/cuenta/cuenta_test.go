package cuenta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

// ------------------------------------------------------------------ sin servidor

// **Un error que no viene en nuestro formato tiene que decir lo que sí vino.**
//
// Sale de una caída de la puerta del 2026-10-02: la prueba de la cuenta falló con
// «El servidor de cuentas ha contestado 500» y de ahí no se podía sacar nada —tres
// pasadas completas para averiguar que no se reproducía—. Un 500 del Worker, por una
// excepción o por un secreto que falta, no trae `{"error": …}`, y sin esto el cuerpo
// se tiraba entero.
func TestUnErrorQueNoEsNuestroDiceLoQueDijoElServidor(t *testing.T) {
	respuesta := func(codigo int, cuerpo string) *http.Response {
		return &http.Response{StatusCode: codigo, Body: io.NopCloser(strings.NewReader(cuerpo))}
	}

	// Lo nuestro se sigue leyendo igual, y **sin añadirle nada**: ese texto se le
	// enseña a una persona.
	err := errorDe(respuesta(409, `{"error":"Esa bóveda no es la de esta cuenta","version":17}`))
	var e *ErrorDelServidor
	if !errors.As(err, &e) {
		t.Fatalf("no es un ErrorDelServidor: %v", err)
	}
	if e.Mensaje != "Esa bóveda no es la de esta cuenta" || e.Version != 17 {
		t.Errorf("el error nuestro ha cambiado: %+v", e)
	}

	// Y lo que no es nuestro: el código **y lo que dijo**.
	err = errorDe(respuesta(500, "Error: Cannot read properties of undefined (reading 'pimienta')"))
	if !errors.As(err, &e) {
		t.Fatalf("no es un ErrorDelServidor: %v", err)
	}
	if !strings.Contains(e.Mensaje, "500") {
		t.Errorf("no dice el código: %q", e.Mensaje)
	}
	if !strings.Contains(e.Mensaje, "pimienta") {
		t.Errorf("no dice lo que contestó el servidor, que es lo único que había: %q", e.Mensaje)
	}

	// Un cuerpo vacío no deja dos puntos colgando.
	err = errorDe(respuesta(502, ""))
	errors.As(err, &e)
	if strings.HasSuffix(e.Mensaje, ":") || strings.Contains(e.Mensaje, ": ") {
		t.Errorf("con el cuerpo vacío el mensaje queda a medias: %q", e.Mensaje)
	}

	// Y un cuerpo largo se recorta, que un error no es un volcado.
	err = errorDe(respuesta(500, strings.Repeat("x", 5000)))
	errors.As(err, &e)
	if len(e.Mensaje) > 400 {
		t.Errorf("el mensaje mide %d caracteres", len(e.Mensaje))
	}
}

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

// bovedaDePrueba es el documento mínimo que el servidor acepta: lo que comprueba es
// la cabecera y los sobres, no el contenido, que no puede leer.
func bovedaDePrueba(id, relleno string) []byte {
	return []byte(`{"esfinge":"bóveda","formato":1,"id":"` + id + `","sobres":[],"cuerpo":"ESF1.` + relleno + `x"}`)
}

// **Dar acceso a una bóveda de proyecto, con dos cuentas de verdad** (ADR 0052).
//
// Es la prueba que de verdad cierra el servidor: todo lo demás está probado por
// piezas, y esto es un protocolo entre dos partes. Lo que recorre es el camino
// entero —dar el acceso, bajar, subir, y que al quitarlo se cierre la puerta—
// contra el Worker levantado en local, no contra un doble.
func TestServidorDarAccesoAUnProyecto(t *testing.T) {
	c := servidor(t)
	const ref = "a1b2c3d4e5f60718"
	const titular = "1111222233334444"
	doc := bovedaDePrueba("0123456789abcdef0123456789abcdef", "")

	duena := alta(t, c, correoNuevo(t), "la maestra de la dueña", bytes.Repeat([]byte{9}, 32))
	correoDeAna := correoNuevo(t)
	ana := alta(t, c, correoDeAna, "la maestra de Ana", bytes.Repeat([]byte{8}, 32))

	if _, err := c.SubirA(ctx, duena.Token, ref, 0, doc); err != nil {
		t.Fatal(err)
	}

	// **Antes de darle acceso, Ana no entra.** Saber la dirección de una bóveda no
	// es tenerla.
	if _, _, _, err := c.BajarCompartida(ctx, ana.Token, duena.Cuenta, ref, 0); err == nil {
		t.Fatal("Ana baja una bóveda a la que no tiene acceso")
	}

	if err := c.DarAcceso(ctx, duena.Token, ref, correoDeAna, "editar", titular); err != nil {
		t.Fatal(err)
	}

	datos, version, _, err := c.BajarCompartida(ctx, ana.Token, duena.Cuenta, ref, 0)
	if err != nil {
		t.Fatalf("Ana no baja la bóveda que le han compartido: %v", err)
	}
	if !bytes.Equal(datos, doc) {
		t.Fatal("lo que baja Ana no es lo que subió la dueña")
	}

	// Y escribe, que es lo que la hace viva: lo que sube lo ve la dueña.
	otro := bovedaDePrueba("0123456789abcdef0123456789abcdef", "lo de Ana")
	if _, err := c.SubirACompartida(ctx, ana.Token, duena.Cuenta, ref, version, otro); err != nil {
		t.Fatalf("Ana no puede escribir con permiso de editar: %v", err)
	}
	vuelta, _, _, err := c.BajarDe(ctx, duena.Token, ref, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(vuelta, otro) {
		t.Fatal("lo que escribió Ana no le ha llegado a la dueña")
	}

	// La lista de quién tiene acceso, **por titular y sin cuentas**.
	miembros, err := c.Miembros(ctx, duena.Token, ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(miembros) != 1 || miembros[0].Titular != titular || miembros[0].Permiso != "editar" {
		t.Fatalf("la lista de miembros es %+v", miembros)
	}

	// Y quitarle el acceso cierra la puerta **en la siguiente petición**, sin esperar
	// a que nadie sincronice nada.
	if err := c.QuitarAcceso(ctx, duena.Token, ref, titular); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := c.BajarCompartida(ctx, ana.Token, duena.Cuenta, ref, 0); err == nil {
		t.Fatal("se le ha quitado el acceso a Ana y sigue bajando la bóveda")
	}
}

// Y el permiso de solo ver, que **lo impone el servidor y no el cifrado**: Ana tiene
// con qué descifrarla, así que puede fabricar un documento válido. Lo que no puede es
// dejarlo aquí.
func TestServidorSoloVerNoSube(t *testing.T) {
	c := servidor(t)
	const ref = "b1b2c3d4e5f60718"
	doc := bovedaDePrueba("1123456789abcdef0123456789abcdef", "")

	duena := alta(t, c, correoNuevo(t), "la maestra de la dueña", bytes.Repeat([]byte{9}, 32))
	correoDeAna := correoNuevo(t)
	ana := alta(t, c, correoDeAna, "la maestra de Ana", bytes.Repeat([]byte{8}, 32))

	if _, err := c.SubirA(ctx, duena.Token, ref, 0, doc); err != nil {
		t.Fatal(err)
	}
	if err := c.DarAcceso(ctx, duena.Token, ref, correoDeAna, "ver", "2222333344445555"); err != nil {
		t.Fatal(err)
	}

	_, version, _, err := c.BajarCompartida(ctx, ana.Token, duena.Cuenta, ref, 0)
	if err != nil {
		t.Fatalf("quien solo puede ver tiene que poder bajarla: %v", err)
	}
	if _, err := c.SubirACompartida(ctx, ana.Token, duena.Cuenta, ref, version, doc); err == nil {
		t.Fatal("quien solo puede ver ha subido")
	}
}

// **Quitarle el acceso a alguien es un 403, no un 401**, y la diferencia no es de
// estilo: el 401 cierra la bóveda de quien lo recibe porque su sesión se perdió, y
// aquí su sesión está perfectamente. Confundirlos cerraría la bóveda personal de Ana
// porque otra persona le quitó el acceso a la suya.
func TestServidorQuitarElAccesoNoEsUnaSesionCaducada(t *testing.T) {
	c := servidor(t)
	const ref = "c1b2c3d4e5f60718"
	doc := bovedaDePrueba("2123456789abcdef0123456789abcdef", "")

	duena := alta(t, c, correoNuevo(t), "la maestra de la dueña", bytes.Repeat([]byte{9}, 32))
	correoDeAna := correoNuevo(t)
	ana := alta(t, c, correoDeAna, "la maestra de Ana", bytes.Repeat([]byte{8}, 32))
	if _, err := c.SubirA(ctx, duena.Token, ref, 0, doc); err != nil {
		t.Fatal(err)
	}
	if err := c.DarAcceso(ctx, duena.Token, ref, correoDeAna, "editar", "3333444455556666"); err != nil {
		t.Fatal(err)
	}
	if err := c.QuitarAcceso(ctx, duena.Token, ref, "3333444455556666"); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := c.BajarCompartida(ctx, ana.Token, duena.Cuenta, ref, 0)
	if err == nil {
		t.Fatal("sigue bajando una bóveda a la que le han quitado el acceso")
	}
	if !SinAcceso(err) {
		t.Errorf("no se reconoce como «te han quitado el acceso»: %v", err)
	}
	if SesionCaducada(err) {
		t.Error("se lee como una sesión caducada, y eso cerraría la bóveda personal de quien lo recibe")
	}

	// Y lo que de verdad lo demuestra: **su propia bóveda sigue funcionando**.
	if _, err := c.SubirA(ctx, ana.Token, "", 0, bovedaDePrueba("3123456789abcdef0123456789abcdef", "")); err != nil {
		t.Fatalf("a Ana le han quitado el acceso a una bóveda ajena y la suya ha dejado de ir: %v", err)
	}
}
