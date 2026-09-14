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

> Esfinge es un gestor de contraseñas que vive en tu ordenador: guarda tus contraseñas en una bóveda
> cifrada, sin cuentas y sin servidores. Esta extensión lo conecta con tu navegador.
>
> Qué hace:
> • Rellena el usuario y la contraseña al entrar en un sitio del que tienes una cuenta guardada.
> • Rellena también el código de un solo uso, si la cuenta lo tiene.
> • Cuando entras, te registras o cambias la contraseña, te ofrece guardarla o actualizarla.
> • Con varias cuentas del mismo sitio, eliges cuál en su panel.
>
> Lo que tienes que saber:
> • Necesita la aplicación Esfinge instalada en tu ordenador, para macOS, Windows o Linux, con el canal
>   con el navegador encendido en sus Ajustes. Se descarga gratis en https://webcafeina.github.io/esfinge/
> • La extensión habla solo con Esfinge, en tu ordenador. No se conecta a ningún servidor, no manda nada
>   a internet ni a Webcafeína y no lleva analítica.
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

> Rellenar y guardar en el navegador las contraseñas de la bóveda de la aplicación Esfinge, instalada en
> el mismo ordenador.

**Justificación de cada permiso:**

| Permiso | Justificación |
|---|---|
| `nativeMessaging` | Es la única forma de hablar con la aplicación Esfinge instalada en el ordenador, que es donde están las contraseñas. La extensión no tiene servidor: sin este permiso no puede hacer nada. |
| `storage` | Guarda dos cosas: el permiso que la aplicación Esfinge da a este navegador para hablar con ella y que la persona ha aceptado el aviso de datos. Ninguna contraseña. |
| `alarms` | Comprueba una vez por minuto si la bóveda sigue abierta, para que el icono de la barra enseñe el candado cuando se cierra sola. La aplicación no puede avisar a la extensión. |
| `favicon` | Enseña en el panel el icono del sitio de la pestaña, sacado de la caché del navegador, sin descargarlo de internet. |
| Acceso a `https://*/*` | Para rellenar hay que encontrar el formulario de entrar en cualquier sitio donde la persona tenga una cuenta guardada, y para ofrecer guardar hay que leer lo que envía. Solo en `https`: nunca en `http` ni en marcos de otro origen. |

**Código remoto:** No, no uso código remoto.

**Uso de datos** —marcar—:

- **Información de autenticación**: usuario, contraseña y código de un solo uso de los formularios de
  entrar, para rellenarlos y para guardarlos en la bóveda.
- **Información personal identificable**: el usuario o el correo con el que se entra.
- **Actividad de navegación web**: la dirección de la pestaña, para saber de qué sitio son las cuentas.

Y no marcar el resto: ni datos de salud, ni financieros, ni comunicaciones, ni ubicación, ni contenido
del sitio más allá del formulario de entrar.

**Certificaciones** —marcar las tres—: no se venden ni se transfieren datos a terceros fuera de los usos
aprobados; no se usan ni transfieren para fines ajenos al propósito único; no se usan para calcular
solvencia ni para préstamos.

**Nota que conviene tener clara al rellenar**: todos esos datos van **solo a la aplicación Esfinge del
mismo ordenador**, por native messaging. Chrome pide declararlos igual, aunque no salgan del equipo.

### Datos de comerciante

Nombre legal, dirección y teléfono de Webcafeína, **públicos al pie de la ficha**. Los rellena el cliente
al dar de alta la cuenta (`pasos.md`).

---

## addons.mozilla.org

**Categoría:** Privacidad y seguridad (`privacy-security`). **Licencia:** todos los derechos reservados
(`all-rights-reserved`), que es la del repositorio.

**Declaración de datos** —ya va en el manifiesto, `data_collection_permissions`—: `authenticationInfo`,
`personallyIdentifyingInfo` y `browsingActivity`. Firefox la enseña al instalar.

**Notas para quien revise** (*Notes to reviewer*):

> Esfinge is a local password manager. This extension only talks to the Esfinge desktop app installed on
> the same computer, through native messaging (host `com.webcafeina.esfinge`); it has no server and makes
> no network requests. The declared data (authentication info, identifying info, browsing activity) is
> sent only to that native app.
>
> To try it you need the desktop app (free, https://github.com/webcafeina/esfinge/releases/latest) with
> "canal con el navegador" enabled in its settings; without it the panel explains that Esfinge cannot be
> found. Nothing runs before the user accepts the data notice in the panel.
>
> The code is bundled with Vite, not obfuscated. Build instructions are in COMPILAR.md at the root of the
> source archive; the build is byte-for-byte reproducible with Node 22 and pnpm 11.20.0.

Lo de las notas va en inglés a propósito: es para quien revisa, no para quien instala.
