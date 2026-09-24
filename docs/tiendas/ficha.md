# Las fichas de la extensión en las tiendas

Lo que se escribe en la Chrome Web Store y en addons.mozilla.org (ADR 0033). **Tiene que decir lo mismo
que el aviso del panel (`navegador/src/panel.html`) y que la política de privacidad
(`web/privacidad.html`)**: si cambia uno, cambian los tres, y sube `VERSION_DEL_AVISO`.

Textos en español, que es el idioma de la extensión.

---

## Lo común a las dos

**Nombre:** Esfinge

**Resumen** —en Chrome es el `description` del manifiesto, 132 caracteres como máximo; en Firefox, 250—:

> Rellena las contraseñas de tu bóveda de Esfinge sin salir del navegador.

**Descripción:**

> Esfinge es un gestor de contraseñas que guarda tus contraseñas en una bóveda cifrada: en tu ordenador,
> o en todos tus equipos con una cuenta que no puede leer nadie más que tú. Esta extensión lo trae a tu
> navegador.
>
> Qué hace:
> • Rellena el usuario y la contraseña al entrar en un sitio del que tienes una cuenta guardada.
> • Rellena también el código de un solo uso, si la cuenta lo tiene.
> • Cuando entras, te registras o cambias la contraseña, te ofrece guardarla o actualizarla.
> • Con varias cuentas del mismo sitio, eliges cuál en su panel.
> • Con cuenta, manda una copia de una contraseña a otra persona, cifrada para ella.
>
> Lo que tienes que saber:
> • Sin cuenta, necesita la aplicación Esfinge instalada en tu ordenador —para macOS, Windows o Linux, con
>   el canal con el navegador encendido en sus Ajustes— y habla solo con ella: no se conecta a internet.
>   Se descarga gratis en https://webcafeina.github.io/esfinge/
> • Con tu cuenta de Esfinge funciona sola, sin la aplicación: guarda tu bóveda cifrada en el navegador y
>   la sincroniza con el servidor de cuentas de Webcafeína, en la UE, que no puede leerla.
> • Al mandar una copia, el servidor ve la dirección de quien la recibe, no lo que le mandas. Si esa
>   persona no tiene cuenta, Webcafeína le manda una invitación con tu dirección dentro.
> • No lleva analítica ni servicios de terceros.
> • La primera vez que abras su panel te explica qué datos toca. Hasta que lo aceptas, no lee ninguna
>   página.
>
> Política de privacidad: https://webcafeina.github.io/esfinge/privacidad.html
> Soporte: https://webcafeina.github.io/esfinge/soporte.html

**Web:** https://webcafeina.github.io/esfinge/
**Soporte:** https://webcafeina.github.io/esfinge/soporte.html
**Correo de soporte:** info@webcafeina.com
**Política de privacidad:** https://webcafeina.github.io/esfinge/privacidad.html

---

## Chrome Web Store

**Categoría:** Herramientas (*Tools*). **Idioma:** español.

### Pestaña «Privacy practices»

**Propósito único** (*Single purpose*):

> Rellenar y guardar en el navegador las contraseñas de la bóveda de Esfinge: la de la aplicación
> instalada en el mismo ordenador o, con cuenta, la que la extensión sincroniza con el servidor de cuentas.
> Con cuenta, desde el panel se puede además mandar una copia de una contraseña de la bóveda a otra
> persona, cifrada para ella.

**Justificación de cada permiso:**

| Permiso | Justificación |
|---|---|
| `nativeMessaging` | Sin cuenta, es la única forma de hablar con la aplicación Esfinge instalada en el ordenador, que es donde están las contraseñas. |
| `storage` | Guarda que la persona ha aceptado el aviso de datos y el permiso de la aplicación para hablar con ella. Con cuenta, además, la bóveda **cifrada**, su última versión común con el servidor —cifrada— y la sesión, cifrada con la clave de la bóveda. La clave de la bóveda abierta va solo en `storage.session`, en memoria. |
| `alarms` | Una vez por minuto: comprobar si la bóveda sigue abierta, para que el icono lo diga; con cuenta, sincronizarla con el servidor y cerrarla a los quince minutos sin usarla. |
| `favicon` | Enseña en el panel el icono del sitio de la pestaña, sacado de la caché del navegador, sin descargarlo de internet. |
| Acceso a `https://*/*` | Para rellenar hay que encontrar el formulario de entrar en cualquier sitio donde la persona tenga una cuenta guardada, y para ofrecer guardar hay que leer lo que envía. Solo en `https`: nunca en `http` ni en marcos de otro origen. Con cuenta, cubre también el servidor de cuentas, `https://esfinge-cuentas.webcafeina.com`. |

**Código remoto:** No, no uso código remoto. El WebAssembly —Argon2id, de `hash-wasm`— va dentro del
paquete; por eso la política de contenido lleva `'wasm-unsafe-eval'`.

**Uso de datos** —marcar—:

- **Información de autenticación**: usuario, contraseña y código de un solo uso de los formularios de
  entrar, para rellenarlos y para guardarlos en la bóveda.
- **Información personal identificable**: el usuario o el correo con el que se entra y, al mandar una
  copia, la dirección de correo de quien la recibe.
- **Actividad de navegación web**: la dirección de la pestaña, para saber de qué sitio son las cuentas.

Y no marcar el resto: ni datos de salud, ni financieros, ni comunicaciones, ni ubicación, ni contenido
del sitio más allá del formulario de entrar.

**Certificaciones** —marcar las tres—: no se venden ni se transfieren datos a terceros fuera de los usos
aprobados; no se usan ni transfieren para fines ajenos al propósito único; no se usan para calcular
solvencia ni para préstamos.

**Nota que conviene tener clara al rellenar**: sin cuenta, esos datos van **solo a la aplicación Esfinge
del mismo ordenador**, por native messaging. Con cuenta, las contraseñas y los códigos se guardan en la
bóveda, que **sale cifrada** hacia el servidor de cuentas y no la puede leer nadie más que la persona; lo
que el servidor ve en claro es el correo y el nombre del navegador. Desde la E2 (ADR 0040) hay que
declararlo así, y **cambiar esto es cambiar el aviso del panel y la política**, que tienen que decir lo
mismo.

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
