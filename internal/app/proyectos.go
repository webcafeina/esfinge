package app

// Las bóvedas de proyecto en la ventana (ADR 0050): el registro, la lista y
// conmutar.
//
// Lo que hace que esto quepa en tan poco es la decisión de **una sola bóveda
// abierta a la vez**: `App.bov` sigue siendo un puntero y no un mapa, así que
// `boveda()` no cambia de firma y los cuarenta métodos que la piden siguen
// hablando de «la bóveda» sin saber cuál es. Conmutar es cambiar qué devuelve ese
// accesor.
//
// Y dos cosas que se cumplen en todo el fichero:
//
//   - **Lo de la cuenta es de la bóveda personal, siempre.** La sesión, el
//     verificador de acceso, la identidad para compartir y la posesión son de ella
//     y no de la activa, así que `rutaBovedaPrincipal()` y no `rutaActiva()`. Lo
//     contrario publicaría la identidad de un proyecto como la de la cuenta, y las
//     copias que te mandaran llegarían cifradas hacia una bóveda que puedes tener
//     cerrada: los dos lados harían lo suyo bien y el buzón diría «No se puede
//     abrir» sin que nadie pudiera entender por qué.
//   - **Con un proyecto abierto no se sincroniza** (hasta la E4). El servidor tiene
//     una bóveda por cuenta y la rechaza con 409 si no es la que espera, así que
//     subir un proyecto por la ruta de la personal no es que no funcione: es que
//     pediría el 409 en cada pasada.

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/escritura"
	"github.com/webcafeina/esfinge/internal/sincro"
)

// nombreDelRegistro vive al lado de las preferencias y de los navegadores.
const nombreDelRegistro = "bovedas.json"

// carpetaDeProyectos es donde van los ficheros, **nombrados por su referencia**.
const carpetaDeProyectos = "proyectos"

// registro es el índice de ficheros, y **no es la verdad**: la verdad son los
// ficheros, y los nombres viven cifrados dentro de la bóveda personal.
//
// Es una caché para poder encontrar un proyecto **sin abrir nada**, y por eso no
// lleva nombres: un `bovedas.json` con «Acme» dentro sería la lista de clientes de
// Webcafeína en claro en el disco, que es justo lo que la [ADR 0024] decidió no
// dejar pasar con los iconos.
type registroDeBovedas struct {
	Version   int               `json:"version"`
	Proyectos []proyectoEnDisco `json:"proyectos"`
}

type proyectoEnDisco struct {
	Ref string `json:"ref"`
	// Fichero es relativo a la carpeta de Esfinge, para que mover la carpeta de
	// configuración entera siga funcionando.
	Fichero string `json:"fichero"`
}

func carpetaDeEsfinge() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge")
}

func rutaDelRegistro() string {
	if c := carpetaDeEsfinge(); c != "" {
		return filepath.Join(c, nombreDelRegistro)
	}
	return ""
}

// rutaDeProyecto es dónde vive el fichero de una referencia.
func rutaDeProyecto(ref string) string {
	if c := carpetaDeEsfinge(); c != "" {
		return filepath.Join(c, carpetaDeProyectos, ref+".esfinge")
	}
	return ""
}

func leerRegistro() registroDeBovedas {
	r := registroDeBovedas{Version: 1}
	datos, err := os.ReadFile(rutaDelRegistro())
	if err != nil {
		return r
	}
	_ = json.Unmarshal(datos, &r)
	if r.Version == 0 {
		r.Version = 1
	}
	return r
}

