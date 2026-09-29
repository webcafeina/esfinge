package navegador

import (
	"net"
	"net/url"
	"strings"
)

// RPIDPermitido dice con qué `rpId` se puede firmar en ese origen, o vacío.
//
// **Es la pieza de seguridad de las llaves de acceso** (ADR 0048): la única vía
// por la que esta función de todo el asunto puede *entregar* algo a un atacante.
// Firmar con el `rpId` equivocado no se ve; es una identificación válida en otro
// sitio.
//
// **Y `Encaja` no sirve para esto**, por mucho que se le parezca. Aquélla compara
// dominio registrable contra dominio registrable, y por eso `accounts.google.com`
// y `mail.google.com` son «el mismo sitio»: es lo que se quiere para ofrecer una
// contraseña. WebAuthn pide otra cosa y más estrecha —el `rpId` tiene que ser el
// anfitrión o un sufijo suyo separado por punto— y además **el hash se calcula
// sobre la cadena exacta**, así que dar por buenos dos nombres distintos no es ser
// tolerante: es firmar algo que el sitio rechazará, o peor, firmar para quien no
// es.
//
// Devuelve **la cadena exacta que hay que hashear**, no un dominio registrable.
func RPIDPermitido(rpID, origen string) string {
	u, err := url.Parse(strings.TrimSpace(origen))
	if err != nil || !strings.EqualFold(u.Scheme, "https") {
		return ""
	}
	anfitrion := strings.ToLower(strings.Trim(u.Hostname(), "[]"))
	if anfitrion == "" || net.ParseIP(anfitrion) != nil {
		return ""
	}
	for _, r := range anfitrion {
		if r > 127 {
			return ""
		}
	}
	if _, err := dominioRegistrable(anfitrion); err != nil {
		return ""
	}

	pedido := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(rpID), "."))
	// Sin `rpId`, manda **el anfitrión entero** y no su dominio registrable: en
	// `login.ejemplo.com` el `rpId` por defecto es `login.ejemplo.com`.
	if pedido == "" {
		return anfitrion
	}
	if pedido == anfitrion {
		return pedido
	}
	if !strings.HasSuffix(anfitrion, "."+pedido) {
		return ""
	}
	// **Que tenga algo por debajo que registrar.** Es lo que separa «el dominio de
	// arriba» de «un sufijo público»: `ejemplo.com` sí, `com` y `github.io` no; si
	// no, cualquier página alojada en `github.io` firmaría por todas las demás.
	if _, err := dominioRegistrable(pedido); err != nil {
		return ""
	}
	return pedido
}
