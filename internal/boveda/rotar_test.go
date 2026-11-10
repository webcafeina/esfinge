package boveda

// Rotar las claves (ADR 0059, publicación B).
//
// La prueba que paga esta deuda es la primera: **a quien tuvo acceso antes y se llevó la
// clave de bóveda, rotar le corta**. Lo demás existe para que al cortarle no se rompa
// nada: que quien sigue teniendo acceso siga abriendo, que un equipo que viene por detrás
// pueda fundir, y que una rotación a medias no deje un fichero que no abre nadie.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// unProyectoCompartido deja una bóveda de proyecto con una entrada y con Ana dentro,
// **con una ranura de las de antes de la 0059**: la que lleva la clave de bóveda.
func unProyectoCompartido(t *testing.T) (ruta string, llave []byte, ana *Boveda, idAna Identidad, laQueSeLlevo []byte) {
	t.Helper()
	_, ruta, llave = personalYProyecto(t)
	ana, idAna = unaPersonal(t, "ana")
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Cerrar()
	if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Cloudflare", Secreto: "el testigo"}); err != nil {
		t.Fatal(err)
	}
	if err := p.PonerAcceso(idDeAna, idAna); err != nil {
		t.Fatal(err)
	}
	if err := p.PonerTitular(Titular{ID: idDeAna, Correo: "ana@ejemplo.com", Permiso: "editar",
		Huella: HuellaDeIdentidad(idAna.Suite, idAna.Cifrado, idAna.Firma), Desde: "2026-11-10T09:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	// Lo que Ana se lleva al abrir una ranura de las de antes: la clave de bóveda.
	p.mu.Lock()
	laQueSeLlevo = append([]byte(nil), p.llave...)
	p.mu.Unlock()
	return ruta, llave, ana, idAna, laQueSeLlevo
}

func identidadesDe(id Identidad) map[string]Identidad {
	return map[string]Identidad{idDeAna: id}
}

// **La prueba que paga la deuda.**
//
// Quien tuvo acceso antes de la 0059 se llevó la clave de bóveda, y con ella se
// desenvolvía el sobre `contenido`, o sea la cadena, o sea todas las claves de contenido
// que vinieran después. Comprobado el 2026-11-10: con esa clave leía lo escrito **después**
// de rotar. Por eso rotar cambia **las dos** claves.
func TestRotarCortaAQuienTuvoAccesoAntes(t *testing.T) {
	ruta, llave, _, idAna, laQueSeLlevo := unProyectoCompartido(t)

	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.RotarLasClaves(identidadesDe(idAna), PorLaMano); err != nil {
		t.Fatalf("rotar: %v", err)
	}
	if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo de después de rotar", Secreto: "el nuevo"}); err != nil {
		t.Fatal(err)
	}
	p.Cerrar()

	if otra, err := AbrirConLaClave(ruta, laQueSeLlevo); err == nil {
		e := otra.Buscar("")
		otra.Cerrar()
		t.Fatalf("con la clave de bóveda de antes se sigue abriendo la bóveda, y se leen %d entradas: rotar no ha cortado a nadie", len(e))
	}
}

// Y la otra mitad: a quien sigue teniendo acceso, rotar **no** le echa.
func TestRotarDejaDentroAQuienSigueTeniendoAcceso(t *testing.T) {
	ruta, llave, ana, idAna, _ := unProyectoCompartido(t)

	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.RotarLasClaves(identidadesDe(idAna), PorLaMano); err != nil {
		t.Fatal(err)
	}
	if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo de después de rotar"}); err != nil {
		t.Fatal(err)
	}
	p.Cerrar()

	deAna, err := AbrirCompartida(ruta, idDeAna, ana)
	if err != nil {
		t.Fatalf("quien sigue teniendo acceso no abre tras rotar: %v", err)
	}
	defer deAna.Cerrar()
	for _, q := range []string{"Cloudflare", "Lo de después de rotar"} {
		if e := deAna.Buscar(q); len(e) != 1 {
			t.Errorf("tras rotar, quien tiene acceso no ve %q", q)
		}
	}
	// Y con la ranura nueva **ya no se lleva la clave de bóveda**, que es el otro medio
	// objetivo de la 0059.
	if deAna.LlaveParaProyectos() != nil {
		t.Error("la ranura reseilada sigue entregando la clave de bóveda")
	}
}

