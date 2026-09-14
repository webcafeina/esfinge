package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/codigos"
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

// El navegador sabe **si** una cuenta tiene segundo factor, y nada más de él.
//
// Sin esto el panel enseñaba «Código» en todas las cuentas y fallaba en las que
// no lo tienen. Lo que se comprueba de verdad es la segunda mitad: que la semilla
// no aparece en lo que cruza el canal, mirando el JSON y no el tipo.
func TestElNavegadorSabeSiHayCodigoPeroNoLaSemilla(t *testing.T) {
	_, _, _, f, _ := conBoveda(t)

	banco, err := f.CuentasDe("banco.es")
	if err != nil || len(banco) != 1 {
		t.Fatalf("cuentas del banco: %+v, %v", banco, err)
	}
	if !banco[0].TieneCodigo {
		t.Error("la cuenta del banco guarda semilla y dice que no tiene código")
	}
	if crudo, _ := json.Marshal(banco); strings.Contains(string(crudo), "GEZDGNBVGY3TQOJQ") {
		t.Errorf("la semilla ha cruzado el canal: %s", crudo)
	}

	correo, _ := f.CuentasDe("correo.com")
	if len(correo) != 1 || correo[0].TieneCodigo {
		t.Errorf("una cuenta sin semilla dice que tiene código: %+v", correo)
	}
}

