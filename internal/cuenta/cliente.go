package cuenta

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/webcafeina/esfinge/internal/red"
)

// RaizPorDefecto es dónde vive el servidor de cuentas.
const RaizPorDefecto = "https://esfinge-cuentas.webcafeina.com"

// TamanoMaximo es lo más que se acepta bajar: el tope del servidor, más margen.
const TamanoMaximo = 16 * 1024 * 1024

var (
	// ErrSinRed: ESFINGE_SIN_RED está puesta, y apaga también esto.
	ErrSinRed = errors.New("Esfinge tiene la red apagada (ESFINGE_SIN_RED)")
	// ErrSinBoveda: la cuenta existe pero todavía no ha subido ninguna bóveda.
	ErrSinBoveda = errors.New("Esta cuenta todavía no tiene bóveda")
)

// ErrorDelServidor es lo que el servidor contesta cuando dice que no. El Mensaje
// viene en español y **se puede enseñar tal cual**.
type ErrorDelServidor struct {
	Estado  int
	Mensaje string
	// Version es la de ahora, cuando se ha intentado escribir sobre otra (412).
	Version int64
}

func (e *ErrorDelServidor) Error() string { return e.Mensaje }

// Conflicto dice si el error es haber escrito sobre una versión que ya no es la
// última, y cuál es la última.
func Conflicto(err error) (int64, bool) {
	var e *ErrorDelServidor
	if errors.As(err, &e) && e.Estado == http.StatusPreconditionFailed {
		return e.Version, true
	}
	return 0, false
}

// SesionCaducada dice si hay que volver a entrar.
func SesionCaducada(err error) bool {
	var e *ErrorDelServidor
	return errors.As(err, &e) && e.Estado == http.StatusUnauthorized
}

// Cliente habla con un servidor de cuentas.
type Cliente struct {
	raiz string
	http *http.Client
	// Version va en el User-Agent, y nada más: ni identificadores ni contadores.
	Version string
}

// Nuevo prepara un cliente contra `raiz` (sin barra final).
//
// Con plazos propios, y el del saludo TLS aparte: una conexión que se queda
// colgada no puede dejar la sincronización esperando para siempre, y un saludo
// que no llega es lo que tiró una publicación entera (CLAUDE.md, la 2.18.0).
func Nuevo(raiz string) *Cliente {
	transporte := http.DefaultTransport.(*http.Transport).Clone()
	transporte.TLSHandshakeTimeout = 10 * time.Second
	transporte.ResponseHeaderTimeout = 30 * time.Second
	return &Cliente{
		raiz: strings.TrimRight(raiz, "/"),
		http: &http.Client{Timeout: 60 * time.Second, Transport: transporte},
	}
}

// Raiz es adonde apunta, para enseñarlo.
func (c *Cliente) Raiz() string { return c.raiz }

// ------------------------------------------------------------------ tipos

// Salud es lo que dice el servidor de sí mismo.
type Salud struct {
	Registro  string `json:"registro"` // cerrado · lista · abierto
	Protocolo int    `json:"protocolo"`
}

// Prelogin es lo que hace falta para derivar la clave de acceso.
type Prelogin struct {
	Sal    []byte
	Argon2 Parametros
}

// Sesion es una sesión abierta en el servidor.
type Sesion struct {
	Cuenta      string `json:"cuenta,omitempty"`
	Token       string `json:"sesion"`
	Dispositivo string `json:"dispositivo"`
	// Confianza es el testigo que ahorra el código la próxima vez en este equipo.
	Confianza string `json:"confianza,omitempty"`
}

// Alta es lo que se manda para crear la cuenta.
type Alta struct {
	Correo        string
	Codigo        string
	Sal           []byte
	Argon2        Parametros
	ClaveDeAcceso []byte
	Posesion      []byte
	Dispositivo   string
	Confiar       bool
	// Llaves son las públicas para compartir (fase B). Opcional.
	Llaves any
}

// CambioDeClave es lo que se manda para cambiar la contraseña de la cuenta.
type CambioDeClave struct {
	Posesion      []byte
	Sal           []byte
	Argon2        Parametros
	ClaveDeAcceso []byte
	// Version es sobre la que se escribe, y Documento la bóveda con la ranura nueva.
	Version   int64
	Documento []byte
}

// Sobre es la ranura de recuperación que entrega el servidor al recuperar.
type Sobre struct {
	Contenedor   string `json:"contenedor"`
	Codificacion string `json:"codificacion,omitempty"`
}

// Equipo es un equipo con sesión en la cuenta.
type Equipo struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Creado int64  `json:"creado"`
	Visto  int64  `json:"visto"`
	Actual bool   `json:"actual"`
}

