package navegador

import (
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// bovedaFalsa es una bóveda de mentira con dos cuentas, una de cada sitio.
type bovedaFalsa struct {
	existe, abierta bool
	testigos        map[string]bool
	niega           bool   // la persona dice que no al emparejar
	pedidos         int    // cuántas veces se ha preguntado por un secreto
	portapapeles    string // lo que Esfinge ha copiado
}

func nuevaFalsa() *bovedaFalsa {
	return &bovedaFalsa{existe: true, abierta: true, testigos: map[string]bool{"el-testigo": true}}
}

var lasEntradas = []struct {
	id, titulo, usuario, sitio, secreto, semilla string
}{
	{"1", "Banco", "yo@ejemplo.es", "https://banco.es", "s3cr3t0", "GEZDGNBVGY3TQOJQ"},
	{"2", "Correo", "otro@ejemplo.es", "https://correo.com", "otra clave", ""},
}

func (b *bovedaFalsa) Estado() Estado { return Estado{Existe: b.existe, Abierta: b.abierta} }

func (b *bovedaFalsa) CuentasDe(dominio string) ([]Cuenta, error) {
	var out []Cuenta
	for _, e := range lasEntradas {
		if Encaja(e.sitio, dominio) {
			out = append(out, Cuenta{ID: e.id, Titulo: e.titulo, Usuario: e.usuario})
		}
	}
	return out, nil
}

func (b *bovedaFalsa) CopiarSecreto(id, dominio string) (Copiado, error) {
	b.pedidos++
	for _, e := range lasEntradas {
		if e.id == id && Encaja(e.sitio, dominio) {
			b.portapapeles = e.secreto
			return Copiado{Portapapeles: 30}, nil
		}
	}
	return Copiado{}, errors.New("Esa entrada no es de ese sitio")
}

func (b *bovedaFalsa) CopiarCodigo(id, dominio string) (Copiado, error) {
	for _, e := range lasEntradas {
		if e.id == id && Encaja(e.sitio, dominio) && e.semilla != "" {
			b.portapapeles = "123456"
			return Copiado{Portapapeles: 30, Quedan: 20}, nil
		}
	}
	return Copiado{}, errors.New("Esa entrada no es de ese sitio")
}

func (b *bovedaFalsa) Emparejar(string) (string, error) {
	if b.niega {
		return "", errors.New("No se ha permitido")
	}
	b.testigos["nuevo"] = true
	return "nuevo", nil
}

func (b *bovedaFalsa) Emparejado(t string) bool { return b.testigos[t] }

func pedir(s *Servidor, p Peticion) Respuesta {
	p.Version = VersionDelProtocolo
	return s.Atender(p, nil)
}

// El camino bueno, entero: preguntar qué hay para un sitio y sacar una
// contraseña.
func TestElCaminoDeUnRelleno(t *testing.T) {
	b := nuevaFalsa()
	s := &Servidor{fuente: b}

	r := pedir(s, Peticion{Que: QueEstado})
	if !r.OK || r.Estado == nil || !r.Estado.Abierta {
		t.Fatalf("estado: %+v", r)
	}

	r = pedir(s, Peticion{Que: QueCuentas, Testigo: "el-testigo", Origen: "https://www.banco.es/entrar"})
	if !r.OK || len(r.Cuentas) != 1 || r.Cuentas[0].Titulo != "Banco" {
		t.Fatalf("cuentas: %+v", r)
	}
	// **Y sin secretos.** No hay campo donde meterlos, que es la forma buena de
	// garantizarlo, pero se comprueba sobre el JSON de verdad por si alguien añade
	// uno algún día.
	crudo, _ := json.Marshal(r)
	if strings.Contains(string(crudo), "s3cr3t0") {
		t.Errorf("la lista de cuentas lleva la contraseña dentro: %s", crudo)
	}

	// **Copia Esfinge, y por el canal no vuelve el secreto**: solo cuánto tardará
	// en borrarse del portapapeles. Es lo que hace que en esta entrega no salga
	// ni un secreto hacia el navegador.
	r = pedir(s, Peticion{Que: QueCopiarSecreto, Testigo: "el-testigo", ID: "1", Origen: "https://banco.es"})
	if !r.OK || r.Copiado == nil || r.Copiado.Portapapeles != 30 {
		t.Fatalf("copiar la contraseña: %+v", r)
	}
	if b.portapapeles != "s3cr3t0" {
		t.Errorf("no ha copiado la contraseña: %q", b.portapapeles)
	}
	if crudo, _ := json.Marshal(r); strings.Contains(string(crudo), "s3cr3t0") {
		t.Errorf("la contraseña ha vuelto por el canal: %s", crudo)
	}

	r = pedir(s, Peticion{Que: QueCopiarCodigo, Testigo: "el-testigo", ID: "1", Origen: "https://banco.es"})
	if !r.OK || r.Copiado == nil || r.Copiado.Quedan != 20 {
		t.Fatalf("copiar el código: %+v", r)
	}
	if b.portapapeles != "123456" {
		t.Errorf("no ha copiado el código: %q", b.portapapeles)
	}
	if crudo, _ := json.Marshal(r); strings.Contains(string(crudo), "123456") {
		t.Errorf("el código ha vuelto por el canal: %s", crudo)
	}
}

// **La prueba que más vale de este fichero.** Cada uno de estos es una forma de
// sacarle a la bóveda algo que no toca, y todos tienen que fallar por el motivo
// que les corresponde: si uno falla por el motivo equivocado, el día que se
// arregle otra cosa dejará de fallar.
func TestLoQueElNavegadorNoPuedeConseguir(t *testing.T) {
	casos := []struct {
		nombre string
		p      Peticion
		motivo string
	}{
		{
			"sin haber emparejado",
			Peticion{Que: QueCuentas, Origen: "https://banco.es"},
			MotivoSinEmparejar,
		},
		{
			"con un testigo inventado",
			Peticion{Que: QueCuentas, Testigo: "me lo he inventado", Origen: "https://banco.es"},
			MotivoSinEmparejar,
		},
		{
			"pidiendo una contraseña sin decir de qué sitio",
			Peticion{Que: QueCopiarSecreto, Testigo: "el-testigo", ID: "1"},
			MotivoOrigenInvalido,
		},
		{
			"pidiendo la contraseña del banco desde otro sitio",
			Peticion{Que: QueCopiarSecreto, Testigo: "el-testigo", ID: "1", Origen: "https://malo.com"},
			MotivoNoEncaja,
		},
		{
			"pidiendo la contraseña del banco desde un dominio que se le parece",
			Peticion{Que: QueCopiarSecreto, Testigo: "el-testigo", ID: "1", Origen: "https://banco.es.malo.com"},
			MotivoNoEncaja,
		},
		{
			"pidiendo el código de un solo uso desde otro sitio",
			Peticion{Que: QueCopiarCodigo, Testigo: "el-testigo", ID: "1", Origen: "https://malo.com"},
			MotivoNoEncaja,
		},
		{
			"sobre texto claro",
			Peticion{Que: QueCuentas, Testigo: "el-testigo", Origen: "http://banco.es"},
			MotivoOrigenInvalido,
		},
		{
			"pidiendo algo que no existe",
			Peticion{Que: "dame la bóveda entera", Testigo: "el-testigo", Origen: "https://banco.es"},
			MotivoNoEntiendo,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			b := nuevaFalsa()
			s := &Servidor{fuente: b}
			r := pedir(s, c.p)
			if r.OK {
				t.Fatalf("ha funcionado, y no debería: %+v", r)
			}
			if r.Motivo != c.motivo {
				t.Errorf("falla por «%s» y debería fallar por «%s»", r.Motivo, c.motivo)
			}
			if r.Copiado != nil || len(r.Cuentas) > 0 {
				t.Errorf("ha soltado algo por el camino: %+v", r)
			}
		})
	}
}

