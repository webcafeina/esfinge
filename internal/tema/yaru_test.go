package tema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLosIconosDeYaruNoSeHanAdaptado compara los trazados que Esfinge pinta con
// los del SVG que publicó Yaru.
//
// # Por qué esta prueba existe
//
// No es una comprobación de aspecto: **es lo que respalda una afirmación de
// licencia**. Yaru es CC BY-SA 4.0, que obliga a publicar **las adaptaciones**
// con la misma licencia; Esfinge no es código abierto, así que lo que permite
// llevar estos dibujos dentro es que **no se adapte ninguno**. Eso está escrito
// en `TERCEROS.md`, en la cabecera de `frontend/src/iconos-yaru.ts` y en la de
// `herramientas/iconos/yaru.py` — y una afirmación que no comprueba nadie es una
// esperanza.
//
// # Qué compara, y por qué no reconstruye el árbol
//
// Exige que **cada `d` del módulo aparezca, carácter a carácter, en el SVG de
// origen**, y que el `viewBox` sea el mismo. Eso es exactamente lo que dice la
// promesa: ningún trazado se ha juntado con otro, recortado ni reescalado.
//
// Lo que **no** hace es volver a recorrer el SVG con las reglas del cosechador y
// comparar el resultado: eso sería reimplementar producción en el ayudante, y
// entonces los dos lados se equivocarían igual. Aquí los ficheros de
// `testdata/yaru` son el original tal como vino, y el módulo es lo que se
// publica; que una `d` esté en los dos no se puede fingir.
//
// # Y lo que la hizo falta
//
// La prueba de interfaz que hay en `gnome.spec.ts` compara **el DOM con el
// módulo**, que es otra cosa: dice que lo que se pinta es lo que el módulo
// guarda —y por eso caza un `transform` perdido— pero **no dice de dónde salió**.
// Mutando un trazado del módulo, aquélla seguía en verde porque los dos lados
// salían del mismo fichero. Esta se pone roja.
func TestLosIconosDeYaruNoSeHanAdaptado(t *testing.T) {
	iconos := leerIconosDeYaru(t)
	if len(iconos) == 0 {
		t.Fatal("no se leyó ningún icono de frontend/src/iconos-yaru.ts")
	}

	origenes, err := filepath.Glob(filepath.Join("testdata", "yaru", "*.svg"))
	if err != nil || len(origenes) == 0 {
		t.Fatalf("no están los SVG de origen en testdata/yaru: %v", err)
	}
	if len(origenes) != len(iconos) {
		t.Errorf("hay %d iconos en el módulo y %d SVG guardados: se cosechó sin guardar el origen, "+
			"o quedó un origen de un icono que ya no se usa. Se arregla volviendo a correr "+
			"«python3 herramientas/iconos/yaru.py».", len(iconos), len(origenes))
	}

	for nombre, icono := range iconos {
		ruta := filepath.Join("testdata", "yaru", nombre+".svg")
		fuente, err := os.ReadFile(ruta)
		if err != nil {
			t.Errorf("%s: no está su SVG de origen (%s). Vuelve a cosechar con "+
				"«python3 herramientas/iconos/yaru.py».", nombre, ruta)
			continue
		}
		svg := string(fuente)

		if !strings.Contains(svg, `viewBox="`+icono.Caja+`"`) {
			t.Errorf("%s: el módulo dice viewBox=%q y el origen no lo trae. Reescalar la caja "+
				"**es adaptar**, y eso es lo que ni la licencia ni este proyecto permiten.",
				nombre, icono.Caja)
		}

		for _, d := range trazadosDe(icono.Nodos) {
			if !strings.Contains(svg, d) {
				corto := d
				if len(corto) > 60 {
					corto = corto[:60] + "…"
				}
				t.Errorf("%s: un trazado del módulo no está en el SVG de origen:\n  %s\n"+
					"Eso significa que se ha tocado el dibujo —juntado, recortado o reescalado—, "+
					"y entonces **es una obra derivada**: Yaru es CC BY-SA 4.0 y habría que "+
					"publicarla con esa licencia. Lo que entra, entra como está.", nombre, corto)
			}
		}
	}
}

type nodoYaru struct {
	T string            `json:"t"`
	A map[string]string `json:"a"`
	H []nodoYaru        `json:"h"`
}

type iconoYaru struct {
	Caja  string     `json:"caja"`
	Nodos []nodoYaru `json:"nodos"`
	De    string     `json:"de"`
}

// trazadosDe saca las `d` de un árbol, en el orden en que se pintan.
func trazadosDe(nodos []nodoYaru) []string {
	var ds []string
	for _, n := range nodos {
		if n.T == "path" {
			if d, hay := n.A["d"]; hay {
				ds = append(ds, d)
			}
		}
		ds = append(ds, trazadosDe(n.H)...)
	}
	return ds
}

// leerIconosDeYaru saca el objeto del módulo de TypeScript.
//
// Se lee el fuente, como hacen las otras guardas de esta casa que cruzan los dos
// lenguajes: el fichero lo escribe un generador, así que el JSON de dentro es
// JSON de verdad y no hay que interpretar TypeScript para sacarlo.
func leerIconosDeYaru(t *testing.T) map[string]iconoYaru {
	t.Helper()
	ruta := filepath.Join("..", "..", "frontend", "src", "iconos-yaru.ts")
	fuente, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", ruta, err)
	}

	const marca = "IconoYaru> = "
	i := strings.Index(string(fuente), marca)
	if i < 0 {
		t.Fatalf("%s no declara ICONOS_YARU como se espera", ruta)
	}
	cuerpo := strings.TrimSpace(string(fuente)[i+len(marca):])
	cuerpo = strings.TrimSuffix(strings.TrimSpace(cuerpo), ";")

	var iconos map[string]iconoYaru
	if err := json.Unmarshal([]byte(cuerpo), &iconos); err != nil {
		t.Fatalf("el cuerpo de ICONOS_YARU no es JSON: %v", err)
	}
	return iconos
}
