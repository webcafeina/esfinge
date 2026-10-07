package agente

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// bovedaFalsa es una fuente entera, con lo justo para que las pruebas puedan mirar qué
// se ha pedido y qué no.
type bovedaFalsa struct {
	existe  bool
	abierta bool
	testigo string
	pedidos int
	// aprobado simula que la persona ya dijo que sí en la ventana.
	aprobado        bool
	pidioAprobacion bool
	// copiado es lo que se puso en el portapapeles: **la prueba mira esto** para
	// distinguir «ha copiado» de «ha devuelto el secreto», que es toda la diferencia.
	copiado string
}

// elSecreto es un centinela: **no tiene que salir por el canal por ningún sitio**, y
// hay una prueba que lo busca en los bytes de todas las respuestas.
const elSecreto = "CONTRASENA-QUE-NO-DEBE-SALIR"

func (b *bovedaFalsa) Estado() Estado {
	return Estado{Existe: b.existe, Abierta: b.abierta, Boveda: "Zeri's Coffee"}
}

func (b *bovedaFalsa) Buscar(texto string) ([]Entrada, int, error) {
	b.pedidos++
	// Dos de las que vuelven, y **más de las que vuelven en total**: es lo que deja
	// comprobar que el total viaja aparte del tope.
	return []Entrada{
		{ID: "a1", Tipo: "credencial", Titulo: "GitHub", Usuario: "zeri", TieneSecreto: true, TieneCodigo: true},
		{ID: "b2", Tipo: "nota", Titulo: "Wifi de casa", TieneSecreto: true},
	}, 7, nil
}

func (b *bovedaFalsa) Ver(id string) (Entrada, error) {
	b.pedidos++
	if id != "a1" {
		return Entrada{}, ErrNoEsta
	}
	return Entrada{ID: "a1", Tipo: "credencial", Titulo: "GitHub", Usuario: "zeri", TieneSecreto: true}, nil
}

func (b *bovedaFalsa) Higiene() (Higiene, error) {
	b.pedidos++
	return Higiene{Repetidas: [][]string{{"a1", "b2"}}, SinCodigo: []string{"b2"}}, nil
}

func (b *bovedaFalsa) Generar(bytes int, alfabeto string) (string, error) {
	return "una-contrasena-nueva", nil
}

// CopiarSecreto: la primera vez pide aprobación, y después de decir que sí copia. Es
// lo que hace la de verdad, y lo que deja probar los dos caminos.
func (b *bovedaFalsa) CopiarSecreto(quien, id string) (Copiado, error) {
	if id != "a1" {
		return Copiado{}, ErrNoEsta
	}
	if !b.aprobado {
		b.pidioAprobacion = true
		return Copiado{}, ErrPideAprobacion
	}
	b.copiado = elSecreto
	return Copiado{Portapapeles: 30, Titulo: "GitHub"}, nil
}

func (b *bovedaFalsa) Emparejar(quien string) (string, error) {
	b.testigo = "testigo-de-prueba"
	return b.testigo, nil
}

func (b *bovedaFalsa) Emparejado(t string) bool { return t != "" && t == b.testigo }

func servidorDePrueba(t *testing.T) (*Servidor, *bovedaFalsa) {
	t.Helper()
	b := &bovedaFalsa{existe: true, abierta: true}
	// Sin frenos: los suyos tienen su propia prueba, y aquí estorbarían.
	return &Servidor{fuente: b}, b
}

// pedir rellena la versión **solo si no la trae**, para que el caso de «otra versión»
// se pueda escribir. Poniéndola siempre, ese caso pasaba en verde sin ejercitar nada.
func pedir(s *Servidor, p Peticion) Respuesta {
	if p.Version == 0 {
		p.Version = VersionDelProtocolo
	}
	return s.Atender(p)
}

