# Seguridad

Última actualización: **2026-09-09**

Qué protege Esfinge y qué no. En una herramienta que cifra, lo segundo importa tanto como lo
primero: una expectativa equivocada sobre lo que protege es exactamente lo que hace daño.

## Lo que hace

Coge un secreto y una clave y produce un contenedor que **solo se abre con esa clave**. Quien tenga
el contenedor y no la clave no tiene nada: ni el contenido, ni una versión parcial, ni pistas sobre
su longitud más allá de la evidente.

Y **detecta si el contenedor ha cambiado**. Alterar un byte, cortarlo por la mitad o reordenar sus
partes hace que la apertura falle en vez de devolver algo distinto de lo que se guardó.

## De qué protege

| Amenaza | Mitigación | Riesgo residual |
|---|---|---|
| Alguien intercepta el texto cifrado por correo o chat | XChaCha20-Poly1305; sin la clave no hay contenido | Se ve **que** hay algo cifrado y su tamaño aproximado |
| Alguien roba el fichero `.esf` de un disco | Lo mismo | Igual |
| Alguien modifica el contenedor por el camino | Etiqueta de autenticación por segmento, con la cabecera autenticada | Ninguno conocido: cualquier cambio rompe la apertura |
| Alguien corta el fichero y lo pasa por entero | Marca en el último segmento ([ADR 0003](adr/0003-marca-de-final.md)) | Cortar dentro del **primer** segmento no se distingue de una clave equivocada |
| Fuerza bruta sobre la clave | Argon2id con 64 MiB y 3 pasadas: cada intento cuesta medio segundo y mucha memoria | Una clave corta sigue siendo una clave corta. El medidor avisa |
| Dos mensajes iguales delatan que lo son | Sal y nonce nuevos en cada operación | Ninguno |

## De qué NO protege

Esto es lo importante de este documento.

- **De quien ya está dentro de la máquina.** Mientras la ventana está abierta, la clave vive en
  memoria. Quien pueda leer la memoria del proceso, poner un registrador de teclas o hacerse pasar
  por el usuario, no necesita romper nada.
- **De perder la clave de un `.esf`.** No hay recuperación, ni puerta trasera, ni copia en ninguna
  parte. Si se pierde la clave, el contenido se ha perdido. Esto no es un fallo: es lo que significa
  cifrar.

  **La bóveda es la excepción, y es deliberada.** Ahí sí hay una segunda llave —la clave de
  recuperación— porque perder la contraseña maestra no puede significar perder *todas* las
  contraseñas de golpe. Eso trae su propia contrapartida, y va abajo.
- **De que se sepa qué has cifrado.** El historial guarda nombres de fichero y fechas
  ([ADR 0010](adr/0010-que-guarda-el-historial.md)). No guarda contenidos ni claves, pero saber que
  el martes cifraste `credenciales-banco.env` ya dice algo. Se puede vaciar desde la propia ventana.
- **De un canal inseguro para la clave.** Si el contenedor y la clave viajan por el mismo correo,
  quien lea ese correo lo tiene todo. Esto no lo puede resolver el programa.
- **De un fichero manipulado antes de cifrarlo.** Esfinge sella lo que le den; no sabe si lo que le
  dieron era lo que debía.
- **Del portapapeles.** Al cifrar, el resultado se copia solo ([ADR 0011](adr/0011-copiado-automatico.md)).
  Cualquier programa que vigile el portapapeles lo verá. Sin la clave no le sirve, pero conviene
  saberlo. Al descifrar no se copia nada por su cuenta, justamente por esto.

  **Y desde la 2.8.0, «Usar como clave» copia también la contraseña generada**, que sí es un secreto
  aprovechable por sí mismo. Va en la misma línea que lo anterior —se copia para poder pegarla en un
  gestor de contraseñas sin dar un rodeo— pero la diferencia importa: lo que queda en el portapapeles
  es la llave, no el candado.

  **Desde la 2.12.0 el portapapeles se borra solo** pasado el plazo que diga Ajustes, y eso arregla ese
  agujero. Con dos límites que conviene saber: solo se borra **si sigue conteniendo lo que Esfinge
  puso** —nunca se pisa lo que se haya copiado después—, y un gestor de portapapeles del sistema, o
  el Portapapeles Universal de Apple, ya se lo pueden haber llevado a otro sitio. Eso no lo puede
  borrar nadie.

