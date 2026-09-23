// Package sincro mantiene la bóveda de este equipo al día con la de la cuenta
// (ADR 0038, plan en docs/cuentas.md).
//
// Una pasada es siempre lo mismo: **bajar, fundir y, si queda algo que el
// servidor no tiene, subir sobre la versión que se bajó**. Si otro equipo sube en
// medio, el servidor dice que la versión ya no es la última y se vuelve a
// empezar. Lo que se recuerda entre pasadas —la última versión vista y sus bytes,
// la base de la fusión— vive junto al fichero de la bóveda.
//
// Dos reglas que vienen de lo que ya costó en Esfinge:
//
//   - **Nada de esto cuenta como actividad.** Sincronizar de fondo no puede
//     mantener abierta una bóveda que nadie está usando (CLAUDE.md, el goteo de
//     iconos y el código de un solo uso). Este paquete no conoce el reloj del
//     bloqueo, y así no puede tocarlo.
//   - **El turno se reserva, no se pregunta.** Dos pasadas a la vez subirían dos
//     veces sobre la misma versión (la lección de `ReservarComprobacion`).
package sincro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cuenta"
	"github.com/webcafeina/esfinge/internal/escritura"
)

// Servidor es lo que hace falta del servidor de cuentas. Lo cumple *cuenta.Cliente.
type Servidor interface {
	Bajar(ctx context.Context, token string, siNoCoincide int64) ([]byte, int64, bool, error)
	Subir(ctx context.Context, token string, siCoincide int64, datos []byte) (int64, error)
}

// Boveda es lo que hace falta de la bóveda. Lo cumple *boveda.Boveda.
type Boveda interface {
	Fundir(remoto []byte, version int64, base []byte, o boveda.OpcionesDeFusion) (boveda.Fusion, error)
	PrepararSubida(version int64) ([]byte, int64, error)
	Serie() int64
}

var (
	// ErrRetroceso: el servidor ha vuelto atrás —una versión más vieja que la ya
	// vista, o ninguna bóveda después de haberla tenido—. No se funde ni se sube
	// encima: se para y se dice.
	ErrRetroceso = errors.New("El servidor tiene una bóveda más vieja que la que ya se vio aquí; no se sincroniza hasta mirarlo")
	// ErrOcupado: ya hay una pasada en marcha. La que está en marcha hará otra al
	// terminar, así que no se pierde nada.
	ErrOcupado = errors.New("Ya se está sincronizando")
)

// Recuerdo es lo que un equipo sabe del servidor desde la última pasada.
type Recuerdo struct {
	// Version es la última versión del servidor que se ha visto aquí.
	Version int64 `json:"version"`
	// Serie es la del fichero de la bóveda cuando se vio: si ha cambiado, hay
	// cambios de aquí sin subir.
	Serie int64 `json:"serie"`
}

// Memoria guarda el Recuerdo y la base de la fusión entre pasadas.
type Memoria interface {
	Cargar() (Recuerdo, []byte, error)
	Guardar(r Recuerdo, base []byte) error
	Olvidar() error
}

// Resultado cuenta lo que ha hecho una pasada.
type Resultado struct {
	Version  int64
	Bajo     bool // ha llegado algo de otro equipo
	Subio    bool // ha subido lo de aquí
	Fusion   boveda.Fusion
	Intentos int
}

// Sincronizador hace las pasadas de un equipo.
type Sincronizador struct {
	Servidor Servidor
	Boveda   Boveda
	Memoria  Memoria
	// Token da la sesión de ahora; puede cambiar entre pasadas.
	Token func() string
	// aunqueBorre vale para **una sola pasada** y se consume al usarla: es lo que
	// pone el botón de «Juntar igual» cuando la fusión se paró porque se llevaba
	// media bóveda (revisión del 2026-09-23). Se guarda aquí y no en un campo
	// corriente porque lo enciende la gorrutina de la ventana mientras otra puede
	// estar sincronizando.
	aunqueBorre atomic.Bool

	turno   atomic.Bool
	otraVez atomic.Bool
}

