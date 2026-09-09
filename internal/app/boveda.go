package app

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/escritura"
)

// Lo que la ventana puede pedirle a la bóveda.
//
// **Todo lo de aquí cruza el puente**, así que la superficie se ha mantenido
// deliberadamente corta y hay una lista blanca que lo vigila
// (`TestLoQueCruzaElPuenteEstaEnLaLista`). La regla que gobierna este fichero:
//
//	los secretos salen de uno en uno, y solo cuando alguien los pide.
//
// La lista de entradas viaja sin contraseñas. Eso hace más por que un secreto no
// acabe en un volcado de memoria que todo el borrado de búferes junto, porque en
// cuanto algo cruza el puente se serializa a JSON y vive en el montón del
// webview, fuera de nuestro alcance y hasta que su recolector decida.

// EstadoBoveda es lo que la ventana necesita para saber qué enseñar.
type EstadoBoveda struct {
	Existe      bool   `json:"existe"`
	Abierta     bool   `json:"abierta"`
	Ruta        string `json:"ruta"`
	Cuantas     int    `json:"cuantas"`
	SoloLectura bool   `json:"soloLectura"`
	// MinutosParaBloquear es lo que dice Ajustes, para poder enseñarlo.
	MinutosParaBloquear int `json:"minutosParaBloquear"`
}

// ResumenImportacion es lo que se cuenta después de traer un CSV de otro gestor.
type ResumenImportacion struct {
	Metidas int `json:"metidas"`
	// Repetidas ya estaban exactamente igual y **no se han metido**: es lo que
	// hace que pasar dos veces el mismo fichero no cambie nada.
	Repetidas int `json:"repetidas"`
	// Conflictos son cuentas que ya estaban con otra contraseña. Ésas sí entran,
	// marcadas, porque una de las dos está mal y no lo decide un importador.
	Conflictos int    `json:"conflictos"`
	DeDonde    string `json:"deDonde"`
	// Fichero es el CSV del que se importó. Se devuelve para poder ofrecer
	// borrarlo: **es una lista de contraseñas en claro en el disco**.
	Fichero string `json:"fichero"`
}

func rutaBoveda() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "boveda.esfinge")
}

// EstadoBoveda dice si hay bóveda, si está abierta y cuánto lleva dentro.
func (a *App) EstadoBoveda() EstadoBoveda {
	ruta := rutaBoveda()
	e := EstadoBoveda{
		Ruta: ruta,
		// De los ajustes y no del vigilante: el vigilante lo lleva un cerrojo que
		// no es de aquí, y el número que hay que enseñar es el que está guardado.
		MinutosParaBloquear: a.ajustes.Ver().MinutosParaBloquear,
	}
	if _, err := os.Stat(ruta); err == nil {
		e.Existe = true
	}
	if a.bov != nil && a.bov.Abierta() {
		e.Abierta = true
		e.Cuantas = a.bov.Cuantas()
		e.SoloLectura = a.bov.SoloLectura()
	}
	return e
}

// CrearBoveda hace una bóveda nueva y devuelve la clave de recuperación.
//
// **Se devuelve una sola vez y no se guarda en ninguna parte.** Quien llame a
// esto tiene que enseñarla, insistir en que se apunte y no volver a pedirla:
// aquí no hay «vuélvemela a enseñar».
func (a *App) CrearBoveda(maestra string) (string, error) {
	if a.bov != nil && a.bov.Abierta() {
		return "", errors.New("Ya hay una bóveda abierta")
	}
	ruta := rutaBoveda()
	if ruta == "" {
		return "", errors.New("No encuentro dónde guardar la bóveda en este sistema")
	}
	if _, err := os.Stat(ruta); err == nil {
		return "", errors.New("Ya hay una bóveda aquí; ábrela en vez de crear otra")
	}

	b, recuperacion, err := boveda.Crear(ruta, maestra)
	if err != nil {
		return "", err
	}
	a.bov = b
	a.Actividad()
	return recuperacion, nil
}