- **De un equipo perdido, olvidarlo protege a medias.** Con cuenta, «Olvidar» en Ajustes le quita la
  sesión en el servidor, y desde la 2.24.5 **le cierra la bóveda** si la tenía abierta —en menos de un
  minuto, al intentar sincronizar—, así que quien lo tenga delante necesita la contraseña maestra. Pero
  **solo si está encendido y con conexión**, y **no borra nada**: la copia de ese equipo sigue en su disco
  y se abre con la contraseña de siempre. Si crees que alguien la sabe, cámbiala: el equipo perdido deja
  de poder ponerse al día, aunque lo que ya tenía sigue ahí.

## Lo que cambia con la bóveda

La bóveda guarda contraseñas, así que **cambia el modelo de amenazas del propio programa**. Antes,
un fallo perdía un fichero; ahora puede perderlas todas. Lo que hay que tener claro:

- **La clave de recuperación es una segunda puerta a todo.** Quien la consiga tiene la bóveda
  entera, y no caduca: hasta que no se rote —lo que genera una clave nueva y deja la anterior sin
  poder abrir **el fichero de ahora**— sigue abriendo. Y hay que decir lo que rotar **no** hace
  (revisión del 2026-09-23): la clave que cifra la bóveda por dentro no cambia nunca, así que quien
  tenga la clave de recuperación vieja **y una copia anterior del fichero** —la `.anterior` que deja
  cada guardado, una copia de seguridad, o una versión guardada en el servidor— saca de ahí esa clave
  y con ella abre la bóveda actual. Si una clave de recuperación se ha perdido de verdad, lo que
  protege no es rotarla: es **crear una bóveda nueva** y llevarse las entradas. Se enseña **una sola vez** al crear la bóveda, no se guarda en ninguna
  parte, y no se puede volver a ver. Guardarla es tan importante como guardar la maestra, y en otro
  sitio distinto.
- **El historial de contraseñas conserva las anteriores.** Cambiar una contraseña no borra la vieja:
  se guarda para el caso de «cambié la contraseña y el servicio no se enteró». Eso significa que un
  secreto sustituido **sigue dentro de la bóveda** hasta que se borre a mano.
- **Y lo borrado también, durante treinta días.** La papelera guarda la entrada entera, con su
  contraseña, para que un clic mal dado no la pierda para siempre
  ([ADR 0026](adr/0026-la-papelera.md)). Se vacía a mano cuando se quiera, y sola a los treinta días
  de haber borrado cada entrada. Es el mismo trato que el historial de arriba: contra quien no tiene
  la maestra no cambia nada, y contra quien la tiene es una entrada más de las que ya había.
- **La bóveda está abierta durante horas**, y mientras lo está, todas las contraseñas viven
  descifradas en memoria. Un volcado de memoria o el fichero de intercambio pueden contenerlas. Es
  la misma limitación de siempre —Go no permite borrar una cadena— pero aquí la ventana es mucho más
  larga. Se acorta con el bloqueo por inactividad, que va a quince minutos por defecto, y
  mandando a la ventana **una contraseña cada vez**, solo cuando se pide, en vez de la lista entera.
- **El segundo factor está al lado de la contraseña, y eso le quita parte de su gracia.** Desde la
  2.15.0 la bóveda calcula los códigos de un solo uso, así que **una bóveda abierta entrega la
  contraseña y el código a la vez** y quien tenga la contraseña maestra los tiene los dos. Contra
  ese atacante concreto el segundo factor deja de ser un segundo factor. Contra el que se inventó
  —una contraseña filtrada en la brecha de un servicio, que es lo que pasa el 99 % de las veces—
  sigue valiendo entero. Se hace así porque la alternativa realista no era tenerlos separados: era
  tenerlos juntos en Dashlane ([ADR 0025](adr/0025-los-codigos-de-un-solo-uso.md)). Quien quiera de
  verdad dos factores separados tiene que dejar la semilla fuera de aquí, en un teléfono o en una
  llave física, y eso Esfinge no lo puede decidir por nadie.
