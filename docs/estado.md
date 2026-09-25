# Estado

Última actualización: **2026-09-25**

## Dónde se paró, y por dónde se sigue

**2026-09-25. Publicada la 2.27.0**, con los siete trabajos en verde y las tiendas incluidas: la C1 y la
C2 —desbloquear la bóveda con Touch ID—. **Y ahí se para hasta que el cliente la pruebe en su Mac.** Se
eligió así a conciencia: la C2 es cgo que esta máquina no compila, sabemos que compila pero **no que
arranque**, y escribir la C3 encima de un diseño sin verificar significa rehacer las dos si falla.

- **Publicada la 2.26.0** (24-09): la B entera —compartir copias, invitaciones a quien no tiene cuenta— con
  los correos ya maquetados.
- **La fase C**, desbloquear con el sistema ([ADR 0044](adr/0044-desbloquear-con-el-sistema.md), estudio en
  [`desbloqueo-del-sistema.md`](desbloqueo-del-sistema.md)). **C1 y C2 hechas**: la ranura local con sus
  dos pantallas, y Touch ID por cgo con el secreto en el llavero de inicio de sesión.
- **Lo siguiente es la C3**, Windows Hello, a ciegas: nadie ha ejecutado nunca Esfinge en un Windows. Antes
  de empezarla, releer la ADR 0044 — la credencial de Hello en un Win32 sin empaquetar **está atada a la
  cuenta de usuario y no a la aplicación**, así que ahí el cerrojo es todavía más cerrojo.
- **Y lo que queda del plan de cuentas**, que el cliente hace junto: **probar Firefox** y **comprobar
  compartir entre sus dos Macs**.

### Lo que hay que mirar en el Mac con la 2.27.0

**Lo primero no es retórico: que arranque.** La 2.9.1 salió con un diagnóstico de Objective-C que compilaba
en verde y cerraba la aplicación nada más abrirla. Si no arranca, se para aquí y se revierte.

1. **Instalar y abrir.** Si arranca, lo demás son detalles; si no, no hay nada más que probar.
2. **Activarlo**: abrir la bóveda con la maestra → Ajustes → «Desbloquear con Touch ID». **Decir si macOS
   pregunta algo** —un diálogo de «Esfinge quiere usar información protegida»— y qué se contestó.
3. **Usarlo**: cerrar la bóveda y volver a la pantalla de desbloquear. Tiene que salir **«Abrir con Touch
   ID»**; pulsarlo, poner el dedo, y abrir.
4. **Cancelar el diálogo del sistema**: tiene que volver al campo de la contraseña **sin error en rojo**,
   porque cancelar no es un fallo.
5. **La maestra sigue abriendo** con Touch ID activado. Es la regla que no se puede romper.
6. **Quitarlo** en Ajustes: el botón desaparece de la pantalla de desbloquear.
7. **Y la que no se puede probar hoy**: cuando salga la versión siguiente, **al abrir tras actualizar**,
   ¿vuelve a pedir permiso el llavero? El binario cambia y Esfinge no está firmada, así que puede. Es la
   decisión 3 del documento de la fase, y la respuesta decide si hace falta decir algo en la pantalla.

## Dónde estamos

Esfinge es una **aplicación de escritorio** con ventana propia, más una línea de comandos que comparte
núcleo y formato. Va por la **2.26.0**, con **las cuentas abiertas**, la extensión publicada en las dos
tiendas —y subiéndose sola a las dos— y **compartir copias** entre cuentas. Funciona de punta a punta:
cifra y descifra textos y ficheros, genera contraseñas, **guarda contraseñas en una bóveda cifrada**,
lleva un historial de qué y cuándo, y se compila sola para macOS, Windows y Linux en GitHub Actions.

Desde la 2.12.0 Esfinge **deja de ser un cifrador sin estado**. La bóveda es la fase 1 de sustituir a
Dashlane, decidida con el cliente y con las fases 2 a 4 —autorrelleno, cuentas y compartir— sin
empezar a propósito: se hace la primera, se usa a diario, y solo entonces se decide si las otras
valen su coste.

La 1.x fue una herramienta de terminal con menús. Esos menús se retiraron: su público —el cliente—
tiene ahora una ventana. La línea de comandos se quedó, que es la que se mete en tuberías y scripts.

## Completado

- **Núcleo criptográfico** (`internal/cripto`): XChaCha20-Poly1305 con Argon2id, formato `ESF1`
  versionado, ficheros por segmentos con marca de final. Con tests que cubren la ida y vuelta, cada
  byte alterado, el truncado y la reordenación de segmentos.
- **Aplicación con ventana** (Wails, React 19 + TypeScript): cifrar y descifrar textos o tandas de
  ficheros, arrastrar y soltar, diálogos del sistema, historial e generador de contraseñas.
- **Actualizaciones dentro de la aplicación**: aviso, descarga comprobada con SHA256 y, en macOS y
  Windows, **reemplazo y reinicio sin que nadie arrastre nada** (ADR 0016). En Linux se le pasa al
  gestor de paquetes, que instala como root. Fue la única conexión que hacía el programa hasta la
  2.14.0; ahora son dos, con la de los iconos (ADR 0024). Las dos están dichas y se apagan en Ajustes,
  y `ESFINGE_SIN_RED` las apaga todas.
- **Menús del sistema en español** en los tres sistemas, construidos a mano porque los roles de Wails
  traen los rótulos en inglés escritos a fuego (ADR 0015). Con atajos ⌘1…⌘6 a las seis pantallas.
- **Apertura de un `.esf`** por doble clic o «Abrir con», que abre la pantalla que le toca según lo
  que lleve dentro: ficheros o texto.
- **Línea de comandos**: intacta desde la 1.5.0, con sus códigos de salida distintos por caso.
- **Color generado desde Go** (`internal/tema`), con el contraste de los dos temas medido en cada
  compilación.
- **Compilación automática** de los tres sistemas, más los seis binarios de la línea de comandos.
- **El vidrio del sistema en la barra lateral** —macOS siempre, Windows 11 con Mica—, con la zona de
  trabajo opaca (ADR 0017). Es donde el sistema pone la vibrancia. En Linux no lo hay, y ahí la
  ventana queda como estaba.
- **Los diálogos recuerdan su carpeta**, una para abrir y otra para guardar.
- **Las tandas se cifran en paralelo**, con tope: veinte ficheros pasaron de 4,42 s a 1,29 s en una
  máquina de cuatro núcleos (ADR 0018).
- **Estructura nativa en los tres sistemas**: barra lateral y contenido en todos, y luego lo de cada
  casa —macOS sin barra de título (ADR 0019), el panel de Fluent en Windows y la cabecera de GNOME en
  Linux (ADR 0020)—.
- **La marca, dentro de la ventana** (ADR 0021): el lockup de Esfinge en la barra lateral, la firma
  `▍ webcafeína` al fondo, la esfinge tenue en el historial vacío y la ficha de producto en Ajustes
  —que es lo que la ADR 0007 prometía y nunca se había construido—. Y el acento de la interfaz pasa a
  ser el oro del tocado en vez del azul del sistema.
- **El formato `ESF1`, congelado con vectores fijos** (ADR 0022): once contenedores grabados una vez
  y no vueltos a generar, más los de la 1.5.0 sellados con el código de entonces. Antes lo único que
  decía congelarlo era un test que se miraba al espejo, y un cambio coherente en los dos sentidos
  habría pasado en verde dejando de abrir lo ya emitido.
- **La bóveda** (ADR 0023): local, cifrada, con clave de recuperación, cuatro clases de entrada,
  historial de contraseñas anteriores, importación desde Dashlane, Bitwarden, 1Password, LastPass y
  Chrome —cada gestor exporta **varios ficheros** y cada uno se reconoce por su forma—, exportación
  en claro para poder salir, bloqueo por inactividad, borrado del portapapeles y borrado de la bóveda
  entera pidiendo la maestra. Con su sección en la ventana, sus dos plazos en Ajustes y
  `esfinge boveda listar|ver|codigo|exportar` en la línea de comandos. **No toca `internal/cripto`**,
  así que los `.esf` y las claves ya emitidos siguen valiendo.
- **Los códigos de un solo uso** (ADR 0025): la bóveda calcula el código de seis cifras a partir de la
  semilla que ya guardaba, con su cuenta atrás en la ventana y con `esfinge boveda codigo` para los
  scripts. `internal/codigos`, sin dependencias, con los vectores de RFC 4226 y RFC 6238 enteros.
  Con eso se va la última cosa que obligaba a tener Dashlane abierto —y entra la consecuencia
  incómoda: **el segundo factor pasa a vivir al lado de la contraseña**, dicho tal cual en
  `docs/seguridad.md`.
