package agente

import (
	"encoding/json"
	"strings"
	"testing"
)

// **Al agente no se le dice «cerrada» de una bóveda que puede usar** (2026-10-20).
//
// En Esfinge «cerrada» quiere decir **bloqueada**: la que pide la contraseña maestra. Una
// bóveda que no es la activa **no está cerrada en ese sentido** —se conmuta sin
// contraseña, porque la clave está en memoria desde la ADR 0050—, pero el canal le
// devolvía un campo `abierta: false` y el modelo juntó los dos sentidos.
//
// Lo que dijo el agente del cliente, con la 2.58.6 en Claude Code: «están cerradas,
// Esfinge no me deja abrirlas desde aquí». **Y sí podía**: al nombrar la bóveda funcionó
// a la primera, y él mismo lo dijo —«funcionó aunque la bóveda figuraba como cerrada»—.
// O sea que una palabra con dos sentidos le quitó una funcionalidad entera que el cliente
// había pedido, y la ADR 0055 existe justo para que el agente no diga que no puede.
//
// Esta guarda es de texto porque el fallo era de texto: **aquí las descripciones son el
// producto**, no documentación.
func TestNoSeLeDiceCerradaAUnaBovedaQueSePuedeUsar(t *testing.T) {
	// El campo de la respuesta se llama por lo que es.
	var b UnaBoveda
	datos, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(datos), `"abierta"`) {
		t.Error("la lista de bóvedas vuelve a decir «abierta», que el modelo lee como «las demás están cerradas»")
	}
	if !strings.Contains(string(datos), `"activa"`) {
		t.Error("la lista de bóvedas tendría que decir cuál es la «activa»")
	}

	// Y la descripción desmiente la lectura equivocada en vez de callarla.
	var bovedas *Herramienta
	for i := range LasHerramientas {
		if LasHerramientas[i].Nombre == "esfinge_bovedas" {
			bovedas = &LasHerramientas[i]
		}
	}
	if bovedas == nil {
		t.Fatal("no está esfinge_bovedas")
	}
	for _, loQueTieneQueDecir := range []string{
		// Que se usan igual.
		"nómbralas",
		// Que conmuta sin pedir nada.
		"sin pedir ninguna contraseña",
		// Y qué quiere decir «cerrada» aquí, que es lo que confundía.
		"bloqueada",
	} {
		if !strings.Contains(bovedas.Descripcion, loQueTieneQueDecir) {
			t.Errorf("la descripción de esfinge_bovedas ya no dice %q, que es lo que evita que el "+
				"agente se niegue a usar las demás", loQueTieneQueDecir)
		}
	}
}
