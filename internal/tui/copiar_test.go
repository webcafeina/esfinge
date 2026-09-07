package tui

import (
	"encoding/base64"
	"strings"
	"testing"
)

// Cifrar un texto pide el copiado sin que nadie lo mande: lo siguiente que se
// hace con un texto cifrado es pegarlo en algún sitio, siempre.
func TestCifrarPideElCopiadoSolo(t *testing.T) {
	msg := trabajarTexto(accCifrar, "un secreto", []byte("clave")).(listoMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	if !msg.copiarSolo {
		t.Error("cifrar no pide copiar al portapapeles")
	}

	// Descifrar no: lo que sale ahí es el secreto en claro, y dejarlo en el
	// portapapeles de la máquina sin que nadie lo haya pedido es meterlo donde
	// puede leerlo cualquier cosa que mire el portapapeles.
	descifrado := trabajarTexto(accDescifrar, msg.texto, []byte("clave")).(listoMsg)
	if descifrado.copiarSolo {
		t.Error("descifrar deja el secreto en claro en el portapapeles sin permiso")
	}
}

// Y el intento de copiado deja rastro en pantalla, salga bien o mal: enterarse
// de que el texto ya está en el portapapeles no puede depender de adivinarlo.
func TestElCopiadoSeVe(t *testing.T) {
	m := nuevo(t)
	msg := trabajarTexto(accCifrar, "un secreto", []byte("clave")).(listoMsg)

	sig, _ := m.Update(msg)
	mm := sig.(modelo)

	if mm.exito == "" && mm.nota == "" {
		t.Fatal("tras cifrar no se dice nada del portapapeles")
	}

	v := mm.View()
	switch {
	case mm.exito != "":
		// Ha copiado: se anuncia con su glifo, para que se distinga sin color.
		if !strings.Contains(v, "Copiado al portapapeles") {
			t.Error("ha copiado y no lo dice en pantalla")
		}
		if !strings.Contains(v, "✓") {
			t.Error("la confirmación no lleva glifo")
		}
		if !mm.aSalvo {
			t.Error("copiado y sin marcar como a salvo: preguntaría al salir sin motivo")
		}
	default:
		// No ha podido —una máquina sin portapapeles, que es el caso de un
		// servidor—: entonces tiene que mandar a la otra vía, no callarse.
		if !strings.Contains(mm.nota, "Guardar") {
			t.Errorf("el fallo al copiar no ofrece alternativa: %q", mm.nota)
		}
	}
}

// El resultado cifrado se manda también por OSC 52, que es la única vía que
// funciona a través de SSH.
func TestSePideElCopiadoAlTerminal(t *testing.T) {
	m := nuevo(t)
	msg := trabajarTexto(accCifrar, "un secreto", []byte("clave")).(listoMsg)
	sig, _ := m.Update(msg)
	mm := sig.(modelo)

	v := mm.View()
	if !strings.Contains(v, "\x1b]52;c;") {
		t.Fatal("no se ha mandado la petición de copiado al terminal")
	}

	// Y lo que va dentro es el contenedor, no otra cosa.
	i := strings.Index(v, "\x1b]52;c;") + len("\x1b]52;c;")
	j := strings.Index(v[i:], "\x07")
	datos, err := base64.StdEncoding.DecodeString(v[i : i+j])
	if err != nil {
		t.Fatalf("la petición no lleva base64 válido: %v", err)
	}
	if string(datos) != mm.resultado {
		t.Errorf("se manda %q y el resultado es %q", datos, mm.resultado)
	}

	// Solo una vez: si se repitiera en cada redibujado, el terminal recibiría la
	// petición decenas de veces por segundo.
	if strings.Contains(mm.View(), "\x1b]52;c;") {
		t.Error("la petición de copiado se repite en el siguiente dibujado")
	}
}

func TestCopiarNadaNoRompe(t *testing.T) {
	m := nuevo(t)
	sig, _ := m.copiar()
	if sig.(modelo).err != nil {
		t.Error("copiar sin nada en pantalla ha dado error")
	}
}
