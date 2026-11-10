package app

// Dar acceso a una bóveda de proyecto, desde la ventana (ADR 0052).
//
// Es la otra cosa que se puede hacer con un proyecto cuando entra otra persona, y
// **no sustituye a entregar una copia** (ADR 0051): entregar sigue al lado, para
// cuando el proyecto de verdad se acaba y lo que se quiere es soltarlo.
//
// Cuatro reglas que esto respeta y que no se ven leyendo el código de una en una:
//
//   - **Se da acceso desde dentro del proyecto**, con él abierto. No es una
//     formalidad: poner la ranura cambia **su** fichero, y lo que lo sube es la
//     sincronización de la bóveda abierta. Haciéndolo desde la lista, la ranura se
//     quedaría aquí y quien recibe el acceso se bajaría una bóveda que no puede
//     abrir hasta que alguien entrara en ese proyecto.
//   - **Lo de la cuenta es de la bóveda personal, siempre**: la identidad con la que
//     se sella el sobre y el buzón salen de ella por `conLaPersonal`, esté abierta la
//     que esté. Es la regla que se saltaba `compartir.go` hasta la 2.39.3.
//   - **La huella se enseña antes.** Es lo único que protege del servidor en el
//     primer envío, y una huella que nadie mira no protege nada (ADR 0043).
//   - **El permiso lo hace cumplir el servidor, no esta capa.** Lo que se guarda aquí
//     es informativo, para no pedirle a nadie que teclee algo que va a acabar en un
//     403 — la regla que costó `ExportarLlaves`.

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/cuenta"
)

// QuienTieneAcceso es una persona con acceso a esta bóveda, como se enseña.
type QuienTieneAcceso struct {
	Titular string `json:"titular"`
	Correo  string `json:"correo"`
	Permiso string `json:"permiso"`
	Desde   string `json:"desde"`
	// EnElServidor dice si el servidor también le deja entrar.
	//
	// **Son dos listas y no hay forma de que sean una**: el servidor no puede leer la
	// bóveda —así que no sabe correos— y la pantalla no puede depender de él para
	// decir nombres. Cuando no coinciden se enseña en vez de disimularlo, que es lo
	// que permite arreglarlo en vez de descubrirlo el día que alguien no entra.
	EnElServidor bool `json:"enElServidor"`
}

// Compartidas son las bóvedas de otras personas a las que tengo acceso, con lo que
// la lista necesita saber de cada una **sin abrirla**.
type CompartidaEnLaLista struct {
	Dueno  string `json:"dueno"`
	Ref    string `json:"ref"`
	Nombre string `json:"nombre"`
	// Permiso es lo que me dijeron. **Informativo**: el que manda es el del servidor.
	Permiso string `json:"permiso"`
	Huella  string `json:"huella"`
	Usado   string `json:"usado"`
	// EnEsteEquipo dice si el fichero ya está aquí. Si no, la lista ofrece traerlo
	// en vez de fallar al abrirlo, que es lo mismo que hacen los proyectos dormidos.
	EnEsteEquipo bool `json:"enEsteEquipo"`
	// Retirada es cuándo el dueño quitó el acceso (ADR 0053). Con fecha, esta fila ya
	// **no es una bóveda**: es el aviso de una que se fue, y lo único que se puede
	// hacer con ella es quitarla de la lista.
	Retirada string `json:"retirada"`
}

// Compartidas lista lo que me han compartido, lo último usado primero.
func (a *App) Compartidas() ([]CompartidaEnLaLista, error) {
	out := []CompartidaEnLaLista{}
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		for _, c := range b.Compartidas() {
			x := CompartidaEnLaLista{
				Dueno: c.Dueno, Ref: c.Ref, Nombre: c.Nombre,
				Permiso: c.Permiso, Huella: c.Huella, Usado: c.Usado,
				Retirada: c.Retirada,
			}
			if r := rutaDeCompartida(c.Dueno, c.Ref); r != "" {
				if _, err := os.Stat(r); err == nil {
					x.EnEsteEquipo = true
				}
			}
			out = append(out, x)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

// DejarDeVerCompartida la saca de mi lista y **borra el fichero de este equipo**.
//
// No toca nada de la otra persona: la bóveda es suya y sigue donde estaba. Lo que
// esto hace es lo que puede hacer quien recibe —irse— y por eso no pide nada a nadie.
func (a *App) DejarDeVerCompartida(dueno, ref string) error {
	if a.bovedaActiva() == ref && a.duenoDeLaActiva() == dueno {
		if err := a.VolverALaBovedaPersonal(); err != nil {
			return err
		}
	}
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		return b.OlvidarCompartida(dueno, ref)
	}); err != nil {
		return err
	}
	if r := rutaDeCompartida(dueno, ref); r != "" {
		borrarElFicheroYSusSatelites(r)
	}
	a.Actividad()
	return nil
}

