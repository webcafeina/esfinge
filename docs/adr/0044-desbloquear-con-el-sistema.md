# ADR 0044 — Desbloquear con el sistema: un cerrojo, dicho por su nombre

**Fecha:** 2026-09-24 · **Estado:** aceptada, C1 y C2 hechas, C3 por escribir · **Es la fase C de la
[0035](0035-las-cuentas.md)** · **Matiza la [0014](0014-comprobacion-de-actualizaciones.md)**, cuyo «no se firma»
es lo que decide todo lo de aquí · **Revisar cuando** se reabra lo de firmar en macOS, o cuando el
cliente diga qué pregunta su Mac al actualizar

## Contexto

El plan de cuentas (`docs/cuentas.md`) dejó para el final «Touch ID o Windows Hello y PIN, como ranuras
solo locales», **con un aviso puesto a propósito**: «puede exigir firmar la aplicación, en contra de la
decisión de no firmar; hay que investigarlo antes». Se investigó el 2026-09-24, y el estudio entero —con
sus fuentes— está en [`../desbloqueo-del-sistema.md`](../desbloqueo-del-sistema.md).

Lo que salió cambia de qué va la fase, y por eso hay ficha:

> **Sin firmar, en los dos sistemas, «desbloquear con el sistema» es un cerrojo y no una llave.** Pide la
> huella y, si el sistema dice que sí, Esfinge abre la bóveda con una clave que estaba ahí de todos modos.
> Protege de quien se sienta delante de tu ordenador desbloqueado. **No** protege de un programa que corra
> como tú, que es de lo que sí protege hoy la contraseña maestra.

En macOS el camino fuerte —Secure Enclave, o control de acceso biométrico en el llavero de protección de
datos— **exige la entitlement `keychain-access-groups`**, que solo lleva una compilación firmada con un
perfil de aprovisionamiento; sin ella `SecItemAdd` devuelve -34018. En Windows, `KeyCredentialManager`
funciona sin empaquetar, pero Microsoft dice que en un Win32 sin empaquetar **la credencial está atada a
la cuenta de usuario y no a la aplicación**: otro programa tuyo que sepa su nombre la usa. Su propia
recomendación es contraseña **y** Hello, no Hello en lugar de la contraseña.

## Decisión

### Se hace, y se dice lo que es

**No se firma en macOS** (cliente, 2026-09-24, reafirmando la 0014), **se hace en los tres sistemas**
—Touch ID, Windows Hello y, en Linux, nada— y **la pantalla donde se activa dice que es un cerrojo**, con
esas palabras y no en la documentación solamente. Lo mismo en `seguridad.md`.

### Sin PIN

**«Siempre será la contraseña maestra»**, dicho así por el cliente. Lo único que la sustituye es la
biometría del sistema; donde no la haya —Linux, un Mac sin Touch ID, un Windows sin Hello— se teclea la
maestra y **la pantalla lo dice en vez de ofrecer algo que no está**.

### La ranura nunca es la única

`llavero-del-sistema` es un tipo de ranura más, de los que el formato admite desde la
[0023](0023-la-boveda.md), y **no se sube nunca** (`ranurasLocales` en Go, `RANURAS_LOCALES` en
TypeScript). La maestra y la clave de recuperación siguen abriendo, y quitar el desbloqueo no puede dejar
a nadie fuera. Activarlo **exige la bóveda abierta**, que es lo que impide ponerlo sin saber la maestra.

### En macOS, dos piezas: `LAContext` pregunta y el llavero guarda

`LAContext.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics)` pide la huella y devuelve **un sí o
un no**. El secreto de 32 bytes que envuelve la ranura vive en el **llavero de inicio de sesión**, el de
toda la vida, que no necesita entitlements. Sin fallback a la contraseña del Mac: lo que esto promete es
la huella.

### Y el orden importa: primero el sistema, después la ranura

