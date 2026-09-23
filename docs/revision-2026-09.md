# Revisión de seguridad, septiembre de 2026

**Qué es esto y qué no.** El cliente decidió el 2026-09-23 no contratar una auditoría externa y pedir en su
lugar una revisión hecha aquí. Se hizo con **cuatro pasadas independientes del modelo Fable** —servidor y
protocolo de cuenta, extensión, formato y criptografía, sincronización y fusión—, cada una sobre el código
de verdad y sin tocarlo, y **cada hallazgo se ha vuelto a comprobar a mano** antes de darlo por bueno.

**Lo que esto no es**: una auditoría externa. Quien revisa aquí es quien ha escrito casi todo el código, y
eso no se arregla leyendo con más cuidado. Lo que una revisión así encuentra son fallos concretos; lo que no
puede darse a sí misma es independencia. `docs/auditoria.md` sigue valiendo el día que se quiera pedir una
de verdad, y la [ADR 0035](adr/0035-las-cuentas.md) sigue diciendo que **abrir el registro pide auditoría
externa**: mientras no la haya, las cuentas siguen por invitación.

## Resumen

| # | Dónde | Gravedad | Estado |
|---|---|---|---|
| 1 | Extensión: un campo escondido por su contenedor se rellenaba solo | Alta | **Arreglado** |
| 2 | Extensión: pisa su bóveda —y pierde lo no subido— si la fusión falla al volver a entrar | Alta | **Arreglado** |
| 3 | Sincronización: «muchos borrados» salta al usar la papelera y **no tiene salida** | Alta | **Arreglado** |
| 4 | Servidor: se puede saber si un correo tiene cuenta (429 frente a 401) | Media | **Arreglado** |
| 5 | Servidor: con solo el correo se deja una cuenta sin códigos una hora | Media | **Arreglado** |
| 6 | Servidor: el testigo de confianza sobrevive al cambio de contraseña | Media | **Arreglado** |
| 7 | Extensión: quién puede pedir lo de la cuenta se decidía por el nombre del puerto | Media | **Arreglado** |
| 8 | Extensión: la tarjeta de guardar se podía pulsar en el instante en que aparece | Media | **Arreglado** |
| 9 | Bóveda: rotar la clave de recuperación no invalida una copia antigua | Media | Decisión: cambiar el texto o recifrar |
| 10 | Fusión: un borrado suave gana a una edición | Media | **Arreglado** |
| 11 | Servidor: el código de alta admite más de cinco intentos en una ráfaga | Media | **Arreglado** |
| 12 | Varios menores | Baja | Ver abajo · cuatro arreglados, cuatro apuntados |

## Lo que ya está arreglado (2.25.5)

### 1 · Un campo escondido por su contenedor se rellenaba solo — **alta**

`navegador/src/campos.ts`. `esVisible` miraba **solo el campo**, y la opacidad no se hereda en el estilo
calculado: dentro de un `<div style="opacity:0">`, el campo decía `opacity: 1`. Lo mismo un campo puesto en
`left: -9999px` o recortado con `clip-path: inset(100%)`.

**Qué permitía**: quien pudiera meter HTML en cualquier página del dominio —un XSS, un subdominio
abandonado— ponía un formulario invisible y se llevaba la contraseña guardada de ese sitio **sin un clic**,
porque con una sola cuenta del sitio Esfinge rellena sola al cargar. El aviso «Rellenado por Esfinge» sale
en las coordenadas del campo: fuera de la pantalla, no se ve. Y `docs/seguridad.md` prometía justo lo
contrario: «nunca uno invisible o de un píxel».

**Arreglado**: se mira `checkVisibility` —que sí mira los ancestros—, que la caja corte el documento, y las
dos recetas con las que se recorta algo sin quitarlo (`clip-path: inset(100%)` y `clip: rect(0,0,0,0)`).
Cuatro casos nuevos en las pruebas, incluido el que **sí** debe rellenarse: un formulario a media opacidad.
Lo que sigue sin detectarse —un recorte cualquiera de un ancestro— está dicho en el código y por qué: las
dos formas de salir de dudas dicen que no se ve cualquier formulario por debajo del pliegue.

### 4 · Se podía saber si un correo tiene cuenta — **media**

`servidor/src/cuenta.ts`. Tras diez fallos, una cuenta real contestaba **429** y un correo sin cuenta
contestaba siempre **401**: once intentos con una contraseña inventada decían si ese correo está registrado.
`docs/seguridad.md` promete que preguntar desde fuera no lo desvela.

**Arreglado**: el freno se mira **después** de comprobar la contraseña. Quien acierta la contraseña ya sabe
que la cuenta existe; quien no, recibe el mismo 401 que un correo desconocido. Los fallos se siguen contando
igual y el freno sigue dejando pasar solo a los equipos de confianza.