// El código que sale para rellenar es el de ahora, con lo que le queda de vida, y
// solo sale de una entrada que tenga semilla.
func TestElCodigoParaRellenarEsElDeAhora(t *testing.T) {
	_, _, _, f, ids := conBoveda(t)

	antes := time.Now()
	c, err := f.RellenarCodigo(ids["Banco"], "banco.es")
	despues := time.Now()
	if err != nil {
		t.Fatal(err)
	}
	semilla, err := codigos.Leer("GEZDGNBVGY3TQOJQ")
	if err != nil {
		t.Fatal(err)
	}
	// Se compara con el de antes y el de después de pedirlo: si la llamada cae justo
	// en el cambio de periodo, cualquiera de los dos es correcto.
	a, _ := semilla.En(antes)
	d, _ := semilla.En(despues)
	if c.Codigo != a && c.Codigo != d {
		t.Errorf("código %q, y en ese momento valían %q o %q", c.Codigo, a, d)
	}
	if c.Quedan < 0 || c.Quedan > 30 {
		t.Errorf("le quedan %d segundos, que no cabe en un periodo de treinta", c.Quedan)
	}

	if _, err := f.RellenarCodigo(ids["Correo"], "correo.com"); err == nil {
		t.Error("una entrada sin semilla ha entregado un código")
	}
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
		if _, err := f.RellenarCodigo(ids["Banco"], dominio); err == nil {
			t.Errorf("el código del banco ha salido para rellenar en «%s»", dominio)
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
	if _, err := f.RellenarCodigo(ids["Banco"], "banco.es"); err == nil {
		t.Error("una entrada de la papelera ha entregado su código para rellenar")
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
		f.RellenarCodigo(ids["Banco"], "banco.es")
		// Y lo de guardar desde la página, que tampoco es alguien delante de la ventana.
		f.Ofrecer("https://banco.es", "banco.es", navegador.Envio{Usuario: "yo@ejemplo.es", Secreto: "otra"})
		f.ActualizarCuenta(ids["Banco"], "banco.es", navegador.Envio{Secreto: "otra"})
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

// ------------------------------------------------ guardar desde la página

// Qué se ofrece después de enviar un formulario, en cada caso, **sin que salga ningún
// secreto** en la oferta.
func TestQueSeOfreceDespuesDeEnviarUnFormulario(t *testing.T) {
	_, _, _, f, _ := conBoveda(t)
	casos := []struct {
		nombre string
		envio  navegador.Envio
		accion string
		cuenta string
	}{
		{"una cuenta que no está",
			navegador.Envio{Usuario: "nuevo@ejemplo.es", Secreto: "x", Forma: navegador.FormaEntrar},
			navegador.OfertaGuardar, ""},
		{"la misma cuenta con la misma contraseña, aunque cambien mayúsculas y espacios",
			navegador.Envio{Usuario: " YO@ejemplo.es", Secreto: "s3cr3t0", Forma: navegador.FormaEntrar},
			navegador.OfertaNada, ""},
		{"la misma cuenta con otra contraseña",
			navegador.Envio{Usuario: "yo@ejemplo.es", Secreto: "otra", Forma: navegador.FormaEntrar},
			navegador.OfertaActualizar, "Banco"},
		{"cambiar la contraseña sin usuario",
			navegador.Envio{Secreto: "nueva", Forma: navegador.FormaCambio},
			navegador.OfertaActualizar, "Banco"},
		{"sin contraseña no hay nada que ofrecer",
			navegador.Envio{Usuario: "yo@ejemplo.es", Forma: navegador.FormaEntrar},
			navegador.OfertaNada, ""},
	}
	for _, c := range casos {
		o, err := f.Ofrecer("https://www.banco.es/entrar", "banco.es", c.envio)
		if err != nil {
			t.Fatalf("%s: %v", c.nombre, err)
		}
		if o.Accion != c.accion {
			t.Errorf("%s: ofrece %q y tocaba %q", c.nombre, o.Accion, c.accion)
		}
		if c.cuenta != "" && (len(o.Cuentas) != 1 || o.Cuentas[0].Titulo != c.cuenta) {
			t.Errorf("%s: cuentas candidatas %+v", c.nombre, o.Cuentas)
		}
		if o.Accion == navegador.OfertaGuardar && o.Titulo != "Banco" {
			t.Errorf("%s: título sugerido %q", c.nombre, o.Titulo)
		}
		if crudo, _ := json.Marshal(o); strings.Contains(string(crudo), "s3cr3t0") {
			t.Errorf("%s: la oferta lleva la contraseña guardada: %s", c.nombre, crudo)
		}
	}
}

// **Guardar desde el navegador pone el sitio del origen y ningún otro**, con el título
// sugerido si no se ha escrito uno, y la ventana se entera.
func TestGuardarDesdeElNavegadorPoneElSitioDelOrigen(t *testing.T) {
	a, s, _, f, _ := conBoveda(t)
	c, err := f.GuardarCuenta("https://login.nuevo-sitio.es/entrar?x=1", "nuevo-sitio.es",
		navegador.Envio{Usuario: " yo@nuevo.es ", Secreto: "clave nueva"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Titulo != "Nuevo-sitio" || c.Usuario != "yo@nuevo.es" {
		t.Errorf("lo guardado: %+v", c)
	}

	var id string
	lista, _ := a.BuscarEnBoveda("")
	for _, e := range lista {
		if e.Usuario == "yo@nuevo.es" {
			id = e.ID
		}
	}
	if id == "" {
		t.Fatal("la cuenta guardada no está en la bóveda")
	}
	e, err := a.VerDeBoveda(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Sitios) != 1 || e.Sitios[0] != "https://login.nuevo-sitio.es" {
		t.Errorf("sitios guardados: %v", e.Sitios)
	}
	if e.Secreto != "clave nueva" || e.Tipo != boveda.TipoCredencial {
		t.Errorf("entrada guardada: tipo %q", e.Tipo)
	}
	if !s.hanAvisadoDe(EventoBovedaCambiada) {
		t.Error("la ventana no se ha enterado de la cuenta nueva")
	}
}

// Actualizar desde el navegador **solo desde el sitio de la cuenta**, y la anterior
// pasa al historial de contraseñas anteriores.
func TestActualizarDesdeElNavegadorGuardaLaAnterior(t *testing.T) {
	a, _, _, f, ids := conBoveda(t)
	if _, err := f.ActualizarCuenta(ids["Banco"], "correo.com", navegador.Envio{Secreto: "robada"}); err == nil {
		t.Fatal("ha cambiado la contraseña del banco desde otro sitio")
	}
	if _, err := f.ActualizarCuenta(ids["Banco"], "banco.es", navegador.Envio{Secreto: "nueva"}); err != nil {
		t.Fatal(err)
	}
	e, err := a.VerDeBoveda(ids["Banco"])
	if err != nil {
		t.Fatal(err)
	}
	if e.Secreto != "nueva" {
		t.Errorf("la contraseña no ha cambiado")
	}
	if len(e.Historial) == 0 || e.Historial[0].Secreto != "s3cr3t0" {
		t.Errorf("la anterior no está en el historial: %+v", e.Historial)
	}
}

// «Nunca en este sitio»: deja de ofrecerse, se ve en Ajustes y se deshace.
func TestNuncaEnEsteSitio(t *testing.T) {
	a, _, _, f, _ := conBoveda(t)
	envio := navegador.Envio{Usuario: "nuevo@ejemplo.es", Secreto: "x", Forma: navegador.FormaEntrar}
	if err := f.NuncaAqui("banco.es"); err != nil {
		t.Fatal(err)
	}
	if o, _ := f.Ofrecer("https://banco.es", "banco.es", envio); o.Accion != navegador.OfertaNada {
		t.Errorf("en un sitio excluido se ofrece %q", o.Accion)
	}
	if l := a.SitiosExcluidos(); len(l) != 1 || l[0] != "banco.es" {
		t.Errorf("Ajustes ve %v", l)
	}
	if err := a.QuitarSitioExcluido("banco.es"); err != nil {
		t.Fatal(err)
	}
	if o, _ := f.Ofrecer("https://banco.es", "banco.es", envio); o.Accion != navegador.OfertaGuardar {
		t.Errorf("tras quitar la exclusión se ofrece %q", o.Accion)
	}
	if l := a.SitiosExcluidos(); l == nil {
		t.Error("una lista vacía llega como nula, y a la ventana como null")
	}
}
