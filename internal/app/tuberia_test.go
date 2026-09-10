package app

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/navegador"
)

// **La tubería entera, y es la prueba que faltaba.**
//
// Este proyecto ya se dio un golpe idéntico con los iconos: cada pieza tenía sus
// pruebas y **ninguna iba de un extremo al otro**, así que salieron rotos dos
// veces seguidas y las dos las vio el cliente a la primera. Aquí ha pasado lo
// mismo con el canal del navegador: el emparejamiento de dominios probado, el
// protocolo probado, el traductor probado, el servidor probado, los manifiestos
// probados… y **nada que fuera del proceso que lanza el navegador hasta la
// bóveda**.
//
// Lo que se ejercita aquí es exactamente eso: los bytes que manda un navegador
// —cuatro de longitud y el JSON detrás— entran por donde entran de verdad,
// cruzan el socket, llegan a la bóveda y vuelven enmarcados. Lo único que no hay
// es el navegador, que aquí no se puede tener.
func TestLaTuberiaEnteraDesdeElNavegador(t *testing.T) {
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Titulo: "Banco", Usuario: "yo@ejemplo.es", Secreto: "s3cr3t0",
		Sitios: []string{"https://banco.es/particulares"},
	}); err != nil {
		t.Fatal(err)
	}

	// El canal, encendido como lo enciende una persona en Ajustes.
	p := a.VerPreferencias()
	p.PuenteDelNavegador = true
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	defer a.pararCanal()
	if e := a.EstadoDelNavegador(); !e.Escuchando {
		t.Fatalf("el canal no está escuchando: %+v", e)
	}

	// El permiso, dado en la ventana.
	if err := a.PermitirNavegador(); err != nil {
		t.Fatal(err)
	}

	socket := navegador.RutaDelCanal()

	// Primer viaje: emparejarse, que es lo que hace la extensión al principio.
	r := unViaje(t, socket, `{"version":1,"que":"emparejar","quien":"Firefox"}`)
	if !r.OK || r.Testigo == "" {
		t.Fatalf("emparejar: %+v", r)
	}

	// Segundo: pedir las cuentas del sitio, con el testigo recién dado.
	r = unViaje(t, socket, `{"version":1,"que":"cuentas","testigo":"`+r.Testigo+
		`","origen":"https://banco.es/entrar"}`)
	if !r.OK {
		t.Fatalf("cuentas: %+v", r)
	}
	if len(r.Cuentas) != 1 || r.Cuentas[0].Titulo != "Banco" {
		t.Fatalf("cuentas: %+v", r.Cuentas)
	}
}

// unViaje hace lo que hace `esfinge-puente`: recibe los bytes del navegador por
// la entrada estándar y devuelve los suyos por la salida.
//
// **Con el enmarcado de verdad por los dos lados**, que es la mitad de lo que se
// quiere comprobar: un mensaje mal enmarcado no falla, se cuelga.
func unViaje(t *testing.T, socket, peticion string) navegador.Respuesta {
	t.Helper()

	var entrada bytes.Buffer
	var largo [4]byte
	binary.LittleEndian.PutUint32(largo[:], uint32(len(peticion)))
	entrada.Write(largo[:])
	entrada.WriteString(peticion)

	var salida bytes.Buffer
	// **Con un plazo**, porque lo que se está persiguiendo es precisamente que se
	// cuelgue: sin esto, una prueba que falla se queda ahí y hay que matarla a
	// mano, que es la clase de prueba que se acaba borrando.
	hecho := make(chan error, 1)
	go func() { hecho <- navegador.Traducir(&entrada, &salida, socket) }()
	select {
	case err := <-hecho:
		if err != nil {
			t.Fatalf("el puente ha fallado: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("el puente se ha quedado colgado sin contestar")
	}

	// Y de vuelta, desenmarcando como haría el navegador.
	crudo := salida.Bytes()
	if len(crudo) < 4 {
		t.Fatalf("no ha contestado nada: %q", crudo)
	}
	n := int(binary.LittleEndian.Uint32(crudo[:4]))
	if len(crudo) != 4+n {
		t.Fatalf("la trama no cuadra: dice %d y hay %d bytes detrás", n, len(crudo)-4)
	}
	var r navegador.Respuesta
	if err := json.Unmarshal(crudo[4:], &r); err != nil {
		t.Fatalf("lo que ha contestado no es JSON: %s", crudo[4:])
	}
	if !r.OK && strings.Contains(r.Error, "no está abierta") {
		t.Fatalf("no ha encontrado a Esfinge en %s: %+v", socket, r)
	}
	return r
}