- **La papelera** (ADR 0026): borrar deja de ser irreversible. Lo borrado se guarda entero, se
  restaura o se tira del todo una a una, se vacía a mano y **se va solo a los treinta días**. El
  coste está dicho en `docs/seguridad.md`: durante esos días la contraseña borrada sigue dentro del
  fichero, igual que las del historial de contraseñas anteriores.
- **La fase 2, entrega 1** (ADR 0027): la extensión del navegador consulta la
  bóveda por un **canal local** —un socket en tu carpeta, no un puerto—, apagado de fábrica y
  encendido en Ajustes. Enseña las cuentas del sitio de la pestaña y copia lo que se le pida.
  Comprobado en un Mac de verdad, en Firefox y en Chrome.
- **La fase 2, entrega 2** (ADR 0028): **rellena los formularios**. Con una cuenta guardada del
  sitio, sola al cargar la página; con varias, desde el panel. Y **en la página no se dibuja nada**:
  el guion lee el formulario, escribe en él y calla. Dos propiedades de la entrega 1 se rompen a
  conciencia y hay que decirlas: **por el canal ya sale una contraseña de verdad** —para escribirla
  en un campo hay que tenerla, y eso no hereda el borrado del portapapeles— y **hay código nuestro en
  cada página `https`**. A cambio, el verbo nuevo lleva su propio freno, más estrecho que el de
  preguntar, y la detección de campos está escrita en negativo: ante la duda, no se rellena.
  **Comprobado en un Mac, en Chrome y en Firefox** (2.18.1): rellena solo al cargar y acierta
  el formulario en Brevo y en Cloudflare, y el botón del panel escribe tras borrar los campos.
- **La fase 2, entrega 3** (ADR 0032): **ofrece guardar y actualizar** lo que se envía en un
  formulario, con una tarjeta en la página. Publicada en la 2.21.0 y corregida hasta la **2.21.2** con lo que salió de usarla —Google con dos pasos y varias cuentas, y el cambio de contraseña de Brevo—; **comprobada en el Mac**.
- **Pruebas de la interfaz** con Playwright contra el Go de verdad, en tema claro y oscuro, en una
  máquina sin entorno gráfico. Son **84**.

## En curso

**La fase 2 va por la entrega 2.** La 1 —el canal y la extensión que consulta y copia— se comprobó
en un Mac en las dos familias de navegador. La 2 —rellenar— está escrita y probada aquí hasta donde
se puede: **falta verla en sitios de verdad**, que es lo único que ninguna prueba puede decir.

De escribirla salió, sin buscarla, una de la entrega 1: **el freno de preguntas no frenaba nada**. El
contador vivía en la conexión y la extensión abre una conexión por petición, así que el tope de
sesenta por minuto que impedía reconstruir la lista de sitios de la bóveda con un diccionario de
dominios no se alcanzaba jamás. La prueba estaba en verde porque le pasaba un contador hecho a mano a
sesenta llamadas seguidas, que es el caso que no ocurre. Es la misma familia que los iconos.

La 2.15.0 cerró los códigos de un solo uso —comprobados contra Dashlane el mismo
día— y la 2.16.0, la papelera. En medio salió, sin buscarla, una que llevaba desde la 2.12.0:
**borrar una nota segura o una tarjeta dejaba su contenido dentro del fichero**, porque la lista de
campos sensibles estaba escrita en dos sitios y solo uno estaba completo.

**Con la bóveda ya no queda nada de Dashlane por traer al escritorio.** Lo siguiente es la fase 2, y
su puerta sigue siendo de uso y no técnica.

La 2.12.0 salió con la bóveda y de usarla salieron cinco versiones seguidas de
correcciones —2.12.1 a 2.12.5—, todas de cosas que **solo aparecen usando la aplicación en un Mac**:
pegar con ⌘V, copiar de un campo de contraseña, el filtro del diálogo de abrir, el importador que
solo entendía una forma de fichero y una negrita que partía los avisos en columnas. Ninguna se
habría encontrado desde esta máquina. Los tres sistemas tienen ya su estructura. La de macOS está probada en un Mac; la de
GNOME se puede mirar aquí, porque el servidor de desarrollo corre en Linux; la de Windows no la ha
visto nadie.

## Comprobado en un Mac de verdad

- Que la comprobación de versiones funciona: con la última instalada no sale la banda, y «Buscar
  ahora» dice que ya se está al día.
- La descarga de una actualización, con su barra.
- El doble clic en un `.esf`, en los dos momentos —con Esfinge cerrada y con Esfinge abierta— y con
  las dos clases de contenedor: el que lleva un fichero abre la pantalla de ficheros y el que lleva
  un texto abre la de texto, con la línea puesta.
- **Copiar y pegar con los atajos de ⌘**, que era lo que más riesgo tenía: al construir los menús a
  mano se perdieron los selectores nativos y esas acciones pasan por código propio.
- **La estructura de macOS** de la 2.6.x: barra lateral, título sin banda y el resaltado del ratón
  respetando la fila activa.
- **El disco del DMG**, después de tres intentos y de que llegaran capturas de discos de verdad.
- **Usar la contraseña generada como clave**, y que **cambiar de sección ya no borra lo escrito**
  (2.9.0), que era la deuda que dejó abierta la versión anterior.
- **El vidrio, por fin, en la 2.10.0.** «Ahora sí se ve». Y comparado con la barra lateral del Finder
  al lado, **se ve igual**: el desenfoque que parecía excesivo es el que macOS 26 pone en todas las
  barras laterales del sistema. No hay nada que calibrar.
- **La actualización desde dentro, y que Gatekeeper no aparece.** Se ha usado de verdad, varias
  versiones seguidas: se descarga, se reemplaza y se reinicia sola sin preguntar nada. Confirma lo
  que sostenía la ADR 0016: la cuarentena la pone quien descarga, y aquí descarga Go. El aviso de
  programa no identificado es cosa **solo de la primera instalación**.
- **El icono del documento `.esf` en el Finder** (2.10.1): sale, y sin tener que forzar la caché de
  LaunchServices. De mirarlo puesto salió la corrección de la 2.10.2: la placa estaba centrada en 636
  y la hoja tiene su centro en 512, así que caía 124 px baja. Se había bajado a propósito, creyendo
  que macOS pone ahí el emblema de los documentos; no era cierto.
- **La bóveda con datos de verdad, en el Mac** (2026-09-09): una exportación de Dashlane entra
  entera —credenciales, notas seguras, tarjetas y documentos, que son cuatro ficheros— y **la clave
  de recuperación abre la bóveda**, que era lo único que ninguna prueba podía decir. Costó cuatro
  versiones: el filtro del diálogo de abrir en macOS, el importador que solo entendía una forma de
  fichero, las tarjetas que salían todas duplicadas y lo exportado que no volvía a entrar entero.
- **El código de un solo uso es el mismo que el de Dashlane** (2026-09-10, 2.15.0), con los dos
  programas abiertos uno al lado del otro. Es la comprobación que ninguna prueba de aquí podía hacer,
  y cierra las dos mitades a la vez: que **la semilla se importó bien** —que era lo que de verdad
  estaba en duda, porque los vectores del RFC solo dicen que el algoritmo está bien— y que el
  cálculo concuerda con el de un gestor que lleva años en producción.
- **El canal con el navegador, de punta a punta** (2026-09-10, 2.17.6): con Firefox y un sitio de
  verdad, la extensión reconoce el sitio, enseña la cuenta guardada y **copia la contraseña al
  portapapeles**. Comprobado además lo que hace que eso sea aceptable: **el portapapeles se borra
  solo** pasado el plazo de Ajustes, igual que copiando desde la ventana. Y el **código de un solo
  uso** también, contra un sitio con segundo factor de verdad. **Y en Chrome igual** (2.17.7), con el
  identificador fijado por la clave pública del manifiesto: las dos familias de navegador funcionan.
  Era lo único que no se podía ejercitar aquí —no hay navegador con el que probar
  `connectNative`— y por eso costó **seis versiones**: el puente, el socket, el emparejamiento de
  dominios y los manifiestos estaban bien desde el principio, y lo que faltaba era **una palabra en
  una lista**, el permiso `storage` del manifiesto de la extensión. Sin él esa API no da error: es
  `undefined`, y el fallo se veía como si el puente no contestara.
- **La banda de versión nueva sale sola** (2026-09-09), con la ventana abierta y sin tocar nada. Es
  la prueba buena de la 2.10.4: antes la comprobación ocurría **solo al arrancar** y esa banda no
  había aparecido nunca, aunque la portada llevara desde la 2.1.0 prometiendo «una vez al día».
- Y de preguntar si ese icono valía para los otros dos sistemas salió la 2.10.3: en **Windows** sí,
  del mismo PNG y sin tocar nada, pero en **Linux** no llegaba y seguía saliendo el de la aplicación.
  El `.deb` instala ahora `application-x-esfinge.png` en `hicolor/<tamaño>/mimetypes/`, verificado
  aquí con `dpkg -c`. Verlo puesto en un escritorio de verdad sigue pendiente.

