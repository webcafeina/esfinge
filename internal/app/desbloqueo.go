package app

// Desbloquear la bóveda con el sistema: Touch ID o Windows Hello (fase C,
// `docs/desbloqueo-del-sistema.md`).
//
// Lo que hay que entender antes de tocar nada, porque manda sobre cómo está
// escrito todo lo de abajo:
//
// **Es un cerrojo y no una llave.** Sin firmar la aplicación, el sistema guarda
// el secreto sin poder atarlo a Esfinge. Protege de quien se sienta delante de tu
// ordenador desbloqueado; no protege de un programa que corra como tú, que es de
// lo que sí protege la contraseña maestra. Por eso:
//
//   - **la ranura del sistema nunca es la única**: la maestra y la clave de
//     recuperación siguen abriendo, y quitar el desbloqueo no puede dejar a nadie
//     fuera;
//   - **se dice en la pantalla donde se activa**, no solo en la documentación;
//   - y **no se activa solo**: hay que pedirlo, con la bóveda abierta.
//
// Y una regla que viene de más atrás: **esto no cuenta como actividad**. Leer el
// secreto es parte de abrir, y abrir ya toca el reloj por su cuenta.

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/escritura"
	"github.com/webcafeina/esfinge/internal/llavero"
)

// idEnElLlavero es con lo que se guarda el secreto en el sistema. **Uno por
// máquina y no uno por bóveda**: y sigue siéndolo desde que hay bóvedas de proyecto
// (ADR 0050), porque la ranura del sistema va **solo en la personal** y un proyecto
// se abre a través de ella. Si fuera una por bóveda, cada actualización pediría
// permiso del llavero una vez por proyecto.
//
// Si la personal se borra y se crea otra, la de antes ya no abre con ese secreto
// porque su sobre se fue con ella.
const idEnElLlavero = "com.webcafeina.esfinge.boveda"

// motivoDelDialogo es lo que el sistema enseña al pedir la huella. Lo lee alguien
// que en ese momento quiere entrar en sus contraseñas, así que dice eso.
const motivoDelDialogo = "abrir tu bóveda de Esfinge"

// EstadoDesbloqueo es lo que la ventana necesita saber para enseñar —o no— el
// botón de la huella.
type EstadoDesbloqueo struct {
	// Hay dice si este equipo tiene con qué. En Linux es siempre falso.
	Hay bool `json:"hay"`
	// Nombre es cómo lo llama el sistema: «Touch ID», «Windows Hello». Vacío si
	// no hay ninguno, y entonces la ventana no ofrece nada.
	Nombre string `json:"nombre"`
	// Puesto dice si **esta bóveda** tiene la ranura del sistema.
	Puesto bool `json:"puesto"`
	// Sugerir dice si hay que ofrecérselo: este equipo puede, **esta bóveda** no lo
	// lleva puesto, y a esta bóveda no se le ha ofrecido todavía.
	//
	// Lo decide Go y no la interfaz, y eso es el arreglo: mientras la pantalla lo
	// deducía de un sí/no de las preferencias, una bóveda distinta en el mismo
	// equipo se quedaba sin oferta. Aquí se compara **con qué bóveda es**.
	Sugerir bool `json:"sugerir"`
	// TrasActualizar avisa de que esta versión **todavía no tiene el permiso del
	// llavero**, así que la primera huella va a traer un diálogo del sistema
	// pidiendo la contraseña del equipo.
	//
	// Es la consecuencia de la decisión 3 de la ADR 0044, contestada en el Mac del
	// cliente: sin firmar, cada versión es un binario nuevo y el permiso se
	// renueva una vez por actualización. Sin avisar, eso se lee como que algo va
	// mal —justo en el programa donde eso más asusta—, así que la pantalla lo dice
	// antes de que salga.
	TrasActualizar bool `json:"trasActualizar"`
}

func (a *App) llaveroDelSistema() llavero.Llavero {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.llavero == nil {
		a.llavero = llavero.Del()
	}
	return a.llavero
}

