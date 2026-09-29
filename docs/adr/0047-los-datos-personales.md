# ADR 0047 — Los datos personales, la quinta clase de entrada

**Fecha:** 2026-09-29 · **Estado:** aceptada y escrita; **revisada el mismo día, en la 2.31.0** —la
dirección se guarda desagregada y no compuesta— · **vista en su Mac con la 2.31.0** · Cierra la última fila de
[`../deuda.md`](../deuda.md) que quedaba de Dashlane · **Revisar cuando** Esfinge rellene formularios de
compra, que es lo único que convierte estos datos en algo que se usa y no solo que se guarda

## Contexto

Dashlane exporta cinco ficheros y Esfinge sabía leer cuatro. El quinto, `personalinfo.csv`, se rechazaba
entero con «no reconozco ninguna columna», y estaba en la deuda como **una decisión de producto por
tomar**, no como un fallo: son direcciones, teléfonos y fechas de nacimiento, o sea **datos que no son
secretos**, y meterlos en una bóveda de contraseñas era algo que había que decidir, no deducir.

El 2026-09-29 el cliente lo decidió. Se le puso delante lo que cuesta y lo que da:

- **Su fichero son dos filas**: su nombre y un correo. Ni una dirección, ni un teléfono.
- **Esfinge no rellena nada de eso.** Rellena contraseñas y códigos de un solo uso. Un nombre y una
  dirección solo sirven el día que rellene formularios de compra, que hoy no está planteado.
- Se le ofreció la alternativa barata —cada fila como nota segura, solo el importador, una tarde— y la
  descartó: **clase nueva completa**.

Y se descartó con conocimiento del coste, que es lo que hace que esto no haya que volver a discutirlo:
un quinto tipo toca la bóveda de Go **y la de TypeScript**, la forma canónica, la fusión, el formulario,
las pestañas y la exportación.

## Decisión

**Una clase nueva, `personal`, con `correo`, `telefono`, `nacimiento` y la dirección en sus nueve
trozos —`destinatario`, `calle`, `edificio`, `piso`, `puerta`, `codigoPostal`, `ciudad`, `provincia`,
`pais`—, que comparte `nombreCompleto` con la identidad.**

Compartirlo no es ahorrar un campo: es el mismo dato, y buscar «Álvaro» tiene que encontrar el carné y
la ficha personal de una vez.

### Y es la primera clase que no guarda un secreto

Hasta aquí, lo que estaba dentro de la bóveda era **lo que no se puede perder ni enseñar**. Un teléfono
no es eso. Se guarda igual porque es lo que guardaba el gestor al que sustituye, y porque **la
alternativa realista no era tenerlo fuera: era tenerlo en Dashlane**.

Eso mueve un listón que estaba escrito en el código: `sirve()` decidía si un CSV vale la pena mirando si
trae «algo que merezca la pena guardar bajo llave». Ahora lo que decide es **si la bóveda lo guarda**, y
está dicho ahí con esas palabras para que no parezca que alguien añadió cuatro campos a una lista sin
mirar.

### Lo que se vacía y lo que lleva ojo no son la misma lista

Son dos preguntas y se contestan distinto, y conviene que esté escrito porque en la pantalla se ve que
no coinciden:

- **De la lista se vacían los cuatro** (`vaciarLoSensible`): una lista no los necesita, y es lo mismo que
  ya se hace con el número de una tarjeta. **`nombreCompleto` no**, y la razón es que el título de una
  ficha personal **sale del nombre** cuando el fichero no trae otro: vaciarlo no escondería nada mientras
  el título lo repite, en la lista y en la papelera.
- **Con ojo van solo la dirección y la fecha de nacimiento.** Son las dos que no se quieren en pantalla
  con alguien al lado. Un correo y un teléfono tapados serían un ojo para ver el número de uno mismo.

### El tipo se sigue deduciendo de los campos, con dos excepciones escritas

