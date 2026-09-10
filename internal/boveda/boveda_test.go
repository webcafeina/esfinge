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
//
// **Y se comprueba por donde pasa la persona, que es Abrir.** Preguntándole solo
// a Normalizar esto pasaba en verde mientras Abrir contestaba «esa llave no abre
// esta bóveda» a una errata: el comentario prometía la distinción y el código se
// la comía. Lo pilló una prueba de interfaz, no ésta.
func TestUnaErrataSeDistingueDeUnaClaveQueNoAbre(t *testing.T) {
	_, rec, ruta := nueva(t)

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

	if _, err := Abrir(ruta, string(roto)); !errors.Is(err, ErrChecksum) {
		t.Errorf("al abrir, una errata se cuenta como «no abre»: %v", err)
	}
	// Y lo que no pretendía ser una clave de recuperación sigue diciendo lo suyo,
	// aunque empiece por las mismas tres letras.
	if _, err := Abrir(ruta, "una contraseña cualquiera"); !errors.Is(err, ErrSinRanura) {
		t.Errorf("una contraseña que no abre: quiero ErrSinRanura, tengo %v", err)
	}
}

// Una contraseña maestra que empiece por ESF es rara, pero es legítima: no puede
// quedarse fuera por parecerse a una clave de recuperación.
func TestUnaMaestraQueEmpiezaPorESFSigueAbriendo(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "boveda.esfinge")
	maestra := "ESFINGE es mi contraseña"
	if _, _, err := Crear(ruta, maestra); err != nil {
		t.Fatal(err)
	}
	if _, err := Abrir(ruta, maestra); err != nil {
		t.Errorf("no abre con su propia maestra: %v", err)
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

// Borrar y volver a traer, que es lo que la papelera existe para permitir.
//
// **Una de cada clase**, y esa es la gracia: el secreto de una credencial es su
// contraseña, el de una nota segura es su texto y el de una tarjeta es su
// número, así que una papelera que solo devolviera bien las credenciales sería
// una papelera rota para tres de las cuatro.
func TestLoBorradoVuelveEnteroDeLaPapelera(t *testing.T) {
	b, _, ruta := nueva(t)
	b.Poner(Entrada{Titulo: "Fuera", Secreto: "s3cr3t0", TOTP: "ABCD"})
	b.Poner(Entrada{Titulo: "Nota", Tipo: TipoNota, Notas: "la combinación es 4242"})
	b.Poner(Entrada{Titulo: "Tarjeta", Tipo: TipoTarjeta,
		Numero: "4111111111111111", Verificacion: "737"})
	b.Poner(Entrada{Titulo: "Documento", Tipo: TipoIdentidad, NumeroDocumento: "12345678Z"})

	for _, e := range b.Buscar("") {
		if err := b.Borrar(e.ID); err != nil {
			t.Fatal(err)
		}
	}
	if b.Cuantas() != 0 {
		t.Error("siguen contando como vivas")
	}
	if b.EnLaPapelera() != 4 {
		t.Errorf("en la papelera hay %d de 4", b.EnLaPapelera())
	}
	// La lista de la papelera es una lista más: **sin secretos**.
	for _, e := range b.Papelera() {
		if e.Secreto != "" || e.Notas != "" || e.Numero != "" || e.NumeroDocumento != "" {
			t.Errorf("la papelera ha traído el secreto de «%s»", e.Titulo)
		}
	}

	// **Se restaura después de cerrar y volver a abrir**, que es el caso de
	// verdad: nadie borra y restaura en el mismo minuto. Y así se comprueba de
	// paso que lo borrado se guardó entero en el fichero.
	b.Cerrar()
	b, err := Abrir(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range b.Papelera() {
		if err := b.Restaurar(e.ID); err != nil {
			t.Fatal(err)
		}
	}
	if b.Cuantas() != 4 || b.EnLaPapelera() != 0 {
		t.Fatalf("después de restaurar hay %d vivas y %d en la papelera",
			b.Cuantas(), b.EnLaPapelera())
	}

	quiero := map[string]string{
		"Fuera": "s3cr3t0", "Nota": "la combinación es 4242",
		"Tarjeta": "4111111111111111", "Documento": "12345678Z",
	}
	for _, l := range b.Buscar("") {
		e, _ := b.Ver(l.ID)
		suyo := e.Secreto + e.Notas + e.Numero + e.NumeroDocumento
		if suyo != quiero[e.Titulo] {
			t.Errorf("«%s» ha vuelto con %q y se borró con %q",
				e.Titulo, suyo, quiero[e.Titulo])
		}
	}
}

// Vaciar la papelera se lleva lo borrado del fichero, y esta vez de verdad.
func TestVaciarLaPapeleraSeLoLlevaDelFichero(t *testing.T) {
	b, _, ruta := nueva(t)
	b.Poner(Entrada{Titulo: "Se queda", Secreto: "vive"})
	b.Poner(Entrada{Titulo: "Se va", Secreto: "muere", Notas: "y su nota"})

	for _, e := range b.Buscar("se va") {
		b.Borrar(e.ID)
	}
	cuantas, err := b.VaciarPapelera()
	if err != nil {
		t.Fatal(err)
	}
	if cuantas != 1 {
		t.Errorf("dice haber vaciado %d entradas", cuantas)
	}
	if b.Cuantas() != 1 || b.EnLaPapelera() != 0 {
		t.Errorf("quedan %d vivas y %d borradas", b.Cuantas(), b.EnLaPapelera())
	}

	// **Sobre lo guardado, no sobre la memoria**: una entrada quitada de la lista
	// con el fichero sin reescribir se vería igual desde aquí.
	b.Cerrar()
	b, err = Abrir(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.cont.Entradas) != 1 || b.cont.Entradas[0].Titulo != "Se queda" {
		t.Errorf("en el fichero quedan %d entradas: %+v", len(b.cont.Entradas), b.cont.Entradas)
	}

	// Y vaciar una papelera vacía no es un error ni reescribe nada.
	if n, err := b.VaciarPapelera(); err != nil || n != 0 {
		t.Errorf("vaciar lo ya vacío devuelve %d, %v", n, err)
	}
}

// La papelera se vacía sola a los treinta días, **y al abrir la bóveda**: un
// reloj solo contaría mientras la aplicación estuviera puesta, así que el plazo
// dependería de cuánto la usa cada uno.
func TestLaPapeleraSeVaciaSolaALosTreintaDias(t *testing.T) {
	b, _, ruta := nueva(t)
	b.Poner(Entrada{Titulo: "Vieja", Secreto: "caduca"})
	b.Poner(Entrada{Titulo: "Reciente", Secreto: "aguanta"})
	b.Poner(Entrada{Titulo: "Viva", Secreto: "ni se toca"})

	viejo := time.Now().Add(-40 * 24 * time.Hour)
	conReloj(t, func() time.Time { return viejo }, func() {
		for _, e := range b.Buscar("vieja") {
			b.Borrar(e.ID)
		}
	})
	for _, e := range b.Buscar("reciente") {
		b.Borrar(e.ID)
	}
	b.Cerrar()

	b, err := Abrir(ruta, maestra)
	if err != nil {
		t.Fatal(err)
	}
	quedan := map[string]bool{}
	for _, e := range b.cont.Entradas {
		quedan[e.Titulo] = true
	}
	if quedan["Vieja"] {
		t.Error("una entrada borrada hace cuarenta días sigue ahí")
	}
	if !quedan["Reciente"] {
		t.Error("se ha llevado una borrada hace un rato")
	}
	if !quedan["Viva"] {
		t.Error("se ha llevado una que no estaba borrada")
	}

	// **Una entrada en la papelera sin fecha no se toca.** Solo puede venir de una
	// versión que no la escribía, y tirar datos por no saber cuándo se borraron es
	// justo lo que no hay que hacer.
	b.cont.Entradas = append(b.cont.Entradas, Entrada{
		ID: "sinfecha", Titulo: "Sin fecha", Papelera: true,
	})
	if n := b.purgarPapelera(time.Now().Add(time.Hour)); n != 1 {
		t.Errorf("ha purgado %d entradas y solo debía llevarse la que tiene fecha", n)
	}
}

// conReloj corre algo con la hora parada donde se diga.
func conReloj(t *testing.T, reloj func() time.Time, hacer func()) {
	t.Helper()
	antes := ahora
	ahora = reloj
	defer func() { ahora = antes }()
	hacer()
}

// Lo borrado no bloquea su propia reimportación.
//
// **Es la trampa que trajo guardar el contenido en la papelera**: con la entrada
// entera ahí dentro, el índice de duplicados la reconocía y volver a pasar el CSV
// la daba por repetida. Se veía como «la borré, la reimporté y no ha vuelto», con
// la única copia escondida en la papelera y a punto de caducar.
func TestLoBorradoNoBloqueaVolverAImportarlo(t *testing.T) {
	b, _, _ := nueva(t)
	fila := Entrada{Titulo: "Banco", Usuario: "yo", Secreto: "s3cr3t0",
		Sitios: []string{"https://banco.es"}}

	if r, err := b.Importar([]Entrada{fila}, "Dashlane"); err != nil || r.Metidas != 1 {
		t.Fatalf("la primera importación: %+v, %v", r, err)
	}
	b.Borrar(b.Buscar("banco")[0].ID)

	r, err := b.Importar([]Entrada{fila}, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if r.Metidas != 1 || r.Repetidas != 0 {
		t.Errorf("reimportar lo borrado no lo devuelve: %+v", r)
	}
	if b.Cuantas() != 1 {
		t.Errorf("hay %d entradas vivas", b.Cuantas())
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