// AbrirBoveda desbloquea con la contraseña maestra o con la de recuperación. No
// hace falta decir cuál es: se prueban las dos ranuras.
func (a *App) AbrirBoveda(llave string) error {
	ruta := rutaBoveda()
	// De paso se limpian los temporales que dejó una interrupción anterior. Solo
	// los del prefijo propio y con más de un día: uno reciente puede ser de otro
	// Esfinge escribiendo ahora mismo.
	escritura.LimpiarHuerfanos(filepath.Dir(ruta), 24*time.Hour)

	b, err := boveda.Abrir(ruta, llave)
	if err != nil {
		return err
	}
	a.bov = b
	a.Actividad()
	return nil
}

// CerrarBoveda la bloquea a mano, sin esperar al reloj.
func (a *App) CerrarBoveda() {
	if a.bov != nil {
		a.bov.Cerrar()
	}
}

// BuscarEnBoveda devuelve lo que encaje, **sin secretos**.
func (a *App) BuscarEnBoveda(q string) ([]boveda.Entrada, error) {
	if a.bov == nil || !a.bov.Abierta() {
		return nil, boveda.ErrCerrada
	}
	a.Actividad()
	return a.bov.Buscar(q), nil
}

// VerDeBoveda devuelve **una** entrada entera, con su contraseña.
//
// De una en una a propósito: es la diferencia entre que un volcado de memoria
// del webview tenga una contraseña o las tenga todas.
func (a *App) VerDeBoveda(id string) (boveda.Entrada, error) {
	if a.bov == nil || !a.bov.Abierta() {
		return boveda.Entrada{}, boveda.ErrCerrada
	}
	a.Actividad()
	e, hay := a.bov.Ver(id)
	if !hay {
		return boveda.Entrada{}, errors.New("Esa entrada ya no está en la bóveda")
	}
	return e, nil
}

// GuardarEnBoveda añade o cambia una entrada.
func (a *App) GuardarEnBoveda(e boveda.Entrada) error {
	if a.bov == nil || !a.bov.Abierta() {
		return boveda.ErrCerrada
	}
	a.Actividad()
	return a.bov.Poner(e)
}

// BorrarDeBoveda manda una entrada a la papelera.
func (a *App) BorrarDeBoveda(id string) error {
	if a.bov == nil || !a.bov.Abierta() {
		return boveda.ErrCerrada
	}
	a.Actividad()
	return a.bov.Borrar(id)
}

// CambiarMaestraDeBoveda pide la vieja aunque la bóveda ya esté abierta.
//
// **No es un trámite**: una bóveda abierta encima de una mesa es una bóveda a la
// que cualquiera que pase puede cambiarle la contraseña y dejar fuera a su
// dueño. Pedir la de antes convierte eso en un problema distinto.
func (a *App) CambiarMaestraDeBoveda(vieja, nueva string) error {
	if a.bov == nil || !a.bov.Abierta() {
		return boveda.ErrCerrada
	}
	if _, err := boveda.Abrir(rutaBoveda(), vieja); err != nil {
		return errors.New("La contraseña de ahora no es ésa")
	}
	a.Actividad()
	return a.bov.CambiarMaestra(nueva)
}

// BorrarBoveda quita la bóveda del disco. **No hay vuelta atrás.**
//
// Es la acción más destructiva de todo el programa: se lleva por delante todas
// las contraseñas de golpe, sin papelera y sin que la clave de recuperación
// sirva de nada, porque lo que se borra es el fichero que ella abriría.
//
// Pide la contraseña maestra, y conviene ser honesto sobre **contra qué protege
// eso**: no contra alguien que quiera hacer daño —quien puede abrir la aplicación
// también puede borrar el fichero desde el Finder— sino contra un clic mal dado y
// contra que lo haga quien no es el dueño de la bóveda estando ésta abierta
// encima de una mesa. Es la misma razón por la que cambiar la maestra pide la de
// antes.
//
// Se borran los dos ficheros: el de la bóveda y el `.anterior` con la generación
// previa, que existe justo para sobrevivir a un desastre y aquí sería un desastre
// a medias. Y los temporales que hubiera, que llevan una copia entera dentro.
func (a *App) BorrarBoveda(maestra string) error {
	ruta := rutaBoveda()
	if _, err := os.Stat(ruta); err != nil {
		return errors.New("Aquí no hay ninguna bóveda que borrar")
	}
	if _, err := boveda.Abrir(ruta, maestra); err != nil {
		return errors.New("Esa no es la contraseña de esta bóveda")
	}

	// Primero se cierra la que hubiera abierta: dejarla en memoria después de
	// borrar el fichero es tener una bóveda sin fichero, y el siguiente guardado
	// la escribiría otra vez.
	if a.bov != nil {
		a.bov.Cerrar()
		a.bov = nil
	}

	if err := os.Remove(ruta); err != nil {
		return err
	}
	// El resto es limpieza: que falte alguno no invalida el borrado, que ya está
	// hecho, y devolver un error aquí haría creer que no se ha borrado nada.
	_ = os.Remove(ruta + ".anterior")
	escritura.LimpiarHuerfanos(filepath.Dir(ruta), 0)
	return nil
}