- **Los iconos van cifrados aunque un icono sea público.** Lo que hay que ocultar no es el dibujo: es
  **la lista de sitios**. Una carpeta con `banco.es.png` y `hacienda.es.png` diría qué hay dentro de
  la bóveda, y hashear los nombres no salvaría nada.
- **El fichero de la bóveda es JSON en claro por fuera.** Lo cifrado son los campos. Quien pueda
  escribirlo no puede leer nada ni fabricar una bóveda que abra, pero sí puede estropearla o
  revertirla a una copia vieja. **Se detecta al abrir** —hay un sello por dentro que no cuadraría—
  pero no se impide.

## Y desde la fase 2, una puerta hacia dentro

La extensión del navegador consulta la bóveda por un **canal local** (ADR 0027). Es distinto de las
otras dos cosas que Esfinge hace fuera de sí misma: aquéllas **salen** a la red y ésta **abre una
puerta** a este ordenador. Por eso viene apagada y se enciende en Ajustes.

Lo que hay que saber, dicho sin adornos:

- **No es la red.** Es un socket de dominio unix en tu carpeta, no un puerto: no se alcanza desde
  otra máquina, ni desde otra sesión, ni desde una página web que pruebe direcciones locales. Lo
  alcanza quien pueda abrir ese fichero, que es tu propio usuario.
- **Baja el listón dentro de «quien ya está en la máquina».** Eso ya estaba arriba como riesgo, pero
  cambia de tamaño: hoy, sacarle secretos a una bóveda abierta exige leer la memoria de otro
  proceso; con el canal encendido, basta con conectarse a un socket. Contra el mismo atacante, menos
  trabajo.
- **Por eso hay que dar permiso**, una vez, en la ventana de Esfinge, y se puede retirar. No protege
  de un programa decidido —el testigo está en un fichero que ese programa puede leer— pero sí
  convierte «cualquier cosa instalada te vacía la bóveda en silencio» en «tiene que pasar por un
  aviso que no esperabas». 1Password resuelve esto comprobando la firma del navegador; aquí no se
  puede, porque **Esfinge no está firmada**.
- **El navegador nunca ve la contraseña maestra.** Con la bóveda cerrada, el canal contesta
  «cerrada» y **no trae la ventana al frente**: si lo hiciera, cualquier programa podría hacer
  aparecer el diálogo de la maestra cuando quisiera, que es la forma de enseñarte a teclearla.
- **Y por el canal sale una contraseña cada vez, siempre con el sitio delante.** Una entrada no se
  entrega a un sitio que no sea el suyo, y el emparejamiento de sitios se hace por dominio
  registrable, nunca comparando cadenas.
- **En Windows, los permisos del socket no significan nada**; ahí lo que lo protege es que vive en
  tu perfil de usuario.
- **Y en Windows, Esfinge escribe en tu registro** (ADR 0034): al encender el canal, una clave por
  navegador bajo `HKCU` que apunta al manifiesto, y al apagarlo las borra. Es el equivalente de los
  ficheros de macOS y Linux, con el mismo punto flojo: **cualquier programa con tus permisos puede
  cambiar a qué apunta**.

### Y desde la entrega 2, Esfinge escribe en las páginas

Rellenar un formulario obliga a dos cosas que hasta ahora no hacían falta (ADR 0028). Las dos
cambian lo que hay que saber, así que van dichas enteras y no de pasada:

- **Hay un guion de Esfinge en cada página `https` que abres.** No dibuja nada, no guarda nada y no
  habla con la página: lee el formulario, escribe en él y calla. Pero está ahí, y un fallo suyo
  sería un fallo en las páginas de otros. Se apaga apagando el canal en Ajustes.
- **Y por el canal sale una contraseña de verdad**, que hasta la entrega 1 no pasaba: entonces
  copiaba Esfinge y por el socket solo volvía cuánto tardaría en borrarse el portapapeles. Para
  escribir una contraseña en un formulario hay que tenerla. **Lo que sale por ahí no se borra
  solo**, porque no pasa por el portapapeles: que la extensión lo escriba en el campo y lo olvide es
  una promesa suya, y Esfinge no puede comprobarla.