// guardarRegistro escribe con `escritura.Atomica` y **no con `os.WriteFile`** como
// los navegadores: un registro a medias deja proyectos invisibles, y un proyecto
// invisible parece un proyecto perdido.
func guardarRegistro(r registroDeBovedas) error {
	ruta := rutaDelRegistro()
	if ruta == "" {
		return errors.New("No encuentro dónde guardar la lista de bóvedas en este sistema")
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	datos, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return escritura.Atomica(ruta, escritura.Opciones{Permisos: 0o600, CrearCarpeta: true},
		func(w io.Writer) error {
			_, err := w.Write(datos)
			return err
		})
}

func apuntarLaBoveda(ref string) error {
	r := leerRegistro()
	for _, p := range r.Proyectos {
		if p.Ref == ref {
			return nil
		}
	}
	r.Proyectos = append(r.Proyectos, proyectoEnDisco{
		Ref:     ref,
		Fichero: filepath.Join(carpetaDeProyectos, ref+".esfinge"),
	})
	return guardarRegistro(r)
}

func olvidarLaBovedaDelRegistro(ref string) error {
	r := leerRegistro()
	var quedan []proyectoEnDisco
	for _, p := range r.Proyectos {
		if p.Ref != ref {
			quedan = append(quedan, p)
		}
	}
	r.Proyectos = quedan
	return guardarRegistro(r)
}

// ------------------------------------------------------------------ la activa

// llaveDeLaPrincipal es con lo que se abre cualquier proyecto, venga de la bóveda
// personal abierta o de lo que se guardó al conmutar.
//
// **Y aquí está la relajación que hay que conocer** (ADR 0050): mientras hay un
// proyecto abierto, los 43 bytes de la clave de la personal se quedan en memoria.
// Sin eso, conmutar de un proyecto a otro pediría la contraseña maestra cada vez,
// que con muchos proyectos no es aceptable. Se borra con `cripto.Borrar` cuando el
// vigilante cierra por inactividad y al cerrar a mano, así que el reloj del bloqueo
// sigue significando lo que dice.
func (a *App) llaveDeLaPrincipal() []byte {
	a.mu.Lock()
	activa, guardada := a.activa, a.llavePrincipal
	a.mu.Unlock()
	if activa == "" {
		// La personal está abierta: se le pide a ella, que es la fuente.
		if b := a.boveda(); b != nil {
			return b.LlaveParaProyectos()
		}
		return nil
	}
	if len(guardada) == 0 {
		return nil
	}
	return append([]byte(nil), guardada...)
}

// ponerActiva apunta cuál es la bóveda abierta y, si es un proyecto, se queda la
// clave de la personal para poder conmutar.
func (a *App) ponerActiva(ref, nombre string, llavePrincipal []byte) {
	a.ponerActivaDe("", ref, nombre, llavePrincipal)
}

// ponerActivaDe es lo mismo diciendo **de quién** es (ADR 0052).
func (a *App) ponerActivaDe(dueno, ref, nombre string, llavePrincipal []byte) {
	a.mu.Lock()
	vieja := a.llavePrincipal
	a.duenoActivo = dueno
	a.activa, a.nombreActivo = ref, nombre
	if ref == "" {
		a.llavePrincipal = nil
	} else {
		a.llavePrincipal = append([]byte(nil), llavePrincipal...)
	}
	a.mu.Unlock()
	if vieja != nil {
		cripto.Borrar(vieja)
	}
}

// olvidarLaPrincipal borra la clave en memoria. La llaman el vigilante al bloquear
// y cerrar a mano: **si esto no se llamara, el reloj del bloqueo dejaría de ser
// verdad** para los proyectos.
func (a *App) olvidarLaPrincipal() {
	a.mu.Lock()
	vieja := a.llavePrincipal
	a.llavePrincipal, a.activa, a.nombreActivo, a.duenoActivo = nil, "", "", ""
	a.mu.Unlock()
	if vieja != nil {
		cripto.Borrar(vieja)
	}
}

// nombreDeLaActiva es cómo se llama la bóveda de proyecto abierta, o vacío.
func (a *App) nombreDeLaActiva() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.nombreActivo
}

// bovedaActiva es la referencia de lo que está abierto: vacío, la personal.
func (a *App) bovedaActiva() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.activa
}

// duenoDeLaActiva es de quién es la bóveda abierta: vacío si es mía —la personal o un
// proyecto— y la cuenta de la otra persona si me han dado acceso a ella (ADR 0052).
//
// Va aparte de `activa` y no pegado a ella en una cadena **a propósito**: lo que
// distingue un proyecto mío de una bóveda ajena no es un formato de texto que haya
// que recordar partir, y las dos cosas que lo preguntan —la ruta del servidor y si se
// puede escribir— quieren el dato, no la cadena.
func (a *App) duenoDeLaActiva() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.duenoActivo
}

// rutaActiva es el fichero de la bóveda abierta, o el de la personal si no hay
// ninguna.
func (a *App) rutaActiva() string {
	if ref := a.bovedaActiva(); ref != "" {
		return rutaDeProyecto(ref)
	}
	return rutaBovedaPrincipal()
}

// ------------------------------------------------------------------ lo que cruza el puente

// Proyecto es lo que la ventana necesita de cada bóveda de proyecto.
type Proyecto struct {
	Ref    string `json:"ref"`
	Nombre string `json:"nombre"`
	Creado string `json:"creado"`
	Usado  string `json:"usado"`
	// Archivado lo saca de la lista del día a día.
	Archivado bool `json:"archivado"`
	// EnEsteEquipo dice si su fichero está aquí. Falso es «dormido»: el proyecto
	// existe y está en el servidor, pero este equipo no lo ha bajado. **No es un
	// error y no se dice como tal.**
	EnEsteEquipo bool `json:"enEsteEquipo"`
	// Activo es el que está abierto ahora mismo.
	Activo bool `json:"activo"`
}

