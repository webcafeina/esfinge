package navegador

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RutaDelCanal es el socket por el que habla la extensión.
//
// **Vive aquí y no en `internal/app`**, y no es colocación: el proceso que lanza
// el navegador solo necesita esta ruta, y si la pidiera al paquete de la
// aplicación se llevaría dentro la bóveda entera —el cifrado, el formato, la
// importación— por una función de seis líneas. Así **el binario que el navegador
// arranca ni siquiera sabe abrir una bóveda**, que es una propiedad que vale la
// pena tener y no solo cuatro megabytes menos.
//
// El nombre va **corto a propósito**: la ruta de un socket de dominio unix no
// puede pasar de 104 caracteres en macOS, y allí la carpeta de configuración ya
// es `~/Library/Application Support`.
func RutaDelCanal() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "puente.sock")
}

// Fuente es lo que el canal puede pedirle a la bóveda.
//
// **Es una interfaz y no la bóveda directamente**, y no por gusto de abstraer:
// este paquete no puede depender de `internal/app` —sería un ciclo— y, sobre
// todo, así la lista de lo que el navegador alcanza está escrita en un sitio y
// se lee de un vistazo. Lo que no esté aquí, el navegador no lo puede pedir.
//
// Fíjate en lo que **no** hay: nada de crear, borrar, exportar, cambiar la
// maestra ni listar la bóveda entera. Y todo lo que devuelve un secreto recibe
// un dominio, para que sea imposible escribir una implementación que devuelva la
// contraseña de un sitio a otro.
type Fuente interface {
	// Estado dice si hay bóveda y si está abierta. No abre nada.
	Estado() Estado
	// CuentasDe son las entradas de ese dominio, **sin secretos**.
	CuentasDe(dominio string) ([]Cuenta, error)
	// CopiarSecreto pone la contraseña de una entrada en el portapapeles del
	// sistema, **si es de ese dominio**. Devuelve en cuántos segundos se borrará.
	//
	// Copia Esfinge y no la extensión: así el secreto no cruza el canal y se
	// aprovecha el borrado que ya existe.
	CopiarSecreto(id, dominio string) (Copiado, error)
	// CopiarCodigo hace lo mismo con el código de un solo uso.
	CopiarCodigo(id, dominio string) (Copiado, error)
	// Rellenar devuelve el usuario y la contraseña de una entrada, **si es de ese
	// dominio**. Es lo único de esta interfaz que entrega un secreto a quien
	// pregunta en vez de dejarlo en el portapapeles.
	Rellenar(id, dominio string) (Relleno, error)
	// Emparejar le pregunta a la persona, en la ventana, si permite que ese
	// navegador hable con la bóveda. Devuelve el testigo si dice que sí.
	Emparejar(quien string) (string, error)
	// Emparejado dice si ese testigo es uno de los que se dieron.
	Emparejado(testigo string) bool
}

// Los topes de preguntas, que son **del canal y no de una conexión**.
//
// Esa distinción es todo el arreglo de la entrega 2, y conviene dejar escrito por
// qué, porque la primera versión parecía correcta y no frenaba nada: el contador
// vivía en `conversar`, es decir **uno por conexión**, y la extensión abre **una
// conexión por pregunta** —lo dice su propio comentario: «un puerto por
// petición», que con MV3 es lo razonable porque el trabajador se muere solo—.
// Cada pregunta llegaba por un proceso nuevo, con su contador a cero. El tope de
// sesenta por minuto no se alcanzaba jamás.
//
// La prueba tampoco lo veía, y por el motivo de siempre: le pasaba **un contador
// hecho a mano** a sesenta llamadas seguidas, que es el caso que no ocurre. Ahora
// los frenos cuelgan del [Servidor] y no hay dónde poner uno por conexión aunque
// se quisiera.
//
// **Existen por la enumeración, que es el ataque que casi se me escapa.** Pedir
// las cuentas de un dominio no devuelve secretos, pero con un diccionario de
// dominios se reconstruye **la lista entera de sitios y usuarios de la bóveda**
// en segundos. Y esa lista es exactamente lo que la ADR 0024 decidió cifrar en el
// disco, con estas palabras: «lo que hay que ocultar no es el dibujo, es la lista
// de sitios». Cifrarla en el disco y regalarla por el socket sería no haber
// entendido la propia decisión.
//
// El tope es holgado para el uso real —una pestaña pregunta una vez— y estrecho
// para un diccionario.
// Y son **dos**, porque las dos cosas no cuestan lo mismo: preguntar de más
// enseña una lista de sitios; rellenar de más entrega contraseñas. El de
// rellenar es estrecho porque el uso real lo es —una persona rellena un
// formulario, no doce— y porque al otro lado hay código nuestro en todas las
// páginas: si alguna vez se cuela algo por ahí, este número es lo que decide
// entre una contraseña y la bóveda entera.
const (
	preguntasPorMinuto = 60
	rellenosPorMinuto  = 12
	ventanaDeCuenta    = time.Minute
)