- **Con una sola cuenta guardada del sitio, se rellena sin que pulses nada.** Es lo que se pidió, y
  el precio es éste: cualquier guion que ya esté corriendo en esa página puede leer del formulario
  lo que Esfinge acaba de escribir. Para eso hace falta que alguien ya ejecute código en ese
  dominio —donde también podría falsificarte el formulario entero—, así que es un escalón menos, no
  una puerta nueva. Con **varias**, desde la 2.21.1, se rellena sola **la que coincida con el usuario que va a
  entrar** —el escrito en el formulario, el escondido que declara el sitio o el de la página anterior—;
  si no se sabe o no coincide ninguna, se elige en el panel. Eso quiere decir que **una página de ese
  sitio puede elegir cuál de tus cuentas de ese sitio se rellena**, poniendo un usuario: es el mismo
  escalón, dentro del mismo dominio.
- **Nunca en un marco de otro origen, y nunca sobre `http://`.** Un `iframe` ajeno puede ser
  cualquiera y la dirección que ves arriba no es la suya. Y sobre texto claro no se rellena, lo que
  deja fuera la página de administración de un router: es una carencia conocida, dicha aquí en vez
  de resuelta a medias.
- **Ante la duda, no se rellena.** No se toca un formulario con dos contraseñas visibles —eso es
  registrarse—, ni un campo declarado como contraseña nueva o código de un solo uso, ni uno
  invisible o de un píxel. Equivocarse de campo es escribir una contraseña donde la lea alguien.
- **Y desde la 2.20.0, Esfinge deja constancia en la página** (ADR 0031): un filete en el campo
  rellenado y, en los rellenos automáticos, un aviso de tres segundos, «Rellenado por Esfinge». El
  aviso no se puede pulsar, va en una sombra cerrada y no lleva nada de la web ni de la bóveda. Pero
  es un elemento nuestro en la página de otro, y con él **una web puede saber con seguridad que usas
  Esfinge**. El icono de la barra, además, **enseña cuántas cuentas tienes de la web que miras** a
  quien mire tu pantalla.
- **Y desde la 2.19.0, también el segundo factor** (ADR 0030). Cuando un sitio pide el código de un
  solo uso y tienes una sola cuenta de ese sitio con código, Esfinge lo escribe. Eso quiere decir que
  **por el canal salen la contraseña y el código**, y que **en la página quedan escritos los dos**:
  lo que la ADR 0025 dijo de la bóveda abierta —que entrega las dos cosas a la vez— vale ahora dentro
  del navegador. El código gasta del mismo freno que la contraseña, así que no hay más margen por
  pedir las dos, y **no se escribe en cualquier campo que se llame «código»**: ni el postal, ni el
  promocional, ni el de la tarjeta.

### Y desde la entrega 3, el navegador escribe en la bóveda

Hasta aquí el navegador solo leía. Ofrecer guardar lo que se envía en un formulario cambia eso, y
cambia también lo que se dibuja en la página (ADR 0032):

- **Quien pueda hablar por el canal puede escribir en la bóveda**: crear cuentas, cambiar contraseñas
  y apuntar sitios en los que no ofrecer. Hace falta el mismo permiso que para leer, y lleva **su propio
  freno, seis escrituras por minuto**. Una contraseña cambiada **va al historial de anteriores**, así que
  un cambio no deseado se deshace desde la ventana. No se confirma en la ventana: lo eligió el cliente.
- **La contraseña que escribes pasa un momento por la memoria de la extensión.** Hace falta porque al
  pulsar «Entrar» la página cambia. Se queda en el trabajador de fondo, **nunca en disco**, como mucho
  dos minutos, y **no llega a la página siguiente**: la tarjeta que pregunta no la tiene.
- **Hay un elemento de Esfinge que se puede pulsar en la página de otro**: la tarjeta «¿Guardar en
  Esfinge?». Va en una sombra cerrada y solo hace caso a clics de una persona. Lo peor que consigue una
  web que engañe para pulsar es guardar **lo que escribiste en esa misma web, para esa misma web**:
  lo que se guarda va siempre para el sitio del que salió.
- **Ante la duda, no se ofrece.** Si el formulario vuelve a salir tras enviarlo, la contraseña se da por
  mala y se olvida; y un formulario que no se sabe si es de entrar, de registrarse o de cambiar, no se
  ofrece.