## Siguiente acción concreta

**Lo decidió el cliente al cerrar la sesión del 10 de septiembre, y va en este orden:**

### ~~1. Las mejoras visuales de la extensión~~ — hechas el 2026-09-14, usadas en el Mac

Resuelto en la [ADR 0029](adr/0029-el-panel-de-la-extension-es-esfinge.md): **el panel es la misma
Esfinge en otro sitio**. Importa los tokens de la ventana tal cual —ni una escala ni una pareja de
color nueva, cada combinación es una de las que ya mide `contraste_test.go`—, la marca va solo en la
cabecera como el lockup de la barra lateral, y el oro solo en «Rellenar». Una forma de fila para todas
las cuentas, con el correo entero siempre que quepa; copiar pasa a iconos con nombre; los estados
tienen título, instrucción y, aparte, lo que dijo el navegador. La extensión estrena **iconos**, que no
tenía, y el protocolo dice **si una cuenta tiene código** para no enseñar el botón donde no lo hay.

Se miró en capturas de cada estado en los dos temas (`pnpm run capturas` en `navegador/`) y hay seis
pruebas del panel compilado. **Lo que falta es verlo en un navegador de verdad en el Mac**: allí la
letra es San Francisco y los anchos cambian, así que lo primero es si el correo sigue cabiendo; y si el
icono de 16 px se lee en la barra.

### ~~2. Rellenar el código de un solo uso~~ — hecho y comprobado en el Mac el 2026-09-14

Resuelto en la [ADR 0030](adr/0030-rellenar-el-codigo-de-un-solo-uso.md). Un verbo nuevo,
`rellenar-codigo`, con las mismas llaves que `rellenar` y **gastando de su mismo freno**. Se escribe en
el campo que el sitio declara, en **seis u ocho casillas de un carácter**, o en un campo con nombre de
segundo factor si no hay contraseña en la página; **nunca en uno que solo se llame «code»**. Solo con
una cuenta del sitio que tenga código, esperando al siguiente si al actual le quedan menos de tres
segundos. El botón «Rellenar» del panel hace formulario y código a la vez.

**Probado en Firefox en el Mac con la 2.19.0: todo funciona salvo el código en Cloudflare**, que no se
detectaba —seis casillas que declaran `one-time-code` y ninguna con `maxlength="1"`—. Arreglado en la
2.19.1: su formulario copiado de la consola está en las pruebas, y la escritura se ha comprobado contra
**su mismo componente**, `OTPField` de Base UI, en React. **Y con la 2.19.1 funciona en Cloudflare, en
Firefox y en Chrome**, comprobado por el cliente el mismo día.

### ~~3. La segunda pasada visual de la extensión~~ — hecha y comprobada en el Mac el 2026-09-14 (2.20.3)

El cliente la pidió después de usar la 2.19.1, **antes** de guardar desde la página, y la decidió por
preguntas con opciones ([ADR 0031](adr/0031-el-icono-con-estados-y-la-marca-en-el-campo.md) y la
ampliación de la [0029](adr/0029-el-panel-de-la-extension-es-esfinge.md)):

- **El icono de la barra es la silueta de Esfinge sin placa**, con contorno de piedra para verse en
  barras claras y oscuras, y **dice el estado de cada pestaña**: número de cuentas, ✓ si ha rellenado,
  candado si la bóveda está cerrada, «!» si algo falla, y la frase al pasar el ratón. Se pone al día
  al cambiar de pestaña y **cada minuto** (permiso `alarms`).
- **El campo rellenado lleva un filete** de oro y piedra hasta que se escribe en él, y en los rellenos
  automáticos **un aviso de tres segundos**, «Rellenado por Esfinge». Esto **matiza la ADR 0028**: en
  la página ya se dibuja algo, aunque no se pueda pulsar ni dure.
- **El panel**: cuadro con la inicial en cada cuenta —la misma función que la ventana—, icono de la web
  sin salir a internet, «● Abierta», «✓ Hecho» en la fila, esfinge tenue en los avisos, cuenta atrás
  del código, firma al pie y atajos de teclado.

Lo que falta es **verlo en el Mac**: si la silueta se lee en la barra, si el candado aparece solo, el
filete y el aviso en Brevo y Cloudflare, y si el correo cabe entero en el panel.

**Y probada en Firefox con la 2.20.0** (2026-09-14): el número y el ✓ del icono se ven bien. El cliente
pidió cuatro correcciones —el filete sin borde negro, el marco de la fila solo con teclado, el candado
del tamaño de una insignia y la cara de la esfinge del aviso en crema—, hechas en la 2.20.1. Falta verlas
en el Mac, y lo que quedaba de antes: si el candado aparece solo al cerrarse la bóveda y si el correo
cabe entero.

Y con la 2.20.1 el cliente pidió **el candado otra vez más grande y el aviso más grande**: en la 2.20.2
el candado es una placa naranja de dos tercios del icono con el candado blanco, y el aviso pasa a 14 px.

Con la 2.20.2 **todo se ve bien en Firefox y en Chrome**. Quedaban dos detalles, hechos en la 2.20.3: la
esfinge tenue de los avisos, entera y con su trazo de serie, y «Rellenar» con la misma letra en Chrome
que en Firefox. **Y con la 2.20.3 el cliente dice «Ahora está perfecto»**: la parte visual de la
extensión queda cerrada.

### ~~4. Guardar y actualizar desde la página~~ — comprobado en el Mac el 2026-09-14 (2.21.2)

Resuelto en la [ADR 0032](adr/0032-guardar-desde-la-pagina.md), decidido con el cliente por preguntas
con opciones. **Al enviar un formulario, una tarjeta arriba a la derecha ofrece guardar la cuenta** —o
actualizar la contraseña, eligiendo la cuenta si hay varias—, con el título del sitio editable, «Ahora no»
y «Nunca en este sitio», que se deshace en Ajustes. Con la bóveda cerrada, la tarjeta dice que la abras.

Dos cosas que cambian y hay que decir: **el navegador escribe en la bóveda** por primera vez —con su
propio freno de seis por minuto y la contraseña anterior al historial— y **la tarjeta es lo primero de
Esfinge que se pulsa en la página de otro**. La contraseña enviada espera **en la memoria del trabajador
de fondo** y no llega a la página siguiente; qué ofrecer lo decide Go; y si el formulario vuelve a salir,
se da por mala y no se ofrece.

Probado aquí por piezas y la tubería de Go entera. **Lo que falta es usarlo en el Mac, en Firefox y en
Chrome**: entrar con una cuenta nueva, entrar con otra contraseña, registrarse, cambiar la contraseña,
un inicio fallido, «Nunca en este sitio» y que la ventana se ponga al día sola.

**Publicada la 2.21.0 el 2026-09-14** (todas las compilaciones en verde). **La siguiente acción concreta
es recoger lo que diga el cliente**, que la prueba la tarde del 14 en su Mac, en Firefox y en Chrome,
con la lista de ocho pasos: cuenta nueva, otra contraseña con la anterior en el historial, registro,
cambio de contraseña, **inicio fallido sin tarjeta**, «Nunca en este sitio» y quitarlo en Ajustes, la
ventana al día sola, y la bóveda cerrada con «Ya la he abierto». Hay que **reinstalar la extensión** con
los zip de la 2.21.0, o la tarjeta no sale.

**Con la 2.21.0 probó hasta cambiar la contraseña** y salió un fallo en Google, que pide usuario y
contraseña en páginas separadas: rellenaba y ofrecía actualizar la cuenta de otro usuario. **Corregido
en la 2.21.1** recordando el usuario de la página anterior, y con varias cuentas se rellena la que
coincida con él, a petición del cliente. **Publicada la 2.21.1 el mismo día**, con todas las
compilaciones en verde, y **Google funciona perfecto** con ella. Después, **cambiar la contraseña en Brevo
no ofrecía nada** —actual y nueva sin `autocomplete`, solo distinguidas por el nombre—: corregido en la
**2.21.2, publicada el mismo día**. **Y Brevo también funciona**, comprobado por el cliente. **Y con la 2.21.2 los pasos 5 a 8 también funcionan**, dicho por el cliente: el inicio fallido no ofrece nada, «Nunca en este sitio» se deshace en Ajustes, la ventana se pone al día sola y con la bóveda cerrada «Ya la he abierto» deja guardar. Y **la extensión no escribe nada en la consola** por
defecto, decidido por el cliente: los diagnósticos se le piden con un fragmento para pegar (ADR 0032, «Corregido tras probarla en
Google»). **Quedan por probar los pasos 5 a 8.**

Se le explicó cómo decide «la contraseña era mala» y se le pidió que apunte **los sitios donde falle en
cualquiera de los dos sentidos** —no sale tras entrar bien, o sale tras un error—: esa heurística se
afina con esos casos (`docs/deuda.md`). Si algo no hace nada, lo primero es la consola de la extensión.

