package cli

// La cuenta de Esfinge desde la línea de comandos, **solo para un servidor** (ADR
// 0057).
//
// Esto existe por una razón sola: un servidor tiene que **recibir** las bóvedas de
// proyecto que se le marquen, y recibir es sincronizar. Y sincronizar necesita una
// cuenta, porque el VPS es otra persona (ADR 0052) y no otro equipo de nadie —si
// fuera otro equipo, entrar le bajaría la bóveda personal entera, con todos los
// secretos de la empresa, que es justo lo que no se quiere—.
//
// **No es la gestión de cuentas de la ventana ni lo pretende.** Aquí no se cambia
// la maestra, ni se recupera, ni se comparte, ni se borra la cuenta: para eso está
// la aplicación, y lo dice la ADR 0054 de los agentes con las mismas palabras.
// Aquí solo: dar de alta la del servidor, entrar y ponerse al día.
//
// Dos cosas que no son detalles:
//
//   - **La sesión y la clave de acceso van selladas con la clave de la bóveda**, en
//     un fichero propio (`servidor.json`). No se meten en el `cuenta.json` de la
//     ventana a propósito: el comentario de aquél dice que sin la maestra un disco
//     robado no habla con el servidor, y eso **sigue siendo verdad en un Mac**.
//     Aquí no lo es, y la diferencia tiene que estar donde se ve.
//   - **Se guarda la clave de acceso, nunca la maestra.** La sesión caduca —30 días
//     deslizantes, 90 como máximo— y volver a entrar exige esa clave; sin ella,
//     alguien tendría que entrar por SSH cada tres meses y el fallo de olvidarse es
//     mudo: el servidor sigue abriendo su copia y las credenciales se quedan viejas.
//     La clave de acceso **no abre el fichero de la bóveda**: abre la cuenta, donde
//     el permiso es de solo ver. Lo eligió el cliente con el coste delante.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/cuenta"
	"github.com/webcafeina/esfinge/internal/escritura"
	"github.com/webcafeina/esfinge/internal/salida"
	"github.com/webcafeina/esfinge/internal/sincro"
)

// raizDeCuentas es dónde vive el servidor de cuentas.
//
// Variable y no constante: es la costura por la que las pruebas apuntan al Worker
// levantado en local, igual que `azar` en `cripto` y `ApuntarAAPI` en la ventana.
// **No sale de una variable de entorno**: dejar que el entorno diga a qué servidor
// se le manda la clave de acceso sería abrir una puerta por comodidad.
var raizDeCuentas = cuenta.RaizPorDefecto

// elPlazoDeLaRed es lo que se espera a una pasada entera. Un servidor no tiene a
// nadie que mire si se ha colgado.
const elPlazoDeLaRed = 2 * time.Minute

// datosDelServidor es lo que este equipo guarda para poder hablar con su cuenta
// **él solo**.
type datosDelServidor struct {
	Correo string `json:"correo"`
	Cuenta string `json:"cuenta,omitempty"`
	Equipo string `json:"equipo,omitempty"`
	// Sesion, Acceso y Confianza van **sellados con la clave de la bóveda**, así que
	// cuelgan de la llave del servidor y no son una segunda llave en el disco.
	//
	// Acceso es la clave de acceso de la cuenta, lo único que hace falta para volver
	// a entrar cuando la sesión caduque. Confianza es el testigo de «este equipo es
	// de confianza», que es lo que ahorra el código del correo — sin él, volver a
	// entrar exigiría que alguien leyera un correo, o sea que no se podría.
	Sesion    string `json:"sesion,omitempty"`
	Acceso    string `json:"acceso,omitempty"`
	Confianza string `json:"confianza,omitempty"`
}

func rutaDatosDelServidor(rutaBoveda string) string {
	return filepath.Join(filepath.Dir(rutaBoveda), "servidor.json")
}