- **Y desde la 2.22.0, nada antes del aviso de datos** (ADR 0033). La extensión recién instalada no
  lee ninguna página ni habla con Esfinge hasta que se abre su panel y se acepta «Esfinge y tus datos»,
  que dice qué lee, que va solo a Esfinge en este ordenador y qué guarda. Lo exige la tienda de Chrome y
  se hace cumplir en el trabajador de fondo y en el guion de la página, no solo en el panel. En Firefox,
  además, el propio navegador enseña al instalar lo declarado en `data_collection_permissions`:
  credenciales, datos que identifican —el correo— y la dirección de las páginas.
- **La lista de «Nunca en este sitio» va cifrada dentro de la bóveda**, porque es una lista de sitios
  que usas. Por eso Ajustes solo la enseña con la bóveda abierta.

### Y desde la E2, la extensión puede tener la bóveda dentro

Con cuenta, la extensión deja de pedirle nada a la aplicación: **guarda la bóveda en el navegador, la
abre con la contraseña maestra y la sincroniza ella sola** con el servidor de cuentas (ADR 0040). Es la
parte más expuesta de todo el producto, y conviene decir por qué y hasta dónde:

- **Mientras está abierta, su clave vive en la memoria del navegador** (`storage.session`), no en disco.
  Se va al pulsar «Bloquear», a los quince minutos sin usarla y al cerrar el navegador. Los guiones que la
  extensión pone en las páginas no pueden leerla. Pero **quien controle el navegador —otra extensión con
  permisos de más, un fallo del navegador, alguien con el equipo desbloqueado— llega ahí**, y no a la
  aplicación, que es otro programa. Contra eso no hay arreglo desde dentro de una extensión; lo que se
  puede hacer es tenerla abierta poco, y eso es lo que hace el bloqueo.
- **En el disco del navegador** (`storage.local`) quedan la bóveda **cifrada**, su última versión común
  con el servidor —cifrada también—, el correo, la sesión **sellada con la clave de la bóveda** y el testigo
  de «este navegador es de confianza» **en claro**, igual que en la aplicación: le ahorra el código por
  correo a quien copie el perfil del navegador **y sepa la contraseña**, que con ese perfil ya abre la
  bóveda sin hablar con nadie.
- **Copiar desde el panel no se borra solo del portapapeles.** La aplicación copia ella y lo borra pasado
  el plazo; con cuenta, la bóveda está en el trabajador de fondo, que no puede tocar el portapapeles, así
  que copia el panel y el panel se cierra enseguida. El panel lo dice al copiar.
- **Una contraseña mal escrita al desbloquear llega al servidor**: si no abre la copia del navegador,
  puede ser la nueva —cambiada en otro equipo—, y se prueba con la cuenta. Es un intento fallido más en
  el servidor; los equipos de confianza no quedan fuera por los fallos.
- **Se conecta a un solo sitio**, el servidor de cuentas, y solo con cuenta. Lo que le llega es lo mismo
  que desde la aplicación —el correo, la bóveda cifrada, el nombre del equipo, que aquí es «Chrome en Mac»
  o similar— y se revoca igual desde Ajustes de la aplicación. **Un 401 cierra la bóveda**, como en la
  aplicación desde la 2.24.5.
- **El WebAssembly de Argon2id va dentro del paquete** (`hash-wasm`), como el cifrado (`@noble/ciphers`):
  la extensión no descarga código. Las dos bibliotecas van a versión exacta, y lo que hacen se comprueba
  contra los vectores fijos del formato y **cruzado con Go** en cada publicación.

## Dónde queda algo en disco