**La siguiente acción concreta son las tiendas (entrega 5)**, que hay que plantear con el cliente antes de escribir nada.

### 5. Las tiendas (entrega 5) — en curso desde el 2026-09-14

**Decidida y planificada el 2026-09-14** (plan aprobado por el cliente), por preguntas con opciones y tras
investigar las dos tiendas en su documentación oficial: **públicas en Chrome y en Firefox**, subida
automática al publicar, cuentas de Webcafeína con info@webcafeina.com, **aviso de consentimiento en el
panel la primera vez**, datos de comerciante públicos en la ficha de Chrome, **web del proyecto con GitHub
Pages** para la privacidad y el soporte, y **Windows arreglado antes de publicar**.

Cuatro bloques, en orden:

1. **~~Windows~~ — hecho el 2026-09-14, sin publicar** (ADR 0034): manifiestos apuntados desde el registro
   y el puente en el instalador. Probado con un registro de mentira aquí; el de verdad y el instalador se
   comprueban en la máquina Windows de GitHub al publicar. **Ningún navegador lo ha lanzado en Windows.**
2. **~~La extensión, lista para las tiendas~~ — hecho el 2026-09-14, sin publicar** (ADR 0033): el aviso
   de datos en el panel la primera vez, que hacen cumplir también el trabajador de fondo y el guion de la
   página; sin `activeTab`; Firefox 140 con `data_collection_permissions`; la versión pasada de verdad;
   **sin `innerHTML`**; y el código fuente para Mozilla, **reproducible byte a byte** en una carpeta limpia.
   Queda para el bloque 4 quitar `key` del paquete de la tienda de Chrome.
3. **~~La web del proyecto~~ — hecha el 2026-09-14, sin publicar** (ADR 0033): `web/` con portada,
   política de privacidad y soporte, con los tokens de la ventana y la estructura de `ollama`, armada por
   `herramientas/armar-web.sh` y publicada por `.github/workflows/web.yml`. **Falta activar GitHub Pages**
   en el repositorio, con «GitHub Actions» como origen, antes de subir el flujo.
4. **~~Las fichas y la subida automática~~ — hecho el 2026-09-14** (ADR 0033): `docs/tiendas/` con los
   textos, las respuestas de privacidad y las imágenes miradas; el zip de Chrome sin `key` y el de código
   fuente en cada publicación; y el trabajo `tiendas`, que comprueba la reproducibilidad y sube a Firefox
   y a Chrome, **o avisa y se salta sin secretos**. **Sin probar contra las tiendas.**

**La 2.22.0, instalada a mano en el Mac del cliente, «todo parece correcto»** (2026-09-14): el aviso de datos
y lo de antes siguen funcionando.

**2026-09-15: la extensión, enviada a Firefox.** El cliente dio de alta la cuenta de Mozilla y guardó sus claves
en GitHub, y la **2.22.1** —sin cambios en la aplicación ni en la extensión— corrió el trabajo `tiendas` por
primera vez: el código fuente dio el mismo paquete byte a byte y **Mozilla validó la extensión, recibió el
código fuente y la dejó esperando revisión** (versión 6486838 en su panel). Chrome se saltó, como estaba
previsto, y la publicación trae ya `esfinge-extension-2.22.1-chrome-tienda.zip` para la primera subida a mano.

**Y Chrome, en marcha**: subido el zip a la consola de la tienda, que asignó el identificador
`jfkkegampjamnnlopobepjoanebemegp`, ya añadido a Esfinge sin publicar. **Ficha y prácticas de privacidad
rellenadas y enviada a revisión el mismo día**, con el aviso de Google de que irá a revisión a fondo por los
permisos de host amplios —esperado: sin ellos no rellena sola ni ofrece guardar—.

**Decidido con el cliente: esperar a que terminen las dos revisiones y publicar la 2.22.2 entonces.**
Publicarla ya mandaría a Firefox una segunda versión idéntica a revisar, porque el trabajo `tiendas` sube
la extensión en cada publicación. **Se asume** que, si Chrome la aprueba antes, quien la instale desde la
tienda verá «No se encuentra Esfinge» hasta la 2.22.2: hoy tiene pocos usuarios o ninguno.

**2026-09-18: las dos revisiones aprobadas y las dos fichas públicas**, con la 2.22.1:
Firefox en https://addons.mozilla.org/es-ES/firefox/addon/esfinge/ y Chrome en
https://chromewebstore.google.com/detail/esfinge/jfkkegampjamnnlopobepjoanebemegp. La web y el README las
enlazan. Decidido con el cliente ese día: **seguir subiendo la extensión en cada publicación**, **quitar sus
zip de la publicación de GitHub** y montar la cuenta de servicio de Chrome. Después, la 2.22.2.

**Y la 2.22.2, publicada el mismo día**: Esfinge deja entrar a la extensión de la Chrome Web Store, la
publicación de GitHub ya no lleva los zip para cargarla a mano, y **el trabajo `tiendas` subió la extensión a
las dos tiendas solo**: a Firefox con `web-ext` y **a Chrome por la API v2 con la cuenta de servicio, a la
primera** —subida aceptada y enviada a revisión—. **La entrega 5 queda cerrada**: lo que falta de ella es
esperar esas revisiones, que ya no piden nada a nadie.

~~**La siguiente acción concreta era esperar las dos revisiones**~~ —Mozilla, normalmente menos de un día;
Google, días o semanas, a fondo— y, cuando terminen, publicar la 2.22.2. Pendiente también, sin prisa, la
cuenta de servicio de Chrome (`pasos.md`, 2.4). Y **queda por decidir con el cliente** si la extensión sube
a las tiendas solo cuando cambia su código, para que una versión que solo toca la aplicación no mande
nada a revisar.

**Lo que queda, con el cliente**: los pasos de `docs/tiendas/pasos.md` —cuentas de Mozilla
y de Google, la primera subida a Chrome a mano, los secretos en GitHub— y **pasarme el identificador de
Chrome**, que hay que añadir a Esfinge.

Lo que se preguntó entonces, ya decidido:

- **Pública u oculta** en cada tienda: la de Chrome admite «no listada» y la de Firefox (AMO) permite
  firmar sin listar.
- **A nombre de quién** van las cuentas de desarrollador —Webcafeína— y quién las administra. La de
  Chrome cuesta una cuota única.
- **El identificador de la extensión en Chrome cambia al subirla**: hay que añadirlo al manifiesto de
  native messaging al lado del actual (`docs/deuda.md`, el canal del navegador), o fijar la clave como
  ahora.
- **Qué contar en las fichas**: los permisos —`alarms`, anfitriones, `favicon`, native messaging— tienen
  que justificarse, y hace falta una política de privacidad que diga lo que ya dice `docs/seguridad.md`.
- **Cómo se actualiza**: hoy la extensión viaja en cada publicación de GitHub en dos zip; con las tiendas,
  si se sigue publicando ahí y quién sube cada versión.
- Y lo que queda de Windows para que la extensión funcione allí: **el manifiesto en el registro** y el
  instalador sin `esfinge-puente` (`docs/deuda.md`).

### 6. Las cuentas (fase 3) — decidida y planificada el 2026-09-18

**La bóveda en todos los equipos, compartir copias con otras cuentas y la extensión como cliente
propio**, con un servidor nuestro en Cloudflare UE que no puede leer nada. Lo decidido con el cliente
está en la [ADR 0035](adr/0035-las-cuentas.md) y el plan entero —criptografía, fusión, servidor,
aplicación, extensión y entregas— en [`docs/cuentas.md`](cuentas.md).

Las entregas, en orden: **A0** el servidor solo; **A1** (2.23.0) el modelo y la fusión, sin interfaz;
**A2** la bienvenida, la cuenta y la sincronización, por invitación; **A3** contraseña, recuperación,
equipos y borrado; **E** la extensión autónoma; **A4** abrir el registro; **B** compartir; **C** Touch ID
o PIN.

**~~La A0~~ — hecha, desplegada y comprobada el 2026-09-18** ([ADR 0036](adr/0036-el-servidor-de-cuentas.md)).
`servidor/` con 39 pruebas dentro de `workerd`, en `make comprobar` y en la puerta de `publicar.yml`;
`make servidor` para levantarlo en local; `.github/workflows/servidor.yml` para desplegarlo a mano. Un
cambio sobre el plan: **la bóveda va en el Durable Object de su cuenta y no en R2**.

Desplegado guiando al cliente: las dos bases D1 **con jurisdicción UE** (leída en la API; producción
contesta desde Milán), el token en GitHub, los secretos puestos por él en el panel —nunca por el chat— y
el entorno `produccion` de GitHub, **solo desde `main` y sin aprobación**, que el cliente no quiere tener
que dar cada vez. Producción vive en **https://esfinge-cuentas.webcafeina.com**, **por invitación** con
`@webcafeina.com` en la lista; el de pruebas, en `https://esfinge-cuentas-pruebas.webcafe-na.workers.dev`.
**El código por correo llega a la bandeja de entrada, con SPF, DKIM y DMARC en `PASS`.**

