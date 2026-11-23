//go:build dev

// Servidor de desarrollo.
//
// Existe por una razón práctica: la aplicación de verdad necesita un entorno
// gráfico y el webview del sistema, y en la máquina donde se escribe esto no
// hay ninguno de los dos. Sin esta puerta, la interfaz solo podría probarse con
// datos inventados, que es la clase de prueba que pasa siempre y no descubre
// nada.
//
// Con ella, el navegador habla con el mismo código Go que hablará la ventana:
// las mismas funciones, los mismos errores, el mismo formato de contenedor. Lo
// único distinto es el transporte.
//
// Va detrás de una etiqueta de compilación para que no acabe dentro de la
// aplicación que se distribuye. Un servidor HTTP en el binario del cliente, por
// local que sea, es una puerta que nadie ha pedido.
package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/webcafeina/esfinge/internal/actualizacion"
	"github.com/webcafeina/esfinge/internal/llavero"
)

// ficheroDeMentira es lo que sirve `/api/_descarga`: unos bytes con la forma de
// un fichero descargable y nada dentro.
//
// Son 64 KiB para que el contador de avance tenga algo que contar —con cuatro
// bytes, la descarga termina antes del primer aviso y el camino del progreso no
// se ejercita— y lo bastante poco para que no se note.
var ficheroDeMentira = func() []byte {
	b := make([]byte, 64*1024)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}()

// SistemaDeDesarrollo hace de escritorio cuando no hay escritorio.
//
// Los diálogos de fichero no existen en un navegador, así que se resuelven
// listando una carpeta que se indica al arrancar. Es suficiente para ejercitar
// el camino entero —elegir, cifrar, ver el resultado— sin fingir nada del lado
// de Go.
type SistemaDeDesarrollo struct {
	portapapeles string
	Carpeta      string

	mu      sync.Mutex
	oyentes map[chan []byte]bool
}

func NuevoSistemaDeDesarrollo(carpeta string) *SistemaDeDesarrollo {
	return &SistemaDeDesarrollo{Carpeta: carpeta, oyentes: map[chan []byte]bool{}}
}

func (s *SistemaDeDesarrollo) ElegirFicheros(_, _ string, varios bool, _ Filtro) ([]string, error) {
	entradas, err := os.ReadDir(s.Carpeta)
	if err != nil {
		return nil, fmt.Errorf("No puedo leer %s: %w", s.Carpeta, err)
	}

	var rutas []string
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		rutas = append(rutas, filepath.Join(s.Carpeta, e.Name()))
		if !varios {
			break
		}
	}
	return rutas, nil
}

func (s *SistemaDeDesarrollo) ElegirDondeGuardar(_, nombreSugerido, _ string) (string, error) {
	return filepath.Join(s.Carpeta, nombreSugerido), nil
}

// Avisar reparte el evento entre los navegadores conectados.
// Cerrar no cierra nada aquí: el servidor de desarrollo no es una ventana, y
// matarlo dejaría las pruebas sin con qué hablar. Se anota y se sigue.
// El portapapeles del servidor de desarrollo es una variable: aquí no hay
// escritorio al que pedírselo, y lo que se prueba es la lógica del borrado —que
// caduca, y que no pisa lo que se haya copiado después—, no el portapapeles del
// sistema.
func (s *SistemaDeDesarrollo) PonerEnPortapapeles(texto string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.portapapeles = texto
	return nil
}

func (s *SistemaDeDesarrollo) LeerPortapapeles() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.portapapeles, nil
}

func (s *SistemaDeDesarrollo) Cerrar() {
	log.Println("La aplicación pediría cerrarse ahora para actualizarse")
}

func (s *SistemaDeDesarrollo) Avisar(evento string, datos any) {
	cuerpo, err := json.Marshal(datos)
	if err != nil {
		return
	}
	mensaje := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", evento, cuerpo))

	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.oyentes {
		select {
		case c <- mensaje:
		default: // un oyente que no lee no puede frenar el trabajo
		}
	}
}

