package cli

// Las bóvedas que le han compartido a este servidor (ADR 0057, sobre la 0052).
//
// Es la mitad que hace útil todo lo demás: el servidor no guarda nada propio —su
// bóveda personal está vacía— y lo que usa son **las bóvedas de proyecto que
// alguien le marque desde la aplicación**. Marcarlas es darle acceso de solo ver, y
// lo que llega aquí es lo mismo que le llegaría a una persona: un sobre en el buzón
// de su cuenta.
//
// Cuatro cosas que no son detalles:
//
//   - **Lo del buzón no entra solo.** Igual que en la ventana: si entrara, cualquiera
//     que supiera el correo de este servidor le metería una bóveda en la lista. Lo
//     que pasa aquí es que quien acepta no es una persona sino esta orden, y por eso
//     **solo se aceptan sobres que abra la identidad de este equipo** — un sobre
//     sellado hacia otro no se puede abrir, así que no hay nada que decidir.
//   - **El acceso se apunta antes de tirar el sobre.** Si se tirara primero y fallara
//     lo de después, no habría forma de volver a saber dónde está esa bóveda. Que la
//     descarga falle se puede reintentar.
//   - **Se comprueba que lo bajado abre antes de dejarlo puesto.** Un fichero en esa
//     carpeta es una bóveda compartida para todo lo demás, y si no lo abre la
//     identidad de aquí es mejor no tenerlo que tenerlo y que falle cuando ya nadie
//     sepa de dónde salió.
//   - **Con permiso de solo ver no se sube nada**, y lo impone el servidor. Aquí no
//     se intenta disimular: la pasada de una compartida sube si puede, y si el
//     servidor dice que no, eso es lo que hay.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cuenta"
	"github.com/webcafeina/esfinge/internal/escritura"
	"github.com/webcafeina/esfinge/internal/rutas"
	"github.com/webcafeina/esfinge/internal/sincro"
)

// comandoServidorRecibir acepta lo que haya en el buzón y se baja lo que falte.
func comandoServidorRecibir(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "recibir",
		Short: "Acepta las bóvedas que le hayan compartido a este servidor",
		Long: "Mira el buzón de la cuenta de este equipo, acepta los accesos que le\n" +
			"hayan dado y se trae los ficheros que falten.\n\n" +
			"Es lo que hay que hacer una vez después de marcar una bóveda «para el\n" +
			"servidor» en la aplicación. Luego se mantiene sola: «sincronizar» pone al\n" +
			"día las que ya están aquí.",
		Example:       "  esfinge boveda servidor recibir",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			e, err := estilos(o.tema)
			if err != nil {
				return err
			}
			b, err := abrirLaBoveda(o, ob)
			if err != nil {
				return err
			}
			defer b.Cerrar()

			nuevas, traidas, err := recibirLoCompartido(b)
			if err != nil {
				return err
			}
			if ob.json {
				return escribirJSON(map[string]any{"aceptadas": nuevas, "traidas": traidas})
			}
			if !o.silencio {
				switch {
				case len(nuevas) > 0:
					fmt.Fprintln(os.Stderr, e.Ok(fmt.Sprintf("Aceptadas %d: %s.", len(nuevas), strings.Join(nuevas, ", "))))
				case len(traidas) > 0:
					fmt.Fprintln(os.Stderr, e.Ok(fmt.Sprintf("Traídas %d que ya estaban aceptadas.", len(traidas))))
				default:
					fmt.Fprintln(os.Stderr, e.Info("No hay nada nuevo para este servidor."))
				}
			}
			return nil
		},
	}
}

// recibirLoCompartido acepta lo del buzón y baja lo que falte en el disco.
//
// Devuelve los nombres de lo aceptado y de lo traído, que son dos cosas distintas:
// un acceso aceptado en otra pasada puede seguir sin fichero aquí si la descarga
// falló entonces.
func recibirLoCompartido(b *boveda.Boveda) (nuevas, traidas []string, err error) {
	s, err := abrirLaCuenta(b, b.Ruta())
	if err != nil {
		return nil, nil, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), elPlazoDeLaRed)
	defer cancelar()

	err = s.conLaSesion(ctx, func(token string) error {
		sobres, err := s.c.Buzon(ctx, token)
		if err != nil {
			return err
		}
		for _, x := range sobres {
			acc, de, err := abrirElSobre(b, x.Sobre)
			if err != nil {
				// Un sobre que esta identidad no abre no es para este equipo, y no es
				// un fallo: se queda en el buzón. Lo que no se hace es tirarlo —no es
				// nuestro— ni parar por él.
				continue
			}
			if err := b.PonerCompartida(boveda.Compartida{
				Dueno: acc.Dueno, Ref: acc.Ref, Nombre: acc.Nombre,
				Titular: acc.Titular, Permiso: acc.Permiso, Huella: de.Huella,
			}); err != nil {
				return err
			}
			// **Apuntado antes de tirar el sobre.**
			if err := s.c.TirarDelBuzon(ctx, token, x.ID); err != nil {
				return err
			}
			// **La huella de quien lo comparte se dice**, aunque aquí no haya nadie
			// que la compare: es lo único que distingue un acceso de quien dices de
			// uno de quien sepa tu correo, y queda en la salida del comando para quien
			// la lea después (ADR 0043).
			nuevas = append(nuevas, fmt.Sprintf("%s (de %s)", acc.Nombre, de.Huella))
		}
		// Y los ficheros que falten, de todo lo que esté en la lista.
		for _, c := range b.Compartidas() {
			if c.Retirada != "" {
				continue
			}
			ruta := rutas.Compartida(c.Dueno, c.Ref)
			if ruta == "" {
				return errors.New("No encuentro dónde guardar la bóveda en este sistema")
			}
			if _, err := os.Stat(ruta); err == nil {
				continue
			}
			if err := bajarLaCompartida(ctx, s.c, token, b, c, ruta); err != nil {
				// **Si ya no es nuestra, se tacha y se sigue** (ADR 0053), que es lo
				// mismo que hace la pasada: parar aquí dejaría sin traer las demás por
				// una a la que alguien quitó el acceso entre medias.
				if cuenta.Revocado(err) {
					_ = b.RetirarCompartida(c.Dueno, c.Ref, ahoraRFC3339())
					continue
				}
				return fmt.Errorf("«%s»: %w", c.Nombre, err)
			}
			traidas = append(traidas, c.Nombre)
		}
		return nil
	})
	return nuevas, traidas, err
}

