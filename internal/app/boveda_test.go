package app

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/escritura"
)

// El camino de verdad, del principio al final: crear, importar de Dashlane,
// buscar, ver un secreto, copiarlo, bloquear y volver a abrir.
func TestElCaminoCompletoDeLaBoveda(t *testing.T) {
	a, s, ahora := conReloj(t)

	// 1. No hay bóveda todavía.
	if e := a.EstadoBoveda(); e.Existe || e.Abierta {
		t.Fatalf("estado de partida: %+v", e)
	}

	// 2. Se crea, y la clave de recuperación sale **una sola vez**.
	rec, err := a.CrearBoveda("una contraseña maestra larga")
	if err != nil {
		t.Fatal(err)
	}
	if rec == "" {
		t.Fatal("no ha devuelto clave de recuperación")
	}
	if _, err := boveda.Normalizar(rec); err != nil {
		t.Errorf("la clave que se le enseña a la persona no pasa su propia comprobación: %v", err)
	}

	// 3. Se importa un CSV de Dashlane por el diálogo del sistema.
	csv := filepath.Join(t.TempDir(), "credenciales-dashlane.csv")
	os.WriteFile(csv, []byte("username,title,password,url,note\n"+
		"yo@ejemplo.com,Banco,s3cr3t0,https://banco.es,una nota\n"+
		"otro@ejemplo.com,Correo,otra clave,https://correo.es,\n"), 0o600)
	s.mu.Lock()
	s.ficheros = []string{csv}
	s.mu.Unlock()

	resumen, err := a.ImportarEnBoveda("Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if resumen.Metidas != 2 {
		t.Fatalf("importadas: %+v", resumen)
	}

	// 4. Buscar no trae secretos.
	lista, err := a.BuscarEnBoveda("banco")
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 1 {
		t.Fatalf("búsqueda: %d resultados", len(lista))
	}
	if lista[0].Secreto != "" {
		t.Error("la lista ha traído la contraseña; los secretos salen de uno en uno")
	}

	// 5. Ver una entrada sí la trae.
	entera, err := a.VerDeBoveda(lista[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if entera.Secreto != "s3cr3t0" {
		t.Errorf("secreto: %q", entera.Secreto)
	}

	// 6. Copiarla arma el borrado del portapapeles.
	if err := a.Copiar(entera.Secreto); err != nil {
		t.Fatal(err)
	}
	if s.verPortapapeles() != "s3cr3t0" {
		t.Errorf("portapapeles: %q", s.verPortapapeles())
	}

	// 7. Pasa el tiempo: se borra el portapapeles y se bloquea la bóveda.
	*ahora = ahora.Add(bloqueoPorDefecto + time.Minute)
	a.repasar()

	if s.verPortapapeles() != "" {
		t.Errorf("el portapapeles no se ha borrado: %q", s.verPortapapeles())
	}
	if e := a.EstadoBoveda(); e.Abierta {
		t.Error("la bóveda no se ha bloqueado sola")
	}
	if _, err := a.BuscarEnBoveda(""); !errors.Is(err, boveda.ErrCerrada) {
		t.Errorf("bloqueada y aun así deja buscar: %v", err)
	}

	// 8. Y se vuelve a abrir con la clave de recuperación.
	if err := a.AbrirBoveda(rec); err != nil {
		t.Fatalf("la clave de recuperación no abre: %v", err)
	}
	if e := a.EstadoBoveda(); !e.Abierta || e.Cuantas != 2 {
		t.Errorf("tras reabrir: %+v", e)
	}
}

// Una bóveda abierta encima de una mesa es una bóveda a la que cualquiera que
// pase puede cambiarle la contraseña y dejar fuera a su dueño.
func TestCambiarLaMaestraPideLaDeAntes(t *testing.T) {
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("la de siempre"); err != nil {
		t.Fatal(err)
	}

	if err := a.CambiarMaestraDeBoveda("la que no es", "la nueva"); err == nil {
		t.Fatal("ha cambiado la contraseña sin saber la de antes")
	}
	if err := a.CambiarMaestraDeBoveda("la de siempre", "la nueva"); err != nil {
		t.Fatal(err)
	}

	a.CerrarBoveda()
	if err := a.AbrirBoveda("la nueva"); err != nil {
		t.Errorf("la nueva no abre: %v", err)
	}
}

func TestNoSeCreanDosBovedas(t *testing.T) {
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("la primera"); err != nil {
		t.Fatal(err)
	}
	a.CerrarBoveda()
	if _, err := a.CrearBoveda("la segunda"); err == nil {
		t.Error("ha dejado crear otra bóveda encima de la que había")
	}
}

// Con la bóveda cerrada, todo lo que necesite la llave tiene que decir que está
// cerrada y no devolver un cero silencioso.
func TestConLaBovedaCerradaNadaFunciona(t *testing.T) {
	a, _, _ := conReloj(t)

	if _, err := a.BuscarEnBoveda(""); !errors.Is(err, boveda.ErrCerrada) {
		t.Errorf("buscar: %v", err)
	}
	if _, err := a.VerDeBoveda("loquesea"); !errors.Is(err, boveda.ErrCerrada) {
		t.Errorf("ver: %v", err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{Titulo: "x"}); !errors.Is(err, boveda.ErrCerrada) {
		t.Errorf("guardar: %v", err)
	}
	if _, err := a.ExportarBoveda(); !errors.Is(err, boveda.ErrCerrada) {
		t.Errorf("exportar: %v", err)
	}
}

// Borrar la bóveda es lo más destructivo que hace el programa, así que lo que se
// comprueba es lo de siempre en estos casos: **que no borre cuando no debe y que
// borre del todo cuando debe.**
func TestBorrarLaBoveda(t *testing.T) {
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("la contraseña de verdad"); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{Titulo: "Banco", Secreto: "s3cr3t0"}); err != nil {
		t.Fatal(err)
	}

	ruta := rutaBovedaPrincipal()
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("la bóveda no está donde debería: %v", err)
	}

	// Con la contraseña equivocada no se borra nada.
	if err := a.BorrarBoveda("la que no es"); err == nil {
		t.Fatal("ha borrado la bóveda sin saber la contraseña")
	}
	if _, err := os.Stat(ruta); err != nil {
		t.Fatal("ha borrado el fichero aunque ha dicho que no")
	}
	if !a.EstadoBoveda().Abierta {
		t.Error("un intento fallido ha cerrado la bóveda")
	}

	// Con la buena, se va entera.
	if err := a.BorrarBoveda("la contraseña de verdad"); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoBoveda(); e.Existe || e.Abierta {
		t.Errorf("después de borrar: %+v", e)
	}

	// **Y no queda ninguna copia detrás**, que es la mitad del trabajo: el
	// `.anterior` existe para sobrevivir a un desastre, y aquí sería un desastre a
	// medias. Los temporales llevan una bóveda entera dentro.
	entradas, err := os.ReadDir(filepath.Dir(ruta))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		if strings.Contains(e.Name(), "boveda") || strings.HasPrefix(e.Name(), escritura.Prefijo) {
			t.Errorf("ha quedado %q detrás", e.Name())
		}
	}

	// Y se puede volver a empezar de cero, que es para lo que se borra.
	if _, err := a.CrearBoveda("otra contraseña distinta"); err != nil {
		t.Errorf("no deja crear otra después de borrar: %v", err)
	}
}

