package app

import (
	"errors"
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

	ruta := rutaBoveda()
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
