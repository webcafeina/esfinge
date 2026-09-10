package app

import (
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/navegador"
)

// conBoveda deja una bóveda abierta con una credencial del banco y otra de un
// correo, y devuelve la fuente que ve el navegador.
func conBoveda(t *testing.T) (*App, *sistemaFalso, *time.Time, navegador.Fuente, map[string]string) {
	t.Helper()
	a, s, ahora := conReloj(t)
	if _, err := a.CrearBoveda("una contraseña maestra larga"); err != nil {
		t.Fatal(err)
	}
	for _, e := range []boveda.Entrada{
		{Titulo: "Banco", Usuario: "yo@ejemplo.es", Secreto: "s3cr3t0",
			Sitios: []string{"https://banco.es/particulares"}, TOTP: "GEZDGNBVGY3TQOJQ"},
		{Titulo: "Correo", Usuario: "otro@ejemplo.es", Secreto: "otra clave",
			Sitios: []string{"https://correo.com"}},
		// Una tarjeta con el mismo sitio: **no se ofrece**, porque no se rellena en
		// un formulario de inicio de sesión y enseñar su título sería contar algo de
		// la bóveda por gusto.
		{Titulo: "Tarjeta del banco", Tipo: boveda.TipoTarjeta, Numero: "4111111111111111",
			Sitios: []string{"https://banco.es"}},
	} {
		if err := a.GuardarEnBoveda(e); err != nil {
			t.Fatal(err)
		}
	}
	ids := map[string]string{}
	for _, e := range a.boveda().Buscar("") {
		ids[e.Titulo] = e.ID
	}
	return a, s, ahora, fuenteDelNavegador{a}, ids
}

// Lo que el navegador ve de un sitio: sus cuentas, sin secretos, y solo las suyas.
func TestLoQueElNavegadorVeDeUnSitio(t *testing.T) {
	_, _, _, f, _ := conBoveda(t)

	cuentas, err := f.CuentasDe("banco.es")
	if err != nil {
		t.Fatal(err)
	}
	if len(cuentas) != 1 || cuentas[0].Titulo != "Banco" || cuentas[0].Usuario != "yo@ejemplo.es" {
		t.Fatalf("cuentas de banco.es: %+v", cuentas)
	}

	// De un sitio que no está, nada. Ni siquiera una lista vacía con pistas.
	if c, _ := f.CuentasDe("otracosa.com"); len(c) != 0 {
		t.Errorf("ha ofrecido algo para un sitio que no tiene cuentas: %+v", c)
	}
}

// **La prueba que impide el desastre**: con el identificador de una entrada a
// mano, pedirla desde otro sitio no la da.
func TestUnaEntradaNoSaleParaUnSitioQueNoEsElSuyo(t *testing.T) {
	_, s, _, f, ids := conBoveda(t)

	if _, err := f.CopiarSecreto(ids["Banco"], "banco.es"); err != nil {
		t.Fatalf("desde su sitio no ha copiado: %v", err)
	}
	if s.verPortapapeles() != "s3cr3t0" {
		t.Fatalf("no ha copiado la contraseña: %q", s.verPortapapeles())
	}

	for _, dominio := range []string{"correo.com", "malo.com", "banco.es.malo.com", ""} {
		if _, err := f.CopiarSecreto(ids["Banco"], dominio); err == nil {
			t.Errorf("la contraseña del banco ha salido para «%s»", dominio)
		}
		if _, err := f.CopiarCodigo(ids["Banco"], dominio); err == nil {
			t.Errorf("el código de un solo uso del banco ha salido para «%s»", dominio)
		}
	}

	// Y un identificador inventado tampoco abre nada.
	if _, err := f.CopiarSecreto("me lo he inventado", "banco.es"); err == nil {
		t.Error("un identificador inventado ha devuelto una contraseña")
	}
}

// Lo que está en la papelera no se ofrece ni se entrega, aunque conserve su
// contenido durante treinta días (ADR 0026).
func TestLoBorradoNoLoVeElNavegador(t *testing.T) {
	a, _, _, f, ids := conBoveda(t)
	if err := a.BorrarDeBoveda(ids["Banco"]); err != nil {
		t.Fatal(err)
	}

	if c, _ := f.CuentasDe("banco.es"); len(c) != 0 {
		t.Errorf("una entrada de la papelera se sigue ofreciendo: %+v", c)
	}
	if _, err := f.CopiarSecreto(ids["Banco"], "banco.es"); err == nil {
		t.Error("una entrada de la papelera ha entregado su contraseña")
	}
	if _, err := f.Rellenar(ids["Banco"], "banco.es"); err == nil {
		t.Error("una entrada de la papelera ha entregado su contraseña para rellenar")
	}
}