func TestNoSePuedeBorrarLoQueNoHay(t *testing.T) {
	a, _, _ := conReloj(t)
	if err := a.BorrarBoveda("lo que sea"); err == nil {
		t.Error("dice que ha borrado una bóveda que no existe")
	}
}

// **Un fallo que no es la contraseña deja escrito que no lo es.**
//
// `BorrarBoveda`, `SalirDeCuenta`, `CambiarMaestraDeBoveda` y entrar en la cuenta
// comprueban la contraseña abriendo la bóveda, y `Abrir` falla por más cosas que
// por la llave: un fichero cortado, una bóveda escrita por una versión más nueva,
// un JSON que esta versión no sabe leer. Las cuatro contestaban «esa no es la
// contraseña de esta bóveda», que **no es un mensaje impreciso sino uno que señala
// a otro sitio**: con la 2.30.0 se buscó una tarde una contraseña equivocada que
// era la correcta.
//
// Lo que se comprueba aquí es lo que no puede cambiar —la ventana sigue viendo
// siempre lo mismo, porque decirle a quien prueba contraseñas en qué ha fallado es
// ayudarle— y lo que sí: que el motivo de verdad quede registrado.
func TestUnFalloQueNoEsLaContrasenaSeRegistra(t *testing.T) {
	a, _, _ := conReloj(t)
	if _, err := a.CrearBoveda("la contraseña de verdad"); err != nil {
		t.Fatal(err)
	}
	ruta := rutaBovedaPrincipal()

	dicho := func(f func() error) (string, string) {
		t.Helper()
		var registro bytes.Buffer
		antes := log.Writer()
		log.SetOutput(&registro)
		defer log.SetOutput(antes)
		err := f()
		if err == nil {
			t.Fatal("no ha fallado, y tenía que fallar")
		}
		return err.Error(), registro.String()
	}

	// 1. La contraseña, de verdad, equivocada: **no se registra nada**. Si no, el
	//    registro se llena de intentos fallidos, que es exactamente lo que no
	//    interesa guardar de un gestor de contraseñas.
	visto, apuntado := dicho(func() error { return a.BorrarBoveda("la que no es") })
	if visto != "Esa no es la contraseña de esta bóveda" {
		t.Errorf("la ventana ve %q", visto)
	}
	if apuntado != "" {
		t.Errorf("una contraseña fallida ha dejado rastro: %q", apuntado)
	}

	// 2. Y ahora el fichero, roto por dentro: lo que ve la ventana **no cambia** y
	//    el motivo sí se apunta.
	roto, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, roto[:len(roto)/2], 0o600); err != nil {
		t.Fatal(err)
	}

	visto, apuntado = dicho(func() error { return a.BorrarBoveda("la contraseña de verdad") })
	if visto != "Esa no es la contraseña de esta bóveda" {
		t.Errorf("la ventana ve %q, y tiene que ver lo mismo que con una contraseña mala", visto)
	}
	if !strings.Contains(apuntado, "borrar la bóveda") || !strings.Contains(apuntado, "no es por la contraseña") {
		t.Errorf("el motivo no ha quedado escrito: %q", apuntado)
	}

	// Y el fichero sigue ahí: un fallo al comprobar nunca borra.
	if _, err := os.Stat(ruta); err != nil {
		t.Errorf("ha borrado la bóveda sin poder comprobar la contraseña: %v", err)
	}
}