// Y el dueño sigue abriendo con su personal, que es lo que la ranura principal reenvuelta
// tiene que garantizar. Sin eso, rotar dejaría la bóveda sin puerta.
func TestRotarNoPierdeElCuerpoNiLaPuerta(t *testing.T) {
	ruta, llave, _, idAna, _ := unProyectoCompartido(t)

	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	antes := p.Cuantas()
	if err := p.RotarLasClaves(identidadesDe(idAna), PorRevocacion); err != nil {
		t.Fatal(err)
	}
	p.Cerrar()

	vuelta, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatalf("tras rotar, el dueño no abre su propio proyecto: %v", err)
	}
	defer vuelta.Cerrar()
	if vuelta.Cuantas() != antes {
		t.Errorf("tras rotar hay %d entradas y había %d", vuelta.Cuantas(), antes)
	}
	if e := vuelta.Buscar("Cloudflare"); len(e) != 1 {
		t.Error("tras rotar no está lo que había")
	}
	vuelta.mu.Lock()
	formato, epoca := vuelta.doc.Formato, vuelta.doc.Epoca
	vuelta.mu.Unlock()
	if formato != FormatoConClavesDeContenido {
		t.Errorf("tras rotar el documento dice que es del formato %d", formato)
	}
	if epoca == "" {
		t.Error("tras rotar no declara época")
	}
}

// Una lápida no se resella: eso sería devolverle el acceso a quien lo perdió.
func TestRotarNoResucitaUnAccesoRetirado(t *testing.T) {
	ruta, llave, ana, idAna, _ := unProyectoCompartido(t)

	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.RetirarAcceso(idDeAna); err != nil {
		t.Fatal(err)
	}
	// Y rotar **sin** pasar su identidad: si la lápida se resellara, faltaría y esto
	// fallaría; que pase es parte de lo que se comprueba.
	if err := p.RotarLasClaves(nil, PorRevocacion); err != nil {
		t.Fatalf("rotar tras quitar el único acceso: %v", err)
	}
	if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo de después"}); err != nil {
		t.Fatal(err)
	}
	p.Cerrar()

	if deAna, err := AbrirCompartida(ruta, idDeAna, ana); !errors.Is(err, ErrSinAcceso) {
		if err == nil {
			deAna.Cerrar()
		}
		t.Errorf("quien perdió el acceso vuelve a abrir tras rotar: %v", err)
	}
	_ = idAna
}