// intentos que se hacen cuando otro equipo sube en medio de una pasada.
const intentos = 5

// Pendiente dice si hay cambios de aquí sin subir.
// UnaVezAunqueBorre deja dicho que **la próxima pasada** funda aunque se lleve más
// de la mitad de las entradas. Lo pide una persona, mirando lo que va a pasar.
func (s *Sincronizador) UnaVezAunqueBorre() { s.aunqueBorre.Store(true) }

func (s *Sincronizador) Pendiente() bool {
	r, _, err := s.Memoria.Cargar()
	return err != nil || r.Serie != s.Boveda.Serie()
}

// Vaciar sube lo que quede de aquí antes de parar, **con un tope de tiempo**: lo
// usa la aplicación al cerrar la bóveda, que no puede quedarse esperando a la red.
// Si hay una pasada en marcha, espera su turno dentro del mismo tope.
func (s *Sincronizador) Vaciar(tope time.Duration) error {
	if !s.Pendiente() {
		return nil
	}
	ctx, cancelar := context.WithTimeout(context.Background(), tope)
	defer cancelar()
	for {
		_, err := s.Sincronizar(ctx)
		if !errors.Is(err, ErrOcupado) {
			return err
		}
		if !esperar(ctx, 20*time.Millisecond) {
			return ctx.Err()
		}
	}
}

// Sincronizar hace una pasada. Si ya hay una en marcha, devuelve ErrOcupado y la
// que está en marcha repite al terminar.
func (s *Sincronizador) Sincronizar(ctx context.Context) (Resultado, error) {
	if !s.turno.CompareAndSwap(false, true) {
		s.otraVez.Store(true)
		return Resultado{}, ErrOcupado
	}
	defer s.turno.Store(false)
	for {
		s.otraVez.Store(false)
		r, err := s.pasada(ctx)
		if err != nil || !s.otraVez.Load() || ctx.Err() != nil {
			return r, err
		}
	}
}

func (s *Sincronizador) pasada(ctx context.Context) (Resultado, error) {
	var r Resultado
	recuerdo, base, err := s.Memoria.Cargar()
	if err != nil {
		return r, err
	}
	token := s.Token()

	for r.Intentos = 1; r.Intentos <= intentos; r.Intentos++ {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		// Sin base no se pregunta «¿ha cambiado desde la N?»: se baja entera.
		siNoCoincide := recuerdo.Version
		if len(base) == 0 {
			siNoCoincide = 0
		}
		datos, version, cambio, err := s.Servidor.Bajar(ctx, token, siNoCoincide)
		var sobre int64
		switch {
		case errors.Is(err, cuenta.ErrSinBoveda):
			if recuerdo.Version > 0 {
				return r, fmt.Errorf("%w (aquí se vio la %d y allí no hay ninguna)", ErrRetroceso, recuerdo.Version)
			}
			sobre = 0 // la cuenta está vacía: lo de aquí es la primera versión
		case err != nil:
			return r, err
		case !cambio:
			r.Version = recuerdo.Version
			if s.Boveda.Serie() == recuerdo.Serie {
				return r, nil // nada nuevo en ningún lado
			}
			sobre = recuerdo.Version
		default:
			if version < recuerdo.Version {
				return r, fmt.Errorf("%w (aquí se vio la %d y allí dice la %d)", ErrRetroceso, recuerdo.Version, version)
			}
			f, err := s.Boveda.Fundir(datos, version, base, boveda.OpcionesDeFusion{
				AunqueBorreMucho: s.aunqueBorre.Swap(false),
			})
			r.Fusion = f
			if err != nil {
				return r, err
			}
			r.Bajo = f.Cambio
			recuerdo, base = Recuerdo{Version: version, Serie: f.Serie}, datos
			if err := s.Memoria.Guardar(recuerdo, base); err != nil {
				return r, err
			}
			r.Version = version
			if !f.Subir {
				return r, nil
			}
			sobre = version
		}

		subida, serie, err := s.Boveda.PrepararSubida(sobre + 1)
		if err != nil {
			return r, err
		}
		nueva, err := s.Servidor.Subir(ctx, token, sobre, subida)
		if _, conflicto := cuenta.Conflicto(err); conflicto {
			continue // otro equipo ha subido en medio: a bajar otra vez
		}
		if err != nil {
			return r, err
		}
		if nueva != sobre+1 {
			return r, fmt.Errorf("El servidor ha guardado la bóveda como la versión %d, y se esperaba la %d", nueva, sobre+1)
		}
		recuerdo, base = Recuerdo{Version: nueva, Serie: serie}, subida
		if err := s.Memoria.Guardar(recuerdo, base); err != nil {
			return r, err
		}
		r.Version, r.Subio = nueva, true
		return r, nil
	}
	return r, errors.New("Otros equipos están subiendo cambios sin parar; se probará más tarde")
}

