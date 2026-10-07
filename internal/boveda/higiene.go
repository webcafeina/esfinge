package boveda

// Lo que está mal en una bóveda, **contado dentro de ella**.
//
// # Por qué vive aquí y no en quien pregunta
//
// Para saber qué contraseñas están reutilizadas hay que **comparar los secretos**, y
// eso solo se puede hacer donde los secretos están. Calculándolo fuera —sacando cada
// entrada con `Ver` y agrupando en un mapa— la bóveda entera acaba en claro en el
// montón de quien pregunta, que es justo lo que `SinSecretos` existe para evitar.
//
// Aquí dentro se recorre una vez, se compara, y **lo que sale son identificadores**.

import (
	"sort"
	"time"
)

// Higiene es lo que está mal, por identificador. **Sin un solo secreto**: dice que
// estas tres comparten contraseña, nunca cuál es.
type Higiene struct {
	// Reutilizadas son los grupos de entradas que comparten contraseña.
	//
	// **No es lo mismo que `Repetidas`**, y confundirlos es contestar la pregunta
	// equivocada con toda confianza: aquello es «la misma cuenta dos veces», un
	// ayudante para cuando se importa de otro gestor, y esto es «la misma contraseña
	// en sitios distintos», que es lo que alguien quiere saber cuando lo pregunta.
	Reutilizadas [][]string
	// SinCodigo son las credenciales que no tienen segundo factor.
	SinCodigo []string
	// Caducadas son las tarjetas y documentos que ya han caducado. **No hay caducidad
	// de contraseñas en este formato**, y no se inventa una.
	Caducadas []string
}

// Higiene recorre la bóveda **una vez** y cuenta lo que está mal.
//
// El orden de lo que sale es estable: los identificadores dentro de cada grupo y los
// grupos entre sí. **Un mapa de Go se recorre en orden aleatorio**, así que sin
// ordenar, dos llamadas iguales darían respuestas distintas — y a quien pregunta, un
// agente que compara dos respuestas, eso le parecería que algo ha cambiado.
func (b *Boveda) Higiene(ahora time.Time) Higiene {
	b.mu.Lock()
	defer b.mu.Unlock()

	var h Higiene
	porSecreto := map[string][]string{}
	for _, e := range b.cont.Entradas {
		if e.Papelera {
			continue
		}
		if e.Secreto != "" {
			porSecreto[e.Secreto] = append(porSecreto[e.Secreto], e.ID)
		}
		if e.Tipo == TipoCredencial && e.TOTP == "" {
			h.SinCodigo = append(h.SinCodigo, e.ID)
		}
		if yaCaduco(e, ahora) {
			h.Caducadas = append(h.Caducadas, e.ID)
		}
	}
	for _, ids := range porSecreto {
		if len(ids) > 1 {
			sort.Strings(ids)
			h.Reutilizadas = append(h.Reutilizadas, ids)
		}
	}
	// Por el primero de cada grupo, que ya está ordenado y es único entre grupos.
	sort.Slice(h.Reutilizadas, func(i, j int) bool {
		return h.Reutilizadas[i][0] < h.Reutilizadas[j][0]
	})
	sort.Strings(h.SinCodigo)
	sort.Strings(h.Caducadas)
	return h
}

// yaCaduco mira el `Caduca` de una tarjeta o un documento, que viene como «2026-03».
func yaCaduco(e Entrada, ahora time.Time) bool {
	if e.Caduca == "" {
		return false
	}
	return e.Caduca < ahora.UTC().Format("2006-01")
}
