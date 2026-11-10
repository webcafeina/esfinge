# 0059 · La clave de contenido: que quitar un acceso signifique algo

**Fecha:** 2026-11-09 · **Estado:** aceptada · **publicación A hecha, la B por hacer** · **Arregla la [0052](0052-dar-acceso-a-un-proyecto.md) · Toca la [0038](0038-sincronizar-la-boveda.md), la [0040](0040-la-extension-cliente-de-la-cuenta.md), la [0050](0050-varias-bovedas.md), la [0053](0053-quitar-el-acceso-borra-la-copia.md) y la [0057](0057-la-boveda-en-un-servidor.md)**

## Contexto

La [0052](0052-dar-acceso-a-un-proyecto.md) prometió que al quitarle el acceso a alguien **se rota la clave
de la bóveda**. Eso no existe: `RetirarAcceso` pone una lápida en su ranura y nada más, y la clave con la
que se cifra el cuerpo **es la misma antes y después**. Quien tuvo acceso una vez se la llevó y la conserva,
así que una copia del fichero que consiga por otro camino —un respaldo, un disco, lo que ya se bajó antes de
la [0053](0053-quitar-el-acceso-borra-la-copia.md)— **la abre para siempre, incluido lo que se escriba
después**.

Se encontró el 2026-10-21 leyendo el código para otra cosa, estaba prometido en **seis sitios** y es lo
único grande que le quedaba al proyecto. Lo que protege hoy es el 403 del servidor y el borrado de la copia:
una puerta y un acuerdo, no un cifrado.

**Y el argumento heredado de por qué no se hizo es falso para el caso que importa.** La 0052 dice que rotar
obligaría a reenvolver todas las ranuras, y que dos no se pueden reenvolver sin tenerlas delante: la
**maestra** y la de **recuperación**. Pero una **bóveda de proyecto no tiene ninguna de las dos** —su única
ranura propia es la que abre la bóveda personal de su dueño (0050)— y el acceso compartido **solo existe
ahí**. Así que ese nunca fue el obstáculo.

El de verdad es otro, y es el que decide el diseño: **la fusión**.

- `Fundir` descifra el documento del servidor con la clave de aquí. Si un equipo rota y otro no, el segundo
  no puede leer lo que baja.
- Ese equipo puede reabrir desde su ranura y conseguir la clave nueva, pero **su fichero local sigue
  cifrado con la vieja**: pierde lo que no hubiera subido.
- Y si dos personas con «editar» rotan a la vez, la fusión elige **una** ranura por fecha —con resolución de
  un segundo—: el cuerpo cifrado con la otra se queda ilegible. Pérdida de datos en silencio.

## Decisión

**El cuerpo se cifra con una clave de contenido, y la clave de bóveda pasa a envolver la cadena de todas las
que ha habido.** La clave de bóveda **no cambia nunca**, así que cualquier equipo abre siempre su fichero
local y el remoto; lo que rota vive **dentro** de la bóveda, al lado de las anteriores.

```
B Ó V E D A   D E   P R O Y E C T O   (formato 2)

  boveda-principal ──envuelta con la clave de la personal──▶ clave de bóveda ─┐
                                                              (no cambia nunca) │
                                                                               ▼
                                                            sobre «contenido»: la cadena
  acceso:<titular> ──sellada hacia su identidad──▶ la misma cadena ───────────▶ clave de la época actual
                                                                                        │
                                                                                        ▼
                                                                                 sello  +  cuerpo
```

El formato está en [`formato-boveda.md`](../formato-boveda.md). Lo que sigue es lo que se decidió y por qué.

### Cuatro cosas que decidió el cliente (2026-11-09)

| | |
|---|---|
| **Solo bóvedas de proyecto** | Es donde existe el acceso compartido. La personal se queda en el formato 1: no se toca su recuperación, ni Touch ID, ni la prueba de posesión, ni cómo la abre la extensión |
| **Se guardan todas las claves** | No es comodidad: sin la época anterior, un equipo que vuelve de estar desconectado no puede fundir su propio fichero |
| **Dos publicaciones: leer y luego escribir** | La disciplina de la [0048](0048-las-llaves-de-acceso.md). La extensión se publica aparte y puede ir por detrás |
| **Rota al revocar y a mano** | El botón no es un extra: es la herramienta de migración de los accesos ya dados |

### El formato se reparte en dos números, y eso destapó una bomba