// conLaPersonal hace algo con la bóveda personal, esté abierta o no.
//
// Es lo que permite que el selector de proyectos funcione con un proyecto abierto:
// los nombres viven en el cuerpo cifrado de la personal, y con un proyecto delante
// la personal está cerrada. Con la clave que se guardó al conmutar se abre su
// fichero el instante que dure la operación y se cierra. **Secuencial y nunca dos
// abiertas de cara a quien mira**, que es lo que la ADR 0050 descartó.
// **Y va en fila de uno** (`muPersonal`). Dos de éstas a la vez abrirían dos
// instancias del mismo fichero, cada una con su copia del cuerpo, y el segundo
// guardado se llevaría por delante lo que escribió el primero: crear un proyecto
// mientras se apunta cuándo se abrió otro perdería uno de los dos. Es la misma
// lección que la cola de la bóveda de TypeScript (ADR 0040), y aquí cuesta una línea.
func (a *App) conLaPersonal(hacer func(*boveda.Boveda) error) error {
	a.muPersonal.Lock()
	defer a.muPersonal.Unlock()
	if a.bovedaActiva() == "" {
		b := a.boveda()
		if b == nil {
			return boveda.ErrCerrada
		}
		return hacer(b)
	}
	llave := a.llaveDeLaPrincipal()
	if len(llave) == 0 {
		return boveda.ErrCerrada
	}
	defer cripto.Borrar(llave)
	p, err := boveda.AbrirConLaClave(rutaBovedaPrincipal(), llave)
	if err != nil {
		return err
	}
	defer p.Cerrar()
	return hacer(p)
}

// Proyectos son las bóvedas de proyecto de la personal.
func (a *App) Proyectos() ([]Proyecto, error) {
	var lista []boveda.Proyecto
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		lista = b.Proyectos()
		return nil
	}); err != nil {
		return nil, err
	}
	activa := a.bovedaActiva()
	out := make([]Proyecto, 0, len(lista))
	for _, p := range lista {
		ruta := rutaDeProyecto(p.Ref)
		_, errF := os.Stat(ruta)
		out = append(out, Proyecto{
			Ref: p.Ref, Nombre: p.Nombre, Creado: p.Creado, Usado: p.Usado,
			Archivado: p.Archivado, EnEsteEquipo: errF == nil, Activo: p.Ref == activa,
		})
	}
	return out, nil
}

// CrearProyecto hace una bóveda de proyecto y devuelve su referencia.
//
// **No pide contraseña y no devuelve clave de recuperación**: se abre con la
// personal, y la de recuperación de la personal lo recupera. La pantalla que llama
// a esto tiene que decirlo, porque quien ha creado una bóveda antes espera la
// ceremonia y su ausencia sin explicar parece un olvido.
func (a *App) CrearProyecto(nombre string) (string, error) {
	if nombre == "" {
		return "", errors.New("Ponle un nombre al proyecto")
	}
	// **Se puede crear desde cualquier bóveda**, también desde dentro de otro
	// proyecto: la lista vive en la personal y `conLaPersonal` llega a ella con la
	// clave que se guardó al conmutar. Prohibirlo obligaría a volver a la bóveda
	// personal —y a teclear la maestra— para una cosa que no lo necesita.
	llave := a.llaveDeLaPrincipal()
	if len(llave) == 0 {
		return "", boveda.ErrCerrada
	}
	defer cripto.Borrar(llave)

	bruta, err := cripto.Azar(8)
	if err != nil {
		return "", err
	}
	ref := hex.EncodeToString(bruta)
	ruta := rutaDeProyecto(ref)
	if ruta == "" {
		return "", errors.New("No encuentro dónde guardar la bóveda en este sistema")
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return "", err
	}
	if _, err := os.Stat(ruta); err == nil {
		return "", errors.New("Ya hay una bóveda con esa referencia")
	}

	p, err := boveda.CrearProyecto(ruta, llave)
	if err != nil {
		return "", err
	}
	p.Cerrar()

	// El orden importa: **primero el fichero, después apuntarlo**. Al revés, un
	// fallo al crear dejaría en la lista un proyecto que no existe — y un proyecto
	// de la lista que no existe es el que se enseña como «dormido en este equipo»,
	// o sea que parecería que está en el servidor y no está en ningún sitio.
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		return b.PonerProyecto(boveda.Proyecto{
			Ref: ref, Nombre: nombre,
			Creado: time.Now().UTC().Format(time.RFC3339),
		})
	}); err != nil {
		return "", err
	}
	if err := apuntarLaBoveda(ref); err != nil {
		// El registro es una caché: si no se puede escribir, el proyecto existe
		// igual y se reconstruye al arrancar. No se deshace nada por esto.
		log.Printf("esfinge: no se ha podido apuntar el proyecto en el registro: %v", err)
	}
	a.Actividad()
	return ref, nil
}

