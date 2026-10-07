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
	"io"
	"net"
	"time"
)

// plazoDeLaConexion es lo que se espera a que Esfinge conteste una llamada.
//
// **Es más de lo que tarda cualquier cosa de la bóveda y menos de lo que aguanta un
// cliente MCP.** En medio está lo único que puede tardar de verdad: que una persona
// mire la ventana y pulse. Eso lo acota Esfinge por su lado, no aquí.
const plazoDeLaConexion = 90 * time.Second

// Traducir atiende a un cliente MCP y lleva lo que haga falta al socket de Esfinge.
func Traducir(entra io.Reader, sale io.Writer, version, socket string) error {
	return Hablar(entra, sale, version, func(h Herramienta, args map[string]any) Resultado {
		p := Peticion{Version: VersionDelProtocolo, Que: h.Verbo, Quien: "un agente"}
		ponerArgumentos(&p, args)

		r, err := preguntar(socket, p)
		if err != nil {
			// **Esto se escribe a mano y no se serializa**, por lo mismo que
			// `respuestaSinEsfinge` en el canal del navegador: este camino es el de
			// cuando ya ha fallado algo, y lo último que hace falta es que dependa de
			// que otra cosa funcione.
			return fallo("Esfinge no está abierta en este equipo, o el canal de agentes está apagado en sus Ajustes.")
		}
		if !r.OK {
			return fallo(r.Error)
		}
		return comoTexto(h, r)
	})
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