`ActivarDesbloqueo` le da a guardar al sistema **antes** de poner la ranura, y deshace lo de fuera si la
ranura falla. Al revés quedaría una ranura que no abre nadie y un botón que promete algo que no funciona.
`QuitarDesbloqueo`, al contrario, **quita la ranura aunque el sistema falle al borrar**: lo que importa es
que la puerta se cierre, y lo que quede suelto en el llavero son bytes que ya no abren nada.

### Y la huella se pide sola al llegar, con la pantalla hecha para ella

**Decidido con el cliente el 2026-09-25, probándolo en su Mac.** La 2.27.0 lo ofrecía como **un botón
más** debajo del primario, y su respuesta fue que eso no es desbloquear con el sistema: *«el uso del Touch
ID debe ser más visual, no con un botón. Que primeramente te ofrezca el Touch ID si está activado, con una
animación visual, de huella o algo»*. Así que:

- **El diálogo del sistema sale al llegar a la pantalla**, sin pulsar nada, como hace Dashlane. Lo eligió
  él con el coste delante: **para teclear la contraseña maestra hay que cancelar el diálogo primero**, y
  sale otra vez cada vez que la bóveda se cierra por inactividad.
- **La huella manda en la pantalla**: un dibujo de 72 px dentro de su tarjeta, con la maestra debajo
  separada por un «o». Son dos caminos para lo mismo, no un paso detrás de otro.
- **Y late mientras espera**, de dentro hacia fuera, que es lo que dice «pon el dedo». En reposo —después
  de cancelar— se queda quieta: una pantalla que parpadea sin que nadie haya pedido nada es un nervio.
- **El dibujo es nuestro.** El glifo de Touch ID es de Apple; esto son arcos a trazo en `currentColor`,
  por la misma razón por la que los glifos de los gestores no son sus logotipos.

### Y la bóveda lo ofrece sola la primera vez

**Decidido con el cliente el 2026-09-25**, al probar la 2.27.1: *«de primeras debería sugerirte activar el
Touch ID si es un Mac (o el método que corresponda según el sistema)»*. Sin eso, desbloquear con el sistema
es una función que **solo encuentra quien ya la estaba buscando**: el interruptor vive en Ajustes.

Una tarjeta arriba de la bóveda abierta, con tres reglas:

- **Solo donde se puede.** Si el equipo no tiene biometría no se ofrece nada, y **no se dice que no la
  tiene**: es ruido sobre algo que no se puede arreglar.
- **Una vez.** Se apunta en las preferencias —locales, como la propia ranura— al contestar, con cualquiera
  de los dos botones. Vuelve mientras no se conteste, y no vuelve nunca después.
- **Y dice lo que es**, repitiendo lo de Ajustes: es un cerrojo, y la maestra y la clave de recuperación
  siguen haciendo falta. Ofrecerlo sin decirlo sería venderlo.

Que la marca sea local importa: en un equipo sin biometría la sugerencia no tiene sentido, y haberla
descartado en el portátil no dice nada del ordenador de la oficina.

**Y también en la pantalla de desbloquear, como casilla** (2.27.3). El cliente esperaba encontrarlo ahí, y
tenía razón en dónde: esa pantalla es el momento en que estás a punto de teclear la maestra otra vez. Lo
que no cabe ahí es un botón que lo active, porque **activarlo exige la bóveda abierta** —es lo que impide
encenderlo sin saber la maestra—. Así que lo que hay es **«Abrir con Touch ID a partir de ahora»**: se
marca, se teclea la maestra, y al abrir queda activado. La condición de seguridad se cumple entera, y no
hace falta un aviso que solo informe.

Dos reglas de esa casilla:

- **Marcarla no cuenta como haber contestado.** Lo que apunta «ya se ofreció» es activarlo o decir «ahora
  no» en la tarjeta; si la casilla lo apuntara, quien la deja sin marcar se quedaría sin la tarjeta y sin
  saber que la función existe.
