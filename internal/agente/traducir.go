package agente

// El traductor entre un cliente MCP y el canal de Esfinge.
//
// Es el gemelo de `navegador.Traducir`, y lo que hace es lo mismo: habla el protocolo
// de uno por la entrada y la salida estándar, y el del otro por el socket. **No sabe
// abrir una bóveda** y no importa el paquete que lo haría.
//
// La diferencia con aquél, y es la que manda: **aquí hay cosas que se contestan sin
// Esfinge**. `initialize` y `tools/list` salen de la tabla de este paquete, así que un
// agente que arranque con la ventana cerrada ve igualmente las herramientas y puede
// decirle a quien le habla que abra Esfinge. Si eso se contestara preguntando al
// socket, un arranque con la ventana cerrada dejaría al agente sin herramientas toda
// la sesión.

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// plazoDeLaConexion es lo que se espera a que Esfinge conteste una llamada.
//
// **Es más de lo que tarda cualquier cosa de la bóveda y menos de lo que aguanta un
// cliente MCP.** En medio está lo único que puede tardar de verdad: que una persona
// mire la ventana y pulse. Eso lo acota Esfinge por su lado, no aquí.
const plazoDeLaConexion = 90 * time.Second

// Traducir atiende a un cliente MCP y lleva lo que haga falta al socket de Esfinge.
//
// **Y se empareja solo**, que es la mitad que faltaba: todo lo que toca la bóveda exige
// un testigo, y el testigo lo da Esfinge cuando una persona dice que sí en su ventana.
// Sin esto, un agente solo podía preguntar el estado — y **no lo vio ninguna prueba**,
// porque todas ponían el testigo a mano, incluida la de la tubería. Lo encontró recorrer
// el camino.
func Traducir(entra io.Reader, sale io.Writer, version, socket string) error {
	t := &traductor{socket: socket, testigo: testigoGuardado()}
	return Hablar(entra, sale, version, t.llamar, t.anotarQuien)
}

type traductor struct {
	socket  string
	testigo string
	// quien es lo que el cliente dijo de sí mismo en el saludo. **No se cree**: sirve
	// para que la ventana pueda escribir «Claude Code quiere…» en vez de «un agente».
	quien string
}

func (t *traductor) anotarQuien(quien string) {
	if quien != "" {
		t.quien = quien
	}
}

func (t *traductor) comoMeLlamo() string {
	if t.quien == "" {
		return "Un agente"
	}
	return t.quien
}

func (t *traductor) llamar(h Herramienta, args map[string]any) Resultado {
	p := Peticion{Version: VersionDelProtocolo, Que: h.Verbo, Quien: t.comoMeLlamo(), Testigo: t.testigo}
	ponerArgumentos(&p, args)

	r, err := preguntar(t.socket, p)
	if err != nil {
		// **Esto se escribe a mano y no se serializa**, por lo mismo que
		// `respuestaSinEsfinge` en el canal del navegador: este camino es el de cuando
		// ya ha fallado algo, y lo último que hace falta es que dependa de que otra
		// cosa funcione.
		return fallo("Esfinge no está abierta en este equipo, o el canal de agentes está apagado en sus Ajustes.")
	}

	// **Sin testigo, se pide uno y se vuelve a intentar.** Una sola vez: si Esfinge
	// sigue diciendo que no, es que nadie ha dicho que sí todavía, y lo que hay que
	// hacer es contárselo a quien habla con el agente, no insistir en un bucle.
	if !r.OK && r.Motivo == MotivoSinEmparejar {
		if nuevo, err := t.emparejarse(); err == nil {
			t.testigo = nuevo
			p.Testigo = nuevo
			if otra, err := preguntar(t.socket, p); err == nil {
				r = otra
			}
		} else {
			return fallo("Esfinge tiene que darte permiso: ábrela, ve a Ajustes y permite este agente. " +
				"Después vuelve a pedir lo que querías.")
		}
	}
	if !r.OK {
		return fallo(r.Error)
	}
	return comoTexto(h, r)
}

// emparejarse pide el testigo. **No espera a nadie**: si no hay permiso, Esfinge avisa a
// su ventana y contesta que no, y el agente lo vuelve a pedir cuando alguien conteste.
func (t *traductor) emparejarse() (string, error) {
	r, err := preguntar(t.socket, Peticion{
		Version: VersionDelProtocolo, Que: QueEmparejar, Quien: t.comoMeLlamo(),
	})
	if err != nil {
		return "", err
	}
	if !r.OK || r.Testigo == "" {
		return "", errors.New(r.Error)
	}
	guardarTestigo(r.Testigo)
	return r.Testigo, nil
}