func leerDatosDelServidor(rutaBoveda string) datosDelServidor {
	var d datosDelServidor
	crudo, err := os.ReadFile(rutaDatosDelServidor(rutaBoveda))
	if err != nil {
		return d
	}
	_ = json.Unmarshal(crudo, &d)
	return d
}

func guardarDatosDelServidor(rutaBoveda string, d datosDelServidor) error {
	crudo, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	crudo = append(crudo, '\n')
	return escritura.Atomica(rutaDatosDelServidor(rutaBoveda),
		escritura.Opciones{CrearCarpeta: true}, func(w io.Writer) error {
			_, err := w.Write(crudo)
			return err
		})
}

// laCuenta es la cuenta de este servidor, con la bóveda abierta delante.
//
// Existe para una cosa que no se puede hacer sin estado: **volver a entrar cuando
// la sesión ha caducado**. Eso solo se sabe cuando el servidor contesta 401, así
// que quien pide algo lo intenta, y si era eso, se vuelve a entrar y se repite.
type laCuenta struct {
	c     *cuenta.Cliente
	b     *boveda.Boveda
	ruta  string
	d     datosDelServidor
	token string
}

func abrirLaCuenta(b *boveda.Boveda, ruta string) (*laCuenta, error) {
	d := leerDatosDelServidor(ruta)
	if d.Correo == "" || d.Acceso == "" {
		return nil, errors.New(
			"Este equipo no está en ninguna cuenta; entra con «esfinge boveda servidor entrar» o créala con «alta»")
	}
	c := cuenta.Nuevo(raizDeCuentas)
	s := &laCuenta{c: c, b: b, ruta: ruta, d: d}
	if d.Sesion != "" {
		if claro, err := b.AbrirSecreto(d.Sesion); err == nil {
			s.token = string(claro)
		}
	}
	return s, nil
}

// conLaSesion hace algo con el testigo y, si había caducado, **vuelve a entrar y lo
// repite una vez**.
//
// Una sola vez a propósito: si tras entrar de nuevo sigue diciendo que la sesión no
// vale, lo que hay es un problema y no una sesión vieja, y reintentar en bucle lo
// taparía.
func (s *laCuenta) conLaSesion(ctx context.Context, hacer func(token string) error) error {
	if s.token != "" {
		err := hacer(s.token)
		if err == nil || !cuenta.SesionCaducada(err) {
			return err
		}
	}
	if err := s.volverAEntrar(ctx); err != nil {
		return err
	}
	return hacer(s.token)
}

// volverAEntrar entra otra vez con la clave de acceso guardada y el testigo de
// confianza, que es lo que ahorra el código del correo.
//
// **Y la confianza se renueva al entrar**, así que mientras este equipo trabaje al
// menos una vez cada noventa días se mantiene solo. Parado más tiempo, caduca y
// hace falta `entrar` a mano.
func (s *laCuenta) volverAEntrar(ctx context.Context) error {
	acceso, err := s.b.AbrirSecreto(s.d.Acceso)
	if err != nil {
		return fmt.Errorf("No he podido leer la clave de la cuenta de este equipo: %w", err)
	}
	defer cripto.Borrar(acceso)
	confianza := ""
	if s.d.Confianza != "" {
		if claro, err := s.b.AbrirSecreto(s.d.Confianza); err == nil {
			confianza = string(claro)
		}
	}
	sesion, _, err := s.c.Entrar(ctx, s.d.Correo, acceso, nombreDeEsteEquipo(), confianza)
	if err != nil {
		return err
	}
	if sesion == nil {
		// El servidor pide un código del correo: este equipo ya no es de confianza
		// —noventa días sin trabajar— y aquí no hay nadie que lea un correo.
		return errors.New(
			"La sesión de este equipo ha caducado y el servidor pide un código del correo; " +
				"entra a mano con «esfinge boveda servidor entrar»")
	}
	return s.quedarseCon(*sesion)
}

