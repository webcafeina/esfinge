# ADR 0045 — El correo deja Resend y sale por el VPS

**Fecha:** 2026-09-28 · **Estado:** **hecha el 2026-09-29**: el cartero desplegado, producción mandando
por él y Resend retirado del DNS · **Continúa la [0041](0041-los-papeles-de-la-cuenta.md)** y **matiza una premisa suya que
resultó falsa** · **Se apoya en la [0036](0036-el-servidor-de-cuentas.md)** · **Revisar cuando**
Cloudflare deje de estar vetado por Google, o cuando el VPS se caiga y se vea qué cuesta

## Contexto

El servidor de cuentas manda diez clases de correo —códigos de alta, de entrada, de recuperación y de
borrado, constancias e invitaciones— por **Resend, en su plan gratuito: cien correos al día**. La
[ADR 0041](0041-los-papeles-de-la-cuenta.md) lo aceptó a sabiendas y escribió la consecuencia: *«el cupo
del correo es el límite de crecimiento del servicio»*. Por eso `TOPE_ALTAS_DIA` se bajó de 200 a 30.

Webcafeína ya manda correo por el **relé SMTP de Google Workspace** en otro proyecto del mismo VPS
(Cronos), y ese camino da **diez mil mensajes al día**. Usarlo aquí quita el techo, quita una clave y
quita un tercero: hoy `web/privacidad.html` nombra a Resend cinco veces como encargado de tratamiento
que conserva los correos treinta días en Estados Unidos.

**Y corrige una premisa falsa de la 0041**, que descartó mandar desde el propio servidor porque «un
Worker no puede». Puede: `connect()` de `cloudflare:sockets` abre TCP saliente y **solo el puerto 25
está bloqueado**. Esa frase cerró esta puerta en su día y hay que decir que no era cierta.

Lo caro que la 0041 daba por principal —«el remitente ya está verificado con SPF, DKIM y DMARC»— ya
estaba hecho.

## Decisión

**El Worker compone el correo y lo entrega `cartero/`, un servicio nuestro en el VPS**, porque el Worker
no puede entregarlo él mismo.

No por la plataforma. **Google rechaza la autenticación SMTP cuando la conexión sale de Cloudflare**:
contesta `535 5.7.8 … p=BadCredentials` a las mismas credenciales que acepta desde un Mac. La IPv4 del
VPS sí está autorizada en el relé —es la que usa Cronos— y ahí no hacen falta credenciales.

Lo demás se descartó **midiendo**, no razonando, y queda escrito para que nadie lo repita:

| Hipótesis | Cómo se descartó |
|---|---|
| El secreto guardado no es el que creemos | La misma huella SHA-256 en Cloudflare y en el Mac: `b20e9e6b` |
| Un espacio pegado al copiarlo | Dieciséis caracteres, sin espacios (y ahora se recortan) |
| Nuestro `AUTH PLAIN` está mal | Con `AUTH LOGIN` Google **acepta el usuario** y rechaza la clave igual |
| La entrada nueva del relé está mal | `smtp.gmail.com`, que no la usa, falla exactamente igual |
| Google bloquea por ubicación | Diría `p=WebLoginRequired`, y **no hubo alerta de seguridad** |

El reparto del trabajo:

- **Compone el Worker** (`servidor/src/mensaje.ts`). Ahí siguen la maqueta, las diez cartas, el asunto
  de RFC 2047 y las partes en base64, con sus pruebas. Es la mitad que se puede probar y se queda donde
  se probaba.
- **Entrega el cartero** (`cartero/`), con nodemailer y **la configuración ya probada de Cronos**: el
  `ipv4first` de todo el proceso —o Google contesta `550 5.7.1 Invalid credentials for relay` porque la
  IPv6 no está en su lista— y el EHLO con un dominio de verdad —o contesta `421 4.7.0`—. Las dos
  costaron encontrarse allí y no se vuelven a pagar aquí.
- **Lo que cruza es el mensaje entero**, no sus trozos.