// ------------------------------------------------------------------ vigilar

// Plazos de Vigilar.
const (
	// EsperaTrasGuardar agrupa los guardados seguidos en una sola subida.
	EsperaTrasGuardar = 3 * time.Second
	// CadaCuanto se mira si hay algo nuevo aunque aquí no se toque nada.
	//
	// **Un minuto, no cinco**: con cinco, el cliente veía los cambios del otro
	// equipo solo al cerrar y volver a abrir la bóveda (2.23.1). Mirar cuesta poco:
	// si no hay nada nuevo, el servidor contesta 304 sin mandar la bóveda.
	CadaCuanto = time.Minute
	// ReintentoMaximo es lo más que se espera tras un fallo de red.
	ReintentoMaximo = 5 * time.Minute
	// TrasOcupado es lo que se espera cuando la pasada se encontró con otra en
	// marcha. Corto: lo único que hace falta es volver a mirar cuando la otra haya
	// terminado.
	TrasOcupado = time.Second
)

// Vigilante hace pasadas cuando se le pide y cada cierto tiempo, hasta que se
// cancela el contexto.
type Vigilante struct {
	S *Sincronizador
	// Avisar recibe el resultado de cada pasada, con su error. Puede ser nil.
	Avisar func(Resultado, error)
	// Espera tras un guardado antes de subir; cero es EsperaTrasGuardar. Es un
	// campo y no una variable del paquete para que las pruebas puedan acortarla
	// sin pisar a otro vigilante que siga vivo.
	Espera time.Duration

	pedido chan struct{}
	ya     chan struct{}
	una    sync.Once
}

func (v *Vigilante) canal() chan struct{} {
	v.una.Do(func() {
		v.pedido = make(chan struct{}, 1)
		v.ya = make(chan struct{}, 1)
	})
	return v.pedido
}

// Ya pide una pasada **sin la espera de después de guardar**: la de un botón o la
// de volver a la ventana. Pedir espera para juntar los guardados seguidos, y eso
// en un botón se lee como que no ha hecho nada (lo pidió el cliente, 2.25.2).
func (v *Vigilante) Ya() {
	v.canal()
	select {
	case v.ya <- struct{}{}:
	default:
	}
}

// Pedir avisa de que hay algo que sincronizar —un guardado—. No espera ni
// bloquea: si ya había un pedido pendiente, éste se suma a él.
func (v *Vigilante) Pedir() {
	select {
	case v.canal() <- struct{}{}:
	default:
	}
}