// BajarProyecto trae del servidor una bóveda de proyecto que todavía no está en
// este equipo: lo que la lista llama «dormido».
//
// **Se baja entera y se escribe tal cual.** No hay nada que fundir —aquí no hay
// fichero con el que fundir— y la primera pasada de la sincronización se encontrará
// lo mismo arriba y abajo. Y se abre con la bóveda personal sin preguntar nada,
// porque la ranura viaja **dentro del fichero** (ADR 0050).
func (a *App) BajarProyecto(ref string) error {
	if ref == "" {
		return errors.New("Esa no es una bóveda de proyecto")
	}
	ruta := rutaDeProyecto(ref)
	if ruta == "" {
		return errors.New("No encuentro dónde guardar la bóveda en este sistema")
	}
	if _, err := os.Stat(ruta); err == nil {
		return nil // ya está aquí: bajarla otra vez pisaría lo que hubiera sin fundir
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	datos, _, _, err := a.cliente().BajarDe(a.ctxCuenta(), token, ref, 0)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	// **Y se comprueba que abre antes de dejarla puesta.** Un fichero en esa carpeta
	// es un proyecto para todo lo demás; si lo que bajó no lo abre esta bóveda
	// personal, es mejor no tenerlo que tenerlo y que falle al abrirlo.
	llave := a.llaveDeLaPrincipal()
	if len(llave) == 0 {
		return boveda.ErrCerrada
	}
	defer cripto.Borrar(llave)
	p, err := boveda.AbrirProyectoBytes("", datos, llave)
	if err != nil {
		return err
	}
	p.Cerrar()

	if err := escritura.Atomica(ruta, escritura.Opciones{Permisos: 0o600, CrearCarpeta: true},
		func(w io.Writer) error {
			_, err := w.Write(datos)
			return err
		}); err != nil {
		return err
	}
	if err := apuntarLaBoveda(ref); err != nil {
		log.Printf("esfinge: no se ha podido apuntar el proyecto bajado en el registro: %v", err)
	}
	a.Actividad()
	return nil
}

// AbrirProyecto conmuta: cierra lo que haya abierto y abre ese proyecto.
func (a *App) AbrirProyecto(ref string) error {
	if ref == "" {
		return a.VolverALaBovedaPersonal()
	}
	llave := a.llaveDeLaPrincipal()
	if len(llave) == 0 {
		return boveda.ErrCerrada
	}
	defer cripto.Borrar(llave)

	ruta := rutaDeProyecto(ref)
	if ruta == "" {
		return errors.New("No encuentro dónde está esa bóveda en este sistema")
	}
	if _, err := os.Stat(ruta); err != nil {
		// Dormido en este equipo. Es un estado y no un fallo, y se dice así.
		return errors.New("Esa bóveda no está en este equipo todavía")
	}
	p, err := boveda.AbrirProyecto(ruta, llave)
	if err != nil {
		return err
	}
	nombre := ""
	_ = a.conLaPersonal(func(b *boveda.Boveda) error {
		if x, hay := b.Proyecto(ref); hay {
			nombre = x.Nombre
		}
		return nil
	})
	a.conmutarA(p, ref, nombre, llave)
	return nil
}

// ------------------------------------------------- las que me han compartido

// carpetaDeCompartidas separa lo ajeno de lo mío **en el disco**, que es lo que hace
// que una no se pueda confundir con la otra al mirar la carpeta o al borrar.
const carpetaDeCompartidas = "compartidas"

// rutaDeCompartida: el fichero local de una bóveda ajena.
//
// Lleva el dueño en el nombre porque **la referencia es de su cuenta, no de la mía**:
// dos personas distintas pueden compartirme bóvedas con la misma referencia sin saber
// nada la una de la otra, y si el fichero se llamara solo por la referencia la segunda
// pisaría a la primera. Es la misma razón por la que el nombre de un proyecto no
// nombra su fichero, una vuelta más arriba.
func rutaDeCompartida(dueno, ref string) string {
	if c := carpetaDeEsfinge(); c != "" {
		return filepath.Join(c, carpetaDeCompartidas, dueno+"-"+ref+".esfinge")
	}
	return ""
}

// BajarCompartida se trae a este equipo una bóveda a la que me han dado acceso.
//
// Es el espejo de `BajarProyecto` con dos diferencias: se pide a la cuenta de la otra
// persona, y **se comprueba que abre con mi ranura antes de dejarla puesta**. Lo
// segundo no es celo: un fichero en esa carpeta es una bóveda compartida para todo lo
// demás, y si lo que bajó no lo abre mi identidad es mejor no tenerlo que tenerlo y
// que falle al abrirlo, cuando ya nadie sabe de dónde salió.
func (a *App) BajarCompartida(dueno, ref string) error {
	c, hay := a.laCompartida(dueno, ref)
	if !hay {
		return errors.New("Esa bóveda compartida ya no está en tu lista")
	}
	ruta := rutaDeCompartida(dueno, ref)
	if ruta == "" {
		return errors.New("No encuentro dónde guardar la bóveda en este sistema")
	}
	if _, err := os.Stat(ruta); err == nil {
		return nil // ya está aquí: bajarla otra vez pisaría lo que hubiera sin fundir
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	datos, _, _, err := a.cliente().BajarCompartida(a.ctxCuenta(), token, dueno, ref, 0)
	if err != nil {
		return err
	}

	if err := a.conLaPersonal(func(mia *boveda.Boveda) error {
		p, err := boveda.AbrirCompartidaBytes("", datos, c.Titular, mia)
		if err != nil {
			return err
		}
		p.Cerrar()
		return nil
	}); err != nil {
		return err
	}

	if err := escritura.Atomica(ruta, escritura.Opciones{Permisos: 0o600, CrearCarpeta: true},
		func(w io.Writer) error {
			_, err := w.Write(datos)
			return err
		}); err != nil {
		return err
	}
	a.Actividad()
	return nil
}

// AbrirCompartida conmuta a una bóveda de otra persona.
//
// Lo que la diferencia de abrir un proyecto propio son dos cosas y las dos importan:
// se abre por **la ranura sellada hacia mi identidad** —que sale de mi bóveda
// personal, no de su clave— y la sincronización apunta a **la cuenta de la otra
// persona**, que es lo que `duenoActivo` lleva.
func (a *App) AbrirCompartida(dueno, ref string) error {
	c, hay := a.laCompartida(dueno, ref)
	if !hay {
		return errors.New("Esa bóveda compartida ya no está en tu lista")
	}
	ruta := rutaDeCompartida(dueno, ref)
	if ruta == "" {
		return errors.New("No encuentro dónde está esa bóveda en este sistema")
	}
	if _, err := os.Stat(ruta); err != nil {
		return errors.New("Esa bóveda no está en este equipo todavía")
	}

	// **Se abre con la personal**, que es de donde sale la identidad que abre mi
	// ranura. Y la personal tiene que estar abierta: aquí no vale la clave guardada,
	// porque lo que hace falta no es la clave sino poder descifrar con la identidad.
	var p *boveda.Boveda
	if err := a.conLaPersonal(func(mia *boveda.Boveda) error {
		var err error
		p, err = boveda.AbrirCompartida(ruta, c.Titular, mia)
		return err
	}); err != nil {
		return err
	}

	llave := a.llaveDeLaPrincipal()
	if len(llave) == 0 {
		return boveda.ErrCerrada
	}
	defer cripto.Borrar(llave)
	a.conmutarACompartida(p, dueno, ref, c.Nombre, llave)
	return nil
}

// laCompartida la busca en la bóveda personal, que es donde vive la lista.
func (a *App) laCompartida(dueno, ref string) (boveda.Compartida, bool) {
	var out boveda.Compartida
	hay := false
	_ = a.conLaPersonal(func(b *boveda.Boveda) error {
		out, hay = b.LaCompartida(dueno, ref)
		return nil
	})
	return out, hay
}

// conmutarACompartida es `conmutarA` diciendo de quién es, con el mismo orden: **el
// estado completo antes del aviso**, porque `cambiarBoveda` avisa a la ventana y la
// ventana contesta preguntando el estado.
func (a *App) conmutarACompartida(p *boveda.Boveda, dueno, ref, nombre string, llavePrincipal []byte) {
	a.alCerrarLaBoveda()
	a.ponerActivaDe(dueno, ref, nombre, llavePrincipal)
	a.cambiarBoveda(p)
	a.Actividad()
	a.buscarIconosSiProcede(a.ctx)
	a.alAbrirLaBoveda(p)
}

// VolverALaBovedaPersonal cierra el proyecto abierto y **deja la personal abierta**,
// sin volver a pedir la contraseña maestra.
//
// **Esto cambió el 2026-10-05, y el porqué importa**, porque antes hacía lo
// contrario. Lo preguntó el cliente: «¿qué diferencia hay entre "Cerrar la bóveda" y
// "Salir del proyecto"? ¿No vuelven los dos a mi bóveda?». Hacían **lo mismo** —las
// dos cerraban todo y pedían la maestra—, o sea dos botones con dos nombres para una
// sola cosa.
//
// Y la razón que había escrita aquí para pedir la maestra no se sostenía: decía que
// la personal no puede quedarse abierta porque **dos bóvedas abiertas a la vez** es
// lo que descartó la ADR 0050 — pero eso vale para tenerlas abiertas *a la vez*, no
// para abrir una **después** de cerrar la otra. Sigue habiendo una sola abierta.
//
// No abre ninguna puerta nueva: la clave de la personal está en memoria mientras hay
// un proyecto abierto —decisión escrita de la ADR 0050— y `conLaPersonal` ya abre
// este mismo fichero con ella cada vez que se lee o se escribe la lista de proyectos.
// Lo único que cambia es que salir de un proyecto deja de parecer que Esfinge se ha
// bloqueado solo. El reloj de inactividad sigue cerrándolo todo igual.
//
// **Si no se puede reabrir, se cierra todo**: dejar el proyecto abierto porque la
// personal falló sería quedarse en la bóveda de un cliente sin haberlo pedido.
func (a *App) VolverALaBovedaPersonal() error {
	if a.bovedaActiva() == "" {
		return nil
	}
	llave := a.llaveDeLaPrincipal()
	defer cripto.Borrar(llave)

	p, err := boveda.ReabrirConLaClave(rutaBovedaPrincipal(), llave)
	if err != nil {
		a.CerrarBoveda()
		return err
	}

	a.alCerrarLaBoveda()
	// **El estado completo antes del aviso**, que es la trampa de siempre:
	// `cambiarBoveda` avisa a la ventana y la ventana contesta preguntando el estado.
	a.ponerActiva("", "", nil)
	a.cambiarBoveda(p)
	a.Actividad()
	a.buscarIconosSiProcede(a.ctx)
	a.alAbrirLaBoveda(p)
	return nil
}

// RenombrarProyecto le cambia el nombre. **No toca su fichero**, que se llama por
// la referencia justo para esto.
func (a *App) RenombrarProyecto(ref, nombre string) error {
	if nombre == "" {
		return errors.New("Ponle un nombre al proyecto")
	}
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		p, hay := b.Proyecto(ref)
		if !hay {
			return errors.New("Ese proyecto no está en tu bóveda")
		}
		p.Nombre = nombre
		return b.PonerProyecto(p)
	}); err != nil {
		return err
	}
	a.Actividad()
	return nil
}

