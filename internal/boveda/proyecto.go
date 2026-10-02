package boveda

// La ranura que abre una bóveda de proyecto con la bóveda personal (ADR 0050).
//
// Un proyecto es una bóveda igual que cualquier otra —mismo formato, mismas
// clases, misma sincronización— y lo único que la distingue es **con qué se
// abre**: no con una contraseña, sino con la clave de bóveda de la personal.
//
// Y ésa es la decisión que hay que leer antes de tocar este fichero, porque la
// alternativa evidente pierde datos. El cliente pidió que los proyectos «se
// abran con su misma contraseña maestra», y lo que parece es envolver la clave
// del proyecto **con la maestra**. Si se hiciera así:
//
//   - el día que la maestra cambie, habría que reenvolver N ficheros en una
//     operación que puede fallar a medias;
//   - y el día que la maestra se **recupere** —que es el caso para el que existe
//     la clave de recuperación—, los proyectos seguirían envueltos con la maestra
//     vieja, que ya nadie sabe. **Se perderían todos.**
//
// Envolviendo con la clave de bóveda de la personal no pasa ninguna de las dos,
// porque **esa clave no cambia nunca**: cambiar la maestra reenvuelve su sobre y
// la clave sigue siendo la misma. Cambiar la maestra no toca ningún proyecto, y
// recuperar la personal recupera la clave que los abre todos.
//
// De ahí salen tres cosas más que no son detalles:
//
//   - **Una sola entrada en el llavero del sistema.** La ranura de Touch ID va
//     solo en la personal y un proyecto se abre *a través* de ella, así que tras
//     una actualización sale **un** diálogo del sistema y no uno por bóveda.
//   - **Una sola ceremonia de clave de recuperación**, la de la personal. Un
//     proyecto no tiene la suya, y la pantalla que lo crea lo dice.
//   - **`PerfilLlave`, no `PerfilInteractivo`**, por lo mismo que la ranura del
//     sistema: el secreto son los 43 bytes de la clave de la personal, no hay
//     diccionario que valga y conmutar de proyecto no paga un segundo de Argon2.
//
// **Y esta ranura SÍ se sube**, al contrario que la del sistema: es lo que hace
// que el segundo equipo pueda abrir el proyecto. Si acabara en `ranurasLocales`,
// una bóveda creada aquí no la abriría nunca el otro Mac y **no se notaría hasta
// llegar a él**. Lo fija `TestLaRanuraPrincipalSeSube`.
//
// Lo que esto no arregla, y está dicho en `docs/seguridad.md`: perder la bóveda
// personal **y** su clave de recuperación es perder todos los proyectos, aunque
// sus ficheros sobrevivan. La salida es entregarlos (ADR 0051), que les pone
// maestra y recuperación propias.

import (
	"errors"
	"os"
	"sort"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

// RanuraPrincipal es el tipo de sobre que guarda la clave de un proyecto envuelta
// con la clave de bóveda de la personal. **No está en `ranurasLocales`**: se sube.
const RanuraPrincipal = "boveda-principal"

// ErrSinRanuraPrincipal: ese fichero es una bóveda, pero no la abre esta personal.
// Pasa con una bóveda que alguien entregó —se le quita la ranura a propósito— y con
// una que viene de otra bóveda personal.
var ErrSinRanuraPrincipal = errors.New("Esa bóveda no se abre con la tuya; ábrela con su propia contraseña")

// Proyecto es lo que la bóveda personal guarda de cada uno de sus proyectos.
//
// **Y lo que NO guarda es la clave del proyecto**, aunque la primera versión del
// plan la pusiera aquí «por redundancia». No hace falta para nada: lo que abre un
// proyecto es la ranura que va dentro de su propio fichero, y esa ranura **se
// sube**, así que un equipo nuevo se baja el fichero y lo abre con la personal sin
// consultar esta lista. Guardar aquí N claves de bóveda sería amontonar las llaves
// de todos los proyectos en un sitio más sin ganar un solo caso de uso.
//
// Lo que sí hace falta es esto: qué proyectos hay, cómo se llaman y cuál se tocó
// antes, que es lo que la lista necesita **y lo que se sincroniza gratis** con la
// personal, sin una ruta nueva en el servidor.
type Proyecto struct {
	// Ref lo identifica y **es el nombre de su fichero**: hex de 8 bytes. No es el
	// identificador de la bóveda —ése va dentro, en claro, y lo usa la
	// sincronización— porque hace falta poder nombrar el fichero antes de crearlo.
	Ref string `json:"ref"`
	// Nombre es lo que se lee en la lista. Se puede cambiar sin tocar el fichero,
	// que es media razón de que el fichero se llame por la ref.
	Nombre string `json:"nombre"`
	Creado string `json:"creado"`
	// Usado es cuándo se abrió por última vez, que es por lo que se ordena la
	// lista: con muchos proyectos, el que se quiere es casi siempre el último.
	Usado string `json:"usado,omitempty"`
	// Archivado lo saca de la lista del día a día. Ver la entrega 6: archivar
	// **borra el fichero local** y deja el del servidor.
	Archivado bool `json:"archivado,omitempty"`
}

// Proyectos son los que abre esta bóveda, ordenados por uso: el último arriba.
func (b *Boveda) Proyectos() []Proyecto {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := append([]Proyecto(nil), b.cont.Proyectos...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Usado > out[j].Usado })
	return out
}