**Y el cartero revisa lo que le llega aunque venga firmado.** Es un servicio que manda correo como
webcafeína a quien se le diga, y un correo creíble desde el dominio propio de un gestor de contraseñas
es exactamente la pieza que le falta a quien quiera pescar a sus usuarios. El secreto compartido lo
cierra; un secreto se filtra. Detrás hay dos reglas: **el remitente es la cabecera nuestra entera y
exacta** —comparar solo la dirección dejaría pasar `Banco Santander <esfinge@webcafeina.com>`— y **un
solo destinatario**, sin `Cc` ni `Bcc`.

## Alternativas descartadas

- **Que el Worker lo entregue.** Era el plan, está escrito y probado, y **Google no lo permite**. El
  diálogo SMTP se ha borrado en vez de dejarlo apagado: aquí el código que solo corre el día malo es
  peor que no tenerlo. Está en el commit que lo quitó.
- **La API de Gmail por HTTPS** (cuenta de servicio con delegación en todo el dominio). Evita el VPS y
  la IP deja de importar, pero da **dos mil correos al día** en vez de diez mil, exige un proyecto de
  Google Cloud y una delegación, y la clave privada que aparece es más poderosa que la contraseña de
  aplicación. Se ofreció al cliente con esos costes y eligió el VPS.
- **Otro proveedor SMTP en la UE** (Amazon SES y parecidos). No cambiaría **ni una línea** de código y
  quitaría el tope, pero deja un tercero en la política de privacidad, que era la mitad de lo que se
  quería quitar.
- **Seguir en Resend.** Deja el techo de cien al día como límite de crecimiento.
- **Meter el endpoint dentro de Cronos.** Ahorraría un contenedor, una red y un nombre, y ataría el
  correo de Esfinge a los despliegues de Cronos. No se hace.

## Consecuencias

- **El VPS entra en el camino crítico del alta.** Si se cae, nadie puede darse de alta ni entrar desde
  un equipo nuevo; lo que ya está sincronizando sigue. Se le preguntó al cliente antes de plantear
  nada y su respuesta fue que el VPS ya es crítico de todos modos.
- **Un secreto nuevo**, compartido entre los Workers y el VPS. Si se filtra, quien lo tenga manda
  correo **como Esfinge**, con SPF en regla, de uno en uno y con nuestro nombre — que es menos de lo
  que daba la contraseña de aplicación, que dejaba mandar como cualquier dirección del dominio.
- **Los topes suben**: `TOPE_ALTAS_DIA` de 30 a 200. Doscientas altas gastan cuatrocientos correos de
  los diez mil. `INVITACIONES_POR_DIA` **se queda en 5**: ese tope nunca fue del correo, es antiabuso,
  y está escrito en la política de privacidad.
- **Un salto más de latencia** en las cinco peticiones que esperan al correo. Medido desde el VPS:
  **483 ms** el código y **332 ms** la invitación, contra los cientos de milisegundos de un POST a
  Resend. La invitación sigue yendo en `ctx.waitUntil`, que es lo que la mantiene muda.
- **El DKIM no alineaba**, y era la consecuencia más cara de todas: Google firma con su clave genérica
  (`d=webcafeina-com.20251104.gappssmtp.com`) porque **el dominio no tiene DKIM de Workspace**: no
  existe `google._domainkey.webcafeina.com`. Hoy DMARC pasa igual **porque alinea el SPF**, y el DMARC
  del dominio está en `p=none`; pero **un correo reenviado se queda sin ninguna autenticación
  alineada**, y es peor que lo que da Resend, que sí firma con `d=webcafeina.com`. Está en
  [`../deuda.md`](../deuda.md) y hay que arreglarlo antes de retirar Resend.

  > **Saldado el 2026-09-29.** El cliente generó la clave de 2048 bits en la consola y se publicó
  > `google._domainkey.webcafeina.com`. Ahora firma `d=webcafeina.com; s=google`. **Y arregla de paso
  > el correo de Cronos y el de las personas**, porque la clave es del dominio y el selector es único.

## Verificación

**Comprobado el 2026-09-28, entregando desde el VPS contra Google:**