// Servidor atiende a la extensión del navegador.
type Servidor struct {
	fuente Fuente
	ruta   string
	// frenos son los topes de preguntas, compartidos por todas las conexiones. En
	// las pruebas puede ser nulo, que significa «sin freno».
	frenos *frenos

	mu      sync.Mutex
	oyente  net.Listener
	parando bool
	// conexiones son las conversaciones vivas.
	//
	// **Hay que llevarlas apuntadas para poder cerrarlas**, y esto costó un cuelgue
	// nada más escribirlo: cerrar el oyente no cierra lo ya aceptado, así que
	// `Parar()` se quedaba esperando para siempre a una extensión que estaba
	// tranquilamente escuchando su socket. Apagar el canal en Ajustes se habría
	// llevado la ventana por delante.
	conexiones map[net.Conn]bool
	abiertas   sync.WaitGroup
}

// Servir abre el canal y se queda escuchando. Devuelve en cuanto está listo.
//
// **La ruta se comprueba antes de escuchar**, no después: la de un socket de
// dominio unix tiene un tope de 104 caracteres y en macOS la carpeta de
// configuración ya es `~/Library/Application Support`, así que aquí es donde se
// descubre, y no con un error del sistema operativo que no dice nada.
func Servir(ruta string, f Fuente) (*Servidor, error) {
	if f == nil {
		return nil, errors.New("No hay bóveda a la que preguntar")
	}
	oyente, err := escuchar(ruta)
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

// Donde dice por dónde está escuchando, para poder enseñarlo en Ajustes: en un
// programa que guarda contraseñas, una puerta abierta se dice dónde está.
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
	// Y las conversaciones abiertas, que el oyente no se lleva por delante.
	for _, c := range vivas {
		c.Close()
	}
	s.abiertas.Wait()
	limpiar(s.ruta)
	return err
}