// retirarLaCompartida tacha la fila y **borra el fichero de este equipo**, cuando el
// servidor ha dicho que el dueño quitó el acceso (ADR 0053).
//
// Es `DejarDeVerCompartida` con dos diferencias, y las dos son el porqué de que no sea
// la misma función:
//
//   - **La fila se tacha en vez de quitarse.** Irse es mío y no hace falta recordarlo;
//     que me echen hay que poder contarlo, y el nombre no puede guardarse fuera de la
//     bóveda. De eso vive el aviso.
//   - **Nadie lo ha pedido aquí**, así que si algo falla no hay a quién decírselo: se
//     registra y se vuelve a intentar en la pasada siguiente, que llegará con el mismo
//     403. Lo que no se hace es quedarse a medias en silencio.
//
// Y el orden es el de siempre en este proyecto: **primero salir, luego la fila, y el
// fichero al final**. Al revés, un fallo en medio deja la bóveda abierta encima de un
// fichero que ya no está.
func (a *App) retirarLaCompartida(dueno, ref string) {
	if dueno == "" || ref == "" {
		return
	}
	if a.bovedaActiva() == ref && a.duenoDeLaActiva() == dueno {
		if err := a.VolverALaBovedaPersonal(); err != nil {
			log.Printf("esfinge: no se ha podido salir de la bóveda retirada: %v", err)
			return
		}
	}
	cuando := time.Now().UTC().Format(time.RFC3339)
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		return b.RetirarCompartida(dueno, ref, cuando)
	}); err != nil {
		log.Printf("esfinge: no se ha podido tachar la bóveda retirada: %v", err)
		return
	}
	if r := rutaDeCompartida(dueno, ref); r != "" {
		if err := borrarElFicheroYSusSatelites(r); err != nil {
			log.Printf("esfinge: no se ha podido borrar el fichero de la bóveda retirada: %v", err)
		}
	}
	// **El estado completo antes del aviso**, que es la trampa de siempre: la ventana
	// contesta a esto pidiendo la lista, y pedirla antes de tachar la traería entera.
	a.sistema.Avisar(EventoBovedaCambiada, nil)
}

