package agente

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/webcafeina/esfinge/internal/canal"
)

// RutaDelCanal es el socket por el que hablan los agentes.
//
// **Es otro socket, no el del navegador**, y ésa es media ADR 0054: allí todo lo que
// toca la bóveda exige un origen `https` que un agente no tiene, el testigo no tiene
// ámbito —quien lo consigue puede los catorce verbos, incluido firmar llaves de
// acceso— y los frenos son del canal, así que un agente se los comería a la
// extensión. Dos puertas con dos llaves cuestan más código y es lo que hay.
func RutaDelCanal() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "agentes.sock")
}

// Fuente es lo que este canal sabe pedirle a la bóveda, **y nada más**.
//
// Es deliberadamente corta, como la del navegador: lo que no está aquí no se puede
// pedir por mucho que exista en `*App`. En la A1 no hay un solo método que entregue
// un secreto; los que lo hagan llegarán con la A3 y **cada uno tendrá que
// justificarse solo**.
type Fuente interface {
	// Estado dice si hay bóveda, si está abierta y en cuál se trabaja. No abre nada.
	Estado() Estado
	// Buscar son las entradas que coinciden, **sin secretos**, con las dos banderas.
	Buscar(texto string) ([]Entrada, error)
	// Ver es una entrada **sin secretos**. Devuelve [ErrNoEsta] si no existe.
	Ver(id string) (Entrada, error)
	// Higiene es lo que está mal, por identificador y sin secretos.
	Higiene() (Higiene, error)
	// Generar devuelve una contraseña nueva. **No toca la bóveda.**
	Generar(bytes int, alfabeto string) (string, error)

	// Emparejar le pregunta a la persona, en la ventana, si permite que ese agente
	// hable con la bóveda. Devuelve el testigo si dice que sí.
	Emparejar(quien string) (string, error)
	// Emparejado dice si ese testigo es uno de los que se dieron.
	Emparejado(testigo string) bool
}

// ErrNoEsta es que esa entrada no existe, o está en la papelera.
var ErrNoEsta = errors.New("Esa entrada no está en la bóveda")

// Los topes, **del canal y no de una conexión**.
//
// Están aquí por la lección que costó el freno de mentira de la entrega 1 del
// navegador: el contador vivía en la conversación y la extensión abre una conexión
// por pregunta, así que cada una llegaba con el contador a cero y el tope no se
// alcanzaba nunca. **Antes de escribir un contador, preguntarse cuánto vive la cosa
// donde se guarda.**
//
// Y son más estrechos que los del navegador a propósito: allí el que pregunta es una
// página que carga; aquí es un programa en un bucle, y lo que hay que acotar no es
// la molestia sino **cuánto se puede enumerar en un minuto**.
const (
	preguntasPorMinuto = 60
	busquedasPorMinuto = 20
	ventanaDeCuenta    = time.Minute
)

// Servidor atiende a los agentes.
type Servidor struct {
	fuente Fuente
	ruta   string
	// frenos son los topes, compartidos por todas las conexiones. En las pruebas
	// puede ser nulo, que significa «sin freno».
	frenos *frenos

	mu      sync.Mutex
	oyente  net.Listener
	parando bool
	// conexiones son las conversaciones vivas, apuntadas para poder cerrarlas:
	// cerrar el oyente no cierra lo ya aceptado, y eso colgó `Parar()` en el canal
	// del navegador nada más escribirlo.
	conexiones map[net.Conn]bool
	abiertas   sync.WaitGroup
}

// Servir abre el canal y se queda escuchando. Devuelve en cuanto está listo.
func Servir(ruta string, f Fuente) (*Servidor, error) {
	if f == nil {
		return nil, errors.New("No hay bóveda a la que preguntar")
	}
	oyente, err := canal.Escuchar(ruta, "los agentes")
	if err != nil {
		return nil, err
	}
	s := &Servidor{
		fuente:     f,
		ruta:       ruta,
		frenos:     nuevosFrenos(),
		oyente:     oyente,
		conexiones: map[net.Conn]bool{},
	}
	go s.aceptar()
	return s, nil
}

// Donde dice por dónde está escuchando, para enseñarlo en Ajustes: en un programa
// que guarda contraseñas, una puerta abierta se dice dónde está.
func (s *Servidor) Donde() string { return s.ruta }

// Parar cierra el canal y espera a que las conversaciones en curso terminen.
func (s *Servidor) Parar() error {
	s.mu.Lock()
	if s.parando {
		s.mu.Unlock()
		return nil
	}
	s.parando = true
	oyente := s.oyente
	vivas := make([]net.Conn, 0, len(s.conexiones))
	for c := range s.conexiones {
		vivas = append(vivas, c)
	}
	s.mu.Unlock()

	err := oyente.Close()
	for _, c := range vivas {
		c.Close()
	}
	s.abiertas.Wait()
	canal.Limpiar(s.ruta)
	return err
}