`FormatoQueSeLee` es lo máximo que se entiende y `FormatoQueSeEscribe` lo máximo que se escribe. Con uno
solo no se puede publicar «primero leer y luego escribir», que es lo que permite que un equipo actualizado y
otro sin actualizar convivan.

Al partirlo salió un fallo latente: los tres sitios calculaban `soloLectura` **al revés** —`doc.Formato <
Formato`—, o sea que una bóveda de un formato **más viejo** se abría sin poder escribir. No se notaba porque
el único formato que existía era el 1, y **el día que se publicara un 2 todas las bóvedas que existen se
habrían vuelto de solo lectura**, con el cliente descubriéndolo con la suya delante.

**Y la regla vive en una función** (`soloSeLee`) y no dentro del constructor, por una razón de método: con
los dos números a 1, el caso que importa —se lee y no se escribe— **no se podía construir**, así que una
prueba metida en el constructor habría tenido que esperar a existir el caso, o sea a después de publicarlo.
Una prueba que no puede nacer hasta después de publicar lo que vigila no es una prueba.

### La época se declara fuera y se autentica dentro

`doc.epoca` va **en claro**, porque hay que elegir la clave antes de poder descifrar nada. Si la declarada
no abre, se prueban las demás de la más nueva a la más vieja —son de `PerfilLlave`, probar cinco es gratis—
y el sello lleva **la misma** dentro: si no cuadran, es «la bóveda no cuadra por dentro». Así el campo de
fuera es **una pista, no una autoridad**, igual que `serie`.

Y el sello se cifra con la clave de contenido, no con la de bóveda: el titular lo necesita para comprobar la
coherencia y para preparar la subida, y una «clave de sello» que no rotara dejaría a un revocado fabricando
sellos para siempre.

### Quien recibe un acceso ya no se lleva la clave de bóveda

Una ranura `acceso:<titular>` del formato 2 lleva dentro **la cadena**, no la clave de bóveda. Eso es el
objetivo entero y no un efecto secundario: mientras el acceso entregaba la clave de bóveda, quitarlo no
significaba nada **y además se estaba regalando la clave que protege la contraseña maestra**.

De ahí sale algo que la 0052 decía y el código no permitía: **quien puede editar puede dar acceso a otros.**
`PonerAcceso` exigía la clave de bóveda, que es justo lo que un titular no tiene; ahora exige **con qué
cifrar el cuerpo**, que es lo que de verdad hace falta. Y por lo mismo, la subida se sella con la clave de la
época: con la de bóveda, un titular no podría subir nada.

**Las dos formas de ranura conviven a propósito.** Migrar deja las viejas como están, porque con la clave de
bóveda que llevan dentro se desenvuelve la cadena igual: es lo que hace que migrar no necesite red ni las
identidades de nadie. Y **quién es cuál lo dice el claro** —si empieza por `{"esfinge":"claves-de-contenido"`
es la nueva—, nunca el `codificacion` del JSON de fuera, que vive donde cualquiera que pueda escribir el
fichero lo cambia.

### Al fundir, los sobres que llevan claves salen de un solo lado

Es la pieza donde un error se paga en datos. Cuando los dos lados declaran épocas distintas, `contenido` y
**todas** las `acceso:*` se resuelven **en bloque** por el lado que gana la época —el que gane su sobre
`contenido`, que es también la clave con la que queda cifrado el cuerpo—.

Resolviéndolos uno por uno, el cuerpo puede quedar cifrado con la clave de un lado y la ranura de un titular
llevar dentro la del otro: esa persona abre su sobre, saca una cadena sin la época del cuerpo y **no abre su
propia bóveda**.

Dos cosas que el bloque **no** se lleva por delante:

- **La liveness la siguen decidiendo las reglas de siempre**, porque una lápida no se resucita ni por el
  lado que pone la época: si no, quitarle el acceso a alguien aquí y rotar en otro equipo se lo devolvería.
- Y si el lado que pone la época **no tiene ranura viva** para esa persona —se le dio el acceso aquí
  mientras allí se rotaba—, la ranura **se cae**. Lo del otro lado lleva una clave con la que el cuerpo no se
  abre, y aquí no se puede sellar la buena porque hace falta su identidad: **una fusión elige contenedores y
  nunca fabrica cifrado**. Está en la deuda, porque la ventana todavía no lo dice.