- **La casilla no activa: lo pide.** Activa la tarjeta de dentro, nada más montarse. La primera versión
  (2.27.3) activaba en la propia pantalla de desbloquear y **se tragaba el error**, porque esa pantalla
  desaparece de todos modos. El cliente marcó la casilla, entró, y la tarjeta le ofreció otra vez lo que
  acababa de pedir, sin nada que explicara por qué; aquí no se pudo reproducir —el llavero de mentira no
  falla— y leyendo el código tampoco salió la causa, que es exactamente lo que pasa cuando algo se calla.
  Ahora activa quien tiene pantalla y sitio para el error, y de paso se ve «Activando…» y después «Touch ID
  activado» en vez de que no pase nada visible.

### Y esto no cuenta como actividad

Leer el secreto es parte de abrir, y abrir ya toca el reloj por su cuenta. Es la regla de siempre —lo que
se repite solo no cuenta— aplicada aquí.

## Alternativas descartadas

**Firmar la aplicación (99 $/año).** Compra el camino fuerte —Secure Enclave y llavero de protección de
datos— y de paso quita el aviso de Gatekeeper de la primera instalación. Es una decisión de producto, no
técnica, y el cliente la mantuvo cerrada: «de momento no firmaremos Mac». Queda dicho aquí para que
reabrirla sea una frase y no una investigación otra vez.

**El secreto en un fichero de 0600 junto a la bóveda.** Era lo predecible: sobrevive a las
actualizaciones, no pregunta nada, y se puede probar entero en esta máquina. Se descartó al escribir la C2
por una razón que conviene no volver a discutir:

> Con el secreto en un fichero, **quien copie tu carpeta abre la bóveda sin poner el dedo**. Eso no es un
> cerrojo peor: es quitar la puerta. Activar Touch ID pasaría de «no tengo que teclear la maestra» a «mi
> bóveda la abre quien se lleve mi carpeta de usuario», y eso es **bajar** la protección de la bóveda a
> cambio de una comodidad.

En el llavero de inicio de sesión está cifrado con la contraseña de macOS. Sigue siendo un cerrojo, pero
**un disco copiado no lo lleva dentro**.

**Un PIN de cuatro o seis cifras.** No puede proteger una bóveda por sí solo: el espacio es tan pequeño
que probarlos todos contra el fichero es instantáneo, y Argon2id no arregla eso —subir el coste hasta que
un millón de intentos duelan haría que abrir tardara minutos—. Solo vale si algo limita los intentos, y
eso solo lo hace el hardware: el Secure Enclave o el TPM, o sea los dos problemas de arriba otra vez.

**Un PIN como desbloqueo rápido dentro de una sesión ya abierta**, que es la salida honesta que usan los
demás: no protege la bóveda, protege un atajo, y al reiniciar se pone la maestra. Daba la misma protección
real **sin una línea de código nativo** y se podía probar aquí entera. El cliente la descartó igual: lo
único que sustituya a la maestra ha de ser la huella, y no un número corto.

## Consecuencias

- **Hay dos ficheros de cgo que esta máquina no compila**, y uno de ellos para un sistema que nadie ha
  ejecutado nunca. La lección de [`vidrio_darwin.go`](0017-vidrio-solo-en-el-marco.md) manda aquí entera: que
  el trabajo de macOS pase en verde **solo dice que compila, no que arranque**.
- **Puede preguntar al actualizar.** La lista de aplicaciones de confianza de un elemento del llavero se
  ata a la firma de quien lo guardó, y Esfinge no está firmada: macOS puede pedir permiso tras cada
  actualización, porque el binario cambia. Se contesta «Permitir siempre». Si preguntara en cada apertura,
  o fallara, el arreglo es **borrar la ranura y pedir que se active otra vez** —que es lo que ya hace el
  código cuando el secreto deja de abrir— y decirlo en la pantalla.
- **Quien activa esto deja de escribir su contraseña maestra**, y una contraseña que no se escribe se
  olvida. Por eso la pantalla dice que la maestra y la clave de recuperación siguen haciendo falta.
- **Y no cambia nada de la sincronización**: la ranura es de un equipo y no sube, así que activarlo en un
  Mac no lo activa en el otro ni le da al servidor una segunda puerta.

## Verificación

**Lo que se ha comprobado de verdad:**