// EstadoDelDesbloqueo dice si se puede ofrecer y si está puesto.
//
// **Mira el fichero, no la bóveda abierta**: es lo que se pregunta en la pantalla
// de desbloquear, cuando todavía no hay nada abierto.
func (a *App) EstadoDelDesbloqueo() EstadoDesbloqueo {
	l := a.llaveroDelSistema()
	ruta := rutaBovedaPrincipal()
	puesto := boveda.RanuraDelSistemaEn(ruta)
	// **Cuál es la bóveda personal de este equipo, esté abierta o cerrada.** Con la
	// bóveda abierta se pregunta a ella; con la pantalla de desbloquear delante hay
	// que mirar el fichero, donde el identificador va en claro.
	//
	// Y desde la ADR 0050: **solo se le pregunta a la bóveda abierta si la abierta es
	// la personal**. Con un proyecto delante contestaría su identificador, al que
	// nunca se le ha ofrecido nada, y la pantalla ofrecería Touch ID **para el
	// proyecto** — que es la forma exacta del fallo que arregló la ADR 0044, con otra
	// cara.
	id := ""
	if b := a.boveda(); b != nil && a.bovedaActiva() == "" {
		id = b.ID()
	} else {
		id = boveda.IDEn(ruta)
	}
	return EstadoDesbloqueo{
		Hay:    l.Hay(),
		Nombre: l.Nombre(),
		Puesto: puesto,
		// **Y no se ofrece con un proyecto abierto**, aunque lo que se ofrezca sea la
		// personal y el identificador salga bien. Son dos cosas: la tarjeta dice «esta
		// bóveda» mirando a la pantalla de un proyecto, que es otra; y sobre todo
		// **activar con un proyecto abierto está prohibido**, así que sería ofrecer algo
		// que al pulsarlo da un error. Es la regla de `ExportarLlaves`: lo que hace
		// falta para decidir se mira antes de pedirle a alguien que decida. No se pierde
		// la oferta —contestar es lo único que la apunta—, se mueve a la pantalla donde
		// se puede aceptar. Se vio en una captura, con la personal todavía sin contestar:
		// el caso que la prueba de la ADR 0050 no cubría, porque partía de contestada.
		Sugerir: l.Hay() && !puesto && id != "" && a.bovedaActiva() == "" &&
			a.ajustes.Ver().DesbloqueoSugeridoPara != id,
		// **Solo cuando hay algo que avisar**: si no está puesto, no va a salir
		// ninguna huella y no hay diálogo del que hablar.
		TrasActualizar: puesto && a.ajustes.Ver().VersionConPermisoDelLlavero != a.version,
	}
}

// ActivarDesbloqueo pone la ranura del sistema en **la bóveda personal**. Exige
// tenerla abierta: es lo que impide que alguien la active sin saber la maestra.
//
// El orden importa y no es el evidente: **primero se le da a guardar al sistema
// y después se pone la ranura**. Al revés, un fallo del sistema dejaría una
// ranura que no abre nadie y un botón que promete algo que no funciona.
//
// **Y con un proyecto abierto no se activa** (ADR 0050). La ranura del sistema va
// solo en la personal, y eso no es una limitación: es lo que hace que haya **una**
// entrada en el llavero y **un** diálogo del sistema tras cada actualización en vez
// de uno por bóveda. Un proyecto se abre a través de la personal, así que la huella
// ya lo abre. Puesta en un proyecto además no serviría de mucho: esa ranura **no se
// sube**, así que sería de este equipo y de nadie más.
func (a *App) ActivarDesbloqueo() error {
	if a.bovedaActiva() != "" {
		return errors.New("Vuelve a tu bóveda para activar el desbloqueo con el sistema")
	}
	b := a.boveda()
	if b == nil {
		return boveda.ErrCerrada
	}
	l := a.llaveroDelSistema()
	if !l.Hay() {
		return llavero.ErrNoHay
	}
	secreto, err := boveda.SecretoDelSistema()
	if err != nil {
		return err
	}
	if err := l.Guardar(idEnElLlavero, secreto); err != nil {
		return err
	}
	if err := b.PonerRanuraDelSistema(secreto); err != nil {
		// Se deshace lo de fuera: un secreto guardado que no abre nada no hace
		// daño, pero tampoco tiene por qué quedarse.
		_ = l.Borrar(idEnElLlavero)
		return err
	}
	a.noVolverAOfrecerA(b.ID())
	a.Actividad()
	return nil
}

