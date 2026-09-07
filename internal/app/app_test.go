package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// sistemaFalso hace de escritorio durante los tests: apunta lo que se le pide en
// vez de abrir diálogos de verdad.
type sistemaFalso struct {
	mu        sync.Mutex
	ficheros  []string
	guardaEn  string
	avisos    []Progreso
	novedades []Novedad
	ordenes   []Orden
	cerrada   bool
	ficheros_ []string
}

func (s *sistemaFalso) Cerrar() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cerrada = true
}

func (s *sistemaFalso) ElegirFicheros(string, bool) ([]string, error) { return s.ficheros, nil }
func (s *sistemaFalso) ElegirDondeGuardar(string, string) (string, error) {
	return s.guardaEn, nil
}

// Avisar apunta los eventos. El candado hace falta porque la comprobación de
// actualizaciones avisa desde su propia gorrutina.
func (s *sistemaFalso) Avisar(evento string, datos any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch evento {
	case EventoProgreso:
		if p, ok := datos.(Progreso); ok {
			s.avisos = append(s.avisos, p)
		}
	case EventoNovedad:
		if n, ok := datos.(Novedad); ok {
			s.novedades = append(s.novedades, n)
		}
	case EventoOrden:
		if o, ok := datos.(Orden); ok {
			s.ordenes = append(s.ordenes, o)
		}
	case EventoFicheroAbierto:
		if ruta, ok := datos.(string); ok {
			s.ficheros_ = append(s.ficheros_, ruta)
		}
	}
}

func (s *sistemaFalso) verNovedades() []Novedad {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Novedad(nil), s.novedades...)
}

// avisosDe devuelve las rutas avisadas por un evento de fichero abierto.
func (s *sistemaFalso) avisosDe(evento string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if evento != EventoFicheroAbierto {
		return nil
	}
	return append([]string(nil), s.ficheros_...)
}

func (s *sistemaFalso) verOrdenes() []Orden {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Orden(nil), s.ordenes...)
}

// nuevaDePrueba monta la aplicación con una carpeta de configuración propia,
// para no escribir en el historial de verdad de quien ejecuta los tests.
func nuevaDePrueba(t *testing.T) (*App, *sistemaFalso) {
	t.Helper()
	casa := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", casa)
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)

	s := &sistemaFalso{}
	return Nueva("prueba", s), s
}

func TestCifrarYDescifrarTexto(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	r, err := a.CifrarTexto("postgres://u:pa55@host/db", "una clave")
	if err != nil {
		t.Fatalf("CifrarTexto: %v", err)
	}
	if !strings.HasPrefix(r.Texto, "ESF1.") {
		t.Fatalf("no ha salido un contenedor: %q", r.Texto)
	}
	if r.Aviso == "" {
		t.Error("cifrar no avisa de que sin la clave no hay vuelta atrás")
	}

	vuelta, err := a.DescifrarTexto(r.Texto, "una clave")
	if err != nil {
		t.Fatalf("DescifrarTexto: %v", err)
	}
	if vuelta.Texto != "postgres://u:pa55@host/db" {
		t.Errorf("la vuelta dio %q", vuelta.Texto)
	}
}

func TestFaltaLoQueHaceFalta(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	casos := []struct{ nombre, texto, clave, espera string }{
		{"sin contenido", "", "clave", "contenido"},
		{"solo espacios", "   ", "clave", "contenido"},
		{"sin clave", "algo", "", "clave"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := a.CifrarTexto(c.texto, c.clave)
			if err == nil {
				t.Fatal("lo ha aceptado")
			}
			if !strings.Contains(strings.ToLower(err.Error()), c.espera) {
				t.Errorf("el mensaje no dice qué falta: %v", err)
			}
		})
	}
}

func TestClaveIncorrecta(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	r, _ := a.CifrarTexto("secreto", "buena")
	if _, err := a.DescifrarTexto(r.Texto, "mala"); err == nil {
		t.Fatal("ha descifrado con la clave equivocada")
	}
}