// **Sin llaves de acceso no se pregunta dónde guardar, y no se escribe nada.**
//
// La 2.32.0 salió al revés: `ExportarLlaves` abría el diálogo del sistema y
// **después** miraba si había algo que exportar, así que con la bóveda sin llaves
// —que es como sale de fábrica— preguntaba dónde guardar un fichero que nunca iba
// a existir. Lo vio el cliente en su Mac la misma tarde.
//
// Lo que se comprueba son las dos mitades: que **no se llega a preguntar** —eso es
// lo que se vio— y que **no queda ningún fichero**, que es lo que de verdad
// importaría si alguien llega a elegir un sitio.
func TestSinLlavesNoSePreguntaDondeGuardar(t *testing.T) {
	a, s, _ := conReloj(t)
	if _, err := a.CrearBoveda("la contraseña de verdad"); err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{Titulo: "Banco", Secreto: "s3cr3t0"}); err != nil {
		t.Fatal(err)
	}

	destino := filepath.Join(t.TempDir(), "llaves.esf")
	s.guardaEn = destino
	s.vecesGuardar = 0

	if _, err := a.ExportarLlaves("una clave cualquiera"); err == nil {
		t.Fatal("ha exportado llaves que no existen")
	}
	// **Se cuentan las veces, no el argumento.** Mirando `desdeGuardar` esta prueba
	// pasaba con el fallo dentro: viene vacío hasta que alguien recuerda una
	// carpeta, así que «no me han llamado» y «me han llamado» se ven igual.
	if s.vecesGuardar != 0 {
		t.Error("ha abierto el diálogo de guardar antes de saber si había algo que guardar")
	}
	if _, err := os.Stat(destino); err == nil {
		t.Error("ha dejado un fichero detrás")
	}

	// Y con una llave dentro sí pregunta y sí escribe, que es la otra mitad: es
	// fácil arreglar esto cortando por lo sano y dejando la exportación muerta.
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Titulo: "GitHub", Tipo: boveda.TipoLlave, RPID: "github.com",
		IDCredencial: "Y3JlZC0x", NombreVisible: "yo@ejemplo.com",
		Algoritmo: -7, ClavePrivada: "cHJpdmFkYQ",
	}); err != nil {
		t.Fatal(err)
	}
	donde, err := a.ExportarLlaves("una clave cualquiera")
	if err != nil {
		t.Fatal(err)
	}
	if donde != destino {
		t.Errorf("dice que la ha dejado en %q", donde)
	}
	if _, err := os.Stat(destino); err != nil {
		t.Errorf("no ha escrito el fichero: %v", err)
	}
}
