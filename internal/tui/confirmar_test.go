package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Salir con una contraseña en pantalla que no se ha guardado ni copiado tiene
// que preguntar antes: cerrar y perderla es irreversible.
func TestPreguntaAntesDePerderElResultado(t *testing.T) {
	m := nuevo(t)
	m, _ = clic(t, m, "menu:2") // Generar
	if m.aSalvo {
		t.Fatal("da por guardada una contraseña recién generada")
	}

	m, _ = clic(t, m, zonaVolver)
	if m.pantalla != pantConfirmar {
		t.Fatalf("no ha preguntado: la pantalla es %v", m.pantalla)
	}

	v := m.View()
	for _, quiero := range []string{"Guardar y salir", "Salir sin guardar", "Cancelar"} {
		if !strings.Contains(v, quiero) {
			t.Errorf("la confirmación no ofrece «%s»", quiero)
		}
	}
}

func TestCancelarVuelveAlResultado(t *testing.T) {
	m := nuevo(t)
	m, _ = clic(t, m, "menu:2")
	contrasena := m.resultado

	m, _ = clic(t, m, zonaVolver)
	m, _ = clic(t, m, zonaCancelar)

	if m.pantalla != pantResultado {
		t.Fatal("cancelar no devolvió al resultado")
	}
	if m.resultado != contrasena {
		t.Error("se ha perdido la contraseña por el camino")
	}
}

func TestGuardarYSalirDejaElFichero(t *testing.T) {
	dir := descargasDePrueba(t)

	m := nuevo(t)
	m, _ = clic(t, m, "menu:2")
	contrasena := m.resultado

	m, _ = clic(t, m, zonaVolver)     // pregunta
	m, _ = clic(t, m, zonaGuardaYSal) // guardar y salir

	if m.pantalla != pantMenu {
		t.Errorf("tras guardar y salir la pantalla es %v", m.pantalla)
	}

	ficheros, _ := filepath.Glob(filepath.Join(dir, "esfinge-contrasena-*.txt"))
	if len(ficheros) != 1 {
		t.Fatalf("esperaba un fichero, hay %v", ficheros)
	}
	datos, _ := os.ReadFile(ficheros[0])
	if strings.TrimSpace(string(datos)) != contrasena {
		t.Error("el fichero no tiene la contraseña que había en pantalla")
	}
}

func TestSalirSinGuardarVuelveAlMenu(t *testing.T) {
	m := nuevo(t)
	m, _ = clic(t, m, "menu:2")
	m, _ = clic(t, m, zonaVolver)
	m, _ = clic(t, m, zonaSalirYa)

	if m.pantalla != pantMenu {
		t.Errorf("salir sin guardar dejó la pantalla en %v", m.pantalla)
	}
	if m.resultado != "" {
		t.Error("la contraseña sigue en el modelo tras salir")
	}
}

// Lo que ya está guardado no vuelve a preguntar.
func TestNoPreguntaSiYaEstaGuardado(t *testing.T) {
	descargasDePrueba(t)

	m := nuevo(t)
	m, _ = clic(t, m, "menu:2")
	m, _ = clic(t, m, zonaGuarda)
	if !m.aSalvo {
		t.Fatal("guardar no lo ha marcado como a salvo")
	}

	m, _ = clic(t, m, zonaVolver)
	if m.pantalla != pantMenu {
		t.Errorf("ha preguntado por algo ya guardado: pantalla %v", m.pantalla)
	}
}

// La ayuda no se pierde al cerrarla, así que tampoco pregunta.
func TestLaAyudaNoPregunta(t *testing.T) {
	m := nuevo(t)
	m, _ = clic(t, m, "menu:3") // Ayuda
	m, _ = clic(t, m, zonaVolver)

	if m.pantalla != pantMenu {
		t.Errorf("la ayuda ha preguntado al cerrarse: pantalla %v", m.pantalla)
	}
}

// Un fichero cifrado ya está en disco: preguntar ahí sería preguntar por nada.
func TestUnFicheroCifradoNoPregunta(t *testing.T) {
	dir := t.TempDir()
	origen := filepath.Join(dir, "datos.txt")
	os.WriteFile(origen, []byte("contenido"), 0o644)

	msg := trabajarFichero(accCifrar, origen, []byte("clave")).(listoMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	if !msg.aSalvo {
		t.Error("un fichero recién escrito en disco no se da por guardado")
	}
}

// El aviso de que sin la clave no hay vuelta atrás sale en la pantalla de
// resultado, que es cuando la persona decide si guarda o cierra.
func TestElAvisoSaleEnElResultado(t *testing.T) {
	msg := trabajarTexto(accCifrar, "secreto", []byte("clave")).(listoMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	if !strings.Contains(msg.aviso, "clave") {
		t.Errorf("el resultado de cifrar no avisa: %q", msg.aviso)
	}

	m := nuevo(t)
	sig, _ := m.Update(msg)
	if v := sig.(modelo).View(); !strings.Contains(v, "no lo abre nadie") {
		t.Error("el aviso no llega a la pantalla")
	}
}