// quedarseCon sella lo que el servidor acaba de dar y lo guarda.
func (s *laCuenta) quedarseCon(sesion cuenta.Sesion) error {
	s.token = sesion.Token
	sellada, err := s.b.SellarSecreto([]byte(sesion.Token))
	if err != nil {
		return err
	}
	s.d.Sesion = sellada
	if sesion.Cuenta != "" {
		s.d.Cuenta = sesion.Cuenta
	}
	if sesion.Dispositivo != "" {
		s.d.Equipo = sesion.Dispositivo
	}
	if sesion.Confianza != "" {
		sellada, err := s.b.SellarSecreto([]byte(sesion.Confianza))
		if err != nil {
			return err
		}
		s.d.Confianza = sellada
	}
	return guardarDatosDelServidor(s.ruta, s.d)
}

// nombreDeEsteEquipo es lo que el servidor enseña en su lista de equipos. El del
// ordenador, que en un servidor es justo lo que se quiere leer ahí.
func nombreDeEsteEquipo() string {
	n, err := os.Hostname()
	n = strings.TrimSuffix(strings.TrimSpace(n), ".local")
	if err != nil || n == "" {
		return "Servidor sin nombre"
	}
	return n
}

// leerLinea pregunta algo que **no es un secreto** —un correo, un código del
// correo— y por eso se ve al teclearlo. El prompt va a la salida de error, como el
// de la clave: lo estándar es de quien está al otro lado de la tubería.
func leerLinea(e salida.Estilos, etiqueta string) (string, error) {
	fmt.Fprint(os.Stderr, e.Acento.Render(salida.GlifoBarra)+" "+e.Cuerpo.Render(etiqueta)+e.Apagado.Render(": "))
	linea, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && linea == "" {
		return "", fmt.Errorf("No he podido leer %s: %w", strings.ToLower(etiqueta), err)
	}
	linea = strings.TrimSpace(linea)
	if linea == "" {
		return "", fmt.Errorf("Hace falta %s", strings.ToLower(etiqueta))
	}
	return linea, nil
}

// ------------------------------------------------------------------ sincronizar

// comandoServidorSincronizar pone este equipo al día **en una pasada**, no en un
// bucle.
//
// Es lo que hace que una credencial cambiada en el Mac llegue aquí, y por eso
// `ejecutar` y los demás la llaman antes de abrir: una credencial vieja en un
// servidor es un fallo mudo —el despliegue falla con un testigo caducado y nada
// dice por qué—, así que lo correcto es pagar unos segundos.
func comandoServidorSincronizar(o *opciones, ob *opcionesBoveda) *cobra.Command {
	return &cobra.Command{
		Use:   "sincronizar",
		Short: "Trae los cambios de la cuenta y sube los de aquí",
		Long: "Hace **una pasada** de sincronización con la cuenta de este equipo y\n" +
			"vuelve. No se queda corriendo: en un servidor no hay nadie que vuelva a\n" +
			"teclear nada, así que no puede haber un proceso con la bóveda abierta.\n\n" +
			"Los demás comandos la hacen por su cuenta antes de abrir la bóveda; ésta\n" +
			"está para poder pedirla a mano y ver qué dice.",
		Example:       "  esfinge boveda servidor sincronizar",
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

			r, err := unaPasada(b)
			if err != nil {
				return err
			}
			if ob.json {
				return escribirJSON(map[string]any{
					"version": r.Version, "bajo": r.Bajo, "subio": r.Subio,
					"traidas": r.Fusion.Traidas, "borradas": r.Fusion.Borradas,
					"conflictos": r.Fusion.Conflictos, "intentos": r.Intentos,
				})
			}
			if !o.silencio {
				switch {
				case r.Bajo && r.Subio:
					fmt.Fprintln(os.Stderr, e.Ok("Al día: ha llegado algo de otro equipo y ha subido lo de aquí."))
				case r.Bajo:
					fmt.Fprintln(os.Stderr, e.Ok("Al día: ha llegado algo de otro equipo."))
				case r.Subio:
					fmt.Fprintln(os.Stderr, e.Ok("Al día: ha subido lo de aquí."))
				default:
					fmt.Fprintln(os.Stderr, e.Info("Ya estaba al día."))
				}
			}
			return nil
		},
	}
}