**~~La A1~~ — hecha el 2026-09-18, sin publicar** ([ADR 0037](adr/0037-claves-de-la-cuenta.md) y
[0038](adr/0038-sincronizar-la-boveda.md), formato en [`formato-boveda.md`](formato-boveda.md)): la bóveda
lleva revisiones, lápidas de seis meses y la versión del servidor sellada; se abre sin escribir, se sube sin
las ranuras de este equipo y **se funde a tres bandas**; `internal/cuenta` deriva la clave de acceso y habla
con el servidor —leyendo el `ETag` débil—, e `internal/sincro` hace las pasadas y vigila. **Nada se ve en la
ventana todavía.** Probado con tres equipos al azar (16 semillas), con una bóveda de la 2.22.2 de verdad y
**contra el servidor de verdad levantado en local**, en `make comprobar` y en la puerta de publicación.

**~~La A2~~ — hecha el 2026-09-18, sin publicar** ([ADR 0039](adr/0039-la-bienvenida-y-la-cuenta-en-la-ventana.md)):
la bienvenida a ventana entera al estrenar Esfinge sin nada, el asistente para crear la cuenta y para
entrar —juntando o apartando la bóveda que ya hubiera, **que nunca se borra**—, la línea de la
sincronización en la bóveda y el grupo «Cuenta y sincronización» en Ajustes, **con «Dejar la cuenta en
este equipo» adelantado de la A3** porque la bienvenida promete poder cambiar de idea. Por debajo,
`internal/app/cuenta.go`: la sesión sellada con la bóveda en `cuenta.json`, la sincronización enganchada a
abrir, cerrar, bloquear y borrar, **subir lo pendiente al cerrar** —lo cazó la prueba de dos equipos— y
nada que cuente como actividad. Probado en Go contra el servidor de verdad y **en la ventana con dos
equipos a la vez** (`e2e/cuentas.spec.ts`), con las capturas miradas en los dos temas. La política de
privacidad, la portada, el README y `docs/seguridad.md` ya cuentan la cuenta.

**Chrome aprobó la 2.22.2 el 2026-09-18**, y el cliente dijo «adelante con todo»: **la 2.23.0 se publicó
el mismo día**, con la A1 y la A2, todo en verde y la extensión subida sola a las dos tiendas. Antes, un agujero que salió al contestarle si era estable: **con cuenta,
cambiar la contraseña maestra en Ajustes la cambiaba solo en la bóveda**, y un equipo nuevo no podría
entrar. Queda bloqueado con un mensaje hasta la A3, y **no se invita a nadie más hasta entonces**.

**La siguiente acción concreta es que el cliente la pruebe en sus dos Macs, el lunes 21 de septiembre en
la oficina** (el viernes 18 no tenía dos a mano): crear la cuenta desde Ajustes con la bóveda que ya tiene,
entrar desde el otro con el código, y trabajar en los dos —una entrada en uno que aparece en el otro, la
misma entrada editada a la vez, guardar desde el navegador y cerrar justo después de guardar—. Se le dio el
guion paso a paso. **Al volver, se empieza por preguntarle cómo ha ido.**

**2026-09-21, probándola en la oficina**, el cliente encontró dos cosas al crear la cuenta con su bóveda
de siempre, arregladas y publicadas al momento en la **2.23.1** —prefiere ir viéndolo en real—: pedía
abrir la bóveda antes aunque la contraseña ya estaba escrita, y no había salida si su contraseña no
llegaba a «Buena». Sigue la prueba.

**Con la 2.23.1, «todo funciona, las modificaciones viajan»**, con una pega: los cambios del otro Mac solo
se veían al cerrar y volver a abrir la bóveda. Era el plazo —cada cinco minutos— y que no se miraba al
volver a la ventana; la lista sí se refrescaba sola al llegar algo. **2.23.2**: se mira al volver a la
ventana y cada minuto.

**Con la 2.23.2, comprobado por el cliente en sus dos Macs: «funciona perfecto»**. Borró una entrada en
uno y **desapareció casi al instante en el otro, sin cerrar la bóveda**. La A2 queda comprobada de verdad.

**~~La A3~~ — hecha el 2026-09-21, en la 2.24.0** ([ADR 0037](adr/0037-claves-de-la-cuenta.md)):
cambiar la contraseña con cuenta —en el servidor y aquí a la vez—, el equipo que se quedó con la de antes
se pone al día al entrar con la nueva **sin perder lo que tenía sin subir**, recuperar la cuenta con la
clave de recuperación sin ningún equipo a mano, la lista de equipos con «Olvidar», y exportar y borrar la
cuenta desde Ajustes. Probado contra el servidor de verdad y en la ventana, con capturas miradas.

**2.24.1**: con la contraseña cambiada en un Mac, **el otro se abre escribiendo la nueva** en «Abrir la
bóveda», sin código ni pasos aparte. Lo preguntó el cliente al ver la 2.24.0; cambiar la contraseña sigue
cerrando las sesiones de los demás equipos, que es lo que eligió. **Comprobado por el cliente en sus dos
Macs**: cambiada en uno, el otro entró en la cuenta con la nueva.

**2.24.2**, de lo que vio el cliente probándola:
- En su segundo Mac, la nueva daba «Esa llave no abre esta bóveda» —el testigo de confianza venía sellado
  de la 2.24.0—. Ahora, sin testigo, la nueva se reconoce y «Abrir la bóveda» pide el código del correo en
  la misma pantalla. Pasaría también al caducar el testigo, a los 90 días.
- **Cada cuenta estaba dos veces en los dos Macs**: salió de «Juntar» al entrar en la cuenta con el
  segundo, que tenía su propia importación de Dashlane. Juntar ya no repite lo igual, y la bóveda avisa de
  las repetidas y **las quita de un clic, a la papelera**. El cliente tiene que pulsarlo en uno de los dos.
- **2.24.3**: el aviso de las repetidas no salía en los Macs del cliente —«igual en todo» era demasiado
  estricto, y sus parejas diferían en lo que no se ve—. Ahora cuenta como la misma cuenta mismo título,
  usuario y contraseña, y junta lo demás en la que se queda.
- **Comprobado por el cliente con la 2.24.3**: 66 repetidas quitadas en un Mac, el otro al día solo, sin
  parejas sueltas; exportar, bien.
- **2.24.4**: la fila «Bóveda» de la barra lateral lleva un candado pequeño, cerrado o con el arco
  levantado, que lo pidió el cliente. Lo avisa Go en cada apertura y cierre (`boveda-estado`), así que
  vale también cuando se cierra sola o se abre desde la cuenta.
- «Exportar» no salía en Ajustes con la bóveda cerrada, y se llamaba «Guardar lo que hay de mi cuenta».
  Ahora está siempre con cuenta, como «Exportar los datos de la cuenta…», y dice que hay que abrir la
  bóveda para usarlo.

**Lo que dijo el cliente de la 2.24.4 (2026-09-22)**: el candado, perfecto; recuperar la cuenta, perfecto;
olvidar un equipo pedía volver a entrar pero **dejaba la bóveda abierta**, y pidió que se cerrara →
**2.24.5**: al perder la sesión, la bóveda se cierra y dice por qué. Borrar la cuenta no lo va a probar.

**La E2, comprobada por el cliente en Chrome (2.25.1, cargada a mano): «funciona todo perfecto»**. Pidió
un botón para sincronizar a mano en la aplicación y en la extensión → **2.25.2**, y que fueran flechas que
giran → **2.25.3**, comprobada: «funciona perfecto». **La siguiente acción concreta es esperar a que las
dos tiendas aprueben la extensión** y probarla entonces en Firefox —lo único que queda de la E3—, con la
versión de la tienda en Chrome en vez de la cargada a mano.

**Revisión de seguridad (2026-09-23)** — [`docs/revision-2026-09.md`](revision-2026-09.md). Cuatro pasadas
del modelo Fable —servidor, extensión, criptografía y fusión—, con cada hallazgo verificado a mano, sobre
el código entero. Salieron **tres
graves**: un campo escondido por su contenedor se rellenaba solo (arreglado), la extensión puede pisar su
bóveda si la fusión falla (propuesto), y «muchos borrados» salta al usar la papelera y no tiene salida
(propuesto). Arreglados también: la enumeración de correos por el freno, el testigo de confianza que
sobrevivía al cambio de contraseña —ahora el otro equipo pide el código una vez, elegido por el cliente—,
quién puede pedir lo de la cuenta en la extensión, y la tarjeta que se podía pulsar nada más aparecer.
**Los tres pendientes quedaron hechos el mismo día (2.25.6)**, y **lo que el informe dejaba abierto quedó
hecho el 2026-09-23**: el borrado suave que ganaba a una edición, los cinco intentos del código de alta
—ahora una sentencia atómica y con freno por IP—, el sello que no cubría el `creado` de cada sobre —un
campo nuevo que las bóvedas de antes no traen y se abren igual—, `Fundir` que cambiaba la memoria sin saber
si el guardado salía bien, y el tope de Argon2id, que baja de 1 GiB a 256 MiB en los cuatro sitios que lo
declaran. Con ello, **`cambiada` se toca al borrar y al restaurar**, sin lo cual restaurar perdía contra la
purga de otro equipo al fundir sin base. Lo que **no** se arregla y queda apuntado: que el servidor pueda
congelar o bifurcar la cadena de versiones, que los plazos de la papelera dependan del reloj de cada
equipo, el Worker de pruebas público con su buzón abierto —eso lo tiene que cerrar el cliente en su panel
de Cloudflare— y si algún día se recifra al rotar la clave de recuperación.