// Vigilar corre hasta que se cancela `ctx`.
func (v *Vigilante) Vigilar(ctx context.Context) {
	pedido := v.canal()
	ya := v.ya
	espera := CadaCuanto
	fallos := 0
	temporizador := time.NewTimer(0) // la primera pasada, al arrancar
	defer temporizador.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pedido:
			// Se deja pasar un momento para juntar los guardados seguidos.
			tras := v.Espera
			if tras <= 0 {
				tras = EsperaTrasGuardar
			}
			// Y un «ya» a mitad de la espera la corta: el botón no espera a nadie.
			if !esperarSalvo(ctx, tras, ya) {
				return
			}
			select {
			case <-pedido:
			default:
			}
		case <-ya:
			// Sin esperar. Y un guardado pendiente va en esta misma pasada.
			select {
			case <-pedido:
			default:
			}
		case <-temporizador.C:
		}
		// Cuando las dos ramas están listas, Go elige al azar: se vuelve a mirar
		// si hay que parar antes de trabajar (CLAUDE.md).
		if ctx.Err() != nil {
			return
		}
		r, err := v.S.Sincronizar(ctx)
		if errors.Is(err, ErrOcupado) {
			// Otra pasada estaba en marcha y ésta no tiene nada que contar. **Pero
			// hay que rearmar el reloj igual**: con un `continue` se saltaba el
			// rearme de abajo y el vigilante se quedaba dormido para siempre —sin
			// pasadas cada minuto—, y la ventana, en «Sincronizando…» hasta el
			// siguiente guardado. Pasa al abrir la bóveda mientras la pasada del
			// vigilante anterior todavía corre, que es justo lo que hace entrar con
			// la contraseña nueva. Lo cazó la puerta de publicación de la 2.25.4, no
			// esta máquina.
			//
			// Y se reintenta pronto: la pasada que estaba en marcha puede ser de un
			// vigilante ya cancelado, y ésa no cuenta nada (alSincronizar se calla
			// con `context.Canceled`).
			espera = TrasOcupado
		} else {
			if v.Avisar != nil {
				v.Avisar(r, err)
			}
			if err != nil && !errors.Is(err, context.Canceled) {
				// Espera creciente: 5 s, 10 s, 20 s… hasta el máximo.
				fallos++
				espera = min(5*time.Second<<min(fallos-1, 10), ReintentoMaximo)
			} else {
				fallos = 0
				espera = CadaCuanto
			}
		}
		if !temporizador.Stop() {
			select {
			case <-temporizador.C:
			default:
			}
		}
		temporizador.Reset(espera)
	}
}

// esperarSalvo es esperar, salvo que llegue algo por `corte`, que la acaba antes.
// Devuelve falso si se ha cancelado el contexto.
func esperarSalvo(ctx context.Context, d time.Duration, corte <-chan struct{}) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	case <-corte:
		return true
	}
}

func esperar(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// ------------------------------------------------------------------ en el disco

// JuntoALaBoveda guarda el recuerdo al lado del fichero de la bóveda:
// `<ruta>.base`, los bytes de la última versión vista, y `<ruta>.sincro`, qué
// versión era. Los dos con permisos 600, como la bóveda. **Quien borre la bóveda
// tiene que borrarlos** (Olvidar), la misma lección que la caché de iconos.
type JuntoALaBoveda struct{ Ruta string }

func (j JuntoALaBoveda) Cargar() (Recuerdo, []byte, error) {
	var r Recuerdo
	crudo, err := os.ReadFile(j.Ruta + ".sincro")
	if errors.Is(err, os.ErrNotExist) {
		return Recuerdo{}, nil, nil
	}
	if err != nil {
		return r, nil, err
	}
	if err := json.Unmarshal(crudo, &r); err != nil {
		// Un recuerdo roto no para nada: se olvida y la próxima pasada baja entera.
		return Recuerdo{}, nil, nil
	}
	base, err := os.ReadFile(j.Ruta + ".base")
	if err != nil {
		return Recuerdo{Version: r.Version, Serie: -1}, nil, nil
	}
	return r, base, nil
}

func (j JuntoALaBoveda) Guardar(r Recuerdo, base []byte) error {
	// La base primero: un recuerdo que apunta a una base que no está obliga a
	// bajar entera, y eso es un coste; al revés sería fundir contra otra cosa.
	if err := escribir(j.Ruta+".base", base); err != nil {
		return err
	}
	crudo, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return escribir(j.Ruta+".sincro", crudo)
}

func (j JuntoALaBoveda) Olvidar() error {
	for _, sufijo := range []string{".base", ".sincro"} {
		if err := os.Remove(j.Ruta + sufijo); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func escribir(ruta string, datos []byte) error {
	return escritura.Atomica(ruta, escritura.Opciones{CrearCarpeta: true}, func(w io.Writer) error {
		_, err := w.Write(datos)
		return err
	})
}
