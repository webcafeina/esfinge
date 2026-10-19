package agente

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// hablar mete esas líneas por la entrada y devuelve lo que sale, una respuesta por
// línea.
func hablar(t *testing.T, llamar Llamar, lineas ...string) []map[string]any {
	t.Helper()
	var sale bytes.Buffer
	if err := Hablar(strings.NewReader(strings.Join(lineas, "\n")+"\n"), &sale, "2.44.0", llamar, nil); err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(sale.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("lo que ha salido no es JSON: %q", l)
		}
		out = append(out, m)
	}
	return out
}

// **El saludo y la lista se contestan sin Esfinge**, y ésta es la prueba que vigila esa
// restricción (ADR 0054).
//
// Claude Code pide `tools/list` al abrir la sesión y **se queda con lo que le demos**:
// si esto dependiera del socket, un arranque con la ventana cerrada dejaría al agente
// sin herramientas toda la sesión, aunque se abriera dos minutos después. Por eso el
// catálogo son datos de este paquete, y por eso la llamada de abajo **falla a
// propósito**: si alguien mueve la tabla a `internal/app`, esto se pone rojo.
func TestElSaludoYLaListaSeContestanSinEsfinge(t *testing.T) {
	sinNadieAlOtroLado := func(Herramienta, map[string]any) Resultado {
		t.Error("tools/list no puede necesitar a Esfinge")
		return fallo("no")
	}
	rs := hablar(t, sinNadieAlOtroLado,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	if len(rs) != 2 {
		t.Fatalf("han salido %d respuestas: %+v", len(rs), rs)
	}

	saludo := rs[0]["result"].(map[string]any)
	if saludo["protocolVersion"] != RevisionDeMCP {
		t.Errorf("el saludo dice la revisión %v", saludo["protocolVersion"])
	}
	if si := saludo["serverInfo"].(map[string]any); si["name"] != NombreDelServidor || si["version"] != "2.44.0" {
		t.Errorf("el saludo dice %+v", si)
	}

	lista := rs[1]["result"].(map[string]any)["tools"].([]any)
	if len(lista) != len(LasHerramientas) {
		t.Fatalf("la lista trae %d herramientas y el catálogo tiene %d", len(lista), len(LasHerramientas))
	}
}

// **Una notificación no se contesta.** Mandar algo a un mensaje sin identificador rompe
// a los clientes estrictos, y es la clase de cosa que solo se ve con un cliente de
// verdad delante.
func TestUnaNotificacionNoSeContesta(t *testing.T) {
	rs := hablar(t, nil,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":7,"method":"ping"}`,
	)
	if len(rs) != 1 {
		t.Fatalf("han salido %d respuestas y tenía que salir una: %+v", len(rs), rs)
	}
	if rs[0]["id"] != float64(7) {
		t.Errorf("la respuesta que ha salido no es la del ping: %+v", rs[0])
	}
}

// Lo que no se puede atender va **como resultado con `isError`**, no como error de
// JSON-RPC: es la convención de MCP para lo que el modelo tiene que leer y explicar.
func TestUnaHerramientaQueNoExisteSaleComoResultado(t *testing.T) {
	rs := hablar(t, nil, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"esfinge_inventada"}}`)
	r := rs[0]["result"].(map[string]any)
	if r["isError"] != true {
		t.Fatalf("no viene marcado como error: %+v", rs[0])
	}
	if rs[0]["error"] != nil {
		t.Error("ha salido además como error de protocolo, y eso es lo que no hay que hacer")
	}
}

// Y el catálogo tiene que cumplir lo que los clientes exigen y lo que esta casa exige.
func TestElCatalogoEstaBienEscrito(t *testing.T) {
	// **Sin `ñ` ni acentos**: los clientes validan contra esto y una tilde deja la
	// herramienta fuera sin decir por qué.
	comoLoQuierenLosClientes := regexp.MustCompile(`^esfinge_[a-z0-9_]+$`)
	vistos := map[string]bool{}
	for _, h := range LasHerramientas {
		if !comoLoQuierenLosClientes.MatchString(h.Nombre) {
			t.Errorf("el nombre %q no vale: los clientes solo admiten letras sin acentos, cifras y guiones bajos", h.Nombre)
		}
		if vistos[h.Nombre] {
			t.Errorf("la herramienta %q está dos veces", h.Nombre)
		}
		vistos[h.Nombre] = true
		if strings.TrimSpace(h.Descripcion) == "" {
			t.Errorf("la herramienta %q no tiene descripción, que es lo único que el modelo lee para decidir", h.Nombre)
		}
		if h.Verbo == "" {
			t.Errorf("la herramienta %q no dice qué verbo pide", h.Nombre)
		}
	}
	// Y toda herramienta pide un verbo **que existe**: una que pidiera uno que no está
	// en la lista contestaría «no entiendo» a quien la llamara, y eso no lo ve nadie
	// hasta que un agente lo intenta.
	for _, h := range LasHerramientas {
		hay := false
		for _, q := range LoQueSePuedePedir {
			if q == h.Verbo {
				hay = true
			}
		}
		if !hay {
			t.Errorf("la herramienta %q pide el verbo %q, que no está en la lista", h.Nombre, h.Verbo)
		}
	}
}

// **Sin Esfinge al otro lado se contesta, y se dice qué hacer.** Un servidor MCP que se
// muere cuando la aplicación está cerrada deja al agente sin saber por qué.
func TestSinEsfingeSeContestaIgual(t *testing.T) {
	var sale bytes.Buffer
	err := Traducir(
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"esfinge_buscar","arguments":{"texto":"x"}}}`+"\n"),
		&sale, "2.44.0", "/no/existe/este.sock")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sale.String(), "Esfinge no está abierta") {
		t.Fatalf("no dice qué pasa: %s", sale.String())
	}
	if !strings.Contains(sale.String(), `"isError":true`) {
		t.Errorf("no viene marcado como error: %s", sale.String())
	}
}

