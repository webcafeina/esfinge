package boveda

import (
	"encoding/json"
	"time"
)

// Repetidas son entradas **iguales en todo** a otra de la misma bóveda menos en
// su identificador, sus fechas y su revisión.
//
// # De dónde salen
//
// Dos bóvedas importadas por separado del mismo gestor tienen las mismas cuentas
// con identificadores distintos. Al entrar en una cuenta con «Juntar», `Traer`
// solo comparaba identificadores, así que las traía todas otra vez: la bóveda
// quedaba con cada cuenta dos veces, en todos los equipos. Lo vio el cliente con
// sus dos Macs (2026-09-21). `Traer` ya no lo hace; esto limpia lo que quedó.
//
// Solo cuenta lo idéntico. Dos entradas del mismo sitio y usuario con distinta
// contraseña no son repetidas: una de las dos es la buena, y eso no lo puede
// decidir Esfinge.

// contenidoDe es la entrada sin lo que la hace otra: identificador, fechas y
// revisión. Dos entradas con el mismo contenido son la misma cuenta dos veces.
func contenidoDe(e Entrada) string {
	e.ID, e.Creada, e.Cambiada, e.Revision = "", "", "", 0
	j, err := json.Marshal(e)
	if err != nil {
		return "" // no se compara lo que no se sabe escribir
	}
	return string(j)
}

// repetidas devuelve los identificadores de las entradas que sobran, con el
// cerrojo cogido. De cada grupo se queda **la de identificador menor**, que no
// depende del orden ni del equipo: si dos equipos limpian a la vez, los dos
// mandan a la papelera las mismas y no se quedan sin ninguna.
func (b *Boveda) repetidas() []string {
	menor := map[string]string{} // contenido → identificador que se queda
	for _, e := range b.cont.Entradas {
		if e.Papelera {
			continue
		}
		c := contenidoDe(e)
		if c == "" {
			continue
		}
		if id, hay := menor[c]; !hay || e.ID < id {
			menor[c] = e.ID
		}
	}
	var sobran []string
	for _, e := range b.cont.Entradas {
		if e.Papelera {
			continue
		}
		if c := contenidoDe(e); c != "" && menor[c] != e.ID {
			sobran = append(sobran, e.ID)
		}
	}
	return sobran
}

// Repetidas dice cuántas entradas sobran por estar repetidas.
func (b *Boveda) Repetidas() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return 0, ErrCerrada
	}
	return len(b.repetidas()), nil
}

// QuitarRepetidas manda a la papelera las que sobran, **no las borra**: se pueden
// restaurar durante treinta días, como cualquier otra. Guarda una sola vez.
func (b *Boveda) QuitarRepetidas() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return 0, ErrCerrada
	}
	sobran := map[string]bool{}
	for _, id := range b.repetidas() {
		sobran[id] = true
	}
	if len(sobran) == 0 {
		return 0, nil
	}
	cuando := ahora().UTC().Format(time.RFC3339)
	for i, e := range b.cont.Entradas {
		if sobran[e.ID] {
			b.cont.Entradas[i].Papelera = true
			b.cont.Entradas[i].BorradaEn = cuando
			b.cont.Entradas[i].Revision++
		}
	}
	b.cuerpoSucio = true
	return len(sobran), b.guardar()
}