func (s *Servidor) aceptar() {
	for {
		conn, err := s.oyente.Accept()
		if err != nil {
			s.mu.Lock()
			parando := s.parando
			s.mu.Unlock()
			if parando || errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		s.mu.Lock()
		if s.parando {
			s.mu.Unlock()
			conn.Close()
			return
		}
		s.conexiones[conn] = true
		s.mu.Unlock()

		s.abiertas.Add(1)
		go func() {
			defer s.abiertas.Done()
			s.conversar(conn)
		}()
	}
}

// conversar atiende una conexión hasta que se cierra. **Aquí no se guarda nada**:
// lo que tenga que contar algo cuelga del [Servidor], que sí dura.
func (s *Servidor) conversar(conn net.Conn) {
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.conexiones, conn)
		s.mu.Unlock()
	}()

	dec := json.NewDecoder(io.LimitReader(conn, 1<<20))
	enc := json.NewEncoder(conn)

	for {
		var p Peticion
		if err := dec.Decode(&p); err != nil {
			return // se ha ido, o ha mandado algo que no es JSON
		}
		if err := enc.Encode(s.Atender(p)); err != nil {
			return
		}
	}
}

type contador struct {
	mu     sync.Mutex
	tope   int
	desde  time.Time
	cuanto int
}

func (c *contador) cabe(ahora time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ahora.Sub(c.desde) >= ventanaDeCuenta {
		c.desde, c.cuanto = ahora, 0
	}
	c.cuanto++
	return c.cuanto <= c.tope
}

type frenos struct {
	preguntas contador
	busquedas contador
}

func nuevosFrenos() *frenos {
	return &frenos{
		preguntas: contador{tope: preguntasPorMinuto},
		busquedas: contador{tope: busquedasPorMinuto},
	}
}

// Atender resuelve una petición. **Es una función de la petición y la fuente**, sin
// sockets por medio, para que todo lo que decide se pueda probar sin montar nada.
//
// El orden es el del canal del navegador y no es casual: **versión, frenos, testigo
// y estado antes de tocar la bóveda**. Los frenos van arriba y no dentro del `switch`
// porque un freno escondido en la rama de un `switch` es un freno que alguien quita
// sin verlo.
func (s *Servidor) Atender(p Peticion) Respuesta {
	if p.Version != VersionDelProtocolo {
		return mal(MotivoNoEntiendo, "Esta versión de Esfinge no entiende esa versión del protocolo")
	}
	ahora := time.Now()
	if s.frenos != nil {
		if !s.frenos.preguntas.cabe(ahora) {
			return mal(MotivoDemasiado, "Demasiadas peticiones seguidas: espera un minuto")
		}
		if esBusqueda(p.Que) && !s.frenos.busquedas.cabe(ahora) {
			return mal(MotivoDemasiado, "Demasiadas búsquedas seguidas: espera un minuto")
		}
	}

	// Estos dos van antes del testigo, porque son los que sirven para conseguirlo.
	switch p.Que {
	case QueEstado:
		e := s.fuente.Estado()
		return Respuesta{OK: true, Estado: &e}
	case QueEmparejar:
		t, err := s.fuente.Emparejar(p.Quien)
		if err != nil {
			return mal(MotivoSinEmparejar, err.Error())
		}
		return Respuesta{OK: true, Testigo: t}
	}

	if !s.fuente.Emparejado(p.Testigo) {
		return mal(MotivoSinEmparejar, "Permite este agente en la ventana de Esfinge")
	}

	// **Generar no toca la bóveda**, así que se contesta esté como esté: pedir una
	// contraseña nueva con la bóveda cerrada es razonable y no cuenta nada de dentro.
	if p.Que == QueGenerar {
		clave, err := s.fuente.Generar(p.Bytes, p.Alfabeto)
		if err != nil {
			return mal(MotivoNoEntiendo, err.Error())
		}
		return Respuesta{OK: true, Clave: clave}
	}

	// Y a partir de aquí hace falta la bóveda. Los dos casos se distinguen porque el
	// arreglo es distinto: una se abre, la otra se crea.
	//
	// **Y no se trae la ventana al frente**, como en el canal del navegador: hacerlo
	// dejaría que cualquier programa de la máquina provocara el diálogo de la
	// contraseña maestra a voluntad.
	est := s.fuente.Estado()
	if !est.Abierta {
		if !est.Existe {
			return mal(MotivoSinBoveda, "Todavía no hay ninguna bóveda en este equipo")
		}
		return mal(MotivoCerrada, "La bóveda está cerrada: ábrela en Esfinge")
	}

	switch p.Que {
	case QueBuscar:
		es, err := s.fuente.Buscar(p.Texto)
		if err != nil {
			return mal(MotivoNoEntiendo, err.Error())
		}
		return Respuesta{OK: true, Entradas: es}
	case QueVer:
		e, err := s.fuente.Ver(p.ID)
		if errors.Is(err, ErrNoEsta) {
			return mal(MotivoNoEsta, err.Error())
		}
		if err != nil {
			return mal(MotivoNoEntiendo, err.Error())
		}
		return Respuesta{OK: true, Entrada: &e}
	case QueHigiene:
		h, err := s.fuente.Higiene()
		if err != nil {
			return mal(MotivoNoEntiendo, err.Error())
		}
		return Respuesta{OK: true, Higiene: &h}
	}
	return mal(MotivoNoEntiendo, "Eso no se puede pedir por aquí")
}

// esBusqueda son los verbos que recorren la bóveda entera, y por eso tienen su
// propio tope: es lo que acota **cuánto se puede enumerar en un minuto**, que es lo
// que la ADR 0024 decidió proteger al cifrar la lista de sitios en el disco.
func esBusqueda(que string) bool {
	return que == QueBuscar || que == QueHigiene
}