// DarAcceso le da acceso a esa dirección **sobre la bóveda de proyecto abierta**.
//
// El orden importa y no es el evidente: **la ranura, el servidor, y el sobre al
// final**. Si el miembro entrara en el servidor antes que la ranura, esa persona
// podría bajarse una bóveda que todavía no puede abrir; y el sobre va el último
// porque es el único paso que **no se puede deshacer** — una vez en su buzón, está.
// Si algo fallara antes, lo que queda es una ranura de más que no abre nada sin él.
func (a *App) DarAcceso(correo, permiso string) error {
	if permiso != "ver" && permiso != "editar" {
		return errors.New("Ese permiso no existe")
	}
	ref := a.bovedaActiva()
	if ref == "" || a.duenoDeLaActiva() != "" {
		return errors.New("Entra en el proyecto que quieras compartir")
	}
	p := a.boveda()
	if p == nil {
		return boveda.ErrCerrada
	}
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return err
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	mia, err := a.miCuenta()
	if err != nil {
		return err
	}

	// A quién: sus llaves públicas, que son las que sellan su ranura.
	l, err := a.cliente().LlavesDe(a.ctxCuenta(), token, c)
	if err != nil {
		return err
	}
	suya := boveda.Identidad{Suite: l.Suite, Cifrado: l.Cifrado, Firma: l.Firma}
	huella := boveda.HuellaDeIdentidad(l.Suite, l.Cifrado, l.Firma)

	// **El titular lo elige quien da el acceso**, no quien lo recibe: es lo que
	// después sirve para quitárselo, y el servidor nunca dice de quién es una cuenta.
	crudo, err := cripto.Azar(8)
	if err != nil {
		return err
	}
	titular := hex.EncodeToString(crudo)

	// 1 · La ranura y el nombre, dentro del fichero del proyecto. Guardar despierta a
	// la sincronización, que es lo que lo sube.
	if err := p.PonerAcceso(titular, suya); err != nil {
		return err
	}
	if err := p.PonerTitular(boveda.Titular{ID: titular, Correo: c, Permiso: permiso, Huella: huella}); err != nil {
		return err
	}

	// 2 · El miembro, en el servidor.
	if err := a.cliente().DarAcceso(a.ctxCuenta(), token, ref, c, permiso, titular); err != nil {
		return err
	}

	// 3 · El sobre, **sellado con la identidad de la personal** y no con la de este
	// proyecto, que no tiene ni debe tener.
	nombre := a.nombreDeLaActiva()
	var sobre boveda.Envio
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		var err error
		// **Y la dirección propia, tal cual, para que al otro lado se sepa de quién
		// viene.** No prueba nada —la escribe quien manda— y por eso la pantalla la da
		// como lo que es. Lo que prueba sigue siendo la huella.
		sobre, err = b.MandarAcceso(boveda.Acceso{
			Dueno: mia, Ref: ref, Nombre: nombre, Titular: titular, Permiso: permiso,
			De: leerDatosCuenta().Correo,
		}, suya)
		return err
	}); err != nil {
		return err
	}
	a.Actividad()
	return a.cliente().Mandar(a.ctxCuenta(), token, c, sobre)
}

// CambiarPermiso le sube o le baja el permiso a quien ya tiene acceso (ADR 0058).
//
// **Se cambia en el sitio, y eso es toda la decisión.** Los dos caminos que parecen
// equivalentes no lo son, y conviene que quede escrito para que nadie los reintente:
//
//   - **Volver a llamar a `DarAcceso`** acuña un titular nuevo (`cripto.Azar(8)`), así
//     que dejaría una **ranura huérfana** en la bóveda y el titular guardado dejaría de
//     ser el que conoce el servidor — y entonces «quitar» apuntaría al que no es.
//   - **Quitar y volver a dar** es correcto y caro: quitar hace que esa persona **borre
//     su copia** (ADR 0053), así que subirle el permiso le costaría volver a bajarse la
//     bóveda entera. En un servidor, en medio de un despliegue, eso es el despliegue
//     caído.
//
// Lo que lo hace barato ya estaba hecho: la tabla del servidor es un `upsert` por
// `(ref, cuenta)` y `PonerTitular` actualiza por identificador. Así que **ni rota la
// clave, ni añade ranura, ni obliga a nadie a volver a bajar nada**.
//
// Y no manda ningún sobre nuevo: el permiso que esa persona tiene guardado se pone al
// día **por la sincronización**, porque desde la 0058 el servidor lo dice al bajar (ver
// `apuntarElPermiso`). Mandando un sobre habría que aceptarlo, y en un servidor eso es
// un `recibir` a mano que nadie va a correr.
func (a *App) CambiarPermiso(titular, permiso string) error {
	if permiso != "ver" && permiso != "editar" {
		return errors.New("Ese permiso no existe")
	}
	ref := a.bovedaActiva()
	if ref == "" || a.duenoDeLaActiva() != "" {
		return errors.New("Entra en el proyecto para cambiar quién tiene acceso")
	}
	p := a.boveda()
	if p == nil {
		return boveda.ErrCerrada
	}
	// **El correo y la huella salen de lo guardado**, no de quien llama: la ventana
	// manda un identificador de ranura y nada más, así que no puede apuntar un permiso
	// a una dirección que no sea la que ya tenía acceso.
	var quien boveda.Titular
	for _, t := range p.Titulares() {
		if t.ID == titular {
			quien = t
		}
	}
	if quien.ID == "" {
		return errors.New("Esa persona no tiene acceso a esta bóveda")
	}
	if quien.Permiso == permiso {
		return nil
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}

	// **El servidor primero**, que es quien lo impone: si lo de dentro se guardara antes
	// y esto fallara, la pantalla diría «puede editar» y el servidor seguiría diciendo
	// que no. Al revés, lo peor que queda es la pantalla diciendo lo de antes de algo
	// que ya puede — y la pasada siguiente lo cuadra.
	if err := a.cliente().DarAcceso(a.ctxCuenta(), token, ref, quien.Correo, permiso, titular); err != nil {
		return err
	}
	quien.Permiso = permiso
	a.Actividad()
	return p.PonerTitular(quien)
}

