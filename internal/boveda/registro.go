package boveda

// Lo que se le ha dado a un agente de IA, apuntado dentro de la bóveda (ADR 0054).
//
// # Por qué existe, y por qué aquí
//
// Hasta la 0054, Esfinge **no registraba qué entradas se abrían**: ni cuándo, ni desde
// dónde. Era deliberado —un gestor de contraseñas con un registro de lo que se consulta
// es otro sitio donde está escrito lo que te importa— y con una persona delante se
// sostiene: lo que has mirado lo has mirado tú.
//
// Con un programa pidiendo cosas, la pregunta «¿qué le di a ese agente la semana
// pasada?» se hace sola, y hoy **nadie puede contestarla**. De ahí esto.
//
// **Y va dentro del cuerpo cifrado**, nunca en el historial de ficheros. Aquello es
// JSON en claro en la carpeta de configuración, y la regla que lo gobierna es absoluta:
// la bóveda no escribe ahí. Una lista de «el agente pidió la contraseña de Hacienda» en
// claro sería la misma señal de tráfico que `credenciales-dashlane.csv`.
//
// # Qué se apunta y qué no
//
// **Solo lo que entrega un secreto o borra**, que es lo que el cliente eligió: buscar e
// inventariar no dejan rastro, y es lo que más va a hacer un agente. Lo que se apunta es
// lo que no se puede deshacer o no se puede ver venir.
//
// Y se apunta **también lo que se le negó**, que es la mitad interesante: «algo pidió la
// contraseña del banco nueve veces y se dijo que no» es la señal por la que esto existe.

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"time"
)

// PlazoDelRegistro es cuánto se guarda un apunte. Se purga **al abrir la bóveda**, como
// la papelera y las lápidas y por la misma razón: una bóveda cerrada no ejecuta nada, y
// un reloj solo contaría mientras la aplicación estuviera puesta.
const PlazoDelRegistro = 90 * 24 * time.Hour

// TopeDelRegistro son los apuntes que caben. Al llegar se va el más viejo.
//
// **Hacen falta los dos, el plazo y el tope**, y no es cinturón y tirantes: el plazo
// solo acota un uso normal, y un agente en un bucle puede escribir miles en una tarde
// —que es ensanchar el cuerpo cifrado, y el cuerpo se sube entero en cada
// sincronización—.
const TopeDelRegistro = 2000

// Apunte es una cosa que se le dio a un agente, o que se le negó.
//
// **Nunca lleva un secreto**: lleva de qué entrada se trata y cómo se llamaba. El título
// se copia a propósito, para poder leer el registro sin ir a buscar la entrada **y
// aunque esa entrada ya no exista**.
type Apunte struct {
	// ID es del apunte, no de la entrada: es lo que deja fundir dos registros sin
	// que un equipo pise lo del otro.
	ID     string `json:"id"`
	Cuando string `json:"cuando"`
	// Quien es como se llamó el agente a sí mismo. **No se cree**, y por eso está
	// aquí junto a lo demás y no como una identidad: sirve para leer el registro.
	Quien string `json:"quien"`
	// Que es el verbo que pidió.
	Que string `json:"que"`
	// Sobre es la entrada, y Titulo cómo se llamaba cuando pasó.
	Sobre  string `json:"sobre,omitempty"`
	Titulo string `json:"titulo,omitempty"`
	// Resultado: `hecho`, `negado`, `sin-respuesta`.
	Resultado string `json:"resultado"`
	// Como: `preguntado` o `valvula`, para poder distinguir lo que se aprobó de
	// una en una de lo que pasó por la válvula de cinco minutos.
	Como string `json:"como,omitempty"`
}

// Los valores de `Resultado`, que son etiquetas y no frases.
const (
	ApunteHecho        = "hecho"
	ApunteNegado       = "negado"
	ApunteSinRespuesta = "sin-respuesta"
)

