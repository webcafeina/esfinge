package boveda

// Las copias que esperan a que quien las recibe tenga cuenta (ADR 0043, B3).
//
// **Por qué existe esto y no un sobre esperando en el servidor.** Cifrar un sobre
// exige las llaves de quien lo recibe, y quien no tiene cuenta no las tiene: el
// servidor entrega unas inventadas pero fijas para no decir quién está en Esfinge,
// así que lo que sale va cifrado hacia nadie. Se podría cifrar con una clave al
// azar y mandarla por correo —es lo que hacen otros—, y entonces **la contraseña
// pasaría por el correo**, que es justo lo que un gestor de contraseñas no debe
// hacer. Se eligió lo otro: el sobre no sale hasta que hay a quién mandarlo.
//
// Mientras tanto queda esto, que es **una nota, no un secreto**: a quién, qué
// entrada y con qué llaves se intentó. El secreto sigue en su entrada de siempre.
//
// Tres reglas que no se pueden perder de vista:
//
//   - **Vive dentro del cuerpo cifrado**, como todo lo demás. Una lista de a quién
//     mandas contraseñas dice tanto como la lista de sitios (ADR 0024).
//   - **Se sincroniza**, así que cualquiera de tus equipos puede rematarlo. El que
//     invitó puede no volver a abrirse en semanas.
//   - **Caduca a los treinta días**, igual que la invitación del servidor. Lo que
//     no llegó en un mes no llega: se vuelve a mandar a mano, sabiéndolo.

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"time"
)

// PlazoPendientes es lo que espera una copia a que la otra persona cree su
// cuenta. **El mismo que dura la invitación en el servidor**: si aquí durara más,
// habría pendientes esperando a un correo que ya nadie recuerda haber recibido.
const PlazoPendientes = 30 * 24 * time.Hour

// Pendiente es una copia mandada a quien todavía no tenía cuenta.
//
// **No lleva el secreto**: lleva a qué entrada apunta. Cuando por fin se manda se
// lee la entrada de entonces, que es lo que se quiere —si la contraseña cambió
// entre medias, lo que llega es la buena— y de paso evita guardar el mismo
// secreto en dos sitios de la bóveda.
type Pendiente struct {
	ID string `json:"id"`
	// Entrada es el identificador de la que se manda. Si ya no está, el pendiente
	// se cae solo: no hay nada que mandar.
	Entrada string `json:"entrada"`
	Correo  string `json:"correo"`
	// Huella es la de las llaves que el servidor dio al mandarlo. **Es el disparo**:
	// mientras siga siendo ésa, esa dirección sigue sin publicar llaves de verdad;
	// en cuanto cambie, hay a quién mandar.
	Huella string `json:"huella"`
	Creado string `json:"creado"`
}

// Pendientes son las copias que siguen esperando, de la más antigua a la más
// nueva. Las caducadas no salen.
func (b *Boveda) Pendientes() []Pendiente {
	b.mu.Lock()
	defer b.mu.Unlock()
	corte := ahora().Add(-PlazoPendientes).UTC().Format(time.RFC3339)
	out := make([]Pendiente, 0, len(b.cont.Envios))
	for _, p := range b.cont.Envios {
		if p.Creado > corte {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Creado < out[j].Creado })
	return out
}

// AnotarPendiente deja dicho que esa copia sigue esperando.
//
// **Una por entrada y correo**: volver a mandarle lo mismo a la misma persona no
// deja dos notas, actualiza la que había. Si no, insistir desde la ventana llenaría
// la bóveda de pendientes que van a acabar todos en el mismo envío.
func (b *Boveda) AnotarPendiente(entrada, correo, huella string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	cuando := ahora().UTC().Format(time.RFC3339)
	for i, p := range b.cont.Envios {
		if p.Entrada == entrada && p.Correo == correo {
			b.cont.Envios[i].Huella = huella
			b.cont.Envios[i].Creado = cuando
			b.cuerpoSucio = true
			return b.guardar()
		}
	}
	b.cont.Envios = append(b.cont.Envios, Pendiente{
		ID:      nuevoIDDePendiente(),
		Entrada: entrada,
		Correo:  correo,
		Huella:  huella,
		Creado:  cuando,
	})
	b.cuerpoSucio = true
	return b.guardar()
}

// OlvidarPendiente quita uno: se mandó, caducó o la entrada ya no está.
func (b *Boveda) OlvidarPendiente(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	fuera := b.cont.Envios[:0]
	for _, p := range b.cont.Envios {
		if p.ID != id {
			fuera = append(fuera, p)
		}
	}
	if len(fuera) == len(b.cont.Envios) {
		return nil
	}
	b.cont.Envios = fuera
	b.cuerpoSucio = true
	return b.guardar()
}

// purgarPendientes se lleva los caducados. Va con la purga de la papelera, al
// abrir, y devuelve si ha quitado algo.
func (b *Boveda) purgarPendientes(limite time.Time) bool {
	corte := limite.UTC().Format(time.RFC3339)
	c := &b.cont
	quedan := make([]Pendiente, 0, len(c.Envios))
	for _, p := range c.Envios {
		if p.Creado > corte {
			quedan = append(quedan, p)
		}
	}
	if len(quedan) == len(c.Envios) {
		return false
	}
	c.Envios = quedan
	return true
}

func nuevoIDDePendiente() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

// fundirPendientes junta las dos listas **como un conjunto por identificador**,
// con la misma regla que los sitios excluidos: lo que estaba en la base y falta en
// un lado es que ese lado lo quitó, y quitarlo gana.
//
// El orden final es **por identificador**, que es el único que los dos equipos
// pueden calcular igual sin ponerse de acuerdo. Si se conservara el de cada lado,
// la bóveda fundida nunca sería igual a la del servidor y se la pasarían sin fin,
// que es lo que ya pasó con las entradas.
//
// Y un detalle que se paga barato: si la nota de un envío ya entregado se pierde,
// **lo peor que pasa es que esa copia llega dos veces**. Al revés —conservar una
// que se quitó— sería lo mismo. Por eso aquí no hay desempates finos.
func fundirPendientes(l, r []Pendiente, b []Pendiente, hayBase bool) []Pendiente {
	en := func(lista []Pendiente) map[string]Pendiente {
		m := map[string]Pendiente{}
		for _, p := range lista {
			m[p.ID] = p
		}
		return m
	}
	ml, mr, mb := en(l), en(r), en(b)
	out := make([]Pendiente, 0, len(ml)+len(mr))
	visto := map[string]bool{}
	for _, m := range []map[string]Pendiente{ml, mr} {
		for id, p := range m {
			if visto[id] {
				continue
			}
			visto[id] = true
			_, enL := ml[id]
			_, enR := mr[id]
			_, enB := mb[id]
			if (enL && enR) || !hayBase || (enL && !enB) || (enR && !enB) {
				// Con los dos lados a mano gana el del servidor, que es quien manda en
				// el resto de la fusión; un pendiente solo cambia al volver a mandarlo.
				if q, hay := mr[id]; hay {
					p = q
				}
				out = append(out, p)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
