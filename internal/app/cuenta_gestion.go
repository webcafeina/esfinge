package app

// Lo que se hace con la cuenta una vez que existe (la A3): cambiar la contraseña,
// recuperarla sin ningún equipo a mano, ver y olvidar equipos, y borrar o exportar
// la cuenta entera.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/cuenta"
	"github.com/webcafeina/esfinge/internal/escritura"
	"github.com/webcafeina/esfinge/internal/sincro"
)

// EquipoDeCuenta es un equipo con sesión en la cuenta, para enseñarlo en Ajustes.
type EquipoDeCuenta struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	// Visto es cuándo se usó por última vez, en RFC3339.
	Visto  string `json:"visto"`
	Actual bool   `json:"actual"`
}

func (a *App) sesionDeCuenta() (string, error) {
	if leerDatosCuenta().Modo != "cuenta" {
		return "", errors.New("Este equipo no está en ninguna cuenta")
	}
	a.cu.mu.Lock()
	token := a.cu.sesion
	a.cu.mu.Unlock()
	if token == "" {
		return "", errors.New("Abre la bóveda de la cuenta para hacer esto")
	}
	return token, nil
}

// ------------------------------------------------------------------ cambiar la contraseña

// cambiarMaestraEnLaCuenta cambia la contraseña de la cuenta, que es la maestra de
// la bóveda, **en el servidor y aquí, y en ese orden** (ADR 0037):
//
//  1. Se sincroniza, para escribir sobre la última versión.
//  2. Se prepara la bóveda con la ranura nueva **sin tocar el fichero de aquí**, y
//     se sube con la clave de acceso nueva: bóveda y verificador cambian a la vez
//     en el servidor, y las demás sesiones se cierran.
//  3. Solo entonces se pone la ranura nueva aquí.
//
// Si el servidor dice que no, aquí no ha cambiado nada. Mientras tanto la
// sincronización de fondo se para: subiría en medio y haría fallar el paso 2.
func (a *App) cambiarMaestraEnLaCuenta(b *boveda.Boveda, nueva string) error {
	if err := maestraSirveParaCuenta(nueva); err != nil {
		return err
	}
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	a.pararSincro()
	defer a.arrancarSincro(b)

	ctx, cancelar := context.WithTimeout(a.ctxCuenta(), time.Minute)
	defer cancelar()
	s := a.nuevoSincronizador(b)
	memoria := sincro.JuntoALaBoveda{Ruta: b.Ruta()}
	for intento := 0; intento < 3; intento++ {
		if _, err := s.Sincronizar(ctx); err != nil {
			return fmt.Errorf("Hace falta conexión con el servidor para cambiar la contraseña de la cuenta: %w", err)
		}
		recuerdo, _, err := memoria.Cargar()
		if err != nil {
			return err
		}
		datos, maestra, err := b.SubidaConMaestra(nueva, recuerdo.Version+1)
		if err != nil {
			return err
		}
		sal, err := cripto.Azar(16)
		if err != nil {
			return err
		}
		clave, err := cuenta.DerivarAcceso(nueva, sal, cuenta.PorDefecto)
		if err != nil {
			return err
		}
		posesion, err := b.Posesion()
		if err != nil {
			return err
		}
		version, _, err := a.cliente().CambiarClave(ctx, token, cuenta.CambioDeClave{
			Posesion: posesion, Sal: sal, Argon2: cuenta.PorDefecto, ClaveDeAcceso: clave,
			Version: recuerdo.Version, Documento: datos,
		})
		if _, conflicto := cuenta.Conflicto(err); conflicto {
			continue // otro equipo ha subido en medio: se vuelve a sincronizar
		}
		if err != nil {
			return err
		}
		serie, err := b.PonerMaestra(maestra)
		if err != nil {
			return err
		}
		return memoria.Guardar(sincro.Recuerdo{Version: version, Serie: serie}, datos)
	}
	return errors.New("Otros equipos están cambiando la bóveda a la vez; prueba otra vez en un momento")
}

// ------------------------------------------------------------------ recuperar

// EmpezarRecuperacion manda un código al correo para recuperar la cuenta. El
// servidor contesta lo mismo haya cuenta o no.
func (a *App) EmpezarRecuperacion(correo string) error {
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return err
	}
	return a.cliente().EmpezarRecuperacion(a.ctxCuenta(), c)
}