// QuitarDesbloqueo deja la bóveda sin él, y borra lo que guardaba el sistema.
//
// **Se quita la ranura aunque el sistema falle al borrar.** Sin sobre, ese
// secreto ya no abre nada: lo que importa es que la puerta se cierre, y lo que
// quede suelto en el llavero es un puñado de bytes inservibles.
func (a *App) QuitarDesbloqueo() error {
	b := a.boveda()
	if b == nil {
		return boveda.ErrCerrada
	}
	if err := b.QuitarRanuraDelSistema(); err != nil {
		return err
	}
	_ = a.llaveroDelSistema().Borrar(idEnElLlavero)
	a.Actividad()
	return nil
}

// AbrirBovedaConElSistema pide la huella y abre.
//
// Lo que devuelve cuando no puede está escrito para que la ventana no tenga que
// decidir nada: cancelar no es un fallo y se vuelve a la contraseña maestra.
func (a *App) AbrirBovedaConElSistema() error {
	ruta := rutaBovedaPrincipal()
	if !boveda.RanuraDelSistemaEn(ruta) {
		return boveda.ErrSinRanuraDelSistema
	}
	secreto, err := a.llaveroDelSistema().Leer(idEnElLlavero, motivoDelDialogo)
	if err != nil {
		return err
	}
	// De paso, lo mismo que hace abrir con la maestra: los temporales que dejó
	// una interrupción anterior, si llevan más de un día.
	escritura.LimpiarHuerfanos(filepath.Dir(ruta), 24*time.Hour)

	b, err := boveda.AbrirConElSistema(ruta, secreto)
	if errors.Is(err, boveda.ErrSinRanuraDelSistema) {
		// El secreto ya no abre: la bóveda se restauró de una copia, o se quitó el
		// desbloqueo en otro equipo y esa copia llegó aquí. Se limpia lo que quedó
		// suelto para no volver a ofrecerlo, y se pide la maestra.
		_ = a.llaveroDelSistema().Borrar(idEnElLlavero)
		return err
	}
	if err != nil {
		return err
	}
	// **Ha abierto, así que esta versión ya tiene el permiso del llavero.** Se
	// anota aquí y no al arrancar: lo que hay que saber no es qué versión corre,
	// es cuál ha conseguido que el sistema le diera la llave sin volver a
	// preguntar. Hasta que eso pasa, la pantalla sigue avisando.
	a.anotarPermisoDelLlavero()

	a.cambiarBoveda(b)
	a.Actividad()
	a.buscarIconosSiProcede(a.ctx)
	a.alAbrirLaBoveda(b)
	return nil
}

// NoOfrecerElDesbloqueo apunta que a **esta** bóveda ya se le ofreció, y por eso
// recibe la bóveda abierta y no un sí/no.
//
// Se llama al contestar: tanto al activarlo como al decir «ahora no». Marcar la
// casilla de la pantalla de desbloquear **no** cuenta, que eso es pedirlo, no
// contestarlo.
func (a *App) NoOfrecerElDesbloqueo() error {
	b := a.boveda()
	if b == nil {
		return boveda.ErrCerrada
	}
	a.noVolverAOfrecerA(b.ID())
	return nil
}

func (a *App) noVolverAOfrecerA(id string) {
	if id == "" {
		return
	}
	p := a.ajustes.Ver()
	if p.DesbloqueoSugeridoPara == id {
		return
	}
	p.DesbloqueoSugeridoPara = id
	_ = a.ajustes.Guardar(p)
}

func (a *App) anotarPermisoDelLlavero() {
	p := a.ajustes.Ver()
	if p.VersionConPermisoDelLlavero == a.version {
		return
	}
	p.VersionConPermisoDelLlavero = a.version
	_ = a.ajustes.Guardar(p)
}