// ------------------------------------------------- el servidor de esta casa

// ConfirmarHuellaDelServidor apunta que alguien ha comparado la huella de ese
// servidor por otro camino (ADR 0058, sobre la 0043).
//
// **Recibe la huella que la pantalla enseñó**, no la que haya ahora, y la comprueba
// contra el servidor antes de darla por buena. Pidiéndola otra vez aquí se guardaría
// lo que contestara en este segundo, que puede no ser lo que la persona miró — es la
// misma razón por la que la tarjeta de un agente lleva referencia.
func (a *App) ConfirmarHuellaDelServidor(huella string) error {
	correo := a.ajustes.Ver().CorreoDelServidor
	if strings.TrimSpace(correo) == "" {
		return errors.New("Primero pon la cuenta de Esfinge de tu servidor")
	}
	ahora, err := a.HuellaDe(correo)
	if err != nil {
		return err
	}
	if ahora.Huella != huella {
		return errors.New(
			"La huella ha cambiado mientras la mirabas; vuelve a pedirla y compárala otra vez")
	}
	p := a.ajustes.Ver()
	p.HuellaDelServidor = huella
	// Por la puerta de los ajustes y no escribiendo el campo a mano, para que pase por
	// el mismo sitio que todo lo demás.
	return a.ajustes.PonerHuellaDelServidor(huella, p.CorreoDelServidor)
}

// DarAccesoAlServidor es lo que hace el interruptor de cada proyecto: **acceso de solo
// ver a la cuenta del servidor de esta casa** (ADR 0057), sin ningún mecanismo nuevo.
//
// Lo que añade sobre llamar a `DarAcceso` con ese correo es la única cosa que faltaba,
// y faltaba de verdad: **comprobar la huella**. Hasta la 0058 el interruptor sellaba la
// clave de la bóveda hacia las llaves públicas que el servidor de cuentas diera para
// esa dirección, **sin que nadie las hubiera comparado nunca** — justo lo que la ADR
// 0043 dice que es lo único que protege en el primer envío. Con una persona la pantalla
// te obliga a mirar la huella; el interruptor se lo saltaba.
//
// **Y se compara en cada proyecto, aunque solo se confirme una vez.** Confirmar una vez
// es lo que se eligió para que marcar el quinto proyecto sea un clic, pero si la
// comparación no se repitiera al sellar, el servidor de cuentas podría publicar otras
// llaves después y solo el primer proyecto estaría protegido. Confirmar es de la
// persona; comparar es de cada vez.
func (a *App) DarAccesoAlServidor() error {
	p := a.ajustes.Ver()
	correo := strings.TrimSpace(p.CorreoDelServidor)
	if correo == "" {
		return errors.New("No hay ningún servidor puesto en Ajustes")
	}
	if p.HuellaDelServidor == "" {
		return errors.New(
			"Antes compara la huella de tu servidor en Ajustes: es lo único que prueba que las contraseñas van a tu equipo")
	}
	ahora, err := a.HuellaDe(correo)
	if err != nil {
		return err
	}
	if ahora.Huella != p.HuellaDelServidor {
		// **No se da el acceso y no se guarda la huella nueva.** Puede ser que el
		// servidor haya vuelto a instalarse, y puede ser otra cosa; las dos se arreglan
		// igual —compararla otra vez por otro camino— y decidirlo no es de aquí.
		return fmt.Errorf(
			"La huella de %s no es la que confirmaste: ahora es %s y tenías apuntada %s. "+
				"No le doy acceso hasta que la compares otra vez en Ajustes",
			correo, ahora.Huella, p.HuellaDelServidor)
	}
	return a.DarAcceso(correo, "ver")
}