func (s *Servidor) aceptar() {
	for {
		conn, err := s.oyente.Accept()
		if err != nil {
			s.mu.Lock()
			parando := s.parando
			s.mu.Unlock()
			if parando {
				return
			}
			// Un fallo pasajero al aceptar no puede tirar el canal entero; uno
			// permanente sí, y se distingue porque el oyente ya está cerrado.
			if errors.Is(err, net.ErrClosed) {
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

// conversar atiende una conexión hasta que se cierra.
//
// Una conexión es **una pregunta**, o casi: la extensión abre un puerto nativo
// por petición porque el trabajador de MV3 se muere solo cada pocos minutos y un
// puerto de larga vida se cae igual. De ahí la regla que costó el freno de
// mentira de la entrega 1: **aquí no se guarda nada**. Lo que tenga que contar
// algo cuelga del [Servidor], que sí dura.
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

// contador limita cuántas veces se hace algo por ventana de tiempo.
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

// frenos son los dos topes del canal, juntos porque se leen juntos.
type frenos struct {
	preguntas contador
	rellenos  contador
}

func nuevosFrenos() *frenos {
	return &frenos{
		preguntas: contador{tope: preguntasPorMinuto},
		rellenos:  contador{tope: rellenosPorMinuto},
	}
}

// Atender resuelve una petición. **Es una función de la petición y la fuente**,
// sin sockets por medio, para que todo lo que decide se pueda probar sin montar
// nada.
func (s *Servidor) Atender(p Peticion) Respuesta {
	if p.Version != VersionDelProtocolo {
		return mal(MotivoNoEntiendo,
			"Esta versión de Esfinge no entiende a esa extensión; actualiza la que se haya quedado atrás")
	}
	ahora := time.Now()
	if s.frenos != nil {
		if !s.frenos.preguntas.cabe(ahora) {
			return mal(MotivoDemasiado, "Demasiadas preguntas seguidas")
		}
		// **El de rellenar se mira aquí y no en su `case`**, para que caiga antes de
		// tocar la bóveda y para que esté al lado del otro: un freno escondido en la
		// rama de un `switch` es un freno que alguien quita sin verlo.
		if p.Que == QueRellenar && !s.frenos.rellenos.cabe(ahora) {
			return mal(MotivoDemasiado, "Demasiados rellenos seguidos")
		}
	}

	// El estado y el emparejamiento son lo único que se contesta sin testigo, y
	// los dos por el mismo motivo: son los que sirven para conseguirlo.
	switch p.Que {
	case QueEstado:
		e := s.fuente.Estado()
		return Respuesta{OK: true, Estado: &e}
	case QueEmparejar:
		testigo, err := s.fuente.Emparejar(p.Quien)
		if err != nil {
			return mal(MotivoSinEmparejar, err.Error())
		}
		return Respuesta{OK: true, Testigo: testigo}
	}

	if !s.fuente.Emparejado(p.Testigo) {
		return mal(MotivoSinEmparejar,
			"Este navegador todavía no tiene permiso para hablar con la bóveda")
	}

	// A partir de aquí todo toca la bóveda, así que todo necesita un origen y la
	// bóveda abierta. En ese orden: **el origen se comprueba antes de mirar si hay
	// bóveda**, para que una dirección que no vale conteste siempre lo mismo, esté
	// la bóveda abierta o cerrada.
	dominio, err := DominioDeOrigen(p.Origen)
	if err != nil {
		return mal(MotivoOrigenInvalido, err.Error())
	}
	if e := s.fuente.Estado(); !e.Abierta {
		if !e.Existe {
			return mal(MotivoSinBoveda, "Aquí no hay ninguna bóveda todavía")
		}
		// **Y no se trae la ventana al frente.** Cualquier proceso de esta máquina
		// podría entonces hacer aparecer el diálogo de la contraseña maestra cuando
		// quisiera, que es enseñarle a alguien a teclear su maestra cada vez que una
		// ventana se lo pide. Se contesta, y que el navegador ofrezca el botón.
		return mal(MotivoCerrada, "La bóveda está cerrada")
	}

	switch p.Que {
	case QueCuentas:
		cuentas, err := s.fuente.CuentasDe(dominio)
		if err != nil {
			return mal(MotivoCerrada, err.Error())
		}
		return Respuesta{OK: true, Cuentas: cuentas}

	case QueCopiarSecreto:
		c, err := s.fuente.CopiarSecreto(p.ID, dominio)
		if err != nil {
			return mal(MotivoNoEncaja, err.Error())
		}
		return Respuesta{OK: true, Copiado: &c}

	case QueCopiarCodigo:
		c, err := s.fuente.CopiarCodigo(p.ID, dominio)
		if err != nil {
			return mal(MotivoNoEncaja, err.Error())
		}
		return Respuesta{OK: true, Copiado: &c}

	case QueRellenar:
		r, err := s.fuente.Rellenar(p.ID, dominio)
		if err != nil {
			return mal(MotivoNoEncaja, err.Error())
		}
		return Respuesta{OK: true, Relleno: &r}
	}

	return mal(MotivoNoEntiendo, "Esfinge no sabe hacer eso")
}
