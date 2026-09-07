// Package app es lo que la ventana puede pedirle a Esfinge.
//
// Todo lo que aquí se exporta acaba siendo una función llamable desde la
// interfaz. Por eso los tipos son sencillos y serializables: lo que cruza el
// puente son cadenas, números y structs planos, nunca punteros ni interfaces.
//
// La lógica de verdad no vive aquí, vive en internal/cripto. Esto es la capa que
// traduce entre una ventana y ese núcleo, y la que decide qué se le enseña a
// quien está mirando.
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// App reúne el estado que dura lo que dura la aplicación abierta.
type App struct {
	ctx     context.Context
	version string

	hist    *Historial
	sistema Sistema

	// abiertoCon es el fichero con el que se arrancó, si se arrancó con uno.
	abiertoCon string
}

// Sistema es lo que la aplicación necesita del escritorio: los diálogos de
// fichero y el aviso de progreso.
//
// Está detrás de una interfaz porque en producción lo sirve Wails y durante el
// desarrollo lo sirve una implementación de mentira. Sin esta costura, probar la
// interfaz exigiría un entorno gráfico completo, y en la máquina donde se
// desarrolla no lo hay.
type Sistema interface {
	ElegirFicheros(titulo string, varios bool) ([]string, error)
	ElegirDondeGuardar(titulo, nombreSugerido string) (string, error)
	Avisar(evento string, datos any)
}

// Nueva construye la aplicación.
func Nueva(version string, sistema Sistema) *App {
	return &App{
		version: version,
		hist:    AbrirHistorial(),
		sistema: sistema,
	}
}

// Arrancar la llama Wails cuando la ventana está lista.
func (a *App) Arrancar(ctx context.Context) { a.ctx = ctx }

// Version es la que se enseña en «Acerca de».
func (a *App) Version() string { return a.version }

// AlAbrirCon guarda el fichero con el que se ha arrancado la aplicación, que es
// lo que llega al hacer doble clic en un .esf.
func (a *App) AlAbrirCon(ruta string) { a.abiertoCon = ruta }

// FicheroDeArranque lo consulta la interfaz al empezar para saber si tiene que
// abrirse directamente en descifrar, con el fichero ya puesto. Se entrega una
// sola vez: si se devolviera siempre, volver al menú y cambiar de pestaña
// repondría el fichero una y otra vez.
func (a *App) FicheroDeArranque() string {
	ruta := a.abiertoCon
	a.abiertoCon = ""
	return ruta
}

// Resultado es lo que sale de cifrar o descifrar un texto.
type Resultado struct {
	Texto string `json:"texto"`
	// Aviso es lo que conviene que la persona lea antes de cerrar la ventana.
	Aviso string `json:"aviso"`
}

// CifrarTexto sella un secreto corto y lo devuelve como contenedor de una línea.
func (a *App) CifrarTexto(texto, clave string) (Resultado, error) {
	if strings.TrimSpace(texto) == "" {
		return Resultado{}, fmt.Errorf("Falta el contenido")
	}
	if clave == "" {
		return Resultado{}, fmt.Errorf("Falta la clave")
	}

	k := []byte(clave)
	defer cripto.Borrar(k)

	cifrado, err := cripto.SellarTexto([]byte(texto), k, cripto.PerfilInteractivo)
	if err != nil {
		return Resultado{}, err
	}

	a.hist.Anotar(AccionCifrar, "Un texto", "")
	return Resultado{
		Texto: cifrado,
		Aviso: "Sin la clave, esto no lo abre nadie: si la pierdes, se pierde el contenido",
	}, nil
}

// DescifrarTexto abre un contenedor de una línea.
func (a *App) DescifrarTexto(texto, clave string) (Resultado, error) {
	if strings.TrimSpace(texto) == "" {
		return Resultado{}, fmt.Errorf("Falta el texto cifrado")
	}
	if clave == "" {
		return Resultado{}, fmt.Errorf("Falta la clave")
	}

	k := []byte(clave)
	defer cripto.Borrar(k)

	datos, err := cripto.AbrirTexto(texto, k)
	if err != nil {
		return Resultado{}, err
	}

	a.hist.Anotar(AccionDescifrar, "Un texto", "")
	return Resultado{
		Texto: string(datos),
		Aviso: "No lo dejes en pantalla más de lo necesario",
	}, nil
}