// RotarLaClave vuelve a cifrar esta bóveda de proyecto con claves nuevas y la deja
// abierta solo para quien tiene acceso ahora mismo (ADR 0059).
//
// **Necesita red**, y por eso esto vive aquí y no en `internal/boveda`: las identidades
// de quien tiene acceso no están dentro de la bóveda, y hay que resellar la ranura de cada
// uno hacia la suya. La huella se compara contra la que se comparó al dar el acceso, y eso
// lo hace la bóveda, que es quien las tiene apuntadas.
func (a *App) RotarLaClave() error {
	ref := a.bovedaActiva()
	if ref == "" || a.duenoDeLaActiva() != "" {
		return errors.New("Entra en el proyecto para cambiar su clave")
	}
	p := a.boveda()
	if p == nil {
		return boveda.ErrCerrada
	}
	if err := a.rotarLaClaveDe(p); err != nil {
		return err
	}
	a.Actividad()
	return nil
}

// rotarLaClaveDe pide al servidor las llaves de quien tiene acceso y rota.
func (a *App) rotarLaClaveDe(p *boveda.Boveda) error {
	vivos := map[string]bool{}
	for _, id := range p.Accesos() {
		vivos[id] = true
	}
	para := map[string]boveda.Identidad{}
	if len(vivos) > 0 {
		token, err := a.sesionDeCuenta()
		if err != nil {
			return err
		}
		for _, t := range p.Titulares() {
			if !vivos[t.ID] {
				continue
			}
			l, err := a.cliente().LlavesDe(a.ctxCuenta(), token, t.Correo)
			if err != nil {
				// **Con el nombre de quién**, que es lo que hace falta para arreglarlo:
				// «no he podido pedir unas llaves» no dice de quién ni a quién preguntar.
				return fmt.Errorf("No he podido pedir las llaves de %s, así que no cambio la clave: %w", t.Correo, err)
			}
			para[t.ID] = boveda.Identidad{Suite: l.Suite, Cifrado: l.Cifrado, Firma: l.Firma}
		}
	}
	return p.RotarLasClaves(para, boveda.PorLaMano)
}