- Pruebas de Go de la C1: la ranura abre, **la maestra sigue abriendo**, no viaja en `PrepararSubida`,
  cancelar no deja nada a medias y un secreto que ya no abre se olvida. La ranura del sistema se envuelve
  con `PerfilLlave` y la maestra con `PerfilInteractivo`, **leído en la cabecera ESF1 del fichero** —8192
  KiB contra 65536— y no medido por lo que tarda, que es lo que hacía la primera versión de esa prueba y
  fallaba sola.
- e2e de las dos pantallas —el interruptor de Ajustes y el botón de desbloquear— en los dos temas, con un
  llavero de mentira detrás (`llavero.DeMentira`, y `cmd/dev -sin-llavero` para el caso de que no haya).
- `TestLoQueCruzaElPuenteEstaEnLaLista` cubre los cuatro métodos nuevos, y **`AbrirBovedaConElSistema` está
  en esa lista a conciencia**: abre la bóveda sin la contraseña maestra.
- Que el cgo de macOS **compila y enlaza** contra LocalAuthentication y Security: `compilar.yml` entero en
  verde en los tres sistemas el 2026-09-24.
- Y los seis objetivos de la línea de comandos, que ahora cruza `make comprobar`.

**Comprobado en el Mac del cliente con la 2.27.0 (2026-09-25):** arranca, se activa, abre con la huella,
cancelar vuelve al campo de la contraseña sin error, la maestra sigue abriendo y al quitarlo desaparece.

~~**Y al activarlo el llavero no pregunta nada.**~~ **Falso, corregido el mismo día**: sí pregunta. Al
guardar la llave, macOS pide **la contraseña del Mac** para autorizar a Esfinge a usar el llavero, y con
«Permitir siempre» no vuelve a preguntarlo. Se dio por comprobado con una sola pasada —la primera
activación, que crea el elemento y no pide nada; pedirlo es acceder a uno que ya está—. La lección es de
método y no de macOS: **«no preguntó» tras hacerlo una vez no es «no pregunta»**.

Y con eso, **la explicación más probable del fallo de la 2.27.3**: la casilla activaba justo al entrar en la
bóveda, así que ese diálogo del sistema salía **sin que nada lo hubiera anunciado y en el peor momento**.
Quien lo cancela —que es lo sensato ante una petición de contraseña que aparece sola— deja la activación
fallando, y el `catch` vacío se tragaba el error. No está probado, pero encaja con todo lo observado y con
que dejara de pasar al mover la activación a otro momento. Ahora **se avisa antes de que salga**, en la
tarjeta, en la casilla y en Ajustes, y **solo en macOS**.

**Lo que no se ha comprobado, y solo puede comprobarse en un Mac de verdad:**

- ~~**Qué pregunta el llavero al actualizar.**~~ **Contestado el 2026-09-25, y la respuesta es que sí
  pregunta**: después de actualizar Esfinge, **la primera vez que se abre la bóveda con la huella macOS pide
  la contraseña del Mac**. Con «Permitir siempre» no vuelve a pedirla **hasta la siguiente actualización**.
  Es lo que cabía esperar sin firmar —cada versión es un binario nuevo y el permiso del llavero se ata a
  quién lo pide—, y la 0014 lo dijo antes que nadie: firmar no es solo quitar el aviso de Gatekeeper.

  **Consecuencia, y es la que esta decisión existía para tomar: hay que decirlo en la pantalla.** Una
  petición de la contraseña del sistema al poner el dedo, sin nada que la explique, se lee como que algo va
  mal justo en el programa donde eso importa más. Se dice **solo cuando toca** —entre una actualización y el
  primer desbloqueo con huella que funcione— y **solo en macOS**.
- **Cómo se ve el latido en el Mac**, y si el diálogo del sistema al llegar resulta cómodo o cansa cuando
  la bóveda se cierra sola varias veces al día. Eso es uso, no una prueba.
- **Cómo se lee el diálogo del sistema** —el motivo que se le pasa sale en pantalla— y si Touch ID responde
  cuando se espera que responda.
- **Windows entero**: la C3 no está escrita, y cuando lo esté seguirá sin haberse ejecutado nunca en un
  Windows.
