# ADR 0043 — La identidad para compartir: una semilla dentro de la bóveda

**Fecha:** 2026-09-23 · **Estado:** aceptada, en uso desde la entrega B2 · **Continúa la [0035](0035-las-cuentas.md)**
y la [0040](0040-la-extension-cliente-de-la-cuenta.md) · **Revisar cuando** se mande la primera copia de verdad

## Contexto

La [ADR 0035](0035-las-cuentas.md) decidió con el cliente que compartir sería **una copia sin permisos**:
al recibirla es de quien la recibe, y si cambia se vuelve a mandar. Para eso hace falta que cada cuenta
tenga **una identidad criptográfica**: una llave pública con la que cifrar lo que se le manda y otra con
la que firmar lo que manda.

El plan (`docs/cuentas.md`) decía que esa identidad se crearía en la fase A, y **la tabla de entregas
afirmaba que la A2 la había hecho**. No es verdad: al empezar la B se buscó y no hay semilla, ni Ed25519,
ni HPKE en ninguna parte. Lo único que existe es **un hueco en el protocolo**: al registrarse se puede
mandar un campo `llaves`, el servidor lo guarda y **no lo lee nadie**. Así que esto se construye entero.

## Decisión

### Dónde vive

**Una semilla de 32 bytes al azar, en una sección `identidad` del cuerpo cifrado de la bóveda.** De ella
sale todo lo demás, y por eso no hace falta guardar ninguna llave: se derivan cuando se necesitan.

- Va **dentro del cuerpo**, que es lo único que el servidor no puede leer.
- Las versiones que no la entienden **la conservan**, porque el contenido guarda las secciones
  desconocidas tal cual (`Extra`).
- **Se crea una sola vez y no cambia nunca.** Cambiarla es cambiar de identidad: lo que te mandaron antes
  se queda sin abrir.

### Qué se deriva

Con HKDF-SHA256 sobre la semilla, con dos etiquetas distintas:

| Para | Etiqueta | Algoritmo |
|---|---|---|
| Recibir (cifrar hacia ti) | `esfinge/identidad/cifrado/v1` | **X25519** |
| Firmar lo que mandas | `esfinge/identidad/firma/v1` | **Ed25519** |

### Con qué se cifra: `DHKEM(X25519) · HKDF-SHA256 · ChaCha20-Poly1305`

El plan dejaba la elección abierta entre **X-Wing** (`MLKEM768X25519`, resistente a cuántica pero
borrador) y el clásico de RFC 9180. Se elige el clásico, y el motivo es el mismo que ha decidido media
arquitectura de este proyecto: **la bóveda existe dos veces** (ADR 0040), y lo que haya aquí hay que
escribirlo también en TypeScript para el navegador. Go 1.27 trae `crypto/hpke` con las dos, pero en el
navegador X25519 y ChaCha20-Poly1305 son una biblioteca pequeña y ML-KEM es otra cosa.

**La suite viaja como un campo del sobre**, así que el día que haya ML-KEM razonable en el navegador se
añade sin romper lo ya mandado.

### La huella

Para comparar identidades por otro canal —es lo único que protege el primer envío— cada identidad tiene
una **huella legible**: el SHA-256 de `suite + llave de cifrado + llave de firma`, en grupos cortos y con
el alfabeto de la clave de recuperación, que ya está pensado para leerse en voz alta y no confundir
caracteres.

Es **TOFU con huella comparable**: la primera vez se confía, y quien quiera estar seguro compara la huella
por teléfono. Lo que eso no protege —un servidor malicioso que sustituya una llave en el primer envío, si
nadie compara— se dice tal cual en `docs/seguridad.md`.

### Cuándo se publican las llaves

**Al sincronizar, no al entrar en «Compartir»** (B2d). Esto parece un detalle de implementación y no lo
es: quien nunca ha mandado nada tiene que poder **recibir**. Si sus llaves no estuvieran publicadas, el
servidor le daría a quien le manda **unas inventadas pero fijas** —que es como no dice quién tiene cuenta y
quién no— y el sobre llegaría cifrado hacia nadie. Los dos lados harían lo suyo bien y el buzón enseñaría
«No se puede abrir» sin que ninguno pudiera entender por qué.

Así que las publican **la ventana al arrancar la sincronización** y **la extensión en cada pasada**, las
dos de cortesía: que falle no puede parar nada. La extensión apunta la huella publicada para no repetir el
envío cada minuto.

### Y a quien todavía no tiene cuenta, una invitación (entrega B3)

Hasta aquí faltaba la mitad: **mandar una copia a una dirección sin cuenta no hacía nada**, y lo hacía en
silencio. El servidor devuelve para ella unas llaves inventadas pero fijas —así es como no dice quién está
en Esfinge—, de modo que el sobre salía cifrado hacia nadie y se tiraba.

Lo que se hace, decidido con el cliente el 2026-09-24:

- **El servidor le manda una invitación** con **quién le invita** y un botón para crear su cuenta. Lo eligió
  así frente a un correo anónimo: sin nombre, un correo que pide darse de alta en algo no lo abre nadie.
- **El sobre no sale y no espera en el servidor.** Se queda una nota dentro de la bóveda de quien lo manda
  —sección `envios`, cifrada como todo lo demás y sincronizada a sus equipos—, y el sobre se manda de verdad
  cuando esa dirección publica sus llaves. Lo que dispara es **que la huella cambie**: mientras sea la
  inventada, es siempre la misma.
- **Lo que se dice al mandar vale para los dos casos.** «Si ya tiene cuenta, le espera en su buzón; si no,
  le hemos mandado una invitación.» No es vaguedad: distinguirlos exigiría que el servidor contestara
  distinto, y eso convertiría compartir en una forma de averiguar quién tiene cuenta.