// ResultadoFichero cuenta cómo le ha ido a cada fichero de una tanda.
type ResultadoFichero struct {
	Origen  string `json:"origen"`
	Destino string `json:"destino"`
	Error   string `json:"error"`
}

// Progreso es lo que se emite mientras se trabaja en una tanda de ficheros, para
// que la ventana pueda enseñar por dónde va. Sin esto, cifrar diez ficheros son
// cinco segundos de nada en pantalla: la derivación de la clave se repite en
// cada uno, a propósito, porque cada contenedor lleva su propia sal.
type Progreso struct {
	Hechos int    `json:"hechos"`
	Total  int    `json:"total"`
	Actual string `json:"actual"`
}

// Nombres de los eventos que viajan hasta la ventana.
const (
	EventoProgreso = "progreso"
	// EventoFicheroAbierto llega cuando el sistema manda un fichero con la
	// ventana ya abierta: en macOS, doble clic en un .esf mientras Esfinge corre.
	EventoFicheroAbierto = "fichero-abierto"
)

// CifrarFicheros sella una tanda con la misma clave.
func (a *App) CifrarFicheros(rutas []string, clave string) ([]ResultadoFichero, error) {
	return a.porTanda(rutas, clave, AccionCifrar, CifrarFichero)
}

// DescifrarFicheros abre una tanda con la misma clave.
func (a *App) DescifrarFicheros(rutas []string, clave string) ([]ResultadoFichero, error) {
	return a.porTanda(rutas, clave, AccionDescifrar, DescifrarFichero)
}

func (a *App) porTanda(
	rutas []string, clave string, accion Accion,
	trabajo func(string, []byte) (string, error),
) ([]ResultadoFichero, error) {
	if len(rutas) == 0 {
		return nil, fmt.Errorf("No hay ningún fichero")
	}
	if clave == "" {
		return nil, fmt.Errorf("Falta la clave")
	}

	k := []byte(clave)
	defer cripto.Borrar(k)

	out := make([]ResultadoFichero, 0, len(rutas))
	for i, ruta := range rutas {
		a.sistema.Avisar(EventoProgreso, Progreso{
			Hechos: i, Total: len(rutas), Actual: filepath.Base(ruta),
		})

		r := ResultadoFichero{Origen: ruta}
		destino, err := trabajo(ruta, k)
		switch {
		case err != nil:
			// Un fichero que falla no detiene la tanda: se anota y se sigue. Parar
			// en el primer error dejaría el resto sin hacer y sin explicación.
			r.Error = err.Error()
		default:
			r.Destino = destino
			a.hist.Anotar(accion, filepath.Base(ruta), destino)
		}
		out = append(out, r)
	}

	a.sistema.Avisar(EventoProgreso, Progreso{
		Hechos: len(rutas), Total: len(rutas),
	})
	return out, nil
}

// Fuerza es la valoración de una clave, para el medidor.
type Fuerza struct {
	Nivel      int     `json:"nivel"`
	Bits       float64 `json:"bits"`
	Etiqueta   string  `json:"etiqueta"`
	Sugerencia string  `json:"sugerencia"`
}

// EvaluarClave valora una clave mientras se teclea.
func (a *App) EvaluarClave(clave string) Fuerza {
	f := cripto.Evaluar(clave)
	return Fuerza{Nivel: f.Nivel, Bits: f.Bits, Etiqueta: f.Etiqueta, Sugerencia: f.Sugerencia}
}

// Alfabeto describe una forma de generar contraseñas, para poder pintar el
// selector sin que la interfaz tenga que saberse los nombres de memoria.
type Alfabeto struct {
	Nombre    string `json:"nombre"`
	Etiqueta  string `json:"etiqueta"`
	SeguroURL bool   `json:"seguroURL"`
	Aviso     string `json:"aviso"`
}