**El servidor está desplegado (2026-09-23)**: `pruebas` y `produccion`, los dos en verde. Producción
contesta `{"registro":"lista","protocolo":1}`, la pre-entrada devuelve sal y coste, y la ruta del buzón de
pruebas da 404 allí, como debe. Con ello, los cambios de cuenta de la revisión —los cupos por propósito, el
testigo de confianza que se va al cambiar la contraseña, el código de alta atómico y con freno, y el tope de
Argon2id— ya están en el servidor que usan sus dos Macs.

**La 2.25.7 se publicó el 2026-09-23** con todo lo anterior, decidido por el cliente, y **con los siete
trabajos en verde, tiendas incluidas**.

**Y las dos tiendas la sirven ya** (comprobado el 2026-09-23 por la tarde): AMO da `2.25.7` en estado
`public`, revisada a las 11:31 UTC —dos minutos después de la publicación—, y el manifiesto de
actualización de Chrome entrega el `.crx` de la `2.25.7`. Se comprueba sin abrir el navegador:

```sh
curl -s https://addons.mozilla.org/api/v5/addons/addon/esfinge/   # .current_version.version
curl -s "https://clients2.google.com/service/update2/crx?response=updatecheck&prodversion=131\
&acceptformat=crx3&x=id%3Djfkkegampjamnnlopobepjoanebemegp%26uc"   # atributo version=
```

**La B1, hecha (2026-09-23)** — [ADR 0043](adr/0043-la-identidad-para-compartir.md). La identidad para
compartir: una semilla de 32 bytes en el cuerpo cifrado, de la que salen X25519 para recibir y Ed25519 para
firmar, con una huella que se puede leer por teléfono. En Go y en TypeScript, con pruebas cruzadas. **Sin
servidor y sin interfaz todavía**: lo siguiente es la B2, mandar y recibir. Y una corrección: la tabla de
entregas decía que la A2 había creado la identidad, y no era verdad.

**La B2, hecha (2026-09-23)** — compartir de punta a punta, en las dos caras. El sobre HPKE firmado con
Ed25519 en Go y en TypeScript, con pruebas cruzadas; el servidor con las llaves, el envío y el buzón, sin
delatar quién tiene cuenta; **la ventana**, con «Compartir» en una entrada, la huella con su aviso antes de
mandar y el buzón que se acepta; y **el panel de la extensión**, con lo mismo: el buzón sale solo cuando
hay algo, la copia no entra en la bóveda hasta pulsar «Guardar», y desde la fila de una cuenta se manda una
copia en dos pulsaciones —primero la huella de quien la recibe, después el envío—. Lo ejercita todo
`navegador/pruebas-reales` con **la extensión cargada de verdad** y dos cuentas contra el servidor local.

De escribir esa prueba salió lo que faltaba y no se veía: **las llaves se publicaban solo al abrir
«Compartir»**, así que quien nunca hubiera mandado nada **no podía recibir** —el servidor le daba a quien le
mandaba una llave inventada y el sobre llegaba ilegible, sin que ninguno de los dos pudiera saber por qué—.
Ahora las publica la ventana al arrancar la sincronización y la extensión en cada pasada ([ADR
0043](adr/0043-la-identidad-para-compartir.md)).

**La B3, hecha (2026-09-24)** — las invitaciones, que cierran el agujero de que un envío a quien no tiene
cuenta se perdiera en silencio. Lo eligió el cliente con dos decisiones que mandan sobre todo lo demás:
**la contraseña no pasa por el correo** y **el correo lleva quién invita, con un botón**.

Cómo queda: al mandar una copia, el servidor mira si esa dirección tiene cuenta. Si la tiene, el sobre va a
su buzón; si no, **el sobre no sale** —iría cifrado hacia unas llaves inventadas— y le manda una invitación.
Quien la mandó se queda **una nota dentro de su bóveda**, cifrada y sincronizada a sus equipos, y en cuanto
esa dirección publica sus llaves —o sea, cuando crea su cuenta— la copia sale sola en la siguiente pasada de
sincronización. A los treinta días se deja de intentar.

Lo que sostiene todo eso: **el servidor contesta exactamente lo mismo en los dos casos**, así que compartir
no se convierte en una forma de averiguar quién tiene cuenta. De ahí que el camino de la invitación se trague
sus errores —el tope de cinco por cuenta y día, un fallo de Resend, su cupo— y que el correo salga en
`waitUntil`, porque esperar a Resend también se mide desde fuera. Y de ahí que lo que dice la pantalla valga
para los dos casos: «Si ya tiene cuenta, le espera en su buzón; si no, le hemos mandado una invitación».

**La B4, hecha (2026-09-24)** — lo que hay que decir, que es lo que faltaba para poder publicar compartir:

- **La política de privacidad** cuenta que el servidor ve **a quién le mandas** una copia, qué guarda de
  compartir —tus claves públicas y los sobres del buzón, que no puede abrir— y, sobre todo, **la
  invitación**: un correo a alguien que no es cliente nuestro, con la dirección de quien invita dentro, que
  no lleva nada del secreto, se manda una vez y cuya dirección se borra a los treinta días.
- **Las condiciones** ganan «Compartir una copia» —es una copia y deja de ser tuya, no se dice si ha
  llegado, y si no tiene cuenta hay que abrir Esfinge antes de treinta días—, sus topes, y una línea en lo
  que se pide: **las invitaciones no se usan para escribir a quien no lo espera**.
- **`VERSION_DEL_AVISO` sube a 3**, así que el panel vuelve a preguntar: por él sale ahora un dato nuevo, la
  dirección de quien recibe la copia.
- Y con ello, **las dos fichas, la portada y el README**.

De hacerlo salió una trampa que llevaba tiempo puesta: la copia en texto de la política —la que se pega en
la ficha de Firefox— **se mantenía a mano**, que es lo mismo escrito en dos sitios. Ahora se genera desde la
web y `make comprobar` falla si no está al día. Y al escribir eso apareció lo mismo en un tercer sitio:
`herramientas/capturas.mjs` daba por aceptada la **versión 1** del aviso, así que desde que subió a 2 todas
las capturas del panel salían del aviso de datos y nadie lo había visto.

**Con esto, compartir se puede publicar.** Lo que queda de la B es nada.

**Y se publica: la 2.26.0** (2026-09-24), decidida por el cliente. Lleva la B entera —compartir copias en la
ventana y en la extensión, el buzón que se acepta, las invitaciones a quien no tiene cuenta— y **los diez
correos maquetados**. Dos cosas que hay que saber de esta versión:

- **`VERSION_DEL_AVISO` sube a 3**, así que el panel de la extensión vuelve a preguntar «Esfinge y tus
  datos» a todo el mundo una vez. Es a propósito: por el canal sale un dato nuevo, la dirección de quien
  recibe la copia.
- **Las dos tiendas la revisan**, como cada versión de la extensión.

**La C, estudiada (2026-09-24) y sin empezar** — [`desbloqueo-del-sistema.md`](desbloqueo-del-sistema.md).
El plan avisaba de que había que mirarla antes por si exigía firmar, y lo que ha salido **cambia de qué
va**: sin firmar, en los dos sistemas «desbloquear con el sistema» es **un cerrojo y no una llave**.

- En **macOS**, el camino fuerte —Secure Enclave o control de acceso biométrico en el llavero— vive en el
  llavero de protección de datos, que **exige la entitlement `keychain-access-groups`**, y ésa solo la
  lleva una compilación firmada con un perfil de aprovisionamiento. Es justo lo que descartó la ADR 0014.
  Lo que sí funciona sin firmar es `LAContext`, que devuelve **un sí o un no**: la clave la guarda otra
  cosa, y quien lea eso abre la bóveda sin poner el dedo.
- En **Windows** se puede sin empaquetar, pero Microsoft lo dice claro: la credencial de Hello **está
  atada a la cuenta de usuario, no a la aplicación**, así que otro programa tuyo que sepa su nombre la
  usa. Su propia recomendación es contraseña **y** Hello, no Hello en lugar de la contraseña.
