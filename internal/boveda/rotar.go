package boveda

// Rotar las claves de una bóveda de proyecto (ADR 0059).
//
// Es lo que hace que quitarle el acceso a alguien signifique algo. Rotar cambia **las
// dos** claves —la de contenido, con la que se cifra el cuerpo, y la de bóveda, con la
// que se envuelve la cadena— y deja la bóveda abierta solo para quien tiene acceso ahora
// mismo.
//
// **Y son las dos porque con una sola no se corta a nadie de los de antes**, que es el
// agujero que se vio al implementar esto (2026-11-10, comprobado en código): una ranura
// de acceso anterior a la 0059 lleva dentro la **clave de bóveda**, y con ella se
// desenvuelve el sobre `contenido`, o sea la cadena entera, o sea todas las claves de
// contenido que vengan después. Quien tuvo acceso y se la guardó leía lo escrito después
// de rotar. Lo eligió el cliente con el coste delante: rotar rota todo, y un equipo que
// venga por detrás tiene que poder rehacerse.
//
// Lo que permite que se rehaga, y por eso una bóveda de proyecto se queda con la clave de
// la personal en memoria: **la clave de bóveda de cada documento sale de ese documento**,
// por su ranura principal (ver `llavesQueAbrenUnDocumento`). Quien tiene acceso no puede
// hacer eso —su ranura va sellada hacia su identidad, que al fundir no está delante— y
// para él existe `RecogerCadenaDe`.

