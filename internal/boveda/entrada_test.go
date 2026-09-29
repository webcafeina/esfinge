package boveda

import (
	"encoding/json"
	"strings"
	"testing"
)

// **La dirección que escribió la 2.30.0 no se queda escondida.**
//
// Aquella versión la guardaba compuesta en un solo campo de texto, `direccion`, y
// la 2.31.0 la parte en nueve (ADR 0047). Un campo que desaparece de la estructura
// **no da error**: cae en `Extra`, se conserva y no se ve nunca más, que es
// exactamente la pérdida silenciosa que `Extra` existe para no tener —ahí se
// guarda lo que escribió una versión **más nueva**, no lo que escribió una vieja y
// esta entiende—.
//
// No se parte en sus trozos porque eso sería adivinar: entra entera en la calle,
// con sus saltos de línea, donde se ve y se puede repartir a mano.
func TestLaDireccionDeLa2300NoSeQuedaEscondida(t *testing.T) {
	crudo := []byte(`{"id":"1","tipo":"personal","titulo":"Casa","creada":"2026-09-29T10:00:00Z",` +
		`"cambiada":"2026-09-29T10:00:00Z","direccion":"Calle Mayor 1\n28001 Madrid"}`)

	var e Entrada
	if err := json.Unmarshal(crudo, &e); err != nil {
		t.Fatal(err)
	}
	if e.Calle != "Calle Mayor 1\n28001 Madrid" {
		t.Errorf("la dirección de antes no ha llegado a la calle: %q", e.Calle)
	}
	if _, sigue := e.Extra["direccion"]; sigue {
		// Si se quedara también en `Extra`, al volver a escribir saldrían las dos y
		// la siguiente lectura la traería dos veces.
		t.Error("sigue en Extra, y entonces se escribiría dos veces")
	}

	// Y al volver a escribirla, `direccion` ya no está: lo que hay son los campos
	// nuevos. Es una migración de ida, y por eso se dice en la ADR.
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"direccion"`) {
		t.Errorf("vuelve a escribir el campo viejo: %s", b)
	}
	if !strings.Contains(string(b), `"calle"`) {
		t.Errorf("no escribe el campo nuevo: %s", b)
	}
}