// TerminarRecuperacion recupera la cuenta **sin ningún equipo a mano**: con el
// código del correo y la clave de recuperación se demuestra al servidor que se
// tiene la bóveda, se pone una contraseña nueva —en el servidor y en la bóveda a
// la vez— y la bóveda de la cuenta queda abierta en este equipo, como al entrar.
//
// Webcafeína no puede hacer esto por nadie: sin la clave de recuperación, el
// servidor solo tiene una bóveda cifrada que no sabe abrir.
func (a *App) TerminarRecuperacion(correo, codigo, clave, nueva string) (ResultadoEntrada, error) {
	c, err := cuenta.NormalizarCorreo(correo)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	if d := leerDatosCuenta(); d.Modo == "cuenta" && d.Correo != c {
		return ResultadoEntrada{}, errors.New("Este equipo está en otra cuenta")
	}
	if err := maestraSirveParaCuenta(nueva); err != nil {
		return ResultadoEntrada{}, err
	}
	ctx := a.ctxCuenta()
	cli := a.cliente()
	reto, sobre, err := cli.ComprobarRecuperacion(ctx, c, codigo)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	llave, err := boveda.LlaveDeRecuperacion(sobre.Contenedor, clave)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	posesion, err := boveda.PosesionDeLlave(llave)
	cripto.Borrar(llave)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	nombre := nombreDelEquipo()
	restringida, err := cli.TerminarRecuperacion(ctx, reto, posesion, nombre)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	datos, version, _, err := cli.Bajar(ctx, restringida.Token, 0)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	remota, err := boveda.AbrirEnMemoria(datos, clave)
	if err != nil {
		return ResultadoEntrada{}, errors.New("La clave de recuperación no abre la bóveda de la cuenta")
	}
	subida, _, err := remota.SubidaConMaestra(nueva, version+1)
	remota.Cerrar()
	if err != nil {
		return ResultadoEntrada{}, err
	}
	sal, err := cripto.Azar(16)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	acceso, err := cuenta.DerivarAcceso(nueva, sal, cuenta.PorDefecto)
	if err != nil {
		return ResultadoEntrada{}, err
	}
	_, normal, err := cli.CambiarClave(ctx, restringida.Token, cuenta.CambioDeClave{
		Posesion: posesion, Sal: sal, Argon2: cuenta.PorDefecto, ClaveDeAcceso: acceso,
		Version: version, Documento: subida,
	})
	if err != nil {
		return ResultadoEntrada{}, err
	}
	// Desde aquí es como entrar con la contraseña nueva: baja la bóveda, y si este
	// equipo ya tenía la suya la pone al día o pregunta qué hacer con otra.
	return a.terminarEntrada(
		&entradaPendiente{correo: c, maestra: nueva, nombre: nombre},
		cuenta.Sesion{Token: normal, Dispositivo: restringida.Dispositivo},
	)
}

// ------------------------------------------------------------------ equipos

// DispositivosDeCuenta lista los equipos con sesión en la cuenta, el de ahora
// marcado.
func (a *App) DispositivosDeCuenta() ([]EquipoDeCuenta, error) {
	token, err := a.sesionDeCuenta()
	if err != nil {
		return nil, err
	}
	es, err := a.cliente().Equipos(a.ctxCuenta(), token)
	if err != nil {
		return nil, err
	}
	out := make([]EquipoDeCuenta, 0, len(es))
	for _, e := range es {
		out = append(out, EquipoDeCuenta{
			ID: e.ID, Nombre: e.Nombre, Actual: e.Actual,
			Visto: time.UnixMilli(e.Visto).UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

// OlvidarDispositivo le quita la sesión a un equipo —uno perdido, uno que ya no se
// usa—: tendrá que volver a entrar con la contraseña y un código. Lo que tuviera
// en su disco se queda allí, cifrado.
func (a *App) OlvidarDispositivo(id string) error {
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	return a.cliente().OlvidarEquipo(a.ctxCuenta(), token, id)
}

// ------------------------------------------------------------------ borrar y exportar

// PedirCodigoParaBorrarCuenta manda al correo el código que hace falta para borrar
// la cuenta. El reto se queda aquí, en Go.
func (a *App) PedirCodigoParaBorrarCuenta() error {
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	reto, err := a.cliente().PedirBorrado(a.ctxCuenta(), token)
	if err != nil {
		return err
	}
	a.cu.mu.Lock()
	a.cu.retoBorrado = reto
	a.cu.mu.Unlock()
	return nil
}

// BorrarCuenta borra la cuenta **del servidor**, entera y sin vuelta atrás: la
// bóveda de allí con sus versiones, los equipos y el correo. Pide la contraseña y
// el código del correo. **La bóveda de este equipo se queda**, en local; la de los
// otros equipos, también, y dejarán de sincronizarse.
func (a *App) BorrarCuenta(maestra, codigo string) error {
	token, err := a.sesionDeCuenta()
	if err != nil {
		return err
	}
	a.cu.mu.Lock()
	reto := a.cu.retoBorrado
	a.cu.mu.Unlock()
	if reto == "" {
		return errors.New("Pide primero el código para borrar la cuenta")
	}
	d := leerDatosCuenta()
	pre, err := a.cliente().Prelogin(a.ctxCuenta(), d.Correo)
	if err != nil {
		return err
	}
	clave, err := cuenta.DerivarAcceso(maestra, pre.Sal, pre.Argon2)
	if err != nil {
		return err
	}
	// Lo que quede sin subir, antes: por si alguien se arrepiente y vuelve a crear
	// la cuenta, que no falte nada en esta bóveda. Da igual si falla.
	a.cu.mu.Lock()
	m := a.cu.marcha
	a.cu.mu.Unlock()
	if m != nil {
		_ = m.s.Vaciar(vaciarAlCerrar)
	}
	if err := a.cliente().BorrarCuenta(a.ctxCuenta(), token, clave, reto, codigo); err != nil {
		return err
	}
	a.cu.mu.Lock()
	a.cu.retoBorrado = ""
	a.cu.mu.Unlock()
	return a.olvidarLaCuentaAqui()
}

// ExportarDatosDeCuenta guarda en un fichero lo que el servidor tiene de la cuenta
// —el correo, los equipos, los avisos y la bóveda **tal cual está allí: cifrada**—
// y devuelve dónde. Vacío si se cancela el diálogo.
func (a *App) ExportarDatosDeCuenta() (string, error) {
	token, err := a.sesionDeCuenta()
	if err != nil {
		return "", err
	}
	datos, err := a.cliente().Exportar(a.ctxCuenta(), token)
	if err != nil {
		return "", err
	}
	destino, err := a.sistema.ElegirDondeGuardar("Guardar lo que hay de tu cuenta", "esfinge-cuenta.json", "")
	if err != nil || destino == "" {
		return "", err
	}
	err = escritura.Atomica(destino, escritura.Opciones{}, func(w io.Writer) error {
		_, err := w.Write(datos)
		return err
	})
	if err != nil {
		return "", err
	}
	_ = os.Chmod(destino, 0o600)
	return destino, nil
}