// Una tanda de ficheros se cifra entera con la misma clave, y va avisando por
// dónde va.
func TestCifrarUnaTanda(t *testing.T) {
	a, s := nuevaDePrueba(t)
	dir := t.TempDir()

	var rutas []string
	for _, n := range []string{"uno.env", "dos.env", "tres.env"} {
		ruta := filepath.Join(dir, n)
		if err := os.WriteFile(ruta, []byte("SECRETO="+n), 0o644); err != nil {
			t.Fatal(err)
		}
		rutas = append(rutas, ruta)
	}

	out, err := a.CifrarFicheros(rutas, "la clave")
	if err != nil {
		t.Fatalf("CifrarFicheros: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("esperaba tres resultados, hay %d", len(out))
	}

	for _, r := range out {
		if r.Error != "" {
			t.Errorf("%s: %s", r.Origen, r.Error)
			continue
		}
		if _, err := os.Stat(r.Destino); err != nil {
			t.Errorf("no existe el cifrado de %s", r.Origen)
		}
		// El original no se toca.
		if _, err := os.Stat(r.Origen); err != nil {
			t.Errorf("ha desaparecido el original %s", r.Origen)
		}
	}

	// Y ha ido avisando: uno por fichero, más el último que cierra.
	if len(s.avisos) != 4 {
		t.Errorf("esperaba cuatro avisos de progreso, hay %d: %+v", len(s.avisos), s.avisos)
	}
	if ultimo := s.avisos[len(s.avisos)-1]; ultimo.Hechos != ultimo.Total {
		t.Errorf("el último aviso no cierra la tanda: %+v", ultimo)
	}
}

// Un fichero que falla no puede llevarse por delante los demás.
func TestUnFicheroQueFallaNoDetieneLaTanda(t *testing.T) {
	a, _ := nuevaDePrueba(t)
	dir := t.TempDir()

	bueno := filepath.Join(dir, "bueno.env")
	os.WriteFile(bueno, []byte("x"), 0o644)
	malo := filepath.Join(dir, "no-existe.env")
	otroBueno := filepath.Join(dir, "otro.env")
	os.WriteFile(otroBueno, []byte("y"), 0o644)

	out, err := a.CifrarFicheros([]string{bueno, malo, otroBueno}, "clave")
	if err != nil {
		t.Fatalf("la tanda entera ha fallado: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("esperaba tres resultados, hay %d", len(out))
	}
	if out[0].Error != "" || out[2].Error != "" {
		t.Error("los ficheros buenos no se han cifrado")
	}
	if out[1].Error == "" {
		t.Error("el fichero que no existe no ha dado error")
	}
}

func TestGenerarContrasena(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	p, err := a.GenerarContrasena(24, "hex")
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 48 {
		t.Errorf("24 bytes en hexadecimal son 48 caracteres, tengo %d", len(p))
	}

	if _, err := a.GenerarContrasena(24, "inventado"); err == nil {
		t.Error("ha aceptado un alfabeto que no existe")
	}

	// La interfaz necesita saber cuáles hay y cuál avisa.
	als := a.Alfabetos()
	if len(als) != 3 {
		t.Fatalf("esperaba tres alfabetos, hay %d", len(als))
	}
	if als[0].Nombre != "hex" || !als[0].SeguroURL {
		t.Error("el hexadecimal no va primero, o no se declara seguro en URL")
	}
	for _, al := range als {
		if !al.SeguroURL && al.Aviso == "" {
			t.Errorf("el alfabeto %q no es seguro en URL y no avisa", al.Nombre)
		}
	}
}

func TestEvaluarClave(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	debil := a.EvaluarClave("1234")
	buena := a.EvaluarClave("caballo grapa batería correcto")
	if debil.Nivel >= buena.Nivel {
		t.Errorf("«1234» sale con nivel %d y una frase larga con %d", debil.Nivel, buena.Nivel)
	}
	if debil.Etiqueta == "" || debil.Sugerencia == "" {
		t.Error("una clave mala no explica qué le pasa")
	}
}

// El historial guarda lo justo: qué y cuándo, nunca el secreto ni la clave.
func TestElHistorialNoGuardaSecretos(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	const secreto = "esto-no-puede-acabar-en-disco"
	const clave = "esta-clave-tampoco"
	r, err := a.CifrarTexto(secreto, clave)
	if err != nil {
		t.Fatal(err)
	}

	entradas := a.VerHistorial()
	if len(entradas) != 1 {
		t.Fatalf("esperaba una entrada, hay %d", len(entradas))
	}
	if entradas[0].Accion != AccionCifrar {
		t.Errorf("la acción anotada es %q", entradas[0].Accion)
	}

	// Y lo que hay en el disco tampoco.
	ruta := a.DondeVive()
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se ha escrito el historial en %s: %v", ruta, err)
	}
	for _, prohibido := range []string{secreto, clave, r.Texto} {
		if strings.Contains(string(datos), prohibido) {
			t.Errorf("el historial en disco contiene algo que no debería: %q", prohibido)
		}
	}

	// Con permisos de solo su dueño.
	info, _ := os.Stat(ruta)
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permisos %o, esperaba 600", perm)
	}
}

