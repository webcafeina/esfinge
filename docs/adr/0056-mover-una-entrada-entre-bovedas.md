# 0056 · Llevar una entrada de una bóveda a otra

**Fecha:** 2026-10-15 · **Estado:** aceptada · **Continúa la [0055](0055-la-boveda-se-nombra-en-cada-accion.md)**

## Contexto

La [ADR 0055](0055-la-boveda-se-nombra-en-cada-accion.md) hizo que la bóveda se pudiera **nombrar** en cada
acción. Y lo primero que el cliente quiso hacer con varias bóvedas fue **llevarse algo de una a otra**:

> *«Guárdame mi credencial de Brevo en Zeri's.»*

No se podía. Lo que el agente tuvo que inventarse —y lo contó él mismo— fue crear la entrada **a medias**,
pedir aprobación para **sacar la contraseña al portapapeles** y pedirle al cliente que la pegara a mano. Y
acto seguido le avisó del resultado: *«vas a tener la misma credencial en dos bóvedas. Si un día cambias la
contraseña en Brevo y actualizas solo una de las dos, la otra se queda desfasada sin que te avise nada»*.

**El rodeo es peor que la funcionalidad**: obliga a una persona a manejar el secreto, lo saca al
portapapeles y deja un duplicado que nadie sincroniza. El propio agente lo pidió: *«te propongo añadirle una
función para mover o duplicar entradas entre bóvedas sin que el secreto pase por Claude, porque ahora es
justo lo que falta»*.

## Decisión

**`esfinge_mover`**: lleva una entrada **entera** —contraseña, segundo factor e historial— de una bóveda a
otra. `boveda` dice de dónde sale; `destino`, a dónde va.

**El secreto no cruza el canal.** Lo lee y lo escribe Esfinge por dentro: `Boveda.Ver` devuelve el original
y lo que lo recorta es la capa del agente, por la que esto no pasa. Es la misma promesa que copiar al
portapapeles —el agente actúa sin ver (ADR 0054)— y aquí ni siquiera sale del programa.

Lo decidido con el cliente, que no se cambia sin preguntar:

| | |
|---|---|
| **Mueve, y copia si se le dice** | De fábrica mueve. Con `dejar_copia` deja también la del origen — porque duplicar una credencial es lo que crea el problema que el propio agente señaló |
| **La del origen se va del todo** | No a la papelera: en un movimiento no se pierde nada, porque está en el destino, y deshacerlo es moverla de vuelta |
| **Pregunta el agente, no la ventana** | *«Debe preguntarte, pero el agente en la conversación, como las bóvedas. No en la ventana»* |

**Y eso cambió el mismo día**, al probarlo: su agente auditó los esquemas y señaló que **mover es más
destructiva que borrar** —quita del origen **sin pasar por la papelera**— y que sin embargo `borrar` pide
aprobación en la ventana y `mover` no pedía nada. La inconsistencia era nuestra: esta ADR argumenta «mover
es crear y borrar, y borrar ya pregunta» y luego se construyó saltándose justo esa pregunta. El cliente
eligió la frontera: **se aprueba lo que quita**. Con `dejar_copia` no quita nada y va directo.

**Y no pasa por la válvula ni por «No volver a preguntar»**, al revés que `borrar`. No es una excepción
inventada: la razón por la que `borrar` sí pasa la dio el cliente el 2026-10-08 —«de todos modos iría a la
papelera»— y aquí **no va**, así que la razón no vale. Es la misma regla que el código de un solo uso y que
cambiar una contraseña: lo que no se deshace no pasa por la válvula.

**No es administrar**: mover es crear y borrar, que el agente ya hace. No crea, archiva, entrega ni comparte
bóvedas.

## Alternativas descartadas

**Que fuera a la papelera, como cualquier borrado** (ADR 0026). Es la regla general y aquí no vale: la
papelera existe para que un clic no pierda una contraseña, y en un movimiento no hay nada que perder. Lo que
de verdad lo decidió es el caso de uso: con papelera, **la contraseña del cliente se queda treinta días en
la bóveda personal de la que se la quería sacar**.

**Pedir el sí en la ventana**, como para copiar un secreto. Se ofreció con su argumento —mover a una bóveda
**compartida** (ADR 0052) se lo enseña a todas las personas con acceso, y eso no se deshace quitándolo
después— y el cliente eligió que preguntara el agente en la conversación, como ya hace con las bóvedas.

**Y eso cuesta algo que hay que dejar escrito**: una frase en la descripción de una herramienta **la lee el
modelo y puede saltársela**; una tarjeta en la ventana la impone Go. Para «en qué bóveda escribo» el coste
de saltársela es una entrada donde no iba. Aquí sería **un secreto en manos de otra persona**. Se compensa
con lo único que no depende del modelo: **el movimiento se apunta en el registro de la bóveda**, que lo
escribe Go, así que «¿qué ha hecho un agente con mis secretos?» se sigue pudiendo contestar.

