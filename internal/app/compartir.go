package app

// Compartir copias con otra cuenta (ADR 0035 y 0043), desde la ventana.
//
// **Una copia sin permisos**: se manda cifrada hacia la identidad de quien la
// recibe, y al llegar es suya. Si luego cambia, hay que volver a mandarla.
//
// Lo que esta capa añade sobre `boveda` y `cuenta` es poco y todo de cuidado:
//
//   - **La huella se enseña siempre**, antes de mandar y al recibir. Es lo único
//     que protege del servidor en el primer envío, y si nadie la mira no protege
//     nada (TOFU, ADR 0043).
//   - **Lo que llega no entra solo en la bóveda.** Espera en el buzón hasta que
//     alguien lo acepta, que es lo que el cliente eligió: si no, cualquiera que
//     sepa tu correo te escribe dentro.

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cuenta"
)

// IdentidadParaCompartir es lo que la ventana enseña de una identidad: su huella
// y poco más. **La bóveda no manda la semilla a ninguna parte.**
type IdentidadParaCompartir struct {
	Huella string `json:"huella"`
	Suite  string `json:"suite"`
}

// EnvioRecibido es lo que espera en el buzón, ya abierto y comprobado.
type EnvioRecibido struct {
	ID string `json:"id"`
	// Huella es la de quien lo manda, para comparar por otro canal.
	Huella string `json:"huella"`
	// Titulo y Usuario son lo justo para decidir si se acepta. **El secreto no
	// cruza el puente hasta que se acepta**, como en el resto de la bóveda.
	Titulo  string `json:"titulo"`
	Usuario string `json:"usuario"`
	Tipo    string `json:"tipo"`
	Momento int64  `json:"momento"`
	// Error dice por qué un envío no se puede abrir, si es el caso: viene de otra
	// identidad, está manipulado o lo hizo una versión más nueva. Se enseña en vez
	// de esconderlo, porque un buzón con algo ilegible y sin explicación es peor.
	Error string `json:"error,omitempty"`
}

// MiIdentidad devuelve la huella de esta bóveda, creándola si hace falta, y **la
// publica en el servidor** si hay cuenta. Es lo que hay que enseñar a la otra
// persona para que compruebe que lo que le llega es tuyo.
func (a *App) MiIdentidad() (IdentidadParaCompartir, error) {
	b := a.boveda()
	if b == nil {
		return IdentidadParaCompartir{}, boveda.ErrCerrada
	}
	i, err := b.Identidad()
	if err != nil {
		return IdentidadParaCompartir{}, err
	}
	// Publicarlas es de cortesía y no puede impedir ver la propia huella: si el
	// servidor no está, se dice en el registro y se sigue.
	if token, err := a.sesionDeCuenta(); err == nil {
		_ = a.cliente().PublicarLlaves(a.ctxCuenta(), token, cuenta.Llaves{Suite: i.Suite, Cifrado: i.Cifrado, Firma: i.Firma})
	}
	return IdentidadParaCompartir{Huella: i.Huella, Suite: i.Suite}, nil
}

// HuellaDe pregunta al servidor por la identidad de un correo y devuelve su
// huella, **para enseñarla antes de mandar nada**.
//
// El servidor contesta lo mismo tenga cuenta o no esa dirección, así que esta
// huella puede ser la de nadie. Eso no se puede distinguir aquí, y por eso la
// ventana dice lo que dice: compárala con quien la tenga delante.
func (a *App) HuellaDe(correo string) (IdentidadParaCompartir, error) {
	token, err := a.sesionDeCuenta()
	if err != nil {
		return IdentidadParaCompartir{}, err
	}
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return IdentidadParaCompartir{}, err
	}
	l, err := a.cliente().LlavesDe(a.ctxCuenta(), token, c)
	if err != nil {
		return IdentidadParaCompartir{}, err
	}
	return IdentidadParaCompartir{
		Huella: boveda.HuellaDeIdentidad(l.Suite, l.Cifrado, l.Firma),
		Suite:  l.Suite,
	}, nil
}

