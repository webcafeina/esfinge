package app

// Dar acceso a una bóveda de proyecto, desde la ventana (ADR 0052).
//
// Es la otra cosa que se puede hacer con un proyecto cuando entra otra persona, y
// **no sustituye a entregar una copia** (ADR 0051): entregar sigue al lado, para
// cuando el proyecto de verdad se acaba y lo que se quiere es soltarlo.
//
// Cuatro reglas que esto respeta y que no se ven leyendo el código de una en una:
//
//   - **Se da acceso desde dentro del proyecto**, con él abierto. No es una
//     formalidad: poner la ranura cambia **su** fichero, y lo que lo sube es la
//     sincronización de la bóveda abierta. Haciéndolo desde la lista, la ranura se
//     quedaría aquí y quien recibe el acceso se bajaría una bóveda que no puede
//     abrir hasta que alguien entrara en ese proyecto.
//   - **Lo de la cuenta es de la bóveda personal, siempre**: la identidad con la que
//     se sella el sobre y el buzón salen de ella por `conLaPersonal`, esté abierta la
//     que esté. Es la regla que se saltaba `compartir.go` hasta la 2.39.3.
//   - **La huella se enseña antes.** Es lo único que protege del servidor en el
//     primer envío, y una huella que nadie mira no protege nada (ADR 0043).
//   - **El permiso lo hace cumplir el servidor, no esta capa.** Lo que se guarda aquí
//     es informativo, para no pedirle a nadie que teclee algo que va a acabar en un
//     403 — la regla que costó `ExportarLlaves`.

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/cuenta"
)

// QuienTieneAcceso es una persona con acceso a esta bóveda, como se enseña.
type QuienTieneAcceso struct {
	Titular string `json:"titular"`
	Correo  string `json:"correo"`
	Permiso string `json:"permiso"`
	Desde   string `json:"desde"`
	// EnElServidor dice si el servidor también le deja entrar.
	//
	// **Son dos listas y no hay forma de que sean una**: el servidor no puede leer la
	// bóveda —así que no sabe correos— y la pantalla no puede depender de él para
	// decir nombres. Cuando no coinciden se enseña en vez de disimularlo, que es lo
	// que permite arreglarlo en vez de descubrirlo el día que alguien no entra.
	EnElServidor bool `json:"enElServidor"`
}

// DarAcceso le da acceso a esa dirección **sobre la bóveda de proyecto abierta**.
//
// El orden importa y no es el evidente: **la ranura, el servidor, y el sobre al
// final**. Si el miembro entrara en el servidor antes que la ranura, esa persona
// podría bajarse una bóveda que todavía no puede abrir; y el sobre va el último
// porque es el único paso que **no se puede deshacer** — una vez en su buzón, está.
// Si algo fallara antes, lo que queda es una ranura de más que no abre nada sin él.
func (a *App) DarAcceso(correo, permiso string) error {
	if permiso != "ver" && permiso != "editar" {
		return errors.New("Ese permiso no existe")
	}
	ref := a.bovedaActiva()
	if ref == "" || a.duenoDeLaActiva() != "" {
		return errors.New("Entra en el proyecto que quieras compartir")
	}
	p := a.boveda()
	if p == nil {
		return boveda.ErrCerrada
	}
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return err
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	mia := leerDatosCuenta().Cuenta
	if mia == "" {
		return errors.New("Para dar acceso hace falta una cuenta")
	}

	// A quién: sus llaves públicas, que son las que sellan su ranura.
	l, err := a.cliente().LlavesDe(a.ctxCuenta(), token, c)
	if err != nil {
		return err
	}
	suya := boveda.Identidad{Suite: l.Suite, Cifrado: l.Cifrado, Firma: l.Firma}
	huella := boveda.HuellaDeIdentidad(l.Suite, l.Cifrado, l.Firma)

	// **El titular lo elige quien da el acceso**, no quien lo recibe: es lo que
	// después sirve para quitárselo, y el servidor nunca dice de quién es una cuenta.
	crudo, err := cripto.Azar(8)
	if err != nil {
		return err
	}
	titular := hex.EncodeToString(crudo)

	// 1 · La ranura y el nombre, dentro del fichero del proyecto. Guardar despierta a
	// la sincronización, que es lo que lo sube.
	if err := p.PonerAcceso(titular, suya); err != nil {
		return err
	}
	if err := p.PonerTitular(boveda.Titular{ID: titular, Correo: c, Permiso: permiso, Huella: huella}); err != nil {
		return err
	}

	// 2 · El miembro, en el servidor.
	if err := a.cliente().DarAcceso(a.ctxCuenta(), token, ref, c, permiso, titular); err != nil {
		return err
	}

	// 3 · El sobre, **sellado con la identidad de la personal** y no con la de este
	// proyecto, que no tiene ni debe tener.
	nombre := a.nombreDeLaActiva()
	var sobre boveda.Envio
	if err := a.conLaPersonal(func(b *boveda.Boveda) error {
		var err error
		sobre, err = b.MandarAcceso(boveda.Acceso{
			Dueno: mia, Ref: ref, Nombre: nombre, Titular: titular, Permiso: permiso,
		}, suya)
		return err
	}); err != nil {
		return err
	}
	a.Actividad()
	return a.cliente().Mandar(a.ctxCuenta(), token, c, sobre)
}

