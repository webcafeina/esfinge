package app

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"os"
	"path/filepath"
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

// El freno de los Ajustes apaga las llaves de acceso **en las tres puertas**
// (ADR 0048).
//
// Es el freno de emergencia de toda la fase: existe para que un sitio que se rompa
// se pueda arreglar apagando un interruptor, sin esperar a una versión en una
// tienda. Por eso no basta con que el banner no salga.
//
//   - `Llaves` tiene que contestar **que no hay ninguna, sin error**: el `shim` cede
//     y sale el diálogo del navegador, que es lo que se vería sin Esfinge. Con un
//     error, el banner diría algo, y lo que tiene que pasar es que Esfinge no se
//     note.
//   - `DominiosConLlave` tiene que quedarse vacía, o el navegador seguiría creyendo
//     que aquí hay algo y con la bóveda cerrada sacaría «abre Esfinge» para nada.
//   - Y `FirmarLlave` tiene que negarse, aunque nadie deba llegar ahí: un freno que
//     solo frena donde se ofrece, y no donde se hace lo consecuente, no es un freno.
//
// Las tres se miran porque las tres se pueden olvidar por separado, y quitar
// cualquiera de ellas deja las otras dos en verde.
func TestElFrenoDeLosAjustesApagaLasLlaves(t *testing.T) {
	a, _, _, f, _ := conBoveda(t)
	// **Una llave de verdad, y no una cualquiera con la clave inventada.** Con una
	// inventada, `FirmarLlave` falla igual por no poder leerla y la comprobación del
	// freno se puede quitar entera sin que nada se ponga rojo: comprobado mutándolo.
	privada, _, err := navegador.CrearLlave()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Titulo: "GitHub", Tipo: boveda.TipoLlave, RPID: "github.com",
		IDCredencial: "Y3JlZC0x", IDUsuario: "dXN1YXJpbw", NombreVisible: "yo@ejemplo.com",
		Algoritmo: -7, ClavePrivada: navegador.B64URL.EncodeToString(privada),
	}); err != nil {
		t.Fatal(err)
	}
	var laLlave string
	for _, e := range a.boveda().Buscar("") {
		if e.Tipo == boveda.TipoLlave {
			laLlave = e.ID
		}
	}

	// Y firmar con ella funciona **antes** de apagar, que es la otra mitad: sin esto,
	// «no ha firmado» al final no distingue el freno de una llave que no servía.
	if _, err := f.FirmarLlave("https://github.com/login", "github.com", laLlave, "cmV0bw"); err != nil {
		t.Fatalf("con el interruptor puesto no firma: %v", err)
	}

	// Encendido —como viene de fábrica— se ve la llave y el dominio se apunta.
	hay, err2 := f.Llaves("https://github.com/login", "github.com", nil)
	if err2 != nil || len(hay) != 1 {
		t.Fatalf("con el interruptor puesto no sale la llave: %+v, %v", hay, err2)
	}
	if d := f.DominiosConLlave(); len(d) != 1 || d[0] != "github.com" {
		t.Fatalf("con el interruptor puesto los dominios son %v", d)
	}

	p := a.ajustes.Ver()
	p.LlavesDeAccesoEnElNavegador = false
	if err := a.GuardarPreferencias(p); err != nil {
		t.Fatal(err)
	}

	apagadas, err := f.Llaves("https://github.com/login", "github.com", nil)
	if err != nil {
		t.Errorf("apagado tiene que contestar que no hay, no fallar: %v", err)
	}
	if len(apagadas) != 0 {
		t.Errorf("apagado sigue ofreciendo %d llaves", len(apagadas))
	}
	if d := f.DominiosConLlave(); len(d) != 0 {
		t.Errorf("apagado sigue apuntando los dominios %v", d)
	}
	if _, err := f.FirmarLlave("https://github.com/login", "github.com", laLlave, "cmV0bw"); err == nil {
		t.Error("apagado ha firmado igual")
	}
}