// Una versión que no se entiende se dice, no se adivina. La extensión se
// actualiza por la tienda y la aplicación empujando una etiqueta: van a estar
// descompasadas casi siempre.
func TestUnaVersionQueNoSeEntiende(t *testing.T) {
	s := &Servidor{fuente: nuevaFalsa()}
	for _, v := range []int{0, 2, 99} {
		r := s.Atender(Peticion{Version: v, Que: QueEstado}, nil)
		if r.OK || r.Motivo != MotivoNoEntiendo {
			t.Errorf("con versión %d: %+v", v, r)
		}
	}
}

// Con la bóveda cerrada **no sale ni un título**, y se dice cuál de los dos casos
// es: no es lo mismo «no hay bóveda» que «está cerrada», y el arreglo tampoco.
func TestConLaBovedaCerradaNoSaleNada(t *testing.T) {
	b := nuevaFalsa()
	b.abierta = false
	s := &Servidor{fuente: b}

	r := pedir(s, Peticion{Que: QueCuentas, Testigo: "el-testigo", Origen: "https://banco.es"})
	if r.OK || r.Motivo != MotivoCerrada {
		t.Errorf("cerrada: %+v", r)
	}
	if len(r.Cuentas) > 0 {
		t.Error("ha enseñado cuentas con la bóveda cerrada")
	}

	b.existe = false
	r = pedir(s, Peticion{Que: QueCuentas, Testigo: "el-testigo", Origen: "https://banco.es"})
	if r.OK || r.Motivo != MotivoSinBoveda {
		t.Errorf("sin bóveda: %+v", r)
	}

	// Y el estado sí se contesta: es lo que la extensión necesita para saber qué
	// enseñar, y no dice nada de dentro.
	if r := pedir(s, Peticion{Que: QueEstado}); !r.OK {
		t.Errorf("el estado tiene que contestarse siempre: %+v", r)
	}
}