func abrirElSobre(b *boveda.Boveda, crudo json.RawMessage) (boveda.Acceso, boveda.Identidad, error) {
	var s boveda.Envio
	if err := json.Unmarshal(crudo, &s); err != nil {
		return boveda.Acceso{}, boveda.Identidad{}, errors.New("Este envío no se entiende")
	}
	return b.AbrirAcceso(s)
}

// bajarLaCompartida la trae y **comprueba que abre antes de dejarla puesta**.
func bajarLaCompartida(ctx context.Context, c *cuenta.Cliente, token string,
	mia *boveda.Boveda, comp boveda.Compartida, ruta string) error {
	datos, _, _, err := c.BajarCompartida(ctx, token, comp.Dueno, comp.Ref, 0)
	if err != nil {
		return err
	}
	prueba, err := boveda.AbrirCompartidaBytes("", datos, comp.Titular, mia)
	if err != nil {
		// **Y la causa más probable va en la frase**, porque pasa siempre la primera
		// vez: dar acceso cambia el fichero de quien lo da, y lo que lo sube es su
		// sincronización. Mirando el servidor en el mismo segundo, lo que hay en el
		// servidor de cuentas es la bóveda de antes, sin esta ranura dentro.
		return fmt.Errorf("lo que ha bajado no lo abre la identidad de este equipo; "+
			"puede que quien te dio el acceso no lo haya subido todavía, así que inténtalo "+
			"otra vez en un minuto (%w)", err)
	}
	prueba.Cerrar()

	return escritura.Atomica(ruta, escritura.Opciones{Permisos: 0o600, CrearCarpeta: true},
		func(w io.Writer) error {
			_, err := w.Write(datos)
			return err
		})
}

// ------------------------------------------------- abrir una por su nombre

// laCompartidaQueSeLlama encuentra una bóveda compartida por su nombre, sin
// adivinar: con dos que encajen, las enseña y pide que se afine, igual que `ver`.
//
// **Por el nombre y no por la referencia** porque la referencia es hex al azar: lo
// que alguien escribe en un guion de despliegue es «Tempero».
func laCompartidaQueSeLlama(b *boveda.Boveda, nombre string) (boveda.Compartida, error) {
	q := strings.ToLower(strings.TrimSpace(nombre))
	var encajan []boveda.Compartida
	for _, c := range b.Compartidas() {
		if c.Retirada != "" {
			continue
		}
		if strings.Contains(strings.ToLower(c.Nombre), q) {
			encajan = append(encajan, c)
		}
	}
	switch len(encajan) {
	case 0:
		return boveda.Compartida{}, fmt.Errorf(
			"A este servidor no le han compartido ninguna bóveda que encaje con «%s»", nombre)
	case 1:
		return encajan[0], nil
	default:
		var nombres []string
		for _, c := range encajan {
			nombres = append(nombres, c.Nombre)
		}
		return boveda.Compartida{}, fmt.Errorf("«%s» encaja con %d bóvedas (%s); afina el nombre",
			nombre, len(encajan), strings.Join(nombres, ", "))
	}
}