// unaPasada sincroniza la bóveda que se le dé, volviendo a entrar si la sesión
// había caducado.
//
// **No hay vigilante ni bucle**, al contrario que en la ventana: aquí el proceso
// vive lo que dura la orden. Y la memoria de la fusión es la misma que usa la
// ventana (`JuntoALaBoveda`), para que los dos lados no se peleen si alguna vez
// corren en la misma máquina.
func unaPasada(b *boveda.Boveda) (sincro.Resultado, error) {
	s, err := abrirLaCuenta(b, b.Ruta())
	if err != nil {
		return sincro.Resultado{}, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), elPlazoDeLaRed)
	defer cancelar()

	var r sincro.Resultado
	err = s.conLaSesion(ctx, func(token string) error {
		sin := &sincro.Sincronizador{
			Servidor: s.c,
			Boveda:   b,
			Memoria:  sincro.JuntoALaBoveda{Ruta: b.Ruta()},
			Token:    func() string { return token },
		}
		var err error
		r, err = sin.Sincronizar(ctx)
		return err
	})
	return r, err
}

// ------------------------------------------------------------------ alta

// comandoServidorAlta crea la cuenta de este servidor **y su bóveda**.
//
// Es la primera vez que la línea de comandos crea una bóveda, y hace falta: un
// servidor no tiene ventana, y sin bóveda personal no tiene dónde vivir su
// identidad — que es lo que abre las bóvedas que le compartan (ADR 0052).
//
// Su bóveda nace **vacía y así se queda**: lo que el servidor usa son las de
// proyecto que se le marquen, y se abren a través de ésta (ADR 0050). Eso es lo que
// hace que haya **una sola llave en el disco** y no una por bóveda.
//
// El orden es el mismo que en la ventana y por la misma razón: **la bóveda se crea
// en memoria y solo se guarda si el servidor dice que sí**. Guardada antes, un
// código mal escrito dejaría una bóveda cuya clave de recuperación no ha visto
// nadie.
func comandoServidorAlta(o *opciones, ob *opcionesBoveda) *cobra.Command {
	var correo string
	cmd := &cobra.Command{
		Use:   "alta --correo DIRECCIÓN",
		Short: "Crea la cuenta de este servidor y su bóveda",
		Long: "Da de alta una cuenta de Esfinge para este equipo y le crea su bóveda,\n" +
			"que se queda vacía: lo que el servidor usa son las bóvedas de proyecto\n" +
			"que se le marquen desde la aplicación, y se abren a través de ésta.\n\n" +
			"**Es la cuenta del servidor, no la tuya.** Si fuera la tuya, este equipo\n" +
			"se bajaría tu bóveda personal entera.\n\n" +
			"Manda un código a ese correo y lo pide. Escribe también la clave de\n" +
			"recuperación, **una sola vez**: apúntala fuera de este equipo.\n\n" +
			"Después hace falta «servidor activar» para que la bóveda se abra aquí sin\n" +
			"contraseña: esto no lo activa solo.",
		Example:       "  esfinge boveda servidor alta --correo servidor@webcafeina.com",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			e, err := estilos(o.tema)
			if err != nil {
				return err
			}
			c, err := cuenta.NormalizarCorreo(correo)
			if err != nil {
				return err
			}
			ruta := ob.ruta
			if ruta == "" {
				ruta = rutaDeLaBoveda()
			}
			if ruta == "" {
				return errors.New("No sé dónde poner la bóveda en este sistema; dilo con --boveda")
			}
			// **Antes de pedirle nada a nadie**: si ya hay una bóveda aquí, esto no es
			// lo que hace falta. Es la regla de `ExportarLlaves`.
			if _, err := os.Stat(ruta); err == nil {
				return fmt.Errorf("Ya hay una bóveda en %s; para entrar en una cuenta que ya existe usa «servidor entrar»", ruta)
			}

			maestra, err := ob.clave.Leer(e, true)
			if err != nil {
				return err
			}
			defer cripto.Borrar(maestra)
			if err := cuenta.MaestraSirve(string(maestra)); err != nil {
				return err
			}

			cliente := cuenta.Nuevo(raizDeCuentas)
			ctx, cancelar := context.WithTimeout(context.Background(), elPlazoDeLaRed)
			defer cancelar()
			if err := cliente.EmpezarAlta(ctx, c); err != nil {
				return err
			}
			if !o.silencio {
				fmt.Fprintln(os.Stderr, e.Info("Le he mandado un código a "+c+"."))
			}
			codigo, err := leerLinea(e, "Código del correo")
			if err != nil {
				return err
			}

			b, recuperacion, err := boveda.CrearEnMemoria(string(maestra))
			if err != nil {
				return err
			}
			sesion, err := altaEnElServidor(ctx, cliente, b, c, codigo, string(maestra))
			if err != nil {
				return err
			}
			// Dicho sí el servidor, ya se puede guardar.
			if err := b.GuardarEn(ruta); err != nil {
				return err
			}
			defer b.Cerrar()

			acceso, err := claveDeAccesoDe(ctx, cliente, c, string(maestra))
			if err != nil {
				return err
			}
			defer cripto.Borrar(acceso)
			if err := guardarLaCuentaDelServidor(b, ruta, c, acceso, sesion); err != nil {
				return err
			}
			// **Las llaves se publican ya**, no cuando se vaya a compartir algo: para
			// que *otro* pueda darle acceso hacen falta las llaves de éste, y quien no
			// ha mandado nada en su vida no habría entrado nunca a publicarlas. Es la
			// regla que costó un sobre cifrado hacia nadie.
			publicarLasLlaves(ctx, cliente, b, sesion.Token)

			fmt.Fprintln(os.Stderr, e.Ok("Cuenta creada para "+c+", con su bóveda en "+ruta+"."))
			fmt.Fprintln(os.Stderr, e.Ojo("Apunta la clave de recuperación FUERA de este equipo. No se vuelve a enseñar."))
			// Por la salida estándar, que es lo que se puede redirigir a un sitio
			// seguro sin llevarse los mensajes de cortesía por delante.
			fmt.Println(recuperacion)
			if !o.silencio {
				fmt.Fprintln(os.Stderr, e.Info("Ahora: «esfinge boveda servidor activar», para que se abra aquí sin contraseña."))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&correo, "correo", "", "Dirección de la cuenta de este servidor")
	_ = cmd.MarkFlagRequired("correo")
	return cmd
}

// altaEnElServidor es la parte que habla con el servidor, con su propia sal.
func altaEnElServidor(ctx context.Context, c *cuenta.Cliente, b *boveda.Boveda,
	correo, codigo, maestra string) (cuenta.Sesion, error) {
	sal, err := cripto.Azar(16)
	if err != nil {
		return cuenta.Sesion{}, err
	}
	clave, err := cuenta.DerivarAcceso(maestra, sal, cuenta.PorDefecto)
	if err != nil {
		return cuenta.Sesion{}, err
	}
	defer cripto.Borrar(clave)
	posesion, err := b.Posesion()
	if err != nil {
		return cuenta.Sesion{}, err
	}
	// **Las llaves no van aquí.** La identidad se crea al pedirla y eso hay que
	// guardarlo, y esta bóveda todavía está en memoria: se publican después de
	// guardarla, que es lo que hace la ventana y por lo mismo.
	return c.TerminarAlta(ctx, cuenta.Alta{
		Correo: correo, Codigo: codigo, Sal: sal, Argon2: cuenta.PorDefecto,
		ClaveDeAcceso: clave, Posesion: posesion,
		Dispositivo: nombreDeEsteEquipo(), Confiar: true,
	})
}

// ------------------------------------------------------------------ entrar

// comandoServidorEntrar entra en una cuenta que ya existe, **a mano y una vez**.
//
// Hace falta en dos momentos: al poner un servidor nuevo con una cuenta que ya
// estaba, y el día que este equipo haya estado parado más de noventa días y su
// testigo de confianza haya caducado — entonces el servidor pide un código del
// correo y aquí no hay nadie que lo lea.
func comandoServidorEntrar(o *opciones, ob *opcionesBoveda) *cobra.Command {
	var correo string
	cmd := &cobra.Command{
		Use:   "entrar --correo DIRECCIÓN",
		Short: "Entra en la cuenta de este servidor",
		Long: "Entra en una cuenta que ya existe y deja este equipo listo para\n" +
			"sincronizar él solo.\n\n" +
			"Guarda la clave de acceso de la cuenta **sellada con la clave de la\n" +
			"bóveda**, que es lo que le permite volver a entrar cuando la sesión\n" +
			"caduque sin que nadie tenga que venir. No guarda la contraseña maestra.\n\n" +
			"Casi siempre pide un código del correo; con el equipo ya de confianza, no.",
		Example:       "  esfinge boveda servidor entrar --correo servidor@webcafeina.com",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			e, err := estilos(o.tema)
			if err != nil {
				return err
			}
			c, err := cuenta.NormalizarCorreo(correo)
			if err != nil {
				return err
			}
			ruta, err := rutaElegida(ob)
			if err != nil {
				return err
			}
			// La bóveda tiene que estar, y se abre con lo que haya: la llave de este
			// equipo si ya está activada, o la maestra.
			b, err := abrirLaBoveda(o, ob)
			if err != nil {
				return err
			}
			defer b.Cerrar()

			// **La contraseña se pide aparte aunque la bóveda ya esté abierta**, porque
			// de ella sale la clave de acceso de la cuenta y no de la clave de bóveda.
			// Con la llave del servidor puesta, la bóveda se abre sin teclear nada y
			// aquí hay que teclear algo igual: es lo que impide poner este equipo en una
			// cuenta sin saber su contraseña.
			maestra, err := ob.clave.Leer(e, false)
			if err != nil {
				return err
			}
			defer cripto.Borrar(maestra)

			cliente := cuenta.Nuevo(raizDeCuentas)
			ctx, cancelar := context.WithTimeout(context.Background(), elPlazoDeLaRed)
			defer cancelar()
			acceso, err := claveDeAccesoDe(ctx, cliente, c, string(maestra))
			if err != nil {
				return err
			}
			defer cripto.Borrar(acceso)

			d := leerDatosDelServidor(ruta)
			confianza := ""
			if d.Confianza != "" {
				if claro, err := b.AbrirSecreto(d.Confianza); err == nil {
					confianza = string(claro)
				}
			}
			sesion, reto, err := cliente.Entrar(ctx, c, acceso, nombreDeEsteEquipo(), confianza)
			if err != nil {
				return err
			}
			if sesion == nil {
				if !o.silencio {
					fmt.Fprintln(os.Stderr, e.Info("Le he mandado un código a "+c+"."))
				}
				codigo, err := leerLinea(e, "Código del correo")
				if err != nil {
					return err
				}
				// **Confiar en este equipo**, que no es una comodidad: sin el testigo de
				// confianza, volver a entrar exigiría que alguien leyera un correo, y
				// aquí no hay nadie.
				s, err := cliente.ConfirmarEntrada(ctx, reto, codigo, true)
				if err != nil {
					return err
				}
				sesion = &s
			}
			if err := guardarLaCuentaDelServidor(b, ruta, c, acceso, *sesion); err != nil {
				return err
			}
			publicarLasLlaves(ctx, cliente, b, sesion.Token)

			if !o.silencio {
				fmt.Fprintln(os.Stderr, e.Ok("Este equipo está en la cuenta de "+c+"."))
				fmt.Fprintln(os.Stderr, e.Info("Volverá a entrar él solo mientras trabaje al menos una vez cada noventa días."))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&correo, "correo", "", "Dirección de la cuenta de este servidor")
	_ = cmd.MarkFlagRequired("correo")
	return cmd
}

// claveDeAccesoDe deriva la clave de acceso con la sal y el coste que diga el
// servidor para esa cuenta.
//
// **El coste lo valida `Prelogin`**, no esto: uno que pidiera una pasada y 8 MiB
// haría que la clave de acceso se pudiera atacar sin conexión mucho más deprisa que
// la ranura de la bóveda, y eso está escrito en `internal/cuenta`.
func claveDeAccesoDe(ctx context.Context, c *cuenta.Cliente, correo, maestra string) ([]byte, error) {
	p, err := c.Prelogin(ctx, correo)
	if err != nil {
		return nil, err
	}
	return cuenta.DerivarAcceso(maestra, p.Sal, p.Argon2)
}

// guardarLaCuentaDelServidor sella la sesión, la clave de acceso y la confianza, y
// las deja en `servidor.json`.
//
// **La clave de acceso se sella con la clave de la bóveda**, así que cuelga de la
// llave de este equipo y no es una segunda llave en el disco. Lo que abre no es el
// fichero de la bóveda: es la cuenta, donde el permiso es de solo ver.
func guardarLaCuentaDelServidor(b *boveda.Boveda, ruta, correo string, acceso []byte, sesion cuenta.Sesion) error {
	selladaAcceso, err := b.SellarSecreto(acceso)
	if err != nil {
		return err
	}
	d := leerDatosDelServidor(ruta)
	d.Correo, d.Acceso = correo, selladaAcceso
	s := &laCuenta{b: b, ruta: ruta, d: d}
	return s.quedarseCon(sesion)
}

// publicarLasLlaves deja en el servidor la parte pública de la identidad de esta
// bóveda, **de cortesía**.
//
// Va aquí y no cuando se vaya a compartir algo por la regla que ya costó un sobre
// cifrado hacia nadie: **para poder recibir hay que haber publicado**, y quien no
// manda nunca no pasaría nunca por ahí. Sin estas llaves, quien le dé acceso a este
// servidor sella el sobre hacia unas llaves inventadas y el acceso no se puede abrir
// — los dos lados haciendo lo suyo bien y nadie pudiendo entender por qué.
func publicarLasLlaves(ctx context.Context, c *cuenta.Cliente, b *boveda.Boveda, token string) {
	i, err := b.Identidad()
	if err != nil {
		return
	}
	_ = c.PublicarLlaves(ctx, token, cuenta.Llaves{Suite: i.Suite, Cifrado: i.Cifrado, Firma: i.Firma})
}

// ponerseAlDia hace una pasada al abrir, **en un servidor y nada más**.
//
// Lo decide `servidor.json`: donde no hay cuenta de servidor —o sea en cualquier
// Mac— esto no hace nada y no cambia nada de lo de siempre.
//
// **Si falla, se dice y se sigue.** Las dos salidas peores son las evidentes: callar
// el fallo deja leyendo una credencial vieja sin que nada lo mencione, que es el
// fallo mudo que esto viene a evitar; y negarse a trabajar deja un servidor parado
// porque la red estaba mal un segundo. Decirlo por la salida de error lo ve también
// un agente, que lee lo que el comando escribe.
func ponerseAlDia(o *opciones, ob *opcionesBoveda, b *boveda.Boveda) {
	if ob.sinSincronizar || b == nil {
		return
	}
	if leerDatosDelServidor(b.Ruta()).Acceso == "" {
		return
	}
	if _, err := unaPasada(b); err != nil {
		if o.silencio {
			return
		}
		if e, err2 := estilos(o.tema); err2 == nil {
			fmt.Fprintln(os.Stderr, e.Ojo("No he podido ponerme al día con la cuenta, así que esto puede estar viejo: "+err.Error()))
		}
	}
}