import (
	"errors"
	"fmt"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// ErrSoloElDueno es que eso lo hace quien comparte, no quien recibe.
var ErrSoloElDueno = errors.New("Rotar la clave de esta bóveda lo hace quien te dio el acceso")

// ErrSinLaPersonal es rotar una bóveda de proyecto que no se abrió con la personal.
//
// Pasa con una abierta desde la línea de comandos con su clave suelta: ahí no hay con qué
// reenvolver la ranura principal, y hacerlo sin ella dejaría la bóveda **sin forma de
// abrirse desde la ventana**, que es justo lo contrario de lo que se quiere.
var ErrSinLaPersonal = errors.New("Esta bóveda hay que abrirla desde tu bóveda personal para poder rotar su clave")

// RotarLasClaves vuelve a cifrar la bóveda con claves nuevas y la deja abierta solo para
// quien tiene acceso ahora mismo. `para` son las identidades de esos titulares, que no
// están en la bóveda y hay que traer del servidor; `porque` es la etiqueta que queda
// apuntada en la cadena.
//
// **Nada se escribe hasta el único `guardar()` del final**, y si falla se repone lo que
// había: una rotación a medias —la cadena nueva con el cuerpo viejo, o la ranura principal
// reenvuelta y el sobre `contenido` no— deja un fichero que no abre nadie.
func (b *Boveda) RotarLasClaves(para map[string]Identidad, porque string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cerrada() {
		return ErrCerrada
	}
	// El sobre `contenido` va envuelto con la clave de bóveda, así que un titular que
	// rotara dejaría al dueño fuera de su propia bóveda.
	if b.sinClaveDeBoveda() {
		return ErrSoloElDueno
	}
	if len(b.principal) == 0 {
		return ErrSinLaPersonal
	}
	if b.soloLectura {
		return ErrFormatoNuevo
	}

	// **Las identidades primero, y todas, antes de tocar nada** (la regla de preguntar
	// antes de preguntar): una que falte a mitad de camino dejaría la rotación hecha para
	// unos y no para otros, y eso es una bóveda que alguien deja de abrir.
	vivos := accesosDe(b.doc)
	for _, id := range vivos {
		if _, hay := para[id]; !hay {
			return fmt.Errorf("Falta la identidad de quien tiene acceso (%s) para volver a sellar su ranura", id)
		}
		if err := b.mismaHuellaQueCuandoSeLeDio(id, para[id]); err != nil {
			return err
		}
	}

	// 1 · Las dos claves nuevas.
	nueva, err := claveDeBovedaNueva()
	if err != nil {
		return err
	}
	// **Y la cadena hereda lo que hubiera, también la degenerada**: en una bóveda que
	// todavía no tenía cadena, su única clave es la de bóveda de antes, y el cuerpo de
	// los ficheros que ya existen está cifrado con ella —el local de un equipo que venga
	// por detrás, la copia `.antes-de-fundir`—. Sin heredarla, esos ficheros no se podrían
	// volver a leer.
	//
	// **Y no es un regalo a quien se la llevó**: para llegar a ella hay que abrir el sobre
	// `contenido`, que a partir de ahora va envuelto con la clave de bóveda nueva. Lo que
	// corta a quien tuvo acceso antes no es esconder esa clave, es que deje de ser la que
	// abre la cadena.
	c := b.claves
	if err := c.poner(porque); err != nil {
		return err
	}
	crudo, err := c.bytes()
	if err != nil {
		return err
	}
	defer cripto.Borrar(crudo)

	cuando := ahora().UTC().Format(time.RFC3339)
	// 2 · La ranura principal, reenvuelta con la clave de la personal: es lo que hace que
	// cualquier equipo del dueño siga abriendo este proyecto.
	principal, err := envolverConSecreto(RanuraPrincipal, b.principal, nueva, cuando)
	if err != nil {
		return err
	}
	// 3 · El sobre `contenido`: la cadena, envuelta con la clave de bóveda nueva.
	sobreContenido, err := cripto.SellarTexto(crudo, nueva, cripto.PerfilLlave)
	if err != nil {
		return err
	}
	// 4 · Y la ranura de cada uno de los que quedan, sellada hacia su identidad. **Nunca
	// las lápidas**: resellarlas sería devolverle el acceso a quien lo perdió.
	resellados := make([]sobre, 0, len(vivos))
	for _, id := range vivos {
		sellada, err := SellarHaciaIdentidad(crudo, para[id])
		if err != nil {
			return err
		}
		resellados = append(resellados, sobre{Tipo: TipoDeAcceso(id), Creado: cuando,
			Contenedor: sellada, Codificacion: CodificacionSellada})
	}

	// 5 · Todo junto y un solo guardado.
	antes := b.apuntarComoEsta()
	b.ponerRanura(principal)
	b.ponerRanura(sobre{Tipo: RanuraDeContenido, Creado: cuando, Contenedor: sobreContenido})
	for _, s := range resellados {
		b.ponerRanura(s)
	}
	b.llave = nueva
	if err := b.ponerEpoca(c, c.Actual); err != nil {
		b.reponer(antes)
		return err
	}
	b.cuerpoSucio = true
	if err := b.guardar(); err != nil {
		b.reponer(antes)
		return err
	}
	return nil
}

// claveDeBovedaNueva son 32 bytes de azar en base64url, **manejados como texto** por lo
// mismo que la de siempre: `--clave-fichero` recorta los saltos de línea del final y una
// clave binaria que acabara en 0x0A se truncaría en silencio.
func claveDeBovedaNueva() ([]byte, error) {
	bruta, err := cripto.Azar(32)
	if err != nil {
		return nil, err
	}
	defer cripto.Borrar(bruta)
	return []byte(b64.EncodeToString(bruta)), nil
}

// ponerRanura deja ese sobre en su tipo, sustituyendo el que hubiera.
func (b *Boveda) ponerRanura(s sobre) {
	if i := b.ranura(s.Tipo); i >= 0 {
		b.doc.Sobres[i] = s
		return
	}
	b.doc.Sobres = append(b.doc.Sobres, s)
}

// mismaHuellaQueCuandoSeLeDio compara la identidad que acaba de dar el servidor con la
// huella que se comparó al dar el acceso (TOFU, ADR 0043).
//
// **Y se aborta diciendo de quién es**, que es la regla de `DarAccesoAlServidor`: puede
// ser que esa persona haya vuelto a instalar Esfinge y puede ser otra cosa, las dos se
// arreglan igual —comparándola otra vez por otro camino— y decidirlo no es de aquí. Sin
// esto, rotar sellaría la bóveda hacia las llaves que el servidor quisiera dar.
func (b *Boveda) mismaHuellaQueCuandoSeLeDio(id string, quien Identidad) error {
	for _, t := range b.cont.Titulares {
		if t.ID != id {
			continue
		}
		if t.Huella == "" {
			return fmt.Errorf("De %s no hay huella apuntada, así que no puedo saber si sus llaves son las de entonces: quítale el acceso y vuelve a dárselo", t.Correo)
		}
		ahora := HuellaDeIdentidad(quien.Suite, quien.Cifrado, quien.Firma)
		if ahora != t.Huella {
			return fmt.Errorf("La huella de %s no es la que comparaste: ahora es %s y tenías apuntada %s. No roto hasta que la compares otra vez", t.Correo, ahora, t.Huella)
		}
		return nil
	}
	return fmt.Errorf("Hay una ranura de acceso (%s) que no sé de quién es, así que no puedo volver a sellarla", id)
}
