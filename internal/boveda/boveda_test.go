package boveda

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

const maestra = "una contraseña maestra larga"

func nueva(t *testing.T) (*Boveda, string, string) {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	b, rec, err := Crear(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	return b, rec, ruta
}

func TestSeCreaYSeAbreConLaMaestra(t *testing.T) {
	b, _, ruta := nueva(t)
	if err := b.Poner(Entrada{Titulo: "Banco", Usuario: "yo", Secreto: "s3cr3t0"}); err != nil {
		t.Fatal(err)
	}
	b.Cerrar()

	otra, err := Abrir(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	if otra.Cuantas() != 1 {
		t.Fatalf("entradas: %d", otra.Cuantas())
	}
	e := otra.Buscar("banco")
	if len(e) != 1 || e[0].Titulo != "Banco" {
		t.Fatalf("búsqueda: %+v", e)
	}
	// La lista no lleva secretos: ver Entrada.SinSecretos.
	if e[0].Secreto != "" {
		t.Error("la lista ha traído la contraseña; eso solo se pide de una en una")
	}
	entera, _ := otra.Ver(e[0].ID)
	if entera.Secreto != "s3cr3t0" {
		t.Errorf("secreto: %q", entera.Secreto)
	}
}

func TestLaDeRecuperacionAbreIgual(t *testing.T) {
	b, rec, ruta := nueva(t)
	b.Poner(Entrada{Titulo: "Correo", Secreto: "x"})
	b.Cerrar()

	otra, err := Abrir(ruta, rec)
	if err != nil {
		t.Fatalf("la clave de recuperación no abre: %v", err)
	}
	if otra.Cuantas() != 1 {
		t.Error("abrió pero no trajo nada")
	}

	// Y tecleada como la teclearía una persona: en minúsculas, sin guiones, con
	// una O donde va un cero.
	suelta := strings.ToLower(strings.ReplaceAll(rec, "-", ""))
	if _, err := Abrir(ruta, suelta); err != nil {
		t.Errorf("no admite la clave tecleada a mano: %v", err)
	}
}

// La diferencia entre «te has equivocado copiando» y «has perdido la bóveda».
func TestUnaErrataSeDistingueDeUnaClaveQueNoAbre(t *testing.T) {
	_, rec, _ := nueva(t)

	// Cambiar un carácter rompe la suma de control.
	roto := []rune(rec)
	for i := len(roto) - 1; i >= 0; i-- {
		if roto[i] != '-' {
			if roto[i] == 'A' {
				roto[i] = 'B'
			} else {
				roto[i] = 'A'
			}
			break
		}
	}
	if _, err := Normalizar(string(roto)); !errors.Is(err, ErrChecksum) {
		t.Errorf("una errata tiene que decirse como tal, no como «no abre»: %v", err)
	}
	if _, err := Normalizar(rec); err != nil {
		t.Errorf("la buena no pasa la comprobación: %v", err)
	}
}

func TestUnaLlaveCualquieraNoAbre(t *testing.T) {
	_, _, ruta := nueva(t)
	if _, err := Abrir(ruta, "otra cosa"); !errors.Is(err, ErrSinRanura) {
		t.Errorf("quiero ErrSinRanura, tengo %v", err)
	}
}

// **El test que demuestra que la jerarquía de claves sirve para algo.**
//
// Si cambiar la contraseña maestra recifrara el cuerpo, con diez mil entradas
// tardaría lo suyo y además reescribiría todo el fichero. Reenvolver son 43
// caracteres.
func TestCambiarLaMaestraNoTocaElCuerpo(t *testing.T) {
	b, rec, ruta := nueva(t)
	b.Poner(Entrada{Titulo: "Algo", Secreto: "x"})

	antes := b.doc.Cuerpo
	if err := b.CambiarMaestra("otra contraseña maestra"); err != nil {
		t.Fatal(err)
	}
	if b.doc.Cuerpo != antes {
		t.Error("el cuerpo se ha vuelto a cifrar: eso es recifrar, no reenvolver")
	}
	b.Cerrar()

	if _, err := Abrir(ruta, "otra contraseña maestra"); err != nil {
		t.Errorf("la maestra nueva no abre: %v", err)
	}
	if _, err := Abrir(ruta, maestra); !errors.Is(err, ErrSinRanura) {
		t.Error("la maestra vieja sigue abriendo")
	}
	// Y la de recuperación sigue valiendo: es la otra mitad de la promesa.
	if _, err := Abrir(ruta, rec); err != nil {
		t.Errorf("cambiar la maestra ha invalidado la clave de recuperación: %v", err)
	}
}

func TestRotarLaDeRecuperacionInvalidaLaAnterior(t *testing.T) {
	b, vieja, ruta := nueva(t)
	nuevaRec, err := b.RotarRecuperacion()
	if err != nil {
		t.Fatal(err)
	}
	b.Cerrar()

	if _, err := Abrir(ruta, nuevaRec); err != nil {
		t.Errorf("la nueva no abre: %v", err)
	}
	if _, err := Abrir(ruta, vieja); !errors.Is(err, ErrSinRanura) {
		t.Error("la clave de recuperación vieja sigue abriendo")
	}
	if _, err := Abrir(ruta, maestra); err != nil {
		t.Errorf("rotar la de recuperación ha tocado la maestra: %v", err)
	}
}

// El JSON de fuera no va autenticado, así que lo que se puede hacer es
// **detectar** que lo han tocado.
func TestDetectaQueHanEditadoElFicheroAMano(t *testing.T) {
	t.Run("quitando la ranura de recuperación", func(t *testing.T) {
		b, _, ruta := nueva(t)
		b.Cerrar()

		var doc documento
		datos, _ := os.ReadFile(ruta)
		json.Unmarshal(datos, &doc)
		doc.Sobres = doc.Sobres[:1] // fuera la de recuperación
		fuera, _ := json.Marshal(doc)
		os.WriteFile(ruta, fuera, 0o600)

		if _, err := Abrir(ruta, maestra); !errors.Is(err, ErrManipulada) {
			t.Errorf("quiero ErrManipulada, tengo %v", err)
		}
	})

	t.Run("revirtiendo el cuerpo a uno viejo", func(t *testing.T) {
		b, _, ruta := nueva(t)
		b.Poner(Entrada{Titulo: "Uno"})
		viejo := b.doc.Cuerpo
		b.Poner(Entrada{Titulo: "Dos"})
		b.Cerrar()

		var doc documento
		datos, _ := os.ReadFile(ruta)
		json.Unmarshal(datos, &doc)
		doc.Cuerpo = viejo
		fuera, _ := json.Marshal(doc)
		os.WriteFile(ruta, fuera, 0o600)

		if _, err := Abrir(ruta, maestra); !errors.Is(err, ErrManipulada) {
			t.Errorf("quiero ErrManipulada, tengo %v", err)
		}
	})
}

// Dos Esfinges abiertas a la vez: el segundo en guardar no puede llevarse por
// delante lo del primero sin decirlo.
func TestNoPisaLoQueOtroHaGuardado(t *testing.T) {
	b, _, ruta := nueva(t)
	b.Poner(Entrada{Titulo: "Del primero"})

	otra, err := Abrir(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	if err := otra.Poner(Entrada{Titulo: "Del segundo"}); err != nil {
		t.Fatal(err)
	}

	// El primero sigue creyendo que va por la serie de antes.
	err = b.Poner(Entrada{Titulo: "Otra del primero"})
	if !errors.Is(err, ErrCambiada) {
		t.Errorf("quiero ErrCambiada, tengo %v", err)
	}
}

func TestNoEsBoveda(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "cualquiera.json")
	os.WriteFile(ruta, []byte(`{"hola":"mundo"}`), 0o600)
	if _, err := Abrir(ruta, maestra); !errors.Is(err, ErrNoEsBoveda) {
		t.Errorf("quiero ErrNoEsBoveda, tengo %v", err)
	}

	os.WriteFile(ruta, []byte("basura que no es json"), 0o600)
	if _, err := Abrir(ruta, maestra); !errors.Is(err, ErrNoEsBoveda) {
		t.Errorf("con basura quiero ErrNoEsBoveda, tengo %v", err)
	}
}

// Una bóveda de una versión más nueva se puede mirar, pero **no se guarda
// encima**: escribir sobre algo que no se entiende del todo es borrarlo.
func TestUnFormatoMasNuevoNoSeAbre(t *testing.T) {
	b, _, ruta := nueva(t)
	b.Cerrar()

	var doc documento
	datos, _ := os.ReadFile(ruta)
	json.Unmarshal(datos, &doc)
	doc.Formato = Formato + 1
	fuera, _ := json.Marshal(doc)
	os.WriteFile(ruta, fuera, 0o600)

	if _, err := Abrir(ruta, maestra); !errors.Is(err, ErrFormatoNuevo) {
		t.Errorf("quiero ErrFormatoNuevo, tengo %v", err)
	}
}

// Lo que escribe una versión más nueva no puede desaparecer al guardar con una
// vieja. Es el escenario de la sincronización, y también el de un cliente que no
// actualiza.
func TestLosCamposDesconocidosSobreviven(t *testing.T) {
	crudo := []byte(`{
	  "id": "abc", "tipo": "credencial", "titulo": "Con extras",
	  "creada": "2026-01-01T00:00:00Z", "cambiada": "2026-01-01T00:00:00Z",
	  "inventadoEnElFuturo": {"algo": 42}
	}`)
	var e Entrada
	if err := json.Unmarshal(crudo, &e); err != nil {
		t.Fatal(err)
	}
	if e.Titulo != "Con extras" {
		t.Fatalf("no ha leído lo conocido: %+v", e)
	}
	if len(e.Extra) != 1 {
		t.Fatalf("no ha guardado lo desconocido: %v", e.Extra)
	}

	vuelta, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	json.Unmarshal(vuelta, &m)
	if _, hay := m["inventadoEnElFuturo"]; !hay {
		t.Error("al volver a escribir se ha perdido el campo desconocido")
	}
	if _, hay := m["titulo"]; !hay {
		t.Error("y de paso se ha perdido lo conocido")
	}
}

func TestElHistorialGuardaLaAnterior(t *testing.T) {
	var e Entrada
	ahora := time.Now()
	e.CambiarSecreto("primera", ahora)
	e.CambiarSecreto("segunda", ahora.Add(time.Hour))

	if e.Secreto != "segunda" {
		t.Errorf("secreto actual: %q", e.Secreto)
	}
	if len(e.Historial) != 1 || e.Historial[0].Secreto != "primera" {
		t.Fatalf("historial: %+v", e.Historial)
	}

	// Con tope, para que una entrada no crezca sin freno.
	for i := 0; i < 20; i++ {
		e.CambiarSecreto(string(rune('a'+i)), ahora)
	}
	if len(e.Historial) > maximoHistorial {
		t.Errorf("el historial ha crecido a %d", len(e.Historial))
	}
}

// Buscar por el secreto sería una forma silenciosa de averiguarlo probando.
func TestLaBusquedaNoMiraLosSecretos(t *testing.T) {
	e := Entrada{Titulo: "Banco", Secreto: "zanahoria", Notas: "zanahoria"}
	if e.Coincide("zanahoria") {
		t.Error("la búsqueda encuentra por el secreto")
	}
	if !e.Coincide("banco") {
		t.Error("y no encuentra por el título")
	}
}

func TestBorrarDejaRastroPeroNoElSecreto(t *testing.T) {
	b, _, _ := nueva(t)
	b.Poner(Entrada{Titulo: "Fuera", Secreto: "s3cr3t0", TOTP: "ABCD"})
	id := b.Buscar("")[0].ID

	if err := b.Borrar(id); err != nil {
		t.Fatal(err)
	}
	if b.Cuantas() != 0 {
		t.Error("sigue contando como viva")
	}
	e, hay := b.Ver(id)
	if !hay {
		t.Fatal("el borrado suave tiene que dejar rastro para poder sincronizar")
	}
	if e.Secreto != "" || e.TOTP != "" || e.Historial != nil {
		t.Error("la papelera guarda que existió, no lo que valía")
	}
}

// La bóveda tiene que abrirse con las herramientas de siempre, sin depender de
// que este paquete sea correcto. Es la historia de recuperación de verdad.
func TestLaBovedaSeAbreConLasPiezasDeSiempre(t *testing.T) {
	b, _, ruta := nueva(t)
	b.Poner(Entrada{Titulo: "A mano", Secreto: "abreme"})
	b.Cerrar()

	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var doc documento
	if err := json.Unmarshal(datos, &doc); err != nil {
		t.Fatal(err)
	}

	// Paso 1: descifrar el sobre maestro con la contraseña. Sale la clave.
	llave, err := abrirTextoDeSiempre(doc.Sobres[0].Contenedor, maestra)
	if err != nil {
		t.Fatalf("el sobre no se abre con las piezas de siempre: %v", err)
	}
	// Paso 2: descifrar el cuerpo con esa clave.
	claro, err := abrirTextoDeSiempre(doc.Cuerpo, string(llave))
	if err != nil {
		t.Fatalf("el cuerpo no se abre con la clave del sobre: %v", err)
	}
	if !strings.Contains(string(claro), "abreme") {
		t.Error("el cuerpo no lleva lo que debía")
	}
}

// abrirTextoDeSiempre es a propósito una llamada pelada a internal/cripto, sin
// pasar por nada de este paquete: si la bóveda solo se pudiera abrir con el
// código de la bóveda, la historia de recuperación dependería de que ese código
// siga siendo correcto dentro de cinco años.
func abrirTextoDeSiempre(contenedor, clave string) ([]byte, error) {
	return cripto.AbrirTexto(contenedor, []byte(clave))
}