Y quien no puede llegar a la clave nueva **se le dice qué hacer**: el dueño la saca del propio documento con
la clave de bóveda, pero un titular no puede —la suya va sellada hacia su identidad, que al fundir no está
delante—, así que se le pide **cerrar y volver a abrir**, que es por donde pasa. No `ErrManipulada`, que
manda a mirar donde no hay nada.

### Una bóveda pasa al formato 2 solo al rotar, y lo dispara una persona

No hay migración perezosa dentro de `guardar()`: un cambio de formato que ocurre por su cuenta es un cambio
que nadie decidió, y el primero en enterarse sería el compañero cuyo Esfinge deja de abrir la bóveda.

**Y migrar, solo, no cierra nada**, que hay que decirlo: las ranuras `acceso:` de antes se quedan, y llevan
dentro la clave de bóveda. Eso es lo que hace la migración barata y es también por lo que **el botón manual
es la herramienta de migración**: el dueño lo pulsa una vez después de actualizar, y a partir de ahí nadie
más tiene la clave de bóveda.

## Alternativas descartadas

- **Reenvolver la maestra y la de recuperación en cada revocación**, que es lo que la 0052 dio por
  imposible: **no aplica**, porque una bóveda de proyecto no tiene ninguna de las dos. Y donde sí las hay
  —la personal— sería pedir la contraseña en cada revocación y **una ceremonia de clave de recuperación por
  revocación**, que es justo lo que la 0050 descartó.
- **Una clave en vez de una cadena.** Más simple y rompe la fusión: un equipo que vuelve de estar
  desconectado no podría descifrar su propio fichero local, y perdería lo que no hubiera subido.
- **Un contador en vez de un identificador al azar.** Dos equipos del mismo dueño rotando a la vez desde la
  época 2 crearían los dos «la 3» con claves distintas, y al unir las cadenas una taparía a la otra —o peor,
  se quedaría la etiqueta de una con la clave de la otra—. Con identificador único la cadena es un
  **conjunto** y la colisión no existe.
- **Unir las dos cadenas al fundir.** Obligaría a **resellar** el sobre `contenido`, y `ESF1` lleva sal y
  nonce al azar: los dos equipos darían bytes distintos y se pasarían la bóveda sin fin (0038). Se queda la
  del lado que gana, y lo que cuesta está en la deuda.
- **Decidir la época con la base, a tres bandas**, como las demás ranuras: habría que descifrar la base
  **antes** de saber con qué clave descifrarla. Y no hace falta: una rotación siempre escribe un `contenido`
  nuevo, así que «el más reciente» y «el que cambió respecto de la base» dicen aquí lo mismo.
- **Migración perezosa al guardar**, descartada arriba.
- **Una clave de sello aparte que no rote**, para que el titular pudiera comprobar la coherencia sin tener
  la del cuerpo: dejaría a un revocado fabricando sellos para siempre.

## Consecuencias

- **Un titular abre, lee y escribe, y no tiene la clave de bóveda**: `LlaveParaProyectos`, `Posesion` y
  `SellarSecreto` se niegan, y ninguna de las tres hace falta en una bóveda de proyecto. Lo que decide si se
  puede trabajar pasa a ser **la cadena** (`cerrada`), y lo que es del dueño se niega aparte
  (`sinClaveDeBoveda`): las 39 guardas que preguntaban por la clave de bóveda se partieron en dos, una a una.
- **La época filtra cuántas veces se ha vuelto a cifrar esta bóveda.** Va en claro y no hay alternativa:
  hay que elegir la clave antes de descifrar.
- **Nada escribe el formato 2 todavía.** En la publicación A una bóveda así se abre entera y **no se
  guarda**, con la frase que ya existe; y fundir se niega con un remoto de ese formato, porque fundir acaba
  guardando.
- **El espejo de TypeScript crece** (`navegador/src/nucleo/claves.ts`), y con él dos trampas que costaron
  encontrarse: `leerDocumento` reconstruye el documento campo a campo, así que una época sin nombrar se
  pierde al guardar; y el sello se armaba **en dos sitios** en los dos núcleos —al guardar y al preparar la
  subida—, así que el campo nuevo entró en uno y no en el otro. Ahora hay un solo sitio en cada núcleo.
- **Dos cosas en la deuda**, las dos de la fusión entre dos épocas: el acceso que se cae sin que la ventana
  lo diga, y la copia `.antes-de-fundir` de una época que se pierde si dos ramas rotaron a la vez.
