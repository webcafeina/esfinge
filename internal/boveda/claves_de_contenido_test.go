package boveda

import (
	"encoding/json"
	"strings"
	"testing"
)

// La cadena se escribe **en una sola forma posible**: sin esto, el mismo contenido daría
// contenedores distintos en Go y en el espejo de TypeScript, cada equipo vería lo del
// otro como un cambio y las dos bóvedas se pasarían la una a la otra sin fin. Es el
// fallo que la ADR 0038 vino a cerrar, aquí un nivel más abajo.
func TestLaCadenaSeEscribeSiempreIgual(t *testing.T) {
	a := cadenaDeClaves{
		Esfinge: marcaDeLaCadena, Version: VersionDeLaCadena, Actual: "bbbb",
		Claves: []ClaveDeContenido{
			{ID: "bbbb", N: 2, Clave: "k2", Creada: "2026-11-09T10:00:00Z"},
			{ID: "aaaa", N: 1, Clave: "k1", Creada: "2026-11-01T10:00:00Z"},
		},
	}
	// La misma cadena, al revés, y con dos de la misma época para forzar el desempate.
	b := cadenaDeClaves{
		Esfinge: marcaDeLaCadena, Version: VersionDeLaCadena, Actual: "bbbb",
		Claves: []ClaveDeContenido{
			{ID: "aaaa", N: 1, Clave: "k1", Creada: "2026-11-01T10:00:00Z"},
			{ID: "bbbb", N: 2, Clave: "k2", Creada: "2026-11-09T10:00:00Z"},
		},
	}
	ba, err := a.bytes()
	if err != nil {
		t.Fatal(err)
	}
	bb, err := b.bytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(ba) != string(bb) {
		t.Errorf("la misma cadena da bytes distintos según el orden en que se montó:\n%s\n%s", ba, bb)
	}
	// Y el orden es por (n, id), no el de inserción. **Se mira dentro de `claves`**: el
	// identificador de la época de ahora sale antes, en `actual`, y buscarlo en toda la
	// cadena hace fallar la aserción por el motivo equivocado.
	lista := string(ba)[strings.Index(string(ba), `"claves"`):]
	if i, j := strings.Index(lista, `"aaaa"`), strings.Index(lista, `"bbbb"`); i < 0 || j < 0 || i > j {
		t.Errorf("la cadena no sale ordenada por época:\n%s", lista)
	}
}

// Dos claves nuevas **nunca chocan**, y esto es lo que un contador no daría: dos equipos
// del mismo dueño rotando a la vez desde la misma época crearían los dos «la 3», y al
// unir las cadenas una taparía a la otra.
func TestDosClavesNuevasNoChocan(t *testing.T) {
	c, err := nuevaCadena(PorMigracion)
	if err != nil {
		t.Fatal(err)
	}
	// Dos ramas desde el mismo punto, como dos equipos que rotan a la vez.
	uno, dos := c, c
	if err := uno.poner(PorRevocacion); err != nil {
		t.Fatal(err)
	}
	if err := dos.poner(PorRevocacion); err != nil {
		t.Fatal(err)
	}
	if uno.Actual == dos.Actual {
		t.Fatal("dos rotaciones a la vez han dado la misma época: al unir, una taparía a la otra")
	}
	// El número sí coincide —y es correcto que coincida: solo ordena—, lo que no puede
	// coincidir es el identificador.
	if uno.Claves[len(uno.Claves)-1].N != dos.Claves[len(dos.Claves)-1].N {
		t.Error("las dos ramas no están a la misma altura, así que esta prueba no comprueba lo que dice")
	}
	if kUno, _ := uno.clave(uno.Actual); string(kUno) == "" {
		t.Error("la clave nueva sale vacía")
	}
}

// La cadena conserva **todas** las claves, que es lo que hace que un equipo que estuvo
// desconectado pueda abrir su propio fichero al fundir.
func TestLaCadenaNoTiraNingunaClave(t *testing.T) {
	c, err := nuevaCadena(PorMigracion)
	if err != nil {
		t.Fatal(err)
	}
	primera := c.Actual
	for range 3 {
		if err := c.poner(PorRevocacion); err != nil {
			t.Fatal(err)
		}
	}
	if len(c.Claves) != 4 {
		t.Fatalf("quiero 4 claves, tengo %d", len(c.Claves))
	}
	if _, hay := c.clave(primera); !hay {
		t.Error("la primera clave ya no está: un fichero de esa época no se podría abrir")
	}
	if c.Actual == primera {
		t.Error("rotar no ha cambiado la época de ahora")
	}
	// Y el orden de probar empieza por la más nueva.
	if orden := c.deLaMasNuevaALaMasVieja(); orden[0].ID != c.Actual {
		t.Errorf("probar no empieza por la época de ahora: %s", orden[0].ID)
	}
}

// Una ranura de acceso vieja lleva los 43 caracteres de la clave de bóveda; una nueva
// lleva la cadena. **Lo dice el claro**, no el campo `codificacion` del JSON de fuera,
// que cualquiera que pueda escribir el fichero cambia.
func TestEsCadenaDistingueUnaRanuraVieja(t *testing.T) {
	c, err := nuevaCadena(PorMigracion)
	if err != nil {
		t.Fatal(err)
	}
	comoCadena, err := c.bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !esCadena(comoCadena) {
		t.Error("una cadena no se reconoce como cadena")
	}
	// Una clave de bóveda de verdad: 43 caracteres de base64url.
	if esCadena([]byte("Zm9vYmFyZm9vYmFyZm9vYmFyZm9vYmFyZm9vYmFyMDA")) {
		t.Error("una clave de bóveda se ha tomado por una cadena")
	}
	if esCadena(nil) || esCadena([]byte("")) {
		t.Error("lo vacío no es una cadena")
	}
}

// Lo que no se entiende no se toca, y lo que miente no se abre.
func TestUnaCadenaRotaNoSeLee(t *testing.T) {
	buena, err := nuevaCadena(PorMigracion)
	if err != nil {
		t.Fatal(err)
	}
	crudo, err := buena.bytes()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := leerCadena(crudo); err != nil {
		t.Fatalf("una cadena buena no se lee: %v", err)
	}

	for _, caso := range []struct {
		nombre string
		toca   func(*cadenaDeClaves)
		quiero error
	}{
		{"de una versión más nueva", func(c *cadenaDeClaves) { c.Version = VersionDeLaCadena + 1 }, ErrCadenaNueva},
		{"sin la marca", func(c *cadenaDeClaves) { c.Esfinge = "otra cosa" }, ErrManipulada},
		{"vacía", func(c *cadenaDeClaves) { c.Claves = nil }, ErrManipulada},
		{"que dice una época que no lleva", func(c *cadenaDeClaves) { c.Actual = "0000000000000000" }, ErrManipulada},
	} {
		c := buena
		c.Claves = append([]ClaveDeContenido(nil), buena.Claves...)
		caso.toca(&c)
		crudo, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		_, err = leerCadena(crudo)
		if err == nil {
			t.Errorf("una cadena %s se ha leído como buena", caso.nombre)
			continue
		}
		if !strings.Contains(err.Error(), caso.quiero.Error()) {
			t.Errorf("una cadena %s da %v, quiero %v", caso.nombre, err, caso.quiero)
		}
	}
}
