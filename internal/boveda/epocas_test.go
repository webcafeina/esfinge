package boveda

// La fusión con dos épocas (ADR 0059).
//
// Es la parte de la clave de contenido donde un error no se ve: se funde, se guarda, y lo
// que queda es un fichero que no vuelve a abrirse —o que no abre **la persona a la que se
// le dio acceso**, que es peor porque aquí nadie se entera—. Por eso estas pruebas
// terminan todas abriendo el fichero de verdad y leyendo lo de dentro, y no comparando
// listas de sobres.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcafeina/esfinge/internal/cripto"
)

const idDeAna = "1111222233334444"

// rotarAMano hace lo que hará `RotarClaveDeContenido` en la publicación B: una clave
// nueva en la cadena, el cuerpo recifrado con ella, el sobre `contenido` resellado con la
// clave de bóveda y cada ranura de acceso **viva** resellada hacia su identidad.
//
// Está aquí porque hasta esa publicación **no hay otra forma de tener dos épocas**, y las
// dos épocas son justo lo que hay que probar: nada escribe todavía el formato 2. Cuando
// exista la de verdad, esto se tira y se llama a aquélla — un ayudante que hace lo que
// hace producción no prueba producción, y esa trampa aquí ya ha costado.
func rotarAMano(t *testing.T, b *Boveda, para map[string]Identidad) cadenaDeClaves {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()

	c := b.claves
	if err := c.poner(PorRevocacion); err != nil {
		t.Fatal(err)
	}
	crudo, err := c.bytes()
	if err != nil {
		t.Fatal(err)
	}
	sobreContenido, err := cripto.SellarTexto(crudo, b.llave, cripto.PerfilLlave)
	if err != nil {
		t.Fatal(err)
	}
	cuando := ahora().UTC().Format(time.RFC3339)

	nuevo := sobre{Tipo: RanuraDeContenido, Creado: cuando, Contenedor: sobreContenido}
	if i := b.ranura(RanuraDeContenido); i >= 0 {
		b.doc.Sobres[i] = nuevo
	} else {
		b.doc.Sobres = append(b.doc.Sobres, nuevo)
	}
	for i, s := range b.doc.Sobres {
		id := IDDeAcceso(s.Tipo)
		if id == "" || esRetirada(s) {
			continue
		}
		quien, hay := para[id]
		if !hay {
			t.Fatalf("para rotar hace falta la identidad de %q, y no se le ha dado", id)
		}
		sellada, err := SellarHaciaIdentidad(crudo, quien)
		if err != nil {
			t.Fatal(err)
		}
		b.doc.Sobres[i] = sobre{Tipo: s.Tipo, Creado: cuando,
			Contenedor: sellada, Codificacion: CodificacionSellada}
	}
	if err := b.ponerEpoca(c, c.Actual); err != nil {
		t.Fatal(err)
	}
	b.cuerpoSucio = true
	if err := b.guardar(); err != nil {
		t.Fatal(err)
	}
	return c
}

// copiaDelFichero es el otro equipo: el mismo fichero en otra ruta.
func copiaDelFichero(t *testing.T, de string) string {
	t.Helper()
	datos, err := os.ReadFile(de)
	if err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(t.TempDir(), "elOtroEquipo.esfinge")
	if err := os.WriteFile(a, datos, 0o600); err != nil {
		t.Fatal(err)
	}
	return a
}

// unProyectoConCadenaYUnTitular deja una bóveda de proyecto del formato 2 —cadena de
// claves, época declarada— con una entrada y con Ana dentro.
func unProyectoConCadenaYUnTitular(t *testing.T) (ruta string, llave []byte, ana *Boveda, idAna Identidad) {
	t.Helper()
	_, ruta, llave = personalYProyecto(t)
	ana, idAna = unaPersonal(t, "ana")
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Cloudflare", Secreto: "el testigo"}); err != nil {
		t.Fatal(err)
	}
	if err := p.PonerAcceso(idDeAna, idAna); err != nil {
		t.Fatal(err)
	}
	conClavesDeContenido(t, p, idDeAna, idAna)
	p.Cerrar()
	return ruta, llave, ana, idAna
}