### 6 · El testigo de confianza sobrevivía al cambio de contraseña — **media**

`servidor/src/cuenta.ts`. Cambiar la contraseña cerraba las sesiones de los demás equipos pero **no**
tocaba sus testigos de «este equipo es de confianza». Quien hubiera pasado una vez el segundo factor —o
hubiera copiado `cuenta.json`, que lo lleva en claro— seguía entrando **sin código** en cuanto consiguiera
la contraseña nueva. Y cambiar la contraseña porque alguien la sabe es justo el caso.

**Arreglado**: al cambiar la contraseña se van los testigos de los demás equipos, y al recuperar la cuenta
se van todos. Los retos de segundo factor pendientes, también.

### 7 · Quién podía pedir lo de la cuenta se decidía por el nombre del puerto — **media**

`navegador/src/fondo.ts`. «Esto solo se pide desde el panel» se comprobaba mirando el **nombre** del puerto,
que lo elige quien lo abre. No se conoce forma de que una web llegue ahí —no hay `externally_connectable` y
el mundo está aislado—, pero toda la valla se apoyaba en el código que corre en la página del atacante.

**Arreglado**: se mira `sender.tab`, que lo pone el navegador: un guion de contenido lo tiene y el panel no.
Y de paso, **el servidor de la cuenta ya no viaja en el mensaje de entrar**: lo pone la compilación.
Aceptarlo desde fuera era dejar que quien pudiera mandar un mensaje se llevara a otro sitio la clave
derivada de la contraseña maestra.

### 8 · La tarjeta de guardar se podía pulsar nada más aparecer — **media**

`navegador/src/tarjeta.ts`. `isTrusted` demuestra que hubo una persona, no que supiera dónde pulsaba: la
página de debajo puede poner un botón suyo donde va a salir la tarjeta —o encima, con `pointer-events:
none`, para que el clic la atraviese— y quedarse con la decisión, que puede ser «actualizar» o «nunca en
este sitio».

**Arreglado**: la tarjeta no hace caso a un clic que llegue en el primer cuarto de segundo, y hay una prueba
que lo comprueba por los dos lados.

### 12 · Menores ya arreglados — **baja**

- **La contraseña maestra a medias** se quedaba en `storage.session` mientras se iba al correo a por el
  código, y solo caducaba al leerla: ahora se barre en el tic del minuto.
- **«El mismo sitio»** para las ofertas pendientes contaba etiquetas a ojo, así que `evil.github.io` y
  `victima.github.io` eran el mismo. Ahora usa la lista de sufijos públicos, que ya iba dentro.

## Lo que se arregló después (2.25.6 y 2.25.7)

### 2 · La extensión pisa su bóveda si la fusión falla al volver a entrar — **alta**

`navegador/src/concuenta.ts`. Al volver a entrar en la cuenta se baja la bóveda del servidor y se funde con
la de aquí; si la fusión lanza —y el hallazgo 3 hace que lance con facilidad—, un `catch` vacío se queda
con la del servidor y **escribe encima**. Lo que la extensión hubiera guardado y no subido se pierde, sin
aviso y sin copia. La aplicación, en el mismo caso, **aparta** el fichero y dice dónde ha quedado.

**Arreglado**: se hace lo mismo que la aplicación. La bóveda de este navegador se guarda aparte —no se
borra sola— y el panel lo dice: «Se ha apartado la bóveda que había en este navegador… ábrela en la
aplicación de Esfinge para recuperarlo».

### 3 · «Muchos borrados» salta al usar la papelera, y no hay salida — **alta**

`internal/boveda/sincronizar.go` y `navegador/src/nucleo/fundir.ts`. La fusión se niega a aplicarse si se
llevaría más de la mitad de las entradas vivas. El problema es doble:

- **Cuenta la papelera como pérdida.** Mandar tres de cuatro entradas a la papelera —un gesto normal y
  reversible— para el resto de equipos es «media bóveda borrada», y **paran todos**.
- **No hay forma de decir que sí.** La opción existe en el código (`AunqueBorreMucho`) y **no la pone nadie**
  fuera de las pruebas. La ventana enseña «Parada: los cambios de otro equipo borrarían media bóveda» y no
  hay botón. Mientras tanto ese equipo tampoco sube lo suyo.

**Arreglado**, las dos cosas y en las dos implementaciones: perdida es la que **desaparece del fichero**, no
la que pasa a la papelera; y la parada tiene salida, «Juntarlo igual», en la línea de la bóveda de la
aplicación y en el panel de la extensión. El permiso vale para **una sola pasada** y se consume al usarla,
con pruebas que lo comprueban por los dos lados.

### 5 · Con solo el correo, una cuenta se queda sin códigos una hora — **media**

