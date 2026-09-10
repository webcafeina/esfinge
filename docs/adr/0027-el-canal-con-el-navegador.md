# ADR 0027 — El canal con el navegador

**Fecha:** 2026-09-10 · **Estado:** aceptada · **Matiza la [0023](0023-la-boveda.md)** ·
**Revisar si** Esfinge se firma, o cuando llegue el relleno de formularios

## Contexto

La bóveda cerró su parte: de Dashlane no queda nada por traer **al escritorio**.
Lo que queda es lo que Dashlane hace **en el navegador**, y eso es la fase 2.

Empezarla rompe a conciencia una frase de la [ADR 0023](0023-la-boveda.md) —«una
bóveda local… sin servidor, sin cuentas, **sin navegador** y sin móvil»— y una
frase que este proyecto tiene escrita en contra de sí mismo, en
`internal/app/dev.go`:

> Un servidor HTTP en el binario del cliente, por local que sea, es una puerta
> que nadie ha pedido.

Ésta se ha pedido. Eso no la hace inocua: la hace **una puerta pedida**, que es
otra cosa y obliga a decir de qué está hecha.

Esta ficha cubre **la entrega 1**: la extensión enseña las cuentas del sitio que
se está mirando y copia lo que se le pida. No toca ninguna página.

## Decisión

**Un socket local, un proceso traductor aparte, y la bóveda solo en la ventana.**

```
página → extensión → esfinge-puente → [socket] → Esfinge → boveda.esfinge
```

### El navegador lanza un proceso; la bóveda la tiene otro

De ahí sale toda la forma. Un navegador solo sabe lanzar un programa y hablarle
por la entrada y la salida estándar. Ese programa —`esfinge-puente`— **no abre la
bóveda**: se conecta a la aplicación que ya está en marcha.

La alternativa era que el puente abriera la bóveda por su cuenta, y no vale por
dos razones que se ven enseguida: la contraseña maestra tendría que llegar desde
el navegador, y cada consulta pagaría medio segundo de Argon2id.

Eso **corrige a medias** lo que `internal/cli/boveda.go` llevaba escrito desde
que existe la bóveda —«la extensión hablará con esto, no con la ventana»—:
acierta en que habla con un binario de línea de comandos y falla en lo demás.

### No es TCP, y viene apagado

Un socket de dominio unix en la carpeta del usuario. Sin puerto: no se alcanza
desde otra máquina, ni desde otra sesión, ni desde una página web que pruebe
direcciones locales. Y el interruptor está en Ajustes, apagado de fábrica, al
lado de las otras dos salidas —con la diferencia dicha: aquéllas **salen**, ésta
**abre**—.

### En la entrega 1 no sale ningún secreto

La contraseña **la copia Esfinge** al portapapeles del sistema; por el canal solo
vuelve cuántos segundos tardará en borrarse. Dos cosas de una: ningún secreto
vive en el navegador, y el borrado del portapapeles de la 2.12.0 vale también
aquí —copiando desde la extensión se habría quedado ahí para siempre, que es el
agujero que aquella versión vino a tapar—.

Cuando llegue el relleno hará falta un verbo que **sí** devuelva la contraseña,
porque para escribirla en un formulario hay que tenerla. Ese día tendrá que
justificarse solo; hoy no hace falta y no está.

### Qué se puede pedir, y nada más

Cinco verbos con su lista y su prueba, igual que el puente con la ventana:
`estado`, `emparejar`, `cuentas`, `copiar-secreto`, `copiar-codigo`. **Todo lo
que toca la bóveda lleva origen**, incluido el código de un solo uso —en el
primer borrador no lo llevaba, y era un agujero de los que se cuelan por parecer
un detalle: los identificadores se enumeran preguntando por cuentas—.

El origen lo da el navegador, no la página, y se empareja por **dominio
registrable** con la lista de sufijos públicos. Cortando por el último punto,
`foo.github.io` y `bar.github.io` serían el mismo sitio.

### El permiso se da en la ventana, una vez

La primera vez que un navegador habla, Esfinge lo pregunta **en su ventana**, que
es lo único de todo esto que la página no puede tocar. El testigo se entrega una
sola vez y se puede retirar en Ajustes.

**Y no es teatro, aunque tampoco es lo que parece.** Cualquier proceso del
usuario puede abrir el socket y leer el testigo del disco. Lo que el permiso
compra es que «cualquier cosa instalada te vacía la bóveda en silencio» pase a
ser «tiene que pasar por un aviso que no esperabas». 1Password resuelve esto
comprobando la firma del navegador; aquí no se puede, porque **Esfinge no está
firmada** ([ADR 0012](0012-sin-firmar.md)).

### Preguntar no cuenta como actividad

Una extensión pregunta sola: al cambiar de pestaña, cada vez que revive su
trabajador. Si eso moviera el reloj del bloqueo, **navegar mantendría la bóveda
abierta para siempre**. Es la tercera vez que aparece esta regla —el goteo de
iconos, el código de un solo uso y esto— y ya está escrita como regla en
`CLAUDE.md`.

### Y con la bóveda cerrada, no se levanta la ventana

Se contesta «cerrada» y ya. Traerla al frente estaba en el primer plan y es un
error: cualquier proceso podría entonces **hacer aparecer el diálogo de la
contraseña maestra cuando quisiera**, que es la forma de enseñarle a alguien a
teclearla en cuanto una ventana se lo pide.