// rutaDelTestigo es donde este proceso se guarda el suyo.
//
// **En disco y no solo en memoria**, porque el cliente MCP arranca un proceso nuevo en
// cada sesión y pedir permiso en cada una sería insoportable. Es lo mismo que hace la
// extensión del navegador, que lo guarda en `storage.local`, y lo que eso significa está
// dicho desde la ADR 0027: **cualquier programa que corra como tú puede leerlo**. Lo que
// el permiso compra no es que nadie más pueda, es que tenga que pasar por un aviso.
func rutaDelTestigo() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Esfinge", "agente-testigo")
}

func testigoGuardado() string {
	ruta := rutaDelTestigo()
	if ruta == "" {
		return ""
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(datos))
}

func guardarTestigo(t string) {
	ruta := rutaDelTestigo()
	if ruta == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return
	}
	// Que no se pueda guardar no rompe nada: se vuelve a pedir en la sesión siguiente.
	_ = os.WriteFile(ruta, []byte(t), 0o600)
}

func ponerArgumentos(p *Peticion, args map[string]any) {
	if args == nil {
		return
	}
	if v, ok := args["texto"].(string); ok {
		p.Texto = v
	}
	if v, ok := args["id"].(string); ok {
		p.ID = v
	}
	if v, ok := args["alfabeto"].(string); ok {
		p.Alfabeto = v
	}
	// Los números de JSON llegan como float64.
	if v, ok := args["bytes"].(float64); ok {
		p.Bytes = int(v)
	}
	if v, ok := args["generar"].(bool); ok {
		p.Generar = v
	}
	// **Lo que decide qué se toca es qué claves vienen**, no qué valores: por eso se
	// copian una a una y no se rellena un objeto con ceros.
	for _, campo := range []string{"tipo", "titulo", "usuario", "secreto", "notas", "carpeta", "totp"} {
		if v, ok := args[campo].(string); ok {
			if p.Campos == nil {
				p.Campos = map[string]string{}
			}
			p.Campos[campo] = v
		}
	}
}

func preguntar(socket string, p Peticion) (Respuesta, error) {
	c, err := net.DialTimeout("unix", socket, plazoDeLaConexion)
	if err != nil {
		return Respuesta{}, err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(plazoDeLaConexion))

	if err := json.NewEncoder(c).Encode(p); err != nil {
		return Respuesta{}, err
	}
	var r Respuesta
	if err := json.NewDecoder(io.LimitReader(c, 1<<20)).Decode(&r); err != nil {
		return Respuesta{}, err
	}
	return r, nil
}

// comoTexto escribe la respuesta para que la lea un modelo.
//
// **El texto se escribe aquí y no se vuelca el JSON entero**, y no es por estética: lo
// que se escriba acaba en la transcripción del cliente, así que lo que no se escriba no
// acaba. Un `json.Marshal` de la respuesta es lo que un día colaría un campo nuevo sin
// que nadie lo decidiera.
func comoTexto(h Herramienta, r Respuesta) Resultado {
	crudo, err := json.Marshal(soloLoQueSale(h, r))
	if err != nil {
		return fallo("Esfinge ha contestado algo que no se puede leer")
	}
	return texto(string(crudo))
}

// soloLoQueSale es la aduana: **se nombra campo a campo lo que puede salir** hacia el
// modelo, en vez de dejar pasar la respuesta entera. Lo que no esté aquí, no sale.
func soloLoQueSale(h Herramienta, r Respuesta) any {
	switch h.Verbo {
	case QueEstado:
		return r.Estado
	case QueBuscar:
		// **El total va aparte de lo que vuelve**, que está topado: si no, un agente
		// que recibe 25 de 1.843 creería que la bóveda tiene 25.
		return map[string]any{"entradas": r.Entradas, "devueltas": len(r.Entradas), "enTotal": r.Cuantas}
	case QueVer:
		return r.Entrada
	case QueHigiene:
		return r.Higiene
	case QueGenerar:
		return map[string]any{"contrasena": r.Clave}
	case QueCopiarSecreto:
		// **Lo que sale es que se ha copiado, nunca lo copiado.** Si algún día alguien
		// mete la contraseña en `Copiado`, esto la dejaría pasar: por eso hay una
		// prueba que busca el secreto en los bytes de todas las respuestas.
		return r.Copiado
	case QueCodigo:
		// **Aquí sí sale algo que el modelo ve.** Es la única, y lo que la hace
		// aceptable es que caduque en treinta segundos y no sirva sin la contraseña.
		return r.Codigo
	case QueCrear, QueEditar, QueBorrar:
		return r.Escrito
	}
	return map[string]any{"hecho": true}
}