// Lo que se rechaza al crear, y **cada cosa por su motivo** (ADR 0048, P3).
//
// Son las cuatro puertas de antes de tocar la bóveda, y se miran por separado porque
// cada una se puede quitar sola. La del algoritmo es la que casi se queda sin probar:
// quitarla del código **no hacía caer nada** —la prueba del ciclo le pasa `-7`, que
// es lo que sí sabemos— y la mutación solo se puso roja porque dejaba un `import` sin
// usar. Eso no es una prueba, es una casualidad del compilador.
func TestLoQueNoSeCreaYPorQue(t *testing.T) {
	a, _, _, f, _ := conBoveda(t)
	reto := navegador.B64URL.EncodeToString([]byte("un reto cualquiera"))

	// **Un algoritmo que Esfinge no sabe firmar.** Crear la llave la registraría en el
	// sitio y la cuenta se quedaría con una credencial muerta, así que no se crea.
	if _, err := f.CrearLlave("https://github.com/", "github.com", navegador.LlaveNueva{
		Reto: reto, Algoritmos: []int{-257, -8},
	}); err == nil {
		t.Error("ha creado una llave con un algoritmo que no sabe firmar")
	}
	// Y una lista vacía es «me da igual», que sí vale.
	if _, err := f.CrearLlave("https://github.com/", "github.com", navegador.LlaveNueva{Reto: reto}); err != nil {
		t.Errorf("sin lista de algoritmos tenía que crearla: %v", err)
	}

	// **Un sitio pidiendo para otro dominio.** Es la misma regla que al firmar, y aquí
	// es peor: crearía en la bóveda una llave atada a un sitio que no la pidió.
	for _, origen := range []string{"https://malo.com/", "https://github.com.malo.com/", "https://malogithub.com/"} {
		if _, err := f.CrearLlave(origen, "github.com", navegador.LlaveNueva{Reto: reto}); err == nil {
			t.Errorf("desde %q ha creado una llave de github.com", origen)
		}
	}

	// **Un reto que no es base64url.** Se mira antes de generar nada.
	if _, err := f.CrearLlave("https://github.com/", "github.com", navegador.LlaveNueva{
		Reto: "esto no es base64url!!",
	}); err == nil {
		t.Error("ha creado una llave con un reto que no se entiende")
	}

	// **Y si el sitio dice que ya tiene una llave nuestra, no se hace otra.** Se coge
	// el identificador de la que se acaba de crear, que es lo que el sitio mandaría.
	var suya string
	for _, e := range a.boveda().Buscar("") {
		if e.Tipo == boveda.TipoLlave {
			entera, _ := a.boveda().Ver(e.ID)
			suya = entera.IDCredencial
		}
	}
	if suya == "" {
		t.Fatal("no hay ninguna llave con la que probar la exclusión")
	}
	if _, err := f.CrearLlave("https://github.com/", "github.com", navegador.LlaveNueva{
		Reto: reto, Excluidas: []string{suya},
	}); err == nil {
		t.Error("ha creado una segunda llave para una cuenta que ya tenía la nuestra")
	}
	// Pero una excluida que **no** es nuestra no impide nada: es la llave que esa
	// persona tenga en su Touch ID, y ahí Esfinge sí puede ofrecer la suya.
	if _, err := f.CrearLlave("https://github.com/", "github.com", navegador.LlaveNueva{
		Reto: reto, Excluidas: []string{navegador.B64URL.EncodeToString([]byte("la del sistema"))},
	}); err != nil {
		t.Errorf("una llave excluida que no es nuestra no tenía que estorbar: %v", err)
	}
}