// LlevarAOtraBoveda mueve una entrada de la bóveda abierta a otra, o la copia.
// Con `aProyecto` vacío, el destino es la bóveda personal.
//
// **El orden importa y es el de las llaves de acceso** (ADR 0048): primero existe
// en el destino, se comprueba que está, y **solo entonces** desaparece del origen.
// Al revés, un fallo en medio pierde la entrada — y aquí lo que se pierde es una
// contraseña que a lo mejor no está en ningún otro sitio.
//
// Y lo que de verdad hace falta decir: **el secreto no cruza el puente en ningún
// momento**. La ventana manda dos identificadores y Go hace el viaje entero por
// dentro, que es mejor de lo que daría tener las dos bóvedas abiertas.
func (a *App) LlevarAOtraBoveda(id, aProyecto string, copiar bool) error {
	origen := a.boveda()
	if origen == nil {
		return boveda.ErrCerrada
	}
	if aProyecto == a.bovedaActiva() {
		return errors.New("Esa entrada ya está en esta bóveda")
	}
	e, hay := origen.Ver(id)
	if !hay {
		return errors.New("Esa entrada ya no está en la bóveda")
	}
	// **Con identificador nuevo**, que es lo que la convierte en una copia y no en
	// la misma entrada en dos sitios: con el mismo, la sincronización las trataría
	// como una sola y la fusión decidiría cuál gana (el argumento está escrito en
	// `envio.go` para una entrada que se manda a otra cuenta).
	e.ID = ""
	// Y sin lo que era de su bóveda de antes: la revisión y las fechas de
	// sincronización las pone la bóveda que la recibe.
	e.Revision = 0

	llave := a.llaveDeLaPrincipal()
	if len(llave) == 0 {
		return boveda.ErrCerrada
	}
	defer cripto.Borrar(llave)

	if err := a.conOtraBoveda(aProyecto, llave, func(d *boveda.Boveda) error {
		antes := d.Cuantas()
		if err := d.Poner(e); err != nil {
			return err
		}
		// **Se comprueba que está, y se comprueba contando la bóveda de destino**,
		// no dando por hecho que un `Poner` sin error significa que se guardó. El
		// guardado es un fichero, y un fichero puede fallar después de que el mapa
		// de memoria ya diga que sí.
		if d.Cuantas() != antes+1 {
			return errors.New("No se ha podido guardar la entrada en la otra bóveda")
		}
		return nil
	}); err != nil {
		return err
	}

	a.Actividad()
	if copiar {
		return nil
	}
	// A la papelera y no borrada del todo: si algo ha salido raro, está a un clic
	// durante treinta días (ADR 0026).
	return origen.Borrar(id)
}

