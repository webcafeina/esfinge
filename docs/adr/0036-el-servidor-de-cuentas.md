# ADR 0036 — El servidor de cuentas: un Durable Object por cuenta, con la bóveda dentro

**Fecha:** 2026-09-18 · **Estado:** aceptada, sin desplegar · **Continúa la [0035](0035-las-cuentas.md)** ·
**Revisar cuando** se despliegue por primera vez en Cloudflare, y en la auditoría

## Contexto

La ADR 0035 decidió un servidor propio en Cloudflare, en la UE, que guarda la bóveda cifrada de cada
cuenta y no puede leerla. El plan (`docs/cuentas.md`) proponía tres piezas: **D1** para el índice, **un
Durable Object por cuenta** para su estado y **R2** para los bytes de la bóveda, con veinte versiones.

Al escribirlo salió que la tercera pieza sobraba, y que dos cosas del plan no se podían hacer como se
habían escrito.

## Decisión

### La bóveda vive en el Durable Object de su cuenta, en trozos, y no en R2

Una bóveda real pesa cientos de kilobytes, y una de veinte mil entradas ronda los 4 MB. La base SQLite
de un Durable Object admite filas de hasta 2 MB, así que la bóveda se parte en **trozos de 1 MB** y se
guarda al lado de todo lo demás de la cuenta.

La razón no es el tamaño: es **la atomicidad**. Con R2 al lado, subir la bóveda es esperar a otro
servicio **en medio** de «¿sigue siendo la versión 17? pues escribo la 18», y en esa espera cabe otra
petición a la misma cuenta. El plan ya lo apuntaba como trampa. Con todo en la misma base, la
comprobación y la escritura van en **una sola transacción** (`transactionSync`), y lo mismo el cambio
de contraseña: bóveda nueva, verificador nuevo y sesiones revocadas, o nada.

### Qué versiones se conservan

**Las diez últimas y la última de cada uno de los treinta días anteriores**, no «veinte». Con Esfinge
subiendo tres segundos después de cada guardado, veinte versiones son una tarde editando, y lo que se
quiere conservar es margen para deshacer **una fusión mala que se haya propagado**. Con todas, la base
crece sin tope.

### Lo demás, pieza a pieza

- **D1 solo guarda lo que hay que buscar sin saber de qué cuenta se trata**: correo → cuenta, altas a
  medias, contadores por día y la lista de admisión. **La sal y el coste de Argon2id viven en el objeto**,
  no en D1: si vivieran en los dos, cambiar la contraseña tendría que escribir en dos sitios y un fallo
  entre medias dejaría la cuenta sin poder entrar.
- **La pre-entrada va siempre a un objeto**, también con un correo sin cuenta (uno vacío, fijo). Si no,
  el salto de más de las cuentas reales las delataría por el tiempo de respuesta.
- **Los frenos viven donde vive lo que frenan**: por IP, en el enlace `ratelimit` de Workers; por día, en
  D1; por cuenta —diez fallos, cinco códigos por hora, sesenta subidas por hora—, en su objeto. **Ninguno
  en una variable del Worker.** Tras diez contraseñas malas, la cuenta **no se bloquea**: solo entran sus
  equipos de confianza. Bloquearla sería dar a cualquiera la forma de dejar fuera a su dueño.
- **Un solo código de recuperación vivo a la vez**, así quien lo usa no tiene que decir cuál es y nadie
  acumula códigos para probar. Con el código solo se entrega **el sobre de recuperación**, y con la
  prueba de posesión, una **sesión restringida** que solo baja la bóveda y cambia la contraseña.
- **Sin sus secretos no contesta nada** (`503`). Un secreto que falta en JavaScript no da error, da
  `undefined`, y los verificadores quedarían firmados con la palabra «undefined». Es el fallo mudo del
  permiso `storage` de la extensión con peores consecuencias.
- **Nada en los registros**: el Worker no escribe, y los registros de invocación de Cloudflare van
  apagados porque incluyen la petición, y la petición lleva la sesión.
- **Los correos**: el código va en el cuerpo, nunca en el asunto, que se ve en la pantalla bloqueada.

### La jurisdicción, por variable

`workerd` —el motor local de las pruebas y de `wrangler dev`— **no implementa las jurisdicciones** y
revienta al pedirlas. Así que el código la lee de `JURISDICCION`, que solo admite `eu` o vacía; los dos
Workers de verdad la llevan a `eu` en `wrangler.jsonc`, y **una prueba lee ese fichero** y se pone roja
si alguien la quita. La de D1 no está en el código: se elige al crear la base, y no se puede cambiar.

## Alternativas descartadas

- **R2 para la bóveda**, que era el plan. Por la atomicidad de arriba; para los tamaños de una bóveda no
  aporta nada, y es un servicio más que configurar, con su jurisdicción aparte.
- **Todo en D1**, sin Durable Objects. D1 no serializa por cuenta: dos subidas a la vez necesitarían una
  comparación con intercambio en SQL, y el cambio de contraseña, varias escrituras sin transacción entre
  peticiones. Y los frenos por cuenta no tendrían dónde vivir.
- **Un Durable Object único como índice** en vez de D1: sería un cuello de botella global para buscar
  correos.
- **Crear las bases D1 desde aquí** con la integración de Cloudflare: solo permite una «preferencia de
  ubicación», que no es una garantía. Las crea el cliente en el panel, con la jurisdicción de verdad.

## Consecuencias

- **Una cuenta es un objeto**, y borrarla es `deleteAll()` sobre él más la fila de D1. Si la fila no se
  llega a borrar, el alta siguiente con ese correo la reutiliza en vez de fallar.
- El tope de una bóveda es de **8 MB**, muy por encima de cualquier uso real, y viene de la memoria de un
  Worker más que de la base.
- **Los secretos los pone el cliente** en el panel de Cloudflare, generados con `openssl rand -hex 32`.
  Cambiar `PIMIENTA` deja fuera a todas las cuentas: su versión se guarda (`pimienta: 1`) para poder
  rotarla algún día sin eso, pero hoy no hay rotación escrita.
- El puerto de siempre de `wrangler dev`, el 8787, lo ocupa otro programa en la máquina de desarrollo:
  `make servidor` usa el 8790.

## Verificación

**Comprobado aquí**, dentro de `workerd` y con `make comprobar`: **39 pruebas** del alta, la entrada con
código y con equipo de confianza, la bóveda —incluidas **seis subidas a la vez sobre la misma versión, de
las que gana una**—, los trozos de más de 3 MB, las versiones, el cambio de contraseña, la recuperación,
los frenos **abriendo una petición nueva por intento**, los equipos, el borrado, la exportación, el
registro por invitación, el buzón de pruebas cerrado en producción y el servidor sin secretos. Además
**se rompieron a propósito** la comprobación de versión y el freno por cuenta, y las pruebas lo cazaron.
Y levantado con `wrangler dev`: salud, alta con su código en el buzón y pre-entrada.

**Sin comprobar**: nada se ha desplegado. Ni la jurisdicción de verdad —el motor local no la tiene—, ni
el freno por IP de Cloudflare, que en local se simula, ni que Resend entregue los correos, ni el dominio
propio.
