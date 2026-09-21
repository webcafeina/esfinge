package boveda

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// Repetidas son **la misma cuenta dos veces** en una bóveda: misma clase, mismo
// título, mismo usuario y mismo secreto, y nada que choque en lo demás.
//
// # De dónde salen
//
// Dos bóvedas importadas por separado del mismo gestor tienen las mismas cuentas
// con identificadores distintos. Al entrar en una cuenta con «Juntar», `Traer`
// solo comparaba identificadores, así que las traía todas otra vez: la bóveda
// quedaba con cada cuenta dos veces, en todos los equipos. Lo vio el cliente con
// sus dos Macs (2026-09-21).
//
// # Por qué no «iguales en todo»
//
// Fue la primera versión (2.24.2), y en los Macs del cliente no encontró ninguna:
// dos importaciones hechas con versiones distintas de Esfinge dejan la misma
// cuenta con diferencias que la ventana no enseña —etiquetas, carpeta, columnas
// del gestor guardadas aparte, la web escrita de otra forma—. A la vista eran
// idénticas. Así que lo que decide es **lo que identifica la cuenta y su
// secreto**, y lo demás se junta en la que se queda: no se pierde nada.
//
// Lo que sí separa dos entradas es que **choquen**: las dos con notas y
// distintas, o con semillas de código distintas. Ahí una de las dos es la buena,
// y eso no lo puede decidir Esfinge. La carpeta y los campos del gestor guardados
// aparte no separan nada: si las dos los tienen distintos, se queda lo de la que
// se queda, que es ordenar y no un secreto.

// claveDeCuenta es lo que tiene que coincidir para que dos entradas sean la misma
// cuenta. Los espacios de alrededor no cuentan —cada importador los ha tratado a
// su manera—, salvo en el secreto, donde pueden ser parte de él.
func claveDeCuenta(e Entrada) string {
	t := strings.TrimSpace
	partes := []string{
		string(e.Tipo), t(e.Titulo), t(e.Usuario), e.Secreto,
		t(e.Titular), soloCifras(e.Numero), t(e.Caduca), e.Verificacion,
		t(e.NombreCompleto), t(e.Documento), t(e.NumeroDocumento),
	}
	if e.Tipo == TipoNota {
		// En una nota, el texto es el secreto: tiene que ser el mismo.
		partes = append(partes, e.Notas)
	}
	return strings.Join(partes, "\x00")
}

// chocan dice si juntar `otra` en `queda` perdería algo que importa: unas notas o
// una semilla de código que las dos tienen, distintas.
func chocan(queda, otra Entrada) bool {
	distinto := func(a, b string) bool {
		return strings.TrimSpace(a) != "" && strings.TrimSpace(b) != "" && strings.TrimSpace(a) != strings.TrimSpace(b)
	}
	return distinto(queda.Notas, otra.Notas) || distinto(queda.TOTP, otra.TOTP)
}

// juntarEn pasa a `queda` lo que `otra` tiene y ella no. Solo suma: nunca cambia
// algo que `queda` ya tuviera (por eso antes se mira `chocan`).
func juntarEn(queda *Entrada, otra Entrada) {
	if strings.TrimSpace(queda.Notas) == "" {
		queda.Notas = otra.Notas
	}
	if strings.TrimSpace(queda.TOTP) == "" {
		queda.TOTP = otra.TOTP
	}
	if strings.TrimSpace(queda.Carpeta) == "" {
		queda.Carpeta = otra.Carpeta
	}
	queda.Sitios = unirSinRepetir(queda.Sitios, otra.Sitios)
	queda.Etiquetas = unirSinRepetir(queda.Etiquetas, otra.Etiquetas)
	hay := map[string]bool{}
	for _, a := range queda.Historial {
		hay[a.Secreto] = true
	}
	for _, a := range otra.Historial {
		if !hay[a.Secreto] && a.Secreto != queda.Secreto {
			queda.Historial = append(queda.Historial, a)
			hay[a.Secreto] = true
		}
	}
	sort.SliceStable(queda.Historial, func(i, j int) bool { return queda.Historial[i].Hasta > queda.Historial[j].Hasta })
	if len(queda.Historial) > maximoHistorial {
		queda.Historial = queda.Historial[:maximoHistorial]
	}
	for k, v := range otra.Extra {
		if _, esta := queda.Extra[k]; !esta {
			if queda.Extra == nil {
				queda.Extra = map[string]json.RawMessage{}
			}
			queda.Extra[k] = v
		}
	}
}