// **Preguntar desde el navegador no cuenta como actividad.** Una extensión
// pregunta sola —al cambiar de pestaña, al revivir su trabajador— así que si esto
// moviera el reloj, navegar mantendría la bóveda abierta para siempre.
func TestElNavegadorNoMantieneLaBovedaAbierta(t *testing.T) {
	a, _, ahora, f, ids := conBoveda(t)

	for i := 0; i < 30; i++ {
		*ahora = ahora.Add(time.Minute)
		f.CuentasDe("banco.es")
		f.CopiarSecreto(ids["Banco"], "banco.es")
		f.CopiarCodigo(ids["Banco"], "banco.es")
		// Y rellenar tampoco, que es donde más se notaría: una página guardada
		// pregunta al cargarse, y navegar por sitios guardados es lo normal.
		f.Rellenar(ids["Banco"], "banco.es")
		f.Estado()
		a.repasar()
	}
	if a.EstadoBoveda().Abierta {
		t.Error("la bóveda sigue abierta después de media hora sin nadie delante: " +
			"preguntar desde el navegador está tocando el reloj del bloqueo")
	}
}

// El emparejamiento: se pide, lo contesta una persona en la ventana, y el testigo
// se entrega **una sola vez**.
func TestElEmparejamientoSePideYSeConcedeUnaVez(t *testing.T) {
	a, _, _, f, _ := conBoveda(t)

	if _, err := f.Emparejar("Chrome"); err == nil {
		t.Fatal("ha dado permiso sin que nadie lo permitiera")
	}
	// Y la ventana se entera de quién lo pide, para poder preguntarlo.
	if a.EstadoDelNavegador().Pide != "Chrome" {
		t.Errorf("la ventana no sabe quién pide: %+v", a.EstadoDelNavegador())
	}

	if err := a.PermitirNavegador(); err != nil {
		t.Fatal(err)
	}
	testigo, err := f.Emparejar("Chrome")
	if err != nil || testigo == "" {
		t.Fatalf("después del sí no ha salido testigo: %q, %v", testigo, err)
	}
	if !f.Emparejado(testigo) {
		t.Error("el testigo recién dado no vale")
	}

	// **Y no se puede volver a recoger.** Si se pudiera, a un programa cualquiera
	// le bastaría con pedir «emparejar» después de que la persona hubiera
	// permitido su navegador, y el permiso no valdría nada.
	if otro, err := f.Emparejar("Un programa cualquiera"); err == nil {
		t.Errorf("ha vuelto a entregar el permiso a quien lo pidiera: %q", otro)
	}

	// La ventana ve el permiso, **sin el testigo**.
	e := a.EstadoDelNavegador()
	if len(e.Permitidos) != 1 || e.Permitidos[0].Quien != "Chrome" {
		t.Fatalf("permitidos: %+v", e.Permitidos)
	}
	if e.Permitidos[0].Testigo != "" {
		t.Error("el testigo ha cruzado el puente hacia la ventana")
	}

	// Y se puede retirar.
	if err := a.OlvidarNavegador(e.Permitidos[0].Desde); err != nil {
		t.Fatal(err)
	}
	if f.Emparejado(testigo) {
		t.Error("el testigo sigue valiendo después de retirar el permiso")
	}
}

// El canal viene apagado y se enciende en Ajustes. Encendido y apagado tienen que
// valer **desde ya**, no desde el siguiente arranque.
func TestElCanalSeEnciendeYSeApagaDesdeAjustes(t *testing.T) {
	a, _, _, _, _ := conBoveda(t)

	if e := a.EstadoDelNavegador(); e.Encendido || e.Escuchando {
		t.Fatalf("viene encendido de fábrica: %+v", e)
	}

	p := a.VerPreferencias()
	p.PuenteDelNavegador = true
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	e := a.EstadoDelNavegador()
	if !e.Encendido || !e.Escuchando {
		t.Fatalf("no ha arrancado: %+v", e)
	}
	if !strings.HasSuffix(e.Donde, "puente.sock") {
		t.Errorf("no dice dónde escucha: %q", e.Donde)
	}

	p.PuenteDelNavegador = false
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}
	if e := a.EstadoDelNavegador(); e.Encendido || e.Escuchando {
		t.Errorf("sigue escuchando después de apagarlo: %+v", e)
	}
}