// VersionGuardada es una de las versiones que conserva el servidor.
type VersionGuardada struct {
	Version int64 `json:"version"`
	Fecha   int64 `json:"fecha"`
	Tamano  int64 `json:"tamano"`
}

var b64 = base64.RawURLEncoding

// ------------------------------------------------------------------ entrar

// Salud pregunta cómo está el servidor.
func (c *Cliente) Salud(ctx context.Context) (Salud, error) {
	var s Salud
	_, err := c.json(ctx, "GET", "/v1/salud", "", nil, &s)
	return s, err
}

// Prelogin pide la sal y el coste de la cuenta, y **comprueba el coste**.
func (c *Cliente) Prelogin(ctx context.Context, correo string) (Prelogin, error) {
	var r struct {
		Sal    string     `json:"sal"`
		Argon2 Parametros `json:"argon2"`
	}
	if _, err := c.json(ctx, "POST", "/v1/prelogin", "", map[string]any{"correo": correo}, &r); err != nil {
		return Prelogin{}, err
	}
	sal, err := b64.DecodeString(r.Sal)
	if err != nil || len(sal) != 16 {
		return Prelogin{}, errors.New("El servidor ha devuelto una sal que no vale")
	}
	if err := r.Argon2.Validar(); err != nil {
		return Prelogin{}, err
	}
	return Prelogin{Sal: sal, Argon2: r.Argon2}, nil
}

// EmpezarAlta pide el código para crear la cuenta.
func (c *Cliente) EmpezarAlta(ctx context.Context, correo string) error {
	_, err := c.json(ctx, "POST", "/v1/registro/inicio", "", map[string]any{"correo": correo}, nil)
	return err
}

// TerminarAlta crea la cuenta con el código que llegó al correo.
func (c *Cliente) TerminarAlta(ctx context.Context, a Alta) (Sesion, error) {
	cuerpo := map[string]any{
		"correo":        a.Correo,
		"codigo":        a.Codigo,
		"sal":           b64.EncodeToString(a.Sal),
		"argon2":        a.Argon2,
		"claveDeAcceso": b64.EncodeToString(a.ClaveDeAcceso),
		"posesion":      b64.EncodeToString(a.Posesion),
		"dispositivo":   a.Dispositivo,
		"confiar":       a.Confiar,
	}
	if a.Llaves != nil {
		cuerpo["llaves"] = a.Llaves
	}
	var s Sesion
	_, err := c.json(ctx, "POST", "/v1/registro/fin", "", cuerpo, &s)
	return s, err
}

// Entrar abre sesión. Si el equipo es de confianza, devuelve la sesión; si no,
// un reto, y el código llega al correo: se termina con ConfirmarEntrada.
func (c *Cliente) Entrar(ctx context.Context, correo string, claveDeAcceso []byte, dispositivo, confianza string) (*Sesion, string, error) {
	cuerpo := map[string]any{
		"correo":        correo,
		"claveDeAcceso": b64.EncodeToString(claveDeAcceso),
		"dispositivo":   dispositivo,
	}
	if confianza != "" {
		cuerpo["confianza"] = confianza
	}
	var r struct {
		Sesion
		Reto string `json:"reto"`
	}
	estado, err := c.json(ctx, "POST", "/v1/sesion", "", cuerpo, &r)
	if err != nil {
		return nil, "", err
	}
	if estado == http.StatusAccepted {
		return nil, r.Reto, nil
	}
	return &r.Sesion, "", nil
}

// ConfirmarEntrada termina de entrar con el código del correo.
func (c *Cliente) ConfirmarEntrada(ctx context.Context, reto, codigo string, confiar bool) (Sesion, error) {
	var s Sesion
	_, err := c.json(ctx, "POST", "/v1/sesion/codigo", "", map[string]any{"reto": reto, "codigo": codigo, "confiar": confiar}, &s)
	return s, err
}

// CerrarSesion deja la sesión sin valor en el servidor.
func (c *Cliente) CerrarSesion(ctx context.Context, token string) error {
	_, err := c.json(ctx, "DELETE", "/v1/sesion", token, nil, nil)
	return err
}

// ------------------------------------------------------------------ la bóveda

// Bajar trae la bóveda. Con `siNoCoincide` distinto de cero, si el servidor sigue
// en esa versión no trae nada y `cambio` es falso.
func (c *Cliente) Bajar(ctx context.Context, token string, siNoCoincide int64) (datos []byte, version int64, cambio bool, err error) {
	cab := map[string]string{}
	if siNoCoincide > 0 {
		cab["If-None-Match"] = fmt.Sprintf("%q", strconv.FormatInt(siNoCoincide, 10))
	}
	resp, err := c.pedir(ctx, "GET", "/v1/boveda", token, "", nil, cab)
	if err != nil {
		return nil, 0, false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotModified:
		return nil, siNoCoincide, false, nil
	case http.StatusNotFound:
		return nil, 0, false, ErrSinBoveda
	case http.StatusOK:
		version, ok := LeerEtiqueta(resp.Header.Get("ETag"))
		if !ok {
			return nil, 0, false, errors.New("El servidor no ha dicho qué versión de la bóveda es")
		}
		datos, err := leerHasta(resp.Body, TamanoMaximo)
		if err != nil {
			return nil, 0, false, err
		}
		return datos, version, true, nil
	default:
		return nil, 0, false, errorDe(resp)
	}
}