- **Y la 0052 se queda con su promesa tachada hasta la publicación B**: rotar no existe todavía, así que
  hasta entonces lo que protege sigue siendo el 403 y el borrado de la copia.

## Verificación

**Lo que se ha comprobado de verdad**, todo con la puerta de Go y la de la extensión en verde —27 paquetes y
200 pruebas de Playwright—, y **cada protección mutada**:

- **Un titular abre con su cadena y no se lleva la clave de bóveda**, y la clave de contenido **no** abre la
  bóveda por la puerta del dueño. La mutación que paga esta pieza es la precisa: las dos primeras salieron
  rojas **fallando al abrir**, o sea que la aserción que importa no llegaba a ejecutarse.
- **Dar acceso en el formato 2 sella la cadena**, y **un titular que puede editar da acceso a otro**, que
  antes el código no permitía.
- **La fusión con dos épocas**, con cuatro pruebas: el cuerpo se queda con la clave que gana y **el fichero
  se vuelve a abrir**; las ranuras de acceso salen del lado de la época aunque la de aquí sea más nueva; una
  lápida no revive; y a un titular sin la clave nueva se le dice que cierre y vuelva a abrir. Las cuatro
  mutaciones caen cada una en su aserción.
- **Las dos pruebas del formato asimétrico**, que son las que la fase 0 no podía escribir: una bóveda de un
  formato que solo se lee se abre, se lee y **no se guarda**; y fundir se niega con un remoto de ese formato
  —también en el espejo, comprobando **el motivo** y no solo que falle—.
- **Las cruzadas con el generador ampliado**: ranuras de acceso en sus tres estados —de antes, con cadena y
  lápida—, sobres `contenido` y las dos épocas, y **cada núcleo calculando el lado por su cuenta**. Con eso,
  quitarle al espejo la resolución en bloque pone en rojo **121 de 401 casos**.
- Y de paso, **una divergencia que llevaba mes y medio dentro**: Go decidía «del mismo segundo, gana la
  lápida» (0052) y **el espejo no tenía esa regla**, así que los dos núcleos fundían distinto ese caso — se
  pasarían la bóveda sin fin y la extensión le devolvería el acceso a quien lo había perdido. No lo decía la
  cruzada porque su generador **no hacía ni una ranura de acceso ni una lápida**. Quitarla otra vez pone en
  rojo **33 de 401**.

**Lo que no se ha comprobado, y hay que decirlo:**

- **Rotar no existe.** Es la publicación B, y con ella `RetirarAcceso` rotando, `Desprender` desmigrando, la
  pantalla y el paseo de dos equipos contra el servidor de verdad. Hasta entonces, **la deuda sigue
  abierta**: esta publicación prepara el formato y no cierra el agujero.
- **Las dos épocas no pueden darse en producción hoy**, porque nada escribe el formato 2: toda esa fusión la
  ejercitan las pruebas y nadie más. El escenario se crea con un ayudante (`rotarAMano`) que hace lo que hará
  `RotarClaveDeContenido`, y **eso es exactamente la trampa del ayudante que hace lo que hace producción**:
  cuando exista la de verdad, ese ayudante se tira y se llama a aquélla.
- **El vector congelado** de una bóveda de proyecto del formato 2 —dos épocas y un titular, que no se
  regenera, como los del formato `ESF1` ([0022](0022-vectores-fijos.md))— **no se puede grabar todavía** por lo mismo. Se graba en la
  B, con el código que lo escribe.
- **Una flota mezclada de verdad** —un Mac con la A y otro sin ella— y que las tiendas hayan aprobado la
  extensión. Eso lo dice el cliente actualizando.

## Anotado y no construido

Llevar la clave de contenido a la **bóveda personal** cerraría un agujero que está escrito en
[`seguridad.md`](../seguridad.md): hoy rotar la clave de recuperación **no protege**, porque la clave de
bóveda no cambia nunca y quien tenga la vieja más una copia anterior del fichero —la `.anterior` de cada
guardado, un respaldo, una versión del servidor— saca de ahí la clave de bóveda y abre la bóveda de ahora.
Con clave de contenido, rotar la recuperación puede traer época nueva y la clave vieja se queda con el
contenido viejo. **Es una época más y ningún invento**, y queda aquí para decidirlo con calma.