La regla de siempre sigue —está en `CLAUDE.md`, que es donde vive lo del importador porque nunca tuvo
ficha propia—: **un CSV puede mentir sobre sí mismo y los campos no**. Pero **un nombre a secas no tiene ningún campo que lo distinga de nada**, y eso es media
exportación de datos personales: la fila `name` de Dashlane solo trae `first_name` y `last_name`.

Los dos únicos sitios donde se puede saber qué es:

1. **El fichero del que salió**, cuando es `personalinfo.csv`.
2. **La columna `type` que Esfinge escribe al exportar**, que es nuestra y no miente. Se lee **solo** en
   nuestro propio fichero y **en último lugar**, cuando los campos no deciden. Sin esto, exportar una
   ficha que solo lleva un nombre y volverla a importar la convertía en **una credencial sin usuario ni
   contraseña**.

## Alternativas descartadas

- **Cada fila como nota segura.** Una tarde de trabajo, sin tocar el formato ni el núcleo de TypeScript,
  y con todo lo que hoy se puede hacer con esos datos: guardarlos y buscarlos. Se descartó por decisión
  del cliente. Lo que costaría después: sacar un teléfono del cuerpo de una nota para rellenar un
  formulario es una migración con análisis de texto.
- **Meterlo en `identidad`.** Encaja el nombre y no encaja nada más: una identidad es un documento, y un
  correo no tiene número ni tipo de documento. Habría obligado a la misma cantidad de campos nuevos sin
  la claridad de una clase.
- ~~**Guardar la dirección compuesta**, en un solo campo de texto.~~ **Fue lo primero y duró unas horas.**
  Se eligió por el formulario —nueve casillas para escribir algo que se lee de un vistazo— y se cambió el
  mismo día, con el cliente, en cuanto se puso encima de la mesa lo que costaba: el día que Esfinge
  rellene formularios de compra, sacar el código postal de un texto ya compuesto es análisis de texto
  **sobre algo que venía separado y habíamos juntado nosotros**. Componer es de una línea; partir es
  adivinar. Como todavía no había ni un dato guardado, cambiarlo no costó ninguna migración de verdad.
- **No importarlo y que se teclee a mano.** Con dos filas es lo más barato de todo. Descartado por el
  cliente.

### Se guarda por trozos y se lee compuesta

En la ficha, la dirección sale escrita **en el orden del sobre** y con su «Copiar», porque una dirección
se lee y se copia entera; nueve filas con su ojo y su botón serían un formulario dentro de una ficha. Ese
orden **no es el de los campos**: por orden de campo saldría «Calle Mayor 1, España, Madrid, 28001», que
es una lista. Y la provincia no se escribe cuando se llama igual que la ciudad, que en media España es lo
normal.

Eso deja **la misma composición escrita dos veces**, en Go y en la ventana, y conviene decir por qué no
es lo mismo que duplicar el formato: las formas canónicas tienen que dar **los mismos bytes** o las dos
bóvedas se pasan la misma entrada sin fin, y por eso las vigilan pruebas cruzadas. Esto es una frase para
leer: si algún día se separan, lo que pasa es que se lee de otra forma. Mandarla por el puente sería un
campo más en el formato que no guarda nada.

## Consecuencias

- **Esfinge deja de ser solo un gestor de secretos.** Lo que dice de sí mismo en la portada y en
  `seguridad.md` tiene que admitir que guarda datos personales, y eso es lo que cambia, no el código.
- **Las seis pestañas de clases no caben con rótulo, y ya no lo llevan salvo la activa.** Es un cambio
  visible en una pantalla que el cliente ya había aprobado, y es el precio de la quinta clase: la columna
  de contenido está topada en 560 px, los cinco rótulos medían 527 y el sexto pide 160 más. Ver abajo.
- **Todo cambio del formato se hace dos veces**, en Go y en `navegador/src/nucleo/`, como manda la
  [0040](0040-la-extension-cliente-de-la-cuenta.md).
- **La exportación gana doce columnas** y sigue volviendo a entrar de una vez.
- **Y el formulario del dato personal es largo**: catorce campos en una columna. Es lo que cuesta guardar
  por trozos, y es la mitad del cambio que se eligió a sabiendas.