// **Rotar pide las identidades, y comprueba la huella.**
//
// No están en la bóveda, así que rotar necesita red; y si el servidor diera otras llaves,
// rotar sellaría la bóveda hacia quien él quisiera. Es la regla de `DarAccesoAlServidor`.
func TestRotarExigeLasIdentidadesYLaHuella(t *testing.T) {
	ruta, llave, _, idAna, _ := unProyectoCompartido(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Cerrar()

	if err := p.RotarLasClaves(nil, PorLaMano); err == nil {
		t.Error("rota sin la identidad de quien tiene acceso: su ranura se quedaría sin resellar")
	}
	_, otra := unaPersonal(t, "otra")
	if err := p.RotarLasClaves(map[string]Identidad{idDeAna: otra}, PorLaMano); err == nil {
		t.Error("rota con unas llaves que no son las que se compararon al dar el acceso")
	}
	// Y con las buenas, sí.
	if err := p.RotarLasClaves(identidadesDe(idAna), PorLaMano); err != nil {
		t.Errorf("con la identidad correcta no rota: %v", err)
	}
}

// Quien recibe el acceso no rota: el sobre `contenido` va envuelto con la clave de bóveda,
// así que un titular que rotara dejaría al dueño fuera de su propia bóveda.
func TestUnTitularNoRota(t *testing.T) {
	ruta, llave, ana, idAna, _ := unProyectoCompartido(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.RotarLasClaves(identidadesDe(idAna), PorMigracion); err != nil {
		t.Fatal(err)
	}
	p.Cerrar()

	deAna, err := AbrirCompartida(ruta, idDeAna, ana)
	if err != nil {
		t.Fatal(err)
	}
	defer deAna.Cerrar()
	if err := deAna.RotarLasClaves(identidadesDe(idAna), PorLaMano); !errors.Is(err, ErrSoloElDueno) {
		t.Errorf("un titular rota la clave de la bóveda de otra persona: %v", err)
	}
}

// **Nada se escribe hasta el guardado del final.** Una rotación a medias —la cadena nueva
// con el cuerpo viejo, o la ranura principal reenvuelta y el sobre `contenido` no— deja un
// fichero que no abre nadie.
func TestRotarEsAtomico(t *testing.T) {
	ruta, llave, _, idAna, _ := unProyectoCompartido(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Cerrar()

	// Otro Esfinge toca el fichero: guardar se negará.
	otro, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := otro.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Desde el otro"}); err != nil {
		t.Fatal(err)
	}
	otro.Cerrar()

	if err := p.RotarLasClaves(identidadesDe(idAna), PorLaMano); err == nil {
		t.Fatal("rota aunque el fichero haya cambiado por debajo")
	}
	// Y lo de aquí sigue en pie: se lee y **se sigue guardando con lo de antes**.
	if e := p.Buscar("Cloudflare"); len(e) != 1 {
		t.Error("la rotación fallida se ha llevado lo que había en memoria")
	}
	p.mu.Lock()
	formato := p.doc.Formato
	p.mu.Unlock()
	if formato != FormatoBase {
		t.Errorf("la rotación fallida ha dejado el documento diciendo que es del formato %d", formato)
	}
}

// **Un equipo que viene por detrás funde tras una rotación**, y sin que nadie reabra nada:
// la clave de bóveda de cada documento sale de ese documento, por su ranura principal.
//
// Sin eso, su fichero lleva la clave de antes, la del que baja solo está dentro del que
// baja, y ese equipo **no podría volver a fundir nunca** — reabrir no ayuda, porque lo que
// reabre es su fichero viejo.
func TestUnEquipoQueVieneDeAntesFundeTrasUnaRotacion(t *testing.T) {
	ruta, llave, _, idAna, _ := unProyectoCompartido(t)
	reloj := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

	conReloj(t, func() time.Time { return reloj }, func() {
		// El equipo de atrás: una copia del fichero antes de rotar.
		rutaAtras := filepath.Join(t.TempDir(), "atras.esfinge")
		datos, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(rutaAtras, datos, 0o600); err != nil {
			t.Fatal(err)
		}

		// Aquí se rota y se escribe.
		reloj = reloj.Add(10 * time.Second)
		p, err := AbrirProyecto(ruta, llave)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.RotarLasClaves(identidadesDe(idAna), PorRevocacion); err != nil {
			t.Fatal(err)
		}
		if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo de este equipo"}); err != nil {
			t.Fatal(err)
		}
		remoto, _, err := p.PrepararSubida(1)
		if err != nil {
			t.Fatal(err)
		}
		p.Cerrar()

		// Y el de atrás, que no sabe nada, escribe lo suyo y funde.
		reloj = reloj.Add(10 * time.Second)
		atras, err := AbrirProyecto(rutaAtras, llave)
		if err != nil {
			t.Fatal(err)
		}
		if err := atras.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo del equipo de atrás"}); err != nil {
			t.Fatal(err)
		}
		if _, err := atras.Fundir(remoto, 1, nil, OpcionesDeFusion{}); err != nil {
			t.Fatalf("el equipo que viene de antes no puede fundir tras una rotación: %v", err)
		}
		atras.Cerrar()

		// Y lo que queda se abre y lleva las dos cosas.
		vuelta, err := AbrirProyecto(rutaAtras, llave)
		if err != nil {
			t.Fatalf("tras fundir la rotación, ese equipo no abre su fichero: %v", err)
		}
		defer vuelta.Cerrar()
		for _, q := range []string{"Cloudflare", "Lo de este equipo", "Lo del equipo de atrás"} {
			if e := vuelta.Buscar(q); len(e) != 1 {
				t.Errorf("tras fundir no está %q", q)
			}
		}
	})
}