// Servir levanta el servidor de desarrollo y no vuelve.
func Servir(a *App, s *SistemaDeDesarrollo, direccion string) error {
	mux := http.NewServeMux()

	// Responde enseguida y sirve para saber que el servidor está en pie. El flujo
	// de eventos no vale para eso: no termina nunca, por diseño.
	mux.HandleFunc("/api/salud", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"listo":true}`))
	})

	// **El llavero de mentira, mandado desde la prueba.**
	//
	// Desde que la huella se pide sola al llegar (2.27.1), el único camino por el
	// que se llega a teclear la contraseña maestra con el desbloqueo puesto es
	// **cancelar el diálogo del sistema**, y eso aquí no lo puede hacer nadie: no
	// hay diálogo. Sin esto, la prueba de que «la maestra sigue abriendo» dejaría
	// de comprobar lo que dice.
	//
	// Va aquí y **no como método de `App`** a conciencia: todo lo que se exporta
	// ahí queda expuesto a la ventana por `Bind`, y «haz que el llavero diga que
	// no» es exactamente la clase de puerta que no se abre por comodidad. Esto
	// vive tras la etiqueta `dev`, con el resto del servidor, y no existe en el
	// binario del cliente.
	mux.HandleFunc("/api/_llavero", func(w http.ResponseWriter, r *http.Request) {
		l, vale := a.llaveroDelSistema().(*llavero.DeMentira)
		if !vale {
			http.Error(w, "Aquí no hay un llavero de mentira", http.StatusNotFound)
			return
		}
		dice := r.URL.Query().Get("dice") == "no"
		l.QueDigaQueNo(dice)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"diceQueNo":%t}`, dice)
	})

	// Olvida que ya se ofreció desbloquear con el sistema en la bóveda abierta.
	//
	// **Existe porque una prueba estaba pasando por accidente.** La tarjeta que
	// ofrece Touch ID sólo sale si nadie ha contestado —`DesbloqueoSugeridoPara`, que
	// va por bóveda—, y la prueba que la comprueba **contesta**. Como los dos temas
	// comparten este Go y esta bóveda, en cuanto `claro` la ejecutaba, `oscuro` no
	// podía verla nunca más: lo que la salvaba era que **otra prueba le borraba la
	// marca sin querer** —cualquier guardado a medias de preferencias lo hace—. O sea
	// que el verde no decía lo que parecía decir.
	//
	// Lo mismo que `_llavero` y `_excluir`: monta un estado que la prueba no puede
	// crear de otra forma, y **no existe en el binario del cliente**, que es lo que
	// impide que esto sea una puerta nueva a la bóveda.
	// **No pide bóveda abierta a propósito**: la pantalla lee ese estado **una sola
	// vez, al montarse**, así que reponerlo con la bóveda ya abierta no sirve de
	// nada —la tarjeta ya se decidió—. Hay que poder llamarlo **antes** de cargar la
	// página, y entonces no hay ninguna bóveda que nombrar. Borrar la marca sin
	// mirar qué bóveda es no pierde nada: sólo hay una, y lo peor que puede pasar es
	// que se vuelva a ofrecer algo que se ofrece una vez.
	mux.HandleFunc("/api/_sugerencia", func(w http.ResponseWriter, r *http.Request) {
		p := a.ajustes.Ver()
		antes := p.DesbloqueoSugeridoPara
		p.DesbloqueoSugeridoPara = ""
		if err := a.ajustes.Guardar(p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"olvidada":%q}`, antes)
	})

	// Siembra un sitio excluido en la bóveda abierta, para poder probar la lista de
	// Ajustes.
	//
	// **Va aquí por lo mismo que `_llavero`, y merece decirse.** «Nunca en este
	// sitio» se decide en la tarjeta de la página y llega por el canal de la
	// extensión (ADR 0032): desde la ventana solo se puede **quitar**, nunca poner.
	// Así que una prueba de `make e2e` no tenía forma de crear ese estado, y la lista
	// se quedaba sin probar —estaba en `docs/deuda.md`—. La salida fácil habría sido
	// exportar un `ExcluirSitio` en `App`, y eso es justo lo que la lista blanca del
	// puente existe para impedir: sería una puerta nueva a la bóveda abierta **en el
	// binario del cliente**, para comodidad de una prueba. Esto, en cambio, no existe
	// fuera de la etiqueta `dev`.
	mux.HandleFunc("/api/_excluir", func(w http.ResponseWriter, r *http.Request) {
		b := a.boveda()
		if b == nil {
			http.Error(w, "La bóveda está cerrada", http.StatusConflict)
			return
		}
		dominio := r.URL.Query().Get("dominio")
		if err := b.Excluir(dominio); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"excluido":%q}`, dominio)
	})

	// Manda el aviso de versión nueva **ahora**, con lo que la última comprobación
	// encontró.
	//
	// Está aquí por lo mismo que `_llavero` y `_excluir`: sirve para montar un estado que
	// una prueba no puede crear de otra forma, y **no existe fuera de la etiqueta `dev`**.
	//
	// El estado en cuestión es «un aviso pasivo llega **después** de que la persona haya
	// cerrado la banda». En producción eso lo manda `mirarSiToca`, que es de minúscula a
	// propósito —no es algo que la ventana deba poder pedir— y además solo sale una vez cada
	// 24 horas, así que desde una prueba no hay forma de provocarlo. Y es justo el caso que
	// se escapó: la banda volvía a abrirse sola y tiró dos intentos de publicar la 2.65.
	// Sirve el fichero que se descargaría, para que probar la descarga no dependa de
	// la red.
	//
	// Está aquí por lo mismo que `_llavero`, `_excluir` y `_novedad`, y **no existe
	// fuera de la etiqueta `dev`**.
	mux.HandleFunc("/api/_descarga", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprint(len(ficheroDeMentira)))
		_, _ = w.Write(ficheroDeMentira)
	})

	mux.HandleFunc("/api/_novedad", func(w http.ResponseWriter, r *http.Request) {
		// **Y con `?local=si`, una novedad que se baja de aquí mismo.**
		//
		// Sin esto, probar el botón de descargar **pasaba por api.github.com y por el
		// adjunto de verdad de la última publicación**: unos cuantos megas por tanda y
		// por tema, con el reloj de la prueba corriendo. Pasaba casi siempre y falló el
		// 2026-11-20 en el tema oscuro, con la puerta de una publicación detrás — y lo
		// que falla por la red no se arregla volviendo a correrlo. Así que la novedad se
		// planta con la URL de este mismo servidor (`r.Host`), su tamaño y **su resumen
		// de verdad**, que es lo único que no se puede falsear: `Descargar` lo comprueba
		// mientras baja, así que el camino que se ejercita sigue siendo el entero.
		if r.URL.Query().Get("local") == "si" {
			suma := sha256.Sum256(ficheroDeMentira)
			a.act.mu.Lock()
			a.act.encontrada = actualizacion.Novedad{
				Hay:     true,
				Version: "99.0.0",
				Pagina:  "https://github.com/webcafeina/esfinge/releases",
				Fichero: "esfinge-de-mentira.bin",
				URL:     "http://" + r.Host + "/api/_descarga",
				Bytes:   int64(len(ficheroDeMentira)),
				Resumen: hex.EncodeToString(suma[:]),
			}
			a.act.mu.Unlock()
		}

		// **Y con `?autorizacion=si`, como si esta copia viniera de un paquete** (ADR 0062).
		// Aquí no hay `pkexec` ni un `/usr/share/esfinge/gestor`, así que la frase que avisa
		// del diálogo de la contraseña no se podría ver de otra forma — y es justo la que no
		// puede faltar, porque sin ella lo sensato al ver ese diálogo es cancelarlo. Monta un
		// estado que una prueba no puede crear de otra manera, igual que `_llavero`.
		//
		// Se apunta **en el actualizador y no en este aviso**: falseándolo sólo aquí, la
		// pregunta de al montar devolvía la banda al modo de verdad.
		a.act.mu.Lock()
		if r.URL.Query().Get("autorizacion") == "si" {
			a.act.comoSiFueraDeUnPaquete = true
		}
		n := a.act.encontrada
		aviso := a.act.novedad(n)
		a.act.mu.Unlock()
		if !n.Hay {
			http.Error(w, "Todavía no se ha encontrado ninguna versión nueva", http.StatusConflict)
			return
		}
		a.sistema.Avisar(EventoNovedad, aviso)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"avisado":%q}`, n.Version)
	})

	mux.HandleFunc("/api/eventos", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		c := make(chan []byte, 16)
		s.mu.Lock()
		s.oyentes[c] = true
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			delete(s.oyentes, c)
			s.mu.Unlock()
		}()

		for {
			select {
			case <-r.Context().Done():
				return
			case m := <-c:
				w.Write(m)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	})

	// Los métodos de App se publican por reflexión, con el mismo nombre que
	// tendrán en la ventana. Así no hay una lista que mantener al día: si un
	// método existe en Go, existe aquí, y el puente de la interfaz no distingue.
	valor := reflect.ValueOf(a)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		nombre := strings.TrimPrefix(r.URL.Path, "/api/")
		metodo := valor.MethodByName(nombre)
		if !metodo.IsValid() {
			responderError(w, http.StatusNotFound, "No existe el método "+nombre)
			return
		}

		var crudos []json.RawMessage
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&crudos)
		}

		tipo := metodo.Type()
		if len(crudos) != tipo.NumIn() {
			responderError(w, http.StatusBadRequest,
				fmt.Sprintf("%s espera %d argumentos y han llegado %d", nombre, tipo.NumIn(), len(crudos)))
			return
		}

		args := make([]reflect.Value, tipo.NumIn())
		for i := range args {
			v := reflect.New(tipo.In(i))
			if err := json.Unmarshal(crudos[i], v.Interface()); err != nil {
				responderError(w, http.StatusBadRequest,
					fmt.Sprintf("El argumento %d de %s no cuadra: %v", i+1, nombre, err))
				return
			}
			args[i] = v.Elem()
		}

		salidas := metodo.Call(args)

		// Si alguno de los valores devueltos es un error, manda.
		//
		// Lo que decide cuál es el error es el tipo declarado, no el valor: un
		// error nil llega como interfaz vacía y no supera una aserción de tipo, así
		// que preguntándole al valor se acaba tomando el error por resultado y
		// devolviendo null cuando todo ha ido bien.
		var resultado any
		for i, s := range salidas {
			if tipo.Out(i) == tipoError {
				if err, _ := s.Interface().(error); err != nil {
					responderError(w, http.StatusBadRequest, err.Error())
					return
				}
				continue
			}
			resultado = s.Interface()
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resultado)
	})

	log.Printf("Servidor de desarrollo en http://%s · ficheros de prueba en %s", direccion, s.Carpeta)
	return http.ListenAndServe(direccion, mux)
}

// tipoError es el tipo declarado de un valor de retorno de error.
var tipoError = reflect.TypeOf((*error)(nil)).Elem()

func responderError(w http.ResponseWriter, estado int, mensaje string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": mensaje})
}