- **Lo que escribió la 2.30.0 se trae solo.** Un campo que desaparece de la estructura no da error: cae en
  `Extra`, se conserva y no se ve nunca más, que es justo la pérdida callada que `Extra` existe para no
  tener —ahí va lo que escribe una versión **más nueva**—. Así que la dirección compuesta entra entera en
  `calle`, con sus saltos de línea, donde se ve y se reparte a mano. Partirla sería adivinar. **Es una
  migración de ida**: la 2.30.0 que vuelva a leer esa bóveda verá la dirección vacía.

## Verificación

**Lo que se comprobó, y cómo**

- **Con el fichero de verdad del cliente**, cabecera entera y sus dos filas tal cual las exportó él, más
  una de teléfono y una de dirección escritas aquí. Antes de tocar nada se ejecutó contra el importador
  de entonces para ver qué hacía: rechazarlo entero.
- **Las cuatro piezas nuevas, por mutación**: quitar la lectura de `type` en nuestro fichero, quitar la
  huella propia del dato personal, quitar la forma del fichero en `tipoDe` y quitar el vaciado de la
  papelera. Las cuatro ponen roja su prueba.
- **Y una que no la ponía**: la de los duplicados miraba `Metidas` y `Repetidas`, y una huella que choca
  no produce ninguna de las dos —produce `Conflictos`, y la entrada entra igual—. La prueba pasaba en
  verde con la huella quitada. Ahora mira las tres.
- **Los mismos bytes en Go y en TypeScript**, con la forma canónica y con la fusión al azar de tres
  equipos.
- **Y una prueba cruzada nueva, `TestCruzadaLoQueSeVacia`**, porque la canónica **no puede** cazar esto:
  ordena las claves, así que un campo que se caiga del espejo de TypeScript vuelve por `extra` y los
  bytes salen idénticos. Lo que se rompe es que `sinSecretos` ya no lo encuentra donde lo borra, y
  entonces **el secreto cruza al panel con todo en verde**. Comprobado quitando dos campos del espejo:
  la canónica seguía verde y la nueva se pone roja. Era un hueco que ya existía para las cuatro clases
  anteriores.
- **La barra de las pestañas se midió, no se razonó.** La captura enseñó el rótulo cortado; medir la
  barra a siete anchos de ventana dijo la causa, que no era la que parecía: la columna **está topada en
  560 px y no crece**, así que el `@media (min-width: 740px)` que devolvía los rótulos preguntaba por la
  ventana cuando lo que manda es la columna. Con cinco clases acertaba por doce píxeles. Hay prueba que
  compara lo que la barra necesita con lo que mide, **con cada pestaña activa**, porque la activa es la
  única con rótulo y mirando solo la primera se comprueba el mejor caso.

**Lo que no se ha comprobado y hay que decir**

- ~~**No se ha visto en un Mac.**~~ **Visto el 2026-09-29** con la 2.31.0 instalada: las pestañas sin
  rótulo salvo la activa, el glifo de la silueta al lado del carné de las identidades y el formulario de
  catorce campos. Las tres se marcaron a propósito como lo que podía chirriar, y el cliente las dio por
  buenas.
- ~~**No consta que se haya importado el fichero del cliente en su Esfinge.**~~ **Importado el
  2026-09-29**, con las dos entradas que trae —su nombre y su correo— entrando bien. Es la comprobación
  que cierra el asunto y que ninguna prueba de aquí podía hacer: lo de aquí se ejercita con la cabecera
  que él pasó, no con el fichero que su Dashlane escribe.
- **La migración de la 2.30.0 no se ha ejercitado sobre una bóveda de verdad**, solo sobre el JSON que
  aquella versión escribía. No había ninguna: se publicó por la tarde y nadie llegó a guardar una
  dirección.
- **Las clases que Dashlane puede sacar y él no tiene guardadas** —dirección, teléfono, y lo que salga de
  `job_title` y `url`— se han escrito a partir de los nombres de las 24 columnas. La dirección se
  ejercita con una fila inventada, así que **el orden del sobre está comprobado contra lo que creemos que
  es una dirección de Dashlane**, no contra una suya.