// MandarCopia manda una copia de la entrada `id` a `correo`.
func (a *App) MandarCopia(id, correo string) error {
	b := a.boveda()
	if b == nil {
		return boveda.ErrCerrada
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return err
	}
	e, hay := b.Ver(id)
	if !hay {
		return errors.New("Esa entrada ya no está")
	}
	l, err := a.cliente().LlavesDe(a.ctxCuenta(), token, c)
	if err != nil {
		return err
	}
	sobre, err := b.MandarEntrada(e, boveda.Identidad{Suite: l.Suite, Cifrado: l.Cifrado, Firma: l.Firma})
	if err != nil {
		return err
	}
	return a.cliente().Mandar(a.ctxCuenta(), token, c, sobre)
}

// Buzon lista lo que ha llegado, **ya abierto y con la firma comprobada**, pero
// sin los secretos: lo que se enseña es de quién viene y qué es.
func (a *App) Buzon() ([]EnvioRecibido, error) {
	b := a.boveda()
	if b == nil {
		return nil, boveda.ErrCerrada
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return nil, err
	}
	brutos, err := a.cliente().Buzon(a.ctxCuenta(), token)
	if err != nil {
		return nil, err
	}
	out := make([]EnvioRecibido, 0, len(brutos))
	for _, x := range brutos {
		r := EnvioRecibido{ID: x.ID, Momento: x.Momento}
		e, de, err := abrirDelBuzon(b, x.Sobre)
		if err != nil {
			r.Error = err.Error()
		} else {
			r.Huella, r.Titulo, r.Usuario, r.Tipo = de.Huella, e.Titulo, e.Usuario, string(e.Tipo)
		}
		out = append(out, r)
	}
	return out, nil
}

// AceptarDelBuzon mete la entrada en la bóveda y quita el envío del servidor.
//
// **Con identificador nuevo**: es una copia, no la misma entrada en dos bóvedas.
// Se lo pone `Poner`, que es lo que hace con cualquier entrada sin identificador.
func (a *App) AceptarDelBuzon(id string) error {
	b := a.boveda()
	if b == nil {
		return boveda.ErrCerrada
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	brutos, err := a.cliente().Buzon(a.ctxCuenta(), token)
	if err != nil {
		return err
	}
	for _, x := range brutos {
		if x.ID != id {
			continue
		}
		e, de, err := abrirDelBuzon(b, x.Sobre)
		if err != nil {
			return err
		}
		// De dónde vino, en las notas: al aceptar una copia conviene poder mirar
		// de quién era sin fiarse de la memoria.
		e.Notas = juntarNotas(e.Notas, fmt.Sprintf("Recibida de la identidad %s", de.Huella))
		if err := b.Poner(e); err != nil {
			return err
		}
		if err := a.cliente().TirarDelBuzon(a.ctxCuenta(), token, id); err != nil {
			return err
		}
		// Lo aceptado hay que subirlo: `Poner` ya despierta al vigilante por
		// `AlGuardar`, así que aquí no hace falta pedir nada más.
		return nil
	}
	return errors.New("Ese envío ya no está en el buzón")
}

// TirarDelBuzon rechaza un envío sin abrirlo.
func (a *App) TirarDelBuzon(id string) error {
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	return a.cliente().TirarDelBuzon(a.ctxCuenta(), token, id)
}

func abrirDelBuzon(b *boveda.Boveda, crudo json.RawMessage) (boveda.Entrada, boveda.Identidad, error) {
	var s boveda.Envio
	if err := json.Unmarshal(crudo, &s); err != nil {
		return boveda.Entrada{}, boveda.Identidad{}, errors.New("Este envío no se entiende")
	}
	return b.AbrirEnvio(s)
}

func juntarNotas(notas, linea string) string {
	if notas == "" {
		return linea
	}
	return notas + "\n\n" + linea
}