// conOtraBoveda abre la bóveda de destino —la personal o un proyecto—, hace algo y
// la cierra. **Secuencial**: nunca hay dos abiertas de cara a quien mira.
func (a *App) conOtraBoveda(ref string, llavePrincipal []byte, hacer func(*boveda.Boveda) error) error {
	if ref == "" {
		return a.conLaPersonal(hacer)
	}
	a.muPersonal.Lock()
	defer a.muPersonal.Unlock()
	ruta := rutaDeProyecto(ref)
	if _, err := os.Stat(ruta); err != nil {
		return errors.New("Esa bóveda no está en este equipo todavía")
	}
	d, err := boveda.AbrirProyecto(ruta, llavePrincipal)
	if err != nil {
		return err
	}
	defer d.Cerrar()
	return hacer(d)
}

// conmutarA deja ese proyecto como la bóveda abierta.
//
// **Pasa por `alCerrarLaBoveda()` y no solo por `cambiarBoveda`**: lo primero
// vacía lo que quede por subir de la bóveda que se abandona, y sin eso se perdería
// su último guardado.
func (a *App) conmutarA(p *boveda.Boveda, ref, nombre string, llavePrincipal []byte) {
	a.alCerrarLaBoveda()
	// **Primero se apunta cuál es la activa y después se cambia**, porque
	// `cambiarBoveda` **avisa a la ventana** y la ventana contesta preguntando el
	// estado. Al revés, ese estado llega con la bóveda nueva y el nombre de nadie: la
	// barra de herramientas se queda sin decir en qué bóveda se trabaja hasta el
	// siguiente aviso, que puede no llegar nunca.
	a.ponerActiva(ref, nombre, llavePrincipal)
	a.cambiarBoveda(p)
	a.marcarProyectoUsado(ref)
	a.Actividad()
	a.buscarIconosSiProcede(a.ctx)
	// Y la sincronización apunta a **esta** bóveda: `nuevoSincronizador` le pregunta
	// a `bovedaActiva()`, que ya es la de ahora. Lo que no se publica desde aquí es
	// la identidad ni la posesión, que son de la personal — lo decide
	// `alAbrirLaBoveda`, no esta función.
	a.alAbrirLaBoveda(p)
}