// **El límite existe por la enumeración**: preguntar por las cuentas de un
// dominio no devuelve secretos, pero con un diccionario de dominios se
// reconstruye la lista entera de sitios de la bóveda, que es justo lo que se
// cifra en el disco.
func TestNoSePuedeEnumerarLaBovedaAPreguntas(t *testing.T) {
	s := &Servidor{fuente: nuevaFalsa()}
	cuenta := &contador{}

	corta := 0
	for i := 0; i < preguntasPorMinuto*3; i++ {
		r := s.Atender(Peticion{
			Version: VersionDelProtocolo, Que: QueCuentas,
			Testigo: "el-testigo", Origen: "https://banco.es",
		}, cuenta)
		if !r.OK && r.Motivo == MotivoDemasiado {
			corta++
		}
	}
	if corta == 0 {
		t.Fatal("se pueden hacer todas las preguntas que se quiera")
	}
	if corta != preguntasPorMinuto*2 {
		t.Errorf("ha cortado %d veces de %d", corta, preguntasPorMinuto*2)
	}

	// Y pasada la ventana se vuelve a contestar: es un freno, no un castigo.
	cuenta.desde = time.Now().Add(-2 * ventanaDeCuenta)
	if r := s.Atender(Peticion{
		Version: VersionDelProtocolo, Que: QueCuentas,
		Testigo: "el-testigo", Origen: "https://banco.es",
	}, cuenta); !r.OK {
		t.Errorf("sigue cortando pasada la ventana: %+v", r)
	}
}

// El emparejamiento lo contesta una persona, y si dice que no, no hay testigo.
func TestElEmparejamientoLoDecideUnaPersona(t *testing.T) {
	b := nuevaFalsa()
	b.niega = true
	s := &Servidor{fuente: b}

	r := pedir(s, Peticion{Que: QueEmparejar, Quien: "Chrome"})
	if r.OK || r.Motivo != MotivoSinEmparejar || r.Testigo != "" {
		t.Fatalf("con un «no» ha salido: %+v", r)
	}

	b.niega = false
	r = pedir(s, Peticion{Que: QueEmparejar, Quien: "Chrome"})
	if !r.OK || r.Testigo == "" {
		t.Fatalf("con un «sí» no ha salido testigo: %+v", r)
	}
	// Y ese testigo ya vale.
	if r := pedir(s, Peticion{
		Que: QueCuentas, Testigo: r.Testigo, Origen: "https://banco.es",
	}); !r.OK {
		t.Errorf("el testigo recién dado no vale: %+v", r)
	}
}