| Qué | Dónde | Permisos |
|---|---|---|
| Historial | Carpeta de configuración del usuario, `Esfinge/historial.json` | `600` |
| **La bóveda** | La misma carpeta, `Esfinge/boveda.esfinge`, más `.anterior` con la copia previa | `600` |
| Los iconos de los sitios | `Esfinge/boveda.esfinge.iconos`, **cifrado con la clave de la bóveda** | `600` |
| Preferencias | La misma carpeta, `Esfinge/preferencias.json` | `600` |
| **La cuenta**, si hay | `Esfinge/cuenta.json`: el correo, el nombre del equipo y **la sesión sellada con la clave de la bóveda** y el testigo de confianza del equipo **en claro**, para que la bóveda se abra con la contraseña nueva cuando se cambió en otro equipo (ADR 0037): le ahorra el código por correo a quien copie el fichero **y sepa la contraseña**, que con ese disco ya abre `boveda.esfinge` sin hablar con nadie | `600` |
| La base de la sincronización, si hay cuenta | `Esfinge/boveda.esfinge.base` —**una copia entera de la bóveda**, cifrada igual— y `.sincro`, qué versión es | `600` |
| La bóveda de antes de una fusión que borró algo | `Esfinge/boveda.esfinge.antes-de-fundir` | `600` |
| La bóveda que había al entrar en una cuenta | `Esfinge/boveda.esfinge.apartada-<fecha>`: **no se borra nunca sola** | `600` |
| La actualización descargada | Carpeta de caché del usuario, `Esfinge/descargas/` | `600` |
| Ficheros cifrados | Junto al original, con `.esf` al final | `600` |
| Lo que se guarda desde la ventana | Donde diga el diálogo del sistema | `600` |

## Lo único que sale de la máquina

Desde la 2.14.0 son **dos**, y **tres si se crea una cuenta** (la tercera, más abajo). Conviene saber
exactamente cuáles. Y la extensión, con cuenta, hace la tercera desde el navegador, sin la aplicación
(ver «la extensión puede tener la bóveda dentro», arriba).

**1 · Una petición `GET` a `api.github.com`, una vez al día**, para preguntar cuál es la última
versión publicada ([ADR 0014](adr/0014-comprobacion-de-actualizaciones.md)). En ella viaja el número
de versión instalada, dentro del `User-Agent`, que es lo que se compara; y GitHub ve la dirección IP,
como cualquier página que se visite.

**2 · El icono de cada sitio de la bóveda, pedido al propio sitio**
([ADR 0024](adr/0024-iconos-de-los-sitios.md)). Se piden poco a poco, espaciados y en orden
aleatorio, unos pocos por sesión. **Nunca a un intermediario**: un servicio de iconos recibiría la
lista completa de sitios donde tienes cuenta.

Y aquí hay que decir algo que no es evidente: **ir directo no oculta esa lista, la reparte**. El
nombre del sitio viaja en claro en la consulta de DNS y en el saludo TLS, antes de que empiece el
cifrado, así que **quien pueda mirar tu red ve a qué sitios se pregunta**. Lo que se gana yendo
directo es no meter a una empresa de por medio, que es otra cosa. Al sitio no se le dice quién
pregunta: la petición no lleva el nombre ni la versión de Esfinge.

Viene **encendido**, se avisa la primera vez y se apaga en Ajustes. Sin él, cada entrada sale con un
cuadro de color y su inicial, que no sale de esta máquina.

**No hay telemetría, ni informes de fallos, ni identificadores.** Nada de lo que se cifra, ni los
nombres de los ficheros, ni cuántas veces se usa el programa, ni nada que permita distinguir una
instalación de otra.

Las dos se apagan en **Ajustes**, donde está dicho con estas mismas palabras.

**Y `ESFINGE_SIN_RED=1` las apaga todas, sin excepción.** Hasta la 2.14.0 esa variable solo la miraba
la línea de comandos —la ventana no la consultaba nunca—, así que quien la ponía creyendo que apagaba
la red apagaba la mitad. Ahora vive en un sitio y la consultan las dos salidas, con una prueba por
cada una. En la línea de comandos, además, no se pregunta nunca si la salida de error no es un
terminal, que es el caso de cualquier script.

**3 · Con cuenta, la bóveda cifrada, al servidor de cuentas de Webcafeína**
([ADR 0035](adr/0035-las-cuentas.md) a [0038](adr/0038-sincronizar-la-boveda.md)). Solo si se elige
«Con cuenta» en la bienvenida o en Ajustes; sin cuenta, esta salida no existe. Va a
`esfinge-cuentas.webcafeina.com`, un servidor en Cloudflare con los datos guardados **en la UE**, y
también la apaga `ESFINGE_SIN_RED`.

**Qué no le llega nunca**: la contraseña maestra, la clave de la bóveda, ni ninguna entrada en claro. La
bóveda sube **tal como está en el disco**, cifrada, y el servidor no tiene con qué abrirla. De la maestra
se deriva aparte una clave de acceso, y el servidor guarda un HMAC de ella.