// **La prueba que más vale de este fichero**, calcada de la del canal del navegador y
// por la misma razón: cada caso **falla por su motivo concreto**. Si uno fallara por el
// motivo equivocado, el día que se arregle otra cosa dejaría de fallar y nadie se
// enteraría.
//
// Y lleva dentro los verbos de administrar **uno a uno**: no están, no van a estar, y
// tenerlos aquí hace que añadir cualquiera de ellos sea una prueba en rojo y no un
// `case` que aparece un martes.
func TestLoQueElAgenteNoPuedeConseguir(t *testing.T) {
	casos := []struct {
		nombre string
		armar  func(s *Servidor, b *bovedaFalsa)
		p      Peticion
		motivo string
	}{
		{"sin emparejar", nil, Peticion{Que: QueBuscar}, MotivoSinEmparejar},
		{"con un testigo inventado", nil, Peticion{Que: QueBuscar, Testigo: "lo-que-sea"}, MotivoSinEmparejar},
		{"con la bóveda cerrada", func(s *Servidor, b *bovedaFalsa) {
			b.abierta = false
		}, Peticion{Que: QueBuscar, Testigo: "testigo-de-prueba"}, MotivoCerrada},
		{"sin bóveda", func(s *Servidor, b *bovedaFalsa) {
			b.abierta, b.existe = false, false
		}, Peticion{Que: QueBuscar, Testigo: "testigo-de-prueba"}, MotivoSinBoveda},
		{"una entrada que no está", nil, Peticion{Que: QueVer, ID: "no-existe", Testigo: "testigo-de-prueba"}, MotivoNoEsta},
		{"otra versión del protocolo", nil, Peticion{Que: QueEstado, Version: 99}, MotivoNoEntiendo},

		// Administrar. **Uno por línea y a propósito.**
		{"exportar", nil, Peticion{Que: "exportar", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"importar", nil, Peticion{Que: "importar", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"cambiar la maestra", nil, Peticion{Que: "cambiar-maestra", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"borrar la bóveda", nil, Peticion{Que: "borrar-boveda", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"abrir la bóveda", nil, Peticion{Que: "abrir", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"crear un proyecto", nil, Peticion{Que: "crear-proyecto", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"cambiar de bóveda", nil, Peticion{Que: "abrir-proyecto", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"compartir", nil, Peticion{Que: "dar-acceso", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"la clave de recuperación", nil, Peticion{Que: "recuperacion", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
		{"firmar una llave de acceso", nil, Peticion{Que: "firmar-llave", Testigo: "testigo-de-prueba"}, MotivoNoEntiendo},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, b := servidorDePrueba(t)
			b.testigo = "testigo-de-prueba"
			if c.armar != nil {
				c.armar(s, b)
			}
			r := pedir(s, c.p)
			if r.OK {
				t.Fatalf("ha contestado que sí: %+v", r)
			}
			if r.Motivo != c.motivo {
				t.Errorf("motivo %q, se esperaba %q (error: %s)", r.Motivo, c.motivo, r.Error)
			}
		})
	}
}

// **Ninguna respuesta lleva un secreto**, y se comprueba sobre los bytes.
//
// Es la prueba que escala: se recorre toda la lista de verbos, se serializa lo que
// contesta cada uno y se busca el centinela dentro. Sobre los tipos no valdría —un
// campo nuevo con un secreto pasaría— y sobre los bytes sí, que es lo que hace que
// añadir una herramienta sin pensarlo salga en rojo.
func TestNingunaRespuestaLlevaUnSecreto(t *testing.T) {
	s, b := servidorDePrueba(t)
	b.testigo = "testigo-de-prueba"
	for _, que := range LoQueSePuedePedir {
		r := pedir(s, Peticion{Que: que, Testigo: b.testigo, ID: "a1", Texto: "git"})
		crudo, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(crudo), elSecreto) {
			t.Errorf("la respuesta de %q lleva el secreto dentro: %s", que, crudo)
		}
	}
}

// Y que no se pueda pedir algo que no está en la lista **ni al revés**: todo lo que
// está en la lista tiene que contestar. Es lo que obliga a que añadir un verbo sea una
// decisión y no un `case` que alguien escribe y nadie declara.
func TestLoQueSePuedePedirEstaEnLaLista(t *testing.T) {
	s, b := servidorDePrueba(t)
	b.testigo = "testigo-de-prueba"
	for _, que := range LoQueSePuedePedir {
		r := pedir(s, Peticion{Que: que, Testigo: b.testigo, ID: "a1"})
		if !r.OK && r.Motivo == MotivoNoEntiendo {
			t.Errorf("%q está en la lista y el servidor no lo entiende", que)
		}
	}
}

// El freno es **del servidor y no de la conexión**, que es la lección que costó el
// freno de mentira del canal del navegador: allí el contador vivía en la conversación,
// la extensión abría una conexión por pregunta, y el tope no se alcanzaba nunca.
func TestNoSePuedeEnumerarLaBovedaAPreguntas(t *testing.T) {
	b := &bovedaFalsa{existe: true, abierta: true, testigo: "testigo-de-prueba"}
	s := &Servidor{fuente: b, frenos: nuevosFrenos()}

	cortadas := 0
	for i := 0; i < busquedasPorMinuto+10; i++ {
		r := pedir(s, Peticion{Que: QueBuscar, Testigo: b.testigo})
		if !r.OK && r.Motivo == MotivoDemasiado {
			cortadas++
		}
	}
	if cortadas != 10 {
		t.Fatalf("se han cortado %d búsquedas y tenían que ser 10", cortadas)
	}
	// Y es un freno, no un castigo: pasada la ventana se vuelve a poder.
	s.frenos.busquedas.desde = time.Now().Add(-2 * ventanaDeCuenta)
	s.frenos.preguntas.desde = time.Now().Add(-2 * ventanaDeCuenta)
	if r := pedir(s, Peticion{Que: QueBuscar, Testigo: b.testigo}); !r.OK {
		t.Fatalf("pasada la ventana sigue cortando: %+v", r)
	}
}

// Generar no toca la bóveda, así que se contesta con la bóveda cerrada: pedir una
// contraseña nueva ahí es razonable y no cuenta nada de dentro.
func TestGenerarFuncionaConLaBovedaCerrada(t *testing.T) {
	b := &bovedaFalsa{existe: true, abierta: false, testigo: "testigo-de-prueba"}
	s := &Servidor{fuente: b}
	r := pedir(s, Peticion{Que: QueGenerar, Testigo: b.testigo, Bytes: 24, Alfabeto: "hex"})
	if !r.OK || r.Clave == "" {
		t.Fatalf("con la bóveda cerrada no ha generado nada: %+v", r)
	}
}

// Y el estado se contesta sin testigo, porque es lo que sirve para saber si hace falta
// emparejarse. Lo que no dice es nada de dentro.
func TestElEstadoSeContestaSinEmparejar(t *testing.T) {
	s, _ := servidorDePrueba(t)
	r := pedir(s, Peticion{Que: QueEstado})
	if !r.OK || r.Estado == nil {
		t.Fatalf("no ha contestado el estado: %+v", r)
	}
}

// **Copiar pide un sí la primera vez, y después copia.**
//
// Lo que esta prueba vigila de verdad es la diferencia entre las dos cosas que podrían
// llamarse igual: que **se copie** y que **se devuelva**. Por eso mira el portapapeles
// del doble, no solo que la respuesta diga que fue bien.
func TestCopiarPideUnSiYLuegoCopia(t *testing.T) {
	s, b := servidorDePrueba(t)
	b.testigo = "testigo-de-prueba"

	// La primera vez: su propio motivo, **y no un error cualquiera**, para que quien
	// llame pueda distinguir «todavía no» de «no».
	r := pedir(s, Peticion{Que: QueCopiarSecreto, ID: "a1", Testigo: b.testigo})
	if r.OK || r.Motivo != MotivoPideAprobacion {
		t.Fatalf("la primera vez contesta %+v", r)
	}
	if !b.pidioAprobacion {
		t.Error("no ha dejado pedido el permiso, así que nadie se entera en la ventana")
	}
	// Y **no ha copiado nada** mientras tanto.
	if b.copiado != "" {
		t.Fatal("ha copiado antes de que nadie dijera que sí")
	}

	// La persona dice que sí, y el agente lo vuelve a pedir.
	b.aprobado = true
	r = pedir(s, Peticion{Que: QueCopiarSecreto, ID: "a1", Testigo: b.testigo})
	if !r.OK || r.Copiado == nil {
		t.Fatalf("tras aprobarlo contesta %+v", r)
	}
	if b.copiado != elSecreto {
		t.Error("ha dicho que sí pero no ha copiado nada")
	}
	// **Y lo que vuelve no lleva la contraseña**, sobre los bytes.
	crudo, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(crudo), elSecreto) {
		t.Errorf("la respuesta lleva la contraseña: %s", crudo)
	}
	if r.Copiado.Portapapeles == 0 {
		t.Error("no dice en cuánto se borra del portapapeles")
	}
}

// Y una entrada que no está no se copia, aunque esté aprobado todo.
func TestNoSeCopiaLoQueNoEsta(t *testing.T) {
	s, b := servidorDePrueba(t)
	b.testigo, b.aprobado = "testigo-de-prueba", true
	r := pedir(s, Peticion{Que: QueCopiarSecreto, ID: "no-existe", Testigo: b.testigo})
	if r.OK || r.Motivo != MotivoNoEsta {
		t.Fatalf("contesta %+v", r)
	}
}