// **El dueño funde una bóveda que el otro equipo ha vuelto a cifrar.**
//
// Su cadena no tiene esa clave, así que la saca del propio documento con la clave de
// bóveda; y lo que queda tiene que abrirse después, que es lo único que de verdad dice si
// la época y la clave del cuerpo han quedado de acuerdo.
func TestFundirConDosEpocasSeQuedaConLaClaveQueGana(t *testing.T) {
	// **Con el reloj en la mano de principio a fin, y no es un detalle.** Las fechas van
	// a segundos, así que si los dos sobres `contenido` nacen en el mismo, quién gana la
	// época lo decide el desempate del contenedor, o sea el azar del cifrado. Y **el
	// escenario entero tiene que ir con el mismo reloj**: fabricando el proyecto con el de
	// verdad y rotando con el de mentira, quién gana depende de la hora del día a la que
	// se corra la prueba — pasaba por la mañana y fallaba a media mañana.
	reloj := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

	var c cadenaDeClaves
	var remoto []byte
	conReloj(t, func() time.Time { return reloj }, func() {
		ruta, llave, _, idAna := unProyectoConCadenaYUnTitular(t)
		// El otro equipo: la misma bóveda, una entrada suya, y rota.
		rutaOtro := copiaDelFichero(t, ruta)
		otro, err := AbrirProyecto(rutaOtro, llave)
		if err != nil {
			t.Fatal(err)
		}
		if err := otro.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo del otro equipo"}); err != nil {
			t.Fatal(err)
		}
		reloj = reloj.Add(10 * time.Second)
		c = rotarAMano(t, otro, map[string]Identidad{idDeAna: idAna})
		remoto, _, err = otro.PrepararSubida(1)
		if err != nil {
			t.Fatal(err)
		}
		otro.Cerrar()

		// Aquí, algo que no está allí, y se funde.
		reloj = reloj.Add(10 * time.Second)
		a, err := AbrirProyecto(ruta, llave)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo de este equipo"}); err != nil {
			t.Fatal(err)
		}
		f, err := a.Fundir(remoto, 1, nil, OpcionesDeFusion{})
		if err != nil {
			t.Fatalf("fundir con el otro lado en otra época: %v", err)
		}
		if !f.Cambio {
			t.Error("la fusión dice que no ha cambiado nada, y el otro lado había rotado")
		}
		// **Y se queda con la de allí, que es la que hace que rotar signifique algo**:
		// con la de aquí, el cuerpo nuevo seguiría cifrado con la clave que tiene quien
		// perdió el acceso.
		if a.doc.Epoca != c.Actual {
			t.Errorf("el cuerpo se queda en la época %q y la que gana es %q", a.doc.Epoca, c.Actual)
		}
		a.Cerrar()

		// **Y se vuelve a abrir**, que es lo que se paga aquí: una época declarada sin la
		// clave con la que está cifrado el cuerpo no da error al guardar, da un fichero
		// perdido.
		vuelta, err := AbrirProyecto(ruta, llave)
		if err != nil {
			t.Fatalf("tras fundir dos épocas, la bóveda no se abre: %v", err)
		}
		defer vuelta.Cerrar()
		for _, q := range []string{"Cloudflare", "Lo del otro equipo", "Lo de este equipo"} {
			if e := vuelta.Buscar(q); len(e) != 1 {
				t.Errorf("tras fundir no está %q", q)
			}
		}
	})
	_ = c
}