// **Y quien tiene acceso recoge la cadena nueva por su bóveda personal.**
//
// Su ranura va sellada hacia su identidad, que la fusión no tiene delante, así que lo que
// baja no lo puede leer solo. Reabrir no sirve: lo que reabre es su fichero de aquí, que es
// justo el que tiene la cadena de antes.
func TestUnTitularRecogeLaCadenaYFunde(t *testing.T) {
	ruta, llave, ana, idAna, _ := unProyectoCompartido(t)
	reloj := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

	conReloj(t, func() time.Time { return reloj }, func() {
		// Ana se baja la bóveda tal como está y la abre.
		datos, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		rutaDeAna := filepath.Join(t.TempDir(), "deAna.esfinge")
		if err := os.WriteFile(rutaDeAna, datos, 0o600); err != nil {
			t.Fatal(err)
		}
		deAna, err := AbrirCompartida(rutaDeAna, idDeAna, ana)
		if err != nil {
			t.Fatal(err)
		}
		defer deAna.Cerrar()

		// El dueño rota y sube.
		reloj = reloj.Add(10 * time.Second)
		p, err := AbrirProyecto(ruta, llave)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.RotarLasClaves(identidadesDe(idAna), PorRevocacion); err != nil {
			t.Fatal(err)
		}
		if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Lo del dueño"}); err != nil {
			t.Fatal(err)
		}
		remoto, _, err := p.PrepararSubida(1)
		if err != nil {
			t.Fatal(err)
		}
		p.Cerrar()

		// A Ana no le alcanza su cadena, **y se le dice qué hacer**.
		if _, err := deAna.Fundir(remoto, 1, nil, OpcionesDeFusion{}); !errors.Is(err, ErrEpocaQueNoSeTiene) {
			t.Fatalf("a un titular tras una rotación se le dice %v", err)
		}
		// La recoge por su personal y entonces sí.
		if err := deAna.RecogerCadenaDe(remoto, ana); err != nil {
			t.Fatalf("recoger la cadena nueva: %v", err)
		}
		if _, err := deAna.Fundir(remoto, 1, nil, OpcionesDeFusion{}); err != nil {
			t.Fatalf("con la cadena recogida sigue sin poder fundir: %v", err)
		}
		if e := deAna.Buscar("Lo del dueño"); len(e) != 1 {
			t.Error("tras fundir no está lo que escribió el dueño")
		}
		if e := deAna.Buscar("Cloudflare"); len(e) != 1 {
			t.Error("y se ha perdido lo que ya había")
		}
	})
}

// Entregar una copia **la devuelve al formato de siempre** (ADR 0051): quien la recibe
// puede tener cualquier versión de Esfinge, así que una copia que declarara el formato
// nuevo no la abriría. Cae del diseño —una bóveda nace sin claves de contenido— y por eso
// mismo conviene que esté escrito.
func TestDesprenderVuelveAlFormatoDeSiempre(t *testing.T) {
	ruta, llave, _, idAna, _ := unProyectoCompartido(t)
	p, err := AbrirProyecto(ruta, llave)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.RotarLasClaves(identidadesDe(idAna), PorMigracion); err != nil {
		t.Fatal(err)
	}
	defer p.Cerrar()

	copia, _, err := p.Desprender("una contraseña larga para la copia")
	if err != nil {
		t.Fatal(err)
	}
	copia.mu.Lock()
	formato, epoca := copia.doc.Formato, copia.doc.Epoca
	tieneContenido := copia.ranura(RanuraDeContenido) >= 0
	copia.mu.Unlock()
	if formato != FormatoBase {
		t.Errorf("la copia entregada dice que es del formato %d", formato)
	}
	if epoca != "" {
		t.Errorf("la copia entregada declara época %q", epoca)
	}
	if tieneContenido {
		t.Error("la copia entregada lleva el sobre de las claves de contenido")
	}
}

