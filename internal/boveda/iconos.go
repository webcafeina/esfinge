package boveda

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/escritura"
)

// Los iconos de los sitios, en un fichero aparte y cifrado con la misma llave.
//
// # Por qué aparte y no dentro de cada entrada
//
// Lo evidente era guardar el icono en la `Entrada`, y `Extra` incluso lo habría
// protegido de las versiones viejas. Se descartó por dos cosas medidas y una de
// fondo:
//
//   - **Cada guardado de la bóveda lee el fichero entero, copia el anterior
//     entero y escribe el nuevo entero, con `fsync`.** Sesenta y cinco iconos
//     multiplican por diez ese fichero, y esa cuenta se paga **al confirmar cada
//     edición de una contraseña**, que no tiene nada que ver con los iconos.
//   - **La lista de la ventana viaja entera en cada tecla del buscador.** Con el
//     icono dentro de la entrada, teclear «ban» son tres viajes de todos los
//     iconos por el puente.
//   - Y la de fondo: un icono es **dato derivado**, no dato de nadie. Se puede
//     tirar y se vuelve a traer. Meterlo en la bóveda lo convierte en algo que hay
//     que exportar, migrar y respetar para siempre.
//
// # Y por qué cifrado, si un icono es público
//
// El icono no es secreto; **la lista sí**. Una carpeta con `banco.es.png` y
// `hacienda.es.png` dice exactamente qué sitios hay dentro de la bóveda, que es
// lo que la bóveda existe para ocultar. Y hashear el nombre del fichero no salva
// nada: un diccionario de dominios se invierte en segundos.
//
// Va sellado con la clave de bóveda y con `PerfilLlave`, igual que el cuerpo:
// estirar una clave que ya es aleatoria no añadiría nada y costaría medio segundo.

// sufijoIconos es lo que se le pega a la ruta de la bóveda.
const sufijoIconos = ".iconos"

// cuantoDuraUnFallo es lo que se tarda en volver a preguntar por un sitio que no
// dio icono.
//
// **Recordar el fracaso es la mitad del diseño.** Sin esto, los sitios que no
// tienen icono —que son unos cuantos— se vuelven a preguntar cada vez que se abre
// la bóveda, para siempre: una ráfaga permanente de peticiones que no van a dar
// nada. Un dominio que no da icono hoy no lo da la semana que viene.
const cuantoDuraUnFallo = 30 * 24 * time.Hour

// Icono es lo que se sabe del icono de un sitio.
type Icono struct {
	// URI es el «data:image/png;base64,…», o vacío si el sitio no dio ninguno.
	URI string `json:"uri,omitempty"`
	// Mirado es cuándo se preguntó por última vez, en RFC3339.
	Mirado string `json:"mirado"`
}

type almacenIconos struct {
	Iconos map[string]Icono `json:"iconos"`
}

func (b *Boveda) rutaIconos() string { return b.ruta + sufijoIconos }

// Iconos devuelve lo que hay guardado, por anfitrión.
//
// Si el fichero no está, está a medias o no abre con esta llave, se devuelve
// vacío **sin error**: es una caché, y una caché que no se puede leer solo
// significa que hay que volver a llenarla. Nada de lo que hay dentro es
// insustituible.
func (b *Boveda) Iconos() map[string]Icono {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.leerIconos()
}

func (b *Boveda) leerIconos() map[string]Icono {
	if b.llave == nil {
		return nil
	}
	datos, err := os.ReadFile(b.rutaIconos())
	if err != nil {
		return nil
	}
	claro, err := cripto.AbrirTexto(string(datos), b.llave)
	if err != nil {
		return nil
	}
	defer cripto.Borrar(claro)

	var a almacenIconos
	if err := json.Unmarshal(claro, &a); err != nil {
		return nil
	}
	return a.Iconos
}

// PonerIconos guarda lo que se acaba de traer, mezclándolo con lo que hubiera.
//
// Se escribe **por tandas y no por icono**: cada escritura sella y sincroniza un
// fichero, y hacerlo sesenta y cinco veces seguidas sería sesenta y cinco veces
// el mismo trabajo para el mismo resultado.
func (b *Boveda) PonerIconos(nuevos map[string]Icono) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}

	todos := b.leerIconos()
	if todos == nil {
		todos = map[string]Icono{}
	}
	for anfitrion, icono := range nuevos {
		todos[anfitrion] = icono
	}

	crudo, err := json.Marshal(almacenIconos{Iconos: todos})
	if err != nil {
		return err
	}
	sellado, err := cripto.SellarTexto(crudo, b.llave, cripto.PerfilLlave)
	if err != nil {
		return err
	}
	// Sin copia `.anterior`: es una caché. Guardar la generación anterior de algo
	// que se puede volver a traer es pagar el doble por nada.
	return escritura.Atomica(b.rutaIconos(), escritura.Opciones{CrearCarpeta: true},
		func(w io.Writer) error {
			_, err := io.WriteString(w, sellado)
			return err
		})
}

// TocaMirar dice si hay que preguntarle a este sitio.
//
// Que no, cuando ya se tiene el icono o cuando se preguntó hace poco y no había.
func TocaMirar(i Icono, ahora time.Time) bool {
	if i.URI != "" {
		return false
	}
	if i.Mirado == "" {
		return true
	}
	cuando, err := time.Parse(time.RFC3339, i.Mirado)
	if err != nil {
		return true
	}
	return ahora.Sub(cuando) >= cuantoDuraUnFallo
}

// BorrarIconos se lleva el fichero de la caché. Lo llama quien borra la bóveda:
// dejarlo detrás sería dejar la lista de sitios en el disco, que es justo lo que
// no puede quedar.
func (b *Boveda) BorrarIconos() {
	_ = os.Remove(b.rutaIconos())
}

// RutaDeIconos dice dónde vive la caché, para poder borrarla desde fuera cuando
// ya no hay bóveda que abrir.
func RutaDeIconos(rutaBoveda string) string { return rutaBoveda + sufijoIconos }