`servidor/src/indice.ts` y `cuenta.ts`. `/v1/recuperacion/inicio` no pide autenticación y el cupo de códigos
por hora es **uno solo** para entrar, recuperar y borrar. Cinco peticiones por hora con el correo de alguien
dejan a esa persona sin poder entrar desde un equipo nuevo ni recuperar la cuenta, y le mandan cinco correos.
Además, pedir otro código de recuperación **invalida el que estuviera usando**.

**Arreglado**: un cupo por propósito —gastar los de recuperación ya no deja sin entrar—, un tope de veinte
códigos al día por cuenta, y **pedir otro código de recuperación ya no mata el anterior**: valen los que
sigan vivos y se comprueban todos. Eso cambia una decisión anterior —«un solo código vivo»—, y el motivo
está escrito al lado de la prueba.

### 10 · Un borrado suave gana a una edición — **media**

`internal/boveda/sincronizar.go`. La ADR 0038 dice que la edición gana al borrado, y es verdad para el
borrado definitivo. Con la papelera no: si un equipo manda una entrada a la papelera y otro le cambia la
contraseña, la fusión campo a campo se queda con las dos cosas —la entrada acaba **en la papelera con la
contraseña nueva**—, desaparece de la lista y a los treinta días se purga.

**Hecho (2.25.7)**: si un lado, respecto a la base, no ha hecho más que mandarla a la papelera y el otro ha
tocado contenido, la entrada se queda fuera de la papelera (`soloALaPapelera`, en Go y en TypeScript). La
prueba borra en un equipo, cambia la contraseña en el otro y exige que la entrada siga viva y con la
contraseña nueva; quitando el arreglo, se va a la papelera.

**Y con ello, `cambiada` se toca al borrar y al restaurar.** Era el tercero de los menores: sin base, la
única regla que queda es «vive si se cambió después de borrarse», y restaurar no tocaba la fecha, así que
una entrada rescatada en un equipo perdía contra la purga de otro y **se iba otra vez, sin decir nada y con
la papelera del otro ya vacía**. Las dos cosas están en `docs/formato-boveda.md`.

### 11 · El código de alta admite más de cinco intentos en una ráfaga — **media**

`servidor/src/indice.ts`. Leer el código, comprobarlo y apuntar el intento son tres viajes a D1, y
`terminarAlta` es la única ruta que no pasa por el freno por IP. Con peticiones a la vez, los «cinco
intentos» de un código de seis cifras no son cinco. No se reproduce en el simulador; contra D1 de verdad
se espera que sí. **Hecho (2.25.7)**: una sola sentencia atómica
(`UPDATE … WHERE intentos < 5 RETURNING codigo`), que además deja en cinco el contador de la fila en vez de
subirlo con cada petición de la ráfaga, y la ruta pasa por el freno por IP —el de entrar, 20 por minuto, que
es el de comprobar un secreto; el de altas ya lo gastó pedir el código—. Dos pruebas: veinte intentos a la
vez gastan cinco, y veinticinco seguidos desde una IP acaban en 429.

**Lo que queda del 2 y del 3**: el estado de la extensión no se marca como «hay que subir» cuando la fusión
de la entrada deja cambios sin subir (hallazgo menor 4 de esa pasada); se autocura en la siguiente pasada,
y está apuntado en `docs/deuda.md`.

### Y los menores que también se arreglaron (2.25.7)

- **El sello no cubría `creado` ni `codificacion` de cada sobre**, y `creado` decide qué ranura gana al
  fundir sin base: un servidor podía envejecer una ranura sin tocar su contenedor y colarle a un equipo
  rezagado una contraseña maestra vieja. Ahora el sello lleva un campo `sobres` con la huella del sobre
  entero. **No rompe nada de antes**: si no está, se comprueba lo de siempre, y nadie puede quitarlo, porque
  el sello va cifrado con la clave de bóveda. En las dos implementaciones, con las pruebas cruzadas
  vigilando que calculen la misma huella.
- **`Fundir` sustituía el contenido en memoria antes de saber si el guardado salía bien.** Si guardar
  fallaba —otro Esfinge tocando el fichero, o el disco—, la ventana se quedaba enseñando una bóveda que no
  estaba en ninguna parte. Ahora se deshace.
- **Parámetros de Argon2id hostiles**: el tope al abrir un sobre baja de 1 GiB a 256 MiB, cuatro veces el
  perfil de siempre. Un giga no tumba un ordenador, pero sí al trabajador de fondo del navegador, que desde
  la E2 abre sobres que **ha elegido otro**. En los cuatro sitios que lo declaran: los dos ESF1, los dos
  costes de cuenta y el servidor, que ya no acepta registrar un coste que sus propios clientes rechazarían.
- **Lo demás queda apuntado**, no arreglado: la bifurcación de versiones y los plazos que dependen del reloj
  de cada equipo van a `docs/seguridad.md` y a `docs/deuda.md`; el Worker de pruebas público y el nombre del
  equipo dentro del correo, a `docs/deuda.md`.