// **Las ranuras de acceso salen del lado que pone la época, aunque la de aquí sea más
// nueva.**
//
// Es el caso que obliga a resolverlas en bloque y el que no se ve de ninguna otra forma:
// mientras un equipo rotaba, en este otro se le volvió a dar el acceso a Ana. Su ranura
// de aquí es **más reciente**, así que las reglas de siempre se la llevarían — y lleva
// dentro la cadena de antes, con la que el cuerpo nuevo no se descifra. Ana abriría su
// sobre, sacaría una cadena sin la época del cuerpo y **no abriría su propia bóveda**.
func TestAlFundirDosEpocasLasRanurasDeAccesoVanConLaClave(t *testing.T) {
	// El escenario entero con el mismo reloj: ver la prueba de arriba.
	reloj := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	var ruta string
	var ana *Boveda
	conReloj(t, func() time.Time { return reloj }, func() {
		var llave []byte
		var idAna Identidad
		ruta, llave, ana, idAna = unProyectoConCadenaYUnTitular(t)
		rutaOtro := copiaDelFichero(t, ruta)
		otro, err := AbrirProyecto(rutaOtro, llave)
		if err != nil {
			t.Fatal(err)
		}
		// Allí se rota, resellando la ranura de Ana con la cadena nueva.
		reloj = reloj.Add(10 * time.Second)
		rotarAMano(t, otro, map[string]Identidad{idDeAna: idAna})
		remoto, _, err := otro.PrepararSubida(1)
		if err != nil {
			t.Fatal(err)
		}
		otro.Cerrar()

		// Y aquí, **después**, se le vuelve a dar el acceso: su ranura de este lado es
		// más nueva y lleva la cadena de antes.
		reloj = reloj.Add(10 * time.Second)
		a, err := AbrirProyecto(ruta, llave)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.PonerAcceso(idDeAna, idAna); err != nil {
			t.Fatal(err)
		}
		if _, err := a.Fundir(remoto, 1, nil, OpcionesDeFusion{}); err != nil {
			t.Fatalf("fundir: %v", err)
		}
		a.Cerrar()
	})

	deAna, err := AbrirCompartida(ruta, idDeAna, ana)
	if err != nil {
		t.Fatalf("tras la fusión, la persona con acceso no abre la bóveda: %v", err)
	}
	defer deAna.Cerrar()
	if e := deAna.Buscar("Cloudflare"); len(e) != 1 {
		t.Errorf("abre y no ve las entradas: %d", len(e))
	}
}

// **Y una lápida no la resucita el lado que pone la época.**
//
// La otra mitad de la de arriba, y la que impide arreglarla de la forma barata: si los
// sobres con claves se cogieran del lado que gana **sin mirar nada más**, quitarle el
// acceso a alguien aquí y rotar en otro equipo se lo devolvería, que es exactamente lo
// que la lápida de la ADR 0052 existe para que no pase.
func TestAlFundirDosEpocasUnaLapidaNoRevive(t *testing.T) {
	// El escenario entero con el mismo reloj: ver la primera prueba de este fichero.
	reloj := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	var ruta string
	var ana *Boveda
	conReloj(t, func() time.Time { return reloj }, func() {
		var llave []byte
		var idAna Identidad
		ruta, llave, ana, idAna = unProyectoConCadenaYUnTitular(t)
		rutaOtro := copiaDelFichero(t, ruta)
		otro, err := AbrirProyecto(rutaOtro, llave)
		if err != nil {
			t.Fatal(err)
		}
		reloj = reloj.Add(10 * time.Second)
		rotarAMano(t, otro, map[string]Identidad{idDeAna: idAna})
		remoto, _, err := otro.PrepararSubida(1)
		if err != nil {
			t.Fatal(err)
		}
		otro.Cerrar()

		// Aquí se le quita el acceso **después** de que allí se rotara.
		reloj = reloj.Add(10 * time.Second)
		a, err := AbrirProyecto(ruta, llave)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.RetirarAcceso(idDeAna); err != nil {
			t.Fatal(err)
		}
		if _, err := a.Fundir(remoto, 1, nil, OpcionesDeFusion{}); err != nil {
			t.Fatalf("fundir: %v", err)
		}
		if hay := a.Accesos(); len(hay) != 0 {
			t.Errorf("la fusión ha devuelto el acceso que se acababa de quitar: %v", hay)
		}
		a.Cerrar()
	})

	if deAna, err := AbrirCompartida(ruta, idDeAna, ana); !errors.Is(err, ErrSinAcceso) {
		if err == nil {
			deAna.Cerrar()
		}
		t.Errorf("quien perdió el acceso vuelve a abrir la bóveda: %v", err)
	}
}