### Cómo se funde

`identidad` es una sección más del contenido, pero con una regla propia: **no cambia nunca**, y si dos
equipos la crearan a la vez —cada uno la suya, antes de sincronizarse— gana **la de fecha menor**, y si
empatan, la de huella menor. Simétrico y sin sorpresas: los dos equipos llegan al mismo sitio.

## Alternativas descartadas

- **X-Wing (`MLKEM768X25519`)**, que es mejor criptografía. Descartada por el navegador, no por Go, y con
  la puerta abierta en el campo `suite`.
- **Derivar la identidad de la clave de bóveda**, sin semilla propia. Es una llave menos que guardar, pero
  ata dos cosas que deben poder cambiar por separado: hoy la clave de bóveda no cambia nunca, y si algún
  día se recifra (ver la revisión de septiembre), la identidad no tiene por qué irse con ella.
- **Guardar las llaves ya derivadas** en vez de la semilla. Ocupa más, se puede desincronizar y no aporta:
  derivarlas cuesta microsegundos.
- **Que el sobre viaje con la invitación**, cifrado con una clave al azar que va en el enlace del correo.
  Es lo que hacen otros y llega aunque quien la manda no vuelva a abrir Esfinge nunca. Descartada por el
  cliente con la razón dicha en una frase: **la contraseña pasaría por el correo**, así que quien tenga
  acceso a ese buzón —o su proveedor— la tiene. En un gestor de contraseñas eso es justo lo que se vende no
  hacer. El coste de lo elegido está en las consecuencias.
- **Que el servidor guarde el sobre hasta que haya cuenta.** No sirve: está cifrado hacia unas llaves que no
  abre nadie, y guardarlo solo sería guardar basura.
- **Que `POST /v1/envios` diga si esa dirección tenía cuenta**, que haría la pantalla mucho más clara
  —«entregada» o «invitada»—. Descartada: sería una forma de enumerar correos sin más coste que un envío, y
  es exactamente lo que el resto del servidor cuida.
- **Una identidad por equipo** en vez de por cuenta. Sería más fino —revocar un equipo revocaría su
  identidad— pero obliga a cifrar cada envío para cada equipo del destinatario y a que el emisor sepa
  cuántos tiene. La cuenta es la unidad que el cliente eligió para todo lo demás.

## Consecuencias

- **La identidad nace con la bóveda, no con la cuenta.** Una bóveda local ya la lleva dentro, y el día que
  entre en una cuenta no hay que crear nada ni pedir nada.
- **Una bóveda restaurada de una copia vieja conserva su identidad**, que es lo que se quiere: lo que le
  mandaron sigue abriéndose.
- **Quien cambie de bóveda cambia de identidad**, y a quien le hubieran mandado algo antes tendrá que
  pedir que se lo manden otra vez. Es el precio de que la identidad viva en la bóveda y no en el servidor.
- **El servidor sabe con quién compartes.** Hay que decirlo en la política (B4).
- **Y desde la B3 manda correo a terceros en tu nombre**, con tu dirección dentro. Eso es nuevo y hay que
  decirlo igual: Webcafeína le cuenta a alguien que no es cliente suyo que tú usas Esfinge. Lo frena un tope
  de cinco invitaciones por cuenta y día, pero el hecho no se quita con un tope.
- **Una copia a quien no tiene cuenta no llega el mismo día**, y puede no llegar nunca: hace falta que
  Esfinge se abra en alguno de tus equipos mientras la nota siga viva, y la nota dura treinta días. Es el
  precio de que la contraseña no pase por el correo, y se dice en la pantalla.
- **Quien manda no sabe si ha llegado**, ni siquiera si esa dirección tenía cuenta. Es deliberado y es la
  misma decisión de no delatar quién está en Esfinge.

## Verificación

**Comprobado** en esta entrega: que la semilla se crea una sola vez y sobrevive a guardar, cerrar y abrir;
que las llaves derivadas son estables —la misma semilla da las mismas llaves en Go y en TypeScript, byte a
byte, en las pruebas cruzadas—; que la huella coincide en los dos; y que fundir dos bóvedas con identidades
distintas deja siempre la misma, la mire quien la mire.

**Comprobado en la B2**: que una cuenta le manda una copia a otra y la otra la abre, en Go y en
TypeScript; que el sobre de otra identidad no se abre y que uno manipulado se rechaza por la firma; y, con
**la extensión cargada de verdad** en un Chromium contra el servidor local, que lo que te mandan espera en
el buzón del panel, que no se rellena hasta pulsar «Guardar», y que desde el panel se manda una copia que
la otra cuenta abre con la contraseña dentro.

**Comprobado en la B3**: que a quien no tiene cuenta le llega la invitación con quién le invita y dónde
crearla; que **la respuesta del servidor no cambia** aunque el tope de invitaciones salte, y que el correo
va por `waitUntil` para que tampoco lo diga el tiempo; que invitar dos veces manda un correo; que la nota
espera dentro de la bóveda **sin el secreto** y las dos implementaciones la funden igual (prueba cruzada al
azar); y, de punta a punta, que la copia sale sola cuando esa persona crea su cuenta —en Go con dos
aplicaciones, y con **la extensión cargada de verdad** contra el servidor local—.

**Sin comprobar**: **la huella no la ha leído en voz alta ningún par de personas**, que es la única prueba
que vale de que se puede comparar por teléfono. Y **nadie ha recibido la invitación en un buzón de verdad**:
aquí el correo se lee de una tabla, así que cómo se ve el botón en Gmail o en Outlook, y si el correo cae en
spam, solo lo dice mandarlo.