// Crear una llave de acceso y **poder firmar con ella** (ADR 0048, P3).
//
// Es el ciclo que importa y el que ninguna prueba de las piezas cubre: lo que se
// guarda al crear tiene que ser exactamente lo que hace falta para firmar después.
// Si faltara un campo —el identificador, la privada, el `rpId`— la llave se
// registraría en el sitio y **la cuenta se quedaría sin forma de entrar**, que es el
// peor fallo posible de esta clase.
//
// Y se comprueba **verificando la firma contra la pública que salió en la
// atestación**, no contra la de la bóveda: la que cuenta es la que el sitio se
// guardó. Con las dos distintas, todo estaría en verde y nadie podría entrar.
func TestCrearUnaLlaveYFirmarConElla(t *testing.T) {
	a, _, _, f, _ := conBoveda(t)

	at, err := f.CrearLlave("https://github.com/registro", "github.com", navegador.LlaveNueva{
		Usuario:    "yo@ejemplo.com",
		IDUsuario:  navegador.B64URL.EncodeToString([]byte("el-usuario-del-sitio")),
		Titulo:     "GitHub",
		Reto:       navegador.B64URL.EncodeToString([]byte("un reto de treinta y dos bytes.")),
		Algoritmos: []int{-7},
	})
	if err != nil {
		t.Fatalf("no ha creado la llave: %v", err)
	}
	if at.IDCredencial == "" || at.Objeto == "" || at.DatosDelCliente == "" {
		t.Fatalf("la atestación sale incompleta: %+v", at)
	}

	// **Lo que el sitio recibe dice lo que tiene que decir.** El tipo es el de crear
	// —no el de firmar— y el origen lo puso este lado, sin el puerto por defecto.
	cliente, err := navegador.B64URL.DecodeString(at.DatosDelCliente)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cliente), `"type":"webauthn.create"`) {
		t.Errorf("el clientDataJSON no dice que se está creando: %s", cliente)
	}
	if !strings.Contains(string(cliente), `"origin":"https://github.com"`) {
		t.Errorf("el origen no es el que puso este lado: %s", cliente)
	}

	// La entrada está en la bóveda, es del sitio y **no se ve la privada en la lista**.
	var laLlave boveda.Entrada
	for _, e := range a.boveda().Buscar("") {
		if e.Tipo == boveda.TipoLlave {
			laLlave, _ = a.boveda().Ver(e.ID)
			if e.ClavePrivada != "" {
				t.Error("la clave privada sale en lo que se enseña")
			}
		}
	}
	if laLlave.ID == "" {
		t.Fatal("la llave no se ha guardado en la bóveda")
	}
	if laLlave.RPID != "github.com" || laLlave.IDCredencial != at.IDCredencial {
		t.Errorf("lo guardado no cuadra con lo que se le dijo al sitio: %+v", laLlave)
	}
	if laLlave.Algoritmo != -7 || laLlave.ClavePrivada == "" {
		t.Errorf("la llave se ha guardado sin lo que hace falta para firmar: %+v", laLlave)
	}
	idDeCredencial, err := navegador.B64URL.DecodeString(at.IDCredencial)
	if err != nil {
		t.Fatal(err)
	}

	// **Y ahora lo que de verdad prueba esto**, en dos mitades y sin descodificar
	// CBOR —que a propósito no existe en este proyecto—:
	//
	//  1. que **la pública que se le mandó al sitio es la de la llave guardada**, o
	//     sea que el COSE que va dentro de la atestación son exactamente esos bytes;
	//  2. y que una firma hecha con la llave guardada **se verifica con esa pública**.
	//
	// Las dos juntas dicen lo que hará el sitio la próxima vez que se entre. Solo la
	// segunda no diría nada: verificar con la pública de la privada que acabo de usar
	// para firmar es comprobar que ECDSA funciona.
	privadaGuardada, err := navegador.B64URL.DecodeString(laLlave.ClavePrivada)
	if err != nil {
		t.Fatal(err)
	}
	k, err := x509.ParsePKCS8PrivateKey(privadaGuardada)
	if err != nil {
		t.Fatal(err)
	}
	suya := k.(*ecdsa.PrivateKey)
	publicaSPKI, err := x509.MarshalPKIXPublicKey(&suya.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	coseGuardada, err := navegador.PublicaEnCOSE(publicaSPKI)
	if err != nil {
		t.Fatal(err)
	}
	objeto, err := navegador.B64URL.DecodeString(at.Objeto)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(objeto, coseGuardada) {
		t.Error("la pública que se le mandó al sitio no es la de la llave guardada: " +
			"la cuenta se quedaría con una llave con la que no se puede entrar")
	}
	// **Y el identificador también va dentro**, que es lo otro que el sitio guarda.
	if !bytes.Contains(objeto, idDeCredencial) {
		t.Error("el identificador de la credencial no está en la atestación")
	}

	af, err := f.FirmarLlave("https://github.com/login", "github.com", laLlave.ID,
		navegador.B64URL.EncodeToString([]byte("otro reto cualquiera")))
	if err != nil {
		t.Fatalf("no ha firmado con la llave que acaba de crear: %v", err)
	}
	datos, err := navegador.B64URL.DecodeString(af.DatosDelAutenticador)
	if err != nil {
		t.Fatal(err)
	}
	cli, err := navegador.B64URL.DecodeString(af.DatosDelCliente)
	if err != nil {
		t.Fatal(err)
	}
	firma, err := navegador.B64URL.DecodeString(af.Firma)
	if err != nil {
		t.Fatal(err)
	}
	if !navegador.VerificarFirma(publicaSPKI, navegador.LoQueSeFirma(datos, cli), firma) {
		t.Error("el sitio no podría verificar la firma de esa llave")
	}
}