// Proyecto devuelve uno por su ref.
func (b *Boveda) Proyecto(ref string) (Proyecto, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, p := range b.cont.Proyectos {
		if p.Ref == ref {
			return p, true
		}
	}
	return Proyecto{}, false
}

// PonerProyecto lo añade o lo actualiza por su ref.
func (b *Boveda) PonerProyecto(p Proyecto) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	if p.Ref == "" {
		return errors.New("Un proyecto sin referencia no se puede guardar")
	}
	if p.Creado == "" {
		p.Creado = ahora().UTC().Format(time.RFC3339)
	}
	for i, v := range b.cont.Proyectos {
		if v.Ref == p.Ref {
			b.cont.Proyectos[i] = p
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	b.cont.Proyectos = append(b.cont.Proyectos, p)
	b.cuerpoSucio = true
	return b.guardar()
}

// OlvidarProyecto lo saca de la lista. **No toca su fichero**: borrar la bóveda de
// un proyecto es otra cosa y la hace quien sabe qué satélites tiene al lado.
func (b *Boveda) OlvidarProyecto(ref string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	for i, v := range b.cont.Proyectos {
		if v.Ref == ref {
			b.cont.Proyectos = append(b.cont.Proyectos[:i], b.cont.Proyectos[i+1:]...)
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	return nil
}

// fundirProyectos junta las dos listas **como un conjunto por ref**, a tres bandas
// contra la base, igual que los pendientes y los sitios excluidos: así un proyecto
// que se olvidó aquí no vuelve, y uno que se creó allí llega.
//
// Lo que sí es distinto de los pendientes, que no cambian nunca: **un proyecto se
// edita en los dos equipos a la vez**, y no vale quedarse con el del servidor a
// ciegas. `Usado` es «la última vez que se abrió», así que **las dos son verdad y
// gana la mayor** — es la misma regla que la fecha `usada` de una llave de acceso
// (ADR 0048). Y lo que no se puede decidir por el reloj se decide por la base: gana
// el lado que lo cambió, y si lo cambiaron los dos, el que es mayor por cadena, que
// es arbitrario pero **igual en los dos equipos**, que es lo que de verdad hace
// falta (ADR 0038: cualquier limpieza que corra en varios equipos tiene que elegir
// igual en todos).
func fundirProyectos(l, r, b []Proyecto, hayBase bool) []Proyecto {
	en := func(lista []Proyecto) map[string]Proyecto {
		m := map[string]Proyecto{}
		for _, p := range lista {
			m[p.Ref] = p
		}
		return m
	}
	ml, mr, mb := en(l), en(r), en(b)
	out := make([]Proyecto, 0, len(ml)+len(mr))
	visto := map[string]bool{}
	for _, m := range []map[string]Proyecto{ml, mr} {
		for ref := range m {
			if visto[ref] {
				continue
			}
			visto[ref] = true
			pl, enL := ml[ref]
			pr, enR := mr[ref]
			pb, enB := mb[ref]
			if !((enL && enR) || !hayBase || (enL && !enB) || (enR && !enB)) {
				continue
			}
			switch {
			case !enL:
				out = append(out, pr)
			case !enR:
				out = append(out, pl)
			default:
				out = append(out, unProyecto(pl, pr, pb, enB))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return out
}

func unProyecto(l, r, b Proyecto, hayBase bool) Proyecto {
	out := r
	// La última vez que se abrió es la mayor de las dos: las dos ocurrieron.
	if l.Usado > out.Usado {
		out.Usado = l.Usado
	}
	// El nombre: el lado que lo cambió respecto a la base.
	switch {
	case !hayBase:
		if l.Nombre > r.Nombre {
			out.Nombre = l.Nombre
		}
	case l.Nombre != b.Nombre && r.Nombre == b.Nombre:
		out.Nombre = l.Nombre
	case l.Nombre != b.Nombre && r.Nombre != b.Nombre && l.Nombre > r.Nombre:
		out.Nombre = l.Nombre
	}
	// Y archivar: si los dos equipos lo movieron a sitios distintos gana
	// **desarchivado**, que es el estado que lo enseña en vez de esconderlo. Un
	// proyecto archivado que reaparece se vuelve a archivar con un clic; uno que
	// desaparece hay que saber que estaba para ir a buscarlo.
	switch {
	case !hayBase:
		out.Archivado = l.Archivado && r.Archivado
	case l.Archivado != b.Archivado && r.Archivado == b.Archivado:
		out.Archivado = l.Archivado
	case l.Archivado != b.Archivado && r.Archivado != b.Archivado:
		out.Archivado = false
	}
	return out
}

// LlaveParaProyectos devuelve **una copia** de la clave de esta bóveda, que es con
// lo que se abren sus proyectos.
//
// Sacar la clave de bóveda de aquí es algo que no se hace a la ligera, así que el
// nombre dice para qué es y van las dos reglas de quien la pide: **borrarla con
// `cripto.Borrar`** cuando deje de hacer falta, y no dejarla cruzar el puente ni
// llegar a un fichero. Lo que la ventana puede pedir son proyectos, nunca esto.
func (b *Boveda) LlaveParaProyectos() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return nil
	}
	return append([]byte(nil), b.llave...)
}

// AbrirConLaClave abre una bóveda con **su propia clave de bóveda**, sin pasar por
// ninguna ranura y sin derivar nada.
//
// Existe para una cosa concreta: con un proyecto abierto, la bóveda personal está
// cerrada y aun así hay que poder leer y escribir su lista de proyectos —el nombre
// de uno nuevo, cuándo se abrió el último— sin pedir la contraseña maestra otra vez.
// Lo que se guarda en memoria al conmutar es esa clave, así que esto es lo que la
// convierte en poder volver a abrir el fichero.
//
// **No es una puerta nueva**: quien tiene la clave de bóveda ya tiene el contenido,
// porque es con lo que se descifra el cuerpo. Lo que añade es poder hacerlo sobre el
// fichero, que es lo que hace falta para escribir.
func AbrirConLaClave(ruta string, llave []byte) (*Boveda, error) {
	if len(llave) == 0 {
		return nil, ErrCerrada
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	doc, err := leerDocumento(datos)
	if err != nil {
		return nil, err
	}
	// Sin purgar: esto se abre para una cosa pequeña y concreta, y vaciar la
	// papelera es trabajo de abrir de verdad.
	return conLlave(ruta, doc, append([]byte(nil), llave...), false)
}

// CrearProyecto hace una bóveda de proyecto, cuya única ranura es la que abre la
// personal. No devuelve clave de recuperación porque no tiene: la de la personal
// lo recupera.
func CrearProyecto(ruta string, llavePrincipal []byte) (*Boveda, error) {
	if len(llavePrincipal) == 0 {
		return nil, ErrCerrada
	}
	b, err := sinRanuras(ruta)
	if err != nil {
		return nil, err
	}
	s, err := envolverConSecreto(RanuraPrincipal, llavePrincipal, b.llave, b.doc.Cambiada)
	if err != nil {
		return nil, err
	}
	b.doc.Sobres = append(b.doc.Sobres, s)
	if ruta == "" {
		return b, nil
	}
	if err := b.Guardar(); err != nil {
		return nil, err
	}
	return b, nil
}

// AbrirProyecto abre una bóveda de proyecto con la clave de la personal,
// **probando solo esa ranura**, por lo mismo que `AbrirConElSistema`: probarlas
// todas pagaría antes una derivación interactiva contra la maestra, que es el
// segundo que esto viene a quitar.
func AbrirProyecto(ruta string, llavePrincipal []byte) (*Boveda, error) {
	// La llave se mira **antes** de tocar el disco: sin bóveda personal esto no va
	// a poder abrir nada, y lo que hay que contestar es eso y no un error de
	// fichero que manda a mirar al sitio equivocado.
	if len(llavePrincipal) == 0 {
		return nil, ErrCerrada
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	return abrirProyectoBytes(ruta, datos, llavePrincipal)
}

// AbrirProyectoBytes es lo mismo sin tocar el disco, para la bóveda que viaja.
func AbrirProyectoBytes(ruta string, datos, llavePrincipal []byte) (*Boveda, error) {
	return abrirProyectoBytes(ruta, datos, llavePrincipal)
}

func abrirProyectoBytes(ruta string, datos, llavePrincipal []byte) (*Boveda, error) {
	if len(llavePrincipal) == 0 {
		return nil, ErrCerrada
	}
	doc, err := leerDocumento(datos)
	if err != nil {
		return nil, err
	}
	i := -1
	for j, s := range doc.Sobres {
		if s.Tipo == RanuraPrincipal {
			i = j
			break
		}
	}
	if i < 0 {
		return nil, ErrSinRanuraPrincipal
	}
	llave, err := cripto.AbrirTexto(doc.Sobres[i].Contenedor, llavePrincipal)
	if err != nil {
		// La ranura está pero esta personal no la abre: el proyecto es de otra
		// bóveda personal. No es un fichero roto y no se dice que lo sea.
		return nil, ErrSinRanuraPrincipal
	}
	return conLlave(ruta, doc, llave, true)
}

// PonerRanuraPrincipal deja que esa personal abra esta bóveda, y reemplaza la
// ranura si ya había una. Exige la bóveda abierta, como rotar la recuperación: es
// lo que impide ponerla desde fuera.
//
// Es también cómo se **adopta** una bóveda que alguien entregó: se abre con su
// contraseña y se le pone esta ranura.
func (b *Boveda) PonerRanuraPrincipal(llavePrincipal []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	if len(llavePrincipal) == 0 {
		return errors.New("No hay bóveda principal con la que abrir ésta")
	}
	s, err := envolverConSecreto(RanuraPrincipal, llavePrincipal, b.llave, ahora().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	if i := b.ranura(RanuraPrincipal); i >= 0 {
		b.doc.Sobres[i] = s
	} else {
		b.doc.Sobres = append(b.doc.Sobres, s)
	}
	return b.guardar()
}

// QuitarRanuraPrincipal corta el lazo: esta bóveda deja de abrirse con la personal.
// Es el paso 2 de entregar una bóveda (ADR 0051), y sin él quien la entrega sigue
// pudiendo abrir la del cliente.
//
// **No exige la bóveda abierta a propósito**: quitar una ranura no da acceso a
// nada, y hace falta poder hacerlo sobre la copia que se va a entregar.
func (b *Boveda) QuitarRanuraPrincipal() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	i := b.ranura(RanuraPrincipal)
	if i < 0 {
		return nil
	}
	b.doc.Sobres = append(b.doc.Sobres[:i], b.doc.Sobres[i+1:]...)
	return b.guardar()
}

// TieneRanuraPrincipal dice si esta bóveda es un proyecto de alguna personal.
func (b *Boveda) TieneRanuraPrincipal() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ranura(RanuraPrincipal) >= 0
}

// RanuraPrincipalEn dice si el fichero de esa ruta trae la ranura, **sin abrirlo**.
// Es lo que mira la lista de Proyectos para distinguir un proyecto de una bóveda
// suelta que alguien dejó en la carpeta, sin pedirle nada a nadie.
func RanuraPrincipalEn(ruta string) bool {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return false
	}
	doc, err := leerDocumento(datos)
	if err != nil {
		return false
	}
	for _, s := range doc.Sobres {
		if s.Tipo == RanuraPrincipal {
			return true
		}
	}
	return false
}