**Qué sí ve**, y hay que decirlo:

- **El correo** de la cuenta, y cuándo se creó.
- **La dirección IP** de cada petición, que la ve Cloudflare. El servidor no la guarda en claro: los
  frenos por IP llevan un HMAC de ella, y se tiran a los dos días.
- **Cuánto ocupa la bóveda y cuándo cambia**, con las diez últimas versiones y una por día del último
  mes, para poder deshacer una fusión mala.
- **Qué equipos hay en la cuenta**: el nombre de cada ordenador y cuándo se usó por última vez.
- **Cuándo se entra, se cambia la contraseña o se recupera la cuenta**: los doscientos últimos eventos.

**Los códigos llegan por correo** y los manda Resend, que ve la dirección y el código. Un correo no es un
segundo factor tan fuerte como una aplicación de códigos: quien entre en el buzón y sepa la contraseña,
entra. Se eligió así con el cliente.

**Lo que no protege, dicho tal cual:**

- **Quien robe el servidor puede probar contraseñas contra la bóveda cifrada** sin conexión, al coste de
  Argon2id, igual que quien robe hoy el fichero del disco. La defensa es la contraseña: por eso, con
  cuenta, tiene que ser al menos «Buena».
- **Un servidor malicioso puede enseñar a un equipo una versión vieja de la bóveda** y no la última. Lo
  que no puede es hacerla pasar por nueva —la versión va sellada dentro, con la clave de la bóveda— ni
  fabricar una ranura para abrirla.
- **Y puede congelar o bifurcar**: la versión es un número, no una cadena que ate cada documento al
  anterior, así que un servidor podría dejar a un equipo parado en la versión que ya tiene, o servirle a
  cada equipo su propia rama, y ninguno se enteraría de que el otro ha subido nada. **No puede inventarse
  contenido**, porque no tiene con qué cifrarlo. Se cerraría sellando también la huella del documento
  anterior; está apuntado en `deuda.md`.
- **Los plazos los decide el reloj del equipo que abre la bóveda.** La papelera de treinta días y las
  lápidas de ciento ochenta se purgan al abrir, y un equipo muy adelantado purga para toda la cuenta. No
  hace falta un servidor malicioso: basta con un ordenador con la hora mal puesta.
- **Resend guarda lo que manda** mientras dure su retención, y en esos correos van los códigos de entrada
  y de borrado. Caducan en diez minutos, pero quien pueda leer esa consola los ve.
- **El servidor sabe quién tiene cuenta.** La pre-entrada contesta igual con cuenta que sin ella, así que
  preguntar desde fuera no lo desvela; al servidor, sí.

Si se descarga una actualización, se comprueba su SHA256 contra el publicado. **Eso protege de una
descarga rota, no de una publicación manipulada**: el resumen sale del mismo sitio que el fichero. Lo
que sostiene la confianza es el TLS contra GitHub, y que la aplicación no se instala sola —el
instalador lo abre quien esté delante—.

## Decisiones que afectan a la seguridad

- [ADR 0002](adr/0002-formato-esf1.md) — El cifrado y por qué esos algoritmos.
- [ADR 0003](adr/0003-marca-de-final.md) — Por qué un fichero cortado se detecta.
- [ADR 0004](adr/0004-contrasenas-en-hexadecimal.md) — Por qué las contraseñas salen en hexadecimal.
- [ADR 0010](adr/0010-que-guarda-el-historial.md) — Qué se guarda y qué no.
- [ADR 0012](adr/0012-sin-firmar.md) — Por qué el sistema avisa al instalarla.

## Lo que no se ha auditado

Nadie de fuera ha revisado esto. **Tampoco el servidor de cuentas**, y por eso el registro es por
invitación hasta que lo revise alguien de fuera (ADR 0035). El núcleo tiene pruebas que cubren la ida y vuelta, la manipulación
de cada byte, el truncado y la reordenación, y usa implementaciones de la biblioteca estándar
extendida de Go —`golang.org/x/crypto`— en vez de nada escrito aquí. Pero **una batería de pruebas
propia no es una auditoría**, y conviene decirlo antes de que alguien confíe más de la cuenta.