// abrirLaCompartidaPorNombre abre una bóveda compartida **a través de la personal**,
// que es de donde sale la identidad que abre su ranura (ADR 0052).
//
// Y la pone al día antes de devolverla, por lo mismo que la personal: una credencial
// vieja en un servidor es un fallo mudo.
func abrirLaCompartidaPorNombre(o *opciones, ob *opcionesBoveda, nombre string) (*boveda.Boveda, error) {
	mia, err := abrirLaBoveda(o, ob)
	if err != nil {
		return nil, err
	}
	defer mia.Cerrar()

	comp, err := laCompartidaQueSeLlama(mia, nombre)
	if err != nil {
		return nil, err
	}
	ruta := rutas.Compartida(comp.Dueno, comp.Ref)
	if _, err := os.Stat(ruta); err != nil {
		return nil, fmt.Errorf(
			"«%s» no está todavía en este equipo; tráela con «esfinge boveda servidor recibir»", comp.Nombre)
	}
	b, err := boveda.AbrirCompartida(ruta, comp.Titular, mia)
	if err != nil {
		return nil, err
	}
	// La pasada va **con la personal todavía abierta**: la sesión vive sellada en
	// ella, y la compartida no la lleva dentro.
	if !ob.sinSincronizar {
		if err := ponerAlDiaLaCompartida(o, mia, b, comp); err != nil {
			// Puede venir ya cerrada —`retirarLaCompartida` la cierra para poder
			// borrar el fichero—, y cerrar dos veces no hace daño.
			b.Cerrar()
			return nil, err
		}
	}
	return b, nil
}

// ponerAlDiaLaCompartida hace una pasada contra **la cuenta de la otra persona**,
// que es lo que distingue esto de sincronizar lo propio.
//
// Casi todo lo que falle aquí **se dice y se sigue**, por lo mismo que
// `ponerseAlDia`: callarlo deja leyendo una credencial vieja, y negarse deja un
// servidor parado porque la red estuviera mal un segundo.
//
// **Salvo una cosa, y es la que devuelve error: que le hayan quitado el acceso**
// (ADR 0053). Entonces la bóveda se va de este equipo y no hay nada que abrir.
func ponerAlDiaLaCompartida(o *opciones, mia, b *boveda.Boveda, comp boveda.Compartida) error {
	s, err := abrirLaCuenta(mia, mia.Ruta())
	if err != nil {
		return nil
	}
	ctx, cancelar := context.WithTimeout(context.Background(), elPlazoDeLaRed)
	defer cancelar()
	err = s.conLaSesion(ctx, func(token string) error {
		sin := &sincro.Sincronizador{
			Servidor: laCompartidaRemota{c: s.c, dueno: comp.Dueno, ref: comp.Ref},
			Boveda:   b,
			Memoria:  sincro.JuntoALaBoveda{Ruta: b.Ruta()},
			Token:    func() string { return token },
		}
		_, err := sin.Sincronizar(ctx)
		return err
	})
	if err == nil {
		return nil
	}
	if cuenta.Revocado(err) {
		return retirarLaCompartida(mia, b, comp)
	}
	if !o.silencio {
		if e, err2 := estilos(o.tema); err2 == nil {
			fmt.Fprintln(os.Stderr, e.Ojo("No he podido poner al día «"+comp.Nombre+"», así que esto puede estar viejo: "+err.Error()))
		}
	}
	return nil
}

// retirarLaCompartida se va de una bóveda a la que han quitado el acceso: **la
// tacha en la lista y borra el fichero de este equipo** (ADR 0053).
//
// Es lo mismo que hace la ventana, y en un servidor importa más: la copia de un
// proyecto de un cliente con el que ya no se trabaja se quedaría ahí, legible por
// quien entre en la máquina, **para siempre**. Rotar la clave al revocar (ADR 0052)
// corta lo que llega, no lo que ya está en el disco.
//
// **`Revocado` y no `SinAcceso`**, que ahí está toda la diferencia entre dejar de
// sincronizar y destruir una bóveda: el 403 también es «solo puedes ver» —que es
// justo el permiso de un servidor— y además puede venir de un portero que nadie
// puso ahí para esto.
//
// La lápida es lo que impide que el otro equipo la resucite al sincronizar.
func retirarLaCompartida(mia, b *boveda.Boveda, comp boveda.Compartida) error {
	ruta := b.Ruta()
	b.Cerrar()
	cuando := ahoraRFC3339()
	if err := mia.RetirarCompartida(comp.Dueno, comp.Ref, cuando); err != nil {
		return fmt.Errorf("Ya no tienes acceso a «%s», y no he podido tacharla: %w", comp.Nombre, err)
	}
	if err := sincro.BorrarLaBovedaEntera(ruta); err != nil {
		return fmt.Errorf("Ya no tienes acceso a «%s», y no he podido borrarla de este equipo: %w", comp.Nombre, err)
	}
	return fmt.Errorf(
		"Ya no tienes acceso a «%s»: quien te la compartió te lo ha quitado, y se ha borrado de este equipo",
		comp.Nombre)
}

func ahoraRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

// laCompartidaRemota apunta la sincronización a la bóveda de otra cuenta. Es el
// espejo de `bovedaRemota` de la ventana.
type laCompartidaRemota struct {
	c          *cuenta.Cliente
	dueno, ref string
}

func (r laCompartidaRemota) Bajar(ctx context.Context, token string, siNoCoincide int64) ([]byte, int64, bool, error) {
	return r.c.BajarCompartida(ctx, token, r.dueno, r.ref, siNoCoincide)
}

func (r laCompartidaRemota) Subir(ctx context.Context, token string, siCoincide int64, datos []byte) (int64, error) {
	return r.c.SubirACompartida(ctx, token, r.dueno, r.ref, siCoincide, datos)
}