## Decisiones que no son un arreglo, sino una elección

### 9 · Rotar la clave de recuperación no invalida las copias antiguas — **media**

La clave de bóveda **no cambia nunca**: cambiar la contraseña maestra o rotar la de recuperación solo la
vuelve a envolver. Así que quien tenga la clave de recuperación vieja **y cualquier copia anterior del
fichero** saca de ella la clave de bóveda, que sigue siendo la de ahora, y con ella abre la bóveda actual
—incluido lo escrito después de rotar—. Copias hay muchas y las hace el propio programa: `.anterior`,
`.base`, `.antes-de-fundir`, las apartadas, las copias de seguridad del sistema y **las versiones que guarda
el servidor**.

`docs/seguridad.md` dice hoy que rotar «deja la anterior inservible», y eso solo es cierto frente al fichero
actual. Hay dos salidas:

- **Cambiar el texto** y decir la verdad: rotar cierra la puerta para el fichero de ahora, y ante una clave
  de recuperación comprometida lo que protege es **crear una bóveda nueva**. Barato y honesto.
- **Recifrar de verdad**: generar una clave de bóveda nueva y volver a envolverla en todos los sobres. Cuesta
  milisegundos, pero obliga a que rotar y cambiar la maestra sean una sola operación —para envolver el sobre
  de recuperación hace falta la clave de recuperación— y cambia la prueba de posesión que usa el servidor,
  que hoy se apoya en que esa clave no cambia (ADR 0037).

**Mi recomendación**: cambiar el texto ahora y dejar el recifrado como trabajo aparte, porque toca el
protocolo de cuenta.

## Lo demás, menor

- ~~**El sello no cubre `creado` ni `codificacion` de cada sobre**~~ → **arreglado en la 2.25.7**, y sin
  subir el formato: un campo `sobres` en el sello, que las versiones de antes no traen y que nadie puede
  quitar.
- **La versión de la bóveda es un número, no una cadena**: el servidor no puede fabricar contenido, pero sí
  bifurcar (enseñar a cada equipo su propia rama) o congelar. Se cerraría sellando la huella del documento
  anterior. **Dicho ya en `docs/seguridad.md` y apuntado en `docs/deuda.md`** (2026-09-23); cerrarlo es una
  subida de formato y no se ha hecho.
- ~~**Restaurar y borrar no tocan la fecha de cambio**~~ → **arreglado en la 2.25.7**: las dos la tocan, en
  Go y en TypeScript, con su prueba de restaurar contra una purga fundiendo sin base.
- **Los plazos de la papelera y de las lápidas los decide el reloj del equipo que abre**: uno muy adelantado
  purga para toda la cuenta. **Dicho en `docs/seguridad.md` y en `docs/deuda.md`**; arreglarlo pide una hora
  de referencia que hoy no hay.
- ~~**`Fundir` sustituye el contenido en memoria antes de saber si el guardado sale bien.**~~ →
  **arreglado en la 2.25.7**: si guardar falla, en memoria se queda lo que había.
- ~~**Parámetros de Argon2id hostiles en los sobres**~~ → **arreglado en la 2.25.7**: el tope baja a
  256 MiB en los dos ESF1, en los dos costes de cuenta y en el servidor.
- **El Worker de pruebas es público y su buzón no pide nada**: cualquiera lee los códigos de esas cuentas.
  Conviene ponerle Cloudflare Access delante y asegurarse de que sus secretos no son los de producción.
  **Apuntado en `docs/deuda.md`: lo tiene que hacer el cliente en su panel de Cloudflare.**
- **El nombre del equipo lo escribe quien entra y sale dentro del correo** que recibe el dueño: 80
  caracteres a su gusto, sin caracteres de control y en un correo de texto, no de HTML. **Aceptado**, y
  apuntado en `docs/deuda.md`.
- **Resend conserva los cuerpos** de los correos durante su retención, y ahí van los códigos. **Dicho en
  `docs/seguridad.md`.**

## Lo que se miró y está bien

Para que se sepa qué quedó cubierto: el contenedor `ESF1` —nonces, datos autenticados, el modo por
segmentos, truncados y reordenaciones—, la jerarquía de claves, la clave de recuperación, la separación
entre la clave de acceso a la cuenta y la que abre la bóveda, las transacciones del servidor al subir y al
cambiar la contraseña, las sesiones y sus caducidades, el aislamiento del panel y de los guiones de página,
el reparto de origen de cada petición, las reglas de dominio con la lista de sufijos, el bloqueo por
inactividad, qué se guarda en el navegador y qué no, la convergencia de la fusión, el historial que salva
la contraseña perdedora de cada choque, y los permisos declarados en los dos manifiestos.