- Y **un PIN de cuatro cifras no protege un fichero**: sin hardware que limite los intentos, se prueban
  todos. Solo vale como desbloqueo rápido dentro de una sesión ya abierta, y hay que llamarlo así.

**Y la C1, hecha el mismo día** tras decidirlo contigo: **no se firma en macOS**, **se hace en los tres
sistemas** y **no hay PIN** —«siempre será la contraseña maestra»—. Lo que hay ya: la ranura
`llavero-del-sistema` en la bóveda, `internal/llavero` con su llavero de mentira para poder moverlo aquí, los
cuatro métodos de la aplicación, el interruptor de Ajustes —donde se dice que esto **es un cerrojo**— y el
botón de la pantalla de desbloquear. Con pruebas de Go y e2e de las dos pantallas en los dos temas.

**Y la C2, escrita el 2026-09-24**: `internal/llavero/llavero_darwin.go`, cgo con `LAContext` para pedir
la huella y el **llavero de inicio de sesión** para guardar el secreto. Lo que se decidió al escribirla, y
está razonado en el documento de la fase: el secreto **no** va a un fichero al lado de la bóveda, aunque
eso sería predecible y sobreviviría a las actualizaciones, porque entonces **quien copie tu carpeta abre
la bóveda sin poner el dedo** — eso no es un cerrojo peor, es quitar la puerta. En el llavero está cifrado
con la contraseña de macOS, y lo que se paga a cambio es que macOS puede preguntar tras cada
actualización, porque el binario cambia y Esfinge no está firmada.

**Y al escribirla salió que Windows llevaba roto desde la C1.** Al partir `internal/llavero` por sistemas,
el fichero de Linux se marcó `!darwin && !windows` contando con que los otros dos llegarían enseguida, y
**los dos objetivos de Windows se quedaron sin `delSistema`**: se habría caído en medio de una
publicación, porque `go vet` mira solo esta máquina y la línea de comandos se cruza en `make publicar`.
Con ello salió la segunda mitad, que no es lo mismo: **un fichero de cgo no entra cuando `CGO_ENABLED=0`**,
que es como se cruza esa línea de comandos, así que un `_darwin.go` de cgo necesita su pareja
`darwin && !cgo`. Ahora **`make comprobar` compila los seis objetivos**, que tarda segundos y para algo.

**Lo siguiente de la C es la C3**, Windows Hello, que se escribe a ciegas: nadie ha ejecutado nunca Esfinge
en un Windows. Y de la C2 queda lo que solo dice un Mac de verdad: que arranque, qué pregunta al
actualizar, y cómo se lee el diálogo.

El documento deja **cuatro cosas que decidir antes de escribir nada**, y la primera ya está contestada:
**de momento no se firma en macOS** (cliente, 2026-09-24). Con eso, si la C se hace, es el camino del
cerrojo, y **hay que escribirlo donde se active y en `seguridad.md`**: protege de quien se sienta delante
de tu ordenador desbloqueado, no de un programa que corra como tú.

**Y las dos cosas que solo podía comprobar el cliente, hechas el mismo 2026-09-24:**

- **La invitación, en un buzón de verdad.** Se creó una cuenta de usar y tirar en producción, se le mandó
  una copia a una dirección sin cuenta y el cliente miró el correo en Gmail. **Salió un fallo que aquí no
  decía nada**: el botón estaba escrito como un `<a>` con `background:` abreviado —lo que vale en cualquier
  navegador— y **Gmail no lo pintaba**. Ahora es una celda con `bgcolor`, que además arregla Outlook de
  escritorio. Comprobado después: **botón, icono y sin spam**, y bien en el móvil. Y de paso pidió el icono
  de Esfinge en la cabecera, que se puso **con el coste delante** —pedir esa imagen cuenta a quien la sirve
  que el correo se ha abierto— y queda dicho en la política.
- **El aviso del cambio de condiciones: no hace falta.** En producción hay **una sola cuenta**, la de la
  casa. El punto 13 promete treinta días de aviso por correo antes de cambiarlas, y no hay a quién avisar.
  Si algún día hay cuentas de fuera, ese aviso deja de ser una formalidad.

Lo que queda sin mirar de esto: **el modo oscuro de los clientes de correo**, que algunos invierten los
colores por su cuenta y ahí la piedra sobre el oro podría acabar en blanco sobre oro, 1,68:1. Y la forma de
comprobarlo de ahora en adelante la dijo el cliente: **los correos de verdad que se vayan mandando**.

**Y con ello, los diez correos dejan de ser texto pelado** (2026-09-24, pedido por el cliente): una maqueta
compartida con la banda de marca, el código grande y seleccionable, el recuadro de aviso y botón solo donde
hay algo que abrir. **El texto pelado sigue yendo** y dice lo mismo. Convertirlos a HTML abrió un agujero
que en texto no existía —el nombre del equipo es texto libre del cliente, y ahí `<img src=x onerror=…>` es
una etiqueta—, así que todo lo interpolado va escapado y hay una prueba que lo intenta de verdad.

**Lo que viene después de todo esto (2026-09-23)**: el cliente pidió **passkeys**, «como Dashlane, que
sale un banner y es darle a Aceptar». Está estudiado y escrito en [`docs/passkeys.md`](passkeys.md), con lo
que ya se ha comprobado de los navegadores para no volver a empezar: **en Chrome no hay API de extensión
para esto** —la que lo parece, `chrome.webAuthenticationProxy`, es de escritorio remoto y suspende todo el
WebAuthn del navegador—, así que el camino es el de los demás gestores: **reemplazar
`navigator.credentials` en el mundo principal de la página**, que Chrome permite y Firefox también desde
la 128. **No es una tarea, es una fase**, y el documento dice por qué, cómo partirla en cuatro y las cinco
cosas que hay que decidir antes de tocar código.

**Y con ella, las tres tareas que quedaban, en el orden que puso el cliente:**

1. ~~**Cerrar el buzón del Worker de pruebas**~~ —entregaba los códigos de cualquier cuenta de ese
   servidor a quien preguntara— **hecho el 2026-09-23 por la tarde**. Una aplicación de Access **sobre la
   ruta** `_pruebas/buzon`, no sobre el Worker, con «Emails ending in `@webcafeina.com`»: el buzón redirige
   a la pantalla de identificación y la API sigue intacta, que era la condición. **Y abre**: el cliente
   entró con el código de su correo y vio el buzón vacío. Detalle y comprobaciones en
   [`deuda.md`](deuda.md).
2. **Probar la extensión en Firefox**, lo último de la E3 y **la última tarea del plan de cuentas**, dicho
   así por él tres veces. Lo de fuera del plan —las passkeys— va después.
3. ~~**Abrir el registro**~~ — **hecho el 2026-09-23**: producción contesta `{"registro":"abierto"}`, el
   tope del día baja a treinta altas para caber en los cien correos de Resend, y las frases de «por
   invitación» salieron de los cinco sitios donde estaban. Lo de antes decía: es la entrega **A4** del plan
   ([`docs/cuentas.md`](cuentas.md)), «solo configuración» —`REGISTRO=abierto`— pero con tres puertas
   delante:
   - **La revisión de la política de privacidad y unas condiciones de uso que todavía no existen.** El
     cliente decidió el 2026-09-23 que **la hace un modelo especializado**, no un abogado
     ([ADR 0041](adr/0041-los-papeles-de-la-cuenta.md)).
   - **Los encargos de tratamiento con Cloudflare y con Resend**, que son papeles que firma él en cada
     panel y que hoy no constan.
   - **Los topes diarios del servidor, por debajo de lo que da Resend.** El plan gratuito se queda
     ([ADR 0041](adr/0041-los-papeles-de-la-cuenta.md)): cien correos al día, y cada alta y cada equipo
     nuevo gastan uno. Hoy `TOPE_ALTAS_DIA` viene en **200**, el doble, y cuando Resend diga que no, el
     servidor contesta «No se ha podido mandar el correo. Prueba otra vez en un momento», que el día que
     se agote el cupo **no es verdad**: es hasta mañana. Se baja y se arregla la frase **antes** de abrir.
   - Y si se abre a clientes de Colombia, **mirar en la fuente** lo que dice la Ley 1581 de 2012 sobre
     guardar los datos en la UE, que la ADR 0035 dejó sin comprobar.

**2.25.4 publicada (2026-09-23)**: la nota del canal con cuenta, el vigilante que ya no se duerme al
encontrarse el turno cogido, y las pruebas de la ventana dejando informe, traza y captura cuando fallan en
la máquina de GitHub. La puerta la paró dos veces; la segunda no se reproduce aquí y está en la deuda.