// QuitarAcceso se lo quita a ese titular: **la puerta del servidor primero**, que es
// la que corta de verdad y en el acto, después la ranura, y al final **la clave de la
// bóveda, que se rota** (ADR 0059).
//
// **Ese orden importa.** Quitar el acceso es lo urgente y lo que corta en el acto; rotar
// es lo que hace que no se pueda volver a entrar con lo que esa persona se llevó, y
// necesita red y las llaves de los que quedan. Si rotar falla —el servidor no contesta, la
// huella de alguien ha cambiado—, **el acceso sigue quitado** y se dice que la clave no se
// ha cambiado: deshacer la revocación porque no se ha podido rotar sería lo contrario de
// lo que ha pedido quien pulsa.
//
// Lo que esto **no** hace, y la pantalla lo dice sin disimularlo: no le quita de la cabeza
// lo que vio mientras tuvo acceso. Lo que sí llega a su disco es el borrado de su copia
// (ADR 0053), que es cooperativo: lo hace su Esfinge cuando el servidor se lo dice.
func (a *App) QuitarAcceso(titular string) error {
	ref := a.bovedaActiva()
	if ref == "" || a.duenoDeLaActiva() != "" {
		return errors.New("Entra en el proyecto para cambiar quién tiene acceso")
	}
	p := a.boveda()
	if p == nil {
		return boveda.ErrCerrada
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	if err := a.cliente().QuitarAcceso(a.ctxCuenta(), token, ref, titular); err != nil {
		return err
	}
	if err := p.RetirarAcceso(titular); err != nil {
		return err
	}
	a.Actividad()
	if err := p.OlvidarTitular(titular); err != nil {
		return err
	}
	if err := a.rotarLaClaveDe(p); err != nil {
		return fmt.Errorf("Le he quitado el acceso, pero no he podido cambiar la clave de la bóveda: %w", err)
	}
	return nil
}

// apuntarElPermiso guarda lo que el servidor acaba de decir que puedo hacer en una
// bóveda ajena (ADR 0058), y pone al día la ventana si era la abierta.
//
// Hasta aquí el permiso solo llegaba **dentro del sobre que se acepta en el buzón**, o
// sea una vez y para siempre. Cambiárselo a alguien no viajaba: el servidor empezaba a
// aceptarle subidas y su Esfinge seguía creyendo que solo podía ver, así que no le
// ofrecía editar. Y al revés es peor de leer —cree que puede, y al subir se encuentra
// un 403—, porque entonces lo escrito se queda en su equipo y en ninguna otra parte.
//
// Dos cosas de cómo está escrito:
//
//   - **Solo escribe si ha cambiado.** Guardar la bóveda personal despierta su
//     sincronización, así que apuntarlo en cada pasada sería subirla cada minuto para
//     decir lo mismo. Es la regla de la fecha de uso de una llave de acceso.
//   - **El estado completo antes del aviso.** La ventana contesta a `EventoEstadoBoveda`
//     preguntando el estado, así que avisar antes de tener el permiso puesto la deja con
//     el de antes hasta el aviso siguiente, que puede no llegar.
func (a *App) apuntarElPermiso(dueno, ref, permiso string) {
	if dueno == "" || ref == "" || permiso == "" {
		return
	}
	cambio := false

	comp, hay := a.laCompartida(dueno, ref)
	if hay && comp.Permiso != permiso {
		comp.Permiso = permiso
		if err := a.conLaPersonal(func(b *boveda.Boveda) error {
			return b.PonerCompartida(comp)
		}); err != nil {
			log.Printf("esfinge: no he podido apuntar el permiso nuevo de «%s»: %v", comp.Nombre, err)
		} else {
			cambio = true
		}
	}

	// Y lo que la ventana tiene en la mano, que se fijó al abrirla: sin esto, a quien
	// esté dentro de la bóveda no le cambia nada hasta que salga y vuelva a entrar.
	a.mu.Lock()
	if a.duenoActivo == dueno && a.activa == ref && a.permisoActivo != permiso {
		a.permisoActivo = permiso
		cambio = true
	}
	a.mu.Unlock()

	if cambio {
		a.sistema.Avisar(EventoEstadoBoveda, a.EstadoBoveda())
	}
}

// QuienTiene lista a las personas con acceso a la bóveda abierta, **juntando las dos
// listas**: la de la bóveda, que es la que tiene los correos, y la del servidor, que
// es la que de verdad deja entrar.
//
// La ven todos los que tienen acceso, no solo el dueño: lo eligió el cliente, y es lo
// coherente con que quien puede editar pueda dar acceso a más gente — si no, entraría
// gente y nadie se enteraría.
func (a *App) QuienTiene() ([]QuienTieneAcceso, error) {
	p := a.boveda()
	if p == nil {
		return nil, boveda.ErrCerrada
	}
	ref := a.bovedaActiva()
	if ref == "" {
		return nil, errors.New("Tu bóveda personal no se comparte")
	}
	enElServidor := map[string]bool{}
	if token, err := a.sesionDeCuenta(); err == nil && a.duenoDeLaActiva() == "" {
		// Solo el dueño puede preguntarle al servidor por los miembros; quien tiene
		// acceso ve la lista de la bóveda, que es la que lleva los nombres.
		if ms, err := a.cliente().Miembros(a.ctxCuenta(), token, ref); err == nil {
			for _, m := range ms {
				enElServidor[m.Titular] = true
			}
		}
	}
	soyElDueno := a.duenoDeLaActiva() == ""
	out := []QuienTieneAcceso{}
	for _, t := range p.Titulares() {
		out = append(out, QuienTieneAcceso{
			Titular: t.ID, Correo: t.Correo, Permiso: t.Permiso, Desde: t.Desde,
			// Para quien no es el dueño no hay nada que comparar, así que no se
			// enseña una diferencia que no puede comprobar.
			EnElServidor: !soyElDueno || enElServidor[t.ID],
		})
	}
	return out, nil
}

// AceptarAcceso mete en mi lista una bóveda que me han compartido y **se la baja**.
//
// Lo que llega al buzón no entra solo, igual que una copia: si entrara, cualquiera
// que supiera mi correo me metería un proyecto en la lista.
func (a *App) AceptarAcceso(id string) error {
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	brutos, err := a.cliente().Buzon(a.ctxCuenta(), token)
	if err != nil {
		return err
	}
	for _, x := range brutos {
		if x.ID != id {
			continue
		}
		acceso, de, err := a.abrirAccesoDelBuzon(x.Sobre)
		if err != nil {
			return err
		}
		if err := a.conLaPersonal(func(b *boveda.Boveda) error {
			return b.PonerCompartida(boveda.Compartida{
				Dueno: acceso.Dueno, Ref: acceso.Ref, Nombre: acceso.Nombre,
				Titular: acceso.Titular, Permiso: acceso.Permiso, Huella: de.Huella,
			})
		}); err != nil {
			return err
		}
		if err := a.cliente().TirarDelBuzon(a.ctxCuenta(), token, id); err != nil {
			return err
		}
		// Y se trae el fichero. **El acceso se apunta antes de tirar el sobre**: si se
		// tirara primero y fallara lo de arriba, no habría forma de volver a saber
		// dónde está esa bóveda. Que la descarga falle se puede reintentar.
		if err := a.BajarCompartida(acceso.Dueno, acceso.Ref); err != nil {
			return fmt.Errorf("El acceso está guardado, pero la bóveda no se ha podido traer: %w", err)
		}
		return nil
	}
	return errors.New("Ese envío ya no está en el buzón")
}

// abrirAccesoDelBuzon abre un sobre de acceso **con la identidad de la personal**.
func (a *App) abrirAccesoDelBuzon(crudo json.RawMessage) (boveda.Acceso, boveda.Identidad, error) {
	var s boveda.Envio
	if err := json.Unmarshal(crudo, &s); err != nil {
		return boveda.Acceso{}, boveda.Identidad{}, errors.New("Este envío no se entiende")
	}
	var acc boveda.Acceso
	var de boveda.Identidad
	err := a.conLaPersonal(func(b *boveda.Boveda) error {
		var err error
		acc, de, err = b.AbrirAcceso(s)
		return err
	})
	return acc, de, err
}

// miCuenta es el identificador de esta cuenta en el servidor, **rellenándolo si falta**.
//
// # Por qué hace falta rellenarlo
//
// Se apunta al entrar (`quedarseCon`), y por eso **a quien ya tenía cuenta antes de la
// 2.40.0 le falta**: su equipo guarda la sesión y no el identificador, y nada lo escribía
// después. El síntoma es «Para dar acceso hace falta una cuenta» en un equipo que lleva
// semanas con cuenta, que no se parece a la causa. Lo vio el cliente al dar el primer
// acceso de verdad, y **ninguna prueba podía verlo**: todas crean la cuenta en el momento,
// así que el campo siempre estaba.
//
// Es la misma forma que la pimienta (ADR 0046): **un dato que solo se escribe al entrar
// deja fuera a todo el que ya estaba**, y la salida es la misma, rellenarlo por cortesía
// cuando se necesita.
//
// Y se arregla sin pedirle nada al servidor, porque **el testigo lleva la cuenta dentro**:
// es el nuestro, nos lo dio él al entrar, y lo que afirme lo comprueba él en cada
// petición. Se guarda al leerlo para no volver a mirarlo.
func (a *App) miCuenta() (string, error) {
	d := leerDatosCuenta()
	if d.Cuenta != "" {
		return d.Cuenta, nil
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return "", err
	}
	mia := cuenta.CuentaDelTestigo(token)
	if mia == "" {
		return "", errors.New("Para dar acceso hace falta una cuenta")
	}
	d.Cuenta = mia
	if err := guardarDatosCuenta(d); err != nil {
		return "", err
	}
	return mia, nil
}