// Los valores de `Como`.
const (
	ApuntePreguntado = "preguntado"
	ApunteValvula    = "valvula"
	// ApunteDirecto: no se preguntó porque ese verbo no lo necesita —crear, o editar
	// algo que no es un secreto—. Se apunta igual, que es lo que deja leer después qué
	// hizo un agente en la bóveda sin que nadie se enterara en el momento.
	ApunteDirecto = "directo"
)

// Apuntar deja constancia. **No falla si no se puede**: lo que se estuviera haciendo ya
// se ha hecho, y un error aquí no puede deshacerlo ni esconderlo.
//
// Lo que sí hace es **no apuntar nada en una bóveda que no se puede escribir** —una de
// solo lectura, o una compartida de solo ver—, y quien llama tiene que decirlo en la
// pantalla: una medida de seguridad que no se aplica en silencio es peor que no tenerla.
func (b *Boveda) Apuntar(a Apunte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	if a.ID == "" {
		a.ID = identificadorDeApunte()
	}
	if a.Cuando == "" {
		a.Cuando = ahora().UTC().Format(time.RFC3339)
	}
	b.cont.Registro = append(b.cont.Registro, a)
	if n := len(b.cont.Registro) - TopeDelRegistro; n > 0 {
		b.cont.Registro = b.cont.Registro[n:]
	}
	b.cuerpoSucio = true
	return b.guardar()
}

// Registro es lo que se ha apuntado, lo último arriba.
func (b *Boveda) Registro() []Apunte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := append([]Apunte(nil), b.cont.Registro...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Cuando > out[j].Cuando })
	return out
}

// purgarRegistro quita los apuntes anteriores a esa fecha y dice cuántos.
func (b *Boveda) purgarRegistro(limite time.Time) int {
	corte := limite.UTC().Format(time.RFC3339)
	var quedan []Apunte
	for _, a := range b.cont.Registro {
		if a.Cuando >= corte {
			quedan = append(quedan, a)
		}
	}
	n := len(b.cont.Registro) - len(quedan)
	b.cont.Registro = quedan
	return n
}

func identificadorDeApunte() string {
	crudo := make([]byte, 8)
	if _, err := rand.Read(crudo); err != nil {
		// Sin azar no se puede dar un identificador que no choque, y chocar aquí
		// significa que una fusión se coma un apunte. El tiempo no sirve: dos
		// apuntes del mismo segundo lo comparten.
		return ""
	}
	return hex.EncodeToString(crudo)
}

// fundirRegistro une dos registros. **Es la sección más fácil de fundir de la bóveda**,
// porque un apunte es un hecho que pasó: no se edita, no se borra y no hay conflicto
// posible. Lo único que hay que hacer es no perder ninguno y no duplicarlos.
//
// **Y no se purga aquí**, que es lo que la haría divergir: el purgado va al abrir, como
// el de la papelera y el de las lápidas, así que los dos lados tiran lo mismo cuando les
// toca. Purgando en la fusión, el lado que ha abierto hace poco le quitaría al otro
// apuntes que el otro todavía tiene que ver.
func fundirRegistro(l, r []Apunte) []Apunte {
	vistos := map[string]bool{}
	out := make([]Apunte, 0, len(l)+len(r))
	for _, lista := range [][]Apunte{l, r} {
		for _, a := range lista {
			if a.ID == "" || vistos[a.ID] {
				continue
			}
			vistos[a.ID] = true
			out = append(out, a)
		}
	}
	// Orden estable y el mismo en los dos lados: por fecha y, a igualdad, por
	// identificador. Sin lo segundo, dos apuntes del mismo segundo saldrían en
	// cualquier orden y los dos lados darían bytes distintos.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cuando != out[j].Cuando {
			return out[i].Cuando < out[j].Cuando
		}
		return out[i].ID < out[j].ID
	})
	// Y el tope también aquí: dos equipos que hayan apuntado mucho suman.
	if n := len(out) - TopeDelRegistro; n > 0 {
		out = out[n:]
	}
	return out
}