## Alternativas descartadas

- **Un servidor HTTP local**, que es lo que ya existe tras la etiqueta `dev`.
  Cualquier proceso —y con un fallo de CORS, cualquier página— llega a un puerto.
  Un socket con permisos es una frontera mejor y no necesita autenticación
  propia para lo mismo.
- **Que el puente abra la bóveda.** Arriba: la maestra acabaría en el navegador.
- **Una tubería con nombre en Windows**, que es lo que usa KeePassXC. Sus tres
  ventajas —saber quién se conecta, un descriptor de seguridad propio, y que
  nadie se adelante— no se pagan solas: la primera no se usa en **ningún**
  sistema, la segunda la da el perfil del usuario y la tercera se consigue
  preguntando. A cambio pedía doscientas líneas de llamadas al sistema de Windows
  **en la frontera de seguridad de un gestor de contraseñas, escritas en una
  máquina donde no se pueden ejecutar**. Así se rompió la 2.9.1.
- **Un subcomando oculto de `esfinge`** en vez de un binario aparte, que es lo
  que pedía la ADR 0001. Duró dos commits: `esfinge` es un binario de consola y
  Chrome lanza el host cada pocos minutos, así que parpadearía una ventana negra
  en Windows cada vez.
- **Verificar la firma del navegador**, como 1Password. Exige estar firmado, y
  además arrastra soporte permanente: rechaza navegadores de Flatpak, de Snap o
  compilados a mano.
- **Safari.** Sus extensiones se distribuyen dentro de una aplicación firmada.

## Consecuencias

- **Esfinge escucha por primera vez.** Está dicho en `docs/seguridad.md`, con lo
  incómodo delante: con el canal encendido, sacarle secretos a una bóveda abierta
  pasa de exigir leer la memoria de otro proceso a exigir un `connect()`. Mismo
  atacante, menos trabajo.
- **Y el manifiesto que declara el puente vive en una carpeta que el usuario
  puede escribir.** Un programa con sus permisos puede reescribirlo y ponerse en
  medio. Contra eso no protege nada de aquí.
- **Hay un binario más** —`esfinge-puente`— que hay que instalar en los tres
  sistemas.
- **La extensión y la aplicación se actualizan por caminos distintos**: una por
  una tienda con revisión de días, la otra empujando una etiqueta. Van a estar
  descompasadas casi siempre, y por eso el protocolo lleva número de versión desde
  el primer día. De ahí sale también una regla: **toda la lógica con consecuencias
  vive en Go**, donde publicar un arreglo es empujar una etiqueta.
- **En Windows falta el manifiesto**, que allí va al registro. El socket sí
  funciona. Está en `docs/deuda.md`.

## Verificación

- **En Go**, el emparejamiento de dominios con su tabla de casos: `banco.es`
  contra `www.banco.es` sí, `mail.google.com` contra `accounts.google.com` sí,
  `banco.es` contra `banco.es.malo.com` **no**, `foo.github.io` contra
  `bar.github.io` **no**. Y las direcciones donde no se rellena: `http://`, una
  IP, un sufijo público, `about:blank`.
- **Que no se consigue lo que no toca**, cada caso fallando **por su motivo**:
  sin emparejar, con un testigo inventado, sin origen, desde otro sitio, sobre
  texto claro. Si uno falla por el motivo equivocado, el día que se arregle otra
  cosa dejará de fallar.
- Que **con la bóveda cerrada no sale ni un título**, y que se distingue «no hay
  bóveda» de «está cerrada».
- Que **treinta preguntas seguidas no impiden que la bóveda se cierre**.
- Que **por el canal no vuelve el secreto** al copiar: se comprueba sobre el JSON
  de verdad, no sobre los tipos.
- Que hay **tope de preguntas por minuto**, que existe por la enumeración: pedir
  las cuentas de un dominio no da secretos, pero con un diccionario se reconstruye
  la lista de sitios de la bóveda, que es lo que la [ADR 0024](0024-iconos-de-los-sitios.md)
  cifró en disco.
- Del traductor: el enmarcado de cuatro bytes por los dos lados, que **por la
  salida estándar solo sale el protocolo**, y que sin Esfinge al otro lado se
  contesta en vez de morirse.
- Del socket: que dos Esfinges no se pisan, que un socket huérfano no bloquea
  para siempre, y que **parar cierra las conversaciones abiertas** —cerrar el
  oyente no las cierra, y eso colgó `Parar()` nada más escribirlo—.
- De los manifiestos: que dicen a quién dejan entrar, que **Firefox y Chromium no
  llevan el mismo campo**, que los seis navegadores de la familia de Chromium lo
  reciben, que **no se toca el perfil de un navegador que no está instalado** y
  que apagar el canal se los lleva.
- Y en la ventana, en los dos temas: que el canal viene apagado, que al
  encenderlo dice dónde escucha y que al apagarlo deja de decirlo.

**Lo que no se ha comprobado, y es lo que importa:** que un navegador de verdad
lance el puente y hable con la bóveda. Aquí no hay navegador con el que probarlo
de punta a punta —el trabajador de una extensión y el `connectNative` no existen
en un Chromium de Playwright sin perfil preparado— así que **eso solo se ve en el
Mac**. Es la misma clase de hueco que la aplicación ensamblada.