func copiaHonda(e Entrada) Entrada {
	e.Sitios = append([]string(nil), e.Sitios...)
	e.Etiquetas = append([]string(nil), e.Etiquetas...)
	e.Historial = append([]Antigua(nil), e.Historial...)
	if e.Extra != nil {
		extra := make(map[string]json.RawMessage, len(e.Extra))
		for k, v := range e.Extra {
			extra[k] = v
		}
		e.Extra = extra
	}
	return e
}

func unirSinRepetir(a, b []string) []string {
	hay := map[string]bool{}
	for _, s := range a {
		hay[strings.ToLower(strings.TrimSpace(s))] = true
	}
	for _, s := range b {
		if k := strings.ToLower(strings.TrimSpace(s)); k != "" && !hay[k] {
			a = append(a, s)
			hay[k] = true
		}
	}
	return a
}

// limpieza es lo que haría QuitarRepetidas: qué entradas sobran y cómo queda cada
// una de las que se quedan.
//
// De cada grupo se queda **la de identificador menor**, y se le juntan las demás
// en orden de identificador. No depende del orden de la lista ni del equipo: si
// dos equipos limpian a la vez, mandan a la papelera las mismas, dejan la que se
// queda igual, y la fusión no ve nada que discutir.
func (b *Boveda) limpieza() (sobran map[string]bool, quedan map[string]Entrada) {
	grupos := map[string][]Entrada{}
	for _, e := range b.cont.Entradas {
		if !e.Papelera {
			k := claveDeCuenta(e)
			grupos[k] = append(grupos[k], e)
		}
	}
	sobran, quedan = map[string]bool{}, map[string]Entrada{}
	for _, g := range grupos {
		if len(g) < 2 {
			continue
		}
		sort.Slice(g, func(i, j int) bool { return g[i].ID < g[j].ID })
		// Una copia de verdad: juntar añade a sus listas y a su mapa, y con la
		// copia plana de Go eso escribiría en la entrada de la bóveda aunque solo
		// se estuviera contando.
		queda := copiaHonda(g[0])
		cambia := false
		for _, otra := range g[1:] {
			if otra.ID == queda.ID || chocan(queda, otra) {
				continue
			}
			juntarEn(&queda, otra)
			sobran[otra.ID] = true
			cambia = true
		}
		if cambia {
			quedan[queda.ID] = queda
		}
	}
	return sobran, quedan
}

// Repetidas dice cuántas entradas sobran por ser la misma cuenta que otra.
func (b *Boveda) Repetidas() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return 0, ErrCerrada
	}
	sobran, _ := b.limpieza()
	return len(sobran), nil
}

// QuitarRepetidas junta cada grupo en la de identificador menor y manda las demás
// a la papelera, **no las borra**: se pueden restaurar durante treinta días, como
// cualquier otra. Guarda una sola vez.
func (b *Boveda) QuitarRepetidas() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return 0, ErrCerrada
	}
	sobran, quedan := b.limpieza()
	if len(sobran) == 0 {
		return 0, nil
	}
	cuando := ahora().UTC().Format(time.RFC3339)
	for i, e := range b.cont.Entradas {
		switch {
		case sobran[e.ID]:
			b.cont.Entradas[i].Papelera = true
			b.cont.Entradas[i].BorradaEn = cuando
			b.cont.Entradas[i].Revision++
		case quedan[e.ID].ID != "":
			q := quedan[e.ID]
			if contenidoDe(q) != contenidoDe(e) {
				q.Cambiada = cuando
				q.Revision++
				b.cont.Entradas[i] = q
			}
		}
	}
	b.cuerpoSucio = true
	return len(sobran), b.guardar()
}

// contenidoDe es la entrada sin lo que la hace otra: identificador, fechas y
// revisión. `Traer` lo usa para no traer lo que ya está.
func contenidoDe(e Entrada) string {
	e.ID, e.Creada, e.Cambiada, e.Revision = "", "", "", 0
	j, err := json.Marshal(e)
	if err != nil {
		return "" // no se compara lo que no se sabe escribir
	}
	return string(j)
}
