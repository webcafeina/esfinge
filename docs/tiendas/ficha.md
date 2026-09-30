# Las fichas de la extensión en las tiendas

Lo que se escribe en la Chrome Web Store y en addons.mozilla.org (ADR 0033). **Tiene que decir lo mismo
que el aviso del panel (`navegador/src/panel.html`) y que la política de privacidad
(`web/privacidad.html`)**: si cambia uno, cambian los tres, y sube `VERSION_DEL_AVISO`.

Textos en español, que es el idioma de la extensión.

## La ficha de Chrome se pega a mano, y por eso hay marcas

**Firefox recibe la suya con cada etiqueta** —`amo-metadata.json` va en el flujo de publicación—, pero
**la de Chrome se escribe en su consola y su API no la edita**: sube el paquete y nada más. Así que todo
lo que cambie aquí de la parte de Chrome **hay que ir a pegarlo**, y nada lo recuerda.

Ya costó una: lo que la entrega de compartir cambió el **2026-09-24** nunca se pegó, y la ficha se quedó
seis días diciendo otra cosa que el aviso del panel. Lo vio el cliente el 2026-09-30, buscando en su
consola una línea que yo daba por puesta.

Por eso los bloques que se pegan van entre `<!-- consola de Chrome: empieza -->` y
`<!-- … acaba -->`, y **`make comprobar` falla si cambian sin que nadie diga que los ha pegado**:

```sh
node navegador/herramientas/ficha-de-chrome.mjs            # dice si hay algo sin pegar
node navegador/herramientas/ficha-de-chrome.mjs --pegado   # después de pegarlo en la consola
```

Lo que queda **fuera** de las marcas es para nosotros: el porqué de cada cosa, y las casillas de uso de
datos, que **no tienen ningún campo donde escribir**.

Y de ahí dos reglas, las dos aprendidas el mismo día y ninguna obvia desde aquí.

**Dentro de las marcas no va markdown.** La consola no lo interpreta, así que un `**` o una comilla
invertida quedarían escritos en la ficha; y si aquí hubiera markdown y allí prosa, la herramienta
vigilaría un texto que no es el de la tienda. Eso ya pasó: las justificaciones de los permisos eran una
tabla con negritas, y lo que se le pasó al cliente para pegar era otra cosa.

**Y solo se marca lo que tiene un campo donde pegarlo.** Esta ficha describe una consola que desde aquí
**no se puede ver**, así que qué campos existen es cosa de quien los mira. En un solo día aparecieron
tres sitios donde se había escrito texto para un campo inexistente: las casillas de uso de datos —solo
casillas—, «Código remoto» —marcando «no», la explicación desaparece— y, antes, una viñeta que yo daba
por puesta y llevaba seis días sin pegar. Lo que va entre marcas es **lo que se ha confirmado que tiene
hueco**; lo demás se queda fuera, como nota, y se dice que no se pega.

---

## Lo común a las dos

**Nombre:** Esfinge

**Resumen** —en Chrome es el `description` del manifiesto, 132 caracteres como máximo; en Firefox, 250—:

Rellena las contraseñas de tu bóveda de Esfinge sin salir del navegador.

**Descripción:**

<!-- consola de Chrome: empieza -->
Esfinge es un gestor de contraseñas que guarda tus contraseñas en una bóveda cifrada: en tu ordenador,
o en todos tus equipos con una cuenta que no puede leer nadie más que tú. Esta extensión lo trae a tu
navegador.

Qué hace:
• Rellena el usuario y la contraseña al entrar en un sitio del que tienes una cuenta guardada.
• Rellena también el código de un solo uso, si la cuenta lo tiene.
• Cuando entras, te registras o cambias la contraseña, te ofrece guardarla o actualizarla.
• Con varias cuentas del mismo sitio, eliges cuál en su panel.
• Con cuenta, manda una copia de una contraseña a otra persona, cifrada para ella.
• Entra con tus llaves de acceso: cuando un sitio pide una, te ofrece la tuya y basta con aceptar.

Lo que tienes que saber:
• Sin cuenta, necesita la aplicación Esfinge instalada en tu ordenador —para macOS, Windows o Linux, con
  el canal con el navegador encendido en sus Ajustes— y habla solo con ella: no se conecta a internet.
  Se descarga gratis en https://webcafeina.github.io/esfinge/
• Con tu cuenta de Esfinge funciona sola, sin la aplicación: guarda tu bóveda cifrada en el navegador y
  la sincroniza con el servidor de cuentas de Webcafeína, en la UE, que no puede leerla.
• Al mandar una copia, el servidor ve la dirección de quien la recibe, no lo que le mandas. Si esa
  persona no tiene cuenta, Webcafeína le manda una invitación con tu dirección dentro.
• Para las llaves de acceso pone una pieza dentro de cada página https, que es la única forma de
  enterarse de cuándo un sitio pide una. La clave privada no sale nunca de tu bóveda: al sitio le
  llega solo la firma. Se puede apagar en los Ajustes de la aplicación.