- Los dos correos llegan **a la bandeja de entrada**, no a spam. Confirmado por el cliente.
- **Los acentos salen bien**, en el asunto y dentro.
- **El botón de la invitación se ve como botón**, que es lo que ya costó una vez.
- **SPF, DKIM y DMARC en `PASS`** — con la salvedad del `d=` de arriba, que es el hallazgo.
- **Los mensajes son los del Worker de verdad**, sacados de `componer` y no escritos a mano.
- **El cierre del cartero, montado y no solo en pruebas**: sin secreto, con uno equivocado y con uno
  casi entero dan `401`; el remitente suplantado, dos destinatarios y una copia oculta dan `400`; el
  bueno sigue dando `200`.
- **Las cuatro comprobaciones de `relevo.ts` se mutaron** —comparar el remitente solo por la dirección,
  dejar de mirar las copias, quedarse con la última cabecera, comparar el secreto con `===`— y las
  cuatro se ponen rojas.

**Y comprobado el 2026-09-29, con el cartero desplegado y el camino entero:**

- **`cartero.webcafeina.com` en pie**, con su red de borde, su sitio de Caddy —validado antes de
  recargar— y su certificado. Los tres sitios vecinos siguieron sirviendo durante todo el proceso.
- **El cierre, desde fuera y no solo desde dentro**: `401` sin secreto y con uno equivocado, `400` con
  el remitente suplantado y con dos destinatarios, `404` en cualquier otra ruta. Y **el secreto no
  aparece en el registro de Caddy**, que es lo que el filtro del sitio tenía que conseguir.
- **El camino entero**: el Worker compone, el cartero entrega y el correo llega a la bandeja con su
  código. **Alta bloqueante: 638-931 ms**, contra los cientos de milisegundos de un `POST` a Resend.
  El cartero tarda 0,27-0,32 s de los suyos.
- **Con el cartero parado**: `502` en **439 ms** con «Prueba otra vez en un momento», y se recupera
  solo al levantarlo. Era un punto abierto en `deuda.md` y deja de serlo.
- **Dos fallos que solo encontró esto**, y los dos con la misma forma —una prueba en verde que no podía
  verlos—: `CORREO` sin recortar, que dejaba el Worker contestando `202` con el correo cayendo en el
  buzón de pruebas; y `fetch` guardado como propiedad, que Workers rechaza con «Illegal invocation»
  porque exige `globalThis` como `this`. **Las seis pruebas del cartero inyectan un doble, y un doble
  no tiene ese problema.** Los dos tienen ahora su prueba, y la del `fetch` se mutó para comprobar que
  caza el fallo.

**Y comprobado en producción, el mismo día:**

- **Un alta de verdad entregada en 769 ms** y el correo en la bandeja con su código.
- **Y otra después de retirar Resend del DNS**, en 859 ms, con `d=webcafeina.com; s=google`, `spf=pass`
  y `dmarc=pass`. Borrar los tres registros de Resend no rozó nada: el SPF del dominio **no lo
  incluía** —Resend firmaba el sobre con su propio subdominio `send`—, así que la parte peligrosa, que
  era tocar el `include` de Google y el de Brevo, no llegó a existir.

**Lo que no se ha comprobado, y hay que decirlo:**
- **Que la invitación siga muda, de punta a punta.** Se sostiene **por construcción** —va en
  `ctx.waitUntil` con el resultado descartado, y `mandar` no lanza—, y eso se comprobó leyendo el
  código; pero ejercitarla exige una cuenta de verdad, y el suite que la crea corre contra un servidor
  local que no pasa por el cartero.
- **El camino de «cupo»**: agotar diez mil correos para ver el `550 5.4.5` no se va a hacer. Si Google
  lo dijera con otro texto, diríamos «prueba otra vez en un momento» cuando la verdad es «mañana» — el
  mismo fallo que la 0041 vino a arreglar, ahora en una esquina mucho más improbable.
- **La entregabilidad a un mes.** Que el primero llegue a la bandeja no dice qué pasará después.
- **Que la comparación del secreto sea de tiempo constante.** Está escrita para serlo; eso una prueba
  no lo mide.
- **Qué pasa cuando el cartero no está.** El Worker contesta que no se ha podido mandar, y eso está
  probado con un doble; **no se ha visto de verdad** con el servicio caído.