// **Lo que sale hacia el modelo es lo que hay que mirar, no lo que devuelve la fuente.**
//
// Esta prueba existe por un fallo que se coló: el tope de resultados estaba puesto y el
// total viajaba por el canal, pero **la aduana hacia el modelo seguía diciendo
// `cuantas = las que vuelven`**, o sea 25 siempre. La prueba que había miraba la fuente,
// que sí daba el total bueno. Un cambio a medias, una aserción en el sitio cómodo, y un
// agente que creería que la bóveda tiene veinticinco cuentas.
func TestLoQueSaleHaciaElModeloDiceCuantasHayDeVerdad(t *testing.T) {
	salida := soloLoQueSale(
		Herramienta{Verbo: QueBuscar},
		Respuesta{OK: true, Entradas: []Entrada{{ID: "a1"}, {ID: "b2"}}, Cuantas: 1843},
	)
	m, ok := salida.(map[string]any)
	if !ok {
		t.Fatalf("lo que sale no es un objeto: %T", salida)
	}
	if m["devueltas"] != 2 {
		t.Errorf("dice que vuelven %v y vuelven dos", m["devueltas"])
	}
	// **`coincidencias`, no «enTotal».** El nombre viejo no decía si contaba la
	// bóveda entera o lo que casa con la búsqueda, y con cero resultados esa
	// diferencia es la que hay entre «no tienes nada guardado» y «aquí no hay nada».
	// Lo señaló el agente del cliente el 2026-10-13.
	if m["coincidencias"] != 1843 {
		t.Errorf("dice que coinciden %v y coinciden 1843: un agente creería que la bóveda "+
			"tiene las que le han cabido", m["coincidencias"])
	}
}

