# ADR 0048 — Las llaves de acceso, la sexta clase de entrada

**Fecha:** 2026-09-29 · **Estado:** aceptada; **la P1 escrita, sin ver en un Mac** · Primera de las cuatro
entregas de `docs/passkeys.md` · **Revisar** al empezar la P2, que es cuando aparece el navegador

## Contexto

El cliente pidió las passkeys el 2026-09-23: *«la opción que tiene Dashlane del passkey, como por ejemplo al
acceder a GitHub, que simplemente sale un banner de passkey que reconoce Dashlane y es darle a Aceptar»*.
Se investigó ese día en `docs/passkeys.md`, que concluye que **no es una tarea, es una fase**, y se partió
en cuatro entregas. **Ésta es la primera**, y es la única que no se ve: la llave de acceso dentro de la
bóveda, sin navegador.

Se publica sola **a propósito**, y ésa es la decisión menos obvia de la ficha: mete el formato nuevo en
todos los equipos y en la extensión **antes** de que existan llaves de verdad. Cuando empiecen a aparecer,
ninguna copia de Esfinge se encontrará una clase que no entiende. Es barato ahora y caro al revés.

## Decisión

**Una clase nueva, `llave`, con los campos de una credencial WebAuthn**: `rpId`, `idCredencial`,
`idUsuario`, `nombreVisible`, `algoritmo` y `clavePrivada`. Los nombres del protocolo se dejan sin traducir,
como `TOTP`: tienen que poder compararse con la especificación sin nada por el medio.

### Sin contador de firmas

WebAuthn define un contador por credencial para que un sitio detecte una llave clonada. **No se guarda**, y
se firmará siempre con cero, que es lo que hacen Dashlane, 1Password y Bitwarden.

La razón no es la pereza: **una llave sincronizada entre equipos no puede llevar un contador coherente**
—dos equipos firman a la vez y lo descuadran—, así que la detección que se perdería no funcionaría de todos
modos. Y tiene una consecuencia técnica que vale la entrada entera: **esta clase no necesita ninguna regla
de fusión propia**. Un contador monótono habría sido el primer campo de la bóveda con reglas a medida,
escritas dos veces, y con ellas la prueba de fusión al azar de tres equipos habría dejado de salir gratis.

### La clave privada no se enseña, no se copia y no cruza

Es la única clase cuyo secreto **no tiene ninguna pantalla**: de una llave no hay nada que leer ni que
teclear en ningún sitio. `vaciarLoSensible` la vacía, así que no llega ni a la ficha; y cuando llegue la P2,
lo que salga del trabajador será **la firma ya hecha**, nunca la clave — igual que `CodigoDeBoveda` cruza el
código de seis cifras y jamás la semilla.

Del identificador de credencial y del de usuario se vacían también. No son secretos —el sitio los emitió él—
pero no hacen falta en ninguna lista, y lo que no hace falta no viaja. **El sitio y el nombre visible sí se
quedan**: son lo único por lo que alguien puede reconocer una llave en una lista donde no se enseña nada
más.

### No se puede crear a mano

`llave` no sale en el selector de «Nueva». Una llave no se inventa: la emite el sitio. Una escrita a mano no
abriría nada, así que ofrecerla sería ofrecer una entrada que no puede funcionar. El sitio y la cuenta se
enseñan en el editor **desactivados**: que estén dice qué llave es ésta; que no se toquen dice que
cambiarlos aquí no cambiaría nada allí.

### Exportar: aparte y cifrada

La exportación en claro **no las lleva**, y lo dice en su aviso. Una fila de CSV con una clave privada
dentro sería lo más peligroso que Esfinge escribiera nunca en el disco —lo que abre la cuenta, en texto— y
además **no le serviría a ningún gestor**, porque ninguno sabe leerla.

Salen por `ExportarLlaves`, en un contenedor **ESF1** con su propia clave, reutilizando `internal/cripto`
tal cual. Así se sigue cumpliendo «una bóveda de la que no se puede salir es una trampa» sin pagar el precio
de cumplirla en claro. **La clave del fichero no es la maestra** y se pide aparte: quien guarda esa copia la
guarda en otro sitio, y reutilizar la maestra haría que perder el fichero fuera perder la bóveda.

### Compartir sí, con su propio aviso