**Firefox, probado por encima y aplazado (2026-09-23)**: el cliente lo miró un poco y prefiere seguir con
lo pendiente; **la prueba de la extensión en Firefox queda para cuando se acabe todo lo demás**, y con ella
lo único que falta de la E3. Lo que hay que probar entonces, con **Esfinge cerrada**: entrar con la cuenta
en el panel, rellenar, un código de un solo uso, guardar algo, y que ese cambio aparezca luego en la
aplicación. Y darle a la extensión los permisos de sitios a mano, que en MV3 de Firefox no se conceden al
instalar.

**Las dos tiendas han aprobado la 2.25.3** (comprobado el 2026-09-23: la ficha de AMO la sirve, y el canal
de actualizaciones de Chrome también). **La siguiente acción concreta es que el cliente termine la E3**:
quitar de Chrome la copia cargada a mano y usar la de la tienda, y probar en Firefox —donde hay que darle
los permisos de sitios a mano, que en MV3 no se conceden al instalar—.

~~**Pendiente de la E**~~ — **hecho en la 2.25.4 (2026-09-23)**: el cliente eligió
«dejarlo como está y avisar», así que el interruptor se queda y con cuenta lleva una nota que dice que no
hace falta. No se apaga solo al entrar en una cuenta: en un navegador donde no se haya entrado con la
cuenta, eso dejaría de rellenar sin avisar (ADR 0040). Lo que decía el pendiente: en Ajustes de la aplicación,
«Dejar que la extensión del navegador consulte la bóveda» **con cuenta sobra** —la extensión va siempre por
la cuenta—, pero sigue haciendo falta en local. Propuesta: mantenerlo, con una nota en modo cuenta («la
extensión no lo necesita: entra con tu cuenta en su panel») y apagado por defecto para quien estrene
Esfinge con cuenta.

**La E2, hecha (2026-09-22, 2.25.0)** — [ADR 0040](adr/0040-la-extension-cliente-de-la-cuenta.md): **la
extensión con cuenta funciona sin la aplicación**. En su panel se entra con el correo, la contraseña y el
código; la bóveda vive cifrada en el navegador, se abre con la maestra, se cierra a los quince minutos sin
tocarla o al cerrar el navegador, y se sincroniza sola; rellena, da códigos y guarda contra ella. Probada
**con la extensión cargada de verdad** en Chromium contra el servidor local, en la puerta. El aviso de
datos sube a la versión 2 y la política, la portada y las fichas de las tiendas lo cuentan. **La
siguiente acción concreta es la E3: que el cliente la pruebe en su Mac** —Chrome y Firefox, con la
aplicación cerrada— y que las dos tiendas acepten la versión nueva, que vuelven a revisar.

**La E1, hecha (2026-09-22, sin publicar: no cambia nada de lo instalado)** — [ADR 0040](adr/0040-la-extension-cliente-de-la-cuenta.md):
el núcleo de la bóveda en TypeScript (`navegador/src/nucleo/`: ESF1, bóveda, fusión, códigos y claves de
la cuenta), probado contra los vectores fijos de Go y **cruzado con Go** en `make comprobar` y en la
puerta: 4.000 fusiones al azar sin una diferencia, y tres roturas a propósito cazadas.

**2.24.5 comprobada por el cliente**: «funciona todo perfecto». **La A3 queda comprobada en sus Macs**, salvo
borrar la cuenta, que no va a probar con la suya. **Orden decidido con él (2026-09-22): primero la E —la
extensión como cliente de la cuenta—**; A4, B y C, detrás. La lista que probó:

1. **El candado de la barra lateral**: cerrado/abierto, que se cierre solo al bloquearse por inactividad, y
   el texto al pasar el ratón.
2. **Olvidar un equipo** desde el otro: el olvidado pide volver a entrar, con contraseña y código.
3. **Recuperar la cuenta** con la clave `ESF-…` (cambia su contraseña de verdad); después el otro Mac tiene
   que abrir escribiendo la nueva.
4. **Borrar la cuenta**, solo con una cuenta de prueba y un equipo fuera de la suya; si no, se deja.

Con eso la A3 queda comprobada entera en su Mac, y **toca decidir con él el orden de lo que queda**: la
extensión autónoma (E), abrir el registro (A4), compartir (B) y Touch ID (C). Con la
A3 hecha **ya se podría invitar a alguien más**: se le pregunta antes.

**Para retomar la A2** (sesión cerrada el 2026-09-18 por la tarde, a petición del cliente): el plan está en
`docs/cuentas.md` («La aplicación» y la tabla de entregas); el protocolo, en `servidor/LÉEME.md`; el formato,
en `docs/formato-boveda.md`. Lo primero de la A2 es la carrera de `BuscarEnBoveda` tras cerrar (deuda). El
Worker de producción **no se ha vuelto a desplegar** tras la A1: el único cambio del servidor —los topes por
IP y día configurables— deja producción igual, y entra con el próximo despliegue.

### Lo que yo recomendaría meter en medio, y no es lo que se decidió

**Levantar Chromium con la extensión cargada de verdad.** Lo dejo escrito porque la razón sigue en
pie aunque el orden sea otro: la entrega 2 salió con la detección de campos bien probada y **con lo
que la envuelve sin probar por nadie**, y ahí aparecieron los dos únicos fallos —el oyente del panel
contestando también desde las tramas de otro origen, y `yaRellenados` bloqueando un relleno pedido a
mano—. Los dos los vio el cliente a la primera. Playwright puede hacerlo (`launchPersistentContext`
con `--load-extension`, más un manifiesto de native messaging escrito en ese perfil apuntando a un
host de mentira). Está en `docs/deuda.md` con severidad alta.

### Y lo que solo dice el uso, que no es una tarea sino una escucha

Si el relleno acierta en un **sitio difícil** —un banco—; si **rellenar solo resulta demasiado**, y
entonces hace falta un interruptor en Ajustes en vez de afinar la detección a ciegas; y si el **clic
de más molesta** cuando hay varias cuentas del mismo sitio, que es de lo que depende el desplegable
dentro del campo.

Y sigue abierta la deuda de Windows, que la entrega 2 no toca: **el manifiesto del navegador va al
registro y no se escribe**, y el instalador no copia `esfinge-puente`.

**Y lo que sigue siendo verdad aunque la fase 2 esté empezada:** la puerta del plan era no empezarla
hasta usar la bóveda a diario, y se empezó con un día. Fue una decisión tomada a sabiendas, no un
descuido, y conviene recordarla si la entrega 2 se hace larga: lo que la justifica es que la bóveda
se use, no que la extensión avance.

De paso sigue pendiente juzgar el oro de la 2.11.0 puesto, y ver **cuántos de los 65 sitios acaban
con icono real**: de ese número depende si hay que leer el HTML de los que faltan.

Después, y sin prisa, sigue pendiente abrir la aplicación en **Windows** y en **GNOME** de verdad.

Para lo que venga: la **compilación con inspector** ya existe.
`gh workflow run compilar.yml -f inspector=true` da un paquete con el Web Inspector abierto, en el
sistema de verdad, para probar hipótesis en vivo sin publicar una versión por cada una. Va marcado
por tres sitios —nombre del artefacto, retención y sufijo en la versión— para que no se confunda con
una compilación normal.

## Bloqueantes

Ninguno técnico. Lo pendiente son comprobaciones que solo puede hacer el humano.

## Preguntas abiertas para el humano

Queda una, y es de código publicado que no se puede ejercitar aquí: en esta máquina no hay Windows ni
GNOME con la aplicación puesta, y lo visual no se afirma desde una compilación en verde.

- **¿Se usa la bóveda?** Es la única pregunta que decide las fases 2 a 4, y no la contesta ninguna
  prueba: la contesta el uso. Con ella hay que mirar tres cosas que aquí no se pueden ver: si el
  CSV de Dashlane se importa entero y bien, si el bloqueo a los quince minutos molesta o tranquiliza,
  y si copiar una contraseña con el borrado a los treinta segundos llega a tiempo de pegarla.
- **¿Cómo queda el oro puesto, en un Mac?** Es la pregunta de la 2.11.0, y sobre todo por **la
  pastilla dorada de la fila activa**: es el cambio más visible de todos, cumple de sobra —8,38:1— y
  es la gramática del icono, pero eso lo dice el cálculo y no el ojo. Si canta, el repliegue está
  pensado: fondo dorado tenue y texto de tinta, como hace Fluent. Mirar también si el ámbar de los
  avisos y el oro se estorban en la pantalla de cifrar, que son vecinos.
- **¿Cómo queda la aplicación en Windows y en GNOME?** La de macOS está juzgada en un Mac; las otras
  dos se escribieron a partir de las convenciones de cada sistema y nadie las ha visto corriendo. Hay
  tres cosas que mirar de una vez: **la estructura** —el panel de Fluent con su barra de acento, la
  cabecera de GNOME—, **el icono de los `.esf`** en el explorador de ficheros, y ahora **el lockup**,
  que ahí sube hasta el borde porque no hay semáforos que esquivar.