// **A un titular que no puede llegar a la clave nueva se le dice qué hacer.**
//
// Su cadena se la sellaron hacia su identidad, y recogerla pasa por su bóveda personal,
// que al fundir no está delante. Lo que no puede salir es `ErrManipulada` —«alguien ha
// editado el fichero a mano»—, que manda a mirar donde no hay nada roto.
func TestUnTitularSinLaEpocaNuevaNoDiceQueLaBovedaEsteRota(t *testing.T) {
	ruta, llave, ana, idAna := unProyectoConCadenaYUnTitular(t)

	rutaOtro := copiaDelFichero(t, ruta)
	otro, err := AbrirProyecto(rutaOtro, llave)
	if err != nil {
		t.Fatal(err)
	}
	rotarAMano(t, otro, map[string]Identidad{idDeAna: idAna})
	remoto, _, err := otro.PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}
	otro.Cerrar()

	deAna, err := AbrirCompartida(ruta, idDeAna, ana)
	if err != nil {
		t.Fatal(err)
	}
	defer deAna.Cerrar()
	_, err = deAna.Fundir(remoto, 1, nil, OpcionesDeFusion{})
	if !errors.Is(err, ErrEpocaQueNoSeTiene) {
		t.Fatalf("a un titular con la época vieja se le dice %v", err)
	}
	// Y lo suyo sigue abierto y legible: lo que no se ha podido es fundir.
	if e := deAna.Buscar("Cloudflare"); len(e) != 1 {
		t.Error("la fusión fallida se ha llevado lo que había en memoria")
	}
}

// **Y un remoto de un formato que se sabe leer pero no escribir no se funde.**
//
// Es exactamente lo que existe en la publicación A de la ADR 0059: se lee el 2 y se
// escribe el 1. Sin esta guarda, fundir —que acaba guardando— escribiría un fichero
// declarando el formato de aquí con la época y la cadena del otro lado dentro: una bóveda
// que no vuelve a abrir nadie. Tampoco se podía probar hasta que los dos números
// dejaran de valer lo mismo.
func TestFundirNoSeFiaDeUnRemotoDeUnFormatoQueNoSeEscribe(t *testing.T) {
	if FormatoQueSeLee <= FormatoQueSeEscribe {
		t.Skip("los dos números valen lo mismo: el caso no existe en esta versión")
	}
	b, _, _ := nueva(t)
	if err := b.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo de aquí"}); err != nil {
		t.Fatal(err)
	}
	subida, _, err := b.PrepararSubida(1)
	if err != nil {
		t.Fatal(err)
	}
	// Como si lo hubiera escrito una versión que ya sabe escribir el formato siguiente.
	// **El formato no va dentro del sello**, así que esto es lo que puede llegar de
	// verdad del servidor, no un fichero imposible.
	remoto := enElFormato(t, subida, FormatoQueSeEscribe+1)
	if _, err := b.Fundir(remoto, 1, nil, OpcionesDeFusion{}); !errors.Is(err, ErrFormatoNuevo) {
		t.Fatalf("fundir un remoto que esta versión no sabe escribir dice %v", err)
	}
}

// enElFormato reescribe el número de formato de un documento, dejando todo lo demás.
func enElFormato(t *testing.T, datos []byte, formato int) []byte {
	t.Helper()
	var doc documento
	if err := json.Unmarshal(datos, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Formato = formato
	fuera, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return fuera
}
