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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

// SistemaDeDesarrollo hace de escritorio cuando no hay escritorio.
//
// Los diálogos de fichero no existen en un navegador, así que se resuelven
// listando una carpeta que se indica al arrancar. Es suficiente para ejercitar
// el camino entero —elegir, cifrar, ver el resultado— sin fingir nada del lado
// de Go.
type SistemaDeDesarrollo struct {
	Carpeta string

	mu      sync.Mutex
	oyentes map[chan []byte]bool
}

func NuevoSistemaDeDesarrollo(carpeta string) *SistemaDeDesarrollo {
	return &SistemaDeDesarrollo{Carpeta: carpeta, oyentes: map[chan []byte]bool{}}
}

func (s *SistemaDeDesarrollo) ElegirFicheros(_ string, varios bool) ([]string, error) {
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

func (s *SistemaDeDesarrollo) ElegirDondeGuardar(_, nombreSugerido string) (string, error) {
	return filepath.Join(s.Carpeta, nombreSugerido), nil
}

// Avisar reparte el evento entre los navegadores conectados.
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