• No lleva analítica ni servicios de terceros.
• La primera vez que abras su panel te explica qué datos toca. Hasta que lo aceptas, no lee ninguna
  página.

Política de privacidad: https://webcafeina.github.io/esfinge/privacidad.html
Soporte: https://webcafeina.github.io/esfinge/soporte.html

<!-- consola de Chrome: acaba -->

**Web:** https://webcafeina.github.io/esfinge/
**Soporte:** https://webcafeina.github.io/esfinge/soporte.html
**Correo de soporte:** info@webcafeina.com
**Política de privacidad:** https://webcafeina.github.io/esfinge/privacidad.html

---

## Chrome Web Store

**Categoría:** Herramientas (*Tools*). **Idioma:** español.

### Pestaña «Privacy practices»

**Propósito único** (*Single purpose*):

<!-- consola de Chrome: empieza -->
Rellenar y guardar en el navegador las contraseñas de la bóveda de Esfinge: la de la aplicación
instalada en el mismo ordenador o, con cuenta, la que la extensión sincroniza con el servidor de cuentas.
Con cuenta, desde el panel se puede además mandar una copia de una contraseña de la bóveda a otra
persona, cifrada para ella. Y cuando un sitio pide una llave de acceso, ofrecer la que está guardada en
esa misma bóveda y firmar con ella.

<!-- consola de Chrome: acaba -->

**Justificación de cada permiso**, uno por campo de la consola. **Sin markdown, sin negritas y sin
comillas invertidas**: la consola no las interpreta y quedarían escritas tal cual. Lo de dentro de las
marcas se pega **literalmente**, y eso es lo que hace que la huella de `ficha-de-chrome.mjs` signifique
algo: si aquí hubiera markdown y allí prosa, la herramienta vigilaría un texto que no es el de la tienda.

`nativeMessaging`:

<!-- consola de Chrome: empieza -->
Sin cuenta, es la única forma de hablar con la aplicación Esfinge instalada en el ordenador, que es
donde están las contraseñas.
<!-- consola de Chrome: acaba -->

`storage`:

<!-- consola de Chrome: empieza -->
Guarda que la persona ha aceptado el aviso de datos y el permiso de la aplicación para hablar con ella.
Con cuenta, además, la bóveda cifrada, su última versión común con el servidor —cifrada— y la sesión,
cifrada con la clave de la bóveda. La clave de la bóveda abierta va solo en storage.session, en memoria,
y se va al cerrar el navegador.
<!-- consola de Chrome: acaba -->

`alarms`:

<!-- consola de Chrome: empieza -->
Una vez por minuto: comprobar si la bóveda sigue abierta, para que el icono lo diga; con cuenta,
sincronizarla con el servidor y cerrarla a los quince minutos sin usarla.
<!-- consola de Chrome: acaba -->

`favicon`:

<!-- consola de Chrome: empieza -->
Enseña en el panel el icono del sitio de la pestaña, sacado de la caché del navegador, sin descargarlo
de internet.
<!-- consola de Chrome: acaba -->

Acceso a los sitios `https` (*host permissions*) — **el que más se mira**, y el que Chrome ya señaló en
la 2.22.1:

<!-- consola de Chrome: empieza -->
Para rellenar hay que encontrar el formulario de entrar en cualquier sitio donde la persona tenga
una cuenta guardada, y para ofrecer guardar hay que leer lo que envía. Y para las llaves de acceso,
un guion en el mundo principal (world: "MAIN"), que es la única forma de enterarse de que un sitio
ha pedido una: sustituye navigator.credentials.get y .create, y cede al método original siempre que
no tenga nada que ofrecer — cuando no hay ninguna llave guardada para ese sitio, en un marco de otro
origen, con mediation: "conditional", ante cualquier forma de petición que no reconozca, o si algo
falla. La clave privada de una llave nunca cruza a la página: la firma se hace donde está la bóveda
y solo la firma ya hecha llega al sitio. La persona puede apagarlo en los Ajustes de la aplicación.
Solo en https: nunca en http. Con cuenta, cubre también el servidor de cuentas,
https://esfinge-cuentas.webcafeina.com.
<!-- consola de Chrome: acaba -->

**Código remoto:** se contesta **no**, y **no hay nada que escribir**: marcando «no uso código remoto»
la consola no deja explicación. Comprobado por el cliente el 2026-09-30. Así que esto es nota nuestra,
para saber por qué se contesta eso: el WebAssembly —Argon2id, de `hash-wasm`— va dentro del paquete, y
por eso la política de contenido lleva `'wasm-unsafe-eval'`. Si algún día hubiera que bajar algo, esta
respuesta cambia y con ella la revisión entera.

**Uso de datos** —**solo hay casillas**, sin ningún campo donde escribir. Comprobado en la consola por
el cliente el 2026-09-30, y conviene saberlo: lo de debajo **no se pega en ninguna parte**. Es la nota de
por qué se marca cada una, para que la próxima vez que cambie lo que la extensión toca se pueda decidir
si hay que marcar una casilla más—:

- **Información de autenticación** ✓ — usuario, contraseña y código de un solo uso de los formularios de
  entrar, para rellenarlos y para guardarlos en la bóveda.
- **Información personal identificable** ✓ — el usuario o el correo con el que se entra y, al mandar una
  copia, la dirección de correo de quien la recibe.
- **Actividad de navegación web** ✓ — la dirección de la pestaña, para saber de qué sitio son las cuentas,
  y el sitio para el que un sitio pide una llave de acceso.

Y no marcar el resto: ni datos de salud, ni financieros, ni comunicaciones, ni ubicación, ni contenido
del sitio más allá del formulario de entrar.

**Las llaves de acceso no añaden ninguna casilla**, y hace falta decirlo para no volver a mirarlo: el
`rpId` que un sitio pide es actividad de navegación, que ya estaba marcada, y **la firma no es un dato de
la persona que se recoja**: se calcula, sale hacia el sitio y no se guarda en ninguna parte.

**Certificaciones** —marcar las tres—: no se venden ni se transfieren datos a terceros fuera de los usos
aprobados; no se usan ni transfieren para fines ajenos al propósito único; no se usan para calcular
solvencia ni para préstamos.

**Nota que conviene tener clara al rellenar**: sin cuenta, esos datos van **solo a la aplicación Esfinge
del mismo ordenador**, por native messaging. Con cuenta, las contraseñas y los códigos se guardan en la
bóveda, que **sale cifrada** hacia el servidor de cuentas y no la puede leer nadie más que la persona; lo
que el servidor ve en claro es el correo y el nombre del navegador. Desde la E2 (ADR 0040) hay que
declararlo así, y **cambiar esto es cambiar el aviso del panel y la política**, que tienen que decir lo
mismo.

**Y lo que hay que decir de las llaves de acceso, sin adornarlo** (ADR 0048): esta versión pone un guion
en el **mundo principal** de cada página `https`. Es la parte de la ficha que más se va a mirar, porque es
lo que Chrome ya señaló en la 2.22.1 por «permisos de host amplios». Lo que lo justifica es que no hay
otra forma: `navigator.credentials` solo existe ahí. Y lo que lo acota, dicho en este orden, es que
**cede al método original** ante cualquier duda —sin llave para ese sitio, en un marco ajeno, con una
forma de petición que no reconoce, con `mediation: "conditional"`, o si algo falla—, que **la clave
privada no cruza el puente** —lo que sube es la firma ya hecha— y que **se puede apagar** en los Ajustes
de la aplicación.

### Datos de comerciante

Nombre legal, dirección y teléfono de Webcafeína, **públicos al pie de la ficha**. Los rellena el cliente
al dar de alta la cuenta (`pasos.md`).

---

## addons.mozilla.org

**Categoría:** Privacidad y seguridad (`privacy-security`). **Licencia:** todos los derechos reservados
(`all-rights-reserved`), que es la del repositorio.

**Declaración de datos** —ya va en el manifiesto, `data_collection_permissions`—: `authenticationInfo`,
`personallyIdentifyingInfo` y `browsingActivity`. Firefox la enseña al instalar. **Compartir no añade
ninguna**: la dirección de quien recibe la copia entra en `personallyIdentifyingInfo`, que ya estaba.

**Notas para quien revise** (*Notes to reviewer*):

> Esfinge is a password manager. The extension works in two modes:
>
> - Without an account, it only talks to the Esfinge desktop app installed on the same computer, through
>   native messaging (host `com.webcafeina.esfinge`), and makes no network requests.
> - With an Esfinge account, it keeps the user's vault **encrypted** in extension
>   storage and syncs it with our account server, https://esfinge-cuentas.webcafeina.com (Cloudflare, EU).
>   The vault is end-to-end encrypted with a key derived from the master password (Argon2id via the bundled
>   hash-wasm WebAssembly, hence `'wasm-unsafe-eval'`, and XChaCha20-Poly1305 via @noble/ciphers); the
>   server only sees the e-mail address, the encrypted vault and the browser's device name.
>
> The panel can also send a copy of one vault entry to another person: the entry is sealed with
> HPKE towards that account's published public key and signed with Ed25519, so the server only relays a
> sealed envelope and sees the recipient's e-mail address. If that address has no account, the server sends
> it a one-off invitation e-mail naming the sender; the envelope itself never leaves the sender's vault
> until the recipient has keys of their own, so no secret travels by e-mail.
>
> To try it you need the desktop app (free, https://github.com/webcafeina/esfinge/releases/latest) with
> "canal con el navegador" enabled in its settings; without it the panel explains that Esfinge cannot be
> found and offers to sign in with an account. Nothing runs before the user accepts the data notice in the
> panel.
>
> The code is bundled with Vite, not obfuscated. Build instructions are in COMPILAR.md at the root of the
> source archive; the build is byte-for-byte reproducible with Node 22 and pnpm 11.20.0.

Lo de las notas va en inglés a propósito: es para quien revisa, no para quien instala.