// Subir escribe la bóveda **sobre la versión `siCoincide`**. Si ya no es la
// última, el error lo dice Conflicto.
func (c *Cliente) Subir(ctx context.Context, token string, siCoincide int64, datos []byte) (int64, error) {
	cab := map[string]string{"If-Match": fmt.Sprintf("%q", strconv.FormatInt(siCoincide, 10))}
	resp, err := c.pedir(ctx, "PUT", "/v1/boveda", token, "application/json", datos, cab)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, errorDe(resp)
	}
	var r struct {
		Version int64 `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&r); err != nil {
		return 0, err
	}
	return r.Version, nil
}

// Versiones lista las versiones que conserva el servidor.
func (c *Cliente) Versiones(ctx context.Context, token string) ([]VersionGuardada, error) {
	var vs []VersionGuardada
	_, err := c.json(ctx, "GET", "/v1/boveda/versiones", token, nil, &vs)
	return vs, err
}

// UnaVersion trae una versión concreta, para poder volver a ella.
func (c *Cliente) UnaVersion(ctx context.Context, token string, version int64) ([]byte, error) {
	resp, err := c.pedir(ctx, "GET", fmt.Sprintf("/v1/boveda/versiones/%d", version), token, "", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errorDe(resp)
	}
	return leerHasta(resp.Body, TamanoMaximo)
}

// CambiarClave cambia la contraseña de la cuenta: bóveda con la ranura nueva y
// clave de acceso nueva, a la vez. Devuelve la versión nueva y, si la sesión era
// la restringida de una recuperación, la sesión normal que la sustituye.
func (c *Cliente) CambiarClave(ctx context.Context, token string, cc CambioDeClave) (int64, string, error) {
	cuerpo := map[string]any{
		"posesion":      b64.EncodeToString(cc.Posesion),
		"sal":           b64.EncodeToString(cc.Sal),
		"argon2":        cc.Argon2,
		"claveDeAcceso": b64.EncodeToString(cc.ClaveDeAcceso),
		"version":       cc.Version,
		"documento":     string(cc.Documento),
	}
	var r struct {
		Version int64  `json:"version"`
		Sesion  string `json:"sesion"`
	}
	_, err := c.json(ctx, "PUT", "/v1/cuenta/clave", token, cuerpo, &r)
	return r.Version, r.Sesion, err
}

// ------------------------------------------------------------------ recuperar

// EmpezarRecuperacion pide el código para recuperar la cuenta. El servidor
// contesta igual haya cuenta o no.
func (c *Cliente) EmpezarRecuperacion(ctx context.Context, correo string) error {
	_, err := c.json(ctx, "POST", "/v1/recuperacion/inicio", "", map[string]any{"correo": correo}, nil)
	return err
}

// ComprobarRecuperacion cambia el código por **el sobre de recuperación** y un
// reto para el último paso.
func (c *Cliente) ComprobarRecuperacion(ctx context.Context, correo, codigo string) (string, Sobre, error) {
	var r struct {
		Reto  string `json:"reto"`
		Sobre Sobre  `json:"sobre"`
	}
	_, err := c.json(ctx, "POST", "/v1/recuperacion/codigo", "", map[string]any{"correo": correo, "codigo": codigo}, &r)
	return r.Reto, r.Sobre, err
}

// TerminarRecuperacion demuestra la posesión de la bóveda y devuelve una sesión
// restringida, que solo sirve para bajar la bóveda y cambiar la contraseña.
func (c *Cliente) TerminarRecuperacion(ctx context.Context, reto string, posesion []byte, dispositivo string) (Sesion, error) {
	var s Sesion
	_, err := c.json(ctx, "POST", "/v1/recuperacion/fin", "", map[string]any{
		"reto": reto, "posesion": b64.EncodeToString(posesion), "dispositivo": dispositivo,
	}, &s)
	return s, err
}

// ------------------------------------------------------------------ equipos, borrar, exportar

// Equipos lista los equipos con sesión en la cuenta.
func (c *Cliente) Equipos(ctx context.Context, token string) ([]Equipo, error) {
	var es []Equipo
	_, err := c.json(ctx, "GET", "/v1/dispositivos", token, nil, &es)
	return es, err
}

// OlvidarEquipo le quita la sesión a un equipo.
func (c *Cliente) OlvidarEquipo(ctx context.Context, token, id string) error {
	_, err := c.json(ctx, "DELETE", "/v1/dispositivos/"+id, token, nil, nil)
	return err
}

// PedirBorrado manda el código para borrar la cuenta y devuelve su reto.
func (c *Cliente) PedirBorrado(ctx context.Context, token string) (string, error) {
	var r struct {
		Reto string `json:"reto"`
	}
	_, err := c.json(ctx, "POST", "/v1/cuenta/borrado", token, nil, &r)
	return r.Reto, err
}

// BorrarCuenta borra la cuenta entera del servidor. No tiene vuelta atrás.
func (c *Cliente) BorrarCuenta(ctx context.Context, token string, claveDeAcceso []byte, reto, codigo string) error {
	_, err := c.json(ctx, "DELETE", "/v1/cuenta", token, map[string]any{
		"claveDeAcceso": b64.EncodeToString(claveDeAcceso), "reto": reto, "codigo": codigo,
	}, nil)
	return err
}

// Exportar trae lo que el servidor tiene de la cuenta, tal cual.
func (c *Cliente) Exportar(ctx context.Context, token string) ([]byte, error) {
	resp, err := c.pedir(ctx, "GET", "/v1/cuenta/exportacion", token, "", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errorDe(resp)
	}
	return leerHasta(resp.Body, TamanoMaximo)
}

// ------------------------------------------------------------------ piezas

var etiqueta = regexp.MustCompile(`^(?:W/)?"?(\d{1,15})"?$`)

// LeerEtiqueta saca la versión de un ETag.
//
// **Acepta la forma débil** (`W/"17"`): Cloudflare debilita el ETag al comprimir
// la respuesta, y eso solo se ve contra el servidor desplegado, nunca en local
// (CLAUDE.md). Sin esto, la bóveda bajaría sin versión en producción.
func LeerEtiqueta(v string) (int64, bool) {
	m := etiqueta.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	return n, err == nil
}

func (c *Cliente) pedir(ctx context.Context, metodo, ruta, token, tipo string, cuerpo []byte, cab map[string]string) (*http.Response, error) {
	// **La primera línea, siempre**: ESFINGE_SIN_RED apaga todas las salidas, y
	// ésta es la tercera (docs/seguridad.md).
	if red.SinRed() {
		return nil, ErrSinRed
	}
	var lector io.Reader
	if cuerpo != nil {
		lector = bytes.NewReader(cuerpo)
	}
	pet, err := http.NewRequestWithContext(ctx, metodo, c.raiz+ruta, lector)
	if err != nil {
		return nil, err
	}
	if tipo != "" {
		pet.Header.Set("Content-Type", tipo)
	}
	if token != "" {
		pet.Header.Set("Authorization", "Bearer "+token)
	}
	agente := "Esfinge"
	if c.Version != "" {
		agente += "/" + c.Version
	}
	pet.Header.Set("User-Agent", agente)
	for k, v := range cab {
		pet.Header.Set(k, v)
	}
	resp, err := c.http.Do(pet)
	if err != nil {
		return nil, fmt.Errorf("No se ha podido hablar con el servidor de cuentas: %w", err)
	}
	return resp, nil
}

// json hace una petición con cuerpo JSON y lee la respuesta en `destino`.
func (c *Cliente) json(ctx context.Context, metodo, ruta, token string, cuerpo, destino any) (int, error) {
	var crudo []byte
	tipo := ""
	if cuerpo != nil {
		var err error
		if crudo, err = json.Marshal(cuerpo); err != nil {
			return 0, err
		}
		tipo = "application/json"
	}
	resp, err := c.pedir(ctx, metodo, ruta, token, tipo, crudo, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return resp.StatusCode, errorDe(resp)
	}
	if destino != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, TamanoMaximo)).Decode(destino); err != nil {
			return resp.StatusCode, fmt.Errorf("El servidor de cuentas ha contestado algo que no se entiende: %w", err)
		}
	}
	return resp.StatusCode, nil
}

func errorDe(resp *http.Response) error {
	var r struct {
		Error   string `json:"error"`
		Version int64  `json:"version"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&r)
	if r.Error == "" {
		r.Error = fmt.Sprintf("El servidor de cuentas ha contestado %d", resp.StatusCode)
	}
	return &ErrorDelServidor{Estado: resp.StatusCode, Mensaje: r.Error, Version: r.Version}
}

func leerHasta(r io.Reader, n int64) ([]byte, error) {
	datos, err := io.ReadAll(io.LimitReader(r, n+1))
	if err != nil {
		return nil, err
	}
	if int64(len(datos)) > n {
		return nil, errors.New("Lo que ha llegado del servidor es demasiado grande")
	}
	return datos, nil
}