// RotarRecuperacionDeBoveda genera una clave de recuperación nueva y deja la
// anterior inservible. También se devuelve **una sola vez**.
func (a *App) RotarRecuperacionDeBoveda() (string, error) {
	if a.bov == nil || !a.bov.Abierta() {
		return "", boveda.ErrCerrada
	}
	a.Actividad()
	return a.bov.RotarRecuperacion()
}

// ImportarEnBoveda trae un CSV de otro gestor, por el diálogo del sistema.
func (a *App) ImportarEnBoveda(deDonde string) (ResumenImportacion, error) {
	if a.bov == nil || !a.bov.Abierta() {
		return ResumenImportacion{}, boveda.ErrCerrada
	}
	rutas, err := a.sistema.ElegirFicheros("Elige la exportación de "+deDonde,
		a.ajustes.CarpetaDeAbrir(), false, FiltroTablas)
	if err != nil || len(rutas) == 0 {
		return ResumenImportacion{}, err
	}
	a.ajustes.RecordarCarpetaDeAbrir(filepath.Dir(rutas[0]))

	datos, err := os.ReadFile(rutas[0])
	if err != nil {
		return ResumenImportacion{}, err
	}
	entradas, _, err := boveda.Leer(datos, nil)
	if err != nil {
		return ResumenImportacion{}, err
	}

	r, err := a.bov.Importar(entradas, deDonde)
	if err != nil {
		return ResumenImportacion{}, err
	}
	a.Actividad()
	// **Nada de esto pasa por el historial**, y es una regla absoluta: ahí van
	// nombres de fichero (ADR 0010), y «credenciales-dashlane.csv» sería una
	// señal de tráfico apuntando a lo que alguien acaba de exportar en claro.
	return ResumenImportacion{
		Metidas: r.Metidas, Repetidas: r.Repetidas, Conflictos: r.Conflictos,
		DeDonde: deDonde, Fichero: rutas[0],
	}, nil
}

// ExportarBoveda escribe las entradas **en claro**, por el diálogo del sistema.
//
// Existe porque una bóveda de la que no se puede salir es una trampa. Quien
// llame a esto desde la ventana tiene que haber avisado antes y en grande.
func (a *App) ExportarBoveda() (string, error) {
	if a.bov == nil || !a.bov.Abierta() {
		return "", boveda.ErrCerrada
	}
	destino, err := a.sistema.ElegirDondeGuardar("Exportar la bóveda sin cifrar",
		"esfinge-sin-cifrar.csv", a.ajustes.CarpetaDeGuardar())
	if err != nil || destino == "" {
		return "", err
	}
	a.ajustes.RecordarCarpetaDeGuardar(filepath.Dir(destino))

	err = escritura.Atomica(destino, escritura.Opciones{}, a.bov.Exportar)
	if err != nil {
		return "", err
	}
	a.Actividad()
	return destino, nil
}

// BorrarElCSVImportado borra el fichero del que se acaba de importar.
//
// Se ofrece con insistencia porque **ese fichero es una lista de contraseñas en
// claro en el disco**, y quien lo exportó de Dashlane rara vez se acuerda de
// quitarlo. Con la honestidad de decir que en un disco de estado sólido un
// borrado normal no lo elimina de verdad.
func (a *App) BorrarElCSVImportado(ruta string) error {
	if ruta == "" {
		return nil
	}
	return os.Remove(ruta)
}
