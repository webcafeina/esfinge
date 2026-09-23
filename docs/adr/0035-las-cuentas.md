# ADR 0035 — Las cuentas: la bóveda en todos los equipos, con un servidor nuestro que no puede leerla

**Fecha:** 2026-09-18 · **Estado:** aceptada, sin empezar · **Matiza la [0023](0023-la-boveda.md)** («sin
servidor») **y la [0014](0014-comprobacion-de-actualizaciones.md)** (qué sale de la máquina) ·
**Revisar cuando** haya cuentas de fuera de la casa, o al pasar a la fase B

## Contexto

Hasta la 2.22.2 Esfinge es solo local: cada ordenador tiene su bóveda y ninguno habla con otro. El cliente
lo usa en varios equipos y quiere **la misma bóveda en todos**, y además **mandar credenciales a la cuenta de
otra persona** —sus clientes— y recibirlas. Es la fase 3 del plan de sustituir a Dashlane, la que la ADR 0023
dejó fuera a propósito.

Cambia lo que el producto es. La portada, la política de privacidad y `docs/seguridad.md` prometen «sin
cuentas y sin servidores» y «Webcafeína no recibe nada»; con cuenta, un servidor nuestro guarda la bóveda.
**Cifrada de extremo a extremo**, así que sigue sin poder leerla, pero ya sabe quién eres, desde dónde y
cuándo.

El plan completo, con la criptografía, la fusión, el servidor y las entregas, está en
[`docs/cuentas.md`](../cuentas.md). Esta ficha recoge **lo que decidió el cliente** y no se cambia sin
preguntar; las decisiones técnicas de cada pieza tendrán su propia ADR (0036 a 0041) al construirse.

## Decisión

Decidido con el cliente el 2026-09-18, por preguntas con opciones:

| Pregunta | Elegido |
|---|---|
| Para quién | Webcafeína y sus clientes |
| Registro | **Libre** cuando la política de privacidad y las condiciones de uso estén revisadas y publicadas; hasta entonces, por lista de admisión. **Abierto el 2026-09-23**, cumplida esa condición |
| Servidor | Propio, en **Cloudflare**, con los datos en la **UE** (D1, Durable Objects y R2 con jurisdicción `eu`) |
| Dirección | `esfinge-cuentas.webcafeina.com` |
| Entrar | **Correo + contraseña maestra**, y **código por correo** en cada equipo nuevo; Touch ID o PIN más adelante |
| Correo | **Resend** |
| Al empezar | **Bienvenida visual al arrancar** la primera vez: «En este ordenador» o «Con cuenta», con ventajas e inconvenientes. **Reversible en los dos sentidos desde Ajustes** |
| Equipos u organizaciones | **Ninguno.** Cada persona tiene su bóveda |
| Compartir | **Una copia, sin permisos**: al recibirla es de quien la recibe; si cambia, se vuelve a mandar |
| Recuperar | **Solo con la clave de recuperación personal.** Webcafeína no puede recuperar nada |
| Contraseña con cuenta | **Exigir «fuerte»** con el medidor que ya existe; en local se avisa como ahora |
| Dos bóvedas en un equipo | **Preguntar cada vez**: «Juntar» o «Quedarme con la de la cuenta», apartando la local en una copia |
| La extensión con cuenta | **Un cliente más**: entra, se desbloquea con la maestra y **lo hace todo sin la aplicación**. **Con cuenta va siempre por la cuenta**, aunque la aplicación esté abierta. Sin cuenta, sigue por el canal nativo |
| Orden | **Por fases**: la bóveda propia sincronizada; la extensión autónoma; abrir el registro; compartir; Touch ID o PIN |

Y tres cosas de fondo que no se discutieron porque sin ellas no hay producto:

- **Conocimiento cero.** El servidor no ve nunca la contraseña maestra, la clave de la bóveda ni una
  entrada. **La contraseña de la cuenta es la maestra de la bóveda**, y la ranura `"servidor"` que la
  ADR 0023 dejó prevista **se descarta**: una ranura envuelta con una clave que guarde el servidor
  rompería justo esto.