func TestVaciarHistorial(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	a.CifrarTexto("algo", "clave")
	if len(a.VerHistorial()) == 0 {
		t.Fatal("no ha anotado nada")
	}

	if err := a.VaciarHistorial(); err != nil {
		t.Fatal(err)
	}
	if n := len(a.VerHistorial()); n != 0 {
		t.Errorf("quedan %d entradas tras vaciar", n)
	}
	if _, err := os.Stat(a.DondeVive()); !os.IsNotExist(err) {
		t.Error("el fichero sigue en el disco tras vaciar")
	}

	// Vaciar dos veces no puede fallar.
	if err := a.VaciarHistorial(); err != nil {
		t.Errorf("vaciar un historial ya vacío da error: %v", err)
	}
}

// El historial sobrevive a cerrar y abrir la aplicación.
func TestElHistorialPersiste(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", casa)
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)

	uno := Nueva("prueba", &sistemaFalso{})
	uno.CifrarTexto("algo", "clave")

	dos := Nueva("prueba", &sistemaFalso{})
	if n := len(dos.VerHistorial()); n != 1 {
		t.Errorf("tras reabrir hay %d entradas y esperaba 1", n)
	}
}

// Un historial corrupto no puede impedir que la aplicación abra.
func TestUnHistorialCorruptoNoRompeNada(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", casa)
	t.Setenv("HOME", casa)
	t.Setenv("USERPROFILE", casa)

	ruta := filepath.Join(casa, "Esfinge", "historial.json")
	os.MkdirAll(filepath.Dir(ruta), 0o700)
	os.WriteFile(ruta, []byte("{esto no es json"), 0o600)

	a := Nueva("prueba", &sistemaFalso{})
	if n := len(a.VerHistorial()); n != 0 {
		t.Errorf("ha leído %d entradas de un fichero corrupto", n)
	}
	if _, err := a.CifrarTexto("algo", "clave"); err != nil {
		t.Errorf("con el historial corrupto ya no se puede cifrar: %v", err)
	}
}

// El historial no crece sin fin.
func TestElHistorialTieneTope(t *testing.T) {
	h := &Historial{ruta: filepath.Join(t.TempDir(), "h.json")}
	for i := 0; i < maximo+50; i++ {
		h.Anotar(AccionCifrar, "fichero", "")
	}
	if n := len(h.Entradas()); n != maximo {
		t.Errorf("hay %d entradas y el tope es %d", n, maximo)
	}
}

func TestGuardarTexto(t *testing.T) {
	a, s := nuevaDePrueba(t)
	s.guardaEn = filepath.Join(t.TempDir(), "resultado.txt")

	ruta, err := a.GuardarTexto("resultado.txt", "ESF1.loquesea")
	if err != nil {
		t.Fatal(err)
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(datos)) != "ESF1.loquesea" {
		t.Errorf("el fichero tiene %q", datos)
	}
	info, _ := os.Stat(ruta)
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permisos %o, esperaba 600", perm)
	}

	// Cancelar el diálogo no es un error ni deja fichero.
	s.guardaEn = ""
	if ruta, err := a.GuardarTexto("x.txt", "y"); err != nil || ruta != "" {
		t.Errorf("cancelar dio ruta %q y error %v", ruta, err)
	}
}

// Lo que cruza el puente hacia la ventana tiene que poder viajar como JSON.
func TestLoQueCruzaElPuenteEsSerializable(t *testing.T) {
	a, _ := nuevaDePrueba(t)

	r, _ := a.CifrarTexto("algo", "clave")
	for nombre, v := range map[string]any{
		"Resultado":    r,
		"Fuerza":       a.EvaluarClave("clave"),
		"Alfabetos":    a.Alfabetos(),
		"Historial":    a.VerHistorial(),
		"Progreso":     Progreso{Hechos: 1, Total: 2, Actual: "x"},
		"Novedad":      a.NovedadPendiente(),
		"Preferencias": a.VerPreferencias(),
		"Orden":        Orden{Que: OrdenIrAAjustes},
	} {
		if _, err := json.Marshal(v); err != nil {
			t.Errorf("%s no se puede serializar: %v", nombre, err)
		}
	}
}