// **Una llave queda confirmada cuando el sitio la nombra, y solo entonces** (ADR 0048).
//
// Es lo que distingue una llave huérfana —creada aquí y que el sitio nunca registró—
// de una que sirve. Y la mitad que importa es la segunda: **firmar no confirma**.
//
// Con `allowCredentials` vacío el sitio no dice qué tiene y Esfinge ofrece las suyas,
// así que una huérfana se firmaría igual y el sitio la rechazaría después. Si bastara
// con haber firmado, la marca diría «buena» de la que no lo es, que es peor que no
// tenerla: daría permiso para borrar la equivocada.
func TestUnaLlaveSeConfirmaCuandoElSitioLaNombra(t *testing.T) {
	a, _, _, f, _ := conBoveda(t)
	privada, _, err := navegador.CrearLlave()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.GuardarEnBoveda(boveda.Entrada{
		Titulo: "GitHub", Tipo: boveda.TipoLlave, RPID: "github.com",
		IDCredencial: "Y3JlZC0x", NombreVisible: "yo@ejemplo.com",
		Algoritmo: -7, ClavePrivada: navegador.B64URL.EncodeToString(privada),
	}); err != nil {
		t.Fatal(err)
	}
	laLlave := func() boveda.Entrada {
		for _, e := range a.boveda().Buscar("") {
			if e.Tipo == boveda.TipoLlave {
				entera, _ := a.boveda().Ver(e.ID)
				return entera
			}
		}
		t.Fatal("la llave no está")
		return boveda.Entrada{}
	}

	// Recién creada, sin confirmar.
	if laLlave().Confirmada != "" {
		t.Error("una llave recién guardada no puede estar confirmada: el sitio no la ha pedido")
	}

	// **El aviso de la página no confirma**: ahí no hay `rpId` ni lista, solo se
	// pregunta si hay algo que ofrecer.
	if _, err := f.Llaves("https://github.com/login", "", nil); err != nil {
		t.Fatal(err)
	}
	if laLlave().Confirmada != "" {
		t.Error("preguntar qué hay ha confirmado la llave sin que el sitio diga nada")
	}

	// **Y firmar tampoco**, que es la mitad que de verdad se puede equivocar: el sitio
	// pide sin lista, Esfinge ofrece la suya y se firma, y eso no dice que la tenga.
	if _, err := f.Llaves("https://github.com/login", "github.com", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.FirmarLlave("https://github.com/login", "github.com", laLlave().ID,
		navegador.B64URL.EncodeToString([]byte("un reto"))); err != nil {
		t.Fatal(err)
	}
	if laLlave().Confirmada != "" {
		t.Error("firmar ha confirmado la llave, y firmar no prueba que el sitio la tenga registrada")
	}

	// **Pero firmar sí apunta que se ha usado**, que es la otra señal y la que de verdad
	// se rellena: en el «entrar con llave de acceso» de un sitio que no nombra ninguna,
	// ésta es la única que llega. Lo vio el cliente con la 2.35.0, cuya ficha seguía
	// diciendo que el sitio no había pedido la llave con la llave funcionando.
	usada := laLlave().Usada
	if usada == "" {
		t.Fatal("se ha firmado con la llave y no se ha apuntado cuándo")
	}

	// **Y se actualiza, que es lo que la separa de `Confirmada`**: lo útil de una fecha
	// de uso es la última. Se comprueba envejeciéndola a mano en vez de esperar un
	// segundo de reloj, que sería una prueba lenta y además intermitente.
	vieja := laLlave()
	vieja.Usada = "2020-01-01T00:00:00Z"
	if err := a.boveda().Poner(vieja); err != nil {
		t.Fatal(err)
	}
	if _, err := f.FirmarLlave("https://github.com/login", "github.com", laLlave().ID,
		navegador.B64URL.EncodeToString([]byte("otro reto"))); err != nil {
		t.Fatal(err)
	}
	if laLlave().Usada == "2020-01-01T00:00:00Z" {
		t.Error("la fecha de uso no se ha actualizado: dice la primera vez y tiene que decir la última")
	}

	// Y con la misma fecha **no se escribe en la bóveda**: las fechas tienen resolución
	// de un segundo y dos firmas seguidas no pueden costar dos escrituras y dos subidas.
	antesDeFirmar := laLlave()
	if _, err := f.FirmarLlave("https://github.com/login", "github.com", laLlave().ID,
		navegador.B64URL.EncodeToString([]byte("y otro"))); err != nil {
		t.Fatal(err)
	}
	if ahora := laLlave(); ahora.Usada == antesDeFirmar.Usada && ahora.Revision != antesDeFirmar.Revision {
		t.Error("se ha escrito en la bóveda sin que la fecha de uso cambiara")
	}

	// **Lo que sí la confirma**: que el sitio la nombre en `allowCredentials`.
	if _, err := f.Llaves("https://github.com/login", "github.com", []string{"Y3JlZC0x"}); err != nil {
		t.Fatal(err)
	}
	primera := laLlave().Confirmada
	if primera == "" {
		t.Fatal("el sitio la ha nombrado y no se ha confirmado")
	}

	// Y no se vuelve a escribir: la marca es de la primera vez, no de la última.
	antes := laLlave().Revision
	if _, err := f.Llaves("https://github.com/login", "github.com", []string{"Y3JlZC0x"}); err != nil {
		t.Fatal(err)
	}
	if laLlave().Confirmada != primera {
		t.Error("la fecha se ha vuelto a escribir: dice cuándo se usó la última vez y no la primera")
	}
	if laLlave().Revision != antes {
		t.Error("se ha escrito en la bóveda otra vez con la llave ya confirmada")
	}
}

// ---------------------------------------- una bóveda compartida de solo ver (ADR 0052)

// conUnaCompartida deja abierta una bóveda **ajena** a la que me han dado acceso con el
// permiso que se diga, y devuelve la fuente que ve el navegador.
//
// Se monta dándome acceso a mí mismo, que es artificial y **recorre el camino de
// verdad**: la ranura se sella hacia la identidad de esta bóveda personal, la fila
// entra en la lista de compartidas como la deja la ventana al aceptar, el fichero va
// donde van las ajenas, y abrirla es `AbrirCompartida`. Así, si `conmutarACompartida`
// dejara de pasar el permiso, esto se pondría rojo — que es justo el agujero que una
// prueba con el estado puesto a mano no vería.
func conUnaCompartida(t *testing.T, permiso string) (*App, navegador.Fuente) {
	t.Helper()
	a, _, _, fuente, _ := conBoveda(t)

	const dueno = "aaaaaaaabbbbbbbbccccccccdddddddd"
	const titular = "1111222233334444"
	mia, err := a.boveda().Identidad()
	if err != nil {
		t.Fatal(err)
	}

	// La bóveda de la otra persona: un proyecto cualquiera con **mi** ranura dentro.
	ruta := rutaDeCompartida(dueno, refDePrueba)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	suya, err := boveda.CrearProyecto(ruta, []byte("una clave de bóveda que no es mía aaaa"))
	if err != nil {
		t.Fatal(err)
	}
	if err := suya.Poner(boveda.Entrada{
		Tipo: boveda.TipoCredencial, Titulo: "Banco", Usuario: "yo@ejemplo.es",
		Secreto: "la del cliente", Sitios: []string{"https://banco.es"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := suya.PonerAcceso(titular, mia); err != nil {
		t.Fatal(err)
	}
	suya.Cerrar()

	// Y la fila de mi lista, como la deja la ventana al aceptar el acceso.
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		return b.PonerCompartida(boveda.Compartida{
			Dueno: dueno, Ref: refDePrueba, Nombre: "Zeri", Titular: titular, Permiso: permiso,
		})
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.AbrirCompartida(dueno, refDePrueba); err != nil {
		t.Fatal(err)
	}
	return a, fuente
}

const refDePrueba = "a1b2c3d4e5f60718"

// **Con una compartida de solo ver, el navegador no ofrece guardar y no escribe.**
//
// No es por celo criptográfico: la clave la tengo y escribir funcionaría. Lo que pasa es
// que el servidor rechaza la subida con un 403, así que lo escrito se quedaría en este
// equipo para siempre — y la tarjeta de guardar la saca Esfinge por su cuenta, en la
// página de otro, donde no hay dónde explicar nada después.
//
// Lo que esta prueba **no** dice: que la ventana haga lo mismo. No lo hace, a propósito,
// y está escrito en `porQueNoSeEscribe`.
func TestConUnaCompartidaDeSoloVerElNavegadorNoEscribe(t *testing.T) {
	a, fuente := conUnaCompartida(t, "ver")

	if !a.soloPuedoVerLaActiva() {
		t.Fatal("con permiso de ver, la app tiene que decir que solo se puede ver")
	}

	// Nada que ofrecer, aunque el envío traiga una contraseña nueva de un sitio que no
	// está en la bóveda: eso en una bóveda propia sería «Guardar».
	o, err := fuente.Ofrecer("https://otrositio.com/entrar", "otrositio.com",
		navegador.Envio{Usuario: "yo@ejemplo.es", Secreto: "una nueva"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Accion != navegador.OfertaNada {
		t.Errorf("en una compartida de solo ver se ofrece %q", o.Accion)
	}

	// Y si alguien lo pide igual —una tarjeta vieja, un mensaje a mano—, se dice por qué
	// no, **con la frase que toca**: «actualiza Esfinge» mandaría a mirar donde no hay nada.
	_, err = fuente.GuardarCuenta("https://otrositio.com/entrar", "otrositio.com",
		navegador.Envio{Usuario: "yo@ejemplo.es", Secreto: "una nueva"})
	if err == nil {
		t.Fatal("ha guardado en una bóveda compartida de solo ver")
	}
	if !strings.Contains(err.Error(), "Solo puedes ver") {
		t.Errorf("el motivo que se da es %q", err)
	}
	if err := fuente.NuncaAqui("otrositio.com"); err == nil {
		t.Error("ha excluido un sitio en una bóveda compartida de solo ver")
	}
	if fuente.PuedeCrearLlaves() {
		t.Error("dice que puede crear una llave de acceso en una bóveda de solo ver")
	}

	// Leer sí, que es para lo que existe el acceso: la cuenta del cliente se rellena.
	cuentas, err := fuente.CuentasDe("banco.es")
	if err != nil {
		t.Fatal(err)
	}
	if len(cuentas) != 1 || cuentas[0].Usuario != "yo@ejemplo.es" {
		t.Fatalf("en la compartida se ven %d cuentas de banco.es: %+v", len(cuentas), cuentas)
	}
}

// **Y con permiso de editar, todo lo de siempre.** Es la mitad que dice que la puerta
// mira el permiso y no «es ajena»: sin esta prueba, negarlo todo en cualquier bóveda
// compartida pasaría en verde.
func TestConUnaCompartidaDeEditarElNavegadorEscribe(t *testing.T) {
	a, fuente := conUnaCompartida(t, "editar")

	if a.soloPuedoVerLaActiva() {
		t.Fatal("con permiso de editar no se puede decir que solo se ve")
	}
	o, err := fuente.Ofrecer("https://otrositio.com/entrar", "otrositio.com",
		navegador.Envio{Usuario: "yo@ejemplo.es", Secreto: "una nueva"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Accion != navegador.OfertaGuardar {
		t.Errorf("en una compartida de editar se ofrece %q y tenía que ofrecer guardar", o.Accion)
	}
	if _, err := fuente.GuardarCuenta("https://otrositio.com/entrar", "otrositio.com",
		navegador.Envio{Usuario: "yo@ejemplo.es", Secreto: "una nueva"}); err != nil {
		t.Fatalf("no ha guardado con permiso de editar: %v", err)
	}
	// Y lo guardado está **en la compartida**, no en la personal: es la comprobación que
	// distingue «ha escrito» de «ha escrito donde tocaba».
	if len(a.boveda().Buscar("otrositio")) != 1 {
		t.Error("lo guardado no está en la bóveda compartida")
	}
}

// **Y al salir de la compartida se olvida el permiso.** Si se quedara puesto, volver a
// la bóveda personal dejaría el navegador sin ofrecer guardar en ella **y sin que nada
// lo dijera**: el fallo sería «Esfinge ha dejado de ofrecerse» y la causa estaría tres
// pantallas atrás.
func TestVolverDeUnaCompartidaDevuelveElPermiso(t *testing.T) {
	a, fuente := conUnaCompartida(t, "ver")
	if err := a.VolverALaBovedaPersonal(); err != nil {
		t.Fatal(err)
	}
	if a.soloPuedoVerLaActiva() {
		t.Fatal("de vuelta en la personal sigue diciendo que solo se puede ver")
	}
	o, err := fuente.Ofrecer("https://otrositio.com/entrar", "otrositio.com",
		navegador.Envio{Usuario: "yo@ejemplo.es", Secreto: "una nueva"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Accion != navegador.OfertaGuardar {
		t.Errorf("de vuelta en la personal se ofrece %q", o.Accion)
	}
}
