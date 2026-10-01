// Package wifi compone el texto que va dentro del código QR de una red.
//
// Es el formato de facto que leen iOS y Android —`WIFI:T:WPA;S:red;P:clave;;`—, y
// cabe en cuarenta líneas. Está aparte de `internal/qr` a propósito: uno sabe qué
// es una red y el otro sabe dibujar cuadrados, y mezclarlos haría que probar el
// dibujo pasara por entender el wifi.
//
// **Lo que de verdad decide si un móvil se conecta no es el QR, es esta cadena.** Un
// escape que falte, un tipo de seguridad equivocado o un SSID hexadecimal sin
// comillas dan un código perfectamente legible que lleva a otra red o a ninguna, y
// eso no se nota mirando el dibujo.
package wifi

import (
	"errors"
	"strings"
)

// Las clases de seguridad que Esfinge guarda, en minúscula y sin inventar nada: son
// lo que se escribe en el campo `seguridad` de la entrada.
const (
	WPA     = "wpa"
	WEP     = "wep"
	Abierta = "abierta"
)

// Normalizar dice qué seguridad tiene una red a partir de lo que diga el fichero del
// que salió y de si trae clave.
//
// **Lo que diga el fichero no manda, y esto no es una precaución teórica.** El
// `wifi.csv` de Dashlane del cliente trae `encription_type: unsecured` en sus dos
// redes **y las dos tienen contraseña**. Creyéndole, el QR saldría con `nopass` y el
// móvil intentaría entrar sin clave: no se conectaría, y el fallo parecería del
// código y no del dato. Así que la clave manda sobre la etiqueta, igual que en el
// importador el tipo de una entrada sale de los campos que vengan rellenos y no de lo
// que el fichero diga de sí mismo.
//
// Lo que sí se respeta es lo que **añade** información: si dice WEP, es WEP; si dice
// cualquier variante de WPA, es WPA.
func Normalizar(dicho string, hayClave bool) string {
	d := strings.ToLower(strings.TrimSpace(dicho))
	if !hayClave {
		// Sin clave no hay nada que proteger, diga lo que diga la etiqueta.
		return Abierta
	}
	if strings.Contains(d, "wep") {
		return WEP
	}
	// Con clave, cualquier otra cosa —incluido «unsecured» y el vacío— es WPA. No se
	// distingue WPA2 de WPA3 porque **el QR no los distingue**: el cifrado lo negocia
	// el router, y marcar `SAE` deja fuera a los móviles que no lo entienden.
	return WPA
}

// Enlace compone el texto del QR de una red.
//
// Devuelve error sin SSID: un QR de una red sin nombre no lleva a ninguna parte, y es
// mejor no dibujar nada que dibujar algo que falla en el móvil de un invitado.
func Enlace(ssid, clave, seguridad string, oculta bool) (string, error) {
	if strings.TrimSpace(ssid) == "" {
		return "", errors.New("Una red sin nombre no se puede poner en un código")
	}

	tipo := "nopass"
	switch Normalizar(seguridad, clave != "") {
	case WEP:
		tipo = "WEP"
	case WPA:
		tipo = "WPA"
	}

	var b strings.Builder
	b.WriteString("WIFI:T:")
	b.WriteString(tipo)
	b.WriteString(";S:")
	b.WriteString(valor(ssid))
	if clave != "" {
		b.WriteString(";P:")
		b.WriteString(valor(clave))
	}
	// **`H:true` solo cuando lo es.** Escribir `H:false` es correcto según el formato y
	// hay lectores que se atragantan con él, así que se omite: lo que no se dice es lo
	// de siempre.
	if oculta {
		b.WriteString(";H:true")
	}
	// El doble punto y coma del final no es un adorno: cierra el último campo y cierra
	// la cadena.
	b.WriteString(";;")
	return b.String(), nil
}

// valor escapa un campo, y lo entrecomilla si hace falta.
//
// **El caso que se olvida es el hexadecimal.** Un SSID como `CAFE` o `1234` son
// dígitos hexadecimales válidos, y el formato dice que entonces el valor se interpreta
// como bytes en hexadecimal: la red `CAFE` se buscaría como los dos bytes `0xCA 0xFE`
// y no se encontraría. Entre comillas se lee como texto. Entrecomillar de más no
// rompe nada; de menos, sí.
func valor(s string) string {
	if esHex(s) {
		return `"` + escapar(s) + `"`
	}
	return escapar(s)
}

// escapar pone una barra delante de los cinco caracteres que el formato reserva.
//
// La barra va primera en la lista a propósito: escapándola después se escaparían las
// barras que acaba de poner esta misma función.
func escapar(s string) string {
	for _, c := range []string{`\`, `;`, `,`, `:`, `"`} {
		s = strings.ReplaceAll(s, c, `\`+c)
	}
	return s
}

func esHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
