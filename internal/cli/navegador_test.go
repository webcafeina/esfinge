package cli

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
)

// esfingeDeMentira levanta algo que habla como Esfinge por un socket: lee una
// petición por línea y contesta lo que se le diga.
func esfingeDeMentira(t *testing.T, contestar func(json.RawMessage) any) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "puente.sock")
	ln, err := net.Listen("unix", ruta)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				dec, enc := json.NewDecoder(c), json.NewEncoder(c)
				for {
					var p json.RawMessage
					if dec.Decode(&p) != nil {
						return
					}
					if enc.Encode(contestar(p)) != nil {
						return
					}
				}
			}()
		}
	}()
	return ruta
}

// enMensaje empaqueta lo que mandaría el navegador: cuatro bytes de largo y el
// JSON detrás.
func enMensaje(cuerpos ...string) []byte {
	var b bytes.Buffer
	for _, c := range cuerpos {
		var largo [4]byte
		binary.LittleEndian.PutUint32(largo[:], uint32(len(c)))
		b.Write(largo[:])
		b.WriteString(c)
	}
	return b.Bytes()
}

// deMensajes desempaqueta lo que el navegador recibiría.
func deMensajes(t *testing.T, crudo []byte) []string {
	t.Helper()
	var fuera []string
	r := bytes.NewReader(crudo)
	for {
		var largo [4]byte
		if _, err := io.ReadFull(r, largo[:]); err != nil {
			return fuera
		}
		cuerpo := make([]byte, binary.LittleEndian.Uint32(largo[:]))
		if _, err := io.ReadFull(r, cuerpo); err != nil {
			t.Fatalf("un mensaje se ha quedado a medias: %q", cuerpo)
		}
		fuera = append(fuera, string(cuerpo))
	}
}

// El camino entero del traductor: dos preguntas del navegador, dos respuestas de
// Esfinge, con el enmarcado de verdad por los dos lados.
func TestElTraductorVaYVuelve(t *testing.T) {
	vistas := []string{}
	socket := esfingeDeMentira(t, func(p json.RawMessage) any {
		vistas = append(vistas, string(p))
		return map[string]any{"ok": true, "eco": json.RawMessage(p)}
	})

	entra := bytes.NewReader(enMensaje(
		`{"version":1,"que":"estado"}`,
		`{"version":1,"que":"cuentas","origen":"https://banco.es"}`,
	))
	var sale bytes.Buffer
	if err := Traducir(entra, &sale, socket); err != nil {
		t.Fatal(err)
	}

	respuestas := deMensajes(t, sale.Bytes())
	if len(respuestas) != 2 {
		t.Fatalf("han vuelto %d respuestas: %q", len(respuestas), respuestas)
	}
	for i, r := range respuestas {
		if !strings.Contains(r, `"ok":true`) {
			t.Errorf("respuesta %d: %s", i, r)
		}
	}
	if len(vistas) != 2 || !strings.Contains(vistas[1], "banco.es") {
		t.Errorf("lo que le ha llegado a Esfinge: %q", vistas)
	}
}

// **Sin Esfinge al otro lado se contesta, no se muere.** La extensión necesita
// poder decir «abre Esfinge», que es una cosa distinta de «algo ha fallado», y
// para eso necesita una respuesta.
func TestSinEsfingeSeContestaIgual(t *testing.T) {
	entra := bytes.NewReader(enMensaje(`{"version":1,"que":"estado"}`))
	var sale bytes.Buffer

	if err := Traducir(entra, &sale, filepath.Join(t.TempDir(), "no-existe.sock")); err != nil {
		t.Fatal(err)
	}
	respuestas := deMensajes(t, sale.Bytes())
	if len(respuestas) != 1 {
		t.Fatalf("respuestas: %q", respuestas)
	}
	if !strings.Contains(respuestas[0], "sin-esfinge") {
		t.Errorf("no dice qué pasa: %s", respuestas[0])
	}
	// Y el motivo es una etiqueta estable, no un texto que alguien vaya a mirar
	// con una expresión regular.
	var r struct {
		OK     bool   `json:"ok"`
		Motivo string `json:"motivo"`
	}
	if err := json.Unmarshal([]byte(respuestas[0]), &r); err != nil {
		t.Fatal(err)
	}
	if r.OK || r.Motivo != "sin-esfinge" {
		t.Errorf("%+v", r)
	}
}

// Lo que el navegador manda mal no puede colgar el proceso ni hacerle escribir
// basura: se termina y ya.
func TestUnMensajeMalFormadoNoRompeNada(t *testing.T) {
	socket := esfingeDeMentira(t, func(json.RawMessage) any {
		return map[string]any{"ok": true}
	})

	casos := map[string][]byte{
		"cortado por la mitad": append(enMensaje(`{"version":1}`)[:6]),
		"solo la longitud":     enMensaje(`{"a":1}`)[:4],
		"vacío":                nil,
		"una longitud de cero": {0, 0, 0, 0},
		"una longitud absurda": {255, 255, 255, 255},
	}
	for nombre, crudo := range casos {
		t.Run(nombre, func(t *testing.T) {
			var sale bytes.Buffer
			if err := Traducir(bytes.NewReader(crudo), &sale, socket); err != nil {
				t.Errorf("ha devuelto error: %v", err)
			}
			if sale.Len() != 0 {
				t.Errorf("ha escrito algo: %q", sale.Bytes())
			}
		})
	}
}

// **Nada que no sea el protocolo puede salir por ahí.** La salida estándar del
// proceso *es* el canal: una línea de más parte una trama y el navegador mata el
// proceso sin decir por qué.
func TestPorLaSalidaSoloSaleElProtocolo(t *testing.T) {
	socket := esfingeDeMentira(t, func(json.RawMessage) any {
		return map[string]any{"ok": true}
	})
	var sale bytes.Buffer
	if err := Traducir(bytes.NewReader(enMensaje(`{"version":1,"que":"estado"}`)), &sale, socket); err != nil {
		t.Fatal(err)
	}

	crudo := sale.Bytes()
	// Cuatro bytes de longitud, y detrás exactamente esa cantidad. Si sobra o
	// falta un byte, alguien ha escrito por donde no debía.
	if len(crudo) < 4 {
		t.Fatalf("no ha salido nada: %q", crudo)
	}
	n := int(binary.LittleEndian.Uint32(crudo[:4]))
	if len(crudo) != 4+n {
		t.Fatalf("la trama no cuadra: dice %d y hay %d bytes detrás", n, len(crudo)-4)
	}
	if !json.Valid(crudo[4:]) {
		t.Errorf("lo que sale no es JSON: %q", crudo[4:])
	}
}