// QuitarAcceso se lo quita a ese titular: **la puerta del servidor primero**, que es
// la que corta de verdad y en el acto, y después la ranura.
//
// Lo que esto **no** hace, y la pantalla lo dice sin disimularlo: no borra lo que esa
// persona ya se bajó, y no le quita de la cabeza la clave que vio mientras tuvo
// acceso. Por eso al lado va rotar la clave.
func (a *App) QuitarAcceso(titular string) error {
	ref := a.bovedaActiva()
	if ref == "" || a.duenoDeLaActiva() != "" {
		return errors.New("Entra en el proyecto para cambiar quién tiene acceso")
	}
	p := a.boveda()
	if p == nil {
		return boveda.ErrCerrada
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	if err := a.cliente().QuitarAcceso(a.ctxCuenta(), token, ref, titular); err != nil {
		return err
	}
	if err := p.RetirarAcceso(titular); err != nil {
		return err
	}
	a.Actividad()
	return p.OlvidarTitular(titular)
}

// QuienTiene lista a las personas con acceso a la bóveda abierta, **juntando las dos
// listas**: la de la bóveda, que es la que tiene los correos, y la del servidor, que
// es la que de verdad deja entrar.
//
// La ven todos los que tienen acceso, no solo el dueño: lo eligió el cliente, y es lo
// coherente con que quien puede editar pueda dar acceso a más gente — si no, entraría
// gente y nadie se enteraría.
func (a *App) QuienTiene() ([]QuienTieneAcceso, error) {
	p := a.boveda()
	if p == nil {
		return nil, boveda.ErrCerrada
	}
	ref := a.bovedaActiva()
	if ref == "" {
		return nil, errors.New("Tu bóveda personal no se comparte")
	}
	enElServidor := map[string]bool{}
	if token, err := a.sesionDeCuenta(); err == nil && a.duenoDeLaActiva() == "" {
		// Solo el dueño puede preguntarle al servidor por los miembros; quien tiene
		// acceso ve la lista de la bóveda, que es la que lleva los nombres.
		if ms, err := a.cliente().Miembros(a.ctxCuenta(), token, ref); err == nil {
			for _, m := range ms {
				enElServidor[m.Titular] = true
			}
		}
	}
	soyElDueno := a.duenoDeLaActiva() == ""
	out := []QuienTieneAcceso{}
	for _, t := range p.Titulares() {
		out = append(out, QuienTieneAcceso{
			Titular: t.ID, Correo: t.Correo, Permiso: t.Permiso, Desde: t.Desde,
			// Para quien no es el dueño no hay nada que comparar, así que no se
			// enseña una diferencia que no puede comprobar.
			EnElServidor: !soyElDueno || enElServidor[t.ID],
		})
	}
	return out, nil
}

// AceptarAcceso mete en mi lista una bóveda que me han compartido y **se la baja**.
//
// Lo que llega al buzón no entra solo, igual que una copia: si entrara, cualquiera
// que supiera mi correo me metería un proyecto en la lista.
func (a *App) AceptarAcceso(id string) error {
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
		acceso, de, err := a.abrirAccesoDelBuzon(x.Sobre)
		if err != nil {
			return err
		}
		if err := a.conLaPersonal(func(b *boveda.Boveda) error {
			return b.PonerCompartida(boveda.Compartida{
				Dueno: acceso.Dueno, Ref: acceso.Ref, Nombre: acceso.Nombre,
				Titular: acceso.Titular, Permiso: acceso.Permiso, Huella: de.Huella,
			})
		}); err != nil {
			return err
		}
		if err := a.cliente().TirarDelBuzon(a.ctxCuenta(), token, id); err != nil {
			return err
		}
		// Y se trae el fichero. **El acceso se apunta antes de tirar el sobre**: si se
		// tirara primero y fallara lo de arriba, no habría forma de volver a saber
		// dónde está esa bóveda. Que la descarga falle se puede reintentar.
		if err := a.BajarCompartida(acceso.Dueno, acceso.Ref); err != nil {
			return fmt.Errorf("El acceso está guardado, pero la bóveda no se ha podido traer: %w", err)
		}
		return nil
	}
	return errors.New("Ese envío ya no está en el buzón")
}

// abrirAccesoDelBuzon abre un sobre de acceso **con la identidad de la personal**.
func (a *App) abrirAccesoDelBuzon(crudo json.RawMessage) (boveda.Acceso, boveda.Identidad, error) {
	var s boveda.Envio
	if err := json.Unmarshal(crudo, &s); err != nil {
		return boveda.Acceso{}, boveda.Identidad{}, errors.New("Este envío no se entiende")
	}
	var acc boveda.Acceso
	var de boveda.Identidad
	err := a.conLaPersonal(func(b *boveda.Boveda) error {
		var err error
		acc, de, err = b.AbrirAcceso(s)
		return err
	})
	return acc, de, err
}