// marcarProyectoUsado apunta en la personal cuándo se abrió, que es por lo que se
// ordena la lista.
//
// Se escribe **después** de haber conmutado, con la clave ya guardada, porque así
// vale igual viniendo de la personal que de otro proyecto. Y si falla no se dice:
// que la lista salga en otro orden no es motivo para que abrir un proyecto parezca
// haber fallado.
//
// Va a segundos, como todas las fechas del formato, y **no se reescribe si no ha
// cambiado**, que es la misma regla que la fecha `usada` de una llave de acceso: un
// guardado de más es una subida de más en cada apertura.
func (a *App) marcarProyectoUsado(ref string) {
	cuando := time.Now().UTC().Format(time.RFC3339)
	err := a.conLaPersonal(func(b *boveda.Boveda) error {
		p, hay := b.Proyecto(ref)
		if !hay || p.Usado == cuando {
			return nil
		}
		p.Usado = cuando
		return b.PonerProyecto(p)
	})
	if err != nil {
		log.Printf("esfinge: no se ha podido apuntar cuándo se abrió el proyecto: %v", err)
	}
}

// ------------------------------------------------------------------ al acabar

// EntregarProyecto escribe una copia independiente de esa bóveda, con la
// contraseña maestra que se le ponga, y devuelve **su clave de recuperación**
// (ADR 0051).
//
// Se enseña con la ceremonia de siempre y **no se puede volver a pedir**: quien
// recibe la bóveda la necesita tanto como su contraseña.
//
// **Se mira antes de preguntar dónde**, que es la regla que costó `ExportarLlaves`:
// con la bóveda cerrada o con un proyecto que no está en este equipo, lo que hay
// que decir es eso, no abrir un diálogo del sistema para un fichero que no se va a
// escribir.
func (a *App) EntregarProyecto(ref, maestraNueva string) (string, error) {
	if ref == "" {
		return "", errors.New("Esa no es una bóveda de proyecto")
	}
	// **El largo mínimo lo pone la pantalla**, como al crear la bóveda: vive en
	// `MINIMO_MAESTRA` de la interfaz y aquí no se repite, que un número en dos
	// sitios acaba siendo dos números. Go comprueba lo que de verdad no puede pasar.
	if strings.TrimSpace(maestraNueva) == "" {
		return "", errors.New("Ponle una contraseña a la bóveda que vas a entregar")
	}
	llave := a.llaveDeLaPrincipal()
	if len(llave) == 0 {
		return "", boveda.ErrCerrada
	}
	defer cripto.Borrar(llave)

	ruta := rutaDeProyecto(ref)
	if _, err := os.Stat(ruta); err != nil {
		return "", errors.New("Esa bóveda no está en este equipo todavía")
	}
	nombre := "proyecto"
	_ = a.conLaPersonal(func(b *boveda.Boveda) error {
		if p, hay := b.Proyecto(ref); hay && p.Nombre != "" {
			nombre = p.Nombre
		}
		return nil
	})

	p, err := boveda.AbrirProyecto(ruta, llave)
	if err != nil {
		return "", err
	}
	defer p.Cerrar()
	copia, recuperacion, err := p.Desprender(maestraNueva)
	if err != nil {
		return "", err
	}

	destino, err := a.sistema.ElegirDondeGuardar("Entregar la bóveda del proyecto",
		nombreDeFichero(nombre)+".esfinge", a.ajustes.CarpetaDeGuardar())
	if err != nil || destino == "" {
		return "", err
	}
	a.ajustes.RecordarCarpetaDeGuardar(filepath.Dir(destino))
	if err := copia.GuardarEn(destino); err != nil {
		return "", err
	}
	a.Actividad()
	return recuperacion, nil
}

