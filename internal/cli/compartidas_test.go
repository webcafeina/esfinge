package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/rutas"
)

// **Cuando el nombre no encaja, el error dice cuáles hay.**
//
// El nombre de una bóveda compartida se anuncia **una sola vez**, al aceptarla con
// «servidor recibir», y después no había dónde volver a verlo: `--compartida` decía
// que no encajaba y se callaba lo que tenía. En un servidor eso es peor que en un
// portátil, porque quien trabaja ahí es un agente y no tiene a quién preguntarle.
//
// Lo vio el paseo del 2026-11-09, llevando el `.env` de Tempero a su bóveda: para
// saber qué le habían compartido a este equipo hubo que **escribir un programa de un
// solo uso** que abriera la bóveda del servidor y leyera `Compartidas()`.
func TestElErrorDeUnaCompartidaQueNoEncajaDiceCualesHay(t *testing.T) {
	o, ob, _ := unaCompartidaEnEsteEquipo(t, "ver")
	ob.compartida = "Nolaquesea"

	err := correr(t, comandoBovedaListar(o, ob), nil)
	if err == nil {
		t.Fatal("ha listado algo con un nombre de bóveda que no existe")
	}
	if !strings.Contains(err.Error(), "Zeri's Coffee") {
		t.Errorf("el error no dice qué bóvedas tiene este equipo: %v", err)
	}
}

// Y **sin ninguna compartida es otra cosa, y se dice distinto**: lo que falta entonces
// no es afinar el nombre, es aceptarlas. Con un solo mensaje para los dos casos, quien
// estrena un servidor se queda buscando un nombre que todavía no existe en su equipo.
func TestSinNingunaCompartidaElErrorMandaARecibir(t *testing.T) {
	rutas.ApartarLaCarpeta(t.TempDir(), t.Setenv)
	mia, _, err := boveda.Crear(rutas.BovedaPrincipal(), laMaestraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	mia.Cerrar()

	o := &opciones{tema: "oscuro", silencio: true}
	ob := &opcionesBoveda{compartida: "Tempero", sinSincronizar: true}
	conLaMaestra := filepath.Join(t.TempDir(), "maestra.txt")
	if err := os.WriteFile(conLaMaestra, []byte(laMaestraDePrueba), 0o600); err != nil {
		t.Fatal(err)
	}
	ob.clave.Fichero = conLaMaestra

	err = correr(t, comandoBovedaListar(o, ob), nil)
	if err == nil {
		t.Fatal("ha listado una bóveda compartida que no existe")
	}
	if !strings.Contains(err.Error(), "recibir") {
		t.Errorf("el error no dice qué hacer cuando no hay ninguna: %v", err)
	}
	if strings.Contains(err.Error(), "encaje") {
		t.Errorf("con ninguna compartida dice lo mismo que con el nombre mal: %v", err)
	}
}