**Copiar en vez de mover, a secas.** No resuelve lo que lo provocó: sacar la credencial de un cliente de la
bóveda personal se quedaría sin hacer, y el duplicado es justo el riesgo que el agente avisó.

## Consecuencias

**Se escribe en dos bóvedas en una sola orden**, y el orden importa: **primero el destino y después el
origen**. Si el destino falla no se ha tocado nada; al revés, un fallo en medio perdería la entrada. Y si
falla el segundo paso **se dice con todas las letras** —«se ha copiado a X y no se ha podido quitar de Y,
así que está en las dos»—, porque callarlo dejaría a alguien creyendo que ha sacado una credencial de su
bóveda y no.

**La entrada estrena identificador.** Los de una bóveda son suyos y llevarse el de la otra podría chocar.
**La fecha de creación se conserva**: es la misma credencial cambiada de sitio, y en las llaves de acceso esa
fecha es lo único que distingue dos filas idénticas en pantalla.

**Quitar del origen deja lápida**, por `Borrar` + `BorrarDelTodo`. Sin ella, el equipo de al lado tiene la
entrada viva y en la siguiente sincronización **la resucita**: mover dejaría la credencial del cliente de
vuelta en la bóveda personal sin que nadie lo hubiera pedido. Los dos pasos existen a propósito en
`BorrarDelTodo` —«una entrada viva se borra en dos pasos»— y aquí se dan los dos de golpe porque no se
pierde nada.

**Y va por fuera de la envoltura de conmutación de la 0055**, como `generar`: se ocupa él de las dos
bóvedas —lee en una, escribe en otra y vuelve— bajo un solo cerrojo. Envolverlo habría hecho que conmutara
dentro de su propio cerrojo y **se bloqueara contra sí mismo**.

## Verificación

- **Que llega entero**: se mueve una credencial con contraseña, segundo factor y sitio, y se **abre el
  fichero del proyecto** para mirar lo guardado, no lo que quedó en memoria.
- **Que no queda nada detrás**: ni en la lista ni **en la papelera**, que es donde se quedaría la contraseña
  treinta días. Mutado quitando el segundo borrado.
- **Que el secreto no sale por el canal**: la respuesta se serializa y se busca la contraseña dentro, sobre
  los bytes. Y el centinela del canal sigue pasando por todas las respuestas.
- **Que `dejar_copia` copia y no mueve.** Mutado.
- **Que una bóveda que no existe no mueve nada**, ni a medias ni a la abierta.
- **Que el campo llega**: `TestLoQueDeclaraElEsquemaLlegaALaPeticion`, que arma los argumentos desde el
  propio esquema.
- **Y la guarda de las descripciones pide a `mover` su propia frase** —«antes de mover, siempre»— y no la de
  los demás: su destino va siempre nombrado, y el que puede ir sin nombrar es el **origen**. Meterle la
  frase de los otros habría sido poner una palabra para que el test callara.

- **Que aprobar deprisa no deja un bucle** (2026-10-16): se pide desde una bóveda que no es la abierta, **se
  aprueba y el respiro vence después** —el orden de quien lo vive— y el reintento mueve. Las dos versiones
  anteriores de esta prueba pasaban en verde con el bucle dentro: la primera ejercitaba la dirección que no
  necesita conmutar, la segunda forzaba el respiro **antes** de aprobar. El porqué del arreglo, que no es de
  este verbo sino del permiso entero, está en la **ADR 0054**, «Qué vida tiene un sí».
- **Que las dos fechas de la entrada se conservan** (2026-10-19). Mover no cambia nada de dentro —el secreto,
  el segundo factor y el historial son los mismos—, así que poner la fecha de cambio a hoy es escribir un
  dato falso **justo en la columna que alguien mira para decidir si rota una contraseña**. `Poner` la pisa, que
  es lo correcto para una edición, y `PonerNueva` la repone: es el único sitio donde se sabe que el contenido
  no se ha tocado. La prueba mueve **de ida y de vuelta**, como lo hizo el agente del cliente, y **espera a que
  cambie el segundo** antes de empezar: sin eso, un movimiento dentro del mismo segundo pasaría por bueno
  pisando la fecha. Mutado.

  **El identificador sí cambia a propósito**, y eso no se toca: los de una bóveda son suyos y llevarse el de
  la otra podría chocar. Lo que cuesta es que una referencia guardada a la entrada deja de valer tras el
  movimiento, y por eso la respuesta dice el nuevo.
- **Que el registro dice cómo se autorizó.** Decía solo «hecho», de cuando mover no pedía nada, y eso no
  contesta la pregunta para la que el registro existe. **Lo vio el paseo a mano** leyendo el registro al
  final, no una prueba. Mutado.

**Lo que no se ha comprobado**: mover **a una bóveda compartida de solo ver** —tiene que contestar su motivo
propio—, por el mismo coste que ya bloquea otras filas de `deuda.md`: hacen falta dos cuentas. Y que esto
resuelva de verdad el caso del cliente, que lo dice él usándolo.