// Alfabetos son los disponibles, en el orden en que conviene enseñarlos.
func (a *App) Alfabetos() []Alfabeto {
	etiquetas := map[string]string{
		"hex":      "Hexadecimal",
		"alnum":    "Letras y números",
		"simbolos": "Con símbolos",
	}
	orden := []cripto.Alfabeto{cripto.AlfHex, cripto.AlfAlnum, cripto.AlfSimbolos}

	out := make([]Alfabeto, 0, len(orden))
	for _, al := range orden {
		out = append(out, Alfabeto{
			Nombre:    al.Nombre,
			Etiqueta:  etiquetas[al.Nombre],
			SeguroURL: al.SeguroURL,
			Aviso:     al.Aviso,
		})
	}
	return out
}

// MedidaContrasena traduce entre las dos formas de pedir una contraseña.
type MedidaContrasena struct {
	Bytes      int `json:"bytes"`
	Caracteres int `json:"caracteres"`
	Bits       int `json:"bits"`
}

// Límites de longitud, en caracteres. El mínimo no es una opinión: por debajo de
// ahí una contraseña se adivina, y Generar ya los rechaza.
const (
	CaracteresMinimo = 16
	CaracteresMaximo = 96
)

// MedirPorCaracteres dice qué sale de pedir esa cantidad de caracteres.
func (a *App) MedirPorCaracteres(caracteres int, alfabeto string) (MedidaContrasena, error) {
	al, ok := cripto.Alfabetos[alfabeto]
	if !ok {
		return MedidaContrasena{}, fmt.Errorf("El alfabeto «%s» no existe", alfabeto)
	}
	if caracteres < CaracteresMinimo {
		caracteres = CaracteresMinimo
	}
	if caracteres > CaracteresMaximo {
		caracteres = CaracteresMaximo
	}

	bytes := cripto.BytesParaCaracteres(al, caracteres)
	return MedidaContrasena{
		Bytes:      bytes,
		Caracteres: cripto.Caracteres(al, bytes),
		Bits:       bytes * 8,
	}, nil
}

// GenerarContrasena devuelve una contraseña al azar.
func (a *App) GenerarContrasena(bytes int, alfabeto string) (string, error) {
	al, ok := cripto.Alfabetos[alfabeto]
	if !ok {
		return "", fmt.Errorf("El alfabeto «%s» no existe", alfabeto)
	}
	return cripto.Generar(al, bytes)
}

// ElegirFicheros abre el diálogo del sistema.
func (a *App) ElegirFicheros(varios bool) ([]string, error) {
	return a.sistema.ElegirFicheros("Elige qué cifrar", varios)
}

// ElegirCifrados abre el diálogo del sistema filtrando por contenedores.
func (a *App) ElegirCifrados() ([]string, error) {
	return a.sistema.ElegirFicheros("Elige qué descifrar", true)
}

// GuardarTexto deja un texto donde diga el diálogo del sistema, y devuelve dónde
// ha quedado.
func (a *App) GuardarTexto(nombreSugerido, contenido string) (string, error) {
	destino, err := a.sistema.ElegirDondeGuardar("Guardar", nombreSugerido)
	if err != nil {
		return "", err
	}
	if destino == "" {
		return "", nil // lo ha cancelado, que no es un error
	}
	// 0600 desde el principio: entre crear el fichero y ajustar los permisos hay
	// una ventana en la que un secreto sería legible por cualquiera de la máquina.
	if err := os.WriteFile(destino, []byte(contenido+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("No he podido guardar el fichero: %w", err)
	}
	return destino, nil
}

// VerHistorial devuelve lo hecho últimamente.
func (a *App) VerHistorial() []Entrada { return a.hist.Entradas() }

// VaciarHistorial lo borra del disco.
func (a *App) VaciarHistorial() error { return a.hist.Vaciar() }

// DondeVive el historial, para poder decirlo en la propia interfaz en vez de
// obligar a creerse que no se guarda nada raro.
func (a *App) DondeVive() string { return a.hist.Ruta() }