Lo eligió el cliente y no se discute. Pero **no es lo mismo que compartir una contraseña** y la pantalla lo
dice antes de pulsar: una contraseña compartida se puede cambiar y el sitio avisa del cambio; una llave
compartida es **la identidad en ese sitio**, no se puede revocar desde Esfinge y **el sitio no se entera
nunca de que ahora hay dos**. Para dejar de compartirla hay que borrar la llave en el sitio.

Técnicamente ya funcionaba —`envio.go` manda la entrada entera— y eso era justo el peligro: **funcionaba sin
que nadie lo hubiera decidido**.

## Alternativas descartadas

- **Guardar la llave dentro de la credencial del sitio**, como el código de un solo uso. Menos entradas y
  todo lo de un sitio junto. Se descartó porque un sitio puede tener llave **sin** contraseña —y entonces
  habría que inventar una credencial vacía— y porque compartir o borrar dejarían de poder separarse.
- **Mantener el contador de firmas.** Ver arriba.
- **Exportar las llaves en el CSV de siempre.** Coherente con «se puede salir de todo», y descartado: un
  fichero de texto con claves privadas dentro.
- **No exportarlas en absoluto**, que es lo que hacen los demás gestores. Se descartó con el cliente: deja
  la bóveda sin salida para esa clase.

## Consecuencias

- **Séptima pestaña en la bóveda.** Medido: los siete glifos con el rótulo de la activa ocupan ~475 px de
  los 560 de la columna. Lo vigila la prueba que ya existe, que mide **con cada pestaña activa**.
- **Un glifo más que convive con el de credencial**, y los dos son una llave: lo que los separa es la
  orientación —la de la contraseña va tumbada, la de acceso de frente—. Hay que mirarlo en un Mac.
- **Todo cambio del formato se hace dos veces**, como manda la 0040.
- **La exportación gana una puerta**, y con ella un método más en la lista blanca del puente. Cruza una
  clave: la del fichero. No cruza ninguna clave privada.

## Verificación

**Lo que se comprobó, y cómo**

- **Cinco mutaciones, las cinco rojas**: sin vaciar la clave privada de la papelera, sin sacar las llaves de
  la exportación en claro, escribiendo el claro al lado de lo cifrado, sin la huella propia de la llave en
  el importador, y sin el identificador en `claveDeCuenta`.
- **La prueba de duplicados mira `Conflictos`**, no solo `Metidas`: una huella que choca no impide que la
  entrada entre, la marca y la mete igual. Es la lección de los datos personales, aplicada de entrada.
- **La lista blanca del puente cazó el método nuevo** antes de que nadie se acordara de él, que es para lo
  que está.
- **Las tres pruebas cruzadas**: forma canónica, lo que se vacía y la fusión al azar de tres equipos.

**Y una cosa que encontró la fusión al azar y no era de esta clase:** el espejo de TypeScript leía **todos**
los campos numéricos como si fueran `revision` —el nombre estaba escrito a fuego, de cuando la revisión era
el único número del formato—. `algoritmo` es el segundo número de la historia del formato, y se guardaba
encima de la revisión: la misma bóveda fundía distinto en los dos lados. Estaba ahí desde el principio,
esperando a que hubiera un segundo número.

**Y un fallo que se vio en el Mac con la 2.32.0 ya publicada:** exportar abría el diálogo del sistema y
descubría **al ir a escribir** que no había ninguna llave, así que pedía una clave y un sitio para un
fichero que no iba a existir — con la bóveda recién creada, que es el caso más probable de todos. Arreglado
en la 2.32.1: se mira antes de preguntar, y la ventana lo dice de entrada. La prueba que lo cubre **pasó en
verde al escribirla**, porque el doble del sistema apuntaba el argumento del diálogo y no si se le había
llamado; ahora cuenta las veces.

**Lo que no se ha comprobado y hay que decir**

- **No se ha visto en un Mac**: ni el glifo al lado del de credencial, ni la ficha de una llave, ni la
  exportación cifrada.
- **No hay ninguna llave de verdad todavía.** Todo lo probado son entradas escritas por las pruebas; la
  primera llave de verdad la creará el navegador en la P2, y hasta entonces no se sabe si los campos que se
  guardan son exactamente los que hacen falta para firmar.