// La tubería entera, por un socket de verdad: es la prueba que este proyecto ya
// sabe que se le olvida, y la que habría cazado los dos fallos de los iconos.
func TestLaConversacionPorElSocketDeVerdad(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "puente.sock")
	s, err := Servir(ruta, nuevaFalsa())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Parar()

	conn, err := net.Dial("unix", ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	// Dos preguntas por la misma conexión, que es como habla una extensión.
	for _, p := range []Peticion{
		{Version: VersionDelProtocolo, Que: QueEstado},
		{Version: VersionDelProtocolo, Que: QueCuentas, Testigo: "el-testigo", Origen: "https://banco.es"},
	} {
		if err := enc.Encode(p); err != nil {
			t.Fatal(err)
		}
		var r Respuesta
		if err := dec.Decode(&r); err != nil {
			t.Fatal(err)
		}
		if !r.OK {
			t.Fatalf("«%s» ha fallado: %+v", p.Que, r)
		}
	}

	// Y al parar, el fichero del socket se va: dejarlo detrás impide arrancar la
	// próxima vez.
	if err := s.Parar(); err != nil {
		t.Fatal(err)
	}
	if _, err := net.Dial("unix", ruta); err == nil {
		t.Error("sigue contestando después de parar")
	}
}

// Dos Esfinges no pueden escuchar en el mismo sitio, y el segundo tiene que
// decirlo en vez de quitarle el canal al primero.
func TestDosEsfingesNoSePisan(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "puente.sock")
	uno, err := Servir(ruta, nuevaFalsa())
	if err != nil {
		t.Fatal(err)
	}
	defer uno.Parar()

	if _, err := Servir(ruta, nuevaFalsa()); err == nil {
		t.Fatal("el segundo ha escuchado encima del primero")
	} else if !strings.Contains(err.Error(), "otro Esfinge") {
		t.Errorf("no dice qué pasa: %v", err)
	}

	// Y el primero sigue vivo.
	c, err := net.Dial("unix", ruta)
	if err != nil {
		t.Fatalf("le han quitado el canal al primero: %v", err)
	}
	c.Close()
}

// Un socket que quedó de un cierre sucio no puede impedir arrancar para siempre.
func TestUnSocketHuerfanoNoBloqueaParaSiempre(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "puente.sock")
	uno, err := Servir(ruta, nuevaFalsa())
	if err != nil {
		t.Fatal(err)
	}
	// Se cierra el oyente **sin limpiar**, que es lo que pasa cuando el proceso
	// muere de golpe: el fichero se queda ahí.
	uno.oyente.Close()

	dos, err := Servir(ruta, nuevaFalsa())
	if err != nil {
		t.Fatalf("un socket huérfano ha dejado a Esfinge sin canal: %v", err)
	}
	dos.Parar()
}

// La lista de lo que se puede pedir se mantiene a mano, como la del puente con la
// ventana. Esta prueba es lo que obliga a que añadir un verbo sea una decisión.
func TestLoQueSePuedePedirEstaEnLaLista(t *testing.T) {
	s := &Servidor{fuente: nuevaFalsa()}

	var contestados []string
	for _, que := range append([]string{}, LoQueSePuedePedir...) {
		r := s.Atender(Peticion{
			Version: VersionDelProtocolo, Que: que,
			Testigo: "el-testigo", Origen: "https://banco.es", ID: "1", Quien: "Chrome",
		}, nil)
		if r.Motivo == MotivoNoEntiendo {
			t.Errorf("«%s» está en la lista y el servidor no lo conoce", que)
			continue
		}
		contestados = append(contestados, que)
	}

	esperados := append([]string{}, LoQueSePuedePedir...)
	sort.Strings(esperados)
	sort.Strings(contestados)
	if !reflect.DeepEqual(esperados, contestados) {
		t.Errorf("contestados %v, en la lista %v", contestados, esperados)
	}
}