// **Sin resultados sale `[]`, nunca `null`.**
//
// En Go un slice sin elementos serializa como `null`, y eso rompe a cualquier cliente
// que lo recorra sin comprobarlo antes — que es lo normal cuando el esquema dice que
// ahí va una lista. Lo vio el agente del cliente buscando algo que no existía.
func TestBuscarSinResultadosDevuelveUnaListaVacia(t *testing.T) {
	salida := soloLoQueSale(Herramienta{Verbo: QueBuscar}, Respuesta{OK: true})
	m, ok := salida.(map[string]any)
	if !ok {
		t.Fatalf("lo que sale no es un objeto: %T", salida)
	}
	es, vale := m["entradas"].([]Entrada)
	if !vale {
		t.Fatalf("«entradas» no es una lista: %T", m["entradas"])
	}
	if es == nil {
		t.Error("«entradas» sale nula, y en JSON eso es null: un cliente que la recorra revienta")
	}
	crudo, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(crudo, []byte(`"entradas":null`)) {
		t.Errorf("sale null por el canal: %s", crudo)
	}
}

// Y lo de copiar: **lo que sale dice que se ha copiado, nunca lo copiado**.
func TestLoQueSaleAlCopiarNoLlevaLaContrasena(t *testing.T) {
	salida := soloLoQueSale(
		Herramienta{Verbo: QueCopiarSecreto},
		Respuesta{OK: true, Copiado: &Copiado{Portapapeles: 30, Titulo: "GitHub"}},
	)
	crudo, err := json.Marshal(salida)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(crudo), "GitHub") {
		t.Errorf("no dice de qué entrada es: %s", crudo)
	}
	if strings.Contains(string(crudo), elSecreto) {
		t.Errorf("lleva la contraseña dentro: %s", crudo)
	}
}

// **Lo que el esquema declara tiene que llegar entero a la petición.**
//
// Ésta es la prueba que faltaba, y por su hueco se cayeron `sitios` y `etiquetas`
// durante cinco días. El esquema los declaraba, el núcleo sabía aplicarlos, y **el
// traductor de en medio no los copiaba**: el agente los mandaba, nadie daba error y la
// entrada nacía sin sitio — o sea sin poder rellenarse sola, que es para lo que existe
// la bóveda.
//
// Lo que lo dejó pasar es de método y vale para cualquier cosa con tres tramos: la
// guarda que había comparaba **los dos extremos** —el esquema contra el núcleo— y daba
// por supuesto el camino entre ellos. Comparar las puntas no prueba que estén unidas.
func TestLoQueDeclaraElEsquemaLlegaALaPeticion(t *testing.T) {
	// Un valor de cada tipo que el esquema puede declarar, para poder armar unos
	// argumentos verosímiles a partir de él y no de una lista escrita a mano.
	ejemplo := map[string]any{
		"string":  "x",
		"boolean": true,
		// JSON no distingue enteros: todo número llega como float64, y por eso el
		// esquema puede decir «integer» y aquí se arma igual.
		"number":  float64(24),
		"integer": float64(24),
		"array":   []any{"https://ejemplo.com"},
	}

	for _, h := range LasHerramientas {
		if len(h.Esquema.Propiedades) == 0 {
			continue
		}
		args := map[string]any{}
		for campo, c := range h.Esquema.Propiedades {
			v, hay := ejemplo[c.Tipo]
			if !hay {
				t.Errorf("%s: el campo %q declara el tipo %q, que esta prueba no sabe armar",
					h.Nombre, campo, c.Tipo)
				continue
			}
			// Los enumerados solo admiten lo suyo.
			if len(c.Enum) > 0 {
				v = c.Enum[0]
			}
			args[campo] = v
		}

		var p Peticion
		p.Que = h.Verbo
		ponerArgumentos(&p, args)

		for campo := range h.Esquema.Propiedades {
			if !llega(p, campo) {
				t.Errorf("%s: el esquema declara %q y la petición no lo lleva: el agente lo "+
					"manda, nadie da error y se pierde por el camino", h.Nombre, campo)
			}
		}
	}
}

