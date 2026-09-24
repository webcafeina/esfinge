// Programa de un solo uso para mirar **la invitación en un buzón de verdad**
// (ADR 0043, B3). No se compromete: se borra al terminar.
//
// Aquí el correo se lee de una tabla, así que cómo se ve el botón en Gmail o en
// Outlook —y si cae en spam— no lo dice ninguna prueba. Esto crea una cuenta de
// usar y tirar en producción, le manda una copia a una dirección **sin cuenta** y
// deja que la invitación salga por Resend de verdad.
//
//	go run ./cmd/invitacion-de-prueba paso1 <correo de la cuenta>
//	go run ./cmd/invitacion-de-prueba paso2 <correo> <código> <a quién invitar>
//	go run ./cmd/invitacion-de-prueba borrar1 <correo>
//	go run ./cmd/invitacion-de-prueba borrar2 <correo> <código>
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/webcafeina/esfinge/internal/boveda"
	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/cuenta"
)

const raiz = "https://esfinge-cuentas.webcafeina.com"

// La maestra de la cuenta de usar y tirar. No protege nada que importe: la bóveda
// lleva una entrada inventada y la cuenta se borra al terminar.
const maestra = "una cuenta de usar y tirar para ver la invitacion 2026"

var donde = filepath.Join(os.TempDir(), "esfinge-invitacion-de-prueba")

type estado struct {
	Correo string `json:"correo"`
	Sesion string `json:"sesion"`
	Sal    []byte `json:"sal"`
}

func main() {
	if len(os.Args) < 3 {
		morir(fmt.Errorf("falta qué hacer"))
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()
	c := cuenta.Nuevo(raiz)
	if err := os.MkdirAll(donde, 0o700); err != nil {
		morir(err)
	}

	switch os.Args[1] {
	case "paso1":
		if err := c.EmpezarAlta(ctx, os.Args[2]); err != nil {
			morir(err)
		}
		fmt.Println("Pedido el código de alta. Mira el buzón de", os.Args[2])

	case "paso2":
		correo, codigo, aQuien := os.Args[2], os.Args[3], os.Args[4]
		b, _, err := boveda.Crear(filepath.Join(donde, "boveda.esfinge"), maestra)
		if err != nil {
			morir(err)
		}
		sal, err := cripto.Azar(16)
		if err != nil {
			morir(err)
		}
		clave, err := cuenta.DerivarAcceso(maestra, sal, cuenta.PorDefecto)
		if err != nil {
			morir(err)
		}
		posesion, err := b.Posesion()
		if err != nil {
			morir(err)
		}
		s, err := c.TerminarAlta(ctx, cuenta.Alta{
			Correo: correo, Codigo: codigo, Sal: sal, Argon2: cuenta.PorDefecto,
			ClaveDeAcceso: clave, Posesion: posesion,
			Dispositivo: "Prueba de la invitación", Confiar: true,
		})
		if err != nil {
			morir(err)
		}
		guardar(estado{Correo: correo, Sesion: s.Token, Sal: sal})
		fmt.Println("Cuenta creada.")

		// Las llaves, como las publica la aplicación al arrancar la sincronización.
		i, err := b.Identidad()
		if err != nil {
			morir(err)
		}
		if err := c.PublicarLlaves(ctx, s.Token, cuenta.Llaves{Suite: i.Suite, Cifrado: i.Cifrado, Firma: i.Firma}); err != nil {
			morir(err)
		}

		if err := b.Poner(boveda.Entrada{
			Tipo: boveda.TipoCredencial, Titulo: "Wifi de la oficina",
			Usuario: "invitados", Secreto: "esta-no-es-de-verdad",
		}); err != nil {
			morir(err)
		}
		var e boveda.Entrada
		for _, x := range b.Buscar("Wifi") {
			e = x
		}
		completa, hay := b.Ver(e.ID)
		if !hay {
			morir(fmt.Errorf("no se encuentra la entrada recién puesta"))
		}

		// Y el envío, exactamente como lo hace la ventana.
		l, err := c.LlavesDe(ctx, s.Token, aQuien)
		if err != nil {
			morir(err)
		}
		sobre, err := b.MandarEntrada(completa, boveda.Identidad{Suite: l.Suite, Cifrado: l.Cifrado, Firma: l.Firma})
		if err != nil {
			morir(err)
		}
		if err := c.Mandar(ctx, s.Token, aQuien, sobre); err != nil {
			morir(err)
		}
		fmt.Println("Mandada la copia a", aQuien+". Si no tiene cuenta, ahí va la invitación.")

	case "borrar1":
		e := leer()
		reto, err := c.PedirBorrado(ctx, e.Sesion)
		if err != nil {
			morir(err)
		}
		e.Correo = os.Args[2]
		guardarReto(reto)
		fmt.Println("Pedido el código para borrar la cuenta. Mira el buzón de", e.Correo)

	case "borrar2":
		e := leer()
		clave, err := cuenta.DerivarAcceso(maestra, e.Sal, cuenta.PorDefecto)
		if err != nil {
			morir(err)
		}
		if err := c.BorrarCuenta(ctx, e.Sesion, clave, leerReto(), os.Args[3]); err != nil {
			morir(err)
		}
		_ = os.RemoveAll(donde)
		fmt.Println("Cuenta borrada y lo de aquí, fuera.")

	default:
		morir(fmt.Errorf("no sé qué es %q", os.Args[1]))
	}
}

func guardar(e estado) {
	b, _ := json.Marshal(e)
	if err := os.WriteFile(filepath.Join(donde, "estado.json"), b, 0o600); err != nil {
		morir(err)
	}
}

func leer() estado {
	b, err := os.ReadFile(filepath.Join(donde, "estado.json"))
	if err != nil {
		morir(err)
	}
	var e estado
	if err := json.Unmarshal(b, &e); err != nil {
		morir(err)
	}
	return e
}

func guardarReto(r string) {
	if err := os.WriteFile(filepath.Join(donde, "reto"), []byte(r), 0o600); err != nil {
		morir(err)
	}
}

func leerReto() string {
	b, err := os.ReadFile(filepath.Join(donde, "reto"))
	if err != nil {
		morir(err)
	}
	return string(b)
}

func morir(err error) {
	fmt.Fprintln(os.Stderr, "Mal:", err)
	os.Exit(1)
}