// nombreDeFichero deja un nombre de proyecto en algo que se pueda escribir en
// cualquier sistema: sin barras, sin dos puntos y sin acentos raros de por medio.
func nombreDeFichero(nombre string) string {
	limpio := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < 32 {
			return '-'
		}
		return r
	}, strings.TrimSpace(nombre))
	if limpio == "" {
		return "proyecto"
	}
	return limpio
}

// ArchivarProyecto lo saca de la lista del día a día **y borra su fichero de este
// equipo**, dejando el del servidor.
//
// Esa es la mitad que lo hace útil: lo archivado deja de estar en el disco, así
// que «lo cerrado no está en memoria» pasa a ser también «no está aquí». Volver a
// traerlo es desarchivarlo, que lo baja.
func (a *App) ArchivarProyecto(ref string, archivar bool) error {
	if ref == "" {
		return errors.New("Esa no es una bóveda de proyecto")
	}
	if a.bovedaActiva() == ref {
		return errors.New("Cierra esa bóveda antes de archivarla")
	}
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		p, hay := b.Proyecto(ref)
		if !hay {
			return errors.New("Ese proyecto no está en tu bóveda")
		}
		p.Archivado = archivar
		return b.PonerProyecto(p)
	}); err != nil {
		return err
	}
	if archivar {
		// **El fichero y sus satélites**, que son copias de lo mismo: la base de la
		// sincronización es una bóveda entera y la caché de iconos es la lista de
		// sitios. Lo que no se borra es lo del servidor: desarchivar lo baja.
		if err := borrarElFicheroYSusSatelites(rutaDeProyecto(ref)); err != nil {
			return err
		}
	}
	a.Actividad()
	return nil
}

// BorrarProyecto se lleva la bóveda de este equipo, **del servidor** y de la lista.
//
// Pide la contraseña maestra, como borrar la bóveda personal y por la misma razón:
// es lo único irreversible que hay aquí, y lo que se lleva son las contraseñas de
// un cliente entero.
func (a *App) BorrarProyecto(ref, maestra string) error {
	if ref == "" {
		return errors.New("Esa no es una bóveda de proyecto")
	}
	if err := comprobarLaMaestra(rutaBovedaPrincipal(), maestra, "borrar una bóveda de proyecto"); err != nil {
		return err
	}
	// Si es la que está abierta, se cierra antes: dejarla en memoria después de
	// borrar el fichero es tener una bóveda sin fichero, y el siguiente guardado la
	// escribiría otra vez. Es lo mismo que hace `BorrarBoveda`.
	if a.bovedaActiva() == ref {
		if err := a.VolverALaBovedaPersonal(); err != nil {
			return err
		}
	}
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		return b.OlvidarProyecto(ref)
	}); err != nil {
		return err
	}
	if err := borrarElFicheroYSusSatelites(rutaDeProyecto(ref)); err != nil {
		return err
	}
	_ = olvidarLaBovedaDelRegistro(ref)
	// Y del servidor. **Que falle no deshace lo de aquí**, que ya está hecho: se
	// vuelve a intentar en la siguiente pasada de la sincronización, y mientras
	// tanto la bóveda no está en ningún equipo porque no está en la lista.
	if token, err := a.sesionDeCuenta(); err == nil {
		if err := a.cliente().OlvidarBoveda(a.ctxCuenta(), token, ref); err != nil {
			log.Printf("esfinge: la bóveda del proyecto sigue en el servidor: %v", err)
		}
	}
	a.Actividad()
	return nil
}

// borrarElFicheroYSusSatelites: el fichero y **todo lo que es una copia de lo que
// había dentro**. La lista sale de `BorrarBoveda`, que ya la tenía escrita, y no se
// reinventa aquí.
func borrarElFicheroYSusSatelites(ruta string) error {
	if ruta == "" {
		return errors.New("No encuentro esa bóveda en este sistema")
	}
	if err := os.Remove(ruta); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Lo de abajo es limpieza: que falte alguno no invalida el borrado, que ya está
	// hecho, y devolver un error aquí haría creer que no se ha borrado nada.
	_ = os.Remove(ruta + ".anterior")
	_ = os.Remove(boveda.RutaDeIconos(ruta))
	_ = (sincro.JuntoALaBoveda{Ruta: ruta}).Olvidar()
	_ = os.Remove(ruta + ".antes-de-fundir")
	escritura.LimpiarHuerfanos(filepath.Dir(ruta), 0)
	return nil
}