// llega dice si ese campo del esquema se ve en la petición ya traducida.
func llega(p Peticion, campo string) bool {
	switch campo {
	case "boveda":
		return p.Boveda != ""
	case "destino":
		return p.Destino != ""
	case "dejar_copia":
		return p.DejarCopia
	case "texto":
		return p.Texto != ""
	case "id":
		return p.ID != ""
	case "alfabeto":
		return p.Alfabeto != ""
	case "bytes":
		return p.Bytes != 0
	case "caracteres":
		return p.Caracteres != 0
	case "generar":
		return p.Generar
	case "sitios":
		return p.Sitios != nil
	case "etiquetas":
		return p.Etiquetas != nil
	default:
		_, hay := p.Campos[campo]
		return hay
	}
}

// **Ninguna descripción lleva basura, y las que escriben dicen lo de la bóveda.**
//
// Lo primero no es paranoia: al añadir la frase de la bóveda, un script dejó escrito
// literalmente `"' + aviso + '"` dentro de la descripción de `esfinge_editar`. **Compila**
// —es una cadena válida— y lo habría leído el modelo. Se vio mirando el fichero, no
// ejecutando nada.
//
// Y lo segundo es lo que esa frase protege: un agente al que le dicen «guarda esto en el
// proyecto de tal cliente» escribe en la bóveda que esté abierta, que puede ser otra, y
// nadie se entera hasta que alguien busca esa cuenta donde no está.
func TestLasDescripcionesSeLeenYDicenLoDeLaBoveda(t *testing.T) {
	for _, h := range LasHerramientas {
		if h.Descripcion == "" {
			t.Errorf("%s no tiene descripción: el modelo elige por ella", h.Nombre)
			continue
		}
		// Restos de plantilla o de interpolación a medias.
		for _, basura := range []string{"' + ", " + '", "%!", "%s", "{{", "<nil>"} {
			if strings.Contains(h.Descripcion, basura) {
				t.Errorf("%s lleva %q en su descripción: %s", h.Nombre, basura, h.Descripcion)
			}
		}
		if !h.Escribe {
			continue
		}
		// **Que se puede nombrar la bóveda** (ADR 0055). Sin esto, el agente tiene el
		// campo en el esquema y no sabe que es la respuesta a «guárdalo en Zeri's»:
		// seguiría pidiéndole a la persona que conmute, que es justo lo que molestaba.
		if !strings.Contains(h.Descripcion, "`boveda`") {
			t.Errorf("%s escribe y no dice que la bóveda se puede nombrar: el campo existe y "+
				"el modelo no se entera, así que sigue pidiendo que conmutes a mano", h.Nombre)
		}
		// **Y que se mire siempre, no solo al nombrar otra bóveda.** Ésa fue la primera
		// versión de la frase y no servía: el caso que duele es el que **no** la nombra
		// —tienes un proyecto abierto, pides guardar algo sin pensar en dónde—, y ahí el
		// agente no miraba y solo podía contarlo **después de escribir**.
		// **Y que sin nombre se mire siempre.** Ésta es la mitad que la ADR 0055 no
		// cambia: nombrar la bóveda arregla el caso en que alguien la nombra, y el que
		// duele es el otro —tienes un proyecto abierto, pides guardar algo sin pensar en
		// dónde—. Ahí el agente tiene que mirar y preguntar, como desde la 2.52.0.
		//
		// **Mover dice lo suyo, y no es la misma frase**, que es lo que se vio al
		// escribirlo: el destino de un movimiento va siempre nombrado —es obligatorio—,
		// así que «antes de escribir» no es su caso. El suyo es el **origen**, que sí
		// puede ir sin nombrar: con un proyecto abierto, mover «la cuenta de Brevo» se
		// llevaría la del proyecto y no la tuya. Pedirle la frase de los demás habría
		// sido meter una palabra para que el test callara.
		frase := "antes de escribir, siempre"
		if h.Verbo == QueMover {
			frase = "antes de mover, siempre"
		}
		if !strings.Contains(h.Descripcion, frase) {
			t.Errorf("%s no pide preguntar %q: el caso que de verdad duele es el que no "+
				"nombra la bóveda, y ahí hace falta", h.Nombre, frase)
		}
	}
}