// **Una bóveda de proyecto del formato 2, congelada** (ADR 0059, y el argumento de la
// [ADR 0022] una planta más arriba).
//
// Lleva dentro lo que no se puede volver a fabricar igual: dos épocas —la de migrar y una
// rotación más—, una ranura de acceso viva, la lápida de uno al que se le quitó, y el
// cuerpo cifrado con la clave de la época de ahora. **No se regenera**: si esta prueba se
// pone roja, o el cambio está mal o toca subir el formato y grabar un vector nuevo al
// lado.
//
// **Y se lee con el reloj de su época**, por lo que ya costó una vez: un dato congelado con
// fechas dentro, leído con el reloj del sistema, caduca solo.
//
// Las dos claves están aquí escritas a propósito: son de un fichero de pruebas, y lo que
// prueban es justo lo que no se puede probar de otra forma — que **la de antes de migrar
// ya no abre nada**.
func TestUnProyectoDelFormato2Congelado(t *testing.T) {
	// La clave de bóveda de la personal con la que se envolvió su ranura principal.
	const principal = "PaeyTzf0AWfdHZ1MywIlMLju5zcRh7s4RQj4GSuE8Mc"
	// Y la clave de bóveda que tenía **antes de la primera rotación**: lo que se habría
	// llevado quien tuviera acceso entonces.
	const laDeAntesDeMigrar = "2-PZXmwMRt7f3HeFmVn9yNVflcqMG0JI6sxSKCLqfv0"

	conReloj(t, func() time.Time { return time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC) }, func() {
		datos, err := os.ReadFile("testdata/proyecto-formato2.esfinge")
		if err != nil {
			t.Fatal(err)
		}
		ruta := filepath.Join(t.TempDir(), "proyecto.esfinge")
		if err := os.WriteFile(ruta, datos, 0o600); err != nil {
			t.Fatal(err)
		}

		p, err := AbrirProyecto(ruta, []byte(principal))
		if err != nil {
			t.Fatalf("una bóveda del formato 2 ya grabada no se abre: %v", err)
		}
		defer p.Cerrar()

		// Lo de dentro, entero.
		cf := buscar(t, p, "Cloudflare de Zeri")
		if cf.Secreto != "el testigo" || len(cf.Sitios) != 1 {
			t.Errorf("la credencial no llega entera: %s", canon(cf))
		}
		if n := buscar(t, p, "Cómo se despliega"); n.Notas != "lo de siempre" {
			t.Errorf("la nota no llega entera: %s", canon(n))
		}

		// La forma del formato 2: dos épocas más la de antes, y la declarada es la de ahora.
		p.mu.Lock()
		formato, epoca, claves, actual := p.doc.Formato, p.doc.Epoca, len(p.claves.Claves), p.claves.Actual
		p.mu.Unlock()
		if formato != FormatoConClavesDeContenido || epoca == "" || epoca != actual {
			t.Errorf("formato %d, época %q y la cadena dice que la de ahora es %q", formato, epoca, actual)
		}
		if claves != 3 {
			t.Errorf("la cadena tiene %d claves y el vector se grabó con tres", claves)
		}

		// Un acceso vivo y una lápida, que no cuenta como acceso.
		if a := p.Accesos(); len(a) != 1 || a[0] != "1111222233334444" {
			t.Errorf("los accesos del vector son %v", a)
		}

		// **Y la clave de antes de migrar no abre nada**, que es la deuda que esto paga.
		if otra, err := AbrirConLaClave(ruta, []byte(laDeAntesDeMigrar)); err == nil {
			otra.Cerrar()
			t.Error("la bóveda se abre con la clave de bóveda de antes de migrar")
		}

		// Y se sigue pudiendo escribir en ella, que es lo que la publicación B añade.
		if p.SoloLectura() {
			t.Error("una bóveda del formato 2 se abre de solo lectura en la versión que lo escribe")
		}
		if err := p.Poner(Entrada{Tipo: TipoCredencial, Titulo: "Después del vector"}); err != nil {
			t.Errorf("no se puede escribir en la bóveda congelada: %v", err)
		}
	})
}