- **La contraseña que se usa para entrar no es la clave que se manda.** De la maestra se deriva, aparte y
  con su propia sal, una clave de acceso; el servidor guarda un HMAC de ella con una pimienta.
- **Lo que no se sincroniza nunca**: el historial de qué y cuándo —regla absoluta desde la ADR 0010—, las
  preferencias, los navegadores permitidos y los iconos.

## Alternativas descartadas

- **Un servicio de sincronización ajeno** (iCloud, Dropbox, una carpeta compartida): sin servidor propio no
  hay compartir entre cuentas, y la fusión de dos equipos escribiendo el mismo fichero la haría otro sin
  saber qué es una entrada.
- **Equipos con bóvedas compartidas y permisos.** Se preguntó y el cliente lo descartó: no gestiona equipos,
  manda credenciales a personas. Una copia es mucho más sencilla de hacer bien y de explicar.
- **Permisos al compartir** («solo ver», «solo usar», como Dashlane). Se explicó que con una copia no
  encajan —la copia se queda vieja y quien la tiene no puede arreglarla— y que «solo usar» lo impone la
  aplicación, no el cifrado. El cliente eligió la copia sin permisos.
- **SMTP de Google** para los códigos. Es lo que usa webcafeina-local, pero desde un Worker hay que guardar
  la contraseña de un buzón y el tope es de 2000 al día. Resend va por API y el inventario de Cronos ya lo
  da por verificado en `webcafeina.com`.
- **Que la extensión con cuenta siga pasando por la aplicación cuando está abierta.** Era lo recomendado —un
  desbloqueo en vez de dos—; el cliente prefirió un solo camino.
- **Que la extensión solo rellene, o guarde «en espera» para que la funda la aplicación.** Era lo
  recomendado, porque así la fusión se escribe una vez. El cliente la quiere completa; el coste es tener la
  fusión en Go y en TypeScript, y el plan lo paga con casos compartidos y una prueba cruzada byte a byte.

## Consecuencias

- **Dos promesas públicas dejan de ser ciertas** en modo cuenta —«sin servidores» y «Webcafeína no recibe
  nada»— y hay que cambiarlas antes de que nadie tenga una: portada, privacidad, `docs/seguridad.md`, las
  fichas de las tiendas y el aviso de la extensión.
- **Robar el servidor permite atacar la maestra sin conexión**, igual que hoy robar el fichero. Por eso se
  exige «fuerte» con cuenta.
- **Un servidor malicioso puede enseñar a un equipo una versión vieja** y no la última. Se detecta que
  retroceda, no que se congele.
- **Un error de fusión se propaga a todos los equipos.** Es el riesgo número uno del plan y el que más
  pruebas se lleva.
- **La extensión abre la bóveda dentro del navegador**, que es lo más expuesto de todo el producto, y **se
  conecta a internet**, que hoy prometen las tiendas que no hace.
- **Registro abierto obliga a papeles**: condiciones de uso, encargos de tratamiento con Cloudflare y
  Resend, y una revisión de los textos ([ADR 0041](0041-los-papeles-de-la-cuenta.md)). Y a que los topes
  diarios del servidor quepan en lo que da el correo.
- **El código de un solo uso y la contraseña, juntos en el servidor**, cifrados. Es lo mismo que dijo la ADR
  0025 de la bóveda abierta, ahora en todos los equipos.

## Verificación

Nada todavía: es la decisión, no la construcción. **Comprobado en la documentación de Cloudflare** que D1 y
los Durable Objects admiten la jurisdicción `eu`, y que en D1 se fija al crear la base y no se puede cambiar
después. **Sin comprobar**: lo que dice la ley colombiana de guardar datos en la UE (Ley 1581 de 2012 y la
lista de países adecuados de la SIC), que hay que mirar en la fuente antes de abrir a clientes de allí.
